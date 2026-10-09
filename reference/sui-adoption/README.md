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

設計判断は `plan/design/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は `plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md` を参照します。
