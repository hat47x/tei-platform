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


## Pairwise計画：発行資格と完全組合せ試験を区別する（2026-11-07追加）

従来の2×2全直積の`Project`に加え、**テスト実行前の純粋な試験計画**を生成する`PlanPairwise`をReferenceに追加した。入力は既存JSON Fixtureの`axes`と正の`maxCases`のみを利用し、出力を`claimCeiling:"planning-only", qualified:false`へ固定する。現行の`Project`の`matrixComplete`とは**別結果**であり、計画上のPairwise coverage 100%と実際のTest/Runtime互換性を混同しない。

```sh
# 既存の全組合せに対するWitness投影
go run ./tools/reference-verified-composition-bom -input tools/reference-verified-composition-bom/fixture.json

# 入力軸からのPairwiseテスト計画（Test実行・Lock解決なし）
go run ./tools/reference-verified-composition-bom -input tools/reference-verified-composition-bom/fixture.json -plan pairwise
```

生成は軸・選択値を辞書順で固定する。まず第1・第2軸の全対を列挙し、第3軸以降は既存行の拡張と必要な追加行によって全2項相互作用をカバーする。**全直積数**は任意精度整数の文字列、**全2項の数／選択行によって被覆された2項の数**は別フィールド。実際に選んだ行からPairwise Coverageを再計算し、完全でなければエラーを返す。生成行数が明示`maxCases`を超えると拒否し、黙って途中結果を合格扱いしない。

**観測した隔離試験（2026-11-07）**：従来4 Testに加え、3軸2値の全直積8→4行、4軸3値の全直積81→少数行、異なる候補数の5軸全直積144、決定性、重複軸/値拒否、上限拒否、入力不変性、単軸の候補全件を含む合計10 TestがPASS。Go 1.23.2の隔離コピーで`GO111MODULE=off go test -count=1 -v`と`GO111MODULE=off go vet ./...`PASS、`gofmt -l`は空。**GitHub側exact-head CIは未実行**。

**制限**：軸値の依存制約、選択済みContract/Providerの不適合、必須Version compatibility、適用不可Target、3-way以上の既知障害、Security/Grant、Stateful/Effect、Test Priority、Runnerとの接続はこのアルゴリズムでは解決しない。生成した行に無効な候補が含まれ得るため、#951/#1016の実Resolverで合法性・足りないScopeを判定したうえで、必要な独立Scenarioを#1180/#2514に渡す。無効行を黙って除外して「残余Pairwise完全」と称してはいけない。Pairwise完全は**無制約の宣言軸集合についてのみ**。本結果を検証済みBOMとして公開しない。


## 実際に提供されたLock候補だけを使うPairwise（2026-11-07）

`-plan pairwise`は無制約のVersion軸の全直積を対象にする。一方、依存の非互換やTarget制限によって**解決済み候補が一部しか存在しない**場合に、無制約計画の行を後で削る方式はPairwise網羅を壊す。この問題を可視化するため、同じReferenceへ`-plan supplied-locks-pairwise`を追加した。

```sh
go run ./tools/reference-verified-composition-bom \
  -input tools/reference-verified-composition-bom/fixture.json \
  -plan supplied-locks-pairwise
```

入力JSONの`locks`内で**実際に与えられたexact候補**だけを選択可能行とし、その候補集合に現れた2項相互作用を決定論的にカバーする少数行を選ぶ。各行には`lockFingerprint`を付け、実際に提供されていない候補を新たに生成しない。

- `declaredPairs`：全宣言値の2項総数。
- `candidatePairs`：提供されたexact Lockに現れた2項数。
- `coveredCandidatePairs`：選択した行がカバーした2項数。
- `unrepresentedDeclaredPairs`：宣言上は存在するが、提供Lockに現れない2項数。**不可能・非互換と断定できない**。
- `candidateUniverseCompleteness: "unknown"`：Lock一覧がSource/Scope/Providerの全件かどうか検証していない。
- `qualified:false`、`claimCeiling:"planning-only"`：依存のResolver/Runtime/実テストに合格したことを示さない。

2軸×各2値の既存Fixtureでは、4宣言組合せに対しLock提供は3件、`candidatePairs=3,coveredCandidatePairs=3,unrepresentedDeclaredPairs=1`となる。3軸×各2値の追加テストでは、8宣言候補中4 Lockから**9/12ペアのみ提供され**、9/9の計画網羅を全12ペアの検証・実現可能性へ昇格させない。

異TargetのLockを混ぜる場合、`target`軸と各Lockの`Target`一致を要求する。Scope/Principal/Tenant/Grantはこの限定JSONの型に存在しないので、同じTarget値でも**同じ利用権限や完全性とは主張できない**。Product接続では#3262がRequired/Loaded/Disclosedを照合し、#951/#1016が実合法候補を解決した結果で試験しなければならない。

**検証範囲**：隔離Go 1.23.2で既存10＋新4＋入力不変性1＝**15 top-level Test PASS**、`-race`、`go vet` PASS、`gofmt -l`出力なし。新4 Testには3軸2値の**255個の非空候補部分集合**を網羅し、選択行が提供候補の全2項をカバーする負例が含まれる。GitHub PR head上のCI/本物のResolver/Host/Plugin Testは別途確認が必要であり、このReferenceの出力はVerified BOMではない。


## 同一AttemptのWitness束縛（Reference追加／製品資格なし）

`witness_audit.go`と`witness_audit_test.go`は、BOM用Test結果を外部データから束ねるときの「別RunのPASSを足した偽の統合合格」を防ぐ**read-only JSON型／純粋関数**である。主語は`WitnessAuditRequest`であり、呼出側が先に固定した各`WitnessCase`の`Lock`・`ScopeRef`・`ScenarioDigest`・`IndependentExpectedDigest`・`RequiredChecks`を、同じ`WitnessAttempt`で得たと申告された結果と突き合わせる。

**呼出入口はGo関数`AuditWitnessBindings(request)`のみ。CLIの`-audit`引数やTEI Runtime/実Provider/CI Run Readerは未実装**。Go側の単体テストは`go test ./tools/reference-verified-composition-bom`で検査できる。

判定：`not-run`、`ambiguous-attempt`、`identity-mismatch`、`incomplete`、`reported-failure`、`reported-pass`。失敗／未完／未知の情報を保存し、同じLockに複数Attemptがあるときは最新版や順序で自動選択しない。未知Lock、同一Evidence Refを別Attemptへ流用、重複Check、Expected/Source/Artifact/Target/Scope/Scenario相違を棄却する。

`reported-pass`は**入力記録の整合性を示すだけ**で、外部提出者が作った文字列・完了フラグやExpectedの独立性を認証したものではない。出力は例外なく`claimCeiling: "fixture-identity-consistency-only", qualified: false`とする。真正なProduct BOM発行は既存の実Artifact/Run/Independent Expected/Conformance/Host Grantと同一Subjectの証拠束縛を実現した場合だけ検討する。

関連：Atomic Trait Fulfillment [#3279](https://github.com/hat47x/tei-platform/issues/3279)の必須実装義務と、HTTP Response Graph [#3274](https://github.com/hat47x/tei-platform/issues/3274)のClient Reaction/未知Commitは、将来BOMの`RequiredChecks`/Scenarioへの入力として利用する。**Version Pairwise Cover、Requirement/Trait/Scenario cover、Go C0/C1、HTTP Decision Graph coverの分母は別**であり、一方の100%から他方の適格性を推論しない。

**今回新規コードの検証**：最新GitHub exact HEADのCI・ローカル実行について、取得済みの証拠が確定するまで`NOT RUN/UNKNOWN`。従来15 TestのPASSが新追加テストを含むとは主張しない。
