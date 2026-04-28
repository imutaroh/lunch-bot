package service

import (
	"fmt"
	"testing"
)

// TestDecideGroupSizes_3to20 は 3〜20 人について、
// すべてのグループが 3〜5 人の範囲に収まり、合計が元の人数と一致することを確認する。
func TestDecideGroupSizes_3to20(t *testing.T) {
	for n := 3; n <= 20; n++ {
		sizes := decideGroupSizes(n)

		sum := 0
		for _, s := range sizes {
			if s < 3 || s > 5 {
				t.Errorf("n=%d: グループ人数 %d が仕様違反（3〜5の範囲外）: %v", n, s, sizes)
			}
			sum += s
		}
		if sum != n {
			t.Errorf("n=%d: 合計 %d が一致しない: %v", n, sum, sizes)
		}

		t.Logf("n=%2d → %v", n, sizes)
	}
}

func TestDecideGroupSizes_TooFew(t *testing.T) {
	for _, n := range []int{0, 1, 2} {
		if got := decideGroupSizes(n); got != nil {
			t.Errorf("n=%d: nil を期待、got=%v", n, got)
		}
	}
}

// Shuffle 全体の動きを1度だけ実行して、グループ合計人数が入力と一致するか確認。
func TestShuffle_PreservesUsers(t *testing.T) {
	users := make([]string, 9)
	for i := range users {
		users[i] = fmt.Sprintf("U%03d", i+1)
	}
	groups := Shuffle(users)

	seen := map[string]bool{}
	for _, g := range groups {
		for _, u := range g {
			if seen[u] {
				t.Errorf("ユーザー %s が重複しています", u)
			}
			seen[u] = true
		}
	}
	if len(seen) != len(users) {
		t.Errorf("人数が合わない: expected=%d got=%d", len(users), len(seen))
	}
}
