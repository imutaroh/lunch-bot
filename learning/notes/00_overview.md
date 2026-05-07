# notes/00_overview.md — 全体俯瞰 (Day 1 Loop 3 前半時点)

このファイルは `lunch-bot` の **全体構造** と **動きの流れ** を1ページで把握するためのまとめ。
Day 3 Loop 8 で ETL対応図 + 設計判断5つ を追記する予定。

> **読了範囲のスナップショット (2026-04-30 時点)**
> - 読んだ: `cmd/bot/main.go` / `internal/config/` / `internal/handler/` / `internal/service/lunch_service.go` の前半 (interface + struct + const + RunRecruit)
> - まだ: `RunAnnounce` / `internal/service/shuffler.go` / `internal/repository/slack_client.go` / `cmd/simulate/main.go`

---

## 1. アーキテクチャ図 (3層レイヤー構造)

```mermaid
flowchart TB
    subgraph EXT["🌐 外部世界"]
        CRON["GitHub Actions cron"]
        SLACK["Slack Web API"]
    end

    subgraph CMD["📦 cmd/ — エントリポイント (配線して起動するだけ)"]
        BOT["cmd/bot/main.go<br/>本番用 (recruit / announce)"]
        SIM["cmd/simulate/main.go<br/>シミュレータ (本物Slack 不要) ※TBD"]
    end

    subgraph HANDLER["🚪 internal/handler/ — 入口の薄い層"]
        H["LunchHandler<br/>・svc *service.LunchService<br/>・Recruit()  → svc.RunRecruit()<br/>・Announce() → svc.RunAnnounce()"]
    end

    subgraph SERVICE["🧠 internal/service/ — 業務ロジック本体"]
        S["LunchService<br/>・slack         SlackRepository ★interface<br/>・channelID     string<br/>・emoji         'bento'<br/>・lookbackHours 26h<br/><br/>RunRecruit() / RunAnnounce()"]
    end

    subgraph REPO["🔌 internal/repository/ — 外部I/Oの本体 ※TBD"]
        R["SlackClient (= SlackRepository 実装)<br/>PostMessage / AddReaction /<br/>GetReactionUsers / WhoAmI /<br/>RecentBotMessages"]
    end

    CRON --> BOT
    BOT -->|配線: repo → svc → handler| H
    SIM -.fake差し替え.-> S
    H -->|委譲するだけ| S
    S -->|★ interface 経由で頼む| R
    R -->|HTTP| SLACK
    SLACK -.募集/発表.-> CRON

    style EXT fill:#fa8c16,stroke:#873800,stroke-width:2px,color:#fff
    style CMD fill:#1890ff,stroke:#003a8c,stroke-width:2px,color:#fff
    style HANDLER fill:#52c41a,stroke:#135200,stroke-width:2px,color:#fff
    style SERVICE fill:#eb2f96,stroke:#780650,stroke-width:2px,color:#fff
    style REPO fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
```

### 各層の役割を1行で

| 層 | ファイル | 役割 (1行) |
|---|---|---|
| `cmd/` | `cmd/bot/main.go` | 起動するだけ。引数を見て recruit/announce を分岐 |
| `config/` | `internal/config/config.go` | 環境変数から token と channelID を読む |
| `handler/` | `internal/handler/lunch_handler.go` | 入口。サブコマンドを service に委譲するだけ |
| `service/` | `internal/service/lunch_service.go` | 業務ロジック本体 |
| `repository/` | `internal/repository/slack_client.go` | Slack API を実際に叩く実体 (HTTP) |

### ★ interface (`SlackRepository`) の役割

```mermaid
flowchart LR
    SVC["service<br/>(LunchService)"]
    IF{{"interface<br/>SlackRepository<br/>= 注文書"}}
    REAL["repository.SlackClient<br/>(本物Slack)"]
    FAKE["fakeSlack<br/>(メモリ上の偽物)"]

    SVC -->|型でしか繋がってない| IF
    IF -.本番.-> REAL
    IF -.テスト/simulate.-> FAKE

    style SVC fill:#eb2f96,stroke:#780650,stroke-width:2px,color:#fff
    style IF fill:#faad14,stroke:#613400,stroke-width:3px,color:#000
    style REAL fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
    style FAKE fill:#13c2c2,stroke:#00474f,stroke-width:2px,color:#fff
```

- 本番: `repository.SlackClient` (本物Slack)
- テスト/simulate: `fakeSlack` (メモリ上の偽物) ← interface があるから差し替え可能

→ これが `cmd/simulate` で本物Slack 無しにロジック検証できる仕組みの土台。

---

## 2. データフロー図 (リクエスト → 処理 → レスポンス)

### A. Recruit フロー (月曜09:00 JST)

```mermaid
sequenceDiagram
    autonumber
    participant Cron as GitHub Actions cron
    participant Main as cmd/bot/main.go
    participant H as handler.Recruit()
    participant S as service.RunRecruit()
    participant R as repository.SlackClient
    participant Slack as Slack Web API

    Cron->>Main: $ ./lunch-bot recruit
    Note over Main: 配線フェーズ<br/>config.Load() → token, channelID<br/>NewSlackClient(token)<br/>NewLunchService(slack, ch)<br/>NewLunchHandler(svc)
    Main->>H: switch mode == "recruit"<br/>handler.Recruit()
    H->>S: return svc.RunRecruit()

    Note over S: ① fmt.Println("[recruit] 募集投稿を出します")
    S->>R: ② PostMessage(channelID, recruitText)
    R->>Slack: HTTP POST chat.postMessage
    Slack-->>R: ts (タイムスタンプ)
    R-->>S: ts
    Note over S: ③ if err { return fmt.Errorf("...: %w", err) }

    S->>R: ④ AddReaction(channelID, ts, "bento")
    R->>Slack: HTTP POST reactions.add
    Slack-->>R: ok
    R-->>S: nil
    Note over S: ⑤ if err { return ... }<br/>⑥ return nil

    Slack-->>Cron: 募集投稿 + 🍱(1票) が表示
```

エラーは各層で `%w` で包んで上に投げる → 最終的に `cmd/bot/main.go` の `log.Fatalf` で出力 + 非0終了。
GitHub Actions が失敗を検知 → 通知。

### B. Announce フロー (火曜09:00 JST)  ※TBD

明日 Loop 4 で `RunAnnounce` を読む。今は**ざっくり**だけ:

```mermaid
flowchart TD
    START(["$ ./lunch-bot announce"])
    A["slack.WhoAmI()<br/>→ bot自身のID"]
    B["slack.RecentBotMessages(...)<br/>→ 直近26h の bot 投稿リスト"]
    C{"冪等性チェック<br/>既に発表済み or<br/>お休み済み?"}
    DONE(["return nil<br/>(何もせず終了)"])
    D["recruitPrefix で<br/>募集投稿を発見"]
    E["slack.GetReactionUsers(...)<br/>→ 🍱を押したユーザー集計"]
    F["excludeUser(users, botID)<br/>→ bot自身を除外"]
    G{"参加者 < 3?"}
    REST["slack.PostMessage(restMessage)<br/>→ お休み投稿で終了"]
    H["Shuffle(users)<br/>→ グループ分け (※Loop 5)"]
    I["slack.PostMessage(announcement)<br/>→ グループ発表"]

    START --> A --> B --> C
    C -- yes --> DONE
    C -- no --> D --> E --> F --> G
    G -- yes --> REST
    G -- no --> H --> I

    style START fill:#1890ff,stroke:#003a8c,stroke-width:2px,color:#fff
    style A fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
    style B fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
    style C fill:#faad14,stroke:#613400,stroke-width:3px,color:#000
    style G fill:#faad14,stroke:#613400,stroke-width:3px,color:#000
    style D fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
    style E fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
    style F fill:#eb2f96,stroke:#780650,stroke-width:2px,color:#fff
    style H fill:#eb2f96,stroke:#780650,stroke-width:2px,color:#fff
    style DONE fill:#595959,stroke:#262626,stroke-width:2px,color:#fff
    style REST fill:#ff4d4f,stroke:#820014,stroke-width:2px,color:#fff
    style I fill:#52c41a,stroke:#135200,stroke-width:2px,color:#fff
```

→ Day 2 で詳細を埋める。

---

## 3. 「配線」の比喩 (= 依存性注入 / DI)

`cmd/bot/main.go` でやってることは、**プラモデルの組み立て**:

```mermaid
flowchart LR
    P1["1. 部品を作る<br/>NewSlackClient(token)"]
    P2["2. 部品に部品を挿す<br/>NewLunchService(slack, ch)"]
    P3["3. さらに挿す<br/>NewLunchHandler(svc)"]
    P4["4. 完成品を起動<br/>handler.Recruit() /<br/>handler.Announce()"]

    P1 -->|repo を svc に| P2
    P2 -->|svc を handler に| P3
    P3 --> P4

    style P1 fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
    style P2 fill:#eb2f96,stroke:#780650,stroke-width:2px,color:#fff
    style P3 fill:#52c41a,stroke:#135200,stroke-width:2px,color:#fff
    style P4 fill:#1890ff,stroke:#003a8c,stroke-width:2px,color:#fff
```

```mermaid
flowchart TB
    H["handler"]
    S["service"]
    IF{{"interface<br/>(注文書)"}}
    R["repository<br/>(実体)"]

    H -->|service だけ知ってる| S
    S -->|interface だけ知ってる| IF
    IF -.実体は知らない.-> R

    style H fill:#52c41a,stroke:#135200,stroke-width:2px,color:#fff
    style S fill:#eb2f96,stroke:#780650,stroke-width:2px,color:#fff
    style IF fill:#faad14,stroke:#613400,stroke-width:3px,color:#000
    style R fill:#722ed1,stroke:#22075e,stroke-width:2px,color:#fff
```

- 上の層 (handler) は下の層 (repo) のことを **直接知らない**。
- service が間にいて、handler は service だけ知ってる。
- service は repo の実体を知らず、interface (注文書) だけを知ってる。

→ 各層が **すぐ下の層しか知らない** = 関心の分離。

---

## 4. ETL 視点の対応 (※Day 3 Loop 8 で完成させる予定 / 仮置き)

```mermaid
flowchart LR
    subgraph E["🟢 Extract (取り出す)"]
        E1["slack.GetReactionUsers"]
        E2["slack.RecentBotMessages"]
    end

    subgraph T["🟡 Transform (変換)"]
        T1["Shuffle<br/>グループ分け"]
        T2["buildAnnouncement<br/>文章組み立て"]
    end

    subgraph L["🔵 Load (書き戻す)"]
        L1["slack.PostMessage"]
        L2["slack.AddReaction"]
    end

    SRC[("Slack<br/>= Source of Truth")]
    SRC --> E --> T --> L --> SRC

    style E fill:#52c41a,stroke:#135200,stroke-width:2px,color:#fff
    style T fill:#faad14,stroke:#613400,stroke-width:2px,color:#000
    style L fill:#1890ff,stroke:#003a8c,stroke-width:2px,color:#fff
    style SRC fill:#595959,stroke:#262626,stroke-width:2px,color:#fff
    style E1 fill:#237804,stroke:#092b00,color:#fff
    style E2 fill:#237804,stroke:#092b00,color:#fff
    style T1 fill:#ad6800,stroke:#613400,color:#fff
    style T2 fill:#ad6800,stroke:#613400,color:#fff
    style L1 fill:#0050b3,stroke:#002766,color:#fff
    style L2 fill:#0050b3,stroke:#002766,color:#fff
```

→ lunch-bot は **Slack を Source of Truth として、Slack から読み Slack に書く** ETL ジョブ。
状態をDBに持たない (= ステートレス) のが特徴。

---

## 5. メモ (今日の発見)

- interface の本当のおいしさ: 「型が合えば誰でもOK」 → fake と差し替え可能
- const が prefix と Text に **物理的に分かれてる** のは: ① 絵文字正規化問題 (Unicode↔colon-code) ② 本文を改修しても prefix さえ変えなければ過去投稿検索が壊れない
- `%w` (エラーラッピング) と `%v` の違い: `%w` は元エラーを潰さず封筒に包む → 上位で `errors.Is` 判別可能
- `fmt.Println` の出力先: stdout → ローカルなら端末、GitHub Actions ならワークフローログ
