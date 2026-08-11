package config

import (
	"fmt"
	"strconv"
	"strings"
)

func (m *Manager) Get(key string) (string, error) {
	switch key {
	case "model.fast":
		return m.config.ModelFast, nil
	case "model.deep":
		return m.config.ModelDeep, nil
	case "cache.hours":
		return strconv.Itoa(m.config.CacheDurationHours), nil
	case "fuzzy.threshold":
		return strconv.Itoa(m.config.FuzzyThreshold), nil
	default:
		return "", fmt.Errorf("unsupported config key %q", key)
	}
}

func (m *Manager) Set(key string, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("config value for %q cannot be empty", key)
	}

	switch key {
	case "model.fast":
		m.config.ModelFast = value
	case "model.deep":
		m.config.ModelDeep = value
	case "cache.hours":
		hours, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse cache.hours as integer: %w", err)
		}
		if hours <= 0 {
			return fmt.Errorf("cache.hours must be greater than zero")
		}
		m.config.CacheDurationHours = hours
	case "fuzzy.threshold":
		threshold, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse fuzzy.threshold as integer: %w", err)
		}
		if threshold <= 0 || threshold > 100 {
			return fmt.Errorf("fuzzy.threshold must be between 1 and 100")
		}
		m.config.FuzzyThreshold = threshold
	default:
		return fmt.Errorf("unsupported config key %q", key)
	}

	return m.Save()
}
