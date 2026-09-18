# SEI Reference Adoption fixtures

このdirectoryは、SEI CognitionのApplication PlaneをTEI-firstで実装できるかを検証するための、**read-only Reference snapshot**を置きます。

現在のfixture:

- `decision-continuity-counterfactual-001.json`
- source repository: `hat47x/sei-cognition`
- source revision: `e943683fbd71ea689bf40fb4c7009531fe98ece5`
- source path: `evaluation/personal-inquiry-decision/decision_continuity_counterfactual_001.json`
- source Git blob: `6354fce0a01093cb5545ed6bb9730aceb3b20023`

このcopyはTEI側の正本ではありません。SEI sourceをReference runごとにnetwork越しで取得する必要をなくし、同一bytesに対する再現可能なprobeを行うための固定snapshotです。

```text
SEI source
  = authority

TEI reference copy
  = frozen experiment input
  != second authority
```

fixtureのschemaをTEIのCanonical schemaへ昇格させません。変更が必要な場合はSEI側source revisionを先に固定し、別revisionのReferenceとして明示的に更新します。

最初のprobeではfixtureの`scenario`を入力とし、`conditions.C_sei_resume.surface_text`は期待Projectionのoracle、`reference_state`はsemantic audit oracleとして扱います。既に生成済みのResume Projectionを入力にして再表示するだけの自己循環は避けます。
