# Business Handoff Second Case ADR Evidence Sufficiency Audit — 2026-09

Status: **Lane C / Contract ADR evidence audit / second realistic case remains required / no semantic candidate change**

## 1. 問い

Current Handoff candidateは:

```text
H1/R3
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

まで絞れている。

Current accepted TEI-original explicit Handoffは:

```text
order-entry.order-handling/BP-09
  -> order-entry.follow-up/FU-00
  condition: follow-up required after earlier order interaction
```

の一件である。

既存corpusのnegative search、Requirement-level corroboration、Watanabe external evidenceもある。

ここで問うのは:

> second realistic TEI-original HandoffをContract ADR前のRequired Evidenceとして残す合理性がまだあるか。

である。

## 2. Evidenceの役割を分ける

Current Evidenceは一種類ではない。

### E1 — necessity Evidence

Current explicit Handoffについて、Procedure bodyだけでは:

- relation endpoint
- continuation condition

のchangeをnormalized Business semantic diffで保持できない。

これは:

```text
Handoff meaningをnormalized Business modelに保持する必要がある
```

というnecessityを強く支える。

### E2 — subtraction / non-inference Evidence

次からHandoffを生成しないPressureがある。

- Step adjacency
- Application Function sequence
- physical Observation
- Lifecycle transition
- Deployment/runtime state

これはauthority boundaryを強くする。

PR #596 / `DESIGN_SYNTHESIS_MANUFACTURING_ACTUALITY_HANDOFF_PROOF_REFERENCE_2026-09.md`では、製造・在庫second-corpus pressureへP5 / P6を適用し、この境界をさらにnegative proofとして確認した。

```text
Command accepted -> physical completion
  = contradiction as derivation rule

Observation execution success -> physical completion
  = contradiction as derivation rule

explicit external Observation Result=true
  -> observed physical fact only
  = supported at that limited scope

Step adjacency / actor change / Function sequence / positive Observation
  -> Handoff
  = contradiction as derivation rule
```

ここで重要なのは、導出規則を`contradiction`として除外できても、domain側のHandoff不存在まで証明したわけではないことである。

```text
invalid inference rejected
  != Handoff proven absent
  != Handoff proven present
```

したがってE2は強化されたが、E4を代替しない。

### E3 — external / requirement corroboration

RQ-025 / RQ-026およびWatanabe external evidenceは、Business Procedure / Business responsibility / Business Flow separationの妥当性を補強する。

ただし具体的な第二Handoff instanceをauthorしない。

### E4 — representation generality Evidence

別の独立Business requirementから生じたHandoffでも:

```text
from { procedure, step }
to   { procedure, step }
condition
```

だけでaccepted meaningをlosslessに保持できるか。

Current missing Evidenceは主にこれである。

R2 negative proof後も:

```text
accepted independent Business seed
  = none

accepted second Handoff
  = none

second Handoff from / to / condition
  = unresolved
```

なのでE4は未取得のままである。

## 3. Option S0 — second caseを不要とし今ADRへ進む

利点:

- actual runner green後すぐContract ADRへ進める。
- current one-case necessity Evidenceは強い。
- schemaは最小でreopen可能にできる。

しかし問題は、current caseが一つのspecific shapeへ偏っていることである。

一件だけでは、別のBusiness responsibility transferで:

- actor/job distinctionがendpoint meaningへ必要になるか
- triggerとconditionだけでは区別できないevent identityが必要になるか
- document/work instruction等のidentityがrelation meaningの一部になるか
- thin Handoffではなくbroader Business Flowが必要になるか

を反証できない。

これらのfieldを今追加すべきという意味ではない。逆に、**追加しなくてよいことを一段強く確認するpositive Evidenceがまだない**。

判定: **reject as current Contract ADR readiness rule**。

## 4. Option S1 — second realistic TEI-original caseをContract ADR前に要求

Current primary。

要求するのは「二件あれば一般化できる」という数量ルールではない。

Second caseは次を満たす必要がある。

1. Handoff検証とは独立したBusiness need / requirementが先に存在する。
2. current order-entry follow-upとは異なるresponsibility boundaryを持つ。
3. exact source / target Procedure + Stepとcontinuation conditionを説明できる。
4. adjacency / state transition / runtime flowから推測していない。
5. `from/to/condition`でlossが出れば、そのlossだけをschema reopenする。
6. lossが出なければevent/document/actor-role等を追加しない根拠を強める。

このcaseの目的はcoverage件数を増やすことではなく、**current thin representationを反証可能にすること**である。

判定: **primary**。

## 5. Option S2 — Contract ADR後、Production adoption前へ送る

Production adoptionでは実loader / source authority / migration cutover等のEvidenceが必要なので、second caseをそこで確認することも技術的には可能である。

しかしContract ADRはproduction Go type名を決めるためだけでなく、semantic / authority contractを固定するDecisionである。

Second caseなしでH1/R3をADR Acceptedへすると、representation insufficiencyが見つかった場合に、ADR直後からsemantic contract revisionが必要になる。

Current Lane Cはまだproduction adoptionを急いでいないため、このリスクを取る合理性は弱い。

判定: **defer / current primaryではない**。

## 6. Negative search / negative proofはsecond positive caseの代替ではない

`BUSINESS_HANDOFF_TEI_ORIGINAL_SECOND_CASE_SEARCH_2026-09.md`はcurrent accepted corpusを捻じ曲げないために重要である。

さらにR2 negative proofにより、製造・在庫pressureについても:

```text
physical fact supported
+
Handoff unresolved
```

を同時に保持でき、adjacency / actor差 / Function sequence / ObservationからHandoffを補完する候補を除外できることが確認できた。

しかし:

```text
qualifying existing caseがない
  !=
current representationが別caseでも十分

invalid inferenceを除外できる
  !=
current H1/R3がindependent positive caseでlossless
```

である。

Negative search / negative proofが証明するのは:

- fixtureを発明しないこと
- non-inference boundaryを守ること
- unsupported actualityを発明しないこと
- physical factとBusiness continuation / Handoff authorityを分離すること
- current corpusからschema expansionを導かないこと

までであり、representation generalityではない。

したがってR2をBlocker B positive Evidenceへ読み替えない。

## 7. External evidenceもsecond positive caseの代替ではない

Watanabe external evidenceは:

- Business / Function / Data view separation
- Business responsibility / procedure meaning
- Business Flowのbroader scope
- adjacencyからresponsibility transferを推測しない方向

をcorroborateする。

一方、TEI-original exact Procedure / Step identityやaccepted continuation conditionをauthorしない。

したがってexternal patternをfixtureへコピーしてE4を満たしたことにしない。

## 8. Blockerの性格

Second realistic Handoff不足は:

```text
model-shape ambiguity
```

ではない。

Current strongest shapeは依然:

```text
H1/R3 from/to/condition
```

であり、追加fieldのconcrete loss Evidenceはない。

R2 negative proofも、現在のthin authority boundaryに新しいfieldが必要だというlossを示していない。

不足しているのは:

```text
independent positive Evidence acquisition
```

である。

したがって:

```text
second case missing
  -X-> eventIdを追加
  -X-> document fieldを追加
  -X-> actor roleをendpointへ追加
  -X-> full Business Flowへ拡張

second case missing
  -> Contract ADR Not Readyのまま保持
  -> independent Business requirementが自然に生じるまで待つ
```

とする。

## 9. Gate Aとの順序

Gate A actual execution Evidenceは、当初は未取得だった（2026-09-20更新: Gate A actual executionは取得済みでPASS。R = `e2856e6e568b6be966c6b8689ebea479ec184a4a`、判定の正本は[Pre-ADR Audit](BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§13）。Runner availability自体は一時状態なので、このauditではavailabilityを固定しない。

既存search正本とISSUE-000050は、actual green前にsecond fixture / accepted Handoffを製造しないことを固定している。

ISSUE-000050では製造・在庫連携のsecond-corpus research targetとprovenance boundaryまでは選定済みだが、accepted independent Business seedや`from/to/condition`は未成立である。

PR #596のR2は、この順序を崩さずに実施できるnegative proofである。Gate A green前に許されるのは、accepted Handoffを作ることではなく、既存authorityから不正な導出を除外し、candidate / unresolved境界を明確にすることまでである。

したがって順序は:

```text
actual Repository Gate A green
  ↓
source-level Pressureをactual Evidenceへ昇格
  ↓
independent Business requirement / Reference seed
  ↓
second realistic Handoff
  ↓
H1/R3 sufficiency test
  ↓
Contract ADR readiness再判定
```

でよい。

Gate A未解決中にsecond caseを埋めるための架空Business requirementを追加しない。Corpus / provenance researchとnegative proofは続けられるが、accepted Business seed / Procedure / Handoff authoringへは進めない。

## 10. Current decision

```text
H1/R3 current minimum candidate
  = unchanged

second realistic TEI-original Handoff
  = Contract ADR Required Evidenceのまま

second-corpus research target
  = selected
  != accepted positive Handoff Evidence

R2 P5/P6 negative proof
  = E2 non-inference / actuality boundaryを強化
  != E4 representation generality
  != Blocker B positive Evidence

reason
  = necessityではなくrepresentation generalityを反証可能にするため

blocker type
  = Evidence acquisition blocker
  != schema ambiguity
  != production implementation blocker

absence of second case
  -X-> schema expansion authority
```

Material blockerの数え方はsole Pre-ADR Auditに従う。

```text
Blocker A
  = actual Gate A green
  = source-uniqueness / candidate-only fresh-process actualを含む

Blocker B
  = independent second realistic TEI-original Handoff
```

Contract ADRはBlocker A / Bの両方が充足するまでNot Readyを維持する（Blocker Aは2026-09-20に充足。残るのはBlocker B）。Fresh-process feasibilityはADR前Required Evidenceのままだが、current material blockerではGate A actual executionへ包含し第三blockerとして二重計上しない。
