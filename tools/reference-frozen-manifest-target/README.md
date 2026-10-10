# 固定Projectionの最小Go起動点（Reference）

このツールは、[#259](https://github.com/hat47x/tei-platform/issues/259)の **E0→E1（設計時と実行時の依存分離）** を観測するための、独立したGo `package main`です。既存の[第一Web Target](../reference-first-web-target/README.md)は変更せず、[固定Projectionの読取りReference](../reference-runtime-snapshot/README.md)だけをimportします。

**実際にできること**は、生成時に独立して確定したApplication ID、Handover B/R、Target Tとraw manifestのSHA-256を指定して、そのJSONをローカルで読取り専用配信することに限ります。Workspace Loader、Driver Resolver、DSL Parser、Provider選択を起動し直しません。実アプリの業務処理や認証・認可は実装しません。

## 起動

すでに受理済みのProjection JSONを、**書込み可能な作業リポジトリの外**に保存します。期待IDとダイジェストは、そのファイルの内容を無条件に信用して計算するのではなく、独立に検証した引継ぎ・ビルド記録から与えてください。

```sh
go run ./tools/reference-frozen-manifest-target \
  -manifest /tmp/tei-accepted-manifest.json \
  -application order_entry \
  -baseline-id B \
  -baseline-revision R \
  -target-revision T \
  -sha256 <independently-pinned-64-hex-digest> \
  -port 8091
```

- HTTPのbind先は**127.0.0.1固定**で、ポートだけ変更できます。TLS・ログイン・Authorizationを提供していないため公開環境へ配置しないでください。
- `GET /manifest.json`は一致したraw bytesを返し、`GET /healthz`は204を返します。業務APIは404です。ヘルス応答は**Business Runtimeの健全性を証明しません**。
- 不完全なID、不一致のdigest、8 MiBを超えるファイル、JSON構文／ID不一致なら**listen前に失敗**します。ファイルを読込んだ後はSnapshot内でbytesをコピーするため、配信する値は固定されます。
- `-application`などは**実行時の照合条件**です。値を切り替えてもビルド時にアプリ別のGo binaryが生成されたことにはなりません。manifestだけの静的入口という限定を守ります。

## 物理依存の検査

```sh
go test -count=1 ./tools/reference-frozen-manifest-target
go vet ./tools/reference-frozen-manifest-target

go run ./tools/reference-target-package-evidence \
  -entry ./tools/reference-frozen-manifest-target \
  > /tmp/tei-frozen-manifest-e0.json
```

`closure_test.go`はGoの実import graphを調査し、TEIメインモジュールの依存を当該CLIと`tei/tools/reference-runtime-snapshot`の**二つだけ**に限定します。将来、Authoring用Workspace/Driver/DSLが混入した場合はテスト失敗とします。

**未証明**：実Web TargetのProjection生成との接続、個別ApplicationのBusiness Result・Observation・Authorization、ブラウザ資産、実ホスト・Secret取扱い、link時の不要コード除去、Artifact digest／署名、実運用への適合性。これらは別の受入条件で判定します。本CLIはProduction Targetや既存Gate Cを置き換えません。
