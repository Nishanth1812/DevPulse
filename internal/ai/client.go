package ai

import (
	"context"
	"fmt"
	"strings"
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

// DefaultModel returns the provider's default model for a fast/deep use case.
func DefaultModel(provider string, deep bool) string {
	switch provider {
	case ProviderGemini:
		if deep {
			return defaultGeminiDeepModel
		}
		return defaultGeminiFastModel
	case ProviderGroq:
		return defaultGroqModel
	default:
		return ""
	}
}

// ModelCompatible reports whether a configured model name plausibly belongs to
// a provider. This prevents a gemini model saved in config from being sent to
// Groq (or vice versa), which would otherwise fail with an unknown-model error.
// Matching is by prefix so that provider names are never rejected merely for
// containing an ambiguous substring (e.g. a Groq model named "...gemini...").
func ModelCompatible(provider, model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return true
	}
	switch provider {
	case ProviderGemini:
		return strings.HasPrefix(model, "gemini-") || strings.HasPrefix(model, "gemini/")
	case ProviderGroq:
		return !strings.HasPrefix(model, "gemini-") && !strings.HasPrefix(model, "gemini/")
	default:
		return true
	}
}

// ResolveModel picks the model for a request: the configured value when set and
// compatible with the provider, otherwise the provider default. The boolean is
// true when a configured value was ignored as incompatible (caller may warn).
func ResolveModel(provider, configured string, deep bool) (string, bool) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		return DefaultModel(provider, deep), false
	}
	if !ModelCompatible(provider, configured) {
		return DefaultModel(provider, deep), true
	}
	return configured, false
}
