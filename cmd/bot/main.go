package main

import (
	"fmt"
	"log"
	"os"

	"github.com/imutaakihiro/lunch-bot/internal/config"
	"github.com/imutaakihiro/lunch-bot/internal/handler"
	"github.com/imutaakihiro/lunch-bot/internal/repository"
	"github.com/imutaakihiro/lunch-bot/internal/service"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	slackRepo := repository.NewSlackClient(cfg.SlackBotToken)
	lunchSvc := service.NewLunchService(slackRepo, cfg.SlackChannelID)
	lunchHdr := handler.NewLunchHandler(lunchSvc)

	mode := os.Args[1]
	switch mode {
	case "recruit":
		err = lunchHdr.Recruit()
	case "announce":
		err = lunchHdr.Announce()
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", mode)
		usage()
		os.Exit(2)
	}

	if err != nil {
		log.Fatalf("%s: %v", mode, err)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: lunch-bot <recruit|announce>")
}
