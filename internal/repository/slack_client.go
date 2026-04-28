package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

type postMessageResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	TS    string `json:"ts"`
}

// PostMessage は指定チャンネルにテキストを投稿し、メッセージのタイムスタンプ(ts)を返す。
// ts はそのメッセージの一意なIDで、後でリアクション取得や返信に使う。
func (c *SlackClient) PostMessage(channel, text string) (string, error) {
	payload := map[string]string{
		"channel": channel,
		"text":    text,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result postMessageResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if !result.OK {
		return "", fmt.Errorf("slack api error: %s", result.Error)
	}

	return result.TS, nil
}

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
// 該当リアクションが無い場合は空スライス。
func (c *SlackClient) GetReactionUsers(channel, timestamp, emoji string) ([]string, error) {
	params := url.Values{}
	params.Set("channel", channel)
	params.Set("timestamp", timestamp)

	req, err := http.NewRequest("GET", "https://slack.com/api/reactions.get?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var result reactionsGetResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if !result.OK {
		return nil, fmt.Errorf("slack api error: %s", result.Error)
	}

	for _, r := range result.Message.Reactions {
		if r.Name == strings.Trim(emoji, ":") {
			return r.Users, nil
		}
	}
	return []string{}, nil
}
