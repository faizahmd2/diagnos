package executor

import (
	"errors"
	"time"

	"github.com/faizahmd2/diagnos/internal/catalog"
	"github.com/faizahmd2/diagnos/internal/planner"
)

type DiagnosticExecutor struct {
	Ansible *AnsibleExecutor
}

func NewDiagnosticExecutor(
	ansible *AnsibleExecutor,
) *DiagnosticExecutor {
	return &DiagnosticExecutor{
		Ansible: ansible,
	}
}

func (e *DiagnosticExecutor) Execute(
	plan planner.Plan,
) []catalog.DiagnosticResult {

	results := make(
		[]catalog.DiagnosticResult,
		0,
	)

	for _, investigation := range plan.Investigations {
		for _, command := range investigation.Diagnostic.Commands {
			result := e.executeCommand(
				investigation,
				command,
			)

			results = append(results, result)
		}
	}

	return results
}

func (e *DiagnosticExecutor) executeCommand(
	investigation planner.PlannedDiagnostic,
	command string,
) catalog.DiagnosticResult {

	startedAt := time.Now()

	output, err := e.Ansible.Run(
		command,
		time.Duration(
			investigation.Diagnostic.TimeoutSeconds,
		)*time.Second,
	)

	endedAt := time.Now()

	result := catalog.DiagnosticResult{
		DiagnosticID: investigation.CapabilityID,
		Area:         investigation.Area,
		Command:      command,
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		Duration:     endedAt.Sub(startedAt),
		Output:       output,
	}

	if err == nil {
		result.Status = catalog.StatusSuccess
		return result
	}

	var timeoutErr *CommandTimeoutError

	if errors.As(err, &timeoutErr) {
		result.Status = catalog.StatusTimeout
		result.Error = err.Error()
		return result
	}

	result.Status = catalog.StatusFailed
	result.Error = err.Error()

	return result
}
