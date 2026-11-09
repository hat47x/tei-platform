# stage1の生成物がstage2を駆動する（SB-C1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) の**SB-C1**として、§35.3のL3（stage1で生成したTEI自身のバイナリがstage2のBuildを駆動した）を、実プロセスで確かめる参照実装です。

**L3の主張範囲を明示します。** ここで成立するのは「同じ1台・同じGo toolchain・同じchildレシピで、stage1のartifactが実プロセスとしてstage2を起動した」ことだけです。Seed Trust、受入、Runtime Authority、別OS・別toolchainへの一般化は含みません。

## 三つの段

```text
stage0 : 外部Go seedで駆動役(reference-go-build-stage-driver)を構築（driven processではない）
stage1 : stage0のdriverがchildを起動し、driver自身を再構築
stage2 : stage1で得たdriver実行ファイルをinvokerとして起動し、同じchildで再構築
```

## 三つの判定を別々に返す

| 欄 | 意味 |
| --- | --- |
| `stage1ArtifactIsStage2Invoker` | stage2のinvoker digest が stage1のartifact digest と一致するか |
| `sameChildAndArguments` | 二段が同じchild実行ファイル・同じ対象package・同じ供給を使ったか |
| `artifactsEqual` | stage1とstage2のartifact digestが一致するか |
| `l3Claimed` | 上三つがすべて真のときだけ`true` |

`Judge`は三つを独立に判定し、**artifactが一致してもinvokerが違えばL3を出しません**。同じbyteを別のプロセスが作った場合が、まさに偽の収束だからです。

## 実測（2026-11-07 / WSL2 Ubuntu 22.04・Go 1.27.3 linux/amd64）

対象は`./tools/reference-go-build-stage-driver`（自分自身を再構築する）。

| 欄 | 値 |
| --- | --- |
| seed Go digest | `sha256:5f5eb6d0b2fb6aa0bb5d5f8c5f1c944e0de955c925f0111af92a9d8f79680b5b` |
| stage1 invoker | `sha256:8b7757cc86eff60ee0f…`（stage0で作ったdriver） |
| stage1 artifact | `sha256:7a8ce833b39a1adf5bc…` |
| stage2 invoker | `sha256:7a8ce833b39a1adf5bc…`（**stage1のartifact**） |
| stage2 artifact | `sha256:7a8ce833b39a1adf5bc…` |
| `stage1ArtifactIsStage2Invoker` | `true` |
| `sameChildAndArguments` | `true` |
| `artifactsEqual` | `true` |
| `l3Claimed` | `true` |

stage2のinvoker digestが**stage1のartifact digestと一字一句一致**しています。stage1の生成物が実プロセスとして起動され、そのプロセスがstage2を作った、という最小の再入が成立しています。

## 退け方を確かめた入力

| 入力 | 期待 | 実測 |
| --- | --- | --- |
| 対象を別package（`reference-go-build-reproducibility`）にした | stage1の生成物がdriverではないのでstage2は駆動できない | `l3Claimed=false`、`stage2 did not produce an artifact: the stage1 artifact could not drive the child (flag provided but not defined: -child)` |
| stage1 artifactをinvokerにしない（`Judge`単体） | artifact一致でもL3不可 | `stage1ArtifactIsStage2Invoker=false`で不合格 |
| 別child（レシピ不一致） | L3不可 | `sameChildAndArguments=false`で不合格 |
| artifact不一致 | L3不可 | `artifactsEqual=false`で不合格 |

二つ目の系統は、**失敗の理由を子のエラーから引用**するようにしました。最初は「stage2がartifactを作らなかった」とだけ出て、原因が分からなかったためです。

## 解釈

**わかったこと**：`reference-go-build-stage-driver`を対象に、stage1のartifactがstage2のinvokerとして実起動され、同じchild・同じレシピで同じbyteが得られる。§35.3のL3の**形**は、この対象について成立しています。

**主張しないこと**：

- **Seed Trust**：stage0のGo toolchainは明示した外部seedです。TEI自身によるGoコンパイラのself-hostingではありません。
- **受入・Runtime Authority**：artifact digestが一致しても、受理済みでも配備可能でもありません。
- **一般化**：1台・1 toolchain・1 childレシピでの観測です。Windowsでも同じ試験が通りますが、OS間のdigest一致は主張しません。
- **対象の一般性**：この実験の対象は「自分自身を再構築できるdriver」であり、`cmd/tei`ではありません。`cmd/tei`はBuild入口を持たないため（[SB-C0 §2](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）、同じ形を`cmd/tei`へ移すには入口の決定が先です。

## 実行した検証

- `gofmt -l tools/reference-self-build-two-stage`：差分なし
- `go vet ./tools/reference-self-build-two-stage`：PASS
- `go test ./tools/reference-self-build-two-stage -count=1`：PASS（WindowsとWSLの両方）

**NOT RUN**：`cmd/tei`を対象にした同型の実験、二回の独立試行、CI、trusted full checkout、Seed Trust。
