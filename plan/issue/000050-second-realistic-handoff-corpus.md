# 000050 second realistic TEI-original Handoff corpus

Status: In Progress

## 目的

Contract ADR blocker Cを解消する。`order_entry`とは独立した二つ目のrealistic TEI-original corpusで、責任の受け渡し（Handoff）の`from` / `to` / `condition`がlosslessに保持できることを確認する。

一つのReferenceで成立した表現を、それだけでcontractへ昇格させないためのEvidenceである。

## 上位基線

- blocker定義: [`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`](../BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md) §8-C
- 十分性監査: [`BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md`](../BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md)
- corpus候補: [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](../LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md) / [`LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`](../LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md)
- 担当割り当て: [`UPSTREAM_BREAKDOWN_2026-09.md`](../UPSTREAM_BREAKDOWN_2026-09.md) §5
- 関連RQ: RQ-002 / RQ-003 / RQ-021 / RQ-022 / RQ-025 / RQ-026

## 担当境界

| 所有する | 所有しない |
| --- | --- |
| 二つ目のcorpusの選定と受け入れ条件 | Handoff表現そのものの決定（[ISSUE-000048](000048-business-procedure-contract-adr-readiness.md)） |
| `from` / `to` / `condition`のlossless確認 | cross-view checkerの実装（[ISSUE-000042](000042-business-procedure-cross-view-reference.md)） |
| 推測で埋めないことの実証 | production採用（[ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)） |

## 守る境界

二つ目のcorpusは「数を増やすため」の材料ではない。次を守れなければEvidenceにならない。

- exact identity / actor / sequenceを**発明しない**。元資料に無いものはunknownとして残す
- physical ObservationからresponsibilityのHandoffを**生成しない**
- Step adjacency（隣接）からHandoffを**推測しない**
- explicit Handoff sourceだけが`H1` / `R3`をauthorする
- second-corpusのData authorityを**synthetic化しない**
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
  Application Command result != physical-world completion
```

### provenance source

public-source auditで確認済みの次をdesign-pressure sourceとする。

- X-TEA HOME — `https://dbc.in.coocan.jp/`
- 渡辺幸三「実行可能な仕様書のライブラリを知的社会資本に」＠IT / ITmedia, 2011-09-28
- TALON「カスタマイズ放題生産管理」 — `https://talon.jp/products/businesspackage/businesspackage-mfs/`
- TALON「カスタマイズ放題生産管理システム Ver.1.3 マニュアル」
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

これらはsecond corpusのpressureを選ぶ根拠であり、個々のexact Business StepやHandoffを直接authorする根拠ではない。

### TEI-original scenarioの対象範囲

最初からfull manufacturing systemを作らず、次の小さな範囲だけをauthoring対象とする。

```text
one manufacturing/inventory business event
  +
small chain of Business Actions
  +
one manual / physical completion boundary
  +
one inventory-state continuation
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

### fact / candidate / unresolved

現段階では次の分類を固定する。

| 区分 | 扱い |
| --- | --- |
| fact | 上記public sourceから高水準に確認できる業務concern / physical pressure |
| candidate | TEI-original小型scenarioのBusiness event / Action composition |
| unresolved | exact actor、trigger、Step sequence、responsibility boundary、Handoff condition、Data identity、Application realization |

特に`from` / `to` / `condition`は**まだunresolved**である。公開資料に製造・在庫の連続性が見えることだけからHandoffを作らない。

### Handoff受け入れ条件

次の3点が同じexplicit source fragmentまたは十分に同一responsibility transferを示すTEI-original accepted sourceから取得できた場合だけ、Handoff candidateを受理する。

```text
from
  transfer元のProcedure-scoped Stepをsourceから特定可能

to
  transfer先のProcedure-scoped Stepをsourceから特定可能

condition
  transferが成立する条件をsourceから保持可能
```

一つでもsourceから確定できない場合はunknownを保持し、Step adjacency・物理Observation・Application execution順から補完しない。

### このincrementで閉じるもの / 閉じないもの

閉じる:

```text
corpus selection
provenance source set
reuse / non-copying boundary
target scope
fact / candidate / unresolved boundary
```

閉じない:

```text
RRB-P2 source/test integration
from / to / condition losslessness
actual negative test
Handoff fail-closed Evidence
Contract ADR blocker C
```

したがって本incrementはRRB-P1へ向けたsource selectionであり、`Reference breadth Evidence established`やContract ADR readyを主張しない。

## 完了条件

- [x] `order_entry`と独立したcorpusを1件選定し、出典・利用条件・対象範囲を記録した
- [ ] `from` / `to` / `condition`が元資料からlosslessに写せることを確認した
- [ ] 元資料に無いidentity / actor / sequenceを補完していないことを確認した
- [ ] Observationや隣接関係からHandoffを生成していないことをactual testで確認した
- [ ] Handoff sourceが無い箇所はunknownのまま保持され、fail closedになることを確認した
- [ ] ISSUE-000048のblocker Cへ、取得revisionとEvidence natureつきで結果を渡した

## 次のincrement

1. 上記sourceから、explicit responsibility transferとして引用せずparaphraseできる最小fragmentを選ぶ
2. `from` / `to` / `condition`のうちsourceで確定できるものだけをTEI-original accepted sourceへauthorする
3. 確定できない項目はunknownのまま保持するnegative PressureをISSUE-000042側へ渡す
4. source/test-level integration後、actual Repository runnerでObservation / adjacencyからHandoffを生成しないことを確認する
5. blocker Cを閉じられるEvidence natureに到達した時だけISSUE-000048へhandoffする

## 非目標

- corpusのフルCRUD化・完全モデル化
- generic Role taxonomyやBusiness Flow surfaceの導入
- 二例あることをもって一般化の証明とすること

## 関連

- [ISSUE-000048](000048-business-procedure-contract-adr-readiness.md) / [ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md)
