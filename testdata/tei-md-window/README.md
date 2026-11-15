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

## 固定した負例も、検証対象から消さない

5件の固定入力について、今度は**どの事例を検証するかという集合そのもの**も回帰テストで確認する。元ファイルのSHA-256が一致していても、候補JSONから重複IDの拒否例を削除したり、`valid: true`へ変更したりすると、以前のGoテストでは基準とFrame変更の2件だけが成功し、負例の欠落を見落とす余地があった。

`window_candidate_oracle_test.go`では、5件のファイル名と件数、各入力の期待状態、正例7行の結果、`layout-only`が基準と同じ意味であること、重複IDが生成拒否候補であること、古い派生表示が不整合であることを**Fixtureの閉包**として固定する。対応する`*.tei.md`の追加・削除も検出する。3つの負例（ケースを削除、重複IDを有効化、古い表示を一致扱い）を用意し、いずれも閉包検査が拒否することを確かめる。

また、`window_sourceindex_bridge_test.go`で、既存Graph／Ruleと同じ`internal/sourceindex`のArtifact・Range・Validateを用い、5入力の元SHA-256と、明示NULL行を含む既知のバイト範囲を照合する。説明文の追記だけで古いSource Indexが失効する負例も確認する。ただし、文字列の位置が正しいことと、その文字列を正式なWindow観測として意味受理することは別である。

**これは期待結果候補の分類が勝手に変わっていないことの検査**であり、重複IDや派生表示を正式Readerで検出できたことではない。将来のReader検査では、Sourceのbyte位置と、選択した定義表、表示・例示だけの領域の違いも確認する。Code FenceやKaTeXの数式表示を、明示されていない業務規則へ昇格させない。これらは既存Language／Operator担当への受入条件であって、数理Toolkit専用Parserを先に作る条件にはしない。

## 入力行と計算結果を別々に照合する

`independent-expected.candidate.json`を`v2`へ更新し、計算結果だけでなく、7つの観測について`obs_id`・系列`grp`・順序`seq`・入力`value`の候補期待値も独立して記録する。B2の`value: null`は、`value`項目自体がない場合や数値の0とは異なる。Goの回帰では`json.RawMessage`で**項目欠落／明示NULL／整数**を区別し、非整数や意図しないNULLも拒否する。さらに、`2^53`を超える整数`9007199254740993`を`int64`で正確に保持できることを確認し、指数表記`1e2`を受入済み整数へ勝手に正規化しない。これはテスト候補のデコード精度であり、SQLiteの巨大整数集計やTEI標準数値型の精度保証ではない。

既存のSQLite方言が生成したSQLを実行した後、**計算結果の7行×3種類のWindow値に加えて、入力行そのものの系列・順序・値・NULLも照合**する。SQL側で系列を取り違えたり、別の数値を読んだりしても、平均が偶然同じなら成功という判定を避ける。

さらに、5つのMarkdown候補それぞれについて、手書きの入力候補と対応する**既知の観測行の原文を一度ずつ含むこと**をbyte位置付きで検査する。重複IDの負例には追加のA3行が一つあることも検査する。これは固定された原文の字句照合であり、Markdown表・コードFence・引用を解釈する正式Readerではない。字句が一致するだけで入力をMeaningに受理しない。

`v2`への更新はExpected候補の内部整理にとどまり、`adoption=proposal-only`と#1180／#2514の正式Owner判断は維持する。`v1`／`v2`の成功をTEIの製品Schema・正式DSL・SQLへの自動Loweringと呼ばない。

## 実DBの入力も、固定候補から渡して比較する

`window_bound_input_test.go`は、候補JSONの七行を実SQLiteの`observations`表へプレースホルダー付きで挿入し、既存SQLite方言が生成したWindow計算SQLを実行する。基準の直前三行と、変更後の直前二行の両方で、各観測行の系列・順序・値・NULLと三種類のWindow結果を独立Expectedに照合する。静的な`VALUES`句だけを実行した結果とは検証範囲を分ける。

基準計算の成功後、実表のA3を30から300へ変更する。生成SQL自体は変えずに再実行し、A3の移動平均が20から110に変化することを検査する。この変更は**未受理の意図的な負例**であり、元のMarkdownやExpectedの書換え、意味受理、モデル改版として扱わない。SQLの構造が同じでも、読んだデータ版と結果が異なることを具体的に観測する。

現行の`Source`→`JSON候補`は正式Readerで選択・受理された経路ではなく、`SQLite`はTEIの通常Provider／CLI／Studio Consumerではない。型付き入力候補から実DBへ渡す段階までがC0であり、これが成功しても`*.tei.md`の受理・標準QueryのWindow意味・製品セルフビルドは未成立である。

## Workspaceが選択した原文のS0-P予備試験

`internal/workspace/tei_md_window_pinned_source_test.go`は、固定された基準・Frame変更の二文書を、既存`tei.workspace.yaml`の`semantic`管理Rootへ配置する。宣言外の`drafts`は管理対象に含めず、Pinned Rootから実際に読んだ元バイトを`sourceSHA256`と`Source Index`で照合する。B2の明示NULL行と、`moving` Frameの行を既知の字句としてバイト位置まで確認し、A3を書き換えれば旧Indexが失効することも検査する。

**この試験は、通常の`Workspace.Load`が`TEI_MD_PARSER_UNBOUND`を返すことを前提にしている。** 管理対象として見つかっただけでは、Markdown表や計算条件がTEIの正式な意味として受理されたことにはならない。実DBへ固定JSONの七行を渡す`window_bound_input_test.go`とも、まだ製品Reader経由では接続していない。

七行の閉包は固定テストデータとしての検査であり、外部Sourceの読み取り完了や全Datasetの完全性を示すものではない。未読データ・未選択の表・部分取得があれば、正式な入力Datasetとしての受理は保留する。新しいWindow Reader・Source Index・Expected正本は作らない。

## 拒否される文書もSourceの取得と診断位置を失わない

S0-PのWorkspace試験は、基準・Frame変更の二文書から、**候補五文書すべて**へ対象を広げる。重複A3の観測行と、古いmoving表示の行は、いずれも候補Sourceの一部として元byte位置を保持する。管理外の`drafts`には別内容を置き、選択対象の原文と取り違えた場合もSHA-256照合で検出する。候補が管理対象から漏れたり、未宣言の草稿を原文に取り違えたりしても、計算テストだけが合格する構成にはしない。

ただし、Source Indexのbyte位置が正しいことは、重複IDの意味上の拒否や、派生表示の不一致を製品Readerが判断した証拠ではない。現行`Workspace.Load`はどの候補も`TEI_MD_PARSER_UNBOUND`で拒否し、読み取った原文から勝手にMeaning／実Queryを生成しない。正式Readerは既存#2621/#988の担当へ引き渡す。

## 既存の実証との接続

`internal/database/sqlgen/sqlite/window_reference_test.go`と`window_frame_change_test.go`では、既存`SelectOptions`から生成したSQLite SQLを実DBで評価している。固定された七行の入力と期待値の照合は**C0／SQL方言**の検証である。今回追加するテストも、このJSONの基準・変更例を使ったC0の照合にとどまる。

**現行Workspace Loaderは`*.tei.md`を正式に解釈できず**、管理Root内で未束縛なら`TEI_MD_PARSER_UNBOUND`とする。Markdown内のテーブルをGoテスト側が勝手に構文解釈し、製品Readerの成功とする実装は置かない。今後のReader・Role・Operatorは[#2621](https://github.com/hat47x/tei-platform/issues/2621)／[#988](https://github.com/hat47x/tei-platform/issues/988)／[#2620](https://github.com/hat47x/tei-platform/issues/2620)の既存責務で実装する。

正式Readerが成立した後は、元byte・表のCell／Source occurrence・意味Revision→受理されたWindow条件→既存SQL方言／実Provider→同一Attemptの実結果と独立Expectedへつなぐ。表の行を単にNative SQLとして貼り付けた実装や、Goにすべてを委譲するラッパーはC1／C2合格としない。Sourceの改版・現在の開示・Plugin版・Build／Evidence再利用は別に判断する。

本Fixtureは数理Toolkit全体の完成や新しい数理用DSL、専門計算Runtime、独立のExpected SSOTを意味しない。全体のセルフビルド最初の題材はGraph／Ruleを優先し、Windowはその後の用途別検証とする。
