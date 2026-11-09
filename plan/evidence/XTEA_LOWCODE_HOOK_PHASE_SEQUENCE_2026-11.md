# フックの位相順序と途中停止の確認（2026-11-08）

## 対象

- 実装: `internal/hook/sequence.go`（`PhaseOrder`・`RunSequence`・`SequenceResult`）と`internal/hook/run.go`（失敗の包みを`wrapFailure`へ寄せた）
- 試験: `internal/hook/sequence_test.go`
- 対象版: `007d39fd4`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.27.0`

## 変更の要点

- `PhaseOrder`として位相の順序（`before-read`→`before-create`→`after-create`→`before-update`→`after-update`→`before-delete`→`after-delete`）を固定しました。宣言順や名前の文字列順では決めません。
- `Runner.RunSequence`は、その順序で宣言されたフックを実行し、`SequenceResult`（`Executed`・`Warned`・`Refused`）でどこまで進んだかを返します。
- 不成立（`NotSatisfied`）または実行の失敗の位相で止まり、以降の位相は実行しません。`on-failure warn`の位相は記録して先へ進みます。
- 順序に無い位相（`during-update`など）は`HOOK_PHASE_UNSUPPORTED`で拒否します。黙って飛ばすと、宣言されているのに実行されないフックが残るためです。
- 失敗の種類を決める処理を`wrapFailure`へ寄せ、`Run`と`RunSequence`が同じ規則を使うようにしました。

## 実行

```powershell
go test ./internal/hook -count=1
go vet ./internal/hook
gofmt -l internal/hook
go test ./internal/hook ./internal/validation ./internal/command ./internal/applicationview -count=1
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestRunSequenceUsesThePhaseOrder` 成功。`before-update`を先に宣言しても、実行は`before-read`→`before-update`→`after-update`の順になる。
- `TestRunSequenceSkipsUndeclaredPhases` 成功。宣言の無い位相は実行されず、記録にも出ない。
- `TestRunSequenceStopsAtARefusal` 成功。`before-update`の不成立で止まり、`Refused`がその位相を指し、`after-update`は実行されない。
- `TestRunSequenceContinuesAfterAWarn` 成功。`on-failure warn`の位相は`Warned`に記録され、後続の位相が実行される。
- `TestRunSequenceRefusesWithoutADeclaration` 成功。フックが一つも無い Pattern は`HOOK_NOT_DECLARED`で拒否される（空の成功にしない）。
- `TestRunSequenceRefusesUnsupportedPhase` 成功。順序に無い位相は`HOOK_PHASE_UNSUPPORTED`で拒否され、何も実行されない。
- `TestRunSequenceRefusesNonRecordPattern` 成功。`record`以外の Pattern は`HOOK_PATTERN_NOT_RECORD`で拒否される。
- 既存の`TestRun...`・`TestDeclared...`・`TestReferenceRule...`・`TestGuarded...`・`TestCommandPatternFor...`は成功のまま。
- `go vet`と`gofmt -l`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- 位相の実行順は、宣言順ではなく固定した順序で決まる。
- 途中で止まった位置は`SequenceResult.Refused`で分かる。
- 実行されない位相（未宣言・未対応）は、成功として扱わない。

## 読み取れないこと

- `RunSequence`をCLI・Studioの実行経路へ配線すること。この試験は`hook.Runner`を直接呼んでいます。
- 位相とトランザクション境界の関係（どの位相が同じ取引に入るか）。
- 複数の規則Pluginが同じ位相に登録した場合の合成順序。
- 実Plugin・実Provider・実データベース。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「同じフック宣言とPluginの版から同じ入力に同じ結果を返す」のうち、実行順序が宣言順に依存しないことを確かめました。
