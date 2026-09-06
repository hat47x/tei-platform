# Business Procedure Normalized Model Pressure — 2026-09

Status: **Lane C / test-only normalized-model pressure / no production `ir.World` adoption**

## 1. 目的

前段で主候補となった二つを、一つのtest-only normalized modelとして組み合わせる。

```text
H1 / R3
  accepted handoffをBusiness側のthin normalized relationとして保持

S4-A
  Procedure-scoped Step local addressをnormalized Procedure側へ保持
```

ここで確認するのは、Authoring sourceをnormalizeした後、Source Index / file path / fingerprintへ戻らずBusiness referential integrityを検査できるかである。

## 2. test-only shape

```text
NormalizedBusinessModel
  Procedures[]
    id
    purpose
    trigger
    actor
    ordered Steps[]
      localKey       // Procedure-scoped address metadata
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

```text
source path
artifact locator
fingerprint
repository revision
byte range
ETag
runtime Workflow instance
execution token
queue / retry / timer / saga / scheduler
Authorization identity
single Application owner
realization review enum
```

Source / Runtime / Securityの責務をBusiness normalized meaningへ戻さない。

## 4. normalized-only validation

`validateNormalizedBusinessModelProbe`はinputとしてnormalized modelだけを受ける。

検査するもの:

- duplicate Procedure address
- empty / duplicate Step local address
- handoff source endpoint existence
- handoff target endpoint existence
- empty Business condition
- duplicate handoff relation

検査時に次を要求しない。

- Reference set artifact locator
- Source Index
- source fingerprint
- file system
- `workspace.Load()`

Sourceを読むのはnormalize前だけである。

## 5. locator invariance

Reference setのProcedure artifact locatorだけを変更しても、すでに解決されたBusiness source meaningが同じならnormalized modelを変えない。

```text
source locator changed
  !=
Business meaning changed
```

RQ-002のSource-to-Canonical boundaryを守る。

## 6. Step local address

local keyはBody meaningそのものではない。

```text
BP-09 -> BP-09-renamed
```

だけならordered Business body wording / refsは同じままでよい。

一方、handoffが旧`BP-09`を指し続ければnormalized referential integrityはfail closedする。

Step local addressとhandoff endpointを同時に更新すればrelationは再び解決できる。

したがって:

```text
BodyMeaningChanged
ReferentialAddressChanged
```

を同一視しない。

## 7. handoff condition

自然言語conditionはcurrent Referenceではaccepted Business relation meaningとして保持する。

condition変更はnormalized model差になる。

ただしconditionを次へ変換しない。

- executable predicate
- branch DSL
- runtime event guard
- state-machine condition

## 8. H0 / S4-Bへの反証

### H0: handoff authoring-only

Procedure bodyだけではcondition / endpoint changeをnormalized semantic diffから観測できない。

### S4-B: Step address Source Index-only

Canonical / normalized handoffを検証するためにSource Indexへ戻る必要が生じ、現行architectureの`Semantic validation does not depend on Source Index`境界と緊張する。

current PressureではH1 + R3 + S4-Aの組合せが最も説明力を持つ。

## 9. まだ決めないもの

- production `BusinessModel` type
- `ir.World.Business` field
- production `BusinessProcedure` / `BusinessHandoff`名
- Step local addressのsyntax
- local address生成規則
- Procedure source DSL
- Business Flow / DFD model
- 000039 change graph integration

## 10. Lane Bとの整合

Lane B RQ-002監査は独立に次を確認している。

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

本probeはこのcontractをproduction採用へ昇格せず、Lane C側のReference Evidenceを具体化する。

## 11. ADR前の意味

このprobeがactual Repository executionでgreenになれば、少なくともcurrent Referenceに対して:

```text
thin Procedure body
+ Procedure-scoped Step local address
+ thin Business handoff relation
```

で、Source infrastructureやWorkflow runtimeを混ぜずにBusiness meaningを正規化できるEvidenceになる。

それでもproduction adoptionは、より現実的なReference pressureとLane B再監査を経て決める。

## 12. 実装位置

`tools/reference-business-procedure-check/normalized_business_model_pressure_test.go`

主Pressure:

1. normalized modelだけでhandoff endpointを検証
2. source/runtime container fieldを持たない
3. artifact locator moveでmodel不変
4. stale Step local addressをfail closed
5. local address + handoff endpoint協調renameは成立
6. missing endpointをSource Indexなしでreject
7. condition変更をBusiness semantic relation差として保持

actual `go test` greenまではtest-only specificationである。
