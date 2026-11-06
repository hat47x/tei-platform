# アクセス経路の計数（`reference-access-survey`）

コードが `internal/access` を通さずに外部へ触っている箇所を数えます。数える対象は**構文木**（`go/parser`）で見つけます。コメントや文字列の中の `os.WriteFile` は呼び出しではないので数えません。

## 使い方

```sh
go run ./tools/reference-access-survey                    # 標準出力へ JSON
go run ./tools/reference-access-survey -out survey.json   # ファイルへ
go run ./tools/reference-access-survey -tests             # 試験ファイルも数える
go run ./tools/reference-access-survey internal cmd       # 対象を絞る
```

既定の対象は `internal`・`cmd`・`tools` です。`vendor`・`node_modules`・`testdata` は辿りません。

## 出力

```json
{
  "roots": ["internal", "cmd", "tools"],
  "files": [
    {"path": "tools/…/main.go", "package": "main", "guarded": false,
     "calls": [{"call": "os.WriteFile", "category": "filesystem", "kind": "write", "line": 42}]}
  ],
  "totals": {"filesScanned": 0, "filesWithAccess": 0, "filesGuarded": 0, "directCalls": 0,
             "byCategory": {}, "byCall": {}},
  "byDirectory": [{"directory": "tools/…", "files": 3, "calls": 79, "guarded": 0}]
}
```

| 項目 | 意味 |
| --- | --- |
| `guarded` | そのファイルが `tei/internal/access` を import しているか。**しているだけ**で強制されているとは限らない |
| `category` | `filesystem`・`network`・`environment`。権限の種別に対応する |
| `kind` | 読み・書き・接続・待受けなどの区別 |
| `byDirectory` | 呼び出し数の多いディレクトリ順。件数だけでは「どこを直すか」が分からないため |

## この計数が示すこと・示さないこと

- 示す: **直接呼んでいる箇所の数**と、その分布。同じ手順で何度でも数え直せる。
- 示さない: 実行時に実際に何が起きるか。`access.Guard` を通っていても拒否されない経路はありうるし、`guarded` は「その層を経由する」以上の意味を持ちません。**強制の証拠ではありません**。

## 試験

`main_test.go` は、既知の呼び出しを持つ雛形に対して数えるので、リポジトリの変化に左右されません。

- コメント・文字列の中の名前を数えない。
- 種別（ファイル・ネットワーク・環境変数）を取り違えない。
- 解析できないファイルを「アクセス無し」として扱わない。
- `guarded` とビルド制約を記録する。
- 同じ入力から同じバイト列を出さないと、2回の計数を比べられない。
