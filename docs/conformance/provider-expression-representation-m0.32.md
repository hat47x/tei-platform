# M0.32 提供元固有の式の表現と比較に関する準拠確認

## 目的

M0.32では、ProviderExpressionに保存した文字列が利用者の記述した式なのか、提供元が内部表現から復元した式なのかを区別します。表記の違いだけを根拠に破壊的な構造変更を生成しないための比較規則を定めます。

対象はIssue #54とADR-0012です。比較結果を確定できない場合は、M0.31で導入した`Diff.Unresolved`に記録します。

## 表現の種類

`physical.ProviderExpression`では、次の二種類の表現を区別します。

```text
authored
provider-normalized
```

`authored`は、保存先の定義などに利用者が記述した、正本となる式の文字列です。

`provider-normalized`は、提供元が自身の内部表現から復元した比較用の文字列です。ここでいう表記統一は、数学的に一意な標準形になったことを意味しません。

## 表記統一の由来

提供元が復元した表現には、必要に応じて次の来歴を付けます。

```text
ExpressionNormalization
  Authority
  ProviderVersion?
```

`Authority`は、提供元が管理するどの表記統一・式の復元規則によって生成したかを示します。

`ProviderVersion`は、表現の規則が提供元の版に依存するとき、どの版で比較できるかを示します。版を取得できない場合は推測で補いません。

## 期待する構造

保存先への対応付けから期待する物理構造として生成した式のキーと条件式は、`authored`になります。

式の文字列は前後の空白だけを除いて保持し、表記統一に関する来歴は付けません。

入力元索引が示す正本は引き続き保存先の定義ファイルです。提供元が復元した表現へ置き換えません。

## PostgreSQLの実状態

PostgreSQLの管理情報を読む接続部が`pg_get_expr`や`pg_get_indexdef`から取得した式と条件式は、`provider-normalized`として扱います。

M0.32では、表記統一の規則を識別する値を次のとおり定めます。

```text
postgresql-catalog-deparse
```

この段階のCatalogReaderは、式の来歴としてサーバーの版を返しません。そのため、`ProviderVersion`は空のままです。

提供元の版が分からない場合、復元された文字列が違っても意味が異なるとは断定しません。

## 比較規則

提供元またはSQLの方言が異なる場合は、明確な構造差として扱います。

提供元とSQLの方言が同じ場合は、次の順に判断します。

### 入力文字列が同じ場合

表現の種類が異なっていても一致とします。

これは式の数学的な同値性を推測したのではなく、提供元固有の式の文字列が一致しているためです。

### 利用者記述どうしで入力が異なる場合

利用者が異なる式を記述した事実に基づき、構造差として扱います。

TEIが一方をもう一方へ自動的にcanonicalizeしたものではないため、利用者が明示的に異なる式を書いた事実を保持します。

### 提供元が統一した表現どうしで入力が異なる場合

次の条件をすべて満たす場合だけ構造差と確定します。

- normalization authorityが同一かつ空でない
- provider versionが同一かつ空でない

同じ規則と同じ版で復元したにもかかわらず、異なる文字列が得られたためです。

版または表記統一の規則が不明な場合や、規則が異なる場合は`Unresolved`とします。

### 利用者記述と提供元表現の入力が異なる場合

信頼できる表記統一の処理をまだ経ていないため、`Unresolved`とします。

この状態を根拠に`drop-index + add-index`を生成することはしません。

### 旧形式の表現

`Representation`が空の旧形式のデータは、式の文字列が一致する場合に限って一致と判断します。

文字列が異なる場合は、表現の種類を推測せず`Unresolved`とします。

## 移行時の安全性

M0.31の規則に従い、`Diff.Unresolved`が残る場合は移行用SQLを生成しません。

したがって、期待する式とPostgreSQLが復元した式の表記だけが異なる可能性がある場合、自動的な索引の削除や再作成には進みません。

## 非目標

- PostgreSQL SQL parserのCore内製
- SQL式の数学的同値性判定
- arbitrary textをPostgreSQL catalog deparseへ通す仕組み
- server version取得SPIの追加
- offline provider normalizer policyの最終形（Issue #59）
- collation / operator class等のnormalization（Issue #58）
- Advanced Index DDL rendering（Issue #55）

## 準拠確認

M0.32では、少なくとも次の条件を確認します。

1. Desired expression / predicateは`authored`になる
2. PostgreSQL Actual expressionは`provider-normalized`になり、authorityを保持する
3. authored / authoredの異なるsourceは構造差になる
4. authored / provider-normalizedの異なるsourceはUnresolvedになる
5. 同じauthority・同じ既知versionのprovider-normalized同士は異なるsourceを構造差と判定できる
6. version不明のprovider-normalized同士は異なるsourceをUnresolvedにする
7. normalization authorityが異なる場合はUnresolvedにする
8. source textが完全一致すればrepresentationが異なっても収束する
9. legacy representationの既存M0.31安全挙動を維持する
10. unresolved comparisonからmigration SQLを生成しない

## 次の段階

M0.32によって、式の表現の種類と比較規則が物理構造上で明示されます。

次の課題は、Issue #59で接続を伴わない物理構造生成と表記統一の規則をどこで管理するか整理することです。そのうえで必要なら、利用者が記述した式から比較可能な提供元固有の表現を生成する境界を設計します。
