# CLI 契約の整合性スイープ（第2回）と #1177 の測定手順

- 記録日: 2026-11-11
- 対象: 読み取りのみ（コード変更なし。`cmd/tei`・`internal/workspace` は無変更）
- 関連: #1177（大規模Workspaceの効率）、#3943・#4001・#4094（同種の「広告と実挙動のずれ」）

## 目的

「広告（`tei commands`）と実挙動が食い違う」型の欠陥を、これまで個別に直してきました。今回は**同じ観点を一括で掃引し、食い違いが残っていないかを確かめる**のが目的です。見つからなかった場合も、その事実を残せば次の調査を繰り返さずに済みます。

## 1. `schemaVersion` の一致（14/14）

`tei commands` が `schemaVersion` を広告する14コマンドについて、到達可能な最小の正常呼び出しを実行し、出力先頭の `schemaVersion` と比較しました。

```text
schemaVersion agreed: 14 / mismatches: 0 / advertised none: 0
```

補足: `version` は `-json` を付けたときだけ封筒を出すため、`-json` なしの比較は「封筒が無い」と誤って見えます。実際には欠陥ではありません。`world` は `schemaVersion` を広告せず、出力も IR の JSON そのものなので一致しています。

## 2. `argumentDefaults` と `-h` の既定値（3/3）

```text
view    -format     advertised text     help text     OK
trace   -direction  advertised both     help both     OK
trace   -mode       advertised summary  help summary  OK
```

## 3. `argumentValues` の値が受理されるか

`view -format` の `html`・`print` が `-application` 単独では拒否される件は #4095 で修正済みです（広告に形式別の制限を明記）。修正後、catalogue の値集合は「そのコマンドのどこかで受理される値の和集合」として正しくなっています。

## 4. 引数エラーと読み込みエラーの符号

引数なし／存在しない Workspace で全コマンドを実行し、`exit` と `error.code` を表にしました。**封筒を出さない2コマンドは異常ではなく、文書化された text モードです。**

| コマンド | 引数なし | 存在しない Workspace | 判断 |
| --- | --- | --- | --- |
| `validate`（`-json` なし） | exit 2・stderr | exit 2・stderr | 広告が `Output: "text; json when -workspace -json"`。**text モードでは封筒を出さない**のが仕様 |
| `world` | exit 2・stderr | exit 2・stderr | `world` は IR の JSON を出すコマンドで、封筒を広告していない |
| その他 17コマンド | exit 2・`INVALID_ARGUMENT` | exit 2・`WORKSPACE_LOAD_FAILED` | 一致 |

`version -workspace ...` は `version accepts only -json` で拒否されます。`version` は Workspace を取らないコマンドなので正しい挙動です。

## 5. `workspace.Load` の呼び出し回数（各1回）

#1177 の「二重解析」を確かめるため、`workspace.Load` に一時的な計測を入れて（**コミットせず戻した**）数えました。

```text
trace              1
inspect            1
knowledge-context  1
validate           1
```

**1コマンド1回**です。「二重解析」は `Load` が2回呼ばれる形ではなく、`Load`（構文解析）と `BuildSourceIndex`（別の解析）が**同じファイル群を2つの段で読む**形を指しています。

## 6. #1177 の測定手順と基準値

計装せずに段を分解する方法を確立しました。

| 呼び出し | 含む段 |
| --- | --- |
| `world -workspace` | Load |
| `source-index -workspace` | Load + BuildSourceIndex |
| `inspect ...` | Load + BuildSourceIndex + 投影 |
| `trace ... -mode full` | 同上（+行の解決） |

`reference/workspace` での実測（5回の中央値、Windows 11、go1.27.0）:

```text
world (Load only)             160.4 ms
validate -json                162.0 ms
source-index                   137.9 ms
inspect                        352.4 ms
trace full                     216.8 ms
model catalog                  177.4 ms
```

**この規模では差が誤差に埋もれます。** プロセス起動が約120msを占め、`Load` と `BuildSourceIndex` の差は数十msです。したがって #1177 の判断には、`reference/workspace` より1〜2桁大きい Workspace が必要で、その生成手段が未整備です。今回はその欠落を特定するところまでとします。

## 主張しないこと

- 性能改善は行っていません。基準値と測定手順の記録だけです。
- 大規模 Workspace を生成していません。したがって「スケールしても完全性が保たれる」ことも「O(n) で伸びる」ことも主張しません。
- 14コマンドすべての正常系を確認したわけではありません。`schemaVersion` の一致だけを見ています。
- 実 Host・認可・外部送出は NOT RUN。
