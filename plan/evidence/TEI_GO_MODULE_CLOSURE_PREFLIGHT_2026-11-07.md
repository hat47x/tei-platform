# Go選択Module閉包の事前観測（SB-0a）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-0aに対応する、読み取り専用の事前観測の記録です。`cmd/tei`をGo adapterで実Buildへ進める前に、**どのModuleが選択され、そのどれがオフラインで解決できなかったか**を区別します。

**これはBuildの実行ではありません。** artifactを作らず、`ResolveSourceClosure`の資格もG4の判定も動かしません。G4は`UNQUALIFIED`のままです。

## 何を分けたか

既存`tools/internal/gobuilder.ResolveSourceClosure`は、次を一つの`error`へ畳みます。

1. 外部Module根の選択（`moduleDir != root`）
2. 外部Module内容の未供給
3. 対象Go Source自体の解決失敗

1と2は供給の追加で解けますが、3はSourceの修正が要ります。畳んだままだと、`cmd/tei`のどのimportが止まっているかを呼び出し側が推測するしかありません。

追加した`ResolveModuleClosurePreflight`（`tools/internal/gobuilder/module_closure.go`）は`go list -deps -json -e -mod=readonly`を1回実行し、per-package failureを保持して次の四つに分類します。

| 分類 | 判定条件 |
| --- | --- |
| `supported` | 選択されたModule根が対象Module1つだけ。全packageが解決済み |
| `external-unresolved` | 外部Module根が選択され内容も解決済み（照合すべき外部入力がある） |
| `missing-module` | 未解決packageのエラーが既知の供給系文面（`module lookup disabled`、`missing go.sum entry` 等）に一致 |
| `unknown` | 上記以外の解決失敗、または`go list`が非0終了 |

優先順位は`unknown` → `missing-module` → `external-unresolved` → `supported`です。認識しない失敗を供給不足へ丸めないため、`unknown`を最上位に置きます。`go list`が非0終了した場合は、部分出力があっても分類を`unknown`へ落とします。

環境は`BuildPackage`と同じdeny-by-default（`GOPROXY=off`、`GOWORK=off`、`GOTOOLCHAIN=local`、`GOFLAGS=`、試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`）です。

## 実行した観測

- 対象checkout: `4fdad69d9f360ef605930cc298f2a72efdfdee8e`（`origin/main`。作業ブランチの分岐点）
- 環境: Go `go1.27.0`、windows/amd64
- 実行入口: `go run ./tools/reference-go-module-closure-preflight -repo . -package <path> -state <一時dir>`
- Go実行ファイルdigest: `sha256:7d828191ba32519a9c9361789ab647486236ed45c660889196c7770a8ff1985c`
- `go.mod`: `sha256:ad0264180541bc910132f3f9183deb81539c89b3f4d88ba68fd2512d0382f19c`
- `go.sum`: `sha256:d5967ad9f867efa398e7fa9afaccb61819a3e8f520aaee7bbb4a5069402c55d3`

| package | 分類 | 解決package数 | 未解決import（位置） |
| --- | --- | --- | --- |
| `./cmd/tei` | `missing-module` | 65 | `github.com/jackc/pgx/v5`（`cmd/tei/dbcheck.go`）、`golang.org/x/text/unicode/norm`（`internal/command/reference_store.go`）、`modernc.org/sqlite`（`internal/command/lifecyclesqlite/sqlite.go`） |
| `./tools/reference-build-authoring-intent` | `supported` | 30 | なし |

`cmd/tei`の3件はいずれも`module lookup disabled by GOPROXY=off`でした。Source・構文の問題ではなく、隔離観測に外部Module内容が無いことによる停止です。

## 解釈と、解釈しないこと

**わかったこと**：現行のdeny-by-default観測では、`cmd/tei`の選択Source閉包へは到達できません。SB-B（`cmd/tei`の実Artifact）へ進む前に、SB-0b（検証済みbyte・`go.sum`・版・オフライン事前供給の最小契約）が要ります。

**わからないこと／主張しないこと**：

- `cmd/tei`の実Buildが必ず失敗する、とは言えません。実装環境には`C:\Users\yhata\go\pkg\mod`（773 MB）が存在し、`dgx/v5`・`x/text`・`modernc.org/sqlite`の版も一部揃っています。Host cacheを使えば進む可能性がありますが、Host cacheの利用は現行方針（試行ごとの隔離`GOMODCACHE`）と両立しません。**この経路は本観測では一切使っていません。**
- 外部Moduleの実byteと`go.sum`の一致、供給元のTrust、署名は**未検証**です。`go.sum`の存在だけでは何も証明しません。
- `supported`は「対象Moduleだけで解決した」という観測であり、artifact成立・closure資格・受入ではありません。
- 別のGo版・別OS・別供給への一般化はできません。

## 限界

- `missing-module`の判定は`go list`の既知のエラー文面に依存します。Go版が文面を変えると`unknown`へ落ちます。落ちる側は安全ですが、分類の網羅性は保証しません。
- 分類は`go list`の観測であり、実Buildが同じファイル集合を使うことの証明は`ResolveSourceClosure`と実Build試行の照合が担います。
- `moduleRootDir`と`dir`はマシン固有の物理パスです。内容同一性として引用しません。

## 実行した検証

- `go vet ./tools/internal/gobuilder ./tools/reference-go-module-closure-preflight`：PASS
- `go test ./tools/internal/gobuilder -count=1`：PASS（既存試験を含む）
- `go test ./tools/reference-go-module-closure-preflight -count=1`：PASS（投げ捨てModuleでの`missing-module`、フラグ不正、実リポジトリの`cmd/tei`分類）
- `go test ./tools/repository-architecture-check -count=1`：PASS（新規`tools/`ディレクトリの配置検査）
- `gofmt -l tools/internal/gobuilder tools/reference-go-module-closure-preflight`：差分なし

**NOT RUN**：`cmd/tei`の実`BuildPackage`実行、二回Buildの再現性、`ResolveSourceClosure`の`cmd/tei`到達、外部Moduleのオフライン供給（SB-0b）、Stage比較（SB-C）、CI、trusted full checkout。

## 参照

- 実装: [`tools/reference-go-module-closure-preflight`](../../tools/reference-go-module-closure-preflight/README.md)、[`tools/internal/gobuilder/module_closure.go`](../../tools/internal/gobuilder/module_closure.go)
- 計画: [`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) の §35.2 と §35.3
- 判定の正本: [docs/status.md](../../docs/status.md)、Gateは[#1119](https://github.com/hat47x/tei-platform/issues/1119)
