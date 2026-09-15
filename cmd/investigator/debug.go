package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

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
)

func newDebugCmd() *cobra.Command {
	var hint string

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

			targetConfig, err := cfg.ResolveTarget(target)
			if err != nil {
				return fmt.Errorf("resolve target: %w", err)
			}

			// --------------------------------------------------
			// Step 2: Validate ansible
			// --------------------------------------------------

			playbookPath, cleanupPlaybook, err := resources.WritePlaybook()
			if err != nil {
				return fmt.Errorf("failed to prepare ansible playbook: %w", err)
			}
			defer cleanupPlaybook()

			ansibleExecutor := executor.NewAnsibleExecutor(
				playbookPath,
				executor.Target{
					Host: targetConfig.Host,
					User: targetConfig.User,
				},
			)

			if err := ansibleExecutor.CheckTarget(); err != nil {
				return err
			}

			logger.Info(
				"target validated",
				"target", target,
			)

			// --------------------------------------------------
			// Step 3: Validate Prometheus connection
			// --------------------------------------------------

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
			// Step 4 will be machine metadata.
			// --------------------------------------------------
			metadataCollector := system.NewMetadataCollector(
				ansibleExecutor,
				30*time.Second,
			)

			metadata, err := metadataCollector.Collect()
			if err != nil {
				return fmt.Errorf(
					"collect machine metadata: %w",
					err,
				)
			}

			contextWriter := collector.NewWriter("debug/context.txt")

			logger.Info(
				"machine metadata collected",
				"hostname", metadata.Hostname,
				"os", metadata.OS,
				"os_version", metadata.OSVersion,
				"kernel", metadata.Kernel,
				"architecture", metadata.Architecture,
				"cpu_cores", metadata.CPUCores,
				"memory_bytes", metadata.Memory.TotalBytes,
				"disk_bytes", metadata.Disk.TotalBytes,
				"uptime_seconds", metadata.UptimeSeconds,
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

			normalizedServices := make([]metrics.NormalizedService, 0, len(prometheusTargets))

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

			finalContextData, err := os.ReadFile("debug/context.txt")
			if err != nil {
				return fmt.Errorf(
					"read final analysis context: %w",
					err,
				)
			}

			investigationCatalog, err := catalog.LoadEmbedded(
				resources.Files,
				"catalog",
			)
			if err != nil {
				return fmt.Errorf(
					"load investigation catalog: %w",
					err,
				)
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
				"debug/prompt1.txt",
				[]byte(prompt),
				0644,
			)
			fmt.Println("\n Prompt written ======================")

			model, err := aiclient.ResolveModel(
				cfg.AI.Model,
				cfg.AI.Models,
			)
			if err != nil {
				return fmt.Errorf(
					"failed to resolve AI model: %w",
					err,
				)
			}

			aiClient := aiclient.NewClient(
				model,
				http.DefaultClient,
			)

			responseData, err := aiClient.Call(
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
				ansibleExecutor,
			)

			results := diagnosticExecutor.Execute(plan)

			relatedEvidence, err := logs.CollectRelatedEvidence(
				plan,
				targetConfig.Host,
				ansibleExecutor,
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
				"debug/prompt2.txt",
				[]byte(finalPrompt),
				0644,
			); err != nil {
				return fmt.Errorf(
					"write final analysis prompt: %w",
					err,
				)
			}

			fmt.Println(
				"\nFINAL AI REQUEST WRITTEN TO debug/prompt2.txt",
			)

			finalResponseData, err := aiClient.Call(
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
					"## System Telemetry\n\n"+
					"%s\n\n"+
					"%s\n",
				string(finalContextData),
				strings.TrimSpace(string(finalResponseData)),
			)

			if err := os.WriteFile(
				"debug/final-report.md",
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

	return cmd
}
