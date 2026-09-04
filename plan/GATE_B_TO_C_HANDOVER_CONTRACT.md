# Gate B → Gate C Handover Contract

## 目的

この文書は、Gate Bを通過したApplication Driverの何を、第一Web Targetで比較する基準としてGate Cへ引き渡すかを定めます。

Gate Cでは、「DriverとWebが同じCanonical Application上の意味を共有している」ことをSemantic Conformanceとして確認します。しかし、Web実装中にもDriverやReferenceは変化し得ます。比較対象を曖昧な「最新Driver」にすると、どの意味に対してConformanceを確認したのかを後から説明できません。

そのため、Gate B通過時に**Handover Baseline**を一つのrepository revisionへ固定し、Gate CのEvidenceから必ずそのbaselineへ戻れるようにします。

この契約は新しいCanonical schemaを導入するものではありません。Git revision、既存Canonical ID、Reference、Gate B Evidenceへの参照を束ねる、検証上のhandover契約です。

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

actual Oracleが未検証の状態でHandoverする場合も、Web側でOracle object / column / datatype / read-only方式 / driver都合を補いません。必要になった時点で#214へ戻します。

### 4. Gate Bの3必須Evidenceを一つのbaseline revisionへ揃える

PR #379を起点とした`ISSUE-000035` / #381の再監査により、actual Oracle固有EvidenceはGate Bの必須完了条件から外し、継続actual external asset Evidenceとして扱います。

Gate Bの最終Ready判定では、次の3 Evidenceについて、**完了判定に用いるEvidenceを同じrepository revisionへ揃えます**。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL engine + ephemeral fixture Evidence

```text
#310 完了Evidence ─┐
#255 完了Evidence ─┼─ repositoryRevision = R
#308 完了Evidence ─┘
                    ↓
             Gate B Ready for R
                    ↓
             Handover Baseline R
```

#214 actual external Oracle Evidenceは、Gate B Readyの必須収束対象ではありません。

ただしHandoverでは#214を消しません。次のいずれかの状態を明示します。

- **verified**: actual external Evidence / acceptance receipt / actual readが存在し、Handover Baseline Rへ追跡できる
- **unresolved / not verified**: #214 / #371 / #376が未完了であり、Oracle compatibility / actual `master_data` integrationを実証済みとは扱わない

外部観測資料そのものの`observedAt`がRの時点である必要はありません。#214をHandover前に完了させる場合は、その資料を現在authoringと照合したacceptance receiptとactual readをRへ結び直します。

Rを決めた時点で、#310 / #255 / #308の完了EvidenceがRと異なるrevisionにしかない場合は、Rで再実行してからReady判定します。「意味は変わっていないはずだ」と推測して、古い成功EvidenceをRの成功へ読み替えません。

## Handover Baselineに必要な情報

具体的なJSON schemaやfile formatはGate B通過時に決めても構いませんが、少なくとも次を復元できる必要があります。

### Repository / Reference identity

- Handover Baselineとなる単一repository revision
- Reference Workspace path / identity
- そのrevisionに保存されているgenerated artifact identity
- Gate B最終監査の対象revision
- #310 / #255 / #308の完了Evidenceが参照するrepository revision
- #214 actual external Oracleの検証状態（verified / unresolved）

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

actual Oracleがunresolvedでも、`Order -> Customer`のCanonical Relation / Contract Query invocation / Datasource meaningをHandoverから外しません。未検証なのはactual Oracle physical realizationであり、Canonical meaningではありません。

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

Customer referenceを含むDriver Scenarioがsynthetic / deterministicに再現できることと、actual Oracleで同じreferenceを実証したことを区別します。

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
- actual Oracleがunresolvedの場合、read-only principal / grant / driver条件をWeb側で補わない制約

### Gate B Evidence references

Gate Bを通過させたEvidenceへ戻れる必要があります。

Handover recordではEvidence本文を複製せず、少なくとも次の完了Evidenceを安定参照します。

#### #310 deterministic Repository

`tools/reference-gate-b-repository-evidence`の成功JSONを参照します。

少なくとも次へ戻れる必要があります。

- `repositoryRevision = Handover Baseline revision`
- `reproducibleRevision = true`
- Canonical Workspace Validation成功
- `final-ui-less-scenario`成功
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功

`external-evidence-intake`成功はactual Oracle read成功ではありません。外部事実を受け入れるfail-closed境界がRepository上で再現することを示します。

#### #255 generated artifact reproducibility

`tools/reference-generated-artifact-evidence`のcheck-only成功JSONを参照します。

少なくとも次を要求します。

- `repositoryRevision = Handover Baseline revision`
- `reproducibleRevision = true`
- World / Plugin Lock golden一致
- Durable Run golden一致

`-write`による修復操作そのものや、修復直後のdirty treeを完了Evidenceには使いません。

#### #308 real PostgreSQL engine

`tools/reference-postgres-header-detail-runtime`の成功JSONを参照します。

少なくとも次を、同じrunで確認したEvidenceへ戻れる必要があります。

- `repositoryRevision = Handover Baseline revision`
- `reproducibleRevision = true`
- `fixtureDatabaseSafety.guardPassed = true`
- `fixtureCleanupSafety.verified = true`
- 2件 / 0件 / Execution Error
- Semantic mapping / support column非露出

#308は実Provider engineを通るEvidenceとして必須に残し、Gate Bをsynthetic Evidenceだけへ縮退させません。

#### #214 actual external Oracle — verifiedの場合

Handover時点で#214が完了している場合は、次へ戻れるようにします。

- actual external Evidence artifactの参照
- `evidenceArtifactFingerprint`
- Handover Baseline revision上のsource-fixed acceptance receipt
  - `accepted = true`
  - `reproducibleRevision = true`
  - `repositoryRevision = Handover Baseline revision`
- #371で確定したactual Oracle driver / Host assembly
- #376で確認した実効read-only Evidence
- Oracle actual read Evidence
- actual readが、同じHandover Baseline revisionのruntime / realizationに対して行われたこと

acceptance receiptだけをactual read Evidenceへ格上げしません。

#### #214 actual external Oracle — unresolvedの場合

Handover時点で#214が未完了でもGate BをNot Readyには戻しません。その代わり、Handover recordへ少なくとも次を明示します。

- #214 actual external Oracle Evidence: `unresolved / not verified`
- #371 actual Oracle driver / Host assembly: 未確定または未検証
- #376 actual Oracle read-only Evidence: 未取得
- Oracle compatibility / actual `master_data` integrationを実証済みとは主張しない
- actual Oracle物理情報をWeb側で補完しない
- #310 synthetic Scenarioや#308 PostgreSQL EvidenceをOracle Evidenceへ読み替えない

このunresolvedはGate Bの失敗を隠すための例外ではありません。Gate B本来の目的とProvider固有external validationを分離した結果として、明示的に後続管理します。

#### Gate B最終監査

`GATE_B_READINESS_AUDIT.md`の最終Ready判定も、同じHandover Baseline revisionを対象とした判定として追跡します。

「#310はR1、#255はR2、#308はR3で成功した」という集合を、revisionを揃えないままR3のGate B Ready Evidenceとして扱いません。

#214がunresolvedの場合は、最終監査にもその状態とclaim制約を明示します。

### unresolved

Gate B通過を妨げないものの、Gate Cまたは後続段階で継続確認する事項を列挙します。

actual Oracle未完了でGate Bを通過する場合は、少なくとも次を必須unresolvedとして残します。

- #214 actual external Oracle Evidence
- #371 actual Oracle driver / Host assembly
- #376 actual Oracle read-only Evidence
- actual Oracle compatibilityを未検証とするclaim boundary

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
#310 / #255 / #308 completion Evidence for R
       ↓
#214 actual Oracle status: verified or unresolved
```

これにより、Web Adapterやbrowser automationのrevisionだけでなく、「どの意味baselineに対するConformanceか」と「どのexternal integrationが未検証か」を固定できます。

Gate C成功を#214 actual Oracle成功へ読み替えません。

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

Gate B完了Evidenceの意味または評価方法を変更する場合は、旧Evidenceを新しいbaselineへ黙って読み替えず、影響するEvidenceを再実行します。

#214がHandover後に完了しても、既存Handover Baselineを黙って`verified`へ書き換えません。actual Oracle完了がCanonical / Driver sourceへ変更を加えない場合は別Evidenceとして追跡でき、Gate C baselineの意味変更が必要な場合だけ新しいHandover Baselineを発行します。

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
3. #310 / #255 / #308の完了Evidenceを新しいbaseline revisionへ揃える
4. #214のverified / unresolved状態も再確認する
5. 新しいHandover Baselineを発行する
6. Gate Cで影響するScenarioを再実行する

## Gate B通過時に作るHandover record

Gate BがReadyになった時点で、Repositoryへ保存する必要があるかを判断します。

保存する場合も、最初から大きな汎用schemaにはしません。第一Referenceで必要な最小項目を持つmachine-readable artifact、またはMarkdown + stable referencesから開始できます。

重要なのは形式ではなく、次を満たすことです。

- 一つのbaseline revisionが固定される
- #310 / #255 / #308の完了Evidenceが同じbaseline revisionへ揃っている
- #214 actual Oracleのverified / unresolved状態が明示される
- Canonical IDへ戻れる
- Driver Scenarioへ戻れる
- Gate B Evidenceへ戻れる
- unresolvedを保持できる
- secretを含まない
- Target固有情報をCanonical baselineへ混ぜない

## Gate C開始条件

第一Web Targetの本格実装は、Gate B Readyに加えて次を満たしてから開始します。

- Handover Baselineのrepository revisionを固定できる
- #310 / #255 / #308の完了Evidenceが同じrepository revisionへ揃っている
- Gate C対象のCanonical identityを列挙できる
- Driver Scenario baselineを再現できる
- Gate B Evidenceへの参照が揃っている
- Gate B最終監査が同じrevisionをReadyと判定している
- #214 actual Oracleのverified / unresolved状態を明示できる
- unresolvedを明示できる

actual Oracleがunresolvedの場合、Gate C開始は許可できますが、Web側でOracle固有事実を補わず、Gate C ReadyをOracle compatibility Readyへ読み替えません。

これは新しいGateを追加するものではありません。Gate Bの出力をGate Cの入力へ曖昧なく接続するためのhandover確認です。

## 非目標

- 新しいCanonical Application schema
- Presentation IRの先行設計
- Web frameworkの選定
- Handover artifactからCanonicalへの逆生成
- Git commitだけをSemantic Conformanceの証明とみなすこと
- #310 / #255 / #308のEvidence成功revisionが異なるまま一つのbaselineへ読み替えること
- external Oracle未検証をactual Oracle成功として扱うこと
- #310 synthetic Evidenceや#308 PostgreSQL Evidenceを#214 actual Oracle Evidenceへ格上げすること
- Handover unresolvedを空にするためOracle物理情報を推測すること
- 外部観測資料の観測時刻をGit revision時刻へ無理に合わせること
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
- `issue/000035-gate-b-actual-oracle-blocker-reaudit.md`
- GitHub #214 / #291 / #381
