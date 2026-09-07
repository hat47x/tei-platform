# Business Procedure Second-Corpus Data Authority Deferral — 2026-09

Status: **Lane C / evidence boundary / no synthetic Data authority / no production schema adoption**

## 1. 目的

manufacturing / inventory second-corpus outlineには、physical-world stateを扱う必要性が強く現れる。

一方で、現時点のprovenance-backed outlineにはTEI-originalのexact identityとして次がまだない。

```text
Data
Relation
Database
Datasource
Application Pattern
Query / Contract Query
Capability
ownership boundary
```

この状態で「在庫はexternal-observed」「生産完了はmanaged-here」等をTEIのaccepted authorityとして作ると、公開sourceから確認できる高水準factを超えて設計者側が正本を発明する。

本書は、その先走りを禁止するためのdeferral recordである。

## 2. Current order_entryから既に証明されていること

Current Referenceではexact normalized World identityがあるため、次をcross-viewで検査できる。

```text
sales_core
  ownership = managed-here

order_entry / datasource sales
  access = read-write

master_data
  ownership = external-observed

order_entry / datasource master
  access = read

order_entry.customer_reference
  -> master datasource
  -> Contract Query
```

このためcustomer masterへwrite intentを持ち込めば、Business / Function probeとaccepted datasource policyの矛盾として検出できる。

ここで重要なのは、Business Procedure sourceがownershipやdatasource accessを複製していないことである。

## 3. Second corpusへそのまま移植しないもの

Manufacturing / inventory outlineに次を機械的にコピーしない。

```text
physical stock -> external-observed
manufacturing completion -> managed-here
shipment state -> external-observed
warehouse -> one Database
subsystem -> ownership boundary
```

これらはもっともらしくても、current outlineからaccepted TEI authorityとしては導けない。

X-TEAのSubsystemもdeployment / Databaseと一対一ではないため、歴史的namingからownershipを推測しない。

## 4. Reopen条件

Second corpus Data authority pressureをcode / fixtureへ進めるのは、TEI-original source上で少なくとも次がexactに与えられた場合だけとする。

1. Business Actionが依存するbusiness factが明示される
2. そのfactを読む／変更するApplication Patternまたはexplicit semantic authorityがある
3. Pattern owner Applicationをnormalized Worldからexact解決できる
4. datasource / target Databaseをtraceできる
5. Database ownershipとApplication accessがaccepted sourceから得られる
6. Business-side要求とaccepted authorityを比較できる

そのとき初めて次のようなdiagnostic pressureを作る。

```text
Business depends on externally observed state
  but Function attempts managed-here write

or

Business requires managed-here mutation
  but Application only has external-observed/read authority
```

## 5. D4との関係

Second corpus outlineだけではdirect Data / Relation refのminimum必要性も証明されない。

したがってD4は現在のまま:

```text
businessAction
applicationPatternRefs[]
validationRefs[]
capabilityRefs[]
```

を維持する。

将来、PatternやCapabilityからlosslessにderiveできないaccepted Business dependencyとしてData / Relation identityが必要になった場合は、D4が黙ってdropせずfail closedしexplicit-ref designをreopenする。

## 6. Physical Observationとの関係

`BUSINESS_PROCEDURE_PHYSICAL_OBSERVATION_PRESSURE_2026-09.md`は、Data authorityを決めずに:

```text
Command accepted != physical fact observed
```

だけをtest-onlyでPressureする。

これは意図的な段階分離である。

```text
Observation boundaryを守る
  !=
physical factのCanonical Data authorityを決める
```

## 7. Handoffとの関係

同様に、physical factが存在／観測されたことだけからH1/R3 relationを作らない。

```text
physical fact observed
  != responsibility transfer accepted
```

Handoffにはexact Procedure / Step endpointとaccepted conditionが別途必要である。

## 8. 現時点の判断

```text
Data authority relevance
  strong

exact second-corpus TEI authority
  absent

synthetic ownership assignment
  prohibited

D4 Data/Relation expansion
  not justified

Gate 15c
  not activated
```

これはEvidence不足を埋めるための保留ではなく、P11の「unresolved remains unresolved」とX-TEA fallbackの「historical nounをTEI nounへ一対一変換しない」を守る設計判断である。
