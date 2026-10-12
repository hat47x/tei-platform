# M0.30 高度な索引の期待構造を記述する際の準拠確認

## 目的

M0.30では、M0.29で取得・比較できるようになった高度な索引の構造を、データベースが管理する保存先の定義から、期待する物理構造として明示できるようにします。

対象はIssue #51とADR-0011です。索引を推奨・自動生成するのではなく、利用者が選んだ物理構造上の意図を正確に記述し、実際の状態との差を既存の構造差分の仕組みで比較することが目的です。

## 記述方法の位置づけ

高度な索引は、業務上の意味を表すモデルの一部ではありません。

業務上の意味から特定の索引が必須だと自動的に決めず、データベースが管理する保存先への対応付けに記述します。これにより、同じ意味上のデータを、異なるデータベースや物理配置へ展開できます。

従来の簡潔な記述形式は維持します。

```tei
index customers_external_id_idx on customer.external_id
```

複数のキーや提供元固有の物理構造が必要な場合に限り、ブロック形式で記述します。

```tei
index customers_name_advanced_idx {
    method btree

    key attribute customer.name {
        collation "pg_catalog.C"
        operator-class "pg_catalog.varchar_pattern_ops"
        order desc
        nulls last
    }

    include customer.external_id

    predicate {
        provider postgres
        dialect postgresql
        source "profile IS NOT NULL"
    }
}
```

## 順序付きのキー

高度な索引では、キーの順序を保持します。

キーの記述方法は、次の二つです。

- `key attribute ...`
- `key expression { ... }`

属性を使うキーは、保存先への対応付けにある列の対応情報を通じて、物理列を特定します。定義に物理列名を重複して記述する必要はありません。

式を使うキーは、提供元固有の式として扱い、その内部の意味を中核部分では解釈しません。

```tei
key expression {
    provider postgres
    dialect postgresql
    source "lower(name)"
}
```

属性と式の両方を指定した内部モデルは、期待する構造を生成する段階で拒否します。

## キーごとの意味

キーごとのブロックでは、必要に応じて次を指定できます。

- `collation`
- `operator-class`
- `order default | asc | desc`
- `nulls default | first | last`

並べ替えやNULLの順序を指定しなければ、物理構造では`default`として扱います。

照合順序と演算子クラスは、提供元固有の識別子です。中核部分で共通の分類体系に変換することはしません。

### PostgreSQL B-treeの既定値

M0.29で定めたPostgreSQLの実状態の表記とそろえるため、B-treeについてデータベースへ接続しなくても決まる既定の動作は、期待する構造を生成するときにも表記を統一します。

| 定義に記述した意味 | 期待する物理構造での表現 |
| --- | --- |
| `ASC NULLS LAST` | `order=default`, `nulls=default` |
| `DESC NULLS FIRST` | `order=desc`, `nulls=default` |
| `ASC NULLS FIRST` | `order=default`, `nulls=first` |
| `DESC NULLS LAST` | `order=desc`, `nulls=last` |

B-treeのキーに照合順序として`pg_catalog.default`を明示した場合も、実際の状態と同じ空の表現にそろえます。

これにより、既定の動作を明示しただけの索引を、構造が異なると誤判定することを防ぎます。

一方、既定の演算子クラスや、提供元・索引方式・物理型・提供元の版に依存する識別子の一般的な表記統一は、M0.30では推測しません。これらはIssue #58で扱います。

## 付加列（INCLUDE）

追加の格納列も、意味上の属性識別子で指定します。

```tei
include customer.external_id
```

期待する構造を生成する際に物理列へ対応付けます。追加の格納列の順序も保持します。

## 部分索引条件と式の由来

部分索引の条件と式を使うキーには、必ず次の出所情報を持たせます。

- Provider
- Dialect
- Source

対象のデータベースの提供元と式に指定した提供元が一致しない場合、期待する構造の生成を拒否します。

提供元固有のSQL式を解析することは、TEIの中核部分の構文解析器の責務にしません。PostgreSQLとOracleの式を、文字列が同じという理由だけで同じ意味とは判断しません。

データベース管理システムが型変換や括弧を補い、異なる文字列を返す場合の表記統一は、Issue #54で別途扱います。

## 一意索引におけるNULLの扱い

一意索引では、NULLの扱いを次のように明示できます。

```tei
null-treatment distinct
null-treatment not-distinct
```

指定しなければ、期待する構造を生成するときに、提供元で実際に適用される既定値を用います。

一意でない索引に`null-treatment`を指定した場合は拒否します。

## 索引方式

`method`は省略できます。

指定しなければ、期待する構造を生成するときに提供元の既定方式を使います。PostgreSQLでは、この準拠確認の範囲で`btree`を採用します。

## 索引名と入力元の識別

一つの保存先への対応付けの中では、索引名を重複させません。

入力元索引では、ファイルのパスに依存しない次の識別情報を使うためです。

```text
Kind = persistence-index
ID   = <Persistence Mapping ID>/<Index Name>
```

同じ対応付けの中に同名の索引を複数許すと、異なる宣言が同じ識別情報を持ち、入力元の案内が誤った位置を指すおそれがあります。そのため、期待する構造の生成と入力元索引の構築の両方で、重複した索引名を拒否します。

## 入力元索引と診断への投影

例えば、次の識別情報を構築します。

```text
persistence-index:sales_core.customer/customers_name_advanced_idx
```

入力元索引が示す範囲は、定義ファイル全体や対応付けの先頭ではなく、索引名そのものの位置です。

この識別情報にはファイルのパスを含めません。ファイルを移動しても索引自体の識別情報は変わらず、入力元索引が指す保存場所だけが変わります。

簡潔な形式で記述した索引も、同じ`persistence-index`の識別情報で追跡します。

さらに、`validation.Diagnostic`がこの識別情報を持つ場合、診断結果の表示から索引名の定義位置へ戻れることを検証します。入力元索引を生成するだけでなく、編集画面や診断表示から正確な場所へたどれることも確認します。

## 期待する構造の生成

保存先の定義から、M0.29で扱う`physical.Index`へ次の情報を生成します。

- ordered Keys
- IncludedColumns
- Predicate
- Unique
- UniqueNullTreatment
- Method
- keyごとのCollation / OperatorClass / Order / NullsOrder

期待する構造の`Index.State`は常にnilです。

実際の索引の有効性・準備状態・存続状態に関する情報を、期待構造の記述から指定することはできません。

## 構造差分

M0.30では、構造差分の比較規則そのものは追加しません。

M0.29で導入した高度な索引の構造比較を、期待する定義から生成した物理構造にもそのまま適用します。

準拠確認のテストでは、次の条件を確かめます。

1. 高度な索引を含む保存先の定義から、期待する物理構造を生成できる
2. 同じ構造を持ち利用可能な実際の索引とは、差分が0件になる
3. 追加の格納列などの構造を変えると、`drop-index`と`add-index`になる
4. 実際の索引の利用状態を、期待する構造の同一性に混ぜない
5. PostgreSQLのB-treeで既定の動作を明示しても、実状態と同じ表現になる

## 構文解析の準拠確認

構文解析では、次の条件を確かめます。

- 従来の簡潔な形式と高度な索引のブロック形式を、同じ保存先の定義に併用できる
- 単独の高度な索引の解析と、保存先の定義に含まれる索引の解析に、同じ本体解析器を使う
- キーのない高度な索引を拒否する
- 未知の提供元を拒否する
- 提供元・SQLの方言・元の式の情報不足を拒否する
- 未知の並べ替え順序やNULLの順序を拒否する
- 一意でない索引に指定された`null-treatment`を拒否する
- `index NAME`の後が`on`でも`{`でもない場合を拒否する

## 作業領域での準拠確認

実際の作業領域の読み込み経路で、高度な索引を含む保存先の定義ファイルを読み取れることを確認します。

同じ試験用データから入力元索引を構築し、簡潔な形式と高度な形式のどちらの索引宣言も、正しい索引名の位置へ対応付けられることを検証します。

加えて、次を確認します。

- `persistence-index`の識別情報を持つ診断結果が、索引名の位置へ対応付けられる
- 同じ対応付け内で索引名が重複すると、入力元索引の構築時にも拒否される

## 参照用作業領域

参照用の作業領域にある従来の単純な索引は変更しません。

従来の定義は、後方互換性を確認する試験用データとして役立つためです。高度な索引のブロック形式は、専用の準拠確認で読み込みから入力元索引、診断表示までを通して検証します。

参照用の作業領域へ高度な索引を追加するためだけに、全体モデルの記録を丸ごと書き換えることは、M0.30の完了条件に含めません。

## マイグレーションSQLとの境界

M0.30で扱うのは、期待する物理構造を定義できることまでです。

高度な索引を安全な提供元固有のDDLへ変換することは、Issue #55で別途扱います。

この段階の移行用SQLの生成機構は、対応していない高度な索引を単純な索引へ暗黙に置き換えず、未対応として明示的に拒否します。

## 非目標

- Index recommendation
- query planからの自動authoring
- provider expression normalizationの一般解（#54）
- provider default / identifier normalizationの一般解（#58）
- Advanced Index migration SQL生成（#55）
- automatic migration apply
- concurrent build policy
- partitioned index semantics
- Desired lifecycle state

## 次に検討する境界

M0.30の完了後も、高度な索引については独立した課題が残ります。

- Issue #54: Desired / Actual間のProviderExpression normalization contract
- Issue #55: provider-owned Advanced Index migration SQL rendering
- Issue #58: expression以外のprovider default / identifier normalization contract

これらを分けて扱うことで、記述構文の都合だけでデータベース固有の式の表記統一、識別子に関する知識、DDLの運用判断を中核部分へ取り込むことを防ぎます。
