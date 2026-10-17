# reference-function-definition-document（#1803）

受理済みの意味から、機能定義書（Markdown）を導出する参照です。設計者やレビュー者が読む成果物で、Workspaceは変更しません。表定義書（`reference-table-definition-document`）、利用の対応表（`reference-access-matrix`）と対をなします。

~~~text
go run ./tools/reference-function-definition-document/cmd/tei-function-definition-document \
  -workspace reference/workspace -out /tmp/function-definitions.md
~~~

出力先のファイルが既にある場合は失敗します。標準出力には、構造の版、文書のdigest、未解決の事項を出します。

## 文書の内容

Applicationごとに、利用する意味、評価する検証（宣言順）、データベース要求、Pattern（種類、対象Data、関連、問い合わせ、Capability）を載せます。Patternが使うCapabilityについては、入力、出力、業務上の結末、宣言された効果を載せます。

## 守っている境界

- 記載は宣言された内容だけです。画面の見た目、実行時の副作用の実績、権限の付与は含みません。文書の冒頭にも書きます。
- 文書の編集は意味の正本へ戻しません。導出元の構造の版を記録し、同じ版からは同じ内容が出ます。
- 存在しないData、Capability、検証への参照は、補完せず「未解決」として載せます。
- Applicationの順序が入力でどうであっても、本文は同じです。

## 未実装

- Patternが`Relation`や`Query`だけを持つ場合の、対象Dataや問い合わせの中身の展開（現在は識別子の表示にとどまります）
- 認可の宣言、画面の入出力の記述（機能の視点の編集は#1127の担当です）
- 書式の差し替え（出力用プラグイン）
- 受理前の候補を区別して載せる表示
