# 000049 Business authoring production adoption readiness

Status: Proposed

## 目的

Contract ADRで決めたBusiness Procedure contractを、**実際のsource authorityとして採用してよい状態**へ到達させる。

Contract ADR readinessより強い条件を扱う。契約が決まっただけでは、production authoringへ切り替えない。

## 上位基線

- 三段readinessの原典: [`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`](../BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md) §4
- 採用Evidence契約: [`BUSINESS_AUTHORING_PRODUCTION_ADOPTION_EVIDENCE_CONTRACT_2026-09.md`](../BUSINESS_AUTHORING_PRODUCTION_ADOPTION_EVIDENCE_CONTRACT_2026-09.md)
- 担当割り当て: [`UPSTREAM_BREAKDOWN_2026-09.md`](../UPSTREAM_BREAKDOWN_2026-09.md) §4
- 関連RQ: RQ-002 / RQ-003 / RQ-017 / RQ-019 / RQ-020 / RQ-021 / RQ-025 / RQ-026
- 着手条件: [ISSUE-000048](000048-business-procedure-contract-adr-readiness.md)のContract ADR採用後

## 担当境界

| 所有する | 所有しない |
| --- | --- |
| production authoring adapterの成立条件 | contract自体の決定（ISSUE-000048） |
| shared Acceptance / repository境界の原子性 | Studio / AIのUI・接続（[ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md) / [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)） |
| fresh processでのcandidate-only steady-state | cross-view checker実装（[ISSUE-000042](000042-business-procedure-cross-view-reference.md)） |
| 失敗時にpartialを残さないこと | Product Gate B / Gate Cのscope |

## 要求する3条件

readiness split §4の3条件をそのまま担当する。

### 1. Business固有のproduction authoring adapter

```text
workspace / Source Index discoveryの再利用
strict production decode / normalize
```

既存のworkspace discoveryを迂回する第二の探索経路を作らない。

### 2. shared Acceptance / repository境界

```text
complete stale envelope
Validation / cross-view Conformance
atomic candidate source apply
atomic authority discovery cutover
```

AI専用・Studio専用の書込み経路を作らない（ADR-0022）。

### 3. fresh process

```text
candidate-only steady-state load
legacy migration adapterがunreachable
same accepted Business semantic revision
current World Conformance
```

legacy migration adapterが到達可能なまま「移行済み」とは扱わない。

## 失敗時に残してはいけない状態

- partial source write
- partial authority switch

どちらもfail closedで拒否し、途中状態を残さないことを実証する。

## 完了条件

- [ ] production authoring adapterがworkspace / Source Index discoveryを再利用し、strict decodeで不正sourceを拒否する
- [ ] shared Acceptance境界がstale envelope・Validation・cross-view Conformance・atomic applyを満たす
- [ ] authority discoveryのcutoverが原子的で、途中失敗でlegacyとcandidateの二重authorityを作らない
- [ ] fresh processでcandidate-onlyをload でき、legacy migration adapterがunreachableである
- [ ] 同じaccepted Business semantic revisionとcurrent World Conformanceを再現できる
- [ ] partial write / partial authority switchを残さないことをactual testで確認した
- [ ] 取得したEvidenceのrevisionとnatureを記録し、synthetic成功をproduction採用の保証へ昇格させていない

## 非目標

- Studio / AI authoringのend-to-end実装
- production migration command UXの完成
- final hash / ETag / revision-token schemaの確定（必要になった時点で別判断）
- Business schemaへのfield追加

## 関連

- [ISSUE-000048](000048-business-procedure-contract-adr-readiness.md) / [ISSUE-000050](000050-second-realistic-handoff-corpus.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md)
- [`ADR-0022`](../adr/0022-ai-proposal-canonical-acceptance-boundary.md): shared Acceptance Boundary
