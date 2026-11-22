# いま進めること

- **状態**：決定（2026年11月17日、所有者から全面的に委任を受けた判断）。優先順位の正本はこの文書である。[全体計画](MASTER_PLAN.md)はこれまでの経緯の記録として残し、優先順位を知るためには読まない。
- **決まり**：この文書は100行以内に保つ。足すときは、終わったものを消す。

## なぜ絞るか

2026年11月17日の時点で、直近に統合した239件の変更のうち、文書だけの変更が122件、利用者が使う機能を足した変更が18件だった。計画と説明の文書は合わせて約27MBある。一方、[実装状況](../docs/status.md)は「一つの機能を実際のDBから読んで利用者へ届ける」経路を未成立としている。調べて書き分ける力は十分にある。足りないのは、利用者が最後まで通せる一本である。

## 到達点は一つ：受注の一変更

**TEIを初めて使う人が、`tei`のコマンドと実際のPostgreSQLだけで、次の五つを通せること。**

1. 参照用の作業領域で、受注一件の明細を、宣言したQueryを通して実際のDBから読む。
2. 読んだ明細の一行ずつに、設計文書へ書いた規則の答え（成立、不成立、判定できない）が出る。
3. 設計文書（`*.tei.md`）に単価を確かめる規則を一つ追加し、近くに例を添える。
4. `tei validate`が通り、同じ明細を読み直すと、新しい規則の答えが加わる。
5. 生成したWebの画面一つで、同じ答えが見える。

入力と、手で書いた期待は[`testdata/destination-order-change/`](../testdata/destination-order-change/README.md)にある。**1〜4は動く。** `tei view -environment`が実PostgreSQLの明細を読み、設計文書の規則を変更した前後の答えを独立した期待と照合する。5も、出荷するのと同じ起動関数に`-serve-read-from-db`を渡し、実PostgreSQLの明細と行別Ruleの判定を第一WebのHTTP画面へ返すところまで動く。規則を追加する前後も、第一WebのHTTP画面で比べる試験が実PostgreSQLのCIで走る（HTTP本文は[#5482](https://github.com/hat47x/tei-platform/pull/5482)、実Chrome DOMの変更前後は[#5492](https://github.com/hat47x/tei-platform/pull/5492)で確認した）。

受入れは二つである。この五つを実際のPostgreSQLで実行し、実装から独立した期待と照合する試験が、CIで毎回走ること。五つを一ページで案内する手順書があり、その手順書のとおりに打てば通ること。

この五段階は、初回の**規則追加**を検収する。公開手順で触るのは、言語宣言・規則の設計文書・Applicationの利用登録の3ファイルである。既存の規則と例を書き換える二回目以降の作業は、別に検収する。

初回利用者がこの五段階を完走した後、同じ規則追加を通常の開発で行う場合と、変更したファイル数や手数を比べる。初回と二回目以降の作業量を混ぜず、[#2018](https://github.com/hat47x/tei-platform/issues/2018)で結果を検証する。

## そこへの道

上から順に進める。下の作業は、上の作業が受入れを通るまで広げない。

| 順 | 作業 | 担当先 |
| --- | --- | --- |
| 1 | ~~異なる構造の業務モデルでも同じ読取入口を通す~~ **動く。** 別のData・Relation・実テーブルに加え、別の属性・Query入力（`batch_key`）・複合主キー（`entry_id, slot`）を持つ独立モデルを、同じ`tei view`と独立した期待で実DBから読む試験がCIで走る（[#5362](https://github.com/hat47x/tei-platform/pull/5362)） | [#2586](https://github.com/hat47x/tei-platform/issues/2586) |
| 2 | **動く（CLI）：** 実PostgreSQL→`tei view -environment`→静的HTML→実Chrome DOMで、主キーごとのRule三値を独立Expectedと照合する（[#5425](https://github.com/hat47x/tei-platform/pull/5425)）。**動く（第一Web TargetのHTTP）：** `-serve-read-from-db`と`TEI_WEB_READ_PG_DSN`で選択した実Providerの明細と判定を同じReaderから返す（[#5431](https://github.com/hat47x/tei-platform/pull/5431)・[#5439](https://github.com/hat47x/tei-platform/pull/5439)）。HTTP画面の明細は、行番号ではなく業務主キー・値・Rule回答で照合する。**試験だけ：** 合成Queryの画面試験。**動く：** 規則変更の前後を、実PostgreSQLの同じHTTP経路で独立した期待と照合する（[#5482](https://github.com/hat47x/tei-platform/pull/5482)）。**動く（第一Webの実Chrome DOM）：** 同じHTTP経路で規則変更前後をそれぞれ画面で確かめ、主キーと独立Expectedで照合する（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。**次の受入：** 手順書の操作を実際に一続きで行い、独立Expectedと照合する。その後、通常開発と比較 | [#1119](https://github.com/hat47x/tei-platform/issues/1119) |
| 3 | **動く：** 参照Workspaceが`covers sales.order.lines`を宣言し、実行Queryと関係の保存を照合できた場合だけ`coverage: complete`とする（[#5507](https://github.com/hat47x/tei-platform/pull/5507)、[#5512](https://github.com/hat47x/tei-platform/pull/5512)）。受注1002の一意は`satisfied`、重複する1001は`not-satisfied`、行を落とすQueryは`partial`に留める。宣言やSQLの静的一致だけを実DBの全件取得の証拠にしない。**同じReadのトランザクションで実テーブルのRLS・継承・種類を確認し、不可視行の可能性を排除できない場合は`partial`と理由へ戻す**（[#5525](https://github.com/hat47x/tei-platform/pull/5525)）。**第一Webも、同じ実ReadのトランザクションでQueryと物理表の資格を確認できたときだけ全件として評価する**（[#5571](https://github.com/hat47x/tei-platform/pull/5571)）。実行時の可視範囲とOwnerによる業務意味の受理は区別する。[判断の文書](design/RULE_NORMAL_FORM_AND_OPERATOR_SIGNATURE_DECISION_2026-11.md)§6 | #2586 |
| 4 | **動く：** `tei desired-schema -format sql`によるDDL生成（[#5385](https://github.com/hat47x/tei-platform/pull/5385)）、`storage decimal(12,3)`の精度指定とDDL反映（[#5411](https://github.com/hat47x/tei-platform/pull/5411)）、生成DDLの実PostgreSQL適用から同じCLIによる読取・独立期待との照合（[#5422](https://github.com/hat47x/tei-platform/pull/5422)）、`tei view -environment -format text`の行別表示（[#5372](https://github.com/hat47x/tei-platform/pull/5372)）。**動く：** 精度を指定した列が実DBでその桁数で作られ、その列から読んだ行のRuleの答えが独立した期待と一致する（`TestDeclaredDecimalPrecisionSurvivesToTheDatabaseAndBack`） | #2586 |
| 5 | **未実装（複数入力の意味・評価）：** Ruleの名前付き入力をOperator署名に沿って既存の意味・評価経路へ渡す。最初は`within(value, low, high)`を使い、主語による第1入力の束縛、残りの入力名による対応、値がある場合のOperator例、NULL・未取得を補わないRule判定を独立Expectedで検証する。次の製品変更は既存の`internal/syntax/rule_authoring.go`、`internal/syntax/rule_authoring_validation.go`、`internal/syntax/operator_cases_authoring.go`、`internal/semantic/model.go`、`internal/semantic/operator_case.go`、`internal/semantic/validationeval/operator.go`を主な対象とし、別文書の署名解決には既存`internal/workspace/rule_authoring_markdown_loader.go`の全Source読込後の経路を用いる。文書の読込順に結果を左右されず、未解決の入力値を意味モデルへ渡さない。解決に失敗した文書を部分的に受理せず、Source移動後の旧Source Indexも再利用しない。受注の五段階受入後に着手し、新DSLや第二の評価器は作らない | [#4359](https://github.com/hat47x/tei-platform/issues/4359) |

道の1は[#5327](https://github.com/hat47x/tei-platform/pull/5327)（固定名の決め打ちを外す）、[#5339](https://github.com/hat47x/tei-platform/pull/5339)（Data・Relationの意味IDと参照テーブルを変える）、[#5362](https://github.com/hat47x/tei-platform/pull/5362)（別の属性・Query入力・複合主キーを持つ独立モデル）で通った。#5362は`checkDestinationAlternateModel`を`TestDestinationOrderChangeReadViaCLIFromPostgres`の中から呼び、CIが実PostgreSQLで走らせる。

道の2の実行経路は、実PostgreSQLの明細をCLIと第一Webで読み、行別Ruleの答えを実Chromeで確認するところまで動く（[#5492](https://github.com/hat47x/tei-platform/pull/5492)）。[#5588](https://github.com/hat47x/tei-platform/pull/5588)では、公開手順のDDL／Seed投入・Workspace複製・Ruleと例の編集を、手順書のbashから読み取り、同じ隔離DB・Workspace・実CLI・実Chromeの一試行で検査した。[#5602](https://github.com/hat47x/tei-platform/pull/5602)では第一Webを製品パッケージからビルドした別OSプロセスとして起動し、実DBから同じ画面へ回答を返した。試験の前後処理はDB名を確認して対象の親子2表に限定し、[#5605](https://github.com/hat47x/tei-platform/pull/5605)で依存先を巻き込む削除をやめた。以上は**動く**。さらに[#5608](https://github.com/hat47x/tei-platform/pull/5608)では、手順2・4のCLIコマンドも公開原文のまま同じ実PostgreSQL・実Chrome試行で実行し、独立Expectedと照合した。さらに[#5633](https://github.com/hat47x/tei-platform/pull/5633)では、手順5に掲載した実`go run`を起動し、同じ編集済みWorkspace・実PostgreSQLから受注1001／1002の画面回答を実Chromeで照合した（CIは待受ポートのみ隔離用に変更）。ここまで**動く（試験で確認）**。この段階のChrome送信は試験用Bootstrap POSTだった。出荷Webの実フォーム操作を確かめる次段と、初回利用者自身による完走を分けて検収する。

**道の2の自動フォーム検証**：既存`tools/reference-first-web-target/first_change_guide_e2e_test.go`の実`go run`／実PostgreSQL／Chrome試行に、出荷Web自身の`GET /custom-submit`を開いて四つの可視入力欄（Login ID・Credential・Browse from・Selected row）を埋め、Submitを押す検査を追加した（[#5642](https://github.com/hat47x/tei-platform/pull/5642)）。受注1001／1002の業務主キー別Rule三値を独立Expectedで照合し、出荷Webバイナリと公開`go run`の両方を使う。試験用Bootstrap POSTによる送信はこの資格に数えない。さらに[#5687](https://github.com/hat47x/tei-platform/pull/5687)では、受注1001がValidation不成立でCommandに進まず、受注1002だけが合成Command／Observationまで進むことを実Chromeで確かめた。[#5715](https://github.com/hat47x/tei-platform/pull/5715)・[#5719](https://github.com/hat47x/tei-platform/pull/5719)では、同じ隔離PostgreSQLの受注3件（顧客・日時・状態・金額を含む）と明細5行の全値が、試行終了時点でSeedと一致することを独立SQLで確認した。ただし途中の変更と復元や、DB全域の副作用不在は保証しない。**実DBの明細読取りは動くが、実DBへの受注確定・Commit・Receiptはこの構成では未実装である。** HTTP 200や画面の`completed`は保存成功の証拠にせず、第一Webの参照フォームにも合成状態の観測が実DB更新の証拠ではないことを表示し、実Chromeで確認する。さらに[#5708](https://github.com/hat47x/tei-platform/pull/5708)では、Validation不成立時にAuthorization／副作用入口へ到達しないことを既存RunnerのSpyと実Chromeで検査した。[#5721](https://github.com/hat47x/tei-platform/pull/5721)では公開手順3の既存Source・リンク経路を編集前に拒否し、[#5722](https://github.com/hat47x/tei-platform/pull/5722)ではRelation全体のRule回答がない正常なReadにもQuery・業務主キー・行数の帰属を画面表示する。ただし前者は3ファイルの原子的保存や外部Writerとの競合拒否、後者はDB Snapshotの同一性を保証しない。**次の受入は、合成試験を増やすことではなく、初回利用者自身が公開手順0〜5を補助なしで完走できるか**の確認である。不成立の場合は、その箇所と独立Expectedに限ってコード・手順を修正する。通った後に通常開発と変更ファイル数・手数を比べる。**Readの資格は流用しない。** 行位置と業務主キー、QueryのScopeと実行Attempt、業務対象集合とDBロールの可視集合を区別する。Relation全件の成立はCLIとWebそれぞれの**同じ読取トランザクション**で取得範囲を確認できた場合だけ認める（[#5571](https://github.com/hat47x/tei-platform/pull/5571)）。非所有者RLSで重複行が隠れる実DB負例でも一意性は`undecided`に留める（[#5583](https://github.com/hat47x/tei-platform/pull/5583)）。観測済み重複は`not-satisfied`、資格のある受注1002は`satisfied`。複数HTTP要求の共通Snapshot、Owner意味受理、Host認可は別の受入条件とする。

**第一Webの到達点と生成資格は区別する。** 現在の出荷相当のフォームは`tools/reference-first-web-target/custom_ui.go`の手書きReference UIであり、設計文書から得たRuleの回答を実PostgreSQL・実Chromeへ表示できることは確認済みだが、Web画面自体がTEIのTarget Generatorから生成された証拠ではない。まず初回利用者の公開手順0〜5の独力完走を検収し、その結果をもって画面生成や正式Source受理を合格とはしない。画面生成の資格は同じSource版からArtifactと実Consumerまでの必要な閉包で別に検収する。

入口は増やさない。接続の秘密は環境変数から受け取り、作業領域へ書かない。

並行して進めてよいのは一本だけである。**Studioが次の自分を作る一巡**（[#4583](https://github.com/hat47x/tei-platform/issues/4583)）。完了の条件は一文にする。Studioで定義を一つ変え、**生成器には触れずに**、Studio自身が作った次のStudioがWindowsで起動し、その変更が画面に出ること。

「生成器には触れずに」が要である。[#5516](https://github.com/hat47x/tei-platform/pull/5516)・[#5522](https://github.com/hat47x/tei-platform/pull/5522)でData定義から実Host応答の必須文字列項目だけをReader契約へ導き、[#5534](https://github.com/hat47x/tei-platform/pull/5534)でData必須指定の変更→実Go Generator→Reader→esbuildの受理・拒否差を局所検査した。[#5542](https://github.com/hat47x/tei-platform/pull/5542)・[#5555](https://github.com/hat47x/tei-platform/pull/5555)では初回InspectとGraph選択後のReader契約・呼出増分を独立Expectedで守る。[#5556](https://github.com/hat47x/tei-platform/pull/5556)・[#5560](https://github.com/hat47x/tei-platform/pull/5560)はStage1・Stage2の実TEI自身から既存Node/esbuildを起動し、生成ReaderからN2 Bundleを作る経路を検収した。さらに[#5572](https://github.com/hat47x/tei-platform/pull/5572)では**変更後の`tei.authoring/v1` Data→各段の実Generator→N2新Bundle→隔離したWails N3埋込み→Stage2 Windows exe→別プロセスの実WebView2で生成Readerを使用する同一試行**が、Windows実機を含む5種類のCI全成功後に統合された。Source・Part・Bundle・埋込・exe・実InvokerとReader契約・呼出増加を照合する。ただし、画面本文が変更前後で目に見えて異なることは、この試行では検収していない。

**Studioで残る一巡は、編集画面で作った変更を正本に保存し、その正本から次のStudioを作ること**（[#4583](https://github.com/hat47x/tei-platform/issues/4583)）。[#5587](https://github.com/hat47x/tei-platform/pull/5587)は実`InspectSubject`のCanonical、[#5597](https://github.com/hat47x/tei-platform/pull/5597)はStage2のWindows実行ファイルと実WebView2の可視Inspector本文について、`snapshotFingerprint.required: false→true`を独立した期待と照合した。**動く（試験で確認）：** 変更前後の二つのWorkspaceを先に用意すれば、生成器を変更せずに画面の差まで届く。ただし、Studio EditorがそのWorkspaceを編集した証拠ではない。

[#5606](https://github.com/hat47x/tei-platform/pull/5606)・[#5604](https://github.com/hat47x/tei-platform/pull/5604)・[#5613](https://github.com/hat47x/tei-platform/pull/5613)・[#5620](https://github.com/hat47x/tei-platform/pull/5620)では、Referenceのオフライン編集フォームでDataの必須指定を変更する候補を作り、既存Shared AcceptanceのReview・明示Acceptを経てSourceの一セルを保存・再読込した。[#5616](https://github.com/hat47x/tei-platform/pull/5616)では模擬DOM操作をStage1／Stage2生成・Windows画面へ接続し、[#5630](https://github.com/hat47x/tei-platform/pull/5630)では**実Microsoft Edgeの候補操作→Referenceの明示Accept→同じSourceを使うStage1／Stage2→実Windows WebView2の可視差**まで同一試行で確認した。ただし、これらは製品Wailsの内蔵Editorによる保存ではない。

**製品Workbenchの現在地――未審査候補の根拠を実画面で確認**：[#5647](https://github.com/hat47x/tei-platform/pull/5647)でHostのInspect／Source Indexに基づく型付き候補を作り、[#5651](https://github.com/hat47x/tei-platform/pull/5651)で選択・Workspace・Inspect世代変更時の失効を実装した。[#5669](https://github.com/hat47x/tei-platform/pull/5669)では実Windows Wails／WebView2でData属性選択→Host Inspect→候補ボタンのクリック→型付きJSON照合→Workspace切替後の旧候補破棄を確認した。これは通常の固定Fixtureだけを使うDesktop smokeとは別の実操作検証である。さらに[#5734](https://github.com/hat47x/tei-platform/pull/5734)を対象HEADの10ワークフロー全成功後に統合し、型付き候補JSONを変えず、作成時のWorkspace・Sourceの観測相対パス／行・指紋と、Source Indexの網羅性・原文照合・書込権限が未確認であることをWindowsの実画面に表示した。同一のSource原文でも配置先・Model Revision・Source Indexの版が変わった場合、旧Inspectと新Catalogを組み合わせた未審査候補を拒否する合成回帰も、[#5730](https://github.com/hat47x/tei-platform/pull/5730)で追加した。これは実Sourceファイル移動後の保存やWriter資格を検証したものではない。**Source情報は候補作成時の観測であり、現在の書込先や許可の証明ではない。** Editorはread-onlyのままで、Sourceを書き換えず、Save／Accept UIも持たない。製品Writer・正式Review／Accept・Owner意味受理は引き続き**未実装**。

**製品Wails保存接続のFit/Gap判定（2026-11-20、NO-GO）**：`cmd/tei-studio/main.go`の`StudioBridge`には、Source用のPreview／Review／Accept、正規Writer、結果照会の入口がない。`internal/studiohost/readonly.go`のWorkspace許可リストとRoot同一性検査も書込みGrantではない。`reference/tei-studio-workbench/source-locations.ts`の未審査候補は、Source Index網羅性・原byte確認・書込資格をいずれも`false`としている。Referenceの`studio-authoring/bridge.mjs`には編集候補の変換、`tools/reference-shared-acceptance`にはPreview／Review／指紋付きAcceptとData表の一セル保存があるが、後者はCLI用`package main`であり、製品Hostの正規Operationではない。したがって、**製品StudioのSource保存は引き続き`unsupported`とする**。

**Sourceの現在性と競合保証は別に判定する。** [#5691](https://github.com/hat47x/tei-platform/pull/5691)では、Operator宣言を原byte不変のまま別ファイルへ移すと、意味は同じでもSource位置とModel Revisionが変わり、旧Indexが失効することを確認した。さらに[#5730](https://github.com/hat47x/tei-platform/pull/5730)では、Workbenchの合成Inspect／Catalogで同じ原byteのSource移動を表し、旧・新の版を混ぜた未審査候補を拒否する回帰を確認した。この#5730は合成回帰に限る。その後[#5747](https://github.com/hat47x/tei-platform/pull/5747)で、実ファイルを原byte不変のまま移動し、同じ製品Go HostからCatalog／Inspectを再読込した。CanonicalとWorld指紋、Source原文の指紋は維持される一方、Source出現位置・Source Index指紋・Model Revision・Snapshot指紋は変化した。**この実Host再読込みも正式Writerの保存資格や非協調Writerとの競合保証ではない。** 加えて`source_accept_lock.go`の`.tei-accept.lock`は同じ入口を使うAccept間の協調ロックに限られる。`source_write_root.go`の既存ファイル更新は、一時ファイルを書いて同じRoot内で`Rename`する方式であり、最終照合と置換の間に協調しないWriterが書き換えても必ず拒否できる原子的CASではない。途中でA→B→Aと変わって最終byteが元に戻れば、現在値の照合だけから履歴を検出することもできない。**検出できた版・位置の不一致を拒否することと、外部編集を含むすべての競合を拒否できることは同義ではない。**

**次の製品コード変更は一セルだけ**：`studio.inspection_response.snapshot_fingerprint.required`を対象に、既存Operationsの正式な接続資格を先に確かめる。Accept直前に現在のWorkspace、意味ID、一意のSource出現位置と原byte、Source Index・Model Revision、Writer Grant、Review指紋を再解決し、旧Workspace・同名別Source・移動済みSource・検出可能な競合・Grant不足を拒否する。**協調するAccept同士の競合**と**協調しない外部Writerの競合**を別の負例で検収し、既存Writerで後者の拒否を保証できない条件は`unsupported`で止める。履歴を観測できないA→B→Aの完全検出は約束しない。失敗時は[#5703](https://github.com/hat47x/tei-platform/pull/5703)で補完した`retryable`／`effectUnknown`の分類だけで再送せず、同一Operation identityによる**実際の結果照会**が成立するまで再Acceptを止める。新Host／第二Writer／RegistryやReference CLI子プロセスを増やさず、保存後に再読込した同一Source版からだけStage1→Stage2を再生成し、Windowsの可視差を照合する。Ownerの業務意味受理、正式Build Intentと依存閉包は別に検収する。

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