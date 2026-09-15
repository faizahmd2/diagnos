package catalog

import "time"

type ResultStatus string

const (
	StatusSuccess ResultStatus = "success"
	StatusFailed  ResultStatus = "failed"
	StatusTimeout ResultStatus = "timeout"
)

type DiagnosticResult struct {
	DiagnosticID string
	Area         Area

	Command string
	Status  ResultStatus

	Output string
	Error  string

	StartedAt time.Time
	EndedAt   time.Time
	Duration  time.Duration
}
