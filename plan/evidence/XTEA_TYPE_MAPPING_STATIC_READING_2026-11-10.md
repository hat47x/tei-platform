# X-TEAの型指定とTypeOptionsの読取り：実装を読んだ記録（2026-11-10）

- **位置づけ**：**静的なコード読取り**の記録です。Javaの実行・X-TEAの起動・画面操作はしていません。旧実機の動作確認ではありません（#3816の「旧実機動作とTEIへの意味受理は別に記録する」に従う）。
- **読んだ版**（issue が引く版と一致することを確認済み）
  - Modeler `68f1535b1c5dd80a327dfced37be401ecd3093c9`（`2021.06.20`）
  - Editor `ee19a27609b2749110ec39b62d4540693347fe09`（`2021.06.20`）
  - Driver `6c077773b74c6d7dbcc253835cf8d347e4ffb1ef`
- **目的**：#3816の第1項目「Modelerの`DataTypeID`、`SQLExpression`、Editorの`Type`／`Size`／`Decimal`、拡張型の元の位置を対応付ける。取込みと既存項目の更新、未知の指定、複数指定を区別する」に対する材料。
- **結論（先に）**：型の同一性・スケール・基本型・オプションの解釈が、**同じ名前を持つ複数の実装で別々に決まっています**。TEIが写像を持つなら、どこを正本にするかを決める必要があります。現状のX-TEAをそのまま写すと、同じ情報が経路によって違う値になります。

## 1. 型まわりの要素と属性

### `DataType`（型の定義）

| 属性 | 意味（コードから読めること） | 根拠 |
| --- | --- | --- |
| `ID` | 型の識別子（連番） | `M:DialogImportXEAF.java L1273–1281` |
| `SortKey` | `型名_長さ` | `M:DialogImportXEAF.java L1266`、`L1282` |
| `Name` | 同じく`型名_長さ` | `M:DialogImportXEAF.java L1283` |
| `BasicType` | `String`・`SignedNumber`・`Date`・`Object` の4語 | `M:DialogImportXEAF.java L1284–1298` |
| `Length` / `Decimal` | 長さと小数桁 | `M:DialogImportXEAF.java L1299–1300` |
| `SQLExpression` | DDLに出す型表記 | `M:DialogImportXEAF.java L1301–1305` |

### `TableField`（項目）

`DataTypeID`（型への参照）・`AttributeType`（`NATIVE`／`DERIVABLE`）・`NotNull`・`NoUpdate`・`Default`・`TypeOptions`・`Alias`・`SortKey`・`Descriptions` を持ちます（`M:DialogImportXEAF.java L745–781`）。

`TypeOptions` は**カンマ区切りのトークン列**で、裸の語（`VIRTUAL`・`NO_EDIT`・`ZERO_SUPPRESS`）と、括弧付きの語（`AUTO_NUMBER(...)`・`KUBUN(...)`・`BOOLEAN(...)`・`VALUES(a;b;c)`）が混在します（`E:Editor.java L20463, L20496, L20506, L20670, L20697, L20725, L20756`。`VALUES` の内側だけ`;`区切り）。

## 2. 読取って分かった破綻

### F-1 型の同一性に小数桁が入らない（`SortKey` は型名＋長さだけ）

```java
// M:DialogImportXEAF.java L1266
if (elementInto.getAttribute("SortKey").equals(dataType + "_" + length)) {
```

`DECIMAL(10,2)` と `DECIMAL(10,4)` は同じ `SortKey = "DECIMAL_10"` になるため、**尺度の違う2つが1つの型に統合されます**。2つ目を取り込むと既存の型が再利用され、その`Decimal`（先に入った値）がそのまま使われます。新規に作る側も`SortKey`を同じ規則で書きます（`L1282`）。

これは#3816の完了条件にある「百分率の尺度」を、**型の層で既に失っている**ことを意味します。

### F-2 Editorは`Decimal`属性を読まず、`SQLExpression`の文字列から推測する

```java
// E:DialogImportModel.java L1289–1327（要点）
String dataDecimal = "";                                  // L1293
wrkStr = element.getAttribute("SQLExpression").toUpperCase();  // L1300
dataSize = element.getAttribute("Length");                     // L1301
pos1 = wrkStr.indexOf("(", 0);
if (pos1 > 0) { … "型(長さ.小数)" を分解 … } else { dataType = wrkStr; }  // L1302–1320
attrMap.put("Decimal", dataDecimal);                      // L1326
```

`dataDecimal` は**括弧の中の `長さ.小数` からしか設定されません**。一方、取込み側は`SQLExpression`に括弧を付けるのは`CHAR`を含む型だけです。

```java
// M:DialogImportXEAF.java L1301–1305
if (dataType.contains("CHAR")) {
    newElement.setAttribute("SQLExpression", dataType + "(" + length + ")");
} else {
    newElement.setAttribute("SQLExpression", dataType);
}
```

したがって**取込みで作られた`DECIMAL`は、`Decimal`属性に値が入っていても、Editorから見ると`Decimal`は空**になります。同じ`Decimal`に「属性」と「`SQLExpression`の文字列」という2つの源があり、片方しか読まれません。

### F-3 基本型の導出が2箇所で別規則・別語彙

| 実装 | 語彙 | 一致のしかた | 結果の例 |
| --- | --- | --- | --- |
| `M:DialogImportXEAF.java L1284–1298`（取込み） | `String`／`SignedNumber`／`Date`／`Object` | **部分一致**（`contains("INT")` など）と一部`equals` | `FLOAT`→`String`、`TIME`→`String`、`TIMESTAMP`→`String`、`BIGINT`→`SignedNumber` |
| `D:XFUtility.java getBasicTypeOf`（実行時） | `INTEGER`／`FLOAT`／`STRING`／`BINARY`／`CLOB`／`BYTEA`／`DATE`／`TIME`／`DATETIME` | **完全一致**（`equals`）と`startsWith` | `FLOAT`→`FLOAT`、`TIMESTAMP*`→`DATETIME`、`DECIMAL`→`FLOAT` |

**同じ「基本型」という名前で、語彙も判定も違います。** 取込み時の `SignedNumber` は実行時のどの語にも対応しません。また`DECIMAL`が`FLOAT`に落ちるため、実行時には小数型と浮動小数型が区別されません。

### F-4 `TypeOptions` のトークン分解が、`VALUES` を除去できていない

```java
// E:Editor.java L11621–11632
int index = options.indexOf("VALUES(");
if (index > -1) {
    int index2 = options.indexOf(")", index);
    if (index2 > -1) {
        String wrkStr = options.substring(index, index2+1);
        typeOptionList.add(wrkStr);
        options.replace(wrkStr+",", "").replace(wrkStr, "");   // ← 戻り値を捨てている
    }
}
StringTokenizer workTokenizer = new StringTokenizer(options, ",");
```

`String.replace`は**新しい文字列を返すだけ**で`options`は変わりません。したがって`VALUES(...)`は手動で1回追加されたうえ、続くカンマ分解でもう一度取り出されます。

- 結果として`VALUES`トークンが**二重に登録**されます。読出しは最初の一致で止まるので値は正しく見えますが、`removeOptionValueWithKeyword("VALUES")` は1つしか消さないため**削除が不完全**になります（`E:Editor.java L11660–11682`）。
- `VALUES`の内側を`;`区切りにしているのは外側のカンマと衝突させないためですが、利用者が値をカンマで書くと分解が壊れます。この防御は上記の除去が効いていない以上、**内側にカンマが無いこと**に依存しています。

### F-5 同じ`TypeOptions`を3通りの規則で読んでいる

「`VIRTUAL`かどうか」ひとつを取っても、読み方が3つあります。しかも**Editorの中に2つが同居**しています。

| 読み方 | 例 | 一致の性質 |
| --- | --- | --- |
| 生の文字列への部分一致 | `M:DialogImportXEAF.java L750`<br>`E:DialogImportModel.java L656, L664`<br>`E:DialogCheckTableModule.java L287, L527` | `getAttribute("TypeOptions").contains("VIRTUAL")`。`NOT_VIRTUAL` でも `KUBUN(VIRTUAL)` でも真になる |
| トークン列の完全一致 | `E:DialogAssistList.java L416–417`<br>`E:DialogCheckTableModule.java L1744`<br>`E:DialogEditTableKeyFields.java L271`<br>`E:DialogScanField.java L385` | `getOptionList(...).contains("VIRTUAL")`。カンマ分解後の**語の完全一致**（`ArrayList.contains`は`equals`） |
| 括弧付き語の値の取出し | `E:Editor.java L11642–11658` | `indexOf(keyword + "(") == 0`。`VIRTUAL`のような**括弧なしの語は、この関数では取り出せない**（常に空文字が返る） |

帰結として、`TypeOptions`に`KUBUN(VIRTUAL)`のようにVIRTUALを含む別の値が入っていると、部分一致の箇所だけが「仮想項目」と判断します。取込みと既存項目の更新を比べる`DialogImportModel.java L656`も部分一致側なので、**更新の有無の判定が値の中身に左右されます**。

### F-6 未知の型指定が、正常値と区別できない

```java
// E:DialogImportModel.java L1291–1293（初期値）
String dataType = "";
String dataSize = "0";
String dataDecimal = "";
```

`DataTypeID` がどの`DataType`にも一致しない場合、`break`に到達しないまま**`Type=""`・`Size="0"`**が返ります（`L1297–1323`）。これは「未知の型」と「サイズ0の型」を区別できない形です。また一致探索は最初の一致で`break`するため（`L1321`）、**同じIDが複数ある場合は先勝ちで、重複は報告されません**。

issue の「未知の指定、複数指定を区別する」に対する現状の答えは、**どちらも区別していない**です。

### F-7 設計書のSQL型表記に精度が出ない

`M:DialogDocuments.java L2404` は型の列を`Name + " " + SQLExpression`で作ります。`Name`は`型名_長さ`（`L1283`）、`SQLExpression`は`CHAR`系以外では型名のみ（`L1304`）なので、`DECIMAL`の列は `DECIMAL_10 DECIMAL` のように出て、**小数桁が表記から読み取れません**。

## 3. issue の完了条件（第1項目）との対応

| 求めること | X-TEAの現状 | 根拠 |
| --- | --- | --- |
| `DataTypeID`の対応付け | IDは連番で、意味を持たない。`SortKey`は`型名_長さ` | F-1 |
| `SQLExpression`の位置づけ | DDL表記であると同時に、Editorの`Size`/`Decimal`の**唯一の源** | F-2 |
| Editorの`Type`／`Size`／`Decimal` | `Decimal`は属性を読まず、括弧表記から推測する | F-2 |
| 拡張型の元の位置 | `TypeOptions`の文字列に、型・値域・表示部品が同居する（既存の研究記録 I-2） | `plan/research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md` |
| 取込みと既存項目の更新の区別 | 同一`SortKey`なら既存型を再利用し、新規か既存かを呼出し元へ返さない | F-1 |
| 未知の指定の区別 | `Type=""`・`Size="0"`で、正常値と区別できない | F-6 |
| 複数指定の区別 | 最初の一致で打切り、重複を報告しない | F-6 |

## 4. TEI側の現状との突き合わせ

TEIの型は基底型と列挙値だけを持ちます（[型の値ファセットの実測](TEI_TYPE_FACET_UNIT_AXIS_GAP_2026-11-10.md)）。X-TEA側は`Length`・`Decimal`・`BasicType`・`SQLExpression`・`TypeOptions`を持ちますが、**同じ情報の源が複数あり、経路によって値が変わります**。

TEIが写像を持つとすると、次の3つを決める必要があります。

1. **スケールの正本**：`Decimal`属性か、`SQLExpression`の表記か（F-2）。TEIには今どちらも無いので、写像で新たに持つことになります。
2. **基本型の正本**：取込み時の4語か、実行時の9語か（F-3）。TEIの`std`型は12語で、どちらとも一致しません。
3. **オプションの扱い**：`TypeOptions`を型の一部として扱うか、表示・入力の指定として分けるか（F-4・F-5）。TEIには値域を表す場所が無いため、`KUBUN`・`VALUES`・`BOOLEAN`は「型の制約」として受けるか「表示部品」として捨てるかの判断が要ります。

## 5. この記録が言わないこと / 未実施

- **Javaの実行もX-TEAの起動もしていません。** 上記は全てソースの読取りです。実際の画面・取込み・DDL生成の挙動は未確認です。
- `KUBUN`／`BARCODE_TYPE`／`ZIPNO`などの**値の意味**（コード表の中身、帳票での出力形式）は確認していません。第2項目以降の範囲です。
- 受注・設備計測・公開資料の三用途での照合（第4項目）、語彙・コード表の公開条件（第5項目）は未着手です。
- TEIへ写像を実装していません。新しい型Registryや固定Enumの追加も判断していません。

## 6. 再現手順

```bash
MODELER=/path/to/X-TEA/ソース/XeadModeler
EDITOR=/path/to/X-TEA/ソース/XeadEditor
DRIVER=/path/to/X-TEA/ソース/XeadDriver

# 版の確認（issue が引く版と一致すること）
git -C "$MODELER" rev-parse HEAD   # 68f1535b1c5dd80a327dfced37be401ecd3093c9
git -C "$EDITOR"  rev-parse HEAD   # ee19a27609b2749110ec39b62d4540693347fe09

# F-1 / F-6: 型の同一性と未知ID
sed -n '1256,1310p' "$MODELER/src/xeadModeler/DialogImportXEAF.java"
sed -n '1289,1328p' "$EDITOR/src/xeadEditor/DialogImportModel.java"

# F-3: 基本型の2つの導出
sed -n '1284,1298p' "$MODELER/src/xeadModeler/DialogImportXEAF.java"
grep -n -A 40 'static String getBasicTypeOf' "$DRIVER/src/xeadDriver/XFUtility.java"

# F-4 / F-5: TypeOptionsの分解と判定
sed -n '11621,11658p' "$EDITOR/src/xeadEditor/Editor.java"
sed -n '745,755p'   "$MODELER/src/xeadModeler/DialogImportXEAF.java"
```

## 関連

- #3816（項目拡張型）、#2495（Data Facet）、#2497（Projection・変換・逆写像）、#2620（意味役割）、#3278（実装義務）
- 既存の研究記録: [`XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md`](../research/XTEA_LOWCODE_KNOWLEDGE_INVENTORY_2026-11.md)（I-2・I-14。本記録はそこが扱っていない関数単位の破綻を補う）
- 設計側: [`XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md`](../design/XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md)
- 型ファセットの実測: [`TEI_TYPE_FACET_UNIT_AXIS_GAP_2026-11-10.md`](TEI_TYPE_FACET_UNIT_AXIS_GAP_2026-11-10.md)
