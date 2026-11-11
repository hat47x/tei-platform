# Command境界の実行前検査と適用不明の記録（2026-11）

- 状態: **参照実装の変更案（A2）**。意味の受理・権限の正本・本番配備は扱わない。
- 関連: [#2996](https://github.com/hat47x/tei-platform/issues/2996)、[#2997](https://github.com/hat47x/tei-platform/issues/2997)、[#2999](https://github.com/hat47x/tei-platform/issues/2999)
- 対象コード: `internal/command/invoke.go`、`internal/command/safety_test.go`、`cmd/tei/cli_view.go`、`internal/studiohost/command.go`

## 1. なぜ必要か

Command境界は「宣言」と「実行」の間に立つ。これまでは、呼出し側が`Authorizer`や`PeriodState`を設定し忘れると、検査を省いて実行する既定だった。また、Executorが宣言外の効果を返した場合、境界はそれを「拒否」としてLifecycleへ渡していた。

どちらも、外部のHarnessやHostから見ると「検査済み」「適用されていない」と読み違えられる。この文書は、**未設定を成功にしない**、**適用有無が分からない失敗を拒否と混ぜない**という二つの判断を、コードと試験で固定する。

## 2. 判断したこと

| 対象 | 変更前 | 変更後 | 理由 |
| --- | --- | --- | --- |
| `Authorizer == nil` | 認可せず実行 | `COMMAND_AUTHORIZER_UNAVAILABLE`で拒否 | 未設定を「認可不要」と読む経路を残さない |
| 参照実行の認可 | 未指定 | `ReferenceAuthorizer{}`を明示 | 参照実行が**認可判断を省略した**ことを型で示す |
| 認可と期間の順序 | 期間→認可 | 認可→期間 | 認可されない呼出しに期間の開閉を返さない |
| `Periods == nil`で`period`宣言あり | 検査しない | `COMMAND_PERIOD_STATE_UNAVAILABLE`で拒否 | 宣言を検査できないことを「拒否なし」と扱わない |
| 宣言外の効果・結末 | rejectedとして記録 | `EffectUnknown`を立ててunknownとして記録 | Executor実行後に検出するため適用済みの可能性がある |
| 適用操作の不一致 | rejectedとして記録 | 同上 | `UnitOfWork`があってもExecutorの取引参加は契約外 |

`ReferenceAuthorizer`は「許可する」と同時に「本物の主体・権限・規則を評価していない」ことを名前で示す。本番用の認可契約ではない。

## 3. 変更後の検査順

```text
入力検証 (BuildCommandPlan)
  → Executor の存在
  → 認可 (Authorizer は必須)
  → 期間 (period が宣言されたら PeriodState は必須)
  → 取引の開始
  → Executor.Execute
  → 宣言との照合（効果・適用操作・結末）
  → Recorder
  → Commit
```

失敗した場合のLifecycleへの写像は次のとおり。

| 失敗 | Lifecycleの段階 | 再送 |
| --- | --- | --- |
| 入力不正・宣言なし・認可拒否・期間拒否 | observed / rejected | 同じowner/generationでは再送しない |
| `EffectUnknown`付きの失敗（commit失敗・rollback失敗・宣言外の効果） | unknown | 同じowner/generationでは再送しない。別owner・高いgenerationでtakeover |

## 4. 具体例

`tei view -command -execute`は参照実行であり、永続化しない。それでも`Authorizer`を省略しない。

```go
invoker := command.Invoker{
    Authorizer: command.ReferenceAuthorizer{},
    World:      loaded.World,
    Executor:   command.ReferenceExecutor{World: loaded.World},
    UnitOfWork: &command.MemoryUnitOfWork{},
}
```

これで、将来この初期化をコピーして別のHostを作った場合も、`Authorizer`を消すと`COMMAND_AUTHORIZER_UNAVAILABLE`で止まる。黙って無認可実行にはならない。

## 5. 根拠

- `go test ./internal/command -count=1`: PASS（`safety_test.go`を含む）
- `go test ./internal/command/... ./internal/execution/... ./internal/security/legacyauthorization/... ./internal/security/effectcommit/... ./internal/studiohost/... -count=1`: PASS
- `go test ./cmd/tei -count=1`: PASS
- `go build ./...`、`go vet ./internal/command ./cmd/tei ./internal/studiohost ./internal/security/legacyauthorization ./internal/security/effectcommit/...`: PASS
- 実行環境: Windows 11、go1.27.0。実PostgreSQLを使う試験はDSN未設定のためskip。

## 6. `committed` と `persisted` の分離（#2998 追記）

### 見つかった誤読

`tei view -command -execute` とStudioの参照実行は、どちらも `committed: true` を固定で返していた。参照実行の `UnitOfWork` は `MemoryUnitOfWork` で、Commitの呼出し回数を数えるだけで何も保存しない。しかし応答の欄名と値だけを見ると「保存された」と読める。

### 決めたこと

| 欄 | 意味 | 参照実行の値 |
| --- | --- | --- |
| `committed` | 構成された `UnitOfWork` の `Commit` が成功したこと。取引境界の呼出し結果であり、永続化の証拠ではない | `true` |
| `persisted` | 実行先へデータが保存されたこと | `false` |
| `durability` | 保存境界の種類。`none` は保存しない参照実行 | `"none"` |

`committed` は名称も意味も変えない。既存の読み手（Studio画面、`product_slice_test.go`、外部Harness）が使っているため、欄を削除・改名する案は採らない。`persisted` と `durability` は**欄の追加**であり、`tei.cli.view/v0` の互換な変更の範囲に収まる（[CLI JSON出力契約](CLI_JSON_OUTPUT_CONTRACT_2026-10.md) §5）。

CLIのテキスト出力は `committed: true (reference unit of work)` と `persisted: false (durability: none)` を別行で出す。Studioの画面も同じ3欄を別行で表示する。

### この分離が防ぐこと

- 参照実行の成功を、実DBへの保存成功と混同しない。
- 実Providerを接続した後も、`committed`（取引のCommit）と `persisted`（独立した読取りで確認した保存）を別の欄として扱える。将来 `persisted` を「独立読取りで確認済み」の意味に強める場合は、`durability` の値と合わせて別の判断として記録する。

### まだ決めていないこと

- 実Provider接続時に `committed` を「取引のCommit成功」と「独立した読取りで確認できたこと」のどちらへ対応させるか。本追記では参照実行の意味だけを固定し、実Providerの意味は未決のまま。
- `schemaVersion` を `v1` へ上げる必要があるか。欄の追加だけなので `v0` のままと判断したが、外部読み手の互換性確認は別途必要。

## 7. 効果を宣言しないCommandの `appliedUnobserved`（#3068 追補）

### 見つかった誤読

`OperationRecord.AppliedUnobserved` は「Executorが適用操作を列挙しなかった」ことを示す。しかし `Applied` が空になる理由は二つある。

1. Executorが列挙できない（**未観測**）。
2. 能力が効果を宣言していないので、比較する対象が無い（**比較対象なし**）。

どちらも `len(result.Applied) == 0` になるため、単純な実装では2を「未観測」と表示していた。効果を宣言しないCommandでは `difference-check: not observed` が常に出る。

### 決めたこと

`AppliedUnobserved` は **能力が効果を宣言している場合だけ** true にする。

- `plan.Effect != nil && len(result.Applied) == 0` → `appliedUnobserved:true`
- `plan.Effect == nil` → `appliedUnobserved:false`

CLIとStudioは、記録が無い場合の既定値を「効果が宣言されていればtrue、されていなければfalse」にした。効果なしのCommandでは `difference-check: not observed` を出さない。

### この分離が防ぐこと

- 「比較する対象が無い」を「Executorが答えられなかった」と読み違えない。
- 効果なしのCommandに、存在しない差の確認を要求しない。

### まだ決めていないこと

- 効果を宣言したが `Applied` が空のままのExecutorを、構成時に拒否するか（#2996の未決）。
- `AppliedUnobserved` を `Differences` の有無と独立に扱うか、`differenceCheck` のような三状態（`checked`・`not-observed`・`not-applicable`）へ改めるか。

## 8. 宣言効果を報告しないExecutorの記録（#2996 追補）

### 見つかった差

能力が効果を宣言しているのに、Executorが `Effect` を返さない場合がある。`effectcommit` アダプタはCommit receiptだけを返し、効果の種類・対象を返さない。この場合、境界は次のどちらかを知らない。

- Executorが宣言効果を適用したが、報告しなかった。
- Executorが何も適用しなかった。

### 決めたこと（記録のみ）

`OperationRecord.ReportedEffectMissing` を追加し、`plan.Effect != nil && result.Effect == nil` のとき true にした。CLIの `referenceCommandRun` とStudioの `CommandRunValue` にも同じ欄を追加し、CLIテキストとStudio画面に `reported-effect: missing` を表示する。

**拒否はしていない。** 宣言効果の報告漏れを `MISSING_COMMAND_EFFECT`（EffectUnknown）として拒否すると、`effectcommit` アダプタのCommit成功がすべてunknownになり、実Providerの正常経路が壊れる。実Providerの意味（Commit成功と効果報告の一致）は、Providerの契約として別に決める必要がある。

### この記録が防ぐこと

- 「効果が報告されていない」を「宣言効果が適用された」と読み違えない。
- `AppliedUnobserved`（操作の列挙漏れ）と `ReportedEffectMissing`（効果の報告漏れ）を別の欄として扱える。

### `effectcommit` アダプタの報告（追記）

`commandadapter.Executor` に `Effect *semantic.DataEffect` を追加した。呼出し側がCommand計画から宣言効果を渡すと、`StatusCommitted`／`StatusReplayed` の結果にその効果を載せる。`nil` なら従来どおりreceipt-onlyで、`ReportedEffectMissing` が記録される。

- `Replayed` は同じOperationID・RequestHashの記録が既にあることの証明であり、資源の現在値の独立読取りではない。したがって `persisted` の意味を強めるものではない。
- 実PostgreSQLの結合試験は `TEI_EFFECT_POSTGRES_DSN` 未設定のため、この環境ではskip。

### まだ決めていないこと

- 呼出し側が宣言効果を渡さない構成を許すか、必須にするか。
- 宣言効果の報告漏れを、どのExecutorで拒否するか。参照Executorは宣言効果をそのまま返すため対象外。
- `effectcommit` のCommit成功を、独立した読取りで宣言効果と照合するか（#2998の `persisted` の意味）。

## 9. 永続化の観測（`persistence`）と `persisted`（#2998 追補）

### 三つの問いを分ける

`committed` と `persisted` の二欄だけでは、次の三つが混ざる。

1. その実行に永続storeがあるか（参照実行は無い）。
2. durableなCommitが起きたか（`UnitOfWork.Commit` の結果）。
3. Commit後に、宣言効果を独立した読取りで見たか。

`command.Result.Persistence` と `PersistenceState` を追加し、次の三状態にした。

| 値 | 意味 |
| --- | --- |
| `not-applicable` | 宣言効果が無い、または `Invoker.DurableStore=false`（参照実行）。観測対象が無い |
| `not-observed` | durable storeはあるが、独立した読取りを構成していない（`PersistenceObserver` がnil）、または読取りが失敗した |
| `receipt-observed` | 独立読取りが、Commit済みOperationのreceiptを確認した。receiptはOperationIDとRequestHashの記録であり、資源の現在値ではない |
| `resource-observed` | 独立読取りが、資源の現在値とCommitしたpayloadの一致を確認した。これだけが `persisted:true` |

`PersistenceObserver` は `Request` と**宣言効果**だけを受け、Executorの `Result` を受けない。Executorの報告をそのまま観測として返せないようにするためである。読取り失敗はCommandの失敗ではなく `not-observed` にする（Commitは既に起きている）。

CLIとStudioは `persisted`（`resource-observed` のときだけtrue）と `persistence` を返す。参照実行は `persisted:false`・`persistence:"not-applicable"`、実Providerでobserver未構成なら `persistence:"not-observed"` になる。

### `Invoker.DurableStore` の位置付け

`Invoker.DurableStore` はcompositionの主張であり、Invokerは検証しない。`UnitOfWork` が `MemoryUnitOfWork` かどうかを型で判定しないのは、参照用の取引と実取引を型で偽装できるためである。参照実行はfalseにして `not-applicable` を明示する。

### `effectcommit` アダプタの観測（追記）

`commandadapter.Executor` に `Lookup bool` を追加した。trueのとき `Observe` は同じ `Request`・`Grant`・`OperationID`・payloadで `Runner.Lookup` を呼び、`StatusReplayed` なら観測成功とする。

- これは**receiptの観測**であり、資源の現在値の観測ではない。Lookupは同じDB境界で記録を照会するだけで、`objects` の現在値を読まない。
- `Lookup` の結果が `not-recorded`・`denied`・`conflict` なら `not-observed`。エラーも `not-observed`。
- 実PostgreSQLの結合試験 `TestPostgresCommandObservesACommittedReceiptByLookup` を追加した。Commit→Lookupで `observed` になり、receipt行を削除すると同じ設定の `Observe` が失敗する（資源の現在値は変わらない）ことを確かめる。`TEI_EFFECT_POSTGRES_DSN` 未設定の環境ではskipする。

### まだ決めていないこと

- 資源の現在値を読む独立観測を実Providerに足すか（現在の `Lookup` はreceiptのみ）。
- `PersistenceObserver` を実Providerの経路へ配線するか（本追記はアダプタ単体の実装）。
- `not-observed` を警告や停止条件にするか。現状は応答の欄としてのみ返す。
- `Replayed` の経路で観測を行うか。Replayは現在値の読取りではなく記録の再確認なので、別の観測が必要。
- `DurableStore` をcompositionから自動で決める方法。現状は呼出し側の明示的な宣言。

## 10. どの認可判断で実行したかを応答に載せる（#2997 追補）

### 見つかった差

#3042 で `Authorizer` 未設定を拒否し、参照実行は `ReferenceAuthorizer` を明示するようにした。しかし応答からは、その実行が**どの認可判断**で通ったのか読めなかった。`ReferenceAuthorizer`（全許可）と `AllowListAuthorizer`（能力名だけ許可）と本物の規則は、どれも「認可された」と同じ形で現れる。

### 決めたこと

`command.Result.AuthorizationMode` を追加し、Invokerが実行後に設定する。値は次の三つ。

| 値 | 意味 |
| --- | --- |
| `reference-allow-all` | `ReferenceAuthorizer`。主体・規則・資源を評価していない |
| `reference-allow-list` | `AllowListAuthorizer`。名前の一致だけを見た参照判断 |
| `external` | それ以外の `Authorizer`。どの規則・付与を参照したかは境界からは分からない |

CLIの `referenceCommandRun` とStudioの `CommandRunValue` に `authorizationMode` を追加し、テキストと画面にも表示する。欄の追加であり、`tei.cli.view/v0` の互換な変更の範囲に収まる。

### この表示が防ぐこと

- 参照実行の全許可を、本物の権限判断と読み違えない。
- `AllowListAuthorizer` を、主体・資源・規則を含む認可と読み違えない。

### まだ決めていないこと

- `external` の内訳（どの規則・付与を参照したか）を応答に載せるか。載せる場合は認可の正本・開示境界の設計が要る。
- `authorizationMode` を必須欄にするか、省略可能のままにするか。
- StudioでAllowListを指定しない実行が全許可になることを、画面で警告として出すか。

## 11. この文書が主張しないこと

- 本物の認可・主体・権限・メニュー制御が実装されたこと。
- `ReferenceAuthorizer`が本番で使えること。
- 宣言外の効果が実際にDBへ適用されたか、巻き戻されたかを区別できること。
- 実Provider・実DBで同じ結果を観測したこと。
- #2996・#2997・#2998が完了したこと。Issueの完了条件のうち、実DB確認と外部読み手の互換性確認は別途残る。

## 8. 実Provider経路での `committed` の候補（未採用）

参照実行の意味は §6 で固定した。実Providerを接続したときの `committed` をどう扱うかは、まだ決めていない。候補を整理する。

| 案 | `committed` の意味 | `persisted` の意味 | 利点 | 欠点 |
| --- | --- | --- | --- | --- |
| P1: 取引のCommit成功 | ProviderのCommitが成功した | 独立した読取りで確認できた | 現在の欄名と一致 | 「Commitした」と「読めた」を混同しやすい。P1だけでは保存の証拠にならない |
| P2: 独立読取りでの確認 | 独立した読取りで対象状態を確認できた | `committed` と同じ | 観測に基づく | 既存の読み手の意味が変わる。`v1` が必要 |
| P3: 取引境界の結末 | ProviderのCommitが成功した | 独立した読取りで確認できた。未実施は`null` | `null`で「まだ確認していない」を表せる | JSONの`bool`から`*bool`への型変更は互換でない |

この文書はどれも採用しない。実Providerの試験（`TEI_EFFECT_POSTGRES_DSN`）でCommit成功と独立読取りを同じ実行で観測できるようになった時点で、候補ごとの結果を記録してから決める。

`committed` をP2へ変える場合は `tei.cli.view` の版を上げる。P1のまま `persisted` を独立読取りの意味に強めるなら、`persisted` が「確認済み」であることを示す別欄（例: `persistenceObserved`）が要る。

## 11. 資源の現在値の観測（追記）

`DetailPersistenceObserver` を追加し、観測がreceiptと資源のどちらを確かめたかを名指しできるようにした。

- `effectcommit.Runner.ReadResource` は、Providerが `ResourceReader` を実装している場合だけ現在値を読む。未実装なら `ErrResourceObservationUnavailable` を返す（「保存されていない」ではない）。
- `commandadapter.Executor` に `ResourceRead bool` を追加。trueならCommit後に別pool接続で `objects` の現在値を読み、Commitしたpayloadと一致した場合だけ `resource-observed`。
- 実PostgreSQLの結合試験 `TestPostgresCommandObservesTheStoredResource` を追加。Commit後に `objects` を書き換えると、同じ設定の観測が失敗することを確かめる。
- receipt Lookupだけでは `receipt-observed` であり、`persisted:true` にはならない。

## 12. Executorの取引参加を契約に含めない（#2996 追補）

### 見つかった差

`Executor` の契約には「`UnitOfWork` へ参加する」と書かれておらず、`Invoker` は `UnitOfWork` がある場合に `Begin`・`Commit`・`Rollback` を自ら呼ぶ。そのため、Executorが成功した後に宣言外の効果・結末・適用操作を返した場合、`Rollback` が呼ばれても Executor の書込みが同じ取引に参加していた保証は無い。

### 決めたこと

- 宣言外の効果・結末・適用操作を検出した場合、`Invoker` は `EffectUnknown` の付いた `ExecutionError` を返す。`UnitOfWork` の有無で分けない。巻き戻し済みと断定できる根拠が無いため。
- `LifecycleInvoker` はこのエラーを `unknown` として記録し、`rejected` としては記録しない（#3042）。`unknown` の Command は同じowner/generationの再送を拒否する。
- `Executor` の型コメントに、取引への参加は契約に含まないこと、適用の有無を確定させるには Executor 側の参加か実行後の独立観測が要ることを書いた（`internal/command/invoke.go`）。
- 試験 `internal/command/safety_test.go` が、`UnitOfWork` あり・なし・`LifecycleInvoker` 経由の3通りで `EffectUnknown` と `unknown` 記録を固定している。

### まだ決めていないこと

- 宣言した効果を Executor が返さない場合（`Effect` が nil）の扱い。現在は拒否せず `ReportedEffectMissing` として記録する。拒否すると `effectcommit` アダプタのCommit成功がすべてunknownになり、実Providerの正常経路が壊れるため。
- Executor に取引参加を契約として要求し、参加しない Executor を構成時に拒否するか（Issueの選択肢c）。
- 実Providerで、宣言外の効果が実際に適用されたかを独立観測で確かめる経路。`ResourceRead` は宣言効果の現在値を見るもので、宣言外の書込みの検出ではない。

## 13. 確定結果と再送の実装仕様（2026-11-11）

[#1174/#2564の実装仕様](CRITICAL_HEADLESS_ISSUE_IMPLEMENTATION_SPEC_2026-11.md) §2〜3に、既存submit分類をCommandから保持する戻り値、OperationIDと要求の保存、Lookupを先に行う回復経路、ABAの版CAS対照を具体化した。これは次の実装の設計候補であり、この文書に記録した実装済みの安全条件を変更したものではない。実行済み試験と新規受入Caseを分けて扱う。
