# ISSUE-000046: Studioの最小Authoring Loopをshared Acceptance Boundaryで実証する

- 状態: Blocked — Gate C Ready待ち
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

## 最小Authoring Loop

### 1. Select

既存Source Index / Canonical identityから対象Dataを選びます。

最初のReferenceでは、利用者に内部UUIDや手書きStable IDの入力を要求しません。

UI上の選択は、既存authoring sourceへ解決可能なqualified logical key / Canonical identityを利用します。

file pathそのものを業務identityにはしません。

### 2. Propose

Studio操作は、その場でauthoritative sourceを直接変更せずcandidateを作ります。

candidateには必要に応じて次を保持します。

- operation intent
- target qualified logical key / Canonical reference
- proposed relation / change
- `baseModelRevision`
- provenance / actor

candidateの具体schemaを最初からplatform-wide標準へ固定しません。

### 3. Preview

Accept前に少なくとも次を確認できるようにします。

- Semantic Diff
- affected Canonical subjects
- persistence / binding impact
- Application Pattern / Driver impact
- Target Projectionへの影響候補
- Validation / Architecture Diagnostic

Preview自体はauthoritative sourceを変更しません。

### 4. Reject

Rejectではauthoritative sourceを変更しません。

Studio session / layout / selection等の非Semantic stateだけが変化してもCanonical meaningへ影響しないことを確認します。

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
- candidate intent
- Preview Semantic Diff
- impact / Diagnostic
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

## 完了条件

- [ ] Gate C Ready後に着手する
- [ ] Studioから既存Data / RelationをSource-to-Canonical trace付きで選択できる
- [ ] small Relation changeをcandidateとして作れる
- [ ] PreviewでSemantic Diff / impactを確認できる
- [ ] Preview / Rejectでauthoritative sourceを変更しない
- [ ] Acceptがfile-native / CLI / AIと同じshared Acceptance原則へ接続する
- [ ] accepted sourceをStudioなしでreloadできる
- [ ] reload後CanonicalがValidation / Conformanceを満たす
- [ ] stale `baseModelRevision`を黙って適用しない
- [ ] file moveをsemantic no-op、logical renameをsemantic changeとして区別できる
- [ ] mandatory authored Stable IDをStudio都合で導入しない
- [ ] Wails Hostからbusiness / persistence semanticsを分離できる
- [ ] Studio Wails HostをApplication Target Evidenceへ数えない

## 非目標

- Gate C前のStudio実装
- 完成IDE
- 全Canonical subjectのvisual editing
- YAML / file-native sourceの廃止
- mandatory UUID / authored Stable IDの導入
- Studio専用Acceptance engine
- WailsをTEI全体の標準Target / Runtimeへ固定すること
- Generated Wails Desktop Application Targetの実装

## 関連文書

- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- `plan/issue/000008-editor-and-ai-followup-note.md`
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000026-target-capability-execution-profile-boundary.md`
