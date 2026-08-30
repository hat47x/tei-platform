# ADR-0003: Reference Workspaceのgolden出力と観測fixtureをRepositoryに保持する

- 状態: 採用
- 日付: 2026-08-30
- 関連Issue: #2, #3

## 背景

M0.22.1の公開準備では、ローカルでは全テストが成功していた一方、最初にGitHubへ正規化したソースだけでは一部のテストが失敗しました。原因はコードではなく、テストが参照する `reference/world.json` と `reference/observed/*.json` が公開対象から漏れていたことでした。

この経験から、Reference Workspaceだけを残せば十分なのか、そこから得られるgolden出力や観測fixtureもRepositoryへ置くべきかを明示する必要が生じました。

## 判断

Reference Workspaceに加えて、テストが仕様上の基準として参照するgolden出力と観測fixtureをRepositoryへ保持します。

現時点では、少なくとも次を対象とします。

- `reference/world.json`
- `reference/observed/production.deployment-state.json`
- `reference/observed/production.canary.deployment-state.json`

`reference/world.json` は、`reference/workspace` からCLIで再生成できるgolden snapshotとして扱います。常設CIでは再生成した結果とRepository上の `reference/world.json` を比較し、差が出た場合は失敗させます。

observed stateは、外部環境の実測値そのものではなく、Deployment / Durable Run / Validationの境界条件を再現するための参照fixtureです。したがって、テストが必要とする状態を明示的にRepositoryへ置き、ローカルの未追跡ファイルへ依存しないようにします。

## この方式を採る理由

生成できるものをすべてRepositoryへ保存しない考え方もあります。しかしgolden snapshotは、入力から導出した結果が意図せず変わったことを検出する役割を持ちます。Reference Workspaceだけを保持し、毎回生成結果をそのまま受け入れてしまうと、意味モデルの変化を見落とす可能性があります。

またobserved stateは、現実の外部環境がなくてもDeploymentやreconcileの条件を再現するために必要です。これを隠れたローカルfixtureとして持つより、Repository上で明示した方が再現性と説明可能性が高くなります。

## 影響

良い影響として、clone直後のRepositoryだけでテストを再現しやすくなり、別の生成AIやCI環境でも同じ前提を共有できます。golden worldの意図しない変化も検出できます。

一方で、Canonical Modelの正当な変更を行った場合は `reference/world.json` の更新も必要になります。その差分を単なる生成物の更新として流さず、「なぜworldが変わったのか」をレビューできる状態にします。

observed fixtureの内容や用途が分かりにくくならないよう、Issue #3で再現性と説明の監査を継続します。このADRは、fixtureの具体的な値を永久に固定するものではありません。
