# Go対象の選択Module閉包の事前観測

[#2831](https://github.com/hat47x/tei-platform/issues/2831)の**SB-0a**（`cmd/tei`の実選択とオフライン供給の分類）に対応する読取り専用の参照実装です。`tools/internal/gobuilder`の共通Go adapterへ、`ResolveSourceClosure`の前に実行する観測を足します。

この実装は**Build・Artifact・受入・権限のいずれも行いません**。`supported`は「選択されたModule根が対象Moduleだけだった」という観測であり、artifact成立やclosure資格を意味しません。

## 解く問題

既存の`gobuilder.ResolveSourceClosure`は、次を一つの失敗へ畳みます。

- 外部Module根が選択された（`moduleDir != root`）
- 外部Moduleの内容が隔離`GOMODCACHE`へ供給されていない
- 対象のGo Source自体が解決できない（構文不正など）

この三つは打つ手が違います。二つ目は供給の追加で解けますが、三つ目はSourceの修正が必要です。畳んだままだと、`cmd/tei`のどのimportが止まっているのかを実装ごとに推測することになります。本Referenceは、`go list -e`の**per-package failure**を保持し、次の四分類で返します。

| 分類 | 意味 | 次の打ち手 |
| --- | --- | --- |
| `supported` | 選択されたModule根が対象Module1つだけ | `ResolveSourceClosure`と`BuildPackage`へ進められる（artifact成立は別） |
| `external-unresolved` | 外部Module根が選択されたが、内容は供給済み | 外部Moduleのbyte・`go.sum`・版の照合（SB-0b）が別途必要 |
| `missing-module` | 選択packageがオフライン解決できなかった（module供給・`go.sum`不足） | 隔離cacheへの明示的な供給（SB-0b） |
| `unknown` | 上記以外の解決失敗、または`go list`自体が非0終了 | 分類しない。Sourceや環境の調査が先 |

`unknown`を`missing-module`へ丸めません。認識していない失敗を、供給の問題として扱うと、Sourceの誤りを「moduleが足りない」と報告してしまいます。

## 観測の作り方

`ResolveModuleClosurePreflight`は`BuildPackage`と同じdeny-by-default環境（`GOPROXY=off`、`GOWORK=off`、`GOTOOLCHAIN=local`、試行ごとの`GOCACHE`/`GOMODCACHE`/`GOPATH`/`HOME`）で`go list -deps -json -e -mod=readonly`を1回だけ実行します。記録するのは次の項目です。

- 対象の`goos`/`goarch`、exact Go実行ファイルのdigest、対象Moduleの物理root
- 解決できたpackage数と、Module根ごとの選択package数
- 解決できなかったpackageのimport path・位置・エラーメッセージ・import stack
- `go.mod`と`go.sum`の存在とdigest（無い場合は`present: false`。空ファイルと不在を区別する）

物理root（`moduleRootDir`、`dir`）はマシン固有の値です。内容の同一性として引用しません。分類と件数は、同じSource・同じ供給・同じGo版で再現できます。

## 実行

```bash
go run ./tools/reference-go-module-closure-preflight -repo . -package ./cmd/tei
```

`-state`を省略すると一時ディレクトリを作り、終了時に消します。独立した2回の観測へ同じ`-state`を共有しません。`-go`でexact Go実行ファイルを指定できます。

終了コードは観測が行えたかだけを示します。分類が`missing-module`や`unknown`でも0を返すのは、分類が**対象の受理可否ではない**ためです。呼び出し側は出力JSONの`classification`と`sourceClosureReachable`を判定に使い、終了コードで「対象が合格した」と読み替えません。

## 2026-11-07 の観測結果（この実行環境）

固定main `4fdad69d9f360ef605930cc298f2a72efdfdee8e` の作業checkout、Go `go1.27.0`（windows/amd64）、`-trimpath`などBuildPackageと同じ固定設定での1回の観測です。Go実行ファイルのdigestは`sha256:7d828191ba32519a9c9361789ab647486236ed45c660889196c7770a8ff1985c`、`go.mod`は`sha256:ad0264180541bc910132f3f9183deb81539c89b3f4d88ba68fd2512d0382f19c`、`go.sum`は`sha256:d5967ad9f867efa398e7fa9afaccb61819a3e8f520aaee7bbb4a5069402c55d3`でした。

| 対象package | 分類 | 解決package | 解決できなかったimport |
| --- | --- | --- | --- |
| `./cmd/tei` | `missing-module` | 65 | `github.com/jackc/pgx/v5`（`cmd/tei/dbcheck.go`）、`golang.org/x/text/unicode/norm`（`internal/command/reference_store.go`）、`modernc.org/sqlite`（`internal/command/lifecyclesqlite/sqlite.go`） |
| `./tools/reference-build-authoring-intent` | `supported` | 30 | なし |

`cmd/tei`側の3件はすべて`module lookup disabled by GOPROXY=off`であり、Sourceや構文の問題ではありません。つまり**現行の隔離観測では`cmd/tei`のSource closureへ到達できません**。これはSB-0b（検証済みbyte・`go.sum`・版・オフライン事前供給の最小Contract）が先に要ることを示します。`cmd/tei`のBuildが常に失敗するという主張ではありません。Host cacheを使って供給すれば進む可能性はありますが、Host cacheの利用は現行のdeny-by-default方針と両立しません。

## 限界

- 一度の`go list`は、その時点のModule供給とGo版での観測です。別の供給・別のGo版・別OSへの一般化はできません。
- 観測はGoコンパイラの選択を`go list`に委ねます。実際のBuildが使うファイルと一致することを保証するのは`ResolveSourceClosure`と実Buildの照合です。
- 外部Moduleの供給・`go.sum`と実byteの照合・Trustは**未実装**（SB-0b）。本Referenceは、その前提となる観測だけを行います。
- `missing-module`の判定は、`go list`のエラーメッセージの既知の文面に依存します。Go版が文面を変えた場合は`unknown`へ落ちます。落ちることは安全側ですが、分類の網羅性は保証されません。
