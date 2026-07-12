package ai

import (
	"context"
	"fmt"
)

const (
	ProviderGroq   = "groq"
	ProviderGemini = "gemini"
)

type Client interface {
	Generate(ctx context.Context, prompt string) (string, error)
	Name() string
}

func NewClient(provider, apiKey, model string) (Client, error) {
	switch provider {
	case ProviderGroq:
		return NewGroqClient(GroqConfig{
			APIKey: apiKey,
			Model:  model,
		})
	case ProviderGemini:
		return NewGeminiClient(apiKey, model)
	default:
		return nil, fmt.Errorf("unknown AI provider %q (supported: groq, gemini)", provider)
	}
}
