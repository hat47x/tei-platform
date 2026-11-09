# 宣言された Pattern と導出された操作は、どこで結べるか（2026-11-07）

- 対象: [操作面の導出と自律区分の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の「W3 宣言と導出の優先順位」
- 位置づけ: **実行して観測した記録**。結合規則の採否・Owner受理・Gate合否ではない。
- 実行HEAD: `7d92bf382`（`origin/main`）。環境は`raw/head.txt`（WSL2 Ubuntu 22.04、Go 1.27.3）。
- 実行前の作業領域: このディレクトリを除いて0行（`raw/pre-status.txt`は空）

前段の[操作面が宣言Patternを読んでいるか](../OPERATION_SURFACE_DECLARED_PATTERN_GAP_2026-11-07/README.md)で、`tei operations`が`pattern`を読んでいないことを確認しました。この記録は、では**宣言されたPatternと導出された操作を、どの手掛かりで結べるか**を、二つのWorkspaceで観測したものです。

## 1. 何をしたか

`reference/workspace`（受理済み基準）と、見積もり候補を重ねた複製を作り、それぞれに `tei operations`（導出された操作）と `tei artifacts`（`function-definition`の`pattern`行）を実行しました。同じWorldから、**宣言側**と**導出側**の両方の一覧を取ったことになります。

- 受理済み基準は編集せず、一時ディレクトリへ複製して候補を重ねました。
- 再現は`run.sh`（同じHEADから同じ手順を実行）。`APP`環境変数でApplicationを変えられます。

## 2. 観測

| | 受理済み | 見積もり候補を重ねた側 |
| --- | --- | --- |
| `worldFingerprint` | `sha256:c92de09f…` | `sha256:44a7ce63…` |
| `modelRevision` | `sha256:f414e2b0…` | `sha256:8ee83bef…` |
| 導出された操作 | 8 | 8（同一） |
| 宣言された`pattern` | 5 | 6（`preview_order_quote`が増える） |
| `create`/`update`の`binding` | すべて`null` | すべて`null` |

宣言側（`pattern`行）:

```text
reference   order_entry.customer_reference
header-detail order_entry.order_lines
record      order_entry.order_record
browse      order_entry.recent_orders
command     order_entry.submit_order
```

導出側（操作ID）:

```text
order_entry.list.sales.order
order_entry.read.sales.order
order_entry.create.sales.order
order_entry.update.sales.order
order_entry.list.sales.order_line
order_entry.read.sales.order_line
order_entry.create.sales.order_line
order_entry.update.sales.order_line
```

Patternの5種類（`browse`・`record`・`reference`・`command`・`header-detail`）と、操作の4種類（`list`・`read`・`create`・`update`）は、**別の語彙**です。両者を結ぶ規則は、操作面の出力にも`function-definition`の出力にもありません。

## 3. 結べる手掛かり（候補の整理）

宣言側の型付き参照は[Application Pattern v1](../../design/APPLICATION_PATTERN_V1_DESIGN.md)が定めています（`data`・`relation`・`query`・`capability`・`datasource`）。導出側が持つのは`target`（Semantic Data）と、書込みの`binding`（能力ID）です。突き合わせ可能な欄は次の三つです。

| 手掛かり | 宣言側 | 導出側 | 受理済み基準で一致する例 | 一致しない例 |
| --- | --- | --- | --- | --- |
| Data ID | `browse`・`record`の`data` | 操作の`target` | `record:order_record` = `sales.order` と `read.sales.order` | `reference`・`command`・`header-detail`は`data`を持たない |
| 能力 ID | `command`の`capability` | 書込み操作の`binding` | **なし**（`binding`が全件`null`） | `submit_order` → `sales.order.submit`に対応する操作が無い |
| Relation target | `reference`・`header-detail`の`relation`（解決後のtarget Data） | `relation`を直接の`target`に持つ操作は**無い** | **なし** | `sales.order.lines` → `sales.order_line` の操作は派生の`list`/`read`/`create`/`update` |

### Data IDで結べるのは2種類だけ

`browse`（`data sales.order` + `query`）と`record`（`data sales.order`）は、どちらも`target=sales.order`の操作とData IDで一致します。ただし**両方が同じ`sales.order`を指す**ため、Data IDだけでは`list`と`read`のどちらに結ぶかを決められません。操作の`kind`（`list`/`read`）とPatternの`kind`（`browse`/`record`）を対応づける表が要りますが、それは現行のどこにもありません。

### 能力 IDで結べない理由

`command`の`capability sales.order.submit`に対応する導出操作は、受理済み基準では存在しません。`create`/`update`の`binding`が全件`null`なのは、能力が効果を宣言していないためです（効果の宣言が無い能力を書込みの実行主体にしない、という既存の規則どおり）。**`binding`が埋まるのは、能力が同じデータへの効果を宣言したときだけ**です。したがって`command`→操作の結合は、能力の効果宣言という別の前提に依存します。

### Relationを直接の対象にする操作が無い

`reference`（`relation sales.order.customer` + `datasource master`）と`header-detail`（`relation sales.order.lines`）は、どちらもRelationを指します。一方、導出側の`target`は常に**Semantic Data**で、Relationを`target`に持つ操作はありません。Relationのtarget Data（`customer.customer`、`sales.order_line`）へ読み替える規則は、現行の導出規則の語彙（`standard-crud/v1`の規則文）に含まれていません。

### `datasource`と`query`は導出側に対応欄が無い

`reference`の`datasource master`と`browse`の`query order_entry.recent_orders`は、操作面の出力に対応する欄がありません（`list`は`target`と`basis`だけ）。これらは読み取り経路の宣言であり、操作の同一性ではなく**取得経路の選択**に関わる情報です。

## 4. この記録が示すこと

- 宣言されたPatternと導出された操作は、同じWorldから同時に取れるが、**現行の出力どうしを結ぶ欄が無い**。
- 突き合わせ可能な欄は Data ID・能力 ID・Relation target の三つで、それぞれ次の限界がある。
  - Data ID: 一致するのは`browse`/`record`だけで、両者が同じDataを指すため種別の対応表が別に要る。
  - 能力 ID: 受理済み基準では`binding`が全件`null`のため、1件も一致しない。
  - Relation target: 導出側にRelationを`target`に持つ操作が無いため、読み替え規則が要る。
- `command`の`capability`と`create`/`update`の`binding`を結ぶには、能力の効果宣言が前提になる。逆に言えば、**効果を宣言しない読み取り専用の`command`（見積もり候補の`preview_order_quote`）は、構造上どの導出操作とも結ばれない**。

## 5. この記録が示さないこと

- **結合規則を採用すべきという判断**。対応表の案は[設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)側に置き、採否はOwner判断です。
- **`binding=null`が欠陥であること**。効果の宣言が無い能力を書込みの主体にしない、という既存の規則の結果です。能力側に効果を宣言すれば変わり得ます。
- **`pattern`の`Kind`語彙が確定していること**。5種類はv1の参照で成立した語彙であり、永久の標準ではありません。
- **実DB・実Provider・権限**。終始、合成WorkspaceとCLI読み取りのみです。

## 6. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（同じHEADから実行） |
| `raw/head.txt` | 実行時のHEADとToolchain |
| `raw/pre-status.txt` | 実行前の作業領域（このディレクトリを除き0行） |
| `raw/accepted-operations.json`・`raw/overlay-operations.json` | 導出された操作（両Workspace） |
| `raw/accepted-artifacts.json`・`raw/overlay-artifacts.json` | `function-definition`の`pattern`行（両Workspace） |
| `raw/summary.txt` | 上の表を作る元の一覧 |

`*.err.txt`はすべて0バイトです。

## 7. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `26e82784bc268c154486448bf2220db8a1b652923882957357b1ef849d7caae5` | 3529 |
| `raw/head.txt` | `0c3a0e6b2731bb62197a00fb206eeee16f309edf15b153d57be637ab10fa9599` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |
| `raw/accepted-operations.json` | `aae8986f434b97758f4adda183ce3286e24164b879f759fc92dbffed5a832232` | 4208 |
| `raw/overlay-operations.json` | `9e05cce4a08e4994676217a681ef973818ee2faa8b21b245494b313dd4d08bb9` | 4208 |
| `raw/accepted-artifacts.json` | `80666f3bc6d835e563f14a655c238d69b81b9bda2d5faaf4cf9190743a3ef8b4` | 10709 |
| `raw/overlay-artifacts.json` | `5d4749d1ab2da987f011122380dc1cc54363e1aae00f55747c070765d2bd926c` | 11198 |
| `raw/summary.txt` | `158288fff7a2eb7a90c75e21c864b1146588ab82bd08b107fde7b8d2b5aa5295` | 2355 |

数値は`7d92bf382`時点のものです。`main`が進むと`worldFingerprint`・`modelRevision`・一覧は変わり得ます。再生成は同じHEADで`run.sh`を実行します。
