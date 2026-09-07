# Lane B — Second Realistic Business Reference Acceptance Contract — 2026-09

Status: **Lane B / supporting Evidence-coverage contract / no new Requirement / PAS / E7 / Gate / production schema**

Related:

- `plan/LANE_B_REALISTIC_BUSINESS_REFERENCE_CORPUS_AUDIT_2026-09.md`
- `plan/LANE_B_REALISTIC_BUSINESS_CORPUS_PUBLIC_SOURCE_AUDIT_2026-09.md`
- `plan/RQ025_RQ026_ACCEPTANCE_TRACE.md`
- Lane C PR #482 / `plan/issue/000042-business-procedure-cross-view-reference.md`

## 1. 目的

B-F19b `TEI-native provenance-backed realistic Reference corpus breadth`を、単なる「第二fixtureが存在する」状態へ縮めず、どのEvidenceが揃えばどのclaimまで上げてよいかを段階化する。

本書は第二Referenceの業務内容やfixture schemaを所有しない。Reference authoring / checker / testsはLane Cが所有する。

またGate 15cを無条件に開くものではない。

```text
B-F19b Reference maturity / breadth
  != Gate 15c automatically required
```

## 2. 三段階status

### RRB-P1 — provenance-safe TEI-native source exists

次をすべて満たす。

1. TEI repository内に、`order_entry`とは独立したBusiness pressure sourceがある。
2. 外部artifactのコピーではなく、TEI独自に再構成したsourceである。
3. provenance manifestまたは同等の記録から、どの公開資料・既存資料をdesign pressureとして使ったか追跡できる。
4. raw external model / manual / implementationを転載していない。
5. 外部資料のlicenseを別artifactへ類推していない。
6. fact / candidate / unresolvedが区別されている。
7. 出典で確認できないactor / trigger / ordering / responsibility / state transitionをaccepted factへ昇格していない。

この段階で主張できること:

```text
TEI-native provenance-backed second Business source exists
```

主張してはいけないこと:

```text
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
6. Business handoff / continuationをRuntime Workflowへ変換しない。
7. H1/R3 handoff-only PASSをfull Business Flow / DFD Conformanceへ読み替えない。
8. Application Command resultと、物理・外部世界のObservation / completionを分離する。
9. D3 / D4 / S4-A / H1 / R3 / C1をcurrent Referenceと同じ基準で比較する。
10. O1 protected-target roleが実際に反復する場合だけO1を適用し、反復しない場合はtarget-role taxonomyを増やさない。
11. new direct Data / Relation refが本当に必要になった場合はD4がfail closedし、designをreopenする。
12. second Referenceから出た新しい不足を、既存A-set / E7へ吸収できるか再監査する。新ID追加を目的化しない。

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

Application数をKPIにしない。

少なくともcurrent `order_entry`では弱い次のpressureを一つ以上、できれば複数含む。

```text
multiple responsibility / job contexts
manual or physical-world completion
external organization / external-world continuation
cross-subject-area or subsystem Data ownership boundary
state change whose business completion is not identical to Application Command result
Business handoff whose meaning is not Runtime orchestration
```

候補としてmanufacturing / inventory continuationは有力だが、固定要件ではない。

## 4. provenance contract

各Business factは少なくとも次のいずれかへ分類する。

```text
fact
  sourceで直接または十分明確に確認できる

candidate
  source pressureから合理的に置いたTEI独自scenario仮説

unresolved
  必要だがsourceだけでは確定できない
```

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

「sourceを参考にした」だけではfact provenanceにならない。

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
  cross-context continuationをlosslessに保持できるか

Do not infer
  full Business Flow / DFD is solved
```

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
E7-08  change impact / exact dependency
E7-09  no premature concept expansion / fail closed
E7-10  accepted source / reload / Shared Acceptance when production meaning requires it
```

## 8. current status

2026-09-07 current:

```text
B-F19a external provenance-backed source availability
  Resolved

B-F19b
  RRB-P1  Not yet achieved in repository
  RRB-P2  Not achieved
  RRB-P3  Not achieved

current order_entry
  remains first small Business Reference

Lane C #482
  owns next Reference fixture / checker work
```

外部source availabilityだけを理由にP1へ上げない。

## 9. Lane Cへのhandoff

第二Referenceを作る場合、最初から広い生産管理systemを再現しない。

推奨する最小単位:

```text
one provenance-backed business event
  +
small chain of Business Actions
  +
one manual/physical/external completion boundary
  +
one cross-subject-area Data/state ownership boundary
  +
only actually evidenced Application/Semantic refs
```

これでD3/D4/O1/H1/R3/S4-A/C1をPressureし、不足が出たところだけ拡張する。

## 10. 非対象

- CONCEPTWARE / TALON artifactの転載
- full manufacturing / inventory package再実装
- Business Flow / DFD production schema採用
- Workflow Engine導入
- generic SemanticRef / Role taxonomy採用
- production `applicationPatternTargets`採用
- new Gate / PAS / E7追加
