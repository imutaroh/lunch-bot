package main

import (
	"log"

	"github.com/androots/lunch-bot/internal/config"
	"github.com/androots/lunch-bot/internal/handler"
	"github.com/androots/lunch-bot/internal/repository"
	"github.com/androots/lunch-bot/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// DI: repository → service → handler の順で注入
	slackRepo := repository.NewSlackClient(cfg.SlackBotToken)
	lunchSvc := service.NewLunchService(slackRepo, cfg.SlackChannelID)
	lunchHdr := handler.NewLunchHandler(lunchSvc)

	if err := lunchHdr.Run(); err != nil {
		log.Fatalf("run: %v", err)
	}
}
