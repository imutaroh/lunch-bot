// cmd/simulate は announce フローを任意の参加者で擬似実行するツール。
//
// 使い方:
//
//	go run ./cmd/simulate -n 8                              # 8人の偽ユーザー
//	go run ./cmd/simulate -n 12 -post                       # 結果を実Slackに投稿 (mentionは偽IDなので通知は飛ばない)
//	go run ./cmd/simulate -users U0AAA,U0BBB,U0CCC -post    # ★ 実IDを指定: 該当ユーザーにガチで通知が飛ぶ
//
// -users は -n より優先される。-users の各 ID は実Slackユーザーなら通知され、
// 偽IDならテキストのまま表示される (両方混ぜてもOK)。
//
// 仕組み: service.SlackRepository の偽実装 (fakeSlack) を service に渡す。
// 読み取りは常に偽、書き込みは -post で本物のSlackに転送するハイブリッド。
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

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

	real        *repository.SlackClient
	realChannel string
}

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

func (f *fakeSlack) AddReaction(channel, timestamp, emoji string) error {
	return nil
}

func (f *fakeSlack) GetReactionUsers(channel, timestamp, emoji string) ([]string, error) {
	return f.reactionUsers, nil
}

func (f *fakeSlack) WhoAmI() (string, error) {
	return f.botUserID, nil
}

func (f *fakeSlack) RecentBotMessages(channel, botUserID string, sinceHours int) ([]repository.BotMessage, error) {
	return []repository.BotMessage{
		{TS: "fake-recruit-ts", Text: f.recruitText},
	}, nil
}

// parseUserList はカンマ区切りのID文字列をスライスに変換する。空・空白は無視。
func parseUserList(s string) []string {
	out := []string{}
	for _, id := range strings.Split(s, ",") {
		id = strings.TrimSpace(id)
		if id != "" {
			out = append(out, id)
		}
	}
	return out
}

func main() {
	n := flag.Int("n", 5, "bot自身を除いた偽ユーザー数 (-users と併用時は -users が優先)")
	usersStr := flag.String("users", "", "実ユーザーIDをカンマ区切りで指定 (例: U0AAA,U0BBB,U0CCC)")
	post := flag.Bool("post", false, "結果を実Slackに投稿する (要 SLACK_BOT_TOKEN / SLACK_CHANNEL_ID)")
	flag.Parse()

	const botID = "Ufakebot"

	// 人間側ユーザーリストを決める。-users が指定されたらそれを使い、
	// なければ -n で偽ID (U001, U002, ...) を生成する。
	var humans []string
	if *usersStr != "" {
		humans = parseUserList(*usersStr)
		fmt.Printf("[simulate] -users 指定: %v\n", humans)
	} else {
		if *n < 0 {
			log.Fatalf("参加者数は0以上で指定してください: %d", *n)
		}
		for i := 1; i <= *n; i++ {
			humans = append(humans, fmt.Sprintf("U%03d", i))
		}
	}

	// bot自身を先頭に入れる (除外ロジックの動作確認も兼ねる)
	users := append([]string{botID}, humans...)

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

	fmt.Printf("[simulate] 人間 %d 人 + bot 1 = 計 %d 人がリアクションした想定で announce 実行\n\n", len(humans), len(humans)+1)

	svc := service.NewLunchService(fake, "fake-channel")
	if err := svc.RunAnnounce(); err != nil {
		log.Fatalf("announce: %v", err)
	}

	fmt.Println("\n[simulate] 完了")
}
