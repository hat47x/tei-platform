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

## Gate B candidate Ready revision

Gate Bの4 Evidence laneは、それぞれのrun内でsource revisionを固定するだけでは十分ではありません。

最終Gate B監査では、**一つのclean Repository revisionを`candidate Ready revision`として明示し、そのrevisionに依存するEvidenceを同じrevisionへ揃えます。**

たとえば、次のように異なるrevisionのEvidenceを寄せ集めてGate B Readyとは判定しません。

```text
#310 repository Evidence      -> R1
#255 generated artifact       -> R2
#308 PostgreSQL Evidence      -> R3
#214 acceptance receipt       -> R4
```

Gate B Ready候補を`R`とする場合、少なくとも次を要求します。

- #310 `reference-gate-b-repository-evidence`の`repositoryRevision = R`
- #255 `reference-generated-artifact-evidence`の`repositoryRevision = R`
- #308 PostgreSQL Evidenceの`repositoryRevision = R`
- #214 external Evidence acceptance receiptの`repositoryRevision = R`
- 上記source-fixed Evidenceはいずれも`reproducibleRevision = true`
- #214 actual Oracle runtime / Semantic E2Eが、Rのauthoring / runtime境界を用いた実行であることを追跡できる
- 最終Readiness Audit自体がRを監査対象revisionとして明示する

### external factsはGit revisionそのものではない

Oracleの実physical object、raw columns、provenance、read-only権限等の**外部観測事実そのもの**は、Git revisionに属する情報ではありません。

したがって、同じ外部Evidence artifact / provenanceが引き続き有効であることを確認できる場合、観測事実そのものをRepository revisionが変わるたびに捨てる必要はありません。

ただし、その外部EvidenceをTEIのauthoringへ結び付ける次の境界はcandidate Ready revisionへ固定します。

- source-fixed acceptance receipt
- SQL realization / mappingとの照合
- actual Oracle runtime / Semantic E2E

外部Evidence artifactを再利用する場合も、新しいcandidate revisionへ黙ってcarry-forwardせず、そのartifactを新candidateのauthoringへ受理・実行した証跡を取り直します。

### candidate revisionが変わった場合

candidate `R`のEvidence取得中にsource変更が必要になった場合、RのEvidenceを新しいrevisionへ黙って読み替えません。

```text
candidate R
  ↓ Evidenceでsource修正が必要と判明
修正・commit
  ↓
new candidate R'
  ↓
R'でsource-dependent Evidenceを再取得
```

少なくとも #310 / #255 / #308 / #214 acceptance receipt と、TEI sourceに依存する#214 actual runtime EvidenceはR'へ揃え直します。

#255の`-write`でgenerated artifact修復commitが必要になった場合も同じです。修復後のcommitを新candidateとして選び、それ以前のrevisionで取得したsource-dependent Evidenceを新candidateの完了Evidenceへ混ぜません。

### mainが進んでもcandidateを暗黙に動かさない

candidate Ready revisionを選んだ後に`main`が別作業で進んでも、candidateを自動的に「最新main」へ読み替えません。

- 既存candidateをそのままReady監査するなら、そのrevisionとEvidenceを固定したまま扱う
- 後続commitをGate B baselineへ含めたいなら、新candidateを選び直し、必要なsource-dependent Evidenceをそのrevisionへ揃える

この規則により、並行開発によってGate Bの比較対象が常に動き続けることを防ぎます。

### Handover revisionとの同一性

Gate B Readyと判定したcandidate revisionを`R`とした場合、**第一Handover Baselineのrepository revisionも必ずRとします。**

Gate BをRで通過した後、Handoverだけを別revisionへ暗黙に進めません。R以後の変更をbaselineへ含める必要がある場合は、変更の性質に応じてGate B Evidenceを再確認し、新しいcandidate / Handover Baselineを発行します。

## Handover Baselineに必要な情報

具体的なJSON schemaやfile formatはGate B通過時に決めても構いませんが、少なくとも次を復元できる必要があります。

### Repository / Reference identity

- repository revision
- Gate B candidate Ready revisionとの同一性
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
- #214 source-fixed acceptance receipt
- header-detail実PostgreSQL Evidence
- Reference generated artifact reproducibility Evidence
- 最終UI-less Scenario通常runner Evidence
- Gate B Readiness Auditの最終判定

Repository revisionを持つEvidenceは、Gate B candidate Ready revisionと同一revisionであることを確認します。

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
Gate B candidate Ready revision
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
2. 新しいGate B candidate Ready revisionを選ぶ
3. source-dependent Gate B Evidenceをそのcandidateへ揃える
4. Gate B Readyを再確認する
5. 新しいHandover Baselineを発行する
6. Gate Cで影響するScenarioを再実行する

## Gate B通過時に作るHandover record

Gate BがReadyになった時点で、Repositoryへ保存する必要があるかを判断します。

保存する場合も、最初から大きな汎用schemaにしません。第一Referenceで必要な最小項目を持つmachine-readable artifactまたはMarkdown + stable referencesで開始できます。

重要なのは形式ではなく、次を満たすことです。

- repository revisionがGate B candidate Ready revisionと一致する
- revisionが固定される
- Canonical IDへ戻れる
- Driver Scenarioへ戻れる
- Gate B Evidenceへ戻れる
- source-dependent Gate B Evidenceが同じcandidate revisionへ揃っている
- unresolvedを保持できる
- secretを含まない
- Target固有情報をCanonical baselineへ混ぜない

## Gate C開始条件

第一Web Targetの本格実装は、Gate B Readyに加えて次を満たしてから開始します。

- Gate B candidate Ready revisionを一つに固定できる
- Handover Baselineのrepository revisionがcandidate Ready revisionと一致する
- source-dependent Gate B Evidenceがcandidate Ready revisionへ揃っている
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
- 異なるrevisionのGate B Evidenceを寄せ集めてReady扱いすること
- 第一TargetだけでTarget-neutral taxonomyを完成させること
- Gate C開始前に第二Targetの差を推測すること

## 関連

- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_B_EVIDENCE_RUNBOOK.md`
- `GATE_C_EVIDENCE_CONTRACT.md`
- `V1_REFERENCE_APPLICATION_PLAN.md`
- `issue/000006-application-driver-vertical-slice.md`
- `issue/000007-first-web-target-projection.md`
- `issue/000028-first-web-target-conformance-evidence.md`
- `issue/000029-gate-b-readiness-audit.md`
- `issue/000030-gate-b-to-c-handover-baseline.md`