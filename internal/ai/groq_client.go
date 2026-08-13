package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Nishanth1812/devpulse/internal/logger"
)

const maxResponseBytes = 4 * 1024 * 1024

type groqClient struct {
	httpClient *http.Client
	apiKey     string
	model      string
	endpoint   string
}

func NewGroqClient(cfg GroqConfig) (Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("groq: API key is required")
	}
	model := cfg.Model
	if model == "" {
		model = defaultGroqModel
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultGroqTimeout
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultGroqBaseURL
	}
	return &groqClient{
		httpClient: &http.Client{Timeout: timeout},
		apiKey:     cfg.APIKey,
		model:      model,
		endpoint:   baseURL + "/chat/completions",
	}, nil
}

func (c *groqClient) Name() string {
	return ProviderGroq
}

func (c *groqClient) Generate(ctx context.Context, prompt string) (string, error) {
	body := groqRequest{
		Model:          c.model,
		Messages:       []groqMessage{{Role: "user", Content: prompt}},
		ResponseFormat: &groqResponseFormat{Type: "json_object"},
	}

	attempt := func(ctx context.Context) (string, error) {
		encoded, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("groq: encode request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(encoded))
		if err != nil {
			return "", fmt.Errorf("groq: build request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		start := time.Now()
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("groq: http request: %w", err)
		}
		defer resp.Body.Close()

		raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		if err != nil {
			return "", fmt.Errorf("groq: read response body: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			var apiErr groqErrorBody
			if json.Unmarshal(raw, &apiErr) == nil && apiErr.Error.Message != "" {
				return "", fmt.Errorf("groq: api error (%s): %s", apiErr.Error.Code, apiErr.Error.Message)
			}
			return "", fmt.Errorf("groq: unexpected status %d", resp.StatusCode)
		}

		var result groqResponse
		if err := json.Unmarshal(raw, &result); err != nil {
			return "", fmt.Errorf("groq: decode response: %w", err)
		}
		if len(result.Choices) == 0 {
			return "", fmt.Errorf("groq: response contained no choices")
		}

		logger.LogAPICall("groq", c.model, result.Usage.TotalTokens, time.Since(start))

		return strings.TrimSpace(result.Choices[0].Message.Content), nil
	}

	return withRetry(ctx, attempt)
}
