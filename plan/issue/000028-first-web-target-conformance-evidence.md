# ISSUE-000028: 第一Web TargetのConformance Evidence契約を実証する

- 状態: 保留
- GitHub Issue: #270
- 本格着手条件: Gate B

## 背景

`ISSUE-000007` / GitHub Issue #70では、第一Web Targetを通してApplication上の意味、Target-neutralなPresentation候補、Web固有差分を分離し、Gate Cを評価します。

技術リサーチからは、UI生成・評価において機能成立と視覚品質を分ける必要があることが確認できました。TEIではさらに上位にCanonical meaningの保持を置き、次の三層を別Evidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

このIssueでは、Web frameworkを先に決めるのではなく、Gate Cで何をEvidenceとして残すかをReference Scenarioから実証します。

契約の基線は[`GATE_C_EVIDENCE_CONTRACT.md`](../GATE_C_EVIDENCE_CONTRACT.md)に置きます。

## 関連するProduct Requirement

- RQ-011 同じCanonical meaningを複数Projectionで扱えること
- RQ-012 Driverを設計検証環境として利用できること
- RQ-013 Target-neutral Presentation / Interaction
- RQ-018 Command / Observation
- RQ-021 Reference / Conformance
- RQ-022 検証段階の明示
- RQ-024 Target別性能

## Gate Bから受け取るもの

本IssueはGate B通過前にWeb実装を開始しません。

Gate Bから少なくとも次をbaselineとして受け取ります。

- browse
- record
- reference
- header-detail
- Validation
- Authentication / Authorization
- command
- Business Result / named Outcome
- Execution Error
- Observation
- Datasource read / read-write境界
- Scenario Evidence

Gate Bの最終形が変わった場合、Web側で独自に補うのではなくbaselineを更新します。

## 実証するEvidenceの三層

### Semantic Conformance

DriverとWebが同じCanonical subjectを利用していることを確認します。

Web固有のcomponent、route、status、message等を業務上の意味の正本にしません。

### Interaction / Functional Evidence

利用者がReference ScenarioをWeb上で実行できることを確認します。

browser automation、GUI agent、人による手順等は実行手段として交換可能にします。

### Visual Fidelity Evidence

Web Targetとしてのlayout、grouping、feedback、readability等を記録します。

VLM、screenshot diff等を使っても、そのscoreだけでSemantic / Functional合格を決めません。

## Scenario Matrix

少なくとも次をDriver baselineと一対一または明示的な合成関係で対応付けます。

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

各Scenarioについて、Application Pattern、Canonical subject、Driver baseline、Web interaction、result classificationを追跡します。

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

## Evidence artifact

最初から最終schemaを固定しませんが、少なくとも次を復元できる形式を実証します。

- scenario ID
- repository revision
- build / run identity
- Application / Pattern ID
- Canonical references
- Driver baseline reference
- Target Adapter / Host
- Target Capability observations
- Execution / Trust observations
- interaction steps
- semantic result
- functional result
- visual artifact reference
- Business Result / Execution Error / Observation classification
- performance measurement reference
- unresolved / Diagnostic

Evidence artifactはderivedです。Canonicalへ暗黙に逆書きしません。

## Evaluator / automation

利用するtest tool、browser automation、GUI agent、VLM等は交換可能にします。

利用した場合は、少なくともtool identity / version、入力artifact、何のEvidence種別を評価したかを追跡します。

外部modelのjudge scoreを唯一の合否根拠にはしません。

## 第二Targetへの引き継ぎ

第一Web Targetで得た差を次へ分類して`ISSUE-000026` / #259へ渡します。

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

- [ ] `GATE_C_EVIDENCE_CONTRACT.md`に沿うEvidence形式をReferenceで実証する
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
- 第二Target実装

## 関連文書

- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/TECH_RESEARCH_INTEGRATION_2026-09.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/issue/000006-application-driver-vertical-slice.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
