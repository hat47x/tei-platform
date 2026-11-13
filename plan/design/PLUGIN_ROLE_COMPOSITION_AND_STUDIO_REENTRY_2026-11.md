# プラグインの役割別合成とStudioの自己再構築：実装・検収引渡し

- **文書の状態**：既存方針を横断して具体化した実装前設計・検収案。新しいContract、Resolver、Runtime、製品Gateを採用するADRではない。
- **確認日**：2026-11-13。GitHub上の現行mainの実装と、統合済みPRおよび進行中のDraftを静的に照合した。新たなGo/Node/Windows実行試験、Stage2 Desktopの自己再構築、業務Ownerの意味受理は行っていない。
- **優先目標**：TEI Studio DesktopをTEI自身の定義から再生成・再構築し、生成した実行ファイルの実Consumerまで確認すること。既存の[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#4583](https://github.com/hat47x/tei-platform/issues/4583)を実装Ownerとする。
- **関連する正本**：[#951](https://github.com/hat47x/tei-platform/issues/951)（Contract/Provider）、[#1016](https://github.com/hat47x/tei-platform/issues/1016)（Application選択）、[#1251](https://github.com/hat47x/tei-platform/issues/1251)（Plugin-first CLI/Studio）、[#3278](https://github.com/hat47x/tei-platform/issues/3278)（Trait履行）、[#3267](https://github.com/hat47x/tei-platform/issues/3267)（BOM）、[#2600](https://github.com/hat47x/tei-platform/issues/2600)（増分再利用）、[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)（Scenario／独立Expected）。
- **併読**：[Plugin-first実装・dogfooding](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)、[原子的Trait](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)、[Hook／Lifecycle Graph](https://github.com/hat47x/tei-platform/pull/4589)、[Desktop統合計画](https://github.com/hat47x/tei-platform/pull/4593)。

## 1. 中心となる設計判断

「Core以外は原則Plugin」とは、すべての機能を動的ロード可能にすることでも、Plugin一件ごとに独立バイナリを作ることでもない。利用者には高凝集な標準Bundleを提示し、内部では**どのTarget・工程・Consumerで、何を必要とするか**に従って必要な実装だけを構築する。

特に次の二つを区別する。

1. **構築に使うPlugin**：Generator、Source解析、Build Adapterなど。Stage1で実行されたからといって、Stage2製品に常駐・同梱する必要はない。
2. **生成製品が提供するPlugin**：Inspector、Modeler、業務Operation Providerなど。Stage2の契約が要求するなら、対応する実装がその成果物へ収録され、実際の製品入口から選択・呼出しできなければならない。

両者のContract/Provider/Source/Version/Target対応を説明できることは重要だが、同じ物理Pluginを両工程へコピーすることを要求してはいけない。Go単一バイナリは静的Compositionを既定とし、Process・WASM・外部Hostは要件と検証費用に応じた選択肢とする。特定Hostや第二のTEI Runtimeを前提にしない。

## 2. 一つのBundleが複数の役割を持ってよい

下表は**製品のCore enumではなく、計画と検証で使う用途別の見方**である。同一Pluginまたは一つのBundleが複数行を実現してもよい。

| 用途 | 実際に確認すること | Stage2への物理同梱 |
| --- | --- | --- |
| Authoring／Model Inspection | Source/Revision、選択Contract/Provider、実Inspector呼出しと型付き結果 | Stage2 Editorが使うなら必要 |
| Generator／Build Adapter | Target、選択Generator、実消費入力、実Builder invocation、生成Artifact | **原則不要**。生成物とGeneratorを区別 |
| Runtime Operation Provider | 対象Operation、Host選択、実Entry/Artifact、Result/Failure、実Effectの資格 | Stage2で提供するなら必要 |
| Presentation／Interaction Adapter | 必要な表示・イベント・Bridge/IPC、生成UI assetと実Consumerによる受領 | 対象Targetに必要な部分だけ |

役割やTraitを宣言しただけで実装義務を履行したことにはならない。Required Traitは既存#3278のDefault／Provider／Generated／Manual／External履行を検査し、未実装なら不適合とする。HintをRequiredに昇格させない。操作の業務意味とProviderの実装、実行時Grantを別に扱う。

UIの表示を増やす前に、生成AIも利用できるCLI/CUIで選択結果と不足理由を一覧・追跡できるようにする。画面・音声・Chat等の媒体は同一Application Operationへ接続するが、各媒体の完全受領・開示資格は独立に検査する。

## 3. 正本を増やさない関連付け

一つの用途について、既存のデータと証拠を次の順に**読み取り専用で関連付ける**。

```text
Source / Accepted Meaning / Required Trait
             |
      Application Operation
             |
 Contract + Profile + Scope + Revision
             |
   Selected Provider / Contribution
        /                  \
 Build-time Requirement    Runtime Requirement
        |                   |
 real Builder + inputs     Target Artifact member
        |                   |
 Generated Source / Asset  actual Host-selected entry
        \                  /
          Result / Consumer observation
                    |
      independent Expected / Grant / Effect
```

- **正本**はSource/Meaning、既存Composition解決、Builder／Build Execution／ArtifactSet、Operation、Host Authority、Scenarioに残す。この図を新しい「万能Plugin Graph」の正本として実装しない。
- **Identityの違い**を記録する。Contract/Provider ID、Sourceのbyte位置、論理Subject、実Artifact Digest、実Process image、Call-site、Consumerの表示対象を同一IDへ畳まない。Source occurrenceは名前や意味Identityの代替ではない。
- **Attemptの違い**を記録する。Stage1→各Builder→Stage2 Build→Stage2 Process→WebView2 Consumerは、工程ごとのAttemptと親子・入力出力関係で結ぶ。別HEAD／別Target／別Attemptの成功を足して一つの資格を作らない。
- **観測者の違い**を記録する。Hostが自分の実EntryやManifestを読んだことと、Providerが自称した内容と、別Runnerで取得したArtifact／実WebView2が受領したbyteは別。Hash一致だけで実Call・Conformance・Grantを主張しない。
- **失効の違い**を保持する。Source内容変更、選択Rule変更、A→B→A、未選択Plugin更新、Generatorだけの変更、Host Grant撤回は、必要なClosureのみ再検査する。過去のPASSを現在資格として自動復活させない。

## 4. Studioセルフビルドの現在位置と次の実装

現時点で[#3381](https://github.com/hat47x/tei-platform/pull/3381)／[#3371](https://github.com/hat47x/tei-platform/pull/3371)の型付きInspector Portと局所二Consumer呼出し、[#3540](https://github.com/hat47x/tei-platform/pull/3540)の静的S2-referenceがmainにある。通常CLIと実Studio Bridgeが**正式Profileで選択した別Provider B**を実呼出しするS2-product、および物理的Plugin BOM／現在Grantは別資格である。[#4539](https://github.com/hat47x/tei-platform/pull/4539)の実PostgreSQL Readは専用Reference CLIまでで、通常Product CLI／Studioの同一Context受領を証明しない。

[#4587](https://github.com/hat47x/tei-platform/pull/4587)の固定HEAD `b26e87781e00b19bf142b22a35cfc62a857e88a3`では、生成Inspector部品の実使用と生成Windows Wails/WebView2の実配信asset照合が同じCIで成功した（B0限定、Draft未統合）。後続B1-A/N0の追加試行は、その新HEADのCIで独立に検収する。次の不足はB1〜B3である。

| 段階 | 既存資産を用いた最小実装 | 受入の上限 |
| --- | --- | --- |
| **B1-N0** | Inspector契約＋変換Source→決定的TS生成、実消費byteと公開assetの一致 | 部品生成と利用の資格のみ |
| **B1-N1/N2** | PoC B generated Go/TS、npm lock＋Node/esbuild→JSを、既存Builder/Executionへ適合 | 生成物の実消費とBuilder来歴。外部CIだけのBuildは不合格 |
| **B1-N3** | nested Studio Go module、embed、build tags、Wails/Windows実BuildをArtifactSet／Subjectへ結びつける | Stage2の実exe／memberの資格。Go rootの単体試験PASSだけでは不可 |
| **B2** | Stage1 **TEI実Process**が同じBuild定義を解決し、必要Builderを発行してStage2 Desktopを作る | 子Process/argv/実Toolchain/Source閉包/子Artifactで再入を確認 |
| **B3** | **そのStage2 exe自身**をWindows WebView2で起動し、生成Inspector／配信JS／IPCを検査 | 固定Fixture／指定OSの実Consumerに限る |

現行`internal/buildintent/intent.go`では`BuildIntent`のSubjectはPluginに制限される一方、`go-command`というRecipeは存在する。「GoコマンドをBuildできる」ことはDesktop.exeを正式なBuild Subjectとして表現できることを意味しない。**Wails exeを便宜上Pluginと偽装しない**。既存#1080/#1097のExecution→ArtifactSet→logical Subject Bindingへ薄いAdapterで収まるかを先に調べ、表せない工程は`unsupported`として残す。新Subject Schemaや全工程対応をP0の先行決定にしない。

## 5. X-TEA移管とHook拡張の位置付け

[#4570](https://github.com/hat47x/tei-platform/pull/4570)の`plugins/xtea-three-view/`は、Builtin以外の**実業務Modeler Plugin**を検証する適切な次の題材である。最初はread-onlyのData／Function／Business三視点、CRUD、項目参照、Sourceへの移動に限定し、モデルの編集・受理済みMeaningへの自動昇格・実DB書込みを同時に要求しない。

このPluginについて、`plugins/`にSourceが存在する、Compositionで選択された、実GoにLinkedされた、StudioBridgeから呼ばれた、Windows Desktopでその画面を利用した、を別に記録する。同名別Identity、重複Relation target、staleなPin、Source参照0件、partial Coverageを合格に読み替えない。#1124と#1824をOwnerに保ち、B1のBuild Graph適格化をこのPluginの実Desktop完成待ちにしない。

Hook/Lifecycle Graphは[#4589](https://github.com/hat47x/tei-platform/pull/4589)／[#4198](https://github.com/hat47x/tei-platform/issues/4198)の既存担当へ戻す。宣言`uses`と実`calls`、Guardによる拒否、`skipped`と`notEvaluated`、変更前後の呼出し、Effect unknownを区別する。GraphのPatch適用や可視化が成功しても、実Hook Providerが一回だけ正しく呼ばれた証拠にはならない。**Graph Compiler完成をStudioセルフビルドの先行条件にはしない。**

## 6. 独立Expectedに追加する検収候補

既存[#1251のPXC-01〜08](https://github.com/hat47x/tei-platform/issues/1251#issuecomment-6846915963)を再利用する。ここではBuild-timeとRuntimeの**役割の違い**に由来する差だけを追加し、第二Scenario正本を作らない。

| Case | 故障注入／対照 | 期待する区別 |
| --- | --- | --- |
| **PXR-01** | Stage1で使うGeneratorがStage2に存在しない。生成TS/JSはStage2で実使用 | **正当な物理除外**。Generator未同梱だけを理由にFAILとしない |
| **PXR-02** | Stage2が提供するInspector B／三視点ProviderがArtifactにないが、Catalogには掲載 | **不合格**。選択・掲載だけで実装済みとしない |
| **PXR-03** | B選択だがAに実配線。両者の返すCanonical Resultは同一 | 実Entry witness不一致でFAIL。CLI・Wailsは別実Consumerで確認 |
| **PXR-04** | 未選択C追加によりGo import/init/embedが変わり、Stage2へCが混入 | 非干渉を否定。Catalog変更のみなら必要Artifactは維持 |
| **PXR-05** | Node/esbuild/WailsがBuild Intent外で動いた／root Go testのみ成功 | B1/B2未成立。対象nested moduleと実Invokerを別照合 |
| **PXR-06** | 前段の生成Assetは正しいが、Stage2 WebView2は旧Bundleを配信 | 実Consumer資格FAIL。実配信byte／実使用を確認 |
| **PXR-07** | Pin参照が0件、同名別Source、Relation対象曖昧、Source変更後の古いpin | 不存在・再束縛の推定は禁止。対象の再照合またはstaleで停止 |
| **PXR-08** | Hookが宣言されるが未呼出し、Guard前にeffectful Call、ACK喪失 | 実Call/EffectをPASSしない。未知Effectなら無条件再送しない |

**実装順**：P0はB1の最小Builder/Artifact接続と#1251の実製品Inspector S2を並行して進める。B2→B3をStudio自己再構築の中心Gateとし、X-TEA実Desktop検証、Graph Patch、他言語Target、2×2 Qualified BOM、AI Harnessの完全同値性は独立の後続検証にする。不要Plugin混入の回避と、必要なPlugin欠落の検出を両側から測る。

## 7. 採否・撤回条件

この設計の価値は、**一つのContractと既存Build/Operation基盤のまま、役割に応じて同梱・非同梱を正しく判別できること**にある。次のいずれかが必要になるなら、抽象を拡大する前に案を縮小して再検討する。

- 実Provider選択に第二Resolverや第二Hostが必要になる。
- 役割ごとにCore固定enumや別のMeaning／Grant正本を増やさなければならない。
- 未使用Generatorを追加しただけで無関係な実Artifactを必ず再生成する。
- 単純な一部品の自己再構築にGraph Compilerや全Target ABIの完成を要求する。
- Build実行の成功・Evidence delivery/ACK・Consumerが実利用した事実を一つの「成功」へまとめてしまう。

本書は、Issue・PR・CI・Artifact・Receiptから必要な事実を結び付けるための設計引渡しである。新しいGo実装・CI試験・実Stage2 Desktop・正式な意味受理は**NOT RUN／NOT ACCEPTED**。
