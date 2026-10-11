# TEIへの貢献

TEIは、アーキテクチャの設計と参照実装を照らし合わせながら検証を続けている実験的なプロジェクトです。

コードの変更以外にも、次のような形で参加できます。

- 参照用ワークスペース（Reference Workspace）やCLIを実際に試した結果
- 既存の業務システムを現在のモデルでは表現しにくい例
- 移行・データベース・配備で問題になる例外的なケース
- 既存のローコード、モデル駆動開発、ワークフロー、スキーマ管理ツールとの比較
- 設計上の調査結果や運用上の知見
- アーキテクチャ上の不整合を小さく再現できる例

中核となる大きな仕組みを追加する前に、解消したい製品要求上の不確実性と、必要になる利用例をIssueや設計文書へ記録します。

## 設計上の基本姿勢

TEIでは、概ね次を優先します。

- runtimeの推測より明示された意味
- special kindを増やすより小さいkernel
- 既に強いnative formatは無理に置き換えない
- AIが不要な箇所ではdeterministicな生成・検証
- command acknowledgementを事実とみなさずObservationを確認する
- 特定技術をSemantic Definitionへ埋め込むよりProvider境界を置く

## 開発フロー（単独オーナー・並行AIセッション）

human ownerは一人ですが、複数のAI session / processが同時に作業するため、Git操作は単独process前提にしません。branch / PRは人的reviewの儀式ではなく、**並行作業の分離、Gateの保持、競合のfail-close**のために使います。

- 既存のtask / lane branchがある作業はそのbranchを再利用する。作業のたびに新しいremote branchを増やさない。
- write前にremoteを再取得し、open PR、対象branchのahead / behind、変更pathの重複を確認する。
- shared live branchはrebase / reset / force updateしない。最新`main`へ追随するときは、必要なら現在のlane tipを第一親にした通常mergeを使う。
- milestone / Gateが完了したらPRまたは通常mergeで`main`へ戻す。統合後、独自aheadもactive PRもないbranchだけをnon-forceで最新`main`へ収束させる。active laneを機械的に削除しない。
- `main`へforce pushしない。並行更新でexpected head / fast-forward条件が外れた場合は、forceで突破せずremote stateを再取得してreconcileする。
- コードを変えたときは、`main`へ統合する前に`tools/repository-health-check.sh`を通す。文書だけの変更は、リンク・正本性・状態表現・Gate / Evidence claimを壊していないかを確認する。
- **Draft PRは変更種別にかかわらずmergeしない。** merge操作の直前にPRを再取得し、`draft=false`、exact head、base、変更path、current `main`を確認する。PR作成時や数分前の状態を使い回さない。
- code変更を含むPRは、**そのexact headに対するfull `tools/repository-health-check.sh` PASSとclean pre/post stateの記録**がmerge authorityである。統合直前に、記録のcandidate headと対象`main`が現在のPR head / current `main`と一致することを確認する。focused test、static tree audit、PRのmergeable / merged判定は代用しない。記録が無い、またはstaleならmergeせず停止する。
- qualification toolがmachine-readable receipt / integration preflightを提供するlaneでは、通常の`git merge`やPR mergeの直前に、exact candidateのclean checkoutからcandidate版preflightを実行し、exact expected mainの別clean integration checkoutを検査する。receiptが固定したcandidate / main / runner（preflight自身を含む）と現在のGit stateが一致しない場合は統合せず、最新mainを取り込んだcandidateでqualificationから取り直す。preflightがPASSした後はbranch名を再解決せず、出力されたexact `candidate-head` SHAを統合対象として引き継ぐ。
- exact-head / exact-main qualification中は、そのreceiptを無効にする`main`移動を避ける。無関係な文書PRでも`main`を動かす場合はexisting laneにparkし、必要ならPRをcloseして、protected integrationが安定してから同じbranch/PRを再開する。Draftだけをmerge防止策にしない。
- 必須PASSなしにcode-changing PRがmergeされた場合、GitHubの`merged`状態をqualificationへ読み替えない。protected candidateを動かさず、current `main` exact headのtrusted full Repository healthを取得してからforward reconciliationを再開する。
- open PRを一括mergeしない。各PRのGate authority、exact head、main movement、並行laneを個別に確認する。
- forward containmentや資格確認のために保持しているbranchは、PRがclosed/mergedになっただけでは通常cleanupへ戻さない。正本のGate handoffが完了し、candidate lineageをもう保持しなくてよいことを確認してからnon-forceで収束させる。
- GitHub Actionsは月間quota等で使えないことがある。hosted CIが使えないことはlocal Gateを省略する理由にしない。
- branch名・commit・PRの有無はProduct / Issue / Gateの状態を表さない。状態は正本文書とEvidenceで管理する。

### R7B qualification branch

`research/selective-knowledge-routing-r6` はFirst Web Target R7Bのlive qualification branchです。canonical 11 sourceは通常のbranch merge対象ではありません。

禁止は次のとおりです。

- B4 branchのgeneric merge / squash / cherry-pickをintegration authorityとして扱う
- PASS receiptなしにcanonical 11を`main`へcopy / mergeする
- B4をrebase / reset / force updateしてlineageを切る
- `main`上にsourceが存在すること自体をqualification Evidenceとみなす

統合はISSUE-000069-first-web-target-r7b-exact-t-replay.mdの順序に従います。

```text
trusted-local full Repository-health PASS receipt
  -> integration receipt preflight
  -> exact-parent merge object preparation
  -> non-force publication
  -> post-merge verification
  -> B4 convergence
  -> Packet M / actual replay
```

この順序を満たさないcanonical source mergeを検出した場合は、履歴を書き換えずforward containmentでGateを復旧し、資格確認済みB4 candidateを保持します。

## 開発時の確認

```bash
gofmt -l .   # 出力が空であること
go vet ./...
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

上記をまとめて確認する読み取り専用のスクリプトが`tools/repository-health-check.sh`です（`--fast`では`go test`を省略します）。GitHub Actionsを使えない間は、mainへpushする前に手元でこの確認を通してください。背景は`ISSUE-000055`にあります。

Reference Workspaceには、過去contractとの互換性を確認するためのwarningが意図的に含まれる場合があります。

文書だけを変更した場合も、リンク、正本性、状態表現、Gate / Evidence claimを壊していないかを確認してください。文書品質の詳細は[`docs/documentation-quality.md`](docs/documentation-quality.md)を参照します。
