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


#### E1.23 E1.21 P1 exact rerun — 2026-09-21

E1.17 / E1.21で更新されたP1 shared validator 3 blobsをGitHub objectからRepository外の一時Go moduleへ再構成し、実行直前に `git hash-object` を再計算した。

```text
tools/internal/firstwebtargetevidence/semantics.go
  expected = 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
  actual   = 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

tools/internal/firstwebtargetevidence/semantics_test.go
  expected = ba17f69a7e375cfdeec0db544986d51f99b26192
  actual   = ba17f69a7e375cfdeec0db544986d51f99b26192

tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  expected = 95553ba10756399a50e52e639f7d860f4420f29f
  actual   = 95553ba10756399a50e52e639f7d860f4420f29f
```

3/3 exact identity一致後、同じworking treeで:

```text
gofmt -l tools/internal/firstwebtargetevidence
  = empty

go vet ./tools/internal/firstwebtargetevidence
  = PASS

go test -count=50 ./tools/internal/firstwebtargetevidence
  = PASS
```

を取得した。

これによりE1.21のcredential pressure correctionを含むP1 exact packageについてfocused Evidenceを更新できた。

ただしE1.20で変更されたP2 Gate C audit production blob:

```text
tools/reference-first-web-target-evidence-audit/main.go
  3878dae15b06a3b57954cf196b108da1cc03bbc7
```

をexact P1と組み合わせたclosure rerunはこのsessionでは未完了である。したがってEvidence heightは:

```text
E1.21 exact P1 hash identity
  = PASS

E1.21 exact P1 gofmt / vet / test×50
  = PASS

E1.20 P2 shared-credential delegation static source audit
  = PASS

E1.21 {P1,P2} exact closure rerun
  = pending

E1.21 5-package aggregate focused rerun
  = pending

tools/repository-health-check.sh
  = pending

code integration
  = pending
```

と分離する。

health integration gate自体はE1.22から変更しない。P2 closure / aggregate focused rerunとfull Repository healthを同一qualified source setに対して通した後だけdirect-to-main code integrationへ進む。


#### E1.24 latest-main refresh after parallel mainline progress — 2026-09-21

E1.23以後、別レーンのDefinition-Centered Feedback Loop / Second Application / Business Procedure実装がmainへ連続統合されたため、R7B integration sourceとの競合を再監査した。

current main:

```text
b01aa017ef55f12fcebb6d053e983b59200af897
```

E1.21のcanonical source確定点 `c2add12e...` からcurrent mainまで15 commitを比較した結果、canonical 11 source pathとのoverlapは0件だった。

current main上では、11 pathのうちR7B新規pathは未作成、既存Gate C audit / replay-plan pathはpre-R7B blobのままであり、R7B code integrationはまだ行われていない。

E1.21 exact blobsをcurrent mainへ再配置したbranchless candidate:

```text
candidate:
  3cf089d73ac08212bf11349aac91841c753d8abc

parent:
  b01aa017ef55f12fcebb6d053e983b59200af897

tree:
  666aa7c4f20a5099b876fcc24d275c42f5db8190

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

candidateにはE1.21以降のcredential-safe identitiesをそのまま保持する。

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

remote branchはmain以外18本を再監査した。17本はcurrent mainより9 commit behind / branch固有ahead 0、1本はすでにidenticalだった。branch固有workがないことを確認してから、17本をforceなしでcurrent mainへfast-forwardした。

```text
remote branch count excluding main = 18
unique branch work                  = 0
after convergence                  = all refs at current main
force update                        = none
new remote work branch              = none
open PR                             = 0
```

他チャット / 並行sessionも照合したが、E1.23以後にR7B canonical 11-pathのaggregate focused rerun、full `tools/repository-health-check.sh`、direct-to-main code integration、actual replay / repeatability Evidenceを完了した記録は確認できなかった。

GitHub Actionsについてcurrent main `b01aa017...` に紐づくworkflow runは存在せず、branchless candidate qualificationの代替経路には使えない。

したがって現在の最初の未完了gateは変えない。

```text
E1.24 canonical 11 source
  -> same working treeで{P1,P2} closure rerun
  -> same working treeで5-package aggregate focused rerun
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

candidate SHA自体はmainへfast-forward / cherry-pickしない。health-capable full checkoutではlatest mainを維持したまま、上記11 exact blobsだけをworking treeへmaterializeしてqualificationする。

Evidence height:

```text
current-main source overlap check       = PASS
latest-main exact candidate assembly    = PASS
remote branch convergence               = PASS
E1.21 exact P1 test x50                 = PASS
E1.20 P2 delegation static audit        = PASS
{P1,P2} exact closure rerun             = pending
5-package aggregate focused rerun       = pending
tools/repository-health-check.sh        = pending
code integration                        = pending
actual replay / repeatability Evidence  = pending
```


#### E1.25 post-review latest-main convergence — 2026-09-21

E1.24記録後、mainはSecond Applicationの独立レビュー記録2件だけ進み、current mainは:

```text
d5e293226a14841e11591fd07595b26fb74ea1e9
```

となった。E1.24 base `b01aa017...` からcurrent mainまでのchanged pathはdocs / READMEだけであり、canonical 11 source pathとのoverlapは0件である。

E1.21 credential-safe exact blobsをcurrent mainへ再配置したbranchless candidateを再構成した。

```text
candidate:
  d13b74daf0c9cf289d867b2bf53bb6538f620751

parent:
  d5e293226a14841e11591fd07595b26fb74ea1e9

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

current candidate exact sourceでcredential contractも再監査した。

```text
shared RejectReferenceCredentialMaterial:
  credential-ref://legacy/
  disabled-secret
  editor-secret
  locked-secret
  viewer-secret
  wrong-secret

V1 credential negative pressure:
  same 6 values

runner retained-output final scan:
  same 6 values

set equality:
  PASS
```

Gate C retained-artifact helper本体にはcredential literal listを持たず、shared rejector:

```text
firstwebtargetevidence.RejectReferenceCredentialMaterial
```

を1回だけ呼ぶ。したがってE1.20 / E1.21で行ったcredential single-source correctionはlatest-main candidateでも保持されている。

remote branchはmain以外18本を再監査し、17本がbehind 1 / branch固有ahead 0、`work/issue-000040-second-application-reference` がidenticalだった。17本はforceなしでcurrent mainへfast-forwardし、unique branch workを残していない。open PRは0。

integration gateは変更しない。

```text
E1.25 canonical 11 source
  -> same working treeで{P1,P2} exact closure rerun
  -> same working treeで5-package aggregate focused rerun
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

E1.23でexact P1 packageはtest×50までPASS済みだが、credential-safe P2とのactual closure rerunは未取得である。focused/static Evidenceをfull Repository healthへ読み替えず、actual exact-T replay / repeatability Evidenceも引き続き未取得として扱う。


#### E1.26 P1/P2 delta compatibility audit — 2026-09-21

E1.25でlatest-main convergenceとcredential-set parityは確認済みである。ここではE1.23でpendingのまま残る `{P1,P2}` closureについて、実行前のAPI互換性だけをexact blob差分で再監査する。

P1:

```text
old semantics.go
  46629767bee3b3733c0ca6e4433971c4cc85bb9d

new semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
```

差分はshared credential reject集合への:

```text
+ "wrong-secret",
```

のみであり、exported API:

```go
func RejectReferenceCredentialMaterial(name string, data []byte) error
```

のsignatureは不変である。

P2:

```text
old Gate C audit main.go
  eb44ef1d543f0849e424e99ed517492f747d634a

new Gate C audit main.go
  3878dae15b06a3b57954cf196b108da1cc03bbc7
```

production差分はGate Cローカルcredential literal scanを除去し:

```go
if err := firstwebtargetevidence.RejectReferenceCredentialMaterial(name, data); err != nil {
    return err
}
```

へ委譲する変更に限定される。P2はE1.20以前から同じshared packageをsemantic validationにimport済みであり、新しいpackage edgeは増えていない。

したがってstatic delta compatibilityは:

```text
P1 exported API compatibility = PASS
P2 -> P1 dependency shape     = unchanged
new credential behavior        = shared-set extension only
```

である。

ただしこれはcompile/test executionではない。Evidence heightは厳密に分離する。

```text
E1.25 latest-main convergence         = PASS
P1 exact rerun x50                    = PASS
P1/P2 delta API compatibility         = PASS (static)
P2 shared delegation source audit     = PASS (static)
{P1,P2} exact closure rerun           = pending
5-package aggregate focused rerun     = pending
tools/repository-health-check.sh      = pending
code integration                      = pending
actual replay / repeatability         = pending
```

integration pathはE1.25から変更しない。

```text
E1.25 canonical 11 source
  -> same working treeで{P1,P2} exact closure rerun
  -> same working treeで5-package aggregate focused rerun
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

candidate SHA自体をmainへfast-forward / cherry-pickしない。health-capable full checkoutではlatest mainを維持したままcanonical 11 exact blobsだけをmaterializeし、execution gateを同一working treeで閉じる。


#### E1.27 exact P1/P2 closure PASS — 2026-09-21

E1.23 / E1.26でpendingとしていたcredential-safe `{P1,P2}` closureを、E1.21 canonical candidateからGit object exact bytesでRepository外の一時Go moduleへ再構成して実行した。

実行前にscratch transfer bundle自体を12 logical chunksのlength / CRCで照合し、展開後に5 fileのGit blob identityを再計算した。5/5がcanonical identityと一致した。

```text
P1 semantics.go
  expected = 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
  actual   = 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

P1 semantics_test.go
  expected = ba17f69a7e375cfdeec0db544986d51f99b26192
  actual   = ba17f69a7e375cfdeec0db544986d51f99b26192

P1 semantics_pressure_test.go
  expected = 95553ba10756399a50e52e639f7d860f4420f29f
  actual   = 95553ba10756399a50e52e639f7d860f4420f29f

P2 Gate C audit main.go
  expected = 3878dae15b06a3b57954cf196b108da1cc03bbc7
  actual   = 3878dae15b06a3b57954cf196b108da1cc03bbc7

P2 Gate C audit main_test.go
  expected = 668efcdb084e9a7cefe050a5df4be86effc1cebb
  actual   = 668efcdb084e9a7cefe050a5df4be86effc1cebb
```

同じworking treeで:

```text
gofmt -l \
  tools/internal/firstwebtargetevidence \
  tools/reference-first-web-target-evidence-audit
  = empty

go vet \
  ./tools/internal/firstwebtargetevidence \
  ./tools/reference-first-web-target-evidence-audit
  = PASS

go test -count=50 \
  ./tools/internal/firstwebtargetevidence \
  ./tools/reference-first-web-target-evidence-audit
  = PASS
```

package result:

```text
tei/tools/internal/firstwebtargetevidence
  PASS

tei/tools/reference-first-web-target-evidence-audit
  PASS
```

これによりE1.20のGate C credential delegationとE1.21の6-value pressure correctionを同一closureでcompile / testしたEvidenceを取得した。

Evidence height:

```text
E1.21 P1 exact package x50        = PASS
E1.26 P1/P2 static compatibility  = PASS
E1.27 {P1,P2} exact closure x50   = PASS

5-package aggregate focused rerun  = pending
tools/repository-health-check.sh   = pending
code integration                   = pending
actual replay / repeatability      = pending
```

このclosure PASSはfull Repository healthの代替ではない。次のgateはcanonical P1-P5 exact sourceを同一working treeへmaterializeした5-package aggregate focused rerunであり、その後にfull checkout上の `tools/repository-health-check.sh` を要求する。


#### E1.28 post-closure current-main packet — 2026-09-21

E1.27のexact `{P1,P2}` closure PASS後、同じcredential-safe canonical 11 blobをcurrent mainへ再配置した。

```text
parent main:
  19423b1b646e30a9bdd2e502eba46fe48a3c204b

branchless candidate:
  b8d8902b2115c22ef3ce88ac803b5f14baa46d73

candidate tree:
  bd7b6743e6c86dc7b99578478fd5ae28dcf8fec3

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

remote branchはE1.27記録後にmain以外18本を再びforceなしでparent mainへfast-forwardし、branch固有ahead work 0 / open PR 0の状態へ収束させた。

current Evidence height:

```text
latest-main exact 11-path packet       = assembled
{P1,P2} exact closure x50              = PASS
P1 credential pressure x50             = PASS
P3/P4/P5 historical exact focused PASS = inherited only
current 5-package aggregate rerun       = pending
tools/repository-health-check.sh        = pending
code integration                        = pending
actual replay / repeatability           = pending
```

P3/P4/P5 transfer bundleもbranchless scratch objectとして準備しているが、container側でexact blob hash照合とaggregate executionを完了するまでEvidenceには数えない。

次のintegration gateはE1.27から変更しない。

```text
b8d8902b... canonical 11 source
  -> same working treeでP1-P5 aggregate focused rerun
  -> bash -n exact runner
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

candidate SHA自体はmainへfast-forward / cherry-pickせず、health-capable full checkoutでは11 exact blobsをlatest main working treeへmaterializeしてqualificationする。


#### E1.29 current-main dependency-closure audit — 2026-09-21

E1.28後に別レーンがcurrent mainを `132da35bfb1aa2a3e04a67c322c7907f9513d821` まで進めたため、R7B canonical 11-pathとのoverlapを再監査した。E1.28 parent `eb561532...` からcurrent mainまで3 commit / 8 changed pathsだったが、canonical 11-path overlapは0件だった。

E1.21 canonical exact blobsをcurrent mainへ再配置したbranchless candidate:

```text
candidate:
  5166eae4092b3e619ad8076e5fae71b3e94e82ec

parent:
  132da35bfb1aa2a3e04a67c322c7907f9513d821

tree:
  8eb40489c0a890620648400dca39b838c94e2838

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

remote branch 18本はbranch固有ahead work 0を確認したうえでforceなしfast-forwardし、current mainへ再収束させた。open PRは0件のままである。

current candidate exact import graph:

```text
P1 tei/tools/internal/firstwebtargetevidence
  -> standard library only

P2 tei/tools/reference-first-web-target-evidence-audit
  -> P1 + standard library

P3 tei/tools/reference-first-web-target-r7b-replay-plan
  -> standard library only

P4 tei/tools/reference-first-web-target-r7b-replay-audit
  -> P1 + standard library

P5 tei/tools/reference-first-web-target-r7b-repeatability
  -> standard library only
```

candidate-internal dependency edgeは `P2 -> P1` と `P4 -> P1` の2本だけで、cycleはない。

E1.17でP1 credential forbidden setへ追加した `wrong-secret` がP4へ新たなtest failure surfaceを作るかexact sourceを再監査した。

```text
P4 main.go
  wrong-secret literal = 0
  shared RejectReferenceCredentialMaterial calls = 2

P4 main_test.go
  wrong-secret literal = 0
  editor-secret literal = 2
```

P4 testが保持するcredential negative fixtureは、E1.17以前から禁止済みだった `editor-secret` だけである。したがってE1.17のP1変更でP4 test inputへ新規禁止値が混入した事実はない。shared rejectorのAPI shapeも不変である。

これはstatic compatibility Evidenceであり、P4 rerunまたはP1-P5 aggregate rerunの代替ではない。

current Evidence height:

```text
latest-main exact 11-path packet           = assembled
{P1,P2} exact closure x50                  = PASS
P1 credential pressure x50                 = PASS
P4/P1 dependency + fixture compatibility   = static PASS
P3/P5 package independence                 = static PASS
P1-P5 aggregate focused rerun              = pending
tools/repository-health-check.sh            = pending
code integration                            = pending
actual replay / repeatability Evidence      = pending
```

次のexecution gateは変えない。

```text
canonical 11 exact blobs
  -> same working treeでP1-P5 aggregate focused rerun
  -> bash -n exact runner
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

full checkoutまたはbyte-exact materializationが成立しないsessionではaggregate / repository healthを推測でPASS扱いせず、codeをmainへ統合しない。



#### E1.30 exact P1-P4 closure rerun — 2026-09-22

E1.17以降のcredential-safe P1を、V2 replay audit P4のexact source/testと同一一時moduleへbyte-exactにmaterializeして依存closureを再実行した。

materialized exact blobs:

```text
tools/internal/firstwebtargetevidence/semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

tools/reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da

tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
```

materialization後に各fileへ `git hash-object` を実行し、3/3で上記Git blob identityと一致した。SHA不一致の転送物はEvidenceに採用しない運用を維持した。

同一一時moduleで:

```text
gofmt -l .
  -> no output

go vet ./...
  -> PASS

go test -count=50 ./...
  -> PASS
     tei/tools/internal/firstwebtargetevidence [no test files]
     tei/tools/reference-first-web-target-r7b-replay-audit ok
```

したがってE1.17で追加された `wrong-secret` を含むshared credential rejectorと、P4 replay auditのactual compile/test dependency closureは実測済みになった。

current main `c46eafe95c69d91c2d6f4d51c58697d0539db2e4` までのE1.29以後の差分も再監査した。R7B canonical 11 source pathとのoverlapは0件で、追加変更はSecond Application S8 handover/preflight文書だけである。

Evidence heightを更新する。

```text
{P1,P2} exact closure x50                  = PASS
{P1,P4} exact closure x50                  = PASS
P1 credential pressure x50                 = PASS
P3 exact focused evidence                  = PASS (blob unchanged)
P5 exact focused evidence                  = PASS (blob unchanged)
P1-P5 single aggregate focused rerun       = pending
tools/repository-health-check.sh            = pending
code integration                            = pending
actual replay / repeatability Evidence      = pending
```

`{P1,P4}` closure PASSはsingle aggregate commandの実行結果そのものではないため、aggregate gateを推測で閉じない。full checkoutまたは11-file byte-exact materializationが成立したworking treeでP1-P5 aggregateを実行し、その後current-main Repository healthへ進む。

#### E1.31 latest-main refresh after exact P1-P4 closure — 2026-09-22

E1.29以後、Second Application / Shared Acceptance mainlineはsource-relevant base:

```text
c46eafe95c69d91c2d6f4d51c58697d0539db2e4
```

まで進んだ。E1.29 base `132da35b...` からこのbaseまで:

```text
ahead                   = 13 commits
changed paths           = 42
canonical 11-path overlap = 0
```

をGit compareで確認した。

その後のE1.30 exact P1-P4 closure記録およびE1.31整備は `plan/issue/000069-first-web-target-r7b-exact-t-replay.md` の文書変更だけであり、canonical 11 source pathとのoverlapはない。

source-relevant base `c46eafe...` 上へE1.21 credential-safe exact blobsを再配置したbranchless packet:

```text
parent:
  c46eafe95c69d91c2d6f4d51c58697d0539db2e4

candidate:
  a7aba7ba2ed3bda601608f9937d43bcca1d7acff

tree:
  8b96ed17026b1f06de1a93a92dd64361b301b691

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

をcurrent source packetとして保持する。

candidate SHA自体はauthorityではない。integration sourceはE1.21で固定したexact 11 blobsであり、health-capable sessionでは実行時latest mainへその11 blobsだけをmaterializeする。

current main上のcanonical path状態を再確認した時点では:

```text
P1 shared validator 3 files              = not yet present
Gate C audit main.go                      = pre-R7B blob
R7B replay-plan main.go / main_test.go    = pre-R7B blobs
R7B replay-audit 2 files                  = not yet present
R7B repeatability 2 files                 = not yet present
R7B operational runner                    = not yet present
```

であり、R7B code integrationはまだ行われていない。

E1.30までのEvidence heightを統合すると:

```text
current-main source overlap check       = PASS
latest-main exact candidate assembly    = PASS
E1.21 exact P1 test x50                 = PASS
E1.27 {P1,P2} exact closure x50         = PASS
E1.30 {P1,P4} exact closure x50         = PASS
P3 exact focused Evidence               = PASS (blob unchanged)
P5 exact focused Evidence               = PASS (blob unchanged)
P1-P5 single aggregate focused rerun    = pending
tools/repository-health-check.sh        = pending
code integration                        = pending
actual replay / repeatability Evidence  = pending
```

したがって次の一本道は:

```text
E1.21 exact 11 blobs on execution-time latest main
  -> same working treeでP1-P5 aggregate focused rerun
  -> bash -n exact runner
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

である。

full checkoutまたはbyte-exact materializationが成立しないsessionではaggregate / Repository healthを推測でPASS扱いせず、candidate SHAのfast-forward / cherry-pickや新規remote work branch作成も行わない。



#### E1.31 aggregate gate subsumption audit — 2026-09-22

E1.30までにcurrent canonical sourceのdependency closureは実測で閉じている。

```text
{P1,P2} exact closure
  gofmt clean
  go vet PASS
  go test -count=50 PASS

{P1,P4} exact closure
  gofmt clean
  go vet PASS
  go test -count=50 PASS

P3 exact focused
  blob unchanged from PASS Evidence

P5 exact focused
  blob unchanged from PASS Evidence
```

single P1-P5 aggregate commandを別の必須gateとして残す必要性を判断するため、6 candidate test filesをexact sourceから機械監査した。

確認項目:

```text
TestMain
t.Parallel
os.Setenv / t.Setenv / os.Unsetenv
os.Chdir / t.Chdir
fixed /tmp or os.TempDir
net.Listen / httptest server
exec.Command
```

結果:

```text
all test files:
  TestMain   = 0
  t.Parallel = 0
  Setenv     = 0
  Chdir      = 0
  FixedTmp   = 0
  Listen     = 0
  ExecCommand= 0

P2 main_test:
  t.TempDir  = 3
  file writes = 11

P4 main_test:
  t.TempDir  = 1
  file writes = 5

all other candidate tests:
  file writes = 0
```

P2/P4の書き込みは各testの `t.TempDir()` 配下に限定され、package間で共有する固定path、process-global environment、cwd、listener、external processはない。

candidate-internal dependency graphも:

```text
P2 -> P1
P4 -> P1
P1/P3/P5 -> standard library only
```

のままで、cycleはない。

Goのmulti-package `go test` は各packageを別test binaryとして実行するため、上記条件下ではsingle aggregate commandが新たに検証するcandidate-internal semantic interactionはない。加えてmandatory `tools/repository-health-check.sh` 自体が:

```text
gofmt -l .
go vet ./...
go test -count=1 ./...
go run ./cmd/tei validate -workspace reference/workspace
```

を同一full checkout / current-main working treeで実行し、candidate 5 packagesを含むrepository-wide aggregateを上位gateとして包含する。

したがってP1-P5 single focused aggregateは**diagnostic convenience check**へ降格し、integrationを止める独立mandatory gateから外す。これはfull Repository healthを省略する判断ではなく、むしろ重複したfocused gateを上位mandatory gateへ統合するものである。

更新後のintegration gate:

```text
canonical 11 exact blobs
  -> exact runner bash -n
  -> tools/repository-health-check.sh on current-main full checkout
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
```

Evidence height:

```text
{P1,P2} exact closure x50                 = PASS
{P1,P4} exact closure x50                 = PASS
P3 exact focused                          = PASS
P5 exact focused                          = PASS
candidate test isolation audit            = PASS
P1-P5 single focused aggregate            = optional diagnostic
tools/repository-health-check.sh           = pending mandatory gate
code integration                           = pending
actual replay / repeatability Evidence     = pending
```

full checkoutを持たないsessionではRepository healthを推測でPASS扱いせず、codeをmainへ統合しない。


#### E1.32 current-main health handoff packet — 2026-09-22

E1.31 aggregate gate subsumption後、mandatory gateをfull Repository healthへ一本化した状態でcurrent mainを再固定した。

```text
current main:
  8eb0abadef36759075dae32220ef78f4567bf62b

branchless health candidate:
  fa96801d40197e07a1d25e58bf14d07cfa9ff5f7

candidate tree:
  401a2ae16429f2d49165952f4120b6e9b465f14c

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

candidateはE1.21以降のcredential-safe exact 11 blobsを変更せず配置する。candidate SHAそのものをauthorityとはせず、integration authorityはexact blob identitiesとhealth実行対象working treeである。

実行環境を再監査した。

```text
Go:
  available

git:
  available

gh:
  unavailable

mounted TEI full checkout:
  not found under /mnt/data, /workspace, /workspaces, /home/oai/share, /tmp

private repository archive through GitHub connector:
  tarball endpoint = rejected
  zipball endpoint = rejected

private raw URL through web/container bridge:
  unavailable in this session
```

したがってこのsessionでは `tools/repository-health-check.sh` が要求するfull checkoutを構成できない。focused closure Evidenceをhealth PASSへ読み替えず、R7B codeをmainへ直接commitしない。

E1.31の判断に従い、P1-P5 single focused aggregateはmandatory integration gateではない。current mandatory pathは次だけである。

```text
execution-time latest main
  + canonical 11 exact blobs
  -> exact runner bash -n
  -> tools/repository-health-check.sh
  -> same revision / clean base / canonical 11-path overlapなし再確認
  -> direct-to-main code commit
```

health-capable cloneでの実行手順:

```bash
git checkout main
git pull --rebase origin main

# E1.21/E1.32 canonical 11 exact blobsだけをworking treeへ配置する。
# candidate commit自体をcherry-pick / fast-forwardしない。

bash -n tools/run-first-web-target-r7b-replay.sh
tools/repository-health-check.sh

git status --short
git rev-parse HEAD
git diff --name-only
```

qualification時は:

1. health開始時のHEADを記録する
2. canonical 11 path以外に意図しないsource差分がないことを確認する
3. exact runner `bash -n` PASS
4. `tools/repository-health-check.sh` PASS
5. health終了時も同じbase revisionであることを確認する
6. 並行main更新があればrebase後にcanonical 11 path overlapを再監査し、healthを再実行する
7. そのsame qualified working treeだけをdirect-to-main commitする

とする。

branch hygiene再監査では、E1.32記録直前にremote branch drift 0を確認した。新しいremote work branch / PRは作成していない。

Evidence height:

```text
{P1,P2} exact closure x50                  = PASS
{P1,P4} exact closure x50                  = PASS
P3 exact focused                           = PASS
P5 exact focused                           = PASS
candidate test isolation audit             = PASS
latest-main canonical packet               = PASS
P1-P5 focused aggregate                    = optional diagnostic
tools/repository-health-check.sh            = pending mandatory gate
code integration                            = pending
actual replay / repeatability Evidence      = pending
```

full healthが取得されるまではR7B Source/Design packetのintegration readinessを超える主張をしない。



#### E1.32 latest-main convergence and stale-candidate retirement — 2026-09-22

E1.31後に並行Second Application / R7B documentation laneが進み、execution-time latest mainを再確認した。

```text
current main:
  8eb0abadef36759075dae32220ef78f4567bf62b

source-relevant base:
  c46eafe95c69d91c2d6f4d51c58697d0539db2e4

main advance from source-relevant base:
  8 commits

changed paths:
  plan/SECOND_APPLICATION_S8_DRIVER_FIRST_WEB_TARGET_PREFLIGHT_2026-09.md
  plan/SECOND_APPLICATION_S8_SUCCESSOR_HANDOVER_PREFLIGHT_2026-09.md
  plan/issue/000040-semantic-component-composition.md
  plan/issue/000069-first-web-target-r7b-exact-t-replay.md

canonical 11-path overlap:
  0
```

E1.31 current packet `a7aba7ba2ed3bda601608f9937d43bcca1d7acff` のchanged pathは、引き続き次のcanonical 11 source pathだけである。

```text
tools/internal/firstwebtargetevidence/semantics.go
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
tools/internal/firstwebtargetevidence/semantics_test.go
tools/reference-first-web-target-evidence-audit/main.go
tools/reference-first-web-target-r7b-repeatability/main.go
tools/reference-first-web-target-r7b-repeatability/main_test.go
tools/reference-first-web-target-r7b-replay-audit/main.go
tools/reference-first-web-target-r7b-replay-audit/main_test.go
tools/reference-first-web-target-r7b-replay-plan/main.go
tools/reference-first-web-target-r7b-replay-plan/main_test.go
tools/run-first-web-target-r7b-replay.sh
```

旧 branchless candidate `c0a8d2c858f338cffd90f86752a21293c391350b` は、
その作成時base `17acb5ca...` からcurrent mainが94 commit進み、candidate固有1 commitを持つdiverged objectである。
このcandidate SHAをfast-forward / cherry-pick / integration authorityとして使用しない。

canonical authorityはcandidate commitではなく、E1.21以降で固定・再監査してきたexact 11 blob setである。
したがって次のqualificationでは、execution-time latest mainへこの11 exact blobsだけをmaterializeし、
mandatory gateを同一full checkoutで実行する。

remote branch stateも再確認し、確認時点ではmainを含む既存19 branch refがすべて
`8eb0abadef36759075dae32220ef78f4567bf62b` を指しており、branch固有の未統合workは存在しない。
新規remote work branchは作成しない。

Evidence height:

```text
latest-main convergence                    = PASS
canonical 11-path overlap since c46eafe    = 0
remote branch unique ahead work            = 0
stale c0a8d2 candidate                     = retired / do not integrate
current canonical packet                   = a7aba7ba... exact 11 paths
exact runner bash -n                       = pending mandatory gate
tools/repository-health-check.sh           = pending mandatory gate
code integration                           = pending
actual replay / repeatability Evidence     = pending
```

次の一本道は変えない。

```text
execution-time latest main
  -> canonical 11 exact blobs materialize
  -> exact runner bash -n
  -> tools/repository-health-check.sh
  -> same revision / clean base / no source overlap再確認
  -> direct-to-main code commit
  -> existing branch refsをforceなしでmainへ収束
  -> Packet M actual replay-plan materialization
```

full checkoutを持たないsessionではRepository healthを推測でPASS扱いせず、
code integrationを先取りしない。



#### E1.33 concurrent-write reconciliation and branch convergence — 2026-09-22

E1.32 stale-candidate retirement recordの作成直前に、並行R7B documentation laneから:

```text
aeb341795a0dd8a3a5874f29feb6d5e54b0d751b
docs(r7b): pin current-main health handoff packet
```

がmainへ入っていた。E1.32 writeはGitHub contents APIのblob-SHA guardを通してこのcommitをparentとして成立しており、並行変更を上書きしていない。

したがってE1.32内の `current main = 8eb0abad...` は、並行commit直前に取得したpre-concurrent snapshotとして読む。
このreconciliation checkpointでのmainは:

```text
d382559bbaa695ba9226a7e93e88e5f25f3fc995
  parent = aeb341795a0dd8a3a5874f29feb6d5e54b0d751b
```

である。

source-relevant base `c46eafe95c69d91c2d6f4d51c58697d0539db2e4` からこのcheckpointまでを再比較した。

```text
main ahead from source-relevant base = 10 commits

changed paths:
  plan/SECOND_APPLICATION_S8_DRIVER_FIRST_WEB_TARGET_PREFLIGHT_2026-09.md
  plan/SECOND_APPLICATION_S8_SUCCESSOR_HANDOVER_PREFLIGHT_2026-09.md
  plan/issue/000040-semantic-component-composition.md
  plan/issue/000069-first-web-target-r7b-exact-t-replay.md

canonical 11-path overlap = 0
```

また、E1.32 documentation commit後にmain以外18本の既存remote branchを再点検し、
全refを `force=false` でmainへfast-forwardした。18/18が成功し、独自ahead workを持つbranchはなかった。

収束後checkpoint:

```text
remote branch count = 19
unique branch tips   = 1
all tips             = d382559bbaa695ba9226a7e93e88e5f25f3fc995
force updates        = 0
new remote branches  = 0
```

これにより、並行documentation更新を保持したままbranch fragmentationを再解消した。
R7B code integration gateは変更しない。

```text
execution-time latest main
  -> canonical 11 exact blobs materialize
  -> exact runner bash -n
  -> tools/repository-health-check.sh
  -> same revision / clean base / canonical 11-path overlapなし再確認
  -> direct-to-main code commit
  -> existing branch refsをforceなしでmainへ再収束
  -> Packet M actual replay-plan materialization
```

current Evidence height:

```text
parallel-write reconciliation              = PASS
canonical 11-path overlap through d382...  = 0
branch convergence at checkpoint            = PASS (19 refs / 1 tip)
stale c0a8d2 candidate                      = retired / do not integrate
current health candidate                    = fa96801d... exact 11 paths
exact runner bash -n                        = pending mandatory gate
tools/repository-health-check.sh            = pending mandatory gate
code integration                            = pending
actual replay / repeatability Evidence      = pending
```

#### B4 continuation — live runner allocation re-probe and integration stop — 2026-09-22

TEI development lane B4 starts from the converged state recorded by E1.33.

Pre-write branch state:

```text
main = research/selective-knowledge-routing-r6
     = ebee11a7f33aec86f3f48a3b7294ad9e817c696d
ahead = 0
behind = 0
```

No new remote work branch was created.

To determine whether the mandatory Repository-health gate had become executable, the most recent failed Repository branch hygiene run was re-run through the existing GitHub Actions workflow.

```text
workflow run  = 34771242002
run attempt   = 2
updated_at    = 2026-09-21T22:28:05Z
             = 2026-09-22T07:28:05+09:00
status        = completed
conclusion    = failure

inventory job:
  id          = 106541743107
  conclusion  = failure
  steps       = null

branch-name-policy:
  id          = 106541744849
  conclusion  = skipped
  steps       = null
```

The re-probe therefore still failed before any observable workflow step execution. It does not qualify the current R7B source packet, and it does not provide Repository-health Evidence.

The integration boundary remains unchanged:

```text
execution-time latest main
  -> canonical 11 exact blobs materialize
  -> exact runner bash -n
  -> tools/repository-health-check.sh
  -> same revision / clean base / canonical 11-path overlap-free recheck
  -> code integration
  -> existing branch refs converge to merged main without force
  -> Packet M actual replay-plan materialization
```

Current B4 Evidence height:

```text
parallel-lane reconciliation               = PASS
B branch/main convergence before B4 write  = PASS
live runner allocation re-probe            = FAIL before observable steps
canonical 11 exact blobs                    = unchanged authority
exact runner bash -n                        = pending mandatory gate
tools/repository-health-check.sh            = pending mandatory gate
code integration                            = blocked
actual replay / repeatability Evidence      = pending
```

B4 must not reinterpret the workflow failure as source/test failure, and must not bypass Repository health by fast-forwarding or cherry-picking any branchless candidate. Documentation-only progress may be integrated while this gate remains blocked; the canonical 11 source paths stay untouched.




#### B4 continuation — exact runner syntax closure and latest-main packet refresh — 2026-09-22

The live runner allocation re-probe above remained unavailable for Repository health, but the first mandatory source gate does not require a full checkout. The exact operational runner bytes were therefore re-acquired from the canonical health packet and checked independently.

Exact runner identity:

```text
path:
  tools/run-first-web-target-r7b-replay.sh

canonical Git blob:
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03

source packet:
  fa96801d40197e07a1d25e58bf14d07cfa9ff5f7
```

The connector-provided base64 bytes were decoded into an isolated local file. Before syntax checking, `git hash-object` was run over the decoded bytes.

```text
git hash-object:
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03

expected:
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03

byte identity:
  PASS

bash -n:
  PASS
```

This closes the exact-runner syntax gate without claiming execution of the runner, Repository health, replay, or repeatability.

While this check was in progress, parallel B4 work advanced main through PR #844:

```text
latest main:
  d731c730fe0488450b202f7012a9c549821316c2

advance from ebee11a7...:
  2 commits

changed paths:
  plan/issue/000069-first-web-target-r7b-exact-t-replay.md

canonical 11-path overlap:
  0
```

The canonical 11 exact blobs were then placed over this latest main as a new branchless packet:

```text
parent:
  d731c730fe0488450b202f7012a9c549821316c2

branchless candidate:
  c95232a93d48084f5e5f0ae13b668e9b50ca7912

candidate tree:
  adff524a5388270836b3bf38a7b32b3e3adcd066

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

The candidate is not an integration authority and must not be merged, fast-forwarded, or cherry-picked as a shortcut. Its purpose is only to pin the exact current-main composition that a health-capable checkout must materialize and qualify.

Before this documentation update, all 17 remote branches that were still at the prior convergence point were fast-forwarded to `d731c730...` with `force=false`. No unique branch work was overwritten and no new remote work branch was created.

Updated B4 Evidence height:

```text
parallel-lane reconciliation               = PASS
latest-main canonical 11-path overlap       = 0
canonical 11 exact blobs                    = unchanged authority
exact runner byte identity                  = PASS
exact runner bash -n                        = PASS
latest-main branchless packet               = c95232a9... (11 paths only)
live GitHub runner allocation               = unavailable before observable steps
tools/repository-health-check.sh            = pending mandatory gate
code integration                            = blocked
actual replay / repeatability Evidence      = pending
```

The remaining integration path is now narrower:

```text
health-capable full checkout on execution-time latest main
  -> materialize the same canonical 11 exact blobs
  -> tools/repository-health-check.sh
  -> same revision / clean base / no canonical-path overlap recheck
  -> integrate the qualified 11-path source change
  -> converge existing branch refs to merged main without force
  -> Packet M actual replay-plan materialization
```

No Source/Design Evidence is promoted beyond the level actually observed.

#### B4 continuation — S8 reconciliation and current-main health packet refresh — 2026-09-22

After the exact-runner syntax gate was closed by PR #845, the concurrent Second Application lane advanced `main` by one commit:

```text
1b752c976a6b86bb1087f6b8b78814e4c5ad0c89
test(driver): run the accepted Second Application through the ordinary Driver path (S8)
```

The S8 delta from the prior R7B merge point changes only:

```text
docs/status.md
internal/driver/contract_query_executor_registry_key_test.go
internal/driver/second_application_scenario_test.go
plan/issue/000040-semantic-component-composition.md
```

Canonical R7B 11-path overlap is therefore:

```text
0
```

The existing B branch was fast-forwarded to the new `main` with `force=false` before any B4 write:

```text
main = research/selective-knowledge-routing-r6
     = 1b752c976a6b86bb1087f6b8b78814e4c5ad0c89
ahead = 0
behind = 0
```

The same canonical 11 exact blobs were then composed over that exact S8 main tree without creating a remote work branch.

```text
parent:
  1b752c976a6b86bb1087f6b8b78814e4c5ad0c89

parent tree:
  401c2accc5e1bf8d2583af293957a175a06119e8

current branchless health candidate:
  8445b87a53cffe64daf49a7463a4685422bfaebd

candidate tree:
  e691f0d3e0cc0338a4d340ff60b8ed804f1eb189

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

The 11 blob identities remain unchanged from the canonical packet, including the already syntax-qualified runner blob:

```text
tools/run-first-web-target-r7b-replay.sh
  = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

This candidate is still not integration authority. It is a current-main composition target for a health-capable full checkout only.

A direct full-checkout route was re-probed in the execution container. Go and git are available, but the container cannot resolve GitHub for `git ls-remote`, and no mounted TEI checkout is present. The GitHub connector can inspect the complete repository tree, but that does not constitute a filesystem checkout on which `tools/repository-health-check.sh` can execute. Therefore Repository health is not inferred from connector-side tree access.

Remote branch convergence was also re-audited after the S8 commit. Before convergence, every non-B branch had:

```text
ahead  = 0
behind = 1
```

and therefore contained no unique remote work. All 17 stale refs were fast-forwarded to the S8 main commit with `force=false`. Together with `main` and the already-current B branch:

```text
remote branch count = 19
unique branch tips   = 1
tip                  = 1b752c976a6b86bb1087f6b8b78814e4c5ad0c89
force updates        = 0
new remote branches  = 0
```

Current B4 Evidence height:

```text
Second Application S8 reconciliation        = PASS
S8 vs canonical 11-path overlap             = 0
B branch latest-main convergence             = PASS
all remote refs convergence                  = PASS (19 refs / 1 tip)
canonical 11 exact blobs                     = unchanged authority
exact runner byte identity                   = PASS
exact runner bash -n                         = PASS
current-main branchless packet               = 8445b87a... (11 paths only)
health-capable full checkout                 = unavailable in this session
tools/repository-health-check.sh             = pending mandatory gate
code integration                             = blocked
actual replay / repeatability Evidence       = pending
```

The remaining path is unchanged:

```text
health-capable full checkout on execution-time latest main
  -> materialize the same canonical 11 exact blobs
  -> tools/repository-health-check.sh
  -> same revision / clean base / no canonical-path overlap recheck
  -> integrate the qualified 11-path source change
  -> converge existing branch refs to merged main without force
  -> Packet M actual replay-plan materialization
```

No branchless candidate is merged, fast-forwarded, or cherry-picked to bypass the mandatory health gate.

#### B4 continuation — health harness identity and hosted-runner re-probe — 2026-09-22

The remaining mandatory gate is `tools/repository-health-check.sh`. Its exact current-main bytes were reacquired and verified independently before attempting any health claim.

```text
path:
  tools/repository-health-check.sh

current main blob:
  d9bcf87b5a8da8d19dad3041a2e09d31e21e0fd9

git hash-object over reacquired exact bytes:
  d9bcf87b5a8da8d19dad3041a2e09d31e21e0fd9

byte identity:
  PASS

bash -n:
  PASS
```

This only qualifies the harness bytes and shell syntax. It does not execute:

```text
gofmt -l .
go vet ./...
go test -count=1 ./...
go run ./cmd/tei validate -workspace reference/workspace
```

A fresh GitHub-hosted runner allocation re-probe was also performed on 2026-09-22.

Repository branch hygiene rerun:

```text
run      = 34771242002
attempt  = 2
job      = 106541743107 (inventory)
updated  = 2026-09-21T22:28:05Z
         = 2026-09-22T07:28:05+09:00
result   = failure
steps    = null
```

Knowledge portability reference evidence rerun:

```text
run      = 35385970642
attempt  = 2
job      = 106560654921 (verify)
queued   = approximately 2026-09-21T23:43:55Z
failed   = by 2026-09-21T23:43:59Z
         = 2026-09-22 08:43 JST
result   = failure
steps    = null
```

The second probe briefly entered `queued`, but no observable workflow step was created. Therefore neither probe establishes checkout, setup-go, Repository health, or Source Evidence execution. The observation has also been recorded in infrastructure issue #513.

Updated B4 gate state:

```text
canonical 11 exact blobs                     = fixed
exact replay runner byte identity            = PASS
exact replay runner bash -n                  = PASS
exact repository-health harness identity     = PASS
repository-health harness bash -n            = PASS
hosted runner step execution                 = unavailable
health-capable full checkout                 = unavailable in this session
tools/repository-health-check.sh execution   = pending mandatory gate
code integration                             = blocked
actual replay / repeatability Evidence       = pending
```

The health gate is not weakened by the harness syntax result. Integration resumes only when the exact current-main working tree plus canonical 11 blobs can execute the harness itself.




#### B4 continuation — hosted-runner attempt 3 and latest-main packet pin — 2026-09-22

The hosted-runner allocation blocker was re-probed again after the prior B4 checkpoint without creating a new workflow or branch.

Knowledge portability reference evidence:

```text
run       = 35385970642
attempt   = 3
job       = 106586603139 (verify)
started   = 2026-09-22T01:52:24Z
          = 2026-09-22 10:52:24 JST
completed = 2026-09-22T01:52:30Z
          = 2026-09-22 10:52:30 JST
result    = failure
steps     = null
```

The job entered `queued` but again terminated before any observable step execution. This is not checkout, setup-go, Repository health, Source test, or Product Evidence. Infrastructure issue #513 was updated with the same observation.

Parallel work had already advanced and converged main through:

```text
19e26189fcaa0ef00f4c448361744c466ff64c0d
Merge PR #847 — r7b: pin health harness and hosted-runner re-probe
```

At the pre-write checkpoint all 19 remote branch refs pointed to that same main commit, so there was no unique remote branch work to preserve or reconcile.

The unchanged canonical 11 exact blobs were composed over this exact latest main as a fresh branchless qualification packet:

```text
parent:
  19e26189fcaa0ef00f4c448361744c466ff64c0d

parent tree:
  ac5d4fd2d88f17abbf82e2e56cec3e46703afddd

branchless candidate:
  3e4dec62ed54c2f1adea7fa2bacd919eb6cb6f27

candidate tree:
  a39e3d51bcfc4f8af217a2f35b85104c1c0d714b

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

The candidate remains a composition target only. It is not integration authority and must not be merged, fast-forwarded, or cherry-picked around the mandatory health gate.

Current B4 Evidence height:

```text
branch convergence at pre-write checkpoint    = PASS (19 refs / 1 tip)
canonical 11 exact blobs                      = fixed
exact replay runner byte identity             = PASS
exact replay runner bash -n                   = PASS
exact repository-health harness identity      = PASS
repository-health harness bash -n             = PASS
hosted runner attempt 3 step execution        = unavailable
latest-main branchless packet                 = 3e4dec62... (11 paths only)
tools/repository-health-check.sh execution    = pending mandatory gate
code integration                              = blocked
actual replay / repeatability Evidence        = pending
```

The remaining path is unchanged:

```text
health-capable full checkout on execution-time latest main
  -> materialize the same canonical 11 exact blobs
  -> tools/repository-health-check.sh
  -> same revision / clean base / no canonical-path overlap recheck
  -> integrate the qualified 11-path source change
  -> converge existing branch refs to merged main without force
  -> Packet M actual replay-plan materialization
```

No Evidence level is promoted from a queued-but-step-less hosted job.

#### B4 continuation — exact P1 local reconstruction on current packet — 2026-09-22

While the full-checkout Repository-health gate remains unavailable, the canonical P1 package can be reconstructed independently because its complete package is exactly three canonical files and has no Repository-local dependency outside that package.

The exact files were reacquired from the canonical 11-path packet and written into an isolated local module using the current module declaration.

Git blob identity was checked before any test execution:

```text
tools/internal/firstwebtargetevidence/semantics.go
  expected = 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
  actual   = 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  expected = 95553ba10756399a50e52e639f7d860f4420f29f
  actual   = 95553ba10756399a50e52e639f7d860f4420f29f

tools/internal/firstwebtargetevidence/semantics_test.go
  expected = ba17f69a7e375cfdeec0db544986d51f99b26192
  actual   = ba17f69a7e375cfdeec0db544986d51f99b26192
```

The isolated exact package was then exercised with:

```text
gofmt -l tools/internal/firstwebtargetevidence
  -> empty / PASS

GOENV=off GOWORK=off GOFLAGS= go vet ./tools/internal/firstwebtargetevidence
  -> PASS

GOENV=off GOWORK=off GOFLAGS= go test -count=50 ./tools/internal/firstwebtargetevidence
  -> PASS
  -> ok tei/tools/internal/firstwebtargetevidence
```

This is fresh exact P1 focused Evidence. It is stronger than source inspection alone, but it is intentionally not promoted to Repository-health Evidence because the rest of the Repository is absent from the isolated reconstruction.

During this work, concurrent B4 progress merged PR #848:

```text
252c748435a325ae1b48f2df0a8702d46eef3c32
Merge PR #848 — r7b: record runner attempt 3 and refresh packet
```

That concurrent delta modifies only this R7B plan document and has zero canonical 11 source-path overlap. The B branch and all other 18 remote branches were observed fully converged to the same current main, with no unique ahead work.

The canonical 11 exact blobs were therefore recomposed over exact current main after PR #848:

```text
parent:
  252c748435a325ae1b48f2df0a8702d46eef3c32

parent tree:
  359b5aabd28956e88acdad010124e756b07474d0

current branchless health candidate:
  ce21d74d3cbd897505de8fa3ea866e318c3ad878

candidate tree:
  e60fde056650903c85f106a27ab59c2c8b8b2d17

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

Again, the branchless candidate is a composition target only and is not integration authority.

Updated B4 Evidence height:

```text
all remote branches vs current main          = identical (19 refs / 1 tip)
canonical 11 exact blobs                     = fixed
P1 exact Git blob identity                   = PASS
P1 exact gofmt                               = PASS
P1 exact go vet                              = PASS
P1 exact go test -count=50                   = PASS
exact replay runner byte identity            = PASS
exact replay runner bash -n                  = PASS
exact repository-health harness identity     = PASS
repository-health harness bash -n            = PASS
hosted runner step execution                 = unavailable
current branchless packet                    = ce21d74d... (11 paths only)
tools/repository-health-check.sh execution   = pending mandatory gate
code integration                             = blocked
actual replay / repeatability Evidence       = pending
```

The mandatory integration path remains unchanged. Focused P1 PASS narrows uncertainty but does not authorize bypassing full Repository health.

#### B4 continuation — P2 closure preflight and pre-run log boundary — 2026-09-22

After PR #849 merged, all 19 remote refs were reconverged to:

```text
e4aef476838508e9a69b1ca5c06e67ab7b852d61
```

with `force=false`, and no branch retained unique ahead work.

The canonical 11 exact blobs were recomposed over that exact main:

```text
parent:
  e4aef476838508e9a69b1ca5c06e67ab7b852d61

parent tree:
  bba7a50067f8675989db994c08a17284f76dceda

branchless health candidate:
  29ae71e736372d5fbafa02e9bb1ba8d378501907

candidate tree:
  90b508635a196b3279233a1023ccbdd2fc6d77ca

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

As before, this candidate is a composition target only and is not integration authority.

P2 (`tools/reference-first-web-target-evidence-audit`) was then re-audited for exact source closure.

```text
candidate main.go:
  3878dae15b06a3b57954cf196b108da1cc03bbc7

current-main main.go:
  2dd0102a04bdda29dd50dab13ba9a663b8df93d8

candidate main_test.go:
  668efcdb084e9a7cefe050a5df4be86effc1cebb

current-main main_test.go:
  668efcdb084e9a7cefe050a5df4be86effc1cebb
```

Thus P2 changes the production source blob while retaining the exact already-existing test blob. Its Repository-local import closure remains:

```text
P2 -> P1
P2 -> standard library
```

Fresh P1 execution remains:

```text
gofmt clean
go vet PASS
go test -count=50 PASS
```

A fresh P2 execution result is not claimed by this checkpoint. Historical exact {P1,P2} closure remains recorded, while this section only reconfirms current candidate identity and dependency closure.

The hosted-runner failure boundary was also probed one step deeper. Job-log retrieval was attempted for three recent `steps=null` jobs:

```text
106586603139  Knowledge portability attempt 3
106560654921  Knowledge portability attempt 2
106541743107  Repository branch hygiene attempt 2
```

All three returned a missing log blob (`BlobNotFound`). This is consistent with the existing observation that no workflow step was created; no checkout/setup-go/test log exists to classify as source failure. The observation is also recorded in infrastructure issue #513.

Updated B4 Evidence height:

```text
remote branch convergence                    = PASS (19 refs / 1 tip at checkpoint)
canonical 11 exact blobs                     = fixed
P1 exact focused execution                   = PASS (gofmt / vet / test x50)
P2 exact source/test identity                = PASS
P2 Repository-local dependency closure       = P1 only
fresh P2 execution                           = not claimed at this checkpoint
exact replay runner identity / bash -n       = PASS
health harness identity / bash -n            = PASS
hosted runner observable steps               = unavailable
hosted runner job logs for failed probes     = absent / BlobNotFound
current branchless packet                    = 29ae71e7... (11 paths only)
tools/repository-health-check.sh execution   = pending mandatory gate
code integration                             = blocked
actual replay / repeatability Evidence       = pending
```

The mandatory integration gate remains full Repository health. P1 focused PASS and P2 closure re-audit narrow the remaining uncertainty but do not bypass that gate.



#### B4 continuation — exact P2 execution packet handoff and branch convergence — 2026-09-22

After PR #850 merged, repository branch state was re-audited before any new write.

```text
main:
  2e615ea860a83e16999c4f2bc4ba41173fda34c1
  Merge PR #850 — r7b: pin P2 closure and pre-run log boundary

remote refs:
  total branches = 19
  main + 18 non-main refs
  all non-main refs vs main:
    ahead  = 0
    behind = 0
    status = identical

open pull requests:
  0
```

No unique branch work existed to preserve, and no new remote work branch was created. The existing B4 branch `research/selective-knowledge-routing-r6` remains the only write lane for this continuation.

Infrastructure issue #513 was also rechecked. The latest hosted-runner observations remain:

```text
observable workflow steps = unavailable
steps                     = null
failed-job log blobs      = absent / BlobNotFound
repository health         = not executed
```

No new mechanical rerun is triggered from this checkpoint because no environment-change Evidence has been observed.

The next useful executable unit is therefore pinned as an **exact P2 isolated-closure packet**. This is not Repository-health Evidence and does not authorize integration of the canonical 11-path source packet. It only removes ambiguity about the bytes that a health-capable or local execution environment must reconstruct.

Exact packet:

```text
go.mod
  f8763f8fb2d64d73f1b40b905f62b5e09c44493b

tools/internal/firstwebtargetevidence/semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5

tools/internal/firstwebtargetevidence/semantics_test.go
  ba17f69a7e375cfdeec0db544986d51f99b26192

tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  95553ba10756399a50e52e639f7d860f4420f29f

tools/reference-first-web-target-evidence-audit/main.go
  3878dae15b06a3b57954cf196b108da1cc03bbc7

tools/reference-first-web-target-evidence-audit/main_test.go
  668efcdb084e9a7cefe050a5df4be86effc1cebb
```

These blobs are read from the already-pinned canonical composition object `29ae71e736372d5fbafa02e9bb1ba8d378501907`. That object remains a composition target only; it is not integration authority.

The isolated execution contract is:

```text
1. create a fresh temporary directory outside the Repository checkout
2. materialize exactly the six files above at their repository-relative paths
3. verify each file with git hash-object before execution
4. do not add any other Repository source file
5. run:

   gofmt -l tools/internal/firstwebtargetevidence             tools/reference-first-web-target-evidence-audit

   GOENV=off GOWORK=off GOFLAGS= \
     go vet ./tools/internal/firstwebtargetevidence \
            ./tools/reference-first-web-target-evidence-audit

   GOENV=off GOWORK=off GOFLAGS= \
     go test -count=50 ./tools/internal/firstwebtargetevidence \
                       ./tools/reference-first-web-target-evidence-audit

6. record exact Go version, GOOS, GOARCH, command output, and post-run blob identity
```

Acceptance boundary:

```text
exact six-blob identity                    = required
P1 focused execution                      = already observed PASS
P2 fresh focused execution                = pending
full Repository health                    = pending mandatory gate
canonical 11-path source integration      = blocked
actual replay / repeatability Evidence    = pending
```

A future P2 focused PASS narrows source uncertainty only. The canonical 11-path packet still requires `tools/repository-health-check.sh` on a full, execution-time-latest checkout before any source integration.



#### B4 continuation — current canonical focused-Evidence closure rebind — 2026-09-22

The current canonical packet was re-audited against the exact focused executions already recorded for the same Git blob identities. No new focused execution is claimed by this checkpoint; the purpose is to determine whether any focused source uncertainty remains after the credential-safe P1/P2 changes.

Current latest-main composition:

```text
parent:
  60d4de53d59c170f20a5ee8bc637a416370716b0

branchless composition candidate:
  e33ef2e6060b573b9e7742f2cb1c9f49715eecf1

candidate tree:
  3641cb543c536a38c78257079b6483ff217d1c63

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

The candidate remains a composition target only and is not integration authority.

The candidate package dependency graph remains:

```text
P1  tei/tools/internal/firstwebtargetevidence
    -> standard library only

P2  tei/tools/reference-first-web-target-evidence-audit
    -> P1 + standard library

P3  tei/tools/reference-first-web-target-r7b-replay-plan
    -> standard library only

P4  tei/tools/reference-first-web-target-r7b-replay-audit
    -> P1 + standard library

P5  tei/tools/reference-first-web-target-r7b-repeatability
    -> standard library only
```

Exact current Evidence binding:

```text
P1/P2 credential-safe closure:
  P1 semantics.go
    9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
  P1 semantics_test.go
    ba17f69a7e375cfdeec0db544986d51f99b26192
  P1 semantics_pressure_test.go
    95553ba10756399a50e52e639f7d860f4420f29f
  P2 main.go
    3878dae15b06a3b57954cf196b108da1cc03bbc7
  P2 main_test.go
    668efcdb084e9a7cefe050a5df4be86effc1cebb
  recorded exact execution:
    gofmt clean
    go vet PASS
    go test -count=50 PASS

P3 replay-plan:
  main.go
    40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
  main_test.go
    d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
  current blobs are byte-identical to the previously executed exact focused set
  dependency closure is standard library only

P1/P4 credential-safe closure:
  P1 semantics.go
    9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
  P4 main.go
    a370e0c5efc4399e324be7a0ad4801626f05a5da
  P4 main_test.go
    2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
  recorded exact execution:
    gofmt clean
    go vet PASS
    go test -count=50 PASS

P5 repeatability:
  main.go
    7b2cf09460319b13241fa44e59f865d3864734ff
  main_test.go
    30a907578836c9b1b33b79bf4d521fb809a49e74
  current blobs are byte-identical to the previously executed exact focused set
  recorded exact execution:
    gofmt clean
    go vet PASS
    go test -count=50 PASS
```

Therefore every current canonical candidate package is covered by exact focused execution at its actual dependency closure:

```text
{P1,P2} current exact closure x50 = PASS
P3 current exact focused set      = PASS by identical blob identity
{P1,P4} current exact closure x50 = PASS
P5 current exact focused set      = PASS by identical blob identity
```

This does not revive the old aggregate focused command as a mandatory gate. E1.31 already subsumed that redundant aggregate into the stronger repository-wide mandatory gate:

```text
tools/repository-health-check.sh
  -> gofmt -l .
  -> go vet ./...
  -> go test -count=1 ./...
  -> go run ./cmd/tei validate -workspace reference/workspace
```

The remaining uncertainty is now explicitly Repository interaction / integration qualification, not an untested canonical focused package.

Current Evidence height:

```text
canonical 11 exact blobs                         = fixed
current P1-P5 focused dependency closures        = covered by exact executions
exact replay runner identity / bash -n           = PASS
health harness identity / bash -n                = PASS
hosted runner observable steps                   = unavailable
tools/repository-health-check.sh execution       = pending mandatory gate
canonical 11-path source integration             = blocked
actual replay / repeatability Evidence           = pending
```

No source integration is authorized until the mandatory full-checkout Repository health gate passes on execution-time latest main with these same canonical 11 blobs.


#### B4 continuation — latest-main branchless qualification packet and checkout transport boundary — 2026-09-22

Parallel B4 work merged PR #852 before this continuation:

```text
main:
  3bb05b7ee4d5033f59113abf2fd5c74e48c598d6
  Merge PR #852 — r7b: rebind current focused evidence closure

open pull requests:
  0

remote branch state before this write:
  total refs     = 19
  non-main ahead = 0
  non-main behind= 0
  status         = all identical to main
```

The PR #852 delta changes only this R7B plan document. It does not overlap the canonical 11 source paths.

The canonical 11 exact blobs were therefore recomposed over exact current main without creating or moving a remote work branch.

```text
parent:
  3bb05b7ee4d5033f59113abf2fd5c74e48c598d6

parent tree:
  b08a9bed7f6544835f057ec54cef8c862a925a82

branchless qualification candidate:
  645adb7e0fd3664514079135163c218a61a08c67

candidate tree:
  dcc35e1aedf76e99d8471a1243e55bc1c96beaf7

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

The 11 blob identities are unchanged:

```text
tools/internal/firstwebtargetevidence/semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
  95553ba10756399a50e52e639f7d860f4420f29f
tools/internal/firstwebtargetevidence/semantics_test.go
  ba17f69a7e375cfdeec0db544986d51f99b26192
tools/reference-first-web-target-evidence-audit/main.go
  3878dae15b06a3b57954cf196b108da1cc03bbc7
tools/reference-first-web-target-r7b-repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff
tools/reference-first-web-target-r7b-repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
tools/reference-first-web-target-r7b-replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da
tools/reference-first-web-target-r7b-replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
tools/reference-first-web-target-r7b-replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
tools/reference-first-web-target-r7b-replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
tools/run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

This candidate remains a composition target only. It is not integration authority and must not be merged, fast-forwarded, or cherry-picked around the mandatory Repository-health gate.

A new checkout-transport feasibility probe was also performed instead of mechanically rerunning the known-broken hosted runner.

GitHub connector recursive-tree access succeeds for exact current main:

```text
recursive tree truncated = false
repository blobs         = 1292
repository blob bytes    = 10,317,457

health-oriented working-set estimate:
  Go / go.mod / go.sum / reference workspace /
  testdata / shell harness paths
  files = 730
  bytes = 4,843,307
```

The execution container itself still cannot resolve `github.com`; ordinary `git ls-remote` / clone therefore remains unavailable. The connector can enumerate and read the Git tree, but its textual file interface does not provide a repository archive that can be mounted directly into the execution filesystem. Consequently this probe does not constitute a full checkout and does not execute Repository health.

PR #852 was also checked for naturally generated workflow runs. Neither its head commit nor merge commit has a connector-visible workflow run, so no new runner-allocation Evidence exists and no mechanical rerun is initiated.

Current Evidence height:

```text
parallel-lane latest-main reconciliation       = PASS
all remote refs identical at pre-write         = PASS
canonical 11 exact blobs                       = unchanged
latest-main branchless composition             = 645adb7e... (11 paths only)
current main recursive Git tree                = fully enumerable
execution-container GitHub DNS                 = unavailable
mountable full checkout                        = unavailable
natural PR #852 workflow observation           = none
current P1-P5 focused dependency closures      = covered by exact executions
tools/repository-health-check.sh execution     = pending mandatory gate
canonical 11-path source integration           = blocked
actual replay / repeatability Evidence         = pending
```

The remaining mandatory path is unchanged:

```text
execution-time latest main full checkout
  -> apply these same canonical 11 exact blobs
  -> tools/repository-health-check.sh
  -> verify clean/same base and no parallel canonical-path overlap
  -> integrate qualified source
  -> converge existing refs with force=false
  -> Packet M actual materialization
```

No Evidence level is promoted from connector-side tree visibility alone.

#### B4 continuation — exact current-revision push probe and pre-step failure cleanup — 2026-09-22

The checkout-transport boundary was tested directly on the existing B branch without creating a new remote branch.

Starting point:

```text
main = research/selective-knowledge-routing-r6
     = a9d09b4a59b5da3d72177cb22ee236ad58ae8f41
open PR = 0
```

A temporary B-branch commit materialized the unchanged canonical 11 exact blobs over that exact main and added one branch-local push workflow whose only purpose was to run the mandatory Repository-health harness.

```text
probe commit:
  585c2599d957968ecf6ed57aaccc2f7d2ef60d00

probe tree:
  8bf28c158954a14bf508652d937361f2545d9ee4

workflow:
  .github/workflows/r7b-repository-health-probe.yml

workflow contract:
  checkout exact branch revision
  setup Go from go.mod
  record git revision / clean state / Go platform
  bash tools/repository-health-check.sh
```

The push created an actual current-revision GitHub Actions run:

```text
run:
  35694275033

workflow:
  R7B repository health probe

event:
  push

head:
  585c2599d957968ecf6ed57aaccc2f7d2ef60d00

created:
  2026-09-22T06:18:54Z
  2026-09-22 15:18:54 JST

completed:
  2026-09-22T06:18:57Z
  2026-09-22 15:18:57 JST

job:
  106637611914 repository-health

result:
  failure

steps:
  null
```

This is stronger infrastructure Evidence than replaying an old workflow because the failed job targets the exact current canonical packet on the execution-time current B branch. However, it still terminates before observable workflow-step allocation. Therefore it proves neither checkout failure nor source/test failure, and it does not execute `tools/repository-health-check.sh`.

No unqualified source is retained on the branch after the probe. The canonical 11 changes and the temporary workflow were removed in a normal forward commit:

```text
cleanup commit:
  b93c06f3a59f2bc7bb893b39a9f7a04152dfe20e

cleanup tree:
  a7104da5c2db03b837858c3a9e1e5e9f29b13626

current main tree at probe start:
  a7104da5c2db03b837858c3a9e1e5e9f29b13626

tree equality:
  PASS

force update:
  false
```

Thus the branch was restored byte-for-byte to the pre-probe main tree before this documentation write. The temporary health workflow is not proposed for main, and the canonical 11 source packet remains unintegrated.

Updated B4 Evidence height:

```text
current P1-P5 focused dependency closures      = covered by exact executions
exact current-revision hosted push probe       = executed
hosted job allocation                          = failure before steps
observable workflow steps                      = unavailable
probe source/workflow retained after test      = no
post-probe branch tree == pre-probe main tree  = PASS
tools/repository-health-check.sh execution     = pending mandatory gate
canonical 11-path source integration           = blocked
actual replay / repeatability Evidence         = pending
```

The remaining integration rule is unchanged: a future environment must allocate the job far enough to execute the full Repository-health harness on execution-time latest main plus the same canonical 11 exact blobs. No source integration is authorized from the pre-step failure.

#### B4 continuation — hosted runner label split probe — 2026-09-22

After PR #854 converged all refs to main, the remaining blocker was tested for runner-label specificity on the same existing B branch.

A temporary current-main qualification commit materialized the unchanged canonical 11 exact blobs and a two-job branch-local workflow:

```text
probe commit:
  8e64573a65cc0587f4be61b1679bd053cbfb917f

base main:
  04c70d1b2b1dbc9ca89c0d68161fec89778eb947

base tree:
  581246881051e38670d2304b7d3e54a2b574ca32

probe workflow:
  ubuntu-22.04
  ubuntu-24.04

each job:
  checkout exact branch revision
  setup Go from go.mod
  record revision / clean state / platform
  bash tools/repository-health-check.sh
```

GitHub Actions run:

```text
run:
  35694713314

head:
  8e64573a65cc0587f4be61b1679bd053cbfb917f

job 106638943505:
  name       = repository-health (ubuntu-22.04)
  conclusion = failure
  steps      = null

job 106638943589:
  name       = repository-health (ubuntu-24.04)
  conclusion = failure
  steps      = null
```

Neither explicit Linux runner label reached an observable workflow step. Combined with the previous current-revision `ubuntu-latest` probe, the available Evidence now shows the blocker is not specific to the `ubuntu-latest` alias.

No Repository-health result is inferred:

```text
checkout executed                         = no Evidence
setup-go executed                         = no Evidence
tools/repository-health-check.sh executed = no
source/test failure                       = not established
hosted runner pre-step allocation blocker = reproduced on 22.04 and 24.04
```

The unqualified source materialization and temporary workflow were removed in a normal forward cleanup commit:

```text
cleanup commit:
  18b0ec7ca4f94b2a2aa2be59fda806820453f2ee

cleanup tree:
  581246881051e38670d2304b7d3e54a2b574ca32

current main tree at probe start:
  581246881051e38670d2304b7d3e54a2b574ca32

tree equality:
  PASS

force update:
  false
```

The infrastructure observation is also recorded in issue #513. The integration boundary remains unchanged: only an environment that actually executes the full Repository-health harness on execution-time latest main plus the same canonical 11 blobs can authorize source integration.




#### B4 continuation — preserve replay-runner executable tree mode — 2026-09-22

A Git-tree audit found one qualification detail that blob-only checks do not cover: the operational replay runner must retain its executable file mode.

The canonical runner content identity is unchanged:

```text
tools/run-first-web-target-r7b-replay.sh
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

However, Git blob identity does not include the tree entry mode. Historical operational candidates that carried the executable runner correctly used:

```text
99eb861e71cfb225cf2f2ccf3db1f96feb716645
  runner mode = 100755
  runner blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

The two recent hosted-health probe materializations carried the same bytes but accidentally downgraded the tree entry to non-executable:

```text
585c2599d957968ecf6ed57aaccc2f7d2ef60d00
  runner mode = 100644
  runner blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03

8e64573a65cc0587f4be61b1679bd053cbfb917f
  runner mode = 100644
  runner blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

This does **not** change the interpretation of either hosted-runner probe: both failed before any observable workflow step was allocated, so the replay runner was never invoked and its mode cannot have caused those pre-step failures.

The qualification packet itself must nevertheless preserve the operational mode. A corrected branchless packet was therefore composed over current main:

```text
parent:
  eea3aa6774ff02585db0dfcf4e7b4eb9aac620c2

parent tree:
  b827026439855e68b354eb7fa9b818cfd8f0ef19

mode-aware branchless candidate:
  47eb285f7de7eafc01f725b3147e50d7a47badbc

candidate tree:
  027c4c9386659055a078bcf2c44c6d9e3177b4db

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only

runner tree entry:
  mode = 100755
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

The branchless candidate remains a composition target only. It is not integration authority and must not be merged or cherry-picked around Repository health.

Qualification authority is therefore refined from blob-only identity to tree-aware identity:

```text
10 Go source/test paths:
  mode = 100644
  blob = the already-fixed canonical blob identity

tools/run-first-web-target-r7b-replay.sh:
  mode = 100755
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

A health-capable integration working tree must verify both byte and mode identity before running the mandatory health harness, for example:

```bash
test "$(git hash-object tools/run-first-web-target-r7b-replay.sh)" =   "2f55ec06f29bbca8f4a9e6ce04816bd311a8af03"

test "$(git ls-files -s tools/run-first-web-target-r7b-replay.sh | awk '{print $1}')" =   "100755"

tools/repository-health-check.sh
```

Updated boundary:

```text
canonical source blob identities             = fixed
canonical runner tree mode                  = 100755 required
current mode-aware branchless packet         = 47eb285f... (11 paths only)
hosted 22.04 / 24.04 pre-step failures       = unchanged interpretation
tools/repository-health-check.sh execution   = pending mandatory gate
canonical source integration                 = blocked
actual replay / repeatability Evidence       = pending
```

#### B4 continuation — cross-platform hosted-runner allocation boundary — 2026-09-22

After PR #856, all 19 remote refs were identical at:

```text
807e0712ea02b6ab25246ab37edf811efb12902f
```

with no open PR and no unique branch-ahead work.

The mode-aware canonical 11-path packet was then materialized over that exact main on the existing B branch. The replay runner tree tuple was explicitly preserved:

```text
tools/run-first-web-target-r7b-replay.sh
  mode = 100755
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

A temporary cross-platform health workflow tested whether the known pre-step failure was limited to Linux GitHub-hosted runners.

```text
probe commit:
  8ff2fe61d72a31a210ebe543fcc7b8482f5f1696

probe tree:
  49af270396607fe565e272b368767bcdfc89a6d0

run:
  35696570153

jobs:
  106644642717  repository-health (macos-15, bash)
  106644642725  repository-health (windows-2025, bash)
```

Raw job metadata:

```text
macos-15:
  conclusion       = failure
  runner_id        = 0
  runner_name      = ""
  runner_group_id  = 0
  runner_group_name= ""
  steps            = []

windows-2025:
  conclusion       = failure
  runner_id        = 0
  runner_name      = ""
  runner_group_id  = 0
  runner_group_name= ""
  steps            = []
```

Neither job reached checkout, setup-go, packet-identity verification, or Repository health. Combined with the previous current-revision Linux probes:

```text
ubuntu-latest  = pre-step failure
ubuntu-22.04   = pre-step failure
ubuntu-24.04   = pre-step failure
macos-15       = pre-step failure
windows-2025   = pre-step failure
```

the observable blocker is now reproduced across the principal GitHub-hosted OS families. This narrows the boundary to hosted-runner allocation before a runner is assigned, but it does not establish a specific cause such as billing, policy, quota, or permission.

The repository itself was searched for an alternate runner path:

```text
self-hosted workflow declarations = none found
runner-group workflow declarations = none found
```

Repository runner-registration and Actions-permission endpoints are outside the connector's allowed read surface, so the existence or configuration of registered self-hosted runners cannot be asserted from this session.

The probe source materialization and temporary workflow were removed immediately in a normal forward commit:

```text
cleanup commit:
  5167d28754a91ae42904f1178dd9267072176b9a

cleanup tree:
  9cf3f8933657317e7161b3f4eb56317552711433

probe-start main tree:
  9cf3f8933657317e7161b3f4eb56317552711433

tree equality:
  PASS

force update:
  false
```

Issue #513 contains the same infrastructure observation.

Updated B4 Evidence height:

```text
canonical 11 blob identities                  = fixed
canonical replay-runner mode                  = 100755
current P1-P5 focused dependency closures     = covered by exact executions
Linux hosted allocation                       = pre-step failure
macOS hosted allocation                       = pre-step failure / runner_id 0
Windows hosted allocation                     = pre-step failure / runner_id 0
repository-defined self-hosted path           = none found
registered self-hosted runner state           = not observable in this session
tools/repository-health-check.sh execution    = pending mandatory gate
canonical 11-path source integration          = blocked
actual replay / repeatability Evidence        = pending
```

No source integration is authorized from runner-allocation Evidence alone. The mandatory gate remains an actual full-checkout execution of `tools/repository-health-check.sh` on execution-time latest main plus the same mode-aware canonical packet.

#### B4 continuation — hosted allocation onset boundary — 2026-09-22

The Actions history was compared across the last known successful allocation and the earliest current-style pre-step failures.

Last confirmed successful hosted allocations:

```text
run 33634416735
  created     = 2026-09-02T13:13:14Z
  label       = ubuntu-24.04
  runner_id   = 1000075820
  runner_name = GitHub Actions 1000075820
  steps       = populated
  conclusion  = success

run 33634239635 / verify
  created     = 2026-09-02T13:11:24Z
  label       = ubuntu-latest
  runner_id   = 1000075811
  runner_name = GitHub Actions 1000075811
  steps       = populated
  conclusion  = success
```

Earliest inspected failures with the current signature:

```text
run 35028517287 / job 104581243332
  created     = 2026-09-15T21:58:21Z
  label       = ubuntu-22.04
  runner_id   = 0
  runner_name = ""
  steps       = []
  conclusion  = failure

run 35028898569 / job 104582482529
  created     = 2026-09-15T22:02:44Z
  label       = ubuntu-22.04
  runner_id   = 0
  runner_name = ""
  steps       = []
  conclusion  = failure
```

The observable transition is therefore bounded by:

```text
2026-09-02  hosted runner allocation succeeds
2026-09-15  hosted job exists but no runner is assigned
```

Current-revision probes now reproduce the same pre-step boundary on:

```text
ubuntu-latest
ubuntu-22.04
ubuntu-24.04
macos-15
windows-2025
```

Public GitHub Status reports Actions operational at the time of this investigation, so no public platform-wide Actions incident is used as the explanation for this repository's failure. Absence of a public incident does not by itself prove an account-specific cause.

Public reports with the same zero-step / pre-allocation symptom exist for billing, spending-limit, and hosted-runner entitlement failures. This repository's account billing and Actions-policy settings are not readable through the current connector, and the failed jobs expose no annotation/log text that identifies a cause. Therefore:

```text
billing / spending-limit / entitlement = leading external hypothesis only
confirmed cause                         = unknown
confirmed technical boundary            = hosted runner not assigned before steps
```

Issue #513 carries the operational follow-up boundary: verify Actions enablement and account billing/budget/payment state outside this execution session, then provide the current run/job IDs to GitHub Support if those settings are clean.

This does not change the source integration rule. Canonical R7B source remains blocked until an environment actually executes the full Repository-health harness.



#### B4 continuation — ubuntu-slim shared-runner probe and transfer-artifact boundary — 2026-09-22

After PR #858 merged, all 19 remote refs were converged to current main before this probe:

```text
main:
  58353f3744d93e54962f03aa183fc6556b6dd93d

main tree:
  4cc0088b2523e063b5659e6af48719e81d1b88d8

open PR:
  0
```

The remaining GitHub-hosted variant not yet tested was `ubuntu-slim`, which uses the standard single-CPU shared-VM path rather than the normal hosted VM shape. A temporary probe on the existing B branch materialized the mode-aware canonical 11-path packet and a workflow that would:

```text
checkout full history
-> setup Go from go.mod
-> verify all 11 canonical blob identities
-> verify replay runner mode = 100755
-> tools/repository-health-check.sh
-> always create a portable Git bundle
-> always upload that bundle as an Actions artifact
```

Probe identity:

```text
probe commit:
  d40d554e381bbb6f8672cfcab0432206a339ddb0

probe tree:
  d2688fe626c3f06906500237533a72946b76394c

workflow:
  .github/workflows/r7b-ubuntu-slim-health-probe.yml

run:
  35697286875

job:
  106646854517 repository-health
```

Observed job metadata:

```text
label              = ubuntu-slim
created            = 2026-09-22T06:58:22Z
started            = 2026-09-22T06:58:22Z
completed          = 2026-09-22T06:58:35Z
conclusion         = failure
runner_id          = 0
runner_name        = ""
runner_group_id    = 0
runner_group_name  = ""
steps              = []
check annotations  = 1
job log download   = BlobNotFound
workflow artifacts = 0
```

The run remained queued longer than the normal VM probes, but no observable workflow step was ever created and no runner was assigned. Therefore the result does not execute checkout, setup-go, canonical identity verification, Repository health, Git-bundle creation, or artifact upload.

The transfer experiment is nevertheless useful because it proves that the planned artifact bridge is blocked at the same pre-step allocation boundary rather than at artifact download or bundle creation.

The temporary canonical source materialization and workflow were removed by a normal forward cleanup commit:

```text
cleanup commit:
  cbecad03f690cbb7ef42d390f0a9b56667d8dd72

cleanup tree:
  4cc0088b2523e063b5659e6af48719e81d1b88d8

probe-start main tree:
  4cc0088b2523e063b5659e6af48719e81d1b88d8

tree equality:
  PASS

force update:
  false
```

Current hosted-allocation coverage is now:

```text
ubuntu-latest   = pre-step failure
ubuntu-22.04    = pre-step failure
ubuntu-24.04    = pre-step failure
ubuntu-slim     = pre-step failure / runner_id 0
macos-15        = pre-step failure / runner_id 0
windows-2025    = pre-step failure / runner_id 0
```

No source/test failure is inferred from these runs. The mandatory source-integration gate remains an actual full-checkout execution of `tools/repository-health-check.sh` on execution-time latest main plus the same mode-aware canonical packet.

#### B4 continuation — immediate rerun recovery probe — 2026-09-22

After PR #859, `research/selective-knowledge-routing-r6` and `main` were rechecked and were identical at:

```text
405d77d278d9c9b6d16f47afc15237c870e66fe9
ahead  = 0
behind = 0
```

The latest `ubuntu-slim` probe was then re-run in place without creating a new branch or workflow:

```text
run:
  35697286875

attempt:
  2

run started:
  2026-09-22T07:09:42Z

run completed:
  2026-09-22T07:09:58Z

job:
  106649790134 repository-health

result:
  conclusion = failure
  steps      = null
```

The re-run initially entered `queued`, but it completed without any observable workflow step. Therefore it does not establish recovery and does not execute checkout, setup-go, canonical packet identity checks, Repository health, Git-bundle creation, or artifact upload.

This connector's approved GitHub read surface does not expose repository Actions permission or registered-runner endpoints. Those reads were attempted and rejected by the connector before reaching GitHub, so they provide no evidence about the repository's actual Actions policy, billing state, entitlement, or runner registration.

Updated boundary:

```text
attempt-2 immediate re-run                     = pre-step failure again
step execution                                 = not observed
Repository health                              = not executed
canonical 11-path source integration           = blocked
actual replay / repeatability Evidence         = pending
specific external cause                        = unknown
```

No new remote work branch was created. The existing B branch remains the sole write lane and must be reconverged to merged `main` after this documentation update.

#### B4 continuation — delayed rerun recovery probe attempt 3 — 2026-09-22

The existing `ubuntu-slim` probe was re-run again in place after the later documentation merges, without creating a new branch or workflow.

```text
run:
  35697286875

attempt:
  3

job:
  106663823910 repository-health

created / started:
  2026-09-22T08:01:34Z

completed:
  2026-09-22T08:02:04Z

label:
  ubuntu-slim

result:
  conclusion        = failure
  runner_id         = 0
  runner_name       = ""
  runner_group_id   = 0
  runner_group_name = ""
  steps             = []

job log download:
  BlobNotFound
```

Attempt 3 therefore reproduces the same pre-step allocation signature as attempts 1 and 2. No checkout, setup-go, canonical packet identity verification, Repository health, Git-bundle creation, or artifact upload step was created.

The delayed re-run is useful only as a recovery check:

```text
hosted-runner recovery at attempt 3 = not observed
step execution                       = not observed
tools/repository-health-check.sh     = not executed
canonical 11-path source integration = still blocked
actual replay / repeatability        = still pending
specific external cause              = still unknown
```

This remains a symptom observation, not proof of a billing, entitlement, quota, policy, or runner-registration root cause.

No new remote branch or workflow was created for this probe. The existing R7B lane remains the sole write lane and must be reconverged to merged `main` after this documentation update.

#### B4 continuation — hosted allocation onset narrowed to an 8-minute boundary — 2026-09-22

The previously recorded onset boundary was re-audited directly against Actions run-attempt job payloads. Issue #513 already preserved an earlier September 4 example than the September 15 pair cited in the prior B4 section; a repository-wide time-window scan narrows the boundary further.

Last confirmed successful hosted allocation:

```text
run:
  33634416735

run created:
  2026-09-02T13:13:13Z

job:
  100261564372 connect

label:
  ubuntu-24.04

job started:
  2026-09-02T13:13:17Z

job completed:
  2026-09-02T13:13:23Z

runner_id:
  1000075820

runner_name:
  GitHub Actions 1000075820

steps:
  6 populated steps

conclusion:
  success
```

A parallel CI run immediately before it also had ordinary hosted allocation:

```text
run 33634239635
  verify runner_id               = 1000075811
  postgres-integration runner_id = 1000075812
  steps                          = populated
  conclusion                     = success
```

The first directly confirmed job with the current allocation-failure signature is:

```text
run:
  33635220666

run created:
  2026-09-02T13:21:24Z

attempt-1 jobs created:
  2026-09-02T13:21:25Z

verify:
  job         = 100264290235
  label       = ubuntu-latest
  runner_id   = 0
  runner_name = ""
  steps       = []
  conclusion  = failure

postgres-integration:
  job         = 100264290641
  label       = ubuntu-latest
  runner_id   = 0
  runner_name = ""
  steps       = []
  conclusion  = failure
```

Two intervening runs at `13:18:57Z` and `13:20:02Z` failed with no jobs at all. Because they never created a job object, they are not classified as runner-allocation failures here.

The directly evidenced transition is therefore:

```text
2026-09-02T13:13:23Z
  hosted runner job completes successfully

        8 minutes 2 seconds

2026-09-02T13:21:25Z
  hosted jobs exist with runner_id=0 / steps=[]
```

This supersedes the weaker September 2 -> September 15 and September 2 -> September 4 onset bounds previously quoted for B4. It does not establish the external cause, but it materially narrows when the repository/account execution state changed.

Updated infrastructure boundary:

```text
last directly confirmed allocated hosted job = 2026-09-02T13:13:23Z
first directly confirmed unassigned job       = 2026-09-02T13:21:25Z
transition window                             = 8m02s
public source/test failure in that window     = not established
specific billing/policy/entitlement cause     = unknown
Repository health                             = still not executed
canonical source integration                  = still blocked
```

The source-integration rule remains unchanged. This historical narrowing is diagnostic Evidence only and does not substitute for execution of `tools/repository-health-check.sh`.



#### B4 continuation — public/private hosted-runner scope comparison — 2026-09-22

The runner-allocation failure was compared against other repositories owned by the same GitHub account to distinguish an account-wide hosted-runner outage from a private-repository-specific boundary.

A public repository under the same owner successfully allocated and executed an `ubuntu-latest` runner after the TEI private-repository failure window had already begun:

```text
repository:
  hat47x/sui-sensemaking
visibility:
  public
run:
  35437442624
created:
  2026-09-19T10:27:12Z
job:
  105882400375 verify
label:
  ubuntu-latest
runner_id:
  1000078128
runner_name:
  GitHub Actions 1000078128
steps:
  populated
conclusion:
  success
```

Representative private repositories under the same owner show the same pre-step signature as TEI during the same period:

```text
repository:
  hat47x/sei-cognition
visibility:
  private
run:
  35693341522
job:
  106634781467 e78-assistance-runtime
label:
  ubuntu-latest
runner_id:
  0
runner_name:
  ""
steps:
  []
conclusion:
  failure

repository:
  hat47x/vs-bear-simulator
visibility:
  private
run:
  35694974389
job:
  106639738000 static-validation
label:
  ubuntu-latest
runner_id:
  0
runner_name:
  ""
steps:
  []
conclusion:
  failure
```

This comparison materially narrows the operational boundary:

```text
GitHub-hosted runners for this account, all repositories = not globally unavailable
public-repository hosted allocation after 2026-09-15       = observed working
multiple private-repository hosted allocations             = observed failing before steps
TEI-specific workflow/source defect                         = not supported by this comparison
private-repository entitlement/billing/policy class         = materially strengthened hypothesis
specific cause within that class                            = still unconfirmed
```

No account billing, spending-limit, payment, Actions-policy, or private-runner entitlement setting is readable through the current connector. Therefore none of those causes is promoted from hypothesis to confirmed fact.

Operational follow-up is now more targeted: inspect the account-level private-repository Actions billing/budget/payment/entitlement state rather than treating the symptom as a TEI-only or all-Actions outage. If those settings are clean, the cross-repository run/job IDs above provide a stronger support case because the same owner has working public hosted allocation while several private repositories fail before runner assignment.

This does not change the R7B source qualification rule. Canonical source remains blocked until a full checkout actually executes `tools/repository-health-check.sh`.

#### B4 continuation — private-repository scope hypothesis corrected by SEI counterexample — 2026-09-22

The cross-repository comparison after PR #861 was expanded across the connected repositories owned by the same account. The result requires a correction to the previous interpretation that the observed hosted-runner failure could be treated as a single private-repository-wide transition.

A private repository under the same owner continued to receive and execute a normal GitHub-hosted runner after TEI had already entered the pre-step failure state.

```text
repository:
  hat47x/sei-cognition

visibility:
  private

successful run:
  33670697154
  Actual Model Artifact Exchange E3b

workflow:
  .github/workflows/actual-model-artifact-exchange-e3b.yml

workflow_id:
  348426986

created:
  2026-09-02T19:00:34Z

successful job:
  100383296117
  two actual models / historical Application preservation

label:
  ubuntu-latest

runner_id:
  1000076312

runner_name:
  GitHub Actions 1000076312

job started:
  2026-09-02T19:00:42Z

job completed:
  2026-09-02T19:03:39Z

steps:
  populated through Complete job

conclusion:
  success
```

This is more than five hours after TEI's first directly confirmed current-signature jobs at `2026-09-02T13:21:25Z`.

The same SEI workflow then switched to the same pre-step failure signature less than five minutes later:

```text
failed run:
  33671503776
  Actual Model Artifact Exchange E3b

workflow:
  .github/workflows/actual-model-artifact-exchange-e3b.yml

workflow_id:
  348426986

created:
  2026-09-02T19:08:28Z

job:
  100385957813
  two actual models / historical Application preservation

label:
  ubuntu-latest

runner_id:
  0

runner_name:
  ""

steps:
  []

completed:
  2026-09-02T19:08:32Z

conclusion:
  failure
```

A second SEI workflow created in the same second shows the same boundary:

```text
run:
  33671503688
  Semantic Runtime Conformance

job:
  100385956315 runtime-conformance

runner_id:
  0

steps:
  []

conclusion:
  failure
```

The SEI transition is therefore directly bounded by:

```text
2026-09-02T19:03:39Z
  private-repository ubuntu-latest job completes successfully

        4 minutes 49 seconds

2026-09-02T19:08:28Z
  same private repository creates unassigned hosted jobs
  runner_id = 0
  steps     = []
```

The TEI and SEI onset windows are materially different:

```text
TEI:
  last allocated completion = 2026-09-02T13:13:23Z
  first unassigned jobs     = 2026-09-02T13:21:25Z

SEI:
  allocated completion      = 2026-09-02T19:03:39Z
  unassigned jobs           = 2026-09-02T19:08:28Z
```

Therefore the previous `private-repository entitlement/billing/policy class` observation must be interpreted more narrowly:

```text
all repositories account-wide outage                 = contradicted by public successes
all private repositories failing from TEI onset      = contradicted by SEI private success
single simultaneous private-repository transition    = not supported
current multi-private pre-step symptom                = observed
same metadata signature implies same root cause       = not established
repository-specific policy/budget state               = possible
later account/private-wide state change               = possible
multiple independent causes with same signature       = possible
specific cause                                        = still unknown
```

A direct scan of connected private repositories after TEI's `13:21:25Z` boundary also found no later success in TEI, `markdown-matrix-injector`, or `vs-bear-simulator`, while SEI has the confirmed success above. Repositories with no Actions runs in that interval do not contribute evidence either way.

TEI's own CI workflow identity was also compared across the successful and first failed runs:

```text
successful run:
  33634239635
  workflow_id = 345688230
  path        = .github/workflows/ci.yml
  verify/postgres jobs allocated normally

failed run:
  33635220666
  workflow_id = 345688230
  path        = .github/workflows/ci.yml
  verify/postgres jobs both runner_id=0 / steps=[]
```

Both workflow versions retain the same `ubuntu-latest` job allocation shape. Their source differs only in later PostgreSQL semantic-mapping verification logic; no runner-label or job-allocation change explains the transition.

Operationally, `runner_id=0 / steps=[]` must now be treated as a **symptom class**, not as proof of one billing, entitlement, policy, or quota root cause. Account and repository settings still need direct inspection outside the connector to identify the actual rejection reason.

This correction does not alter the R7B integration gate:

```text
tools/repository-health-check.sh actual execution = pending
canonical 11-path source integration             = blocked
actual replay / repeatability Evidence           = pending
```

#### B4 continuation — account-wide included-minutes exhaustion weakened; repository-scoped stop remains plausible — 2026-09-22

The runner-allocation symptom was compared with current GitHub Actions billing semantics and with successful-run history across private repositories owned by the same personal account.

Current GitHub billing semantics relevant to this investigation:

```text
standard GitHub-hosted runners in public repositories:
  free

standard GitHub-hosted runners in private repositories:
  consume the repository owner's included Actions minutes
  then require paid usage when included quota is exceeded

included Actions minutes:
  account/plan allowance
  reset each billing cycle

Actions budgets:
  may be scoped at account or repository level
  may be configured to stop usage when the budget threshold is reached
```

The connector cannot read the account's actual billing usage, budget configuration, payment state, or repository-specific budget state. The current GitHub Billing Usage REST endpoint was tested through the connector and rejected by the connector's allowed endpoint surface before reaching GitHub, so no billing amount is inferred.

Run-count scan for the interval:

```text
2026-09-01T00:00:00Z
  through
2026-09-02T13:21:24Z

TEI successful workflow runs:
  323

SEI successful workflow runs:
  1245

other connected private repositories inspected:
  no successful runs in this interval
  or no relevant Actions history
```

Run counts are not billed minutes and are not converted into a quota estimate here. They only establish that both TEI and SEI were actively consuming private-repository hosted execution during the billing period.

The key temporal counterexample remains stronger than the raw counts:

```text
TEI first directly confirmed unassigned hosted jobs:
  2026-09-02T13:21:25Z
  runner_id = 0
  steps     = []

SEI later private-repository hosted success:
  run 33670697154
  job 100383296117
  ubuntu-latest
  runner_id = 1000076312
  completed = 2026-09-02T19:03:39Z
  steps     = populated
  conclusion= success
```

Because GitHub documents the normal included-minute allowance for private repositories as an owner-account plan allowance, **simple account-wide exhaustion of included Actions minutes at the TEI 13:21Z transition cannot by itself explain the later SEI private-repository hosted success**.

Therefore the billing-related hypothesis is narrowed:

```text
simple account-wide included-minutes exhaustion at TEI onset:
  weakened / temporally inconsistent with later SEI private success

public-repository free-runner behavior:
  consistent with GitHub billing semantics
  not diagnostic by itself

repository-scoped Actions budget with hard stop:
  compatible with staggered repository behavior
  not confirmed

repository-specific policy / entitlement / billing state:
  compatible
  not confirmed

later account-wide state change after SEI success:
  compatible
  not confirmed

multiple independent causes producing runner_id=0 / steps=[]:
  compatible
  not confirmed
```

This means the most useful manual billing check is no longer just "did the personal account exhaust its included minutes?" It is:

```text
Billing & licensing
  -> Budgets and alerts
     -> Actions product/SKU budgets
     -> account scope and repository scopes
     -> whether "Stop usage when budget limit is reached" is enabled

plus:
  payment method / paid-usage eligibility
  repository Actions policy / enablement
```

The diagnostic result does not alter source qualification:

```text
actual tools/repository-health-check.sh execution = pending
canonical 11-path source integration              = blocked
actual replay / repeatability Evidence            = pending
```



#### B4 continuation — same-head mixed hosted allocation and latest-main packet rebind — 2026-09-22

The delayed recovery probe attempt 3 is already preserved on the existing B4 branch. Before adding this checkpoint, the branch is one documentation commit ahead of current main and has no canonical source-path delta.

Current integration base:

```text
main:
  9c5cfe85fcccf016f5c7e117bb2a858d17042422
  Merge PR #863 — r7b: narrow Actions billing hypothesis

B4 branch:
  ahead  = 1
  behind = 0
  changed path = plan/issue/000069-first-web-target-r7b-exact-t-replay.md only
```

The unchanged canonical 11 exact blobs were recomposed over this exact main without creating or moving another remote branch.

```text
parent:
  9c5cfe85fcccf016f5c7e117bb2a858d17042422

parent tree:
  decaf9d7f0156a47338c162a0a78baa68a763646

branchless qualification candidate:
  ea5b9e635ab6479d79b5ff2912b906df3c3a5a4f

candidate tree:
  9db7c07f4413a8cdea6bb232edc1e136f6c3039d

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only
```

The candidate is still a qualification composition target only. It is not integration authority.

A deeper check-run comparison also changes how the SEI timing evidence must be interpreted.

For the first confirmed TEI pre-step CI failure commit:

```text
TEI head:
  83a7eef2731284eed64a3b815374dccc4420c45f

verify check:
  100272370585
  conclusion        = failure
  annotations_count = 1

postgres-integration check:
  100272370239
  conclusion        = failure
  annotations_count = 1
```

The connector exposes the annotation count but does not allow the check-run annotations endpoint, so the annotation body cannot be read in this session. No billing/policy/error message is inferred from the count alone.

More importantly, one SEI head SHA shows successful and pre-step-failed hosted jobs at essentially the same time:

```text
SEI head:
  765eb189f97d7dba402cec864960cecb3a5a91a6

successful workflow:
  run 33670697154
  Actual Model Artifact Exchange E3b
  job 100383296117
  two actual models / historical Application preservation
  event      = pull_request
  runs-on    = ubuntu-latest
  started    = 2026-09-02T19:00:42Z
  completed  = 2026-09-02T19:03:39Z
  steps      = populated through Complete job
  conclusion = success

pre-step failed workflow on the same head:
  run 33670696988
  Semantic Runtime Conformance
  job 100383295376
  event      = pull_request
  runs-on    = ubuntu-latest
  started    = 2026-09-02T19:00:34Z
  completed  = 2026-09-02T19:00:39Z
  steps      = null
  conclusion = failure

another pre-step failed workflow on the same head:
  run 33670696938
  External Model-backed Method E3b M1b
  job 100383295882
  event      = pull_request
  runs-on    = ubuntu-latest
  started    = 2026-09-02T19:00:35Z
  completed  = 2026-09-02T19:00:38Z
  steps      = null
  conclusion = failure
```

The three workflow definitions at that exact head all use GitHub-hosted `ubuntu-latest`. The successful and failed definitions also share ordinary `contents: read` permissions and standard `actions/checkout` / setup actions; no failing-only runner label difference was found.

Therefore the earlier 4m49s SEI boundary remains valid only for the **same E3b workflow**:

```text
E3b success completes:
  2026-09-02T19:03:39Z

same E3b workflow next observed failure:
  2026-09-02T19:08:28Z
```

It is not a valid repository-wide transition boundary, because other workflows on the same repository and same head were already failing before steps while E3b still acquired and used a hosted runner.

Updated diagnostic boundary:

```text
single account-wide runner outage:
  not supported

single private-repository-wide transition:
  not supported

single deterministic repository policy switch at 19:03Z:
  not supported by same-head mixed allocation

workflow/job admission symptom:
  directly observed

repository-scoped budget/policy state:
  still possible
  but not established as a simple all-jobs boundary

concurrent admission / quota-update race:
  possible
  not established

multiple independent causes with the same runner_id=0 / steps=[] signature:
  possible

specific external cause:
  still unknown
```

The historical successful-run artifact route was also checked as a possible way to bypass the currently unavailable checkout path without weakening Evidence requirements:

```text
TEI last confirmed CI success:
  run 33634239635
  workflow artifacts = none

SEI E3b confirmed hosted success:
  run 33670697154
  workflow artifacts = none
```

No reusable full-checkout or source snapshot artifact was available from those successful runs.

Current Evidence height:

```text
delayed ubuntu-slim recovery attempt 3          = pre-step failure reproduced
same-head SEI mixed hosted allocation           = directly observed
check-run failure annotations                   = present, body inaccessible
latest-main canonical 11 composition            = ea5b9e63... (11 paths only)
historical full-checkout artifact salvage       = unavailable
execution-container GitHub DNS                  = unavailable
tools/repository-health-check.sh execution      = pending mandatory gate
canonical 11-path source integration            = blocked
actual replay / repeatability Evidence          = pending
```

The next high-value external observation is the actual check-run annotation text and the account/repository Actions billing/budget/policy state. Neither can be read through the current connector. Until an execution-capable checkout exists, the mandatory source-integration rule remains unchanged.

#### B4 continuation — same-head burst shows intermittent hosted admission, not a deterministic workflow split — 2026-09-22

The SEI same-head mixed-allocation case was expanded across every workflow run created for the exact head:

```text
repository:
  hat47x/sei-cognition

head:
  765eb189f97d7dba402cec864960cecb3a5a91a6

runs created at:
  2026-09-02T19:00:34Z

workflow runs on this head:
  7 total
```

Job-level outcomes:

```text
6 jobs:
  label       = ubuntu-latest
  runner_id   = 0
  runner_name = ""
  steps       = []
  conclusion  = failure

1 job:
  label       = ubuntu-latest
  runner_id   = 1000076312
  runner_name = GitHub Actions 1000076312
  steps       = populated
  conclusion  = success
```

The successful job was:

```text
run:
  33670697154
job:
  100383296117
workflow:
  Actual Model Artifact Exchange E3b
started:
  2026-09-02T19:00:42Z
completed:
  2026-09-02T19:03:39Z
```

The six failed jobs were created in the same second and completed within a few seconds without runner assignment. Their job timestamps show immediate pre-step failure rather than ordinary waiting behind the successful job.

The workflow definitions at this exact head were compared directly. The successful workflow does not have a unique runner class or privileged permission boundary:

```text
successful E3b workflow:
  runs-on     = ubuntu-latest
  permissions = contents: read
  checkout    = actions/checkout@v4
  setup       = setup-python / setup-go
  concurrency = workflow/ref group

failed workflows include examples with the same:
  runs-on     = ubuntu-latest
  permissions = contents: read
  checkout    = actions/checkout@v4
  setup       = setup-python / setup-go
  concurrency = workflow/ref group
```

A failed workflow also uses an explicit timeout, so `timeout-minutes` does not distinguish the successful admission.

The previous successful-run scan adds one more constraint:

```text
SEI latest success before the 19:00Z E3b success:
  run 33634890983
  created 2026-09-02T13:18:02Z

no SEI successful workflow runs observed:
  2026-09-02T13:18:24Z .. 2026-09-02T19:00:33Z

isolated later success:
  run 33670697154
  created 2026-09-02T19:00:34Z
  completed 2026-09-02T19:03:39Z
```

Therefore the observed behavior is best recorded as an **intermittent hosted admission symptom**:

```text
deterministic workflow-config cause:
  not supported by the compared fields

simple repository-wide hard stop:
  not sufficient to describe the observed 6-fail / 1-success burst

ordinary one-job concurrency queue:
  not supported by failed job timestamps

transient admission pulse / threshold race:
  compatible
  not proven

specific billing / budget / policy / platform cause:
  still unknown
```

This strengthens the requirement not to promote `runner_id=0 / steps=[]` into a single inferred root cause without the inaccessible check annotation or account/repository Actions state.

The R7B integration boundary remains unchanged:

```text
tools/repository-health-check.sh actual execution = pending
canonical 11-path source integration             = blocked
actual replay / repeatability Evidence           = pending
```

#### B4 continuation — service-container admission probe, four pre-step attempts — 2026-09-22

A new current-main qualification probe tested whether adding a PostgreSQL service container changed hosted-runner admission behavior.

The existing B branch was used; no new remote branch was created. The probe temporarily materialized:

```text
canonical 11 exact blobs
+
.github/workflows/r7b-service-container-health-probe.yml
```

Probe commit:

```text
ddb57b02dd4a3a383c6734ff39d9d5e2711c5b30
```

Workflow contract:

```text
runs-on:
  ubuntu-latest

services:
  postgres:17

then:
  checkout exact branch revision
  setup Go
  verify canonical 11 blob identities and replay-runner mode
  bash tools/repository-health-check.sh
  create portable Git bundle
  upload bundle artifact
```

GitHub Actions run:

```text
run:
  35704623810

head:
  ddb57b02dd4a3a383c6734ff39d9d5e2711c5b30
```

All four attempts failed at the same pre-step boundary:

```text
attempt 1:
  job          = 106670449951
  created      = 2026-09-22T08:24:24Z
  completed    = 2026-09-22T08:24:27Z
  runner_id    = 0
  runner_name  = ""
  steps        = []
  conclusion   = failure

attempt 2:
  job          = 106670736447
  created      = 2026-09-22T08:25:22Z
  completed    = 2026-09-22T08:25:25Z
  runner_id    = 0
  runner_name  = ""
  steps        = []
  conclusion   = failure

attempt 3:
  job          = 106670894525
  created      = 2026-09-22T08:25:54Z
  completed    = 2026-09-22T08:25:57Z
  runner_id    = 0
  runner_name  = ""
  steps        = []
  conclusion   = failure

attempt 4:
  job          = 106671004859
  created      = 2026-09-22T08:26:16Z
  completed    = 2026-09-22T08:26:19Z
  runner_id    = 0
  runner_name  = ""
  steps        = []
  conclusion   = failure
```

No service container was started, because the job never acquired a runner. No checkout or workflow command executed.

Therefore:

```text
PostgreSQL service declaration changes admission outcome:
  no Evidence

service container started:
  no

checkout executed:
  no

canonical blob/mode verification executed:
  no

tools/repository-health-check.sh executed:
  no

portable Git bundle created:
  no

artifact uploaded:
  no
```

The canonical source packet and temporary workflow were immediately removed in a normal forward cleanup commit:

```text
cleanup commit:
  ef7adf87c60f2238c0e942599e2847a752aaf712

cleanup tree:
  7f803008316df9ed356450e9af6d928f4ef94954

current main tree before probe:
  7f803008316df9ed356450e9af6d928f4ef94954

tree equality:
  PASS

force update:
  false
```

This probe adds one useful boundary only: the currently observed pre-step failure is not bypassed merely by selecting a workflow shape that includes a PostgreSQL service container. It does not identify the underlying cause.

External public evidence was also checked separately. GitHub Status reports no public incident on 2026-09-02, while recent GitHub Community reports show similar pre-allocation symptoms under several different billing / budget / service-side contexts. Those reports are corroborative only and are not treated as evidence for this account's root cause.

Current B4 gate remains:

```text
hosted step execution                         = unavailable in current probes
tools/repository-health-check.sh execution    = pending mandatory gate
canonical 11-path source integration           = blocked
actual replay / repeatability Evidence         = pending
specific external cause                        = unknown
```



#### B4 continuation — historical rerun carry-forward correction and bounded service-probe result — 2026-09-22

A historical successful TEI CI job was re-run to test whether an unchanged old workflow could still acquire a hosted runner under the current external state.

Historical run:

```text
run:
  33634239635

original head:
  fc6e67ad70689ef2709ea05a5837b8618da16e30

original workflow:
  .github/workflows/ci.yml

attempt 1:
  verify
    job       = 100260961812
    runner_id = 1000075811
    steps     = populated
    conclusion= success

  postgres-integration
    job       = 100260962264
    runner_id = 1000075812
    steps     = populated
    conclusion= success
```

The `verify` job was requested for re-run. Attempt 2 shows:

```text
verify:
  job          = 106669226103
  run_attempt  = 2
  started      = 2026-09-22T08:20:11Z
  completed    = 2026-09-22T08:20:15Z
  runner_id    = 0
  runner_name  = ""
  steps        = 0
  conclusion   = failure

postgres-integration:
  job          = 106669227881
  run_attempt  = 2
  started      = 2026-09-02T13:11:27Z
  completed    = 2026-09-02T13:12:17Z
  runner_id    = 1000075812
  runner_name  = GitHub Actions 1000075812
  steps        = 11
  conclusion   = success
```

The second record is **not evidence that PostgreSQL integration acquired a runner on 2026-09-22**. Its execution timestamps and runner identity are the original 2026-09-02 values. GitHub carried the prior successful job result into the re-run attempt while only the requested `verify` job actually re-entered admission.

Therefore the earlier provisional interpretation that a service-container job still acquired a runner under the current state is withdrawn.

To test the service-container hypothesis directly, a temporary mode-aware canonical health workflow was materialized on the existing B branch:

```text
probe commit:
  ddb57b02dd4a3a383c6734ff39d9d5e2711c5b30

workflow:
  .github/workflows/r7b-service-container-health-probe.yml

run:
  35704623810

job:
  postgres-admission-health

runner:
  ubuntu-latest

service:
  postgres:17

Repository-health semantics:
  canonical 11 blob identities checked first
  replay runner mode 100755 checked first
  PostgreSQL DSN not exported to the Repository-health process
  service host port moved to 55432
  tools/repository-health-check.sh would run only after those checks
```

Five bounded attempts were made on the exact same probe commit and workflow:

```text
attempt 1:
  job = 106670449951
  runner_id = 0
  steps = 0
  failure

attempt 2:
  job = 106670736447
  runner_id = 0
  steps = 0
  failure

attempt 3:
  job = 106670894525
  runner_id = 0
  steps = 0
  failure

attempt 4:
  job = 106671004859
  runner_id = 0
  steps = 0
  failure

attempt 5:
  job = 106671126587
  runner_id = 0
  steps = 0
  failure
```

No attempt reached checkout, setup-go, canonical identity verification, Repository health, Git-bundle creation, or artifact upload.

The direct conclusion is narrower than the abandoned provisional hypothesis:

```text
service-container presence alone causes hosted admission:
  not supported

historical carry-forward job metadata proves current runner allocation:
  false

current mode-aware canonical health execution:
  still not observed

runner_id=0 / steps=[]:
  remains an admission symptom class
  root cause still unknown
```

The temporary workflow and canonical source materialization were removed by a forward cleanup commit:

```text
cleanup commit:
  ef7adf87c60f2238c0e942599e2847a752aaf712

cleanup tree:
  7f803008316df9ed356450e9af6d928f4ef94954

probe-start main tree:
  7f803008316df9ed356450e9af6d928f4ef94954

tree equality:
  PASS

force update:
  false
```

The mandatory R7B integration boundary is unchanged:

```text
tools/repository-health-check.sh actual execution = pending
canonical 11-path source integration             = blocked
actual replay / repeatability Evidence           = pending
```


#### B4 continuation — official Actions hard-budget semantics and latest-main packet rebind — 2026-09-22

Before this checkpoint, all 19 remote refs were re-audited and were identical to current `main`, with no open pull request and no unique ahead work.

Current integration base:

```text
main:
  978fc324f8bf2dbf734ac7636fee4ccb929c02ba

main tree:
  ca0c5ec2b2bdd1fdec4a07e71810cefff2a9e348
```

The unchanged mode-aware canonical 11-path packet was recomposed over that exact main without creating or moving another remote branch:

```text
branchless qualification candidate:
  374e331d86938285e0b47c1517bd120cf2c9275b

candidate tree:
  acc10649cb080273a7af86307383af433cf342d4

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only

replay runner:
  mode = 100755
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

The candidate remains a qualification composition target only. It is not integration authority.

The current GitHub billing documentation was then checked against the observed hosted-runner symptom.

GitHub currently documents that:

```text
private repositories:
  standard GitHub-hosted runner usage consumes the owner's included Actions minutes
  usage beyond included amounts is billed when paid usage is available

hard budget:
  if an applicable budget has "Stop usage when budget limit is reached" enabled
  and that budget is exhausted, additional metered usage is blocked

Actions product budget example:
  exhausted hard Actions budget blocks all GitHub-hosted runners
  until the next billing cycle or until the budget is increased
```

Official references checked on 2026-09-22:

- GitHub Docs — "Setting up budgets to control spending on metered products"
  https://docs.github.com/en/billing/how-tos/set-up-budgets
- GitHub Docs — "GitHub Actions billing"
  https://docs.github.com/en/billing/concepts/product-billing/github-actions

This strengthens one narrow statement: an exhausted applicable hard Actions budget is a documented mechanism that can block hosted-runner usage.

It does **not** identify the cause of the current symptom, because the public documentation does not specify that a blocked job must surface as:

```text
runner_id = 0
steps     = []
job log   = BlobNotFound
```

and it does not document the propagation/atomicity semantics needed to explain the already-observed same-head SEI burst in which one `ubuntu-latest` workflow acquired a runner while several other workflows on the exact same head failed before steps.

Therefore the current root-cause boundary is:

```text
hard Actions budget can block hosted runners:
  documented

this account/repository actually has an exhausted blocking budget:
  not observable through the current connector

runner_id=0 / steps=[] uniquely means budget hard stop:
  not established

same-head mixed allocation disproves every budget explanation:
  not established

budget-state propagation / admission race:
  compatible
  not proven

repository/workflow policy or another external admission mechanism:
  compatible
  not proven

specific root cause:
  unknown
```

A final transport-path check also confirmed that the GitHub connector can issue a short-lived signed private-file download URL, but the execution container still cannot resolve the GitHub raw-content host. That path therefore does not produce a mountable checkout and does not change Evidence height.

Updated B4 Evidence height:

```text
all 19 remote refs at pre-write checkpoint        = identical
latest-main canonical 11 composition              = 374e331d... (11 paths only)
canonical replay-runner mode                      = 100755
current P1-P5 focused dependency closures         = covered by exact executions
Linux/macOS/Windows hosted allocation probes      = pre-step failure
ubuntu-slim / service-container probes            = pre-step failure
same-head mixed hosted allocation                 = directly observed
hard Actions budget blocking capability           = documented by GitHub
actual account/repository budget state            = inaccessible
private raw-file transport to execution container = blocked by container DNS
tools/repository-health-check.sh execution        = pending mandatory gate
canonical 11-path source integration              = blocked
actual replay / repeatability Evidence            = pending
```

No additional mechanical runner probe is justified from this checkpoint. The next Evidence-raising event must be either:

1. direct inspection/correction of the account or repository Actions billing/budget/policy state followed by one bounded recovery run, or
2. availability of an execution-capable full checkout through another trusted path.

The mandatory source-integration gate remains an actual execution of `tools/repository-health-check.sh` on execution-time latest main plus these same mode-aware canonical 11 paths.


#### B4 continuation — post-#869 latest-main mode-aware packet refresh — 2026-09-22

PR #869 advanced `main` with documentation only. The canonical source authority is unchanged, but the qualification composition must remain based on execution-time latest main.

Fresh branchless composition:

```text
parent:
  f73b9957eebe971cc8aed9028e10ecb914b17eec

parent tree:
  383e6f0c21d7cd56592ba20413507ece487aaa91

branchless qualification candidate:
  d3703503f8be0893098e22649aa9e43717bcb471

candidate tree:
  a149388c414a07b97b6166470f52d490f1bb4e2d

candidate vs parent:
  ahead  = 1
  behind = 0
  changed paths = canonical 11 source paths only

replay runner:
  mode = 100755
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

The candidate is still a branchless composition target only. It is not integration authority and must not be merged or cherry-picked around Repository health.

No further mechanical hosted-runner probe is added at this checkpoint. Existing probes already establish the current pre-step admission symptom across the relevant hosted-runner shapes, while the specific external cause remains unobserved through the available connector surface.

Mandatory next gate remains unchanged:

```text
execution-capable full checkout on latest main
  -> materialize the same tree-aware canonical 11 paths
  -> verify all canonical blob identities
  -> verify replay runner mode = 100755
  -> tools/repository-health-check.sh
  -> same-revision / clean-base / canonical-overlap recheck
  -> qualified source integration
```


#### B4 continuation — same-owner private-repository billing refusal evidence — 2026-09-22

The post-#869 mode-aware packet refresh already merged via PR #870 remains the current qualification composition basis:

```text
main:
  6042da1b6ca2cd8777b60b557787e6327670a69b

latest current-main mode-aware branchless packet:
  6112563fc4c0d10cfebf1b5a4011844dfba6a744

candidate tree:
  67b9043e033260b0fa1eac2c554c229fc64ed3ff

changed paths:
  canonical 11 source paths only

replay runner:
  mode = 100755
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

A cross-repository inspection found materially stronger infrastructure Evidence under the same personal GitHub owner.

In private repository `hat47x/vs-bear-simulator`, Issue #21 comment `5751891478` preserves an operational observation and explicitly quotes GitHub's job-start refusal:

```text
The job was not started because recent account payments have failed
or your spending limit needs to be increased.
```

The same record identifies `Billing & plans` as the account-side boundary and states that no workflow step executed.

The referenced repository commit is:

```text
hat47x/vs-bear-simulator
488d6a264c8399dcc0a67f23243d35060ad5e54b
```

GitHub check metadata for that exact commit independently matches the current pre-step symptom class:

```text
run 35530180932
  static-validation job 106129271553
  started      = 2026-09-20T18:47:12Z
  completed    = 2026-09-20T18:47:15Z
  conclusion   = failure
  steps        = null
  annotations  = 2

run 35530177597
  static-validation job 106129261731
  started      = 2026-09-20T18:47:08Z
  completed    = 2026-09-20T18:47:11Z
  conclusion   = failure
  steps        = null
  annotations  = 2
```

This is stronger than public community analogy because the provider-message Evidence comes from another private repository under the same owner and its pre-step metadata was rechecked directly.

The provenance boundary is kept explicit:

```text
vs-bear provider refusal message:
  directly preserved in that repository's operational issue comment

vs-bear pre-step GitHub metadata:
  directly rechecked through GitHub

TEI check annotation body:
  still inaccessible through the current connector

TEI exact rejection string:
  not directly observed
```

The cause classification is therefore tightened without promoting an inference to fact:

```text
same-owner private repository affected by GitHub Billing & plans
startup refusal:
  directly observed

TEI showing the same pre-step symptom class because of the same
owner-side billing/spending-limit condition:
  strongest current hypothesis

TEI-specific proof of that exact provider message:
  still missing

specific Billing & plans condition:
  payment failure vs account Actions budget vs repository-scoped budget
  remains unresolved
```

Issue #513 comment `5775335128` carries the same cross-repository Evidence.

No additional mechanical hosted-runner retry is justified from this finding. The next high-value action is direct inspection/correction of GitHub `Billing & plans` / `Billing & licensing` payment and Actions budget state. After an actual state change, perform one bounded current-revision recovery run.

The R7B integration boundary remains unchanged:

```text
tools/repository-health-check.sh actual execution = pending
canonical 11-path source integration             = blocked
actual replay / repeatability Evidence           = pending
```


#### B4 continuation — local-execution qualification policy — 2026-09-22

External compute resources are paused until explicitly re-enabled. No additional hosted-runner, Runpod, Replit, Codespaces, or other remote-compute probe is part of the active R7B qualification path.

The active execution model is now:

```text
trusted local full checkout
  -> existing branch research/selective-knowledge-routing-r6
  -> execution-time latest origin/main as base
  -> same mode-aware canonical 11-path packet
  -> exact identity / mode preflight
  -> tools/repository-health-check.sh
  -> unchanged revision / clean working tree
  -> remote-main overlap recheck
  -> only then merge the existing B branch to main
  -> converge all existing refs with force=false
```

No new qualification branch is authorized. The existing B branch is the single source-candidate lane.

Local preflight after the canonical packet is staged on that branch:

```bash
git fetch origin --prune
git switch research/selective-knowledge-routing-r6
git pull --ff-only origin research/selective-knowledge-routing-r6
git fetch origin main

test -z "$(git status --porcelain)"

BASE_MAIN="$(git rev-parse origin/main)"
HEAD_BEFORE="$(git rev-parse HEAD)"

git merge-base --is-ancestor "${BASE_MAIN}" "${HEAD_BEFORE}"
```

The branch diff against `origin/main` must contain exactly these 11 paths:

```text
tools/internal/firstwebtargetevidence/semantics.go
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
tools/internal/firstwebtargetevidence/semantics_test.go
tools/reference-first-web-target-evidence-audit/main.go
tools/reference-first-web-target-r7b-repeatability/main.go
tools/reference-first-web-target-r7b-repeatability/main_test.go
tools/reference-first-web-target-r7b-replay-audit/main.go
tools/reference-first-web-target-r7b-replay-audit/main_test.go
tools/reference-first-web-target-r7b-replay-plan/main.go
tools/reference-first-web-target-r7b-replay-plan/main_test.go
tools/run-first-web-target-r7b-replay.sh
```

Canonical byte identities:

```text
semantics.go
  9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
semantics_pressure_test.go
  95553ba10756399a50e52e639f7d860f4420f29f
semantics_test.go
  ba17f69a7e375cfdeec0db544986d51f99b26192
evidence-audit/main.go
  3878dae15b06a3b57954cf196b108da1cc03bbc7
repeatability/main.go
  7b2cf09460319b13241fa44e59f865d3864734ff
repeatability/main_test.go
  30a907578836c9b1b33b79bf4d521fb809a49e74
replay-audit/main.go
  a370e0c5efc4399e324be7a0ad4801626f05a5da
replay-audit/main_test.go
  2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
replay-plan/main.go
  40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
replay-plan/main_test.go
  d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
run-first-web-target-r7b-replay.sh
  2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
```

The replay runner must additionally retain:

```text
Git tree mode = 100755
```

Required local checks before Repository health:

```bash
test "$(git hash-object tools/run-first-web-target-r7b-replay.sh)" =   "2f55ec06f29bbca8f4a9e6ce04816bd311a8af03"

test "$(git ls-files -s tools/run-first-web-target-r7b-replay.sh | awk '{print $1}')" =   "100755"

bash -n tools/run-first-web-target-r7b-replay.sh
```

Then execute the mandatory gate:

```bash
bash tools/repository-health-check.sh
```

A PASS is qualified only if the checkout remains on the same commit and clean:

```bash
test "$(git rev-parse HEAD)" = "${HEAD_BEFORE}"
test -z "$(git status --porcelain)"
```

After a local PASS, fetch `origin/main` again. If main moved, inspect every intervening changed path against the canonical 11 before integration. Any canonical-path overlap requires recomposition and requalification; no-overlap documentation or unrelated-source movement does not authorize skipping that check.

Until this local gate is executed successfully:

```text
canonical source integration       = blocked
actual replay / repeatability      = pending
remote-compute fallback            = paused by current operating policy
```

#### B4 continuation — operator-confirmed monthly Actions exhaustion and hosted-runner pause — 2026-09-23

The operator has now provided the account-side fact that was not observable through the GitHub connector during the earlier runner investigation:

```text
GitHub Actions monthly usage:
  exceeded

desired operating state:
  stop GitHub Actions usage
```

This supersedes the earlier **current-cause uncertainty** for this operating period. The historical observations remain useful as provenance, but they must no longer be read as saying that the present blockage is still unexplained.

Current cause / policy boundary:

```text
current monthly-usage exhaustion:
  operator-confirmed

GitHub-hosted Actions reruns / recovery probes:
  stopped

new hosted-runner diagnostic workflows:
  stopped

automatic hosted-runner consumption:
  stopped

active qualification path:
  trusted local full checkout only

resume condition:
  monthly usage/budget state is reset
  AND GitHub Actions use is explicitly resumed
```

Repository-side protection is already in place:

```text
remaining workflow:
  .github/workflows/knowledge-portability-reference-evidence.yml

trigger:
  workflow_dispatch only

job guard:
  if: ${{ false }}

pause integration:
  PR #874
```

Therefore no further hosted-runner Evidence is required or desired while this pause is active. Issue #513 remains useful as the infrastructure/history record, but R7B qualification must not wait for another Actions probe.

Current B4 source state at this checkpoint:

```text
main:
  288db99d52ea2736b4750f2c5c6a951f0d91c886

qualification branch:
  research/selective-knowledge-routing-r6
  d9bcfb73591f19ebf4d7eb38e64336a4c5b74fc6

branch vs main:
  ahead  = 3
  behind = 0
  net changed paths = canonical 11 source paths only

Open PR:
  0
```

The canonical 11 blob identities remain unchanged, and the replay runner remains:

```text
tools/run-first-web-target-r7b-replay.sh
  blob = 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03
  mode = 100755
```

The local qualification entrypoint is:

```text
tools/run-first-web-target-r7b-local-qualification.sh
  blob = 2d68e20a08614c3ce5b9f4a976804a1b8d97f9d8
  mode = 100755

optional for diagnostic qualification runs:
  TEI_R7B_QUALIFICATION_RECEIPT=<repository-external path>

required for any qualification run intended to authorize canonical 11-path integration:
  TEI_R7B_QUALIFICATION_RECEIPT must be set
  the resulting PASS receipt is the revision-specific integration authority
  parent directory must already exist
  relative / .. forms are canonicalized before the repository-external check
  existing files and symlinks are rejected before qualification
  a receipt path that appears during qualification is rejected
  noclobber prevents replacement during receipt creation
  same-directory temporary file + hard-link publication provides an atomic publish boundary
  ln is a required preflight command for the atomic publish primitive
  partial receipt output is removed if receipt creation fails
  invalid receipt output is removed if the post-receipt exact-clean check fails
  HEAD / worktree cleanliness is rechecked after receipt publication
```

It verifies branch/base ancestry, exact canonical path set, all canonical blob identities and modes, the required Go version, replay-runner shell syntax, full `tools/repository-health-check.sh`, unchanged HEAD/clean worktree, and post-run remote-main movement/overlap before reporting PASS.

The remaining integration boundary is therefore deliberately narrow:

```text
trusted local full checkout
  -> research/selective-knowledge-routing-r6
  -> bash tools/run-first-web-target-r7b-local-qualification.sh
  -> PASS
  -> re-fetch / overlap check
  -> merge the existing B4 lane to main
  -> converge existing refs with force=false

until PASS:
  canonical 11-path source integration = blocked
  actual replay / repeatability        = pending
```

No new qualification branch, remote-compute fallback, or Actions-based probe is authorized by this checkpoint.


#### B4 continuation — cross-chat qualification convergence checkpoint — 2026-09-23

The current GitHub state, accessible ChatGPT conversation history, and retained Library validation material were rechecked together before further R7B work.

No source provides an actual PASS receipt from a trusted local full checkout for the current canonical 11-path candidate. The retained Library files record isolated gofmt / go vet / go test or draft validation only and explicitly do not claim a full `tools/repository-health-check.sh` PASS on the private repository checkout.

Therefore these observations must not be promoted into integration authority:

```text
static/source-shape audit                    != local source qualification
isolated focused Go checks                  != repository health PASS
historical draft validation                 != current candidate qualification
absence of canonical-path conflict          != permission to merge code
```

The only Evidence-raising next step remains:

```text
trusted local full checkout
  -> current research/selective-knowledge-routing-r6 tip
  -> optional Repository-external qualification receipt path
  -> tools/run-first-web-target-r7b-local-qualification.sh
  -> repository-health-check PASS on the exact candidate revision
  -> unchanged HEAD / clean worktree / no canonical overlap
  -> only then integrate the existing R7B lane into main
```

No new qualification branch, Actions retry, external-compute fallback, or synthetic substitute should be introduced while this boundary remains unresolved.


#### B4 continuation — qualification checkout exact-clean hardening — 2026-09-23

The previously identified qualification-checkout index-state gap has now been absorbed into the local qualification helper by PR #881.

Current helper:

```text
tools/run-first-web-target-r7b-local-qualification.sh
  blob = 2d68e20a08614c3ce5b9f4a976804a1b8d97f9d8
  mode = 100755
```

It now fail-closes on all of the following before Repository health:

```text
tracked / untracked / ignored worktree residue
non-standard tracked index state (including skip-worktree / assume-unchanged forms)
wrong qualification branch
local tip != remote qualification tip
current remote main not ancestor of qualification HEAD
wrong Go version
canonical 11 path-set drift
canonical blob or mode drift
```

The same exact-clean conditions are rechecked after Repository health and again after optional qualification receipt creation. PR #882 additionally guarantees that failed receipt creation or a failed post-receipt exact-clean check cannot leave a PASS-looking receipt behind; partial or invalid receipt output is removed before failure is returned. Therefore the separate manual index-state preflight introduced in the immediately preceding checkpoint is superseded by helper blob `2d68e20a08614c3ce5b9f4a976804a1b8d97f9d8` and must not be treated as an additional operator step.

The integration authority remains unchanged:

```text
trusted local full checkout
  -> current research/selective-knowledge-routing-r6 tip
  -> tools/run-first-web-target-r7b-local-qualification.sh
  -> PASS on exact candidate revision
  -> latest-main / canonical-overlap recheck
  -> only then integrate the existing R7B lane into main
```

No full-checkout PASS receipt has yet been observed, so canonical source integration remains blocked.

#### B4 continuation — post-PASS qualification-tip integration fence — 2026-09-24

The local qualification helper validates that the local candidate HEAD equals the remote qualification-branch tip **before** Repository health begins and re-checks the same remote qualification tip immediately after Repository health. It also re-fetches `main` and fail-closes on rewrite / canonical-path overlap.

That closes qualification-branch movement during the helper run. A separate integration-time fence is still required because the branch can legitimately move **after** the helper exits and before the canonical source merge is executed.

Therefore a PASS receipt authorizes only the exact revision named by:

```text
qualificationRepositoryRevision
```

and does **not** authorize "whatever the qualification branch points to later".

Immediately before source integration, the operator must perform a fresh remote fetch and bind the merge target to the receipt revision:

```bash
git fetch origin --prune

receipt_revision="<qualificationRepositoryRevision from PASS receipt>"
remote_qualification_revision="$(
  git rev-parse origin/research/selective-knowledge-routing-r6
)"

test "$remote_qualification_revision" = "$receipt_revision" || {
  printf 'Qualification branch moved after qualification: receipt=%s remote=%s\n' \
    "$receipt_revision" "$remote_qualification_revision" >&2
  exit 1
}
```

The same integration attempt must then re-read `origin/main` and confirm that any movement since the receipt's `baseMainAfter` is a forward-only change with **zero canonical-11 overlap**. If either the qualification branch or a canonical source path moved, the old receipt is stale and the current branch tip must be qualified again.

When integration is performed through a pull request or API merge, the merge operation must also pin the expected PR head SHA to the same receipt revision. A concurrent B-lane update must therefore cause the merge to fail rather than silently integrating an unqualified tip.

Final authority is consequently:

```text
PASS receipt revision
  == current remote qualification tip
  == expected merge head SHA

AND

current main movement since baseMainAfter
  = forward-only
  + zero canonical-11 overlap
```

This fence is an integration-time concurrency guard. It does not replace or weaken any checks inside `tools/run-first-web-target-r7b-local-qualification.sh`, and it does not authorize canonical source integration until a real trusted-local PASS receipt exists.



#### B4 continuation — live qualification-branch lineage preservation — 2026-09-24

The historical notes above include branchless candidate recomposition and rebasing language from phases where candidate commits were deliberately unreferenced. Those records do **not** authorize rewriting the live qualification branch.

For the current single source-candidate lane:

```text
research/selective-knowledge-routing-r6
```

branch identity and ancestry are part of the coordination contract.

Required branch discipline:

```text
previous remote B4 tip
  must remain an ancestor of
new remote B4 tip

latest main movement
  -> merge into the existing B4 tip
  -> preserve B4 ancestry

B4 ref update
  -> fast-forward only
  -> force=false

rebase / reset / delete+recreate / force-update of the live B4 ref
  -> prohibited
```

If latest main does not overlap the canonical 11 paths, carry it into B4 with a normal merge commit while preserving the current B4 tip as first parent. If a canonical path overlaps, stop, reconcile the source on top of the current B4 tip, and re-run qualification after the resulting forward commit/merge.

If another ChatGPT lane advances B4 between read and update, the stale candidate must be abandoned and recomposed from the newly observed B4 tip. A non-fast-forward rejection is a coordination success condition; it must never be bypassed with `force=true`.

Historical branchless candidate SHAs may still be used as content references for exact blobs, but never as replacement ancestry for the live B4 ref.

The current integration fence remains stricter still: a trusted-local PASS receipt authorizes only its exact `qualificationRepositoryRevision`, and integration must require that exact revision to remain the current remote B4 tip and expected merge head.

#### B4 continuation — qualification helper closes in-run B-tip movement — 2026-09-24

The local qualification helper now re-checks the remote qualification-branch tip immediately after Repository health and the post-health exact-clean check.

Current helper:

```text
tools/run-first-web-target-r7b-local-qualification.sh
  blob = 2d68e20a08614c3ce5b9f4a976804a1b8d97f9d8
  mode = 100755

independent exact-byte check:
  git hash-object = 2d68e20a08614c3ce5b9f4a976804a1b8d97f9d8

independent shell syntax:
  bash -n = PASS
```

The added fail-closed boundary is:

```bash
git fetch "$remote" --prune
remote_head_after="$(git rev-parse "$remote/$qualification_branch")"

test "$remote_head_after" = "$head_before" || {
  printf 'Qualification branch moved during local R7B qualification: before=%s after=%s\n' \
    "$head_before" "$remote_head_after" >&2
  exit 1
}
```

This closes the interval from qualification start through completion of Repository health: if another ChatGPT lane advances the live B4 branch during the health run, no PASS receipt is emitted for the now-stale revision.

The integration-time fence remains mandatory because the branch can still move **after the helper exits** and before the source merge is executed:

```text
during helper:
  remote B tip after health == qualificationRepositoryRevision

after helper / before merge:
  fresh remote B tip == qualificationRepositoryRevision
  expected merge head SHA == qualificationRepositoryRevision

main:
  movement since baseMainAfter remains forward-only
  canonical-11 overlap remains zero
```

Therefore qualification authority is revision-specific across both concurrency windows. No receipt for an older B4 tip may authorize a newer tip, even when the canonical file bytes happen to be unchanged.

No full-checkout PASS receipt has yet been observed, so canonical source integration remains blocked.



#### B4 continuation — integration-authorizing receipt requirement — 2026-09-24

The helper may still run without `TEI_R7B_QUALIFICATION_RECEIPT` for diagnostic purposes, but stdout-only PASS is not sufficient authority to integrate the canonical 11-path candidate.

Any qualification run intended to authorize source integration must set a Repository-external receipt path and successfully publish the receipt. Integration authority is then bound to the exact receipt fields:

```text
qualificationRepositoryRevision
baseMainAfter
repositoryHealthPassed = true
repositoryPostRunClean = true
```

Immediately before integration:

```text
current remote B tip == qualificationRepositoryRevision
expected merge head   == qualificationRepositoryRevision
current main movement since baseMainAfter = forward-only
canonical-11 overlap since baseMainAfter   = zero
```

A helper run without a receipt may prove local diagnostics but must never be used to merge the canonical source candidate.

#### B4 continuation — exact-main qualification binding — 2026-09-24

The previous integration fence allowed a qualification run to remain authoritative when `main` moved forward during or after Repository health, provided the intervening paths did not overlap the canonical 11. That is insufficient for a full-repository health gate.

`tools/repository-health-check.sh` executes repository-wide checks:

```text
gofmt -l .
go vet ./...
go test -count=1 ./...
go run ./cmd/tei validate -workspace reference/workspace
```

A change outside the canonical 11 can therefore alter compile, test, validation, or cross-package behavior even when no canonical file path overlaps. A PASS must authorize the exact tree that is about to be integrated, not only the canonical path subset.

This checkpoint supersedes the earlier forward-only / zero-overlap allowance.

Current qualification helper:

```text
tools/run-first-web-target-r7b-local-qualification.sh
  blob = 4fb4d84147669ad7d7700999b370ba3a2f72f7b5
  mode = 100755
```

After Repository health, the helper now requires both remote refs to remain exact:

```text
remote qualification tip after health
  == qualificationRepositoryRevision

remote main after health
  == baseMainBefore
  == baseMainAfter
```

Any `main` movement during qualification is fail-closed, even when the changed paths are documentation-only or have zero canonical-11 overlap. The recovery path is:

```text
latest main moves
  -> merge latest main into the existing B4 tip
  -> preserve B4 lineage / force=false
  -> rerun the full local qualification helper
  -> publish a new PASS receipt for the new exact tree
```

The integration-time fence is correspondingly exact. Immediately before integration:

```text
current remote B tip == receipt.qualificationRepositoryRevision
current main         == receipt.baseMainAfter
expected merge head  == receipt.qualificationRepositoryRevision
```

If either ref moved after the helper exited, the receipt is stale. Do not rely on path-overlap analysis to preserve authority; merge the new `main` into B4 and rerun full qualification.

This makes the PASS receipt a two-revision lock:

```text
(candidate B4 revision, exact main revision)
```

and ensures that the repository-wide health result applies to the exact integration tree.

#### B4 continuation — qualified candidate tree binding — 2026-09-24

The integration-authorizing PASS receipt now records the exact Git tree that Repository health qualified:

```text
qualificationRepositoryRevision = candidate commit
qualificationRepositoryTree     = candidate commit^{tree}
baseMainAfter                   = exact main revision
```

Current helper:

```text
tools/run-first-web-target-r7b-local-qualification.sh
  blob = 4fb4d84147669ad7d7700999b370ba3a2f72f7b5
  mode = 100755
```

The helper derives the tree directly from the already-bound candidate HEAD:

```bash
head_before="$(git rev-parse HEAD)"
tree_before="$(git rev-parse HEAD^{tree})"
```

and publishes both values in the Repository-external PASS receipt.

This closes a separate integration Evidence gap. The source merge normally creates a new merge commit on `main`; its commit SHA is therefore expected to differ from the qualified B4 revision even though the resulting repository tree must be identical to the health-qualified candidate tree.

For the canonical source integration, the merge method must preserve the qualified candidate tree. Immediately after the merge, verify:

```text
merged main tree
  == receipt.qualificationRepositoryTree
```

while the pre-merge fence still requires:

```text
current remote B tip == receipt.qualificationRepositoryRevision
current main         == receipt.baseMainAfter
expected merge head  == receipt.qualificationRepositoryRevision
```

If the merge result tree differs from `qualificationRepositoryTree`, the integration result is not the tree that passed Repository health. Do not proceed to Packet M / actual replay Evidence; stop and reconcile.

This post-merge tree check complements, rather than replaces, the exact-main and exact-B revision locks. Together they bind qualification authority across:

```text
exact base main
  + exact candidate commit
  + exact candidate tree
  + exact merge head
  + exact merged tree
```

No full-checkout PASS receipt has yet been observed, so the canonical 11-path candidate remains unintegrated.


#### B4 continuation — machine-checked integration receipt preflight — 2026-09-24

The exact B revision / candidate tree / exact main requirements are now backed by a machine-checkable integration preflight rather than operator transcription alone.

Added tools:

```text
tools/reference-first-web-target-r7b-qualification-receipt-audit
tools/verify-first-web-target-r7b-integration-receipt.sh
```

The Go audit rejects unknown receipt fields and trailing JSON, then requires the receipt to match the current remote identities exactly:

```text
receipt.qualificationRepositoryRevision == current remote B4 tip
receipt.qualificationRepositoryTree     == current remote B4 tip^{tree}
receipt.baseMainBefore                  == current remote main
receipt.baseMainAfter                   == current remote main
receipt.mainMovedDuringQualification    == false
receipt.repositoryHealthPassed          == true
receipt.repositoryPostRunClean          == true
receipt canonical path count / Go version / replay runner identity = canonical values
```

The shell preflight itself must run from an exact-clean local checkout whose HEAD equals current remote `main`; it rejects ignored residue and non-standard tracked index state before executing the audit. This prevents a locally modified verifier from authorizing integration.

Usage after a trusted-local PASS receipt exists:

```bash
git switch main
git pull --ff-only origin main

bash tools/verify-first-web-target-r7b-integration-receipt.sh \
  /tmp/tei-r7b-evidence/local-qualification.json
```

A PASS from this verifier still does not itself perform the merge. The actual API/PR merge must pin the expected head SHA to the verified qualification revision, and the merged main tree must then equal the receipt `qualificationRepositoryTree` before Packet M work proceeds.

#### B4 continuation — integration receipt verifier local package qualification — 2026-09-24

The integration-receipt verifier added around PR #890/#891 was independently reconstructed from the exact GitHub blobs and exercised outside a full repository checkout.

Verified blobs:

```text
tools/reference-first-web-target-r7b-qualification-receipt-audit/main.go
  c702b7b4f768445e5b2706c6d5deeb3471018a0c

tools/reference-first-web-target-r7b-qualification-receipt-audit/main_test.go
  ce8d2a8da6eeaac36cebd4893275bbd73594e018

tools/verify-first-web-target-r7b-integration-receipt.sh
  004d7be0f84824d2cadbb01a767bf30a75ed8ab4
```

The reconstruction was normalized with `gofmt` and then checked by `git hash-object`; the resulting hashes matched the GitHub blobs exactly.

Environment:

```text
go version go1.23.2 linux/amd64
```

Focused verifier qualification:

```text
GO111MODULE=off go test -count=1
  PASS

GO111MODULE=off go vet
  PASS

bash -n tools/verify-first-web-target-r7b-integration-receipt.sh
  PASS
```

The decoder test was also corrected so the unknown-field case is valid JSON and therefore actually exercises `DisallowUnknownFields`, rather than passing because the JSON itself is malformed. The test first round-trips a valid receipt through `json.Marshal` / `decodeReceipt` before exercising unknown-field and trailing-JSON rejection.

The shell preflight now repeats exact-clean checks and re-fetches both remote identities after the Go audit:

```text
post local HEAD/status/index == preflight start state
post-fetch origin/main       == preflight main revision
post-fetch remote B tip      == preflight candidate revision
post-fetch candidate tree    == preflight candidate tree
```

Any movement during receipt verification is fail-closed.

This focused PASS qualifies only the newly added receipt-audit package and integration-preflight shell syntax/logic boundary. It is **not** a substitute for the mandatory trusted-local full-repository qualification of the canonical 11-path candidate.

No integration-authorizing full-checkout PASS receipt has yet been observed.


#### B4 continuation — exact-parent source integration protocol — 2026-09-24

The machine-checked receipt preflight fixes the candidate commit, candidate tree, and exact main revision, but an ordinary pull-request merge does not compare-and-swap the base branch SHA. Pinning only the PR head is therefore insufficient for the final canonical source integration.

For this integration boundary, construct an explicit Git merge commit from the verified receipt:

```text
first parent  = receipt.baseMainAfter
second parent = receipt.qualificationRepositoryRevision
tree          = receipt.qualificationRepositoryTree
```

Before publishing it, audit the created merge object itself:

```text
merge.parents[0] == receipt.baseMainAfter
merge.parents[1] == receipt.qualificationRepositoryRevision
merge.tree       == receipt.qualificationRepositoryTree
```

Then update `main` to that merge commit with `force=false` only.

This acts as the base-side compare-and-swap:

```text
if main is still receipt.baseMainAfter
  -> the ref update is a valid fast-forward

if main moved after receipt verification
  -> the explicit merge commit does not contain the new main tip
  -> force=false ref update is rejected
  -> merge latest main into the existing B4 lane
  -> rerun trusted-local qualification
  -> publish a new receipt
```

Do not use a normal PR merge for the final canonical 11 source integration unless the mechanism independently pins the exact base main revision. `expected_head_sha` protects the candidate side only.

After a successful main ref update, verify:

```text
main^{tree} == receipt.qualificationRepositoryTree
```

before Packet M or replay Evidence proceeds.

If B4 has not moved since verification, converge `research/selective-knowledge-routing-r6` to the new main tip with a normal fast-forward (`force=false`). If another lane advanced B4 concurrently, leave that newer tip intact: the qualified revision was integrated exactly, while the newer B4 work remains unqualified and must follow the normal qualification cycle.
#### B4 continuation — machine-checked post-merge qualified-tree verification — 2026-09-24

The exact-parent integration protocol now has a dedicated post-merge verifier:

```text
tools/verify-first-web-target-r7b-post-merge.sh
  mode = 100755
  blob = 4ba5f8f76c84c551980387de51a6884d73669903
```

The verifier runs from an exact-clean checkout at current remote `main`, reads the Repository-external qualification receipt, and requires current `main` to be an exact two-parent merge object.

It derives:

```text
first parent  = merge.parents[0]
second parent = merge.parents[1]
merge tree    = main^{tree}
```

and reuses the strict qualification-receipt audit with:

```text
expected main      = first parent
expected candidate = second parent
expected tree      = merge tree
```

Therefore a post-merge PASS proves that the merge object itself satisfies the receipt contract:

```text
merge.parents[0] == receipt.baseMainBefore
                 == receipt.baseMainAfter

merge.parents[1] == receipt.qualificationRepositoryRevision

merge.tree       == receipt.qualificationRepositoryTree

receipt.repositoryHealthPassed = true
receipt.repositoryPostRunClean = true
```

The verifier also repeats exact-clean local-state checks and re-fetches remote `main` after the receipt audit. If `main` moves during post-merge verification, the check fails closed.

Independent validation:

```text
git hash-object reconstructed script
  = 4ba5f8f76c84c551980387de51a6884d73669903

bash -n
  PASS
```

A synthetic Git repository was then used to exercise the object-level contract.

Positive case:

```text
base commit
  -> qualification candidate commit
  -> explicit integration merge commit

merge first parent  = base
merge second parent = qualification candidate
merge tree          = candidate tree

matching PASS receipt
  -> receipt audit PASS
  -> post-merge verifier PASS
```

Negative case:

```text
after the valid integration merge,
advance main with an ordinary one-parent commit

post-merge verifier
  -> exit 1
  -> "Current main must be an exact two-parent R7B integration merge commit."
```

This synthetic validation proves the verifier's merge-object boundary; it does not qualify the real canonical 11-path candidate.

The actual integration sequence remains:

```text
trusted-local full qualification PASS receipt
  -> integration receipt preflight PASS
  -> explicit exact-parent merge object
  -> force=false main ref update
  -> post-merge verifier PASS
  -> only then Packet M / actual replay
```

No integration-authorizing full-checkout PASS receipt has yet been observed.
#### B4 continuation — exact-parent merge object preparation helper — 2026-09-24

The final canonical source integration now has a preparatory helper that constructs the exact merge object without moving any ref:

```text
tools/prepare-first-web-target-r7b-exact-parent-merge.sh
  mode = 100755
  blob = 13118b68520afcabe56227233d289e8628c1b51f
```

The helper first runs `tools/verify-first-web-target-r7b-integration-receipt.sh` against the Repository-external PASS receipt. It then re-fetches remote state and derives:

```text
main revision      = current remote main
candidate revision = current remote B4 tip
candidate tree     = candidate^{tree}
```

It creates an **unreferenced** Git merge object with `git commit-tree`:

```text
tree          = candidate tree
first parent  = exact main revision
second parent = exact B4 qualification revision
```

and immediately audits the created object before printing its SHA. The helper does not update local `main`, remote `main`, or the B4 branch.

After object creation it repeats exact-clean local-state checks and re-fetches `main`, B4, and the candidate tree. Any in-run movement fails closed.

Independent exact-byte and syntax checks:

```text
git hash-object reconstructed helper
  = 13118b68520afcabe56227233d289e8628c1b51f

bash -n
  PASS
```

A synthetic bare-origin repository was then used to exercise the object-preparation boundary.

Positive case:

```text
local checkout = current origin/main
origin/B4      = descendant candidate
receipt preflight boundary = stubbed PASS for this isolated merge-object test

result:
  exact two-parent merge object created
  object tree == candidate tree
  parent[0]   == origin/main
  parent[1]   == origin/B4
  origin/main unchanged
  origin/B4 unchanged
  PASS
```

Stale-main case:

```text
advance origin/main after the local checkout
rerun merge preparation

result:
  exit 1
  stale local main rejected before merge-object publication
```

The synthetic test intentionally stubs the receipt-preflight command because that verifier has its own focused package/shell qualification recorded above. This test isolates the `commit-tree` object construction and non-mutating/CAS boundary.

The intended final sequence is now:

```text
trusted-local full qualification PASS receipt
  -> machine-checked integration receipt preflight
  -> prepare exact-parent unreferenced merge object
  -> inspect exact parents/tree
  -> publish main with non-force update only
  -> machine-checked post-merge verification
  -> Packet M / actual replay
```

No full-checkout PASS receipt has yet been observed, so the helper must not be used to publish the real canonical source packet yet.
Current-blob revalidation after the dependency preflight was extended to include `tr`:

```text
helper blob
  13118b68520afcabe56227233d289e8628c1b51f

git hash-object reconstructed helper
  exact match

bash -n
  PASS

synthetic exact-parent merge preparation
  PASS

remote refs unchanged after object preparation
  PASS

stale remote main
  rejected with exit 1
```
#### B4 continuation — qualification platform provenance and current-helper supersession — 2026-09-24

The Repository-external PASS receipt now records the Go execution platform in addition to the already pinned Go version:

```text
goVersion = go env GOVERSION
goOS      = go env GOOS
goArch    = go env GOARCH
```

Current local qualification helper:

```text
tools/run-first-web-target-r7b-local-qualification.sh
  blob = babc0f80edeaee67a59c0bf8ddcccfec7083fad9
  mode = 100755
```

Current receipt-audit package blobs:

```text
tools/reference-first-web-target-r7b-qualification-receipt-audit/main.go
  e2046453a64c5aeb9d9cd463a60d8d1beb716cee

tools/reference-first-web-target-r7b-qualification-receipt-audit/main_test.go
  678a5e0c7583c68b026c3e4424c63f29a7b40806
```

The receipt audit requires `goOS` and `goArch` to be non-empty provenance. It does **not** constrain trusted local qualification to one operating system or architecture; `goVersion` remains the pinned execution-version requirement, while GOOS/GOARCH record the environment in which Repository health passed.

Focused exact-blob validation of the receipt-audit package:

```text
gofmt + git hash-object
  main.go      = e2046453a64c5aeb9d9cd463a60d8d1beb716cee
  main_test.go = 678a5e0c7583c68b026c3e4424c63f29a7b40806

GO111MODULE=off go test -count=1
  PASS

GO111MODULE=off go vet
  PASS
```

The producer-side shell delta is limited to reading `GOOS` / `GOARCH`, writing those two JSON fields into the external receipt, and printing the platform in the final diagnostic. The previously qualified exact-clean, branch-tip, main-revision, canonical-path, and atomic-receipt publication gates remain unchanged.

Earlier sections that label older helper blobs as "Current helper" are historical checkpoints. For all new qualification runs after this checkpoint, `babc0f80edeaee67a59c0bf8ddcccfec7083fad9` is the current helper identity.

This support change does not authorize canonical source integration. A real trusted-local full Repository-health PASS receipt for the exact current B4 tip and exact current main revision is still mandatory.
#### B4 continuation — merge-preparation snapshot rebind to trusted receipt — 2026-09-24

The exact-parent merge preparer now closes the interval between the standalone integration preflight and the preparer's own ref snapshot.

Current preparer:

```text
tools/prepare-first-web-target-r7b-exact-parent-merge.sh
  blob = 13118b68520afcabe56227233d289e8628c1b51f
  mode = 100755
```

After the preparer fetches current remote state and derives `main_revision`, `candidate_revision`, and `candidate_tree`, it runs the strict receipt audit **again** against those exact values before `git commit-tree`:

```text
receipt candidate == preparer candidate revision
receipt tree      == preparer candidate tree
receipt main      == preparer exact main
```

This prevents a stale receipt from authorizing a B4 revision that moved after the standalone preflight returned but before merge-object construction began.

Current-blob exact validation:

```text
git hash-object reconstructed preparer
  = 13118b68520afcabe56227233d289e8628c1b51f

bash -n
  PASS
```

A synthetic bare-origin repository then exercised this new race boundary with the real receipt-audit package:

```text
matching receipt + current main/B/tree
  -> receipt audit PASS
  -> exact-parent merge object prepared
  -> parents/tree exact
  -> PASS

move only remote B after receipt creation
  -> preparer fetches new B tip
  -> direct receipt rebind compares old receipt to new B tip
  -> stale receipt rejected
  -> exit 1
```

The moved-B failure was observed at the receipt-audit boundary with the candidate mismatch, before `git commit-tree` could authorize that newer unqualified tip.

The standalone integration preflight remains required. The direct rebind is a second, narrower compare-and-swap-style check over the exact snapshot used by merge preparation; it does not replace trusted-local full Repository-health qualification.

No integration-authorizing full-checkout PASS receipt has yet been observed.

#### B4 continuation — exact-parent publication helper and active-lane cleanup guard — 2026-09-24

The final non-force publication step is isolated behind a dedicated helper so operators do not manually transcribe the prepared merge SHA into a push command.

```text
tools/publish-first-web-target-r7b-exact-parent-merge.sh
```

The helper accepts the Repository-external qualification receipt and the prepared merge revision. It reruns the integration receipt verifier, refetches exact current `main` and B4 identities, rebinds the receipt to those identities, and audits the prepared merge object itself:

```text
merge parent[0] == current exact main == receipt.baseMainAfter
merge parent[1] == current exact B4   == receipt.qualificationRepositoryRevision
merge tree      == current B4 tree    == receipt.qualificationRepositoryTree
```

It repeats exact-clean checks, refetches main/B4/tree immediately before publication, and publishes only with a normal non-force push:

```text
git push <remote> <merge-revision>:refs/heads/main
```

If main or B4 moved, or the prepared object does not match the receipt, publication stops before the push.

Synthetic bare-origin validation of the exact script bytes:

```text
bash -n
  PASS

positive exact-parent publication
  remote main == prepared merge revision
  PASS

stale-main case
  remote main advanced after merge preparation
  publication rejected before push
  exit 1
```

The helper is not integration authority by itself. It can only succeed with an integration-authorizing trusted-local PASS receipt and the exact prepared merge object.

Cross-lane cleanup rule:

```text
R7B cleanup may move an unrelated branch to main only when:
  ahead_by(main, branch) == 0
  AND no open PR uses the branch as head

otherwise:
  preserve the branch as an active/independent lane
  do not move, rewrite, or force-update it
```

Current protected independent lane: PR #895 head research/tei-ld-0a-corpus-v0 at 6ebb85485cdbfeedd097d841efbbb8f9aab75a5a.

After a successful publication, update an exact-clean local main checkout to the published revision and run the existing post-merge verifier with the same receipt before Packet M / replay Evidence proceeds.
