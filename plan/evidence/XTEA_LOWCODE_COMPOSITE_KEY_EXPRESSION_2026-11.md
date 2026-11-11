# 複合キーの表現（2026-11-11）

## 対象

- 読取り対象: `internal/validation/application_pattern.go`（`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DUPLICATE`、Data型パラメータの対応検査）、`internal/hook/calls.go`（`BindInputsFromReads`）、`internal/command/reference_store.go`（`recordKeyFields`）
- 対象版: `342d58c79`（`main`）
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 背景

`AC-2`の残る未確認に「複合キーを1つのData型パラメータへ渡す表現」がありました。TEIが複合キーをどう表すかを実測しました。

## 実測した事実

### 1. 1つのパラメータへ2つの属性を結ぶことはできない

`sales.order`型のパラメータ`order`へ`sales.order.order_id`と`sales.order.line_no`の両方を結ぶ宣言を書くと、検証が拒否します。

```text
error APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DUPLICATE
  record pattern order_entry.order_record binds command sales.submit_order
  input order more than once
```

**同じパラメータ名を2回結ぶことはできない**ため、「1つのData型パラメータに複合値を入れる」表現は存在しません。

### 2. 複合キーは「属性ごとに1パラメータ」で表す

`sales.order.order_id`を`order`へ、`sales.order.line_no`を`line`へ結ぶ宣言は検証を通り（`ok`）、実行できます。

```json
"inputs":[
  {"name":"order","type":"std.identifier","attribute":"sales.order.order_id","value":"ORD-1","provided":true},
  {"name":"line","type":"std.integer","attribute":"sales.order.line_no","value":"2","provided":true}
]
```

### 3. 適用時のキーは自動的に複合になる

`-reference-store`と`-operation-log`で適用内容を記録しました。

```json
"applied":[{"kind":"create","data":"sales.order","key":"line=2\u0000order=ORD-1"}]
```

`recordKey`（`KeyFields`が空のとき）が**入力の全パラメータ**からキーを作るため、複合キーは「複数のパラメータ」として自然に表現されます。`KeyFields`を指定すれば、その部分集合でキーを作れます（`reference_store_composite_test.go`が`[]string{"order","status"}`で確認）。

### 4. フックの`reads`からも同じ形で結ばれる

`hook.BindInputsFromReads`は`command`の`input`が述べる対応で`reads`をパラメータへ写すため、複合キーは**フックが2つの属性を読む**ことで表せます。

## 判断

「複合キーを1つのData型パラメータへ渡す表現」は**存在せず、必要もない**と確認しました。TEIの表現は次の2つです。

1. **属性ごとに1パラメータ**（`input order sales.order.order_id` と `input line sales.order.line_no`）
2. 適用キーの範囲は`ReferenceDataExecutor.KeyFields`で選ぶ

1つのData型パラメータへ複合値を渡す案は、**パラメータ名の重複拒否**（`..._INPUT_DUPLICATE`）と正面から衝突します。この拒否を緩めるのは意味の設計判断であり、本記録はそれを先取りしません。

## 読み取れること

- 複合キーは複数パラメータとして宣言でき、検証・実行・フックの`reads`結び付けのいずれでも通る。
- 適用キーは既定で全パラメータの複合、`KeyFields`指定で部分集合になる。
- 1パラメータへの複数属性の結び付けは検証が拒否する。

## 読み取れないこと

- **1つのData型パラメータで複合値を運ぶ表現**を将来許すか（意味の設計判断。現状は拒否）。
- 複合キーの**一意性制約**を耐久Storeでどう表現するか（実装待ち）。
- `KeyFields`をホスト（CLI・Studio）から指定する経路（どちらも指定していません）。

## 位置付け

参照実装の確認記録であり、製品の受入や本番適格性を示しません。「複合キーの表現」が既に**複数パラメータ**として成立していることを示し、`AC-2`の残る未確認から本項を外す根拠にします。
