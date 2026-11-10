# 再現の記録が「同じbuilder」を確かめていなかった

§4.21の再現の観測は「**同じchild実行物**を、別stateで二回走らせて同じバイトが出た」という主張です。本記録は、その主張の三つの入力のうち**一つが記録されているだけで比較されていなかった**ことを示し、直した結果を残します。

## 見つけたこと

`artifactReproducibility`の`secondChildDigest`は、コメントで「the executable digest of the second run, so a reader can see that both runs used the same binary」と説明されていました。

**しかし実際に入っていたのは、entry自身（`tei`）のdigestでした。**

```go
record.SecondChildDigest = evidence.InvokerDigest   // ← entry の digest
```

`driveSelfBuild`の`InvokerDigest`は`os.Executable()`のdigestです。childのdigestは`evidence.Invoked.ExecutableDigest`に入っています。

| 欄 | 実際の中身 |
| --- | --- |
| `invokerDigest` | 走っている`tei`自身 |
| `invoked.executableDigest` | **child**の実行物 |
| `artifactReproducibility.secondChildDigest` | **entry**（名前と説明に反する） |

**さらに、二回の実行でbuilderが同じだったことを比較する検査がありませんでした。** `childPath`は同じ値を渡しているので「たぶん同じ」ですが、**記録はそれを示していませんでした**。二回の間に実行物が差し替われば、同じバイトが出ても「同じbuilderの決定性」の観測にはなりません。

## 直したこと

| 変更 | 内容 |
| --- | --- |
| `firstChildDigest`を追加 | 一回目に**実際に起動した**childのdigest |
| `secondChildDigest`を修正 | childのdigest（`evidence.Invoked.ExecutableDigest`）へ |
| `reproducibilityBuilderVerdict`を追加 | 二つのdigestを比較し、**違えば不成立**、**未記録も不成立** |
| `mismatchReason`を追加 | 何が食い違ったか（builderか、バイトか）を述べる |

**未記録を不成立として扱う**のは、SC0-18（欄が無いことを一致と読ませない）と同じ規則です。

## 実測

| 試験 | 結果 |
| --- | --- |
| `TestReproducibilityBuilderVerdict`（4ケース：同一／別builder／1回目未記録／2回目未記録） | **PASS** |
| `TestReproducibilityRecordsTheBuildersDigestAndNotTheEntrys`（実childで`reproduced`、`firstChildDigest`と`secondChildDigest`が**childのdigestと一致**し、**entryのdigestではない**） | **PASS**（7.86秒） |

環境：WSL2 Ubuntu 22.04、`go1.27.3 linux/amd64`。

## この修正が変える主張

**変わらない**：`reproduced`の意味（同じplanを二回buildして同じバイト）。

**強くなる**：「同じchild実行物」が**記録され、比較される**ようになった。これまでは記録の欄名が示すだけで、検査が無かった。

**新しく不成立になる場合**：二回の間にchild実行物が差し替わった試行。これまでは`reproduced`になり得ました。

## 言えないこと

- **childの実装が正しいこと**（従来どおり。digestを比べるだけで中身は見ない）
- 別ホスト・別Go版での成立（§4.26の範囲）
- `tei`自身の同一性（`invokerDigest`は従来どおり記録されるが、本件の対象外）

## 検証

- `go test ./cmd/tei -run 'SelfBuild|AssessL2B|ApplyEvidenceDefaults|Reproducibility'`：**PASS**（28.7秒）
- `gofmt`差分なし／`go vet ./cmd/tei`：PASS

**NOT RUN**：`darwin`・`arm64`、CI。

## 参照

- 設計：[`SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md`](../design/SELF_BUILD_REENTRY_CONTRACT_SB0C_2026-11.md) §4.21
- 再現の実測：[`TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md`](TEI_GO_SELF_BUILD_ARTIFACT_REPRODUCIBILITY_2026-11-09.md)
