# X-TEA作者環境の受容設計 — TEI Studioの基本機能へ渡す形（2026-11）

## 1. この文書の役割

X-TEA（XEAD）の Modeler・Editor・Driver を読み、**作者環境（TEI Studio）とローコード開発の基本機能**として受け止める形を整理します。受容先は[受容設計](XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)の層（L0〜L6）を使い、この文書は主に **L5（設計・編集環境）** と **L4（標準業務の提供元）** を扱います。

- 個別機構の一覧は[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)（§4.8 A、§4.9 B、§4.6 S）にあります。この文書は、そこで「受容先」だけが書かれていた部分を、Studioの操作・契約・診断へ具体化します。
- 判断は**仮承認**です。受容の採否は別途です。
- 根拠は固定版ソースの静的な読解です。配布JARの動作、実DB、旧Runtimeの実測ではありません（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §6）。

## 2. 読解の対象

| 対象 | 版 | 読んだ範囲 |
| --- | --- | --- |
| XeadEditor | `ee19a27609b2749110ec39b62d4540693347fe09` | `Editor.java`、`DialogCheckLayout.java`、`DialogCheckTableModule.java`、`DialogCheckFunctionsCalled.java`、`DialogScan.java`、`DialogAddFunction.java`、`DialogSQL.java`、`Res_ja.java` |
| XeadModeler | `68f1535b1c5dd80a327dfced37be401ecd3093c9` | `DialogMatrixList.java`、`DialogCreateTableStatement.java`、`DialogScan.java`、`Modeler.java`、`Res.java`、`Res_ja.java` |
| XeadDriver | `6c077773b74c6d7dbcc253835cf8d347e4ffb1ef` | `Session.java`、`XFScriptable.java`、`XF100.java`、`XF200.java`、`XF290.java`、`XF310.java` |

補足として、配布物の同一性は未確認です（[X-TEA startup JARの配布証拠](XTEA_STARTUP_JAR_DISTRIBUTION_EVIDENCE_2026-10.md)）。

## 3. 作者環境で観測した機構と、受け止める形

### 3.1 画面配置の検査は「種別ごとの面」に分かれている

`DialogCheckLayout`は、機能型と対象（列・絞り込み・項目・明細タブ）ごとに別の面を持ちます。

| 面 | 対象 | 根拠 |
| --- | --- | --- |
| `Function100ColumnList` | XF100の列 | `E:DialogCheckLayout.java L135` |
| `Function100FilterList` | XF100の絞り込み | 同 L138 |
| `Function110ColumnList` | XF110の列 | 同 L141 |
| `Function110FilterList` | XF110の絞り込み | 同 L144 |
| `Function110BatchFieldList` | XF110の一括更新項目 | 同 L147 |
| `Function200FieldList` | XF200の項目 | 同 L150 |
| `Function200TabFieldList` | XF200のタブ項目 | 同 L153 |
| `Function300HeaderFieldList` | XF300の見出し項目 | 同 L156 |
| `Function300DetailFieldList` | XF300の明細（タブ単位） | 同 L159 |
| `Function300DetailFilterList` | XF300の明細の絞り込み | 同 L167 |
| `Function310HeaderFieldList` | XF310の見出し項目 | 同 L175 |
| `Function310DetailFieldList` | XF310の明細 | 同 L178 |

入力部品も、面ごとに型付きのクラスへ分かれています。`DialogCheckLayoutComboBox`（同 L1958）、`TextField`（L2061）、`UrlField`（L2152）、`PromptCallField`（L2189）、`ImageField`（L2247）、`CheckBox`（L2317）、`DateField`（L2335）、`ByteaField`（L2367）、`TextArea`（L2387）、`FYearBox`（L2429）、`MSeqBox`（L2474）、`YMonthBox`（L2513）です。

示しているのは、**表示部品の選択が入力値の意味（年度・月度・日付・画像）と結び付いている**ことです。`FieldOptions`の`WIDTH`・`ROWS`・`CAPTION`・`ALIGNMENT`・`HORIZONTAL`・`VERTICAL`・`PROMPT_CALL`・`COMMENT`を同じダイアログで読みます（同 L1420、L1424、L1575、L1617、L1735、L1761、L1775、L1798、L1877）。

**受け止める形（L5→L2へ返す診断）**

- Studioは、配置の検査を「意味の検査」と分けて持ちます。検査は**面（Pattern種別＋対象）ごと**に行い、結果は診断としてL2の宣言へ返します。
- 項目の**意味の役割**（必須・参照・導出・年度・月度）と、**表示指定**（幅・行数・整列・部品）を別の欄として編集します。X-TEAは同じ`FieldOptions`文字列に同居させていたため、実行先を変えると再設計になりました（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §5の2）。
- 年度・月度は、表示部品ではなく**標準意味（L4の暦）**の値として宣言します。X-TEAの`FYearBox`・`MSeqBox`・`YMonthBox`は、暦の規則を`Session`に持たせ、表示だけを部品にしていました（`D:Session.java L1444–1980`）。

### 3.2 表とモジュールの同期は、検査と操作が一つの画面にある

`DialogCheckTableModule`のボタンは、`モジュール作成`・`モジュール修正`・`定義の補完`・`モジュール削除`です（`E:Res_ja.java`の`ModuleCreate`・`ModuleModify`・`ModulePut`・`ModuleDelete`、`E:DialogCheckTableModule.java L125–141`）。

結果の文言は「定義にもとづいてモジュールが作成されました」「定義とモジュールが同期されました。必要に応じてデータを修正してください」「定義とモジュールは同期しています。再作成の必要はありません」の3種に分かれます（`ModuleCheckMessage43`・`44`・`35`）。外部キー制約がDriverに不要とされる場合の注記もあります（`ModuleCheckMessage45`）。

**受け止める形（L5の操作境界）**

- Studioは、**差分の提示（読み取り）**と**物理スキーマへの適用（書き込み）**を別の操作にします。X-TEAは同じ画面のボタンでしたが、変更の履歴・承認・復旧が弱いままでした（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) A-6）。
- 適用は`plan/`の生成物（DDL・差分）を入力にし、**どの受理版から生成したか**を記録します。X-TEAの「同期しています」は、定義と実DBの比較結果を画面に出すだけで、版を残しませんでした。
- 削除は破壊的操作として分離し、承認と復旧の手順を要求します（A4の判断）。

### 3.3 横断検索の対象種別は列挙されている

`DialogScan`は、検索対象を種別ごとのチェックボックスで選びます。`System`・`Department`・`TaskType`・`TableType`・`DataType`・`FunctionType`・`Terms`・`MaintenanceLog`・`SubjectArea`・`Role`・`Task`・`Subsystem`・`Table`・`Function`です（`M:DialogScan.java L80–94`）。大文字小文字の区別（`CaseSensitive`、同 L78）と、ロール・サブシステムでの絞り込み（同 L96–98）もあります。

**受け止める形（L5の検索契約）**

- Studioの検索は、**Subject種別の列挙**（`inspect`の`subject-kind`と同じ語彙）で対象を選びます。X-TEAの種別名をそのまま語彙にせず、TEIのSubjectに対応させます。
- 置換は、**識別子と表示名を分ける**ことを要求します。X-TEAは同じ操作で両方を置換でき、参照を壊し得ました（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) A-4）。
- 検索結果は診断と同じく版に結び付けます（`worldFingerprint`）。検索した版と異なる版への置換適用は拒否します（AC-9の境界）。

### 3.4 変更の履歴は要素単位で、複数要素にまたがる操作は原子的でない

`Editor`の`UndoManager`は、`actionArray`（`Add`・`Remove`・`Modify`）、対象`node`、所有者`node`、旧DOM、新DOMを並行する配列に積みます（`E:Editor.java L14720–14760`）。`savedDomElement`と`savedOwnerNode`を別に持ち、`indexOfLastUndo`・`indexOfRedoMax`で位置を管理します。

**受け取れること**: 要素単位の追加・削除・変更は正確に戻せます。
**受け取れないこと**: 取込み・表の突合せのような複数要素の操作は、単位がそろわず原子的に戻せません（A-8）。

**受け止める形（L5の編集履歴）**

- Studioの編集履歴は、**1操作＝1つの受理候補**として扱い、複数要素にまたがる操作は「まとまり」として記録します。X-TEAの配列は要素を並べるだけで、まとまりの境界を持っていませんでした。
- 受理（L1の変更）は履歴の再生ではなく、**新しい版の受理**として行います。取り消しは版の巻き戻しではなく、新しい版の提案です。

### 3.5 機能の雛形生成は、8型の分岐で行われる

`DialogAddFunction`の種別選択は`XF000`・`XF100`・`XF110`・`XF200`・`XF290`・`XF300`・`XF310`・`XF390`の8つで（`E:DialogAddFunction.java L170–177`）、種別ごとに生成する要素が分岐します（同 L426–452）。単票系（`XF100`・`XF110`・`XF200`・`XF290`）と見出し明細系（`XF300`・`XF310`・`XF390`）で主表の扱いが変わります。

**受け止める形（L3の生成器）**

- Studioの雛形生成は、**Pattern種別ではなく能力（Capability）と対話契約の組合せ**を入力にします。8型は[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) F-11のとおり、複数性・編集性・出力形態の組合せとして説明できます。
- 生成した雛形は**未受理の提案**として置き、生成根拠（元のData・属性・受理版）を残します。X-TEAは表から推測して雛形を作り、利用者が確認しないまま残り得ました（O-6）。

### 3.6 SQLコンソールは接続を選んで任意SQLを実行する

`DialogSQL`は、接続の選択（`AvailableConnection`）と文の入力欄を持ち、`Statement`・`ResultSet`・`ResultSetMetaData`を使って実行します（`E:DialogSQL.java L107–122`、L43–46）。

**受け止める形（L5の実行境界）**

- Studioは、**受理済みの意味を読む問い合わせ**（read-only）と、**データを変える操作**を分けます。X-TEAのSQLコンソールは両方を同じ面に置き、認可・効果の境界がありませんでした（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) §5の12、S-4）。
- データを変える操作は、Command境界（`Invoker`・`UnitOfWork`・`Authorizer`）を通します。参照実行で確かめ、実適用は別の受理を要します。

### 3.7 標準業務の値は、コードと設定に埋まっている

`Session`は暦・税・通貨・採番を持ちます。

| 機構 | 根拠 | 観測した規則 |
| --- | --- | --- |
| 年度・締め月 | `D:Session.java L1461` | 会計年度の許容月を`ALLOWED_FISCAL_MONTH`で取得し、締め月を`FIRST_MONTH`で持つ |
| 休日・営業日 | 同 L786、L1691、L1882 | 休日を`KBCALENDAR`＋日付の組で持ち、暦種別ごとに判定する |
| 税額 | 同 L2315–2334 | 税率表を開始日で選び、正の金額は切り捨て、負は切り上げる |
| 採番 | 同 L1980 | 伝票番号などを`getNextNumber`で採番する |

**受け止める形（L4の標準意味）**

- 年度・締め月・休日・税率・採番は、案件コードではなく**宣言された意味**として持ちます（D6）。X-TEAは`Session`とシステム変数に埋め、意味として宣言していませんでした。
- 端数処理（切捨て・切上げ・四捨五入）と適用期間（開始日のみ・終了日あり）を、宣言の属性にします。X-TEAは正負で切り捨てと切り上げを切り替え、適用終了日を持ちませんでした（[知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md) S-2）。
- 休日の暦種別は、案件ごとの表ではなく**暦の宣言**として持ち、期間（`Period`）と結びます。

### 3.8 行列出力は5種で、正本ではない

`DialogMatrixList`は、業務フロー＆業務・業務フロー＆機能・業務＆機能・業務＆テーブル・テーブル＆機能の5種の行列を出力します（`M:DialogMatrixList.java L121–134`、`M:Res_ja.java`の`S75`–`S79`）。

**受け止める形（L3の派生出力）**

- 行列は**派生表示**であり、正本ではありません。セルの意味は凡例に依存するため、TEIではセルの種別（作成・参照・更新・削除）を機械可読な列にします（O-3）。
- 出力には**入力版**（受理済みの`worldFingerprint`）を付け、出力から入力版へ戻れるようにします。

## 4. Studioの基本機能としての受容先

上の観測を、Studioが持つべき基本機能へ整理します。

| 基本機能 | 入力 | 出力 | 境界（してはいけないこと） |
| --- | --- | --- | --- |
| 意味の表示 | 受理済みのWorld | ビュー（JSON・HTML・印刷） | 意味を変えない |
| 診断と修正案 | 受理済みのWorldの版 | 診断・版付きの案（未受理） | 案を自動で受理しない |
| 配置の検査 | Patternの対話契約と表示意図 | 面ごとの診断 | 座標を意味にしない |
| 検索と置換 | Subject種別・版 | 結果・置換案 | 別の版へ適用しない |
| 雛形生成 | Data・属性・能力 | 未受理の提案 | 生成根拠を失わない |
| 差分と物理適用 | 受理版・生成物 | DDL・差分・manifest | 生成物を正本にしない |
| 参照実行 | Command・入力・許可 | 結末・効果・差・記録 | 実適用と混同しない |
| 標準業務の照会 | 暦・税・通貨・採番 | 計算結果と根拠 | 案件コードへ埋めない |
| 編集履歴 | 変更操作 | まとまり付きの履歴 | 巻き戻しを受理にしない |

## 5. 実装の順序（案）

[移行計画](../workstreams/XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md)のWPに対応させます。

1. **WP9（診断）**: 配置の検査を面ごとに分ける。診断は版に結び付ける（既存の`validation.Proposal`を再利用）。
2. **WP9（検索）**: Subject種別を`inspect`の語彙に揃え、識別子と表示名を分けた置換を作る。
3. **WP3（対話）**: 項目の意味の役割と表示指定を別の欄にする。年度・月度は標準意味へ送る。
4. **WP6（帳票・派生）**: 行列と設計書の出力へ入力版を付ける。
5. **WP8（標準業務）**: 暦・税・通貨・採番を宣言として持ち、参照実装で照会する。
6. **WP5（実行）**: SQLコンソールを参照実行と実適用に分ける。

各段階で、参照実装の試験と証拠を`plan/evidence/`へ残します。

## 6. 未確認の事項

- `DialogCheckLayout`の全分岐（面ごとの入力部品の選択規則、`PROMPT_CALL`の受け渡し方向）は、全組合せを確認していません。
- `DialogCheckTableModule`の「定義の補完」が実際にどの要素を補うかは、静的な読解の範囲です。
- `Editor`の編集履歴が、取込み・突合せのような複数要素操作でどう振る舞うかは、実行して確認していません。
- 配布JAR（Modeler M50、Editor/Driver M36）と固定ソースの同一性は未確認です。
- 実DBでのSQL・トランザクション・帳票出力、旧Runtimeの実測は[#940](https://github.com/hat47x/tei-platform/issues/940)が担当します。
- Studio（`cmd/tei-studio`）の画面操作・Wails RPCの実HTTP経路は、[起動確認](../evidence/XTEA_LOWCODE_EXAMPLE_STUDIO_SMOKE_2026-11.md)の範囲を超えて確認していません。

## 7. 実装済みの受容との接点

この設計の各項目は、既に参照実装で確かめた受容と接続できます。

| 設計の項目 | 既存の受容 | 接点 |
| --- | --- | --- |
| 診断と修正案（§3.1、§3.3） | `validation.Proposal`・`tei validate -json`・Studioの`proposals` | 案は版付きで未受理。Studioの画面にも「修正案（未受理）」として出る |
| 面ごとの検査（§3.1） | `internal/validation/application_pattern.go`の対話検査 | Pattern種別ごとの分岐を、面の単位へ整理する |
| 参照実行の記録（§3.6） | `Invoker.Recorder`・CLIの`-operation-log`・Studioの`records` | SQLコンソールの実行も同じ記録へ載せる |
| 効果の宣言照合（§3.6） | `Differences`・`AppliedUnobserved` | 参照実行と実適用の差を、同じ形で残す |
| 標準業務の宣言（§3.7） | 期間（`Period`）・値域（`Type.Values`） | 暦・税・通貨・採番を、期間と同じ宣言面へ置く |
| 派生出力の版（§3.8） | `semanticFingerprint`・印刷の受領票 | 行列・設計書の出力へ同じ版を付ける |
| 未受理の生成物（§3.5、§3.3） | `validation.Accept`の版一致要求 | 雛形・置換案も同じ受理境界を通す |

## 8. 関連

- [受容設計](XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)
- [知識目録](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)
- [受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)
- [移行計画](../workstreams/XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md)
- [実装対応](../evidence/XTEA_LOWCODE_ACCEPTANCE_TRACE_2026-11.md)
- [作者支援ツールの移植対応表](AUTHORING_TOOL_FEATURE_TRANSFER_MAP_2026-10.md)
- [X-TEA派生出力の受容設計](XTEA_DERIVED_OUTPUT_RECEPTION_2026-11.md)（設計書・DDL・行列・差分）




