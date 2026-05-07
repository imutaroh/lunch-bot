# notes/07_slack_client.md — `internal/repository/slack_client.go`

Loop 6。`repository` 層の本体 = **Slack API を HTTP で叩く実装**。

このループの方針 (Day 2 セッションでご主人様判断):
- **ブラックボックス読み** で進める。HTTP の詳細は「Webを支える技術」(山本陽平 著) で別途学習予定
- ここでは **struct構造 + 5メソッドの入出力 + interface との対応** に集中

学び目標 (plan.md より、ブラックボックス版に修正):
- ~~auth.test, conversations.history, helpers~~ → **詳細はパス**
- **struct メソッド化のメリット** (Phase 1 ベタ書きとの違い)
- **interface との完全対応** (SlackRepository の5メソッドを全部実装)
- **helper関数 (doJSON / doGet) による DRY**

---

## ◆ 比較読み (Comparison Read) — コードを開く前に書く

### Q1. SlackClient struct はどんなフィールドを持っていそう?

(チャットで明示的に答えてないので空欄。実物 = `token` + `httpClient` の2つ)

### Q2. Phase 1 (`main.go`) は HTTP 操作をベタ書きしてた。なぜ struct メソッドに整理し直したと思う?

> ご主人様の応答: 「HTTP操作がよく分かってないんだよね😢」
>
> → 答えとしては引き出せなかったが、Day 1 Loop 2 でやった "コンストラクタパターン" + "状態保持" の発想に繋がる:
> 1. `cfg.SlackBotToken` を毎回引数で渡したくない (= 一度だけ受け取って struct に保持)
> 2. `httpClient` も使い回したい
> 3. handler / service / repository の3層分担で、repository は "HTTP 担当" に役割を集約

### Q3. fakeSlack との対比で interface の威力が見える?

(これは Day 2 復習で interface を集中的にやったあとの "本物の例" として扱う)

---

## ◆ 実物を読んだあとに埋める (notes 6 項目)

### ◆ 抽象まとめ (まずココで全体像を掴む)

**1文要約**:
> slack_client.go = `SlackRepository` interface を **HTTP で実装した本物**。tokenとhttpClientを保持し、Slack API に POST/GET を叩いて結果を返す。helper関数で共通処理を切り出して DRY。

**抽象構造 (4要素)**:

```
┌────────────────────────────────────────────────────┐
│ ① 状態保持   SlackClient (struct)                    │
│              token + httpClient                       │
├────────────────────────────────────────────────────┤
│ ② 5メソッド  PostMessage / AddReaction /              │
│              GetReactionUsers / WhoAmI /              │
│              RecentBotMessages                         │
│              ↑ SlackRepository interface と完全一致   │
├────────────────────────────────────────────────────┤
│ ③ helper     doJSON / doGet                           │
│              POST系/GET系の共通処理 (重複排除)        │
├────────────────────────────────────────────────────┤
│ ④ レスポンス struct 群                                │
│              postMessageResponse, etc                  │
│              JSON を Go struct にマッピングする型      │
└────────────────────────────────────────────────────┘
```

---

### 1. このファイルが何をするか (1行サマリ)

`SlackRepository` interface を **HTTP で実装した本物**。Slack の Web API (chat.postMessage, reactions.add, conversations.history, auth.test など) を叩いて、結果を Go の値に変換して返す。

→ "外部システムとの通信" を全部この層に閉じ込めることで、上位層 (service) は HTTP を意識せず業務ロジックに集中できる。

---

### 2. 主な型 (struct, interface)

#### `SlackClient` (struct) — 15-18行目

```go
type SlackClient struct {
    token      string         // Slack Bot Token (認証用、xoxb-... の形式)
    httpClient *http.Client   // 標準ライブラリの HTTP クライアント
}
```

→ Day 1 Loop 2 の `LunchHandler` と同じ **コンストラクタパターン**。
→ token を一度だけ受け取って保持 = メソッド呼び出し時に毎回 token を渡さなくて済む。

#### レスポンス struct 群 (各メソッドのすぐ上に定義)

```go
type postMessageResponse struct {
    OK    bool   `json:"ok"`
    Error string `json:"error,omitempty"`
    TS    string `json:"ts"`
}
```

→ Slack API が返す JSON を Go の値に変換するための **マッピング定義**。
→ `json:"ok"` のタグで「JSONのキー名」と「Goフィールド名」を対応付ける。
→ 詳細パースは `json.Unmarshal(respBody, &result)` で自動。

---

### 3. 主な関数 / メソッド (引数・戻り値・1行説明)

| 名前 | 入力 | 出力 | 役割 |
|---|---|---|---|
| `NewSlackClient` | `token string` | `*SlackClient` | コンストラクタ。httpClientは標準デフォルト |
| `(c) PostMessage` | `channel, text string` | `ts string, err` | Slackに投稿する |
| `(c) AddReaction` | `channel, ts, emoji string` | `err` | 既存メッセージにスタンプを付ける |
| `(c) GetReactionUsers` | `channel, ts, emoji string` | `[]string, err` | 特定スタンプを押したユーザー一覧 |
| `(c) WhoAmI` | (なし) | `botID string, err` | bot自身のユーザーIDを取る (= auth.test) |
| `(c) RecentBotMessages` | `channel, botID string, sinceHours int` | `[]BotMessage, err` | 直近N時間のbot投稿を取る |
| `(c) doJSON` (helper) | `method, url string, body []byte` | `[]byte, err` | POSTリクエストの共通処理 (token付き) |
| `(c) doGet` (helper) | `url string` | `[]byte, err` | GETリクエストの共通処理 (token付き) |

→ ブラックボックス読み: **入出力さえ分かれば service 層からは使える**。

---

### 4. 自分が引っかかった所

#### (a) HTTP の概念がまだ完全には腹落ちしてない (正直な現状)

「リクエスト/レスポンス」「メソッド (POST/GET)」「ヘッダー (Authorization, Content-Type)」「ボディ」「ステータスコード」あたりは Day 1 で名前は知った。**ただ "なぜそういう仕組みなのか" が言葉にできない**。

→ 「Webを支える技術」(山本陽平 著) を別途学習予定。読み終わったら Loop 6 に戻って細部 (`json.Marshal`, `http.NewRequest`, `c.doJSON` の中身) を再読する。

→ それまでは **ブラックボックスとして「外部APIを叩いて値を返す層」** という抽象で扱う。Level 2 (Zenn記事 + スライド) の達成にはこれで十分。

#### (b) "JSONタグ" の仕組み (`json:"ok"`)

```go
type postMessageResponse struct {
    OK    bool   `json:"ok"`
    Error string `json:"error,omitempty"`
}
```

最初「\`json:"ok"\` ってなに?」と思った。これは **構造体タグ (struct tag)** = フィールドにメタ情報を付ける Go の機能。

`encoding/json` パッケージがこのタグを読んで、JSONキー名と Go フィールド名を対応付ける:
- JSON: `{"ok": true, "error": "..."}`
- Go: `result.OK = true`, `result.Error = "..."`

→ **`omitempty`** = JSONを書き出すときに値が空ならフィールドを省略する指示。

#### (c) `*http.Client` (ポインタ) と `http.DefaultClient` の関係

```go
httpClient *http.Client = http.DefaultClient
```

`http.DefaultClient` = 標準ライブラリが提供する **共有のHTTPクライアント**。タイムアウト無し、リダイレクト追従ありのデフォルト設定。

→ 自分で `&http.Client{Timeout: 5*time.Second}` のようにカスタマイズもできる。lunch-bot は今のところデフォルトで十分。

---

### 5. 他にもありえた選択 (Counter-design)

#### 案A: SlackClient struct ではなく関数で書く

```go
func PostMessage(token, channel, text string) (string, error) { ... }
func AddReaction(token, channel, ts, emoji string) error { ... }
```

却下理由:
- token を毎回引数で渡す必要 (冗長)
- httpClient のカスタマイズがしづらい (毎回作り直し or グローバル)
- service との結合点に **interface (SlackRepository)** を挟みたい → 値レシーバが必要 → struct パターンが自然

#### 案B: 5メソッドを別パッケージに分割する (chat / reactions / users / auth)

Slack 公式SDK のような分け方。lunch-bot では却下:
- 機能が小さい (5メソッドだけ)
- 過剰な分割は見通しを悪くする
- 1つの struct で完結する方がインポート/初期化が楽

採用余地: 機能が増えてきたら検討する選択肢。

#### 案C: helper (doJSON / doGet) を作らずに、各メソッドにベタ書き

却下理由:
- 認証ヘッダー、エラーハンドリングが5箇所で重複 → DRY 違反
- 共通処理が変わったとき5箇所を修正 → 保守コスト増

→ Phase 1 (main.go) は1本だけだったから直書きで成立してた。Phase 2 で5本になったから helper 化が必要になった。

#### 案D: HTTPライブラリ (resty, sling, など) を使う

却下理由:
- 標準ライブラリ (`net/http`) で十分書ける
- 外部依存を増やしたくない
- 学習教材として "Go で生 HTTP を書く" 例を残す価値

→ プロダクションでは状況次第 (テスト容易性、リトライ、認証フローが複雑なら検討)。

#### 案E: token を環境変数から直接読む (Configを介さない)

却下理由:
- "外との接続" は config 1箇所に集約しておきたい (= 起動時 fail-fast)
- repository が環境変数を直接読むと、テスト時に環境を弄る必要 → テスト容易性が落ちる
- 「依存は外から注入する (DI)」原則の徹底

→ Day 1 Loop 2 で学んだ Config の役割と一貫した設計。

---

### 6. 次に読む人へのアドバイス (= 未来の自分へ)

1. **HTTP 詳細は別途学べ**: lunch-bot 解読の流れで HTTP を深掘りすると焦点が散る。「Webを支える技術」のような専門書で集中して学んでから戻る方が効率的。

2. **ブラックボックス読みは "戦略的撤退" ではなく "適切な抽象化"**: 上位層 (service) の理解には slack_client の中身は不要。「外部APIを叩いて値を返す層」という抽象でひとまず処理すれば、Level 2 達成に支障なし。

3. **5メソッドのシグネチャだけ覚えとけ**: `PostMessage(channel,text)→(ts,err)` のように。これは interface (SlackRepository) と一致してて、service 層から呼ばれる契約。

4. **interface との対応で "fake が動く" ことの本質が見える**: 本物 SlackClient は HTTP を叩く、fake は何もしない。**同じシグネチャ、全然違う中身**。Day 2 復習で集中的にやった interface の威力の "本物の例"。

5. **helper (doJSON / doGet) は "リファクタの典型例"**: Phase 1 で1本だった HTTP コードが5本になったから共通化された。「重複が3回出たら抽象化を考える」というルールの実例。

6. **JSONタグは "JSON ↔ Go" の翻訳辞書**: `json:"ok"` で JSONキーと Go フィールドを対応付ける。Slack に限らず、外部APIを叩くときに必ず出てくるパターン。

7. **`*http.Client` のポインタ理由は "共有しても再生成コストゼロ" にするため**: 値だとメソッド呼び出しごとにコピーが発生する。HTTPクライアントは内部に接続プールを持つ "重い" オブジェクトなので共有が前提。

8. **将来の HTTP 学習が終わったら戻ってくるべき行**:
   - `json.Marshal` (Go値 → JSON bytes)
   - `json.Unmarshal` (JSON bytes → Go値)
   - `http.NewRequest` (リクエスト構築)
   - `c.httpClient.Do(req)` (送信)
   - `io.ReadAll(resp.Body)` (ボディ読み出し)
   - `resp.StatusCode` 判定
   - `c.doJSON` の中身全部

   → これらが分かれば repository 層は完全に読める。

---

**Loop 6 終了時点のキーセンテンス** (これが言えれば抽象理解OK):

> slack_client.go は **`SlackRepository` interface を HTTP で実装した本物**。token と httpClient を保持し、5メソッド + 2helper で Slack API を叩く。fake と同じシグネチャを持ちつつ中身は HTTP 通信。"interface を介して上位層と疎結合" + "helper で DRY" の典型構造。詳細な HTTP 実装は別途学習予定。
