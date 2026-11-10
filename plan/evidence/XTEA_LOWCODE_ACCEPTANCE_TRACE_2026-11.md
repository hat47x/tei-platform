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
| AC-2 | 一部 | フック宣言の検証、`internal/hook`の実行契約、`hook.Declared`（宣言した能力以外の呼出しを拒否。[確認記録](XTEA_LOWCODE_HOOK_DECLARED_CALLS_2026-11.md)）、`hook.CommandCaller`（フックの呼出しをCommand境界へ委譲）、`hook.Runner.Caller`（宣言に制限した呼出し元を実装へ渡す配線）、`hook.ReferenceRule`（宣言した読み取りだけを評価する規則Pluginの参照実装）、`hook.Guarded`（判定が通ったときだけ宣言済みの能力を呼ぶ連結）、`hook.CommandPatternFor`（能力を宣言する`command` Patternを1件だけ解決し、0件・複数件を拒否。[確認記録](XTEA_LOWCODE_HOOK_RUNNER_OUTCOME_2026-11.md)）、印刷面（`-format print`）も同じフック宣言を表示（[確認記録](XTEA_LOWCODE_HOOK_PRINT_SURFACE_2026-11.md)）、`hook.RunSequence`が位相を固定順で実行し途中停止位置を返す（[確認記録](XTEA_LOWCODE_HOOK_PHASE_SEQUENCE_2026-11.md)）、`hook.ReadsFor`が宣言した読み取りだけを実装へ渡す（未観測・宣言外を拒否。[確認記録](XTEA_LOWCODE_HOOK_DECLARED_READS_2026-11.md)）、`RunSequence`は位相ごとに宣言した読み取りだけを渡す（[確認記録](XTEA_LOWCODE_HOOK_PER_PHASE_READS_2026-11.md)）、`hook.ProducesFor`が宣言した`produces`だけを提案として返す（[確認記録](XTEA_LOWCODE_HOOK_DECLARED_PRODUCES_2026-11.md)）、機能定義書（`function-definition`）に`hook`行を位相順で出す（[確認記録](XTEA_LOWCODE_HOOK_FUNCTION_DEFINITION_2026-11.md)）、`view -command -execute`が宣言フックを呼び、未評価・不成立・到達しない位相を区別して報告する（[確認記録](XTEA_LOWCODE_HOOK_CLI_WIRING_2026-11.md)）、Studioの参照実行も同じ境界でフックを呼ぶ（[確認記録](XTEA_LOWCODE_HOOK_STUDIO_WIRING_2026-11.md)）、`hook.RunReachable`が到達できる位相だけを`PhaseOrder`の順に実行し、到達しない位相と前提が欠けた位相を別々に記録する（[確認記録](XTEA_LOWCODE_HOOK_REACHABLE_PHASES_2026-11.md)）、フックが使う能力を宣言する`command` Patternの不在を検証で拒否（`APPLICATION_PATTERN_INTERACTION_HOOK_USE_WITHOUT_COMMAND`）、`hook.RunReachable`が`RunWithDeclaredValues`を通るため、宣言した`produces`が`PhaseReport.Produced`に残り、宣言外の提案は`HOOK_PRODUCE_UNDECLARED`で拒否される（[確認記録](XTEA_LOWCODE_HOOK_REACHABLE_PROPOSES_2026-11.md)）、提案はCLIのテキストとStudioの画面にも候補として出る（[確認記録](XTEA_LOWCODE_HOOK_SHOW_PROPOSALS_2026-11.md)）、`hook.RunReachable`は`uses`を宣言した位相を「実行済み」と報告せず、呼ばれなかった能力を`Uncalled`に名指す（[確認記録](XTEA_LOWCODE_HOOK_DECLARED_CALL_HONESTY_2026-11.md)）、`hook.RunDeclaredCalls`が宣言した能力を宣言制限済みの呼出し元を通して実際に呼び、CLIとStudioの参照実行が`CommandCaller`を配線する（[確認記録](XTEA_LOWCODE_HOOK_DECLARED_CALL_WIRING_2026-11.md)）、`hook.BindInputsFromReads`が、interactionの`command`の`input`が述べる対応で、フックの`reads`を能力のパラメータへ結ぶ（読んでいない属性は埋めない。[確認記録](XTEA_LOWCODE_HOOK_READS_TO_PARAMS_2026-11.md)）、結び付けた属性型とパラメータ型が原始型同士で食い違えば`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_TYPE_MISMATCH`で拒否する（[確認記録](XTEA_LOWCODE_HOOK_BINDING_TYPE_CHECK_2026-11.md)）、パラメータ型がDataを名指す場合は別Dataの属性を結ぶ宣言を`APPLICATION_PATTERN_INTERACTION_COMMAND_INPUT_DATA_MISMATCH`で拒否し、1つの属性を複数のパラメータへ束ねる形は受理する（[確認記録](XTEA_LOWCODE_HOOK_BINDING_DATA_MATCH_2026-11.md)）、`hook.Runner.ByPhase`が位相ごとに別の実装を供給でき、供給の無い位相は共通実装へ落ちるか`HOOK_IMPLEMENTATION_UNAVAILABLE`で止まる（[確認記録](XTEA_LOWCODE_HOOK_PHASE_IMPLEMENTATION_2026-11.md)）、`hook.RunRows`が行単位のフックを行ごとに適用し、各行の値は宣言した`reads`だけに絞り、拒否した行を名指して以降を訪問しない（`declaredHook`は`browse`を受理する。[確認記録](XTEA_LOWCODE_HOOK_ROW_REACHABILITY_2026-11.md)）、`hook.RunRowCalls`が行単位フックの`uses`を1行ごとにその行の値で果たし、解決できない呼出しは行番号を名指して止まる（[確認記録](XTEA_LOWCODE_HOOK_ROW_USES_CALLS_2026-11.md)）、Data型パラメータへ結べるのは同一Dataの属性で、識別子に限らず、実行時はその属性の値がそのまま渡る（規約として識別子へ結ぶことを記録。[確認記録](XTEA_LOWCODE_DATA_PARAM_CONVENTION_2026-11.md)）、受理される原始型の結び付けは両側が同じ下位原始型に解決されるため、実行時の値変換を要さず、変換が必要になる異なる原始型の組は検証が拒否する（[確認記録](XTEA_LOWCODE_VALUE_CONVERSION_2026-11.md)）、`hook.Runner.RunBeforeChange`→変更の適用→`hook.Runner.RunAfterChange`と`hook.AppliedChangeReach`の順で`after-*`のフックを実行できる（[確認記録](XTEA_LOWCODE_HOOK_AFTER_CHANGE_2026-11.md)）、`view -execute -reference-store`が効果をメモリ上の参照Storeへ適用し、変更の前後でフックを二巡して`after-*`を実行する（[確認記録](XTEA_LOWCODE_HOOK_AFTER_CHANGE_CLI_2026-11.md)）、変更後の位相が拒否した場合は適用した効果をStoreのスナップショットから戻し、応答に`effectReverted`を載せる（既定の参照規則は拒否しないため、この経路は拒否する規則の供給で到達する。[確認記録](XTEA_LOWCODE_HOOK_AFTER_CHANGE_REVERT_2026-11.md)）、Studioは`ExecuteCommandWithReferenceStore`が効果をメモリ上の参照Storeへ適用して変更後のフックを実行し、併合は`hook.PhaseReport.Merge`を両ホストが共有する（[確認記録](XTEA_LOWCODE_HOOK_AFTER_CHANGE_STUDIO_2026-11.md)）、フックの`uses`呼出しは参照Executorを通りデータを適用しないため、`after-*`拒否時に戻すのはコマンド本体の効果だけで足りる（[確認記録](XTEA_LOWCODE_HOOK_USES_COMPENSATION_2026-11.md)） | フック宣言・実装契約の単体試験、宣言外呼出しの拒否試験、Command境界への委譲試験、Runnerからの配線と宣言外拒否の試験、参照規則の決定性・未観測拒否の試験、同位相の重複宣言の拒否試験、判定と能力呼出しの連結の試験、能力からPatternの解決（0件・複数件の拒否）の試験、画面と印刷の両方で同じフック宣言が出る試験、能力を宣言する`command` Patternが無いフックの拒否試験、位相の実行順と停止位置の試験、宣言した読み取りだけを渡す試験、宣言外の提案の拒否試験、機能定義書へのフック行の試験、CLI参照実行からのフック呼出しの試験、到達範囲ごとの位相選択の試験、参照実行の経路での提案の保持と宣言外拒否の試験、提案の表示の試験、宣言した呼出しの未実行を名指す試験、宣言した能力の実呼出しの試験、`reads`とパラメータの対応の試験、結び付けの型検査の試験、Data対応と束ねの試験、位相別実装の選択の試験、行単位適用の試験、行単位`uses`の実行の試験、Data型パラメータの結び付けの試験、受理・拒否される型の組の試験、`after-*`の実行経路の試験、`after-*`のCLI配線の試験、`after-*`拒否時の取り消しの試験、Studioの`after-*`配線の試験、`uses`呼出しの適用範囲の確認 | 異なる原始型の結び付けを許すか（意味の設計判断）、識別子限定の検査、複合キーを1つのData型パラメータへ渡す表現、規則Pluginの契約と束縛、`reads`・`produces`の実データ観測、StudioのUIからの`after-*`経路の呼出し、Studio経路での取り消しの表明、フックの`uses`呼出しを参照Storeへ適用するか（意味の設計判断） |
| AC-3 | 一部 | `role`・`required`・`immutable`・`values`、表示ラベル | 検証・ビュー・参照Executorの試験 | 幅・行数・整列・部品の表示指定 |
| AC-4 | 一部 | `layout`の順序投影 | layoutの検証・ビュー | 座標を含む配置とプロファイル比較 |
| AC-5 | 一部 | `Outcome`・`ExecutionError`・期間拒否、`ReferenceFailure`の符号（`record-missing`・`record-exists`・`value-rejected`・`rule-rejected`）と`FailureOutcome`、CLIの`COMMAND_REFUSED`、フックの`hook.NotSatisfied`（業務上の不成立を実行の失敗と区別。[確認記録](XTEA_LOWCODE_HOOK_RUNNER_OUTCOME_2026-11.md)）、`hook.NotSatisfiedOutcomes`がフックの結末名を列挙し、`hook.CommandOutcomeFor`がCommand側の拒否名へ写す（写しは`command.FailureOutcome`を呼ぶだけで、対応の無い`not-satisfied`は写さない。[確認記録](XTEA_LOWCODE_HOOK_OUTCOME_VOCABULARY_2026-11.md)）、`hook.ValidateOutcome`が語彙の外の結末名を`HOOK_OUTCOME_UNDECLARED`で拒否する（[確認記録](XTEA_LOWCODE_HOOK_OUTCOME_CHECK_2026-11.md)）、参照Executorは結末を2つ以上宣言した能力で結末を返さず、`outcomeUndetermined`に理由を残す（選択規則は未決定。[確認記録](XTEA_LOWCODE_MULTI_OUTCOME_2026-11.md)）。適用・不存在・競合・拒否・不明・未実行の6つは区別できるが、**多重一致は参照Storeがキーで一意化するため表現できず、符号を発明していない**（[読取り記録](XTEA_LOWCODE_MULTI_MATCH_GAP_2026-11.md)）、フックの結末名がCommand側の名前を持つ場合は`commandOutcome`としてJSONとテキストに出し、対応の無い`not-satisfied`はCommandの拒否理由として出さない（[確認記録](XTEA_LOWCODE_OUTCOME_CORRESPONDENCE_USE_2026-11.md)） | Command実行境界・期間の試験、参照Storeの拒否符号の試験、拒否と実行障害の区別の試験、不成立と`warn`・実行失敗の区別の試験、結末名の語彙とCommand拒否名の対応の試験、語彙外の結末名の拒否試験、複数結末の未確定の試験、結末名対応の出力の試験 | 複数結末から一つを選ぶ規則、多重一致の表現、実Providerでの結末、プロセスを跨ぐ経路での検査|
| AC-6 | 一部 | 値域・必須・候補キー・削除ポリシー、`COMMAND_DATASOURCE_READ_ONLY`（Applicationの読み取り専用datasource宣言を実行前に検査。[確認記録](XTEA_LOWCODE_READONLY_DATASOURCE_2026-11.md)） | 参照Storeの単体・例の試験、読み取り専用宣言の拒否・読み書き宣言の許容・宣言なしを推測しない試験 | データソース側の実行時強制（三段目） |
| AC-7 | 一部 | `-format html`・`print`、印刷テンプレート、`tei-semantic-fingerprint`（画面・帳票が読んだ意味の版） | HTML・printの単体試験、record・header-detail・browseの印刷面が版を持つ試験 | 表示（座標・用紙・部品）の版照合、manifestの版 |
| AC-8 | 一部 | Executorが返すEffectと宣言の照合、`Invoker.Recorder`、CLIの`-operation-log`、Studioの`records`、参照Storeの`Applied`照合と`Differences`（宣言外の種別は拒否、宣言と異なる対象への同種別適用は未解決の差として保持。CLIのテキスト・JSONとStudioの画面にも表示。操作を列挙しないExecutorは`AppliedUnobserved`で「未確認」と示す。[確認記録](XTEA_LOWCODE_OPERATION_DIFFERENCE_2026-11.md)） | 効果の不一致試験、記録の試験、画面の件数表示、適用操作の照合試験、カスケードを差として残す試験、実行結果の差の表示試験 | 実行時に発行されたSQL文・CRUD操作そのものの記録 |
| AC-9 | 一部 | 検証レポートの`worldFingerprint`・`sourceIndexFingerprint`、`validation.Proposal`と`validation.Accept`、`tei validate -json`の`proposals`、Studioの`ValidationResponse.proposals`（[確認記録](XTEA_LOWCODE_DIAGNOSTIC_PROPOSAL_2026-11.md)、[Studio表示の確認](XTEA_LOWCODE_STUDIO_PROPOSALS_2026-11.md)） | 検証の版照合、修正案が版に結び付くこと、版不一致・版欠落・二重受理を拒否する試験 | 作成者が修正案を採る操作の画面・手順 |
| AC-10 | 一部 | `effectcommit`のreceipt、Commandアダプタ、CLIの`COMMAND_EFFECT_UNKNOWN`と`COMMAND_LIFECYCLE_REFUSED` | アダプタの単体試験、失敗の符号化試験、実PostgreSQLはDSN設定時のみ | Command再実行での「不明」状態の永続化と自動再実行の禁止 |
| AC-11 | 一部 | `effectcommit.Runner`の認可境界、`command.Invoker.Authorizer`、`AllowListAuthorizer`、CLI/StudioのAllowList、`legacyauthorization/commandbinding.CommandAuthorizer`（subject・action・resourceを既存規則へ接続）、`command.GrantAuthorizer`（能力が宣言した許可と発行済み許可の照合。部分許可・無宣言・不正宣言を拒否）。確認記録: [既存認可規則](XTEA_LOWCODE_COMMAND_AUTHORIZATION_2026-11.md)、[宣言許可の照合](XTEA_LOWCODE_GRANT_AUTHORIZATION_2026-11.md) | 認可付きPostgreSQL試験（DSN設定時のみ）、Authorizerの許可・拒否試験、CLIの`-allow-capability`、Studioの`ExecuteCommandWithPolicy`、Patternが宣言していても付与が無ければExecutorへ到達しない試験、能力の宣言許可と発行許可の照合試験 | 権限の正本・付与の管理、`security.authorize`への実運用接続 |
| AC-12 | 未実装 | 設計のみ（[作者環境の受容設計](../design/XTEA_AUTHORING_SURFACE_RECEPTION_2026-11.md)） | — | 面ごとの配置検査、意味の役割と表示指定の分離、生成物の未受理扱い | Package、Semantic/Applicationの宣言 | Package・Applicationの検証と例 | Core外の型定義との組合せ |
| RQ-C1 | 一部 | 標準Patternの再利用、Packageの検証・ビュー、RQ-C1の確認方法（[確認記録](XTEA_LOWCODE_RQC1_VERIFICATION_2026-11.md)）、契約欄（適合試験名・投影）の検査（[確認記録](XTEA_LOWCODE_PACKAGE_CONFORMANCE_2026-11.md)） | 単体試験 | 適合試験そのものの実行 |
| RQ-C2 | 一部 | フック宣言、CommandとObservationの分離、RQ-C2の確認方法（[確認記録](XTEA_LOWCODE_RQC2_VERIFICATION_2026-11.md)）、フック宣言の範囲`scope`（[確認記録](XTEA_LOWCODE_HOOK_SCOPE_2026-11.md)・[実行時の適用](XTEA_LOWCODE_HOOK_SCOPE_RUN_2026-11.md)・[行範囲の適用](XTEA_LOWCODE_HOOK_ROW_SCOPE_2026-11.md)・[到達可能な実行への配線](XTEA_LOWCODE_HOOK_ROW_SCOPE_WIRING_2026-11.md)・[宣言の修正](XTEA_LOWCODE_HOOK_ROW_SCOPE_DECLARABLE_2026-11.md)・[貫通試験](XTEA_LOWCODE_HOOK_SCOPE_THROUGH_2026-11.md)） | 単体試験 | Plugin境界での完全な宣言強制、規則Pluginの供給路 |
| RQ-C1 | 一部 | 標準Patternの再利用、Packageの検証・ビュー、RQ-C1の確認方法（[確認記録](XTEA_LOWCODE_RQC1_VERIFICATION_2026-11.md)） | 単体試験 | パッケージごとの独立した適合試験 |
| RQ-C2 | 一部 | フック宣言、CommandとObservationの分離、RQ-C2の確認方法（[確認記録](XTEA_LOWCODE_RQC2_VERIFICATION_2026-11.md)）、フック宣言の範囲`scope`（[確認記録](XTEA_LOWCODE_HOOK_SCOPE_2026-11.md)・[実行時の適用](XTEA_LOWCODE_HOOK_SCOPE_RUN_2026-11.md)・[行範囲の適用](XTEA_LOWCODE_HOOK_ROW_SCOPE_2026-11.md)・[到達可能な実行への配線](XTEA_LOWCODE_HOOK_ROW_SCOPE_WIRING_2026-11.md)） | 単体試験 | Plugin境界での完全な宣言強制、規則Pluginの供給路 |
| RQ-C3 | 一部 | 期間、値域、型正規化、`calendar`宣言と期間との結合（[確認記録](XTEA_LOWCODE_CALENDAR_DECLARATION_2026-11.md)）、`internal/businesscalendar`（暦からの日付差・営業日加算・会計年度・期間キー。[確認記録](XTEA_LOWCODE_CALENDAR_CALCULATION_2026-11.md)、[期間キーの確認](XTEA_LOWCODE_PERIOD_KEY_2026-11.md)）、`internal/taxrate`（適用期間の両端・端数処理の宣言・重なりの検出。Data宣言の属性役割からの行の結合。[確認記録](XTEA_LOWCODE_TAX_RATE_2026-11.md)、[Data結合の確認](XTEA_LOWCODE_TAX_RATE_BINDING_2026-11.md)）、`internal/currencyrate`（同上。年次フォールバックの宣言。[確認記録](XTEA_LOWCODE_CURRENCY_RATE_2026-11.md)、[フォールバックの確認](XTEA_LOWCODE_CURRENCY_FALLBACK_2026-11.md)、[宣言語彙の確認](XTEA_LOWCODE_CURRENCY_FALLBACK_ROLE_2026-11.md)）、`internal/numbering`（同上。Data宣言の属性役割からの行の結合。[確認記録](XTEA_LOWCODE_NUMBERING_2026-11.md)、[Data結合の確認](XTEA_LOWCODE_NUMBERING_BINDING_2026-11.md)、[発行・取消の取引結合](XTEA_LOWCODE_NUMBERING_TRANSACTION_2026-11.md)、[取引境界との結合](XTEA_LOWCODE_NUMBERING_RESERVATION_TRANSACTION_2026-11.md)、[複数規則の行識別](XTEA_LOWCODE_NUMBERING_RULE_IDENTITY_2026-11.md)、[規則名から発行までの経路](XTEA_LOWCODE_NUMBERING_RULE_RESERVE_2026-11.md)、[Command取引境界とのアダプタ](XTEA_LOWCODE_NUMBERING_COMMAND_ADAPTER_2026-11.md)、[月内締めの期間化](XTEA_LOWCODE_MONTHLY_CLOSING_PERIOD_2026-11.md)、[区分ごとの閉め](XTEA_LOWCODE_CLOSING_SEGMENT_SCOPE_2026-11.md)、[CLI観察と状態源](XTEA_LOWCODE_CLOSING_SCOPE_CLI_2026-11.md)、[期間の閉め操作](XTEA_LOWCODE_PERIOD_CLOSE_OPERATION_2026-11.md)、[利用者設定](XTEA_LOWCODE_USER_SETTINGS_2026-11.md)、[設定の書込み](XTEA_LOWCODE_USER_SETTINGS_WRITE_2026-11.md)、[例の締め日と設定Data](XTEA_LOWCODE_EXAMPLE_CLOSING_SETTINGS_2026-11.md)、[為替の2表構成](XTEA_LOWCODE_CURRENCY_TWO_TABLES_2026-11.md)、[採番の永続格納](XTEA_LOWCODE_NUMBERING_SQLITE_STORE_2026-11.md)、[2プロセス同時実行](XTEA_LOWCODE_NUMBERING_PROCESS_RACE_2026-11.md)）、`holiday-date`役割からの休日行の結合（[確認記録](XTEA_LOWCODE_HOLIDAY_BINDING_2026-11.md)）、`holiday-calendar`役割による複数暦の分離（[確認記録](XTEA_LOWCODE_MULTI_CALENDAR_HOLIDAY_2026-11.md)）、暦識別の型値域による検査（[確認記録](XTEA_LOWCODE_CALENDAR_DOMAIN_2026-11.md)）、期間→暦→休日の一続きの解決（[確認記録](XTEA_LOWCODE_HOLIDAY_RESOLVE_2026-11.md)）、標準業務の属性役割17個の受理と綴り誤りの拒否（[確認記録](XTEA_LOWCODE_ROLE_VOCABULARY_2026-11.md)）、役割が要求するプリミティブ型との一致検査（[確認記録](XTEA_LOWCODE_ROLE_TYPE_CHECK_2026-11.md)）、例のWorkspaceでの宣言と結合（[確認記録](XTEA_LOWCODE_EXAMPLE_STANDARD_DATA_2026-11.md)）、標準業務Dataの投影（[確認記録](XTEA_LOWCODE_STANDARD_BUSINESS_PROJECTION_2026-11.md)）、`ApplicationView.periods`の`calendar`（期間が従う暦を投影。Studioの「期間」表にも列を追加）、RQ-C3の確認方法（[確認記録](XTEA_LOWCODE_RQC3_VERIFICATION_2026-11.md)）、期間の`close-requires`の実行時検査（[確認記録](XTEA_LOWCODE_CLOSE_REQUIRES_2026-11.md)） | 参照Store・例の試験、暦宣言の構文・検証試験、日付導出の試験（13件）、期間要件の試験 | 税・通貨・採番の適用期間と評価時点、期間キーの導出、`correction`の実行時検査 |

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



















