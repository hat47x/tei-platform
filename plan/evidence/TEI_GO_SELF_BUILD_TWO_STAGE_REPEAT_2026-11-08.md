# 二回の独立した駆動試行でも同じstage2成果物になること

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C1で記録した「stage1のartifactがstage2を駆動する」を、**独立した二つの作業ディレクトリ**でも確かめた記録です。

前回の記録（[TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md](TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md)）は1回の実行でした。NOT RUNに「二回の独立試行」が残っていたため、それを埋めます。

## 何を確かめたか

同じchild・同じ対象・同じ供給で、`-work`を別々にした2回の実行を行い、次を比較します。

| 比較 | 意味 |
| --- | --- |
| 両方の`l3Claimed`が`true` | 2回とも「stage1のartifactがstage2のinvoker」まで到達した |
| stage1 artifact digestが一致 | 同じchild・同じレシピで作ったdriverが同じbyteになる |
| stage2 artifact digestが一致 | 再入の結果が同じbyteになる |
| 各回のstage2 invoker = その回のstage1 artifact | 再入の向きが同じ |
| seed Go digestが一致 | toolchainが同じ |

`-work`を分けたのは、同じディレクトリで二回走らせると状態を共有し、「独立した二回」と数えられないためです。

## 実測

- 環境: WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64、および Windows 11 / Go 1.27.0
- 対象: `./tools/reference-go-build-stage-driver`

| 判定 | 結果 |
| --- | --- |
| 1回目の`l3Claimed` | `true` |
| 2回目の`l3Claimed` | `true` |
| stage1 artifact一致 | `true` |
| stage2 artifact一致 | `true` |
| 各回のinvoker = その回のstage1 artifact | `true` |
| seed Go digest一致 | `true` |

両OSで試験が通りました。

## 主張しないこと

- **Seed Trust**：stage0のGo toolchainは明示した外部seedです。TEI自身によるGoコンパイラのself-hostingではありません。
- **受入・Runtime Authority**：digest一致は受理済みでも配備可能でもありません。
- **一般化**：1台・1 toolchain・1 childレシピでの観測です。OS間のdigest一致は主張しません（WindowsとLinuxで同じ試験が通ることは確認したが、同じ値になることは確認していない）。
- **対象**：`reference-go-build-stage-driver`自身であり、`cmd/tei`ではありません。

## 実行した検証

- `gofmt -l tools/reference-self-build-two-stage`：差分なし
- `go vet ./tools/reference-self-build-two-stage`：PASS
- `go test ./tools/reference-self-build-two-stage -count=1`：PASS（WindowsとWSLの両方）
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：`cmd/tei`を対象にした同型の実験、Seed Trust、別toolchainでの再観測。
