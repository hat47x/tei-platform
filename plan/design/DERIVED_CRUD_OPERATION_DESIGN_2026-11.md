# Data-first から標準CRUD・検索・保守画面までの導出（設計候補）

- 状態: **提案中（未採用）**
- 対応課題: [#1127](https://github.com/hat47x/tei-platform/issues/1127)（Critical Product Gap: Data-first から標準CRUD/検索/保守画面までの即時実行）
- 前提: [操作面の導出と、操作ごとの自律区分](OPERATION_SURFACE_DESIGN_2026-11.md)（以下「操作面の設計」）
- 関連: [Application Pattern v1](APPLICATION_PATTERN_V1_DESIGN.md)、[業務副作用の段階・確定後効果](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)、#1129、#1130、#1640（D2）、#1124（統合モデラー）、#1803（設計成果物）
- この文書の役割: #1127の「初期デモ」と「受入」を、実装できる粒度まで決める。操作面の一般規則は操作面の設計に置き、ここでは標準CRUDに固有の決定だけを書く。

## 1. 現在できていないこと（2026-11-01 時点の main で確認）

`order_entry` には次の宣言がある。

```tei
pattern order_entry.order_record record {
    data sales.order
}
```

この宣言は「アプリケーションが `sales.order` を一件として扱う」ことだけを意味し、[Application Pattern v1](APPLICATION_PATTERN_V1_DESIGN.md) が明記するとおり、項目・入力規則・更新可能性・遷移・押せる操作を持たない。Workspace Validation は参照先の存在と種別ごとの field の可否だけを検査する（[`internal/validation/application_pattern.go`](../../internal/validation/application_pattern.go)）。したがって現在は、次のどれも宣言からは決まらない。

- 一覧に出す項目、並び順、件数の上限
- 一件参照のキーと、キーが無い場合の振る舞い
- 登録・更新の入力項目と、更新してよい属性
- 同時更新の検出方法
- 削除してよいか、廃止（状態変更）にすべきか
- 一覧から参照へ、参照から更新へ進む遷移

この状態では、データを定義しただけでは、一覧も入力画面も出ない。一覧を出すには Query とパターンの宣言を別に書き、入力画面はさらに実行先の実装を要する。TEIの最初の利用者価値（データを書いたら業務画面が動く）が、もっとも早い地点で止まる。

## 2. 何を導出するか

`data`、`persist`、`candidate-key`／`primary-key`、`validation`、アプリケーションのデータソース別名から、次の標準操作を導出する。導出は読み取り専用であり、正本を増やさない。

| 操作 | 導出条件 | 入力 | 効果 | 自律区分の初期値 |
| --- | --- | --- | --- | --- |
| `list` | `persist` があり、データソースが読取り可能 | なし（上限のみ） | なし | `unattended` |
| `read` | `persist` と、候補キーまたは主キーがある（候補キーを優先） | キー属性 | なし | `unattended` |
| `create` | `persist` があり、データソースが書込み可能で、キーが生成可能または入力可能 | キー以外の必須属性 | create | `attended` |
| `update` | `persist` とキーがある（同時更新は期待状態の比較を既定とする。仮説H1） | キー＋更新可能属性 | update | `attended` |
| `retire` | 状態属性と、廃止に対応する `capability` の効果が宣言されている | キー＋理由 | update（状態） | `attended` |

導出の細部。

- **一覧**は `read` と同じキーの昇順、既定の上限（例: 200件）で返す。検索条件・並び替え・集計は導出しない（#1640 D2の担当）。上限に達した場合は「打ち切った」ことを結果に含め、空の結果と区別する（CLI C1の `summary` と `full` の区別と同じ規律）。
- **一件参照**はキーが一意であることを要求する。主キーも候補キーも無い場合は `missing-semantics` の gap とし、行の全列一致で代用しない。
- **登録**の入力は、`persist` にある列のうち、キーが自動生成でないものを必須、`optional` の列を任意とする。`validation` は対象データまたは属性に結び付いているものを前提として付ける。
- **更新**は、同時更新の検出方法を要求します。PoCの仮説H1として、**期待状態の比較を既定**とします（[PoCの仮説](../workstreams/POC_HYPOTHESES_2026-11.md)）。呼出し側はキーと、読んだときの値を送り、一致した場合だけ書きます。版属性による比較は、業務が版番号の一致を要求した時点で足す別方式です。どちらも無い状態を「後勝ちでよい」と解釈しません。
- **廃止**は削除ではない。`data` に状態属性があり、その状態へ遷移する `capability` が `effect update` を持つ場合だけ導出する。削除は、依存する `data` が無く、かつ削除を明示した場合に限る。依存がある場合の削除は `conflict` として返す（参照整合性を暗黙に壊さない）。
- **論理削除・監査列・採番**は導出しない。#1129の効果規則で宣言する（3章）。

導出の手順（疑似コード）。順序まで決めておくと、実装と試験の期待値が一致します。

```text
for each application A:
  for each datasource alias a in A:                       # a.access: read | write | read-write
    db = database definition bound by a.target
    for each data D with a persistence entry p in db:
      if a.access == "write": gap(list, conflict); continue   # 書込み専用から読めない
      key = first candidateKey of D, else p.primaryKey
      emit list(D)                                        # unattended
      if key == null: gap(read, missing-semantics)
      else:           emit read(D)                        # unattended
      if a.access == "read": continue
      emit create(D)                                      # attended
      if key != null and concurrencyMethod(D) != null: emit update(D)
      else:                                            gap(update, missing-semantics)
      dependents = datas in any db that declare a foreign key to D,
                   plus relations whose target is D
      if dependents is not empty and lifecycle(D) == null: gap(retire, missing-semantics)
      if lifecycle(D) != null:                             emit retire(D)

sort operations by (target, kind rank)   # kind rank: list < read < create < update < retire
```

`emit` は操作面へ追加することを意味し、正本を変更しません。`gap` は理由付きで未解決へ追加します。

- キーは、業務上の識別子である候補キーを優先し、無い場合に物理の主キーを使います。どちらを使ったかを `basis` に残します（両者は同じものではないため）。
- `dependents` は `persist` の外部キーと `relation` の両方から求めます。片方だけを見ると、参照整合性の検査が抜けます。

## 3. #1129・#1130との分担

| 関心事 | 担当 | この文書での扱い |
| --- | --- | --- |
| 書込み前の変更、連鎖更新、確定境界、確定後効果 | #1129（[業務副作用の設計](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)） | 標準操作は「一つの対象への一回の効果」だけを持つ。採番・履歴・通知は効果規則で足す |
| 一括・夜間の実行、再開、二重起動 | #1130 | 標準操作を束ねる親の操作として宣言する。標準操作自体は一括の意味を持たない |
| 検索条件・並び替え・集計・ページ分割 | #1640（D2） | 導出しない。`out-of-scope` の gap として明示する |
| 画面の配置・部品・レスポンシブ | 実行先（UI構成キット #997） | 導出した操作を提示するだけで、配置は決めない |
| 全体の編集・投影・受理 | #1124（統合モデラー） | 操作面は読み取りビュー。編集は #1124 の受理経路を通す |

## 4. 画面への提示（Web の最初の経路）

導出した操作を、既存の五つのパターンに対応づけて提示する。**パターンの宣言を変えずに**、足りない情報を操作面から補う。

| パターン | 提示 | 補う情報 |
| --- | --- | --- |
| `browse` | 一覧 | 列（`persist` の列順）、キー順、上限、打切り表示 |
| `record` | 参照・登録・更新の三態 | 項目、必須・任意、検証、同時更新の検出 |
| `reference` | 親から子への参照リンク | 参照先の `read` 操作 |
| `header-detail` | 見出しと明細 | 明細の `list`、明細の追加・削除（宣言がある場合だけ） |
| `command` | 実行ボタン | 前提となる検証、確認要否、結果の表示 |

提示は、既存の第一Web実行先（`tools/reference-first-web-target`）の経路に載せる。新しい Web の仕組みを作らない。

## 5. 受入

### 5.1 正常系（P-1127-01〜03）

| 番号 | 状況 | 合格の条件 |
| --- | --- | --- |
| P-1127-01 | `order_entry` の操作面を出力する | `sales.order` に `list`・`read`・`create`・`update` が `derived` で現れ、`gaps` が空でない場合は理由が `missing-input`／`missing-semantics`／`out-of-scope` のいずれかである |
| P-1127-02 | 一覧 → 一件参照 → 更新 → 独立した読取りで観測する | 更新前の値、更新後の値、観測値が一致し、更新は `update` 操作の入力だけを使う |
| P-1127-03 | 一覧の件数が上限を超える | 打ち切ったことと総件数（取得できた場合）が結果に含まれ、空の一覧と区別できる |

### 5.2 負例（成功に丸めない）

| 番号 | 状況 | 期待する結果 |
| --- | --- | --- |
| N-1127-01 | 主キーも候補キーも無いデータ | `read` と `update` が `missing-semantics` の gap になり、操作面に現れない |
| N-1127-02 | 読み取り専用のデータソースに `record` パターンがある | `create`・`update` が `conflict` になり、受理を止める |
| N-1127-03 | 期待状態を伴わない更新要求 | 拒否し、後勝ちで上書きしない（仮説H1） |
| N-1127-04 | 他データから参照されているデータの削除 | `conflict`。状態遷移による廃止だけを候補にする |
| N-1127-05 | `validation` に違反する入力 | 業務結果（拒否）として返し、実行エラーにしない（ADR-0020） |
| N-1127-06 | 意味を変更した後に、古い操作面で実行する | `OPERATION_SURFACE_STALE` で停止し、暗黙に最新の意味で実行しない |

### 5.3 検証コマンド

```sh
go test ./internal/operations -run 'OperationSurface' -count=1
go run ./cmd/tei operation surface -workspace reference/workspace -application order_entry
go run ./cmd/tei validate -workspace reference/standard-crud/workspace
go run ./cmd/tei scenario -workspace reference/workspace -scenario reference/driver-scenarios/order-entry-derived-crud.json
```

## 6. 作業分解

| # | 作業 | 変更する場所 | 完了条件 | 規模 |
| --- | --- | --- | --- | --- |
| W1 | 操作面の型と導出の骨格 | `internal/operations/surface.go`、`surface_test.go`（新規） | `data`・`persist`・キーから `list`／`read` が決定的に導出され、順序が安定する単体試験 | M |
| W2 | ギャップと衝突の分類 | 同上 | `missing-input`／`missing-semantics`／`conflict`／`out-of-scope` を返し、空の成功にしない試験 | S |
| W3 | 登録・更新・廃止の導出条件 | 同上 | 版属性あり／なし、読み取り専用、依存ありの四つの負例が期待どおりの分類になる | M |
| W4 | 入力項目と検証の結び付け | `internal/operations/surface.go`、`internal/validation/application_pattern.go`（参照のみ） | `validation` が対象データ・属性に結び付く場合だけ前提に付き、結び付かない場合は gap になる | S |
| W5 | CLI と出力契約 | `cmd/tei/cli_surface.go`（新規）、`cli_contract.go`、`cli_result.go` | `tei commands` に現れ、`schemaVersion` と失敗時のエンベロープが既存の形式に揃う | M |
| W6 | 受入用の負例作業領域 | `reference/standard-crud/workspace/`（新規） | 5.2 の N-01〜N-05 を一つの作業領域で再現でき、`validate` の診断コードが固定される | S |
| W7 | 一覧 → 参照 → 登録 → 更新の縦切り（[#2485](https://github.com/hat47x/tei-platform/issues/2485)） | `internal/driver/bound_write_test.go`、`reference/standard-crud/` | 束縛ありの create・update が能力呼出しを通り、独立した観測で結果を確かめられる。Scenario の語彙追加は別の判断 | M |
| W8 | 第一Web実行先への提示 | `tools/reference-first-web-target/`（参照のみ、必要なら小さな追加） | P-1127-02 が Web 経由でも同じ結果になる。表示の忠実度は判定しない | M |

W1〜W4 は独立して進められる。W5 は W1〜W4 の後、W6 は W3 と並行、W7 は W3・W5 の後、W8 は W7 の後。

**実装状況（2026-11-01）。** W1・W2 に加え、W3（`create`・`update`・`retire` の導出）と W4（検証の結び付け）、W6（負例の作業領域）を追加しました。

- `create` は、書込み可能な別名・保存の対応・識別キーがある場合に導出し、入力（保存列の順）、前提（アプリケーションが宣言した検証のうち、そのデータの属性に付くもの）、効果（`create:<data>`）、自律区分（`attended`）を付けます。
- `update` は、仮説H1に従い、期待状態の比較（`concurrency: expected-state`）を付けて導出します。版属性による比較は、必要になった時点で足す別方式です。
- `retire` は、そのデータへの更新の意図（`effect update`）があるのにライフサイクルの宣言が無い場合に、`missing-input` の未解決として返します。意図が無ければ何も返しません。
- 同じ保存先へ複数の別名から届く場合、操作と未解決を重複させません。

`go test ./internal/operations -run Surface` が通り、参照用作業領域 `order_entry` では `list`・`read`・`create`・`update` が出ます。書き込みは、同じ効果を宣言した能力への `binding` として実行へ渡します（[PoCの仮説](../workstreams/POC_HYPOTHESES_2026-11.md) H4）。負例の作業領域では `sales.order.register`（create）と `sales.order.cancel`（update）を実現へ解決して実行し、期待状態の不一致が `outcome conflict` の業務結果として返り、行が変わらないことを合成の保存領域で確かめました。実DB・並行トランザクションの競合は未検証です。負例の作業領域は [`reference/standard-crud/`](../../reference/standard-crud/README.md) にあり、識別キーの欠落・ライフサイクルの欠落・書込み専用の別名を固定しています。W5（CLI）、W7・W8（Scenario・Web）は未着手です。

## 7. 停止条件

- 操作面を実行の可否判定に使い始めた場合は停止する（操作面の設計の15章が未決のため）。
- 導出した `update` から `concurrency` を落とした場合は停止する。
- 削除を、依存関係の検査なしに提供した場合は停止する。
- 画面の項目・配置を、操作面の外（実行先の設定）へ意味として持ち込んだ場合は停止する。

## 8. 効果と撤回

- 効果: データ定義から、一覧・参照・登録・更新までの最短経路が、画面の実装なしに成立する。初回の業務モデル変更（#1161）と First Value UX（#1168）の測定で使える土台になる。
- 観測: データ定義から最初の一覧表示までの手作業の数、導出された操作のうち実際に使えた割合、gap の理由の分布。
- 撤回: 参照作業領域で `missing-semantics` が支配的で、操作面が「宣言が足りない」以外を出さない場合は、CRUDの導出を取りやめ、`browse`／`record` の宣言に項目を持たせる案（#1124の編集機能側）へ戻す。

## 9. Owner判断が要る点

1. 標準操作の `update` に、版属性の宣言を必須にするか（本稿の案）、それとも既定で期待状態の比較を使うか。前者は意味の追加、後者は実行コストの増加を招く。
2. `retire` を標準操作に含めるか、それとも #1129 の効果規則だけに任せるか。
3. 一覧の既定上限（200件）を、製品の既定値として固定してよいか。
