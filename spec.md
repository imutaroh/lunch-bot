# Slack ランチシャッフル bot — 要件定義

## 1. 概要

毎週 **月曜09:00 (JST)** に Slack チャンネルへ「ランチ参加者募集」の投稿が自動で流れる。
🍱 リアクションを押した人を **火曜09:00 (JST)** にランダムで 3〜5 人組のグループに分けて、同じチャンネルに発表する bot。発表されたグループで **水曜にランチ** を実施。

ひとことで言うと: **Slack で🍱を押した人をランダムにランチグループへ分ける bot**。

> **Phase 2 設計判断**: Phase 1 は `go run ./cmd/bot` で起動する **ターミナル常駐方式** だったが、「翌日まで自分のマシンを起こし続ける」運用負担が大きいため、**GitHub Actions cron で1発実行する方式** に切り替える。Goコードは時間管理を持たず、cron からサブコマンド (`recruit` / `announce`) で1回だけ呼ばれて即終了する。状態は **Slack 側に置く** ことで、プロセス間の引き継ぎも不要にする（ステートレス化）。

## 2. ユーザーストーリー

| 役割 | 行動 | ゴール |
|---|---|---|
| 運営（=ご主人様） | 一度 GitHub Actions に cron を仕込んだら、あとは放置 | 毎週手を動かさずにランチ会を回したい |
| 参加希望メンバー | 月曜の募集投稿に 🍱 を押す | 1クリックで参加表明したい |
| 全員 | 火曜09:00 にグループ発表を見る | 水曜のランチで誰と行くか把握したい |

## 3. 機能要件

### 3.1 起動コマンド（GitHub Actions から呼ばれる）

サブコマンド方式で2つのモードを持つ:

```bash
go run ./cmd/bot recruit    # 月曜09:00 に呼ばれる: 募集投稿 + 自分で🍱を1個押す
go run ./cmd/bot announce   # 火曜09:00 に呼ばれる: リアクション集計 → グループ発表
```

- `os.Args[1]` を見て分岐（`go test` / `go build` と同じスタイル）
- 不正な引数は `usage` を表示してエラー終了
- どちらのモードも **1ジョブで完結** し、即プロセスを終了する（待機なし）

### 3.2 募集投稿（recruit モード）

- 投稿先: 設定済みチャンネル (`SLACK_CHANNEL_ID`)
- 投稿テキスト:
  ```
  🍽️ 今週のランチ参加者募集！
  参加したい人は :bento: を押してね
  締切: 火曜09:00 / 水曜のランチで 3〜5人組にランダム振り分けます
  ```
- **bot 自身が🍱を1個押す**（ファーストペンギン問題の解消: 誰も押してない投稿は心理的に押しにくい）
  - `chat.postMessage` 成功直後に `reactions.add` を呼ぶ
  - リアクション付与に失敗した場合は **エラー終了**（投稿のみ成功・スタンプなしの中途半端状態を残さない）

### 3.3 リアクション集計（announce モード）

- 集計対象: **直近23時間以内** の bot 自身の「ランチ募集投稿」に押された 🍱 (`bento`)
- **募集投稿の識別**: テキスト先頭が `🍽️ 今週のランチ参加者募集！` で始まる、bot 自身の投稿
  - `conversations.history` で取得
  - **テキストを変えると識別が壊れる**（疑似コントラクト）ことを実装上のコメントに残す
- **bot 自身のリアクション除外**:
  - `auth.test` で自分の bot user ID を取得し、リアクションのユーザーリストから除外
  - **これは仕様（bot が自分で1個押す）の前提条件**。除外を忘れると参加者0人でも1人カウントの不具合になる
- **募集投稿が見つからない場合**: エラー終了して GitHub Actions を失敗（赤）にする
  - メール通知で運営が異常に気付ける
  - 「黙ってスキップ」は失敗を隠してしまうので採用しない

### 3.4 グループ分け（変更なし）

- 基本: 4人組
- 4人組を基準に 3〜5 人で割り切れるよう調整（実装は `decideGroupSizes`）

| 人数 | 例 |
|---|---|
| 3 | [3] |
| 7 | [4, 3] |
| 9 | [5, 4] |
| 11 | [4, 4, 3] |
| 12 | [4, 4, 4] |

メンバー選定はランダムシャッフル（過去ペアは記録しない）。

### 3.5 発表投稿

```
🎉 今週のランチグループ決定！

グループA: <@U012> <@U345> <@U678> <@U901>
グループB: <@U234> <@U567> <@U890>

水曜のランチで楽しんで！🍱
```

メンションは `<@Uxxxx>` 形式で本人通知が飛ぶ。

### 3.6 少人数時のフォールバック

参加者 0〜2 人（bot自身を除いた数）の場合:

```
😿 今週は参加者が少なかったのでお休みです
また来週叩いてください
```

### 3.7 announce の冪等性（重複発表の防止）

手動実行ミス等で `announce` が同日に複数回起動されるケースに備え:

- 直近の bot 投稿を見て、**最新のメッセージが "今週分の announce 完了シグナル" ならスキップ**（exit 0）
- 完了シグナルは2種類:
  - 「グループ発表」: テキスト先頭が `🎉 今週のランチグループ決定！`
  - 「お休み」: テキスト先頭が `😿 今週は参加者が少なかった`（参加者 < 3 で発表せず終わった場合も "完了" とみなす）

これにより、人間の操作ミスを実害なく吸収できる。E2Eテストで「お休み」側を見落としていたバグが見つかった (2026-04-28)。

## 4. 非機能要件

| 項目 | 要件 |
|---|---|
| 実行環境 | **GitHub Actions cron**（schedule + workflow_dispatch） |
| 実行時刻 | 月曜09:00 JST 募集 / 火曜09:00 JST 発表 |
| タイムゾーン | Asia/Tokyo (JST) — Goコード内では時刻を扱わない（GHAが管理） |
| 言語 | Go 1.26 |
| アーキテクチャ | 3層レイヤード (handler / service / repository) |
| 永続化 | **なし（ステートレス）**。状態は Slack に置く（Source of Truth） |
| 設定 | GitHub Secrets（本番） + ローカルの `.env`（開発） |
| 失敗通知 | GitHub Actions の標準メール通知に任せる |

## 5. Slack App 設定

### 5.1 必要な OAuth Scope (Bot Token Scopes)

| Scope | 用途 | 状態 |
|---|---|---|
| `chat:write` | 募集・発表の投稿 | ✅ 設定済 |
| `reactions:read` | リアクション取得 | ✅ 設定済 |
| `users:read` | ユーザー情報取得（メンション用） | ✅ 設定済 |
| `conversations:history` | bot自身の過去投稿を探す | **追加必要** |
| `reactions:write` | bot が🍱を自分で押す | **追加必要** |

**追加後は Reinstall して新トークンを GitHub Secrets と `.env` に貼り直す**。

### 5.2 Socket Mode

- **不要**（GitHub Actions から HTTP API を直接叩く）

### 5.3 追加設定

- **不要**（既存の bot 招待済みチャンネルでそのまま動作）

## 6. 環境変数

```bash
# .env (gitignore 済 — ローカル開発用)
export SLACK_BOT_TOKEN=xoxb-...        # チャット投稿・リアクション用
export SLACK_CHANNEL_ID=C0AU1BXKPB6    # 動作する固定チャンネル
```

GitHub Actions では同じ2変数を **Repository Secrets** に登録し、workflow から `env:` 経由で渡す。

`.env` は `.gitignore` に登録済み（誤コミット事故防止）。

## 7. ディレクトリ構成

```
learning/lunch-bot/
├── .github/
│   └── workflows/
│       └── lunch-bot.yml            # cron 2本 + workflow_dispatch
├── cmd/
│   └── bot/
│       └── main.go                  # サブコマンド分岐 (recruit/announce)
├── internal/
│   ├── config/
│   │   └── config.go                # 環境変数読み込み
│   ├── handler/
│   │   └── lunch_handler.go         # サブコマンドごとに service を呼ぶ
│   ├── service/
│   │   ├── lunch_service.go         # recruit / announce のオーケストレーション
│   │   └── shuffler.go              # グループ分けアルゴリズム
│   └── repository/
│       └── slack_client.go          # Slack API ラッパー（PostMessage / AddReaction / FindLatestBotMessage / GetReactionUsers / WhoAmI）
├── main.go                          # Phase 1 接続テスト用（学習資産として残す）
├── spec.md                          # この要件定義
└── scrap.md                         # 学習ログ（main.go へのリンク含む）
```

**Phase 1 から削除されるもの**:
- `internal/service/scheduler.go` — 時間管理を GHA に外出ししたため不要
- `RunSession` の `time.Sleep` 部分

## 8. 依存ライブラリ方針

- **標準ライブラリのみ** (`net/http`, `encoding/json`, `os`, `fmt` など)
- サードパーティライブラリは **使わない**（Phase 1 と同じ）

## 9. スコープ外（やらないこと）

- ❌ 過去ペア履歴の記録/回避
- ❌ 参加者の事前候補抽出（誰でも自由参加）
- ❌ ランチ場所の提案
- ❌ カレンダー連携
- ❌ 複数チャンネル対応
- ❌ Web UI
- ❌ DM 通知
- ❌ 締切の手動制御コマンド（`/lunch close` 等）
- ❌ Slack スラッシュコマンド対応
- ❌ Socket Mode

## 10. 実装フェーズ

### Phase 1: ターミナル起動 MVP — ✅ 完了 (2026-04-22)
- 3層レイヤードで vibe-code 実装
- ローカル `go run ./cmd/bot` で動作
- shuffler のテスト実装

### Phase 2: GitHub Actions cron 化 — 🚧 進行中
- `cmd/bot/main.go` を recruit / announce のサブコマンド方式に分割
- `internal/service/scheduler.go` を削除
- Slack repository に追加: `AddReaction`, `FindLatestBotMessage`, `WhoAmI`
- bot自身のリアクション除外を実装
- announce の冪等性チェック実装
- Slack App に scope 2つ追加 + Reinstall
- `.github/workflows/lunch-bot.yml` 新設（schedule + workflow_dispatch）
- GitHub Secrets 登録

### Phase 3: 自分で書き直し — 未着手
- 部分的に消して書き直し
- エラーハンドリングの理解

## 11. 設計の背骨（このプロジェクトを貫く哲学）

**「自分のプロセスが何を持つか」を減らせば減らすほど、システムは壊れにくくなる**:

| 何を | どこに委ねたか |
|---|---|
| 時間 | GitHub Actions cron |
| 状態 | Slack（Source of Truth） |
| 識別 | 投稿テキストの先頭文字列 |
| 認可 | Slack OAuth scope |
| 失敗通知 | GitHub Actions の標準機能 |

Goコード自体は「呼ばれたら一発で仕事して終わる」だけの軽い存在になる。これは Phase 4（Cloud Run 等への移行）でも同じ思想で進む。
