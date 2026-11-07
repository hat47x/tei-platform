# フック実行の配線と失敗の区別の確認（2026-11-07）

## 対象

- 配線: `internal/hook/run.go`（`Runner.Caller`）と`internal/hook/calling.go`（`CallingImplementation`・`NotSatisfied`）
- 宣言の検査: `internal/validation/application_pattern.go`（同じ位相の重複宣言を拒否）
- 規則Pluginの参照実装: `internal/hook/reference.go`（`ReferenceRule`）と`internal/hook/guarded.go`（`Guarded`）
- 委譲先: `internal/hook/command_caller.go`（`CommandCaller`・`DeclaredCommand`。`971bc772b`で`main`へ統合済み。この変更では触れていません）
- 試験: `internal/hook/run_caller_test.go`、`internal/hook/reference_test.go`、`internal/hook/guarded_test.go`、`internal/validation/application_pattern_hook_duplicate_test.go`
- 対象版: `0b36094a8`（`main`）を基点とした作業ブランチ
- 環境: Windows（Go `1.27.0`）、WSL2 Ubuntu（Go `1.27.3`）

## 変更の要点

- `hook.Runner`に`Caller`を追加しました。`Implementation`が`CallingImplementation`（`RunWithCaller`）を実装している場合、Runnerは`Declared{Inner: Caller, Hook: 宣言}`を渡します。フックが到達できる能力は、宣言の`uses`と、Runnerが設定した下位境界の両方で決まります。
- 呼出し先（`Runner.Caller`）が未設定でも、`Declared`は`HOOK_IMPLEMENTATION_UNAVAILABLE`として拒否します。未設定を許可として扱いません。
- 能力を呼ばない`Implementation`は従来どおり`Run`で動きます。新しい契約は任意です。
- `hook.NotSatisfied`を追加しました。フックが業務上の条件を満たさないと判断した場合は、実行の失敗（`Error`・`Warning`）として包まず、そのまま返します。`on-failure warn`でも、不成立は警告に変えません。
- `hook.ReferenceRule`を追加しました。宣言した読み取り（`Required`・`Positive`）だけを、属性名の昇順で評価します。同じ入力には同じ拒否文言を返します。実データの読取りは呼出し側の責務です。

## 実行

```powershell
go test ./internal/hook ./internal/command -count=1
go vet ./internal/hook ./internal/command
gofmt -l internal/hook internal/command
go build ./...
go run ./cmd/tei validate -workspace reference/workspace
go run ./cmd/tei validate -workspace examples/xtea-lowcode
```

## 観測

### 配線と失敗の区別（`internal/hook/run_caller_test.go`）

- `TestRunnerHandsTheCallingImplementationADeclaredCaller` 成功。宣言した能力は呼出し先へ1回だけ届く。
- `TestRunnerRefusesACallTheHookDidNotDeclare` 成功。宣言外の能力は`HOOK_UNDECLARED_CALL`で拒否され、外側は`HOOK_REJECTED`として返り、呼出し先へ0回。
- `TestRunnerReportsAMissingBoundary` 成功。呼出し先の未設定は`HOOK_IMPLEMENTATION_UNAVAILABLE`で、許可と区別される。
- `TestRunnerReturnsNotSatisfiedAsItself` 成功。業務上の不成立は`Error`として包まず、そのまま返る。
- `TestRunnerKeepsNotSatisfiedUnderWarn` 成功。`on-failure warn`でも不成立は警告に変えない。
- `TestRunnerWrapsAnExecutionFailure` 成功。通常の失敗は`HOOK_REJECTED`のままで、不成立と区別される。
- `TestRunnerStillRunsAPlainImplementation` 成功。能力を呼ばない実装は`Run`のまま動く。
- `TestRunnerWarnsOnACallingFailure` 成功。`on-failure warn`でも拒否の種類（`HOOK_UNDECLARED_CALL`）は保持される。
- `TestRunnerKeepsCallerFailureDistinctFromRefusal` 成功。下位境界の失敗は不成立ではなく`HOOK_REJECTED`として返り、原因（`errors.Is`）も保持される。

### 宣言の検査（`internal/validation`）

- `TestValidateInteractionHookRejectsDuplicatePhase` 成功。同じ位相（`before-update`など）のフックを二つ宣言すると`APPLICATION_PATTERN_INTERACTION_HOOK_PHASE_DUPLICATE`で拒否される。従来は後から書いた宣言が黙って実行されない状態を許していた（`hook.Runner`は位相から一つだけ解決する）。既存の`TestValidateInteractionHook`（正常系）と各負例は`ok`のまま、`reference/workspace`と`examples/xtea-lowcode`の`tei validate`も`ok`。

### 規則Pluginの参照実装（`internal/hook/reference_test.go`）

- `TestReferenceRuleAcceptsObservedPositiveValue` 成功。観測済みで条件を満たす値は通る。
- `TestReferenceRuleRefusesAnUnobservedAttribute` 成功。宣言した読み取りが未観測なら`not-satisfied`で拒否される（沈黙を許可とみなさない）。
- `TestReferenceRuleRefusesANonPositiveValue` 成功。条件を満たさない値は`value-rejected`として返る。
- `TestReferenceRuleRefusesANonNumber` 成功。数値として読めない値も`value-rejected`で、解析の原因を保持する。
- `TestReferenceRuleIsDeterministic` 成功。同じ入力で同じ拒否文言になる。
- `TestReferenceRuleReportsTheSameOrderForSeveralRefusals` 成功。複数の不成立があっても属性名の昇順で同じものを報告する（宣言順やmapの反復順に依存しない）。
- `TestReferenceRuleIsNotAWarn` 成功。`on-failure warn`でも業務上の不成立は警告に変えない。
- `TestReferenceRuleIgnoresUndeclaredReads` 成功。規則が宣言していない値は判定しない。

### 判定と能力呼出しの連結（`internal/hook/guarded.go`、`internal/hook/guarded_test.go`）

- `TestGuardedRunsTheCapabilityWhenTheRuleHolds` 成功。規則を満たす入力では、宣言した能力が設定した入力で境界へ1回だけ届く。
- `TestGuardedDoesNotCallWhenTheRuleFails` 成功。業務上の不成立では能力を呼ばない（0回）。
- `TestGuardedCannotCallAnUndeclaredCapability` 成功。規則が満たされていても、宣言に無い能力は`HOOK_UNDECLARED_CALL`で拒否され、境界へ0回。
- `TestGuardedFailsWithoutABoundaryWhenTheRuleHolds` 成功。判定が通っても呼出し先が無ければ`HOOK_REJECTED`で、成功にしない。
- `TestGuardedNeedsACapability` 成功。能力の未設定は`HOOK_CAPABILITY_NOT_CONFIGURED`で報告される。

### 周辺

- `go test ./internal/hook ./internal/command ./internal/validation`はWindowsで`ok`、`internal/hook`と`internal/command`はWSL2でも`ok`。
- `go vet`と`gofmt -l`は無出力（WindowsとWSL2）。
- `go build ./...`は成功。
- `reference/workspace`の`tei validate`は`ok`（既存の警告3件のみ）。`examples/xtea-lowcode`も`ok`（既存の警告1件のみ）。
- `tools/repository-plan-check`は`ok`。`tools/repository-architecture-check`はWindows・WSL2のどちらでも600秒のtimeoutになった（この環境の制約）。`tools/repository-node-tests`はWSL2のNode v12が`--test`に対応しないため失敗し、Windowsでは`ok`。最終的な判定はCIの`repository-health`に委ねる。

## 読み取れること

- フックが到達できる能力は、宣言（`uses`）とRunnerが設定した境界の両方で決まる。実装の裁量ではない。
- 宣言外の呼出しは、Command境界へ届く前に拒否される。
- 業務上の不成立と実行の失敗は、呼出し側が型で区別できる。
- 参照規則は、宣言した読み取りだけを、宣言順に依存しない順序で評価する。

## 読み取れないこと

- 実規則Plugin（外部Plugin・Provider）からの供給路。`ReferenceRule`は参照実装です。
- `hook.Runner`をCLI・Studioの実行経路へ配線すること。対話の位相とCommand実行の順序・取引境界の関係は未決です。
- フックの`reads`・`produces`が実データを読み書きしたかの観測。`reads`は`Invocation`の入力として渡されます。
- 実Provider・実データベース・本番配備。Executorは参照用です。
- `hook.NotSatisfied`の`Outcome`名と`command.ReferenceFailure`の結末名（`not-found`など）の対応規則は未決です。
- 同じ能力を宣言する`command` Patternが複数ある場合の優先規則。`CommandCaller`は能力→Patternの対応を呼出し側から受けます。
- 複数の規則が同じ位相に登録した場合の合成順序。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。`AC-2`の「宣言外の能力呼出しの拒否」と`AC-5`の「拒否と実行障害の区別」は、フック境界の参照実装として配線しました。実Plugin・実Provider・CLI/Studioの実行経路への配線は未実装です。`CommandCaller`（[#3108](https://github.com/hat47x/tei-platform/pull/3108)）の観測はこの文書に含めていません。
