# reference-order-migration-dual-db

技術検証用サンプル[`order-migration-dual-db`](../../reference/order-migration-dual-db/README.md)の工程1〜3を、PostgreSQLの三つのインスタンス(旧DB、新しい受注の主DB、在庫の参照系DB)で実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.10節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#410-旧受注システムからの移行と並走二つのdbをまたぐ引当order-migration-dual-db新設)です。

この実装はサンプル固有の業務処理で、`internal/`の基盤には入れていません。区切りの再開地点、DBをまたぐ確定境界、補償は、基盤に宣言の手段がないため、ここで最小限に作りました(後述の「基盤の不足」)。

## 工程と受け渡し

| 工程 | 関数(CLIの命令) | 受け取るもの | 渡すもの |
| --- | --- | --- | --- |
| 1 旧DBの観測と移行計画 | `MakePlan`(`plan`) | 旧DBの受注残(`omdd.legacy_order`、`omdd.legacy_order_line`) | 計画のJSON。件数、明細数、金額合計、月別の金額合計、行ハッシュの集合の照合値、区切り(キー範囲)、照合の規則、計画の照合値(`digest`) |
| 2 全件の写し | `TakeSnapshot`(`snapshot`) | 工程1の計画 | 主DBの写しの表(`stage_*`)と写しの記録。記録には計画の照合値、PostgreSQLのsnapshot(`pg_current_snapshot()`)、同じ文で読んだWALの位置、写しの照合値、写しの照合値自身の値(`snapshotDigest`)を入れる |
| 3 区切りごとの移行と二つのDBへの引当 | `Migrate`、`Dispatch`、`Compensate`(`migrate`、`dispatch`、`run`) | 工程2の写しと記録、工程1の計画 | 主DBの受注残、効果記録(アウトボックス)、区切りごとの再開地点(`mig_checkpoint`)、参照系DBの引当、補償の記録 |
| 3の確認 | `Status`、`Reconcile`(`status`、`reconcile`) | 計画 | 受注残ごとの二つのDBの状態の分類と、計画の規則による照合の結果 |

- 計画の照合値は、計画の`digest`以外の全項目のSHA-256です。工程2は主DBに計画を1件だけ記録し、工程3以降は受け取った計画の照合値が記録と一致しなければ`plan digest mismatch`で止まります。区切りの再開地点、効果記録、受注残の各行、照合の結果も同じ照合値を持ちます。
- 区切りは、キーをバイト順(`COLLATE "C"`)で並べた連続する範囲です。最初の区切りは下限なし、最後の区切りは上限なしで、工程1の後に追加されたキーもどれか一つの区切りに入ります。
- 行ハッシュは、受注と明細の項目を長さ付きで連結したSHA-256です(`sha256-canonical-order-v1`)。金額は`numeric`を文字列で読み、整数の円として合計します。

## DBをまたぐ手順

1. 主DBの1トランザクションで、一つの区切りの受注残と明細、受注残ごとの効果記録(`reserve-stock`)、区切りの再開地点を書きます。受注残の引当の状態は`awaiting`です。
2. 中継の処理が未配送の効果記録を読み、参照系DBの1トランザクションで受注残1件の全明細を引き当てます。参照系は効果識別子で重複を排除し、在庫不足の拒否も効果識別子とともに記録します。拒否を記録した効果は、後で在庫が増えて再配送されても引き当てません。
3. 主DBで、配送の結果を効果記録と受注残に記録します(`allocated`または`rejected`)。
4. `rejected`の受注残は補償します。補償は独自の識別子を持つ別の操作として1回だけ記録し、受注残を`compensated`(在庫なしで保留)にします。移行したデータなので受注残の行は消しません。結果が分からない配送(`unknown`)は補償しません。

参照系DBへの接続と操作には上限時間(既定は各3秒)があり、接続できない場合や応答がない場合は、効果記録を未配送のまま残して理由を記録します(`inventory-unavailable`、`unknown`、`not-attempted-circuit-open`)。「在庫がない」と「参照系に聞けなかった」は区別して記録します。

`Status`は、受注残を次のように分類します。`undetermined:*`は、一方のDBだけで確定した状態です。成功にも失敗にも数えません。

| 分類 | 主DB | 参照系DB |
| --- | --- | --- |
| `both-committed` | `allocated` | 引当済み |
| `compensated` | `compensated` | 拒否を記録済み |
| `undetermined:primary-only-awaiting-allocation` | `awaiting` | 記録なし |
| `undetermined:inventory-reserved-primary-unacknowledged` | `awaiting` | 引当済み |
| `undetermined:inventory-rejected-primary-unacknowledged` | `awaiting` | 拒否を記録済み |
| `undetermined:rejected-compensation-pending` | `rejected` | 拒否を記録済み |
| `undetermined:inventory-unobserved` | 任意 | 参照系を観測できなかった |
| `inconsistent` | 上記以外の組合せ | |

## 失敗の注入

- 強制終了: 環境変数`TEI_OMDD_CRASH_AT=<注入点>:<n>`を指定すると、プロセスはその注入点に`n`回目に到達したとき、自分自身をkillします(UnixではSIGKILL)。後始末は実行されず、未確定のトランザクションはDB側で破棄されます。注入点は`chunk-mid`(区切りの途中)、`after-primary-commit`(主DBの確定の直後)、`before-send`(配送の直前)、`after-inventory-commit`(参照系の確定の直後、主DBへの記録の前)、`before-compensation`(拒否の記録の後、補償の前)です。試験では、試験バイナリをCLIとして別プロセスで起動し、これをkillさせます。
- 参照系DBの停止と遅延: 試験は、検証用スクリプトが作ったコンテナー(名前が`tei-omdd-`で始まるもの)だけを`docker pause`(接続は受け付けるが応答しない)と`docker stop`(接続を拒否する)で止めます。配送の途中で止める場合は、試験の内部の注入点(`beforeSendHook`)を使います。
- 在庫不足: 参照系の在庫を少なくして、引当の途中で不足を起こします。
- 1円の誤差: 主DBの明細1行の金額を1円増やします。

## 実行

単体試験(DB不要):

```sh
go test ./tools/reference-order-migration-dual-db/ -count=1
```

実DBの試験は、DSNがなければskipします。skipは合格の証拠になりません。WSL2などGoとDockerがある環境で、リポジトリのルートから次を実行します。

```sh
python3 tools/verify-order-migration-dual-db.py --out /tmp/omdd-receipt
```

スクリプトは、`postgres:16-alpine`のコンテナーを三つ(`tei-omdd-<乱数>-legacy`、`-primary`、`-inventory`)作り、専用のDB(`tei_omdd_legacy`、`tei_omdd_primary`、`tei_omdd_inventory`)を作ってから、パッケージの全試験を`go test -json`で実行します。終了時に、作った三つのコンテナーだけを削除します。全試験がPASSし、skipが0件の場合だけ成功です。受領記録(`receipt.json`)には、コミット、作業ツリーが清浄か、Goとサーバーの版、試験ごとの結果、試験が記録した観測値(`observed.jsonl`)を残します。出力先はリポジトリの外に限ります。

試験は、三つのDSNが別々のインスタンス(`pg_control_system()`の`system_identifier`が異なる)であることと、接続先のDB名が上の専用名であることを確かめてから、スキーマ`omdd`を作り直します。

CLIは、同じ環境変数(`TEI_OMDD_LEGACY_DSN`、`TEI_OMDD_PRIMARY_DSN`、`TEI_OMDD_INVENTORY_DSN`)で接続先を受け取ります。

```sh
go run ./tools/reference-order-migration-dual-db/cmd/tei-order-migration plan -chunk-size 20 -out /tmp/plan.json
go run ./tools/reference-order-migration-dual-db/cmd/tei-order-migration snapshot -plan /tmp/plan.json
go run ./tools/reference-order-migration-dual-db/cmd/tei-order-migration run -plan /tmp/plan.json
go run ./tools/reference-order-migration-dual-db/cmd/tei-order-migration status -plan /tmp/plan.json
go run ./tools/reference-order-migration-dual-db/cmd/tei-order-migration reconcile -plan /tmp/plan.json
```

## 実DBの試験が確かめること

| 試験 | 要件 | 確かめること |
| --- | --- | --- |
| `TestPostgresPMIG01ResumeAfterKillsAndDetectOneYen` | P-MIG-01 | 区切りの途中でプロセスを10回killして再開する。毎回、受注残の行数が再開地点の合計と一致し、killした区切りの行が残らない。最後に件数、明細数、合計、月別合計、行ハッシュが一致する。1行に1円を足すと、合計、月別合計、その行の行ハッシュ、集合の照合値が不一致になり、件数の規則は不一致にならない |
| `TestPostgresPTX01KillAroundDelivery` | P-TX-01 | 主DBの確定の直後、配送の直前、参照系の確定の直後でkillする。途中の状態を`undetermined:*`として記録し、再開後は受注残の件数、参照系が適用した効果の件数、引当の明細数と数量、在庫の引当数が一致する。参照系の確定の直後にkillした1件は、再配送で重複として排除される |
| `TestPostgresPTX03ShortageConvergesBothOrCompensated` | P-TX-03 | 在庫不足で拒否された受注残を、補償の前にkillすると`undetermined:rejected-compensation-pending`として記録する。再開後は全件が`both-committed`か`compensated`になり、補償は1件に1回だけ記録される。在庫を増やして補償済みの効果を再配送しても引き当てない |
| `TestPostgresPDB02InventoryStopBoundedAndResolved` | P-DB-02 | 参照系を応答なし(pause)、配送の途中で応答なし、停止(stop)にする。主DB側の処理が上限時間内に戻り、全区切りの移行が進み、引当待ちが全件理由付きで記録される。復旧後に全件が`both-committed`になる |
| `TestPostgresThroughSlice1` | 通し | 工程1〜3をCLIの別プロセスで順に実行し、kill、在庫不足、参照系の停止を挟む。計画の照合値が写しの記録、再開地点、効果記録、受注残、照合の結果まで同じ値で引き継がれる。区切りの大きさを変えて計画を作り直すと、`migrate`、`reconcile`、`status`が不一致で止まる。計画のファイルを書き換えると、照合値の再計算で止まる |

## 基盤の不足

このサンプルで、基盤にないためサンプル側で作ったものです。4.10節の「基盤への改善圧力と現状」に対応します。

- P-MIG-01: 区切り、再開地点、照合の規則を宣言する手段がありません。計画のJSON、`mig_checkpoint`、`Compare`はサンプル固有の形式です。
- P-TX-01、P-TX-03: DBごとの確定境界、効果記録、受け手の重複排除、補償の宣言がありません。`effect_outbox`、`applied_effect`、`compensation`と、`Status`の分類はサンプル固有です。[業務副作用の設計](../../plan/design/BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)の2・6・7・8章に沿わせていますが、`reference/effect-fence`と`tools/reference-capability-mutation-protocol`はメモリ内の参照実装で、DBをまたぐ流れに使えないため使っていません。準備済みトランザクション(2相コミット)には頼っていません。
- P-DB-02: 参照先が止まったときの方針(上限時間、保留、要再確認の記録)を宣言する手段がありません。上限時間と記録の値は`DispatchOptions`と`effect_outbox.last_outcome`に直接持っています。
- P-MIG-02(工程2の一部): 写しの時点と変更記録の位置を正確に対応づける手段がありません。WALの位置はsnapshotと同じ文で読んだ近似値で、`exactness`にそう記録しています。正確な対応には論理レプリケーションスロットの一貫点が必要で、工程4で扱います。

## 範囲外

- 工程4〜6(並走中の変更の反映、マスタの正本の切替、照合と切替の関門、逆同期と巻戻し)。
- 異種DBを前提とする要件(P-DB-03、P-DB-04)。三つのDBはどれもPostgreSQLで、この結果は異種のDBで成立した証拠になりません。計画の`knownDifferences`は空です。
- P-DB-01(読取りの複製の鮮度)、P-MIG-02、P-INT-03、P-MIG-03、P-TX-02。
- 同じ機械の上の合成データで確かめた結果であり、実際の旧システムや本番の規模での成立は確かめていません。中継の処理を複数同時に動かす場合の排他も確かめていません。
