# 000048 Business Procedure Contract ADR readiness

Status: In Progress

## 目的

Business Procedureの最小production Canonical contractについて、**ADRを起票してよい状態**へ到達させる。

対象となるcontract decisionは次である。

```text
Business Procedure minimum meaning
O1 protected-target representation
Procedure-scoped Step address
thin Handoff relation
separate normalized Business semantic plane
migration-adapter / candidate-only-loader architecture direction
freshness / authority-cutover invariants
```

ADR前に必要なのは**contractを反証できるEvidence**であり、最終production substrateの完成ではない。

## 上位基線

- 三段readinessの原典: [`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`](../BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md)
- 判定の正本: [`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](../BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)
- 設計品質: [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- 担当割り当て: [`UPSTREAM_BREAKDOWN_2026-09.md`](../UPSTREAM_BREAKDOWN_2026-09.md) §4
- 関連RQ: RQ-002 / RQ-003 / RQ-017 / RQ-018 / RQ-021 / RQ-022 / RQ-025 / RQ-026
- PAS: PAS-07

## 担当境界

本Issueが所有するのは、**Contract ADR readinessの到達と、ADRの起票**である。

| 所有する | 所有しない |
| --- | --- |
| blocker A / Bの解消 | cross-view Reference・checker・Pressure testの実装（[ISSUE-000042](000042-business-procedure-cross-view-reference.md)） |
| ADR起票と、そこへ記録する判断 | Ready / Not Ready判定そのもの（Pre-ADR Auditが正本） |
| defer判断のADRへの記録 | production採用条件（[ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)） |
| — | second corpusの取得（[ISSUE-000050](000050-second-realistic-handoff-corpus.md)） |
| — | Studio / AIへの接続（[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md) / [ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md)） |

ISSUE-000042のoperational checklistを本Issueへ再掲しない。checklistの消化はISSUE-000042、判定はPre-ADR Audit、起票は本Issueが持つ。

## 現在のblocker

Pre-ADR Auditのcurrent判定をそのまま担当する。

- **A. actual Repository runner green未取得** — `gofmt` / `go vet` / `go test` / checkerに加え、source-uniqueness、source-root membership、candidate-only fresh-process、second-corpus non-inference Pressureを同一revision `R`でactual実行する。実行手順は[`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`](../BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md)に従い、exact revision `R`・runner kind・environmentを実行時に記録する
- **B. independent second realistic TEI-original responsibility transfer未充足** — ISSUE-000050では製造・在庫連携のsecond-corpus research target、provenance、Evidence nature、Business Seed Acceptance Contractまで整理済み。ただしaccepted independent Business seed、Procedure / Step、自然に生じたresponsibility transfer、`from/to/condition` positive Evidenceは未成立

Blocker Bは「Handoff fixtureが二つあること」ではない。Handoff検証を目的に業務を逆算せず、独立したBusiness needから自然に生じた責任継続でcurrent H1/R3の一般性をpressureできることを要求する。

Candidate-only fresh-process feasibilityはADR前に必要なEvidence項目だが、現在はsource-level実装済みであり、独立したmaterial blockerとしてAと二重計上しない。Blocker Aのactual executionで同じrevision上にEvidenceを得る。

## 現在の作業順序

Material blockerはA / Bの二つだが、positive Evidence取得の順序は独立ではない。

```text
Blocker A
  actual Gate A green
  ↓
Blocker B
  Handoff検証とは独立したBusiness need / Reference seed
  ↓
  ISSUE-000050 S1〜S8 quality review
  ↓
  minimal accepted Procedure / Step
  ↓
  responsibility transferが自然に成立した場合だけ
  explicit second Handoff
  ↓
  from / to / condition losslessness
  ↓
Pre-ADR readiness再判定
```

Gate A green前にISSUE-000050で進めてよいのはcorpus / provenance /利用条件 / Evidence nature / candidate design question / seed acceptance criteriaまでである。

Gate A green前には:

- candidate Action poolをaccepted Step sequenceへ昇格しない
- exact actor / trigger / responsibility boundaryを補完しない
- second-case検証のためにaccepted Procedure / Handoff fixtureを作らない
- `from/to/condition`をpositive Evidenceとしてauthorしない
- system result / physical Observation / actor差からresponsibility transferをbackfillしない

したがってcurrent execution priorityは**Blocker A**である。Actual runnerが利用可能になった時点で、second Handoff authoringより先にGate Aを同一revisionで実行する。

## ADR前に要求するEvidence

1. actual Repository mechanical green
2. D4 + singular O1 / S4-A / H1/R3のactual losslessness / fail-closed
3. multi-source normalization、source freshness、Conformance-base freshness、coordinated change-setのactual test
4. legacy + candidate simultaneous authority rejectのactual test
5. candidate-only sourceのfresh-process再構成feasibility
6. independent Business needから成立したsecond realistic TEI-original responsibility transferで`from/to/condition`がlossless
7. second caseでsystem result / external actuality / Business continuationを短絡せず、unknownを推測で埋めないこと
8. derived Data / CRUD guardrailがactualに推測を抑制すること

6 / 7は同じsecond corpusから取得できるが、Handoff fieldの形式確認とBusiness design qualityを同じclaimへ潰さない。

## ADRの前提にしないもの

次が無いことはimplementation gapであり、Business schemaへfieldを追加する根拠ではない。

- generic / shared Acceptance service実装の完成
- Studio / AI authoring統合
- production multi-file writerの完成
- production repository transaction APIの完成
- final hash / ETag / revision-token schema
- production migration command UX
- Business design quality review categoryの永続enum化
- Handoff condition成立Evidenceの専用field

## ADRへ記録するdefer判断

契約決定時に、次を「negative Evidenceに基づくdefer」として明記する。放置ではない。

- **C/U/D field**: qualifying C / D / non-state U caseの不存在はfield追加の根拠にしない。unknownはunknownのまま保持し、Database write / Pattern kind / Lifecycleから推測しない。将来のgeneric mutation / effect contract採用時のreopen triggerとして残す
- **`ir.World.Business` embedding**: separate Business planeでもcross-view Conformanceをfail closedできるため、embeddingはdeferする。concrete insufficiencyが出た時だけplacementをreopenする
- **generic ref-role taxonomy**: 複数のrealistic pressureから必要性を判断する（ISSUE-000042 checklist 19）
- **Evidence-nature persistence**: source-backed / accepted / derived / proposal / unresolvedのreview distinctionは維持するが、専用Canonical enumはrealistic Referenceでdurable trace lossが出るまでdeferする
- **Handoff Evidence field**: conditionとObservation / Evidenceを意味上は分けるが、existing Source / Provenance mechanismで不足が証明されるまで専用fieldを追加しない

## 完了条件

- [ ] blocker A / Bが解消し、Pre-ADR AuditがContract ADR readyと判定した
- [ ] 上記Evidenceを、取得revisionとEvidence natureつきで参照できる
- [ ] second corpusがISSUE-000050 S1〜S8を満たし、Handoff testのためにBusiness meaningを捏造していない
- [ ] ADRを起票し、採用したcontractとdefer判断（C/U/D、placement、taxonomy、Evidence nature、Handoff Evidence）を記録した
- [ ] Lane C Pre-ADR Gate CとProduct Gate Cを混同しない記述になっている
- [ ] Gate B必須Evidence三系統をLane Cのactual runner greenで代替していない

## 非目標

- Workflow Engine、full Procedure schema、global Step identity、full Business Flow schemaの固定
- production採用（ISSUE-000049）
- Studio / AI end-to-end実装（Product Gate C後）
- 上流pressure文書の結論の再判定
- second caseを成立させるためのsynthetic actor / Data / Application / Handoffの作成

## 関連

- [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md): Lane C Reference / Evidenceの親
- [ISSUE-000049](000049-business-authoring-production-adoption-readiness.md) / [ISSUE-000050](000050-second-realistic-handoff-corpus.md)
- [`RQ025_RQ026_ACCEPTANCE_TRACE.md`](../RQ025_RQ026_ACCEPTANCE_TRACE.md) / [`SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md`](../SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md)
