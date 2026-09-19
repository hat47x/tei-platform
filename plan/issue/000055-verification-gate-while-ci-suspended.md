# ISSUE-000055: CI停止中でも、未検証の変更がmainへ入り続けない仕組みを決める

- 状態: Done — main緑化、health check script（`--fix`付き）、`AGENTS.md`での必須化を実施し、方式を決定（2026-09-20）
- 提案日: 2026-09-19
- 性格: Repository health / verification gate / Evidence reliability
- 主担当要求: RQ-021 / RQ-022 / RQ-023
- 関連: ISSUE-000029 / ISSUE-000028 / ISSUE-000037 / ISSUE-000060 / CONTRIBUTING.md「Branch / PR運用」§8・§11

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

## 決定（2026-09-20）

利用者は一人で開発しており、人的reviewを介する段階になく、その工数もありません。これを前提に次のとおり決めました。

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
- [x] main直接commitの扱いを決め、`CONTRIBUTING.md` / `AGENTS.md`と実態を一致させる（直接commitを正とした）
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
