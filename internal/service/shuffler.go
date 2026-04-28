package service

import (
	"math/rand"
)

// Shuffle は参加者IDをランダムに3〜5人組のグループに分割する。
//
// 入力例: ["U001", "U002", "U003", "U004", "U005", "U006", "U007"]
// 出力例: [["U003", "U005", "U001", "U007"], ["U002", "U006", "U004"]]
//
// ルール:
//   - 各グループは 3 人以上 5 人以下
//   - 入力人数が 3 未満のときは nil を返す（呼び出し側でフォールバック）
//   - 順序はランダム化する
func Shuffle(userIDs []string) [][]string {
	if len(userIDs) < 3 {
		return nil
	}

	shuffled := make([]string, len(userIDs))
	copy(shuffled, userIDs)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	sizes := decideGroupSizes(len(shuffled))
	groups := make([][]string, 0, len(sizes))
	start := 0
	for _, size := range sizes {
		groups = append(groups, shuffled[start:start+size])
		start += size
	}
	return groups
}

// decideGroupSizes は n 人を 3〜5 人組に分けたときの各グループ人数を返す。
// 4 人組を基準に、端数が出たら +1/-1 して 3〜5 人の範囲に収める。
//
// 例: 9 → [5, 4]、11 → [4, 4, 3]、15 → [4, 4, 4, 3]
func decideGroupSizes(n int) []int {
	if n < 3 {
		return nil
	}

	// グループ数 = n/4 を四捨五入。(n+2)/4 で切り上げ/切り捨ての境界がちょうど 0.5 相当になる。
	numGroups := (n + 2) / 4
	base := n / numGroups     // 全グループの最低人数
	extra := n % numGroups    // この数だけ +1 人のグループを作る

	sizes := make([]int, numGroups)
	for i := 0; i < numGroups; i++ {
		if i < extra {
			sizes[i] = base + 1
		} else {
			sizes[i] = base
		}
	}
	return sizes
}
