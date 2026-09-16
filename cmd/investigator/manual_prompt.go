package main

import (
	"strings"
)

// buildManualPrompt is deliberately provider-neutral. It is written in
// app-metrics mode so an operator can paste it into any AI chat without
// starting an API call from Diagnos.
func buildManualPrompt(evidence, hint string) string {
	var builder strings.Builder
	builder.WriteString("You are assisting an infrastructure engineer. Review the collected Linux host and Prometheus evidence below. Give a concise, developer-friendly investigation note: current state, the most important abnormal or healthy signals, likely explanations clearly labelled as hypotheses, and the next two or three safe read-only checks. Do not invent measurements or claim a root cause that the evidence cannot prove.\n\n")
	if hint = strings.TrimSpace(hint); hint != "" {
		builder.WriteString("Investigation hint (context only, not proof): ")
		builder.WriteString(hint)
		builder.WriteString("\n\n")
	}
	builder.WriteString("Collected evidence:\n\n")
	builder.WriteString(strings.TrimSpace(evidence))
	builder.WriteString("\n")
	return builder.String()
}
