# TEI Markdown：Window平均の固定入力と独立Expected候補

これは[Graph／Ruleの初回固定Fixture](../tei-md-graph/README.md)に続いて、統計Windowという別分野でも**Markdown上の定義元と、派生した表示・説明を区別**できるかを検証するための資料です。入力は手書きの`*.tei.md`、期待値はコード生成に先立って固定した`independent-expected.candidate.json`です。正式な検収正本は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)が所有し、このJSONの`adoption=proposal-only`を変更する権限はありません。

| 入力候補 | 一つの差分 | 合否の要点 |
| --- | --- | --- |
| `baseline.tei.md` | 七行と直前二行＋現在行の平均 | B2のNULL、B2/B3同順位、A3=20/A4=30/B3=10 |
| `frame-narrowed.tei.md` | movingのみ直前一行＋現在行に変更 | A3=25/A4=35/B3=15。累積ROWS/RANGEは変わらない |
| `layout-only.tei.md` | 説明段落の配置だけ変更 | Source byteは異なるが、Windowの意味と期待結果はbaselineと同じ |
| `duplicate-observation-id.tei.md` | A3を別の値で重複記述 | IDの重複を黙って統合せず、意味受理前に拒否する |
| `derived-view-stale.tei.md` | narrowedの定義に対し旧movingの派生表を表示 | 定義元を表示結果で上書きせず、不一致を診断する |

## 既存の実証との接続

`internal/database/sqlgen/sqlite/window_reference_test.go`と`window_frame_change_test.go`では、既存`SelectOptions`から生成したSQLite SQLを実DBで評価している。固定された七行の入力と期待値の照合は**C0／SQL方言**の検証である。今回追加するテストも、このJSONの基準・変更例を使ったC0の照合にとどまる。

**現行Workspace Loaderは`*.tei.md`を正式に解釈できず**、管理Root内で未束縛なら`TEI_MD_PARSER_UNBOUND`とする。Markdown内のテーブルをGoテスト側が勝手に構文解釈し、製品Readerの成功とする実装は置かない。今後のReader・Role・Operatorは[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#988](https://github.com/hat47x/tei-platform/issues/988)／[#2620](https://github.com/hat47x/tei-platform/issues/2620)の既存責務で実装する。

正式Readerが成立した後は、元byte・表のCell／Source occurrence・意味Revision→受理されたWindow条件→既存SQL方言／実Provider→同一Attemptの実結果と独立Expectedへつなぐ。表の行を単にNative SQLとして貼り付けた実装や、Goにすべてを委譲するラッパーはC1／C2合格としない。Sourceの改版・現在の開示・Plugin版・Build／Evidence再利用は別に判断する。

本Fixtureは数理Toolkit全体の完成や新しい数理用DSL、専門計算Runtime、独立のExpected SSOTを意味しない。全体のセルフビルド最初の題材はGraph／Ruleを優先し、Windowはその後の用途別検証とする。
