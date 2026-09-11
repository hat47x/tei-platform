# 000050 second realistic TEI-original Handoff corpus

Status: In Progress

## 目的

Contract ADR blocker Bを解消する。`order_entry`とは独立した二つ目のrealistic TEI-original corpusで、責任の受け渡し（Handoff）の`from` / `to` / `condition`がlosslessに保持できることを確認する。

一つのReferenceで成立した表現を、それだけでcontractへ昇格させないためのEvidenceである。

ただし本Issueの目的を「二つ目のHandoffを作ること」へ縮めない。**Handoffが存在しなくても独立したBusiness設計価値を持つseedを先に成立させ、そのseedからresponsibility transferが自然に生じた場合だけHandoffをauthorする。**

## 上位基線

- blocker定義: [`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](../BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) Remaining Contract ADR blockers
- readiness境界: [`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`](../BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md) §8-B
- 設計品質: [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- 十分性監査: [`BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md`](../BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md)
- corpus候補: [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](../LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md) / [`LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`](../LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md)
- 担当割り当て: [`UPSTREAM_BREAKDOWN_2026-09.md`](../UPSTREAM_BREAKDOWN_2026-09.md) §5
- 関連RQ: RQ-002 / RQ-003 / RQ-018 / RQ-021 / RQ-022 / RQ-025 / RQ-026

## 担当境界

| 所有する | 所有しない |
| --- | --- |
| 二つ目のcorpusの選定とBusiness seed受け入れ条件 | Handoff表現そのものの決定（[ISSUE-000048](000048-business-procedure-contract-adr-readiness.md)） |
| `from` / `to` / `condition`のlossless確認 | cross-view checkerの実装（[ISSUE-000042](000042-business-procedure-cross-view-reference.md)） |
| 推測で埋めないことの実証 | production採用（[ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)） |
| provenance-safe TEI-original seed取得計画 | external model / manualの再配布や互換実装 |

## 守る境界

二つ目のcorpusは「数を増やすため」の材料ではない。次を守れなければEvidenceにならない。

- exact identity / actor / sequenceを**発明しない**。元資料に無いものはunknownとして残す
- unsupported detailで空欄を埋め、false completenessを作らない
- physical ObservationからresponsibilityのHandoffを**生成しない**
- Step adjacency（隣接）からHandoffを**推測しない**
- Function sequence / state transition / actor changeからHandoffを**推測しない**
- explicit Handoff sourceだけが`H1` / `R3`をauthorする
- Handoff conditionと、その成立を支えるObservation / document等のEvidenceを同一視しない
- second-corpusのData authorityを**synthetic化しない**
- Referenceを動かせるように見せるためだけのApplication Pattern / Capabilityを作らない
- 出典と利用条件を記録する（public sourceを使う場合はライセンスと引用範囲を明示）

## corpus選定 — 2026-09-10

### 選定結果

二つ目のcorpusは、**受注生産を背景とする在庫・製造連携の小型TEI-original scenario**とする。

これはCONCEPTWARE / TALONのmodelやimplementationを複製するものではない。既存のpublic-source auditで確認済みの高水準な業務pressureだけをprovenance sourceとして使い、TEI自身のReferenceとして小さく再構成する。

`order_entry`とは、次の点で独立したpressureを持つ。

```text
order_entry
  sales / order interaction中心

second corpus
  manufacturing / inventory continuation
  physical completion boundary
  multiple responsibility context
  cross-subsystem Data-state pressure
  Application Command / entry result != physical-world completion
```

### provenance source

public-source auditで確認済みの次をdesign-pressure sourceとする。

- X-TEA HOME — `https://dbc.in.coocan.jp/`
- 渡辺幸三「実行可能な仕様書のライブラリを知的社会資本に」＠IT / ITmedia, 2011-09-28
- TALON「カスタマイズ放題生産管理」 — `https://talon.jp/products/businesspackage/businesspackage-mfs/`
- TALON「カスタマイズ放題生産管理システム Ver.1.3 マニュアル」 — `https://talon.jp/manual/10.TALON_MPS_Manual.pdf`
- TALON「想定業務フロー」 — `https://talon.jp/manual/20.TALON_MPS_BusinessFlow.pdf`
- TALON「カスタマイズ放題在庫管理」 — `https://talon.jp/products/businesspackage/businesspackage-ims/`
- TALON「モデリング分科会の発足」 — 2015
- XeadModeler GitHub repository — repository sourceについてBSD-3-Clause

利用条件は[`LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`](../LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md) §3 / §5を正とする。

特に次を固定する。

```text
XeadModeler repositoryのBSD-3-Clause
  != CONCEPTWARE model artifactのlicense

publicly readable TALON material
  != raw model / manual / implementationの再配布許諾
```

そのためTEI repositoryへ外部`.xeaf`、model file、TALON package、manual図表、画面、文章、function/table inventoryをコピーしない。

### 採用する高水準fact

public-source auditで確認済みの範囲だけをfactとして利用できる。

```text
cross-subsystem concernとして
  manufacturing instruction
  material issue / receipt
  warehouse stock movement / stocktaking
  shipment

physical / external-world pressureとして
  material receipt / issue
  manufacturing completion
  shipment

multiple responsibility contextが存在する
```

さらに公開manual / flowは、設計pressureとして次の分離が実在することを示す。

- instruction / entryの段階と、在庫や会計上のactual changeが同時とは限らない
- manufacturing instruction、physical manufacturing / completion、systemへのcompletion / receipt記録、shipment workは一つのsystem commandへ必ずしも畳めない
- inventoryではreceipt / issueというactual movementと、それに伴うhistory / stock stateを区別して扱う必要がある

これらは**どの意味を分離して検討すべきか**のprovenanceであり、個々のexact TEI Business Step、actor、sequence、Data identity、Handoffを直接authorする根拠ではない。

### TEI-original scenarioの対象範囲

最初からfull manufacturing systemを作らず、次の小さな範囲だけをauthoring対象候補とする。

```text
one manufacturing/inventory business need
  +
small set of Business Actions
  +
one manual / physical completion boundary
  +
one inventory-state continuation question
  +
only actually evidenced Application / Semantic refs
```

候補となるAction語彙は、既存auditのminimum pressure outlineから次を使える。

```text
determine required material
request / receive material
issue material to work
record manufacturing completion
move finished goods into stock
prepare / perform shipment
```

ただし、この並び自体をaccepted sequenceとは扱わない。TEI-original scenarioを構成する際のcandidate poolである。

## Evidence nature — current phase

[`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)に従い、現在の材料を次のように扱う。

| Nature | Current second corpusでの扱い |
| --- | --- |
| source-backed fact | 上記public sourceから高水準に確認できる業務concern / physical pressure / instructionとactualityの分離pressure |
| accepted TEI Business meaning | **まだ無い**。Gate A green後に独立Business seedからauthorする |
| derived cross-view fact | accepted seedが無いため、second corpusとしてはまだauthoritative projectionを作らない |
| proposal / candidate | TEI-original小型scenarioのbusiness need / Action composition / cross-view question |
| unresolved | exact actor、trigger、Step sequence、responsibility boundary、Handoff condition、Data identity、Application realization、actuality evidence |

特に`from` / `to` / `condition`は**まだunresolved**である。公開資料に製造・在庫の連続性が見えることだけからHandoffを作らない。

## Gate A green前に深めるcandidate design question

現段階ではaccepted Procedureをauthorせず、次の問いをpressureとして深める。

```text
manufacturing / inventory continuationで、
Business instruction、system entry / result、physical completion、
Business上の受入・次責任へのcontinuationは
どこまで同じで、どこから別のmeaningなのか。
```

この問いから次をreviewする。

1. instruction / entryが存在しても、physical inventory movementがまだ成立しない局面をTEIがfalse completionなしに表せるか。
2. manufacturing completionのsystem記録とphysical completionを無条件に同一視しないか。
3. physical completionが観測されても、それだけでshipment等へのresponsibility Handoffを生成しないか。
4. Business continuation conditionと、その成立を支えるObservation / document / accepted recordを区別できるか。
5. Data identity / Application realizationが未証明ならunknownのまま残せるか。
6. Business / Function / Dataの一視点だけでは見落とすcontradiction / unresolvedを作れるか。

この問いはHandoffの存在を前提にしない。

## Business Seed Acceptance Contract

Gate A green後にsecond corpusをaccepted sourceへ進める前に、次を満たすBusiness seedを先に成立させる。

### S1 — independent business need

Handoff fieldを検証したいから作るのではなく、Handoffを取り除いてもdomain上説明できるpurpose / problemを持つ。

### S2 — bounded scope

full manufacturing / inventory systemを再現しない。扱うbusiness need、扱わない隣接領域、外部境界を明示する。

### S3 — provenance-safe known facts

public sourceから確認したfactと、TEIが独自に採用するBusiness meaningを混ぜない。外部modelのactor / sequence / table / function inventoryを写さない。

### S4 — explicit unresolved

actor、trigger、sequence、Data、realization、actuality、responsibility boundary等の未確定点をseed authoring前に列挙する。unknownを埋めるための架空detailを作らない。

### S5 — actuality boundary

少なくとも一つ、system action / resultとmanual / physical / external actualityを同一視すると設計が壊れる箇所を持つ。ただし具体的なObservation identityを証拠なしに発明しない。

### S6 — meaningful cross-view question

Data / Function / Businessの少なくとも二視点から見ないと判断できない設計問題を持つ。単にProcedureの文章量を増やすseedは採らない。

### S7 — no forced Handoff

seedがHandoffなしでもReferenceとして価値を持つ。accepted Business meaningからresponsibility transferが自然に生じた場合だけHandoff candidateへ進む。

### S8 — no fake realization

Referenceを動かせるように見せるためだけのApplication Pattern / Capability / Data identityを作らない。existing TEI identityにexactに接続できるものだけref候補とする。

S1〜S8はproduction schema fieldではなく、second-corpus authoring前reviewの品質条件である。

## Responsibility transfer / Handoff受け入れ条件

Business seed成立後、次の3点が**同じaccepted Business change-setから整合的に説明できる場合だけ**Handoff candidateを受理する。一つのphysical fileや一つの文だけに同居することは要求しない。

```text
from
  transfer元のProcedure-scoped Stepをaccepted sourceから特定可能

to
  transfer先のProcedure-scoped Stepをaccepted sourceから特定可能

condition
  responsibility continuationがBusiness上成立するpredicateを
  accepted sourceからlosslessに保持可能
```

追加規則:

- `from/to`は「前後に並んでいるStep」だから選ばない
- actorが異なることだけをtransfer根拠にしない
- actorが同じことだけを理由にtransferを否定しない
- `condition`をtarget triggerへ暗黙同期しない
- `condition`をObservation ID / runtime event IDへ置き換えない
- 「前Stepが終わったら」のようなtautologyで十分かは、そのcompletion自体にindependent Business meaningがある場合だけ受理する
- system result / physical fact / state transitionだけからconditionをbackfillしない

一つでもaccepted sourceから確定できない場合はunresolvedを保持し、Step adjacency・物理Observation・Application execution順から補完しない。

## Authoring sequence — Gateではない

Corpus selection / provenance整理はaccepted Business meaningをまだauthorしていないresearch incrementなのでGate A actual green前でも行ってよい。

一方、`BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md`が固定した順序は維持する。

```text
provenance pressure selection
  ↓
non-authoritative Business seed proposal
  ↓ actual Repository Gate A green後
accepted independent Business need / seed
  ↓
minimal Procedure / Step accepted meaning
  ↓ responsibility transferが自然に成立した場合だけ
explicit Handoff candidate
  ↓
from / to / condition losslessness Pressure
  ↓
actual execution Evidence
```

Handoff candidateが自然に成立しない場合、seedを失敗扱いしてHandoffを捏造しない。「このseedはH1/R3 generality Evidenceにならない」というnegative resultを残し、別のindependent Business seedを探索する。

したがってGate A green前には次を行わない。

- candidate Action poolをaccepted Step sequenceへ昇格する
- exact actor / trigger / responsibility boundaryを補完する
- accepted Procedure / Step fixtureをsecond-case検証のために作る
- `from` / `to` / `condition`をauthorする
- candidate Data / Application identityをsyntheticに作る
- source/test integrationをBlocker Bのpositive Evidenceとして扱う

## このincrementで閉じるもの / 閉じないもの

閉じる:

```text
corpus selection
provenance source set
reuse / non-copying boundary
target scope
Evidence nature boundary
Business Seed Acceptance Contract
responsibility-transfer review rules
Gate A green前のcandidate design question
```

閉じない:

```text
accepted Business seed
accepted Procedure / Step
RRB-P2 source/test integration
from / to / condition losslessness
actual negative test
Handoff fail-closed actual Evidence
Contract ADR blocker B
```

したがって現時点はRRB-P1へ向けたsource / quality-contract preparationであり、`Reference breadth Evidence established`やContract ADR readyを主張しない。

## 完了条件

- [x] `order_entry`と独立したcorpusを1件選定し、出典・利用条件・対象範囲を記録した
- [x] accepted seed前の品質条件（S1〜S8）とEvidence natureを定義した
- [x] Gate A green前に検討可能なcandidate design questionを定義した
- [ ] actual Gate A green後に、Handoff検証とは独立したBusiness need / Reference seedを成立させた
- [ ] seedのknown / accepted / derived / proposal / unresolvedをreviewできる形で記録した
- [ ] `from` / `to` / `condition`がaccepted TEI-original sourceからlosslessに写せることを確認した
- [ ] 元資料に無いidentity / actor / sequenceを補完していないことを確認した
- [ ] system result / Observation / adjacency / actor差からHandoffを生成していないことをactual testで確認した
- [ ] Handoff sourceが無い箇所はunknownのまま保持され、fail closedになることを確認した
- [ ] ISSUE-000048のblocker Bへ、取得revisionとEvidence natureつきで結果を渡した

## 次のincrement

### Gate A green前

1. candidate design questionを、accepted meaningへ昇格させずにreviewする
2. public-source auditでinstruction / system entry / physical actualityの分離pressureに不足がないか確認する
3. S1〜S8で将来seedのacceptance review templateを準備する
4. candidate / unresolved分類を維持し、accepted Business sourceへ昇格しない
5. Gate A actual executionが可能になればISSUE-000048 Blocker Aを優先する

### Gate A green後

1. Handoff検証とは独立したsmall Business need / Reference seedをS1〜S8でreviewしてauthorする
2. seedだけからminimal Procedure / Step meaningを成立させる
3. source responsibility / target responsibility / continuation conditionをreviewする
4. responsibility transferが成立する場合だけexplicit Handoff candidateをauthorする
5. condition成立Evidenceが必要な場合、まず既存Source / Provenance / Evidence mechanismでtrace可能か確認する
6. 確定できない項目はunresolvedのまま保持するnegative PressureをISSUE-000042側へ渡す
7. source/test-level integration後、actual Repository runnerでsystem result / Observation / adjacency / actor差からHandoffを生成しないことを確認する
8. blocker Bを閉じられるEvidence natureに到達した時だけISSUE-000048へhandoffする

## Reopen trigger

second corpusで具体的lossが出た場合だけcurrent minimum contractをreopenする。

候補:

- `from/to/condition`だけではresponsibility transferをlosslessに保持できない
- actor / responsibility detailをstructuredにしないとmeaningが失われる
- Handoff conditionとEvidence traceを既存Provenanceで十分にreviewできない
- Business FlowをHandoff-only relationより広く持たないと重要な情報 / document / party flowが失われる
- current D4 / O1 ref separationではactual Business needを説明できない

外部手法にfieldがある、diagramを豊かにしたい、test fixtureを作りたい、という理由ではreopenしない。

## 非目標

- corpusのフルCRUD化・完全モデル化
- generic Role taxonomyやBusiness Flow surfaceの先行導入
- 二例あることをもって一般化の証明とすること
- external modelの再現・互換性実装
- Gate A green前のaccepted Business seed / Handoff authoring

## 関連

- [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- [ISSUE-000048](000048-business-procedure-contract-adr-readiness.md) / [ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md)
