package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Nishanth1812/devpulse/internal/logger"
	"google.golang.org/genai"
)

const (
	defaultGeminiModel   = "gemini-2.5-flash"
	defaultGeminiTimeout = 60 * time.Second
)

type geminiClient struct {
	client *genai.Client
	model  string
}

func NewGeminiClient(apiKey, model string) (Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("gemini: API key is required")
	}
	if model == "" {
		model = defaultGeminiModel
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: create client: %w", err)
	}

	return &geminiClient{
		client: client,
		model:  model,
	}, nil
}

func (c *geminiClient) Name() string {
	return ProviderGemini
}

func (c *geminiClient) Generate(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultGeminiTimeout)
	defer cancel()

	attempt := func(ctx context.Context) (string, error) {
		start := time.Now()
		result, err := c.client.Models.GenerateContent(ctx, c.model, genai.Text(prompt), &genai.GenerateContentConfig{
			ResponseMIMEType: "application/json",
		})
		if err != nil {
			return "", fmt.Errorf("gemini: generate content: %w", err)
		}

		text := result.Text()
		totalTokens := 0
		if result.UsageMetadata != nil {
			totalTokens = int(result.UsageMetadata.TotalTokenCount)
		}

		logger.LogAPICall("gemini", c.model, totalTokens, time.Since(start))

		return strings.TrimSpace(text), nil
	}

	return withRetry(ctx, attempt)
}
