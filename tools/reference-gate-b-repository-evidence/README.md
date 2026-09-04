# reference-gate-b-repository-evidence

Gate Bのうち、Repository checkoutとGo toolchainだけで再現できる**deterministic Repository Evidence**を一つの入口から実行するtoolです。

GitHub Issue #310の実行補助を目的とします。

## 対象

このtoolは次だけを扱います。

1. repository revisionの取得
2. working treeがcleanであることの確認
3. Go toolchain identityの取得
4. `reference/workspace`のCanonical Validation
5. 最終UI-less Scenario test
6. external Evidence intake boundary test
7. 明示指定時だけRepository全体の`go test ./...`

次は扱いません。

- #214 actual Oracle asset Evidence
- #308 real PostgreSQL engine Evidence
- #255 World / Plugin Lock / Durable Runの更新
- GitHub Actionsの有効化
- external assetやgenerated artifactの書換え

## 既定実行

Repository rootで実行します。

```bash
go run ./tools/reference-gate-b-repository-evidence
```

既定では次のtargeted Evidenceを実行します。

```bash
go test ./internal/driver \
  -run '^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$' \
  -count=1

go test ./internal/database/externalqueryevidence \
  ./tools/reference-external-query-evidence \
  -count=1
```

test stdout / stderrは標準エラー側へ流し、最終的な小さいJSON Evidenceだけをstdoutへ出します。

## broad test

Repository全体も同じrunで確認する場合だけ`-broad`を指定します。

```bash
go run ./tools/reference-gate-b-repository-evidence -broad
```

これによりtargeted Evidence成功後に次を追加します。

```bash
go test ./...
```

広いtestはtargeted Gate B Evidenceとは別checkとしてJSONへ記録します。

## clean revisionを要求する理由

Gate B完了Evidenceは、後から同じsource revisionへ戻れる必要があります。

そのため既定では次が空でない場合、test実行前にfail closedで停止します。

```bash
git status --porcelain
```

working treeがdirtyな状態で`git rev-parse HEAD`だけを記録すると、実際に実行したsourceと記録revisionが一致しません。

診断目的に限り`-allow-dirty`を指定できます。

```bash
go run ./tools/reference-gate-b-repository-evidence -allow-dirty
```

この場合はtest自体を実行できますが、JSONでは`reproducibleRevision = false`となり、最後に非0終了します。**Gate B完了Evidenceには使用できません。**

## Workspace Validation

DB接続やtestの前提として、`reference/workspace`を通常Loaderで読み、`validation.Validate`を実行します。

Validation Errorが一件でもある場合、targeted testへ進みません。

Warningだけを新しい拒否条件にはしません。

## JSON Evidence

stdoutには概ね次の形を出します。

```json
{
  "evidenceNature": "deterministic-repository",
  "repositoryRevision": "<git commit>",
  "workingTreeClean": true,
  "reproducibleRevision": true,
  "goVersion": "go version ...",
  "workspace": "reference/workspace",
  "workspaceValidation": {
    "passed": true
  },
  "checks": [
    {
      "name": "final-ui-less-scenario",
      "command": ["go", "test", "./internal/driver", "..."],
      "passed": true
    },
    {
      "name": "external-evidence-intake",
      "command": ["go", "test", "..."],
      "passed": true
    }
  ],
  "broadRequested": false
}
```

JSONへtest stdout、DSN、credential、runtime parameter valueは保存しません。

### JSONをファイルへ保存する場合

shellの`>`はコマンド開始前にredirect先を作成します。Repository内の新規pathへ直接redirectすると、そのfile自身を`git status --porcelain`が検出してdirtyとなり、意図どおりfail closedします。

保存する場合は、まずworking tree外へ出します。

```bash
go run ./tools/reference-gate-b-repository-evidence \
  > /tmp/tei-gate-b-repository-evidence.json
```

EvidenceをRepositoryへ永続保存するか、どのpathを標準とするかはこのtoolから先行決定しません。

## Evidenceの意味

成功しても証明するのは、そのclean repository revisionでdeterministicなGate B境界が通常Go runnerから成立したことだけです。

次へ格上げしません。

- actual Oracle physical Evidence
- real PostgreSQL provider Evidence
- production external state Evidence
- Reference generated artifactが最新であるというEvidence

生成物再現性は`tools/reference-golden`、PostgreSQL provider Evidenceは`tools/reference-postgres-header-detail-runtime`、actual Oracle Evidenceは`tools/reference-external-query-evidence`を含む#214の手順で別々に扱います。

## 関連

- GitHub Issue #310
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
