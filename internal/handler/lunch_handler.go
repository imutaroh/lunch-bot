package handler

import (
	"github.com/androots/lunch-bot/internal/service"
)

// LunchHandler は外部からのトリガー（今は cmd/bot/main.go から直接呼ぶ）を受けて
// service 層を起動する責務を持つ。
// 将来 HTTP / Slash Command などのトリガーが増えたら、ここに分岐を追加する。
type LunchHandler struct {
	svc *service.LunchService
}

func NewLunchHandler(svc *service.LunchService) *LunchHandler {
	return &LunchHandler{svc: svc}
}

func (h *LunchHandler) Run() error {
	return h.svc.RunSession()
}
