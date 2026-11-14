# TEI Markdown：Window平均の固定入力と独立Expected候補

これは[Graph／Ruleの初回固定Fixture](../tei-md-graph/README.md)に続いて、統計Windowという別分野でも**Markdown上の定義元と、派生した表示・説明を区別**できるかを検証するための資料です。入力は手書きの`*.tei.md`、期待値はコード生成に先立って固定した`independent-expected.candidate.json`です。正式な検収正本は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)が所有し、このJSONの`adoption=proposal-only`を変更する権限はありません。

| 入力候補 | 一つの差分 | 合否の要点 |
| --- | --- | --- |
| `baseline.tei.md` | 七行と直前二行＋現在行の平均 | B2のNULL、B2/B3同順位、A3=20/A4=30/B3=10 |
| `frame-narrowed.tei.md` | movingのみ直前一行＋現在行に変更 | A3=25/A4=35/B3=15。累積ROWS/RANGEは変わらない |
| `layout-only.tei.md` | 説明段落の配置だけ変更 | Source byteは異なるが、Windowの意味と期待結果はbaselineと同じ |
| `duplicate-observation-id.tei.md` | A3を別の値で重複記述 | IDの重複を黙って統合せず、意味受理前に拒否する |
| `derived-view-stale.tei.md` | narrowedの定義に対し旧movingの派生表を表示 | 定義元を表示結果で上書きせず、不一致を診断する |

## SourceとExpectedの対応を固定する

`independent-expected.candidate.json`の各ケースに、**対応する`*.tei.md`の元UTF-8バイト列のSHA-256**を`sourceSHA256`として記録した。SQL生成器のGo回帰テストは、期待値を読む前に全5件のSourceの実バイト列を照合する。Sourceが変更された場合は、数値結果が偶然一致していても`STALE_SOURCE_CLOSURE`として停止する。

これは正式Readerに代わるものではない。ハッシュは**Sourceが同じかどうか**を示すだけで、どのMarkdown表が意味として選択されたか、NULLやFrameが正しく解釈されたか、意味変更か表示変更か、Ownerが受理したかを判定しない。表示のみの変更でもraw byteのSHA-256は変わるため、Sourceの再確認は必要となる。意味が同じことを確認できた範囲でのBuild／Artifact再利用は、既存のWorld／Model／Artifact Revisionと現在の資格で別に判定する。

Sourceを修正する場合、ハッシュの再計算は必要だが**ハッシュだけ自動更新して合格扱いにしない**。定義と派生表示の意味差、Independent Expectedの正しさ、対象Role／Operator、変更を受理するOwnerの判断を確認し、変更と再確認の記録を残す。JSONの`adoption=proposal-only`は保持し、数理Toolkit専用のScenario SSOTや製品Schemaへ昇格しない。

## 既存の実証との接続

`internal/database/sqlgen/sqlite/window_reference_test.go`と`window_frame_change_test.go`では、既存`SelectOptions`から生成したSQLite SQLを実DBで評価している。固定された七行の入力と期待値の照合は**C0／SQL方言**の検証である。今回追加するテストも、このJSONの基準・変更例を使ったC0の照合にとどまる。

**現行Workspace Loaderは`*.tei.md`を正式に解釈できず**、管理Root内で未束縛なら`TEI_MD_PARSER_UNBOUND`とする。Markdown内のテーブルをGoテスト側が勝手に構文解釈し、製品Readerの成功とする実装は置かない。今後のReader・Role・Operatorは[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#988](https://github.com/hat47x/tei-platform/issues/988)／[#2620](https://github.com/hat47x/tei-platform/issues/2620)の既存責務で実装する。

正式Readerが成立した後は、元byte・表のCell／Source occurrence・意味Revision→受理されたWindow条件→既存SQL方言／実Provider→同一Attemptの実結果と独立Expectedへつなぐ。表の行を単にNative SQLとして貼り付けた実装や、Goにすべてを委譲するラッパーはC1／C2合格としない。Sourceの改版・現在の開示・Plugin版・Build／Evidence再利用は別に判断する。

本Fixtureは数理Toolkit全体の完成や新しい数理用DSL、専門計算Runtime、独立のExpected SSOTを意味しない。全体のセルフビルド最初の題材はGraph／Ruleを優先し、Windowはその後の用途別検証とする。
