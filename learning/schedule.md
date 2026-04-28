# lunch-bot 読解スケジュール — at a glance

このファイルは "毎日見ながら進める用"。詳細な思想は `plan.md`、時系列ログは `scrap.md`。

---

## 全体マップ

```
Phase 1 ✅  vibe-code MVP (ターミナル起動方式)
Phase 2 ✅  GitHub Actions cron 化 (自動運用)
Phase 3 ◀  読解 (今ここ★ — 4日 + バッファ1日)
Phase 4 📅  自分で書き直し (将来)
```

## 4日スケジュール

| Day | 時間 | 主タスク | 出力 |
|---|---|---|---|
| Day 0 | 30分 | 比較読み | `notes/00_my_hypothesis.md` |
| Day 1 | 5.5h | Loop 1+2+3 + デイリーレビュー | `notes/01〜04`, `notes/05` 前半 |
| Day 2 | 6h | Loop 4+5+6前半 + 破壊テスト + DR | `notes/05` 完成, `06`, `07` 前半 |
| Day 3 | 6h | Loop 6後半+7+8 + 破壊テスト + DR | `notes/07` 完成, `08`, `00_overview` |
| Day 4 | 6h | Zenn記事 + スライド派生 + DR | `article.md`, `slides.pdf` |
| Day 5 | バッファ | リハ / Level 3 つまみ食い | (任意) |

(DR = デイリーレビュー)

---

## Loop 詳細表

### Loop 0 (Day 0 / 30分): 比較読み

| 項目 | 内容 |
|---|---|
| 読まない | コード一切 |
| 読む | `spec.md` + `README.md` |
| 書く | `notes/00_my_hypothesis.md` (全体構成 / データ流れ / 想定エラー / 主要関数の予想 / 困りそうな所 — 各3〜5行) |
| 完了 | ファイル存在、5項目すべて埋まってる |

### Loop 1 (Day 1 / 1.5h): エントリポイント

| 項目 | 内容 |
|---|---|
| 読む | `main.go` (50行) + `cmd/bot/main.go` (45行) |
| 概念 | package, import, `os.Getenv` vs `os.Args`, `log.Fatal`, DI 配線, switch 分岐 |
| 書く | `notes/01_main.md`, `notes/02_cmd_bot.md` |
| 逆設計問い | なぜ `main.go` と `cmd/bot/main.go` の2つある？ なぜ switch？ |
| 要約 | "main.goは50行のPhase 1記念碑、cmd/bot/main.goは本体エントリで config→DI→分岐" |

### Loop 2 (Day 1 / 1.5h): config + handler

| 項目 | 内容 |
|---|---|
| 読む | `internal/config/config.go` (27行) + `internal/handler/lunch_handler.go` (22行) |
| 概念 | struct, ポインタレシーバ, error 返値, 最小レイヤの存在意義 |
| 書く | `notes/03_config.md`, `notes/04_handler.md` |
| 逆設計問い | handler は service を呼ぶだけ。なぜ存在？ |
| 要約 | "configは環境変数を構造体に詰める、handlerは service 呼び出しの薄いラッパー" |

### Loop 3 (Day 1 / 2h): service 前半 ★ interface 初対面

| 項目 | 内容 |
|---|---|
| 読む | `internal/service/lunch_service.go` 前半 (interface定義 + RunRecruit、~70行) |
| 概念 | **interface (★最重要)**, 依存性注入, `fmt.Errorf("%w", err)`, const |
| 書く | `notes/05_service.md` 前半 |
| 逆設計問い | interface 切らず具象 `*SlackClient` を渡しても動く。なぜ interface？ |
| 要約 | "interfaceがあるから、本物SlackClientと fakeSlack をサービスに差し替えられる" |

### Loop 4 (Day 2 / 2h): service 後半 (RunAnnounce)

| 項目 | 内容 |
|---|---|
| 読む | `lunch_service.go` 後半: `RunAnnounce` (~80行) |
| 概念 | 早期 return, 冪等性チェック, prefix 判定, bot自身除外, slice 操作, フォールバック |
| 書く | `notes/05_service.md` 完成。**8段の処理ステップ** をリスト化 |
| 逆設計問い | "お休み" も冪等性対象にした理由は？ |
| 要約 | "RunAnnounceは8段: WhoAmI→履歴→冪等→募集探索→集計→bot除外→人数判定→投稿" |

### Loop 5 (Day 2 / 2h): shuffler + テスト

| 項目 | 内容 |
|---|---|
| 読む | `shuffler.go` (60行) + `shuffler_test.go` (60行) |
| 概念 | `math/rand`, `rand.Shuffle`, 数学 (`decideGroupSizes`), テーブル駆動テスト |
| 書く | `notes/06_shuffler.md`。**12人 → [4,4,4]** の数学を1段ずつ追う |
| 逆設計問い | `decideGroupSizes` を再帰で書くこともできた。なぜループ？ |
| 要約 | "Shuffle は rand.Shuffle → decideGroupSizes でサイズ列 → スライス区切り" |

### Loop 6 前半 (Day 2 / 1h): slack_client.go 前半

| 項目 | 内容 |
|---|---|
| 読む | `slack_client.go` 前半 (`PostMessage`, `AddReaction`、~80行) |
| 概念 | `net/http`, `encoding/json`, 構造体タグ, POST + Authorization Bearer |
| 書く | `notes/07_slack_client.md` 前半 |
| 逆設計問い | `doJSON` ヘルパーの抽出理由は？ なかったらコードはどう肥大化する？ |
| 要約 | "PostMessage と AddReaction は JSON POST、共通処理は doJSON に抽出" |

### Day 2 破壊テスト (30分)

| 項目 | 内容 |
|---|---|
| 実験案 | `excludeUser` を消したらどうなる？ シミュレータ `-n 8` で検証 |
| 期待観察 | bot自身が参加者扱い、グループに混じる、人数 +1 |
| 出力 | `scrap.md` に観察ログを残す |

### Loop 6 後半 (Day 3 / 2h): slack_client.go 後半

| 項目 | 内容 |
|---|---|
| 読む | `slack_client.go` 後半 (`auth.test`, `conversations.history`, helpers、~130行) |
| 概念 | GET, query params, `doGet` ヘルパー, `oldest` パラメータ |
| 書く | `notes/07_slack_client.md` 完成 |
| 逆設計問い | botUserID 一致でフィルタを repository でやってる。service 側でやることもできた。なぜ repository？ |
| 要約 | "5つのSlack APIをHTTPで叩くラッパー、共通処理は doJSON/doGet にまとめ" |

### Loop 7 (Day 3 / 2h): シミュレータ — interface の威力体感

| 項目 | 内容 |
|---|---|
| 読む | `cmd/simulate/main.go` (110行) |
| 概念 | テストダブル (fake), 依存性注入の効用, `flag` パッケージ, ハイブリッド (read=fake, write=real) |
| 書く | `notes/08_simulate.md`。**「同じservice が、本物とfakeで動く」** の発見をメモ |
| 逆設計問い | なぜ `cmd/bot` に統合せず、別 `cmd/simulate` にした？ |
| 要約 | "fakeSlackがSlackRepositoryを満たすから、本物Slackなしでロジック検証ができる" |

### Loop 8 (Day 3 / 1h): overview + 設計判断 5つ

| 項目 | 内容 |
|---|---|
| 書く | `notes/00_overview.md` (新規): (1) ETL対応図 (2) 5つの設計判断、各200-400字 |
| 5つ | interface / サブコマンド / 冪等性 / prefix識別 / simulator |
| 逆設計問い | この lunch-bot を ETL 以外の構造にできるとしたら？ (例: イベント駆動) |
| 要約 | "lunch-bot は Slack を Extract+Load 両方の口にした最小 ETL、変換の核は excludeUser+Shuffle" |

### Day 3 破壊テスト (30分)

| 項目 | 内容 |
|---|---|
| 実験案 | 冪等性チェックを消す。シミュレータ `-post` で連続実行 |
| 期待観察 | (本来 skip される) 2度目のお休みメッセージが投稿される |
| 出力 | `scrap.md` |

### Day 4: Zenn記事 (3.5h)

| 項目 | 内容 |
|---|---|
| 目標 | 3000字以上、コードブロック 10個以上 |
| 構成 | 1.きっかけ → 2.lunch-botとは → 3.ETL構造 → 4.コードツアー → 5.設計判断5つ → 6.Phase 2バグ2つ → 7.シミュレータ → 8.学び |
| 出力 | `learning/article_draft.md` (公開は任意、下書き状態でも可) |

### Day 4: スライド派生 (2h)

| 項目 | 内容 |
|---|---|
| 枚数 | 10〜15枚 |
| 元 | Zenn 記事の章を圧縮 |
| 流れ | タイトル → 全体図 → 各層 → 設計判断 → 学び → まとめ |

---

## 毎日のルーチン

| 時間帯 | 内容 |
|---|---|
| 朝 (5分) | `plan.md` のチェックボックス確認、今日のループを宣言 |
| 各 Loop 後 (15分) | `notes/` 更新 + `scrap.md` に時系列メモ |
| 夜 (30分) | デイリーレビュー (3+3+3) を `scrap.md` に書く |

## 詰まった時のルール

| 状況 | 対処 |
|---|---|
| 5分詰まる | 同じ場所を5分見つめる + 「何が分からないか」を言葉に |
| 15分詰まる | Claude に**仮説ベース**で質問 (例: "X だと思うけど、Y で合ってる？") |
| エラーが出た | 必ず**自分で先に読む**。`file:line:col message` の構造から意味を解読 |
| その日に終わらない | 翌日に持ち越さず、`scrap.md` に「明日最初に潰す」と書く |

## 読み方の作法 5要素 (毎日意識)

1. **比較読み** — 読む前に「自分ならどう書くか」を書く (Loop 0 で実施)
2. **逆設計問い** — 各 notes に「他にもありえた選択」を1個入れる
3. **破壊テスト** — Day 2 / Day 3 に1回ずつわざと壊す
4. **教えるつもり読み** — 後輩に説明する想定で読む
5. **デイリーレビュー** — 1日の終わりに 3+3+3

## 完了基準 (Level 2 — Definition of Done)

- [ ] 9 ファイルに `notes/XX.md` (各 6 項目)
- [ ] ETL 対応図 1枚
- [ ] 主要設計判断 5つ × 各 200〜400字
- [ ] Zenn 記事 1本 (3000字 + コードブロック10個以上)
- [ ] スライド 10〜15枚

---

## 今すぐやること

**Day 0 開始**: `learning/notes/00_my_hypothesis.md` を新規作成して、コードを見ずに 30分で 5項目を書く。完了したら Claude に見せて、Loop 1 へ進む。
