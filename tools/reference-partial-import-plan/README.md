# reference-partial-import-plan（#1805）

別のWorkspaceから選んだData定義を取り込む場合に、取込み先で何が起きるかを分類する読取り専用の参照です。定義の書込みも、受理もしません。

~~~text
go run ./tools/reference-partial-import-plan/cmd/tei-partial-import-plan \
  -source <取込み元Workspace> -target <取込み先Workspace> -data <Data ID,...>
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

## 未実装

- 取込みの受理（#267）と、取込み先への反映
- 型、機能、業務手順など、Data以外の定義
- 取込み元が別の所有境界にある場合の診断（#1132）
- 取込み記録の保存
