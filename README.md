# lunch-bot

Slack で 🍱 を押した人をランダムにランチグループへ分ける bot。

毎週月曜の朝に募集投稿を出し、火曜の朝に集計→グループ発表→水曜のランチで会う、という運用ループを GitHub Actions cron で完全自動化。

詳細仕様は [spec.md](./spec.md)、開発過程の学習ログは [scrap.md](./scrap.md)。

---

## 現状 (2026-04-28)

✅ Phase 2 完了。GitHub Actions cron で自動運用、E2E動作確認済み。

| 項目 | 状態 |
|---|---|
| 実装 | recruit / announce のサブコマンド方式 (`cmd/bot`) |
| 自動運用 | schedule (cron) + workflow_dispatch (手動) |
| 環境分離 | dev / prod 両方 Secrets 登録済み |
| デフォルト実行先 | **dev** (本番切替は yml 1行変更で可能) |
| シミュレータ | `cmd/simulate` で任意人数の擬似実行が可能 |
| テストカバレッジ | shuffler は単体テスト、E2E は workflow_dispatch で実走確認済み |

---

## 使い方

### A. 自動運用 (放置でOK)

何もしなくても以下が動く:

| 曜日・時刻 | 動作 |
|---|---|
| 月曜 09:00 JST | recruit が走る → 募集投稿 + bot が 🍱 を押す |
| 火曜 09:00 JST | announce が走る → 集計→グループ発表 (or 「お休み」) |

(GitHub Actions のスケジュールは数分遅延することがある)

### B. 手動で動かしたい時

3つの入口がある。用途で使い分け。

#### B-1. GitHub Web UI (一番手軽)

https://github.com/Imutaakihiro/lunch-bot/actions/workflows/lunch-bot.yml
→ 右上「Run workflow」ボタン → `mode` (recruit/announce) と `env` (dev/prod) を選んで Run。

#### B-2. ターミナルから `gh` コマンド

```bash
# 起動
gh workflow run lunch-bot.yml -f mode=recruit  -f env=dev
gh workflow run lunch-bot.yml -f mode=announce -f env=dev
gh workflow run lunch-bot.yml -f mode=recruit  -f env=prod   # 本番

# 結果確認
gh run list --repo Imutaakihiro/lunch-bot --limit 5
gh run view <RUN_ID> --repo Imutaakihiro/lunch-bot --log
```

#### B-3. ローカル直接実行 (デバッグ最速)

```bash
cd ~/repos/androots/lunch-bot
source .env
go run ./cmd/bot recruit
go run ./cmd/bot announce
```

`.env` (gitignore済) に `SLACK_BOT_TOKEN` と `SLACK_CHANNEL_ID` を書いておく。

### C. シミュレータ (実Slackを使わずロジック検証)

任意の参加者数で announce フローをローカル実行できる。

```bash
go run ./cmd/simulate -n 8                                  # 8人参加 (偽ID、コンソール出力のみ)
go run ./cmd/simulate -n 0                                  # 0人 → 「お休み」パスを確認
go run ./cmd/simulate -n 12 -post                           # 12人 → 結果を実Slackへ投稿 (mention は偽IDなのでテキスト扱い)
go run ./cmd/simulate -users U0AAA,U0BBB,U0CCC -post        # 実IDを混ぜると mention が解決される
```

`-post` を付けない時はSlackには触らないので、ロジック確認だけしたい時に便利。

---

## ディレクトリ構成

```
.
├── cmd/
│   ├── bot/main.go               # 本体エントリ (recruit/announce サブコマンド)
│   └── simulate/main.go          # ローカル擬似実行ツール
├── internal/
│   ├── config/config.go          # 環境変数読み込み
│   ├── handler/lunch_handler.go  # cmd → service の入口
│   ├── repository/slack_client.go # Slack API ラッパー (5メソッド)
│   └── service/
│       ├── lunch_service.go      # 業務ロジック (RunRecruit, RunAnnounce)
│       └── shuffler.go           # グループ分けアルゴリズム + テスト
├── .github/workflows/lunch-bot.yml # cron 2本 + workflow_dispatch
├── main.go                       # Phase 1 接続テスト用 (記念碑として保存)
├── spec.md                       # 詳細仕様
└── scrap.md                      # 学習ログ
```

---

## 環境変数 / Secrets

| 名前 | 用途 |
|---|---|
| `SLACK_BOT_TOKEN` | Slack Bot User OAuth Token (`xoxb-...`) |
| `SLACK_CHANNEL_ID` | 投稿先チャンネル ID (`Cxxxxxxxx`) |

### GitHub Actions

Repository Secrets ではなく **Environment Secrets** で dev/prod 別に管理:

- Settings → Environments → `dev` / `prod`
- yml の `environment: ${{ inputs.env || 'dev' }}` で切替

確認:
```bash
gh secret list --env dev  --repo Imutaakihiro/lunch-bot
gh secret list --env prod --repo Imutaakihiro/lunch-bot
```

### ローカル実行用

`.env` ファイル (gitignore済):

```bash
export SLACK_BOT_TOKEN=xoxb-...
export SLACK_CHANNEL_ID=Cxxxxxxxx
```

---

## 自動運用を本番 (prod) に切替えたい時

1. prod env の secrets が登録済みか確認 (`gh secret list --env prod`)
2. `.github/workflows/lunch-bot.yml` の以下を編集:
   ```yaml
   environment: ${{ inputs.env || 'dev' }}
   #                              ^^^^^ ← 'prod' に変更
   ```
3. `git commit && git push`
4. 翌週月曜から本番チャンネルで自動運用開始

dev は workflow_dispatch から手動トリガー用に温存される。

---

## 開発に必要なもの

- Go 1.26.2+
- `gh` CLI (GitHub Actions 操作用)
- Slack App + Bot Token + 招待済みチャンネル

セットアップ手順は spec.md §5 を参照。

---

## 設計の背骨

**自分のプロセスが何を持つか、減らせば減らすほどシステムは壊れにくくなる**。

- 時間 → GitHub Actions cron に外出し
- 状態 → Slack を Source of Truth として扱う
- 識別 → 投稿テキストの先頭文字列で判定
- 認可 → Slack OAuth scope に分離
- 失敗通知 → GitHub Actions の標準メール通知

Goコードは「呼ばれたら一発で仕事して終わる」だけの軽い存在に保つ。
