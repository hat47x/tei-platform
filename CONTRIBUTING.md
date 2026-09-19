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

## 開発フロー（単独開発）

このRepositoryは一人で開発しており、人的reviewを介する段階にありません。branch / PR / reviewの手続きは置かず、**mainへ直接commitします**。

- 変更は`main`へ直接commitする。PRは作らない。
- remote branchを作らない。作業branchが必要ならlocalの一時branchとして使い、mainへ取り込んだら削除する。
- pushの前に`git pull --rebase origin main`する。複数のAI sessionやprocessが同じmainへpushするため、直前にfetchして最新へ載せてからpushする。mainへforce pushしない。
- コードを変えたときは、pushの前に`tools/repository-health-check.sh`を通す。文書だけの変更は、リンク・正本性・状態表現・Gate / Evidence claimを壊していないかを確認する。
- commitは小さく、戻せる単位にする。commit messageには、何を変えたかだけでなく、なぜ変えたかを書く。安全網はGit historyであり、review代わりのPRは使わない。
- branch名・commit・PRの有無は、Product / Issue / Gateの状態を表さない。状態は、それぞれの正本文書とEvidenceで管理する。
- GitHub Actionsは月間quotaの都合で使えないことがあるため、検証の正本はlocalのhealth checkに置く。手動診断用のworkflowだけを残す。

過去のbranch / PR運用（workstream単位のremote branch、PR review、branch監査tool）は、単独開発では手続きの費用に見合わないため廃止しました。当時の内容はGit historyから辿れます。

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
