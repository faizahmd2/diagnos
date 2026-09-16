package logs

import (
	"fmt"
	"time"

	"github.com/faizahmd2/diagnos/internal/config"
	"github.com/faizahmd2/diagnos/internal/executor"
	"github.com/faizahmd2/diagnos/internal/planner"
)

func CollectRelatedEvidence(
	plan planner.Plan,
	targetHost string,
	nativeExecutor *executor.NativeExecutor,
	configuredApplications []config.ApplicationLogConfig,
	since time.Duration,
	timeout time.Duration,
) ([]Evidence, error) {

	if nativeExecutor == nil {
		return nil, fmt.Errorf(
			"native executor cannot be nil",
		)
	}

	if since <= 0 {
		return nil, fmt.Errorf(
			"log collection window must be greater than zero",
		)
	}

	collectors := map[string]func() ([]Evidence, error){
		"system": func() ([]Evidence, error) {
			collector := NewSystemCollector(
				nativeExecutor,
				timeout,
			)

			return collector.CollectEvidence(
				targetHost,
				since,
			)
		},
		"process": func() ([]Evidence, error) {
			collector := NewApplicationCollector(
				nativeExecutor,
				timeout,
				20,
			)

			return collector.CollectEvidence(
				targetHost,
				nil,
				since,
			)
		},
		"kernel": func() ([]Evidence, error) {
			collector := NewKernelCollector(
				nativeExecutor,
				timeout,
			)

			return collector.CollectEvidence(
				targetHost,
				since,
			)
		},
		"application": func() ([]Evidence, error) {
			var sources []LogSource

			for _, application := range configuredApplications {
				for _, path := range application.Paths {
					if path == "" {
						continue
					}

					sources = append(sources, LogSource{
						Name:       application.Name,
						Path:       path,
						Service:    application.Service,
						SourceType: "configured",
					})
				}
			}

			if len(sources) == 0 {
				return nil, nil
			}

			collector := NewApplicationCollector(
				nativeExecutor,
				timeout,
				20,
			)

			return collector.CollectEvidence(
				targetHost,
				sources,
				since,
			)
		},
		"docker": func() ([]Evidence, error) {
			collector := NewDockerCollector(
				nativeExecutor,
				timeout,
			)

			return collector.CollectEvidence(
				targetHost,
			)
		},
	}

	seen := make(map[string]struct{})

	var evidence []Evidence

	for _, investigation := range plan.Investigations {
		for _, source := range investigation.Diagnostic.LogSources {
			if _, exists := seen[source]; exists {
				continue
			}

			seen[source] = struct{}{}

			collect, exists := collectors[source]
			if !exists {
				return nil, fmt.Errorf(
					"unsupported log source %q for capability %q",
					source,
					investigation.CapabilityID,
				)
			}

			sourceEvidence, err := collect()
			if err != nil {
				return nil, fmt.Errorf(
					"collect related logs for source %q: %w",
					source,
					err,
				)
			}

			evidence = append(
				evidence,
				sourceEvidence...,
			)
		}
	}

	return evidence, nil
}
