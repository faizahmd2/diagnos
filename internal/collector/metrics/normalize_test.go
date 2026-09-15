package metrics

import (
	"testing"
	"time"
)

func TestNormalizeRangeGroupsByLabels(t *testing.T) {
	base := time.Now()

	samples := []Sample{
		{
			Metric: map[string]string{
				"__name__": "node_memory_MemAvailable_bytes",
				"instance": "host-a",
			},
			Value: 100,
			Time:  base,
		},
		{
			Metric: map[string]string{
				"__name__": "node_memory_MemAvailable_bytes",
				"instance": "host-a",
			},
			Value: 80,
			Time:  base.Add(time.Minute),
		},
		{
			Metric: map[string]string{
				"__name__": "node_memory_MemAvailable_bytes",
				"instance": "host-a",
			},
			Value: 60,
			Time:  base.Add(2 * time.Minute),
		},
	}

	result := NormalizeRange(samples, 30)

	if len(result) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(result))
	}

	metric := result[0]

	if metric.Name != "node_memory_MemAvailable_bytes" {
		t.Fatalf("unexpected metric name: %s", metric.Name)
	}

	if metric.Labels["instance"] != "host-a" {
		t.Fatalf("expected instance label")
	}

	if metric.Range.StartValue != 100 {
		t.Fatalf("unexpected start value: %v", metric.Range.StartValue)
	}

	if metric.Range.EndValue != 60 {
		t.Fatalf("unexpected end value: %v", metric.Range.EndValue)
	}

	if metric.Range.Min != 60 {
		t.Fatalf("unexpected min: %v", metric.Range.Min)
	}

	if metric.Range.Max != 100 {
		t.Fatalf("unexpected max: %v", metric.Range.Max)
	}
}

func TestNormalizeRangeDownsamples(t *testing.T) {
	base := time.Now()

	samples := make([]Sample, 100)

	for i := range samples {
		samples[i] = Sample{
			Metric: map[string]string{
				"__name__": "node_load1",
			},
			Value: float64(i),
			Time:  base.Add(time.Duration(i) * time.Minute),
		}
	}

	result := NormalizeRange(samples, 10)

	if len(result) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(result))
	}

	values := result[0].Range.Samples

	if len(values) != 10 {
		t.Fatalf("expected 10 samples, got %d", len(values))
	}

	if values[0] != 0 {
		t.Fatalf("expected first sample to be preserved")
	}

	if values[len(values)-1] != 99 {
		t.Fatalf("expected last sample to be preserved")
	}
}

func TestNormalizeRangeSeparatesLabels(t *testing.T) {
	base := time.Now()

	samples := []Sample{
		{
			Metric: map[string]string{
				"__name__": "node_network_receive_bytes_total",
				"device":   "eth0",
			},
			Value: 100,
			Time:  base,
		},
		{
			Metric: map[string]string{
				"__name__": "node_network_receive_bytes_total",
				"device":   "eth1",
			},
			Value: 200,
			Time:  base,
		},
	}

	result := NormalizeRange(samples, 30)

	if len(result) != 2 {
		t.Fatalf("expected 2 distinct metrics, got %d", len(result))
	}
}
