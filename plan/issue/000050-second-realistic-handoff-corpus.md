# 000050 second realistic TEI-original Handoff corpus

Status: Proposed

## 目的

Contract ADR blocker Cを解消する。`order_entry`とは独立した二つ目のrealistic TEI-original corpusで、責任の受け渡し（Handoff）の`from` / `to` / `condition`がlosslessに保持できることを確認する。

一つのReferenceで成立した表現を、それだけでcontractへ昇格させないためのEvidenceである。

## 上位基線

- blocker定義: [`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`](../BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md) §8-C
- 十分性監査: [`BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md`](../BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md)
- corpus候補: [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](../LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md) / [`LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`](../LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md)
- 担当割り当て: [`UPSTREAM_BREAKDOWN_2026-09.md`](../UPSTREAM_BREAKDOWN_2026-09.md) §5
- 関連RQ: RQ-002 / RQ-003 / RQ-021 / RQ-022 / RQ-025 / RQ-026

## 担当境界

| 所有する | 所有しない |
| --- | --- |
| 二つ目のcorpusの選定と受け入れ条件 | Handoff表現そのものの決定（[ISSUE-000048](000048-business-procedure-contract-adr-readiness.md)） |
| `from` / `to` / `condition`のlossless確認 | cross-view checkerの実装（[ISSUE-000042](000042-business-procedure-cross-view-reference.md)） |
| 推測で埋めないことの実証 | production採用（[ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)） |

## 守る境界

二つ目のcorpusは「数を増やすため」の材料ではない。次を守れなければEvidenceにならない。

- exact identity / actor / sequenceを**発明しない**。元資料に無いものはunknownとして残す
- physical ObservationからresponsibilityのHandoffを**生成しない**
- Step adjacency（隣接）からHandoffを**推測しない**
- explicit Handoff sourceだけが`H1` / `R3`をauthorする
- second-corpusのData authorityを**synthetic化しない**
- 出典と利用条件を記録する（public sourceを使う場合はライセンスと引用範囲を明示）

## 完了条件

- [ ] `order_entry`と独立したcorpusを1件選定し、出典・利用条件・対象範囲を記録した
- [ ] `from` / `to` / `condition`が元資料からlosslessに写せることを確認した
- [ ] 元資料に無いidentity / actor / sequenceを補完していないことを確認した
- [ ] Observationや隣接関係からHandoffを生成していないことをactual testで確認した
- [ ] Handoff sourceが無い箇所はunknownのまま保持され、fail closedになることを確認した
- [ ] ISSUE-000048のblocker Cへ、取得revisionとEvidence natureつきで結果を渡した

## 非目標

- corpusのフルCRUD化・完全モデル化
- generic Role taxonomyやBusiness Flow surfaceの導入
- 二例あることをもって一般化の証明とすること

## 関連

- [ISSUE-000048](000048-business-procedure-contract-adr-readiness.md) / [ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md)
