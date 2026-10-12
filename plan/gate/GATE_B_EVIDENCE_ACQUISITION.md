# Gate B Evidence Acquisition

## 目的

この文書は、判定段階Bで**どの検証根拠が必要か、それぞれが何を証明し、何を証明しないか**を定めます。

- 具体的な命令・実行順序・安全側に停止する条件： [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)
- 合格・不合格の判定： [`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)
- 判定段階BからCへの引継ぎ： [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md)

実行手順書とは別に、第二の実行手順の正本は作りません。この文書では、検証根拠の種類、証明する範囲、来歴、完了条件を扱います。

上位要求との追跡では、PAS-02を主な対象、PAS-03を副次的な対象として扱います。

## 現行Gate B scope

判定段階Bの合格に必須となる、ソースの版に依存した検証根拠は次の三つです。

1. #310 リポジトリの決定論的な検証根拠
2. #255 生成成果物の再現性に関する検証根拠
3. #308 実際のPostgreSQL提供実装と一時的な試験用環境の検証根拠

三つとも、同じ変更のない候補版`R`を対象に取得します。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
same-run child exact artifacts
       ↓
independent artifact audit
       ↓
Gate B Readiness Audit for R
       ↓ Ready only
Handover Baseline B / revision R
```

実際の外部資産やOracleに関する検証根拠は引き続き重要ですが、第四の必須経路にはしません。確認済みであれば検証済みの外部資産の根拠として引き継ぎ、未完了なら`unresolved`として渡します。

実際に実行して得た検証根拠が揃ったため、判定段階Bは**合格**しています（候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`、判定日2026年9月19日）。判定の詳細は[`GATE_B_READINESS_AUDIT.md`](GATE_B_READINESS_AUDIT.md)を正本とします。

## Evidence natureを相互代替しない

判定段階Bでは、少なくとも次の種類の検証根拠を区別します。

| Evidence nature | 代表例 | 主に証明するもの |
| --- | --- | --- |
| deterministic repository | #310 | Repository内Application / Driver / Evidence intake境界が固定sourceで再現すること |
| generated artifact reproducibility | #255 | 正規入力から保存generated artifactを固定sourceで再現できること |
| real-provider ephemeral fixture | #308 | 実PostgreSQL provider / driver / native SQLとtool-owned fixture runtime |
| actual external asset | #214等 | TEI非所有の既存外部資産の実物理・provenance・read-only・actual read |

次を読み替えません。

- 決定論的な検証や合成データでの成功 → 実際の外部資産での成功
- 生成成果物の一致 → ドライバーのシナリオの成功
- PostgreSQLの一時的な試験用環境での成功 → 本番の既存データベースに関する事実
- PostgreSQLでの成功 → 実際のOracleでの成功
- 受理記録の成功 → 実際の外部データの読取りの成功
- 自動検査の成功 → 製品の検証根拠を取得済み

業務上の結果、実行時エラー、観測、診断結果も互いに読み替えません。

## 共通Evidence原則

### source-dependent Evidenceはclean revisionへ固定する

判定段階Bで必須の検証根拠は、開始時だけでなく、各実行器が要求する終了時まで、同じ変更のないリポジトリの版に固定します。

少なくとも次を追跡します。

- `repositoryRevision`
- `reproducibleRevision`
- 作業ツリーと索引に変更がないこと
- post-run source確認
- 実行中に版が変わらないこと

異なる版で得た成功結果を寄せ集めません。

候補版`R`の検証中にソースや生成成果物の修復が必要になった場合は、その一連の検証を停止します。変更の意味を確認してコミットし、新しい変更のない版`R'`で#310・#255・#308・#387を初めから実行し直します。

mainが更新されても、検証対象の候補版を暗黙のうちに最新mainへ切り替えません。

### EvidenceはCanonical authorityではない

```text
Authoritative Source / Canonical meaning
        ↓
Validation / execution / observation
        ↓
Evidence artifact
```

検証成果物から意味の正本へ暗黙のうちに書き戻しません。外部の検証根拠も、意味の正本へ取り込みません。

### secret / private runtime materialを通常Evidenceへ保存しない

少なくとも次の情報を、通常の検証根拠には保存しません。

- DSN
- 認証情報や秘密情報
- 元の秘密情報ファイルのパス
- 一時的な引継ぎファイルのパス
- 実行時の引数の値
- 実際のデータベース名
- 不要な実データ

## Required Evidence A — deterministic Repository / #310

関連: `ISSUE-000032` / GitHub #310

### 目的

リポジトリを取得し、通常のGo実行器を用いて、アプリケーションのドライバーに関する主要な意味の境界と、外部の検証根拠を取り込む際の準拠確認を再現できることを示します。

正規の実行器は`tools/reference-gate-b-repository-evidence`です。具体的な命令は実行手順書を正本とします。

### 必要なEvidence

一つのソースの版を固定した検証根拠に、少なくとも次をまとめます。

- 正本となる作業領域の検証
- 画面を使わない最終シナリオ
- 外部の検証根拠を取り込む際の境界の準拠確認
- リポジトリのソースを観測する際の準拠確認
- Goの開発・構築ツール一式の識別情報
- 実行前後のリポジトリのソース

画面を使わない最終シナリオでは、少なくとも次をつなぎます。

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation read
```

### source / Workspace identity

判定段階Bの検証根拠を取得するため、リポジトリ全体を専用に取得します。追跡対象・追跡対象外・無視対象のいずれにもローカルの変更状態を許容しません。追跡対象の索引項目は通常の`H`だけを許容し、`assume-unchanged`・`skip-worktree`・部分取得などの非標準の索引状態を成功の根拠に含めません。

作業領域の識別情報は、`reference/workspace`を正式な契約に厳密に対応付けます。

### 完了条件

判定段階Bに採用する#310の検証根拠には、少なくとも次を要求します。

- `evidenceNature = deterministic-repository`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- 作業領域の検証が成功
- 画面を使わない最終シナリオが成功
- 外部の検証根拠の取込みが成功
- リポジトリのソースの境界の準拠確認が成功
- 検証根拠の生成元を確認する`name / command / passed`が正式な契約と一致
- 実行後のソースを確認済み
- 実行前後HEAD同一
- 終了時にソースと索引に変更がないこと

合成データから得た検証根拠を、実際の外部資産についての検証根拠へ格上げしません。

## Required Evidence B — generated artifact reproducibility / #255

関連: `ISSUE-000025` / GitHub #255

### 目的

正本となる作業領域などの正式な入力から、保存済みの生成成果物を、同じ変更のない版で再生成し、照合できることを示します。

対象:

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/order_rollout.durable-run.m0.21.start.json`

正本は記述した入力元です。生成成果物を新たな正本としたり、意味の正本へ書き戻したりしません。

### checkとrepairを分ける

判定段階Bで最終的な検証根拠を得る入口は、変更を加えずに確認する`tools/reference-generated-artifact-evidence`です。

生成成果物が古い場合に限り、正式な生成器を使って修復します。差分を意味の観点から確認してコミットします。`-write`の直後に作業ツリーへ変更が残っている状態や、安全な置換処理の成功だけでは完了の検証根拠になりません。

### 完了条件

少なくとも次を要求します。

- 意味の正本とプラグインの実装固定情報が期待出力と一致
- 永続的な実行記録が期待出力と一致
- 検証根拠の生成元を確認する`name / command / passed`が正式な契約と一致
- `repositoryRevision = R`
- `reproducibleRevision = true`
- 実行後のソースを確認済み
- 実行前後HEAD同一
- 終了時にソースと索引に変更がないこと

## Required Evidence C — real PostgreSQL ephemeral fixture / #308

関連: `ISSUE-000031` / GitHub #308

### 目的

`Order -> OrderLine[]`について、確認済みの意味上の関係、関係の保存方法、契約に基づく問い合わせの実現、ドライバーの実行が、**実際のPostgreSQLの実行環境**でも同じ意味上の境界を保つことを示します。

これは、本番にある既存のPostgreSQLデータベースの状態を観測した検証根拠ではありません。

### Evidence専用Database

#308では、検証用ツールが管理する一時的な試験用環境を使います。準備を始める前に少なくとも次を確認し、条件を満たさなければ安全側に停止します。

- 正本となる作業領域の検証成功
- 既定のデータベースではない
- 利用者が定義した関係が0件
- Event Trigger 0件
- 試験用の構成要素が存在しない

既存の関係やイベントトリガーを削除・無効化し、検証用のデータベースに転用することはしません。

### DB-side fixture ownership / cleanup

試験用の構成要素を作成するとき、データベース側に所有者を示す印を付けます。

後片付けでは表をロックし、データベース側の印が作成時の印と厳密に一致する構成要素だけを削除します。同名の構成要素が別の主体によって置き換えられた疑いがある場合は、名前だけを根拠に削除しません。

後片付けができず、検証ツールの試験用構成要素が残る可能性は許容します。別の主体の構成要素を誤って削除するより安全なためです。そのデータベースは、次回の空のデータベースであることを求める安全確認には通りません。

### runtime Evidence

同じ実行で少なくとも次を確認します。

- 2件取得
- 正常0件
- 表が存在しない場合を実行時エラーとして扱う
- 元の実行結果から意味上の表現への明示的な対応付け
- 補助列を外部へ露出しないこと
- 引数名を追跡し、値を通常の検証根拠へ保存しないこと

### 完了条件

判定段階Bに採用する#308の検証根拠には、少なくとも次を要求します。

- `evidenceNature = real-postgres-ephemeral-fixture`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- Canonical target identity / parameter / deterministic Semantic attributesが正規contractと一致
- `fixtureDatabaseSafety.guardPassed = true`
- データベース側の所有者を示す印が厳密に一致
- `fixtureCleanupSafety.verified = true`
- 後片付け後、利用者が定義した関係0件・イベントトリガー0件・試験用構成要素なし
- 実行後のソースを確認済み
- 実行前後HEAD同一
- 終了時にソースと索引に変更がないこと
- DSN・認証情報・引数の値・実際のデータベース名を保存しないこと

## Final convergence Evidence / #387

関連: `ISSUE-000037` / GitHub #387

### 目的

三つのRequired Evidenceが個別に再現可能なだけでなく、**同じcandidate revision R**へ収束したことを示します。

final runnerは `tools/reference-gate-b-final-evidence` です。具体起動条件はRunbookを正とします。

### lane順序

```text
#310
  ↓ source check
#255
  ↓ source check
#308
  ↓ source check
final source check
```

#310または#255が失敗した場合、不要なPostgreSQL fixture操作へ進みません。

### final success

final成功には少なくとも次を要求します。

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
workingTreeClean = true
reproducibleRevision = true
allLanesSameRevision = true
childArtifactsRetained = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

各laneには少なくとも:

- `passed = true`
- `repositoryRevision = R`
- `reproducibleRevision = true`
- `artifactFile`
- `evidenceArtifactFingerprint`

を要求します。

final runnerが実際に読んだchild stdout exact bytesをRepository外へ保持し、そのSHA-256を`evidenceArtifactFingerprint`として記録します。fingerprintだけでEvidence本文や合否を代替しません。

## Final runのexecution Evidence境界

### Repository source observation

共有source observationは概念上次を同じHEAD観測窓で取得します。

```text
HEAD before
  ↓
git status --porcelain --untracked-files=all --ignored=matching
  ↓
git ls-files -v -z
  ↓
HEAD after
```

少なくとも次をfail closedで要求します。

- HEAD stable
- tracked / untracked / ignored local stateなし
- tracked index entryが通常`H`のみ
- full checkout
- Repository / work tree / common dir / index / object store / namespace等のGit identity overrideなし
- command-scope Git config injectionなし

source observation不能、HEAD drift、ignored local state、非標準index stateを成功Evidenceへ持ち込みません。

### Go environment

final runnerのcompile前からRepository外のGo environment / workspace / flagsを暗黙入力へしません。

actual final JSONでは少なくとも:

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

を確認します。

### PostgreSQL secret exposure

final launch environmentへdirect `TEI_TEST_POSTGRES_DSN`を置きません。

Repository外の一行secret fileを指定し、final runnerはsecret-free clean-source preflight成功後にoriginal secret fileをstable single-openします。normalized DSN本文はmemoryに保持したうえで、Repository外owner-only temporary handoff fileへ複製します。

```text
outer go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathのみ

Go / Git probe / #310 / #255
  -> DSN本文なし
  -> original secret file pathなし
  -> handoff file pathなし

#308 go run / compiler / linker
  -> DSN本文なし
  -> original secret file pathなし
  -> temporary handoff file pathのみ

#308 runtime
  -> handoff fileからDSN本文を読込
  -> DSN / file path environmentをsource probe前に消去
  -> 以後DSN本文はmemoryのみ
```

normal final runner return時にtemporary handoff directoryをcleanupします。process強制killやhost停止に対するOS-level完全消去までは保証せず、Repository外owner-only temporary scopeへ限定します。

DSN本文、credential、original secret path、handoff pathを通常Evidenceへ保存しません。

これらのenvironment / secret境界はEvidence natureを増やすものではなく、Required Evidence取得の再現性・least exposure条件です。

## Same-run child exact artifact independent audit

final convergence成功だけでGate B Readyにはしません。

#387が実際に保持したsame-run child exact artifactsを、read-only補助tool `tools/reference-gate-b-readiness-artifact-audit` で独立に再監査します。具体command / 順序はRunbookを正とします。

このartifact auditは第四Required laneではありません。`artifactAuditPassed = true`だけでReadyを宣言しません。

独立監査では少なくとも次を確認します。

- fixed 3 lane / fixed artifact filename
- final `artifactFile`との対応
- child exact bytes SHA-256 == final `evidenceArtifactFingerprint`
- same `repositoryRevision = R`
- child / final lane envelopeのnature / revision / reproducible一致
- #310 Workspace identity
- #310 / #255 producer checkの`name / command / passed` exact一致
- #308 Canonical target identity / parameter / deterministic Semantic attributes
- #308 fixture前後安全性 / ownership / cleanup
- post-run clean source
- secret / credential / DSN / physical identity等の禁止field非保存

同じRで後から再実行した別artifactを、final convergenceが読んだsame-run artifactへ読み替えません。

## Optional / unresolved Evidence — actual external asset / Oracle

関連:

- `ISSUE-000022` / #214
- `ISSUE-000033` / #371
- `ISSUE-000034` / #376

### 性質

これは**actual external asset Evidence**です。

TEIが所有していない既存Oracle資産について、実際に観測したphysical facts、provenance、effective read-only、concrete driver behavior、actual readを扱います。

Evidence natureとして重要ですが、Gate B Readyの第四Required laneではありません。

### 未観測事実を補完しない

少なくとも次を観測前に推測しません。

- physical object kind / identity
- physical parameter / raw columns
- physical type / NULL semantics
- native Oracle SQL
- concrete driver constraints
- effective read-only mechanism

### Evidence intakeとactual readを分ける

external Evidence artifactは非Canonicalのまま、明示authoring済みrealizationと照合します。

acceptance receiptが成功してもactual read成功にはしません。

```text
external observed artifact
      ↓
source-fixed acceptance
      ↓
actual read-only runtime
      ↓
raw result observation
      ↓
explicit Semantic mapping
```

external observed artifact自体の`observedAt`がcandidate Rの時点である必要はありません。ただし、そのartifactをRに対するverified claimとしてHandoverへ載せる場合は、Rとのacceptance / runtime / actual read provenanceを明示します。

旧revisionのacceptanceやactual readをRへ黙ってcarry-forwardしません。

### Gate B通過時の扱い

actual Oracle Evidenceが取得済みでRに対するverified claimとして十分なら、Handoverから追加のverified external asset Evidenceとして参照できます。

未取得、actual read未完了、またはRへの結び付きが不足する場合は、未検証範囲を`unresolved`としてHandoverします。

第一Web Target側で不足Oracle factsを補完しません。

## Gate B Required Evidence Acquisitionの完了条件

少なくとも次が必要です。

- [ ] #310 actual Evidenceがcandidate Rで成立
- [ ] #255 actual Evidenceがcandidate Rで成立
- [ ] #308 actual Evidenceがcandidate Rで成立
- [ ] #387 final convergenceがRで成立
- [ ] Required Evidenceすべて`reproducibleRevision = true`
- [ ] final `allLanesSameRevision = true`
- [ ] `childArtifactsRetained = true`
- [ ] 三つのsame-run child exact artifactsを保存済み
- [ ] child SHA-256とfinal fingerprint一致
- [ ] independent artifact auditをsame-run artifactsへ実行し`artifactAuditPassed = true`
- [ ] #310 Workspace identity / #310・#255 producer commandが正規contractと一致
- [ ] #308 lane固有Semantic / fixture safety / cleanup条件成立
- [ ] final run終了時もR / clean source / index state
- [ ] PostgreSQL fixture前後安全性がactual runで成立
- [ ] Go environment / source observation / secret exposure条件がactual runで成立
- [ ] Evidence natureを相互代替していない
- [ ] secret / credential / DSN本文 / original secret path / handoff path / 実Database名を通常Evidenceへ保存していない

これらを取得した後、`GATE_B_READINESS_AUDIT.md`がRをReady / Not Ready判定します。一つでも成立しなければNot Readyです。

## Handoverとの関係

Gate BがRについてReadyになった場合、同じRを`GATE_B_TO_C_HANDOVER_CONTRACT.md`へ渡します。

```text
Handover Baseline R
  ├─ Required Evidence
  │    ├─ #310
  │    ├─ #255
  │    └─ #308
  │
  ├─ same-run Evidence provenance
  │    ├─ #387 final
  │    └─ retained child exact artifacts
  │
  └─ external asset status
       ├─ verified actual Evidence
       └─ unresolved
```

Handover recordは第二のCanonical authorityではありません。

## 非目標

- actual OracleをGate B第四Required laneへ戻すこと
- artifact auditを第四Required laneへすること
- artifact audit成功だけでReadyを宣言すること
- Evidence natureを一つへ畳むこと
- synthetic / generated / provider fixture / actual external Evidenceを相互代替すること
- external EvidenceからCanonical realizationを自動生成すること
- generated artifactを手編集すること
- 認証情報や秘密情報をEvidenceへ保存すること
- Runbook以外に実行commandの第二正本を作ること
- CIをProduct Evidenceへ読み替えること

## 関連

- `GATE_B_EVIDENCE_RUNBOOK.md`
- `GATE_B_READINESS_AUDIT.md`
- `GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `tools/reference-gate-b-readiness-artifact-audit/README.md`
- ISSUE-000022-external-master-data-physical-query-evidence.md
- ISSUE-000025-reference-generated-artifacts-refresh.md
- ISSUE-000029-gate-b-readiness-audit.md
- ISSUE-000030-gate-b-to-c-handover-baseline.md
- ISSUE-000031-gate-b-postgres-header-detail-evidence.md
- ISSUE-000032-gate-b-deterministic-repository-evidence.md
- ISSUE-000035-gate-b-external-provider-scope-audit.md
- ISSUE-000037-gate-b-three-lane-final-evidence.md
- ISSUE-000061-gate-b-repository-source-observation-stability.md
- ISSUE-000062-gate-b-final-go-environment-isolation.md
- ISSUE-000063-gate-b-child-go-environment-isolation.md
- ISSUE-000064-gate-b-postgres-fixture-ownership-marker.md
- ISSUE-000065-gate-b-final-goenv-launch-guard.md
- ISSUE-000043-gate-b-probe-postgres-dsn-isolation.md
- ISSUE-000044-gate-b-git-probe-postgres-dsn-isolation.md
- ISSUE-000045-gate-b-final-postgres-dsn-secret-file.md
