# 供給からの実Buildと、Source閉包資格の食い違い（X1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) の**SB-B事前確認**と、§35.2の負例**X1**（「artifactはできるがSource閉包の資格は成立しない」）を、実際に走らせて記録したものです。§35.2の負例1に対応します。

**これはBuildの実行記録であって、artifactの受理ではありません。** 二回Buildの再現性も、供給元のTrustも、統合方針の採用も含みません。G4は`UNQUALIFIED`のままです。

## 前提

[SB-0a](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)で、`cmd/tei`は`GOPROXY=off`の隔離観測では`missing-module`でした。[SB-0b](https://github.com/hat47x/tei-platform/pull/3142)で、実選択12件が`go.sum`の`h1`と一致し、供給だけでオフライン再解決できることを確認しました。

残っていたのは、**再解決できる**ことと、**資格ある閉包が成立する**ことの間の段差です。本作業はその段差を実Buildで測りました。

## 二つを別欄で返す

追加した`tools/reference-go-build-with-supply`は、同じ対象について次を別々に返します。

| 欄 | 意味 |
| --- | --- |
| `buildSucceeded` / `artifactDigest` | 隔離環境でBuildが終了し、artifactができた |
| `sourceClosureState` | 同じadapterの`ResolveSourceClosure`が同じ対象を`qualified`とするか`rejected`とするか |
| `artifactProducedDespiteUnqualifiedClosure` | 上二つが食い違っていることの明示 |

三つ目を独立の真偽値にしたのは、**「artifactがある」を「closureが資格を持つ」と読み替えない**ためです。

環境は`BuildPackage`と同じdeny-by-default（`CGO_ENABLED=0`、`GOENV=off`、`GOEXPERIMENT=`、`GOFLAGS=`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`GOVCS=*:off`、`GOWORK=off`）に、`GOPROXY=file://<供給>`と試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`を重ねます。**Host cacheは継承しません。**Go実行ファイルは一度だけ解決してdigestを記録し、後からPATHが変わっても記録が実行したバイナリを指すようにしています。

## 観測条件

- 環境: WSL2 Ubuntu 22.04、Go `1.27.3`（linux/amd64）
- 対象checkout: `388412958`（`origin/main`）
- 供給: `/home/hat47x/go/pkg/mod/cache/download`（Go自身のdownload cache形式）
- 対象package: `./cmd/tei`

## 観測結果

| 欄 | 値 |
| --- | --- |
| `buildSucceeded` | `true`（exit 0） |
| `artifactDigest` | `sha256:a512ffba1e87beb63779d6b592bb110b5d1e49e5ccc07fb9c006d9d62fc74efd` |
| `sourceClosureState` | `rejected` |
| `artifactProducedDespiteUnqualifiedClosure` | `true` |
| `goExecutableDigest` | `sha256:5f5eb6d0b2fb6aa0bb5d5f8c5f1c944e0de955c925f0111af92a9d8f79680b5b` |

closureの拒否理由は`module lookup disabled by GOPROXY=off`でした。`ResolveSourceClosure`自身が`GOPROXY=off`の隔離状態で走るため、供給は渡っていません。**供給を渡したとしても、`ResolveSourceClosure`は`moduleDir != root`で外部Module根を拒否する**設計なので、どちらの停止理由でも資格は成立しません。理由の形は変わっても`rejected`であり続けることを試験が固定しています。

## 読み取り

**わかったこと**：

- `cmd/tei`は、隔離環境と`go.sum`一致の供給だけで**実際にビルドできる**。SB-Bの「実artifactを作る」側の前提は満たせる。
- 同じ対象のSource閉包は、同じadapterでは**資格を返さない**。したがってX1（artifact成功とqualified closureの混同）は現実の危険であり、欄を分ける必要がある。

**主張しないこと**：

- このartifact digestは受理済みでも配備可能でもありません。二回Buildの再現性も未確認です。
- 「closureが拒否されたからartifactが無効」でも「artifactがあるからclosureが成立」でもありません。二つは独立です。
- 供給元のTrust・署名・発行者は未検証です。
- Linux amd64での1回の観測です。同じ試験がWindowsでも通ることは別途確認していますが、artifact digestのOS間一致は主張しません。

## 検証

- `gofmt -l tools/reference-go-build-with-supply`：差分なし
- `go vet ./tools/reference-go-build-with-supply`：PASS
- `go test ./tools/reference-go-build-with-supply -count=1`：PASS（4件。X1、自己完結対象の正例、環境がHost cacheを継承しないこと、file URLのvolume保持）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：二回Buildの再現性、二回の独立試行でのdigest一致、closure方針の変更（[#1090](https://github.com/hat47x/tei-platform/issues/1090)の所有者判断）、Stage比較（SB-C）、CI、trusted full checkout。

## 次の判断点

`cmd/tei`をSB-Bの対象にするなら、**Source閉包の外部Module方針**をどう扱うかを先に決める必要があります。本作業はその方針を変えておらず、変えるかどうかは[#1090](https://github.com/hat47x/tei-platform/issues/1090)の所有者の判断です。`rejected`のまま進める選択肢（artifactとclosureを別Claimとして扱い続ける）も残ります。

## 参照

- 実装: [`tools/reference-go-build-with-supply`](../../tools/reference-go-build-with-supply/README.md)
- 前段: [Go選択Module閉包の事前観測](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)（SB-0a）、[Go Module供給の実観測](TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md)、[オフラインGo Module供給の検証](TEI_GO_MODULE_SUPPLY_VERIFICATION_2026-11-07.md)（SB-0b）
- 設計: [`GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md`](../design/GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md) §5
- 計画: [`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) §35.2
