# Business-system design Reference

このdirectoryは、`reference/workspace`のCanonical meaningを変更せずに、Data / Function-Interaction / Businessのcross-view設計境界を再現するためのReference fixtureを保持します。

現時点ではBusiness ProcedureをCanonical Workspaceのsource rootへ追加しません。

## `order-entry-business-procedure.json`

`order_entry`の既存Application / Semantic identityを参照しつつ、Business側でしか持てないpurpose / trigger / actor候補 / manual actionを保持するReference authoring fixtureです。

このfixture自身はCanonical World authorityではありません。一方、Business Procedure側の固有意味まで「derived」とみなすこともしません。

```json
{
  "artifactRole": "reference-authoring-fixture",
  "canonical": false,
  "procedure": {
    "identityStatus": "reference-local-stable"
  }
}
```

`procedure.id`はReference corpus内でreview target、handoff endpoint、cross-referenceを安定して指すためのauthoring keyです。top-level Canonical `Procedure`を採用したことを意味しません。

主な圧力は次です。

- Business ActionとApplication Functionを同一視しない
- manual / external / unresolved stepにfake Capabilityを要求しない
- Application Patternをexact identityで参照する
- Patternがすでに所有するData / Relation / Query / Capability / DatasourceをBusiness Procedureへ重複記述しない
- Pattern-owned meaningはnormalized WorldからBusiness Action × Pattern meaning viewへderivedする
- Procedure側で独自に必要なValidation等のexplicit referenceだけを別に保持する
- 名称類似からmissing relationやCapabilityを推測しない

### Authoring inputとderived Projectionを分ける

Business Procedure fixtureには、Application / Semantic Worldからは導出できない意味があります。

```text
purpose
trigger / event candidate
actor / job context candidate
manual / external / unresolved Business Action
judgment / continuation
```

これらはReference実験のauthoring inputです。

一方、Business ProcedureからApplication Patternを参照した場合、次の意味はPattern側を正とします。

```text
Application Pattern
  ├─ Data
  ├─ Relation
  ├─ Query
  ├─ Capability
  └─ Datasource
```

したがって、たとえばBP-02は次だけをauthoring inputとして保持します。

```text
Business Action:
  confirm the customer associated with the order

Application Pattern:
  order_entry.customer_reference
```

`customer_reference`が`relation=sales.order.customer`、`datasource=master`であることはfixtureへ再記述せず、`workspace.Load()`後のnormalized Application Patternから導出します。

同様に、BP-07が`order_entry.submit_order`を参照するとき、`capability=sales.order.submit`はPatternから導出します。

三視点を分離することと、同じ意味を三箇所へコピーすることを混同しません。

### direct semantic reference

Patternから導けない意味をBusiness Procedure側が明示的に要求する場合だけ、Data / Relation / Validation / Capabilityのdirect referenceを持てます。

現在のfixtureではBP-05のValidationがこれに該当します。

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

これらのMatrix / diagnostic出力はderived Projectionです。Reference authoring fixtureへ逆書きしません。

## `order-entry-follow-up-procedure.json`

同じ`order_entry.order_record`を別のBusiness Procedure文脈から利用するReference authoring fixtureです。

primary Procedureと同じPatternを再利用してもBusiness Actionを一つへ畳まず、multiple Procedure -> same Patternのtraceabilityを圧力試験します。

このfixtureも`identityStatus=reference-local-stable`であり、Reference内のlogical authoring keyを持ちます。

## `order-entry-procedure-reference-set.json`

二つのProcedure authoring fixtureとProcedure間handoffを同じReference setとして束ねます。

Procedure keyとartifact locatorを分離します。

```text
order-entry.order-handling
  -> order-entry-business-procedure.json

order-entry.follow-up
  -> order-entry-follow-up-procedure.json
```

handoffはfile pathではなくProcedure key + Step keyを参照します。

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
```

そのため、pure file moveではlocator mappingだけを更新し、handoff semantic endpointは変えません。

一方、Procedure key自体をlogical renameしたのにhandoff linkを更新しなければfail closedにします。類似名称から接続し直しません。

このReferenceはhandoffをruntime Workflowへ変換しません。retry / timer / queue / scheduler等を含意しません。

## `second-corpus/`

`order_entry`とは独立した、二つ目のBusiness Procedure Referenceです（受注生産の製造と、完成品の入庫）。Applicationを参照しない二つのProcedureと、その間の責任の受け渡し（Handoff）一件、出典・fact・設計判断・未解決事項を分けて記録するseed ledgerを持ちます。

同じloader、正規化Business model、D4導出、standalone Handoff source形式、baseline checkerへそのまま通ります。詳細、review状態、主張しないことは[`second-corpus/README.md`](second-corpus/README.md)を参照します。

## Review / approval continuity

Reference-local Procedure keyはreview historyのtargetを束ねるためにも使います。

ただしapproval Evidenceは次のsource snapshotへ束縛します。

```text
Procedure key
Repository revision
Artifact fingerprint
Source locator
Decision
```

同じProcedure keyを保ってBusiness Action文言を変更した場合、review targetの連続性は保ちますがfingerprint / revisionが変わるため旧approvalを自動carry-forwardしません。

pure file moveでbytesが同じ場合も、Procedure key / fingerprintは保てますがsource-dependent Evidenceを別revisionへ黙って継承しません。

Procedure key自体のrenameはlogical identity changeとして扱います。

詳細は`plan/BUSINESS_PROCEDURE_REVIEW_HANDOFF_PRESSURE_2026-09.md`を参照します。

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

再現は次のとおりです。

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
6. Procedure keyだけの変更ではPattern-derived cross-view meaning自体は変わらない
7. multiple Procedure -> same Patternを区別してtraceできる
8. 類似名称からRelationを推測しない
9. Source Index Artifact fingerprintはpure file moveでは保たれるがsemantic identityとしては使わない
10. manual / external Business ActionはPattern-derived Projectionから再生成できない
11. Business-only wording changeをApplication semantic changeと混同しない
12. Reference-local stable keyでProcedure間handoffを解決できる
13. artifact moveでhandoff semantic endpointを変えない
14. logical Procedure rename後のstale handoffをfail closedにする
15. review target continuityとapproval snapshot validityを分離する
16. fixtureの未知field、未対応schemaVersion、別artifact roleを黙って受理しない

## Authority boundary

このReferenceで証明するのはBusiness Procedure viewpoint、authoring residual、cross-view diagnostic、Reference-local identity pressureです。

このartifactの存在だけを根拠に、次を採用しません。

- Canonical `Procedure`
- Canonical `Step`
- Workflow Engine
- retry / timer / queue / scheduler
- Business actorからAuthorization Roleへの自動mapping

review continuityとhandoffを加えた結果、安定した**authoring-level Procedure key**には具体的価値があることが確認できました。

ただし現時点で必要なのはReference-local stable keyであり、Canonical World上のProcedure identityではありません。

次に残る圧力は、Business Procedureをfile-native accepted authoring sourceへ進めた場合のreload / Source Index / Shared Acceptance、複数Subject Area / Subsystemを跨ぐhandoff、外部contractや長期reuseでCanonical Procedure subjectが必要になる具体例です。
