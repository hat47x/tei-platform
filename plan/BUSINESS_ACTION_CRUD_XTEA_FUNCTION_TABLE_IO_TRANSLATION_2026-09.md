# Business Action × CRUD — X-TEA Function/Table I/O Translation — 2026-09

Status: **Lane C / X-TEA fallback translation / no production contract adoption**

## 1. 目的

Current TEIではBusiness Action × CRUDのうちRの一部はexact structured authorityからderiveできるが、C/U/DはBusiness Actionへ安全に帰属できない。

本書は、この不足をX-TEA / XeadModelerのFunction/Table I/O構造へ立ち返って確認し、TEIで将来operation attribution contractが必要になった場合にどの層を第一候補とすべきかを絞る。

Historical X-TEA shapeをそのままTEIへ移植しない。

## 2. X-TEA source evidence

XeadModeler sourceでは`TaskFunctionIO`がFunctionで利用するI/O Tableを表し、少なくとも次の属性を保持する。

```text
TableID
OpC
OpR
OpU
OpD
```

`DialogImportXEAD.java`では`TaskFunctionIO` import時に`TableID`と`OpC / OpR / OpU / OpD`を明示的にcopyしている。

`DialogMatrixList.java`では`TaskFunctionIO`を走査し、Function ID + TableIDをkeyとして`OpC / OpR / OpU / OpD`の`+`をCRUD markへ集約する。

したがってCRUD matrixは:

```text
free-form Business Procedure text
```

から推測するのではなく、

```text
Function <-> Table I/O structured relation
  + explicit operation marks
```

から機械導出される。

これは渡辺幸三氏の「matrixをstructured design informationから機械的に得る」という設計姿勢とも一致する。

## 3. TEI current equivalent / gap

TEIはX-TEAのFunction / Table I/Oを一つのobjectへ集めず、意味を複数authorityへ分離している。

```text
Business Action
Application Pattern
Semantic Data / Relation
Semantic Capability
Query
Database Contract
Realization / Provider Binding
```

### R

Current Rの一部は分離されたauthorityを辿って再構成できる。

```text
browse Pattern
  -> Query
  -> readable datasource
  -> Data

reference Pattern
  -> Contract Query
  -> Database Contract read
  -> target Data

header-detail Pattern
  -> realization binding
  -> Contract Query
  -> Database Contract read
  -> detail Data
```

これはX-TEAの`OpR`相当meaningがTEIでは複数層へ分解されているが、exact link chainが揃っているためderived projection可能だと解釈できる。

### C/U/D

Current TEIには:

```text
Database Contract Writes[]
```

は存在するが、

```text
Application Function / Capability
  -> this mutation authority
```

というaccepted linkがない。

さらに`Writes[]`はC/U/D subtypeを区別しない。

したがってX-TEAの`TaskFunctionIO.OpC / OpU / OpD`に相当するoperation attribution informationが、current TEIではまだlosslessに表現されていない。

## 4. Translation conclusion

X-TEA fallbackから得られる結論は:

```text
Business ProcedureへCRUD letterをauthorする
```

ではない。

むしろ:

> C/U/D attributionがrealistic Referenceで必要になった場合、Business ProcedureよりFunction / I/O realization境界を第一候補として再検討する。

である。

Business Procedureは:

```text
what business work happens
who / when / why
manual / external / judgment / handoff
which Function is used
```

を保持し、FunctionがDataへどう作用するかというoperation meaningは別authorityへ残す。

## 5. TEI option refinement

### X0 — Business Procedure CRUD annotation

```text
BP-07 U sales.order
```

X-TEA translation上もrejectする。

理由:

- historical CRUD sourceはFunction/Table I/OでありBusiness Procedure textではない
- Data / Function側authorityとの二重SOTになる
- realization変更でdriftする

### X1 — Application Function / Pattern Data-I/O operation relation

Conceptual example:

```text
Application Function
  -> Data I/O relation
     target semantic Data / Attribute
     operation C/R/U/D
```

X-TEA shapeへ最も近いtranslation。

Pros:

- Function reviewからCRUD matrixを直接derive可能
- Business Procedureを太らせない
- semantic Data identityへ結べる

Risks:

- `Application Pattern`は現在UI-independent intentであり、persistence operationまで所有させると責務が膨張する
- Capability / Query / Database Contractとのduplicate authorityになり得る
- external API / physical action等をCRUDへ押し込む危険

Current: **candidate only / not adopted**。

### X2 — Capability / Application realization -> mutation authority binding

Conceptual example:

```text
Application + Capability
  -> effect realization
     Database Contract mutation target
```

Pros:

- Semantic Capabilityをpersistence-neutralに維持できる
- provider / database dependentなeffectをrealization側へ置ける
- Custom Go / Plugin / external API realizationの違いを保持できる

Risks:

- Business reviewでFunction/Data CRUDを得るためにrealizationまで辿る必要がある
- one Capabilityが複数effectsを持つ場合のaggregation semanticsが必要
- Database以外のeffectとのgeneralizationを先回りしやすい

Current: **strong future candidate / evidence insufficient**。

### X3 — Database mutation contract + explicit Function binding

Current Query Contractに対するmutation counterpartを設け、Function / Capability realizationからtarget identityへbindする案。

Conceptual example:

```text
Database Contract Mutation
  target Data / Attribute
  operation semantics

Capability realization
  -> Database Contract Mutation target
```

Pros:

- read Query Contractと対称に近いexplicit mutation authorityを持てる
- physical SQL本文からoperationを推測しない
- Database ownership / access policyとcross-checkしやすい

Risks:

- current Database Contract `Writes[]`とのrelationを整理する必要
- CRUD C/U/DをDatabase contract vocabularyへ固定してよいか未証明
- external non-DB effectには別modelが必要

Current: **future pressure / not adopted**。

### X4 — generic Capability Effect

```text
capability ... {
  effect ...
}
```

Currentはdeferする。

X-TEAのCRUDを一般化して`effect`へ広げるのは、current C/U/D exampleが0の段階では抽象化が強すぎる。

## 6. Predecessor mutation-boundary comparison

Current TEIの前身設計であるSemantic PlatformのAction execution draftでは、Actionは必要に応じてsubject Dataを持ち、internal Action callを明示し、runtime execution中の`create / update / delete` stepごとにownershipを再検査していた。

概念上は:

```text
Action
  subject Data
  explicit internal call
  create/update/delete step
        ↓
Runtime mutation gate
  owner/subsystem check
        ↓
commit
```

である。

さらにexternal implementationへ無制限なbusiness DB commit権限を渡さず、result / `MutationIntent`を返してRuntime側でvalidate / commitする方向が検討されていた。Detailed MutationIntent schemaは当時もopenだった。

このpredecessor designをcurrent TEIへそのまま戻さない。特にAction Script、Subsystem、MutationIntentをcurrent Canonical kindとして復活させる根拠にはしない。

一方、設計圧力としては重要である。

### 6.1 X2との整合

Predecessor boundaryは:

```text
Semantic/Business Function meaning
  != unrestricted physical DB mutation
```

を明確に分けていた。

External / custom implementationが直接DB authorityになるのではなく、mutationをRuntime-controlled authorityへ戻す方向は、X2の:

```text
Application + Capability realization
  -> explicit mutation authority
```

と整合する。

したがってX2は単なる新規抽象化案ではなく、TEI系譜内でも「実装をauthorityにしない」という既存設計原則の延長として比較できる。

### 6.2 X3との整合

Predecessor `MutationIntent`は詳細schema未確定だったため、current Database Contract / Query Contract分離を踏まえると、mutationをDatabase-owned contractとして明示しFunction realizationからbindするX3も自然な比較候補になる。

ただし:

```text
old MutationIntent
  != current Database Mutation Contract
```

であり、名前やshapeを継承しない。

### 6.3 Direct / effective operationとの関係

Predecessor Action modelではinternal callが明示され、旧Reference/Studioではdirect CUDとsame-Subsystem call propagationによるeffective CUDを別表示していた。

したがってfuture mutation attribution modelは、少なくとも:

```text
direct mutation authority
explicit Function call dependency
effective operation projection
published / ownership boundary
```

を区別できる必要がある。

Flattened CRUD cellだけをCanonical SOTにしない。

### 6.4 Current TEIとの差

Current `semantic.Capability`はinput / output / outcomeのみを持ち、call/effect/mutationを所有しない。

Current `Application Pattern command`もCapability identityだけを参照し、persistence effectを所有しない。

そのためpredecessor boundaryをcurrentへ適用するには、まずrealistic C/U/D caseから「何を失っているか」を証明する必要がある。

## 7. Current ranking

Current Evidenceだけなら:

```text
X0 Business Procedure CRUD     -> reject
X1 Function/Data-I/O relation  -> plausible, responsibility risk
X2 realization effect binding  -> plausible, current TEI separation + predecessor mutation gateに整合しやすい
X3 mutation contract + binding -> plausible, Query Contract分離 + predecessor MutationIntent pressureと比較可能
X4 generic Effect              -> defer
```

X2 / X3のどちらがよいかは、Create / Delete / non-state UpdateのTEI-original realistic Referenceがないため決めない。

## 8. Required future pressure

Qualifying C/U/D exampleが自然に現れた場合、少なくとも次を比較する。

1. 同じBusiness Action / Function meaningを保ったままrealizationをDatabase -> external APIへ変えた場合、operation attributionはどこに残るべきか。
2. one Capabilityが複数Data / resourceへeffectを持つ場合、matrix cellをどうderiveするか。
3. Create / DeleteがSemantic Data lifecycle meaningなのかpersistence effectなのか。
4. non-state UpdateとBusiness Lifecycle transitionをどう分離するか。
5. external-observed Databaseに対するwrite prohibitionをoperation relationからcross-checkできるか。
6. operation projectionをBusiness Procedure / Function / Dataの三視点reviewへどう戻すか。
7. direct mutationとexplicit Function-call由来のeffective mutationを分離できるか。
8. published / cross-boundary Function内部mutationをcallerのdirect CUDへ誤帰属しないか。
9. custom / external implementationにunrestricted DB authorityを渡さず、accepted mutation authorityを検査できるか。

## 9. Current production boundary

今は追加しない。

- Business Procedure CRUD field
- Application Pattern generic CRUD field
- Semantic Capability generic Effect
- Database Contract C/U/D enum
- mutation binding
- MutationIntent
- Function/Data-I-O canonical kind
- direct/effective CRUD production fields
- Action Script / generic call DSL

Current partial R projectionを維持し、C/U/Dはunknownを保持する。

## 10. X-TEA fallback result

Historical shapeをそのまま移植しない翻訳結果は:

```text
X-TEA
  Function/Table I/O owns explicit CRUD marks
  -> matrix derived

TEI predecessor
  Action subject + explicit call + mutation gate
  -> direct/effective CUDを分離
  -> external implementationはunrestricted DB authorityではない

TEI current
  R meaning is distributed but exact-link recoverable
  C/U/D attribution link is missing
  Capabilityにはcall/effect/mutation authorityがない

TEI direction
  keep Business Procedure thin
  investigate Function / realization / mutation-authority boundary when realistic C/U/D evidence appears
  preserve direct/effective distinction as future guardrail
  derive matrix; do not author matrix
```

この結論は:

- `BUSINESS_ACTION_CUD_TEI_ORIGINAL_REFERENCE_SEARCH_2026-09.md`
- `BUSINESS_ACTION_CRUD_DIRECT_EFFECTIVE_BOUNDARY_PRESSURE_2026-09.md`
- `BUSINESS_PROCEDURE_BP01_PREPARE_OPERATION_AMBIGUITY_PRESSURE_2026-09.md`

と組み合わせて扱う。
