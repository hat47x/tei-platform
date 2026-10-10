# 生成時と実行時のProjection境界（読取り専用Reference）

このパッケージは、第一Web Targetの `buildWebTargetProjection` が生成し得る**派生JSON manifest**を、実行時にWorkspace Loader・Driver Resolver・DSL Parserへ再接続せず保持・配信するための、最小の契約候補です。採用済みのBusiness Runtime、配布Builder、Plugin Lock、認可・Business Result・Observationの実装ではありません。

生成時（既存の権限を保持する側）は、Accepted Handover `B / R` とTarget build `T`、Application IDに対応するProjection bytesを生成する。ビルドと検証の責任を持つ呼出し側は、その**独立に確定した**4つのIDとraw JSONのSHA-256を固定する。実行側の `Accept(raw, expected)` は、文字列IDとraw bytesの完全一致を検査し、opaque JSON bytesをコピーして保持する。manifestの`patterns`や`unresolved`を解決し直したり、意味を付け替えたりしない。

受理した `Snapshot.Handler()` は `GET /manifest.json` と `GET /healthz` のみを提供する。既存の `/scenario/read`、`/scenario/submit`、`/scenario/customer-lookup` 等は未実装で404となり、未実装の業務処理・認証・認可を成功したものとして表示しない。HTTP Serverのlisten/TLS/identity/authorization、実ビルドへの組込みは呼出し側の責任であり、このReferenceからProduction公開してはならない。

`ManifestSHA256`は呼出し側が**独立に確定した値**であることが前提で、未信頼のmanifest自身から期待digestを計算して与えてもprovenanceの証明にならない。SHA-256だけでは、Source revision・Builder・署名・権限・Secret不在・機能同等性を保証しない。また現在のReferenceはtop-levelの4つのIDの構文と一意性だけを要求し、追加のProjection schemaを確定しない。

```sh
go test -count=1 ./tools/reference-runtime-snapshot
go vet ./tools/reference-runtime-snapshot
```

`closure_test.go`は実`go list -deps -json`を実行し、このパッケージから他のTEI main-module packageが推移的にimportされないことを確認します。したがって設計用Workspace LoaderやDriver Resolverがこの読取り境界へ紛れた場合は局所テストが失敗します。これは**Goソースの依存閉包**の検査であり、実行バイナリやBrowser/Serverの依存除去・安全性を証明するものではありません。

上記はパッケージの隔離テストであり、第一Web TargetのFunctional/Visual/Gate Cや、#259 E1のBrowser+API分離に必要な認可・Transport・実配備物の証拠ではない。既存のGate C Target Sourceを変更せず、将来のアプリ固有compile-time entryでこのReferenceを使用するには、同一Handover meaning、データ/Provider、認可とBusiness Result、実行時の操作を別途実証する必要がある。
