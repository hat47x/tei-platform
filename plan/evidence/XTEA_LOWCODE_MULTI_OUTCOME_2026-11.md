# 複数の結末があるとき、どれかを選んだことにしない確認（2026-11-10）

## 対象

- 実装: `internal/command/reference.go`・`reference_store.go`（結末の選択）、`internal/command/invoke.go`・`operation.go`（`OutcomeUndetermined`の記録）、`cmd/tei/cli_view.go`と`internal/studiohost/command.go`（表示）
- 試験: `internal/command/reference_outcome_test.go`
- 対象版: `8d93a8c82`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）、`gofmt`も同じ版

## 見つけた欠陥

`AC-5`の残る未確認「多結果・多重一致の表現」を調べる過程で見つけました。

`capability`は複数の`outcome`を宣言できます（`security.authenticate`は`authenticated`と`denied`の2つ）。ところが参照Executorは

```go
if len(capability.Outcomes) > 0 {
    outcome := capability.Outcomes[0]
    result.Outcome = &outcome
}
```

と、**先頭の結末を返して**いました。さらに`internal/workspace/loader.go`が`Outcomes`を**名前で昇順に並べ替えて**います。つまり「先頭」は**名前の辞書順で最初のもの**であり、どれを返すかを決める規則はどこにもありません。

## 実測

宣言順を変えても結果が変わらないことを確かめました。

| 宣言 | 返った結末 |
| --- | --- |
| `outcome denied` → `outcome granted` | `denied` |
| `outcome granted` → `outcome denied` | `denied` |

宣言順ではなく、名前の辞書順で選ばれています。**結末名を改名すると、能力が返す結末が変わる**状態でした。認証のような「許可/拒否」を表す能力では、これは業務上の判断を実装が勝手に決めていることになります。

## 変更の要点

選択規則を発明せず、**決まっていないことを決まったように報告しない**ようにしました。

- 結末が1つのときだけ、その結末を返します（選ぶ余地がないので規則が要りません）。
- 結末が0個のときは、従来どおり結末なし（「宣言が無い」であって「未確定」ではありません）。
- 結末が2つ以上のときは、結末を返さず`OutcomeUndetermined`に理由を入れます（「能力XはN個の結末を宣言しており、どれを選ぶかを述べた規則がない」）。
- 同じ扱いを`ReferenceDataExecutor`にも入れました（2箇所目も同じ欠陥でした）。
- `Result`と`OperationRecord`、CLIのJSON・テキスト、Studioの実行結果に`outcomeUndetermined`を載せました。欠けている理由が読めるので、結末の不在を「結末が無い能力」と取り違えません。

## 実行

```powershell
go test ./internal/command ./internal/studiohost ./internal/hook ./internal/validation -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/command ./cmd/tei ./internal/studiohost
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- `TestReferenceExecutorDoesNotChooseAmongSeveralOutcomes` 成功。2つの結末を持つ能力で結末が返らず、理由に能力名と個数が入る。修正前は先頭（辞書順）を返していた。
- `TestReferenceExecutorReturnsTheSingleOutcome` 成功。1つだけのときは従来どおり返る。
- `TestReferenceExecutorWithNoOutcomeHasNoReason` 成功。結末を宣言しない能力は「未確定」ではない。
- `TestInvokerRecordsWhyNoOutcomeWasDetermined` 成功。理由が操作記録に残る。
- **実測（修正前→後）**: 2つの結末を持つfixtureで、修正前は`"outcome":{"name":"denied"}`、修正後は`outcome: not determined (capability sec.login declares 2 outcomes and no declared rule selects one)`。
- `internal/command`・`internal/studiohost`・`internal/hook`・`internal/validation`・`cmd/tei`はすべて`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`の`tei validate`は`ok`。

## 読み取れること

- 複数の結末を持つ能力は、参照実行で結末を返さない。理由が機械可読に残る。
- 結末が1つの能力は従来どおり動く。

## 読み取れないこと

- **結末を選ぶ規則そのもの**（どの条件で`authenticated`か`denied`か）。これは意味の設計で、実装が発明するものではありません。決まっていないので、決まったふりをしない形にしました。
- `security.authenticate`のような実在の2結末能力をコマンド境界から実行する経路。現在それを実行する試験は無く、`legacyauth`のアダプタが`capabilityinvoke`で結末を明示的に返す別経路を使っています。
- 実Providerでの結末、プロセスを跨ぐ経路での検査。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-5`の「多結果・多重一致の表現」について、**選択規則が無いことを選択規則があるように見せない**形へ直した記録です。規則そのものは未決定のまま残ります。
