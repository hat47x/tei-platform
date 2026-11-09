# Gate B Readiness Audit

- 判定日：2026年9月5日（初回 Not Ready）／2026年9月19日（再監査 Ready）
- 対象：`order_entry`のアプリケーションドライバー／判定段階B
- 判定：**Ready**（候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）
- 製品受入シナリオ：PAS-02を主対象、PAS-03を副次的対象とする
- 検証根拠の要件： [`GATE_B_EVIDENCE_ACQUISITION.md`](GATE_B_EVIDENCE_ACQUISITION.md)
- 実行手順: [`GATE_B_EVIDENCE_RUNBOOK.md`](GATE_B_EVIDENCE_RUNBOOK.md)
- リポジトリの課題：ISSUE-000029-gate-b-readiness-audit.md／GitHub #273
- 引継ぎ： [`GATE_B_TO_C_HANDOVER_CONTRACT.md`](GATE_B_TO_C_HANDOVER_CONTRACT.md) ／基準版の記録： [`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md)

## 目的

判定段階Bの「画面なしで主要シナリオを再現できる」という要件は、終了した課題の数や設計済みという主張ではなく、**ソースの版を固定して実際に取得した検証根拠**に基づいて判定します。

最初のウェブ実行先へ進む前に、アプリケーション上の意味をウェブ固有のコードで補う必要がないことと、必須の検証根拠を変更のない一つの版に固定できることを確認します。

## 現行scope

判定段階Bの合格に必要な、ソースの版に依存する検証根拠は次の三つです。

| Evidence | Repository / GitHub Issue | nature | 現在地 |
| --- | --- | --- | --- |
| #310 | `ISSUE-000032` / #310 | deterministic repository | 成立（R = `e5231839`） |
| #255 | `ISSUE-000025` / #255 | generated artifact reproducibility | 成立（R = `e5231839`） |
| #308 | `ISSUE-000031` / #308 | real-provider ephemeral fixture | 成立（R = `e5231839`） |

三つとも、変更のない同じ候補版`R`を対象とします。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
Readiness Audit for R
```

実際のOracleに関する#214・#371・#376は、外部資産についての重要な検証根拠として引き続き扱います。ただし、第四の必須検証経路ではありません。その取得状況は、この判定段階Bの結果に影響しません（後述の「actual Oracle」を参照）。

## 現在の結論

**判定段階B：Ready**です（候補版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。

### 実行経緯

2026年9月5日のNot Ready判定後、実際の検証根拠を取得するため、専用のリポジトリ全体の取得、空のPostgreSQLデータベース、DSNの秘密情報ファイルの分離を用意しました。その際、`#310`の事前確認である`go run ./cmd/tei validate -workspace reference/workspace`は、既存のソース位置の追跡に関する二つの不具合で失敗しました。

1. `internal/syntax/semantic_anchor.go`の`semanticDataAnchors`が、関係の宣言に続く`key <source> -> <target>`句（`reference/workspace/semantic/sales/order.tei`）を未知の構成要素として拒否していました。
2. `internal/workspace/source_index.go`の`database-contract-query`による対象の検索で、`contract <id> <version> { ... }`に囲まれた問い合わせの宣言に対し、最上位の宣言専用の位置特定用部品を誤って使用していました。

どちらも、位置情報と索引を読み取る軽量な解析器が、本体の構文解析器の文法拡張に追従していなかったことが原因です。意味の正本やコアスキーマを変更するものではありません。PR [#726](https://github.com/hat47x/tei-platform/pull/726)で、`internal/syntax/semantic_anchor.go`に`skipSemanticRelationKeys`、`internal/syntax/anchor.go`に`ContractQueryDeclarationAnchor`を追加し、対応する単体試験も追加しました。検証が通るようになってから判明した、`reference/world.json`・`reference/workspace/.tei/plugin-lock.json`・永続実行記録の期待出力の古さも、同じPRで正式な生成器（`tools/reference-golden -write`・`tools/reference-durable-run-golden -write`）を使って再生成しました。この差分は、新しい問い合わせ実行の提供実装、`sales_core.order_line`の契約追加、`order_entry.order_lines`の重複項目の除去など、正本の作業領域で実際に進んだ変更を反映しています。

修正後に`#387`の最終確認を実行すると、独立した成果物監査（`tools/reference-gate-b-readiness-artifact-audit`）が三つ目の不具合を検出しました。

3. `tools/reference-gate-b-readiness-artifact-audit/main.go`は、`#308`の`contractQueryTarget`について、問い合わせ名だけ（`find_by_order`）を期待値として固定していました。しかし、`internal/realization/contract_query.go`の`ContractQueryTarget.Identity()`は、設計上、`Database/Contract/Query`の三つ組（`sales_core/sales_core.order_line/find_by_order`）を返します。監査ツール側の期待値が古い形式のまま残っていました。PR [#727](https://github.com/hat47x/tei-platform/pull/727)で、監査ツールの期待値と対応する試験用データを三つ組の形式に更新しました。

PR #726と#727はいずれもsquash mergeで`main`へ統合しました（`main`の統合コミット：`bba97e21`、`e5231839`）。統合後の`main`のHEAD（`e5231839a5dcc62cce1181a2731fcb7c70d16736`）を変更のない状態で取得し、正本となる作業領域の検証、最終確認の実行器の事前試験、`#387`の最終確認、独立した成果物監査を初めから再実行しました。いずれも成功しました。

### 満たされた条件

- #310 / #255 / #308 actual run（same R = `e5231839a5dcc62cce1181a2731fcb7c70d16736`）
- 三つのsame-R final convergence（`#387`）
- final convergenceが実際に読んだsame-run child Evidence artifact保持（`childArtifactsRetained = true`）
- same-run child exact artifactへの独立artifact audit（`artifactAuditPassed = true`）
- Rについて本監査を再実施（本ファイル、判定日2026-09-19）

各条件の充足状況は、後述の「Ready条件」のチェックリストを参照してください。

## 成立済みの意味基線

次の項目は、設計が未完了であることによる主要な阻害条件ではありません。

### Application / Driver

`browse / record / reference / header-detail / command`を同じ`order_entry`アプリケーションから解決し、少なくとも次の処理を一連の流れとして接続する構造が成立しています。

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation read
```

命令の受理応答と、実際に観測した状態を同一視しません。

### Data / Capability / authority

- logical Datasource / read-read-write境界
- Contract Query
- explicit input / raw result / Semantic mapping
- Relation Storage
- Capability Binding
- Authentication / Authorization
- Business Result / named Outcome / Execution Error
- external Evidence intake

生の物理的な事実や提供実装固有の戻り値から、意味の正本を逆向きに定義しません。

## Required Evidence 1 — #310 deterministic Repository

正規runner:

```text
tools/reference-gate-b-repository-evidence
```

合格判定には、少なくとも次の条件を求めます。

- 判定段階Bの検証根拠を取得する専用の完全なリポジトリ取得
- 追跡対象・追跡対象外・無視対象のローカル状態がないこと
- Gitの索引で、追跡対象の項目が通常の`H`のみであること（`assume-unchanged`・`skip-worktree`なし）
- 作業領域の検証に成功
- 画面を使わない最終シナリオに成功
- 外部の検証根拠の取込みに成功
- リポジトリのソース境界の準拠確認に成功
- `repositoryRevision = R`
- `reproducibleRevision = true`
- 実行後のソースを確認済み
- 実行前後でHEADが同一
- 終了時にソースと索引に変更がないこと

合成データによる検証根拠を、実際の外部資産の検証根拠として読み替えません。

## Required Evidence 2 — #255 generated artifact reproducibility

正規runner:

```text
tools/reference-generated-artifact-evidence
```

合格判定には、少なくとも次の条件を求めます。

- 正本のモデルとプラグインの実装固定情報が一致
- 永続的な実行記録が一致
- `repositoryRevision = R`
- `reproducibleRevision = true`
- 実行後のソースを確認済み
- ソースと索引が変更のない同じ状態

生成成果物が古い場合に限り、正式な生成器を使って修復します。差分を確認してコミットした後、変更のない新しい版で、変更を加えない確認を再実行します。

`-write`の直後に作業ツリーへ変更が残っている状態は、完了の検証根拠にはなりません。

## Required Evidence 3 — #308 real PostgreSQL

正規runner:

```text
tools/reference-postgres-header-detail-runtime
```

これは本番の既存データベースについての検証根拠ではありません。実際のPostgreSQLエンジンと、ツールが管理する一時的な試験用の構成要素による検証根拠です。

合格判定には、少なくとも次の条件を求めます。

### source

- 判定段階Bの検証根拠を取得する専用の完全なリポジトリ取得
- 追跡対象・追跡対象外・無視対象のローカル状態がないこと
- Gitの索引で、追跡対象の項目が通常の`H`のみ
- `repositoryRevision = R`
- `reproducibleRevision = true`
- 実行後のソースを確認済み

### fixture前

- 既定のデータベースではない
- 利用者定義の関係が0件
- イベントトリガーが0件
- 試験用の構成要素が存在しない
- `fixtureDatabaseSafety.guardPassed = true`

### runtime

- 2件取得
- 正常0件
- 表が存在しない場合の実行時エラー
- 元の結果から意味上の表現への明示的な対応付け
- 補助列を外部へ露出しないこと

### cleanup

- データベース側の所有者を示す印が厳密に一致
- 後片付け後、利用者定義の関係が0件
- イベントトリガーが0件
- 試験用の構成要素が存在しない
- `fixtureCleanupSafety.verified = true`

DSN、認証情報、引数の値、実際のデータベース名は、通常の検証根拠には保存しません。

## #387 final convergence

候補版`R`のソース修正をすべて終えた後、`GATE_B_EVIDENCE_RUNBOOK.md`の正式な条件に従い、最終確認の実行器を実行します。

成功を示す最終結果には、少なくとも次を要求します。

```text
evidenceNature = gate-b-three-lane-final-convergence
repositoryRevision = R
workingTreeClean = true
reproducibleRevision = true
allLanesSameRevision = true
childArtifactsRetained = true
postRunRepository.checked = true
postRunRepository.revisionStable = true
postRunRepository.workingTreeClean = true
```

各検証経路では、次を要求します。

```text
passed = true
repositoryRevision = R
reproducibleRevision = true
artifactFile != empty
evidenceArtifactFingerprint != empty
```

検証根拠の種類は次のとおりです。

```text
#310 deterministic-repository
#255 generated-artifact-reproducibility
#308 real-postgres-ephemeral-fixture
```

## same-run child Evidenceを監査する

最終結果だけで合格とは判定しません。

最終確認の実行器が実際に読み取った三つの子処理のJSONのバイト列そのものを、同じ実行に属する成果物としてリポジトリの外へ保存していることを要求します。

保存する成果物は次のとおりです。

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

合格判定では、内容を変更しない補助ツール`tools/reference-gate-b-readiness-artifact-audit`を用いて、最終結果と同一実行内の子処理の成果物を独立して再照合します。実行するコマンドと順序については、`GATE_B_EVIDENCE_RUNBOOK.md`を唯一の正式な手順とし、この文書にコマンド本文を複製しません。

この成果物監査は第四の必須検証経路ではありません。`artifactAuditPassed = true`だけで判定段階Bの合格を宣言しません。

合格判定では、次を確認します。

1. 最終結果の`artifactFile`が、同一実行で得た所定のファイルを指すこと
2. 元のファイルのバイト列のSHA-256が`evidenceArtifactFingerprint`と一致すること
3. 子処理のJSON本文と最終結果の検証経路の記録で、種類・版・再現可能性が一致すること
4. #310の作業領域の識別情報、および#310・#255の各確認項目の生成元コマンドが正式な契約と厳密に一致すること
5. 子処理のJSON本文から#310・#255・#308に固有の詳細な安全条件を確認できること
6. 秘密情報・認証情報・DSN・物理的な識別情報など、禁止された項目を通常の検証根拠に保存していないこと

を確認します。確認項目の名前だけ一致していても、生成元のコマンドが異なる成果物を同じ検証根拠として扱いません。

**同じ版Rで後から再実行した別の成果物を、最終確認で読み取った成果物と同一だとは推測しません。**

## final runner実行環境

### Repository

- リポジトリのルートから起動
- 判定段階Bの検証根拠を取得する専用の完全なリポジトリ取得
- 追跡対象・追跡対象外・無視対象のローカル状態がないこと
- Gitの索引で、追跡対象の項目が通常の`H`のみ
- `assume-unchanged`・`skip-worktree`なし
- 変更のない候補版R
- 各検証経路の実行後にソースを再確認
- 最終的にソースを再確認

### Go environment

実際の実行では、次を要求します。

```text
goEnvironment.externalWorkspaceActive = false
goEnvironment.goFlagsEmpty = true
goEnvironment.goEnvDisabled = true
```

を要求します。

実行器自体をコンパイルする前から、正確な`GOENV=off`の指定、外部のGOWORKの無効化、空のGOFLAGSを要求します。

### PostgreSQL secret

- 最終確認の起動環境へ`TEI_TEST_POSTGRES_DSN`を直接設定しない
- `TEI_TEST_POSTGRES_DSN_FILE`はリポジトリ外にある既存の通常ファイルを指定
- シンボリックリンクを解決した後もリポジトリ外
- 最終確認の実行器が元の秘密情報ファイルを安定した形で一度だけ開き、正規化したDSNをリポジトリ外の所有者専用の一時的な引継ぎファイルへ複製する
- Go・Gitの確認処理、#310、#255にDSNの内容・元の秘密情報ファイルのパス・引継ぎ用パスを渡さない
- #308の`go run`・コンパイラ・リンカーにはDSNの内容を渡さず、一時的な引継ぎ用パスだけを渡す
- #308の実行環境が引継ぎファイルからDSNを読み、ソースの確認処理より前にDSN・ファイルパスの環境情報を消去する
- DSNの内容・元の秘密情報ファイルのパス・引継ぎ用パスを検証根拠に保存しない

### child Evidence artifact store

- リポジトリ外に保存すること
- 所有者だけがアクセスできるディレクトリ
- 子処理のファイルは新規作成
- 既存のファイルを上書きしない
- 子処理の標準出力のバイト列をそのまま保存

## actual Oracle

#214・#371・#376は、削除したり完了したものとして扱ったりしません。

実際の物理的な事実・来歴・実効的な読取り専用の制約・具体的なドライバーとの連携・実際のデータの読取りが未完了なら、そのまま未完了として扱います。

判定段階Bの合格に必要な第四の検証経路とはしません。

引継ぎでは、次のように扱います。

- 実際のOracleで得た十分な検証根拠があれば、検証済みの外部資産の根拠として追加参照
- 未取得・実際の読取りが未完了・版Rとの結び付きが不足する場合は`unresolved`

最初のウェブ実行先側で、Oracleの物理的な事実を推測して補いません。

## candidate revision

必須の検証根拠は一つの版Rに固定します。

```text
#310.repositoryRevision = R
#255.repositoryRevision = R
#308.repositoryRevision = R
final.repositoryRevision = R
```

版Rを固定する作業領域はリポジトリ全体を取得し、追跡対象・追跡対象外・無視対象のローカル状態がなく、Gitの索引で追跡対象の項目が通常の`H`だけであることを要求します。

異なる版で得た成功の結果を寄せ集めません。

ソースの修正が必要になった場合は一連の検証を停止し、修正をコミットして新しい版Rを選びます。その版で三つの検証経路と最終確認を初めからやり直します。

mainが更新されても、候補版Rを暗黙のうちに最新mainへ切り替えません。

## Ready条件

判定段階BをReadyとするには、少なくとも次のすべてを検証根拠から説明できる必要があります。

- [x] #310 actual EvidenceがRで成立 — `deterministic-repository.json`、`repositoryRevision = e5231839a5dcc62cce1181a2731fcb7c70d16736`、`passed = true`
- [x] #255 actual EvidenceがRで成立 — `generated-artifact-reproducibility.json`、同revision、`passed = true`
- [x] #308 actual EvidenceがRで成立 — `real-postgres-ephemeral-fixture.json`、同revision、`passed = true`
- [x] #387 final convergenceがRで成立 — `evidenceNature = gate-b-three-lane-final-convergence`、`repositoryRevision = e5231839a5dcc62cce1181a2731fcb7c70d16736`
- [x] `allLanesSameRevision = true`
- [x] `childArtifactsRetained = true`
- [x] 三つのsame-run child exact artifactを保存済み（`deterministic-repository.json` / `generated-artifact-reproducibility.json` / `real-postgres-ephemeral-fixture.json`、Repository外owner-only directoryへ保存）
- [x] 各artifact fingerprintがfinal summaryと一致（`evidenceArtifactFingerprint`と`reference-gate-b-readiness-artifact-audit`のlane別fingerprintが一致）
- [x] 独立artifact auditをsame-run artifactsへ実行し `artifactAuditPassed = true` — `finalFingerprint = sha256:3bfcdc459f9e3dd77019e1b9ae368a64bad760f65917c44e1d7d533b6a1f5f97`
- [x] #310 Workspace identity / #310・#255 producer commandが正規contractと一致（`reference-gate-b-readiness-artifact-audit`のcommand-exact-binding checkが`passed = true`）
- [x] child詳細から各lane固有安全条件を再確認できる（`fixtureDatabaseSafety.guardPassed = true`、`fixtureCleanupSafety.verified = true`、`contractQueryTarget = sales_core/sales_core.order_line/find_by_order`ほか）
- [x] final run終了時もR / clean source/index state — `postRunRepository.revisionStable = true` / `workingTreeClean = true`
- [x] Go実行環境fail-closed条件成立 — `goEnvironment.externalWorkspaceActive = false` / `goFlagsEmpty = true` / `goEnvDisabled = true`
- [x] Repository root / source observation条件成立 — Gate B Evidence専用full checkout、`git status --short --untracked-files=all --ignored=matching`空、`git ls-files -v`全`H`を実行前後で確認
- [x] PostgreSQL fixture前後安全性成立 — fixture前`userRelationsBeforeFixture = 0` / `eventTriggersBeforeFixture = 0`、cleanup後`userRelationsAfterCleanup = 0` / `eventTriggersAfterCleanup = 0` / `fixtureObjectAbsentAfterCleanup = true`
- [x] DSN / credential / secret /実Database名を通常Evidenceへ保存していない（`reference-gate-b-readiness-artifact-audit`のsensitive-field拒否checkが`passed = true`、目視でもchild JSONにDSN/secret文字列なしを確認）
- [x] Evidence natureを相互代替していない — 各lane `evidenceNature`が`deterministic-repository` / `generated-artifact-reproducibility` / `real-postgres-ephemeral-fixture`のまま
- [x] Business Result / Execution Error / Observation境界維持 — `#308` child JSONで`executionErrorObserved = true`（missing-table Execution Error）とnormalなrow取得が別fieldとして区別されている
- [x] external未観測事項を推測していない — #310/#255/#308はいずれもsynthetic/PostgreSQL fixture Evidenceのままで、actual Oracleへ格上げしていない
- [x] actual Oracle未完了範囲をHandover `unresolved`へ列挙可能 — 下記「actual Oracle」節、Handover Baseline記録にも反映
- [x] 監査結果をRへ固定可能 — 本ファイルの判定はすべて`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`に紐づく

## 次の行動

判定段階BのReady確定後、引継ぎ基準版`B`をリポジトリの版`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`に固定しました。記録は[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](GATE_B_TO_C_HANDOVER_BASELINE_B.md)にあります。

```text
actual full checkout preflight
        ↓
#310 / #255 / #308 pre-check
        ↓ source変更があればcommit
candidate R固定
        ↓
#387 final convergence
        ↓
same-run child exact artifact independent audit
        ↓
Gate B Readiness Audit for R  ← 完了（本ファイル、2026-09-19、Ready）
        ↓ Ready only
Handover Baseline B / revision R  ← 完了（GATE_B_TO_C_HANDOVER_BASELINE_B.md）
        ↓
First Web Target  ← 次のIssue（`ISSUE-000007` / #70）
```

## 非目標

- 実際のOracleを第四の必須検証経路に戻すこと
- 最終確認の実行器の成功だけで自動的にReadyを宣言すること
- 成果物の監査の成功だけで自動的にReadyを宣言すること
- 同じ版Rの別の実行を、同一実行内の成果物として読み替えること
- GitHub Actionsの成功を製品の検証根拠として格上げすること
- 最初のウェブ実行先側で不足している意味を推測して補うこと