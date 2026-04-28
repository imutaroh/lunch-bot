package service

import (
	"fmt"
	"strings"

	"github.com/imutaakihiro/lunch-bot/internal/repository"
)

// SlackRepository は service が必要とする Slack 操作のインターフェース。
type SlackRepository interface {
	PostMessage(channel, text string) (string, error)
	AddReaction(channel, timestamp, emoji string) error
	GetReactionUsers(channel, timestamp, emoji string) ([]string, error)
	WhoAmI() (string, error)
	RecentBotMessages(channel, botUserID string, sinceHours int) ([]repository.BotMessage, error)
}

type LunchService struct {
	slack         SlackRepository
	channelID     string
	emoji         string
	lookbackHours int
}

func NewLunchService(slack SlackRepository, channelID string) *LunchService {
	return &LunchService{
		slack:         slack,
		channelID:     channelID,
		emoji:         "bento",
		lookbackHours: 26,
	}
}

// Slack は chat.postMessage の Unicode絵文字を内部で colon-code に正規化して保存する。
// (例: "🍽️" → ":knife_fork_plate:" / "🎉" → ":tada:")
// conversations.history から戻ってくる text は colon-code 形式なので、
// prefix 検索もそれに合わせる。投稿側 (recruitmentText) は Unicode のままでOK。
const (
	recruitPrefix  = ":knife_fork_plate: 今週のランチ参加者募集！"
	announcePrefix = ":tada: 今週のランチグループ決定！"
)

const recruitmentText = `🍽️ 今週のランチ参加者募集！
参加したい人は :bento: を押してね
締切: 火曜09:00 / 水曜のランチで 3〜5人組にランダム振り分けます`

const restMessageText = `😿 今週は参加者が少なかったのでお休みです
また来週叩いてください`

// RunRecruit は月曜09:00 (JST) に呼ばれる: 募集投稿 + bot 自身が🍱を1個押す。
func (s *LunchService) RunRecruit() error {
	fmt.Println("[recruit] 募集投稿を出します")
	ts, err := s.slack.PostMessage(s.channelID, recruitmentText)
	if err != nil {
		return fmt.Errorf("post recruitment: %w", err)
	}
	fmt.Printf("[recruit] 投稿成功 ts=%s\n", ts)

	if err := s.slack.AddReaction(s.channelID, ts, s.emoji); err != nil {
		return fmt.Errorf("add reaction: %w", err)
	}
	fmt.Println("[recruit] 自分で🍱を押しました")
	return nil
}

// RunAnnounce は火曜09:00 (JST) に呼ばれる: 募集投稿を見つけてリアクションを集計し、
// グループ分けして発表する。冪等性チェックあり (直近が発表済みならスキップ)。
func (s *LunchService) RunAnnounce() error {
	botID, err := s.slack.WhoAmI()
	if err != nil {
		return fmt.Errorf("who am i: %w", err)
	}
	fmt.Printf("[announce] bot user id = %s\n", botID)

	msgs, err := s.slack.RecentBotMessages(s.channelID, botID, s.lookbackHours)
	if err != nil {
		return fmt.Errorf("recent bot messages: %w", err)
	}
	if len(msgs) == 0 {
		return fmt.Errorf("bot 投稿が直近 %dh に見つからない (recruit が走っていない可能性)", s.lookbackHours)
	}

	// 冪等性チェック: 直近の bot 投稿が既に発表ならスキップ
	if strings.HasPrefix(msgs[0].Text, announcePrefix) {
		fmt.Println("[announce] 直近に発表済みのためスキップ (idempotency)")
		return nil
	}

	// 募集投稿を探す
	var recruit *repository.BotMessage
	for i := range msgs {
		if strings.HasPrefix(msgs[i].Text, recruitPrefix) {
			recruit = &msgs[i]
			break
		}
	}
	if recruit == nil {
		return fmt.Errorf("募集投稿が直近 %dh に見つからない", s.lookbackHours)
	}
	fmt.Printf("[announce] 募集投稿を発見 ts=%s\n", recruit.TS)

	users, err := s.slack.GetReactionUsers(s.channelID, recruit.TS, s.emoji)
	if err != nil {
		return fmt.Errorf("get reactions: %w", err)
	}
	users = excludeUser(users, botID)
	fmt.Printf("[announce] 参加者 %d 人 (bot自身を除外後)\n", len(users))

	if len(users) < 3 {
		if _, err := s.slack.PostMessage(s.channelID, restMessageText); err != nil {
			return fmt.Errorf("post rest message: %w", err)
		}
		return nil
	}

	groups := Shuffle(users)
	announcement := buildAnnouncement(groups)
	if _, err := s.slack.PostMessage(s.channelID, announcement); err != nil {
		return fmt.Errorf("post announcement: %w", err)
	}
	fmt.Println("[announce] グループ発表完了")
	return nil
}

func excludeUser(users []string, exclude string) []string {
	out := make([]string, 0, len(users))
	for _, u := range users {
		if u != exclude {
			out = append(out, u)
		}
	}
	return out
}

func buildAnnouncement(groups [][]string) string {
	var sb strings.Builder
	sb.WriteString("🎉 今週のランチグループ決定！\n\n")
	for i, g := range groups {
		mentions := make([]string, len(g))
		for j, uid := range g {
			mentions[j] = fmt.Sprintf("<@%s>", uid)
		}
		fmt.Fprintf(&sb, "グループ%c: %s\n", 'A'+i, strings.Join(mentions, " "))
	}
	sb.WriteString("\n水曜のランチで楽しんで！🍱")
	return sb.String()
}
