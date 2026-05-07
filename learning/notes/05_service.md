# notes/05_service.md — `internal/service/lunch_service.go` (前半)

Loop 3 (前半)。`service` 層の入口。**interface (= Go の抽象化の核)** が初登場する、4日間で一番重要な回。

このループでは **`SlackRepository` interface の宣言部分 + `LunchService` の構造体・コンストラクタ・`RunRecruit` メソッド** までを読む。
(後半 `RunAnnounce` は Day 2 / Loop 4 に持ち越し)

---

## ◆ 比較読み (Comparison Read) — コードを開く前に書く

> ⚠️ `internal/service/lunch_service.go` を開かずに、まず予想を書く。
> 質より量、勢いで埋める。後で実物と見比べたときの差分が学び。

さっきのを参考にすると、
handlerでつかわれていたメソッドとか定義してる場所だと思う。

だけど、handlerとserviceとrepsoの関係がわかってないから
それを把握しておきたいよね

### Q1. service って「何をする層」だと思う?

> handler / repository との違いを意識して、自分の言葉で1〜2行。
> 02_cmd_bot.md や 04_handler.md で書いた "層の分担" を思い出していい。

これは、えっと、文章を作成するとか、集計、ランダム化するとかを書いている部分だと思うな。だから、機能のほぼ本体くらいの感じだよね？

(ここに予想)

### Q2. `LunchService` 構造体はどんなフィールドを持っていそう?

Slackのスタンプの集計のやつ
あと、グループ分けのやつとかかな

> ヒント: handler と同じパターン。**何かを受け取って中に持っている** はず。
> 何個? 何が?

(ここに予想)

### Q3. `RunRecruit()` メソッドは中で何をやっていそう?

recruitは募集のメッセージを出力する
その後に、スタンプをつける
くらいかな。

> Slack に「今日のランチ参加者募集!」みたいなメッセージを投稿する処理が、
> どこかにあるはず。それを **誰かに頼んで** やってもらう書き方になる気がする。
>
> ステップを箇条書きで予想してみる (3〜5ステップでいい)。

(ここに予想)

### Q4. **interface** という単語、聞いたことある? 何だと思う?

あーこれまじで聞いたことあるけど、理解できていない部分だよ
interfaceは、言葉だけだと、他の型でも使えるようにするみたいな感じ。
汎用性を出すイメージだな。

> いまは知らなくても OK。「インターフェース」と日常で使う時の意味から推測する。
> 例: 「この機械のインターフェースは…」とか。
>
> service.go で **なんで interface を使うんだろう** と仮で言語化してみる。

(ここに予想)

### Q5. (応用) `cmd/simulate/main.go` が「実 Slack に繋がず動かせる」のは、なぜ可能だと思う?

うわわすれちゃったな、

> 02_cmd_bot.md / 04_handler.md で「冷蔵庫」の比喩を使った。
> もし冷蔵庫を **本物 / 模型 (fake)** で差し替えられる仕組みがあったら、
> どうやってそれを実現する?
>
> 完全に分からなくていい。**「こういう仕組みがあれば可能なはず」** という想像を書く。

(ここに予想)

---

✏️ ここまで書き終えたら Claude に見せる → 実物を一緒に読みにいく

---

## ◆ 実物を読んだあとに埋める (notes 6 項目)

> 凡例: section 4 (自分が引っかかった所) は **Day 1 で実際に詰まった点だけ** 記載。Day 2 以降に追加で詰まったら追記。それ以外のセクションは Day 1 + Day 2 想定で書いてある。

---

### ◆ 抽象まとめ (まずココで全体像を掴む)

**1文要約**:
> service.go = ランチbot の "頭脳"。Slack を **借り物 (interface 経由)** で使い、`RunRecruit` (募集を出す) と `RunAnnounce` (集計→グループ発表) の **2つのシナリオ** を提供する。

**抽象構造 (5要素)**:

```
┌────────────────────────────────────────────────────┐
│ ① 注文書    SlackRepository (interface)              │ ← 外との契約
│              5メソッド (Post / Reaction系3つ / Auth)  │
├────────────────────────────────────────────────────┤
│ ② 状態      LunchService (struct)                    │ ← 持ってる物
│              slack / channelID / emoji / lookback    │
├────────────────────────────────────────────────────┤
│ ③ 投稿テンプレ const                                 │ ← 出す側の文章
│              recruitmentText, restMessageText        │
├────────────────────────────────────────────────────┤
│ ④ 検索キー  prefix 定数                              │ ← 探す側の鍵
│              recruitPrefix, announcePrefix, restPrefix │
├────────────────────────────────────────────────────┤
│ ⑤ シナリオ  RunRecruit / RunAnnounce + helpers       │ ← 実際の手順
└────────────────────────────────────────────────────┘
```

**2つのシナリオを抽象化すると**:

| | RunRecruit (月曜) | RunAnnounce (火曜) |
|---|---|---|
| 入力 | なし (ただ呼ばれる) | なし (ただ呼ばれる) |
| Slack 操作 | 書く (Post + Reaction) | 読む4回 + 書く1回 |
| 副作用 | 募集投稿 + 自分の1票 | グループ発表 or お休み投稿 |
| ステート保持 | なし | なし (過去投稿が記憶) |
| 冪等性 | 弱い (毎週違う投稿になる) | 強い (二重実行をスキップ) |

**4つの設計思想 (この service が体現してるもの)**:

1. **ステートレス**: DB を持たない。Slack の投稿履歴が唯一の記憶 (= Source of Truth)。
2. **冪等性**: `RunAnnounce` を二重実行してもグループが2回発表されない (prefix で判定)。
3. **interface で外部依存を切る**: テスト/シミュレータで本物Slack 不要に。
4. **早期 return + エラーラッピング**: ネスト浅く、エラー文脈を失わずに上に伝える。

→ ここまでが抽象。詳細は下の6項目で。

---

### 1. このファイルが何をするか (1行サマリ)

`lunch-bot` の **業務ロジック本体**。Slack を「借り物 (interface)」として使い、`RunRecruit` (募集投稿) と `RunAnnounce` (集計→グループ発表) の2シナリオを提供する。

handler は呼び出すだけ、repository は HTTP を叩くだけ。**「いつ何をするか」のシナリオは全部この service にある**。

---

### 2. 主な型 (struct, interface)

#### `SlackRepository` (interface) — 10-17行目

service が Slack に頼みたい操作の **「注文書」**。中身 (実装) は無く、メソッドの型だけ並ぶ。

```go
type SlackRepository interface {
    PostMessage(channel, text string) (string, error)
    AddReaction(channel, timestamp, emoji string) error
    GetReactionUsers(channel, timestamp, emoji string) ([]string, error)
    WhoAmI() (string, error)
    RecentBotMessages(channel, botUserID string, sinceHours int) ([]repository.BotMessage, error)
}
```

→ service が外 (Slack) に依存する範囲が **この6行で全部見える**。
→ この5メソッドを満たすやつなら誰でも受け取れる (= fake と差し替え可能)。

#### `LunchService` (struct) — 19-24行目

業務ロジックを抱える本体。フィールド4つ。

```go
type LunchService struct {
    slack         SlackRepository  // ← interface 経由で外部依存を保持
    channelID     string           // ← 投稿先チャンネル
    emoji         string           // ← 投票絵文字 ("bento" 固定)
    lookbackHours int              // ← 過去何時間分の投稿を漁るか (26)
}
```

→ `slack` が **interface 型** なのが超重要ポイント。

---

### 3. 主な関数 / メソッド (引数・戻り値・1行説明)

| 名前 | 引数 | 戻り値 | 1行説明 |
|---|---|---|---|
| `NewLunchService` | `slack SlackRepository, channelID string` | `*LunchService` | コンストラクタ。`emoji="bento"`, `lookbackHours=26` をデフォで仕込む |
| `(s) RunRecruit` | なし | `error` | 募集投稿 + bot 自身が🍱を1個押す |
| `(s) RunAnnounce` | なし | `error` | bot自身を識別→過去投稿取得→冪等性チェック→募集投稿発見→集計→bot除外→人数判定→Shuffle→発表投稿 |
| `excludeUser` | `users []string, exclude string` | `[]string` | 指定ユーザーを除いた新スライスを返す (主にbot自身を除外) |
| `buildAnnouncement` | `groups [][]string` | `string` | グループA/B/C…形式の発表文章を組み立てる |

#### `RunRecruit` の中身 (53-66行目) のステップ

```
1. ログ出力: "[recruit] 募集投稿を出します"
2. slack.PostMessage(channelID, recruitmentText) → ts 取得
3. エラーなら fmt.Errorf("post recruitment: %w", err) で包んで return
4. ログ出力: "投稿成功 ts=..."
5. slack.AddReaction(channelID, ts, "bento")
6. エラーなら fmt.Errorf("add reaction: %w", err) で包んで return
7. ログ出力: "自分で🍱を押しました"
8. return nil
```

→ ポイント: 各ステップの **節目で fmt.Println**。障害時にどこまで成功したか追える。
→ ポイント: エラーは `%w` で包む = 上位で `errors.Is` 判別可能にする。

#### `RunAnnounce` の中身 (70-127行目) のステップ

```
1. slack.WhoAmI() → bot 自身の user ID を取る
   ・なぜ必要? → 後で「集計から自分を除外する」「過去投稿の絞り込み」のため

2. slack.RecentBotMessages(channelID, botID, 26h) → 直近26時間の bot 投稿リスト
   ・空なら error: "bot 投稿が直近 26h に見つからない (recruit が走っていない可能性)"
   ・= 月曜の recruit が失敗してるとここに来る

3. 冪等性チェック (= 二重実行ガード)
   ・msgs[0] (= 一番新しい bot 投稿) のテキストの先頭を見る
   ・announcePrefix or restPrefix で始まる → 既に announce 完了 → return nil でスキップ
   ・なぜ rest も対象? → 「お休み」も "今週の announce 完了シグナル" だから (Phase 2 発見バグ修正)

4. 募集投稿を探す
   ・msgs を頭から舐めて recruitPrefix で始まるやつを探す
   ・無ければ error
   ・あれば recruit.TS を取る (= 後で reaction 集計の宛先)

5. slack.GetReactionUsers(channelID, recruit.TS, "bento") → 🍱を押した user ID のリスト
6. excludeUser(users, botID) → bot 自身 (RunRecruit で1票仕込んだやつ) を除外
   ・ログ: "参加者 N 人 (bot自身を除外後)"

7. 人数判定:
   ・参加者 < 3 → restMessageText を投稿 → return nil (お休み)
   ・参加者 >= 3 → 続行

8. Shuffle(users) → グループ分け (3〜5人組) ※実装は shuffler.go (Loop 5)
9. buildAnnouncement(groups) → 発表テキスト組み立て
10. slack.PostMessage(channelID, announcement) → 投稿
11. return nil
```

→ ポイント: **「ステートレス」**。DBを持たず、Slack の投稿履歴だけを source of truth にする。冪等性も「過去投稿を見て決める」。
→ ポイント: **早期 return の連発**。エラー or 条件不成立で即 return することで、ネストが深くならない。

#### `excludeUser` の中身 (129-137行目)

```go
func excludeUser(users []string, exclude string) []string {
    out := make([]string, 0, len(users))   // 容量だけ確保 (アロケ削減)
    for _, u := range users {
        if u != exclude {
            out = append(out, u)
        }
    }
    return out
}
```

→ 元のスライスを破壊せず **新しいスライス** を返す (immutable な作法)。
→ `make([]string, 0, len(users))` の第3引数 = 容量。最大要素数を予約しておくと append で再アロケが起きにくい。

#### `buildAnnouncement` の中身 (139-151行目)

```go
func buildAnnouncement(groups [][]string) string {
    var sb strings.Builder           // 文字列を効率的に組み立てるバッファ
    sb.WriteString("🎉 今週のランチグループ決定！\n\n")
    for i, g := range groups {
        mentions := make([]string, len(g))
        for j, uid := range g {
            mentions[j] = fmt.Sprintf("<@%s>", uid)   // Slack のメンション形式
        }
        fmt.Fprintf(&sb, "グループ%c: %s\n", 'A'+i, strings.Join(mentions, " "))
    }
    sb.WriteString("\n水曜のランチで楽しんで！🍱")
    return sb.String()
}
```

→ `strings.Builder` を使う理由: 文字列連結 (`+=`) は毎回新しい文字列を作るので非効率。Builder は内部バッファに書き溜めて最後に1回 `String()`。
→ `'A' + i` = rune の演算。`i=0` なら 'A'、`i=1` なら 'B'…と進む。グループ A/B/C/D/… の連番ラベル生成の定石。
→ `<@uid>` = Slack でメンションを発火させる形式。これで実ユーザーに通知が飛ぶ。

---

### 4. 自分が引っかかった所

#### (a) `slack SlackRepository` という引数の書き方

`NewLunchService(slack SlackRepository, channelID string)` の `SlackRepository` の位置に **interface 名** が入ってるのが最初は意味不明だった。

整理:
- `slack` = 引数の名前
- `SlackRepository` = 型 (= 上で宣言した interface)
- 普通の `func f(x int)` の `int` の場所に interface が入ってるだけ
- 意味: 「`SlackRepository` を満たすやつなら **誰でも** 受け取る」

→ これがわかると interface の威力が見える。「型は具体的な struct じゃない、契約 (= 5メソッド持ってるか) で受け取る」。

#### (b) `%w` と `%v` の違い

最初「`%w` は変数を埋め込むやつ?」と思ってた。それは `%v` `%s`。

- `%v` → 値を文字列化して埋める。元 err は文字列に潰れる
- `%w` → 元 err を **潰さず封筒に包む**。後で `errors.Is(err, ...)` で中身判別可能

比喩: 写経して送る (`%v`) vs 封筒に入れて送る (`%w`)。

#### (c) `recruitPrefix` と `recruitmentText` が **物理的に分かれてる** のがなぜか

最初「DRY 違反では?」と思った。実はちゃんと理由がある:

- 投稿: Unicode (`🍽️`) で書く → Slack 内部で colon-code に正規化
- 検索: Slack から返ってくる text は colon-code → `recruitPrefix` も colon-code じゃないと一致しない
- → 物理的に同じ文字列にできない (Phase 2 で発見されたバグ)

#### (d) `fmt.Println` の出力先

stdout。
- ローカル → ターミナル
- GitHub Actions → ワークフローログ
- 普段は誰も読まない、**障害時に開く**。

#### (e) interface の "実装する側" と "使う側" の因果関係 (Day 2 復習で誤解した点)

最初「interface を定義したら、必ずどこかで実装しなきゃいけない」と思った。**因果が逆**。

正しくは:
- interface 定義そのものは何も強制しない (ただの型宣言)
- **interface を "受け取る場所"** (フィールド / 引数 / 戻り値) に値を渡そうとした瞬間、その値の型が **5メソッド全部実装してるか** 検査される
- 欠けてたらコンパイルエラー: `*fakeSlack does not implement SlackRepository (missing method WhoAmI)`

→ つまり「interface があるから実装が要る」ではなく「**使う場所があるから実装が要る**」。
→ 実用的には "interface 定義 + 使う場所 + 実装する型" の3点セットで初めて意味がある。

ついでに、最初「形は何でもいい、5つ持ってれば」と曖昧に理解したら混乱した。正確には:

| 何が | 自由度 |
|---|---|
| **struct の中のフィールド (内部状態)** | 何でもいい (本物は token/httpClient、fake は msgs/real など全然違う) |
| **interface に書かれたメソッド** | 5つ全部必須、1個でも欠けたらアウト |

#### (f) `make([]string, 0, len(users))` の `len(users)` の意味

第3引数 = **容量** (= "最大何件入る予定?" の事前申告)。第2引数 (長さ) とは別物。

- 第2引数 `0` → 初期の要素数 (空)
- 第3引数 `len(users)` → 最大値の予測 = 事前にメモリを確保しておくサイズ

なぜ事前確保?
- スライスは append で容量を超えると **裏で全コピーして拡張** する (引っ越し)
- 最大サイズが分かってれば最初に確保しておく → 引っ越しゼロ → ちょっと速い

excludeUser は「bot を除外するだけ」 → out のサイズは絶対に users 以下 → `len(users)` で十分予測できる。

#### (g) `msgs` の正体 = Slack のスナップショット

最初「`msgs` は何? どこで定義されてる? 毎回 Slack を見にいってるの?」で混乱した。

整理:
- `msgs` はローカル変数 (Goメモリ上)
- 中身 = `RecentBotMessages()` が返した **`[]repository.BotMessage`** (struct のスライス)
- 毎回 RunAnnounce が走るたびに **HTTP で Slack から取得し直す** (= スナップショット)
- `s.lookbackHours = 26` で時間制限 (直近26時間の bot 投稿だけ)

`BotMessage` 自体は超シンプル: `{ TS string; Text string }` だけ。

→ ETL の文脈で見ると一発で腹落ちする:
```
[Extract]   77行: RecentBotMessages()      ← Slack から取得 (msgs)
[Transform] 88-120行: 冪等性 / 募集探し / 集計 / Shuffle / 整形
[Load]      122行: PostMessage()             ← 結果を Slack に投稿
```

これが `notes/00_overview.md` で図示する "ETL 対応図" の中核。

---

### 5. 他にもありえた選択 (Counter-design)

#### 案A: `recruitPrefix = recruitmentText` にしてしまえば DRY じゃない?

却下理由は2つ:

1. **絵文字の物理的差**: 投稿側 Unicode / 検索側 colon-code → そもそも同じ文字列にできない
2. **本文改修への耐性**: prefix を本文と分けておくと、本文を1文字直しても過去投稿の検索が壊れない。同一にすると「本文を更新した瞬間、それ以前の投稿が検索ヒットしなくなる」 → 冪等性チェックが壊れる

→ DRY を捨てた代わりに **「変更に対する安全性」** を取った設計判断。

#### 案B: `SlackRepository` を interface にせず、直接 `*repository.SlackClient` を持てば良かったのでは?

もし service が `slack *repository.SlackClient` を持つと:
- 短期的には動く
- でも `cmd/simulate` で本物Slack 抜きにテストしたいとき、差し替え不能 (型が固定)
- → 「テスト/シミュレーション可能性」を確保するために interface にしてる
- → これが Day 1 の最大の学び

#### 案C: `RunRecruit` を handler に直接書いて、service 層を作らなければ良かったのでは?

- handler が肥大化する
- 「Slack を呼ぶロジック」と「コマンド分岐」が同じ層に混ざる → 関心分離が崩れる
- handler は "委譲だけ" の薄い層に保つことで、新しいトリガー (例: HTTP / Slack slash command) を増やしても service ロジックは触らずに済む

#### 案D: `%w` じゃなく `%v` で十分だったのでは?

- ログを出すだけなら `%v` で十分
- でも将来 「特定エラーだけリトライしたい」 のような分岐がしたくなったら、`%w` で包んでないと中身を取り出せない
- → **防御的に `%w`** にしてる (コストほぼゼロ、後悔リスク回避)

#### 案E: 冪等性チェックを最初から DB で持てば良かったのでは?

`RunAnnounce` の冪等性チェックは **「過去の bot 投稿の prefix を見る」** 方式。代わりに DB (Postgres / Redis) に「今週は処理済み」フラグを立てれば確実かもしれない。

採用しなかった理由 (推測):

- DB を増やすと運用コスト上がる (接続・スキーマ管理・ホスティング)
- 「Slack 投稿があれば処理済み」と素直に解釈すれば DB 不要
- → **ステートレス維持** という設計思想を優先 (Source of Truth は1つ = Slack)
- 副作用: prefix 文字列を変えるとバグる、絵文字正規化問題が発生する → 案A の議論につながる

これは **設計判断のトレードオフの典型例**。「シンプルさ vs 確実性」のバランスを取った例として読むと深い。

#### 案F: `excludeUser` を service の中じゃなく `[]string` メソッドにすれば?

例えば `users.Without(botID)` みたいに。Goでは型エイリアスを切ればできる:

```go
type UserList []string
func (u UserList) Without(exclude string) UserList { ... }
```

- 採用しない理由: Go の慣例として **「ただの一覧」にメソッドを生やすのは過剰**
- 関数で十分な処理にわざわざ型を作ると、読み手が「この型の意味は?」と考えてしまう
- → 「素直に書ける処理は素直に関数で」という判断 (Go の "small is beautiful")

---

### 6. 次に読む人へのアドバイス (= 未来の自分へ)

1. **interface を見たら「契約」と読み替える**: メソッド名と型のリストは「これらの操作ができるやつ」という型条件の宣言。具体的な struct を渡す必要はない。`cmd/bot/main.go` で渡してる `SlackClient` は「契約を満たす1つの実装」にすぎない。

2. **「型が一致すれば気付かない」を体感したいなら `cmd/simulate` (Loop 7)**: 本物Slack 不要でロジック検証できる仕組みは、ここの interface 抜きには成立しない。

3. **const を読むときは「投稿用」と「検索用」を区別する**: 同じ "募集" のテキストでも、投稿用 (Unicode) と検索用 (colon-code) が別々の定数になってる。Slack の正規化挙動を知らないと意味不明だが、知れば自然。

4. **`%w` を見たら「上位がエラー判別したいかも」と読む**: ログだけなら `%v` でも良いが、`%w` を選んでるなら「将来 `errors.Is` するかもしれない」という防御。

5. **ログ出力は障害調査の道具**: `fmt.Println` の節目はランダムじゃない。「ここまで成功した」を後で追えるように仕込まれてる。GitHub Actions のログでこの行が出てるか出てないかが、原因切り分けの最初の手がかり。

6. **`RunAnnounce` は "7段階パイプライン" として読む**: 認証 → 過去投稿取得 → 冪等性チェック → 募集投稿発見 → 集計 → 人数判定 → 発表投稿 (or お休み投稿)。各段階で **失敗すれば早期 return**。これがネストを深くしないコツ。

7. **冪等性チェックを読むときは「何を見て判定してるか」に注目**: `msgs[0].Text` (= 一番新しい bot 投稿の本文) の **先頭** を見て、`announcePrefix` or `restPrefix` で始まれば「今週分処理済み」と判定。**お休み (rest) も完了シグナルに含めてる** のが Phase 2 で発見されたバグの修正跡 (= 設計判断3)。

8. **`prefix` 検索は「タイトル相当」だけで識別する設計**: 過去投稿の本文全体ではなく **先頭1行** だけを比較する。本文を1文字直しても識別が壊れない柔軟さがある。

9. **`Shuffle` と `buildAnnouncement` の役割分担**: `Shuffle` は **数学的処理** (グループ分けロジック)、`buildAnnouncement` は **文章組み立て**。"計算" と "表現" を分けてる典型。テストもしやすくなる (Shuffle 単体でテーブル駆動テスト可能)。

10. **`<@USERID>` は Slack のメンション形式**: `buildAnnouncement` で `fmt.Sprintf("<@%s>", uid)` してるのは、これで実ユーザーに通知が飛ぶから。シミュレータで偽IDを渡すとメンションリンクが死ぬが、ロジック検証には影響しない。

---

**Day 1 終了時点のキーセンテンス** (これが言えれば抽象理解OK):

> service.go は、Slack を **借り物 (interface)** として使う **2シナリオ実装**。投稿用と検索用で文字列を物理分離し、ステートレス設計で冪等性を保つ。エラーは `%w` で包んで上に投げ、ログは障害調査用に節目で吐く。
