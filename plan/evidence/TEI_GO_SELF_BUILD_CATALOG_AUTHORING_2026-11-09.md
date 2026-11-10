# 能力宣言からcatalogを作り、Claimまで一本で通した（§4.33）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.32は、catalogを**製品の外のデータ**と決めました。その決定が残した「誰が書くか」を埋めた記録です。

## 作ったもの

`tools/reference-build-catalog-authoring`。**能力宣言**を読み、**artifactのdigestを実ファイルから計算して**、catalog JSONを書きます。

| 欄 | 扱い |
| --- | --- |
| `plugin`・`version`・`implementation`・`contract`・`contractMajor`・`recipeKinds`・`targets`・`profiles` | 宣言から写す |
| `authorityScope` | **書かせない。** 常に`build-tool` |
| `artifact` | `-root`からの**相対パス**。このファイルのdigestが`artifactDigest`になる |
| `artifactDigest` | **書かせない。** 書くと拒否する |

**digestを宣言させないことが要点です。** 手で書いたdigestを許すと、リポジトリに無いbuilderを名乗るcatalogが作れてしまう。ツールは実ファイルを読み、そのバイトのdigestを入れます。

## 拒否するもの

| 入力 | 理由 |
| --- | --- |
| 宣言の`artifactDigest` | 読んでいないバイトを名乗れなくする |
| `-root`の外を指す`artifact`（`../…`・絶対パス） | ルートが持たないファイルをcatalogに載せない |
| 存在しない・通常ファイルでない`artifact` | 同上 |
| 既存の`-output` | 上書きしない |
| 組み立てたcatalogが`ParseBuildCatalog`で読めない | entryが読めないcatalogを「catalogがある」と誤解させない |

## 実測：宣言からClaimまで一本で通った

| 段 | 結果 |
| --- | --- |
| 能力宣言を書く | 1 provider |
| ツールでcatalogを作る | `catalogDigest=sha256:6d1e592923900d1c45aae897efdf94e9b60cc315abdf7144544094719e0955ee` |
| ツールが計算した`artifactDigest` | `sha256:84db775286d27d66a3e8ceda2364e95b64e45d239c4c35cc318248470b4b5f6f` |
| **実ファイルのdigest**（`sha256sum`） | `84db775286d27d66a3e8ceda2364e95b64e45d239c4c35cc318248470b4b5f6f`（**一致**） |
| 参照実装がこのcatalogを解決 | `planFingerprint=a6b756b8ef50ebe289bf0dacdd572acfdc34e5b94534e0aaa4a70cedf02fda12` |
| **`tei self-build`（`-rebuild-check`を書かない）** | `derivedBy=entry-in-process`・`artifactReproducibility=reproduced`・**`l2b.claimed=true`**・`qualified=false`・`ok=true` |

**§4.29で「catalogを書く場所が無い」と分かってから、製品の契約を一つも変えずに、宣言からL2-bのClaimまでが一本で通りました。**

## この形式は製品の形式ではない

能力宣言には**互換性の約束も`formatVersion`もありません**。製品が読むのは、このツールが出力するcatalogだけです。宣言の形を変えても、catalogの形（`BuildCatalog`）を変えなければ製品には影響しません。

## この実測が言えないこと

- **catalogが正しいこと**（K-3・SC0-26のまま）。名指しされたデータを使った記録しかない
- **誰が能力を宣言してよいか**（権限）。catalogは「buildできると名乗るデータ」で、名乗りの妥当性は範囲外
- **builderの実装が正しいこと**。digestを計算するだけで中身は見ない
- 既定の試行でのClaim（§4.30のまま。catalogを渡す一手が要る）
- 能力宣言の**版付け**。今は約束しない

## 検証

- `go test ./tools/reference-build-catalog-authoring -count=1`：**6件** PASS
  - artifactのdigestが実ファイルから計算される（宣言の値ではない）
  - 宣言の`artifactDigest`を拒否
  - `-root`外の`artifact`を拒否
  - 存在しない`artifact`を拒否
  - 既存の`-output`を上書きしない
  - 出力に呼び手が要る`catalogDigest`が入る
- 実行1回：上の表のとおり（エンドツーエンド）
- `go test ./tools/repository-plan-check -count=1`：PASS（`tools/README.md`の索引も更新）
- `doc-reference-probe` 未解決参照0件／文字化け検査 該当なし

**NOT RUN**：Windows、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.33
- catalogを製品の外に置く決定：[同 §4.32](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 使い方：[`reference-build-catalog-authoring/README.md`](../../tools/reference-build-catalog-authoring/README.md)
- 欄の宣言面が無いこと：[`TEI_GO_SELF_BUILD_CATALOG_FIELD_PROVENANCE_2026-11-09.md`](TEI_GO_SELF_BUILD_CATALOG_FIELD_PROVENANCE_2026-11-09.md)
