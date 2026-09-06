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

## 現時点の判断

### 維持

- `Procedure` / `Step`はCanonical Worldへ追加しない
- Business Procedure fixtureはderived Referenceのままにする
- Pattern-owned Data / Relation / Query / Capability / Datasourceはnormalized Patternから導出する
- manual / external Business Actionへfake Function / Capabilityを要求しない

### 新しく絞れた未解決点

次に問うべきなのは、

> Business Procedure meaningをCanonical化すべきか

ではなく、

> cross-revision review / impact / Evidenceがfile moveや再生成を越えて同じartifactを指す必要があるか。その場合、Source/Evidence identityで足りるか

である。

Source/Evidence identityで足りるなら、Business ProcedureをCanonical semantic kindへ増やさずにdurable traceabilityを得られる可能性がある。

## 次のPressure

1. 同一fixtureを内容非変更で移動した場合のreview / Evidence anchor
2. step reorder / instruction wording changeとsemantic impactの区別
3. Pattern renameまたはaccepted source change時のreverse impact
4. manual residual explanationの再生成耐性
5. 二つのProcedure間でhandoffを表現したとき、stable semantic Procedure identityが本当に必要になるか

これらでSource/Evidence identityでは不足する反復Evidenceが得られた場合だけ、Canonical Procedureの最小contractをADR候補として再検討する。

## Verification

実装test:

```bash
go test ./tools/reference-business-procedure-check
```

identity pressureの主要test:

- `TestMultipleProceduresCanReferenceSamePatternWithoutCanonicalProcedureIdentity`
- `TestArtifactAndStepEvidenceLocatorSurvivesExperimentLocalProcedureRename`
- `TestArtifactRenameStillChangesNonCanonicalEvidenceLocator`

actual Repository runnerによるgreen確認まではEvidence in progressとする。
