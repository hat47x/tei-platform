# 000048 Business Procedure Contract ADR readiness

- 状態: In Progress

## このIssueを最初に読むときの要点

このIssueは、Business Procedureの実装を完成させるための作業ではありません。

ここで確かめたいのは、**現在検討している最小contractを長く残るArchitecture判断としてADRへ固定してよいだけの反証可能なEvidenceがそろったか**です。

```text
Reference / pressure test
        ↓
「このcontractで説明できない現実が残っていないか」を確認
        ↓
Pre-ADR Audit
        ↓
ADRへ固定してよい / まだ早い
```

したがって、runnerが動いた、schema候補が書けた、Handoff fixtureが二つになった、といった個別の達成だけでは完了しません。

### このIssueが減らす不確実性

主に次の問いへ答えます。

1. 現在のBusiness Procedure最小contractは、実際のRepository上でlosslessかつfail-closedに扱えるか。
2. 最初の`order_entry`だけに過適合せず、独立した二つ目の現実的なBusiness caseでも責任移譲を説明できるか。
3. まだEvidenceがない概念を、将来必要そうという理由だけでCanonical fieldへ入れていないか。
4. ADRへ固定する判断と、production source authorityへ採用する判断を分離できているか。

### 完了すると何を判断できるか

このIssueが完了すると、**Business Procedureの最小contractをADRへ記録し、そのcontractを前提にproduction adoptionの検証へ進んでよい**と判断できます。

逆に、このIssueが完了しても次はまだ言えません。

- production authoringへ切り替えてよい
- Studio / AIからBusiness sourceを書き換えてよい
- Product Gate CがReadyである
- Workflow Engineやfull Procedure schemaが必要である

production採用は[ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)、Studio / AI接続は[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md) / [ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md)が別に扱います。

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
- 設計品質: [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../design/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- second Reference acceptance: [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](../LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)
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

- **A. actual Repository runner green — 充足（2026-09-20、R = `e2856e6`、[Pre-ADR Audit](../BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§13）** — 次を同一revision `R`でactual実行した: `gofmt` / `go vet` / `go test` / checkerに加え、source-uniqueness、source-root membership、candidate-only fresh-process、second-corpus non-inference Pressureを同一revision `R`でactual実行する。実行手順は[`BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md`](../BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md)に従い、exact revision `R`・runner kind・environmentを実行時に記録する
- **B. independent second realistic TEI-original responsibility transfer — Evidence取得済み（2026-09-21、R = `82e4301`、[ISSUE-000050](000050-second-realistic-handoff-corpus.md)実施記録）。ただし作成者self-reviewのみで、独立reviewとPre-ADR Auditによる再判定は未実施** — `order_entry`から独立した製造・入庫のReference（二つのProcedureと、自然に生じた責任のHandoff一件）をauthorし、RRB-P1、RRB-P2（source / test level。項目4は未確認）、RRB-P3（Gate A A1〜A8 PASS、critical 39件）のEvidenceを取得した。Handoff一般の妥当性、独立reviewを経たこと、Contract ADR readyは主張しない

Blocker Bは「Handoff fixtureが二つあること」ではない。Handoff検証を目的に業務を逆算せず、独立したBusiness needから自然に生じた責任継続でcurrent H1/R3の一般性をpressureできることを要求する。

Candidate-only fresh-process feasibilityはADR前に必要なEvidence項目だが、現在はsource-level実装済みであり、独立したmaterial blockerとしてAと二重計上しない。Blocker Aのactual executionで同じrevision上にEvidenceを得た（2026-09-20、A7のcritical 26件がPASS）。

## 現在の作業順序

Material blockerはA / Bの二つだが、positive Evidence取得の順序は独立ではない。

```text
Blocker A
  actual Gate A green
  ↓
Blocker B
  Handoff検証とは独立したBusiness need / Reference seed
  ↓
  Lane B RRB-P1 acceptance review
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

Gate A green前にISSUE-000050で進めてよいのはcorpus / provenance /利用条件 / Evidence nature / candidate design question / RRB-P1適用準備までである。

Gate A green前には:

- candidate Action poolをaccepted Step sequenceへ昇格しない
- exact actor / trigger / responsibility boundaryを補完しない
- second-case検証のためにaccepted Procedure / Handoff fixtureを作らない
- `from/to/condition`をpositive Evidenceとしてauthorしない
- system result / physical Observation / actor差からresponsibility transferをbackfillしない

したがってGate Aをsecond Handoff authoringより先に同一revisionで実行し、2026-09-20にPASSした。Blocker Bは2026-09-21にEvidenceを取得した。2026-09-24に作成者とは別系統の生成AI（OpenAI GPT-5.6 Sol）が独立reviewを実施し、[`BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_RESULT_2026-09.md`](../BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_RESULT_2026-09.md)へ`review-blocked`を記録した。主なblocking findingは、conditionがtarget Step ST-00のmeaningを内包すること、count accountability transferとU-4/U-6/U-9の不整合、RRB-P2 item 4未確認である。current execution priorityはこれらのmeaning / pressure修正と必要な再reviewであり、その後にPre-ADR AuditがBlocker Bを再判定する。2026-09-24のremediation incrementではF-1/F-2/F-4とsame-actor pressureをsource/testへ反映したが、RRB-P2 item 4は未解決であり、remediation authorとreviewerが同一のため独立再reviewも必要である。旧R = `82e4301`のGate A PASSをcurrent remediation revisionへ引き継がない。独立review自体はAudit再判定ではなく、Contract ADRはNot Readyを維持する。

## ADR前に要求するEvidence

Pre-ADR AuditのGateを詳細化する作業チェックとして、次を追う。**この一覧は新しいmaterial blockerを追加せず、Ready / Not Ready判定はPre-ADR Auditを正とする。**

1. actual Repository mechanical green
2. D4 + singular O1 / S4-A / H1/R3のactual losslessness / fail-closed
3. multi-source normalization、source freshness、Conformance-base freshness、coordinated change-setのactual test
4. legacy + candidate simultaneous authority rejectのactual test
5. candidate-only sourceのfresh-process再構成feasibility
6. RRB-P1を満たすindependent Business sourceから自然に成立したsecond realistic TEI-original responsibility transferで`from/to/condition`がlossless
7. Blocker BのEvidence record上、system result / external actuality / Business continuationを短絡せず、unknownを推測で埋めていないことを説明できる
8. derived Data / CRUD guardrailがactualに推測を抑制すること

6 / 7はBlocker Bの同じEvidence packageを形式と品質の別観点からreviewするもので、第三のmaterial blockerではない。

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
- [ ] second corpusがLane B RRB-P1 acceptance条件を満たし、Handoff testのためにBusiness meaningを捏造していない
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

- [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../design/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](../LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md): Lane C Reference / Evidenceの親
- [ISSUE-000049](000049-business-authoring-production-adoption-readiness.md) / [ISSUE-000050](000050-second-realistic-handoff-corpus.md)
- [`RQ025_RQ026_ACCEPTANCE_TRACE.md`](../RQ025_RQ026_ACCEPTANCE_TRACE.md) / [`SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md`](../SUBSYSTEM_CANONICAL_REOPEN_GATE_2026-09.md)