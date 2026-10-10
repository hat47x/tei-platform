# 三視点の意味参照と変更影響 — #1120 初期契約

状態: **候補（Draft PR、未採用・未統合）**。対象は `internal/modelref` の独立した小契約であり、Accepted Canonical World のschemaを変更しない。

## 目的と責務の境界

業務（task / actor / event）、データ（entity / attribute / relation）、機能（function / form / page / report）を、表示ラベル・Sourceファイル名・物理表名・生成IDを介さず結び付ける。Core側が知るのは **scope付きの種類付き論理参照** と **参照の検証・変更影響** だけである。モデル種別、edgeの意味、循環禁止範囲、表示ラベル・物理名の影響条件はPlugin（またはReferenceに渡すRule）が宣言する。

既存機構との分担は次のとおり。

| 機構 | 既存の正本・責務 | 本候補に追加する最小機構 |
| --- | --- | --- |
| `internal/subject` | ファイル位置から独立した `kind / id` | `scope` と型検査付き `Ref` |
| `internal/core` | 既存Canonical ID文法・Runtime等 | 文法を書き換えず、ユーザー識別子のハイフンを含む参照を局所的に表現 |
| `internal/sourceindex` | Artifact digest、source locator、subject逆引き | source tokenはprovenanceとして保持し、Identityへ昇格させない |
| `internal/index` / `knowledgecontext` | 現行WorldからのDerived dependenciesと探索 | edge種別ごとの解決診断、diffからの逆方向impact |
| #950 / #1031 | legacy read-only transcription / derived evidence | 観測結果をAccepted meaningとしてインポートしない |
| #267 / #958 | ProposalとShared Acceptance、文脈・確認履歴 | 確認権限や受理履歴を新しいGraphに移管しない |
| #1019 / #951 | Coreの拡張圧力とPlugin Capability | 新規Model kindごとに `ir.World` の固定列挙を増やさない |

## v0 JSON契約

`Graph.schema = "tei.modelref.v0"`。JSONの構造はGoの `Graph` / `Node` / `Edge` / `Rule` / `Transition` を正とする。概念例:

```json
{
  "schema": "tei.modelref.v0",
  "nodes": [
    {"key": {"scope": "orders-v1", "subject": {"kind": "data", "id": "order-line"}}, "version": "1", "displayName": "注文明細"},
    {"key": {"scope": "orders-v1", "subject": {"kind": "function", "id": "submit-order"}}, "version": "2", "displayName": "注文登録"},
    {"key": {"scope": "orders-v1", "subject": {"kind": "task", "id": "confirm-order"}}, "version": "1"}
  ],
  "edges": [
    {"from": {"scope": "orders-v1", "subject": {"kind": "function", "id": "submit-order"}}, "to": {"scope": "orders-v1", "subject": {"kind": "data", "id": "order-line"}, "version": "1"}, "relation": "reads"},
    {"from": {"scope": "orders-v1", "subject": {"kind": "task", "id": "confirm-order"}}, "to": {"scope": "orders-v1", "subject": {"kind": "function", "id": "submit-order"}, "version": "2"}, "relation": "triggers"}
  ]
}
```

- **Key**: `scope + subject(kind,id)` が論理Identity。同一scope・kind・idの重複は不正。実装上のSource path、line / offset、primary key、生成ランタイムIDと交換しない。scopeはnamespace / application等の所属を明示する契約であり、runtime sessionとは別。
- **Ref**: `Scope` を明示するのが原則。空欄は全scopeで一意なら解決できるが、複数一致した場合は `ambiguous-reference`。 `Version` は厳密一致条件で、曖昧なsemver互換性の推測をしない。
- **Node**: `DisplayName` はラベル、`PhysicalName` はこのReferenceでの任意の物理名ヒント。後者はN:M Database Mapping、DDL、実行上のauthorityを表さない。対象別の複数物理対応は #1126 等のPlugin/Mapping契約に委譲する。
- **Evidence**: `SourceToken` / `ArtifactFingerprint` はlegacy sourceに戻るための手掛かり。`RuleFingerprint` は派生説明の同定子。 `Authority` / `Context` は現状のReference候補であり、#958のprovenance、確認イベント、as-of、文脈の完全表現ではない。 `accepted` と書いてもShared Acceptanceの実行証跡にはならない。
- **Rule**: `relation / fromKind / toKind` はPlugin側で追加可能。循環禁止が必要な依存だけ `acyclicGroup` を宣言する。 `reads` の存在から `writes`、認可、actual effect、トランザクション境界を推定しない。

## 変更と移行の意味

- Source **file rename / path move**: Keyは不変。Source Indexのlocatorのみ変わる。これは意味変更とは限らない。
- **display rename**: `display-renamed`。 `displayAffects` のあるedgeに限り一次消費者へ波及する。たとえば単なるDataのreadはラベルへの依存を証明しない。
- **physical rename**: `physical-renamed`。 `physicalAffects` のあるedgeにのみ一次波及する。物理適用可否はDB Migration Gateで別途判定する。
- **revision / reference / evidence change**: 個別の差分種別。意味参照の変更をNode label変更へ吸収しない。Versionや参照先を変えた結果の不整合を検出できる。
- **logical ID rename**: 同一Keyのrenameとは扱わない。`removed + added` を検出し、明示的な `rekey` を必要とする。旧IDを自動alias化しない。参照を更新していなければ dangling reference として残る。
- **split / merge**: `Transition{kind: split|merge, from, to}` で基数と前後の存在を検査する。関数の成功は意味継承・参照移行・受理権限の証明ではない。Project間の再利用・namespace移動も暗黙同一視せず、移行と承認を要求する。
- **削除**: 旧Graphで参照するConsumerを `Impact` によって逆引きする。新Graphの参照解決で欠落を通知する。影響範囲はdependency graph上の候補であり、business effectの証拠ではない。

移行表現例（これ自体では書き込まない）:

```json
{
  "kind": "rekey",
  "from": [{"scope":"orders-v1","subject":{"kind":"data","id":"order-line"}}],
  "to":   [{"scope":"orders-v1","subject":{"kind":"data","id":"order-detail"}}]
}
```

実際の受理は#267のShared Acceptanceが **base revision確認 → preview/diagnostic → human decision → authoritative source更新 → reload / conformance** を行う。旧source token・候補identity・承認前後の紐付けを捨てない。

## 診断・試験・互換性

`Validate` は `unsupported-schema`, `invalid/duplicate-node-key`, `missing/ambiguous-reference`, `wrong-kind`, `version-mismatch`, `invalid-edge-type`, `cycle`, `unbound-source-evidence`, `unbound-derivation-evidence` 等を返す。循環はRuleが指定した依存群に限るため、業務ループを一律禁止しない。

`Diff` はlabel変更、物理名変更、version変更、参照追加・削除、provenance変更、node追加・削除を分離。`Impact` は変更前Graphを使って削除済み参照も追跡する。出力は安定sortされるが、まだ包括的なsemantic diff、rename自動推定、cycle path説明ではない。

Focused tests（ローカル独立再現）:
```sh
go test -race -count=1 ./internal/modelref
go vet ./internal/modelref
```

互換性: `tei.modelref.v0` は候補versionであり他バージョンは明示拒否。既存 `ir.World`, `sourceindex.Index`, Shared Acceptance、Runtimeには配線していない。将来の受理時は旧Graphを黙って書換えず明示schema migrationを通す。

## 未完了のGate（次の作業）

1. 本Branchの**trusted full checkout**でfocused testsおよび `tools/repository-health-check.sh` を実施し、実行前後のclean状態・Go toolchain・exact head/mainを記録する。今回はisolated reproductionのみでRepository Gateは未検証。
2. #1124/#1125/#1126の既存Canonical Sourceを入力としてAdapterから本Graphに**read-only**に投影し、現行 `index/knowledgecontext` と同じ参照先を確認する。GraphをWorldの第二正本にしない。
3. 差分規則のnegative pressure: edge単位のEvidence変更、複数候補があるversion選択、Project越境、cycleのpathと根拠、物理mappingの複数対応。現実のSource差分に基づく最小の破壊的変更テストを追加する。
4. 公開前にlegacy source token・文脈ID・機微locatorの出力制約を #941 / #1031へ照合する。Referenceの小さな擬似IDをprivateな実資産のIDへ置換しない。
5. Publisher・Readerが `Rule` を同じversion / fingerprintで評価する方法を#1019/#951へ返し、Coreに固定種別を追加しない。

この段階では **#1120全体の完了、Accepted model連携、UI双方向編集、DB migration、actual Runtime authority** を主張しない。

## R1 — 実装圧力試験とSource Index再接続（2026-10-10）

初期の `Diff / Impact` のうち、次の誤判定を回帰テストによって修正した。

1. **派生edgeの根拠更新を無視しない**: 同じ `from/to/relation` でも、source token、artifact fingerprint、derivation rule fingerprint等の証拠が変わった場合、`reference-evidence-changed` を返す。単なるEvidence集合の順序変更は差分にしない。ただしrule自体の一般的な変更・semantic applicabilityの再証明にはまだ対応しない。
2. **表示名・物理名による波及を拡大しすぎない**: `displayAffects/physicalAffects` の要件を**すべての伝播経路**で評価する。一段目の表示依存が存在しても、次段が通常の `triggers/reads` だけであれば、その先を表示変更の影響とは断定しない。
3. **曖昧・未登録の参照を経由して影響を発明しない**: 有効な型付きedgeだけを逆隣接索引に入れる。欠落・version不一致・scope曖昧・未知relationからconsumerを生成しない。参照の誤り自体はまず `Validate` の診断で明示する。局所的な大規模Graphでも、毎edgeを毎回全Nodeから探索する設計を避ける。

### read-only Source Navigation bridge

`internal/modelref/sourcebridge` を新設し、既存の `sourceindex.Index` からscope付き `modelref.Key` へ、検証済みArtifact fingerprintとRangeを持つ**読み取り専用のNavigation**を返す。

- 呼び出し元のreaderに対し、既存の `sourceindex.Index.Validate` でファイルdigest・位置メタデータの整合を確認する。Source Indexの実装・既存Definitionの識別子を変更しない。
- `Source Index` にscopeフィールドが無い現段階では、異なるscopeの同一 `kind/id` を任意の一方のSourceに接続しない。`subject-scope-ambiguous` を返す。これは将来のscope-aware Indexの必要性を示す具体的な負例である。
- indexに `RoleDerived` だけがある場合は `derived-only-source` として止める。単なるLocator検出を「編集可能な正本」やShared Acceptance済みの意味に読み替えない。
- Snapshotのdigestが古い、source位置が破損した、元のDefinition/Authoring参照が見つからない場合は `unresolved`。欠落した候補を自動生成しない。
- 返す `located` は**位置の観測結果**であって、特定snapshotが正式なAccepted Sourceであるとの認定ではない。Accept/reloadの成否・Concurrent Edit保護・Source更新は引き続き#267/#1124の既存境界で扱う。

このBridgeは `modelref -> sourceindex` の逆依存を作らない独立したconsumer Packageであり、`ir.World` / `index.Build` / DSL / Runtimeを改修しない。#1124側のAuthoring Projectionは、同一revisionのWorld/Source Indexを得た場合に限り、ここからSource locatorを候補表示できる。

追加のfocused suite:

```sh
go test -race -count=1 ./internal/modelref/...
go vet ./internal/modelref/...
```

**実行範囲:** Go 1.23.2の隔離ミラーで `modelref` と `sourcebridge` をテストした。後者の隔離環境では `sourceindex` の最小同型shimを使用したため、trusted full checkoutでの実Package間試験と `tools/repository-health-check.sh` の資格確認は依然として未実施である。合格証跡を混同しない。現在のStageはReference Contract + source navigation adapter candidateであり、Accepted Worldの三視点実データ接続やStudio E2Eが成立したわけではない。
