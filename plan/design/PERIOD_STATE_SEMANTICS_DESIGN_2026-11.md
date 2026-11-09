# 業務日付・会計期間の状態・確定した行の不変性（設計候補）

- 状態: **提案中（未採用）**。中核の意味を増やす提案であり、採用は Owner の判断（[AGENTS.md](../../AGENTS.md) の A4）に当たる。
- 対応課題: [#2144](https://github.com/hat47x/tei-platform/issues/2144)。担当課題が無かった要件 **P-TIME-01・P-TIME-02・P-TIME-04** を引き受ける。
- 関連: #1727（P-TIME-03、有効時間と記録時間）、#1720（P-TIME-05、暦）、#1129（効果規則）、#1130（締めの実行計画）、#2147（段階的な移行）
- 正本: 実行時の時点の扱いは[副作用の設計](BUSINESS_EFFECT_LIFECYCLE_DESIGN_2026-10.md) 2.1節（効果記録の順序付けの鍵）と[時間意味の区別](TEMPORAL_MEANING_DISTINCTION_2026-10.md)（D14）が持つ。本稿は**期間の状態**と**確定した行の不変性**を足す。
- 根拠: 技術検証用サンプル11（月次締め）が、要件をサンプル側の実装で満たし、基盤側は未着手と記録している（[4.11節の改善圧力](TECH_VALIDATION_SAMPLES_2026-10.md)）。

## 1. 解決したい問題

月次締めの現場では、次の問いに答えられなければならない。

1. 月末23時59分に受け付けた受注は、どちらの月に入るのか（**P-TIME-01**）。
2. 締めた期間の伝票に誤りが見つかったとき、締めた帳票を変えずにどう直すのか（**P-TIME-02**）。
3. 締めの後に届いた前月分の実績を、どの期間に計上するのか（**P-TIME-04**）。

現在のTEIは、この三つに答えられない。`data` に「計上日」の意味がなく、期間という対象がなく、締め済みの行を守る規則がない。サンプル11はサンプル内の PostgreSQL の表とコードで答えを出したが、意味の層には戻っていない（[サンプル11のREADME](../../reference/month-end-close-correction/README.md)）。

この状態のまま締めの実行計画（#1130）を作ると、締めの正しさがコードの中に閉じる。期間の状態を意味の層に置く理由はここにある。

## 2. 区別する四つの時間

時間の混同は、この領域で最も多い事故の原因である。四つを別の概念として扱う。

| 概念 | 意味 | 持ち主 |
| --- | --- | --- |
| 業務日付（`business date`） | 業務上、その記録が属する日付。計上日 | この文書 |
| 記録時刻（`recorded at`） | システムがその記録を知った時刻 | `std.datetime` の属性（既存） |
| 評価時点（`as of`） | 過去のどの時点の知識で評価するか | #1727（D14） |
| 会計期間（`period`） | 締めの単位。状態を持つ | この文書 |

四つを一つの `timestamp` に畳まない。`asOf` だけで期間の締めを表そうとすると、「いつ締めたか」と「どの時点の知識か」が分からなくなる。

## 3. 宣言の候補

### 3.1 業務日付

```tei
data sales.invoice {
    attribute invoiceId type sales.invoice_id
    attribute issuedOn type std.date
    business-date sales.invoice.issued_on
    period sales.accounting_period
}
```

- `business-date` は、そのデータの記録が属する日付を決める属性を一つ指定する。
- `period` は、業務日付から期間を決める規則を指定する。
- 業務日付を持たないデータは、期間の締めの対象にならない（締めの集計に入れない。黙って含めない）。

### 3.2 会計期間

```tei
period sales.accounting_period {
    key sales.period.year_month
    states open, closing, closed
    close requires capability sales.close_period
    closed rejects update, delete
    correction via capability sales.red_black_amend
    late-arrival hold
}
```

| 要素 | 意味 |
| --- | --- |
| `key` | 期間を識別する属性（年月、年度＋四半期など） |
| `states` | 期間の状態。`open`（受付中）、`closing`（締め中）、`closed`（締め済み）を必須とし、追加の状態は許さない（追加すると、締めの意味が製品ごとに変わる） |
| `close requires` | 締めを起動できる能力。権限の判定は既存の `authorization` が持つ |
| `closed rejects` | 締め済みの期間で拒否する操作（`update`／`delete`）。`create` は既定で許す（翌期の計上として扱う）か拒否するかを選ぶ |
| `correction via` | 訂正の経路。赤黒の訂正を行う能力を指定する |
| `late-arrival` | 締め後に届いた記録の方針。`hold`（保留）／`current`（当月に計上）／`reopen`（再オープン。権限と監査を要求） |

### 3.3 遷移

```text
open ──close の起動──▶ closing ──確定──▶ closed
  ▲                        │                 │
  └──── 中止 ──────────────┘                 │
  └──────────── reopen（権限と監査）──────────┘
```

- 遷移は、起動した能力、実行者、時刻、対象の期間を記録する。
- `closing` から `open` へ戻せるのは、締めが確定する前だけである。`closed` からの `reopen` は別の能力とし、記録を残す。
- 状態の変更は、操作面の設計でいう「人の判断を含む操作」なので、既定では `unattended` にしない。

## 4. 締めの一貫点

締めは「ある時点を切り出す」操作である。切り出した時点の前後で、集計と明細が一致しなければならない。

- 締めの対象は、`(期間, 切り出しの時点)` で決まる。切り出しの時点は締めの起動時に一度だけ決め、途中で読み直さない。
- オンラインの更新は止めない。締めの実行中に確定した記録は、切り出しの時点より後として扱い、次の期間または保留へ回す。
- 集計と明細の一致は、締めの確定の条件にする。一致しなければ `closed` にしない。
- 切り出しの時点を、DBの現在時刻から暗黙に取らない。宣言された時計（評価文脈、#1720）から取る。

## 5. 確定した行の不変性と赤黒の訂正

`closed` の期間に属する行は、`update` と `delete` を拒否する。

- 拒否は、診断（受理前）と実行時（確定境界）の両方で行う。診断だけに頼ると、期間の状態が後から変わった場合に抜ける。
- 訂正は、元の行を書き換えず、**取消（赤）**と**再計上（黒）**の二つの記録で表す。どちらも元の行を参照し、訂正の理由と、訂正を計上する期間を持つ。
- 締め時点の帳票は、締めの記録（切り出しの時点と実行の識別子）から再生成できる。訂正の後でも、締め時点の値は変わらない。
- 取消と再計上は、一つの能力の中で両方を行うか、二つの能力に分けるかを宣言で選ぶ。途中で片方だけが確定しないように、同じ確定境界に置く（副作用の設計7章）。
- `reopen` は期間を開き直す操作であり、過去の判断を有効に戻す操作ではない。判断の有効性（いつ・誰が・何を根拠に決めたか）はSEIが持ち、TEIは期間の状態と行の不変性だけを持つ（[TAI上流への立ち戻り](../workstreams/TAI_UPSTREAM_ALIGNMENT_2026-11.md)）。

## 6. 遅延到着と順序

**P-TIME-04** に答える。

- 業務日付と受信時刻を区別する。遅れて届いた記録は、業務日付が過去でも、受信時刻は現在である。
- 遅延到着の方針（`late-arrival`）に従い、`hold` は保留の置き場へ、`current` は当期の記録として、`reopen` は権限を確かめて期間を開き直す。
- 同一の業務対象に対する更新の順序は、対象ごとの順序の鍵と版で決める。受信順で上書きしない。版が古い更新は拒否し、成功として記録しない。
- 締め済みの期間に属する記録の到着は、`hold` を既定とする。既定を `current` にすると、締めた帳票と当期の集計が黙って食い違う。

## 7. 受入

| 番号 | 状況 | 合格の条件 |
| --- | --- | --- |
| P-2144-01 | 締めの集計中に、前月分の訂正と当月の新規受注を差し込む | 締めの結果と明細の合計が一致し、境界の受注が業務日付の規則どおりの月に入る |
| P-2144-02 | 締め済みの伝票を更新しようとする | 拒否され、赤黒の訂正として当期に計上される。締め時点の帳票は変わらない |
| P-2144-03 | 事象の順序を入れ替え、一部を締め後に届ける | 順序どおりに適用した場合と同じ最終状態になり、締め済みの値が無断で変わらない |

| 番号 | 負例 | 期待する結果 |
| --- | --- | --- |
| N-2144-01 | `closed` の期間の行を `delete` する | 診断と実行時の両方で拒否する |
| N-2144-02 | 締めの実行中に期間を読み直す | 切り出しの時点が変わったとして停止する（成功にしない） |
| N-2144-03 | `reopen` を権限なしで行う | 拒否し、監査の記録を残す |
| N-2144-04 | 業務日付を持たないデータを締めの集計に入れる | 入れない。件数が合わない場合は締めを確定しない |
| N-2144-05 | 古い版の更新を、受信順を理由に適用する | 拒否する。後勝ちにしない |
| N-2144-06 | 赤のみ確定し、黒が確定していない状態を訂正済みと表示する | 表示しない。片方だけの確定を「訂正済み」に丸めない |

検証コマンド。

```sh
go test ./internal/validation -run 'Period' -count=1
go run ./cmd/tei validate -workspace reference/period/workspace
go test ./tools/reference-period-state -count=1
```

## 8. 作業分解

| # | 作業 | 変更する場所 | 完了条件 | 規模 |
| --- | --- | --- | --- | --- |
| W1 | `business-date`・`period` の構文 | `internal/syntax/period.go`（新規）、`parser.go` | 正負の構文試験。`business-date` の属性が同一データに属するかの検査 | M |
| W2 | 意味モデルと正規化 | `internal/semantic/`、`internal/ir/` | 期間の識別、状態、遷移の能力が決定的に正規化される | M |
| W3 | 診断（受理前） | `internal/validation/period.go`（新規） | N-2144-01・04 を診断で止める。業務日付の欠落を検出する | M |
| W4 | 実行時の拒否（確定境界） | `internal/driver/`、`internal/database/` の既存の境界 | `closed` の行への update／delete を確定境界で止める結合試験 | L |
| W5 | 赤黒の訂正 | `tools/reference-period-state/`（新規） | 取消と再計上が同一の確定境界で確定し、片方だけを訂正済みと表示しない | M |
| W6 | 切り出しの時点 | 同上、`reference/derivation-context`（参照） | 締めの実行中に時点を読み直すと停止する | S |
| W7 | 遅延到着の方針 | 同上 | `hold`／`current`／`reopen` の三つが、同じ入力から宣言どおりに分岐する | M |
| W8 | 締めの実行計画との接続 | #1130 の `run-plan` | 月次締めの実行計画が、期間の状態遷移と結び付く | S |

W1・W2 が前提。W3 と W4 は別の層なので並行できる。W4 の規模が大きいため、最初は PostgreSQL の一つの提供元に限定する。

## 9. 停止条件と Owner 判断

これは中核の意味を増やす提案である。次のいずれかに当たるため、**採用の判断は Owner が行う**。

1. 意味の受理（A4）。
2. 既存の `data`／`persist` の宣言に、新しい任意要素を足すこと。
3. 締め済みの更新拒否を、実行時の境界へ入れること（既存の実行経路の挙動が変わる）。

実装は隔離ブランチで参照実装まで進め、採用前に `main` へ統合しない。採用しない場合の代替は、次のとおり。

- **代替A（Plugin／Semantic Extension）**: 期間の状態と赤黒を、案件固有の拡張として表現する。中核を変えずに済むが、製品間で意味が揃わない。
- **代替B（実行時のみ）**: サンプル11のように、実行時の表とコードで期間を扱う。意味の層では持たない。

停止条件。

- 期間の状態を、案件ごとに別の語彙で持った場合は停止する。
- 締めの切り出しの時点を、DBの現在時刻から暗黙に取った場合は停止する。
- `closed` の拒否を、診断だけで実装した場合は停止する。

## 10. 効果と撤回

- 効果: 締めの正しさが、コードではなく意味から説明できる。#1130の締めの実行計画と、#1803の締め時点の帳票が、同じ期間の状態を参照できる。
- 観測: 締めの実行で拒否された更新の件数、赤黒の訂正の件数、締め時点の帳票の再現一致率、`hold` に回った遅延到着の件数。
- 撤回: 参照実装で `reopen` が一度も使われず、`hold` だけで運用できる場合は、`reopen` を落とす。期間の状態そのものが案件ごとに異なり、共通の三状態で表せない場合は、代替Aへ戻す。

## 11. 未決

1. `business-date` を `data` の宣言にするか、`persist` の列の属性にするか。前者は意味、後者は保存の都合に近い。
2. 期間の状態を、`period` の宣言（本稿）にするか、`capability` の状態遷移（[BUSINESS_LIFECYCLE_V1](BUSINESS_LIFECYCLE_V1_DESIGN.md)）の一例として表すか。後者なら新しい宣言を足さずに済む可能性がある。
3. `closed` の期間での `create` を許すか（翌期への計上）。本稿は `late-arrival` と合わせて選ぶ案にしている。
4. 締めの切り出しの時点を、評価文脈（#1720）と期間の状態のどちらが持つか。
## 15. 実装計画（2026-11-03、仮承認D-2に基づく）

期間の状態は意味の層に閉じます（D-2 仮承認）。CLI の段階は増やしません。次の順で、段階ごとに試験を通します。

| 段階 | 範囲 | 変更する場所 | 完了条件 |
| --- | --- | --- | --- |
| P1 | モデル・構文・検証 | `internal/semantic`（`Period` 型、パーサ）、`internal/validation` | `period` を宣言でき、対象データ・業務日付属性の実在と、`open`・`closing`・`closed` の三状態を検査する。負例（未定義のデータ、日付でない属性、状態の欠落）で診断が出る |
| P2 | 操作面との接続 | `internal/operations` | `closed` の期間を持つデータは `create`・`update` を `human-gated` にし、`retire` の未解決を期間の宣言で解消する。決定性を保つ |
| P3 | 成果物への反映 | `internal/artifact` | 機能定義書に期間の行を出し、整合検査（`CheckSet`）の対象に加える |

### P1 の形（案）

```text
period sales.month {
    data sales.order
    business-date sales.order.order_date
    states open, closing, closed
    close-requires sales.period.close
    closed-rejects create, update
    correction via sales.period.correct
}
```

- `states` は三状態をすべて含むことを要求する（順序は問わない）。
- `business-date` は対象データの属性で、型が日付（`std.date` 系）であることを要求する。
- `close-requires`・`correction` は能力の識別子で、同じ名前空間に実在することを要求する。
- `closed-rejects` は操作種別（`create`・`update`・`delete`）の列挙とし、未知の種別を拒否する。

### 各段階の検証

```sh
go test ./internal/semantic ./internal/validation -count=1   # P1
go test ./internal/operations -count=1                        # P2
go test ./internal/artifact ./internal/operations -count=1    # P3
go run ./cmd/tei validate -workspace reference/workspace      # 既存の作業領域を壊さない
```

### 戻る条件

- P1 で、既存の `data`・`attribute` の宣言と衝突する（同じ名前・同じ位置に別の意味を置く必要が出る）場合は、構文の案を設計へ戻す。
- P2 で、`human-gated` にすることが実行の拒否と混同される場合は、自律区分の意味を設計へ戻す。
### P1 の実装状況（2026-11-03）

P1（モデル・構文・検証）を実装しました。

- `semantic.Period` と `Namespace.Periods` を追加（`internal/semantic/model.go`）。
- `period` ブロックを解析する（`data`・`business-date`・`states`・`close-requires`・`closed-rejects`・`correction via`）。一覧は空白区切りで、次のキーワードで止まる（`internal/syntax/parser.go`、`period_test.go`）。
- 検証を追加（`internal/validation/period.go`）: 対象データの実在、業務日付が対象データの属性であること、`open`・`closing`・`closed` の三状態、`close-requires`・`correction` の能力実在、`closed-rejects` の操作種別。診断コードは `PERIOD_UNKNOWN_DATA`・`PERIOD_UNKNOWN_BUSINESS_DATE`・`PERIOD_STATES_INCOMPLETE`・`PERIOD_UNKNOWN_CAPABILITY`・`PERIOD_UNKNOWN_CLOSED_REJECT`。

検証: `go test ./internal/syntax ./internal/validation ./internal/semantic ./internal/operations` が通り、`validate -workspace reference/workspace` も `ok`（期間を宣言していない作業領域に影響しない）。

**未着手**: P2（操作面との接続）と P3（成果物への反映）。
### P2 の実装状況（2026-11-03）

P2（操作面との接続）のうち、**期間の参照と閉じた期間が拒否する操作の提示**を実装しました。

- 操作面の各操作に `period`（期間の識別子）と `periodRejects`（閉じた期間が拒否する操作）を付ける。期間はデータに紐づくため、`list`・`read`・`create`・`update` のすべてに付く（`internal/operations/surface.go`）。
- 期間を宣言していないデータは空のまま（既存の作業領域に影響しない）。
- 自律区分は変えない。**閉じた期間は実行時に拒否される**という事実を、操作面に載せる形にした（`human-gated` に読み替えない。自律区分は「誰が判断するか」、期間の状態は「いつ実行できるか」で、別の軸のため）。

**未着手**: 閉じた期間の拒否を実行時に強制する仕組み（P2 の残り）と、P3（成果物への反映）。
### P3 の実装状況（2026-11-03）

P3（成果物への反映）のうち、**機能定義書への期間の行**を実装しました。

- `FunctionDefinitionInput` に `Periods` を追加し、`uses semantic` の名前空間の期間を `period <id> data <data> states <open,closing,closed>` の行で出す（`internal/artifact/functiondef.go`）。
- 生成規則（`function-definition/v1`）の行種別に `period` を追加。並びは期間の識別子順。
- 作業領域からの一括生成で、名前空間の期間を渡す（`internal/artifact/workspace.go`）。
### P2 の追補（2026-11-03）: 書き込みの前提

閉じた期間が拒否する操作（`closed-rejects` に `create`・`update` がある場合）には、操作面の `preconditions` に `period:<期間>:open` を加えました。実行する側は、期間が開いていることの証明を渡さなければなりません。**実行時に拒否を強制する仕組みはまだありません**（契約としての前提のみ）。
### 実行時の拒否（2026-11-03 追記）

前提を検査する関数を追加しました（`internal/operations/period.go` の `CheckPeriods`）。

- 入力: 操作面と、期間ごとの実行時の状態（`open`・`closing`・`closed`）。
- 出力: 実行してはいけない操作の一覧（`PeriodRejection`）。**拒否であって実行エラーではない**。
- 期間が `open` なら拒否なし。`closing`・`closed` は、その期間が `closed-rejects` で宣言した操作だけを拒否する。
- 状態が分からない期間（未指定）は拒否しない。**不明を「閉じている」とみなさず、判断を呼び出し側に残す**。

期間の状態は受理済みの意味ではなく実行時の状態なので、状態の置き場所（誰が・いつ記録するか）は別途決めます。この関数は、その置き場所が決まったときに実行の入口から呼べる形です。
### 作業領域の読み込み（2026-11-03 追記）

期間は `semantic.Namespace` の項目として読み込まれます（`internal/workspace/loader.go`）。読み込みの直後に、**期間が IR に載っていない**ことが分かったため修正しました。`tei periods` が空を返す症状で気づいた不具合です。再発を防ぐため、`internal/workspace/period_load_test.go` で「読み込んだ世界に期間と拒否操作が残る」ことを固定しています。
### 実行入口のガード（2026-11-03 追記）

能力の呼出し直前に期間を確かめる関数を追加しました（`internal/operations/periodguard.go` の `GuardCapability`）。**driver ではなく operations に置いています**。driver から operations を参照すると、Driver の依存閉包が広がり、`tools/reference-first-web-target/sharedprojection` の閉包試験が失敗したためです（束縛の知識は操作面が持つ、という層の分け方にも合います）。

- 入力: 操作面、期間の状態、呼び出そうとしている能力の識別子。
- 出力: `(PeriodRejection, bool)`。**拒否は値として返し、エラーにしない**（業務上の結末として利用者へ返すため）。
- 束縛（`binding`）がこの能力を指し、期間を持つ操作だけを対象にする。期間を持たない能力はガードしない（操作面が知らないため）。

期間の状態の**出所**（誰が・いつ記録するか）は、まだ決めていません。現状は呼び出し側が渡します（CLI の `-period-states` も同じ）。出所が決まるまでは、状態を渡さない限り拒否しない方針を保ちます。
### 状態の出所（2026-11-03 追記）

期間の状態は受理済みの意味ではないため、**出所を呼び出し側から注入する**形にしました（`internal/operations` の `PeriodStateSource`）。

- `PeriodStates(ctx, applicationID)` を実装する任意の出所を渡せる。今日は運用の記録や CLI の指定、将来は外部システムや観測の記録。
- 出所が状態を返さない期間はガードしない（不明を「閉じている」とみなさない方針を維持）。
- 出所のエラーは**エラー**として返す。拒否（業務上の結末）と、判断できなかったことを区別する。
- 出所が nil の場合はガードしない（既存の呼び出しに影響しない）。

**未着手**: 実際の出所（運用の記録をどこへ置くか）と、実行入口での呼び出し。
### 出所の実装（2026-11-03 追記）

最初の出所として、**運用者が置く JSON ファイル**を実装しました（`internal/periodstate` の `File`。CLI とツールの両方から使えるよう、`cmd/tei` から切り出しました）。

```json
{ "sales.month": "closed", "sales.week": "open" }
```

- `tei periods -workspace <root> -application <id> -period-state-file <path>` で読み込む
- 未知の状態（`open`・`closing`・`closed` 以外）と、読めないファイルは**エラー**（拒否に丸めない）
- ファイルに無い期間はガードしない（不明を「閉じている」とみなさない）
- `-period-states` と併用した場合は、ファイルの値を優先する

これは運用の記録の最小形です。運用の記録をどこに置くか（リポジトリ外の管理、観測との接続）は、実データでの確認と合わせて後続で決めます。
### 実行入口の合成（2026-11-03 追記）

`internal/execution` に、能力の呼出しと期間のガードを合成する関数を追加しました（`execution.Invoke`）。

- ガードが拒否した場合、**driver を呼ばずに**拒否を返す（`*operations.PeriodRejection`）。
- 出所のエラーはエラーとして返し、driver を呼ばない（判断できていない状態で実行しない）。
- 出所が nil ならガードせず、そのまま実行する（既存の呼び出しに影響しない）。
- この層は driver の外に置いています。driver が操作面へ依存すると依存閉包が広がるためです（`sharedprojection` の閉包試験が失敗した経緯）。
### 通しの確認（2026-11-03 追記）

作業領域のテキストから拒否までを通す試験を追加しました（`internal/execution/period_guard_workspace_test.go`）。

1. `period` と `effect create` の能力、`persist` を持つ作業領域を読み込む
2. 操作面が `create` を導出し、期間と拒否操作を載せ、能力へ束縛する
3. 期間が `closed` のとき、`GuardCapabilityWith` が拒否を返す
4. `execution.Invoke` は**driver を呼ばずに**拒否を返す

これで「宣言 → 操作面 → 束縛 → 実行入口での拒否」が、ライブラリの範囲で一本につながりました。**実装の入口（実際の能力呼出しを行うツール）への適用と、実データベースでの確認は未着手**です。
### 実行者ツールへの適用（2026-11-03 追記）

`tools/reference-data-first-persistence` に、期間のガードを前置できる提出経路を追加しました（`submitIfReadyGuarded`）。

- 既存の `submitIfReady` はガード無しのまま（挙動を変えない）。
- ガードが拒否した場合、**driver を呼ばず**、既存の拒否の形（`stageResult{Stage: "period", Status: unresolved}`）で返す。ツールに別のエラー経路を増やさないため。
- ガードの出所のエラーは、既存の `Err` として返す。
- 拒否の発生源はまだ CLI から指定できない（状態ファイルの受け口は `tei periods` 側にある）。運用の記録をこのツールへ渡す配線は後続。