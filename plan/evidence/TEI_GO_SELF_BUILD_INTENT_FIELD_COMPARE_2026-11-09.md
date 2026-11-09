# 宣言の他の欄もplanと照合する（§4.15末尾の残作業）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7が残作業に挙げていた**「宣言の他の欄の照合」**を実装した記録です。

**L2-bは依然として成立しません。** 増えるのは「宣言とplanが同じ意味を指す」ことの確認範囲です。

## これまで照合していたもの

builder childは、宣言とplanについて次の3種類だけを比べていました。

| 照合 | 内容 |
| --- | --- |
| subject identity | PluginのIDとversion |
| build座標 | `outputKind`・`target`・`profile` |

## 足したもの

`compareIntentFields` を追加し、次を比べます。

| 欄 | 比べる内容 |
| --- | --- |
| `recipe.kind` / `recipe.entrypoint` | 完全一致 |
| `sources` | **kind と id**（集合として） |
| `dependencies` | **kind と id**（集合として） |
| `secretRequirements` | **名前**（既に`SecretRequirementNames`としてreceiptに載る） |

## 比べないもの（とその理由）

| 欄 | 比べない理由 |
| --- | --- |
| source の `locator`・`digest` | **plan側は導出値**。宣言は参照（`ref`）を書き、planは解決済みのlocatorとbyte digestを持つ。比べると**正しいbuildをすべて不一致と呼ぶ** |
| dependency の `version`・`artifactDigest` | 同じくplan側の導出値 |
| `parameters` | resolverが正規化・順序替えをする可能性がある。不一致が「誤った宣言」と「異なる正規化」を区別しない |

**「比べられる欄」と「比べるべき欄」は違う。** 導出値を比べると常に不一致になるので、比べないことを明示しています。

## 実測（単体試験7件）

| 検証 | 結果 |
| --- | --- |
| 全欄が一致する宣言 | **通る** |
| `recipe.kind` を変える | 拒否、`recipe kind` を名指し |
| `recipe.entrypoint` を変える | 拒否、`recipe entrypoint` を名指し |
| source の kind を変える | 拒否、`source` を名指し |
| source を1つ足す（件数不一致） | 拒否、`sources` を名指し |
| dependency を1つ足す（件数不一致） | 拒否、`dependencies` を名指し |
| plan にだけある source | 拒否（件数不一致） |
| source の locator・digest を変える（導出値） | **通る**（不一致と読まない） |
| dependency の version・digest を変える（導出値） | **通る**（不一致と読まない） |
| childを実プロセスで起動し recipe.kind を変えた宣言 | exit 1、`recipe kind` を名指し、**stdout 0バイト**（reportを出さない） |

最後の行が重要です。**照合に失敗した宣言について「bound」と報告しない**ことを、reportの不在で確かめています。

## 既存fixtureを直した点

`writeFixture`のplanが source ID に `build-child-test-definition` を使い、宣言は `definition` を使っていました。**新しい照合がこれを正しく不一致として検出**したため、fixtureのplanを `definition` に揃えました。

`fixture.go:133` の実装（`Locator: src.Ref`、IDは宣言の `src.ID`）を見ると、実フローでは両者が一致します。**fixtureが実フローと違っていた**のであり、検査を緩めたのではありません。

## 何が増えて、何が増えないか

| 増える | 増えない |
| --- | --- |
| 宣言とplanが同じrecipe・source集合・dependency集合を指すことの確認 | **L2-b**（`l2bClaimed`は`false`） |
| 照合失敗時にreportを出さないことの固定 | 「その宣言がその計画を導く」こと（導出は依然resolver） |
| 比べない欄とその理由の明示 | (c)（artifactがその計画の出力であること） |

## 検証

- `gofmt -l tools/reference-plugin-package-build-child`：差分なし
- `go vet ./tools/reference-plugin-package-build-child`：PASS
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（既存9件＋新規7件）
- `go build ./...`：PASS
- `go test ./cmd/tei -count=1 -run "TestSelfBuild|TestPlan|TestIntent"`：PASS（66秒）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、実プロセスでの全欄一致観測（単体試験のみ）、`parameters`の照合（別の判断が要る）。

## 限界

- **`parameters`は照合していません。** 正規化の意味を決める必要があり、別の判断です。
- **`sources`と`dependencies`は identity だけ**です。導出値（locator・digest・version）を比べないので、「同じ参照を指すが解決結果が違う」場合は検出できません。それは`planFingerprint`と`planRecompute`が担う領域です。
- G4は`UNQUALIFIED`のまま。
