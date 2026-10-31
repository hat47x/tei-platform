# TEIを動かす

この文書では、TEIを取得して動作を確認するまでの手順を説明します。設計の考え方から知りたい場合は、[はじめてTEIを知る方へ](for-readers.md)を参照してください。

## 前提

TEIは現在、設計上の境界をGoのコードで検証している段階です。この文書では完成した画面製品の操作ではなく、参照実装を使った動作確認を説明します。

主要な手順を試すには、Goの開発環境が必要です。

```bash
go version
```

## 最初の小さな作業領域から始める

大きな参照用作業領域である`reference/workspace`を調べる前に、独立した小さな作業領域で、業務上の意味を一か所変更し、検証と入力元索引で結果を確認できます。

手順は[最初の意味変更を試す](../examples/first-workspace/README.md)に記載しています。Goの開発環境だけで実行できます。ただし、これはモデルの検証例であり、画面の生成やデータの登録・照会・更新・削除はできません。

## コマンドラインを生成AIや自動化から使う

当面はコマンドラインから`tei`を操作する方法を基本とします。現在使える主な読み取り専用コマンドは、次のコマンドが返すJSON形式の一覧で確認できます。

```bash
go run ./cmd/tei commands
```

結果を記録して後から再現できるようにするには、使った`tei`の識別も一緒に残します。`go build`で作った実行ファイルに対して次を実行すると、埋め込まれたコミット（`vcsRevision`）、未コミット変更の有無（`vcsModified`）、Goのバージョン、OSとCPUの種類が返ります。

```bash
go build -o tei ./cmd/tei
./tei version -json
```

`go run`や`-buildvcs=false`でbuildした場合はコミットが埋め込まれないため、`source`が`unavailable`になります。識別は再現の手掛かりであり、配布の承認や署名ではありません。

モデルや定義の整合性を調べる場合、生成AIなどの呼出し元は次のようにJSON形式の診断結果を取得できます。

```bash
go run ./cmd/tei validate \
  -workspace reference/workspace \
  -json > /tmp/tei-validation.json
```

結果には`schemaVersion`、`ok`、`diagnostics`が含まれます。構文の誤りも読込みの失敗もなく読めた場合は、`worldFingerprint`と`sourceIndexFingerprint`も返します。値は`model catalog`・`inspect`・`trace`と同じ計算で、同じファイル内容なら一致し、内容が変われば変わります。どの版のWorkspaceを検証した結果かを後から照合する証拠であり、内容が正しいことや受理されたことの証拠ではありません。構文に誤りがある場合と読込みに失敗した場合は、版が存在しないため両項目を省きます。構文に誤りがあれば元ファイルの位置を示し、意味の検証では可能な範囲で対象要素と入力元索引の位置を返します。`-json`を指定しない従来の`validate -workspace`は、テキスト形式で結果を返します。警告だけの場合、テキストは警告行を出力した後に`ok`を出力し、終了コードは`0`です。標準出力全体が`ok`と一致することを成功条件にすると、警告付きの成功を失敗と誤判定します。成否は終了コード、または`-json`の`ok`で判定してください。`ok`が`true`でも`diagnostics`に警告が残ることがあります。

能力の入出力については、宣言した型の実在を検査しますが、ある能力の出力と次の能力の入力が接続できることは検査しません（[参照例](../reference/product-contracts/README.md#境界)）。

意味要素の重複診断では、同じ識別子を持つローカルの宣言位置を候補として複数返すことがあります。候補の順序はファイルと位置による表示順であり、どの宣言を採用・拒否すべきかは示しません。importや標準定義など位置を網羅できない場合は候補を返しません。

版を固定する値は、checkoutの改行（CRLF/LF）の受け取り方にも依存します。`sourceIndexFingerprint`と`modelRevision`（`model catalog`が返す、管理対象Sourceの版。Shared Acceptanceが照合する`baseModelRevision`と同じ値）は、Sourceの生バイトをsha256で識別するため、行末だけが違うcheckoutでも別の版になります。`worldFingerprint`は正規化したモデルの内容を識別しますが、元定義に書かれた本文をそのまま載せる欄（移行のDDLや問い合わせのSQLなど）があると、その本文の改行も反映されます。2026-10-31にmainで実測した結果では、`reference/workspace`（`databases/sales-core/migrations/20260826_001/migration.tei`と`persistence/order.tei`がSQL本文を持つ）ではCRLFのcheckoutで`worldFingerprint`も変わり、その種の欄を持たない`examples/first-workspace`では`worldFingerprint`は変わらず`sourceIndexFingerprint`と`modelRevision`だけが変わりました。別の環境で作った版固定の値を照合するときは、比較する側のcheckoutの改行をそろえてください。リポジトリ内のWorkspaceの木（`reference/workspace`、`examples/`配下、`reference/product-contracts`ほか、`tei.workspace.yaml`のある8木）は`.gitattributes`で`-text`に固定してあり、新しいcheckoutでは改行が書き換わらない。`core.autocrlf`が有効な環境で作った古いcheckoutで該当pathが`modified`に見えるときは、差分を確認し、必要な変更を保存したうえで、意図したファイルだけを復元してください。`git restore`は未保存の変更を破棄します。版識別を改行正規化後のバイトで計算する契約へ変えるかどうかは引き続き未決であり、[#2399](https://github.com/hat47x/tei-platform/issues/2399)で扱います。

終了コードは`0`が検証成功（警告を含む）、`1`が検証不成立、`2`が入力・読込みなどの実行失敗です。`-json`で返す結果を利用するときも終了コードを確認してください。`commands`はすべてのコマンドや拡張機能を列挙する完全な登録簿ではありません。詳細とウェブやデスクトップへの接続方針は[コマンドライン優先のインターフェース設計](../plan/design/CLI_FIRST_INTERFACE_DESIGN.md)を参照してください。

作業領域のモデル要素を一覧で確認するには、`model catalog`を使います。返された`entries[].subject.kind`と`entries[].subject.id`をそのまま`inspect`へ渡せます。`ambiguous=true`の項目は重複宣言があるため、詳細取得を拒否します。

```bash
go run ./cmd/tei model catalog -workspace reference/workspace
go run ./cmd/tei model catalog -workspace reference/workspace -namespace sales
```

一覧と詳細は読取り専用です。元ファイルを変更する権限や、変更を受理した証拠は与えません。一覧の取得と詳細の取得は別の処理なので、同じ版かどうかは返された`worldFingerprint`と`sourceIndexFingerprint`を比較します。

正規化された業務モデルから特定の要素だけを確認するには、`inspect`を使います。現在対応するのは`semantic`、`semantic-type`、`semantic-data`、`semantic-attribute`、`semantic-relation`、`semantic-candidate-key`、`semantic-functional-dependency`、`semantic-capability`、`semantic-validation`、`application`、`application-pattern`、`query`の12種類です。対象の種類と正本の識別子を、完全一致で指定します。

```bash
go run ./cmd/tei inspect \
  -workspace reference/workspace \
  -subject-kind semantic-data \
  -subject-id sales.order
```

`inspect`は`tei.cli.inspect/v0`形式のJSONを返します。成功時の`result.canonical`は既存の正規化した全体モデルから選択した要素そのもの、`result.sources`は既存入力元索引の参照と成果物の照合値です。`worldFingerprint`は読取り時点の正本となる全体モデルの内容を識別するためのハッシュであり、競合更新を防ぐ版を示す値や変更の受理証跡ではありません。`sourceStatus=not-indexed`は入力元索引に参照がないことを意味し、正本となる定義の不存在とは区別します。

業務モデルの要素が何を参照し、何から参照されているかを調べるには、読み取り専用の`trace`を使います。`inspect`と同じ種類・識別子（例：`semantic-relation: sales.order.customer`）を指定して、意味の情報と参照関係を確認できます。現在扱えるのは、既存の派生索引で表現できる、宣言済みの直接参照だけです。

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

`-scope`には、呼出し元が**関係図を作る際の文脈を識別する値**を指定します。作業領域名や業務上の所属名前空間、受理の状態、認可の範囲を自動的に証明するものではありません。`-mode summary`は件数・未解決件数を取得し、`-mode full`では宣言された直接の参照を省略せず返します。大規模な結果を分割する場合のみ`-limit 100`のように指定し、`result.trace.nextOffset`と**`result.snapshotFingerprint`（作業領域全体の継続用トークン）**を次回の`-offset`、`-expected-fingerprint`へ渡してください。`result.trace.snapshotFingerprint`は関係図、関係規則、選択条件の照合値であり、コマンドラインで結果の取得を続けるためには使いません。入力した関係図、対象や方向、正規化した全体モデル、入力元索引（元ファイルの内容や場所を含む）が変化している場合、`STALE_SNAPSHOT`で継続を拒否します。

結果に含まれる`worldFingerprint`と`sourceIndexFingerprint`は、`inspect`が返す同名の項目と比較できます。意味の正本を識別する前者が同じでも、入力元ファイルの移動や注記の変更があれば、後者は変わります。結果の`source`は選択要素の元定義、`relatedSources`は全件出力のそのページに登場した参照先・参照元の情報源の候補を示します。`projectionUnresolved`は、派生索引を作った時点で解決できなかった参照を示します。関係の`resolved`は宣言の解決に成功したことのみを意味し、実行の成功・人間による承認・書込み許可を意味しません。入力元索引と原ファイルの間で生じる同時更新は、一つの処理として固定できていません。

成功時の終了コードは`0`、不存在・重複宣言は`1`、引数不正・未対応の種類・作業領域読込み失敗などは`2`です。失敗時も可能な範囲で`ok=false`と`error.code`を持つJSONを出力します。現時点で対応するのは宣言済みの直接参照であり、複数段の依存探索、すべての種類の要素取得、受理済み業務フローとの完全な横断参照には対応しません。

## 1. テスト

```bash
go test ./...
```

すべてのパッケージのテストが成功したか確認します。

## 2. 参照用ワークスペースを検証する

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

参照用データには過去の配布版を意図的に残しています。そのため、過去の契約に関する警告が表示されます。最後に`ok`と表示されれば、検証は成功です。

入力元索引で位置を解決できる診断結果には、`path:line:column`を伴って表示されます。

## 3. 正規化した全体モデルを見る

```bash
go run ./cmd/tei world \
  -workspace reference/workspace > /tmp/tei-world.json
```

この結果には、意味、アプリケーション、データベース、環境、配布版、拡張機能などの正規化したモデルが含まれます。

`reference/world.json`は、同じ参照用作業領域から再生成できる基準データです。自動検査でも、生成結果が一致することを確認しています。

正本となる全体モデルには、入力ファイルの場所を含めません。元ファイルの場所は、次に説明する入力元索引で確認します。

## 4. 入力元索引で定義元を確認する

入力元索引の全体を見るには、次のコマンドを実行します。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace
```

特定の意味上の対象だけを確認する場合は、種類と識別子を指定します。

```bash
go run ./cmd/tei source-index \
  -workspace reference/workspace \
  -subject-kind query \
  -subject-id order_entry.recent_orders
```

参照用問い合わせでは、同じ`query:order_entry.recent_orders`に対して次の二つが表示されます。

- `definition`: `applications/order-entry/queries/recent_orders.tei`
- `authoring`: `applications/order-entry/queries/recent_orders.sql`

`.tei`には問い合わせの宣言を、`.sql`には正本となるSQL本文を記述します。成果物には内容の照合値も含まれます。

`recent_orders.tei`自体はSQL本文を埋め込まず、次の形で元形式のソースを参照しています。

```tei
query order_entry.recent_orders {
    datasource sales
    sql postgres file "recent_orders.sql"
}
```

SQLファイルを移動し、参照先だけを更新した場合でも、正本となる問い合わせの識別子と正規化した全体モデルは変わらず、入力元索引の場所だけが変わることを試験しています。

この境界の詳細は [準拠確認 — M0.23 入力元から正本への追跡可能性](conformance/source-traceability-m0.23.md) を参照してください。

## 5. 拡張機能の固定情報と実行計画

```bash
go run ./cmd/tei plugin-lock \
  -workspace reference/workspace > /tmp/tei-plugin-lock.json

go run ./cmd/tei plugin-runtime-plan \
  -workspace reference/workspace \
  -environment production \
  -plugin-lock /tmp/tei-plugin-lock.json
```

この結果から、環境が利用するプラグインとその版、提供される機能、必要な権限、資源の対応付け、隔離に関する要求を確認できます。

リポジトリに保存した参照用プラグインの実装固定情報についても、作業領域から再生成して一致することを自動検査で確認しています。

## 6. 配備の参照経路

参照用作業領域と観測試験用データを使い、保存済みの配備実行記録を再生成できます。

```bash
go run ./cmd/tei deployment-start \
  -workspace reference/workspace \
  -solution order_rollout \
  -version 2026.8.27 \
  -deployment reference/observed/production.deployment-state.json \
  -run-id order-rollout-prod-m0.21 \
  -plugin-lock reference/workspace/.tei/plugin-lock.json
```

保存済みの `reference/order_rollout.durable-run.m0.21.start.json` はこの入力から再生成可能な参照試験用データです。詳しくは [reference/README.md](../reference/README.md) を参照してください。

## 7. 外部プロセスとして動くプラグインの参照例

M0.24では、参照用のプラグインをOS上の子プロセスとして起動できます。最初に補助プログラムを構築します。

```bash
go build -o /tmp/tei-reference-process-plugin ./tools/reference-process-plugin
```

次に、既存の`DeploymentExecutor`接続部から、標準入出力で通信するプロセスへ命令を渡します。

```bash
go run ./examples/process-plugin \
  -plugin /tmp/tei-reference-process-plugin
```

出力の`outcome`は`accepted`、拡張機能の呼び出しの`boundary`は`process`になります。参照用の拡張機能には`--tei-reference-marker=argv value preserved`という空白を含む一つの引数要素も渡され、子プロセス側がその境界を確認してから成功応答を返します。`accepted`は実行役の受け付け通知であり、外部状態が整ったことの証明ではありません。

この例では、ローカルのプロセスを一度だけ起動します。プロセスIDを実行状態の識別情報として永続化していないため、実行環境の再起動後や別の環境から再接続できることを実証したものではありません。実プロセス境界は [M0.24実プロセス 準拠確認](conformance/process-runtime-m0.24.md)、実行ファイルと引数の境界は [M0.25 準拠確認](conformance/process-command-argv-m0.25.md) を参照してください。

## 8. PostgreSQLで期待状態と実際の状態を比較する

PostgreSQLを用意して、接続文字列を実行環境の環境変数として渡すと、参照用の連携試験を実行できます。

```bash
TEI_TEST_POSTGRES_DSN='postgres://user:password@127.0.0.1:5432/database?sslmode=disable' \
  go run ./tools/reference-postgres-integration
```

この試験では専用の試験用構造を準備し、データ保存の対応付けから期待状態を生成してから、実際のデータベースの定義情報を読み取ります。基準状態では差分がない状態を要求し、その後実際の状態側だけに列を追加してずれ検出を確認します。試験用の構造定義は連携試験用の仕組みであり、TEIの自動的な構造変更機能ではありません。

詳細は [M0.26 準拠確認](conformance/postgres-actual-schema-m0.26.md) を参照してください。

## 9. 実行中のプロセスを別の実行環境から引き継ぐ例

```bash
go run ./examples/runtime-reattach
```

この例では、実行環境Aが管理していた処理を、実行環境Bが新しい識別値を使って引き継ぎます。Aが使っていた古い識別値による操作は、実行環境側で拒否されます。

重要なのは、実行状態の識別情報が`missing`だった場合に自動再実行しないことです。外部副作用の有無を確定できないため、復旧処理へ送ります。

## 10. コードを読む順番

実装を初めて読む場合は、次の順にたどると全体の関係を把握しやすくなります。

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

ソースと意味の境界を読む場合は、[準拠確認 — M0.23 入力元から正本への追跡可能性](conformance/source-traceability-m0.23.md) がコードとテストの対応表として使えます。

拡張機能と実行環境の安全境界を読む場合は、[準拠確認 — M0.22.1](conformance/plugin-runtime-m0.22.1.md) を参照してください。

## 注意

現段階の参照用接続部は、設計上の契約を検証するためのものです。本番で利用する場合は、Vault、Kubernetes、etcdなど、実際の運用環境に合わせた接続部が必要です。
