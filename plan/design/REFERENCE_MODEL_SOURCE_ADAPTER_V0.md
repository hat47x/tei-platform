# Reference Model Source Adapter v0

- 状態: Candidate
- 対象: ISSUE-000078 / GitHub #950
- 実装: `tools/reference-xead-source-adapter/`

## 目的

Xead / Xeafの設計資産をTEIへ転写する前に、元のXMLを欠落なく観測するためのReference実装を定める。

このAdapterはImporterではない。XMLを読めたことをCanonical Acceptanceへ読み替えず、旧モデルの要素名やIDをTEI Coreへ自動昇格させない。

## 二つの出力を分ける

Source Adapterは、Repository外で扱うローカル観測と、public Repositoryへ残せるreceiptを分ける。

### Observed Document

Observed Documentは元XMLを調べるためのローカル成果物である。属性値、要素内テキスト、legacy ID、SortKeyなどを保持できる。

この情報にはReference Model本文や環境依存情報が含まれ得るため、そのままpublic Repositoryへ保存しない。

Observed Documentの情報はsource factであり、Canonical authorityではない。

### Publication Receipt

Publication Receiptはpublic Evidenceとして扱える範囲に情報を減らす。

保持するのは次である。

- source artifactのSHA-256
- source format
- element kindごとの件数
- attribute名ごとの件数
- exact snapshotに結び付いたlocator総数と集合fingerprint
- source factの種類と、追跡が必要なfactだけに付与するlocator digest、authority limit
- unresolvedの種類とlocator digest
- receipt全体のfingerprint

属性値、要素内テキスト、legacy IDの値、説明文、Script本文は含めない。全要素のlocator digest一覧も公開せず、全体は件数と集合fingerprintへ集約する。

## Source Locator

Source Locatorは、exact source snapshot内の位置を示す。

構造上のpathは、親子関係と同種の兄弟要素内でのordinalから作る。artifact SHA-256と組み合わせることで、どのsnapshotのどの位置を指していたかを区別する。

legacy IDやSortKeyはローカル観測の補助情報として保持できるが、Canonical identityにはしない。

sourceの並び順が変わった場合、snapshot locatorの対応が変わることは許容する。その差をsemantic identityの変更とはみなさない。

## unknown constructの扱い

未知のelementをwarningだけで捨てない。Observed Documentへ要素名、属性、テキスト、locatorを残し、unresolvedとして数える。

既知elementに新しい属性が増えた場合も、属性を破棄しない。現時点で意味を解釈しない属性でもObserved Documentには名前と値を残す。public Publication Receiptでは任意の追加attribute名は公開せず、known constructごとのattribute出現数として観測事実だけを残す。

「意味をまだ理解していない」と「sourceに存在しなかった」を分けるためである。

## source factとauthority limit

初期Referenceでは、意味へ誤昇格しやすい次の情報をsource factとして抽出する。

- TableKey
- TableKeyField
- Relationshipのkey reference
- Function / DetailのKeyFields

ローカル観測では実際の値を保持する。Publication Receiptでは値を除き、fact種別とattribute名だけを残す。

これらのfactには、business identity、candidate key、functional dependency、physical key authority等へ自動昇格しないことをauthority limitとして付ける。

## full-source accounting

2026-10-07にRepository外のexact sourceへReference実装を適用した。

| Source | SHA-256 | 全element | 未認識element |
| --- | --- | ---: | ---: |
| MfgOrder.xead | `31af1fe317c661f57c94c69285b7b3e91c7c2fcad940cd7639956aff781088a9` | 8,718 | 0 |
| MfgOrder.xeaf | `f4b73b36abd8398aa25ad86ecd7570ae0334e7fd93c3ee204c5a05bddebdb754` | 4,053 | 0 |

初期の転写契約に含まれていなかったconstructもこの走査で見つかった。Xeadでは`TableType`、`SubsystemTable`、`SubsystemRelationship`、`IOPanelField`、`IOPanelStyle`、`IOSpoolStyle`、`MaintenanceLog`、Xeafでは`HeaderPhrase`、`Phrase`、`Option`、`PrintFont`、`BatchField`、`AddRowListColumn`、`AddRowListButton`、`MaintenanceLog`である。

これらをsource inventoryとtranscription contractへ追加した。多くはPresentation、Projection、Provenance、Realizationとして扱い、Core固有概念にはしていない。

未認識elementが0件であることは、source constructを数えられたことだけを示す。意味の転写が完了したことや、TEIとのsemantic equivalenceは示さない。

## Plugin境界

Source Adapterはauthoring時のTool Capabilityである。Application Runtime Authorityを持たない。

Plugin Capability Contractの一般化はISSUE-000080 / GitHub #951で扱う。このAdapterを登録するためだけにCoreの固定`CapabilityKind`へ`source-adapter`を追加しない。

#951の契約が確定するまで、#950はreader、locator、inventory、receipt、source fact projectionを独立して進める。

## 公開境界

Reference Model本体はRepositoryへ含めない。テストfixtureはTEI側で独自に作り、原モデルの名称、文章、Script、画面文言を転写しない。

exact modelを用いたcoverage確認では、public Repositoryへ集計値、digest、分類、authority limitだけを戻す。

## 現時点の非主張

このReferenceは次を主張しない。

- X-TEA / XEADの完全互換Importer
- Source Adapterのproduction Plugin contract
- 旧Runtimeとのbehavior equivalence
- source factからCanonical meaningへの自動変換
- Reference Model全体のsemantic transcription完了

## Classification coverage

Source Adapterがsourceを観測できることと、そのconstructに転写方針があることも分ける。

`CheckClassificationCoverage`はローカル観測上のelement kindと`transcription-contract-v0.json`の`constructMappings`を比較する。既知の標準constructでmappingがない場合は名前を出して診断できるが、任意のunknown kind名はpublic出力へ載せず、`unmappedObservedKindCount`と`unknownObservedElements`へ集約する。

2026-10-07時点では、exact sourceで観測したXead 30 kind、Xeaf 21 kindのすべてにmapping entryがある。

この結果は転写方針の網羅性だけを示す。mapping entryの存在をSemantic Acceptance、behavior equivalence、Runtime Authorityへ読み替えない。


## XMLレベルのaccounting

elementとattributeだけを数えても、source全体を観測したことにはならない。Reference実装では、XML comment、processing instruction、directiveもローカル観測へ残す。

public Publication Receiptへはmarkup本文を出さず、種類ごとの件数だけを残す。benchmarkのexact sourceでは、Xead / XeafともXML declarationに対応するprocessing instructionが1件ずつあり、comment、directiveは0件だった。

namespaceもsource factとして区別する。たとえばnamespace付きの`Table`は、local nameが同じでもXead標準の`Table`として扱わない。unknown / namespaced elementが観測されている場合、転写契約に同名のmappingがあってもClassification coverageをcompleteにしない。

この境界は、将来のXead/Xeaf派生形式や拡張metadataを「既知の要素名に見える」という理由だけで取り込まないために必要である。


## Candidate classification boundary

Source Adapterの観測結果をCanonicalへ直接変換しないため、classificationを独立したcandidate段階として扱う。

ローカルでは、各Observed Nodeについてsource locator、source kind、転写契約上のclassification、担当Extension、rules、authority limitを`LocalTranscriptionCandidate`へ投影できる。mappingが無い場合や未知namespaceの要素は、推測で近いmappingへ寄せず`mappingStatus: unmapped`のまま残す。

public Evidenceではlocator単位の一覧を出さず、kind単位の`ClassificationProjection`へ集約する。通常の既知要素とnamespace付きlookalike等が同じlocal nameで混在する場合は`mappingStatus: mixed`と未解決件数を返す。

mappingの有無とsemanticな未確定状態は別である。転写契約にmappingが存在しても、そのclassificationが`canonical-candidate`や`unresolved`であればAccepted meaningではない。逆に`mappingStatus: unmapped`は、意味が誤りだと断定するのではなく、現在の転写契約では扱いを決めていないことを表す。

この分離により、Source Adapterは次の三段階を混同しない。

```text
source observation
  -> mapping/classification candidate
  -> Shared Acceptance後のauthoritative meaning
```

CLIの`-local-candidates`は最初の二段階をsource locator付きで調べるためのローカル機能であり、public Evidenceへ保存しない。


## Unresolved projection boundary

Source Adapterで見つかった未解決状態を、そのまま恒久的なloss ledgerへ書き込まない。まず`UnresolvedProjection`へ集約する。

projectionでは、source観測時の未知状態、mappingの未決定、mapping済みcandidateのsemantic未確定を区別する。これにより、XMLを理解できなかった状態と、転写方針がまだ無い状態と、転写方針はあるが意味を確定していない状態を一つの`unresolved`へまとめない。

`mixed`なkindでは、既知要素へ適用されたExtensionを未知側の観測へ引き継がない。未知側はmapping未決定のまま扱う。Extensionは、mappingが実際に適用されたsemantic candidateにだけ結び付ける。

public projectionにはsource valueとstructural locatorを含めない。件数、kind、未解決の種類、mapping status、必要なauthority limitだけを残す。

`loss-ledger.json`は論点単位の台帳であり、このprojectionからRML IDを自動発番しない。複数の観測が同じ既存論点へ集約される場合もあれば、新しい論点を示す場合もあるためである。

## Public schema-name redaction

Publication Receiptで値と本文を除くだけでは十分ではない。企業独自のXead / Xeaf派生形式では、未知element名や追加attribute名そのものがローカルな業務語彙・内部schemaを表す可能性がある。

旧v1 Reference実装は`elementCounts` / `unknownElements`へ未知elementのlocal nameを、`attributeNameCounts`へ任意のattribute名を出力していた。benchmarkではunknown elementが0件のため、この問題は通常のcoverage結果には現れなかった。2026-10-08のv2 candidateでは、これらの任意名をローカル観測へ限定し、public出力をknown標準constructの件数、known constructごとのattribute出現数、unknown総数へ縮退させた。

したがってpublic-safe境界では、任意のsource由来schema名をローカル`ObservedDocument`へ限定する。public Evidenceには存在件数とdeterministicな集合fingerprint等を残し、unknownをsilent dropせずに名称自体は公開しない。

既知のXead / Xeaf標準schema名についても、公開可能性はReference Model本体とは別に根拠を持つ。少なくとも「unknownだからpublicへ名前を出してよい」とは扱わない。

v2 candidateでredaction境界を実装したが、現headのfocused testとfull Repository Healthは未取得である。RMP-047 / RML-022は、redaction testのactual PASSを得るまで停止条件として維持する。
