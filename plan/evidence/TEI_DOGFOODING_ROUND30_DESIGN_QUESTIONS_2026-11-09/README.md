# trace の outcome 辺と未解決参照を、設計の問いで確かめた（2026-11-09、第30ラウンド）

- 対象: [ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3884](https://github.com/hat47x/tei-platform/issues/3884)（前ラウンドで解消）。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の実測。新しい欠陥は検出しなかった。**

## 1. 目的

前ラウンドで `trace` が capability の `outcome` を辺として扱うようになった。その結果を使って、**設計上の問いに CLI だけで答えられるか**を確かめる。24ラウンドまでで「`inspect` と `trace` の語彙が揃っていない」型を直したので、その効果を測る回でもある。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `8492e296a009de76e81b493b53372bc23daf8f9e`（`origin/main`） |
| CLI実行物 | `sha256:a9d58f8b926902a843e232b2a34fbecac39f0a404aad77223e4e2ac4df59867d` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 設計の問いと答え

### Q1: 認証→認可の連鎖は繋がっているか

**繋がっている。**

```text
security.authenticate: total=4 unresolved=0
   outcome -> security.authenticated_subject
   outcome -> security.authentication_denial
   input   -> std.identifier
   input   -> security.credential

security.authorize: total=5 unresolved=0
   outcome -> security.authorization_grant
   outcome -> security.authorization_denial
   input   -> security.authenticated_subject     ← 認証の outcome と一致
   input   -> security.authorization_action
   input   -> security.authorization_resource
```

`security.authenticated_subject` を、`authenticate` が `outcome` で生み `authorize` が `input` で受ける。前ラウンドの修正前は後者しか見えなかった。

### Q2: モデルに宙に浮いた参照はあるか

**無い。** catalog の宣言70件に対し、`trace` が返す辺の到達先で未宣言のものは0件。`semantic-capability`・`semantic-data`・`semantic-validation` の全 subject で `unresolved=0`。

### Q3: 各 capability の入力に生産者はいるか

**この問いは私の立て方が悪かった。** 「入力に生産者がいなければ設計の穴」という仮説で走査すると `security.authenticate` の `std.identifier`・`security.credential` が「生産者なし」と出る。しかし入力は**モデル外（利用者・外部システム）から来てよい**もので、生産者を要求するのは誤り。`sales.order.submit` は `input order sales.order` と `output sales.order` を両方持ち、自分自身が生産者として現れる（これも正常な往復）。

**入力の生産者不在は設計の穴ではない。** この仮説は棄却した。

### Q4: 解決できない outcome payload を、ツールは検出するか

**検出する。** ただし私が最初に見た場所では見えなかった。

```tei
namespace app

type Known from std.text

capability do_it {
    input x app.known
    outcome done app.not_declared_anywhere
}
```

| コマンド | 結果 |
| --- | --- |
| `validate` | exit 1 `UNKNOWN_SEMANTIC_CAPABILITY_OUTCOME_PAYLOAD`（正しい） |
| `inspect` | `outcomes[0].payload = app.not_declared_anywhere`（宣言どおり返す） |
| `trace` | `result.trace.total: 1`・`result.trace.unresolved: 0` — **outcome 辺は出ない** |
| `trace` | `result.projectionUnresolved: ["unresolved-semantic-element:app.not_declared_anywhere"]` — **ここに出る** |

**私は最初「trace が未解決を報告していない」と読み、欠陥だと考えた。** しかし `result.projectionUnresolved` に正しく出ており、`docs/getting-started.md:212` に説明もある。

```text
`projectionUnresolved`は、派生索引を作った時点で解決できなかった参照を示します。
関係の`resolved`は宣言の解決に成功したことのみを意味し…
```

`reference/workspace` でも 17 件の `projectionUnresolved`（`unmateralized-from:database-contract:*` など）が出ており、機構は正常に動いている。

**したがって欠陥ではない。** 私の読み違いだった。

## 4. 起票しなかった理由

Q3 の仮説は棄却、Q4 は既存の仕組みで正しく報告されていた。**新しい欠陥は検出していない。** 重複Issueを作らない方針に従い、起票しない。

## 5. ただし1点、発見可能性に所感がある

`trace` の `result.trace.unresolved` は**行の未解決**（`resolved`/`unresolved` の status を持つ行の数）で、`result.projectionUnresolved` は**投影時に脱落した辺**である。名前が似ていて、私は取り違えた。

- 前者は「選択した subject に直接繋がる辺のうち、解決できなかったもの」
- 後者は「グラフ全体を作る過程で捨てた参照」

`inspect` ユーザーが `trace` を読むとき、`unresolved: 0` を見て「未解決は無い」と結論しうる。実際には `projectionUnresolved` に17件ある（`reference/workspace`）。これは**欠陥ではなく語の衝突**だが、読み違いを招く。契約文書 `plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の `trace` 行に両者の違いを明記する価値はある。**#3419 の範疇**として所感を残すに留める。

## 6. 読み取れないこと（NOT RUN）

- `projectionUnresolved` の 17 件（`database-contract`・`database` 系）が「未実装だから」か「意図的にモデル外」か。
- 入力の生産者不在を**設計上の警告にすべきか**（Q3 で棄却したが、外部入力と内部欠落を区別する宣言が無いため、機械的には判定できない）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 7. 位置付け

前ラウンドの修正（`outcome` 辺）が意図どおり効いていることを、実際の設計の問いで確認した。**欠陥は見つからなかった**が、以下の3点を記録として残す。

1. `outcome` 辺により認証→認可の連鎖が両側から見える。
2. モデルに宙に浮いた参照は無い（70宣言・未解決0）。
3. 「入力に生産者が無い」は設計の穴ではない（私の仮説は誤り）。
4. `trace.unresolved` と `trace.projectionUnresolved` は別物で、取り違えやすい。

**このラウンドは「測って問題が無かった」ことの記録であり、それが目的の一部でもある。** ドッグフーディングは欠陥を出すためだけではなく、道具が設計の問いに答えられることを確かめるために行っている。
