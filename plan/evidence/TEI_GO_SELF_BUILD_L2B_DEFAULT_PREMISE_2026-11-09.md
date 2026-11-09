# `l2b.claimed` は `-build-catalog` だけで立つ（§7の前提を実測で訂正）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §7は、`l2b.claimed` について「今は呼び手が **in-process と `-rebuild-check` を明示する必要がある**」と書いています。**§4.30 の実装後、後半が成り立ちません。**

## 何が変わったか

§4.30 は「`-build-catalog` を渡した試行は、`-rebuild-check` を明示しなくても二回目のbuildを行う」と決めました。`-rebuild-check=false` を明示すれば断れます。

したがって `l2b.claimed` の五条件のうち、五つ目（再現）が **`-build-catalog` だけで満たされます**。

## 実測

### `-build-catalog` のみ（`-rebuild-check` を渡さない）

| 項目 | 結果 |
| --- | --- |
| 終了コード | 0 |
| `artifactReproducibility.state` | **`reproduced`**（`unperformed` ではない） |
| `l2b.claimed` | **`true`** |
| `l2b.unmet` | 空 |

**`-rebuild-check` を一度も渡さずに claim が立ちました。**

### `-rebuild-check=false` を明示した場合

| 項目 | 結果 |
| --- | --- |
| `artifactReproducibility.state` | `unperformed` |
| 理由 | **`-rebuild-check=false` を名指し**（「渡されなかった」ではなく「呼び手が断った」） |
| `l2b.claimed` | `false` |

**断り方が区別されています。** 「flagが無い」と「明示的に断った」が同じ `unperformed` でも、理由文が違います。

## §7 の記述のどこが古いか

| 記述 | 現状 |
| --- | --- |
| 「in-process を明示する必要がある」 | **正しい**（`-build-catalog` 一式が要る） |
| 「`-rebuild-check` を明示する必要がある」 | **誤り**。§4.30 以降、catalog経路では既定 |

**正しくは「in-process 経路（`-build-catalog` 一式）を明示すれば足りる」**です。

## 何が残る未決か

**修正後も「既定の試行」の問いは残ります。** 今の状態は:

| 試行 | claim |
| --- | --- |
| 何も渡さない | 立たない（計画を導出しない） |
| `-build-catalog` 一式 | **立つ**（五条件が揃えば） |
| `-intent-resolver` | 立たない（entryが解釈しない） |
| `-build-catalog` + `-rebuild-check=false` | 立たない（呼び手が断った） |

**「何も渡さない試行」でclaimを立てるか**が §7 の本来の問いであり、それは未決です。§4.29 が指摘した「catalogを書く場所を製品に作るか」が前提になります。

## この記録が言っていないこと

- **§7 の判断をしていない。** 記述の訂正と実測だけである。
- 「既定でclaimを立てるべき」。それは §4.29 のcatalog決定を待つ。
- 別ホスト・別Go版での成立（§4.26 の範囲）。

## 検証

- `go vet ./cmd/tei`：PASS
- `go build ./...`：PASS
- `gofmt -l cmd/tei`：差分なし
- `go test ./cmd/tei -run "TestBuildCatalogAlone|TestExplicitRebuildCheckFalse"`：2件PASS
- `go test ./cmd/tei -count=1 -timeout 3600s`：実行中

**NOT RUN**：Windows、別Go版。

## 限界

- 参照fixture（決定論的なchild）での観測です。実装が同じ性質を持つかは別。
- 五条件のうち、宣言byte一致と計画一致は child の申告に依存します。childが偽れば検出できません（§4.23の限度）。
- G4は `UNQUALIFIED` のまま。
