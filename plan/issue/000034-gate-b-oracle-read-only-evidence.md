# ISSUE-000034: actual Oracle read-only Evidenceを確定する

- 状態: Blocked
- GitHub Issue: #376
- 親Issue: `ISSUE-000022` / GitHub Issue #214
- 関連Issue: `ISSUE-000033` / GitHub Issue #371
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-009 / RQ-012 / RQ-014

## 現行scope

[ISSUE-000035](000035-gate-b-external-provider-scope-audit.md)の判断に従い、本Issueは[ISSUE-000022](000022-external-master-data-physical-query-evidence.md)配下の外部資産検証です。Gate Bの第4・第5laneではありません。driver・接続確認は[ISSUE-000033](000033-gate-b-oracle-driver-integration.md)、本Issueは実効権限と安全な確認を担当します。実Oracleの権限条件が未取得である限りBlockedを維持します。

以下のGate B Phase 1 / 2に関する記述はscope変更前の経緯です。権限Evidenceとactual readを外部資産検証の対象revisionへ追跡可能にする条件は維持し、Gate B前の実Oracle検証を再び必須にはしません。

## 検証の目的

第一Referenceの`master_data`へのactual Oracle readについて、**実効的にread-onlyであることを、実外部assetへ試し書きせず安全にEvidence化する**ための条件を確定します。

本IssueはGate Bの5本目のEvidence laneではありません。`ISSUE-000022` / #214 actual external Oracle laneの完了条件の一部です。

`find_by_id`がQueryであること、native SQLが`SELECT`に見えること、TEI Runnerが`QueryContext`を使うことのいずれも、それ単独ではDatabase principal自体が書込不能であることを証明しません。

一方、実外部assetへDML / DDLを試しに発行して失敗を確認する方法は、権限設定が想定より広かった場合に実データやavailabilityへ影響し得ます。本Issueでは採用しません。

## read-onlyを一つのmechanismへ押し込めない

第一Referenceでは、少なくとも次を別の防御層として扱います。

```text
Database grant / principal
        ↓
Host / Reference assembly
        ↓
Oracle transaction boundary
        ↓
TEI query-only runtime
        ↓
actual native Query
```

各層が証明するものを混ぜません。

- Database grant: credential / principalに何が許されているか
- transaction: 当該actual readをどのtransaction modeで行ったか
- TEI runtime: Reference read pathからどのDB APIを利用するか
- native Query: 実際にどのreadを実行したか

## 第一候補の構成

actual Oracle versionが対応し、既存資産側でgrant調整が可能なら、第一候補は次です。

```text
dedicated least-privilege principal
  └─ 必要objectへのREAD object privilegeだけ
        ↓
credential / DSNはRepository外
        ↓
SET TRANSACTION READ ONLY
        ↓
TEI Oracle Contract Query Runner
  └─ QueryContext
        ↓
actual native Query
```

`READ` privilege、read-only transaction、query-only Runnerのどれか一つだけを「read-only保証全体」とは扱いません。

## Oracle `READ` object privilege

Oracle Database 12c Release 1 (12.1.0.2)では、queryだけを許可するための`READ` object privilegeと`READ ANY TABLE` system privilegeが追加されています。

Oracle公式Security Guideでは、queryだけを許可したい場合に`SELECT`ではなく`READ` object privilegeを使うことが推奨されています。

`SELECT` object privilegeはqueryに加えて、少なくとも次を許可できます。

- `SELECT ... FOR UPDATE`
- `LOCK TABLE ... IN EXCLUSIVE MODE`

`READ` object privilegeはこれらを許可しません。

したがって、actual Oracleが`READ`をサポートし、対象objectへのgrantを調整可能なら、必要objectへのobject-level `READ`を第一候補にします。

### 広いsystem privilegeを既定にしない

`READ ANY TABLE`はquery-onlyのsystem privilegeですが、第一Referenceが必要とするobjectより広い範囲を参照できます。

便利さだけを理由に採用せず、必要objectへ限定したgrantで成立するならobject-level `READ`を優先します。

`SELECT ANY TABLE`はさらに`SELECT ... FOR UPDATE`等を許し得るため、第一Referenceのquery-only用途では必要性が確認されない限り採用しません。

### 古いOracleへの適用

`READ` privilegeは12.1.0.2で追加されたため、actual Oracleがそれ以前なら利用できません。

その場合は、object-level `SELECT`等を含む最小権限構成を実環境に合わせて評価し、`READ`が存在するかのようにEvidenceを書き換えません。

## `SET TRANSACTION READ ONLY`の位置づけ

Oracleの`SET TRANSACTION READ ONLY`はcurrent transactionをread-onlyにし、そのtransaction内のqueryへtransaction-level read consistencyを提供します。

これはactual Queryに対する有用なruntime防御ですが、**Database principalの恒久的な書込不能性を証明するものではありません。**

Oracleでは`COMMIT` / `ROLLBACK`でtransactionが終了します。またDDLはtransaction境界へ影響します。

そのため、次の推論を禁止します。

```text
SET TRANSACTION READ ONLYが成功した
  therefore
credentialにはwrite権限がない
```

第一Referenceでは、Database grant / principal Evidenceを主たる権限根拠とし、`SET TRANSACTION READ ONLY`をactual Queryの追加防御として扱います。

採用する場合、Oracle仕様どおり対象transactionの最初のSQL statementとして成立することを`ISSUE-000033`のdriver integrationで確認します。

## privilege Evidence

実assetへwriteを試すのではなく、grant / enabled privilegeの観測結果と管理側の再確認可能なsourceをEvidenceにします。

Oracleには少なくとも次のData Dictionary Viewがあります。

- `SESSION_PRIVS`: current sessionで有効なsystem privilege
- `SESSION_ROLES`: current sessionで有効なrole
- `USER_SYS_PRIVS`: current userへ直接付与されたsystem privilege
- `USER_ROLE_PRIVS`: current userへ直接付与されたrole
- `USER_TAB_PRIVS`: current userがgrantee等であるobject privilege
- 必要に応じて`ROLE_SYS_PRIVS` / `ROLE_TAB_PRIVS`等

ただし、一つのself-query viewだけから「間接grantを含め絶対にwrite権限がない」と推論しません。

少なくとも次の可能性を意識します。

- role経由のgrant
- enabled / disabled roleの違い
- `PUBLIC`等の間接的なgrant
- object ownershipに伴う権限
- system privilegeとobject privilegeの双方

DBA側のgrant定義、role構成、承認記録等へ戻れるEvidence参照を残します。

principal identityを外部公開してよいかも確認し、必要なら匿名化 / alias化し、その加工をprovenanceへ残します。credentialやpasswordは保存しません。

## safe verification原則

read-only確認だけを目的として、実外部assetへ次のprobeを原則行いません。

- `INSERT`
- `UPDATE`
- `DELETE`
- `MERGE`
- `CREATE` / `ALTER` / `DROP`
- `SELECT ... FOR UPDATE`
- exclusive `LOCK TABLE`

理由は、権限設定が想定より広かった場合に次を起こし得るためです。

- 実データ変更
- DDL変更
- row / table lock
- 他利用者へのavailability影響

read-only保証は、grant Evidence、transaction境界、query-only runtime、actual SELECT成功の組合せで確認します。

negative probeが必要だと後から判断した場合も、production既存assetではなく、用途が明示された専用検証資産で別Evidenceとして設計します。

## TEI runtime側の境界

現在のOracle Contract Query Runnerは`database/sql`のQuery実行境界を利用します。

第一Referenceのactual read pathからwrite用`Exec`を公開する必要はありません。

ただし、次は別の主張です。

```text
TEI Runnerがwrite APIを公開しない
```

```text
Database credentialにwrite権限がない
```

前者を後者の代替にしません。

runtime Evidenceでは、actual Queryがread-only Reference pathを通ったことを確認します。権限Evidenceではprincipal / grantを確認します。

## `READ`と`SELECT`のEvidence上の意味

actual Oracleでobject-level `READ`を利用できる場合、Evidenceには単に`query privilegeあり`と丸めず、少なくとも次を区別できるようにします。

- `READ`
- `SELECT`
- `READ ANY TABLE`
- `SELECT ANY TABLE`
- DML object privileges
- DDL / ANY系system privileges

ただし、これらをTEI Canonical modelへ列挙することは本Issueの目的ではありません。

external read-only Evidence artifactまたはその参照先で、実際のgrant境界を再確認可能にすることが目的です。

## source / provenance

read-only Evidenceには少なくとも次へ戻れる参照を要求します。

- どのprincipal / role構成を確認したか
- どのDatabase / environmentに対するものか
- どのgrant / privilege sourceから確認したか
- self-queryかDBA側確認か
- いつ観測したか
- 匿名化・縮約等を行ったか

Evidence artifactが実principal名を匿名化する場合も、元sourceへ戻れる管理上の参照を失わないようにします。

## `ISSUE-000033`との責務分離

### 本Issue

- principal / role / grant
- read-only transactionを採用する条件
- safe verification
- read-only Evidence参照

### `ISSUE-000033`

- concrete Oracle driver
- Host / Reference assembly
- connection / authentication / TCPS / Wallet
- named parameter
- actual result / NULL / error
- driver-reported raw column metadata

接点はactual Oracle readです。

`SET TRANSACTION READ ONLY`の発行方法がdriver固有になる場合、その実装は`ISSUE-000033`側で行いますが、何をもってread-only Evidenceとするかは本Issueの契約に従います。

## Gate B candidate revision Rとの関係

read-only transaction setupやHost assemblyにsource変更が必要なら、Gate B Phase 1で確定・commitします。

その後にcandidate revision `R`を固定します。

Phase 2の#214 actual readでは次を同時に満たします。

- RのWorkspace / realization / runtime / driver assemblyを使う
- source-fixed external Evidence acceptance receiptをRで取得する
- read-only grant Evidenceへ戻れる
- actual Queryを採用したread-only runtime境界で実行する
- actual read EvidenceがRを参照する

古いrevisionでread-only integrationが成功していても、Rのactual read Evidenceへ読み替えません。

## 完了条件

- [ ] actual Oracle versionを確認し、`READ` privilege利用可否を確定する
- [ ] 第一Reference用principal / role構成を確認する
- [ ] 対象objectへ必要最小限のquery privilegeだけが与えられていることを確認する
- [ ] 不要なDML object privilegeがないことを再確認できるEvidence参照を得る
- [ ] 不要なDDL / ANY系system privilegeがないことを必要範囲で再確認する
- [ ] indirect role / `PUBLIC`等を含む権限経路を必要範囲で確認する
- [ ] credential / passwordをEvidenceへ保存しない
- [ ] principal identityを匿名化する場合、そのprovenanceを残す
- [ ] `SET TRANSACTION READ ONLY`を採用するか確定する
- [ ] 採用する場合、actual driverで対象transactionの最初のSQLとして成立することを確認する
- [ ] TEI Reference read pathがquery-only runtime境界を保つことを確認する
- [ ] 実assetへのwrite / lock probeを行わずread-only Evidenceを取得する
- [ ] #214 external Evidenceの`readOnly` Evidence参照を実根拠へ結び付ける
- [ ] #214 actual read Evidenceへread-only Evidenceを対応付ける
- [ ] candidate Rで最終actual readを再取得する
- [ ] `ISSUE-000029` / #273へ結果を対応付ける

## 非目標

- 実外部assetへの試し書きで権限を検証すること
- `SELECT`に見えるSQLだけでread-onlyを主張すること
- `SET TRANSACTION READ ONLY`だけでcredentialをread-onlyと判定すること
- `READ ANY TABLE`等の広い権限を便利さだけで採用すること
- privilege情報をCanonical Semanticへ持ち上げること
- credential / secretをEvidenceへ保存すること
- read-only Evidenceからexternal physical schemaを推測すること
- 本Issueを独立した5本目のGate B Evidence laneとすること
- GitHub Actionsを再有効化すること

## 関連

- GitHub Issue #69 / #214 / #273 / #371 / #376
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000033-gate-b-oracle-driver-integration.md`
- `internal/driver/queryruntime/oracle/`

現時点ではactual Oracle principal / grant構成が未観測のため、本IssueはBlockedです。未観測の権限を推測して完了扱いしません。
