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
