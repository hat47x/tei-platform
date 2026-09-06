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
- Application Pattern / Data / Relation / Validation / Capabilityをidentityで参照する
- Business Action × Application Pattern viewをderivedできる
- explicit Data / Relation referenceだけをderivedし、名称類似から推測しない

baseline確認:

```bash
go run ./tools/reference-business-procedure-check
```

成功時はJSON reportを出力し、blocking diagnosticなしで終了します。

出力には各Business Actionについて、Application Pattern参照または`noApplicationFunction: true`が含まれます。

## `order-entry-customer-edit-conflict.json`

最初の意図的cross-view defectです。

`order_entry.customer_reference`は`master` datasourceを使いますが、現行Application contractは`master`を`read`として要求しています。

probeはBP-02へexperiment-onlyの`read-write`要求を重ねます。

```text
Business / Function candidate
  edit customer through order_entry.customer_reference
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
3. baselineの`master=read`はconflictにならない
4. edit probeの`read-write`要求はaccess conflictになる
5. fixtureの未知fieldを黙って受理しない

## Authority boundary

このReferenceで証明するのはBusiness Procedure**viewpointとcross-view diagnosticの必要性**です。

このartifactの存在だけを根拠に、次を採用しません。

- Canonical `Procedure`
- Canonical `Step`
- Workflow Engine
- retry / timer / queue / scheduler
- Business actorからAuthorization Roleへの自動mapping

Procedure identityのdurabilityは、impact analysis、review history、reuse、regenerationで意味が失われるかを別Evidenceとして確認してから判断します。
