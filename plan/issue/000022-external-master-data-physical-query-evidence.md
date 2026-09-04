# ISSUE-000022: external-observed `master_data`の物理Query Evidenceを確定する

- 状態: In Progress
- GitHub Issue: #214
- 親Issue: `ISSUE-000018` / GitHub Issue #179
- 関連Issue: `ISSUE-000020` / `ISSUE-000021`
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

第一Referenceの`master_data`は、TEIが所有・生成するDatabaseではなく、既存資産を観測して利用する`external-observed` Databaseです。`development` Environmentでは`master_data.dev`としてBindingされ、providerはOracleです。

RepositoryにはDatabase Contract、Reference invocation、Environment / executor Binding、Oracle向けruntime境界までありますが、第一Referenceで本当に利用する物理資産そのものはまだ観測されていません。

未確定なのは、たとえば次です。

- 実際に参照するtable / view / API / stored procedure等
- Contract input `id`へ対応するphysical parameter
- 実際に返却されるraw result列
- raw result列と`customer.customer`のSemantic attributeの対応
- native Oracle SQLまたは他の実realization source
- これらを確認したsource / provenance
- 書込み不能を実効的に保証する境界

`CUSTOMER`、`CUSTOMER_ID`等を命名慣習から仮定して埋めることはしません。未観測の外部事実をTEI側から補完すると、既存システム受入れのReferenceがTEI都合の架空Schemaへ変質するためです。

## 現在の判定

**物理Evidenceを受け入れるための非Canonical境界と、その受入れをclean Repository revisionへ固定するreceipt境界は成立していますが、第一Referenceの実物理Evidence自体は未取得です。**

したがって#214は引き続きGate Bのexternal Evidence blockerです。

受入境界は、外部EvidenceをCanonical Worldへ取り込むものではありません。観測事実を受け取った後、別途明示authoringされた`ContractQuerySQLRealization`と矛盾していないことを確認し、**どのEvidence artifactを、どのTEI source revisionで照合したか**を非Canonicalなacceptance receiptへ残します。

```text
external physical evidence
  object / parameter / raw columns
  native source proof
  provenance
  effective read-only proof
          ↓
non-canonical Evidence intake
          ↓ compare only
explicit ContractQuerySQLRealization
          ↓
source-fixed acceptance receipt
          ↓
Workspace / Driver runtime
          ↓
actual Oracle read Evidence
```

EvidenceからSQL realizationを生成・補完する逆向き経路は設けません。acceptance receiptもactual Oracle readの代替にはしません。

## 既に成立している実行境界

物理Schemaの具体名を知らなくても、意味と物理実装を混ぜないための境界は先に成立しています。

```text
Database Contract Query
  master_data.customer / find_by_id
        ↓ explicit realization
ContractQuerySQLRealization
  ├─ target: Database / Contract / Query
  ├─ provider
  ├─ Contract input -> physical parameter
  ├─ native SQL source
  └─ raw result column -> Semantic attribute
        ↓
Environment / Database instance / executor Binding
        ↓
instance-aware runtime Registry
        ↓
Oracle database/sql Runner
        ↓ explicit result mapping
ContractQueryResult
  └─ Semantic rows
```

成立済みの主な境界は次です。

- Relation keyからContract Query inputを準備する
- Contract Query signatureを型付きで表現・Validationする
- Reference invocationを明示する
- Relation / invocation / Query signatureを横断Validationする
- Database / Contract / Queryをrealization target identityとする
- Contract input → physical parameterを明示mappingする
- raw result column → Semantic attributeを明示mappingする
- SQL realizationを通常Workspaceから読み込む
- SQL realization authoring / native SQL sourceをSource Indexで追跡する
- Environment / Database instance / executor Capabilityを明示Bindingする
- runtime RegistryでDatabase instance identityを保持する
- 同一Capabilityを共有する別Database instanceへfallbackしない
- Oracle向け最小`database/sql` Runnerで明示SQLとnamed parameterをraw executionへ渡す
- 0 rowとExecution Errorを区別する
- SQL realizationがないReference Workspaceでは物理Runnerを呼ばず明示的に停止する

これらは実物理名の代用品ではありません。

## external Evidence受入境界

#214のため、`internal/database/externalqueryevidence`を非CanonicalなEvidence検査境界として追加します。

Evidenceが保持するのは、第一段階では次です。

- Contract Query target identity
- observed provider
- physical objectのkind / identity
- Contract inputとphysical parameterの観測対応
- raw result列とSemantic attributeの観測対応
- physical type / nullable等の補助観測情報
- native SQL source Evidence参照
- authoring済みnative SQL本文のfingerprint
- physical factsのsource / provenance
- Reference用Evidenceへの加工方法
- 実効的read-only保証のmechanism / Evidence参照

### SQL source fingerprint

Evidenceが古いSQLの観測結果を新しいrealizationへ流用しないよう、`ContractQuerySQLRealization.Source`のfingerprintを照合します。

SQL本文を解析して意味を推測するのではなく、**このEvidenceが確認したSQL本文と現在authoringされているSQL本文が同一か**だけを確認します。

### strict JSON loader

Evidence JSONは未知fieldや後続JSON値を黙って無視しません。

formatの不一致は明示的に停止し、その後にCanonical World / SQL realizationとの意味整合を検査します。

### Reference検査ツール

`tools/reference-external-query-evidence`を追加し、外部から受け取ったEvidence JSONをWorkspaceと照合できるようにします。

```text
go run ./tools/reference-external-query-evidence \
  -workspace reference/workspace \
  -evidence <observed-evidence.json>
```

`-evidence`は必須です。Repositoryに架空のmaster_data Evidenceを標準配置しません。

対応するtarget + providerのSQL realizationがWorkspaceに存在しない場合は停止します。Evidenceからrealizationを作ることはありません。

## source-fixed acceptance receipt

PR #340で、`tools/reference-external-query-evidence`の成功を人向け文字列だけで終わらせず、**どのexternal Evidence artifactを、どのclean Repository revisionで照合したか**を機械可読なreceiptとして残す境界を追加しました。

Gate Bへ採用する受入れは、Evidence JSONを読む前に次を確認します。

1. `git rev-parse HEAD`でRepository revisionを取得する
2. `git status --porcelain`でworking treeがcleanであることを確認する
3. dirtyならEvidenceを読む前に停止する

Evidence / Workspace / SQL realizationの照合後にもRepository状態を再確認し、次の場合は成功receiptを出しません。

- 実行中にHEADが変化した
- 実行後にtracked / untracked差分が残った

Repository状態の観測には`tools/internal/repositoryevidence`を利用し、#310 / #255 / #308と同じclean revision規則へ揃えます。

成功receiptは概ね次の非機密metadataだけを保持します。

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

receiptへexternal Evidence本文を複製しません。少なくとも次はreceiptへ出しません。

- physical object identity
- physical parameter / raw result column
- physical type / NULL情報
- provenance本文
- read-only mechanism本文
- Evidence file path
- credential / secret

### Evidence artifact fingerprint

`evidenceArtifactFingerprint`は、toolが実際に読み込んでstrict parseした**Evidence JSON fileのexact bytes**に対するSHA-256です。

```text
sha256:<hex>
```

空白や改行だけが変わっても別artifactとして別fingerprintになります。これは、意味的に近い別fileではなく、実際に受入検査したartifactそのものを識別するためです。

このfingerprintと`nativeSource.fingerprint`は役割が異なります。

- `evidenceArtifactFingerprint`: どのEvidence JSON artifactを受入検査したか
- `nativeSource.fingerprint`: Evidenceが観測したnative sourceが現在authoring済み実行SQL本文と一致するか

相互代替しません。独立したprovenanceも引き続き`nativeSource.evidenceRef` / `provenance.sourceRef`で追跡します。

receiptが成功しても、それは**external Evidence artifactと明示authoringの受入照合が成立したこと**だけを示します。Oracleへactual readしたこと、read-only principalが実際に効いたこと、Semantic Customerを実取得したことは別Evidenceです。

## Evidence受入時に検査すること

現時点の受入境界では、少なくとも次を機械的に確認します。

- 実行開始時のRepositoryがclean revisionである
- target identityがSQL realizationと完全一致する
- providerが既知かつ`generic`ではない
- providerがSQL realizationと一致する
- 対象Databaseが`external-observed`である
- SQL realization自体がDatabase Contract Query signatureと整合する
- physical objectのkind / identityが明示されている
- Contract inputの観測parameterがauthoring mappingと一致する
- raw result列の観測mappingがauthoring mappingと一致する
- native SQL fingerprintが現在のSQL本文と一致する
- physical factsのprovenanceがある
- Evidence加工方法がある
- read-only強制mechanismと、その根拠へのEvidence参照がある
- 実行後も同じHEADかつclean working treeである
- 受入れたEvidence exact bytesのfingerprintをreceiptへ残せる

これらの検査は「Evidenceの記載内容が真実である」ことを自動証明するものではありません。外部資料・DB権限・接続設定等の実根拠は別途観測する必要があります。

## 受入境界がしないこと

- external Evidenceを`ir.World`へ取り込まない
- EvidenceからSQL realizationを生成しない
- SQL本文からphysical parameterやSemantic attributeを推測しない
- physical object名からRelationやData意味を推測しない
- `SELECT`文字列だけをread-only保証とみなさない
- physical typeをCanonical Semantic型へ自動変換しない
- Evidence fixtureを業務意味の第二SSoTにしない
- acceptance receiptをactual Oracle read Evidenceとして扱わない
- Evidence本文をacceptance receiptへ複製しない
- dirty / source変化中のrunをGate B受入Evidenceとして採用しない

## まだ必要な実物理Evidence

受入境界ができても、第一Referenceの事実は外部から得る必要があります。

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
- raw columnsの型・NULLの扱い
- 第一Referenceで必要なSemantic attributeとの対応

外部Schema全体を取り込む必要はありません。

### native realization source

Oracle SQLを利用する場合は、実際に利用するnative SQL sourceをEvidenceとして確認します。

既存SQLを匿名化・縮約してReferenceへ持ち込む場合は、元EvidenceとReference用SQLの関係をprovenanceとして残します。

### provenance

少なくとも次を追跡します。

- 何を観測したか
- どのsourceから確認したか
- 直接利用か、匿名化・縮約等の加工をしたか
- どこまでが観測事実で、どこからがReference用加工か

## read-onlyの実効保証

`find_by_id`がQueryであることやSQLが`SELECT`に見えることだけではread-only保証としません。

実環境に応じ、たとえば次のいずれかで書込み不能を確認します。

- read-only権限だけを持つDatabase principal / credential
- Database側permissions
- read-only session / transaction
- write APIを公開しないReference adapter

受入Evidenceにはmechanism名だけでなく、その事実を再確認できるEvidence参照も要求します。

ただしmechanismの真偽そのものをTEIが文字列から判定するわけではありません。実接続・権限Evidenceによる確認が必要です。

## 実Oracle driverとのintegration

特定Oracle driverに依存しない`database/sql` Runnerまでは成立しています。

物理Evidence取得後、採用driverについて少なくとも次を確認します。

- named parameterが期待どおりに渡る
- 対象native SQLが実行できる
- input / resultの型変換
- NULLの扱い
- raw column名
- Query Error / row取得Errorの伝播
- read-only強制との整合

特定driverの都合に合わせてCanonical Contractを変更しません。

## Evidence取得後の作業順

1. 実物理object / parameter / raw列 / native source / provenance / read-only根拠を観測する
2. Evidence JSONを作成する
3. 観測事実を根拠に第一Referenceの`ContractQuerySQLRealization` authoringとnative SQL sourceを追加する
4. clean revisionで`reference-external-query-evidence`を実行し、EvidenceとWorkspace realizationを照合してacceptance receiptを取得する
5. receiptの`evidenceArtifactFingerprint` / repository revision / `reproducibleRevision = true`を確認する
6. read-onlyを強制した接続境界を用意する
7. 採用Oracle driverでnamed parameter / raw result / NULL / error伝播を確認する
8. `development / master_data.dev / database.query.oracle`のruntime Executorから実readする
9. Workspace → 実read → Semantic resultを一続きに確認する
10. Gate B監査へexternal Evidenceとして反映する

GitHub Actionsは現行Repositoryでは意図的に使用していません。将来再導入する場合も、CIのgreenをactual external Evidenceの代替にはしません。

## 完了条件

### Evidence受入境界

- [x] external EvidenceをCanonical Worldから分離して表せる
- [x] Evidence JSONを未知fieldを黙認せず読み込める
- [x] target / providerをSQL realizationと照合できる
- [x] external-observed ownershipを検査できる
- [x] Contract input → physical parameterをEvidenceとauthoringで照合できる
- [x] raw result → Semantic attributeをEvidenceとauthoringで照合できる
- [x] native SQL source fingerprintのstaleを検知できる
- [x] physical facts provenanceの欠如を拒否できる
- [x] read-only mechanism / Evidence参照の欠如を拒否できる
- [x] SQL realizationがない場合にEvidenceから補完せず停止できる
- [x] dirty working treeではEvidence本文を読む前に受入れを停止できる（PR #340）
- [x] Evidence受入れ後にHEAD / working treeを再確認できる（PR #340）
- [x] Evidence JSON exact bytesのSHA-256を受入receiptへ残せる（PR #340）
- [x] receiptへexternal Evidence本文を複製せず、target / provider / source revision等の非機密metadataだけを残せる（PR #340）

### 第一Referenceの実Evidence

- [ ] 第一Referenceで参照する実physical objectが明示されている
- [ ] Contract input `id`から実physical parameterへの対応が明示されている
- [ ] 実際のraw result列が明示されている
- [ ] raw result列から必要なSemantic attributeへの対応が明示されている
- [ ] 実native SQL / API / adapter等のrealization sourceを追跡できる
- [ ] 物理情報のsource / provenanceを追跡できる
- [ ] clean revisionでacceptance receiptを実取得し、受入れたEvidence artifactをfingerprintで特定できる
- [ ] `master_data`を実効的にread-onlyのまま実行できる
- [ ] 採用Oracle driverとのintegration Evidenceがある
- [ ] 第一ReferenceのWorkspace → 実read → Semantic resultを一続きに確認できる
- [ ] physical fixture / adapter / EvidenceがSemantic meaningの正本として扱われていない

### 既存runtime

- [x] 0件を正常なnot-foundとして扱える
- [x] Execution Errorと0件を区別できる
- [x] SQL realization sourceをSource Indexで追跡できる
- [x] runtime RegistryがDatabase instance identityを保持する
- [x] Oracle向け`database/sql` Runner境界がある

## 非目標

- 外部Database全体をTEI管理下へ移すこと
- 不明な物理SchemaをTEIが生成・推測すること
- external EvidenceをCanonicalへ昇格すること
- 汎用ORMの導入
- Oracle固有の高度なHA / Connection PoolをこのIssueだけで完成させること
- synthetic Conformance SQLを第一Referenceの実SQLとして採用すること
- read-only安全性をSQL文字列解析だけで判定すること
- acceptance receiptをactual Oracle read Evidenceとして扱うこと
- Evidence本文やcredential / secretをacceptance receiptへ複製すること

## 関連

- GitHub Issue #179 / #185 / #214 / #222 / #224
- PR #288 / #300 / #306 / #340
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000020-contract-query-runtime-instance-identity.md`
- `plan/issue/000021-oracle-contract-query-sql-runner.md`
- `reference/workspace/databases/master-data/database.tei`
- `reference/workspace/databases/master-data/contracts/customer.tei`
- `reference/workspace/environments/development/environment.tei`
- `reference/workspace/environments/development/contract-query-executors/customer-find-by-id.tei`
- `internal/database/externalqueryevidence/`
- `internal/realization/contract_query_sql.go`
- `internal/driver/reference_workspace_missing_realization_test.go`
- `internal/driver/queryruntime/oracle/runner.go`
- `tools/reference-external-query-evidence/`