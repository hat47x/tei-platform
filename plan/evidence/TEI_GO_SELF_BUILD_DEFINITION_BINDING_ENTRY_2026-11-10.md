# §4.36の照合をentry側にも置く

§4.36は「計画が名指すsourceと、builderが読んだファイルの一致」を**builder childが自分で確かめる**形にしました。本記録は、その検査が**builderの実装に依存する**ことを指摘し、entry側にも同じ比較を置いた結果を残します。

## 見つけたこと

§4.36のskewは、**childが自分で照合して自分で止まる**ことで防いでいます。

| 前提 | 誰が守るか |
| --- | --- |
| 読んだdefinitionが計画のものか | **child**（成果物を作る前に停止） |

**childがその検査を実装していなければ、entryは何も気づきません。** §4.37で「childの申告はすべて信じている」と整理した4項目のうち、**「§4.36の停止が働くこと」**がこれに当たります。

## 直したこと

**entryが、childの申告した`definitionBytesDigest`を、自分の計画が記録する値と比較する。**

| 変更 | 内容 |
| --- | --- |
| `childDefinitionBytesDigest`を evidence へ | childの申告 |
| `bindDefinitionToPlan(receipt, reported)` | 計画の`plugin-definition` sourceのdigestと比較 |
| `definitionBinding` を報告へ | `state`（`bound`／`mismatched`／`unperformed`）、`plannedDigest`、`reportedDigest`、`reason` |
| 不一致の扱い | **試行を失敗させる**（`appendSelfBuildFailure`） |

**未申告は`unperformed`**（一致として扱わない。SC0-18と同じ規則）。**申告が無いことは失敗ではない**——報告に理由が残り、条件としては満たされない。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestBindDefinitionToPlan`（一致／不一致／未申告／receipt無し／計画にdigest無し） | **PASS** |
| `TestBindDefinitionToPlanDoesNotTreatSilenceAsAgreement` | **PASS** |
| `TestEntryFailsWhenTheBuilderSaysItReadTheWrongDefinition`（childが`sha256:0000…`と申告 → **失敗**し、記録に両方のdigestが残る） | **PASS** |
| `TestEntryAcceptsABuilderThatNamesThePlannedDefinition`（一致する申告 → 成功し`bound`） | **PASS** |
| `cmd/tei` 全suite | 下記 |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## 何が変わったか

| 項目 | 前 | 後 |
| --- | --- | --- |
| childが照合を実装していない場合 | **気づかない** | **entryが不一致を検出して試行を失敗させる** |
| 記録 | childの`definitionMatchesPlan`（自己申告） | `definitionBinding`（**entryが比較した結果**） |

**§4.37の4項目のうち「§4.36の停止が働くこと」が、信じる対象から外れた。** 残る信じている項目は、宣言を実際に読んだこと・計画から成果物を作ったこと・申告どおりのstateを使ったこと。

## 言えないこと

- **childが本当にそのファイルを読んだこと**（digestはchildの申告。読んだ行為は見えない）
- ディレクトリのclosureの照合（単一ファイルのみ）
- `go-package`など他のsource種別の照合
- 別ホスト・別Go版での成立

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet ./cmd/tei`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.36
- 信頼の範囲の整理：[同 §4.37](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- builder側の照合：[`TEI_GO_SELF_BUILD_SOURCE_BINDING_2026-11-10.md`](TEI_GO_SELF_BUILD_SOURCE_BINDING_2026-11-10.md)
