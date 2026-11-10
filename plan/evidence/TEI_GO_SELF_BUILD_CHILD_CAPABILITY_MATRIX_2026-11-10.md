# どのchildが、どのL2-b条件を満たせるか

L2-bの五条件（§4.23）は**childの能力に依存します**。本記録は、参照実装のどれがどの条件を満たせるかを測って一覧にしたものです。

## なぜ必要か

**§4.21でstate分離の検査を足したとき、実buildの観測が黙って`unperformed`になりました**（[記録](TEI_GO_SELF_BUILD_GO_BUILDER_STATE_2026-11-10.md)）。childが申告しない欄を条件にすると、**そのchildを使った観測は止まります**。

**止まること自体は正しい。** 悪いのは、**止まったことに気づかない**ことです。条件を足す前に、どのchildが何を申告するかを見る必要があります。

## 一覧

| child | 条件1（in-process） | 条件2（宣言byte） | 条件3（計画fingerprint） | 条件4（plan→artifact） | 条件5（再現） |
| --- | --- | --- | --- | --- | --- |
| `reference-plugin-package-build-child` | 呼び手が`-build-catalog`を渡せば満たす | **満たす** | **満たす** | **満たす** | **満たす** |
| `reference-go-build-with-supply` | 同上 | **満たさない** | **満たさない** | **満たさない** | **満たす**（#4043） |
| `reference-go-build-reproducibility` | — | — | — | — | **満たす**（`stateDir`を持つ） |

## 測ったこと

| 事実 | 根拠 |
| --- | --- |
| `reference-go-build-with-supply`の報告に`planFingerprint`・`intentBytesDigest`・`stateDir`が無かった | 実行して報告のキーを列挙（[記録](TEI_GO_SELF_BUILD_GO_BUILDER_STATE_2026-11-10.md)） |
| `stateDir`を足した後、実buildで`reproduced`が戻った | `tei self-build -rebuild-check`の実測（同上） |
| `reference-go-build-reproducibility`は`stateDir`を持つ | ソースの`json:"stateDir"`を確認 |
| `reference-build-execution-binding`・`reference-go-build-stage-driver`はchildではない | 前者は照合tool、後者はchildを起動するdriver |

## この表が意味すること

**`l2b.claimed`が立つのは、五条件すべてを満たすchildだけです。** 現在それは`reference-plugin-package-build-child`だけで、**これはplanを解釈する参照実装**であって、**TEI自身をビルドする道具ではありません**。

つまり:

> **TEI自身（Goのバイナリ）を、五条件を満たして自己構築することは、今はできない。**

理由は**flagの不足ではなく**、**計画を解釈してGoをコンパイルするbuilderが存在しない**ことです（§4.10.2のP-2を採らない判断と同じ理由）。

## 最短経路に要るもの

| 必要なもの | 何のためか |
| --- | --- |
| 計画を読む（`-plan`） | 条件3のfingerprintを自分の入力から出す |
| 宣言を読む（`-intent`） | 条件2のbyte digestを自分の読みから出す |
| 計画のsource closureから`go build`する | 条件4・5を自分の成果物で満たす（§4.36の照合もここに入る） |

**`reference-go-build-with-supply`に欄を足すのでは足りません。** それは転記であり、§4.10.2がP-2として退けた案です。

## 検査を足すときに確かめること

1. **既存の観測が止まらないか**（新しい条件が要求する申告を、そのchildが出すか）
2. **止まるのが正しい停止か**（正しい停止でも、気づかなければ観測を失う）
3. **そのchildを使う試験があるか**（無ければCIが知らせない）

## 検証

- 上の表のうち、`reference-go-build-with-supply`の欄は実行して確認した
- `reference-go-build-reproducibility`はソースで確認した（実行していない）
- `plan/evidence/README.md`の索引に載せた
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：CI（`tools/reference-go-build-with-supply`の新試験はWindowsで単独実行しPASS）。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.38
- 述語：[同 §4.23](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 信頼の範囲：[同 §4.37](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
