# Business-system design Reference

このdirectoryは、`reference/workspace`のCanonical meaningを変更せずに、Data / Function-Interaction / Businessのcross-view設計境界を再現するためのexperiment fixtureを保持します。

現時点ではBusiness ProcedureをCanonical Workspaceのsource rootへ追加しません。

## `order-entry-business-procedure.json`

`order_entry`の既存Application / Semantic identityを参照するderived Business Procedure fixtureです。

このfixture自身はCanonical authorityではありません。

```json
{
  "artifactRole": "derived-reference-fixture",
  "canonical": false
}
```

主な圧力は次です。

- Business ActionとApplication Functionを同一視しない
- manual / external / unresolved stepにfake Capabilityを要求しない
- Application Patternをexact identityで参照する
- Patternがすでに所有するData / Relation / Query / Capability / DatasourceをBusiness Procedureへ重複記述しない
- Pattern-owned meaningはnormalized WorldからBusiness Action × Pattern meaning viewへderivedする
- Procedure側で独自に必要なValidation等のexplicit referenceだけを別に保持する
- 名称類似からmissing relationやCapabilityを推測しない

### 三視点は三重管理ではない

Business ProcedureからApplication Patternを参照した場合、次の意味はPattern側を正とします。

```text
Application Pattern
  ├─ Data
  ├─ Relation
  ├─ Query
  ├─ Capability
  └─ Datasource
```

したがって、たとえばBP-02は次だけを保持します。

```text
Business Action:
  confirm the customer associated with the order

Application Pattern:
  order_entry.customer_reference

Business-side access expectation:
  master = read
```

`customer_reference`が`relation=sales.order.customer`、`datasource=master`であることはfixtureへ再記述せず、`workspace.Load()`後のnormalized Application Patternから導出します。

同様に、BP-07が`order_entry.submit_order`を参照するとき、`capability=sales.order.submit`はPatternから導出します。

これはData / Function / Businessの視点を分離するために、同じ意味を三箇所へコピーする設計を避けるためです。

### direct semantic reference

Patternから導けない意味をBusiness Procedure側が明示的に要求する場合だけ、Data / Relation / Validation / Capabilityのdirect referenceを持てます。

現在のfixtureではBP-03 / BP-05のValidationがこれに該当します。

checkerはdirect referenceもexact identityで解決し、似た名称から補完しません。

baseline確認:

```bash
go run ./tools/reference-business-procedure-check
```

成功時はJSON reportを出力し、blocking diagnosticなしで終了します。

出力には少なくとも次が含まれます。

- `businessActionFunctionRows`
  - 各Business ActionのApplication Pattern参照
  - Patternを持たない場合は`noApplicationFunction: true`
- `businessActionPatternMeaningRows`
  - PatternからderivedしたData / Relation / Query / Capability / Datasource
- `explicitBusinessMeaningRows`
  - Business Procedure fixture自身が明示したdirect semantic reference

## `order-entry-customer-edit-conflict.json`

最初の意図的cross-view defectです。

`order_entry.customer_reference`は`master` datasourceを使いますが、現行Application contractは`master`を`read`として要求しています。

probeはBP-02へexperiment-onlyの`read-write`要求を重ねます。

```text
Business / Function candidate
  edit customer through order_entry.customer_reference
        ↓
Pattern-derived datasource
  master
        ↓
Application contract
  datasource master = read
        ↓
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

再現:

```bash
go run ./tools/reference-business-procedure-check \
  -probe reference/business-system-design/order-entry-customer-edit-conflict.json \
  -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

`-expect-diagnostic`を指定した場合、期待したblocking diagnosticだけが出たときに成功します。期待外のsemantic / architecture-policy diagnosticが同時に出た場合は失敗します。

このprobeは「Customer編集を永久に禁止する」という要求ではありません。

現在の`order_entry` contractではそのinteractionを実現できないことを示します。実際にCustomer更新が必要なら、Projectionだけでwriteを追加せず、別Application / accepted Capabilityへ接続するか、authoritative Application / Database Contractへ戻る必要があります。

## Unit test

```bash
go test ./tools/reference-business-procedure-check
```

少なくとも次を確認します。

1. baseline fixtureのexplicit referenceがnormalized Worldへresolveする
2. BP-04 / BP-09がApplication Functionなしで正常に残る
3. Pattern-owned Data / Relation / Capabilityをfixtureが二重管理せず、normalized Patternから導出する
4. baselineの`master=read`はconflictにならない
5. edit probeの`read-write`要求はaccess conflictになる
6. experiment-local Procedure IDをrenameしてもcross-view resolution結果は変わらない
7. 類似名称からRelationを推測しない
8. fixtureの未知field、未対応schemaVersion、別artifact roleを黙って受理しない

## Authority boundary

このReferenceで証明するのはBusiness Procedure**viewpointとcross-view diagnosticの必要性**です。

このartifactの存在だけを根拠に、次を採用しません。

- Canonical `Procedure`
- Canonical `Step`
- Workflow Engine
- retry / timer / queue / scheduler
- Business actorからAuthorization Roleへの自動mapping

現時点のrename experimentでは、experiment-local Procedure IDを変更してもApplication PatternおよびPattern-derived semantic meaningへのcross-view解決は変化しません。

これは**基本的なcross-view参照のためだけなら、durable Canonical Procedure identityを直ちに要求するEvidenceはまだない**ことを示します。

一方、impact analysis、review history、reuse、regeneration、複数Procedure間の参照にdurable identityが必要かはまだ未検証です。そこを別Evidenceとして確認するまでCanonical化を決定しません。
