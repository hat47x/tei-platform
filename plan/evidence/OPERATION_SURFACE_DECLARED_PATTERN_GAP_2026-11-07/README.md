# 操作面は宣言された Pattern を読んでいるか（2026-11-07）

- 対象: [操作面の導出と自律区分の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の「W3 宣言と導出の優先順位」
- 位置づけ: **実行して観測した記録**。意味の受理・Owner判断・Gate合否ではない。
- 実行HEAD: `ea666377`（`origin/main`）。実行環境は`raw/head.txt`（Linux/WSL2 Ubuntu 22.04、Go 1.27.3）。
- 実行前の作業領域: このディレクトリ以外は0行（`raw/pre-status.txt` は空）

## 1. 何を確かめたか

設計の14章は「W1・W2 は実装済み、**W3（上書き）は未着手**」と書いています。W3の完了条件は「`origin`／`overriddenBy` が正しく付き、抑制が来歴に残る試験」です。

一方、[COMPOSITION_NEXT_EDGE_STATIC_SURVEY_2026-11-07](../COMPOSITION_NEXT_EDGE_STATIC_SURVEY_2026-11-07/README.md)で、Aの操作面とBの構成選択が同居していないことを確認しました。Aの操作面が「業務操作の一覧」として使えるかどうかは、**宣言された`pattern`を読んでいるか**に懸かります。読んでいなければ、`order_entry.preview_order_quote`（宣言された`command`）はAの出力に現れず、Bの業務ActionとAのOperationを結ぶ手掛かりがA側に存在しないことになります。

そこで、受注の参照Workspaceと、見積もり候補を重ねたWorkspaceを作り、`tei operations`の出力を比べました。

## 2. 何をしたか

- `reference/workspace`（受理済みの基準）は変更していません。一時ディレクトリへ**複製**し、`reference/candidates/order-quote`の`applications/order-entry/`と`semantic/sales/`を上書きしました。既存の[見積もり認可の試験](../../../reference/web-composition/go-host/canonical_quote_authorization_test.go)と同じ重ね方です。
- それぞれに `tei validate`、`tei operations -application order_entry`、候補側に `tei artifacts -application order_entry` を実行しました。
- 再現は`run.sh`（同じHEADから同じ手順を実行）と`gitdir.sh`（Windowsで作ったworktreeをWSLのgitで読むための補助）です。

## 3. 観測

| 観測 | 受理済みWorkspace | 見積もり候補を重ねたWorkspace |
| --- | --- | --- |
| `validate` | `ok`（警告3件） | `ok`（警告3件） |
| `worldFingerprint` | `sha256:c92de09f…d65a` | `sha256:44a7ce63…c226` |
| `modelRevision` | `sha256:f414e2b0…3fd2` | `sha256:8ee83bef…b94c` |
| 操作数 | 8 | 8 |
| `gaps` / `conflicts` | 0 / 0 | 0 / 0 |
| `summary.declared` | **0** | **0** |
| `origin`の内訳 | `derived: 8` | `derived: 8` |
| 出力に現れる操作ID | `list`/`read`/`create`/`update` × `sales.order`・`sales.order_line` | **同じ8件** |
| `function-definition`の`pattern`行 | （未取得） | **6件**（`preview_order_quote`を含む） |
| `function-definition`の`unresolved` | （未取得） | 4件（`capability declares no effect`） |

候補を重ねると`worldFingerprint`と`modelRevision`は変わり、`function-definition`には`pattern order_entry.preview_order_quote command`が出ます。**しかし操作面の出力は1件も変わりません。**`summary.declared`は0のままで、8件すべてが`derived`です。

つまり現行の`tei operations`は、`pattern`を読んでいません。

## 4. 何が起きているか（静的な所見）

`internal/operations/surface.go`を読むと、`origin`に`"declared"`を代入する箇所はありません（`map`の集計側に`case "declared"`はあるが、生成側は`"derived"`のみ）。`Operation`構造体は`origin`／`overriddenBy`を持ちますが、**宣言された`pattern`から操作を作る経路が無い**という形です。

これは「W3が未着手」という設計文書の記述と一致します。**設計の未着手項目が、そのまま「AとBを結ぶ手掛かりがAの出力に無い」という観測可能な結果になっています。**

## 5. この記録が示すこと

- 現行の`tei operations`は、`pattern`（`command`を含む）を操作面に載せていません。WORLDが変わっても操作面は同じです。
- したがって「AのOperation IDとBの業務Actionを同一identityで結ぶ」作業は、**Aの出力に相手が現れない状態**から始まります。手掛かりは`function-definition`の`pattern`行か、`pattern`の宣言Sourceそのものです。
- 設計文書の「W3未着手」は、少なくともこの観測時点の`main`で事実として一致します。

## 6. この記録が示さないこと

- **W3を実装すべきという判断**。W3は設計候補の作業項目であり、採用・優先順位はOwnerの判断です。本記録は「現状そうなっていない」ことだけを示します。
- **宣言された`pattern`を操作面へ載せると正しいか**。`pattern`の`Kind`（`command`等）と操作の`kind`（`list`/`read`/…)の対応は設計の未決であり、載せ方によっては効果を増やして見せる危険があります。
- **`function-definition`の`unresolved`の意味**。4件の`capability declares no effect`は`sales.order.quote_lines`を含みますが、これは読み取り専用の宣言が「効果なし」として未解決に数えられているという現状の記録であり、能力の欠陥の証明ではありません。
- **実DB・実Provider・権限**。終始、合成のWorkspaceとCLIの読み取りだけです。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（同じHEADから実行） |
| `gitdir.sh` | Windows作成worktreeをWSLのgitで扱うための補助（該当時のみ作用） |
| `raw/head.txt` | 実行時のHEADとToolchain |
| `raw/pre-status.txt` | 実行前の作業領域（このディレクトリを除き0行） |
| `raw/accepted-validate.txt`・`raw/overlay-validate.txt` | 両Workspaceの`tei validate` |
| `raw/accepted-operations.json`・`raw/overlay-operations.json` | 両Workspaceの`tei operations` |
| `raw/overlay-artifacts.json` | 候補側の`tei artifacts`（`pattern`行と未解決を含む） |
| `raw/summary.txt` | 上の表の数値を生成した要約 |

生データの`*.err.txt`はすべて0バイトです。SHA-256は下の表にあります。

## 8. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `c6beff5c545c421fb64893e2409dad039d47bacc2002053d3f46f0769cd6d6bb` | 4094 |
| `gitdir.sh` | `f3144d5bf13aee898081b399545bd4f666368621e6cc1cd00666386d37e992a2` | 1208 |
| `raw/head.txt` | `6ee969764f8cf1cf6ebd078e36f6880544252c3cbaccaf465cde4e049d640bc3` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |
| `raw/accepted-validate.txt`・`raw/overlay-validate.txt` | `2f495c06bc0281ebba5161dd931e25dea0966dc6f704d9a3c52cde7e7e9f22fa`（同一内容） | 505 |
| `raw/accepted-operations.json` | `aae8986f434b97758f4adda183ce3286e24164b879f759fc92dbffed5a832232` | 4208 |
| `raw/overlay-operations.json` | `9e05cce4a08e4994676217a681ef973818ee2faa8b21b245494b313dd4d08bb9` | 4208 |
| `raw/overlay-artifacts.json` | `5d4749d1ab2da987f011122380dc1cc54363e1aae00f55747c070765d2bd926c` | 11198 |
| `raw/summary.txt` | `8c253845aa3503ba510410ba283a5831d0c471c2b0ae5b34fb9665bbff17c759` | 3181 |

`accepted-validate.txt`と`overlay-validate.txt`が同じSHA-256なのは、警告3件がどちらのWorkspaceでも同一だからです（受理済み基準の警告は候補の重ね方では変わらない、という意味に留まります）。

数値は`ea666377`時点のものです。`main`が進むと、`worldFingerprint`・`modelRevision`・操作数は変わり得ます。再生成は同じHEADで`run.sh`を実行します。

