# reference-month-end-close-correction

技術検証用サンプル[`month-end-close-correction`](../../reference/month-end-close-correction/README.md)の工程1〜3を、PostgreSQLの一つのインスタンス(専用のDB`tei_mecc`)で実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.11節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#411-月次締めと締め後に届く訂正の計上month-end-close-correction新設)です。工程4〜6はまだありません。

この実装はサンプル固有の業務処理で、`internal/`の基盤には入れていません。版つきの暦、ファイル仕様と取込み単位、ジョブの業務キー、貸与の永続化、期間の状態は、基盤に宣言の手段がないため、ここで最小限に作りました(後述の「基盤の不足」)。製品コードからは、`internal/resource`の`SupervisorOwnershipStore`のインターフェースだけを使っています。

## 工程と受け渡し

| 工程 | 関数(CLIの命令) | 受け取るもの | 渡すもの |
| --- | --- | --- | --- |
| 1 営業日カレンダーと締め日の確定 | `LoadCalendar`、`ReviseCalendar`、`Decide`(`calendar-load`、`calendar-revise`、`payable-load`、`decide`) | カレンダーの版、締めの規則(`testdata/close-rule.json`)、評価の基準時刻(明示の入力)、未払案件の名目の期日 | 締めの決定のJSON。対象月、カレンダーの識別子・版・照合値、地域と時間帯、評価の基準時刻、締め日、ファイルの締め時刻、ジョブの予定、支払期日、業務日付の規則、決定の照合値(`digest`)。前の決定があれば、改訂の影響(`calendar_impact`) |
| 2 締め時刻のファイル取込みと照合 | `ImportFile`、`ReceiveMessage`、`ReconcileShipments`、`Reemit`(`import`、`message`、`reconcile-shipments`、`reemit`) | 工程1の決定、取引先の固定長ファイル、ファイル仕様の宣言(`testdata/file-spec.json`)、模擬のメッセージ | 取込み単位の記録(`import_unit`。ファイルのSHA-256、送り手と連番、件数と合計、保留の件数、決定の照合値とカレンダーの版)、レコードごとの記録と保留の理由、拒否の記録(`import_rejection`)、出荷の事象と受領の記録、照合の記録(`shipment_reconciliation`) |
| 3 締めジョブの実行 | `RunClose`(`close`)、`PlaceEntry`、`AllocateEntry` | 工程1の決定、工程2の取込み単位(全件が同じ決定の照合値であること)、オンラインの受注と訂正 | 対象月を業務キーとする締めの実行記録(`close_run`。決定の照合値、カレンダーの版、取込み単位の照合値、締めの対象の時点、排他トークン)、区切りごとの記録と明細(`close_chunk`、`close_detail`)、締めの結果(`close_total`)、起動の記録(`close_attempt`)、貸与の記録(`lease_event`) |

- 決定の照合値は、決定の`digest`以外の全項目のSHA-256です。評価の基準時刻も含みます。工程2と3は、受け取った決定の照合値を再計算し、対象月の有効な決定(`close_month`)と一致しなければ`decision mismatch`で止まります。工程3は、さらに対象月の全取込み単位が同じ決定の照合値を持つことを確かめます。
- カレンダーの版は不変です。同じ版を別の内容で登録すると拒否します。改訂は、最新の版から次の版を作る場合だけ受け付けます。決定は、評価の基準時刻より後に記録された版を使えません(`ErrCalendarNotYetKnown`)。
- 対象月の締めが始まった後は、決定を変えられません(`ErrMonthClosing`)。

## 工程1 営業日カレンダーと締め日の確定(P-TIME-05)

カレンダーは、週の休日(土日)と休日の一覧を持つ版つきのJSONです。`testdata/calendar-v1.json`は、2026年の国民の祝日の日付を写した合成のデータです。締めの規則は宣言で持ちます。

| 項目 | 試験の値 | 意味 |
| --- | --- | --- |
| `closeDay` | `last-business-day` | 対象月の最終営業日を締め日とする。`nth-business-day`(第N営業日)も実装している |
| `cutoffLocal` | `17:00` | 締め日の現地時刻17時が、ファイルの締め時刻 |
| `closeJobBusinessDaysAfter`、`closeJobLocal` | `1`、`00:00` | 締め日の翌営業日の0時に締めジョブを起動する |
| `dueDateAdjustment` | `previous-business-day` | 名目の期日が休日なら前営業日へ繰り上げる |
| `businessDateRule` | `local-date-of-acceptance` | 受注の業務日付は、カレンダーの時間帯での受付日。サーバやDBのセッションの時間帯は使わない |

`Decide`は、既に有効な決定があれば、新しい決定との差(ジョブの予定のずれ、支払期日のずれ、期日が締め日の当日になる案件、追加された休日)を`calendar_impact`に記録してから、新しい決定を有効にします。

## 工程2 締め時刻のファイル取込みと照合(P-INT-01、P-INT-02)

### ファイル仕様の宣言

`testdata/file-spec.json`は、文字コードの変種(`windows-31j`)、レコード長(80バイト)、行の区切り(CRLF)、レコードの種類(見出し`H`、明細`D`、後書き`T`)ごとの項目のバイト位置・長さ・型、後書きの件数と合計の項目、事象の項目への写像、比較のときに同じとみなす文字の組を宣言します。項目は、レコードを隙間なく覆う必要があります。項目の切出しはバイト単位で、文字単位では行いません。

`windows-31j`の復号と符号化には`golang.org/x/text/encoding/japanese`の`ShiftJIS`を使います。これはWHATWGのShift_JISの表に従い、CP932の写像(0x8160はU+FF5E、NECとIBMの拡張文字を含む)になります。JIS系の写像(0x8160をU+301Cとするもの)とは、波ダッシュなど七つの文字で写像が異なります。比較で同じとみなす組には、そのうち1バイトのバックスラッシュ(U+005C)と衝突しない六つを宣言しています。

### 保留と拒否

- 文字の項目は、復号して符号化し直したバイト列が元と一致しなければ、そのレコードを保留します。保留したレコードは事象として適用せず、元のバイト列のまま記録します。
  - `non-roundtrip-code`: 同じ文字に二つの符号がある重複符号(IBM拡張の0xFA5Cは0xED40に戻る、など)。
  - `undecodable-bytes`: 復号できないバイト列(利用者定義領域の外字0xF040、項目の境界で切れた全角文字の先行バイト)。
- 次の場合は、ファイル全体を拒否し、何も適用せずに拒否の理由を記録します: レコード長の倍数でない、後書きがない(途中で切れた)、項目の形式の誤り、後書きの件数・合計の不一致、見出しの対象月の不一致、締め時刻の後の受信、同じファイル(SHA-256)の二度目の投入、同じ送り手と連番で内容の違うファイル、有効でない決定。
- 取込みは一つのトランザクションです。送り手の行を排他して、同じ送り手の取込みを直列にします。
- `reemit`は、記録から元のファイルを作り直します。適用したレコードは復号した値を仕様で符号化し直し、保留したレコードは元のバイト列を使います。

### ファイルとメッセージの照合

出荷の事象は、ファイルの明細と模擬のメッセージ(`Message`。メッセージ識別子と事象)の両方から届きます。事象は出荷番号で一つに決まり、内容の照合値は、宣言した文字の組を正規化した後の項目で計算します(JIS系の写像で書かれたメッセージの波ダッシュと、ファイルのCP932の波ダッシュを同じとみなす)。

| 受領の結果 | 条件 |
| --- | --- |
| `applied` | その出荷の最初の受領 |
| `confirmed-same-event` | 別の経路か別の参照から、同じ内容で届いた |
| `conflict-held` | 別の内容で届いた。事象は変えず、照合で`conflict`になる |
| `redelivered` | 同じメッセージ識別子(またはファイルの同じレコード)が再び届いた。配達回数だけを数える |

`reconcile-shipments`は、出荷ごとに届いた経路を`file+message`、`file-only`、`message-only`、`message-only+file-held`(ファイル側は保留)、`conflict`に分類して記録します。

## 工程3 締めジョブの実行(P-JOB-01、P-OPS-01、P-TIME-01)

### 業務キーと起動の拒否

締めジョブは対象月で識別します(`close_run`の主キー、貸与の鍵`close:<対象月>`)。起動は次の順に判定し、拒否はすべて`close_attempt`に残します。

1. 決定が有効でない、または取込み単位の決定の照合値が違う: `refused-decision`
2. 評価の時点(`-as-of`)が決定の締めジョブの予定より前: `refused-not-scheduled`
3. 取込み単位がない: `refused-no-import`
4. 完了済み: `refused-completed`
5. 期限内の貸与を別の担当が持つ: `refused-running`。強制終了された担当の貸与も、期限が切れるまでは有効なので、直後の再起動は拒否されます
6. 貸与が切れていれば引き継ぎ、新しい排他トークンで再開します(`resumed`)

### 締めの対象の時点

最初の起動は、一つのトランザクションで期間の状態を`closing`に変え、対象月に計上された受注の件数・合計・最大の識別子と、`pg_current_snapshot()`を締めの対象の時点として記録します。オンラインの受注(`PlaceEntry`)は期間の行を共有ロックで読むので、状態の変更は書込み中の受注の確定を待ち、それ以降の受注は`closing`を見ます。対象月が開いていなければ、受注は業務日付を保ったまま次の開いた期間へ計上されます(`redirected:period-not-open`)。期間の行を経由しない書込みは、トリガー`guard_period`が拒否します。

締めは、計上された受注を識別子の順に区切り(既定10件、試験は5件か7件)で明細へ写し、区切りごとに確定します。再開は、明細の最大の識別子の次から行います。最後に、明細の件数と合計が締めの対象の時点と一致することを確かめてから、結果を書き、期間を`closed`にします。

### 貸与と排他トークン

`PgOwnershipStore`は、製品コードの`internal/resource.SupervisorOwnershipStore`(取得、延長、確認、解放)をPostgreSQLで実装したものです。

- 期限の判定はすべてDBの時計(`clock_timestamp()`)で行います。ホストの時計は記録するだけです。
- 排他トークンは鍵ごとの連番の行から取り、期限切れ、解放、引継ぎをまたいで単調に増えます。延長は期限切れの貸与を生き返らせません。
- 締めの書込み(`close_run`、`close_chunk`、`close_detail`、`close_total`)は、すべて書き手の排他トークンを持ちます。トリガー`check_fence`は、トークンが現在の貸与のトークンと一致し、DBの時計で期限内の場合だけ書込みを許します。違えばSQLSTATE`TF001`で拒否します。
- 拒否した取得では、保持者がホストの時計で書いた期限と、取得しようとした側のホストの時計を比べた結果(ホストの時計に頼る設計なら引き継いだか)を`lease_event.naive_would_take_over`に残します。

## 失敗の注入

- 強制終了: `TEI_MECC_CRASH_AT=<注入点>:<n>`で、プロセスはn回目に注入点へ到達したとき自分自身をkillします。注入点は`import-before-commit`(取込みの確定の直前)、`close-chunk-mid`(区切りの明細を半分書いた後)、`close-before-complete`(結果を書いた後、確定の前)です。
- 一時停止: `TEI_MECC_STOP_AT=before-chunk-write:<n>`で、プロセスはn回目の区切りを選んだ後、書く前に自分自身へSIGSTOPを送ります。延長も含めて全体が止まり、試験がSIGCONTを送ると、選んでおいた区切りをそのまま書こうとします。Unixだけで動きます。
- 時計のずれ: `TEI_MECC_CLOCK_OFFSET=<Goの時間>`(例`-10m`)で、そのプロセスのホストの時計をずらします。
- 締めの間の書込み: 試験は、締めの実行中に、前月の業務日付の訂正と翌月の新規受注を別の接続から書き続けます。

## 実行

単体試験(DB不要):

```sh
go test ./tools/reference-month-end-close-correction/ -count=1
```

実DBの試験は、DSNがなければskipします。skipは合格の証拠になりません。WSL2などGoとDockerがある環境で、リポジトリの清浄な複製のルートから次を実行します。

```sh
python3 tools/verify-month-end-close-correction.py --out /tmp/mecc-receipt
```

スクリプトは、`postgres:16-alpine`のコンテナー(`tei-mecc-<乱数>-pg`)を一つ作り、専用のDB`tei_mecc`を作ってから、パッケージの全試験を`go test -json`で実行します。終了時に、作ったコンテナーだけを削除します。全試験がPASSし、skipが0件の場合だけ成功です。受領記録(`receipt.json`)には、コミット、作業ツリーが清浄か、Goとサーバーの版、サーバーの時間帯、試験ごとの結果、試験が記録した観測値(`observed.jsonl`)を残します。出力先はリポジトリの外に限ります。試験は、接続先のDB名が`tei_mecc`であることと、`TEI_MECC_CONFIRM_ISOLATED=tei_mecc`を確かめてから、スキーマ`mecc`を作り直します。

CLIは、環境変数`TEI_MECC_DSN`で接続先を受け取ります。時刻はすべて明示の入力で、ホストの時計を既定値にしません。

```sh
CLI="go run ./tools/reference-month-end-close-correction/cmd/tei-month-end-close"
D=tools/reference-month-end-close-correction/testdata
$CLI calendar-load -file $D/calendar-v1.json
$CLI payable-load -file payables.json
$CLI decide -month 2026-09 -calendar jp-business -version 1 -rule $D/close-rule.json -evaluated-at 2026-09-28T09:00:00+09:00 -out d1.json
$CLI calendar-revise -calendar jp-business -from 1 -holiday 2026-10-01=臨時休業 -recorded-at 2026-09-29T10:00:00+09:00
$CLI decide -month 2026-09 -calendar jp-business -version 2 -rule $D/close-rule.json -evaluated-at 2026-09-29T12:00:00+09:00 -out d2.json
$CLI import -decision d2.json -spec $D/file-spec.json -file partner.dat -received-at 2026-09-30T16:30:00+09:00
$CLI message -decision d2.json -spec $D/file-spec.json -file messages.json
$CLI reconcile-shipments -decision d2.json
$CLI reemit -spec $D/file-spec.json -import-id 1 -out partner.reemit
$CLI close -decision d2.json -owner srv-a -ttl 3s -chunk-size 10 -as-of 2026-10-02T00:30:00+09:00
```

拒否(ファイルの拒否、締めの起動の拒否とフェンシング、決定の不一致)は終了コード4を返し、標準出力に`refused`と理由を出します。

## 実DBの試験が確かめること

| 試験 | 要件 | 確かめること |
| --- | --- | --- |
| `TestPostgresPTIME05CalendarRevisionImpact` | P-TIME-05 | 締め日(9月30日)の前日に、10月1日を臨時の休日とする版2を記録する。版2は記録の前の基準時刻では使えない。版2の決定で、締めジョブが10月1日0時から10月2日0時へずれ、名目の期日が10月1日の2件が締め日の当日へ繰り上がったことが影響の一覧に記録される。古い決定は不一致で止まり、10月1日1時の締めの起動は予定前として拒否される。登録済みの版1を別の内容で置き換えられない |
| `TestPostgresPINT01FixedLengthImportAndRejections` | P-INT-01 | 30件のファイル(波ダッシュ、全角で埋まった項目と全角で始まる次の項目、半角カナ、NEC特殊文字、IBM拡張の重複符号、外字、境界をまたぐ全角文字)を取り込む。27件を適用し3件を保留する。再出力はバイト単位で一致する。0x8160はU+FF5Eで記録される。二度目の投入、件数のずれ、途中で切れたファイル、連番の再利用、締め時刻の後の受信、月の違いは全体が拒否され、事象は増えない。確定の直前にkillした取込みは何も残さず、再投入で一回だけ入る |
| `TestPostgresPINT02FileAndMessageOneEvent` | P-INT-02 | 10件をメッセージ(JIS系の写像、一部は再配達)で先に受け、途中で切れたファイルを拒否してから完全なファイルを取り込み、さらにメッセージの重複配信と内容の違うメッセージを混ぜる。事象は30件で、各出荷に一つ。数量と金額の合計はファイルの値と一致し、内容の違うメッセージは事象を変えない。照合の記録が全出荷について残る |
| `TestPostgresPJOB01KillRestartDoubleAndAfterComplete` | P-JOB-01 | 3つ目の区切りの途中でkillすると、2区切り(10件)だけが残る。直後の再起動は拒否され、期限切れの後の再起動はトークン2で引き継いで再開する。その実行中の二つ目の起動と、完了後の起動は拒否される。明細、区切り、結果、締めの対象の時点、計上された受注の件数と合計がすべて一致し、結果は1行 |
| `TestPostgresPOPS01FencingAfterTakeoverWithClockSkew` | P-OPS-01 | ホストの時計が10分遅い担当を、2つ目の区切りを選んだ後で一時停止させる。ホストの時計が10分進んだ担当の取得は、DBの時計で期限内なので拒否される(ホストの時計で判定する設計なら引き継いでいた)。期限切れの後に引き継いだ担当の実行中に古い担当を再開すると、その書込みは`token 1 is not the current token 2`で拒否される。逆向きのずれでも、DBの時計の期限切れで引き継げ(ホストの時計なら20分待っていた)、古い延長は拒否される |
| `TestPostgresPTIME01OnlineWritesDuringCloseAndBoundary` | P-TIME-01 | 締めの実行中に、9月の業務日付の訂正と10月の新規受注を書き続ける。訂正は10月へ計上され、締めの結果、明細、9月に計上された受注の件数と合計が一致する。9月30日23時59分59秒の引当待ちの受注は9月に引当待ちとして入り、10月1日0時(UTCでは9月30日)の受注は入らず、9月1日0時(UTCでは8月31日)の受注は入る。締め済みの期間の金額の直接の更新は拒否される。締めの後の引当は、月も結果も変えない |
| `TestPostgresThroughSlice1` | 通し(工程1〜3) | 工程1〜3をCLIの別プロセスで順に実行する。版1の決定での取込みは拒否される。版2の決定で取込み、重複の拒否、再出力、メッセージ、照合を行う。版2では10月1日の締めの起動が予定前として拒否される。途中で版3へ改訂して決定し直すと、版2の決定の締めは不一致で、版3の決定の締めは取込み単位の版の違いで止まる。版2の決定を同じ基準時刻で有効に戻した後、killと引継ぎを挟んで締めが完了する。決定の照合値と版2が、取込み単位、照合の記録、締めの実行記録、締めの結果まで同じ値で引き継がれ、締めの後は決定を変えられない |

## 既存の足場の利用

| 足場 | 使い方 |
| --- | --- |
| `internal/plugin/runtime/lease.go`、`internal/resource/ownership.go` | `SupervisorOwnershipStore`のインターフェースと、その意味(単調に増えるトークン、延長は期限切れを生き返らせない、同じ担当の取得は延長)を`PgOwnershipStore`で実装した。`leaseController`は資源ブローカーの貸与の監督に結び付いた非公開の型なので使っていない。`MemoryOwnershipStore`は期限をホストの時計(`Now`)で判定するメモリ実装なので、プロセスをまたぐ試験には使えない |
| `tools/reference-batch-completeness` | 使っていない。合成の入力を読取り専用で検査するメモリ内の参照で、締めの集計はSQLで行い、導出の失敗を持たない。明細が欠けたら0件として扱わず不一致で止まる点(`ErrCloseMismatch`)は同じ考え方に沿わせた |
| `tools/reference-query-input-nullability` | 依頼にあった`reference/query-nullability`はない。最も近いこの資産は問合せの入力の未指定と空値の区別を扱い、この工程の入力にない区別なので使っていない |
| `reference/derivation-context`、`tools/reference-evaluation-context` | コードは使っていない。評価の基準時刻と時間帯を明示の入力にし、ホストの時計を既定値にしない点を、決定(`-evaluated-at`)と締めの起動(`-as-of`)で同じ方針にした。暦の提供元はこの資産の対象外で、版つきの暦はここで作った |
| `tools/reference-data-import-check` | 使っていない。タブ区切りの行の形を検査する段階D0で、文字コードの変種と固定長を扱わない |

## 基盤の不足

このサンプルで、基盤にないためサンプル側で作ったものです。4.11節の「基盤への改善圧力と現状」に対応します。

- P-TIME-05: 版つきの暦と、改訂の影響の特定を宣言する手段がありません。カレンダーのJSON、`calendar_version`、締めの規則、決定とその照合値、`calendar_impact`はサンプル固有です。影響の一覧は、この決定が計算するジョブの予定と支払期日だけで、他の計算済みの期日(画面、別のジョブ)は探していません。
- P-INT-01: ファイル仕様(文字コードの変種、レコード長、項目の型、後書きの照合)と、取込み単位の同一性を宣言する手段がありません。`file-spec.json`の形式、`import_unit`、`import_record`、`import_rejection`はサンプル固有です。保留したレコードを人が直して適用する手順はありません。文字コードは`windows-31j`だけで、利用者定義領域の外字の対応表はありません。
- P-INT-02: 連携方式を実装プラグインにし、事象の同一性と照合を方式から分けて宣言する手段がありません。`shipment_event`、`shipment_receipt`、`delivery_log`、`shipment_reconciliation`、比較のときに同じとみなす文字の組はサンプル固有です。メッセージは関数呼出しの模擬で、仲介の製品を使っていません。同期APIの経路はありません。
- P-JOB-01: ジョブの実行を業務キーで識別し、完了済みの再実行と同時実行を拒否する宣言がありません。`close_run`、`close_chunk`、`close_attempt`と再開の手順はサンプル固有です。起動しそこねた実行(予定の時刻を過ぎても起動されなかった)の扱いはありません。
- P-OPS-01: 所有の記録の永続実装がありません。`PgOwnershipStore`、`job_lease`、`job_lease_seq`、トリガー`check_fence`はサンプル固有です。製品の`leaseController`と資源ブローカーには結んでいません。
- P-TIME-01: 業務日付、期間の状態、締めの対象の時点を意味モデルで宣言する手段がありません。`period`、`sales_entry`の業務日付と計上期間、計上先の付け替え、トリガー`guard_period`はサンプル固有です。

## 範囲外

- 工程4〜6(遅延到着、赤黒の訂正と遡及の改定、端数処理と帳票の再出力)。P-TIME-02、P-TIME-03、P-TIME-04、P-CALC-01はまだありません。
- P-B2C-02(締めの時刻の会員向けの照会の急増)。負荷の生成と、4の会員向けの経路が必要です。主筋を優先したため行っていません。締めジョブと社内の受注入力の応答時間は測っていません。
- 締めの対象の時点は、期間の状態の変更で書込みを次の期間へ回すことで一つに決めています。締め中も同じ期間へ書き続けながら、スナップショットだけで切り出す方式は確かめていません。記録した`pg_current_snapshot()`は判断に使っていません。
- 担当サーバは同じ機械の上の別プロセスで、時計のずれはプロセスの中の模擬です。実際の別の機械のNTPのずれ、DBのフェイルオーバー、ネットワークの分断は確かめていません。
- ファイルは合成で、実際の取引先のファイルや外字の運用は確かめていません。
