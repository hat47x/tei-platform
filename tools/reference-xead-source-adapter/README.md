# Xead / Xeaf Source Adapter Reference

GitHub #950のsource-faithful adapter候補です。

`Observe`はXMLをローカル観測用の`ObservedDocument`へ読み込みます。未知要素や未解釈属性も捨てません。Observed Documentには元sourceの属性値やテキストを含み得るため、public Evidenceとして保存しません。

`PublicReceipt`は、artifact digest、element / attribute名の集計、locator総数と集合fingerprint、必要なsource factのlocator digest、authority limitだけを残します。元sourceの属性値やテキストは出力しません。

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
