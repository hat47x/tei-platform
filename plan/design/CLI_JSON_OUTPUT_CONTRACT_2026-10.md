# CLI読取りコマンドのJSON出力契約（schemaVersionと失敗時の形式）の現状調査と追加案

- **状態**: 調査と案（提案中、未採用）。一覧は2026-10-31時点のorigin/main `cd47b6fa4` のコードと、WSL go1.26.9でbuildした`tei`による実行結果の記録である。
- **担当課題**: [#2354](https://github.com/hat47x/tei-platform/issues/2354)。
- **関連文書**: [CLIを主入口とするインターフェース設計](CLI_FIRST_INTERFACE_DESIGN.md)、[inspect・traceの開示境界](CLI_C2_DISCLOSURE_BOUNDARY_DESIGN.md)、[全体構成](../../docs/architecture.md)、[開発手順](../../docs/getting-started.md)。

## 1. 役割と正本との関係

この文書の役割は、`tei commands`が広告する読取り専用コマンドの出力形式について、(1)現状の一覧、(2)`tei.cli.*/v0`の中で互換とみなせる変更と版を上げる変更の区別の規則案、(3)schemaVersionを既存の欄を変えずに足す範囲の案、(4)失敗時の形式の決定案、を一か所にまとめることである。

出力形式の正本は実装（`cmd/tei`）と、実装と表が一致することを確認する試験である。この文書は調査結果と案を記録するもので、契約そのものの正本にはならない。ここで決めない判断（外部契約にする対象の確定、v0の維持かv1への移行か）は、Issueに残したままOwnerの判断を待つ。

## 2. 対象範囲

`tei commands`が広告する13コマンド（`cmd/tei/cli_contract.go`）のうち、読取り専用でJSONを出すものを扱う。書込み系（deployment-*など）と、`commands`自身の引数扱いは、失敗時の形式の一部としてだけ触れる。`scenario`や`model`のサブコマンド以外の広告外コマンドは扱わない。

## 3. 現在の結論（先に要約）

- `world`を除く、`tei commands`が広告する読取り専用JSONコマンドは`tei.cli.<command>/v0`を持ち、成功時は`ok: true`と`result`、失敗時は`ok: false`と`error.code`を標準出力のJSON 1件で返す。終了コードの意味（0/1/2）は従来どおり。
- `world`は例外で、`tei.cli.*`のエンベロープを持たない。`tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs`が`tei world`の生標準出力SHA-256を受理済みDecisionの`derivedWorldSha256`と照合するため、インデントやエンベロープの追加は証跡照合と両立しない。`world`は内部・証跡用途であり、外部の安定JSON契約の対象外とする（2026-11-02 Owner判断）。
- 実測では、製品CLI（`cmd/tei`）が外部のGo moduleに依存しないのと同様に、**広告された読取りコマンドのうち`tei.cli.*/v0`を持たない出力は、内部構造体の変更がそのまま外部向け変更になる**。`world`はこの例外として扱い、その他のコマンドは投影型で版固定する。

## 4. 現状の一覧（実測）

実行環境: WSL go1.26.9で`go build ./cmd/tei`、参照作業領域`reference/workspace`。失敗時の例は存在しない作業領域、存在しない対象、未対応のflagを与えた場合である。終了コードは成功0、意味的な見つからない1、引数と読込みの失敗2が基本である。

| コマンド | schemaVersion | 成功時の出力 | 失敗時の標準出力 | 内部構造体の直接出力 |
| --- | --- | --- | --- | --- |
| `version -json` | `tei.cli.version/v0` | JSON | 引数不正は標準エラー | いいえ（投影型） |
| `commands` | `tei.cli.discovery/v0` | JSON | 引数不正は標準エラー | いいえ（投影型） |
| `model catalog` | `tei.cli.model/v0` | JSON | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`） | いいえ（投影型。内側に`tei.operations.model-catalog`の版を持つ） |
| `inspect` | `tei.cli.inspect/v0` | JSON | JSON（`INVALID_ARGUMENT`、`SUBJECT_NOT_FOUND`、`INSPECTION_FAILED`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`） | いいえ（投影型） |
| `trace` | `tei.cli.trace/v0` | JSON | JSON（`inspect`と同じ形） | いいえ（投影型） |
| `validate -workspace -json` | `tei.cli.validate/v0` | JSON | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`、`SOURCE_INDEX_FAILED`、`WORKSPACE_VALIDATION_FAILED`） | いいえ（投影型） |
| `workspace-files` | `tei.cli.workspace-files/v0` | `result`に`workspace.ManagedFile`の配列 | JSON（`INVALID_ARGUMENT`、`WORKSPACE_LOAD_FAILED`） | いいえ（投影型。#2354第一陣で包んだ） |
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

観測の補足:

- 引数の未対応flagは、#2332で全広告コマンドが拒否するようになった。拒否も標準エラー（`inspect`、`trace`、`validate -json`はJSON）で、flag解析のusageの分だけstderrが長くなるコマンドがある（例: `knowledge-context`に未知のflag）。
- `validate -json`の失敗JSONは`diagnostics`を常に空配列として持ち、`error.code`と`error.message`のみで種別を伝える。
- `plugin-lock`は出力自身に`formatVersion: 2`を持つ。schemaVersionはないが、lock形式には版がある。

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
- 一覧の表と実装の一致を機械検査にするかどうか。`tei commands`の`output`欄と実装の対応は試験で保たれているが、schemaVersionの有無の一覧はこの文書の手動の記録であり、#2354の完了条件には入っていない。
- 実測は参照作業領域と失敗例の一組だけである。大規模な作業領域での出力の形（配列の空のときの`null`/`[]`の違いなど）は確認していない。
