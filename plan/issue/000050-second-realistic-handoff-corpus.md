# 000050 second realistic TEI-original Handoff corpus

- 状態: In Progress

## 目的

Contract ADR blocker Bを解消する。`order_entry`とは独立した二つ目のrealistic TEI-original corpusで、責任の受け渡し（Handoff）の`from` / `to` / `condition`がlosslessに保持できることを確認する。

一つのReferenceで成立した表現を、それだけでcontractへ昇格させないためのEvidenceである。

ただし本Issueの目的を「二つ目のHandoffを作ること」へ縮めない。**Handoffが存在しなくても独立したBusiness設計価値を持つseedを先に成立させ、そのseedからresponsibility transferが自然に生じた場合だけHandoffをauthorする。**

## 上位基線

- blocker定義: [`BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md`](../BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) Remaining Contract ADR blockers
- readiness境界: [`BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md`](../BUSINESS_PROCEDURE_ADR_VS_PRODUCTION_ADOPTION_READINESS_SPLIT_2026-09.md) §8-B
- 設計品質: [`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`](../BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)
- 第二Reference受入条件の正本: [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](../LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)
- 十分性監査: [`BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md`](../BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md)
- corpus provenance: [`LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`](../LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md)
- 担当割り当て: [`UPSTREAM_BREAKDOWN_2026-09.md`](../UPSTREAM_BREAKDOWN_2026-09.md) §5
- 関連RQ: RQ-002 / RQ-003 / RQ-018 / RQ-021 / RQ-022 / RQ-025 / RQ-026

## 担当境界

| 所有する | 所有しない |
| --- | --- |
| 二つ目のcorpusの選定と取得作業 | 第二Reference acceptance / maturity条件の正本（Lane B Acceptance Contract） |
| accepted seed成立後の`from` / `to` / `condition` lossless確認 | Handoff表現そのものの決定（[ISSUE-000048](000048-business-procedure-contract-adr-readiness.md)） |
| 推測で埋めないことの具体的適用 | cross-view checkerの実装（[ISSUE-000042](000042-business-procedure-cross-view-reference.md)） |
| provenance-safe TEI-original seedのauthoring計画 | production採用（[ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)） |

本IssueはRRB-P1 / P2 / P3を再定義しない。statusとacceptance条件はLane B Acceptance Contractを正とする。

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

これらはDesign Quality ModelとLane B Acceptance Contractの具体適用であり、本Issue固有の第二contractではない。

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

さらに公開manual / flowは、design pressureとして次の分離が実在することを示す。

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
| accepted TEI Business meaning | **まだ無い**。Gate A green後にRRB-P1を満たすindependent Business seedとしてauthorする |
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

### R2 negative proofで確認済みの範囲 — 2026-09

PR #596 / `DESIGN_SYNTHESIS_MANUFACTURING_ACTUALITY_HANDOFF_PROOF_REFERENCE_2026-09.md`で、上記1〜4・6のうちauthority短絡に関する主要部分をnegative proofとして確認した。

```text
Command accepted -> physical completion
  = contradiction as derivation rule

Observation execution success -> physical completion
  = contradiction as derivation rule

explicit external Observation Result=true
  -> observed physical fact only
  = supported at that limited scope

observed physical fact -> Business acceptance / continuation
  = unresolved

Step adjacency / actor change / Function sequence / positive Observation
  -> Handoff
  = contradiction as derivation rule

second-corpus Handoff existence / from / to / condition
  = unresolved
```

この結果により、Gate A green前のcandidate-design researchで同型の「もっともらしいHandoff短絡」を追加し続ける必要はない。

```text
R2 negative proof
  -> non-inference / actuality boundaryを強化
  -> false candidateを縮約

R2 negative proof
  != accepted Business seed
  != positive Handoff Evidence
  != RRB-P1 / P2 / P3
```

今後Gate A前にnegative caseを追加するのは、既存R2では説明できない**具体的なprovenance gap / contradiction class**が見つかった場合だけとする。

## RRB-P1 acceptanceの本Issueへの適用

第二Reference acceptance条件の正本は`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`であり、本Issueでは再定義しない。

製造・在庫候補へ適用する際は、特に次を確認する。

```text
independent Business need
  Handoff testを除いても価値がある

bounded scope
  full manufacturing systemを再現しない

provenance / non-copying
  external factとTEI-original decisionを混ぜない

explicit unresolved
  actor / trigger / sequence / Data / realization / actuality / responsibilityを捏造しない

actuality boundary
  system resultとmanual / physical factを短絡すると意味が壊れるpressureがある

meaningful cross-view question
  Data / Function / Businessを相互に見ないと判断できない

no forced Handoff / no fake realization
  responsibility transferやApplication refをtest都合で作らない
```

これらはRRB-P1の本Issue固有の読み方であり、独立したS1〜S8 contractや新Gateを作らない。

### Gate A green後のRRB-P1 seed review checklist — 事前準備

このchecklistはauthoring内容を先に決めるものではない。Gate A green後にseed candidateをreviewするとき、false completenessやHandoff test由来の逆算を防ぐための**review順序**だけを準備する。

1. **Independent value**
   - Handoff検証を完全に外しても、このBusiness needをReferenceへ置く価値があるか。
2. **Provenance**
   - source-backed factとTEI-original design decisionを別々に示せるか。
   - 外部model / manualの構造や語彙を無断で複製していないか。
3. **Bounded scope**
   - full manufacturing / inventory systemへ膨張せず、一つのBusiness needとして閉じているか。
4. **Meaning / unknown ledger**
   - known / accepted / derived / proposal / unresolvedを区別できるか。
   - actor / trigger / sequence / Data / realizationを「埋めるため」に発明していないか。
5. **Actuality boundary**
   - instruction / system result / physical fact / Business acceptanceを短絡していないか。
6. **Cross-view value**
   - Business / Function / Dataの複数viewを照合しないと見えない意味上の問いがあるか。
7. **No fake realization**
   - Application Pattern / Capability / Data identityをReferenceを動かすためだけに作っていないか。
8. **No forced Handoff**
   - seed単独で成立した後に責任移転が自然に現れるか。
   - 現れなければHandoffを作らずnegative resultとして受理できるか。

Gate A green前には、この各欄の**回答内容を固定しない**。将来candidateをreviewするための観点だけを固定する。

## Responsibility transfer / Handoff受け入れ条件

RRB-P1 source成立後、responsibility transferが自然に存在する場合だけHandoff candidateを検討する。

次の3点が**同じaccepted Business change-setから整合的に説明できる場合**に、current H1/R3のpositive pressureへ進む。一つのphysical fileや一つの文だけに同居することは要求しない。

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

Handoffが自然に存在しない場合、そのseedはH1/R3 generality Evidenceにはならない。seedを変形してHandoffを作らずnegative resultとして記録する。

## Authoring sequence — Gateではない

Corpus selection / provenance整理はaccepted Business meaningをまだauthorしていないresearch incrementなのでGate A actual green前でも行ってよい。

一方、`BUSINESS_HANDOFF_SECOND_CASE_ADR_EVIDENCE_SUFFICIENCY_AUDIT_2026-09.md`とISSUE-000048が固定した順序は維持する。

```text
provenance pressure selection
  ↓
non-authoritative Business seed proposal / design question
  ↓ actual Repository Gate A green後
RRB-P1を満たすaccepted independent Business source
  ↓
minimal Procedure / Step accepted meaning
  ↓ responsibility transferが自然に成立した場合だけ
explicit Handoff candidate
  ↓
from / to / condition losslessness Pressure
  ↓
RRB-P2 source/test integration
  ↓
actual execution Evidence / RRB-P3
```

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
candidate design questionの主要negative proof
RRB-P1 acceptance criteriaの製造・在庫候補への適用方法
RRB-P1 seed review checklistのreview順序
responsibility-transfer review rules
```

閉じない:

```text
RRB-P1 accepted Business source
accepted Procedure / Step
RRB-P2 source/test integration
from / to / condition losslessness
actual negative test
Handoff fail-closed actual Evidence
RRB-P3
Contract ADR blocker B
```

したがって現時点はRRB-P1へ向けたsource / review preparationであり、`Reference breadth Evidence established`やContract ADR readyを主張しない。

（2026-09-21: この節は2026-09-10時点のincrementの記録である。「閉じない」に挙げた項目は、下の「実施記録」のincrementで扱った。）

## 実施記録 — 2026-09-21

Gate A green後の順序（独立needのseed、責任の確認、最小のProcedure、自然に生じた場合だけ明示的なHandoff、from / to / conditionの無損失確認）に従い、二つ目のReferenceをauthorし、同じrevisionでactual runnerを実行した。

### authorしたもの

`reference/business-system-design/second-corpus/`（詳細は[`README.md`](../../reference/business-system-design/second-corpus/README.md)）。

| 対象 | 内容 |
| --- | --- |
| Procedure 1 | `make-to-order.production-run`（受注生産の製造一回。指示、物理的な製造、物理的な確認、完了記録、入庫への引き渡しを別のStepとして持つ） |
| Procedure 2 | `finished-goods.stocking`（引き渡された完成品の引き受け、到着確認、入庫、記録） |
| Handoff | `make-to-order.production-run / PR-06` から `finished-goods.stocking / ST-00`。conditionは、引き渡された品が生産側の確認した数量で受け入れ側の責任に引き受けられ、品と数量への責任が生産から倉庫へ移ること |
| seed ledger | 出典8件（すべてdesign-pressure-only、コピー無し、license類推無し）、source-backed fact、TEI-original設計判断、未解決事項9件、cross-viewの問い2件、Handoffが自然に生じる根拠と導出元ではないもの |

二つのProcedureは、それぞれ自分のBusiness needを持ち、Handoffを取り除いても成立する。どのStepもApplication Pattern / Data / Relation / Validation / Capabilityを参照せず、Observationのidentityも持たない（fake realizationを作っていない）。

### RRB-P1

14項目すべてについて、作成者が確認箇所を対応づけた（対応表は`second-corpus/README.md`）。機械的に確認できる項目（ledgerの区分、コピー無し、fake realization無し、trigger / actorが`candidate`のまま、Handoffの導出元）は`TestSecondRealisticReference*`が保持する。

**この確認は作成者自身によるself-reviewである。独立したreviewは未実施**（ledgerの`review`に`pending`として残している）。step順序はTEI-originalなauthoring判断であり、accepted operational sequenceではない（未解決事項U-3）。

### RRB-P2（source / test level）

同じloader、正規化Business model、D4導出、standalone Handoff source形式、baseline checkerへ通した。項目ごとに、確認したことと確認していないことを分ける。

| Contract項目 | 二つ目のReferenceでの扱い |
| --- | --- |
| 1 fake Function / Capabilityなしで手作業・物理・外部Actionを保持 | 確認した。fake realizationが無く（Application文脈、Pattern / Data / Relation / Validation / Capability参照、`application-use`の約束のいずれも無い）、baseline checkerの全Stepが`noApplicationFunction`である |
| 2, 3 既存authorityへのexplicit ref、Pattern-owned meaningの非複製 | 該当なし（既存のSemantic / Applicationを参照しない） |
| 4 Data-state / ownership境界をaccepted authorityからcross-check | **未確認**。Data identityは未解決U-6で、照合する対象authorityが無い |
| 5 projectionでunknownをknownへ丸めない | 確認した。baseline checkerはPattern由来のmeaning行を出さず、Step単位のexplicit meaning行（Data / Relation / Validation / Capability）も空である |
| 6, 9 指示 / 記録 / 物理的actuality / 継続の非同一視 | 確認した。指示、物理的な製造、物理的な確認、完了記録を別Stepにし、確認と記録を一つのStepへ畳まない（realizationの区分とBusiness Action文言について、このfixtureをtestで固定した）。Application Command resultは存在しないため、Command resultとの分離は対象外 |
| 7 HandoffをRuntime Workflowへ変換しない | fixtureはstrict decodeを通り、runtime / workflow系のfieldを持たない |
| 8 H1/R3 PASSをfull Business Flow / DFDへ読み替えない | 主張しない（`second-corpus/README.md`） |
| 10, 11 Handoffを物理Observation / actor差 / Step adjacencyから生成しない | 確認した。隣接するStep、異なるactor、正のphysical Observationが揃っていても、explicit Handoff sourceを外すと正規化modelにもD4 modelにもHandoffは現れない（導出がそれらを入力にしない）。Function sequence / Lifecycle transitionは入力に無く、対象外 |
| 12 conditionを継続predicateとして保持し、Observation identityへ畳まない | 確認した。conditionはcanonical、正規化model、standalone sourceを無損失に往復する。あわせて、このfixtureのconditionがtarget triggerと文字列として一致せず、Step identityやObservation語彙を含まないことをtestで固定した。意味上の言い換えを検出する仕組みではない |
| 13 D3 / D4 / S4-A / H1 / R3 / C1を同じ基準で比較 | 下記 |
| 14 O1 | 適用しなかった（protected-target roleが反復しない。Application Patternを参照しない）。target-role taxonomyは増やしていない |
| 15 新しいdirect Data / Relation ref | 発生しなかった（D4がfail closedする条件が生じていない） |
| 16 新しい不足のA-set / E7への再監査 | 下記 |

項目13の内訳:

- **H1 / R3**: from / to / conditionを、正規化したhandoff-only relationとstandalone Handoff sourceの両方で無損失に保持できることを確認した。
- **D3 / D4**: 正規化Business modelとD4導出は二つ目のReferenceでも成立した。ただしdirect Data / Relation refを一つも持たない（U-6）ため、D4のref kindの十分性を積極的に押してはいない。確認したのは、unknownをrefへ昇格しない側だけである。
- **S4-A**: Handoff endpointをProcedure key + Step keyで解決し、古いendpoint、未知のProcedure、空のcondition、重複したrelationをfail closedにすることを確認した。二つのProcedureはStep IDの接頭辞（`PR-` / `ST-`）が異なるため、別ProcedureでStep IDが衝突するcaseは確認していない。
- **C1**: どのProcedureも単一のApplication ownerへ戻さずに説明できる（baseline checkerが`application`を出さない）。ただしApplicationを一つも持たないため、複数のApplicationを跨ぐProcedureへの圧力は掛かっていない。

項目16: このReferenceから、既存のfrom / to / condition、Procedure-scoped address、handoff-only relationで吸収できない不足は出なかった。[Pre-ADR Audit](../BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md)§11が挙げた確認点のうち、条件の自然言語による十分性、triggerとconditionの分離、actor差からの非導出は確認した。一方、**actorが同じ場合にtransferを否定しないこと**（両Procedureのactorは異なる）、**conditionを支えるevent / documentのidentityの要否**（U-5で未author）、**actor / responsibility detailをstructuredにする要否**（actorは`candidate`のラベルのまま）は、不足が出なかったのではなく**圧力を掛けていない**。「Reopen trigger」が成立しなかった根拠にはするが、これらが不要であることの証明にはしない。

### RRB-P3（actual execution）

Repositoryへpushした`R = 82e430109f8f02944519738045f3d6c0bef4f295`を、pristineなdetached cloneで`tools/run-business-procedure-gate-a.sh`により実行した（Ubuntu 22.04 / WSL2、`go1.25.14 linux/amd64`、`GOENV=off GOWORK=off GOFLAGS=`）。

- Step A1〜A8がすべてexit 0、`gateAPassed = true`、`failedSteps = []`
- A7: critical manifest 39件のexact nameがすべて存在し、focused run 39件すべてがPASS（従来の26件と`TestSecondRealisticReference*`の13件）
- A8（新設）: 二つのfixtureをbaseline checkerへ通し、blocking diagnosticなし、Application Function rowなし
- Evidence: GitHub Release [`business-procedure-gate-a-evidence-R-82e4301`](https://github.com/hat47x/tei-platform/releases/tag/business-procedure-gate-a-evidence-R-82e4301)（tagは`R`を指す）のasset `tei-lane-c-gate-a-evidence-R-82e4301.tar.gz`（6,009 bytes）、SHA-256 `3a433835241b82b564e47cb5eb48e63c24f7a5afd3797ae1090afa9add058d7a`。再ダウンロードしてSHA-256と`MANIFEST.sha256`を照合した。DSN / credential / secretが含まれないことをgrepで確認した

作成時の補助検証として、seed data（fixtureとledger）へ35通りの変異を入れ、すべてtestが失敗することを確認した。この変異検証はrunnerのEvidenceには含めない。

### 主張しないこと

- **独立reviewを経たこと**（作成者とself-reviewerが同一のAI）
- Handoff一般の妥当性。二例あることは一般化の証明ではない。二例目も同じ作成者が同じprotocolでauthorしており、独立した第三者の業務設計ではない
- 実際のApplication / Data / Runtimeが存在すること
- full Business Flow / DFDのConformance
- Contract ADRのready（判定はPre-ADR Auditが持つ）

### Blocker Bの状態

RRB-P1 / P2 / P3のEvidenceを取得した（上の限定つき）。ISSUE-000048のBlocker Bへ、revision `R`とEvidence natureを渡した。RRB-P2は項目4が未確認のまま残るため、達成とみなしてよいかもAuditが判断する。**Pre-ADR Auditによるreadiness再判定は未実施**であり、Contract ADRは**Not Ready**を維持する。利用者の方針（2026-09-21）により、再判定は**別の生成AIによる独立レビューの後**に行い、self-reviewだけを根拠にしない。

## 完了条件

- [x] `order_entry`と独立したcorpusを1件選定し、出典・利用条件・対象範囲を記録した
- [x] Evidence natureとcandidate design questionを整理した
- [x] Lane B RRB-P1 acceptance条件を製造・在庫候補へどう適用するか整理した
- [x] R2 negative proofでsystem result / actuality / Handoff non-inferenceの主要short-circuitをreviewした
- [x] Gate A green後にseed candidateをreviewするissue-local checklistを準備した
- [x] actual Gate A green後に、Handoff検証とは独立したBusiness need / Reference seedをRRB-P1として成立させた（前提のGate A greenは2026-09-20に充足。2026-09-21、作成者self-review。独立reviewは未実施）
- [x] seedのknown / accepted / derived / proposal / unresolvedをreviewできる形で記録した（`second-corpus/seed-ledger.json`。`derived`と`proposals`は意図して空）
- [x] responsibility transferが自然に存在する場合、`from` / `to` / `condition`がaccepted TEI-original sourceからlosslessに写せることを確認した（authorしたTEI-original sourceについて。独立acceptanceは未実施）
- [x] 元資料に無いidentity / actor / sequenceを補完していないことを確認した（trigger / actorは`candidate`のまま。step順序はTEI-originalな判断で、operational sequenceとしては未解決U-3）
- [x] system result / Observation / adjacency / actor差からHandoffを生成していないことをactual testで確認した（R = `82e4301`のA7）
- [x] Handoff sourceが無い箇所はunknownのまま保持され、fail closedになることを確認した（R = `82e4301`のA7）
- [x] ISSUE-000048のblocker Bへ、取得revisionとEvidence natureつきで結果を渡した
- [x] 独立review（OpenAI GPT-5.6 Sol）を実施し、RRB-P1 14項目・seed review checklist・O-1〜O-5・RRB-P2/P3を記録した（2026-09-24）。結果は[`BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_RESULT_2026-09.md`](../BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_RESULT_2026-09.md)の`review-blocked`。Pre-ADR Audit再判定は未実施
- [ ] Pre-ADR AuditがBlocker Bを再判定した（判定はAuditが持つ）

## 独立review結果 — 2026-09-24

作成者とは別系統の生成AI（OpenAI GPT-5.6 Sol）が固定revision `R = 82e430109f8f02944519738045f3d6c0bef4f295`をreviewした。

結果:

```text
review result = review-blocked

RRB-P1 = 未達
  item 14を満たす「forced Handoffではない」ことが未立証
  conditionがST-00のmeaningを内包
  count accountability transferとU-4/U-6/U-9が不整合

RRB-P2 = 未達
  item 4 Data-state / ownership cross-check未確認
  condition continuation predicateも再authoring必要

RRB-P3 = 達成
  recorded actual Gate A EvidenceをRelease metadataで照合
  reviewer自身のrunner再実行はしていない

recommended B-F19b = Unresolved
```

詳細とrequired fixesは[`BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_RESULT_2026-09.md`](../BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_RESULT_2026-09.md)を正とする。

このreviewはPre-ADR Auditの再判定ではない。次はsource meaning / pressureのblocking findingsを解消し、必要なら再reviewした後にAudit §17を再判定する。

## 独立review後remediation — 2026-09-24

独立reviewのF-1 / F-2 / F-4とsame-actor coverage recommendationを受け、current source / pressureを狭い範囲で改訂した。

変更したmeaning:

```text
production independent need
  + physical completion separation
  + terminal release from production control
  != a stocking-specific endpoint invented for Handoff

Handoff condition
  = explicit production-side release from production control
  + permission for warehouse intake to begin
  + received-count reconciliation remains pending
  != target Procedure trigger

condition
  != ST-00 already completed
  != ST-01 reconciliation succeeded
  != count accountability already transferred
```

追加critical pressure:

- `TestSecondRealisticReferenceProductionReleaseHasIndependentBusinessMeaning`
- `TestSecondRealisticReferenceKeepsCountAccountabilityUnresolvedUntilReconciliation`
- `TestSecondRealisticReferenceExplicitHandoffDoesNotDependOnActorDifference`

これら3件を`tools/business-procedure-gate-a-critical-tests.txt`へ追加した。

境界:

- RRB-P2 item 4のData-state / ownership authorityは**未解決のまま**で、fake Data identityは追加していない。
- remediationは独立reviewer自身（GPT-5.6 Sol）がauthorしたため、旧reviewをcurrent sourceのindependent acceptanceへ転用しない。`seed-ledger.json`は`re-review-required`を保持する。
- 旧`R = 82e4301`のGate A PASSはhistorical Evidenceであり、current remediation revisionのP3 Evidenceではない。current sourceは新しいGate A actual runが必要。
- Pre-ADR Auditの再判定はまだ行わない。

current next state:

```text
P1 semantic remediation source
  -> actual Gate A rerun
  -> independent re-review by a reviewer independent of this remediation author
  -> P2 item 4 resolution / status decision
  -> Pre-ADR Audit re-judgment
```

## 次のincrement

### Gate A green前

1. R2で未説明の**具体的なprovenance gap / contradiction class**が見つかった場合だけcandidate-design researchを追加する
2. public-source auditに不足がある場合だけ、instruction / system entry / physical actualityの分離pressureを追加確認する
3. RRB-P1 seed review checklistの観点を維持するが、回答内容やBusiness meaningは先に固定しない
4. candidate / unresolved分類を維持し、accepted Business sourceへ昇格しない
5. 同型negative Handoff caseやsynthetic sequenceを増やして「準備が進んだ」ことにしない
6. Gate A actual executionが可能になればISSUE-000048 Blocker Aを優先する（2026-09-20に実行・PASS済み。以後は下の「Gate A green後」へ進める）

Gate A前のresearchに具体的gapが無ければ、**追加文書を作らず待つこと自体を正しい停止条件**とする。

### Gate A green後

1. Handoff検証とは独立したsmall Business need / Reference seedをRRB-P1条件でreviewしてauthorする
2. seedだけからminimal Procedure / Step meaningを成立させる
3. source responsibility / target responsibility / continuation conditionをreviewする
4. responsibility transferが成立する場合だけexplicit Handoff candidateをauthorする
5. condition成立Evidenceが必要な場合、まず既存Source / Provenance / Evidence mechanismでtrace可能か確認する
6. 確定できない項目はunresolvedのまま保持するnegative PressureをISSUE-000042側へ渡す
7. source/test-level integration後、actual Repository runnerでsystem result / Observation / adjacency / actor差からHandoffを生成しないことを確認する
8. RRB-P2 / P3とblocker Bを閉じられるEvidence natureに到達した時だけISSUE-000048へhandoffする

2026-09-21: 1〜4、6、7は「実施記録」のとおり実施し、8はISSUE-000048への引き渡しとして行った。5（conditionを支えるEvidenceを既存mechanismで追えるかの確認）は、そのevent / documentをauthorしていない（U-5）ため実施していない。blocker Bを閉じるか（再判定）はPre-ADR Auditが持つ。

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
- [`LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`](../LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md)
- [ISSUE-000048](000048-business-procedure-contract-adr-readiness.md) / [ISSUE-000049](000049-business-authoring-production-adoption-readiness.md)
- [ISSUE-000042](000042-business-procedure-cross-view-reference.md)