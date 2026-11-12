# CLI読取りコマンドのJSON出力契約（schemaVersionと失敗時の形式）の現状調査と追加案

- **状態**: 調査と案（提案中、未採用）。一覧は2026-10-31時点のorigin/main `cd47b6fa4` のコードと、WSL go1.26.9でbuildした`tei`による実行結果の記録である。
- **担当課題**: [#2354](https://github.com/hat47x/tei-platform/issues/2354)。
- **関連文書**: [CLIを主入口とするインターフェース設計](CLI_FIRST_INTERFACE_DESIGN.md)、[inspect・traceの開示境界](CLI_C2_DISCLOSURE_BOUNDARY_DESIGN.md)、[全体構成](../../docs/architecture.md)、[開発手順](../../docs/getting-started.md)。

## 1. 役割と正本との関係

この文書の役割は、`tei commands`が広告する読取り専用コマンドの出力形式について、(1)現状の一覧、(2)`tei.cli.*/v0`の中で互換とみなせる変更と版を上げる変更の区別の規則案、(3)schemaVersionを既存の欄を変えずに足す範囲の案、(4)失敗時の形式の決定案、を一か所にまとめることである。

出力形式の正本は実装（`cmd/tei`）と、実装と表が一致することを確認する試験である。この文書は調査結果と案を記録するもので、契約そのものの正本にはならない。ここで決めない判断（外部契約にする対象の確定、v0の維持かv1への移行か）は、Issueに残したままOwnerの判断を待つ。

## 2. 対象範囲

`tei commands`が広告する読取り専用コマンド（2026-11-07時点で19件。`cmd/tei/cli_contract.go`）のうち、JSONを出すものを扱う。件数はこの文書に固定せず、`tei commands`の出力と`cmd/tei/cli_contract_test.go`の一致を正本とする。書込み系（deployment-*など）と、`commands`自身の引数扱いは、失敗時の形式の一部としてだけ触れる。`scenario`や`model`のサブコマンド以外の広告外コマンドは扱わない。

## 3. 現在の結論（先に要約）

- `world`を除く、`tei commands`が広告する読取り専用JSONコマンドは`tei.cli.<command>/v0`を持ち、失敗時は`ok: false`と`error.code`を標準出力のJSON 1件で返す。終了コードの意味（0/1/2）は従来どおり。
- **`ok`/`result`を持つ成功時の形は、`version -json`には当てはまらない。** `version -json`は`ok`を持たず、build識別（`goVersion`等）を`schemaVersion`と同じ階層に並べて返す。2026-11-07から`tei commands`は`version`の`successShape`を`identity`として広告するので、呼び手は成功時に`ok`を仮定しない。`jsonContract`の`enveloped`は失敗時の形（`schemaVersion`と`error.code`）を指し、成功時の本体は`successShape`が示す（#2824）。
- `world`は例外で、`tei.cli.*`のエンベロープを持たない。`tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs`が`tei world`の生標準出力SHA-256を受理済みDecisionの`derivedWorldSha256`と照合するため、インデントやエンベロープの追加は証跡照合と両立しない。`world`は内部・証跡用途であり、外部の安定JSON契約の対象外とする（2026-11-02 Owner判断）。
- 実測では、製品CLI（`cmd/tei`）が外部のGo moduleに依存しないのと同様に、**広告された読取りコマンドのうち`tei.cli.*/v0`を持たない出力は、内部構造体の変更がそのまま外部向け変更になる**。`world`はこの例外として扱い、その他のコマンドは投影型で版固定する。

## 4. 現状の一覧（実測）

実行環境: WSL go1.26.9で`go build ./cmd/tei`、参照作業領域`reference/workspace`。失敗時の例は存在しない作業領域、存在しない対象、未対応のflagを与えた場合である。終了コードは成功0、意味的な見つからない1、引数と読込みの失敗2が基本である。

| コマンド | schemaVersion | 成功時の出力 | 失敗時の標準出力 | 内部構造体の直接出力 |
| --- | --- | --- | --- | --- |
| `version -json` | `tei.cli.version/v0` | JSON | 引数不正は標準エラー | いいえ（投影型） |
| `commands` | `tei.cli.discovery/v0` | JSON | 引数不正は標準エラー | いいえ（投影型） |
| `model catalog` | `tei.cli.model/v0` | JSON（`declarationCounts`を含み、各`entries`要素は標準意味型を`prelude:true`で示す） | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`、`NAMESPACE_NOT_FOUND`） | いいえ（投影型。内側に`tei.operations.model-catalog`の版を持つ） |
| `model overview` | `tei.cli.model-overview/v0` | JSON（2026-11-10追加。`sections`の各要素に`descendants`・`kindCounts`・`attention`。`-limit`で切った後も`totals`は全区画の合計。`unattributed`は区画に属さないfinding） | JSON（終了コード2: `INVALID_ARGUMENT`・`WORKSPACE_LOAD_FAILED`・`SOURCE_INDEX_FAILED`・`MODEL_OVERVIEW_FAILED`） | いいえ（投影型。内側に`tei.operations.model-overview`の版を持つ） |
| `model locate` | `tei.cli.model-locate/v0` | JSON（2026-11-10追加。`path`・`children`・`siblings`・`outgoing`・`incoming`・`coverage`、同日に配下の要注意件数`attention`を追加。一覧は`-limit`で切り`total`・`omitted`を持つ） | JSON（終了コード2: `INVALID_ARGUMENT`・`WORKSPACE_LOAD_FAILED`・`SOURCE_INDEX_FAILED`・`MODEL_LOCATE_FAILED`。終了コード1: `SUBJECT_NOT_FOUND`・`SUBJECT_AMBIGUOUS`） | いいえ（投影型。内側に`tei.operations.model-location`の版を持つ） |
| `model document-draft` | `tei.cli.model-document-draft/v0` | JSON（2026-11-11追加。受理済みモデルを読む派生Projection。`-document-kind element-explanation`（既定）は一要素＋一関係から `model`・`subjects`・`relations`・`occurrences`・`authorityLimits` を返す。`-document-kind crud-table` は `-application` を取り、`scope`（`application`・`crudRule`）とCRUD行列（`table` Occurrenceの`rowSubjects`で各行の元Identityを保持）を返す。`-document-kind specification` は `-section <kind>:<id>[@<relation-id>]`（repeatable・章順）を取り、章だての`occurrences`と`scope.skippedSubjects`（宣言されていないSubjectと理由）を返す。`-document-kind three-element-diagram` は `-model <path>`（三要素分析モデルJSON）を取り、`diagram` Occurrence（nodes/edges/cruds/diagnostics/unresolvedEdges）を返す。**この種類はWorkspaceを読まない**ので `-workspace` は不要で、`model.workspace` も記録されない。`-frozen` では `frozen`・`frozenRevision` を追加する。Source書換え・意味受理は行わない） | JSON（終了コード2: `INVALID_ARGUMENT`・`WORKSPACE_LOAD_FAILED`・`SOURCE_INDEX_FAILED`。終了コード1: `SUBJECT_NOT_FOUND`・`SUBJECT_AMBIGUOUS`・`UNSUPPORTED_SUBJECT_KIND`・`UNSUPPORTED_DOCUMENT_KIND`・`RELATION_NOT_FOUND`・`APPLICATION_NOT_FOUND`・`DOCUMENT_DRAFT_FAILED`） | いいえ（投影型。内側に`tei.document-draft/v0`を持つ） |
| `model document-reconcile` | `tei.cli.model-document-reconcile/v0` | JSON（2026-11-11追加。保存したDraftと現在のWorkspaceの読取り専用の比較。`modelReference`（`current`/`stale`と動いたdigest。`frozen` Draftでpinが動いた場合は `frozenRevision` diff、三要素分析図では `modelFingerprint` diffを追加）・`occurrences`（`unchanged`・`content-changed`・`generation-conditions-changed`・`subject-missing`・`subject-ambiguous`・`unknown-rule`・`unsupported-subject-kind`・`occurrence-removed`・`occurrence-added`）・`needsReconfirmation`。`crud-table`・`specification` Draftは現在の投影を再導出してOccurrence単位で比較する。**`three-element-diagram` DraftはWorkspaceを読まない**ので `-workspace` は不要で、記録されたモデルpathを読み直して比較する） | JSON（終了コード2: `INVALID_ARGUMENT`・`INVALID_DRAFT`・`DRAFT_READ_FAILED`・`DRAFT_DECODE_FAILED`・`WORKSPACE_LOAD_FAILED`・`SOURCE_INDEX_FAILED`。終了コード1: `DOCUMENT_DRAFT_FAILED`） | いいえ（投影型。内側に`tei.document-draft.reconcile/v0`を持つ） |
| `inspect` | `tei.cli.inspect/v0` | JSON | JSON（`INVALID_ARGUMENT`、`SUBJECT_NOT_FOUND`、`INSPECTION_FAILED`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`） | いいえ（投影型） |
| `trace` | `tei.cli.trace/v0` | JSON | JSON（`INVALID_ARGUMENT`、`INVALID_SUBJECT`、`SUBJECT_NOT_FOUND`（終了コード1）、`SUBJECT_AMBIGUOUS`（1）、`STALE_SNAPSHOT`（1）、`UNSUPPORTED_SUBJECT_KIND`（2）、`TRACE_FAILED`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`）。**`inspect`と違い、`semantic-candidate-key`・`semantic-functional-dependency`は投影しないため`UNSUPPORTED_SUBJECT_KIND`になる（#3180）** | いいえ（投影型） |
| `view` | `tei.cli.view/v0` | JSON（`-format json`。内側の`result`に`tei.applicationview/v0`を持ち、外側に`worldFingerprint`・`sourceIndexFingerprint`を持つ） | JSON（終了コード2: `INVALID_ARGUMENT`・`WORKSPACE_LOAD_FAILED`・`PERIOD_STATE_LOAD_FAILED`。終了コード1: `VIEW_FAILED`・`VIEW_UNSUPPORTED`・`COMMAND_REFUSED`・`OPERATION_LOG_WRITE_FAILED`・`LIFECYCLE_STATE_FAILED`、および`-execute`の実行失敗として `internal/command` が返す符号（`ExecutionError.Code`をそのまま出す。`INVALID_COMMAND_INPUT`・`COMMAND_UNAUTHORIZED`・`COMMAND_NOT_DECLARED`・`COMMAND_EXECUTOR_UNAVAILABLE`・`COMMAND_AUTHORIZER_UNAVAILABLE`・`COMMAND_DATASOURCE_READ_ONLY`・`COMMAND_PERIOD_CLOSED`・`COMMAND_PERIOD_STATE_UNAVAILABLE`・`COMMAND_PERIOD_STATE_FAILED`・`COMMAND_PERIOD_SCOPE_UNAVAILABLE`・`COMMAND_PERIOD_SCOPE_UNKNOWN`・`COMMAND_PERIOD_CORRECTION_REQUIRED`・`COMMAND_COMMIT_FAILED`・`COMMAND_ROLLBACK_FAILED`・`COMMAND_TRANSACTION_BEGIN_FAILED`・`COMMAND_EXECUTION_FAILED`・`COMMAND_RECORD_FAILED`・`COMMAND_LIFECYCLE_REFUSED`・`COMMAND_LIFECYCLE_RECORD_FAILED`・`COMMAND_LIFECYCLE_UNAVAILABLE`・`MISSING_COMMAND_EFFECT`・`UNDECLARED_COMMAND_EFFECT`。効果が適用された可能性がある場合は`COMMAND_EFFECT_UNKNOWN`が優先する）。符号を持たない失敗は`COMMAND_FAILED`） | いいえ（投影型） |
| `validate -workspace -json` | `tei.cli.validate/v0` | JSON（`declarationCounts`・`sourceRootFindings`を含む） | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`、`WORKSPACE_VALIDATION_FAILED`） | いいえ（投影型） |
| `workspace-files` | `tei.cli.workspace-files/v0` | `result`に`managedFiles`（各要素は`readByLoader`付き）と`sourceRootFindings` | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`） | いいえ（投影型。#2354第一陣で包んだ） |
| `source-index` | `tei.cli.source-index/v0` | 対象なし: `result`に入力元索引そのもの（`mode=index`）。対象あり: `result`に`operations.SubjectSources`（`mode=subject`） | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`） | いいえ（投影型。`mode`で2形。#2354第一陣） |
| `knowledge-context` | `tei.cli.knowledge-context/v0` | `result`に`knowledgecontext.Context`（`mode=context`）。`-disclose`では開示の絞り込み（`mode=disclosure`） | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`KNOWLEDGE_BUILD_FAILED`。fingerprint不一致は終了コード1） | いいえ（投影型。`mode`で2形。#2354第二陣） |
| `knowledge-qualify` | `tei.cli.knowledge-qualify/v0` | `result`に`knowledgecontext.QualificationResult`（`mode=qualification`） | JSON（`INVALID_ARGUMENT`、`KNOWLEDGE_BUILD_FAILED`） | いいえ（投影型。#2354第二陣） |
| `knowledge-review-plan` | `tei.cli.knowledge-review-plan/v0` | `result`に`knowledgecontext.BuilderReviewPlan`（`mode=plan`） | JSON（`INVALID_ARGUMENT`、`KNOWLEDGE_BUILD_FAILED`。fingerprint不一致は終了コード1） | いいえ（投影型。#2354第三陣） |
| `knowledge-review-handoff` | `tei.cli.knowledge-review-handoff/v0` | `result`に`knowledgecontext.BuilderReviewHandoff`（`mode=handoff`） | JSON（`INVALID_ARGUMENT`、`KNOWLEDGE_BUILD_FAILED`。fingerprint不一致は終了コード1） | いいえ（投影型。#2354第三陣） |
| `knowledge-review-disclose` | `tei.cli.knowledge-review-disclose/v0` | `result`に`knowledgecontext.DisclosureProjection`（`mode=disclosure`） | JSON（`INVALID_ARGUMENT`、`KNOWLEDGE_BUILD_FAILED`。fingerprint不一致は終了コード1） | いいえ（投影型。#2354第三陣） |
| `knowledge-data-design-candidate-review` | `tei.cli.knowledge-data-design-candidate-review/v0` | `result`に`knowledgecontext.DataDesignCandidateReviewPacket`（`mode=review`） | JSON（`INVALID_ARGUMENT`、`KNOWLEDGE_BUILD_FAILED`。fingerprint不一致は終了コード1） | いいえ（投影型。#2354第三陣） |
| `validate -workspace`（テキスト） | なし（テキスト） | テキスト | テキスト、終了コード1（診断あり）/2 | - |
| `world` | なし（証跡照合用の内部出力） | `ir.World` | 標準エラー＋終了コード2 | はい（raw stdoutのSHA-256をlive-r1-smokeが照合するため。#2354 Owner判断で外部契約の対象外） |
| `requirements` | `tei.cli.requirements/v0` | `result`に`dependency.Derive(w)`の配列 | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`） | いいえ（投影型。#2354第一陣） |
| `plugin-lock` | `tei.cli.plugin-lock/v0` | `result`に`plugin.Lock`（lock形式自体は`formatVersion: 2`を持つ） | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`PLUGIN_LOCK_FAILED`） | いいえ（投影型。#2354第一陣） |
| `plugin-resolve` | `tei.cli.plugin-resolve/v0` | `result`に`plugin.ResolvedSet` | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`UNKNOWN_ENVIRONMENT`、`INVALID_PLUGIN_LOCK`、`PLUGIN_RESOLVE_FAILED`） | いいえ（投影型。#2354第一陣） |
| `plugin-runtime-plan` | `tei.cli.plugin-runtime-plan/v0` | `result`に`pluginruntime.EnvironmentPlan` | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`UNKNOWN_ENVIRONMENT`、`INVALID_PLUGIN_LOCK`、`PLUGIN_RUNTIME_PLAN_FAILED`。構築失敗は終了コード1） | いいえ（投影型。#2354第一陣） |
| `plugin-inventory` | `tei.cli.plugin-inventory/v0` | `result`に`management.Report`（`-plugin`で`mode=detail`、`-diagnose`で`mode=diagnose`） | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`UNKNOWN_ENVIRONMENT`、`INVALID_PLUGIN_LOCK`、`PLUGIN_RESOLVE_FAILED`） | いいえ（投影型。診断は許可を与えない） |

観測の補足:

- **`requirements`だけは現行の成功時`result`がJSON配列**。`tei commands`の`requirements` entryに`resultShape: "array"`を追加する変更候補（#3347）は、既存の`tei.cli.requirements/v0`応答を変えずに例外を発見可能にする。`successShape`（外側の成功応答の構造）・`jsonContract`（エンベロープ等の扱い）・`resultShape`（`result`のJSON型）は異なる概念。`resultShape`を省略した他のコマンドは**object保証ではなく未宣言**であり、`world`のraw JSONや`version -json`のidentityに`result`があると推定しない。現在の`requirements`を実ビルド済CLIで検査する回帰試験に結び、他コマンドの`result`/diagnostics平坦化やbuiltin混在は本件で無断変更せず#3347で別判定とする。
- 引数の未対応flagは、#2332で全広告コマンドが拒否するようになった。拒否も標準エラー（`inspect`、`trace`、`validate -json`はJSON）で、flag解析のusageの分だけstderrが長くなるコマンドがある（例: `knowledge-context`に未知のflag）。`version -json`の引数不正も、2026-11-07から実装が標準エラーへ返す（表の記載と一致）。
- 2026-11-08から、`view -format json`・`source-index`（`mode=index`／`mode=subject`）・`knowledge-context`・`requirements`・`plugin-lock`・`plugin-resolve`・`plugin-runtime-plan`の成功応答も`workspace`を持つ。`-world`ファイルだけを入力した場合、manifestを読んでいないため`workspace`を省き、値を推測しない。これで読み取りを行う全コマンドが同じ欄で読み元Workspaceを示す（#3455）。既存の`result`内の`workspace`（`workspace-files`・`model catalog`・`inspect`・`trace`）は残す。
- 2026-11-07から、`validate -json`・`model catalog`・`inspect`・`trace`・`workspace-files`の成功応答は`workspace`を持つ。`name`はmanifestの`workspace:`、`formatVersion`はmanifestの版、`manifest.sha256`はmanifestのバイトの照合値である。複数のWorkspaceが同じCanonical IDを持つ場合に、`worldFingerprint`（内容の同一性）だけでは読み元を識別できないためで、既存の欄を変えずに足す互換な変更とする（#3181）。`workspace:`を書かないmanifestは読み込まれ、`name`は空になる。この欄は受理・権限・排他の証拠ではない。
- 2026-11-09から、**必須の引数が書かれていない**場合は、コマンドによらず`INVALID_ARGUMENT`（終了コード2）で返す。`inspect`は`INVALID_SUBJECT`ではなく`INVALID_ARGUMENT`を返し、`INVALID_SUBJECT`は「値は渡されたが subject として不正」に残す。`knowledge-context`・`knowledge-qualify`・`knowledge-review-*`の`-subject-kind`/`-subject-id`欠落も`INVALID_ARGUMENT`とし、`KNOWLEDGE_BUILD_FAILED`は知識の構築に失敗した場合だけに残す。これにより「引数不足」を1条件で判定でき、`-workspace`の付け忘れを subject の問題や構築失敗と誤診しない（#3733）。`validate`（`-json`無し）は従来どおり標準エラーに理由を1行書き、`-json`付きの場合は同じ失敗を`INVALID_ARGUMENT`の1 JSONで返す。
- 2026-11-09から、`model catalog`の`result`は`validation`（`ok`・`errorCount`・`warningCount`）を持つ。`ok`は`validate -workspace ... -json`と同じ規則（error重大度の診断もerror重大度のsource-root findingも無い）で、`model catalog`自身の`ok`とは別である。`model catalog`は案内用の投影であり、検証NGのWorkspaceでも`ok:true`・終了コード0で応答する（編集中の閲覧を妨げないため）。したがって「このモデルは健全か」を結果から判断する呼出し元は`validation.ok`を読む。診断の詳細は複製せず`validate -json`を参照する（#3674）。
- 2026-11-08から、`model catalog`の各`entries`要素は`prelude`を持つ。`true`はプロセス内の標準意味型（`std`名前空間とその型）であり、Workspaceが宣言したものではない（#3347）。`entries`だけを見ても、標準定義13件と利用者の宣言が混ざっていたためである。`declarationCounts.total`は`prelude`の要素を含まないので、空のWorkspaceは`entries`が13件でも`total:0`になる。欄の追加であり、`tei.cli.model/v0`と`tei.operations.model-catalog/v0`の既存欄は変えない。
- 2026-11-08から、`view -format json`の応答は外側に`worldFingerprint`と`sourceIndexFingerprint`を持つ。`model catalog`・`inspect`・`trace`・`validate -json`と同じ計算（`operations.WorkspaceProjectionFingerprints`）であり、同じWorkspaceなら値が一致する（#3345）。`result.semanticFingerprint`は残すが、こちらはApplicationが使用する名前空間だけを対象にした**別の観測**であり、`worldFingerprint`と比較してはならない。両方の欄はテキスト・HTML・printの出力には出さない（JSON契約の欄である）。Source Indexを構築できないWorkspaceでは2欄を省き、`view`自体は従来どおり応答する。
- 引数不正の出力先は、`cmd/tei/cli_argument_failure_streams_test.go`が実プロセスの標準出力と標準エラーを分けて検査する（#2993）。`version`・`commands`は標準出力を空のまま標準エラーへ理由を書き、`validate -json`は同じ失敗を標準出力の1 JSONで返す。この差はコマンドごとの決定であり、CLI全体の既定として読まない。
- `-h`／`-help`は引数不正ではない（#3185）。2026-11-07から、`inspect`・`view`・`trace`・`workspace-files`・`knowledge-*`・`commands`・`version`・`model catalog`は、終了コード0・標準出力に`usage: tei <command> [options]`とそのコマンドが広告するflagの説明を書き、標準エラーは空にする。未知のflag（`-not-a-flag`など）は従来どおり`INVALID_ARGUMENT`・終了コード2で、`-h`とは区別する。`cmd/tei/cli_help_test.go`が両方を実プロセスで検査する。`validate`は元からflag packageの既定（標準エラーに`Usage of`、終了コード0）で、この変更の対象外である。
- `tei commands`の`output`欄は、同じentryの`jsonContract`と一貫していることを`cmd/tei/cli_advertised_output_test.go`が検査する（#2824）。`json`／`text; json when …`と`enveloped`の組合せ、`raw`と`json`の組合せを定数で照合し、あわせて実プロセスでtext既定の3コマンドがJSONエンベロープを書かないこと、json既定の5コマンドが引数不足でも標準出力へ1件のエンベロープを書くことを確かめる。負例は`world`の`output`を`text`へ変えると失敗する。
- `validate -json`の失敗JSONは`diagnostics`を常に空配列として持ち、`error.code`と`error.message`のみで種別を伝える。
- `validate -json`の`diagnostics[]`は、要素が`{"diagnostic":{…},"sources":[…],"unresolvedSourceOrigins":[…]}`という**入れ子**の形で、同じ応答の`proposals[]`（`code`・`severity`・`diagnostic`・`accepted`・`revision`を持つ平坦な要素）とは異なる。この差は意図的に残している（#3347）。`diagnostics[]`は位置情報（`sources`）を要素側に持つため、診断本体を`diagnostic`の下へ置くと位置と診断の対応が1つの要素で読める。`proposals[]`は受理候補の表示用で、位置を要素ごとに持たない。平坦化は既存の読み手（`diagnostics[].diagnostic.code`を読む経路）の解釈を変えるため、v1を要する破壊的変更として別の判断に残す。`cmd/tei/cli_validation_test.go`の`TestValidateJSONDiagnosticsKeepNestedDiagnostic`が、v0の要素に入れ子の`diagnostic`があり、最上位に`code`が無いことを固定する。
- `validate -json`は、2026-11-07から`sourceRootFindings`を返す。`sources`の存在しないrootは`severity:"error"`（`ok:false`・終了コード1）、Loaderが解釈しない未知rootと、Loaderが読まない近似名の管理ファイルは`severity:"warning"`である。Canonical Subjectを持たないため`diagnostics`とは別の欄にする（#2823）。
- 空のWorkspace（モデル要素0件）は`validate`が`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`（warning）を返す。`ok:true`・終了コード0のままなので、`diagnostics`のwarningを確認する（#2822）。
- `validate -json`と`model catalog`は、2026-11-07から`declarationCounts`を返す。`std`の標準意味型を除いたWorkspace自身の宣言件数であり、空のWorkspace（`total:0`）と宣言のあるWorkspaceを機械的に区別できる（#2822）。
- `model catalog`の`modelRevision`は、管理対象Sourceのpathと生バイトの組から作る版であり、**読み込んだWorldの内容の版ではない**。管理対象Sourceを1件も読まないWorkspaceでは空入力のSHA-256（`sha256:e3b0c442…b855`）になり、標準意味型の`entries`が並んでいてもこの値になる。空かどうかは`modelRevision`ではなく`declarationCounts.total`で判定する（#2822）。この区別を`internal/sourceindex`・`internal/operations`・`cmd/tei`の試験と`docs/getting-started.md`へ書いた。
- `tei commands`の各entryは、2026-11-07から`schemaVersion`と`jsonContract`（`enveloped`／`raw`／`none`）を持つ。`world`だけが`raw`で、`tei.cli.*`のエンベロープを持たないことをdiscoveryから読める。
- 2026-11-07から、discoveryが広告する`schemaVersion`と、そのコマンドが実プロセスで返す応答の`schemaVersion`の一致を`cmd/tei/cli_discovery_process_test.go`が検査する。定数同士の照合（`cli_contract_test.go`）だけでは、応答側の改名と広告のずれを検出できなかった（#2824）。`version -json`は`ok`/`error`を持たないため、この検査は`schemaVersion`の一致だけを求める。
- 2026-11-07から、`tei commands`の各entryは`modes`（応答の`mode`が取りうる値の閉じた集合）を持つ。空文字列の要素は「その応答では`mode`が省略されうる」ことを示す。`mode`で結果の形が変わる`source-index`（`index`／`subject`）と`knowledge-context`（`context`／`disclosure`）、`plugin-lock`（`environment`または省略）、`plugin-resolve`（`unvalidated-candidates`または省略）が対象で、残りの`knowledge-*`は1形の結果型を`mode`で名乗る。`cmd/tei/cli_discovery_modes_test.go`が、実際の応答の`mode`が広告された集合に含まれることを実プロセスで検査する。
- 2026-11-09から、**同じ flag を2回以上書いた**場合、その flag が1つの値しか持たないなら `INVALID_ARGUMENT`（終了コード2）で拒否する。Go の flag は最後の値を残して何も報告しないため、環境既定値と個別指定を連結する呼出し元が、先に渡した値を黙って失っていた（#3675）。繰り返しを意図する flag は `tei commands` の `repeatableFlags` で広告する（`view` の `-input`・`-allow-capability`、`validate` の `-analysis`・`-schema`・`-plan`・`-evidence`）。`repeatableFlags` に無い flag は1値であり、2回目で拒否される。`model catalog` は自前の FlagSet を持つため同じ検査をその場で適用する。`-input` のように値が次の引数へ続く flag は、値が `-` で始まっても flag と誤認しない。位置引数より後は flag 解析が止まるので検査もしない。
- 2026-11-10から、`trace`の「未解決」を表す2つの欄の意味を次のとおり確定する（#3906）。
  - `result.trace.unresolved` は**返した行のうち未解決だった行の件数**である。`result.trace.rows[].issue` が入っている行を数える。
  - `result.projectionUnresolved` は**グラフ全体の投影で落ちた参照の一覧**であり、要求した subject に限定されない。実測では `reference/workspace` の72 subject 中66を trace し、全件が同一の17件を返し、いずれも要求した subject を名指していなかった。`-scope` を変えても同じ17件である。
  - したがって **`trace.unresolved: 0` は「Workspace に未解決が無い」を意味しない**。`projectionUnresolved` が空でない場合、未解決は存在するが、その subject の行としては現れていない。両方を読むこと。
  - `rows[].issue` に入りうる値は `invalid-source`・`ambiguous-source`・`invalid-target`・`invalid-edge-type`・`invalid-rule`・`conflicting-rule` の6つだが、**CLI 経路で到達できるのは一部だけ**である。`internal/modelref/derivedbridge/projection.go` が辺から規則を作るため `invalid-edge-type`・`invalid-rule`・`conflicting-rule` は構造的に発生せず、同名要素の二重宣言は `validate` が `DUPLICATE_SEMANTIC_ELEMENT` で先に拒否するため `ambiguous-source` も trace まで届かない。到達不能な値を「起こりうる」と読まないこと。
- 2026-11-10から、`inspect`の`result`は`authorityLimits`を持つ。内容は固定の4項（`inspection-is-a-derived-read-only-projection`・`source-location-is-not-semantic-acceptance`・`source-location-is-not-write-authority`・`local-read-is-not-disclosure-permission`）。`trace`と`model catalog`は既に同種の表明を持っており、Sourceのパスと件数を返す`inspect`だけが無表明だった（#1178）。**`local-read-is-not-disclosure-permission`は認可の判定結果ではない**: ローカルで読めることは別の相手へ渡してよいことを意味しない、という限界の表明であり、開示の可否を決めるのは既存のAuthorization Capabilityである。`-scope`は認可scopeでもtenant境界でもない。
- 2026-11-10から、知識系コマンドの**読み込む JSON ファイルの不備**はすべて`INVALID_ARGUMENT`（終了コード 2）で答える。対象は、パスが開けない・JSON として不正・知らないフィールド・**JSON 値が2つ以上**である。以前は最後の1つだけが生のエラーを返して`KNOWLEDGE_BUILD_FAILED`になっていた（#3943）。メッセージは**パスを渡したオプション名**（`-qualification-requirements` または`-review-handoff`）を必ず含める。1つの読込担当を複数のオプションで共有していたため、以前は `knowledge-review-disclose -review-handoff` でも `-qualification-requirements` を指示していた。`KNOWLEDGE_BUILD_FAILED` は、ファイルを読めた上で知識の構築に失敗した場合に限る。
- 2026-11-09から、`trace -mode full`の結果は`requestedOffset`を持ちうる。`offset`は**このページが実際に始まった位置**で、要求した`-offset`が最終行を超えているときは行数へ切り詰められる。その場合だけ`requestedOffset`に要求値を残すので、切り詰められたページと「残りが0件」のページを応答だけで区別できる。切り詰めが無ければ欄は省略する。`mode=summary`は行を返さないため常に省略する（#1177の「切捨て有無を観測できる」）。
- 2026-11-09から、`trace`の`-direction`・`-mode`・`-expected-fingerprint`は、フラグを書いたうえで値が空なら`INVALID_ARGUMENT`（exit 2）で拒否する。フラグを省略した場合だけ既定（`direction=both`・`mode=summary`）を使う。空文字列を「省略」として別の値へ落とすと、`mode`では`summary`と`full`が入れ替わり、`-expected-fingerprint`では照合が省略される（#3605）。
- 2026-11-08から、`source-index`の`mode=index`応答は、参照が0件でも`result.references`を`[]`で返す（`null`にしない）。`internal/sourceindex.Index`の`MarshalJSON`が空の`artifacts`・`references`を空配列にする。同じ「参照の一覧」を`inspect`・`trace`が配列で返すため、読み手が`null`と`[]`を個別に扱う必要をなくす（#3456）。`null`を「欄が無い」と解釈していた読み手には、この変更を告知する。
- 2026-11-07から、`tei commands`の各entryは`successShape`（`envelope`＝`ok`/`result`、`identity`＝`version -json`のbuild識別）を持つ。省略時は`envelope`であり、成功時に`ok`を持ち、`result`を持つ。`version`だけが`identity`で、`ok`も`result`も持たない。`cmd/tei/cli_discovery_shape_test.go`が、実応答の本体が広告された形と一致することを実プロセスで検査する。
- 2026-11-07から、`tei commands`の各entryは、閉じた値集合を持つ引数について`argumentValues`（許容値）、`argumentType`（受け付ける形。`-disclose`はカンマ区切り）、`argumentDefaults`（省略時の値）を持つ。対象は`inspect`・`trace`の`-subject-kind`（両者は集合が違う）、`trace`の`-direction`と`-mode`、`view`の`-format`、`knowledge-context`の`-disclose`、`knowledge-data-design-candidate-review`の`-data-design-candidate-kind`である。値集合は実装の検証に使う変数（`operations.InspectionSubjectKinds`・`operations.TraceSubjectKinds`・`modelref.TraceDirections`・`modelref.TraceModes`・`knowledgecontext.SelectableDisclosureSections`）を参照しており、定数を別に複製していない。`cmd/tei/cli_discovery_argument_values_test.go`が、広告値の全件を実プロセスで受理し、集合外の値を拒否すること、広告と実装の集合が一致することを検査する。パス・ID・fingerprint・scopeのように閉じた集合を持たない引数は載せない（#3242）。
- 2026-11-07から、`view`のentryは`writeFlags`（`-operation-log`・`-lifecycle-state`）を持つ。`effect: "read-only"`は**正本の意味を書き換えない**ことを示し、プロセスがファイルを書かないことまでは示さない。`view -execute`は、これらのflagで指定した場所にだけ記録・段階storeを作る（指定しなければ書かない）。ファイル書込みを禁じる呼び手は、`effect`だけでなく`writeFlags`のいずれも渡っていないことを確認する。`cmd/tei/cli_discovery_write_flags_test.go`が、実バイナリでflagごとにファイルが作られること、flagなしでは作業ディレクトリに何も作られないことを検査する（#3182）。
- 2026-11-07から、`-h`（と`-help`）の出力に、`argumentValues`・`argumentDefaults`と同じ値集合と既定値を表示する（`values:`／`default:` 行）。値集合は discovery と同じ変数を参照しており、`-h` と `tei commands` のどちらも正本にならない。`cmd/tei/cli_help_values_test.go`が、`inspect`／`trace`／`view`／`knowledge-context` の `-h` が discovery と同じ集合と既定値を出し、閉じた集合を持たない flag には `values:` 行を出さないことを検査する（#3242）。
- 2026-11-11から、`tei commands`の各entryは`requiredFlags`・`optionalFlags`を持つ。どちらも**フラグ名だけ**（先頭の`-`なし、散文・括弧書きなし）で、両者の和がそのコマンドの受理する全フラグになる。既存の`required`・`optional`は残す。こちらは「なぜ必須か」「どの組み合わせで使うか」という規則を文で持ち、機械は読めない。たとえば`exactly one of -workspace or -world`は`requiredFlags`では両方の名前として現れ、二者択一という制約は`required`側の文が担う。`optionalFlags`は受理集合から必須を除いて導くため、コマンドが読まないフラグを広告しない（#4166）。あわせて`version`の`required`から`none`（フラグ名の位置にあった値）を削除し、`validate`の`-json (workspace only)`から括弧書きを外した（`output`がすでに`text; json when -workspace -json`と述べている）。

- 2026-11-09から、明示の空文字列（`-mode ""` など）を「未指定」として扱わない。未指定はフラグを省いた場合だけであり、空文字列は `INVALID_ARGUMENT`（終了コード2）で拒否する。対象は値が閉じた集合か完全一致の選択子か指紋である flag（`trace` の `-mode`・`-direction`・`-expected-fingerprint`・`-scope`・`-subject-kind`・`-subject-id`、`inspect`・`source-index` の選択子、`knowledge-context` の `-disclose` と指紋、`plugin-*` の `-environment`・`-plugin-lock` など）。`model catalog` は自前の FlagSet を持つため同じ規則をその場で適用し、`-namespace ""` を拒否する（`-namespace` を省いた場合は従来どおり絞り込み無し）。この規則により、`argumentDefaults` が広告する既定値（`trace` の `mode: summary`）と、`-mode ""` が内部で `full` に落ちていた適用値の食い違いが解消される。`cmd/tei/cli_empty_argument_test.go` が、単体と実バイナリで、空文字列の拒否・省略時の既定維持・`model catalog` の `-namespace` を検査する（#3605）。
- 2026-11-09から、`trace` の2つの「未解決」の意味を区別して記す。`result.trace.unresolved` は**選択した subject に直接繋がる行**のうち解決できなかった行の数であり、`result.projectionUnresolved` は**グラフ全体を投影する過程で捨てた辺**の一覧である。両者は別物だが名前が近く、`reference/workspace` は前者が0で後者が17件になるため、内側の数だけを見て「未解決は無い」と結論すると誤る（#3906）。また `modelref` の行が持ちうる `issue`（`invalid-source`・`ambiguous-source`・`invalid-target`・`invalid-edge-type`・`invalid-rule`・`conflicting-rule`）のうち、`invalid-edge-type`・`invalid-rule`・`conflicting-rule` は CLI 経路では起こらない。`derivedbridge.FromIndex` が投影した辺そのものから規則を作るため規則の欠落・空欄・重複が生じず、`invalid-target` は投影時に `projectionUnresolved` の `unmaterialized-to` へ落ちて行にならず、`ambiguous-source` は同名の要素を2種宣言すると `validate` が `DUPLICATE_SEMANTIC_ELEMENT` で先に拒否する。**`trace` の応答を読む側は、未解決の有無を `projectionUnresolved` で判断する。** 実測は `plan/evidence/TEI_DOGFOODING_ROUND31_TRACE_ISSUE_REACHABILITY_2026-11-09/` にある。
- 2026-11-08から、`knowledge-context` は `-subject-kind` の `semantic` を名前空間として解決する（従来の`semantic-namespace`も受け付ける）。`model catalog` の `entries[].subject.kind` と `inspect`・`trace` は名前空間を `semantic` と呼ぶため、catalog の値をそのまま渡した呼び手が `result.semanticElement` 無しの `ok:true` を受け取り、「定義が無い」と誤読していた。`cmd/tei/cli_knowledge_context_kind_test.go` が、catalog が広告する kind と `semantic-namespace` の両方が `semanticElement` を解決すること、未知の kind は解決せず subject を書き換えないことを実バイナリで検査する（#3242の後続）。

## 5. v0の中での互換な変更と版を上げる変更（規則案）

`tei.cli.<command>/v0`を掲げる出力は、次の区別を契約とすることを提案する。

互換な変更（v0のまま行ってよい）:

- 既存の読み手が未知の欄を無視する前提で、応答に新しい欄を足すこと（#2318の`version -json`、#2363までの`validate`の`worldFingerprint`追加がこの前提で行われている）。
- 配列の要素型に新しい欄を足すこと。
- `error.code`に新しい値を足すこと。読み手は既知のcodeだけを特別扱いし、未知のcodeを一般失敗として扱う。
- 出力の文章（`message`）を、`code`の意味を変えずに整えること。

版を上げる変更（`v1`を新設し、`v0`の読み手への告知を伴う）:

- 既存欄の削除、名前の変更、型の変更。
- 既存欄の意味の変更（例: `worldFingerprint`を競合制御の版として扱う告知）。
- 失敗時の応答の形の変更（`error`の構造の変更、schemaVersionを持たない出力にschemaVersionを足して契約に上げることもこれに当てる）。
- 終了コードの意味の変更。

schemaVersionを持たない現在の出力は、この規則の対象外であり、契約として扱わない。内部構造体の変更（例: `ir.World`の欄の追加）が、告知なく出力の変更になる。

## 6. schemaVersionを足す範囲の案

既存の欄を変えずに足せるのは、欄の追加が互換な変更だからである。ただし足す対象は、版固定が必要な読み手がいる範囲に限定する。次の三つを比べた。

- **A案（推奨）: 外部契約に上げるコマンドを選び、投影型を挟む。** 対象候補は`workspace-files`、`source-index`、`requirements`、`plugin-lock`、`plugin-resolve`、`plugin-runtime-plan`、`knowledge-context`のうち、外部の自動化（TAIなど）が機械処理しやすい一覧・索引・lock系。`world`は`live-r1-smoke`の生バイト証跡照合があるため対象外（2026-11-02 Owner判断）。投影型は既存の欄をそのまま写し、`schemaVersion`と既存の出力を`result`として包む形にする。`inspect`と同じ`ok`/`error`の形に揃えれば、失敗時の形式も同時に機械可読になる。既存の出力を直接読んでいる読み手は、包まれた出力をそのまま読み続けられる（包むことは形の変更なので、読み手の告知は必要）。
- **B案: 現在のまま、内部デバッグ用と宣言する。** `commands`の出力の`output`欄に「json（内部構造、契約外）」と明記し、契約は`tei.cli.*/v0`を持つコマンド群に限定する。変更は最小だが、外部読み手が版を判別できない状態が続く。
- **C案: すべての広告コマンドに一律にschemaVersionを足す。** 対象の選別が不要な代わりに、契約として維持する意図のない出力まで維持義務を生む。

A案の場合の具体的な足し方（既存の欄を変えない）:

- `source-index`は対象指定の有無で出力の形が変わるため、二つの契約（例: `tei.cli.source-index/v0`と`tei.cli.source-index-subject/v0`）に分けるか、`mode`欄で区別する。`knowledge-context`の`-disclose`も同様である。
- `plugin-lock`は出力自体が`formatVersion: 2`を持つため、包む投影型には`schemaVersion`を足さず、lock形式の版をそのまま正本として扱う選択もある。

## 7. 失敗時の形式の決定案

- **D案（推奨）: 機械可読の失敗応答を、schemaVersionを持つ投影型コマンドへ広げる。** 形は`validate -json`の`error.code`＋`error.message`、応答全体は`inspect`と同じ`ok`/`error`とする。codeの集合は`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`を基準に、コマンド固有のcodeを足す。終了コードの意味（0/1/2）は変えない。
- **E案: 現状を契約として明記する。** schemaVersionを持たないコマンドは「失敗時、標準出力は空か不定、標準エラーに自由文、終了コード1/2」と明記し、機械可読化はしない。呼び手は終了コードと文面に頼る。`scenario`の実装（標準エラー＋終了コード1）と同じ方向である。

D案を選んでも、失敗時JSONをstderrでなくstdoutへ出す現在のC1コマンドの慣行に合わせる。どちらの場合も、テキスト出力の`validate`（非`-json`）の診断の文面は契約外のままである。

## 8. 結論を出していない範囲

- `world`は2026-11-02のOwner判断で外部契約の対象外とし、生の`ir.World`出力を維持する。エンベロープ化する場合は、`tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs`の生バイト証跡照合（`derivedWorldSha256`）を先に再設計する。`plugin-lock`はlock形式の`formatVersion: 2`を正本として扱い、`tei.cli.plugin-lock/v0`の`result`内に保持する。
- schemaVersionを足す場合の、v0のまま足すか、新契約としてv1を新設するか。既存の出力を包む場合は読み手への告知が必要で、告知の要否と時期を含めてIssueで決める。
- 一覧の表と実装の一致は、2026-11-07から`cmd/tei/cli_contract_doc_test.go`が機械検査する。`tei commands`が広告するenvelopedの各コマンドが、§4の表に同じ`schemaVersion`で載っていること、表が広告にないコマンドを載せていないことを検査する。表の行を増減したときは、この試験が対応を保つ。`tei commands`の`output`欄と実装の対応も別の試験で保たれている。
- 実測は参照作業領域と失敗例の一組だけである。大規模な作業領域での出力の形（配列の空のときの`null`/`[]`の違いなど）は確認していない。

