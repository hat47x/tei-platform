# 「同じバイトを二回出さない」は、条件5のバイト比較で止まる

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.37 の表で、**entry 側で未測定だった最後の1行**を測りました。前の記録（[表の全行の分類](TEI_GO_SELF_BUILD_STOP_TABLE_MEASURED_2026-11-11.md)）で「未測定」としていた行です。

## 実測

child をラッパーにし、**2回目の build で成果物へ1バイト追記し、その追記後の digest を申告**させました。申告とディスクは一致したまま、**二回の成果物のバイトだけが違う**構成です。

| 項目 | 結果 |
| --- | --- |
| 終了コード | **1** |
| `ok` | **`false`** |
| `error.code` | `SELF_BUILD_CHILD_FAILED` |
| **`artifactReproducibility.state`** | **`mismatched`** |
| **`mismatchReason`** | **`the two builds of the same plan produced different artifact bytes`** |
| `firstArtifactDigest` | `sha256:f94f76fd0c390f5fe117089…` |
| `secondArtifactDigest` | `sha256:0f24fa89169c9313113512c…` |
| **`l2b.claimed`** | **`false`** |
| `l2b.unmet` | `the artifact was not reproduced from the plan (mismatched)` |

**原因はバイトの比較そのものです。** 前周に測った「成果物と違うdigestを申告する」場合は**申告とディスクの不一致**で止まりましたが、本記録では**申告はディスクと一致**しており、止めたのは**一回目と二回目のバイトが違うこと**だけです。

## 決定的にしたこと（測り方の切り分け）

最初の試行では `-artifact-out` を `-child-arg` に入れて **entry に拒否されました**。

```text
INVALID_ARGUMENT: -child-arg must not set -artifact-out; this entry supplies it
```

**entry は成果物の出力先を自分で握っています。** child に出力先を選ばせないため、二回目の書き込み先も entry が決めます。**この統制は本記録の測定を可能にしている一方、child が「どこへ書くか」を entry が固定できることを意味します。**

## §4.37 の表：entry 側の行がすべて実測済みになった

| 行 | 実測 |
| --- | --- |
| 宣言と違うバイトを読んだと申告する（条件2） | [止まる](TEI_GO_SELF_BUILD_STOP_TABLE_MEASURED_2026-11-11.md) |
| 計画と違うfingerprintを申告する（条件3） | [止まる](TEI_GO_SELF_BUILD_STOP_TABLE_MEASURED_2026-11-11.md)（条件4も崩れる） |
| 成果物と違うdigestを申告する（条件4） | [止まる](TEI_GO_SELF_BUILD_ARTIFACT_DIGEST_STOPS_2026-11-11.md) |
| **同じバイトを二回出さない（条件5）** | **本記録：止まる** |
| 二回の間に実行物を差し替える | [止まる／プロキシでは止まらない](TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10.md) |
| `-state`を無視して同じstateで二回buildする | [申告が正しければ止まらない](TEI_GO_SELF_BUILD_STATE_SEPARATION_SCOPE_2026-11-11.md) |
| 計画と違うdefinitionを読んでbuildする | **未測定**（child側の停止。§4.36） |

**残る1行は entry の検査ではありません。** child が自分で止まる話で、entry からは観測できません（§4.37「検査で止められないもの」4）。

## 何を言っていないか

- **`-state` の行は「止まらない」ままです。** 本記録はそれを変えません
- **§4.36 の child 側停止は測っていません**
- `linux/amd64` の一回の観測です
- **`qualified` は `false`** です
