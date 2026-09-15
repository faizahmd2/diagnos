package aiclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

type HTTPCaller struct {
	Client *http.Client
}

func NewHTTPCaller(client *http.Client) *HTTPCaller {
	if client == nil {
		client = &http.Client{}
	}

	return &HTTPCaller{
		Client: client,
	}
}

func (c *HTTPCaller) PostJSON(
	ctx context.Context,
	url string,
	body []byte,
	headers map[string]string,
) ([]byte, error) {

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create AI request: %w",
			err,
		)
	}

	request.Header.Set(
		"Content-Type",
		"application/json",
	)

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := c.Client.Do(request)
	if err != nil {
		return nil, fmt.Errorf(
			"AI request failed: %w",
			err,
		)
	}

	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to read AI response: %w",
			err,
		)
	}

	if response.StatusCode < 200 ||
		response.StatusCode >= 300 {

		return nil, fmt.Errorf(
			"AI request returned HTTP %d: %s",
			response.StatusCode,
			string(responseBody),
		)
	}

	if len(bytes.TrimSpace(responseBody)) == 0 {
		return nil, fmt.Errorf(
			"AI request returned an empty response",
		)
	}

	return responseBody, nil
}
