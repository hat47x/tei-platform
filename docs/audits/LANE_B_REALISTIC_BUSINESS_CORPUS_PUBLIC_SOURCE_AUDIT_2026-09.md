# Lane B — Realistic Business Corpus Public Source Audit — 2026-09

Status: **Lane B / public provenance source audit / no copied external artifact / no new Requirement / PAS / Gate / production schema**

## 1. 目的

B-F19 `provenance-backed realistic cross-subject-area Business corpus breadth`について、current TEI repository内に第二Business corpusがないことと、外部に再利用可能なdesign-pressure sourceがないことを分ける。

本監査は外部Reference artifactの転載許諾を認定するものではない。

## 2. 公開sourceから確認できること

### X-TEA / CONCEPTWARE

X-TEA公式ポータルはCONCEPTWAREを、複数の業種・業態別Reference Model群として位置づけ、一部に実動実装があると説明している。

公開イベント資料でも、実業務で使えるReference Modelを複数業務で公開していたことが確認できる。

また渡辺幸三氏自身の解説では、CONCEPTWARE／販売管理を「本格的な業務システムのレファレンスモデル」としてDriver実装のdesign pressureに使った経緯が示されている。

### CONCEPTWARE／受注生産 → TALON再現

TALONはCONCEPTWARE／受注生産を忠実に再現した生産管理Referenceを公開している。

公開マニュアル / 製品説明から少なくとも次のpressureが確認できる。

```text
subsystems
  system administration
  master management
  purchasing
  manufacturing
  inventory
  sales
  accounting
  statistics

cross-subsystem business concerns
  order / manufacturing instruction
  material issue
  purchase / receipt
  warehouse stock / transfer / stocktaking
  shipment
  accounts payable / receivable
  payment approval / receipt
  statistical aggregation

responsibility context examples
  system administrator
  purchasing staff
  accounting approval

physical / external-world pressure
  material receipt / issue
  manufacturing completion
  shipment
  warehouse stock movement / stocktaking
```

TALON公開説明では150機能規模で、画面・帳票・batch・common logicを含む実装として説明されている。

したがってcurrent `order_entry`より広い:

```text
multiple subsystem / responsibility context
manual or physical-world continuation
Data-state across subsystem boundaries
Function / batch / report variety
business handoff / continuation pressure
```

を検討するprovenance sourceとして十分に具体的である。

## 3. ライセンス / provenance境界

### XeadModeler source license

XeadModeler GitHub repositoryはBSD-3-Clauseで公開されている。

ただしこのlicenseはModeler source repositoryについて確認できるものであり、CONCEPTWAREの各model artifactへ自動的に同じlicenseが適用されるとは扱わない。

### TALON implementation

TALONの生産管理packageは無償公開・customize可能と説明されている一方、利用にはTALON本体および利用規約への同意が必要と明記されている。

またTALON版はCONCEPTWARE／受注生産の一部機能を統廃合し、仕様にも変更があると明記されている。

したがって:

```text
publicly readable / free to use in stated environment
  !=
TEI repository may copy / redistribute raw model or implementation

TALON faithful reproduction
  !=
bit-identical original CONCEPTWARE
```

を固定する。

## 4. B-F31 — external realistic source availabilityとTEI-native corpusを分離

### 判定

B-F19を次へ細分化する。

```text
B-F19a external provenance-backed realistic source availability
  Resolved

B-F19b TEI-native provenance-backed realistic Reference corpus breadth
  Unresolved
```

外部にrealistic pressure sourceがないわけではない。

一方、TEI repositoryにはそのsourceを根拠としつつ、TEI自身のauthoring / Reference / checkerで再現した第二Business corpusはまだない。

## 5. 安全な利用方法

現時点では外部artifactをTEI repositoryへコピーしない。

避ける:

- CONCEPTWARE `.xeaf` / database / model fileの無断転載
- TALON package / implementation filesの転載
- manual図表・画面・文章の実質的複製
- 外部modelのtable / function inventory全体をTEI Referenceとして再配布
- 「公開されているからOSS」とする推定

可能なpressure利用:

1. 公開資料から確認できる高水準の業務事実をprovenance付きで列挙する。
2. fact / candidate / unresolvedを分ける。
3. 外部modelの構造を逐語転写せず、TEI独自の小型Reference scenarioを作る。
4. TEIで検証したいsemantic boundaryだけを抽出する。
5. source URL / publication / access dateをEvidence registerへ残す。
6. copied contentが必要になった場合は、そのartifact固有のlicense / permissionを別途確認する。

## 6. 第二Reference候補としてのpressure shape

`order_entry`と単にApplication数を増やすためではなく、B-F19bを閉じる候補としては受注生産 / 在庫・製造連携が強い。

理由:

```text
existing order_entry
  sales/order interaction中心

manufacturing / inventory pressure
  physical movement
  multiple responsibility contexts
  read/write ownership across subsystems
  order -> material -> manufacturing -> inventory -> shipment continuation
  approval / accounting continuation
```

を追加できるためである。

ただしTEI版scenarioは外部modelの再現を目的としない。

### 最小pressure候補

```text
Business event
  accepted production demand / manufacturing need

Business actions
  determine required material
  request / receive material
  issue material to work
  record manufacturing completion
  move finished goods into stock
  prepare / perform shipment

cross-view questions
  which Data state is authoritative here?
  which Application interaction realizes only some actions?
  where is physical/manual completion outside Application?
  where does read-only / managed-here ownership change?
  which handoff is Business meaning but not Runtime workflow?
  which Observation is external-world evidence rather than Command result?
```

上記はTEIのpressure outlineであり、CONCEPTWARE / TALONの具体的手順を転記したものではない。

## 7. B-F19bを閉じるEvidence条件

第二TEI-native corpusは少なくとも次を満たす。

```text
provenance
  public source / user source / domain sourceが明示される

independence
  current order_entryの単なるrename / copyではない

breadth
  multiple responsibility or subsystem context
  manual / physical / external action
  multiple Data-state ownership
  Application interaction where actually present
  cross-view contradiction opportunity

uncertainty
  sourceで確認できないactor / trigger / sequence / ruleをcandidate / unresolvedで保持

non-copying
  external artifactの表現・inventoryをそのまま転載しない

TEI-native
  authoritative Business meaning / explicit refs / ProjectionをTEI自身のReference contractで表現
```

actual runner greenとは別のEvidence axisである。

## 8. 次

1. `CONCEPTWARE／受注生産` / TALON公開資料をB-F19bのprovenance source候補としてEvidence registerへ登録する。
2. 外部artifactをcopyせず、上記minimum pressureからTEI-original第二Business Reference outlineを作る。
3. fact / candidate / unresolvedを明記し、実業務truthを捏造しない。
4. D3 / D4 / O1 / H1 / R3がこのsecond pressureでも崩れないか比較する。
5. このsecond corpusが実際に新しいmeaningを要求した場合だけschema / ref kind / role taxonomyをreopenする。
6. B-F19bが閉じるまで「realistic Reference maturity / breadth established」と主張しない。

## 9. Public provenance references

- X-TEA HOME — `https://dbc.in.coocan.jp/`
- 渡辺幸三「実行可能な仕様書のライブラリを知的社会資本に」＠IT / ITmedia, 2011-09-28
- TALON「カスタマイズ放題生産管理」 — `https://talon.jp/products/businesspackage/businesspackage-mfs/`
- TALON「カスタマイズ放題生産管理システム Ver.1.3 マニュアル」
- TALON「モデリング分科会の発足」 — 2015
- XeadModeler GitHub repository — BSD-3-Clause for repository source

## 10. 境界

- external artifact importなし
- license推定なし
- Product Requirement / PAS / E7 / Gate追加なし
- production Canonical schema採用なし
- B-F19b未解消
- actual Repository execution Evidenceとは別管理
