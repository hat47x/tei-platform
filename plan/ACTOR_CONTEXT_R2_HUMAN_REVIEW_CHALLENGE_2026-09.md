# Actor Context R2 — Human Review Challenge: customer edit proposalをコード精読前に判断する

- Status: ISSUE-000053 / Phase 2 challenge baseline / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R1_AI_BUILDER_CHANGE_CHALLENGE_2026-09.md`
- Product Gate impact: None
- Runtime impact: None
- Production schema impact: None

## 1. 目的

生成AIがApplication変更の大部分を担う場合、人間のreviewをgenerated code全量の精読へ依存させると、TEIが目指す認知効率を実現できない。

R2では、R1と同じ要求、

> 注文画面で参照している顧客情報を、その場で編集・更新できるようにする。

について、人間がsource codeを読む前に、

- 何を変えようとしているか
- current accepted / Reference meaningは何か
- どこで矛盾しているか
- 何がまだ未決定か
- どのauthorityを変更する可能性があるか
- どのsourceへ掘り下げればよいか

を高密度なProjectionで理解できるかをPressureする。

## 2. Human Reviewで最初に見せるもの

R2のfirst-pass reviewは、次の順序を基本とする。

```text
Intent
  customer_referenceからcustomerを編集・更新したい

Current meaning
  BP-02 = associated customerをconfirm
  Pattern = reference
  Relation = sales.order.customer

Current technical authority
  datasource master
  access read
  Contract master_data.customer
  query find_by_id
  Contract writes = none

Detected conflict
  direct read-write candidate
  vs
  accepted master access = read

Unresolved
  customer update ownership
  write authority path
  Business responsibility expansion
  Capability or handoff design

Drill-down
  exact TEI / Reference source
```

この時点では生成コードの詳細はprimary review surfaceにしない。

必要な場合だけnative / generated sourceへ掘り下げる。

## 3. Human Review Projectionが答える問い

Projection自身が設計判断を行うのではない。

人間が少なくとも次を判断できる情報を揃える。

### Q1 — これはUIだけの変更か

No。

current source上、direct mutationにはApplication access / Contract等のauthority変更が関係する可能性がある。

### Q2 — 現在どこまで確定しているか

確定:

- current Patternはreference
- current datasourceはmaster
- current accessはread
- current Contractはread / find_by_id
- direct read-write candidateはconflict

未確定:

- customer updateのowner
- write経路
- Business responsibility
- Capability / handoff方式

### Q3 — どのsourceを読むべきか

最初からRepository全体を読むのではなく、最低限:

1. order_entry Application datasource definition
2. customer_reference Pattern
3. master_data.customer Contract
4. BP-02 Business Procedure Reference
5. intentional conflict Evidence

へdrill-downできる。

### Q4 — 人間が何を承認することになるか

R2 Projectionはこれを自動決定しない。

候補によって承認対象が変わる。

例:

- read-only + handoffならhandoff / UX / owner contract
- direct writeならApplication access / Contract / Authorization / Capability / Business scope
- separate CapabilityならCapability ownership / Binding / Authority / Observation

したがって`Approve`ボタンへ一つの巨大なcode diffを押し込む設計は避ける。

## 4. Review packetのconceptual shape

production schemaではない。

```text
HumanReviewProjection

  intent
  reference_fingerprint
  required_input_fingerprints[]

  current_meaning[]
    business
    semantic
    application
    database

  detected_conflicts[]
    conflict
    authority_ref
    evidence_ref

  unresolved_dimensions[]

  potentially_affected_authorities[]

  review_questions[]

  drill_down_refs[]

  authority_limits[]
```

重要なのは、summary文そのものではなく各項目がauthority / Evidenceへ戻れること。

R2ではR1が確定したchallenge-local `required_input_fingerprints[]`をそのまま継承する。Human Review用にsource pathを別管理して第二のfreshness基線を作らない。

```text
R1 decisive inputs
  -> R2 review drill-down

review summary
  != independent source inventory
```

## 5. potentially affected authorityと「変更必須」を分ける

direct write案を採る場合、

- Application datasource access
- Database Contract
- Authorization
- Capability
- Business responsibility

等が再検討対象になり得る。

しかし、

```text
potentially affected
  != must change
```

である。

別ownerへのhandoffを採れば、order_entryのDatabase Contractは変更しない可能性がある。

Human Review Projectionは、特定解決案を先に選んで「変更必須一覧」を生成しない。

## 6. human cognitive efficiency

R2で測りたいのは文章の短さだけではない。

### 良い圧縮

- irrelevant sourceを隠す
- exact authorityへ戻れる
- known / unresolvedを分ける
- conflictを先に見せる
- 変更範囲の候補を示す
- native codeが必要な箇所だけdrill-downする

### 悪い圧縮

- AIの自然言語要約だけを信じさせる
- unresolvedを省略する
- authority sourceを消す
- 「危険そう」「たぶん大丈夫」のscoreへ畳む
- code diffを見なくてよいという名目でEvidenceも隠す

```text
less information shown
  != higher cognitive efficiency

higher cognitive efficiency
  = less irrelevant information
    + preserved decisive information
    + cheap drill-down
```

## 7. R1との関係

R1はAI Builderがpremature codingを避けられるかを見る。

R2は同じstructured factsを人間のreviewへ再利用する。

```text
same accepted meaning
  ├── Builder context
  │     -> prevent premature implementation
  └── Human review
        -> focus decision attention
```

これがstructure-to-application leverageの一例になる。

同じ`master access = read`を、

- cross-view validation
- AI Builder context
- Human Review
- future Agent Operational Projection

へ再利用する。

## 8. R2 test-only pressure

最初のsourceではR1 constraint projectionを入力として、

- intent
- current meaning
- conflict
- unresolved dimensions
- potentially affected authorities
- review questions
- exact drill-down refs
- authority limits

を組む。

testでは少なくとも次を守る。

1. conflictを保持する
2. Contract writesなしを保持する
3. unresolvedを削らない
4. review questionをanswer / decisionへ変換しない
5. potentially affected authorityをmust-changeへ変換しない
6. exact sourceへdrill-downできる
7. R1のReference / input fingerprintsを失わず継承する
8. drill-down refsを同じfingerprint input集合から導出し、固定文字列の第二source一覧を作らない
9. Approved / Rejected / Safe / Score等を持たない
10. generated codeをreview authorityにしない

## 9. R2で新設しないもの

- production HumanReviewProjection schema
- review score
- auto approval
- AI judge
- mandatory code-free review
- one-click authority expansion
- customer update ownership
- write Capability
- Database Contract変更
- Business Procedure変更

## 10. Evidence高さ

```text
Design Evidence
  review information boundary

Source Evidence
  test-only projection

Executable Evidence
  actual tests

Human Evidence
  human can orient / decide with less irrelevant reading

Adoption / Outcome
  repeated AI-generated changesでreview burden / semantic errorが減る
```

R2 source実装だけではHuman Evidenceを主張しない。
