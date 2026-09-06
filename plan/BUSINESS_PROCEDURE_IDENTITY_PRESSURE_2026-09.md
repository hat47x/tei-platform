# Business Procedure Identity Pressure — 2026-09

Status: Evidence in progress

## 目的

`Procedure`をtop-level Canonical identityへ昇格させる前に、Business Procedure Referenceでどのidentityが本当に必要かを段階的に確認する。

この文書はCanonical schema追加の提案ではない。

review / approval continuityとProcedure間handoffの追加圧力は、`BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`へ分離して記録する。

## Authoring / Derivedの前提

対象は`order_entry`の二つのReference authoring fixtureである。

- `reference/business-system-design/order-entry-business-procedure.json`
- `reference/business-system-design/order-entry-follow-up-procedure.json`

現在は次を明示する。

```text
artifactRole = reference-authoring-fixture
canonical = false
Procedure identityStatus = reference-local-stable
```

`reference-local-stable`はReference corpus内のauthoring keyであり、Canonical World identityではない。

fixtureにはApplication / Semantic Worldから導出できないBusiness側の意味を保持する。

```text
purpose
trigger / event candidate
actor / job context candidate
manual / external / unresolved Business Action
judgment / continuation
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

基本的なmultiple Procedure -> same Pattern traceabilityにはCanonical Procedure identityは不要である。

Reference-local Procedure key、step key、explicit Pattern identityで区別できる。

## Pressure 2: Procedure key renameとcross-view meaning

Procedure keyだけを変更し、Business Action / step / Pattern referenceを変えない場合、Pattern-derived cross-view meaningは変化しない。

これはProcedure keyがApplication Pattern meaningのauthorityではないことを示す。

一方、review / handoffでstable authoring keyとして使う段階では、key renameはlogical authoring identity changeとなる。

つまり次を分ける。

```text
Pattern-derived meaning
  Procedure key renameだけでは不変

review / handoff target
  Procedure key renameで変化
```

## Pressure 3: artifact path rename

同じProcedure / step / PatternでもReference authoring fixtureのpathを変えると、pathを含むlocatorは変化する。

file pathをBusiness semantic identityとして使わない。

## Pressure 4: 既存Source Indexでどこまで足りるか

TEIには`internal/sourceindex`とneutralな`subject.Subject`がある。

```text
Subject
  = path-independent semantic / logical target

Artifact
  = path + format + content fingerprint

Reference
  = Subject -> ArtifactPath + Range + Role
```

### 4A pure file move

同一bytesを別pathへ移した場合、Pathは変わるがArtifact Fingerprintは一致する。

pure moveのcontent continuityには既存Fingerprintを利用できる。

### 4B formatting-only change

同じJSON meaningでもbytesを変えればFingerprintは変化する。

したがってFingerprintはsemantic identityではない。

### 4C existing Pattern Subject

複数Business Procedureが同じApplication Patternを参照しても、既存path-independent Pattern Subjectへ複数Referenceを結び付けられる。

Pattern変更からBusiness Procedure利用箇所を辿るためだけなら、新しいCanonical Procedure kindは不要である。

## Pressure 5: Business authoring residualはPatternから再生成できるか

primary fixtureにはPatternを持たないBP-04 / BP-09がある。

```text
BP-04
  manual-unresolved

BP-09
  manual-external-unresolved
```

これらはauthoring fixtureには存在するがPattern-derived meaning rowsには存在しない。

### 観測

Pattern-owned meaningだけからBusiness Procedureを再構成するとmanual / external / unresolved Business Actionを失う。

したがってBusiness Procedure authoring sourceはPattern-derived cacheではない。

purpose / trigger / actor candidateも同じくBusiness側のauthoring meaningである。

### Business-only wording change

manual Business Action文言だけを変更した場合:

```text
Business-side Projection
  changed

Pattern-derived Application meaning
  unchanged

cross-view diagnostic
  unchanged unless a real cross-view conflict changes
```

Business視点をFunction/Data視点の単なる表示へ還元しない。

## Pressure 6: review / handoffによるstable authoring keyの必要性

前段のPressure 1〜5では、Canonical Procedure identityは不要というEvidenceが中心だった。

review history continuityとProcedure間handoffを加えると、安定したauthoring-level logical keyには具体的な価値が出る。

そのためReferenceでは`procedure.id`を次として扱う。

```text
identityStatus = reference-local-stable
```

ただしこれはCanonical `Procedure`への昇格ではない。

詳細:

- `plan/BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`
- `reference/business-system-design/order-entry-procedure-reference-set.json`
- `tools/reference-business-procedure-check/review_handoff_pressure_test.go`

## 000039との境界

`plan/issue/000039-business-change-rehearsal.md`は将来のrevision-aware semantic diff / dependency propagationを所有する。

レーンCはそのgraphを先取り実装しない。

今回固定するidentity区別は000039の入力として利用できる。

```text
same Procedure key + changed revision/fingerprint
  = same review target, changed source snapshot

changed Procedure key
  = logical authoring identity change
```

## 現時点の判断

### 維持する境界

- Business Procedure fixtureはReference authoring input
- Matrix / diagnosticはderived Projection
- manual / external Business Actionをfake Functionへ変換しない
- Pattern-owned meaningをBusiness Procedureへ重複記述しない
- Source Index Artifact fingerprintをsemantic Procedure identityとして流用しない
- `procedure.id`はReference-local stable authoring keyとして使う
- `Procedure` / `Step`はCanonical Worldへ追加しない

### identityを混同しない

```text
A. Canonical business semantic identity
B. Reference-local authoring identity
C. Source / Evidence locator
D. Repository revision provenance
```

現在Bの必要性にはEvidenceが出たが、Aの必要性はまだ実証されていない。

## 次のPressure

1. Business Procedureをfile-native accepted authoring sourceへ進めた場合、Bだけでreload / Source Index / Shared Acceptanceが成立するか
2. multiple Subject Area / Subsystemを跨ぐhandoffでもBで十分か
3. external contract / 長期reuseでAが必要になる具体例が出るか

不足が反復して実証されるまではCanonical Procedureの最小contractをADR候補へ上げない。

## Verification

```bash
go test ./tools/reference-business-procedure-check
```

主なtest:

- `TestMultipleProceduresCanReferenceSamePatternWithoutCanonicalProcedureIdentity`
- `TestArtifactAndStepEvidenceLocatorSurvivesExperimentLocalProcedureRename`
- `TestArtifactRenameStillChangesNonCanonicalEvidenceLocator`
- `TestSourceIndexArtifactFingerprintSurvivesPureFileMove`
- `TestSourceIndexArtifactFingerprintIsContentIdentityNotSemanticIdentity`
- `TestCanonicalPatternSubjectRemainsPathIndependentAcrossProcedureArtifacts`
- `TestPatternDerivedProjectionCannotReplaceManualBusinessAuthoring`
- `TestBusinessWordingChangeDoesNotPretendApplicationMeaningChanged`
- `TestPurposeAndActorCandidatesRemainAuthoringMeaning`
- review / handoff pressure tests

actual Repository runnerによるgreen確認まではEvidence in progressとする。
