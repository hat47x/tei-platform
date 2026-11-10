# builder childが宣言を読み、byte突合が実施される（§4.14）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.13末尾が挙げた「より近い一歩」、すなわち**builder childに宣言を読ませて`intentFingerprint`を申告させる**ことを実装し、実測した記録です。

**L2-bは依然として成立しません。** 本記録が増やすのは、**`intentBinding.intentByteCheckPerformed=true`**、つまり「entryが保持した宣言byte」と「builder childが実際に読んだ宣言byte」の突合が**初めて実施された**ことです。

## なぜ重要か

これまで`intentBinding`は`intentByteCheckPerformed=false`のままでした。builder childは`-plan`のファイルだけを読み、宣言を読まなかったためです。entryがいくら宣言のdigestを持っていても、**childが同じ宣言を読んだか**は確かめられませんでした。

childに`-intent`を渡すと、childは共有readerで宣言を読み、**自分が読んだbyteのdigest**と**照合値**を申告します。entryはそれを、自分が保持したbyteのdigestと比べます。**§4.8の(a)と(b)がこれで埋まりました。**

## 何をしたか

### child側

`tools/reference-plugin-package-build-child`に`-intent`（省略可）を足しました。

| 欄 | 意味 |
| --- | --- |
| `intentPath` | 読んだ宣言のパス |
| `intentBytesDigest` | **childが読んだbyteのdigest**（§4.8の(a)） |
| `intentFingerprint` | childが導いた宣言の照合値 |
| `intentBoundToPlan` | 宣言のsubjectがplanのPluginと一致することを**child自身が確かめた**か |
| `intentBindingNote` | 何を主張しないかの文 |

宣言のsubjectとplanのPluginが一致しなければ**artifactを書いた後にreportを出さずexit 1**にします。二つの入力が別のものを指しているのに、両方を「読んだ値」として返すと、**一致し得ない値の比較**を読ませてしまうためです。

`-intent`を渡さない場合はこれらの欄を**出しません**。欄が無いことを合格と読ませないためです。

### entry側

entryは既にchildの`intentBytesDigest`と`intentFingerprint`を読む経路を持っていたため、**`cmd/tei`の変更は不要でした。** childが申告するようになったことで、`intentBinding`が自動的に埋まります。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `b14202ce0`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:94de0f0c178cdce7bf1d972e7d5d382016ae16e6e083dce441478f1c14346c9a` |
| 解決child | `sha256:f47f757a618cb16ddef88ac70a12dd6f9e2a101347da1d39970e6c02174ecc59` |
| builder child | `sha256:4185931c686e11565a7cf32dd4e6f7ef1ef22fd6f22c95361eff96101e34bfad` |
| 宣言のbyte digest | `sha256:d5131c505d7d013d713ae5196e76aaa5a0ff238b51a82378a7786a130082940b` |

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| `intentDeclaration.digest` | `sha256:d5131c50…`（entryが保持） |
| **`intentBinding.intentBytesDigest`** | 同じ`sha256:d5131c50…` |
| **`intentBinding.childIntentBytesDigest`** | **同じ`sha256:d5131c50…`（childが読んだ）** |
| **`intentBinding.intentByteCheckPerformed`** | **`true`** |
| **`intentBinding.intentByteCheckMatched`** | **`true`** |
| `intentDeclaration.intentFingerprint` | `c4cf576f…`（entryが導出） |
| `intentBinding.childIntentFingerprint` | **同じ値**（childが導出） |
| `planReceipt.intentFingerprint` | **同じ値**（解決childが導出） |
| `planCrossCheck.state` | `matched` |
| `planArtifactBinding.state` | `bound` |
| `planArtifactBinding.childLockMatched` | `true` |
| `intentBinding.l2bClaimed` | **`false`** |
| `planArtifactBinding.l2bClaimed` | **`false`** |

**三つの独立した導出（entry・解決child・builder child）が、同じ宣言から同じ照合値になりました。** そしてentryとbuilder childの**byte digestも一致**しています。

## この実測が言えること（§4.8の(a)(b)）

- **(a)** builder childが読んだ宣言byteのdigestを、**child自身が申告した**
- **(b)** それが**entryの保持したbyte**と一致した（`intentByteCheckMatched=true`）
- childは宣言のsubjectがplanのPluginと一致することも自分で確かめた（`intentBoundToPlan=true`）

## この実測が言えないこと

- **L2-b（解釈の成立）。** `l2bClaimed`は両欄とも`false`のまま
  - (a)(b)が埋まったことは、**childが同じ宣言を読んだ**ことを示す。それは「childがその宣言からそのplanを導いた」証明ではない。childは`-plan`を入力として受け取り、宣言は**identityの照合にだけ**使う
  - **(c)「その計画がそのartifactを生んだか」**は`bound`でも出ない（§4.11・§4.12の限界がそのまま残る）
- **childの申告が真であること。** entryが比べられるのは自分が導いた値との一致まで
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust・Windows・別Go版

## 残る段差と次の一歩

(a)(b)が埋まったので、残るのは**childが宣言からplanを導くこと**、つまりI-1b（`Resolve`を`internal/`へ移し、entryまたはchildが計画まで独立に導く）である。ただしこれは`ResolutionContext`の設計判断を伴う。

**より小さい一歩**として、childが「自分が読んだ宣言のsubjectがplanのPluginと一致した」ことに加えて、**宣言の`outputKind`・`target`・`profile`もplanと比べる**ことができる。今はsubject identityだけである。これは§4.8の(c)を直接埋めはしないが、**宣言とplanが同じbuild座標を指すこと**を増やせる。

## 検証

- `gofmt -l tools/reference-plugin-package-build-child`：差分なし
- `go vet ./tools/reference-plugin-package-build-child`：PASS
- `go test ./tools/reference-plugin-package-build-child -count=1`：PASS（**7件**。既存4件＋宣言の申告・別subjectの拒否・`-intent`無しで欄を出さない）
- 実行1回：`ok=true`、`intentByteCheckPerformed=true`、`intentByteCheckMatched=true`、`planCrossCheck=matched`、`planArtifactBinding=bound`、exit 0
- `go test ./cmd/tei -count=1 -run 'SelfBuild'`：PASS（entry側は無変更）

**NOT RUN**：Windows、別Go版、二回目の独立試行、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.8・§4.13
- child：[`tools/reference-plugin-package-build-child`](../../tools/reference-plugin-package-build-child/README.md)
- entryの宣言照合値：[`TEI_GO_SELF_BUILD_ENTRY_INTENT_FINGERPRINT_2026-11-09.md`](TEI_GO_SELF_BUILD_ENTRY_INTENT_FINGERPRINT_2026-11-09.md)
