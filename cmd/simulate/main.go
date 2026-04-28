// cmd/simulate は本番 Slack に触らずに announce フローを擬似実行するツール。
//
// 使い方:
//
//	go run ./cmd/simulate -n 8     # 8人参加した想定でグループ分けを見る
//	go run ./cmd/simulate -n 0     # 0人 → 「お休み」メッセージのパスを見る
//	go run ./cmd/simulate -n 11    # 11人 → 4-4-3 の3グループ分割を見る
//
// 仕組み: service.SlackRepository インターフェイスを満たす偽実装 (fakeSlack) を
// 定義し、本番の repository.SlackClient の代わりに service.NewLunchService に
// 渡す。サービスは「Slack と話しているつもり」で動くが、API 呼び出しはすべて
// メモリ内で完結し、ネットワークも環境変数も不要。
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/imutaakihiro/lunch-bot/internal/repository"
	"github.com/imutaakihiro/lunch-bot/internal/service"
)

// fakeSlack は SlackRepository の5メソッドを満たす偽実装。
// 本番の SlackClient と外側からは見分けがつかないが、内部では
// 事前に仕込んだ値を返すだけでネットワークも認証も発生しない。
type fakeSlack struct {
	botUserID     string
	reactionUsers []string
	recruitText   string
}

// PostMessage は本番なら Slack に投稿するメソッド。
// シミュレータでは「投稿されたつもりのテキスト」をコンソールに出すだけ。
func (f *fakeSlack) PostMessage(channel, text string) (string, error) {
	fmt.Println("─── [fake] PostMessage ───")
	fmt.Println(text)
	fmt.Println("──────────────────────────")
	return "fake-post-ts", nil
}

// AddReaction は recruit モードでだけ呼ばれる (announce では通らない)。
// 念のため何もしない実装にしておく。
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
// 偽実装では仕込んだ botUserID をそのまま返せばよい。
func (f *fakeSlack) WhoAmI() (string, error) {
	return f.botUserID, nil
}

// RecentBotMessages は「直近に bot が投稿した募集メッセージ」を1件だけ仕込んで返す。
// テキストの先頭が service.recruitPrefix と一致するので、サービスはこれを
// 募集投稿として認識して処理を進める。冪等性チェック (発表/お休みなら skip) も
// このメッセージは引っかからないので素通りする。
func (f *fakeSlack) RecentBotMessages(channel, botUserID string, sinceHours int) ([]repository.BotMessage, error) {
	return []repository.BotMessage{
		{TS: "fake-recruit-ts", Text: f.recruitText},
	}, nil
}

func main() {
	n := flag.Int("n", 5, "bot自身を除いた人間の参加者数")
	flag.Parse()

	if *n < 0 {
		log.Fatalf("参加者数は0以上で指定してください: %d", *n)
	}

	const botID = "Ufakebot"

	// 参加者リストを組み立てる。bot自身を入れておくのが大事:
	// 「bot 除外ロジックがちゃんと働くか」もこの仕掛けで検証できる。
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

	fmt.Printf("[simulate] 人間 %d 人 + bot 1 = 計 %d 人がリアクションした想定で announce 実行\n\n", *n, *n+1)

	svc := service.NewLunchService(fake, "fake-channel")
	if err := svc.RunAnnounce(); err != nil {
		log.Fatalf("announce: %v", err)
	}

	fmt.Println("\n[simulate] 完了")
}
