# ISSUE-000021: Oracle Contract Queryの`database/sql` Runner境界を追加する

- 状態: Done
- GitHub Issue: #224
- 親Issue: `ISSUE-000018` / GitHub Issue #179
- 前提Issue: `ISSUE-000020` / GitHub Issue #222
- 関連RQ: RQ-005 / RQ-009 / RQ-012 / RQ-014

## 背景

Database Contract QueryのSQL realizationでは、次の境界までprovider非依存に明示できるようになっています。

```text
Contract Query input
        ↓ explicit mapping
physical parameter
        ↓
native SQL source
        ↓
raw result column
        ↓ explicit mapping
Semantic attribute
```

第一Referenceでは`master_data.dev`のproviderがOracleであり、executor Capabilityとして`database.query.oracle`を明示しています。

当初、SQL realizationを実際のDatabase接続へ渡す最下層の`sqlquery.Runner`には、Oracle向けの最小実装がありませんでした。ここでOracleの物理SchemaやSQLをRunner自身に持たせると、realizationとprovider adapterの責務が再び混ざります。

## 採用した境界

Oracle Runnerは、外部Databaseの意味を決めません。責務を次に限定します。

1. 上位から渡されたnative SQL sourceをそのまま使う
2. 既にphysical parameter名へ変換済みの値をnamed argumentとして`database/sql`へ渡す
3. Database接続が返した列名とrow値をraw resultとして返す
4. Evidenceにはparameter名だけを残し、runtime値を保存しない

接続先は注入済み`*sql.DB`だけを利用します。RunnerがEnvironment、Database definition、provider名、DSNなどから接続先を探すことはありません。

## named parameter

physical parameterはMapから受け取りますが、実行順とEvidenceを安定させるためparameter名を辞書順に並べて`sql.Named`へ渡します。

ここで行うのは順序の安定化だけです。parameter名を別名へ変換したり、Contract input名をphysical parameter名として暗黙流用したりしません。

## raw result

Runnerは`database/sql`が観測した列名とrow値を`queryresult.RawResult`へ返します。

Semantic attributeへの変換は上位の`ContractQuerySQLRealization.ResultColumns`が所有します。Runnerは列名の類似、SELECT順序、Semantic型などから意味を補完しません。

`[]byte`は返却前に複製し、driver側bufferの寿命から切り離します。

## Errorと0件

この境界では、次を明確に分けます。

- 0 row: 正常なraw result
- Query実行失敗: Execution Error
- row取得・走査中の失敗: Execution Error
- Database接続未指定: 構成Error
- native SQL未指定: 構成Error

`zero-or-one`などのContract cardinality判定は、このRunnerより上位のContract Query実行境界が扱います。

## 完了判断

PR #225で`internal/driver/queryruntime/oracle`を追加し、標準`database/sql/driver`だけで完結するConformanceを用意しました。

Conformanceでは次を確認しています。

- native SQL sourceを変更しない
- physical parameterをnamed argumentとして安定順で渡す
- parameter値をEvidenceへ保存しない
- raw列名とrow値を保持する
- 0 rowを正常結果として扱う
- Query失敗とrow取得途中の失敗をErrorとして保持する
- Database接続を暗黙生成しない
- 空のnative SQLを実行しない

この実装はPR #285でApplication Driverの主要境界とともに`main`へ収束しました。

このIssueの目的は、**Oracle向けの最小`database/sql` Runner境界を追加し、物理意味の決定を上位realizationから奪わないこと**です。この実装・Conformanceは成立したため、本Issueは完了とします。

採用するOracle driverとのnamed parameter互換性、第一Referenceのactual native SQL、read-only connectionでの実行Evidenceは、Runner境界そのものの未実装ではありません。external-observed `master_data`のactual physical Evidenceは`ISSUE-000022` / GitHub Issue #214、Gate B全体のEvidence判定は`ISSUE-000029` / GitHub Issue #273で追跡します。

## 完了条件

- [x] 注入済み`*sql.DB`以外から接続先を推測しない
- [x] physical parameter名を安定順でnamed argumentへ変換する
- [x] parameter値をPayload / Evidenceへ保存しない
- [x] raw resultの列名とrow値を返す
- [x] 0 rowを正常結果として返す
- [x] Query実行失敗をErrorとして返す
- [x] row取得途中の失敗をErrorとして返す
- [x] `[]byte`をdriver側bufferから切り離す
- [x] 特定Oracle driver packageをCoreへ固定しない
- [x] 実装を`main`へ収束する

## 後続Evidence

本Issueの完了とは分け、次を後続で確認します。

- #214: 実Oracle driverのnamed parameter / result / NULL / error伝播互換性
- #214: 第一Referenceのactual SQLをread-only connectionで実行し、Workspace → actual read → Semantic resultを確認すること
- #273: Gate B Evidenceとしてexternal / runner / generated artifactを総合判定すること
- #192または後継のCI管理: Actions再開後に通常runnerでConformanceを継続再検証すること

actual Oracle Evidenceが得られるまで、table / view / column名やSQL本文をRunner側から推測・生成しません。

## 非目標

- Oracle driver libraryの最終選定
- DSN / credential / Connection Pool / HA設計
- Oracle SQL本文や物理object名の確定
- Semantic mappingの再実装
- SQL文字列解析によるread-only判定

read-onlyの実効的な保証は、`ISSUE-000022`でcredential / session / transaction等の実際の強制境界として確認します。

## 関連

- GitHub Issue #179 / #192 / #214 / #222 / #224 / #273
- PR #225 / #285
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000020-contract-query-runtime-instance-identity.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `internal/driver/queryruntime/sql/executor.go`
- `internal/driver/queryruntime/oracle/runner.go`
- `internal/realization/contract_query_sql.go`
