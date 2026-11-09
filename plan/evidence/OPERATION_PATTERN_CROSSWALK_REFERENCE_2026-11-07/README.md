# 操作とPatternの突き合わせを、参照実装で試す（2026-11-07）

- 対象: [操作面の導出と自律区分の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の「追記（2026-11-07）: W3 の対応表」
- 位置づけ: **参照実装と実行記録**。W3の採用判断・Owner受理・Gate合否ではない。
- 実行HEAD: 下の`raw/head.txt`。環境はWSL2 Ubuntu 22.04 / Go 1.27.3。

設計の追記は、W3の対応表を**案**として書きました。この記録は、その案を**実装して動かした**結果です。案が正しいという証明ではなく、案が何を読み、何を読まないかを観測できる形にしました。

## 1. 何を作ったか

`tools/reference-operation-pattern-crosswalk`（新規・参照実装）を作りました。

- 入力: 受理済みの`World`（`workspace.Load`）。
- 導出された操作: **既存の`operations.DeriveOperationSurface`**をそのまま使う。二つ目の導出規則を作らない。
- 宣言されたPattern: `application.Pattern`を読む。
- 突き合わせ: Data ID・能力ID・Relation targetの三つだけ。名前の類似・prefix・ロード順は使わない。
- 出力: 一致（`links`）、一致しなかった宣言（`unlinked`、`operation-surface`と同じgap語彙）、同じ操作を取り合う宣言（`conflicts`）。`header-detail`のように複数の操作種別へ対応するKindは`ambiguous`として全一致を並べ、どちらかを選ばない。
- 書込みなし。識別子・操作・受理を`World`へ戻しません。

CLI: `go run ./tools/reference-operation-pattern-crosswalk/cmd/tei-operation-pattern-crosswalk -workspace <root>`

## 2. 何を観測したか

受理済みの`reference/workspace`と、見積もり候補を重ねた複製に、同じCLIを当てました。

| | 受理済み | 候補を重ねた側 |
| --- | --- | --- |
| `order_entry`の操作 | 8 | 8 |
| `order_entry`のPattern | 5 | 6 |
| 一致（`links`） | 4 | 4 |
| 一致しない宣言 | 2 | 3 |
| 衝突 | 0 | 0 |
| `ambiguous` | `order_entry.order_lines` | 同左 |

一致した4件（どちらのWorkspaceでも同じ）:

```text
list.sales.order      <- recent_orders   [browse]        via=data     subject=sales.order
read.sales.order      <- order_record    [record]        via=data     subject=sales.order
list.sales.order_line <- order_lines     [header-detail] via=relation subject=sales.order_line
read.sales.order_line <- order_lines     [header-detail] via=relation subject=sales.order_line
```

一致しなかった宣言:

```text
（受理済み・候補とも）
customer_reference    [reference] missing-input  no derived read operation on customer.customer
（受理済み）
submit_order          [command]   out-of-scope   capability sales.order.submit declares no effect
（候補のみ）
preview_order_quote   [command]   out-of-scope   capability sales.order.quote_lines declares no effect
```

`repair_intake`は操作0件・Pattern 1件で、`customer_reference`が`missing-input`になります。

## 3. この観測が示すこと

- **対応表の案は、実装できる形になっている。** 5種類のKindのうち`browse`・`record`・`header-detail`は受理済みの意味だけで一致し、`reference`と`command`は宣言が足りないことを`unlinked`として返します。既定値で埋めた箇所はありません。
- **`command`は、能力が効果を宣言していないと1件も一致しない。** 受理済みの`submit_order`も、候補の`preview_order_quote`も`out-of-scope`です。設計の追記が「効果を操作面の側で補わない」と書いたとおりの結果です。
- **`reference`は、参照先Dataを読む標準操作が無いと一致しない。** `customer.customer`には`read`が導出されていないため（Persistenceの対応が無い）、`missing-input`になります。Relationの読み替えだけでは足りず、**読み取り経路（`datasource`）に対応する標準操作の有無**が効きます。
- **`header-detail`は`list`と`read`の両方に一致し、`ambiguous`になる。** 案は「両方並べる」と決めたので、ここでは衝突にしていません。どちらか一方だけを選ぶ規則を足すと、この観測は変わります。

## 4. この観測が示さないこと

- **対応表を採用すべきという判断。** 参照実装は案を試すためのもので、W3は**未着手のまま**です。採用するなら規則の版を上げる判断が要ります（設計15章の未決2番）。
- **`unlinked`が欠陥であること。** `submit_order`が`out-of-scope`なのは、能力が効果を宣言していないという宣言の状態の結果です。能力側に効果を宣言すれば変わり得ます。
- **実運用の正しさ。** 合成のWorkspaceと読み取りのみです。実DB・実Provider・権限・実Consumerは見ていません。
- **`ambiguous`の扱いの妥当性。** `header-detail`を`list`と`read`の両方に結ぶのが正しいかは、UIの意図に関わる判断で、この記録では決めていません。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（受理済みと候補の複製を作り、CLIを2回実行） |
| `raw/head.txt` | 実行時のHEADとToolchain |
| `raw/pre-status.txt` | 実行前の作業領域（このディレクトリを除く） |
| `raw/accepted-crosswalk.json`・`raw/overlay-crosswalk.json` | 二つのWorkspaceの突き合わせ結果 |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt`はすべて0バイトです。

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `72a8876a603c5cf03d57ccfb6f0c2b7ff4b8345d2e4aa0974cc8e4cf48014c3b` | 3494 |
| `raw/head.txt` | `7a67e1342194cc3ed31136fd8ca789d9bf2106bff711a0b9c22bce1cb7b92905` | 73 |
| `raw/pre-status.txt` | `e13dab54e065b16beedac67c9fdafc14e055c47581a70e6409fff5067a975ddd` | 216 |
| `raw/accepted-crosswalk.json` | `5e0d1452ad989d9882b63cd272ddc33d8821e08c98a604d3adabceb2f2efc0b3` | 3396 |
| `raw/overlay-crosswalk.json` | `658901b07bd75a7262cd897d3a36f66d53fe6931797bea5c650a42d126167bf5` | 3701 |
| `raw/summary.txt` | `54401fdbf7795e75c89e878af18ca097f690cd6c3a83e0fdb582edf7ab568a2b` | 2523 |

`raw/pre-status.txt`が0行でないのは、**計測時点で新規ツール本体が未追跡**だったためです（`tools/reference-operation-pattern-crosswalk/`の3ファイル）。ツールはこのPRで別途コミット済みですが、**計測は未追跡の状態で1回だけ行い、コミット後の再実行はしていません**。`raw/head.txt`のHEADは計測時の版であり、PRが載る版とは同じではありません。
