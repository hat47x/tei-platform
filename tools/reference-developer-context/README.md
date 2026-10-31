# 開発文脈ナビゲーター（Issue #2196 / #2021）

TEIのGoソースを1ファイル指定し、**実在する関連テストと設計資料の調査候補**を返す、読み取り専用の参照PoCです。正式なStudio機能ではありません。新しい意味正本や実行権限を作らず、既存のCognition Port / Shared Acceptanceに従う「候補探索の独立した実験」として扱います。

## 使い方

リポジトリのルートから実行します。

```sh
go run ./tools/reference-developer-context -root . -target internal/crossview/qualification.go
```

結果はJSONで、対象のSHA-256、候補の相対パス・SHA-256・探索根拠・元の表示順を含みます。元ファイルやテストを実行・変更しません。

ローカルのSystem One互換判断モデルを利用する場合だけ、別プロセスで`llama-server`等を127.0.0.1に起動したうえで、次のように実行します。

```sh
go run ./tools/reference-developer-context \
  -root . -target internal/crossview/qualification.go \
  -mode systemone \
  -endpoint http://127.0.0.1:8080/v1/systemone \
  -model YOUR_LOADED_MODEL_ID
```

`YOUR_LOADED_MODEL_ID`は実際に読み込んだdecision modelのIDに置き換えます。モデルは同一候補集合に対する`choice`（`relevant` / `uncertain` / `not_relevant`）だけを返します。**結果は表示順を変えるだけ**で、候補の削除・編集・保存・検証実行を行いません。判断確率は未校正であり、正解率や重要性の保証ではありません。

## 実装している最小処理

1. 指定ファイルがリポジトリ内の正規化済みGoソースであることを検査する。対象の存在と内容を確認し、SHA-256を記録する。
2. 同じGoパッケージのテストファイルから、ファイル名または本文が対象のstemと一致する候補を拾う。
3. `docs/`と`plan/adr/`のMarkdownから、対象のファイルパスまたは親ディレクトリパスへの明示的な参照を持つ資料を拾う。
4. 最大12候補を決定的順序で提示する。各候補は実在するfileとSHA-256を根拠とする。
5. `-mode systemone`指定時のみ、HTTP loopbackの型付き判断モデルへ候補集合を送り、妥当な応答なら順位を付け直す。不通・不正・拒否なら`model-failed`を明示し、baselineの全候補と順序を維持する。

ファイルは1 MiB上限、モデル応答は256 KiB上限、タイムアウトは5秒です。対象と候補のパストラバーサル・symlinkによるroot外への脱出を拒否し、Markdown探索件数にも上限を設けます。モデル接続先は**リテラルのloopback IPと`/v1/systemone`パスのみ**を許容し、リダイレクトは追跡しません。

`-mode baseline`は完全にローカルの決定的処理です。`-mode systemone`は指定ファイルパス、指紋、候補一覧をローカル推論サーバーへ送ります。このPoCはOS sandbox・通信隔離・資格情報除去を証明せず、信頼する端末内サーバーだけを利用してください。

## 試験

```sh
go test ./tools/reference-developer-context/... -count=1
```

fixtureでは、直接一致するテスト、本文一致テスト、参照する設計文書、root外パス・symlink・過大入力の拒否、mock System Oneの型付き応答、モデル失敗時のbaseline保持を検査します。

## Evidenceの境界と次のWork作業

このPoCが示せるのは「調査候補を根拠付きで提示できること」までです。**テストPASS、production caller、Data・Function・Businessの意味対応、Owner受理、正式Host/Provider/DBの接続は判定しません。** 誤った高確信の順位、関係が弱いものを消したくなる危険、候補探索の漏れは別途評価します。

Work側では、最新mainとPRのexact headを固定し、実際の`go test`、`go run`、Windowsでのループバックのdecision model疎通を確認してください。対象例は`internal/crossview/qualification.go`です。

比較時は、関連候補recall@k、不要な候補数、実際に不足証拠へ到達するまでの操作数、未知ケースの保留、p50/p95レイテンシ、RSS、ファイル改訂時の誤参照、ローカルモデル不在時のフォールバックを測定します。決定的baselineに独立した価値増分がなければ、認知順位付けを既定機能に昇格しません。

関連: [#2196](https://github.com/hat47x/tei-platform/issues/2196)、[#2021](https://github.com/hat47x/tei-platform/issues/2021)、[ADR-0038](../../plan/adr/0038-cognition-as-proposal-port.md)、[参照Cognition Port](../reference-cognition-port/README.md)。
