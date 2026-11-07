# 外部Moduleを含む選択Sourceの観測

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-A1（範囲付き完全性）と SB-B の前提として、**対象が実際にコンパイルする入力のうち、外部Module由来のものを一覧できるようにする**参照です。

**これは観測であって、閉包の資格でも受理でもありません。** `qualified`は常に`false`です。既存の`ResolveSourceClosure`が外部Module根を拒否する方針（[#1090](https://github.com/hat47x/tei-platform/issues/1090)）は**変えていません**。

## 解く問題

SB-B/X1（[#3156](https://github.com/hat47x/tei-platform/pull/3156)で統合済み）で、`cmd/tei`は供給だけでBuildできるのに、同じadapterのSource閉包は`rejected`になることを記録しました。しかし「では具体的に何が閉包に入っていないのか」は、`ResolveSourceClosure`のエラーメッセージ（最初の外部Moduleで止まる）からは分かりません。

追加した`ObserveSourceClosure`は、`go list -deps -json`の出力を次の二つに分けて返します。

| 欄 | 中身 |
| --- | --- |
| `repositoryMembers` / `repositoryDigest` | 対象Module内の選択ファイル。既存`SourceClosureDigest`と同じ符号化なので、資格ある閉包と比較できる |
| `externalMembers` / `externalModules` | 外部Module由来の選択ファイル。名前は`module:<path>@<version>/<file>`で、**物理cache位置に依存しない** |
| `observedDigest` | 両方を束ねた観測用digest。`observed\n`を前置するので、資格ある閉包のdigestと取り替えられない |

## 設計上の判断

1. **`qualified`を持たせたが常にfalse**。呼び出し側が「観測＝資格」と読み替えないよう、構造体に欄を置いて明示し、`qualificationWithheldReason`に理由を入れる。
2. **外部メンバーの名前はmodule相対**。`/home/.../v5@v5.9.2/dbcheck.go`のような物理パスを名前に入れると、cacheの場所を変えただけで同一内容のdigestが変わる。モジュール識別子＋モジュール内相対パスにした。
3. **`module@version/file`の分割は右端の`@`から探し、次が版の形かを検査**。Module pathにもファイル名にも`@`が現れうるため、単純な`LastIndex`では誤分割する。実際に`github.com/dustin/go-humanize@v1.0.1/big.go`で誤分割を起こし、テストが捕まえた。
4. **未追跡のリポジトリメンバーはエラー**。既存`ResolveSourceClosure`と同じ停止線を維持する。
5. **物理パスはresolver出力から読む**。再構成すると別のcache配置で誤るため、`go list`が報告した`Dir`だけを使う。

## 2026-11-07 の観測

- 環境: WSL2 Ubuntu 22.04、Go `1.27.3 linux/amd64`
- 対象checkout: `43765839f`（`origin/main`）
- 供給: Go download cache形式

| 対象 | repository digest | 外部Module数 | 外部メンバー数 | qualified |
| --- | --- | --- | --- | --- |
| `./cmd/tei`（供給あり） | `sha256:0ef4e294d2a2546228a5a0ef1afce990d94d84f468423c057aad9115224afb76` | 12 | 325 | `false` |
| `./tools/reference-build-authoring-intent`（供給なし） | `sha256:170a6a2400803caba8ffd367d12e302f75f67b29e7039445dbc63f4a75be000f` | 0 | 0 | `false` |

## 検証

- `gofmt -l tools/internal/gobuilder`：差分なし
- `go vet ./tools/internal/gobuilder`：PASS
- `go test ./tools/internal/gobuilder -count=1 -run 'ObserveSourceClosure|ObserveFromList'`：PASS（3件）

## 限界

- 観測は`go list`の選択結果であり、実Buildが同じファイル集合を使うことの証明は別（`ResolveSourceClosure`と実Buildの突合）です。
- 外部Moduleの方針そのものは変えていません。`cmd/tei`の閉包は依然として`rejected`です。方針変更は#1090の所有者の判断です。
- Linux amd64 での1回の観測です。
