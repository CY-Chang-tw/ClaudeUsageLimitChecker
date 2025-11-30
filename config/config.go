package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	ClaudeSessionKey  string
	ClaudeOrgID       string
	DiscordWebhookURL string
	UsageThreshold    float64
	CheckInterval     int
	WarningLevels     []float64
}

func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	cfg := &Config{
		ClaudeSessionKey:  os.Getenv("CLAUDE_SESSION_KEY"),
		ClaudeOrgID:       os.Getenv("CLAUDE_ORG_ID"),
		DiscordWebhookURL: os.Getenv("DISCORD_WEBHOOK_URL"),
	}

	// Parse usage threshold
	thresholdStr := os.Getenv("USAGE_THRESHOLD")
	if thresholdStr == "" {
		thresholdStr = "80"
	}
	threshold, err := strconv.ParseFloat(thresholdStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid USAGE_THRESHOLD: %w", err)
	}
	cfg.UsageThreshold = threshold

	// Parse check interval
	intervalStr := os.Getenv("CHECK_INTERVAL")
	if intervalStr == "" {
		intervalStr = "60"
	}
	interval, err := strconv.Atoi(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid CHECK_INTERVAL: %w", err)
	}
	cfg.CheckInterval = interval

	// Parse warning levels
	levelsStr := os.Getenv("WARNING_LEVELS")
	if levelsStr == "" {
		levelsStr = "80,90,95"
	}
	levelStrs := strings.Split(levelsStr, ",")
	cfg.WarningLevels = make([]float64, 0, len(levelStrs))
	for _, ls := range levelStrs {
		level, err := strconv.ParseFloat(strings.TrimSpace(ls), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid WARNING_LEVELS: %w", err)
		}
		cfg.WarningLevels = append(cfg.WarningLevels, level)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.ClaudeSessionKey == "" {
		return fmt.Errorf("CLAUDE_SESSION_KEY is required")
	}
	if c.ClaudeOrgID == "" {
		return fmt.Errorf("CLAUDE_ORG_ID is required")
	}
	if c.DiscordWebhookURL == "" {
		return fmt.Errorf("DISCORD_WEBHOOK_URL is required")
	}
	if c.UsageThreshold < 0 || c.UsageThreshold > 100 {
		return fmt.Errorf("USAGE_THRESHOLD must be between 0 and 100")
	}
	if c.CheckInterval <= 0 {
		return fmt.Errorf("CHECK_INTERVAL must be greater than 0")
	}
	for _, level := range c.WarningLevels {
		if level < 0 || level > 100 {
			return fmt.Errorf("WARNING_LEVELS must be between 0 and 100")
		}
	}
	return nil
}
