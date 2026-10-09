# Xead / Xeaf Source Adapter Reference

GitHub #950のsource-faithful adapter候補です。

`Observe`はXMLをローカル観測用の`ObservedDocument`へ読み込みます。未知要素や未解釈属性も捨てません。Observed Documentには元sourceの属性値やテキストを含み得るため、public Evidenceとして保存しません。

`PublicReceipt`は、artifact digest、既知標準constructの件数、attribute出現数、unknown総数、locator総数と集合fingerprint、必要なsource factのname-independent locator token、authority limitだけを残します。元sourceの属性値・テキスト・任意のunknown schema名・構造pathは出力しません。

独自fixtureのテストでは、同一bytesの決定性、未知constructの保持、公開receiptからのsource値除去、legacy key系source factのauthority limitを確認します。

Repository外のexact `MfgOrder.xead` / `MfgOrder.xeaf`にも適用し、既存EvidenceのSHA-256と主要件数が一致すること、全12,771 elementが既知constructとしてaccountingされることを確認しました。原model本体やObserved DocumentはRepositoryへ追加していません。

この実装はproduction ImporterでもCanonical authorityでもありません。Plugin登録境界はISSUE-000080 / GitHub #951へ分離しています。

## 実行

```bash
go run ./tools/reference-xead-source-adapter -format xead -input /path/to/model.xead

go run ./tools/reference-xead-source-adapter -format xead -input /path/to/model.xead \
  -contract reference/reference-model-transfer/transcription-contract-v0.json
```

`-contract`を付けた場合は、Publication Receiptに加えて、観測したelement kindが転写契約へ割り当て済みかを返します。mapping entryが存在してもSemantic Acceptance済みとはみなしません。


## XMLレベルの観測

element / attributeに加え、XML comment、processing instruction、directiveをローカル観測でaccountingします。公開receiptには内容を出さず、件数だけを残します。

namespace付きの要素はlocal nameだけで既知constructに一致させません。同名の標準constructに見えてもunresolvedとし、Classification coverageを未完了のままにします。


## Candidate classification

`-contract`を指定すると、coverageだけでなく`classificationProjection`も返します。これは観測したelement kindごとに、転写契約の`classification`、担当Extension、authority limitを結び付けたpublic-safeな集計です。

`mappingStatus`とsemanticな`classification`は別です。`mappingStatus: mapped`でも、転写契約側のclassificationに`unresolved`を含む場合があります。前者は「転写方針が存在するか」、後者は「意味が未確定か」を表します。

namespace付きlookalike等が通常のkindと同じlocal nameで混在した場合は、`mappingStatus: mixed`と`unresolvedObservedCount`を返します。既知の`Table` mappingがあることを理由に、未知namespaceの`Table`を既知扱いしません。

exact source locatorごとのclassificationが必要な場合だけ、`-local-candidates`を追加します。

```bash
go run ./tools/reference-xead-source-adapter \
  -format xead \
  -input /path/to/model.xead \
  -contract reference/reference-model-transfer/transcription-contract-v0.json \
  -local-candidates
```

`localCandidates`にはexact source snapshot内のstructural locatorが入ります。これはローカル調査用であり、public Reference Model Evidenceへ保存しません。属性値やsource本文は含みませんが、locator自体をRepositoryへ公開する必要もありません。


## Unresolved projection

`-contract`を指定した出力には`unresolvedProjection`も含めます。これは、loss ledgerへ取り込む前段の集計です。

未解決状態を三つに分けます。

- source observationで未知・namespace付きとして残ったもの
- mappingが無い、または既知要素と未知要素が混在するもの
- mappingはあるが、転写契約上の意味が`unresolved`のもの

このprojectionからRML番号を自動生成しません。永続的なloss ledgerは論点単位で管理するため、集計結果を確認してから既存項目へ結び付けるか、新しい論点として起票します。

`mixed`なkindでは、未知側の要素を既知側Extensionへ帰属させません。Extensionを出すのは、mappingが実際に適用されたsemantic candidateだけです。

`unresolvedProjection`は件数とkind、未解決の種類だけを扱い、source valueやstructural locatorは公開しません。

## Publication Receipt v2

public出力では、任意のunknown element名や追加attribute名を公開しません。exact name / value / textはローカル`ObservedDocument`に保持し、public側はknown標準constructの件数、known constructごとのattribute出現数、unknown総数へ集約します。

`artifactSha256`がexact source bytesを固定するため、public Evidenceへunknown schema名のhash一覧を追加して擬似匿名化することもしません。詳細比較が必要な場合はRepository外のObserved Documentを使います。

この変更によりPublication Receiptは`schemaVersion: 2`です。v1のreceipt fingerprintをv2へ流用しません。

## Source format profile

`Observe`はsourceをローカルEvidenceとして読む層なので、未対応`FormatVersion`でもparse可能な限り観測を保持します。

`-contract`を指定してclassification / coverageを行う場合は別です。転写契約の`sourceProfiles`に宣言されたroot kindと`FormatVersion`へ一致しなければmappingを適用しません。現在のReference contractはXead / Xeafとも`System` / `1.2`を宣言しています。

未対応versionを、element名が同じという理由だけで既知mappingへ流さないためのfail-closed境界です。対応versionの追加はCore変更ではなくSource Adapter / transcription contract側で行います。

既存の`transcription-map-v0.json`とR0 coverageは旧contract blobに結び付いた履歴Evidenceです。`sourceProfiles`追加後のcontractへ自動で再帰属させず、current contractでのcoverageは再実行後にのみ更新します。


## RMT-3 bounded Evidence

`BuildTranscriptionCandidateEvidence`は、exact `ObservedDocument`のnode ordinalを一件だけ指定し、`tei.reference.rmt-transcription-candidate-evidence/v0`を生成します。

Evidenceはexact source artifact digest、FormatVersion、name-independent source token、mapping contract fingerprintへbindします。known / mapped constructでは標準construct名、classification、担当Extensionを保持します。unknown / namespaced constructではsource名を公開せず、`unmapped`のまま残します。

`VerifyTranscriptionCandidateEvidence`はEvidenceに保存済みの`SourceToken`をexact external source内で再解決し、同じmapping contractからEvidenceを再構成します。replay時にnode ordinalを別入力として保持しません。source artifact digestが変わった場合、tokenがexact source内で解決できない場合、mapping contractが変わった場合はstaleまたは不一致として拒否します。

このEvidenceはCanonical Acceptance、human Decision witness、Source Adapter Conformance、Runtime Authorityを与えません。Shared Acceptanceへのtyped artifact接続はGitHub #1008 / #267側で扱います。

## Duplicate attribute

`encoding/xml`がtokenとして返すduplicate attributeをmapへ黙って上書きしません。同じexpanded nameのattributeが一つのelement内で重複した場合、`Observe`はparser-invalidとしてエラーにします。

特に`FormatVersion`の重複をfirst-wins / last-winsで解決するとsource profile判定を攻撃・誤判定できるため、semantic mappingへ到達する前に停止します。

## Single document root

`Observe`はXML token streamを複数文書の連結として受け入れません。root elementを一つだけ要求し、二つ目のroot、root外の非空白text、rootless inputをエラーにします。root前後の空白は許容します。

これにより、最初の`System.FormatVersion`でprofileを判定しながら、別rootのnodeまでclassificationへ混ぜる経路を閉じます。
