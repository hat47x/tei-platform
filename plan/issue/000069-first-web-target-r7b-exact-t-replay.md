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

Source / Design上の入口は揃っている。actual executionの停止点は、adopted Gate C Release archiveをGo-enabled full checkoutの実行環境で再取得・照合し、R7A handoff、R7B intake、R7B replay planをactual materializeしたうえで、health-check済みのoperational replay runnerをexact Tへ実行していないことである。

このIssueはその停止点から先を所有する。
