# notes/11_polaris_unified_api.md — `ingestion/unified-api/` 深掘り

5/15 発表の **本編の中身**。「3層 + DI」を Polaris のコードでどう実現しているかを、lunch-bot と並べて見る。

---

## ◆ unified-api は何屋か (1行)

> **「外部API (Amazon / 楽天 / 広告) から HTTP/CLI で起動されてデータを集めてきて BigQuery に書き込む Go 製サーバ + バッチ」**

つまり **lunch-bot のフルスタック (handler + service + repository) を、外部API向けに作り直したもの**。違いは:
1. 入口が **HTTP と CLI の 2 系統**
2. **連携先が17種類** (Slack だけじゃない)
3. **書き込み先が BigQuery**

---

## ◆ ディレクトリ構造 (深掘り対象)

```
ingestion/unified-api/
├── cmd/                          ★ 入口 (起動点)
│   ├── api/                      ─→ HTTP サーバ用 (Cloud Run 常駐)
│   │   ├── main.go               ─→ chi ルータを作って http.Server を起動
│   │   └── routes.go             ─→ ★ DI 組立場所 (=cmd/bot/main.go と同じ役割)
│   ├── jobs/                     ─→ CLI バッチ用 (Cloud Run Jobs)
│   │   └── main.go               ─→ -platform / -source / -tenant フラグで service を直叩き
│   └── scraping/                 ─→ スクレイピング系 CLI (yamato/japanpost等)
│
└── internal/
    ├── handler/                  ★ 入口層 (約20ファイル, プラットフォーム別)
    │   ├── handler.go            ─→ 共通: decode/validate/error整形
    │   ├── ecforce.go            ─→ Ecforce 用 handler
    │   ├── rakuten.go            ─→ 楽天用 handler
    │   ├── amazon_*.go           ─→ Amazon 用 handler (複数)
    │   └── ...
    │
    ├── service/                  ★ 業務ロジック層 (プラットフォーム別パッケージ)
    │   ├── ecforce/
    │   │   ├── interface.go      ─→ EcforceService + EcforceHTTPRepository の interface
    │   │   ├── service.go        ─→ DefaultEcforceService の実装
    │   │   └── types.go          ─→ レスポンス型
    │   ├── etl/
    │   │   └── runner.go         ─→ 共通の ETL オーケストレータ (各 service が使う部品)
    │   └── ...19パッケージ
    │
    ├── jobs/                     ★ ジョブ専用 runner (handler を経由しない経路)
    │   ├── ecforce/runner.go     ─→ Run(ctx, source, tenantID, targetDate) 共通シグネチャ
    │   └── ...10パッケージ
    │
    ├── repository/               ★ 外部連携層 (連携先別)
    │   ├── ecforce/              ─→ Ecforce HTTP クライアント
    │   ├── slack/
    │   │   ├── interface.go      ─→ Repository / UserChecker interface
    │   │   └── bot_client.go     ─→ BotRepository (lunch-bot の SlackClient とほぼ同じ骨格!)
    │   ├── amazon/               ─→ SP-API 等
    │   ├── loader/
    │   │   └── bq.go             ─→ BigQuery への書き込み (Inserter().Put())
    │   └── ...17ディレクトリ
    │
    └── infrastructure/
        └── bigquery/client.go    ─→ BigQuery client (低レイヤラッパ)
```

★ が今日の主戦場。`cmd/api/routes.go` → `internal/handler/` → `internal/service/` → `internal/repository/` の流れが lunch-bot と完全に対応する。

---

## ◆ 3層 + DI を lunch-bot と並べて見る

### 1. Repository 層 — 外部APIを叩くやつ

#### lunch-bot

```go
// internal/repository/slack_client.go (抜粋)
type SlackClient struct {
    token string
    http  *http.Client
}

func NewSlackClient(token string) *SlackClient {
    return &SlackClient{
        token: token,
        http:  &http.Client{Timeout: 10 * time.Second},
    }
}

func (c *SlackClient) PostMessage(channel, text string) (string, error) {
    // chat.postMessage を叩く
}
```

#### Polaris (Slack)

```go
// ingestion/unified-api/internal/repository/slack/bot_client.go (抜粋)
type BotRepository struct {
    httpClient *http.Client
    config     Config
}

func NewBotRepository(config Config) *BotRepository {
    return &BotRepository{
        httpClient: &http.Client{Timeout: 30 * time.Second},
        config:     config,
    }
}

func (r *BotRepository) PostMessage(ctx context.Context, req PostMessageRequest) (*PostMessageResponse, error) {
    // 同じく chat.postMessage を叩く
}
```

> **ほぼコピペレベルで同じ骨格**。違いは `ctx context.Context` を渡しているのと、interface を別ファイルに切り出していること。

#### Polaris の追加ポイント: interface を別ファイルに切る

```go
// ingestion/unified-api/internal/repository/slack/interface.go
type Repository interface {
    PostMessage(ctx context.Context, req PostMessageRequest) (*PostMessageResponse, error)
}
type UserChecker interface {
    UserExists(ctx context.Context, userID string) (bool, error)
}
```

> lunch-bot は service 側に `SlackRepository` interface を置いていた。Polaris は **repository 側のパッケージに interface を置く** スタイル。
> どっちも DI のためのもの。Polaris の方が「repository パッケージを開けば必要な操作が一覧できる」という見通しの良さがある。

---

### 2. Service 層 — 業務ロジック

#### lunch-bot

```go
// internal/service/lunch_service.go
type SlackRepository interface { /* repository が満たすべき5メソッド */ }

type LunchService struct {
    slack         SlackRepository
    channelID     string
    emoji         string
    lookbackHours int
}

func NewLunchService(slack SlackRepository, channelID string) *LunchService {
    return &LunchService{slack: slack, channelID: channelID, emoji: "bento", lookbackHours: 26}
}

func (s *LunchService) RunRecruit() error {
    ts, err := s.slack.PostMessage(s.channelID, recruitmentText)
    if err != nil {
        return fmt.Errorf("post recruitment: %w", err)
    }
    // ...
}
```

#### Polaris (Ecforce)

```go
// ingestion/unified-api/internal/service/ecforce/service.go
type DefaultEcforceService struct {
    httpRepo  EcforceHTTPRepository  // ← 外部APIクライアント
    bqRepo    *loader.BqRepository   // ← BigQuery 書込
    mdmRepo   mdm.Repository         // ← Supabase からテナント情報
    etlRunner *etl.Runner            // ← 共通 ETL オーケストレータ
}

var _ EcforceService = (*DefaultEcforceService)(nil)  // 「俺はこの interface を満たしてるぜ」のコンパイル時アサート

func NewService(httpRepo EcforceHTTPRepository, bqRepo *loader.BqRepository,
    mdmRepo mdm.Repository, etlRunner *etl.Runner) *DefaultEcforceService {
    return &DefaultEcforceService{httpRepo: httpRepo, bqRepo: bqRepo, mdmRepo: mdmRepo, etlRunner: etlRunner}
}

func (s *DefaultEcforceService) PostCustomers(ctx context.Context, tenant, targetDate string) (string, error) {
    items, err := s.httpRepo.GetCustomers(ctx, siteURL, targetDate, apiKey)
    if err != nil {
        return "", fmt.Errorf("fetch customers: %w", err)
    }
    // BigQuery に書き込み...
}
```

#### 並べてみると

| 観点 | lunch-bot | Polaris (Ecforce) |
|---|---|---|
| struct + フィールド | ある | ある |
| interface で repository を受ける | `SlackRepository` | `EcforceHTTPRepository` |
| `NewXxx` コンストラクタ | ある | ある |
| 依存の数 | 1 個 (slack) | 4 個 (httpRepo / bqRepo / mdmRepo / etlRunner) |
| `fmt.Errorf("...: %w", err)` で wrap | ある | ある |
| `var _ Interface = (*Type)(nil)` のアサート | なし | **ある** (interface 充足の見える化) |

> **設計の骨格はそっくり同じ**。違うのは依存の数だけ (lunch-bot は Slackだけ、Polaris は 外部API + BQ + Supabase + ETL Runner)。

---

### 3. Handler 層 — 入口

#### lunch-bot (CLI 入口)

```go
// internal/handler/lunch_handler.go (全体)
type LunchHandler struct {
    svc *service.LunchService
}

func NewLunchHandler(svc *service.LunchService) *LunchHandler {
    return &LunchHandler{svc: svc}
}

func (h *LunchHandler) Recruit() error  { return h.svc.RunRecruit() }
func (h *LunchHandler) Announce() error { return h.svc.RunAnnounce() }
```

> **超薄い**。「サブコマンド名 → service の対応メソッド呼び出し」しかない。

#### Polaris (HTTP 入口)

```go
// ingestion/unified-api/internal/handler/ecforce.go (要点抜粋)
func (h *EcforceHandler) PostCustomers(w http.ResponseWriter, r *http.Request) {
    resp := NewResponder(w)
    var req EcforceRequest
    if err := decodeRequest(w, r, &req); err != nil { return }
    if err := validateStruct(&req); err != nil { return }

    ctx := logger.ContextWithAttrs(r.Context(), "platform", "ecforce", "tenant_id", req.Tenant)
    result, err := h.service.PostCustomers(ctx, req.Tenant, req.TargetDate)  // ← service 呼び出し
    if err != nil {
        h.notifyError("customers", req.Tenant, err)
        resp.Error(h.toErrorResponse(err)); return
    }
    h.notifySuccess("customers", req.Tenant)
    resp.SuccessWithData(http.StatusOK, result, TenantResponseBodyData{Tenant: req.Tenant})
}
```

#### 並べてみると

| 観点 | lunch-bot | Polaris |
|---|---|---|
| handler の役割 | サブコマンド分岐 → service 呼び出し | HTTP 受信 → decode/validate → service 呼び出し → JSON レスポンス |
| 入力形式 | 引数なし (process 引数) | `*http.Request` から JSON デコード |
| 出力形式 | `error` を返す (cmd/bot/main.go が log.Fatal) | `*http.ResponseWriter` に JSON 書込 |
| ロジック | 1 行 (service 呼ぶだけ) | 6 工程 (decode / validate / ctx 加工 / service / Slack通知 / レスポンス) |
| 共通化 | なし | `decodeRequest` / `validateStruct` / `Responder` で共通化 |

> **handler の本質的な役割は同じ**: 入口の形式を吸収して service に橋渡しするだけ。
> Polaris は HTTP の作法 (decode / validate / status 変換 / 通知) が増えた分だけ厚くなっているが、**「service を呼ぶ」のは 1 行**で同じ。

---

## ◆ DI の組み立て方 — `routes.go` の役割

ここが今日の **「DIってこれか!」を体感できるポイント**。

### lunch-bot の DI 組立 (`cmd/bot/main.go`)

```go
slack := repository.NewSlackClient(cfg.SlackToken)             // 1. 一番下を作る
svc   := service.NewLunchService(slack, cfg.ChannelID)         // 2. それを渡して service を作る
h     := handler.NewLunchHandler(svc)                          // 3. それを渡して handler を作る

// あとはサブコマンドで分岐
switch os.Args[1] {
case "recruit":  err = h.Recruit()
case "announce": err = h.Announce()
}
```

> **「下から順に組み上げる」だけ**。これが DI (Dependency Injection)。

### Polaris の DI 組立 (`cmd/api/routes.go`)

```go
// 共通リソースは setupRouter() で 1 回だけ作る
bqRepo       := loader.NewBqRepository(...)
mdmRepo      := mdm.NewRepository(...)
slackService := slackservice.NewAsyncService(...)

// プラットフォームごとに register 関数を呼ぶ
registerEcforceRoutes(r, mdmRepo, bqRepo, slackService)
registerRakutenRoutes(r, mdmRepo, bqRepo, slackService)
// ... 20回くらい

// register 関数の中身 (Ecforce の例)
func registerEcforceRoutes(r chi.Router, mdmRepo mdm.Repository, bqRepo *loader.BqRepository, slackService slackservice.AsyncService) {
    httpRepo := ecfrepo.NewHTTPRepository(...)                        // 1. repository を作る
    etlRunner := etl.NewRunner(...)                                   // 2. 部品も作る
    svc := ecfsvc.NewService(httpRepo, bqRepo, mdmRepo, etlRunner)    // 3. service を作る
    eh := handler.NewEcforceHandler(svc, slackService)                // 4. handler を作る
    r.Route("/ecforce", func(r chi.Router) {                          // 5. URL に紐付ける
        r.Post("/customers", eh.PostCustomers)
        r.Post("/orders", eh.PostOrders)
    })
}
```

> **lunch-bot の `cmd/bot/main.go` と完全に同じ「下から順に組み上げる」パターン**。違いは:
> - 共通リソース (BigQuery / Supabase / Slack通知) を **1 回だけ** 作って使い回す
> - プラットフォームごとに register 関数に分けて、20 ファイル作っても見通しが効く
> - 最後に **chi の `r.Post(...)` で URL に紐付ける** (lunch-bot の `switch os.Args[1]` の代わり)

---

## ◆ ジョブ呼び出し vs API エンドポイント呼び出し — 同じ service を別の入口から呼ぶ

unified-api の **一番大事な設計判断** がここ。

### 2 つの経路

```
┌─ API エンドポイント経由 ─────────────────────────────────────┐
│                                                              │
│  Cloud Tasks ──HTTP POST /api/v2/ecforce/customers──>        │
│                                                              │
│        cmd/api/main.go (chi router)                          │
│              │                                               │
│              ▼                                               │
│        internal/handler/ecforce.go                           │
│              │                                               │
│              ▼                                               │
│        internal/service/ecforce/service.go ← ★ ここで合流   │
│                                                              │
└──────────────────────────────────────────────────────────────┘
                          ▲
                          │ (同じ service を共有)
                          ▼
┌─ ジョブ経由 ────────────────────────────────────────────────┐
│                                                              │
│  Cloud Run Jobs ──CLI 起動──> cmd/jobs/main.go               │
│                                  │                           │
│                                  ▼                           │
│                          internal/jobs/ecforce/runner.go     │
│                                  │ (handler を経由しない)    │
│                                  ▼                           │
│                          internal/service/ecforce/service.go │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### 何が同じで何が違うか

|  | API エンドポイント | ジョブ |
|---|---|---|
| 起動の仕方 | HTTP リクエスト | CLI フラグ (`-platform ecforce -source customers`) |
| 入口 | `cmd/api/main.go` (常駐サーバ) | `cmd/jobs/main.go` (1回走って終わる) |
| handler を通る | 通る | **通らない** (`internal/jobs/<platform>/runner.go` が直接 service を叩く) |
| 通知の場所 | handler 側で `notifyError/notifySuccess` | runner 側で `NotifySuccess/NotifyError` |
| デプロイ先 | Cloud Run (Service) | Cloud Run **Jobs** |
| 中身の service | **同じ** (`ecfsvc.EcforceService`) | **同じ** (`ecfsvc.EcforceService`) |

### なぜ2系統あるのか (一言)

> **「短時間で終わるものは API エンドポイント経由 (Cloud Tasks 経由)、長時間かかるものは Cloud Run Jobs (CLI) 経由」と使い分けている。**

- API エンドポイント = Cloud Run の HTTP タイムアウト (30分) 以内に終わるもの
- ジョブ = それを超える可能性があるもの (Cuenote の大量データなど)

両方とも **service の中身は変えずに使い回せる**のが、interface + DI の威力。

---

## ◆ 押さえる核 (発表で言うべきこと)

> **「Polaris の unified-api は、lunch-bot と全く同じ 3 層 + DI で作られている。違うのは、起動口が HTTP と CLI の 2 系統あること、連携先が17種類あること、書き込み先が BigQuery であること、それだけ。」**

> **「同じ service を、HTTP からも CLI からも呼べる。これが interface を切ったことで実現できる『実際の使い回し』。lunch-bot で言えば、本物の Slack と fake の Slack を差し替えられたのと、本質的に同じこと。」**

---

## ◆ 次に読むファイル

- `notes/12_polaris_task_dispatcher.md` — この HTTP を「いつ・どこに」叩くかを決める dispatcher の話
