# Getting Started

## 前提

現段階のTEIはGoで実装されたarchitecture probeです。まずはGo toolchainがあれば主要部分を試せます。

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

## 3. 正規化Worldを見る

```bash
go run ./cmd/tei world \
  -workspace reference/workspace > /tmp/tei-world.json
```

ここにはSemantic、Application、Database、Environment、Release、Plugin等のnormalized modelが含まれます。

## 4. Plugin LockとRuntime Plan

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace > /tmp/tei-plugin-lock.json

go run ./cmd/tei plugin-runtime-plan \
  -workspace reference/workspace \
  -environment production \
  -plugin-lock /tmp/tei-plugin-lock.json
```

EnvironmentがどのPlugin/version/capabilityを使い、どのPermission / Resource Binding / Sandbox policyを要求するか確認できます。

## 5. Runtime reattach example

```bash
go run ./examples/runtime-reattach
```

この例では、Host Aが監督していた外部runtimeをHost Bが新しいfencing tokenで引き継ぎます。旧Hostのtokenはruntime側で拒否されます。

重要なのは、runtime handleが`missing`だった場合に自動再実行しないことです。外部副作用の有無を確定できないため、Recoveryへ送ります。

## 6. コードを読む順番

初めて読む場合は、次の順が比較的追いやすいです。

1. `reference/workspace/tei.workspace.yaml`
2. `reference/workspace/semantic/`
3. `reference/workspace/applications/`
4. `reference/workspace/databases/`
5. `reference/workspace/environments/`
6. `reference/workspace/releases/`
7. `internal/ir/`, `internal/workspace/`, `internal/validation/`
8. `internal/database/`, `internal/query/`
9. `internal/deployment/`
10. `internal/plugin/runtime/`, `internal/resource/`

## 注意

現段階のReference adapterは設計契約を検証するためのものです。production用Vault/Kubernetes/etcd adapter等としてそのまま利用することは想定していません。
