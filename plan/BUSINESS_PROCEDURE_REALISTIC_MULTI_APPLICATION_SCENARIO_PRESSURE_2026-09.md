# Business Procedure Realistic Multi-Application Scenario Pressure — 2026-09

Status: **Lane C / intermediate test-only Reference pressure / no production Reference expansion**

## 1. 目的

既存multi-Application pressureは、qualified Pattern identityが一つのProcedureから複数Applicationへ跨げることを確認する構造試験だった。

Pre-ADR Gateではさらに、Business Procedureをsocial-system / business-manual viewpointとして見たときに、次が同じ手順の中で共存できるかを確認する。

```text
Application A interaction
Application B interaction
manual work
external / physical work
```

production `reference/workspace`へ第二Applicationを追加せず、temp Workspaceで中間Pressureをかける。

## 2. scenario

既存`order_entry` Business Procedureへ、test-onlyで次を続ける。

```text
accepted order interaction
  ↓
review whether the accepted order can proceed through fulfillment
  -> fulfillment_console.order_record
  ↓
coordinate physical preparation of the order with warehouse staff
  -> manual / external
  ↓
continue shipment-related work with the external delivery party when required
  -> manual / external
```

このscenarioはproduction業務仕様ではない。

Business Procedure contractがApplication境界とphysical/manual workを跨げるかを見るpressure fixtureである。

## 3. second Application

第二Applicationは既存multi-Application pressureと同じ最小形を使う。

```text
application fulfillment_console {
}

pattern fulfillment_console.order_record record {
  data sales.order
}
```

Database / Contract / datasource / Provider設定は持たせない。

このscenarioで確認したいのはBusiness Procedureの意味境界であり、Gate B realization readinessではないためである。

## 4. Pressure

`tools/reference-business-procedure-check/realistic_multi_application_pressure_test.go`で次を確認する。

1. `order_entry`と`fulfillment_console`のqualified Pattern refsが一つのProcedureに共存する。
2. manual warehouse coordinationをfake Application Function / Capabilityなしで保持する。
3. external delivery continuationをfake Function / Capabilityなしで保持する。
4. thin realization-subtracted meaningでも上記Business Actionを失わない。
5. Procedureにmandatoryな単一Application ownerを作らない。
6. Procedureが関係するApplication集合はqualified Pattern refsからderivedできる。
7. Business actor candidateをAuthorization identityへ変換しない。

actual Repository runnerで実行されるまではgreen Evidenceではない。

## 5. Watanabe pressureとの関係

Business Modelを単なる「画面操作順」へ縮めず、manual / physical / external partyを含む業務マニュアルとして扱う。

同時にFunction-Interaction viewpointはApplication Pattern refsで明示し、Data viewpointはPatternから既存Canonical Dataへtraceする。

```text
Business
  manual / external / purpose / actor / trigger
        ↕ explicit cross-check
Function-Interaction
  Application Pattern
        ↕ derived trace
Data
  Canonical Data / Relation
```

三視点を一つの巨大schemaへ融合するのではなく、それぞれの不足が互いを批判できる構造を保つ。

## 6. まだ「realistic Reference完成」とはみなさない

このPressureは一段現実的だが、第二Applicationはsyntheticであり、domain expertsとのreviewを経た業務仕様でもない。

したがってPre-ADR Gateの「current order_entryより一段現実的なmulti-Application / manual / external Reference pressure」への**中間Evidence**とする。

production Canonical contract採用の最終根拠にはしない。

## 7. 次の判断

この中間Pressureでthin candidateが崩れないことを確認した後、次のどちらかを選ぶ。

1. `order_entry`周辺にもう一つ現実的なReference application / subject areaを設計し、より本格的なBusiness Procedure pressureへ進む。
2. 既存Reference corpusの別scenarioで同じcross-Application / manual / external圧力が得られるなら、それを再利用する。

新しいproduction Referenceを増やすこと自体を目的化しない。
