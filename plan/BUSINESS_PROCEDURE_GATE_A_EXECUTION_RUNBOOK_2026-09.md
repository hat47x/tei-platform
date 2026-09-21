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

PR #482はmainへmerge済みなので、旧Lane C長期branch名をexecution条件にしない。

実行前に**Pre-ADR判定対象にするexact revision**を固定する。

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

- Evidenceにはexact `R`を記録する
- worktreeに意図しない変更がない
- branch名はmetadataでありPASS条件ではない
- main上で実行する場合は`R`がその時点のmain headであることを記録する
- fix / verification branchで実行する場合はmainとの差分とmerge baseを記録する

Contract ADR readinessをmainのaccepted stateについて判定する場合、最終的には**その判定対象treeを含むrevision**でGate Aを実行する。旧PR headやmerge前branchのPASSを、後続main revisionのPASSへ黙って読み替えない。

`main` behind 0は望ましいが、execution PASSの定義そのものではない。behindがある場合はそのrevision条件を明示する。

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
  != Contract ADR Ready

checker baseline PASS
  != PAS-07 accepted
```

Gate Aはmechanical execution Evidenceである。

### Partial CIをGate Aへ合成しない

異なるworkflow / revision / package subsetで得たPASSを寄せ集めてGate A greenにしない。

```text
Gate B source qualification PASS
+ Issue #50 full compile PASS
+ focused Lane C source inspection
  != Gate A PASS
```

Gate A PASSは同一revision `R`について、本runbookの必須command一式をactual executionしたEvidenceから判定する。

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
- source-root membership freshness
- logical Procedure source uniqueness
- standalone Handoff source uniqueness
- Conformance-base freshness
- multi-source bundle
- coordinated rename
- migration preview / authority cutover
- candidate-only disk reload
- fresh-process candidate-only reload
- actor / trigger Handoff independence
- second-corpus non-inference pressure

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

## 11. Step A7 — Focused critical Pressure preflight / triage

Full package PASSがEvidenceの正本であり、個別test実行はfailure triage / critical Pressure再確認のためのdebug aidである。

ただしGoの`-run`はregexが0件matchでもcommand自体が成功し得るため、**個別testを実行したつもりで0件matchをPASSへ読み替えない**。

Triage前に、実行revision `R`でcritical test名が存在することを確認する。

critical manifestは`tools/business-procedure-gate-a-critical-tests.txt`（1行1test名）を唯一の正本とする。このRunbookとrunnerへ同じ一覧を複製しない。一覧に載せるtestを追加・削除した場合は、そのfileだけを更新する。

Procedure-scoped Step address、stale endpoint fail-closed、actor context independence、trigger / Handoff condition authority separationはPre-ADR Gate Cのsemantic guardsである。PR #543でexact-name preflightへ移行した際にこれらをsubsetから落とさず、critical triageでも明示的に追えるようにする。Source-discovery層のendpoint-only identity guardとnormalized/canonical層のsame-endpoint/different-condition distinctionも両方追い、片方だけgreenに見えるtriageを避ける。

またPre-ADRが要求するauthority transitionは:

```text
legacy-read migration preview
  -> legacy + candidate simultaneous authority reject
  -> accepted candidate-only authority
  -> candidate-only disk / fresh-process reload
```

である。Standalone Handoffについてもstrict decode / round-trip / normalized thin relationをこのtransitionと同じcritical triageで追う。これらはproduction cutover完成の主張ではなく、current test-only feasibility Pressureの実行確認である。

この一覧は永続的なGate schemaではない。実行時は必ずrevision `R`のsourceを正とする。

`tools/run-business-procedure-gate-a.sh`は、A7で次をfail-closedで行う。

1. package内のtest名を`go test -list 'Test.*'`で列挙し、manifestの**exact nameが全件存在する**ことを確認する（存在しなければfailure）。
2. manifestのtestだけを`-run '^(...)$' -v`で実行する。
3. 実行後の`--- PASS:`行の数がmanifestの件数と一致することを確認する（regexが一部にしかmatchしなかった場合をPASSに読み替えない）。

このsubset PASSはStep A4のfocused package PASSを代替しない。Triageでfixが入ってrevisionが変わった場合、Step A1〜A6を含むGate A全体を新revisionで再実行する。

### Step A8 — second realistic Reference baseline checker

`reference/business-system-design/second-corpus/`は、`order_entry`から独立した二つ目のBusiness Procedure Reference（[ISSUE-000050](issue/000050-second-realistic-handoff-corpus.md)）である。そのpressure testはA4 / A7が既に走らせ、critical manifestにも載る。A8は、その二つのfixtureを実際のbaseline checker CLIへ通す。

```text
go run ./tools/reference-business-procedure-check \
  -workspace reference/workspace \
  -fixture reference/business-system-design/second-corpus/<fixture>.json
```

対象は`make-to-order-production-run.json`と`finished-goods-stocking.json`である。二つ目のReferenceはApplicationを持たないため、blocking diagnosticとApplication Pattern rowのどちらも出ないことを期待する。access probeは適用しない（Datasourceが無い）。

A8のPASSは、RRB-P3のうち「二つ目のReferenceが実際のrunnerでbaseline checkerを通る」ことだけを示す。独立reviewを経たことも、Handoff一般の妥当性も、Contract ADRのreadinessも示さない。

## 12. Evidence record format

一回のactual runごとに、`tools/run-business-procedure-gate-a.sh`が次を`gate-a-summary.json`とstepごとのlogへ書き出す。最低限:

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

Evidenceはrepositoryへcommitせず、Repository外のartifact directoryへ書き出す（runnerは`TEI_GATE_A_ARTIFACT_DIR`を要求し、repository内を拒否する）。

採用したrunのartifactは、Gate B / Gate Cと同じ保管規則（`ISSUE-000057`、`GATE_B_EVIDENCE_RUNBOOK.md`「保管」）に従う。容量が小さいため、`MANIFEST.sha256`を同梱した`tar.gz`として、対象revision `R`を指すtagを付けたGitHub Release（prerelease）へ格納する。判定文書には、revision・archiveのSHA-256・locatorだけを記録する。

## 14. Failure handling

### Lane C code failure

新規Pressure test / checker changeが原因なら:

```text
failure
  -> fix on focused branch
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

GitHub Actionsは使えないため、Lane C Gate Aは手元のclean full checkoutで実行する。検証の共通の入口はlocalの`tools/repository-health-check.sh`である。

過去にあったGate B専用のsource qualification workflowや、Issue #50用の一回限りのworkflowは、Lane C Gate A Evidenceを代替しないうえ、既に削除している。それらの実行結果をLane C Gate Aへ流用しない。

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

Focused Lane C testsには、そのrevisionに存在するsource-uniqueness / source-root membership / fresh-process / second-corpus non-inference Pressureも含まれる。

この時初めてPre-ADR auditの:

```text
Gate A — actual execution
```

をgreen候補へ更新できる。

## 17. Gate A後すぐ行うこと

Gate A green後の順序:

1. Evidenceをrevision-boundで記録。
2. Pre-ADR audit Gate A status更新。
3. source-level / test-only Pressureのうち実行されたものをactual Evidenceへ再分類。
4. independent Business requirement由来のsecond realistic Handoff authoringへ進む。
5. production loader / schema prototypeの着手可否を再判定。

Gate A greenを理由に即ADR採用へしない。

## 18. Current state

```text
Gate A actual Evidence
  = PASS（R = e2856e6e568b6be966c6b8689ebea479ec184a4a、2026-09-20）
  = runner: tools/run-business-procedure-gate-a.sh
  = record: Pre-ADR Audit section 13 /
    GitHub Release business-procedure-gate-a-evidence-R-e2856e6

Gate A actual Evidence（A8とcritical 13件を加えた後の再実行）
  = PASS（R = 82e430109f8f02944519738045f3d6c0bef4f295、2026-09-21）
  = record: Pre-ADR Audit section 17 /
    GitHub Release business-procedure-gate-a-evidence-R-82e4301

Pre-ADR
  = Not Ready
  = Blocker B: Evidence取得済み（2026-09-21、作成者self-review）。Auditによる再判定が残る
```

Gate A Evidenceは、そのrevision `R`のmechanical Evidenceである。判定対象のtreeが変わった場合は、`tools/run-business-procedure-gate-a.sh`で新しいrevisionについて取り直す。
