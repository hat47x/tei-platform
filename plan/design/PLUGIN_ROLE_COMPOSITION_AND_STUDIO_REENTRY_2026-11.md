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


---

<!-- plugin-stage2-xtea-conformance-20261113 -->
## 8. 最初の非Builtin Providerを通した製品構成の実証（2026-11-13追補）

この章では[#4570](https://github.com/hat47x/tei-platform/pull/4570)のX-TEA由来の「三要素分析法のモデル」を、Plugin-firstとStudio自己再構築が交差する**具体的な圧力試験**として位置付ける。利用者向けの概念名は「三要素分析法のモデル」とし、内部パッケージ名や既存のOperation識別子 `three-view` は移行コスト・互換性の検証なしに変更しない。画面に表示する三つの視点を「三視点」と呼ぶことと、モデル全体の名称を取り違えない。

### 8.1 現行実装のFit／Gap

#4570のDraft Sourceを静的に確認すると、`cmd/tei-studio/host_three_view.go`が`plugins/xtea-three-view`の`Navigate`を明示的にimportし、`internal/threeviewadapter/host.go`が既存`composition.Registry`で`authoring-tool` scopeを選択する。Hostは実関数symbolを照合する。通常の`host_default.go`と生成版`host_generated.go`はいずれも`withStudioThreeView`を呼ぶ構成になっている。この事実から、**二つのBuild tagで同じ論理Providerを扱うSource上の候補**までは確認できる。

ただし、これは#4570のDraft branch上の構成であり、mainへの統合・正式Application Profileの選択・必要Source closureを持つBuild Intent・実Windows Wails/WebView2での当該Provider呼出し、Stage2再構築の証拠ではない。静的Go importで実装が利用可能であることと、対象Profileで必要なProviderが正しく選択され、成果物へ収録され、実Consumerから呼ばれたことは区別する。

また、`threeviewadapter`には`inspectoradapter`と似たHost選択・束縛が局所的に存在する。重複を増やすのは望ましくないが、両者の選択失効・Source照合・エラー開示・call-site差を独立Expectedで比較するまでは**新しい万能Adapterへ一括統合しない**。共通化するのは判別できた契約だけとする。

### 8.2 Role×Stage×Consumerを一つの検収票にする

Pluginの種類をCore固定enumへ追加せず、**用途別Roleと、どの段階で必要か**を、既存の解決結果からread-onlyで投影する。最小票の列は次のとおり。

| 列 | 何を記録するか |
| --- | --- |
| Subject/Target/Build variant | 対象Application・Studio通常版／`studio_generated`・OS・build tags |
| Contract/Scope/Required | `tei.operations.three-view-navigation/v0`、`authoring-tool`、このProfileで提供必須か |
| Selected/Implementation | exact Provider ID・選択epoch・実inventory binding・Host観測entry |
| Build provenance | 実消費Go source/import・nested module・Toolchain・Build Intent/Plan・ArtifactSet・実binary |
| Consumer/Attempt | Stage1 Builder、Stage2 `StudioBridge.NavigateThreeView`、実Windows WebView2の呼出しと受領、親子Attempt |
| Source/Result/Qualification | `.xead`原byte digest、Workspace・Source位置・pin・coverage・typed failure・独立Expected・現在資格 |

**「宣言はあるが実Consumerが無い」**と**「必要なのに実binaryから欠落」**は異なる不合格原因であり、同じ`missing plugin`に潰さない。SourceにIDがない業務ActionのIdentityを補完しない。Source occurrenceはSource byte位置であってCanonical IDではなく、参照0件はSubject不存在の証拠にしない。

### 8.3 最小の実装順

- **X0（#4570/#1124）**：合成`.xead`でread-only`NavigateThreeView`の独立Expectedを固定する。原典Fact／構造候補／受理済みMeaningを分離し、契約v0の`not-bound`を守る。Functionの宣言CRUDと業務Actionの推測CRUDを混同しない。
- **X1（#1251/#951）**：通常Studioの実Host選択→Provider実Entry→Typed Resultと、Inspectorで既にある同種経路を比較する。CLI/Studioを同一契約の別Consumerへ拡張するのは製品S2接続後。内部Adapter重複の縮小はこの検証から必要部分だけ抽出する。
- **X2（#1824/#4570）**：通常版と`studio_generated`版を**それぞれ実Windows Desktop**としてビルドし、実`StudioBridge`→Provider→Resultと、画面のSource navigationを確かめる。server buildやGo testの成功はDesktop試験の代わりにしない。
- **X3（#4583/#2831）**：Studio全体のN0〜N3を既存Build Execution／ArtifactSetへ載せ、Stage1 TEI実ProcessがStage2を構築した後、**そのStage2 exe自身**が当該Providerを呼んだ証拠を取る。B1/B2/B3に新しいGateを追加せず、X-TEA試験はB1のHard Dependencyにしない。
- **X4（#3267/#2600）**：同じ必要Providerが収録される正例と、未選択Plugin Cが混入しない負例を、`go list -deps`、選択Source・import、実binary・必要symbol・実呼出しの複数観測で判別する。inlining等によりsymbol一つが無いことだけで物理欠落を断定しない。

### 8.4 PXR-09〜15：既存#1180/#2514へ引き渡す追加の独立Expected候補

| ID | 故障注入・対照 | 期待判定 |
| --- | --- | --- |
| PXR-09 | 通常版だけX-TEA Providerが入り、`studio_generated`版では`withStudioThreeView`の実Callが欠落 | 同じ契約を提供すると宣言したTargetの資格だけFAIL。別版の成功を流用しない |
| PXR-10 | 固定Providerは同じだが、生成版に別実装が同じJSONを返す | selected ID／実Entryとbinaryの一致を照合。結果同値だけではProvider同一性を認めない |
| PXR-11 | Bundleに未選択Provider Cを追加し、静的import／init等を経て生成Desktopへ混入 | 実閉包非干渉をFAIL。候補Catalogの増加だけなら許容 |
| PXR-12 | 画面に「三要素分析法のモデル」が現れるがProviderは呼ばれていない／既存JS結果を表示 | 実`StudioBridge` call／actual entry／Source読取を要求。単なる画面表示は実Plugin利用の証拠でない |
| PXR-13 | 同名別ID、ID重複、Source変更後のpin、参照0件、coverage partial | 自動同定・全件性・意味受理は禁止。Qualified targetの保持／stale／partialを区別 |
| PXR-14 | `threeviewadapter`と`inspectoradapter`の重複を解消した際、一方のみProvider不在をBuiltinへfallback | 既存の両入口・負例を保持。欠落・失効はfail closed |
| PXR-15 | Stage1 Source/CIはPASSだがStage2実binaryに必要Providerが含まれず、旧exeで画面試験を実施 | Stage2の実image digest／Build Attempt／Provider／実Consumerを結べないならB3・X3は未成立 |

これらは**既存のPXR-01〜08、PXC-01〜08を置換しない補助試験候補**である。実装に先立って独立Expectedを固定し、同じFixtureから正負例を比較する。単に列を埋めたレポートをConformance/Owner受理と呼ばない。

### 8.5 採否とScopeの制限

この検証から「一つの契約と既存Composition/Build/Operationで、通常Studio・生成Studioの必要な実装を選択し、実Consumerまで追える」ことが示されれば、同じ方法をDSL/Query、DB Provider、SHO連携、CLI/MCP、実行Applicationへ段階展開する。ただし各Targetでの実Grant、部分取得、Effect、外部Host、表示・編集の適格性は別に検証する。モデル専門家の意味受理やX-TEA原典との対応を、プラグインの適合試験で代行しない。

**状態**：この章は現行Draft Sourceの静的照合と実装・検収計画の追補。PXR-09〜15の新Go/CLI/Windows/Stage2試験、全Build Intent、正式Owner意味受理は**NOT RUN／NOT ACCEPTED**。


---

<!-- plugin-assurance-declared-observed-enforced-20261113 -->
## 9. 宣言・観測・強制を区別するプラグイン実装保証（2026-11-13横断追補）

### 9.1 設計判断

Plugin-firstの保証は、**宣言（Declared）／観測（Observed）／強制（Enforced）**を分けて扱う。これは新たなCore enum、万能Proof Store、横断認証機構ではなく、既存の#951/#1016（Selection）、#2595/#4583（Builder/Source）、#4486（Host境界）、#3267（Artifact）、#1251（Invocation）、#1174（Effect/Grant）にある事実を読み取り専用で照合するための判定上の区分である。各段階のSubject・Target・Source Revision・Process・Attempt・観測範囲を保持する。

| 区分 | 確認対象 | 主張できないこと |
| --- | --- | --- |
| **Declared** | 契約・Trait義務・ProfileでのProvider選択・Build Input/Recipe・Resource/Effect要求 | 本当にビルドした、読み取った、呼び出した、権限がある |
| **Observed** | Host-selected entry・実Process image/argv・実Artifact/配信byte・事前/事後のSource digest・Consumer受領 | 未宣言のInput/Processが**存在しなかった**こと、時点間のTOCTOU/ABAが無かったこと |
| **Enforced** | 明示した境界WにおけるOS/Hostの拒否。例：未許可Resource書込み、誤Provider Binding、Grant拒否 | W外の読取り・ネットワーク・別OSも禁止したこと、業務意味・実Effectが受理済みであること |

**誤った昇格を避ける：** Windows CIでN0〜N3の名前付きファイルSHAを採取しても、後読みは実Builderが同じbyteを消費した証拠ではない。Go `go list -deps`・静的importは入力閉包の候補だが、Child側に未列挙のファイル読込み、Node/esbuildのtransitive input、実行中差替えは別に判別する。実Entry symbolの観測は呼出しの参考になっても真正な署名・Conformance・現在Grantではない。強制された**不存在**は特定の資源種別と境界W内でだけ主張する。

[#4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `d7ad13ceabc39b3917a129409630879564f8384c`のWindows CIについては、N0〜N3の**指定したファイル**と実生成Desktopの観測が成功している。これは計画に役立つ足場だが、同PRの`qualifiedBuildIntent=false`／`dependencyClosure=unknown`を維持する（[#4583の実装観測記録](https://github.com/hat47x/tei-platform/issues/4583#issuecomment-6847330010)）。その後のHEAD/CIは別試行として検収する。

[#4486](https://github.com/hat47x/tei-platform/issues/4486)ではLinux opt-in `tei self-build -state-boundary`の書込み隔離が局所試験で成立した。しかし読取り共有、ネットワーク、環境変数、常駐Process、別マウント、Windowsは境界Wの外である。二回目のstateが存在していたという事後検査（`entry-post-hoc`）と、境界内の書込み禁止（`host-boundary`）を混同しない。前者から「隠れた共有が不可能」と推論しない。**この境界をWindowsのB1/B2へ機械的に当てはめない**。

### 9.2 B1で最初に閉じる一辺

Studio全工程の一般化に先立ち、**N0の生成Inspector TS → N2 esbuild → N3 Go embed**の一経路だけを、既存Build ExecutionとArtifactSetへ接続する。

1. **宣言**：入力Source/Transform、選択済みBuilder・Toolchain、出力のrole、Target/build tags、前段成果物への参照を、既存の受理可能な型へ落とす。Desktop exeを偽Plugin Subjectにしない。
2. **実起動**：Entryが起動するProcess image、argv、作業領域、受け渡す入力・出力、Child Attemptを固定して記録する。選択Builder B／実起動A、宣言N0／実消費旧TSを区別する。
3. **実受領**：N2のJS byte→N3 embedに渡した資産→Stage2 exeが配信したJSとInspector実使用を照合する。単に事後ファイルをハッシュしただけなら`observed-post-hoc`相当の限定記録にとどめ、実読取時点が不明な辺を`unknown`とする。
4. **強制境界**：固定Snapshot／入力の実読取観測／OS境界のうち既存仕組みで成立するものだけを採用。未実装の読取り・ネットワーク・Windows隔離は`unsupported`または`unknown`。B1の最小実呼出し証拠を、全OS完全隔離の実装まで止めない。
5. **再利用**：#2600には「真に未選択・未消費のPlugin変更」と「未列挙だが実消費されたSource変更」を対照させる。Build byte一致、Evidence再利用、現在の受理資格は個別に判定する。

### 9.3 独立Expectedの追加候補 PRA-01〜08

既存#1180/#2514のScenarioに接続する差分であり、第二Runner・Gateは作らない。

| Case | 注入・比較 | 期待される判定 |
| --- | --- | --- |
| **PRA-01** | N0出力TSをN2起動前に差替え、事後は元byteへ戻す（A→B→A） | 後読みSHA一致のみではPASSせず、実読取/固定Snapshotが無ければunknown |
| **PRA-02** | 選択Builder Bと実Process Aの出力が同じ | 実image/argv不一致でFAIL |
| **PRA-03** | `package-lock`以外の実transitive JS inputを変更、目録から欠落 | 実依存閉包をqualifiedとしない。差分/読み取りを追跡 |
| **PRA-04** | Stage2に必要なProviderが欠ける一方、Stage1のGeneratorは動いている | Stage2提供資格FAIL、Stage1のGenerator成功は維持 |
| **PRA-05** | Stage1専用GeneratorがStage2から除外される | 選択Targetが必要としなければ正当。単純な「Plugin欠落」では拒否しない |
| **PRA-06** | Linux書込境界が通るが読取り共有・ネットワークに依存 | W内の書込み制約のみ支持。完全なstate isolationは未成立 |
| **PRA-07** | Hookの`uses`を宣言したがGuard拒否／二重Call／Commit ACK喪失 | 宣言≠呼出。Guard拒否時の0回・多重作用・unknown Effectを別に判定 |
| **PRA-08** | DB ProviderがRLSで一部の行のみ返し、UI/AIが全件として表示 | 実Read成功でも全体被覆・現在開示は未成立。partial/unestablishedを維持 |

### 9.4 DSL・Trait・Hook・AI協業への適用順

| 接続先 | 共通にすること | 固有のOwnerと未証明事項 |
| --- | --- | --- |
| **DSL／Target Generator** | Source→型付き意味→選択Operator/Provider→生成実装の対応。構文同値だけでなく、評価順序・数量精度・NULL・依存失効の比較 | #4361〜#4367、#1121、#2595。新DSLの全文法・全Target Conformanceを本Plugin計画で確定しない |
| **原子的Trait** | `required`→Default/Provider/Generated/Manual/Externalの実Fulfillmentと失敗診断。単に宣言されたHintを保証にしない | #3278/#951。Trait追加で新Core enumを要求しない |
| **Hook／Lifecycle** | Contract選択・実呼出し回数・Guard・Failure/Effectを区別し、Graphの静的表示を実Callとみなさない | #4589/#4602/#4198/#1174。未知Effect時の再送禁止。全Graph CompilerをB1の前提にしない |
| **DB Provider／Query** | 選択済みEnvironment→実Reader→Rowset/Context→Consumer受領。Source/Session/Grant・RLSの未知を分ける | #2586/#3270。実PG Reference CLI成功をStudio/本番Grantに拡大しない |
| **AIと人間の協業** | AIの候補Proposal、固定Selection・Source/Revision、実Apply、Build開始、実Consumerを分離。結果に根拠と欠落を示す | #2514/#2516/#1824。AI推定・検証PASSを業務意味の受理・現在権限に置換しない |

実装・配布の第一目標は引き続き**Studio DesktopをTEI自身が再構築し、生成されたStage2実行ファイルで利用できること**。PRAは必要な縦断の反例を短く固定するものであり、X-TEA実Source、全Plugin移行、DSL全面自己記述、AI Harness自律Apply、他言語TargetをP0へ積み上げるものではない。

**状態**：現行Issue/PR/Sourceの静的照合に基づく設計・検収引渡し。PRA-01〜08の新Go/Node/Windows/OS境界試験、Stage1→Stage2 Desktop全工程、正式Owner意味受理は**NOT RUN／NOT ACCEPTED**。


---

<!-- plugin-profile-closure-20261113 -->
## 10. 案件別の選択と静的Composition：固定参照構成から生成製品への移行

### 10.1 現行の固定構成が実証していること

[#4570](https://github.com/hat47x/tei-platform/pull/4570)のDraft Sourceでは、`cmd/tei-studio/host_three_view.go`が`tei/plugins/xtea-three-view`をbuild tagなしでimportし、`studioThreeViewInventory()`に明示的な`Navigate`関数を登録する。`newStudioThreeViewHost()`は`xteathreeview.ProviderID`を引数としてHost選択する。通常版`host_default.go`と生成版`host_generated.go`はともに`withStudioThreeView`を経由する。

これは**固定参照構成で、Providerを選択し、同じGo実装へ到達する候補経路を持つ**という有意義な実装である。しかし、Application Composition Profileがプラグインの必要性を解決し、その結果から**target別のGo import／linkを生成・除外**した証拠ではない。また、`host_three_view.go`の無条件importがある現構成で「非選択PluginはStage2から物理除外される」と主張してはならない。現在の固定Bundleでは、あらかじめ必要として組み込んだプラグインであることと、Product Profileの任意選択機能であることを混同しない。

この差は、新Resolverや動的ローダーの不足ではない。**現在の選択結果を、Target向けの静的Composition・Build Closureへ伝える接続の不足**である。

### 10.2 プラグインに関する三つの問いを独立に扱う

1. **この製品構成で必要か**：#1016/#951の既存Profile/Contract/Scope/Trait要件から、選択する機能・実Providerを確定する。標準セットに含まれるという事実だけでは、全Targetへの実同梱義務は生じない。一方、標準Profileが明示的に要求するものは必要である。
2. **成果物へ組み込まれたか**：#2595/#1080/#1097/#3267の実Build入力、生成Go import・link、embed、ArtifactSetで検証する。Providerが存在しても、選択先の実Targetに入っているとは限らない。未選択CがGo import/initで入る場合、物理除外の要求を満たさない。
3. **誰がどの操作で使えるか**：#1158/#1251の実Call-site／Typed Result、#1174/#2516の現在Grant・開示を確認する。実Binaryに含まれることは、自動公開・権限付与・Effect成功を意味しない。

この三問の回答を、**Source/Model Revision・Target/Profile・Contract/Provider ID・Build Attempt・実Artifact・実Consumer Attempt**で結び付ける。World Fingerprint／Model Revision／Artifact Revisionは、対象集合と失効用途が違うため単一Hashにしない。異なるTarget間でBinary byte一致を要求せず、契約が同じ場合の型付きOperationの意味・負例を比較する。

### 10.3 固定参照構成とProduct構成の合格条件

| 対象 | どこまで合格と言えるか | 次に必要な検査 |
| --- | --- | --- |
| 固定した参照Bundle（現#4570） | Hostが選んだProviderを実Go call-siteで呼出す局所Reference | 実Windows `StudioBridge`／Provider／Read結果とSource navigation |
| 通常Studioと`studio_generated`の固定構成 | 両variantの宣言された必須Providerが実同梱・実Callされた | 各variantのbinary／実WebView2／独立Expected。server-only成功では不可 |
| 案件別Application ProfileからのProduct構成 | Profileで選択したContract/Providerを、既存Composition→Target Generator→実Buildへ束縛できる | **required／not-required**の両Profileで生成Go import・link/Artifact・実Call/露出差を比較 |
| 配備後の現在資格 | 現在のHost/Grant・要求Scope・Consumerに適合する | 旧Artifact/Evidenceの再利用が現在のGrant/Owner意味受理を復活させない |

最初の段階で、製品CLI・Studio・全Plugin・全Targetの動的切替は要求しない。**参照構成に必要な実装が入ることの立証をP0、案件別Profileに不要な実装を物理除外する保証をその後のP1**とし、順序を逆転しない。固定参照Bundleの成功からProduct Profile対応済みと呼ばない。

### 10.4 PPC-01〜08：選択閉包の独立Expected候補

既存#1180/#2514のScenarioへ接続し、新たなSource/Contract/Scenario正本を作らない。

| ID | 差分・故障注入 | 必須判定 |
| --- | --- | --- |
| PPC-01 | 固定参照構成が`xtea-three-view`をimport・選択し、実Bridgeから呼ぶ | 限定ReferenceとしてPASS可能。案件別Profileによる生成とは判定しない |
| PPC-02 | Profileが三要素分析法Modelerを要求するが、静的生成importかStage2成果物から欠ける | 必須Provider欠落。Build/Consumer資格FAIL |
| PPC-03 | Profileでは三要素分析法Modelerが不要だが、無条件Go import/initで成果物に残る | 物理除外を要求するProfileではFAIL。未公開なら現在Grant違反まで推定しない |
| PPC-04 | 同じ論理ContractでProvider AとBを切替。結果JSONは同じだが、生成import・実Entryが旧Aのまま | Selected B→actual AをFAIL。Result同値は不十分 |
| PPC-05 | PluginのPresentationだけを変更し、業務Operation/ContractとGenerator入力には変更なし | 影響対象だけ再生成。Source/入力閉包が無関係なら過剰失効を検出 |
| PPC-06 | Required TraitのProvider履行が欠けるが、Presentation表示とHook宣言だけは存在 | 義務未履行として診断。Hint／画面表示を実装保証に昇格しない |
| PPC-07 | Dirty Buffer Bを画面表示中、保存済Source AをBuild Intentで選択。BuilderがBを読んだ | Plan/Actual不一致。暗黙保存・Meaning受理はしない。AをBuildするならAと明示 |
| PPC-08 | AIまたはREAの外部候補を既存Modelerへ受渡したが、旧Pin/別Source／Scopeで実Providerへ接続 | 候補受領≠現在の対象確定。Source・Targetを再確認し、業務MeaningやGrantを自動付与しない |

PPC-05は`PRA`の実Builder消費入力、#2600のfalse invalidation/false reuse、#2595の消えた依存辺・実読取時点を再利用する。PPC-07は#4583のDirty Buffer/Canonical Source、PPC-08は#4515/#3394/#2516の既存Ownerへ戻す。重複実装をしない。

### 10.5 実装の最短順と撤回条件

- **S0 現在の固定参照構成を据え置く**：#4570を無理に汎用Plugin Hostへ書き換えない。X-TEA固有の読み取りと、共通Operation・Target依存の責務を維持する。
- **S1 #4583 B1-Bを優先**：N0→N2→N3の既存Builder実入力とArtifactSetの接続を実装する。Stage2 WailsのBuild Inputの完全性やOS隔離を未実証なら`unknown`とする。#4587のCI Shellを正式Builderだと偽らない。
- **S2 #1251の製品呼出し**：通常CLI実binaryと実Wails Bridgeが選択済みProviderを呼んだWitnessを別々に取る。read-only Inspectorから始め、三要素分析法のモデルを独立の二例目にする。
- **S3 #1016/#951/#1121のProfile→静的Composition**：既存Selectionが`required/not-required`を確定できる場合に限り、Target側の明示import・必要Artifactへ翻訳する。新Plugin Manager、Core固定enum、無条件のGo Module分割は不要。
- **S4 差分Build・可視化**：#2600に選択あり/なしのBuild閉包と再生成時間を渡す。Studio/CLIの既存Inspect/診断へ「宣言／実選択／組込／実呼出／現在資格」の読取り専用情報を投影し、未知・部分観測も隠さない。

**撤回条件**：この方法に第二Resolver／第二Runtime Authorityが必要になる、Targetの静的構成を記述するだけで毎回全Pluginを一括importしなければならない、各表現の表示変更が必ず全Go成果物の意味再受理を必要にする、またはCIの事後Hashを実消費閉包と扱わないと成立しない場合は、Profile連携の抽象を縮小し、一機能の差分へ戻す。

**資格**：本章は#4570/#4587等の現行Draft Source、他チャットのUX・DSL・REA・Trait設計に基づくFit/Gapと独立Expectedの提案である。PPC-01〜08の新Go/CLI/Windows Target試験、Profileからの静的import生成、Stage1自身によるStage2全Desktop再構築、正式Owner受理は**NOT RUN／NOT ACCEPTED**。


---

<!-- plugin-crosschat-stage2-domain-toolkit-20261114 -->
## 11. 正式Desktop Build対象とドメインToolkitへ展開する条件（2026-11-14横断統合）

### 11.1 優先する一本の製品経路と、先行成果の資格

直近の[#4650](https://github.com/hat47x/tei-platform/pull/4650)（Desktop設計Draft）、main統合済み[#4652](https://github.com/hat47x/tei-platform/pull/4652)（DSL診断の四層引渡し）／[#4589](https://github.com/hat47x/tei-platform/pull/4589)（Hook/Lifecycle設計）および[#4623](https://github.com/hat47x/tei-platform/pull/4623)（選択Builderの実行前照合Draft）を、別々の合格として読み替えず接続する。

最優先は**Studio DesktopをPlugin Subjectへ偽装せず、非Pluginの正式なBuild対象として、既存Builder／Build Execution／ArtifactSetへ接続すること**である。ここで検収する単位は、単なる「Studio生成成功」ではなく、同一の選択・Build Attemptとその子Attemptが示す次の連鎖である。

1. 保存済みSource・Build対象・Target/Variant・選択Builderを固定する。EditorのDirty Bufferは別の状態であり、暗黙にSourceとみなさない。
2. 対象に必要な入力を解決し、実Builder Process／argv・Toolchain・子Process・実消費したbyteと出力を結び付ける。入力閉包が未証明なら、その範囲を**unknown**として残す。
3. N0生成Inspector→N2 bundle→N3 embed→Windows Wails exeを、既存のBuild Execution/ArtifactSetの実Artifactとして追う。入力名と実消費Sourceが一致しない場合は、出力一致でも不合格。
4. Stage1の**実TEI Process**がStage2のBuildを駆動し、Stage2の実行ファイルを起動したあと、**当該Stage2自身のWebView2／Bridge／Provider**が必要な生成assetとOperationを実際に使うことを観測する。
5. 必要なProviderの同梱、実呼出し、現在Grant、業務意味受理は別列で検証する。未選択Pluginの不存在は、単なるSymbol欠落や後読みによって断定しない。

[#4623](https://github.com/hat47x/tei-platform/pull/4623)のBuilder SHA pinは「選択した実行ファイルとの実行前一致」の候補であり、実Build IntentがそのBuilderを選んだこと、入力閉包の完全性、OSの起動時差替えが不可能なことを証明しない。Stage2はArtifactの候補であって、それだけでは実Consumerの資格を得ない。製品の受入票では **Definition／Build／Runtime／Delivery、Source／実Process／Artifact／Consumer／Grant** を異なる証拠として扱うが、新しい横断Gate・証拠正本・万能Build Managerは作らない。

### 11.2 ドメインToolkitは新しい言語ではなく、同じContractの用途別の束

TEIによるアプリケーション開発の生産性向上を測る領域として、統計処理、WBS/PERT、科学計算、バイオサイエンスなどを**ドメインToolkitの候補**に加える。これらはStudio自己再構築を完了するための追加Hard Gateではなく、Studio上でのDSL／図解／表／テスト／生成を共通の構造から導出できるかを測る後続の圧力試験とする。

それぞれについて、以下を独立した貢献として合成する。

| 貢献 | 分離する責務 | 結び付ける既存正本 |
| --- | --- | --- |
| **Domain Model／記法** | ドメイン概念、単位・粒度・時間、入力制約と意味。Markdown・Graphviz・表・DSL等の表示糖衣を構造から分ける | Canonical／Source occurrence／Meaning／Trait |
| **Operator／計算法** | 同じ意味入力に対する実装。Go実装、外部R／Mathematica由来の検証済みアルゴリズム、別Target生成などを選択可能にする | Contract／Provider／Toolchain／Conformance |
| **Data Provider** | 実計測・DB・ファイル・外部API取得と、部分取得・鮮度・対象範囲 | Environment／Read Result／Source provenance／Grant |
| **Presentation／Report** | グラフ、科学的可視化、表、文書、UIをConsumer別に生成する。描画座標を計算上の意味へ逆流させない | Operation／Target Artifact／Consumer／Disclosure |

**重要**：RやMathematicaの式を転記・抽出したというだけでは、新しいTEIの意味正本・再利用許可・数値適合は成立しない。利用可能な契約・ライセンス、入力データの出所、アルゴリズムの適用条件、精度・丸め・収束・乱数seed、欠損値とNULL、単位変換、Targetの数値演算差を明示する。外部エンジンへの常時依存は基本形にせず、必要なProvider／Builderとして選択した場合だけ実消費閉包へ加える。実Toolchain/Providerと、表示だけの依存も分離する。

DSL構文そのものの抽象定義、JSON Schema等の構文取込み、図解への可逆変換は有望だが、**新しい任意文法エンジンをP0へ先行導入しない**。まず既存Parser／Canonical／Source Index／Studio診断を利用し、一つの構文糖衣で「原byte・参照位置→構造→表示／編集→原形式への往復」を検査する。構文がParse可能でもMeaning受理・Provider適格・公開Entry・Current Grantは別であり、[#4652](https://github.com/hat47x/tei-platform/pull/4652)の四層を越境しない。

### 11.3 初期Toolkitの候補と、製品価値の測り方

| 候補 | 最小のEnd-to-end課題 | 共通基盤の検査点 | 失敗時の縮小先 |
| --- | --- | --- | --- |
| **統計・時系列・Reporting** | 行の粒度・欠損・時間窓を宣言し、移動集計とwindow式から結果表・グラフ・報告書を生成 | 入力完全性、RDBとin-memoryの集計同値、窓境界・時差・丸め、実Consumerの集計値 | まず一つのwindow Operatorと固定Fixtureだけ |
| **WBS／PERT・工程計画** | Task／依存Relation／日程／所要時間を一つのモデルとして編集し、最早/最遅時刻とクリティカルパスを出す | 多重辺・同名別ID、カレンダーと経過時間、循環拒否、変更伝播、図／表／予定の同じ対象参照 | 資源制約なしのDAGと決定的期間に限定 |
| **系・システムと科学計算** | 単位付き状態量と関係式、初期条件、時間発展を定義し、計算表と時系列を表示 | 次元整合、境界条件、解法と許容誤差、収束失敗、再現性 | 微分方程式一般化の前に純粋な保存則／次元検査 |
| **バイオサイエンスの可視化・解析** | 出所が確かな少量Datasetで、変換→集計→多層可視化を再生成 | データ粒度・欠損・識別子、由来・開示許可、可視化のread-only再現 | 汎用グラフ／時系列Toolkitへの投影のみ |

最初に採用する一例は、**外部エンジンなしで完結する統計window Operatorと独立Expected**を推奨する。数値結果、ソース追跡、Query/関数合成、Studio Inspector、レポート出力を一つのFixtureで比較しやすく、外部Runtimeや巨大な科学演算ライブラリをセルフビルドの前提にしない。その後にWBS/PERTの多重辺・時刻モデルを試す。系のシミュレーションやバイオ可視化は、基礎の数値・グラフ・出所の契約が実例で適格化してから選別する。

生産性向上はコード行数だけで測らない。モデル定義から**同じ意味で生成できた実Target数、編集から診断・再生成・Consumer受領までの待ち、手修正した箇所と失効範囲、テスト作成・参照の再利用率、手書き実装との同値・相違点**を同一条件で記録する。StudioセルフビルドそのもののBuild時間と、Toolkitで作るアプリケーションの開発時間は別指標とする。

### 11.4 後続の独立Expected：PDK-01〜08

既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)で扱う**候補**であり、新たな検収正本は作らない。

| Case | 変更する一因子 | 独立Expected |
| --- | --- | --- |
| **PDK-01** | 統計入力の完全取得0行と、未取得/一部取得を入れ替える | 0とunknown/partialを混同せず、母集団全件を主張しない |
| **PDK-02** | 同じ数値に対して単位・時間窓境界・NULLを変える | 必要な型/次元/集計意味の再検証が発生し、単なる表示変更扱いにしない |
| **PDK-03** | 同名Task二件・同一端点間の多重有向辺・閉路を入力 | ID/向き/多重性を保存し、閉路を予定計算前に拒否 |
| **PDK-04** | 図の位置だけ変える／依存辺の向きを変える | 前者は業務意味・計算を不必要に失効させず、後者は計算を再評価 |
| **PDK-05** | 微分方程式解法の許容誤差またはアルゴリズム版を変える | 出力byte一致だけで数値Conformanceと再利用適格を認めない |
| **PDK-06** | 外部R/Mathematica Adapterを未選択にする／選択する | 未選択ならTarget閉包へ常時同梱しない。選択時だけ実Toolchain・権利・実行を確認 |
| **PDK-07** | 同じ業務Meaningで統計表だけ別Presentationへ変更 | Meaning/数値結果は維持し、必要な表示Artifact・開示だけ再評価 |
| **PDK-08** | DSL/JSON Schema風の糖衣を往復し、コメントやSource参照を失う | 正本の要素ID／Source occurrence／保存形式の保全失敗を診断し、黙って整形・受理しない |

**実装への引渡し順**：P0はあくまで#4583/#2831の正式Desktop Subject→B1→B2→B3と、#1251の実Provider呼出し。並行して既存DSL/Query/Standard DataのOwnerがPDK-01/02の最小Fixtureを定義できる。P1では「統計window」の一例をOperation→Go Provider→実Target→独立Expectedまで縦断し、WBS/PERTは多重辺と依存時刻の検証を第二例にする。他領域は正式な標準ライブラリ採用や独立プロダクト化を宣言せず、実例で優位性が示されたものだけToolkitへ格上げする。

**この追補の資格**：他チャットの構想と現在のIssue/PRの状態から統合した、設計上の採否・順序・受入案。新しいToolkitの実装、R/Mathematica実実行、数値適合試験、非Plugin正式Build全体の実Windowsセルフビルドは**NOT RUN／NOT ACCEPTED**。
