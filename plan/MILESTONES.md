# TEIのマイルストーン

## 方針

TEI（鼎／テイ）のマイルストーンには期限を設けません。

マイルストーンは「いつまでに終えるか」ではなく、**どの機能と検証根拠がそろえば、その能力が成立したと判断できるか**を示します。

マイルストーンの番号は、必ずしも着手する順番を示しません。特に判定段階Cの後は、マイルストーン5〜7を同じ検証根拠に基づいて並行して進められます。

個別のM0.xxは構成上の細かな確認点として引き続き利用できますが、本書では製品全体としての能力の到達点を扱います。

関連文書：

- `plan/MASTER_PLAN.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`

## 第1版の参照アプリケーションとの対応

`order_entry`を用いて、マイルストーン1〜4を一連の流れで実証します。

```text
マイルストーン1・2
意味の定義・アプリケーション・既存資産
        ↓
判定段階A — Ready
        ↓
マイルストーン3
設計の実行・検証
        ↓
判定段階B — 三つの必須検証根拠でReady（R = e5231839a5dcc62cce1181a2731fcb7c70d16736）
        ↓
引継ぎの基準B・対象版R — 固定済み
        ↓
マイルストーン4
最初のWeb向け実行先・ビルドT1
        ↓
判定段階C — PAS-04の検証根拠
        ↓
   ┌──────────┴─────────┐
   ↓                    ↓
マイルストーン6・7       マイルストーン5
編集環境・AI             二つ目の実行先
PAS-01・PAS-06          PAS-04・RQ-024
```

引継ぎは新しい番号付きの判定段階ではありません。判定段階Bで`Ready`と判断した意味の基準を、判定段階Cへ固定して渡すための契約です。

## マイルストーン0 — 上位の基準をたどれる

### 到達条件

- 製品価値、将来像、設計原則をリポジトリで参照できる
- 製品要求を固定の識別子で参照できる
- 製品の受入シナリオから要求、計画、検証根拠へと追跡できる
- 全体計画、マイルストーン、設計判断記録、計画上の課題の役割が分かれている
- `unresolved`（未解決）を成功として扱わない

### 意味

今後の作業が「何を確かめるためか」を判断できる地図がある状態です。

## マイルストーン1 — 意味を定義し、追跡・検査できる

### 到達条件

- 人が記述した情報源から、正本となる意味までたどれる
- ファイルのパスと意味上の識別子が分離されている
- 既存の形式のソースを正本として保ちながら接続できる
- 診断結果から元のソースへ戻れる
- アプリケーションの構成パターンを、そのアプリケーションが管理する意図として扱える
- ファイルの移動と論理的な名称変更など、意味が変わるかどうかを区別できる

### 意味

「定義を読み込める」だけでなく、意味と元のソースとの関係を保ちながら変更・検査できる状態です。

第1版で用いる五つの構成パターンを、TEI全体で恒久的に使う標準の一覧にはしません。

## マイルストーン2 — 現実の既存業務を扱える

### 到達条件

- アプリケーションとデータベースの関係を実例で扱える
- データベースの契約、元形式のSQL、外部資産を明示的に対応付けられる
- 既存の認証・認可の仕様を保ったまま接続できる
- 機能ごとに標準実装と既存実装を入れ替えられる
- 外部で観測する資産について、まだ観測していない事実を推測で補わない
- 実際の外部資産から得た検証根拠と、合成データ・試験データによる根拠を区別できる

### 意味

理想化したサンプルだけでなく、実際の既存業務の資産を段階的に移行できる状態です。

PostgreSQL以外の実際のデータベースで得る検証根拠も重要です。最初の参照事例ではOracleに関する#214、#371、#376が該当し、この能力とマイルストーン8の提供実装の検証に役立ちます。ただし、判定段階Bにおける四つ目の必須条件にはしません。

## マイルストーン3 — 設計を実行・検証環境で動かせる

製品の受入シナリオ：PAS-02を主とし、PAS-03も対象とする

### 到達条件

- アプリケーションの構成パターンを実行・検証環境で扱える
- 一覧、記録、参照、見出しと明細を、画面なしで扱える
- 検証・認証・認可を実行できる
- 機能を呼び出した際の業務結果と実行エラーを区別できる
- 命令の受付と観測結果を区別できる
- 同じソースを使った参照シナリオを再現できる
- 生成物、実際の提供実装、決定的な再現性についての検証根拠を混同しない

### 判定段階B

判定段階Bで必要な検証根拠は次の三つです。

1. #310 リポジトリの決定的な再現性を示す検証根拠
2. #255 生成物の再現性を示す検証根拠
3. #308 実際のPostgreSQLと一時的な試験データによる検証根拠

三つを変更のない同一の候補版`R`についてそろえ、#387で最終的な整合を確認します。

判定段階Bで`Ready`と判断した後、その`R`を引継ぎの基準として固定します。

## Milestone 4 — 一つの実用TargetでApplicationを通せる

Product Acceptance Scenario: PAS-04 primary

### 到達条件

- Handover Baseline `B` / revision `R`から第一Web TargetへProjectionできる
- Web build revision `T1`をbaseline `R`と別に追跡できる
- browse / record / reference / header-detail / commandを同じApplication meaningへ接続できる
- Validation / Authorization / Capability / Result / ObservationをTarget側で再定義しない
- standard ProjectionとCustom UIを同じ上位意味へ接続できる
- Web Adapterを除去してもCanonical Worldが変わらない

### Gate C

Gate Cでは次を別Evidenceとして扱います。

1. Semantic Conformance
2. Interaction / Functional Evidence
3. Visual Fidelity Evidence

VLM / screenshot scoreだけでSemantic / Functionalを合格にしません。

Gate CではTarget Capability / Execution / Trust / Web固有差を第二Targetへ渡せることも確認します。

## Milestone 5 — 同じ意味を複数Application Targetへ展開できる

Product Acceptance Scenario: PAS-04 primary

### 到達条件

- 第二Application Targetを第一Targetとは別build identityで実証できる
- 同じHandover Baseline、または明示successor baselineを比較基準にできる
- Target Capability差を説明できる
- Runtime Binding差を説明できる
- Execution / Trust Profile差を説明できる
- Target固有実装をCanonicalへ逆流させない
- RQ-024のperformance measurementをTarget / Host条件付きで比較できる

### 意味

Target-neutralが第一Web Targetだけからの仮説ではなく、性質の異なる第二Targetで圧力試験された状態です。

Studio / Editorのdeveloper-facing Wails HostはApplication Target Evidenceへ数えません。

## Milestone 6 — Studioが安全なAuthoring環境として成立する

Product Acceptance Scenario: PAS-01 primary

### 到達条件

- StudioがCanonicalの第二正本にならない
- high-density read / inspectができる
- Select → Propose → Preview → Accept / Rejectの最小authoring loopを実証できる
- file-native / CLI / AIとShared Acceptance Boundaryを共有できる
- Preview / Rejectでauthoritative sourceを変更しない
- Accept後にsourceをStudioなしでreloadできる
- stale `baseModelRevision`を黙って適用しない
- Studio都合でmandatory authored Stable IDを導入しない

### 意味

厳格なmeaning / source境界を壊さず、日常的なdeveloper-facing authoringへ接続できる状態です。

最初のReferenceは`ISSUE-000046`で扱います。

## Milestone 7 — knowledge-first / AI支援が厳格さを軽いUXへ変える

Product Acceptance Scenario: PAS-06 primary / PAS-01 secondary

### 到達条件

- Human / AIが既存Semantic資産をread-only探索できる
- applicable / incompatible / unknownを区別できる
- search ranking / similarityをSemantic Evidenceへ格上げしない
- AI Proposalとaccepted Decision / source changeを区別できる
- ProposalがCanonical / Source / Evidenceへtraceできる
- AI ProposalもShared Acceptance Boundaryを通る
- stale Proposalを黙って適用しない
- autonomyとauthorityを分ける
- Derived Knowledgeを削除してもCanonical Worldが変わらない

### 意味

AI機能が追加されたというだけでなく、既存知識探索・変更候補・検証を通して利用者負担を減らせる状態です。

read-only探索は`ISSUE-000047`、AI Proposal / Acceptanceは`ISSUE-000027`で扱います。

## Gate C後のMilestone 5〜7の関係

Milestone 5、6、7は厳密な直列ではありません。

```text
Gate C Ready
   ├─→ Milestone 5 第二Target pressure test
   ├─→ Milestone 6 Studio minimal loop
   └─→ Milestone 7 knowledge-first / AI Proposal
```

それぞれのEvidenceを相互に戻します。

- 第二Target差 → Studioが表示すべきTarget差 / AIが参照すべきEvidence
- Studio authoring Evidence → Target-neutral source / acceptance境界
- knowledge-first / AI Evidence → reusable meaning / traceability不足

一つのlaneの完成を他laneの開始条件にしません。

## Milestone 8 — Provider / Runtimeの交換可能性が実証される

### 到達条件

- PostgreSQL以外のDatabase Providerを小さい実例で接続できる
- Provider固有knowledgeがCoreへ漏れない
- actual external asset / Provider Evidenceを取得できる
- external Runtime / orchestratorをCapabilityとして接続できる
- Plugin / Bundle / trust boundaryを説明できる
- unknown / takeover / fencing / recoveryをProvider差を超えて扱える

### 意味

「交換可能」という主張が一つのReference Providerだけに依存しない状態です。

## Milestone 9 — 実利用へ向けた運用境界が整う

### 到達条件

- production永続Store / HA境界が確立される
- credential / secret / Resource authorityが運用可能に分離される
- signed artifact / registry / supply-chain Evidenceを扱える
- compatibility / upgrade policyがある
- observability / audit / failure analysisに必要なEvidenceを取得できる
- production adapterとConformanceを持つ

### 意味

architecture validationを越え、限定された実利用へ責任を持って進められる状態です。

## Version Horizon

### v0 — Architecture / Product Validation

Product Valueと境界を実行可能Referenceで検証する期間です。

現在のM0.xx系列はここに含みます。

### v1 — Practical Developer Platform vertical slice

少なくともMilestone 0〜4を満たし、一つの現実的Applicationを設計・Driver・Database接続・第一Targetまで一続きに扱える状態を目安とします。

Studio / AIの完成、第二Target、production HAはv1 first-target成立の前提にはしません。

ただしGate C後にMilestone 5〜7を検証できるtrace / plan / acceptance境界を残します。

### v2 — Target / Authoring / Provider Expansion

Milestone 5〜8を中心に、複数Target、Studio、knowledge-first / AI、Provider交換可能性を広げます。

Milestone 5〜7は必要に応じて並行します。

### v3以降 — Operational Maturity

Milestone 9を中心に、production運用、互換性、供給網、組織規模での再利用を深めます。

## 先に固定しないもの

- 完全なTarget Capability taxonomy
- 第一Web Target framework
- Studio最終UI
- AI provider / vector database
- Proposal最終schema
- mandatory authored Stable ID / UUID
- Patch-only authoring model
- 第二Targetの種類
- 一律performance threshold
- production HA一般化

Reference / Evidenceが必要性を示す前に固定しません。
