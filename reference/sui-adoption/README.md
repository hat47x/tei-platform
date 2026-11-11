# SUI Reference Adoption fixtures

このdirectoryは、SUI Sensemaking → TEI Reference AdoptionのStage 0で使用する**固定fixture**です。

Product runtime、TEI Canonical source、SUI source of truthではありません。

## Source contract

SUI側の既存contract:

- `03_Implement/frontend/src/domain/contracts/island_contracts.ts`
- `IslandHierarchyContractV1Document`
- `validateIslandHierarchyContractV1()`
- `projectIslandHierarchyContractV1()`

2026-09-18時点のSUI test corpusにあるM1〜M4を、TEI側のread-only probe準備用に同じ意味で固定しています。

Stage 0がshadowしているexact baselineは次です。

| Source | Pinned value |
|---|---|
| repository | `hat47x/sui-sensemaking` |
| navigation revision | `725e9d9c8958e63c18d8edca94f19c4b98db0dde` |
| `island_contracts.ts` blob | `5d63cbec446abf0dd8054a5c498f726377c51a9f` |
| `island_hierarchy_handoff.ts` blob | `9bf17490a8d5936b4b7679370da861439e90a8ab` |
| `island_contracts.test.ts` blob | `4c91399cbc2da4526171812178ecdf98f9272027` |
| `island_hierarchy_handoff.test.ts` blob | `e069871a13c932dbb1c11cbed5e08e318f93be8c` |

revision SHAはbaselineを辿るためのprovenanceです。shadow conformanceのfreshnessはrepository全体のrevisionではなく、上記のactual dependency blobへ束縛します。SUIの無関係なfile変更だけでStage 0をstale扱いにはしません。

TEI側でfixtureを変更してSUI contractの意味を拡張してはいけません。契約変更が必要ならSUI側を正本として判断し、fixtureはその結果へ追随します。

## Cases

| File | Expected | Purpose |
|---|---|---|
| `M1-root.valid.json` | pass | root islandだけを受理 |
| `M2-three-level.valid.json` | pass | 3-level hierarchyを受理 |
| `M3-missing-parent.invalid.json` | fail | 欠損parentを拒否 |
| `M4-cycle.invalid.json` | fail | cycleを拒否 |

## Read-only Reference probe

Stage 0のconsumer-side probeは `tools/reference-sui-adoption` に置く。

```bash
go test ./tools/reference-sui-adoption

go run ./tools/reference-sui-adoption \
  -workspace reference/sui-adoption/workspace \
  -fixture reference/sui-adoption/M2-three-level.valid.json \
  -expect valid

go run ./tools/reference-sui-adoption \
  -workspace reference/sui-adoption/workspace \
  -fixture reference/sui-adoption/M4-cycle.invalid.json \
  -expect invalid
```

probeは次を確認する。

- TEI Workspaceに`sui.island` definitionが一意に存在する
- `sui.island.parent`が`sui.island`へのone relationとして解決できる
- ただしSUIの`parentIslandId: string | null`に対するrootのparent不在意味は、current TEI Relationの`one | many`だけではlosslessと証明しない
- reportでは`parentPresenceSemantics = unresolved`、`lossless = false`を明示する
- report schema v2では`constraintCoverage`を出力し、single-valued shapeだけがTEI definitionで観測できることと、nullable presence / hierarchy整合性が未表現またはReference-local shadowであることを分離する
- SUI既存handoff fixtureのM1〜M4をread-onlyで検査できる
- fixture内のIsland instanceをTEI Canonical objectへ昇格しない
- Product runtimeやfixtureを書き換えない

consumer-side validatorはSUI contractを置き換える正本ではない。SUI側の既存契約に対する**shadow conformance probe**としてのみ扱う。

probe reportにはpinned SUI revisionとactual dependency blob SHAを残し、「何をshadowした結果か」を後から再現できるようにする。

shadow validatorはSUI正本のcontract validatorより新しいdomain constraintを追加してはいけない。たとえば現行`IslandHierarchyContractV1`はhierarchy側でisland IDの非空性を検査しておらず、`schemaVersion`も空文字だけをrejectするため、TEI probeもそのpass / fail semanticsを独自に厳しくしない。

一方、JSON consumerとしてunknown fieldをfail closedすることは**consumer compatibility / transport guard**として別に扱う。これはSUI domain constraintが増えたことを意味しない。source contractが変わった場合は、SUI側を正本としてfixture / shadow probeを再確認する。

## M2 Canonical definition probe

`workspace-m2/` は、SUI の文書構造を `.tei` で宣言した場合の M2 の作業領域です。Stage 0 の `workspace/`（IslandHierarchy のみ）とは別に置き、既存 probe を置き換えません。

M2 が宣言するもの:

- `sui.document` — `documentId`、`revisionState`、`disclosureState`
- `sui.card` — `cardId`、`x`、`y`、`claimType`、`holdState`、`textReviewState`
- `sui.island` — `islandId`、`titleReviewState`
- `sui.containment`（視覚的な包含。occurrence ID を持つ）
- `sui.affiliation`（明示的・可逆・多対多の所属。occurrence ID を持つ）
- `sui.edge` — `edgeId`、`edgeType`、`fromKind`/`fromId`、`toKind`/`toId`
- `sui.island_parent`（親子 link）

**本文・題名はAttributeにしません。** 参照envelope（`tai.reference.sui-document-reference/v0`）はカード本文・島題名・文書題名を運ばないため、型にも置きません。運ぶのはその**状態**（review・disclosure・revision）だけで、それを宣言します。本文を型に置くかは M4（開示投影）と SUI #3369 の判断であり、この定義の対象外です。

**Edge の端点は kind+id 属性**にします。TEIの`relation`は静的な対象型を一つ要求するため、型付きrelation（`fromCard`/`toCard`）では island→island の Edge が表現できません。多相端点を属性で表し、この制約を定義から隠さずに記録します。

M2 が**宣言しない**もの（損失として報告する）:

- Tree の閲覧状態（折り畳み・focus・選択）
- Canvas の transform・z-order・zoom・gesture
- Island の geometry（polygon・rect・placard）
- review attribution（誰が・いつ）
- relation rationale
- narrative・summary・critique
- 本文・題名（card.text / island.title / document.title）。参照envelopeが運ばないため

確認:

```bash
go run ./cmd/tei validate -workspace reference/sui-adoption/workspace-m2
go run ./tools/reference-sui-canonical-probe -workspace reference/sui-adoption/workspace-m2
go test ./tools/reference-sui-canonical-probe/... -count=1
```

probe の報告は次を含みます。

- `containmentAndAffiliationSeparate`: 二つの membership が別 Data として解決し、それぞれが occurrence ID を持つこと
- `parallelEdgesDistinguishable`: Edge が自身の ID と kind+id 端点を持ち、型付きrelationを使っていないこと（island→island が表現できる）
- `reviewAndRevisionDeclared`: 参照が実際に運ぶ状態（document の revision/disclosure、card の textReview、island の titleReview）が宣言されていること
- `undeclared`: 宣言しない損失の一覧（空にならない。本文・題名もここに含まれる）
- `parentPresenceSemantics`: `unresolved`。現行 Relation の cardinality は `one|many` のみで、root の parent 不在（0..1）は lossless と証明していない
- `canonicalizedInstanceData: false`, `instanceCount: 0`: instance を Canonical object へ昇格していないこと

この probe は read-only の定義解決だけを行い、Product runtime・Canonical source・Owner の意味受理を主張しません。

## Scope isolation conformance matrix（M6）

`scope-isolation-matrix.v1.json` は、[ADR-0059](https://github.com/hat47x/sui-sensemaking/blob/main/01_Plans/adr/ADR-0059-saas-tenant-authorization-boundary.md) の implementation gate 5 が挙げる14経路を、**言語非依存の機械可読な行列**として書き出したものです。Go/TypeScript/Python のどれでも読め、特定のテストランナーに依存しません。

確認:

```bash
go run ./tools/reference-scope-isolation-matrix -matrix reference/sui-adoption/scope-isolation-matrix.v1.json
# SUI の checkout を渡すと、行列が指す試験名の実在まで照合する
go run ./tools/reference-scope-isolation-matrix -matrix reference/sui-adoption/scope-isolation-matrix.v1.json -sui-repo <SUI checkout>
go test ./tools/reference-scope-isolation-matrix/... -count=1
```

検査するもの:

- 14経路が矩阵に一つも欠けていないこと（欠落を「機能が無い」と黙って畳まない）
- `covered` は実試験名を1つ以上持つこと。試験ファイルと試験関数名が SUI 側に実在すること（改名で行列が黙って stale にならない）
- `not-applicable` は理由を持ち、試験参照を持たないこと
- `uncovered` は理由と担当（owner）を持つこと
- `source.languageIndependent` が true であること

現在地（2026-11-12 時点）:

| 状態 | 件数 | 内訳 |
| --- | --- | --- |
| covered | 11 | get / put / list / export / share / import / mcp / audit / agent ほか |
| not-applicable | 4 | search / count / webhook / job（SUI に該当機能が無い） |
| uncovered | 1 | cache（browser 保存の切り替え時破棄を tenant 境界として固定する試験が無い） |

**この矩阵は認可の正本ではなく、実装の越境遮断を主張しません。** 検査するのは行列の形と根拠の実在だけです。合成FixtureやHTTP単体のPASSを、実Provider・実Postgresの越境遮断の証拠として扱いません。`uncovered` の `cache` が 1 件残ることは、行列が「網羅した」と主張しないための表示です。

## Constraint coverage inventory

Stage 0 reportの`constraintCoverage`は、SUI正本のconstraintをTEI native機能として誤認しないためのEvidenceです。

| Constraint | Current TEI coverage | Stage 0扱い |
|---|---|---|
| parentがsingle-valued | `relation ... one`でshapeのみ観測 | `shape-only` |
| rootでparent不在を許す | current Relationに独立presence契約なし | `unresolved` |
| duplicate island / duplicate child | native coverageを主張しない | `reference-local-shadow` |
| parent / child existence | native coverageを主張しない | `reference-local-shadow` |
| parent-child consistency | native coverageを主張しない | `reference-local-shadow` |
| acyclic hierarchy | native coverageを主張しない | `reference-local-shadow` |

ここで`shadow-only`は「TEIにその意味が実装済み」という意味ではありません。SUI正本のpass / fail semanticsをReference toolが比較用に再現しているだけです。このinventoryを第二Referenceでも比較し、同じgapが繰り返し現れた場合にだけreusable extension / Product gap候補へ昇格します。

## Stage 0 rules

- inputを変更しない
- Canvas座標やpolygonを追加しない
- placardをこのcontractへ勝手に追加しない
- card / island instanceをTEI Canonical objectへ変換しない
- pass / fail結果をTEI Product Requirement成立のEvidenceへ読み替えない
- 最初のprobeはread-onlyとする

設計判断は `plan/design/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は ISSUE-000052-sui-reference-adoption-and-plugin-surface-evaluation.md を参照します。
