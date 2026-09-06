# Business Procedure Identity Pressure — 2026-09

Status: Evidence in progress

## 目的

`Procedure`をtop-level Canonical identityへ昇格させる前に、現在のderived Business Procedure Referenceで何が追跡でき、どこからstable identityが必要になるかを具体的なReference圧力で確認する。

この文書はCanonical schema追加の提案ではない。

## 基線

対象は`order_entry`の二つのderived fixtureである。

- `reference/business-system-design/order-entry-business-procedure.json`
- `reference/business-system-design/order-entry-follow-up-procedure.json`

どちらも次を明示する。

```text
artifactRole = derived-reference-fixture
canonical = false
Procedure identityStatus = experiment-local
```

両fixtureは`order_entry.order_record`を参照する。これにより、同一Application Patternを複数のBusiness Procedure viewpointが利用する圧力を作る。

## Pressure 1: multiple Procedure -> same Pattern

次の二つのusageを同時に保持できるかを確認する。

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

`identity_pressure_test.go`は、二つのfixtureから同じPatternへのusageを2件としてtraceできることを確認する。

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

一方、同じProcedure / step / PatternであってもReference artifactのpath自体をrenameすると、`artifact + step + Pattern`で構成した非Canonical locatorは変化する。

これは実装不具合ではなく、現在のlocatorがfile provenanceに依存していることの観測である。

### 含意

長期review history、comment anchor、外部Evidence、cross-revision approval等が**file renameを越えて同じBusiness Procedure artifactを指す必要がある**なら、何らかのstable identityが必要になる可能性がある。

ただし、そのidentityを直ちにCanonical `Procedure`にする必要はない。

候補は少なくとも分離して検討する。

```text
A. Canonical business semantic identity
B. Source artifact identity
C. Evidence / review anchor identity
D. Repository path + revision provenance
```

A〜Dを同じIDで兼用すると、Business meaningとRepository運用上の履歴identityが再び結合する。

## Pressure 4: 既存Source Indexでどこまで足りるか

TEIにはすでに`internal/sourceindex`とneutralな`subject.Subject`がある。

既存Source Indexの性質は次である。

```text
Subject
  = path-independent semantic / logical target

Artifact
  = path + format + content fingerprint

Reference
  = Subject -> ArtifactPath + Range + Role
```

また、既存Workspace Source IndexはCanonical subjectとsource locatorを分離しており、file moveではsubjectを保ったままlocatorだけを変更する方針を持つ。

Business Procedure Referenceで新しいidentity機構を設計する前に、この既存モデルを圧力テストする。

### Pressure 4A: pure file move

同一bytesのBusiness Procedure fixtureを別pathへ移した場合、`sourceindex.NewArtifact`で生成されるPathは変わるがFingerprintは一致する。

これは**内容非変更のpure moveをcontent identityとして検出する**用途には既存Artifact fingerprintが使えることを示す。

### Pressure 4B: semantic-equivalent formatting change

同じJSON meaningへwhitespaceだけを追加した場合、fixture decode結果は同一でもArtifact fingerprintは変化する。

したがってArtifact fingerprintはsemantic identityではない。

```text
same semantic fixture
  + whitespace-only content change
        ↓
Source Index Artifact fingerprint changes
```

これは期待どおりであり、fingerprintをstable Business Procedure identityとして流用してはいけない。

### Pressure 4C: existing Pattern Subject

二つのBusiness Procedure artifactが同じ`order_entry.order_record`を参照する場合、既存のpath-independent `application-pattern` Subjectへ二つのderived Referenceを結び付けることはできる。

つまりPattern変更から「どのBusiness Procedure artifactがそのPatternを参照しているか」を辿るために、新しいCanonical Procedure kindは必須ではない。

ただし、Business Procedureそのもののcross-revision identityはこれでは解決していない。

## 000039との境界

`plan/issue/000039-business-change-rehearsal.md`は、将来の共通change analysisについて次を既に完了条件としている。

```text
Canonical IDを保ったfile rename / moveだけなら
Semantic差分なし
Source locatorだけ変更
```

レーンCはこの共通解析器を先取り実装しない。

本Referenceでは次だけを確認する。

- existing Source IndexのArtifact / Subject分離がBusiness Procedure圧力でも有効か
- Pattern subjectへのreverse referenceが可能か
- fingerprintで解けないsemantic continuityがどこに残るか

Validation / Bindingを含む本格的なchange graph、revision-aware impact、Scenario propagationは000039の責務に残す。

## 現時点の判断

### 維持

- `Procedure` / `Step`はCanonical Worldへ追加しない
- Business Procedure fixtureはderived Referenceのままにする
- Pattern-owned Data / Relation / Query / Capability / Datasourceはnormalized Patternから導出する
- manual / external Business Actionへfake Function / Capabilityを要求しない
- Source Index Artifact fingerprintをsemantic Procedure identityとして流用しない

### 新しく絞れた未解決点

次に問うべきなのは、

> Business Procedure meaningをCanonical化すべきか

ではなく、

> Source Indexの既存Subject / Artifact / Referenceモデルとrevision provenanceだけでは、Business Procedure自身のcross-revision review identityに何が不足するか

である。

pure file moveはArtifact fingerprintで補助でき、Pattern側のreverse impactは既存Pattern Subjectで辿れる。

残る圧力は、**内容が変わりつつ同じBusiness Procedureとしてreview historyを継続したい場合**、および**Procedure間handoff / reuseがBusiness semantic identityを本当に要求する場合**である。

## 次のPressure

1. step reorder / instruction wording changeとsemantic impactの区別
2. Pattern renameまたはaccepted source change時のreverse impact
3. manual residual explanationの再生成耐性
4. 内容変更＋file moveを越えるreview anchor
5. 二つのProcedure間でhandoffを表現したとき、stable semantic Procedure identityが本当に必要になるか

これらで既存Subject / Source Index / revision provenanceでは不足する反復Evidenceが得られた場合だけ、Canonical Procedureの最小contractをADR候補として再検討する。

## Verification

実装test:

```bash
go test ./tools/reference-business-procedure-check
```

identity pressureの主要test:

- `TestMultipleProceduresCanReferenceSamePatternWithoutCanonicalProcedureIdentity`
- `TestArtifactAndStepEvidenceLocatorSurvivesExperimentLocalProcedureRename`
- `TestArtifactRenameStillChangesNonCanonicalEvidenceLocator`
- `TestSourceIndexArtifactFingerprintSurvivesPureFileMove`
- `TestSourceIndexArtifactFingerprintIsContentIdentityNotSemanticIdentity`
- `TestCanonicalPatternSubjectRemainsPathIndependentAcrossProcedureArtifacts`

actual Repository runnerによるgreen確認まではEvidence in progressとする。
