# SQLite introspection の制限と未対応事項

このドキュメントは、TEI の SQLite スキーマ introspection 実装で未対応または
制限がある事項を整理する。既存 Issue へのリンクと、PoC 段階でのsafe defaultsを記載する。

## 1. SQLite の動的型付き列（動的型付き言語的挙動）

SQLite は列の型宣言を「アフィニティ」として扱い、実際の値の型は実行時に決定される。
TEI の `mapPhysicalType()` は宣言された型名を正規化するが、以下は保証しない:

- `INTEGER` 列に格納された TEXT 値
- `TEXT` 列に格納された INTEGER 値
- BLOB 列に格納された任意の型

**safe default**: 宣言型を正規化して `db.integer` / `db.text` / `db.decimal` / `db.blob` に写す。
実行時の値のアフィニティ逸脱は、上位層の Validation で検証する。

## 2. CHECK 制約のパース

SQLite の CHECK 制約は `sqlite_master.sql` から取得できるが、
式の構造は複雑で、依存関係のない孤立した式として扱う。

**safe default**: CHECK 制約は `UniqueConstraint` / `Index` として観測するが、
TEI の標準 Candidate Key / Relation には変換しない。診断 `check-constraint-skipped` をemitする。

## 3. ALTER TABLE の制限

SQLite の `ALTER TABLE` は以下の変更に制限される:

- `RENAME TABLE`
- `ADD COLUMN`（制限付き）
- `DROP COLUMN` (3.35.0+)
- `RENAME COLUMN` (3.25.0+)
- `ADD CONSTRAINT` / `DROP CONSTRAINT` / `ALTER COLUMN TYPE` は不可

**safe default**: TEI の `schemadiff` / `plan-migration` は、
PostgreSQL の完全な DDL を前提に設計されている。SQLite に対しては、
変更検出結果を「破壊的変更」として報告し、実際の migration は手動または
専用の SQLite 移行ツール（例: `golang-migrate` の SQLite 方言）へ委譲する。

## 4. 外部キーの宣言的制約と実行時強制

SQLite は FK 制約を `PRAGMA foreign_keys = ON` で初めて強制する。
 introspection 時点では、制約が宣言されているか、実行時に有効かは分からない。

**safe default**: FK を `Relation` として写すが、`referential-integrity-not-enforced` 診断をemitする。

## 5. 部分インデックスと式インデックス

SQLite は `CREATE INDEX ... WHERE ...` と式インデックスをサポートするが、
 introspection で `sqlite_master` から式の内容を取得する方法は限定される。

**safe default**: 式インデックスは `Index.Expression` が `nil` として観測される場合、
診断 `expression-index-skipped` をemitし、TEI の AccessPath としては記録しない。

## 6. ビューの依存性追跡

SQLite のビュー定義は `sqlite_master.sql` に格納されるが、
ビューが参照するテーブルや他のビューの依存関係は静的に解析する必要がある。

**safe default**: ビューは `TableMeta.Kind == "view"` として観測し、
物理テーブルと区別して `semantic.Data` に写す。依存関係の深い検証は未実施。

## 7. トランザクション分離と並行性

SQLite のデフォルトは `SERIALIZABLE` だが、WAL モードと `PRAGMA journal_mode`
で挙動が変わる。 introspection は単一接続で行うため、並行 DDL の検出は不可能。

**safe default**: 観測時点のスナップショットを `CapturedAt` で固定する。
並行変更の検出はアプリケーション層で `contentFingerprint` の比較により
 eventual に検出する。

## 8. 権限制限

SQLite はファイルシステムレベルの権限のみをサポートし、
ロールベースのアクセス制御はない。

**safe default**: 権限不足は `os.Open` / `sql.Open` のエラーとして伝播する。
診断 `permission-denied` をemitする。

## 9. 既存ブランチのブロッカー（この実装とは無関係）

- `internal/query/standard` に import cycle が存在する（PoC 範囲外）
- `cmd/tei/cli_operations_artifacts.go` の `ReadBuildIdentity` が未定義

これらは SQLite introspection 機能の検証範囲外である。
別 Issue で個別に修正する。

---

## 参照 Issue

- [#3676](https://github.com/hat47x/tei-platform/issues/3676) SQLite introspection の実装
- [#3681](https://github.com/hat47x/tei-platform/issues/3681) PostgreSQL の row lock・再接続の確認
- [#3807](https://github.com/hat47x/tei-platform/issues/3807) 共通失敗モデルの provider error code 分類
- [#2999](https://github.com/hat47x/tei-platform/issues/2999) Command 段階の再起動境界
