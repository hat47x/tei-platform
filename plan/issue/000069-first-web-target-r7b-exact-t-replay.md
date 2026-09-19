# ISSUE-000069: First Web TargetのR7B exact-T replayとrepeatability Evidenceを取得する

- 状態: In Progress
- 提案日: 2026-09-20
- 性格: Replayable Behavior Evidence / operational verification / Reference execution
- 主担当要求: RQ-010 / RQ-021 / RQ-022 / RQ-023 / RQ-025
- 関連: ISSUE-000007 / ISSUE-000028 / ISSUE-000053 / ISSUE-000057

## 背景

R7A handoff、R7B execution intake、R7B replay planまではSourceとして実装され、第一Web TargetのGate C origin EvidenceもGitHub Release `gate-c-evidence-T-0ba87a4` に保管されている。

現在まだ確認できていないのは、同じTarget build `T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe` に対して、計画した6 behavior unitを別々の実行runで再現し、transport boundaryとsemantic Observation classが安定していることをactual Evidenceとして示せるかである。

一度の成功を二度コピーしたもの、current mainでの再実行、response byteの一致だけではrepeatability Evidenceにしない。

## 防ぎたい失敗

- current mainの動作を、origin Gate C Target `T` のrepeatabilityへ読み替える。
- 同じserver processの結果を複製して「2 run」と扱う。
- HTTP 200だけを見てAuthentication / Authorization / Validation / Command Execution Errorの意味境界を確認しない。
- Gate C auditorとR7B auditorで別のsemantic truthを実装し、判定がdriftする。
- request credentialをreplay planや保存artifactへ残す。
- raw responseが存在するだけで、exact SHA-256 identityなしにEvidenceとして採用する。
- behavior-unit repeatabilityをwhole-workflow coverageへ昇格させる。

## 減らす不確実性

第一Web Targetについて、同一の明示Target buildと同一のbehavior identityを使えば、別process・別runでも次が保たれるかを確認する。

- expected HTTP boundary
- expected semantic Observation class
- B / R / T provenance
- raw artifactの追跡可能性
- credential非保持
- run間の独立性

response byteの完全一致は要求しない。意味とtransport boundaryが安定していることを確認する。

## 入力境界

実行時にはRepository外の次のactual artifactを要求する。

1. `r7b-replay-plan.json`
2. `r7b-replay-plan-receipt.json`
3. adopted Gate C Evidence archive `tei-gate-c-evidence-T-0ba87a4e.tar.gz`
4. 新規のRepository外output directory

Gate C archiveは次を両方通したものだけを採用する。

- archive SHA-256 = `de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3`
- 展開後の `MANIFEST.sha256` 全件一致

replay plan receiptはexact replay-plan bytesへbindingされ、実行前には `replayExecuted=false` / `repeatabilityObserved=false` でなければならない。

## Packet M actual materialization runbook

Packet MはRepository sourceを変更しない。authenticated GitHub accessとGo-enabled clean full checkoutがある実行環境で、採用済みGate C Evidenceからactual R7B replay planまでをRepository外にmaterializeする。

前提:

```text
current checkout = clean main
gh auth          = private Release asset download可能
Go               = repository runner実行可能
output root      = Repository外の新規directory
```

current adopted origin:

```text
Release tag:
  gate-c-evidence-T-0ba87a4

asset:
  tei-gate-c-evidence-T-0ba87a4e.tar.gz

archive SHA-256:
  de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3
```

### M1. adopted Gate C archiveを再取得する

```bash
set -euo pipefail

root="$(mktemp -d)"
archive="$root/tei-gate-c-evidence-T-0ba87a4e.tar.gz"

gh release download gate-c-evidence-T-0ba87a4 \
  --pattern 'tei-gate-c-evidence-T-0ba87a4e.tar.gz' \
  --dir "$root"

printf '%s  %s\n' \
  de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3 \
  "$archive" | sha256sum -c -

tar -xzf "$archive" -C "$root"

origin="$root/gate-c-T-0ba87a4e"
(
  cd "$origin"
  sha256sum -c MANIFEST.sha256
)
```

archive digestまたはmanifestが1件でも不一致なら以後へ進まない。

### M2. actual R7A handoffをmaterializeする

```bash
TEI_R7A_GATE_C_READINESS="$origin/gate-c-readiness-audit.json" \
TEI_R7A_SEMANTIC_FUNCTIONAL_SUMMARY="$origin/semantic-functional-summary.json" \
TEI_R7A_OUTPUT_DIR="$root/r7a" \
bash tools/run-first-web-target-r7a-handoff.sh
```

期待出力:

```text
$r7a/
  r7a-replayable-behavior-handoff.json
  r7a-handoff-receipt.json
  source-tests.log
```

この時点では:

```text
R7A handoff acquired
  != replay executed
  != repeatability observed
```

### M3. actual R7B execution intakeをmaterializeする

```bash
TEI_R7B_R7A_HANDOFF="$root/r7a/r7a-replayable-behavior-handoff.json" \
TEI_R7B_R7A_RECEIPT="$root/r7a/r7a-handoff-receipt.json" \
TEI_R7B_OUTPUT_DIR="$root/r7b-intake" \
bash tools/run-first-web-target-r7b-intake.sh
```

期待出力:

```text
$r7b-intake/
  r7b-execution-intake.json
  r7b-intake-receipt.json
  source-tests.log
```

intake / receiptはexact R7A source identityへbindingされ、依然:

```text
ReplayExecutionStatus = not-executed
RepeatabilityStatus   = not-observed
```

でなければならない。

### M4. actual R7B replay planをmaterializeする

**前提: E0 replay-plan launch provenance carryがhealth-check済みでmainへ入っていること。**

current replay-plan sourceが `HandoverBaselineID / BaselineRepositoryRevision / TargetAdapter / Host` をまだcarryしていないrevisionではM4をfinal Evidenceとして実行しない。

M1〜M3をE0前に予備実行した場合、そのartifactは入力検証には使えるが、final Packet Mとして採用するreplay plan / receiptはE0統合後のclean mainから再生成する。

```bash
TEI_R7B_EXECUTION_INTAKE="$root/r7b-intake/r7b-execution-intake.json" \
TEI_R7B_INTAKE_RECEIPT="$root/r7b-intake/r7b-intake-receipt.json" \
TEI_R7B_REPLAY_PLAN_OUTPUT_DIR="$root/r7b-plan" \
bash tools/run-first-web-target-r7b-replay-plan.sh
```

期待出力:

```text
$r7b-plan/
  r7b-replay-plan.json
  r7b-replay-plan-receipt.json
  source-tests.log
```

replay planはorigin TをReplayTargetBuildRevisionとして保持し、6 behaviorをexact setで持つ。

### M5. Packet M completion boundary

Packet M完了として記録できるのは、少なくとも次がactual execution resultとして確認できた場合だけである。

- Release archive exact SHA-256 PASS
- `MANIFEST.sha256` 全件PASS
- R7A handoff runner PASS
- R7B intake runner PASS
- E0 launch provenance carryがhealth-check済みmainに統合済み
- R7B replay-plan runner PASS
- replay planがB / R / T / TargetAdapter / Hostをlossless carry
- 各段階のreceipt exact SHA-256取得
- current source checkoutが各runner前後でclean / same revision
- derived artifactがすべてRepository外

Packet Mが成功しても:

```text
actual replay plan acquired
  != replay executed
  != replay passed
  != repeatability observed
```

である。

Packet Mのartifact rootは、後続Packet Eへpathだけでなくexact digest群としてhandoffする。filenameだけ一致する別artifactへの差し替えを許さない。

## 実装単位

### 1. semantic validatorを共有化する

既存 `tools/reference-first-web-target-evidence-audit` が持つ以下の判定を、Gate CとR7Bから共用できる小さいhelperへ分離する。

- read result
- command success
- authentication denied
- authorization denied
- validation not-satisfied
- command invocation execution error
- B / R / T provenance

R7B用に同じ条件を再記述しない。

この共有化は「Gate C Ready判定」を共有するものではない。共有対象はraw responseのsemantic classificationだけとする。

#### 1.1 current Gate C validator extraction map

current source `tools/reference-first-web-target-evidence-audit/main.go` では、R7Bが再利用すべき判定が一つのAPIとしてまとまっていない。

現状の対応は次である。

| R7B behavior | current Gate C判定 |
| --- | --- |
| `first-web/read-only` | `read.json` のexact provenance、Scenario ID / Application ID、`requireRuntimeBindings` |
| `first-web/command-success` | exact provenance + `requireSuccessfulCommand` |
| `first-web/authentication-denied` | exact provenance + `authentication-denied` stage / denied Authentication / Command・Observation不在のinline判定 |
| `first-web/authorization-denied` | exact provenance + `authorization-denied` stage / denied Authorization / Command・Observation不在のinline判定 |
| `first-web/validation-not-satisfied` | exact provenance + quantity-positive Validation 1件 / not-satisfied / Authorization・Command・Observation不在のinline判定 |
| `first-web/execution-error` | exact provenance + command stage / Authorization allowed / Command・Observation不在 / synthetic adapter errorのinline判定 |

credential非保持は別途 `requireNoReferenceCredentialMaterial` が担っている。これはsemantic classificationではないが、R7B raw artifact retentionでも同じReference secret setをrejectする必要がある。

初回共有化は判定意味を変更せず、**code motionを先に行う**。

候補配置:

```text
tools/internal/firstwebtargetevidence
```

このpackageはFirst Web TargetのEvidence toolからだけ利用するtool-local helperとし、`internal/core` やCanonical modelへ昇格させない。

最小APIはgenericなUniversal Event classifierではなく、behaviorごとの明示関数にする。

概念形:

```go
ValidateReadArtifact(bytes, expectedProvenance)
ValidateCommandSuccessArtifact(bytes, expectedProvenance)
ValidateAuthenticationDeniedArtifact(bytes, expectedProvenance)
ValidateAuthorizationDeniedArtifact(bytes, expectedProvenance)
ValidateValidationNotSatisfiedArtifact(bytes, expectedProvenance)
ValidateExecutionErrorArtifact(bytes, expectedProvenance)
RejectReferenceCredentialMaterial(bytes)
```

`Validate(kind, payload)` のような汎用dispatcherを初回から作らない。6 behaviorの存在を新しいCanonical taxonomyへ見せないためである。

read validatorだけはGate C auditorが検証後の `RuntimeBindings` を `TargetDifferenceHandoff` の生成にも使っているため、単なる `error` だけでは不足する。したがってreadについては、検証済みのruntime binding projectionを返せる薄い結果型を許す。

```text
raw read JSON
  -> shared validation
  -> validated runtime bindings
     -> Gate C TargetDifferenceHandoff

raw replay JSON
  -> same shared validation
  -> R7B semantic observation
```

一方、command系5件はcurrent Gate C auditorが判定後のdecoded payloadを別のsummary意味へ再利用していないため、初回はvalidation resultだけを返せばよい。

#### 1.2 extraction invariants

共有化commitでは次を同時に変えない。

- Gate C required artifact set
- B / R / T exact provenance equality
- runtime bindingの期待値
- successful commandのAuthentication / Validation / Authorization / Command / Observation条件
- denial時の後続Command / Observation不在条件
- Validation not-satisfied条件
- Execution Errorのcurrent synthetic failure条件
- credential reject set
- R7Bのexpected HTTP status
- R7Bのexpected semantic Observation class
- CoverageClass / DeepWorkflowCovered

特に、Execution Errorのerror文字列条件が将来過度にfixture-specificだと判断されても、validator共有化と同じcommitで緩和しない。まずGate C / R7Bが同じ判定を呼ぶ状態を作り、その後に意味変更を別Evidenceで検討する。

共有化完了の最低検証:

```text
existing Gate C evidence-audit tests PASS
+ new shared-validator focused tests PASS
+ R7B replay-audit focused tests PASS
+ repository-health-check PASS
```

Gate C auditorのsummary shapeと既存判定条件を保ったままshared helperへ移せない場合は、R7B側へコピーして先へ進まず停止する。

#### 1.3 V1 test migration matrix

current `tools/reference-first-web-target-evidence-audit/main_test.go` は次の3 integration testを持つ。

- Semantic / Functional happy path
- Target revision mismatch reject
- retained raw artifactへのcredential material混入reject

この3本はshared classifier extraction後もGate C auditor側に残す。これらは「shared helper単体」ではなく、Gate C audit全体のsource qualification / summary / Target Difference Handoff / credential scanまで含むintegration boundaryを検証しているためである。

一方、current happy-path fixtureは6 behaviorを一度にvalidとして作るだけで、各semantic boundaryのnegative caseを独立には固定していない。V1ではshared package側に少なくとも次のtable-driven pressureを追加する。

| validator | minimum negative pressure |
| --- | --- |
| provenance | B / R / T / Target Adapter / Hostのいずれか1つが違えばreject |
| read | wrong Scenario ID、wrong Application ID、browse binding drift、customer binding drift、order-lines binding driftをreject |
| command success | wrong stage、Authentication欠落/非authenticated、Validation count/result drift、Authorization非allowed、Command capability/mode drift、Observation欠落/identity/nature/result driftをreject |
| authentication denied | wrong stage、Authentication欠落/非denied、CommandまたはObservationが存在すればreject |
| authorization denied | wrong stage、Authorization欠落/非denied、CommandまたはObservationが存在すればreject |
| validation not-satisfied | wrong stage、Validation件数/identity/result drift、Authorization / Command / Observationが存在すればreject |
| execution error | wrong stage、Authorization欠落/非allowed、Command / Observationが存在、current synthetic execution-error conditionを満たさなければreject |
| credential scan | editor / viewer / locked / disabled secret、およびlegacy credential-ref materialをreject |

V1の順序:

```text
shared DTO / validator testsを先に追加
  ↓
current Gate C predicatesをshared packageへ移動
  ↓
Gate C auditorをshared validator callへ置換
  ↓
existing Gate C integration testsを維持
  ↓
full repository health check
```

先にproduction-like code motionを行い、後からtestを合わせる順序にしない。

また `writeAuditFixture` 相当のtest fixtureはGate C integration test用として維持できるが、shared validator unit testは巨大なartifact directory全体を要求しない。各behaviorのraw JSON bytesを最小fixtureとして直接検証し、failure原因をbehavior boundaryへ局所化する。

V1ではerror message文字列そのものをpublic contractにしない。判定PASS/FAILと、どのbehavior / provenance boundaryがrejectされたかをtestする。

### 2. R7B replay audit sourceを追加する

新しいaudit sourceは、各runの6 raw responseを読み、replay planと照合して少なくとも次を記録する。

- behavior identity
- replay run identity
- replay Target build revision
- expected / observed HTTP status
- expected semantic Observation class
- semantic validation result
- raw artifact filename
- raw artifact SHA-256

run identityの重複、behavior欠落、Target revision不一致、digest不正はfail closedとする。

### 3. operational replay runnerを追加する

runnerはcurrent mainのworktreeをcheckoutし直さず、Repository外のisolated worktreeまたは同等のclean checkoutをexact `T` で作る。

各runは次の順で独立に実行する。

```text
fresh Target process
  -> health
  -> six behavior units
  -> raw artifact保存
  -> Target停止
  -> run audit
```

これを最低2 run行う。run 1とrun 2は別process、別run identity、別raw artifact directoryを持つ。

run identityは同一execution内で異なるだけでなく、同じrunner revision / 同じTarget Tを後日再実行した場合にも不用意に衝突しないようにする。固定のTとrunner source revisionだけからdefault run IDを作らず、per-acquisition nonce / UTC timestamp等の非semantic discriminatorを含め、採用Evidence側には生成後のexact identityを記録する。

#### 3.0 E0 — replay plan launch provenanceをlosslessにする

operational runnerの具体化で、current `r7b-replay-plan.json` がTarget起動に必要なprovenanceを完全にはcarryしていないことが分かった。

execution intakeには既に:

```text
HandoverBaselineID
BaselineRepositoryRevision
TargetBuildRevision
TargetAdapter
Host
```

が存在し、current Gate C既知値とのexact consistencyは `validateIntake` がfail closedで確認している。

E0は新しい意味やauthorityを追加せず、そのvalidated launch provenanceをreplay planへlossless carryする。

```text
recorded Gate C constants
    ↓ validateIntake: drift reject
validated execution intake
    ↓ buildReplayPlan: lossless carry
replay plan B / R / T / TargetAdapter / Host
```

##### 3.0.1 exact source delta

current `replayPlan` top-levelへ次を追加する。

```go
HandoverBaselineID         string `json:"handoverBaselineId"`
BaselineRepositoryRevision string `json:"baselineRepositoryRevision"`
TargetAdapter              string `json:"targetAdapter"`
Host                       string `json:"host"`
```

既存:

```go
ReplayTargetBuildRevision string `json:"replayTargetBuildRevision"`
```

と合わせ、launch provenanceのsingle sourceをplan top-levelへ置く。

`buildReplayPlan` はaccepted `executionIntake` からexact carryする。

```text
plan.HandoverBaselineID
  = intake.HandoverBaselineID

plan.BaselineRepositoryRevision
  = intake.BaselineRepositoryRevision

plan.ReplayTargetBuildRevision
  = intake.TargetBuildRevision

plan.TargetAdapter
  = intake.TargetAdapter

plan.Host
  = intake.Host
```

B/R/Adapter/Hostを各 `replayUnit` へ複製しない。behavior unitはbehavior-local recipe、origin raw artifact、Target revision境界だけを保持する。

current `validateIntake` のknown Gate C exact-value checkはE0で変更・除去しない。定数はinput drift rejectを担い、plan outputの別authorityにはしない。

runner / replay audit / shared semantic validatorは、E0後のplan provenanceをruntime expected valueとして使用する。runnerがB/R/adapter/hostをcompile-time固定値だけから再構成してplanを無視する実装は禁止する。

##### 3.0.2 test pressure

既存happy pathで少なくとも次を確認する。

```text
got.HandoverBaselineID
  == intake.HandoverBaselineID

got.BaselineRepositoryRevision
  == intake.BaselineRepositoryRevision

got.ReplayTargetBuildRevision
  == intake.TargetBuildRevision

got.TargetAdapter
  == intake.TargetAdapter

got.Host
  == intake.Host
```

JSON serialization後も5 fieldがexact field name / exact valueで存在することを確認する。

`TestBuildReplayPlanRejectsGateCOrExecutionStateDrift` 相当へ少なくとも次の独立subtestを追加する。

- HandoverBaselineID empty / drift
- BaselineRepositoryRevision empty / drift
- TargetBuildRevision empty / drift
- TargetAdapter empty / drift
- Host empty / drift

candidate側B/R/T provenanceとtop-level intakeの不一致をrejectする既存fail-closedも維持する。

既存のcredential / selector / coordinate / execution authority field禁止testを維持する。

生成後planの改ざん・field欠落rejectはbuilderの責務ではなく、V2 replay audit / E1 runnerのinput validationでPressureする。

##### 3.0.3 non-goals

E0と同じincrementで次を行わない。

- expected Gate C constantの変更
- behavior setの変更
- fixture mappingの変更
- expected HTTP statusの変更
- Observation classの変更
- cross-revision compatibility追加
- current Authorization / production execution authorityの導出
- B/R/Adapter/Hostのunit-level複製

##### 3.0.3 E0 branchless candidate preflight

direct-to-main規則とfull health-check gateを両立するため、E0 sourceをremote branchへ置かず、`main` refを動かさないGit object candidateとして準備した。

検証対象source blob:

```text
tools/reference-first-web-target-r7b-replay-plan/main.go
  git blob = 40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab

tools/reference-first-web-target-r7b-replay-plan/main_test.go
  git blob = 704fa68924fb647c59f0ae558a1e41d8ab9f6864
```

この2 blobについてRepository外の独立Go packageへ同一bytesをmaterializeし、`git hash-object` でGitHub blob identityと一致することを確認した上で次のfocused preflightを実行した。

```text
gofmt -l .             -> empty
go vet ./...           -> PASS
go test -count=50 ./... -> PASS
```

このpreflightはE0の局所実装健全性を確認するだけで、Repository全体のintegration gateではない。

```text
focused candidate PASS
  != repository-health-check PASS
  != E0 integrated into main
```

E0を`main`へ入れるsessionは、上記blob内容または同値差分を最新`main`へ適用し、**そのexact candidateを含むfull checkout**で `tools/repository-health-check.sh` を通す。mainが進んで対象2ファイルが変更された場合はblobを機械的に適用せず、current sourceへreconcileしてからhealth checkする。

##### 3.0.4 integration gate

E0は `tools/reference-first-web-target-r7b-replay-plan` の小変更であり、Packet V shared classifierとはfile境界が分かれる。

Go-enabled full checkoutで変更後に必ず:

```bash
tools/repository-health-check.sh
```

を実行する。

このhealth checkは実際に:

```text
gofmt -l .
go vet ./...
go test -count=1 ./...
go run ./cmd/tei validate -workspace reference/workspace
```

を要求する。

focused replay-plan testだけのPASSではmain統合条件を満たさない。health checkが失敗した場合はE0 codeをpushせず、原因を修正してから再実行する。

E0完了判定は、full health check PASS対象のexact commitがそのまま `main` へ入ったことまでを含む。

#### 3.1 Gate C fixture replay map

初回R7B operational replayのrequest fixtureは、existing `tools/run-first-web-target-gate-c-evidence.sh` がactual Gate C Evidence取得に使ったJSON Scenarioだけを再利用する。Custom UI / HTML captureはISSUE-000069の対象外とする。

| behavior | Target-local request | synthetic runtime input | expected HTTP |
| --- | --- | --- | ---: |
| `first-web/read-only` | `GET /scenario/read` | `from=2026-09-18T08:30:00Z, row=0` | 200 |
| `first-web/command-success` | `POST /scenario/submit` | editor credential / `from=08:30, row=0` | 200 |
| `first-web/authentication-denied` | `POST /scenario/submit` | editor identity + wrong credential / `from=08:30, row=0` | 200 |
| `first-web/authorization-denied` | `POST /scenario/submit` | viewer credential / `from=08:30, row=0` | 200 |
| `first-web/validation-not-satisfied` | `POST /scenario/submit` | editor credential / `from=08:00, row=1` | 200 |
| `first-web/execution-error` | `POST /scenario/submit` | editor credential / `from=07:00, row=2` | 422 |

時刻はすべて `2026-09-18T..:..:00Z` で、exact JSON / query encodingはexisting Gate C runnerを実装上の参照元とする。

この表のcredentialはReference synthetic runtime inputであり、replay plan / observation / retained raw response / receiptのidentityへ入れない。

```text
fixture resolution
  = Target-local operational input

credential value
  != replay-plan identity
  != semantic Observation identity
  != retained Evidence field
```

初回runnerではfixture解決を別の新しい設定schemaへ一般化しない。existing Gate C request recipeとR7B replay planの6 fixture identityをexactに対応付け、未知fixtureはfail closedする。

#### 3.2 current-main orchestrator / exact-T Target isolation

operational replay runnerはcurrent `main`側のtoolとして動くが、Target processはplanに固定されたorigin Tから起動する。

最小構造:

```text
current main checkout
  owns:
    replay runner
    shared semantic validator
    R7B replay audit / repeatability assessment

Repository-external detached worktree at exact T
  owns:
    reference-first-web-target source
    reference/workspace
    legacy-auth fixture source
    live Target process
```

Target worktreeはconceptually:

```bash
git worktree add --detach "$target_checkout" \
  0ba87a4e0a96f70a198b03ff7652fb00e3826abe
```

で作り、開始前に少なくとも:

- `git -C "$target_checkout" rev-parse HEAD` がplanのReplayTargetBuildRevisionとexact一致
- worktreeがclean
- non-standard tracked index stateなし
- `reference/workspace`
- `reference/legacy-auth/users.json`
- `reference/legacy-auth/authorization.json`
- `tools/reference-first-web-target`

の存在を確認する。

Target起動引数はGate C actual acquisitionと同じ意味境界を使う。

```text
-workspace reference/workspace
-application order_entry
-baseline-id B
-baseline-revision e5231839a5dcc62cce1181a2731fcb7c70d16736
-target-revision 0ba87a4e0a96f70a198b03ff7652fb00e3826abe
-legacy-users reference/legacy-auth/users.json
-legacy-authorization reference/legacy-auth/authorization.json
-listen 127.0.0.1:<ephemeral-or-explicit-port>
```

runner source revisionとTarget build revisionは別identityとしてreceiptへ残す。

```text
replayRunnerRepositoryRevision
  != replayTargetBuildRevision
```

current mainがTより先へ進んでいても、その差をTargetへ混ぜない。

#### 3.3 fresh-process run boundary

run 1 / run 2は同じdetached T worktreeを使ってよいが、Target processは必ず停止して新規起動する。

各run:

1. fresh loopback Target processを起動
2. `/healthz` 成功を確認
3. 6 fixtureを一度ずつ実行
4. raw response bytesとobserved HTTP statusを保存
5. shared semantic validatorで6件を分類
6. credential material scan
7. `run-observation.json` を確定
8. Target processを停止してwait
9. 次runへ進む

同じprocessで12 requestを流してrun IDだけ分ける実装はrejectする。

port番号はEvidence identityにしない。runごとに利用可能なloopback portを選べるが、Target bindがloopbackであることはobservationへ残す。

### 4. repeatability summaryを生成する

各behaviorは、2 run以上について次がすべて成立した場合のみrepeatableとする。

- same explicit replay Target build
- distinct run identities
- expected HTTP boundary一致
- expected semantic Observation class一致
- raw artifactごとの有効なSHA-256 identity
- credential material非保持

exact response bytesは一致してもよいし異なってもよい。byte equalityは必要条件でも十分条件でもない。

最終summaryは次の境界を保持する。

```text
CoverageClass = behavior-unit
DeepWorkflowCovered = false
```


#### 4.1 repeatability aggregation contract

Packet Eの集約器は、raw responseそのものを再分類する責務を持たない。各runのobservationはPacket Vのshared validatorを通過した結果として受け取り、集約器はplanとの一致とrun間の独立性だけを判定する。

入力の概念形:

```text
ReplayPlan
  - ReplayTargetBuildRevision
  - Unit[]
      - BehaviorID
      - ExpectedHTTPStatus
      - ExpectedObservationClass
      - RequiredDistinctRuns

Run[]
  - RunIdentity
  - ReplayTargetBuildRevision
  - Observation[]
      - BehaviorID
      - RunIdentity
      - ReplayTargetBuildRevision
      - ObservedHTTPStatus
      - ObservationClass
      - SemanticValidationPassed
      - RawArtifactFile
      - RawArtifactSHA256
```

集約時は次を要求する。

1. plan内のBehavior IDが一意である。
2. 全unitのReplay Targetがplan全体のexplicit Tと一致する。
3. 初回contractでは全unitのRequiredDistinctRunsが2以上で同一である。
4. run identityが空でなく、run間で一意である。
5. 各runのTarget revisionがexplicit Tと一致する。
6. 各runがplanの全behaviorをちょうど1件ずつ持つ。
7. observationのrun identityが親runと一致する。
8. observationのTarget revisionがexplicit Tと一致する。
9. observed HTTP statusがplanのexpected boundaryと一致する。
10. shared validatorがPASSし、Observation classがplanのexpected classと一致する。
11. raw artifact filenameが空でなく、SHA-256 identityが有効である。

behavior単位のrepeatable判定は、上記を満たすdistinct runがRequiredDistinctRuns以上存在することだけから導出する。

```text
same response bytes
  != repeatability

different response bytes
  != non-repeatability

same T
  + distinct runs
  + same transport boundary
  + same semantic class
  + exact raw artifact identities
  -> behavior-unit repeatability
```

集約器はraw artifact SHA-256同士の一致を要求しない。digestは各raw artifactのidentityであって、run間の同値判定値ではない。

初回summaryの最低shape:

```text
ReplayTargetBuildRevision
RequiredDistinctRuns
ObservedDistinctRuns
Behaviors[]
  - BehaviorID
  - DistinctRuns
  - Repeatable
RepeatabilityObserved
CoverageClass = behavior-unit
DeepWorkflowCovered = false
```

全behaviorがrequired distinct runsを満たした場合だけ `RepeatabilityObserved=true` とする。一部behaviorの成功を全体repeatabilityへ昇格させない。

このaggregation contractについては、Repository外の小さいprototypeで次のfail-closed性を確認済みである。ただしこれはfull RepositoryのSource Evidenceではなく、実装形の事前検証にすぎない。

- 2 distinct runsでraw digestが互いに異なっていても、T / HTTP / semantic classが一致すればrepeatability成立
- duplicate run identityを拒否
- HTTP boundary driftを拒否
- semantic class driftを拒否
- raw artifact SHA-256不正を拒否
- Target revision driftを拒否

Repositoryへ実装する際はこのprototypeをそのまま正本化せず、current mainの型・Package境界へ合わせ、full `tools/repository-health-check.sh` を通す。

#### 4.2 repeatability aggregator prototype preflight

4.1のcontractを実装形へ落とすため、Repository外の独立Go moduleでrepeatability aggregator prototypeを作成した。

候補path:

- `tools/reference-first-web-target-r7b-repeatability/`

prototype source identity:

```text
tools/reference-first-web-target-r7b-repeatability/main.go
  git blob = 79df9595d25ef8aefd0c364d01b103277226d228
  size     = 12149 bytes

tools/reference-first-web-target-r7b-repeatability/main_test.go
  git blob = dc233d3bd58a057ab298a046d9719afc1c6efa64
  size     = 8472 bytes
```

focused preflight:

```text
gofmt -l
  = empty

go vet ./tools/reference-first-web-target-r7b-repeatability
  = PASS

go test -count=50 ./tools/reference-first-web-target-r7b-repeatability
  = PASS
```

aggregatorはraw responseを読まず、V2が生成したaudit-derived `run-observation.json` だけを入力する。

CLIの概念形:

```text
-replay-plan <r7b-replay-plan.json>
-run-observation <run-01/run-observation.json>
-run-observation <run-02/run-observation.json>
```

planはstructとbytesを別authorityとして受け取らず、aggregator自身が**exact plan bytesからdecode**し、そのbytesをSHA-256 fingerprintする。

各runについて少なくとも:

- `SourceReplayPlanSHA256 == actual replay-plan bytes SHA-256`
- 全runの `SourceReplayPlanReceiptSHA256` が同一
- non-emptyかつdistinctなRunIdentity
- B / R / T / TargetAdapter / Hostがplanと一致
- `TargetListenScope=loopback`
- planの全behaviorをexact 1件ずつ保持
- observed HTTP == plan expected HTTP
- `SemanticValidationPassed=true`
- Observation class == plan expected class
- raw artifact filenameがnon-empty
- raw artifact SHA-256がvalid

を要求する。

`RequiredDistinctRuns` は全behaviorで同一かつ2以上を要求し、入力run数が不足する場合はsummaryを生成せずfail closedする。

summaryは少なくとも:

```text
EvidenceNature
  = first-web-target-r7b-repeatability-summary

SourceReplayPlanSHA256
SourceReplayPlanReceiptSHA256
ReplayTargetBuildRevision
RequiredDistinctRuns
ObservedDistinctRuns

SourceRuns[]
  - RunIdentity
  - RunObservationSHA256

Behaviors[]
  - BehaviorID
  - DistinctRuns
  - Repeatable

RepeatabilityObserved
CoverageClass = behavior-unit
DeepWorkflowCovered = false
```

を保持する。

特にraw artifact digestはrun間同値判定に使わない。prototype happy pathは2 runの同一behaviorで**異なるraw SHA-256**を与えても、same T / same HTTP boundary / same semantic class / audit PASSならrepeatabilityが成立することを確認している。

fail-closed pressureでは少なくとも:

- duplicate run identity
- required run不足
- Target / B / R / adapter / host / listen scope drift
- source replay-plan digest drift
- run間replay-plan receipt identity drift
- HTTP boundary drift
- semantic class drift
- `SemanticValidationPassed=false`
- invalid raw SHA-256
- missing / duplicate behavior
- behavior間のRequiredDistinctRuns不一致

をrejectする。

このprototypeはaggregation責務の事前検証であり、Repository source candidateやactual repeatability Evidenceではない。

```text
repeatability aggregator focused PASS
  != V2 integrated
  != E1 lifecycle Evidence
  != repository-health-check PASS
  != repeatability actually observed
```

正本化時はE0 → V1 → V2のhealth-check済み統合後、latest run-observation shapeへreconcileし、E1 runnerから2 fresh process runのaudit outputを渡す。raw semantic validationをaggregator側へ追加しない。



## 実装ファイル境界

R7B実装でsemantic判定を複製しないため、初回incrementの責務をファイル単位で次へ固定する。名称は実装時のGo package制約で微修正してよいが、責務を再混合しない。

### shared semantic classifier

候補path:

- `tools/internal/firstwebtargetevidence/`

このpackageだけが、First Web Target raw JSONについてGate CとR7Bが共有すべきsemantic classificationを所有する。

最低限共有するもの:

- raw envelope / Provenanceのdecode shape
- exact expected Provenanceとの照合
- read result validation
- command success validation
- authentication denied validation
- authorization denied validation
- Validation not-satisfied validation
- Command Invocation Execution Error validation
- semantic Observation class constant / result

共有しないもの:

- Gate C Ready判定
- Visual Fidelity
- architecture conformance
- replay run identity
- repeatability判定
- HTTP transportの実行
- request fixture / credential

既存 `tools/reference-first-web-target-evidence-audit` はこのshared classifierを呼ぶ側へ移し、現在の意味判定を変えない。refactorの目的はR7Bのための第二semantic truthを防ぐことであり、Gate C判定を変更することではない。

#### V1 branchless candidate preflight — 2026-09-20

E0とremote branchを競合させず次段を準備するため、V1 shared semantic classifierを**unreferenced Git commit**として作成した。main / branch refは動かしていない。

candidate chain:

```text
main snapshot
  17acb5ca3679d5931a4522580726fdd9a2ad49c2
    ↓ E0 branchless candidate
  c0a8d2c858f338cffd90f86752a21293c391350b
    ↓ V1 branchless candidate
  fe627f3cb24a6bbef7cb34ccebd7702ff43ae07a
```

V1 candidateの変更は次の3ファイルだけである。

```text
tools/internal/firstwebtargetevidence/semantics.go
  blob = 46629767bee3b3733c0ca6e4433971c4cc85bb9d

tools/internal/firstwebtargetevidence/semantics_test.go
  blob = b94db230f15871a36151f077be8aab13928b9d06

tools/reference-first-web-target-evidence-audit/main.go
  blob = eb44ef1d543f0849e424e99ed517492f747d634a
```

shared packageはgeneric `Validate(kind, payload)` dispatcherを置かず、current six behaviorの明示validatorを公開する。

```text
ValidateReadArtifact
ValidateCommandSuccessArtifact
ValidateAuthenticationDeniedArtifact
ValidateAuthorizationDeniedArtifact
ValidateValidationNotSatisfiedArtifact
ValidateExecutionErrorArtifact
```

Observation classのexact stringとReference credential reject helperもtool-local packageへ置く。Canonical model / `internal/core` へ昇格させない。

Gate C audit candidateは、current source blob
`2dd0102a04bdda29dd50dab13ba9a663b8df93d8`
から6 raw behaviorのacceptance predicateだけをshared validator呼び出しへ移す。Visual / architecture / final readiness / artifact-set / credential-set / Target Difference Handoffの責務は移さない。validated `read.json` は既存Gate C-local型へ再decodeし、Target Difference Handoff生成shapeを変更しない。

shared packageのexact candidate bytesについて、このsessionで次を確認した。

```text
git hash-object semantics.go
  = 46629767bee3b3733c0ca6e4433971c4cc85bb9d

git hash-object semantics_test.go
  = b94db230f15871a36151f077be8aab13928b9d06

gofmt -l
  = empty

go vet ./tools/internal/firstwebtargetevidence
  = PASS

go test -count=50 ./tools/internal/firstwebtargetevidence
  = PASS
```

ただしprivate Repository full checkoutをこのsessionへ取得できないため、candidate Gate C auditを含むfull Repository qualificationは未実行である。

```text
shared package focused PASS
  != Gate C audit package full PASS
  != repository-health-check PASS
  != V1 integrated
```

統合sessionはE0を先にhealth-checkしてmainへ入れた後、latest mainへV1 candidateをreconcileし、`tools/repository-health-check.sh` をexact candidate上で通す。PASS前に `fe627f3c...` をmainへ参照させない。

### R7B replay audit

新規候補path:

- `tools/reference-first-web-target-r7b-replay-audit/`

このauditは:

1. exact replay planを読む
2. run metadataと6 raw responseを読む
3. HTTP status / Target revision / run identityをreplay planと照合する
4. raw bytesをSHA-256 fingerprintする
5. shared semantic classifierへraw responseを渡す
6. expected Observation classとの一致を確認する
7. per-run observationを生成する

semantic判定条件を自身へcopyしない。

#### V2.1 run capture input boundary

V2 auditはrunnerの主張をそのままEvidenceへ昇格させない。runnerから受け取るmetadataは、raw responseをどのbehaviorのtransport observationとして取得したかを結び付ける最小情報に限定する。

概念shape:

```text
run-metadata.json
  EvidenceNature = first-web-target-r7b-run-metadata
  RunIdentity
  SourceReplayPlanSHA256
  SourceReplayPlanReceiptSHA256
  ReplayTargetBuildRevision
  TargetProcessInstanceIdentity
  TargetListenScope = loopback
  Captures[]
    - BehaviorID
    - ObservedHTTPStatus
    - RawArtifactFile
```

ここには `ObservationClass`、`SemanticValidationPassed`、`RawArtifactSHA256` をrunnerの自己申告値として持ち込まない。これらはauditがraw bytesとshared classifierから導出する。

```text
runner claims
  observed HTTP + file mapping
        ↓
R7B replay audit
  exact plan digest binding
  raw bytes fingerprint
  shared semantic validation
        ↓
run-observation.json
```

`TargetProcessInstanceIdentity` はrunごとに生成するnonsemantic identityで、2 run間で異なることを後段が確認できるようにする。ただしこのfield単独をfresh OS processの独立証明とは扱わない。fresh start / stop / waitを実際に行う責務はE1 runnerにあり、最終receiptはそのlifecycleをbindingする。

run metadataではactual port番号をEvidence identityへ採用しない。auditに必要なのは `loopback` scopeであり、ephemeral port自体の同一性ではない。

#### V2.2 audit-derived run observation

V2が生成する `run-observation.json` は少なくとも次を保持する。

```text
EvidenceNature = first-web-target-r7b-run-observation
RunIdentity
TargetProcessInstanceIdentity
SourceReplayPlanSHA256
SourceReplayPlanReceiptSHA256
HandoverBaselineID
BaselineRepositoryRevision
ReplayTargetBuildRevision
TargetAdapter
Host
TargetListenScope = loopback
Observations[]
  - BehaviorID
  - ObservedHTTPStatus
  - ObservationClass
  - SemanticValidationPassed
  - RawArtifactFile
  - RawArtifactSHA256
```

B / R / T / adapter / hostはplanからexpected valueを受け、raw response内Provenanceをshared classifierでexact照合した結果としてoutputへ保持する。metadata側に同じ値を重複記述して一致を装う方式にはしない。

V2は1 runのEvidence化までを担当し、2 runのrepeatability集約は行わない。したがって `RepeatabilityObserved` やbehaviorの `Repeatable` をrun observationへ書かない。

#### V2.3 V2 fail-closed test matrix

V2実装時はsynthetic fixtureで少なくとも次をPressureする。

1. replay-plan actual bytesのSHA-256と `SourceReplayPlanSHA256` が不一致ならreject。
2. replay-plan receipt identityがmetadataのexpected receipt digestと不一致ならreject。
3. empty run identity / empty process instance identityをreject。
4. plan Tとrun metadata Tのdriftをreject。
5. non-loopback listen scopeをreject。
6. missing / duplicate / unknown behaviorをrejectし、6 behavior exact setだけを受理。
7. expected HTTP statusとobserved statusのdriftをreject。
8. raw artifact fileの欠落、absolute path、parent traversal、重複mappingをreject。
9. raw bytesのSHA-256はaudit自身が計算し、runner supplied digestをauthorityにしない。
10. raw ProvenanceのB / R / T / adapter / host driftをshared classifier経由でreject。
11. expected Observation classとshared classifier結果のdriftをreject。
12. known Reference credential materialを含むraw artifactをreject。
13. happy pathでは6 observationすべてにaudit-derived SHA-256とsemantic classが残る。
14. audit outputにcredential、request body、selector、coordinate、current Authorization shortcutを含めない。

これらのtestはGate C semantic条件のcopyではなく、shared packageを呼ぶintegration pressureとして書く。

#### V2.4 branchless prototype preflight

V2 replay auditの責務とfail-closed境界について、Repository外の独立Go moduleでsource prototypeを作成し、V1 candidateと同じshared validator APIへ接続した。

prototype source identity:

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  git blob = 9544b9b0a9d741a9b2b4b67359399287aec008cb
  size     = 16647 bytes

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  git blob = 8e8ba6b8aff37b53800d18f553cefe7cc70a4cfd
  size     = 14708 bytes
```

focused preflight:

```text
gofmt -l
  = empty

go vet ./tools/internal/firstwebtargetevidence \
       ./tools/reference-first-web-target-r7b-replay-audit
  = PASS

go test -count=50 \
  ./tools/internal/firstwebtargetevidence \
  ./tools/reference-first-web-target-r7b-replay-audit
  = PASS
```

prototypeはrunnerから次の3値だけをcapture authorityとして受け取る。

```text
BehaviorID
ObservedHTTPStatus
RawArtifactFile
```

runner suppliedの `ObservationClass` / `SemanticValidationPassed` / `RawArtifactSHA256` は入力shapeへ持たせない。

audit自身が:

1. actual replay-plan / receipt bytesをSHA-256 fingerprintする
2. plan / receipt / run-metadataのexact bindingを検証する
3. raw artifact bytesを読み、自身でSHA-256を計算する
4. V1 shared validatorでB / R / T / adapter / hostとsemantic boundaryを検証する
5. shared Observation class constantとplan expected classを照合する
6. Reference credential materialをrejectする
7. 1 run分の `first-web-target-r7b-run-observation` を生成する

という境界を持つ。

focused prototypeでは少なくとも次をfail closedで確認した。

- replay-plan / receipt / metadata digest mismatch
- receiptのsource-intake identity drift
- receiptがreplay済み / repeatability観測済みを主張する状態
- empty run identity / process instance identity
- Target revision drift / non-loopback listen scope
- missing / duplicate / unknown behavior
- absolute path / parent traversal / wrong raw artifact mapping
- expected HTTP boundary drift
- plan expected Observation classとshared validator classのdrift
- raw ProvenanceのB / R / T / adapter / host drift
- retained raw artifactへのReference credential material混入

happy pathでは6 behaviorすべてについて、audit-derived raw SHA-256、shared semantic class、`SemanticValidationPassed=true` を持つ1 run observationを生成する。

ただしこのprototype sourceはまだRepository source candidateとして統合されておらず、full Repository qualificationも未実行である。

```text
V2 focused prototype PASS
  != V1 integrated
  != V2 repository candidate qualified
  != repository-health-check PASS
  != replay executed
  != repeatability observed
```

V2を正本化するsessionは、E0 → V1をhealth-check済みでmainへ統合した後、そのlatest main上でprototype責務をreconcileし、`tools/repository-health-check.sh` を通す。shared semantic predicateをV2側へcopyしない。



#### V2 branchless candidate preflight — 2026-09-20

V1 shared-validator candidateを利用しつつremote branchを増やさないため、V2 replay auditも**unreferenced Git commit**として作成した。main / branch refは動かしていない。

candidate chain:

```text
main snapshot
  17acb5ca3679d5931a4522580726fdd9a2ad49c2
    ↓ E0 branchless candidate
  c0a8d2c858f338cffd90f86752a21293c391350b
    ↓ V1 branchless candidate
  fe627f3cb24a6bbef7cb34ccebd7702ff43ae07a
    ↓ V2 branchless candidate
  a5b70735a1b2ebc6b9772908bef7871f9097bd8a
```

V2 candidateの新規sourceは次の2ファイルである。

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  blob = 521d7c7b9d62e35cef64188810d970fb691c317f

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  blob = 249a335a6ba2c57445bffdc48b68932932790363
```

candidateは次を実装する。

- replay plan / replay-plan receipt exact bytesへのSHA-256 binding
- E0後のB / R / T / TargetAdapter / Host exact provenance validation
- run / Target process instance identity必須化
- loopback scopeの固定
- 6 behavior exact set / duplicate / unknown reject
- expected HTTP boundary照合
- raw artifact filenameのabsolute path / parent traversal / duplicate mapping reject
- raw artifactをaudit自身でSHA-256 fingerprint
- V1 shared validatorによるsemantic validation
- plan expected Observation classとshared classの一致
- Reference credential material reject
- audit-derived `run-observation.json` shape

runner supplied metadataには `ObservationClass` / `SemanticValidationPassed` / `RawArtifactSHA256` をauthorityとして持たせない。これらはV2 auditがraw bytesから導出する。

exact candidate bytesをRepository外のisolated moduleへmaterializeし、GitHub blob identityとの一致を確認したうえで次を実行した。

```text
git hash-object main.go
  = 521d7c7b9d62e35cef64188810d970fb691c317f

git hash-object main_test.go
  = 249a335a6ba2c57445bffdc48b68932932790363

gofmt -l
  = empty

go vet ./tools/reference-first-web-target-r7b-replay-audit ./tools/internal/firstwebtargetevidence
  = PASS

go test -count=50 ./tools/reference-first-web-target-r7b-replay-audit ./tools/internal/firstwebtargetevidence
  = PASS
```

preflight中、raw artifact filenameのWindows separator rejectが二重backslashではなく**単一backslash**を拒否するexact candidateであることもblob identityを合わせて再確認した。

ただしこれはV1同様focused candidate preflightであり、full Repository qualificationではない。

```text
V2 focused PASS
  != E0 integrated
  != V1 integrated
  != repository-health-check PASS
  != V2 integrated
  != actual replay Evidence
```

統合sessionはE0をhealth-check済みでmainへ入れ、続いてV1をlatest mainへreconcileしてfull health checkを通した後、V2 candidateをreconcileする。V2もexact candidateを含むfull checkoutで `tools/repository-health-check.sh` がPASSするまでmainへ参照させない。

#### V1 pressure強化を含むV2 composition candidate

V1 branchless candidateのfocused preflight後、正本のV1 test migration matrixに対してnegative pressureが不足していることを再監査した。

不足分だけをtest-only fileとして追加したV1強化candidate:

```text
fe627f3cb24a6bbef7cb34ccebd7702ff43ae07a
  ↓ add semantics_pressure_test.go
fba329f92dbb1a209a6fd89c2c84a51be78f6eb1
```

追加blob:

```text
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6
```

このpressure fileはexact shared source blob
`46629767bee3b3733c0ca6e4433971c4cc85bb9d`
とRepository外の一時Go packageで組み合わせ、次を確認した。

```text
git hash-object semantics.go
  = 46629767bee3b3733c0ca6e4433971c4cc85bb9d

gofmt -l .
  = empty

go vet ./...
  = PASS

go test -count=50 ./...
  = PASS
```

追加pressureは少なくとも次を網羅する。

- B / R / T / TargetAdapter / Hostのempty / driftを6 validatorすべてでreject
- read Scenario / Application / browse / customer / order-lines binding drift
- command-success stage / Authentication / Validation / Authorization / Command / Observation drift
- authentication-denied stage / Authentication / Command / Observation drift
- authorization-denied stage / Authorization / Command / Observation drift
- validation-not-satisfied stage / Validation / downstream-result presence drift
- execution-error stage / Authorization / Command / Observation / current execution-error condition drift
- Reference credential reject set

V2 branchless candidate `a5b70735...` は旧V1 candidate `fe627f3c...` を親にしているため、V1強化candidateとは枝分かれしていた。code conflictはなくtest file 1件の追加だけなので、V2 treeへ同じblobを載せたcomposition candidateを作成した。

```text
a5b70735a1b2ebc6b9772908bef7871f9097bd8a
  ↓ add ce77296e... test-only blob
056527b6203feaf3c3830db4d33251d64c475be3
```

`056527b6...` はV2 source candidateを変更せず、V1 negative pressureを保持するための候補treeである。

Evidence heightは分離する。

```text
a5b70735 V2 focused PASS
  + ce77296e pressure focused PASS
  != 056527b6 combined full-package PASS
  != repository-health-check PASS
  != integrated
```

したがって統合sessionでは、E0統合後のlatest mainへV1/V2 sourceをreconcileする際に **`056527b6...` のtree内容を優先参照**し、そのexact reconcile結果を含むfull checkoutで `tools/repository-health-check.sh` を実行する。unreferenced commit SHAそのものをmainへ機械的にfast-forwardしない。

### operational runner

新規候補path:

- `tools/run-first-web-target-r7b-replay.sh`

runnerはorchestrationだけを所有する。

#### E1.0 replay-plan receipt validation

E1は `r7b-replay-plan.json` と `r7b-replay-plan-receipt.json` を別々に「存在確認」するだけでは不十分である。runner開始前にreceiptをdecodeし、plan exact bytesと相互bindingを検証する。

current receipt shape:

```text
EvidenceNature
ReplayPlanToolRepositoryRevision
RepositoryPostRunRevision
RepositoryPostRunClean
SourceQualificationPassed
SourceExecutionIntakeSHA256
SourceExecutionIntakeReceiptSHA256
ReplayPlanArtifact
ReplayPlanArtifactSHA256
ReplayExecuted
RepeatabilityObserved
```

E1開始条件:

1. `EvidenceNature == first-web-target-r7b-replay-plan-acquisition-receipt`
2. `ReplayPlanToolRepositoryRevision` がvalid 40-hex Git revision
3. `RepositoryPostRunRevision == ReplayPlanToolRepositoryRevision`
4. `RepositoryPostRunClean == true`
5. `SourceQualificationPassed == true`
6. `ReplayPlanArtifact == r7b-replay-plan.json`
7. `ReplayPlanArtifactSHA256` がactual plan bytesのSHA-256とexact一致
8. receiptの `SourceExecutionIntakeSHA256` がplanの同名source identityとexact一致
9. receiptの `SourceExecutionIntakeReceiptSHA256` がplanの同名source identityとexact一致
10. `ReplayExecuted == false`
11. `RepeatabilityObserved == false`

どれか1つでも満たさなければTarget processを起動しない。

```text
plan exists
  != plan acquired correctly

plan digest matches
  != source qualification proven

receipt present
  != replay may start

exact plan bytes
  + exact receipt binding
  + clean qualified acquisition
  + pre-execution state
  -> E1 input eligible
```

receipt内のtool revisionは「そのrevisionが現在も最新である」ことを意味しない。これはplanを生成したsource revisionのprovenanceであり、current main authorityではない。

E1はplan / receiptをRepository外outputへcopyする場合、copy後bytesを再fingerprintし、source digestと一致することを最終receiptへbindingする。path文字列だけをsource identityとして採用しない。

#### E1.0.1 V2 preflight validation reuse candidate

E1 runnerがreplay-plan / receipt validationをshell側へ再実装すると、V2 auditとの間に第二のinput truthができる。これを避けるため、V1 pressureを含むV2 composition candidate `056527b6203feaf3c3830db4d33251d64c475be3` の子として、V2のexact validation pathをTarget起動前にも呼べるbranchless candidateを作成した。

```text
056527b6203feaf3c3830db4d33251d64c475be3
  ↓ V2 plan/receipt preflight reuse
ed17d529f605097a2171fc39cfdb5ed9d2c117ac
```

変更対象はV2 replay auditの2ファイルだけである。

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  blob = 62b526dc865ad49214aa84fcf4e9b40eeb6c4dff

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  blob = 28464cfdc3c06bd4e2baac8cf6173eb3210703ce
```

差分は次に限定する。

- replay-plan / receiptのdecode、exact byte SHA-256、`validatePlan`、`validateReceipt` を `decodeAndValidateReplayInputs` へ集約
- normal V2 run auditも同じ関数を利用
- CLIへ `-preflight-only` を追加
- preflight modeではrun metadata / raw directoryを要求せず、plan / receipt exact validationだけを行って成功時に `ok` を返す
- missing launch provenance、receipt digest drift、already-executed receiptをnegative pressureする

E1 runnerの開始時はconceptually:

```text
go run ./tools/reference-first-web-target-r7b-replay-audit \
  -replay-plan <r7b-replay-plan.json> \
  -replay-plan-receipt <r7b-replay-plan-receipt.json> \
  -preflight-only

PASS
  -> Target process lifecycleへ進める

FAIL
  -> Target processを起動しない
```

とする。

```text
E1 input validation
  = V2 exact plan/receipt validation reuse

E1 shell-local JSON predicate
  = prohibited
```

このcandidateはunreferenced Git commitであり、main / remote branch refは動かしていない。差分監査ではsemantic validator、run observation shape、repeatability判定には変更を加えていない。

同じvalidation再利用形はRepository外prototypeで `gofmt` clean、`go vet` PASS、`go test -count=50` PASSを確認しているが、`ed17d529...` のexact candidateを含むfull Repository qualificationは未実行である。

```text
preflight reuse prototype PASS
  != ed17d529 exact full-package qualification
  != repository-health-check PASS
  != V2 integrated
  != E1 runner integrated
```

統合時はE0 → V1 → V2をhealth-check済みでmainへreconcileした後、この小差分をlatest V2 sourceへ適用し、同じfull `tools/repository-health-check.sh` gateを通す。

#### E1.0.2 repeatability aggregation candidate

2 runの結果集約をshellへ埋め込まず、V2が生成したrun observationだけを入力にする小さいGo toolをbranchless candidateとして追加した。

最初にV2 composition candidate `056527b6...` の子として試作candidate `f180ed04...` を作ったが、その直後にE1 preflight reuse candidate `ed17d529...` が同じ親から追加された。candidate chainを枝分かれさせないため、repeatability sourceを `ed17d529...` の子へ載せ直した。

今後の優先candidateは次である。

```text
056527b6203feaf3c3830db4d33251d64c475be3
  ↓ V2 plan/receipt preflight reuse
ed17d529f605097a2171fc39cfdb5ed9d2c117ac
  ↓ repeatability aggregation
d779f292d17b21e01a859fda0e77b2cc44c53371
```

`f180ed04...` はbranch / main refから参照されておらず、統合候補としては使用しない。

追加source:

```text
tools/reference-first-web-target-r7b-repeatability/main.go
  blob = a4510fca4e073eb3a9982b3dfd79d6844b69a2c5

tools/reference-first-web-target-r7b-repeatability/main_test.go
  blob = 04f6d0fa4339946d4eead6b663f4beb34ea72ce5
```

集約器はraw responseを再分類しない。入力はexact replay planとV2 audit済みrun observationsだけであり、次を要求する。

- plan内Behavior IDが一意
- unitの `CoverageClass=behavior-unit`
- 全unitの `RequiredDistinctRuns >= 2` かつ同一
- run observationがexact replay-plan SHA-256へbinding
- 全runが同一 replay-plan receipt SHA-256へbinding
- run identityが一意
- Target process instance identityが一意
- run Target revisionがplan Tとexact一致
- 各runがplan behavior setをexactに1件ずつ保持
- expected HTTP boundary一致
- `SemanticValidationPassed=true`
- Observation classがplan expectationと一致
- raw artifact filenameが非空
- raw artifact SHA-256が有効

raw artifact SHA-256の**run間一致は要求しない**。digestは各artifact bytesのidentityであり、意味的repeatabilityの比較値ではない。

summary shapeは少なくとも:

```text
EvidenceNature = first-web-target-r7b-repeatability-summary
SourceReplayPlanSHA256
SourceReplayPlanReceiptSHA256
ReplayTargetBuildRevision
RequiredDistinctRuns
ObservedDistinctRuns
RunIdentities[]
Behaviors[]
  - BehaviorID
  - DistinctRuns
  - Repeatable
RepeatabilityObserved
CoverageClass = behavior-unit
DeepWorkflowCovered = false
```

を保持する。

Repository外focused preflightではexact source bytesについて:

```text
git hash-object main.go
  = a4510fca4e073eb3a9982b3dfd79d6844b69a2c5

git hash-object main_test.go
  = 04f6d0fa4339946d4eead6b663f4beb34ea72ce5

gofmt -l
  = empty

go vet ./tools/reference-first-web-target-r7b-repeatability
  = PASS

go test -count=50 ./tools/reference-first-web-target-r7b-repeatability
  = PASS
```

を確認した。

negative pressureでは少なくとも:

- duplicate run identity
- duplicate Target process identity
- replay-plan receipt identity drift
- Target revision drift
- HTTP boundary drift
- Observation class drift
- semantic validation failure
- invalid raw digest
- missing / duplicate / unknown behavior
- required run不足
- plan内required run count不一致

をrejectする。

ただし `d779f292...` はE1 preflight reuse sourceとrepeatability sourceを同一treeへcompositionした候補であり、**combined full-package test / full repository health checkは未実行**である。統合時はこのtree内容をlatest mainへreconcileし、full `tools/repository-health-check.sh` を通す。

runnerはorchestrationだけを所有する。

- current source checkout qualification
- input plan / receipt / origin Evidence verification
- exact T isolated checkout作成
- Target process lifecycle
- named request fixture resolution
- HTTP status capture
- raw response保存
- replay audit呼び出し
- 2 runの独立性確保
- repeatability summary生成
- artifact credential scan
- current source checkout不変確認

runner内のfixture resolutionではReference credentialをruntime inputとして使ってよいが、request body / credentialをEvidence artifactへ保存しない。

## Observation class mapping

replay planのcurrent exact mappingをsemantic classifierとの接続契約とする。

| Behavior | Expected HTTP | Expected Observation class |
| --- | ---: | --- |
| `first-web/read-only` | 200 | `read-result` |
| `first-web/command-success` | 200 | `business-result-plus-separate-observation` |
| `first-web/authentication-denied` | 200 | `business-denial-authentication` |
| `first-web/authorization-denied` | 200 | `business-denial-authorization` |
| `first-web/validation-not-satisfied` | 200 | `validation-not-satisfied` |
| `first-web/execution-error` | 422 | `command-invocation-execution-error` |

この文字列をR7B側だけで別定義しない。replay planとshared classifierの間でdriftした場合はfail closedする。

HTTP statusはtransport EvidenceでありObservation classそのものではない。特にAuthentication / Authorization denialとValidation not-satisfiedがHTTP 200であることを「成功」と読み替えない。

## incremental integration order

direct-to-main flowでは、remote work branchを新設せず、health-check可能なfull checkoutから小さく統合する。

1. **V1: shared classifier extraction**
   - Gate C auditのcurrent tests / behaviorを不変に保つ
   - shared package testsを追加
   - full repository health check
2. **V2: R7B replay audit**
   - synthetic test fixturesでplan binding / classification / fail-closedをPressure
   - actual replay成功はまだ主張しない
   - full repository health check
3. **E1: operational runner**
   - exact-T isolated checkout / 2 fresh processes / artifact retentionを実装
   - full repository health check
4. **M/E actual execution**
   - adopted Release archiveをactual materialize
   - R7A -> R7B intake -> replay planをactual materialize
   - operational runnerをactual実行
   - repeatability Evidenceを記録

各incrementの直前にmainを再取得する。同じpackageを別sessionが変更していた場合は、その変更を取り込んでから続行し、並行した別semantic classifierを作らない。

## 対象behavior

replay planに定義された次の6件だけを初回対象とする。

- `first-web/read-only`
- `first-web/command-success`
- `first-web/authentication-denied`
- `first-web/authorization-denied`
- `first-web/validation-not-satisfied`
- `first-web/execution-error`

初回IssueでCustom UI replayやdeep workflow replayへ広げない。

## 出力

Repository外へ概ね次の構造を保持する。

```text
r7b-replay/
  source/
    replay-plan identity
    replay-plan-receipt identity
    origin-evidence identity
  run-01/
    <six raw response artifacts>
    run-observation.json
  run-02/
    <six raw response artifacts>
    run-observation.json
  repeatability-summary.json
  r7b-replay-receipt.json
  source-tests.log
```

receiptにはsource revision、replay Target build、run identities、raw artifact identities、actual repeatability observedの有無を残す。

Booleanの `originManifestVerified=true` だけではexact Evidence identityとして不足するため、少なくとも adopted Gate C archive SHA-256、展開後 `MANIFEST.sha256` 自体のSHA-256、replay plan / receipt SHA-256、各run-observation SHA-256、repeatability-summary SHA-256を最終receiptへbindingする。

## fail-closed条件

少なくとも次を拒否する。

- dirty current source checkoutでのsource qualification
- replay plan / receipt digest mismatch
- replay-plan receiptのEvidenceNature / source qualification / clean revision / pre-execution state不整合
- receipt source-intake identitiesとplan source identitiesの不一致
- Gate C archive digest mismatch
- `MANIFEST.sha256` failure
- origin raw artifact digest mismatch
- replay Target revisionがplanのTと異なる
- clean isolated T checkoutを作れない
- duplicate run identity
- completed runが2未満
- behavior setが6件未満・重複・未知
- raw artifact欠落
- raw artifact SHA-256不正
- HTTP boundary mismatch
- semantic classification mismatch
- retained artifactへのReference credential混入
- raw responseだけでなく、server / test log、metadata、summary、copyしたsource receiptを含むretained output set全体のcredential scan未実施
- operational run中のcurrent Repository source変更

## 検証gate

コードをmainへ入れる前に、Go-enabled full checkoutで次を必須とする。

```bash
tools/repository-health-check.sh
```

focused package testだけではmain統合条件を満たさない。CIはこの条件の代替にしない。

actual replay executionはさらに、Release archiveの再取得・digest照合とactual R7A/R7B materializationを済ませてから行う。

## 完了条件

- [ ] Gate CとR7Bが同じsemantic validatorを利用する
- [ ] replay audit sourceが6 behaviorとexact plan bindingを検証する
- [ ] operational runnerがisolated exact-T checkoutを使う
- [ ] fresh processで2 distinct runsを実行する
- [ ] 12個のraw behavior artifact（6 × 2）にSHA-256 identityがある
- [ ] credential materialがretained artifactへ含まれない
- [ ] behaviorごとのtransport / semantic repeatabilityをsummaryへ記録する
- [ ] `CoverageClass=behavior-unit` / `DeepWorkflowCovered=false`を保持する
- [ ] `tools/repository-health-check.sh` が変更後のfull checkoutでPASSする
- [ ] actual replay Evidenceの保存先とfingerprintを記録する

## 非目標

- current mainのrepeatability評価
- cross-revision compatibility
- whole-workflow coverage
- Visual Fidelityの再評価
- production execution authorityの付与
- current Authorization authorityの推論
- selector / coordinate / click sequenceのCanonical化
- response byte一致の要求

## 現在の停止点

Source / Design上の入口は揃っている。現在の最初のcode stopはE0 replay-plan launch provenance carryであり、これをGo-enabled full checkoutで実装してhealth checkを通す必要がある。その後、adopted Gate C Release archiveを再取得・照合し、R7A handoff、R7B intake、**E0後の**R7B replay planをactual materializeする。actual replay executionはさらにPacket V / Packet Eをhealth-check済みで統合し、exact Tへ2 distinct runsを実行するまで未成立である。

このIssueはその停止点から先を所有する。
