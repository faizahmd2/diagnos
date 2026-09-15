package metrics

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
)

const (
	MaxInventorySeries   = 2000
	MaxInventoryFamilies = 500
	MaxSeriesPerFamily   = 50
)

type MetricFamily struct {
	Name   string   `json:"name"`
	Jobs   []string `json:"jobs,omitempty"`
	Labels []string `json:"labels,omitempty"`
	Series int      `json:"series"`
}

type TelemetryInventory struct {
	Target         string         `json:"target"`
	TargetSelector string         `json:"target_selector"`
	Families       []MetricFamily `json:"families"`
	TotalSeries    int            `json:"total_series"`
	Complete       bool           `json:"complete"`
}

type seriesResponse struct {
	Status string `json:"status"`

	Data []map[string]string `json:"data"`

	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
}

func (c *PrometheusCollector) DiscoverInventory(
	target string,
	prometheusTarget PrometheusTarget,
) (TelemetryInventory, error) {
	if target == "" {
		return TelemetryInventory{}, fmt.Errorf(
			"telemetry target is required",
		)
	}

	job := prometheusTarget.Job()
	instance := prometheusTarget.Instance()

	if job == "" {
		return TelemetryInventory{}, fmt.Errorf(
			"prometheus target is missing job label",
		)
	}

	if instance == "" {
		return TelemetryInventory{}, fmt.Errorf(
			"prometheus target is missing instance label",
		)
	}

	targetSelector := fmt.Sprintf(
		`job=%s,instance=%s`,
		strconv.Quote(job),
		strconv.Quote(instance),
	)

	params := url.Values{}
	params.Add("match[]", fmt.Sprintf(
		"{%s}",
		targetSelector,
	))

	body, err := c.request(
		"/api/v1/series",
		params,
	)
	if err != nil {
		return TelemetryInventory{}, fmt.Errorf(
			"discover prometheus series: %w",
			err,
		)
	}

	var response seriesResponse

	if err := json.Unmarshal(body, &response); err != nil {
		return TelemetryInventory{}, fmt.Errorf(
			"decode prometheus series response: %w",
			err,
		)
	}

	if response.Status != "success" {
		return TelemetryInventory{}, fmt.Errorf(
			"prometheus series discovery failed: %s: %s",
			response.ErrorType,
			response.Error,
		)
	}

	return buildInventory(
		target,
		targetSelector,
		response.Data,
	)
}

func buildInventory(
	target string,
	targetSelector string,
	series []map[string]string,
) (TelemetryInventory, error) {
	inventory := TelemetryInventory{
		Target:         target,
		TargetSelector: targetSelector,
		Families:       make([]MetricFamily, 0),
		Complete:       true,
	}

	families := make(map[string]*MetricFamily)

	for _, labels := range series {
		inventory.TotalSeries++

		if inventory.TotalSeries > MaxInventorySeries {
			inventory.Complete = false
			break
		}

		name := labels["__name__"]
		if name == "" {
			return TelemetryInventory{}, fmt.Errorf(
				"prometheus series is missing __name__",
			)
		}

		family, exists := families[name]

		if !exists {
			if len(families) >= MaxInventoryFamilies {
				inventory.Complete = false
				break
			}

			family = &MetricFamily{
				Name:   name,
				Jobs:   make([]string, 0),
				Labels: make([]string, 0),
			}

			families[name] = family
		}

		family.Series++

		if family.Series > MaxSeriesPerFamily {
			inventory.Complete = false
			continue
		}

		if job := labels["job"]; job != "" {
			family.Jobs = appendUnique(
				family.Jobs,
				job,
			)
		}

		for label := range labels {
			if label == "__name__" {
				continue
			}

			family.Labels = appendUnique(
				family.Labels,
				label,
			)
		}
	}

	names := make([]string, 0, len(families))

	for name := range families {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		family := families[name]

		sort.Strings(family.Jobs)
		sort.Strings(family.Labels)

		inventory.Families = append(
			inventory.Families,
			*family,
		)
	}

	return inventory, nil
}

func appendUnique(
	values []string,
	value string,
) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}

	return append(values, value)
}
