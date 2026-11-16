# X-TEAの項目拡張型を意味別に分離する：分類・投影Referenceの実装と負例（2026-11-16）

- **位置づけ**：**静的なコード読取り**と、対象5語に限定したread-onlyな分類・投影Referenceの実装記録です。Javaの実行、X-TEAの起動、画面操作、実機での取込み・DDL生成はしていません（#3816の「旧実機動作とTEIへの意味受理は別に記録する」に従う）。
- **読んだ版**：Modeler `68f1535b1c5dd80a327dfced37be401ecd3093c9`、Editor `ee19a27609b2749110ec39b62d4540693347fe09`、Driver `6c077773b74c6d7dbcc253835cf8d347e4ffb1ef`。Issueが引く版と一致することを`git rev-parse`で確認した。
- **目的**：#3816の第2項目「KUBUN／VALUES／BOOLEAN、VIRTUAL、NO_EDITの既存扱いを調査し、TEI側の再利用先を特定する」と、その結果を小さく検証する。
- **これは意味受理ではありません。** 分類した結果は候補であり、TEIの標準データ構造へ採用したことを示しません。

## 1. 先に結論

`TypeOptions`／`FieldOptions`は、**値そのもの・表示名・値域・入力補助・更新可否・物理DB型・導出・保存の有無・表示書式**を1つの文字列に混在させています。加えて、旧実装はその同じ文字列を**4通りの異なる規則**で読んでいます。したがって「X-TEAの`TypeOptions`の意味」を1つのparserで再現する行為は、**どの読み方を正本にするかを決めないまま別の解釈を追加する**ことになります。

本Referenceは、その決断を先送りしたまま次の2つだけを行います。

1. 宣言された文字列を**失わずに**保持し、どの軸を語り得るか／語り得ないかを対応付ける。
2. 読めないもの（未知名、曖昧な連結、括弧の崩れ、外部表の中身）を**未解決のまま**残す。

## 2. 旧実装の読み方（4規則）

| 規則 | 実装 | 一致の性質 | 該当 |
| --- | --- | --- | --- |
| **R1 生文字列の部分一致** | `getAttribute("TypeOptions").contains(KW)` | token化しない。`NOT_X`、`X`を内側に含む別指定、前方一致する別語がすべて真になる | `VIRTUAL`: `Editor.java:37118`, `37141`, `DialogImportModel.java:656`, `664`, `DialogImportXEAF.java:750`。`KUBUN(`: `Editor.java:30544`, `38528`, `39343`。`BOOLEAN(`: `Editor.java:30545`, `30727`, `30744`, `38529`, `38682`, `39344`, `39534`, `39551`, `39763` |
| **R2 カンマ分割後の語の完全一致** | `getOptionList(s).contains(KW)` | `NOT_X`、`XFYEAR`、括弧付き形に安全 | `VIRTUAL`: `XF200.java:2329`, `2627`, `XF100.java:2997`, `Session.java:3821`, `3854`ほか約30箇所。`NO_EDIT`: 全30箇所（`XFUtility.java:223`〜`DialogCheckLayout.java:637`） |
| **R3 キーワード前置の値取出し** | `indexOf(KW+"(")==0` → `substring(pos1+len+1, length-1)` | 値を返す唯一の規則。末尾1文字を無条件に落とす | `KUBUN`, `VALUES`, `BOOLEAN`（`XFUtility.java:709-726`, `Editor.java:11642-11658`） |
| **R4 `VALUES(`だけの括弧保護** | 最初の`VALUES(`から`)`までを丸ごと1 tokenとして追加し、除去を試みる | **除去が効かない**（`String.replace`の戻り値を捨てている）。保護は最初の1個だけ | `XFUtility.java:691-699`, `Editor.java:11624-11632` |

R4の実害。`NO_EDIT,VALUES(a,b)`は次になる。

```text
["VALUES(a,b)", "NO_EDIT", "VALUES(a", "b)"]
```

`VALUES`の値は正しく`a,b`と読めるが、**payloadのカンマがtoken列を汚染**するため、`contains("b)")`や`contains("VALUES(a")`が真になる。`KUBUN`と`BOOLEAN`はこの保護を受けないので、`KUBUN(a,b)`は`KUBUN(a`と`b)`に割れ、R3が空文字を返して**「KUBUN未指定」として黙って扱われる**。

## 3. 対象5語の分類

| 指定 | 宣言されていること | 語り得ないこと |
| --- | --- | --- |
| `KUBUN(<表ID>)` | コードの同一性、値域の存在、入力時の補助 | 更新可否、物理DB型、導出、保存の有無 |
| `VALUES(<v1>;<v2>;...)` | 値域、入力時の補助 | **表示名の分離**、更新可否、物理DB型、導出、保存 |
| `BOOLEAN(<真>;<偽>)` | 真偽に対応する保存値、入力時の補助 | 更新可否、物理DB型、導出、保存、表示名 |
| `VIRTUAL` | 導出の存在、保存しないこと、物理列にしないこと | 値域、更新可否、表示書式、入力時の補助 |
| `NO_EDIT` | 表示書式のみ | **更新可否**、値域、物理DB型、導出、保存 |

### 3.1 `KUBUN`の値域はmodel fileに無い

コード表の**名前**は`<System UserVariantsTable="...">`にある（`Session.java:595`）。**中身**は実行時DBの行で、読取りは常にSQLである。

```java
String sql = "select * from " + dialog_.getSession().getTableNameOfUserVariants()
    + " where IDUSERKUBUN = '" + strWrk + "' order by SQLIST";   // XF200.java:3326
```

列は`IDUSERKUBUN`・`KBUSERKUBUN`・`TXUSERKUBUN`・`SQLIST`（`XF200.java:3329-3330`）。したがって`KUBUN`は**値域が宣言されているが中身は未解決**という状態になる。本Referenceはこれを`unresolvedAgainstSourceByKind`へ計上する。

### 3.2 `VALUES`は値と表示名を分離できない

`VALUES`の各literalは**保存値と表示文字列を兼ねる**。`XF200.java:3289-3290`で同じtokenを`keyValueList`と表示項目の両方へ入れている。表示名だけを別に持つ構文が無いため、「値そのものと表示名を区別する」という要求は`VALUES`単独では満たせない。

### 3.3 `BOOLEAN`の1要素形は空と偽を潰す

`XFUtility.java:6327-6335`: token数1なら`valueTrue=token`, `valueFalse=""`。2なら両方設定。3以上は両方空のまま。

```java
if (workTokenizer.countTokens() == 1) { valueTrue = workTokenizer.nextToken(); valueFalse = ""; }
if (workTokenizer.countTokens() == 2) { valueTrue = ...; valueFalse = ...; }
```

1要素形では**未入力と偽値が同じliteral**になる。しかもEditorの書込みは2要素を要求する（`Editor.java:20720`）ため、Driverが受理する1要素形をEditorは作れない。UI既定値は`T;F`（`Editor.java:45619`）。

### 3.4 `NO_EDIT`は更新可否の指定ではない

これが最も誤解されやすい点である。`NO_EDIT`は**全30箇所すべてがR2（語の完全一致）**で、`setEditable`／`setEditMode`の式に一度も現れない。実際に効いているのは表示書式である。

- 先頭ゼロと桁区切りを保つ（`XFUtility.java:223-231`, `1946-1948`）
- 数値セルを文字列セルにする（`XF200.java:2001`, `XF300.java:3095`, `XF390.java:3067`）

更新可否の正本は`NoUpdate`属性である。

```java
if (workElement.getAttribute("NoUpdate").equals("T")) { isNoUpdate = true; }   // XF200.java:2326-2327
if (mode.equals("EDIT") && this.isEditable && !this.isNoUpdate) { ... }        // XF200.java:2861
```

さらに`PROMPT_LIST`／`PROMPT_CALL`は`isNoUpdate`を**falseへ上書きする**（`XF200.java:2360`, `2377`）。よって`NoUpdate="T"`と`PROMPT_LIST`が併存すると後者が勝つ。Editor側のラベルは`{"EditTypeNoEdit","No Edit"}`（`Res.java:289`）だが、これは5つの**書式**選択肢の1つで、選ぶと`ZERO_SUPPRESS`／`HH_MM`／`PERCENT`が落ちる（`Editor.java:20482-20526`）。

**したがって`NO_EDIT`を更新可否の軸へ写すと、旧実装より強い主張になる。** 本Referenceは`NO_EDIT`を表示書式のみに限定し、「更新可否を語らない」ことを明示する。

### 3.5 `VIRTUAL`は物理列を作らないことだけを語る

`AttributeType=DERIVABLE`への写像（`DialogImportXEAF.java:750-753`）、結果セットからの読取り抑止（`XF200.java:2870`）、表複製時の列除外（`Session.java:3821`, `3854`）が確認できる。一方、**導出規則・計算時点・キャッシュの有無は宣言されていない**。

`KUBUN(VIRTUAL)`は実在しうる危険な組合せである。R1の5箇所は真、R2の約30箇所は偽になる。**同じ項目が、あるmoduleでは仮想、別のmoduleでは非仮想**になる。

## 4. 意味が異なる複数指定と曖昧な連結

### 4.1 競合の解決順が画面ごとに違う

`XF200`は`KUBUN`/`VALUES`を`BOOLEAN`より先に検査する（`XF200.java:2399` → `:2408`）。`XF300`は逆で`BOOLEAN`を最初に検査する（`XF300.java:5756` → `:5778` → `:5814`）。`DialogCheckLayout.java`は**同じクラス内**に両方の順序を持つ（`:1555`と`:1804`）。

`XF300`のコメントは経緯と後退を自認している。

```java
// Steps to check BOOLEAN should be here because the field can be specified
// as PROMPT_LIST1/2. This happens because BOOLEAN is placed recently
```

`XF200`の連鎖（`:2336-2472`）にはこの修正が入っていない。**どちらが正しいかを決める根拠がソース内に無い**ため、本Referenceは競合を件数として記録し、勝者を選ばない。

### 4.2 `NO_EDITFYEAR`は分割しない

R2の完全一致なので`NO_EDITFYEAR`は`NO_EDIT`に一致せず、旧Driverも1語として読む。2つの指定のつもりで書かれた可能性はあるが、**分割すると挙動が変わる**。本Referenceは`ambiguous-concatenation-not-split`として件数だけ残す。

## 5. 実装したReference

`internal/xeadsource/fielddesignation.go`に、対象5語だけを扱う分類を追加した。公開面は`tools/reference-xead-source-inspect`の`fieldDesignations`集計で、**件数のみ**を出す。

区別する軸（`SeparationAxis`）は9つ: `code-identity`、`display-name`、`value-domain`、`input-assist`、`updatability`、`physical-db-type`、`derivation`、`persistence`、`display-formatting`。

各tokenは`informs`（語り得る軸）と`doesNotInform`（語ってはいけない軸）の**両方**を持つ。片方だけだと「その軸について何も言っていない」と「その軸について誤読されている」を区別できないためである。

公開reportは次を出さない: 宣言文字列、コード表名、保存literal、旧項目ID、source locator。`TestBuildReportOmitsFieldDesignationDeclarations`で3種のsecretがreportのJSONに現れないことを固定した。

### 5.1 作らなかったもの

- **巨大な旧TypeOptions Parser**：R1〜R4が併存する以上、1つのparserは「X-TEAの意味」を名乗れない。
- **正式な新型Registry**：5語の分類は新しい型体系の根拠にならない。
- **旧Driver完全互換Engine**：画面ごとに順序が異なる挙動を再現することは、意味の確定ではない。

`tokenizationPolicyRevision = "tei.xeadsource.field-designation-tokenization/v1"`として分割規則を記録した。規則を変えるときはこの値を変える。

## 6. 負例の検証

`internal/xeadsource/fielddesignation_test.go`に、実装から独立したExpectedを書いた。

| 負例 | 期待 | 確認していること |
| --- | --- | --- |
| `VALUES(001;002;010)` | payloadが`001;002;010`のまま | 旧Driverの`getStringNumber`は`[-0-9.]`以外を捨て、`PERCENT`は`Float.parseFloat`で100除算する。数値として読むと先頭ゼロが消える。**宣言textを保持**する |
| `SOMETHING_NEW(X)`, `AUTO_NUMBER(4)` | 両方`unknown`、payload無し | 対象外の語に意味を与えない |
| `NO_EDITFYEAR` | 1 token、`ambiguous-concatenation-not-split` | 分割しない。`unrecognized-designation-token`とは区別する |
| `KUBUN(KBFLAG1),VALUES(T;F)` | 2 token保持、`value-domain`競合を報告、勝者なし | 黙って片方を捨てない |
| `KUBUN(a,b)` | 2 tokenとも未解決、`unbalanced-parenthesis-after-comma-split` | カンマで割れた半端を指定として受理しない |
| `virtual,no_edit,kubun(x)` | 全て未解決 | 旧実装の一致は大文字小文字を区別する |
| `VIRTUAL,VIRTUAL(X),NO_EDIT,NO_EDIT(X)` | `[virtual, unknown, no-edit, unknown]` | 括弧付き形は裸のflagではない |
| `f:TypeOptions="VIRTUAL,NO_EDIT"` | fact 0件 | 外部名前空間の同名属性を読まない |
| `TypeOptions=""` | `EmptyDeclared=true`、token 0件 | 明示的な空と属性不在を区別する |
| `BOOLEAN(T)` | `boolean-single-part-collapses-empty-and-false` | 1要素形で空と偽が潰れることを記録する |
| `KUBUN(...)`を含むfield | `unresolvedAgainstSource`に計上 | 値域の存在と中身の既知を区別する |

実行結果は`go test -count=1 ./internal/xeadsource/`と`./tools/reference-xead-source-inspect/`で確認した（結果は§8）。

## 7. 未対応・未確認

- **対象外の語**：`AUTO_NUMBER`、`YMONTH`、`FYEAR`、`MSEQ`、`HH_MM`、`PERCENT`、`ZERO_SUPPRESS`、`ACCEPT_MINUS`、`KANJI`、`KATAKANA`、`ZIPNO`、`ZIPADRS`、`DIAL`、`URL`、`IMAGE`、`BARCODE_TYPE`、`SCAN`、`ADATE`。いずれも未分類のまま件数だけ残す。
- **`FieldOptions`の表示・操作系**：`WIDTH`、`ROWS`、`CAPTION`、`ALIGNMENT`、`PROMPT_CALL*`、`LINKED_CALL*`、`COMMENT`。本Referenceでは**意図的に未解決**にした（値の宣言ではないため）。ただし`PROMPT_LIST`／`PROMPT_CALL`が`isNoUpdate`を上書きすることは§3.4の通り確認しており、更新可否の軸に影響する。**この上書きは未実装**。
- **`NoUpdate`属性**：更新可否の実正本だが、本Referenceは読んでいない。`NO_EDIT`を更新可否へ写さない判断の根拠として読んだだけである。
- **`ADATE`**：Editorに説明があるが、照合したDriverに同名の分岐を特定していない。
- **実機の動作**：Java実行、X-TEA起動、画面操作、取込み、DDL生成、帳票出力はすべて**未実施**。すべてソース読取りに基づく。
- **model instance**：`**/*.xead`が見つからず、実データでの`TypeOptions`分布は測っていない。§3.3の`T;F`はUI既定値とseed SQLのliteralに基づく。
- **`PERCENT`の尺度**：`XF100.java:1854-1860`で`Float.parseFloat(wrk)/100`と`0%`書式を確認したが、「内部値が比率か百分率か」を型の層で決める作業は#3816の第3項目以降。
- **意味の受理**：TEIの標準データ構造へ採用していない。新しい型Registryも固定Enumも追加していない。

## 8. 検証結果

```text
go test -count=1 ./internal/xeadsource/                  ok
go test -count=1 ./tools/reference-xead-source-inspect/   ok
```

`go vet`と`gofmt`は対象2 packageでクリーン。Repository Healthの結果はPR本文へ記録する。

## 9. 再現手順

```bash
MODELER="/path/to/X-TEA/ソース/XeadModeler"
EDITOR="/path/to/X-TEA/ソース/XeadEditor"
DRIVER="/path/to/X-TEA/ソース/XeadDriver"

git -C "$MODELER" rev-parse HEAD   # 68f1535b1c5dd80a327dfced37be401ecd3093c9
git -C "$EDITOR"  rev-parse HEAD   # ee19a27609b2749110ec39b62d4540693347fe09
git -C "$DRIVER"  rev-parse HEAD   # 6c077773b74c6d7dbcc253835cf8d347e4ffb1ef

# R3/R4: 値取出しとVALUES保護の欠陥（両実装で同一）
sed -n '688,726p' "$DRIVER/src/xeadDriver/XFUtility.java"
sed -n '11621,11658p' "$EDITOR/src/xeadEditor/Editor.java"

# R1: VIRTUALの生部分一致（5箇所）
grep -n 'contains("VIRTUAL")' "$EDITOR/src/xeadEditor/Editor.java"
sed -n '748,755p' "$MODELER/src/xeadModeler/DialogImportXEAF.java"

# NO_EDITが更新可否で使われないこと（0件が期待値）
grep -n 'NO_EDIT' "$DRIVER/src/xeadDriver/XF200.java" | grep -i 'setEditable\|setEditMode'

# 更新可否の正本
sed -n '2320,2335p' "$DRIVER/src/xeadDriver/XF200.java"
sed -n '2850,2870p' "$DRIVER/src/xeadDriver/XF200.java"

# 画面ごとの順序差
sed -n '2395,2412p' "$DRIVER/src/xeadDriver/XF200.java"
sed -n '5750,5830p' "$DRIVER/src/xeadDriver/XF300.java"

# 先頭ゼロを壊す経路
sed -n '179,210p' "$DRIVER/src/xeadDriver/XFUtility.java"
sed -n '1845,1866p' "$DRIVER/src/xeadDriver/XF100.java"
```

注：これらの`.java`はShift_JISなので、UTF-8前提のviewerでは文字化けする。行番号は原fileに対して確認済み。

## 関連

- #3816（項目拡張型）、#2495（Data Facet）、#2544（Facet演算・矛盾判定）、#2620（意味役割）、#2497（写像・逆写像）
- 既存記録：[`XTEA_TYPE_MAPPING_STATIC_READING_2026-11-10.md`](XTEA_TYPE_MAPPING_STATIC_READING_2026-11-10.md)（F-1〜F-7。本記録は同じ版に対して、5語の**語彙単位の読み方**と軸の分離を補う）
- 既存記録：[`TEI_TYPE_FACET_UNIT_AXIS_GAP_2026-11-10.md`](TEI_TYPE_FACET_UNIT_AXIS_GAP_2026-11-10.md)（TEIの型に値域・単位・尺度を書く場所が無いことの実測）
- 研究棚卸し：[`XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md`](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)（I-2・I-18）
