# Resolveを`internal/`へ移す（§4.13 I-1b）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7が残作業に挙げていた**I-1b（宣言から計画まで独立に導く経路）**の実装記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは「entryが計画を自分で導ける」土台です。

## 何が難しかったか

§4.13はI-1bを採らない理由として**「`ResolutionContext`を移すとfixture catalogが製品の一部に見える」**を挙げていました。`ParseFixtureCatalog`・`NewFixtureContext`は名前のとおり参照実装の固定入力であり、`internal/`へ置くと「TEIの製品機能」と読まれます。

## どう解いたか

**interfaceと`Resolve`だけを移し、fixture実装は`tools/`に残しました。**

| 移した（`internal/buildintent/resolve.go`） | 残した（`tools/reference-build-authoring-intent/`） |
| --- | --- |
| `SourceResolver`・`BuilderResolver`・`DependencyResolver`・`TargetCatalog`（interface 4つ） | `DirSourceResolver`（fixtureのSource解決） |
| `ResolutionContext`・`ResolutionContextFingerprints` | `ParseFixtureCatalog` |
| `ResolvedBuildReceipt`・`BuilderRequest`・`BuilderResolution`・`ResolvedSource` | `NewFixtureContext` |
| `Resolve`・`MarshalReceipt`・`VerifyDerivedReceipt` | fixture catalogのJSON |

**§4.13の反対理由はcatalogについてであり、portについては当たらなかった**わけです。`internal/buildintent`は「計画を導くのに必要な口」だけを定義し、その口に何を挿すかは呼び手が決めます。fixtureは口の一つの実装にすぎません。

`tools/`側の`resolve.go`は新しいパッケージへの薄い別名（type alias と関数の委譲）になりました。既存の`Resolve`利用箇所と試験はそのまま通ります。

## 増えること・増えないこと

| 増える | 増えない |
| --- | --- |
| **呼び手が計画を自分で導ける**（subprocessを起動しない） | **L2-b**。`l2bClaimed`は`false`のまま |
| `Resolve`の決定性を直接確かめられる | artifactがその計画の出力であること（(c)） |
| 二つの導出を比べられる（同じ入力なら同じreceipt） | 意味の受理・Runtime Authority・供給のTrust |
| 派生receiptの改竄検出（`VerifyDerivedReceipt`） | — |

## 実測

`internal/buildintent/resolve_test.go`に3件を追加しました。fixture catalogではなく**テスト用のport実装**を挿しています。

| 検証 | 結果 |
| --- | --- |
| 計画の導出 | `Authority=derived-cache`、`Plan.Plugin.ID`が宣言のsubjectと一致、`PlanFingerprint`と`IntentFingerprint`が**別の値** |
| 決定性 | 同じ入力で2回導出し、`PlanFingerprint`・`ResolutionContextFingerprint`・marshal結果が**一致**。`VerifyDerivedReceipt`が通る |
| 改竄検出 | `planFingerprint`を書き換えたreceiptは`VerifyDerivedReceipt`が**拒否** |
| 不完全なContext | `Targets`を`nil`にすると**拒否** |

既存の`tools/reference-build-authoring-intent`の試験（fixture catalogを使うend-to-endのreceipt検証を含む）も**そのまま通ります**。

## 何を言えないか

- **`Resolve`が`internal/`にあることは、entryがそれを呼んだことを意味しません。** entryの`-intent-resolver`は依然としてchild起動で、subprocess経由の経路が既定のままです。I-1bが可能にしたのは**選択肢**であり、既定の切り替えは別の判断です。
- **(c)は変わらない。** `Resolve`が計画を導いても、その計画がartifactを生んだことは別の観測です。
- **`planCrossCheck`の限界も変わらない。** 二つの自己申告の一致という性質は、導出の主体を変えても残ります。

## 検証

- `gofmt -l internal/buildintent tools/reference-build-authoring-intent`：差分なし
- `go vet ./internal/buildintent ./tools/reference-build-authoring-intent ./cmd/tei`：PASS
- `go build ./...`：PASS
- `go test ./internal/buildintent -count=1`：PASS（12件。既存9件＋新規3件）
- `go test ./tools/reference-build-authoring-intent -count=1`：PASS（既存試験が別名経由で通る）
- `go test ./cmd/tei -count=1 -timeout 900s`：PASS（342秒）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：entryの実際の経路を`Resolve`直呼びへ切り替える判断、Windows、別Go版、CI。

## 限界

- **entryの既定経路は変わっていません。** `-intent-resolver`によるchild起動のままで、`internal/buildintent.Resolve`を`cmd/tei`が呼ぶ経路は未実装です。
- この記録は**移動と検証**であり、Claimの追加ではありません。
- 供給のTrust・署名、Seed Trust、`go.sum`のTrust根拠は未検証。G4は`UNQUALIFIED`のまま。
