# Business Procedure Design Quality Model — 2026-09

Status: **Lane C / design-quality model / no new Product Requirement / no new Gate / no production schema adoption**

## 1. 目的

Business Procedureの品質を、field数・diagramの埋まり具合・自動化率ではなく、**業務意味をどこまで根拠を失わずに説明・検証・修正できるか**で評価する。

本書はRQ-025 / RQ-026とPAS-07を、Reference authoring・cross-view review・second realistic corpusへ落とすための設計品質モデルである。

```text
high quality Business design
  != every cell is filled
  != every Action has an Application Function
  != every physical fact has a system event
  != every continuation is a Workflow edge

high quality Business design
  = accepted meaning is distinguishable from inference
  + unresolved remains visible
  + exact authority can be traced
  + realization and actuality are not confused
  + cross-view contradiction can return to authoritative source
```

この品質モデルはCanonical schema、永続enum、coverage scoreを定義しない。ReferenceやEditorでどのmetadataをdurableに持つべきかは、具体的なlossが観測された場合だけ別途決める。

## 2. Product上の中心価値

Business設計で最も避けるべき失敗は、情報不足を「それらしい完全なmodel」で覆うことである。

TEIは次を優先する。

1. 業務上の目的・責任・行為をApplication実装から独立して確認できる。
2. systemが実現する部分とmanual / external / physicalな部分を同じBusiness文脈で扱える。
3. system result、external / physical actuality、Business上の受入・完了を区別できる。
4. Data / Function-Interaction / Businessの一視点から他視点を推測で埋めず、exact sourceへ辿って相互検証できる。
5. 確定できない点をunresolvedとして残したまま、分かっている範囲の設計価値を失わない。
6. reviewerが「何が事実で、何がTEIで採用した判断で、何がderivedで、何が未確定か」を説明できる。

したがって、**明示されたunknownは品質低下ではない。根拠のない補完によるfalse completenessの方を品質欠陥とする。**

## 3. Evidence natureを混ぜない

Business設計reviewでは少なくとも次の性質を区別する。これは現時点でpersisted enumを要求するものではなく、authoring / review上の分類である。

| Nature | 意味 | Authority上の扱い |
| --- | --- | --- |
| source-backed fact | public / user / repository sourceから確認できる事実・制約・業務concern | TEI固有のexact Step / actor / Handoffを自動authorしない |
| accepted TEI Business meaning | TEI Reference / authoritative sourceとして明示的に採用したpurpose・Action・responsibility・condition等 | Business側のaccepted meaning |
| derived cross-view fact | accepted refをexact解決してData / Function / Security等から機械的に導出した関係 | derived Projection。sourceへ逆書きしない |
| proposal / inference | AI、人、外部Reference、heuristic等が提示した候補 | Acceptance前はauthorityなし |
| unresolved | 情報不足、複数解釈、source未取得等で確定できない点 | 成功・不存在・defaultへ丸めない |

許されないauthority direction:

```text
derived Projection
  -X-> authoritative Business sourceへの暗黙backfill

external high-level fact
  -X-> exact TEI actor / Step sequence / Handoffの自動authoring

name similarity / adjacency / execution order
  -X-> semantic relation

unresolved
  -X-> convenient default
```

一方、source-backed factやderived contradictionはaccepted sourceを**再検討する理由**にはなる。修正する場合もShared Acceptance Boundaryを通してauthoritative source側を変更する。

## 4. Business設計品質の10観点

以下はreview観点であり、採点表ではない。domainで不要な観点を無理に構造化しない。

### Q1 — Intent clarity

何のための仕事か、何を契機に扱うのかを説明できるか。triggerが未確定なら未確定であることを隠さない。

### Q2 — Responsibility clarity

Business上の責任文脈をApplication ownerやAuthorization Roleから独立して説明できるか。exact actorが未確定でも、責任境界自体が既知か未知かを区別する。

### Q3 — Action truthfulness

Business ActionをFunctionへ還元していないか。manual / external / physical / judgmentをfake Capabilityで埋めていないか。

### Q4 — Realization traceability

Application Pattern / Validation / Capability等を利用する場合、accepted identityへexactに辿れるか。実現が無いActionをinvalidにしない。

### Q5 — Actuality discipline

system command / result、external / physical fact、Business上の受入・完了を同一視していないか。

### Q6 — Continuation clarity

仕事がどこで完了・継続・保留されるかを説明できるか。責任の受け渡しがある場合だけexplicit Handoffをauthorしているか。

### Q7 — Cross-view integrity

Data / Function-Interaction / Businessのexact refからcorroboration・contradiction・unresolvedを区別できるか。trace absenceをdependency absenceへ丸めていないか。

### Q8 — Epistemic transparency

fact / accepted meaning / derived / proposal / unresolvedのnatureをreview時に説明できるか。未証明のdetailでdiagramを埋めていないか。

### Q9 — Authority discipline

同じconcernに複数の暗黙正本を作っていないか。path順、input順、first/last writer、schemaVersion等でauthorityを裁定していないか。

### Q10 — Change auditability

Business meaning change、source locator change、derived projection change、World / Conformance base changeを区別できるか。stale candidateを最新版へ黙って適用していないか。

### Derived procedural candidateを使う場合の追加確認

trajectory、agent exploration、過去のsuccessful composition等からprocedural candidateを作る場合も、Business Procedure品質の判定をその候補へ委譲しない。

```text
past execution order
  != accepted Procedure order

successful trajectory
  != correct Business Procedure

derived procedural graph
  != Business authority
```

利用する場合は、少なくとも元Execution / Observation / Evidenceへtraceでき、current Business context / accepted authorityで再確認できることを求める。derived graph / indexを削除してもaccepted Business meaningが失われないことを維持する。

この確認は新しい品質scoreやpersisted statusを追加するものではない。

## 5. Intention / system result / actuality / Business acceptanceを分ける

製造、物流、外部承認、SaaS連携等では次が同時に現れやすい。

```text
Business intention / instruction
        ↓ may request
Application / system action
        ↓ may report
system result
        ↓ may be corroborated by
Observation / external evidence
        ↓ may support
Business acceptance / completion / continuation
```

矢印は「必ず自動的に成立する」を意味しない。

```text
instruction issued
  != physical work completed

command accepted
  != external fact observed

system record updated
  != physical-world fact necessarily true

physical fact observed
  != Business responsibility automatically transferred

Business condition
  != Observation identity
```

domainによって複数段が一致することはあり得るが、**一致はaccepted contractから証明する**。実装都合や時系列だけから畳まない。

この分離はRQ-018だけのRuntime論点ではない。RQ-025ではBusiness Actionを正しく説明するため、RQ-026ではBusiness / Function / Dataの矛盾を見つけるための設計品質条件でもある。

## 6. Responsibilityとperformerを分ける

Business責任は「誰がボタンを押したか」「どのApplicationが処理したか」と同義ではない。

```text
Business responsibility context
  != performer identity
  != authenticated principal
  != Authorization role
  != Application owner
```

Procedure.actor / job contextはBusiness意味を説明する一要素だが、Handoff identityそのものではない。

したがって:

```text
actor changed
  != Handoff proven

actor unchanged
  != Handoff impossible
```

actor差はcorroborationになり得るが、責任移転はexplicit Business sourceで決める。

## 7. Handoffは「次の処理」ではなく責任継続のrelation

Current H1/R3の最小形は次である。

```text
Handoff
  from { procedure, step }
  to   { procedure, step }
  condition
```

このrelationが表すのは、単なるchronological edgeではなく、**あるBusiness Stepで保持されていた仕事が、Business conditionの成立を境に、別のaddressable Business Stepで継続できる／継続すべきになること**である。

### from

responsibility transferの出発点としてreview可能なProcedure-scoped Step。runtime node IDではない。

### to

continuation側のBusiness responsibilityを担うaddressable Step。単に次の配列要素ではない。

### condition

responsibility continuationがBusiness上成立するpredicate。target Procedureのtrigger、Observation record、runtime event IDと同一ではない。

良いconditionは、変更すると責任継続の意味も変わる。単なる「前のStepが終わったら」のようなtautologyで十分かは、domain上その完了に独立したBusiness meaningがある場合だけ判断する。

### Handoffを証明しないもの

```text
Step adjacency
Function call / Function sequence
actor difference
Lifecycle transition
Command success
physical Observation
queue / event / scheduler
```

これらはHandoffをcorroborateすることはあってもauthorしない。

## 8. Handoff conditionとEvidenceを分ける

Business conditionは「何が成立すれば責任を継続できるか」というmeaningであり、Evidenceは「そのconditionが成立したと判断する根拠」である。

```text
Business condition
  = continuation semantics

Observation / document / approved record / external response
  = condition成立を支えるEvidence候補
```

この二つを同一fieldへ畳む必要性は現在証明されていない。

Observation APIやphysical factからHandoffを自動生成しない一方、将来second realistic caseで「conditionだけではreviewに必要なEvidence traceを保持できない」という具体的lossが出た場合は、relationを拡張する前にSource / Provenance / existing Evidence mechanismで表現できるかを確認する。

## 9. Cross-view reviewの結果を成功／失敗だけへ潰さない

RQ-026のcross-view reviewでは、少なくとも次の三つを区別して扱う。

```text
supported
  exact accepted authoritiesが互いに矛盾せずcorroborateする

contradiction
  accepted authorities間に具体的な不整合がある

unresolved
  exact identity / authority / actuality等が不足し、肯定も否定もできない
```

これは新しいdiagnostic severity enumを要求しない。review / reportでの意味分類である。

重要な非推論:

```text
no Data trace
  != no Data dependency

no Function realization
  != invalid Business Action

no Observation
  != physical failure

no explicit Handoff
  != inferred Handoff
  = responsibility transferは少なくとも未証明
```

## 10. Procedure design sufficiency review

Business Procedureは「全fieldが埋まったら完成」ではなく、対象scopeについて次の問いに答えられるかでreviewする。

1. **Why:** この仕事のpurpose / business needは何か。
2. **When:** trigger / continuation条件は何がknownで何がunresolvedか。
3. **Responsibility:** どのBusiness responsibility contextで行われるか。actor詳細が未知でも境界を捏造していないか。
4. **Work:** どのBusiness Action / judgment / manual / external workが存在するか。
5. **Realization:** どこがApplicationで支援され、どこは支援されない／未確定か。
6. **Actuality:** system resultとexternal / physical factをどう区別するか。
7. **Continuation:** completion / hold / Handoffが何を根拠に成立するか。
8. **Cross-view:** Data / Function / Security等から何がsupported / contradiction / unresolvedになるか。
9. **Authority:** 各meaningのauthoritative sourceは何か。derived Projectionを第二正本にしていないか。
10. **Change:** source move、Business change、referenced authority changeを区別してreviewできるか。

対象domainで「不明」が正しい答えならunresolvedでよい。レビューの目的は空欄をなくすことではなく、**現在のknowledge boundaryを正しくすること**である。

## 11. Realistic Reference seedのreview原則

第二・第三のReferenceは既存contractを通すためのfixtureではなく、Handoff等を取り除いても独立したBusiness設計価値を持つseedから始める。

ここではBusiness Procedure一般のdesign-quality review原則だけを示す。**second realistic Referenceのacceptance / maturity条件（RRB-P1 / P2 / P3）の正本は `LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md`** とし、本節を別のacceptance contractにしない。

accepted seedへ進める前のreviewでは、少なくとも次を見る。

### 11.1 independent business need

「このfieldを検証したい」ではなく、domainとして説明可能なbusiness need / problemがある。

### 11.2 bounded scope

full domainを再現せず、何を扱い何を扱わないかが明示される。

### 11.3 provenance-safe known facts

外部sourceを使う場合、high-level factとTEI-original design decisionを分離し、外部modelを転記しない。

### 11.4 explicit unresolved

actor、trigger、sequence、Data identity、realization、actuality、responsibility boundary等で分からない点を先に列挙する。

### 11.5 meaningful cross-view question

Data / Function / Businessの少なくとも二視点を使わないと見落とす品質問題を含む。

### 11.6 no forced Handoff

seedがHandoffなしでも価値を持つ。責任移転がaccepted Business meaningから自然に現れた場合だけHandoff candidateをauthorする。

### 11.7 no fake realization

Referenceを「動かせる」ように見せるためだけのApplication Pattern / Capability / Data identityを作らない。

## 12. second manufacturing / inventory corpusへの適用

現在選定済みの受注生産背景のmanufacturing / inventory corpusでは、外部公開資料をexact TEI Procedureとして転写しない。

Gate A green前に深めてよいのは、次のような**design question**までである。

```text
manufacturing / inventory continuationで、
Business instruction、system entry / result、physical completion、
Business上の受入・次責任へのcontinuationは
どこまで同じで、どこから別のmeaningなのか。
```

この問いはHandoffを前提にしない。

特に次を確認対象とする。

- instruction / entryが存在してもphysical inventory movementがまだ成立しない局面を表せるか
- manufacturing completionのsystem記録とphysical completionを無条件に同一視しないか
- physical completionが観測されても、それだけでshipment等へのresponsibility transferを生成しないか
- cross-viewで必要なData identity / Application realizationが未証明ならunknownのまま残せるか
- accepted Business needからresponsibility transferが生じた場合、`from/to/condition`だけでlosslessか

exact actor、accepted Step sequence、Handoff conditionは引き続きunresolvedであり、本書からauthorしない。

## 13. Authoring sequence — Gateではない

second realistic corpusでは次の順序を使う。これは新しいGate / state machineではない。second Referenceのformal maturity判定はLane B Acceptance Contractに従う。

```text
provenance pressure selection
  ↓
non-authoritative Business seed proposal
  ↓  Lane C Business Procedure Gate A actual green後
RRB-P1を満たすaccepted independent Business source
  ↓
minimal Procedure / Step meaning
  ↓  responsibility transferが自然に成立した場合のみ
explicit Handoff candidate
  ↓
D4 / O1 / S4-A / H1/R3 cross-view Pressure
  ↓
RRB-P2 / actual RRB-P3 Evidence
```

Handoff candidateが自然に成立しない場合、second corpusを失敗扱いしてHandoffを捏造しない。その場合は「このseedはH1/R3 generality Evidenceにならない」というnegative resultを記録し、別のindependent seedを探す。

## 14. Reopen trigger

Current minimum contractを拡張するのは、realistic sourceで具体的lossが観測された場合だけとする。

例:

- `from/to/condition`ではresponsibility transferのaccepted meaningをlosslessに保持できない
- Procedure-scoped Step addressでは別責任境界を一意に指せない
- actor / responsibility detailをstructuredにしないとcross-view contradictionを検出できない
- Handoff conditionとEvidence traceを既存Provenance機構で分離できない
- Business FlowをHandoff-only relationより広く持たないと重要な情報・文書・party flowが失われる
- separate Business planeではConformance / change impactをfail closedできない

実装が不便、diagramが寂しい、外部手法に同名概念がある、という理由だけではreopenしない。

## 15. 文書責務

```text
TEI_PRODUCT_REQUIREMENTS.md
  -> Product Requirement正本

requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md
  -> RQ-025 / RQ-026 interpretation index

docs/business-system-design.md
  -> stable architecture boundary

本書
  -> Business Procedure design-quality review model

plan/LANE_B_SECOND_REALISTIC_REFERENCE_ACCEPTANCE_CONTRACT_2026-09.md
  -> second realistic Reference acceptance / maturity条件

plan/issue/000042...
  -> Lane C detailed Reference / Pressure inventory

plan/issue/000048...
  -> Contract ADR readiness work ownership

plan/issue/000050...
  -> manufacturing / inventory second-corpus acquisition / application plan
```

外部知見はquestion-specific research / provenance documentへ近接保持し、public cross-cutting reference noteを第三者Evidenceの第二正本へ戻さない。

## 16. Current decision

```text
Business design quality
  = completeness scoreではなく
    meaning + evidence + authority + actuality + responsibilityの透明性

Current Handoff
  = explicit responsibility / continuation relation
  = from/to/condition
  != chronology / actor change / runtime edge

second realistic corpus
  = Handoff fixtureから始めない
  = independent Business seedから始める
  = acceptance / maturityはLane B Acceptance Contractを正とする

unknown
  = preserve
  != convenient default
```

このmodelから新しいproduction fieldは導入しない。具体的Referenceでlossが出た場合だけ、Requirements / Architecture / ADRへ戻って設計をreopenする。
