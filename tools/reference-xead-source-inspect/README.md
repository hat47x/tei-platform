# reference-xead-source-inspect

Repository外に置いたXead/Xeaf source snapshotを、RMT-2のsource-faithful adapterと
`transcription-contract-v0.json`で検査するReference toolです。

このtoolはCanonical sourceを変更しません。標準出力も、原modelのraw XML、業務名称、
legacy ID、完全なsource locatorを複製しない集計Evidenceに限定します。

~~~text
go run ./tools/reference-xead-source-inspect \
  -format xead \
  -source /path/to/MfgOrder.xead
~~~

主な出力:

- schema version（現在は `reference-xead-source-inspection-v3`）
- source format / SHA-256
- XML construct inventory
- unknown construct件数（名称は公開出力しない）
- candidate classificationの集計
- legacy / runtime keyとRelation mappingの集計
- DERIVABLEなkey memberの件数
- DERIVABLEなkeyを参照するRelationshipの件数
- unresolved diagnostic codeと、source digest / resolution policy revision / diagnostic ordinalから作るopaque diagnostic token
- relational source fact全体の整合状態を示す `relationalResolutionStatus = resolved | partial | unresolved`
- fact種別ごとの `resolved / unresolved / unvalidated` 件数を示す `relationalFactResolutions`
- readiness判定算法のrevisionを示す `relationalResolutionPolicyRevision`
- transcription contract fingerprint
- normalized classification policy fingerprint
- authority limit

出力はReference Modelの利用条件を変更しません。public RepositoryへEvidenceを保存する場合は、
`plan/REFERENCE_MODEL_PUBLICATION_BOUNDARY_2026-10.md`に従います。


`relationalResolutionStatus`と`relationalFactResolutions`は、source factをCanonical meaningとして承認したことを示しません。`relationalResolutionPolicyRevision`も判定算法のprovenanceであり、semantic authorityではありません。
`resolved`は、その検証scopeで構造diagnosticが残っていないことだけを表します。
`partial`は、diagnosticは無いものの現行validatorでは安全に検証できないfactが残る状態です。
`unresolved`の場合も観測したsource factはEvidenceとして保持します。

全体statusが`unresolved`でも、無関係なfactまで自動的に意味不明になるとは限りません。
内部Inspectionは各factを`resolved / unresolved / unvalidated`へ分けます。ただし局所`resolved`はloweringの「候補資格」であり、
Canonical meaning、business identity、functional dependency等の承認ではありません。局所loweringを行う場合も、
そのfactが依存するKey / Table / Field / Relationshipに未解決が伝播していないことをexplicitなResolution Projectionから確認します。

公開reportではraw locatorやlegacy IDを出さず、fact種別ごとの件数だけを出します。

diagnostic tokenもsource locator文字列のhashにはしません。構造pathに業務名や独自schema名が含まれる場合、単純なlocator hashは候補pathから辞書照合できるためです。v3ではexact source digest、readiness判定policy revision、diagnostic ordinalからtokenを生成します。このtokenは同一report内の相関用であり、source node identityやCanonical identityとして使いません。
