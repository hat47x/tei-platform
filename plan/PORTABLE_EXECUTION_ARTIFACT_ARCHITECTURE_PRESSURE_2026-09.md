# Portable Execution Artifact Architecture Pressure — 2026-09

Status: **architecture pressure / pre-Gate-C / 未採用**

## 1. この文書の役割

本書は、TEIの技術リサーチ短報で2026-09-17〜18に扱った新しい研究・実装動向を、現在のTEIアーキテクチャへそのまま輸入せず、既存の意味境界に当ててPressureするための検討文書である。

対象はFlowGen、Affora、ProgramDistill、Compositional Policy Violations、HazardAuditor、The Router Within、Hike、WebAssembly Component Model / WASI 0.3である。

目的は技術採用数を増やすことではない。**current TEI architectureにまだ明示されていない独立次元を見つけ、Canonical / Target / Provider / Runtime / Evidence境界を壊さずに扱える形へ整理すること**である。

本書はProduct Requirement、ADR、production schema、Target Pack定義ではない。

## 2. 結論

今回のArchitecture上の中心示唆は次である。

```text
semantic meaning
  != interaction representation
  != runtime execution artifact

Target
  != execution artifact
  != execution host

step-level validity
  != composed execution validity
```

とくにWasmをWeb / Desktop / Mobile / API / MCP / VUIと同列のTargetへ置かない。

同じWeb Targetでもbrowser JS、freestanding Wasm、Worker-hosted Wasm、remote Go runtime等を取り得る。同じDesktop Targetでもnative Go、Wails host + remote Go、Wails host + embedded Wasm、external process等を取り得る。

したがってcurrent TEIでは、**Runtime Binding内部にexecution artifact / ABI / hostという実現次元が存在するものとしてPressureし、Gate C前には独立production conceptへ固定しない**のが最も自然である。

## 3. current TEIとの対応

current TEIでは既にTarget CapabilityとExecution / Trust Profileを分けている。

今回のPressureから、Runtime Binding内部では少なくとも次の差を観測できる。

```text
Runtime Binding
  ├─ execution host family
  ├─ execution artifact form
  ├─ ABI / interface contract
  ├─ host imports / exports
  ├─ scheduling / concurrency mode
  └─ execution / trust / resource profile
```

ただしここから直ちに`ExecutionArtifactProfile`等のCanonical kindを導入しない。Referenceで独立ライフサイクル・identity・Conformanceが必要と確認された場合だけformalizeする。

## 4. Go-primary方針とWasmは競合しない

TEI Platform CoreをGo中心にする判断は維持できる。

```text
Go
  -> Parser / Resolver / Validator
  -> Analyzer / Compiler
  -> Runtime / Plugin Host
  -> deterministic platform core

Wasm
  -> selected generated execution artifact
  -> selected plugin/provider artifact
  -> portable sandboxable compute unit
```

Wasm採用のためにTEI CoreをWasmへ移す必要はない。逆にcurrent TEI CoreのGo sourceをそのままWasm化することを最初の目標にするのも不適切である。

## 5. Hikeから何を学ぶか

Hikeは任意のGo sourceをそのままWasmへcompileするGo compilerではない。Go風のsyntax / ergonomicsを持つ別のsystems languageであり、compiler frontendはGoで実装され、LLVM IRを介してnative / C-ABI shared library / freestanding Wasm等へ下ろす。

TEIで重要なのは言語採用そのものではなく、次のdesign patternである。

```text
narrow frontend / typed semantics
        ↓
stable low-level IR boundary
        ↓
mature backend / optimizer / linker
        ↓
small target-specific artifact
```

TEIへ読み替えると、

```text
accepted TEI meaning
        ↓
deterministically derivable typed execution subset
        ↓
artifact-provider IR
        ↓
Wasm / native / other execution artifact
```

になる。

初期Pressureではpure validation、deterministic transformation、parsing / formatting、calculation、local filtering / projection等が向く。

一方、Canonical mutation authority、long-lived business orchestration、implicit credential access、opaque shared mutable state、Business Procedure自体をWasm化しやすさから逆設計しない。

## 6. 二つのWasm profileを分けて考える

### A. freestanding core Wasm

```text
wasm32 core module
no libc / no WASI or minimum imports
explicit exports
explicit host imports
small memory surface
```

小さいartifact、browser / Worker / embedded hostへの持ち込み、host authorityのimport明示に向く。

ただしfilesystem / network / clock / randomness等はhost contractが必要で、custom JS bridgeが第二仕様にならないよう注意する。browser threadsはSharedArrayBuffer / cross-origin isolation等のhost条件へ依存するため、binary sizeだけをArchitecture KPIへしない。

### B. Component Model / WASI 0.3

```text
WIT interface
Component Model
WASI 0.3 where needed
async func / stream / future
resource handles
cross-language composition
```

ABIの手書きbridge依存を減らし、capability import / export、cross-language provider、async / streaming boundaryを扱いやすい。

ただしtoolchain / runtime差、browser側の実装差、artifact size / startup costはfreestanding Wasmと別評価する。WASI capabilityをTEI Business Capabilityへ写さない。

したがって、

```text
freestanding Wasm
  != WASI Component
```

としてPressureする。

## 7. Targetとexecution artifactを直交させる

候補Architectureは次のように読む。

```text
Accepted Application Meaning
        ↓
Target-neutral Presentation / Interaction
        ↓
Target Adapter
        ├─ Web
        ├─ Desktop
        ├─ Mobile
        ├─ API
        ├─ MCP
        └─ VUI
        ↓
Runtime Binding
        ├─ Host
        │   ├─ browser main
        │   ├─ browser worker
        │   ├─ Wails/native host
        │   ├─ server process
        │   └─ WASI/component runtime
        ├─ Artifact
        │   ├─ native Go
        │   ├─ freestanding Wasm
        │   ├─ Wasm Component
        │   ├─ JS/TS artifact
        │   └─ external process/plugin
        ├─ Interface
        │   ├─ Go/native API
        │   ├─ WIT
        │   ├─ C ABI
        │   ├─ JS bridge
        │   └─ stdio / RPC
        └─ Trust / Resource Profile
```

`Target × Artifact`を一対一対応にしない。

## 8. FlowGenから得られる示唆

FlowGenの価値はSemantic Relational Graphという形式をCanonicalへ採用することではなく、自然言語から抽出したsemantic relationを中間構造にしてflow candidateを生成・検証する点にある。

TEIでは既存の、

```text
accepted meaning
  -> rebuildable derived Meaning Graph
  -> procedure / interaction candidate
  -> cross-view proof
  -> Proposal
```

という境界を補強する材料として読む。

LLM抽出結果やgraph databaseをM1 authorityへしない。

## 9. Afforaから得られるTarget設計

Afforaの示唆は、人間UIとagent-only UIを二重管理するより、同じinteraction meaningをmachine-readableに保ったshared interfaceを作ることである。

TEIでは、

```text
one accepted interaction meaning
   ├─ human visual projection
   ├─ accessibility / semantic DOM projection
   ├─ agent-readable interaction metadata
   └─ optional automation projection
```

としてPressureできる。

agent-friendly Web UIはMCP Application Targetと同一ではない。Application自身が業務interactionをMCP protocolとして公開する場合だけ、current architecture通りMCP Targetとして分離する。

将来ReferenceではDOM selectorではなくsemantic action identity、subject identity、Outcome identity、interaction stateへ戻れるprojectionをPressureする。ただしHTML attributeやARIA構造そのものをCanonicalへ持ち込まない。

## 10. ProgramDistillから得られるEvidence設計

working applicationからreplay可能behaviorを得る考え方は、Target Conformance Evidenceへ有効である。

```text
working Reference
  -> interaction trace
  -> replay
  -> expected observable signals
```

ただし、

```text
observed behavior
  != accepted semantic source
```

である。

replayはFunctional / Interaction Evidenceとして使い、Semantic Evidence、runtime Observation、Visual Evidenceと分ける。

## 11. HazardAuditorとnormalized execution event

異なるTarget / Hostのraw eventをそのまま比較する代わりに、

```text
Target-specific event
        ↓
derived normalized execution event
        ↓
Evidence / replay / policy evaluation
```

という層は有望である。

候補eventはApplication Action、target/host identity、Business Result / Execution Error reference、Observation reference、resource/authority boundary、order/causality、raw Evidence pointer等へ戻れる必要がある。

DOM click、OS keycode、MCP transport message等をBusiness meaningと同一視しない。production event ontologyはGate C前に固定しない。

## 12. Compositional Policy Violationが示すwhole-trace境界

step単位のValidation / Authorizationが正しくても、workflow全体でauthority limitやcumulative thresholdを破る場合がある。

```text
per-action check
  +
whole-trace policy evaluation
```

の両方が必要になる可能性がある。

whole-trace policyはBusiness Procedureの隣接関係から推測せず、accepted policy / authority / threshold / review requirementへ戻る。runtimeはraw / normalized provenanceからguarded quantityを再計算し、LLM summaryやderived stateだけをpolicy truthにしない。

## 13. The Router Withinはlocal cognitionの最適化候補

```text
purpose / residual problem
   ↓
derived local routing
   ↓
selected skill / capability candidate
   ↓
exact qualification
   ↓
bounded execution
```

という位置づけならcurrent architectureと整合する。

routing resultはauthorityではない。skill数が増えたときにlexical / embedding / sparse-HDC / model-native routingをbenchmark比較する候補とし、現時点で特定routerをproduction contractへしない。

## 14. Git DAG型memoryの読み方

AgoraのようなGit DAG型shared memoryは、Proposal ancestry、verification Evidence、independent reproduction、superseded decisionを追跡する補助材料になる。

ただし、

```text
agent memory
  != accepted semantic source
```

を維持する。TEIへ汎用Claim DAGを先行導入しない。

## 15. Design Synthesisへの影響

same semantic candidateに複数realization variantを持たせられる。

```text
same semantic candidate
   ├─ native Go realization
   ├─ freestanding Wasm realization
   ├─ Wasm Component realization
   └─ remote/provider realization
```

provider feasibilityはこのrealization scopeへ適用する。

freestanding Wasmでfilesystem依存をrealizeできないことや、browser shared-memory profileがcross-origin isolationを要求することはsemantic contradictionではない。

## 16. Gate C後にPressureしたいcase

### PEA-1: same meaning, native Go vs freestanding Wasm

小さなpure validation / transformationを一つ選び、semantic result、artifact size、startup、memory、host imports、deterministic replay、error mapping、build reproducibilityを比較する。

### PEA-2: same Wasm artifact, multiple hosts

同一semantic kernelをbrowser main/worker、native/Wails host、standalone Wasm runtime等へ載せ、Target semanticsを変えずhost差だけを観測する。

### PEA-3: WIT component boundary

小さいCapability contractをWITで表し、provider language差、async / stream、host resource authority、Error / Outcome mappingをPressureする。

### PEA-4: cross-target interaction replay

同じsemantic Actionを複数Targetへ投影し、Target-specific traceをnormalized eventへ変換して同じfunctional expectationへ戻れるか確認する。

### PEA-5: whole-trace policy

個々のactionはvalidだが累積traceではpolicy違反になるnegative caseを作る。

## 17. Gate C前に固定しないもの

- `WasmTarget`という新Target kind
- `ExecutionArtifactProfile` production schema
- Hikeへの依存
- LLVM IRをTEI Canonical IRにすること
- WITをBusiness Capability schemaにすること
- WASI capabilityをBusiness Capabilityへ写すこと
- freestanding Wasm / Component Modelのどちらかを標準Runtimeにすること
- browser threadを必須Target capabilityにすること
- normalized runtime eventのproduction ontology
- replay traceをSemantic sourceにすること
- whole-trace policy engineのproduction実装
- model-native skill router
- Git Claim DAG

## 18. current architectureへ今取り込む判断

現時点で取り込むのはschemaではなく次の境界である。

```text
Target
  != execution artifact
  != execution host

Target Capability
  != artifact / ABI capability
  != Execution / Trust Profile

agent-friendly UI
  != separate Business Target

runtime event
  != Business meaning

per-step validity
  != whole-trace validity

Go primary platform
  != every generated artifact must be Go
```

この境界を維持したまま`ISSUE-000026`の第二Target Pressureへportable execution artifact / ABI差を追加し、formal conceptが必要かは実Evidence後に判断する。
