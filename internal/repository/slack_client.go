package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type SlackClient struct {
	token      string
	httpClient *http.Client
}

func NewSlackClient(token string) *SlackClient {
	return &SlackClient{
		token:      token,
		httpClient: http.DefaultClient,
	}
}

// --- chat.postMessage ---

type postMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	TS    string `json:"ts"`
}

func (c *SlackClient) PostMessage(channel, text string) (string, error) {
	payload := map[string]string{
		"channel": channel,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	respBody, err := c.doJSON("POST", "https://slack.com/api/chat.postMessage", body)
	if err != nil {
		return "", err
	}
	var result postMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("slack chat.postMessage error: %s", result.Error)
	}
	return result.TS, nil
}

// --- reactions.add ---

type reactionsAddResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// AddReaction は指定メッセージに emoji リアクションを bot 名義で付ける。
// emoji は "bento" のようにコロン無し（":bento:" でもOK、内部で剥がす）。
func (c *SlackClient) AddReaction(channel, timestamp, emoji string) error {
	payload := map[string]string{
		"channel":   channel,
		"timestamp": timestamp,
		"name":      strings.Trim(emoji, ":"),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	respBody, err := c.doJSON("POST", "https://slack.com/api/reactions.add", body)
	if err != nil {
		return err
	}
	var result reactionsAddResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("slack reactions.add error: %s", result.Error)
	}
	return nil
}

// --- reactions.get ---

type reactionsGetResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
	Message struct {
		Reactions []struct {
			Name  string   `json:"name"`
			Users []string `json:"users"`
		} `json:"reactions"`
	} `json:"message"`
}

// GetReactionUsers は指定メッセージに押された emoji リアクションのユーザーID一覧を返す。
// bot 自身の除外は呼び出し側 (service) でやる。該当リアクションが無い場合は空スライス。
func (c *SlackClient) GetReactionUsers(channel, timestamp, emoji string) ([]string, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("timestamp", timestamp)

	respBody, err := c.doGet("https://slack.com/api/reactions.get?" + params.Encode())
	if err != nil {
		return nil, err
	}
	var result reactionsGetResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("slack reactions.get error: %s", result.Error)
	}
	for _, r := range result.Message.Reactions {
		if r.Name == strings.Trim(emoji, ":") {
			return r.Users, nil
		}
	}
	return []string{}, nil
}

// --- auth.test ---

type authTestResponse struct {
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	UserID string `json:"user_id"`
}

// WhoAmI は bot 自身の Slack user ID を返す。「自分が押したリアクションを除外する」用。
func (c *SlackClient) WhoAmI() (string, error) {
	respBody, err := c.doJSON("POST", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return "", err
	}
	var result authTestResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("slack auth.test error: %s", result.Error)
	}
	return result.UserID, nil
}

// --- conversations.history ---

type BotMessage struct {
	TS   string
	Text string
}

type historyResponse struct {
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
	Messages []struct {
		User string `json:"user"`
		Text string `json:"text"`
		TS   string `json:"ts"`
	} `json:"messages"`
}

// RecentBotMessages は指定チャンネルの直近 sinceHours 時間以内に投稿された、
// botUserID による投稿のみを返す。Slack API の既定通り「新しい順」で並ぶ。
func (c *SlackClient) RecentBotMessages(channel, botUserID string, sinceHours int) ([]BotMessage, error) {
	oldest := time.Now().Add(-time.Duration(sinceHours) * time.Hour).Unix()
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("oldest", strconv.FormatInt(oldest, 10))
	params.Set("limit", "100")

	respBody, err := c.doGet("https://slack.com/api/conversations.history?" + params.Encode())
	if err != nil {
		return nil, err
	}
	var result historyResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("slack conversations.history error: %s", result.Error)
	}

	out := make([]BotMessage, 0, len(result.Messages))
	for _, m := range result.Messages {
		if m.User == botUserID {
			out = append(out, BotMessage{TS: m.TS, Text: m.Text})
		}
	}
	return out, nil
}

// --- HTTP helpers ---

func (c *SlackClient) doJSON(method, url string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (c *SlackClient) doGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
