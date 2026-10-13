# TEIの開発計画

[開発全体の計画](MASTER_PLAN.md)と[マイルストーン](MILESTONES.md)は、期限を設けず、製品の価値、作業の依存関係、完了条件を整理する正本です。全体像は[TEIの考え方](../docs/overview.md)、現時点での実証範囲は[実装状況](../docs/status.md)を参照してください。

`plan/`は、製品要求、着手順、受入シナリオ、設計判断、検証根拠をつなぐ計画の入口です。全体計画とマイルストーンは直下に置き、個別の計画文書は役割ごとのディレクトリにまとめます。

- 守る価値：`product/`
- 満たす要求：`requirements/`
- 検証に用いるシナリオ：`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- 不確実性を減らす順序：`plan/`

## 文書の配置

`plan/`の直下には、入口となる文書を置きます。詳細な計画は、次の分類から探してください。

| 調べたいこと | 参照先 |
| --- | --- |
| 全体の着手順と現在の主線 | [`MASTER_PLAN.md`](MASTER_PLAN.md) |
| 要求から受け入れシナリオ、検証根拠まで | [`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md) |
| 個別作業の目的・完了条件 | GitHubの課題 |
| 採用した設計判断 | [`adr/README.md`](adr/README.md) |
| 複数の課題にまたがる機能設計 | [`design/README.md`](design/README.md) |
| 段階判定の条件・手順・記録 | [`gate/README.md`](gate/README.md) |
| 合成KernelのReference・実Process・製品移行・Target別の判定 | [`gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md`](gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md) |
| 要求、受入条件、検証根拠 | [`evidence/README.md`](evidence/README.md) |
| 技術調査や採用前の設計評価 | [`research/README.md`](research/README.md) |
| 複数作業にまたがる計画・移行 | [`workstreams/README.md`](workstreams/README.md) |
| 特定時点の監査結果 | 公開ツリーには個別の監査・検討記録を含めません |
| 現在の実装・実証状況 | [`../docs/status.md`](../docs/status.md) |

詳細な調査・検討記録は公開リポジトリには含めません。採用した設計判断は設計判断記録、現行の要求は要求文書、作業条件はGitHubの課題で管理します。

## 最初に読む文書

1. [`MASTER_PLAN.md`](MASTER_PLAN.md)：現在の作業分野と開発の主線
2. [`V1_REFERENCE_APPLICATION_PLAN.md`](V1_REFERENCE_APPLICATION_PLAN.md)：`order_entry`を使う第1版の実証順
3. [`UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`](UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md)：製品要求、受入シナリオ、検証根拠の対応
4. [`PRODUCT_REQUIREMENT_GAP_AUDIT.md`](evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md)：現在の要求に関する検証根拠と不足
5. [`MILESTONES.md`](MILESTONES.md)：期限を設けない機能の到達目標
6. [`UPSTREAM_BREAKDOWN_2026-09.md`](workstreams/UPSTREAM_BREAKDOWN_2026-09.md)：上流の設計検証や監査で得た結論と、計画や後続の担当との対応を示す一覧

## 第1版に向けた現在の開発の流れ

```text
アプリケーション構成パターンの第1段階 — Done
        ↓
判定段階A（Gate A）— Ready
        ↓
設計の実行・検証 — 主な意味を扱う機構は準備済み
        ↓
判定段階B（Gate B）— Ready（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
引継ぎの基準B・対象版R — 固定済み
        ↓
最初のWeb向けビルドT1 — 検証根拠取得済み（T = 0ba87a4e0a96…）
        ↓
判定段階C（Gate C）— PAS-04 — Ready
        ↓
       ┌──────┼────────────┐
       ↓      ↓            ↓
    編集環境  知識・AI    二つ目の実行先
    PAS-01  PAS-06／01       PAS-04
```

判定段階Cの後は、設計・編集環境、既存知識を優先する探索、AIによる変更提案、二つ目の実行先の開発を並行して進められます。

## Gate A

判定条件、実行手順、判定記録は[`gate/README.md`](gate/README.md)にまとめています。

- [`GATE_A_READINESS_AUDIT.md`](gate/GATE_A_READINESS_AUDIT.md)
- 現在：**Ready**

アプリケーションの型、データソース、検証、認証・認可、機能の対応付け、業務結果・実行エラーなどをドライバーへ渡すための意味上の基準が成立しています。

## Gate B

### 正本文書

- [`GATE_B_EVIDENCE_ACQUISITION.md`](gate/GATE_B_EVIDENCE_ACQUISITION.md)：必要な検証根拠と、それによって証明する内容
- [`GATE_B_EVIDENCE_RUNBOOK.md`](gate/GATE_B_EVIDENCE_RUNBOOK.md)：正式な実行手順
- [`GATE_B_READINESS_AUDIT.md`](gate/GATE_B_READINESS_AUDIT.md)：`Ready`／`Not Ready`の判定記録
- ISSUE-000037-gate-b-three-lane-final-evidence.md：三系統の検証根拠を最終的にそろえる作業

### 現在

**Ready**です（判定に用いた候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

必要な検証根拠は次の三つです。いずれも同じ実行と同じ対象版`R`について成立しています。

| 検証根拠 | 課題 | 状態 |
| --- | --- | --- |
| リポジトリの決定的な再現性 | `ISSUE-000032` / #310 | 成立 |
| 生成物の再現性 | `ISSUE-000025` / #255 | 成立 |
| 実際のPostgreSQLと一時的な試験データ | `ISSUE-000031` / #308 | 成立 |

#387で検証根拠を最終的にそろえた経緯と、独立した成果物の監査（`artifactAuditPassed = true`）は、[`GATE_B_READINESS_AUDIT.md`](gate/GATE_B_READINESS_AUDIT.md)に記録しています。

実際のOracle環境に関する#214、#371、#376は、外部資産についての重要な検証作業として継続します。ただし、判定段階Bで必須となる四つ目の検証条件には加えず、未完了のまま`unresolved`として引き継いでいます。

## 判定段階BからCへの引継ぎ

- [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](gate/GATE_B_TO_C_HANDOVER_CONTRACT.md)：引継ぎの規則と原則
- [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](gate/GATE_B_TO_C_HANDOVER_BASELINE_B.md)：実際に固定した基準`B / R`の記録
- ISSUE-000030-gate-b-to-c-handover-baseline.md

判定段階Bで`Ready`となった候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`は、引継ぎの基準`B`として固定済みです。

引継ぎは、新しい番号付きの判定段階ではなく、意味の第二の正本でもありません。

## 判定段階Cと最初のWeb向け実行先

- [`GATE_C_READINESS_AUDIT.md`](gate/GATE_C_READINESS_AUDIT.md)：`Ready`判定の記録
- [`GATE_C_EVIDENCE_CONTRACT.md`](gate/GATE_C_EVIDENCE_CONTRACT.md)
- [`GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`](gate/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md)
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000028-first-web-target-conformance-evidence.md

判定段階Cは2026-09-19に**Ready**と判定しました（`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`）。判定記録は[`GATE_C_READINESS_AUDIT.md`](gate/GATE_C_READINESS_AUDIT.md)です。画面の見た目の忠実さを評価したのは単一のAI評価者であり、人による確認ではありません。

最初のWeb向け実行先のビルド版`T1`と、意味の基準となる改訂版`R`は、別々に追跡します。

```text
改訂版Rに対する引継ぎの基準B
      ↓ 意味の基準
Web向けビルドT1
      ↓
判定段階Cの検証根拠
```

判定段階Cでは、次の検証根拠を分けて扱います。

1. 意味上の準拠性
2. 操作と機能の検証根拠
3. 表示の忠実度に関する検証根拠
4. 必要に応じた性能の検証根拠

実行先側で基準となる意味を変更する必要がある場合、Web固有の項目や規則で補わず、次の引き継ぎを発行します。

## 業務システム設計とPAS-07

この領域は、判定段階B・Cの必須検証根拠を増やす新しい判定段階ではありません。RQ-025とRQ-026を、現実的な業務の参照事例で厳しく検証する、並行する作業領域です。

関連する文書と課題は、次の順で読むと理解しやすくなります。

1. [`../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)：RQ-025とRQ-026の要求解釈
2. [`../docs/business-system-design.md`](../docs/business-system-design.md)：データ、機能・操作、業務を分ける設計上の境界
3. ISSUE-000042-business-procedure-cross-view-reference.md：作業領域Cの参照事例と検証対象の一覧
4. ISSUE-000050-second-realistic-handoff-corpus.md：受理に関する契約を製造・在庫の候補へ適用し、独立した業務上の初期資料から二つ目の現実的な引継ぎの検証根拠を取得する計画（2026-09-21に実施。記録あり）

現在、品質面で重視するのは、業務手順を単に「すべての項目を埋めた流れ図」として作ることではありません。

```text
既知の意味・受理済みの意味を正確に保つ
+ 派生情報・提案・未解決事項を混同しない
+ 業務上の指示・システムの結果・外部の実際の状態・業務上の受理を混同しない
+ 時系列や担当者の違いだけから、責任の引継ぎを作り出さない
```

二つ目の事例も、引継ぎ用の試験データから始めるのではありません。引継ぎがなくても成立する、独立した業務上の必要性や初期資料を先に確かめます。**Lane C Business Procedure Gate A**で実際に合格するまでは、出所の確認、設計上の検討候補、RRB-P1の適用準備までに留めます。受理済みの業務手順や引継ぎを、作業上の障害を解消するために作り出しません。

ここでいう**Lane C Business Procedure Gate A**は、この文書の前半で説明したアプリケーション構成パターンの**Gate A — Ready**とは別の、設計判断の採用前に行う実行上の判定です。名前が似ていても、両者の検証状態は区別します。

## 定義を中心にした改善の循環とAI

設計・編集環境、既存知識を優先する探索、AIによる変更提案、言語モデルに依存しない認知支援を別々の機能として発展させるのではなく、**簡潔な正本の定義を中心とした一つの改善の循環**へつなぎます。

設計方針の本文：

- [`../docs/definition-centered-feedback-loop.md`](../docs/definition-centered-feedback-loop.md)：簡潔な定義、三種類の記憶の区別、問題の振分け、最小限の意味変更、参照モデルの役割
- [`DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`](workstreams/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md)：判定段階Cの前後の実証順と既存課題の役割の整理

中心となる流れ：

```text
簡潔な正本の定義
  ↓
正本となる意味
  ↓
派生表示・実行基盤・模擬実行・索引
  ↓
人による確認・観測・参照事例による検証
  ↓
問題を管理する部分へ振り分ける
  ↓
正確な照合・構成パターン・認知処理・言語モデルによる候補
  ↓
決定的な方法による適格性確認・事前の影響表示
  ↓
共通の受理境界
  ↓ 受理された最小限の意味変更
正本となる情報源 → 読み直し・再生成
```

この仕組みでは、言語モデルや認知用の索引に新たな判断権限を与えません。参照モデルは回答集ではなく、具体的な設計例、実行できる内容、準拠条件の検証材料、設計の型を裏付ける根拠、開発基盤を改善するための情報源として扱います。

設計案を統合する仕組みの長期目標は、上流の目的から複数の構造化された設計候補を作り、正確な照合、制約、構成パターン、複数の視点、連想による認知処理で候補を絞ることです。そのうえで、ローカルの言語モデルに少数の候補の比較・統合と、最小限の意味変更の提案を担わせます。受理された意味から実装と試験をできる限り再生成し、人が繰り返し記述する作業を減らして、意味の選択と受理に集中できるようにします。

判定段階Cの前に行うのは、定義の一覧化、問題の振分け、記憶の区別、参照事例の構成パターンの出所などを、読み取り専用で整理する設計準備までです。判定段階Cの後に、`ISSUE-000046`、`000047`、`000027`の作業を一つの小さな意味変更で合流させます。その後、`ISSUE-000051`で言語モデルを使わない認知方式を比較します。

## 判定段階Cの後の計画

### 編集環境とPAS-01

- ISSUE-000046-studio-shared-acceptance-authoring-loop.md

最小限の操作の流れ：

```text
選択 → 提案 → 影響の確認 → 受理・却下
                    ↓ 受理された場合だけ
                 共通の受理境界
                    ↓
              正本の情報源 → 読み直し
```

設計・編集環境は、正本となる意味の第二の正本ではありません。定義を中心とした改善の循環では、モデル全体を編集するより、選んだ定義と関連する近くの要素を扱います。AIを使う場合も、ファイル全体の再生成より、確認できる最小限の意味の変更を優先します。

### 既存知識を優先する探索とPAS-06

この領域では、個別のアルゴリズムや機能設計に入る前に、製品として何を実現するかを確認します。

1. ISSUE-000047-knowledge-first-semantic-exploration.md：人とAIが共通に使う、既存知識を優先する参照事例
2. ISSUE-000051-derived-cognitive-retrieval-evaluation.md：候補の検索方式を比較する構成上の検証

人とAIが同じ正本の情報源、意味、検証根拠を読み取り専用で探索し、`applicable / incompatible / unknown`を区別することが、最初の小さな実証です。FlyHash、HDC、VSAなどは、候補生成に役立つかを比較する方式です。最終目標は検索そのものではなく、候補の生成・絞り込み・受理から実装と試験の再生成まで、改善の循環をつなぐことです。

### AIの変更提案とPAS-01・PAS-06

- [`adr/0022-ai-proposal-canonical-acceptance-boundary.md`](adr/0022-ai-proposal-canonical-acceptance-boundary.md)
- ISSUE-000027-ai-proposal-acceptance-boundary.md

AIによる提案には、意味の正本を変更する判断権限はありません。ファイルの直接編集、編集環境、コマンドラインからの変更と同じく、共通の受理境界で扱います。AIに期待する標準的な役割は、モデル全体を再生成することではありません。現在の版、明示した参照、検証根拠に基づく**小さな意味の差分、比較、説明**を作ることです。

### 二つ目の実行先とPAS-04

- ISSUE-000026-target-capability-execution-profile-boundary.md

最初の実行先だけで、実行先に依存しない分類体系が完成したとは判断しません。第二の実行先で、利用可能な機能、実行環境との対応付け、実行時の信頼条件、RQ-024の性能要件を検証します。

開発者向けに用意したStudioのWailsホストは、アプリケーションの実行先を検証した根拠には含めません。

## 設計判断記録・計画上の課題・機能設計

- [`adr/`](adr/)：長く残す設計判断
- [`issue/`](issue/)：複数のプルリクエスト、参照事例、検証根拠にまたがる作業の定義
- [`design/`](design/README.md)：機能単位の設計本文。全体構成の第二の正本にはしない

計画段階にある将来の作業を、必要性が確認できないままGitHubの課題へ重複登録することは目的にしません。

計画上の課題の記載方法は、GitHubの課題を参照してください。

## 判定文書と調査記録の役割

「監査」というファイル名だけで保管先を決めません。現在の判定や受入条件を管理する文書と、特定の時点に実施した調査記録を区別します。

| 役割 | 正本または参照先 | 扱い |
| --- | --- | --- |
| Gate A・B・Cの判定 | [Gate A](gate/GATE_A_READINESS_AUDIT.md)／[Gate B](gate/GATE_B_READINESS_AUDIT.md)／[Gate C](gate/GATE_C_READINESS_AUDIT.md) | 判定した対象・版・検証根拠に結び付く正式記録。現在の`main`が検証済みであることは意味しない。 |
| 製品要求の不足と根拠 | [要求別の不足監査](evidence/PRODUCT_REQUIREMENT_GAP_AUDIT.md) | 更新型の照合文書。要求と実証の正本を置き換えない。 |
| 業務手順の契約に関する設計判断記録を作成できるか | 関連する課題と採用済みの設計判断記録 | 現行の作業条件と採用済みの判断を、それぞれの正本で確認する。 |
| 特定時点の調査結果 | 公開対象外 | 詳細な調査・検討記録は製品文書と分けて管理します。 |

独立した確認や根拠の引き継ぎに使う文書は、関連する課題や判定文書との関係を確かめてから配置を変更します。旧パスに残す移動案内は参照をつなぐためのものであり、本文の正本ではありません。

## 移動案内の整理

文書を移動しても、旧パスへの参照がすぐになくなるとは限りません。参照元にはMarkdownのリンクのほか、コード、検査手順、生成物、別のリポジトリもあります。そのため、旧パスの案内を残します。

案内を削除するときは、現在の参照元をすべて確認し、必要な参照を正本のパスへ変更してから行います。検索で一致が見つからなくても、検索範囲が不完全なら参照が存在しないとは証明できません。未確認の参照元がある間は、案内を削除しません。

## 現在の正式な基準を記す方針

全体計画、第1版の計画、マイルストーン、要求の不足に関する監査、判定条件の契約などの上位文書には、判断履歴を無制限に追記しません。**現在の正式な基準**を優先して記載します。

過去の判断経緯はGitの履歴、ADR、完了または置き換え済みの計画上の課題から追跡します。

同じ実行手順や意味上の境界について、複数の文書を競合する正本にしません。
