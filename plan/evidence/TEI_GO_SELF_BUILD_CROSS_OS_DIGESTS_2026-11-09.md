# 同じ宣言からWindowsとLinuxで同じ計画とartifactが出る（§4.26）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.21・§4.23・§4.24は、いずれも「**このホスト**での観測」と範囲を限定していました。本記録はその限定を**二つのホスト**に広げます。

## 何を測ったか

同じ宣言・同じsource・同じcatalogから、**Windows（`windows/amd64`）とLinux（`linux/amd64`）**で次を導き、値を比べました。

| 値 | 何が決めるか |
| --- | --- |
| `intentFingerprint` | 宣言の正規化 |
| `sourceClosure` | sourceの相対member pathと内容 |
| `planFingerprint` | 上記＋catalogのfingerprint |
| `artifactDigest` | 計画から `MaterializePluginPackage` が作るバイト |

## 観測結果

| 項目 | Windows | Linux |
| --- | --- | --- |
| OS / Go | `go1.27.0 windows/amd64` | `go1.27.3 linux/amd64` |
| `intentFingerprint` | `22e346d1a14f8cc692cec777b427fc55a0604a808e12da368a87fa0caeb3216c` | **同じ値** |
| `sourceClosure` | `582ec417c6573d403d35df092927b73881167d430507040658b97a2fc3dedaa6` | **同じ値** |
| `planFingerprint` | `e61096a4d77d6bacc87e8b3d74cedf7f3b8d7f4dcc28ad5e584dbf63c42482f3` | **同じ値** |
| `artifactDigest` | `sha256:675dc815e0e717d9e907049283dee012033b8d6c39a892d4a3dfbc894eb56fb4` | **同じ値** |
| artifactのバイト数 | 457 | 457 |

**四つの照合値がすべて一致しました。** 宣言の正規化も、source closureのdigestも、計画のfingerprintも、artifactのバイトも、OSに依存していません。

## L2-bのClaimもWindowsで成立した

`cmd/tei`のエンドツーエンド試験（§4.24）を**WindowsのGoで直接**実行しました。

| 試験 | 結果 |
| --- | --- |
| `TestSelfBuildEstablishesL2BEndToEnd` | **PASS**（24.66秒） |

`l2b.claimed=true`になる試行が、Windowsでも成立します。**SKIPではありません。**

## なぜ一致するのか（設計上の理由）

一致は偶然ではありません。

- `sourceClosure`は**相対member pathと内容だけ**から決まり、rootの場所に依存しない（§4.22で`DirSourceResolver`を移した際の性質）
- `planFingerprint`は`plugin.NormalizeBuildPlan`が列挙順を除いた構造から決まる
- `artifactDigest`は`MaterializePluginPackage`が正規化した定義を`json.Marshal`した結果であり、**改行やパスの混入が無い**

**したがって「ホスト依存が無い」ことは、実装の性質として説明できる。** 実測はその説明を確かめたことになる。

## この実測が言えること

- 同じ宣言から、**WindowsとLinuxの両方で同じ計画と同じartifactが出る**
- L2-bのClaimが**両方のホストで成立する**

## この実測が言えないこと

- **すべてのOSで一致する**とは言えない。**二つのホスト**を観測しただけである（`darwin`・`arm64`・別Go版は未観測）
- **Go版が違う**（1.27.0と1.27.3）。したがって「同じGo版なら一致する」以上のことは言えない。**版の違いをまたいで一致した**のは、むしろ強い観測である
- catalogが正しいこと（§4.22 K-3、SC0-26）、受理・Runtime Authority・Trust
- 既定の試行でClaimが立つこと（§4.23末尾）

## 範囲の更新

この観測を受けて、Claimの`boundary`文言を「**このホスト**」から「**観測した二つのホスト（`linux/amd64` と `windows/amd64`）**」に改めます。**観測していないホストを含めない**という原則は変えません。

## 検証

- `TestSelfBuildEstablishesL2BEndToEnd`（Windows、`go test`）：**PASS**（24.66秒）
- Windows側の測定：`go build`した`tei`の子（参照実装と参照builder child）を直接起動し、digestを比較
- Linux側の測定：同じ宣言・同じcatalogで同じ手順
- `go test ./tools/repository-plan-check -count=1`：PASS
- 文字化け検査：該当なし

**NOT RUN**：`darwin`、`arm64`、Windows以外のGo版の組合せ、全体`go test ./...`（CIに委ねる）、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.26
- 再現の設計：[同 §4.21](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 再現の実測：[`TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md`](TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md)
- Claimの述語：[`TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md`](TEI_GO_SELF_BUILD_L2B_CLAIM_2026-11-09.md)
