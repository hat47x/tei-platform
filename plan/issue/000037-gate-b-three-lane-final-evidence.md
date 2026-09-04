# ISSUE-000037: Gate Bの三つの必須Evidenceを同一revisionへ最終収束する

- 状態: In Progress — runner実装 / 制御フローpure test / 隔離source guard確認済み、actual final run待ち
- GitHub Issue: #387
- Gate B scope判断: `ISSUE-000035` / #380
- 文書同期: `ISSUE-000036` / #383（Done）
- Gate B監査: `ISSUE-000029` / #273

## 目的

Gate Bの完了判定に使う三つのEvidenceが、**同じcandidate repository revision**から得られたことを機械的に確認します。

`ISSUE-000035` / #380以降、Gate Bで必須とする実行Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

各laneはすでに、自分自身の実行前後でRepository sourceを固定します。

本Issueではさらに、その三つが相互にも同じrevisionを参照していることを確認します。

## なぜ横断確認が必要か

各laneが個別に再現可能でも、次の状態をGate B Readyにしてはいけません。

```text
#310 -> R1
#255 -> R2
#308 -> R3
```

この三つを人手で「だいたい同じ時期の成功」とまとめると、第一Web Targetへ渡すHandover Baselineのsourceが一意になりません。

そこで、candidate revisionを`R`として固定し、三laneを同じRで実行したことを最後に確認します。

## Runbookとの権威関係

具体的なGate Bの実行順、安全条件、停止条件の正本は引き続き`plan/GATE_B_EVIDENCE_RUNBOOK.md`です。

本toolは、Runbookで定義された三lane最終収束を機械的に実行・照合するための**薄い実行helper**です。各laneの正規runnerやRunbookを置き換える第二の実行契約にはしません。

したがって、toolの実装やREADMEとRunbookが矛盾した場合は、矛盾を解消するまで成功Evidenceへ採用せず、Runbook側の契約を確認します。

## 正規tool

```bash
TEI_TEST_POSTGRES_DSN='...' \
go run ./tools/reference-gate-b-final-evidence \
  > /tmp/tei-gate-b-final-evidence.json
```

Repository内の新規fileへstdoutを直接redirectしません。shellがcommand開始前にfileを作成し、clean working tree条件を壊すためです。

#310でRepository全体の`go test ./...`も実行したい場合だけ`-broad`を付けます。

## 実行順

```text
clean candidate revision R
        ↓
#310 deterministic Repository Evidence
        ↓ runner source再確認
#255 generated artifact reproducibility
        ↓ runner source再確認
#308 real PostgreSQL Provider Evidence
        ↓ runner source再確認
runner自身の最終source確認
```

#310または#255に失敗した場合は#308へ進みません。

これは、deterministicな境界やgenerated artifactに問題が残る状態で、Evidence用PostgreSQLへfixtureを作成する必要がないためです。

各laneの成功・失敗にかかわらず、lane終了後には最終runner自身がRepository sourceを再確認します。次のすべてを確認できない場合、その時点でfail closedし、次laneへ進みません。

- `postRunRepository.checked = true`
- `revisionStable = true`
- `workingTreeClean = true`

lane runnerが失敗した場合も、source再確認を飛ばして即returnしません。lane失敗とsource変化が同時に起きた状態を見落とさないためです。

## 照合する情報

各laneのJSONから、次の小さいenvelopeだけを横断照合します。

- `evidenceNature`
- `repositoryRevision`
- `reproducibleRevision`

成功には次をすべて要求します。

- 各runnerが正常終了する
- `evidenceNature`が期待するlaneと一致する
- `repositoryRevision = R`
- `reproducibleRevision = true`
- 三laneすべてが同じRを参照する
- 各lane後のsource再確認が完了している
- 最終確認時もRepository HEADがRのままである
- 最終確認時もworking treeがcleanである

## child Evidence artifactのidentity

最終runnerが実際に読み込んだchild Evidence JSONを後から識別できるよう、各laneのstdout exact bytesに対するSHA-256を`evidenceArtifactFingerprint`として残します。

このfingerprintは、child JSONから横断判定へ使う三fieldとは別の**追跡情報**です。空白や改行だけが変化した場合も別artifactとして扱います。

目的は、個別runnerから別途保存したEvidenceと、final runnerが実際に読み込んだartifactを後から突き合わせられるようにすることです。

次の意味へは格上げしません。

- fingerprintだけでchild Evidence本文の存在や真実性を証明したことにする
- fingerprintから保存していないchild Evidence本文を推測・再構成する
- fingerprint一致だけでlaneの`evidenceNature` / revision / reproducible条件を省略する

## 最終JSON

横断Evidenceには、個別laneの詳細本文を複製しません。

主に次を残します。

- `evidenceNature = gate-b-three-lane-final-convergence`
- candidate `repositoryRevision`
- `workingTreeClean`
- `reproducibleRevision`
- `postRunRepository`
- Go version
- `broadRequested`
- `allLanesSameRevision`
- laneごとのname / command / Evidence nature / revision / reproducible / `evidenceArtifactFingerprint` / pass-fail

次は保存しません。

- PostgreSQL DSN
- credential / secret
- runtime parameter値
- PostgreSQL Database名
- test stdout
- 各laneのEvidence本文

詳細Evidenceが必要な場合は、各laneの正規runnerから別途保存します。

## 事前確認と最終収束を分ける

このtoolは、Gate B準備中の診断や修復を自動化するものではありません。

事前確認では個別toolを使います。

- #310でdeterministicな不足を確認する
- #255でstaleを確認し、必要な場合だけ正規`-write`で修復する
- 修復差分を確認してcommitする
- #308をEvidence専用空DBで必要に応じて事前確認する

source変更が出尽くした後にRを固定し、本toolを最終収束の確認として利用します。

## #214 actual Oracleとの関係

#214 / #371 / #376は本Issueの必須laneではありません。

actual Oracle EvidenceはMilestone 2の既存外部資産実証、Milestone 8の第二Provider圧力試験として継続します。

未完了のままGate Bを通過する場合は、Handover Baselineへ`unresolved`として明示します。

本toolの成功をactual Oracle成功Evidenceへ読み替えません。

## 2026-09-04 隔離実行実績

### package-level pure test

GitHub上の現行runner実装と同じロジックを隔離Go moduleへ配置してpackage-levelのcompile / pure testを実行しました。

初期のenvelope / lane順序testでは次を確認しました。

```text
ok  tei/tools/reference-gate-b-final-evidence  0.002s
```

- Evidence nature mismatchをrejectする
- candidate revision mismatchをrejectする
- `reproducibleRevision=false`をrejectする
- 三lane同一revisionの収束条件
- PostgreSQL laneを最後に実行する順序
- `-broad`を#310だけへ渡すこと

PR #394では制御フローをpure test可能にし、次を追加しました。

- lane runner失敗時にもsourceを再確認してから停止する
- lane後のsource driftを検出したら次laneへ進まない
- 3 lane成功時、各lane後3回 + 最終1回のsource確認を通って初めて`reproducibleRevision=true`になる

PR #396ではさらに、`Checked=false / RevisionStable=true / WorkingTreeClean=true`という未確認状態でも次laneへ進まないtestを追加しました。

PR #396後の隔離moduleでは、新testを含めて次を確認しています。

```text
ok  tei/tools/reference-gate-b-final-evidence  0.001s
```

これはGitHub branch上の実装とtest seamのcompile / pure test確認であり、actual Gate B Evidenceではありません。

### actual `repositoryevidence`実装との組合せ

最小stubだけの確認で終わらせず、Repository上の`tools/internal/repositoryevidence/repository.go`実装へ差し替え、そのpackage testも隔離Git repositoryで実行しました。

```text
ok  tei/tools/internal/repositoryevidence              0.002s
ok  tei/tools/reference-gate-b-final-evidence          0.002s
```

さらにrunnerを実際に起動し、source guardのfail-closed順序を確認しました。

#### clean repository + DSN未設定

- exit codeは非0
- `workingTreeClean = true`
- `reproducibleRevision = false`
- laneは未実行
- `TEI_TEST_POSTGRES_DSN`不足としてPostgreSQL接続前に停止

#### dirty repository + DSN設定済み

- exit codeは非0
- `workingTreeClean = false`
- `reproducibleRevision = false`
- laneは未実行
- DSN利用より先にworking tree guardで停止

いずれも成功Evidenceへ誤昇格しません。

### この確認の限界

これらは**TEI本体の実Repository checkout上の#310 / #255 / #308 Evidenceではありません**。

通常のGitHub cloneはこのセッション環境でDNS解決できないため、`main`またはcandidate revision `R`をcheckoutしたactual final runは実行できていません。

したがって、通常Repository checkoutのpackage test、Evidence専用PostgreSQLを使うactual final run、Gate B Readyの根拠へは格上げしません。

## 完了条件

- [x] 三laneを順に実行するrunnerを実装した
- [x] `evidenceNature` / revision / reproducible条件のpure testを追加した
- [x] PostgreSQL laneを最後に実行する順序をtestで固定した
- [x] `-broad`を#310だけへ渡すことをtestで固定した
- [x] lane失敗時にもsource再確認してから停止することをtestで固定した（PR #394）
- [x] lane後のsource driftで次laneへ進まないことをtestで固定した（PR #394）
- [x] `postRunRepository.checked=true`を次laneへの必須条件として固定した（PR #396）
- [x] 3 lane成功後の最終source再確認まで通って初めて再現可能とすることをtestで固定した（PR #394）
- [x] 三laneのcommand path / lane名 / `evidenceNature`をinterface testで固定した（PR #399）
- [x] child Evidence artifactのexact bytesをfingerprintできる境界を追加した
- [x] READMEで事前確認と最終収束を分離した
- [x] 隔離Go moduleでpackage-level compile / pure testを実行した
- [x] actual `repositoryevidence`実装と組み合わせて両package testを実行した
- [x] 隔離Git repositoryでclean / dirty source guardのfail-closed動作を確認した
- [ ] 実TEI Repository checkoutの通常Go環境でtool package testを実行する
- [ ] Evidence専用空PostgreSQL Databaseを用意してactual final runを実行する
- [ ] 三laneが同じcandidate revision Rへ揃ったJSONを取得する
- [ ] #273のGate B最終監査へ結果を対応付ける

## 非目標

- 各laneのrunnerを置き換えること
- Runbookを第二の文書へ複製すること
- #255の修復を自動実行すること
- #308のDatabase安全guardを迂回すること
- #214 actual Oracle Evidenceを代替すること
- fingerprintだけでchild Evidence本文を証明・再構成すること
- toolのexit codeだけでGate B Readyを自動宣言すること
- GitHub Actionsを再有効化すること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `ISSUE-000029` / #273
- `ISSUE-000035` / #380
- `ISSUE-000036` / #383（Done）
- #255 / #308 / #310 / #387
- PR #389 / #394 / #395 / #396 / #399 / #401
