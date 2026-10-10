# First Value UX Audit（初回体験の摩擦計測）

[#1168](https://github.com/hat47x/tei-platform/issues/1168) の最小 read-only メトリクス導出器です。TEIによる初回の業務モデル変更や、将来のデータを扱う機能までの手間を、**観測したイベント記録**から再現可能な件数に変換します。計測器はモデルを変更しません。特定のRuntimeやTargetへ接続しません。

## 操作

リポジトリのルートから、Goだけで実行できます。

```sh
go run ./tools/first-value-ux-audit -input tools/first-value-ux-audit/testdata/synthetic-session.json
GO111MODULE=off go test -race -count=1 ./tools/first-value-ux-audit
GO111MODULE=off go vet ./tools/first-value-ux-audit
```

`testdata/synthetic-session.json` は**架空のイベント・架空の経過時間・架空のSHA**で構成した、Schemaと集計ロジックの見本です。実ユーザーの測定値や動作実証ではありません。

## 記録Schema（v1）

1回の作業セッションを1つのJSONにします。トップレベル:

- `schema`: `tei.first-value-ux-session/v1` 固定。
- `sessionId`, `taskId`: セッションと課題を識別する非空文字列。ユーザー名や秘密情報を入れない。
- `phase`: `E0`（最初のモデル変更）または`E1`（データを扱う業務機能）。E1はE0の合格を意味しません。
- `actor`: `new-developer`, `new-agent`, `experienced-developer` のいずれか。実測ではセッション間の情報引継ぎ・事前知識も別途記録する。
- `provenance`: `synthetic`（架空入力）または`observed`（人間が観測したと申告）。`observed` も第三者検証済みの意味ではありません。
- `repositorySHA`: 作業開始時の40桁のGit commit SHA。Fixtureでは40桁のゼロを使用。
- `startedAt` と任意の `finishedAt`: RFC 3339形式。終了時刻なしの場合、所要時間は**空欄**とする。
- `events`: 時刻昇順のイベントを1件以上。時刻は開始前・終了後に出てはいけません。

イベント種類:

| kind | 意味 | 入力条件 |
| --- | --- | --- |
| `command` | CLI等の実行1回 | `exitCode` 必須。非0を失敗コマンドとして数える |
| `source-edit` | 編集操作1回 | 追加パラメータ不要 |
| `manual-fields` | 手入力項目数を計上 | 正の整数 `count` |
| `recovery` | エラーからの復帰操作1回 | 追加パラメータ不要 |
| `independent-read` | 独立した再読込が行われた**という申告** | E1のみ、`evidenceRef` 必須 |

すべてのイベントに `at` (RFC 3339) が必要です。`evidenceRef` は任意の根拠参照（識別子またはファイル相対パス）であり、計測器はファイルを開いたり実行・検証したりしません。資格認定に使用する場合は、参照先の内容、Source revision、実行Actor、Commandと独立した観測を**別途**確認してください。

## 出力と証拠の限界

`tei.first-value-ux-metrics/v1` をstdoutのJSONとして出力します。所要秒数（終了時刻がある場合）、コマンド数と非0終了数、編集数、手入力項目数、復帰操作数、独立再読込の**申告件数**、重複排除したEvidence参照を含みます。JSONへない操作や実際の滞在時間を推定しません。順序の逆転、時間窓外、未知のイベント、余分なJSON項目・値、1MiBを超える入力は拒否します。

**`qualification` は常に `unverified`** です。`finishedAt` があっても成功とはみなしません。E1で`independent-read`が記録されなければ`independent-read-not-recorded`とします。記録されていても、その独立性やDB/Providerへの実到達を自動検証しません。架空データは常に`synthetic-session-not-user-evidence`を返します。計測器のテストPASSを製品のE1/業務利用可能性の証拠にしません。

初回の測定対象は [#1161](https://github.com/hat47x/tei-platform/issues/1161) の最小Workspaceです。正解編集手順を渡さない初見セッションと熟練者セッションを分けて保存します。後続の [#1127](https://github.com/hat47x/tei-platform/issues/1127) と [#1163](https://github.com/hat47x/tei-platform/issues/1163) によるactual data-backedな縦断Evidenceの確認は別工程です。現時点で人間による2件の実測は取得していません。

作業ログには環境変数・トークン・個人情報を含めないでください。外部共有の前にログを検査して機密情報を除外してください。
