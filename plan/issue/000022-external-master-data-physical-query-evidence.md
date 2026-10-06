# ISSUE-000022: external-observed `master_data`の物理Query Evidenceを確定する

- 状態: Blocked — actual Oracle実環境情報待ち / Gate Bではunresolved可
- GitHub Issue: #214
- 親Issue: `ISSUE-000018` / GitHub Issue #179
- 関連Issue: `ISSUE-000020` / `ISSUE-000021` / `ISSUE-000033` / `ISSUE-000034` / `ISSUE-000035`
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-004 / RQ-005 / RQ-007 / RQ-009 / RQ-012 / RQ-014

## 現行scopeと分担

[ISSUE-000035](000035-gate-b-external-provider-scope-audit.md)のscope判断に従い、actual OracleはGate B必須blockerから外し、既存外部資産・第二Providerの継続検証として扱います。本Issueはphysical facts、sourceとの照合、actual readへの統合を担当し、driver / Host assemblyは[ISSUE-000033](000033-gate-b-oracle-driver-integration.md)、実効read-onlyの確認は[ISSUE-000034](000034-gate-b-oracle-read-only-evidence.md)へ集約します。

未取得の実Oracle Evidenceを完了扱いせず、[ISSUE-000030](000030-gate-b-to-c-handover-baseline.md)のunresolvedへ残します。以下に残るGate Bの4必須laneやcandidateへのOracle必須収束は過去の契約であり、現行scopeと衝突する場合は本節と[Gate B監査](../GATE_B_READINESS_AUDIT.md)を優先します。

## Provider-neutralな位置づけ

本IssueとISSUE-000033 / 000034は、題名と記述がOracleに寄っていますが、満たすべき要求はOracle固有ではありません。RQ-005 / RQ-007 / RQ-012が求めるのは、PostgreSQL以外の実Database Providerでも、物理事実の観測、read-only境界、driver assembly（ADR-0007）、actual readという同じ契約でprovider-neutral境界を検証できることです。

第一Referenceの`master_data`が既存のOracle資産であるため、最初の具体例がOracleになっています。別のProviderを加える場合も、このEvidence契約を再利用し、Oracle固有の事実（`READ` object privilege、`SET TRANSACTION READ ONLY`、driver候補等）をprovider-neutralなCore契約へ昇格しません。

## Evidenceの目的

第一Referenceの`master_data`について、TEIが所有していない既存資産の実物理事実、provenance、read-only境界、Oracle driverを通したactual readを確認し、`Order -> Customer` Referenceが実環境でもSemantic meaningを保って動作することをEvidenceとして確定します。

`master_data`は`external-observed` Databaseです。`development`では`master_data.dev`としてBindingされ、providerはOracleです。

このIssueでは、未観測の物理SchemaをTEI側から生成・推測しません。`CUSTOMER`、`CUSTOMER_ID`等を命名慣習から仮置きすると、既存資産受入れのReferenceがTEI都合の架空Schemaへ変質するためです。

## Gate Bとの関係

PR #379を起点とした`ISSUE-000035` / #381の再監査により、**本Issueのactual Oracle EvidenceはGate Bの必須完了条件ではなくなりました。**

これはactual Oracle Evidenceが不要になったという意味ではありません。

本Issueは次を主張するための継続actual external asset Evidenceです。

- actual Oracle compatibility
- actual `master_data` integration
- actual Oracle driver / Host assembly
- actual read-only principal / grant / transaction境界
- actual read → Semantic Customer

Gate B判定時点で未完了の場合は、#214 / #371 / #376をHandover Baselineへ`unresolved / not verified`として明示します。

その状態で第一Web Targetへ進んでも、次は主張しません。

- Oracle Providerがactual環境で確認済み
- `master_data` actual assetが接続済み
- Oracle datatype / NULL / named bind挙動が確認済み
- Oracle read-only principal / grantが確認済み

Web側で未観測Oracle物理情報やread-only方式を補完しません。

## 現在の判定

**external Evidenceを受け入れる非Canonical境界はReadyですが、第一Referenceのactual physical facts / read-only / Oracle actual readは未取得です。**

したがって本Issue自体はBlockedのままです。ただし、このBlocked状態だけでGate BをNot Readyにはしません。

PR #340以降、Evidence受入れの成功は単なる人向けメッセージではなく、**どのexternal Evidence artifactを、どのclean Repository revisionで照合したか**を示す機械可読receiptとして残します。

PR #364 / #366以降、Oracle Runnerはactual read時にdriverが実際に返したraw result列名と、driverが提供できた範囲の列metadataをOracle固有Payloadへ保持できます。ただし、これはactual physical factsを取得済みという意味ではありません。

PR #372で具体Oracle driver / Host assemblyの確定作業を`ISSUE-000033` / #371へ、PR #378でactual read-only Evidenceの確定作業を`ISSUE-000034` / #376へ分離しました。

## 既に成立している実行境界

物理Schemaの具体名が未観測でも、Semantic meaningと物理実行を混ぜないための境界は成立しています。

```text
Semantic Relation
  Order -> Customer
        ↓ explicit key
Reference Contract Query invocation
        ↓
Database Contract Query
  master_data.customer / find_by_id
        ↓ explicit realization
ContractQuerySQLRealization
  ├─ Database / Contract / Query target
  ├─ provider
  ├─ Contract input -> physical parameter
  ├─ native source
  └─ raw result column -> Semantic attribute
        ↓
Environment / Database instance / executor Binding
        ↓
instance-aware runtime Registry
        ↓
Oracle database/sql Runner
        ↓ raw columns / driver-reported column metadata
        ↓ explicit result mapping
ContractQueryResult
  └─ Semantic Customer row / not-found / Execution Error
```

成立済みの主な境界:

- Relation keyからContract Query inputを準備する
- Contract Query signatureを型付きで表現・Validationする
- Reference invocationを明示する
- Relation / invocation / Query signatureを横断Validationする
- Database / Contract / Queryをrealization target identityとする
- Contract input → physical parameterを明示mappingする
- raw result column → Semantic attributeを明示mappingする
- SQL realizationを通常Workspaceから読み込む
- SQL realization authoring / native sourceをSource Indexで追跡する
- Environment / Database instance / executor Capabilityを明示Bindingする
- runtime RegistryでDatabase instance identityを保持する
- 同一Capabilityを共有する別Database instanceへfallbackしない
- Oracle向け最小`database/sql` Runnerで明示native sourceとnamed parameterをraw executionへ渡す
- actual read時にdriverが返したraw result列名をOracle固有Payloadへ保持する
- driverが提供できた`databaseTypeName` / `nullable` / `length` / `precision` / `scale`を、未提供値を推測せずOracle固有Payloadへ保持する
- 0 rowとExecution Errorを区別する
- SQL realizationがないWorkspaceでは物理Runnerを呼ばず停止する

これらはactual external factsの代用品ではありません。

## external Evidence受入境界

`internal/database/externalqueryevidence`は、external EvidenceをCanonical Worldへ昇格させず、明示authoring済み`ContractQuerySQLRealization`との照合にだけ利用します。

Evidenceが保持する主な観測情報:

- Contract Query target identity
- observed provider
- physical objectのkind / identity
- Contract inputとphysical parameterの観測対応
- raw result列とSemantic attributeの観測対応
- physical type / nullable等の補助観測情報
- native source Evidence参照
- authoring済みnative source本文とのfingerprint
- physical factsのsource / provenance
- Reference用Evidenceへの加工方法
- 実効的read-only mechanism / Evidence参照

### strict loader

Evidence JSONは未知fieldや後続JSON値を黙って無視しません。

formatが不正なら停止し、その後にWorkspace / SQL realizationとの意味整合を検査します。

### Workspace Validation

Evidence受入れ前に通常WorkspaceをLoadし、`validation.Validate`のErrorが一つでもあれば停止します。

Evidenceが正しくても、無効なCanonical authoringを前提に受理しません。

### SQL realizationをEvidenceから作らない

`tools/reference-external-query-evidence`はEvidenceのtarget + providerに完全一致するSQL realizationを一意に解決します。

該当realizationが存在しなければ停止し、Evidenceに書かれたtable名、column名、parameter名、SQL等から不足authoringを生成・補完しません。

このintake境界のConformanceは#310 deterministic Repository Evidenceの必須checkとしてGate Bに残ります。

## source-fixed acceptance receipt

PR #340で、受入れrun自体を一つのclean Repository revisionへ固定し、実際に検査したEvidence artifactをfingerprintで識別する境界を追加しました。

### 実行開始前

external Evidenceを読む前に次を確認します。

```text
git rev-parse HEAD
git status --porcelain
```

working treeがdirtyなら受入れを停止します。

Repository状態の観測には`tools/internal/repositoryevidence`を利用します。

PR #355以降、`postRunRepository.checked=true`になるのは、実行後repository revisionとworking tree状態の両方を観測できた後だけです。

### Evidence artifact fingerprint

Evidence fileをexact bytesで読み、SHA-256を計算します。

```text
evidenceArtifactFingerprint = sha256:<hex>
```

これは、**toolが実際に読み込んだexternal Evidence JSON artifactそのもの**を識別します。

### 受入れ照合

次を維持したままEvidenceを照合します。

- strict JSON load
- Workspace Validation
- target / provider一致
- `external-observed` ownership
- Contract input → physical parameter一致
- raw result → Semantic attribute一致
- native source fingerprint一致
- provenance
- Reference用加工方法
- read-only mechanism / Evidence参照

### 実行終了後

照合後にHEAD / working treeを再確認します。

- 実行中にHEADが変化した
- 実行後にtracked / untracked差分が残った
- post-run revision文字列が開始revisionと一致しない

のいずれかなら成功receiptを出しません。

### receipt

成功時はstdoutへ小さいJSON receiptを出します。

```json
{
  "evidenceNature": "external-query-evidence-acceptance",
  "repositoryRevision": "<git commit>",
  "workingTreeClean": true,
  "reproducibleRevision": true,
  "postRunRepository": {
    "checked": true,
    "repositoryRevision": "<same git commit>",
    "revisionStable": true,
    "workingTreeClean": true
  },
  "evidenceArtifactFingerprint": "sha256:<hex>",
  "target": "<Database / Contract / Query identity>",
  "provider": "oracle",
  "accepted": true
}
```

Gate B用にreceiptを保存する場合、Repository内へstdoutを直接redirectしません。

## 二つのfingerprintを分離する

### `evidenceArtifactFingerprint`

external Evidence JSON exact bytesを識別します。

### `nativeSource.fingerprint`

Evidenceが観測したnative sourceと、現在authoringされている`ContractQuerySQLRealization.Source`が一致するかを確認します。

両者は相互代替しません。また、fingerprint一致だけでEvidence記載内容の真実性や独立性を自動証明しません。

実根拠は少なくとも次で別途追跡します。

- `nativeSource.evidenceRef`
- `provenance.sourceRef`
- read-only mechanism / Evidence参照
- 実接続・権限・driver Evidence

## 受入境界がしないこと

- external Evidenceを`ir.World`へ取り込まない
- EvidenceからSQL realizationを生成しない
- SQL本文からphysical parameterやSemantic attributeを推測しない
- physical object名からRelationや業務意味を推測しない
- `SELECT`に見える文字列だけをread-only保証とみなさない
- physical typeをCanonical Semantic型へ自動変換しない
- external Evidence / fixture / adapterを第二のSemantic SSoTにしない
- acceptance receiptをactual Oracle read Evidenceへ格上げしない
- receiptへEvidence本文やsecretを複製しない

## receiptへ保存しない情報

- physical object identity
- physical parameter / raw result columns
- physical type / NULL情報
- provenance本文
- read-only mechanism本文
- Evidence file path
- credential / secret

actual physical factsそのものは、元のexternal Evidence artifactと、そのprovenance / Evidence参照で追跡します。

## まだ必要なactual physical Evidence

### physical object

- 実際に参照するobject kind
- object identity
- Referenceで利用してよい範囲

### input

- Contract input `id`に対応する実physical parameter
- parameterの実物理型
- NULL / driver変換上の扱い

### result

- 実際に返るraw columns
- raw columnsの型
- NULLの扱い
- 第一Referenceで必要なSemantic attributeとの対応

外部Schema全体を取り込む必要はありません。

### native realization source

Oracle SQLを利用する場合は、実際に利用するnative SQL sourceをEvidenceとして確認します。

既存SQLを匿名化・縮約してReferenceへ持ち込む場合は、元EvidenceとReference用sourceの関係をprovenanceとして残します。

### provenance

少なくとも次を追跡します。

- 何を観測したか
- どのsourceから確認したか
- 直接利用か、匿名化・縮約等の加工をしたか
- どこまでが観測事実で、どこからがReference用加工か

## read-onlyの実効保証

`find_by_id`がQueryであることやnative SQLが`SELECT`に見えることだけではread-only保証としません。

read-only Evidenceの取得・判定は`ISSUE-000034` / #376で具体化します。

第一Referenceでは、実Oracle versionと権限構成に応じて、少なくとも次を分離して確認します。

- 専用のleast-privilege principal / roleと対象objectへのquery privilege
- system / object privilege、role、必要に応じてPUBLIC等を再確認できるgrant Evidence
- 採用する場合の`SET TRANSACTION READ ONLY`等のtransaction境界
- write用`Exec`をReference read pathへ露出させないquery-only runtime境界
- actual Queryが上記境界の中で成功したこと

principal / grant Evidenceを主たるread-only根拠とし、transactionやTEI側のquery-only境界を追加防御として扱います。

read-only確認のためだけに、実外部assetへ`INSERT` / `UPDATE` / `DELETE` / `MERGE` / DDL、`SELECT ... FOR UPDATE`、exclusive `LOCK TABLE`等を試行して失敗確認する方法は採りません。

Evidenceにはmechanism名だけでなく、その事実を再確認できるEvidence参照を要求します。credential / passwordは保存しません。

## Oracle driver integration

特定Oracle driverに依存しない`database/sql` Runner境界までは成立しています。

具体driverの選択・登録、DSN / credential、TLS / TCPS、Wallet等のHost / Reference assemblyは`ISSUE-000033` / #371で確定します。

PR #364 / #366以降、actual read時のdriver観測をSemantic mapping後に失わないため、Oracle Runner固有Payloadへ次を保持できます。

- 実際に`database/sql`へ渡したnamed parameterの名前
- Databaseから実際に返されたraw result列名
- driverが提供できた範囲の`databaseTypeName`
- driverが提供できた場合だけの`nullable`
- driverが提供できた場合だけの`length`
- driverが提供できた場合だけの`precision` / `scale`

`database/sql.ColumnType`が情報を提供できない場合、falseや0を推測せず未観測として保持します。Go側の`ScanType`やraw row値も通常Evidenceへ保存しません。

このruntime metadataは、provenance付きexternal Evidenceに記録した`PhysicalType` / `Nullable`を置き換える第二の物理正本ではありません。

actual physical Evidence取得後、採用driverについて少なくとも次を確認します。

- named parameterが期待どおりに渡る
- 対象native sourceが実行できる
- input / resultの型変換
- NULLの扱い
- raw column名
- driver-reported raw column metadata
- Query Error / row取得Errorの伝播
- read-only強制との整合

特定driverの都合に合わせてCanonical Contractを変更しません。

## actual Evidence取得後の作業順

```text
1. 実physical object / parameter / raw columns / native sourceを観測
2. provenanceを確認
3. ISSUE-000034 / #376でread-only grant / transaction / query-only runtime境界を確認
4. ISSUE-000033 / #371でactual Oracle driver / Host assemblyを確定
5. external Evidence JSONを作成
6. 観測事実を根拠にContractQuerySQLRealizationを明示authoring
7. clean revisionでreference-external-query-evidenceを実行
8. source-fixed acceptance receiptを取得
9. read-onlyを強制したOracle接続境界を用意
10. 採用Oracle driverでparameter / result / NULL / error伝播を確認
11. development / master_data.dev / database.query.oracleからactual read
12. actual readでraw result列名 / driver-reported metadataを観測
13. raw resultを明示mappingしてSemantic Customerを得る
14. Workspace -> actual read -> Semantic resultを一続きに確認
15. actual readが参照したrepository source identityをEvidenceへ残す
16. #273へactual external asset Evidenceとして対応付ける
```

acceptance receiptとactual read Evidenceは別工程です。

## Gate B / Handover Baselineとの関係

### Gate B判定前に完了した場合

#214をGate B判定前に完了させる場合は、candidate Rのauthoring / runtimeへacceptance receiptとactual readを結び付け、Handoverへ`verified`として含めます。

### Gate B判定時点で未完了の場合

Gate Bを自動的にNot Readyへ戻しません。

Handoverへ次を明示します。

- #214 actual external Oracle Evidence = unresolved / not verified
- #371 actual Oracle driver / Host assembly = 未確定または未検証
- #376 actual Oracle read-only Evidence = 未取得
- Oracle compatibility / actual `master_data` integrationを実証済みとは主張しない
- Web側でOracle固有事実を補わない

### Gate B通過後に完了する場合

既存Handover Baselineを黙って`verified`へ書き換えません。

actual Oracle作業がCanonical / Driver sourceへ意味上の変更を加えず、Gate C baselineへ影響しない場合は、別Evidenceとして継続追跡できます。

Gate Cへ取り込むApplication / Driver meaningやsource-dependent境界が変わる場合は、影響するGate B Evidenceを再確認し、新しいHandover Baselineを発行します。

## 完了条件

### Evidence受入境界

- [x] external EvidenceをCanonical Worldから分離して表せる
- [x] Evidence JSONを未知fieldを黙認せず読み込める
- [x] target / providerをSQL realizationと照合できる
- [x] `external-observed` ownershipを検査できる
- [x] Contract input → physical parameterをEvidenceとauthoringで照合できる
- [x] raw result → Semantic attributeをEvidenceとauthoringで照合できる
- [x] native source fingerprintのstaleを検知できる
- [x] physical facts provenance欠如を拒否できる
- [x] read-only mechanism / Evidence参照欠如を拒否できる
- [x] SQL realizationがない場合にEvidenceから補完せず停止できる
- [x] Evidence受入れ前にWorkspace Validationを行う
- [x] 受入れ開始前にclean Repository revisionを要求する（PR #340）
- [x] Evidence JSON exact bytesのSHA-256を計算する（PR #340）
- [x] 受入れ後にHEAD / working treeを再確認する（PR #340）
- [x] `reproducibleRevision = true`を含むacceptance receiptを出力できる（PR #340）
- [x] receiptへEvidence本文 / path / credential / secretを複製しない（PR #340）
- [x] `evidenceArtifactFingerprint`と`nativeSource.fingerprint`を別責務として扱う（PR #340）
- [x] post-run観測が完了した場合だけ`postRunRepository.checked = true`にする（PR #355）
- [x] actual Oracle driver integrationを`ISSUE-000033` / #371へ分離した（PR #372）
- [x] actual Oracle read-only Evidenceを`ISSUE-000034` / #376へ分離した（PR #378）
- [x] actual OracleをGate B必須blockerから継続Evidenceへ再分類した（ISSUE-000035 / #381）

### 第一Referenceのactual Evidence

- [ ] 第一Referenceで参照する実physical objectが明示されている
- [ ] Contract input `id`から実physical parameterへの対応と型 / NULL扱いが明示されている
- [ ] 実raw result columnsと型 / NULL扱いが明示されている
- [ ] raw resultから必要なSemantic attributeへの対応が明示されている
- [ ] 実native SQL / API / adapter等のrealization sourceを追跡できる
- [ ] 物理情報のsource / provenanceを追跡できる
- [ ] actual Evidence artifactをclean sourceで受理し`reproducibleRevision = true`のreceiptを取得できる
- [ ] `ISSUE-000034` / #376のEvidenceに基づき`master_data`を実効的にread-onlyのまま実行できる
- [ ] `ISSUE-000033` / #371で確定した採用Oracle driverとのintegration Evidenceがある
- [ ] actual readが参照したRepository sourceを追跡できる
- [ ] actual readでraw result列名 / driver-reported metadataを観測できる
- [ ] actual readの結果分類とSemantic Customerへの明示mappingを追跡できる
- [ ] actual read Evidenceへcredential / secret / 不要なparameter value / 推測metadataを保存していない
- [ ] 第一ReferenceのWorkspace → actual read → Semantic resultを一続きに確認できる
- [ ] physical fixture / adapter / EvidenceがSemantic meaningの正本として扱われていない
- [ ] 結果を`ISSUE-000029` / #273へ対応付ける

### 既存runtime

- [x] 0件を正常なnot-foundとして扱える
- [x] Execution Errorと0件を区別できる
- [x] SQL realization sourceをSource Indexで追跡できる
- [x] runtime RegistryがDatabase instance identityを保持する
- [x] Oracle向け`database/sql` Runner境界がある
- [x] actual read時のraw result列名をOracle Runner固有Payloadへ保持できる（PR #364）
- [x] driverが提供できたraw列metadataを未提供値の推測なしで保持できる（PR #366）

## 非目標

- 外部Database全体をTEI管理下へ移すこと
- 不明な物理SchemaをTEIが生成・推測すること
- external EvidenceをCanonicalへ昇格すること
- acceptance receiptをactual Oracle read Evidenceとして扱うこと
- actual Oracle driver未確定の段階で汎用Evidence schemaを先行固定すること
- driverが提供していないraw列metadataをfalse / 0 / 慣習的Oracle型から推測すること
- 汎用ORMの導入
- Oracle固有の高度なHA / Connection Poolを本Issueだけで完成させること
- synthetic Conformance SQLを第一Referenceの実native sourceへ流用すること
- SQL文字列だけでread-onlyを推論すること
- read-only確認のため実外部assetへDML / DDL / `SELECT ... FOR UPDATE` / exclusive lockを試行すること
- #310 synthetic Evidenceや#308 PostgreSQL Evidenceを本Issueのactual Oracle Evidenceへ読み替えること
- Gate B Readyをactual Oracle Readyへ読み替えること
- external Evidence本文やcredentialをacceptance receiptへ複製すること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #69 / #179 / #185 / #214 / #222 / #224 / #255 / #273 / #291 / #308 / #310 / #371 / #376 / #381
- PR #288 / #300 / #306 / #340 / #342 / #354 / #355 / #356 / #358 / #360 / #364 / #366 / #369 / #372 / #374 / #378 / #379
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/issue/000033-gate-b-oracle-driver-integration.md`
- `plan/issue/000034-gate-b-oracle-read-only-evidence.md`
- `plan/issue/000066-gate-b-actual-oracle-blocker-reaudit.md`
- `tools/internal/repositoryevidence/`
- `tools/reference-external-query-evidence/README.md`
- `tools/reference-external-query-evidence/main.go`
- `internal/driver/queryruntime/oracle/runner.go`
