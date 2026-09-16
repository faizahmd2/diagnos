package executor

import (
	"context"
	"time"

	"github.com/faizahmd2/diagnos/internal/catalog"
	"github.com/faizahmd2/diagnos/internal/planner"
	"github.com/faizahmd2/diagnos/internal/transport"
)

type DiagnosticExecutor struct {
	Transport transport.Executor
}

func NewDiagnosticExecutor(
	exec transport.Executor,
) *DiagnosticExecutor {
	return &DiagnosticExecutor{
		Transport: exec,
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
			// Commands are ordered primary -> fallback. A non-zero exit is
			// deliberately not a transport error, but it does select the next
			// catalog fallback. Once one succeeds no redundant probe is run.
			if result.Status == catalog.StatusSuccess {
				break
			}
		}
	}

	return results
}

func (e *DiagnosticExecutor) executeCommand(
	investigation planner.PlannedDiagnostic,
	command string,
) catalog.DiagnosticResult {

	startedAt := time.Now()

	raw := e.Transport.Run(context.Background(), transport.Command{Key: investigation.CapabilityID, Argv: command, Timeout: time.Duration(investigation.Diagnostic.TimeoutSeconds) * time.Second})
	endedAt := time.Now()

	result := catalog.DiagnosticResult{
		DiagnosticID: investigation.CapabilityID,
		Area:         investigation.Area,
		Command:      command,
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		Duration:     endedAt.Sub(startedAt),
		Output:       raw.Stdout,
	}

	if raw.Err == nil && raw.ExitCode == 0 && !raw.TimedOut {
		result.Status = catalog.StatusSuccess
		return result
	}

	if raw.TimedOut {
		result.Status = catalog.StatusTimeout
		result.Error = "command timed out"
		return result
	}

	result.Status = catalog.StatusFailed
	if raw.Err != nil {
		result.Error = raw.Err.Error()
	} else {
		result.Error = raw.Stderr
		if result.Error == "" {
			result.Error = "remote command exited non-zero"
		}
	}

	return result
}
