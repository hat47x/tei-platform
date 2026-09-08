# Business Procedure Application Context Pressure — 2026-09

Status: **Lane C / test-only architecture pressure / C1 implemented in experiment checker / no Core adoption**

## 1. 問い

current Reference fixtureはtop-levelに

```text
application = order_entry
```

を持つ。

Business Procedureがone Procedure -> multiple Applications、manual work、external workを跨げることがPressureで示された後も、このfieldをBusiness Procedureのowner / authorityとして扱う必要があるかを再監査した。

## 2. 確認できたこと

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

## 3. 旧前提と修正

旧`evaluate()`には次の検査が残っていた。

```text
fixture.Application
  -> findApplication(world, fixture.Application)
  -> missingなら BUSINESS_APPLICATION_NOT_FOUND
```

これはcross-view access判定には使われず、one Procedure -> multiple Applications pressureとも緊張していた。

実験checkerではこの存在必須checkを除去し、`fixture.Application`をoptional reporting metadataへ下げた。

```text
fixture.Application
  -> reportへ表示してよい
  -> absentでもよい
  -> unknownでもsemantic ownerとはみなさない
  -> Pattern / policy authorityには使わない
```

## 4. Option比較

### C1 — optional reporting context

```text
reference fixture
  application?  // authoring/reporting only

Canonical Business Procedure
  no application owner
```

実験checkerで採用した。

確認Pressure:

- context absentでもexplicit Pattern refsだけでbaseline cross-view evaluationが成立
- unknown contextをblocking semantic errorにしない
- reportにはcontext labelをそのまま残せる
- thin Canonical Business meaningは不変

### C2 — reporting context自体をfixtureから除去

まだ採用しない。

Referenceの入口や人間向け説明にprimary contextが有用な場合があるため、authoring metadataまで削る必要性はcurrent Evidenceからは出ていない。

### 採らない案

```text
BusinessProcedure.application = required owner
```

one Procedure -> multiple Applications pressureと整合しないため採らない。

## 5. test / implementation pressure

Repository上では次をtest-onlyで固定した。

1. fixture Application変更でthin meaning不変
2. cross-Application accessはPattern owner Applicationを使用
3. fixture primary contextが存在してもProcedure ownerへ昇格しない
4. fixture Applicationが空でもbaseline cross-view evaluationを継続
5. explicit Pattern refからData meaningを導出
6. unknown reporting contextをsemantic ownerとしてblocking validationしない
7. reporting labelを別Applicationへ暗黙補完しない

actual Repository runnerでgreenになるまでは実行Evidenceとは数えない。

## 6. pre-ADR判断

```text
single Application owner
  -> production Canonical Business Procedureへ入れない

Pattern owner Application
  -> cross-view realization / policy traceのauthority

fixture Application
  -> optional authoring/reporting context
  -> semantic ownership authorityではない
```

C1でcurrent Pressureは閉じる。

C2のfield完全除去は、実際にauthoring surfaceを設計するときに必要性が出た場合だけ再検討する。

## 7. Watanabe pressureとの関係

Business Modelを社会的な業務手順として扱うなら、一つの手順が複数システム、人、外部組織、物理作業を跨ぐことは自然である。

そのためBusiness Procedureを一つのApplication containerへ閉じ込めず、Function-Interactionとの関係をexplicit referenceでcross-checkする方が、三視点を独立させつつ相互批判可能にする設計へ整合する。

## 8. 非対象

- production DSL syntax確定
- Reference fixture schemaVersion更新
- reporting field正式名称
- Application Pattern ownership modelの変更
- Authorization owner model
- Runtime Workflow ownership
- Gate B / Gate C scope変更
