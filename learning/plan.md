# lunch-bot 読解 学習計画

## 主目標 (Level 2 — Definition of Done)

- [ ] 9 ファイル全部に `notes/XX_filename.md` を完成 (各 6 項目 — 後述「notes の書き方」参照)
- [ ] ETL対応図を1枚描く (Extract / Transform / Load にコードのどこが対応するか)
- [ ] 主要設計判断 5つ を各 200〜400字で説明できる文章を残す
- [ ] Zenn記事 1本 (3000字以上、コードブロック10個以上)
- [ ] スライド 10〜15枚 (記事から派生)
- [ ] **読み方の作法 5要素** を毎日実践 (後述)

## 読み方の作法 — 一流エンジニアの5要素

「動くコードを動くまま使う」を超えて「**動くコードを解体して理解する**」へ。
新卒1年目で最大の差がつくのが、ここの習慣。

### 1. 比較読み (Comparison Read)
読む前に **「自分なら同じ仕様でどう書くか」を 30分で書く**。Day 1 開始前に Loop 0 として実施し、`notes/00_my_hypothesis.md` を作る。読みながら "自分の仮説 vs 実物" の差分を意識する → **差分こそが学び**。

### 2. 逆設計問い (Counter-design)
**「なぜこう書かなかったか」を1個ずつ問う**。各 notes に「他にもありえた選択」セクションを設けて、最低1つの代替案 + "選ばなかった理由の自分なりの推測" を書く。

### 3. 破壊テスト (Break Test)
**1日1回、わざと壊してみる**。1行コメントアウト・引数変更・空入力など。Day 2〜4 で各1回、実験ログを `scrap.md` に残す。Level 3 の入口を Level 2 に少し混ぜる。

### 4. 教えるつもり読み (Teach-as-you-read)
**読みながら口頭で「後輩に説明するなら」と再構成**。詰まる = 理解の穴。穴を notes の「自分が引っかかった所」に書き出す。

### 5. デイリーレビュー (Meta-reflection)
**1日の終わりに 3+3+3** を `scrap.md` に書く:
- 今日学んだこと Best 3
- 次に潰したい課題 Top 3
- まだ "なんで？" な疑問 3つ

「自分が何を知って、何を知らないか」を毎日整理する習慣。これが新卒3ヶ月で土台を作る最強のクセ。

## ペース

- **目標: 4日完了** / **上限: 5日 (Day 5 バッファ)**
- 1日 6 時間目安
- 各日の最後に **30分のデイリーレビュー枠** を必ず確保

## ループ計画

### Day 0 (Day 1 冒頭 30分): 比較読み

- [x] **Loop 0** (30分) — `spec.md` と `README.md` だけ読んで、コードを見ずに `notes/00_my_hypothesis.md` を書く
  - 内容: 自分なら lunch-bot をどう実装するか (ファイル構成・関数の引数戻り値・想定エラー・データの流れ)
  - **これは絶対にコードを見る前に書く** — 後で見比べた時の "自分の仮説" は、書いた瞬間しか正直に残せない

### Day 1 (5.5h): 全体地図 + 入口

- [x] **Loop 1** (1.5h) — `main.go` (Phase 1) + `cmd/bot/main.go`
  - 学び: package, import, os.Args, log.Fatal, DI 配線
  - 出力: `notes/01_main.md` + `notes/02_cmd_bot.md` (Counter-design 含む)
  - ※ `01_main.md` の整理リライトと `02_cmd_bot.md` の型名追記は持ち越し (Loop 2-6 で型定義を読んだ後に戻る)
- [x] **Loop 2** (1.5h) — `internal/config/config.go` + `internal/handler/lunch_handler.go`
  - 学び: struct, ポインタレシーバ, error 返値, 関数 vs メソッド, コンストラクタパターン (`NewXxx`), error 型は "通知書", 「そのまま外に返す」の意味
  - 出力: `notes/03_config.md` + `notes/04_handler.md`
- [ ] **Loop 3** (2h) — `internal/service/lunch_service.go` 前半 (interface + RunRecruit)
  - 学び: **interface**, 依存性注入 (DI), `fmt.Errorf %w`
  - 出力: `notes/05_service.md` 前半
- [ ] **Day 1 デイリーレビュー** (30分) — `scrap.md` に 3+3+3

### Day 2 (6h): 業務ロジック深掘り + 初の破壊テスト

- [ ] **Loop 4** (2h) — `lunch_service.go` 後半 (RunAnnounce)
  - 学び: 冪等性, prefix判定, bot除外, 早期return
  - 出力: `notes/05_service.md` 完成
- [ ] **Loop 5** (2h) — `internal/service/shuffler.go` + `shuffler_test.go`
  - 学び: math/rand, スライス分割, テーブル駆動テスト
  - 出力: `notes/06_shuffler.md`
- [ ] **Loop 6 前半** (1h) — `slack_client.go` 前半 (PostMessage, AddReaction)
- [ ] **Day 2 破壊テスト** (30分) — 例: bot 除外の `excludeUser` を消したらどうなる？シミュレータで検証。`scrap.md` に観察ログ
- [ ] **Day 2 デイリーレビュー** (30分)

### Day 3 (6h): I/O + シミュレータ + 振り返り

- [ ] **Loop 6 後半** (2h) — `slack_client.go` 後半 (auth.test, conversations.history, helpers)
- [ ] **Loop 7** (2h) — `cmd/simulate/main.go`
  - 学び: fakeSlack, **interface の威力体感**, テストダブル
  - 出力: `notes/08_simulate.md`
- [ ] **Loop 8** (1h) — `notes/00_overview.md` (ETL対応図 + 設計判断5つ)
- [ ] **Day 3 破壊テスト** (30分) — 例: 冪等性チェックを消したら？シミュレータで2回連続実行
- [ ] **Day 3 デイリーレビュー** (30分)

### Day 4 (6h): プレゼン化

- [ ] Zenn記事執筆 (3.5h) — 8 ループの notes を総括して1本にまとめる
- [ ] スライド派生 (2h) — 記事から 10〜15枚に圧縮
- [ ] **Day 4 デイリーレビュー** (30分) — 4日全体を振り返る

### Day 5: バッファ / リハ / Level 3 つまみ食い

- [ ] 必要に応じて読解の積み残し対応
- [ ] スライドのリハーサル (実際に声に出して時間を計る)
- [ ] (余裕があれば) Level 3 から1つ追加実験

## 4ステップ学習法 (各ループに適用)

global CLAUDE.md より。各ステップに上の「読み方の作法」を忍ばせる。

1. **読む** (~30分): 該当ファイルを1行ずつ読む + Teach-as-you-read で口頭再構成
2. **書く** (~30分): 真似てちょっと書いてみる、または break して挙動を見る (Break Test の機会)
3. **読む (実コード)** (~15分): 関連 Go ドキュメント、または別の例 — Counter-design の問いに材料を集める
4. **まとめる** (~15分): `notes/XX.md` を更新 (Counter-design 必須) + `scrap.md` に時系列メモ

## 主要設計判断 5つ (Day 3 完了時に各 200〜400字で説明できるようにする)

1. **なぜ `SlackRepository` を interface として切ったか** (テスト容易性 + 入口の柔軟性、`cmd/simulate` で本物を fake に差し替えられる仕組み)
2. **なぜ recruit / announce をサブコマンドに分けたか** (時間管理を GitHub Actions に外出しするため、Goコードは1発実行に)
3. **なぜ「お休み」も冪等性チェック対象にしたか** (E2E発見バグの修正、"完了シグナル" は1種類じゃない)
4. **なぜテキスト先頭で識別する設計にしたか** (ステートレス維持、Slack を Source of Truth に)
5. **なぜ `cmd/simulate` を作ったか** (実Slackに人を集めずに任意人数のロジック検証ができる、interface があるから可能)

## notes/ の書き方ルール (6 項目)

各 `notes/XX_filename.md` には最低 6 項目:

1. **このファイルが何をするか** (1行サマリ)
2. **主な型** (struct, interface)
3. **主な関数** (引数・戻り値・1行説明)
4. **自分が引っかかった所** (具体的に、後で見返した時の記憶のフック)
5. **他にもありえた選択 (Counter-design)** — 最低1つ。「なぜこう書かれてないか」の自分なりの推測
6. **次に読む人へのアドバイス** (= 未来の自分へ)

行数目安は 30〜50行。それ以上になりそうなら詳細は `scrap.md` に逃がす。

## Zenn記事の構成案 (Day 4 用ドラフト)

1. **きっかけ**: AIに作ってもらったコードを自力で読むことにした背景 (新卒・未経験・ETL業務予定)
2. **lunch-bot は何をする bot か** (1図 + 動作デモのスクショ)
3. **ETL 構造としての俯瞰** (Extract / Transform / Load 対応図)
4. **コードツアー**: 9 ファイルを順番に最小コードブロックで紹介
5. **5つの設計判断と、それぞれが何を解いているか**
6. **Phase 2 で発見した2つのバグ**: emoji正規化 / 冪等性穴
7. **シミュレータが本物の Slack 無しでロジック検証できる仕組み** (interface と fake)
8. **学んだこと / 次にやること**

## 進捗の記録方法

- このファイル (plan.md) のチェックボックスをチェックしていく
- 1日の終わりに `scrap.md` に **デイリーレビュー (3+3+3)** を必ず書く
- 引っかかった所は逐次 `scrap.md` に時系列で記録
