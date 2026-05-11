# 5/15 発表 スライド構成 (20分)

## 全体方針

- **主戦場**: 「自作 lunch-bot ↔ Polaris (unified-api / task-dispatcher) の対応づけ」(重み8)
- **副戦場**: 「なぜその設計にするのか」(重み2)
- 厚く話す: Go / Unified API / API エンドポイント呼び出し / ジョブ呼び出し / Task Dispatcher
- 補足で軽く: dbt / staging / intermediate / mart
- 「分からない言葉が出てくると諦める」聞き手対策: 専門用語を出すスライドの **直前** で、その用語を 1 行説明 (用語集と一致)

合計 **18 スライド**、推定 **約 19 分 30 秒** (Q&A 別)。

---

## スライド 1: タイトル + 自己紹介

- **目的**: 「26卒未経験の私が Go の自作アプリを足場に Polaris の構成を理解した話」というテーマを伝える
- **要点**:
  - タイトル「Slack ランチ bot を足場に Polaris (unified-api) の構成を理解する」
  - 自己紹介: 26卒、未経験、Go の基本文法習得済み、Cloud は学習中
  - vibe coding で作った lunch-bot を「読解教材」として骨の髄まで分解した
  - 今日は **設計の対応づけ** に集中して話す
- **使う図/コード**: なし (タイトルスライド)
- **推定時間**: 60 秒
- **依存する用語**: Go / Polaris / lunch-bot / vibe coding
- **lunch-bot 対応**: アプリそのものの紹介

---

## スライド 2: 発表のゴール共有

- **目的**: 「今日聞き手に持ち帰ってほしい3つ」を最初に宣言する
- **要点**:
  - ゴール 1: lunch-bot が **3層構造 (handler / service / repository)** で動いていることを説明する
  - ゴール 2: lunch-bot と **Polaris (unified-api / task-dispatcher)** が同じ骨格でできていることを説明する
  - ゴール 3: 私が Go のコードを Polaris 全体の構成に **関連づけて** 理解していることを示す
  - 「分からない用語が出たら止めてください」を最初に宣言 (聞き手安心化)
  - 用語は出すたびに 1 行で説明する、と約束
- **使う図/コード**: ゴール3項目の箇条書きスライド
- **推定時間**: 60 秒
- **依存する用語**: handler / service / repository / unified-api / task-dispatcher
- **lunch-bot 対応**: 全体方針の宣言

---

## スライド 3: lunch-bot とは何か (デモ画像)

- **目的**: 「lunch-bot が何をする bot か」を 1 枚の絵で全員が分かる状態にする
- **要点**:
  - 週 2 回、Slack にランチ募集メッセージを投げる
  - お弁当絵文字でリアクションした人を集めて、ランダムに 3〜4 人組に分けて発表する
  - GitHub Actions の **cron** で自動実行 (人手は不要)
  - 入力も出力も Slack。Slack が唯一のデータソース
  - これを 1 個の Go バイナリで実装している (300〜400 行程度)
- **使う図/コード**: 募集投稿 → リアクション → 発表投稿 のスクリーンショット 3 枚 (もしくは概念図)
- **推定時間**: 90 秒
- **依存する用語**: Slack / cron / GitHub Actions
- **lunch-bot 対応**: アプリ全体の動きを共有

---

## スライド 4: lunch-bot の処理を ETL で見る

- **目的**: 「lunch-bot は小さな ETL である」と先に枠組みを与える (Polaris との接続点)
- **要点**:
  - **Extract (取る)**: Slack API でリアクション一覧を取得
  - **Transform (変換)**: 参加者をシャッフルしてグループ分け
  - **Load (戻す)**: Slack に発表メッセージを投稿
  - つまり「取って・変えて・戻す」の 3 段
  - Polaris も同じ ETL を、もっと大規模・複数ソースでやっているだけ
- **使う図/コード**: `Slack(取得) → service(変換) → Slack(投稿)` の 3 ボックス図
- **推定時間**: 60 秒
- **依存する用語**: ETL / Extract / Transform / Load / API
- **lunch-bot 対応**: アプリ全体を ETL の枠で再解釈

---

## スライド 5: lunch-bot の3層構造 (handler / service / repository)

- **目的**: lunch-bot のフォルダ構造と「責務分離」を 1 枚で見せる
- **要点**:
  - **handler**: 入口の係 (どのコマンドが来たか分岐するだけ)
  - **service**: 業務ロジックの係 (シャッフル、集計、メッセージ組立)
  - **repository**: 外部の係 (Slack API を叩く)
  - 各層は **interface** で繋がっており、上の層は下の具体実装を知らない
  - `cmd/bot/main.go` で 3 層を組み立てて `Recruit()` を呼ぶ
- **使う図/コード**: `cmd → handler → service → repository → Slack` の縦型 4 段絵 + ファイル名対応
- **推定時間**: 90 秒
- **依存する用語**: handler / service / repository / interface / 責務分離 / 3層アーキテクチャ
- **lunch-bot 対応**: `internal/handler/lunch_handler.go` / `internal/service/lunch_service.go` / `internal/repository/slack_client.go`

---

## スライド 6: lunch-bot の DI と interface (差し替え可能性)

- **目的**: 「interface で受けて DI する」ことで本物 ⇄ 偽物を差し替えられる体験を見せる
- **要点**:
  - service は `SlackRepository` という **interface** を受け取る (具体実装ではない)
  - 本番は `*SlackClient` を渡す (実際に Slack API を叩く)
  - テスト用 `cmd/simulate` では `fakeSlack` を渡す (Slack を叩かず、メモリで動く)
  - 同じ service コードのまま、入れ替えるだけで動く = **DI の威力**
  - これは Polaris で「同じ service を HTTP と CLI 両方から呼べる」のと **同じ仕組み**
- **使う図/コード**: 抜粋コード (15 行以内)
  ```go
  type SlackRepository interface { /* 5メソッド */ }
  type LunchService struct { slack SlackRepository }
  func NewLunchService(slack SlackRepository, ch string) *LunchService { ... }

  // 本番
  svc := service.NewLunchService(repository.NewSlackClient(token), ch)
  // テスト
  svc := service.NewLunchService(&fakeSlack{}, ch)
  ```
- **推定時間**: 90 秒
- **依存する用語**: interface / DI / コンストラクタ / struct / fake (モック)
- **lunch-bot 対応**: `internal/service/lunch_service.go` の `SlackRepository` interface + `cmd/simulate`

---

## スライド 7: Polaris とは何か

- **目的**: Polaris の正体を 1 枚で「大きな ETL 工場」として伝える
- **要点**:
  - Amazon / 楽天 / 広告 API などから **毎日データを集めて BigQuery に貯めて、分析しやすい形に整える工場**
  - やっていることの種類は lunch-bot と同じ (ETL)。違うのは **規模・連携先の数・起動の仕組み**
  - 本編で扱う 2 フォルダ:
    - `ingestion/unified-api/` (Go) — 集めて BQ に書く本体
    - `orchestration/task-dispatcher/` (Go) — 時間で叩き、ジョブを Cloud Tasks に積む係
  - 補足で扱う: `transformation/androots/` (dbt)
- **使う図/コード**: `notes/10_polaris_overview.md` の「工場の流れ (1枚絵)」を簡略化した図
- **推定時間**: 60 秒
- **依存する用語**: Polaris / ingestion / orchestration / transformation / BigQuery / dbt
- **lunch-bot 対応**: lunch-bot は「ingestion + transformation + application を 1 個に圧縮」した小さな工場であると説明

---

## スライド 8: lunch-bot ↔ Polaris 対応表 ★ メインスライド

- **目的**: 主戦場。「lunch-bot を 10 倍に拡張すると Polaris になる」を 1 枚で示す
- **要点**:
  - 観点ごとに「lunch-bot 側」「Polaris 側」を並べる
  - 「**起動の引き金**」「**入口層 (handler)**」「**業務 (service)**」「**外部連携 (repository)**」「**DI 組立場所**」「**データ保存先**」「**実行単位**」の 7 行
  - 結論: 「**3層 + DI の骨格は同じ**。違うのは ① 起動口の数 ② 連携先の数 ③ 書込先 (BQ) ④ 1 cron が **N テナント分** に展開されること」
  - このスライドは **30 秒では話せない** ので 3 分使う
- **使う図/コード**: `notes/10_polaris_overview.md` の対応表 (7 行に絞って整形)
- **推定時間**: 180 秒
- **依存する用語**: handler / service / repository / DI / cron / Cloud Scheduler / Cloud Tasks / Cloud Run / テナント / BigQuery
- **lunch-bot 対応**: 全層

---

## スライド 9: unified-api の中身 — handler / service / repository

- **目的**: 「unified-api は lunch-bot と同じ 3 層」だとコードレベルで示す
- **要点**:
  - フォルダ構造を見せる: `cmd/api/` `cmd/jobs/` `internal/handler/` `internal/service/<platform>/` `internal/repository/<platform>/`
  - `repository/slack/bot_client.go` の `BotRepository` は lunch-bot の `SlackClient` と **ほぼコピペ** (骨格はそっくり)
  - service は依存が増えただけ (lunch-bot は 1 個 / Polaris は httpRepo + bqRepo + mdmRepo + etlRunner の 4 個)
  - handler は「decode → validate → service 呼ぶ → JSON 返す」の HTTP 作法が増えただけで、本質は lunch-bot と同じ「service を 1 行呼ぶ」
- **使う図/コード**: `notes/11_polaris_unified_api.md` の「並べてみると」表 (Service 層の比較表)
- **推定時間**: 120 秒
- **依存する用語**: chi / handler / service / repository / struct / receiver / interface / decode / validate
- **lunch-bot 対応**: lunch-bot の 3 層と完全に対応

---

## スライド 10: unified-api の DI 組立 (`routes.go`)

- **目的**: 「下から順に組み上げる」DI のパターンが lunch-bot と Polaris で同じだと示す
- **要点**:
  - lunch-bot: `cmd/bot/main.go` で `NewSlackClient → NewLunchService → NewLunchHandler` を順に組む (3 行)
  - Polaris: `cmd/api/routes.go` の `registerXxxRoutes()` がプラットフォームごとに同じことをやる
  - 共通リソース (BigQuery / Supabase / Slack 通知) は **1 回だけ** 作って register 関数に渡す
  - 最後に **chi の `r.Post("/customers", h.PostCustomers)`** で URL に紐付け (lunch-bot の `switch os.Args[1]` 相当)
  - **「DI の本質は順番に組むだけ」**を強調
- **使う図/コード**: `notes/11_polaris_unified_api.md` の `registerEcforceRoutes` 抜粋 (10 行) + lunch-bot 抜粋 (3 行) を上下に並べる
- **推定時間**: 120 秒
- **依存する用語**: chi / chiルータ / DI / コンストラクタ / register / URL / HTTPメソッド (POST)
- **lunch-bot 対応**: `cmd/bot/main.go` の DI 組立 3 行

---

## スライド 11: ジョブ呼び出し vs API エンドポイント呼び出し ★

- **目的**: 「同じ service を 2 つの入口から呼べる」設計判断を示す (interface + DI の実利)
- **要点**:
  - **API エンドポイント呼び出し**: HTTP で `/api/v2/ecforce/customers` を叩く (Cloud Run 常駐サーバが受ける)
  - **ジョブ呼び出し** (狭義): Cloud Tasks 経由で上記 HTTP を **非同期に** 叩くこと
  - **Cloud Run Jobs** (CLI バッチ): `cmd/jobs/main.go` を CLI 起動。handler を **通らず** runner が直接 service を叩く
  - 使い分け: 短時間 → Cloud Tasks 経由 / 30分超 → Cloud Run Jobs
  - **service は同じ**。入口だけ 2 系統。これが interface + DI の実利
  - lunch-bot で言えば「本物 Slack」と「fake Slack」を差し替えたのと **本質的に同じ**
- **使う図/コード**: `notes/11_polaris_unified_api.md` の「2 つの経路」アスキーアート図
- **推定時間**: 120 秒
- **依存する用語**: API エンドポイント / ジョブ呼び出し / HTTP / Cloud Run / Cloud Run Jobs / Cloud Tasks / 同期 / 非同期 / runner
- **lunch-bot 対応**: `cmd/simulate` で fake を差し替えた = service を別入口から呼んだ、と同じ発想

---

## スライド 12: task-dispatcher と 4段非同期の絵 ★

- **目的**: lunch-bot の「cron 1 段」が Polaris では「4 段非同期」に拡張されていると見せる
- **要点**:
  - 4 段の絵を見せる: ① Cloud Scheduler (タイマー) → ② task-dispatcher (振り分け) → ③ Cloud Tasks (待ち行列) → ④ unified-api (働く)
  - lunch-bot は **1 段同期** (GitHub Actions cron が直接 Go バイナリを叩く)
  - Polaris は **4 段非同期** (cron は積むだけで終わり、後段が独立に動く)
  - task-dispatcher の仕事: Supabase からテナント一覧を引いて、テナント数だけ Cloud Tasks に積む (= ファンアウト)
  - 1 タスク = 「unified-api の `/api/v2/ecforce/customers` を tenant=X で叩け」というジョブ
- **使う図/コード**: `notes/12_polaris_task_dispatcher.md` の 4 段アスキーアート図
- **推定時間**: 120 秒
- **依存する用語**: Cloud Scheduler / Cloud Tasks / task-dispatcher / unified-api / cron / OIDC token / 非同期 / 同期 / テナント / ファンアウト / Supabase
- **lunch-bot 対応**: `.github/workflows/recruit.yml` の cron + `cmd/bot/main.go` を 4 段に分解した版

---

## スライド 13: なぜ Cloud Tasks を挟むのか (設計の核)

- **目的**: 「重み2の『なぜその設計か』」を 1 分でズバリ言う
- **要点**:
  - 唯一の理由: **「個別リトライ・並列実行・失敗の局所化」をタダで手に入れるため**
  - 例: 50 テナント中 3 つだけ Amazon API が一時エラー → その 3 つだけリトライ。残り 47 は成功で完了
  - 例: 30 分かかる処理 → cron 側はタイムアウトしない (積んだ瞬間に cron の仕事は終わる)
  - lunch-bot は同期 1 段なので、途中で 1 個コケたら全部失敗扱い
  - 「**間に待ち行列を挟む = 各タスクを独立した運命にする**」が一撃の核
- **使う図/コード**: `notes/12_polaris_task_dispatcher.md` の「1段同期だと困ること / 4段非同期で解決」表
- **推定時間**: 60 秒
- **依存する用語**: Cloud Tasks / リトライ / 並列 / 非同期 / 同期 / キュー / 失敗の局所化 / タイムアウト
- **lunch-bot 対応**: lunch-bot の cron + 1 バイナリでは 1 個コケると全部失敗、を反例として使う

---

## スライド 14: dbt 補足 — BigQuery → staging → intermediate → mart

- **目的**: 「集めたあとどこで加工されるか」の流れだけ示す。深入りしない
- **要点**:
  - unified-api が集めたデータは BigQuery の `Raw_*` テーブルに貯まる
  - そこから先は **dbt** (SQL + YAML だけで動く別世界) が引き継ぐ
  - 4 層: **staging** (型を整える) → **intermediate** (組み立てる) → **dwh** (倉庫に置く) → **mart** (BI 用に出荷)
  - 料理工程の「下ごしらえ → 仕込み → 保管 → 提供」と同じ
  - 「Go の世界はここで終わり、SQL の世界に入る」とだけ伝える
- **使う図/コード**: `notes/13_polaris_dbt.md` の 4 段階の絵 (簡略化)
- **推定時間**: 60 秒
- **依存する用語**: dbt / SQL / BigQuery / dataset / staging / intermediate / dwh / mart / dim / fct / スタースキーマ / マクロ
- **lunch-bot 対応**: lunch-bot は in-memory で済ませている部分が、Polaris では BQ + dbt に分担されている

---

## スライド 15: 全体を 1 枚に圧縮 (lunch-bot ⇄ Polaris の通し図)

- **目的**: ここまでの話を 1 枚で再生できる「持ち帰り絵」を渡す
- **要点**:
  - 上段: lunch-bot の流れ (cron → cmd/bot → handler → service → repository → Slack)
  - 下段: Polaris の流れ (Cloud Scheduler → task-dispatcher → Cloud Tasks → unified-api (handler → service → repository) → BigQuery → dbt → mart)
  - 上下を矢印で繋ぐ: 「同じ骨格を **規模拡張 + 非同期化 + 加工層追加** したのが Polaris」
  - **このスライドが今日の答え**
- **使う図/コード**: 上下 2 段の通し図 (新規作成、`notes/10` と `notes/12` の絵を合体)
- **推定時間**: 90 秒
- **依存する用語**: 全部 (ここまでの登場語の総復習)
- **lunch-bot 対応**: 全体

---

## スライド 16: できるようになったこと

- **目的**: 「読解の到達点」を聞き手 (上司・先輩) に共有
- **要点**:
  - lunch-bot の 3 層を **自分の言葉で** 説明できる
  - unified-api の `routes.go` を読んで DI の組立順を追える
  - task-dispatcher の `main.go` を見て「Scheduler から Cloud Tasks に積む」流れを説明できる
  - 「ジョブ呼び出し」「API エンドポイント呼び出し」を区別して話せる
  - **Go コードを読んで「何が起きるか」を頭で再生できるようになった**
- **使う図/コード**: 4 項目の箇条書き (チェックボックス風)
- **推定時間**: 60 秒
- **依存する用語**: handler / service / repository / DI / routes.go / task-dispatcher / unified-api / ジョブ呼び出し / API エンドポイント
- **lunch-bot 対応**: 学習成果としての lunch-bot

---

## スライド 17: まだ課題なこと (正直に)

- **目的**: 嘘をつかない。次の学習対象を宣言する
- **要点**:
  - **dbt の中身**: SQL マクロは流れだけ理解、書ける段階ではない
  - **Cloud (GCP) の細部**: IAM / OIDC / Terraform は表面しか触っていない
  - **エラー処理の運用設計**: `fmt.Errorf %w` で wrap しているが、本番でどうログ集約しているかは未調査
  - **テストの書き方**: lunch-bot にはテストがほぼない。Polaris のテストパターンはこれから読む
  - 次に読むのは: Polaris の **テスト** + **Cloud Tasks 経由のジョブ実行ログ**
- **使う図/コード**: 4 項目の箇条書き
- **推定時間**: 60 秒
- **依存する用語**: dbt / マクロ / OIDC token / Terraform / fmt.Errorf %w / wrap
- **lunch-bot 対応**: 課題として lunch-bot にテストがないことを言う

---

## スライド 18: 締め

- **目的**: 持ち帰る 1 行を残す
- **要点**:
  - 「**Polaris の unified-api は、lunch-bot と全く同じ 3層 + DI で作られている。違うのは ① 起動口の数、② 連携先の数、③ 書込先 (BQ)、④ 1 cron が N テナント分にファンアウトされること、それだけ。**」
  - 「**自作の小さなアプリを骨の髄まで読むと、業務の大きなコードも同じ目で読める**」
  - 質問はチャットでも可
- **使う図/コード**: 1 行の太字メッセージ + 「ありがとうございました」
- **推定時間**: 30 秒
- **依存する用語**: unified-api / DI / 3層 / テナント / ファンアウト
- **lunch-bot 対応**: 締めの 1 行で lunch-bot を主役に置く

---

## 推定時間まとめ

| # | タイトル | 秒 |
|---|---|---|
| 1 | タイトル + 自己紹介 | 60 |
| 2 | 発表のゴール | 60 |
| 3 | lunch-bot とは何か | 90 |
| 4 | ETL で見る | 60 |
| 5 | 3層構造 | 90 |
| 6 | DI と interface | 90 |
| 7 | Polaris とは何か | 60 |
| 8 | **対応表 ★** | 180 |
| 9 | unified-api の3層 | 120 |
| 10 | unified-api の DI 組立 | 120 |
| 11 | **ジョブ vs API ★** | 120 |
| 12 | **task-dispatcher 4段絵 ★** | 120 |
| 13 | なぜ Cloud Tasks を挟むか | 60 |
| 14 | dbt 補足 | 60 |
| 15 | 通し図 | 90 |
| 16 | できるようになったこと | 60 |
| 17 | まだ課題なこと | 60 |
| 18 | 締め | 30 |
| **合計** | | **1,170 秒 = 約 19 分 30 秒** |

★ = 主戦場 (重み 8)
