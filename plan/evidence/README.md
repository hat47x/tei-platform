# 受入条件と検証根拠

要求が受け入れシナリオや検証根拠にどう結び付くかを整理する文書を置きます。要求そのものは[`requirements/`](../../requirements/README.md)、段階判定の記録と実行手順は[`gate/`](../gate/README.md)、現在の実装状況は[`docs/status.md`](../../docs/status.md)を参照してください。

- [製品品質の検証計画](PRODUCT_QUALITY_EVIDENCE_PLAN_2026-09.md)
- [製品要求と実証状況の照合](PRODUCT_REQUIREMENT_GAP_AUDIT.md)
- [Plugin-firstのCLI/Studio共通InspectorへのD1 Fit/Gap](PLUGIN_FIRST_D1_AUTHORING_ROUTE_FIT_GAP_2026-11.md)（現行Go経路の静的監査と具体的な検証計画。実Provider二Consumerは未実証）
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
- [フックの宣言外呼出し拒否の確認](XTEA_LOWCODE_HOOK_DECLARED_CALLS_2026-11.md)（参照実装の試験記録。Command境界への委譲は[#3108](https://github.com/hat47x/tei-platform/pull/3108)）
- [フック実行の配線と失敗の区別の確認](XTEA_LOWCODE_HOOK_RUNNER_OUTCOME_2026-11.md)（参照実装の試験記録。Runnerからの配線、NotSatisfiedと実行失敗の区別、参照規則。実PluginとCLI・Studioの配線は未実装）
- [フック宣言の印刷面への反映の確認](XTEA_LOWCODE_HOOK_PRINT_SURFACE_2026-11.md)（参照実装の試験記録。画面と同じフック宣言が印刷面にも出る。実印刷・用紙面の版照合は未確認）
- [フックの位相順序と途中停止の確認](XTEA_LOWCODE_HOOK_PHASE_SEQUENCE_2026-11.md)（参照実装の試験記録。位相の実行順、不成立での停止位置、`warn`の続行、未対応位相の拒否。CLI・Studioへの配線は未実装）
- [フックの位相ごとの読み取り範囲の確認](XTEA_LOWCODE_HOOK_PER_PHASE_READS_2026-11.md)（参照実装の試験記録。狭い宣言の位相が広い宣言の値を受け継がないこと、未観測での停止。実データ取得は未実装）
- [フックが読むのは宣言した属性だけであることの確認](XTEA_LOWCODE_HOOK_DECLARED_READS_2026-11.md)（参照実装の試験記録。未観測・宣言外の読み取りを実装へ渡す前に拒否。実データの取得と`produces`は未実装）
- [フックが使う能力のcommand Pattern不在の拒否](XTEA_LOWCODE_HOOK_COMMAND_REACHABILITY_2026-11.md)（参照実装の試験記録。実行経路の無いフック宣言を検証で拒否。複数commandの曖昧さは未診断）
- [Studio検証結果の修正案表示の確認](XTEA_LOWCODE_STUDIO_PROPOSALS_2026-11.md)（参照実装の試験記録。案を受理する操作は未実装）
- [暦の宣言と期間への結合の確認](XTEA_LOWCODE_CALENDAR_DECLARATION_2026-11.md)（参照実装の試験記録。日付計算と税・通貨・採番は未実装）
- [暦からの日付導出の確認](XTEA_LOWCODE_CALENDAR_CALCULATION_2026-11.md)（参照実装の試験記録。期間キー・税・通貨・採番は未実装）
- [期間キーの導出の確認](XTEA_LOWCODE_PERIOD_KEY_2026-11.md)（参照実装の試験記録。月内締めの期間化と状態との結合は未実装）
- [税率の適用期間と端数処理の確認](XTEA_LOWCODE_TAX_RATE_2026-11.md)（参照実装の試験記録。税率Dataの宣言との結合と税区分の運用は未実装）
- [通貨換算の率の種類と方向の確認](XTEA_LOWCODE_CURRENCY_RATE_2026-11.md)（参照実装の試験記録。Data宣言との結合と年次・月次の優先順位は未実装）
- [採番の契約の確認](XTEA_LOWCODE_NUMBERING_2026-11.md)（参照実装の試験記録。採番Data・取引との結合は未実装）
- [税率Dataの宣言と行の結合の確認](XTEA_LOWCODE_TAX_RATE_BINDING_2026-11.md)（参照実装の試験記録。Workspace例への追加と実DBの実測は未実施）
- [為替Dataの宣言と行の結合の確認](XTEA_LOWCODE_CURRENCY_BINDING_2026-11.md)（参照実装の試験記録。例への追加と年次・月次の優先順位の宣言は未実施）
- [採番Dataの宣言と行の結合の確認](XTEA_LOWCODE_NUMBERING_BINDING_2026-11.md)（参照実装の試験記録。取引内の発行・取消、同時実行、例への追加は未実装）
- [休日Dataの宣言と行の結合の確認](XTEA_LOWCODE_HOLIDAY_BINDING_2026-11.md)（参照実装の試験記録。例への追加と複数暦の識別は未実装）
- [標準業務の属性役割語彙の確認](XTEA_LOWCODE_ROLE_VOCABULARY_2026-11.md)（参照実装の試験記録。例への追加は未実装）
- [役割と型の対応検査の確認](XTEA_LOWCODE_ROLE_TYPE_CHECK_2026-11.md)（参照実装の試験記録。要求プリミティブの拡張は未実装）
- [例の標準業務Dataの宣言と結合の確認](XTEA_LOWCODE_EXAMPLE_STANDARD_DATA_2026-11.md)（参照実装の試験記録。行データ・実DBは未確認）
- [標準業務Dataの投影の確認](XTEA_LOWCODE_STANDARD_BUSINESS_PROJECTION_2026-11.md)（参照実装の試験記録。行データとWails RPC経由の確認は未実施）
- [複数暦の休日Dataの確認](XTEA_LOWCODE_MULTI_CALENDAR_HOLIDAY_2026-11.md)（参照実装の試験記録。例への追加は未実装）
- [休日Dataの暦識別値の検査の確認](XTEA_LOWCODE_CALENDAR_DOMAIN_2026-11.md)（参照実装の試験記録。例への追加は未実装）
- [休日の結合と暦識別検査を一続きにする確認](XTEA_LOWCODE_HOLIDAY_RESOLVE_2026-11.md)（参照実装の試験記録。例での複数暦利用と実DBは未確認）
- [為替の年次フォールバックの宣言の確認](XTEA_LOWCODE_CURRENCY_FALLBACK_2026-11.md)（参照実装の試験記録。例への追加は未実装）
- [為替フォールバックの宣言語彙の確認](XTEA_LOWCODE_CURRENCY_FALLBACK_ROLE_2026-11.md)（参照実装の試験記録。例への追加と別Data構成は未実装）
- [採番の発行・取消の取引結合の確認](XTEA_LOWCODE_NUMBERING_TRANSACTION_2026-11.md)（参照実装の試験記録。取引実装との結合と実DBは未確認）
- [採番の予約と取引の結合の確認](XTEA_LOWCODE_NUMBERING_RESERVATION_TRANSACTION_2026-11.md)（参照実装の試験記録。InvokerのUoW共通化と実DBは未実装）
- [X-TEA受容要件の実装対応](XTEA_LOWCODE_ACCEPTANCE_TRACE_2026-11.md)（`AC-1`〜`AC-11`・`RQ-C1`〜`RQ-C3`の参照実装と未確認事項。受容判断は仮承認）
- [実DBで実行時の判断を確かめる計画](REAL_DATABASE_RUNTIME_DECISION_PLAN_2026-11.md)（期間・権限・効果の判断を実DBで見る案。**NOT RUN**）





- [Studio固定Host負例のCI実行記録](STUDIO_FIXED_HOST_CI_2026-11-07.md)（#2948のGo版修正後に試験本体へ到達し、固定Hostのroot置換負例が失敗した記録。#3061で追跡）
- [宣言と実行の差の保持](XTEA_LOWCODE_OPERATION_DIFFERENCE_2026-11.md)（#3055の記録）
- [Go選択Module閉包の事前観測](TEI_GO_MODULE_CLOSURE_PREFLIGHT_2026-11-07.md)（`cmd/tei`は現行の隔離観測で`missing-module`、自己完結したtoolは`supported`。Build・Artifact・権限は未実行。SB-0a / #2831）
- [Go Module供給のcorrupt-supply分類](TEI_GO_MODULE_CORRUPT_SUPPLY_2026-11-07.md)（#2831 SB-0b。archive破損と展開済み改変の実観測、`go list`では展開済み改変が見えないこと、`go mod verify`が捕まえることの記録。供給manifestと`h1:`照合の接続は未実装）
- [Go Module供給の実観測](TEI_GO_MODULE_SUPPLY_OBSERVATION_2026-11-07.md)（#2831 SB-0bの前提。供給の有無・破損・改変が分類と成果物へどう現れるかの実測。TEIのBuild経路は未実行）
- [構成Kernel移行Gateの最初の実証パケット実行記録](COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07/README.md)（固定HEADでA・B・C三経路を別々に実行し、`operations`と`artifacts`が同じ`modelRevision`を共有したことだけを記録。三経路の同一業務Request接続・Owner受理・実DB効果は未確認）
- [構成Gateの次の一辺を依存グラフで測る](COMPOSITION_NEXT_EDGE_STATIC_SURVEY_2026-11-07/README.md)（Goファイル2876件と実行単位167件のimport閉包を走査し、`internal/operations`と`internal/composition`を同居させるファイル・実行単位が0件、`internal/servicehost`の非試験呼出し元も0件であることを記録。実行時の不接続や権限は示さない）
- [構成の二つの層は、いまも同じ場所に居ないか（再調査）](COMPOSITION_COLOCATION_RESURVEY_2026-11-08/README.md)（先行調査の2つの数を現在のmainで取り直し、両方をimportするファイル0件・servicehostの非試験importer 0件が維持されることを確認。あわせてパッケージの依存閉包でも数え、両方の閉包に入るパッケージが1件（reference/interface-adapter、間接import）だけあること、mainパッケージでは0件のままであることを記録。先行記録は訂正しない）
- [構成Kernel移行Gateの依存の形を、いまのmainで確かめる](COMPOSITION_GATE_DEPENDENCY_RECHECK_2026-11-08/README.md)（Gate 4.1 の3つの主張を依存閉包で再測。A(cmd/tei)は composition を含まず operations を含み、B(go-host)は operations を含まず composition を含み、servicehost は非試験 importer 0件（ファイル3件はすべて _test.go、閉包でも自分以外0件）。3つとも成立し、A/Bの非対称が個別PASSを混ぜない根拠であることを示す。Gateの合否は判定しない）
- [宣言されたPatternと導出された操作はどこで結べるか](PATTERN_KIND_OPERATION_MAPPING_2026-11-07/README.md)（同じWorldから `tei operations` の導出操作8件と `tei artifacts` の `pattern` 5件を取り、Data ID・能力ID・Relation target の三つの手掛かりごとに一致と限界を記録。受理済み基準では書込み操作の `binding` が全件nullで、読み取り専用の `command` は構造上どの導出操作とも結ばれない。結合規則の採否は示さない）
- [操作とPatternの突き合わせを参照実装で試す](OPERATION_PATTERN_CROSSWALK_REFERENCE_2026-11-07/README.md)（W3の対応表を `tools/reference-operation-pattern-crosswalk` として実装し、受理済みと見積もり候補のWorkspaceに当てた記録。`browse`/`record`/`header-detail` は一致4件、`command` は効果を宣言しない能力で一律 `out-of-scope`、`reference` は参照先Dataに `read` が無いため `missing-input`。既定値で埋めないことと、W3未着手のままであることを示す）
- [突き合わせ規則の反例](OPERATION_PATTERN_CROSSWALK_COUNTEREXAMPLES_2026-11-07/README.md)（同じWorkspaceに宣言を一つだけ足す5つの場合で、一致・不一致・衝突が1件ずつ変わることを記録。効果を宣言する能力は `command` を一致させ、参照先DataのPersistenceは `reference` を一致させ、同じDataを指す2つ目の `browse` は衝突として両方の名前を出して選ばない。効果の宣言だけでは標準操作が導出されないことも `missing-input` として残る）
- [操作面はCLIの公開面に出ているか](OPERATIONS_COMMAND_PUBLICATION_GAP_2026-11-07/README.md)（`tei commands` は `selected-read-only-commands` の19件だけを載せ、CLIは53件を受け付ける。`operations` は公開されておらず、成功時は単段JSONで `ok`/`result` を持たず、失敗時はstdout 0バイトでエンベロープにならない。W5の3条件のうち満たすのは「動く」だけで、公開の可否は15章の未決1のまま）
- [操作種別の語彙はどこに置くか](OPERATION_VOCABULARY_PLACEMENT_2026-11-07/README.md)（プラグインの `CapabilityKind` は7種の閉じた列挙で未知を宣言時に拒否し、操作面の種別は生成コード内の文字列で検査が無い。`operationKindRank` は未知を rank 5 として通す。設計49行が掲げる「規則の版で語彙を管理する」は未実装であることを示す。どちらへ置くかの判断は15章の未決2のまま）
- [自律区分は何から導かれているか](OPERATION_AUTONOMY_DECLARATION_SURVEY_2026-11-07/README.md)（操作面が `unattended` を付けるのは読み取り操作だけで、`create`/`update` は `attended`、`human-gated` は0件。`Environment` にサービス主体の欄が無く、`surface.go` が `"human-gated"` を代入する箇所も無い。設計8章の「判断段階が未解決なら human-gated に倒す」は未実装で、未決3は宣言の置き場所と生成規則の新設の判断になる）
- [実行時の照合は何を止められるか](OPERATION_DRIFT_CHECK_SURVEY_2026-11-07/README.md)（属性を1つ足すと `worldFingerprint` と `modelRevision` は変わるが、操作5件の `basis` は全件不変。`tei operations` に照合値を渡すフラグが無く、`OPERATION_SURFACE_STALE` はリポジトリ全体で0件、実行の入口 `period_guard.go` は指紋を比べない。未決4は「既定で有効にするか」の前に照合値の受け渡し経路の判断が要る）
- [操作面のdrift照合を参照実装で試す](OPERATION_DRIFT_CHECK_REFERENCE_2026-11-07/README.md)（`tools/reference-operation-drift-check` として案(a)「呼出し側が照合値を渡す」を実装。同じWorkspace+保存値は `current`、属性を1つ足したWorkspace+同じ値は `stale` で終了コード1、未指定は `unchecked` として current と区別する。Basis は属性追加で変わらないため比べない。既定で有効にするかは15章未決4のまま）
- [宣言されたプラグイン能力は環境から選ばれているか](PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/README.md)（`tools/reference-plugin-capability-coverage` で参照Workspaceを集計し、宣言12件のうち6件がどの環境からも選ばれていないことを記録。database-provider 2件は instance が提供技術名を指すための数え方の限界、editor-intelligence はプラグインが明示選択されているのに能力のBindingが無い例、migration.atlas は能力の一部だけが選ばれている例。選ばれていないことを欠陥とはしない）

- [database-providerの選択を技術名ではなく能力で解決する](PLUGIN_CAPABILITY_COVERAGE_TECHNOLOGY_RESOLUTION_2026-11-08/README.md)（先行記録では instance の提供技術名を能力IDと文字列比較していたため database-provider 2件を数えなかった。既存の validation と同じ Capability.SupportsDatabaseProvider で解決すると両方が selected になり、未選択は6件から4件に減る。先行記録は履歴として残し、訂正しない）- [操作面は宣言されたPatternを読んでいるか](OPERATION_SURFACE_DECLARED_PATTERN_GAP_2026-11-07/README.md)（受注Workspaceと見積もり候補の重ねで`tei operations`を比較し、World指紋は変わるのに操作面8件と`summary.declared=0`が不変であることを記録。W3未着手の設計記述と一致し、AのOperationとBの業務ActionをA側で結ぶ手掛かりが現状無いことを示す。権限・意味の受理は示さない）
- [環境のBindingを、能力の対応表で一括して解決する](PLUGIN_SELECTION_MATRIX_REFERENCE_2026-11-08/README.md)（`tools/reference-plugin-selection-matrix` で6種類のBindingを既存の `Supports*` だけで解決し、参照Workspaceの24件のうち22件が selected・2件が missing・ambiguous 0件を記録。missing は `editor.database_workbench` の2件で、現行の検証が機能のうち `editor.sql_analysis` だけを能力と突き合わせるための差であり、製品の診断ではないと明記）

- [構成検証の選択規則はBindingの形ごとに揃っているか](PLUGIN_SELECTION_RULE_SURVEY_2026-11-08/README.md)（internal/validation を走査し、6つのBinding形式すべてに対応なし・複数・能力非対応の診断があることを確認。ただし環境の機能だけは eature == environment.FeatureSQLAnalysis の分岐で対象が1つに限られ、editor.database_workbench は Known() の検査だけを受けて能力との突き合わせを受けない。docs/plugins.md の記述は6形式に当てはまり誤っていない）
- [環境の機能は、すべて選択を検査されるか](FEATURE_SELECTION_CHECK_ASYMMETRY_2026-11-08/README.md)（受理済みWorkspaceを3つ複製し、有効な機能だけを変えて実行。プラグインも能力も無い条件で、editor.sql_analysis は ENVIRONMENT_FEATURE_PLUGIN_MISSING で終了コード1、editor.database_workbench は診断0件で終了コード0。受理済みの複製も0件で終わり、この差が受理済みでは見えないことを示す。検査を足すべきかは示さない）
- [DSH外部HarnessからのTEI read-only CLI実起動](DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07/README.md)（#2509。外部AIハーネスの1実Sessionで固定実行物の実起動・実引数・実出力を同一ログで突合。H3・四Session比較・正答採点・Egressは未実施）
- [DSH実HarnessでのH2/H3四Session実行](DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md)（#2509・TAI #168。H2→H3→H3→H2を同一Taskで実行し、H3のSkill実Load byteを確認。回答採点・Skill効果の因果・Egressは未確認）




- [DSHセッションログ拡張の実送出計測](DSH_SESSION_LOG_EGRESS_MOCK_2026-11-07/README.md)（#2509・TAI #232。ローカルMock宛で拡張fieldの水位・再有効化・Tool結果の複製送出を計測。実Providerへの送出とEgress制御はNOT RUN）
- [オフラインGo Module供給の検証](TEI_GO_MODULE_SUPPLY_VERIFICATION_2026-11-07.md)（`cmd/tei`の実選択12件を`go.sum`の`h1`へ束縛してオフライン再解決。供給元・署名・Buildは未検証。SB-0b / #2831）
- [DSHセッションログ拡張の既定・再試行・切断の追測](DSH_SESSION_LOG_RETRY_AND_DEFAULT_2026-11-07/README.md)（#2515 H6-E2・#2509。出荷既定で拡張fieldが載り、非2xxと空streamでは水位が進まず未受理区間が再送されることを計測。実Provider送信とEgress制御はNOT RUN）

- [TEI自身の設計をCLIで扱う実測](TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md)（#3149。SHA-256 `bf6930a047cf94f276fb63b082bf71599986d71f1a5c53804105c04aa0021cfd` の固定実行物でread-only CLIを50出力ぶん実行し、成立した範囲と未対応を区別。意味の受理・実Provider・CI再現は未確認）
- [供給からの実BuildとSource閉包資格の食い違い](TEI_GO_BUILD_WITH_SUPPLY_X1_2026-11-07.md)（`cmd/tei`は供給だけでBuildできるが、同じadapterのSource閉包は`rejected`。artifactと資格を別Claimとして扱う。SB-B事前確認 / X1 / #2831）
- [Goのinternal規則とbuilder移動の影響範囲](TEI_GO_INTERNAL_RULE_AND_BUILDER_MOVE_2026-11-07.md)（SB-C0の案Aの前提がGoのinternal規則で拒否されることの実測と、移動で触れる29ファイルの内訳。実装は未着手。SB-C0 / #2831）
- [pwsh不安定時のWSL検証手順](WSL_VERIFICATION_RUNBOOK_2026-11-07.md)（worktreeの`.git`書換え・ログインシェル回避・読み取り専用cacheの後始末・両OS結果の扱い）
- [DSH外部Harnessのread境界の実測](DSH_H6E1_READ_BOUNDARY_2026-11-07/README.md)（#2515 H6-E1・#2509。read-onlyでもraw配下・.env・cwd外の合成canaryが読め、モデル入力とSession Logの両方へ載ったことを記録。実OS/ACLの拒否試験はNOT RUN）
- [DSH外部Harnessが暗黙に取り込むContextの実測](DSH_H6E3_IMPLICIT_CONTEXT_2026-11-07/README.md)（#2515 H6-E3・#2509。AGENTS.md・CLAUDE.md・Skillのdescriptionが、Toolで読まずにモデル入力とSession Logへ載ることを合成canaryで観測。Skill本文は未ロード。Tool poison・実OS拒否はNOT RUN）
- [二回の独立Buildにおける再現性の観測](TEI_GO_BUILD_REPRODUCIBILITY_2026-11-07.md)（`cmd/tei`を別々の試行ID・別の隔離stateで二回Buildし、同じartifact digestを得た記録。Builder信頼・受入・権限は未確認。SB-N8 / #2831）

- [DSH外部HarnessのHost棚卸し](DSH_H6E0_HOST_INVENTORY_2026-11-07/README.md)（#2515 H6-E0。Tool ProcessからOS・mount・環境変数・loopback到達性を列挙し、read-onlyのConstrainedLanguageと危険modeの差、親の環境変数が子へ継承されないことを観測。別Host・実Egress制御はNOT RUN）
- [DSH外部HarnessのLane分離](DSH_H6E4_LANE_SEPARATION_2026-11-07/README.md)（#2515 H6-E4・#2509。ディレクトリ分離だけの2 Laneで、各Laneが相手Laneの合成canaryをcompletedで読み、モデル入力とSession Logへ載ったことを記録。worktreeだけでは分離にならない。別Identity/ACL/mount/egressの実拒否はNOT RUN）
- [DSH `workspace-write` の書込み・読取り非対称](DSH_WORKSPACE_WRITE_ASYMMETRY_2026-11-07/README.md)（#2515 P0-a・H6-E1。宣言root外への書込みは`UnauthorizedAccessException`で拒否されファイルも作られない一方、同じroot外のファイルは絶対・相対pathとも読めた。`DSH_PERMISSION_MODE`は空、`LanguageMode`はFullLanguage。読取り拒否の実OS構成はNOT RUN）
- [フックの宣言外呼出し拒否の確認](XTEA_LOWCODE_HOOK_DECLARED_CALLS_2026-11.md)（参照実装の試験記録。Command境界への接続は未実装）
- [外部Moduleを含む選択Sourceの観測](TEI_GO_SOURCE_CLOSURE_OBSERVATION_2026-11-07.md)（`cmd/tei`の選択入力をリポジトリ内318件と外部Module 12件・325メンバーに分けて列挙し、資格は常に`false`として閉包方針を変えない記録。SB-A1 / #2831）

- [DSH外部Harnessの再開と既送出Session Log](DSH_H6E7_RESUME_AFTER_DISCLOSURE_2026-11-07/README.md)（#2515 H6-E7・#2509。Raw canaryを読んだSessionを再開すると、Session Logは水位`27`から進み再送されない一方、モデル入力には過去の会話が再構成されcanaryが再び載る。Grant失効・ACK不明replayはNOT RUN）
- [DSH外部Harnessが読んだ内容の保存先棚卸し](DSH_H6E6_SINK_INVENTORY_2026-11-07/README.md)（#2515 H6-E6・#2509。1 Sessionで読んだ合成canaryが、headless stdoutと永続Session Logの2つに残り、`storages`・`cache`・`attachments`には残らなかった。永続Session Logは9連結zstd frameで、単一frame展開では見落とす。Issue/CI/外部ProviderのsinkはNOT RUN）
- [DSH外部Harness上での生成候補の逸脱試行](DSH_H6E5_CANDIDATE_ESCAPE_2026-11-07/README.md)（#2515 H6-E5・#2509。合成候補から、非許可の隣接Sourceは読め、子Processは起動でき、自己書換えはConstrainedLanguageの初期化失敗で止まった。自己書換えが書込みPolicyで拒否された証拠ではない。workspace-writeでの再試行・D1→D2・Selector同一Source証明はNOT RUN）
- [子Processは親Toolの到達範囲を引き継ぐ](DSH_CHILD_PROCESS_REACH_2026-11-08/README.md)（#2515 受入条件3・H6-E1・#2509。親`pwsh`が起動した子`cmd.exe`は、`read-only`でも宣言root外を読み、書込みは`read-only`で止まり`danger-full-access`で通った。子Processを境界にできず、実効保証は最弱の経路を基準に下げる必要がある。別OS資格・別Userでの子起動、Network到達性はNOT RUN）

- [外部Go Module入力の明示opt-in受入](TEI_GO_EXTERNAL_MODULE_ADMISSION_OPTIN_2026-11-07.md)（既定の拒否方針は不変のまま、供給を明示した呼び出しだけが外部Moduleメンバーを含む閉包を受ける参照実装。受入記録に方針・根拠・拒否理由を残す。SB-B / 案C / #2831）
- [起動主体と被起動主体を分けたBuild駆動の記録](TEI_GO_BUILD_STAGE_DRIVER_2026-11-07.md)（`cmd/tei`を子プロセスとして駆動し、invoker/invokedの実行ファイルdigest・argv・exitと、子の主張とディスク実物のartifact digestが一致することを記録。L3・受入・権限は未確認。SB-C0/C1 / #2831）
- [セルフビルドP0の証拠台帳](TEI_SELF_BUILD_P0_EVIDENCE_LEDGER_2026-11-08.md)（SB-0a〜SB-C1の統合済みEvidenceをClaim別に並べ、L1で成立したもの・未成立のもの・L3までの残差を一か所にした索引。本稿は新規の主張ではなく、各項目の再実行もしていない。SB-P0 / #2831）
- [SB-N13の自称invoker照合](TEI_GO_STAGE_DRIVER_INVOKER_CLAIM_2026-11-08.md)（駆動主体が自称するdigestを実行中process imageと照合する欄をstage driverへ追加。一致しても`l3Claimed`はfalseのまま。独立した失敗は連結して保持する。SB-C0/C1 / #2831）

- [stage1の生成物がstage2を駆動する（L3）](TEI_GO_SELF_BUILD_TWO_STAGE_L3_2026-11-07.md)（stage1のartifactをstage2のinvokerとして実起動し、invoker一致・同一レシピ・artifact一致が揃うことを記録。Seed Trust・受入・権限・別OS一般化は未確認。SB-C1 / #2831）

- [TEI自身の設計をCLIで扱う実測 第2ラウンド](TEI_DOGFOODING_ROUND2_2026-11-08/README.md)（#3149。現行main `4fd88c2b1` の単一実行物（SHA-256 `16c9090d…`）で再実測し、第1ラウンドの10課題のうち7件が解消、D-3・D-9・D-10 が残ることを記録。MCP入口は未実装のため CLI 経路のみ）
- [値域付き型宣言のanchor走査](SYNTAX_TYPE_VALUES_ANCHOR_2026-11-08.md)（`type ... values { ... }` が Source Index と validate を失敗させていた記録。anchor 走査の読み飛ばしのみで、値集合の意味検証は未確認）


- [TEIのread-only MCP AdapterによるCLI/MCP一致の実測](TEI_DOGFOODING_MCP_ROUND_2026-11-08/README.md)（#3400・#3149。参照実装のAdapterで `model catalog`・`inspect`・`trace`・`validate` をMCPから呼び、CLIと同じ値（`worldFingerprint`・`snapshotFingerprint`・`schemaVersion`）になることを確認。製品のMCP契約・実Host・認可・外部送出はNOT RUN）

- [knowledge-contextの名前空間kind名](KNOWLEDGE_CONTEXT_NAMESPACE_KIND_2026-11-08.md)（catalogの`semantic`を渡すと`semanticElement`無しの成功になっていた。`semantic`と`semantic-namespace`の両方を受け付け、未知kindは解決しない。実Provider・他kindの全件突合は未確認）
- [二回の独立した駆動試行でも同じstage2成果物になること](TEI_GO_SELF_BUILD_TWO_STAGE_REPEAT_2026-11-08.md)（`-work`を分けた2回の実行で、両方L3に到達しstage1/stage2のartifact digestが一致することを記録。Seed Trust・受入・権限・OS間一致は未確認。SB-C1 / #2831）

