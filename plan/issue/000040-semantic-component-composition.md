# ISSUE-000040: 業務の意味資産を探索し、明示参照で組み立てる機能を提供する

- 状態: Proposed
- 提案日: 2026-09-04
- 提案順位: 3（根幹機能の新規提案内）
- 本格着手条件: Gate C通過後

## 提案する根幹機能

既存のData、Relation、Validation、Capability、Application Patternを探索し、必要な依存と適用条件を確認して、新しいApplicationへ明示参照で組み込める機能を提供します。

利用者は複数案件を扱う開発者とAIです。定義の所在を知る人しか再利用できなければ、業務の意味を毎回作り直し、修正も繰り返すことになります。保持する意味と契約を、安全に組み立てられる知識資産へ進めます。

最初はRepository内を対象に、人とAIが同じ候補と適用できない理由を取得できる境界を設けます。AI providerやEditorの完成は前提にしません。

## 既存実装と不足

2026-09-04時点で、次の基盤はすでにあります。

- `internal/semantic/model.go`: Namespace、Type、Data、Relation、Capability、Validation。
- `internal/application/model.go`: `SemanticUses`、Validation参照、5種類のApplication PatternとData / Relation / Query / Capabilityの明示参照。
- `internal/index/index.go`: Semantic要素、依存、Application・Database・Binding等の派生索引。
- `internal/validation/application_pattern.go`等: 参照先の存在、所有Application、Pattern種別、Datasource、Relation等の整合性検査。
- `cmd/tei/main.go`: `index`、`source-index`等の検査入口。

追加するのは、目的から候補を絞り、依存と不足Bindingを説明し、適用候補を共通Validationへ渡す一続きの機能です。要素一覧や直接参照だけでDomain Componentの組み立てが成立したとは扱いません。

## 最小の機能範囲

### 1. 既存資産の探索と説明

意味の種類、Canonical ID、入出力型、Relation、利用Application等から候補を取得します。検索条件はReferenceで必要なものに絞ります。

候補から正本、関連する意味、利用例、検証結果へ辿れるようにします。索引は正本から再生成し、名前の類似を意味の同一性とは扱いません。自然言語検索を加えても、検索順位を適合性の根拠にはしません。

### 2. 依存と適用条件の検査

候補が要求するType、Data、Relation、Validation、Capability、Contractを辿り、必要な依存を説明します。既存索引・Validationを用い、不足する関係だけを追加します。依存経路と由来の共通解析は[ISSUE-000039](000039-business-change-rehearsal.md)へ集約し、本Issueは候補の検索条件と適用先の選択を担当します。共通境界が必要な範囲では000039の最小解析部分を先に利用し、変更リハーサル全体の完成は前提にしません。

型、所有境界、Contract、read / read-write、必要なBindingを検査し、適用可能、不適合、情報不足を区別します。情報不足なら不足項目を返し、既存Bindingや権限を名前から推測して補いません。実行権限やResource accessは別の既存境界で確認します。

### 3. Applicationへの明示的な組み立て

共有する意味は同じCanonical IDを参照し、Application固有のPattern、Datasource alias、Bindingは対象Applicationの選択として表します。既存ApplicationのPatternを共有正本へ昇格させず、適用先が所有するPatternを作ります。native SQLは正本を維持して明示参照し、共有する意味と適用先の意図を分けます。

組み立て結果は変更候補として提示し、採用境界をADR-0022 / [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)と共有して通常のsourceとValidation / reloadへ接続します。候補作成と検査はread-onlyで利用でき、stale revisionは黙って適用しません。

`Domain Component`の専用schemaが必要か、既存Namespaceと明示参照で足りるかはReferenceから決めます。導出できる一覧を人に二重管理させません。

## Knowledge Portabilityから受け取る境界

別Applicationへの組み立てでは、ISSUE-000053 Phase 5のauthority / provenance境界を利用する。

詳細は `plan/KNOWLEDGE_PORTABILITY_HANDOFF_2026-09.md` を参照する。

組み立て時は次を区別する。

```text
shared authority reference
  Semantic / compatible Contract / Capability等

destination-owned choice
  Application Pattern / datasource alias / Binding / Environment binding等

provenance-only relation
  origin Application / Patternを参考にした履歴
```

Application-owned Patternをoriginからshared authorityとして継承せず、destinationが自分のPattern identityをauthorする。

同様に、originのdatasource aliasやEnvironment bindingをdestinationへ暗黙copyしない。

第二Application Referenceでは、通常の適合性検査に加え、少なくとも次のchange propagationを確認する。

- shared Semantic changeは実依存consumerをrequalification対象にする
- Contract major changeはcompatible-major dependencyを再評価する
- origin Pattern changeはdestination local variationへauto-applyしない
- destination Pattern changeはoriginへback-propagateしない
- origin datasource alias / Environment binding changeはdestinationへ伝播しない
- unrelated origin Application changeでdestinationをstale-allにしない

これらを検証する前にinheritance / override graphやauto-sync protocolをproduction化しない。

## 第二Application Pre-Gate Contract

Gate C前に第二Application sourceを先取りせず、着手時に必要な意味境界だけを固定する。

正本:

- `plan/SECOND_APPLICATION_REFERENCE_PRE_GATE_CONTRACT_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/second_application_reference_pre_gate_pressure_test.go`

最初のscenario labelは `repair_intake-test-only` とするが、これはApplication名の採用ではない。

Pre-Gateでは必ず、

```text
requiredGate = Gate C Ready

canonicalApplicationCreated = false
sourceAuthoringAttempted = false
acceptanceAttempted = false
environmentBindingCreated = false
targetImplementationStarted = false
```

を維持する。

Gate C後の最初のread-only Referenceでは、

- shared authority: `customer.customer`, `master_data.customer ^2`
- destination-owned: Application identity / Business Relation / Application Pattern / datasource alias
- Environment-owned: development / production binding

を分離する。

また、shared dependency change / origin provenance drift / destination-local change / unrelated origin changeの期待値をPre-Gateで固定するが、ISSUE-000039のimpact engineを前倒し実装しない。

### Pre-Gate schema sufficiency

第二Application専用schemaを先に作らず、既存Application / Pattern / Environment grammarで最小構成を表現できるかをtest-onlyにPressureする。

正本:

- `plan/SECOND_APPLICATION_SCHEMA_SUFFICIENCY_PRESSURE_2026-09.md`

test-only source:

- `tools/reference-business-procedure-check/second_application_schema_sufficiency_pressure_test.go`

初回read-only Referenceについてcurrent grammarで、

- shared Semantic
- compatible Database Contract requirement
- destination-owned datasource alias
- destination-owned reference Pattern
- shared Contract Query / input mapping
- destination Environment binding

を表現できるなら、専用Knowledge Package / inheritance / override / sync構文は追加しない。

追加schemaの必要性は、actual second ApplicationのAcceptance / reload / change propagationからEvidenceが出た場合に再評価する。

## 第二Applicationでの受入検証

`order_entry`と異なるApplicationを一つ用い、意味資産の組み立て機能を受入検証します。

一業務、一つの既存read-only query、Pattern 2〜3種類に絞ります。修理受付の照会等から共有する意味と固有条件を選びます。固有条件を共通化のために削らず、再利用できない理由も返します。

Driverと第一Web Targetで適用結果を確認します。閉じたReference環境で共有部分と固有部分をそれぞれ変更し、両Applicationへ伝わるべき変更と、伝わってはいけない変更を検査します。必要なら`ISSUE-000039`の変更影響機能を利用します。

## 完了条件

- [ ] 同じRepository revisionと検索条件から、同じ候補identityと根拠を取得できる。
- [ ] 各候補から正本、依存、利用例へ辿れ、存在しないEvidenceは未取得と示す。
- [ ] 必要な依存と適用先の選択を分離し、不足Binding、型不一致、所有境界違反を具体的に説明できる。
- [ ] 正常例に加え、未知参照、意味の異なる類似候補、適合しないContract、未指定Binding、stale revisionの拒否・未確定例を用意した。
- [ ] 第二Applicationが同じCanonical IDを参照し、固有PatternとBindingを明示した構成を生成・検査できる。
- [ ] sourceへ反映した構成をAIや探索索引なしでreloadし、同じ意味とConformance結果を再現できる。
- [ ] Driverと第一Web Targetで同じ業務の意味を再現し、共通変更の伝播と固有条件の保持を確認した。
- [ ] 索引を削除・再生成しても意味が変わらず、元のnative queryが正本として維持される。
- [ ] 追加schemaの要否と、共有できた範囲・共有しなかった理由をReferenceに基づいて記録した。

測定対象は、参照できた同一identity、依存の検出範囲、不適合・未確定の検出、変更伝播、手で重複管理した箇所とします。コードのコピー率や生成量だけで再利用成功を判定しません。

## 上位要求と既存Issueとの関係

関連要求はRQ-001〜RQ-004、RQ-008、RQ-012、RQ-014〜RQ-017、RQ-019、RQ-021〜RQ-023です。Workstream Bの意味知識をCのApplicationとHの開発支援へ接続します。

ADR-0017 / ADR-0018に従い、Patternは各Applicationが所有します。複数ApplicationとTargetのEvidenceなしに標準CatalogやKernelへ昇格させません。

- [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)はAI Proposalと採用の境界、本Issueは採用前に探索・組み立て・適合性検査できる意味資産の機能です。
- [ISSUE-000022](000022-external-master-data-physical-query-evidence.md)のactual Oracle Evidence、[ISSUE-000003](000003-legacy-authentication-compatibility-reference.md)の既存認証、[ISSUE-000026](000026-target-capability-execution-profile-boundary.md)の第二Targetを再実装せず、その完了条件も代替しません。
- `ISSUE-000039`は変更による影響、本Issueは新しい利用先への組み立てを扱います。共通する索引や追跡情報は共有します。

## 非目標

- 巨大なpackage manager、Marketplace、remote registry、vector databaseの先行構築
- 5種類のPatternを恒久標準へ固定すること
- SQLや既存コードの全面変換、第二ApplicationのフルCRUD化
- AI固有の正本、暗黙のBinding選択・権限付与・本番実行
- Gate B / Cへの条件追加や、既存主線の置き換え

## 関連文書

- [ISSUE-000038](000038-business-lifecycle-semantics.md)の新Lifecycleと、[ISSUE-000039](000039-business-change-rehearsal.md)のリハーサル全体の完成は前提にしない。必要な依存解析は000039の最小境界を利用し、既存の意味資産で最初の組み立てを実証する。
- `product/value/TEI_PRODUCT_VALUE.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/MASTER_PLAN.md`
- `plan/adr/0017-application-pattern-ownership.md`
- `plan/adr/0018-application-pattern-authoring-source.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
