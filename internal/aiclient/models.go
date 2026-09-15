package aiclient

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

func buildGeminiRequest(
	model Model,
	prompt string,
) (RequestPayload, error) {

	if model.Config.BaseURL == "" {
		return RequestPayload{}, fmt.Errorf(
			"base URL is not configured for AI model %q",
			model.Name,
		)
	}

	modelName := strings.TrimPrefix(
		model.Name,
		"gemini/",
	)

	if modelName == "" {
		return RequestPayload{}, fmt.Errorf(
			"gemini model name is empty",
		)
	}

	endpoint := fmt.Sprintf(
		"%s/v1beta/models/%s:generateContent",
		strings.TrimRight(model.Config.BaseURL, "/"),
		modelName,
	)

	payload := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return RequestPayload{}, fmt.Errorf(
			"failed to encode Gemini request: %w",
			err,
		)
	}

	headers := make(map[string]string)

	if model.Config.APIKeyEnv != "" {
		apiKey := os.Getenv(model.Config.APIKeyEnv)

		if apiKey == "" {
			return RequestPayload{}, fmt.Errorf(
				"AI API key environment variable %q is not set",
				model.Config.APIKeyEnv,
			)
		}

		headers["x-goog-api-key"] = apiKey
	}

	return RequestPayload{
		URL:     endpoint,
		Body:    data,
		Headers: headers,
	}, nil
}

func parseGeminiResponse(
	body []byte,
) ([]byte, error) {

	var response geminiResponse

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf(
			"failed to parse Gemini response: %w",
			err,
		)
	}

	if len(response.Candidates) == 0 ||
		len(response.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf(
			"Gemini returned an empty candidate list or text payload",
		)
	}

	resultText := response.Candidates[0].Content.Parts[0].Text

	if strings.TrimSpace(resultText) == "" {
		return nil, fmt.Errorf(
			"Gemini returned an empty text content",
		)
	}

	return []byte(resultText), nil
}
