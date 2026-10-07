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

既知elementに新しい属性が増えた場合も、属性を破棄しない。現時点で意味を解釈しない属性でもObserved Documentには残し、Publication Receiptでは属性名の出現として確認できる。

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
