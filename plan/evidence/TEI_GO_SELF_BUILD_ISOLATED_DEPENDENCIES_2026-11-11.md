# 依存のある対象でも、計画だけからビルドできた（§4.49）

§4.47の記録は「依存のある対象の隔離ビルドは未対応」で終わっていました。本記録は、**それを実測した**結果です。

## 何が要だったか

§4.46により、計画は**ビルドが読む source だけ**を名指さなければなりません。`./cmd/tei` は module 内だけで **73 の package** を読みます。手で並べるのは現実的でないので、**計画を書くツール**を足しました（`tools/reference-build-plan-authoring`）。

| 段 | 内容 |
| --- | --- |
| 1 | `go list -deps` で module 内の package を集める |
| 2 | 各 source を `DirSourceResolver` で解決し、**digest は実ファイルから計算**（宣言はできない） |
| 3 | `go-module` として `go.mod`・`go.sum` も名指す |
| 4 | `BuildPlan` として書き出す |

## 実測

`./cmd/tei` を対象に、計画を書いてから隔離ビルドを実行しました。

| 項目 | 値 |
| --- | --- |
| 計画の source 数 | **75**（`go-package` 73・`go-module` 2） |
| `buildSource` | **`plan-only`** |
| `buildSucceeded` | **true** |
| `isolatedTree.moduleFiles` | **なし**（作業ツリーからの持ち込みゼロ） |
| 被覆検査 | **covered=73・uncovered=0**、external=36（供給から来る外部 module） |
| lock | **validated** |
| artifact | `sha256:3e9a8d07a714300d8103fc4381b98a525ea19ec333842524f21325e413f266ee` |

**依存のある対象でも、計画の source だけでビルドできました。** module context も計画から来ており、作業ツリーからは何も取っていません。

## 何が言えるようになったか

| 以前 | 今 |
| --- | --- |
| 「隔離ビルドは依存の無い対象だけ」 | **依存のある対象（`./cmd/tei` 全体）でも隔離できる** |
| 「計画を書くには 73 件を手で並べる」 | **ツールで作れる**（digest は実ファイルから） |

## 言っていないこと

- **`./cmd/tei` を対象にした五条件の Claim**。隔離ビルドが通ることと、Claim が立つことは別です（§4.39段2の観測は隔離前のもの）
- 外部 module の完全性（供給の Trust）
- 計画の自動生成が「読むもの全部」であることの証明。**ビルド時の被覆検査が確かめます**（§4.41）
- `replace`・`vendor/`（§4.48で拒否）

## 検証

- 上の実測（計画 75件 → 隔離ビルド成功）
- `go test ./tools/reference-build-plan-authoring -count=1`：**PASS**（6.9秒）
- ツールの試験：書いた計画が validate し、自己完結の対象では 1+2 sources、閉包のある対象では5件以上

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.49
- 計画が読むsourceだけを名指す：[同 §4.46](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 被覆検査：[同 §4.41](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 隔離の既定化：[同 §4.47](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
