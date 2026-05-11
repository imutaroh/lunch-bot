# notes/13_polaris_dbt.md — `transformation/androots/` (dbt) 補足

5/15 発表の **補足パート**。本編 (unified-api / task-dispatcher) のあとに 1〜2 分で「集めたデータがここで加工される」と説明できれば充分。**深入り厳禁**。

---

## ◆ dbt は何屋か (1行)

> **「BigQuery の中だけで SQL を順番に走らせて、生データを分析しやすい形に加工するツール」**

つまり Go や Python ではなく、**SQL + YAML だけで動く別世界**。Polaris ではこれが `transformation/androots/` に置かれている。

---

## ◆ 4段階の絵 (発表のサブスライド素材)

```
[unified-api が書き込む]            [dbt が SQL で順に作る]                      [使う側]

BigQuery (Raw_*)         ──→       staging (Hrd_*)             ──→   dwh ──→   mart       ──→   人間 / BI
生データそのまま                   型・列名を整形                   全社共通          BI直結用
1ファイル = 生API応答              マクロ1行で量産                  の倉庫            (dim_/fct_)
                                            │
                                            ▼
                                     intermediate
                                  (1_extract → 2_join
                                  → 3_union → 4_soft-business
                                   番号付きで処理順を可視化)
```

**4 つの層の役割**:

| 層 | 役割 | 例 |
|---|---|---|
| **staging** (`Hrd_*`) | 生データの **型・命名統一** だけ | Amazonの `Raw_..._OrderItem` を `Hrd_..._OrderItem` に正規化 |
| **intermediate** | 抽出 → 結合 → 統合 → ビジネスルール の段階加工 | 複数テナントの注文データを 1 つに UNION する |
| **dwh** (bdw / rdw / ref) | **全社共通の整形済み倉庫** | 顧客マスタ、商品マスタ、参照データなど |
| **mart** (dim_* / fct_*) | BI / レポート / 書き戻し用の **最終形** | スタースキーマ (日付ディメンション、売上ファクト) |

> 「staging で形を整えて、intermediate で組み立てて、dwh で倉庫に置いて、mart でお客さんに出す形にする」 = **料理工程と同じ**。

---

## ◆ コード例 (各層 1ファイルだけ)

### staging — マクロ1行で量産

`models/staging/amazon/Hrd_EsellaFrom_Amazon_..._OrderItem.sql`:

```sql
{{ hard_amazon_orderItem_v2('Raw_EsellaFrom_Amazon_..._OrderItem') }}
```

> **これだけ**。中身は `macros/` のマクロが全部やる。staging の SQL は基本「マクロ呼び出し1行」。

### intermediate — 番号付きディレクトリで処理順を可視化

`models/intermediate/1_extract/mdm/int_mdm__teams_extracted.sql`:

```sql
{{ extract_and_union(
    source_type = 'mdm_team',
    models = ['Hrd_Androots_StaffDB_Default_Team'],
    is_historical = False
) }}
```

> ディレクトリ名が `1_extract` → `2_join` → `3_union` → `4_soft-business` → `99_transform` と番号付き。**処理の順番がフォルダ名で見える** のが工夫ポイント。

### mart — スタースキーマで BI 直結

`models/mart/star-schema/dim_date.sql` (一部):

```sql
with first as (
    select
        format_date('%Y%m%d', date_day) as date_key,
        date_day as full_date,
        bqfunc.holidays_in_japan__asia_northeast1.holiday_name(date_day) as public_holiday_name
    from {{ ref('metricflow_time_spine') }}
)
```

> 日付ディメンション (年月日 + 曜日 + 祝日 + 営業日番号) を作るおなじみのスタースキーマ部品。

---

## ◆ 押さえる核 (発表で言うべき1分要約)

> **「unified-api が集めてきたデータは BigQuery の `Raw_*` テーブルに貯まる。**
> **そこから先は dbt が SQL を順番に走らせて、staging で型を整え、intermediate で段階加工し、dwh で倉庫に置き、mart で BI 用の形に整える。**
> **Go の世界からは抜けて、SQL + YAML の世界に入る。**
> **僕がいま入口 (Go) を読んでるところで、ここから先 (dbt) はまだ流れだけつかんだ段階。」**

---

## ◆ 発表で問われたら答える Q&A

### Q1. なんで dbt を使うの? Goでも書ける?
> **「BigQuery の中で完結する処理は SQL の方が速くて読みやすいから」**。
> Go で書くと一旦 BigQuery から取り出して計算してまた書き戻すので無駄。dbt は SQL を BigQuery に投げるだけで全部 BigQuery 内で完結する。

### Q2. staging / intermediate / dwh / mart の差を一言で
> **「整える / 組み立てる / 倉庫に置く / 出荷する」**。料理工程の「下ごしらえ / 仕込み / 保管 / 提供」と同じ。

### Q3. macros って何?
> **「SQL の関数」**。同じ整形処理を 50 ファイルでコピペするのを避けるため、共通処理をマクロに切り出して staging の SQL を 1 行にしている。Go で言う関数の切り出しと同じ発想。

---

## ◆ ここは深入りしない (発表でも触れない)

- BigQuery の dataset 設計の細かい話
- dbt のテスト機能
- Cloud Composer / dbt Cloud / dbt Platform の運用差
- dwh の bdw / rdw / ref の使い分け

「データはここで加工されてる」と流れだけ示せば充分。**入口側 (Go) を理解できていれば、それで足場は完成**。

---

## ◆ 全体まとめ (5/15 発表の通しイメージ)

```
1. ゴール共有 (1分)
2. lunch-bot の概要 (2分)            ──→ Slack 経由のシャッフル
3. lunch-bot の構成 (4分)            ──→ handler / service / repository + DI
4. 自作と Polaris の対応関係 (3分)   ──→ notes/10_polaris_overview.md の対応表
5. unified-api 深掘り (4分)          ──→ notes/11_polaris_unified_api.md
6. task-dispatcher 深掘り (3分)      ──→ notes/12_polaris_task_dispatcher.md
7. データインジェスチョン → モデリング (1分)
                                     ──→ ★このファイル (dbt は流れだけ)
8. できるようになったこと / 課題 (1分)
9. 締め (30秒)
```

→ **計 約20分 (Q&A 込みで25〜30分想定)**。
