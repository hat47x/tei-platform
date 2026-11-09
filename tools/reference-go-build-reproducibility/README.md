# Go成果物の二回Build再現性の観測

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C前提として、同じ対象を**隔離した二つの状態**で実際にBuildし、成果物byteの一致を測る読取り専用の参照実装です。§35.2の負例SB-N8（「同digestだが別Attempt」）に対応します。

この実装は**Buildを実行しますが、artifactの受理・署名・権限・配備は行いません**。`reproducible: true`は「同一マシン・同一toolchainで二つの独立試行が同じbyteを出した」という観測です。G4は`UNQUALIFIED`のままです。

## 三つを別欄で返す

「二回Buildして同じdigestだった」と書くだけでは、次の区別が失われます。

1. 試行が本当に二回あったか
2. 両方成功したか
3. 状態が独立していたか

本Referenceは、**試行IDを32 byteの乱数で発行**し（artifact digestから導出しない）、試行ごとにstateディレクトリを分け、次を別々に返します。

| 欄 | 意味 |
| --- | --- |
| `attempts[].attemptId` | 乱数。同じbyteでも別の試行として記録される |
| `attempts[].stateDir` | 試行ごとの隔離状態（`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`） |
| `bothAttemptsSucceeded` | 二回ともexit 0かつartifact生成 |
| `artifactDigestsEqual` | 二つのdigestが一致 |
| `attemptIdentitiesDistinct` | 二つのattempt IDが異なる |
| `reproducible` | 上三つがすべて真 |

片方でも欠ければ`reproducible: false`になり、`failureCause`に理由が入ります。**digestが一致しても試行IDが同じなら不合格**です（同じ実行を二度数えただけになるため）。

## 環境

`BuildPackage`と同じdeny-by-default（`CGO_ENABLED=0`、`GOENV=off`、`GOEXPERIMENT=`、`GOFLAGS=`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`GOVCS=*:off`、`GOWORK=off`）に、`GOPROXY=file://<供給>`と試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`を重ねます。**Host cacheは継承しません。**Go実行ファイルは一度だけ解決し、`go version`とdigestを記録します。

## 実行

```bash
go run ./tools/reference-go-build-reproducibility \
  -repo . -package ./cmd/tei -supply <供給ディレクトリ>
```

`-state-root`を省略すると一時ディレクトリを作り、終了時に消します。終了コードは観測が行えたかだけを示し、再現性の成否ではありません。

## 2026-11-07の観測結果（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

対象checkout `5ad04cbdf`、供給はGo自身のdownload cache形式、対象は`./cmd/tei`。

| 欄 | 値 |
| --- | --- |
| `bothAttemptsSucceeded` | `true`（両試行 exit 0） |
| `artifactDigestsEqual` | `true` |
| `attemptIdentitiesDistinct` | `true` |
| `reproducible` | `true` |
| artifact digest（両試行） | `sha256:303a350c42cf4aa1f4ff445e9aa8a3beb27888825f31981c07cbb6a70d2db2f3` |

`./tools/reference-build-authoring-intent`（外部Moduleを選ばない対象）でも同じ試験が通りました。

## 限界

- 同一マシン・同一toolchainでの一致です。別OS・別Go版・別供給での一致は未確認で、artifact digestのOS間一致は主張しません。
- 供給byteと`go.sum`の一致は`tools/reference-go-module-supply-verify`（SB-0b）が担っており、本Referenceはそれを前提にしています。
- 二回の試行は同じリポジトリcheckoutを共有します。checkoutの途中変更を検出するのは`ResolveSourceClosure`側の役割で、本Referenceは見ていません。
- Stage0/1/2の exact identity 比較（SB-C本体）は未着手です。

## 実行した検証

- `gofmt -l tools/reference-go-build-reproducibility`：差分なし
- `go vet ./tools/reference-go-build-reproducibility`：PASS
- `go test ./tools/reference-go-build-reproducibility -count=1`：PASS（4件。自己完結対象の正例、`cmd/tei`の正例、試行IDの分離、file URLのvolume保持）

**NOT RUN**：Windows実行、別Go版、Stage0/1/2の exact identity 比較、CI、trusted full checkout。
