# Gate C First Web Target Evidence Runbook

## 目的

第一Web TargetのGate C Evidenceを、Handover Baseline `B / R`とTarget build revision `T`を混同せず、同一clean revisionから再取得できるようにする。

このRunbookのrunnerは次を取得する。

1. Source qualification
2. Semantic Conformance raw artifacts
3. Interaction / Functional raw artifacts
4. standard Projection / Custom UIのHTML artifact

**Visual Fidelity Evidenceは取得しない。**

HTMLの存在をScreenshot / Visual Fidelityへ読み替えない。Semantic / Functionalが成功しても、Visual Evidenceが未取得なら`gateCReady = false`のままとする。

契約の正本:

- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/GATE_B_TO_C_HANDOVER_BASELINE_B.md`

実装:

- `tools/run-first-web-target-gate-c-evidence.sh`
- `tools/reference-first-web-target-evidence-audit`

## 前提

Repository rootのclean full checkoutから実行する。

runnerは少なくとも次をfail closedで確認する。

- current working directoryがRepository root
- tracked / untracked / ignored local stateがない
- tracked index entryが通常の`H` stateのみ
- `curl`が利用可能
- First Web Target sourceが`gofmt`済み
- 対象Go packageのsource testsが成功
- run前後で同じ`targetBuildRevision = T`
- artifact directoryがRepository外

Go実行環境はrunner内で次へ固定する。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

## Handover Baseline

現在の第一Handover Baseline:

```text
B
baselineRepositoryRevision R =
e5231839a5dcc62cce1181a2731fcb7c70d16736
```

runner既定値もこのBaselineを使うが、Evidence採用時はsummary内の`B / R / T`を必ず確認する。

`T`はrunner開始時のclean `git rev-parse HEAD`であり、`R`へ固定しない。

## 実行

推奨はRepository外の空pathを明示する。

```bash
TEI_GATE_C_ARTIFACT_DIR=/tmp/tei-gate-c-first-web-T \
bash tools/run-first-web-target-gate-c-evidence.sh
```

既定の`B / R`を上書きする場合だけ明示する。

```bash
TEI_GATE_C_BASELINE_ID=B \
TEI_GATE_C_BASELINE_REVISION=e5231839a5dcc62cce1181a2731fcb7c70d16736 \
TEI_GATE_C_ARTIFACT_DIR=/tmp/tei-gate-c-first-web-T \
bash tools/run-first-web-target-gate-c-evidence.sh
```

既存artifact directoryは上書きしない。

## same-run raw artifacts

成功runでは少なくとも次をRepository外に保持する。

```text
run-metadata.json
source-tests.log
server.log
health.txt
manifest.json
read.json
submit-success.json
submit-authentication-denied.json
submit-authorization-denied.json
submit-validation-not-satisfied.json
submit-execution-error.json
custom-submit-get.html
custom-submit-success.html
semantic-functional-summary.json
```

### read

`read.json`はReference診断用read-only endpointから取得する。

確認対象:

- browse
- selected record
- customer reference
- header-detail
- Runtime Binding

standalone read endpoint自体をproductionのAuthentication済みread設計へ格上げしない。

### successful Command

`submit-success.json`では次を同じScenarioで確認する。

```text
Authentication
 -> read
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation
```

Command acknowledgementをObservationへ読み替えない。

### Business denial

次を別artifactにする。

- Authentication denied
- Authorization denied

どちらもHTTP transport errorやExecution Errorへ畳まない。

### Validation not-satisfied

`submit-validation-not-satisfied.json`ではCanonical Validationが`not-satisfied`となり、Authorization / Command / Observationへ進まないことを確認する。

Reference runtime fixtureではorder `1002`をこのPressureに使う。これはCanonical dataではない。

### Execution Error

`submit-execution-error.json`ではCapability Invocationを完了できない状態をBusiness Resultへ変換しない。

Reference runtime fixtureではorder `1003`のsynthetic Command adapter failureを使う。

HTTP層はTarget-localな422 response envelopeとして:

- `B / R / T`
- partial Scenario result
- `executionError`

を保持する。

これは新しいCanonical Outcomeではない。

## 独立audit

runnerはraw artifacts取得後に:

```bash
go run ./tools/reference-first-web-target-evidence-audit ...
```

を実行し、`semantic-functional-summary.json`を作る。

auditは少なくとも次を再確認する。

- raw artifact間の`B / R / T` exact一致
- pre/post runが同じclean `T`
- server process / health artifact
- 5 Application Pattern identity
- read-only Datasource boundary
- Oracle / PostgreSQL Runtime Binding
- successful Command path
- Authentication / Authorization denial
- Validation not-satisfied
- Command Execution Error
- Command Business Result / Observation分離
- Custom UI / standard interactionのshared semantic boundary
- raw artifactへのReference credential material混入なし

raw exact bytesのSHA-256をsummaryへ残す。

## Visual Fidelity

このrunnerはVisual Fidelityを成立させない。

summaryの期待値:

```text
sourceQualification.passed = true
semanticConformance.passed = true
functionalEvidence.passed = true
visualEvidence.status = pending
visualEvidence.required = true
gateCReady = false
```

Screenshotまたは同等の再確認可能なVisual artifactを**同じTarget build revision T**に対して別途取得する。

HTML artifactやDOMの存在だけでVisual Evidenceを成立させない。

## Credential boundary

runtime requestにはReference用synthetic credentialを使うが、通常Evidenceへ保存しない。

auditは少なくとも次の混入をfail closedで拒否する。

- Reference synthetic credential文字列
- `credential-ref://legacy/`

request bodyそのものをartifactへ保存しない。

## 停止条件

次のいずれかではGate C Evidenceへ採用しない。

- source test失敗
- unformatted source
- dirty / non-standard index checkout
- run中のHEAD drift
- server health不成立
- B / R / T不一致
- expected Pattern / Runtime Binding不一致
- Business denialがExecution Errorへ変換された
- Validation不成立後にCommandへ進んだ
- Execution ErrorがBusiness Resultへ変換された
- credential materialがraw artifactへ残った
- Visual未取得なのにGate C Readyと宣言した

## Gate C判定

このrunnerの成功はSemantic / Functional Evidence取得であり、Gate C Readyそのものではない。

Visual Fidelityを別Evidenceとして取得し、`plan/GATE_C_EVIDENCE_CONTRACT.md`の全条件を独立監査した後だけGate Cを判定する。
