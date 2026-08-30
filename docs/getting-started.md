# Getting Started

この文書は、TEIを実際にcloneして動かしたい開発者向けの入口です。まず考え方から知りたい場合は、[はじめてTEIを知る方へ](for-readers.md) を先に読む方が自然です。

## 前提

現段階のTEIはGoで実装されたarchitecture probeです。完成したGUI製品の操作手順ではなく、設計境界を実行可能なコードで確認するための手順です。

Go toolchainがあれば主要部分を試せます。

```bash
go version
```

## 1. テスト

```bash
go test ./...
```

すべてのpackage testが通ることを確認します。

## 2. Reference Workspaceをvalidate

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

Referenceには過去Releaseを意図的に残しているため、historical contractについてWarningが表示されます。最後が`ok`で終わればValidation自体は成功です。

Source Indexで位置を解決できるfindingは、`path:line:column`を伴って表示されます。

## 3. 正規化Worldを見る

```bash
go run ./cmd/tei world \
  -workspace reference/workspace > /tmp/tei-world.json
```

ここにはSemantic、Application、Database、Environment、Release、Plugin等のnormalized modelが含まれます。

`reference/world.json` は同じReference Workspaceから再生成できるgolden snapshotで、CIでも一致を確認しています。

重要なのは、Canonical Worldにauthoring file pathを混ぜないことです。Sourceの物理位置は次のSource Indexから確認します。

## 4. Source Indexでsourceへ戻る

Source Index全体を見るには次を実行します。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace
```

特定のsemantic subjectだけを確認する場合は、kindとIDを指定します。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

Reference Queryでは、同じ`query:order_entry.recent_orders`に対して次の二つが表示されます。

- `definition`: `applications/order-entry/queries/recent_orders.tei`
- `authoring`: `applications/order-entry/queries/recent_orders.sql`

`.tei`はQuery declaration、`.sql`はauthoritative SQL bodyです。Artifactにはcontent fingerprintも含まれます。

`recent_orders.tei`自体はSQL本文を埋め込まず、次の形でnative sourceを参照しています。

```tei
query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}
```

SQL fileを別pathへ移し、この参照だけを更新した場合、Canonical Query IDとnormalized Worldは変わらず、Source Index locatorだけが変わることをテストしています。

この境界の詳細は [Conformance — M0.23 Source-to-Canonical Traceability](conformance/source-traceability-m0.23.md) を参照してください。

## 5. Plugin LockとRuntime Plan

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace > /tmp/tei-plugin-lock.json

go run ./cmd/tei plugin-runtime-plan \
  -workspace reference/workspace \
  -environment production \
  -plugin-lock /tmp/tei-plugin-lock.json
```

EnvironmentがどのPlugin / version / capabilityを使い、どのPermission / Resource Binding / Sandbox policyを要求するか確認できます。

Repositoryに保存したReference Plugin Lockも、CIでWorkspaceから再生成して一致を確認しています。

## 6. Deploymentの参照経路

Reference Workspaceと観測fixtureからDurable Deployment Runを再生成できます。

```bash
go run ./cmd/tei deployment-start \
  -workspace reference/workspace \
  -solution order_rollout \
  -version 2026.8.27 \
  -deployment reference/observed/production.deployment-state.json \
  -run-id order-rollout-prod-m0.21 \
  -plugin-lock reference/workspace/.tei/plugin-lock.json
```

保存済みの `reference/order_rollout.durable-run.m0.21.start.json` はこの入力から再生成可能な参照fixtureです。詳しくは [reference/README.md](../reference/README.md) を参照してください。

## 7. 実process Pluginの参照例

M0.24では、Reference Pluginを実際のOS child processとして起動できます。まずhelper binaryをbuildします。

```bash
go build -o /tmp/tei-reference-process-plugin ./tools/reference-process-plugin
```

次に、既存`DeploymentExecutor` adapterからstdio processへcommandを渡します。

```bash
go run ./examples/process-plugin \
  -plugin /tmp/tei-reference-process-plugin
```

出力の`outcome`は`accepted`、Plugin Invocationの`boundary`は`process`になります。`accepted`はexecutor acknowledgementであり、外部stateの完成証明ではありません。

この例はone-shot local processです。PIDをDurable Runtime Handleとして扱わず、Host restart / cross-host reattachを実現したとはみなしません。詳細は [M0.24実process Conformance](conformance/process-runtime-m0.24.md) を参照してください。

## 8. Runtime reattach example

```bash
go run ./examples/runtime-reattach
```

この例では、Host Aが監督していた外部runtimeをHost Bが新しいfencing tokenで引き継ぎます。旧Hostのtokenはruntime側で拒否されます。

重要なのは、runtime handleが`missing`だった場合に自動再実行しないことです。外部副作用の有無を確定できないため、Recoveryへ送ります。

## 9. コードを読む順番

初めて実装を読む場合は、次の順が比較的追いやすいです。

1. `reference/workspace/tei.workspace.yaml`
2. `reference/workspace/semantic/`
3. `reference/workspace/applications/`
4. `reference/workspace/databases/`
5. `reference/workspace/environments/`
6. `reference/workspace/releases/`
7. `internal/ir/`, `internal/workspace/`, `internal/sourceindex/`, `internal/validation/`
8. `internal/database/`, `internal/query/`
9. `internal/deployment/`
10. `internal/plugin/runtime/`, `internal/resource/`

Source / meaning境界を読む場合は、[Conformance — M0.23 Source-to-Canonical Traceability](conformance/source-traceability-m0.23.md) がコードとテストの対応表として使えます。

Plugin / Runtimeの安全境界を読む場合は、[Conformance — M0.22.1](conformance/plugin-runtime-m0.22.1.md) を参照してください。

## 注意

現段階のReference adapterは設計契約を検証するためのものです。production用Vault / Kubernetes / etcd adapter等としてそのまま利用することは想定していません。
