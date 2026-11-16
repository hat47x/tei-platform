# 壊れた版番号は「互換な plugin が無い」と読めるが、management は区別している（2026-11-17）

- 対象: [`internal/plugin/model.go`](../../../internal/plugin/model.go) の `ParseVersion`・`CompatibleMajor`・`CompareVersion`、[`internal/plugin/management/document_preflight.go`](../../../internal/plugin/management/document_preflight.go) の選択状態
- 位置づけ: **現行挙動の試験による固定と、境界間の食い違いの記録**。製品コードの挙動は変えていません。
- 計測した版: `raw/head.txt`（`origin/main` `df76ca9fa` の作業ツリー＋このPRの追加）。

## 1. 版の受理は1箇所で決まる

```go
func CompatibleMajor(v string, major int) bool {
	p, err := ParseVersion(v)
	return err == nil && p[0] == major        // 解析できなければ false
}
```

resolver はこの関数で候補をふるい落とします。したがって **`Version` が解析できない plugin は、存在しない plugin と同じ扱い**になります。

`CompareVersion` はさらに解析エラーを捨てます。

```go
av, _ := ParseVersion(a)   // エラーを無視 → 0.0.0
bv, _ := ParseVersion(b)
```

## 2. 観測: 壊れた版は「無い」と読める

`Version: "1.0"` の plugin を宣言した環境で `major 1` を要求すると:

```text
--- PASS: TestAMalformedPluginVersionIsReportedAsNoCompatiblePlugin
      ResolveCode = PLUGIN_RESOLVE_NO_COMPATIBLE_PLUGIN
      message = "no plugin tei.a compatible with major 1"
```

**メッセージは選択（requirement）を責めます。** しかし実際に壊れているのは**宣言（catalog の entry）**です。読者がメッセージに従って要求を広げても解決しません。

## 3. management は既に区別している

同じ状況を [`document_preflight.go`](../../../internal/plugin/management/document_preflight.go) は2つに分けます。

| 状態 | 定数 | 意味 |
| --- | --- | --- |
| `version-unknown` | `DocumentSelectionVersionUnknown` | 版が**解析できない**（宣言が壊れている） |
| `version-mismatch` | `DocumentSelectionVersionMismatch` | 版は解析できるが **major が合わない** |

**情報は片方の境界に在って、もう片方で落ちています。** resolver の診断語彙には `version-unknown` に当たるコードがありません。

## 4. 壊れた版が使える版を隠さないこと

同じ catalog に `"1.0"` と `"1.2.3"` が並ぶ場合:

```text
--- PASS: TestAMalformedVersionDoesNotShadowAUsableOne
      resolved tei.a 1.2.3
```

**壊れた entry は `0.0.0` として並ぶので、有効な版より下に落ち**、有効な版が選ばれます。これは実害が無い側の挙動です。

## 5. 固定した挙動（試験）

| 試験 | 固定したこと |
| --- | --- |
| `TestAMalformedPluginVersionIsReportedAsNoCompatiblePlugin` | 壊れた版は `NO_COMPATIBLE_PLUGIN` になり、解析エラーは表に出ない |
| `TestCompareVersionTreatsUnparseableVersionsAsZero` | 解析できない版は `0.0.0` として比較される（5ケース） |
| `TestAMalformedVersionDoesNotShadowAUsableOne` | 壊れた entry が有効な entry を隠さない |
| `TestCompatibleMajorRefusesAMalformedVersion` | 2要素・4要素・空・非数値・負値を拒否する（7ケース） |

負値（`"-1.0.0"`）は当初「`major -1` と一致する」と予想しましたが、`ParseVersion` が `n < 0` を拒否するため**解析自体が失敗**します。**予想を実測に合わせて直した**のはこの1点です（コードは変えていません）。

## 6. この記録が示すこと

- **壊れた宣言と、要件の不一致は、呼出し側にとって別の修復行动です。** resolver は両方を1つのコードに畳んでいます。
- **management 層は既に区別しています。** 語彙を揃えるかどうかは設計判断で、**この記録では決めていません**。
- **壊れた entry が有効な entry を隠さない**ことは確認できました。並び順の副作用で、意図して守られているかは不明です。

## 7. この記録が示さないこと

- **resolver に `version-unknown` を追加すべきか**（未決。Owner の判断です）。
- **`CompareVersion` のエラー無視を直すべきか**（未決。並び順に依存する箇所があります）。
- **`ParseVersion` の受理範囲**（2要素を拒否する現行仕様の是非は扱っていません）。

## 8. 生データ

| ファイル | 内容 |
| --- | --- |
| `run.sh` | 再現手順（4試験の実行と、該当箇所の抜き出し） |
| `raw/head.txt` | 計測した版と Toolchain |
| `raw/anchors.txt` | `CompatibleMajor`・`CompareVersion`・management の選択状態 |
| `raw/tests.txt`・`raw/tests-summary.txt` | 4試験の出力 |

`run.sh` は git を使いません。worktree の `.git` は host のパスを指すファイルで、Windows 上の POSIX shell からは解決できないためです。版は引数で受け取ります。

## 9. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `run.sh` | `5ef7736c1f74b1290f9e94e433514ef17ae31291e074fe86204caa73cc1c6fe3` | 2010 |
| `raw/head.txt` | `e4ffb5c9614b271d9b4d2f162d40924bb812e579d4f9f176790863efb18aba2d` | 73 |
| `raw/anchors.txt` | `f0efbfa60f6c85d8f84feaf3b05b19312353a74880863a14f71c5be746b3e398` | 1282 |
| `raw/tests.txt` | `692ad7cd6a95a28d85b07ddab012dd0bdc06eb06882aa1952a35fd1fd1b6dbf0` | 1435 |
| `raw/tests-summary.txt` | `975353ec8a631c484a141a12bf52675522e51c5c564a248370b8f11c4ae2ac34` | 1430 |
