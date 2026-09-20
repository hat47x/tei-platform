# ISSUE-000046: Studioの最小Authoring Loopをshared Acceptance Boundaryで実証する

- 状態: In Progress — Gate C Ready済み。Studioが使うShared Acceptance Boundaryは決定論的Referenceとして成立（2026-09-21、`tools/reference-shared-acceptance`）。Studio UI・Data / Relationの選択・Relation変更は未着手
- 起点: `ISSUE-000008` Editor / AI follow-up
- 関連: `ISSUE-000027` AI Proposal / Acceptance Boundary
- Product Acceptance Scenario: **PAS-01 primary**
- 関連RQ: RQ-001 / RQ-002 / RQ-003 / RQ-010 / RQ-011 / RQ-017 / RQ-019 / RQ-021 / RQ-022

## 目的

Gate C通過後、TEI Studio / EditorがCanonical meaningの第二正本にならず、**file-native authoringと同じshared Acceptance Boundaryを使って小さな意味変更を安全に行えること**をReferenceで実証します。

最初から統合IDE、巨大なvisual modeler、自由編集UIを作りません。

最小loopは、`order_entry`周辺の既存Data / Relationを使った一つのRelation変更を対象にします。

```text
Select Data
  ↓
Propose AddRelation / small relation change
  ↓
Preview
  ├─ Semantic Diff
  ├─ Persistence impact
  └─ Target / UI impact
  ↓
Accept / Reject
  ↓ Accept only
Shared Acceptance Boundary
  ↓
authoritative file-native source
  ↓ reload
Canonical meaning / Conformance
```

より広い設計上、このIssueは [`../../docs/definition-centered-feedback-loop.md`](../../docs/definition-centered-feedback-loop.md) の**local authoring / Preview / Accept-Reject**を担当します。Studioにknowledge storeや独立したAI memoryを持たせず、`ISSUE-000047`の探索、`ISSUE-000027`のProposal、将来のderived cognitionを同じselected-definition contextへ接続します。

## Studioの位置づけ

Studioは**Projection / Authoring Client**です。

```text
Canonical / Source Index
        ↓ read
Studio Projection
        ↓ candidate
Shared Acceptance Boundary
        ↓ accepted change
authoritative source
```

Studio自身を次にはしません。

- Canonical meaningの第二正本
- YAML / TEI file-native sourceの代替正本
- 独自Validation engine
- 独自Authorization / business rule engine
- Target固有meaningの発生源

Studioを削除しても、accepted authoring sourceとCanonical meaningを再読込できることを要求します。

## Definition-centered authoring

Studioの役割は「定義体をGUIへ置き換えること」ではありません。

原則は、**authoritative sourceに必要なirreducible meaningだけを変更し、そこから派生する結果をPreviewすること**です。

```text
selected concise definition
      ↓
minimum semantic delta
      ↓
Preview causal consequences
      ↓
Accept / Reject
```

### Minimum Semantic Delta

AI、Studio form、drag-and-drop、CLI等の入口にかかわらず、初回Referenceではwhole file / whole modelの再生成より、現在revisionに対する小さいmeaning changeを優先します。

candidateには少なくとも次を説明できることを求めます。

- 何を直したいか
- exact target identityは何か
- accepted baselineから何が意味上変わるか
- 何は変わらないか
- どのSource / Evidenceが根拠か

これはplatform-wide Patch DSLを固定する要求ではありません。

### Derived defaultをEditorへ再authorしない

標準CRUD、derived View、dependency matrix、reverse cross-reference等をEditor上で見せても、それを利用者へ再入力させません。

```text
derived Projection is visible
  != user must author it
```

利用者がoverrideする理由が生じた場合だけ、どのmeaningを明示化するかを別candidateとして扱います。

## Local-first workspace

realistic Reference規模ではwhole-model graphを常設編集面にしません。

selected definitionを中心に次を局所的に辿れることを目指します。

```text
selected definition
  ├─ authored local meaning
  ├─ incoming / outgoing exact refs
  ├─ Data / Function / Business lenses
  ├─ Source / Evidence
  ├─ affected derived projections
  ├─ applicable / incompatible / unknown candidates
  └─ diagnostics / unresolved
```

全体graph / matrixは必要に応じたProjectionです。検索・scope・local neighborhoodで認知負荷を抑えます。

AIを利用する場合も、原則としてこのlocal contextとexplicit referencesを渡し、Repository全体を暗黙に再解釈させることを標準にはしません。

## 最小Authoring Loop

### 1. Select

既存Source Index / Canonical identityから対象Dataを選びます。

最初のReferenceでは、利用者に内部UUIDや手書きStable IDの入力を要求しません。

UI上の選択は、既存authoring sourceへ解決可能なqualified logical key / Canonical identityを利用します。

file pathそのものを業務identityにはしません。

Select時には、`ISSUE-000047`のread-only exploration結果やReference pattern candidateを表示できても、それらをselected authorityへ自動昇格しません。

### 2. Propose

Studio操作は、その場でauthoritative sourceを直接変更せずcandidateを作ります。

candidateには必要に応じて次を保持します。

- operation intent
- target qualified logical key / Canonical reference
- proposed relation / change
- `baseModelRevision`
- provenance / actor

candidateの具体schemaを最初からplatform-wide標準へ固定しません。

AIが候補を作る場合も`ISSUE-000027`のProposalとして扱い、Studio internal stateへmeaningを隠しません。

### 3. Preview

Accept前に少なくとも次を確認できるようにします。

- Semantic Diff
- affected Canonical subjects
- persistence / binding impact
- Application Pattern / Driver impact
- Target Projectionへの影響候補
- Validation / Architecture Diagnostic

Preview自体はauthoritative sourceを変更しません。

さらにDefinition-centered loopでは、**結果だけでなく因果を表示できること**を目指します。

例:

```text
required Relation added
  -> Action input requirement changes
  -> reference selector candidate appears
  -> persistence migration classification changes
  -> Runtime referential-integrity guard changes
```

各edgeはexact dependency / deterministic derivationへ追跡できる必要があります。AI explanationだけをimpact authorityにはしません。

### 4. Reject

Rejectではauthoritative sourceを変更しません。

Studio session / layout / selection等の非Semantic stateだけが変化してもCanonical meaningへ影響しないことを確認します。

candidate cognition、AI chat、Preview cacheを削除しても同じaccepted stateへ戻れることも維持します。

### 5. Accept

AcceptはStudio専用write経路を通しません。

```text
file edit -------┐
Studio ----------|
CLI / refactor --|--> candidate --> Shared Acceptance Boundary --> authoritative source
AI Proposal -----|
Matrix ----------|
optional Patch --┘
```

対象に応じて次を共有します。

```text
Parse / Decode
  ↓
Normalize
  ↓
Validate
  ↓
Architecture / Policy Check
  ↓
Semantic Diff
  ↓
Impact Analysis
  ↓
Acceptance Policy
  ↓
authoritative source update
  ↓
reload
  ↓
Conformance
```

入口ごとに別のsemantic validationやauthority modelを作りません。

Accept後、derived UI / matrix / index / cognition等はaccepted sourceから再生成します。Studio stateを次revisionのsemantic prerequisiteにしません。

## feedback routing

Studioで問題を発見しても、常にselected Semantic definitionを直すとは限りません。

例:

```text
business relation missing
  -> Semantic source candidate

Web-only interaction friction
  -> Target / Presentation candidate

PostgreSQL realization mismatch
  -> Mapping / Provider candidate

search candidate missed
  -> derived retrieval / cognition candidate
```

利用者が最も目につく場所ではなく、meaningを所有する最小concernへ変更をrouteします。

初回Referenceでは少なくとも一つ、UI symptomをSemantic sourceへ誤修正しないnegative caseを持つことを検討します。

## revision conflict

Studioがcandidateを作った後にsourceが変わることを通常ケースとして扱います。

`baseModelRevision`が現在revisionと一致しない場合、黙って最新sourceへ適用しません。

少なくとも次へ明示分岐します。

- reject
- re-read / regenerate candidate
- explicit rebase
- manual conflict resolution

初回Referenceでは複雑な自動mergeより、staleを確実に検出して誤適用しないことを優先します。

## identityとfile-native source

### file move

file移動だけでqualified logical identity / Canonical meaningが変わらないケースをReference fixtureに含めます。

### logical rename

logical key / Canonical identityを変更するrenameはSemantic changeとして扱います。

file path変更とlogical renameを同じものにしません。

### authored Stable IDを必須にしない

初回Studio loopのためだけに、すべてのauthoringへUUID / mandatory authored Stable IDを追加しません。

identity要件は既存のqualified logical key / Canonical identity / Source Indexと整合させます。

## Wails Hostを使う場合

Studioのdeveloper-facing desktop hostとしてGo + Wails / WebViewを使う場合、次の境界を維持します。

```text
Web frontend
    ↓
Wails developer host
    ↓
adapters/studiowails
    ↓
app/studio
    ↓
Shared Acceptance Boundary
    ↓
Canonical / authoritative source
```

Wails outer hostへ次を持たせません。

- business validation
- Authorization rule
- persistence semantics
- Canonical identity rule
- direct Canonical mutation authority

StudioのWails Hostは、Generated Wails Desktop Application Targetとは別物です。

Studio GUIが動作したことを、第二Application TargetのSemantic / Functional / Visual Evidenceへ読み替えません。

## Evidence

初回Referenceでは少なくとも次を残します。

- source / model revision before
- selected logical identity
- local neighborhood / exact refs used for review
- candidate intent
- minimum semantic delta
- Preview Semantic Diff
- impact / Diagnostic and derivation reason
- feedback routing result
- Accept / Reject result
- source revision after
- reload result
- Canonical identity / meaning after reload
- Validation / Conformance result
- stale candidate negative case
- Studio session state削除後もCanonicalが再構成できること

さらに、同じ意味変更をfile-native authoringから行った場合とStudioから行った場合が、同じAcceptance原則・Semantic Diffへ収束できることを比較します。

byte-for-byte同じ編集操作を要求するのではなく、**accepted Canonical meaningと検証境界が入口依存にならないこと**を確認します。

## Gate Cとの関係

本IssueはGate C前に実装を開始しません。

Gate Cから少なくとも次を受け取ってから着手します。

- Handover Baseline / Application identity
- 第一Web Targetで観測したPresentation / Interaction差
- Semantic / Functional / Visual Evidenceの分離
- Target Capability / Execution / Trustのunresolved

これにより、Studio UI都合からApplication meaningを先行固定しません。

2026-09-21: Gate CはReadyとなり、上記4項目は[Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md)（Handover Baseline `B / R / T`、Semantic / Functional / Visual Evidenceの分離、carry-forwardのunresolved、第二Targetへのhandoff）から受け取れる。本Issueは着手可能である。着手条件はProduct Gate Cだけで、Lane C（000048 / 000049）の完了は待たない。

Gate C後の最初のReferenceでは、[`../DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`](../DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md)に従い、`ISSUE-000047`の探索と`ISSUE-000027`のProposalを同じone-change loopへ接続します。

## 実証状況

### 最初のincrement — Studioが使う受入境界（2026-09-21）

Studioの操作はcandidateを作るだけで、authoritative sourceを直接変更しない。そのcandidateを受け取る境界は、[`tools/reference-shared-acceptance`](../../tools/reference-shared-acceptance/README.md)として、UIなしで実証した（[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)の実証状況と同じ実装）。Preview（Semantic Diff・exact dependencyに基づくimpact・unresolved）、Reject、Accept、stale `baseModelRevision`、受理後のtoolなしの再読込・Conformance、派生物の再生成、file moveとlogical renameの区別を、Reference workspaceのscratch copyで確認している。

このincrementで扱った変更は、本Issueが最小loopとして想定するRelation変更ではなく、`order_entry`が使う`sales`への**Validation追加**である（`ISSUE-000027`のFeedback Loop Planが許す最初の変更候補）。Relation変更はpersistence・Application Patternへの影響が大きいため、この受入境界へoperation kindとして追加する次のincrementで扱う。

未着手: Studio UI（Select・local neighborhood表示・Preview表示）、既存Data / RelationのSource-to-Canonical trace付き選択、Relation変更のcandidate、Wails Host。Studio Wails HostをApplication Target Evidenceへ数えない原則は変わらない。

## 完了条件

- [x] Gate C Ready後に着手する（2026-09-21着手）
- [ ] Studioから既存Data / RelationをSource-to-Canonical trace付きで選択できる
- [ ] selected definition / local neighborhoodだけで初回changeをreviewできる
- [ ] small Relation change等をminimum semantic delta candidateとして作れる
- [x] PreviewでSemantic Diff / impact / derivation reasonを確認できる（CLI / JSONで確認できる。Studio上の表示は未実装）
- [x] Preview / Rejectでauthoritative sourceを変更しない（共有受入境界のReferenceでbyte単位に確認）
- [ ] feedbackをvisible symptomではなくmeaning-owning concernへrouteできる
- [ ] Acceptがfile-native / CLI / AIと同じshared Acceptance原則へ接続する
- [x] accepted sourceをStudioなしでreloadできる（共有loader / Validationだけで再読込）
- [x] reload後CanonicalがValidation / Conformanceを満たす
- [ ] derived projection / indexをaccepted sourceから再生成できる
- [x] stale `baseModelRevision`を黙って適用しない
- [x] file moveをsemantic no-op、logical renameをsemantic changeとして区別できる
- [x] mandatory authored Stable IDをStudio都合で導入しない（追記されるsourceはqualified logical keyだけ）
- [ ] Wails Hostからbusiness / persistence semanticsを分離できる
- [ ] Studio Wails HostをApplication Target Evidenceへ数えない

## 非目標

- Gate C前のStudio実装
- 完成IDE
- 全Canonical subjectのvisual editing
- YAML / file-native sourceの廃止
- mandatory UUID / authored Stable IDの導入
- Studio専用Acceptance engine
- whole-model graphを必須authoring surfaceにすること
- AI chatをStudioの主authoring modelにすること
- derived CRUD / matrix等の人手再authoring
- WailsをTEI全体の標準Target / Runtimeへ固定すること
- Generated Wails Desktop Application Targetの実装

## 関連文書

- `docs/definition-centered-feedback-loop.md`
- `plan/DEFINITION_CENTERED_FEEDBACK_LOOP_PLAN_2026-09.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
