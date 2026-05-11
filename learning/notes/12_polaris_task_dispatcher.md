# notes/12_polaris_task_dispatcher.md — `orchestration/task-dispatcher/` 深掘り

5/15 発表の **本編後半**。「ジョブ呼び出し」と「Task Dispatcher の役割」を 1〜2 分で説明できるレベルまで噛み砕く。

---

## ◆ task-dispatcher は何屋か (1行)

> **「時間が来たら、Supabase からテナント一覧を引いて、テナント別のジョブを Cloud Tasks に積む係」**

つまり **lunch-bot の GitHub Actions cron を、4段非同期に拡張した版**。

---

## ◆ 4段の絵 (発表のメインスライド素材)

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

**lunch-bot で言うと**:

```
GitHub Actions cron        cmd/bot/main.go
(.github/workflows/...)    (recruit / announce 分岐)        ←── これを 4 段に分けたのが Polaris
        │
        └── go run ./cmd/bot recruit を直接叩く
```

lunch-bot は **「① タイマーが直接 ④ 働く係を叩く」1段同期**。
Polaris は同じことを **「① タイマー → ② 振り分け → ③ 待ち行列 → ④ 働く」4段非同期**にしている。

---

## ◆ なぜ 4 段に分けたか — 設計の核

```
1段同期だと困ること                   4段非同期で解決
─────────────────────              ─────────────────────
30分かかる処理 → cron が             ② が ③ にタスクを積んだ瞬間
タイムアウトする                     ② の仕事は終わり (処理は ④ で)

50テナント中 3 つだけ Amazon         ③ が「タスク単位で個別リトライ」
API が一時エラー → 全体失敗          残り 47 は普通に成功

50テナントを順次処理 → 遅い          ③ が「並列で ④ を叩く」
                                     50 並列で爆速

cron 設定変更が cron yaml と         ② のコードは時間に関心ない
コードに分散                         (① だけ知ってる)
```

> **「間に Cloud Tasks を挟む唯一の理由は『個別リトライ・並列実行・失敗の局所化』」**。これが発表で言うべき核。

---

## ◆ ① Cloud Scheduler の中身 (1個だけ抜粋)

`terraform/environments/production/cuenote.tf`:

```hcl
resource "google_cloud_scheduler_job" "cuenote_delivery_dispatch" {
  name     = "cuenote-delivery-dispatch"
  schedule = "0 0 * * *"           # 毎日 00:00 に発火
  time_zone = "Asia/Tokyo"
  http_target {
    http_method = "POST"
    uri         = "${var.task_dispatcher_url}/dispatch/cuenote/delivery"
    oidc_token { service_account_email = google_service_account.scheduler.email }
  }
  retry_config { retry_count = 3 }
}
```

**ポイント**:
- HTTP body は **空**
- 「いつ叩くか」と「どこを叩くか」しか知らない
- OIDC token で認証 (= Google が発行する身分証で「俺は本物の Scheduler だぜ」と証明)

> lunch-bot の `.github/workflows/recruit.yml` と発想は同じ。違いは認証の仕組みと、叩き先が「task-dispatcher」という中継地点であること。

---

## ◆ ② task-dispatcher 本体の役割

### エントリーポイント

`/Users/imutaakihiro/repos/androots/polaris/orchestration/task-dispatcher/cmd/server/main.go`

main.go がやることを 10 行で:
1. `.env` から `PROJECT_ID/QUEUE/SERVICE_URL/OIDC_SERVICE_ACCOUNT` を読む
2. Supabase クライアントを作る (mdm + mdm_new の 2 系統)
3. Cloud Tasks クライアントを作る
4. handler を作る (DI: TaskService / RunJobsService / BigQueryService を注入)
5. chi ルータに 22 個の `/dispatch/<platform>` をマウント
6. webhook (dbt Cloud completion 用) も2本マウント
7. `:8081` で listen

> **「main.go の構造は unified-api と全く同じ」**: 環境変数読込 → 依存生成 → handler 作る → ルータに紐付け → 起動。

### handler の中で何をしてるか

例: `/dispatch/ecforce` を受けたら:
1. **Supabase からテナント一覧を引く** (`mdmRepo.GetTenantsByPlatform("ecforce")`)
2. テナント分ループして、テナントごとに Cloud Tasks にタスクを 1 個積む
3. 1 タスク = 「unified-api の `/api/v2/ecforce/customers` を、tenant=X で叩け」というジョブ

### Cloud Tasks にタスクを積むコア (`internal/service/task_service.go`)

```go
task := &cloudtaskspb.Task{
    MessageType: &cloudtaskspb.Task_HttpRequest{
        HttpRequest: &cloudtaskspb.HttpRequest{
            HttpMethod: convertHTTPMethod(req.Method),
            Url:        url,        // serviceURL + endpoint = unified-api の URL
            Headers:    headers,
            Body:       body,       // {"tenant": "X", "targetDate": "..."}
        },
    },
    Name: fmt.Sprintf("%s/tasks/%s", s.queuePath, taskName),
}

// 自分が unified-api を叩く時の身分証 (OIDC token) を埋め込む
if s.config.OIDCServiceAccount != "" {
    task.GetHttpRequest().AuthorizationHeader = &cloudtaskspb.HttpRequest_OidcToken{
        OidcToken: &cloudtaskspb.OidcToken{
            ServiceAccountEmail: s.config.OIDCServiceAccount,
            Audience:            audience,
        },
    }
}
task.DispatchDeadline = durationpb.New(30 * time.Minute)  // 30分超えたら諦めろ
s.client.CreateTask(ctx, &cloudtaskspb.CreateTaskRequest{Parent: s.queuePath, Task: task})
```

> **やっていることは「`http.Request` を Cloud Tasks に渡してるだけ」**。
> 違いは「今すぐ叩く」のではなく「あとで Cloud Tasks が代わりに叩いてくれる」だけ。

---

## ◆ ジョブ呼び出し vs API エンドポイント呼び出し — 発表で必ず聞かれる用語

unified-api と task-dispatcher の文脈で、この用語は **2 階層で混ざる** ので注意。

### 階層1: 「ジョブ」が Cloud Tasks に積まれる HTTP のこと

これが **task-dispatcher の文脈での「ジョブ」** で、一番よく言う意味。

```
task-dispatcher  ──→  Cloud Tasks に「タスク」(= ジョブ) を積む  ──→  unified-api の API エンドポイントを叩く
```

つまり:
- **「ジョブ呼び出し」 = Cloud Tasks 経由の非同期 HTTP 呼び出し**
- **「API エンドポイント呼び出し」 = 上記の `unified-api` 側のエンドポイント (`/api/v2/ecforce/customers` 等)**

→ **ペアの関係**。ジョブ呼び出しが「いつ・誰の・どのデータを」叩くかを決め、API エンドポイントが実際に叩かれて働く。

### 階層2: 「Cloud Run Jobs (バッチ)」のこと

`cmd/jobs/main.go` のこと。これは **Cloud Run の "Jobs" モード**(コンテナを 1 回 CLI 起動して終了)で動く。

例: Cuenote だけはこちら。task-dispatcher の handler が:

```go
runJobsService.RunJob(ctx, "cuenote-etl-job", []string{"-platform","cuenote","-source","delivery","-tenant",tenantID})
```

を呼ぶ。Cloud Tasks は使わない、Cloud Run Jobs を直接 ExecuteJob で起動する。

→ **長時間バッチ専用の脱出口**。30分タイムアウトを超える可能性がある時だけここを使う。

### まとめ表

| 言葉 | 何のこと | 起動方法 | 入口 |
|---|---|---|---|
| **API エンドポイント呼び出し** | `unified-api` の HTTP エンドポイント (`/api/v2/...`) を叩くこと | HTTP POST | `cmd/api/main.go` (Cloud Run 常駐) |
| **ジョブ呼び出し** (狭義) | Cloud Tasks 経由で上記を非同期に叩くこと | Cloud Tasks → HTTP | 同上 |
| **Cloud Run Jobs** | `unified-api/cmd/jobs/main.go` を CLI 起動 | gcloud run jobs execute | `cmd/jobs/main.go` |

---

## ◆ docs/ にある実例 (5 パターン)

`/Users/imutaakihiro/repos/androots/polaris/orchestration/task-dispatcher/docs/`:

| パターン | やってること |
|---|---|
| **assist-alignment** | Assist (顧客マッチング) の典型フロー。Scheduler → dispatcher → Tasks → unified-api |
| **rakuten-inventory** | 楽天在庫を 01:00 JST に取得、テナント並列 |
| **delivery-status** | 配送ステータス取得 → dbt → 長期不在SMS、3段スケジューラ chain |
| **cuenote** | 6パターンのスケジューラ + dbt Webhook で chain dispatch (一番複雑) |
| **implementation_logs** | 日付別の作業ログ (UUID化、Cloud Run Jobs 移行など) |

> **「同じ4段パターンを各プラットフォームに当てはめて並列運用している」**のが Polaris の運用の姿。

---

## ◆ lunch-bot の cron との比較 (発表で言うべきこと)

| 観点 | lunch-bot | Polaris |
|---|---|---|
| 時間トリガー | GitHub Actions cron | Cloud Scheduler |
| 認証 | GitHub Actions Secret (Slack token) | OIDC token (Google が発行) |
| 起動方法 | Goバイナリを直接走らせる | HTTP POST で task-dispatcher を叩く |
| 同期 / 非同期 | **同期 1段** (cron が完了まで待つ) | **非同期 4段** (cron は積むだけで終わり) |
| リトライ | ワークフロー全体を再実行 | 1 タスク単位で個別リトライ |
| 並列 | なし | テナント数だけ並列 |
| タイムアウト | GitHub Actions の上限 (デフォルト 6時間) | 30分 / タスク |
| 失敗の局所化 | なし (1 つコケたら全部失敗扱い) | あり (1 タスクだけ失敗) |

---

## ◆ 押さえる核 (発表で言うべき1分要約)

> **「lunch-bot は GitHub Actions cron が直接 Go バイナリを叩いて Slack に投げる『1段同期』。**
> **Polaris の task-dispatcher は、同じ cron 起動を『4段非同期』に拡張したもの。**
>
> **Cloud Scheduler が時間で叩き、task-dispatcher が Supabase からテナント一覧を引いてファンアウトし、Cloud Tasks に1テナント1ジョブとして積み、Cloud Tasks が retry/deadline 付きで unified-api を呼ぶ。**
>
> **間に Cloud Tasks を挟む唯一の理由は『個別リトライ・並列実行・失敗の局所化』。lunch-bot で言えば『参加者ごとに個別リトライ可能なシャッフルキュー』を挟んだ版。」**

---

## ◆ 次に読むファイル

- `notes/13_polaris_dbt.md` — BigQuery に貯まったデータをここから先で加工する話 (補足)
