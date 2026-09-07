# Business Model Shared Acceptance Bundle Pressure — 2026-09

Status: **Lane C / test-only Acceptance pressure / no production revision API adoption**

## 1. 問い

Current authoring candidatesは一つのfileに閉じない。

```text
Procedure source A
Procedure source B
...
standalone Handoff source
```

Procedure source内にはD4 + S4-A + singular narrow O1があり、Handoffはcross-Procedure relationとして別sourceへ置く方が自然である。

このときShared Acceptanceをfile単位へ分解すると:

- Procedure wording change
- protected-target-only change
- Handoff condition change

が別々のapproval / revisionへ分裂し、cross-view Business meaningの一貫したreview単位を失う可能性がある。

本Pressureでは、複数source kindを一つの**normalized Business semantic revision**へ束ねられるかを確認する。

## 2. Test-only bundle

`business_model_shared_acceptance_bundle_pressure_test.go`では概念的に:

```text
BusinessAuthoringBundle
  Procedures[]
    inline Procedure source
      D4 + S4-A + singular narrow O1

  Handoffs
    standalone Handoff source
```

を入力とする。

Bundleはsource path / artifact locator / runtime stateをsemantic fieldsとして持たない。

## 3. Normalization target

複数sourceは一つのnormalized modelへ落とす。

```text
Procedure source A ─┐
Procedure source B ─┼─> normalize -> BusinessModel
...                ─┤
Handoff source     ─┘

BusinessModel
  Procedures[]
  Handoffs[]
```

Current test-only bundleから得るmodelは、既存Reference Set + sidecar pressureから得ていたcurrent normalized Business modelと同じmeaningになることをPressureする。

つまりsource decompositionを変えてもBusiness semantic meaningを変えない。

## 4. One semantic revision across source kinds

次のaccepted Business source changeはいずれも、同じBusiness semantic revisionの差として観測する。

### Procedure wording

```text
BP-04 businessAction change
  -> Business semantic revision changes
```

### Protected target

```text
BP-06 protectedApplicationPattern change
  -> Business semantic revision changes
```

### Handoff

```text
Handoff condition change
  -> Business semantic revision changes
```

これらを:

```text
Procedure revision
O1 revision
Handoff revision
```

という独立した意味正本へ分割しない。

Source fileは分かれても、accepted Business meaningは一つのnormalized review unitとして扱える。

## 5. Source ordering / locatorはmeaningではない

Procedure source配列の記述順だけを変更してもnormalized Business semantic revisionを変えない。

同様に既存Pressureではpure Procedure artifact locator moveでnormalized meaningが変化しない。

したがってproduction Acceptanceでも:

```text
source enumeration order
file path
manifest locator
```

をBusiness semantic identityへしてはいけない。

## 6. Relation to O1 sidecar pressure

O1 sidecar単独では、Procedure-only fingerprintがtarget-only changeを見逃す問題があった。

そのためsidecarをproduction採用するならcomposite source fingerprintが必要だった。

Current candidateではO1をProcedure sourceへinlineする方向へ進んだため、O1についてはこの複雑さを減らせる。

一方Handoffはcross-Procedure relationなのでstandalone sourceが自然であり、複数source自体は残る。

Shared Acceptance bundle pressureは、この残る複数sourceをsemantic unitとして統合する。

## 7. Relation to separate Business plane

Current strongest placement candidate:

```text
Business authoring sources
  -> normalize
  -> Business semantic revision

TEI workspace sources
  -> ir.World
  -> runtime/application revision

Business model
  <-> cross-view Conformance <-> ir.World
```

Business bundleのどのsourceが変わってもBusiness semantic revisionは変わり得るが、それだけで`ir.World` revisionを変更する必要はない。

これは:

```text
first-class Business meaning
  != same runtime/compiler revision
```

というplacement pressureを補強する。

## 8. Production APIとしてまだ決めないもの

本Pressureは次を採用しない。

- `BusinessRevision` production type
- semantic hash algorithm
- ETag format
- multi-file transaction API
- Git commit = Business revisionという固定規則
- `ir.World` revisionとのcoupled version
- independent O1 / Handoff approval stores
- production `internal/business` package

Testで使うsemantic fingerprintは比較語彙でありproduction hash contractではない。

## 9. Production Shared Acceptanceが満たすべき性質

将来APIを設計する場合、少なくとも:

1. 複数Procedure source + Handoff sourceを一つのnormalized Business modelへloadできる。
2. same Procedure identityのduplicate sourceをfail closedする。
3. stale Handoff endpointをfail closedする。
4. protected Application targetを`ir.World`へexact resolutionする。
5. source order / pathだけのchangeをsemantic no-opとして扱える。
6. Procedure wording / O1 target / Handoff conditionのchangeをsemantic changeとして扱う。
7. Accept直前にsource freshnessを再確認できる。
8. Accept後に全sourceをreloadし同じnormalized Business modelを再現できる。
9. cross-view Conformance失敗時に部分acceptしない。
10. Projection / Matrix / diagnosticをauthoritative sourceへ逆書きしない。

## 10. Atomicity pressure

複数sourceを一つのsemantic revisionとして扱う場合、production採用時にはatomicityを検討する必要がある。

例:

```text
Procedure Step rename
  + Handoff endpoint rename
```

は協調changeである。

一方だけacceptするとstale endpointを作る。

したがってShared Acceptanceは少なくとも:

```text
candidate change-set
  -> normalize all affected Business sources
  -> validate Business referential integrity
  -> cross-view validate against current ir.World
  -> accept/reject as one semantic change-set
```

というPressureを持つ。

ただしGit transaction、filesystem locking、repository commit方式等の物理実装は未決とする。

## 11. Current result

Current test-only Evidenceは:

```text
multi-source authoring
  -> one normalized Business semantic meaning
```

が成立し得ることを支持する。

したがってPre-ADR Gate Eの問いは:

```text
複数sourceを一つのBusiness semantic revisionへ束ねられるか？
```

から:

```text
productionでそのnormalization / freshness / atomic acceptanceを
どのAPI・revision mechanismとして実装するか？
```

へ一段進んだ。

## 12. Current decision

```text
Conceptual multi-source semantic bundle
  -> test-onlyで有力

Raw file layout as revision authority
  -> reject

One normalized Business semantic review unit
  -> current strongest Acceptance candidate

Production revision/hash/API
  -> not adopted

Pre-ADR
  -> Not Ready
```

Actual Repository runner greenとproduction adoption/migration設計が得られるまではADRへ進めない。
