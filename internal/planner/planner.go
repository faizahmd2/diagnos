package planner

import (
	"fmt"

	"github.com/faizahmd2/diagnos/internal/aiclient"
	"github.com/faizahmd2/diagnos/internal/catalog"
)

type PlannedDiagnostic struct {
	CapabilityID string
	Area         catalog.Area

	Confidence float64
	Evidence   []string

	Diagnostic catalog.Diagnostic
}

type Plan struct {
	Investigations []PlannedDiagnostic
}

func Build(
	response aiclient.InitialAnalysisResponse,
	catalogData *catalog.Catalog,
) (Plan, error) {

	if catalogData == nil {
		return Plan{}, fmt.Errorf(
			"catalog cannot be nil",
		)
	}

	plan := Plan{
		Investigations: make(
			[]PlannedDiagnostic,
			0,
			len(response.Investigations),
		),
	}

	for _, investigation := range response.Investigations {
		diagnostic, ok := catalogData.FindCapability(
			investigation.CapabilityID,
		)

		if !ok {
			return Plan{}, fmt.Errorf(
				"AI selected unknown capability %q",
				investigation.CapabilityID,
			)
		}

		plan.Investigations = append(
			plan.Investigations,
			PlannedDiagnostic{
				CapabilityID: investigation.CapabilityID,
				Area:         diagnostic.Area,
				Confidence:   investigation.Confidence,
				Evidence:     investigation.Evidence,
				Diagnostic:   diagnostic,
			},
		)
	}

	return plan, nil
}
