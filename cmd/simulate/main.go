// cmd/simulate は announce フローを任意の参加者数で擬似実行するツール。
//
// 使い方:
//
//	go run ./cmd/simulate -n 8              # 8人参加した想定 (コンソール出力のみ)
//	go run ./cmd/simulate -n 0              # 0人 → 「お休み」メッセージのパス
//	go run ./cmd/simulate -n 12 -post       # ★ 12人参加 + 結果を実際にSlackへ投稿
//
// 仕組み: service.SlackRepository インターフェイスを満たす偽実装 (fakeSlack) を
// service に渡す。読み取り (GetReactionUsers / RecentBotMessages / WhoAmI) は常に
// 偽実装で、N人を仕込める。書き込み (PostMessage) は -post を付けると本物の
// SlackClient へ委譲し、実チャンネルに発表が出る (mention は U001 等の偽IDなので
// テキストとして表示される)。
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/imutaakihiro/lunch-bot/internal/config"
	"github.com/imutaakihiro/lunch-bot/internal/repository"
	"github.com/imutaakihiro/lunch-bot/internal/service"
)

// fakeSlack は SlackRepository の5メソッドを満たす偽実装。
// real が nil でなければ PostMessage だけ本物の Slack に流す (ハイブリッドモード)。
type fakeSlack struct {
	botUserID     string
	reactionUsers []string
	recruitText   string

	real        *repository.SlackClient // 非nilなら PostMessage を実Slackへ転送
	realChannel string
}

// PostMessage はコンソールに出力 + (-post時のみ) 実Slackへも投稿。
func (f *fakeSlack) PostMessage(channel, text string) (string, error) {
	fmt.Println("─── [fake] PostMessage ───")
	fmt.Println(text)
	fmt.Println("──────────────────────────")
	if f.real != nil {
		ts, err := f.real.PostMessage(f.realChannel, text)
		if err != nil {
			return "", fmt.Errorf("real post: %w", err)
		}
		fmt.Printf("[real] Slack に実投稿しました ts=%s\n", ts)
		return ts, nil
	}
	return "fake-post-ts", nil
}

// AddReaction は recruit モードでだけ呼ばれる (announce では通らない)。
func (f *fakeSlack) AddReaction(channel, timestamp, emoji string) error {
	return nil
}

// GetReactionUsers は事前に仕込んだ参加者IDリストをそのまま返す。
// これがシミュレータの中核: 「N人が🍱を押した状態」の正体は
// このスライスにN個のIDが入っている、というだけ。
func (f *fakeSlack) GetReactionUsers(channel, timestamp, emoji string) ([]string, error) {
	return f.reactionUsers, nil
}

// WhoAmI は service が「自分のID」を取得するときに呼ぶ。
func (f *fakeSlack) WhoAmI() (string, error) {
	return f.botUserID, nil
}

// RecentBotMessages は「直近に bot が投稿した募集メッセージ」を1件だけ仕込んで返す。
// テキストの先頭が service.recruitPrefix と一致するので募集として認識される。
// 冪等性チェック (発表/お休みなら skip) もこのメッセージは素通りする。
func (f *fakeSlack) RecentBotMessages(channel, botUserID string, sinceHours int) ([]repository.BotMessage, error) {
	return []repository.BotMessage{
		{TS: "fake-recruit-ts", Text: f.recruitText},
	}, nil
}

func main() {
	n := flag.Int("n", 5, "bot自身を除いた人間の参加者数")
	post := flag.Bool("post", false, "結果を実際のSlackチャンネルに投稿する (要 SLACK_BOT_TOKEN / SLACK_CHANNEL_ID)")
	flag.Parse()

	if *n < 0 {
		log.Fatalf("参加者数は0以上で指定してください: %d", *n)
	}

	const botID = "Ufakebot"

	users := make([]string, 0, *n+1)
	users = append(users, botID)
	for i := 1; i <= *n; i++ {
		users = append(users, fmt.Sprintf("U%03d", i))
	}

	fake := &fakeSlack{
		botUserID:     botID,
		reactionUsers: users,
		recruitText:   ":knife_fork_plate: 今週のランチ参加者募集！\n(simulator が仕込んだダミー)",
	}

	if *post {
		cfg, err := config.Load()
		if err != nil {
			log.Fatalf("config (-post には SLACK_BOT_TOKEN と SLACK_CHANNEL_ID が必要): %v", err)
		}
		fake.real = repository.NewSlackClient(cfg.SlackBotToken)
		fake.realChannel = cfg.SlackChannelID
		fmt.Printf("[simulate] -post モード: 結果を %s に実投稿します\n", cfg.SlackChannelID)
	}

	fmt.Printf("[simulate] 人間 %d 人 + bot 1 = 計 %d 人がリアクションした想定で announce 実行\n\n", *n, *n+1)

	svc := service.NewLunchService(fake, "fake-channel")
	if err := svc.RunAnnounce(); err != nil {
		log.Fatalf("announce: %v", err)
	}

	fmt.Println("\n[simulate] 完了")
}
