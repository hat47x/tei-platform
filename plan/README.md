# TEIの開発計画

[開発全体の計画](MASTER_PLAN.md)と[マイルストーン](MILESTONES.md)は、期限を設けずに、製品の価値・着手の依存関係・完了条件を整理する正本です。全体像は[TEIの考え方](../docs/overview.md)、現時点での実証範囲は[実装状況](../docs/status.md)を参照してください。

`plan/`は、製品要求、着手順、受け入れシナリオ、設計判断、検証根拠をつなぐ計画の入口です。全体計画とマイルストーンは直下に置き、個別の計画文書は役割ごとのディレクトリにまとめます。

- 何を守るか: `product/`
- 何を満たすか: `requirements/`
- どのシナリオで確かめるか: `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- どの順で不確実性を減らすか: `plan/`

## 文書の配置

`plan/`の直下には、入口となる文書を置きます。詳細な計画は、次の分類から探してください。

| 調べたいこと | 参照先 |
| --- | --- |
| 全体の着手順と現在の主線 | [`MASTER_PLAN.md`](MASTER_PLAN.md) |
| 要求から受け入れシナリオ、検証根拠まで | [`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) |
| 個別作業の目的・完了条件 | GitHub Issues |
| 採用した設計判断 | [`adr/README.md`](adr/README.md) |
| 複数の課題にまたがる機能設計 | [`design/README.md`](design/README.md) |
| 段階判定の条件・手順・記録 | [`gate/README.md`](gate/README.md) |
| 要求、受入条件、検証根拠 | [`evidence/README.md`](evidence/README.md) |
| 技術調査や採用前の設計評価 | [`research/README.md`](research/README.md) |
| 複数作業にまたがる計画・移行 | [`workstreams/README.md`](workstreams/README.md) |
| 特定時点の監査結果 | 公開ツリーには個別の監査・検討記録を含めません |
| 現在の実装・実証状況 | [`../docs/status.md`](../docs/status.md) |

詳細な調査・検討記録は公開ツリーに含めません。採用した設計判断はADR、現行の要求は要求文書、作業条件はGitHubの課題でそれぞれ管理します。

## 最初に読む文書

1. [`MASTER_PLAN.md`](MASTER_PLAN.md) — 現在の作業分野と開発の主線
2. [`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md) — `order_entry`を使う第1版の実証順
3. [`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) — 製品要求、受け入れシナリオ、検証根拠の対応
4. [`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md) — 現在の要求に関する検証根拠と不足
5. [`MILESTONES.md`](MILESTONES.md) — 期限を設けない機能の到達目標
6. [`UPSTREAM_BREAKDOWN_2026-09.md`](workstreams/UPSTREAM_BREAKDOWN_2026-09.md) — 上流の設計上の検証や監査で得た結論を、計画と下流の担当へ対応付けた一覧

## 現在のv1主線

```text
Application Pattern Stage 1 — Done
        ↓
Gate A — Ready
        ↓
Application Driver — major semantics ready
        ↓
Gate B — Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
Handover Baseline B / revision R — 固定済み
        ↓
First Web Target build T1 — Evidence取得済み（T = 0ba87a4e0a96…）
        ↓
Gate C — PAS-04 — Ready
        ↓
   ┌──────┼──────────────┐
   ↓      ↓              ↓
Studio  knowledge / AI  Second Target
PAS-01  PAS-06 / 01     PAS-04
```

Gate Cの後は、設計・編集環境（Studio）、既存の知識を優先する探索、AIによる変更提案、第二の実行先の開発を並行して進められます。

## Gate A

判定条件、実行手順、判定記録は[`gate/README.md`](gate/README.md)にまとめています。

- [`GATE_A_READINESS_AUDIT.md`](gate/GATE_A_READINESS_AUDIT.md)
- 現在: **Ready**

アプリケーションの型、データソース、検証、認証・認可、機能の対応付け、業務結果・実行エラーなどをドライバーへ渡すための意味上の基準が成立しています。

## Gate B

### 正本文書

- [`GATE_B_EVIDENCE_ACQUISITION.md`](gate/GATE_B_EVIDENCE_ACQUISITION.md) — 必要な検証根拠と、それによって証明する内容
- [`GATE_B_EVIDENCE_RUNBOOK.md`](gate/GATE_B_EVIDENCE_RUNBOOK.md) — 正式な実行手順
- [`GATE_B_READINESS_AUDIT.md`](gate/GATE_B_READINESS_AUDIT.md) — Ready / Not Readyの判定記録
- ISSUE-000037-gate-b-three-lane-final-evidence.md — three-lane final convergence

### 現在

**Ready**です（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

必要な検証根拠は次の三つで、いずれも同じ実行・同じ改訂版Rについて成立しています。

| Evidence | Issue | 状態 |
| --- | --- | --- |
| deterministic Repository | `ISSUE-000032` / #310 | 成立 |
| generated artifact reproducibility | `ISSUE-000025` / #255 | 成立 |
| real PostgreSQL Provider + ephemeral fixture | `ISSUE-000031` / #308 | 成立 |

#387の最終的な集約と独立した成果物監査（`artifactAuditPassed = true`）の詳細は[`GATE_B_READINESS_AUDIT.md`](gate/GATE_B_READINESS_AUDIT.md)を参照してください。

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、Gate Bの第四Required laneではなく、未完了のままHandoverへ`unresolved`として渡しています。

## Gate B → Gate C Handover

- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](gate/GATE_B_TO_C_HANDOVER_CONTRACT.md) — Handoverの規則・原則
- [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](gate/GATE_B_TO_C_HANDOVER_BASELINE_B.md) — 実際に固定したBaseline `B / R`の記録
- ISSUE-000030-gate-b-to-c-handover-baseline.md

Gate B Ready candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`をHandover Baseline `B`として固定済みです。

引き継ぎ（Handover）は、新たな番号付きのGateでも、正本となる意味の第二の正本でもありません。

## Gate C / First Web Target

- [`GATE_C_READINESS_AUDIT.md`](gate/GATE_C_READINESS_AUDIT.md) — Ready判定の記録
- [`GATE_C_EVIDENCE_CONTRACT.md`](gate/GATE_C_EVIDENCE_CONTRACT.md)
- [`GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`](gate/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000028-first-web-target-conformance-evidence.md

Gate Cは2026-09-19に**Ready**と判定しました（`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`）。判定記録は[`GATE_C_READINESS_AUDIT.md`](gate/GATE_C_READINESS_AUDIT.md)です。画面の見た目の忠実さを評価したのは単一のAI評価者であり、人による確認ではありません。

最初のWeb向け実行先のビルド版`T1`は、基準となる改訂版`R`とは分けて追跡します。

```text
Handover B at R
      ↓ meaning baseline
Web build T1
      ↓
Gate C Evidence
```

Gate Cでは次を分離します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence
4. 必要に応じたPerformance Evidence

実行先側で基準となる意味を変更する必要がある場合、Web固有の項目や規則で補わず、次の引き継ぎを発行します。

## Business system design / PAS-07

この領域はGate B / Gate C Required Evidenceへ追加する新しいGateではなく、RQ-025 / RQ-026を現実的なBusiness Referenceで圧力試験するparallel workstreamです。

根幹から次の順で読みます。

1. [`../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md) — RQ-025 / RQ-026の要求解釈
2. [`../docs/business-system-design.md`](../docs/business-system-design.md) — Data / Function-Interaction / BusinessのArchitecture境界
4. ISSUE-000042-business-procedure-cross-view-reference.md — Lane C Reference / Pressure inventory
7. ISSUE-000050-second-realistic-handoff-corpus.md — acceptance contractを製造 / 在庫candidateへ適用し、independent Business seedからsecond realistic Handoff Evidenceを取得する計画（2026-09-21: 実施済み。実施記録あり）

現在、品質面で重視するのは、業務手順を単に「すべての項目を埋めた流れ図」として作ることではありません。

```text
known / accepted meaningをexactに保つ
+ derived / proposal / unresolvedを混同しない
+ Business instruction / system result / external actuality / Business acceptanceを短絡しない
+ responsibility Handoffをchronologyやactor差から発明しない
```

second corpusもHandoff fixtureから始めず、Handoffが無くても価値を持つindependent Business need / seedを先に成立させます。**Lane C Business Procedure Gate A**のactual green前はprovenance / candidate design question / RRB-P1適用準備までに留め、accepted Procedure / HandoffをBlocker解消のために捏造しません。

ここでいうLane C Business Procedure Gate Aは、本README前半のApplication Pattern **Gate A — Ready**とは別のPre-ADR execution gateです。名称を理由に両者のEvidence statusを混同しません。

## Definition-centered feedback / AI era

設計・編集環境、既存知識を優先する探索、AIによる変更提案、言語モデルに依存しない認知支援を別々の機能として発展させるのではなく、**簡潔な正本の定義を中心とした一つの改善の循環**へつなぎます。

設計本文:

- [`../docs/definition-centered-feedback-loop.md`](../docs/definition-centered-feedback-loop.md) — concise definition / three-memory separation / feedback routing / minimum semantic delta / Reference Modelの役割
- [`DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`](workstreams/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md) — Gate C前後の実証順と既存Issueの責務統合

中心flow:

```text
concise authoritative definition
  ↓
Canonical meaning
  ↓
derived projection / runtime / simulator / index
  ↓
human review / Observation / Reference pressure
  ↓
route feedback to owning concern
  ↓
exact / pattern / cognition / LLM candidates
  ↓
deterministic qualification + Preview
  ↓
Shared Acceptance Boundary
  ↓ accepted minimum semantic delta
authoritative source → reload / regenerate
```

この仕組みでは、言語モデルや認知用の索引へ新たな決定権限を与えません。参照モデルは回答集ではなく、具体的な設計例、実行できる内容、準拠条件の検証材料、設計の型を裏付ける根拠、開発基盤を改善するための情報源として扱います。

Design Synthesisの長期目標は、上流purposeから複数のstructured design candidateを作り、exact / constraint / pattern / cross-view / associative cognitionで候補空間を狭め、local LLMには少数案の比較・統合・minimum semantic delta提案を担わせることです。accepted meaningからimplementationとtestをできる限り再生成し、人間の反復authoringをmeaning selection / Acceptanceへ圧縮します。

Gate C前はdefinition inventory、feedback routing、memory separation、Reference pattern provenance等のread-only設計準備までです。Gate C後、`ISSUE-000046` / `000047` / `000027`を一つの小さいsemantic changeで合流させ、その後に`ISSUE-000051`のnon-LLM cognition比較へ進みます。

## Gate C後のplanning

### Studio / PAS-01

- ISSUE-000046-studio-shared-acceptance-authoring-loop.md

最小loop:

```text
Select → Propose → Preview → Accept / Reject
                    ↓ Accept only
          Shared Acceptance Boundary
                    ↓
          authoritative source → reload
```

設計・編集環境は、正本となる意味の第二の正本ではありません。定義を中心とした改善の循環では、モデル全体を編集するより、選んだ定義と関連する近くの要素を扱います。AIを使う場合も、ファイル全体の再生成より、確認できる最小限の意味の変更を優先します。

### knowledge-first / PAS-06

この領域は、個別algorithmやfeature設計からではなく、Product上の趣旨から順に読みます。

4. ISSUE-000047-knowledge-first-semantic-exploration.md — Human / AI共通のknowledge-first Product Reference
5. ISSUE-000051-derived-cognitive-retrieval-evaluation.md — candidate retrieval方式を比較するarchitecture probe

Human / AIが同じauthoritative Source / Canonical / Evidenceをread-only探索し、`applicable / incompatible / unknown`を分けることは最初の実証スライスです。FlyHash、HDC/VSA等はその候補生成部分に価値があるかを比較する方式であり、最終目標はretrieval単体ではなく、設計候補の生成・縮約・Acceptance・implementation/test derivationまでfeedback loopを閉じることです。

### AI Proposal / PAS-01 / PAS-06

- [`adr/0022-ai-proposal-canonical-acceptance-boundary.md`](adr/0022-ai-proposal-canonical-acceptance-boundary.md)
- ISSUE-000027-ai-proposal-acceptance-boundary.md

AI ProposalはCanonical authorityではなく、file-native / Studio / CLI等とShared Acceptance Boundaryへ収束します。AIの標準的な価値はwhole modelを再生成することではなく、current revision・explicit refs・Evidenceに基づく**小さい意味差分、比較、説明**を作ることです。

### Second Target / PAS-04

- ISSUE-000026-target-capability-execution-profile-boundary.md

最初の実行先だけで、実行先に依存しない分類体系が完成したとは判断しません。第二の実行先で、利用可能な機能、実行環境との対応付け、実行時の信頼条件、RQ-024の性能要件を検証します。

開発者向けに用意したStudioのWailsホストは、アプリケーションの実行先を検証した根拠には含めません。

## ADR / Plan Issue / 機能設計

- [`adr/`](adr/) — 長く残す設計判断
- [`issue/`](issue/) — 複数PR / Reference / Evidenceへまたがる作業定義
- [`design/`](design/README.md) — 機能単位の設計本文。Architectureの第二正本にはしない

計画段階にある将来の作業まで、必要性を確かめずGitHubの課題として二重に登録することは目的にしません。

Plan Issueの構成ルールはGitHub Issuesを参照します。

## 判定文書と調査記録の役割

「監査」というファイル名だけで保管先を決めません。現在の判定や受入条件を管理する文書と、特定の時点に実施した調査記録を区別します。

| 役割 | 正本または参照先 | 扱い |
| --- | --- | --- |
| Gate A・B・Cの判定 | [Gate A](gate/GATE_A_READINESS_AUDIT.md)／[Gate B](gate/GATE_B_READINESS_AUDIT.md)／[Gate C](gate/GATE_C_READINESS_AUDIT.md) | 判定した対象・版・Evidenceに結び付く正式記録。現在の`main`の検証済みを意味しない。 |
| 製品要求の不足と根拠 | [要求別の不足監査](evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md) | 更新型の照合文書。要求と実証の正本を置き換えない。 |
| 業務手順の契約ADR作成可否 | 関連するIssueと採用ADR | 現行の作業条件と採用済み判断をそれぞれの正本で確認します。 |
| 特定時点の調査結果 | 公開対象外 | 詳細な調査・検討記録は製品文書と分けて管理します。 |

独立した確認や根拠の引き継ぎに使う文書は、関連する課題や判定文書との関係を確かめてから配置を変更します。旧パスに残す移動案内は参照をつなぐためのものであり、本文の正本ではありません。

## 移動案内の整理

文書を移動しても、旧パスへの参照がすぐになくなるとは限りません。参照元にはMarkdownのリンクのほか、コード、検査手順、生成物、別のリポジトリもあります。そのため、旧パスの案内を残します。

案内を削除するときは、現在の参照元をすべて確認し、必要な参照を正本のパスへ変更してから行います。検索で一致が見つからなくても、検索範囲が不完全なら参照が存在しないとは証明できません。未確認の参照元がある間は、案内を削除しません。

## current snapshot方針

全体計画、第1版の計画、マイルストーン、要求の不足監査、Gate契約などの上位文書には、判断履歴を無制限に追記せず、**現在の正式な基準**を優先して記載します。

過去の判断経緯はGitの履歴、ADR、完了または置き換え済みの計画上の課題から追跡します。

同じ実行手順や意味上の境界について、複数の文書を競合する正本にしません。
