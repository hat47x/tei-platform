# entryがcatalogを名指しされて計画をin-processで導く（§4.22 段2）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.22が決めた移行の**段2**を実施した記録です。

## 何をしたか

`tei self-build`の§4形式に三つのflagを足し、**entry自身が宣言から計画を導く**経路を作りました。

| flag | 意味 |
| --- | --- |
| `-build-catalog` | 解決に使うcatalogのJSON |
| `-build-catalog-digest` | そのcatalogの`sha256` digest |
| `-build-source-root` | 宣言のsource参照を解決する根 |

**三つは同時に要求します。** catalogだけ渡されてdigestが無いと「どのデータで導いたか」を記録できず、source rootが無いと参照を読めません（SC0-25）。

**`-intent-resolver`とは排他**です。§4.16が「二つの経路が同じ出力欄を書くと記録の意味が経路に依存する」と警告したため、**一回の試行は一つの導出**にしました。

## 副産物：entryと参照実装が同じ計画fingerprintを出した

**同じ宣言と同じcatalogから、entryと参照実装が同じ`planFingerprint`を導きました。**

| 導出元 | `planFingerprint` |
| --- | --- |
| entry（`-build-catalog`、in-process） | `351c2964169b6e3e14d06ec27f2954b3f97d33571f4edd3f785a632d17c9e350` |
| 参照実装（`-intent-resolver`、child） | **同じ値** |

これは二つの独立した経路が同じ結果に至ることの観測です。§4.19の「経路の違う二つの組み立てが同じfingerprintになる」と同じ種類の観測を、**計画の導出**について得たことになります。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `a0d473ad7`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:1172c8a06a9c2e0616be5ab620738bf49cd751db17a5f3fa94cc2cdd0c2e5a75` |
| builder child | `sha256:7101c06acfc2435a9e76e5f6158184be78f01c60b0c2d7141868b045a5ea5623` |
| 参照実装 | `sha256:3c252c4a41963ed0780e8cc2903fa43ed8b78ac1770e25f039a14923c29a39ee` |
| catalog digest | `sha256:cedde9d564dc82df2a940ec3ace50e2ef80f91cc64f8a58b73f0b4916f4dd8e4` |

| 欄 | 値 |
| --- | --- |
| `ok` / `qualified` | `true` / **`false`** |
| **`planReceipt.derivedBy`** | **`entry-in-process`** |
| `planReceipt.state` | `resolved` |
| `planReceipt.planFingerprint` | `351c2964…` |
| `planReceipt.catalogDigest` / `sourceRoot` | 記録される |
| `planReceipt.authority` | `derived-cache` |
| `planReceipt.l2Claimed` | **`false`** |
| `planRecompute.state` | `matched` |
| `planCrossCheck.state` | **`matched`**（entry導出の計画とbuilderの申告を比較） |
| `planArtifactBinding.state` | **`bound`** |
| `planArtifactBinding.l2bClaimed` | **`false`** |

### digestが違うcatalogは起動しない

`-build-catalog-digest`に`sha256:0000…`を渡すと**exit 2**で止まり、`planReceipt.state=refused`・`planReceipt.status`に理由が残りました。

```text
build catalog digest is sha256:cedde9d5… but the caller declared sha256:0000…
```

## 設計上の判断：導出を経路に依存させない

in-processのreceiptは、**参照実装のreceiptと同じ形**で直列化して`ReceiptJSON`へ入れています。そうしないと、下流の観測（`planArtifactBinding`・`planRecompute`）が「計画が無い」として`unperformed`になり、**経路によって記録の意味が変わる**からです。

実際、最初の実装では`planArtifactBinding`が`unperformed`になりました。receiptの形を揃えたことで`bound`になり、`planRecompute`も`matched`になりました。**同じ出力欄が同じ意味を持つ**ようにしたのが要点です。

## この実測が言えること

- **entryが自分で宣言を読み、名指しされたcatalogを引き、計画を導いた**（`derivedBy=entry-in-process`）
- その計画は**参照実装の計画と同じfingerprint**になった
- **どのcatalogで導いたか**をdigestで記録している（SC0-26の範囲：catalogが正しいとは言わない）
- 下流の観測（`planCrossCheck=matched`・`planArtifactBinding=bound`・`planRecompute=matched`）が**in-process経路でも成立**する

## この実測が言えないこと

- **L2-bの成立。** `l2Claimed`と`l2bClaimed`は`false`のまま
  - §4.22は「段2が終わると『このentryが宣言を解釈した』と言えるようになる」と書いた。**構造としては言えるようになった**（entryが自分で解釈し、自分で導いた）
  - ただし**Claimとして立てるかは別の判断**である。L2-bのClaimは`l2bClaimed`欄の意味と、(a)(b)(c)の同時成立を要求する。本節はそこを動かしていない。**欄を書き換えて「成立した」と読ませることはしていない**
- **catalogの信頼性**（SC0-26）
- **既定経路**。§4.16のR-2据え置きは変わらない。呼び手が明示したときだけin-processになる
- 意味の受理・Runtime Authority・供給のTrust・Seed Trust・Windows・別Go版

## 残る作業

| 項目 | 状態 |
| --- | --- |
| 既定をin-processへ切り替えるか（段3） | **未決。** §4.16のR-1/R-2/R-3は据え置き |
| L2-bのClaimを立てるか | **未決。** 構造は揃ったが、欄の意味を変える判断が必要 |
| `-rebuild-check`を既定にするか | **未決**（§4.21末尾） |
| D-1（catalogの`scope`拡張） | **未決** |

## 検証

- `gofmt -l cmd/tei`：差分なし／`go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -run 'InProcess|ACatalog|BothDerivation'`：PASS（**4件**）
  - in-process導出が成功し、`derivedBy`・`planFingerprint`・catalog digest・source root・authorityを記録し、`l2Claimed`が`false`であること
  - catalogだけ渡した場合の拒否（builderが起動しないことも確認）
  - catalog digestが合わない場合の拒否
  - 二つの経路を同時に指定した場合の拒否
- `go test ./cmd/tei -count=1 -run 'SelfBuild'`：PASS（既存の経路は無変更で通る）
- 実行3回：in-process導出（`ok=true`、`bound`、exit 0）／digest不一致（exit 2）／plan fileなし（builder側で停止し、計画導出は成功している）

**NOT RUN**：Windows、別Go版、二回目の独立試行全体、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.22
- 段1の記録：[`TEI_GO_SELF_BUILD_CATALOG_MOVE_2026-11-09.md`](TEI_GO_SELF_BUILD_CATALOG_MOVE_2026-11-09.md)
- 欄照合の限界（§4.20）：[`TEI_GO_SELF_BUILD_BUILDER_REQUIREMENT_2026-11-09.md`](TEI_GO_SELF_BUILD_BUILDER_REQUIREMENT_2026-11-09.md)
