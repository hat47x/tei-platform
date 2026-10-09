# RMT transcription candidate Evidence

`tei.reference.rmt-transcription-candidate-evidence/v0`は、Reference Model Transferで観測した一つのsource nodeを、Shared Acceptanceへ渡すためのtyped Evidence contractです。

このEvidenceはsemantic changeそのものではありません。exact external source artifact上のどの観測を根拠にしたかを固定し、semantic deltaはShared AcceptanceのCandidate側で別に明示します。

保持するのは次です。

- source format / FormatVersion
- exact source artifact digest
- source nodeを指すname-independent token
- mapping contract fingerprint
- known standard constructであるか
- mapped / unmapped
- candidate classification / Extension
- authority limit
- Evidence fingerprint

保持しないものは次です。

- source value / text
- legacy ID値
- exact structural locator
- arbitrary unknown element / attribute名
- Reference Model本文
- Acceptance Decision
- Runtime Authority

unknown / namespaced constructは名前をEvidenceへ出さず、`sourceConstructKnown=false`かつ`mappingStatus=unmapped`のまま保持します。

Evidence fingerprintは内容の同一性を示すだけで、観測者の署名やConformance、human approvalを証明しません。現在のexternal sourceとmapping contractへ再照合する場合は、Source Adapter側でEvidenceに保存された`SourceToken`をexact source内で再解決し、そのnodeからEvidenceを再構成してfingerprint一致を確認します。再入場時に元のnode ordinalを会話や一時状態から渡し直す必要はありません。

Shared Acceptanceへ接続するときは、このschemaをtyped artifactとして検証します。任意JSONをartifact Evidenceとして受け入れるための一般化には使いません。


## 公開語彙の境界

typed Evidenceは、producerが`sourceConstructKnown=true`と書いただけではconstructをknownとして扱いません。Xead / Xeafごとに公開可能な標準constructを検証し、異なるformatのconstructや任意の名称を拒否します。

`classification`もRMTで定義した語彙だけを受け付けます。`mappingStatus=unmapped`の場合は`classification=unresolved`だけを許可し、Extensionを付けません。これにより、Source Adapterを経由せずに手作業でEvidenceを組み立てても、unknown constructをmapped candidateとして表現できないようにします。

この語彙検査はobserver Conformanceの代わりではありません。標準construct名として妥当であることと、qualified Source Adapterが実際のsourceを観測したことは別の主張です。

Source Adapter、`transcription-contract-v0.json`、typed Evidenceは責務が異なるため、語彙の定義自体は別に持ちます。ただし、三者の更新ずれはReference testで検出します。Xead / Xeafのconstruct集合と、転写契約で使うclassificationが一致しなければtestを失敗させます。
