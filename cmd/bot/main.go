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
	// 実行コマンドで、Recruit,Announceがない場合の分岐
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	// 環境変数を取得する関数を呼び出す
	cfg, err := config.Load()
	// もしもエラーがあったら、実行する
	if err != nil {
		// Fatal = 致命的 → ログを出してもう続けられないから死ね
		log.Fatalf("config: %v", err)
	}
	// 依存注入開始
	slackRepo := repository.NewSlackClient(cfg.SlackBotToken) // Tokenに依存している メッセージの送信とか、スタンプ関連の機能を定義している
	lunchSvc := service.NewLunchService(slackRepo, cfg.SlackChannelID)
	lunchHdr := handler.NewLunchHandler(lunchSvc)

	// いったん全部見る！コマンドの最初をmode に入れる
	mode := os.Args[1]
	switch mode {
	case "recruit":
		err = lunchHdr.Recruit()
	case "announce":
		err = lunchHdr.Announce()

	// modeがないというエラーを返す
	default:
		fmt.Fprintf(os.Stderr, "unknown mode: %s\n", mode)
		usage()
		os.Exit(2)
	}

	// エラーが起こったら終了
	if err != nil {
		log.Fatalf("%s: %v", mode, err)
	}
}

// usageはこうやって使ってね！という慣習名。
func usage() {
	fmt.Fprintln(os.Stderr, "Usage: lunch-bot <recruit|announce>")
}
