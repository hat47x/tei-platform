# コア Subject 値型

`core/subject/`は、TEIの論理対象をソースファイル上の位置に依存せず識別する、Go標準ライブラリだけで利用できる最小の値型です。

- 正本は[subject.go](subject.go)の`Subject`（`Kind`、`ID`、`IsZero()`）です。
- 以前の`tei/internal/subject`は、既存利用者向けの**型エイリアスのみ**です。同じ型の二重実装を置きません。
- この移管は最小限のデータ型の配置を確定するもので、Service Contractの仕様、発行権限、プラグイン実行権限を新設するものではありません。

依存のないビルドと外部利用者からの呼び出しは、`bash tools/check-core-subject-isolation.sh`で隔離確認します。これにより実際の製品ソースの一部が`core/`へ移りますが、**コア全体の移管完了や別Goモジュールへの分割を意味しません**。
