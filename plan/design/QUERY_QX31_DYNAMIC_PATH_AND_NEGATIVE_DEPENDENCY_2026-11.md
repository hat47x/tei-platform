# QX-31：Relation path再束縛・否定的集合依存の実SQL圧力試験（2026-11-02）

**位置付け**：[#2387 Query Explain](https://github.com/hat47x/tei-platform/issues/2387)／[Draft #2388](https://github.com/hat47x/tei-platform/pull/2388)の読取り設計に追加する**限定的な実SQLite反証資料**。Native SQLはSQL由来の正本、DerivedIRは説明用導出であり、TEI専用Host／新Reactive Engine／新Proof Registry／新SQL Runnerは提案しない。Ownerによる業務意味受理・製品E2E・D2の実行範囲拡大は対象外。

## 1. 他チャットとの接合で残る本質的な欠落

- [#2582 DC-01〜12](https://github.com/hat47x/tei-platform/issues/2582#issuecomment-6579762087)は合成Pythonで**途中Relation pathの付替え**と**現在0件の集合に残る否定依存**を検討した。今回、同じ型の問題が**実SQL JOIN＋相関NOT EXISTS**で成立するかを、独立した人工DBと手書きExpectedで検証する。別レーンのPython 12件と今回のSQL 12件を製品E2E 24件とは呼ばない。
- [#2624 Function Context Mapping](https://github.com/hat47x/tei-platform/pull/2624)の**表示Mapping**とRuleが必要とする**Input/Relation path**は別。QX-25（SQLite Auth）、QX-27（Go DerivedIR）もOutputField入力とCondition/Selectionを区別するが、実Readの時点でPathがどの対象に解決されたかは別証拠。
- [#2627 Provider/Read Cohort](https://github.com/hat47x/tei-platform/pull/2627)とQX-26は、各Pageの実Scope/時点/Rowset＋Field completenessが異なり得ることを確認。NOT EXISTSの一致0件は、**Scope内の否定を主張するためのRead Witness**があって初めて、その特定の時点・範囲に限定して利用できる。
- [#951顧客照会契約・Draft #2640](https://github.com/hat47x/tei-platform/pull/2640)は0件をSemantic Result・Provider Actual Acquisition・Consumerへの公開結果と分離し、0→1→0のABA／非単調Queryの撤回を補足。SQLが0件を返した事実から、業務拒否・実取得の全件性・不存在の永久保証・現在のAction資格を自動導出しない。

## 2. 固定した人工SQLとStatementの意味範囲

```sql
SELECT o.id
FROM orders o
JOIN customer c ON c.id = o.customer_ref
JOIN account a ON a.id = c.account_ref
WHERE o.id = 'O1'
  AND o.amount <= a.credit_limit
  AND NOT EXISTS (
    SELECT 1 FROM holds h
    WHERE h.order_id = o.id AND h.active = 1
  );
```

- 基本世界：`O1.amount=600`、`O1.customer_ref=C1`、`C1.account_ref=A1`、`A1.credit_limit=700`、Active Holdなし。実SQL出力はO1。
- 別世界：`C2→A2`、`A2.credit_limit=500`、対象外O2。O1の金額や旧A1 limitが同じでも途中の参照がC2/A2へ再束縛されるとO1は出力から消える。
- **これは架空のEligibility SQL**であり、正式な与信業務Rule、Ownerの受理済み判断、RLS／Policy／実Host Grantは存在しない。SQLのWHEREが0行にする理由（不成立、UNKNOWN、JOIN参照不在、実Provider障害、非公開等）は異なる。今回のfixtureは正常なSQLite実行ケースだけを扱う。

## 3. 実行済みReference（GitHub保存内容とローカル実行内容の一致を検証）

[**test_qx31_relation_path_negative_read.py**](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/fixtures/query-workbench-f0/test_qx31_relation_path_negative_read.py)

- Python **3.13.5／SQLite 3.46.1**で `python3 -m unittest -v test_qx31_relation_path_negative_read.py`：**12/12 PASS**。`python3 -m py_compile`：PASS。
- 実行したPython source **6298 bytes**。ローカル `git hash-object`とGitHub保存blobの一致：**`187277b93fb61e7e715c1f39fe9d2c362925567c`**。
- SQLiteの`set_authorizer(SQLITE_READ)`は**SQL準備時に要求された列参照**であり、実際に読み取った物理行数、Source粒度、サーバーRLS、他ProviderのScope／実行時Principal、Host認可を証明しない。

| Case | 変更・観測 | 設計上の反例／正例 |
| --- | --- | --- |
| 01 | O1→C1→A1(limit700)、Hold0件、O1を返す | **現在このSQLite世界に限った**正例。恒久的不存在や他ProviderのCompleteではない |
| 02 | O1.customerRef=C1→C2、amount600/旧A1=700不変 | Pathの再束縛でInputがA2.limit500へ移り、O1が消える。旧Valueだけ追うと過少失効 |
| 03 | C1.accountRef=A1→A2、O1.customerRef不変 | 中継Relationの再束縛でも結果が変わる。最終列の旧Field ID監視では不足 |
| 04 | Active Hold0→1 | `NOT EXISTS`成立行が撤回。**現在行0件でもPredicate/Scopeへの依存は残る** |
| 05 | 非Active Hold1→active1 | Predicate対象外の行追加ではO1維持、対象内へ変更するとO1撤回 |
| 06 | O2のHoldのみ増加 | 合成世界ではO1の結果に影響なし。全変更で全再取得する過大失効の対照 |
| 07 | 全Hold先頭1ページにO2のHoldだけ、O1のHoldは次に存在 | **部分ページで見えない≠Scoped NOT EXISTS成立**。全体評価ではO1が消える |
| 08 | Hold0→1→0で初期と最終のSQL結果が一致 | 値/Result同値から「間で変更が無かった」を導けない。ABAの歴史的Witnessが必要 |
| 09 | A2.limitも700としたうえでO1参照をC1→C2 | 同じ結果O1・同じ数値でも`(C1,A1)`→`(C2,A2)`でPath/対象資格は別 |
| 10 | A1.limit=NULL | `600<=NULL`はSQL UNKNOWN、行は不出力。Business falseを自動確定しない |
| 11 | O2のActive Holdだけ存在、Subqueryの`h.order_id=o.id`を落とした変異SQL | 正しいQueryはO1を返し、誤Queryは返さない。**否定的なPredicateのScope**が意味そのもの |
| 12 | sqlite3 Authorizerに列参照を記録 | 中継`customer_ref/account_ref`、条件`credit_limit`、否定集合の`holds.order_id/active`を要求。物理Read証拠ではない |

## 4. 結論：Value Lineageに「Row Membership/Path Closure」を混入しない

下記は**新Core Schemaの提案ではなく、既存Contract/Source/Provider/Consumer Ownerが保有する証拠を読むための比較票**。

| ConsumerのClaim | 必要な根拠／失効の条件 | それだけで保証できないこと |
| --- | --- | --- |
| **表示値が現在も同じ** | 対象・表示ValueのSource版／RowID／Result Slice／選択epoch／現在の読取資格 | RuleやNOT EXISTS条件が不変、非表示Fieldの開示許可 |
| **Rule／Validationをいま評価できる** | Query Predicate、Relation pathの各辺の実Binding、Current Input Field/NULL、Rowset/Field complete、Requested Scope、Rule版／適用対象 | Ownerによる業務意味受理、Commit権限 |
| **NOT EXISTSが現在成立** | 対象Scope内のCompleteな読取・有効な否定証人（read+change witness等）、Read Cohort/Principal/Policy/時点 | 別Provider／別Windowでも成り立つ、t0以降変化なし、未来も存在しない |
| **無関係変更で安全に再利用** | Consumer Claimに対する依存閉包、実変更範囲と観測完全性、対応Input/Path/Predicate一致 | Change通知が無いだけで実世界に変更がないこと |
| **結果を公開／Actionとして使える** | Purpose/Caller別Disclosure、Hostの現在Grant/対象Payload、実Provider/Effect Receipt | Raw SQL/IRを外部AIへ送ってよい、SQL評価成功が実Effect確定を意味する |

**静的と動的の対応**：SQL Workerの`ColumnRefs`、Go`OutputField.Inputs`／`RelNode.Inputs`／Expression、Function Contextの宣言されたRelation pathは、どの項目・条件が関連し得るかを表す材料。**実時点でPathがどのEntityに向いていたか、NOT EXISTSのScopeが実際に閉じたか、現在のPolicyが維持されているかは、実Provider側の独立Witnessが必要**である。手書き`analyzer.Result`から`Lowering=full`と表示できても、この独立Witnessを代替しない。

**過大失効と過少失効を両方避ける**：O1のRuleで無関係なO2 Hold追加まで一律失効させる根拠はないが、実変更通知の欠落、別Replicaでの差異、Snapshot/Policy不明なら「影響なし」という現在の保証もない。`unaffected-candidate`と`qualified-to-reuse`を別表示し、後者の条件をConsumer/Claimごとに確認する。

## 5. 既存Workstreamへの実装可能な引継ぎ（M0〜M3）

1. **M0 — #2387/QX-27（Query解析）**：実SQLGlot Workerからの`analyzer.Result`が`JOIN customer/account`のOccurrence/Path、`NOT EXISTS holds`のScope/Predicate、否定依存をどこまで保持するか観測。Go Synthetic characterizationと今回SQLiteを同じSQL byte/Schemaで比較し、未対応なら`unknown`。#1640のD2実行拒否を緩めない。
2. **M1 — #2506/#2624/#2586（Context/Rule）**：明示Function Context、実Selected O1、Ruleの条件入力（Account limit、Active Hold）、Source/Mapping版、Data SliceのScope・Field Presenceを一Attemptで照合。未取得・NULL・不完全・Provider拒否を同じBusiness falseへ落とさない。
3. **M2 — #2582/#2622/#2627（Reactive/Provider）**：実ProviderでRelation再束縛、集合への新規加入、ABA、Page部分取得、Replica切替・変更監視欠落に対する**過少失効／過大失効**を独立比較。Current ResultとHistorical Evidence、Result byteとRead Cohort/Scopeを別に評価。
4. **M3 — #2516/#1178/#2515/#2598/#1174（公開・作用）**：秘密のRule条件や否定集合の存在を説明へ無断公開せず、必要な判断情報のみ開示できるかを検証。現在Action対象/Host Authority/実Commit/ReceiptとSQLによる読取り成功を分離。

**NOT RUN／未受理**：TEI本体SQLGlot→Go→Function Context／Rule Runner、PostgreSQL/API Provider、Hostの権限変化、Windows/Studio/SHO/外部AIへの実送信、実Business Effect、Owner意味受理。合成世界の12PASSを製品合格へ合算しない。既存のSource正本と#1119 G0〜G4を維持する。
