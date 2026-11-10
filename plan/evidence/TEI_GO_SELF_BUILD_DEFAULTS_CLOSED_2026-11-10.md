# 既定に関する残りの判断を閉じた（§4.34）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7に残っていた既定の問いに、**測った材料の範囲で**答えました。いずれも**今の振る舞いを変えない**判断で、覆す条件を添えています。

## 閉じた4つ

| # | 問い | 決定 | 根拠 |
| --- | --- | --- | --- |
| 1 | `-rebuild-check`をすべての試行で既定にするか | **しない** | child経路では費用が倍（参照+110 ms／実build 1.926倍）になるだけで`l2b.claimed`が増えない。catalogを名指しした試行では§4.30が既に既定で行う |
| 2 | 既定の導出経路をin-processへ切り替えるか（R-1） | **据え置き** | entryが名指しされていないcatalogを読むことは§4.22 K-3の取り消しになる。既定はchild経路のまま |
| 3 | `l2b.claimed`を既定の試行で立てるか | **立てない** | #2の帰結。Claimは呼び手が`-build-catalog`一式を渡す**行為** |
| 4 | D-1（catalogの`scope`拡張）を採るか | **採らない** | 外在consumerの前提を測れないまま変えない。失われるのは発見性だけで、`-h`とREADMEで確保した |

## 決定の形

**どれも「今の振る舞いを変えない」判断です。** 変えるのは、§7に残っていた問いを**閉じた**ことです。

| 決定 | 覆す条件 |
| --- | --- |
| 1 | child経路がL2-bを立てられるようになったとき |
| 2 | 呼び手が**一度だけ設定する**catalogの場所が製品の契約として決まったとき（`formatVersion`改版を伴う） |
| 3 | #2を覆すとき |
| 4 | 外在consumerの一覧ができ、`scope`ではなく`effect`を見ることを契約に書けるようになったとき |

## 閉じなかったもの

| 項目 | 状態 |
| --- | --- |
| `darwin`・`arm64`・別のGo版の観測 | **未観測のまま。** このホストでは実行できない |
| 能力宣言の版付け | **未決のまま。** 第二のconsumerが現れた時点で決める |
| `-strong`（§4.25のP-4） | **採らない。** §4.30が「catalogを名指しすれば最強の観測が既定で揃う」を実現したので、強さを選ぶ別flagは要らない |

## §7の残り

**未チェックの項目は2つだけになりました。**

1. `darwin`・`arm64`・別のGo版の観測（このホストでは実行できない）
2. 能力宣言の版付け（第二のconsumerが現れた時点で決める）

## 検証

- `go test ./tools/repository-plan-check -count=1`：PASS
- `doc-reference-probe` 未解決参照0件／文字化け検査 該当なし

**NOT RUN**：CI（文書のみの変更）。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.34
- 費用の実測：[`TEI_GO_SELF_BUILD_REBUILD_CHECK_COST_2026-11-09.md`](TEI_GO_SELF_BUILD_REBUILD_CHECK_COST_2026-11-09.md)・[`TEI_GO_SELF_BUILD_GO_BUILD_REBUILD_COST_2026-11-09.md`](TEI_GO_SELF_BUILD_GO_BUILD_REBUILD_COST_2026-11-09.md)
- D-1の材料：[`CLI_CATALOG_SCOPE_D1_IMPACT_2026-11-09.md`](CLI_CATALOG_SCOPE_D1_IMPACT_2026-11-09.md)
- catalogの出所：[同 §4.32](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
