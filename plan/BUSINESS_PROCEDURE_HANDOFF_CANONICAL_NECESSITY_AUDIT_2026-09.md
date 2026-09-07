# Business Procedure Handoff Canonical Necessity Audit — 2026-09

Status: **Lane C / pre-ADR pressure audit / normalized meaning required hypothesis / no dedicated Core kind adopted**

## 1. 問い

Business Procedure間handoffをcurrent Referenceでaccepted Business meaningとして扱う場合、そのrelationをAuthoring / Reference setだけへ残し、normalized / Canonical meaningから完全に落としてよいか。

ここで問うのは専用`BusinessHandoff` kindの採用ではない。

```text
H0: accepted handoffはauthoring-only
H1: accepted handoffの薄いnormalized meaningを保持
```

を比較する。

## 2. Requirement pressure

RQ-025解釈ではBusiness Procedureが必要に応じて次を説明する。

```text
purpose
trigger / event
actor / job context
Business Action
manual / external work
Application Function usage
handoff
completion / continuation
```

PAS-07 supporting contractでも、少なくとも次がAcceptance pressureとして置かれている。

```text
P25-4
  handoff != Runtime Workflow

P26-4
  reuse / handoff endpointをlogical keyで追跡
```

したがってcurrent Referenceに明示されたhandoffを単なるgraph decorationとみなす根拠は弱い。

## 3. H0 — authoring-only handoff

```text
Canonical / normalized Procedure bodies
  order-entry.order-handling
  order-entry.follow-up

Authoring-only relation
  order-entry.order-handling/BP-09
    -> order-entry.follow-up/FU-00
    + condition
```

### 問題

handoff conditionだけが変更された場合、Procedure bodyは変わらない。

```text
before body == after body
```

同様に、handoff endpointだけを`BP-09 -> BP-08`へ変更してもProcedure bodyは変わらない。

Canonical / normalized側にbodyしかなければ、accepted handoff relationの意味変更をsemantic diff / impact / Conformanceから観測できない。

これはidentity-only Procedure shellがBusiness-only wording changeを失った問題と同型である。

## 4. H1 — thin normalized handoff meaning

最小Pressure形:

```text
Business handoff meaning
  from
    Procedure address
    Step local address
  to
    Procedure address
    Step local address
  condition
```

### 保持できる差

- condition change
- from endpoint change
- to endpoint change
- stale endpoint
- duplicate relation

### 入れないもの

- source path
- fingerprint
- repository revision
- runtime instance
- queue
- retry
- timer
- scheduler
- saga
- execution ownership
- transport
- Authorization identity

H1はWorkflow Engineではない。

## 5. test-only反証

`tools/reference-business-procedure-check/handoff_canonical_necessity_pressure_test.go`で次をPressureする。

1. handoff conditionだけを変更してもProcedure bodiesは不変
2. body-only snapshotではrelation changeを観測できない
3. normalized handoff projectionを含めるとcondition changeを観測
4. endpoint変更もProcedure bodyを変えずreferential relation差として観測

既存`canonical_handoff_pressure_test.go`と合わせ、locator moveではhandoff meaningを変えず、missing endpointはfail closedする。

actual Repository runnerでgreenになるまでは実行Evidenceとは数えない。

## 6. Source Indexとの境界

H1を採る場合、Canonical semantic validationがSource Indexへ依存しない現行architectureを守る必要がある。

したがってhandoff endpointがStepを指すなら、Step addressabilityをSource Indexだけに置くS4-Bは不利になる。

```text
H1 + Canonical validation
  -> S4-A pressure
     normalized Procedure-scoped Step local address metadata
```

ただしこれはglobal Step IDを要求しない。

## 7. 専用kindはまだ不要

H1から直ちに次を導かない。

```text
ir.World.BusinessHandoffs []BusinessHandoff
```

必要なのは「accepted handoff meaningをnormalized semantic layerが失わないこと」である。

実装候補は別に比較できる。

- thin dedicated relation
- Procedure-related generic Relation
- Business semantic edge collection
- 将来のBusiness Flow modelの一部

既存Micro / Semantic Kernelへ専用kindを増やす負担はADR前に比較する。

## 8. 現時点の判断

current `order_entry` ReferenceではhandoffをBusiness authoring meaningとして明示し、RQ-025 / PAS-07もhandoffを業務手順のAcceptance pressureとして扱っている。

そのためpre-ADR判断は次とする。

```text
H0 authoring-only handoff
  -> accepted semantic changeをnormalized layerから失うため不十分な可能性が高い

H1 thin normalized handoff meaning
  -> primary candidate
```

ただしproduction dedicated kind採用は未決定。

## 9. 次のPressure

H1を前提候補として、次を確認する。

1. S4-AのProcedure-scoped local Step addressだけでreferential integrityを閉じられるか
2. handoff relationをgeneric semantic relationで表せるか、専用Business relationが必要か
3. handoff conditionを自然言語meaningのまま保持して十分か
4. Business Flow / DFD型flowとProcedure handoffを同一objectへ畳まないか
5. 000039 semantic diff / impactがH1を扱える粒度を持てるか

## 10. 非対象

- production Canonical kind採用
- Workflow Engine
- executable branch / predicate DSL
- runtime event bus
- retry / queue / scheduler
- Business Flow全体のmodel確定
- 000039実装
