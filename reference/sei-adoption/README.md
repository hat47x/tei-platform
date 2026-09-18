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

2026-09-18の最初のChange 1実行結果は `change1-execution-20260918.json` に固定しています。これは当時のexact blob:

- `main.go`: `a62c0a10ac32d3c84156b74cec57b83f12866dfb`
- `main_test.go`: `7e06b9e289fa00bb19345240146844abad8a5989`
- fixture: `6354fce0a01093cb5545ed6bb9730aceb3b20023`

に対する履歴Evidenceです。

その後のChange 1統合でpinned-source authority metadataが一度後退したため、authority境界を復元したcurrent sourceを別途exact-blob再実行しました。結果は `change1-authority-restored-execution-20260918.json` に固定しています。

- `main.go`: `52a53930ceae1c5c173a020d7d3834fe08940bb3`
- `main_test.go`: `8288d3ab7b81942626b6cbb992a971475675b0ec`
- fixture: `6354fce0a01093cb5545ed6bb9730aceb3b20023`
- Go 1.23.2 / linux/amd64
- `gofmt -d`: 0 bytes
- `go test -v ./tools/sei-resume-reference`: 10/10 PASS
- `go vet ./tools/sei-resume-reference`: PASS
- `go run ./tools/sei-resume-reference`: PASS
- report: `authority_scope = pinned-revision-only`
- report: `current_source_conformance = not-claimed`

Change 1のtemporal projection pressureとpinned-source authority / snapshot blob integrityが同じcurrent sourceで共存することまで確認できました。

どちらのreceiptもminimal temporary `go.mod`を使うisolated package実行であり、TEI repository全体のverificationではありません。またSEI current source conformanceやnative SEI再実行を意味しません。
