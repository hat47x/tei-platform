# builder childが宣言とplanのbuild座標を照合する（§4.15）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.14末尾が「より小さい一歩」として挙げた、**宣言とplanのbuild座標（`outputKind`・`target`・`profile`）の照合**を実装し、実測した記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは、**宣言とplanが同じbuildを指していること**の観測です。

## なぜ要るか

§4.14でchildは**subject identity**（Plugin のIDとversion）だけをplanと比べていました。identityが一致しても、**別のbuild**を指している場合があります。

| 一致する | 一致しないことがある |
| --- | --- |
| Plugin ID・version | `outputKind`（`runtime-artifact` と `plugin-package`） |
| | `target`（`host/linux/amd64` と `host/windows/amd64`） |
| | `profile`（`release` と `debug`） |

同じPluginの別のbuildを指す二つの入力を「一致した」と読ませると、**artifactの座標が宣言と食い違う**のに気づけません。

## 何をしたか

childが宣言を読むとき、identityに加えて**三つの座標を一つずつ比べ**、違えば**artifactを書いた後にreportを出さずexit 1**にします。どれが違ったかを名指しするので、呼び手は原因を追えます。

```text
the declaration's target "host/windows/amd64" does not match the plan's "host/linux/amd64"
```

**`profile`の省略は一致として扱いません。** 宣言が`profile`を書かないことは「planの`profile`と同じ」ではなく「profileを宣言していない」です。空文字とplanの値が違えば拒否します。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `a60f32d4d`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:94de0f0c178cdce7bf1d972e7d5d382016ae16e6e083dce441478f1c14346c9a`（**無変更**） |
| 解決child | `sha256:f47f757a618cb16ddef88ac70a12dd6f9e2a101347da1d39970e6c02174ecc59` |
| builder child | `sha256:b72375a925bc21f1ea217fe884cf8958fd69a5298b4b65ccf94855b8ed70752e` |
| planの座標 | `plugin-package` / `host/linux/amd64` / `release` |

### 座標が一致する場合

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| `intentByteCheckPerformed` / `Matched` | `true` / `true` |
| `childIntentFingerprint` | `650dac9cdbfa49af7334ac895969f9366718ca8198fa6e1e54941f4ab3cbfba9`（entryの値と一致） |
| `planArtifactBinding.state` | `bound` |
| `planArtifactBinding.l2bClaimed` | **`false`** |

### 座標が違う場合

宣言の`target`だけを`host/windows/amd64`に変えたところ、**childがexit 1で止まり、stdoutは0バイト**でした。stderrは違った座標を名指ししています。

| 確認 | 結果 |
| --- | --- |
| child の終了コード | `1` |
| child の stdout | **0バイト**（reportを出していない） |
| child の stderr | `the declaration's target "host/windows/amd64" does not match the plan's "host/linux/amd64"` |

## この実測が言えること

- childが宣言とplanの**三つの座標を照合**し、違えば**reportを出さずに止まる**
- 一致する場合は§4.14の観測（(a)(b)の突合、`bound`）がそのまま成立する

## この実測が言えないこと

- **L2-b（解釈の成立）。** `l2bClaimed`は`false`のまま
  - 座標の一致は「宣言とplanが**同じbuildを指す**」ことを示すが、「**childがその宣言からそのplanを導いた**」ことではない。childは`-plan`を入力として受け取る
  - **(c)「その計画がそのartifactを生んだか」**は`bound`でも出ない（§4.11・§4.12の限界がそのまま）
- **宣言の他の欄**（`sources`・`builder`・`recipe`・`dependencies`・`parameters`・`secretRequirements`）は照合していない。座標は三つだけである
- childの申告が真である保証はない
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust・Windows・別Go版

## 残る段差

**(a)** childが宣言byteを申告（§4.14）、**(b)** entryの保持byteと一致（§4.14）、**build座標**の照合（本節）まで来た。残るは**I-1b**、すなわち**childが宣言から計画を導くこと**である。

座標の照合を**さらに広げる**選択肢もある（`sources`・`recipe`・`dependencies`）。ただしこれらは`BuildPlan`の内容そのものであり、照合を広げるほど**childが宣言からplanを導く処理に近づく**。区別が曖昧になると「導出した」と「照合した」を混同させるため、**I-1bとしてまとめて扱う**方が読ませ方を誤らせない。

## 検証

- `gofmt -l tools/reference-plugin-package-build-child`：差分なし
- `go vet ./tools/reference-plugin-package-build-child`：PASS
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（**9件**。座標3種の不一致と`profile`省略の拒否を追加）
- 実行2回：座標一致（`ok=true`、`bound`、exit 0）、座標不一致（exit 1、stdout 0バイト）
- `go test ./cmd/tei -count=1 -run 'SelfBuild'`：PASS（`cmd/tei`は無変更）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows、別Go版、二回目の独立試行、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.14
- child：[`tools/reference-plugin-package-build-child`](../../tools/reference-plugin-package-build-child/README.md)
- 宣言の突合：[`TEI_GO_SELF_BUILD_CHILD_DECLARATION_2026-11-09.md`](TEI_GO_SELF_BUILD_CHILD_DECLARATION_2026-11-09.md)
