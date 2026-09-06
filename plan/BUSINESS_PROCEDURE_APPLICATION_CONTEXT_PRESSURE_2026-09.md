# Business Procedure Application Context Pressure — 2026-09

Status: **Lane C / test-only architecture pressure / no Core adoption**

## 1. 問い

current Reference fixtureはtop-levelに

```text
application = order_entry
```

を持つ。

Business Procedureがone Procedure -> multiple Applications、manual work、external workを跨げることがPressureで示された後も、このfieldをBusiness Procedureのowner / authorityとして扱う必要があるかを再監査する。

## 2. すでに確認できたこと

### 2.1 thin Canonical meaningには不要

`fixture_application_context_subtraction_test.go`では、fixture-level `Application`だけを変更してもrealization-subtracted thin Business meaningは変わらない。

したがってcurrent candidateでは、単一Application ownerはBusiness Procedure body meaningに含めない。

### 2.2 access authorityには使わない

cross-Application access probeは、fixture-level Applicationではなくnormalized `ir.World`上でexact解決したApplication Patternのowner Applicationからdatasource requirementを取得する。

```text
Business Procedure step
  -> explicit Pattern ref
  -> exact Pattern owner Application
  -> datasource requirement / accepted access
```

primary fixture contextが`order_entry`でも、`fulfillment_console.*` Patternのpolicyは`fulfillment_console`から読む。

### 2.3 Application associationはderived

Procedureが関係するApplication集合は、explicit Application Pattern refsをnormalized Worldで解決して得る。

Pattern ID文字列のprefix分解やfixture-level owner fieldをauthorityにしない。

## 3. 発見した旧前提

current `evaluate()`にはまだ次の検査が残る。

```text
fixture.Application
  -> findApplication(world, fixture.Application)
  -> missingなら BUSINESS_APPLICATION_NOT_FOUND
```

これはcross-view access判定にはもう使われない。

したがって現在の`application` fieldは二つの意味が混ざった過渡状態にある。

```text
A. Reference reporting / scenario context
B. semantic owner / authority
```

Bは現在のPressureでは不要になった。

## 4. 次の比較

### Option C1 — optional reporting context

```text
reference fixture
  primaryApplicationContext?  // authoring/reporting only

Canonical Business Procedure
  no application owner
```

- contextはUI / report / Reference説明にのみ使う
- absentでもexplicit Pattern refsが解決すればcross-view evaluation可能
- unknown contextをblocking semantic errorにしない

### Option C2 — reporting context自体をfixtureから除去

Referenceの説明はProcedure purpose / explicit refs / surrounding corpusから行い、単一primary contextを持たない。

- schemaは薄くなる
- true multi-Application Procedureに自然
- 一方、Reference reportの入口や人間向け説明が弱くならないか確認が必要

### 採らない案

```text
BusinessProcedure.application = required owner
```

これはone Procedure -> multiple Applications pressureと整合しない。

## 5. test / implementation pressure

既存testsで次は実装済み。

1. fixture Application変更でthin meaning不変
2. cross-Application accessはPattern owner Applicationを使用
3. fixture primary contextが存在してもProcedure ownerへ昇格しない

次に閉じるPressureは以下。

4. fixture Applicationを空 / absentにしてもexplicit refsだけでbaseline cross-view evaluationが成立するか
5. report上のcontextが必要ならsemantic validationと分離できるか
6. contextが未知でもPattern refsがexact解決する場合、blocking semantic errorにする根拠があるか
7. current JSON fieldをrename / optional化 / 除去する場合のReference source compatibility

## 6. pre-ADR判断

現時点では次を固定する。

```text
single Application owner
  -> production Canonical Business Procedureへ入れない

Pattern owner Application
  -> cross-view realization / policy traceのauthority

fixture Application
  -> authoring/reporting context候補
  -> semantic ownership authorityではない
```

`fixture.Application`のoptional化 / 除去は、actual testsとReference source compatibilityを確認してから決める。

## 7. Watanabe pressureとの関係

Business Modelを社会的な業務手順として扱うなら、一つの手順が複数システム、人、外部組織、物理作業を跨ぐことは自然である。

そのためBusiness Procedureを一つのApplication containerへ閉じ込めず、Function-Interactionとの関係をexplicit referenceでcross-checkする方が、三視点を独立させつつ相互批判可能にする設計へ整合する。

## 8. 非対象

- production DSL syntax確定
- Reference fixture schemaVersion更新
- Application Pattern ownership modelの変更
- Authorization owner model
- Runtime Workflow ownership
- Gate B / Gate C scope変更
