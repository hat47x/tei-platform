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

## 依存versionの解決

Reference resolverは、同じmajorの候補では高いversionを先に試します。ただし、そのversionの推移依存が全体の制約と両立しない場合は、低いcompatible versionへ戻って探索します。

これはload順を決定論的にするだけの処理ではありません。最高versionを固定すると衝突する一方、低いcompatible versionなら全体を解決できる場合を、誤って「解決不能」と判定しないためです。

```text
A -> X@1
B -> Y@1

X 1.2 -> Z@2
X 1.1 -> Z@1
Y 1.0 -> Z@1
```

この例では、`X 1.1`と`Z 1.x`を選べば全体を解決できます。

Reference段階のconstraint languageは、`Extension ID + compatible major`に限定します。一般的なpackage solverを先に完成させることは目的にしません。現在のconstraint languageの範囲で、有効解があるのにgreedyな探索順だけで解決不能としないことを検証します。

versionごとに依存graphが異なる場合も、ある候補だけがcycleを作り、別候補ならcycleを避けられるときは代替候補を探索します。すべての有効候補がcycleを作る場合は解決失敗とします。

複数の有効解が存在する場合も、solver内部の偶然の探索順を隠れたpolicyにはしません。このReferenceでは、`highest-compatible-complete-search/by-extension-id/v0`を明示的なresolution policyとしてcatalogへ出力します。

このpolicyは、Extension IDを正規化した順で選択対象を決め、各IDについて高いcompatible versionから完全解が存在する候補を選ぶReference規則です。policy identityはSemantic resolution fingerprintへ含めます。将来、より適切なpolicyへ変更する場合はpolicy identityも更新し、同じSemantic Setとして黙って扱いません。

このReference policyをProductionの最終contractとして固定するものではありません。目的は「複数解をsolver実装の偶然で決めない」「選択規則が意味再現性へ影響することをfingerprintへ含める」ことの実証です。

backtrackingは探索量を増やし得るため、Reference helperにはcandidate attempt数で探索を止めるoperational budget境界も用意します。budget超過はdependency conflictとは別errorです。

budget値はSemantic resolution policyやSemantic Set fingerprintへ含めません。十分なbudgetなら同じ意味解へ到達し、不足したbudgetは「解がない」ことを意味しません。

## conflict方針

同じsubject identityをCoreとExtension、または複数Extensionが宣言した場合、読み込み順で勝者を決めません。

そのsubjectはresolved catalogから外し、`subject-conflict`を`unresolved`へ出します。複数の選択済みExtensionが同じnamespaceを所有する場合も、互換性contractが無い段階ではどちらかを暗黙に採用せず、双方のsemantic contributionを公開しないまま`namespace-ownership-conflict`として残します。Relation / Constraintが存在しないsubjectを参照した場合も、推測で補いません。

compatible majorを満たすExtensionが無い場合、dependency resolution自体をfail closedにします。同じmajor内に複数versionがある場合は、最も新しいversionを決定論的に選びます。root requirementは正規化してcatalogへ残し、同じExtensionに異なるmajorを同時要求した場合は拒否します。

## Semantic Interface

package identityへの依存と、意味契約への依存を分けて扱います。Semantic Interfaceは単なる名前ではなく、identity、contract version、正規化したcontract fingerprint、入出力の意味、unknown / failureの意味、provider conformance profileを持ちます。

provider側の宣言だけでは適合済みと扱いません。providerはInterface identity、exact contract version、contract fingerprintを宣言し、resolverには別入力のConformance Evidenceが必要です。Conformance Evidenceはprovider、Interface、exact contract fingerprint、conformance profileへ結び付けます。

適合するproviderが一つだけなら、そのproviderへ決定論的にwiringできます。複数の適合providerが残る場合はpackage名やversion順で選ばず、明示wiringが無ければ`interface-provider-ambiguous`としてUnresolvedにします。明示wiringで指定したproviderがcontractまたはConformance条件を満たさない場合もfail closedです。

`resolutionFingerprint`にはInterface contractとprovider wiringを含めます。一方、同じcontract、provider、profileに対するConformance Evidenceの更新だけでは意味identityを変えません。そのfingerprintは`evidenceFingerprint`側で追跡します。

CLIでは、必要に応じて`-interface-wiring FILE`と`-interface-conformance FILE`を繰り返し指定できます。どちらもstrict JSONとして読み込み、未知fieldや余分な末尾データを拒否します。

この形式もReference contractです。ProductionのPlugin manifestやSemantic Lock形式としてはまだ固定しません。

## Namespace authority

`std`と`std.*`はstandard authority用に予約します。Extension descriptorがそのnamespaceを名乗るだけではownershipを得ません。Acceptedなnamespace authorityを別入力で与えた場合にだけ、指定したExtensionのsemantic contributionを公開します。

同じnamespaceを複数Extensionが宣言しても、読み込み順では決めません。authorityが無ければUnresolvedにし、authorityが一つのExtensionを選んだ場合も、他のclaimantは`namespace-authority-mismatch`として残します。Coreが既に所有するnamespaceをauthority入力で上書きすることもできません。

namespace authorityのEvidence fingerprintはprovenanceとして`evidenceFingerprint`へ含めますが、同じnamespaceとownerに対するEvidence更新だけでは`resolutionFingerprint`を変えません。CLIでは`-namespace-authority FILE`を繰り返し指定できます。

subnamespace delegateやpublisher trustの最終contractはまだ固定していません。Referenceでは、first claimantをAuthorityにしないことと、standard Extensionも外部のAccepted Authority Evidenceを必要とすることを先に検証します。
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

## Identityとprovenance

catalogは意味の解決結果とSourceの来歴を分けます。

- `descriptorFingerprint`は、正規化したExtension descriptorの意味内容から作るidentityです。
- `sourceFingerprint`はdescriptor fileそのもののfingerprintで、保存時のprovenanceです。
- `coreSubjectFingerprint`は、catalogが実際に利用するCore subject集合のidentityです。
- `coreSourceIndexFingerprint`は、現在のWorkspace Source Index全体のprovenanceを示します。
- `resolutionFingerprint`は、Core subject集合、正規化root requirements、選択されたExtensionと解決結果から作ります。raw sourceのpathやfingerprintは含めません。
- `evidenceFingerprint`は、Core Source IndexとExtension descriptor fileのprovenanceを含むEvidence側のidentityです。

このため、subject集合を変えないSource Index更新や、descriptor fileを同じ意味のまま移動・再整形した場合は`resolutionFingerprint`を維持し、`evidenceFingerprint`だけを更新できます。Core subject集合やExtensionの正規化された意味が変われば、同じroot requirementsでも別のresolutionとして扱います。
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
