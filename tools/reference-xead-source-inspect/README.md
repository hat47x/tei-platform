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

- source format / SHA-256
- XML construct inventory
- unknown construct名
- candidate classificationの集計
- legacy / runtime keyとRelation mappingの集計
- DERIVABLEなkey memberの件数
- DERIVABLEなkeyを参照するRelationshipの件数
- unresolved diagnostic codeとsource-locator digest
- transcription contract fingerprint
- normalized classification policy fingerprint
- authority limit

出力はReference Modelの利用条件を変更しません。public RepositoryへEvidenceを保存する場合は、
`plan/REFERENCE_MODEL_PUBLICATION_BOUNDARY_2026-10.md`に従います。
