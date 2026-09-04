# ISSUE-000021: Oracle Contract Queryの`database/sql` Runner境界を追加する

- 状態: In Progress
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

一方、SQL realizationを実際のDatabase接続へ渡す最下層の`sqlquery.Runner`には、Oracle向けの最小実装がありませんでした。ここでOracleの物理SchemaやSQLをRunner自身に持たせると、realizationとprovider adapterの責務が再び混ざります。

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

## 現在地

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

実装は`work/reference-data-access-convergence`へ統合済みです。

2026年9月3日現在、GitHub Actions workflowはRepositoryから明示的に削除され、収束線にも停止が取り込まれています。そのため、このConformanceは現時点では通常runnerによる実行Evidenceを持ちません。CI未実行をgreenとは扱わず、CIをこの作業線から独断で再有効化もしません。再開後に通常runnerで改めて検証します。

特定のOracle driver packageはまだ選定していません。実Oracle driverが`database/sql`のnamed argumentと対象SQLをどのように扱うかは、`ISSUE-000022`の物理Evidenceが揃った後のintegrationで確認します。

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
- [ ] CIが明示的に再有効化された後、通常runnerでConformanceが実行され、成功する
- [ ] 採用したOracle driverとのnamed parameter互換性を実接続で確認する
- [ ] 第一Referenceの実SQLでRunnerを通過するEvidenceを得る

## 非目標

- Oracle driver libraryの最終選定
- DSN / credential / Connection Pool / HA設計
- Oracle SQL本文や物理object名の確定
- Semantic mappingの再実装
- SQL文字列解析によるread-only判定

read-onlyの実効的な保証は、`ISSUE-000022`でcredential / session / transaction等の実際の強制境界として確認します。

## 関連

- GitHub Issue #179 / #192 / #214 / #222 / #224
- PR #225
- `plan/issue/000018-database-contract-query-execution.md`
- `plan/issue/000022-external-master-data-physical-query-evidence.md`
- `plan/issue/000020-contract-query-runtime-instance-identity.md`
- `internal/driver/queryruntime/sql/executor.go`
- `internal/driver/queryruntime/oracle/runner.go`
- `internal/realization/contract_query_sql.go`
