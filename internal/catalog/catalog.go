package catalog

import "fmt"

type Area string

const (
	AreaDisk    Area = "disk"
	AreaCPU     Area = "cpu"
	AreaMemory  Area = "memory"
	AreaProcess Area = "process"
	AreaNetwork Area = "network"
	AreaTCP     Area = "tcp"
	AreaSystem  Area = "system"
)

type Cost string

const (
	CostLow    Cost = "low"
	CostMedium Cost = "medium"
	CostHigh   Cost = "high"
)

type LogSource string

const (
	LogSourceApplication LogSource = "application"
	LogSourceProcess     LogSource = "process"
	LogSourceDocker      LogSource = "docker"
	LogSourceSystem      LogSource = "system"
	LogSourceKernel      LogSource = "kernel"
)

type Diagnostic struct {
	ID          string
	Area        Area
	Description string

	Cost Cost

	// Commands are ordered primary -> fallback.
	Commands []string

	// Maximum execution time for one attempt.
	TimeoutSeconds int

	// Optional condition that makes this diagnostic eligible.
	Trigger *Trigger

	LogSources []string
}

type Trigger struct {
	Signal   string
	Operator string
	Value    float64
}

type Catalog struct {
	Diagnostics map[Area][]Diagnostic
}

func NewCatalog() *Catalog {
	return &Catalog{
		Diagnostics: make(map[Area][]Diagnostic),
	}
}

func (c *Catalog) Add(diagnostic Diagnostic) {
	c.Diagnostics[diagnostic.Area] =
		append(c.Diagnostics[diagnostic.Area], diagnostic)
}

func (c *Catalog) ForArea(area Area) []Diagnostic {
	return c.Diagnostics[area]
}

func (c *Catalog) FindCapability(
	id string,
) (Diagnostic, bool) {

	for area, diagnostics := range c.Diagnostics {
		for _, diagnostic := range diagnostics {
			capabilityID := fmt.Sprintf(
				"%s.%s",
				area,
				diagnostic.ID,
			)

			if capabilityID == id {
				return diagnostic, true
			}
		}
	}

	return Diagnostic{}, false
}
