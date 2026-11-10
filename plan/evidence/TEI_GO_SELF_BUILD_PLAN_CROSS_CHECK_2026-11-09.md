# `tei self-build`の計画突合（§4.8の三つ目の観測）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.8が「一つの出力に載せる」と定めた三項目のうち、**両者の突合**を実装した記録です。

**L2-bは依然として成立しません。** この記録が増やすのは「解決childとbuilder childが同じplan fingerprintを名乗ったか」という**二つの自己申告の一致**です。

## 何を足したか

`tei self-build`の出力に`planCrossCheck`を足しました。解決child（`-intent-resolver`）の`planFingerprint`と、builder childが報告した`planFingerprint`を比べます。

| `state` | 意味 |
| --- | --- |
| `matched` | 両者が同じplan fingerprintを名乗った |
| `mismatched` | 名乗りが違う。`failureCause`へ記録し、`ok=false` |
| `unperformed` | 片方が報告していない。`unperformedReason`に理由 |

`unperformed`は**一致として扱いません**。builderが報告しない場合は「`-child-arg`で報告引数を渡していない」と明記します。resolverを指定しない場合は`planCrossCheck`自体を出しません（解決計画が無いことを空欄で誤解させないため、この場合は`planReceipt`も出ません）。

## 設計上の判断

1. **突合は両方のchildが終わった後に行う。** 事前条件ではないので、不一致は「停止しなかったが記録された発見」として`ok=false`と`failureCause`に載せます。事前に止められるのは解決child側（`-expected-plan-fingerprint`の不一致）だけです。二つは別の検査であり、混ぜません。
2. **builderの申告は`-child-arg`経由で受け取る。** entryがchildの引数を発明しない方針（#3571）を維持します。
3. **境界文を必ず出す。** `planCrossCheckBoundary`は「二つの自己申告の一致であり、builderがその宣言からその計画を導いた証明でも、artifactがその計画の出力である証明でもない」と述べます。

## 実測（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

解決childは`tools/reference-build-authoring-intent`（`testdata/{intent.json,catalog.json,src}`）、builder childは`tools/reference-go-build-with-supply`（対象`./tools/reference-build-authoring-intent`）。

| 実行 | 結果 |
| --- | --- |
| 実プロセス（builderは計画を報告しない） | `ok=true`、`state=unperformed`、`resolverPlanFingerprint=38c999fb…`、`builderReportedPlanFingerprint`は無し、`l2bClaimed=false` |
| 単体試験：両者が一致 | `state=matched`、`matched=true`、`l2bClaimed=false`、`boundary`非空、`qualified=false` |
| 単体試験：不一致 | `state=mismatched`、`matched=false`、`ok=false`、`failureCause`に記録 |
| 単体試験：builderが報告しない | `state=unperformed`、`unperformedReason`非空、`ok`は保たれる |
| 単体試験：resolver無し | `planCrossCheck`は出ない |

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -timeout 900s`：PASS（342秒）
- `go test ./cmd/tei -run TestPlanCrossCheck`：4件PASS
- `go test ./tools/repository-plan-check -count=1`：PASS（mainの未解決リンク2件は前周までに解消済み）

## 限界（L2-bを主張しない理由）

- **builderの`planFingerprint`は自己申告。** `reference-go-build-with-supply`はこの欄を返さないため、実測の実プロセスでは`unperformed`になる。一致を実物で観測するには、計画を報告するbuilderを用意する必要がある（単体試験では報告するfixtureで確かめた）。
- **artifactと計画の結び付きは未観測。** 突合は「同じ計画を名乗った」までで、「artifactがその計画の出力である」ことは言えない。
- 供給のTrust・署名、`go.sum`のTrust根拠、Seed Trustは未。catalogへ載せていない（D-3）。G4は`UNQUALIFIED`のまま。
