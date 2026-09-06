# Business Procedure Identity Pressure — 2026-09

Status: Evidence in progress

## 目的

`Procedure`をtop-level Canonical identityへ昇格させる前に、現在のBusiness Procedure Reference authoring fixtureで何が追跡でき、どこからstable identityが必要になるかを具体的な圧力で確認する。

この文書はCanonical schema追加の提案ではない。

## Authoring / Derivedの前提

対象は`order_entry`の二つのReference authoring fixtureである。

- `reference/business-system-design/order-entry-business-procedure.json`
- `reference/business-system-design/order-entry-follow-up-procedure.json`

どちらも次を明示する。

```text
artifactRole = reference-authoring-fixture
canonical = false
Procedure identityStatus = experiment-local
```

fixtureにはApplication / Semantic Worldから導出できないBusiness側の意味を保持する。

```text
purpose
trigger / event candidate
actor / job context candidate
manual / external / unresolved Business Action
```

一方、Application Patternから得られるData / Relation / Query / Capability / Datasource、Business Action × Pattern Matrix、diagnosticはderived Projectionとして扱う。

## Pressure 1: multiple Procedure -> same Pattern

二つのauthoring fixtureは同じ`order_entry.order_record`を利用する。

```text
order-entry.order-handling / BP-01
  -> order_entry.order_record

order-entry.follow-up / FU-01
  -> order_entry.order_record
```

期待する性質:

- Pattern identityは同じまま再利用する
- Business Actionを一つへ畳まない
- Procedureごとに別Capabilityや別Data identityを捏造しない
- 同じPatternを使うこと自体を曖昧性エラーにしない

### 観測

このtraceabilityには現時点でCanonical Procedure identityは不要である。

Reference artifact、experiment-local Procedure ID、step ID、explicit Pattern identityを組み合わせれば、同じPatternを利用する複数のBusiness Actionを区別できる。

## Pressure 2: experiment-local Procedure rename

`order-entry.follow-up`を`order-entry.follow-up-local-rename`へ変更し、Business Action / step / Pattern referenceは変えない。

確認対象:

- Business Action -> Pattern
- Pattern-derived Data / Relation / Query / Capability / Datasource
- diagnostics
- artifact + step + Patternで構成したEvidence locator

### 観測

Procedure local IDだけのrenameでは、cross-view meaningと`artifact + step + Pattern` locatorは変化しない。

したがって、基本的な三視点cross-checkと局所impact traceだけを理由に、Procedure IDをCanonical World identityへ直ちに昇格させるEvidenceはまだない。

## Pressure 3: artifact path rename

同じProcedure / step / PatternであってもReference authoring fixtureのpath自体をrenameすると、`artifact + step + Pattern`で構成した非Canonical locatorは変化する。

これは実装不具合ではなく、locatorがfile provenanceに依存していることの観測である。

長期review history、comment anchor、外部Evidence、cross-revision approval等がfile renameを越えて同じBusiness Procedure authoring artifactを指す必要があるなら、stable identityが必要になる可能性がある。

ただし、そのidentityを直ちにCanonical `Procedure`にする必要はない。

## Pressure 4: 既存Source Indexでどこまで足りるか

TEIにはすでに`internal/sourceindex`とneutralな`subject.Subject`がある。

```text
Subject
  = path-independent semantic / logical target

Artifact
  = path + format + content fingerprint

Reference
  = Subject -> ArtifactPath + Range + Role
```

### 4A pure file move

同一bytesのBusiness Procedure authoring fixtureを別pathへ移した場合、Pathは変わるがArtifact Fingerprintは一致する。

内容非変更のpure moveをcontent identityとして検出する用途には既存Fingerprintを使える。

### 4B semantic-equivalent formatting change

同じJSON meaningへwhitespaceだけを追加した場合、fixture decode結果は同一でもArtifact Fingerprintは変化する。

したがってFingerprintはsemantic identityではない。stable Business Procedure IDとして流用しない。

### 4C existing Pattern Subject

二つのBusiness Procedure authoring fixtureが同じ`order_entry.order_record`を参照する場合、既存のpath-independent `application-pattern` Subjectへ二つのderived Referenceを結び付けられる。

つまりPattern変更から「どのBusiness Procedure artifactがそのPatternを参照しているか」を辿るために、新しいCanonical Procedure kindは必須ではない。

Business Procedure authoring artifactそのもののcross-revision identityは、これでは解決していない。

## 000039との境界

`plan/issue/000039-business-change-rehearsal.md`は、将来の共通change analysisについて次を既に完了条件としている。

```text
Canonical IDを保ったfile rename / moveだけなら
Semantic差分なし
Source locatorだけ変更
```

レーンCは000039のchange graphを先取り実装しない。

本Referenceでは次だけを確認する。

- existing Source IndexのArtifact / Subject分離がBusiness Procedure圧力でも有効か
- Pattern subjectへのreverse referenceが可能か
- fingerprintで解けないsemantic continuityがどこに残るか

Validation / Bindingを含むrevision-aware impact、Scenario propagationは000039の責務に残す。

## 現時点の判断

### 維持する境界

- `Procedure` / `Step`はCanonical Worldへ追加しない
- Business ProcedureはReference実験では`reference-authoring-fixture`として扱う
- authoring fixtureにしかないmanual / trigger / actor候補をderived扱いしない
- Pattern-owned Data / Relation / Query / Capability / Datasourceはnormalized Patternから導出する
- Matrix / diagnosticはderived Projectionとする
- manual / external Business Actionへfake Function / Capabilityを要求しない
- Source Index Artifact fingerprintをsemantic Procedure identityとして流用しない

### 残るidentity問題

pure file moveはArtifact fingerprintで補助でき、Pattern側のreverse impactは既存Pattern Subjectで辿れる。

残る圧力は、**内容が変わりつつ同じBusiness Procedure authoring artifactとしてreview historyを継続したい場合**と、**Procedure間handoff / reuseがBusiness semantic identityを本当に要求する場合**である。

候補は混同しない。

```text
A. Canonical business semantic identity
B. authoring Source identity
C. Evidence / review anchor identity
D. Repository path + revision provenance
```

A〜Dを同じIDで兼用すると、Business meaningとRepository運用上の履歴identityが再結合する。

## 次のPressure

1. instruction wording changeとsemantic impactの区別
2. Pattern accepted source change時のreverse impact
3. manual residual explanationの再生成耐性
4. 内容変更＋file moveを越えるreview anchor
5. 二つのProcedure間handoffでstable semantic Procedure identityが本当に必要になるか

既存Subject / Source Index / revision provenanceでは不足する反復Evidenceが得られた場合だけ、Canonical Procedureの最小contractをADR候補として再検討する。

## Verification

```bash
go test ./tools/reference-business-procedure-check
```

主要test:

- `TestMultipleProceduresCanReferenceSamePatternWithoutCanonicalProcedureIdentity`
- `TestArtifactAndStepEvidenceLocatorSurvivesExperimentLocalProcedureRename`
- `TestArtifactRenameStillChangesNonCanonicalEvidenceLocator`
- `TestSourceIndexArtifactFingerprintSurvivesPureFileMove`
- `TestSourceIndexArtifactFingerprintIsContentIdentityNotSemanticIdentity`
- `TestCanonicalPatternSubjectRemainsPathIndependentAcrossProcedureArtifacts`

actual Repository runnerによるgreen確認まではEvidence in progressとする。
