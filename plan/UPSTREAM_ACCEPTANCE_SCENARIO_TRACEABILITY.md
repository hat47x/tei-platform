# TEIの上位受入シナリオと追跡関係

状態：現行の計画上の基準

## 1. 目的

この文書は、TEIの製品価値と製品要求を、利用者が確認できる受入シナリオ、設計判断、計画上の課題、参照事例、検証根拠、判定段階と製品に関する主張まで結び付けるための基準です。

この文書では、新しい製品要求、意味の正本のスキーマ、判定段階、要求の達成率を追加しません。

```text
製品価値・将来像・設計原則
        ↓
製品要求（RQ-xxx）
        ↓
製品の受入シナリオ（PAS-xx）
        ↓
設計判断記録・計画上の課題・設計
        ↓
参照シナリオ・試験用データ
        ↓
検証根拠
        ↓
判定段階・製品についての主張
```

要求の記述だけから実装を直接導くのではなく、利用者や保守担当者が何を確認できれば要求を満たしたと判断できるかを、受入シナリオで示します。

一つのシナリオが成立しても、関連する要求全体を自動的に`verified`へ変更しません。検証根拠の性質、情報源の版、提供実装・実行先・ホスト、未解決事項を区別して判断します。

## 2. 追跡原則

1. 製品要求を実装の都合で言い換えない。
2. 一つの要求が複数の受入シナリオに関係してもよい。
3. 一つの受入シナリオが複数の要求を扱ってもよい。
4. 受入シナリオは、利用者または保守担当者が確認できる結果で記述する。
5. 設計判断記録は受入シナリオを満たすための判断であり、要求そのものではない。
6. 課題のクローズ、テスト合格、CIの成功、画面評価の点数だけでは、受入シナリオが成立したとは判断しない。
7. 合成データ、決定的な再現性、生成物、実際の提供実装、外部資産に関する検証根拠を混同しない。
8. 情報源の内容に依存する検証根拠は取得時の版と結び付け、後続の版へ確認なく引き継がない。
9. 実行先の検証では、意味の基準版と実行先のビルド版を区別する。
10. 不足情報は`unresolved`として残し、成功へ丸めない。
11. 判定段階を増やすより、既存の判定段階で何を証明するかを明確にする。
12. 人による操作、編集環境、コマンドライン、AIなど、変更する経路ごとに意味の第二の正本を作らない。

## 3. PAS-01 — 意味を壊さず設計を変更・再読込できる

### 利用者価値

開発者は、ファイルの直接編集、編集環境、コマンドラインからの構造整理、AIによる変更提案など、どの経路でも変更前後の意味の差と影響を確認できます。共通の受理原則に従い、正本となる情報源を安全に変更し、読み直すことができます。

### 代表Scenario

```text
正本のソースと入力元索引を読む
    ↓
一意に特定できる論理上の識別子を選ぶ
    ↓
変更候補を作る
    ↓
意味の差分と影響を事前に確認する
    ↓
受理・却下を判断する
    ↓ 受理された場合だけ
共通の受理境界
    ↓
正本となるファイルに反映する
    ↓
読み直す
    ↓
意図した意味・識別子・準拠性を再構成する
```

事前確認や却下によって、正本となる情報源は変更しません。

変更候補は必要に応じて`baseModelRevision`を持ちます。元の版が古くなっていた場合は、却下、再読込み、明示的な基準版の更新などを行い、最新版へ確認なく適用しません。

### 識別子についての注意

ファイルのパスそのものを意味上の識別子にはしません。

- ファイルを移動しても一意な論理上の識別子が変わらない場合：意味を変更しない操作として扱える
- 論理上の名称や正本の識別子を変更する場合：意味の変更として扱う

すべての変更後に同じ識別子を維持するのではなく、**変更の意図に対して識別子の変化を説明できること**を要求します。

編集環境やAIの都合だけで、人が記述する安定識別子やUUIDを必須にはしません。

### 主なRequirement

- RQ-001 業務上の意味を実装技術から独立して保持
- RQ-002 Authoring sourceとCanonical meaningの追跡
- RQ-003 暗黙の複数正本を作らない
- RQ-010 Editor
- RQ-011 複数Projection
- RQ-015 AI knowledge-first workflow
- RQ-016 AI suggestion / decision history
- RQ-017 二重管理の最小化
- RQ-019 Runtime / Resource authority
- RQ-021 Reference / Conformance

### 共通の受理境界

```text
ファイルの編集 ---------┐
編集環境 ---------------|
コマンドライン・整理 ----|--> 変更候補 --> 共通の受理境界 --> 正本の情報源
AIによる提案 -----------|
対応表 -----------------|
必要に応じた差分 -------┘
```

対象に応じて次を組み合わせます。

```text
解析・読取り
 → 正規化
 → 検証
 → 構成と方針の確認
 → 意味の差分
 → 影響の分析
 → 受理の方針による判断
 → 正本となる情報源の更新
 → 再読込み
 → 準拠性の確認
```

すべての変更を一つの固定APIや処理手順に限定しません。ただし、変更の入口によって意味の検証規則や権限が変わらないことを求めます。

### 現在のPlan / Evidence位置

判定段階Cより前には、編集環境やAIの詳細な接続方式を固定しません。

Gate C Ready後:

- `ISSUE-000046`：編集環境での最小限の記述・変更・受理（主にPAS-01）
- `ISSUE-000027`：AIの変更提案と共通の受理境界（主にPAS-01、関連してPAS-06）
- ADR-0022：AIの変更提案と意味の正本の受理境界を分ける判断

編集環境は表示と記述を行うための機能であり、意味の第二の正本にはしません。

開発者向け編集環境のWailsホストを、Wailsで生成したデスクトップアプリケーションの実行先についての検証根拠とはみなしません。

## 4. PAS-02 — アプリケーションの意味を実行・検証環境で確認できる

### 利用者価値

開発者は画面やWebの開発基盤を用意する前でも、アプリケーションの意味を実行・検証環境で確かめられます。その際、検証、認可、業務上の結果、実行エラー、観測結果を区別します。

### 代表Scenario

```text
認証
 → 一覧表示
 → 記録
 → 参照
 → 見出しと明細
 → 検証
 → 認可
 → 命令
 → 業務上の結果
 → 独立した明示的な読取り
 → 状態の観測
```

命令が受け付けられたことだけでは、期待した外部の状態になったとは判断しません。

### 主なRequirement

- RQ-005 Application / Database N:M
- RQ-008 Capability単位の実装置換
- RQ-009 既存認証・認可Compatibility
- RQ-012 Driverによる設計検証
- RQ-014 Application Pattern再利用
- RQ-018 Command / Observation分離
- RQ-021 Reference / Conformance

### 現在のEvidence位置

判定段階Aは`Ready`です。実行・検証環境でも、主要な意味の境界が成立しています。

判定段階Bは`Ready`です（候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`、判定日：2026-09-19）。

必要な検証根拠は次の三つです。いずれも変更のない同じ候補版`R`に対して、一度の実行で成立しています。

1. #310 リポジトリの決定的な再現性を示す検証根拠
2. #255 生成物の再現性を示す検証根拠
3. #308 実際のPostgreSQLと一時的な試験データによる検証根拠

`ISSUE-000037` / #387のfinal convergenceと`#273`の独立artifact auditも成立済みで、Handover Baseline `B / R`を固定しています（[`GATE_B_READINESS_AUDIT.md`](gate/GATE_B_READINESS_AUDIT.md) / [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](gate/GATE_B_TO_C_HANDOVER_BASELINE_B.md)）。

## 5. PAS-03 — 既存資産を保ちながら段階的に移行・置換できる

### 利用者価値

既存のSQL、データベース、認証、プラグイン、外部の提供実装などを、TEI独自の形式へ全面的に移し替える必要はありません。意味、契約、実現方法、外部の観測を区別したまま、段階的に利用できます。

### 代表Scenario

```text
正本の契約・アプリケーションの意図
        ↓
明示した対応付け・既存形式での実現
        ↓
提供実装・外部資産
        ↓
命令または問い合わせの実行
        ↓
検証根拠・観測結果
```

まだ観測していない外部の物理的な事実を、推測で補ったり生成したりしません。提供実装に固有の知識は、その提供実装と接続部品に保持し、正本の契約を提供実装の都合で変更しません。

### 主なRequirement

- RQ-004 Native asset利用
- RQ-005 Application / Database N:M
- RQ-006 Desired / Actual / Diff / Unresolved
- RQ-007 Provider固有知識の隔離
- RQ-008 Capability単位の置換
- RQ-009 既存認証・認可Compatibility
- RQ-018 Command / Observation分離
- RQ-019 Permission / Resource制約
- RQ-021 Reference / Conformance

### 現在のEvidence位置

判定段階Bの#308で実際のPostgreSQLを用いる検証は、PAS-03の一部を厳しく確かめるものです。しかし、実際のOracle環境で得る検証根拠の代わりにはなりません。

実際のOracle環境に関する#214、#371、#376は判定段階Bの四つ目の必須条件には加えません。外部資産、既存資産との互換性、二つ目の提供実装についての検証として継続します。

判定段階Bで`Ready`となった時点で未完了の事項は、引継ぎの基準に`unresolved`として記録します。最初の実行先側で物理的な事実やSQL、読み取り専用の方式などを推測で補いません。

## 6. PAS-04 — 同じApplication meaningを複数TargetへProjectionできる

### 利用者価値

同じCanonical ApplicationをWeb、Desktop、Mobile、API、MCP等へ接続しても、Target固有UI / transport / Host事情から業務意味が逆定義されません。

### baselineとTarget buildを分ける

Gate B Ready candidate revisionを`R`、Handover Baselineを`B`、Target実装revisionを`T`とします。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
        ↓ meaning baseline
Target build T
        ↓
Target execution / Evidence
```

通常`T != R`です。

Target buildでbaseline subjectのmeaningを変更する必要が生じた場合、Target固有field / ruleで埋めません。authoritative sourceへ戻り、必要なDriver / Gate B Evidenceを再確認し、successor Handover Baseline `B'` / `R'`を発行します。

旧baseline Evidenceを新baselineへ黙ってcarry-forwardしません。

### Target評価

少なくとも次を分離します。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence
4. 必要に応じたPerformance Evidence

### 主なRequirement

- RQ-001 業務意味の実装技術非依存
- RQ-010 EditorとのProjection境界
- RQ-011 複数Projection
- RQ-013 Target-neutral Presentation / Interaction
- RQ-017 二重管理の最小化
- RQ-021 Reference / Conformance
- RQ-024 Target別性能要求

### 現在のPlan / Evidence位置

Gate B Ready + Handover固定後:

- `ISSUE-000007` — First Web Target Projection
- `ISSUE-000028` — Gate C Conformance Evidence
- `GATE_C_EVIDENCE_CONTRACT.md`

Gate CではすべてのrunからHandover `B` / baseline `R`とWeb build `T1`へ追跡します。

Gate C Ready後:

- `ISSUE-000026` — second Target pressure test

第一Targetだけで完全なPresentation IR / Target Capability taxonomyを固定しません。

StudioがWailsを使うことと、Generated Wails Desktop Application Targetは別です。Studio HostをPAS-04のTarget成功Evidenceへ数えません。

## 7. PAS-05 — 外部状態が不確かな場合でも安全に実行・再開できる

### 利用者価値

外部Runtime / Resourceの状態が不明な場合に、「成功したはず」と推測したり、副作用を無条件にreplayしたりせず、Observation、ownership、fencing、durable handle等を使って安全に判断できます。

### 主なRequirement

- RQ-006 Desired / Actual / Diff / Unresolved
- RQ-018 Command / Observation分離
- RQ-019 Permission / Resource制約
- RQ-020 再実行 / takeover / unknown
- RQ-021 Reference / Conformance

### Evidence位置

Deployment / Plugin / Resource lifecycle等のReferenceで継続します。

PAS-05を現行Gate Bの新しいblockerには追加しません。現在Gate Bに必要な範囲は既存3 Required Evidenceで確認し、より広いrecovery能力は対応Milestone / Referenceで別Evidenceとして扱います。

## 8. PAS-06 — 既存の意味知識を探索してから新規実装へ進める

### 利用者価値

開発者またはAIは毎回ゼロから実装せず、既存Data、Relation、Validation、Capability、Application Pattern、利用例、Evidenceを探索し、適用可能性と不足条件を確認してから変更・構成候補へ進めます。

### 代表Scenario

```text
purpose / intent
   ↓
read-only Semantic exploration
   ↓
identity / dependency / source / evidenceを確認
   ↓
applicable / incompatible / unknown
   ↓
explicit-reference candidate
   ↓
Shared Acceptance Boundary
```

検索順位、文字列類似度、embedding similarity、AI confidenceだけをSemantic Evidenceにしません。

AI Knowledge、index、cache、過去trajectoryはderivedでありCanonicalの第二正本ではありません。

### 主なRequirement

- RQ-002 Source traceability
- RQ-003 単一権威
- RQ-014 Application Pattern再利用
- RQ-015 knowledge-first AI
- RQ-016 AI suggestion / history
- RQ-017 二重管理最小化
- RQ-021 Reference / Conformance
- RQ-023 一人でも維持可能な構造

### 現在のPlan / Evidence位置

Gate C Ready後:

- `ISSUE-000047` — Human / AI共通read-only Semantic exploration（PAS-06 primary）
- `ISSUE-000027` — exploration結果からAI Proposal / Acceptanceへ接続

最初から巨大Knowledge Graph、vector database、AI専用Knowledge Storeを作りません。

## 9. PAS-07 — 手作業を失わずData / Function / Businessを相互検証して設計修正できる

### 利用者価値

開発者は、業務をApplication内部の実行手順へ還元せず、人・外部作業を含むBusiness Procedureを確認しながらData / Function-Interaction / Businessの三視点から矛盾・不足・過剰な仮定を見つけ、Projectionだけを補修するのではなくauthoritative sourceへ戻って設計を修正できます。

### 代表Scenario

```text
accepted Data / Relation / Application meaningを読む
        ↓
Business固有meaningをaccepted authoring sourceとして保持する
        ↓
manual / external Actionをfake Functionへ変換しない
        ↓
existing Application / Semantic identityをexplicitに参照する
        ↓
derived cross-view Projectionで相互検証する
        ↓
contradiction / omission / unknown / impactを発見する
        ↓
correction candidateをauthoritative source側へ返す
        ↓
Shared Acceptance Boundary
        ↓
reload / regenerate / reviewする
```

### 壊してはいけない境界

```text
Business Action != Application Function
Business Flow != Procedure != Runtime Workflow Orchestration
Business Actor / Job != Authentication Principal != Authorization Role / Policy
Validation PASS != whole-business approval
Application Business Result != external-world Observation
Projection != second source of truth
```

Application Patternの並びやPAS-02のDriver validation sequenceを、accepted Business Procedureなしに利用者の業務手順へ読み替えません。

### 主なRequirement

- RQ-025 業務上の行為・手順をApplication Functionから独立して設計
- RQ-026 Data / Function-Interaction / Business Procedureの相互検証
- RQ-021 Reference / Conformance
- RQ-022 設計検証段階と完成製品の区別

RQ-025はPAS-07 primary / PAS-02・PAS-04 secondaryです。

RQ-026はPAS-07 primary / PAS-01・PAS-02・PAS-04 secondaryです。

### 現在のPlan / Evidence位置

Reference / Evidence実装はLane C `ISSUE-000042` / PR #482が所有します。Lane BはReference fixture / checkerを複製せず、EvidenceをPAS-07 claimへ対応付けます。

現時点では次を分離します。

```text
E7-01〜09
  Lane C fixture / checker / pressure tests implementation exists
  actual Repository runner Evidence unresolved

E7-10
  file-native accepted-source pressure exists
  production Shared Acceptance end-to-end Evidence unresolved
```

したがって:

```text
implementation exists
  !=
PAS-07 execution Evidence accepted
```

です。GitHub-hosted runnerがcheckout前に終了する状態やCI表示だけをcode failure / PAS failureへ読み替えません。

Business-side Representation EvidenceとPAS-02 Runtime Evidenceも分けます。

```text
Lane C Business Procedure Reference
  -> manual / external Action
  -> Business judgment / Validation separation
  -> actor / authority separation
  -> Command / Observation separation
  -> cross-view reference / diagnostic

PAS-02 Driver Evidence
  -> Validation execution
  -> Authentication / Authorization decision
  -> Capability execution
  -> Business Result
  -> separate explicit read / Observation
```

片方をもう片方の代替にしません。

また、Reference-local stable Procedure keyはreview / handoff / stale detection / file move分離に具体的価値がありますが、それだけでCanonical `Procedure` / `Step`を採用しません。

file-native Business Procedure sourceのShared Acceptanceを安全に行う機械的pressureはCanonical Procedureなしでも成立しており、現在残るidentity論点は主にRQ-002のCanonical trace targetです。minimal Canonical identity shell、Subject contract拡張、formal Authoring identity等の比較はArchitecture pressureとして別に扱い、PAS-07成立条件からfull Workflow modelを逆導出しません。

### Gateへの影響

PAS-07は新Gateではありません。

- Gate B Required Evidence 3 laneへ追加しない
- Gate C First Web Target Required Evidenceへ追加しない
- Lane C actual runner greenはPAS-07 Evidence statusを更新するが、それだけでPAS-07全体を`accepted`へ上げない
- end-to-end correction / acceptance / reloadはPAS-01 Shared Acceptance Evidenceとcross-checkする

詳細なE7 mappingは `RQ025_RQ026_ACCEPTANCE_TRACE.md` をsupporting Evidence contractとして参照します。

## 10. Cross-cutting Requirements

### RQ-021 — Reference / Conformance

すべてのPASに横断します。

「設計上できる」ではなく、小さいReference、fixture、Conformance、Evidenceから境界を再現できることを要求します。

### RQ-022 — 完成製品と設計検証段階を区別

すべてのPASに横断します。

architecture probe、local fixture、synthetic adapter、beta framework、actual provider run、production claimを同じ`done`へ畳みません。

### RQ-023 — 一人でも維持可能な開発構造

主にPAS-01 / PAS-06へ関係しますが全体の作業分割へ横断します。

同じRepository内でRequirement → PAS → Decision → Reference → Evidenceを追えることを優先し、不要な第二正本 /巨大subsystemを増やしません。

### RQ-024 — Target別性能

PAS-04で具体化します。

第一Target `T1`と第二Target `T2`の実測を、Handover baseline、Target / Host、Runtime / Provider条件と一緒に比較します。

根拠のない一律thresholdを第一Targetだけから固定しません。

### RQ-034 — 標準的な開発環境からの改修・実行・検証

主にPAS-06で開発contextへの機械到達を確認し、PAS-01で通常のSource変更が既存のAuthoring / Acceptance境界へ戻れることを確認します。

ChatGPT等のチャット実行環境、Codex、Claude Code、一般的なIDE / CLI、CI等を、TEI専用Agent Runtimeの下位実装にはしません。各環境は独立したcheckout / worktreeでSourceを編集し、TEI CLI、Host-provided MCP、通常のToolchainを使ってBuild / Testまで進め、Git等の既存経路から変更を受け渡せます。TEI固有のdeveloper MCP serverは必須にしません。

Referenceでは、Agent Session、Checkpoint、一般的な検証状態等をTEIのProduct stateへ追加せずに、この一往復が成立するかを確認します。Agent Adapterや独自Change Setが無い場合にも成立することをnegative boundaryとして扱います。

## 11. Requirement → Primary PAS mapping

| Requirement | Primary PAS | Secondary / cross-cutting |
| --- | --- | --- |
| RQ-001 | PAS-01 | PAS-04 |
| RQ-002 | PAS-01 | PAS-06 |
| RQ-003 | PAS-01 | PAS-06 |
| RQ-004 | PAS-03 | - |
| RQ-005 | PAS-02 | PAS-03 |
| RQ-006 | PAS-03 | PAS-05 |
| RQ-007 | PAS-03 | PAS-04 |
| RQ-008 | PAS-02 | PAS-03 |
| RQ-009 | PAS-02 | PAS-03 |
| RQ-010 | PAS-01 | PAS-04 |
| RQ-011 | PAS-04 | PAS-01 |
| RQ-012 | PAS-02 | PAS-04 |
| RQ-013 | PAS-04 | - |
| RQ-014 | PAS-02 | PAS-06 |
| RQ-015 | PAS-06 | PAS-01 |
| RQ-016 | PAS-06 | PAS-01 |
| RQ-017 | PAS-01 | PAS-04 / PAS-06 |
| RQ-018 | PAS-02 | PAS-03 / PAS-05 |
| RQ-019 | PAS-05 | PAS-01 / PAS-03 |
| RQ-020 | PAS-05 | - |
| RQ-021 | cross-cutting | all PAS |
| RQ-022 | cross-cutting | all PAS |
| RQ-023 | cross-cutting | PAS-01 / PAS-06 |
| RQ-024 | PAS-04 | - |
| RQ-025 | PAS-07 | PAS-02 / PAS-04 |
| RQ-026 | PAS-07 | PAS-01 / PAS-02 / PAS-04 |
| RQ-027 | PAS-06 | PAS-01 |
| RQ-028 | PAS-01 | PAS-06 / PAS-07 |
| RQ-029 | PAS-06 | PAS-01 / PAS-03 |
| RQ-030 | PAS-07 | PAS-02 / PAS-05 |
| RQ-031 | PAS-06 | PAS-01 / PAS-07 |
| RQ-032 | PAS-06 | PAS-01 / PAS-07 |
| RQ-033 | PAS-06 | PAS-01 |
| RQ-034 | PAS-06 | PAS-01 |

Primaryは最初に具体化する代表PASであり、coverage scoreではありません。

RQ-027〜RQ-030は2026-09-18のProduct再定義で追加されましたが、この表は2026-09-21までRQ-026で止まっていました。そのため4件はPrimary PASもEvidence tierも持たず、`PRODUCT_REQUIREMENT_GAP_AUDIT.md`の優先順位にも現れませんでした。上の4行は2026-09-21に割り当てたもので、最初に具体化する代表PASの提案です。`tools/repository-plan-check`の`TestEveryRequirementIsMappedToAnAcceptanceScenario`が、以後この表と監査表の両方がRequirement集合と一致することをfail-closedで確認します。

RQ-031〜RQ-033は2026-10-03に、固定されたモデル体系ではなく意味論を後から拡張・合成できることをProduct Valueへ明示した際に追加しました。現時点では新しいPASを増やしません。まずPAS-06のknowledge-first探索を主な検証面とし、Shared Acceptanceを扱うPAS-01と、複数の意味視点を相互検証するPAS-07を必要に応じて組み合わせます。独立したAcceptance Scenarioが必要かどうかは、Semantic ExtensionのReferenceで既存PASでは表せない利用者レベルの成立条件が確認された時点で判断します。

## 12. 現行Gate / Stageとの関係

```text
Gate A
  -> PAS-02のApplication meaningをDriverへ渡せることを確認

Gate B
  -> PAS-02中心 + PAS-03のreal-provider境界を含む3 Required Evidenceをsame-Rへ収束

Gate B → C Handover
  -> PAS-02 / PAS-03で確認したbaseline revision RとunresolvedをBとして固定

Gate C
  -> PAS-04をFirst Web Target build T1で実証
  -> baseline RとTarget build T1を分離

Gate C後 — 並行可能
  -> PAS-01: ISSUE-000046 Studio minimal authoring loop
  -> PAS-06: ISSUE-000047 knowledge-first exploration
  -> PAS-01 / PAS-06: ISSUE-000027 AI Proposal / Acceptance
  -> PAS-04: ISSUE-000026 second Target pressure test

Gate非依存のparallel Reference pressure
  -> PAS-07: ISSUE-000042 Business Procedure cross-view Reference
  -> Gate B / Gate C Required Evidenceへ追加しない
```

PAS-05はruntime / deployment workstreamへ接続します。現行Gate B必須scopeを本書から拡張しません。

PAS-07のReference runnerがgreenになってもGate B / CのReady条件は変わりません。greenはPAS-07 Evidence statusへ記録し、Shared AcceptanceやPAS-02 Runtime Evidenceを自動代替しません。

## 13. 新しいADR / Plan Issueへ要求する最小Trace

新しい上流設計・機能提案では、必要に応じて次を明示します。

- 関連Product Requirement
- Primary PAS
- Scenarioで失敗すると利用者 / maintainerに何が見えるか
- 保存するsemantic boundary
- Provider / Target / Hostへ残す差
- baseline / source / build identity
- Reference / fixture
- 必要なEvidence nature
- 既存Gateへ影響するか否か
- unresolved
- reopen condition

実装が先に存在する場合でも、この欄を埋めるためだけに架空Requirementを作りません。

## 14. 現行Authoring基線

PAS-01 / PAS-06では次を現行前提とします。

- authoritative sourceはfile-nativeで保持可能
- qualified logical key / Canonical identityをfile pathと分離
- Studio / CLI / AI等はcandidate入口
- Preview / Rejectはnon-mutating
- AcceptはShared Acceptance Boundaryへ収束
- accepted sourceは入口toolなしでreload可能
- stale base revisionは明示処理
- derived index / AI Knowledge / trajectoryは第二正本ではない
- 独立した開発環境は、TEI専用Agent Adapterやnative MCP serverなしでSourceを直接編集し、CLI / Host-provided MCP / 通常のToolchainでBuild / Testまで行える
- Git commit / branch / Pull Request等を変更の第一級の受け渡し経路として利用でき、Gitで足りる場合にTEI独自Change Setを要求しない

PAS-07のBusiness Procedure pressureでも、Business固有meaningをfile-native authoring sourceへ保持し、Pattern-owned Data / Relation / Query / Capability / DatasourceやMatrix / diagnostic等のderived Projectionと分離します。

現時点でReference-local stable Procedure keyの有用性は確認していますが、accepted Business sourceをRQ-002上どのCanonical trace targetへ接続するかは`unresolved`です。この一点からfull Canonical Procedure modelやRuntime Workflowを先に固定しません。

次を現行前提にはしません。

- Semantic Patchだけを唯一のauthoring経路にすること
- Base Semantic Digestを全candidateの必須基準にすること
- mandatory authored Stable ID / UUID
- Studio独自semantic authority
- AI独自Canonical Knowledge Store
- Agent Session / Workspace / Checkpoint / 一般的な検証状態をTEIの標準Product stateにすること
- Agent Adapterや特定Chat UIを標準Authoringの必須経路にすること
- Canonical `Procedure` / `Step`の先行採用
- Business ProcedureからRuntime Workflow Orchestrationへの自動変換

optional Patchを利用する場合も、file-native / Studio / CLI / AIと同じAcceptance原則へ接続します。

## 15. 非目標

- Product Requirementsの置換
- Canonical object `ProductAcceptanceScenario`の追加
- runtime Scenario Registryの実装
- Requirement coverage率 / 完成率の算出
- Gate B / Gate Cのscopeを本書だけで変更すること
- 第一Targetだけから完全なPresentation IRを固定すること
- Gate C前にStudio / AIの詳細interfaceを固定すること
- PAS-07を理由にCanonical `Procedure` / `Step`、Workflow Engine、新Gateを自動導入すること
