# Gate B → Gate C Handover Contract

## 目的

判定段階Bを通過したアプリケーションとドライバーの意味の基準版を、最初のウェブ実行先と判定段階Cへ、曖昧さを残さず引き継ぎます。

判定段階Cでは、「最新のドライバー」を比較対象にはしません。判定段階Bで合格とした特定のリポジトリの版`R`を、引継ぎ基準版`B`として固定します。

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

引継ぎは、新しい正本のスキーマでも第二の正本でもありません。Gitの版、既存の正本上の識別情報、ドライバーのシナリオ、判定段階Bの検証根拠、未解決事項への参照をまとめる、検証上の契約です。

製品の受入シナリオでは、PAS-02を主な対象、PAS-03を副次的な対象とします。

## 現行Gate B scope

判定段階Bで必須となる検証根拠は、次の三つです。

1. #310 リポジトリの決定論的な検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験用環境の検証根拠

三つとも、変更のない同じ候補版`R`を対象に取得します。

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

実際のOracleに関する#214・#371・#376は、外部資産の検証根拠として引き続き重要です。ただし、第四の必須検証経路にはしません。未完了であれば`unresolved`として引き継ぎます。

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

引継ぎ記録へアプリケーションの意味を複製したり、その記録から正本へ書き戻したりしません。

### 一つのbaseline revisionへ揃える

異なる版で得た成功の検証根拠を寄せ集め、単一の合格版での結果として読み替えません。

- #310 `repositoryRevision = R`
- #255 `repositoryRevision = R`
- #308 `repositoryRevision = R`
- 全lane `reproducibleRevision = true`
- #273が同じRをReadyと判定

ソースの変更が必要になった場合、旧版Rの成功を新しい版へそのまま引き継ぎません。新しい版で必要な検証根拠を取り直します。

### same-Rだけでなくsame-run provenanceを要求する

引継ぎに使用する#310・#255・#308の詳細な検証根拠は、同じ版Rで成功した任意の三つのJSONではありません。

**#387の最終確認が同じ実行で実際に読み取り、保存した子処理の検証成果物そのもの**を、必須の検証根拠として参照します。

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

同じ版Rで後から再実行した別の成果物を、#387が実際に読み取った成果物と同一だと推測しません。

### Target側の不足をbaselineへ逆輸入しない

ウェブの実装中に必要となったアプリケーションの意味がB・Rにない場合、ウェブ固有の項目や独自の規則で補って判定段階Cを通過させることはしません。

アプリケーションやドライバーへ戻り、正本となる入力元を変更します。そのうえで、新しい版について必要な判定段階Bの検証根拠を再確認し、後継の引継ぎ基準版を発行します。

## Handover Baseline Bに必要な情報

具体的なJSONスキーマやファイル形式は、判定段階Bの通過前には固定しません。ただし、少なくとも次の情報を復元できる必要があります。

### Baseline identity

- Handover Baseline identity `B`
- `baselineRepositoryRevision = R`
- Reference Workspace identity
- Gate B Readiness Audit対象revision

### Canonical Application baseline

最初の参照事例では、少なくとも次の識別情報へたどれるようにします。

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

引継ぎで列挙するのは参照先の一覧であり、意味の本文を複製するものではありません。

### Driver Scenario baseline

少なくとも次のシナリオへたどれるようにします。

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

業務上の結果、名前付きの結果、実行時エラー、観測、診断結果の違いを、実行先へ引き継ぎます。

### Datasource / authority boundary

少なくとも次を保持します。

- read / read-write
- external-observed / managed-here等、Evidence上必要なownership性質
- Database instance / executor Capability Binding
- secret / credential / runtime parameterを通常Evidenceへ保存しない制約

## Gate B Evidence provenance

引継ぎ記録へ検証根拠の本文を複製しません。安定した参照先・識別情報・照合値から、実際の成果物へたどれるようにします。

### #387 final convergence

引継ぎ記録から、合格判定に使用した#387の最終確認の成果物へたどれる必要があります。

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

#387が保存した三つの子処理の成果物を、必須の検証根拠として参照します。

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

#387の実行器は最終的な成功判定の直前にも保存ファイルを読み直し、照合値を確認します。さらに#273の合格監査でも保存成果物を独立して照合します。実行器内の確認だけでは合格を宣言しません。

### #310 deterministic Repository

same-run `deterministic-repository.json`から少なくとも次へ戻ります。

- Workspace Validation成功
- `final-ui-less-scenario`成功
- `external-evidence-intake`成功
- `repository-evidence-source-boundary`成功

合成データから得た検証根拠を、実際の外部資産で得たものとして格上げしません。

### #255 generated artifact reproducibility

same-run `generated-artifact-reproducibility.json`から少なくとも次へ戻ります。

- World / Plugin Lock golden一致
- Durable Run golden一致

`-write`による修復操作や、修復直後に変更の残る作業ツリーを、完了の検証根拠にはしません。

### #308 real PostgreSQL Provider

same-run `real-postgres-ephemeral-fixture.json`から少なくとも次へ戻ります。

- fixture前安全guard
- 2件 / 正常0件 / missing-table Execution Error
- explicit Semantic mapping
- support column非露出
- DB-side ownership-protected cleanup
- cleanup後user relation 0 / Event Trigger 0 / fixture object不在

本番の既存データベースに関する検証根拠とは読み替えません。

### #273 Gate B Readiness Audit

#273と`GATE_B_READINESS_AUDIT.md`によって、同じ版Rについて#387の最終結果と同一実行内の子処理の成果物を監査し、合格と判定したことを参照します。

最終確認の実行器の終了コードだけでは合格と判定しません。

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
- ISSUE-000007-first-web-target-projection.md
- ISSUE-000028-first-web-target-conformance-evidence.md
- ISSUE-000029-gate-b-readiness-audit.md
- ISSUE-000030-gate-b-to-c-handover-baseline.md
- ISSUE-000035-gate-b-external-provider-scope-audit.md
- ISSUE-000037-gate-b-three-lane-final-evidence.md
