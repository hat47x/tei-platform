# `after-*`へ到達する実行経路の確認（2026-11-10）

## 対象

- 実装: `internal/hook/reach.go`（`AppliedChangeReach`・`RunBeforeChange`・`RunAfterChange`・`runReachable`）
- 試験: `internal/hook/after_change_test.go`
- 対象版: `bdeefb38c`（`main`）を基点とした作業ブランチ
- 環境: Windows、Go `1.26.10`（`go.mod`の版）

## 見つけた欠落

`AC-2`の残る未確認「`after-*`へ到達する実行経路」に着手しました。

`Runner.RunReachable`は、宣言された位相を`PhaseOrder`の順に**一回の走査**で実行します。`PhaseOrder`は`before-read`→`before-create`→`after-create`→`before-update`→`after-update`→…の順なので、一回の走査では`after-create`を`before-update`の前に走らせます。ところが**変更は`before-*`と`after-*`の間**に起きます。一回の走査には「その間」がありません。

そのため、参照実行の`ReferenceReach`は`after-*`を一律に到達不能として理由つきでskipしており、**変更を適用する実行経路でも`after-*`を走らせる手段がありませんでした**。

## 変更

### 1. 位相を半分に分けた入口

- `Runner.RunBeforeChange`: `isBeforeChange`が真の位相だけを実行します。
- `Runner.RunAfterChange`: それ以外（`after-*`）だけを実行します。
- `RunReachable`は従来どおり全位相を実行します（内部で`runReachable`に絞り込み関数を渡す形へ抽出し、挙動は変えていません）。

呼出し側は **`RunBeforeChange` → 変更を適用 → `RunAfterChange`** の順に使います。順序は呼出し側が持ち、runnerが推測しません。

### 2. 適用後の到達範囲

```go
type AppliedChangeReach struct{}
func (AppliedChangeReach) UnreachableReason(string) string { return "" }
```

この型は「変更が適用された後」という**事実を表すだけ**で、事実を作りません。適用していないのに渡せば`after-*`が変更の無い時点で走り、「変更後の検証をした」と読めてしまいます。そのため`RunAfterChange`と組にして使う前提を、型のコメントに明記しました。

### 3. 半分を尋ねたら半分だけ報告する

`RunBeforeChange`は`after-*`を**skippedとして報告しません**。呼出し側が尋ねていない半分を「skipした」と報告すると、尋ねていない検査をしたように読めるためです（`RunReachable`が宣言の無い位相を報告しないのと同じ扱い）。

## 試験（4件）

- `TestAfterChangePhaseRunsAfterTheChangeIsApplied` — `RunBeforeChange`で`before-update`が実行され、`command.NewMemoryStore`へ効果を適用し（`Get`で適用を確認）、`RunAfterChange`＋`AppliedChangeReach`で`after-update`が実行される。**これが`after-*`へ到達する実行経路です。**
- `TestAfterChangePhaseStaysUnreachableWithoutAnAppliedChange` — `ReferenceReach`で`RunAfterChange`を呼ぶと`after-update`は`ReasonAfterChangeUnreachable`でskipされ、実行されない。
- `TestBeforeChangeHalfDoesNotReportTheAfterHalf` — 前半を尋ねたとき、後半がskippedにも理由つきにもならない。
- `TestRunReachableStillRunsEveryDeclaredPhase` — 絞り込みの抽出で`RunReachable`の挙動が変わっていない（全位相を順に実行）。

## 実行

```powershell
go test ./internal/hook -run 'TestAfterChange|TestBeforeChangeHalf|TestRunReachableStill' -count=1
go test ./internal/hook ./internal/command ./internal/studiohost ./internal/validation -count=1
go test ./cmd/tei -count=1 -timeout 30m
go vet ./internal/hook
gofmt -l .
go run ./cmd/tei validate -workspace reference/workspace
```

## 観測

- 上記4件すべて成功。
- `internal/hook`・`internal/command`・`internal/studiohost`・`internal/validation`は`ok`。`go vet`と`gofmt -l .`は無出力。`reference/workspace`は`ok`。

## 読み取れること

- 変更を適用した後に`after-*`のフックを実行できる経路がある。
- 適用していない実行では、`after-*`は従来どおり理由つきでskipされる。

## 読み取れないこと

- **ホストの配線**。CLI（`cli_view_hooks.go`）とStudio（`studiohost/hooks.go`）は`ReferenceReach`と`RunReachable`のままです。CLIの参照実行は「何も書かない」と表明しており（`persisted:false`・`persistence:not-applicable`）、適用する経路へ変えるのは製品の意味の変更になります。今回入れたのは**その気になれば繋げる入口**までです。
- `ReferenceDataExecutor`（MemoryStoreへ効果を適用する実行者）は、どのホストからも使われていません（`cmd`・`internal`の参照0件）。試験でのみ使われます。
- `after-read`（読取り後）の位置づけ。`isBeforeChange`は`before-read`だけを前とし、`after-read`は後半に入ります。読取りの変更は無いので、`after-read`を後半に置くのが正しいかは未決定です。
- 実Pluginの`after-*`フック、実Providerでの変更適用。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`after-*`へ到達する**実行経路をフック層に用意し、試験で通した**記録です。ホスト配線と、参照実行の「何も書かない」表明を変えるかどうかは、意味の設計判断として残します。
