# reference-semantic-extension-catalog

RQ-031〜RQ-033の最初のReferenceです。目的は、TEI Coreへ新しいSemantic kindを追加せずに、別の意味視点を後付けし、既存Source Indexのsubjectと同じ探索面へ載せられるかを圧力試験することです。

## 何を固定しないか

このtoolのdescriptor JSONはReference専用です。

- production DSLではありません
- Canonical Worldの新しい正本形式ではありません
- Plugin runtime descriptorの拡張仕様でもありません
- Marketplace / remote registryの設計でもありません
- Extensionを選択したことはAcceptanceやRuntime authorityを意味しません

Referenceで複数の意味体系に共通する欠落が確認されるまでは、Core schemaへ昇格しません。

## 検証する境界

Reference descriptorは、次を機械可読に持ちます。

- Extension identity
- namespace
- version
- applies-to kind
- compatible-major dependency
- open-ended subject kindを持つconcept
- concept間、またはCore subjectへのrelation
- Constraintとvalidator reference
- Projection / Capability等へのconnection reference

resolverは選択されたExtensionのdependency closureを決定論的に作り、既存WorkspaceのSource Index subjectとExtension conceptを同じcatalogへ載せます。

```text
reference/workspace
  -> Source Index subjects
                       \
                        +-> semantic catalog
                       /
extension descriptors
  -> dependency resolve
  -> open-ended concepts
  -> relations / constraints / connections
```

## conflict方針

同じsubject identityをCoreとExtension、または複数Extensionが宣言した場合、読み込み順で勝者を決めません。

そのsubjectはresolved catalogから外し、`subject-conflict`を`unresolved`へ出します。Relation / Constraintが存在しないsubjectを参照した場合も、推測で補わずUnresolvedにします。

compatible majorを満たすExtensionが無い場合、dependency resolution自体をfail closedにします。同じmajor内に複数versionがある場合は、最も新しいversionを決定論的に選びます。

## Reference scenario

testdataではCoreに存在しない次のkindを追加します。

- `service-level-policy`
- `temporal-window`

`service-level-policy:support.case.standard-response`は、既存の`semantic-data:repair_intake.intake`を`governs`し、別Extensionが提供する`temporal-window:support.window.business-hours`を利用します。

Data ModelへService Levelの意味を押し込まず、新しい視点として横から追加するpressureです。

## 実行例

```bash
go run ./tools/reference-semantic-extension-catalog \
  -workspace reference/workspace \
  -extensions tools/reference-semantic-extension-catalog/testdata/extensions \
  -require tei.reference.service-level-policy@1
```

subjectを絞る場合:

```bash
go run ./tools/reference-semantic-extension-catalog \
  -workspace reference/workspace \
  -extensions tools/reference-semantic-extension-catalog/testdata/extensions \
  -require tei.reference.service-level-policy@1 \
  -subject-kind service-level-policy \
  -subject-id support.case.standard-response
```

## Authority boundary

出力には次を明示します。

- `catalog-is-not-canonical-authority`
- `extension-selection-is-not-acceptance`
- `connection-reference-is-not-runtime-authority`
- `reference-descriptor-is-not-production-dsl`

このReferenceが成立しても、Plugin / Extensionへwrite権限やResource authorityを与えるものではありません。

## 検証

focused test:

```bash
go test ./tools/reference-semantic-extension-catalog -count=1
```

現時点では、exact-headのfocused test / repository health PASSを取得するまでProduct Evidence成立とは扱いません。
