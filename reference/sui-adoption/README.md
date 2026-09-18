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
- SUI既存handoff fixtureのM1〜M4をread-onlyで検査できる
- fixture内のIsland instanceをTEI Canonical objectへ昇格しない
- Product runtimeやfixtureを書き換えない

consumer-side validatorはSUI contractを置き換える正本ではない。SUI側の既存契約に対する**shadow conformance probe**としてのみ扱う。

## Stage 0 rules

- inputを変更しない
- Canvas座標やpolygonを追加しない
- placardをこのcontractへ勝手に追加しない
- card / island instanceをTEI Canonical objectへ変換しない
- pass / fail結果をTEI Product Requirement成立のEvidenceへ読み替えない
- 最初のprobeはread-onlyとする

設計判断は `plan/SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`、作業単位は `plan/issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md` を参照します。
