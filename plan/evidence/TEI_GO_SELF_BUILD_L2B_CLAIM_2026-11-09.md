# L2-bのClaimを五条件の述語として立てる（§4.23）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.8が定めたL2-bのClaimを、**観測から計算する述語**として実装し、実測した記録です。

## 何が変わったか

これまで`l2bClaimed`は**定数`false`**でした。これは正しい判断で、**一つの観測をL2-bと読ませない**ためでした。`intentByteCheckMatched`だけでも、`bound`だけでも、`reproduced`だけでもL2-bではありません。

**その原則は変えていません。** 変わったのは、**すべてが同時に成り立つ試行を作れるようになった**ことです。§4.22の段2で「entry自身が解釈した」が入り、五つ目の材料が揃いました。

## 述語

`l2b.claimed=true`は次の**五条件がすべて成立**するときだけです。欠ければ`l2b.unmet`に**欠けた条件をすべて**列挙します。

| # | 条件 |
| --- | --- |
| 1 | entry自身が計画を導いた（`derivedBy=entry-in-process`かつ`resolved`） |
| 2 | 宣言byteの突合が**実施され、一致した** |
| 3 | builderの計画がentryの計画と一致した |
| 4 | plan→artifactの束縛が`bound` |
| 5 | artifactが再現された（`reproduced`） |

**「欄が存在する」で判定しません。** `unperformed`はすべて不成立として扱います（SC0-18の原則を述語にも適用）。

## 単一観測の欄は`false`のまま

`planArtifactBinding.l2bClaimed`・`intentBinding.l2bClaimed`は**据え置き**です。それぞれの欄は「その観測だけではL2-bではない」ことを述べており、**その記述は正しい**。組み合わせの判断は新しい`l2b`欄が持ちます。

**述語はそれらの欄を読みません。** 読むと「`false`を書くことが目的の欄」が主張の入力になり、意味が循環します。試験で固定しました。

## 観測結果

| 項目 | 値 |
| --- | --- |
| 作業領域 | 隔離worktree（`origin/main` = `1a15e7c29`） |
| OS / Go | WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64` |
| `tei` 実行物 | `sha256:2be04236650a856cf45f2cb6c49db6a6f52ed76e2e8404e7d9e9abc97540ec68` |
| builder child | `sha256:7101c06acfc2435a9e76e5f6158184be78f01c60b0c2d7141868b045a5ea5623` |
| 参照実装 | `sha256:3c252c4a41963ed0780e8cc2903fa43ed8b78ac1770e25f039a14923c29a39ee` |
| `planFingerprint` | `6aa9193e64e1aff7c2c0ca08c9d922e42c6df44edc6b42ddd7ddd392bd83bd90`（entryと参照実装で一致） |

### 三つの場合

| 場合 | `ok` | `l2b.claimed` | `l2b.unmet` |
| --- | --- | --- | --- |
| in-process ＋ `-intent` ＋ `-rebuild-check` | `true` | **`true`** | 空 |
| 同じで`-rebuild-check`のみ無し | `true` | `false` | `the artifact was not reproduced from the plan (unperformed)` |
| `-intent-resolver`経由 | `true` | `false` | `the plan was derived by "", not by this entry` |

**一つ目の場合、`planArtifactBinding.l2bClaimed`は`false`のまま**でした。単一観測の欄と組み合わせの欄が、別のことを述べています。

### `l2b.boundary`（出力そのまま）

```text
this attempt read the declaration, interpreted it into a plan itself, matched the builder's declaration bytes to the ones it held, saw the builder report the same plan, and reproduced the artifact from that plan; that is L2-b for this attempt on this host with these binaries. It is not acceptance, not Runtime Authority, not a statement about the catalog's correctness, not a trust judgement about the supply, and not L2-c or L3
```

## このClaimが言えること

- **この試行について**、entryが宣言を解釈して計画を導き、builderが同じ宣言byteと同じ計画を報告し、その計画からartifactが再現された
- 欠けた条件がある場合、**どれが欠けたか**を名指しする

## このClaimが言えないこと

- **受理・Runtime Authority**（`qualified`は`false`のまま）
- **catalogが正しいこと**（§4.22 K-3、SC0-26）
- **供給のTrust・Seed Trust**
- **L2-c・L3**
- **このホスト・このbinaryの外**での成立

## 既定は変えていない

**Claimを立てるには呼び手が明示する必要があります。** 既定経路はchild起動のまま（§4.16のR-2）、`-rebuild-check`も既定off（§4.21末尾）です。したがって**既定の試行では`l2b.claimed`は`false`**になります。

これは意図した設計です。Claimの材料を集める費用（二回目のbuild、in-process化）を払うかどうかは呼び手の判断であり、**既定を切り替えるのは別の契約判断**として§7に残しました。

## 検証

- `gofmt -l cmd/tei`：差分なし／`go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -run 'AssessL2B'`：PASS（**4件**＋表11ケース）
  - 五条件が揃えば主張する
  - 一つずつ外すと、**該当する条件を名指しして**主張しない（11ケース）
  - 空の試行では**五条件すべて**を列挙する（計画receiptの三つの事実は排他の一問なので1件にまとまる）
  - **単一観測の欄の値を変えても結果が変わらない**（述語がそれを読まないことの固定）
- 実行3回：Claim成立（exit 0）／`-rebuild-check`無し（`unmet`が1件）／resolver経由（`unmet`が1件）

**NOT RUN**：Windows、別Go版、全体`go test ./cmd/tei`（CIに委ねる）、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.23
- L2の二層：[同 §4.8](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 材料の記録：[`TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md`](TEI_GO_SELF_BUILD_INPROCESS_RESOLUTION_2026-11-09.md)・[`TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md`](TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md)
