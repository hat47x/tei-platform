# ISSUE-000055: CI停止中でも、未検証の変更がmainへ入り続けない仕組みを決める

- 状態: Reopened — 2026-10-09。Draft / 未資格mergeの再発を受け、汎用qualification receiptを追加検証中
- 提案日: 2026-09-19
- 性格: Repository health / verification gate / Evidence reliability
- 主担当要求: RQ-021 / RQ-022 / RQ-023
- 関連: ISSUE-000029 / ISSUE-000028 / ISSUE-000037 / ISSUE-000060 / CONTRIBUTING.md「Branch / PR運用」§8・§11

## 2026-09-28 運用更新（現行ルール）

2026-09-20の決定のうち、**「main直接commitを正とする」「PRを不要とする」部分は現行運用では失効**しています。human ownerが一人でも、複数のAI session / processが同時にRepositoryを更新することが常態化し、protected Gate対象のsourceがqualification receiptなしにgeneric mergeへ巻き込まれる事象も発生しました。2026-09-27に#909で `AGENTS.md` / `CONTRIBUTING.md` をfail closedへ更新し、現在はそちらを正本とします。

現行ルールは次のとおりです。

- `main`へ直接commitしない。既存のtask / lane branchを再利用し、並行作業ごとにsibling branchを増やさない。
- shared live branchでrebase / reset / force updateを行わない。最新`main`への追随が必要なら、現在のlane tipを第一親にした通常mergeを使う。
- milestone / Gate完了時にPRまたは通常mergeで`main`へ戻し、統合後は独自aheadもactive PRもないbranchだけをnon-forceで収束または削除する。
- code変更を含むDraft PRはmerge authorityではない。**そのexact headに対するfull `tools/repository-health-check.sh` PASSとclean pre/post state**が必要で、focused test、static tree audit、GitHubの`mergeable`表示は代用しない。
- R7Bのような個別qualification contractを持つprotected laneでは、repository-wide healthに加えて、その計画文書が定めるreceipt-bound sequenceを満たすまでgeneric merge / squash / cherry-pick / source copyを行わない。

GitHub側のserver enforcementにも依存できません。2026-09-28のAPI確認では`main`は `protected=false`、required status checksはoffでした。またprivate repositoryのrepository rulesets APIは、現在のGitHub planでは利用不可でした。したがって、現時点のfail-closed Gateは**GitHubが強制する保護ではなく、Repository内の運用契約**です。GitHub側のmerge操作が可能であること自体をintegration authorityとみなしません。

この更新は2026-09-20の判断記録を削除するものではありません。下記「旧決定」は当時の履歴として残し、矛盾する箇所では本節と現行`AGENTS.md` / `CONTRIBUTING.md`を優先します。

## 2026-09-28 merge-bypass incident

上記の運用更新後にも、server-side protectionが無いことの実害が再発した。

2026-09-28T04:14:06Z、次の二つのDraft PRがRepository操作によってmergeされた。

```text
#910 docs-only
  head  = b6b6ce4786e3284ef75c4bc2282f2821d75e7112
  merge = c7137690e381a5678a24303fd6f46517c6705195

#908 code-changing containment
  head  = 91b0dad6b945485e16165f5b2631deaad4293a05
  merge = ee5b5b14f87536e08c9f713504de99f71f082c52
```

#908は、exact-head full `tools/repository-health-check.sh` PASSが観測されるまでDraftのまま保持する、とPR本文・コメント・Repository ruleで明示していた。それにもかかわらずmergeされたため、**Draft flagはこのRepositoryではmerge防止機構として扱えない**ことが実証された。

merge後の監査:

- #908 head `91b0dad6...`: GitHub status 0 / workflow run 0;
- current main `ee5b5b14...`: GitHub status 0 / workflow run 0;
- Library search: exact-head/full-checkout Repository-health PASSなし;
- current main treeはcontainment意図どおり（R7B canonical 11をpre-R7B baselineへ11/11復元、TEI-LD four pathsを4/4 absenceへ復元）。

よってこのincidentでは、source treeのforward containment自体は成立しているが、**merged stateはqualification Evidenceではない**。current main exact head `ee5b5b14...` のtrusted full Repository healthを新しいforward stopとし、それまでprotected R7B / TEI-LD candidate refsを動かさない。

このincident以降の追加運用:

- exact-head / exact-main qualification中は、無関係なdocs-only変更であっても`main`を動かさない;
- main-moving PRはexisting laneに保持し、必要ならPR自体をclose/parkする。Draftだけを安全装置にしない;
- open PRを機械的・一括にmergeしない。各PRのGate authority / exact head / main movement / parallel laneを個別確認する;
- 必須PASSなしにcode-changing PRがmergeされた場合、既成事実として後続qualificationへ流用せず、current main exact headを新しいforward qualification boundaryとして扱う;
- protected candidate branchはそのforward qualificationが成立するまで凍結する;
- docsの未統合変更は同じ既存docs laneを再利用し、sibling branchを増やさない。

この追記は`AGENTS.md` / `CONTRIBUTING.md`の同日hardeningと同じ事故記録を保持する。

## 2026-10-09 再発と汎用qualification receipt

2026-09-28のmerge-bypass incident後も、Repository内の運用契約だけでは同種の事故を止められませんでした。

- PR #967は2026-10-08T06:06:01Z、Draftかつactual exact-head Repository Health / RQ-034 Reference未取得のままmergeされた。
- そのforward containmentとして作成したPR #1027も、本文で「actual qualification取得までDraftを維持しmerge authorityを主張しない」と明記していたにもかかわらず、2026-10-09T07:18:10ZにDraftのままmergeされた。
- #1027のmergeは、そこで追加したclean-state hardeningのqualification Evidenceには数えない。current mainからforward-onlyに扱う。
- 2026-10-09の再確認でも`main`は`protected=false`である。private repositoryのrulesets APIは現行GitHub planでは利用できず、branch protectionの管理APIも現在のGitHub connectorからは利用できない。

このため、文書規則を強めるだけでなく、trusted hostで取得するRepository-wide qualificationを一つの機械可読receiptへまとめるcandidateを追加する。

`tools/run-repository-local-qualification.sh`の責務は次だけに限定する。

```text
remote refsを取得
  -> expected candidate HEAD / expected current mainを固定
  -> mainがcandidateのancestorであることを確認
  -> tracked / untracked / ignored / index stateをexact-clean確認
  -> full repository-health-check.sh
  -> HEAD / clean state再確認
  -> remote refsを再取得してmain移動を拒否
  -> Repository外へoptionalなJSON receiptをatomic publish
```

receiptは少なくともcandidate commit / tree、expected main、health runner blob、qualification helper blob、Go環境、health PASS、HEAD / main安定、clean pre/postを持つ。Repository内へ保存してcandidate HEADを変えない。

このreceiptもGitHub server側の強制保護ではない。merge操作を行うAI / Humanは、直前にPRを再取得し、`draft=false`であることに加えて、code変更ならreceiptの`candidateHead`がPRのexact head、`expectedMain`がその時点のcurrent `main`と一致することを確認する。mainまたはPR headが動けばreceiptはstaleであり、mergeせず取り直す。

この再開作業の完了条件は次のとおり。

- [x] 汎用qualification runnerのshell / Git制御を合成Repositoryでpressureする。local bare originを使い、正常receipt、ignored artifact、stale main、health中のmain移動、Repository内receipt、skip-worktreeを再現した。さらに`tools/repositoryqualification`へ同じ契約のGo回帰テストを追加し、隔離した合成moduleで6 subtestがPASSした。
- [ ] trusted actual TEI checkoutでrunnerを実行し、full Repository Health PASSを含むreceiptを取得する。
- [ ] そのexact receiptを使ったcode-changing PRのmerge前照合を一度実施する。
- [ ] bypass後の`merged`状態をqualificationへ読み替えず、current mainのhealthを独立に確認する。

## 2026-09-29 verification-anchor audit

current main `ee5b5b14f87536e08c9f713504de99f71f082c52` のqualificationを、過去の「green」記録から暗黙継承できないことを履歴上で再確認した。

### Gate B actual full-checkout Evidence

Gate B Readyの正本 `plan/GATE_B_READINESS_AUDIT.md` は、candidate `R = e5231839a5dcc62cce1181a2731fcb7c70d16736` についてactual full checkout、same-R child Evidence、clean source/index、Go環境隔離、independent artifact auditを記録している。

これはGate B Product Evidenceとして有効だが、後続全Repository revisionのhealth PASSではない。current mainはこのRより大きく進んでいるため、RのEvidenceをcurrent mainのqualificationへ延長しない。

### Last confirmed repository-wide Go green

PR #830 merge `ac2fc24f5f3549fc7d54f24e1be44e0359dac857` は次を明示している。

```text
gofmt -l .               empty
go vet ./...             PASS
go test -count=1 ./...   PASS (64 packages)
```

ただし当時は現在の `tools/repository-health-check.sh` 導入前であり、次がreceiptにない。

- Reference workspace validation;
- exact pre/post HEAD binding;
- tracked/untracked/ignored clean state;
- tracked index normal-state check;
- current health-runner blob binding.

したがって `ac2fc24...` も現行full-health receiptではない。

### Health script introduction

PR #832 merge `51bdbb6c80211c7b9cbc4f416b87661b42adc4c3` で `tools/repository-health-check.sh` が導入されたが、そのPR自身のTest planは:

```text
tools/repository-health-check.sh --fast
  PASS
```

であり、`--fast` は `go test -count=1 ./...` を省略する。よってこれもfull-health authorityではない。

### Current gap

`ac2fc24...` からcurrent mainまでは439 commits aheadで、code / Reference / toolingを含む多数の変更がある。さらに#908 containment merge後のcurrent main自身にはGitHub status / workflow runがなく、Libraryにもexact full-checkout PASSはない。

結論:

```text
Gate B actual Evidence at e5231839...
  != current repository health

all-Go green at ac2fc24...
  != current full-health receipt

--fast health at 51bdbb6...
  != full health

current main ee5b5b14...
  requires a fresh exact-main trusted full Repository-health receipt
```

current main receiptはcommit/tree/toolchain/health-runner blob/pre-post cleanへbindingし、full gofmt/vet/test/workspace validationを同じcheckoutで通す。過去anchorを組み合わせてcurrent PASSを推論しない。

## 何に困っているのか

GitHub Actionsは月間quotaの都合で停止しており、[`CONTRIBUTING.md`](../../CONTRIBUTING.md)も手動診断だけに使うと定めています。そのためmainへ入る変更を、機械が検証しない状態が続いています。

一方でmainへは、人手のPR経由と、PRを経由しない直接commitの両方が高い頻度で入っています。2026-09-19時点のmain（`6660ddc`）で`--first-parent`の履歴を数えると、2026-09-16〜19の4日間は220commitのうち77commit（35%）がPR番号を持たない直接commitでした。`CONTRIBUTING.md`§8は通常作業でのmain直接書き込みを禁じています。

検証されないまま積み重なった結果が、この日のうちに実際に見つかっています。

| 見つかった契機 | 内容 | 原因の型 |
| --- | --- | --- |
| Gate B Evidence取得（#310 / #255 / #308 / #387） | `tei validate`が2件、独立artifact auditが1件、実際に実行して初めて失敗 | 補助helperが、本体の文法・出力形式の意図的な進化に追従していない |
| Gate C Evidence取得（Runbookのsource qualification） | architecture-evidenceのtestが**コンパイルできない**（文字列literalに`\n`ではなく実改行）。他5packageにgofmt差 | 書いたコードを一度も`go test` / `gofmt`していない |
| 同Visual review | 標準Projectionのheaderでrevision値が重なる | CSSの折り返し指定漏れ（Custom UIだけ持っていた） |
| リポジトリ全体のsweep（`6660ddc`） | gofmt差85file、`go vet`失敗、**testが1073件走らない**package、失敗test 11件 | 下記 |

sweepの内訳です。

- `tools/reference-business-procedure-check`は、#775以降コンパイルできませんでした。定義がmainに一度も存在しないhelper（`actorContextR3AGOutcome`）をtestが呼び、別のtestは未使用importを持っていました。毎回のPRで追加されてきたR3系・R7Aのpressure testが、実際には実行されていませんでした。
- パッケージをコンパイルできるようにすると、4件のtestが失敗しました。うち3件は#775より前から、1件は最近の追加分で、すべてtest側の不備でした（slice aliasing、期待文言の食い違い、nilと空sliceの比較、ソート順を無視したindex比較）。
- `internal/workspace` 5件、`internal/driver` 2件のtestは、OrderLineの物理authoringをReferenceへ意図的に追加した2026-09-03〜04（認証inputの1件は2026-09-02）以降、失敗し続けていました。「まだauthoringされていないこと」を検証する段階的なtestが、Referenceの前進に追従していませんでした。

## 防ぎたい失敗

Gate Readyや「pressure testがgreen」という主張が、コンパイルできない、または一度も実行されていないコードの上に立つことです。Evidenceは、実際に実行して初めてEvidenceになります。

## 守る原則

- 検証は、CIが使えるかどうかに依存させない。手元で再現できる手順を正本にする。
- Gate Evidence（通常は必要範囲だけを実行するnarrowなEvidence）と、リポジトリ全体の健全性（`go test ./...`）を混同しない。Gate B / Gate C Readyは前者に基づく判定であり、後者がgreenだったことを意味しません。
- 検証gateは、branchの自動削除・resetのようなauthorityを増やさない（`CONTRIBUTING.md`§11の原則を維持する）。

## 現在分かっていること

- `CONTRIBUTING.md`のDevelopment節は`go test` / `go vet` / `tei validate`を定めていますが、`gofmt`を含みませんでした。
- 検証手順は文書にあっても、実行を強制する仕組みがありません。直接commitも§8の禁止のとおりには止まっていません。
- リポジトリはprivateです。

## この起票までに実施したこと（2026-09-19）

- main緑化: `tools/reference-business-procedure-check`のコンパイル修正とtest 4件の修正（#828）、Reference前進に追従していないtest 7件の更新（#829）、gofmt差85fileの整形（#830）。`main`（`b0f466c`）で`gofmt -l .`が空、`go vet ./...`がclean、`go test ./...`が64package全てok、`tei validate`がok、Markdown相対リンクの切れ0件。
- 緑化と同日、`b0f466c`から27commit進んだ`8b964a5`でも、health checkの4ステップすべてがPASSでした。緑化の直後に再発しなかったことの確認であり、gate方式の決定を不要にするものではありません。
- Read-onlyの`tools/repository-health-check.sh`を追加し、`CONTRIBUTING.md`のDevelopment節へ`gofmt`とscriptを追記（`--fast`で`go test`を省略）。

## 決めること

方式は次の組み合わせから選びます。どれも排他的ではありません。

| 案 | 内容 | 主な性質 |
| --- | --- | --- |
| A | `tools/repository-health-check.sh`をPR前・main統合前の手順として運用する（実施済み） | 強制力なし。実行の規律に依存する |
| B | client側`pre-push` hookで同scriptを実行する | 手元では止まるが、迂回できる。hookはcloneごとに設定が必要 |
| C | GitHubのbranch protectionでPRを必須にする | 直接commitを止められる。単独開発では管理者が迂回でき、運用上の意味を決める必要がある |
| D | 自分のWSL2マシンにself-hosted runnerを置き、Actionsのcheckを再開する | hosted minutesを消費しない。private repositoryなので危険は小さいが、runnerの管理が増える |
| E | quotaが復旧したhosted runnerで再開する（`#513`） | 月次quotaのため断続的になる |

あわせて、main直接commitについて、§8の運用と実態のどちらを正とするかを決めます（例外条件を明文化するか、抑止するか）。

## 旧決定（2026-09-20、2026-09-27以降は一部失効）

以下は2026-09-20時点の判断記録です。当時はhuman ownerが一人であることを中心に運用を単純化しましたが、その後の複数AI session並行更新とGate逸脱を受け、上記2026-09-28節で一部を明示的に失効させています。

- **A（script運用）を採用**し、`tools/repository-health-check.sh`をpush前の必須手順にする（`AGENTS.md`）。gofmt差は`--fix`で自動整形できる。並行して動くAI sessionにも同じ規則が適用される。
- B（pre-push hook）は採用しない。Goが（この環境では）WSL2にだけあるため、Windows側cloneのhookは動かず、cloneごとの設定も要る。
- C（branch protection）は採用しない。PRを不要とする方針と矛盾する。
- D・E（self-hosted runner・hosted runnerの再開）は採用しない。単独開発で、CI結果を見る人がいない。
- **main直接commitを正とする。** `CONTRIBUTING.md`の§8を含むbranch / PR運用を廃止し、`AGENTS.md` / `CONTRIBUTING.md`を「main直接・push前にhealth check」へ改めた。
- Gate B `#310`の`-broad`は、必要になったときに実行する。Ready判定の`R`は変えない。

決定と同日、並行commitによりmainが再び赤くなること（gofmt差7file、型不一致によるコンパイルエラー）を確認して修正しました。`--fix`と`AGENTS.md`での必須化は、この再発を受けたものです。規則を守るかどうかは各sessionに依存するため、今後も定期的な点検で確認します。

## 完了条件

- [x] mainをgofmt / vet / test / validateすべてgreenにする（#828 / #829 / #830、`b0f466c`）
- [x] 手元で再現できる検証scriptを用意し、`CONTRIBUTING.md`へ記載する
- [x] gate化の方式（上のA〜E）を決める（Aを採用。方式の選択はこのIssueへ記録し、ADRにはしない）
- [x] main直接commitの扱いを決める（2026-09-20は直接commitを正としたが、2026-09-27以降は現行`AGENTS.md` / `CONTRIBUTING.md`により失効し、既存lane経由へ戻した）
- [x] 決めた方式の運用として、作業の開始時とpush前にhealth checkを実行することを`AGENTS.md` / `CONTRIBUTING.md`へ記載した
- [x] Gate B `#310`の`-broad`は必要になったときに実行する方針とする（Gate B Readyの`R`は変えない）

## 非目標

- GitHub Actionsの再有効化そのもの（`#513`のrunner問題）
- 過去のGate B / Gate C Ready判定を覆すこと（narrow Evidenceに基づく判定であり、本Issueはその外側の健全性を扱う）
- 検証gateへbranch削除・resetなどの強いauthorityを与えること
- 一度のsweepの成功を、以後のgreenの保証と読み替えること

## 関連

- `CONTRIBUTING.md`
- `tools/repository-health-check.sh`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md` / `plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md`
- `plan/issue/000060-legacy-one-shot-workflows-inventory.md`
