# 同じplanを二回buildしてartifactの再現を観測する（§4.21）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.8(c)が問う「**その計画がそのartifactを生んだか**」に対し、lockの一致（`bound`）では答えられないことを§4.11・§4.12が記録しました。本記録は**buildの再現**で答えます。

## なぜlockでは足りないか

`planArtifactBinding=bound`が示すのは「plan fingerprintとartifact digestが一つの検証済みlockとして整合し、digestがディスクと合う」ことです。**そのplanを実行するとそのバイトが出る、という観測ではありません。** 二つの値を結び付けただけであり、builderが申告を偽らないという前提に依存します。

**再現はその前提を観測で置き換えます。** plan Pでbuilderを二回、別のstateで走らせて同じバイトDが出て、ディスク上のartifactがDであれば、そのartifactはPの出力です（決定性が観測されたことと、digest衝突がないことを条件に）。

## 実装

### entry側

`tei self-build`の§4形式に`-rebuild-check`を足しました。

| 段 | 動き |
| --- | --- |
| 一回目のbuild | 従来どおり（`-output`へ書き、no-clobberを保つ） |
| 二回目のbuild | **同じchild実行物**を、**新しい一時state**と**一時出力先**で起動 |
| 比較 | 二回目の申告digest == 一回目の申告digest、かつ ディスク上の`-output` == 一回目の申告digest |

`-artifact-out`と`-state`はentryがchild argvへ足している値なので、**二回目だけを差し替え**ます。呼び手が渡した他の引数（`-plan`・`-plugin`・`-intent`など）はそのまま使うため、条件の違うbuildを比べているのではありません。

**二回目は呼び手の`-output`へ書きません。** 成功した一回目のartifactを検査が上書きしないためです。ただしchildが`-artifact-out`を無視する場合は一回目の出力先へ書くため、**二回目の後にディスクを読み直して**一回目の申告と比べます。

### 出力

`artifactReproducibility`を**常に**出します。

| `state` | 意味 |
| --- | --- |
| `reproduced` | 二回のbuildが同じバイトを出し、ディスク上も一致した |
| `mismatched` | 一致しない。**`ok=false`・`failureCause`** |
| `unperformed` | 実行していない／実行できなかった。**理由を`unperformedReason`へ** |

`-rebuild-check`を渡さない場合も`unperformed`と理由を出します。**欄が無いことを一致と読ませない**ためです（SC0-18と同じ規則）。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `84d780be5`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:942aa1ba9720071c1af2fa1485d08c8d75bc72129d42b82dccb788eeb92607eb` |
| 解決child | `sha256:4e9cff21cb420a18a9b5c7ed96be16d5ac4cbcde1f64c4b8a4f79e425d6218f8` |
| builder child | `sha256:13a9eb8acb1d3e5648abab20dafb097262778bcebc6ae2e71ca26ae4630eaaec` |

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| `planCrossCheck.state` | `matched` |
| `planArtifactBinding.state` | `bound` |
| **`artifactReproducibility.state`** | **`reproduced`** |
| **`firstArtifactDigest`** | **`sha256:7d2f6ff73176e1c817242a225d1499230579fd54ae2c69ff0f845535482f3ae4`** |
| **`secondArtifactDigest`** | **同じ値** |
| **`onDiskDigest`** | **同じ値** |
| `secondChildDigest` | `sha256:942aa1ba…`（**`tei`実行物と同一**。二回目も同じchild実行物を使った） |
| `intentBinding.intentByteCheckMatched` | `true` |
| `l2bClaimed` | **`false`** |

**別々のstateで二回buildして、artifactのバイトが一致しました。** ディスク上のファイルも同じdigestです。

## この実測が言えること

- **同じplanから、同じchild実行物で、独立したstateの二回のbuildが同じバイトを出した**（このホスト・この版で）
- そのバイトは**ディスク上のartifactと一致した**
- したがって**そのartifactはそのplanの出力である**（決定性が観測された範囲で）

**これは§4.8(c)への回答です。** §4.12までの`bound`は「二つの値が整合した」まででしたが、本記録は「そのplanを実行するとそのバイトが出る」を観測しました。

## この実測が言えないこと

- **L2-b（宣言の解釈）。** `l2bClaimed`は`false`のまま
  - (c)は「そのplanがそのartifactを生んだ」ことであり、「entryが宣言を解釈してそのplanを導いた」ことではない。後者は依然として解決childの申告に依存する
  - **したがって(c)が埋まってもL2-bは埋まらない。** 両者を混ぜない
- **決定性の証明。** 二回の観測である。別ホスト・別Go版・別builder版で同じバイトが出ることは確かめていない。**再現性は範囲付きで主張する**（`boundary`が明記する）
- **digest衝突**の不在
- **build後の改変**。二回目の後にディスクを読み直してはいるが、その後の改変は検出しない
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust

## 既定を`-rebuild-check`にするか

**本節は既定を変えません。** 二回目のbuildは時間とI/Oを倍にするため、呼び手の明示が要ります。

ただし**既定にすべき理由はある**。(c)は「そのplanがそのartifactを生んだか」であり、これを観測しない限り`bound`が(c)の代わりに読まれる危険が残ります。既定を切り替えるかは別の契約判断として残します。

## 検証

- `gofmt -l cmd/tei`：差分なし／`go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -run 'RebuildCheck|WithArtifactAndState'`：PASS（**5件**）
  - 再現する場合（`reproduced`、三つのdigestが一致）
  - 二回目だけバイトが違う場合（`mismatched`・`ok=false`）
  - `-rebuild-check`無し（`unperformed`＋理由、digestは空）
  - `withArtifactAndState`が**自分の二つの値だけ**を差し替え、呼び手のargvを書き換えないこと
  - 片方のflagが無いargvを拒否すること
- `go test ./cmd/tei -count=1 -run 'SelfBuild'`：PASS
- 実行1回：`reproduced`、3つのdigest一致、`qualified=false`、`l2bClaimed=false`、exit 0

**NOT RUN**：Windows、別Go版、別builder版、二回目の独立試行全体、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.21
- lockでの束縛：[`TEI_GO_SELF_BUILD_CHILD_LOCK_2026-11-09.md`](TEI_GO_SELF_BUILD_CHILD_LOCK_2026-11-09.md)
- Go buildの再現性（前提となる考え方）：[`TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md`](TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md)
