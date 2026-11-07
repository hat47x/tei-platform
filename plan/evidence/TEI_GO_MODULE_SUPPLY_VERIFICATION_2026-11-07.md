# オフラインGo Module供給の検証（SB-0b）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-0bに対応する、読み取り専用の検証記録です。SB-0aで`cmd/tei`が`missing-module`（`module lookup disabled by GOPROXY=off`）と分類されたことを受けて、**何を供給すれば対象の閉包が閉じるか**を、宣言ではなく実選択と`go.sum`で拘束します。

**これはBuildの実行ではありません。** artifactを作らず、供給元のTrustや署名も検証しません。G4は`UNQUALIFIED`のままです。

## 分けたこと

SB-0aの観測だけでは、次の二つの安易な進め方を退けられません。

1. **Host cacheをそのまま使う**：`C:\Users\yhata\go\pkg\mod`を指せばBuildは通りますが、「たまたま手元にあった」だけであり、`go.sum`との一致も供給の同一性も確認していません。試行ごとの隔離`GOMODCACHE`という現行方針とも両立しません。
2. **`go.mod`の宣言Moduleを全部供給する**：`cmd/tei`の`go.mod`は17件を宣言していますが、実際に選ぶのは12件です。宣言と実消費を同一視すると、選択されない5件の欠落で正しい供給を不合格にします。

追加した`VerifySupply`（`tools/reference-go-module-supply-verify`）は、**先に実選択を解決し、その選択結果にだけbyte照合を適用**します。

| 状態 | 判定 |
| --- | --- |
| 再解決が失敗（exit非0 または未解決package） | `verified: false`。宣言Moduleのbyte照合も回して原因を切り分ける |
| 選択されたModuleが`go.mod`に無い | `verified: false`（宣言外の混入） |
| 選択されたModuleの`.mod`/`.zip`が`go.sum`と不一致、または欠落 | `verified: false` |
| 宣言したが選択されないModule | `unselectedDeclaredModules`へ記録。不合格にはしない |
| 対象が外部Moduleを1つも選ばない | `verified: false`（何も検査していない成功を返さない） |

`h1:`はGoのmodule hashを`archive/zip`と`crypto/sha256`だけで独立に再計算します（メンバーの内容SHA-256（hex）を名前順に`"<hex>  <name>\n"`で連結し、全体のSHA-256をbase64）。Go toolchain自身も`go.sum`不一致を拒否しますが、その成否に依存せず別実装で同じ値を出します。

## 環境

`BuildPackage`と同じdeny-by-default（`GOENV=off`、`GOWORK=off`、`GOTOOLCHAIN=local`、`GOSUMDB=off`、`GOVCS=*:off`、`CGO_ENABLED=0`）に、`GOPROXY=file://<供給>`と試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`を重ねます。**Host cacheは継承しません。**

## 実行した観測

- 対象checkout: `041db3d2c72867bb6d38d83053bcedb09841d39a`（`origin/main`）
- 環境: Go `go1.27.0`、windows/amd64
- 供給: `C:\Users\yhata\go\pkg\mod\cache\download`（Go自身のdownload cache形式）
- `go.sum` digest: `sha256:d5967ad9f867efa398e7fa9afaccb61819a3e8f520aaee7bbb4a5069402c55d3`

| 対象 | 宣言 | 実選択 | 宣言のみ（未選択） | 結果 |
| --- | --- | --- | --- | --- |
| `./cmd/tei` | 17 | 12 | 5（`google/uuid`、`golang.org/x/sys`、`puddle/v2`、`go-ora/v2`、`x/sync`） | `verified: true`、再解決exit 0 |

選択された12件は`pgx/v5`、`x/text`、`modernc.org/sqlite`とその推移依存（`pgpassfile`、`pgservicefile`、`go-humanize`、`go-isatty`、`go-strftime`、`bigfft`、`modernc/libc`、`mathutil`、`memory`）です。

**供給の準備についての開示**：この観測の前に、宣言Moduleのうち`github.com/google/uuid@v1.6.0`と`golang.org/x/sys@v0.22.0`がHostのdownload cacheに無かったため、`go mod download`で補いました。これは観測用fixtureの準備であり、供給元のTrustでも署名でもありません。補った2件はいずれも実選択に含まれません（未選択側）。

## 退け方を確かめた入力

| 入力 | 期待 | 実測 |
| --- | --- | --- |
| `pgpassfile@v1.0.0`のzipを1 byte改ざんした供給で`cmd/tei` | `verified: false`、原因がbyte不一致 | PASS（該当Moduleだけ`zip.computedHash`が`go.sum`と不一致、他は再解決失敗として記録） |
| `go.sum`に無い版（`v9.9.9`）を宣言 | `verified: false`、`.mod`/`.zip`双方に理由 | PASS |
| 外部Moduleを選ばない自己完結tool + 空供給 | `verified: false`（空虚な成功を返さない） | PASS |
| 完全な供給で`cmd/tei` | `verified: true` | PASS |

## 解釈と、主張しないこと

**わかったこと**：`cmd/tei`が実際に選ぶ12件のModuleは、Go download cache形式の供給と`go.sum`だけで、Host cacheを継承せずにオフライン再解決でき、byteも一致します。SB-0bの最小契約として成立する素材がそろっています。

**わからないこと／主張しないこと**：

- 供給元・発行者・署名・Trustは**未検証**です。`go.sum`自体を誰が受理したかも別のClaimです。
- **供給を用意しても、既存`gobuilder.ResolveSourceClosure`は外部Module根を拒否するため`cmd/tei`のSource閉包へは到達しません。** これは[#1090](https://github.com/hat47x/tei-platform/issues/1090)の安全側の停止線であり、本作業は変更していません。証拠として`TestResolveSourceClosureStillRefusesVerifiedExternalModules`が現状を固定しています。方針変更は所有者の判断です。
- 同じ内容を別zipへ再梱包した供給は通ります（h1の定義どおり）。byte同一性の主張ではありません。
- 別OS・別Go版への一般化はしていません。

## 実行した検証

- `gofmt -l tools/reference-go-module-supply-verify`：差分なし
- `go vet ./tools/reference-go-module-supply-verify`：PASS
- `go test ./tools/reference-go-module-supply-verify -count=1`：PASS（8件）。**Windows 11上のGo 1.27.0（windows/amd64）と、WSL2 Ubuntu 22.04上のGo 1.27.3（linux/amd64）の両方**で実行し、どちらもPASSしました。片方のOSで通ったことを他方の一般化には使っていません。
- `go test ./tools/repository-plan-check -count=1`：PASS（新規Markdownの相対リンク）

`h1`の再計算は、Go toolchainが記録した`go.sum`の値と突き合わせて一致しました。`moduleHash`を自作しただけでなく、**Go自身の出力と同じ値になること**を確認しています。

**NOT RUN**：`cmd/tei`の実`go build`、二回Buildの再現性、`cmd/tei`のSource閉包到達（上記の方針課題）、Stage比較（SB-C）、CI、trusted full checkout。

## 検証中に分かった実装上の制約

Go toolchainは抽出済みModule cacheのファイルを読み取り専用（`0444`）にします。`t.TempDir()`の自動削除はこのままでは権限エラーで失敗し、**試験本体がPASSしていてもcleanupでFAILと報告されます**（Linux実行で実際に発生）。本作業では、権限を戻してから削除する専用のcleanupを追加し、試験の成否とcleanupの成否を混同しないようにしました。

## 参照

- 実装: [`tools/reference-go-module-supply-verify`](../../tools/reference-go-module-supply-verify/README.md)
- 前段の観測: [Go選択Module閉包の事前観測](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)（SB-0a）
- 計画: [`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) の §35.2
- 判定の正本: [docs/status.md](../../docs/status.md)、Gateは[#1119](https://github.com/hat47x/tei-platform/issues/1119)
