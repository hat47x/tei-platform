# reference-order-multilingual-operation

技術検証用サンプル[`order-multilingual-operation`](../../reference/order-multilingual-operation/README.md)(サンプル8)の工程1〜6を、一つの通しの記録として実行する実装です。設計の正本は[技術検証用サンプルの統合設計 4.8節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#48-受注業務の英語化運用と原文改訂時の訳の失効order-multilingual-operation)です。

この実装はサンプル固有の処理で、`internal/`の基盤には入れていません。既存の多言語資産(`reference/standard-messages`、`message-localization`、`multilingual-overlays`、`multilingual-resources`、`multilingual-business-data`、`multilingual-format`)は変更せずに呼びます。製品コードは`internal/workspace`、`internal/plugin`、`internal/sourceacceptance`、`internal/modelref/derivedbridge`、`internal/index`をそのまま使います。`reference/workspace`は一時ディレクトリへ複製して使い、元は読むだけです。

## 構成

| 役 | 実体 | 担当 |
| --- | --- | --- |
| 作業領域 | `reference/workspace`の複製(`workspace.go`) | 工程1で受注の所有プラグイン`sales.order_entry@1.0.0`と`std.message`を、工程2で言語プラグイン`std.lang_en`、`std.lang_de`、`partner.lang_en`を加える。受注の業務定義(`semantic/sales/order.tei`)は変更しない |
| 原文宣言 | [`testdata/source-r1.json`](testdata/source-r1.json)、[`testdata/source-r2.json`](testdata/source-r2.json)(`sources.go`) | 日本語の確認メッセージと、派生索引の型つきの識別子に結ぶ表示名・機能名。r1からr2への一回の変更が原文改訂 |
| 言語プラグインの訳 | [`testdata/translations.json`](testdata/translations.json) | 訳ごとに、確認した原文の版(r1、r2)を持つ。`BasedOn`は資産ごとの規則で計算する |
| ホスト | 試験のプロセス内の`Host`(`host.go`) | 現在の受理(作業領域の索引、原文宣言のダイジェスト、固定情報の照合値)を持ち、`sourceacceptance.VerifyCurrent`で照合する。メッセージの選択と配布候補の作成の前に同じ確認を行う。合成のホストで、製品の受理機構ではない |
| 業務の表示 | `Scene.Show`(`scene.go`) | 英語の経路が拒否されたら、拒否の理由を記録して日本語の原文へ明示的に切り替える。保存された訳へは戻らない |
| 実DB | DockerのPostgreSQL一つと、Pythonの`sqlite3`のファイル(`store.go`) | 工程5。呼出しのたびに新しい接続(SQLiteは別プロセス)を開く |
| 翻訳器 | 試験のプロセス内の`MockTranslator`(`stage5.go`) | 外部へ何も送らない。呼出し回数、提供元での処理回数、渡された文脈を記録する |
| JavaScript | Nodeで`render.mjs`を実行 | 工程6。`reference/multilingual-format/probe.mjs`をそのままimportする |

## 既存資産を呼べるか

| 工程 | 既存資産 | 呼べるか | 理由と扱い |
| --- | --- | --- | --- |
| 1 | `reference/standard-messages` | ライブラリとして呼べる | `Catalog.Validate`、`Lookup`、`Revision`、`ValidateArguments`。版は定義全体のJSONのSHA-256 |
| 2 | `reference/message-localization` | ライブラリとして呼べる | `ResolveWorkspace`が複製を実際に読み込む。`std.message`が環境で選ばれている必要があり、複製に加えた。項目名は資産内で`message-body`に固定 |
| 3・4 | `reference/multilingual-overlays` | ライブラリとして呼べる | `BuildLockedLocaleBundle`、`CheckLocaleBundleIntegrity`、`CheckLocaleBundleAgainstLock`、`BuildLockedDisplay`。版は表示文字列のSHA-256 |
| 3 | `reference/multilingual-overlays`の`BuildAcceptedWorkspaceDisplay` | 呼べるが失敗する | 複製では`workspace.IndexedLoadedSnapshot`が`databases/sales-core/README.md`(解析されない管理ファイル)で失敗する。仮に通っても原文は`Symbol`(ASCIIの識別子`Order`)で、日本語の表示名を渡せない。一回呼んで失敗を記録する |
| 3・4 | `reference/multilingual-resources` | 上の二つの内部で呼ばれる | 固定データ`testdata/order-entry.json`は所有者`order-entry`、項目名`action-label`、版`source-12`で、他の資産と合わない。識別子の対応の確認に読む |
| 4 | `reference/multilingual-integration` | 呼べない | 試験ファイルだけのパッケージで、公開の入口がない。通しの根拠にしない |
| 5 | `reference/multilingual-business-data` | ライブラリとして呼べる | `Select`、`Validate`、`PrepareTranslation`。`testdata/schema.sql`をファイルとして読み、両方のDBへそのまま適用する |
| 6 | `reference/multilingual-format` | 呼べる | Goの`Probe`と、`probe.mjs`の`probe`。メッセージとの接続はサンプル側 |

## 識別子の一本化

通しの記録は、次の一組だけを使います(`identity.go`の`Unified`)。資産ごとの合成値は、呼出し、ファイル、試験のソースから読み取って`chain-record.json`の`identities`に並べます。

| 識別子 | 通しの記録 | 資産ごとの合成値 |
| --- | --- | --- |
| 所有プラグイン | `sales.order_entry@1.0.0`(複製に追加) | `std.order_entry@1.2.0`(overlaysの`ReferenceOrderCase`)、`sales.order_ui@1.0.0`(message-localizationの試験)、`tei.order.ui@1.0.0`(overlaysの作業領域の試験)、`order-entry`(resources) |
| 環境 | `development`(`reference/workspace`) | `env.order_reference`(overlays) |
| アプリケーション | `order_entry`(`reference/workspace`) | 業務データの試験のテナント`store-a` |
| 原文言語 | `ja-JP` | `en-US`(message-localizationの試験、overlaysの作業領域の試験) |
| 項目名 | メッセージは`message-body`、表示名と機能名は`display-name` | `action-label`(resources) |
| 対象 | 範囲`tei_reference`の`semantic-data`(`sales.order`、`sales.order_line`)と`semantic-capability`(`sales.order.submit`)。派生索引から導く | 範囲`sales`の種類`function`(overlays、resources) |

対応できないものは次のとおりです。

- 原文版の規則は一本化できません。メッセージは定義全体のSHA-256、表示名は表示文字列のSHA-256、業務データは文章版です。通しの記録は、原文宣言のダイジェスト(改訂一回)と、対象ごとの資産固有の版の改訂前後を突合表に並べます。
- 業務データの属性`catalog.product.description`と、受注明細の`itemCode`から`product:<itemCode>`への対応は、作業領域にないサンプルの値です。

異なる識別子を混ぜると拒否されることを、完了条件`identity-unified-and-mixing-refused`で確かめます(他の資産の所有者、環境、項目名、対象の形、所有者の版、原文言語への訳、他の環境の許可の8件)。

## 工程の流れ

1. 工程1(`Stage1`): 作業領域を複製し、所有プラグインと`std.message`を加えます。原文宣言r1のメッセージを検証し(重複、未定義、整数引数への文字列を拒否)、言語プラグインなしで原文のまま解決します。英語は`missing-translation`で拒否されます。
2. 工程2(`Stage2`): 言語プラグイン三つを環境に加え、固定情報を作り直します。通常のプラグインの固定の項目は変わらず、増えたのは言語プラグインの三件だけであることを確かめます。r1の訳を確認し、ストア、束縛、許可、ホストの確認口を経て、各メッセージをja-JP、en-US、de-DEで選びます。
3. 工程3(`Stage3`): 同じ環境と固定情報で、表示名(`sales.order`、`sales.order_line`)と機能名(`sales.order.submit`)のen-USとja-JPの配布候補を作り、ダイジェストを固定します。
4. 工程4(`Stage4`): 原文宣言をr2へ一回改訂します(確認メッセージ、`sales.order`の表示名、`sales.order.submit`の機能名)。[`testdata/expected-matrix.json`](testdata/expected-matrix.json)の対象一覧の各行を、改訂前と改訂後に、その行の訳だけを渡して資産で評価します。旧訳は履歴に残し、改訂した対象の英訳だけを再確認して配布候補を作り直します。
5. 工程5(`Stage5`、条件付き): 商品説明をPostgreSQLとSQLiteへ同じ手順で保存し、原文改訂、別接続の再読込、翻訳案の準備、並びの比較を行います。
6. 工程6(`Stage6`、条件付き): 改訂後の確認メッセージの整数引数`count`を、複数形と桁区切りに接続し、GoとJavaScriptで比べます。

工程ごとの記録は前の記録の照合値でつなぎ、`VerifyThrough`が読むだけで照合します(工程の欠落、識別子の差替え、工程2〜4の固定情報の違い、工程3と4の配布候補のつながり)。

## 失敗注入と合否

観測値は、下の「安定性の記録」の実行で得た値です。版の照合値は先頭だけを示します。

| 要件 | 記録のID | 注入 | 合否の観測 |
| --- | --- | --- | --- |
| P-MIX-03(工程4) | `pmix03-old-bundle-after-revision` | 改訂直後に、改訂前の配布候補(`296848a4…`)を表示しようとする | 整合性とLockの照合はどちらも通る。サンプルの原文照合が`sales.order`と`sales.order.submit`のen-USとja-JPの4件を旧原文として拒否する。再確認前の再生成は`stale-translation`で発行されない。画面は確認メッセージと表示名・機能名を日本語の原文で出し、旧英訳の表示は0件。別の提供元の`order.submit_rejected`の英訳は表示される |
| P-DB-02(工程2・3) | `pdb02-grant-revoked` | 英語の許可を外す | メッセージは`translation-not-granted`、配布候補は`overlay-not-permitted`で拒否し、日本語の原文で表示。固定した配布候補は整合性とLockの照合を通るが、再生成が拒否されて表示されない |
| P-DB-02 | `pdb02-acceptance-revoked` | ホストの受理を取り消す | `SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE`で英語も原文のみの経路も拒否され、受理済みの原文宣言の日本語を直接表示する(`source-direct`) |
| P-DB-02 | `pdb02-lock-expired` | 英語プラグインの定義を変え、固定情報は古いまま | `locked plugin std.lang_en@1.0.0 fingerprint diverged`で全経路が拒否され、日本語の原文を直接表示する |
| P-DB-02 | `pdb02-english-plugin-removed` | `std.lang_en`を環境から外し、固定情報を作り直す | `unselected-translation-provider`、`unselected-locale-plugin`で拒否し、日本語の原文で表示。`std.lang_en`の訳の表示は0件。`partner.lang_en`の訳は残る。固定した配布候補は`environment-lock-mismatch` |
| P-DB-02 | `pdb02-other-environment` | 同じ訳を`production`で選ぶ | `std-message-unselected`で拒否 |
| 固有(工程3・4) | `bundle-text-rewritten` | 配布候補の訳文だけを書き換える | `digest-mismatch` |
| 固有(工程3・4) | `bundle-text-rewritten-digest-recomputed` | 訳文を書き換え、ダイジェストも計算し直す | 整合性とLockの照合は通る。固定したダイジェストと現在の再生成(`296848a4…`)との比較で拒否 |
| 固有(工程3・4) | `environment-lock-swapped` | 英語プラグインの定義を変えて固定情報を作り直す | 固定情報の照合値が`2df56be1…`から`b8e5ffb8…`へ変わり、固定した配布候補は`environment-lock-mismatch`。再生成した候補のダイジェストは改訂前と異なる(`a8eaaccb…`)。元に戻すと固定情報は元の照合値に戻る |
| 固有(工程5) | `concurrent-edit-postgresql`、`concurrent-edit-sqlite` | 翻訳器の呼出し中に、別の接続から訳先(P-101の英訳を公開)または原文(P-102の日本語を改訂)を更新する | `target-changed`、`source-changed`で拒否し、何も保存しない。競合のない翻訳案は`review_state=candidate`で保存し、公開しない(候補1件、公開はen-1のまま) |
| P-TX-02(工程5) | `ptx02-postgresql`、`ptx02-sqlite` | 翻訳器の失敗、取消し、結果不明(提供元で処理した後に切断)。機密区分の文章の外部送信。許可のない外部送信 | 失敗は`translation-failed`、取消しは`canceled`、結果不明は資産では`translation-failed`だが、サンプルが誤りの型で`outcome-unknown`と分類する。いずれも呼出し1回で再試行せず、何も保存・公開しない。外部送信の二件は`external-transfer-blocked`で、翻訳器の呼出しは0回 |
| 固有(工程5) | `business-data-revision-history-postgresql`、`-sqlite` | 商品P-100の日本語を改訂する | 旧英訳の行は履歴に残り(1行)、現在値ビューから消える。資産のGoの`Select`とSQLの現在値ビューが行単位で一致(19行、翻訳案と同時更新の後は20行) |
| P-DB-04(工程5) | `pdb04-declared-binary-order` | 日英の商品説明20行(濁点の合成済みと結合文字、半角カナ、全角記号、大文字小文字)を並べた一覧を両製品で取得する | 宣言した並びの鍵(UTF-8の符号位置順)で、件数20、ハッシュ`1531cb21…`が一致。両製品から3行ずつ交互に続き頁を取ると、抜けと重複は0。PostgreSQLだけICUの並び(`und-x-icu`)にすると、ハッシュは`92392500…`になり、交互の続き頁で重複1行、抜け8行 |

P-DB-04で確かめられることと、確かめられないことは次のとおりです。

- 確かめられること: このPostgreSQL 16.15(UTF8、`COLLATE "C"`)とSQLite 3.45.1(`BINARY`)で、宣言した二進の鍵による並びが同じになり、鍵による続き頁を製品をまたいで取っても行が抜けず重複しないこと。言語依存の並びを片方に混ぜると抜けと重複が起きること。
- 確かめられないこと: 言語上の同一視です。二進の鍵は正規化しないため、`が`(合成済み)と`か゛`(結合文字)、`ｶﾞ`と`ガ`は別の値として並びます。正規化した鍵の宣言、照合順序の版の変化の検知、索引の利用、他の製品は扱っていません。PostgreSQLのデータベースの既定(alpineの`en_US.utf8`)の並びは今回は二進と同じでしたが、glibcの環境では一致しないため、根拠にしません。

## 通しの完了条件

完了条件は、`chain-record.json`の`completionConditions`です。どれも別々の試験の足し算ではなく、同じ通しの記録の上で判定します。

- `one-revision-expires-dependent-translations-only`: 改訂前後の同じ対象一覧(14行)の突合表で、すべての行が、宣言した期待値と、資産固有の版の変化から導いた規則の両方に一致します。失効したのは、確認メッセージのen-USとde-DE、表示名`sales.order`のen-US、機能名`sales.order.submit`のen-USの4行です。残ったのは、改訂していない表示名`sales.order_line`の同じ提供元のen-USと他言語のde-DE、他の提供元(`partner.lang_en`)の`order.submit_rejected`のen-US、他言語のde-DE、英語で独立に書いた`order.tax_notice`、各原文です。固定情報は工程2と同じ照合値で、改訂前の英訳3件は履歴に残ります。
- `regenerated-bundle-differs-and-old-pin-refused`: 再確認して作り直した配布候補のダイジェスト(`d9cb6492…`)は改訂前(`296848a4…`)と異なり、改訂前のダイジェストに固定した表示は旧原文の4件で拒否されます。新しい候補は受け入れられ、確認メッセージ、表示名、機能名がen-USで改訂後の原文版に揃います。de-DEの確認メッセージは再確認していないため失効のままです。
- `japanese-continues-without-silent-old-translation`: P-DB-02の5件とP-MIX-03の1件が合格です。
- `identity-unified-and-mixing-refused`: 上の識別子の一本化です。

条件付きの工程5と6は`conditionalConditions`に記録し、完了に数えません。工程5の前提(受注明細の商品コードから商品説明への接続)は、作業領域に商品説明の属性がないため成立していません(`stage5-precondition`の`established=false`)。工程6は、`order.batch_confirmed`の英訳(単数・複数の別形)を件数0、1、2、999、1000、12345、1234567、2147483647のen-USとja-JPで、GoとJavaScriptが16件すべて一致して表示します(例: `Confirmed 1 order and issued shipping instructions.`、`Confirmed 12,345 orders and issued shipping instructions.`、`12,345件の受注を確定し、出荷を指示しました`)。

## 実行

作業ブランチのコミットを浅いcloneした清浄な複製の中で実行します。

```bash
git clone --depth 1 --branch <作業ブランチ> file:///<リポジトリ> /tmp/tei-omo
cd /tmp/tei-omo
python3 tools/verify-order-multilingual-operation.py --out /tmp/tei-omo-receipt
```

- スクリプトは`tei-omo-<乱数>-pg`という名前のPostgreSQL(既定は`postgres:16-alpine`)を一つ起動し、終了時にそのコンテナーだけを削除します。`TEI_OMO_DSN`、`TEI_OMO_CONFIRM_ISOLATED`、`TEI_OMO_OUT_DIR`、`TEI_OMO_PYTHON`、`TEI_OMO_NODE`を与えて`go test -json -count=1`を実行し、全試験がPASSでskipが0件、通しの記録が合格、試験の後も作業ツリーが清浄なときだけ合格です。
- 環境変数がないと、通しの試験はskipします。工程1〜4の試験と単体試験はDBなしで動きます。
- 一回の実行は約20秒です(コンテナーの起動を除く)。

## 時間に依存する判定

時刻で合否を決める判定はありません。翻訳器の取消しは`context`の取消しで、結果不明は誤りの型で起こします。同時編集は翻訳器の呼出しの中で別の接続から行い、時間待ちを使いません。コンテナーの起動待ちは回数で数えます。

## 安定性の記録

(反復の結果をこの節に記録します。)

## 基盤の不足

`chain-record.json`の`platformGaps`と同じ内容です。

| 要件 | 領域 | 不足 | サンプルでの扱い |
| --- | --- | --- | --- |
| 固有 | データ | `reference/workspace`に、受注の業務定義を所有するプラグイン、`std.message`、言語プラグイン、日本語の表示名がない。構文の識別子はASCIIで、`Symbol`は識別子。dataとattributeの`display`文字列は構文にあるが使われておらず、capabilityにはない | 複製に所有プラグイン、`std.message`、言語プラグインを加え、日本語の表示名とメッセージは原文宣言に置いて派生索引の型つきの識別子へ結ぶ |
| 固有 | コア | 原文版の導出規則が資産ごとに異なる | 資産固有の版をそのまま使い、原文宣言のダイジェストと並べる |
| 固有 | プラグイン | 翻訳の許可と受理の確認口が、メッセージ(`ReviewGate`)と表示名(許可のみで、配布候補の作成に確認口がない)で別 | サンプルのホストが同じ確認を両方の前に行う。製品の確認口ではない |
| 固有 | コア | 複製した`reference/workspace`で`workspace.IndexedLoadedSnapshot`が`databases/sales-core/README.md`のために失敗し、overlaysの作業領域の経路が使えない | ホストの照合には`workspace.BuildSourceIndex`を使う |
| P-MIX-03 | 実行時 | 配布候補の整合性とLockの照合だけでは、原文改訂と許可の取消しを検出できない。ダイジェストは訳文と同時に計算し直せる | サンプル側で、各表示の原文版を現在の原文と照合し、現在の条件で再生成した候補のダイジェストと比べる。署名と認証された配布経路はない |
| 固有 | 実行時 | メッセージの引数展開と件数書式の接続がなく、`named-v0`に複数形の選択がない | 訳に複数形の別形をサンプルの形で持たせ、`ValidateText`で検査し、`Probe`の分類で選ぶ |
| 固有 | データ | 業務データの多言語属性が意味モデルにない。受注明細の`itemCode`から商品説明への関係がない | 属性と対応はサンプルの値。工程5は完了に数えない |
| P-TX-02 | ツール | 翻訳依頼と応答の追跡、受理トランザクション内での再照合がない。結果不明は資産では失敗と区別されない | 模擬の翻訳器の誤りの型でサンプルが分類し、自動で再試行しない |
| P-DB-04 | データ | 並びと比較の意味を宣言する手段がない | サンプルが並びの鍵を宣言し、PostgreSQLは`COLLATE "C"`、SQLiteは`BINARY`へ写す |

## 範囲外

- 製品の受理機構、権限、Shared Acceptanceへの接続。ホストと許可は合成です。
- 実際の翻訳サービスや生成AI。翻訳器は模擬で、外部へ何も送りません。翻訳記憶、用語集、人間の確認は扱っていません。
- 配布候補の署名、認証された配布経路、Web・Desktopの画面と表示のキャッシュの失効。
- 言語上の照合(正規化、かなの同一視)、照合順序の版の変化、別の機械に置いたDB。
- `reference/multilingual-integration`の横断の試作は、公開の入口がないため呼んでいません。
