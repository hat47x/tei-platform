# 計画を読んでビルドするbuilderができた（§4.39 段1）

§4.39は「計画を解釈してGoをコンパイルするbuilderが無い」ことを受け、**そのbuilderが受け取る計画の形と実装の順序**を決めました。本記録は、**段1を実装して実際にビルドした**結果を残します。

## 作ったもの

`tools/reference-go-build-from-plan`（[README](../../tools/reference-go-build-from-plan/README.md)）。

| 段 | 内容 |
| --- | --- |
| 1 | `-plan`を**厳格に**読む（未知キー・case違い・後続値は拒否） |
| 2 | `-intent`があれば読み、byte digestを申告 |
| 3 | 計画の`Sources`を**`DirSourceResolver`で同じrootから解釈し直し**、計画のdigestと比較。**違えばビルドしない** |
| 4 | `gobuilder.BuildPackage`で`go build`（隔離state・`-trimpath`・`-buildvcs=false`・`-mod=readonly`） |
| 5 | 成果物のdigestを計算し、**自分でlockを作って検証** |
| 6 | 全部を1つのJSONで出す |

**`DirSourceResolver`をそのまま使ったのは意図的です。** 枠組みを二か所に書くと必ず食い違うので（§4.36で実際に食い違った）、entryと同じ実装を呼びます。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestRunBuildsFromThePlanAndReportsWhatItDid`（`./tools/reference-build-authoring-intent`を実際にビルドし、`planFingerprint`・`artifactDigest`・`stateDir`・lockの整合を確かめる） | **PASS** |
| `TestRunRefusesASourceThePlanDoesNotDescribe`（計画のdigestを書き換えると**ビルドしない**） | **PASS** |
| `TestRunRefusesAnUnknownPlanKey`（未知キーは拒否） | **PASS** |
| `TestRunRequiresItsInputs`（`-plan`・`-artifact-out`・`-state`・位置引数） | **PASS** |
| `go test ./tools/reference-go-build-from-plan -count=1` | **PASS**（14.9秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## 五条件との対応

| 条件 | このtoolが出すもの |
| --- | --- |
| 2（宣言byte） | `intentBytesDigest`（`-intent`を渡したとき） |
| 3（計画fingerprint） | `planFingerprint`（計画から自分で計算） |
| 4（plan→artifact） | `artifactDigest`と、**自分で作って検証したlock** |
| 5（再現） | `stateDir`と、`-artifact-out`・`-state`の尊重 |
| 1（entryが計画を導く） | **呼び手の仕事**（`-build-catalog`） |

## まだ足りないこと

- **段2（catalogのproviderとして宣言し、`tei self-build -build-catalog`で鎖を通す）は未着手。** したがって**`l2b.claimed`はまだ立たない**
- **TOCTOUは閉じていない。** 照合とcompileの間にファイルが変わり得る（§4.39の案Bで閉じる）
- **計画に載っていないsourceを読んでいないこと**は示していない（`go list -deps`との突き合わせは案のまま）
- 受理・Runtime Authority・供給のTrust。いずれも行わない

## 検証

- 上の表のとおり（実buildを含む）
- `gofmt`差分なし／`go vet`：PASS
- `go test ./tools/repository-plan-check -count=1`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.39
- 能力の一覧：[同 §4.38](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 計画の形の実測：[`TEI_GO_SELF_BUILD_GO_COMMAND_PLAN_2026-11-10.md`](TEI_GO_SELF_BUILD_GO_COMMAND_PLAN_2026-11-10.md)
