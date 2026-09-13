# 000049 Business authoring production adoption readiness

Status: Proposed

## 目的

Contract ADRで決めたBusiness Procedure contractを、**実際のsource authorityとして採用してよい状態**へ到達させる。

Contract ADR readinessより強い条件を扱う。契約が決まっただけでは、production authoringへ切り替えない。

このIssueで問うのは「schemaが正しそうか」だけではない。新しいBusiness authoring sourceへ切り替えたときに、旧sourceと新sourceが同時にauthorityを持ったり、途中失敗で一部だけ書き換わったり、fresh processでは旧migration pathに依存していたことが判明したりしないかを確認する。

## このIssueで減らす不確実性

Contract ADRが採用された後でも、次はまだ分からない。

1. 新しいBusiness sourceを、既存Workspace / Source Indexの探索規則から一貫して発見できるか。
2. invalid / staleなcandidateをproduction sourceとして受け入れずに済むか。
3. source書込みとauthority切替を、一つの失敗可能な操作として安全に扱えるか。
4. processを新しく起動しても、legacy migration adapterへ暗黙依存せずcandidate-only sourceから同じBusiness meaningを再現できるか。
5. 途中で失敗した場合に、旧sourceと新sourceが双方authoritativeになる状態や、部分的なsource更新を残さないか。

したがって本Issueは「production loaderを実装する作業」というより、**採用済みcontractをproduction authorityへ昇格しても単一正本・stale safety・fail-closed性が崩れないことを実証する作業**である。

## 完了後に可能になる判断

このIssueの完了後には、少なくとも次を判断できる状態にする。

> Business Procedureの新しいauthoring contractを、Reference / candidate-only mechanismではなく、production source authorityとして使ってよいか。

一方、このIssueが完了してもStudio / AIのauthoring UXが完成したとは扱わない。UIやAI Proposalは後から同じshared Acceptance Boundaryへ接続する。

```text
Contract ADR adopted
        ↓
本Issue
production sourceとして安全に採用できるか
        ↓ yes
production authority cutover
        ↓
Studio / AI / CLI等は同じAcceptanceへ接続
```

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

ここでいう**production authoring adapter**は、採用済みBusiness sourceを既存Workspaceへ読み込むproduction向けの入口である。**shared Acceptance**は、AI / Studio / CLI等の経路に依存せず同じValidation・stale check・Conformance・applyを通す受理境界である。**fresh process**は、既存process内のcacheやmigration済み状態へ頼らず、新しく起動したprocessで正本だけから同じ意味を構成できることを指す。

## 要求する3条件

readiness split §4の3条件をそのまま担当する。

### 1. Business固有のproduction authoring adapter

```text
workspace / Source Index discoveryの再利用
strict production decode / normalize
```

既存のworkspace discoveryを迂回する第二の探索経路を作らない。

この条件で防ぎたいのは、Business sourceだけが専用directory scanや専用registryを持ち、他のauthoring sourceとは別の「見つけ方」が正本化することです。

### 2. shared Acceptance / repository境界

```text
complete stale envelope
Validation / cross-view Conformance
atomic candidate source apply
atomic authority discovery cutover
```

AI専用・Studio専用の書込み経路を作らない（ADR-0022）。

たとえばcandidate sourceの書込みだけ成功し、authority discoveryの切替だけ失敗すると、新旧sourceのどちらが正本か分からない状態になり得る。このためsource applyとauthority cutoverは、途中状態を外部へ残さない境界として検証する。

### 3. fresh process

```text
candidate-only steady-state load
legacy migration adapterがunreachable
same accepted Business semantic revision
current World Conformance
```

legacy migration adapterが到達可能なまま「移行済み」とは扱わない。

同じprocess内で一度migrationした後に動くだけでは、production sourceだけで自立しているEvidenceにならない。新しいprocessからcandidate-only sourceを読み、同じaccepted Business meaningを再構成できることを確認する。

## 失敗時に残してはいけない状態

- partial source write
- partial authority switch

どちらもfail closedで拒否し、途中状態を残さないことを実証する。

失敗した操作が「何も変えなかった」のか、「一部だけ変えた」のかを後から推測しなければならない状態は、production authority cutoverとして不十分とする。

## 完了条件

- [ ] production authoring adapterがworkspace / Source Index discoveryを再利用し、strict decodeで不正sourceを拒否する
- [ ] shared Acceptance境界がstale envelope・Validation・cross-view Conformance・atomic applyを満たす
- [ ] authority discoveryのcutoverが原子的で、途中失敗でlegacyとcandidateの二重authorityを作らない
- [ ] fresh processでcandidate-onlyをload でき、legacy migration adapterがunreachableである
- [ ] 同じaccepted Business semantic revisionとcurrent World Conformanceを再現できる
- [ ] partial write / partial authority switchを残さないことをactual testで確認した
- [ ] 取得したEvidenceのrevisionとnatureを記録し、synthetic成功をproduction採用の保証へ昇格させていない

完了条件は「adapterのコードが存在する」ことではなく、**production authorityへ切り替えても単一正本と再現性が維持されるとEvidenceから判断できること**を意味する。

## 非目標

- Studio / AI authoringのend-to-end実装
- production migration command UXの完成
- final hash / ETag / revision-token schemaの確定（必要になった時点で別判断）
- Business schemaへのfield追加

## 関連

- [ISSUE-000048](000048-business-procedure-contract-adr-readiness.md) / [ISSUE-000050](000050-second-realistic-handoff-corpus.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md)
- [`ADR-0022`](../adr/0022-ai-proposal-canonical-acceptance-boundary.md): shared Acceptance Boundary
