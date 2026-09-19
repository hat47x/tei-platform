# ISSUE-000069: First Web TargetのR7B exact-T replayとrepeatability Evidenceを取得する

- 状態: Ready
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

current top-levelはReplay Target `T` を保持する一方、execution intakeに存在する次の値をplan top-levelへ保持していない。

```text
HandoverBaselineID
BaselineRepositoryRevision
TargetAdapter
Host
```

First Web Targetの起動とshared semantic validatorは `B / R / T / TargetAdapter / Host` のexact provenanceを使う。したがってrunnerだけが既知のB/Rをhard-codeすると、replay plan単体では「何をどのprovenanceで再実行するか」をlosslessに表せない。

E1へ入る前に、execution intakeの既存値をreplay planへexact carryする。

```text
execution intake
  HandoverBaselineID
  BaselineRepositoryRevision
  TargetBuildRevision
  TargetAdapter
  Host
        ↓ lossless carry
replay plan
  HandoverBaselineID
  BaselineRepositoryRevision
  ReplayTargetBuildRevision
  TargetAdapter
  Host
```

これは新しいauthorityや意味を追加する変更ではなく、すでに受領済みのexecution provenanceを実行planへ保持するだけである。

実装条件:

- replay-plan builderはintakeの5値をexact carryする
- replay-plan testはcarryされた値のdrift / 欠落を拒否する
- operational runnerはB/R/adapter/hostをcompile-time固定値だけから決めず、planをexpected provenanceとして使う
- replay audit / shared validatorも同じplan provenanceをexpected valueとして使う
- current Referenceで既知のB/R/Tと矛盾するplanはfail closedする
- field追加をcross-revision compatibility、current Authorization、production authorityへ読み替えない

既存3.2の起動引数例にあるB/R/Tはcurrent Referenceの具体例として残してよいが、runner実装上の唯一のauthorityにはしない。

E0は `tools/reference-first-web-target-r7b-replay-plan` の小変更であり、Packet Vのshared classifier packageとはfile境界が分かれる。ただしコード変更なので、full `tools/repository-health-check.sh` PASS前にはmainへ入れない。

##### 3.0.1 E0 exact implementation delta

current sourceに対する初回差分は次に限定する。

`replayPlan` top-levelへ追加:

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

と合わせ、plan launch provenanceは:

```text
B / R / T / TargetAdapter / Host
```

の5要素になる。

`buildReplayPlan` はaccepted `executionIntake` から次をそのまま代入する。

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

同じ4値を各 `replayUnit` へ複製しない。behavior unitはbehavior-local recipe / raw artifact / Target revision境界を保持し、launch provenanceのsingle sourceはplan top-levelとする。

current `validateIntake` が持つknown Gate C exact-value checkはE0で除去しない。これはcurrent adopted Reference inputをfail closedで受ける境界であり、plan carryの代替ではない。

##### 3.0.2 E0 exact test delta

最低限、既存happy-path testで次を追加確認する。

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

さらに `TestBuildReplayPlanRejectsGateCOrExecutionStateDrift` 相当へ少なくとも:

- baseline ID drift
- baseline revision drift
- Target Adapter drift
- Host drift

を独立subtestとして追加し、current known Gate C Referenceから外れたlaunch provenanceが受理されないことを固定する。

JSON encoding後のtop-level keyもtestで確認し、fieldをstructへ追加しただけでserializationから落ちる変更を防ぐ。

禁止事項:

- B/R/Adapter/Hostを`replayUnit`ごとへ重複保持
- runner側だけのcompile-time constantをplan provenanceの代替にする
- E0と同時にexpected Gate C constantを変更
- E0と同時にbehavior set / fixture / HTTP status / Observation classを変更
- E0をcurrent Authorizationやexecution permissionへ読み替える

##### 3.0.3 E0 integration gate

Go-enabled full checkoutでは、変更後に必ず:

```bash
tools/repository-health-check.sh
```

を実行する。

これは実際に:

```text
gofmt -l .
go vet ./...
go test -count=1 ./...
go run ./cmd/tei validate -workspace reference/workspace
```

を要求する。

focused replay-plan testだけのPASSではE0をmainへ入れない。health checkが失敗した場合、E0 codeをpushせず原因を修正してから再実行する。

##### 3.0.1 E0のexact source delta

E0ではschemaを一般化しない。current `replayPlan` top-levelへ、execution intakeですでに検証済みのlaunch provenanceをそのまま追加する。

```go
type replayPlan struct {
    // existing exact source identities ...
    HandoverBaselineID           string `json:"handoverBaselineId"`
    BaselineRepositoryRevision   string `json:"baselineRepositoryRevision"`
    ReplayTargetBuildRevision    string `json:"replayTargetBuildRevision"`
    TargetAdapter                string `json:"targetAdapter"`
    Host                         string `json:"host"`
    // Units / AuthorityLimits ...
}
```

`buildReplayPlan` では値をcompile-time定数から再構成せず、validation済みintakeからexact carryする。

```text
plan.HandoverBaselineID         = intake.HandoverBaselineID
plan.BaselineRepositoryRevision = intake.BaselineRepositoryRevision
plan.ReplayTargetBuildRevision  = intake.TargetBuildRevision
plan.TargetAdapter              = intake.TargetAdapter
plan.Host                       = intake.Host
```

ここでcurrent Gate C既知値との照合を担当するのは既存 `validateIntake` である。定数は「recorded Gate C provenanceからdriftした入力をrejectする」ために使い、plan出力値の別authorityにはしない。

このためE0後の責務は次の二段になる。

```text
recorded Gate C constants
    ↓ validateIntake: drift reject
validated execution intake
    ↓ buildReplayPlan: lossless carry
replay plan B / R / T / adapter / host
```

runner / replay auditはplanを実行時expected provenanceとして読み、さらにcurrent Referenceで許容されたprovenanceと矛盾する場合はfail closedする。runnerがplanを無視してB/R/adapter/hostを自身の固定値だけから生成する形は禁止する。

##### 3.0.2 E0 test pressure

既存testを壊さず、最低限次を追加・拡張する。

1. happy pathで `HandoverBaselineID / BaselineRepositoryRevision / ReplayTargetBuildRevision / TargetAdapter / Host` がintake値とexact一致する。
2. JSON serialization後も5 fieldがexact field nameで存在し、値が失われない。
3. `HandoverBaselineID` の空・driftをrejectする。
4. `BaselineRepositoryRevision` の空・driftをrejectする。
5. `TargetBuildRevision` の空・driftをrejectする。
6. `TargetAdapter` の空・driftをrejectする。
7. `Host` の空・driftをrejectする。
8. candidate側B/R/T provenanceとtop-level intakeの不一致は既存fail-closedを維持する。
9. credential / selector / coordinate / execution authority fieldを追加しない既存negative testを維持する。

「plan生成後にfieldを改ざんしてbuilderがrejectする」という不可能な責務はbuilder testへ置かない。生成済みplanの改ざん・欠落rejectはV2 replay audit / E1 runnerのinput validationでPressureする。

E0完了判定は、focused package testだけでなくfull `tools/repository-health-check.sh` PASSと、そのPASS対象commitがそのまま`main`へ入ったことまでを含む。

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

### operational runner

新規候補path:

- `tools/run-first-web-target-r7b-replay.sh`

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
