# entryが、builderの隔離記録を読む（§4.45）

§4.44は「隔離ビルドの経路でも鎖が成立する」までしか言えませんでした。**entryがchildの`isolatedTree`を解析していなかった**からです。本記録は、それを読んで記録するようにしたことを残します。

## 何をするか

| 段 | 内容 |
| --- | --- |
| 1 | childの`isolatedTree`（`root`・`members`・`moduleFiles`・`boundary`）を解析 |
| 2 | **`moduleFiles`が空なら`isolated`**、あれば`isolated-with-external-context`（理由に何を持ち込んだかを並べる） |
| 3 | 報告の`childIsolation`として出す。**五条件には加えない** |

## 五条件に加えない理由

**条件を増やすことは契約の変更**で、§4.23の述語を変えます。**記録を足すだけなら述語はそのままで、読む人が「この試行は隔離されていた」を確認できます。** 加えるべきかは`-isolate`を既定にする判断と同じ場所で決めます（§4.43）。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestReadChildIsolation`（未申告／`isolated`／`isolated-with-external-context`と理由） | **PASS** |
| `TestSelfBuildEstablishesL2BFromARealGoBuild`（`childIsolation.state=isolated`・`moduleFiles`空・`memberCount=4`） | **PASS**（12.8秒） |

**`memberCount=4`の内訳**：package・計画が持つ宣言・`go.mod`・`go.sum`。**宣言は§4.40で定義sourceではなくなりましたが、計画のsourceとしては残っています**（ビルドは読まないが、写す対象ではある）。

## 言っていないこと

- **entryが隔離木を見たこと**。見えるのはchildの申告だけです（§4.37の「信じている」に属する）
- 五条件への追加（**契約変更であり、別途決める**）
- `-isolate`の既定化

## 検証

- 上の表のとおり
- `gofmt`差分なし／`go vet ./cmd/tei`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.45
- 隔離ビルドの回帰：[同 §4.44](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
- 信頼の範囲：[同 §4.37](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md)
