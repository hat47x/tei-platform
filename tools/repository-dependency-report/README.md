# Repository direct import inventory

Goの実装パッケージ間の依存方向を、ソースコードから読み取り専用で確認するための開発用ツールです。

Repository rootで次を実行します。

```sh
go run ./tools/repository-dependency-report -root . > /tmp/tei-direct-import-graph.json
go test -count=1 ./tools/repository-dependency-report
```

JSONの`packages`は`internal/`配下のGoパッケージごとに、production Goファイル数、異なる直接依存先の数、直接依存元の数を出力します。`dependencies`には一意な`from → to`の組と、そのimportを書いた元ファイルを列挙します。ファイル順やmapの列挙順に結果が左右されないよう、出力をソートします。実行時刻などの不安定な情報は埋め込みません。

対象は`internal/`以下の`_test.go`を除く`.go`ソースで、`testdata`、ドット・アンダースコアで始まるディレクトリは除外します。Goのbuild tagやGOOS/GOARCHごとの実行対象を解決しないため、**実際の一つのビルドで有効な依存グラフ**とは異なります。依存辺が存在しても、その機能が実行可能・権限付与済み・Provider適合済みであるとは主張しません。

欠けた`go.mod`、ソースの構文エラー、ディレクトリ走査失敗などは黙って空の成功結果にしません。特定revisionの報告が必要なら、実行した`git rev-parse HEAD`、toolchain、worktree状態を報告と一緒に保存します。ツール自身はcommitやファイルを変更しません。

このツールは依存の**観測**を目的とします。許容/禁止する依存方向の検査は`tools/repository-architecture-check/dependency_test.go`側に置き、観測結果に合わせて自動的に許容ルールを緩めません。両者とも本番用Runtime APIやTEIのCanonical meaningではありません。
