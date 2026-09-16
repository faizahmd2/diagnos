package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	providerai "github.com/faizahmd2/diagnos/internal/ai"
	"github.com/faizahmd2/diagnos/internal/aiclient"
	"github.com/faizahmd2/diagnos/internal/catalog"
	collector "github.com/faizahmd2/diagnos/internal/collector/initial-context"
	"github.com/faizahmd2/diagnos/internal/collector/logs"
	"github.com/faizahmd2/diagnos/internal/collector/metrics"
	"github.com/faizahmd2/diagnos/internal/collector/system"
	"github.com/faizahmd2/diagnos/internal/config"
	"github.com/faizahmd2/diagnos/internal/executor"
	"github.com/faizahmd2/diagnos/internal/planner"
	"github.com/faizahmd2/diagnos/internal/resources"
	"github.com/faizahmd2/diagnos/internal/transport"
)

func newDebugCmd() *cobra.Command {
	var hint string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "debug <machine/IP>",
		Short: "Run an investigation against a target machine",
		Args:  cobra.ExactArgs(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			reportDir, err := config.ResolveOutputDirectory(cfg.Output.Directory)
			if err != nil {
				return fmt.Errorf("resolve report directory: %w", err)
			}

			if err := os.MkdirAll(reportDir, 0755); err != nil {
				return fmt.Errorf("create report directory: %w", err)
			}

			targetConfig, err := cfg.ResolveTarget(target)
			if err != nil {
				return fmt.Errorf("resolve target: %w", err)
			}

			investigationCatalog, err := catalog.LoadEmbedded(resources.Files, "catalog")
			if err != nil {
				return fmt.Errorf("load investigation catalog: %w", err)
			}
			if dryRun {
				for _, area := range []catalog.Area{catalog.AreaDisk, catalog.AreaCPU, catalog.AreaMemory, catalog.AreaProcess, catalog.AreaNetwork, catalog.AreaTCP, catalog.AreaSystem} {
					for _, diagnostic := range investigationCatalog.ForArea(area) {
						for _, command := range diagnostic.Commands {
							fmt.Printf("%s.%s: %s\n", area, diagnostic.ID, command)
						}
					}
				}
				return nil
			}

			sshTransport, err := transport.Dial(context.Background(), transport.Config{Host: targetConfig.Host, User: targetConfig.User, Port: targetConfig.Port, KeyPath: cfg.SSH.KeyPath, KnownHosts: cfg.SSH.KnownHosts, HostKeyPolicy: transport.HostKeyPolicy(cfg.SSH.HostKeyPolicy), JumpHosts: cfg.SSH.JumpHosts, ConnectTimeout: cfg.SSH.ConnectTimeout, CommandTimeout: cfg.SSH.CommandTimeout, MaxParallel: cfg.SSH.MaxParallel, MaxOutputBytes: cfg.SSH.MaxOutputBytes, Logger: logger})
			if err != nil {
				return err
			}
			defer sshTransport.Close()
			nativeExecutor := executor.NewNative(sshTransport)

			if err := nativeExecutor.CheckTarget(); err != nil {
				return fmt.Errorf("target %q is not reachable through SSH: %w", target, err)
			}
			if facts, err := sshTransport.Facts(context.Background()); err != nil {
				logger.Warn("could not probe target OS facts; using common catalog", "host", targetConfig.Host, "error", err)
			} else {
				logger.Info("target OS facts collected", "host", targetConfig.Host, "os_id", facts.OSID, "family", facts.Family, "kernel", facts.Kernel)
				if facts.Family == transport.FamilyUnknown {
					logger.Warn("unrecognised target OS; using common catalog", "host", targetConfig.Host, "os_id", facts.OSID)
				}
			}

			logger.Info(
				"target validated",
				"target", target,
			)
			metadataCollector := system.NewMetadataCollector(nativeExecutor, 30*time.Second)
			metadata, err := metadataCollector.Collect()
			if err != nil {
				return fmt.Errorf("collect machine metadata: %w", err)
			}
			contextWriter := collector.NewWriter(filepath.Join(reportDir, "context.txt"))
			logger.Info("machine metadata collected", "hostname", metadata.Hostname, "os", metadata.OS, "os_version", metadata.OSVersion, "kernel", metadata.Kernel, "architecture", metadata.Architecture, "cpu_cores", metadata.CPUCores, "memory_bytes", metadata.Memory.TotalBytes, "disk_bytes", metadata.Disk.TotalBytes, "uptime_seconds", metadata.UptimeSeconds)

			// --------------------------------------------------
			// Step 3: Validate Prometheus connection
			// --------------------------------------------------

			var normalizedServices []metrics.NormalizedService
			if strings.TrimSpace(cfg.Telemetry.Prometheus.URL) != "" {
				prometheusCollector := metrics.NewPrometheusCollector(
					cfg.Telemetry.Prometheus.URL,
					metrics.PrometheusAuth{
						Type:     cfg.Telemetry.Prometheus.Auth.Type,
						Username: cfg.Telemetry.Prometheus.Auth.Username,
					},
				)

				if err := prometheusCollector.CheckConnection(); err != nil {
					return err
				}

				logger.Info(
					"prometheus connection verified",
					"base_url", cfg.Telemetry.Prometheus.URL,
				)

				// --------------------------------------------------
				// Step 5: Collect telemetry inventory
				// --------------------------------------------------

				hostIP, err := metrics.ResolveHostIP(
					targetConfig.Host,
				)
				if err != nil {
					return fmt.Errorf(
						"resolve target IP: %w",
						err,
					)
				}

				prometheusTargets, err :=
					prometheusCollector.DiscoverTargets(hostIP)

				if err != nil {
					return fmt.Errorf(
						"discover telemetry targets: %w",
						err,
					)
				}

				logger.Info(
					"prometheus targets discovered",
					"target", target,
					"host_ip", hostIP,
					"targets", len(prometheusTargets),
				)

				normalizedServices = make([]metrics.NormalizedService, 0, len(prometheusTargets))

				for _, telemetryTarget := range prometheusTargets {
					logger.Info(
						"prometheus target",
						"job", telemetryTarget.Job(),
						"instance", telemetryTarget.Instance(),
						"address", telemetryTarget.Address(),
						"health", telemetryTarget.Health,
						"scrape_url", telemetryTarget.ScrapeURL,
					)

					end := time.Now()
					start := end.Add(-30 * time.Minute)

					query := fmt.Sprintf(
						`{job=%s,instance=%s}`,
						strconv.Quote(telemetryTarget.Job()),
						strconv.Quote(telemetryTarget.Instance()),
					)

					samples, err := prometheusCollector.QueryRange(
						query,
						metrics.RangeQueryOptions{
							Start: start,
							End:   end,
							Step:  time.Minute,
						},
					)
					if err != nil {
						return fmt.Errorf(
							"collect telemetry for job %q: %w",
							telemetryTarget.Job(),
							err,
						)
					}

					normalizedService := metrics.NormalizeService(
						telemetryTarget,
						samples,
						start,
						end,
						8,
					)

					normalizedServices = append(
						normalizedServices,
						normalizedService,
					)

					logger.Info(
						"telemetry normalized",
						"job", telemetryTarget.Job(),
						"raw_samples", len(samples),
						"normalized_metrics", len(normalizedService.Metrics),
					)
				}
			} else {
				logger.Info("Prometheus is not configured; continuing without telemetry")
			}

			if err := contextWriter.WriteTargetSnapshot(
				metadata,
				normalizedServices,
			); err != nil {
				return fmt.Errorf(
					"write target snapshot: %w",
					err,
				)
			}

			if err := contextWriter.WriteTelemetry(
				normalizedServices,
				cfg.AI.RequestLimits.MaxLinesContextFile,
			); err != nil {
				return fmt.Errorf(
					"write telemetry context: %w",
					err,
				)
			}

			finalContextData, err := os.ReadFile(filepath.Join(reportDir, "context.txt"))
			if err != nil {
				return fmt.Errorf(
					"read final analysis context: %w",
					err,
				)
			}
			if cfg.Output.ReportType == "app-metrics" {
				manualPrompt := buildManualPrompt(string(finalContextData), hint)
				if err := os.WriteFile(filepath.Join(reportDir, "prompt1.txt"), []byte(manualPrompt), 0644); err != nil {
					return fmt.Errorf("write manual AI prompt: %w", err)
				}
				report, err := writeMetricsOnlyReport(
					filepath.Join(reportDir, "final-report.md"),
					string(finalContextData),
				)
				if err != nil {
					return fmt.Errorf("write metrics report: %w", err)
				}
				fmt.Printf("\nMETRICS REPORT\n%s\n\nManual AI prompt written to debug/prompt1.txt\n", report)
				return nil
			}

			capabilities := aiclient.BuildCapabilities(
				investigationCatalog,
			)

			initialRequest := aiclient.InitialAnalysisRequest{
				Context:      string(finalContextData),
				Hint:         hint,
				Capabilities: capabilities,
			}

			prompt, err := aiclient.BuildInitialRequest(
				initialRequest,
			)
			if err != nil {
				return fmt.Errorf(
					"build initial analysis request: %w",
					err,
				)
			}

			fmt.Println("\n================ INITIAL AI REQUEST ================")
			// fmt.Println(prompt)
			os.WriteFile(
				filepath.Join(reportDir, "prompt1.txt"),
				[]byte(prompt),
				0644,
			)
			fmt.Println("\n Prompt written ======================")

			var callAI func(context.Context, string) ([]byte, error)
			if cfg.AI.Provider != "" {
				provider, err := providerai.New(providerai.Config{Name: cfg.AI.Provider, Model: cfg.AI.Model, BaseURL: cfg.AI.BaseURL, APIKey: cfg.AI.APIKey, Timeout: cfg.AI.Timeout})
				if err != nil {
					return err
				}
				callAI = func(ctx context.Context, prompt string) ([]byte, error) {
					response, err := provider.Complete(ctx, providerai.Request{Prompt: prompt, MaxTokens: 4096, Temperature: 0.2})
					return response.JSON, err
				}
			} else {
				model, err := aiclient.ResolveModel(cfg.AI.Model, cfg.AI.Models)
				if err != nil {
					return fmt.Errorf("failed to resolve AI model: %w", err)
				}
				aiClient := aiclient.NewClient(model, http.DefaultClient)
				callAI = aiClient.Call
			}

			responseData, err := callAI(
				context.Background(),
				prompt,
			)
			if err != nil {
				return fmt.Errorf(
					"initial AI analysis failed: %w",
					err,
				)
			}

			analysisResponse, err := aiclient.ParseInitialResponse(
				responseData,
			)
			if err != nil {
				return fmt.Errorf(
					"failed to parse initial AI response: %w",
					err,
				)
			}

			if err := aiclient.ValidateInitialResponse(
				analysisResponse,
				investigationCatalog,
			); err != nil {
				return fmt.Errorf(
					"invalid initial AI response: %w",
					err,
				)
			}

			fmt.Printf(
				"\nINITIAL AI RESPONSE\n%s\n",
				string(responseData),
			)

			fmt.Printf(
				"\nPARSED INITIAL RESPONSE\n%+v\n",
				analysisResponse,
			)

			plan, err := planner.Build(
				analysisResponse,
				investigationCatalog,
			)
			if err != nil {
				return fmt.Errorf(
					"failed to build investigation plan: %w",
					err,
				)
			}

			for _, investigation := range plan.Investigations {
				fmt.Printf(
					"\nCAPABILITY: %s\n",
					investigation.CapabilityID,
				)

				fmt.Printf(
					"LOG SOURCES: %v\n",
					investigation.Diagnostic.LogSources,
				)

				fmt.Printf(
					"COMMANDS: %v\n",
					investigation.Diagnostic.Commands,
				)
			}

			diagnosticExecutor := executor.NewDiagnosticExecutor(
				sshTransport,
			)

			results := diagnosticExecutor.Execute(plan)

			relatedEvidence, err := logs.CollectRelatedEvidence(
				plan,
				targetConfig.Host,
				nativeExecutor,
				cfg.Logs.Applications,
				30*time.Minute,
				30*time.Second,
			)
			if err != nil {
				return fmt.Errorf(
					"collect related log evidence: %w",
					err,
				)
			}

			fmt.Printf(
				"\nRELATED LOG EVIDENCE\n%s\n",
				logs.Render(relatedEvidence),
			)

			finalRequest := aiclient.FinalAnalysisRequest{
				Context:        string(finalContextData),
				Investigations: analysisResponse.Investigations,
				Unsupported:    analysisResponse.UnsupportedObservations,
				DiagnosticData: results,
				LogEvidence:    logs.Render(relatedEvidence),
			}

			finalPrompt, err := aiclient.BuildFinalRequest(
				finalRequest,
			)
			if err != nil {
				return fmt.Errorf(
					"build final analysis request: %w",
					err,
				)
			}

			if err := os.WriteFile(
				filepath.Join(reportDir, "prompt2.txt"),
				[]byte(finalPrompt),
				0644,
			); err != nil {
				return fmt.Errorf(
					"write final analysis prompt: %w",
					err,
				)
			}

			fmt.Printf(
				"\nFINAL AI REQUEST WRITTEN TO %s\n",
				filepath.Join(reportDir, "prompt2.txt"),
			)

			finalResponseData, err := callAI(
				context.Background(),
				finalPrompt,
			)
			if err != nil {
				return fmt.Errorf(
					"final AI analysis failed: %w",
					err,
				)
			}

			finalReport := fmt.Sprintf(
				"# Diagnos Investigation Report\n\n"+
					"## Collected system metrics\n\n"+
					"%s\n\n"+
					"## AI investigation\n\n"+
					"%s\n",
				metricsReportContext(string(finalContextData)),
				strings.TrimSpace(string(finalResponseData)),
			)

			if err := os.WriteFile(
				filepath.Join(reportDir, "final-report.md"),
				[]byte(finalReport),
				0644,
			); err != nil {
				return fmt.Errorf(
					"write final investigation report: %w",
					err,
				)
			}

			fmt.Printf(
				"\nFINAL INVESTIGATION REPORT\n%s\n",
				finalReport,
			)

			return nil
		},
	}

	cmd.Flags().StringVar(
		&hint,
		"hint",
		"",
		"known-issue hint, e.g. 'disk:95%'",
	)
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print catalog commands without connecting")

	return cmd
}
