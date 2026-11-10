# 照合した木をビルドする（§4.42、案Bの第一歩）

§4.39の案Aには**TOCTOU**が残ります — 計画と作業ツリーを照合してから`go build`するまでの間に、ファイルは変わり得ます。本記録は、**案Bの第一歩**を実装したことを残します。

## 何をするか

`-isolate`を付けると、**計画が名指すsourceだけを新しい木へ写し、その木をビルド**します。

| 段 | 内容 |
| --- | --- |
| 1 | 各sourceを`DirSourceResolver`で読み、**locatorの形を保って**新しい木へ書く |
| 2 | 写した木を**同じresolverで読み直し**、計画のdigestと一致することを確かめる。**違えばビルドしない** |
| 3 | `go.mod`・`go.sum`を作業ツリーから取り、**そのdigestを記録**する |
| 4 | その木で`go list -deps`と`go build`を行う（§4.41の被覆検査もこの木に対して） |

**「確かめたもの」と「ビルドしたもの」が同じ木になります。** 作業ツリーの後からの変更はビルドに届きません。

## 素直でない点：module contextは計画のsourceではない

`go build`には`go.mod`が必要ですが、**これはsourceではない**ので計画は名指しません。そこで**作業ツリーから取り、`moduleFiles`としてdigestを記録**し、**境界に明記**しました。

**これは穴を塞ぎ切っていません。** `go.mod`を後から変えれば依存の解決は変わり得ます。塞ぐにはmodule contextも計画の一部にする（`Dependencies`の拡張）必要があり、未着手です。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestStageSourcesCopiesExactlyThePlannedBytes` | **PASS** |
| `TestStageSourcesRefusesACopyThatDoesNotMatchThePlan`（違えば**拒否**） | **PASS** |
| `TestRunWithIsolateBuildsFromTheStagedTree`（`-isolate`で実build、木のdigestと境界が記録され、被覆検査も通る） | **PASS** |
| `go test ./tools/reference-go-build-from-plan -count=1` | **PASS**（20.6秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## 既定にしない理由

1. **module contextが計画の外にある**ので、隔離しても主張は完全には閉じない。既定にすると「閉じた」と読まれやすい
2. 依存のある対象では、**module解決の合成規則**がまだ無い

**既定にできるのは、module contextを計画の一部にした後です。**

## 言っていないこと

- **TOCTOUを完全には閉じていない**（`go.mod`・`go.sum`は作業ツリーから取る）
- 複数sourceが同じdirectoryを共有する場合の合成規則（今はlocatorごとに独立して写す）
- `vendor/`・`replace`の扱い
- 既定化

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.42
- 案A・案B：[同 §4.39](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 被覆検査：[同 §4.41](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
