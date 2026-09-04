# ADR-0012: ProviderExpressionのauthor textとprovider-normalized representationを分離する

- 状態: 採用
- 日付: 2026-09-01
- 関連Issue: #54, #58, #59, #60
- 前提: ADR-0010, ADR-0011 / M0.29〜M0.31

## 背景

M0.29では、PostgreSQLのexpression indexとpartial predicateをActual Physical Schemaへ取り込むため、`pg_get_indexdef`や`pg_get_expr`から得た文字列を`ProviderExpression.Source`へ格納しました。

M0.30では、同じ型をDesired側でも使い、Persistence authoringに書かれたprovider-native expressionを`Source`へ格納しました。

しかし、二つの`Source`は同じ性質ではありません。

- Desired側は、利用者が記述したauthoritativeなauthor textです。
- Actual側は、DBMSが内部表現から再構成したprovider-decompiled textです。

PostgreSQLの`pg_get_expr`や`pg_get_indexdef`は、元のSQL文字列を保存して返す仕組みではありません。内部表現からSQL表現を再構成します。したがって、DesiredとActualの`Source`を単純な文字列一致だけで比較すると、同じ意味でも表記差によってfalse driftになる可能性があります。

一方、TEI CoreへPostgreSQL専用のSQL parserや一般的な式同値性判定器を組み込むことも避けます。Provider固有の表現問題を、共通Coreの文字列補正規則へ押し込むべきではありません。

M0.31では、この問題に先立ってSchema Diffへ`Unresolved`を導入しました。これにより、「構造差がある」と「比較を確定するだけの情報が足りない」を分離できます。

## 判断

`ProviderExpression`は、provider-nativeな文字列だけでなく、その文字列がどの表現段階にあるかを明示します。

概念上は次です。

```text
ProviderExpression
  Provider
  Dialect
  Representation
  Source
  Normalization?
```

`Representation`は、少なくとも次を区別します。

```text
authored
provider-normalized
```

`provider-normalized`では、比較可能な範囲を説明するためのprovenanceを持てます。

```text
ExpressionNormalization
  Authority
  ProviderVersion?
```

ここでいう`provider-normalized`は、数学的に同値な式を必ず同じ文字列へ変換する普遍的なcanonical formではありません。特定providerの内部表現から、そのprovider自身またはprovider-ownedな処理によって再構成された比較用表現です。

## authoredは利用者の記述を表す

Persistence authoringからDesired Physical Schemaへrealizeしたexpression keyとpredicateは`authored`です。

利用者が書いたsource textを、normalizationの都合でprovider-decompiled textへ置き換えません。Source IndexやEditorから戻るauthoritative sourceも、引き続きPersistence authoringです。

前後の空白を除く既存の軽い整形は維持しますが、SQLの意味を書き換える正規化は行いません。

## PostgreSQL Actualはprovider-normalizedとして扱う

PostgreSQLのActual observationで取得したexpression keyとpredicateは`provider-normalized`です。

M0.32ではnormalization authorityを次の値で記録します。

```text
postgresql-catalog-deparse
```

これは、PostgreSQL catalogの内部表現から`pg_get_expr` / `pg_get_indexdef`系の経路で再構成された表現であることを示します。

現行の`CatalogReader`はserver versionをexpression provenanceとして返しません。そのため、M0.32のPostgreSQL Actualでは`ProviderVersion`を推測せず空のままとします。

## provider versionを推測しない

provider-normalized representationは、provider versionによって表記が変化する可能性があります。

そのため、異なるprovider-normalized textを「確実に別の構造」と判定するには、少なくとも同じnormalization authorityに加え、同じ既知のprovider versionで生成されたことを確認します。

M0.32では次を採用します。

- 同じauthority、同じ既知version: 異なるnormalized textを構造差と判定できる
- version不明: 異なるnormalized textだけでは構造差と断定しない
- authority不明または異なる: 異なるtextだけでは構造差と断定しない

server versionの取得方法そのものは、現在のruntime introspection SPIへ場当たり的に追加しません。必要なprovider-owned policyやmetadata境界はIssue #59で整理します。

## 比較規則

ProviderExpressionの比較は、一致・不一致・未確定の三つを区別します。

### ProviderまたはDialectが異なる

明確な構造差です。

異なるproviderやdialectの式を、文字列が似ているという理由だけで同一視しません。

### Sourceが同一

Provider / Dialectが同じで、trim後のSourceが同一なら一致とします。

`authored`と`provider-normalized`の組み合わせであっても、比較対象のprovider-native text自体が一致しているためです。

### authored同士でSourceが異なる

明確な構造差です。

利用者が異なる式をauthoritative intentとして記述しているため、TEIが勝手に同義式と推測しません。

### authoredとprovider-normalizedでSourceが異なる

比較未確定です。

Desired author textがproviderのdeparse表現へ変換されたときに同じ文字列になるかを、この時点では確認できません。M0.31の`Diff.Unresolved`へ送り、`drop-index + add-index`を生成しません。

### provider-normalized同士でSourceが異なる

同じnormalization authorityと同じ既知provider versionを共有する場合だけ、構造差と判定します。

authorityまたはversionの比較契約が不足している場合は`Unresolved`とします。

### legacy representation

M0.31以前の`Representation`を持たない値は、Sourceが同一なら一致できます。

Sourceが異なる場合は、`authored`か`provider-normalized`かを推測せず`Unresolved`とします。旧データを理由に破壊的な変更へ進みません。

## 「比較未確定」はChangeではない

normalization情報が不足していることは、Database構造の変更そのものではありません。

したがって、M0.31で採用した次の分離を維持します。

```text
Diff
  Changes[]
  Unresolved[]
```

`Changes`には確定した構造差とActual lifecycle state driftだけを含めます。

`Unresolved`には、比較の確定に追加情報が必要な項目を保持します。

ProviderExpressionだけが未確定で、他に明確なIndex構造差がない場合、Indexの`drop-index + add-index`は生成しません。

一方、INCLUDE、key数、sort order、unique semanticsなどに明確な構造差があれば、ProviderExpressionの未確定だけを理由に既知の差まで隠しません。そのIndexは通常の構造差として扱います。

## lifecycle stateとは分離する

`valid / ready / live`はActualから得た観測事実です。

ProviderExpressionの比較が未確定でも、Actual Indexが利用可能状態を満たしていなければ`index-state-drift`は従来どおり報告できます。

比較未確定と運用状態を同じ概念へ混ぜません。

## arbitrary textをCoreでcanonicalizeしない

`pg_get_expr`は、既にPostgreSQL catalogへ保存された内部expression treeをrelationの文脈でdecompileする仕組みです。任意のtextをread-onlyに渡してcanonical formを得るためのAPIではありません。

したがって、通常のWorkspace loadやvalidationでDB接続を必須化しません。また、CoreにPostgreSQL parserを内製しません。

将来normalizationが必要になった場合も、次のようなprovider-ownedな手段を検討します。

- provider versionに対応したoffline parser / deparser
- 隔離されたprovider環境でserver自身に解釈させる経路
- connected validationで生成した比較用表現を、source hashとnormalization provenance付きの生成物として保持する経路

production targetへ、normalizationだけを目的としたDDLを発行する設計にはしません。

## runtime Provider SPIへ直接足さない

現行`database/provider.Provider`は、`Capabilities`と`Introspect`を中心に、`context.Context`と`DatabaseInstance`を伴うruntime SPIです。

Desired realizationのoffline normalizationをこのinterfaceへ直接追加すると、Workspace buildへ不要なruntime assemblyやDB接続依存を持ち込む恐れがあります。

そのため、provider固有のpureなdefault解決やnormalization policyの所有境界はIssue #59で別に設計します。

ADR-0012では、将来のinterface shapeを先取りして巨大な抽象化を導入しません。

## provider identifier normalizationとは分ける

expression / predicateの表現問題と、collation / operator class / access method等のprovider identifierやeffective defaultの問題は関連しますが、同じ問題ではありません。

- ProviderExpression representation / comparison: Issue #54 / 本ADR
- expression以外のprovider default / identifier normalization: Issue #58
- provider-owned offline policy境界: Issue #59
- Schema Diffの比較未確定: Issue #60 / M0.31
- Advanced Index migration SQL: Issue #55

一つの巨大な`NormalizeEverything()`へまとめません。

## migrationとの境界

M0.31の契約どおり、`Diff.Unresolved`が残る場合はmigration SQLを生成しません。

これは「変更がない」とみなすのではなく、「安全に変更判断できるだけの比較情報がない」と扱うものです。

Issue #55でAdvanced Index DDLを実装するときも、この安全境界を維持します。

## Source Indexとの関係

ProviderExpressionにrepresentationやnormalization provenanceを追加しても、source navigationのidentityは変えません。

Persistence側のauthor textが引き続きauthoritative sourceです。M0.30で導入した`persistence-index:<mapping>/<index>`からIndex declarationへ戻れます。

provider-normalized expressionをSource Index上のauthoritative definitionへ昇格させません。

## 採用しない案

### Source文字列だけを永続的な比較キーにする

providerのdeparse差によるfalse driftを避けられないため採用しません。

### 表記が違う式をすべて構造差とする

安全側に見えても、不要なIndex再作成へ直結し得るため採用しません。

### 表記が違う式をすべて同一視する

本当に異なる式まで一致させるため採用しません。

### CoreにPostgreSQL parserを持つ

provider境界を壊し、version追随責任までCoreへ持ち込むため採用しません。

### provider-normalizedならversionを無視して比較できると仮定する

provider versionによるserialization差を検証せず固定することになるため採用しません。

## Conformance

M0.32では次を固定します。

1. Desired expression / predicateは`authored`になる
2. PostgreSQL Actual expression / predicateは`provider-normalized`になり、normalization authorityを保持する
3. authored同士の異なるSourceは構造差になる
4. authoredとprovider-normalizedの異なるSourceは`Unresolved`になる
5. 同じauthority・同じ既知versionのprovider-normalized同士は、異なるSourceを構造差と判定できる
6. version不明のprovider-normalized同士は、異なるSourceを`Unresolved`にする
7. normalization authorityが異なる場合は`Unresolved`にする
8. Sourceが同一ならrepresentationが異なっても収束する
9. legacy representationの異表記は保守的に`Unresolved`とする
10. unresolved comparisonからmigration SQLを生成しない
11. Actual lifecycle state driftを比較未確定へ混ぜない

## 影響

- Physical SchemaはProviderExpressionの表現段階を明示できる
- Desired author textを失わずにActual provider-decompiled textと区別できる
- false textual driftから破壊的migrationへ進む経路を抑止できる
- provider versionが不明なとき、不確実性を隠さず保持できる
- provider-owned normalizerの実装はIssue #59へ分離できる
- expression以外のprovider normalizationはIssue #58として独立して進められる
