# ISSUE-000028: 第一Web TargetのConformance Evidence契約を実証する

- 状態: In Progress — actual Evidence取得済み、Gate C Ready（T = `0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、2026-09-19）。残は条件付き項目（baseline変更時の切替 / Replayable Behavior利用時のtrace）
- GitHub Issue: #270
- 本格着手条件: Gate B Ready + Handover Baseline固定
- Product Acceptance Scenario: PAS-04 primary

## 現在の入口

2026-09-19: clean checkoutの`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`でactual Evidenceを取得し、Gate C Readyと判定した（判定記録: [`GATE_C_READINESS_AUDIT.md`](../GATE_C_READINESS_AUDIT.md)）。Visual Fidelityの評価者は単一のAI評価者で、人による確認ではない。以下は、Ready到達までの入口の記述として保持する。

着手条件だったGate B ReadyとHandover Baseline `B / R`固定は成立した。

`ISSUE-000007`では`tools/reference-first-web-target`を進め、projection shellに加えてread-only Driver sliceとAuthentication / Validation / Authorization / Command / separate ObservationのScenario sourceまで接続している。manifest / HTTP responseには`B / R / T`を別identityとして保持する。

Authentication / AuthorizationのBusiness Outcome、Validation結果、Command Business Result、Observationを別結果として保持するsource pressureも追加した。さらにCustom UIとJSON APIが同じ`commandScenarioRunner`へ同じsemantic inputを渡すsource pressureを追加し、Custom UIを第二の業務正本にしていない。

さらにactual Target runからsame-run raw artifactを取得する`tools/run-first-web-target-gate-c-evidence.sh`と、raw exact bytesを再監査する`tools/reference-first-web-target-evidence-audit`を追加している。successful pathだけでなくAuthentication denied / Authorization denied / Canonical Validation not-satisfied / Command Invocation Execution Errorを別artifactとして保持する。実行手順は[`GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`](../GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)へ整理した。

ただし、runner / audit sourceや`httptest`用test sourceの存在をGate C Evidenceへ読み替えない。actual clean Target build/runに紐づくSemantic / Functional artifactは未取得であり、Visual Fidelity Evidenceも未取得である。Semantic / Functional audit sourceもVisual未取得時は`gateCReady = false`を固定する。

Visual側には`tools/reference-first-web-target-visual-evidence-audit`を追加し、同じ`B / R / T`に対するScreenshot集合とattributed evaluator recordを受け入れる。capture toolは固定せず、必須capture state / criterion / artifact fingerprint / provenanceだけをReference pressureとして扱う。Visual criteriaが全passでもaudit単独では`gateCReady = false`のままとする。

Visual captureに先立ち、Custom UIのsuccess / authorization-denied / validation-not-satisfied / execution-errorを構造化HTMLへ分離し、Semantic / Functional runnerが同じstateのraw HTML artifactを保存・auditするようにした。standard Projectionについても`projection-index.html`をsame-run raw artifactとして保存する。

Visual manifest scaffoldはSemantic / Functional summary内のraw artifact digestから、各required captureへ`sourceArtifact / sourceSHA256`を自動設定する。これによりScreenshotはsame-run Target-rendered HTMLのexact bytesへ追跡でき、reviewerはcredential付き操作を再入力せず保存済みHTML replayからcaptureできる。

Runbookには各必須Screenshot stateを再現するdeterministic form input recipeも残すが、第一推奨はdigest-bound HTML replayとする。

その入口として`tools/reference-first-web-target-visual-replay`を追加し、source digest一致・artifact directory境界・credential非混入を確認した上でrequired stateをloopback routeへexact bytesで配信する。helperはTarget runtime再実行でもVisual evaluatorでもない。

Visual manifest scaffoldを併用すると、review indexでcapture route / Screenshot filename / source digest / evaluator / criterion-capture対応を一覧できる。review helperは判定を持たず、Visual audit sourceを第二の正本にしない。

これはVisual captureの準備Evidenceであり、HTML artifactをScreenshotへ読み替えない。

actual Screenshot / evaluator Evidenceは未取得である。

Visual manifestのprovenance / required capture / criterionを手書きで複製しないため、Visual audit toolにmanifest scaffold modeも追加した。scaffoldはSemantic / Functional summaryから同じ`B / R / T`を引き継ぐが、全criterionを`unresolved`で生成するため、生成しただけではVisual passにならない。

Gate C final readiness audit sourceとして`tools/reference-first-web-target-gate-c-readiness-audit`も追加した。Semantic / Functional、Visual、Architectureが同じ`B / R / T`へ揃い、既知unresolvedが明示分類され、blocking unresolvedがない場合だけ最終`gateCReady`を判定する。各下位auditが単独でReadyを宣言しない構造を維持する。

現時点ではfinal audit sourceもSource Evidenceであり、actual `gate-c-readiness-audit.json`は未取得である。

Architecture negative casesについては`tools/reference-first-web-target-architecture-evidence`を追加し、同一clean `T`でHandover `R`からのWorkspace / generated World不変、Web Target authority boundary、standard / Custom UI shared Command boundaryを再確認する。Semantic / Functional acquisition runnerから`architecture-conformance.json`としてsame-run取得する。

actual architecture artifactは未取得である。

## 背景

`ISSUE-000007` / GitHub Issue #70では、第一Web Targetを通してApplication上の意味、Target-neutralなPresentation候補、Web固有差分を分離し、Gate Cを評価します。

TEIでは第一Web Targetの評価を一つの「成功」へ畳まず、次の三層を別Evidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

このIssueではWeb frameworkを先に決めるのではなく、Gate Cで何をEvidenceとして残すかをReference Scenarioから実証します。

契約の正本は[`GATE_C_EVIDENCE_CONTRACT.md`](../GATE_C_EVIDENCE_CONTRACT.md)、Gate Bからのbaseline固定規則は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)です。

## Gate Bから受け取るHandover Baseline

Gate B Ready時に`ISSUE-000030` / GitHub Issue #291で第一Handover Baseline `B`を作成します。

少なくとも次へ追跡できる必要があります。

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- `order_entry` Application
- browse / record / reference / header-detail / command
- Data / Relation / Query / Datasource
- Validation
- Authentication / Authorization
- Capability / Binding
- Business Result / named Outcome
- Execution Error
- Observationに使う明示read境界
- Driver Scenario identity
- Gate B Evidence
- unresolved

Handover recordはCanonicalの第二正本ではありません。

## Baseline revision RとTarget build revision Tを分ける

第一Web Targetの実装commitを含むため、Gate CのTarget build revision `T`は通常、Gate B baseline revision `R`とは異なります。

```text
Handover Baseline B at R
        ↓ meaning baseline
Web Target build at T
        ↓ execution
Gate C Evidence
```

Gate C Evidenceでは次を別々に残します。

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- `targetBuildRevision = T`
- Target Adapter / Host identity
- run identity

`T`にWeb Adapter等を追加することは許容しますが、baseline subjectの意味を黙って変更しません。

Application / Driver meaningを変更する必要が生じた場合は、authoritative sourceへ戻って必要なEvidenceを再確認し、新しいHandover Baseline `B'` / `R'`を発行します。旧baselineのGate C Evidenceを新baselineへ黙ってcarry-forwardしません。

## 実証するEvidenceの三層

### Semantic Conformance

Handover BaselineとWebが同じCanonical subject / meaningを利用していることを確認します。

Web固有component、route、HTTP status、message等を業務上の意味の正本にしません。

### Interaction / Functional Evidence

利用者がTarget build `T`上でReference Scenarioを実行できることを確認します。

browser automation、GUI agent、人による手順等は実行手段として交換可能にします。

### Visual Fidelity Evidence

Web Targetとしてのlayout、grouping、feedback、readability等を記録します。

VLM、screenshot diff等を使っても、そのscoreだけでSemantic / Functional合格を決めません。

## Scenario Matrix

少なくとも次をHandover BaselineのDriver Scenarioと一対一または明示的な合成関係で対応付けます。

- browse
- record open
- reference lookup
- header-detail read
- header-detail edit（baselineがwrite意味を持つ場合）
- validation satisfied
- validation not satisfied
- authorization allowed
- authorization denied
- command invocation
- Business Result
- Execution Error
- Observation

各Scenarioについて、Handover Baseline、Application Pattern、Canonical subject、Driver Scenario、Web interaction、result classificationを追跡します。

Handover Baselineに存在しない意味をWeb側だけで追加してcoverage不足を隠しません。

## Architecture negative cases

少なくとも次を確認します。

- Web Adapterを除去してもCanonical Worldが変わらない
- Web固有fieldをCanonical Semantic Layerへ追加しない
- native SQLをWeb側へ複製しない
- Capability implementationをWeb側へ複製しない
- Validation ruleをWeb側の独立正本にしない
- Authorization ruleをWeb側の独立正本にしない
- read-only Datasourceへwriteしない
- HTTP statusからSemantic Outcomeを逆定義しない
- named OutcomeとExecution Errorを一つのerrorへ畳まない
- Command acknowledgementだけでObserved state成立としない
- standard ProjectionとCustom UIが同じCanonical意味へ接続する
- Handover Baselineに不足した意味をWeb側だけで補わない
- `targetBuildRevision = T`を`baselineRepositoryRevision = R`へ黙って読み替えない

## Evidence artifact

最初から最終schemaを固定しませんが、少なくとも次を復元できる形式を実証します。

- Handover Baseline ID
- `baselineRepositoryRevision = R`
- `targetBuildRevision = T`
- scenario ID
- build / run identity
- Application / Pattern ID
- Canonical references
- Driver Scenario reference
- Target Adapter / Host
- Target Capability observations
- Runtime Binding observations
- execution artifact / ABI / Host identity / observations（適用可能な範囲）
- Execution / Trust observations
- interaction steps
- replayable behavior identity / precondition / expected semantic operation（利用する場合）
- raw Target Evidence reference（replay / normalized Evidenceを利用する場合）
- normalized event / behavior Evidence referenceとraw provenance（利用する場合）
- Business Result / named Outcome / Execution Error / Observation classification
- semantic result
- functional result
- visual artifact reference
- performance measurement reference
- evaluator / automation identity（利用した場合）
- unresolved / Diagnostic

Evidence artifactはderivedです。Canonicalへ暗黙に逆書きしません。

## Evaluator / automation

利用するtest tool、browser automation、GUI agent、VLM等は交換可能にします。

利用した場合は、少なくともtool identity / version、入力artifact、何のEvidence種別を評価したかを追跡します。

外部modelのjudge scoreを唯一の合否根拠にはしません。judge結果より入力artifactを優先して残し、別evaluatorで再評価できるようにします。

## 第二Targetへの引き継ぎ

第一Web Targetで得た差を次へ分類して`ISSUE-000026` / GitHub Issue #259へ渡します。

1. Application上の意味
2. Target-neutral Presentation候補
3. Target Capability候補
4. Runtime Binding差
5. execution artifact / ABI / Host差（観測可能な範囲）
6. Execution / Trust差
7. Web固有実装
8. Host control plane固有差
9. Canonicalへ持ち上げてはいけない差
10. unresolved

第一Targetだけでtaxonomyを確定しません。

## 完了条件

- [x] `GATE_C_EVIDENCE_CONTRACT.md`を定義する
- [x] Gate B→C Handover baseline契約を定義する
- [x] baseline `R`とTarget build `T`を別identityとしてEvidence契約へ定義する
- [x] Gate B Ready時に第一Handover Baselineを受け取る
- [x] 各Gate C Evidence runからHandover Baseline ID / `R`へ追跡できる
- [x] 各runの`targetBuildRevision = T`を追跡できる
- [ ] `T`でbaseline meaningが変わった場合、新Handover Baselineへ切り替える（条件付き。今回は`R → T`でCanonical不変を確認しており切替は発生していない）
- [x] Driver baselineとWeb Scenarioを対応付ける
- [x] Semantic / Functional / Visual Evidenceを別々に保存・再確認できる
- [ ] replayable behaviorを利用する場合、Handover Scenario / semantic operation / raw Target Evidenceへtraceできる（任意。今回のGate C Evidenceでは未利用）
- [x] normalized Evidenceを利用する場合、raw Target Evidenceへ戻れる
- [x] Business Result / Execution Error / Observationを別々に扱える
- [x] architecture negative casesを再現可能に確認できる
- [x] Web Adapter除去時のCanonical不変を確認できる
- [x] standard ProjectionとCustom UIを同じ上位意味から実行できる
- [x] Target Capability / Runtime Binding / execution artifact・ABI・Host / Execution・Trust差を第二Targetへ渡せる
- [x] evaluator固有scoreがSemantic / Functional合否を上書きしない
- [x] Evidence artifactがCanonicalの第二正本になっていない
- [x] Gate CをIssue closeではなくEvidenceから判定できる

## 非目標

- Gate B前のWeb実装
- Web frameworkの先行固定
- 完全なPresentation IR
- 完全なTarget Capability taxonomy
- Gate Cのためだけの複数execution artifact / Wasm比較
- execution artifact / ABI / Hostのproduction taxonomy / schema
- pixel-perfect design system
- VLM / GUI agent標準化
- Replayable Behavior / normalized event形式のproduction標準化
- Universal Event Modelの先行設計
- 一律のvisual score threshold
- Handover artifactのCanonical化
- Target buildをbaseline revisionと同じGit revisionへ固定すること
- 第二Target実装

## 関連文書

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
