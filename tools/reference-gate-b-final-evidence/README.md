# Gate B 3-lane Final Evidence Runner

Gate Bの必須Evidenceを、同じcandidate repository revisionへ揃えたことを最終確認するためのtoolです。

`ISSUE-000035` / #380以降、Gate Bの必須実行Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

このtoolは三つの既存runnerを置き換えません。それぞれを正規入口から実行し、返されたEvidence JSONの最小envelopeを照合します。

具体的なGate Bの実行順、安全条件、停止条件の正本は`plan/GATE_B_EVIDENCE_RUNBOOK.md`です。このREADMEはtool固有の利用境界を説明するもので、第二のRunbookにはしません。両者に矛盾が見つかった場合は、矛盾を解消するまで成功Evidenceへ採用しません。

## 実行

Evidence専用の空PostgreSQL Databaseを用意し、`TEI_TEST_POSTGRES_DSN`を設定します。

```bash
TEI_TEST_POSTGRES_DSN='...' \
go run ./tools/reference-gate-b-final-evidence \
  > /tmp/tei-gate-b-final-evidence.json
```

Repository内の新規fileへstdoutを直接redirectしません。command開始前にshellがfileを作成し、開始時clean条件を壊すためです。

Repository全体の`go test ./...`も#310で実行したい場合だけ`-broad`を付けます。

```bash
TEI_TEST_POSTGRES_DSN='...' \
go run ./tools/reference-gate-b-final-evidence -broad \
  > /tmp/tei-gate-b-final-evidence.json
```

## 実行順

```text
clean candidate revision R
        ↓
#310 deterministic Repository Evidence
        ↓ runner自身もsource再確認
#255 generated artifact reproducibility
        ↓ runner自身もsource再確認
#308 real PostgreSQL Provider Evidence
        ↓ runner自身もsource再確認
final source check
```

#310または#255が失敗した場合、#308へ進みません。

つまり、deterministic境界やgenerated artifactが壊れている状態で、不要なPostgreSQL fixture操作を開始しません。

各laneのrunnerが失敗した場合も、その失敗を返す前に最終runner自身がRepository sourceを再観測します。lane実行後に次のいずれかを満たせなければ、その時点で停止して次laneへ進みません。

- `postRunRepository.checked = true`
- candidate revisionからHEADが変わっていない
- working treeがcleanである

これにより、「laneは失敗したがsourceも変化していた」「post-run観測が完了していない」といった状態を、後続laneの前提へ持ち越しません。

## 成功条件

各laneについて次を要求します。

- subprocessが正常終了する
- `evidenceNature`が期待値と一致する
- `repositoryRevision`がrunner開始時のcandidate revisionと一致する
- `reproducibleRevision = true`

さらにlane間でも同じrevisionであることを確認します。

```text
#310.repositoryRevision == R
#255.repositoryRevision == R
#308.repositoryRevision == R
```

最後にrunner自身もRepository HEAD / working treeを再観測し、`repositoryevidence.IsReproducible`の条件を満たす同じclean revisionに留まっていることを確認します。

## JSONへ残すもの

最終JSONには、三laneを再掲せず、照合に必要な最小情報だけを残します。

- `evidenceNature = gate-b-three-lane-final-convergence`
- candidate `repositoryRevision`
- `workingTreeClean`
- `reproducibleRevision`
- `postRunRepository`
- Go version
- `broadRequested`
- `allLanesSameRevision`
- laneごとの
  - name
  - command
  - expected / observed Evidence nature
  - repository revision
  - reproducible revision
  - pass / fail

次は保存しません。

- `TEI_TEST_POSTGRES_DSN`
- credential / secret
- runtime parameter値
- 各laneのEvidence本文
- test stdout
- PostgreSQL Database名

三laneそれぞれの詳細Evidenceは、必要に応じて個別runnerから別途保存します。

## 事前確認と最終収束を分ける

このtoolは、Gate B準備中のdiagnostic / repairを一つへまとめるものではありません。

事前確認では引き続き各laneの正規toolを個別に使います。

- #310でdeterministic failureを確認する
- #255がstaleなら個別golden toolの`-write`で修復し、差分を確認してcommitする
- #308はEvidence専用空DBで安全条件を確認する

source変更が出尽くした後にcandidate revision `R`を固定し、本toolを**最終収束の確認**として利用します。

#255の修復操作やdirty treeを本toolから実行する経路は持ちません。

## #214 actual Oracleとの関係

#214 / #371 / #376は本toolのlaneではありません。

actual Oracle EvidenceはMilestone 2の既存外部資産実証、Milestone 8の第二Provider圧力試験として継続します。未完了のままGate Bを通過する場合は、Handover Baselineの`unresolved`へ明示します。

本toolの成功をactual Oracle成功Evidenceへ格上げしません。

## 非目標

- 三laneの個別runnerを置き換えること
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`をこのREADMEで置き換えること
- generated artifactを修復すること
- PostgreSQLの既存Databaseをfixture用に変更すること
- actual Oracle Evidenceを代替すること
- Gate B Readyをtoolのexit codeだけで自動宣言すること
- GitHub Actionsを再有効化すること
