# ADR-0003: Reference Workspaceのgolden出力と観測fixtureをRepositoryに保持する

- 状態: 採用
- 日付: 2026-08-30
- 関連Issue: #2, #3

## 背景

M0.22.1の公開準備では、ローカルでは全テストが成功していた一方、最初にGitHubへ正規化したソースだけでは一部のテストが失敗しました。原因はコードではなく、Reference Workspaceに付随する `reference/world.json` と `reference/observed/*.json` が公開対象から漏れていたことでした。

Issue #3で改めて監査したところ、同じ問題が実行例にも一つ残っていました。`tools/reference-deployment-attempt` は、既定入力として `reference/order_rollout.durable-run.m0.21.start.json` を読む設計ですが、このファイルがM0.22.1の公開ツリーから漏れていました。そのため、コード自体はビルド・テストできても、clone直後のRepositoryでは実行例を既定値のまま動かせない状態になっていました。

一方で、Reference Workspaceから生成できるものをすべて保存すればよいわけでもありません。何をgoldenとして保持し、何をシナリオfixtureとして手書きで固定し、どこまでCIで再生成を確認するのかを区別する必要があります。

## 判断

Reference Workspaceに加えて、テストや実行例の共有基準として必要なgolden出力と観測fixtureをRepositoryへ保持します。

現時点では、次をReference artifactとして扱います。

- `reference/world.json`
- `reference/workspace/.tei/plugin-lock.json`
- `reference/observed/production.deployment-state.json`
- `reference/observed/production.canary.deployment-state.json`
- `reference/order_rollout.durable-run.m0.21.start.json`

このうち、World、Plugin Lock、Durable Runの開始状態は、他のRepository上の入力から決定的に再生成できるgoldenです。保存するだけではなく、常設CIで再生成した結果と比較し、意図しない差分が生じた場合は失敗させます。

observed state 2件は、外部環境からその都度取得する最新値ではありません。Deployment、Validation、Durable Runの境界条件を再現するために、時点と状態を固定したシナリオfixtureです。`observedAt`は現在時刻や実環境の鮮度を表すものではなく、そのシナリオ内の時刻として扱います。

各artifactの役割と再生成方法は `reference/README.md` にまとめます。

## Goldenとして保持するもの

生成可能なファイルをgoldenとしてRepositoryへ置くのは、その出力の変化自体をレビュー対象にしたい場合です。

`reference/world.json` は、Reference Workspaceを正規化したWorldの基準です。Canonical Modelやloaderの変更によってWorldが変わったとき、その差分を「生成し直したから」で済ませず、意味の変化として確認できるようにします。

`reference/workspace/.tei/plugin-lock.json` は、同じWorkspaceからどのPlugin / Bundleが選ばれ、どのruntime policyへ解決されたかを固定します。実行例が別々の解決結果を暗黙に使わないための共有基準でもあります。

`reference/order_rollout.durable-run.m0.21.start.json` は、基準となるobserved state、Solution Release、Plugin Lockから生成されるDeploymentの開始状態です。現在も `tools/reference-deployment-attempt` の入力として使うため、実行例と一緒に保持します。

ファイル名の`m0.21`は、その実行例がM0.21で導入した縦切りを再現していることを示す歴史的な識別子です。Repository全体の現在地はM0.22.1であり、名称だけを現在のマイルストーンへ機械的に書き換えません。

## 観測fixtureとして保持するもの

`reference/observed/production.deployment-state.json` は、通常のrollout開始前の基準状態を表します。`order_entry` 3.7.0と`sales_core` 4.0.0が稼働している状態から、次の展開を組み立てるために使います。

`reference/observed/production.canary.deployment-state.json` は、`order_entry` 3.7.0と3.8.0が並行して稼働し、trafficが70% / 30%に分かれているcanary途中の状態を表します。

これらは現実のproduction環境を模した参照シナリオであり、Repositoryを開いた時点の「現在の本番状態」ではありません。この区別が曖昧になる命名や説明は避けます。

## この方式を採る理由

Reference Workspaceだけを保持し、生成結果を毎回そのまま受け入れる方式では、意味モデルや解決規則の意図しない変化を見落とす可能性があります。反対に、CLIが生成した中間物をすべて保存すると、Reference treeが過去の出力置き場になり、何が基準なのか分からなくなります。

そこで、テストや実行例が安定して依存する共有基準だけをReference artifactとして選びます。生成可能なgoldenはCIで再現性を検査し、外部状態を模したfixtureは、どのシナリオを表すのかを文章で説明します。

この方式なら、別の生成AIや開発者が過去の会話やローカル未追跡ファイルを持っていなくても、cloneしたRepositoryだけで同じ前提を再現できます。

## 影響

Canonical Model、Plugin解決、Deployment planなどを正当に変更した場合は、対応するgoldenの更新も必要になります。その差分は単なる生成物の更新ではなく、入力や設計判断の変化と一緒にレビューします。

Reference artifactを追加するときも、生成できることだけを理由にはしません。何を再現するのか、どのテストや実行例が依存するのか、生成物なのか手書きfixtureなのかを説明できるものに限ります。

一方、実行時刻や一時的なopaque handleなど、実行ごとに変わる値を含む出力までgolden化する必要はありません。たとえば `tools/reference-deployment-attempt` の出力は、完全一致ではなく、CIで実行が正常に完了することを確認します。
