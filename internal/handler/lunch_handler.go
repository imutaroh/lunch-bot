package handler

import (
	"github.com/imutaakihiro/lunch-bot/internal/service"
)

// LunchHandler は cmd/bot/main.go から呼ばれるトリガー入口。
// サブコマンドごとに service の対応メソッドを呼ぶ。
type LunchHandler struct {
	svc *service.LunchService
}

func NewLunchHandler(svc *service.LunchService) *LunchHandler {
	return &LunchHandler{svc: svc}
}

func (h *LunchHandler) Recruit() error {
	return h.svc.RunRecruit()
}

func (h *LunchHandler) Announce() error {
	return h.svc.RunAnnounce()
}
