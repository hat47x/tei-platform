# 本物の対象で、鎖を一本に通した（§4.50）

§4.49は「`./cmd/tei` の隔離ビルドは通ったが、**五条件の Claim は再測していない**」で終わっていました。本記録は、**その接続を閉じた**結果です。

## 通した鎖

| 段 | 内容 |
| --- | --- |
| 宣言 | `./cmd/tei` と **module 内の閉包 73 件**、`go.mod`・`go.sum`（合計 **75 source**） |
| catalog | 能力宣言から作成（provider の `artifact` は実バイトから計算） |
| 計画 | reference resolver で解決 |
| build | `tei self-build`（`-build-catalog`・`-supply`、隔離は既定） |

## 実測

| 項目 | 値 |
| --- | --- |
| `planFingerprint` | `42e8a622d3590ec15b440019ca8b6ec7dd81da0afecf5f0094fcc331c2295b10` |
| `ok` | **true** |
| `l2b.claimed` | **true**（`unmet` 空） |
| `planReceipt` / `planCrossCheck` | resolved / matched |
| `planArtifactBinding` | bound |
| `artifactReproducibility` | **reproduced** |
| `definitionBinding` | **bound** |
| `childIsolation` | **isolated**（members 75・`moduleFiles` なし） |
| artifact | `sha256:6b9290ed442d798f7227ac41847ef2da5ccc15f776b2c002ff29a3d4507c8ca7` |

**`./cmd/tei`（73 の package を持つ実在の対象）で、五条件がすべて成立しました。** しかもビルドは**計画の source だけを写した木**で行われています。

## 試験に載せた

`TestSelfBuildFullChainOnTheRealSubject`（`cmd/tei`）。

| 項目 | 値 |
| --- | --- |
| 実行時間 | **254秒** |
| `-short` | skip |

**この試験は重い**（内部ツリー全体を二回ビルドする）。**それでも載せた**のは、これが「セルフビルドが実現した」と言うための唯一の観測だからです。

## 途中で踏んだこと（重要）

**`t.TempDir` の後片付けが失敗しました。** Go は module cache を**読み取り専用**で書くので、`RemoveAll` が `permission denied` で止まり、**試験は通っているのに失敗として報告**されました。手動のディレクトリと、権限を直す後片付けに変えました。

**「合格したのに不合格と表示される」**のは、観測として最悪の形です。

## 言っていないこと

- **外部 module の完全性**（供給の Trust は範囲外）
- 別ホスト・別 Go 版（`darwin`・`arm64` は未観測）
- 受理・Runtime Authority・G4 適格性（**`qualified` は `false` のまま**）
- CI の標準時間に収める工夫（**254秒は重い**）

## 検証

- 上の実測（`ok=true`・`l2b.claimed=true`）
- `go test ./cmd/tei -run TestSelfBuildFullChainOnTheRealSubject -count=1`：**PASS**（254秒）

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.50
- 隔離ビルドの実測：[同 §4.49](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 計画の著者：[同 §4.46](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
