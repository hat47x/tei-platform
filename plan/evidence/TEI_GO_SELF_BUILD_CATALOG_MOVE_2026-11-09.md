# catalogの機構を`internal/buildintent`へ移す（§4.22 段1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.22が決めた移行の**段1**を実施した記録です。

## 何をしたか

Builder catalogとSource解決の機構を、`tools/reference-build-authoring-intent`から`internal/buildintent`へ移しました。**名前から`fixture`を外し**、製品側の名前で置いています。

| 移動前（tools） | 移動後（internal/buildintent） |
| --- | --- |
| `FixtureCatalog` | `BuildCatalog` |
| `FixtureProvider` | `BuildProvider` |
| `FixtureDependency` | `BuildDependencyItem` |
| `FixtureTargetEntry` | `BuildTargetEntry` |
| `ParseFixtureCatalog` | `ParseBuildCatalog` |
| `NewFixtureContext` | `NewBuildContext` |
| `DirSourceResolver` | `DirSourceResolver`（同じ名前） |

`tools/reference-build-authoring-intent/fixture.go`は**型aliasと薄い委譲だけ**になりました。参照実装の呼び出し側（`main.go`・3つの試験ファイル）は**無変更**で通っています。

## なぜ`DirSourceResolver`も移したか

source解決はホストI/Oを持つため「参照実装の都合」に見えますが、**symlinkの拒否・root外参照の拒否・綴りの完全一致要求は方針**です。entryが独自実装すると方針が二箇所に分かれるため、§4.22の判断どおり一緒に移しました。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `ce7a9d5d0`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |

| 確認 | 結果 |
| --- | --- |
| `go test ./internal/buildintent -count=1` | PASS（既存＋新設6件） |
| `go test ./tools/reference-build-authoring-intent -count=1` | **PASS（試験は無変更）** |
| `go vet ./...` | PASS |
| `gofmt -l` | 差分なし |

**参照実装の試験が無変更で通った**ことが、この移動が挙動を変えていないことの根拠です。試験は元の挙動を固定しており、移動後も同じコードを通ります。

## 新設した試験（`internal/buildintent`）

移動先が所有する性質だけを固定しました。解決の規則そのものは参照実装の試験が通るので、同じ行を二度試しません。

| 試験 | 固定する内容 |
| --- | --- |
| `TestParseBuildCatalogAcceptsACompleteCatalog` | 完全なcatalogを読める |
| `TestParseBuildCatalogRefusesAmbiguousCatalogs` | 未知キー・大文字小文字の別名・未対応format・provider重複・target重複を拒否する。**二通りに読めるcatalogはdigestで記録して信頼できない** |
| `TestNewBuildContextWiresTheThreePorts` | 三つのportが**一つのcatalogから**答える |
| `TestBuildCatalogIgnoresProvidersOutsideTheBuilderScope` | `authorityScope`が`build-tool`でない項目をbuilderとして選ばない |
| `TestBuildCatalogRefusesAnAmbiguousProvider` | 適合するproviderが二つなら列挙順で選ばず拒否する |
| `TestBuildCatalogFingerprintsIgnoreOrder` | 内容が同じならfingerprintが変わらない |

## この移動が言えること・言えないこと

**言える。** catalogの機構は製品非依存であり、`internal/`に置いて`cmd/tei`からも参照実装からも同じコードを通せる。**挙動は変わっていない**（参照実装の試験が無変更で通る）。

**言えない。** **段2（`cmd/tei`が`-build-catalog`でin-process解決する経路）は未実装**である。したがって「entryが宣言を解釈した」とはまだ言えない。既定経路も変わっていない（§4.16のR-2据え置き）。L2-bは未成立のまま。

## 次

**段2**：`cmd/tei`に`-build-catalog`・`-build-catalog-digest`を足し、`internal/buildintent`の`NewBuildContext`と`Resolve`でin-processに計画を導く経路を作る。§4.22のとおり**両方そろわなければ起動しない**（SC0-25）。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.22
- 移動先：[`internal/buildintent/catalog.go`](../../internal/buildintent/catalog.go)
- 先行例（I-1aの同じ形）：[`TEI_GO_SELF_BUILD_ENTRY_INTENT_FINGERPRINT_2026-11-09.md`](TEI_GO_SELF_BUILD_ENTRY_INTENT_FINGERPRINT_2026-11-09.md)
