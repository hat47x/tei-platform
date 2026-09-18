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

## 2. Portable runner / Workflow

portable entry point:

- `tools/run-knowledge-portability-reference-evidence.sh`

GitHub Actions wrapper:

- `.github/workflows/knowledge-portability-reference-evidence.yml`

local / compatible runnerでは次で同じcheckを実行できる。

```bash
bash tools/run-knowledge-portability-reference-evidence.sh
```

Evidence logを保存する場合:

```bash
TEI_REFERENCE_EVIDENCE_OUT=knowledge-portability-reference-test.txt \
  bash tools/run-knowledge-portability-reference-evidence.sh
```

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

## 4. Trigger / current runner availability

当初はpull request triggerを設定したが、PR #744 headで作成されたrun `35385856132` はattempt 1 / 2とも約数秒でfailureとなった。

raw job metadataでは両attemptとも、

```text
steps = []
runner_id = 0
runner_name = ""
labels = ["ubuntu-latest"]
```

であり、test stepは開始されていない。

このため、現時点ではtest failureとして扱わず、GitHub-hosted runner allocation未成立として扱う。

既存のRepository branch hygiene Workflowでも同種のrunner制約が記録されているため、不要なPR failureを常態化させないよう、本Workflowは一時的に`workflow_dispatch`のみとする。

```text
runner allocation unavailable
  != test failure
  != Executable Evidence PASS
```

successful hosted-runner allocationが一度得られた後、関連pathsへのpull_request trigger再有効化を検討する。

## 5. Evidence artifact

run結果を、

```text
knowledge-portability-reference-test.txt
```

としてartifactへ保存する。

成功時も失敗時もEvidence取得を試みる。

artifactはCanonical sourceではない。

## 6. 成功条件

- runnerが実際に割り当てられる
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

2026-09-19時点ではGitHub-hosted runner上のsuccessful runは未取得である。

したがって現状は、

```text
runner contract / script
  -> Source Evidence

GitHub Actions attempts 1 / 2
  -> runner allocation failure Evidence

Executable Evidence
  -> not acquired
```

と記録する。

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
