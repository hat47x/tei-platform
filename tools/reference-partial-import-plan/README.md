# reference-partial-import-plan（#1805）

別のWorkspaceから選んだData定義を取り込む場合に、取込み先で何が起きるかを分類する読取り専用の参照です。定義の書込みも、受理もしません。

~~~text
go run ./tools/reference-partial-import-plan/cmd/tei-partial-import-plan \
  -source <取込み元Workspace> -target <取込み先Workspace> -data <Data ID,...>

# 取込み先が読み取り専用で参照している名前空間と、所有者の境界を渡す
go run ./tools/reference-partial-import-plan/cmd/tei-partial-import-plan \
  -source <取込み元> -target <取込み先> -data <Data ID,...> \
  -reference-only-namespaces shared,sales -source-owner team-a -target-owner team-b
~~~

## 分類

| 状態 | 意味 |
| --- | --- |
| `new` | 取込み先にない |
| `identical` | 同じ識別子で、意味を持つ構造が同じ |
| `update` | 同じ識別子で、構造が異なる。差異を`changes`に列挙する |
| `conflict` | 識別子は別だが名称が重なる、または識別子は同じで名前空間が異なる |

## 守っている境界

- 識別子の一致を、意味の同一性の根拠にしません。名称の重なりや名前空間の違いを検出します。
- 取込み先にない型、取込み対象外の関連先は、`unresolved`として保持します。黙って落としません。
- 更新が、取込み先の他の定義（関連のキー、候補キー）が使う属性を削除・型変更する場合は`dependent-definition-affected`とし、取り込めないと判定します。
- `importable`は、競合と未解決がない場合だけ`true`です。`applied`は常に`false`です。
- 取込み元の選択に存在しないIDは`not-in-source`として残します。
- 取込み先が読み取り専用で参照している名前空間（#1570）の定義は、取込み対象にしません。`referenceOnly`に`reference-only-namespace`として出し、`importable`を偽にします。
- 取込み元と取込み先の所有者が違う場合は、`different-ownership-boundary`として診断し、取込み不可にします。

## 受理境界への接続

`importable`で、すべてが`new`の計画は、Shared Acceptance（`tools/reference-shared-acceptance`）が受け付ける変更候補（`add-data`）へ変換できます。

~~~text
REV=$(go run ./tools/reference-shared-acceptance revision -workspace <取込み先> | grep -o 'sha256:[0-9a-f]\{64\}')
go run ./tools/reference-partial-import-plan/cmd/tei-partial-import-plan   -source <取込み元> -target <取込み先> -data <Data ID,...>   -candidate-out candidate.json -base-revision "$REV"
go run ./tools/reference-shared-acceptance preview -workspace <取込み先> -candidate candidate.json
~~~

- 候補は、出力先が既にあれば作りません。受理（`accept`）はこのツールでは行いません。
- 現在の受理境界は追加の演算（`add-*`）だけを受け付けます。`update`と`identical`は候補にせず、拒否します。近い別の演算へ読み替えません。
- 取込み元にしかない型（`type-only-in-source`）は`unresolved`のため、候補を作りません。型の取込みは未対応です。
- 取込み元の2つのWorkspaceで、標準型だけを使うDataを追加する候補が、`preview`の全11段階を通ることを手元で確認しました。

## 未実装

- `update`の取込み（受理境界に該当する演算がありません）
- 型の取込み
- 型、機能、業務手順など、Data以外の定義
- 取込み記録の保存
