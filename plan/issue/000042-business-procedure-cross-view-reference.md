# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry`を使い、Business ProcedureをCanonical schemaへ先に昇格させずに、Data / Function-Interaction / Businessの三視点cross-checkを実行可能なReferenceとして再現する。

正式基線:

- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/business-system-design.md`
- `plan/ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md`
- `plan/UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`
- `plan/BUSINESS_PROCEDURE_IDENTITY_PRESSURE_2026-09.md`

## 背景

現行基線は次を固定している。

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Data / Function-Interaction / Business = same accepted meaningを相互牽制する視点
```

Procedureにはpurpose / trigger / actor context / manual action / judgment等、Application Patternから導出できないBusiness側の意味がある。

したがって、Reference実験のBusiness Procedure fixture全体を`derived artifact`とはみなさない。

一方、`Procedure`をdurable Canonical World identityにするかは未決である。このIssueではschema追加ではなくReference Evidenceから判断する。

## 一つの成果

次を同じReferenceで再現できること。

1. `order_entry`の既存Application / Semantic identityを再定義せず参照するBusiness Procedure **Reference authoring fixture**
2. Application Patternを持たないmanual / external Business Actionが正常に残る
3. Business Action × Application Pattern Matrixをderivedできる
4. Patternが所有するData / Relation / Query / Capability / Datasourceをfixtureへ重複記述せずnormalized Worldからderivedできる
5. Procedure側で独自に明示されたValidation等のsemantic referenceをexact identityで解決できる
6. `customer_reference`を利用しながらread-only `master` datasourceへwriteを要求する意図的defectを検出できる
7. multiple Procedure -> same PatternをBusiness Actionを畳まずtraceできる
8. existing Source Index / Subjectで解けるidentity圧力と、まだ解けないcross-revision identityを分離できる

## Authoring / Derived境界

Business Procedure fixtureは次を明示する。

```text
artifactRole = reference-authoring-fixture
canonical = false
identityStatus = experiment-local
```

fixtureにしかない意味:

```text
purpose
trigger / event candidate
actor / job context candidate
manual / external / unresolved Business Action
```

これらはReference authoring inputであり、Patternから再生成できる派生情報ではない。

一方、次はderived Projectionである。

```text
Business Action × Application Pattern
Business Action × Pattern-owned Data / Relation / Query / Capability / Datasource
cross-view diagnostics
```

checkerはauthoring fixtureへderived結果を逆書きしない。

## 実装境界

実験toolは既存`workspace.Load()`が返すnormalized `ir.World`を読み、独自に`.tei`をparseしない。

```text
Business Procedure Reference authoring fixture
                 +
reference/workspace
                 ↓
          workspace.Load()
                 ↓
        normalized World
                 ↓
 experiment-only cross-view checker
                 ↓
 derived Matrix / diagnostics
```

Business Procedure fixtureはCanonical Workspaceのsource rootへ追加しない。

## 三視点は三重管理ではない

Business ProcedureがApplication Patternを参照する場合、Patternがすでに所有する意味をfixtureへコピーしない。

```text
Business Action
  ↓ explicit Pattern identity
Application Pattern
  ├─ Data
  ├─ Relation
  ├─ Query
  ├─ Capability
  └─ Datasource
```

Procedure自身がPatternとは独立に必要とするValidation等だけをdirect semantic referenceとして保持する。

## 最初のdefect

```text
BP-02 customer reference
  Pattern = order_entry.customer_reference
  Pattern-derived datasource = master
  accepted Application access = read

experiment probe:
  requested access = read-write
```

期待diagnostic:

```text
BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT
severity = architecture-policy
```

意味は、現在の`order_entry` Application contractではそのinteractionを実現できないため、Projectionだけでwriteを生やしてはいけない、である。

Customer更新が本当に必要なら、別Application / accepted Capabilityへ接続するかauthoritative Application / Database Contractへ戻る。

## Acceptance

### A1 authoring residual is preserved

purpose / trigger / actor candidate / manual or external Business ActionをReference authoring fixtureに保持できる。

Patternを持たないBP-04 / BP-09を欠陥扱いしない。

### A2 exact reference reuse

fixtureが参照するApplication Patternとdirect semantic referenceはnormalized World上のidentityへexactにresolveする。名称類似だけでmissing referenceを補完しない。

### A3 derived Function Matrix

Business ActionごとにApplication Pattern参照または`noApplicationFunction=true`をderived出力できる。

### A4 Pattern-owned meaning is derived

Pattern-owned Data / Relation / Query / Capability / Datasourceをfixtureへ重複記述せずnormalized Patternからderived出力する。

```text
BP-01 order_entry.order_record
  -> Data sales.order

BP-02 order_entry.customer_reference
  -> Relation sales.order.customer
  -> Datasource master

BP-03 order_entry.order_lines
  -> Relation sales.order.lines

BP-07 order_entry.submit_order
  -> Capability sales.order.submit
```

### A5 direct Procedure semantic reference

Patternから導けないProcedure-side explicit semantic referenceはexact identityで解決する。初回はBP-03 / BP-05のValidationを対象とする。

### A6 read-only conflict

intentional probeが`BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT`を再現し、Projectionがauthorityを拡張しない。

### A7 baseline clean

probeなしbaselineにはblocking diagnosticがない。

### A8 fail closed

fixture / probe JSONの未知field、未対応schemaVersion、別artifact roleを黙って解釈しない。

### A9 multiple Procedure pressure

primary Procedureとfollow-up Procedureが同じ`order_entry.order_record`を参照しても、二つのBusiness Action usageとして区別できる。

### A10 identity pressure remains layered

- experiment-local Procedure ID renameだけではcross-view meaningを変えない
- pure file moveではSource Index Artifact fingerprintを保てる
- whitespace-only changeではfingerprintが変わるため、fingerprintをsemantic Procedure identityとして流用しない
- existing path-independent Pattern Subjectから複数Procedure artifactへのderived referenceを表現できる

### A11 no Canonical expansion

このIssue完了だけを根拠に`Procedure` / `Step`をCanonical Worldへ追加しない。

## Procedure identity判定へのEvidence

現時点では、基本cross-view、multiple Procedure -> same Pattern、pure file move、Pattern reverse referenceのためだけならCanonical Procedure identityを要求するEvidenceはない。

残る圧力は次である。

- 内容変更を伴うcross-revision review history
- instruction wording changeとsemantic impactの区別
- manual residual explanationのregeneration耐性
- Procedure間handoff / reuse
- accepted authoring sourceへ昇格するときのstable identity

既存Subject / Source Index / revision provenanceで不足する反復Evidenceが得られた場合だけ、別ADRで最小durable contractを検討する。

## 000039との境界

000039が将来所有するrevision-aware semantic diff / dependency graph / Scenario propagationを本Issueへ実装しない。

本IssueはReference fixture内のexplicit relationと既存Source Indexの性質を使う局所pressure testに限定する。

## 現在のVerification status

Repository上には次を実装済み。

- primary / follow-up Reference authoring fixture
- intentional access-conflict probe
- cross-view checker
- Pattern-owned meaning derivation
- strict decoder
- baseline / conflict / no-fake-function / non-duplication / local-rename / multiple-Procedure / Source Index pressure tests
- PR pathに連動したGitHub Actions workflow

GitHub-hosted Actionsはrunnerが割り当てられず、`runner_id=0` / `steps=[]`のまま終了している。checkoutやGo testへ到達する前なのでコードのPASS / FAIL Evidenceには数えない。

接続済み開発端末もofflineである。

したがって本Issueは`In Progress`、PRはDraftのまま維持し、actual Repository runnerでの`gofmt / go vet / go test / baseline / conflict`成功を完了Evidenceとする。

## 他Issueとの境界

- 000038: Business Lifecycle / state transitionを所有。本IssueはProcedure cross-viewのみ。
- 000039: 共通change / dependency analysisを所有。本Issueは局所pressure testのみ。
- 000040: semantic component reuseを所有。本IssueはPattern libraryを実装しない。
- 000008 / 000027: Editor / AI authoring acceptance。本IssueはUI authoringを実装しない。

## 完了時のEvidence

- Reference authoring fixtures
- intentional conflict probe
- checker source
- baseline / conflict / strict-decode / Pattern-derived / Procedure identity / Source Index pressure unit tests
- Reference READMEへの再現手順
- actual Repository runnerによるgreen verification
