# 読まれない TEI ソースを警告する（2026-11-11、第72ラウンド）

- 対象: 第71ラウンドで記録した「静かな失敗」（**起票していなかった**もの）。
- 前段: [第71ラウンド](../TEI_DOGFOODING_ROUND71_PERSISTENCE_LAYER_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **配置の誤りで読まれない `.tei` を `validate` が警告するようにした。13ワークスペースで誤検出ゼロ。**

## 1. 目的

第71ラウンドで「`ok` を返したが何も起きていない」現象を記録し、**修正方針が決まらないため起票しなかった**。本ラウンドで方針を決めて直す。

## 2. 方針の決定

前ラウンドで迷っていたのは「警告を出す」か「もっと広い場所を読む」か。**「警告を出す」を選んだ。**

理由: ローダの読み取り範囲は**設計判断**（`databases/<db>/persistence/` に置くという規約）であり、**規約を広げると別の曖昧さ（同じマッピングが2箇所に置ける）を生む**。一方、**「読まれていない」ことは事実**なので、警告として出すのは安全側である。

## 3. 実装

`internal/workspace` に `CheckWorkspaceUnreadSources` を足し、`Load` の**`markManagedFilesReadByLoader` の後**で呼ぶ。

```go
markManagedFilesReadByLoader(&r)
r.SourceRootFindings = append(r.SourceRootFindings, CheckWorkspaceUnreadSources(r.ManagedFiles)...)
```

**順序が本質**である: `ReadByLoader` はマーキング後にしか分からないので、他の managed-file finding と同時には出せない。

条件: `ReadByLoader == false` かつ `Extension == ".tei"` かつ**名前の検証で既に説明されていない**（二重報告を避ける）。

## 4. 実測（CLI）

```text
誤った場所に置いた persistence マッピング:
  warning WORKSPACE_SOURCE_ROOT managed file "databases/master-data/stray.tei" is a
  TEI source that no loader reads, so its declarations never reach the model;
  check its location (for example a persistence mapping belongs in
  databases/<database>/persistence/)

同じファイルを databases/master-data/persistence/ へ移動:
  warning 0件（ok）
```

**リポジトリ内の13ワークスペースすべてで、新しい警告は 0 件。** 誤検出なし。

## 5. 試験

`internal/workspace/unread_source_test.go`:

| 試験 | 内容 |
| --- | --- |
| `TestUnreadTIESourceIsReported` | 読まれない `.tei` を warning として報告する |
| `TestReadTIESourceIsNotReported` | 読まれた `.tei` は報告しない（**名前ではなく実測に従う**） |
| `TestUnreadSourceDoesNotDuplicateTheNameFinding` | 名前の検証で説明済みのファイルを二重報告しない |
| `TestNonTEIUnreadFilesAreNotReported` | `.txt` などは対象外 |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 呼び出しを削る（修正前の挙動） | FAIL `the unread source was not reported: []` |
| M2 | `.tei` の絞り込みを外す | FAIL `a non-TEI file was reported: ...notes.txt...` |
| M3 | `ReadByLoader` を無視する | FAIL `a file the loader read was reported as unread: ...persistence/thing.tei...` |

**M1 は最初「通ってしまった」。** `git checkout -- internal/workspace/loader.go` で変異を戻したとき、**私の修正そのものも一緒に revert された**ためである。気づいて再適用し、変異を作り直した。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/workspace` | ok |
| `./cmd/tei` | ok（514.7s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. 前ラウンドとの関係

**「起票しない」判断が正しかった回**である。第71ラウンドで方針未決定のまま起票していれば、Issue には「警告を出すか広げるか」という未決の問いが残り、実装時に方針が変われば Issue が嘘になる。**1ラウンド寝かせて方針を決め、起票せずに直接直した。**

## 8. 読み取れないこと（NOT RUN）

- 読まれない `.tei` を**エラー**にすべきか（現状は warning）。
- ローダの読み取り範囲を広げるべきか（設計判断として保留）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**「静かな失敗」に声を与えた回**である。第71ラウンドでは `validate: ok` が「何も起きていない」ことを意味していた。今は同じ状況で警告が出る。

ただし**エラーにはしていない**。ワークスペースが将来のローダ用にファイルを置いておくことは正当なので、warning に留めた。
