# Gate B Readiness Artifact Audit

`tools/reference-gate-b-readiness-artifact-audit` は、#387 final convergence 成功後に、`GATE_B_READINESS_AUDIT.md` / #273 が要求する same-run child Evidence の独立再監査を補助する read-only tool です。

この tool 自体は Gate B Ready を宣言しません。actual Repository checkout、PostgreSQL、secret transport、external-observed の再実行もしません。

## Input

#387 final summary と、その run が保持した Repository 外 child artifact directory を指定します。

```bash
go run ./tools/reference-gate-b-readiness-artifact-audit \
  -final /tmp/tei-gate-b-R/final.json \
  -artifact-dir /tmp/tei-gate-b-R/children
```

入力 artifact は regular non-symlink file を要求します。child directory 自体も symlink directory を受理しません。

## 独立再監査するもの

### final summary

- `evidenceNature = gate-b-three-lane-final-convergence`
- `repositoryRevision = R`
- `workingTreeClean = true`
- `reproducibleRevision = true`
- `allLanesSameRevision = true`
- `childArtifactsRetained = true`
- post-run source が同じ clean `R`
- Go environment の fail-closed boolean
- 3 lane の name / nature / `R` / passed / reproducible
- fixed `artifactFile`
- `evidenceArtifactFingerprint`

### exact child bytes

fixed filename:

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

を再読込し、exact bytes の SHA-256 が final summary の fingerprint と一致することを確認します。

### #310

- Workspace Validation success
- final UI-less Scenario success
- external Evidence intake success
- Repository source-boundary Conformance success
- `-broad` final run の場合だけ repository-wide Go test success も要求
- post-run same clean `R`

### #255

- World / Plugin Lock golden success
- Durable Run golden success
- post-run same clean `R`

### #308

- preexisting fixture protection
- tool-owned fixture
- dedicated non-default empty Database guard
- fixture 前 user relation 0 / Event Trigger 0 / fixture absent
- 2 rows / normal 0 rows / missing-table Execution Error
- explicit Semantic mapping
- support column `order_id` 非露出
- parameter value 非保存
- cleanup verified
- cleanup 後 user relation 0 / Event Trigger 0 / fixture absent
- actual Database identity 非保存
- post-run same clean `R`

通常 Evidence に保存してはいけない `dsn` / credential / password / parameter values / actual Database name / secret path / handoff path 等の field key も fail closed で拒否します。

## Output

成功時は、監査した final bytes の fingerprint と3 child の fingerprintを含む補助 JSON を stdout へ返します。

```text
evidenceNature = gate-b-readiness-artifact-audit
artifactAuditPassed = true
repositoryRevision = R
```

これは第四 Required Evidence lane ではなく、#273 の independent artifact audit を再現可能にする補助出力です。

`artifactAuditPassed=true` だけで Gate B Ready にはしません。`GATE_B_READINESS_AUDIT.md` の全条件を R について別途再判定し、Ready の場合だけ #291 Handover Baseline `B / R` へ進みます。
