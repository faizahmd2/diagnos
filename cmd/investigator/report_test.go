package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMetricsReportDoesNotDuplicateExistingSection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "final-report.md")
	first, err := writeMetricsOnlyReport(path, "first metrics")
	if err != nil {
		t.Fatal(err)
	}
	second, err := writeMetricsOnlyReport(path, "changed metrics")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("metrics section should not be duplicated or rewritten")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != first {
		t.Fatal("report contents changed")
	}
}

func TestMetricsReportStopsBeforeTelemetryService(t *testing.T) {
	context := "TARGET SNAPSHOT\nPROMETHEUS SERVICES\nservice\n\nTELEMETRY SERVICE\nmetric details\n"
	if got := metricsReportContext(context); got != "TARGET SNAPSHOT\nPROMETHEUS SERVICES\nservice" {
		t.Fatalf("unexpected compact report context: %q", got)
	}
}
