# SB-N13：自称invokerを実行imageと照合する

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0/C1で用意した[`reference-go-build-stage-driver`](../../tools/reference-go-build-stage-driver/README.md)に、**駆動主体が自称するdigestを実行中のprocess imageと照合する**欄を足した記録です。

**これは照合機能の記録であって、L3の成立ではありません。** `l3Claimed`は常に`false`のままです。

## 解く問題

SB-N13は「Stage1のexecutable digestをmanifestに書くだけで、実際に起動されたのは別SHAのバイナリ」を不合格と定めている（[計画 §35.3](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md)）。

既存のdriverは`driverSelfDigest`を自分で取っていましたが、**自称digestを受け取る欄が無かった**。自称を受け取らない設計は安全側だが、次の二つを区別できない。

- 自称と実imageが一致している（SB-N13の不合格ではない）
- 自称と実imageが違う（SB-N13の不合格）

区別できないままだと、「manifestに書いたdigest」と「実行されたバイナリ」の食い違いを、L3の手前で検出できない。

## 変更

`-claimed-self`を追加した。駆動主体が自称するdigestを渡すと、driverは**実行中のprocess imageのdigest**と比較する。

| 欄 | 意味 |
| --- | --- |
| `claimedInvokerDigest` | 自称digest（指定時のみ） |
| `invokerClaimChecked` | 比較を実施したか |
| `invokerClaimMatched` | 比較結果。`checked=false`のときは意味を持たない |
| `failureCause` | 不一致なら`claimed invoker digest differs from the running process image` |

**自称は単独で受理しない。** `invokerClaimMatched=true`でも`l3Claimed`は`false`であり、自称は「同一性の主張」にすぎない。そのimageがStage1の生成物であることは、Stage1のBuild記録との照合で別に確かめる必要がある。

独立した失敗は`;`で連結して保持するようにした。自称の不一致とartifact照合の不一致が同時に起きても、片方が他方を隠さない。

## 試験（4件を追加）

| 試験 | 確かめていること |
| --- | --- |
| `TestDriveRejectsAClaimedInvokerDigestThatDiffersFromTheImage` | 不一致の自称が`matched=false`・`failureCause`ありになり、`l3Claimed`は`false`のまま |
| `TestDriveRecordsAMatchingClaimWithoutClaimingL3` | 一致する自称は`checked=true`・`matched=true`で記録されるが、L3にはならない |
| `TestDriveKeepsBothIndependentFailureCauses` | 自称不一致とartifact不一致が同時に起きても、両方が`failureCause`に残る |
| （既存7件） | invoker/invokedの記録、artifactの照合、no-clobber、PATH非解決、失敗の記録 |

## 主張しないこと

- L3の成立。`l3Claimed=false`のまま。
- 自称が一致したことの意味。実行imageが自称どおりであることしか言わず、**そのimageがStage1の生成物であること**は言わない。
- 受理・署名・権限・配備。
- 別OS・別toolchainでの一致。Linuxで試験が通ることは確認したが、digestのOS間一致は主張しない。

## 実行した検証

- `gofmt -l tools/reference-go-build-stage-driver`：差分なし
- `go vet ./tools/reference-go-build-stage-driver`：PASS
- `go test ./tools/reference-go-build-stage-driver -count=1`：PASS（10件、WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：Windows実行、Stage1の生成物を駆動主体にするL3実験、二回の駆動試行の比較、CI、trusted full checkout。

## 参照

- 実装：[`tools/reference-go-build-stage-driver`](../../tools/reference-go-build-stage-driver/README.md)
- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.4・§5
- 計画：[`plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md`](../workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md) §35.3 のSB-N13
- 台帳：[`TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md`](TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md)
