package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/faizahmd2/diagnos/internal/collector/metrics"
	"github.com/faizahmd2/diagnos/internal/collector/system"
)

type Writer struct {
	Path string
}

func NewWriter(path string) *Writer {
	return &Writer{
		Path: path,
	}
}

func (w *Writer) WriteTargetSnapshot(
	metadata system.MachineMetadata,
	services []metrics.NormalizedService,
) error {
	if err := w.ensureDir(); err != nil {
		return err
	}

	content := fmt.Sprintf(
		`TARGET SNAPSHOT

| Field | Current |
|---|---|
| Host | %s (%s) |
| OS | %s %s |
| Kernel | %s (%s) |
| CPU | %d cores, load 1m=%.2f 5m=%.2f 15m=%.2f |
| RAM | %s total, %s available |
| Disk | %s total, %s available |
| Uptime | %s |

TOP PROCESSES

| PID | Process | CPU | RAM |
|---:|---|---:|---:|
%s
PROMETHEUS SERVICES

%s
`,
		metadata.Hostname,
		metadata.IP,
		metadata.OS,
		metadata.OSVersion,
		metadata.Kernel,
		metadata.Architecture,
		metadata.CPUCores,
		metadata.LoadAverage.OneMinute,
		metadata.LoadAverage.FiveMinutes,
		metadata.LoadAverage.FifteenMinutes,
		formatBytes(metadata.Memory.TotalBytes),
		formatBytes(metadata.Memory.AvailableBytes),
		formatBytes(metadata.Disk.TotalBytes),
		formatBytes(metadata.Disk.AvailableBytes),
		formatUptime(metadata.UptimeSeconds),
		formatProcesses(metadata.TopProcesses),
		formatServices(services),
	)

	return os.WriteFile(
		w.Path,
		[]byte(content),
		0644,
	)
}

func formatServices(
	services []metrics.NormalizedService,
) string {
	if len(services) == 0 {
		return "| No Prometheus services attached |\n"
	}

	var builder strings.Builder

	builder.WriteString("| Service | Address | Health |\n")
	builder.WriteString("|---|---|---|\n")

	for _, service := range services {
		fmt.Fprintf(
			&builder,
			"| %s | %s | %s |\n",
			service.Name,
			service.Address,
			service.Health,
		)
	}

	return builder.String()
}

func formatBytes(bytes uint64) string {
	const (
		GB = 1024 * 1024 * 1024
		MB = 1024 * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	default:
		return fmt.Sprintf("%.2f KB", float64(bytes)/1024)
	}
}

func formatUptime(seconds uint64) string {
	duration := time.Duration(seconds) * time.Second

	days := duration / (24 * time.Hour)
	duration %= 24 * time.Hour

	hours := duration / time.Hour
	duration %= time.Hour

	minutes := duration / time.Minute

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	default:
		return fmt.Sprintf("%dm", minutes)
	}
}

func formatProcesses(processes []system.ProcessInfo) string {
	if len(processes) == 0 {
		return "| - | No processes found | - | - |\n"
	}

	var builder strings.Builder

	for _, process := range processes {
		fmt.Fprintf(
			&builder,
			"| %d | %s | %.1f%% | %.1f%% |\n",
			process.PID,
			process.Command,
			process.CPU,
			process.Memory,
		)
	}

	return builder.String()
}

func (w *Writer) WriteTelemetry(
	services []metrics.NormalizedService,
	maxLines int,
) error {
	content, err := w.buildTelemetryContent(
		services,
		maxLines,
	)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(
		w.Path,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		return fmt.Errorf(
			"open context file: %w",
			err,
		)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf(
			"write telemetry context: %w",
			err,
		)
	}

	return nil
}

func (w *Writer) WritePrometheus(
	prometheusURL string,
) error {
	content := fmt.Sprintf(
		`PROMETHEUS
----------
provider: prometheus
endpoint: %s

`,
		prometheusURL,
	)

	file, err := os.OpenFile(
		w.Path,
		os.O_APPEND|os.O_WRONLY|os.O_CREATE,
		0644,
	)
	if err != nil {
		return fmt.Errorf(
			"open context file: %w",
			err,
		)
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf(
			"write prometheus context: %w",
			err,
		)
	}

	return nil
}

func formatLabels(labels map[string]string) string {
	keys := make([]string, 0, len(labels))

	for key := range labels {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var builder strings.Builder

	builder.WriteString("{")

	for i, key := range keys {
		if i > 0 {
			builder.WriteString(",")
		}

		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(labels[key])
	}

	builder.WriteString("}")

	return builder.String()
}

func (w *Writer) ensureDir() error {
	dir := filepath.Dir(w.Path)

	if dir == "." {
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf(
			"create context directory: %w",
			err,
		)
	}

	return nil
}

func allocateMetricLines(
	services []metrics.NormalizedService,
	budget int,
) []int {
	allocations := make([]int, len(services))

	if budget <= 0 {
		return allocations
	}

	active := make([]int, 0, len(services))

	for i, service := range services {
		if len(service.Metrics) > 0 {
			active = append(active, i)
		}
	}

	for len(active) > 0 && budget > 0 {
		share := budget / len(active)

		if share == 0 {
			share = 1
		}

		nextActive := make([]int, 0, len(active))

		for _, index := range active {
			available := len(services[index].Metrics) - allocations[index]

			take := share
			if take > available {
				take = available
			}

			allocations[index] += take
			budget -= take

			if allocations[index] < len(services[index].Metrics) {
				nextActive = append(nextActive, index)
			}
		}

		active = nextActive
	}

	return allocations
}

func (w *Writer) buildTelemetryContent(
	services []metrics.NormalizedService,
	maxLines int,
) (string, error) {
	if maxLines <= 0 {
		return "", fmt.Errorf(
			"max context file lines must be greater than zero",
		)
	}

	// First build the fixed part of every service.
	serviceHeaders := make([]string, len(services))
	totalFixedLines := 0

	for i, service := range services {
		var builder strings.Builder

		builder.WriteString("TELEMETRY SERVICE\n")
		builder.WriteString("-----------------\n")
		builder.WriteString("name: ")
		builder.WriteString(service.Name)
		builder.WriteString("\n")
		builder.WriteString("address: ")
		builder.WriteString(service.Address)
		builder.WriteString("\n")
		builder.WriteString("health: ")
		builder.WriteString(service.Health)
		builder.WriteString("\n")
		builder.WriteString("scrape_url: ")
		builder.WriteString(service.ScrapeURL)
		builder.WriteString("\n")
		builder.WriteString("collection_start: ")
		builder.WriteString(service.CollectionStart.Format(time.RFC3339))
		builder.WriteString("\n")
		builder.WriteString("collection_end: ")
		builder.WriteString(service.CollectionEnd.Format(time.RFC3339))
		builder.WriteString("\n")
		builder.WriteString("duration: ")
		builder.WriteString(service.Duration.String())
		builder.WriteString("\n")
		builder.WriteString("raw_samples: ")
		builder.WriteString(fmt.Sprintf("%d", service.RawSamples))
		builder.WriteString("\n")
		builder.WriteString("normalized_metrics: ")
		builder.WriteString(fmt.Sprintf("%d", len(service.Metrics)))
		builder.WriteString("\n\n")
		builder.WriteString("METRICS\n")
		builder.WriteString("-------\n")

		header := builder.String()

		serviceHeaders[i] = header
		totalFixedLines += strings.Count(header, "\n")
	}

	// Remaining lines are available for metric data.
	metricBudget := maxLines - totalFixedLines

	if metricBudget < 0 {
		metricBudget = 0
	}

	allocations := allocateMetricLines(
		services,
		metricBudget,
	)

	var builder strings.Builder

	for i, service := range services {
		builder.WriteString(serviceHeaders[i])

		for metricIndex := 0; metricIndex < allocations[i]; metricIndex++ {
			writeMetric(
				&builder,
				service.Metrics[metricIndex],
			)
		}

		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func writeMetric(
	builder *strings.Builder,
	metric metrics.NormalizedMetric,
) {
	builder.WriteString(metric.Name)

	if len(metric.Labels) > 0 {
		builder.WriteString(" ")
		builder.WriteString(formatLabels(metric.Labels))
	}

	if metric.Range != nil {
		fmt.Fprintf(
			builder,
			" start=%g end=%g min=%g max=%g",
			metric.Range.StartValue,
			metric.Range.EndValue,
			metric.Range.Min,
			metric.Range.Max,
		)

		if metric.Range.Constant {
			builder.WriteString(" constant=true")
		} else {
			builder.WriteString(" samples=")
			builder.WriteString(
				fmt.Sprint(metric.Range.Samples),
			)
		}
	}

	builder.WriteString("\n")
}
