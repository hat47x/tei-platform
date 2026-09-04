# TEI Planning

このディレクトリには、TEI（鼎／テイ）の上位価値とProduct Requirementsを、実際の着手順、並走関係、設計判断、調査課題へつなぐ文書を置きます。

`plan/`はProduct Valueを定義する場所ではありません。

- 何を守るか: `product/`
- 何を満たすか: `requirements/`
- 何から進め、何をEvidenceとして確認するか: `plan/`

## 主な文書

- [`MASTER_PLAN.md`](MASTER_PLAN.md): 開発全体のworkstreamと依存関係
- [`MILESTONES.md`](MILESTONES.md): 期限を置かないCapability MilestoneとVersion Horizon
- [`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](PRODUCT_REQUIREMENT_GAP_AUDIT.md): Product Requirementsと現行実装のEvidence / Gap
- [`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md): `order_entry`を使ったv1 Application実証の主線とEvidence Gate
- [`TECH_RESEARCH_INTEGRATION_2026-09.md`](TECH_RESEARCH_INTEGRATION_2026-09.md): 技術リサーチをRequirement / Gate / ADRへ変換する統合方針
- [`GATE_A_READINESS_AUDIT.md`](GATE_A_READINESS_AUDIT.md): Driver開始前に必要なApplication上の意味の監査
- [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md): Application Driverから第一Web Targetへ進めるかをEvidenceから判定する監査
- [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md): Gate Bに必要なEvidenceの取得対象と位置づけ
- [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md): Repository / provider fixture / external assetの各Evidenceを、安全に取得・再現する具体的な実行手順と停止条件
- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md): Gate B通過時のDriver baselineをGate Cへ固定して渡すHandover契約
- [`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md): 第一Web TargetでGate Cを判定するSemantic / Functional / Visual Evidence契約
- [`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md): Application Patternの設計・実装基線
- [`LEGACY_AUTH_V1_DESIGN.md`](LEGACY_AUTH_V1_DESIGN.md): 既存認証仕様を保持してAuthentication / Authorization Capabilityへ接続する設計
- [`CAPABILITY_OUTCOME_V1_DESIGN.md`](CAPABILITY_OUTCOME_V1_DESIGN.md): CapabilityのBusiness Result / named OutcomeとExecution Errorを分離するv1設計
- [`adr/`](adr/): 長く残す設計判断
- [`issue/`](issue/): 調査、実装、Evidence取得、文書整備などの作業単位

## 現在のv1主線

既存の`order_entry`をReference Applicationとして育てています。

```text
Application Pattern Stage 1   完了
            ↓
        Gate A                 Ready
            ↓
    Application Driver         主要境界は成立
            ↓
        Gate B                 Not Ready / Evidence取得中
            ↓
   Gate B → C Handover         Gate B Ready時にbaseline固定
            ↓
      第一Web Target
            ↓
        Gate C
            ↓
       Editor / AI支援
            ↓
第二Targetで境界を圧力試験
```

Application Pattern側のStage 1実証は完了し、Gate Aで必要だった既存認証Compatibility、Capability Outcome / Execution Error、業務Validation、Capability BindingはReferenceへ合流しました。

`order_entry`では、次の5種類をApplication上の意図として実Referenceまで扱っています。

```text
browse
record
reference
command
header-detail
```

この5分類をTEI全体の永久的な標準語彙として確定したわけではありません。具体的な業務Applicationで、Target固有UIやPersistenceから切り離したApplication上の意図として成立することを確認した段階です。

詳細は[`APPLICATION_PATTERN_V1_DESIGN.md`](APPLICATION_PATTERN_V1_DESIGN.md)と[`ISSUE-000005`](issue/000005-application-pattern-reference.md)を参照します。

## Gate Bの現在地

Application Driverの主要な意味境界とScenario構造は成立し、PR #285で`main`へ収束しました。

その後、収束branch側で進んでいたexternal-observed Query Evidence intake、PostgreSQL fixtureのfail-closed安全化、Gate B Evidence RunbookをPR #300で`main`へ収束しました。これにより、Gate B監査が成立済みとして参照している主要なEvidence受入・安全境界と、`main`上の実装との不整合は解消しています。

現在の正式判定は **Gate B: Not Ready** です。ただし、残件の中心は追加設計ではなく、次のEvidence取得です。

1. #214 external-observed `master_data`の実Oracle物理事実 / read-only Evidence
2. header-detailの実PostgreSQL engine + ephemeral fixture Evidence
3. Reference World / Plugin Lock / Durable Runの正規再生成Evidence
4. 最終UI-less Scenarioおよびexternal Evidence intakeの通常Go runner実行Evidence

#214は、Evidenceを非Canonicalに受け入れて明示SQL realizationと照合する境界まではReadyであり、未完了なのはactual external assetの観測事実とread-only実証です。

PostgreSQL側も、既存`public.order_lines`がある接続先では停止し、tool自身が作成したfixtureだけをcleanupする安全境界まではReadyです。未完了なのは、専用の空PostgreSQL Databaseで実際にprovider Evidenceを取得することです。

Evidenceの対象と位置づけは[`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)、具体的なコマンド、実行順序、fail-closedの停止条件は[`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)、最新判定は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正とします。進行管理は[`ISSUE-000006`](issue/000006-application-driver-vertical-slice.md) / GitHub Issue #69と[`ISSUE-000029`](issue/000029-gate-b-readiness-audit.md) / GitHub Issue #273で行います。

GitHub Actionsによる継続的な再検証は重要ですが、Gate BのProduct Evidenceそのものとは分けます。Actions停止を意味設計Gapへ読み替えず、逆に未実行をgreenとも扱いません。

## CommandとObservation

Command acknowledgementとObserved stateは[`ADR-0023`](adr/0023-command-observation-explicit-read-boundary.md)に従い分離します。

```text
Command Business Result
        ↓ separate operation
explicit read boundary
        ↓
Observation read result / Execution Error
```

v1ではObservation専用の新しいApplication PatternやWorkflow基盤を先に作らず、既存Query / Contract Query / read adapter等の明示read境界をCommandとは別操作として利用します。

production Driverの最小Observation境界と最終UI-less Scenarioへの接続は成立し、GitHub Issue #266は完了しています。

## Gate BからGate CへのHandover

Gate Bを通過した時点のDriver baselineは、曖昧な「最新Driver」として第一Web Targetへ渡しません。

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)と[`ISSUE-000030`](issue/000030-gate-b-to-c-handover-baseline.md) / GitHub Issue #291に従い、Gate B ReadyとなったrevisionからHandover Baselineを固定します。

Handover Baselineは第二の正本ではありません。repository revision、Canonical identity、Driver Scenario、Datasource / authority境界、Business Result / Execution Error / Observation、Gate B Evidence、unresolvedへの参照を束ねます。

Gate Cの途中でApplication / Driverの意味が変わった場合は既存baselineへ黙って上書きせず、必要なGate B Evidenceを再確認したうえで新しいbaselineを発行します。

## 第一Web TargetとGate C

Gate Bを通過し、Handover Baselineを固定した後、[`ISSUE-000007`](issue/000007-first-web-target-projection.md) / GitHub Issue #70で第一Web Targetへ進みます。

Gate Cは「Web画面が表示できた」だけでは通過しません。[`GATE_C_EVIDENCE_CONTRACT.md`](GATE_C_EVIDENCE_CONTRACT.md)と[`ISSUE-000028`](issue/000028-first-web-target-conformance-evidence.md) / GitHub Issue #270に従い、少なくとも次を分けて取得します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

各Gate C Evidence runからHandover Baselineへ戻れることを要求し、Web側で不足した意味を独自fieldやruleとして補ってCanonicalへ逆輸入しません。

第一TargetだけでTarget-neutral Presentation modelを完成させず、Webで得た共通候補・Target Capability候補・Runtime Binding差・Execution / Trust差・Web固有差・unresolvedを第二Targetへ渡します。

## 技術リサーチから追加した横断境界

2026年9月の技術リサーチでは、AI支援、UI生成評価、Wails v3等を採用技術一覧へ直接追加するのではなく、次の境界へ変換して計画へ取り込みます。

- AI出力はProposal / Structured Intent candidateとし、Canonical authorityから分離する
- Editor / CLI / AI等を共通Acceptance Boundaryへ収束させる
- AIのautonomyとwrite / invoke / Resource authorityを分離する
- 過去の成功trajectory等はDerived Experience Knowledgeとして扱い、第二の正本にしない
- Target CapabilityとExecution / Trust Profileを分離して第二Targetで検証する
- Target ConformanceをSemantic / Functional / Visual Evidenceへ分ける
- HostのMCP / control面とApplication自身のMCP Targetを区別する

設計判断は[`ADR-0022`](adr/0022-ai-proposal-canonical-acceptance-boundary.md)、AIの実証は[`ISSUE-000027`](issue/000027-ai-proposal-acceptance-boundary.md) / GitHub Issue #267、第二Targetの境界検証は[`ISSUE-000026`](issue/000026-target-capability-execution-profile-boundary.md) / GitHub Issue #259で扱います。

詳細なAI interface、Web framework、Target Capability taxonomy、visual evaluatorはGate C前に固定しません。

## ADR

ADRには、実装方法が変わっても残る設計判断を記録します。

ファイル名は4桁の連番です。

```text
plan/adr/0016-example.md
```

`docs/adr/0001`〜`0015`までの既存ADRと番号を衝突させません。既存ADRは番号を維持したまま段階的に`plan/adr/`へ収束します。

現在の上位・横断的な判断は次です。

- ADR-0016: 文書体系と権威関係
- ADR-0017: Application Patternの初期所有境界
- ADR-0018: Application Patternの正本とfile pathからのidentity分離
- ADR-0019: Authentication / AuthorizationをSemantic Capabilityとして扱う境界
- ADR-0020: Business Result / named OutcomeとExecution Errorの境界
- ADR-0021: keyless header-detail Relation traversalのrealization境界
- ADR-0022: AI ProposalとCanonical Acceptance Boundary
- ADR-0023: Command acknowledgementとObservationの明示read境界

## Issue文書

Repository内に長く残す作業定義は6桁の連番で`plan/issue/`へ置きます。

```text
plan/issue/000001-example.md
```

GitHub Issue番号とは別のidentityです。

- Repository Issue: 長く残す背景、制約、完了条件、Evidence契約
- GitHub Issue: 実際の進行、会話、PRとの接続

小さな作業まで機械的にRepository Issueへ増やす必要はありません。

## 計画の更新方法

計画は日付を消化するためではなく、Product Valueに対する不確実性を減らすために更新します。

各作業では少なくとも次を確認します。

1. どのProduct Requirementに関係するか
2. 何がまだ分かっていないか
3. 実験・設計・実装・Evidence取得のどれで確かめるか
4. 他のworkstreamと並走できるか
5. 先に固定すると後戻りしにくい判断があるか
6. 完了時に何をEvidenceとして残すか

実装しやすい領域だけが先行し続けないよう、MilestoneではSemantic、Application、Database、Editor、AI、Target、Runtimeを横断して確認します。

## 自然な日本語の確認

計画文書、ADR、Issueは、内容を整理した後に全文を読み直し、意味を変えず自然な日本語へ整えます。

機械的な箇条書きだけで背景や判断理由が失われている場合は文章として補い、長い文章の中で条件や完了基準が埋もれる場合は箇条書きへ分けます。
