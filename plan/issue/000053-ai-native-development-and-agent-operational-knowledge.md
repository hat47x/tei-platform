# ISSUE-000053: AI-native継続開発とAgent Operational KnowledgeをReferenceで検証する

- 状態: Proposed / phased
- 提案日: 2026-09-18
- 性格: Product validation / AI-native development / operational agent reference
- 主担当要求: RQ-010 / RQ-015 / RQ-018〜RQ-021 / RQ-025〜RQ-030
- 関連ADR: ADR-0022 / ADR-0023 / ADR-0028
- 関連Issue: ISSUE-000027 / ISSUE-000042 / ISSUE-000047 / ISSUE-000052

## 目的

TEIを、

1. 生成AIが継続的にApplicationを構築・変更するための開発知識基盤
2. 人間がgenerated code全体を読まずに意味・影響・例外・Evidenceを理解する認知基盤
3. AI Business Actorが許可された範囲で業務を遂行するOperational Knowledge基盤
4. Domain / Infrastructure知識を別Applicationへ移植する再利用基盤

として利用できるかを、小さいReferenceで段階的に検証する。

このIssueはAI agent frameworkをTEI Coreへ追加するIssueではない。

## 減らす不確実性

### A. AI Builder re-entry

別session / 別agentが既存Applicationへ入るとき、Repository全体を自由探索して意味を再構成しなくても、

- change target
- dependency / impact
- accepted Decision
- Capability / Binding
- native asset
- unresolved
- Evidence

へ機械的に到達できるか。

### B. Human cognitive review

AIが行った変更について、人間がgenerated code全体を読む前に、

- semantic diff
- impact
- exception
- unresolved
- authority change
- Evidence

を高密度なProjectionで把握できるか。

### C. AI Business Actor

accepted Business Procedure / Action / Capabilityを読み、

- 自分が実行可能なAction
- Authorization不足
- human / external handoff
- Command後の必要Observation
- completion / failure / unknown

を判定できるか。

### D. Knowledge portability

一つのReference Applicationから抽出したDomain / Infrastructure knowledgeを別Applicationへ再利用したとき、

- origin
- assumption
- Variation
- dependency
- destination override
- Evidence

を失わず差分適用できるか。

## Phase 0 — 既存資産だけでMachine-readable contextを組む

新しいAI専用Knowledge Storeを作らない。

まずcurrent TEIにある、

- Canonical ID
- Source Index
- Semantic / Application dependency
- Business Procedure cross-view
- Capability
- Binding
- Validation
- Evidence / diagnostic
- Reference Workspace

から、AI Builder / AI Business Actorが必要とするcontextをどこまでderivedに作れるかを棚卸しする。

不足があっても、ただちにCanonical kindを増やさない。

## Phase 1 — AI Builder re-entry Reference

既存Reference Applicationへ小さいchange challengeを設定する。

例:

```text
change request
  -> relevant Canonical subjects
  -> current accepted Decision / constraint
  -> impacted Projection / Capability / native asset
  -> proposed change
  -> Shared Acceptance
  -> reload / conformance
```

比較対象:

- source / Repositoryをほぼ自由探索するbaseline
- TEI structured contextを先に利用する経路

観測候補:

- 読み込む必要があったartifact数
- irrelevant context量
- semantic sourceを取り違えた回数
- change amplification
- acceptance前に検出できたimpact
- re-entry時に人へ聞き直した事項

token量だけをProduct KPIにはしない。

## Phase 2 — Human Review Projection

同じchange challengeから、

- semantic diff
- impacted subjects
- unresolved
- custom / native code touchpoint
- authority / permission change
- validation / conformance result
- Evidence

を一つのreviewable projectionへ出せるかを検証する。

人間がsourceを一切見なくてよいことを成功条件にはしない。

成功条件は、**sourceを見る前に、どこを見るべきかを正しく絞れること**である。

## Phase 3 — read-only AI Business Actor

Business Procedure Referenceが十分に安定した段階で、最初は副作用のないActionを選ぶ。

AI Business Actorが、

1. current Procedure / Actionを読む
2. relevant Data / Constraint / Capabilityを特定する
3. Authorization requirementを読む
4. read-only Capabilityをinvokeする
5. Observation / Evidenceを解釈する
6. next Actionまたはhandoffを返す

経路をReferenceで検証する。

GUI click imitationはbaseline比較に使っても、標準経路にはしない。

## Phase 4 — side-effect Action

read-only境界が成立した後だけ検討する。

必要条件:

- explicit Capability authority
- least privilege Resource
- stale / idempotency / retry boundary
- Command / Observation separation
- expected Evidence
- human escalation / stop
- audit / provenance

Product GateやBusiness Procedure production authorityを先取りしない。

## Phase 5 — Knowledge portability Reference

SUI Reference Adoption等で得た知識を含め、別Applicationへ次のいずれかを移植する。

- Domain pattern
- Validation / Constraint
- Procedure fragment
- Capability contract
- Infrastructure / Provider mapping knowledge

copy-pasteではなく、

```text
origin knowledge
  + destination context
  -> applicability / mismatch / override
  -> accepted reused knowledge
```

を確認する。

## 不変条件

```text
AI understands != AI authorized
AI autonomous != AI authorized
AI reads Procedure != AI owns Procedure
Agent plan != accepted Business fact
Command accepted != Business completed
Operational Projection != Canonical source
AI memory != Application source of truth
knowledge reused != assumption automatically valid
```

## 非目標

- 特定agent frameworkの採用
- 全Business Procedureの自動実行
- GUI computer-useの排除
- 人間reviewの撤廃
- AI専用Canonical copyの新設
- prompt / trajectoryをBusiness authorityへすること
- token削減率を単独の成功KPIにすること

## 完了Evidence

- [ ] AI Builder re-entryでstructured contextの有効性を比較できた
- [ ] human review projectionがCanonical / Evidenceへ逆追跡できた
- [ ] AI Business Actorがread-only Actionとhandoffを正しく扱えた
- [ ] unauthorized Capability invocationをfail closedできた
- [ ] CommandとObservationを分離した
- [ ] side-effectを扱う場合はretry / unknown / Resource境界を満たした
- [ ] knowledge reuseでassumption / Variation差を明示できた
- [ ] derived AI / operational contextを削除してもCanonical Worldを再構成できた

## 着手順

Phase 0 / 1のread-only設計・fixture準備は先行可能。

Phase 3以降のBusiness Actor Referenceは、ISSUE-000042 / 000048 / 000049等のBusiness Procedure contract / production authority Evidenceを侵食しない範囲で進める。

side-effect Actionは、対応するProduct GateとAuthority境界が成立するまで実装しない。
