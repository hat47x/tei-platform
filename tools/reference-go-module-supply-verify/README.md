# オフラインGo Module供給の検証

[#2831](https://github.com/hat47x/tei-platform/issues/2831)の**SB-0b**に対応する読取り専用の参照実装です。SB-0aで`cmd/tei`が`missing-module`と分類されたことを受けて、**供給の事前準備を桁上げで済ませない**ための最小契約を実装します。

この実装は**Build・Artifact・受入・権限のいずれも行いません**。`verified: true`は「対象が実際に選んだModuleのbyteが`go.sum`と一致し、その供給だけでオフライン再解決できた」という観測です。供給元の正しさ・発行者・署名・Trustは検証しません。

## 解く問題

SB-0aは`cmd/tei`が`missing-module`（`module lookup disabled by GOPROXY=off`）だと示しました。次に必要なのは「では何を供給すればよいか」の契約です。ここで二つの安易な進め方を退けます。

1. **Host cacheをそのまま使う**：`C:\Users\yhata\go\pkg\mod`を指すだけでBuildは通りますが、それは「たまたま手元にあった」であり、供給の同一性も`go.sum`との一致も確認していません。試行ごとの隔離`GOMODCACHE`という現行方針とも両立しません。
2. **`go.sum`に載っている`go.mod`宣言Moduleを全部供給する**：`go.mod`は17件を宣言していますが、`cmd/tei`が実際に選ぶのは12件です。宣言と実消費を同一視すると、選択されない5件の欠落で正しい供給を不合格にします。

本Referenceは、**先に実選択を解決し、その選択結果にだけbyte照合を適用**します。

## 判定の順序

```text
go.mod の宣言を読む
        |
        v
GOPROXY=file://<supply> + 隔離GOMODCACHE + GOPROXY=off系固定設定で go list -deps -json -e
        |
        +---- 解決できたModule集合 ----+
        |                             |
        v                             v
go.sum に無い選択Module       選択されたModuleごとに
  -> verified=false            .mod と .zip の h1 を再計算
  (宣言外の混入)                 -> go.sum と一致しなければ verified=false
```

- **宣言外の選択Moduleが出たら不合格**。`go.mod`に無い物を勝手に選んだ供給は、何を供給したのか分からない状態です。
- **宣言したが選択されないModuleは`unselectedDeclaredModules`へ記録**して不合格にしません。対象の供給要件ではないためです。
- **対象が外部Moduleを1つも選ばない場合は不合格**（`verified: false`）。何も検査していないのに`verified: true`を返すのは空虚な主張です。
- **再解決が失敗した場合は、宣言Moduleのbyte照合を回して原因を切り分けます**。byte改ざんなのか単なる欠落なのかを`failureCause`で区別します。どちらも`verified: false`です。

## `h1`の再計算

`go.sum`の`h1:`はGoのmodule hashです。本Referenceは`archive/zip`と`crypto/sha256`だけを使って独立に再計算します。

- `.zip`：メンバーごとの内容SHA-256（hex）を名前順に `"<hex>  <name>\n"` で連結し、そのSHA-256をbase64で表す
- `.mod`：単一メンバー`go.mod`に同じ構成を適用

Go toolchain自身も`go.sum`不一致の供給を拒否しますが、本Referenceは**その成否に依存せず同じ値を別実装で再計算**します。`go.sum`の行が3フィールドでない場合は、読めない`go.sum`を証拠にしないためエラーにします。

## 環境

`BuildPackage`と同じdeny-by-default構成（`GOENV=off`、`GOWORK=off`、`GOTOOLCHAIN=local`、`GOSUMDB=off`、`GOVCS=*:off`、`CGO_ENABLED=0`）に、`GOPROXY=file://<supply>`と試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`を重ねます。**Host cacheは継承しません。**`GOPROXY`を`file://`にして`GOSUMDB=off`のままなので、供給のbyteは`go.sum`だけで拘束されます。

## 実行

```bash
go run ./tools/reference-go-module-supply-verify \
  -repo . -supply <供給ディレクトリ> -package ./cmd/tei
```

`-supply`はGoのmodule download-cache形式（`GOPROXY=file://`の形式）です。`<root>/<module path>/@v/<version>.mod|.zip`を読みます。`-state`を省略すると一時ディレクトリを作り、終了時に消します。

終了コードは観測が行えたかだけを示します。`verified: false`は「供給が要件を満たさない」という証拠であり、対象の受理可否ではありません。

## 2026-11-07の観測結果（この実行環境）

- 対象checkout: `041db3d2c72867bb6d38d83053bcedb09841d39a`（`origin/main`）
- Go: `go1.27.0`（windows/amd64）
- 供給: `C:\Users\yhata\go\pkg\mod\cache\download`（Go自身のdownload cache形式）
- `go.sum` digest: `sha256:d5967ad9f867efa398e7fa9afaccb61819a3e8f520aaee7bbb4a5069402c55d3`

| 対象 | 宣言 | 実選択 | 宣言のみ（未選択） | 結果 |
| --- | --- | --- | --- | --- |
| `./cmd/tei` | 17 | 12 | 5（`google/uuid`、`golang.org/x/sys`、`puddle/v2`、`go-ora/v2`、`sync`） | `verified: true`（選択12件の`.mod`と`.zip`が`go.sum`と一致、再解決exit 0） |

**供給の準備についての開示**：この観測の前に、宣言Moduleのうち`github.com/google/uuid@v1.6.0`と`golang.org/x/sys@v0.22.0`がHostのdownload cacheに無かったため、`go mod download`で補いました。これは**観測用fixtureの準備**であり、供給元のTrustでも署名でもありません。補った2件はどちらも`cmd/tei`の実選択に含まれません（未選択側）。

## 検証した退け方

| 入力 | 期待 | 実測 |
| --- | --- | --- |
| `pgpassfile@v1.0.0`のzipを1 byte改ざんした供給 | `verified: false`、原因がbyte不一致 | PASS（`zip.computedHash`が`go.sum`と不一致） |
| `go.sum`に無い版（`v9.9.9`）を宣言 | `verified: false`、`.mod`/`.zip`双方に理由 | PASS |
| 外部Moduleを選ばない自己完結tool + 空供給 | `verified: false`（空虚な成功を返さない） | PASS |
| 完全な供給 + `cmd/tei` | `verified: true`、12件すべて一致 | PASS |

## 限界と、次の課題

- 供給元・発行者・署名・Trustは**未検証**です。本Referenceは「`go.sum`と一致するbyteが、そのパスに、その供給だけで存在する」ことだけを言います。
- `go.sum`自体のTrust（誰が`go.sum`を受理したか）は別のClaimです。
- **供給を用意しても、既存`gobuilder.ResolveSourceClosure`は外部Module根を拒否するため`cmd/tei`のSource閉包には到達しません。** この方針は[#1090](https://github.com/hat47x/tei-platform/issues/1090)の安全側の停止線であり、本Referenceは変更していません。方針の変更は所有者の判断です。証拠として`TestResolveSourceClosureStillRefusesVerifiedExternalModules`が現状を固定しています。
- 本Referenceは`verified`を出してから再解決を行うため、判定は2回の観測（byte照合と再解決）に依存します。同じSource・同じ供給・同じGo版での再現性はありますが、OS・Go版をまたぐ一般化はしていません。
- `.zip`のメンバー順序・圧縮率の差は内容hashに影響しません（Goのh1と同じ定義）。したがって「同じ内容を別zipへ再梱包した供給」は通ります。これは仕様どおりですが、byte同一性の主張ではありません。

## 実行した検証

- `gofmt -l tools/reference-go-module-supply-verify`：差分なし
- `go vet ./tools/reference-go-module-supply-verify`：PASS
- `go test ./tools/reference-go-module-supply-verify -count=1`：PASS（8件）。Windows 11上のGo 1.27.0（windows/amd64）とWSL2 Ubuntu 22.04上のGo 1.27.3（linux/amd64）の両方で実行。

**NOT RUN**：`cmd/tei`の実`go build`、二回Buildの再現性、`cmd/tei`のSource閉包到達（上記の方針課題）、Stage比較（SB-C）、CI、trusted full checkout。
