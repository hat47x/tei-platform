# entryが宣言の照合値を自分で導く（§4.13 I-1a）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.13が次の一歩とした**I-1a**を実装し、実測した記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは、**entryが宣言の照合値をchildの申告に頼らず自分で導ける**ことです。

## 何をしたか

宣言を読む実装（`ParseIntent`・`IntentFingerprint`）を`internal/buildintent`へ移し、`tools/`の参照実装と`cmd/tei`の両方が同じパッケージを使うようにしました。

| 項目 | 変更前 | 変更後 |
| --- | --- | --- |
| 宣言を読む主体 | `tools/reference-build-authoring-intent`のみ | **`internal/buildintent`**（両方からimport） |
| `cmd/tei`の宣言読み | 独自の緩い`map[string]json.RawMessage` | **共有の厳密reader** |
| entryの宣言照合値 | 無し（childの申告だけ） | **`intentDeclaration.intentFingerprint`（entryが導出）** |

### 移動したもの・残したもの

| 対象 | 扱い |
| --- | --- |
| `BuildIntent`とその型・`ParseIntent`・`ValidateSourceRef`・`NormalizeIntent`・`IntentFingerprint`・`FingerprintJSON`・`ValidateInputJSON` | **`internal/buildintent`へ移動** |
| `Resolve`・`ResolutionContext`・`DirSourceResolver`・`ParseFixtureCatalog`・`NewFixtureContext` | **`tools/`に残す** |

`tools/reference-build-authoring-intent/intent.go`は**型aliasと薄い委譲**だけになりました。型は複製ではなくaliasなので、宣言の定義はリポジトリに一つだけです。`Resolve`の試験は`tools/`側にそのまま残り、**無変更で通っています**。

`fixture catalog`を移さなかったのは§4.13の判断どおりです。source bytes・builder catalog・dependency catalog・target catalogは参照実装の固定入力であり、製品の概念ではないためです。

## entryの宣言読みが厳密になった

これまで`cmd/tei`の`readIntentDeclaration`は`map[string]json.RawMessage`で一部の欄だけを見ており、**未知のキー・大文字小文字の別名・導出値の手書きを見逃していました**。共有readerに置き換えたことで、entryも同じ規則で拒否します。

**これは挙動の変更であり、試験の入力を直す必要がありました。** 既存の試験fixture `{"formatVersion":1,"subject":{"kind":"plugin","id":"intent.sample"}}` は`version`・`sources`・`builder`・`recipe`・`outputKind`・`target`を持たないため、厳密readerでは拒否されます。`outerIntentFile`（実際の宣言）へ差し替えました。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `6fd42c4f6`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:94de0f0c178cdce7bf1d972e7d5d382016ae16e6e083dce441478f1c14346c9a` |
| 解決child | `sha256:f47f757a618cb16ddef88ac70a12dd6f9e2a101347da1d39970e6c02174ecc59` |
| builder child | `tools/reference-plugin-package-build-child`、`sha256:26c8ed5b6378a26b1396e32fd33d42d2f4651fcc4f7d284eeab2123c25430676` |
| 宣言のbyte digest | `sha256:53d11b4b318f5ca746536f083b308dff5796345cd85499b0c02c1424a7e3c670` |

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| **`intentDeclaration.intentFingerprint`** | **`2508ec16a33cfc262e27d7073f1dd14e245744644aab6a280b0641ec51ee4ca4`**（**entryが導出**） |
| `planReceipt.intentFingerprint` | **同じ値**（解決childの申告） |
| `intentDeclaration.resolved` | `false`（entryは計画を導かない） |
| `planReceipt.planMatchesExpected` | `true` |
| `planCrossCheck.state` | `matched` |
| `planArtifactBinding.state` | `bound` |
| `planArtifactBinding.childLockMatched` | `true` |
| `planArtifactBinding.l2bClaimed` | **`false`** |

**entryが導いた宣言の照合値と、解決childが申告した値が一致しました。** 経路の違う二つの導出（entry自身の`internal/buildintent`と、解決childの同じパッケージ）が同じ値になります。

## この実測が言えること

- entryが、childの申告に依存せずに**宣言の照合値を自分で導いた**
- その値が解決childの申告と一致した
- entryの宣言読みが**参照実装と同じ厳密reader**になった（未知キー・別名・導出値の手書きを同じ規則で拒否）

## この実測が言えないこと

- **L2-b（解釈の成立）。** `l2bClaimed`は`false`のまま
  - entryが導いたのは**宣言の照合値**であり、**計画ではない**。計画の導出は依然childにある
  - `planCrossCheck`の限界（二つの自己申告の一致）は変わらない
  - (c)（artifactがその計画の出力であること）も変わらない
- **宣言byteの突合は未実施。** `intentBinding.childIntentFingerprint`は`<absent>`であり、builder childが宣言を読んでいない（`-plan`のファイルだけを読む）ためである
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust・Windows・別Go版

## 次に詰めること

§4.13が挙げた**I-1b**（`Resolve`も`internal/`へ移し、entryが計画まで独立に導く）である。ただしこれは`ResolutionContext`の設計判断を伴う。

もう一つ、**より近い一歩**がある。entryが宣言の照合値を持つようになったので、**builder childに宣言を読ませて`intentFingerprint`を申告させる**ことができる。そうすれば`intentBinding`が「entryの値」と「builderが実際に読んだ宣言の値」を比べられ、今`<absent>`になっている突合が埋まる。childは`-plan`のファイルだけを読んでいるため、**計画と宣言の結び付け**がここで初めて観測できる。

## 検証

- `gofmt -l cmd/tei internal/buildintent tools/reference-build-authoring-intent`：差分なし
- `go vet ./cmd/tei ./internal/buildintent ./tools/reference-build-authoring-intent`：PASS
- `go test ./internal/buildintent -count=1`：PASS（新設の6件。妥当な宣言・導出値の拒否・意図したpinの許可・未知キー/別名/末尾・順序非依存・不正な宣言）
- `go test ./tools/reference-build-authoring-intent -count=1`：PASS（**無変更**）
- `go test ./cmd/tei -count=1 -run 'SelfBuild'`：PASS
- 実行1回：`ok=true`、`planCrossCheck=matched`、`planArtifactBinding=bound`、exit 0

**NOT RUN**：windows、別Go版、二回目の独立試行、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.13
- 共有reader：[`internal/buildintent`](../../internal/buildintent/intent.go)
- 参照実装：[`tools/reference-build-authoring-intent`](../../tools/reference-build-authoring-intent/)
