# reference-order-migration-dual-db

技術検証用サンプル[`order-migration-dual-db`](../../reference/order-migration-dual-db/README.md)の工程1〜6を、PostgreSQLの三つのインスタンス(旧DB、新しい受注の主DB、在庫の参照系DB)で実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.10節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#410-旧受注システムからの移行と並走二つのdbをまたぐ引当order-migration-dual-db新設)です。

この実装はサンプル固有の業務処理で、`internal/`の基盤には入れていません。区切りの再開地点、DBをまたぐ確定境界、補償、変更記録の取込み、項目ごとの正本、名前の分割の段階、外部への送信の冪等性、切替の関門、逆同期は、基盤に宣言の手段がないため、ここで最小限に作りました(後述の「基盤の不足」)。

## 工程と受け渡し

| 工程 | 関数(CLIの命令) | 受け取るもの | 渡すもの |
| --- | --- | --- | --- |
| 1 旧DBの観測と移行計画 | `MakePlan`(`plan`) | 旧DBの受注残(`omdd.legacy_order`、`omdd.legacy_order_line`)と、移行用に用意した品目コードの対応表(`omdd.legacy_item_code_map`) | 計画のJSON。件数、明細数、金額合計、月別の金額合計、行ハッシュの集合の照合値、区切り(キー範囲)、照合の規則、品目コードの対応表、既知の差分の宣言、計画の照合値(`digest`) |
| 2 全件の写し | `TakeSnapshot`(`snapshot`) | 工程1の計画 | 主DBの写しの表(`stage_*`、得意先マスタを含む)と写しの記録。記録には計画の照合値、PostgreSQLのsnapshot(`pg_current_snapshot()`)、同じ文で読んだWALの位置、変更記録の開始点(論理レプリケーションスロットの名前と一貫点)、写しの照合値自身の値(`snapshotDigest`)を入れる |
| 3 区切りごとの移行と二つのDBへの引当 | `Migrate`、`Dispatch`、`Compensate`(`migrate`、`dispatch`、`run`) | 工程2の写しと記録、工程1の計画 | 主DBの受注残、効果記録(アウトボックス)、区切りごとの再開地点(`mig_checkpoint`)、参照系DBの引当、補償の記録。一対多の品目コードを含む受注残は移さず、保留の待ち行列(`cdc_held`)へ入れる |
| 3の確認 | `Status`、`Reconcile`(`status`、`reconcile`) | 計画 | 受注残ごとの二つのDBの状態の分類と、計画の規則による写しと主DBの照合の結果 |
| 4 並走中の変更の反映 | `ApplyChanges`(`apply-changes`) | 工程2のスロットと写しの時点、工程3の再開地点(全区切りの完了が条件) | 旧DBの取引ごとの処理の記録(`cdc_tx`)、反映した受注残と得意先、保留した変更(`cdc_held`)、マスタの衝突(`master_conflict`) |
| 4 マスタの正本の切替 | `SwitchFieldOwner`(`switch-owner`)、`UpdateCustomer`、`ResolveMasterConflict` | 計画 | 項目ごとの正本(`master_field_owner`)とその履歴、逆同期の待ち行列(`reverse_outbox`) |
| 4 得意先の拡張 | `ExpandNameSplit`(`expand-name`)、`CheckContractNameSplit`(`contract-name`) | 計画 | 姓と名の列、段階の記録(`schema_change`、`schema_change_event`)、書き手の版(`schema_client`)、分割できない名前の一覧(`schema_issue`) |
| 4 請求システムへの送信 | `SendBilling`、`ReconcileBilling`(`send-billing`、`reconcile-billing`) | 得意先の変更と同じ取引で書いた送信の記録(`billing_outbox`) | 冪等キーごとの送信の結果、結果不明の照会の結果 |
| 5 照合と切替の関門 | `Gate`(`gate`) | 工程3・4の結果、工程1の照合の規則と既知の差分 | 関門の記録(`cutover_gate`)。計画の照合値、写しの照合値、照合の結果、待ち行列の件数、切替を止めた理由、関門の照合値。通過した場合だけ、受注の正本と得意先の全項目の正本を主DBへ移す |
| 6 逆同期 | `ReverseSync`(`reverse-sync`)、`PlaceOrder`、`UpdateOrderStatus` | 関門を通過した主DB | 旧DBへの書込みと受領の記録(`legacy_reverse_applied`)。受領の記録は計画の照合値と関門の照合値を持つ |
| 6 巻戻しの訓練 | `RollbackDrill`(`rollback`) | 関門の記録、逆同期の待ち行列 | 訓練の記録(`rollback_drill`)。計画の照合値、関門の照合値、旧DBと主DBの照合の結果。差分がなければ、受注と得意先の正本を旧DBへ戻す |

- 計画の照合値は、計画の`digest`以外の全項目のSHA-256です。工程2は主DBに計画を1件だけ記録し、工程3以降は受け取った計画の照合値が記録と一致しなければ`plan digest mismatch`で止まります。区切りの再開地点、効果記録、受注残の各行、変更記録の処理の記録、逆同期の待ち行列、関門の記録、旧DB側の逆同期の受領、巻戻しの訓練の記録も同じ照合値を持ちます。
- 区切りは、キーをバイト順(`COLLATE "C"`)で並べた連続する範囲です。最初の区切りは下限なし、最後の区切りは上限なしで、工程1の後に追加されたキーもどれか一つの区切りに入ります。
- 行ハッシュは、受注と明細の項目を長さ付きで連結したSHA-256です(`sha256-canonical-order-v1`)。金額は`numeric`を文字列で読み、整数の円として合計します。
- 品目コードの対応表は計画に含まれ、計画の照合値の対象です。対応表があると、計画は既知の差分`item-code:primary-codes-compared-as-legacy-codes-via-plan-itemCodeMap`を宣言します。照合は、主DBの品目コードを対応表で旧コードへ戻してから行います。新コードから旧コードへの対応は必ず一つに決まります。宣言がなければ変換せずに照合し、差分になります。

## DBをまたぐ手順(工程3)

1. 主DBの1トランザクションで、一つの区切りの受注残と明細、受注残ごとの効果記録(`reserve-stock`)、区切りの再開地点を書きます。受注残の引当の状態は`awaiting`です。品目コードは計画の対応表で変換します。旧コード一つに新コードが複数ある受注残は推測で埋めず、移さずに保留します。
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

## 並走中の変更の反映(工程4、P-MIG-02)

### 変更の捕捉の方式

PostgreSQLの論理デコーディング(出力プラグイン`test_decoding`、SQLのスロット関数)を使います。旧DBには`wal_level=logical`が必要です。トリガーと変更記録の表は使いません。

選んだ理由は、旧DBに対する更新のうち、トリガーを通らないものも捕捉できるからです。旧DBには旧システム自身のトリガーによる変更記録(`omdd.legacy_change_log`)を置き、試験では`session_replication_role = replica`でトリガーを止めた一括更新を行います。この一括更新はトリガーの記録に1行も残りませんが、論理デコーディングでは全行を捕捉して主DBへ反映します。トリガーの方式を選ぶと、この種の更新は照合で差分として見つけるまで分かりません。代わりに、旧DBのWAL設定とスロットの管理(スロットが残る間はWALが消えない)が必要になります。設計文書は旧DBの代役をSQLiteとしてトリガーの方式を想定していますが、ここでは旧DBもPostgreSQLなので、その想定は確かめていません。

### 写しの時点との対応(工程2の位置の扱い)

工程2は、写しを取る前にスロット(`tei_omdd_cdc`)を作り、その一貫点を写しの記録に入れます。一貫点の後に確定した取引だけがデコードされるので、写しに含まれない取引はすべて一貫点の後に確定しています。デコードした取引を写しに含めるかどうかは、その取引のxidを写しの`pg_snapshot`で判定して決めます(`xmin`未満は含む、`xmax`以上は含まない、その間は実行中の一覧になければ含む)。写しに含まれる取引は`skipped-visible-in-snapshot`として記録だけします。

写しの記録にある、同じ文で読んだWALの位置は従来どおり近似値で、何の判断にも使いません。記録の`exactness`と`changeFeed.exactness`にそう書いています。複製用の接続でスロットを作るときの`EXPORT_SNAPSHOT`を使えば一貫点と写しの時点が一致しますが、ここではSQLの関数で作ったスロットとxidの判定で同じ結果を得ています。

### 反映の手順

1. 工程3の全区切りに再開地点がなければ止まります(`ErrStep3Incomplete`)。変更はスロットに残ります。最初の呼出しで、写した得意先マスタを主DBへ入れ、全項目の正本を旧DBとします。
2. スロットから取引を読み(`pg_logical_slot_peek_changes`)、旧DBの取引ごとに主DBの1トランザクションで反映し、同じトランザクションで`cdc_tx`へ確定の位置とxidと処理の結果を記録します。`cdc_tx`の主キーは確定の位置なので、同じ取引を二度処理できません。
3. 主DBの確定の後にスロットを進めます(`pg_replication_slot_advance`)。間で強制終了すると、次の呼出しは処理済みの取引をもう一度読みますが、`cdc_tx`にある位置以下の取引は飛ばします。
4. 逆同期で旧DBに書いた取引(`omdd.legacy_sync_marker`に1行書く)は送り返さず、得意先の基準値の更新だけを行います(`echo-of-reverse-sync`)。主DB側では、変更の反映が書いた変更を逆同期の待ち行列へ入れません(セッション変数`omdd.origin`)。

反映せずに保留する変更と、その理由は次のとおりです。保留は`cdc_held`に残り、関門を止めます。

| 理由 | 内容 |
| --- | --- |
| `one-to-many-item-code` | 新コードが複数ある旧コード。人の判断(`ResolveHeldOrder`、判断は`code_decision`に残る)の後、旧DBの現在の受注を読み直して入れる |
| `order-already-held` | 保留中の受注残への後続の変更 |
| `quantity-or-item-changed-on-migrated-order`、`line-added-to-migrated-order`、`line-deleted-from-migrated-order`、`order-deleted` | 引当済みの受注残の数量・品目・明細の変更と削除。引当のやり直しはこのサンプルにない |
| `customer-deleted` | 得意先の削除 |
| `truncate-not-applicable`、`item-code-map-changed-after-plan`、`unknown-legacy-table` | 行として反映できない変更、計画の後の対応表の変更、想定外の表 |
| `legacy-write-after-cutover` | 関門の通過後(正本が主DBの間)に旧DBへ書かれた変更 |

単価と金額だけの変更(一括の単価改定など)は、引当に影響しないので反映します。

### 捕捉できない変更の扱い

論理デコーディングでも、DDL、スロットを作る前の変更、スロットを失った後の変更は捕捉できません。スロットを作る前の変更は写しに含まれます。DDLは捕捉せず、表の構造が変われば反映が失敗して止まります。スロットがなくなっていれば`ErrChangeFeedLost`で止まり、関門の照合で欠けた行が差分として残ります。該当するキーの写し直しは実装していません。

## 得意先マスタの正本と衝突(工程4、P-INT-03)

得意先の項目(`name`、`address`、`credit_limit`)ごとに正本を持ちます。最初は全項目が旧DBで、`switch-owner`で項目ごとに主DBへ移します。主DBの新しい業務処理(`UpdateCustomer`)は、主DBが正本でない項目への書込みを拒否します(`ErrNotOwner`)。

旧DBの変更は、変更記録から届いた得意先の行を、最後に見た旧DBの値(`customer_legacy_base`)と比べて、変わった項目ごとに次の規則で扱います。

| 正本 | 主DB側も基準値から変えていた | 結果 |
| --- | --- | --- |
| 旧DB | どちらでも | 主DBへ反映する。主DB側に独自の値があれば`owner-wins:legacy`として記録する |
| 主DB | 変えていた | 主DBの値を残す(`owner-wins:primary`)。逆同期が旧DBの値を上書きする |
| 主DB | 変えていない | 正本でない側の単独の書込みで、どちらが正しいかを規則では決められない。待ち行列へ回す(`queued:non-owner-write-by-legacy`)。主DBの値は変えない |
| 宣言なし | | 待ち行列へ回す(`queued:no-declared-owner`) |

待ち行列は人が解決します(`ResolveMasterConflict`で旧DBの値を採るか、主DBの値を残す)。どちらの場合も、逆同期で旧DBを主DBの値にそろえます。主DBが正本の項目の変更は、同じトランザクションで逆同期の待ち行列へ入ります。

品目コードの一対多の対応は、工程3と工程4の両方で保留します。一対一の対応は変換し、対応表にないコードはそのまま移します。

## 姓と名への分割(工程4、P-MIG-03)

主DBの得意先は、最初は氏名(`full_name`)だけを持ちます。`expand-name`は、拡張の段階で姓と名の列とトリガーを加え、移送の段階で既存の行を埋めます。

- 旧形式の書込み(氏名だけを変える)は、トリガーが空白(半角または全角)で姓と名に分けます。分けられない氏名は推測で埋めず、姓と名を空値にして`schema_issue`に記録します。
- 新形式の書込み(姓と名だけを変える)は、トリガーが氏名を組み立てます。両方を食い違う値で書くと拒否します。
- 旧形式の書き手として、変更記録の反映(`legacy-cdc`、旧システム自身)を登録します。試験では、主DBへ直接書く旧版の連携(`old-integration`)も登録します。
- `contract-name`は縮約(氏名の列の削除)の可否を判定します。旧形式の書き手が一つでも有効な間、分割できない氏名が残る間、移送の段階を終えていない間は拒否し、拒否を記録します。旧システムは巻戻しの訓練の後も旧形式の書き手として残るので、このサンプルでは縮約に到達しません。列の削除そのものは実装していません。

## 請求システムへの送信(工程4、P-TX-02)

得意先の氏名、住所、与信限度の変更は、どの経路の変更でも、同じトランザクションで`billing_outbox`に記録します。冪等キーは`billing:<得意先>:<連番>`です。

1. `send-billing`は、未送信の記録を連番の順に、冪等キーを付けて送ります。タイムアウトや応答の喪失は結果不明(`unknown`)として記録し、その回の送信を止めます。結果不明の記録がある間は、後の記録を送りません。
2. `reconcile-billing`は、結果不明の記録を冪等キーで請求システムに照会します。適用済みなら`delivered`(`confirmed_by=query`)、未受信なら未送信へ戻します。
3. 同じキーを再送しても、請求システムは1回だけ適用し、同じ適用番号を返します。同じキーで内容が違う要求は拒否します。

請求システムは、試験の中でHTTPの模擬(`BillingMock`)として動かします。適用の後に応答を返さない、適用の前に応答を返さない、の二つの遅延を注入できます。

## 照合と切替の関門(工程5)

`gate`は、次の順に確かめます。一つでも残れば切替を拒否し、拒否の理由と照合の結果を記録します。

1. 変更記録を最後まで反映する。
2. 参照系DBを観測し、全受注残が`both-committed`か`compensated`であり、二重の引当がない。
3. 保留した変更、待ち行列の衝突、分割できない氏名、請求の未確定、逆同期の未配送が0件。
4. 旧DBと主DBを、計画の照合の規則(件数、明細数、合計、月別合計、鍵の集合、行ハッシュ、集合の照合値)と、宣言した既知の差分だけを適用して照合し、差分がない。得意先は項目ごとに一致する。

通過した場合は、関門の記録と同じトランザクションで、受注の正本と得意先の全項目の正本を主DBへ移します。関門は旧DBへの書込みを止めません。関門を通った後に旧DBへ書かれた変更は、変更記録の反映が`legacy-write-after-cutover`として保留します。

## 逆同期と巻戻しの訓練(工程6)

- 関門の通過後、主DBの受注残と明細の変更は、トリガーで逆同期の待ち行列に入ります。新しい業務処理(`PlaceOrder`、`UpdateOrderStatus`)は、主DBが正本の間だけ書けます。
- `reverse-sync`は、待ち行列をキーごとにまとめ、旧DBの1トランザクションで主DBの現在の状態を書きます。同じトランザクションで目印の行と、待ち行列の連番ごとの受領(計画と関門の照合値付き)を書きます。品目コードは対応表で旧コードへ戻します。旧DBの確定の後に主DBの待ち行列を配送済みにします。間で強制終了すると、次の回は受領が揃ったまとまりを書き直しません。
- `rollback`は、主DBへの書込みを止め(`rolling-back`)、逆同期を最後まで配送し、変更記録を反映してから、旧DBと主DBを照合します。差分、主DBにだけある受注残、待ち行列のどれかが残れば訓練は不合格で、正本を戻しません。合格すれば受注と得意先の全項目の正本を旧DBへ戻します。その後の旧DBへの書込みは、変更記録の反映が再び主DBへ反映します。

## 失敗の注入

- 強制終了: 環境変数`TEI_OMDD_CRASH_AT=<注入点>:<n>`を指定すると、プロセスはその注入点に`n`回目に到達したとき、自分自身をkillします(UnixではSIGKILL)。後始末は実行されず、未確定のトランザクションはDB側で破棄されます。注入点は`chunk-mid`(区切りの途中)、`after-primary-commit`(主DBの確定の直後)、`before-send`(配送の直前)、`after-inventory-commit`(参照系の確定の直後、主DBへの記録の前)、`before-compensation`(拒否の記録の後、補償の前)、`after-change-applied`(旧DBの取引を主DBへ反映して記録した後、スロットを進める前)、`after-reverse-legacy-commit`(逆同期を旧DBで確定した後、主DBの待ち行列を配送済みにする前)です。試験では、試験バイナリをCLIとして別プロセスで起動し、これをkillさせます。
- 参照系DBの停止と遅延: 試験は、検証用スクリプトが作ったコンテナー(名前が`tei-omdd-`で始まるもの)だけを`docker pause`(接続は受け付けるが応答しない)と`docker stop`(接続を拒否する)で止めます。配送の途中で止める場合は、試験の内部の注入点(`beforeSendHook`)を使います。
- 在庫不足: 参照系の在庫を少なくして、引当の途中で不足を起こします。
- 1円の誤差: 主DBの明細1行の金額を1円増やします。
- 並走中の書込み: 試験は、別の接続で旧DBへ受注の追加と状態の変更を書き続けながら、写しと移行を行います。写しの直前には、試験の内部の注入点(`afterSlotHook`)で、スロットの作成の後に確定する取引と、写しの時点で実行中の取引を一つずつ作ります。
- トリガーを通らない一括更新: `session_replication_role = replica`で、旧DBの明細の単価と金額を一括で変えます。
- 請求システムの遅延: 模擬の請求システムに、適用の後または前で応答を返さない遅延を入れます。

## 実行

単体試験(DB不要):

```sh
go test ./tools/reference-order-migration-dual-db/ -count=1
```

実DBの試験は、DSNがなければskipします。skipは合格の証拠になりません。WSL2などGoとDockerがある環境で、リポジトリのルートから次を実行します。

```sh
python3 tools/verify-order-migration-dual-db.py --out /tmp/omdd-receipt
```

スクリプトは、`postgres:16-alpine`のコンテナーを三つ(`tei-omdd-<乱数>-legacy`、`-primary`、`-inventory`)作り、旧DBのコンテナーだけを`wal_level=logical`で起動します。専用のDB(`tei_omdd_legacy`、`tei_omdd_primary`、`tei_omdd_inventory`)を作ってから、パッケージの全試験を`go test -json`で実行します。終了時に、作った三つのコンテナーだけを削除します。全試験がPASSし、skipが0件の場合だけ成功です。受領記録(`receipt.json`)には、コミット、作業ツリーが清浄か、Goとサーバーの版、各インスタンスの`wal_level`、試験ごとの結果、試験が記録した観測値(`observed.jsonl`)を残します。出力先はリポジトリの外に限ります。

試験は、三つのDSNが別々のインスタンス(`pg_control_system()`の`system_identifier`が異なる)であることと、接続先のDB名が上の専用名であることを確かめてから、スキーマ`omdd`を作り直し、旧DBのスロット`tei_omdd_cdc`を削除します。

CLIは、同じ環境変数(`TEI_OMDD_LEGACY_DSN`、`TEI_OMDD_PRIMARY_DSN`、`TEI_OMDD_INVENTORY_DSN`)で接続先を受け取ります。

```sh
CLI="go run ./tools/reference-order-migration-dual-db/cmd/tei-order-migration"
$CLI plan -chunk-size 20 -out /tmp/plan.json
$CLI snapshot -plan /tmp/plan.json
$CLI run -plan /tmp/plan.json
$CLI status -plan /tmp/plan.json
$CLI reconcile -plan /tmp/plan.json
$CLI apply-changes -plan /tmp/plan.json
$CLI switch-owner -plan /tmp/plan.json -field credit_limit -owner primary
$CLI expand-name -plan /tmp/plan.json
$CLI contract-name -plan /tmp/plan.json
$CLI send-billing -plan /tmp/plan.json -billing-url http://127.0.0.1:8080
$CLI reconcile-billing -plan /tmp/plan.json -billing-url http://127.0.0.1:8080
$CLI reverse-sync -plan /tmp/plan.json
$CLI gate -plan /tmp/plan.json
$CLI rollback -plan /tmp/plan.json
```

`reconcile`、`gate`、`rollback`は、不合格のとき終了コード3を返します。`contract-name`は拒否のとき終了コード1を返します。請求システムは試験の模擬だけで、単独で起動する命令はありません。

## 実DBの試験が確かめること

| 試験 | 要件 | 確かめること |
| --- | --- | --- |
| `TestPostgresPMIG01ResumeAfterKillsAndDetectOneYen` | P-MIG-01 | 区切りの途中でプロセスを10回killして再開する。毎回、受注残の行数が再開地点の合計と一致し、killした区切りの行が残らない。最後に件数、明細数、合計、月別合計、行ハッシュが一致する。1行に1円を足すと、合計、月別合計、その行の行ハッシュ、集合の照合値が不一致になり、件数の規則は不一致にならない |
| `TestPostgresPTX01KillAroundDelivery` | P-TX-01 | 主DBの確定の直後、配送の直前、参照系の確定の直後でkillする。途中の状態を`undetermined:*`として記録し、再開後は受注残の件数、参照系が適用した効果の件数、引当の明細数と数量、在庫の引当数が一致する。参照系の確定の直後にkillした1件は、再配送で重複として排除される |
| `TestPostgresPTX03ShortageConvergesBothOrCompensated` | P-TX-03 | 在庫不足で拒否された受注残を、補償の前にkillすると`undetermined:rejected-compensation-pending`として記録する。再開後は全件が`both-committed`か`compensated`になり、補償は1件に1回だけ記録される。在庫を増やして補償済みの効果を再配送しても引き当てない |
| `TestPostgresPDB02InventoryStopBoundedAndResolved` | P-DB-02 | 参照系を応答なし(pause)、配送の途中で応答なし、停止(stop)にする。主DB側の処理が上限時間内に戻り、全区切りの移行が進み、引当待ちが全件理由付きで記録される。復旧後に全件が`both-committed`になる |
| `TestPostgresThroughSlice1` | 通し(工程1〜3) | 工程1〜3をCLIの別プロセスで順に実行し、kill、在庫不足、参照系の停止を挟む。計画の照合値が写しの記録、再開地点、効果記録、受注残、照合の結果まで同じ値で引き継がれる。区切りの大きさを変えて計画を作り直すと、`migrate`、`reconcile`、`status`が不一致で止まる。計画のファイルを書き換えると、照合値の再計算で止まる |
| `TestPostgresPMIG02ChangeFeedNoLossNoDuplicate` | P-MIG-02 | 旧DBへ書き続けながら写しと移行を行い、トリガーを止めた一括更新を混ぜる。スロットの作成の後で写しの前に確定した取引は写しに入り、変更記録では`skipped-visible-in-snapshot`になる。写しの時点で実行中だった取引は変更記録から入る。3件目の取引の反映の後にkillしても、再開後に同じ取引を二度処理しない(xidの重複0件)。一括更新はトリガーの記録に0行で、変更記録では全行が反映される。写しに入った一対多の品目コードの受注残は工程3で保留され、判断の後に入る。最後に旧DBと主DBの受注が全規則で一致し、新しい受注残の引当が全件`both-committed`になる |
| `TestPostgresPINT03FieldOwnersAndCodeMap` | P-INT-03 | 同じ得意先の住所を旧DBで、与信限度を主DBで同時に変えると、両方が両側に届く。主DBが正本でない項目の書込みは拒否される。与信限度を両側で変えると主DBの値が残り(`owner-wins:primary`)、旧DBだけが変えると待ち行列に入る。人の判断の後、得意先の差分が0件になる。一対一の品目コードは変換され、一対多の品目コードの受注残は保留され(後続の変更も保留)、照合で鍵の集合の差分として見える。対応表にないコードの判断は拒否され、正しい判断の後に照合が通る。既知の差分を宣言しない計画では、変換したコードが行ハッシュの差分になる |
| `TestPostgresPMIG03ExpandWithOldWritersContractRefused` | P-MIG-03 | 拡張と移送の後、旧DBの旧形式の書込み(半角と全角の空白)と旧版の連携の直接の書込みで、姓と名が埋まる。分割できない氏名は空値のまま記録される。縮約は、旧形式の書き手と未解決の氏名がある間は拒否され、旧版の連携を退役させても旧システムが残る間は拒否される。旧形式の書き手を全部無効にした場合だけ許される(試験ではロールバックする)。最後に全行で氏名と姓・名が一致し、旧DBとも一致する |
| `TestPostgresPTX02BillingTimeoutResendAppliedOnce` | P-TX-02 | 適用の後に応答しない請求システムへの送信が結果不明になり、後の送信が止まる。照会で適用済みと確定する。適用の前に応答しない場合は照会で未受信と分かり、再送される。送信済みの記録を未送信に戻して再送しても、請求システムは同じキーを再適用しない。適用は記録の件数と同じで、キーごとに1回 |
| `TestPostgresThroughSteps1To6` | 通し(工程1〜6) | 工程1〜6をCLIの別プロセスで順に実行する。旧DBへの書込みを続けながら写しと移行を行い、移行と変更の反映でkillを挟む。項目の正本の切替、拡張、縮約の拒否、請求の結果不明と照会を行う。関門は、保留した受注、待ち行列の衝突、照合の差分がある間は拒否し、正本を移さない。判断の後に関門を通る。作り直した計画は関門で止まる。切替後に主DBで受注の追加、状態の変更、得意先の変更を行い、逆同期の途中でkillしても重複して書かない。巻戻しの訓練で、旧DBと主DBの差分が0件、主DBにだけある受注残が0件になり、旧DBの受領に関門の照合値が残る。巻戻しの後は主DBへの受注の書込みが拒否され、旧DBの書込みが再び主DBへ反映される。計画の照合値が、関門、変更記録の処理の記録、逆同期の待ち行列と受領、巻戻しの訓練の記録まで同じ値で引き継がれる |

## 基盤の不足

このサンプルで、基盤にないためサンプル側で作ったものです。4.10節の「基盤への改善圧力と現状」に対応します。

- P-MIG-01: 区切り、再開地点、照合の規則を宣言する手段がありません。計画のJSON、`mig_checkpoint`、`Compare`はサンプル固有の形式です。
- P-TX-01、P-TX-03: DBごとの確定境界、効果記録、受け手の重複排除、補償の宣言がありません。`effect_outbox`、`applied_effect`、`compensation`と、`Status`の分類はサンプル固有です。[業務副作用の設計](../../plan/design/BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md)の2・6・7・8章に沿わせていますが、`reference/effect-fence`と`tools/reference-capability-mutation-protocol`はメモリ内の参照実装で、DBをまたぐ流れに使えないため使っていません。準備済みトランザクション(2相コミット)には頼っていません。
- P-DB-02: 参照先が止まったときの方針(上限時間、保留、要再確認の記録)を宣言する手段がありません。上限時間と記録の値は`DispatchOptions`と`effect_outbox.last_outcome`に直接持っています。
- P-MIG-02: 写しの時点と変更記録の位置を一組で記録する手段、変更記録の取込み、逆同期、正本の切替の宣言がありません。スロットの作成、xidによる写しとの対応づけ、`test_decoding`の出力の解釈、`cdc_tx`、`cdc_held`、目印の表による送り返しの防止、`reverse_outbox`と旧DBの受領、`mig_authority`はサンプル固有です。`test_decoding`は検証用の出力プラグインで、本番での利用を想定したものではありません。捕捉できない変更(DDL、スロットの喪失)の検出は止まるだけで、写し直しの手段はありません。
- P-INT-03: マスタの項目ごとの正本、衝突の待ち行列、有効期間と多重度を持つコード変換表を宣言する手段がありません。`master_field_owner`、`customer_legacy_base`、`master_conflict`、計画の`itemCodeMap`、`code_decision`はサンプル固有です。変換表に有効期間はありません。
- P-MIG-03: 拡張、移送、縮約の段階と、旧形式の書込みを新しい列へ写す規則を宣言する手段がありません。トリガー`customer_name_sync`、`schema_change`、`schema_client`、`schema_issue`はサンプル固有です。旧形式の書き手は自己申告の登録で、実際の書込みの版を観測していません。
- P-TX-02: 外部への送信に効果識別子を渡す手段と、結果不明を照会で確定させる手順の宣言がありません。`billing_outbox`、冪等キーの形式、照会の手順、模擬の請求システムはサンプル固有です。
- 照合と切替の関門: 照合の規則、既知の差分、待ち行列の件数を入力にして切替を止める関門を宣言する手段がありません。`Gate`と`cutover_gate`、`RollbackDrill`と`rollback_drill`はサンプル固有です。既知の差分は、品目コードの変換一つだけを名前で照合しています。

## 範囲外

- P-DB-01(読取り専用の複製の鮮度)。主DBのストリーミング複製の構築が必要で、工程4〜6の主筋を優先したため行っていません。判断に使う読取りは、どれも主DBへの直接の読取りです。
- 異種DBを前提とする要件(P-DB-03、P-DB-04)。三つのDBはどれもPostgreSQLで、この結果は異種のDBで成立した証拠になりません。計画の既知の差分は品目コードの変換だけで、方言の差は宣言していません。旧DBをSQLiteとしたときのトリガーによる変更記録の方式も確かめていません。
- 在庫残の旧と新の照合。旧DBは在庫を持たないため、照合していません。
- 縮約(氏名の列の削除)の実行、引当済みの受注残の数量・品目の変更に伴う引当のやり直し、受注と得意先の削除の反映と逆同期、保留した変更のうち一対多の品目コード以外の解決手順、逆同期での得意先の追加。
- 関門と旧DBへの書込みの停止の連動。関門は旧DBの書込みを止めず、関門の後の書込みは保留として扱います。
- 同じ機械の上の合成データで確かめた結果であり、実際の旧システムや本番の規模での成立は確かめていません。中継の処理、変更の反映、逆同期を複数同時に動かす場合の排他も確かめていません。
