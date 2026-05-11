package config

import (
	"fmt"
	"os"
)

type Config struct {
	SlackBotToken  string
	SlackChannelID string
}

// 環境変数を取得する関数
func Load() (*Config, error) {
	// いまのプロセスの環境変数を読む
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

	// うまくいったときは、*Configにcfg、errorはnilが入る
	return cfg, nil
}
