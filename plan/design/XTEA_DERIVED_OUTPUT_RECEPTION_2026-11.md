# X-TEA派生出力の受容設計 — 設計書・DDL・行列・差分（2026-11）

## 1. この文書の役割

X-TEA（XEAD）の Modeler が持つ**派生出力**（設計書、DDL、行列、差分、一覧の往復）を読み、TEIの[導出成果物](../design/DERIVED_DESIGN_ARTIFACTS_DESIGN_2026-11.md)と[受容アーキテクチャ](XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)のL3へ受け止める形を整理します。

- 個別機構は[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §4.7（O-1〜O-6）にあります。この文書は、そこから先の**契約**（版・未解決・非昇格）を具体化します。
- 判断は**仮承認**です。受容の採否は別途です。
- 根拠は固定版（XeadModeler `68f1535b1c5dd80a327dfced37be401ecd3093c9`）の静的な読解です。配布JARの動作、実DBでの実行ではありません（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §6）。

## 2. 観測した機構

### 2.1 設計書は表ごと・機能ごとに1シートを作る

`DialogDocuments`は、種別（表定義書／機能定義書）とサブシステムを選び、`generateTableDocument`・`generateFunctionDocument`で1要素＝1シートのブックを作ります（`M:DialogDocuments.java L505–548`）。

シート名は`(<SortKey>)<Name>`、フッターは`(<subsystem SortKey>)<subsystem Name> - (<SortKey>)<Name>  Page &P / &N`です（同 L507–514、L530–538）。並び順の識別子（`SortKey`）が、名前と一緒に出力へ入ります。

表のシートは5つの節を持ちます。

| 節 | 関数 | 内容 |
| --- | --- | --- |
| 概要 | `setupTableSummary` | 表の要約 |
| 項目一覧 | `setupTableFieldList` | 項目 |
| キー一覧 | `setupTableKeyList` | 主キー・副キー |
| 使われ方 | `setupTableUsageList` | 機能からの使われ方 |
| キー詳細 | `setupTableKeyDetails` | キーの構成 |

関数のシートも5つです（`setupFunctionSummary`・`setupFunctionIOList`・`setupFunctionsCalledByThisFunction`・`setupFunctionsCallingThis`・`setupTasksUsingThisFunction`、同 L550–1403）。**呼出し元と被呼出しの両方**を別の節に持ちます。

1要素の生成に失敗しても処理は止まらず、`countOfErrors`と`errorFunctionNames`に積みます（同 L376、L518–524）。出力の最後に「要素数」と「エラー数」を出します（同 L376）。

### 2.2 DDLはオプションで生成する

`DialogCreateTableStatement`は、テーブル名の扱い（モデルの名前／別名）、コメント行の文字、セクション区切り、`COMMENT`句／`COMMENT-ON`／コメント無し、参照制約を組み込まない、キー定義の警告を無視する、キャメル記法をアンダーバー区切りへ変換する、追加句、を選べます（`M:DialogCreateTableStatement.java L94–140`、`M:Res_ja.java`の`DialogCreateTableStatement03`〜`17`）。

**なまの`CREATE TABLE`文**を出す機能でもあります（`E:DialogCreateTableStatement`相当のヘルプ文は`E:CreateTableComment`に、有効なデータ型の列挙つきで残っています）。

### 2.3 行列は5種、区分は凡例に依存する

`DialogMatrixList`は5種の行列（業務フロー＆業務、業務フロー＆機能、業務＆機能、業務＆テーブル、テーブル＆機能）をシートとして出します（`M:DialogMatrixList.java L121–134`、L280–388）。セルは`○`（`S80`）で表し、凡例は`Res`の文言です。

### 2.4 差分はファイル単位で、DDLへの対応づけを持つ

`DialogToListChangesOfFiles`が2つのモデルの差分を出し、`DialogTableSynchronize`が実DBとの同期を持ちます（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) O-4）。

### 2.5 一覧の往復は名称で対象を特定する

`DialogImport`がCSV・タブ区切りを取り込み、名称で対象を特定します（O-5）。

## 3. 受け止める形

### 3.1 成果物は「版・規則・未解決」を持つ

X-TEAの設計書は、どの版から出したかを内容に持ちませんでした（シート名とフッターは`SortKey`と名前だけ）。TEIの導出成果物は、`inputRevision`・`rule`・`generator`・`unresolved[]`を持ちます（[導出成果物の契約](../design/DERIVED_DESIGN_ARTIFACTS_DESIGN_2026-11.md) §2）。

**受け止める形**

- シート名・フッターに相当する**人間向けの見出し**とは別に、成果物のmanifestへ`inputRevision`（`worldFingerprint`）を必ず入れます。
- `SortKey`のような**並び順の識別子**は、成果物の安定した順序付けに使います。ただしIDの同一性の根拠にはしません（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) M-1）。

### 3.2 生成の失敗は「未解決」として残す

X-TEAは1要素の失敗を数え、最後に件数だけを出しました（`countOfErrors`）。どの要素の何が足りないかは、メッセージ文字列に閉じます。

**受け止める形**

- 生成できなかった要素は、`unresolved[]`へ**要素・理由・必要なら候補**を載せます。空欄で埋めません（契約4の「非隠蔽」）。
- 件数の集計は、`unresolved[]`から機械的に作ります。X-TEAの`countOfErrors`に相当する値は、成果物から再計算できる従属値になります。

### 3.3 DDLのオプションは「規則」として識別する

X-TEAのDDLオプション（コメント方式・FK省略・命名変換・追加句）は、ダイアログの状態で決まり、出力に記録されませんでした。

**受け止める形**

- オプションは`rule{id, version, fingerprint}`として識別します。同じ受理版でも規則が違えば別の成果物です。
- 「参照制約を組み込まない」のような**意味を削る選択**は、`unresolved[]`へ「参照制約を意図的に省略した」と残します。黙って落としません。
- 命名変換（キャメル→アンダーバー）は物理名の規則であり、意味の識別子を変えないことを宣言します。

### 3.4 行列のセルは種別を持つ

X-TEAの行列は`○`で、セルの意味は凡例に依存しました（O-3）。

**受け止める形**

- セルは`create/read/update/delete`の列挙（`IOTable`の`OpC`・`OpR`・`OpU`・`OpD`に対応）を持ちます。`○`に潰しません。
- 対応の欠落は、空欄ではなく`unresolved[]`に載せます（#1126・#1125の対応が無いことは、それ自体が発見）。

### 3.5 差分は要素単位ではなく意味の変化として出す

X-TEAの差分はファイルと要素の単位でした（O-4）。

**受け止める形**

- 差分は「名前の変更」「型の変更」「制約の追加」「対応の欠落」を区別します（契約2の「意味の差分」）。
- 差分からDDLへ進む場合、DDLは**差分に含まれる変更だけ**を出し、適用手順をmanifestへ記録します。

### 3.6 一覧の往復は版と識別子で照合する

X-TEAの取込みは名称で対象を特定しました（O-5）。名称の重複・変更を検出できず、IDを更新する経路がありました。

**受け止める形**

- 取込みは**識別子**（Canonical ID）で照合し、名称は表示にだけ使います。
- 取込み元の受理版を要求し、現在の受理版と異なる場合は適用しません（AC-9の境界と同じ）。
- 取込み結果は未受理の提案として置きます。

## 4. 受容先の層

| 成果物 | 層 | 置くもの | 置かないもの |
| --- | --- | --- | --- |
| 設計書（表・機能） | L3 | 要素の要約、項目・キー・使われ方、版・規則・未解決 | 意味の第二の正本 |
| 行列 | L3 | 5種の対応とセルの種別、欠落の記録 | 対応の推測 |
| DDL | L3 | 生成規則、制約名、適用順序、省略の記録 | 実DBの状態 |
| 差分 | L3 | 意味の変化の種別、適用順序 | 意味の受理 |
| 一覧の往復 | L5 | 取込み案、版照合、未受理の提案 | 自動受理 |

## 5. 実装の順序（案）

1. **成果物のmanifest**: 種別・`inputRevision`・`rule`・`generator`・`unresolved[]`を共通の形にする（既存の`semanticFingerprint`を入力版として使う）。
2. **設計書（表・機能）**: 5節ずつを投影し、生成失敗を`unresolved[]`へ載せる。
3. **行列**: セルを種別付きの列にし、欠落を`unresolved[]`へ載せる。
4. **DDL**: オプションを`rule`として識別し、意味を削る選択を記録する。
5. **差分**: 意味の変化の種別を出し、DDLへ渡す。
6. **一覧の往復**: 識別子照合と版要求を入れ、案を未受理で置く。

## 6. 未確認の事項

- `setupTableSummary`・`setupFunctionSummary`などの各節が、実際にどの属性を読むかの全量は確認していません（関数名と生成の流れは確認済み）。
- `DialogTableSynchronize`の差分判定の規則、`DialogToListChangesOfFiles`の差分の単位は、静的な読解の範囲です。
- 設計書のExcel体裁（フォント・列幅・改ページ）は再現対象ではありません。
- 配布JAR（Modeler M50）と固定ソースの同一性は未確認です。
- 実DBでのDDL適用（PostgreSQL等）は、[#940](https://github.com/hat47x/tei-platform/issues/940)と既存の実DB証拠が担当します。

## 7. 関連

- [X-TEA知見の受容アーキテクチャ](XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)（D7、D8）
- [X-TEA作者環境の受容設計](XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md)（Studioの基本機能）
- [導出成果物の設計](DERIVED_DESIGN_ARTIFACTS_DESIGN_2026-11.md)
- [知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §4.7
- [移行計画](../workstreams/XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md)
- [作者支援ツールの移植対応表](AUTHORING_TOOL_FEATURE_TRANSFER_MAP_2026-10.md)
