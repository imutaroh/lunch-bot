# notes/04_handler.md — `internal/handler/lunch_handler.go`

Loop 2 (後半)。`handler` 層がそもそも何をする層なのかを、自分の言葉で言えるようにする。

---

## ◆ 比較読み (Comparison Read) — コードを開く前に書く

次の機能を呼び出すって感じだった気がする。
それだけでいいかな？

> ⚠️ `internal/handler/lunch_handler.go` を開かずに、まず予想を書く。
> ヒントは `cmd/bot/main.go` で `handler` がどう使われていたかの記憶だけ。
> 質より量、勢いで埋める。

### Q1. handler ってそもそも「何をする層」だと思う？

これは、次の層を呼び出すっていみだと思うけど、それ以上はわからないんだよね

> 自分の言葉で1〜2行。`service` や `repository` との違いを意識して。
> (`notes/02_cmd_bot.md` の最後で書いた "層の役割" を思い出していい)

(ここに予想)

### Q2. `LunchHandler` はどんな状態 (フィールド) を持っていそう？

これもなにもわかりません。
次の機能を呼び出すってのをすればいいのかな？

> ヒント: `cmd/bot/main.go` で `handler.New...(...)` に何が渡されていたか思い出す。
> 何個? 何が?

(ここに予想)

### Q3. handler はどんなメソッドを持っていそう？ 名前を勘で書いてみる

わからん。

> recruit / announce のサブコマンドで呼ばれていたことを手がかりに。
> 引数は何を取りそう? 戻り値は?

NewLunchHandlerは
svcを受け取って、\*LunchHandlerを返す。
これは名前からも新しいsvcを作ってかえすって感じかな

Recuruitは、RunRecruitを返すメソッド
Announceは、RunAnnnouceを返すメソッド

### Q4. handler のメソッドの中身は何行くらいになりそう?

呼び出すだけなら、そこまで長くはなさそう。

> 1行? 5行? 50行?
> 「handler は何をやって、何をやらないか」を考えると見当がつく。

(ここに予想)

---

✏️ ここまで書き終えたら Claude に見せる → 実物を一緒に読みにいく

---

## ◆ 実物を読んだあとに埋める (notes 6 項目)

### 1. このファイルが何をするか (1行サマリ)

mainの識別結果に応じて、必要な機能（Recruit,Annnouce）を呼び出すための機能だね.

(あとで)

### 2. 主な型 (struct, interface)

structとして、LunchHandlerを使用している。

それと、メソッドとして、Recruit()とAnnouce()だね。

### 3. 主な関数 (引数・戻り値・1行説明)

あとは関数で、NewLunchHandlerっていうものがある。

> recruit / announce のサブコマンドで呼ばれていたことを手がかりに。
> 引数は何を取りそう? 戻り値は?

NewLunchHandlerは
svcを受け取って、\*LunchHandlerを返す。
これは名前からも新しい構造体を作るってことかな？を作ってかえすって感じかな

Recuruitは、RunRecruitを読んで、その戻り値（error）を返すメソッド
Announceは、RunAnnnouceをよんで、その戻り値（error）返すメソッド

### 4. 自分が引っかかった所

関数とメソッドがそれぞれ何をしているかわかんないところ
これに関しては、まずメソッドには、構造体がいる

structに関してなんだけど、ここのsvc \*service.LunchServiceってのがなにを意味しているかがまったくわかりません。

それとメソッドに関しては、return h.svc.RunRecruitってのがなにをやっているのかわからんな

### 5. 他にもありえた選択 (Counter-design)

ここで、まずわけるのかな？RecruitとAnnnouceの関数を！
確かに無駄な層かもしれないとは思ったが、
新しい機能を追加するとなったとき、起動をここにかけばいい！っていうDryな制約を守ることができそうだよね。

> 問い候補: なぜ handler を分けた? service を直接 main から呼んでもよかったのでは?
> 問い候補: handler のメソッド内でロジックが薄い (= 委譲しかしてない) 場合、それは "無駄な層" なのか "必要な層" なのか?

(あとで)

### 6. 次に読む人へのアドバイス (= 未来の自分へ)

まず、この関数とメソッドがなにをしているのかちゃんと確認してみよう！
