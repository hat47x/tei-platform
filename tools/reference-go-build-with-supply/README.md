# 供給だけを使う実Build試行（SB-B事前確認 / X1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831)の**SB-B事前確認**と、§35.2の負例**X1**（「Build成功だがclosureは外部Module拒否」）を、実際に走らせて記録する読取り専用の参照実装です。

この実装は**Buildを実行しますが、artifactの受理・署名・権限・配備は行いません**。`artifactDigest`は、その試行で作られたバイト列の同一性だけを示します。G4は`UNQUALIFIED`のままです。

## 分けたこと

SB-0b（`tools/reference-go-module-supply-verify`）で、`cmd/tei`の実選択12件が`go.sum`と一致し、供給だけでオフライン再解決できることを確認しました。しかし再解決できることと、Buildできること、ましてや**Source閉包の資格が成立していること**は別です。

本Referenceは、同じ対象について次の三つを**別々の欄**で返します。

| 欄 | 意味 |
| --- | --- |
| `buildSucceeded` / `artifactDigest` | 隔離環境で実Buildが終了し、artifactができた |
| `sourceClosureState` | 同じadapterの`ResolveSourceClosure`が同じ対象を`qualified`とするか`rejected`とするか |
| `artifactProducedDespiteUnqualifiedClosure` | 上二つが食い違っていることの明示 |

三つ目を独立の真偽値にしたのは、**「artifactがある」を「closureが資格を持つ」と読み替えない**ためです。

## 環境

`BuildPackage`と同じdeny-by-default構成（`CGO_ENABLED=0`、`GOENV=off`、`GOEXPERIMENT=`、`GOFLAGS=`、`GOSUMDB=off`、`GOTOOLCHAIN=local`、`GOVCS=*:off`、`GOWORK=off`）に、`GOPROXY=file://<供給>`と試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`/`TMP*`を重ねます。**Host cacheは継承しません。**`GOPROXY=off`ではなく供給を指すため、`go.sum`と一致するbyteだけが隔離cacheへ入ります。

Go実行ファイルは一度だけ解決し、digestを記録します。`-go`にパスを渡すと逐語的に使い、`go`のような名前だけの場合はPATH解決します。後からPATHの中身が変わっても、記録したdigestが実行したバイナリを指すようにするためです。

## 実行

```bash
go run ./tools/reference-go-build-with-supply \
  -repo . -package ./cmd/tei -supply <供給ディレクトリ>
```

`-artifact-out`を省略すると一時ディレクトリへ出力します。`-state`を省略すると一時ディレクトリを作り、終了時に消します。終了コードは観測が行えたかだけを示し、Buildの成否や資格ではありません。

## 2026-11-07の観測結果（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

供給は`/home/hat47x/go/pkg/mod/cache/download`（Go自身のdownload cache形式）。対象は`./cmd/tei`。

| 欄 | 値 |
| --- | --- |
| `buildSucceeded` | `true`（exit 0） |
| `artifactDigest` | `sha256:a512ffba1e87beb63779d6b592bb110b5d1e49e5ccc07fb9c006d9d62fc74efd` |
| `sourceClosureState` | `rejected` |
| `artifactProducedDespiteUnqualifiedClosure` | `true` |
| `goExecutableDigest` | `sha256:5f5eb6d0b2fb6aa0bb5d5f8c5f1c944e0de955c925f0111af92a9d8f79680b5b` |

closureの拒否理由は`module lookup disabled by GOPROXY=off`（`ResolveSourceClosure`自身が`GOPROXY=off`の隔離状態で走るため、供給が渡っていない）です。**供給を渡したとしても、`ResolveSourceClosure`は`moduleDir != root`で外部Module根を拒否する**設計なので、どちらの停止理由でも資格は成立しません。理由の形は変わっても、`rejected`であり続けることを試験が固定しています。

## わかったこと／主張しないこと

**わかったこと**：

- `cmd/tei`は、隔離環境と`go.sum`一致の供給だけで**実際にビルドできる**。SB-Bの「実artifactを作る」側の前提は満たせる。
- 同じ対象のSource閉包は、同じadapterでは**資格を返さない**。したがってX1（artifact成功とqualified closureの混同）は現実の危険であり、欄を分ける必要がある。

**主張しないこと**：

- このartifact digestは受理済みでも配備可能でもありません。二回Buildの再現性も未確認です。
- 「closureが拒否されたからartifactが無効」でも「artifactがあるからclosureが成立」でもありません。二つは独立です。
- 供給元のTrust・署名・発行者は未検証です。
- Linux amd64での1回の観測です。Windows実行は同じ試験がWindowsでも通ることを別途確認していますが、artifact digestのOS間一致は主張しません。

## 実行した検証

- `gofmt -l tools/reference-go-build-with-supply`：差分なし
- `go vet ./tools/reference-go-build-with-supply`：PASS
- `go test ./tools/reference-go-build-with-supply -count=1`：PASS（4件。X1、自己完結対象の正例、環境がHost cacheを継承しないこと、file URLのvolume保持）

**NOT RUN**：二回Buildの再現性、closure方針の変更（[#1090](https://github.com/hat47x/tei-platform/issues/1090)の所有者判断）、Stage比較（SB-C）、CI、trusted full checkout。

## 限界

- `moduleCacheEntries`は隔離`GOMODCACHE`に現れたパスを列挙したもので、「どのbyteが実際にコンパイラへ渡ったか」の完全な証明ではありません。その証明は`ResolveSourceClosure`と実Buildの突合が担います。
- 供給に無いModuleが必要ならBuildは失敗し、`buildError`に記録されます。失敗を成功へ丸めません。
- `ResolveSourceClosure`側の隔離状態は`GOPROXY=off`のままなので、この比較は「供給ありのBuild」対「供給なしのclosure」です。供給ありのclosureは存在しない経路（外部Module根の拒否）なので、比較として成立します。
