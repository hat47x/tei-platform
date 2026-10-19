# reference-ddl-entry（#1803）

一つのデータベースの物理対応から、提供元（Provider）が出力できるDDLを表ごとに取り出す入口です。データベースにも、Workspaceにも書き込みません。

~~~text
go run ./tools/reference-ddl-entry/cmd/tei-ddl-entry -workspace reference/workspace -database sales_core
~~~

## 位置づけ

このツールはSQLの文法を持ちません。物理対応から望ましい物理の形を導き、空の状態との差分にして、提供元のレンダリング（`migration.Render`）へ渡します。SQLの組み立ては提供元の責務のままです（ADR-0015）。

出力は、宣言された物理対応から導いた**新規作成用**のDDLです。既存のデータベースへの移行文ではありません。

## 守っている境界

- 表ごとに提供元へ渡し、出力できた表（`tables`）と、提供元が出力できなかった表（`unresolved`、理由つき）を分けて返します。一つの表の拒否が、他の表を隠しません。
- 提供元が拒否した内容を、このツールで補いません。

## 現在分かっていること

PostgreSQLの提供元は、新規作成する表の**主キー**を、列の追加の後に`ALTER TABLE ... ADD PRIMARY KEY (...)`として出力します。名前は付けず、PostgreSQLが決めます（[設計案](../../plan/design/POSTGRES_CONSTRAINT_RENDERING_DESIGN_2026-10.md)の案C）。`reference/workspace`の`sales_core`は、二つの表とも出力できます。

一方、**一意制約と外部キーはまだ出力できません**。これらを持つ表は、提供元が未対応の変更種別として拒否し、`unresolved`に理由つきで出ます。黙って省くことはありません。主キーの変更（既存の表）も未対応です。出力文を実データベースで実行して確かめる検証は、まだ行っていません。

## 未実装

- 提供元の外部キー・一意制約を含む新規作成のレンダリング（提供元の担当。設計案のP2・P3）
- PostgreSQL以外の提供元
- DDLの書式の差し替え、コメントの付与
- ファイルへの出力
