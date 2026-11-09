# stage1の生成物がstage2を駆動するか測る（SB-C1）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) の**SB-C1**の参照実装です。§35.3のL3（stage1で生成したバイナリがstage2のBuildを駆動した）を、実プロセスで確かめます。

**L3の主張範囲を明示します。** 成立するのは「同じ1台・同じGo toolchain・同じchildレシピで、stage1のartifactが実プロセスとしてstage2をinvokerとして起動した」ことだけです。Seed Trust、受入、Runtime Authority、別OSへの一般化は含みません。

## 三つの段

```text
stage0 : 外部Go seedで駆動役(reference-go-build-stage-driver)を構築（driven processではない）
stage1 : stage0のdriverがchildを起動し、driver自身を再構築
stage2 : stage1で得たdriver実行ファイルをinvokerとして起動し、同じchildで再構築
```

stage0は`go build`であり、駆動されたプロセスではありません。stage1とstage2だけが「driverがchildを起動した」段です。

## 三つの判定を別々に返す

| 欄 | 意味 |
| --- | --- |
| `stage1ArtifactIsStage2Invoker` | stage2のinvoker digest が stage1のartifact digest と一致するか |
| `sameChildAndArguments` | 二段が同じchild実行ファイル・同じ対象package・同じ供給を使ったか |
| `artifactsEqual` | stage1とstage2のartifact digestが一致するか |
| `l3Claimed` | 上三つがすべて真のときだけ`true` |

`Judge`は三つを独立に判定します。**artifactが一致してもinvokerが違えばL3を出しません**。同じbyteを別のプロセスが作った場合は偽の収束だからです。

子が失敗したときは、子のエラーを`failureCause`へ引用します。対象packageがdriverでない場合、stage2は「`-child`が未定義」で止まり、その理由が記録に残ります。

## 実行

```bash
go run ./tools/reference-self-build-two-stage \
  -repo . \
  -driver-child <reference-go-build-with-supply のビルド済み実行ファイル> \
  -target ./tools/reference-go-build-stage-driver \
  -supply <供給ディレクトリ> \
  -work <作業ディレクトリ>
```

`-driver-child`はPATH解決しません。`-work`を省略すると一時ディレクトリを作り、終了時に消します。終了コードは観測が行えたかだけを示します。

## 実測（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

対象は`./tools/reference-go-build-stage-driver`（自分自身を再構築する）。

| 欄 | 値 |
| --- | --- |
| seed Go digest | `sha256:5f5eb6d0b2fb6aa0bb5d5f8c5f1c944e0de955c925f0111af92a9d8f79680b5b` |
| stage1 invoker | `sha256:8b7757cc86eff60ee0f…`（stage0で作ったdriver） |
| stage1 artifact | `sha256:7a8ce833b39a1adf5bc…` |
| stage2 invoker | `sha256:7a8ce833b39a1adf5bc…`（**stage1のartifact**） |
| stage2 artifact | `sha256:7a8ce833b39a1adf5bc…` |
| `l3Claimed` | `true` |

## 退け方を確かめた入力

| 入力 | 実測 |
| --- | --- |
| 対象を別package（driverでない）にした | `l3Claimed=false`、`stage2 did not produce an artifact: the stage1 artifact could not drive the child (flag provided but not defined: -child)` |
| stage1 artifactをinvokerにしない（`Judge`単体） | `stage1ArtifactIsStage2Invoker=false`で不合格 |
| 別child（レシピ不一致） | `sameChildAndArguments=false`で不合格 |
| artifact不一致 | `artifactsEqual=false`で不合格 |

## 限界

- **Seed Trust**：stage0のGo toolchainは明示した外部seedです。TEI自身によるGoコンパイラのself-hostingではありません。
- **受入・Runtime Authority**：artifact digestの一致は、受理済みでも配備可能でもありません。
- **一般化**：1台・1 toolchain・1 childレシピでの観測です。Windowsでも同じ試験が通りますが、OS間のdigest一致は主張しません。
- **対象の一般性**：この実験の対象は「自分自身を再構築できるdriver」であり、`cmd/tei`ではありません。`cmd/tei`はBuild入口を持たないため、同じ形を移すには入口の決定が先です（[SB-C0設計](../../plan/design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)）。

## 実行した検証

- `gofmt -l tools/reference-self-build-two-stage`：差分なし
- `go vet ./tools/reference-self-build-two-stage`：PASS
- `go test ./tools/reference-self-build-two-stage -count=1`：PASS（WindowsとWSLの両方。`Judge`の4分岐と実装実験）

**NOT RUN**：`cmd/tei`を対象にした同型の実験、二回の独立試行、CI、trusted full checkout、Seed Trust。
