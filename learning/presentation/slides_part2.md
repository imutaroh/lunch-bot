# slides_part2.md — スライド 8〜13 (発表の主戦場)

このパートは 5/15 発表 (20分) の **重み 8 の本丸**。
合計 6 枚 / 推定 720 秒 = 12 分。

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
