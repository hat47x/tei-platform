# 二回の独立Buildにおける再現性の観測

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-Cの前提として、**同じ対象を隔離した状態で二回Buildし、成果物の byte digest が一致するか**を実測します。§35.2 の負例「同digest別Attempt」（SB-N8）に対応します。

**これは再現性の観測であって、Builderの信頼でも受入でもありません。** 二回とも同じマシン・同じGo toolchainでの一致であり、別OS・別toolchainでも一致する保証はありません。G4は`UNQUALIFIED`のままです。

## 解く問題

「二回Buildして同じdigestだった」と書くだけでは、次の三つの区別が失われます。

1. **試行が本当に二回あったか**。同じ実行を二度数える、あるいはdigestから試行IDを作ると、二回を主張できません。
2. **両方成功したか**。片方が失敗してdigestが無い場合と、両方成功して一致した場合を混ぜません。
3. **状態が独立していたか**。`GOCACHE`/`GOMODCACHE`を共有すると、二回目は一回目の成果物を再利用している可能性があります。

追加した`tools/reference-go-build-reproducibility`は、**試行IDを乱数で発行**し（digestから導出しない）、試行ごとに`state`ディレクトリを分け、成功・digest一致・試行同一性を別々の欄で返します。

| 欄 | 意味 |
| --- | --- |
| `attempts[].attemptId` | 32 byte の乱数。artifact digest から計算しない |
| `attempts[].stateDir` | 試行ごとに分かれた隔離状態（`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`） |
| `bothAttemptsSucceeded` | 二回とも exit 0 かつ artifact 生成 |
| `artifactDigestsEqual` | 二つの digest が一致 |
| `attemptIdentitiesDistinct` | 二つの attempt ID が異なる |
| `reproducible` | 上三つがすべて真 |

片方でも欠ければ`reproducible: false`で、`failureCause`に理由が入ります。digestが一致しても試行IDが同じなら不合格です（同じ実行を二度数えただけになるため）。

## 環境

`BuildPackage`と同じdeny-by-default（`CGO_ENABLED=0`、`GOENV=off`、`GOEXPERIMENT=`、`GOFLAGS=`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`GOVCS=*:off`、`GOWORK=off`）に、`GOPROXY=file://<供給>`と試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`を重ねます。**Host cacheは継承しません。**

## 観測結果

- 環境: WSL2 Ubuntu 22.04、Go `go1.27.3 linux/amd64`
- 対象checkout: `5ad04cbdf`（`origin/main`、SB-0a/0b/X1 統合後）
- 供給: `/home/hat47x/go/pkg/mod/cache/download`
- 対象: `./cmd/tei`

| 欄 | 値 |
| --- | --- |
| `bothAttemptsSucceeded` | `true`（両試行 exit 0） |
| `artifactDigestsEqual` | `true` |
| `attemptIdentitiesDistinct` | `true` |
| `reproducible` | `true` |
| artifact digest（両試行） | `sha256:303a350c42cf4aa1f4ff445e9aa8a3beb27888825f31981c07cbb6a70d2db2f3` |

自己完結した対象（`./tools/reference-build-authoring-intent`）でも同じ試験が通りました。

## 解釈

**わかったこと**：`cmd/tei`は、供給を固定した隔離環境で二回Buildすると、**別々の試行IDと別々のstate**で同じ成果物byteを出す。SB-Cの比較に使える最小の再現性は、この対象について成立しています。

**主張しないこと**：

- Builderの信頼、artifactの受理、Runtime Authority、配備資格は一切動いていません。
- 別OS（Windows）・別Go版・別供給での一致は未確認です。X1の観測と同様、Linux amd64 の1環境での結果です。
- `go.sum`と供給byteの一致はSB-0b（[#3142](https://github.com/hat47x/tei-platform/pull/3142)で統合済み）が担っており、本作業はそれを前提にしています。
- Stage0/1/2の exact identity 比較（SB-C本体）は未着手です。本作業はその手前の「同じSourceを二回Buildすると一致するか」だけを見ています。

## 検証

- `gofmt -l tools/reference-go-build-reproducibility`：差分なし
- `go vet ./tools/reference-go-build-reproducibility`：PASS
- `go test ./tools/reference-go-build-reproducibility -count=1`：PASS（4件。自己完結対象の正例、`cmd/tei`の正例、試行IDの分離、file URLのvolume保持）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows実行、別Go版、別供給での一致、Stage0/1/2の exact identity 比較（SB-C本体）、CI、trusted full checkout。

## 参照

- 実装: [`tools/reference-go-build-reproducibility`](../../tools/reference-go-build-reproducibility/README.md)
- 前提: [Go選択Module閉包の事前観測](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)、[オフラインGo Module供給の検証](TEI_GO_MODULE_SUPPLY_VERIFICATION_2026-11-07.md)、[供給からの実BuildとSource閉包資格の食い違い](TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)
- 計画: [`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) §35.2
