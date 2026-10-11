# TEIを動かす

この文書では、TEIを取得して動作を確かめる手順を説明します。設計の考え方を先に知りたい場合は、[はじめてTEIを知る方へ](for-readers.md)を参照してください。

## 前提

現在のTEIは、設計上の境界をGoのコードで検証する段階にあります。ここで説明するのは完成したGUI製品の操作ではなく、参照実装を使った動作確認です。

主要な確認にはGoの開発環境が必要です。

```bash
go version
```

## 最初の小さなWorkspaceから始める

`reference/workspace`を読み解く前に、独立した最小Workspaceで「業務の意味を一か所変更し、ValidationとSource Indexで確認する」操作を試せます。

[First Model Change — 最初の意味変更を試す](../examples/first-workspace/README.md) を参照してください。Go toolchainだけで動きます。これはModelの検証例であり、生成画面やCRUDを利用できる状態ではありません。

## CLIを生成AIや自動化から使う

当面は`tei` CLIを操作の入口として優先します。現在のread-onlyな主要コマンドは、次のJSON一覧から確認できます。

```bash
go run ./cmd/tei commands
```

モデルや定義の整合性を調べる場合、生成AIは次のようにJSON診断を受け取れます。

```bash
go run ./cmd/tei validate \\
  -workspace reference/workspace \\
  -json > /tmp/tei-validation.json
```

結果には`schemaVersion`、`ok`、`diagnostics`が含まれます。構文エラーでは元ファイルの位置を、意味上の検証では可能な範囲でSubjectとSource Indexの位置を返します。従来の`validate -workspace`はテキスト形式のままです。

終了コードは`0`が検証成功（警告を含む）、`1`が検証不成立、`2`が入力・読込みなどの実行失敗です。`-json`で返す結果を利用するときも終了コードを確認してください。`commands`はすべてのコマンドやPlugin機能を列挙する完全な登録簿ではありません。詳細とWeb／デスクトップへの接続方針は[CLI優先のインターフェース設計](../plan/design/CLI_FIRST_INTERFACE_DESIGN.md)を参照してください。

正規化済みの業務モデルから特定の要素だけを読む場合、`inspect`を利用できます。初期対応は`semantic-data`、`semantic-relation`、`application`、`application-pattern`、`query`の5種類です。kindとCanonical IDを完全一致で指定します。

```bash
go run ./cmd/tei inspect \
  -workspace reference/workspace \
  -subject-kind semantic-data \
  -subject-id sales.order
```

`inspect`は`tei.cli.inspect/v0`形式のJSONを返します。成功時の`result.canonical`は既存のNormalized Worldから選択した要素そのもの、`result.sources`は既存Source Indexの参照とArtifact fingerprintです。`worldFingerprint`は読取り時点のCanonical Worldの内容を識別するためのハッシュであり、競合更新を防ぐrevisionや変更の受理証跡ではありません。`sourceStatus=not-indexed`はSource Indexに参照がないことを意味し、Canonicalの不存在とは区別します。

業務モデル要素の参照先と参照元を調べるには、読み取り専用の`trace`を使用します。`inspect`と共通のkind/ID（例：`semantic-relation: sales.order.customer`）から意味情報と参照関係の両方を取得できます。初期段階では既存のDerived Indexが表現できる宣言済みの直接参照に限定しています。

```bash
go run ./cmd/tei trace \
  -workspace reference/workspace \
  -scope reference-observation \
  -subject-kind semantic-relation \
  -subject-id sales.order.customer \
  -direction both -mode summary

go run ./cmd/tei trace \
  -workspace reference/workspace \
  -scope reference-observation \
  -subject-kind semantic-relation \
  -subject-id sales.order.customer \
  -direction both -mode full
```

`-scope`は呼出し元が指定する**候補グラフの文脈識別子**です。Workspace名、業務上の所属namespace、受理状態や認可scopeを自動的に証明する値ではありません。`-mode summary`は件数・未解決件数を取得し、`-mode full`では宣言された直接の参照を省略せず返します。大規模な結果を分割する場合のみ`-limit 100`のように指定し、`result.trace.nextOffset`と**`result.snapshotFingerprint`（Workspace全体の継続用トークン）**を次回の`-offset`、`-expected-fingerprint`へ渡してください。`result.trace.snapshotFingerprint`はGraphと関係規則・選択条件だけの識別子であり、CLIの継続には使いません。入力Graph、対象や方向、正規化World、Source Index（元ファイルの内容や場所を含む）が変化している場合、`STALE_SNAPSHOT`で継続を拒否します。

結果の`worldFingerprint`と`sourceIndexFingerprint`は、`inspect`が返す同名のフィールドと比較できます。前者が一致しても、Source移動・注記変更などがあれば後者は変わります。結果の`source`は選択要素の元定義、`relatedSources`はFull出力のそのページに登場した参照先・参照元の出所候補を示します。`projectionUnresolved`はDerived Indexの投影時点で解決できなかった参照です。関係の`resolved`は宣言の解決に成功したことのみを意味し、実行の成功・人間による承認・書込み許可を意味しません。Source Indexと原ファイルの間で生じる同時更新は、トランザクション的に固定できていません。

成功時の終了コードは`0`、不存在・重複宣言は`1`、引数不正・未対応kind・Workspace読込み失敗などは`2`です。失敗時も可能な範囲で`ok=false`と`error.code`を持つJSONを出力します。現時点で対応するのは宣言済みの直接参照であり、複数段の依存探索、全種類の要素取得、受理済み業務フローとの完全な横断参照には対応しません。

## 1. テスト

```bash
go test ./...
```

すべてのパッケージのテストが成功することを確認します。

## 2. 参照用ワークスペースを検証する

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

参照用データには過去のリリースを意図的に残しているため、過去の契約に関する警告が表示されます。最後が`ok`で終われば検証は成功です。

Source Indexで位置を解決できるfindingは、`path:line:column`を伴って表示されます。

## 3. 正規化したモデル（World）を見る

```bash
go run ./cmd/tei world \
  -workspace reference/workspace > /tmp/tei-world.json
```

ここにはSemantic、Application、Database、Environment、Release、Plugin等のnormalized modelが含まれます。

`reference/world.json` は同じReference Workspaceから再生成できるgolden snapshotで、CIでも一致を確認しています。

重要なのは、Canonical Worldにauthoring file pathを混ぜないことです。Sourceの物理位置は次のSource Indexから確認します。

## 4. Source Indexで定義元を確認する

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

## 7. 外部プロセスとして動くプラグインの参照例

M0.24では、Reference Pluginを実際のOS child processとして起動できます。まずhelper binaryをbuildします。

```bash
go build -o /tmp/tei-reference-process-plugin ./tools/reference-process-plugin
```

次に、既存`DeploymentExecutor` adapterからstdio processへcommandを渡します。

```bash
go run ./examples/process-plugin \
  -plugin /tmp/tei-reference-process-plugin
```

出力の`outcome`は`accepted`、Plugin Invocationの`boundary`は`process`になります。Reference pluginには`--tei-reference-marker=argv value preserved`という空白を含む一つのargv elementも渡され、子process側がその境界を確認してから成功Responseを返します。`accepted`はexecutor acknowledgementであり、外部stateの完成証明ではありません。

この例はone-shot local processです。PIDをDurable Runtime Handleとして扱わず、Host restart / cross-host reattachを実現したとはみなしません。実process境界は [M0.24実process Conformance](conformance/process-runtime-m0.24.md)、executable / argv境界は [M0.25 Conformance](conformance/process-command-argv-m0.25.md) を参照してください。

## 8. actual PostgreSQL Desired / Actual / Diff

PostgreSQLを用意し、接続文字列をHost-side environment variableとして渡すと、Reference integrationを実行できます。

```bash
TEI_TEST_POSTGRES_DSN='postgres://user:password@127.0.0.1:5432/database?sslmode=disable' \
  go run ./tools/reference-postgres-integration
```

このharnessは専用fixture schemaを準備し、Persistence MappingからDesiredを生成したうえでactual catalogをintrospectします。baselineでは差分0件を要求し、その後actual側だけにcolumnを追加してdrift検出を確認します。fixture DDLはintegration test harnessであり、TEIのautomatic schema apply機能ではありません。

詳細は [M0.26 Conformance](conformance/postgres-actual-schema-m0.26.md) を参照してください。

## 9. 実行中のプロセスを別のHostから引き継ぐ例

```bash
go run ./examples/runtime-reattach
```

この例では、Host Aが監督していた外部runtimeをHost Bが新しいfencing tokenで引き継ぎます。旧Hostのtokenはruntime側で拒否されます。

重要なのは、runtime handleが`missing`だった場合に自動再実行しないことです。外部副作用の有無を確定できないため、Recoveryへ送ります。

## 10. コードを読む順番

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
