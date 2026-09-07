# Business Procedure Manufacturing / Inventory Corpus Pressure Outline — 2026-09

Status: **Lane C / provenance-backed TEI-original pressure outline / no Reference fixture / Gate 15c not activated / no production schema adoption**

## 1. 目的

B-F19b `TEI-native provenance-backed realistic Reference corpus breadth`を将来閉じるため、外部Reference artifactをコピーせず、公開sourceから確認できる高水準factだけを起点にTEI-original second corpusのpressure outlineを作る。

本書はReference fixtureではない。

```text
public provenance source
  -> high-level business facts / design pressure
  -> TEI-original outline
  != copied external model
  != accepted Business truth
  != Gate 15c activation
```

Current `order_entry`へApplication数だけを足すことを目的にしない。

## 2. Provenance source

Access / verification date: **2026-09-07**

### X-TEA HOME

- https://dbc.in.coocan.jp/
- X-TEA / Three-Element Analysis portal
- CONCEPTWAREを業種・業態別のReference Model群として説明
- 一部modelには実動実装があると説明

### Kozo Watanabe / ITmedia

- 「“実行可能な仕様書のライブラリ”を知的社会資本に」, 2011-09-28
- https://www.itmedia.co.jp/im/articles/1109/28/news134_3.html
- `CONCEPTWARE／販売管理`をDriver実装のpressureとして用いたこと、Reference Modelと実装が相互に改善を促す考えを確認

### TALON — カスタマイズ放題生産管理

- https://talon.jp/products/businesspackage/businesspackage-mfs/
- https://talon.jp/manual/10.TALON_MPS_Manual.pdf

公開ページ / Ver.1.3 manualから確認できる高水準fact:

```text
CONCEPTWARE／受注生産をTALONで再現したsystem
150 functions on product page

8 subsystem contexts in manual
  system administration
  master management
  purchasing
  manufacturing
  inventory
  sales
  accounting
  statistics

business / physical-world pressure examples
  purchase / receipt
  material issue
  manufacturing instruction / completion
  warehouse stock / transfer
  stocktaking
  shipment
  accounts payable / receivable continuation
  payment approval / receipt
```

Manualは同時に、TALON版でCONCEPTWAREの一部機能を統廃合し、仕様にも変更があることを明記している。

したがってTALON版をoriginal CONCEPTWAREとbit-identicalなsourceとは扱わない。

## 3. License / copying boundary

確認済み:

- `xead/XeadModeler` repository sourceはBSD-3-Clause
- そのlicenseをCONCEPTWARE model artifactやTALON package / manualへ自動適用しない
- TALON manualは利用規約への同意が必要であることを明記

本outlineでは外部artifactをRepositoryへ取り込まない。

避ける:

- `.xeaf` / DB / package / manual図表の転載
- external table / function inventoryの逐語転写
- screen / flow図の再配布
- 「free / public = OSS / redistributable」という推定
- TALON固有の実装shapeをTEI Canonical shapeへ移植

## 4. Fact / candidate / unresolvedの分離

### Fact-level pressureとして利用可能

公開sourceから確認できるのは少なくとも次である。

```text
multiple subsystem contexts exist
multiple responsibility contexts exist
purchasing / manufacturing / inventory / sales / accounting are separated concerns
material receipt / issue and manufacturing completion affect physical world
inventory transfer / stocktaking affect warehouse state
shipment continues beyond a software command acknowledgement
```

### TEI candidate — sourceから逐語導出しない

次はTEIが比較用に置くcandidateであり、外部sourceの正確なProcedure sequenceを主張しない。

```text
Candidate Business context
  manufacturing / inventory continuation

Candidate business need
  accepted demand requires material and physical completion before shipment can be considered complete

Candidate ordered pressure
  determine required material
  request or receive material
  issue material to work
  recognize manufacturing completion
  recognize finished-goods stock
  prepare / perform shipment
```

これらの順序・actor・trigger・conditionを外部sourceの確定事実として扱わない。

### Unresolved

```text
exact actor / responsibility for each action
exact trigger identity
exact order among exceptional paths
approval rules
material shortage policy
inventory valuation / accounting rule
exact Application Function realization
exact Data ownership split in TEI
exact Observation contract
```

sourceで確定できないものを、もっともらしさだけでacceptedへ丸めない。

## 5. Current candidate stackへ当てる問い

### D3 / D4 — residual Business meaning

このpressureで次が必要になるかを確認する。

- Business Action wordingだけでmanual / physical workを保持できるか
- current D4のApplication Pattern / Validation / Capability refsだけでminimum bodyが足りるか
- Patternから独立したdirect Data / Relation refが本当に必要になるか

もしdirect Data / Relationがaccepted Business meaningとして不可欠ならD4をfail closedしてreopenする。

### O1 — ref role

Authorizationだけでなく、manufacturing / inventory側で:

```text
same authority kind
  but target / concern != direct realization
```

が反復するかを確認する。

反復しなければgeneric Role taxonomyを作らない。

### H1 / R3 — Business Handoff

次のようなrelation-only meaningがProcedure bodyと独立して必要かを確認する。

```text
material available / physical completion confirmed
  -> next Business responsibility may continue
```

ただしRuntime Workflow transitionへ変換しない。

### Observation boundary

特に次をPressureする。

```text
application command accepted
  != material physically received
  != manufacturing physically completed
  != stock physically present
  != shipment physically completed
```

Command resultからexternal-world completionを推測しない。

### Business Flow / DFD

Procedure/Handoffだけでは情報・物・document movementのaccepted meaningを失う具体例が出た場合だけT-GAP-06をreopenする。

外部sourceにBusiness Flowがあることだけを理由にfull Business Flow schemaを導入しない。

## 6. Cross-view contradiction opportunities

Second corpusが価値を持つにはhappy-path説明だけでなく、少なくとも次の矛盾を作れる余地が必要である。

```text
Business says physical completion is required
  but Function only records a command / acknowledgement

Business needs external-observed state
  but Application assumes managed-here write authority

Business handoff depends on actual stock / completion
  but source only preserves screen adjacency

Business action is manual / physical
  but projection invents a Capability to make it executable

Business meaning needs a direct authority ref
  but D4 would silently drop it
```

これらを将来TEI-native fixtureへ落とす際はintentional defectとbaselineを分ける。

## 7. B-F19bを閉じる条件

このoutlineだけではB-F19bを閉じない。

少なくとも:

1. provenance sourceを明示する
2. external artifactをコピーしないTEI-original authoring sourceを作る
3. fact / candidate / unresolvedをsource上でも区別する
4. `order_entry`のrename/copyではない独立meaningを持つ
5. multiple responsibility / subsystem contextをPressureする
6. manual / physical / external actionをfake Functionなしで保持する
7. Data-state authority / ownershipをcross-viewで検査できる
8. Command / Observation separationをphysical-world caseで検査する
9. D3 / D4 / O1 / H1 / R3の不足があれば具体的に記録する
10. actual Repository runnerでRelevant testsを実行する

ことを必要とする。

## 8. Gate 15との関係

```text
B-F19b breadth work
  != Gate 15c automatically open
```

このoutlineからD3 / D4 / O1 / H1 / R3の具体的contract insufficiencyが見つかるまでは15cを開かない。

一方、B-F19bはReference maturity / breadth claimとして独立に未解消のままである。

```text
scoped minimum contract pressure may become sufficient
  while
realistic Reference maturity remains unresolved
```

を許容する。

## 9. 次の作業

1. actual Repository execution経路が戻れば、まずcurrent #482 pressure testsをgreenにする。
2. actual green前に本outlineをfixture schemaへ昇格しない。
3. actual green後、B-F19bを進める価値がある場合だけ小さなTEI-original sourceを作る。
4. 最初のsourceではfact / candidate / unresolvedを明示し、外部modelのexact sequenceを再現しない。
5. 新しい意味不足が出た場合だけD4 ref kind / O1 role / H1-R3 / T-GAP-06をreopenする。

## 10. 非対象

- external Reference artifact import
- CONCEPTWARE / TALON reproduction
- external license inference
- production Canonical schema adoption
- Gate 15c activation
- generic ref-role taxonomy adoption
- full Business Flow / DFD adoption
- Gate B / Gate C Required Evidence change
