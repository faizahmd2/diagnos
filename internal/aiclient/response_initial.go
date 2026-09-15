package aiclient

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/faizahmd2/diagnos/internal/catalog"
)

func ValidateInitialResponse(
	response InitialAnalysisResponse,
	catalogData *catalog.Catalog,
) error {

	if catalogData == nil {
		return fmt.Errorf(
			"catalog cannot be nil",
		)
	}

	if len(response.Investigations) > 3 {
		return fmt.Errorf(
			"initial AI response contains %d investigations, maximum is 3",
			len(response.Investigations),
		)
	}

	seen := make(map[string]struct{})

	for _, investigation := range response.Investigations {
		if strings.TrimSpace(investigation.CapabilityID) == "" {
			return fmt.Errorf(
				"investigation contains empty capability ID",
			)
		}

		if _, exists := seen[investigation.CapabilityID]; exists {
			return fmt.Errorf(
				"duplicate capability ID %q",
				investigation.CapabilityID,
			)
		}

		seen[investigation.CapabilityID] = struct{}{}

		if investigation.Confidence < 0 ||
			investigation.Confidence > 1 {
			return fmt.Errorf(
				"invalid confidence %.4f for capability %q",
				investigation.Confidence,
				investigation.CapabilityID,
			)
		}

		if len(investigation.MetricObservations) > 10 {
			return fmt.Errorf(
				"too many metric observations for capability %q, maximum is 3",
				investigation.CapabilityID,
			)
		}

		if len(investigation.ConnectedTelemetryObservations) > 10 {
			return fmt.Errorf(
				"too many connected telemetry observations for capability %q, maximum is 3",
				investigation.CapabilityID,
			)
		}

		if len(investigation.Evidence) == 0 {
			return fmt.Errorf(
				"no evidence provided for capability %q",
				investigation.CapabilityID,
			)
		}

		if _, exists := catalogData.FindCapability(
			investigation.CapabilityID,
		); !exists {
			return fmt.Errorf(
				"AI selected unknown capability %q",
				investigation.CapabilityID,
			)
		}
	}

	return nil
}

func ParseInitialResponse(
	data []byte,
) (InitialAnalysisResponse, error) {

	var response InitialAnalysisResponse

	if err := json.Unmarshal(data, &response); err != nil {
		return InitialAnalysisResponse{}, fmt.Errorf(
			"failed to parse initial analysis response: %w",
			err,
		)
	}

	return response, nil
}
