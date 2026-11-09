# 能力入力の結び付けのData対応と、1属性の複数パラメータへの束ねの確認（2026-11-10）

## 対象

- 実装: `internal/validation/application_pattern.go`（`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH`）
- 試験: `internal/validation/application_pattern_binding_type_test.go`、`internal/hook/calls_test.go`
- 対象版: `0f4a5c827`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 見つけた欠陥

能力のパラメータの型がData（`sales.order`など）を名指すとき、結び付ける属性が**そのDataの属性であるか**を見ていませんでした。

実測: `sales.order`型のパラメータ`order`へ、`sales.customer.customer_id`（別のDataの属性）を結んだfixtureは`tei validate`で`ok`。Command境界は値の形だけを見て、どのDataに由来するかを見ないため、**別のレコードの値を正しい型として受理**します。

## 変更の要点

- パラメータ型が宣言済みのDataを名指すとき、結び付けた属性がそのDataの属性かを確かめ、違えば`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH`で拒否します。
- 診断文は「どのDataの、どの属性へ結んだか」を示します。
- 報告の順序を直しました。Dataの食い違いは、既存の「PatternのDataの属性か」という一般的な検査でも同じ結び付けで失敗します。一般的な診断を出して`continue`した後にDataの検査を置くと、より具体的な原因へ届きません。Dataの検査を先に置き、具体的な原因を報告します。
- 試験を2件追加。既存の`TestValidateBindingLeavesADataParameterAlone`は「Dataパラメータは比較しない」という趣旨でしたが、今回の規則により**同一Dataの属性は受理し、別Dataの属性は拒否する**形へ意味が具体化しました。名前と意図を保ったまま、隣に受理・拒否の対を足しています。

## あわせて確認したこと（実装変更なし）

「1つの属性を複数のパラメータへ束ねる規則」は**既に動いていました**。`BindInputsFromReads`はパラメータ名で束ねを保持するため、同じ属性を指す2つのパラメータは2つの束ねとして残ります。

実測: `input original sales.order.order_id` と `input copy sales.order.order_id` を持つfixtureで、フックの`reads sales.order.order_id`が両方へ届き、`called:true`。`tei validate`は`ok`。

仮定で「未実装」と書かず、試験で固定しました（`TestBindInputsFromReadsFansOneAttributeToTwoParameters`）。同じパラメータを二度結んだ場合は最初の宣言を採ることも固定しました（`TestBindInputsFromReadsKeepsTheFirstBindingForOneParameter`）。

## 実行

```powershell
go test ./internal/validation ./internal/hook ./internal/command ./internal/studiohost -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/validation ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
go run ./cmd/tei validate -workspace examples/xtea-lowcode
```

## 観測

- `TestValidateBindingAcceptsAnAttributeOfTheParameterData` 成功。`sales.order`パラメータへ`sales.order.order_id`を結ぶ形は通る。
- `TestValidateBindingRefusesAnotherDatasAttribute` 成功。`sales.customer.customer_id`を結ぶ形は`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH`で拒否され、診断文が`sales.order`と`sales.customer.customer_id`の両方を示す。
- **実測（修正前→後）**: 別Dataの属性を結んだfixtureが、修正前は`ok`、修正後は上記の診断で停止。
- `TestBindInputsFromReadsFansOneAttributeToTwoParameters` 成功。1つの読み取りが2つのパラメータへ届く。
- `TestBindInputsFromReadsKeepsTheFirstBindingForOneParameter` 成功。同じパラメータの二度目の結び付けは最初を上書きしない。
- `reference/workspace`・`examples/xtea-lowcode`・既存fixtureはいずれも`ok`（`examples`は`UNBOUND_DATASOURCE`の警告のみ、終了コード0）。過剰な拒否は出ていません。
- `internal/validation`・`internal/hook`・`internal/command`・`internal/studiohost`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。

## 読み取れること

- Data型パラメータへ別Dataの属性を結ぶ宣言は、宣言の段階で拒否される。
- 1つの属性を複数のパラメータへ束ねる形は、既に動く（試験で固定）。
- 同じパラメータを二度結んだ場合は、最初の宣言が採られる。

## 読み取れないこと

- Data型パラメータへ**同一Dataの属性**を結ぶ形が意味的に正しいか（`sales.order`パラメータに`sales.order.order_id`を渡すのは「そのDataの識別子を渡す」という規約であり、宣言された規則ではありません）。受理はしていますが、規約として文書化されていません。
- 実行時の値変換（正規化・丸め）。
- 実Pluginからの供給路、`reads`・`produces`の実データ観測、`after-*`へ到達する実行経路。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「1つの属性を複数のパラメータへ束ねる規則」と「Dataを名指すパラメータと属性の対応の検査」を埋めた記録です。前者は既に動いていたので試験で固定し、後者は新しく検査を入れました。
