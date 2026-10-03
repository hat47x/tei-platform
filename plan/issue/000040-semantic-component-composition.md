# ISSUE-000040: 業務の意味資産を探索し、明示参照で組み立てる機能を提供する

- 状態: In Progress — S5 / S6 / S7完了（2026-09-21）。S8 Driver scenario実施（2026-09-22）。残るTarget Projectionは未評価。Semantic Extension Reference candidateを実装中（2026-10-04）
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

## Gate C後の開始パケット

Gate C Ready後の第二Application Referenceは、探索・Portability・composition・Acceptanceを別branchへ分裂させず、一つのISSUE-000040 workstreamとして開始する。

Entry preflight:

- `plan/GATE_C_SECOND_APPLICATION_ENTRY_PREFLIGHT_2026-09.md`
- `tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go`

Gate CのReady labelだけでは開始せず、`B/R/T1`、Semantic / Functional / Visual Evidence、Target差分が揃っていることを確認する。

開始契約:

- `plan/GATE_C_SECOND_APPLICATION_START_PACKET_2026-09.md`

開始時にはGate Cの、

- Handover Baseline `B`
- baseline revision `R`
- First Web Target build `T1`
- Semantic Conformance Evidence
- Interaction / Functional Evidence
- Visual Fidelity Evidence
- Target差分 / unresolved

をentry recordへ固定する。

その後、

```text
ISSUE-000047 qualification
  -> ISSUE-000053 Phase 5 portability relation
  -> ISSUE-000040 composition candidate
  -> Shared Acceptance
  -> destination source
  -> Environment Binding Proposal
  -> Driver / First Web Target
  -> change propagation Evidence
```

の順で進める。

同じ第二Applicationについてexploration / Pattern / Environment / Driverを別remote branchへ分割せず、意味単位のcommitをmainへ収束させる。branch / PRの有無をProduct / Issue / Acceptance状態として扱わない。

## 第二Application actual progress — S0〜S5

Gate C Readyを受け、actual second-Application workstreamを開始した。

S0:

- `plan/SECOND_APPLICATION_GATE_C_ENTRY_2026-09.md`
- `tools/reference-business-procedure-check/second_application_gate_c_actual_entry_pressure_test.go`

S1 / S2:

- `plan/SECOND_APPLICATION_S1_S2_QUALIFICATION_PORTABILITY_2026-09.md`
- `tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go`

current qualification / portability result:

- shared Customer Semantic / Customer ID Type = linked reuse
- Customer Contract / `find_by_id` Query = conditional linked reuse
- `order_entry.customer_reference` = adopted local variation provenance
- origin development / production binding = rebind-required

S3:

- `plan/SECOND_APPLICATION_S3_DESTINATION_DECISION_PACKET_2026-09.md`

でdestination-owned candidateとして `repair_intake` / `repair_intake.intake.customer` / `repair_intake.customer_reference` / `customer_master` を明示した。これは `repair_intake-test-only` scenario labelからの自動昇格ではない。

S4:

- `plan/SECOND_APPLICATION_S4_COMPOSITION_CANDIDATE_2026-09.md`

でexisting Application / Pattern grammarだけを使うcomposition Proposalを固定した。

S5:

- `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md`
- `plan/SECOND_APPLICATION_S5_FRESHNESS_CAPTURE_2026-09.md`

でexact Proposal identity、Semantic Diff、current shared-authority freshnessまで固定した。

2026-09-21、mainへ決定論的Shared Acceptance Boundaryの最初のincrement（`tools/reference-shared-acceptance`）が入ったため、Second Application S5を再照合した。

current Referenceは `add-validation` / semantic-only diff / semantic-only containmentに限定され、Semantic + Application + Patternのthree-source compositionはまだ表現・Previewできない。Second Application専用の別Acceptance経路は作らず、同じShared Acceptance Boundaryの次incrementをpressureする。

正本:

- `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_COMPATIBILITY_PRESSURE_2026-09.md`
- `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md`
- `plan/SECOND_APPLICATION_S5_ACCEPTANCE_EXECUTION_PACKET_2026-09.md`
- `plan/SECOND_APPLICATION_S6_SOURCE_RELOAD_VALIDATION_PACKET_2026-09.md`
- `plan/SECOND_APPLICATION_S7_ENVIRONMENT_BINDING_PREFLIGHT_2026-09.md`
- `plan/SECOND_APPLICATION_S8_DRIVER_FIRST_WEB_TARGET_PREFLIGHT_2026-09.md`
- `plan/SECOND_APPLICATION_S8_SUCCESSOR_HANDOVER_PREFLIGHT_2026-09.md`

現在のpre-Acceptance停止条件は三つ独立している。

1. revised three-source Proposalのhealth-checked executable pressure
2. exact `c18ae...` Proposalをcurrent `baseModelRevision`へ結び付けたmulti-layer Shared Acceptance Preview PASS
3. exact reviewed Proposalに対するexplicit human Acceptance decision

2026-09-21更新: 2は成立した。`tools/reference-shared-acceptance`が型付きoperation `add-data` / `add-application` / `add-reference-pattern`を扱い、実際の`reference/workspace`（base revision `sha256:f70da9b64a5b…`）に対する`c18ae...`のread-only Previewが全stage PASSした（fingerprint・差分・impact・unresolvedは[実装記録](../SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_IMPLEMENTATION_PACKET_2026-09.md)）。1も成立した: `tools/reference-business-procedure-check`のS3 / S4 / S5 review / freshness pressureを改訂版3-source Proposalへ更新し、temporary workspaceへのfull-workspace Validation（positive 1件とnegative control 2件）を追加して、focused `TestSecondApplication...`・reference-evidence runner・repository health checkがPASSした（[実行記録](../SECOND_APPLICATION_S5_EXECUTABLE_PRESSURE_IMPLEMENTATION_PACKET_2026-09.md)「10. 実行記録」）。残る停止条件は3だけで、Previewもexecutable pressureも人のAcceptanceへ読み替えない。

Shared Acceptanceはまだ成立しておらず、authoritative destination source / Environment binding / Target implementationは作成していない。PreviewをAcceptanceへ読み替えず、一般的な作業継続指示もProposal採用Decisionへ読み替えない。

**Acceptanceの進め方（利用者の方針、2026-09-21）**: exact `c18ae...`のAcceptanceは、**別の生成AIによる独立レビューの後**に行う。レビューが済むまで`accept`は実行しない。作成者（`claude-sonnet-5`）のself-reviewは独立レビューの代わりにしない。レビューの指摘は、accept前に対応するか、理由つきで見送る。レビュー結果とAcceptance decision（exact `c18ae...`、判断、その時点のrevision、残るunresolved）は、このIssueへ追記する。この方針は「レビューの後にacceptする」という順序の決定であり、レビュー結果が出る前のacceptを許すものではない。

独立レビュー入力は `plan/SECOND_APPLICATION_S5_INDEPENDENT_REVIEW_HANDOFF_2026-09.md` に固定し、2026-09-21にGPT-5.6 Sol（OpenAI）がレビューを実施した。

結果:

- `plan/SECOND_APPLICATION_S5_INDEPENDENT_REVIEW_RESULT_2026-09.md`
- verdict = `review-clear-with-conditions`
- required pre-Acceptance fixes = none
- human Acceptance decision = not made by the review

R6が併せて指摘したtool側の欠陥（派生物の書き出しだけが原子的でなく、`-derived-out`に既存のgoldenを指されると失敗時にそれを壊し得る）は、2026-09-21に修正した。`accept`は派生物も`writeFileAtomic`で書き、書けない場合は既にあるfileを変えずにrefuseしてsourceを巻き戻す（`TestAcceptDoesNotDamageAnExistingDerivedArtifactWhenItCannotBeWritten`。修正前のcodeではこのtestが失敗することを確認済み）。上のintegration手順（`-derived-out`に`reference/world.json`を指さない）は、修正後も引き続き正本とする。

Repository integration条件は、actual acceptで`-derived-out reference/world.json`を直接使わず、source accept後に`go run ./tools/reference-golden -write`でcurrent goldenを生成し、golden check / repository health PASSまで同じRepository change内で完了すること。historical Gate B/C `B / R / T` はrevision-scoped Evidenceとして保持し、post-accept current Workspaceへcarry-forwardしない。

後続のShared Acceptance更新により、successful acceptは`.tei/decisions/<sha>.json`へaccepted Decisionを非authoritative Evidenceとして記録する。このrecordはmanaged sourceでもmodel revisionでもない。Second Applicationでは3 authoritative source + regenerated golden + exact Decision recordを同一integration unitへ含め、後段integration失敗でpre-acceptへ戻す場合はその試行のrecordも削除する。独立レビュー後のこの差分は再qualification済みで、review結論を変更しない。

S6はAcceptance sourceをもう一度別経路でauthorする段階ではなく、accepted 3-source compositionが通常Workspaceとして自立することを確認するEvidence段階とする。normal `tei validate`、Workspace model revision、`reference/world.json`、Source Index 6 subjects、shared authority / Environment不変を確認し、その後だけS7 Environment Binding Proposalへ進む。正本は `plan/SECOND_APPLICATION_S6_SOURCE_RELOAD_VALIDATION_PACKET_2026-09.md`。

S7 preflightでは、current development / productionが既に `master_data.dev` / `master_data.prod`（definition `master_data`, provider `oracle`）を持ち、`tei.reference.providers` bundleがOracle providerを供給していることを確認した。したがってS6 complete後の最小Environment候補は `repair_intake.customer_master` を既存instanceへbindする2行であり、新Database instance / Provider / Permission / Resource grantを先行作成しない。exact S7 Proposal fingerprint / base revisionはS6完了後に固定する。

独立レビューによりaccept後のRepository integration条件も評価済みである。actual acceptでは`-derived-out reference/world.json`を使わず、source accept後に`go run ./tools/reference-golden -write`でcurrent goldenを再生成し、golden checkとrepository health PASSまで同じRepository changeで完了する。Gate B / Gate Cの`B / R / T`はrevision-scoped historical Evidenceであり、post-Gate-C current Referenceの進化を旧baselineへcarry-forwardしない。実行手順は`plan/SECOND_APPLICATION_S5_ACCEPTANCE_EXECUTION_PACKET_2026-09.md`を正本とする。

## S5 Acceptance witness と実行記録 — 2026-09-21

`plan/SECOND_APPLICATION_S5_ACCEPTANCE_EXECUTION_PACKET_2026-09.md` §9の形式で記録する。

```text
proposal fingerprint:            c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
human Acceptance decision:       accepted
witness identity / locus:        Repository owner（hat47x）。Claude Code session上の明示的な回答「Acceptします。」
                                 直前に提示した内容: 独立レビュー完了（review-clear-with-conditions、必須修正なし）、
                                 残る停止条件はexplicit human Acceptance decisionだけであること
decision time:                   2026-09-21
repository revision before accept: f8785800c66dbca735caefa75d284f6ec5b33396
baseModelRevision:               sha256:f70da9b64a5b48c7682b24c64f455b0b0c2a7278be4faa297b6ff99c2d3707bf
Acceptance candidate fingerprint: sha256:8a936cf96ddeb4faa220285dcef5055d418030373f9425ed8d0d1c2b3e7c9b11
modelRevisionAfter:              sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
accepted source paths:           reference/workspace/semantic/repair-intake/intake.tei
                                 reference/workspace/applications/repair-intake/application.tei
                                 reference/workspace/applications/repair-intake/patterns/customer-reference.tei
decision record:                 reference/workspace/.tei/decisions/7a6931fba0dbe2f9dd0db840ba4f1ed4fcc2d4bc8e65d6595d556a828f4f5e66.json
golden update result:            reference/world.json のみ変更（68行追加・0行削除。すべてrepair_intake部分木）
reference-golden verification:   PASS
reference evidence runner:       PASS（run-knowledge-portability-reference-evidence.sh、46 test PASS / 0 FAIL）
repository health check:         PASS（gofmt / go vet / go test ./... / tei validate）
retained unresolved:             ENVIRONMENT_BINDING_NOT_AUTHORED
                                 DRIVER_SCENARIO_NOT_RUN
                                 TARGET_PROJECTION_NOT_EVALUATED
historical Gate B/C scope:       R = e5231839... / T = 0ba87a4e... はそのexact revisionへ紐づく歴史的Evidenceとして保持。
                                 受理後のcurrent Workspaceを旧B/R/Tのcurrent equivalentへcarry-forwardしない
resulting main commit:           0423b95
```

### 手順の遵守

§4の必須re-Previewを直前に実行し、記録済みの期待値と全一致した（acceptable、11/11 stage、candidate fingerprint、base revision、modelRevisionAfter、changed subjects 6、changed collections `[applications, semantics]`、unresolved 3件、impact edge 15本）。`-derived-out`は使わず、source acceptの後に`go run ./tools/reference-golden -write`でgoldenを再生成した。

受理後、ディスク上の3 sourceから再構成したbundleが**741 bytes / `c18ae0fd...`と一致**することを確認した。受理されたのはレビュー済みProposalそのものである。

### 受理に伴って更新したtest前提

受理により「`repair_intake`はまだ存在しない」という前提が正しく成り立たなくなり、3 packageの24 testが失敗した。いずれもAcceptanceの欠陥ではなく、Referenceの内容が進んだことによる前提の陳腐化である。各testの主張を変えずに前提だけを更新した。

| 対象 | 変更 |
| --- | --- |
| `internal/workspace` 3件 | 主題は`order_entry`のPatternであり、「ReferenceのApplicationは1件」は副次的な足場だった。`referenceApplicationByID`でID検索へ置き換えた |
| `tools/internal/changeanalysis` 2件 | 「第二Applicationを追加する」差分の起点を`referenceWorkspaceCopyBeforeSecondApplication`（受理前のReference）に変えた |
| `tools/reference-shared-acceptance` 19件 | 同様に`scratchWorkspaceBeforeSecondApplication`を導入した。これらのtestが証明する内容（composition、containment、rollback、identity衝突の拒否、決定記録）は受理前と同一のworkspaceに対して変わらず成立する |

guardを緩めた箇所は無い。`TestSecondApplicationCompositionRejectsAnExistingIdentity`のような衝突検出testは、受理前のworkspaceを起点にすることで意味を保っている。

### S6 — 完了（2026-09-21）

`plan/SECOND_APPLICATION_S6_SOURCE_RELOAD_VALIDATION_PACKET_2026-09.md` §14のexit criteriaを全て満たした（実行記録は同§15）。Acceptance toolを使わない通常経路で、`tei validate` PASS、model revisionがPreviewの`modelRevisionAfter`と一致、`tei world`がgoldenとbyte一致、Source Index 6 subjectがexpected artifactへ戻ること、共有authorityとEnvironmentが不変であることを確認した。

決定記録については、機械到達できること（`accepted` / `current` / `intact`）と、**記録を除いてもmodel revisionとCanonical Worldが変わらないこと**の両方を確認した。記録はEvidenceであってauthorityではない。

初回Second Applicationについては、専用のportability grammar（Knowledge Package / inheritance / override / sync）は不要だった。既存のSemantic / Application / reference Pattern / Database Contract参照だけで成立している。一般化はしない。

### S7 — executable pressure実施済み、Acceptance未了（2026-09-21）

`plan/SECOND_APPLICATION_S7_ENVIRONMENT_BINDING_PREFLIGHT_2026-09.md` §9のpressureを実装・実行し、§13へ記録した。最小のEnvironment binding 2行が共有Validationを通り、§5の5種類のdefectがそれぞれ期待するcodeでfail closedし、変更が2 Environment fileの1行ずつに収まることを確認した。Repositoryのauthoritative Environment sourceは変更していない。

```text
S7 Proposal fingerprint = sha256:9fb68c2878f468ed57e50b73b142344df9426cbc0eb20c5d60e9389ec367e2e4
S7 baseModelRevision    = sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
S7 Acceptance decision  = absent
```

**Acceptance経路（仮採用、2026-09-21）**: 受入境界へ`add-environment-binding`を追加し、S5と同じ経路で受理する（preflight §13 / §14）。Environmentだけを境界の例外にすると、S5の前例とも`RQ-027`の決定記録とも食い違うためである。仮の判断であり、見直し条件はpreflight §13に記録した。

実際の`reference/workspace`へのread-only Previewは11 stage全PASSで、変わるcollectionは`environments`だけ、意味の差分は0件、impact 8本、`RUNTIME_AUTHORITY_NOT_GRANTED`ほかをunresolvedとして残す（candidate fingerprint `sha256:fa359de4…`）。`accept`は未実行で、`reference/workspace`は未変更である。

2026-09-21、利用者の明示的なAcceptance decision（「S7 Accept します」）によりS7 Proposalを受理した。witnessと実行記録はpreflight §15を正本とする。

```text
accepted source        environments/development/environment.tei（+1行）
                       environments/production/environment.tei（+1行）
modelRevisionAfter     sha256:4c0ec01c83b1412454a4ab2ba4adf8cd8904ea7609f1326acb910ba5652f1f5c
decision record        .tei/decisions/d89b5434…json
health check           PASS
```

受理に伴い、「まだbindingが無い」を前提にしたtestの起点を更新した。その過程で、`...BeforeSecondApplication`系のfixtureが**Applicationを消しながらbindingを残していた**（存在しないApplicationへbindしたWorld）ことが分かり、あわせて修正した。

`RUNTIME_AUTHORITY_NOT_GRANTED` / `DRIVER_SCENARIO_NOT_RUN` / `TARGET_PROJECTION_NOT_EVALUATED`は未解消のままS8へ渡す。bindingができたことを実行可能性へ読み替えない。

### S8 — Driver / First Web Target preflight（2026-09-21）

正本: `plan/SECOND_APPLICATION_S8_DRIVER_FIRST_WEB_TARGET_PREFLIGHT_2026-09.md`

source inspectionで、developmentはaccepted `repair_intake.customer_master -> master_data.dev` bindingに加え、既存の `master_data.customer/find_by_id -> database.query.oracle` Contract Query executor bindingを持つため、instance-aware synthetic Driver scenarioを既存Driver APIだけで再現できることを確認した。runtime inputは`repair_intake.intake.customer_id`をSemantic columnとするmapped rowとして渡し、Relation keyからQuery input `id`へ到達させる。

productionには`master_data.prod / master_data.customer/find_by_id`のexecutor bindingが無いため、development bindingやprovider名から暗黙補完しない。

First Web TargetはProjection自体はApplication IDを受けるが、current read scenario runnerは`order_entry`固定で、`-application repair_intake`でも`/scenario/read`がorder_entryを実行し得る。このmanifest / runner identity mismatchをS8の最初のTarget pressureとする。Second Applicationの最小surfaceはCustomer ID input -> mapped Semantic row -> Driver reference lookupで、order_entryのbrowse / command / custom-submitを暗黙copyしない。

actual Oracle principal / credential / network authorityは未観測のままなので、deterministic synthetic Evidenceとactual external Oracle Evidenceを分離する。`RUNTIME_AUTHORITY_NOT_GRANTED`は保持する。

S8a Driver test-only candidateは `decdb5c16d403e72e8ce12dcd62698d29b3ae40f`（base `50ce3bd6...`）へ更新した。successor B2 preflightはdocs-onlyでS8 sourceと非競合だったため、強化版test blob `a1fc1da3...` を変更せず最新main直上へ積み直した。並行R7B更新はdocs-onlyでS8 sourceと非競合だったため、強化版 `015814af...` のtest blob `a1fc1da3...` を変更せず最新main直上へ積み直した。remote branch / PRは作らずmain未接続。accepted Semantic source key、development execution、not-found、zero-or-one violation、exact registry、provider mismatch、production executor binding不足の7 pressureを持つ。test blob `a1fc1da3...` はlocal gofmt後blobと一致する。ただしfocused test / reference evidence / repository healthはcheckout不在とGitHub DNS制約で未取得のためExecutable Evidenceとは扱わず、health PASS前にmainへ接続しない。

S8 Target provenanceは旧Gate Cの `B / R` をcarry-forwardしない。`repair_intake`は旧 `R=e523...` に存在しないため、S8a Driver scenarioをcurrent sourceで成立させた後、そのrevisionで必要なHandover Evidenceを再qualificationし、successor `B2 / R2`を発行してからFirst Web Targetへ渡す。Target buildは別identity `T2`として追跡する。

S8bへ進む前のsuccessor Handover契約を `plan/SECOND_APPLICATION_S8_SUCCESSOR_HANDOVER_PREFLIGHT_2026-09.md` に固定した。R2はS8a integration + health PASS後のclean revisionまで割り当てない。exact R2で既存Gate B Required Evidence #310/#255/#308をfinal convergence `-broad`でsame-run再取得し、retained child artifactsをreadiness artifact auditで独立再照合する。その上でSecond Application Driver Evidenceを追加参照してB2/R2を新規発行する。historical B/R/Tは不変で、#308 PostgreSQLをactual Oracle Evidenceへ読み替えない。

### S8 — Driver scenario（2026-09-22）

S7が残した`DRIVER_SCENARIO_NOT_RUN`について、受理済みの第二ApplicationがAcceptance toolを介さない通常のDriver経路で解決・実行できるかを確認した。実装は`internal/driver/second_application_scenario_test.go`である。

#### 分かったこと

**1. 第二Applicationのために新しいEnvironment sourceは要らなかった。**

Contract Query executor bindingは`(environment, instance, database, contract, query)`で識別され、**Applicationでは識別されない**。`repair_intake.customer_reference`は`order_entry`と同じ`master_data.customer / find_by_id`を`master_data.dev`上で使うため、`order_entry`のために書かれた既存bindingがそのまま効いた。S7で受理したのはdatasource aliasのbinding 2行だけで、executor側は一行も足していない。

developmentでの実行は通り、Relation keyが宣言どおりparameterへ渡り、結果が共有`customer.customer.customer_id`へ戻る。

**2. productionでは実行できない。これは第二Application固有ではない。**

`reference/workspace/environments/production`には**Contract Query executor bindingが一件も無い**（あるのはdevelopmentの2件だけ）。そのためproductionでは、S7 bindingによりaliasが`master_data.prod`まで解決した後、executor bindingの不在で止まる。

`order_entry.customer_reference`をproductionで実行しても同じ失敗になることを確認した。current Referenceの既存の状態であり、S5 / S7の受理が壊したものではない。これがReferenceとして意図的な範囲なのか、埋めるべき欠落なのかはここでは判定しない。

重要なのは、**developmentのexecutorへ暗黙に落ちない**ことである。失敗は`master_data.prod`まで解決した上でのexecutor binding不在として現れる。

**3. 製品側のtest coverageの穴を見つけて塞いだ。**

`ContractQueryExecutorRuntimeKey`は「Capability IDだけへ縮退すると接続先を取り違え得る」という理由でEnvironment / instance / Capabilityの三つで構成されている。変異検査で、`ExecutorForContractQuery`が組み立てるkeyから**Environment成分を落としても`internal/driver`と`internal/realization`の全testが通ってしまう**ことが分かった。instance成分とbinding解決側のEnvironment判定は既存testが守っていたが、runtime keyのEnvironment成分だけ固定されていなかった。

`internal/driver/contract_query_executor_registry_key_test.go`を追加して塞いだ。追加後は三つの変異（registry keyがEnvironmentを無視 / instanceを無視 / binding解決がEnvironmentを無視）すべてをtestが検出する。

この穴は、current Referenceにexecutor bindingがdevelopmentにしか無いために実Workspace経由では構成できず、registryを直接検査して初めて固定できた。

#### この段階で主張しないこと

使ったExecutorはtest doubleである。実際のDatabaseへ接続した証拠ではない。S7が残した`RUNTIME_AUTHORITY_NOT_GRANTED`（Permission grant / Resource binding / lease / 実行principal / credential / network authority）はそのまま未解決である。

`TARGET_PROJECTION_NOT_EVALUATED`も未着手で、First Web Target上の`repair_intake` surfaceは評価していない。

```text
DRIVER_SCENARIO_NOT_RUN          -> developmentについて解消（test doubleによるDriver scenario）
                                    productionはexecutor binding不在のため未実行のまま
RUNTIME_AUTHORITY_NOT_GRANTED    -> 未解決
TARGET_PROJECTION_NOT_EVALUATED  -> 未着手
```

## S4 full-validation correction

S6 preflightで、旧Application + Pattern Proposalがdestination Semantic Relation sourceを欠き、current full workspace Validationへそのまま接続できないことを確認した。

- old fingerprint: `b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2` — superseded before Acceptance
- revised fingerprint: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- revised design: `plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md`

revised Proposalは `repair_intake.intake.customer_id` と `repair_intake.intake.customer` を持つ最小destination Semanticを含む。authoritative sourceはまだ作成しない。`plan/SECOND_APPLICATION_S5_EXECUTABLE_PRESSURE_IMPLEMENTATION_PACKET_2026-09.md` で、temporary workspaceへのthree-source注入、`workspace.Load`、`validation.Validate`、S3/S4/S5 pressure更新、reference-evidence scriptのgofmt対象追随、repository health checkまでを次のhealth-capable execution契約として固定した。

2026-09-20の再検査では、document freshness baseを `c185937ab27a3a31332fefcd63ba5940b97b021e` まで更新し、関連shared/provenance/validation blobが不変であることを確認した。また `validation.Validate()` がdata-access Validation集約経路を通じてRelation key / Contract Query cross-layer検査を含むことも確認した。一方、`second_application_s5_freshness_capture_pressure_test.go` のbase revision constantは旧 `6ffaa40e...` のままであるため、これも次のexecutable-pressure code deltaへ追加した。

そのcode deltaは、当初unreferenced candidate `e007c67b...` として生成したが、actual `gofmt -d` でS3とfull-validation testにformat driftを検出したためsupersedeした。修正版は当初`dc5d5690dd393b871ba8ee0f2c242352741585a2` を親とするunreferenced branchless candidate `eda2e367bc1a0f12d83a41fc55e253b96bf76e9d` として作成した。その後のmain更新がR7B/Second Application計画文書だけで対象sourceと競合しないことを確認し、6 source blobを変更せず`b12962eebe75eea907241cf6cf2b831da7e99474` 上へ積み直したcurrent candidateは `9f0dd71dbd181e4798f7b8efefb09759bdcc256e` である。remote branch / PRは追加していない。candidateはS3/S4/S5 pressure、new full-workspace Validation pressure、reference-evidence runnerだけを変更し、authoritative `reference/workspace` sourceには触れていない。5 Go filesはactual gofmt差分ゼロかつlocal formatted blob SHAとcandidate blob SHAが一致した。current candidateへの積み直し後も6 source blobは全件同一である。さらにrunnerはactual `bash -n` PASSかつcandidate blob `42db7c4e...` とlocal blobが一致し、current Semantic/Validation APIとhelperのstatic照合でもcompile surfaceの不一致は見つからなかった。RepositoryのKnowledge portability workflowはhosted runner allocation unavailableのためmanual-onlyであり、current candidateのExecutable Evidenceには利用していない。Persistence Mapping未定義もcurrent validatorではそれ自体がerror条件ではない。full Validation pressureはpositive-onlyの偽陽性を避けるため、candidate `8c570c746f78a3346348d3c9bb1f6a3bd6e3debb` でnegative controlを追加し、その6 source blobを変更せずsource-relevant integration base `40209d25...` 直上へ積み直したcandidateは `ccd2544a8a14501774ae3970813c7535e350ec90` である（そのbaseに対しahead 1 / behind 0）。このcandidateを記録する後続docs-only commitだけではsource driftと扱わない。unknown Relationでは `UNKNOWN_APPLICATION_PATTERN_RELATION`、Relation `many` とQuery `zero-or-one` の不整合では `REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH` をexact codeで要求する。強化したtest blobは `0f1e69c1a62077ef31c6f599f567286209c4f0d5` で、local gofmt後blobと一致する。focused実行では `TestSecondApplicationS4ThreeSourceProposalPassesFullWorkspaceValidation`、`TestSecondApplicationS4FullValidationRejectsUnknownRelation`、`TestSecondApplicationS4FullValidationRejectsQueryCardinalityMismatch` の3件を必須とする。その後、Shared Acceptance Boundary実装を含むsource-relevant main `ca0ad042c0fc967421d9643e0c8a104349abf204` 上へ同じ6 source blobをrebaseし、current executable-pressure candidateを `b38869a48dcd8fb3e826a80de239063ee918c366` とした（baseに対しahead 1 / behind 0）。focused test / reference evidence / `tools/repository-health-check.sh` PASSは未取得なので、candidateをmainへ接続せずExecutable Evidenceとも扱わない。

## Semantic Extension Reference — 2026-10-04

RQ-031〜RQ-033で追加した「特定のモデル体系をCoreで特権化しない」「意味論を後から宣言・合成できる」「AIとToolが同じ意味拡張を探索できる」という要求を、最初のReferenceへ落とした。

`tools/reference-semantic-extension-catalog`では、production DSLやCore schemaを先に固定せず、Reference専用descriptorからExtensionを解決する。descriptorはExtensionのidentity、namespace、version、適用対象kind、compatible-major依存、concept、relation、Constraint、Projection / Capabilityへのconnection referenceを持つ。conceptのsubject kindはopenな文字列であり、新しいkindを追加するためにCoreのswitchやCanonical modelを変更しない。

最初のpressureでは、Coreに存在しない`service-level-policy`と`temporal-window`を追加し、既存Source Indexの`semantic-data:repair_intake.intake`と同じcatalogへ載せる。`service-level-policy:support.case.standard-response`は既存Dataを`governs`し、別Extensionが提供する`temporal-window:support.window.business-hours`を利用する。Service Levelの意味をData Modelへ押し込まず、別の意味視点として関係付ける。

resolverはcompatible majorを満たす依存closureを決定論的に作る。同じmajorに複数versionがあれば最も新しいversionを選ぶ。一方、CoreとExtension、または複数Extensionが同じsubject identityを宣言した場合は読み込み順で上書きせず、そのsubjectをresolved catalogから外して`subject-conflict`をUnresolvedへ残す。同一namespaceを複数Extensionが同時に所有する場合も、互換性contractが未定義の間は双方のsemantic contributionを公開せず`namespace-ownership-conflict`とする。存在しないrelation端点やConstraint対象も推測で補わない。

catalog出力には選択したExtensionのnamespace、version、適用対象kind、依存条件を残す。Core側は正規化Source Indexのfingerprint、Extension側は正規化descriptorのfingerprintを解決identityへ使い、raw descriptor path / file fingerprintはprovenanceとして分離する。root requirementsも正規化して記録し、これらから`resolutionFingerprint`を作る。これによりAIやToolは、何を要求してどのCore / Extension状態からcatalogを得たかを再現できる。

このReferenceはCanonical authority、Acceptance、Runtime authorityを追加しない。descriptor JSONもproduction DSLではない。複数の意味体系で共通する不足がReferenceから確認されるまではCore primitiveへ昇格させない。

現時点ではexact-headのfocused testと`tools/repository-health-check.sh`を実行していない。したがってRQ-031〜RQ-033のEvidence tierや本Issueの完了条件は更新しない。
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
- [ ] RQ-029のGapについて、上の変更伝播と固有条件の保持の確認に基づき、`Variation`の独立した表現が要るかを理由つきで記録した。要ると判断した場合は、実装の前に提案ADRを起こす。

測定対象は、参照できた同一identity、依存の検出範囲、不適合・未確定の検出、変更伝播、手で重複管理した箇所とします。コードのコピー率や生成量だけで再利用成功を判定しません。

## 上位要求と既存Issueとの関係

関連要求はRQ-001〜RQ-004、RQ-008、RQ-012、RQ-014〜RQ-017、RQ-019、RQ-021〜RQ-023、RQ-031〜RQ-033です。Workstream Bの意味知識をCのApplicationとHの開発支援へ接続します。

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
