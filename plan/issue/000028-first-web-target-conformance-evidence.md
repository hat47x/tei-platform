# ISSUE-000028: 第一Web TargetのConformance Evidence契約を実証する

- 状態: 保留
- GitHub Issue: #270
- 本格着手条件: Gate B Ready + Handover Baseline固定

## 背景

`ISSUE-000007` / GitHub Issue #70では、第一Web Targetを通してApplication上の意味、Target-neutralなPresentation候補、Web固有差分を分離し、Gate Cを評価します。

TEIでは第一Web Targetの評価を一つの「成功」へ畳まず、次の三層を別Evidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

このIssueではWeb frameworkを先に決めるのではなく、Gate Cで何をEvidenceとして残すかをReference Scenarioから実証します。

契約の基線は[`GATE_C_EVIDENCE_CONTRACT.md`](../GATE_C_EVIDENCE_CONTRACT.md)、Driver baselineの固定規則は[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)に置きます。

## Gate Bから受け取るHandover Baseline

本IssueはGate B通過前にWeb実装を開始しません。

Gate B Ready時に`ISSUE-000030` / GitHub Issue #291で第一Handover Baselineを作成し、少なくとも次へ追跡できる状態から開始します。

- repository revision
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

Gate C途中にApplication / Driverの意味が変わった場合、既存baselineへ黙って上書きせず、Gate Bに関係するEvidenceを再確認して新しいbaselineを発行します。

## 実証するEvidenceの三層

### Semantic Conformance

Handover BaselineとWebが同じCanonical subjectを利用していることを確認します。

Web固有のcomponent、route、HTTP status、message等を業務上の意味の正本にしません。

### Interaction / Functional Evidence

利用者がReference ScenarioをWeb上で実行できることを確認します。

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
- header-detail edit
- validation satisfied
- validation not satisfied
- authorization allowed
- authorization denied
- command invocation
- Business Result
- Execution Error
- Observation

各Scenarioについて、Handover Baseline、Application Pattern、Canonical subject、Driver Scenario、Web interaction、result classificationを追跡します。

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

## Evidence artifact

最初から最終schemaを固定しませんが、少なくとも次を復元できる形式を実証します。

- Handover Baseline identity
- scenario ID
- repository / build / run revision
- Application / Pattern ID
- Canonical references
- Driver Scenario reference
- Target Adapter / Host
- Target Capability observations
- Runtime Binding / Execution / Trust observations
- interaction steps
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
5. Execution / Trust差
6. Web固有実装
7. Host control plane固有差
8. Canonicalへ持ち上げてはいけない差
9. unresolved

第一Targetだけでtaxonomyを確定しません。

## 完了条件

- [x] `GATE_C_EVIDENCE_CONTRACT.md`を定義する
- [x] Gate B→C Handover baseline契約を定義する
- [ ] Gate B Ready時に第一Handover Baselineを受け取る
- [ ] 各Gate C Evidence runからHandover Baselineへ追跡できる
- [ ] Driver baselineとWeb Scenarioを対応付ける
- [ ] Semantic / Functional / Visual Evidenceを別々に保存・再確認できる
- [ ] Business Result / Execution Error / Observationを別々に扱える
- [ ] architecture negative casesを再現可能に確認できる
- [ ] Web Adapter除去時のCanonical不変を確認できる
- [ ] standard ProjectionとCustom UIを同じ上位意味から実行できる
- [ ] Target Capability / Execution / Trust差を第二Targetへ渡せる
- [ ] evaluator固有scoreがSemantic / Functional合否を上書きしない
- [ ] Evidence artifactがCanonicalの第二正本になっていない
- [ ] Gate CをIssue closeではなくEvidenceから判定できる

## 非目標

- Gate B前のWeb実装
- Web frameworkの先行固定
- 完全なPresentation IR
- 完全なTarget Capability taxonomy
- pixel-perfect design system
- VLM / GUI agent標準化
- 一律のvisual score threshold
- Handover artifactのCanonical化
- 第二Target実装

## 関連文書

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
- `plan/issue/000030-gate-b-to-c-handover-baseline.md`
