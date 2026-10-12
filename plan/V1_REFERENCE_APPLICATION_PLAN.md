# TEI第1版に向けた参照アプリケーションの実証計画

## 目的

この文書は、TEI（鼎／テイ）が第1版へ進むために必要なアプリケーション側の実証を、参照アプリケーション`order_entry`を中心に整理したものです。

個々の課題の完了件数よりも、同じ業務上の意味を、意味の定義、アプリケーション、実行・検証環境、引継ぎ、実行先、検証根拠へと追跡できることを重視します。

期限は設けません。次の段階へ進めるかどうかは、参照事例、準拠性確認、検証根拠に基づいて判断します。

過去の対象範囲や旧来の検証作業の経緯は本文に重ねて記さず、Gitの履歴、設計判断記録、課題から追跡します。

## 第1版の参照アプリケーション

第1版では、理想化した新しい例を作るのではなく、既存の`reference/workspace/applications/order-entry`を拡充します。

主な資産は次のとおりです。

- 意味を管理する名前空間：`sales`、`customer`、`security`
- `sales.Order` / `sales.OrderLine`
- データ間の関係：`sales.order.customer`
- 一対多の関係：`sales.order.lines`
- 実行機能：`sales.order.submit`
- 認証・認可の機能：`security.authenticate`、`security.authorize`
- 読み書き可能なデータソース：`sales_core`
- 読み取り専用で外部の状態を観測するデータソース：`master_data`
- データベース契約、元形式の問い合わせ、問い合わせの実現方法
- アプリケーションパターン
- 能力の対応付け
- 正本上の検証規則
- 既存の認証・認可方式との互換性を確認する参照事例
- 業務上の結果と実行エラーの境界
- 見出し・明細形式の契約に従う問い合わせの実現
- データベースが管理する関係の保存
- 入力元と意味の正本との追跡関係

## 第1版の業務シナリオ

```text
Authentication
    ↓
browse Orders
    ↓
record open
    ↓
reference Customer
    ↓
header-detail OrderLine[]
    ↓
Canonical Validation
    ↓
Authorization
    ↓
Command Business Result
    ↓ separate explicit read
Observation
```

業務上の意味を、実行・検証環境、ウェブ、編集環境、AIでそれぞれ作り直すことはしません。同じ正本の意味を共有し、各層に必要な表示・対応付け・検証根拠だけを追加します。

## PASとの対応

| PAS | v1での主な実証 |
| --- | --- |
| PAS-01 | Gate C後のStudio minimal Authoring Loop / AI Proposal Acceptance |
| PAS-02 | Application Driver / Gate B / Handover |
| PAS-03 | external asset / Provider / Handover unresolved |
| PAS-04 | 第一Web Target / Gate C / 第二Target |
| PAS-05 | Runtime / deployment recovery系で継続。Gate B blockerへ追加しない |
| PAS-06 | Gate C後のknowledge-first Semantic exploration |

## 全体の進行

```text
Stage 0  Product Baseline / Gap Audit                 Done
               ↓
Stage 1  Application Pattern                          Done
               ↓
Gate A   Application meaning → Driver                 Ready
               ↓
Stage 2  Application Driver                           major semantics ready
               ↓
Gate B   3 Required Evidence same-R convergence       Ready
               ↓
Handover Baseline B / revision R                      固定済み
               ↓
Stage 3  First Web Target / build revision T1         Evidence取得済み
               ↓
Gate C   Semantic / Functional / Visual Evidence      Ready
               ↓
        ┌───────────────┴────────────────┐
        ↓                                ↓
Stage 4A Authoring / Knowledge        Stage 4B Second Target
  - Studio PAS-01                       - PAS-04
  - knowledge PAS-06                    - RQ-024
  - AI Proposal PAS-01/06
```

段階4Aと4Bは並行して進められます。編集環境やAI支援機能の完成を、二つ目の実行先の検証に必要な前提とはしません。

引継ぎは、独立した新しい判定段階ではありません。判定段階Bの出力を判定段階Cの入力として固定するための契約です。

## Stage 0 — Product Baseline / Gap Audit

**完了**しています。

製品の価値、将来像、設計原則、要求、受入シナリオ、全体計画、不足事項の調査結果を、リポジトリの基準資料として扱います。

## Stage 1 — Application Pattern

関連: `ISSUE-000005` / GitHub Issue #67

**完了**しています。

最初の参照事例では、次の5種類をアプリケーション側の意図として実証しています。

```text
browse
record
reference
header-detail
command
```

この5分類を、将来にわたって変更のないTEI全体の標準一覧とはみなしません。

## Gate A — Application上の意味をDriverへ渡せる

関連：`ISSUE-000013`／GitHub Issue #124  
監査：`GATE_A_READINESS_AUDIT.md`

**Ready**です。

主に次の領域で成立しています。

- アプリケーションパターン
- データ接続先の読取り・書込みの区別
- 能力の対応付け
- 正本上の検証規則
- 業務上の結果と実行時エラーの区別
- 認証と認可
- 既存認証との互換性
- 入力元と対応付けの追跡可能性

判定段階Aの`Ready`は、実行・検証環境が完成したことではなく、そこへ引き渡す意味の契約が成立したことを示します。

## Stage 2 — Application Driver

関連：`ISSUE-000006`／GitHub Issue #69  
Gate B監査：`GATE_B_READINESS_AUDIT.md`／`ISSUE-000029`／#273  
検証根拠の要件：`GATE_B_EVIDENCE_ACQUISITION.md`  
実行手順：`GATE_B_EVIDENCE_RUNBOOK.md`  
最終確認：`ISSUE-000037`／#387

### 現在の状態

主要な意味の境界とシナリオの構造は成立しています。

判定段階Bは**Ready**です（候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`、判定日：2026年9月19日）。

### Gate B Required Evidence

必須の検証根拠は次の三つです。いずれも、変更のない同じ候補版`R`について、同一実行の中で成立しています。

1. #310 リポジトリの決定論的な検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験用環境の検証根拠

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R  ← e5231839a5dcc62cce1181a2731fcb7c70d16736
#308 ─┘
       ↓
#387 final convergence  ← 成立
```

実際のOracleに関する#214・#371・#376は、第四の必須検証経路ではありません。

実際の物理的な事実・来歴・実効的な読取り専用の制約・ドライバーとの連携・実際の読取りは、引き続きPAS-03の重要な外部資産の検証根拠です。未完了であれば、引継ぎ時に`unresolved`として明示します。

### Driverで成立している主な境界

- アプリケーションパターンの解決
- データ接続先の解決
- 問い合わせと契約に基づく問い合わせ
- 入力と結果の明示的な対応付け
- 関係をたどる処理
- 見出し・明細の処理
- 検証規則の評価
- 認証と認可
- 能力の対応付け
- 命令の業務上の結果
- 実行時エラー
- 独立した読取りによる観測

### Command / Observation

命令が受け付けられたことを、外部で実際に確認した状態と同一視しません。

```text
Command Business Result
        ↓ separate operation
explicit read boundary
        ↓
Observation / Execution Error
```

業務上の結果・実行時エラー・観測結果・診断結果は、それぞれ区別して扱います。

### Gate B blocker（解消済み）

実際に実行して検証することが直接の阻害条件でしたが、2026年9月19日に解消しました。

- [x] 実際のTEI作業領域で#310を確認
- [x] 変更のない状態で、書込みを伴わない#255を確認
- [x] 検証専用の空のPostgreSQLデータベースで#308を確認
- [x] #387で同じ版Rに対する最終確認が成立
- [x] #273でReadyの再監査が完了

この過程で、ソース位置の追跡に関する不具合2件と、独立した成果物監査で期待値が古くなっていた不具合1件を検出して修正しました（PR [#726](https://github.com/hat47x/tei-platform/pull/726)／[#727](https://github.com/hat47x/tei-platform/pull/727)）。

## Gate B — UIなしで主要Scenarioを再現できる

判定記録：**Ready**（候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）

Ready判定に必要な次の条件は、すべて満たしています。

- #310・#255・#308が同じ候補版`R`を対象としている
- 各検証根拠に`reproducibleRevision = true`が記録されている
- ソースの変化や作業ツリーに変更が残った状態を採用しない
- 生成成果物を手作業で編集しない
- PostgreSQLの試験用環境の作成前後で安全条件が成立する
- 合成データ・生成成果物・実際の提供実装・実際の外部資産による検証根拠を相互に代用しない
- アプリケーションの意味の不足をウェブ側へ先送りしない

`Ready`の判定記録の正本は`GATE_B_READINESS_AUDIT.md`です。

## Gate B → Gate C Handover Baseline

関連: `GATE_B_TO_C_HANDOVER_CONTRACT.md` / `ISSUE-000030` / #291

判定段階BのReadyを受け、候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`を引継ぎ基準版`B`として固定しました。記録は[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](gate/GATE_B_TO_C_HANDOVER_BASELINE_B.md)にあります。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
```

少なくとも次の情報まで追跡できます。

- 引継ぎ基準版のID
- 基準となるリポジトリの版`R`
- 参照用の作業領域
- アプリケーション`order_entry`
- パターン・データ・関係・問い合わせ・データ接続先
- 検証規則の評価
- 認証と認可
- 能力とその対応付け
- 業務上の結果・実行時エラー・観測
- ドライバーのシナリオ
- 判定段階Bの検証根拠
- 未解決事項

引継ぎの記録を、意味についての第二の正本とはしません。

Oracleについて未確認の範囲を、ウェブ側で推測して補いません。

## Stage 3 — 第一Web Target

関連：`ISSUE-000007`／#70  
検証根拠：`ISSUE-000028`／#270  
契約：`GATE_C_EVIDENCE_CONTRACT.md`

### 着手条件

**判定段階BがReadyとなり、引継ぎ基準版を固定した後**に着手します。2026年9月19日時点で両方の条件が成立し、作業を開始しました。同日、判定段階Cの検証根拠を実際に取得し、Readyと判定しました（`T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`、判定記録：[`GATE_C_READINESS_AUDIT.md`](gate/GATE_C_READINESS_AUDIT.md)）。

### baseline RとTarget build T1

ウェブ向け実装の版は、`T1`として別に追跡します。

```text
Handover Baseline B at R
        ↓ meaning baseline
First Web Target build T1
        ↓
Gate C Evidence
```

通常、`T1 != R`となります。

`T1`の実装で基準版の対象の意味を変更する必要が生じても、ウェブ固有の項目や規則で補いません。権威ある入力元に戻り、必要なドライバーと判定段階Bの検証根拠を再確認して、新しい引継ぎ基準版を発行します。

### 対象Scenario

- browse
- record open
- reference lookup
- 見出し・明細の処理 read
- 見出し・明細の処理 edit（baselineにwrite意味がある場合）
- 検証規則の評価 satisfied / not satisfied
- 認可で許可される場合と拒否される場合
- command
- Business Result
- 実行時エラー
- 観測
- 標準の派生表示
- 個別実装の画面

### 観測する差

1. Application meaning
2. Target-neutral Presentation候補
3. Target Capability候補
4. Runtime Binding差
5. execution artifact / ABI / Host差（観測可能な範囲）
6. Execution / Trust差
7. Web固有実装
8. Host control plane差
9. Canonicalへ持ち上げてはいけない差
10. unresolved

## Gate C — 第一Targetの境界をEvidenceで説明できる

最初のウェブ向け画面が表示できただけでは、判定段階Cを通過したとはみなしません。

検証根拠は次の三種類に分けます。

1. 意味の準拠確認
2. 操作と機能の検証根拠
3. 表示の忠実度の検証根拠

すべての実行結果から、引継ぎ基準版`B`・版`R`と実行先の構築版`T1`を追跡できる必要があります。

判定段階Cで確認すること：

- ドライバーとウェブが同じアプリケーションの意味を共有する
- 基準版`R`と実行先の構築版`T1`を混同しない
- 意味の準拠確認が成立
- 参照シナリオの機能の検証根拠が成立
- 再生可能な動作を利用する場合、意味上の操作と実行先の元の検証根拠まで追跡できる
- 表示の検証根拠を別に保持
- 業務上の結果・実行時エラー・観測を区別して維持
- ウェブの変換器を取り除いても正本が変わらない
- 標準の派生表示と個別実装の画面が同じ上位の意味へ接続
- 実行先の能力・実行時の対応付け・実行成果物・ABI・ホスト・実行と信頼の違いを二つ目の実行先へ引き継げる
- 未解決事項を成功として扱わない

表示の評価点や視覚言語モデルによる判定だけで、意味や機能の検証を合格にはしません。

## Stage 4A — Authoring / Knowledge / AI

判定段階Cが`Ready`となった後に着手します。

### Studio minimal Authoring Loop — `ISSUE-000046`

製品受入シナリオ：PAS-01

```text
Select
  ↓
Propose
  ↓
Preview Semantic Diff / impact
  ↓
Accept / Reject
  ↓ Accept only
Shared Acceptance Boundary
  ↓
authoritative source
  ↓ reload / Conformance
```

編集環境は表示と記述を扱いますが、意味の第二の正本ではありません。

ファイルを直接扱う方式・CLI・AI支援と、同じ受理の原則に従います。

開発者向けのStudioのWailsホストと、生成されるWails製デスクトップアプリケーションの実行先を混同しません。

### knowledge-first Semantic exploration — `ISSUE-000047`

製品受入シナリオ：PAS-06

人とAIが新たな記述を始める前に、既存のデータ・関係・検証規則・能力・アプリケーションパターン・検証根拠を読取り専用で探索します。

探索候補を次の三つに分類します。

- applicable
- incompatible
- unknown

検索順位・類似度・埋め込み表現・AIの確信度は、意味に関する検証根拠ではありません。

### AI Proposal / Acceptance — `ISSUE-000027` / #267

製品受入シナリオ：PAS-01／PAS-06

AIによる提案には、意味の正本を変更する権限はありません。

- 正本・ソース・検証根拠への明示的な参照
- `baseModelRevision`
- 意味上の差分の事前表示
- 古い版に基づく競合
- 共有の受理手続き
- 受理済みソースの再読込み
- 検証規則の評価 / Conformance
- 自律的な作業と変更権限の分離

AI支援の都合だけで、手作業で記述する安定IDの必須化や、パッチだけに限定する記述方式を導入しません。

## Stage 4B — 第二Target pressure test

関連：`ISSUE-000026`／#259  
製品受入シナリオ：PAS-04

段階4Aが完成するのを待つ必要はなく、判定段階Cが`Ready`となった後は並行して着手できます。

二つ目の実行先は、対応先の数を増やすことではなく、最初のウェブ向け実装から得られた設計上の境界を検証することを目的として選びます。

比較する観点は次のとおりです。

- 引継ぎ基準版と正本上の意味
- 実行先の構築版`T1`・`T2`
- 実行先の能力
- 派生表示の違い
- 実行時の対応付け
- 実行成果物・ABI・ホスト
- 実行と信頼に関する設定
- 認可と資源
- 観測
- 意味・機能・表示の検証根拠
- RQ-024の性能測定

生成されるWails製デスクトップアプリケーション、モバイル、サーバー、API、MCPのアプリケーション実行先などが候補です。

StudioのWailsホストは、二つ目のアプリケーション実行先の検証根拠には数えません。

二つ目の実行先で境界の修正が必要になっても、それ自体を失敗とはみなしません。最初の実行先に固有の都合を共通の意味として誤って取り込んでいなかったかを確認します。

## Evidenceの原則

どの段階も、「実装した」という事実だけでは完了と判定しません。

検証根拠として使う候補は次のとおりです。

- 権威あるソース
- ソース索引と診断結果
- 決定論的な試験用データ
- 準拠確認の試験
- 生成成果物と期待出力
- ドライバーのシナリオ
- 実際の提供実装を通じた観測
- 実際の外部資産に対する観測
- 引継ぎ基準版
- 実行先の構築版と実行記録の識別情報
- 意味・機能・表示の検証根拠
- 実行先の比較
- 設計判断記録
- 合格判定の監査
- 未解決事項 Diagnostic

検証成果物や引継ぎ記録を、正本の意味を定義する第二の正本にはしません。

実験結果が予想と異なる場合は、違いを隠して計画に合わせるのではなく、要求・構造・計画を見直します。

## まだ固定しないこと

- アプリケーションパターンの永久標準一覧
- 最初のウェブ実行先のフレームワーク
- 完全な表示用の中間表現
- 完全な実行先の能力の分類体系
- Studioの最終的な画面のフレームワーク
- AIの提供事業者・実装
- ベクトルデータベース
- 提案の最終的なスキーマ
- 手作業で記述する安定ID・UUIDの必須化
- パッチだけを使う記述方式
- 引継ぎに関する汎用スキーマ
- 二つ目の実行先の種類
- B2Cに適用する一律の性能基準
- 認証方式の単一化
- 汎用の作業手順実行機構
- イベントソーシングの標準化
- 本番環境における高可用性の一般化

## v1へ進んだと判断する条件

第1版のアプリケーションと最初の実行先の成立を判断する際は、少なくとも次の事項をリポジトリの検証根拠から説明できることを目安とします。

- `order_entry`を意味の定義から最初の実行先まで追跡できる
- 判定段階BのReadyを得た版`R`を引継ぎ基準版として固定できる
- 最初の実行先の検証根拠から基準版`R`と構築版`T1`を追跡できる
- 複数のデータベースを自然に利用できる
- 独自のSQLや個別実装を正本として利用できる
- 既存の認証仕様を保ったまま能力へ接続できる
- アプリケーションパターンをDriver / Targetで共有できる
- 画面がなくてもドライバーで主要シナリオを検証できる
- 標準の派生表示 / Custom UIを同じ上位意味から利用できる
- 業務上の結果・実行時エラー・観測を分離できる
- 意味・機能・表示の検証根拠を混同しない
- 何が未保証なのかを監査記録と未解決事項から判断できる

Studio・AI支援の完成、二つ目の実行先、本番環境の高可用性は、最初のウェブ実行先までの第1版の成立条件には含めません。ただし、それらを判定段階Cの後で検証するための境界と計画上の課題は保持します。

## 関連文書

- `product/value/TEI_PRODUCT_VALUE.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/MASTER_PLAN.md`
- `plan/MILESTONES.md`
- `plan/gate/GATE_A_READINESS_AUDIT.md`
- `plan/gate/GATE_B_READINESS_AUDIT.md`
- `plan/gate/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/gate/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/gate/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/gate/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/adr/0023-command-observation-explicit-read-boundary.md`
- ISSUE-000006-application-driver-vertical-slice.md
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000008-editor-and-ai-followup-note.md
- ISSUE-000026-target-capability-execution-profile-boundary.md
- ISSUE-000027-ai-proposal-acceptance-boundary.md
- ISSUE-000028-first-web-target-conformance-evidence.md
- ISSUE-000029-gate-b-readiness-audit.md
- ISSUE-000030-gate-b-to-c-handover-baseline.md
- ISSUE-000037-gate-b-three-lane-final-evidence.md
- ISSUE-000046-studio-shared-acceptance-authoring-loop.md
- ISSUE-000047-knowledge-first-semantic-exploration.md
