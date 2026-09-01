# M0.31 Schema Diff Unresolved Comparison Conformance

## 目的

M0.31では、Schema Diffが「構造差を確認できた状態」と「比較に必要な正規化情報が不足している状態」を区別できるようにします。

Issue #60を対象とし、ProviderExpressionの表記差を主な最初の事例とします。

## 背景

Advanced Indexでは、expression keyとpartial predicateを`ProviderExpression`として保持しています。Desired側はauthoring textを、PostgreSQL Actual側はcatalogからdecompileされたtextを持つため、同じprovider / dialectであっても、同じ意味が常に同じ文字列になるとは限りません。

この状態で文字列差を即座に`drop-index` + `add-index`へ変換すると、意味上は同じIndexを不要に再作成する可能性があります。

一方、表記が違うだけで常に同じとみなすこともできません。信頼できるprovider normalizationがまだない場合は、「同じ」「違う」のどちらにも確定しない状態として返します。

## Diffの二つの出力

`schemadiff.Diff`は次を分離します。

```text
Diff
  Database
  Changes[]
  Unresolved[]
```

`Changes`は構造差またはActual lifecycle state driftとして確定した項目です。

`Unresolved`は、比較に追加のprovider normalization等が必要で、現時点では構造差を安全に断定できない項目です。

`Unresolved`を`ChangeKind`には含めません。比較不能はmigration操作ではないためです。

## ProviderExpressionの初期規則

同じ名前のIndexを比較するとき、ProviderExpressionについて次を適用します。

1. 両方nilなら一致
2. 一方だけnilなら構造差
3. providerまたはdialectが異なれば構造差
4. provider / dialectが同じでsource textも一致すれば一致
5. provider / dialectが同じでsource textだけが異なれば比較未確定

5について、CoreはSQL式の意味同一性を推測しません。

## Index全体の判定

ProviderExpressionに比較未確定があっても、INCLUDE、key数、column、collation、operator class、sort、NULL ordering、unique semantics、method等に明確な構造差がある場合は、Index全体として構造差を確定できます。

その場合は従来どおり`drop-index` + `add-index`を返し、同じIndexについてProviderExpressionの`Unresolved`を重ねません。

明確な構造差がなく、ProviderExpressionだけが比較未確定の場合は`drop-index` / `add-index`を返さず、`Unresolved`だけを返します。

## lifecycle state

Actual Indexの`valid / ready / live`は比較未確定ではなく観測事実です。

したがって、構造比較が未確定であってもActual Indexがusableでなければ、`index-state-drift`は従来どおりChangeとして返せます。

## FieldPath

初期実装では、比較未確定箇所を次のようなfield pathで示します。

```text
index.keys[0].expression
index.predicate
```

配列indexはPhysical Schemaの0始まり位置です。

## migrationとの境界

`migration.Render`は`Diff.Unresolved`が1件でも残っているDiffからSQLを生成しません。

確定したChangeだけを部分的に実行することもM0.31では行いません。ひとつのDiffに未確定比較が残る状態でmigration planを作ると、適用後の収束状態をTEIが説明できなくなるためです。

将来、object単位で安全に分離して部分適用する必要が生じた場合は、別途設計します。

## 非目標

- ProviderExpressionの意味正規化そのもの（Issue #54）
- provider offline policyの実装（Issue #59）
- collation / operator class等の一般正規化（Issue #58）
- Advanced Index DDL生成（Issue #55）
- SQL ASTによる式同値性判定

## 完了条件

- ProviderExpressionの同一provider/dialect・異なるsourceが`Unresolved`になる
- そのケースで`drop-index` + `add-index`が発生しない
- provider/dialect差やnil有無は従来どおり構造差になる
- ProviderExpression以外に明確な構造差があれば通常のIndex変更になる
- lifecycle state driftを失わない
- `migration.Render`がUnresolvedを含むDiffを拒否する
- 既存の確定Diffの出力を維持する
