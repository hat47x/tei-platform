# 計画とartifactの結合を`tei self-build`の出力へ足す（§4.8の(c)）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.10.3が「次に詰める」とした**(c)「artifactがその計画の出力であること」**へ向けた実装の記録です。

**L2-bは依然として成立しません。** `l2bClaimed`は`false`のままです。本記録が増やすのは、**計画fingerprintとartifact digestを一つの検証済み`BuildArtifactLock`として束ね、そのartifact digestをディスクのバイトと独立に突き合わせる観測**です。

## これまでとの違い

`planCrossCheck`は**二つの計画fingerprint**を比べるだけで、artifactについては何も言いませんでした。`(c)`が要るのはそのためです。今回足した`planArtifactBinding`は、次の三つを一つの記録にまとめます。

| 材料 | 出所 |
| --- | --- |
| 計画のidentity（Plugin・outputKind・target・profile） | 解決childのreceiptが持つ`plan`オブジェクト |
| builder childが名乗った`planFingerprint` | childのstdout |
| builder childが名乗った`artifactDigest`と、**ディスクから読み直したdigest** | childのstdout／entry自身の読み取り |

これらから`internal/pluginのBuildArtifactLock`を組み立て、`Validate`と`BuildArtifactLockFingerprint`を通します。

## 実装

| 段 | 内容 |
| --- | --- |
| 1 | 解決childのreceiptから`plan`を取り、identityを読む（要約欄ではなくplan本体から取る） |
| 2 | builder childの`planFingerprint`が解決childの`planFingerprint`と違えば`mismatched`で停止（誤った計画へ束ねない） |
| 3 | `BuildArtifactLock`を`Validate`し、`BuildArtifactLockFingerprint`を取る（digest形式や必須項目の欠落は`unperformed`） |
| 4 | childの`artifactDigest`を**ディスクのバイト**と比べる。ディスクが読めない、または不一致なら`mismatched` |

### 三つの状態

| `state` | 意味 | `ok`への影響 |
| --- | --- | --- |
| `bound` | 検証済みlockが組め、artifact digestがディスクと一致した | 影響なし |
| `mismatched` | 計画が違う、またはartifact digestがディスクと違う | **`ok=false`・`failureCause`** |
| `unperformed` | receiptが無い／childが計画やartifact digestを申告しない／lockが検証を通らない | 影響なし（理由を`unperformedReason`へ） |

**`unperformed`を`bound`として扱いません。** §4.8の他の欄と同じ方針です。

## 単体試験（3件）

| 試験 | 確かめていること |
| --- | --- |
| 一致 | 実リファレンス解決childのreceiptから`bound`・`matched`・lock fingerprint非空・`artifactDigest`とディスクが一致・`l2bClaimed=false`・`qualified=false` |
| 計画不一致 | builderが別の計画を名乗ると`mismatched`・`failureCause`非空・終了1 |
| resolver無し | `planArtifactBinding`・`planCrossCheck`・`planReceipt`のいずれも出さない |

## この記録が言えないこと

- **L2-b（解釈の成立）。** `bound`が言うのは「計画fingerprintとartifact digestが一つのlockとして整合し、そのartifact digestがディスクの実体と一致した」ことだけである
- **builderがentryの保持した宣言からその計画を導いたこと。** lockに載る`planFingerprint`と`artifactDigest`は、**どちらもbuilder childの自己申告**である。entryが独立に確かめたのは、artifact digestがディスクと合うことと、lockが`internal/plugin`の検証を通ることだけである
- **意味の受理・Runtime Authority・供給のTrust・Seed Trust**
- **`planCrossCheck=matched`と`planArtifactBinding=bound`の同時観測**（実プロセスでは未実施。単体試験で別々に確かめている）

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -run 'PlanArtifactBinding'`：3件PASS
- `go test ./cmd/tei -count=1 -run 'SelfBuild|PlanCrossCheck|PlanRecompute|PlanArtifactBinding'`：既存を含め全件PASS
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：実プロセスでの`bound`観測（`matched`との同時観測）、Windows、別Go版、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.8・§4.10.3
- 実装：[`cmd/tei/cli_self_build_plan_artifact_binding.go`](../../cmd/tei/cli_self_build_plan_artifact_binding.go)
- 計画突合：[`TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md`](TEI_GO_SELF_BUILD_PLAN_MATCHED_2026-11-09.md)
