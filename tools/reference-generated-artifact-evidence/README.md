# reference-generated-artifact-evidence

Gate Bの`ISSUE-000025` / GitHub #255で、generated Reference artifactが**一つのclean repository revisionから再現できること**をcheck-onlyで確認するEvidence runnerです。

## 役割

このrunnerは修復toolではありません。

次の二つの正規golden checkを一つのrunで実行し、その前後でrepository sourceが変化していないことを確認します。

```bash
go run ./tools/reference-golden
go run ./tools/reference-durable-run-golden
```

確認対象:

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- 保存Durable RunのPlugin Lock依存
- `reference/order_rollout.durable-run.m0.21.start.json`全体
- 実行前後のrepository HEAD
- 実行前後のworking tree clean

## 実行

Repository rootのclean revisionで実行します。

```bash
go run ./tools/reference-generated-artifact-evidence
```

stdoutには小さいJSON Evidenceだけを出します。個別golden toolのstdout / stderrはstderr側へ流します。

概ね次の形です。

```json
{
  "evidenceNature": "generated-artifact-reproducibility",
  "repositoryRevision": "<git commit>",
  "workingTreeClean": true,
  "reproducibleRevision": true,
  "postRunRepository": {
    "checked": true,
    "repositoryRevision": "<same git commit>",
    "revisionStable": true,
    "workingTreeClean": true
  },
  "goVersion": "go version ...",
  "checks": [
    {
      "name": "world-plugin-lock-golden",
      "command": ["go", "run", "./tools/reference-golden"],
      "passed": true
    },
    {
      "name": "durable-run-golden",
      "command": ["go", "run", "./tools/reference-durable-run-golden"],
      "passed": true
    }
  ]
}
```

## fail-closed条件

Gate B完了Evidenceとして採用できるのは、次をすべて満たすrunだけです。

- 実行開始時にworking treeがclean
- `reference-golden`が一致
- `reference-durable-run-golden`が一致
- 実行後のrepository source確認が完了している（`postRunRepository.checked = true`）
- 実行前後でHEADが同一
- 実行後もworking treeがclean

いずれかを満たさない場合は非0終了します。

`reproducibleRevision`は「repository sourceがたまたま同じだった」という意味だけではありません。PR #398以降は、二つのgolden checkが両方成功し、かつpost-runのsource確認を完了した後にだけ`true`にします。

どちらかのgolden checkが失敗した場合、HEADとworking treeが同じclean revisionのままでも、その失敗JSONでは`reproducibleRevision = false`を維持します。失敗runをsource再現性だけからGate B完了Evidenceへ誤読しないためです。

このrunnerには`-allow-dirty`を設けません。dirty treeでの診断は個別golden toolを直接利用し、Gate B完了Evidenceと混同しないためです。

## staleだった場合

本runnerは**絶対に`-write`を呼びません**。

不一致を検出した場合は、個別toolの診断を確認します。修復が必要なものだけ、明示的に次を実行します。

```bash
go run ./tools/reference-golden -write
go run ./tools/reference-durable-run-golden -write
```

必要なtoolだけを使用します。

その後:

1. `git diff`で現在のCanonical authoring / fixed observed deployment stateから説明できる差分だけか確認する
2. 意図したgenerated artifactだけをcommitする
3. 新しいclean revisionへ固定する
4. 本runnerをcheck-onlyで再実行する

`-write`直後のdirty working treeでgoldenが一致したことだけを完了Evidenceにはしません。

## Evidenceの意味

`reproducibleRevision = true`で成功して証明するのは、記録された一つのclean repository revisionについて、二つの正規golden checkが成功し、保存generated artifactが正規生成結果と一致し、check実行中にもsource revisionが変化しなかったことです。

次の代替にはしません。

- #310 deterministic Application Driver / Evidence intake test
- #308 real PostgreSQL engine Evidence
- #214 actual Oracle asset Evidence
- production external state Evidence

## 保存時の注意

stdoutのJSONを保存する場合、Repository内の新規fileへ直接redirectするとshellがcommand開始前にfileを作り、開始時clean条件を壊します。

まずworking tree外へ保存します。

```bash
go run ./tools/reference-generated-artifact-evidence \
  > /tmp/tei-gate-b-generated-artifact-evidence.json
```

Evidence JSONのRepository内標準保存先を、このrunnerから先行決定しません。

## 関連

- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/issue/000025-reference-generated-artifacts-refresh.md`
- GitHub #255 / #273
- `tools/reference-golden/`
- `tools/reference-durable-run-golden/`
