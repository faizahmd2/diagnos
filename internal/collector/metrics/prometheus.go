package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Sample struct {
	Metric map[string]string `json:"metric"`
	Value  float64           `json:"value"`
	Time   time.Time         `json:"time"`
}

type PrometheusCollector struct {
	PrometheusURL string
	Timeout       time.Duration
	Client        *http.Client

	Auth PrometheusAuth
}

type PrometheusAuth struct {
	Type     string
	Token    string
	Username string
	Password string
}

func NewPrometheusCollector(
	prometheusURL string,
	auth PrometheusAuth,
) *PrometheusCollector {
	return &PrometheusCollector{
		PrometheusURL: strings.TrimRight(prometheusURL, "/"),
		Timeout:       30 * time.Second,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Auth: auth,
	}
}

type queryResponse struct {
	Status string `json:"status"`

	Data struct {
		ResultType string `json:"resultType"`

		Result []struct {
			Metric map[string]string `json:"metric"`
			Value  []json.RawMessage `json:"value"`
		} `json:"result"`
	} `json:"data"`

	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
}

type RangeQueryOptions struct {
	Start time.Time
	End   time.Time
	Step  time.Duration
}

type rangeQueryResponse struct {
	Status string `json:"status"`

	Data struct {
		ResultType string `json:"resultType"`

		Result []struct {
			Metric map[string]string   `json:"metric"`
			Values [][]json.RawMessage `json:"values"`
		} `json:"result"`
	} `json:"data"`

	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
}

func (c *PrometheusCollector) Query(
	promQL string,
) ([]Sample, error) {
	params := url.Values{}
	params.Set("query", promQL)

	output, err := c.request(
		"/api/v1/query",
		params,
	)
	if err != nil {
		return nil, err
	}

	var response queryResponse

	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf(
			"failed to decode prometheus response: %w",
			err,
		)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf(
			"prometheus query failed: %s: %s",
			response.ErrorType,
			response.Error,
		)
	}

	samples := make([]Sample, 0, len(response.Data.Result))

	for _, result := range response.Data.Result {
		if len(result.Value) != 2 {
			return nil, fmt.Errorf(
				"unexpected prometheus sample format",
			)
		}

		var timestamp float64
		var value string

		if err := json.Unmarshal(
			result.Value[0],
			&timestamp,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to parse prometheus timestamp: %w",
				err,
			)
		}

		if err := json.Unmarshal(
			result.Value[1],
			&value,
		); err != nil {
			return nil, fmt.Errorf(
				"failed to parse prometheus value: %w",
				err,
			)
		}

		numericValue, err := strconv.ParseFloat(
			value,
			64,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to parse prometheus value %q: %w",
				value,
				err,
			)
		}

		samples = append(samples, Sample{
			Metric: result.Metric,
			Value:  numericValue,
			Time: time.Unix(
				int64(timestamp),
				int64(
					(timestamp-float64(int64(timestamp)))*1e9,
				),
			),
		})
	}

	return samples, nil
}

func (c *PrometheusCollector) QueryRange(
	promQL string,
	options RangeQueryOptions,
) ([]Sample, error) {
	params := url.Values{}
	params.Set("query", promQL)
	params.Set("start", formatPrometheusTime(options.Start))
	params.Set("end", formatPrometheusTime(options.End))
	params.Set("step", formatPrometheusDuration(options.Step))

	output, err := c.request(
		"/api/v1/query_range",
		params,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to query prometheus range: %w",
			err,
		)
	}

	var response rangeQueryResponse

	if err := json.Unmarshal([]byte(output), &response); err != nil {
		return nil, fmt.Errorf(
			"failed to decode prometheus range response: %w",
			err,
		)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf(
			"prometheus range query failed: %s: %s",
			response.ErrorType,
			response.Error,
		)
	}

	samples := make([]Sample, 0)

	for _, result := range response.Data.Result {
		for _, value := range result.Values {
			if len(value) != 2 {
				return nil, fmt.Errorf(
					"unexpected prometheus range sample format",
				)
			}

			var timestamp float64
			var valueString string

			if err := json.Unmarshal(value[0], &timestamp); err != nil {
				return nil, fmt.Errorf(
					"failed to parse prometheus timestamp: %w",
					err,
				)
			}

			if err := json.Unmarshal(value[1], &valueString); err != nil {
				return nil, fmt.Errorf(
					"failed to parse prometheus value: %w",
					err,
				)
			}

			numericValue, err := strconv.ParseFloat(valueString, 64)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to parse prometheus value %q: %w",
					valueString,
					err,
				)
			}

			samples = append(samples, Sample{
				Metric: result.Metric,
				Value:  numericValue,
				Time: time.Unix(
					int64(timestamp),
					int64((timestamp-float64(int64(timestamp)))*1e9),
				),
			})
		}
	}

	return samples, nil
}

func formatPrometheusTime(t time.Time) string {
	return strconv.FormatFloat(
		float64(t.UnixNano())/1e9,
		'f',
		3,
		64,
	)
}

func formatPrometheusDuration(d time.Duration) string {
	seconds := d.Seconds()

	if seconds == float64(int64(seconds)) {
		return fmt.Sprintf("%.0fs", seconds)
	}

	return fmt.Sprintf("%.3fs", seconds)
}

func (c *PrometheusCollector) request(
	endpoint string,
	params url.Values,
) ([]byte, error) {
	requestURL := c.PrometheusURL + endpoint

	if len(params) > 0 {
		requestURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(
		http.MethodGet,
		requestURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create prometheus request: %w",
			err,
		)
	}

	switch c.Auth.Type {
	case "", "none":
		// No authentication.

	case "bearer":
		if c.Auth.Token == "" {
			return nil, fmt.Errorf(
				"prometheus bearer token is empty",
			)
		}

		req.Header.Set(
			"Authorization",
			"Bearer "+c.Auth.Token,
		)

	case "basic":
		req.SetBasicAuth(
			c.Auth.Username,
			c.Auth.Password,
		)

	default:
		return nil, fmt.Errorf(
			"unsupported prometheus auth type %q",
			c.Auth.Type,
		)
	}

	response, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf(
			"prometheus request failed: %w",
			err,
		)
	}

	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read prometheus response: %w",
			err,
		)
	}

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {
		return nil, fmt.Errorf(
			"prometheus returned HTTP %d: %s",
			response.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	return body, nil
}

func (c *PrometheusCollector) CheckConnection() error {
	_, err := c.request("/-/ready", nil)
	if err != nil {
		return fmt.Errorf(
			"prometheus connection failed: %w",
			err,
		)
	}

	return nil
}
