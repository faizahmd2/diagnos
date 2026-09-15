package aiclient

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type Client struct {
	Model Model
	HTTP  *HTTPCaller
}

type RequestPayload struct {
	URL     string
	Body    []byte
	Headers map[string]string
}

func NewClient(
	model Model,
	httpClient *http.Client,
) *Client {
	return &Client{
		Model: model,
		HTTP:  NewHTTPCaller(httpClient),
	}
}

func (c *Client) Call(
	ctx context.Context,
	prompt string,
) ([]byte, error) {

	if c == nil {
		return nil, fmt.Errorf(
			"AI client is nil",
		)
	}

	if prompt == "" {
		return nil, fmt.Errorf(
			"AI prompt cannot be empty",
		)
	}

	request, err := buildRequest(
		c.Model,
		prompt,
	)
	if err != nil {
		return nil, err
	}

	response, err := c.HTTP.PostJSON(
		ctx,
		request.URL,
		request.Body,
		request.Headers,
	)
	if err != nil {
		return nil, err
	}

	parsedResponse, err := parseResponse(
		c.Model,
		response,
	)
	if err != nil {
		return nil, err
	}

	return parsedResponse, nil
}

func buildRequest(
	model Model,
	prompt string,
) (RequestPayload, error) {

	switch model.Name {
	case "gemini/gemini-3.5-flash-lite":
		return buildGeminiRequest(model, prompt)

	default:
		return RequestPayload{}, fmt.Errorf(
			"AI request builder is not implemented for model %q",
			model.Name,
		)
	}
}

func ResolveModel(
	name string,
	configs map[string]ModelConfig,
) (Model, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return Model{}, fmt.Errorf(
			"AI model is not configured",
		)
	}

	config, ok := configs[name]
	if !ok {
		return Model{}, fmt.Errorf(
			"AI model %q is not configured",
			name,
		)
	}

	return Model{
		Name:   name,
		Config: config,
	}, nil
}

func parseResponse(
	model Model,
	body []byte,
) ([]byte, error) {

	switch model.Name {
	case "gemini/gemini-3.5-flash-lite":
		return parseGeminiResponse(body)

	default:
		return nil, fmt.Errorf(
			"AI response parser is not implemented for model %q",
			model.Name,
		)
	}
}
