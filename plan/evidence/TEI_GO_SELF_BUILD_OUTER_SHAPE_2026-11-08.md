# `tei self-build`の§4外形（workspace・intent・supply・state・output）

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4で固定した外形を、[#3433](https://github.com/hat47x/tei-platform/pull/3433) の薄い入口へ足した記録です。

**`qualified`は常に`false`です。** artifactができたことは、受理でも、閉包資格でも、Runtime Authorityでもありません。

## 何を足したか

`tei self-build`に、§4の外形とSC0-01〜SC0-05の拒否条件を実装しました。既存の`-child`/`-artifact`形はそのまま残し、**§4のflagを1つでも使った呼び出し**だけが外形へ進みます。

| flag | 意味 |
| --- | --- |
| `-workspace` | Build Intentの宣言元。読み込め、`validate`にerrorが無いことを要求 |
| `-intent` | 宣言ID。**記録するだけで解決しない** |
| `-supply` | childへ渡す供給ディレクトリ |
| `-state` | 試行ごとの隔離状態（必須・既定値なし） |
| `-output` | artifact出力（既存なら起動前に拒否） |
| `-child` | exact child executable（必須・PATH解決しない） |
| `-child-arg` | childへ渡す追加引数（JSON配列） |

childへは`-supply <dir> -artifact-out <output> -state <state>`を渡します。これは既存`tools/reference-go-build-with-supply`の文法と同じです。

## 拒否条件

| ID | 条件 | コード | exit |
| --- | --- | --- | --- |
| SC0-09 | `-child`なし | `INVALID_ARGUMENT` | 2 |
| SC0-01 | `-workspace`なし | `SELF_BUILD_PRECONDITION_FAILED` | 2 |
| SC0-05 | Workspaceが読めない、`validate`がerror | 同上 | 2 |
| SC0-03 | `-state`なし | 同上（共有stateを「二回」と数えない） | 2 |
| SC0-04 | `-output`なし／既存 | 同上（no-clobber） | 2 |
| SC0-06 | 子の報告とディスク実物が不一致 | `SELF_BUILD_CHILD_FAILED` | 1 |

**検査の順序**：`-child`の有無 → `-workspace` → Workspace読込みとvalidate → `-state` → `-output`。builderの**digest**と`schemaVersion`の妥当性はchild側のprimitive入口が担うため、外形は「起動前に何を確かめるか」だけを持ちます。

## 実測（WSL2 Ubuntu 22.04 / Go 1.27.3 linux/amd64）

`reference/workspace`を渡し、childに`tools/reference-go-build-with-supply`（`./tools/reference-build-authoring-intent`対象）を起動しました。

| 欄 | 値 |
| --- | --- |
| `ok` | `true` |
| `workspace` | `reference/workspace` |
| `intent` | `intent.sample`（記録のみ） |
| `buildState` / `output` | 指定した隔離stateと出力先 |
| `childRun.invoked.executableDigest` | `sha256:a175a013722730cc925356254fca50a015cd78464f0f2ba0f820dee552cef1e8` |
| `childRun.artifactCrossCheckMatched` | `true` |
| `qualified` | **`false`** |

childのargvには`-supply`・`-artifact-out`・`-state`が含まれ、呼び出し側の追加引数（`-repo`・`-package`）と連結されていることを記録に残しています。

## 実装中に捕まえた問題

1. **既存テストのJSON組み立て**（`"-child-arg", '["-artifact","' + artifact + '"]'`）がWindowsのパス（`\U`）で壊れていた。`encoding/json`で組むhelperに置き換えた。これは**mainに既存の不具合**で、外形の変更とは独立に直した。
2. **外形のflag setが`-child-arg`を受けていなかった**。childへ追加引数を渡せないと実Buildに届かないため、外形でも受けて転送するようにした。
3. **最初の版で`-child`を外形のflag setに含め忘れ**、`-child`を外形扱いしていなかった。現在は外形でも受ける。

## 検証

- `gofmt -l cmd/tei`：差分なし
- `go vet ./cmd/tei`：PASS
- `go test ./cmd/tei -count=1 -run TestSelfBuild`：PASS（primitive 5件＋外形 7件）
- `go test ./cmd/tei -count=1`：PASS（全体）
- 実プロセスでのend-to-end：上表のとおり成功

**NOT RUN**：`-intent`の解決（この外形はIDを記録するだけ）、`-supply`の照合（SB-0bの分類の再実行）、2回の独立試行、`cmd/tei`を対象にしたL3（別途[記録](TEI_SELF_BUILD_CMD_TEI_L3_2026-11-08.md)がmainにある）、Seed Trust、CI。

## 限界

- **`-intent`を解決していません。** SB-C0 §4.2の拒否条件2（Build Intentが解決できない場合）は未実装で、IDの記録だけです。
- **`-supply`を照合していません。** 供給が壊れていても、childが失敗するまで分かりません（SC0-02は未実装）。
- **L2（TEI-defined build）は成立しません。** Build Intent/Graphから実Buildへの結線はまだ無く、外形はchildへ引数を渡すだけです。
- catalogへ載せていないため`tei commands`からは見えません（D-3）。
- 供給のTrust・署名は未検証。G4は`UNQUALIFIED`のままです。
