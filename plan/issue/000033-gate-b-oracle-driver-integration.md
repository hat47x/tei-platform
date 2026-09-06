# ISSUE-000033: Gate Bでactual Oracle driver integrationを確定する

- 状態: Blocked
- GitHub Issue: #371
- 親Issue: `ISSUE-000022` / GitHub Issue #214
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連ADR: `ADR-0007` actual database driverの組み立てをProvider Coreから分離する
- 関連Issue: `ISSUE-000006` / #69、`ISSUE-000032` / #310
- 関連RQ: RQ-009 / RQ-012 / RQ-014

## 目的

`master_data`のactual Oracle readに利用する具体driverとHost / Reference assemblyを、実Oracleの接続条件に基づいて確定し、named parameter、result、NULL、error伝播、raw column metadata、read-only境界を実接続で確認します。

このIssueはOracle Provider CoreやCanonical modelへ具体driverを固定するためのものではありません。

`ADR-0007`に従い、次はHost / Reference assembly側の責務として扱います。

- driver選択と登録
- DSN
- credential
- TLS / TCPS
- Wallet等の接続設定
- connection pool等のHost設定

Provider Coreは既存の`database/sql`境界を維持します。

## 現在地

TEIの`go.mod`はGo 1.23.2です。

#214では、特定driverに依存しないOracle向け`database/sql` Runner境界まで成立しています。PR #364 / #366により、actual readで実際に観測したraw列名と、driverが提供できた範囲のcolumn metadataをprovider固有Payloadへ保持できます。

一方、実Oracleのversion、authentication、TCPS / Wallet、read-only方式等はまだ観測されていません。

したがって、現時点ではdriver dependencyをRepositoryへ追加せず、候補と選定条件だけを固定します。

## 候補の現在評価

### Oracle公式 `github.com/oracle/go-oracledb/v26`

Oracle公式のnative Go `database/sql` driverです。

2026-09-04時点で確認できる公開版はbetaで、少なくとも次の条件があります。

- Oracle Database 19c以上を対象
- moduleのGo要件は1.26.6

公式実装であることは重要な利点ですが、現在のTEI Go 1.23.2からtoolchain変更を伴います。

**#214だけを理由にTEI全体のGo toolchainを引き上げません。** 実Oracle条件と公式driverを採用する価値を確認し、toolchain変更が必要なら別の判断として扱います。

### `github.com/sijms/go-ora/v2`

pure Goの`database/sql` driverです。

- Oracle Database 10.2+を対象
- moduleのGo要件は1.18

現行TEI Go 1.23.2と直接整合するため、実Oracleのversion / authentication / TCPS・Wallet / datatype等の要件を満たす場合は、第一Referenceで最初に検証しやすい候補です。

これは**暫定的な第一候補**であり、採用決定ではありません。

### `github.com/sijms/go-ora/v3`

23ai対応側ですが、moduleのGo要件は1.24です。

v2で必要要件を満たせず、v3固有機能が必要な理由を確認できた場合に検討します。

### `github.com/godror/godror`

成熟したOracle driverの選択肢ですが、現行upstreamはGo 1.25系で、Oracle Client / CGOを伴う構成です。

Oracle Client固有機能が必要である等の明確な理由がない限り、第一Referenceで最初に固定しません。

## upstream静的監査

実Oracleへ接続する前に、第一候補がTEIの`database/sql`境界と形状上かみ合うかだけをupstream sourceで確認しました。

これは**driver integration成功Evidenceではありません**。READMEやsourceに実装が存在することと、第一Referenceの実Oracleで期待どおり動作することを分離します。

### `go-ora/v2`の`database/sql` metadata境界

2026-09-04時点のupstream sourceでは、v2のresult setが少なくとも次の`database/sql/driver` column metadata境界を実装しています。

- `ColumnTypeDatabaseTypeName`
- `ColumnTypeNullable`
- `ColumnTypeLength`
- `ColumnTypePrecisionScale`

`ColumnTypeDatabaseTypeName`はcolumnのOracle datatype表現を返し、nullable / length / precision / scaleもそれぞれdriverが保持するcolumn metadataから返す実装です。

この形は、PR #366でTEI Oracle Runnerが`Rows.ColumnTypes()`から観測する情報と直接対応します。

ただし、実Oracleでどのdatatypeに対してどの値が返るかは別途actual readで確認します。source実装の存在から、第一Referenceの実値を推測しません。

### named parameter

upstream README / exampleには`sql.Named`を用いたnamed parameterの実例があり、v2 connection / statementも`driver.NamedValue`を受け取る境界を持ちます。

したがって、TEIのauthoring済みphysical parameter名を`database/sql`のnamed bindとして渡す構造とは形状上整合します。

ただし、第一Referenceのnative SQLで同名bindの再利用、case、datatype、NULL等が期待どおり動くかはactual Oracleで確認します。

### TLS / TCPS / Wallet

upstreamの接続設定には、少なくとも次の経路が存在します。

- SSL / TLS有効化
- server certificate verification
- Wallet path
- TCPS auth / protocol

このため、実OracleがTCPS / Walletを要求する場合でもv2を即座に候補外とする必要はありません。

ただし、対象環境のWallet形式、certificate chain、authentication方式との互換性はactual接続前に確認します。設定項目が存在することを互換性Evidenceへ格上げしません。

### v3との比較で見つかった注意点

同じupstream revisionのv3 sourceでは、`ResultSet.ColumnTypeDatabaseTypeName`が空文字を返す実装箇所を確認しました。

v3は23ai対応等の新しい機能を持ちますが、少なくともGate Bで必要なraw physical metadata Evidenceだけを理由に「新しいv3を優先する」とは判断しません。

実Oracle要件がv3を必要とする場合は、その時点の採用versionで`ColumnTypes()`挙動を改めて確認します。

### 現時点の候補順位

静的監査後も、第一検証候補は次の順を維持します。

1. `go-ora/v2` — 現行Go 1.23.2と整合し、named bind / ColumnTypes / TCPS・Walletの必要境界がsource上存在する
2. Oracle公式`go-oracledb/v26` — 公式実装の利点は大きいが、現行beta / Go 1.26.6 / Oracle 19c+という条件がある
3. `go-ora/v3` — 実Oracle要件がv3固有機能を必要とする場合に再評価する
4. `godror` — Oracle Client / CGO等を必要とする明確な理由がある場合に再評価する

これは採用順ではなく、**実環境条件を得た後に最初に適合確認する順序**です。

## driver固定前に必要な実環境情報

実Oracleから少なくとも次を確認します。

### Database / protocol

- Oracle Database version
- service / connection方式
- TCP / TCPS
- Walletの要否
- server certificate / trustの扱い

### authentication / read-only

- authentication方式
- 第一Reference用principal / credentialの方式
- read-onlyを実効強制するDatabase permission / principal / session等の方式
- connection確立後にwrite APIを利用できないことの確認方法

### Query integration

- named bind parameterの実挙動
- 第一Referenceで必要なOracle datatype
- SQL NULLのdriver表現
- Query Error / row取得Errorの挙動
- `database/sql.Rows.ColumnTypes()`が実際に返すmetadataの範囲

### Host制約

- Oracle Client導入可否
- CGO利用可否
- network policy
- certificate / Wallet等をHostへ配置する方法

未確認条件を慣習やdriver READMEだけから補完しません。

## 選定原則

1. Canonical Contractをdriver都合で変更しない
2. `database/sql`境界を維持する
3. 現行Go 1.23.2で必要要件を満たせる候補を優先する
4. pure Goは配布・再現性上の利点として評価するが絶対条件にはしない
5. actual Oracle version / authentication / encryption要件を満たす
6. read-onlyの実効強制方式と両立する
7. raw列名とdriver観測metadataをactual readで確認できる
8. driverがmetadataを提供しない項目をTEI側で推測しない
9. DSN / credential / parameter値 / raw業務値を通常Evidenceへ保存しない
10. driver追加をCanonical modelやProvider-neutral境界へ逆流させない

## PR #364 / #366との接続

actual Oracle Runnerは次をprovider固有Payloadへ保持できます。

- 実行時に利用したphysical parameterの名前
- `Rows.Columns()`が返したraw result列名
- `Rows.ColumnTypes()`からdriverが提供できた範囲のmetadata
  - `name`
  - `databaseTypeName`
  - `nullable`
  - `length`
  - `precision`
  - `scale`

次は保存しません。

- runtime parameter value
- raw row value
- credential
- DSN
- Go側の`ScanType`

metadata未提供時は未知のままにし、false / 0 / Oracleの慣習値へ補完しません。

`Rows.Columns()`と`Rows.ColumnTypes()`の件数または列名が一致しない場合も、誤対応させずExecution Errorとして停止します。

## actual integrationで確認すること

採用driverを決めた後、第一Referenceでは少なくとも次を確認します。

1. Host / Reference assembly側でdriverを`database/sql`へ登録する
2. secretをRepositoryへ保存せずread-only Oracle connectionを構成する
3. `Ping`または同等の接続確認を行う
4. authoring済みnamed parameterをそのまま渡す
5. actual native Oracle sourceを実行する
6. 1件以上の正常resultを確認する
7. 0件を正常な0件として確認する
8. Query / row取得失敗をExecution Errorとして確認する
9. raw列名を実観測する
10. driverが提供できるraw column metadataを確認する
11. SQL NULLを別値へ変換していないことを確認する
12. raw resultから明示Semantic mappingを適用する
13. credential / DSN / runtime parameter value / raw業務値が通常Evidenceへ残らないことを確認する

## `PhysicalType`との関係

external Evidenceの`PhysicalType`と、actual read時にdriverが返す`DatabaseTypeName`は、同じ文字列になるとは限りません。

実Oracleを観測する前に、両者の自動文字列一致規則やnormalization規則を固定しません。

まず実driverの返却値を観測し、必要であれば次の順に判断します。

1. 同じ物理型を異なる表現で示しているだけか
2. provenance上どの表現をEvidenceへ残すべきか
3. driver固有normalizationをHost / adapter境界へ置くべきか
4. provider-neutral modelへ持ち上げる意味が本当にあるか

Conformanceを通すためだけの文字列置換は行いません。

## Gate B candidate revision Rとの関係

このIssueでdriver dependencyやHost assemblyを追加するとrepository revisionが変わります。

そのためdriver integrationはGate B Phase 1で確定・commitし、その後にcandidate revision `R`を固定します。

Phase 2では、#214の最終actual readを**RのWorkspace / realization / runtime / driver assembly**で再取得します。

古いrevisionでdriver integrationが成功していても、Rのactual read Evidenceへ読み替えません。

## 完了条件

- [ ] 実Oracle versionを確認する
- [ ] connection / authentication / TCPS・Wallet条件を確認する
- [ ] read-only実効強制方式を確認する
- [ ] 必要datatype / NULL / named bind条件を確認する
- [ ] Host制約を確認する
- [ ] 採用driverとversionを決定する
- [ ] Go toolchain変更が必要なら別判断として明示する
- [ ] driverをHost / Reference assembly側へ追加する
- [ ] read-only Oracle connectionからactual Queryを実行する
- [ ] named parameter / result / NULL / error伝播を確認する
- [ ] raw列名 / driver観測metadataを確認する
- [ ] secret / parameter値 / raw業務値が通常Evidenceへ残らないことを確認する
- [ ] #214 actual readへ統合する
- [ ] candidate Rで最終actual readを再取得する
- [ ] `ISSUE-000029` / #273へ結果を対応付ける

## 非目標

- driverをCanonical modelへ固定すること
- Oracle Provider Coreへcredential / DSNを持ち込むこと
- 未観測のOracle object / column / datatypeを推測すること
- driver都合でContract Query signatureを変更すること
- `PhysicalType`と`DatabaseTypeName`の自動一致規則を実観測前に固定すること
- external Evidence acceptance receiptをactual readへ格上げすること
- PostgreSQL fixture Evidenceで代替すること
- upstream sourceの静的実装確認をactual Oracle成功Evidenceへ格上げすること
- driver選定だけを理由にGitHub Actionsを再有効化すること

## 関連

- GitHub Issue #69 / #214 / #273 / #371
- PR #340 / #354 / #360 / #364 / #366 / #369 / #372
- `plan/adr/0007-database-driver-assembly.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `internal/driver/queryruntime/oracle/`
