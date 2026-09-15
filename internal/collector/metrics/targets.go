package metrics

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type PrometheusTarget struct {
	Labels           map[string]string `json:"labels"`
	DiscoveredLabels map[string]string `json:"discoveredLabels"`
	ScrapePool       string            `json:"scrapePool"`
	ScrapeURL        string            `json:"scrapeUrl"`
	GlobalURL        string            `json:"globalUrl"`
	Health           string            `json:"health"`
	LastError        string            `json:"lastError"`
}

func (t PrometheusTarget) Job() string {
	return t.Labels["job"]
}

func (t PrometheusTarget) Instance() string {
	return t.Labels["instance"]
}

func (t PrometheusTarget) Address() string {
	return t.DiscoveredLabels["__address__"]
}

type targetsResponse struct {
	Status string `json:"status"`

	Data struct {
		ActiveTargets  []PrometheusTarget `json:"activeTargets"`
		DroppedTargets []PrometheusTarget `json:"droppedTargets"`
	} `json:"data"`

	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
}

func (c *PrometheusCollector) DiscoverTargets(
	hostIP string,
) ([]PrometheusTarget, error) {
	if hostIP == "" {
		return nil, fmt.Errorf(
			"target IP is required",
		)
	}

	params := url.Values{}
	params.Set("state", "active")

	body, err := c.request(
		"/api/v1/targets",
		params,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"discover prometheus targets: %w",
			err,
		)
	}

	var response targetsResponse

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf(
			"decode prometheus targets response: %w",
			err,
		)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf(
			"prometheus target discovery failed: %s: %s",
			response.ErrorType,
			response.Error,
		)
	}

	targets := make([]PrometheusTarget, 0)

	for _, target := range response.Data.ActiveTargets {
		if targetBelongsToHost(target, hostIP) {
			targets = append(targets, target)
		}
	}

	return targets, nil
}

func targetBelongsToHost(
	target PrometheusTarget,
	hostIP string,
) bool {
	for _, value := range []string{
		target.Instance(),
		target.Address(),
	} {
		if value == "" {
			continue
		}

		if value == hostIP ||
			strings.HasPrefix(value, hostIP+":") {
			return true
		}
	}

	return false
}
