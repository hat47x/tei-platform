# X-TEA標準業務ライブラリの受容設計 — 暦・税・通貨・採番（2026-11）

## 1. この文書の役割

X-TEA（XEAD）の Driver が持つ**標準業務ライブラリ**（日付・年度・営業日・税額・通貨換算・採番・利用者設定）を読み、TEIの[D6（標準業務意味）](XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)と[期間状態の設計](PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md)へ受け止める形を整理します。

- 個別機構は[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §4.6（S-1〜S-6）にあります。この文書は、D6.1で「最初は暦と会計期間だけ」と決めた範囲を、**宣言の形と端数・境界の扱い**まで具体化します。
- 判断は**仮承認**です。受容の採否は別途です。
- 根拠は固定版（XeadDriver `6c077773b74c6d7dbcc253835cf8d347e4ffb1ef`）の静的な読解です。実DB・旧Runtimeの実測ではありません（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §6）。

## 2. 観測した機構と規則

### 2.1 年度と締め月

`Session`は、システム変数`ALLOWED_FISCAL_MONTH`（許容する会計月）と`FIRST_MONTH`（年度開始月）を持ちます（`D:Session.java L1461`、L2383–2387）。年度は日付から計算し、締め月は変数で決まります。

**観測した規則**: 年度の開始月と許容範囲は、**システム変数（設定値）**であり、案件のDataとして宣言されていません。

### 2.2 休日と営業日

休日は`offDateList`に`KBCALENDAR + ";" + 日付`の組で持ちます（`D:Session.java L786`、L1691、L1882）。営業日数は、暦種別ごとに休日を飛ばして数えます（同 L1765–1830）。

**観測した規則**: 暦種別（`kbCalendar`）は**文字列の識別子**で、既定は`"00"`です（同 L1760）。休日はDataではなく、実行時に読み込む表です。

### 2.3 税額

`getTaxAmount(date, amount)`は、税率表を開始日（`DTSTART`）の降順で読み、対象日以降で最も新しい行の税率を使います（`D:Session.java L2315–2334`）。正の金額は切り捨て（`Math.floor`）、負の金額は切り上げ（`Math.ceil`）ます。

**観測した規則**: 適用は**開始日のみ**で、終了日を持ちません。端数処理は**金額の符号で決まり**、宣言できません。

### 2.4 通貨換算

`getAnnualExchangeRate(currencyCode, fYear, type)`は、年度単位の表を`DTNEND`（年度）で引き、`type`が`TTB`なら買値、`TTS`なら売値、それ以外は中値を使います（`D:Session.java L2253–2281`）。システム通貨は率1です。

**観測した規則**: 率の種類（中値・買値・売値）は**引数の文字列**で選び、適用期間は年度単位です。日次・月次は別の表（`getMonthlyExchangeRate`、同 L2281–2313）です。

### 2.5 採番

`getNextNumber(numberID)`は、採番表の行を`IDNUMBER`で引き、`NRCURRENT`・`NRNUMDIGIT`・`TXPREFIX`・`FGWITHCD`から次の番号を作り、`NRCURRENT`を1増やして`UPDCOUNTER`を+1します（`D:Session.java L1980–2010`）。

**観測した規則**

- 桁あふれは1へ戻ります（`countUpNumber`、同 L2061–2100）。
- チェックディジットは**モジュラス10**で、各桁の和を10で割った余りを10から引きます（同 L2050–2057）。ただし桁が0の場合は付きません。
- 更新は`UPDCOUNTER`の一致を条件にした**楽観的更新**です。一致しなければ更新は効かず、結果は検査されません（同 L1995–2002）。
- 例外は握りつぶされ、空文字が返ります（同 L2005–2007）。

### 2.6 日付の数え方

`getDaysBetweenDates(from, thru, countType, kbCalendar)`は、`countType`で数え方を切り替えます（`D:Session.java L1761–1830`）。

| `countType` | 意味 |
| --- | --- |
| 0 | 実日数（ミリ秒差を86400000で割る） |
| 1 | 営業日数（休日を飛ばす） |
| 2 | 暦日数（休日を含む日数） |

**観測した規則**: 数え方は**呼出し側の整数引数**で決まり、意味として宣言されません。夏時間のある地域では、ミリ秒差の割り算が1日ずれ得ます。

### 2.7 利用者設定

`getSystemVariant*`・`setVariant`・`getUserValueOf`で、システム変数と利用者別の値を読み書きします（`D:Session.java L2112`、`D:XFScriptable.java L56–60`）。

## 3. 属性の役割語彙

標準業務のDataは、列名ではなく**属性の役割**で結合します（§5の三段の「結合」）。役割は`role <name>`として属性に宣言し、検証は既知の役割だけを受理します。標準業務で使う役割は次の17個です。

| 対象 | 役割 | 必須 |
| --- | --- | --- |
| 税 | `tax-applies-from`・`tax-applies-until`・`tax-value`・`tax-rounding` | 必須 |
| 税 | `tax-category` | 任意 |
| 通貨 | `currency-rate-currency`・`currency-rate-fiscal-year`・`currency-rate-month-sequence`・`currency-rate-middle` | 必須 |
| 通貨 | `currency-rate-buying`・`currency-rate-selling` | 任意 |
| 通貨 | `currency-rate-fallback` | 任意（値は`none`か`annual`。行ごとに一致させる） |
| 採番 | `numbering-counter`・`numbering-digits` | 必須 |
| 採番 | `numbering-prefix`・`numbering-check`・`numbering-overflow` | 任意 |
| 採番 | `numbering-rule` | 任意（複数規則を1つのDataに置く場合） |
| 休日 | `holiday-date` | 必須（休日Dataを宣言する場合） |
| 休日 | `holiday-calendar` | 任意（複数の暦が1つのDataを共有する場合。型が値域を宣言していれば、宣言外の識別は報告される） |

- 必須の役割が欠けたDataは結合で拒否され、欠けた役割名が報告されます。
- 任意の役割は、列が空なら契約の既定（接頭辞なし・検査数字なし・停止など）のままです。空欄から別の規則を推測しません。
- 語彙の追加は`internal/validation/validation.go`の`attributeRoles`に置き、綴りの誤りは引き続き`SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED`で報告します。
- 役割が要求するプリミティブ型（`date`・`decimal`・`integer`）は`internal/validation/attribute_role_type.go`の`rolePrimitives`に置きます。宣言の型が要求と違う場合は`SEMANTIC_ATTRIBUTE_ROLE_TYPE_MISMATCH`で報告し、別名（`type X from Y`）は下地まで辿って判定します。型名と役割名のどちらも「何でも受理する」ことはしません。

## 3.1 受け止める形

### 3.1 年度・締め月は「暦の宣言」に置く

X-TEAは年度開始月と許容範囲をシステム変数に置きました。TEIでは、これを**暦の宣言**として持ち、期間の宣言（`period`）と結びます。

```tei
calendar sales.fiscal_calendar {
    fiscal-year starts month 4
    week starts monday
    business-day excludes holiday
}

period sales.accounting_period {
    calendar sales.fiscal_calendar
    key sales.period.year_month
    states open, closing, closed
    closed rejects update, delete
}
```

- `fiscal-year starts month`は年度の開始月です。X-TEAの`FIRST_MONTH`に対応します。
- 休日は暦の宣言から参照するDataとし、実行時の表読み込みにしません。
- 期間の識別（年月・年度＋四半期）は`period key`で宣言します（既存の[期間状態の設計](PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md)§3.2）。

### 3.2 休日はDataとして持ち、暦から参照する

X-TEAは`KBCALENDAR`の文字列で暦を切り替えました。TEIでは暦を**宣言された識別子**（Canonical ID）にし、休日をDataとして持ちます。

```tei
data sales.holiday {
    attribute calendarId type sales.calendar_id
    attribute holidayOn type std.date
}

calendar sales.operation_calendar {
    business-day excludes sales.holiday where calendarId = sales.operation_calendar
}
```

- 暦の識別子は文字列ではなくCanonical IDです。同じ文字列が別の案件で別の意味になることを防ぎます。
- 休日の追加・削除は、意味の変更として受理の対象になります。

### 3.3 税は「適用期間」と「端数処理」を宣言する

X-TEAの税率は開始日だけで選び、端数処理は金額の符号で決まりました。TEIでは、税率Dataに**適用期間の両端**を持たせ、端数処理を宣言します。

```tei
data tax.rate {
    attribute rateId type tax.rate_id
    attribute appliesFrom type std.date
    attribute appliesUntil type std.date
    attribute category type tax.category
    attribute rate type std.decimal
    attribute rounding type tax.rounding
}
```

- `appliesUntil`を持つことで、改正日の前後を**境界で試験**できます。
- 端数処理（`floor`・`ceil`・`half-up`・`half-even`）は、金額の符号ではなく宣言で決めます。X-TEAの「正は切り捨て・負は切り上げ」は、`floor`を宣言すれば同じ結果になります。
- 適用期間が重なる場合は、**一意性の検査**（候補キー`category`＋`appliesFrom`）で検出します。

### 3.4 通貨は「率の種類」と「適用日」を宣言する

X-TEAの率の種類（中値・買値・売値）は引数の文字列でした。TEIでは`RateKind`として宣言し、適用日と結びます。

- `TTB`（買値）・`TTS`（売値）・中値は、`rate kind`の列挙にします。
- 換算の方向（外貨→自国通貨、自国通貨→外貨）を、能力の入力として明示します。X-TEAは呼出し側の使い方に依存していました。
- 年度単位と日次・月次の区別は、率Dataの粒度（`period`）として宣言します。年次へのフォールバックは`Fallback`（既定`none`、`annual`で年次行）として宣言し、月次行の欠落を0で埋めません。

### 3.5 採番は「契約」として持ち、形式は提供元に置く

X-TEAの採番は、表の行・桁・接頭辞・チェックディジット・巻き戻しが一つの関数に埋まっていました。TEIでは、**連続性・一意性・欠番の扱い**を契約にし、形式（接頭辞・桁・チェックディジット）は提供元の設定にします（D6.1の推奨）。

| 契約の項目 | 内容 | X-TEAの対応 |
| --- | --- | --- |
| 一意性 | 同じ採番対象の中で、発行済みの番号を再発行しない | `UPDCOUNTER`の楽観的更新（結果は未検査） |
| 連続性 | 発行順に単調増加する | 1ずつ増加 |
| 欠番 | 失敗時に欠番を許すか | 例外で空文字を返し、欠番が生じる |
| 桁あふれ | 上限に達したときの扱い（停止・巻き戻し） | 1へ戻る（**巻き戻し**） |
| 検査数字 | 付すか、方式は何か | モジュラス10 |

- **桁あふれの巻き戻しは危険**です。TEIの既定は「停止」にし、巻き戻しは宣言で選ぶ形にします。X-TEAの既定（巻き戻し）は、一意性を壊します。
- 採番の発行は、取引の中で行い、失敗時に取り消せるようにします。X-TEAは`commit`の外側で更新する経路がありました。`internal/numbering`は`Reserve`で予約し、`Commit`で確定、`Release`でカウンタを戻します。予約の間に別の呼出しが発行した場合は`ErrNumberingConflict`で報告し、後続の発行を上書きしません。`ReserveInTransaction`は番号の発行と呼出し側の変更を同じ取引に入れ、取引Commitが失敗したらカウンタを戻します。

### 3.6 日付の数え方は列挙にする

`countType`の整数は、`DayCount`の列挙にします。

| X-TEA | TEI |
| --- | --- |
| 0（実日数） | `elapsed` |
| 1（営業日） | `business` |
| 2（暦日） | `calendar` |

- 夏時間のある地域で実日数がずれる問題は、ミリ秒差ではなく**日付の差**として計算します。
- 営業日の数え方は、暦の宣言（休日）を入力にします。

### 3.7 利用者設定は、承認された範囲だけを意味にする

X-TEAの`setVariant`・`setUserValueOf`は、スクリプトから任意の値を書き換えられました。TEIでは、**宣言された設定項目**だけを対象にし、書込みは受理の対象にします。

## 4. 受容先の層

| 対象 | 層 | 置くもの | 置かないもの |
| --- | --- | --- | --- |
| 暦（年度開始・休日・営業日） | L1 | 暦の宣言、休日のData、期間との結び付き | 実行時の設定ファイル |
| 会計期間 | L1 | 期間の宣言、状態、拒否する操作、期間キー（年度＋月順序） | 締めの手順（L2のCommand） |
| 税 | L1＋L4 | 税率Data、適用期間、端数処理の宣言、計算能力 | 税率の正本（法制度） |
| 通貨 | L1＋L4 | 率Data、率の種類、適用日、換算能力 | 市場データの取得 |
| 採番 | L4（契約）＋L1（宣言） | 一意性・連続性・欠番・桁あふれの契約、形式の設定 | 番号の実体（表） |
| 利用者設定 | L1 | 宣言された設定項目 | 任意のキー |

## 5. 実装の順序と現在地

1〜6は参照実装まで進みました（6は読み取りと更新、監査と権限は未実装）。どの段階も「宣言 → 実行時の値の結合 → 照会」の三段で作り、各段階の試験記録を`plan/evidence/`へ残しています。

| 段階 | 実装 | 結合（Data宣言の属性役割） | 確認記録 |
| --- | --- | --- | --- |
| 1. 暦の宣言と休日Data | `internal/syntax/semantic_calendar.go`、`internal/validation/calendar.go`、`internal/businesscalendar/holiday_binding.go` | `business-day excludes <data>`（暦が休日Dataを名指す）、`holiday-date`（Dataが日付属性を名指す） | [暦の宣言と期間への結合](../evidence/XTEA_LOWCODE_CALENDAR_DECLARATION_2026-11.md) |
| 2. 期間との結び付き | `period`の`calendar`、`internal/businesscalendar` | 期間が従う暦を名指す（未知・別名前空間は拒否） | [暦からの日付導出](../evidence/XTEA_LOWCODE_CALENDAR_CALCULATION_2026-11.md)、[期間キーの導出](../evidence/XTEA_LOWCODE_PERIOD_KEY_2026-11.md) |
| 3. 税 | `internal/taxrate` | `tax-applies-from`・`tax-applies-until`・`tax-category`・`tax-value`・`tax-rounding` | [税率の適用期間と端数処理](../evidence/XTEA_LOWCODE_TAX_RATE_2026-11.md)、[Data結合](../evidence/XTEA_LOWCODE_TAX_RATE_BINDING_2026-11.md) |
| 4. 通貨 | `internal/currencyrate`（年次フォールバックは`currency-rate-fallback`の宣言で選ぶ） | `currency-rate-currency`・`-fiscal-year`・`-month-sequence`・`-middle`・`-buying`・`-selling`・`-fallback` | [通貨換算の率の種類と方向](../evidence/XTEA_LOWCODE_CURRENCY_RATE_2026-11.md)、[Data結合](../evidence/XTEA_LOWCODE_CURRENCY_BINDING_2026-11.md) |
| 5. 採番 | `internal/numbering`（`ReserveInTransaction`で取引に参加） | `numbering-counter`・`numbering-digits`・`numbering-prefix`・`numbering-check`・`numbering-overflow` | [採番の契約](../evidence/XTEA_LOWCODE_NUMBERING_2026-11.md)、[Data結合](../evidence/XTEA_LOWCODE_NUMBERING_BINDING_2026-11.md) |
| 6. 利用者設定 | `internal/usersetting`（`Resolve`・`Bind`・`Settings.Value`・`Table.Change`・`Table.Apply`） | `setting-key`・`setting-value`・`setting-user` | [読み取りの確認](../evidence/XTEA_LOWCODE_USER_SETTINGS_2026-11.md)、[書込みの確認](../evidence/XTEA_LOWCODE_USER_SETTINGS_WRITE_2026-11.md) |

三段の分け方には理由があります。

- **宣言**はWorldに置き、受理の対象にします。X-TEAは表名・システム変数・列名を実行時に解決したため、宣言と実行の対応を読む人が確かめられませんでした。
- **結合**は、宣言された役割から保存行を実行時の値へ写します。行の解釈失敗（非数値・解釈できない日付・非正の率）は、その行と属性を名指して報告し、飛ばしません。飛ばせば、欠落が後段の別の場所で「率が無い」として現れ、原因を追えなくなります。
- **照会**は、結合済みの値だけを入力にします。ここへ既定値を混ぜないので、宣言の欠落が既定値で隠れません。

## 5.1 期間キーの導出（実装済み）

`internal/businesscalendar`は、業務日付から**期間キー**（会計年度のラベルと、開始月を1とする月順序）を導きます。`period key`が年月なら、この組が閉めの単位になります。

X-TEAの`getMSeqOfDate`（`D:Session.java L2371–2476`）は、締め日の設定（`LAST_DAY`）より後の日付を**翌月の順序**へずらし、`getFYearOfDate`（同 L2440–2507）も同じ理由で年度を変えていました。TEIの導出はその規則を引き継ぎません。日付は、その日付が名指す月に属します。月内の締め（20日締めなど）を期間にしたい場合は、期間が`closes on day N`を宣言します。日付は月から動かず、月の前半と後半が`PeriodKey.ClosingSegment`で分かれます（`internal/businesscalendar/closing.go`）。

## 6. 未確認の事項

- 税率表・為替表・採番表の実スキーマ（列名は`DTSTART`・`VLTAXRATE`・`NRCURRENT`等として読めますが、制約・索引・整合は未確認）です。
- `getDaysBetweenDates`の`countType`の全分岐（負の向き、夏時間、`kbCalendar`の既定）は、静的な読解の範囲です。
- `ALLOWED_FISCAL_MONTH`の使われ方の全量（締めの許容範囲がどの操作に効くか）は未確認です。
- `getNextNumber`の`FGWITHCD`の全値、`countUpNumber`の桁上限（9桁）を超える場合の扱いは未確認です。
- 配布JAR（Driver M36）と固定ソースの同一性は未確認です。
- 実DBでの税額・換算・採番の結果は、[#940](https://github.com/hat47x/tei-platform/issues/940)が担当します。

### 6.1 参照実装で未実装のまま残したこと

| 残り | 理由 | 何が分かれば決まるか |
| --- | --- | --- |
| 月内締めの閉めの実行内容 | 状態遷移（`open`→`closing`→`closed`）・記録（`periods -advance`）・`close-requires`と`correction`の実行時検査は実装したが、対象データの集計・繰越・帳票、訂正能力を走らせる経路は未実装 | 閉めを Command にするか、運用操作として分けるか |
| 採番の例での規則行データ | 例のDataに`numbering-rule`の役割を足したが、行は試験で与えている | 例へ行データを置くかどうかの判断 |
| 為替の粒度を宣言必須にするか | `ResolveTable`が宣言と行を照合し、空の表は拒否する。宣言は任意のままなので、宣言の無い表は行だけに頼る | 宣言を必須にするか、任意のままにするか |
| 番号のPostgreSQL実装 | SQLiteの永続格納と2プロセス同時実行の不変条件（同一番号を出さない・カウンタが発行数に一致する）は確認したが、PostgreSQLは未実装 | PostgreSQLを先にするか、多プロセス構成を広げるか |
| 利用者設定の監査と権限 | 更新の検査と記録（`Table.Change`・`Table.Apply`）は実装したが、変更履歴と「誰が変えられるか」は未実装 | 設定更新をCommandにするか、運用操作として分けるか |

実装が進んだことで、以前ここに挙げていた項目のうち次の3つは解消しました。

- **Workspace例への税・通貨・採番Dataの追加**: `examples/xtea-lowcode`へ役割付きで追加済み（[確認記録](../evidence/XTEA_LOWCODE_EXAMPLE_STANDARD_DATA_2026-11.md)）。
- **為替の年次・月次の優先順位**: `currency-rate-fallback`役割で宣言する形にした（[確認記録](../evidence/XTEA_LOWCODE_CURRENCY_FALLBACK_ROLE_2026-11.md)）。
- **採番の発行と取消の取引結合**: `ReserveInTransaction`で呼出し側の取引と同じ境界に入れた（[確認記録](../evidence/XTEA_LOWCODE_NUMBERING_RESERVATION_TRANSACTION_2026-11.md)）。

## 7. 関連

- [X-TEA知見の受容アーキテクチャ](XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)（D6、D6.1）
- [期間状態の設計](PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md)
- [知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §4.6
- [評価時点の決定性](EVALUATION_CONTEXT_DETERMINISM_2026-10.md)
- [移行計画](../workstreams/XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md)（WP8）
- [受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)（RQ-C3）











