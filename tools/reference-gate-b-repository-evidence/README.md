# reference-gate-b-repository-evidence

Gate Bのうち、Repository checkoutとGo toolchainだけで再現できる**deterministic Repository Evidence**を一つの入口から実行するtoolです。

GitHub Issue #310 / `plan/issue/000032-gate-b-deterministic-repository-evidence.md`の実行補助を目的とします。

## 対象

このtoolは次だけを扱います。

1. repository revisionの取得
2. working treeがcleanであることの確認
3. Go toolchain identityの取得
4. `reference/workspace`のCanonical Validation
5. 最終UI-less Scenario test
6. external Evidence intake boundary test
7. Repository source固定境界のpure Conformance test
8. 明示指定時だけRepository全体の`go test ./...`
9. 各check直後と最終時点でのHEAD不変・working tree状態の再確認

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

go test ./tools/internal/repositoryevidence \
  -count=1
```

最後のcheckは、PR #336で#310 / #255 / #308へ共通化したRepository source観測について、開始時clean、post-run確認済み、HEAD同一、終了時cleanをすべて要求するpure Conformanceを`-broad`なしでも確認するものです。

runner自身のpackageはtargeted `go test`へ含めません。将来runner testが`run()`を呼ぶようになった場合の自己再帰余地を作らず、共有helperの契約だけを独立して検証します。

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

## cleanかつ固定されたrevisionを要求する理由

Gate B完了Evidenceは、後から**実際に実行した同じsource revision**へ戻れる必要があります。

そのため実行前に次が空であることを確認します。

```bash
git status --porcelain
```

working treeがdirtyな状態で`git rev-parse HEAD`だけを記録すると、実際に実行したsourceと記録revisionが一致しません。

さらに、PR #412以降はtargeted test / broad testをすべて終えてからだけ確認するのではなく、**各checkの直後にも**次を再確認します。

```bash
git rev-parse HEAD
git status --porcelain
```

開始時cleanな通常Evidence runでは、各check直後に次のいずれかを検出した時点で、後続checkへ進まずfail closedします。

- repository sourceの観測が完了していない
- HEADが実行開始時のcandidate revisionから変化した
- check実行によってworking treeに新しい差分が残った

これにより、testや並行操作がsourceへ副作用を残した状態で次のEvidence checkを続けず、そのrunを元revisionの成功Evidenceとして誤採用しません。

すべてのcheckが終わった後にも最終のpost-run確認を行います。途中確認と最終確認の両方を通過して初めて、run全体を一つのclean revisionへ固定できます。

開始時にworking treeがcleanでも、それだけでは`reproducibleRevision = true`にしません。PR #400以降、`reproducibleRevision`は次をすべて確認した後にだけtrueになります。

- Workspace Validationが成功している
- 三つのtargeted checkがすべて成功している
- `-broad`指定時はRepository全体のtestも成功している
- 各check直後のsource確認を通過している
- `postRunRepository.checked = true`
- 実行前後でHEADが同一
- 実行後もworking treeがclean

したがってtargeted checkまたはbroad checkが失敗した場合、Repository sourceが同じclean revisionのままでも、その失敗JSONでは`reproducibleRevision = false`を維持します。Go toolchain取得、Workspace Load / Validationなどでpost-run確認前に失敗したJSONでもfalseのままです。

診断目的に限り`-allow-dirty`を指定できます。

```bash
go run ./tools/reference-gate-b-repository-evidence -allow-dirty
```

`-allow-dirty`は、**開始時から存在するdirty working treeを診断目的で許容するためのオプション**です。PR #412以降も、HEADが変化した場合やrepository sourceを観測できない場合は途中で停止します。開始時からdirtyであることだけを理由に各check間で停止はしませんが、このrunは最後まで成功しても`reproducibleRevision = false`となり、非0終了します。**Gate B完了Evidenceには使用できません。**

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
  "postRunRepository": {
    "checked": true,
    "repositoryRevision": "<same git commit>",
    "revisionStable": true,
    "workingTreeClean": true
  },
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
    },
    {
      "name": "repository-evidence-source-boundary",
      "command": ["go", "test", "./tools/internal/repositoryevidence", "-count=1"],
      "passed": true
    }
  ],
  "broadRequested": false
}
```

`reproducibleRevision = true`になるには、少なくとも次がすべて必要です。

- Workspace Validationが成功
- targeted checkがすべて成功
- `-broad`指定時はbroad checkも成功
- 実行前working treeがclean
- 各check直後のrepository source確認を通過
- 実行後repository状態を確認済み
- 実行前後でHEADが同一
- 実行後working treeもclean

check失敗やsource確認未完了を含め、これらを最後まで満たせないrunでは`reproducibleRevision`はfalseです。

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

`reproducibleRevision = true`で成功して証明するのは、そのcleanかつ実行中に不変だったrepository revisionで、Workspace Validationと必要なdeterministic Gate B checksが通常Go runnerから成立したことです。

次へ格上げしません。

- actual Oracle physical Evidence
- real PostgreSQL provider Evidence
- production external state Evidence
- Reference generated artifactが最新であるというEvidence

生成物再現性は`tools/reference-golden`、PostgreSQL provider Evidenceは`tools/reference-postgres-header-detail-runtime`、actual Oracle Evidenceは`tools/reference-external-query-evidence`を含む#214の手順で別々に扱います。

## 関連

- GitHub Issue #310
- `plan/issue/000032-gate-b-deterministic-repository-evidence.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
