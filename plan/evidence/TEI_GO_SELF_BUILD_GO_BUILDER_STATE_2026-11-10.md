# 実buildの再現が、state分離の検査を通るようになった

再現の観測に「**別state**」の検査を足したとき（[記録](TEI_GO_SELF_BUILD_REPRODUCIBILITY_STATE_2026-11-10.md)）、**実際のGo buildを走らせるchildは`stateDir`を申告していませんでした**。本記録は、その結果として実buildの再現が`unperformed`になっていたことと、childが自分の使ったstateを申告するようにして**実buildで`reproduced`が戻った**ことを残します。

## 見つけたこと

`tools/reference-go-build-with-supply`は`-state`を受け取り、その中に`GOCACHE`・`GOMODCACHE`・`GOPATH`・`HOME`を置いて**実際に分離して使っています**。しかし**報告には`stateDir`がありませんでした**。

そのため、state分離の検査を足した後は、このchildを使った再現の観測が次のようになっていました。

| 項目 | 値 |
| --- | --- |
| `artifactReproducibility.state` | **`unperformed`** |
| 理由 | 「childが使ったstateを申告しないので、二回の分離が示せない」 |

**これは検査として正しい振る舞いです。** 分離を示せないのに`reproduced`と言わない、というだけです。ただし**実buildの再現の観測が失われていました**。

## 直したこと

`BuildEvidence`に`stateDir`を足し、**この実行が実際に使ったstate**を報告するようにしました。

**これはP-2（計画を解釈しないchildに`planFingerprint`欄を足す）とは違います。** P-2は「childが導いていない計画を、呼び手が渡した値を転記する」案で、主張を強めないので採っていません（§4.10.2）。**stateの申告は、この実行が何をしたかの事実**であり、計画の解釈ではありません。

## 実測

Windows 11・`go1.27.0 windows/amd64`。`tei self-build -rebuild-check`を、`reference-go-build-with-supply`をchildにして実行しました。childの固定引数は、コマンドライン引用符の問題を避けるため**コンパイルしたwrapper**が設定しています（`-repo`・`-package ./internal/core`・`-supply <GOMODCACHE>/cache/download`・`-go <exact go>`）。

| 項目 | 値 |
| --- | --- |
| `artifactReproducibility.state` | **`reproduced`** |
| `firstStateDir` | `C:\Users\...\gobuild-rebuild-probe\state` |
| `secondStateDir` | `C:\Users\...\tei-self-build-rebuild-596829925\state` |
| `firstArtifactDigest` | `sha256:efe9efa2560526a722cf34e60f6616cd903bd32b591bf4af5005f3ea841d6724` |
| `secondArtifactDigest` | 同上 |
| `onDiskDigest` | 同上 |
| `ok` | **`true`** |

**二つのstateディレクトリは異なり、そのことが記録に残っています。** 実buildのバイトは一致しました。

## 何が変わったか

| 項目 | 前 | 後 |
| --- | --- | --- |
| 実buildの再現 | `unperformed`（分離を示せない） | **`reproduced`（分離を記録の上で示す）** |
| 記録 | stateの記述なし | `firstStateDir`・`secondStateDir` |

**主張は弱まっていません。** 実buildでも「別stateで二回、同じバイト」が**記録に基づいて**言えるようになりました。

## 言えないこと

- **childが申告どおりにstateを使ったこと**の証明（申告である）
- 計画のfingerprint（このchildは計画を解釈しない。§4.10.2のP-2を採らない判断は変わらない）
- `intentBytesDigest`（宣言を読まない）
- **したがってこのchildでは`l2b.claimed`は立たない**（条件2・3が満たされない）。本記録が戻したのは**条件5だけ**です

## 検証

- `tools/reference-go-build-with-supply`の全試験：**PASS**（Windows・54.6秒）
  - WSLのworktreeでは`git`が`.git`のWindowsパスを解決できず失敗する（環境の制約。mainのcheckoutでは発生しない）
- 上の実測（`tei self-build -rebuild-check`）：**PASS**（`ok=true`・`reproduced`）

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.21
- state分離の検査：[`TEI_GO_SELF_BUILD_REPRODUCIBILITY_STATE_2026-11-10.md`](TEI_GO_SELF_BUILD_REPRODUCIBILITY_STATE_2026-11-10.md)
- 実buildの費用：[`TEI_GO_SELF_BUILD_GO_BUILD_REBUILD_COST_2026-11-09.md`](TEI_GO_SELF_BUILD_GO_BUILD_REBUILD_COST_2026-11-09.md)
