# Knowledge Portability Executable Evidence Runner

- Status: focused CI runner baseline
- Date: 2026-09-19
- Scope: Knowledge Portability R0 / second Application Pre-Gate / schema sufficiency
- Product Gate impact: None
- Canonical source impact: None

## 1. 目的

これまでSource Evidenceとしてのみ保持していたKnowledge Portability / second Application系pressureを、GitHub-hosted runner上で再現可能なExecutable Evidenceへ上げる。

このrunnerはGate CやProduct Acceptanceを代替しない。

```text
test source exists
  -> Source Evidence

focused CI passes
  -> Executable Evidence

second Application accepted / reloads
  -> Reference Evidence

repeated development benefit
  -> Product Evidence
```

を分離する。

## 2. Workflow

`.github/workflows/knowledge-portability-reference-evidence.yml`

対象test prefix:

- `TestKnowledgePortabilityR0`
- `TestSecondApplication`

Goの`-run`で実行対象を絞るが、package全体はcompileされる。

したがって、

- focused behavior verification
- same package compile consistency

を同時に確認できる。

## 3. なぜ全Reference pressureを最初からGate化しないか

`tools/reference-business-procedure-check`には多数のresearch pressureが共存している。

全件実行を最初からrequired statusにすると、

- unrelated research lane
- unfinished pressure
- temporary source experiment

までKnowledge PortabilityのEvidence取得をblockし得る。

最初は同じ問題領域のtestだけを実行する。

runner安定後に対象拡張を検討する。

## 4. Trigger

pull request to `main`で、次の変更がある場合に実行する。

- Workflow自身
- Knowledge Portability / second Application pressure source
- `reference/workspace/**`
- `reference/business-system-design/**`
- 関連するApplication / Database / Environment / IR / Source Index / Syntax / Workspace model
- `go.mod` / `go.sum`

さらに`workflow_dispatch`を残す。

## 5. Evidence artifact

run結果を、

```text
knowledge-portability-reference-test.txt
```

としてartifactへ保存する。

成功時も失敗時もEvidence取得を試みる。

artifactはCanonical sourceではない。

## 6. 成功条件

- focused Go filesが`gofmt`済み
- package compileが成立
- targeted testsがPASS
- workflow runがcurrent PR head SHAへ結び付く
- artifactがrunへ結び付く

## 7. 非目標

- Gate B / C required check化
- 全Repository `go test ./...`
- 全Reference pressureの必須化
- Product Acceptance
- second Application authoring
- actual external Provider Evidence
- production runtime validation

## 8. Evidenceの扱い

successful runを得た場合にのみ、

```text
Knowledge Portability R0
second Application Pre-Gate contract
second Application schema sufficiency
```

についてExecutable Evidenceが得られたと記録する。

runが存在しない、queued、cancelled、failureの場合はPASSを主張しない。

## 9. 将来拡張

runnerが安定した場合、次を段階的に検討する。

1. candidate qualification pressure
2. AI Builder / Human Review projection pressure
3. read-only Business Actor pressure
4. package全体のstable subset
5. Gate C後のactual second Application Reference

対象拡張はtest数ではなく、同じEvidence claimを共有するまとまりで行う。
