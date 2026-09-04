# Gate B → Gate C Handover Contract

## 目的

この文書は、Gate Bを通過したApplication Driverの何を第一Web Targetの比較baselineとして引き渡すかを定めます。

Gate Cでは「DriverとWebが同じCanonical Application上の意味を共有している」ことをSemantic Conformanceとして確認します。しかし、Web実装中にもDriverやReferenceは変化し得ます。比較対象を曖昧な「最新Driver」とすると、どの意味に対してConformanceを確認したのか後から説明できません。

そのため、Gate B通過時に**Handover Baseline**を一つのrevisionへ固定し、Gate C Evidenceから必ずそのbaselineへ戻れるようにします。

この契約は新しいCanonical schemaを導入するものではありません。Git revision、既存Canonical ID、Reference / Evidenceへの参照を束ねる検証上のhandover契約です。

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

意味上の変更をGate Cへ取り込みたい場合は、新しいrepository revisionを持つHandover Baselineとして再発行し、旧baselineとのSemantic Diff / 影響範囲を記録します。

### 3. Web側の不足をbaselineへ逆輸入しない

Web実装中に必要な意味がHandover Baselineに存在しないと分かった場合、Web固有fieldや独自ruleで補ってGate Cを通しません。

必要ならApplication / Driver側へ戻り、authoritative sourceを変更し、Validation / Driver Scenarioを再確認して新しいbaselineを発行します。

## Handover Baselineに必要な情報

具体的なJSON schemaやfile formatはGate B通過時に決めても構いませんが、少なくとも次を復元できる必要があります。

### Repository / Reference identity

- repository revision
- Reference Workspace path / identity
- 必要なgenerated artifactのrevision
- Gate B監査の対象revision

独自のWorkspace fingerprintをこの契約のためだけに先行導入しません。Git revisionと既存artifact identityで再現性が足りないことがEvidenceから確認された場合に検討します。

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

列挙は意味の複製ではなく、「このGate Cで比較対象に含めるCanonical identity」のmanifestです。

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

Scenarioのtest名、再現手順、または安定したScenario IDへ戻れる必要があります。

### Result boundary

Gate Cへ次の意味の区別を明示的に引き渡します。

- Business Result
- named Outcome
- Execution Error
- Observation read result
- Observationの期待状態評価
- developer Diagnostic

Target側のHTTP status、DOM、toast、dialog等をこの意味の定義には使いません。

### Datasource / authority boundary

少なくとも次を引き渡します。

- read / read-writeの区別
- external-observed / managed-here等、Evidence上必要なownership性質
- Database instance / executor Capabilityの明示Binding
- secret / credential / runtime parameterを通常Evidenceへ保存しない制約

### Gate B Evidence references

Gate Bを通過させたEvidenceへ戻れる必要があります。

少なくとも次を参照します。

- external-observed `master_data`物理 / read-only Evidence
- header-detail実PostgreSQL Evidence
- Reference golden再生成Evidence
- 最終UI-less Scenario通常runner Evidence
- Gate B Readiness Auditの最終判定

Evidence本文をhandoverへ複製しません。

### unresolved

Gate B通過を妨げないがGate C / 第二Targetで継続確認する事項を列挙します。

unresolvedを空欄にするためだけに、十分なEvidenceのない共通概念へ押し込みません。

## Gate C Evidenceからの参照

`GATE_C_EVIDENCE_CONTRACT.md`で要求する`Driver baseline identity`は、このHandover Baselineを指します。

各Gate C Evidence runから少なくとも次を辿れるようにします。

```text
Gate C Evidence run
       ↓
Handover Baseline identity
       ↓
repository revision
       ↓
Canonical Application / Driver Scenario
       ↓
Gate B Evidence
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

Canonical / Driver baselineが変わらなければHandover Baselineは変更しません。

### B. Evidence / Diagnosticだけの変更

意味を変えないEvidence metadataやDiagnostic改善の場合も、元のCanonical / Driver baselineを維持できるか確認します。

repository revisionが変わるため、必要なら「baseline source revisionは同じ、evaluator / target revisionだけが変わった」と区別して記録します。

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
3. 新しいHandover Baselineを発行する
4. Gate Cで影響するScenarioを再実行する

## Gate B通過時に作るHandover record

Gate BがReadyになった時点で、Repositoryへ保存する必要があるかを判断します。

保存する場合も、最初から大きな汎用schemaにしません。第一Referenceで必要な最小項目を持つmachine-readable artifactまたはMarkdown + stable referencesで開始できます。

重要なのは形式ではなく、次を満たすことです。

- revisionが固定される
- Canonical IDへ戻れる
- Driver Scenarioへ戻れる
- Gate B Evidenceへ戻れる
- unresolvedを保持できる
- secretを含まない
- Target固有情報をCanonical baselineへ混ぜない

## Gate C開始条件

第一Web Targetの本格実装は、Gate B Readyに加えて次を満たしてから開始します。

- Handover Baselineのrepository revisionを固定できる
- Gate C対象のCanonical identityを列挙できる
- Driver Scenario baselineを再現できる
- Gate B Evidenceへの参照が揃っている
- unresolvedを明示できる

この条件は新しいGateを追加するものではありません。Gate Bの出力をGate Cの入力へ曖昧なく接続するhandover確認です。

## 非目標

- 新しいCanonical Application schema
- Presentation IRの先行設計
- Web frameworkの選定
- Handover artifactからCanonicalへの逆生成
- Git commitだけをSemantic Conformanceの証明とみなすこと
- 第一TargetだけでTarget-neutral taxonomyを完成させること
- Gate C開始前に第二Targetの差を推測すること

## 関連

- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_C_EVIDENCE_CONTRACT.md`
- `V1_REFERENCE_APPLICATION_PLAN.md`
- `issue/000006-application-driver-vertical-slice.md`
- `issue/000007-first-web-target-projection.md`
- `issue/000028-first-web-target-conformance-evidence.md`
