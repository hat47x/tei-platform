# Business Authoring Production Loader / Versioning Option Audit — 2026-09

Status: **Lane C / Pre-ADR option audit / no production loader or schema adoption**

## 1. 目的

Current migration pressureでは:

```text
legacy Procedure + O1 overlay + Reference Set Handoff
  ↓ migration preview
inline O1 Procedure + standalone Handoff
  ↓ one authority cutover
candidate-only authority
```

まで絞れている。

残るproduction blockerの一つは:

> loader / schema versioningをどう設計すれば、migrationのためにlegacy formatを読める一方、cutover後にold/new permanent dual authorityを再導入しないか。

本書ではproduction APIやschema名を採用せず、loader ownershipとversioning directionだけを比較する。

## 2. 固定済み前提

### Semantic candidate

```text
Procedure source
  D4 + S4-A + singular narrow O1

standalone Handoff source
  H1/R3
```

### Authority cutover

```text
legacy source
  -> migration input / Evidence only

candidate source
  -> Preview時はnon-authoritative

Shared Acceptance
  -> one atomic authority switch

post-cutover
  -> candidate-only authoritative load
```

### 禁止

```text
legacy + candidate
  -> heuristic merge

legacy + candidate
  -> priority / last-writer-wins

legacy fallback
  -> silently enabled forever
```

## 3. Option L0 — current Reference fixture schemaをそのままproduction化

Current `business-procedure-reference-v0` / Reference Set / pressure overlayをproduction loaderへそのまま持ち込む。

### 問題

- `canonical: false` / `reference-*` artifactRole等、Reference pressure用の意味を含む。
- O1 sidecarを残すことでcomposite Acceptance complexityを恒久化する。
- Reference SetはProcedure artifact locatorとHandoff meaningを同じassembly artifactへ持つ。
- migration targetとmigration inputを区別できない。

**判定: reject.**

Reference fixtureはEvidenceとして残し、production contractへ名称ごと昇格しない。

## 4. Option L1 — 一つのproduction loaderがlegacy / candidateを常時dual-read

Conceptual:

```text
loadBusinessModel()
  accepts legacy Procedure
  accepts O1 overlay
  accepts Reference Set Handoff
  accepts inline O1 Procedure
  accepts standalone Handoff
  merges everything
```

### 問題

- migration完了後もold/new authority ambiguityが残る。
- source discovery順やmerge priorityが意味へ影響する。
- RQ-003の暗黙複数正本を再導入する。
- stale legacy fileが後から復活してmeaningを変える可能性がある。
- loaderがmigration policyとsteady-state semanticsを同時所有する。

**判定: reject.**

## 5. Option L2 — migration adapterとsteady-state loaderを分離

Conceptual:

```text
Migration adapter
  read legacy authoritative sources
  ↓
  produce candidate source objects
  ↓
  normalize / compare / Conformance
  ↓
  Shared Acceptance
  ↓
  authority cutover

Steady-state Business loader
  read candidate Procedure sources
  read standalone Handoff sources
  ↓
  normalize one BusinessModel
```

Migration adapterはlegacy sourceをread-onlyに扱い、production steady-state loaderのfallback pathではない。

### 利点

- authority directionが明瞭。
- post-cutover loaderにlegacy merge semanticsが残らない。
- migration codeを後から削除してもaccepted candidate sourcesをreloadできる。
- migration previewとnormal runtime authoring loadを別Evidenceとして検証できる。
- RQ-003 / RQ-017と整合する。

### Risk

- cutover時にsource discovery / configurationを一度切り替える必要がある。
- migration adapterとcandidate loaderが同じnormalized meaningへ収束するEvidenceが必要。
- production repository transaction ownershipは別途必要。

**判定: current primary option.**

ただしproduction package / APIは未採用。

## 6. Option L3 — schemaVersionだけでold/new authorityを切替

例:

```text
business-procedure-v0
business-procedure-v1
```

を一loaderが読み、version番号の大小でauthorityを決める。

### 問題

Schema versionはdecode contractのversionであり、authority priorityではない。

```text
newer schemaVersion
  != accepted authority
```

である。

Migration candidateがv1だからといって、Shared Acceptance前にlegacy v0 authorityを上書きしてはいけない。

**判定: reject as authority mechanism.**

Schema versionはformat compatibilityへ限定する。

## 7. Option L4 — repository revision / manifest flagだけでlegacy modeを永続切替

Repository全体に:

```text
businessAuthoringMode: legacy | candidate
```

のようなmode flagを持つ案。

### 評価

Explicit cutover markerとしては理解しやすいが、current Evidenceでは独立fieldが必要とは限らない。

一つのrepository transactionで:

- candidate files create
- old authoritative discovery remove / disable
- candidate discovery enable

をatomicに成立できるなら、別mode正本は二重管理になる。

**判定: defer.**

Concrete repository/discovery implementationがmode markerを必要とする場合だけreopenする。

## 8. Schema versioningの原則

Production schemaを将来採る場合も:

```text
schemaVersion
  -> decoder / compatibility contract

source authority
  -> Shared Acceptance / repository discovery contract

Business semantic revision
  -> normalized meaning comparison

source freshness
  -> accepted base consistency
```

を分ける。

一つのversion numberへ全役割を畳まない。

## 9. Unknown / future field handling

Current Reference pressureは`DisallowUnknownFields`でfail closedしている。

Productionでもminimum candidateの意味を守るため、少なくとも:

- unknown semantic fieldを黙って無視してlossy normalizeしない
- unsupported newer schemaを旧decoderでbest-effort mergeしない
- migration adapterが理解できないlegacy cardinalityをcandidateへ縮退しない

を要求する。

特にcurrent O1 candidateはsingularなので:

```text
legacy multi-target O1
  -> migration error / unresolved
  -X-> first target only
```

を維持する。

## 10. Candidate loader discovery boundary

Steady-state loaderはsemantic sourceをpath文字列の意味で識別しない。

必要なのは概念上:

```text
source discovery
  -> source kind / logical identityをresolve
  -> strict decode
  -> duplicate logical source reject
  -> normalize
```

である。

ただしcurrent Evidenceだけで:

- directory naming convention
- manifest file
- registry DB
- generated source index

のどれを採るかは決めない。

File pathをBusiness identityへしないことだけ固定する。

## 11. Accept後reload Evidence

Production adoption前に必要なEvidence:

```text
accepted cutover revision R
  ↓
start fresh process / loader
  ↓
legacy migration adapterを使わない
  ↓
load candidate Procedure + standalone Handoff
  ↓
normalize BusinessModel
  ↓
expected Business semantic revision
  ↓
cross-view Conformance current World
```

ここでlegacy filesがRepositoryにhistorical Evidenceとして残っていても、steady-state authoritative discoveryから到達しないことを確認する。

Physical deletionを必須にしない。

## 12. Historical Evidenceとauthorityの分離

旧Reference filesは:

- regression fixture
- migration Evidence
- historical comparison

としてRepositoryに残り得る。

したがって:

```text
file exists
  != authoritative source
```

である。

重要なのはdeleteではなくauthority discoveryから外れること。

これによりReference Evidenceを失わずRQ-003を守れる。

## 13. Shared Acceptanceとの接続

Lane Cはgeneric transaction APIを所有しない。

Production cutoverではexisting Shared Acceptance workへ次を要求するだけとする。

1. legacy source base freshnessを確認できる。
2. candidate normalize / diagnostics / cross-view ConformanceをPreviewできる。
3. Preview後のBusiness source / World staleを検出できる。
4. candidate writeとauthority discovery switchをpartialに残さない。
5. Accept後candidate-only loaderでreloadできる。

具体APIは`ISSUE-000027 / 000046`等のshared Acceptance設計と整合して決める。

## 14. Current option ranking

```text
L2 migration adapter + candidate-only steady-state loader
  -> primary

L4 explicit mode marker
  -> defer until repository discovery requires it

L0 Reference schema direct production adoption
  -> reject

L1 permanent dual-read loader
  -> reject

L3 schemaVersion as authority priority
  -> reject
```

## 15. Pre-ADR implication

Gate C / Dのproduction migration blockerはさらに:

```text
representation choice
  -> largely narrowed

loader ownership
  -> migration adapter vs candidate steady-state separation preferred

schemaVersion authority semantics
  -> rejected

permanent dual-read
  -> rejected

remaining
  -> actual loader/discovery implementation + atomic cutover + fresh-process reload Evidence
```

へ絞られる。

## 16. Current decision

```text
production loader API
  = not adopted

production schema name/version
  = not adopted

migration adapter / steady-state loader separation
  = current primary architecture

legacy compatibility fallback after cutover
  = reject

historical legacy files must be deleted
  = no; existence != authority

Pre-ADR
  = Not Ready
```
