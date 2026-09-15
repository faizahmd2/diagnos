package logs

import (
	"fmt"
	"time"

	"github.com/faizahmd2/diagnos/internal/executor"
	"github.com/faizahmd2/diagnos/internal/planner"
)

func CollectRelatedEvidence(
	plan planner.Plan,
	targetHost string,
	ansibleExecutor *executor.AnsibleExecutor,
	since time.Duration,
	timeout time.Duration,
) ([]Evidence, error) {

	if ansibleExecutor == nil {
		return nil, fmt.Errorf(
			"ansible executor cannot be nil",
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
				ansibleExecutor,
				timeout,
			)

			return collector.CollectEvidence(
				targetHost,
				since,
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
