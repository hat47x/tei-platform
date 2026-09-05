# Gate B → Gate C Handover Contract

## 目的

この文書は、Gate Bを通過したApplication Driverの何を、第一Web Targetで比較する基準としてGate Cへ引き渡すかを定めます。

Gate Cでは、「DriverとWebが同じCanonical Application上の意味を共有している」ことをSemantic Conformanceとして確認します。しかし、Web実装中にもDriverやReferenceは変化し得ます。比較対象を曖昧な「最新Driver」にすると、どの意味に対してConformanceを確認したのかを後から説明できません。

そのため、Gate B通過時に**Handover Baseline**を一つのrepository revisionへ固定し、Gate CのEvidenceから必ずそのbaselineへ戻れるようにします。

この契約は新しいCanonical schemaを導入するものではありません。Git revision、既存Canonical ID、Reference、Gate B Evidenceへの参照を束ねる、検証上のhandover契約です。

Upstream traceでは、PAS-02をprimary、PAS-03をsecondaryとして扱います。

## 2026-09-04 現行Gate B scope（ISSUE-000035 / #380）

`ISSUE-000035` / #380の再監査により、Gate Bの必須完了Evidenceは次の三つへ整理しました。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider Evidence

Gate Bのcandidate revisionを`R`とする場合、この三つのsource-dependent Evidenceを同じ`R`へ揃え、各Evidenceが要求する`reproducibleRevision = true`等の条件を満たしたうえで最終監査を行います。

external-observed assetについて、未観測事実を推測しないこと、external EvidenceをCanonicalへ昇格させないこと、明示authoringとEvidenceを照合することはGate Bの意味境界として引き続き必須です。#310にはexternal Evidence intake boundaryのConformanceを含めます。

`ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376のactual Oracle Evidenceは削除しませんが、Gate B Readyの必須blockerにはしません。

Handover時の扱いは次の二通りです。

- actual Oracle Evidenceが取得済みなら、追加のverified external asset Evidenceとして参照する
- 未取得なら、`unresolved`として#214 / #371 / #376と未観測事項を明示する

未取得の場合でも、Web側でOracleのphysical object、column、SQL、driver挙動、read-only方式を推測・補完しません。#310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual Oracle Evidenceへ格上げしません。

**Gate Bはこのscope変更だけではReadyになりません。** #310 / #255 / #308の実Evidenceを同一candidate revisionへ揃え、最終監査でReadyと判定する必要があります。第一Web Target / Gate Cはその後に開始します。

## 基本原則

### 1. baselineは「その時点の正本」への参照であり、第二の正本ではない

Handover record自体へApplicationの意味を複製しません。

```text
Authoritative Source / Canonical Application
                 │
                 ├─ Driver execution / Evidence
                 │
                 └─ Handover record
                       └─ source revision / Canonical ID / Evidence reference
```

Handover recordからCanonicalへ逆書きしません。

### 2. Gate Cの比較対象を途中で黙って差し替えない

Gate C開始後にDriverやCanonical Applicationが変わっても、既存のGate C runが参照するHandover Baselineを暗黙に「最新」へ読み替えません。

意味上の変更をGate Cへ取り込みたい場合は、新しいrepository revisionを持つHandover Baselineとして発行し直し、旧baselineとのSemantic Diffと影響範囲を記録します。

### 3. Web側の不足をbaselineへ逆輸入しない

Web実装中に必要な意味がHandover Baselineに存在しないと分かった場合、Web固有fieldや独自ruleで補ってGate Cを通しません。

必要ならApplication / Driver側へ戻り、authoritative sourceを変更し、ValidationとDriver Scenarioを再確認したうえで、新しいbaselineを発行します。

### 4. Gate Bの3 Required Evidenceを一つのbaseline revisionへ揃える

各Required Evidenceが個別にsource-fixedであっても、別々のGit revisionで成功したEvidenceを寄せ集めてGate B Readyにはしません。

Gate Bの最終Ready判定では、次の三つについて、**完了判定に用いるEvidenceを同じrepository revisionへ揃えます**。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL engine + ephemeral fixture Evidence

```text
#310 Repository Evidence ─┐
#255 Artifact Evidence   ─┼─ repositoryRevision = R
#308 PostgreSQL Evidence ─┘
                          ↓
                   Gate B Ready for R
                          ↓
                   Handover Baseline R
```

Rを決めた時点で、Rと異なるrevisionの完了EvidenceしかないRequired Evidenceは、Rで再実行・再受入れしてからReady判定します。「意味は変わっていないはずだ」と推測して、古い成功EvidenceをRの成功へ読み替えません。

actual Oracle Evidenceはこの三つとは別に扱います。取得済みでHandover Baseline `R`に対するverified claimとして参照する場合は、actual observed artifact、source-fixed acceptance receipt、actual read、対象runtime / realization / repository revisionの関係を明示します。

external observed artifactそのものの`observedAt`が`R`の時点である必要はありません。以前に観測した資料を使えます。ただし、その資料を`R`に対するverified claimとして扱う受入れとactual readのprovenanceは明示します。

古いOracle Evidenceを`R`の成功へ黙って読み替えません。`R`への十分な結び付きがない場合、またはactual readが未完了の場合は、その範囲を`unresolved`として残します。Oracle EvidenceをGate B Readyの第四Required laneにはしません。

## Handover Baselineに必要な情報

具体的なJSON schemaやfile formatはGate B通過時に決めても構いませんが、少なくとも次を復元できる必要があります。

### Repository / Reference identity

- Handover Baselineとなる単一repository revision
- Reference Workspace path / identity
- そのrevisionに保存されているgenerated artifact identity
- Gate B最終監査の対象revision
- 3 Required Evidenceの完了Evidenceが参照するrepository revision
- actual Oracle等を参照する場合、そのEvidenceが対象とするrevision / realizationとverified / unresolvedの状態

Handover用に`generated artifact revision`という別versionを新設しません。#255の完了Evidenceによって、Handover Baseline revisionに保存されたWorld / Plugin Lock / Durable Runを、その同じrevisionから再現できることを示します。

独自のWorkspace fingerprintも、この契約のためだけには先行導入しません。Git revisionと既存artifact identityだけでは再現性が足りないことがEvidenceから確認された場合に検討します。

### Canonical Application baseline

少なくとも第一ReferenceでGate Cへ渡すものをCanonical IDで列挙します。

- Application: `order_entry`
- Application Pattern
  - browse
  - record
  - reference
  - header-detail
  - command
- Data / Relation / Query
- Datasource
- Validation
- Authentication / Authorization Capability
- Command Capability
- Capability Binding
- Business Result / named Outcome
- Observationで利用する明示read境界

この列挙は意味の複製ではなく、「このGate Cで比較対象に含めるCanonical identity」のmanifestです。

### Driver Scenario baseline

Gate Bで成立したScenario identityを記録します。

第一Referenceでは少なくとも次を含みます。

```text
Authentication
  -> browse
  -> record Pattern resolution
  -> reference Customer
  -> header-detail OrderLine[]
  -> Canonical Validation
  -> Authorization
  -> Command Business Result
  -> separate Observation read
```

Scenarioのtest名、再現手順、または安定したScenario IDのいずれかから、対象Scenarioへ戻れる必要があります。

### Result boundary

Gate Cへ、次の意味の区別を明示的に引き渡します。

- Business Result
- named Outcome
- Execution Error
- Observation read result
- Observationの期待状態評価
- developer Diagnostic

Target側のHTTP status、DOM、toast、dialog等を、この意味の定義には使いません。

### Datasource / authority boundary

少なくとも次を引き渡します。

- read / read-writeの区別
- external-observed / managed-here等、Evidence上必要なownership性質
- Database instance / executor Capabilityの明示Binding
- secret / credential / runtime parameterを通常Evidenceへ保存しない制約

### Gate B Evidence references

Gate Bを通過させたEvidenceへ戻れる必要があります。

Handover recordではEvidence本文を複製せず、Required Evidenceとoptional / unresolved external Evidenceを分離して参照します。

#### Required — #310 deterministic Repository

`tools/reference-gate-b-repository-evidence`の成功JSONを参照します。

少なくとも次へ戻れる必要があります。

- `repositoryRevision = Handover Baseline revision`
- `reproducibleRevision = true`
- Canonical Workspace Validation成功
- `final-ui-less-scenario`成功
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功

#### Required — #255 generated artifact reproducibility

`tools/reference-generated-artifact-evidence`のcheck-only成功JSONを参照します。

少なくとも次を要求します。

- `repositoryRevision = Handover Baseline revision`
- `reproducibleRevision = true`
- World / Plugin Lock golden一致
- Durable Run golden一致

`-write`による修復操作そのものや、修復直後のdirty treeを完了Evidenceには使いません。

#### Required — #308 real PostgreSQL engine

`tools/reference-postgres-header-detail-runtime`の成功JSONを参照します。

少なくとも次を、同じrunで確認したEvidenceへ戻れる必要があります。

- `repositoryRevision = Handover Baseline revision`
- `reproducibleRevision = true`
- `fixtureDatabaseSafety.guardPassed = true`
- `fixtureCleanupSafety.verified = true`
- 2件 / 0件 / Execution Error
- Semantic mapping / support column非露出

#### Optional / unresolved — #214 actual external Oracle

actual Oracle Evidenceが取得済みで、Handover Baselineに対するverified external asset Evidenceとして参照する場合は、少なくとも次へ戻れる必要があります。

- actual external Evidence artifactの参照
- `evidenceArtifactFingerprint`
- source-fixed acceptance receipt
- 実効的なread-only境界を使ったOracle actual read Evidence
- actual readが対象としたruntime / realization / repository revision

acceptance receiptだけをactual read Evidenceへ格上げしません。

取得済みEvidenceがHandover Baseline revisionに対するverified claimとして不足する場合や、actual read自体が未完了の場合は、その不足を`unresolved`として記録します。#310 synthetic Evidenceや#308 PostgreSQL Evidenceで代替したことにはしません。

#### Gate B最終監査

`GATE_B_READINESS_AUDIT.md`の最終Ready判定も、同じHandover Baseline revisionを対象とした判定として追跡します。

「#310はR1、#255はR2、#308はR3で成功した」という集合を、revisionを揃えないままR3のGate B Ready Evidenceとして扱いません。

actual Oracle EvidenceはこのReady判定集合へ加えません。取得済みなら別のverified external asset Evidenceとして、未完了または対象revisionへの結び付きが不足するなら`unresolved`として追跡します。

### unresolved

Gate B通過を妨げないものの、Gate Cまたは第二Targetで継続確認する事項を列挙します。

actual Oracleの未観測範囲も、Gate B blockerへ戻さずこの欄で保持できます。

unresolvedを空欄にするためだけに、十分なEvidenceのない共通概念へ押し込みません。

## Gate C Evidenceからの参照

`GATE_C_EVIDENCE_CONTRACT.md`で要求する`Driver baseline identity`は、このHandover Baselineを指します。

各Gate C Evidence runから、少なくとも次の経路を辿れるようにします。

```text
Gate C Evidence run
       ↓
Handover Baseline identity
       ↓
repository revision R
       ↓
Canonical Application / Driver Scenario
       ↓
Gate B final audit for R
       ↓
3 Required Evidence completion for R
       ↓
optional verified external Evidence / unresolved
```

これにより、Web Adapterやbrowser automationのrevisionだけでなく、「どの意味baselineに対するConformanceか」を固定できます。

## baseline変更時の扱い

Gate C進行中にApplication / Driverを変更する場合、変更を次の三種類へ分けます。

### A. Target実装だけの変更

例:

- CSS
- DOM構造
- Web Adapter内部実装
- browser automation

Canonical / Driver baselineが変わらなければ、Handover Baselineは変更しません。

### B. Evidence / Diagnosticだけの変更

意味を変えないEvidence metadataやDiagnosticの改善であれば、元のCanonical / Driver baselineを維持できるか確認します。

repository revisionが変わるため、必要なら「baseline source revisionは同じで、evaluator / target revisionだけが変わった」と区別して記録します。

Gate B完了Evidenceの意味または評価方法を変更する場合は、旧Evidenceを新しいbaselineへ黙って読み替えず、影響するRequired Evidenceを再実行します。

actual Oracle等のexternal Evidenceの評価方法だけを変更した場合も、そのEvidenceのverified / unresolved状態を更新しますが、それだけでGate B Readyを再判定しません。

### C. Application / Driverの意味変更

例:

- Validation rule変更
- Relation / Query / Datasource変更
- Capability / Binding変更
- Business Result / Outcome変更
- Observation read境界変更
- Pattern上の意図変更

この場合は旧baselineへ黙って上書きしません。

1. authoritative sourceを変更する
2. Driver / Validation / Gate Bに関係するEvidenceを再確認する
3. #310 / #255 / #308の3 Required Evidenceを新しいbaseline revisionへ揃える
4. actual Oracle等のexternal Evidenceについて、新revisionへ対するverified claimとして再利用できるかを明示的に判定し、不足すれば`unresolved`へ戻す
5. 新しいHandover Baselineを発行する
6. Gate Cで影響するScenarioを再実行する

## Gate B通過時に作るHandover record

Gate BがReadyになった時点で、Repositoryへ保存する必要があるかを判断します。

保存する場合も、最初から大きな汎用schemaにはしません。第一Referenceで必要な最小項目を持つmachine-readable artifact、またはMarkdown + stable referencesから開始できます。

重要なのは形式ではなく、次を満たすことです。

- 一つのbaseline revisionが固定される
- #310 / #255 / #308の3 Required Evidenceが同じbaseline revisionへ揃っている
- Canonical IDへ戻れる
- Driver Scenarioへ戻れる
- Gate B Evidenceへ戻れる
- actual Oracle等のexternal EvidenceをRequired Evidenceと分離してverified / unresolvedとして表せる
- unresolvedを保持できる
- secretを含まない
- Target固有情報をCanonical baselineへ混ぜない

## Gate C開始条件

第一Web Targetの本格実装は、Gate B Readyに加えて次を満たしてから開始します。

- Handover Baselineのrepository revisionを固定できる
- #310 / #255 / #308の3 Required Evidenceが同じrepository revisionへ揃っている
- Gate C対象のCanonical identityを列挙できる
- Driver Scenario baselineを再現できる
- Gate B Evidenceへの参照が揃っている
- Gate B最終監査が同じrevisionをReadyと判定している
- actual Oracle等のexternal Evidenceについて、verified / unresolvedを明示できる
- unresolvedを明示できる

これは新しいGateを追加するものではありません。Gate Bの出力をGate Cの入力へ曖昧なく接続するためのhandover確認です。

## 非目標

- 新しいCanonical Application schema
- Presentation IRの先行設計
- Web frameworkの選定
- Handover artifactからCanonicalへの逆生成
- Git commitだけをSemantic Conformanceの証明とみなすこと
- Evidenceの成功revisionが異なるまま一つのbaselineへ読み替えること
- actual Oracle EvidenceをGate B Readyの第四Required laneへ戻すこと
- external-observed assetの未観測事実をsynthetic / PostgreSQL Evidenceから推測すること
- 外部観測資料の観測時刻をGit revision時刻へ無理に合わせること
- 第一TargetだけでTarget-neutral taxonomyを完成させること
- Gate C開始前に第二Targetの差を推測すること

## 関連

- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_B_EVIDENCE_RUNBOOK.md`
- `GATE_C_EVIDENCE_CONTRACT.md`
- `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `UPSTREAM_TRACE_AUDIT_2026-09.md`
- `V1_REFERENCE_APPLICATION_PLAN.md`
- `issue/000006-application-driver-vertical-slice.md`
- `issue/000007-first-web-target-projection.md`
- `issue/000028-first-web-target-conformance-evidence.md`
- `issue/000029-gate-b-readiness-audit.md`
- `issue/000030-gate-b-to-c-handover-baseline.md`
- `issue/000035-gate-b-external-provider-scope-audit.md`
