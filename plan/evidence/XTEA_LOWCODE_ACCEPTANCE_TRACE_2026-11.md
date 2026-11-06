# X-TEA受容要件の実装対応（2026-11）

- 状態: **暫定**。受容要件の候補（`AC-1`〜`AC-11`、`RQ-C1`〜`RQ-C3`）と、現在の参照実装・検証を対応付ける作業記録です。
- 受容要件: [XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)
- 設計: [XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md](../design/XTEA_LOWCODE_RECEPTION_ARCHITECTURE_2026-11.md)
- 計画: [XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md](../workstreams/XTEA_LOWCODE_TRANSFER_PLAN_2026-11.md)
- 例: [examples/xtea-lowcode](../../examples/xtea-lowcode/README.md)

「実装済み（参照）」は、参照Executor・MemoryStore・CLI・Studio Hostで確認した範囲です。実Provider、実データベース、旧X-TEA Runtimeでの確認ではありません。受容判断は仮承認であり、最終採否は別途です。

## 対応表

| 要件 | 状態 | 主な実装 | 検証 | 残る未確認 |
| --- | --- | --- | --- | --- |
| AC-1 | 実装済み（参照） | Application Packageの検証・ビュー | Packageビュー、`TestXTEAExampleWorkspace` | 取り外し後の受理済み意味の保持を実DBで確認 |
| AC-2 | 一部 | フック宣言の検証、`internal/hook`の実行契約 | フック宣言・実装契約の単体試験 | 宣言外の能力呼出しを実行時に完全に拒否すること |
| AC-3 | 一部 | `role`・`required`・`immutable`・`values`、表示ラベル | 検証・ビュー・参照Executorの試験 | 幅・行数・整列・部品の表示指定 |
| AC-4 | 一部 | `layout`の順序投影 | layoutの検証・ビュー | 座標を含む配置とプロファイル比較 |
| AC-5 | 一部 | `Outcome`・`ExecutionError`・期間拒否 | Command実行境界・期間の試験 | 不存在・多重一致・不明を含む全結末の網羅 |
| AC-6 | 一部 | 値域・必須・候補キー・削除ポリシー | 参照Storeの単体・例の試験 | 読取専用データソースの宣言・実行前・実行時の三段 |
| AC-7 | 一部 | `-format html`・`print`、印刷テンプレート | HTML・printの単体試験 | 画面と帳票の版照合、manifestの版 |
| AC-8 | 未実装 | — | — | 宣言したCRUDと実行時の操作の差の保持 |
| AC-9 | 未実装 | — | — | 診断と受理済み意味の版の結合、修正案の非自動受理 |
| AC-10 | 一部 | `effectcommit`のreceipt、Commandアダプタ | アダプタの単体試験、実PostgreSQLはDSN設定時のみ | Command再実行での「不明」状態と自動再実行の禁止 |
| AC-11 | 一部 | `effectcommit.Runner`の認可境界 | 認可付きPostgreSQL試験（DSN設定時のみ） | Commandから権限を宣言・検査する経路、メニュー非依存の拒否 |
| RQ-C1 | 実装済み（参照） | Package、Semantic/Applicationの宣言 | Package・Applicationの検証と例 | Core外の型定義との組合せ |
| RQ-C2 | 一部 | フック宣言、CommandとObservationの分離 | 単体試験 | Plugin境界での完全な宣言強制 |
| RQ-C3 | 一部 | 期間、値域、型正規化 | 参照Store・例の試験 | 暦・会計期間・税・通貨・採番の適用期間と評価時点 |

## 実DB・実Providerで未確認のこと

- PostgreSQLの`effectcommit`は、`TEI_EFFECT_POSTGRES_DSN`を設定した環境でのみ試験します。この環境では未実行です。
- 実データベースの主キー・一意制約・外部キー・分離レベルは確認していません。
- 旧X-TEA Runtimeとの同一性、配布JARの利用条件は未確認です（#940、#941）。

## 次の判断

- AC-8・AC-9・AC-11の要求としての採否。
- AC-2・AC-5の実行時強制の範囲。
- 実Provider接続を先に行うか、残りの受入条件を先に具体化するか。
