# reference-month-end-close-correction

技術検証用サンプル[`month-end-close-correction`](../../reference/month-end-close-correction/README.md)の工程1〜6を、PostgreSQLの一つのインスタンス(専用のDB`tei_mecc`)で実行する実装です。工程6の税額の照合だけは、SQLiteのメモリ上のDBでも計算します(後述の「SQLiteの経路」)。設計の正本は[技術検証用サンプルの統合設計 4.11節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#411-月次締めと締め後に届く訂正の計上month-end-close-correction新設)です。

この実装はサンプル固有の業務処理で、`internal/`の基盤には入れていません。版つきの暦、ファイル仕様と取込み単位、ジョブの業務キー、貸与の永続化、期間の状態、事象の順序の鍵と遅延到着の方針、二つの時間軸の単価、赤黒の伝票、端数処理の規則は、基盤に宣言の手段がないため、ここで最小限に作りました(後述の「基盤の不足」)。製品コードからは、`internal/resource`の`SupervisorOwnershipStore`のインターフェースだけを使っています。

## 工程と受け渡し

| 工程 | 関数(CLIの命令) | 受け取るもの | 渡すもの |
| --- | --- | --- | --- |
| 1 営業日カレンダーと締め日の確定 | `LoadCalendar`、`ReviseCalendar`、`Decide`(`calendar-load`、`calendar-revise`、`payable-load`、`decide`) | カレンダーの版、締めの規則(`testdata/close-rule.json`)、評価の基準時刻(明示の入力)、未払案件の名目の期日 | 締めの決定のJSON。対象月、カレンダーの識別子・版・照合値、地域と時間帯、評価の基準時刻、締め日、ファイルの締め時刻、ジョブの予定、支払期日、業務日付の規則、決定の照合値(`digest`)。前の決定があれば、改訂の影響(`calendar_impact`) |
| 2 締め時刻のファイル取込みと照合 | `ImportFile`、`ReceiveMessage`、`ReconcileShipments`、`Reemit`(`import`、`message`、`reconcile-shipments`、`reemit`) | 工程1の決定、取引先の固定長ファイル、ファイル仕様の宣言(`testdata/file-spec.json`)、模擬のメッセージ | 取込み単位の記録(`import_unit`。ファイルのSHA-256、送り手と連番、件数と合計、保留の件数、決定の照合値とカレンダーの版)、レコードごとの記録と保留の理由、拒否の記録(`import_rejection`)、出荷の事象と受領の記録、照合の記録(`shipment_reconciliation`) |
| 3 締めジョブの実行 | `RunClose`(`close`)、`PlaceEntry`、`AllocateEntry` | 工程1の決定、工程2の取込み単位(全件が同じ決定の照合値であること)、オンラインの受注と訂正 | 対象月を業務キーとする締めの実行記録(`close_run`。決定の照合値、カレンダーの版、取込み単位の照合値、締めの対象の時点、排他トークン)、区切りごとの記録と明細(`close_chunk`、`close_detail`)、締めの結果(`close_total`)、起動の記録(`close_attempt`)、貸与の記録(`lease_event`)。売上伝票の件数と金額の切出し(`close_run.cut_slips`、`cut_slip_amount`)と、締め時点の帳票とその照合値(`close_report`) |
| 4 締め後の遅延到着の受付 | `ApplyEvent`、`ReleaseHeld`(`events`、`release`) | 工程1の決定(締め後も対象月の有効な決定であること)、遅延到着の方針(`testdata/late-arrival-policy.json`)、受注明細の事象(順不同、締めの後の到着を含む) | 事象ごとの業務日付、受信時刻、順序の鍵(受注番号と版)、到着時の期間の状態、適用した方針(`order_event`)、配達の記録(`order_event_delivery`)、受注明細の射影(`order_state`)、事象が起こした伝票 |
| 5 遡及の改定と赤黒の訂正 | `LoadItems`、`RevisePrice`、`CancelRevision`、`PriceAt`、`AmendSlipInPlace`(`item-load`、`price-revise`、`price-cancel`、`price-at`、`slip-amend`) | 工程4の事象と、単価の有効期間・記録時点の履歴(`item_price`) | 元伝票と結んだ赤黒の伝票(`slip`。`reverses`、`corrects`)、計上した期間と理由、選んだ単価の有効日(`price_valid_on`)と記録時点(`price_known_at`)、訂正した締め(`closed_month`、`close_token`、`close_decision_digest`)、単価の変更の記録(`price_change`。拒否を含む) |
| 6 端数処理と締め時点の帳票の再出力 | `CheckTax`、`Reissue`(`tax`、`report`) | 工程3の締めの記録と帳票、工程5の伝票、端数処理の規則(`testdata/tax-rule.json`) | 請求書単位・税率別の税額と明細への按分(PostgreSQL、Go、SQLiteの比較)、締め時点の帳票の再出力とバイト比較、現在時点の残高と、締めとの差を作った伝票の一覧 |

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

### 売上伝票と締め時点の帳票

工程4〜6の売上伝票(`slip`)は、工程3の受注の計上(`sales_entry`)とは別の台帳です。伝票は1行で1枚とし、受注明細の事象から作ります。締めは、最初の起動で`sales_entry`と同じトランザクションで伝票の件数と金額を切り出し、完了のトランザクションで、切出しと伝票が一致することを確かめてから、締め時点の帳票を作って`close_report`に保存します。`close_report`もトリガー`check_fence`の対象です。

## 工程4 締め後の遅延到着の受付(P-TIME-04、P-INT-02の続き)

事象は受注明細ごとの登録(版1)、数量の変更(版2以降)、取消し(版2以降)です。順序の鍵は受注番号と版で、到着の順ではなく版の大きいものが勝ちます。`ApplyEvent`は一つのトランザクションで、受注明細の行を排他してから次を行います。

| 到着 | 結果 |
| --- | --- |
| 同じ事象識別子が再び届いた | `redelivered`。配達回数だけを数える |
| 同じ鍵(受注番号と版)が別の識別子・経路から同じ内容で届いた | `confirmed-same-event` |
| 同じ鍵が別の内容で届いた | `conflict-held`。射影を変えない |
| 業務日付の月が開いていない | 方針の宣言に従う。`adjust-open-period`は受け付けて次の開いた月へ計上し、`hold`は保留して射影を変えない |
| 登録がまだ届いていない | `pending-base`。事象は捨てずに残し、登録が届いたときに版の順に畳み込む |
| より大きい版が既にある | `stale-version`。受け付けるが、射影は変わらない |

射影(`order_state`)は、受け付けた事象を版の順に畳み込んだ結果(`FoldOrder`)です。射影が変わると、伝票との差を工程5の赤黒で計上します。試験の方針(`testdata/late-arrival-policy.json`)は、締めた月の登録と変更を当月(次の開いた月)で調整し、取消しは保留します。保留した事象は、人が`release`で適用します。

## 工程5 遡及の改定と赤黒の訂正(P-TIME-02、P-TIME-03)

### 二つの時間軸の単価

`item_price`の各行は、有効期間(`valid_from`〜`valid_to`、日付の半開区間)と記録期間(`recorded_from`〜`recorded_to`、時刻の半開区間)を持ちます。記録時点は明示の入力で、ホストやDBの時計を使いません。照会(`PriceAt`)は、有効日と記録時点の両方を含む行がちょうど1行であることを求め、すき間は`ErrNoPrice`にします。範囲型や拡張機能は使わず、比較だけで書いています。

| 操作 | 内容 |
| --- | --- |
| 改定(`RevisePrice`) | 指定日から、その日を含む行の終わりまでの単価を変える。記録時点の日付より前なら`retroactive`、後なら`reserve`、同じなら`revise`と記録する |
| 取消し(`CancelRevision`) | 指定日に始まる改定を取り下げ、直前の行の単価に戻す |
| 素朴な登録(`RegisterPriceRaw`) | 既知の行を閉じずに追加する。現在の行と重なれば`ErrPriceOverlap`で拒否する |

変更は、重なる現在の行の記録期間を記録時点で閉じ、範囲の外側の部分をそのまま入れ直し、新しい行を入れます。記録時点より前に知っていたことは変わりません。品目の最新の記録時点より前の記録時点での変更は`ErrPriceRecordedBackwards`で拒否します。トリガー`guard_price`は、同じ品目で両方の軸が重なる行の挿入を拒否し(SQLSTATE`TQ001`)、記録期間を閉じる以外の更新と削除を拒否します(`TQ002`)。拒否は`price_change`に残します。

### 赤黒の伝票

伝票は変えません。射影や単価が伝票と違えば、`resync`が、生きている伝票(打ち消されていない元伝票か黒伝票)をちょうど打ち消す赤伝票(`reverses`。数量と金額を負にし、単価は元のまま)と、新しい値の黒伝票(`corrects`)を計上します。黒伝票の単価は、出荷日に有効な値を、訂正を記録した時点(事象の受信時刻、改定の記録時点、保留の解除時刻)の知識で選びます。伝票は業務日付の月が開いていればその月へ、開いていなければ次の開いた月へ、方針を理由として計上します。締めた月の伝票は、その締めの排他トークンと決定の照合値を持ちます。

トリガー`guard_slip`は、開いていない月への挿入と、開いていない月の伝票の更新・削除を拒否し(`TP001`)、開いている月の伝票の更新・削除も拒否します(`TP002`)。`AmendSlipInPlace`(`slip-amend`)は、伝票を直接更新する素朴な訂正で、必ず拒否されます。

## 工程6 端数処理と締め時点の帳票の再出力(P-CALC-01)

### 端数処理の規則

`testdata/tax-rule.json`は、実装した唯一の規則(`DefaultTaxRule`)と同じ内容であることを確かめて受け付けます。

| 項目 | 値 | 意味 |
| --- | --- | --- |
| `unit`、`invoiceKey` | `invoice-per-rate`、`customer+posting-month` | 請求書(得意先と計上月)ごと・税率ごとに1回だけ丸める |
| `method` | `truncate-toward-zero` | 円未満を0の方向へ切り捨てる。赤伝票だけの請求書の税額は、元の税額のちょうど符号違いになる |
| `lineAmountScale`、`taxScale` | `2`、`0` | 明細の金額は銭まで保持し、税額は円 |
| `allocation`、`remainderTo` | `proportional-truncate-toward-zero`、`largest-absolute-amount-then-smallest-slip-id` | 税額を明細の金額の比で按分して切り捨て、残りを金額の絶対値が最大の明細(同じなら伝票番号の小さいもの)に寄せる |

`CheckTax`は、同じ伝票について三つの経路で請求書ごとの税額と明細への按分を計算し、すべて一致しなければ`ErrTaxPathsDisagree`を返します。

- PostgreSQL: `numeric(14,2)`の金額と、0の方向へ切り捨てる`div()`。
- Go: 銭単位の整数と整数の除算。
- SQLite: 銭単位の`INTEGER`(`STRICT`の表)と整数の除算。

観測のために、明細ごとに丸めて足す設計、負の無限大の方向へ丸める設計、SQLiteの`REAL`、Goの`float64`で計算した場合に、税額が違う請求書の数も出します。

### SQLiteの経路

go.modにSQLiteのドライバーはなく、新しい依存は追加しません。[`reference/effect-fence`](../../reference/effect-fence/README.md)は、ビルドタグ`tei_sqlite_reference`とcgoでSQLiteのC APIを直接呼びますが、`sqlite3.h`と`libsqlite3.so`(開発用パッケージ)が必要で、検証に使ったWSLの環境にはありません(`go vet -tags tei_sqlite_reference ./reference/effect-fence`は`sqlite3.h: No such file or directory`で失敗)。`sqlite3`のCLIもありません。そのため、SQLiteの経路は、Pythonの標準モジュール`sqlite3`(同じ環境のlibsqlite3)をメモリ上のDBで使う別プロセスにしました。SQLの本文はすべてGoの側にあり、`python3 -I`で起動するスクリプトはそれを実行して結果を返すだけです。インタープリタは環境変数`TEI_MECC_PYTHON`で指定でき、なければ`python3`を探します。見つからなければ、税額の照合は失敗します(skipにしません)。このSQLiteは製品のDBではなく、計算の経路の比較のためだけに使います。

### 帳票の再出力

帳票はUTF-8、LF区切りのテキストで、見出しに対象月、決定の照合値、カレンダーの版、締め日、締めの排他トークン、締めの切出しの時刻(DBの時計)、端数処理の規則を持ち、伝票の行、請求書ごとの税額、合計を続けます。

| 基準(`-basis`) | 内容 |
| --- | --- |
| `close` | その月に計上し、締めの切出しの時刻までに記録された伝票を、締めと同じ方法で出力し、`close_report`の保存値とバイト単位で比べる |
| `current` | その月の業務日付の伝票を、後の月に計上した赤黒も含めて現在の記録で出力し、締めとの差が締めの後の伝票の合計と一致することを確かめて、その一覧を返す |

`Reissue`は、受け取った決定が対象月の有効な決定で、かつ締めに使った決定であることを確かめます。

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

スクリプトは、`postgres:16-alpine`のコンテナー(`tei-mecc-<乱数>-pg`)を一つ作り、専用のDB`tei_mecc`を作ってから、パッケージの全試験を`go test -json`で実行します。終了時に、作ったコンテナーだけを削除します。全試験がPASSし、skipが0件の場合だけ成功です。工程6の税額のSQLiteの経路には、スクリプトを実行したPython(`TEI_MECC_PYTHON`)の標準モジュール`sqlite3`を使います。受領記録(`receipt.json`)には、コミット、作業ツリーが清浄か、Goとサーバーの版、サーバーの時間帯、PythonとSQLiteの版、試験ごとの結果、試験が記録した観測値(`observed.jsonl`)を残します。出力先はリポジトリの外に限ります。試験は、接続先のDB名が`tei_mecc`であることと、`TEI_MECC_CONFIRM_ISOLATED=tei_mecc`を確かめてから、スキーマ`mecc`を作り直します。

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
$CLI item-load -file items.json
$CLI events -decision d2.json -policy $D/late-arrival-policy.json -file september-events.json
$CLI close -decision d2.json -owner srv-a -ttl 3s -chunk-size 10 -as-of 2026-10-02T00:30:00+09:00
$CLI events -decision d2.json -policy $D/late-arrival-policy.json -file late-events.json
$CLI release -decision d2.json -policy $D/late-arrival-policy.json -event ev-SO-L025-v2 -recorded-at 2026-10-03T09:00:00+09:00
$CLI price-revise -item IT-09 -valid-from 2026-09-01 -price 125.00 -recorded-at 2026-10-02T19:00:00+09:00 -decision d2.json -policy $D/late-arrival-policy.json
$CLI price-at -item IT-09 -on 2026-09-05 -known-at 2026-10-01T00:00:00+09:00
$CLI slip-amend -slip 1 -qty 9
$CLI tax -tax-rule $D/tax-rule.json -months 2026-09,2026-10
$CLI report -decision d2.json -basis close -out close-report.txt
$CLI report -decision d2.json -basis current -out current-balance.txt
```

拒否(ファイルの拒否、締めの起動の拒否とフェンシング、決定の不一致、締めた月や伝票の変更、単価の重なりと記録時点の逆行)は終了コード4を返し、標準出力に`refused`と理由を出します。税額の経路の不一致と帳票の不一致は失敗(終了コード1)です。

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
| `TestPostgresPTIME04LateAndReorderedEvents` | P-TIME-04、P-INT-02 | 100件の受注明細の126件の事象を、Aでは順序どおり締めの前に、Bでは混ぜて適用する。Bでは登録より先に変更を届け、3件を再配達し、1件を別の経路で、1件を別の内容で届け、23件を締めの後に届けて、そのうち1件を再配達する。Bの締めの後の到着は、締めた月の伝票、締めの結果、帳票、締めの切出しを変えず、すべて10月へ方針どおりの理由で計上され、締めの排他トークンと決定の照合値を持つ。取消し4件は保留され、それ以外の受注明細の射影と、伝票の数量・金額の正味はAと一致する。保留を解除すると全件が一致する |
| `TestPostgresPTIME02PTIME03RedBlackAndBitemporalPrice` | P-TIME-03、P-TIME-02 | 9月10日の記録で9月1日からの単価改定(遡及)、9月12日の記録で11月1日からの予約改定、9月15日の記録で別の品目の遡及改定の取消しを行う。9月5日の単価を9月7日と9月12日の記録時点で照会すると101.37と104.11が、取り消した品目は9月7日、12日、20日で33.33、34.56、33.33が返る。予約改定は9月11日の記録時点では見えない。重なる素朴な登録、SQLでの重なる挿入(`TQ001`)、記録時点の逆行、単価の行の更新(`TQ002`)、二度目の取消しは拒否され、拒否が3件記録される。締めの後、9月の伝票の直接の更新・削除・挿入は`TP001`で拒否される。10月2日の記録の遡及改定と遅れて届いた数量の変更は、元伝票と結んだ赤黒3組として10月へ計上され、赤は元のちょうど符号違い、黒は出荷日の有効値を訂正の記録時点の知識で選ぶ。開いている10月の伝票も直接は更新できない(`TP002`) |
| `TestPostgresPCALC01TaxPathsAndReissue` | P-CALC-01 | 8%と10%が混在し銭の端数が出る単価(0.99、33.33、98.73など)の100件の受注明細を締め、締めの後に数量の変更20件と、遡及の単価改定1件を加える。9月、10月、両方の伝票で、PostgreSQL、Go、SQLiteの請求書ごとの税額と按分が一致する。明細ごとに丸める設計は9月の請求書で税額がずれ、10月には負の請求書があって負の無限大の方向への丸めでもずれる。赤黒の差額に銭の端数が出る。締め時点の帳票の再出力は保存値とバイト単位で一致し、各伝票の単価を伝票の記録時点の知識で選び直しても一致するが、現在の知識で選び直すと一致しない。現在時点の残高は、生きている受注明細を現在の単価で計算した値と一致する |
| `TestPostgresThroughSlice2` | 通し(工程1〜6) | 工程1〜6をCLIの別プロセスで順に実行する。版2の決定で取込み、品目の登録、9月の事象、締めを行い、締めの後の事象は版1の決定では不一致で止まり、版2の決定で受け付けられる。締めた月の伝票の直接の更新は拒否され、遡及改定は赤黒で計上され、税額の三つの経路が一致し、締め時点の帳票は締めの保存値と同じ照合値で再出力される。版1の決定での帳票の再出力は止まる。決定の照合値と版2が、事象、全伝票、締めの実行記録と帳票まで同じ値で引き継がれ、締めの後の伝票と両方の帳票が締めの排他トークンを持つ |

## 既存の足場の利用

| 足場 | 使い方 |
| --- | --- |
| `internal/plugin/runtime/lease.go`、`internal/resource/ownership.go` | `SupervisorOwnershipStore`のインターフェースと、その意味(単調に増えるトークン、延長は期限切れを生き返らせない、同じ担当の取得は延長)を`PgOwnershipStore`で実装した。`leaseController`は資源ブローカーの貸与の監督に結び付いた非公開の型なので使っていない。`MemoryOwnershipStore`は期限をホストの時計(`Now`)で判定するメモリ実装なので、プロセスをまたぐ試験には使えない |
| `tools/reference-batch-completeness` | 使っていない。合成の入力を読取り専用で検査するメモリ内の参照で、締めの集計はSQLで行い、導出の失敗を持たない。明細が欠けたら0件として扱わず不一致で止まる点(`ErrCloseMismatch`)は同じ考え方に沿わせた |
| `tools/reference-query-input-nullability` | 依頼にあった`reference/query-nullability`はない。最も近いこの資産は問合せの入力の未指定と空値の区別を扱い、この工程の入力にない区別なので使っていない |
| `reference/derivation-context`、`tools/reference-evaluation-context` | コードは使っていない。評価の基準時刻と時間帯を明示の入力にし、ホストの時計を既定値にしない点を、決定(`-evaluated-at`)と締めの起動(`-as-of`)で同じ方針にした。暦の提供元はこの資産の対象外で、版つきの暦はここで作った |
| `reference/derivation-temporal` | コードは使っていない。合成の読取り専用の参照で、二時間軸のDBと履歴の保存を持たない。業務の有効時点と記録時点を同じものとして扱わない点を、`item_price`の二つの期間と`PriceAt`の二つの引数で同じ方針にした |
| `reference/derivation-typed-arithmetic`(`internal/semantic/derivationref.Multiply`) | 使っていない。丸めの方式はHALF_UPとHALF_EVENで、暗黙の切捨てをしない設計のため、この規則(0の方向への切捨て、請求書単位・税率別、按分の端数寄せ)を表せない。明細の金額は丸めずに銭で保持するので、乗算は整数で足りる |
| `reference/effect-fence` | 使っていない。SQLiteへの接続の方式(ビルドタグとcgo)を確かめたが、検証環境に`sqlite3.h`がなくビルドできないため、SQLiteの経路は別プロセスにした(「SQLiteの経路」) |
| `tools/reference-data-import-check` | 使っていない。タブ区切りの行の形を検査する段階D0で、文字コードの変種と固定長を扱わない |

## 基盤の不足

このサンプルで、基盤にないためサンプル側で作ったものです。4.11節の「基盤への改善圧力と現状」に対応します。

- P-TIME-05: 版つきの暦と、改訂の影響の特定を宣言する手段がありません。カレンダーのJSON、`calendar_version`、締めの規則、決定とその照合値、`calendar_impact`はサンプル固有です。影響の一覧は、この決定が計算するジョブの予定と支払期日だけで、他の計算済みの期日(画面、別のジョブ)は探していません。
- P-INT-01: ファイル仕様(文字コードの変種、レコード長、項目の型、後書きの照合)と、取込み単位の同一性を宣言する手段がありません。`file-spec.json`の形式、`import_unit`、`import_record`、`import_rejection`はサンプル固有です。保留したレコードを人が直して適用する手順はありません。文字コードは`windows-31j`だけで、利用者定義領域の外字の対応表はありません。
- P-INT-02: 連携方式を実装プラグインにし、事象の同一性と照合を方式から分けて宣言する手段がありません。`shipment_event`、`shipment_receipt`、`delivery_log`、`shipment_reconciliation`、比較のときに同じとみなす文字の組はサンプル固有です。メッセージは関数呼出しの模擬で、仲介の製品を使っていません。同期APIの経路はありません。
- P-JOB-01: ジョブの実行を業務キーで識別し、完了済みの再実行と同時実行を拒否する宣言がありません。`close_run`、`close_chunk`、`close_attempt`と再開の手順はサンプル固有です。起動しそこねた実行(予定の時刻を過ぎても起動されなかった)の扱いはありません。
- P-OPS-01: 所有の記録の永続実装がありません。`PgOwnershipStore`、`job_lease`、`job_lease_seq`、トリガー`check_fence`はサンプル固有です。製品の`leaseController`と資源ブローカーには結んでいません。
- P-TIME-01: 業務日付、期間の状態、締めの対象の時点を意味モデルで宣言する手段がありません。`period`、`sales_entry`の業務日付と計上期間、計上先の付け替え、トリガー`guard_period`はサンプル固有です。
- P-TIME-04: 事象の順序の鍵と版、業務日付と受信時刻の区別、締めた月に届いた事象の方針を宣言する手段がありません。`order_event`、`order_event_delivery`、`order_state`、`FoldOrder`、方針のJSON(`adjust-open-period`と`hold`)と保留の解除はサンプル固有です。前月を開き直す方針はありません。事象は数量の変更と取消しだけで、品目や出荷日の変更、同じ受注番号の再登録は扱いません。
- P-TIME-02: 確定した行の不変性、赤黒の訂正、訂正を計上する期間の規則を宣言する手段がありません。`slip`の`reverses`と`corrects`、`resync`、トリガー`guard_slip`はサンプル固有です。期間の再オープンと、その権限・監査はありません。伝票は1行1枚で、複数行の伝票や伝票番号の採番はありません。
- P-TIME-03: 有効時間と記録時間の二つの軸で値を持ち、時点を指定して照会する手段がありません。`item_price`、`price_change`、`PriceAt`、改定・取消し・素朴な登録、トリガー`guard_price`はサンプル固有です。重なりの検査は品目の行の排他とトリガーの比較で行い、PostgreSQLの範囲型や排他制約には頼っていません。改定は品目ごとに記録時点が単調に増えることを求め、同じ時点の二つの変更や、過去の記録時点への訂正(記録の誤りの訂正)はできません。
- P-CALC-01: 端数処理の単位、方式、桁、按分の端数を寄せる規則を意味モデルで宣言する手段がありません。`TaxRule`は一つの規則だけを実装し、宣言のJSONはその規則と同じ内容であることだけを確かめます。税率は品目に固定で、税率の改定(有効期間)はありません。SQLiteの経路は、Pythonの標準モジュールを使う別プロセスで、TEIのProviderではありません。
- 帳票: 締め時点の帳票の形式と、その照合値を保存する手段がありません。`close_report`、`RenderReport`、`Reissue`はサンプル固有です。

## 範囲外

- 工程4〜6の売上伝票は、工程2の出荷の事象(`shipment_event`)からは作っていません。工程2の事象は品目と税率を持たないため、受注明細の事象を別に受けています。出荷の事象と伝票を結ぶこと、工程2のファイルとメッセージを工程4の事象の経路にすることは確かめていません。
- 締め時点の帳票は、締めの切出しの時刻(DBの時計)までに記録された伝票で作ります。事象や単価の記録時点(明示の業務の時刻)で伝票を選び直す再出力はしていません。各伝票の単価は伝票の記録時点の知識で選び直せることを、P-CALC-01の試験で確かめているだけです。
- 複合シナリオC4(会員向けの経路と承認途中の案件)はありません。
- P-B2C-02(締めの時刻の会員向けの照会の急増)。負荷の生成と、4の会員向けの経路が必要です。主筋を優先したため行っていません。締めジョブと社内の受注入力の応答時間は測っていません。
- 締めの対象の時点は、期間の状態の変更で書込みを次の期間へ回すことで一つに決めています。締め中も同じ期間へ書き続けながら、スナップショットだけで切り出す方式は確かめていません。記録した`pg_current_snapshot()`は判断に使っていません。
- 担当サーバは同じ機械の上の別プロセスで、時計のずれはプロセスの中の模擬です。実際の別の機械のNTPのずれ、DBのフェイルオーバー、ネットワークの分断は確かめていません。
- ファイルは合成で、実際の取引先のファイルや外字の運用は確かめていません。
