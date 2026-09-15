package executor

import (
	"time"

	"github.com/faizahmd2/diagnos/internal/catalog"
	"github.com/faizahmd2/diagnos/internal/planner"
)

type Executor struct {
	Ansible AnsibleRunner
}

type AnsibleRunner interface {
	Run(
		cmd string,
		timeout time.Duration,
	) (string, error)
}

func New(ansible AnsibleRunner) *Executor {
	return &Executor{
		Ansible: ansible,
	}
}

func (e *Executor) Execute(
	plan planner.Plan,
) []catalog.DiagnosticResult {
	results := make([]catalog.DiagnosticResult, 0)

	for _, investigation := range plan.Investigations {
		results = append(
			results,
			e.executeDiagnostic(investigation),
		)
	}

	return results
}

func (e *Executor) executeDiagnostic(
	investigation planner.PlannedDiagnostic,
) catalog.DiagnosticResult {

	diagnostic := investigation.Diagnostic

	for _, command := range diagnostic.Commands {
		startedAt := time.Now()

		output, err := e.Ansible.Run(
			command,
			time.Duration(diagnostic.TimeoutSeconds)*time.Second,
		)

		endedAt := time.Now()

		if err == nil {
			return catalog.DiagnosticResult{
				DiagnosticID: diagnostic.ID,
				Area:         diagnostic.Area,
				Command:      command,
				Status:       catalog.StatusSuccess,
				Output:       output,
				StartedAt:    startedAt,
				EndedAt:      endedAt,
				Duration:     endedAt.Sub(startedAt),
			}
		}
	}

	return catalog.DiagnosticResult{
		DiagnosticID: diagnostic.ID,
		Area:         diagnostic.Area,
		Command:      "",
		Status:       catalog.StatusFailed,
	}
}
