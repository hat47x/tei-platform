# TEIへのContribute

TEIは、現在もArchitectureとReferenceを相互にpressureしながら育てている実験的なプロジェクトです。

Code contributionだけでなく、次のような参加も重要です。

- Reference WorkspaceやCLIを実際に試した結果
- 既存の業務システムが現在のmodelへ素直に当てはまらない例
- migration / database / deploymentのedge case
- 既存のlow-code、model-driven、workflow、schema toolとの比較
- 設計上の調査結果や運用上の知見
- Architecture上の不整合を小さく再現できる例

大きなCore primitiveを追加する前に、まず「どのProduct Requirement上の不確実性を減らしたいか」「どの利用例で必要か」をIssueや設計文書へ残すことを優先します。

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
- code変更を含むDraft PRは、GitHub上でmerge可能でも統合しない。**そのexact headに対するfull `tools/repository-health-check.sh` PASSとclean pre/post stateの記録**がmerge authorityであり、focused test、static tree audit、PRのmergeable判定は代用しない。
- forward containmentや資格確認のために保持しているbranchは、PRがclosed/mergedになっただけでは通常cleanupへ戻さない。正本のGate handoffが完了し、candidate lineageをもう保持しなくてよいことを確認してからnon-forceで収束させる。
- GitHub Actionsは月間quota等で使えないことがある。hosted CIが使えないことはlocal Gateを省略する理由にしない。
- branch名・commit・PRの有無はProduct / Issue / Gateの状態を表さない。状態は正本文書とEvidenceで管理する。

### R7B qualification branch

`research/selective-knowledge-routing-r6` はFirst Web Target R7Bのlive qualification branchです。canonical 11 sourceは通常のbranch merge対象ではありません。

禁止:

- B4 branchのgeneric merge / squash / cherry-pickをintegration authorityとして扱う
- PASS receiptなしにcanonical 11を`main`へcopy / mergeする
- B4をrebase / reset / force updateしてlineageを切る
- `main`上にsourceが存在すること自体をqualification Evidenceとみなす

統合は[`plan/issue/000069-first-web-target-r7b-exact-t-replay.md`](plan/issue/000069-first-web-target-r7b-exact-t-replay.md)の順序に従います。

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

## Development

```bash
gofmt -l .   # 出力が空であること
go vet ./...
go test ./...
go run ./cmd/tei validate -workspace reference/workspace
```

上記をまとめて実行するRead-onlyの補助として`tools/repository-health-check.sh`があります（`--fast`で`go test`を省略）。GitHub Actionsを使えない間は、mainへpushする前に手元でこの確認を通してください。背景は[`ISSUE-000055`](plan/issue/000055-verification-gate-while-ci-suspended.md)にあります。

Reference Workspaceには、過去contractとの互換性を確認するためのwarningが意図的に含まれる場合があります。

文書だけを変更した場合も、リンク、正本性、状態表現、Gate / Evidence claimを壊していないかを確認してください。文書品質の詳細は[`docs/documentation-quality.md`](docs/documentation-quality.md)を参照します。
