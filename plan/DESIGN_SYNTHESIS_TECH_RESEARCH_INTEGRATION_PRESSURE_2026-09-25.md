# Design Synthesis — 2026-09-25 技術リサーチ統合Pressure

## 位置づけ

「TEI 技術リサーチ短報」で追っている2026-09後半の技術動向を、Design Synthesis / local cognitionのcurrent boundaryへ当てるread-only architecture pressureである。

これは採用ADRではない。Gate C前にproduction contract、AI provider、Wasm runtime、UI IRを先行固定しない。外部技術の主張はArchitectureのauthorityではなく、既存境界に対するpressure / hypothesisとして扱う。

## 今回の入力

### Jev / System One Models

TypeSafe AIはJevを、unstructured stateから事前定義されたtyped decisionとprobabilityを返すmachine-native decision primitiveとして提示している。workflow evalでは、狭い質問へ分解し、programmatic ruleとmodel judgmentを組み合わせる構成を前面に出している。

TEIへの示唆は「LLMをJevへ置換する」ことではない。R1〜R6でdeterministicに縮約した後のbounded residualを、生成モデルだけでなくtyped probabilistic decision engineでも評価できるよう、**decision mechanismとsemantic authorityを分離する**ことである。

Jev等のprobability / confidenceは次を意味しない。

- accepted meaning
- proof result
- current applicability
- candidate winner
- automatic Shared Acceptance

したがって将来providerを導入する場合も、`DecisionObservation`相当のderived EvidenceとしてProposalへ添付し、Acceptance ownerを持たせない。

### Hike系の軽量 Go → Wasm producer

2026-09にはGo風の記述から小さいWasmを生成するHike系実装が注目されている一方、公開issueにはwasm32での型不一致や巨大固定長配列アクセス時のstack overflow等も残る。

TEIではこれを新しいDriverとはみなさない。Pressureとして価値があるのは、**portable deterministic kernelのrealization targetをHostから独立させられるか**である。

候補境界は次とする。

```text
Canonical / accepted semantics
  -> deterministic kernel contract
      -> native Go realization
      -> lightweight Wasm realization (experimental)
      -> other target realization

Host
  -> Web
  -> Wails desktop
  -> mobile / MCP / API
```

Wasm producerの制約やfailureはmatching realization variantの問題であり、Canonical meaningへ逆流させない。これはprovider-feasibility scope pressureと同じ原則をruntime targetへ適用したものになる。

### Wails v3 beta.24 / beta.25

Wails v3はdesktop APIのBeta compatibilityを掲げつつ、9月後半もprocess lifecycle、asset middleware security、MCP origin / remote access、binding等の修正が続いている。

TEIではWailsをCore runtimeではなく**Desktop Host / adapter**として扱う現在の方向を維持する。Host固有security、process lifecycle、WebView request cancellation、packaging等はHost capability / realization Evidenceであり、Business / Canonical semanticsを所有しない。

またWails側がdesktop/mobileへ同一Go codebaseを広げても、TEI側では「同じHost abstractionで全targetを隠す」ことを目的にしない。target固有capabilityと制約をEvidence付きで露出できる方を優先する。

### Power Apps model-driven app-builder / MCP Apps

2026-09のPower Apps app-builderは、自然言語要求からまずapplication planを作り、人間がreviewした後にtables / relationships / forms / views / security roles / business process flows等の標準artifactを生成・編集する流れをGAとして提示している。MCP Appsではtool JSON outputからself-contained UI widgetを生成するpreviewも進んでいる。

TEIへの重要なPressureは、生成UIそのものより**plan-before-mutate**と**standard artifactへpromotionする境界**である。

TEIでは次の順序を崩さない。

```text
intent
  -> semantic candidates
  -> deterministic contraction
  -> bounded AI / human synthesis
  -> proposed semantic delta
  -> validation / Evidence
  -> Shared Acceptance
  -> target-native artifacts / UI realization
```

AIが直接target artifactを生成できても、そのartifactからCanonical meaningを逆authorしない。生成UIはaccepted semanticsのderived realizationであり、必要ならExperience/UIの中間projectionを置いてframework-specific code generationと分離する。

### Model-driven UI prototyping研究

2026年のmodel-driven UI研究では、process / concept modelからintermediate UI modelを構成し、その後pattern selectionとframework-specific code generationへ進む構造が提示されている。

TEIではこれを、CanonicalへUI widget taxonomyを混ぜる根拠にはしない。一方で、将来の`Experience Projection`（仮称）を**derived / regeneratable**に保ち、semantic modelとframework realizationの間でscreen state / route / view / action mappingを検証するpressureとして有効である。

### Skill RSI型 evidence-backed promotion

Skill RSIはcandidate variantを同じeval条件で比較し、prompt-level Evidenceを保持しながらchampionをpromotionするloopを公開している。

TEIのS3〜S5に対する示唆は、self-improvementそのものより次にある。

- proposalとaccepted baselineを分ける
- comparison conditionを固定する
- deltaを小さくする
- promotion decisionをEvidenceへtraceできるようにする
- dead end / regression Evidenceを捨てない

ただしTEIではmodel judgeやscoreだけでsemantic meaningを自動promotionしない。評価結果はShared Acceptanceの入力でありauthorityではない。

## 統合後のarchitecture hypothesis

今回の外部動向を入れても、Coreの中心をAI agentへ移す必要はない。むしろ次の分離が強くなる。

```text
M1 Accepted Semantic World
  |
  +-> deterministic projection / cross-view proof
  |      -> supported / contradiction / unresolved
  |
  +-> candidate frontier
          |
          +-> deterministic rule / solver
          +-> typed probabilistic decision provider (Jev-class)
          +-> generative reasoning provider (LLM-class)
          +-> human judgment
                  |
                  v
             Proposal / semantic delta
                  |
             Validation + Evidence
                  |
             Shared Acceptance
                  |
                  v
          Accepted semantic change
                  |
       +----------+-----------+
       |          |           |
   native Go   Wasm kernel   target/UI projection
       |          |           |
       +------ Host / Driver / Adapter ------+
                  |
           Observation / Replay
                  |
             S5 feedback

M2 Derived Knowledge / Cognitive Index / Experience Projection
  -> regeneratable
  -> build-input freshness
  -> never M1 authority
```

## 新しく強まった設計原則

### 1. Intelligence Portは一種類にしない

`LLMProvider`をAI境界の唯一の抽象にすると、Jev型のtyped decision engineや将来のdeterministic learned classifierを不自然にLLMへ偽装することになる。

ただしGate C前にproduction `IntelligenceProvider` interfaceを作らない。actual S3 Referenceで、少なくともgenerative reasoningとtyped probabilistic decisionの双方に同じbounded residualを与える必要が確認された場合にのみinterfaceを抽出する。

### 2. Decision probabilityとproof confidenceを混ぜない

Jev型providerのprobabilityはmodel observationである。R1〜R6のcross-view proofはaccepted authorityに基づくdeterministic qualificationである。

同じ`confidence` fieldへ畳まない。

### 3. Portable kernelとHostを分離する

WasmはWeb専用Driverではなく、deterministic logicを複数Hostへ持ち出すrealization target候補として評価する。WailsはDesktop Host、browserはWeb Host、MCPはtool/UI Hostとして扱い、semantic kernelとHost lifecycleを分ける。

### 4. AI生成UIはExperience authorityにしない

AI-generated page/widgetはProposalまたはderived realizationである。screen / route / actionの意味を保持する必要が出た場合は、Canonicalへframework componentを入れず、accepted semanticsから再生成できるExperience Projectionを先に検討する。

### 5. Promotion loopはEvidence-firstにする

S3〜S5のclosed loopでは、生成品質の平均scoreよりも、どのsemantic deltaを、どのaccepted baselineに対し、どのEvidenceでpromotionしたかを再現できることを優先する。

## Gate C前に追加しないもの

今回の調査だけを理由として、次を追加しない。

- Jev専用production adapter
- generic `IntelligenceProvider` / `DecisionProvider` interface
- probability / confidenceのCanonical field
- Hike依存またはWasm必須化
- Wasmを前提としたCore再設計
- Wails固有semantic model
- production Experience IR / UI DSL
- MCP App widgetをCanonical artifact化する仕組み
- automatic champion promotion / self-improvement loop
- model scoreによるShared Acceptance

## Gate C後の最小実証順

actual Referenceが開いたら、次の順で比較する。

1. realistic S1 frontierを作る。
2. R1〜R6 / provider scopeでdeterministically縮約する。
3. unresolved residualを質問単位へ分解する。
4. 同じresidualについて、必要ならLLM-classとtyped-decision-classの出力shape / latency / Evidence traceを比較する。
5. minimum semantic deltaだけをProposal化する。
6. Shared Acceptance後にtarget realizationを生成する。
7. native Go / Web / Wails / experimental Wasm等のrealization差をsemantic acceptanceから分離して検証する。
8. Observation / replayをS5へ戻し、semantic defect・realization defect・Host defect・operational recoveryをowner-routingする。

## current conclusion

最新動向は、TEIを「AIがアプリを丸ごと生成する基盤」へ寄せるより、現在の**semantic authority + deterministic contraction + bounded cognition + evidence-backed promotion + multi-target realization**という方向を補強している。

追加で重要になったのは、cognition内部も一枚岩にせず、`deterministic / typed probabilistic / generative / human`をauthorityから独立したdecision mechanismsとして扱うこと、そしてWasm / Wails / generated UIをsemantic Coreではなくrealization / Host側へ閉じることである。

この結論はread-only architecture hypothesisであり、Gate C前のproduction adoptionを意味しない。

## 外部Reference（2026-09-25確認）

- TypeSafe AI, Introducing System One Models & Jev: https://typesafe.ai/blog/introducing-system-one-models-and-jev
- TypeSafe AI workflow evals: https://evals.typesafe.ai/
- Wails v3 changelog: https://v3.wails.io/changelog/
- Wails v3 roadmap: https://v3.wails.io/status/
- Microsoft Power Apps, Generate MCP App widgets with AI code generation tools: https://learn.microsoft.com/en-us/power-apps/maker/model-driven-apps/generate-mcp-app-widgets
- Microsoft Power Platform, September 2026 feature update: https://www.microsoft.com/en-us/power-platform/blog/power-apps/whats-new-in-power-platform-september-2026-feature-update/
- Frontiers in Computer Science, model-driven automated UI prototyping: https://www.frontiersin.org/journals/computer-science/articles/10.3389/fcomp.2026.1813208/full
- Hike公開issue: https://github.com/kanryu/hike-lang/issues
- Skill RSI: https://github.com/justinwetch/Skill-RSI
