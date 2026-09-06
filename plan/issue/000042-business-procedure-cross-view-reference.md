# 000042 Business Procedure cross-view Reference

Status: In Progress

## 目的

`order_entry` を使い、Business ProcedureをCanonical schemaへ先に昇格させずに、Data / Function-Interaction / Businessの三視点cross-checkを実行可能なReferenceとして再現する。

正式基線:

- `requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`
- `docs/business-system-design.md`
- `plan/ORDER_ENTRY_BUSINESS_PROCEDURE_REFERENCE_EXPERIMENT.md`
- `plan/UPSTREAM_XTEA_DESIGN_FALLBACK_PROTOCOL.md`
- `plan/XTEA_DESIGN_EVIDENCE_REGISTER_2026-09.md`

## 背景

現行基線は次をすでに固定している。

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Data / Function-Interaction / Business = same accepted meaningを相互牽制する視点
```

一方、`Procedure`をdurable Canonical identityにするか、traceable derived artifactで足りるかは未決である。

このIssueではその判断をschema追加ではなくReference Evidenceから進める。

## 一つの成果

次を同じReferenceで再現できること。

1. `order_entry` の既存Application / Semantic identityを再定義せず参照するBusiness Procedure fixture
2. Application Patternを持たないmanual / external Business Actionが正常に残る
3. Business Action × Application Pattern viewを生成できる
4. Patternが所有するData / Relation / Query / Capability / DatasourceをBusiness Procedureへ重複記述せず、normalized Worldからderivedできる
5. Procedure側で独自に明示されたValidation等のsemantic referenceをexact identityで解決できる
6. `customer_reference` を利用しながらread-only `master` datasourceへwriteを要求する意図的defectを検出できる
7. defect解消のためにProjectionが勝手にwrite accessを追加しない

## 非対象

- `Procedure`の新しい`.tei` syntax
- `Procedure`のtop-level Canonical kind
- Workflow Engine / retry / timer / queue / saga / scheduler
- Business actorからAuthorization Roleを生成する仕組み
- 新しいProduct Requirement ID
- Gate B / Gate Cの完了条件変更
- Business Lifecycle 000038のstate transition実装

## 実装境界

実験toolは既存 `workspace.Load()` が返すnormalized `ir.World` を読み、独自に`.tei`をparseしない。

```text
reference Business Procedure fixture
          +
reference/workspace
          ↓
workspace.Load()
          ↓
normalized World
          ↓
experiment-only cross-view checker
          ↓
resolved references / derived rows / diagnostics
```

Business Procedure fixtureはCanonical Workspaceのsource rootへ追加しない。Reference上のreviewed experiment fixtureとして扱う。

### 三視点は三重管理ではない

Business ProcedureがApplication Patternを参照する場合、Patternがすでに所有している意味をfixtureへコピーしない。

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

checkerはPattern identityをnormalized Worldでexactに解決し、`businessActionPatternMeaningRows`として上記をderivedする。

Business Procedure自身がPatternとは独立に必要とするValidation等だけを`explicitBusinessMeaningRows`へ残す。

これにより、Data / Function / Businessのcross-viewを維持しながら、同じ意味の人手による二重・三重管理を避ける。

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

このdiagnosticは「Customerを編集してはいけない」という業務規則ではない。

意味は、**現在の`order_entry` Application contractではそのinteractionを実現できないため、Projectionだけでwriteを生やしてはいけない**、である。

設計者は次のいずれかを明示的に選ぶ。

- order_entryではreference onlyとする
- acceptedな別Application / Capabilityへ接続する
- 要求が本当に変わったならauthoritative Application / Database Contractへ戻る

## Acceptance

### A1 reference reuse

fixtureが参照するApplication Patternとdirect semantic referenceは、normalized World上のidentityへexactにresolveする。

名称類似だけでmissing referenceを補完しない。

### A2 no fake function

少なくとも一つのBusiness ActionがApplication Pattern / Capabilityなしで正常に保持され、checkerが欠陥扱いしない。

### A3 derived Function matrix

Business ActionごとにApplication Pattern参照または`noApplicationFunction=true`をderived出力できる。

### A4 Pattern-owned meaning is derived

Application Patternが所有するData / Relation / Query / Capability / DatasourceをBusiness Procedure fixtureへ重複記述せず、normalized Patternからderived出力できる。

少なくとも次を確認する。

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

Patternから導けないProcedure-side explicit semantic referenceはexact identityで解決し、名称類似から推測しない。

初回fixtureではBP-03 / BP-05のValidationを対象とする。

### A6 read-only conflict

意図的probeが`BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT`を再現する。

### A7 baseline clean

同じBP-02でrequested access=`read`なら上記diagnosticは発生しない。

### A8 fail closed

fixture / probe JSONの未知field、未対応schemaVersion、別artifact roleを黙って解釈しない。

### A9 no Canonical expansion

このIssue完了だけを根拠に`Procedure` / `Step`をCanonical Worldへ追加しない。

## Procedure identity判定へのEvidence

### 現時点の小さいEvidence

experiment-local Procedure IDだけを変更し、Business Action、Pattern参照、Pattern-derived semantic rows、direct semantic rowsを変えない場合、cross-view解決結果とdiagnosticは変わらないことをtestする。

この結果が成立する限り、**基本cross-view参照だけを理由にdurable Canonical Procedure identityを要求する根拠はまだない**。

### まだ評価が必要な圧力

- fixture / step identityがimpact analysis・review historyで不足するか
- manual residual explanationがregenerationで失われるか
- multiple Procedure -> same Patternのtraceabilityにdurable identityが必要か
- Source / Evidenceへ正式に接続するためCanonical identityが必要か
- Procedure間reuse / handoffでstable identityが必要か

不足が実証されなければ、`Procedure` Canonical化は保留する。

不足が反復して実証された場合だけ、別ADRで最小durable contractを検討する。

## 現在のVerification status

Repository上には次を実装済み。

- Reference fixture
- intentional access-conflict probe
- cross-view checker
- Pattern-owned meaning derivation
- strict decoder
- baseline / conflict / no-fake-function / non-duplication / local-rename / no-name-guess tests
- PR pathに連動したGitHub Actions workflow

GitHub-hosted Actionsは現時点のrunでrunnerが割り当てられず、`runner_id=0` / `steps=[]` のまま終了している。これはcheckoutやGo testへ到達する前の状態であり、コードのPASS / FAIL Evidenceには数えない。

別の隔離type-compatible harnessではcheckerのformatとロジックを確認しているが、actual Repository全体をcheckoutしたConformance Evidenceではない。

したがって本Issueは`In Progress`、PRはDraftのまま維持し、actual Repository runnerでの`gofmt / go vet / go test / baseline / conflict`成功を完了Evidenceとする。

## 他Issueとの境界

- 000038: Business Lifecycle / state transitionを所有。本IssueはProcedure cross-viewのみ。
- 000039: 将来の共通change / dependency analysisを所有。本Issueではfixture内explicit referenceの局所impactだけ扱う。
- 000040: semantic component reuseを所有。本IssueはPattern libraryを実装しない。
- 000008 / 000027: Editor / AI authoring acceptance。本IssueはUI authoringを実装しない。

## 完了時のEvidence

- Reference fixture
- intentional conflict probe
- checker source
- baseline / conflict / strict-decode / Pattern-derived / Procedure-local-rename unit tests
- Reference READMEへの再現手順
- actual Repository runnerによるgreen verification

