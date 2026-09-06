# Business Procedure Normalized Model Pressure — 2026-09

Status: **Lane C / test-only normalized-model pressure / H1+R3+S4-A / no production `ir.World` adoption**

## 1. 目的

current primary pressureを一つのtest-only normalized modelとして組み合わせる。

```text
D3
  thin Business Procedure body
  + Step body meaning / addressability separation

H1 / R3
  accepted handoffをBusiness側のthin handoff-only relationとして保持

S4-A
  Procedure-scoped Step local addressをnormalized Procedure側へ保持
```

Authoring sourceをnormalizeした後、Source Index / file path / fingerprintへ戻らずBusiness Procedure / Handoffのreferential integrityを検査できるかを確認する。

このmodelは**Business Flow / DFD modelではない**。

## 2. test-only shape

```text
NormalizedBusinessModel
  Procedures[]
    id
    purpose
    trigger
    actor
    ordered Steps[]
      localKey
      body
        businessAction
        explicit semantic refs

  Handoffs[]
    from Procedure/localKey
    to   Procedure/localKey
    condition
```

正式type名 / package名 / `ir.World` fieldではない。

## 3. 含めないもの

### Source / review infrastructure

```text
source path
artifact locator
fingerprint
repository revision
byte range
ETag
```

### Runtime / security

```text
Workflow instance
execution token
queue / retry / timer / saga / scheduler
Authorization identity
single Application owner
realization review enum
```

### full Business Flow / DFD

```text
BusinessFlows
FlowNodes
ProcessNodes
ExternalParties
Roles as flow participants
InformationFlows
Documents
Stores
Payloads
FlowType
```

これらをHandoff用normalized modelへ混ぜない。

## 4. normalized-only validation

`validateNormalizedBusinessModelProbe`はnormalized modelだけをinputとする。

検査:

- duplicate Procedure address
- empty / duplicate Step local address
- handoff source endpoint existence
- handoff target endpoint existence
- empty Business condition
- duplicate handoff relation

要求しないもの:

- Reference artifact locator
- Source Index
- source fingerprint
- file system
- `workspace.Load()`

Sourceを読むのはnormalize前だけである。

## 5. locator invariance

Procedure artifact locatorだけを変更しても、accepted Business source meaningが同じならnormalized modelを変えない。

```text
SourceLocatorChanged
  !=
BusinessMeaningChanged
```

RQ-002のSource-to-Canonical境界を守る。

## 6. Step local address

local keyはBusiness body meaningそのものではなく、Procedure-scoped referential metadata候補である。

```text
BP-09 -> BP-09-renamed
```

だけならBusiness Action bodyは不変でよい。

一方handoffが旧`BP-09`を指し続ければreferential integrityはfail closedする。

Step local addressとhandoff endpointを協調更新すればrelationは再解決できる。

概念上:

```text
BodyMeaningChanged
ReferentialAddressChanged
SourceLocatorChanged
```

を分ける。

このtaxonomyをproduction enumへはまだ固定しない。

## 7. handoff condition

自然言語conditionをcurrent accepted Business relation meaningとして保持する。

condition変更はnormalized model差になる。

まだ次へ変換しない。

- executable predicate
- branch DSL
- runtime event guard
- state-machine guard

## 8. H0 / S4-Bへの反証

### H0 — handoff authoring-only

Procedure bodyだけではcondition / endpoint changeをnormalized semantic diffから観測できない。

### S4-B — Step address Source Index-only

normalized handoffを検証するためにSource Indexへ戻る必要が生じ、`Semantic validation does not depend on Source Index`というarchitecture境界と緊張する。

current PressureではH1 + R3 + S4-Aが最も説明力を持つ。

## 9. Business Flow / DFDとのnegative boundary

X-TEA fallbackでは、Business FlowはProcess(Task) node、他participant / storage-like node、DataflowLine等を持つ別のdesign pressureだった。

したがってcurrent modelの:

```text
Procedures[] + Handoffs[]
```

をBusiness Flow completenessとみなさない。

`business_flow_scope_pressure_test.go`で次を固定する。

1. normalized modelにBusiness Flow collectionを置かない
2. Handoffは`From / To / Condition`だけ
3. endpointは`Procedure / Step`だけ
4. Actor / Role / ExternalParty / Process / Data / Information / Document / Store / Payload / FlowType等をHandoffへ入れない
5. current validator PASSをBusiness Flow / DFD Conformanceと呼ばない

関連Evidence:

- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md` E-014
- `plan/BUSINESS_PROCEDURE_HANDOFF_VS_BUSINESS_FLOW_PRESSURE_2026-09.md`

## 10. Application context

single Application ownerはnormalized Business Procedure meaningへ入れない。

Applicationとの関係やpolicyはexplicit Pattern refsをnormalized Worldで解決してderiveする。

fixture-level Applicationはoptional reporting contextでありsemantic authorityではない。

## 11. まだ決めないもの

- production `BusinessModel` type
- `ir.World.Business` field
- production `BusinessProcedure` / `BusinessHandoff`名
- Step local address syntax / generation rule
- Procedure source DSL
- Business Flow / DFD model
- generic Business Edge
- 000039 change graph integration

## 12. Lane Bとの整合

Lane B RQ-002監査も独立に次を確認している。

```text
identity-only shell
  insufficient as complete Business meaning

thin normalized Business residual meaning
  strongest current production candidate pressure
  not adopted

Step body meaning
  != Step addressability

thin Business handoff relation
  useful independent semantic pressure
  not adopted
```

本probeはこれをproduction採用へ昇格せず、Lane C Reference Evidenceとして具体化する。

## 13. ADR前の意味

actual Repository executionでgreenになれば、current Referenceに対して少なくとも:

```text
thin Procedure body
+ Procedure-scoped Step local address
+ thin handoff-only Business relation
```

で、Source infrastructure / Workflow runtime / full Business Flow modelを混ぜずにaccepted Business meaningを正規化できるEvidenceになる。

それでもproduction採用には:

- actual green
- より現実的なReference pressure
- Lane B再監査
- source / Canonical / Projection authority境界の再確認

が必要である。

## 14. 実装位置

- `tools/reference-business-procedure-check/normalized_business_model_pressure_test.go`
- `tools/reference-business-procedure-check/business_flow_scope_pressure_test.go`

主Pressure:

1. normalized modelだけでhandoff endpoint検証
2. Source/runtime fieldsなし
3. artifact locator moveでmodel不変
4. stale local address fail closed
5. coordinated address rename成立
6. missing endpointをSource Indexなしでreject
7. condition変更をBusiness semantic relation差として保持
8. full Business Flow surfaceを所有しない
9. Handoff validationをBusiness Flow Conformanceへ過大主張しない

actual `go test` greenまではtest-only specificationである。
