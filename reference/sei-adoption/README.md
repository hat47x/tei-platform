# SEI Reference Adoption fixtures

このdirectoryは、SEI CognitionのApplication PlaneをTEI-firstで実装できるかを検証するための、**read-only Reference snapshot**を置きます。

現在のfixture:

- `decision-continuity-counterfactual-001.json`
- source repository: `hat47x/sei-cognition`
- source revision: `e943683fbd71ea689bf40fb4c7009531fe98ece5`
- source path: `evaluation/personal-inquiry-decision/decision_continuity_counterfactual_001.json`
- source Git blob: `6354fce0a01093cb5545ed6bb9730aceb3b20023`

このcopyはTEI側の正本ではありません。SEI sourceをReference runごとにnetwork越しで取得する必要をなくし、同一bytesに対する再現可能なprobeを行うための固定snapshotです。

TEI側snapshotのGit blobも`6354fce0a01093cb5545ed6bb9730aceb3b20023`であり、固定SEI source blobと同一bytesです。

```text
SEI pinned revision/blob
  = this Stage 0 authority baseline

TEI reference copy
  = frozen experiment input
  != second authority

pinned-baseline conformance
  != current SEI main conformance
```

2026-09-18のcross-repository再確認では、SEI current `main`は`a485a658ee622433990dcb7323859a70f4be6f49`で、固定revision `e943683f...`はその254 commit後の子孫です。current `main`にはfixture pathがありません。一方、`develop/v0.2.0`等のactive refは固定revisionを祖先として保持し、`develop/v0.2.0`上でも同じsource path / blobを確認できました。

branch refはnavigation情報でありauthority fingerprintにはしません。このStage 0が主張するのは固定revision/blobへの適合だけで、current source conformanceは`not-claimed`とします。

fixtureのschemaをTEIのCanonical schemaへ昇格させません。変更が必要な場合はSEI側source revisionを先に固定し、別revisionのReferenceとして明示的に更新します。

最初のprobeではfixtureの`scenario`を入力とし、`conditions.C_sei_resume.surface_text`は期待Projectionのoracle、`reference_state`はsemantic audit oracleとして扱います。既に生成済みのResume Projectionを入力にして再表示するだけの自己循環は避けます。

## Change 1 execution Evidence

2026-09-18のChange 1実行結果は `change1-execution-20260918.json` に固定しています。

GitHub current blobと一致する `main.go` / `main_test.go` / fixtureを隔離再構成し、Go 1.23.2で `gofmt` 0差分、`go test` 9/9 PASS、`go vet` PASS、Reference tool実行時のsemantic audit全項目trueを確認しました。

これはexact sourceに対するReference-local Executable Evidenceです。ただしminimal temporary `go.mod`を使うisolated package実行であり、TEI repository全体のverificationではありません。
