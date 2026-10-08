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

Evidence fingerprintは内容の同一性を示すだけで、観測者の署名やConformance、human approvalを証明しません。現在のexternal sourceとmapping contractへ再照合する場合は、Source Adapter側で同じnode ordinalからEvidenceを再構成し、fingerprint一致を確認します。

Shared Acceptanceへ接続するときは、このschemaをtyped artifactとして検証します。任意JSONをartifact Evidenceとして受け入れるための一般化には使いません。
