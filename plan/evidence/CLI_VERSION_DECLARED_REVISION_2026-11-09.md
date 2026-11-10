# `-buildvcs=false` の `tei` が版を自己申告できるようにする（記録）

- 記録日: 2026-11-09
- 対象: `cmd/tei/cli_version.go`、`docs/getting-started.md`
- 関連: #3558、#2399、#3149。

## 観測（修正前）

`go build -buildvcs=false` で作った `tei` は、`version -json` が `source:"unavailable"` を返し、`vcsRevision` を持たなかった。checkoutの外でbuildする呼出し元は、実行物そのものからcommitを確かめられず、build前後でcheckoutが変わっていないことに頼るしかなかった。TEIの他のツールは `-ldflags="-X main.referenceAPISourceRevision=…"` で外部注入の経路を持っていたが、`cmd/tei` には同等の変数が無かった。

## 変更（案(a)を採用）

- `cmd/tei/cli_version.go` に `var teiDeclaredRevision string` を追加した。`-ldflags "-X main.teiDeclaredRevision=<commit>"` で注入できる。
- `cliBuildIdentity` に `declaredRevision`（`omitempty`）を追加し、`vcsRevision` とは**別の欄**にした。
- `source` は3値:
  - `build-info`: toolchainが `vcs.revision` を埋め込んだ（最も強い証拠）。
  - `ldflags`: 注入された自己申告だけがある。
  - `unavailable`: どちらも無い（従来の契約を維持）。
- 両方ある場合は `source` は `build-info` のまま、`declaredRevision` も保持する（強い証拠を弱い宣言で置き換えない）。
- テキスト出力は `declared revision: <値> (self-reported by the builder)` の行を追加し、`revision=` 行の値は変えない。
- `docs/getting-started.md` に注入例と、`declaredRevision` が自己申告であること（実行物とsourceの一致を証明しないこと）を追記した。

## 検証

- `cmd/tei/cli_version_declared_revision_test.go` を追加。
  - 注入のみ: `source: "ldflags"`、`declaredRevision` あり、`vcsRevision` は空。
  - toolchain stampと注入の併存: `source` は `build-info`、両方の値が残る。
  - どちらも無い: `unavailable` で両欄が空（従来どおり）。
  - テキスト: `revision=unavailable` と自己申告行が両立する。
  - **実プロセス**: `-buildvcs=false -ldflags "-X main.teiDeclaredRevision=…"` でbuildしたバイナリの `version -json` が `ldflags` と注入値を返し、`version` のテキストに自己申告行が出る。
- 負例: `case identity.DeclaredRevision != "":` を無効化すると、単体と実プロセスの両試験が失敗する（コードは戻した）。
- `gofmt -l cmd` 空、`go vet ./cmd/tei` PASS、`go build ./...` PASS、`go test ./cmd/tei -run 'TestVersion|TestBuildIdentity'` PASS、`go test ./cmd/tei -run 'TestContractDocument|TestDiscovery'` PASS。

## 主張しないこと

- 注入値はbuildした側の自己申告であり、`build-info` の `vcs.revision` と同じ強さの証拠ではない。実行物とsourceの一致、ましてや配布の承認・署名を意味しない。
- `-expect-commit` のような照合の使い方はTAI側の判断で、TEIは欄の意味だけを定めた。
- リリース成果物への署名・来歴（provenance）の経路は扱っていない。
