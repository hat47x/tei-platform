# ISSUE-000085: TEIが出すsha256の値ごとに、ハッシュ対象のbyte規則を宣言する

- 状態: Proposed — 評価待ち。現状の読解と宣言の案までを記録した段階で、実装は始めていない
- 提案日: 2026-10-09
- 性格: Requirement詳細化 / 他製品との境界
- 主担当要求: RQ-021（主）/ RQ-027 / RQ-003
- 関連Issue: [ISSUE-000040](000040-semantic-component-composition.md)（受理と決定記録）、[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)（受入境界）、[ISSUE-000071](000071-approval-sufficient-projection.md)（承認に足るProjection）
- Gate impact: None

## この文書の役割

TEIは`sha256:<64桁の16進>`という同じ形の文字列を、性質の違う複数の値に使っています。形が同じでも、何のbyte列にSHA-256を掛けたかは値ごとに違います。この規則はコードを読まないと分かりません。

このIssueは、値ごとの規則を一か所に宣言することを求めます。値そのものは変えません。他製品の規則への合わせ込みも、ここでは決めません。

## 背景

2026-10-09時点のコード（`main`の`fbcf4b8f`）を読むと、Shared Acceptance Boundaryだけで、`sha256:`の値が7種あります。ハッシュ対象のbyte列の作り方は、4つの系統に分かれます。

| 系統 | 値 | 実装 | ハッシュ対象 |
| --- | --- | --- | --- |
| A. byte列そのもの | source fingerprint | `internal/sourceindex/model.go`の`Fingerprint` | ファイルのbyte列 |
| B. Goの`json.Marshal` | candidate fingerprint | `tools/reference-shared-acceptance/candidate.go`の`fingerprintOf` | `baseModelRevision`とdeltaを直列化したbyte列 |
| B | evidence fingerprint | 同`evidenceFingerprintOf` | Evidenceの一覧を直列化したbyte列。一覧が空なら空文字 |
| B | review fingerprint | `review.go`の`fingerprintReviewProjection` | review Projectionから`ReviewFingerprint`を空にして直列化したbyte列 |
| C. インデント付きの`Encoder` | 導出したWorldのダイジェスト | `decision.go`の`worldSHA256` | Worldを、2スペースのインデントと末尾の改行つきで直列化したbyte列 |
| C | 決定記録のfile名 | 同`recordDecision` | 記録本体を同じ形で直列化したbyte列。file名は、そのfileの内容のSHA-256に一致する |
| D. 独自の連結 | model revision | `snapshot.go`の`revisionOf` | 管理対象sourceを相対path順に並べ、`path`、NUL、source fingerprint、改行を連ねたbyte列 |

系統Bは、構造体のフィールド宣言順に出力し、`<`、`>`、`&`をエスケープします。系統Cも同じエスケープをしたうえで、インデントと末尾の改行を足す形です。どちらも、キーを整列するJCS（RFC 8785）ではありません。

決定記録のfile名がその内容のSHA-256に一致することは、`decision.go`の`loadDecisionRecords`が、読むたびに検査しています。この系統Cの1件は、すでにコードで守られています。

Shared Acceptance Boundaryの外にも、`sha256`を計算する箇所が、`tools/`に少なくとも20あります。規則は、このIssueでは調べていません。

他製品も同じ形の値を持ちます。SEIは、byte列そのものに掛ける`sei.raw-bytes+sha256/v1alpha1`と、RFC 8785のJCSで正規化してから掛ける`sei.jcs+sha256/v1alpha1`を、契約として定めています（`hat47x/sei-cognition:contracts/profiles/`）。MMIは、キーを整列した詰め書きのJSONに改行を1つ足して掛ける独自の規則を持ちます。MMIの文書は、同じオブジェクトでも製品によってダイジェストが変わると書いています（`hat47x/markdown-matrix-injector:02_Architecture/structured-artifact-neutral-observation-v0.md` §13）。

TEIの系統と照らすと、次のとおりです。

- 系統Aは、SEIの`raw-bytes`と同じ規則です。
- 系統BとCは、JCSではありません。
- 系統Dは、JSONではない独自の連結規則です。

これらの値は、witness記録やreview文書へ、そのまま固定されています。たとえば第二Applicationの受理は、candidate fingerprint `sha256:8a936cf9…`で識別されており、規則を変えると、過去の記録が指す識別子が変わります。

## 防ぎたい失敗

- 見た目が同じ`sha256:`を、製品をまたいで比べる。一致すれば同じ内容だと読み、食い違えば別の内容だと読んでしまう。
- 同じTEIの中でも、系統の違う値を混同する。たとえば、review fingerprintをsource fingerprintと同じように、ファイルのbyte列から再計算しようとする。
- 将来TEIの値をSEIのexact bindingなどへ渡すとき、受け手が規則を知らず、検証に失敗しても原因を特定できない。
- 再入場したAIが記録のダイジェストを検証しようとして、規則をコードから逆算するしかない（RQ-027）。

## 守る原則

- 既存の値は変えません。過去のwitnessとgoldenが指す識別子を書き換えないためです。
- 製品間で比較できるかどうかを、値の見た目から推定しません。比べるときは、規則を名指しします。
- TEIは、SEIのprofileに依存しません。製品の親和性は依存ではない、というSEI側の原則（`hat47x/sei-cognition:product/definition/SEI_PRODUCT_SHAPE.md` §6.4）と、TEI側の立場は一致しています。TEIが持つのは対応表だけです。

## 決めること

1. 宣言の置き場所。仮に、`tools/reference-shared-acceptance/README.md`へ表を置く案を採ります。記録のschemaは変えません。将来、値を別の製品へ渡す事例が出たら、値の隣に規則名を持たせる案を再検討します。
2. 他製品の規則への対応表を、いつ作るか。仮に、実際に値を渡す事例が出るまで作りません。
3. 製品間で渡す値だけ、JCSなど別の規則で計算し、別の識別子（たとえば別のprefix）を付けるか。仮に、事例が出るまで決めません。既存の値を書き換える形は取りません。
4. Shared Acceptance Boundaryの外にある`tools/`の計算箇所を、どこまで宣言の対象にするか。仮に、他製品や次の世代の記録へ渡る可能性のある値に限ります。

## 完了条件

- [ ] 上の表の7種について、実装位置とハッシュ対象を、宣言の置き場所へ移した
- [ ] 宣言が実装と食い違わないことを確かめるtestを置いた。決定記録のfile名は検査済みなので、系統A、B、Dの各値について、宣言どおりに独立に再計算した値が実装と一致することを確かめる
- [ ] 決めること1〜4を、採用・修正・却下のいずれかに判定し、理由を残した

## 非主張範囲

- TEIのダイジェストが誤っているとは主張しません。値ごとに規則が違うことを宣言していないだけです。
- SEIのJCS profileへ合わせるべきだとは主張しません。
- 現時点で、TEIの値を他製品へ渡す経路はありません。
- SEIとMMIの規則は、各リポジトリの文書を読んだ結果です。それぞれの実装は実行していません。
- Shared Acceptance Boundaryの外にある`tools/`の計算箇所の規則は、調べていません。
