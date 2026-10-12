# 三視点の意味参照と変更影響 — #1120 初期契約

状態: **参照契約の実装・試験をmainに導入済み**。受理済みモデルの第二の正本ではなく、読み取り専用の候補として扱う。以下の初期設計と、その後の追加検証を区別する。

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

## R2 — 現行Derived Indexからの型付き投影（2026-10-10）

`internal/modelref/derivedbridge.FromIndex(index.Derived, scope)` を追加した。現行の `index.Build(World)` が作る `SemanticElements`、`SemanticDependencies`、`DeclaredDependencies`、`ApplicationPatterns`、`Queries` を**読み取り専用**で使う。既存のWorldとIndexに新しいschema、Dependency、Write-pathを追加しない。

- 元のsemantic definition / Application PatternのIDを保持し、`subject.Kind` を既存Source Indexの種類に合わせて `semantic-data`、`semantic-relation`、`application-pattern` 等へ変換する。
- `application-pattern -> semantic-data`、`semantic-relation -> semantic-data`、`application -> application-pattern` など、**既存Indexに明記されたedgeのみ** を投影する。一般的な `semantic-element` 参照は候補が一意に解決できる場合だけ具体化する。
- 導出Index内に表示されないunknown target、duplicate semantic definition、必要なscope欠落は `Unresolved` に残す。Sourceで確認できない対象、Function-level CRUDからのBusiness Action実更新、Accepted Business Procedureは生成しない。
- Relation Kindに対応する `Rule` はDerived Indexからの**観測用型組**として返す（cycle policyや実行/認可権限を与えない）。`Graph` のversion / Evidence bindingは空のままとし、原Source fingerprintは `sourcebridge` と既存Source Indexに委ねる。
- `index.Derived` だけでは既存Canonicalモデルのrevision・Acceptance履歴・original Sourceの全意味が確定しない。正式なconsumerはWorldとIndexを同一base revisionから取得し、その整合をShared Acceptanceで確認すること。

Referenceの最小試験では、受注/明細RelationとApplication Patternの機能/データ参照を5 node / 3 edgeとして再投影する正例と、未知のData参照およびscope欠落の負例を用意した。これは**実Workspaceで三視点モデルを一往復編集した証跡ではない**。Business Procedureが現在のCanonical Worldへ未収容という不足を残したまま、非対応事項を隠さず#1125と#1124へ渡す。

`go test -race -count=1 ./internal/modelref/...` と `go vet ./internal/modelref/...` をGo 1.23.2の**隔離ミラー**で実行してPASS。`derivedbridge` は現在、そのミラーでは `index.Derived` の最小同型shimに対してテストしている。GitHub上のSource Bridge/Derived BridgeのGoファイルと、対応する隔離ミラー内ファイル4本のGit blob SHA一致を確認したが、`modelref/model.go` とその依存はexact-headのfull checkoutで再度検証する必要がある。Repository Healthは未実施であり、PRをDraftに維持する。

## R3 — 実Referenceの横断テストとNavigationの費用境界（2026-10-10）

### 同じWorkspaceからData・Functionの意味参照とSourceへ戻る

`internal/workspace/modelref_cross_view_integration_test.go` を追加した。対象は公開Repository内の既存 `reference/workspace` である。実際の `workspace.Load` と `workspace.BuildSourceIndex`、`index.Build(loaded.World)` から本PRのBridgeを呼ぶ。

- `order_entry.customer_reference` （Application Pattern）→ `sales.order.customer` （Semantic Relation）→ `customer.customer` （Semantic Data）、`order_entry` → Patternの型付きedgeが、そのWorld由来で存在すること。
- PatternのSourceが `applications/order-entry/patterns/customer-reference.tei`、RelationのSourceが `semantic/sales/order.tei` として、**検証済みSource Index**から取得されること。参照先をファイル名から逆算しない。
- Relationを変更したという候補に対し、PatternとApplicationを上流のReview対象として逆引きできること。
- Acceptance済みのBusiness Stepを本投影が勝手に生成しないこと。現在のWorldにないBusinessの意味・actual effectは別レーンで検証する。

これは**実Referenceを対象とするテストコードの追加**であり、現在のPR headではまだtrusted full checkoutで実行していない。「期待されたAssertionを記述した」ことと「AssertionがPASSした」ことは分けて扱う。

### Source Indexを1回検証し、複数視点で使い回す

初期 `sourcebridge.Locate(graph, index, key, read)` は、Source Index全件を検証し直すため、呼出し数に比例して元ファイルの読み取りが増加する。表・ER・Functionを多数の行・要素へ投影する際に無駄なI/Oが発生する。

- `sourcebridge.NewResolver(graph, index, read)` は、indexのArtifact fingerprint / Rangeを一度だけ検証して索引を構築する。
- `resolver.Locate(key)` は**検証済みsnapshotのメモリ上の逆引きのみ**でナビゲーションを返す。同一snapshotの複数要素についてSourceを読み直さない。
- 既存の単発API `sourcebridge.Locate(...)` もResolverへ一本化し、失敗時は従来どおり `unresolved` を返す。
- 同一kind/IDが複数scopeに現れた場合と、同一subjectにDefinition（またはAuthoring）のanchorが複数ある場合は、ひとつを恣意的に選ばない。未知のGraph schemaも拒否する。
- `NewResolver` 成功はfile systemの更新ロックを意味しない。**World・Graph・Source Index・Readerが同一Workspace revision由来**であること、同一revisionの不変snapshotであることはcaller側の要求であり、受理段階では既存Shared Acceptanceがcurrent revisionを再確認する。

性能テストは1000種類の論理subjectに対して一度だけIndexを検証し、以後の1000回のLocateでreaderの追加呼出しが0であることを検査する。実運用規模の壁時計時間・メモリ使用量は未計測。さらに `derivedbridge` 側のgeneric semantic-element解決もID→Kind索引へ変更し、追加Pluginがもたらす多数のSemantic Nodeに対して全件走査を繰り返さないようにした。

### 未完了のEvidenceと次のGate

1. trusted full checkoutで `go test -race -count=1 ./internal/modelref/... ./internal/workspace` と `go vet ./internal/modelref/...` をactual `index`/`sourceindex`で実行し、失敗時は本PRで修正する。
2. exact PR headに `tools/repository-health-check.sh` を実行し、Go toolchain、pre/post clean、current mainとの比較を記録する。非実施のままPRをDraftから変更しない。
3. #1124は `derivedbridge` と `sourcebridge.NewResolver` の**read-only consumer**になり得るが、Accepted Business Procedureの不足は #1125、ER・関数従属性の編纂は#1126、Actual Function CUDは#1132に保持する。
4. 同一revisionのSnapshot Bind / concurrent edit / stale Proposal / shared Acceptance前後のProofは、このReferenceだけで成立していない。生成物をAccepted meaningとして保存する機構も追加していない。

## R4 — Graph Validationの拡張性・性能・根拠の再評価（2026-10-10）

### 誤用・性能劣化を防ぐ修正

現行の`Validate`は、元々それぞれのedgeについて全Nodeを走査して参照解決し、すべてのPlugin Ruleを繰り返し走査していた。多数のData/Function/Business nodeを表示する製品モデラーでは、この繰返しが編集時の検査時間を押し上げる。

- `internal/modelref/lookup.go`: 検査対象となる一つのimmutable graph snapshotに対し、Nodeをlogical IDで索引化する。種類・Scope・Versionは解決時に厳密に判定し、種類不一致・未解決・複数候補を従来の診断に残す。`Resolve`の単発APIも同じ検索意味論を使う。
- `Validate`: Plugin側Ruleを`(relation, fromKind, toKind)`で索引化する。Rule未定義・空値、同じ型組に矛盾するRuleがある場合は`invalid-rule` / `conflicting-rule`を出す。先に登録されたRuleを黙って「正式」とみなさず、診断を消費する側がfail-closedに止める必要がある。
- 以前の実装では参照元が不正・参照先が欠落したedgeについて、そこで`continue`するためEvidence不足の診断を落としていた。Edge provenanceを**参照の成否にかかわらず**検査することで、欠落参照と不完全な派生根拠を同時に示せる。
- `Diff`のNode Evidenceも順序非依存の比較とし、source token一覧の並べ替えだけで不要な再レビューを発生させない。実際のfingerprint変更は`evidence-changed`のまま。

### Extension Policyのfingerprint

`RuleFingerprint([]Rule)`を追加。Ruleの集合を一定の規約（`tei.modelref.rules.v0`）で正規化してSHA-256化し、登録順や同一Ruleの重複による差を除外する。Relation型、循環禁止group、表示名・物理名への依存条件が変わればfingerprintも変わる。これによりGraphのNode/Edgeが不変でも、**解釈するExtensionのRule変更で過去のValidate/Impact結果がstaleになる**ことを検出する基礎ができた。

digest自体は署名・確定済みの規則・読み込んだPlugin sourceの検証ではない。利用側はPlugin ID/version/source digest、RuleFingerprint、World/Source snapshot revisionを同時に保持し、変化時に必ず再評価する。既存Shared Acceptanceがその文脈を正式に束縛する前に、digest一致だけを根拠に受理したことにはしない。

### 追加した検証と数値の位置付け

- 4096件のtyped edge・8192件のNodeを持つ擬似Graphの正例・欠落参照の負例を追加。
- `BenchmarkValidate4096References`を追加。Go 1.23.2 / Linux amd64 の独立ミラーで`-benchtime=5x`実行時は約27ms/op（5回）だった。ごく小さなベンチマーク実行なので、旧実装に対する速度倍率や実Referenceの性能を主張しない。
- Ruleの登録順の交換、同一Ruleの重複、Display依存とcycle policyの変更を反映するfingerprint、Plugin Rule衝突、Evidence不足と欠落参照の重複診断、Node Evidence並べ替えの負例を追加。
- 隔離ミラーの`go test -race -count=1 ./internal/modelref/...`、`go vet ./internal/modelref/...`はPASS。ただしミラーは一部のコードを独立に再現しており、現在のPR exact headの全Go package一致とRepository Healthの合格を証明しない。

### 残る設計・資格確認上の注意

`RuleFingerprint`はPlugin catalogのlock / Capability解決、署名検証、明示的なAcceptance決定とは別責務。破壊的Rule変更時のMigration戦略は#1019/#951と合意する。Graphの意味をCoreに固定しない原則を守る。

`Lookup`の索引によって典型的な参照解決の反復全走査を抑えたが、**極端に同じIDが多数のScope/Kindに存在するGraph**での一意性診断、cycle path説明、Property-based test、realistic 1万件規模のEditor応答測定は未完了。実装数やsynthetic PASSは三視点の実製品Acceptanceそのものではない。

### R4補足 — Source Snapshotの読み取りコスト

`sourcebridge.NewResolver`では、`Index.Validate`が同じArtifactを複数のDefinition/Authoring anchorから検査する際、1回の構築内で読み取ったbytesをパス別にコピーして再利用する。これによりSourceファイル数に応じた実I/Oに抑え、各anchorごとのファイル再読込を避ける。既存のSource Indexによるfingerprint/Range整合性は省略しない。

追加したテストでは1 Artifactと1 Definition Referenceについて、構築時のreader呼出しが**厳密に1回**となり、以後の1000件Navigationは0回であることを確認する（隔離ミラーでPASS）。なお一度のキャッシュ検証は時点間のAtomic Snapshotや同時編集排除を保証しない。呼出し元が同じimmutable Source revisionを渡し、Acceptの直前にShared Acceptanceで競合を再検査することが必須である。

## R5 — rename・split・mergeの参照網羅性をread-onlyで審査（2026-10-10）

最初の`ValidateTransition`は旧・新IDの存在と基数を調べるだけだったため、論理IDの変更時に**参照するFunction/Business要素の更新が省略されたままでも**、移行自体を「有効な形」と評価できる問題があった。

`internal/modelref/migration_review.go` に`ReviewMigration(before, after, transition, retargets, rules, baselineRevision)`を追加した。正本への書き込みもalias追加も行わない。元・新Graphは呼び出し元が準備する候補であり、個々の参照変更を`EdgeRetarget{from,relation,old,new}`として明示する。

### 受入前の確認内容

- `ValidateTransition`だけでなく、変更前後両方の`Validate`を実行する。不正な参照を「移行中だから」と認めない。
- 旧IDを参照する**すべてのincoming edge**について、元の`from/relation/oldRef`と新しい`from/relation/newRef`が一致する明示的な`EdgeRetarget`を要求する。
- 同じedgeへの重複Retarget、存在しない移行先、意図しないedge削除・追加、移行対象以外のNode変更は不整合とする。array offsetによる参照の同定は行わない。
- splitでは個々のConsumerがどちらの新IDへ向かうかを推定しない。mergeでは二つの異なる旧IDから同じ新IDへ集約しても、Consumerごとに承認候補の再束縛が必要。
- **移行元Nodeが参照元でもある場合（outgoing edge）**、現契約ではそのSource自身の移動／分割を自動処理せず`outgoing-reference-requires-own-review`で止める。別の編集レーンが明示的に意味継承を審査する。
- baseline revisionが空なら診断する。ただし文字列があることはそのRevisionを検証できたことではない。
- `RuleFingerprint`、`Diff`、`Impact`、根拠と同じ構造の`MigrationReview`を返す。`Consistent=true`は**与えられた候補内での参照整合**であり、human confirmation / Shared Acceptance / 本番データ移行の完了ではない。

### 参照変更の例（擬似受注明細）

```json
{
  "transition": {
    "kind": "rekey",
    "from": [{"scope":"orders-v1","subject":{"kind":"data","id":"order-line"}}],
    "to": [{"scope":"orders-v1","subject":{"kind":"data","id":"order-detail"}}]
  },
  "retargets": [{
    "from": {"scope":"orders-v1","subject":{"kind":"function","id":"submit-order"}},
    "relation": "reads",
    "old": {"scope":"orders-v1","subject":{"kind":"data","id":"order-line"},"version":"1"},
    "new": {"scope":"orders-v1","subject":{"kind":"data","id":"order-detail"},"version":"1"}
  }]
}
```

上記の例だけでは変更先Graphを自動生成しない。実際には同一snapshotのold/new Graph候補とPolicy一式、baseRevisionが必要になる。Studio/AI/CLIが生成した変更案も#267のShared Acceptanceがrevision再検証・明示受理・Source更新・reload・Conformanceを実行する必要がある。

### 回帰試験

`migration_review_test.go`と`migration_more_test.go`に次を実装した。

- DataのID変更に伴うFunction参照の明示Retargetと、Business task / Formまでの影響候補。
- 欠落・重複Retarget、baseline欠落、意図しないFunction→Formの削除を拒否。
- 別scopeへの論理要素移動は同一IDでも明示Retargetを要求。
- 複数の旧Dataを1つにmergeする際、各Functionの別々の参照移行を要求。
- 移行対象以外の表示ラベルの変更は、本処理だけで認可せず別のProposalへ戻す。
- preview実行で渡された旧GraphのJSON内容が変化しないことを検査。

Go 1.23.2の隔離作業領域で、これらを含む`go test -race -count=1 ./internal/modelref/...`と`go vet ./internal/modelref/...`はPASS。少なくとも追加した`migration_review.go`および`migration_review_test.go`のGit blob SHAはリモートの追加ファイルと一致した。**これは既存`index/sourceindex`のstubを含む作業領域での試験であり、trusted full checkout exact-headのRepository Health Gateを満たした証拠ではない。**

この変更は#1120のidentity migrationの参照審査を具体化するもので、既存Source書き込みや他レーンのauthoring機能を再実装しない。

## R6 — 移行レビューにおける根拠の置換事故を防ぐ（2026-10-10）

`ReviewMigration`の敵対的な再検討で、`from / relation / to`の置換先が正しくても、**edgeのEvidenceが別の根拠に差し替わる**可能性が残っていた。従来は`reference-evidence-changed`を検出する`Diff`があっても、移行レビュー側はそれを十分に拘束していなかった。

追加した検査は次の二つである。

1. **移行対象edge**: `EdgeRetarget`で指定された旧・新edgeのEvidence集合（出典token、Artifact fingerprint、Rule fingerprint、文脈・種別）が一致しないと`retarget-evidence-changed`。別の根拠へ変更する必要があるなら、その変更を隠さず、別ProposalとShared Acceptanceで審査する。
2. **移行対象でないedge**: `from / relation / to`が不変でもEvidence集合が変わった場合は`unreviewed-reference-evidence`。再キー設定案の「ついで」に出典・解釈が入れ替わるのを防ぐ。

どちらもEvidenceの配列順だけは差分とみなさない。実装は`migration_review.go`、負例は`migration_more_test.go`の`TestMigrationCannotSilentlyReplaceReferenceEvidence`に記録した。

**検証**: `/mnt/data/tei1120`の隔離Go 1.23.2環境で`go test -race -count=1 ./internal/modelref/...`（3 Package）、`go vet ./internal/modelref/...`が成功し、`gofmt -l internal/modelref`は空だった。変更した2ファイルの`git hash-object`はGitHubに書き込んだblob SHAと一致した。ただし隔離環境には`sourceindex` / `index`の代替実装があり、Repository全体のexact-head Git checkoutではない。

**統合停止条件**: `AGENTS.md`の規則に従い、trusted full checkoutのexact headとcurrent mainを固定した`tools/repository-health-check.sh`の**全項目PASS**、前後のclean state、必要なWorkspace integration testの証拠が揃うまでは、Draftを解除せずmainへ統合しない。実際のSourceの受理やDecision履歴の追跡が未成立なら、#1120の完了判定も行わない。


## R4 — 型付き参照の読取り用直接追跡（2026-10-10、別Draft候補）

`modelref.TraceDirect(graph, rules, key, options)` を、既存の `Graph` / `Rule` / `Resolve` を消費する操作として設ける。新しいグラフを組み立てる処理ではない。初期範囲は**直接の宣言済み参照**に限り、業務上の依存や実行時の作用を名前の類似から補完しない。

- 対象は `scope + kind + id` の完全一致。対象が存在しない場合と同一キーの多重定義を分けて失敗させる。
- `direction` は `outgoing / incoming / both`。scope未指定の参照は、全scopeで一意に解決できるときだけ確定する。曖昧・欠落・version不一致・未知edge種別・規則衝突・不正な根拠は `unresolved` とその診断を返し、正しい参照に見せかけない。
- 行には宣言済み `Edge` と元の `Evidence` を保持する。解決できたときだけ `resolvedTo` を付ける。結果全体も `authority=candidate-read-only` であり、Shared Acceptance済みとは解釈しない。
- `mode=summary` は合計件数と未解決件数を返し、明細行を省く。`mode=full` は指定がなければ**全件を返す**。暗黙の走査上限・文字数による切捨てを設けない。利用者が `limit` を指定したときだけ `nextOffset` を明示してページを分割する。
- `snapshotFingerprint` は入力GraphのJSONと拡張規則 `RuleFingerprint` を組み合わせた内容ハッシュ。`expectedFingerprint` を付けてページを続ける場合は一致を要求し、食い違いを `STALE_SNAPSHOT` として拒否する。これは**書込みの楽観的排他、読取のトランザクション整合性、Accepted revisionではない**。Graphの登録順序が変わるだけでもfingerprintは変わり得るが、その場合も不安全な継続より再取得を優先する。
- 行の順序は種類付きキー・関係・根拠に基づき安定化する。Evidenceは意味上順序を持たないため出力用コピーを正規化し、入力Graphは変更しない。

実装の正本は `internal/modelref/trace.go`。同一のReference Workspaceから `index.Build → derivedbridge.FromIndex` で得た候補Graphを用い、`application-pattern → semantic-relation → semantic-data` の出典付き直接参照を確かめる統合テストも**追加した（未実行）**。この投影に含まれないBusiness Procedure、Acceptedな業務フロー、実行時CRUDの関係を生成したとは扱わない。

### CLIへの接続順序

C1 [#1165](https://github.com/hat47x/tei-platform/issues/1165) の `tei trace` は、`#1139` の本Modelref契約と、`#1169` のCLI inspect/operations境界が双方とも検証・統合された後に接続する。Web/Desktopも同じ型付き操作を呼び、argvやJSONの形式はAdapterが所有する。`Source Index` の元定義への案内は `sourcebridge` の検証済みNavigationを使い、Edgeの存在だけで書込み許可を与えない。

スコープ・Graph版・Policy版、Host側の情報開示・認可、秘匿Evidenceの伏字化、ページ取得間の情報鮮度については、実アダプターでの追加受入検証を必要とする。今回のGoのテストコード追加は実行成功の証明ではない。統合には、対象のexact headにおけるリポジトリ全体のHealth Checkが引き続き必要となる。

## R7 — 変更影響の「空」と「未解決」を区別する（2026-10-12）

PR #1139がmainへ統合された後、現行`modelref.Impact`を再点検したところ、`Validate`が`conflicting-rule`を検出する一方、`Impact`は競合する二つの拡張規則を個別に採用して影響先候補を生成できることが判明した。たとえば`function --reads--> data`の表示名依存がfalseとtrueで競合した場合、true側の規則を選び、表示名変更の波及先を誤って提示し得る。

今回の対応は次のとおり。

- `Impact`の逆向き索引構築では、`(relation, fromKind, toKind)`に対応する拡張規則を一意に選ぶ。**矛盾した規則・無効な規則・参照元/先の重複・不完全な辺の根拠**は、影響経路として使わない。
- 新しい`ReviewImpact`は、Graph全体と変更対象を`Validate`に通し、診断があれば`GraphValid=false`と`Diagnostics`を返す。未解決な参照があるときに`Affected=[]`だけを返して「影響なし」と誤読させない。
- `GraphValid=true`であっても、**それは渡された候補Graphと拡張規則の構造検査が成功したことだけ**を表す。元Source revisionの正当性、受理済み意味、権限、実行時の更新効果を証明しない。
- `Impact`の従来のシグネチャーは維持し、`ReviewImpact`を追加の安全な利用口とする。新しい表や図の消費側は`ReviewImpact`の診断を先に確認してから影響一覧を表示する。

負例は拡張規則の競合、参照先の欠落、派生出典の不足、未知の変更分類、存在しない変更元。正例はData→Function→Task/Formの3件の影響と、表示名に依存しないedgeでは影響が空になるケース。

この改修は#1120の共通契約に閉じるもので、#1124のStudioや#1125の業務手順を推測して実装せず、既存`ir.World`・Source Index・Shared Acceptance・Runtimeを変更しない。

**資格確認:** 本PRの正確なコミットでの`go test -race -count=1 ./internal/modelref/... ./internal/workspace`、`tools/repository-health-check.sh`、作業ツリーの事前/事後clean状態はまだ取得できていない。従来の#1139の統合事実を、本追加変更の品質保証や受理証拠へ読み替えない。完了まではDraftとする。

## R8 — 変更前後の両Snapshotによる影響確認（2026-10-12）

`ReviewImpact(before, changes, rules)`は従来の`Impact`が扱う**変更前モデルの既存Consumer**には使えるが、`added`には新要素が存在しない。変更前だけの結果が空でも、新しい要素に依存する機能が変更後モデルに増えていれば影響候補が存在する。

- 変更前しか入力のない`ReviewImpact`で`added`を受け取った場合は`after-snapshot-required`とし、`GraphValid=false`にする。過去のGraphに存在しない要素を正常にレビュー済みと断言しない。
- `ReviewImpactDiff(before, after, rules)`を追加した。双方を型・循環・根拠について検査し、いずれかに診断があれば`Affected`を返さず未確定として示す。
- 両Graphが整合する候補である場合、`Diff`の`added`を**変更後**、`removed`を**変更前**のGraphで辿る。version/evidence/参照関係の変更は**両方**を辿り、導入直後の新たなConsumerも回収する。重複する同じ`Changed/Consumer/Reason`はまとめ、決定的に整列する。
- 改名を自動認定することはない。`order-line`を`order-detail`に変更した場合、`Diff`はadd/removeを維持する。明示`Transition`の意味継承審査は別途`ReviewMigration`が担当する。
- 結果の`GraphValid`は、**二つの候補Graphと規則の局所整合性のみ**を表す。Sourceファイルrevisionの同一性、AI提案の受理、Shared Acceptance、実行時の副作用を実証しない。

回帰テストは、Dataの追加に伴って新たにFunction→Task/Formの逆向き影響を拾えること、変更後の参照が欠落する場合に結果を未確定にすること、同一DataのVersion変更後に追加されたConsumerを見逃さないことを対象とした。

### 完了の制限

ここまでの実装は`internal/modelref`に閉じた読み取り専用の契約である。正式な完了には、`tools/repository-health-check.sh`と実Workspace統合試験の完全な実行、変更前後の正本となるSource revisionの束縛、AI提案と旧資産の受理前後を追う実シナリオが必要。**追加したテストはまだtrusted full checkoutで実行していない。**

## R9 — 受理前後の実Workspaceと種類付き識別子の対応検証（2026-10-12）

共通参照機構を受理機構とは別に設計しているため、型付き参照の試験だけでは「提案→検証→受理→Source更新→再読み込み」の間にIDと出典が保持される保証にならない。逆に、Shared Acceptanceの既存テストだけでは `modelref.Key{Scope,Subject}` とSource Indexとの同時整合が検証されていなかった。

`tools/reference-shared-acceptance/modelref_identity_lineage_test.go`を追加した。既存の `scratchWorkspaceBeforeSecondApplication`、`unitPriceCandidate`、`takeSnapshot`、`acceptViaCLI`、`decisionsOf`を利用し、受理経路を複製しない。

検証シナリオは次のとおり。

1. `reference/workspace`の一時複製を正規の`workspace.Load`と`Source Index`で読み取り、`index.Build→derivedbridge.FromIndex`で種類・Scope付きのData、Attribute、Validation、Application Patternを観測する。
2. AI起点の候補を用意し、`Origin.Kind=ai`、`Origin.Actor=proposal-only`、未検証の旧資産メモを明示する。新Validation ID `sales.order_line.unit_price_positive`は、受理前のGraphにもSource Indexにも存在しないことを確認する。旧資産の実取込を行ったと主張するものではない。
3. 共通`evaluate`で読み取り専用の事前評価を行い、管理対象Sourceの全ファイル内容と基準revisionに変更がないことを確認する。
4. 既存CLIの`reviewFingerprint`拘束付き`accept`を通し、Decisionの基準・変更前・変更後revision、AI起点の出典メモを確認する。検証対象は**一時Workspace内だけ**であり、公開Reference本体を更新しない。
5. 再読込したWorldから再び`derivedbridge`を構築し、既存Data/Attribute/Validation/Patternの`Key`を保持し、新Validationが`added`として現れることを`ReviewImpactDiff`で検査する。新Validationの定義元は現在のSource Indexの`semantic/sales/validation.tei`に存在することを確認する。
6. Decision Indexから候補fingerprintが辿れること、`Origin.Actor`が`DecisionActorStatus`や`DecisionAuthorityStatus`に格上げされず、`unresolved`に留まることを確認する。

これにより、同一の**実Reference構成を複製した環境**で、候補と受理前後の論理識別子、定義ファイル、記録の追跡を一つのシナリオにした。既存のSource Index移動不変性試験やShared Acceptance Decision試験の責務を再実装しない。

**残る制限**：今回のテストはコードを追加した段階で、信頼できる完全なチェックアウトでのGo実行結果は未取得。AI起点であることは提案者の自己申告であり、人間による実際の承認を証明しない。旧資産の注記も実取込・元システムのID照合ではない。Legacy importの生データ→Source Token→受理済みモデルへの追跡、他Scopeとの混同、受理時の実行権限については別途検証する。受理前後のGraph比較は同一観測用Scopeを呼び出し側が渡した範囲に限定され、Scopeそのものの正当性を保証するものではない。

## R10 — 旧資産Source Tokenと受理済み論理IDを別々に追跡する（2026-10-12）

旧資産取り込みとTEI受理後の識別子追跡を、独立した二つの検証だけで済ませると、どちらのSourceから得た根拠で、どのReviewを経て、どの意味IDが受理されたかが失われる。

`tools/reference-shared-acceptance/modelref_rmt_lineage_test.go`に、既存の`reference-xead-source-adapter`を**実際に起動して生成するSource Token**を使い、受理前からDecision Indexまでを追う試験を追加した。

1. 一時複製したReference Workspaceの既存型付きID（Data）とSource Index定義を読み、受理前には追加Validationが存在しないことを確認する。
2. 独立した合成X-TEA XMLを実Source Adapterで読み、Artifact digest、mapping契約の指紋、元Source要素に結び付くTokenを得る。手書きの注記を根拠の代わりにしない。
3. Source EvidenceをAI提案に添付し、正規化済みCandidateと実Shared Acceptance Previewの差分に対して`prepareRMTReviewedLink`を作る。`Source Token`とTEI論理IDを同一視せず、リンクは`proposed`・意味同値性は`unresolved`にとどめる。
4. Reviewに拘束した既存Acceptを一時Workspace上で実施し、Decisionの出典、Source Token、両指紋、受理前後Revisionを照合する。
5. 再投影された`modelref.Key{Scope,Subject}`を`ReviewImpactDiff`で読み、追加されたValidationと既存Dataの識別子を区別し、Source Indexから新しいValidationの定義元まで再到達できることを確認する。Decision Indexも確認し、外部証跡を`current`や人の承認権限へ格上げしない。

**重要な限界**：使用する元XMLは独立した合成資産で、実際の顧客や原システムの旧資産ではない。Adapterが生成するTokenはSource Artifactと要素位置に束縛されるが、元の`ID`を公開用Evidenceに複製する設計ではない。合成`Table`がTEIの単価制約に相当するという意味の同値性も証明しない。形式の読込みと識別子・根拠の**追跡経路**は検証できても、実旧資産の意味移行とHuman Acceptanceの権限付与は引き続き別の判定段階である。

この試験は既存Adapter／Shared Acceptance／modelref／Source Indexの公共APIだけを呼び、新しい受理口やCore固定の旧資産メタモデルを作らない。実行結果はCIの該当headに紐づけて記録する。

## R11 — JSON互換性の保護（2026-10-12）

種類付き参照をJSONで交換する際、Go標準の`json.Unmarshal`だけでは未知の項目が読み飛ばされ、同じ名前の項目が重複すると後の値が前の値を上書きする。将来、拡張側が意味や出典に関わる項目を追加した場合、古い読み取り側がそれを黙って消し、別の意味の定義として書き戻すおそれがある。

`modelref.DecodeGraph`を追加し、`tei.modelref.v0`の読み取りに以下の条件を課す。

- トップレベルだけでなく、論理識別子・参照・出典の入れ子部分についても未知のJSON項目を拒否する。
- 同じJSONオブジェクト内で繰り返された項目を拒否する。Go標準の大文字・小文字を許容する読み取りにも対応し、`scope`と`Scope`の重複も認めない。
- 本文の後に別のJSON値が続く場合、構文が壊れている場合、未対応の版や版指定がない場合を拒否する。
- 正常な記録では、所属範囲・種類付きID・参照先の版・表示名と物理名・出典トークン・資料と規則の指紋を往復で保持する。

この処理が判定するのは**記録形式の整合性だけ**である。参照の型、存在、循環、規則は`Validate`に任せる。また、解析できたという理由で受理や実行権限を認めない。

互換性の方針は、同じ版の未知項目を無視せず、必要なら明示的な変換を実施するというもの。新しい版を理解しない古い読み取り側での自動的な読み直し・書き戻しを避け、元の情報を保持する。Coreのモデル種別やプラグイン規則を増やして対応することはしない。

**試験の範囲**：正常な往復、未知項目、重複や大文字小文字だけが異なる項目、未対応の版、複数のJSON記録、壊れた構文、形式が正しくても参照先が存在しない場合を追加する。GitHub Actionsの結果はPRの対象コミットに結び付けて記録する。

## R12 — 業務手順の編集後に三視点の参照候補を再投影する（2026-10-12）

既存の受理境界を飛び越えて`ir.World`へ業務手順を追加するのではなく、Reference Workspaceと既存の業務手順の**編集用資料**を組み合わせ、種類付き参照の保存を試す。新設した`tools/reference-business-procedure-check/modelref_authoring_reprojection_test.go`は、専ら試験内の小さな接続表を使う。業務手順の正式な正本やプラグイン規則をCoreへ追加しない。

1. 実`reference/workspace`を一時領域へ複製し、既存`acceptProcedureSource`で`order-entry.order-handling`を**編集用資料として**登録する。Source Index上の役割は`authoring`であって、Canonical Business Procedureの定義ではない。
2. 同じWorkspaceの`index.Build→derivedbridge.FromIndex`が返すData/Function側の参照に、業務手順の記述に含まれるStep→Application Pattern/Validationの明示的な参照だけを足す。Business→Function側の関係は既存`evaluate`が解決できた参照に限る。既存のFunction・Data間の宣言された関係は変更・推測しない。
3. 業務Step `BP-02`の説明を一時Source内で変更する。旧Source指紋の再利用を拒否し、新指紋との一致を確認して再読込する。Procedure ID・Step ID・Data/Pattern/Relation ID・Sourceファイルへの対応を維持する。ただし編集後にSourceのバイト数が変われば、Source Indexの終了位置は新しいバイト数に従って更新される。位置の数値そのものを不変の識別子として扱わない。
4. 前後のGraphに`ReviewImpactDiff`を適用し、説明変更は同じStepの`display-renamed`にとどまり、他要素の追加削除・参照変更がないことを確かめる。逆方向の影響候補も、実Data `customer.customer`→Relation→Pattern `order_entry.customer_reference`→`BP-02`まで辿る。
5. 存在しないApplication PatternのIDにすり替えた場合は、型付きGraphの組み立てを拒否する。未登録のBusiness手順を空白や表示名から補完しない。

これは**実Workspaceに登録可能なBusiness authoring資料を使った参照整合性試験**であり、受理済みCanonical Business Procedureの双方向編集や、人間の承認、DB更新、実行効果を証明しない。Workspaceの意味上の正本は従来のまま維持する。既存#1125の業務手順編集候補、#1124のStudio受理、#1126のデータ編集・関数従属性と責務が重ならないことを確認し、これらの実装に新機能を加えない。

この試験をCIで合格させても、Issue #1120の「受理済み三視点を同一revisionから再投影する」受入条件まで完了とは扱わない。業務手順の受理済み意味を正式に扱うための機構の進展とあわせて、継続評価する。
