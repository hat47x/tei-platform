# ピン留めしない3経路は、何のために Runtime を直接呼ぶのか（2026-11-08）

- 対象: [Runtimeへの経路は`servicehost`だけではない](../RUNTIME_CARRIER_ALTERNATE_PATHS_2026-11-08/README.md)と[構成Kernel移行Gate](../../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)
- 位置づけ: **静的な読み取りの記録**。3経路の是非は示しません。
- 計測した版: `raw/head.txt`（`origin/main` `6177f9f0` の作業ツリー）。

前の記録は、`InvokeWithProvenance` を直接呼ぶ非試験の経路が3件あることを示しました。この記録は、**その3件が何のために呼ぶのか**を、呼出し箇所の文脈と語彙から読みます。

## 1. 3つの経路

| ファイル | パッケージ | 何をするか（読み取れた範囲） |
| --- | --- | --- |
| `internal/plugin/runtime/adapters.go` | `runtime`（製品パッケージ内） | `DeploymentExecutor.Execute` が**配備実行**を載せる。`Supports(c)` で能力と action を先に確認してから呼ぶ |
| `tools/reference-order-rollout-host-continuity/host.go` | `orderrollouthost`（参照実装） | Host が**`migration.atlas` の配備を実行**する。直前で `pruntime.Registry` を自前で組み立て、Sandbox・Verifier・Resources・SupervisorOwnership を渡す |
| `tools/reference-plugin-capability-contract/runtime_authority_bridge.go` | `main`（参照実装の検証用） | **Contract Query** を実行する。`RuntimeAuthorizedContractQueryExecutor` が Provider の Scope・Contract・ArtifactDigest を構築時に検証する |

## 2. Host世代のピン留め語彙の有無

3ファイルが「Host・generation・lease・Coordinator・pin」の語を含むかを機械的に見ました。

```text
internal/plugin/runtime/adapters.go                                     Host=- generation=- lease=Y Coordinator=- pin=-
tools/reference-order-rollout-host-continuity/host.go                   Host=Y generation=- lease=Y Coordinator=- pin=Y
tools/reference-plugin-capability-contract/runtime_authority_bridge.go  Host=- generation=- lease=- Coordinator=- pin=-
```

- **`generation` と `Coordinator` は3件とも含みません。** どの経路も `composition.Coordinator` を使っていません。
- `host.go` の `pin` は `pinned` の語（「stand-in は pinned な executable ではない」という**注記**）で、世代のピン留めではありません。
- `adapters.go` の `lease` は `PermissionLease`・`ResourceLease*`（**資源の貸与**）で、世代のリースではありません。

## 3. この観測が示すこと

- **3経路とも `composition.Coordinator` を使っていません。** したがって Host世代のピン留め（`Acquire` → `Binding` の照合）を受けないのは、**設計上そう組まれていない**ためです。
- 3経路はそれぞれ別の用途です。**製品パッケージ内の配備実行アダプタ**、**参照実装の配備実行**、**参照実装の Contract Query** で、互いに同じ用途ではありません。
- `adapters.go` は製品パッケージ（`internal/plugin/runtime`）の中にあり、`DeploymentExecutor.Execute` として配備の実行を載せます。**この1件だけが製品コードの中の経路**です。
- Gateの「Cの試験成功を実HostからのService Carrier呼出しと読み替えない」は、**この3経路が Coordinator を使わない**ことと整合します。Cは別の入口だからです。

## 4. この記録が示さないこと

- **3経路がピン留めを必要としないこと**。Coordinator を使っていないことは、必要ないことの証明ではありません。
- **`adapters.go` の経路が正しいこと**。配備実行が世代を固定すべきかは、この記録では判断していません。
- **語彙の有無と設計意図の一致**。語を含むかどうかは機械的な指標で、コメントの意味を読み取ったものではありません。
- **実行時の挙動**。静的な読み取りのみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（3経路の呼出し箇所・直前の文脈・語彙の有無） |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（0行） |
| `raw/sites.txt` | 各経路の呼出し箇所と直前8行 |
| `raw/vocabulary.txt` | 語彙の有無 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `38c55510e9a74c24c06a1dab396a7107399369b728f521bd9d60b20dba99541a` | 2467 |
| `raw/head.txt` | `76178cb7b94981f121dedbd53eb45fb0600a1292a065520341a5507ab43f12c3` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` | 0 |
| `raw/sites.txt` | `6ae1e815598b63b73f2fe7813e6febceb529f5429b7564e2093959814d1e87a3` | 2040 |
| `raw/vocabulary.txt` | `704090929577ab13a3d6874dc30e1101fad54d2ee6a7bebafbe8ef36cc890b18` | 417 |

`raw/sites.txt` の「直前8行」は**機械的に切り出した範囲**で、コメントブロックの境界ではありません。`adapters.go` では直前が関数の途中になるため、文脈として読める範囲が限られます。
