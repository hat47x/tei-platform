# ISSUE-000076: Brownfield REF-01で段階的な意味立ち上げを実証する

- 状態: Ready
- 起点: GitHub Issue #915
- 関連: ISSUE-000027 / ISSUE-000042 / ISSUE-000046 / ISSUE-000047
- 関連RQ: RQ-002 / RQ-004 / RQ-006 / RQ-015 / RQ-027 / RQ-031〜RQ-033

## 目的

既存システムへTEIを導入するとき、最初からCanonical meaning一式を手入力したり、Repository全体をAIへ再解釈させたりせず、**一件の変更要求に必要な範囲だけを既存SourceとAccepted meaningから辿り、現在のblockerを根拠付きで説明できること**をReferenceで実証します。

本Issueは新しいImporter、Knowledge Store、Proposal schema、Acceptance engineを作るためのものではありません。

すでにmainにある次の境界を、一つのbrownfield変更調査として接続します。

- Source Index / normalized World
- `tei knowledge-context`
- deterministic `knowledge-qualify`
- qualification Evidence
- Business Procedure cross-view checker
- Shared Acceptance Boundary

GitHub Issue #915は研究・価値仮説の議論を保持します。本ファイルは、最初に実装するReference work unitの正本です。

## 防ぎたい失敗

既存資産から意味を立ち上げる処理では、検索結果が多いほど正しいわけではありません。

特に次を防ぎます。

1. 変更要求に関係するSourceを見つけたという理由だけで、無関係な認証・認可・Capabilityまで因果列へ混ぜる。
2. `customer_reference`がread-onlyであるため書けない状況を、存在しないwrite CapabilityやAuthorization resourceで自動補完する。
3. Source上の現在挙動を、そのまま今後も保持すべきAccepted meaningへ昇格する。
4. `knowledge-qualify=applicable`を変更のAcceptanceやwrite authorityとみなす。
5. Repository全体のrevisionをfreshness根拠にして、無関係な変更で調査Evidenceを無効化する。
6. 一つのsynthetic成功を、実案件のbrownfield導入価値として一般化する。

## 最初のReference変更要求

既存fixtureをそのまま使います。

`reference/business-system-design/order-entry-customer-edit-conflict.json`

要求は次です。

```text
order-entry.order-handling / BP-02
  order_entry.customer_reference
  required access = read-write
```

このfixtureはCanonical authorityではなく、変更要求を模したexperiment-only candidateです。

現在のAccepted側は次です。

`reference/workspace/applications/order-entry/patterns/customer-reference.tei`

```text
pattern order_entry.customer_reference reference
  relation   sales.order.customer
  datasource master
  contract   master_data.customer
  query      find_by_id
```

Business Procedure checkerはPattern所有Applicationのdatabase requirementへ戻り、現在の`master` accessが`read`であることから次を出します。

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

このdiagnosticがREF-01Bの最初のblockerです。

## REF-01AとREF-01Bを分ける

### REF-01A — native authentication / authorizationを再利用する

既存の次のReferenceは、native assetをTEI標準RBACへ書き換えずSemantic Capabilityへ接続できることをすでに検証しています。

- `reference/legacy-auth/authenticate-function.json`
- `reference/legacy-auth/users.json`
- `reference/legacy-auth/authorization.json`
- `internal/security/legacyauthorization`
- `docs/conformance/legacy-authorization-v1.md`

この経路の価値は、allow / deny / execution errorとSource来歴を保ったままlegacy ruleを利用できることです。

customer edit probeのblockerを説明するために、REF-01AのAuthorizationを因果列へ追加しません。

具体的なApplication action / resourceがAccepted changeとして定まった後にAuthorization変更が必要なら、別candidateとして扱います。

### REF-01B — desired accessとAccepted Application contractの差を止める

customer edit probeでは次だけを辿ります。

```text
Desired candidate
  BP-02 / order_entry.customer_reference / read-write
        ↓
exact Application Pattern
  order_entry.customer_reference
        ↓
Accepted Pattern meaning
  datasource = master
        ↓
Pattern owning Application requirement
  master access = read
        ↓
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

この時点で不足するwrite CapabilityやAuthorization resourceを生成しません。

## 最初のコードwork unit

最初のコード変更は、Product APIや新CLIを増やす前に、既存機構がこの因果列を一貫して再現できることを一つのpressure testで固定します。

配置候補:

`tools/reference-business-procedure-check/brownfield_ref01_pressure_test.go`

### Step 1 — exact subjectからAccepted contextへ入る

Reference Workspaceを読み、Source Indexを構築します。

対象subject:

```text
kind = application-pattern
id   = order_entry.customer_reference
```

`knowledgecontext.Build`と`QualifyWithSources`を既存実装のまま利用します。

requirements:

```text
expected kind = application-pattern
application   = order_entry
required dependency
  kind = database-contract-query
  id   = master_data.customer/find_by_id
```

期待:

- status = `applicable`
- candidate Evidenceに`applications/order-entry/patterns/customer-reference.tei`が含まれる
- dependency Evidenceに`databases/master-data/contracts/customer.tei`が含まれる
- source fingerprintが空でない
- qualificationがAcceptance / write authorityを持たない

新しいbrownfield用IndexやKnowledge storeを作りません。

### Step 2 — existing diagnosticへ接続する

既存の次を読みます。

- `order-entry-business-procedure.json`
- `order-entry-customer-edit-conflict.json`

既存の`run` / `evaluate` / `applyAccessProbe`経路を使い、blocking diagnosticが次の一件だけであることを確認します。

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
```

さらにderived rowから、少なくとも次の因果を確認します。

- Step = `BP-02`
- Pattern = `order_entry.customer_reference`
- Datasource = `master`
- Pattern relation = `sales.order.customer`
- requested access = `read-write`
- accepted datasource access = `read`

このテストのためだけにBusiness Procedure fixtureへPattern-owned meaningを重複記述しません。

### Step 3 — 関係のないAuthorityを捏造しない

BP-02のPatternには現在、customer update用Capabilityはありません。

pressure testでは、調査結果へ次を追加しないことを確認します。

- synthetic write Capability
- `edit + order_entry.customer_reference`のsynthetic Authorization resource
- Business actorからAuthorization roleへの自動mapping
- write可能なdatabase requirement

「見つからない」ことを一般的な禁止へ読み替えるのではなく、**現在のblockerを説明するのに不要な意味を生成しない**ことを確認します。

### Step 4 — causal negative/positive pair

scratch workspaceだけで、Pattern owning Applicationの`master` accessを`read-write`へ変更します。

同じBusiness Procedure / same Pattern identity / same desired probeを再評価し、`BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT`が消えることを確認します。

これはread-writeをmainへ採用するAcceptanceではありません。

目的は、diagnosticがcustomer編集という文言へhard-codeされているのではなく、**現在のAccepted Application contractとの差**から導出されていることを確認することです。

scratch変更から新しいCapabilityやAuthorizationを自動生成しません。

### Step 5 — Source非変更

read-only調査経路では、Reference Workspaceのauthoritative source byteを変更しません。

pressure testはscratch copyを必要とするcaseを除き、元Referenceをread-onlyで扱います。

## Shared Acceptanceとの接続

REF-01のread-only調査がblockerまたは変更候補へ到達しても、その結果をCanonicalへ直接writeしません。

変更案を実際に受理する段階は既存境界を使います。

```text
read-only brownfield investigation
        ↓
candidate
        ↓
qualification Evidence attachment where applicable
        ↓
Shared Acceptance Preview
        ↓
current Source IndexでEvidence再確認
        ↓
human Accept / Reject
        ↓ Accept only
authoritative source write
        ↓
reload
        ↓
Validation / Conformance
```

PR #918で、qualification EvidenceはShared Acceptance Preview時にcurrent Source Indexへ戻って再検証されます。

したがってREF-01専用のAcceptance engine、Evidence freshness counter、repository-wide revisionを追加しません。

## Product Evidence

最初のReferenceで測る対象はmodel accuracyではなく、変更調査の作業量です。

比較A:

- Repository / Referenceを人またはagentが直接探索する

比較B:

- exact Business Procedure / Pattern identityから既存knowledge contextとcheckerを使う

最低限記録する値:

- 開いたSource / artifact数
- 手動検索query数
- 最初のblocking Conflict / Unresolvedへ到達するまでの操作数
- 変更可能性の判断までの時間
- Sourceに無いCapability / Authorization / relationを補完した件数
- 最終diagnosticからauthoritative Sourceへ再到達できるか
- 二回目の関連変更で既存Evidenceを再利用できたか

Bが速くても、根拠の無い意味を増やした場合は失敗とします。

## Acceptance

最初のwork unitの完了条件:

- [ ] `order_entry.customer_reference`をexact identityで探索できる
- [ ] candidate source fingerprintへ戻れる
- [ ] `master_data.customer/find_by_id`のrequired dependency Evidenceへ戻れる
- [ ] customer edit probeが`BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT`へ到達する
- [ ] conflictの因果が`Pattern -> datasource master -> owning Application access read`として再構築できる
- [ ] synthetic write Capabilityを生成しない
- [ ] synthetic Authorization resourceを生成しない
- [ ] scratchで`master=read-write`に変えた場合だけ同diagnosticが消える
- [ ] read-only調査で元Reference sourceを書き換えない
- [ ] qualificationをAcceptance / write authorityへ昇格しない
- [ ] Shared Acceptance以外のwrite経路を追加しない

## 次段階へ進む条件

上記pressure testがexact-head full `tools/repository-health-check.sh`を通過してmainへ統合された後、初めてProduct-facing surfaceを検討します。

候補は、新しい巨大なbrownfield commandではなく、既存のmachine-readable出力を一つの変更調査として扱いやすくする薄いorchestrationです。

次段階でも次を守ります。

- Source observationとAccepted meaningを同じstatusへ畳まない
- AI confidenceをAuthorityにしない
- whole-repository reverse documentationを前提にしない
- MMI / SUIを必須依存にしない
- native assetを不要にTEI形式へコピーしない

## 実装Gate

本Issueのコード変更はTEIの通常Gateに従います。

- latest mainから既存lane競合を確認する
- code変更PRはDraftで開始する
- exact-head full `tools/repository-health-check.sh` PASS
- pre/post clean state
- Gate EvidenceをPRへ記録する
- PASSなしでmainへ統合しない

GitHub Actionsの利用可否はこのGateの代替にしません。

## 非目標

- 全legacy systemの自動仕様化
- 新しいKnowledge Graph
- 新しいProposal schema
- brownfield専用Acceptance engine
- legacy Authorizationとcustomer-edit blockerの自動統合
- write Capabilityの自動発明
- Application contractの自動変更
- reverse-engineered behaviorの自動Canonical化
- production brownfield価値の実証済み主張

## 関連

- GitHub #915
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000042-business-procedure-cross-view-reference.md`
- `plan/issue/000046-studio-shared-acceptance-authoring-loop.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `docs/conformance/legacy-authorization-v1.md`
- `reference/business-system-design/README.md`
