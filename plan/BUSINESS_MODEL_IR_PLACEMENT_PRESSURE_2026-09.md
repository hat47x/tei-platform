# Business Model IR Placement Pressure — 2026-09

Status: **Lane C / architecture placement pressure / no production package or ir.World change**

## 1. 問い

Business Procedure / Handoffをfirst-classな設計意味として扱う場合、必ずcurrent `ir.World`へ:

```text
Business
Procedures
Handoffs
```

等を追加する必要があるかを確認する。

Current TEIの`ir.World`はApplication / Database / Query / Realization / Environment等のnormalized compilation unitである。

Business Procedureは一方でX-TEA / 渡辺幸三氏の三要素におけるBusiness Model側の社会システム意味を保持する。

この二つを「first-classだから同じstructに入れる」と短絡しない。

## 2. Current mechanical result

`business_model_ir_placement_pressure_test.go`ではcurrent `ir.World`にBusiness fieldを追加せず:

```text
normalized Business model
  +
ir.World
  -> cross-view validation
```

が成立することをPressureする。

Current Business model candidateは:

```text
D4
+ singular narrow O1
+ S4-A
+ H1/R3
```

である。

## 3. Separate semantic planes

Current architecture candidate:

```text
Application / Data / Database / Realization plane
  -> ir.World

Business / social-system plane
  -> normalized Business model

Cross-view layer
  -> exact identity resolution
  -> consistency diagnostics
  -> derived Function/Data/CRUD/ownership lenses
```

これはBusiness Modelを二級のcomment artifactにする意味ではない。

Business model側にも:

- stable Procedure identity
- Business Action meaning
- actor / trigger uncertainty
- exact Application / Validation / Capability refs
- singular protected Application target
- local Step address
- Handoff relation

をlosslessに保持する。

違いはruntime / compiler IRと同一containerにするかどうかだけである。

## 4. Why not expand ir.World yet

### 4.1 Current Driver does not execute Business Procedure as workflow

Business Procedureは:

```text
manual
external
human judgment
physical observation boundary
application-use
security boundary
```

を同じ社会システム文脈に置く。

これはRuntime Workflow DSLではない。

したがってDriver executionのために`ir.World`へ入れる必要性は未証明である。

### 4.2 Business-only changes should not pretend runtime meaning changed

Current Pressureでは少なくとも次を独立したBusiness semantic changeとして観測できる。

```text
BP-04 Business wording change
BP-06 protected Application target change
Handoff condition change
```

いずれも:

```text
Business semantic revision changes
ir.World stays unchanged
```

でよい。

`business_model_ir_placement_pressure_test.go`では、これらのchange後もcurrent `ir.World`をmutationせずcross-view validationできることをPressureする。

Business design reviewとruntime/application compilation revisionを不必要に同一revisionへ丸めない。

### 4.3 Cross-view validation still fails closed

Separate planeでもprotected target `order_entry.submit_order`がApplication Worldから消えればBusiness model validationはfail closedできる。

つまりseparateであることはcross-view integrityを弱めない。

### 4.4 Handoff integrity does not require runtime IR ownership

Handoff endpoint / conditionはBusiness plane内でreferential validationできる。

```text
Procedure-scoped Step local address
  + explicit Handoff relation
  -> normalized Business integrity
```

をSource Indexやruntime workflow stateへ依存せず検査できる。

したがってHandoff meaningのfirst-class性だけを理由に`ir.World`へ埋め込む必要も未証明である。

## 5. Relation to X-TEA three-element design

X-TEAの重要点はData / Function / Businessを別々のviewpointとして持ち、相互批判できることにある。

```text
separate viewpoints
  != disconnected artifacts
```

である。

Current TEIへの翻訳も:

```text
Business plane
  <-> Application/Function projection
  <-> Data/Relation projection
```

とし、Business modelをApplication/Data modelへ吸収しない方が三要素の緊張関係を保ちやすい。

## 6. Why not create `internal/business` production package yet

Separate planeが有力でも、今すぐ:

```text
internal/business
ir.BusinessWorld
workspace.Business
```

をproduction追加しない。

理由:

- production Canonical contract自体がPre-ADR Not Ready
- actual runner green未取得
- O1 production source syntax未確定
- Handoff production source / placement未確定
- Shared Acceptance / semantic revision API未確定
- Business Flow / DFDとの境界を広げるべきでない

Current test-only typesでplacement pressureだけを確認する。

## 7. Reopen conditions for embedding in ir.World

次のいずれかが具体的に必要になった場合、`ir.World` integrationを再検討する。

1. compiler / DriverがBusiness Procedure meaningを直接必要とする。
2. release / compatibility / deployment判定がBusiness modelを同一compiled unitとして必要とする。
3. one atomic accepted revisionでApplication/Data/Businessを必ずcommitしなければ整合性を保てない実例が出る。
4. separate Business modelではcross-view Conformanceをlosslessに実装できない。
5. Target projection / Editor reloadでseparate planeが二重SOTを生む。
6. Business model changeとruntime/application revisionを分離するとactual operational inconsistencyが発生する。

Current Referenceではどれも未証明である。

## 8. What should remain shared

Containerを分けてもidentity vocabularyは共有する。

```text
Application Pattern canonical identity
Semantic Validation identity
Semantic Capability identity
Procedure identity where Business-owned
```

Business planeが独自にApplication/Data identityを複製・renameしない。

Cross-view linkはexact existing identityを参照する。

Authorization policy、Database ownership、Data/Relation trace、CRUD matrix等のderived/corroborating情報をBusiness planeの第二SOTにしない。

## 9. Acceptance implication

Business modelはseparate planeでも独自のaccepted semantic revisionを持ち得る。

Conceptual:

```text
Business authoring sources
  -> normalized Business model
  -> Business semantic revision

TEI workspace sources
  -> ir.World
  -> Application/Data runtime revision

both
  -> cross-view Conformance
```

Current test-only semantic fingerprintでは:

```text
Business wording change             -> changes
protected target change             -> changes
Handoff condition change            -> changes
pure Procedure artifact move        -> unchanged
source Procedure ordering only      -> unchanged
```

をPressureしている。

このfingerprintはproduction hash/revision APIではないが、**semantic revisionの対象をraw file layoutではなくnormalized Business meaningに置く**方向を支持する。

## 10. Current decision

```text
Business Model first-class meaning
  -> yes, as current design pressure

Business Model inside ir.World
  -> not required by current Evidence

Separate normalized Business plane
  -> strongest current placement candidate

Cross-view validation against ir.World
  -> required

Business semantic revision separate from runtime/application revision
  -> strong current pressure

production internal/business package
  -> deferred

runtime workflow execution
  -> out of scope
```

Pre-ADRでは:

> 「BusinessProcedureを採用するか」
> 「Business meaningをどのauthoring syntaxで記述するか」
> 「normalized Business planeをどのproduction packageへ置くか」
> 「Business meaningをir.Worldへ埋め込むか」

を別Decisionとして扱う。

Current Evidenceでは最後のDecisionを急ぐ理由がない。
