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

## 6. Negative searchはsecond positive caseの代替ではない

`BUSINESS_HANDOFF_TEI_ORIGINAL_SECOND_CASE_SEARCH_2026-09.md`はcurrent accepted corpusを捻じ曲げないために重要である。

しかし:

```text
qualifying existing caseがない
  !=
current representationが別caseでも十分
```

である。

Negative searchが証明するのは:

- fixtureを発明しないこと
- non-inference boundaryを守ること
- current corpusからschema expansionを導かないこと

までであり、representation generalityではない。

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

## 9. Runnerとの順序

Current Gate A actual runnerは未取得である。

既存search正本はactual green前にsecond fixtureを製造しないことを既に固定している。

したがって順序は:

```text
actual Repository green
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

Runner unavailable中にsecond caseを埋めるための架空Business requirementを追加しない。

## 10. Current decision

```text
H1/R3 current minimum candidate
  = unchanged

second realistic TEI-original Handoff
  = Contract ADR Required Evidenceのまま

reason
  = necessityではなくrepresentation generalityを反証可能にするため

blocker type
  = Evidence acquisition blocker
  != schema ambiguity
  != production implementation blocker

absence of second case
  -X-> schema expansion authority
```

Contract ADRはactual green + candidate-only fresh-process feasibility + independent second realistic Handoffが揃うまでNot Readyを維持する。
