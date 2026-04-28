package service

import (
	"fmt"
	"time"
)

// NextMorningAt は「今から見て次の指定時刻 (JST)」の time.Time を返す。
// 例: いま火曜10:00 で hour=9 なら、水曜09:00 を返す。
// 例: いま火曜08:00 で hour=9 なら、火曜09:00 を返す（同日の未来時刻があればそれを優先）。
func NextMorningAt(now time.Time, hour, minute int) time.Time {
	jst := time.FixedZone("JST", 9*60*60)
	nowJST := now.In(jst)

	target := time.Date(nowJST.Year(), nowJST.Month(), nowJST.Day(), hour, minute, 0, 0, jst)
	if !target.After(nowJST) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}

// SleepUntil は指定時刻まで現在のgoroutineをブロックする。
// 過去時刻が渡された場合は即時 return。
func SleepUntil(t time.Time) {
	d := time.Until(t)
	if d <= 0 {
		return
	}
	fmt.Printf("[scheduler] %s まで待機します (約 %s)\n", t.Format("2006-01-02 15:04:05 MST"), d.Round(time.Second))
	time.Sleep(d)
}
