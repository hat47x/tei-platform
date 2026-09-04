# ISSUE-000022: external-observed `master_data`の物理Query Evidenceを確定する

- 状態: In Progress
- GitHub Issue: #214
- 親Issue: `ISSUE-000018` / GitHub Issue #179
- 関連Issue: `ISSUE-000020` / `ISSUE-000021`
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 目的

第一Referenceの`master_data`について、TEIが所有していない既存資産の実物理事実、provenance、read-only境界、Oracle driverを通したactual readを確認し、`Order -> Customer` Referenceが実環境でもSemantic meaningを保って動作することをGate B Evidenceとして確定します。

`master_data`は`external-observed` Databaseです。`development`では`master_data.dev`としてBindingされ、providerはOracleです。

このIssueでは、未観測の物理SchemaをTEI側から生成・推測しません。`CUSTOMER`、`CUSTOMER_ID`等を命名慣習から仮置きすると、既存資産受入れのReferenceがTEI都合の架空Schemaへ変質するためです。

## 現在の判定

**external Evidenceを受け入れる非Canonical境界はReadyですが、第一Referenceのactual physical facts / read-only / Oracle actual readは未取得です。**

したがって本IssueはGate B blockerのままです。

PR #340以降、Evidence受入れの成功は単なる人向けメッセージではなく、**どのexternal Evidence artifactを、どのclean Repository revisionで照合したか**を示す機械可読receiptとして残します。

ただし、acceptance receiptはactual Oracle read Evidenceではありません。

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

## source-fixed acceptance receipt

PR #340で、受入れrun自体を一つのclean Repository revisionへ固定し、実際に検査したEvidence artifactをfingerprintで識別する境界を追加しました。

### 実行開始前

external Evidenceを読む前に次を確認します。

```text
git rev-parse HEAD
git status --porcelain
```

working treeがdirtyなら受入れを停止します。

Repository状態の観測には`tools/internal/repositoryevidence`を利用し、#310 deterministic Repository Evidence、#255 generated artifact Evidence、#308 PostgreSQL Provider Evidenceと同じsource再現性規則へ揃えます。

### Evidence artifact fingerprint

Evidence fileをexact bytesで読み、SHA-256を計算します。

```text
evidenceArtifactFingerprint = sha256:<hex>
```

これは、意味的に近い別fileではなく、**toolが実際に読み込んだexternal Evidence JSON artifactそのもの**を識別します。空白や改行だけを変更した場合も別artifactとして別fingerprintになります。

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

Gate B用にreceiptを保存する場合、Repository内へstdoutを直接redirectしません。command開始前に新規fileが作成されるとworking treeがdirtyになり、clean revision要件と自己矛盾するためです。working tree外の一時領域等へ保存します。

## 二つのfingerprintを分離する

### `evidenceArtifactFingerprint`

external Evidence JSON exact bytesを識別します。

証明するのは「どのEvidence artifactを実際に受理検査したか」です。

### `nativeSource.fingerprint`

Evidenceが観測したnative sourceと、現在authoringされている`ContractQuerySQLRealization.Source`が一致するかを確認します。

証明するのは「観測対象native sourceと現在の実行authoringが同じか」です。

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

acceptance receiptは照合追跡用の小さい非Canonical Evidenceです。

次はreceiptへ複製しません。

- physical object identity
- physical parameter / raw result columns
- physical type / NULL情報
- provenance本文
- read-only mechanism本文
- Evidence file path
- credential / secret

actual physical factsそのものは、元のexternal Evidence artifactと、そのprovenance / Evidence参照で追跡します。

## まだ必要なactual physical Evidence

受入境界が成立しても、第一Referenceの外部事実は実対象から得る必要があります。

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

実環境に応じ、たとえば次のいずれかで書込み不能を確認します。

- read-only権限だけを持つDatabase principal / credential
- Database側permissions
- read-only session / transaction
- write APIを公開しないReference adapter

Evidenceにはmechanism名だけでなく、その事実を再確認できるEvidence参照も要求します。

ただし、mechanism文字列の記載だけをTEIが真実と判定するわけではありません。実接続・権限Evidenceによる確認が必要です。

## Oracle driver integration

特定Oracle driverに依存しない`database/sql` Runner境界までは成立しています。

actual physical Evidence取得後、採用driverについて少なくとも次を確認します。

- named parameterが期待どおりに渡る
- 対象native sourceが実行できる
- input / resultの型変換
- NULLの扱い
- raw column名
- Query Error / row取得Errorの伝播
- read-only強制との整合

特定driverの都合に合わせてCanonical Contractを変更しません。

## actual Evidence取得後の作業順

```text
1. 実physical object / parameter / raw columns / native sourceを観測
2. provenance / read-only根拠を確認
3. external Evidence JSONを作成
4. 観測事実を根拠にContractQuerySQLRealizationを明示authoring
5. clean revisionでreference-external-query-evidenceを実行
6. source-fixed acceptance receiptを取得
7. read-onlyを強制したOracle接続境界を用意
8. 採用Oracle driverでparameter / result / NULL / error伝播を確認
9. development / master_data.dev / database.query.oracleからactual read
10. raw resultを明示mappingしてSemantic Customerを得る
11. Workspace -> actual read -> Semantic resultを一続きに確認
12. #273 Gate B監査へactual external asset Evidenceとして対応付ける
```

acceptance receiptとactual read Evidenceは別工程です。

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

### 第一Referenceのactual Evidence

- [ ] 第一Referenceで参照する実physical objectが明示されている
- [ ] Contract input `id`から実physical parameterへの対応と型 / NULL扱いが明示されている
- [ ] 実raw result columnsと型 / NULL扱いが明示されている
- [ ] raw resultから必要なSemantic attributeへの対応が明示されている
- [ ] 実native SQL / API / adapter等のrealization sourceを追跡できる
- [ ] 物理情報のsource / provenanceを追跡できる
- [ ] actual Evidence artifactをclean revisionで受理しsource-fixed receiptを取得できる
- [ ] `master_data`を実効的にread-onlyのまま実行できる
- [ ] 採用Oracle driverとのintegration Evidenceがある
- [ ] 第一ReferenceのWorkspace → actual read → Semantic resultを一続きに確認できる
- [ ] physical fixture / adapter / EvidenceがSemantic meaningの正本として扱われていない
- [ ] 結果を`ISSUE-000029` / #273へ対応付ける

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
- acceptance receiptをactual Oracle read Evidenceとして扱うこと
- 汎用ORMの導入
- Oracle固有の高度なHA / Connection Poolを本Issueだけで完成させること
- synthetic Conformance SQLを第一Referenceの実native sourceへ流用すること
- SQL文字列だけでread-onlyを推論すること
- external Evidence本文やcredentialをacceptance receiptへ複製すること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #69 / #179 / #185 / #214 / #222 / #224 / #255 / #273 / #308 / #310
- PR #288 / #300 / #306 / #340 / #342
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `tools/internal/repositoryevidence/`
- `tools/reference-external-query-evidence/README.md`
- `tools/reference-external-query-evidence/main.go`
