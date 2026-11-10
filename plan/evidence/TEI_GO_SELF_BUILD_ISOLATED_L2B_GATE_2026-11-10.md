# 隔離ビルドでもL2-bが立つことを回帰試験で確かめた（§4.44）

§4.42・§4.43で隔離ビルドを作りましたが、**その経路で五条件が立つかは確かめていませんでした**。本記録は、回帰試験を隔離ビルドへ切り替え、**隔離した木からビルドしても`l2b.claimed=true`が立つ**ことを固定した結果です。

## 変えたこと

| 対象 | 変更 |
| --- | --- |
| 宣言 | `go-module`のsourceとして`go.mod`・`go.sum`を追加（§4.43の規則） |
| child args | `-isolate`を追加 |
| 試験 | 主張に加えて、**childが計画を読むbuilderであること**（`childSchema`）を確かめる |

## 実測

| 項目 | 値 |
| --- | --- |
| `TestSelfBuildEstablishesL2BFromARealGoBuild` | **PASS**（14.5〜71.3秒。環境の負荷で変動） |
| `l2b.claimed` | **true** |
| `planCrossCheck` / `planArtifactBinding` / `artifactReproducibility` / `definitionBinding` | すべて成立 |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## この試験が示していないこと

**隔離が実際に行われたことを、entryは見ていません。** entryはchildのstdoutをdigestとして記録しますが、`isolatedTree`の中身は解析しません。**隔離そのものの観測は、ツール自身の試験**（写した木が計画と一致する・`moduleFiles`が空・実buildが通る）**が担います**。

**したがって本記録が言えるのは「隔離ビルドの経路でも鎖が成立する」までで、「この試行が隔離された」ではありません。** 両者を一つの試験で言うには、entryがchildの隔離記録を読む必要があります（未着手）。

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet ./cmd/tei`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.44
- 隔離ビルド：[同 §4.42](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- module context：[同 §4.43](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
