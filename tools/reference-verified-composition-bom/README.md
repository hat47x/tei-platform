# Verified Composition BOM: 2×2 Matrix の限定Reference

**担当**: [#3267](https://github.com/hat47x/tei-platform/issues/3267)。関連: #951（既存Contract/Provider解決）、#1016（案件Profile）、#1180/#2514（独立ExpectedとScenario）、#943/#2600（Artifact/Evidence）。

## これは何をするか

受注QueryのProvider二版×Consumer二版など、**既に選ばれたバージョン候補の有限直積**を安定順に列挙し、その組合せと一致する**外部から渡されたexact Lock**、**外部で実行済みと主張されたTest Witness**を照合して限定的な一覧を出力する。組合せの総数上限を超える入力は拒否する。

- `unresolved`：その組合せを固定したLockがない
- `not-run`：Lockはあるが必須のTest Witnessが未取得
- `unverified`：SKIP/unknown/ExpectedやEvidence参照欠落
- `failed`：少なくとも一つの必須Checkがfail
- `observed-pass`：参照Fixture内の必須CheckすべてがPASSと報告され必要参照がある

**重要**：`observed-pass`は第三者が提出したJSONの内部的な対応が揃ったことだけを意味する。独立Expectedの真正性、Run/Artifact/Source/Host実体、Contractの意味適合、Publisher/Consumer現在Trust/Grant/Disclosureは確認しない。出力は常に`qualified:false,claimCeiling:"reference-fixture-only"`であり、これ自体を「Verified BOM」として公開・推奨しない。

## 実行

リポジトリのGo環境で：

```sh
go test -count=1 ./tools/reference-verified-composition-bom
go run ./tools/reference-verified-composition-bom -input tools/reference-verified-composition-bom/fixture.json
```

ReferenceのGoコードは標準ライブラリのみを使う。専用の外部環境での再現には、該当ディレクトリへ移動して`GO111MODULE=off go test -count=1`も使える。

Fixtureは**実Provider/実Sourceから得たEvidenceではない**。4行の予想：
`consumer=2.0|provider=1.4` = observed-pass、
`consumer=2.0|provider=1.5` = failed、
`consumer=2.1|provider=1.4` = unverified、
`consumer=2.1|provider=1.5` = unresolved。すべて合格ではないので`matrixComplete=false`、`qualified=false`。

## 製品本体への最短接続（設計、NOT RUN）

1. #3262 のread-only Design Contextから候補軸とRequired集合を得る。上流`Stable`も`Open/unpromised`も対象。**未読込Sourceは不在とみなさない**。
2. #951/#1016の既存Resolverで組合せ候補ごとにexact Provider/Contract/Targetを決定し、Lockの識別を受け取る。ここで版範囲解決・曖昧候補拒否を二重実装しない。
3. #943のactual Build Closure/Artifact Member、#1180/#2514の独立Expected/本当のTest Run、#2600の再資格化を同じAttempt/Lock/Target/Sourceに束縛する。
4. その範囲で評価した各行から、**Verified-in-scope composition**（品質・互換性の検証済み組合せ）の発行候補を導出。Test Resultを偽造可能な単純JSONと同等扱いしない。現在Grant、業務Meaning、Publisher trustは別判定。
5. 多数の版がある場合は、**対象とCoverageを限定したpairwise/boundary/changed行の計画**を既存Scenario生成機構へ渡す。未試験行・3-way相互作用をpassにしない。初回Referenceは小さい全直積の上限制約までであり、pairwiseは未実装。

## 非目標

新たなPlugin Resolver、Version-range solver、独立Test Runner、署名・Trusted Evidence Writer、別Lock/Registry、SBOM全形式へのExport、実Host/DBの認可。これらを小Referenceで実装済みと主張しない。Jenkins型の検証済みBOMを製品基本機能へ昇格するための**最小の失敗状態と組合せの判別Reference**である。

## Source

- [Jenkins Plugin BOM](https://www.jenkins.io/doc/developer/plugin-development/dependency-management/)：実際にテストしたPlugin版の推奨組合せ。
- [Jenkins BOM GitHub](https://github.com/jenkinsci/bom)：Plugin Compatibility Testingにも使用。
- [GitHub Actions matrix](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax)：include/excludeとfail-fast。
- [CycloneDX ECMA-424](https://ecma-international.org/publications-and-standards/standards/ecma-424/)：SBOM在庫と証拠の表現。TEIのCompatibility BOMと混同しない。

**検証状況**：同内容の隔離コピーでGo 1.23.2、`GO111MODULE=off go test -count=1 -v` 4 test PASS、`GO111MODULE=off go vet ./...` PASS、`gofmt -l`空を観測。GitHubに格納された版のexact-head CI、リポジトリ全体Health、実TEI Resolver/Runner/Artifact結合はNOT RUN。
