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
| AC-8 | 一部 | Executorが返すEffectと宣言の照合、`Invoker.Recorder`、CLIの`-operation-log`、Studioの`records`、参照Storeの`Applied`照合 | 効果の不一致試験、記録の試験、画面の件数表示、適用操作の照合試験 | 実行時に発行されたCRUD操作の記録と差の保持 |
| AC-9 | 一部 | 検証レポートの`worldFingerprint`・`sourceIndexFingerprint`、`validation.Proposal`と`validation.Accept`、`tei validate -json`の`proposals`（[確認記録](XTEA_LOWCODE_DIAGNOSTIC_PROPOSAL_2026-11.md)） | 検証の版照合、修正案が版に結び付くこと、版不一致・版欠落・二重受理を拒否する試験 | 作成者が修正案を採る操作の画面・手順 |
| AC-10 | 一部 | `effectcommit`のreceipt、Commandアダプタ、CLIの`COMMAND_EFFECT_UNKNOWN`と`COMMAND_LIFECYCLE_REFUSED` | アダプタの単体試験、失敗の符号化試験、実PostgreSQLはDSN設定時のみ | Command再実行での「不明」状態の永続化と自動再実行の禁止 |
| AC-11 | 一部 | `effectcommit.Runner`の認可境界、`command.Invoker.Authorizer`、`AllowListAuthorizer`、CLI/StudioのAllowList、`legacyauthorization.CommandAuthorizer`（subject・action・resourceを既存規則へ接続）、`command.GrantAuthorizer`（能力が宣言した許可と発行済み許可の照合。部分許可・無宣言・不正宣言を拒否）。確認記録: [既存認可規則](XTEA_LOWCODE_COMMAND_AUTHORIZATION_2026-11.md)、[宣言許可の照合](XTEA_LOWCODE_GRANT_AUTHORIZATION_2026-11.md) | 認可付きPostgreSQL試験（DSN設定時のみ）、Authorizerの許可・拒否試験、CLIの`-allow-capability`、Studioの`ExecuteCommandWithPolicy`、Patternが宣言していても付与が無ければExecutorへ到達しない試験、能力の宣言許可と発行許可の照合試験 | 権限の正本・付与の管理、`security.authorize`への実運用接続 |
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
## CLI認可の観察（2026-11-06）

`examples/xtea-lowcode`に対して、許可外の能力が実行前に拒否されることを確認しました。

```text
tei view ... -execute -allow-capability sales.other
{"ok":false,"error":{"code":"COMMAND_FAILED","message":"COMMAND_UNAUTHORIZED: capability sales.submit_order is not allowed by the reference authorizer"}}
```

許可した場合は参照Executorが実行されます。

```text
tei view ... -execute -allow-capability sales.submit_order
outcome: submitted -> sales.order
effect: update sales.order
committed: true (reference unit of work)
persisted: false (durability: none)
```

これは参照用AllowListによる能力単位の拒否です。利用者・資源・メニュー表示・権限の付与は確認していません。

