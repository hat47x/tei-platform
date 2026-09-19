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
projection-index.html
manifest.json
read.json
submit-success.json
submit-authentication-denied.json
submit-authorization-denied.json
submit-validation-not-satisfied.json
submit-execution-error.json
custom-submit-get.html
custom-submit-success.html
custom-submit-authorization-denied.html
custom-submit-validation-not-satisfied.html
custom-submit-execution-error.html
semantic-functional-summary.json
architecture-conformance.json
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

actual runの判定記録: `plan/GATE_C_READINESS_AUDIT.md`


## Visual Evidence intake

Semantic / Functional acquisitionとは別に、同じ`targetBuildRevision = T`へScreenshot等のVisual artifactを結び付ける。

### manifest scaffold

Visual manifestの`B / R / T`、required capture ID、criterion / capture referenceを手作業で複製しないため、audit tool自身からscaffoldを生成する。

```bash
mkdir -p /tmp/tei-gate-c-first-web-T/visual

GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-visual-evidence-audit \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -emit-manifest-scaffold \
  -visual-run-id visual-T \
  -evaluator-identity reference-human-review \
  -evaluator-version 1 \
  -evaluation-method screenshot-review \
  -viewport-width 1440 \
  -viewport-height 1000 \
  > /tmp/tei-gate-c-first-web-T/visual/visual-manifest.json
```

scaffoldはSemantic / Functional summaryから`B / R / T / Target Adapter / Host / source run identity`を転記し、required captureをdeterministic filenameで列挙する。

viewport高さに注意する。上の例の1440x1000では、標準ProjectionもCustom UIも下部が切れる。全体を撮る場合は`-viewport-height`を増やし、申告するviewportと実際に撮影したviewportを一致させる。第一Reference実績は1440x3000で、Playwright同梱の`chrome-headless-shell`を`--window-size=1440,3000 --screenshot=<png>`で使い、replay helperの`/capture/<id>`を撮影した。

さらに各captureへ、

- `sourceArtifact`
- `sourceSHA256`

を自動設定する。これはsame-runでTarget自身がrenderしたHTML artifactと、そのexact bytes digestである。Screenshotはこのreplay sourceへ明示的に結び付く。

重要:

```text
generated scaffold
  -> every criterion = unresolved
  -> Visual Fidelity passではない
  -> Gate C Readyではない
```

Screenshot取得後、attributed evaluatorが各criterionを`pass / fail / unresolved`へ更新し、Notesへ必要な根拠を残す。その後にVisual auditへ進む。

capture tool自体はこのReferenceで固定しない。browser automation、GUI agent、人手captureのいずれでもよいが、Screenshotと評価記録を残す。

監査入口:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-visual-evidence-audit \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -visual-manifest /tmp/tei-gate-c-first-web-T/visual/visual-manifest.json \
  -artifact-dir /tmp/tei-gate-c-first-web-T/visual \
  > /tmp/tei-gate-c-first-web-T/visual-audit.json
```

第一ReferenceではScreenshot artifactを`image/png`で受け入れる。これはCanonicalな画像形式ではなく、第一ReferenceのEvidence formatである。

### 必須capture

Visual manifestには少なくとも次のcaptureを含める。

```text
projection-index
custom-submit-initial
custom-submit-success
custom-submit-authorization-denied
custom-submit-validation-not-satisfied
custom-submit-execution-error
```

各captureは:

- ID
- surface
- state
- PNG filename
- media type
- viewport width / height
- source HTML artifact
- source HTML SHA-256

を持つ。

filenameはartifact directory直下のbasenameだけを許可し、path traversalやartifact directory外へのsymlink解決を受理しない。

### Visual capture state recipe

第一推奨は、Semantic / Functional same-runで保存した**Target-rendered HTML artifactのreplay**からScreenshotを取得する方法とする。

replay source mapping:

```text
projection-index
  <- projection-index.html

custom-submit-initial
  <- custom-submit-get.html

custom-submit-success
  <- custom-submit-success.html

custom-submit-authorization-denied
  <- custom-submit-authorization-denied.html

custom-submit-validation-not-satisfied
  <- custom-submit-validation-not-satisfied.html

custom-submit-execution-error
  <- custom-submit-execution-error.html
```

これらはすべて同じ`T`のactual Target responseから取得され、Semantic / Functional summaryへSHA-256が保存される。Visual manifest scaffoldはそのdigestを自動転記する。

したがってVisual reviewerはcredentialを再入力せず、保存済みHTMLをbrowserでrenderしてScreenshotを取得できる。

```text
same-run Target response HTML
  -> exact SHA-256 in Semantic / Functional summary
  -> sourceArtifact / sourceSHA256 in Visual manifest
  -> browser render
  -> PNG Screenshot
  -> attributed Visual evaluation
```

live Targetから直接captureする方式も許可するが、その場合もmanifestの`sourceArtifact / sourceSHA256`は同じstateのsame-run HTML responseへ一致させる。

### digest-verified replay helper

保存済みHTMLを手作業で探さず、安全なloopback routeからrenderするため:

- `tools/reference-first-web-target-visual-replay`

を使える。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-visual-replay \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -artifact-dir /tmp/tei-gate-c-first-web-T \
  -visual-manifest /tmp/tei-gate-c-first-web-T/visual/visual-manifest.json \
  -listen 127.0.0.1:18181
```

replay helperは起動前に:

- Semantic / Functional source qualification pass
- Semantic Conformance pass
- Functional Evidence pass
- Visual pending
- Gate C Readyではない
- required HTML artifactのexact SHA-256一致
- artifact directory外symlink拒否
- credential material非混入

を確認する。

capture route:

```text
http://127.0.0.1:18181/capture/projection-index
http://127.0.0.1:18181/capture/custom-submit-initial
http://127.0.0.1:18181/capture/custom-submit-success
http://127.0.0.1:18181/capture/custom-submit-authorization-denied
http://127.0.0.1:18181/capture/custom-submit-validation-not-satisfied
http://127.0.0.1:18181/capture/custom-submit-execution-error
```

`/capture-manifest.json`には各route、source artifact、source SHA-256、`B / R / T`を保持する。

`-visual-manifest`を渡した場合、`http://127.0.0.1:18181/`をVisual review indexとして表示する。ここでは:

- Visual run / evaluator provenance
- Screenshot filename
- capture route
- source artifact / SHA-256
- criterionのcurrent result
- criterionが参照するcapture

を一画面で確認できる。

review indexはmanifestを変更せず、評価結果を書き込まない。Visual auditの代替でもない。

helperはHTML bytesを起動時にmemoryへ固定してそのまま返す。業務処理やcredential入力を再実行しない。

```text
Visual replay helper
  != Target runtime re-execution
  != Screenshot Evidence
  != evaluator judgment
  != Gate C Ready
```

状態をlive Targetで再現する必要がある場合のdeterministic recipeは次とする。

```text
projection-index
  GET /

custom-submit-initial
  GET /custom-submit

custom-submit-success
  POST /custom-submit
  loginId = legacy-order-editor
  credential = editor-secret
  from = 2026-09-18T08:30:00Z
  row = 0

custom-submit-authorization-denied
  POST /custom-submit
  loginId = legacy-order-viewer
  credential = viewer-secret
  from = 2026-09-18T08:30:00Z
  row = 0

custom-submit-validation-not-satisfied
  POST /custom-submit
  loginId = legacy-order-editor
  credential = editor-secret
  from = 2026-09-18T08:00:00Z
  row = 1

custom-submit-execution-error
  POST /custom-submit
  loginId = legacy-order-editor
  credential = editor-secret
  from = 2026-09-18T07:00:00Z
  row = 2
  expected HTTP status = 422
```

これらのcredentialはReference runtime inputであり、Screenshot / manifest / evaluator notesへ値を転記しない。POST後のrendered pageではcredential fieldを再表示しない。

replay captureを使う場合、reviewerはcredentialを再入力する必要がない。

Semantic / Functional runnerは同じ状態のHTML artifactも保存するため、Visual evaluatorはScreenshotで見えているstateがraw functional artifactと対応しているかを照合できる。ただしHTML一致をScreenshot Evidenceの代用にはしない。

### Visual評価項目

Gate C contractに対応して少なくとも次を評価する。

```text
field-grouping-readability
relation-reference-comprehensibility
header-detail-locality
validation-feedback-distinguishable
authorization-denial-vs-system-error-distinguishable
business-result-vs-observation-distinguishable
standard-vs-custom-ui-difference-explainable
```

各項目は:

```text
pass
fail
unresolved
```

のいずれかとし、評価根拠となるcapture IDを明示する。

特に:

- validation feedback → `custom-submit-validation-not-satisfied`
- authorization denied vs system error → `custom-submit-authorization-denied` と `custom-submit-execution-error`
- Business Result vs Observation → `custom-submit-success`
- standard vs Custom UI → `projection-index` と `custom-submit-initial`

を参照する。

### evaluator provenance

Visual manifestには少なくとも:

- evaluator identity
- evaluator version
- evaluation method

を保持する。

人、VLM、image comparison等の方式をCanonical意味へ含めない。

### Visual auditの意味

Visual auditは:

- Semantic / Functional summaryが同じ`B / R / T`で成立済み
- Screenshot集合が完全
- ScreenshotがPNG artifactとして存在
- capture pathが安全
- Screenshot exact bytes SHA-256
- 各Screenshotの`sourceArtifact / sourceSHA256`がSemantic / Functional summaryのsame-run raw HTML digestへ一致
- required criterionが存在
- criterionが必要なcapture stateを参照
- credential materialがmanifestへ混入していない

ことを確認する。

全criteriaが`pass`でも、Visual audit単独ではGate C Readyを宣言しない。

期待:

```text
visualFidelityPassed = true   # evaluatorが全項目passの場合
gateCReady = false
```

最終Gate C判定はSemantic / Functional / Visualを別Evidenceとして揃えた後、Gate C contract全体へ戻って行う。


## Architecture negative Evidence

Semantic / Functional acquisition runnerは同じclean `T`で:

```text
architecture-conformance.json
```

も生成する。

実装:

- `tools/reference-first-web-target-architecture-evidence`

第一Referenceでは次をfail closedで確認する。

### R → T Canonical不変

```text
reference/workspace/**
reference/world.json
```

について、Handover Baseline revision `R`からTarget revision `T`までGit差分がないことを要求する。

これは第一Referenceのcoarseな不変条件であり、将来のproduction semantic diff taxonomyではない。

差分が存在する場合はWeb側で補完せず、baseline subject変更の可能性としてsuccessor Handoverへ戻る。

### Web Target authority boundary

`tools/reference-first-web-target`配下で:

- `.tei` authoring sourceを持たない
- `.sql` sourceを持たない
- `SELECT ... FROM`
- `INSERT INTO`
- `UPDATE ... SET`
- `DELETE FROM`
- DDL / MERGE

等のnative SQL本文を持たないことを確認する。

これは「すべての業務二重実装が数学的に存在しない」ことの証明ではなく、第一Referenceで観測可能なauthority boundary Pressureである。

### shared Command / Result boundary

actual `T`でfocused source testを実行し:

- standard JSON interactionとCustom UIが同じ`commandScenarioRunner`を利用
- Validation not-satisfied後にAuthorization / Commandへ進まない
- Command Execution ErrorがBusiness Result / Observationへ畳まれない
- Execution Error HTTP envelopeが`B / R / T`を保持

ことを再確認する。

### Repository source

Architecture Evidence自身も:

- clean full checkout
- fixed `GOENV=off / GOWORK=off / GOFLAGS=`
- baseline `R`が`T`のancestor
- run前後同一clean `T`

を要求する。

このEvidenceはVisual Evidenceの代替ではない。


## Final Gate C Readiness Audit

Semantic / Functional、Visual、Architectureのactual Evidenceが**同じ`B / R / T`**へ揃った後だけ最終監査を実行する。

実装:

- `tools/reference-first-web-target-gate-c-readiness-audit`

例:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-gate-c-readiness-audit \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -architecture-evidence /tmp/tei-gate-c-first-web-T/architecture-conformance.json \
  -visual-audit /tmp/tei-gate-c-first-web-T/visual-audit.json \
  > /tmp/tei-gate-c-first-web-T/gate-c-readiness-audit.json
```

### 最終監査が要求するもの

少なくとも:

- Semantic / Functional source qualification成功
- Semantic Conformance成功
- Functional Evidence成功
- Visual required captures完備
- Visual criteriaすべてresolved
- Visual Fidelity pass
- Architecture Evidenceが同一clean `T`でreproducible
- `R → T` Canonical World不変
- Web Target authority boundary成立
- standard / Custom UI shared Command boundary成立
- Business Result / Execution Error / Observation境界成立
- Target差分handoffが復元可能
- すべてのunresolvedが明示分類済み
- blocking unresolvedがない

ことを要求する。

Visual auditやSemantic / Functional auditが単独で`gateCReady = false`を返すのは意図した設計である。最終auditだけが三Evidenceとarchitecture conformanceを合わせてGate C判定を行う。

### unresolved分類

第一Referenceで既知の未解決は、黙ってsuccessへ変換しない。

```text
visual-fidelity-evidence-not-acquired
  -> 独立Visual Evidenceで解消された場合だけ resolved-by-visual-evidence

browse-environment-instance-binding-not-exposed-by-current-driver-api
  -> carry-forward / ISSUE-000026

actual-external-command-observation-not-acquired
  -> carry-forward / external-runtime-evidence
     synthetic Observationをactual external Evidenceへ格上げしない

target-capability-taxonomy-remains-candidate-until-second-target
  -> carry-forward / ISSUE-000026

general-target-neutral-presentation-taxonomy-not-yet-defined
  -> carry-forward / ISSUE-000026
```

これ以外の未知unresolvedが現れた場合、最終auditはfail closedで停止する。

### 第二Targetへの差分handoff

Semantic / Functional auditはactual runから次をTarget difference handoffへ残す。

- Application
- Target Adapter / Host
- observed Target Capability候補
- Runtime Binding
- execution artifact
- host ABI
- Execution / Trust observation
- Web固有差
- Canonicalへ持ち上げてはいけない差
- carry-forward unresolved

`go run`で起動したactual Evidence runではGo version / GOOS / GOARCH / launch mode / loopback bindをrun metadataから取得し、execution artifact / ABIを推測ではなく観測値としてhandoffする。

これは第一Targetで得た観測recordであり、Target CapabilityやExecution Profileの最終taxonomyではない。第二TargetのPressureを受けて`ISSUE-000026`で再検討する。

### Gate C Readyの意味

最終auditが`gateCReady = true`を返しても、次を意味しない。

- production-ready Web product
- actual external Oracle / production Command成功
- Target-neutral Presentation taxonomy完成
- 第二Target不要
- unresolved消滅

Gate C contractに定めた第一Targetの意味・操作・視覚・architecture境界が、同一provenanceのEvidenceから再確認できたことだけを意味する。

actual final auditを取得するまでは、source implementationだけからGate C Readyを宣言しない。
