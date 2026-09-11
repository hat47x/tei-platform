# Lane B — Second Realistic Business Reference Acceptance Contract — 2026-09

Status: **Lane B / supporting Evidence-coverage contract / no new Requirement / PAS / E7 / Gate / production schema**

Related:

- `plan/BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`
- `plan/LANE_B_REALISTIC_BUSINESS_REFERENCE_CORPUS_AUDIT_2026-09.md`
- `plan/LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`
- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- `plan/issue/000042-business-procedure-cross-view-reference.md`
- `plan/issue/000048-business-procedure-contract-adr-readiness.md`
- `plan/issue/000050-second-realistic-handoff-corpus.md`

## 1. 目的

B-F19b `TEI-native provenance-backed realistic Reference corpus breadth`を、単なる「第二fixtureが存在する」状態へ縮めず、どのEvidenceが揃えばどのclaimまで上げてよいかを段階化する。

本書は第二Referenceの**acceptance / maturity条件**を所有する。具体的な業務内容・corpus取得作業・fixture schemaは所有しない。Reference authoring / checker / testsはLane C、corpus取得計画はISSUE-000050が所有する。

またGate 15cを無条件に開くものではない。

```text
B-F19b Reference maturity / breadth
  != Gate 15c automatically required
```

Business Procedure一般のreview原則は`BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md`を参照する。本書はその原則を第二realistic Referenceのacceptance条件へ絞る。

## 2. 三段階status

### RRB-P1 — provenance-safe TEI-native Business source exists

次をすべて満たす。

#### provenance / independence

1. TEI repository内に、`order_entry`とは独立したBusiness need / pressure sourceがある。
2. 「Handoff fieldを検証するため」ではなく、Handoffを取り除いてもdomain上説明できるindependent business needを持つ。
3. 外部artifactのコピーではなく、TEI独自に再構成したsourceである。
4. provenance manifestまたは同等の記録から、どの公開資料・既存資料をdesign pressureとして使ったか追跡できる。
5. raw external model / manual / implementationを転載していない。
6. 外部資料のlicenseを別artifactへ類推していない。

#### epistemic quality

7. source-backed fact、accepted TEI Business meaning、proposal / inference、unresolvedをreview上区別できる。derived cross-view factがある場合もaccepted sourceへ逆書きしない。
8. 出典で確認できないactor / trigger / ordering / responsibility / state transition / Data identity / realizationをaccepted factへ昇格していない。
9. unknownをdiagram completenessのために埋めず、false completenessを作っていない。

#### Business quality

10. scopeがboundedであり、full domain再実装を目的にしていない。
11. manual / physical / external actualityをApplication resultへ還元すると意味が壊れるpressureを一つ以上持つ、または同等に`order_entry`では弱い独立Business pressureを持つ。
12. Data / Function-Interaction / Businessの少なくとも二視点から見ないと判断できないmeaningful cross-view questionを持つ。
13. Referenceを実行可能に見せるためだけのfake Application Pattern / Capability / Data identityを作っていない。
14. HandoffがなくてもReferenceとして独立価値を持ち、responsibility transferをtest都合で捏造していない。

この段階で主張できること:

```text
TEI-native provenance-backed second Business source exists
```

主張してはいけないこと:

```text
second Handoff proven
cross-view contract passed
Reference breadth Evidence established
production Canonical contract validated
```

### RRB-P2 — cross-view pressure is source/test-level integrated

P1に加えて、Lane CのReference checker / test-only pressureへ接続し、少なくとも次をsource-levelに確認する。

1. manual / physical / external Business Actionをfake Application Function / Capabilityなしで保持できる。
2. existing TEI Semantic / Application authorityを参照する場合だけexplicit refを置く。
3. Pattern-owned Data / Relation / Query / Capability / DatasourceをBusiness sourceへ複製しない。
4. Data-state / ownership boundaryを名称prefixではなくaccepted authorityからcross-checkする。
5. Business Action × Function-Interaction × Dataのprojectionでunknownを勝手にknownへ丸めない。
6. Business instruction / system result / external actuality / Business acceptance・continuationを自動的に同一視しない。
7. Business handoff / continuationをRuntime Workflowへ変換しない。
8. H1/R3 handoff-only PASSをfull Business Flow / DFD Conformanceへ読み替えない。
9. Application Command resultと、物理・外部世界のObservation / completionを分離する。
10. physical Observationだけからresponsibility Handoffを生成しない。
11. actor差、Step adjacency、Function sequence、Lifecycle transitionだけからresponsibility Handoffを生成しない。
12. Handoff conditionをBusiness continuation predicateとして保持し、その成立を支えるObservation / document等と同一identityへ畳まない。
13. D3 / D4 / S4-A / H1 / R3 / C1をcurrent Referenceと同じ基準で比較する。
14. O1 protected-target roleが実際に反復する場合だけO1を適用し、反復しない場合はtarget-role taxonomyを増やさない。
15. new direct Data / Relation refが本当に必要になった場合はD4がfail closedし、designをreopenする。
16. second Referenceから出た新しい不足を、既存A-set / E7へ吸収できるか再監査する。新ID追加を目的化しない。

Handoffが自然に生じた場合は追加で:

```text
from
  accepted Business Stepをexactに指す

to
  accepted Business Stepをexactに指す

condition
  responsibility continuationのBusiness predicateをlosslessに保持する
```

を確認する。Handoffが自然に生じないP1 sourceを、P2達成のために改変してHandoffを作らない。そのsourceはH1/R3 generality Evidenceにはならないと記録する。

この段階で主張できること:

```text
second realistic source is integrated as source/test-level cross-view pressure
```

まだ主張してはいけないこと:

```text
actual Repository execution green
Reference breadth Evidence fully established
```

### RRB-P3 — actual execution-qualified breadth Evidence

P2に加えてactual Repository environmentで、第二Referenceを含む対象checker / testsがgreenである。

最低限:

```text
gofmt / format check
go vet or equivalent static check
go test relevant checker package
baseline checker
intentional negative / conflict probe where applicable
second Reference specific pressure tests
```

をactual runnerで確認する。

この段階で初めて次を主張できる。

```text
provenance-backed TEI-native realistic Reference breadth
  execution-qualified at current scope
```

それでも、業務一般への普遍性や全業種Conformanceを意味しない。

## 3. 第二Referenceが持つべき最低Business pressure

Application数やStep数をKPIにしない。

少なくともcurrent `order_entry`では弱い次のpressureを一つ以上、できれば複数含む。

```text
multiple responsibility / job contexts
manual or physical-world completion
external organization / external-world continuation
cross-subject-area or subsystem Data ownership boundary
state change whose business completion is not identical to Application Command result
Business handoff whose meaning is not Runtime orchestration
instruction / result / observed actuality / Business acceptanceの分離
```

候補としてmanufacturing / inventory continuationは有力だが、固定要件ではない。

## 4. provenance / Evidence-nature contract

各Business statementは、review時に少なくとも次のnatureを区別できるようにする。

```text
source-backed fact
  sourceで直接または十分明確に確認できる

accepted TEI Business meaning
  TEI-native authoritative sourceとして明示的に採用した

derived cross-view fact
  accepted refをexact解決して機械的に導出した

proposal / inference
  source pressureから置いたTEI独自candidateで、Acceptance前

unresolved
  必要だがsource / accepted decisionだけでは確定できない
```

これはproduction enumを要求するものではない。

provenance recordには可能な範囲で:

```text
source title / publisher or author
source URL or repository reference
access / research date
observed high-level fact
reuse status / license status if known
what is NOT copied
```

を残す。

「sourceを参考にした」だけではfact provenanceにならない。また外部source-backed factからexact TEI actor / Step / Handoffを自動authorしない。

## 5. current candidate stackへの圧力

第二Referenceは新概念を足すためではなく、current candidateを壊すために使う。

### D3 / D4

```text
Question
  current D4 ref kindsだけでBusiness cross-view meaningを保持できるか

If yes
  D4 minimum sufficiency evidence strengthens

If no
  missing accepted meaningを特定
  D4 fail closed
  direct ref kind / representationをreopen
```

### S4-A / H1 / R3

```text
Question
  Procedure-scoped local address + handoff-only relationで
  cross-context responsibility continuationをlosslessに保持できるか

Do not infer
  full Business Flow / DFD is solved
  actor difference = Handoff
  physical completion = Handoff
```

Handoff `condition`とcondition成立Evidenceの分離がlossyなら、H1/R3へfieldを足す前に既存Source / Provenance / Evidence mechanismでtrace可能か確認する。

### O1

```text
Question
  target != realization role separationが別Referenceでも反復するか

If repeated
  field-specific source representation pressureを強める

If not repeated
  generic Role taxonomyを作らない
```

### C1

```text
Question
  Business Procedureをsingle Application ownerへ戻さず説明できるか
```

## 6. B-F19b closure rule

B-F19bを一つのbooleanへ潰さず、statusを次で記録する。

```text
Unresolved
  P1未達

Source-level corpus exists
  P1達成 / P2未達

Source/test-level breadth pressure implemented
  P1 + P2達成 / P3未達

Execution-qualified breadth Evidence
  P1 + P2 + P3達成
```

`Reference maturity / breadth established`という表現はP3前には使わない。

## 7. Gate / PASとの関係

- 新Product Requirementを追加しない。
- PAS-07へ新E7を追加しない。
- Gate B / Gate C Required Evidenceを変更しない。
- Gate 15cを自動的にopenしない。
- 第二Referenceから具体的なminimum-contract不足が見つかった場合だけ15c / pre-ADR candidateをreopenする。

既存PAS-07への主な接続:

```text
E7-01  manual / external Action != Function
E7-02  explicit authority trace / no duplicate authority
E7-04  cross-view contradiction / meaning change
E7-06  address / handoff integrity
E7-08  unknown保持 / fail closed / change impact
E7-09  no premature concept expansion
E7-10  accepted source / reload / Shared Acceptance when production meaning requires it
```

## 8. current status — 2026-09-11

```text
B-F19a external provenance-backed source availability
  Resolved

manufacturing / inventory candidate corpus
  selected
  provenance / non-copying boundary prepared
  candidate design question prepared

Business design quality model
  prepared

B-F19b
  RRB-P1  Not yet achieved in repository
  RRB-P2  Not achieved
  RRB-P3  Not achieved

current order_entry
  remains first accepted small Business Reference
```

ISSUE-000050でcorpus選定・provenance・pre-authoring review準備まで進んでも、accepted TEI-native Business sourceが無い限りP1へ上げない。

External source availabilityや文書上のscenario outlineだけを理由にP1へ上げない。

## 9. Lane C / ISSUE-000050へのhandoff

第二Referenceを作る場合、最初から広い生産管理systemを再現しない。

推奨する最小単位:

```text
one provenance-backed independent business need
  +
small set of Business Actions
  +
one manual / physical / external actuality boundary
  +
one meaningful cross-view question
  +
only actually evidenced Application / Semantic refs
```

authoring順序:

```text
provenance / candidate question
  ↓
ISSUE-000048のactual Gate A green
  ↓
RRB-P1を満たすTEI-native Business source
  ↓
source/test cross-view integration
  ↓
responsibility transferが自然に存在する場合だけH1/R3 pressure
  ↓
actual executionでRRB-P3
```

これでD3/D4/O1/H1/R3/S4-A/C1をPressureし、不足が出たところだけ拡張する。

## 10. 非対象

- CONCEPTWARE / TALON artifactの転載
- full manufacturing / inventory package再実装
- Business Flow / DFD production schema採用
- Workflow Engine導入
- generic SemanticRef / Role taxonomy採用
- production `applicationPatternTargets`採用
- Evidence-nature専用Canonical enumの先行採用
- Handoff Evidence専用fieldの先行採用
- new Gate / PAS / E7追加
