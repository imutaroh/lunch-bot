package config

import (
	"fmt"
	"os"
)

type Config struct {
	SlackBotToken  string
	SlackChannelID string
}

func Load() (*Config, error) {
	cfg := &Config{
		SlackBotToken:  os.Getenv("SLACK_BOT_TOKEN"),
		SlackChannelID: os.Getenv("SLACK_CHANNEL_ID"),
	}

	if cfg.SlackBotToken == "" {
		return nil, fmt.Errorf("SLACK_BOT_TOKEN is required")
	}
	if cfg.SlackChannelID == "" {
		return nil, fmt.Errorf("SLACK_CHANNEL_ID is required")
	}

	return cfg, nil
}
