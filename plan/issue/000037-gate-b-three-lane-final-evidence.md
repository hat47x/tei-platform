# ISSUE-000037: Gate Bの三つの必須Evidenceを同一revisionへ最終収束する

- 状態: In Progress
- GitHub Issue: #387
- Gate B scope判断: `ISSUE-000035` / #380
- 文書同期: `ISSUE-000036` / #383
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
        ↓ same R / clean
#255 generated artifact reproducibility
        ↓ same R / clean
#308 real PostgreSQL Provider Evidence
        ↓ same R / clean
runner自身の最終source確認
```

#310または#255に失敗した場合は#308へ進みません。

これは、deterministicな境界やgenerated artifactに問題が残る状態で、Evidence用PostgreSQLへfixtureを作成する必要がないためです。

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
- 最終確認時もRepository HEADがRのままである
- 最終確認時もworking treeがcleanである

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
- laneごとのname / command / Evidence nature / revision / reproducible / pass-fail

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

## 完了条件

- [x] 三laneを順に実行するrunnerを実装した
- [x] `evidenceNature` / revision / reproducible条件のpure testを追加した
- [x] PostgreSQL laneを最後に実行する順序をtestで固定した
- [x] `-broad`を#310だけへ渡すことをtestで固定した
- [x] READMEで事前確認と最終収束を分離した
- [ ] 通常Go環境でtool package testを実行する
- [ ] Evidence専用空PostgreSQL Databaseを用意してactual final runを実行する
- [ ] 三laneが同じcandidate revision Rへ揃ったJSONを取得する
- [ ] #273のGate B最終監査へ結果を対応付ける

## 非目標

- 各laneのrunnerを置き換えること
- #255の修復を自動実行すること
- #308のDatabase安全guardを迂回すること
- #214 actual Oracle Evidenceを代替すること
- toolのexit codeだけでGate B Readyを自動宣言すること
- GitHub Actionsを再有効化すること

## 関連

- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `ISSUE-000029` / #273
- `ISSUE-000035` / #380
- `ISSUE-000036` / #383
- #255 / #308 / #310 / #387
