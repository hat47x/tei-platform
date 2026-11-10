# 能力宣言に版を付けない（配るのはcatalogだから、§4.35）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7は「能力宣言の版付け」を「必要になった時点で決める」としていました。しかし実測（[先行記録](TEI_GO_SELF_BUILD_DECLARATION_VERSIONING_2026-11-10.md)）は、**その時点では決められない**ことを示していました。本記録は、その未決を**「宣言を誰に配るか」**という問いとして決めます。

## 決定

**能力宣言に版を付けない。配るのはcatalogであり、catalogは既に版を持つ。**

| 対象 | 配るか | 版の扱い（実測） |
| --- | --- | --- |
| **catalog**（entryが読む） | **配る**。呼び手が名指しし、digestで固定する | `formatVersion`を持ち、**未知の版を拒否する** |
| **能力宣言**（ツールの入力） | **配らない**。ツールと同じリポジトリに置く | 版を持たない。キー集合に一致しない入力を拒否する |

| 理由 | 内容 |
| --- | --- |
| 配る側が版を持つ | 版が要るのは「作った時と読む時が離れている」データ。catalogはその条件を満たし、宣言は満たさない |
| 宣言はツールと同じ場所で変わる | ツールを上げるときに宣言も同じリポジトリで直す。**版を名指す相手がいない** |
| **版を足すこと自体が拒否される** | 後から足すと、その時点で宣言を持つ全員にとって破壊的変更になる（実測） |

**宣言を配ることになった場合は、配る前に`formatVersion`を足す。** これは「後から決める」ではなく「**配る前に決める**」であり、判断の時期を「必要になった時点」から**配布の決断時**へ移したことになります。

## 実測（この記録で取った値）

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

| 入力 | 結果 |
| --- | --- |
| 版を書かない宣言 | **成功**。catalogを生成 |
| `formatVersion`をトップレベルに足した宣言 | **`declaration: json: unknown field "formatVersion"`** で拒否 |
| `formatVersion: 2`のcatalog | **`refused: unsupported build catalog format 2`** |

**配る側（catalog）は未知の版を拒否し、配らない側（宣言）は未知のキーを拒否する。** どちらも「分からない入力を成功として扱わない」という同じ原則の現れです。

catalogの版拒否は既存の試験でも固定されています（`internal/buildintent/catalog_test.go`の`unsupported format`ケース）。

## この決定が言っていないこと

- 宣言を**配らない**ことの妥当性は、§4.32の「製品の外のデータ」という決定に依拠する。**配布する製品になった場合は本節を覆す**
- catalogの版を**上げる方法**（版番号の意味、移行手順）は決めていない
- 誰が能力を宣言してよいか（権限）は範囲外のまま

## §7への影響

未チェック項目が**1つ**になりました。

| 残り | なぜ閉じないか |
| --- | --- |
| `darwin`・`arm64`・別のGo版の観測 | **このホストでは実行できない**（macOS無し・`qemu-aarch64`無し） |

## 検証

- 三つの入力を実際に走らせ、上の表の結果を得た
- `go test ./tools/repository-plan-check -count=1`：PASS
- `doc-reference-probe` 未解決参照0件／文字化け検査 該当なし

**NOT RUN**：CI（文書のみ）。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.35
- 版付けの余地の実測：[`TEI_GO_SELF_BUILD_DECLARATION_VERSIONING_2026-11-10.md`](TEI_GO_SELF_BUILD_DECLARATION_VERSIONING_2026-11-10.md)
- catalogを製品の外に置く決定：[同 §4.32](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- ツール：[`reference-build-catalog-authoring/README.md`](../../tools/reference-build-catalog-authoring/README.md)
