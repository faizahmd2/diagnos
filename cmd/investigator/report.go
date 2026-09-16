package main

import (
	"fmt"
	"os"
	"strings"
)

const metricsSection = "## Collected system metrics"

const telemetryServiceSection = "\nTELEMETRY SERVICE\n"

func metricsReportContext(context string) string {
	if before, _, found := strings.Cut(context, telemetryServiceSection); found {
		return strings.TrimSpace(before)
	}
	return strings.TrimSpace(context)
}

// writeMetricsOnlyReport keeps a previously written metrics section intact.
// This prevents report-only reruns from needlessly growing or rewriting a
// final report while still putting Go-collected evidence at the top.
func writeMetricsOnlyReport(path string, metrics string) (string, error) {
	if existing, err := os.ReadFile(path); err == nil && strings.Contains(string(existing), metricsSection) {
		return string(existing), nil
	}
	report := fmt.Sprintf("# Diagnos Investigation Report\n\n%s\n\n%s\n", metricsSection, metricsReportContext(metrics))
	if err := os.WriteFile(path, []byte(report), 0644); err != nil {
		return "", err
	}
	return report, nil
}
