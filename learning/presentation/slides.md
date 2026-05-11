# 5/15 発表 スライド + 台本

- **発表者**: ご主人様 (26卒・未経験エンジニア)
- **対象**: 笹本さん他、計9名
- **目的**: 自作 lunch-bot を足場に、Polaris (unified-api / task-dispatcher) の構成を理解していることを伝える
- **想定時間**: 約 25〜26 分 (Q&A 別)
- **構成**: 18 スライド (タイトル → 自己紹介 → ゴール → 自作 → ETL → 3層 → DI → Polaris → 対応表 → 深掘り → 締め)

## 進行のしかた

- 各スライドの下に `> **🎤 台本**` の引用ブロックで台本が埋め込まれている
- 通しで 2 回リハーサルしてから本番に臨む
- 専門用語が出てきたら必ず冒頭で 1 行説明 (用語集 `01_glossary.md` 参照)

---

# スライド 1: Slack ランチ bot を足場に Polaris (unified-api) の構成を理解する

- 発表者: 26卒・未経験エンジニア
- Go の基本文法は習得済み / Cloud は学習中
- vibe coding (= 雰囲気と AI の出力に頼って書くスタイル) で作った lunch-bot を、**読解教材として骨の髄まで分解した**
- 今日は **「自作の小さなアプリ」と「業務の大きなコード」の対応づけ** に集中して話します

> **🎤 台本** (推定 60 秒)
>
> (沈黙、聞き手を見渡す)
>
> はじめます。タイトルは「Slack ランチ bot を足場に Polaris の構成を理解する」です。
>
> 私は26卒の未経験エンジニアで、Go の基本文法は習得済み、クラウドは学習中です。
>
> 自分が vibe コーディング、雰囲気と AI に頼って書くやり方で作った lunch-bot を、今度は読解教材として骨の髄まで分解しました。動くけど自分で直せない、という状態から抜け出すためです。
>
> 今日は、その自作の小さなアプリと、業務の Polaris という大きなコードが、実は同じ骨格でできている、という話をします。よろしくお願いします。

---

# スライド 2: 今日のゴール (3つ)

1. lunch-bot が **3層構造 (handler / service / repository)** で動いていることを説明する
2. lunch-bot と **Polaris (unified-api / task-dispatcher)** が同じ骨格でできていることを説明する
3. Go コードを Polaris 全体の構成に **関連づけて** 理解していることを示す

> 用語は出すたびに 1 行で説明します。分からない言葉が出たら止めてください。

> **🎤 台本** (推定 60 秒)
>
> 今日持ち帰ってほしいことを 3 つ宣言します。
>
> 1 つめ、lunch-bot は handler、service、repository の 3 層構造で動いている、という話。それぞれ「入口の係」「中身の係」「外と話す係」です。
>
> 2 つめ、その lunch-bot と、業務の Polaris のうち unified-api と task-dispatcher が、同じ骨格でできている、という話。今日の主戦場です。
>
> 3 つめ、Go のコードを Polaris 全体に関連づけて理解できている、というのを示したいです。
>
> 用語は出すたびに 1 行で説明します。分からない言葉が出たら止めてください。

---

# スライド 3: lunch-bot とは何か

- 週 2 回、Slack に **ランチ募集メッセージ** を投げる Slack bot
- お弁当絵文字でリアクションした人を集めて、ランダムに 3〜4 人組に分けて発表する
- **cron** (= 「毎週月曜9時」のような定期実行スケジュール) で自動実行 / 人手は不要
- 入力も出力も **Slack** だけ。Slack が唯一のデータソース
- 1 個の Go バイナリで実装 (本体のみ約 640 行)

> **🎤 台本** (推定 90 秒)
>
> まず lunch-bot 自体の説明です。週 2 回、Slack のチャンネルに「今日のランチどうですか」って募集メッセージを自動で投げる、それだけのボットです。
>
> 流れはこうです。募集メッセージが投げられて、行きたい人はお弁当の絵文字でリアクションする。一定時間後に bot がもう一度起動して、リアクションした人をランダムに 3 人や 4 人のグループに分けて、発表メッセージを投げる。それで終わりです。
>
> ポイントは 2 つあります。1 つめ、人手はいっさい要らない。cron という、「毎週月曜と火曜の朝 9 時に動かす」みたいな定期実行の仕組みで勝手に動きます。
>
> 2 つめ、入力も出力も Slack しかありません。データベースも持ってなくて、Slack のメッセージとリアクションが唯一のデータの置き場所です。
>
> これを 1 個の Go バイナリ、本体のみ約 640 行で実装しています。小さなアプリです。

---

# スライド 4: lunch-bot の処理を ETL で見る

- **ETL** = **E**xtract / **T**ransform / **L**oad の頭文字。「取って・変えて・戻す」処理パターン

```
[Extract: 取る]         [Transform: 変える]        [Load: 戻す]

  Slack API     ──→     service (シャッフル)   ──→   Slack API
  リアクション取得          グループ分け                   発表メッセージ投稿
```

- Polaris も **同じ ETL を、もっと大規模で複数のソースに対して** やっているだけ

> **🎤 台本** (推定 60 秒)
>
> ここで 1 つ枠組みを渡します。lunch-bot は ETL という考え方で見るとシンプルに整理できます。
>
> ETL は Extract、Transform、Load の頭文字、「取って・変えて・戻す」の 3 段階です。
>
> lunch-bot だと、Extract は Slack からリアクション一覧を取ってくるところ。Transform は service の中で参加者をシャッフルしてグループ分けするところ。Load は Slack に発表メッセージを投稿するところ。きれいに 3 段階に分かれます。
>
> 後半の Polaris も、まったく同じ ETL です。違うのは規模と連携先の数だけです。

---

# スライド 5: lunch-bot の3層構造 (handler / service / repository)

```
  cmd/bot/main.go           (起動点・3層を組み立てる)
        │
        ▼
  handler                   (入口の係 = 受付カウンター)
        │                    どのコマンドが来たか分岐するだけ
        ▼
  service                   (業務ロジックの係 = 調理場)
        │                    シャッフル・集計・メッセージ組立
        ▼
  repository                (外部の係 = 仕入れ係)
        │                    Slack API を叩く
        ▼
  Slack
```

- 各層は **interface** (= 「これらのメソッドを持っていれば誰でも OK」という契約書) で繋がる
- 上の層は下の **具体実装** を知らない (= **責務分離**)

> **🎤 台本** (推定 90 秒)
>
> ここから中身に入ります。lunch-bot は handler、service、repository の 3 層構造です。
>
> handler は入口の係。レストランで言うと受付カウンターで、注文を受けて奥に伝票を回すだけ、料理は作りません。recruit が来たら募集する側、announce が来たら発表する側、と分岐するだけの薄い層です。
>
> service は業務ロジック、調理場です。シャッフル、集計、メッセージ組立、「やりたいこと」の本体は全部ここです。
>
> repository は外部の係、仕入れ係。Slack の API を叩くコードはここに集約されていて、他の層は Slack を直接触りません。
>
> この 3 つを上から下に組み立てるのが cmd/bot/main.go で、最後に handler の Recruit を呼びます。
>
> 各層は interface、「このメソッドを持っていれば誰でも OK」という契約書で繋がります。上の層は下の具体的な中身を知らない。これが責務分離です。

---

# スライド 6: lunch-bot の DI と interface (差し替えできる嬉しさ)

- **DI (Dependency Injection)** = 依存先を「外から渡してもらう」設計手法
- service は **interface** を受け取る (具体実装ではない)

```go
type SlackRepository interface { /* 5メソッド */ }

// 本番: 実物の Slack クライアントを渡す
svc := service.NewLunchService(repository.NewSlackClient(token), ch)

// テスト: fake (偽物) を渡す → Slack を叩かずメモリで動く
svc := service.NewLunchService(&fakeSlack{}, ch)
```

- service のコードは 1 文字も変えていない。**渡す部品を入れ替えるだけ**で本物⇄偽物が切り替わる
- これが DI の威力 / **同じ仕組みが Polaris の中でも使われています** (詳しくは後ほど)

> **🎤 台本** (推定 90 秒)
>
> ここで DI と interface の話を軽くしておきます。
>
> DI、Dependency Injection は「依存先を外から渡してもらう」設計手法です。料理人が自分で市場に行かないで、食材は誰かが届けてくれる前提で料理する、みたいなイメージです。
>
> lunch-bot だと、service は SlackRepository という interface、「このメソッドを持ってる人なら誰でも OK」という契約書を受け取っています。具体的な中身は知りません。
>
> 本番では実物の Slack クライアントを渡して API を叩きます。テスト用には fake、偽物の Slack を渡します。Slack を叩かずメモリだけで動く偽物です。
>
> 嬉しいのは、service のコードは 1 文字も変えていない、ということ。渡す部品を入れ替えるだけで本物にも偽物にも切り替わる。これが DI の威力です。
>
> そして、まったく同じ仕組みが Polaris の中でも使われています。後半で回収します。

---

# スライド 7: Polaris とは何か

- Amazon / 楽天 / 広告 API などから **毎日データを集めて BigQuery に貯めて、分析しやすい形に整える工場**
- やっていることは lunch-bot と同じ ETL / 違うのは **規模・連携先の数・起動の仕組み**

```
[外部のサービス]                  [Polaris の役割]                  [出口]

Amazon / 楽天 / 広告API   ──→   ingestion/ (集めてくる)       ──→   BigQuery
                                 unified-api ★ 本編の主役
                                 task-dispatcher ★ 本編の主役

                                 transformation/ (dbt で加工)  ──→   BigQuery
                                                                    (整形済みデータ)
```

- 本編で扱う 2 フォルダ: **`unified-api`** (集める本体) / **`task-dispatcher`** (時間で叩く係)
- 補足で扱う: `transformation/` (dbt = SQL で加工する世界)

> **🎤 台本** (推定 60 秒)
>
> 前半の最後に、Polaris が何か 1 枚で押さえます。
>
> Polaris は、Amazon・楽天・広告 API みたいな外部サービスから毎日データを集めて、BigQuery という倉庫に貯めて、分析しやすい形に整える、社内の ETL 工場です。
>
> 大事なのは、やっていることは lunch-bot と同じ ETL だ、ということ。違うのは規模・連携先の数・起動の仕組み、この 3 つだけです。
>
> 本編で扱うのは 2 つのフォルダ。unified-api は集めてくる本体、task-dispatcher は時間で unified-api を叩く係。この 2 つが lunch-bot とどう対応するか、が後半のメインです。
>
> ここから対応表に入ります。

---

# スライド 8: lunch-bot ↔ Polaris 対応表 ★ メインスライド

| 観点 | lunch-bot (自作) | Polaris (実務) |
|---|---|---|
| 起動の引き金 | GitHub Actions の **cron** | **Cloud Scheduler** (= 時間で HTTP を叩いてくれるタイマー) |
| 入口層 (handler) | `internal/handler/lunch_handler.go` (1 ファイル) | `ingestion/unified-api/internal/handler/` (約 20 ファイル) |
| 業務 (service) | `internal/service/lunch_service.go` (1 個) | `internal/service/<platform>/service.go` (プラットフォーム別) |
| 外部連携 (repository) | `internal/repository/slack_client.go` (Slack のみ) | `internal/repository/<platform>/` (17 種類) |
| DI 組立 | `cmd/bot/main.go` で 3 行 | `cmd/api/routes.go` の `registerXxxRoutes()` 群 |
| データ保存 | なし (Slack 自体がソース) | **BigQuery** (= GCP の超大容量 DWH) |
| 実行単位 | 1 cron = 1 同期処理 | 1 cron = N **テナント** (= 顧客 1 社) 分にファンアウト |

> **🎤 台本** (推定 180 秒)
>
> ここからが今日の本丸です。一枚で「lunch-bot ↔ Polaris」の対応をまとめた表を出しました。7 行ありますが、180 秒で全部は読めません。**上から 3 行、起動の引き金・入口層・業務、ここに集中します。残りは後ろのスライドで個別に深掘りするので今は流します。**
>
> まず一行目、起動の引き金。lunch-bot は GitHub Actions の cron で動いています。Polaris は Cloud Scheduler、これは時間で HTTP を叩いてくれる GCP のタイマーです。役割は完全に同じで、「決まった時刻に、決まった処理を呼び起こす」やつです。
>
> 二行目、入口層 handler。lunch-bot は `lunch_handler.go` 一個だけ。Polaris は約 20 ファイルあります。でも構造は同じで、「リクエストを受けて service に橋渡しするだけ」の薄い層です。違いは「連携先が増えたから、その分ファイルが増えた」だけ。
>
> 三行目、業務 service。lunch-bot は `lunch_service.go` 一個に全部のロジックが入っています。Polaris はプラットフォームごと、Ecforce / 楽天 / Amazon と分かれているので、その単位で service が一個ずつあります。これも増えただけ。骨格は同じです。
>
> 残りの 4 行、repository、DI 組立、データ保存、実行単位は次のスライド以降で順番に拾います。
>
> 結論を一行で言うと、**「3 層プラス DI の骨格は完全に同じ。違いは 4 つだけ。起動口の数、連携先の数、書き込み先が BigQuery になったこと、そして 1 つの cron が N テナント分にファンアウトされること」**。これだけです。次のスライドから、この 4 つの違いを順に見ていきます。

---

# スライド 9: unified-api の 3 層 (handler / service / repository)

**結論: lunch-bot と同じ 3 層。依存の数が増えただけ。**

```go
// lunch-bot (15行以内)
type LunchService struct {
    slack         SlackRepository  // 依存 1 個
    channelID     string
}
func NewLunchService(slack SlackRepository, ch string) *LunchService {
    return &LunchService{slack: slack, channelID: ch}
}
func (s *LunchService) RunRecruit() error {
    ts, err := s.slack.PostMessage(s.channelID, recruitmentText)
    if err != nil { return fmt.Errorf("post recruitment: %w", err) }
    // ...
}
```

```go
// Polaris (Ecforce, 15行以内)
type DefaultEcforceService struct {
    httpRepo  EcforceHTTPRepository  // 外部API
    bqRepo    *loader.BqRepository   // BigQuery
    mdmRepo   mdm.Repository         // テナント情報
    etlRunner *etl.Runner            // 共通 ETL
}
func NewService(httpRepo EcforceHTTPRepository, bqRepo *loader.BqRepository,
    mdmRepo mdm.Repository, etlRunner *etl.Runner) *DefaultEcforceService {
    return &DefaultEcforceService{httpRepo, bqRepo, mdmRepo, etlRunner}
}
```

- **handler**: 入口の係。HTTP の作法 (decode / validate / JSON 返す) が増えただけで、本質は「service を 1 行呼ぶ」
- **repository**: lunch-bot の `SlackClient` と Polaris の `BotRepository` は **ほぼコピペレベルで同じ骨格**

> **🎤 台本** (推定 120 秒)
>
> 中身を見ていきます。lunch-bot と Polaris の service を、それぞれ 15 行に圧縮して並べました。
>
> 左が lunch-bot、`LunchService` という struct、フィールドは `slack` と `channelID`。コンストラクタ `NewLunchService` で引数を受け取って struct を返す。`RunRecruit()` の中で `PostMessage` を呼んで、エラーは `fmt.Errorf` で wrap。これだけです。
>
> 右が Polaris の Ecforce service。同じく struct、コンストラクタ、メソッド。**形は完全に一緒**。違いはフィールドの数だけ。lunch-bot は依存 1 個、Polaris は 4 個、外部 API・BigQuery・Supabase・共通 ETL ランナー。**増えただけで組み方は同じ**です。
>
> handler は HTTP の作法、デコードとバリデーションが増えてますが、**「最後に service を 1 行呼ぶ」のは lunch-bot と同じ**。repository は lunch-bot の `SlackClient` と Polaris の `BotRepository` が **ほぼコピペ**。
>
> 結論、**3 層は完全に対応。骨格は同じ、依存が増えただけ**。

---

# スライド 10: unified-api の DI 組立 (`routes.go`)

**結論: lunch-bot と同じ「下から順に組み上げる」。違いは register 関数で分割していること。**

```go
// lunch-bot: cmd/bot/main.go (3行)
slack := repository.NewSlackClient(cfg.SlackToken)        // 1. 一番下を作る
svc   := service.NewLunchService(slack, cfg.ChannelID)    // 2. それを渡して service を作る
h     := handler.NewLunchHandler(svc)                     // 3. それを渡して handler を作る
```

```go
// Polaris: cmd/api/routes.go (Ecforce の register 関数, 約10行)
func registerEcforceRoutes(r chi.Router, mdmRepo mdm.Repository,
    bqRepo *loader.BqRepository, slackService slackservice.AsyncService) {
    httpRepo  := ecfrepo.NewHTTPRepository(...)                       // 1. repository
    etlRunner := etl.NewRunner(...)                                   // 2. 部品
    svc := ecfsvc.NewService(httpRepo, bqRepo, mdmRepo, etlRunner)    // 3. service
    eh  := handler.NewEcforceHandler(svc, slackService)               // 4. handler
    r.Route("/ecforce", func(r chi.Router) {                          // 5. URL に紐付け
        r.Post("/customers", eh.PostCustomers)                        //    (chi = HTTP のパスを関数に紐付ける Go ライブラリ)
        r.Post("/orders", eh.PostOrders)
    })
}
```

- 共通リソース (BigQuery / Supabase / Slack 通知) は **1 回だけ** 作って各 register に渡す
- 最後の `r.Post(...)` が lunch-bot の `switch os.Args[1]` 相当
- **DI の本質は「順番に組むだけ」**

> **🎤 台本** (推定 120 秒)
>
> 次は DI の組立場所。スライド 6 で渡した「依存先を外から渡してもらう」やつ、これが Polaris ではどう書かれているか、を lunch-bot と並べて見ます。
>
> 上が lunch-bot。`cmd/bot/main.go` の中で **たった 3 行**。一行目で一番下、SlackClient を作る。二行目で service を作る、引数に slack を渡す。三行目で handler を作る、引数に service を渡す。**下から順に積み上げているだけ**。これが DI です。
>
> 下が Polaris、`cmd/api/routes.go` の `registerEcforceRoutes`。一行目 repository、二行目 ETL ランナー、三行目 service、四行目 handler。**lunch-bot と全く同じ「下から順に積む」パターン** です。違いはプラットフォームが 20 種類あるので、register 関数に分割していること、それだけ。
>
> 共通リソース、BigQuery・Supabase・Slack 通知は全プラットフォームで使い回すので、**最初に 1 回だけ作って** register 関数に引数で渡します。
>
> 最後の `r.Post("/customers", eh.PostCustomers)`、ここで URL とハンドラを紐付け。**chi** は Go で HTTP のパスを関数に紐付けるライブラリ。lunch-bot で言う `switch os.Args[1]` の代わりです。CLI のサブコマンド分岐が、HTTP の URL 分岐に置き換わっただけ。
>
> 一撃で覚えてほしいのは、**DI の本質は「順番に組み上げるだけ」**。難しい概念じゃないです。

---

# スライド 11: ジョブ呼び出し vs API エンドポイント呼び出し ★

**ここが今日の山場の 1 つ。スライド 6 の「fake と本物の差し替え」が、Polaris では「2 つの入口」として実際に効いている。**

```
┌─ API エンドポイント経由 ─────────────────────────────────┐
│  Cloud Tasks ──HTTP POST /api/v2/ecforce/customers──>    │
│        cmd/api/main.go (chi router)                      │
│              ▼                                           │
│        internal/handler/ecforce.go                       │
│              ▼                                           │
│        internal/service/ecforce/service.go ← ★ 合流     │
└──────────────────────────────────────────────────────────┘
                          ▲
                          │ (同じ service を共有)
                          ▼
┌─ ジョブ経由 (Cloud Run Jobs / CLI) ──────────────────────┐
│  Cloud Run Jobs ──CLI 起動──> cmd/jobs/main.go           │
│                                  ▼                       │
│                          internal/jobs/ecforce/runner.go │
│                                  ▼ (handler を通らない)  │
│                          internal/service/ecforce/service.go │
└──────────────────────────────────────────────────────────┘
```

- **API エンドポイント呼び出し** (= `unified-api` の `/api/v2/...` を HTTP で叩く): 短時間処理向け
- **ジョブ呼び出し** (= **Cloud Run Jobs** で `cmd/jobs/main.go` を CLI 起動): 30 分超の長時間処理向けの脱出口
- **service は完全に同じものを使い回す**

> **🎤 台本** (推定 120 秒)
>
> ここが今日の山場の 1 つです。**スライド 6 で言った「fake と本物の Slack を差し替えられた」話、覚えてますか? あれが Polaris では『2 つの入口』として実際に効いています**。
>
> 図を見てください。上と下の経路が、真ん中の `service.go` で **合流** しています。**service は 1 つだけ**。それを 2 通りの入口から呼べる設計です。
>
> 上が **API エンドポイント呼び出し**。Cloud Tasks が HTTP で叩く、chi ルータ、handler、service の順、普通の Web サーバの流れ。
>
> 下が **Cloud Run Jobs**、Cloud Run のバッチ版で HTTP を持たず CLI を 1 回走らせて終わる。`runner.go` が **handler を通らず** 直接 service を呼びます。
>
> なぜ 2 系統あるか。答えは **時間**。Cloud Run の HTTP は 30 分でタイムアウト。30 分以内なら上、超えるなら下、それだけ。
>
> ここで **service は完全に同じもの** を使い回せている。理由は **interface 経由で repository を受け取っている** から。lunch-bot で fake と本物を差し替えたのと **全く同じ仕組み**。テスト用に見えた話が、本番では **「2 種類の起動口」を実現する核** になっています。
>
> 一撃の核、**「interface を切ったから、同じ service を別の入口から再利用できる」**。DI の本番回収です。

---

# スライド 12: task-dispatcher と 4 段非同期の絵 ★

**結論: lunch-bot の「1 段同期」を、Polaris は「4 段非同期」に拡張している。**

```
[① タイマー]                  [② 振り分け係]              [③ 待ち行列]               [④ 働く係]

Cloud Scheduler              task-dispatcher              Cloud Tasks                unified-api
(cron + OIDC)         ──→    (Cloud Run)           ──→    (キュー)             ──→    (Cloud Run)
                              POST /dispatch/x             1テナント=1タスク           POST /api/v2/x
   "00:00 になった"              ↑                          ↑                          ↑
   "/dispatch/cuenote          Supabase から              個別リトライ               実際に外部API
    /delivery を叩け"          tenant 一覧を引く            並列実行                  (Amazon等) を叩く
                               テナント別にfanout         デッドライン管理            BQ に書き込む
```

**lunch-bot の場合 (1 段同期)**:

```
GitHub Actions cron        cmd/bot/main.go
(.github/workflows/...)    (recruit / announce 分岐)
        │
        └── go run ./cmd/bot recruit を直接叩く  ← これだけ
```

- **task-dispatcher** (= Scheduler から叩かれて Cloud Tasks にテナント別ジョブを積む係)
- **OIDC token** (= Google が発行する「俺は本物のサービスだ」と証明する身分証)
- **ファンアウト** = 1 つの cron 起動を、テナント数だけのジョブに展開すること
- **Supabase** (= テナント設定を保管する DB。lunch-bot で言う `config.go` を Web UI 付き DB にしたもの)

> **🎤 台本** (推定 120 秒)
>
> もう 1 つの山場、task-dispatcher と 4 段非同期の話。
>
> 上の絵、4 つの箱が並んでます。① タイマー、② 振り分け係、③ 待ち行列、④ 働く係。
>
> ① **Cloud Scheduler** が「00:00 になったから `/dispatch/cuenote/delivery` を叩け」と命令を出す。OIDC token、Google が発行する身分証を付けて叩きます。
>
> ② **task-dispatcher**。叩かれると **Supabase**、これはテナント設定を保管する DB、lunch-bot で言う config.go を Web UI 付き DB にしたもの、ここからテナント一覧を引く。「この platform を使う顧客は何社いるか」。50 社いたら 50 件分のジョブを ③ に積む。これが **ファンアウト**、1 入力を N 出力に展開すること。
>
> ③ **Cloud Tasks**、「あとでこの HTTP を叩いてね」を覚えておく **待ち行列**。1 タスクずつ、リトライしながら、並列で、デッドラインを管理しながら、④ を叩きにいきます。
>
> ④ **unified-api**、さっき見た本体。HTTP で叩かれて外部 API を叩き BigQuery に書く。
>
> 下の絵、**lunch-bot は 1 段**。cron が `cmd/bot` を直接叩いて終わり。途中に何も挟まらない。同期 1 段。
>
> Polaris は同じ「時間で起動する」を **4 段に分解しただけ**、これが見方の核。なぜ 4 段にしたか、答えは次のスライドで一撃で言います。

---

# スライド 13: なぜ Cloud Tasks を挟むのか (設計の核)

| 1 段同期だと困ること | 4 段非同期で解決 |
|---|---|
| 30 分かかる処理 → cron がタイムアウト | ② が ③ に積んだ瞬間に ② の仕事は終わる |
| 50 テナント中 3 つだけ Amazon 一時エラー → 全体失敗 | ③ が **タスク単位で個別リトライ**。残り 47 は成功 |
| 50 テナントを順次処理 → 遅い | ③ が **並列で ④ を叩く**。50 並列で爆速 |

> **間に Cloud Tasks (= 「あとでこの HTTP を叩いてね」と頼める待ち行列サービス) を挟む唯一の理由は、**
> **『個別リトライ・並列実行・失敗の局所化』をタダで手に入れるため。**

> **🎤 台本** (推定 60 秒)
>
> 60 秒で核を言います。
>
> **間に Cloud Tasks を挟む唯一の理由は『個別リトライ・並列実行・失敗の局所化』、この 3 つをタダで手に入れるためです**。これだけ。
>
> 50 テナントのうち 3 つだけ Amazon API がエラー、その 3 つだけリトライ。残り 47 は成功。30 分かかる処理でも cron は積んだ瞬間に終わる。50 件を並列で一気に叩く。lunch-bot は 1 段同期なのでこれが何ひとつ無い。
>
> 一撃の核、**「間に待ち行列を挟む = 各タスクを独立した運命にする」**。今日一番持ち帰ってほしい言葉です。

---

# スライド 14: dbt 補足 — BigQuery → staging → intermediate → mart

> **dbt**: BigQuery の中だけで SQL を順番に走らせてデータを加工するツール (SQL + YAML の別世界)

- unified-api が集めたデータは BigQuery の `Raw_*` テーブルに貯まる
- そこから先は **dbt** が引き継ぐ — Go の世界はここで終わり
- 4 段階で加工する:
  - **staging** (型を整える) → **intermediate** (組み立てる) → **dwh** (倉庫に置く) → **mart** (BI 用に出荷)
- 料理工程の「下ごしらえ → 仕込み → 保管 → 提供」と同じ流れ

> **🎤 台本** (推定 60 秒)
>
> ここは補足です。unified-api が集めたデータは BigQuery の Raw テーブルに貯まります。そこから先は dbt が引き継ぐ。dbt は SQL と YAML だけで動く別世界で、Go の世界はここで終わります。加工は 4 段階、staging で型を整えて、intermediate で組み立てて、dwh で倉庫に置いて、mart で BI 用に出荷する。料理の下ごしらえ・仕込み・保管・提供と同じです。今日は流れだけ押さえてください。

---

# スライド 15: 全体を 1 枚に圧縮 — lunch-bot ⇄ Polaris の通し図 ★

```
[lunch-bot]  (1段同期)

  GitHub Actions cron
        │
        ▼
   cmd/bot/main.go ── handler ── service ── repository ── Slack
                                                            │
                                                  データの行き先 = Slack 自体
                                                  (取って・変えて・戻すで完結)


       ↕  同じ骨格を「規模拡張 + 非同期化 + 加工層追加」したのが下


[Polaris]  (4段非同期 + 加工層)

  Cloud Scheduler ──▶ task-dispatcher ──▶ Cloud Tasks ──▶ unified-api ──▶ BigQuery ──▶ dbt ──▶ mart
   (タイマー)         (ファンアウト)         (待ち行列)      (handler →        (Raw_*)    (staging→      (BI出荷)
                       N テナント分に                          service →                  intermediate→
                       展開して積む                            repository)                dwh→mart)
```

- 上段 = lunch-bot、下段 = Polaris。**骨格は同じ 3 層 + DI**
- 違いは 4 つだけ:
  - ① 起動口の数 (cron 1 個 → Scheduler 多数)
  - ② 連携先の数 (Slack 1 個 → 17 種類)
  - ③ 書込先 (Slack 自体 → BigQuery)
  - ④ 1 cron が **N テナント分にファンアウト** されること

> **🎤 台本** (推定 90 秒)
>
> この絵が今日の答えです。上が lunch-bot、下が Polaris。上は cron が cmd/bot を起動して、handler から service、repository を経て Slack を叩く。行き先は Slack 自体、これで完結。下は Cloud Scheduler が task-dispatcher を叩き、task-dispatcher が Cloud Tasks にテナント数分のジョブを積み、Cloud Tasks が unified-api を非同期で叩く。unified-api の中は同じ 3 層、書込先は BigQuery、その先は dbt が staging から mart まで加工する。上下を矢印で繋いでいるのは、同じ骨格を規模拡張・非同期化・加工層追加したのが Polaris だからです。違いは 4 つだけ、起動口の数・連携先の数・書込先・テナント分のファンアウト、それだけ。

---

# スライド 16: できるようになったこと

- lunch-bot の **3 層 (handler / service / repository)** を自分の言葉で説明できる
  - 「handler は受付、service は調理場、repository は仕入れ係」と即答できる
- unified-api の `cmd/api/routes.go` を読んで **DI の組立順** を追える
  - 「下から順に組んで chi で URL に紐付ける」流れが見える
- task-dispatcher の `cmd/server/main.go` を見て **「Scheduler から Cloud Tasks に積む」** 流れを説明できる
- **「ジョブ呼び出し」と「API エンドポイント呼び出し」を区別** して話せる
  - ジョブ呼び出し = Cloud Tasks 経由の非同期 HTTP / API エンドポイント = unified-api 側の `/api/v2/...`
- Go コードを読んで **「何が起きるか」を頭で再生** できるようになった

> **🎤 台本** (推定 60 秒)
>
> できるようになったことです。1 つ目、lunch-bot の 3 層を自分の言葉で説明できる、handler は受付、service は調理場、repository は仕入れ係。2 つ目、unified-api の routes.go を読んで DI の組立順を追える。3 つ目、task-dispatcher の main.go から Scheduler が Cloud Tasks に積む流れを説明できる。4 つ目、ジョブ呼び出しと API エンドポイント呼び出しを区別して話せる。Go コードを読んで何が起きるかを頭で再生できる、ここまで来ました。

---

# スライド 17: まだ課題なこと (正直に)

- **dbt の中身**: SQL マクロは流れだけ理解。書ける段階ではない
- **Cloud (GCP) の細部**: IAM / OIDC / Terraform は表面しか触れていない
- **エラー処理の運用設計**: `fmt.Errorf("...: %w", err)` で wrap しているが、本番でどうログ集約しているかは未調査
- **テスト**: lunch-bot は `shuffler_test.go` と `cmd/bot/main_test.go` の 2 ファイルしかない。service / handler / repository 本体のテストは未着手。Polaris のテストパターンはこれから読む
- **次に読むのは**: Polaris の **テスト** + **Cloud Tasks 経由のジョブ実行ログ**

> **🎤 台本** (推定 60 秒)
>
> 課題も正直に話します。1 つ目、dbt の中身、SQL マクロは流れだけで書ける段階ではない。2 つ目、GCP の細部、IAM や OIDC や Terraform は表面しか触れていない。3 つ目、エラー処理の運用設計、コードでは wrap していますが、本番のログ集約は未調査です。4 つ目、テスト、lunch-bot にあるのは shuffler_test.go と main_test.go の 2 つだけで、service・handler・repository 本体のテストは書けていません。次は Polaris のテストと Cloud Tasks 経由のジョブ実行ログを読みます。

---

# スライド 18: 締め

> **Polaris の unified-api は、lunch-bot と全く同じ 3 層 + DI で作られている。**
>
> **違うのは、**
> **① 起動口の数、② 連携先の数、③ 書込先 (BQ)、④ 1 cron が N テナント分にファンアウトされること、それだけ。**

- 自作の小さなアプリを骨の髄まで読むと、業務の大きなコードも同じ目で読める
- 質問はチャットでも OK です
- ありがとうございました

> **🎤 台本** (推定 30 秒)
>
> 持ち帰る 1 行です。Polaris の unified-api は lunch-bot と同じ 3 層 + DI で作られている。違うのは、起動口の数、連携先の数、書込先が BigQuery、N テナント分にファンアウトされること、それだけです。ありがとうございました。

---
