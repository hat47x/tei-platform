# Gate B → Gate C Handover Contract

## 目的

Gate Bを通過したApplication / Driverの意味baselineを、第一Web Target / Gate Cへ曖昧なく引き渡します。

Gate Cの比較対象を「最新Driver」にせず、Gate B Readyと判定した単一repository revision `R`をHandover Baseline `B`として固定します。

```text
Gate B Ready for R
        ↓
Handover Baseline B
  baselineRepositoryRevision = R
        ↓
Target build T
        ↓
Gate C Evidence
```

Handoverは新しいCanonical schemaや第二の正本ではありません。Git revision、既存Canonical identity、Driver Scenario、Gate B Evidence、unresolvedへの参照を束ねる検証上のcontractです。

Product Acceptance ScenarioはPAS-02 primary / PAS-03 secondaryです。

## 現行Gate B scope

Gate B Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃えます。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
same-run child Evidence artifacts
       ↓
#273 Gate B Readiness Audit for R
       ↓ Ready only
Handover Baseline B / R
```

actual Oracle #214 / #371 / #376は重要なactual external asset Evidenceとして継続しますが、第四Required laneではありません。未完了なら`unresolved`としてHandoverへ渡します。

## 基本原則

### Handoverは参照であり第二の正本ではない

```text
Authoritative Source / Canonical Application
                 │
                 ├─ Driver execution / Evidence
                 │
                 └─ Handover Baseline B
                       └─ revision / Canonical identity / Evidence reference
```

Handover recordへApplication meaningを複製し、そこからCanonicalへ逆書きしません。

### 一つのbaseline revisionへ揃える

異なるrevisionの成功Evidenceを寄せ集めて一つのReady revisionへ読み替えません。

- #310 `repositoryRevision = R`
- #255 `repositoryRevision = R`
- #308 `repositoryRevision = R`
- 全lane `reproducibleRevision = true`
- #273が同じRをReadyと判定

source変更が必要になった場合、旧Rの成功を新revisionへcarry-forwardせず、新しいRで必要なEvidenceを取り直します。

### same-Rだけでなくsame-run provenanceを要求する

Handoverへ使う#310 / #255 / #308 detailed Evidenceは、単に同じRで成功した任意の三つのJSONではありません。

**#387 final convergenceが実際に同一runで読み、保持したexact child Evidence artifacts**をRequired Evidenceとして参照します。

```text
#387 final summary
  repositoryRevision = R
  allLanesSameRevision = true
  childArtifactsRetained = true
        │
        ├─ deterministic-repository.json
        ├─ generated-artifact-reproducibility.json
        └─ real-postgres-ephemeral-fixture.json
              ↓
        #273 Ready Audit for R
              ↓
        Handover B / R
```

同じRで後から再実行した別artifactを、#387が実際に読んだartifactと同一だと推測しません。

### Target側の不足をbaselineへ逆輸入しない

Web実装中に必要なApplication meaningがB / Rに存在しない場合、Web固有fieldや独自ruleで補ってGate Cを通しません。

Application / Driverへ戻り、authoritative sourceを変更し、必要なGate B Evidenceを新revisionで再確認してsuccessor Handoverを発行します。

## Handover Baseline Bに必要な情報

具体的JSON schemaやfile formatはGate B通過前に固定しません。ただし、少なくとも次を復元できる必要があります。

### Baseline identity

- Handover Baseline identity `B`
- `baselineRepositoryRevision = R`
- Reference Workspace identity
- Gate B Readiness Audit対象revision

### Canonical Application baseline

第一Referenceでは少なくとも次のidentityへ戻れるようにします。

- Application `order_entry`
- Application Pattern
  - browse
  - record
  - reference
  - header-detail
  - command
- Data / Relation / Query
- Datasource
- Validation
- Authentication / Authorization
- Capability / Binding
- Business Result / named Outcome
- Execution Error
- Observationに使う明示read境界
- Source / Binding traceability

Handoverの列挙はmanifestであり、意味本文の複製ではありません。

### Driver Scenario baseline

少なくとも次のScenarioへ戻れるようにします。

```text
Authentication
  -> browse
  -> record
  -> reference Customer
  -> header-detail OrderLine[]
  -> Canonical Validation
  -> Authorization
  -> Command Business Result
  -> separate Observation read
```

Business Result、named Outcome、Execution Error、Observation、Diagnosticの区別をTarget側へ引き渡します。

### Datasource / authority boundary

少なくとも次を保持します。

- read / read-write
- external-observed / managed-here等、Evidence上必要なownership性質
- Database instance / executor Capability Binding
- secret / credential / runtime parameterを通常Evidenceへ保存しない制約

## Gate B Evidence provenance

Handover recordへEvidence本文を複製しません。stable reference / identity / fingerprintから実artifactへ戻れるようにします。

### #387 final convergence

Handoverから、Ready判定へ利用した#387 final convergence artifactへ戻れる必要があります。

少なくとも:

- `evidenceNature = gate-b-three-lane-final-convergence`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- `allLanesSameRevision = true`
- `childArtifactsRetained = true`
- final source check成功
- Go environment安全条件成立

を確認済みのartifactを参照します。

### same-run child exact artifacts

#387が保持した三つのchild artifactをRequired Evidenceとして参照します。

- `deterministic-repository.json`
- `generated-artifact-reproducibility.json`
- `real-postgres-ephemeral-fixture.json`

各artifactについて少なくとも:

- artifact stable reference
- fixed filename / lane identity
- `evidenceArtifactFingerprint`
- `repositoryRevision = R`
- `reproducibleRevision = true`

へ戻れる必要があります。

#387 runnerはfinal success直前にも保存fileを再読込してfingerprintを確認しますが、#273 Readiness Auditでも保存artifactを独立に再照合します。runner内検証だけでReadyを宣言しません。

### #310 deterministic Repository

same-run `deterministic-repository.json`から少なくとも次へ戻ります。

- Workspace Validation成功
- `final-ui-less-scenario`成功
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功

synthetic Evidenceをactual external Evidenceへ格上げしません。

### #255 generated artifact reproducibility

same-run `generated-artifact-reproducibility.json`から少なくとも次へ戻ります。

- World / Plugin Lock golden一致
- Durable Run golden一致

`-write`修復操作や修復直後のdirty treeを完了Evidenceにしません。

### #308 real PostgreSQL Provider

same-run `real-postgres-ephemeral-fixture.json`から少なくとも次へ戻ります。

- fixture前安全guard
- 2件 / 正常0件 / missing-table Execution Error
- explicit Semantic mapping
- support column非露出
- DB-side ownership-protected cleanup
- cleanup後user relation 0 / Event Trigger 0 / fixture object不在

production既存Database Evidenceへ読み替えません。

### #273 Gate B Readiness Audit

#273 / `GATE_B_READINESS_AUDIT.md`が同じRについて、#387 final summaryとsame-run child exact artifactsを監査し、Readyと判定したことを参照します。

final runner exit codeだけをReady判定にはしません。

## actual Oracleの扱い

#214 / #371 / #376はRequired Evidence集合へ加えません。

取得済みでB / Rに対するverified external asset Evidenceとして十分なら追加参照します。少なくとも:

- actual observed artifact
- evidence fingerprint
- source-fixed acceptance receipt
- actual read Evidence
- runtime / realization / repository revisionとの関係

を区別します。

未取得、actual read未完了、Rへの結び付き不足なら`unresolved`として残します。

第一Web Target側でOracle physical object / column / SQL / datatype / driver behavior / read-only方式を推測・補完しません。

## Gate C Evidenceからのtrace

Gate C Evidenceでは意味baselineとTarget build identityを分けます。

```text
Gate C Evidence
  ├─ Handover Baseline B
  │    └─ baselineRepositoryRevision = R
  │         └─ #273 Ready Audit
  │              └─ #387 final convergence
  │                   └─ same-run #310 / #255 / #308 exact artifacts
  │
  └─ Target build identity T
```

通常`T != R`です。

Gate C Evidenceから少なくともB / R / Tへ戻れる必要があります。

## baseline変更時

### Target実装だけの変更

CSS、DOM、Web Adapter、browser automation等だけが変わり、Canonical / Driver meaningが変わらない場合はB / Rを維持できます。

### Evidence / Diagnosticだけの変更

評価方法やDiagnostic変更がGate B completion Evidenceの意味へ影響する場合は、影響するRequired Evidenceを再実行します。旧Evidenceを新baselineへ黙って読み替えません。

### Application / Driver meaning変更

Validation、Relation / Query / Datasource、Capability / Binding、Business Result / Outcome、Observation read境界、Pattern intent等を変える場合:

1. authoritative sourceを変更
2. Driver / Validationを再確認
3. 必要なGate B Evidenceを新revision `R2`で取得
4. #387を`R2`で再実行しsame-run child artifactsを取得
5. #273で`R2`を再監査
6. Readyならsuccessor Handover `B2 / R2`を発行
7. Gate Cの影響Scenarioを再実行

旧B / Rへ意味変更を黙って上書きしません。

## Handover record

Gate B Ready後に第一Handover recordを作成します。

最初から大きな汎用schemaへしません。Markdown + stable referencesでも、必要性が確認できれば小さいmachine-readable artifactでも構いません。

最低条件:

- `B / R`を固定
- Canonical identityへ戻れる
- Driver Scenarioへ戻れる
- #387 final convergenceへ戻れる
- #387 same-run #310 / #255 / #308 exact artifactへ戻れる
- child fingerprintを再照合できる
- #273 Ready Auditへ戻れる
- optional actual external Evidenceと`unresolved`をRequired Evidenceから分離
- secretを含まない
- Target固有情報をCanonical baselineへ混ぜない

## Gate C開始条件

第一Web Targetの本格実装は、少なくとも次を満たしてから開始します。

- Gate B Ready
- Handover Baseline `B / R`を固定
- #310 / #255 / #308 same-run Required EvidenceがRへ揃っている
- #387 final convergenceとsame-run child artifactsへ追跡できる
- #273が同じRをReadyと判定
- Gate C対象Canonical identityを列挙できる
- Driver Scenario baselineを再現できる
- actual Oracle等のexternal Evidenceをverified / unresolvedとして明示できる
- unresolvedを明示できる

Handover確認は新しい番号付きGateではありません。Gate Bの出力をGate Cの入力へ固定するcontractです。

## 非目標

- 新しいCanonical Application schema
- Handover machine-readable schemaの先行固定
- Presentation IRの先行設計
- Web frameworkの選定
- Handover artifactからCanonicalへの逆生成
- Git commitだけをSemantic Conformanceの証明とすること
- same-R別runを#387 same-run Evidenceへ読み替えること
- actual Oracleを第四Required laneへ戻すこと
- external-observed未観測事実をsynthetic / PostgreSQL Evidenceから推測すること
- 第一TargetだけでTarget-neutral taxonomyを完成させること

## 関連

- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_EVIDENCE_ACQUISITION.md`
- `GATE_B_EVIDENCE_RUNBOOK.md`
- `GATE_C_EVIDENCE_CONTRACT.md`
- `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `V1_REFERENCE_APPLICATION_PLAN.md`
- `issue/000007-first-web-target-projection.md`
- `issue/000028-first-web-target-conformance-evidence.md`
- `issue/000029-gate-b-readiness-audit.md`
- `issue/000030-gate-b-to-c-handover-baseline.md`
- `issue/000035-gate-b-external-provider-scope-audit.md`
- `issue/000037-gate-b-three-lane-final-evidence.md`
