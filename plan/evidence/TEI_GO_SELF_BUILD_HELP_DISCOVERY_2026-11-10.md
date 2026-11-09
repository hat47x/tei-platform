# `tei self-build -h`が実際の文法を示すようになった

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0で足した§4のflagが、**`-h`のどこにも出ていませんでした**。catalog経路は設計文書を読まないと見つからない状態でした。本記録はその修正と、利用者向けの手引きを足したことを記録します。

## 何が起きていたか

`tei self-build -h`の出力は、次の11個だけでした。

```text
-workspace -intent -intent-file -supply -state -output
-child -child-arg -child-arg-file -artifact -claimed-self
```

`-build-catalog`・`-build-catalog-digest`・`-build-source-root`・`-intent-resolver`・`-intent-resolver-arg`・`-expected-plan-fingerprint`・`-rebuild-check`・`-supply-verifier`・`-supply-verifier-arg`が**出ていません**。

## 原因

`cmd/tei/cli_commands.go`の`helpText`が、`self-build`について**flagの一覧を手書き**していました。その一覧が、後から足したflagに追随していませんでした。

手書きだったのは意図的で、「広告しないコマンドなので`tei commands`を指せない」という事情があります。しかし**文法を二か所に書くと、片方だけが育つ**——実際にそうなりました。

## 直したこと

**一つのFlagSetから描くようにしました。**

| 変更 | 内容 |
| --- | --- |
| `writeFlagHelp`を切り出し | 1つのFlagSetを`-h`の形で描く。`wanted`が`nil`なら全flag |
| `selfBuildHelpFlags`を追加 | §4形式のFlagSet（`newSelfBuildOuterFlagSet`）から作り、primitive専用の`-child-arg-file`・`-artifact`・`-claimed-self`を足す |
| `helpText`の`self-build`分岐 | 手書き11行を消し、`writeFlagHelp(&out, selfBuildHelpFlags(), command, nil)`へ |

**flagを足しても、この一覧は自動で追随します。** 二か所に書かないので、また乖離することはありません。

## 直ったこと

`tei self-build -h`に**20個のflag**が出ます。catalog経路と、§4.30の既定も読めます。

```text
  -build-catalog string
    	build catalog JSON the entry resolves against in process (§4.22)
  -build-catalog-digest string
    	sha256 digest of -build-catalog; required with it
  -build-source-root string
    	root the declaration's source references resolve from; required with -build-catalog
  ...
  -rebuild-check
    	build the same plan a second time in fresh state and compare the artifact bytes (§4.21); on by default when -build-catalog is given, and -rebuild-check=false keeps it off (§4.30)
```

`-h`は引き続き**exit 0**です（文法を読む要求であって、引数の誤りではない）。

## 利用者向けの手引きも足した

`reference/plugin-self-build/README.md`に「CLIから自己構築を実行する（`tei self-build`）」を追加しました。

| 書いたこと |
| --- |
| まず`tei self-build -h`を見る |
| catalog 3点セットは同時に指定し、`-intent-resolver`とは排他 |
| `-build-catalog`を渡すと二回目のbuildが既定（`-rebuild-check=false`で断れる・§4.30） |
| `l2b.claimed`・`l2b.unmet`・`l2b.boundary`の意味と、`qualified`は常に`false` |
| catalogは呼び手が用意するデータで、**既存の宣言からは生成できない**（§4.29） |

## 検証

- 新規試験4件：catalog経路のflagが**すべて出る**／primitive形式のflagが**落ちていない**／`-rebuild-check`の行が**§4.30の既定を述べている**／**他のコマンドのhelpが変わっていない**（`view`）
- `go test ./cmd/tei -count=1 -run 'Help|Commands|Advertised|Discovery'`：**PASS**（322秒）
- `go run ./cmd/tei self-build -h`：20個のflag、exit 0
- `doc-reference-probe`：新規の未解決参照なし（既存の`internal/validation.Validate`の1件のみ。本変更前から存在）
- 文字化け検査：該当なし

**NOT RUN**：Windows、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.6・§4.22・§4.30
- 利用者向けの手引き：[`reference/plugin-self-build/README.md`](../../reference/plugin-self-build/README.md)
- 既定の決定：[`TEI_GO_SELF_BUILD_INPROCESS_DEFAULT_EVIDENCE_2026-11-09.md`](TEI_GO_SELF_BUILD_INPROCESS_DEFAULT_EVIDENCE_2026-11-09.md)
