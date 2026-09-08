# Business Procedure Gate A Execution Runbook — 2026-09

Status: **Lane C / execution runbook / no new Gate / no CI architecture adoption**

## 1. 目的

Lane Cの最大blockerは、Repositoryにtest-only Pressureが存在する一方でactual Repository runnerによるgreen Evidenceをまだ取得していないことである。

本書はrunnerが利用可能になった時に:

- 何を実行するか
- どの順序で実行するか
- 何をEvidenceとして残すか
- failureをどう分類するか

を固定する。

新しいGitHub Actions workflow、Gate、production contractを追加する文書ではない。

## 2. 対象revision

実行前に必ずcurrent PR headを固定する。

Conceptual:

```text
R = git rev-parse HEAD
```

確認:

```text
git status --short
git branch --show-current
git rev-parse HEAD
git merge-base HEAD main
git rev-list --left-right --count main...HEAD
```

期待:

- branch = `work/business-procedure-cross-view-reference-current`
- worktreeに意図しない変更がない
- Evidenceにはexact `R`を記録

`main` behind 0は望ましいが、execution PASSの定義そのものではない。behindがある場合はそのrevision条件を明示し、黙ってlatest mainのEvidenceへ読み替えない。

## 3. Runbook原則

### Read-only first

最初のverificationでsourceを書き換えない。

禁止:

```text
gofmt -w ...
git commit
git push
```

最初にformat checkをread-onlyで行う。

### Fail fastしすぎない

一つのcommand failureで残りを全部未実行にせず、可能なら独立commandを続けてfailure surfaceを収集する。

ただしcompile failureでdependent checkerが意味を持たない場合は未実行理由を残す。

### PASSを拡張解釈しない

```text
go test PASS
  != Production ADR Ready

checker baseline PASS
  != PAS-07 accepted
```

Gate Aはmechanical execution Evidenceである。

## 4. Step A0 — Environment capture

記録:

```bash
go version
git --version
go env GOOS GOARCH GOVERSION CGO_ENABLED
```

必要なら:

```bash
go env GOMOD GOWORK
```

secret / private environment variable一覧をEvidenceへ保存しない。

## 5. Step A1 — Formatting check

Current Go source scopeは既存Issue #50 read-only workflowと同じく:

```bash
test -z "$(gofmt -l cmd internal examples tools)"
```

を第一候補とする。

追加でLane C changed Go filesだけを確認したい場合も、全scope PASSの代替にはしない。

Failure時:

```text
classification = formatting
```

として列挙fileを記録する。

自動修正はEvidence採取後の別commitにする。

## 6. Step A2 — Static analysis

```bash
go vet ./...
```

Failure時に:

- package
- diagnostic
- relevant file / line

を記録する。

Lane C変更と無関係なbaseline failureの場合も隠さず、base revisionとの比較対象にする。

## 7. Step A3 — Full test suite

```bash
go test ./...
```

可能なら初回はcache前提を避けるため:

```bash
go test -count=1 ./...
```

もEvidence run候補とする。

ただし既存Repositoryの標準commandを無断変更しない。`go test ./...`を必須基線とする。

Failure分類:

```text
compile
unit / integration test
fixture / golden mismatch
external dependency / environment
panic / timeout
```

を区別する。

## 8. Step A4 — Lane C focused package

全suiteに加えてfocused Evidenceとして:

```bash
go test -count=1 ./tools/reference-business-procedure-check
```

を実行する。

このpackageには少なくとも:

- D4 / O1 minimum pressure
- Handoff / Step address pressure
- Data / CRUD projection pressure
- source freshness
- Conformance-base freshness
- multi-source bundle
- coordinated rename
- migration preview / authority cutover
- actor / trigger Handoff independence

が含まれる。

Focused PASSはfull suite PASSの代替ではない。

## 9. Step A5 — Reference checker baseline

Primary baseline:

```bash
go run ./tools/reference-business-procedure-check \
  -workspace reference/workspace \
  -fixture reference/business-system-design/order-entry-business-procedure.json
```

Expected:

- process exit 0
- blocking unexpected diagnosticなし
- reportがprimary Procedure identityへtraceできる

OutputをEvidenceとして保存する場合、Derived reportでありauthoritative sourceへ逆書きしない。

## 10. Step A6 — Intentional defect checker

Current intentional conflict:

```text
order-entry-customer-edit-conflict.json
```

Existing `main_test.go`の`TestCustomerEditConflictIsDetected`が要求するexpected diagnosticは:

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

である。

Run:

```bash
go run ./tools/reference-business-procedure-check \
  -workspace reference/workspace \
  -fixture reference/business-system-design/order-entry-business-procedure.json \
  -probe reference/business-system-design/order-entry-customer-edit-conflict.json \
  -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

重要:

```text
intentional defect detected
  = PASS

defect accepted silently
  = FAIL
```

である。

Expected codeはcurrent test contractから取得した値であり、将来diagnostic contractを意図的にrenameした場合はtestとrunbookを同じaccepted changeとして更新する。

## 11. Step A7 — Focused new Pressure names

Full package PASSで十分だが、failure triage時は次を個別に実行できる。

```bash
go test -count=1 ./tools/reference-business-procedure-check \
  -run 'TestBusinessAcceptancePreviewBase|TestBusinessAuthoringAuthorityCutover|TestBusinessHandoffConditionAndTargetProcedureTrigger|TestBusinessHandoffRemainsAddressed'
```

これはdebug aidであり、Evidence suiteをこのsubsetだけへ縮めない。

## 12. Evidence record format

一回のactual runごとに最低限:

```text
repositoryRevision
branch
baseRevision / mergeBase
runnerKind
OS / arch
Go version
startedAt / finishedAt

formatCheck
  command
  exit

goVet
  command
  exit

goTestAll
  command
  exit

goTestLaneC
  command
  exit

checkerBaseline
  command
  exit
  output reference

checkerIntentionalDefect
  command
  expected diagnostic
  exit
  output reference

unresolved
```

を残す。

## 13. Evidence storage

First actual runでは、Repositoryへ巨大raw logを無条件commitしない。

候補:

1. PR comment + exact revision + concise output
2. small `plan/evidence/...md` record
3. CI run URL / artifact reference

を比較する。

Stable Evidenceとして長期保持する必要が確認できた場合だけRepository fileへ昇格する。

## 14. Failure handling

### Lane C code failure

新規Pressure test / checker changeが原因なら:

```text
failure
  -> fix on same branch
  -> new revision R2
  -> rerun all Gate A commands
```

旧revisionの部分PASSをR2へcarry-forwardしない。

### Baseline failure

mainでも同じfailureが再現する可能性がある場合:

- Lane C failureと断定しない
- base revisionで比較する
- blocker / unrelated baseline issueを分ける

### Runner infrastructure failure

checkout前終了、runner unavailable、network setup failure等は:

```text
execution Evidence unresolved
```

でありcode FAILへ丸めない。

## 15. GitHub Actions利用条件

現在確認できるworkflowはIssue #50専用branch / path trigger等が中心であり、Lane C arbitrary branchのgeneric read-only verification workflowは見つかっていない。

Issue #50 workflowの一部は`gofmt -w` + auto-commit / pushも行うため、そのままLane C Evidenceへ流用しない。

将来generic workflowが独立理由で整備された場合は利用可能だが、Gate AをgreenにするためだけにLane CからCI architectureを新設しない。

## 16. Gate A PASS条件

同一revision `R`で:

```text
format check = PASS
go vet ./... = PASS
go test ./... = PASS
focused Lane C tests = PASS
baseline checker = PASS
intentional defect checker = expected diagnostic PASS
```

をactual runnerから得る。

この時初めてPre-ADR auditの:

```text
Gate A — actual execution
```

をgreen候補へ更新できる。

## 17. Gate A後すぐ行うこと

Gate A green後の順序:

1. Evidenceをrevision-boundで記録。
2. Pre-ADR audit Gate A status更新。
3. A91–A99 test-only Pressureをactual Evidenceへ再分類。
4. second realistic Handoff authoringへ進む。
5. production loader / schema prototypeの着手可否を再判定。

Gate A greenを理由に即ADR採用 / PR Readyへしない。

## 18. Current state

```text
HAT47X
  = offline

GitHub generic manual read-only verification workflow
  = not found

Gate A actual Evidence
  = unresolved

Runbook
  = defined

Pre-ADR
  = Not Ready
```
