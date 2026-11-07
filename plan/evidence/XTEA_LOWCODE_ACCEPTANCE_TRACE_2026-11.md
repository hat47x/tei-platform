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
| AC-2 | 一部 | フック宣言の検証、`internal/hook`の実行契約、`hook.Declared`（宣言した能力以外の呼出しを拒否）、`hook.CommandCaller`（フックの呼出しをCommand境界へ委譲。[確認記録](XTEA_LOWCODE_HOOK_DECLARED_CALLS_2026-11.md)） | フック宣言・実装契約の単体試験、宣言外呼出しの拒否試験、Command境界への委譲試験 | フック実行（`hook.Runner`）からの実際の配線 |
| AC-3 | 一部 | `role`・`required`・`immutable`・`values`、表示ラベル | 検証・ビュー・参照Executorの試験 | 幅・行数・整列・部品の表示指定 |
| AC-4 | 一部 | `layout`の順序投影 | layoutの検証・ビュー | 座標を含む配置とプロファイル比較 |
| AC-5 | 一部 | `Outcome`・`ExecutionError`・期間拒否、`ReferenceFailure`の符号（`record-missing`・`record-exists`・`value-rejected`・`rule-rejected`）と`FailureOutcome`、CLIの`COMMAND_REFUSED` | Command実行境界・期間の試験、参照Storeの拒否符号の試験、拒否と実行障害の区別の試験 | 多結果・多重一致の表現、実Providerでの結末 |
| AC-6 | 一部 | 値域・必須・候補キー・削除ポリシー、`COMMAND_DATASOURCE_READ_ONLY`（Applicationの読み取り専用datasource宣言を実行前に検査。[確認記録](XTEA_LOWCODE_READONLY_DATASOURCE_2026-11.md)） | 参照Storeの単体・例の試験、読み取り専用宣言の拒否・読み書き宣言の許容・宣言なしを推測しない試験 | データソース側の実行時強制（三段目） |
| AC-7 | 一部 | `-format html`・`print`、印刷テンプレート、`tei-semantic-fingerprint`（画面・帳票が読んだ意味の版） | HTML・printの単体試験、record・header-detail・browseの印刷面が版を持つ試験 | 表示（座標・用紙・部品）の版照合、manifestの版 |
| AC-8 | 一部 | Executorが返すEffectと宣言の照合、`Invoker.Recorder`、CLIの`-operation-log`、Studioの`records`、参照Storeの`Applied`照合と`Differences`（宣言外の種別は拒否、宣言と異なる対象への同種別適用は未解決の差として保持。CLIのテキスト・JSONとStudioの画面にも表示。操作を列挙しないExecutorは`AppliedUnobserved`で「未確認」と示す。[確認記録](XTEA_LOWCODE_OPERATION_DIFFERENCE_2026-11.md)） | 効果の不一致試験、記録の試験、画面の件数表示、適用操作の照合試験、カスケードを差として残す試験、実行結果の差の表示試験 | 実行時に発行されたSQL文・CRUD操作そのものの記録 |
| AC-9 | 一部 | 検証レポートの`worldFingerprint`・`sourceIndexFingerprint`、`validation.Proposal`と`validation.Accept`、`tei validate -json`の`proposals`、Studioの`ValidationResponse.proposals`（[確認記録](XTEA_LOWCODE_DIAGNOSTIC_PROPOSAL_2026-11.md)、[Studio表示の確認](XTEA_LOWCODE_STUDIO_PROPOSALS_2026-11.md)） | 検証の版照合、修正案が版に結び付くこと、版不一致・版欠落・二重受理を拒否する試験 | 作成者が修正案を採る操作の画面・手順 |
| AC-10 | 一部 | `effectcommit`のreceipt、Commandアダプタ、CLIの`COMMAND_EFFECT_UNKNOWN`と`COMMAND_LIFECYCLE_REFUSED` | アダプタの単体試験、失敗の符号化試験、実PostgreSQLはDSN設定時のみ | Command再実行での「不明」状態の永続化と自動再実行の禁止 |
| AC-11 | 一部 | `effectcommit.Runner`の認可境界、`command.Invoker.Authorizer`、`AllowListAuthorizer`、CLI/StudioのAllowList、`legacyauthorization/commandbinding.CommandAuthorizer`（subject・action・resourceを既存規則へ接続）、`command.GrantAuthorizer`（能力が宣言した許可と発行済み許可の照合。部分許可・無宣言・不正宣言を拒否）。確認記録: [既存認可規則](XTEA_LOWCODE_COMMAND_AUTHORIZATION_2026-11.md)、[宣言許可の照合](XTEA_LOWCODE_GRANT_AUTHORIZATION_2026-11.md) | 認可付きPostgreSQL試験（DSN設定時のみ）、Authorizerの許可・拒否試験、CLIの`-allow-capability`、Studioの`ExecuteCommandWithPolicy`、Patternが宣言していても付与が無ければExecutorへ到達しない試験、能力の宣言許可と発行許可の照合試験 | 権限の正本・付与の管理、`security.authorize`への実運用接続 |
| AC-12 | 未実装 | 設計のみ（[作者環境の受容設計](../design/XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md)） | — | 面ごとの配置検査、意味の役割と表示指定の分離、生成物の未受理扱い | Package、Semantic/Applicationの宣言 | Package・Applicationの検証と例 | Core外の型定義との組合せ |
| RQ-C2 | 一部 | フック宣言、CommandとObservationの分離 | 単体試験 | Plugin境界での完全な宣言強制 |
| RQ-C3 | 一部 | 期間、値域、型正規化、`calendar`宣言（年度開始月・週開始・休日Data）と期間との結合（[確認記録](XTEA_LOWCODE_CALENDAR_DECLARATION_2026-11.md)） | 参照Store・例の試験、暦宣言の構文・検証試験（7件） | 税・通貨・採番の適用期間と評価時点、暦からの日付計算 |

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
































