package service

import (
	"fmt"
	"strings"
	"time"
)

// SlackRepository は service が必要とする Slack 操作のインターフェース。
// 具象実装は repository パッケージに置く。
type SlackRepository interface {
	PostMessage(channel, text string) (string, error)
	GetReactionUsers(channel, timestamp, emoji string) ([]string, error)
}

type LunchService struct {
	slack     SlackRepository
	channelID string
	emoji     string
}

func NewLunchService(slack SlackRepository, channelID string) *LunchService {
	return &LunchService{
		slack:     slack,
		channelID: channelID,
		emoji:     "bento",
	}
}

const recruitmentText = `🍽️ 今週のランチ参加者募集！
参加したい人は :bento: を押してね
締切: 翌日 09:00 / 3〜5人組でランダムに振り分けます`

// RunSession は1回分のランチ抽選を最後まで実行する。
// 1) 募集投稿 → 2) 翌日09:00まで待機 → 3) リアクション集計 → 4) グループ発表
func (s *LunchService) RunSession() error {
	fmt.Println("[lunch] 募集投稿を出します")
	ts, err := s.slack.PostMessage(s.channelID, recruitmentText)
	if err != nil {
		return fmt.Errorf("post recruitment: %w", err)
	}
	fmt.Printf("[lunch] 投稿成功 ts=%s\n", ts)

	deadline := NextMorningAt(time.Now(), 9, 0)
	SleepUntil(deadline)

	fmt.Println("[lunch] リアクションを集計します")
	users, err := s.slack.GetReactionUsers(s.channelID, ts, s.emoji)
	if err != nil {
		return fmt.Errorf("get reactions: %w", err)
	}
	fmt.Printf("[lunch] 参加者 %d 人\n", len(users))

	if len(users) < 3 {
		_, err := s.slack.PostMessage(s.channelID, "😿 今週は参加者が少なかったのでお休みです\nまた来週叩いてください")
		if err != nil {
			return fmt.Errorf("post rest message: %w", err)
		}
		return nil
	}

	groups := Shuffle(users)
	announcement := buildAnnouncement(groups)

	if _, err := s.slack.PostMessage(s.channelID, announcement); err != nil {
		return fmt.Errorf("post announcement: %w", err)
	}
	fmt.Println("[lunch] グループ発表完了")
	return nil
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
	sb.WriteString("\n楽しいランチを！🍱")
	return sb.String()
}
