# lunch-bot — Slack ランチシャッフル bot

毎週火曜にランチ希望者を募り、🍱 を押した人を4人ずつのグループに分ける Slack bot を作りながら Go を学ぶ。

進め方は **「vibe-code → 分解」** の4フェーズ:
1. Phase 0: 仕様を一緒に書く（spec.md）
2. Phase 1: vibe-code で動くMVP
3. Phase 2: 1ファイルずつ分解して理解
4. Phase 3: 部分的に書き直して "自分で書ける" に

## 2026-04-20 Slack アプリのセットアップ完了

達成したこと:
- テスト用ワークスペース `go-practice` を新規作成
- Slackアプリ「26卒シャッフルランチ」を作成
- Socket Mode を有効化 → App-Level Token (`xapp-...`) 取得
- Bot Token Scopes を3つ設定: `chat:write` / `reactions:read` / `users:read`
- ワークスペースにインストール → Bot User OAuth Token (`xoxb-...`) 取得
- bot をテストチャンネルに招待

学んだこと:
- Slackアプリは「権限（scope）」と「存在（bot user）」と「動作（events）」が**別画面で設定**される（責務分離）
- トークンは2種類: `xapp-`（Socket Mode用）と `xoxb-`（API呼び出し用）
- scope = ボットができることの宣言。**最小権限の原則** で必要なものだけ要求するのがプロ作法
- bot は招待されたチャンネルでしか発言できない（権限とアクセス先は別の話）

## 2026-04-20 接続テスト: Goから "Hello" を投稿

`main.go` で 50行未満の最小コードを書いて Slack 投稿成功を確認する。
このテストの目的: 後で200行のbotを書く前に、**認証パイプが通ることを保証** する。

### コードでやってること（ざっくり）

1. **環境変数** から token と channel ID を読む（`os.Getenv`）
2. **JSON ペイロード** を作る（`json.Marshal` + `map`）
3. **HTTP POST リクエスト** を Slack API に送る（`net/http` 標準ライブラリ）
4. **レスポンスを表示** して成否を確認

### 実行コマンド

```bash
cd learning/lunch-bot
cp .env.example .env
# .env を開いて SLACK_BOT_TOKEN を貼る
source .env
go run .
```

### 成功時のレスポンス例

```json
{"ok":true, "channel":"C...", "ts":"...", "message":{...}}
```

`"ok":true` ならOK。Slack のチャンネルにも `Hello from Go! 👋` が投稿されているはず。

### 失敗時のレスポンス例と対処

| `error` の値 | 意味 | 対処 |
|---|---|---|
| `invalid_auth` | トークンが無効 | `.env` の `xoxb-` トークンを確認 |
| `not_in_channel` | bot がチャンネルに居ない | Slackで `/invite @bot名` |
| `channel_not_found` | チャンネルIDが間違い | `.env` の `SLACK_CHANNEL_ID` を確認 |
| `missing_scope` | scope が足りない | アプリ設定で必要な scope を追加 → 再インストール |

## 2026-04-20 11:00 接続テスト成功 — 5連続エラーの旅

`go run .` 一発で動かず、**5種類のエラー**を順に潰してようやく成功。それぞれが別レイヤーの学びだったので記録する。

### エラー履歴

#### ① `command not found: socket-token=xapp-...`
**原因:** `.env` に `socket-token=xapp-...` と書いた。zshの変数名は **英数字とアンダースコアのみ** で、`socket-token` のハイフンは無効。さらに `export` も付いてないので、zsh は「コマンド実行」と解釈。
**学び:** シェル変数名は `[A-Za-z_][A-Za-z0-9_]*`。Pythonの変数名と同じ。それと、そもそも今回のテストには `xapp-` (App-Level Token) は不要だった。

#### ② `command not found: SLACK_BOT_TOKEN`
**原因:** `SLACK_BOT_TOKEN = xoxb-...` のように `=` の前後にスペースを入れた。シェルでは `FOO = bar` は「`FOO` というコマンドに `=` と `bar` を渡す」と解釈される。
**学び:** **シェルでは `=` の前後にスペース禁止。** Python (`x = 1`) や Go (`x := 1`) と違うシェル独特のクセ。

#### ③ `.env:export:1: not an identifier: 962696748689-...`
**原因:** `xoxb-` トークンの**途中にスペース**が混入（コピペ事故）。`export` は複数変数を一度に書ける構文 (`export A=1 B=2`) なので、スペースで「2つ目の引数開始」と解釈された。先頭が数字でハイフンを含む `962696748689-...` は変数名として無効。
**学び:** トークンをコピペする時は**コピーボタン**を使う（手動選択は欠損リスクあり）。値にスペースを含めたい時は `export FOO="..."` でクオート。

#### ④ `{"ok":false,"error":"not_allowed_token_type"}`
**原因:** `SLACK_BOT_TOKEN` に `xapp-` で始まるトークンを入れていた。`chat.postMessage` は `xoxb-` トークン専用。
**学び:** Slackのトークンは用途別に分かれている。
- `xoxb-` = HTTP API用（メッセージ送信など）
- `xapp-` = Socket Mode (WebSocket) 専用
- `xoxp-` = User token（ユーザー権限で叩く）

#### ⑤ `{"ok":false,"error":"invalid_auth"}`
**原因:** トークン漏えい対策で Revoke した古いトークンをまだ `.env` に書いていた。Reinstall して新トークンを発行したが、貼り直しを忘れていた。
**学び:** Slackの `invalid_auth` は**形式は合っている、値が無効**。`xoxb-` で始まり長さも十分なのに認証通らないなら、ほぼ「Revoke済み or Reinstallで変わった」を疑う。

#### ⑥ 成功！

```json
{"ok":true,"channel":"C0AU1BXKPB6","ts":"1776650717.694419","message":{...}}
```

### 振り返り — このデバッグから得た一般則

- **エラーは「どのレイヤーか」を最初に切り分ける。** ①②③ はシェル層、④⑤ はAPI層。`go run` まで届く前に死んでいるなら Go コードを疑っても無駄。
- **エラーメッセージを「コマンド」と「文脈」に分けて読む。** `command not found: SLACK_BOT_TOKEN` の本質は「`SLACK_BOT_TOKEN` をコマンドとして実行しようとした」=「シェルが代入と解釈してくれなかった」。表面の文字列ではなく動作仮説に翻訳する練習。
- **同じ症状でも原因が変わる**。①②③ は全部「シェル設定エラー」で症状は似てるけど、原因は別。一個ずつ潰すしかない。
- **「進歩しているか」を判断材料に**: ④→⑤ で error メッセージが変わったのは前進のサイン。同じエラーが続くなら違うアプローチが必要。

### Slackレスポンスから学んだJSON構造

成功レスポンスを観察してわかったこと:
- `ts` (タイムスタンプ) = メッセージID。後でリアクション検知する時に **チャンネル + ts** でメッセージを特定する
- `\u30b7\u30e3...` = 日本語のUnicodeエスケープ。Goの `json.Marshal` も同じ挙動（読む側で自動デコード）
- ネスト深い: `message.bot_profile.icons.image_36` のように4階層。Goで扱うなら struct を入れ子にするか `map[string]any` で受ける
- `bot_id` (B...) と `app_id` (A...) は別物。1 App に 1 Bot が紐づく
