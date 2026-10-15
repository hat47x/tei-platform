# reference-authoring-auxiliary（#1808）

設計の補助情報（用語、改訂履歴、注記、しおり、区分の類）を、意味の正本・導出・利用者ごとの局所の状態・表示情報・運用の記録のどれに置くかを分類し、その結果を読取り専用で出力する参照です。SourceやWorkspaceを書き換えず、定義を受理もしません。

~~~text
go run ./tools/reference-authoring-auxiliary/cmd/tei-authoring-auxiliary -input request.json
~~~

`-input`は分類の依頼を書いたJSONです。`-`で標準入力から読みます（既定）。判断のための中間ファイルであり、意味の正本ではありません。

## 入力

- `candidates`: 補助情報の候補。`affectsExecution`、`recomputableFromCanonical`、`perViewer`、`presentationOnly`を持つ
- `annotations`: 注記。`subject`（意味の参照）、`kind`（`issue` / `question` / `note`）、`state`（`open` / `done`）、`author`、`text`
- `revisionEvents`: 版管理が記録した改訂
- `revisionReasons`: 人が書いた改訂の理由
- `terms`: 用語。`symbol`（名称の索引）、`definition`、`accepted`、`basis`
- `display` / `operational` / `local`: 意味を含まない情報の例

## 出力

- 各候補の置き場所（`meaning-canonical` / `derived` / `user-local` / `display-info` / `operational-record`）と、その理由
- 構造化された注記の一覧
- 版管理から導出した改訂の事実と、人が書いた理由
- 名称の索引と、受理済みの意味の定義
- 意味の版（`meaningVersion`）
- 診断

## 判定の順序

候補は次の問いに上から順に答え、最初に当てはまる置き場所へ置きます。

1. 変えると実行・検証・認可・Targetへの変換が変わるか（`meaning-canonical`）
2. 意味の正本や版管理から再計算できるか（`derived`）
3. 閲覧する人ごとの好みか（`user-local`）
4. 見せ方だけに関わるか（`display-info`）
5. 上のいずれでもない作業上の記録か（`operational-record`）

2を1より前に置きません。再計算できる情報でも、変えると実行が変わるなら意味の一部です。

## 守っている境界

- 注記は、記述欄の中の印（`@NOTE@`）を走査して一覧しません。文字列に残った印は`embedded-marker-not-structured`として報告し、構造化した宣言への移行を促します。意味の参照を持たない注記は`annotation-without-subject`です。
- 改訂の事実は版管理から導出し、保存しません。人が同じ内容を書き直した場合は`duplicated-derived-fact`、版管理に無い版へ理由を付けた場合は`reason-without-revision`として報告します。
- 用語の名称の索引は導出です。意味の定義は、受理済みで根拠を持つときだけ意味の候補になります。未受理は`definition-not-accepted`、受理済みでも根拠が無い場合は`definition-without-basis`として`unresolved`のままにし、推測で補完しません。
- 意味の版（`meaningVersion`）は、`meaning-canonical`の候補と受理済みの定義だけから作ります。表示情報・運用の記録・利用者ごとの局所の状態を変えても動きません。
- 出力は常に読取り専用です。定義の受理（#267）や、変更案の適用は行いません。

## 未実装

- 変更案の受理（#267）と再投影
- 意味の正本との実際の結線（`candidates`と`terms`は依頼側が用意する）
- 表示情報の版管理と、表示先ごとの差分
- 注記の共有と権限（受理の真正性確認）
