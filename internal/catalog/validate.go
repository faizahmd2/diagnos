package catalog

import (
	"fmt"
)

var validAreas = map[Area]bool{
	AreaDisk:    true,
	AreaCPU:     true,
	AreaMemory:  true,
	AreaProcess: true,
	AreaNetwork: true,
	AreaTCP:     true,
	AreaSystem:  true,
}

var validCosts = map[Cost]bool{
	CostLow:    true,
	CostMedium: true,
	CostHigh:   true,
}

var validOperators = map[string]bool{
	">":  true,
	">=": true,
	"<":  true,
	"<=": true,
}

func Validate(catalog *Catalog) error {
	seenIDs := make(map[string]bool)

	for area, diagnostics := range catalog.Diagnostics {
		if !validAreas[area] {
			return fmt.Errorf(
				"invalid area %q",
				area,
			)
		}

		for _, diagnostic := range diagnostics {
			if diagnostic.ID == "" {
				return fmt.Errorf(
					"diagnostic in area %q has no ID",
					area,
				)
			}

			if seenIDs[diagnostic.ID] {
				return fmt.Errorf(
					"duplicate diagnostic ID %q",
					diagnostic.ID,
				)
			}

			seenIDs[diagnostic.ID] = true

			if !validCosts[diagnostic.Cost] {
				return fmt.Errorf(
					"diagnostic %q has invalid cost %q",
					diagnostic.ID,
					diagnostic.Cost,
				)
			}

			if len(diagnostic.Commands) == 0 {
				return fmt.Errorf(
					"diagnostic %q has no commands",
					diagnostic.ID,
				)
			}

			if diagnostic.TimeoutSeconds <= 0 {
				return fmt.Errorf(
					"diagnostic %q must have a positive timeout",
					diagnostic.ID,
				)
			}

			if diagnostic.TimeoutSeconds > 60 {
				return fmt.Errorf(
					"diagnostic %q timeout cannot exceed 60 seconds",
					diagnostic.ID,
				)
			}

			if diagnostic.Trigger != nil {
				if diagnostic.Trigger.Signal == "" {
					return fmt.Errorf(
						"diagnostic %q trigger has no signal",
						diagnostic.ID,
					)
				}

				if !validOperators[diagnostic.Trigger.Operator] {
					return fmt.Errorf(
						"diagnostic %q has invalid trigger operator %q",
						diagnostic.ID,
						diagnostic.Trigger.Operator,
					)
				}
			}
		}
	}

	return nil
}
