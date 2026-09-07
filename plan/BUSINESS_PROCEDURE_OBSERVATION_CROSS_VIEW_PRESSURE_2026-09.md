# Business Procedure Observation Cross-View Pressure — 2026-09

Status: **Lane C / Observation cross-view boundary resolved for current Reference / no new Canonical identity / actual runner Evidence pending**

## 1. 問い

Gate 15bのsocial-system realism Pressureでは、`order_entry` Business ProcedureのBP-08を次のBusiness Actionとして保持している。

```text
verify relevant external state when business continuation depends on actual state
```

Authorizationについては既存Semantic authority `security.authorize`が存在したため、BP-06からexplicit semantic refを追加できた。

同じ原則でBP-08にも既存の安定したObservation意味identityを参照できるかを確認する。

ここで判断を誤ると、runtime operationやEvidence locatorをBusiness Canonical identityへ昇格させてしまう。

## 2. 既存Driver Observationの性質

`internal/driver/observation.go`には次がある。

- `ObservationEvidenceNature`
- `ObservationReadFunc`
- `ObservationReadExecution`
- `ExecuteObservationRead`

ただしsource自身が境界を明記している。

### 2.1 Evidence natureはCanonical taxonomyではない

`ObservationEvidenceNature`はsynthetic / external-observed / managed-here等を区別するためのEvidence metadataであり、Canonicalな業務意味や固定taxonomyではない。

### 2.2 ReadIdentityはCanonicalIDではない

`ObservationReadExecution.ReadIdentity`はcallerが実際に選択したQuery / Contract Query / Binding / Adapter等のread境界を保持する。

CanonicalIDであることを要求しない。

synthetic fixtureやHost固有readをCanonicalへ誤昇格させないための意図的な設計である。

### 2.3 ExecuteObservationReadはruntime operation

`ExecuteObservationRead`はCommand Business Resultとは別に、callerが明示したreadを一回実行するruntime関数である。

これはObservation専用Canonical contractやSemantic Capabilityを定義しない。

したがって:

```text
ExecuteObservationRead
ObservationReadExecution
ReadIdentity
ObservationEvidenceNature
```

のいずれも、BP-08が参照すべきBusiness/Semantic identityではない。

## 3. ADR-0023との整合

ADR-0023は次を採用している。

```text
Command invocation
  -> Business Result / acknowledgement
  -> separate explicit read boundary
  -> read result or Execution Error
  -> Observation Evidence
```

同時に:

- Observationのためだけに新しいCanonical read正本を複製しない
- Observation専用Application Patternを導入しない
- Observation専用Semantic Capabilityを導入しない
- CommandとObservationを自動対応付けしない

としている。

複数Referenceから独立したObservation意味が必要だと確認された場合のみ、その時点でApplication PatternやSemantic Capability等への昇格を再検討する。

Lane CのBusiness Procedureだけを理由にこの判断を覆さない。

## 4. Authorizationとの非対称性は正しい

BP-06では既に:

```text
Business Action
  -> security.authorize
```

という独立したSemantic Capability authorityが存在する。

一方BP-08では:

```text
Business Action
  -> ?
```

に対応する独立Canonical Observation identityがcurrent Referenceには存在しない。

この非対称性を揃えるためにfake identityを作ってはいけない。

```text
Authorization
  existing semantic authority exists
  -> explicit ref

Observation
  runtime/evidence role exists
  independent semantic authority not proven
  -> keep residual Business wording
```

とする。

## 5. BP-08の現在の扱い

BP-08はD3で次だけを保持する。

```text
Procedure-scoped local address
Business Action wording
explicit semantic refs = none, for now
```

`realization = observation-boundary`はReference authoring / review labelであり、production Core kindではない。

したがってD3が現在主張するのは:

```text
Observation-related Business intention is preserved
```

までである。

次はまだ主張しない。

```text
BP-08 is statically linked to a Canonical Observation contract
BP-08 selects a concrete read boundary
BP-08 proves observed state
BP-08 defines retry / polling behavior
```

## 6. 禁止する近道

次を行わない。

- `ExecuteObservationRead`をCanonicalIDとして参照する
- `ObservationReadExecution`をBusiness Procedure Step kindとして使う
- `ReadIdentity`をBusiness Canonical identityとみなす
- synthetic `synthetic.order_state.read`をSemantic identityへ昇格する
- `ObservationEvidenceNature`をBusiness Step taxonomyへ転用する
- `realization = observation-boundary`をproduction enumへ昇格する
- Business Action本文から対応readを推測する
- preceding Command Capabilityからreadを自動選択する

これらはいずれもRuntime / Evidence / Authoring labelとBusiness meaningを混同する。

## 7. D3 / Gate 15bへの結論

current EvidenceではD3を次のまま維持できる。

```text
D3
  thin Business Procedure residual meaning
  + explicit refs where an existing semantic authority actually exists
  + Step body / addressability separation
```

ここで重要なのは:

```text
explicit refs are evidence-driven, not symmetry-driven
```

という点である。

BP-06にCapability refがありBP-08にないことは欠陥ではない。

BP-08のBusiness intentionがD3から失われる場合は欠陥だが、現在はBusiness Action wordingとして保持される。

## 8. 将来reopenする条件

Observation identityをBusiness Procedureからexplicitに参照する設計は、たとえば複数のrealistic Referenceで次が現れた場合にreopenする。

- 同じObservation intentionを複数Procedure / Application / Targetから再利用する必要がある
- read implementationとは独立したBusiness/Semantic meaningとして差分・影響分析したい
- designer/domain userがObservation meaningそのものをstable identityとして編集・参照する必要がある
- cross-view Conformanceが文字列Business Actionだけではambiguityを解消できない

その場合でも、まず既存Query / Contract Query / Application Pattern / Semantic Capabilityのどこがauthorityかを比較し、新しいObservation概念を最後の選択肢とする。

## 9. Evidence level

今回確認したのはRepository source / ADR / Reference構造である。

```text
Observation boundary source audit: PASS
new Canonical Observation identity necessity: NOT PROVEN
D3 residual Observation wording: retained by source-level Pressure
actual go test Evidence: Pending
Pre-ADR: Not Ready
PR: Draft維持
```

actual `gofmt / go vet / go test`は、この作業時点でも接続されたRepository実行端末がないため未取得。

## 10. 次

1. actual runnerで15b / D3 testsを実行する
2. D3の残りのexplicit refsについて、既存authorityがあるものだけを同じ基準で監査する
3. `realization` subtraction後にcurrent Referenceのcross-view diagnosisが弱くなる箇所を探す
4. realistic Referenceからstable Observation identityの必要性が出るまでは新概念を追加しない
