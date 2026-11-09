# フックが使う能力の`command` Pattern不在の拒否（2026-11-07）

## 対象

- 実装: `internal/validation/application_pattern.go`（`APPLICATION_PATTERN_INTERACTION_HOOK_USE_WITHOUT_COMMAND`を追加）
- 試験: `internal/validation/application_pattern_hook_test.go`
- 対象版: `79a9a908f`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.27.0`

## 変更の要点

- フックの`uses`が指す能力について、その Application が能力を宣言する`command` Patternを持つかを検証で確かめるようにしました。
- 無ければ`APPLICATION_PATTERN_INTERACTION_HOOK_USE_WITHOUT_COMMAND`で拒否します。能力が実在していても、実行できる経路（Command境界）が無ければフックは走らないためです。
- 既存の試験の固定具に、能力を宣言する`command` Patternを足しました（宣言どおりの前提を満たすように）。

## 実行

```powershell
go test ./internal/validation -count=1
go vet ./internal/validation
gofmt -l internal/validation
go run ./cmd/tei validate -workspace reference/workspace
go run ./cmd/tei validate -workspace examples/xtea-lowcode
```

## 観測

- `TestValidateInteractionHookRejectsUseWithoutCommand` 成功。能力を宣言する`command` Patternを取り除くと、`APPLICATION_PATTERN_INTERACTION_HOOK_USE_WITHOUT_COMMAND`が出る。
- 既存の`TestValidateInteractionHook`（正常系）・位相・読み取り・能力名・失敗時の負例は成功のまま。
- `internal/validation`の全試験が`ok`。`go vet`・`gofmt -l`は無出力。
- `reference/workspace`と`examples/xtea-lowcode`の`tei validate`は`ok`（既存の警告のみ）。`examples/xtea-lowcode`の`order_entry.order_record`は、能力を宣言する`order_entry.submit_order`を持つため、この新しい診断は出ません。

## 読み取れること

- フックの`uses`は、能力名の実在だけでなく、実行経路（`command` Pattern）の実在も要求される。
- 実行できないフック宣言は、実行時に黙って失敗するのではなく、検証の段階で拒否される。

## 読み取れないこと

- `command` Patternが複数ある場合の優先規則。この検証は「1件以上ある」ことだけを見ます。`hook.CommandPatternFor`は複数件を拒否しますが、検証では曖昧さを診断していません。
- 位相と`command` Patternの対応（`before-read`にどのcommandを対応させるか等）。
- 実 Plugin・実 Provider・本番配備。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「宣言していない能力の呼出しを、検証時と実行時の両方で拒否する」のうち、実行経路の不在を検証時に拒否する部分を担います。実行時の拒否は既存の`hook.Declared`が担います。
