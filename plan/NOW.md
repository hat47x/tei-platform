# いま進めること

- **状態**：決定（2026年11月17日、所有者から全面的に委任を受けた判断）。優先順位の正本はこの文書である。[全体計画](MASTER_PLAN.md)はこれまでの経緯の記録として残し、優先順位を知るためには読まない。
- **決まり**：この文書は100行以内に保つ。足すときは、終わったものを消す。

## なぜ絞るか

2026年11月17日の時点で、直近に統合した239件の変更のうち、文書だけの変更が122件、利用者が使う機能を足した変更が18件だった。計画と説明の文書は合わせて約27MBある。一方、[実装状況](../docs/status.md)は「一つの機能を実際のDBから読んで利用者へ届ける」経路を未成立としている。調べて書き分ける力は十分にある。足りないのは、利用者が最後まで通せる一本である。

## 到達点は一つ：受注の一変更

**TEIを初めて使う人が、`tei`のコマンドと実際のPostgreSQLだけで、次の五つを通せること。**

1. 参照用の作業領域で、受注一件の明細を、宣言したQueryを通して実際のDBから読む。
2. 読んだ明細の一行ずつに、設計文書へ書いた規則の答え（成立、不成立、判定できない）が出る。
3. 設計文書（`*.tei.md`）の規則を一つ書き換え、近くに例を一行足す。
4. `tei validate`が通り、同じ明細を読み直すと、書き換えた規則の答えに変わる。
5. 生成したWebの画面一つで、同じ答えが見える。

入力と、手で書いた期待は[`testdata/destination-order-change/`](../testdata/destination-order-change/README.md)にある。**1〜4は動く。** `tei view -environment`が実PostgreSQLの明細を読み、設計文書の規則を変更した前後の答えを独立した期待と照合する。5も、出荷するのと同じ起動関数に`-serve-read-from-db`を渡し、実PostgreSQLの明細と行別Ruleの判定を第一WebのHTTP画面へ返すところまで動く。規則変更の前後も、第一WebのHTTP画面で比べる試験が実PostgreSQLのCIで走る（HTTP本文は[#5482](https://github.com/hat47x/tei-platform/pull/5482)、実Chrome DOMの変更前後は[#5492](https://github.com/hat47x/tei-platform/pull/5492)で確認した）。

受入れは二つである。この五つを実際のPostgreSQLで実行し、実装から独立した期待と照合する試験が、CIで毎回走ること。五つを一ページで案内する手順書があり、その手順書のとおりに打てば通ること。

この五つが通ったら、同じ変更を通常の開発で行った場合と、触ったファイルの数と手数を比べる。TEIの価値はその比較で述べる。

## そこへの道

上から順に進める。下の作業は、上の作業が受入れを通るまで広げない。

| 順 | 作業 | 担当先 |
| --- | --- | --- |
| 1 | ~~異なる構造の業務モデルでも同じ読取入口を通す~~ **動く。** 別のData・Relation・実テーブルに加え、別の属性・Query入力（`batch_key`）・複合主キー（`entry_id, slot`）を持つ独立モデルを、同じ`tei view`と独立した期待で実DBから読む試験がCIで走る（[#5362](https://github.com/hat47x/tei-platform/pull/5362)） | [#2586](https://github.com/hat47x/tei-platform/issues/2586) |
| 2 | **動く（CLI）：** 実PostgreSQL→`tei view -environment`→静的HTML→実Chrome DOMで、主キーごとのRule三値を独立Expectedと照合する（[#5425](https://github.com/hat47x/tei-platform/pull/5425)）。**動く（第一Web TargetのHTTP）：** `-serve-read-from-db`と`TEI_WEB_READ_PG_DSN`で選択した実Providerの明細と判定を同じReaderから返す（[#5431](https://github.com/hat47x/tei-platform/pull/5431)・[#5439](https://github.com/hat47x/tei-platform/pull/5439)）。HTTP画面の明細は、行番号ではなく業務主キー・値・Rule回答で照合する。**試験だけ：** 合成Queryの画面試験。**動く：** 規則変更の前後を、実PostgreSQLの同じHTTP経路で独立した期待と照合する（[#5482](https://github.com/hat47x/tei-platform/pull/5482)）。**動く（第一Webの実Chrome DOM）：** 同じHTTP経路で規則変更前後をそれぞれ画面で確かめ、主キーと独立Expectedで照合する（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。**次の受入：** 手順書に従う五段階の実利用と、通常開発との比較 | [#1119](https://github.com/hat47x/tei-platform/issues/1119) |
| 3 | **動く：** 参照Workspaceが`covers sales.order.lines`を宣言し、実行Queryと関係の保存を照合できた場合だけ`coverage: complete`とする（[#5507](https://github.com/hat47x/tei-platform/pull/5507)、[#5512](https://github.com/hat47x/tei-platform/pull/5512)）。受注1002の一意は`satisfied`、重複する1001は`not-satisfied`、行を落とすQueryは`partial`に留める。宣言やSQLの静的一致だけを実DBの全件取得の証拠にしない。**同じReadのトランザクションで実テーブルのRLS・継承・種類を確認し、不可視行の可能性を排除できない場合は`partial`と理由へ戻す**（[#5525](https://github.com/hat47x/tei-platform/pull/5525)）。実行時の可視範囲とOwnerによる業務意味の受理は区別する。[判断の文書](design/RULE_NORMAL_FORM_AND_OPERATOR_SIGNATURE_DECISION_2026-11.md)§6 | #2586 |
| 4 | **動く：** `tei desired-schema -format sql`によるDDL生成（[#5385](https://github.com/hat47x/tei-platform/pull/5385)）、`storage decimal(12,3)`の精度指定とDDL反映（[#5411](https://github.com/hat47x/tei-platform/pull/5411)）、生成DDLの実PostgreSQL適用から同じCLIによる読取・独立期待との照合（[#5422](https://github.com/hat47x/tei-platform/pull/5422)）、`tei view -environment -format text`の行別表示（[#5372](https://github.com/hat47x/tei-platform/pull/5372)）。**動く：** 精度を指定した列が実DBでその桁数で作られ、その列から読んだ行のRuleの答えが独立した期待と一致する（`TestDeclaredDecimalPrecisionSurvivesToTheDatabaseAndBack`） | #2586 |
| 5 | 規則の正規形とOperatorの署名。判断の文書の作業1〜4、6〜8 | [#4362](https://github.com/hat47x/tei-platform/issues/4362) |

道の1は[#5327](https://github.com/hat47x/tei-platform/pull/5327)（固定名の決め打ちを外す）、[#5339](https://github.com/hat47x/tei-platform/pull/5339)（Data・Relationの意味IDと参照テーブルを変える）、[#5362](https://github.com/hat47x/tei-platform/pull/5362)（別の属性・Query入力・複合主キーを持つ独立モデル）で通った。#5362は`checkDestinationAlternateModel`を`TestDestinationOrderChangeReadViaCLIFromPostgres`の中から呼び、CIが実PostgreSQLで走らせる。

道の2は、実PostgreSQLの明細を第一Web TargetのHTTP画面へ渡し、規則変更の前後を実Chrome DOMで照合するところまで動く（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。[#5510](https://github.com/hat47x/tei-platform/pull/5510)では、DDL適用からCLI読取り、規則・例の編集、`tei validate`、再読取り、画面確認までを**一つのCI試験で順に実行**した。ただし編集操作はGoのテストヘルパーによるものだった。その後、[#5521](https://github.com/hat47x/tei-platform/pull/5521)でローカルCLIの起動、試行ごとの新しい作業領域、Webフォームの入力を手順書と揃え、[#5523](https://github.com/hat47x/tei-platform/pull/5523)では作業領域の複製と規則編集を**手順書から読み取ったシェルコマンドで実行**し、`tei validate`まで確認した。[#5529](https://github.com/hat47x/tei-platform/pull/5529)でSeed投入時のSQLエラーも停止するようにした。これらは有効な別々の検証だが、**初めて使う人が手順0〜5をすべて記載どおりに完走した証拠ではない**。その後、[#5532](https://github.com/hat47x/tei-platform/pull/5532)でDDL生成の失敗を見逃さない手順と原子的なDB投入を整え、[#5538](https://github.com/hat47x/tei-platform/pull/5538)では**手順0の実CLI・実`psql`・実PostgreSQL**で正常投入と途中失敗時のDDLごとのロールバックを検査した。[#5539](https://github.com/hat47x/tei-platform/pull/5539)では、第一Webが必要とする親受注3件と明細5行を公開手順のSeedに揃え、五段階E2Eのテスト専用表への差し替えを廃止した。その後、[#5546](https://github.com/hat47x/tei-platform/pull/5546)で、両変更を統合したmainの試験に欠けていた親受注Seedを補い、親受注3件・明細5行の確定と途中失敗時の両テーブルのロールバックを実PostgreSQLで再検証した。これらの試験は**動く**が、初回利用者が手順0〜5を逐語実行した証拠ではない。次は一つの試行で手順0〜5を確認し、成立後に通常開発との変更ファイル数・操作手数を比較する。複数HTTP要求が同一Snapshotだったという主張もしない。

**道の2で守る対応関係**：表示行番号は読取時の位置であり、業務キーやノードの意味IDではない。行別Ruleの答えは実際の明細行と同じReadで作り（[#5424](https://github.com/hat47x/tei-platform/pull/5424)）、画面とCommandは選択した同じReaderを使う（[#5439](https://github.com/hat47x/tei-platform/pull/5439)）。実DBのHTTP画面は主キー`order_line_id`ごとに値とRule三値を独立Expectedで検査し、同じ`line_number`の二行を混同しない。**第一WebのRelation全体Ruleの全件性判定も、実PostgreSQLで動く**（[#5571](https://github.com/hat47x/tei-platform/pull/5571)）。明細Queryの結果と同じ読取専用Repeatable Readトランザクション内で、宣言・実行Query・保存定義の対応に加え、物理表の種類・RLS・継承を確かめ、資格を得た対象Queryだけを`complete`とする。受注1002の一意性は成立、重複を含む1001は不成立、RLSで不可視行の可能性がある場合や別Provider・別Readは理由付きの`partial`に留める。CLIの別試行の証拠を流用せず、二つのHTTP要求が同一Snapshotだったとも主張しない。**次の検収**は[`docs/first-change.md`](../docs/first-change.md)のコマンドと編集を初回利用者の順序で0〜5まで実行し、実PostgreSQLと実Chromeで独立Expectedに照合すること。手順と製品にずれがあればその箇所を修正し、通過後に通常開発と変更ファイル数・操作手数を比較する。新しいRuntimeや一般Rule Engineは増やさない。

入口は増やさない。接続の秘密は環境変数から受け取り、作業領域へ書かない。

並行して進めてよいのは一本だけである。**Studioが次の自分を作る一巡**（[#4583](https://github.com/hat47x/tei-platform/issues/4583)）。完了の条件は一文にする。Studioで定義を一つ変え、**生成器には触れずに**、Studio自身が作った次のStudioがWindowsで起動し、その変更が画面に出ること。

「生成器には触れずに」が要である。[#5516](https://github.com/hat47x/tei-platform/pull/5516)・[#5522](https://github.com/hat47x/tei-platform/pull/5522)で、`tei.authoring/v1`のData定義から生成Readerの必須応答項目を導出し、実`InspectSubject`の文字列項目に限定した。[#5534](https://github.com/hat47x/tei-platform/pull/5534)はData必須指定の変更→実Go Generator→生成Reader→実esbuild→受理・拒否の動作差を局所検査した。[#5542](https://github.com/hat47x/tei-platform/pull/5542)と[#5555](https://github.com/hat47x/tei-platform/pull/5555)で、Stage2の初回InspectとGraph選択後に同じReader契約を使い、呼出回数が増えるという判定と独立した負例を復旧・固定した。[#5556](https://github.com/hat47x/tei-platform/pull/5556)は実生成Readerを正規WorkbenchのNode/esbuild子プロセスへ接続し、[#5560](https://github.com/hat47x/tei-platform/pull/5560)ではStage1・Stage2それぞれの実TEIが`self-build -child`でそのNode Builderを起動し、**同じ固定SourceからReader→新JS Bundleを生成する工程**まで検証した。従来の[#5373](https://github.com/hat47x/tei-platform/pull/5373)は旧Bundleを使うWindows Stage2実行である。これらを足して、変更後のDataから作った新BundleをWindows Stage2が使用した一続きの検収としない。

**次の製品コード変更は[#4583](https://github.com/hat47x/tei-platform/issues/4583)のN2→N3接続とWindows実使用に絞る。** 隔離した同一`tei.authoring/v1`のData定義を一つ変更し、Stage1の実Generatorから得たReader Partを、実TEIが起動する既存N2 Node/esbuild Childへ渡す。生成された**新Bundleのbyte列そのもの**を隔離したWails/GoのN3 Build入力に渡し、Stage1が実際に起動したBuilderでStage2 exeを構築する。そのexeを別プロセスのWindows WebView2で起動し、初回InspectとGraph選択後のReader契約・呼出増加を同一HEAD・Source版・Attemptで照合する。Part／Bundle／埋込／exeのdigestと実Invokerが違えば拒否する。**DOMに記録したReader digestの変化は画面本文の変化ではない。** 利用者に見えるInspectorの表示または既存の可視検証結果が実際に変わったことを、別の独立Expectedで確認する。旧Bundle使用、異なるSource、不可視のメタデータだけの変化を完成扱いしない。正式Build Intentの工程選択・依存閉包、Studio Editorによる実Source編集・保存、Ownerの意味受理はそれぞれ別検収。新しいHost・Reader正本・Builder・DSLは増やさない。

## 到達点まで止めるもの

到達点の五つに載っていない領域では、新しい設計文書、調査、検証記録の変更を出さない。気づいたことは、該当する課題へコメント一つで残す。止めるのは次のような領域である。組織の高度なモデル化、消費者向けの画面設計、科学計算と立体表示、多言語対応の拡張、二つ目の実行先、性能の自動調整、応答の判定グラフの拡張、Studioの子プロセス・プロセス間通信・WASMの比較、外部の作業基盤を使った委任の実験。

止めることは、捨てることではない。到達点を通った後、利用者が次に困る順に再開する。

## 変更の決まり

1. **変更は二種類だけにする。** 利用者が打つコマンドの結果を変える変更（コードと、実装から独立した期待）。または、その変更を直接止めている判断を一つ決める文書。
2. **文書だけの変更は、次に出るコードの変更を名指しする。** 名指しできない文書は出さない。
3. **検証の記録や調査の経過を、リポジトリへ足さない。** 変更の説明に書く。段階判定が求める記録だけを置く。
4. **状態は三つの言葉で書く。** 「動く」（実際のDBか実際の画面で確かめた）、「試験だけ」（合成の入力や代役で確かめた）、「未実装」。これより細かい区別は、必要な箇所の本文で述べ、状態の語を増やさない。
5. **mainで失敗している検査を、ほかの統合より先に直す。** 見つけた人が担当の課題へ知らせる。道に載る検査なら、直るまでその領域の統合を止める。統合の前には、自分の変更が起こした失敗がないことを、失敗した検査の名前で確かめる。
6. **全体計画へ追記しない。** 優先順位が変わるときは、この文書を書き換える。

## 比べるのをやめて決めたこと

到達点を通るまで、次の選択は比べ直さない。

| 関心事 | 決めたこと |
| --- | --- |
| DB | サーバーはPostgreSQL、手元はSQLite |
| 実装の言語 | GoとTypeScript |
| 設計文書 | `*.tei.md`の`tei`フェンスと、印を付けた表。言語は`tei.authoring/v1`一つ。`tei.rule-authoring/v0`へ機能を足さない |
| 値の受渡し | 型の情報を持つJSON。十進数は文字で渡す |
| Studio | Wailsの一体型。画面は標準のWeb ComponentsとCodeMirror |
| 生成するWebの画面 | 標準のHTML、CSS、Web Components |
| AIの開発環境 | 通常のCLI、Git、エディター。TEI専用の接続を必須にしない |

## この判断を改める条件

- 到達点の五つのどれかが、設計の誤りで通せないと分かった。通せない理由をこの文書へ書き、道を書き換える。
- 道の先頭の作業が、ほかの作業を止めたまま進まない。作業領域を一つ選び直し、選び直した理由を書く。
- 所有者が到達点を変える。