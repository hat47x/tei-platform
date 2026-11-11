# Relation Query未対応事項（2026-11）

この文書は、`relation-query`機構の実装により到達した現在地点を記録し、
未対応のリレーション種別・取得方式を整理する。

## 未対応のリレーション種別・取得方式

| 項目 | 状態 | 備考 |
|------|------|------|
| Lazy loading実行 | 未実装 | DSL/IRで`fetch lazy`は受理するが、実行時は`未実装です`エラーを返す |
| Batch loading実行 | 一部実装 | `fetch batch` + `batch N` は `LIMIT/OFFSET` で連続実行する。N+1防止のための最適化は未実施 |
| Temporal条件のSQL実行 | 未実装 | `AS OF` / `BETWEEN` はIR/DSLで受理し、SQLのWHERE句にそのまま出力するが、実際のtemporal evaluationはProvider側未対応 |
| 時間条件付きリレーションの意味判定 | 未対応 | bitemporal validity / transaction time の区別、履歴再計算は参照実装の範囲外 |
| 階層データ（JSON/XML）への拡張 | 設計のみ | RDBを対象としたSQL生成は実装済み。階層データへの適用は標準データモデルを介した拡張設計が必要 |
| グラフデータ（Property Graph）への拡張 | 設計のみ | 同上 |
| 実DBを用いた統合テスト | 要環境 | `TEI_RELATION_QUERY_PG_DSN` が設定された隔離DBで実行可能。ローカルCIでは未実行 |
| 性能測定（実環境） | 未実施 | N+1防止の単一SQL結合は確認済み。実DBでの実行時間・件数スケールは未測定 |

## 既知の制約

- 曖昧な経路は現状「複数のRelationが存在する場合」に unconditionally reject する。
  明示的選択による許可は今後の拡張ポイント。
- 循環参照は multi-step cycle (A -> B -> A) を reject する。
  自己参照 (A -> A) は許可する。
- 複合キーは SQL builder が最初の1組の key だけを使う。
  複数 key の完全な mapping は未検証。

## 次のステップ

1. Lazy loading の実行実装（必要に応じて複数クエリ or deferred join）
2. Batch loading の単一クエリ最適化（`LIMIT/OFFSET` ループから単一クエリ化）
3. Temporal 条件の Provider 依存実装（`FOR SYSTEM_TIME AS OF` など）
4. 実DB統合テストのCI化（disposable database の自動 provisioning）
5. 性能測定の自動化（query plan / row count / elapsed time の記録）
