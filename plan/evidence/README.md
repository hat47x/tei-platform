# 受入条件と検証根拠

要求が受け入れシナリオや検証根拠にどう結び付くかを整理する文書を置きます。要求そのものは[`requirements/`](../../requirements/README.md)、段階判定の記録と実行手順は[`gate/`](../gate/README.md)、現在の実装状況は[`docs/status.md`](../../docs/status.md)を参照してください。

- [製品品質の検証計画](PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md)
- [製品要求と実証状況の照合](PRODUCT_REQUIREMENT_GAP_AUDIT.md)
- [E1合成証拠の8件原本保全と再検証](TEI_E1_SYNTHETIC_ARCHIVE_README_20261013.md)（試験用証拠。製品Gateは未資格）
- [E2a-1実PostgreSQL先行証拠の原本保全](TEI_E2A1_REAL_POSTGRES_ARCHIVE_20261013.md)（手書きSQL／同一プロセス別pool。製品G2は未資格）
- [E1合成WebとE2a-1実DBの横断証拠誤結合防止](TEI_E1_E2_JOIN_LIMITS_20261013.md)（異なる版・Application・Authority。8件のG2阻害条件）
- [#1803 新規作成DDLの実PostgreSQL実行](POSTGRES_DDL_ENTRY_REAL_DB_20261020.md)（#2103のheadで主キー・一意制約・外部キー。tei-ddl-entryの表ごとの出力で外部キーの順序が崩れる例を含む）
- [RQ-025・RQ-026の受入根拠](RQ025_RQ026_ACCEPTANCE_TRACE.md)
- [再生手順の再現性契約](R7B_REPLAY_RECIPE_REPEATABILITY_CONTRACT_2026-09.md)
- [知識移植の実行可能な検証](KNOWLEDGE_PORTABILITY_EXECUTABLE_EVIDENCE_2026-09.md)
- [X-TEA固定版 NotNull／Relationship任意性／DDL・Web認証の静的対照](XTEA_NOTNULL_RELATION_WEB_AUTH_STATIC_CROSSCHECK_20261026.md)（旧実装の比較証拠。Accepted Meaning・本番適格性の実証ではない）
- [TEI Studio PoCの証拠採取手順](TEI_STUDIO_POC_EVIDENCE_RUNBOOK_20261014.md)
- [E1合成証拠の索引（JSON）](TEI_E1_SYNTHETIC_MANIFEST_20261012_38267656696.json)と、その原本[`tei-e1-synthetic-38267656696/`](tei-e1-synthetic-38267656696/)（試験用証拠。製品Gateは未資格。バイト列で照合するため[`.gitattributes`](../../.gitattributes)で改行変換を除外している）
- [E2a-1実PostgreSQL先行証拠の受領記録（JSON）](TEI_E2A1_REAL_POSTGRES_RECEIPT_38311557882.json)（上の保全文書が参照する原本。製品G2は未資格）
- [#1803 新規作成DDLの実PostgreSQL実行の記録（JSON）](POSTGRES_DDL_ENTRY_REAL_DB_20261020.json)（上の説明文書が参照する、生の結果）
- [操作面・成果物・束縛実行の検証記録](SURFACE_AND_ARTIFACT_VERIFICATION_2026-11.md)
- [候補ビューの実変更履歴での負担測定](CANDIDATE_VIEW_REAL_HISTORY_MEASUREMENT_2026-11.md)（`reference/workspace` の4変更。合成ではない変更で候補件数とSource修正量を測定。major変更の実履歴はなし）
- [実データベースでの確認手順](REAL_DATABASE_VERIFICATION_PROCEDURE_2026-11.md)（PostgreSQL 16.14で実行。CAS試験の同期不足を修正し、DDL・NULL・Web E2Eまで確認。健康診断receiptで `TRUSTED_FULL_HEALTH_NOT_RECORDED` を解消し、残る2診断は暫定オーナー判断。実Provider・本番Authorityは未実行）
- [健康診断receipt（`005bbee7` / Actions 42152908986）](QUALIFICATION_FULL_HEALTH_005BBEE7_42152908986.json)と[暫定audit結果](REAL_PROVIDER_PROVISIONAL_AUDIT_005BBEE7.json)（`UNQUALIFIED`、残る診断は `E2_PROVIDER_GUARANTEE_NOT_ATOMIC` と `PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE`）
- [D8〜D11暫定経路のaudit](PROVISIONAL_CHAIN_AUDIT_005BBEE7.json)と[Runner Evidence](PROVISIONAL_CHAIN_RUNNER_EVIDENCE_005BBEE7.json)（`atomic-with-commit` の暫定宣言。`PROVISIONAL_GRANT_NOT_G2_EVIDENCE` 等は暫定オーナー判断。chain testは単一経路と同じ健康診断receipt入力に対応）
- [chain経路の健康診断receipt（`f887ab97` / Actions 42195019838）](QUALIFICATION_FULL_HEALTH_F887AB97_42195019838.json)と[receipt適用後のaudit](PROVISIONAL_CHAIN_AUDIT_F887AB97.json)（`TRUSTED_FULL_HEALTH_NOT_RECORDED` が解消し、残る診断は暫定3件）
- [Workspace Source Readerの環境別Evidence](WORKSPACE_SOURCE_READER_ENV_EVIDENCE_2026-11.md)（Windows 11・WSL2 Ubuntuでの許容／拒否／NOT RUNとLoadコスト。#2571）
- [暫定チェーンの実行記録](TEI_PROVISIONAL_CHAIN_E2E_20261103.md)（D8〜D11を通した一改訂E2Eの実PostgreSQL実行と監査結果。G1・G2は未合格）
- [X-TEA例のStudioサーバー起動確認](XTEA_LOWCODE_EXAMPLE_STUDIO_SMOKE_2026-11.md)（`examples/xtea-lowcode`のserver buildと`/health`。画面・Wails RPC・Desktop WebViewは未確認）
- [既存認可規則とCommand境界の接続確認](XTEA_LOWCODE_COMMAND_AUTHORIZATION_2026-11.md)（参照実装の試験記録。権限の正本・付与の管理は未接続）
- [診断の修正案と受理境界の確認](XTEA_LOWCODE_DIAGNOSTIC_PROPOSAL_2026-11.md)（参照実装の試験記録。作成者向けの受理操作は未実装）
- [能力の宣言許可と発行許可の照合確認](XTEA_LOWCODE_GRANT_AUTHORIZATION_2026-11.md)（参照実装の試験記録。許可の管理と実行経路への接続は未実装）
- [読み取り専用datasourceへの書込み拒否の確認](XTEA_LOWCODE_READONLY_DATASOURCE_2026-11.md)（参照実装の試験記録。データソース側の実行時強制は未実装）
- [フックの宣言外呼出し拒否の確認](XTEA_LOWCODE_HOOK_DECLARED_CALLS_2026-11.md)（参照実装の試験記録。Command境界への接続は未実装）
- [Studio検証結果の修正案表示の確認](XTEA_LOWCODE_STUDIO_PROPOSALS_2026-11.md)（参照実装の試験記録。案を受理する操作は未実装）
- [暦の宣言と期間への結合の確認](XTEA_LOWCODE_CALENDAR_DECLARATION_2026-11.md)（参照実装の試験記録。日付計算と税・通貨・採番は未実装）
- [X-TEA受容要件の実装対応](XTEA_LOWCODE_ACCEPTANCE_TRACE_2026-11.md)（`AC-1`〜`AC-11`・`RQ-C1`〜`RQ-C3`の参照実装と未確認事項。受容判断は仮承認）
- [実DBで実行時の判断を確かめる計画](REAL_DATABASE_RUNTIME_DECISION_PLAN_2026-11.md)（期間・権限・効果の判断を実DBで見る案。**NOT RUN**）





- [Studio固定Host負例のCI実行記録](STUDIO_FIXED_HOST_CI_2026-11-07.md)（#2948のGo版修正後に試験本体へ到達し、固定Hostのroot置換負例が失敗した記録。#3061で追跡）
- [宣言と実行の差の保持](XTEA_LOWCODE_OPERATION_DIFFERENCE_2026-11.md)（#3055の記録）
- [Go選択Module閉包の事前観測](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)（`cmd/tei`は現行の隔離観測で`missing-module`、自己完結したtoolは`supported`。Build・Artifact・権限は未実行。SB-0a / #2831）
- [構成Kernel移行Gateの最初の実証パケット実行記録](COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07/README.md)（固定HEADでA・B・C三経路を別々に実行し、`operations`と`artifacts`が同じ`modelRevision`を共有したことだけを記録。三経路の同一業務Request接続・Owner受理・実DB効果は未確認）
- [構成Gateの次の一辺を依存グラフで測る](COMPOSITION_NEXT_EDGE_STATIC_SURVEY_2026-11-07/README.md)（Goファイル2876件と実行単位167件のimport閉包を走査し、`internal/operations`と`internal/composition`を同居させるファイル・実行単位が0件、`internal/servicehost`の非試験呼出し元も0件であることを記録。実行時の不接続や権限は示さない）

