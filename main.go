package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	// 1. 環境変数からトークンとチャンネルIDを取得
	token := os.Getenv("SLACK_BOT_TOKEN")
	channel := os.Getenv("SLACK_CHANNEL_ID")
	if token == "" || channel == "" {
		log.Fatal("SLACK_BOT_TOKEN と SLACK_CHANNEL_ID を環境変数で設定してください")
	}

	// 2. Slack に送るデータを JSON で組み立てる
	payload := map[string]string{
		"channel": channel,
		"text":    "Hello from Go! 👋",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Fatal(err)
	}

	// 3. HTTP リクエストを作って送る
	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// 4. レスポンスを表示（Slack API の成否を確認）
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Slackからのレスポンス:", string(respBody))
}
