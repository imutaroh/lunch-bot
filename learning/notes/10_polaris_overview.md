# notes/10_polaris_overview.md — Polaris 全体地図 + lunch-bot 対応表

5/15 発表の **本編の核**。これだけ覚えれば、自作アプリ ↔ Polaris の対応づけが言える。

---

## ◆ Polaris は何屋か (1行)

> **「Amazon・楽天・広告API などから毎日データを集めてきて、BigQueryに貯めて、分析しやすい形に整える工場」**

つまりやっていることは大きな ETL (Extract / Transform / Load)。
lunch-bot がやっていた「Slackから取って・グループ分けして・Slackに書き戻す」と **やっていることの種類は同じ**。違うのは **規模 + 中継地点の数 + 起動の仕組み**。

---

## ◆ 工場の流れ (1枚絵)

```
[外部のサービス]                  [Polaris の役割]                    [出口]

Amazon / 楽天 / 広告API   ──→   ingestion/ (集めてくる)         ──→   BigQuery
                                                                     (生データ置き場 = staging)
                                                                       │
                                  transformation/ (dbt で加工)   ──→   BigQuery
                                                                     (intermediate → dwh → mart)
                                                                       │
                                  application/ (使う側)          ──→   人間 / 他のサービス
```

**lunch-bot で言うと**:

```
Slack (絵文字リアクション)  ──→  internal/service (集計)  ──→  Slack (発表投稿)
       ↑ ここが ingestion 相当            ↑ ここは transformation 相当だが        ↑ Polaris で言うと
                                          lunch-bot は service 内で完結              application 相当
```

つまり lunch-bot は **「ingestion + transformation + application を 1 個のアプリに圧縮」した小さな工場**。Polaris は **同じ役割をフォルダ単位で分業**している。

---

## ◆ Polaris のフォルダ全体像

| フォルダ | 役割 | 発表での扱い |
|---|---|---|
| **`ingestion/`** | データ収集 (Cloud Functions / Cloud Run) | **本編** |
| ├ `unified-api/` (Go) | 新世代の統合 ETL システム ★ | **本編の主役** |
| ├ `amazon-*/` 等 | Python の旧世代 ingestion (Cloud Functions) | 触れない |
| **`orchestration/`** | 実行制御・パイプライン管理 | **本編** |
| ├ `task-dispatcher/` (Go) | Cloud Scheduler から受けて Cloud Tasks に積むディスパッチャ ★ | **本編** |
| ├ `cloud-workflow/` | Cloud Workflows + Scheduler 定義 (YAML) | 触れない |
| **`transformation/androots/`** | dbt によるデータ加工 (SQL + YAML) | **補足のみ** |
| `application/` | 使う側 (Streamlit / Next.js MDM 等) | 触れない |
| `transfer/` | 加工後データを外部に書き戻す | 触れない |
| `analytics/` | モニタリング・通知 | 触れない |

★ = ご主人様が深掘りすべき2フォルダ。`unified-api` (中身を作る方) と `task-dispatcher` (時間で叩く方) のペアが本編の核。

---

## ◆ lunch-bot ↔ Polaris 対応表 (発表のメインスライド素材)

| 観点 | lunch-bot (自作) | Polaris (実務) |
|---|---|---|
| **起動の引き金** | GitHub Actions の cron (週月09:00 / 水08:30) | Cloud Scheduler (cron) |
| **起動コマンド** | `go run ./cmd/bot recruit` を直接叩く | `POST /dispatch/<platform>` (HTTP) を叩く |
| **起動を受ける入口** | `cmd/bot/main.go` のサブコマンド分岐 | `orchestration/task-dispatcher/cmd/server/main.go` |
| **入口層 (handler)** | `internal/handler/lunch_handler.go` | `ingestion/unified-api/internal/handler/*.go` (約20ファイル) |
| **業務ロジック (service)** | `internal/service/lunch_service.go` | `ingestion/unified-api/internal/service/<platform>/service.go` |
| **外部連携 (repository)** | `internal/repository/slack_client.go` (Slack のみ) | `ingestion/unified-api/internal/repository/<platform>/` (17 種類) |
| **interface でモック差替え** | `SlackRepository` interface (`cmd/simulate` で fake に差し替え) | `slack/interface.go` 等で同じパターン (Polaris では必要な所だけ切る) |
| **DI の組立場所** | `cmd/bot/main.go` で `NewSlackClient → NewLunchService → NewLunchHandler` を順に組む | `cmd/api/routes.go` の `registerXxxRoutes()` 関数群が同じ役割 |
| **データ保存先** | なし (Slack 自体が Source of Truth) | BigQuery (`internal/repository/loader/bq.go` がストリーミング挿入) |
| **後段の加工** | service内で in-memory で処理して終わり | dbt が staging → intermediate → dwh → mart で段階加工 |
| **エラー処理** | `fmt.Errorf("...: %w", err)` で wrap | 同じ。全層で統一 |
| **実行の単位** | 1 cron = 1 サブコマンド = 1 同期処理 | 1 cron = N テナント分のジョブを Cloud Tasks にファンアウト = 並列・個別リトライ |

---

## ◆ ここで一撃で押さえる核

> **lunch-bot は Polaris の「unified-api ＋ task-dispatcher」の縮小版。**
>
> - lunch-bot の handler / service / repository = unified-api の handler / service / repository (フォルダ構造そのまま)
> - lunch-bot の GitHub Actions cron = Polaris の **Cloud Scheduler → task-dispatcher → Cloud Tasks → unified-api** (同じ役割を4段非同期に拡張したもの)
> - lunch-bot で「Slack 経由でのみデータが流れる」ところが、Polaris では「BigQuery + dbt」でデータが貯まり加工される

---

## ◆ 用語の最低限の地図 (発表で迷わないため)

| 言葉 | 一言で |
|---|---|
| **Cloud Scheduler** | 時間で HTTP を叩いてくれるタイマー。lunch-bot で言う GitHub Actions の cron |
| **Cloud Run** | Goのバイナリを 1 個ずつコンテナで起動するサービス。HTTPを受けるサーバを置く場所 |
| **Cloud Run Jobs** | Cloud Run のバッチ版 (HTTPなし、CLIで1回走って終わる) |
| **Cloud Tasks** | 「あとでこのHTTPを叩いてね」キュー。リトライ・並列実行・遅延を任せられる |
| **task-dispatcher** | Scheduler が叩いてきた時に、Supabase からテナント一覧を引いて Cloud Tasks に「テナント別ジョブ」を積む係 |
| **unified-api** | task-dispatcher (or 直接) から HTTP を受けて、外部APIを叩いて BigQuery に書き込む本体 |
| **Supabase** | テナント設定を保管する DB。lunch-bot で言う `config.go` を Web UI 付き DB にしたもの |
| **BigQuery** | 集めたデータを置く倉庫 (DWH = データウェアハウス) |
| **dbt** | BigQuery の中だけで SQL を順番に走らせて加工するツール (Goとは別) |
| **staging / intermediate / dwh / mart** | dbt の中の4段階。生 → 整形 → 統合 → 分析用 |

---

## ◆ 発表でよく聞かれそうな質問 (即答練習用)

### Q1. なんで Polaris は unified-api と task-dispatcher を 2 個に分けてるの？
> **「叩く時間を決める係」と「叩かれて働く係」を分けるため**。
> task-dispatcher は時間とテナント一覧だけを知ってる。unified-api は外部APIの叩き方だけを知ってる。役割を分けると、片方だけスケールしたり、片方だけ再デプロイしたりできる。
> lunch-bot で言うと、cron 設定の YAML と Go コードを物理的に切り離してるのと同じ感覚。

### Q2. なんで間に Cloud Tasks を挟んでるの？
> **「個別リトライ」と「並列実行」と「失敗の局所化」をタダで手に入れるため**。
> 例えば 50 テナント分のジョブを積んだ時、3 テナントだけ Amazon API が一時的に死んでも、その3つだけリトライされる。残り47は普通に成功で終わる。
> lunch-bot は同期1段なので、途中で1個コケたら全部失敗扱い。

### Q3. lunch-bot との一番大きな違いは？
> **規模じゃなく「非同期」かどうか**。
> lunch-bot は cron が直接 Goバイナリを叩いて、終わるまで待つ。Polaris は cron が dispatcher を叩いた瞬間に dispatcher の仕事は終わり、本体の処理は Cloud Tasks 経由で別の Cloud Run が拾って動く。
> これにより 30 分かかる処理でも cron 側はタイムアウトしない。

---

## ◆ 次に読むファイル

- `notes/11_polaris_unified_api.md` — 3層 + DI の深掘り (本編の中身)
- `notes/12_polaris_task_dispatcher.md` — Cloud Scheduler → Cloud Tasks の実行制御
- `notes/13_polaris_dbt.md` — staging/intermediate/dwh/mart (補足、軽め)
