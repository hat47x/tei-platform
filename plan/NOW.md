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

道の2は、実PostgreSQLの明細を第一Web TargetのHTTP画面へ渡し、規則変更の前後を実Chrome DOMで照合するところまで動く（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。[#5510](https://github.com/hat47x/tei-platform/pull/5510)では、DDL適用からCLI読取り、規則・例の編集、`tei validate`、再読取り、画面確認までを**一つのCI試験で順に実行**した。ただし編集操作はGoのテストヘルパーによるものだった。その後、[#5521](https://github.com/hat47x/tei-platform/pull/5521)でローカルCLIの起動、試行ごとの新しい作業領域、Webフォームの入力を手順書と揃え、[#5523](https://github.com/hat47x/tei-platform/pull/5523)では作業領域の複製と規則編集を**手順書から読み取ったシェルコマンドで実行**し、`tei validate`まで確認した。[#5529](https://github.com/hat47x/tei-platform/pull/5529)でSeed投入時のSQLエラーも停止するようにした。これらは有効な別々の検証だが、**初めて使う人が手順0〜5をすべて記載どおりに完走した証拠ではない**。その後、[#5532](https://github.com/hat47x/tei-platform/pull/5532)でDDL生成の失敗を見逃さない手順と原子的なDB投入を整え、[#5538](https://github.com/hat47x/tei-platform/pull/5538)では**手順0の実CLI・実`psql`・実PostgreSQL**で正常投入と途中失敗時のDDLごとのロールバックを検査した。[#5539](https://github.com/hat47x/tei-platform/pull/5539)では、第一Webが必要とする親受注3件と明細5行を公開手順のSeedに揃え、五段階E2Eのテスト専用表への差し替えを廃止した。これらはそれぞれ**動く**が、足し合わせても初回利用者が手順0〜5を逐語実行した証拠ではない。次は一つの試行で手順0〜5を確認し、成立後に通常開発との変更ファイル数・操作手数を比較する。複数HTTP要求が同一Snapshotだったという主張もしない。

**道の2で守る対応関係**：表示行番号は読取時の位置であり、業務キーやノードの意味IDではない。行別Ruleの回答は実際の明細行と同じReadから作る（[#5424](https://github.com/hat47x/tei-platform/pull/5424)）。Readの対象Query・親Scope・入力と実行Providerを区別し、画面とCommandが同じ選択済Readerを使う（[#5439](https://github.com/hat47x/tei-platform/pull/5439)）。ただし、二つのHTTP要求やReadが同一スナップショットであるとの証拠にはしない。実DBのHTTP画面では主キー`order_line_id`ごとに値とRule三値を独立Expectedで検査し、同じ`line_number`の二行を混同しない。**次の検収**は、[`docs/first-change.md`](../docs/first-change.md)のコマンドと編集手順を実際にたどり、初めて使う人が実PostgreSQLで五段階を通せるか確かめること。製品と手順が食い違えばその箇所を修正し、成立後に通常開発と変更ファイル数・操作手数を比較する。取得範囲を証明できない全体Ruleは、理由付きの「判定できない」を保つ。CLIはQuery・保存定義の照合に加え、同じ読取トランザクションで実テーブルのRLS・継承・種類を確認してから全件性を認定する（#5525）。一方、第一Web TargetのRelation全体Ruleは`CoveragePartial`固定であり、同じ受注でも全件性が必要な答えは一致しない場合がある。次のコード変更は、第一Webの`read_scenario_postgres.go`で**当該明細Queryと同一の実行トランザクション**内に物理表の可視性と取得範囲の資格を束縛し、`read_scenario_answers.go`のRule評価へ渡すこと（#1119・#2586）。業務対象集合・実行ロールの閲覧可能集合・実際に取得した集合を混同せず、RLSや別Read／別Providerの証拠なら`partial`と理由を保つ。標準Graphや新しいRuntimeは前提にしない。

入口は増やさない。接続の秘密は環境変数から受け取り、作業領域へ書かない。

並行して進めてよいのは一本だけである。**Studioが次の自分を作る一巡**（[#4583](https://github.com/hat47x/tei-platform/issues/4583)）。完了の条件は一文にする。Studioで定義を一つ変え、**生成器には触れずに**、Studio自身が作った次のStudioがWindowsで起動し、その変更が画面に出ること。

「生成器には触れずに」が要である。[#5325](https://github.com/hat47x/tei-platform/pull/5325)で生成Readerを製品の唯一のReaderへ戻し、[#5365](https://github.com/hat47x/tei-platform/pull/5365)で未対応の宣言を拒否した。[#5516](https://github.com/hat47x/tei-platform/pull/5516)では`tei.authoring/v1`のData定義から必須項目をReader契約へ導出し、[#5522](https://github.com/hat47x/tei-platform/pull/5522)ではその項目を実`InspectSubject`応答型の必須トップレベル文字列に限定した。未知名・非文字列・任意項目・別Bridgeは生成前に拒否する。[#5373](https://github.com/hat47x/tei-platform/pull/5373)では生成されたStage1がStage2のWindows Desktopを構築・起動し、別プロセスのWebView2で生成Readerの使用を実測した。ただし、これは**変更前の契約による一試行**である。[#5541](https://github.com/hat47x/tei-platform/pull/5541)でDataの必須項目変更に伴う生成Partの差を合成契約から判断する**自己試験**が増えたが、実`tei.authoring/v1` Sourceの変更をStage1→新Bundle→Stage2実Windows WebView2へ渡したことは、まだ実証していない。Source位置・指紋・Model Revision・入力契約digest、Builderと実Consumerの識別を別々に残す。

**次のコード変更は[#4583](https://github.com/hat47x/tei-platform/issues/4583)の一巡を一つの変更・一つのWindows試行で確かめる。** `tei.authoring/v1`の既知の文字列項目について必須指定を変更し、生成器本体のソースは変更せず、旧Readerの契約digestが更新後契約と一致しないこと、変更後Readerが必須属性欠落を拒否して実Host応答を受け入れることを独立Expectedで照合する。その変更後Sourceから生成Part→正規Node/esbuild Bundle→Wails埋込→Stage1自身が起動する実Builder→Stage2 exe→**そのexeを別プロセスで起動したWebView2の実Inspector**まで、同一HEAD・Attemptで因果関係を記録する。Node/esbuildの直接起動を正式Build Intent経由と偽らず、未接続の依存閉包・Builder選択は未実装のまま示す。Editorからの実Source編集／保存／意味受理は別に検収し、テスト用Source変更だけで「Studioで編集した」とは認めない。旧#5373のStage2起動と新#5522の型適合を足して完成としない。新しいHost・Reader正本・Builder・DSLは増やさない。

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