package metrics

import (
	"sort"
	"time"
)

const defaultMaxSamples = 8

type NormalizedMetric struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
	Source string            `json:"source"`

	Instant *NormalizedSnapshot `json:"instant,omitempty"`
	Range   *NormalizedRange    `json:"range,omitempty"`
}

type NormalizedSnapshot struct {
	Value float64 `json:"value"`
}

type NormalizedRange struct {
	Start time.Time     `json:"start"`
	End   time.Time     `json:"end"`
	Step  time.Duration `json:"step"`

	StartValue float64 `json:"start_value"`
	EndValue   float64 `json:"end_value"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`

	Constant bool `json:"constant"`

	Samples []float64 `json:"samples,omitempty"`
}

type NormalizedService struct {
	Name            string             `json:"name"`
	Address         string             `json:"address"`
	Health          string             `json:"health"`
	ScrapeURL       string             `json:"scrape_url"`
	Source          string             `json:"source"`
	CollectionStart time.Time          `json:"collection_start"`
	CollectionEnd   time.Time          `json:"collection_end"`
	Duration        time.Duration      `json:"duration"`
	RawSamples      int                `json:"raw_samples"`
	Metrics         []NormalizedMetric `json:"metrics"`
}

func NormalizeInstant(sample Sample) NormalizedMetric {
	return NormalizedMetric{
		Name:   sample.Metric["__name__"],
		Labels: removeMetricNameLabel(sample.Metric),
		Source: "prometheus",
		Instant: &NormalizedSnapshot{
			Value: sample.Value,
		},
	}
}

func NormalizeRange(samples []Sample, maxSamples int) []NormalizedMetric {
	if len(samples) == 0 {
		return nil
	}

	if maxSamples <= 0 {
		maxSamples = defaultMaxSamples
	}

	grouped := groupSamples(samples)

	result := make([]NormalizedMetric, 0, len(grouped))

	for _, group := range grouped {
		if len(group) == 0 {
			continue
		}

		sort.Slice(group, func(i, j int) bool {
			return group[i].Time.Before(group[j].Time)
		})

		minValue := group[0].Value
		maxValue := group[0].Value

		for _, sample := range group[1:] {
			if sample.Value < minValue {
				minValue = sample.Value
			}

			if sample.Value > maxValue {
				maxValue = sample.Value
			}
		}

		normalized := NormalizedMetric{
			Name:   group[0].Metric["__name__"],
			Labels: removeMetricNameLabel(group[0].Metric),
			Source: "prometheus",
			Range: &NormalizedRange{
				Start:      group[0].Time,
				End:        group[len(group)-1].Time,
				Step:       inferStep(group),
				StartValue: group[0].Value,
				EndValue:   group[len(group)-1].Value,
				Min:        minValue,
				Max:        maxValue,
				Constant:   minValue == maxValue,
			},
		}

		if !normalized.Range.Constant {
			values := make([]float64, 0, len(group))

			for _, sample := range group {
				values = append(values, sample.Value)
			}

			normalized.Range.Samples = downsample(values, maxSamples)
		}

		result = append(result, normalized)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Name != result[j].Name {
			return result[i].Name < result[j].Name
		}

		return normalizedMetricKey(result[i]) < normalizedMetricKey(result[j])
	})

	return result
}

func groupSamples(samples []Sample) map[string][]Sample {
	grouped := make(map[string][]Sample)

	for _, sample := range samples {
		key := metricKey(sample.Metric)
		grouped[key] = append(grouped[key], sample)
	}

	return grouped
}

func metricKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))

	for key := range labels {
		if key == "__name__" {
			continue
		}

		keys = append(keys, key)
	}

	sort.Strings(keys)

	key := labels["__name__"]

	for _, label := range keys {
		key += "|" + label + "=" + labels[label]
	}

	return key
}

func removeMetricNameLabel(metric map[string]string) map[string]string {
	result := make(map[string]string, len(metric))

	for key, value := range metric {
		if key == "__name__" {
			continue
		}

		result[key] = value
	}

	return result
}

func inferStep(samples []Sample) time.Duration {
	if len(samples) < 2 {
		return 0
	}

	return samples[1].Time.Sub(samples[0].Time)
}

func downsample(values []float64, maxSamples int) []float64 {
	if len(values) <= maxSamples {
		return append([]float64(nil), values...)
	}

	if maxSamples <= 1 {
		return []float64{values[len(values)-1]}
	}

	result := make([]float64, 0, maxSamples)

	result = append(result, values[0])

	interval := float64(len(values)-1) / float64(maxSamples-1)

	for i := 1; i < maxSamples-1; i++ {
		index := int(float64(i) * interval)
		result = append(result, values[index])
	}

	result = append(result, values[len(values)-1])

	return result
}

func normalizedMetricKey(metric NormalizedMetric) string {
	return metricKey(metric.Labels)
}

func NormalizeService(
	target PrometheusTarget,
	samples []Sample,
	start time.Time,
	end time.Time,
	maxSamples int,
) NormalizedService {
	return NormalizedService{
		Name:            target.Job(),
		Address:         target.Address(),
		Health:          target.Health,
		ScrapeURL:       target.ScrapeURL,
		Source:          "prometheus",
		CollectionStart: start,
		CollectionEnd:   end,
		Duration:        end.Sub(start),
		RawSamples:      len(samples),
		Metrics:         NormalizeRange(samples, maxSamples),
	}
}
