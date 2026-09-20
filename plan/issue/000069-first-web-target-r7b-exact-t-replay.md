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

#### E1.0.3 origin raw Evidence identity verification candidate

E1のarchive verificationでは、archive SHA-256と `MANIFEST.sha256` の一致だけでなく、replay planが固定した6 origin raw artifact identityとの一致も確認する必要がある。これをshell側のJSON解析へ複製しないため、repeatability composition candidate `d779f292d17b21e01a859fda0e77b2cc44c53371` の子としてV2 preflightを拡張したbranchless candidateを作成した。

```text
d779f292d17b21e01a859fda0e77b2cc44c53371
  ↓ V2 origin Evidence identity verification
2038948424ab4ccb13dd8e0009beb8baba4864a6
```

変更対象はV2 replay auditの2ファイルだけである。

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  blob = 720ca6b284ffecf2e813b6fccc3f68e5226a0f68

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  blob = 8825d61068e33fd642c9f758d6774655ff6e93a8
```

candidateはreplay plan unitから次をdecodeする。

```text
OriginRawArtifactFile
OriginRawArtifactSHA256
```

さらにcurrent six behaviorについてorigin filenameをV2のTarget-local ruleへ固定し、plan側で:

- exact expected filename
- safe basename-only path
- valid SHA-256

を要求する。

`-preflight-only` に `-origin-artifact-dir <extracted Gate C Evidence root>` を併用した場合、V2は各origin raw fileをregular fileとして読み、Reference credential materialがないことを確認し、actual bytes SHA-256をplanの `OriginRawArtifactSHA256` とexact照合する。

したがってE1 runnerはconceptually:

```text
archive exact SHA-256
  + MANIFEST.sha256 verification
  ↓
V2 -preflight-only
  + -origin-artifact-dir
  ↓
plan / receipt exact binding
  + B / R / T / adapter / host
  + six origin raw filename / SHA-256 identities
  ↓
Target process may start
```

という順にする。

shell側はplan JSONからorigin file / digestを再抽出しない。archive / manifest verificationは外殻として残し、planが要求するorigin raw identityとの結合をV2へ集約する。

candidate差分監査では親 `d779f292...` に対してV2の2ファイルだけが変更されている。

```text
main.go      +58 / -3
main_test.go +72 / -7
```

ただし `20389484...` のexact candidateについてfocused Go test / full Repository qualificationはまだ実行していない。

```text
origin identity source candidate
  != focused exact candidate PASS
  != repository-health-check PASS
  != V2 integrated
  != origin Evidence adopted in an actual E1 run
```

統合時はlatest V2 sourceへこの責務をreconcileし、focused testとfull `tools/repository-health-check.sh` の両方を通す。E1 runnerはこのvalidationを通過する前にTarget processを起動しない。

#### E1.0.4 preferred composition candidate — exact receipt-bound aggregation

origin raw Evidence verification candidate `2038948424ab4ccb13dd8e0009beb8baba4864a6` は、親treeにE1 preflight reuseと旧repeatability aggregatorを含む。

旧aggregator `d779f292...` は各runが同じ replay-plan receipt SHA-256を指すことは確認するが、**actual `r7b-replay-plan-receipt.json` bytes自体を入力せず、plan↔receipt exact binding / clean qualified acquisition / pre-execution stateを集約時に再検証しない**。

E1.0 contractとの整合を上げるため、receipt実体を必須入力とする強化aggregatorをRepository外で作成し、focused verificationを行った。

exact source blobs:

```text
tools/reference-first-web-target-r7b-repeatability/main.go
  blob = 48c9c595311a4fa8fc7c4e7b2a30c4a4fd999965

tools/reference-first-web-target-r7b-repeatability/main_test.go
  blob = 1ebe3a7bfed00b38ae7604143ae06f929267f665
```

focused verification:

```text
gofmt -l .
  = empty

go vet ./...
  = PASS

go test -count=50 ./...
  = PASS
```

強化aggregatorは:

- actual replay-plan bytesをSHA-256 fingerprint
- actual replay-plan receipt bytesをSHA-256 fingerprint
- receipt `ReplayPlanArtifactSHA256` とactual plan bytesをexact照合
- receipt source-intake identitiesとplan source identitiesをexact照合
- clean qualified acquisitionを要求
- `ReplayExecuted=false`
- `RepeatabilityObserved=false`
- run observationの `SourceReplayPlanReceiptSHA256` を**actual receipt bytes digest**へexact binding
- B / R / T / TargetAdapter / Hostをplanと各runで照合
- exact six behavior set
- distinct run identity
- distinct Target process instance identity
- expected HTTP / semantic class
- valid raw artifact SHA-256
- raw digestのrun間同値を要求しない

を保持する。

枝分かれを残さないため、最終的にorigin raw Evidence preflight candidate `20389484...` のtreeへこの2 blobを置き換えたcomposition candidateを作成した。

```text
056527b6203feaf3c3830db4d33251d64c475be3
  ↓ V2 plan/receipt preflight reuse
ed17d529f605097a2171fc39cfdb5ed9d2c117ac
  ↓ older repeatability candidate
d779f292d17b21e01a859fda0e77b2cc44c53371
  ↓ origin raw Evidence verification
2038948424ab4ccb13dd8e0009beb8baba4864a6
  ↓ replace aggregator with exact receipt-bound blobs
a789401cabbd1280100f30de3d94b9154ab26c0d
```

統合時の**preferred composition treeは `a789401c...`** とする。

ただしEvidence heightは分離する。

```text
V1 pressure focused PASS
+ V2 baseline focused PASS
+ strong repeatability focused PASS
+ origin preflight source present
  != a789401c combined exact candidate PASS
  != origin preflight exact candidate PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

`d779f292...` / `9adc68a4...` / `f43fb0e3...` はbranch / main refから参照せず、統合候補としては `a789401c...` に収束させる。

実際の統合sessionではunreferenced SHAをmainへfast-forwardせず、latest mainへ `a789401c...` の責務とexact source blobsをreconcileし、そのexact resultをfull `tools/repository-health-check.sh` でqualificationする。

#### E1.1 exact-T operational runner composition candidate

E1 preflight / origin raw identity / exact receipt-bound aggregationを一つの直列candidateへ収束させたうえで、operational runner本体をbranchless candidateとして追加した。

preferred chain:

```text
a789401cabbd1280100f30de3d94b9154ab26c0d
  ↓ V2 validated launch provenance output
  ↓ E1 exact-T orchestration
73f06bfa2cdceb29c9550604ce2e766e05f2fad5
```

`73f06bfa...` は `a789401c...` のstrong receipt-bound aggregatorを保持し、V2側へvalidated launch provenance出力を加え、次を追加する。

```text
tools/run-first-web-target-r7b-replay.sh
  blob = 63da55a8efe07abdf28ab700d4725584c3e23287
  mode = 100755
```

V2 launch provenance追加blob:

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  blob = 1d15fde01cf4d1d0fae89c7c01bfe551e19d2903

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  blob = a1cb0495d59e5d850209f044793cac6da5dca73a
```

runnerはV2 preflightを次の形で呼ぶ。

```text
-preflight-only
-origin-artifact-dir <verified Gate C Evidence root>
-emit-launch-provenance-tsv
```

V2がplan / receipt / B-R-T-adapter-host / six origin raw artifact identityを検証した後にだけ、validated `B / R / T / TargetAdapter / Host` を1行TSVでrunnerへ返す。runnerはplan JSONをshell側で再解析せず、そのvalidated値からexact-T processを起動する。

runner candidateの主な境界:

1. current checkout clean / non-standard index不在を確認
2. `tools/repository-health-check.sh` をactual execution前にも実行
3. plan / receiptをRepository外outputへcopyし、copy後bytesを再fingerprint
4. adopted Gate C archive exact SHA-256を確認
5. archive展開後 `MANIFEST.sha256` 全件照合とmanifest file自体のSHA-256を保持
6. V2 preflightでplan / receipt / origin six raw identitiesをexact照合
7. planからvalidation済みlaunch provenanceを取得
8. Repository外detached worktreeをexact Tで作成し、clean / required source pathを確認
9. exact-T First Web Target source testを実行
10. fresh Target processでrun 01を起動し6 behaviorを各1回実行、停止 / wait
11. fresh Target processでrun 02を同様に実行、停止 / wait
12. 各run raw responseをV2 auditへ渡し、audit-derived `run-observation.json` を取得
13. 6 raw artifactの `raw-artifacts.sha256` をrunごとに保持
14. strong repeatability toolへactual plan / actual receipt / 2 run observationを渡す
15. current source checkout不変を確認
16. final receiptへarchive / manifest / plan / receipt / launch provenance / 2 run observation / 2 raw manifest / repeatability summary SHA-256をbinding
17. retained output tree全体をReference credential materialについてscan

run identity defaultは固定T / runner revisionだけにせず:

```text
UTC timestamp + process id
```

をacquisition discriminatorに含める。明示上書き `TEI_R7B_REPLAY_ACQUISITION_ID` もsafe character setへ制限する。

request credentialはpipeでruntime inputとして渡し、request bodyは保存しない。final retained-output scanには少なくとも:

- editor-secret
- viewer-secret
- wrong-secret
- locked-secret
- disabled-secret
- credential-ref://legacy/

を含める。

runner exact blob `63da55a8...` はRepository外にmaterializeした同一bytesについて:

```text
bash -n tools/run-first-web-target-r7b-replay.sh
  -> PASS

git hash-object tools/run-first-web-target-r7b-replay.sh
  -> 63da55a8efe07abdf28ab700d4725584c3e23287
```

を確認した。

同じexact bytesをtemporary Git Repositoryへ置いた軽量fail-closed smokeでは、actual Target / Gate C Evidenceへ進む前の次の4条件もすべてexit non-zeroで拒否した。

```text
required environment variables missing
  -> reject

replay input path inside Repository
  -> reject

replay output path inside Repository
  -> reject

dirty current checkout
  -> reject
```

このsmokeはexternal servicesやexact-T processを起動していないため、runner lifecycle correctnessやactual replay PASSのEvidenceには数えない。

candidate diffはparent `a789401c...` に対して:

```text
V2 main.go       +18 / -3
V2 main_test.go  +18 / -0
runner           +449 / -0
```

に限定される。

ただし、これはshell syntaxとexact blob identityまでのpreflightであり、candidate全体のGo focused test、exact-T live replay、full Repository health checkは未実行である。

```text
runner bash syntax PASS
  != V2 changed candidate focused PASS
  != candidate composition package PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

統合sessionではunreferenced candidate SHAをmainへfast-forwardせず、latest mainへ `73f06bfa...` の責務とexact blobsをreconcileする。そのexact resultに対してfull `tools/repository-health-check.sh` を通した後にだけmainへ直接統合する。

#### E1.2 runner lifecycle / retained Evidence hardening candidate

E1.1 runnerを静的監査したところ、actual replay前に閉じるべき追加境界が見つかった。

第一に、`go run` をbackground起動すると、runnerが保持するPIDがGo tool processであり、actual Target child processの停止境界を間接的に扱う形になる。fresh-process Evidenceではprocess lifecycleを曖昧にしないため、exact-T checkoutでTarget binaryをRepository外scratchへ**事前build**し、各runではsubshellから `exec` してそのbinaryを直接background processにする。

第二に、固定base portに既存のcompatible health endpointがいた場合、Target起動直後のhealth判定が別processを誤認する余地がある。各run開始前に同じloopback `/healthz` が既に成功しないことを確認してからTargetを起動する。direct binaryがbindに失敗した場合はPID生存判定も失敗するため、そのrunを開始しない。

第三に、E1.1 final receiptはT / adapter / hostを保持していた一方、validated launch provenanceのB / RはTSV digestに間接bindingされるだけだった。hardening candidateでは `HandoverBaselineID / BaselineRepositoryRevision` もfinal receiptへ明示する。

さらに次を追加した。

- source health check後、Target Evidence取得前にcurrent Repositoryがsame revision / cleanであることを再確認
- `repository-health-check.log` / exact-T Target source test log / origin manifest check logのSHA-256をfinal receiptへbinding
- retained output credential scanのPASS artifactとSHA-256をfinal receiptへbinding
- final receipt生成後にretained output tree全体をもう一度credential scan
- output directoryを作る**前**にparentをphysical canonicalizeし、relative path経由でRepository内へdirectoryを作ってからrejectする余地を除去
- output parentは既存directoryを要求し、basename `.` / `..` をreject
- base portを1024..65534の整数へ限定
- runnerが利用するcore command dependencyを開始時に明示確認

branchless candidate chainは次へ直列化した。

```text
a789401cabbd1280100f30de3d94b9154ab26c0d
  ↓ E1.1 runner
73f06bfa2cdceb29c9550604ce2e766e05f2fad5
  ↓ lifecycle / receipt hardening
92b8ca7f9efcc8135fa1ec4b42d7f2606d8b25ba
  ↓ port expression correction path
76b7745b6b724da401a43e66b6b1d231816d3b62
  ↓ preserve literal bash PID
f2f184dda2d99cf25cb70226f21093ae32b8db72
  ↓ explicit command dependencies
d5f7c54f69997036f31811771dbfaac4f2eb9dde
  ↓ pre-materialization output-path qualification
  ↓ post-health-check source immutability check
3460457950ca355bf5d33c55b76cfc647da9c660
```

現在の**preferred runner candidate headは `34604579...`** とする。

途中の `76b7745...` はJavaScriptのreplacement-string `$` semanticsにより直前treeと実質同一になったno-op candidateである。literal bash PID `$` が実際に保持されたのは `f2f184dd...` 以降であり、`34604579...` でも:

```bash
base_port="${TEI_R7B_REPLAY_BASE_PORT:-$((19080 + ($ % 700)))}"
```

をexact sourceで再確認した。

current preferred runner blob:

```text
tools/run-first-web-target-r7b-replay.sh
  blob = 86c5b9a36a13b43ccded3e6f336effba3bc023f0
  mode = 100755
```

E1.1の元runner blob `63da55a8...` は同一bytesについて `bash -n` PASSと軽量fail-closed smokeがある。hardening後blob `86c5b9a3...` はlocalized diffを静的監査し、B/R receipt binding、direct exact-T binary lifecycle、credential scan binding、output path qualificationを確認したが、**このexact hardening blobについてのbash syntax再実行とfull Repository qualificationはまだ行っていない**。

したがってEvidence heightは:

```text
E1.1 exact runner bash -n / early fail-closed smoke PASS
+ E1.2 hardening source candidate
+ exact preferred runner blob identity fixed
  != E1.2 exact bash -n rerun
  != candidate composition Go package PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

で止める。

統合sessionでは `34604579...` をmainへfast-forwardせず、latest mainへ責務とexact sourceをreconcileする。そのfull checkoutでrunner `bash -n`、focused package tests、`tools/repository-health-check.sh` を順に通した後にだけmainへ直接統合する。

#### E1.3 branchless integration packet

candidate chainの古いdocs snapshotをmainへ巻き戻さず、code sourceだけをlatest mainへ載せるため、current preferred candidate `a315e8566f4784ecc005de209d6a6dae23d5b7cd` から取得するpathをexactに固定する。

```text
tools/internal/firstwebtargetevidence/semantics.go
  46629767bee3b3733c0ca6e4433971c4cc85bb9d

tools/internal/firstwebtargetevidence/semantics_test.go
  b94db230f15871a36151f077be8aab13928b9d06

tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6

tools/reference-first-web-target-evidence-audit/main.go
  eb44ef1d543f0849e424e99ed517492f747d634a

tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab

tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed

tools/reference-first-web-target-r7b-replay-audit/main.go
  1d15fde01cf4d1d0fae89c7c01bfe551e19d2903

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  a1cb0495d59e5d850209f044793cac6da5dca73a

tools/reference-first-web-target-r7b-repeatability/main.go
  48c9c595311a4fa8fc7c4e7b2a30c4a4fd999965

tools/reference-first-web-target-r7b-repeatability/main_test.go
  1ebe3a7bfed00b38ae7604143ae06f929267f665

tools/run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
  mode = 100755
```

candidate起点 `17acb5ca3679d5931a4522580726fdd9a2ad49c2` からcurrent mainまでを再比較した時点で、上記11 pathと重なるmain側変更は **0件** だった。別レーンのdocs / second-application変更をcandidate treeへ巻き戻す必要はない。

Go-enabled integration sessionではconceptually次のように扱う。

```text
latest main
  ↓ git pull --rebase origin main
clean / normal index
  ↓
candidate a315e856... から上記11 pathだけをrestore
  ↓
blob identityを上記expected SHAと照合
  ↓
bash -n tools/run-first-web-target-r7b-replay.sh
  ↓
focused Go tests
  ↓
tools/repository-health-check.sh
  ↓
same revision source / expected diffを再確認
  ↓
mainへ1つの適切なcode commit
```

Git commandの一例:

```bash
git restore --source=a315e8566f4784ecc005de209d6a6dae23d5b7cd -- \
  tools/internal/firstwebtargetevidence \
  tools/reference-first-web-target-evidence-audit/main.go \
  tools/reference-first-web-target-r7b-replay-plan/main.go \
  tools/reference-first-web-target-r7b-replay-plan/main_test.go \
  tools/reference-first-web-target-r7b-replay-audit \
  tools/reference-first-web-target-r7b-repeatability \
  tools/run-first-web-target-r7b-replay.sh
```

これはcandidate commitをmainへfast-forward / cherry-pickする指示ではない。**latest mainのdocs/historyを保持したまま、qualified source pathだけをworking treeへmaterializeする**ための手順である。

restore後にexpected path以外が変更されている、expected blob SHAと一致しない、またはmain側で同じpathが並行変更されていた場合は、そのままcommitせず再reconcileする。

remote work branchは作らない。full health checkがPASSしたcode incrementをmainへ直接commitし、候補Git objectは履歴参照用のpreflight sourceとしてのみ扱う。


#### E1.4 cross-session convergence audit / exact runner syntax rerun — 2026-09-20

並行sessionとremote branchを再監査した。current mainに対して、main以外のremote branchはすべて固有ahead commitが0件だった。したがって既存branch refを新しい作業起点にはせず、direct-to-main policyを維持する。古いbranchがbehindで残っていること自体は未統合workを意味しない。

current preferred code compositionは引き続き:

```text
3460457950ca355bf5d33c55b76cfc647da9c660
```

であり、E1.3の11-path integration packetだけをlatest mainへreconcileする。candidate commit全体のfast-forward / cherry-pickは行わない。

preferred runner exact source blob:

```text
tools/run-first-web-target-r7b-replay.sh
  86c5b9a36a13b43ccded3e6f336effba3bc023f0
  mode = 100755
```

をGitHub candidate objectから3区間で再取得し、Repository外へ同一bytesをmaterializeした。再取得したsourceは495行で、次を確認した。

```text
git hash-object run-first-web-target-r7b-replay.sh
  = 86c5b9a36a13b43ccded3e6f336effba3bc023f0

bash -n run-first-web-target-r7b-replay.sh
  = PASS
```

これによりE1.2で未実施だった **exact hardening runner blob自体のbash syntax rerun** は完了した。

exact source上では引き続き少なくとも次を確認している。

- source checkout clean / normal index check
- actual replay前のfull `tools/repository-health-check.sh`
- health check後のsame revision / clean再確認
- adopted Gate C archive exact SHA-256
- `MANIFEST.sha256` 全件照合
- V2によるplan / receipt / six origin raw Evidence identity preflight
- validated B / R / T / TargetAdapter / Hostだけをrunnerへhandoff
- Repository外detached exact-T worktree
- exact-T Targetを事前buildし、各runでdirect binaryをfresh processとして起動
- 起動前のloopback health endpoint不存在確認
- run 01 / run 02で各6 behaviorを1回ずつcapture
- V2 audit-derived run observation
- exact receipt-bound repeatability aggregation
- current source checkout不変確認
- final receiptへのB / R / T、archive / manifest / plan / receipt / run / summary / log identity binding
- retained output credential scanとscan artifact binding
- final receipt生成後のretained tree再scan

Evidence heightは次へ更新する。

```text
E1.1 early fail-closed smoke PASS
+ E1.2 hardening source candidate
+ preferred runner exact blob identity confirmed
+ E1.2 exact runner bash -n PASS
  != preferred 11-path composition focused Go package PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

残るcode integration gateは:

```text
preferred 11-path composition
  ↓ focused Go package tests
latest main full checkout
  ↓ tools/repository-health-check.sh
same exact working tree
  ↓ direct main commit
```

である。full checkoutを持たないsessionではこの境界を越えてcodeをmainへ入れない。

途中で作成した `3bcc17f0...` / `dad81da8...` はGit-object転送時にPID表現が単独の`$`へ崩れた不採用candidateであり、統合元に使用しない。clean preferred headは `a315e856...` のみとする。

new preferred runner blob `2f55ec06...` はGitHub candidate objectから3区間で再取得し、Repository外へ同一bytesをmaterializeした。再構成後:

```text
line count
  = 496

git hash-object run-first-web-target-r7b-replay.sh
  = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03

bash -n run-first-web-target-r7b-replay.sh
  = PASS
```

を確認した。したがってacquisition identity hardening後のexact runner syntax gateも閉じた。

さらに同じexact bytesをtemporary Git Repositoryへ置き、actual Target / Gate C Evidenceへ進む前のearly fail-closed smokeを再実行した。

```text
required environment variables missing
  -> reject

replay input path inside Repository
  -> reject

replay output path inside Repository
  -> reject

dirty current checkout
  -> reject
```

4件すべてexit non-zeroで拒否した。このsmokeはactual replay / lifecycle Evidenceではないが、E1.1で確認したearly boundaryがnew preferred runnerでも維持されていることを示す。

Evidence heightは:

```text
new preferred runner blob 2f55ec06...
  exact blob identity PASS
  bash -n exact PASS
  early fail-closed smoke 4/4 PASS
  != preferred 11-path composition focused Go package PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

となる。

残るintegration gateは:

```text
preferred 11-path composition
  ↓ focused Go package tests
latest main full checkout
  ↓ tools/repository-health-check.sh
same exact working tree
  ↓ direct main commit
```

である。remote work branchは作らない。

#### E1.5 current-main reconcile / transport contract hardening — 2026-09-20

E1.3 integration packetをcurrent mainへ再reconcileした。

base:

```text
28a5b8398f2d70c0fef6a77dce835faddacf853b
```

current branchless integration candidate:

```text
6c13178c56a8105647f43dd2bdad7a7bf6e20770
```

candidateはbaseに対してahead 1 / behind 0で、変更対象はE1.3と同じ11 source pathだけである。remote branch / PRは新設していない。

再監査でV2 replay auditのtransport fail-closedに1点不足を確認した。従来candidateはrun metadataのObserved HTTPとplanのExpected HTTPが一致することは検証していた一方、**plan側Expected HTTPそのものがcurrent six-behavior contractの200/422 mappingと一致することをpreflightで固定していなかった**。

これを閉じるためV2 ruleへExpected HTTPを追加し、plan validationをexact mappingにした。

```text
first-web/read-only                = 200
first-web/command-success          = 200
first-web/authentication-denied    = 200
first-web/authorization-denied     = 200
first-web/validation-not-satisfied = 200
first-web/execution-error          = 422
```

同時にcurrent replay contractの `RequiredDistinctRuns=2` も `>= 2` ではなくexact 2でpreflight rejectする。

updated exact blobs:

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  38e58c930bd993b44e7f14d408a4cc3b4e1c94d9

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
```

追加test pressure:

- plan Expected HTTP drift
- plan RequiredDistinctRuns drift

runner blobはE1.4でbash syntax PASS済みのexact blob
`86c5b9a36a13b43ccded3e6f336effba3bc023f0`
から変更していない。

Evidence height:

```text
preferred runner exact bash -n PASS
+ current-main 11-path candidate assembled
+ V2 transport/cardinality fail-closed source hardened
  != updated V2 exact candidate focused Go PASS
  != 11-path composition focused Go PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

このsessionのlocal runtimeにはGo 1.23.2があるが、private repositoryのfull checkoutに必要なGitHub DNS / authenticated clone経路がないため、full health checkを代替実行したとは扱わない。

integration sessionではunreferenced candidate SHAをmainへfast-forward / cherry-pickせず、latest mainへ `6c13178c...` の11 source pathをreconcileし、そのexact working treeでfocused Go testsと `tools/repository-health-check.sh` を通してからdirect-to-main commitする。

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

#### E1.5 latest-main reconciled integration candidate — 2026-09-20

並行sessionの文書更新を巻き戻さず、E1.3で固定した11-path packetだけをその時点のlatest mainへreconcileしたbranchless candidateを作成した。

```text
latest main parent
  c185937ab27a3a31332fefcd63ba5940b97b021e
    ↓ 11-path code packet only
reconciled candidate
  44b302a9735d991b7bc669f3faceb4a7ac2cf1bd
```

このcandidateはremote branch / PR / main refを作成・更新していない。

parentからcandidateへの変更pathはexactに次の11件だけである。

```text
tools/internal/firstwebtargetevidence/semantics.go
tools/internal/firstwebtargetevidence/semantics_test.go
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
tools/reference-first-web-target-evidence-audit/main.go
tools/reference-first-web-target-r7b-replay-plan/main.go
tools/reference-first-web-target-r7b-replay-plan/main_test.go
tools/reference-first-web-target-r7b-replay-audit/main.go
tools/reference-first-web-target-r7b-replay-audit/main_test.go
tools/reference-first-web-target-r7b-repeatability/main.go
tools/reference-first-web-target-r7b-repeatability/main_test.go
tools/run-first-web-target-r7b-replay.sh
```

E1.3で固定したexact blob identityをそのまま使用しており、別レーンのdocs / second-application変更はcandidate treeへ巻き戻していない。

cross-session比較では、candidate起点の親mainまでに上記11 pathと重なるmain側変更は0件だった。open PRも0件である。

したがって次のintegration sessionは古いcandidate chainを直接fast-forward / cherry-pickせず、`44b302a9...` の11-path責務を**その時点のlatest main**へ再確認してからqualificationする。

現在のEvidence height:

```text
preferred exact runner bash -n PASS
+ shared validator focused PASS
+ V2 baseline focused PASS
+ strong repeatability focused PASS
+ latest-main reconciled 11-path candidate fixed
  != reconciled candidate focused Go composition PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

このsessionではGitHub connector外のprivate raw download / cloneがnetwork制約で利用できず、Repository内bootstrap payloadもconnector出力のbyte-exact materialization境界を越えられなかったため、full checkout qualificationは未実行のままfail closedとする。full checkoutを持つsessionだけが `tools/repository-health-check.sh` PASS後にmainへ直接code commitする。

#### E1.6 converged current-main composition — latest runner + transport hardening

parallel candidate chainを再統合した。

一方のcandidate:

```text
a315e8566f4784ecc005de209d6a6dae23d5b7cd
```

はacquisition identity collision hardeningを含み、runner exact blob:

```text
2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

についてexact blob identity / `bash -n` / early fail-closed smoke 4/4 PASSが確認済みである。

もう一方のcurrent-main candidate:

```text
6c13178c56a8105647f43dd2bdad7a7bf6e20770
```

はV2 replay auditに次を追加した。

- plan Expected HTTPをcurrent six-behavior contractへexact固定
- `RequiredDistinctRuns == 2` をpreflightでexact要求
- plan HTTP drift / distinct-run drift test pressure

この2系統をlatest main:

```text
6e9c63b429a8f8f174ad24154afa8f3be2890818
```

上で11 source pathだけに再合成した。

current preferred branchless composition candidate:

```text
abd8b748c4ff6c7b4d174d719d8a1be5c46158bb
```

baseに対して:

```text
ahead = 1
behind = 0
changed paths = 11
```

である。remote branch / PR / main refはcandidate作成では変更していない。

current exact path identities:

```text
tools/internal/firstwebtargetevidence/semantics.go
  46629767bee3b3733c0ca6e4433971c4cc85bb9d

tools/internal/firstwebtargetevidence/semantics_test.go
  b94db230f15871a36151f077be8aab13928b9d06

tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6

tools/reference-first-web-target-evidence-audit/main.go
  eb44ef1d543f0849e424e99ed517492f747d634a

tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab

tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed

tools/reference-first-web-target-r7b-replay-audit/main.go
  38e58c930bd993b44e7f14d408a4cc3b4e1c94d9

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99

tools/reference-first-web-target-r7b-repeatability/main.go
  48c9c595311a4fa8fc7c4e7b2a30c4a4fd999965

tools/reference-first-web-target-r7b-repeatability/main_test.go
  1ebe3a7bfed00b38ae7604143ae06f929267f665

tools/run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
  mode = 100755
```

このcandidateではV2がplan自身のtransport contractをpreflightで固定するため:

```text
observed HTTP == plan expected HTTP
  + plan expected HTTP == adopted six-behavior mapping
```

の二段階が成立する。

Evidence height:

```text
preferred runner exact blob / bash -n PASS
+ runner early fail-closed smoke 4/4 PASS
+ V1 shared-validator focused evidence
+ V2 baseline focused evidence
+ strong repeatability focused evidence
+ latest-main 11-path composition assembled
+ transport/cardinality fail-closed source hardened
  != updated V2 exact focused Go PASS
  != full 11-path composition focused Go PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

したがってintegration sessionは `a315e856...` または `6c13178c...` を単独の統合元にせず、**`abd8b748...` の責務 / exact blobsをlatest mainへreconcileする**。

candidate SHAをmainへfast-forward / cherry-pickしない。full checkoutでfocused Go testsと `tools/repository-health-check.sh` を同じexact working treeに対して通した後だけdirect-to-main commitする。

#### E1.7 repeatability summary plan-contract hardening

V2 auditをexact transport contractへ固定した後、repeatability aggregatorも追加監査した。

従来aggregatorは:

```text
run observed HTTP == plan expected HTTP
run Observation class == plan expected Observation class
RequiredDistinctRuns >= 2
```

を確認していたが、plan自身について:

```text
Expected HTTP > 0
Expected Observation class != empty
RequiredDistinctRuns >= 2
```

までしか要求していなかった。

したがってplan / receipt / run observationを相互に整合させて改変した場合、最終summary単体ではcurrent six-behavior contractからのdriftをfail closedできない余地があった。

aggregatorの責務はraw artifactのsemantic再分類ではないが、**採用済みreplay plan contractと一致することの再検証**はsummary生成境界で保持すべきなので、次をexact固定した。

```text
first-web/read-only
  HTTP 200
  read-result

first-web/command-success
  HTTP 200
  business-result-plus-separate-observation

first-web/authentication-denied
  HTTP 200
  business-denial-authentication

first-web/authorization-denied
  HTTP 200
  business-denial-authorization

first-web/validation-not-satisfied
  HTTP 200
  validation-not-satisfied

first-web/execution-error
  HTTP 422
  command-invocation-execution-error

RequiredDistinctRuns
  = 2
```

追加test pressure:

- 全6 behaviorを `RequiredDistinctRuns=3` に揃えたplan / receipt / 3 runをreject
- plan Expected HTTP driftをreject
- plan Expected Observation class driftをreject

updated exact blobs:

```text
tools/reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff

tools/reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
```

repeatability-only branchless candidate:

```text
68ae19d53e2551f0fc36769b35f9ae315f10e276
```

V2 transport/cardinality hardening、latest runner、repeatability hardeningをその時点のlatest main:

```text
172db0df3855a58c9a46ae71cbbfef1f08c489f3
```

へ11 source pathだけ再合成したcurrent preferred candidate:

```text
0ac445c699f8c576ecb51e055aa67622d388df74
```

baseに対して:

```text
ahead = 1
behind = 0
changed paths = 11
```

である。remote branch / PR / main refはcandidate作成で変更していない。

このsessionではGit blob textをローカルへ分割materializeする試行も行ったが、再構成物のbyte countは一致した一方でGit blob SHAがexpected SHAと一致しなかった。connector側では同じcontentのfetch→create round-tripがexpected SHAへ戻るため、ローカル転送過程の転記誤差と判断し、**そのローカル再構成物をfocused PASS Evidenceとして採用していない**。

Evidence height:

```text
preferred runner exact blob / bash -n PASS
+ runner early fail-closed smoke 4/4 PASS
+ V1 shared-validator prior focused evidence
+ V2 baseline prior focused evidence
+ repeatability prior focused evidence
+ V2 transport/cardinality source hardening
+ repeatability plan-contract source hardening
+ latest-main 11-path composition assembled
  != updated V2 exact focused Go PASS
  != updated repeatability exact focused Go PASS
  != full 11-path composition focused Go PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

integration sessionは旧 `abd8b748...` / `68ae19d5...` を個別に統合せず、**`0ac445c6...` の責務 / exact blobsをその時点のlatest mainへreconcileする**。

#### E1.8 latest-main rebase note

E1.7記録後、Second Application側の文書commitがmainへ入ったが、R7B 11 source pathとの重複は0件だった。

そのためE1.7のexact source blobsを変更せず、current main:

```text
7970fa35536df745098e35abf7513ff8ecba7f3a
```

上へ再配置したbranchless candidateを作成した。

```text
5e1357a3e9ca9f0bbdf7226fe31e2b554ef9644a
```

これは新しい設計案ではなく、`0ac445c6...` と同一の11-path責務 / exact source blobsをlatest main history上へ載せ直したものだけである。

remote branch / PRは追加していない。integration時はcandidate SHAの新旧ではなく、E1.6〜E1.7で固定した11 path / exact blob identityを正本としてlatest mainへreconcileする。

#### E1.9 current-main replay packet refresh

E1.8後にSecond Application側の文書commitがさらに1件mainへ入ったが、R7Bの11 source pathとの重複は0件だった。

current main:

```text
004d429b356b0709776c77c369ab183f06b38670
```

へE1.6〜E1.7で固定したexact source blobsを変更せず再配置したbranchless candidate:

```text
12c05a44f835e8db60b81622572f7f87ccbcce2e
```

compare:

```text
ahead = 1
behind = 0
changed paths = 11
```

changed pathはR7B integration packetの11 source pathだけで、plan / docs / Second Application sourceはcandidateへ混入していない。

このrefreshでもremote branch / PRは追加していない。

また主要レーンのうち:

```text
research/selective-knowledge-routing-r6
work/issue-000007-first-web-target
```

は独自ahead commitがないことを確認してcurrent mainへforceなしfast-forwardした。

```text
docs/design-synthesis-local-cognition
work/issue-000040-second-application-reference
```

は既にcurrent mainとidenticalだった。

Evidence heightは変えない。

```text
latest-main 11-path candidate assembled
  != updated V2 exact focused Go PASS
  != updated repeatability exact focused Go PASS
  != full 11-path composition focused Go PASS
  != repository-health-check PASS
  != integrated
```

integration sessionはunreferenced candidate SHAそのものをmainへfast-forward / cherry-pickせず、latest mainへexact 11 blobsをreconcileし、同一working treeでfocused testsとfull repository health checkを通した後だけcodeをmainへ入れる。

#### E1.10 exact focused verification and gofmt convergence — 2026-09-20

E1.9 candidateを再監査したところ、V2 replay auditのsource blob:

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  38e58c930bd993b44e7f14d408a4cc3b4e1c94d9
```

は機能test自体は通る一方、exact bytesが `gofmt -l` cleanではなかった。full Repository qualificationでは `gofmt -l .` が必須なので、このblobをそのままintegration sourceには採用しない。

意味変更なしでgofmt相当の整形だけを行ったreplacement blob:

```text
tools/reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da
```

について、Repository外isolated moduleで:

```text
git hash-object main.go
  = a370e0c5efc4399e324be7a0ad4801626f05a5da

gofmt -l
  = empty

go vet
  = PASS

go test -count=50
  = PASS
```

を確認した。V2 test blob `2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99` は変更しない。

同時にE1.7で追加されたrepeatability plan-contract hardeningのexact blobs:

```text
tools/reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff

tools/reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
```

をGitHub objectからRepository外へbyte-for-byte再構成し、両方の `git hash-object` が上記identityと一致することを確認したうえで:

```text
gofmt -l .
  = empty

go vet ./...
  = PASS

go test -count=50 ./...
  = PASS
```

を実行した。

これによりE1.7のcurrent six-behavior repeatability contract:

- six Behavior ID exact set
- HTTP 200 / 422 exact mapping
- Observation class exact mapping
- `RequiredDistinctRuns = 2`
- distinct run identity
- distinct Target process identity
- exact plan / receipt binding
- run provenance / loopback binding
- raw digestの形式検証
- raw digestのrun間一致は要求しない

についてfocused verificationを得た。

その後current main:

```text
2ee13811c04200cf7df86d61ece883349dc499a3
```

を親に、E1.6〜E1.7の11 source pathを再配置し、V2 mainだけをgofmt-clean replacement blobへ差し替えたbranchless candidateを作成した。

```text
f903daef60c5a786192e52b0d1ce1d64fccb2319
```

compare:

```text
ahead = 1
behind = 0
changed paths = 11
```

current exact integration source identitiesは:

```text
tools/internal/firstwebtargetevidence/semantics.go
  46629767bee3b3733c0ca6e4433971c4cc85bb9d
tools/internal/firstwebtargetevidence/semantics_test.go
  b94db230f15871a36151f077be8aab13928b9d06
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6
tools/reference-first-web-target-evidence-audit/main.go
  eb44ef1d543f0849e424e99ed517492f747d634a
tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
tools/reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da
tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
tools/reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff
tools/reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
tools/run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

remote branch / PR / main refはcandidate作成で変更していない。

Evidence heightを更新する。

```text
V1 shared-validator exact focused PASS
V2 formatted exact focused PASS
repeatability hardened exact focused PASS
runner exact blob / bash -n PASS
runner early fail-closed smoke 4/4 PASS
latest-main 11-path candidate assembled
  != Gate C audit + shared validator current full-package PASS
  != full 11-path composition focused Go PASS
  != repository-health-check PASS
  != integrated
  != replay executed
  != repeatability observed
```

次のintegration gateはGate C auditの既存test suiteをshared validator抽出後のexact sourceと同時に実行し、その後11-path composition全体をfull checkoutへ載せて `tools/repository-health-check.sh` を通すことである。

#### E1.11 cross-lane branch convergence and V1 semantic-equivalence audit — 2026-09-20

並行chat / laneを再監査した。current mainは:

```text
7e192a9707bdc70595eb68c08d846026e56b99d7
```

であり、E1.10後の増分はSecond Application側の文書のみだった。R7B 11 source pathとの重複は0件である。

remote branch全18本をcurrent mainと比較したところ、作業開始時点では:

- 4本 = current mainとidentical
- 14本 = branch固有ahead commit 0、current mainより3〜4 commit behind

だった。14本はいずれも未統合作業を含まないため、forceを使わずfast-forwardでcurrent mainへ収束させた。

再監査結果:

```text
remote branch count excluding main = 18
branch unique commits               = 0
behind main                         = 0
status                              = all identical to current main
force update                        = none
new remote branch / PR              = none
```

したがって、過去branch refは残っていても、現在は分岐した作業履歴を指していない。今後も新しいremote work branchは作らずdirect-to-main policyを維持する。

R7B sourceについてはE1.10と同一の11 exact blobsをcurrent mainへ再配置したbranchless candidate:

```text
0fe69628464a1f153813333a5ae730a5cba64a02
```

を作成した。これはcurrent mainに対してahead 1で、11 source path以外を変更しない。

またV1 shared validator抽出について、candidate導入前のGate C audit semantic blockとshared packageを直接照合した。

旧Gate C auditが保持していた6 behavior条件:

1. read: exact provenance / Scenario / Application / runtime bindings
2. command success: Scenario / stage / Authentication / 2 Validations / Authorization / Command Business Result / separate Observation
3. authentication denied: stage / denied outcome / Commandなし / Observationなし
4. authorization denied: stage / denied outcome / Commandなし / Observationなし
5. validation not-satisfied: exact Validation identity/result / Authorizationなし / Commandなし / Observationなし
6. execution error: command stage / Authorization allowed / Commandなし / Observationなし / exact synthetic adapter error fragment

は、`tools/internal/firstwebtargetevidence/semantics.go` の対応validatorへ同一条件として移されていることを確認した。

Gate C audit candidate側は6 raw JSONをshared validatorへ渡し、その後 `read.json` をTargetDifferenceHandoff生成のためだけにGate C-local shapeへ再decodeする。semantic acceptance ruleをGate C側で二重定義していない。

したがって静的照合ではV1抽出によるsemantic条件の欠落・追加は確認されない。ただしこれは既存Gate C test suiteをcandidate sourceと同時実行した結果ではないため、Evidence heightは:

```text
V1 old/new semantic condition equivalence
  = static exact-source audit PASS
  != Gate C audit + shared validator current full-package test PASS
```

とする。

#### E1.12 inherited focused composition PASS by exact blob identity — 2026-09-20

他チャットで先行取得されていたbranchless candidate:

```text
589d6c57351f7ec5bd0ef7bc9ba1b575338d04ee
parent = 172db0df3855a58c9a46ae71cbbfef1f08c489f3
```

では、shared validator + Gate C audit + E0 replay-plan のfocused compositionについて、対象source blobをGit identityで照合した後に:

```text
gofmt
  = PASS

go vet
  = PASS

go test -count=10
  = PASS
```

が実行済みだった。

この検証対象の7 filesをcurrent R7B candidateと照合したところ、全blobがbyte-identicalだった。

```text
tools/internal/firstwebtargetevidence/semantics.go
  46629767bee3b3733c0ca6e4433971c4cc85bb9d
tools/internal/firstwebtargetevidence/semantics_test.go
  b94db230f15871a36151f077be8aab13928b9d06
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6
tools/reference-first-web-target-evidence-audit/main.go
  eb44ef1d543f0849e424e99ed517492f747d634a
tools/reference-first-web-target-evidence-audit/main_test.go
  668efcdb084e9a7cefe050a5df4be86effc1cebb
tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
```

さらに検証時parent `172db0df...` からcurrent main:

```text
2f052468a5aad11df405452ec9c480df1c158ab6
```

までをcompareした結果、changed pathは次の4つのMarkdown計画文書だけだった。

```text
plan/SECOND_APPLICATION_S5_EXECUTABLE_PRESSURE_IMPLEMENTATION_PACKET_2026-09.md
plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md
plan/issue/000040-semantic-component-composition.md
plan/issue/000069-first-web-target-r7b-exact-t-replay.md
```

Go / shell / runtime sourceおよび上記7 exact blobsのdependency surfaceは変更されていない。

したがって、E1.11でstatic auditに留めていたGate C / V1境界は、byte identityとsource-dependency不変性により次のEvidenceとして継承できる。

```text
V1 shared validator + Gate C audit + E0 replay-plan
  = exact focused composition gofmt PASS
  = exact focused composition go vet PASS
  = exact focused composition go test -count=10 PASS
```

これはcurrent full Repository healthを意味しない。残るgateは:

```text
V2 formatted exact focused PASS
repeatability hardened exact focused PASS
runner exact blob / bash -n PASS
runner early fail-closed smoke 4/4 PASS
V1 + Gate C + E0 exact focused composition PASS
  != full 11-path composition focused Go PASS
  != tools/repository-health-check.sh PASS
  != integrated
  != replay executed
  != repeatability observed
```

である。

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

Source / Design段階だけでなく、E0 / V1 / V2 / repeatability / E1 runnerのbranchless source candidateも揃っている。canonical integration sourceはE1.13の11 path / blob identityで固定され、dependency closure単位のfocused Evidenceも取得済みである。

したがって現在の最初のcode stopは**E0の新規実装ではない**。Go-enabled full checkoutでcanonical 11 sourceをcurrent mainへrestoreし、E1.14のaggregate focused commandと `tools/repository-health-check.sh` を**同一working tree**でPASSさせることが最初の未完了gateである。

このgateを閉じた後に、その同じworking treeをdirect-to-mainでcommitする。remote work branch / PRを新設せず、別candidateのcherry-pickで履歴を再分岐させない。

code integration後の次段はactual Evidence acquisitionである。

```text
integrated canonical 11 source
  -> adopted Gate C Release archive再取得 / digest照合
  -> actual R7A handoff materialization
  -> actual R7B intake materialization
  -> integrated E0 sourceによるR7B replay plan materialization
  -> exact-T detached target
  -> two fresh-process runs
  -> V2 run observations
  -> repeatability summary
  -> final receipt / retained artifact credential scan
```

actual exact-T replay / repeatability Evidenceはこの2-run executionが完了するまで未成立とする。

このIssueはこのhealth integration gateから先を所有する。


#### E1.13 exact-source convergence and branch hygiene — 2026-09-20

並行laneで得られたfocused Evidenceをexact Git blob identityで再収束した。current canonical integration sourceは、candidate commit SHAそのものではなく次の11 path / blob identityを正本とする。

```text
tools/internal/firstwebtargetevidence/semantics.go
  46629767bee3b3733c0ca6e4433971c4cc85bb9d
tools/internal/firstwebtargetevidence/semantics_test.go
  b94db230f15871a36151f077be8aab13928b9d06
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6
tools/reference-first-web-target-evidence-audit/main.go
  eb44ef1d543f0849e424e99ed517492f747d634a
tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
tools/reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da
tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
tools/reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff
tools/reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
tools/run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
  mode = 100755
```

V2では旧blob `38e58c93...` をgofmt非cleanのため統合元から外し、意味不変のgofmt-clean replacement `a370e0c5...` だけを採用する。

現在までにexact identityで継承できるfocused Evidenceは:

```text
V1 shared validator + Gate C audit + E0 replay-plan
  gofmt PASS
  go vet PASS
  go test -count=10 PASS

V2 formatted exact source
  gofmt clean
  go vet PASS
  go test -count=50 PASS

repeatability hardened exact source
  gofmt clean
  go vet ./... PASS
  go test -count=50 ./... PASS

runner exact source
  bash -n PASS
  early fail-closed smoke 4/4 PASS
```

remote branchをcurrent mainと再比較した結果、main以外18本はいずれも固有ahead commit 0であり、全branchがcurrent mainとidenticalへ収束している。force update、新規remote branch、open PRはいずれもない。

したがって残るintegration gateは:

```text
current exact 11-path source set
  -> same working treeでfull focused composition
  -> tools/repository-health-check.sh
  -> same revision / clean working tree再確認
  -> direct-to-main code commit
```

である。full checkoutを持たないsessionではfocused Evidenceをfull Repository healthへ読み替えず、codeをmainへ統合しない。actual exact-T replay / repeatability Evidenceも引き続き未取得として扱う。

#### E1.14 package-closure proof and health execution handoff — 2026-09-21

canonical 11-path source setが触るGo packageは5つだけである。

```text
P1  tei/tools/internal/firstwebtargetevidence
P2  tei/tools/reference-first-web-target-evidence-audit
P3  tei/tools/reference-first-web-target-r7b-replay-plan
P4  tei/tools/reference-first-web-target-r7b-replay-audit
P5  tei/tools/reference-first-web-target-r7b-repeatability
```

candidate source / testのimport graphをexact sourceから再確認した。

```text
P1 -> standard library only
P2 -> P1 + standard library
P3 -> standard library only
P4 -> P1 + standard library
P5 -> standard library only

all candidate *_test.go:
  same-package tests
  no candidate cross-package import beyond package-under-test
```

したがってcandidate内部の非標準依存edgeは:

```text
P2 -> P1
P4 -> P1
```

の2本だけであり、cycleはない。

すでにexact blob identityで取得済みのfocused Evidenceはdependency closure単位で:

```text
{P1, P2, P3}
  gofmt PASS
  go vet PASS
  go test -count=10 PASS

{P1, P4}
  gofmt clean
  go vet PASS
  go test -count=50 PASS

{P5}
  gofmt clean
  go vet PASS
  go test -count=50 PASS

runner
  bash -n PASS
  early fail-closed smoke 4/4 PASS
```

である。全5 packageは少なくとも一度、自身のcandidate-internal dependencyを含むclosureでactual compile/test済みである。

この事実はfull aggregate commandの実行結果そのものではないため:

```text
package-closure static composition
  = PASS

single aggregate focused command
  = pending
```

と区別する。

health-capable full checkoutでは、current mainへcandidateを接続する前に次を同一working treeで順に実行する。

```bash
git pull --rebase origin main

git restore --source=99eb861e71cfb225cf2f2ccf3db1f96feb716645 -- \
  tools/internal/firstwebtargetevidence/semantics.go \
  tools/internal/firstwebtargetevidence/semantics_test.go \
  tools/internal/firstwebtargetevidence/semantics_pressure_test.go \
  tools/reference-first-web-target-evidence-audit/main.go \
  tools/reference-first-web-target-r7b-replay-plan/main.go \
  tools/reference-first-web-target-r7b-replay-plan/main_test.go \
  tools/reference-first-web-target-r7b-replay-audit/main.go \
  tools/reference-first-web-target-r7b-replay-audit/main_test.go \
  tools/reference-first-web-target-r7b-repeatability/main.go \
  tools/reference-first-web-target-r7b-repeatability/main_test.go \
  tools/run-first-web-target-r7b-replay.sh

test "$(git diff --name-only | wc -l)" -eq 11

test -z "$(gofmt -l \
  tools/internal/firstwebtargetevidence \
  tools/reference-first-web-target-evidence-audit \
  tools/reference-first-web-target-r7b-replay-plan \
  tools/reference-first-web-target-r7b-replay-audit \
  tools/reference-first-web-target-r7b-repeatability)"

go vet \
  ./tools/internal/firstwebtargetevidence \
  ./tools/reference-first-web-target-evidence-audit \
  ./tools/reference-first-web-target-r7b-replay-plan \
  ./tools/reference-first-web-target-r7b-replay-audit \
  ./tools/reference-first-web-target-r7b-repeatability

go test -count=20 \
  ./tools/internal/firstwebtargetevidence \
  ./tools/reference-first-web-target-evidence-audit \
  ./tools/reference-first-web-target-r7b-replay-plan \
  ./tools/reference-first-web-target-r7b-replay-audit \
  ./tools/reference-first-web-target-r7b-repeatability

bash -n tools/run-first-web-target-r7b-replay.sh

tools/repository-health-check.sh
```

ここで `99eb861e...` はsource-relevant base `40209d25...` 直上のbranchless composition candidateであり、後続のdocs-only commitだけではcanonical 11 blobのfreshnessを失わない。統合前には11 pathのblob identityをE1.13のcanonical listと再照合する。

focused aggregate PASSだけではmain統合しない。必須条件は:

```text
aggregate focused PASS
AND repository-health-check PASS
AND candidate 11 blob identity exact
AND no overlapping source change since health run
AND no new remote work branch / open PR conflict
```

である。

direct-to-main時は同じworking treeをcommitし、別candidateへのcherry-pickやremote branch作成を行わない。


#### E1.15 single aggregate focused composition PASS — 2026-09-21

E1.14でpendingとしていたsingle aggregate focused commandを、canonical sourceとGit blob identityが一致するRepository外focused working treeで実行した。

検証treeへ配置したchanged Go source 10 filesと既存Gate C audit test 1 fileについて、実行直前に `git hash-object` を再計算し、E1.13のcanonical identityおよび既存Gate C test identityと11/11一致することを確認した。

```text
semantics.go
  46629767bee3b3733c0ca6e4433971c4cc85bb9d
semantics_test.go
  b94db230f15871a36151f077be8aab13928b9d06
semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6
reference-first-web-target-evidence-audit/main.go
  eb44ef1d543f0849e424e99ed517492f747d634a
reference-first-web-target-evidence-audit/main_test.go
  668efcdb084e9a7cefe050a5df4be86effc1cebb
reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da
reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff
reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
```

同じworking tree上でE1.14と同じ5 packageを一括指定し:

```text
gofmt -l
  = empty

go vet
  = PASS

go test -count=20
  = PASS
```

を得た。

package別test result:

```text
tei/tools/internal/firstwebtargetevidence
  PASS
tei/tools/reference-first-web-target-evidence-audit
  PASS
tei/tools/reference-first-web-target-r7b-replay-plan
  PASS
tei/tools/reference-first-web-target-r7b-replay-audit
  PASS
tei/tools/reference-first-web-target-r7b-repeatability
  PASS
```

またE1.14のhandoff candidate `99eb861e71cfb225cf2f2ccf3db1f96feb716645` と、独立にlatest source-relevant base上で再構成した `f265dab08e5196ef75220cb5580b65358eaff2d6` をGit treeで比較したところ、両者は同一tree:

```text
6fdaf751cf9aeb1ffe9c31efc7ec183328602459
```

を指し、canonical 11 pathのblob identityも全件一致した。したがってbranchless candidate chain間のsource driftはない。

Evidence heightを更新する。

```text
canonical 11-path exact source fixed
+ V1 / Gate C / E0 focused PASS
+ V2 formatted focused PASS
+ repeatability hardened focused PASS
+ runner bash -n / early smoke PASS
+ single aggregate 5-package focused PASS
  != tools/repository-health-check.sh PASS
  != integrated
  != replay executed
  != repeatability observed
```

このaggregate PASSはRepository外のfocused moduleであり、full Repository healthを意味しない。mainへのcode integration条件は引き続き、Go-enabled full checkoutでcanonical 11 sourceをcurrent mainへ配置した**同じworking tree**に対する `tools/repository-health-check.sh` PASS、source overlapなし、clean/same-revision確認である。


#### E1.16 health integration environment boundary — 2026-09-21

E1.15でsingle aggregate focused compositionはPASS済みとなった。current main `f57efd01fae19362b36e21ff7513db8469d277d7` を親に、E1.13 canonical 11 sourceをそのまま配置した最新branchless candidateを再構成した。

```text
candidate:
  86902a74678756318c554da041a705c1d71ae5f0

parent:
  f57efd01fae19362b36e21ff7513db8469d277d7

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

E1.15のsource-relevant base `40209d25e36b0709776c77c369ab183f06b38670` からcurrent mainまでを比較すると、変更は次の4 docs pathだけであり、canonical 11 source pathとのoverlapは0である。

```text
plan/SECOND_APPLICATION_S5_EXECUTABLE_PRESSURE_IMPLEMENTATION_PACKET_2026-09.md
plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md
plan/issue/000040-semantic-component-composition.md
plan/issue/000069-first-web-target-r7b-exact-t-replay.md
```

remote branch 18本をcurrent mainと比較した結果、全件:

```text
ahead  = 0
behind = 0
status = identical
```

であり、open PRも0。新しいremote branch / PRは作成していない。

現在のsessionではfull Repository healthを実行できない理由を実行環境側で再確認した。

```text
container DNS:
  github.com                    resolution failure
  api.github.com                resolution failure
  raw.githubusercontent.com     resolution failure

private git clone / ls-remote:
  unavailable because github.com cannot be resolved

GitHub-hosted Actions:
  .github/workflows/knowledge-portability-reference-evidence.yml
  が runner allocation unavailable / runner_id=0 を明記
```

`tools/repository-health-check.sh` のactual contractも再確認した。full checkoutで必要なのは:

```text
gofmt -l .
go vet ./...
go test -count=1 ./...
go run ./cmd/tei validate -workspace reference/workspace
```

であり、focused moduleのPASSをこれへ読み替えることはしない。

したがってcurrent Evidence heightは:

```text
canonical 11-path exact source fixed
+ all package dependency closures focused PASS
+ single aggregate 5-package focused PASS
+ runner bash -n / early fail-closed smoke PASS
+ no source overlap since focused source-relevant base
+ all remote branches converged / open PR 0
  != repository-health-check PASS
  != code integrated
  != replay executed
  != repeatability observed
```

次のhealth-capable full checkoutでは、current mainをpull/rebaseした後、candidate commit自体をcherry-pickせずE1.13 canonical 11 blobsを同一working treeへrestoreし、E1.14のaggregate focused commandと `tools/repository-health-check.sh` を実行する。両方PASSし、実行前後でsame revision / clean base / source overlapなしを確認した場合のみ、その同じworking treeをdirect-to-main commitする。


#### E1.17 credential-safe canonical packet correction — 2026-09-21

E1.16のhealth handoffを再監査し、runnerのretained-output credential scanとshared validatorの禁止集合に1点のdriftを確認した。

runnerは `wrong-secret` をReference credential materialとして拒否する一方、E1.13 canonical P1 sourceではこの値だけが `RejectReferenceCredentialMaterial` の禁止集合から欠落していた。Authentication denial用のTarget-local runtime credentialであり、retained Evidenceへ残してよい値ではないため、canonical packetを次の2 blobだけ更新する。

```text
tools/internal/firstwebtargetevidence/semantics.go
  old 46629767bee3b3733c0ca6e4433971c4cc85bb9d
  new 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

tools/internal/firstwebtargetevidence/semantics_test.go
  old b94db230f15871a36151f077be8aab13928b9d06
  new ba17f69a7e375cfdeec0db544986d51f99b26192
```

Git object同士のexact diffは局所的である。

```text
semantics.go:
  + "wrong-secret",

semantics_test.go:
  forbidden value tableへ "wrong-secret" を1件追加
```

他のP1/P2/P3/P4/P5 sourceとrunner blobはE1.16から変更しない。

credential rejection境界については新 `semantics.go` exact blobに対するtargeted verificationでgofmt clean / go vet PASS / repeated Go test PASSを取得済みである。E1.15のaggregate focused PASSは旧P1 blobに対するものであるため、このdelta proofをもってfull aggregate rerunやfull Repository health PASSへ読み替えない。

current main:

```text
085b4aa15fe9da1423b639bedbad4022ff4be474
```

を親に、credential-safe P1とE1.16の残りcanonical sourceを再配置したbranchless candidate:

```text
052285f526f8590c7dc03c76a41acabaeea5f4bf
```

を作成した。candidate vs parentはahead 1 / behind 0、changed pathsはcanonical 11 source pathsだけである。remote branch / PR / main refはcandidate作成では変更していない。

E1.17以降のcanonical integration sourceは次とする。E1.13/E1.16のP1先頭2 blob identityはこの一覧で上書きする。

```text
tools/internal/firstwebtargetevidence/semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
tools/internal/firstwebtargetevidence/semantics_test.go
  ba17f69a7e375cfdeec0db544986d51f99b26192
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6
tools/reference-first-web-target-evidence-audit/main.go
  eb44ef1d543f0849e424e99ed517492f747d634a
tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
tools/reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da
tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
tools/reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff
tools/reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
tools/run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
  mode = 100755
```

integration gateは引き続き変えない。

```text
credential-safe canonical 11 source
  -> same working treeでaggregate focused rerun
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

full checkoutを持たないsessionではここを越えてcodeをmainへ統合しない。actual exact-T replay / repeatability Evidenceも未取得のままとする。


#### E1.18 credential-set parity proof — 2026-09-21

E1.17で更新したshared validator exact blobとcurrent runner exact blobをGit objectから直接取得し、credential literal集合を機械比較した。

```text
runner:
  credential-ref://legacy/
  disabled-secret
  editor-secret
  locked-secret
  viewer-secret
  wrong-secret

shared RejectReferenceCredentialMaterial:
  credential-ref://legacy/
  disabled-secret
  editor-secret
  locked-secret
  viewer-secret
  wrong-secret
```

結果:

```text
missing in validator = 0
extra in validator   = 0
set equality         = PASS
```

runner request pathでは `editor-secret` / `wrong-secret` / `viewer-secret` がruntime inputとして実使用され、retained-output final scanは上記6値すべてを拒否する。したがってE1.17後はruntime credential集合とretained Evidence禁止集合のdriftを解消している。

このparity proofはstatic exact-blob contract Evidenceであり、credential-safe canonical packetに対するaggregate focused rerunやfull Repository health PASSの代替ではない。integration gateはE1.17のまま維持する。

#### E1.20 Gate C credential rejection single-source correction — 2026-09-21

E1.17 / E1.18後にGate C audit candidateのcredential scanを再監査し、追加のdriftを確認した。

Gate C auditは6 behaviorのsemantic validationをshared `firstwebtargetevidence` packageへ委譲している一方、retained artifact credential scanだけは旧ローカルhelperを保持していた。そのローカル禁止集合は:

```text
editor-secret
viewer-secret
locked-secret
disabled-secret
credential-ref://legacy/
```

であり、Authentication denialのruntime credential `wrong-secret` を含んでいなかった。

したがってE1.18で証明したrunner/shared set equalityだけでは、Gate C retained-artifact scanまで単一化できていなかった。

修正はGate C auditのhelper shapeを維持し、各artifact bytesをshared rejectorへ渡すだけに限定する。

```go
func requireNoReferenceCredentialMaterial(dir string, names []string) error {
    for _, name := range names {
        data, err := os.ReadFile(filepath.Join(dir, name))
        if err != nil {
            return fmt.Errorf("read %s for credential scan: %w", name, err)
        }
        if err := firstwebtargetevidence.RejectReferenceCredentialMaterial(name, data); err != nil {
            return err
        }
    }
    return nil
}
```

これによりcredential禁止集合の正本は:

```text
tools/internal/firstwebtargetevidence/semantics.go
  RejectReferenceCredentialMaterial
```

だけになる。

exact blob更新:

```text
tools/reference-first-web-target-evidence-audit/main.go
  old eb44ef1d543f0849e424e99ed517492f747d634a
  new 3878dae15b06a3b57954cf196b108da1cc03bbc7
```

P1 canonical blobsはE1.17のcredential-safe版を維持する。

```text
tools/internal/firstwebtargetevidence/semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
tools/internal/firstwebtargetevidence/semantics_test.go
  ba17f69a7e375cfdeec0db544986d51f99b26192
```

残る8 Go source/test blobsとrunner blobはE1.17から不変である。

current main `ad5b97f9d340b4d40db316396ef477ad285316de` を親に、credential-safe P1 + Gate C delegation +残りcanonical sourceを再配置したbranchless candidate:

```text
92677eb4a37d31a8405e2971244a46ed79fe32ef
```

を作成した。candidateはparentに対しahead 1 / behind 0、changed pathsはcanonical 11 source pathsのみである。remote branch / PR / main refはcandidate作成では変更していない。

Gate C helperのexact sourceとshared function signatureはGit objectから再取得して一致を確認した。

```text
Gate C local credential literal list
  = removed

Gate C shared reject calls
  = 1

shared forbidden values
  = editor-secret
    viewer-secret
    wrong-secret
    locked-secret
    disabled-secret
    credential-ref://legacy/
```

この修正はsource contract上の単一正本化であり、まだaggregate focused rerunではない。E1.15のaggregate PASSは旧P1 / 旧Gate C blobに対するものなので、integration gateは次へ更新する。

```text
E1.20 canonical 11 source
  -> same working treeでaggregate focused rerun
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

full checkoutを持たないsessionではこのgateを越えてcodeをmainへ統合しない。actual exact-T replay / repeatability Evidenceも未取得のままとする。


#### E1.19 adopted Gate C Release origin metadata audit — 2026-09-21

Packet Mのactual materialization前に、runbookが固定しているGate C origin identityをGitHub Release / Git refの現物metadataから再照合した。

GitHub Release:

```text
release id:
  392136119

tag:
  gate-c-evidence-T-0ba87a4

release target_commitish:
  0ba87a4e0a96f70a198b03ff7652fb00e3826abe

release name:
  Gate C Evidence (T = 0ba87a4)

draft:
  false

prerelease:
  true

published_at:
  2026-09-19T16:27:40Z
```

Git ref:

```text
refs/tags/gate-c-evidence-T-0ba87a4
  -> object type = commit
  -> object sha  = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe
```

annotated tag objectを介さず、Release tag refがexact Target build `T` commitを直接指している。

Release asset:

```text
asset id:
  575087696

name:
  tei-gate-c-evidence-T-0ba87a4e.tar.gz

size:
  961001 bytes

state:
  uploaded

content type:
  application/x-gtar

GitHub asset digest:
  sha256:de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3
```

この現物metadataはPacket M runbookの:

```text
gh release download gate-c-evidence-T-0ba87a4
  --pattern tei-gate-c-evidence-T-0ba87a4e.tar.gz

expected archive SHA-256:
  de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3
```

と一致した。

特にtag suffixとasset filename suffixは同じ文字列ではない。

```text
Release tag:
  ...T-0ba87a4

asset filename:
  ...T-0ba87a4e.tar.gz
```

M1はこの差を既に正しく扱っているため、asset名をtag名として使用する変更は行わない。

Evidence境界は分離する。

```text
Release metadata / tag ref / advertised asset digest equality
  = PASS

binary asset body re-download
  = NOT EXECUTED

downloaded bytes SHA-256 recomputation
  = NOT EXECUTED

expanded MANIFEST.sha256 verification
  = NOT EXECUTED

Packet M materialization
  = NOT EXECUTED
```

したがってこのpreflightはM1の入力identityを確定するが、M1 completionやactual Gate C archive qualificationそのものではない。

health integration gateも変更しない。credential-safe canonical 11 sourceについてaggregate focused rerunと `tools/repository-health-check.sh` を同一full checkoutでPASSさせ、code integrationを完了した後に、このverified Release identityを使ってPacket M actual materializationへ進む。


#### E1.21 credential pressure parity correction — 2026-09-21

E1.20でproduction側のcredential rejectorはsingle sourceへ収束したが、V1 negative matrixを担う `semantics_pressure_test.go` をexact blobで再監査すると、credential reject pressureだけがE1.17以前の5値集合を保持していた。

旧pressure set:

```text
editor-secret
viewer-secret
locked-secret
disabled-secret
credential-ref://legacy/
```

production shared rejector / runner / ordinary shared testのcurrent setには:

```text
wrong-secret
```

も含まれる。

したがってproduction behaviorのdriftではないが、**negative pressure contractの1値欠落**が残っていた。

修正はtest-only 1行追加に限定する。

```text
tools/internal/firstwebtargetevidence/semantics_pressure_test.go

old:
  ce77296ebd8acccc386c4bab8bf819f9a81ae3d6

new:
  95553ba10756399a50e52e639f7d860f4420f29f

delta:
  + "wrong-secret",
```

これによりcredential contractの3面が同じ6値集合になる。

```text
runner retained-output scan
shared RejectReferenceCredentialMaterial
V1 credential negative pressure

=
  credential-ref://legacy/
  disabled-secret
  editor-secret
  locked-secret
  viewer-secret
  wrong-secret
```

E1.20のGate C delegation blobとE1.17 credential-safe P1を含め、current main `eb9335f48e225612d9f4dd6183ca8adeceea83b2` を親にcanonical 11 sourceを再構成したbranchless candidate:

```text
2e2b7d063370eb468c8f86e64df7843b86f77e1e
```

を作成した。candidateはremote branch / PR / main refを変更していない。

E1.21以降、canonical credential-related identitiesは:

```text
tools/internal/firstwebtargetevidence/semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

tools/internal/firstwebtargetevidence/semantics_test.go
  ba17f69a7e375cfdeec0db544986d51f99b26192

tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  95553ba10756399a50e52e639f7d860f4420f29f

tools/reference-first-web-target-evidence-audit/main.go
  3878dae15b06a3b57954cf196b108da1cc03bbc7
```

とする。残りcanonical blobsはE1.20から不変である。

この変更はtest-only pressure correctionであり、まだcredential-safe / single-source canonical packet全体のaggregate focused rerunではない。

```text
exact pressure-set parity
  = PASS by source inspection

aggregate focused rerun
  = pending

tools/repository-health-check.sh
  = pending

code integration
  = pending

actual replay / repeatability
  = pending
```

integration gateはさらに増やさず、E1.20と同じ一本道を維持する。

```text
E1.21 canonical 11 source
  -> same working treeでaggregate focused rerun
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```


#### E1.22 latest-main credential-safe refresh and branch convergence — 2026-09-21

E1.21 canonical source以後、mainは5 commit進んだ。E1.21 candidate parent:

```text
eb9335f48e225612d9f4dd6183ca8adeceea83b2
```

からcurrent main:

```text
775a9b4b0d6a3a4de145c7ce3c97195c1f40c80c
```

までを比較したところ、canonical 11 source pathとのoverlapは0件だった。増分はR7B文書、repository plan check、Lane C / Definition-Centered Feedback Loop sourceなどであり、R7B replay packetの11 pathは変更していない。

したがってE1.21のexact blobsをcurrent mainへそのまま再配置したbranchless candidateを作成した。

```text
candidate:
  2fb101ac74dfc854a5807c8e8fd049add7ede888

parent:
  775a9b4b0d6a3a4de145c7ce3c97195c1f40c80c

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

E1.22 current canonical identities:

```text
tools/internal/firstwebtargetevidence/semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

tools/internal/firstwebtargetevidence/semantics_test.go
  ba17f69a7e375cfdeec0db544986d51f99b26192

tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  95553ba10756399a50e52e639f7d860f4420f29f

tools/reference-first-web-target-evidence-audit/main.go
  3878dae15b06a3b57954cf196b108da1cc03bbc7

tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab

tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed

tools/reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99

tools/reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff

tools/reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74

tools/run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
  mode = 100755
```

remote branch全18本もcurrent mainと再比較した。全件でbranch固有ahead commitは0だったため、forceを使わずcurrent mainへfast-forwardした。

```text
remote branch count excluding main = 18
unique ahead commits                = 0
after convergence:
  ahead  = 0
  behind = 0
  status = identical
force update                        = none
new remote work branch              = none
open PR                             = 0
```

E1.21でcredential-related sourceが更新されたため、E1.15のaggregate focused PASSをcurrent canonical packet全体のPASSへ読み替えない。現在の最初の未完了gateは引き続き:

```text
E1.22 canonical 11 source
  -> same working treeでaggregate focused rerun
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

である。

このsessionではGo 1.23.2のlocal runtimeは利用できる一方、private repository full checkoutを取得できるauthenticated clone / GitHub-hosted runner経路がない。GitHub-hosted workflowもrunner allocation unavailableを明記している。

したがって:

```text
latest-main exact candidate assembly
  = PASS

source overlap check
  = PASS

branch convergence
  = PASS

E1.21 aggregate focused rerun
  = pending

tools/repository-health-check.sh
  = pending

code integration
  = pending

actual replay / repeatability
  = pending
```

とする。candidate SHAそのものをmainへfast-forward / cherry-pickせず、health-capable integration sessionではE1.22 exact blobsをlatest mainへrestoreして同一working tree上でqualificationする。
