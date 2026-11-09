# corrupt-supply分類の実装と実観測

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-0aで追加した`go list -e`の分類に、**`corrupt-supply`**を足した記録です。設計案と前提の観測は[Draft PR #3103](https://github.com/hat47x/tei-platform/pull/3103)（`plan/design/GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md`と`plan/evidence/TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md`）にあります。このPRのcheckoutには未統合のため、本記録から相対リンクでは辿れません。

**これは分類の実装と、既存Go toolchainを使った観測です。** Build・artifact・Lock・受入・権限・G4の判定は動かしません。G4は`UNQUALIFIED`のままです。

## 何を変えたか

SB-0aの四分類（`supported` / `external-unresolved` / `missing-module` / `unknown`）では、**供給が届いていて内容が記録と合わない**失敗が`unknown`に落ちていました。`unknown`は「Sourceか環境を調べる」失敗であり、供給の差し替えという打ち手を指せません。

そこで`gobuilder.ResolveModuleClosurePreflight`の分類に`corrupt-supply`を足しました。

| 分類 | 優先順 | 判定 |
| --- | --- | --- |
| `unknown` | 1 | 未解決packageのエラーが既知の供給系・破損系のどちらの文面にも一致しない、または`go list`が非0終了 |
| `corrupt-supply` | 2 | 未解決packageのエラーが破損系の文面に一致（`flate: corrupt input`、`dir has been modified`、`checksum mismatch` など） |
| `missing-module` | 3 | 未解決packageが供給系の文面に一致 |
| `external-unresolved` | 4 | 外部Module根が選択され、未解決が無い |
| `supported` | 5 | 選択されたModule根が対象Moduleだけ |

`unknown`を最上位に置く理由は変わっていません。認識していない失敗が混ざった列を、供給の差し替えで直る問題として扱わないためです。`corrupt-supply`を`missing-module`より上に置く理由は、**内容が在ること**と**内容が正しいこと**を別の打ち手へ結び付けるためです。

判定は`classifyModuleRoots`（純粋関数）にあり、`go list`の生出力を渡す単体試験で固定しています。実プロセスを起動するのは`ResolveModuleClosurePreflight`だけで、その環境は`BuildPackage`と同じdeny-by-defaultです。

## 観測条件

| 項目 | 値 |
| --- | --- |
| checkout | `D:\GIT\hat47x\tei-wt-sb0b-impl`（branch `feat/gobuilder-corrupt-supply-20261107`、base `0b9c14a83`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`（`/usr/local/go/bin/go`） |
| 供給元 | `~/go/pkg/mod/cache/download`（582MB）と、展開済み`~/go/pkg/mod/golang.org/x/text@v0.41.0`を複製。Host cacheを直接参照しない |
| 固定設定 | `GOPROXY=off`、`GOSUMDB=off`、`GOWORK=off`、`GOTOOLCHAIN=local`、`CGO_ENABLED=0`、`GOFLAGS=-mod=readonly`、状態ごとに独立した`GOMODCACHE`/`GOCACHE`/`GOPATH` |

## 観測1：archive破損は`corrupt-supply`になる

`golang.org/x/text@v0.41.0`のarchive（元 `e63f35daaae749d0ffff97a295ad8f4837a662938a46b7a87f18a88e85a5cbf9`）の1byteを変えた（`51e0501b5e9f169410d43e39b3ae9fc571e4137862b16886258f61bf476f6d29`）。同じ供給でpreflightを実行した。

| 分類 | 解決package | 未解決 |
| --- | --- | --- |
| `corrupt-supply`（旧実装では`unknown`） | 88 | `golang.org/x/text/secure/precis`、`golang.org/x/text/unicode/norm`。いずれも`unzip ...: flate: corrupt input before offset 44` |

`summary`は失敗したimportと、その生のエラーを含みます。`sourceClosureReachable`は`false`です。

## 観測2：展開済みツリーの改変は`go list`では見えない

`golang.org/x/text@v0.41.0/unicode/norm/normalize.go`へコメント行を1行足した（`9e4fdc54…` → `fde42f2b…`）。archiveは元のまま、展開済みツリーだけを変えた状態でpreflightを実行した。

| 分類 | 解決package | 未解決 |
| --- | --- | --- |
| `external-unresolved` | 101 | 0 |

**読み取れること**：`go list`は展開済みツリーを読むが、記録との照合はしない。したがって**分類だけでは供給の改変を検出できない**。これは実装の不足ではなく、`go list`に照合を求めた設計の限界である。
**設計への帰結**：SB-0bの§3.2が求める`go mod verify`と`h1:`照合は、分類の外側に置く必要がある。分類は「どの打ち手か」を選ぶだけで、「供給が正しいか」を判定しない。

## 観測3：同じ状態で`go mod verify`が改変を捕まえる

観測2と同じ状態で`go mod verify`を実行した。

| コマンド | 終了コード | 出力 |
| --- | --- | --- |
| `go mod verify` | 1 | `golang.org/x/text v0.41.0: dir has been modified (/tmp/sb0b-extracted/modcache/golang.org/x/text@v0.41.0)` |
| `go mod download -json golang.org/x/text` | 0 | `Sum: h1:vz/seA0lnX87Othu2f/0L24RcgrXD9/YFTSuGjj3rH8=`（`go.sum`の`h1:`行と一致）、`GoModSum: h1:jvf1O8ajNzZqhSrQBPbutR/EB83Cc0CFrezNQIwbb5M=` |

**読み取れること**：観測2の改変は、既存toolchainで検出できる。`corrupt-supply`の判定文面`dir has been modified`は、この出力に対応する。
**読み取れないこと**：`go.sum`の内容が正しいこと。照合は「期待値と実byteが一致する」ことだけを言う。

## 実装の範囲と、まだ範囲外のこと

実装したのは次です。

- `ModuleClosureCorrupt = "corrupt-supply"`の追加
- `isModuleCorruptionFailure`（既知の破損系文面の判定）
- `classifyModuleRoots`の優先順位変更（`unknown` → `corrupt-supply` → `missing-module` → 外部根 → `supported`）
- 単体試験3件（archive破損、`dir has been modified`、未認識失敗との混在で`unknown`が勝つ）
- 参照ツールのREADMEへ五分類と限界を反映

**まだ範囲外**（設計案§3・§5の残り）：

- `ModuleSupplyManifest`（供給元・`h1:`・展開ツリーdigestの宣言）
- 分類の外側での`go mod verify`／`h1:`照合の自動実行
- 外部Moduleを含むclosureの別値化と、`ResolveSourceClosure`の前提変更
- SBP-24〜30の実試験

したがって**`corrupt-supply`は供給の正しさの証明ではありません**。供給manifestと照合が接続されるまで、この分類は「次に何を直すか」を示すだけです。

## 主張しないこと

- 本変更はTEIのBuild経路を通したBuildを追加しない。artifact・Lock・受入・権限・G4は未変更。
- `corrupt-supply`の判定は`go list`の既知文面に依存し、Go版が文面を変えると`unknown`へ落ちる。落ちる側は安全だが、網羅性は保証しない。
- 観測は1つのGo版・1つのOS・`x/text`1件での確認であり、他のModuleや他の供給元への一般化はできない。

## 実行した検証

- `/usr/local/go/bin/gofmt -l tools/internal/gobuilder`：差分なし
- `/usr/local/go/bin/go vet ./tools/internal/gobuilder`：指摘なし
- `/usr/local/go/bin/go test ./tools/internal/gobuilder -count=1`：PASS
- `/usr/local/go/bin/go test ./tools/reference-go-module-closure-preflight -count=1`：PASS
- 観測1〜3：上記スクリプトを1回ずつ実行

**NOT RUN**：CI、trusted full checkout、SB-0bの供給manifest実装、`ResolveSourceClosure`の外部Module入力、TEI Build経路での実Build。

## 参照

- 設計案と前提の観測：[Draft PR #3103](https://github.com/hat47x/tei-platform/pull/3103)（`GO_MODULE_SUPPLY_FOR_SELF_BUILD_SB0B_2026-11.md`、`TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md`。本branchには未統合）
- SB-0aの記録：[`TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md`](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)
- 判定の正本：[`docs/status.md`](../../docs/status.md)
