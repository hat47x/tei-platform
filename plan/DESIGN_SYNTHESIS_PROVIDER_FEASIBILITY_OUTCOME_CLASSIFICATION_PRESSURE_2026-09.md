# Design Synthesis Provider Feasibility Outcome Classification Pressure — 2026-09

Status: **Lane D / S2-S5 read-only refinement / no production enum adoption**

## 1. この文書の役割

本書は、`DESIGN_SYNTHESIS_PROVIDER_FEASIBILITY_SCOPE_PRESSURE_2026-09.md`で分離した:

```text
semantic candidate
  != realization / binding variant
```

を一段深め、current PostgreSQL sourceが返す「実行できない」を一種類の`provider incompatible`へ畳まないことをPressureする。

新しいProduct Requirement、Gate、PAS、production result enum、provider abstraction、feedback ontologyは導入しない。

確認する問いは一つである。

> Provider pathが進めないとき、その理由は何で、どのowner / candidate scopeへ返すべきか。

## 2. 入力となるcurrent PostgreSQL source

current `internal/database/providers/postgres/`には少なくとも次の異なる停止理由がある。

```text
A. required Evidence missing
   DropTable + incomplete partition observation
   -> partition-observation-unavailable

B. observed safety incompatibility
   DropTable target is partition child
   DropTable target has partition descendants
   -> destructive readiness blocked

C. current realization structurally unsupported
   ChangePrimaryKey
   CreateTable with PrimaryKey
   AlterColumn Default change
   unsupported / unknown ChangeKind
   -> explicit unsupported error / no partial SQL

D. operational recovery decision required
   IndexStateDrift
   -> migration SQL is not generated automatically

E. current realization supported
   complete observation + ordinary DropTable
   supported non-destructive changes
```

すべてが最終的に`error`や「SQLを生成しない」という表面へ現れ得るが、Design Synthesis上の意味は同じではない。

## 3. 中心結論

```text
same failure surface
  != same epistemic result
  != same owner
  != same contraction scope
```

S2 / S5では少なくとも次を区別して読む。

| current fact | Design Synthesisでの扱い | 主owner | semantic candidate |
| --- | --- | --- | --- |
| required Evidence missing | unresolved | provider Evidence / observation | retain |
| observed safety incompatibility | realization contradiction | provider safety / binding | retain |
| structurally unsupported | realization contradiction to current provider capability claim | provider capability / binding | retain |
| operational recovery required | unresolved operational state | runtime / operations | retain |
| supported realization | feasible realization evidence only | provider / binding | retain; not accepted |

これはproduction enumではない。actual Gate C / provider Referenceで必要な情報量を決めるためのread-only分類である。

## 4. Missing Evidenceはincompatibilityではない

例:

```text
DropTable
+ PartitionObservationComplete=false
```

current sourceは実行を止める。

しかし結論は:

```text
plain PostgreSQL DropTable is unsafe
```

ではなく:

```text
plain PostgreSQL DropTable can not yet be qualified
because required provider-owned observation is missing
```

である。

したがって:

```text
missing Evidence
  -> unresolved
  -X-> realization contradiction
  -X-> semantic contradiction
```

とする。

## 5. Observed unsafeはmatching realizationへ閉じる

complete observationで:

```text
target is partition child
or
target has partition descendants
```

が成立した場合、current plain `DROP TABLE` realization claimはcurrent authorityと両立しない。

```text
current plain PostgreSQL DropTable realization
  -> contradiction / eliminate this realization variant

logical desired-state deletion meaning
  -> retain
```

Target / Providerがaccepted hard constraintなら、そのsemantic + realizationのcomposite candidateは縮約できる。ただし理由をsemantic falseへ書き換えない。

## 6. Structurally unsupportedはprovider capabilityの現在値

current rendererが:

```text
ChangePrimaryKey
CreateTable.PrimaryKey
AlterColumn Default
unsupported ChangeKind
```

を明示errorにする場合、contradictされるclaimは:

```text
current PostgreSQL realization contract can realize this operation now
```

である。

これは:

```text
Primary Key change is invalid business meaning
Default change is invalid semantic meaning
```

を意味しない。

また、future provider implementationで状況が変わり得るため、provider capabilityのcurrent snapshotへ束縛して読む。

## 7. Operational recoveryをstructural unsupportedへしない

`IndexStateDrift`はcurrent rendererで:

```text
operational recovery decision required
migration SQL is not generated automatically
```

となる。

これは重要なnegative caseである。

```text
current operation cannot proceed automatically
  != current PostgreSQL provider can never realize the semantic intent
```

したがってDesign Synthesisでは:

```text
IndexStateDrift
  -> operational unresolved / recovery required
  -> runtime / operations ownerへroute
  -> semantic candidate retain
  -> realization variantをstructurally unsupportedとして永久除外しない
```

とする。

S5の:

```text
provider / runtime limitation
implementation defect
semantic inconsistency
```

を同じ「失敗」へ潰さない境界と接続する。

## 8. Supportedもsemantic Acceptanceではない

complete observationでordinary tableのDropTableがsafeでも:

```text
provider can realize it safely
  != desired semantic deletion is accepted
  != Business intent is correct
```

である。

Provider supportはrealization feasibilityのEvidenceであり、Shared Acceptanceを代替しない。

## 9. S2 contractionへの適用

概念上、S2が必要とするのはgeneric error countではなく:

```text
candidate / realization scope
current authority or observation
reason category
owner layer
current result
```

へ戻れることである。

ただしこれをproduction `ProviderFeasibilityResult` schemaとして先行固定しない。

### 9.1 realization elimination可能

```text
observed safety incompatibility
structurally unsupported current realization
```

は、exact matching realization claimへ閉じればdeterministic contractionに使える。

### 9.2 unresolvedとして残す

```text
missing provider Evidence
operational recovery required
```

は「今進めない」ことを示すが、現在のsemantic / realization designがfalseであるとは限らない。

### 9.3 hard provider constraint

accepted requirementが:

```text
must run on current PostgreSQL target/profile
```

まで固定している場合だけ、realization incompatibilityからcomposite candidate縮約へ広げられる。

その場合もsemantic subject自体はreview可能な形で残す。

## 10. S5 feedback routingへの適用

同じclassificationはruntime feedbackのowner routingにも使える。

```text
missing observation
  -> Evidence acquisition / provider observation

observed unsafe topology
  -> provider safety / realization redesign

unsupported current operation
  -> provider capability gap / alternate realization

IndexStateDrift
  -> operations / recovery decision

provider success
  -> realization succeeded
     != business actuality completed
```

これにより、generic provider errorをLLMへ渡してBusiness meaningを書き換える短絡を防ぐ。

## 11. test-only Pressure

`design_synthesis_provider_feasibility_outcome_classification_pressure_test.go`では次を固定する。

1. missing partition observationはunresolved / Evidence ownerとなる。
2. observed partition incompatibilityはmatching realizationだけを除外する。
3. current unsupported ChangePrimaryKeyはprovider capability contradictionだがsemantic meaningを消さない。
4. IndexStateDriftはoperational unresolvedであり、structural incompatibilityとしてrealizationを除外しない。
5. supported realizationもsemantic Acceptanceに昇格しない。
6. 同じgeneric `error` surfaceだけではcontractionを決められない。

## 12. 非主張

本Pressureから次を主張しない。

- production provider outcome enumが必要
- provider error taxonomyを固定すべき
- `IndexStateDrift` recoveryをDesign Synthesisが自動実行すべき
- unsupported operationをLLMがSQL生成で補完すべき
- provider supportならsemantic candidateをAcceptしてよい
- actual PostgreSQL package greenを得た
- Gate B / Gate Cが進んだ

## 13. current判断

今回のcurrent sourceから得られる境界は:

```text
provider failure surface
  -> reason / owner / scopeへ戻す
  -> exact realization claimだけを必要に応じ縮約
  -> missing Evidence / operational recoveryはunresolvedで保持
  -> semantic meaningへ逆流させない
```

である。

これでS2のTarget / Provider feasibilityとS5のfeedback routingの間にある一つの情報損失を、production schemaなしで閉じられる。
