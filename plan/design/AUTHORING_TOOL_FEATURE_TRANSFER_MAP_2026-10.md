# 設計ツールの機能移植対応表（X-TEA ModelerとEditor）

- 状態: 調査記録と設計候補。ここに挙げる機能は、実装済みでも採用済みでもない。
- 調査日: 2026-10-14
- 対象: X-TEA ModelerとX-TEA Editorの公開ソースと同梱の画面資源
- 親Issue: [#1119](https://github.com/hat47x/tei-platform/issues/1119)（三視点の実証）。本表の追跡は[#1802](https://github.com/hat47x/tei-platform/issues/1802)で行う。

## 1. この文書の役割

TEIが、設計者の仕事を引き継ぐ機能をどこまで持つかを確かめるため、X-TEA ModelerとX-TEA Editorの機能を洗い出し、次の三つに仕分けた記録です。

- すでにIssueが担当している機能
- 担当するIssueがなく、新しく起票した機能
- TEIでは移植しない機能と、その理由

機能の複製は目的にしません。Java Swingの画面、`.xead`と`.xeaf`の書式、X-TEA固有の識別子や並び順は、TEIの意味の正本へ持ち込みません。移すのは、設計者がその機能で行っていた**仕事**です。仕事の意味と、旧ツールの実行規約や画面操作を分けて扱います。

意味の正本の置き場所、受理の境界、実行権限は、この文書では決めません。それぞれ[製品要求](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、ADR、各Issueが担当します。

## 2. 調査の根拠と制約

| 項目 | 内容 |
| --- | --- |
| Modelerの版 | `68f1535b1c5dd80a327dfced37be401ecd3093c9` |
| Editorの版 | `ee19a27609b2749110ec39b62d4540693347fe09` |
| 読んだもの | 画面資源（英語版）、ダイアログの構成、検査・取込み・出力の処理の見出し |
| 読んでいないもの | 各処理の全分岐、旧Runtimeでの実際の動作、利用者が保持するモデル本体 |
| 保管場所 | 非公開の資料庫（`tei-platform-private-resources`）の`archives/x-tea-2026-10-14` |

画面資源の文言から機能を推定した箇所があります。そのため、各機能の「何を保証するか」は、移植を担当するIssueが着手する時点で、原ソースの該当処理を読み直して確定します。旧Runtimeで動かした結果は取得していません（[#940](https://github.com/hat47x/tei-platform/issues/940)）。

参照モデルの内容と、手引書の本文は、この文書へ転載しません。利用条件の確認は[#941](https://github.com/hat47x/tei-platform/issues/941)の担当です。

## 3. Modelerの機能と移植判断

Modelerは、業務、データ、機能の三つの視点を一つのファイルに保持し、人が編集する道具です。

### 3.1 モデルの対象と編集

| 領域 | X-TEAでの機能 | TEIでの扱い | 担当 |
| --- | --- | --- | --- |
| 業務の流れ | 職務（Role）と作業（Task）、作業の契機（Event）、作業の下位操作（Action）、一階層の業務フロー図、作業種別 | 担当者、契機、手作業、分岐を業務手順の宣言として表し、図の配置は表示情報へ置く | [#1125](https://github.com/hat47x/tei-platform/issues/1125) |
| 業務と機能の対応 | 作業の操作が、機能の入出力画面や帳票を指す。機能を持たない手作業も許す | 業務の操作と機能の対応は一対一に縮約しない。手作業は欠落として扱わない | [#1125](https://github.com/hat47x/tei-platform/issues/1125)、[#1132](https://github.com/hat47x/tei-platform/issues/1132) |
| データの構造 | テーブル、項目、主キー・外部キー・副キー・索引、関連（親子・参照・派生の三種）、項目の属性種別（固有・継承・導出） | 論理的な実体・項目と物理表を分け、キー、関数従属性、関連を同じ意味から投影する | [#1126](https://github.com/hat47x/tei-platform/issues/1126) |
| データの図 | データモデル図、関連線の表示と非表示、項目の表示指定 | 図の配置は表示情報へ置き、正本へ戻す対象は意味に限る | [#1124](https://github.com/hat47x/tei-platform/issues/1124) |
| 具体例 | データモデル上のインスタンス（項目ごとの例値）、スライド順の設定、スライドショー | 型付きの例と、表示順を分ける | [#1131](https://github.com/hat47x/tei-platform/issues/1131) |
| 機能の構造 | 機能種別、機能の入出力（画面・帳票・Webページ・表）、表ごとの作成・参照・更新・削除、呼び出す機能と呼び出される機能、機能の構造図 | 機能の契約と実現を分け、宣言された更新と実際の副作用を区別する | [#1127](https://github.com/hat47x/tei-platform/issues/1127)、[#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#1132](https://github.com/hat47x/tei-platform/issues/1132) |
| 項目の算定 | 導出項目（DERIVED）、更新禁止指定 | 導出値の意味契約と適用先を分ける | [#1637](https://github.com/hat47x/tei-platform/issues/1637)以降のData Derivation系列 |
| サブシステムの所有 | サブシステム、外部テーブル、外部テーブルへの作成・更新・削除を避ける注意表示 | 所有境界を診断として扱い、強制的な不正とは決めない | [#1132](https://github.com/hat47x/tei-platform/issues/1132) |
| 三視点の一覧 | 作業と機能、作業と表、表と機能の対応一覧（行列）、主題領域と作業・機能の対応一覧 | 対応表は導出した表示であり、正本にしない | 新規：[#1803](https://github.com/hat47x/tei-platform/issues/1803) |

### 3.2 一括操作と周辺機能

| 領域 | X-TEAでの機能 | TEIでの扱い | 担当 |
| --- | --- | --- | --- |
| 検索と置換 | モデル全体の文字列の走査と置換。要素種別、属性、大文字小文字の区別、置換の対象を選択 | 検索は読み取り、置換は変更案として受理境界へ渡す | 新規：[#1804](https://github.com/hat47x/tei-platform/issues/1804) |
| 別ファイルからの取込み | 別のモデルから、サブシステム、役割、機能、表、データフローを取り込む。照合・更新規則は対象種別と形式で異なる。M49の既存照合は内部ID一致とは限らず、形式・要素によって更新／追加／skipが分かれる。Dataflowは既存の子を置換してNode/LineのIDを再採番する。未解決参照や先行DOM変更の扱いも経路別で、欠落参照が常にログ出力・skipされるわけではない | 取込みは候補として受け取り、受理を経て反映する。欠落した参照を黙って落とさない | 新規：[#1805](https://github.com/hat47x/tei-platform/issues/1805) |
| 表定義の突合せ | 別ファイルの同じ表との差異の列挙（名称、項目、型、非NULL、既定値、キー、種別）と、取込みによる同期 | 外部の意味名前空間の参照を読み取り専用で扱う。同期は取込みの受理境界へ従う | [#1570](https://github.com/hat47x/tei-platform/issues/1570)、[#1805](https://github.com/hat47x/tei-platform/issues/1805) |
| 版間の差異 | 二つのファイルを比べた変更一覧と、変更に対応するDDLの生成 | 意味の差分と依存の影響を共通機能にする | ISSUE-000034（[#362](https://github.com/hat47x/tei-platform/issues/362)から計画文書へ移管） |
| データ型の再編 | データ型の統合（項目を別の型へ移し、元の型を削除する） | 意味の型の統合と、影響を受ける項目の一覧 | 新規：[#1806](https://github.com/hat47x/tei-platform/issues/1806) |
| DDLの取込み | CREATE TABLE文から表、項目、関連を読み込む。型の対応、非表示項目、コメントの扱いを指定 | 曖昧さを保った候補として読み込む | [#1126](https://github.com/hat47x/tei-platform/issues/1126) |
| DDLの出力 | CREATE文の生成。表名の方式、コメントの方式、外部キー制約の有無、命名の変換、キー定義の警告 | 方言ごとのプラグインが担当し、意味から導出する | [#1803](https://github.com/hat47x/tei-platform/issues/1803)（出力の入口）、データベース方言プラグイン設計 |
| 設計書の出力 | 表定義書と機能定義書をシート単位で出力。機能の画面イメージ、入出力、呼び出し関係、関連作業を含む | 受理した意味から導出した成果物として出す。書式を意味へ戻さない | 新規：[#1803](https://github.com/hat47x/tei-platform/issues/1803) |
| 一覧の出力・取込み | 作業一覧、表一覧、機能一覧を表計算形式（CSV）で出力する。CSVの取込みでは、名称で対象を特定してIDを更新する | 出力は導出成果物。取込みは変更案として扱う | 新規：[#1807](https://github.com/hat47x/tei-platform/issues/1807) |
| 設計の補助情報 | 用語と規則、改訂履歴、注記（`@NOTE@`の印）、しおり、区分（主題領域、作業種別、表種別、機能種別、課） | 意味の正本、表示情報、運用上の記録のいずれかへ分類する | 新規：[#1808](https://github.com/hat47x/tei-platform/issues/1808) |
| 保存と編集履歴 | 保存前の退避、他の利用者が更新した場合の確認、要素ごとの取消とやり直し | 版を基準にした競合検出と復旧 | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#1529](https://github.com/hat47x/tei-platform/issues/1529) |
| 表示言語 | 日本語・英語・韓国語の画面資源 | 意味と表示資源を分ける多言語対応 | [#1551](https://github.com/hat47x/tei-platform/issues/1551)、[#1649](https://github.com/hat47x/tei-platform/issues/1649) |

## 4. Editorの機能と移植判断

Editorは、Modelerの機能定義から実行用の機能定義を作り込み、データベースを接続して確かめる道具です。三つの視点のうち機能の視点を詳細化する役目を持ちます。

| 領域 | X-TEAでの機能 | TEIでの扱い | 担当 |
| --- | --- | --- | --- |
| 機能の詳細化 | 十種の機能種別（照会一覧、更新一覧、単票、ヘッダ明細など）に応じた定義、結合表と結合条件、並び順、列と絞り込み、ボタン | 機能の契約と、標準部品（Application Pattern）の選択・案件差分を分ける | [#1127](https://github.com/hat47x/tei-platform/issues/1127) |
| 機能の追加支援 | 表から機能を作る補助、参照表の追加、明細表の追加、結合表の追加、項目の追加 | 表の意味から標準機能を直ちに作る経路 | [#1127](https://github.com/hat47x/tei-platform/issues/1127) |
| スクリプト | 機能の処理手順の記述、使用する表・項目の索引、スクリプト内の文字列検索 | 任意の処理手順を外へ出さず、宣言と実現プラグインへ分ける | [#1129](https://github.com/hat47x/tei-platform/issues/1129) |
| 機能の呼び出しの検査 | 呼び出される機能の未定義、呼び出されない機能、明細機能や見出し機能の未指定 | 機能の呼び出し関係の検査 | [#1132](https://github.com/hat47x/tei-platform/issues/1132) |
| 画面配置の検査 | 列や絞り込み条件、入力項目の配置の確認 | 表示の検査は画面部品側の担当 | [#997](https://github.com/hat47x/tei-platform/issues/997) |
| 物理表との照合 | 定義と実際の表の、型・桁・非NULL・主キー・副キー・索引の照合、表の作成・変更・削除 | 望む状態と実際の状態の差異、移行の安全境界 | 実装済み（PostgreSQLの望む状態と実際の状態の比較と移行）。全方言は対象外 |
| モデルの取込み | Modelerで作った定義を、機能・表単位でEditorの定義へ取り込む。主表、見出し表、明細表、結合表、キーの整合を検査 | 取込みの受理境界と検査 | [#1805](https://github.com/hat47x/tei-platform/issues/1805) |
| 変更一覧 | 二つの定義の差異の列挙 | 意味の差分と依存の影響 | ISSUE-000034 |
| データの取込み | タブ区切りデータの取込み（挿入と更新）、初期データの挿入 | 業務データの検証を伴う取込み | 新規：[#1807](https://github.com/hat47x/tei-platform/issues/1807) |
| 操作記録の閲覧 | 実行時の表操作の記録を条件付きで一覧 | 実行の観測。業務結果と観測を分ける | [#1121](https://github.com/hat47x/tei-platform/issues/1121) |
| データの保守 | 任意の問い合わせの実行（SQL）、表データの直接編集 | 移植しない。意味の検証と権限を迂回するため | 7章 |
| 接続と設定 | データベース接続の確認と再試行、メニュー構成、フォント、既定値 | 配備設定とUI構成の担当 | 案件のUI構成（[#997](https://github.com/hat47x/tei-platform/issues/997)） |

## 5. 新規に起票した移植対象

既存Issueのいずれも担当していない仕事を、次の六つに分けて起票しました。各Issueでは、旧ツールの機能名をそのまま要求にせず、守るべき意味の境界を受入条件に書いています。

### 5.1 設計書・一覧・DDLの導出出力（[#1803](https://github.com/hat47x/tei-platform/issues/1803)）

人が読む定義書、対応表、DDLを、受理した意味から導出する経路です。X-TEAでは、定義書の一枚ごとに画面イメージと入出力を載せ、対応一覧は主題領域、作業、機能、表の間で五種類を持ちます。

守る境界:

- 出力した書式の編集を、意味の正本へ戻さない
- どの版の意味から出力したかを、成果物から辿れる
- 受理前の候補や推測した関係を、受理済みの意味として載せない
- 一覧上のセルを編集可能にする場合は、編集できる正本の箇所を示す

### 5.2 横断検索と一括置換（[#1804](https://github.com/hat47x/tei-platform/issues/1804)）

意味の要素の名称、記述、規則の文言を横断して探し、置換を変更案として扱う仕事です。X-TEAでは、識別子と名称を同じ操作で置換できるため、参照を壊す危険があります。

守る境界:

- 検索は読み取りであり、正本を変更しない
- 置換は、対象、置換前後、影響を示す変更案とし、受理を経て反映する
- 表示名の変更と、論理識別子の変更を同じ操作にしない
- 置換の対象を、意味の記述、表示用の文言、スクリプトの本体から選択できる

### 5.3 別モデルからの部分取込み（[#1805](https://github.com/hat47x/tei-platform/issues/1805)）

別の設計から、サブシステム、役割、機能、表、業務フローを取り込む仕事です。固定M49の取込み照合は形式・要素ごとに異なり、内部ID一致を共通upsert規則とすることはできません。Function/Table等の照合とDataflow子要素の置換・再採番も、一つの「ID一致による更新」にまとめられません。未選択の参照先へ既存bindできる場合、参照を解決できず全コピーを見送る場合、先行DOM更新だけが残り得る場合、変換途中で例外に至り得る場合を区別します。

守る境界:

- 識別子の一致を、同じ意味であることの根拠にしない。同じ識別子で意味が違う場合を検出する
- 依存先が欠けた定義を、黙って落とさず、未解決として保持する
- 主キーのように、別の定義が依存する要素の変更を、関連があれば拒否する
- 取込み元が別の所有境界にある場合を診断する
- Dataflowの元Node IDが重複する場合も、最初の一致への端点再対応付けを意味上の一意な解決とみなさない。M49の`DialogImportXEAD.importDataflow()`は元/先Node IDを並行配列に蓄積し、Lineの`NodeID1/NodeID2`を`nodeIDListFrom.indexOf(...)`で付け替える（L1052–1116）。同一元Node IDが複数ある場合は先頭の一致に寄り、未一致の場合は`get(-1)`へ至り得る（静的導出、実行未確認）。詳細と既存受入例は[#1805のDataflow証拠](https://github.com/hat47x/tei-platform/issues/1805#issuecomment-6207997267)に集約する
- `DataflowLine@ID`の重複は、元`DataflowNode@ID`の重複や同一Nodeへの複数参照とは別に診断する。M49の`DialogImportXEAD.importDataflow()`（L1091–1116）は元Lineの`ID`を照合せず、元DOMの各Lineを順次複製し、取込先DOMの最大`DataflowLine@ID`+1を新IDとして`appendChild`する（最大値探索は`getLastDomElementOfTheType()` L2823–2841）。元DOMに同じLine IDが複数あり、Task/Role・端点解決が正常に進みこの複製ループへ到達する条件では、個別のLineとして再採番され、同一元IDを根拠に上書き・統合されない（静的導出、未実行）。これは**部分Importの当該ループ限定**の観察で、通常Open/Saveの重複ID拒否・保存後の表示結果・M50配布版は未確認。TEI側では元IDが衝突した事実と各要素のsource locatorを残し、再採番で曖昧性を隠さない（[#1805](https://github.com/hat47x/tei-platform/issues/1805)、[#950](https://github.com/hat47x/tei-platform/issues/950)）。
- 外部の意味名前空間の読み取り専用の参照（[#1570](https://github.com/hat47x/tei-platform/issues/1570)）と、定義の取込みを混同しない

固定M49ソースの静的照合による**XEADとXEAFの取込規則の差**（旧Runtime実測でもTEI採用仕様でもない）:

- XEAD `DialogImportXEAD.importTables()`（`xead/XeadModeler@68f1535b1c5dd80a327dfced37be401ecd3093c9`、609–685行）は、選択元Subsystemかつ非空`Table.SortKey`のTableを対象にし、既存Tableの同一`SortKey`一致時、対象Subsystemも一致すれば`updateTableDefinition()`、所属が異なればスキップ、一致先がなければ`createTableDefinition()`へ進む。照合はSource `SortKey`を用い、同一キーの意味的同一性は証明しない。
- XEAF `DialogImportXEAF.importTables()`（同commit、341–428行）は、選択元SubsystemのSource `Table@ID`と取込先Tableの`SortKey`を比較し、一致した場合は**更新せずスキップ**、不一致なら新規Tableを作成する。XEAF `importFunctions()`（430–509行）も既存Function一致時はスキップ。**両形式の「一致＝update」を共通仕様にしない**。
- XEAF `importSubsystem()`（840–846行）は選択された取込先Subsystemの`Name`/`Descriptions`へSource属性を無条件に`setAttribute`する。DOM `getAttribute`の属性未設定時と明示空値の区別はこの代入で失われ得る。これはTable/Function一致時のskipとは別経路。
- XEAFの`request()`（187–239行）と`jButtonStart_actionPerformed()`（262–320行）は元・先Subsystemを選び、`importTables()`（358–364行）は元Subsystemに属するTableだけを対象とする。`XeadImportLog<日時>.txt`には元・先、選択範囲、作成/skip等が記録されるが、**全Source項目の完全inventory・除外理由・source削除の証明ではない**。部分選択による欠落をdelete/tombstoneへ自動変換しない。
- TEIでは#950のsource-faithful観測と#1805の取込候補・衝突/欠落・#267の明示受理を分離する。同一論理Source項目のstable key/版/意味同一性/更新方針は別途検証し、旧ID・SortKey・XML位置をCanonical identityへ昇格しない。取込後の再実行（内部ID・子要素・参照の冪等性）と、元ファイルの完全snapshotに基づく削除検知は**未実証**。

根拠: [XEAD固定source](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAD.java#L609-L685)（blob `691f5e3c4071a23825f26012f71ebfa3a864e47c`）、[XEAF固定source](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAF.java#L341-L509)（blob `2ebd83e61595c1195311472039a8ffd90bfb9936`）。M50配布JARでの同一性、GUI実行、TEIのwrite/upsert実装済み保証は未確認。

#### 固定M49の証拠：未選択参照・entity-kind・部分適用（2026-10-26、静的照合）

本追補は既存[#950](https://github.com/hat47x/tei-platform/issues/950)・[#1805](https://github.com/hat47x/tei-platform/issues/1805)の責務を変更せず、追加の一般Importerを設計しない。過去#2会話・archiveの全文を回収したという意味ではない。

- **固定資料**：XeadModeler M49 source commit `68f1535b1c5dd80a327dfced37be401ecd3093c9`、[XEAD Importソース](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAD.java) Git blob `691f5e3c4071a23825f26012f71ebfa3a864e47c`、保存SHA-256 `1217eb273e24f857de5777740bd51c121078e2131febcd0f41f1dcff149bab08`。[XEAF Importソース](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAF.java) Git blob `2ebd83e61595c1195311472039a8ffd90bfb9936`。配布M50 `xteamdl.jar` SHA-256 `0305791a8f5a61291d436f7f04dc8ff1180fa79841bda88b01524d636fb9f04f`は成果物の識別のみで、M49とのbytecode同等性・実動作を証明しない。
- **参照先が未選択（M49確認事実）**：XEAD `checkTaskDefinition():926–945`は、取込元`DataflowNode@TaskID`を元`Task`に解決してSortKeyを読み、取込先`Task`の最初の一致IDを返す。`importDataflow():988–1035`はProcess→TaskとSubject→Roleを型別に検査し、未解決カウンタがあればDataflowコピーを見送る。参照先TaskはDataflow取込みで再作成しない。Roleの数値ID一致だけでTaskとしてbindする経路ではない。
- **dangling / entity-kindの境界（M49事実＋静的推論）**：`checkTaskDefinition():927–945`は、元Taskが見つからない場合`taskSortKey=""`のまま取込先TaskのSortKeyを検索する。このため同値の空SortKeyを持つ取込先Taskがあれば、そのIDへ誤bindし得る（**推論・実行未検証**）。「元Taskが欠落し、同じ数値IDのRoleだけがある」条件を、単なる未選択Taskへの正常bindと区別する。sourceでtagが違うことは一致の根拠にせず、失効／kind不一致／空キー照合を別々に未解決候補として保持する。Taskの存在だけで業務意味受理を主張しない。
- **逐次書込みと復元（M49確認事実＋静的推論）**：XEAD `importTables():609–711`は共有型→Table群→Relationshipの順にDOMを操作し、`createTableDefinition():1720`は子Field生成前にTableを編集中DOMへ追加する。`:1725–1746`のField生成途中で例えば`:1733`の数値変換が失敗した場合、`jButtonStart_actionPerformed():589–598`の例外処理に以前のDOM変更を一括復元するコードはない。先行Tableや型定義のDOM変更残存は**静的推論**であり、実DBでの部分commitやディスク保存結果ではない。元Import内の失敗と、再import時の冪等性・親欠落・ID衝突は別の受入論点である。
- **既存受入との対応**：[#950](https://github.com/hat47x/tei-platform/issues/950)は原典のkind・ID・親scope・SortKey・参照をsource factとして保存し、Canonical identityへ昇格しない。[#1805](https://github.com/hat47x/tei-platform/issues/1805)には欠落参照保持・意味衝突診断・受理前の正本不変が**既に存在**する。追加の**合成負例候補（未実行）**は、(a)未選択で元/先Taskが一致する正常bind、(b)元Taskがdanglingで異種別Roleだけ同じ数値ID、かつ取込先Taskが空SortKeyの異常入力、(c)先行Table追加後に後続Field変換が失敗しても正本不変、の3件を独立に確認すること。診断・原子性は既存#1805/#267へ渡し、Source Adapterに適用権限を持たせない。関連する証拠は[#1805 Dataflow](https://github.com/hat47x/tei-platform/issues/1805#issuecomment-6207997267)、[#1805 Table依存](https://github.com/hat47x/tei-platform/issues/1805#issuecomment-6210000415)。
- **未確認**：M49実行/GUIと保存後XML、M50 bytecode／配布版動作、旧#2 archive全履歴の一次再取得、TEIの全Target適用での正本原子性、合成負例のPASSは未確認。旧ツールの静的挙動からTEI受入済みを推定しない。

#### M49固定版に見るXEAD parser・Dataflow参照の失敗境界（2026-10-27、静的確認）

**比較元**：XeadModeler commit [`68f1535b1c5dd80a327dfced37be401ecd3093c9`](https://github.com/xead/XeadModeler/tree/68f1535b1c5dd80a327dfced37be401ecd3093c9)。`DialogImportXEAD.java` のGit blobは `691f5e3c4071a23825f26012f71ebfa3a864e47c`、`DialogDataflowLine.java` は `319d3a6eb9914fee825098b121c78ef6a0d5f355`。以下は旧ツールのソース上の事実と条件付き推論であり、TEIの実装証拠ではない。

- **解析失敗とモデル状態**：`DialogImportXEAD.request(String)` [221–245行](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAD.java#L221-L245) は `domDocumentFrom=null` にしてから解析し、`IOException`/`SAXException`を空のcatchで処理した後、null確認なしで`getElementsByTagName("System")`へ進む。この**部分Import入口**には取込先`frame_.domDocument`を消去する操作はないが、通常Open全体のエラー後状態を保証するものではない。
- **namespaceと未知子要素**：`getElementsByTagName("System")`等はタグの**修飾名**による検索で、namespace URIの正しさを検証しない。接頭辞付き`x:System`は文字列`System`とは一致せず、既定namespaceの`System`はURIが異なっても名前検索だけでは区別できない。取込元DOMの構文解析成功は、取込先への未知データ保持を意味しない。特に`importDataflow()` [1040–1116行](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAD.java#L1040-L1116) は既存子要素を全削除し、既知ノード・線を再構築するため、未知子要素の引継ぎを保証しない。Table更新には既存要素を再利用する別経路もある（[1531–1697行](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAD.java#L1531-L1697)）。
- **欠落・重複参照ID**：`importDataflow()` [1052–1116行](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogImportXEAD.java#L1052-L1116) は元Node ID→新Node IDの対応表を作り、Line端点を`indexOf`で変換する。欠落IDは`get(-1)`に至り得る一方、**重複した元ID**は最初の一致へ束縛され得る。いずれも既存子要素を削除した後の処理であり、途中失敗を取り消すrollbackは確認できない。取込時の欠落端点は[#1805既存記録](https://github.com/hat47x/tei-platform/issues/1805#issuecomment-6207997267)、重複IDは[追補](https://github.com/hat47x/tei-platform/issues/1805#issuecomment-6418435553)を参照する。
- **編集UIでの欠落端点**：`DialogDataflowLine.request()` [374–448行](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/DialogDataflowLine.java#L374-L448) は一致しないNode IDでも選択index 0が残る経路を持つ。表示される先頭候補を確定参照とみなさない（[#1125追補](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6418455306)）。

**TEIへ引き継ぐ検証候補**：Source解析失敗、未知・namespaced構造、参照ID欠落／重複、候補更新途中失敗を区別する。未解決参照を初回一致や表示上の選択で補完せず、Sourceの来歴と失敗理由を保持し、明示的に受理するまではAccepted正本を変更しない。Source Adapterの既存qualified-name／unknown保持契約は[#950](https://github.com/hat47x/tei-platform/issues/950)を再利用し、別のparserやAcceptance機構を増やさない。

**未確認**：M50配布JARとの同等性、通常Open経路の全体状態、GUI・保存後ファイル、旧実モデルへの出現、TEIの負例試験／full Repository Health。旧ツールのソース静的確認を製品動作・TEI適合の実証に読み替えない。

### 5.4 データ型の再編（[#1806](https://github.com/hat47x/tei-platform/issues/1806)）

意味の型（コード、金額、数量など）を整理し、複数の型を統合する仕事です。X-TEAでは、型の定義が桁と基本型とSQL上の表現を持ち、項目が型を指します。型を統合すると、キーの結合に使う項目の型が一致しなくなる場合があります。

守る境界:

- 型の統合が、キーや関連の整合に与える影響を、変更前に示す
- 業務上の意味が異なる型を、見た目の桁数が同じという理由で統合しない
- 物理的な型の変更と、意味の型の変更を分ける

### 5.5 設計の補助情報の置き場所（[#1808](https://github.com/hat47x/tei-platform/issues/1808)）

用語と規則、改訂履歴、注記、しおり、主題領域などの区分を、意味の正本、表示情報、運用の記録のどれに置くかを決める仕事です。X-TEAでは、注記を記述欄の中の特定の印（`@NOTE@`）で表し、一覧にしています。

**M49 Sourceの追加確認（TEI要件ではない）:** `System@DocFileFormat`はセミコロン区切りの外部資料候補として読み、`&ID`・`&NAME`・`<CURRENT>`を置換してFunctionの資料一覧を構成します。M49ではSystem定義から読み書きし、`xteamdl.properties`のキーとして参照するコードは確認できません。これは単なる設計書出力形式ではなく、Functionに付く外部資料参照を扱うSource factです。TEI側での意味・保管先の対応付けは未決で、[#1802のSource-only追補](https://github.com/hat47x/tei-platform/issues/1802#issuecomment-6499985744)に証拠を記録しました。旧書式・token構文をTEI契約へそのまま移しません。

守る境界:

- 記述欄の中の印のような、文字列に埋め込んだ約束を、構造化した宣言へ置き換える
- 改訂履歴を、版管理から導出できる事実と、人が書く理由に分ける
- 用語を、名称の索引と、意味の定義に分ける

### 5.6 一覧の往復とデータの取込み（[#1807](https://github.com/hat47x/tei-platform/issues/1807)）

人が表計算で編集した一覧を取り込む仕事と、業務データ（タブ区切り、初期データ）を取り込む仕事です。

守る境界:

- 名称で対象を特定してIDを更新する方式を、そのまま取り入れない。名称の重複と変更を検出する
- 取り込んだ一覧の編集は、変更案として扱う
- 業務データの取込みは、型・参照・制約の検証を伴う。検証を迂回する経路にしない
- 初期データを、受理した意味の一部と、案件の運用データに分ける

## 6. 移植時に落としやすい暗黙前提

原ソースの処理から読み取れた、旧ツールでは暗黙に成り立っていた前提です。移植を担当するIssueは、これらを明示的に扱います。

| 暗黙前提 | 旧ツールでの現れ | TEIで必要な扱い |
| --- | --- | --- |
| 識別子と並び順が一つの値 | `ID(seq)`が識別と一覧順を兼ねる | 論理識別子と表示順を分ける |
| 取込みの同一性判定が単純なID一致 | M49 XEADのTableはSortKey＋所属Subsystem、子TableFieldは親Table.SortKey＋Alias／Nameで照合する。XEAF Functionは元IDを既存SortKeyと比較して一致時skipする別方式 | 形式・entity-kind・親scope・照合キー・source来歴を保持し、意味の同一性やCanonical IDへ暗黙に昇格させない（#950/#1805） |
| 取込みの照合・付替え規則が要素種別で異なる | Function/TableのSortKey照合と、Dataflowの子要素置換・ID再採番・元Node IDの先着対応が併存する（Modeler M49） | 旧IDをCanonical identityにせず、衝突候補と由来を区別する |
| 取込みが部分的に成功する | 欠落時は経路によりskip・一部DOM変更・例外があり得る。DataflowLine端点欠落は`indexOf=-1`から例外に至る経路がある（M49、実行未確認） | 欠落を未解決として保持し、受理前に副作用を起こさない |
| 関連の条件付きは制約にしない | DDLの出力で、条件付きの関連は外部キーにならない | 条件付きの関連を、実行側の規則として宣言する |
| 型の不一致は警告にとどまる | キーの型が一致しないと警告し、DDLは出力する | 型の不一致を診断として固定し、続行の条件を決める |
| 画面操作の文が業務の意味 | 作業の操作に「ボタンを押す」等の文が入る | 業務の操作と、画面固有の操作指示を分ける |
| 注記が文字列の印 | 記述欄の`@NOTE@`を一覧化する | 構造化した宣言にする |
| 他の利用者の更新は保存時の確認 | 保存時に更新の有無を確認する | 版を基準にした競合検出 |
| 一つの実行環境と一つの利用者 | 接続、記録、保守が一つの環境で完結 | 複数のTarget、複数の主体、権限を前提にする |
| 形式の版の表示が内容の世代と一致する | Modeler M49の通常openは`FormatVersion`を数値比較し、1.2より大きい値でプロセスを終了する。欠落・非数値は汎用XML読込失敗として終了し、`NaN`と`-Infinity`は版比較を通過する（後続読込成功は未確認）。編集保存では、変更があり旧値が1.2未満なら属性を1.2へ上げる分岐があるが、その分岐は内容移行を検証しない。保存全体には別の正規化もある。M50配布版との同等性は未確認 | 形式の版の表示だけで世代を判定しない。変換を一か所に置き、変換に成功したときだけ版を上げる（[#1920](https://github.com/hat47x/tei-platform/issues/1920)） |
| 外部キーの列対応はKey情報だけで一意に決まる | Modeler M49の関係作成時は型・名称・別名で候補Fieldを照合し、両側のTableKey IDとKey構成Field IDを保存する。この経路では列pairを独立宣言する属性は確認できない。DDLのFK出力は関係種別・存在条件・画面設定で分岐し、逆向きSUBTYPEの曖昧な候補も警告無視の設定では必ずしも`(?)`にならない。M50配布版との同等性と実出力は未確認 | 列の対応を関連の意味として明示し、推測した対応は候補として示す（[#1803](https://github.com/hat47x/tei-platform/issues/1803)） |
| 物理表の照合は値を保つ移行である | XeadEditor M34の表照合は、DB方言別比較で型が非互換なら列をdrop/add候補にする。型が同等でも特定の桁差は同じ候補に加え、主キーがある場合に旧値の退避・変換・復元を試みる。Field ID変更は旧列dropと新列addの候補になり、両者の値を対応付ける経路はない。変更処理はDDL/DMLを順次発行するが、同一transactionでの原子性は明示保証していない（接続設定・DBMS実動作は未確認） | 値を失う可能性と変換条件を示し、差分を段階的な移行へ渡す（[意味の変更に伴う既存データの移行設計](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md)） |
| 楽観ロック列の有無は表属性で制御する | Editor M34では`Table@UpdateCounter`を参照し、未指定なら既定の`UPDCOUNTER`列をField一覧とは別にDDLへ追加する。`*NONE`なら追加せず、別名も指定できる。Driver M34では有効時に旧counter値をUPDATE/DELETE条件へ含める | 同時更新の検出方式と、更新件数が0件のときの業務結果を宣言する（[#1129](https://github.com/hat47x/tei-platform/issues/1129)） |

表の最後の4行は原ソースの静的読解から加えた（Modeler M49、Editor M34、実行系はXeadDriver `6c077773b74c6d7dbcc253835cf8d347e4ffb1ef`）。保存配布物はModeler M50のJARとEditor/Driver M36のJARで、同名クラスが含まれる箇所も原ソースと同じ実装かは確認していない。したがってM50/M36配布版の同等性や旧Runtimeの実動作は未確認である（[#940](https://github.com/hat47x/tei-platform/issues/940)）。

## 7. 移植しないもの

| 機能 | 理由 |
| --- | --- |
| Swingの画面、レイアウト、配色の設定 | 実行環境固有の表示。TEIの表示はTargetと画面部品が担当する |
| `.xead`と`.xeaf`の書式の採用 | 外部の書式。TEIの意味の正本にしない。取込みは候補として扱う |
| 任意のSQLの実行と、表データの直接編集 | 意味の検証、認可、観測を迂回する。診断用の読み取りは別途検討する |
| X-TEAの機能種別をTEIの核の列挙へ複製すること | 互換の再現は目的にしない。標準部品と案件の差分で表す |
| 旧Driver固有の実行規約（呼び出し元に依存する確定、戻り値による取消、ログイン時の処理） | 明示的な境界が必要。[#1129](https://github.com/hat47x/tei-platform/issues/1129)が担当する |
| Modeler・EditorのJava実装の流用 | 公開の許諾条件の確認と、構成の再設計が必要。参照として読む範囲にとどめる |

## 8. 未確認の事項

- 各機能の全分岐と、旧Runtimeでの実際の動作は確認していない。実機での確認は[#940](https://github.com/hat47x/tei-platform/issues/940)に残る。2026-10-15の追加調査で静的に確かめた範囲は第9節にある。
- 画面資源から機能を推定した箇所がある。新規Issueの着手時に、原ソースの処理を読み直す。
- 設計書の出力の既存形式（表計算の書式）と、TEIの成果物の対応は、[#1803](https://github.com/hat47x/tei-platform/issues/1803)で決める。
- 参照モデルの利用条件が確認されるまでは、モデルの内容を成果物の例へ使わない（[#941](https://github.com/hat47x/tei-platform/issues/941)）。

## 9. 旧実装の追加調査（2026-10-15）から担当Issueへ渡す条件

2026-10-15に、公開ソース、配布された実行ファイル、参照モデル12件、参照モデルに付随するデータベースの構造を、読み取りだけで照合しました。詳細な根拠は、非公開の資料庫（`tei-platform-private-resources`）の`research/low-code-management-tools/xtea/N-open-questions.md`にあります。ここには、担当Issueへ渡す結論だけを置き、参照モデルの内容は転載しません（[#941](https://github.com/hat47x/tei-platform/issues/941)）。旧ツールを動かした結果ではなく、静的な読解と構造の照合です。実測は[#940](https://github.com/hat47x/tei-platform/issues/940)が担当します。

| 確認したこと | 渡す条件 | 担当 |
| --- | --- | --- |
| 業務の手順（`TaskAction`）は、12モデルの1,606件のうち、値のある識別子を持つものが0件。719件は空の識別子属性を持ち、残りは属性自体が無い。空の属性は、業務の貼り付けが写したものだった | 識別子が「無い」ことと「空である」ことを原典のまま保全する。識別子を持たない要素は、親の識別子と原典の文書順で位置を特定し、識別子の一致を同一性の根拠にしない | [#950](https://github.com/hat47x/tei-platform/issues/950)、[#1805](https://github.com/hat47x/tei-platform/issues/1805) |
| 廃止済みの表属性（有効範囲と範囲キー）が、173〜174表に空の値で残り、値があるのは1表だけだった。実行系は2016年に読まなくなっている | 「定義にはあるが実行時に効かない」値は、黙って捨てず、警告として保持する。値のある1件は損失台帳へ原文のまま残す | [#1920](https://github.com/hat47x/tei-platform/issues/1920)、[#950](https://github.com/hat47x/tei-platform/issues/950) |
| 帳票の旧形式を、編集環境は6属性すべて移して新形式へ変換するが、実行系は4属性を使わず、見出し側の属性を明細に使う。旧形式で結合列を持つ23件のうち12件で、結合に使う列が変わる（推定。実行は未確認） | 旧形式の変換は一か所に置く。読み手ごとに別の変換を持たない。負例として、旧形式の見本が、どの読み手を通っても同じ結果になることを試験する | [#1920](https://github.com/hat47x/tei-platform/issues/1920) |
| 別モデルの取込みは、2015年からキーの種別（索引を含む）も写すが、利用者向けの説明は主キーと副キーだけと書いたままだった | 取込みの範囲を、説明文ではなく試験で固定する。取込み候補に、写される要素の種別を一覧する | [#1805](https://github.com/hat47x/tei-platform/issues/1805) |
| 配布された実行ファイルは、公開ソースの最新版に機能を足した別の版だった。削除処理の条件が逆になっている欠陥は両方に残る。サーバーの認証の不具合は、配布版だけが直っている | 旧Runtimeの実測では、対象の版（配布物の識別）を記録する。ソースの読解を実測と呼ばない。実測で見つけた欠陥は「原典の挙動」と「意図した意味」を分けて記録し、TEIは欠陥を再現しない | [#940](https://github.com/hat47x/tei-platform/issues/940)、[#1969](https://github.com/hat47x/tei-platform/issues/1969) |
| 付随するデータベースは、9件とも定義と表・列・型・索引の数まで一致し、全表に楽観ロックの列（整数、既定値0）があった。1件だけ、定義が新しく、制御表が1つデータベースに無い。どのデータベースも正常に停止しないまま保存されていた | 既存のデータベースに競合検出の列があることを前提にせず、無い場合の縮退を宣言する。定義とデータベースの版のずれを、移行の段階の入口で検出する | [#2096](https://github.com/hat47x/tei-platform/issues/2096)、[#2147](https://github.com/hat47x/tei-platform/issues/2147) |

### 9.1 まだ確認していないこと

- 付随するデータベースの行の中身と、ログを再適用した状態の件数。起動と書込みを伴うため、原本から展開し直した複製で行う必要があります。
- 列の長さと桁、制御表の既定の行の中身。
- 旧形式の帳票12件が、実行時に誤った明細を出すか。旧Runtimeの実測（[#940](https://github.com/hat47x/tei-platform/issues/940)）が必要です。
