# Business Procedure Blocker B — 独立生成AIレビュー結果

- レビュー日: 2026-09-24
- reviewer / model: OpenAI GPT-5.6 Sol
- independence basis: Blocker B Evidenceの作成者 / self-reviewerはClaude Sonnet 5。レビュアーはAnthropic Claude系ではなく、review handoffを準備したClaude Opus 5.5とも別系統
- review target revision: `82e430109f8f02944519738045f3d6c0bef4f295`
- review input: [BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_HANDOFF_2026-09.md](BUSINESS_PROCEDURE_BLOCKER_B_INDEPENDENT_REVIEW_HANDOFF_2026-09.md)
- review result: **review-blocked**
- Pre-ADR Audit re-judgment: **未実施**
- Contract ADR: **Not Readyのまま**

## 1. 結論

second corpusは、`order_entry`とは独立したBusiness need、provenance、bounded scope、actuality separation、fake realizationを作らない境界という点では有効な第二Referenceである。Gate A actual execution Evidenceもrevision `R`へ紐づいて存在する。

一方、Blocker Bが要求する「独立Business needから自然に生じたsecond realistic TEI-original Handoffで、from / to / condition sufficiencyを確認した」というclaimは、現状のsourceだけでは成立したと判定できない。

主な理由は三つある。

1. Handoff conditionがtarget Step `ST-00`のBusiness Actionを意味的に内包しており、continuation predicateではなくtarget actionの完了を言い換えている可能性が高い。
2. conditionは「goods and countへのaccountabilityがwarehouseへ移る」と確定する一方、その直後の`ST-01`で初めてphysical arrivalとproduction-confirmed quantityを照合する。責任境界U-4、Data ownership U-6、acceptance criteria U-9がunresolvedのままなので、数量責任の移転時点をpositive Handoff meaningとして固定するのは早い。
3. RRB-P2 item 4はAcceptance Contract上の必須項目だが、Data-state / ownership authorityが無いため未確認である。したがってP2達成、ひいては`Execution-qualified breadth Evidence`は主張できない。

よって、Pre-ADR Auditの再判定前にsource meaning / pressureを修正または再authoringする必要がある。

## 2. finding

### F-1 — blocking — conditionがtarget Step completionを意味的に内包する

Handoff source:

- `reference/business-system-design/second-corpus/production-and-stocking-reference-set.json:20-24`
  - to = `finished-goods.stocking/ST-00`
  - condition = “the released goods ... **have been taken in charge by the receiving responsibility** ...”

target Step:

- `reference/business-system-design/second-corpus/finished-goods-stocking.json:20-22`
  - `ST-00` = “**take the released goods in charge**”

conditionは文字列としてST-00と一致しないが、Business meaningとしてはST-00の完了を含んでいる。

current pressureは:

- `tools/reference-business-procedure-check/second_realistic_reference_pressure_test.go:557-563`

でtarget trigger / Step actionとの**exact string equality**等をrejectするだけで、意味的な言い換えを検出しない。

したがって「conditionはtarget Stepとは独立したBusiness continuation predicateである」というpositive claimは未立証である。

**Audit再判定前に必要:** condition / endpoint semanticsを見直し、target Stepの完了を条件自身へ循環的に埋め込まない形へする。少なくともparaphrase / semantic containmentをreview可能なpressureを追加する。

### F-2 — blocking — count accountability transferがST-01より先に確定している

Handoff conditionは:

- `production-and-stocking-reference-set.json:24`

で“accountability for the goods and for their **count** moves from production to the warehouse”とする。

一方:

- `finished-goods-stocking.json:25-27`
  - `ST-01` = “check what physically arrived against what production confirmed as completed”

であり、実到着数量とproduction-confirmed quantityの照合はHandoff target `ST-00`の**後**である。

さらにledgerは:

- `seed-ledger.json:56` U-4 — damage / loss / **count** accountabilityのhandover window中の分担は未決定
- `seed-ledger.json:58` U-6 — completed / received / stock quantityのData identity / ownershipは未決定
- `seed-ledger.json:61` U-9 — physical confirmation / arrival checkのacceptance criteriaは未決定
- `seed-ledger.json:64` CQ-1 —どのquantityをどの時点でauthoritativeとし、誰がownするか自体がcross-view question

と記録する。

このため、conditionが数量責任の移転を確定していることとunresolved ledgerの間に意味上の緊張がある。

**Audit再判定前に必要:** custody responsibilityとcount accountabilityを同一時点で移すのか、ST-01照合後にcount responsibilityを確定するのかをauthoritative Business meaningとして決める。決められなければconditionからcount transferの確定表現を外しunresolvedを保持する。

### F-3 — blocking — RRB-P2 item 4が未達

Acceptance Contract `2のRRB-P2は「次をsource-levelに確認する」とし、item 4として:

> Data-state / ownership boundaryを名称prefixではなくaccepted authorityからcross-checkする

を要求する。

しかし:

- `seed-ledger.json:58` U-6 — Data identity / ownership unresolved
- Pre-ADR Audit `17 — RRB-P2 item 4は対象Data authorityが無く未確認

である。

Acceptance Contract `6のclosure ruleは:

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

と明示する。

**Audit再判定前に必要:** item 4をactual accepted authorityでcross-checkするpressureを追加するか、P2未達を明示してB-F19bを下位statusに留める。未確認itemをP2達成へ数えてはならない。

### F-4 — major — PR-06がHandoff endpoint用Stepでないことが未立証

production Procedureのpurpose:

- `make-to-order-production-run.json:8`

は「requested quantityを作り、physical confirmationでactual completionを確立し、instruction / recordと分離する」ことを目的とする。

ledgerのindependent need / TD-1も:

- `seed-ledger.json:6`
- `seed-ledger.json:44`

で同じ独立needを置く。

一方、`PR-06`:

- `make-to-order-production-run.json:50-52`
  - “release the confirmed goods for stocking”

はこのproduction-only purposeの達成には必須ではなく、Handoffのfrom endpointとして自然に追加されたStepなのか、Handoff testのために置かれたStepなのかをcurrent Evidenceでは区別できない。

`TestSecondRealisticReferenceProcedureBodiesStandAloneWithoutTheHandoff`（test:442-454）はHandoff relationを外してもmodel validationが通ることを示すが、PR-06自体のBusiness necessityまでは確認しない。

**Audit再判定前に必要:** PR-06がHandoff relationを除いてもproduction responsibilityの独立Business Actionとして必要である根拠を追加する。説明できない場合はendpoint / Procedure boundaryを再authorする。

### F-5 — minor — same-author / same-protocol bias

second caseはfirst caseと同一作成者・同一protocolでauthorされている。これはsource defectそのものではないが、「二例ある」ことだけをgeneralization Evidenceにできない。

今回の独立reviewによりreviewer independenceは補われたが、authoring independenceそのものは変わらない。したがって二例目をHandoff一般性の証明へ昇格しない現行boundaryは維持する。

## 3. RRB-P1 item-by-item

| item | review | 根拠 / 判断 |
| --- | --- | --- |
| 1 | clear | `order_entry`とは独立したmanufacturing / stocking needがある |
| 2 | clear-with-condition | Handoff relationを外しても二Procedureは成立する。ただしPR-06の独立Business necessityはF-4で未立証 |
| 3 | clear | TEI-original sourceとしてauthorされている |
| 4 | clear | ledgerにdesign-pressure sourceと利用境界が記録される |
| 5 | clear | copiedArtifactsは空、fixtureも外部artifact転載ではない |
| 6 | clear | license analogyを行っていない |
| 7 | clear | sourceBackedFacts / teiOriginalDecisions / unresolvedを分離 |
| 8 | not-clear | Handoff conditionがcount accountability transferを確定する一方、U-4/U-6が未解決。accepted responsibility detailの境界が不整合 |
| 9 | not-clear | U-4/U-6/U-9を残しつつconditionだけがtransfer timingを確定し、false completeness riskがある |
| 10 | clear | scope in/outがbounded |
| 11 | clear | PR-03 / PR-04 / ST-01等でmanual / physical actualityとrecordを分離 |
| 12 | clear | CQ-1 / CQ-2がBusiness×Data / Business×Functionのcross-view question |
| 13 | clear | Application Pattern / Capability / Data identityをfake realizationとして作っていない |
| 14 | **not-achieved** | F-1/F-4によりHandoffが独立needから自然に出たことを十分に立証できない |

**RRB-P1 = 未達。**

P1の独立Business seed自体は有用だが、P1は14項目すべてを満たす契約であるためitem 14等が残る現状では達成としない。

## 4. seed review checklist

1. **Independent value — clear-with-condition**  
   manufacturing / stockingの独立needは有る。PR-06の必要性だけはF-4を解く必要がある。
2. **Provenance — clear**  
   source-backed factとTEI-original decisionsが分かれ、copied artifactも無い。
3. **Bounded scope — clear**  
   purchasing / costing / MRP / shipping / accounting等をoutに置く。
4. **Meaning / unknown ledger — clear-with-blocking inconsistency**  
   ledger自体は良いが、U-4/U-6/U-9とpositive Handoff conditionの間を整合させる必要がある。
5. **Actuality boundary — clear**  
   physical confirmation / arrival checkをApplication resultへ畳んでいない。
6. **Cross-view value — clear**  
   CQ-1がquantity / responsibility ownershipの未解決cross-view pressureを示す。
7. **No fake realization — clear**  
   Application Pattern / Capability / Datasource / fake Data identityを導入していない。
8. **No forced Handoff — not-achieved**  
   F-1/F-4によりendpointとconditionがHandoff用に整形された可能性を排除できない。

## 5. Handoff受け入れ条件

### from / to

`PR-06 -> ST-00`はaccepted source上でexactに参照でき、stale / missing endpointのfail-closedもmechanically確認されている。この構造面はclear。

### adjacency / actor difference

explicit Handoff sourceを取り除いたとき、adjacency / actor difference / positive ObservationからHandoffが合成されないtestは有効である。

ただし「自動導出されない」ことと「authoring時にHandoffを置くためendpoint Stepを設計していない」ことは別である。後者はF-4として残る。

### condition

lossless round-tripはclear。ただしBusiness predicateとしての独立性はblocked。

```text
condition:
  goods have been taken in charge ...

target ST-00:
  take the released goods in charge
```

であり、conditionがtarget action completionを内包する。

### target trigger

trigger:

```text
finished goods released for stocking are waiting to be taken in charge
```

はPR-06後のstateとして自然ではある。しかしconditionがその次のST-00完了を表すため:

```text
PR-06 release
  -> trigger: waiting to be taken in charge
  -> ST-00: take in charge
  -> condition: have been taken in charge
```

となり、Handoff relationの独立Business predicateが薄い。

### system result / physical fact backfill

Command / system resultからのbackfillはしていない。このnegative boundaryはclear。

一方、count accountabilityをST-01 physical reconciliation前に移すpositive meaningはF-2として再authoringが必要。

## 6. O-1〜O-5 judgment

| observation | judgment |
| --- | --- |
| O-1 conditionと受け手Stepの循環 | **妥当 / blocking**。exact-string testではsemantic paraphraseを捕捉できない |
| O-2 数量の責任移転と照合順序 | **妥当 / blocking**。U-4/U-6/U-9とconditionを整合させる必要がある |
| O-3 triggerとfrom Stepの一致 | **一部妥当 / major**。trigger自体は自然なstateだが、ST-00完了conditionと組み合わさることで独立predicateが薄くなる |
| O-4 endpointのためのStep | **妥当 / major**。PR-06はproduction purposeから必須である説明が足りない |
| O-5 同一作成者・同一protocol | **妥当 / minor-methodological**。今回の独立reviewでreviewer biasは緩和するがauthoring independenceは増えない |

## 7. RRB-P2 / P3

### RRB-P2

**未達。**

最小理由はitem 4未確認である。加えてitem 12「Handoff conditionをBusiness continuation predicateとして保持する」もF-1のためpositive sufficiencyが確立していない。

### RRB-P3

**mechanical execution Evidenceとして達成。**

GitHub Release:

```text
tag:
  business-procedure-gate-a-evidence-R-82e4301

target:
  82e430109f8f02944519738045f3d6c0bef4f295

asset:
  tei-lane-c-gate-a-evidence-R-82e4301.tar.gz

size:
  6009 bytes

GitHub release asset digest:
  sha256:3a433835241b82b564e47cb5eb48e63c24f7a5afd3797ae1090afa9add058d7a
```

はreview handoff記載値と一致する。Release bodyもA1-A8 exit 0、A7 39/39、A8 diagnosticsなしを記録する。

ただしreviewerはarchive本体を再download / 展開して`MANIFEST.sha256`を再実行していないため、これは**記録済みactual Evidenceの独立metadata照合**であり、reviewerによるrunner再実行ではない。

## 8. recommended B-F19b status

現状は:

```text
RRB-P1 = 未達
RRB-P2 = 未達
RRB-P3 = 達成（mechanical execution）
```

なのでAcceptance Contract `6に従い:

```text
recommended B-F19b status = Unresolved
```

とする。

F-1/F-2/F-4を修正してP1を達成しても、P2 item 4が残る間は最大でも:

```text
Source-level corpus exists
```

である。

## 9. not-pressured points

- **actorが同じ場合でもexplicit Handoffを否定しないこと**  
  ADR前にminimal pressureを追加することを推奨する。current rule自身がactor差をidentityにしないことを要求しており、positive caseがactor差ありだけなのはcoverage gap。
- **conditionを支えるevent / document identity**  
  今回のblocking原因そのものではない。F-1/F-2修正後も具体的lossが無ければunresolved / reopen triggerとして保持可能。
- **actor / responsibility detailのstructured化**  
  current thin Handoffで具体的lossが示されていないため、ADR前mandatory fieldにはしない。
- **S4-Aで別ProcedureのStep IDが衝突するcase**  
  procedure-scoped addressのgeneric pressureが別testで既に十分ならsecond corpus固有の追加caseは不要。Audit再判定時にgeneric Evidenceの存在だけ再確認する。
- **C1で複数Applicationを跨ぐProcedure**  
  このcorpusはApplication非依存をpressureする目的なので、second-case blockerにはしない。concrete multi-Application lossが出た時のreopen triggerでよい。

## 10. required fixes before Audit re-judgment

1. Handoff conditionをST-00の意味的完了と循環しないBusiness continuation predicateへ再authorする、またはendpoint semanticsを見直す。
2. goods custodyとcount accountabilityの移転時点をST-01との関係で決め、U-4/U-6/U-9とsource conditionを整合させる。
3. PR-06がHandoff fixture都合ではなくproduction responsibilityの独立Business Actionとして必要である根拠を示す。示せなければStep / endpointを再設計する。
4. RRB-P2 item 4をaccepted Data-state / ownership authorityでactual cross-checkするか、P2未達を正式に保持する。
5. actor同一でもexplicit source Handoffを保持できるminimal pressureをADR前に追加することを推奨する。
6. 上記修正後、独立reviewを再実施してからPre-ADR Audit `17を再判定する。

## 11. retained unresolved

次は今回のreviewでfield追加理由にはしない。

- U-5 event/document identity
- structured actor / responsibility detail
- multi-Application Procedure
- full Business Flow / DFD
- generic Role taxonomy

具体的lossが出るまでcurrent unresolved / reopen triggerとして保持する。

## 12. Evidence integrity / execution

```text
evidence integrity check:
  partial

  GitHub Release tag target:
    verified

  release asset size:
    verified

  release asset SHA-256 metadata:
    verified

  archive bytes download:
    not performed

  MANIFEST.sha256 internal verification:
    not performed

test / runner re-executed:
  no

recorded actual Evidence used:
  yes
```

このreviewはPre-ADR Auditの再判定ではない。Contract ADRはAuditが再判定するまでNot Readyを維持する。
