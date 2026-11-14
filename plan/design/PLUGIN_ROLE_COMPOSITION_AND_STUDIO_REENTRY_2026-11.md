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


---

<!-- plugin-crosschat-execution-lifecycle-20261114 -->
## 12. 横断プラグインを「記述・選択・実行・観測」の一連の契約として扱う（2026-11-14追補）

この章では、プロセス起動モデル、DSLの自己記述、標準データの意味論、Lifecycle Graph、Script Launch、Studio／SHO／SUI、外部Hostへの接続を、同じ設計原則から整理する。第11章が扱う「**何を生成物に含め、実際に使ったか**」に対し、本章は「**いつ、誰が、どの契約に従って処理を起動し、その結果をどこまで保証できるか**」を扱う。

最初に実装するのは新しい汎用Runtimeではない。既存のApplication Operation、Provider解決、Host、Hook、Builder、Evidenceを実際の製品入口でつなぎ、変更できる部分と守るべき条件を明確にする。

### 12.1 Pluginの「役割」と「実体」を分ける

Pluginは機能の提供単位であり、DSL・画面・実行処理を一つのパッケージで提供してよい。ただし、そのパッケージ名だけを根拠に、記述上の意味や実行権限をまとめて認めない。設計・依存解析では、次の四つを区別する。

| 層 | 主な責務 | Pluginから提供できるもの | 越えてはならない境界 |
| --- | --- | --- | --- |
| **構文（Syntax）** | Sourceの読取り、記法の定義、保存形式・Source位置の維持 | Language Adapter、表記の糖衣、入力補助、診断 | Parse成功を意味受理にしない。原文のコメントや参照位置を無断で失わない |
| **意味（Meaning）** | 型・Role・Facet・Relation・Operationの解釈と制約 | 意味契約の候補、整合規則、専門領域の型・Operator定義 | Pluginが自己申告しただけでOwnerの意味受理を得ない |
| **評価（Evaluation）** | 受理された入力に対する計算・検証・問合せ | 純粋関数、Rule、Window演算、検証Provider | 結果値の一致だけでProvider選択や実呼出しを証明しない |
| **実現（Realization）** | Targetへの生成、実Provider、画面・帳票・外部連携 | Generator、Data Provider、Presentation、Host Adapter | 同梱・表示・Tool公開を実行許可や業務上のEffectにしない |

この四層は新しいCoreの固定列挙ではなく、設計を検証するための見方である。例えば、JSON Schema風の構文拡張は「構文＋意味への対応」、Graphviz風の記法は「意味Graph＋表示投影」、WBS/PERTは「意味Graph＋計算Operator＋表示投影」の組合せとして扱う。GraphのNode ID、Edge ID、向き、多重性、属性、Source位置を保ち、座標などの表示情報を意味Identityの代用にしない。

**採用順**は、既存の汎用構造・Facet／Role・Operatorで表現できるかを確認し、不足した範囲だけをContract／Providerへ広げる。記法が増えるたびに新しい独立DSLやCompilerを作るのではなく、Source⇄構造⇄図／表／文書の往復可能性を一例ずつ検証する。既存[#988](https://github.com/hat47x/tei-platform/issues/988)、[#2619](https://github.com/hat47x/tei-platform/issues/2619)、[#2497](https://github.com/hat47x/tei-platform/issues/2497)、[#4652](https://github.com/hat47x/tei-platform/pull/4652)の判断を優先する。

### 12.2 アプリケーション共通の起動構造は、Targetごとの実現に委ねる

Web、CLI、Desktop、MCP、VUI、バッチ、イベント購読の各入口には、概ね次の共通構造がある。

```text
TargetのHost／Processを起動・初期化
    ↓
入力・要求・時刻・外部イベントを受け取る
    ↓
TriggerをEntry／型付きApplication Operationに結び付ける
    ↓
入力と対象を確定し、必要な現在のGuard・権限を検査する
    ↓
選択された実Provider／生成コードを起動する
    ↓
結果・失敗・作用を記録し、必要な入口に応答する
    ↓
次のイベントを待つ、終了する、または外部Hostへ制御を返す
```

この図は**意味上の共通構造**であり、TEI専用の待受けループ、スレッドモデル、Scheduler、Serverを作る指示ではない。HTTPは応答を返すことが多いが、タイマーやキューでは応答がなくてもよい。CLIの一回起動、Wails/WebViewのイベント、外部SaaSのWebhook、OSの定時起動では、実際のProcess／Event Loop／Transportは異なる。従って、共通Operationの意味と、Eventの配送保証・応答形式・同時実行・順序・期限・再送・停止条件を別に記述する。

外部Hostの認証・認可、対象Resourceの保護、実際のCommitはTarget側の責任であり、Eventを受け取ったこと、TransportでACKしたこと、実行開始、業務Commit、別Observerによる作用確認を同じ「成功」にまとめない。[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[#4502](https://github.com/hat47x/tei-platform/issues/4502)、[#3422](https://github.com/hat47x/tei-platform/issues/3422)を再利用する。

### 12.3 実行の節目とBuildの節目を混同しない

実行入口では、時点ごとに必要な証拠が異なる。**T0 Preview → T1 Resolve/Bind → T2 Admit/Start → T3 Commit/Emit → T4 Observe/Consume**を比較用の節目として使うが、これを新たな固定Runtime状態機械として実装しない。

| 節目 | 既存の担当に委ねる判断 | 主張してよい範囲 |
| --- | --- | --- |
| **T0 提案／Preview** | Source・対象・候補・説明を表示する。AI・SHO・SUIの提案もここへ入る | まだ選択、承認、実行していない |
| **T1 選択／束縛** | exact Contract・Provider・Target・Artifact・入力版を解決する | 選択された実装候補が確定した。実起動は別 |
| **T2 入場／開始** | Hostが現在のScope・Grant・Resource制約を検査し、実Entryへ到達する | 実呼出しが観測できた。Commit成立は別 |
| **T3 作用／発行** | Effect OwnerがCommit直前の現在性と作用範囲を確認し、Receipt等を残す | 確認できた作用範囲だけを報告する。ACK喪失は効果不明を許す |
| **T4 観測／受領** | 別Reader、WebView2、CLI等の実Consumerが対象世代と結果を受け取る | 実際に観測した対象・範囲・時点についてのみ成立 |

一方、Buildは**S0 Intent → S1 Resolver → S2 実Builderの同一性照合 → S3 実呼出し → S4 Artifact／収録 → S5 実Consumer**と区別する。これも既存Builder／ArtifactSetを理解するための検収順であり、Script RunnerとBuild Engineを一つに作り直すものではない。RuntimeのT1を通過したProviderと、Build時のS2で確認したBuilderは、Identityも権限も別である。同じArtifactに複数の論理Pluginが入っても、各Contract・Provider Binding・Grantは統合されない（[#1097](https://github.com/hat47x/tei-platform/issues/1097)）。

### 12.4 Lifecycle Graphは実行順の自由度を表すが、守るべき条件を弱めない

Lifecycle Graphは、一定の手順を固定列挙する代わりに、**型付きNode／Edgeと必須条件を持つBase Graph**に、Pluginが差分を加える方式を検討する。ただし、Graphそのものを新しい統合実行エンジンにはしない。既存Operation、Hook、Command、Providerの実行機構へLoweringできる範囲で検証する。

- **変更できる部分**：追加Validation、純粋な変換、Branch、Target Adapter、Presentation、並列化できる独立処理、独自の診断。
- **変更できない条件**：対象Identity／Source版の確定、必須Guardの到達、型と意味の整合、実Providerとの一致、現在Grant、実作用のFence、Effect／Failure／unknownの保持、開示条件。
- **Patchの適用条件**：Baseに対する追加・置換・削除の対象Node／EdgeをIDで特定し、入出力型、前後関係、必須Guardへの到達可能性、実行主体、呼出し回数・作用範囲を検査する。競合・曖昧な順序・条件の脱落は適用拒否にする。
- **図の編集**：座標変更や折畳みはPresentationだけを更新する。Node／Edgeの意味変更は型付きModel PatchとしてSourceに戻し、受理されるまで実行資格を持たない。
- **実行中の変更**：Graph Patchが構文上妥当、現在のモデルで適用可能、稼働中のRunへ安全に反映可能、は異なる資格である。旧RunのresumeやProvider差替えは現在のHost／Effect Ownerの再確認を要する。

最初は二段階承認、read-only Inspector、before/after Hookを比較する。Guardを満たさない経路から実作用を起こせる場合、Graphに分岐・並列の自由度を追加しない。SCXML／BPMN／DCRとの互換は比較対象にとどめ、共通モデルへの全面統合を急がない。[#4198](https://github.com/hat47x/tei-platform/issues/4198) R0〜R4と、main統合済み[#4589](https://github.com/hat47x/tei-platform/pull/4589) §20を受入の参照先とする。

### 12.5 Pluginの「使える」を一つの真偽値にしない

Pluginの状態は、次の**八つの観測面**を個別に記録する。単一の直列状態遷移や新しい永続Registryを設ける意味ではない。

| 観測面 | 問い |
| --- | --- |
| ① 宣言 | Contract／役割／必要条件が記述されているか |
| ② 導入可能 | 対象環境に対応し、必要Artifactや依存を準備できるか |
| ③ 有効設定 | Workspace／Application／Targetで明示的に有効か |
| ④ Composition選択 | 競合なくexact ProviderとScopeが解決されたか |
| ⑤ Build収録 | 当該生成Targetが要求するMember／Resourceを実際に含むか |
| ⑥ 実利用 | 当該入口から選択した実Entry／Providerを呼んだか |
| ⑦ 現在資格と作用 | Hostが要求時点のGrantを確認し、Effect／Failureを正しく観測したか |
| ⑧ 保持・開示 | 結果やEvidenceを許可範囲で保存・取得・表示できるか |

特に、⑤の同梱と⑥の呼出し、⑥の呼出しと⑦の作用、⑦の結果取得と⑧の開示を混同しない。未選択Pluginの宣言やインストールだけでは、製品が必要とする依存閉包へ自動的に加えない。必要なものを除外していないことと、不要なものが実際に除外されていることは別の証拠である。Pluginの更新が意味契約・実消費入力・Target結果に影響しないなら、変更範囲だけを再評価する（[#2600](https://github.com/hat47x/tei-platform/issues/2600)）。

StudioのInspectorには、可能な範囲でこの八面を**read-onlyの一覧と詳細**として投影する。観測していないものは「未検証」、Targetで実現できないものは「非対応」、観測したが情報が足りないものは「不明」と表示し、宣言から現在資格を推定しない。これを実装するために新たなPlugin管理Hostや第二Resolverを増やさず、既存の選択・Build・実呼出し・Evidenceを参照する。

### 12.6 実例は二つに分け、安全な拡張を先に通す

**第一例：read-onlyの製品機能。** X-TEA三視点／Inspectorを、共通の型付きOperationから通常CLIとStudio/Wailsで呼び出す。双方で同じContractと選択済みProviderを照合する一方、画面表示・Result Envelope・実Processは入口ごとの違いを認める。必要なPluginをStage2へ収録しても、実WebView2が古いJSを受け取った場合は不合格とする。#1251、[#4611](https://github.com/hat47x/tei-platform/issues/4611)、[#4583](https://github.com/hat47x/tei-platform/issues/4583)、[#1124](https://github.com/hat47x/tei-platform/issues/1124)に接続する。

**第二例：実作用を伴うHook。** 既存の`uses`／`Guarded.RunWithCaller`を混在させたまま、Graphの可変BranchからDB更新・通知へ進ませない。宣言による自動CallとImplementation内Callが重複し得るため、まず呼出し主体・Guard順序・回数を一つのOperation／Attemptで測る。作用不明のFailureをwarnで握り潰さず、Receiptを照会する。完全取得した0行と、未取得・一部取得も区別する。既存[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#4602](https://github.com/hat47x/tei-platform/pull/4602)に委ね、[#4589](https://github.com/hat47x/tei-platform/pull/4589)のHG-10〜19を再利用する。

この二例を一つの実行方式に早急に統合しない。読取り専用機能を動かすことと、Commit時の権限・再送・部分作用を保証することは、必要な証拠が異なるためである。

### 12.7 既存Scenarioに引き渡す負例：PLC-01〜09

以下は**独立Expectedの追加候補**であり、この設計書自体が新しいテスト正本になるわけではない。既存のHG／PRA／PDKと重なる実験は再実装せず、差分だけを[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ戻す。

| Case | 一つだけ変える条件 | 判定 |
| --- | --- | --- |
| **PLC-01** | 同じOperationをHTTPと定時Triggerから呼び、定時側には応答先がない | 応答不要でもRun可能。HTTPの応答形を全Targetへ強制しない |
| **PLC-02** | 予定発火／配送ACKだけが成功し、実Providerは起動しない | Triggerの成功と業務実行を別にする。Run／Effect成功を返さない |
| **PLC-03** | GraphにGuard迂回のEdgeを追加する | Patchを拒否し、実作用到達0回。HG-06/HG-10と共通Expected |
| **PLC-04** | Graphの座標だけを変更し、意味Node／Edge ID・Contractは不変 | 不要な意味再受理・Builder再実行を起こさない。HG-07／PDK-04を再利用 |
| **PLC-05** | ProfileはProvider Bを選択したのにStage2はAを同梱、呼出結果は同じ | exact選択と実Build Member・実Callの不一致でFAIL。PPC-04を再利用 |
| **PLC-06** | ソース上では導入済みだが、必要なProviderが実Targetに含まれない | ①〜④の適格性を⑤へ自動昇格せず、Target収録不備を示す |
| **PLC-07** | Commit後にACKを失い、重複Triggerが届く | Effectをunknownとして保持し、lookup-firstの確認なしに再実行しない |
| **PLC-08** | Parserで構文が通るが、Graph多重辺のIdentityまたはNULL／欠損意味が変化する | 構文適合とは別に意味差とSource往復差を検出。PDK-03/08を再利用 |
| **PLC-09** | 既存承認済みGraph／Pluginから新revisionへ更新し、旧Runだけを再開する | 旧Grant／Artifactを現在資格と見なさず、再開不可なら保留・要照合にする |

初回実装で必須とするのは、既存#4583の**選択Builder→実Child→Artifact→Stage2 Consumer**と、#1251の**選択済みProvider→実CLI／Studio入口**である。PLC-01/02の外部Scheduler、PLC-07の実DB Commitは、該当Targetの資格を検証するときに実施する。製品P0の先行条件として広げない。

### 12.8 計画へ反映する実装順と停止条件

| 順位 | 実装・判断のOwner | 完了の境界 |
| --- | --- | --- |
| **P0-A** | #2831／#4583／#943／#1097 | 非Plugin Desktop Build対象、exact Builder実呼出し、N2出力→N3実消費、Stage1実Process→Stage2実Desktop→実WebView2／Bridge受領を同じ因果で観測 |
| **P0-B（並行）** | #1251／#4611／#1124／#951 | Inspectorと三視点が既存Resolverを共有し、CLI／Studioから選択済み実Providerを呼ぶ。A/B誤選択やWorkspace違いを拒否 |
| **P1-A** | #4198／#1129／#1174 | 現行Hook／Graphの型・順序・回数を棚卸しし、HG-10/11/13〜19の必要な負例に適合した実作用Bindingだけ解禁 |
| **P1-B** | #988／#2497／#2619／#4660 | 標準GraphのID保全、Source往復と統計window一例を純粋/read-onlyで実証。新文法Compiler／外部科学Runtimeは後回し |
| **P2** | #1016／#1121／#2390／#3422 | 案件別Profileの静的Composition、外部Hostの利用、イベント入口差・再送・現在GrantをTarget別に適格化 |

**停止条件**は、第二の契約正本・Resolver・Runtime Authority・TEI独自Hostを要する、Graph Patchが必須Guardを外す、意味と図の座標が不可分になる、同じEffectを二重起動する、Build収録だけで実Providerの使用を証明したとみなす、作用不明を成功または安全な再送へ変える、のいずれかである。その場合は新しい抽象を広げず、一件のread-only Operationと、失敗を含む実観測の縦切りへ縮小する。

**この章の証拠資格**：他チャットで進めた実行モデル・DSL／標準データ・Graph／Hook・Plugin／Studio設計を、既存IssueとPRの担当範囲へ整理した**計画追補**である。新たなGraph Compiler、正式Schema、Go/Windows E2E、実Scheduler、実DB Commit、Ownerの業務意味受理を実施・達成したものではない。実装のPASSは各Ownerのexact HEAD・実Target・独立Expected・現在の権限に基づいて判定する。


---

<!-- tei-plugin-qualified-contribution-and-source-roundtrip-20261114 -->
## 13. Pluginの提供物を意味・構文・実装・表示へ分け、拡張の根拠を一つにつなぐ（2026-11-14横断追補）

### 13.1 中心判断：拡張対象は「プラグインの種類」より「契約に対する貢献」である

標準データ構造の意味論、DSL自己記述、Domain Toolkit、Studioの汎用UI、AIとの共同編集を横断すると、「Language Plugin」「Operator Plugin」「Data Provider」「Target Generator」という区分だけでは、実際の依存を十分に説明できない。一つのPluginが複数の機能を提供してよい一方、利用側がそれらを一括導入する必要はない。

そこで**契約に対する貢献（contribution）**を設計上の基本単位として扱う。これは新しいCore型や固定enumの採用ではない。既存のContract／Provider／Trait／Composition／Build／Source Indexで実現できるかを検証するための分解である。

| 貢献の種類（用途別の見方） | 意味上の提供物 | 実現と依存の例 |
| --- | --- | --- |
| **構文・読取り** | 既存意味構造への構文写像、構文版、出典位置 | Parser／Language Adapter。作成・解析時のみ必要 |
| **意味・制約** | 型・Relation・Facet・Role、単位、粒度、必須条件 | Semantics／Rule Contract。意味受理は別Owner |
| **演算・変換** | 意味付きの演算と結果・失敗・unknownの契約 | Pure Operator、Query、統計Window、Solver等のProvider |
| **データ接続** | 対象集合、時点、完全性、変更の意図 | DB／API／FileのProviderとHost Context |
| **画面・文書** | 標準データ構造の編集・図表・説明の投影 | Studio／SHO／Web／PDF等のTarget別Presenter |
| **実行・生成** | 型付きOperationの実呼出し、Build成果物、イベント接続 | Target Generator、Runtime Provider、Host Adapter |
| **検証** | 独立Expected、失敗例、意味上の適格条件 | Driver Scenario／Conformance Fixture。自己申告のPASSとは別 |

Plugin Package／Toolkit／生成Artifactは、これらの貢献を配送・選択しやすく束ねたものにすぎない。単一Packageに複数貢献がある場合でも、**それぞれのSource版、Contract、必要Target、実選択、Build消費、実呼出し、現行権限を別に追跡**する。同一バイナリへの静的合成は論理的なAuthorityの共有を意味しない。既存[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1016](https://github.com/hat47x/tei-platform/issues/1016)／[#1097](https://github.com/hat47x/tei-platform/issues/1097)の解決・Bindingに乗らない新Registryを作らない。

### 13.2 記法の拡張を受け入れる最低条件：同じ意味へ戻り、原文を壊さない

JSON Schemaのような外部Schema、Markdown表、関数チェーン、GraphvizやSQLは、同一の業務概念を扱うための**入口・投影**として使える。ただし、単に構文を読めるだけでは、既存Sourceを置き換える資格を持たない。

最初のReferenceでは、次の一往復を固定する。

**保存された原文（byte・Source occurrence・版）→ 構文解析（位置つき）→ 標準データ構造／意味Role候補 → 型付き検査 → 汎用Studioの表／図／Property表示 → 一件の局所編集 → 保存形式を保持した差分 → 再Parse／再投影**。

- 入力の**値・未設定・明示NULL・完全な空集合・未取得・部分取得・permission-denied**を区別する。表記の短縮を理由に意味を埋めない。
- Element ID、Graph Edge ID、向き・多重性、単位・精度・時間範囲、出典とSchema／Contract版を保持する。別表現へ写した結果、保持できない情報があればlossyと明示し、元へ戻せない編集は拒否する。
- コメント、空白、順序、Markdown見出し・Fence、SQL方言、Graphviz固有記述など、保存形式として意味を持つ部分を無断整形しない。意味上の同値と原byteの同一性は別に判定する。
- 表の1行や図の1本の辺だけが編集対象として一意に決まることを検査する。集計・Graph→Tree・圧縮表示などの**多対一投影**から対象を推測して更新しない。
- 元Sourceへ戻れることと、編集が業務上受理されることは別。AIやStudioが出した候補は、現在のSource／対象Revision／Owner受理境界を通るまで候補に留まる。

[#4359](https://github.com/hat47x/tei-platform/issues/4359)が確認した現行ParserとSource Anchorの**二重字句走査**は特に重要な実装上のGapである。新しい記法をParserに追加しても、AnchorやSource Indexが追随しなければ、Studioが「正しい場所を編集できる」とは言えない。最初の実装は、当該IssueのS0（独立Expected）→ S1（最小構文）→ S2（値と位置を一度のParseから得る）を採り、任意文法CompilerやUnicode識別子を先行導入しない。意味受理、Language Plugin選択、Source保存は[#988](https://github.com/hat47x/tei-platform/issues/988)／[#1018](https://github.com/hat47x/tei-platform/issues/1018)／[#2621](https://github.com/hat47x/tei-platform/issues/2621)のOwnerに委ねる。

### 13.3 変更の伝播は「何が変更されたか」で決める

Pluginの変更を一律に「プラグイン更新」と扱うと、必要な再検査を省略する一方、不要な全体Buildを招く。実際に消費する対象と契約を基準に、少なくとも次の差を区別する。

| 一因子だけの変更 | 再評価が必要な範囲 | それだけでは変わらない範囲 |
| --- | --- | --- |
| ヘルプ・図の座標・表示文言 | 該当Presentation／Disclosure／生成表示Artifact | 業務Meaning、純粋Operator、関係ないProvider |
| Sourceの保存形式・コメント | Source Index／原文保全／必要な表示 | 実消費されないSemantic／Target。byteが変わった事実は別に保持 |
| DSL文法やParser版 | Syntax適合、Source Span、構造への写像 | Runtime Grant。Parser更新だけで意味を再受理しない |
| 意味Rule／型／Facet版 | 意味適合、依存するOperation／Scenario、必要なLowering | その意味を消費しないUI／Artifact |
| Operator／実Provider版 | Selected ProviderのConformance、必要なBuild／実結果 | 他のProvider・他Targetの現在資格を自動失効させない |
| Pluginの配送物・Builder版 | 影響するBuild入力閉包、Artifact／Member／Loader一致 | その配布と無関係なAccepted Meaning |
| Host Grant／Scope／Disclosure変更 | 現在の実行可否、保持済み結果の開示可否 | Sourceの歴史的な意味受理、過去の実行の事実 |

この表は**判定規則の候補**であり、ハッシュが一致しただけで安全な再利用を認めない。実消費入力と依存集合、Target／選択Provider、Buildの実工程、現在のTrust／権限が揃って初めて、そのClaimに必要な再利用資格を得る。反対に、無関係なPlugin追加やlayout-only変更で全Artifactを無条件に失効させない。[#2600](https://github.com/hat47x/tei-platform/issues/2600)／[#2595](https://github.com/hat47x/tei-platform/issues/2595)／[#3267](https://github.com/hat47x/tei-platform/issues/3267)で実測する。

### 13.4 StudioとAIへの公開面は、汎用Inspectを基準にする

初めて導入したPluginやRoleでも、専用画面を作らずに**型・構造・Relation・出典・検査結果・実選択状態**が読めることを品質基準にする。機能固有のModelerは、その汎用基盤の上で認知順に情報を配置する拡張とし、原モデルとは独立の意味正本にしない。

編集操作では、**画面の選択対象、編集中Buffer、保存済みSource、受理済みMeaning、生成対象、実際のConsumer、現在のGrant**を分離する。画面で選択した行・図形・Source spanを、安定したQualified ID／Revisionへ対応付け、後から届いた別対象・旧版の非同期応答は現在の編集結果として適用しない。情報が不足して対象を確定できない場合は、候補と出典を表示し、編集や実行へ自動昇格させない。

AIに許すのは、Source／Graph／UIの候補提案、影響分析、独立Expectedの作成補助、適切なOwnerへの判断依頼である。**AI生成テキスト、SHOの図面、MCPのTool一覧、プラグインの自己記述、外部資料の命令**は、それ自体では現在の書込み権限や正式な業務意味の根拠にならない。Provider呼出しは既存の型付きOperationとHost Authorityを通し、外部Harnessの作業セッションも対象製品のGrantを継承しない。[#1124](https://github.com/hat47x/tei-platform/issues/1124)／[#2516](https://github.com/hat47x/tei-platform/issues/2516)／[#3422](https://github.com/hat47x/tei-platform/issues/3422)に接続する。

### 13.5 三つの実証で、共通化が必要な範囲を決める

抽象化の是非は、次の三種類を同一の検証票で比較して決める。物理Plugin方式は**Go単一バイナリへの静的合成**を標準とし、別Process／Wasm／Target固有Extensionは実際に必要なときだけ選択する。

| 代表例 | 最小の機能 | ここで確認する境界 |
| --- | --- | --- |
| **A：純粋な表現・変換** | 記法／Graphの一件の読み取り・Source往復、統計Windowの計算 | 型・意味・不変条件・独立Expected。Build時専用Pluginは実行Targetへ常時同梱しない |
| **B：外部データのRead** | 同じ型付きFunctionが実DB／APIから選択対象のRowsetを返す | Provider選択、Source・Schema・粒度、取得完全性、現在Grant、実Consumer受領。全件取得を推定しない |
| **C：作用を伴うHook／Command** | Guardが成立した場合だけ単一のEffectを実施し、Receiptと別観測を残す | Guard先行、実Call回数、Target Scope、Commit前の現在性、ACK喪失・不明結果・重複Triggerの停止 |

Aの成功をBの実外部Data Provider適格へ、Bの読取り成功をCの本番Effect保証へ昇格させない。Aは[#4359](https://github.com/hat47x/tei-platform/issues/4359)／[#2497](https://github.com/hat47x/tei-platform/issues/2497)／[#4660](https://github.com/hat47x/tei-platform/issues/4660)、Bは[#2586](https://github.com/hat47x/tei-platform/issues/2586)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)、Cは[#1129](https://github.com/hat47x/tei-platform/issues/1129)／[#1174](https://github.com/hat47x/tei-platform/issues/1174)／[#4198](https://github.com/hat47x/tei-platform/issues/4198)の既存Scenario・Ownerを再利用する。

### 13.6 既存Expectedへ引き渡す差分テスト：PCX-01〜10

本章は新しいテストRunner／正式Schema／製品Gateを設けない。下記は[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ委譲する**追加候補**であり、既存PPC／PDK／HG／PLCと重なるものは同一Fixtureを使う。

| Case | 比較する条件 | 独立Expected |
| --- | --- | --- |
| **PCX-01** | Schema風記法のSource解析は通るがAnchorが別行へずれる | Source span不一致。Studio編集への適格化は拒否（#4359 S2） |
| **PCX-02** | 表で2本の同名・同端点Graph Edgeを1行に畳む | 多重性が失われたと診断し、原モデルの単一Edge更新を拒否 |
| **PCX-03** | 完全な0行と、未取得・partial・permission-deniedを同じ空表へ表示 | 表示が同じでも読み取り保証と不存在Claimは別。見えないデータを書き戻さない |
| **PCX-04** | 説明だけ変更／意味Rule変更／選択Provider変更を順に試す | 再解析・意味検査・BuildとRuntime再資格化の範囲を区別し、局所再利用を測る |
| **PCX-05** | 画面選択A→B後にAの旧応答が遅れて届く | Bに旧Aの出典・結果・編集候補を適用しない。Aの過去結果自体は偽造しない |
| **PCX-06** | Parser／Providerが追加されても、Workspaceでは未選択 | 未選択実装を黙ってProviderへBindしない。意味やBuildを不必要に失効させない |
| **PCX-07** | 同一Packageの構文AdapterはBuilderで必要、実ProviderはStage2で不要 | Build時実消費とStage2収録を分け、無条件同梱を要求しない |
| **PCX-08** | 同じContractに適合するA/BのうちBを選び、実TargetはAを呼ぶ | 数値・JSONが一致しても失敗。exact Provider実Callを必要とする |
| **PCX-09** | AI／外部資料が「受理済み」「実行許可」と主張する | 自己申告をGrantやMeaning受理に使わず、Host／Ownerの現在判定に戻す |
| **PCX-10** | Hookによる更新Commit直後にACK喪失、同一Triggerを再送 | lookup-firstが必要。再送可否unknownを通常失敗や安全な再実行へ変えない |

### 13.7 優先順位とOwnerへの引き渡し

- **P0の変更なし**：[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#4583](https://github.com/hat47x/tei-platform/issues/4583)が正式な非Plugin Desktop Build→Stage1→Stage2→実Windows Consumerを完成させる。並行して[#1251](https://github.com/hat47x/tei-platform/issues/1251)／[#4611](https://github.com/hat47x/tei-platform/issues/4611)が通常CLI／Studioの選択済み実Provider呼出しを検証する。
- **P1-A（DSL・意味・表示）**：#4359の位置つき最小構文、#2497の可逆／不可逆Projection、#1124の汎用Inspectと固定選択を、まずpure/read-onlyで一往復させる。専用Grammar Engineや各分野の専用Editorを前提にしない。
- **P1-B（Plugin／Toolkits）**：#951/#1016のContribution別選択→#1121のTarget Lowering→#2600の局所Build再利用を、一つの統計WindowまたはGraph表示で確認する。全Plugin動的ロードや外部科学Runtimeの先行採用をしない。
- **P2（実作用・外部Host）**：#1129/#1174の現在GrantとEffect、#2390のTEI非依存の外部Consumerを個別に適格化する。pure/read-onlyの実証を本番DB Commitに昇格させない。

**撤回条件**：必要な構文のたびに第二のMeaning Source／Parser Anchor／Resolverを増やさないと成立しない、Plugin提供物を小さく選択できず全Targetへ常時同梱する、保存時に元形式を壊す、Graphの表示上の一致を対象Identityや業務意味の一致とみなす、または不明な実作用を成功と記録する場合は、抽象の適用範囲を縮小する。

**証拠資格**：既存Issue／PRと他チャットの設計を照合した計画の追補。新たなParser・Operator・Plugin Runtime・Windows E2E、正式Schema、Owner意味受理をこの章で達成したものではない。現行実装に存在するもの、Draft、未検証のものを各担当で区別する。


---

<!-- tei-plugin-phase-closure-bootstrap-20261114 -->
## 14. 必要な機能からBuild・実行時の依存を導き、Pluginの配布と自己記述を段階的に実証する（2026-11-14横断追補）

本章は、横断Plugin設計の残る実装上の問い――「必要な機能を指定したとき、どの実装を、どの工程・Targetに含めればよいか」を具体化する。第13章の貢献別整理を、案件ごとの構成選択、実装義務、Build、配布、外部Host、Studio自己再構築へ接続する。ただし、正式な新Manifest／ABI／Graph Engineや第二Resolverを採用する章ではない。

### 14.1 三つの関係と、四つの工程を分けて扱う

TEIの「依存」は一種類ではない。少なくとも次の三つは、元のSourceと同じ識別子・版へ戻れるようにしながら、別の関係として保持する。

| 関係 | 問い | 正本・適用範囲 |
| --- | --- | --- |
| **意味上の要求** | このOperation、Trait、Data、表示が成立するには何が必要か | 受理済み意味、Contract、Role、Trait。意味受理はOwner |
| **構成上の選択** | 対象Application／Targetでは、どのContractをどのProviderに担わせるか | #1016のApplication Profile、#951のexact Provider解決。競合は拒否 |
| **物理的な実現** | どのSource、Builder、Artifact Member、実Process／Loader、Consumerが使われたか | #943／#1097／#2600のBuildと、#1251／#1174の実Call・Effect |

これらを単一の万能Graphへ混ぜない。各関係を安定IDと版で結び付け、Modelerからは必要に応じて重ねて可視化できるようにする。とくに「意味上のRelationがある」ことから、DBの自動JOIN、Pluginの自動ロード、外部Resourceへのアクセスを暗黙に許さない。

物理的な実現は、**作成時（Authoring）／構築時（Build）／アプリ実行時（Runtime）／結果受領時（Delivery）**の工程ごとに必要なものを分ける。例えばDSLのParserやGeneratorはAuthoring／Buildだけで使い、実行時の業務アプリから除外できる。一方、生成済みアプリのInspector／Operation ProviderはRuntimeで必要なら収録・実呼出しを証明し、WebView2のJavaScriptや帳票資産はDeliveryまで追跡する。

### 14.2 Compositionの解決を、既存機構へ落とせる順序に固定する

ここで示す手順は、既存のResolver・Builder・Hostの**呼び出し順と検収項目**であって、別の製品サービスではない。

1. **必要な機能を確定する。** Application Profile、受理済みOperation、required Trait、Target・OS・配布制約から、必要なContract／RoleとScopeを取り出す。optional／advisoryの役割も区別する。
2. **実装をexactに選ぶ。** #951の既存ResolverでContract版・fingerprint・Conformance、選択済みProvider、Target対応を照合する。候補が複数で決まらない、Providerが未導入、Scopeが違う場合は、ロード順やlatestへ黙ってフォールバックしない。
3. **Traitの履行を点検する。** required Traitを、標準生成、Provider、手動実装などのいずれが満たすかを#3278の既存契約で検査する。宣言だけ、Hintだけ、Function名が似ているだけでは履行済みとしない。まだ証拠がなければ未検証とする。
4. **工程とTargetごとの必要集合を導く。** 選ばれた貢献をAuthoring／Build／Runtime／Deliveryに振り分ける。論理Pluginと一つの物理Artifactの1対1を強制せず、必要なArtifact Member、Host Import、Resource、外部依存だけを列挙する。できないTargetはunsupportedと明示する。
5. **実際の構築・呼出しを照合する。** #943／#1097のBuild ExecutionとArtifactSet、#1251の実Callで、宣言した入力と実消費byte、選択Builderと実Child、選択Providerと実Entryを別々に確認する。Stage1がStage2を構築した場合、親子Attemptの因果関係を保存する。
6. **現在の利用資格を判定する。** 実際のHost Grant、対象Revision、Effect、Disclosure、Consumer受領はBuild完了とは別に確認する。過去のBOM／Fixture／Receiptを、失効した現在権限の代わりに使わない。

**出力は一つの合否に畳まない。** 契約選択に成功しても実Artifactへの収録が未確認なら「選択済み／収録未検証」であり、実呼出しが確認されてもCommitや業務上の受理まで示さない。部分的な証拠、不明、非対応、拒否、未実行を区別して、既存InspectorやScenarioの読取り情報へ投影する。

### 14.3 最小構成と検証済み構成は別の成果物である

「このPlugin群を入れれば機能する」という提案と、「この組合せ・版・Targetで実際に検証した」という事実は異なる。構成の推奨には、既存[#3267](https://github.com/hat47x/tei-platform/issues/3267)が所有する **RQ-037 Verified Composition BOM** を使用する（RQ-035はData Source Workbench、RQ-036はDomain Model Map）。新しいBOMの正本は作らない。

BOMが保持・参照する根拠は、少なくとも要求したContract、解決済みProviderとScope、Target、必要なArtifact Member、Fixture／Expected版、実際のBuild／Call／Consumer、検証の被覆範囲である。個々のPluginが別々に合格しただけでは、それらを組み合わせた製品が合格したとは扱わない。バージョン行列は全直積を常に実行するのではなく、境界条件・変更影響・必要な組合せに合わせて選ぶ。

初回の実証は、既存のread-only Inspectorに加えて、同じContractに対する二つのProvider、二つのConsumer、二つのPlugin構成（必要／不要）を使い、**選択・Build収録・実Callの対応が説明できるか**を確かめる。対象が増えたときの計算量・依存数・診断時間も記録する。

### 14.4 配布方式は、必要な隔離と更新条件から選ぶ

標準配布は引き続き**Go単一バイナリへの静的Composition**とする。ただし、Pluginの論理契約は同じでも、実行場所・更新頻度・隔離要件により適切な物理方式は異なる。配布方式の選択を、Coreの固定Plugin種別へ押し込まない。

| 方式 | 最初に採る条件 | 採用前に確認すること |
| --- | --- | --- |
| **静的合成** | 頻繁な無停止更新を要さず、信頼できる標準機能を一括配布する | 必要なProviderだけが実Targetに含まれ、現在のHostで正しく呼べること |
| **別Process** | 障害・Resource・Toolchain・更新単位を分離する理由がある | 起動する実image、RPC／stdio等の契約、隔離の実強制、停止・再送・Effect不明の扱い |
| **Wasm Component等** | 同じ実装を複数Hostで使う利点が、Adapter／Toolchain負荷を上回る | 実Componentを実Hostで呼び、型・意味・Host Importの差を独立検査。#1861の採否を先取りしない |
| **Target固有Library／Extension** | Java/.NET/Browser/外部アプリが既存Hostの方式を必要とする | Contract資産と薄いBindingを独立利用できるか。Hostの配布・認可・ライセンス規則を守れるか |

MCP／VUI／AIハーネスは、原則として型付きOperationを公開・利用する**Consumer／Adapter**であり、あらゆるPluginを実行する共通ABIではない。外部製品へは、可能であればContract／Fixtureを実装から分離して提供し、Provider実装やUIは必要に応じて選べるようにする。外部Hostからの再利用は[#2390](https://github.com/hat47x/tei-platform/issues/2390)のTEI非依存Consumerで検証し、TEI専用の常駐サービスを必須としない。

これらの比較に、更新頻度、信頼境界、Target対応、起動／呼出しの実測、成果物サイズ、ライセンス、配備・RollbackとEffectの区別を用いる。最初から全方式を製品化せず、**純粋変換→外部Data Provider→信頼境界を越える実作用**の三例に対して、必要性を反証する。

### 14.5 自己記述DSLは、段階ごとの権限を切り分けて導入する

TEI自身のDSLやPluginの定義を標準データ構造で表せれば、Studioから言語やOperatorを検査・編集し、Targetへ生成できる可能性がある。ただし、**自分自身を読めたことが、その定義を正しいと承認することにはならない**。自己記述は既存の固定した解釈基盤を足場に、次の順で検証する。

| 段階 | 検証すること | ここでは認めないこと |
| --- | --- | --- |
| **B0：固定Seed** | 現行Parser／Canonical／Source Indexを使い、Language・Operator・Roleの最小宣言をデータとして読める | 新定義から現行Parserを無条件に置換 |
| **B1：Language選択** | WorkspaceがLanguage ID・版・Implementation pin・対象Sourceを指定し、#1018が決定論的に選ぶ | 未宣言Sourceの勝手な解釈、Provider不在時の黙ったfallback |
| **B2：解釈候補** | 選択Parserが位置つきSourceと意味候補を返し、現行固定Seedと比較できる | 構文Parseの成功を意味受理に昇格 |
| **B3：意味採用** | Contract／Rule／Traitの検査とOwner受理、版・出典を記録できる | AI候補・自己申告だけでProvider／Grantを承認 |
| **B4：生成と実利用** | 受理された定義に対応するBuilder／ProviderがTargetを生成・実呼出しし、独立Expectedと比較できる | Builderの成功だけでStudio全体の自己再構築・本番資格を宣言 |

実装順は、既存[#4359](https://github.com/hat47x/tei-platform/issues/4359)の最小構文・Source Anchor統合、[#1018](https://github.com/hat47x/tei-platform/issues/1018)のWorkspace Language選択、[#2620](https://github.com/hat47x/tei-platform/issues/2620)のRole、[#2831](https://github.com/hat47x/tei-platform/issues/2831)のセルフビルドと照合する。**B4をStudio DesktopのP0完成条件へ追加しない**。まず現行Seedによる製品実再構築を成立させ、その後に選択的な自己記述部品へ置き換える。

### 14.6 追加の独立Expected候補：PCC-01〜10

新しいExpected StoreやGateは設けない。PCCは新しい独立テスト群の採番ではなく、[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ渡す**工程別の追加assertの索引**である。既存PCX／PPC／HG／TC／VCBと重なるFixture・期待値を複製せず、既存の試験に不足するTarget Member、必要工程、実Consumerの検査だけを追加する。

| Case | 変える一因子 | 独立に確認すること |
| --- | --- | --- |
| **PCC-01** | PPC-04／PCX-08のA→B切替に、Profileのrequired依存閉包を加える | 既存の実A拒否に加え、選択済みBの必須MemberがBuildへ伝播したかを別検査 |
| **PCC-02** | PCX-07のPackageを、CLIだけ／Studio UIありの二Profileへ展開 | ParserのBuild消費、UI不要時のMember除外、必要時の実Consumerへの収録を別々に検査 |
| **PCC-03** | TC-01/02のTrait充足経路を、選択ProfileとTarget別に切替 | 必須Traitの適合診断を再利用し、選択されなかった履行実装を対象のBuildへ混入させない |
| **PCC-04** | PPC-02の必須Provider欠落試験に、Memberは存在するが必須Host Importが欠ける条件を追加 | 実Loader／Invokerがunsupportedを返し、Build収録成功を実呼出し成功にしない |
| **PCC-05** | Runtime Providerを同一Artifact内の別Memberへ束縛 | Artifact Digestが同じでも、論理Pluginと実Entryの違いを判別する |
| **PCC-06** | Contractは固定し、配布をGo静的合成から別Processへ変更 | 型・結果・失敗意味を保持し、Host制約／実Process・Effect資格は別検証 |
| **PCC-07** | VCBの既存A/B組合せFixtureで、単体適合済みのまま共通依存だけ版衝突させる | 既存BOM判定を再利用し、選択閉包の未解決箇所と組合せ不適格を明示 |
| **PCC-08** | #1018のLanguage不在・版違いFixtureに、Seed B0と候補B2の差を追加 | 暗黙fallback拒否を再利用し、Parse可能／Anchor適合／意味採用の三資格を混同しない |
| **PCC-09** | IB-07／DI-04の既存依存差分試験で、選択Predicateから依存辺が削除・追加される | 未選択Providerを不必要に再Buildせず、追加・削除された実依存辺の閉包は再検証 |
| **PCC-10** | VCB／PCFの旧BOM Fixtureで、過去Artifactは同byteのまま現在Host Grantだけ失効 | 履歴のBuild Conformanceを維持し、現在の実行・開示資格だけを拒否または未検証へ戻す |

PCC-01/02/04/05は#4583/#1251の製品経路と合わせて確認するが、**P0のStage2 Buildを遅らせる追加Hard Gateにはしない**。PCC-03/07/09は#3278/#3267/#2600で参照評価、PCC-06/08/10は実Provider／Language／外部Hostの後続資格とする。

### 14.7 完了条件と設計を縮小する条件

第一の完了条件は、一つのread-only機能について、**Application要求→選択Contract／Provider→必要工程→実Build Member→実Provider呼出し→Consumer受領**を、同じ対象とVersion／Attemptで追跡できること。次に、不要な貢献を除いた静的Buildと、二つのConsumerからの同じ契約への呼出しを検証する。実作用と別Hostは、その後に独立した受入へ進める。

同じ内容を成立させるために第二のResolver／契約正本／Host／認可Engineが必要になる、定義と物理Artifactを一対一に固定しないと成立しない、Buildだけで実利用を主張しなければならない、表記の違いで意味IDを作り直さなければならない、またはProvider選択がロード順に依存する場合は、共通抽象の採用を保留し、既存の一Function・一Provider・一Targetの境界へ戻す。

**今回の資格**：現行の設計・Issue・他チャットの判断を統合した実装順と検証条件。正式なPlugin Manifest／ABI、汎用Graph Compiler、動的Loader、実Wasm Component、外部Hostの隔離保証、自己記述DSLの正式採用、実Windows Stage2／実DB更新／Owner意味受理を達成したものではない。

## 15. 依存先が公開する図解を契約に従って再利用する（2026-11-14横断統合）

Pluginは自らのFunction/Modelを実装するだけでなく、利用側がその意味を理解・検証・表示するための図解・型付きRelation・説明・診断を**選択的に公開**できるようにする。依存関係の上位にあるPluginの情報も、途中のPluginが再公開を許可された範囲で継承的に利用できることを要件とする。基本設計と検証の詳細は[プログラム実行モデルの§24〜27](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)に接続する。

この「継承」は実装言語のクラス継承でも、Pluginフォルダの全内容をコピーすることでもない。既存#951のContract/Provider・#2620の意味Role・#2497のProjection・#1016/#3267のexact Compositionと、#1124のSource参照をつなぐ**公開済み情報の合成**である。

### 15.1 契約で区別する権利と情報の範囲

| 契約で区別する事項 | 判断すること | 不正な昇格 |
| --- | --- | --- |
| 公開対象 | 具体的な型付きNode/Edge、Role、Diagram Facet、必須Obligation、説明、診断、テスト参照のうち何を公開したか | 表示Facetを公開しただけで元の全Source・秘密Fixtureも公開したとみなす |
| 受け手と用途 | Plugin/Consumer、Scope、Target、read-only表示、派生図の生成、再公開、編集提案などの許される用途 | 内部閲覧可から任意の外部AI/SHO配布可へ拡大 |
| 版と出典 | 公開元Plugin/Contract/Subject ID、Facet内容の版とdigest、Source/Meaningとの関係、依存の選択経路 | 同じ名前・同じ画像・同じversion文字列なら資格が同じと判断 |
| 合成条件 | 複数親の重複と競合、再公開可能な依存辺、許されたPatch/Overlay、保持すべきObligation | 登録順で上書き、Guard非表示で義務消失、同名Nodeの無断統合 |
| 機密と更新 | 非公開属性の除去、現在Disclosure、公開縮小・撤回と古いCache/説明の扱い | 一度見えた図から現在の秘密情報を再送出 |
| 実現資格 | 必要なProvider/Generator/Artifact/Consumerへの選択と適合、現在Grant | 図の閲覧・継承だけでPluginの実呼出しと外部Effectを許可 |

既定では**直接公開されたFacetのread-only再利用**から始める。依存Bが上位Aの公開図を表示できても、Bの利用側CへAの図を自動再公開する権利は生じない。必要ならA側の再公開条件とB側の明示的なexportを両方解決する。参照に必要な契約情報が取得できない場合は、継承を確定させずunresolved/permission-denied等の既存診断として扱う。

外部Pluginや研究Toolkit由来の図解・説明には、利用許諾・配布範囲・出所も必要に応じて添える。公開されたMetadataが自由な複製ライセンスを意味するわけではない。実プラグイン実装の配布資格と、図面の再利用・外部送出資格は別とする。

### 15.2 上位図を使った差分合成の実例（契約候補）

次の三層の例は仮想的なSubjectであり、実在するPluginの正式IDや実受理Contractではない。

- **共通Lifecycle Plugin A**：Entry→Admit→Invoke→Observeの意味上の辺と、Authority/Effectの必須条件を、許可した範囲で公開する。
- **業務Plugin B**：Aの公開Facetを固定参照し、受注検証・受注確定・通知など業務固有のNode/Edgeを自分のSourceで追加する。AのGuardを削除・迂回することはできない。
- **Studio/文書Plugin C**：Bが明示的に再公開したFacetだけを使って図・表・説明文を導出する。CからBへの局所的な表示変更はView Metadataに、業務変更はOwnerのSource Change Proposalへ戻す。

この例では、合成済みの一枚の図面があっても、**意味の由来はA/B、Viewの構成はC、実行資格は選択Host/Provider**と区別する。BのSourceをCが所有することも、AのEffect権限がCへ伝搬することもない。A/BのFacetが新しい版になったときは、影響するNode/Edge/Consumerだけ再投影し、必要がある場合に限りBuild/Conformanceを再確認する。

### 15.3 実装先と優先順

- **既存Resolver/Compositionを使う**：Contractが公開するFacetと利用側の要求を#951/#1016で解決し、依存の選択結果・再公開範囲を確定する。第二のPlugin RegistryやScope解決器を作らない。
- **既存の標準データへ載せる**：Value/Record/Relation/GraphとRoleを用い、Source ID/Revision、Edge identity・多重度、Presentation Metadata、未表現情報を分離する。専用の継承Graph型を初期導入しない。
- **CLI/JSONで先に表示する**：上位Facet＋局所Overlayと、公開しなかった情報・未解決の依存を、read-onlyの照合表で比較する。実Studio Graph Editorや書込み連携は後続。
- **必要なときにだけTargetへ渡す**：公開Facetの参照と、実Builder/Generatorが消費する入力を分ける。Studio表示のみのFacet更新で全Go binaryを無条件に作り直さない。
- **対象側のAuthorityは別に判断する**：Hook/Guard、DB更新、AI Tool、外部Host実行、開示の現在資格を、上位図の可視性から推論しない。

独立Expectedは同一のA/B/C依存構成で、（a）公開Facetを継承できる、（b）非公開AttributeはCへ届かない、（c）Bからの再公開なしではAの図がCへ出ない、（d）上位Obligationを子Patchで消せない、（e）Facetの表示だけの変更で意味Buildを過剰失効しない、の五つを比較する。これは[PX-38〜43・PX-47](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)の既存Issue向け提案と同一のFixtureを使い、別テスト正本を作らない。

**採用判断を保留するもの**：公開Facetの正式Schema、権限表現、複数親の優先規則、実Targetでの具体的な再公開・配布方法。上述のContract解決と負例が先に成立するかを見て、既存Ownerが決める。

## 16. Plugin自身をTEIで生成する比率を高める（2026-11-14横断統合）

最初にStudioが完成している必要はない。実TEI CLI／MCPから、Markdownの表・図解とDSLで表された意味を解決して、**Pluginの一部を実際に生成できるか**を先に確かめる。完成したPluginを単にGoへ委譲するだけでは、TEIの表現力やセルフビルドの進展は測れない。プラグインの自己構築は、既存#2831/#943/#2620/#1121/#951のBuild/Role/Providerと共通の経路で段階的に進める。

| 小さな到達点 | 必要な成果 | 合格としない例 |
| --- | --- | --- |
| **P-A Plugin意味の可視化** | 一PluginのFunction・依存・公開Facetを、Source-backedな表/Graph/説明へ投影 | 図が描けたが元Subject/Relationへ戻れない |
| **P-B TEI定義から実コード生成** | 計算/Validation/Ruleの少なくとも一処理を、TEI表・式・図の受理済み意味から実Go生成し独立Expectedで照合 | Go Sourceのファイル名と呼出し先をDSLに書いただけ |
| **P-C Plugin内のTEI生成範囲拡大** | 同じPluginで生成部分と手書きProvider/Target Adapterを区別し、依存と再生成条件を確認 | 手書き規則を隠れた別Go処理へ移しただけ |
| **P-D Pluginの再構築** | TEI/Pluginの実Builderで次世代Plugin Artifactを生成し、実Load/Call/Consumerまで確認 | テスト環境が生成ファイルだけ出した／別Headの証拠を合算 |
| **P-E Desktop全体へ接続** | 上記実証済み部品を使い、正式なStage1→Stage2 Studio Desktopと実Windows受領へ接続 | 局所Pluginの成功をDesktop全体セルフビルドと呼ぶ |

運用順としては**P-A→P-BをHeadlessで先行可能**とし、P-C/P-Dを段階的に増やす。現在の製品出口としてのDesktop Stage1→Stage2（#2831/#4583）は最優先のまま維持し、Headlessの小実証を理由に完成条件を変更しない。MCPは共通APIの薄い入口であり、専用のAI実行Hostや別Builderは作らない。

成果を測るときは、生成Goの行数よりも、**意味処理のうち実際にTEIの図表・DSLから導出された割合、手書き業務・検証規則の削減、同じ意味定義のTarget・文書・テストへの再利用、Plugin自身の自己生成範囲、変更から実Consumer確認までの時間**を区別する。最初に固定版の基準値を測り、測定前から高い割合を約束しない。単純にGoの呼出し先を増やしてTEIファイルを短くすることは改善とは扱わない。

初回は公開Facetを持つ**一つのpure Function/Validation Plugin**を題材に、契約付きの継承図→受理済み意味→Go生成→実Provider→CLI/独立Consumerを一つの版に束縛する。後続で別Pluginが同じFacetを使う構成を追加し、上位の契約変更時に生成物とViewのどちらを再生成するかを分けて検証する。この順序なら、Pluginの表現能力を実装で確かめながら、TEI定義そのものを段階的に簡潔にできる。

**状態**：設計・計画。上位公開Facetの実継承、TEI-native Plugin実生成、P-A〜P-Eの実行、正式なPlugin Manifest/ABI、全Windows Stage2、Owner意味受理は **NOT RUN / NOT ACCEPTED**。関係する反例は共通のPX-38〜47として引き渡し、別の試験正本を増やさない。

## 17. Application GraphをPlugin拡張点の対象選択に使う（2026-11-14横断統合）

アプリケーション全体の画面・Route・Entry・Operation・Function・Provider・Build Artifactの関係を既存の意味Roleと型付きRelationで表示すれば、計測・監査・診断などの横断機能をどこへ適用するか、設計者が確認しやすくなる。基本契約は[実行モデル§28〜30](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)へ集約し、このPlugin計画では**何を選択・生成・実呼出しするか**だけを扱う。

- **Extension Pointを先に解決する**：対象はGraph上の任意座標やラベルではなく、Role/Subject ID、対象ProfileとTarget、既存Lifecycleの位相、適用条件で確定する。新しいPluginが既存Guardを任意の名前のHookで迂回することは認めない。
- **ApplicabilityとCompositionを分ける**：Graphによる対象選択は候補であり、#1016/#951が現在のContract・ProviderとCapabilityをexactに解決して初めて構成済みのPlanになる。未選択PluginをBuildやRuntimeへ暗黙混入させない。
- **生成と実利用を分ける**：Browser向けのScript、Server側Interceptor、CLI診断、Build時のコード差し込みは異なる実現方式である。#1121/#2090へ渡す。実Artifactへの収録・実Load・実Call・Consent・Host Grant・独立Consumer受領は別に判定する。
- **安全と失敗時の扱いを分ける**：任意の計測が失敗した場合の欠測診断と、業務更新に必須の監査Guardが失敗した場合の拒否は同じではない。Hookの呼出し主体、Guard先行、重複・結果不明は#1129/#1174の既存検証に従う。
- **対象外の製品を尊重する**：完全NativeなB2C画面やHeadless APIでは、TEI共通UI/Router/JavaScriptの常時介在を要求しない。Adapterが不要なRouteも適法な対象として扱い、#3509の非拘束原則を維持する。

第一例は、一つのRouteと一つのEntryに対し、read-onlyの対象適用表を出す。次に任意のログ・計測用PluginのSource-backedな一点拡張を、生成Targetで実使用・欠測観測する。最後にGuard必須の監査Hookへ拡張し、任意計測との失敗意味の違いを比較する。アスペクト用の第二Runtime/Weaverや専用中央Registryは作らない。

## 18. Plugin説明を共通入口から階層的に参照する

Pluginの利用者が人間でも生成AIでも、用途や正しい使い方を個別の暗黙知に頼らず確認できることを要件とする。**Pluginは実Contractと版に対応した説明を発見でき、依存先・上位Pluginの公開説明も、許された範囲で辿れる**ようにする。説明の管理は既存Plugin/Contract・Source/文書Provenanceへつなぎ、文書だけの別Registry、Skill独自の契約正本を作らない。

初期の共通説明の内容は、長文を強制するのではなく、必要な項目を同じ順序で調べられることを重視する。

| 最初に知りたいこと | 最小の説明内容と確認先 |
| --- | --- |
| 何を提供するか | Plugin/Contractの識別子・版、できること、用途、Role、公開Operationと依存能力 |
| どう始めるか | CLI/Studio/APIからの検査入口、小さなread-only例、入力とResultの形、代表的な診断 |
| どこまで使えるか | Provider/Target/Profileの適合、必要Grant/Effect、選択済みか実組込み済みか、未対応条件 |
| 何に依存するか | 直接依存と上位の公開Facet、Version/Pin、必要な再公開許可、循環・競合時の診断 |
| 何を参照するか | 原Source/意味Role、公式・Plugin文書、詳細契約、独立Expected、実検証の根拠と現在資格 |

説明の階層を辿る際は、Aに依存するBの説明を読むことと、Aの内部Source・秘密設定・非公開APIを取得することを区別する。Bが公開可能なAの説明を参照する場合も、元Plugin/Versionと公開・再公開範囲を維持する。文書の古いCacheが残っていても、現在の開示許可に反する内容をAI Harnessへ渡さない。

Harness向けのTEI導入Skillは短くし、**対象の特定→Inspect→Plugin/契約の説明→提案→独立Expected→実行資格の確認**という道筋を示す。詳細なDSL文法、Plugin全件の一覧、複雑なCLI引数一覧をSkillへ埋め込まない。必要な詳細をCLI/GUIの現行情報から辿れるようにする。

最初に受入を検討するのは、（a）一つのPluginから概要と詳細を参照できる、（b）依存する上位Pluginの公開説明へ正しい版で辿れる、（c）非公開Facetにアクセスできない、（d）公開APIと実選択Providerが一致しない場合に旧説明で成功を偽装しない、（e）CLIとStudioとHarnessが同じSubject/Revisionを使って同じ情報境界を説明できる、の五つである。これらは[PX-55〜58](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)および#1180/#2514の既存Scenarioへ渡す候補とする。

全体P0のDesktop Stage1→Stage2と実Provider二入口に、新しいPlugin文書生成器やHarness接続を追加の必須条件として積まない。まず現行Contract/Sourceからread-onlyの共通説明を導出し、Plugin自身のTEI-nativeコード生成が増えた時点でも同じ入口で説明できるかを検証する。

**状態**：設計・受入候補。共通Plugin説明API、上位文書の実合成、Aspect Binding/Target生成、Harnessからの実操作、実Browser計測、Owner受理はNOT RUN / NOT ACCEPTED。


## 19. 公開Pluginの設計知識を共通の入口から利用する（#4679）

本節は§15の「上位Pluginの公開モデル・図解の継承」、§18の「共通説明」を、**実際に探す・読む・上位へ辿る・対象を固定してAIへ渡す・変更を検証する**という一続きの利用経路にする設計・実装計画である。[#4679](https://github.com/hat47x/tei-platform/issues/4679)を横断受入票、[#951](https://github.com/hat47x/tei-platform/issues/951)を公開Contract/Provider、[#1016](https://github.com/hat47x/tei-platform/issues/1016)を依存構成、[#2497](https://github.com/hat47x/tei-platform/issues/2497)を構造・図のProjection、[#3262](https://github.com/hat47x/tei-platform/issues/3262)を用途相対の読取り情報閉包、[#2605](https://github.com/hat47x/tei-platform/issues/2605)をTEI Product SkillのOwnerとする。文書専用の第二Registry/Resolver、第二Canonical、独立Documentation Host、AI専用意味正本は作らない。

### 19.1 共通の利用体験と三つの入口

1. **短いTEI Product Skill**：TEIが扱うモデルとPlugin、実CLI能力の発見方法、利用目的・対象の特定、必要な説明への移動方法、未確認情報と実権限の区別を教える。Skillは「操作の道案内」であり、Plugin機能一覧と構文リファレンスを複製しない。既存の `skills/yi-tei-modeling/SKILL.md` と `references/read-only-model-investigation.md` を出発点とし、固定済みHarness評価条件は変更しない。詳細なマニュアルはSkillから**必要時のみ**参照する。
2. **型付きApplication Operationのread-only投影**：`tei commands`、既存`model catalog`/`inspect`/`trace`/`validate`等の実在入口を確認し、解決済みPlugin/Contract/Lockから概要・Documentation Index・公開モデル・直接/推移依存・適用条件を参照する。将来のDoc/Dependency Queryは**操作の設計候補**であり、現行CLIの実装済みCommandと混同しない。CLI、Studio、任意MCP Adapterは同じOperation/意味の別投影とする（#1158/#3419、実Plugin inventory #4376）。
3. **Plugin所有の説明と設計資産**：人が書くMarkdownの概要、入門、操作手引き、制約、移行方法、実装上の注意と、公開Contract/型付きモデル/図Projection/テストに対応する索引をPluginの管理下に置く。オフラインSkill配布へ同梱しても元の文書/版/Provenanceを保持し、コピーを第二正本にしない。人向け手引きと機械的契約を一つの巨大な書式へ押し込まない。

### 19.2 最小の公開説明面（Documentation Surface）とデータの所有者

**公開・配布されるPlugin**の共通説明面は、Plugin ID・提供役割・短い紹介、選択済みContract/Version、対応Target/必要Capability、最短のread-only利用例、公開拡張点、直接上位依存、制限・未対応、詳細文書索引を最低限の発見項目とする。ただし全Pluginに厚いマニュアルの執筆を強制しない。内部Pluginの詳細文書は開示条件に従い非公開にできる。

- **機械的事実**：Plugin identity、Contractと版、公開Model/Relation/Hook、依存辺、Source revision、Provider/Target選択、検証・利用状態は既存意味/契約/Lock/観測から導出する。手書き本文で実CapabilityやGrantを上書きできない。
- **Plugin所有の文書**：概念説明、使用例、判断の背景、注意事項、障害対応をMarkdownなどの可読Sourceで保存し、Doc ID／文書種別／元Plugin版／Source位置を紐付ける。機能のリファレンスは契約からの導出可能部分と、手書きの補足を関連付ける。
- **図・モデル**：意味は型付きSubject/Relation/ObligationのID/向き/多重度/Scope/出典で表し、Graphviz/Mermaid/ER/表/画像は投影とする。配置変更は意味変更ではない。編集は公開拡張点に対する型付きPatch/受理へ戻し、表示から未知の意味を逆生成しない。
- **文書の資格**：文書の存在（discoverable）、取得（received）、本文の版整合（applicable）、契約との適合（checked）、実Consumerが確認した範囲（observed）、Owner受理を別に扱う。古いマニュアルを現在の実Provider保証としない。

共通の論理応答（採用前の**概念例**）は `{pluginRef, resolvedContractRef, subjectRef?, documentationRefs[], modelRefs[], parentDependencies[], scope, revision, status, unresolved[]}` 程度で足りるかをまず試す。専用の正本を増やさず、既存Resultへのread-only投影／参照を第一候補とする。Node、Relation、文書のRefは表示ラベルではなく、出典と版に結び付ける。

### 19.3 上位Pluginの階層と公開情報を辿る

利用者が下位Plugin Bを選択したときは、まずB自身の説明を返し、必要に応じてBが依存する上位Aの**固定された実契約版**、公開文書索引、公開モデル、図Projectionへ辿る。さらにAの上位へ辿る場合も同じ規則を適用する。

- **Dependency ≠ Export ≠ Re-export ≠ Edit/Execute grant**。上位に依存すること自体は、内部文書・非公開Graph・下位からの再公開・Graph Patch・実作用の許可にならない。
- 多親・ダイヤモンドで同一参照は一つにまとめ、異なる版や意味の衝突は診断する。循環は停止し、発見順・ロード順から勝手な優先順位を作らない。
- 上位由来と下位固有の説明・モデル拡張差分を並べて見せる。下位の説明から上位の原文を黙って変更しない。引き継げるのは契約で公開したFacetとExtension Pointのみ。
- 公開文書を段階取得するときは、`Plugin+Contract+Source/Doc revision+Consumer scope`に束縛し、ページ継続・end marker/欠落を検査する。Page間でSource/Grantが変わったら単一の完全な結果としない。禁止されたSourceについては本文・識別子・件数・Digest・存在自体を不正に漏らさない。
- 読取りの結果は `available / not-found / not-loaded / partial / stale / unsupported / permission-denied / unknown` 等の意味を区別する。どの状態を実際に表現できるかは既存APIの型とのGapとして記録し、今回の設計だけで全状態の製品実装を宣言しない。

### 19.4 設計要素の固定選択からHarnessへ渡す情報

TEI CUI/Studio/SHOで選択したModel/Diagram Node/RelationをHarnessが扱う場合、自由な説明文を唯一の入力とせず、**対象を固定した参照と小さな作業文脈**を渡す。

必要な最小情報は、(a) Workspace/Source revision、(b) stable Subject kind+ID（Relationならoccurrence identityも）、(c) 意図した操作・参照範囲、(d) 選択Plugin/Contract/Providerの固定版、(e) 開示用途と対象Scope、(f) 関連する公開Doc/Model/Constraint/ScenarioのRef、(g) 未解決・欠落状態、である。図の描画用edge IDや表示座標だけで対象を特定したと扱わない。実行時GrantやOwner AcceptanceはTask Packetを受け取ったHarnessが自己付与できない。

Harnessはまず短いTEI Product Skillに従い、**対象発見→必要な公開説明/依存の取得→差分候補→型付き検証→独立Expected→受理/Host実行を別経路で確認**する。Skillにプラグイン全件の文書やSourceを前置しない。CLIとStudio/SHOは同一Operation/Context契約のAdapterとして順次接続し、GUI選択が未実装ならHeadlessで対象参照の縦切りを先行する（#4503、#4515、#3262、#2516）。

### 19.5 初期実証と受入条件（#4679 D0〜D4）

| 段階 | 実証内容 | 完了条件とOwner |
| --- | --- | --- |
| D0 read-only fixture | A（公開モデル・図・短いmanual）とB（Aに依存し局所拡張）の二Plugin。Doc/Model/Contract refを固定 | 元の公開範囲・版・文書とGraphの出典を辿れる。#4679/#951/#2497 |
| D1 CLI | 現行Plugin Inventory/Resolverと共通Application OperationのRead投影 | 実WorkspaceでDoc indexと上位依存を取得し、unsupported/unresolvedを示す。#4376/#1158/#3262 |
| D2 合成・変更 | 複数親、ダイヤモンド、非公開Ref、上位版改訂、文書改訂、レイアウトだけの差分 | 競合・漏洩拒否・依存部分の再検査と無関係変更の非干渉。#951/#1016/#2497/#3267 |
| D3 Skill/Harness | Skill無しと短いSkill＋遅延資料ありを同一実CLI/Workspace/Scope/Expectedで比較 | 実Skill/Docロード、実Tool呼出し、全Reply受領、誤使用、Token/指示量、Source参照の正しさを計測。#2605/#2509/TAI #168 |
| D4 Studio/TEI化 | 同じDoc/Model Refを図から往復し、Plugin自体の `*.tei.md` 生成範囲を増やす | Studioから固定対象→説明・上位図→AI変更案→検証の往復。Go委譲だけでない生成実装とDSL簡素化を計測。#2831/#1251/#1124 |

独立Expectedへ渡す負例（PD-01〜10）：(01) 文書はあるが実契約が未解決、(02) Skillに未実装Command、(03) 文書版と依存Lockが異なる、(04) 非公開Nodeを継承、(05) 多段参照で情報漏洩、(06) ダイヤモンド・循環・同名文書競合、(07) Page途中切断、(08) layout-only変更で過剰失効、(09) Bの業務意味変更を合成PASSで自動受理、(10) 選択したPlugin Bと実Call Aが一致しないのに説明やResult一致だけで成功扱い。#1180/#2514を唯一のScenario/Expected Ownerとし、この節を新Runner/新Validation SSOTにしない。

**優先順位**：D0/D1はセルフビルドP0の阻害要件にしない並行実装。現在の最優先はStudio Stage1→Stage2実Windows WebView2 (#2831) とCLI/Studioから選択済み実Providerへの到達 (#1251)。D2〜D4の完成によってのみ「Pluginの知識が継承でき、実装が簡潔になった」と強く主張する。今回の追記は仕様候補と検収計画であり、共通Doc APIやGUI、正式Owner意味受理、実外部Harness全経路はNOT RUN / NOT ACCEPTED。


### 19.6 文書を独立した意味の正本にしないための参照契約

Documentation Surfaceは新たなPlugin/Document Registryではなく、**既存のPlugin Identity・公開Contract・解決済み構成・Source参照を辿るread-onlyの投影**とする。正式Schemaを先に固定せず、次の情報が既存型の組合せで失われずに扱えるか、D0/D1の合成Fixtureで確認する。

| 必要な参照 | 役割と同一性の条件 | 既存の責務 |
| --- | --- | --- |
| Plugin ref | Plugin IDと実選択Version、構成・Targetに束縛。表示名だけで特定しない | #951/#1016 |
| Contract / Export ref | 公開Facet（Capability、Model、Relation、Extension Point、Obligation等）と有効版・公開用途。DependencyとExportを同一視しない | #951/#3279 |
| Document ref | 所有Plugin、安定Doc ID、文書Role、対象Contractの版範囲、Source locator、原byte/Revision識別、文書内の公開Subject ref。移動パスや見出しテキストを唯一のIDにしない | #2621/#2497/#4679 |
| Graph / Diagram ref | 意味Subject・Relation occurrence identity、Graph/View変換と損失、Source、必要なら表示用Layout版を別管理 | #2497/#4198 |
| Traversal / Consumer receipt | 用途、Workspace/Lock/Scope、実取得・受領した参照、欠落状態、Source/Permissionの改訂、ページ末尾/完全性。読取りの成功を実Grant/意味受理へ昇格させない | #3262/#2516/#4503 |

上位Pluginの説明は、**契約から導出する機械的リファレンス**と**人間が執筆するマニュアル本文**を組み合わせて表示する。契約署名、依存版、選択Provider、実装義務は常に現在の契約・実Workspaceから導出し、Markdownにある古い記述を優先しない。手書き文書側には対応する契約ID・版を付け、食い違いを文書不整合として検出する。

文書間の参照には「上位Pluginの説明へ移動」「元の公開モデルへ移動」「下位Pluginが公開Facetを拡張した箇所へ移動」を明示する。**文書のナビゲーションはモデルの継承ではない**。モデルの利用・拡張・再公開は、各公開契約上の権利とExtension Pointを満たす場合にのみ成立する。下位で文書を補足しても、上位の意味定義やGrantは変更されない。

### 19.7 依存階層の探索を決定的にする

Read-onlyな「選択Pluginから上位の公開情報を調べる」操作は、少なくとも次の順序で成立させる。

1. **対象固定**：呼出元がWorkspace、用途、期待Revision、Audience/Scope、Plugin ID（または選択済みContract Ref）、参照深度・取得予算を渡す。実WorkspaceのResolver/Lockで固定されたPlugin/Provider版を採用し、リンク先に書かれたlatestやToolのロード順から版を選ばない。
2. **現在の開示を先に確認**：本人の許可された範囲で説明面・Doc索引・公開Facetを取得する。非公開項目は、内部利用者と外部Harnessへの送出を別判定にする。秘密のDoc ID、存在件数、Source/Digestを未許可Audienceへ漏らさない。
3. **直接依存を辿る**：Plugin dependency graphの辺と、Contractに宣言されたDoc/Model referenceの辺を区別する。直接の上位を調べ、必要な場合だけその上位の公開参照へ辿る。文書本文に書かれた任意URLを、自動的なPlugin依存や秘密資料の読取先へ昇格させない。
4. **多親・ダイヤモンドを合成**：同一Plugin・同一選択版・同一Contract/Doc ref・同一Scopeなら重複を束ねる。異なる版、同名別ID、循環、公開条件の競合は診断し、先着勝ち・暗黙の上書き・無限再帰を禁止する。複数版の共存は正式な構成資格がある場合だけ認める。
5. **読み込みと完全性を分ける**：必要な文書・モデルを受領した分だけ説明する。予算切れ、Source未ロード、ページ途中切断、Source/Grant/Lock変更があれば、結果をpartial/stale/unknown等の状態と根拠で返す。結果が空だから「上位の情報が存在しない」とは判断しない。
6. **結果の出所を保持**：各文書・モデル片に、提供元Plugin、公開Contract、固定された版、元Source/文書、適用用途、失われた情報、取得したSnapshotを残す。人間向けには短い概要から詳細への階層を示し、生成AI向けには同じRefを機械可読で返す。

下位Pluginの最終説明を一枚の「継承済みマニュアル」に物理的統合することは必須ではない。**下位固有の説明と、上位から有効に参照できる説明を重ねて表示する**方式を標準候補とする。変更可能な意味定義はContractによる明示的な拡張点だけに限定する。

### 19.8 文書の更新・再利用・再検証の粒度

上位Pluginの更新をすべて下位Pluginの再生成・再受理につなげない。変更の性質と、そのDoc/Model/Extension Pointを実際に参照しているConsumerを区別する。

| 変更 | 必要な再確認 | 不要と考える再作業（独立依存がなければ） |
| --- | --- | --- |
| 誤字・文体・説明文の更新 | 文書索引・リンク・配布コピー・AI向け説明の改訂確認 | 業務モデル・生成Goコードの意味受理 |
| 図の配置や表示設定だけ変更 | 図Projection/Layoutの表示・リンク確認 | Graph意味と無関係なBuild/Contract受理 |
| 公開Contractの署名・Extension Point変更 | 依存PluginのBinding、Trait履行、生成物、独立Conformance | 影響しないSubjectの一律失効 |
| 上位モデルのRelation/Invariant変更 | 対象となる下位Patch、生成・Scenario・Owner判断 | 依存外の文書とモデルの再生成 |
| 公開範囲・再公開許可・現在Grant変更 | 文書/モデルの現在の参照資格と外部送出停止・既存Cacheの再検査 | 過去の正当な受領Evidenceの書換え |
| Generator/Target/Build入力変更 | 使用されたBuild/Target/Artifactの資格とVersion Lock | 文書本文しか参照しない読取り結果の機械的失効 |

**World Fingerprint、Model Revision、Artifact Revision、文書Source Revisionは同一視しない。** 同じ文書の再フォーマットが業務意味を変えるとは限らず、同じGraph byteでも公開Contractや現在の開示条件が変われば利用資格は変わり得る。参照した上位Facetの集合・否定依存・出典を残し、依存範囲に応じた再検証を候補とする（#2600/#3262/#3267）。

版不一致の文書を「最新」として無条件に表示しない。旧版は履歴資料として表示可能でも、現在の選択済みPluginで有効な手順とは別の資格表示にする。

### 19.9 AI Skill・外部文書・操作権限の信頼境界

Product Skillの役割は、最小の概念・実CLI Discovery・対象の固定・説明と結果の読み方を教えること。**Plugin所有のマニュアルは参照資料であって、HarnessやHostへ無条件に命令する権限ではない。** 外部PluginのREADME、コード例、本文内の「このコマンドを実行せよ」「資格情報を送信せよ」等は、Toolの許可・現在のHost権限・利用者の目的に照らして検査する対象である。

- Doc本文の任意シェルコマンド、生成コード、外部リンク、画像等は、取得しただけで実行・外部送信しない。URLとローカルSourceの参照境界、Path traversal、秘密の例示値に注意する。
- Skill本文やマニュアルに記されたCommand名は、現在のCLI Discoveryおよび実バイナリで確認してから使用する。存在しない操作をAIが補完して成功したことにしない。
- 必要な文書だけ遅延取得し、返却件数・Page・end marker・Tool Reply全受領を検査する。前置された巨大Manualの読込成功を、内容が全件読めた／理解できた根拠としない。
- Harnessに渡すSubject選択は、意味ID/Relation occurrenceとSource/Lock/Scopeに結び付ける。Workbenchが描画用に生成するRelation edge IDは業務意味IDとして渡さない（#4515）。
- Skill無しTool-only、短いSkill＋必要時Manual、長いSkill本文（比較条件）の三腕を、同じ実CLI/Task/固定Source・独立Expectedで比較する。実Tool/DocsのRead、Tool request/response数、途中切断、不要送出、正答率、Token/時間・指示量を測る。既存#2509の不完全な比較Attemptを新SkillのPASSとみなさない。

「Skillを短くする」こと自体を目的にせず、**初回の見通しがよく、必要な説明へ確実に到達でき、誤操作を減らす**という実利用効果で評価する。Manual側を厚くしても構わないが、常時全投入しない。

### 19.10 DSL・Plugin Self-buildへの接続と評価

Documentation Surfaceは、Pluginの**公開Contractに結び付くAuthoring Sourceの参照**も案内する。ただし、Markdownマニュアルと意味定義を混同しない。将来の `*.tei.md` Readerには、意味Role・Source span・型付きSubject/Relation・Extension Pointを確定できることが必要（#2621）。現時点の未束縛Readerを前提に「完全TEI self-build済み」とは主張しない。

セルフビルドの独立Fixtureは、同名Labelを持つ二Nodeと、同一端点への意味Roleが異なる有向多重Relationを含める。上位Plugin Aは識別子・Relation・検査Rule・説明索引を契約公開し、下位Plugin Bがそれを参照して一つのRule/Graph Patchを追加する。少なくとも (1)上位をコピーせずモデルを参照、(2)公開図と元Sourceを相互追跡、(3)上位/下位の合成モデルから実Goコードとテストを生成、(4)CLIで生成成果物を実呼出、(5)上位のLayout-only更新では実Codeの意味上不要な再生成をしない、(6)上位の意味変更では必要な下位検証を失効、を段階受入する。

効果を次の独立指標で見る。

- **モデル再利用率**：下位が独自に複製しなかった、利用可能な上位の公開モデル・制約・型付きRelationの割合。
- **TEI由来の実生成範囲**：生成されたGo等の実装・テストのうち、TEI定義から実際に導出した単位。Goコードへの単純委譲は別計上。
- **定義体の簡素化**：上位の共通機能を利用した前後で、同一Meaning/テストCoverageを保ったまま減らせた独自の記述・例外・手作業。
- **変更局所性**：文書/Layout-onlyの変更で不要な再生成を起こさず、契約・意味変更で必要な検証を漏らさない割合。
- **操作到達性**：短いSkillから未事前学習の公開Pluginを発見し、正しい上位資料・モデル・検証操作へ到達できた割合。

生成成果物を結ぶReceiptはSource/Contract/Generator/Target/Artifact/実Consumerを分け、#2831のStudio Stage1→Stage2実再構築や#1251の実Provider二入口資格をこの小FixtureのPASSへ置き換えない。既存BuilderとPlugin/Build SSOTを利用し、Documentation専用Generator Runtimeを新設しない。

### 19.11 作業分割・反証Gate・非採用条件（#4679の次の実装票）

1. **PDC-0：公開参照の静的Fixture**（実装票 [#4701](https://github.com/hat47x/tei-platform/issues/4701)、#4679、#951、#2621）— A/BのPlugin ID、公開Doc/Graph/Extension Point/版範囲、非公開Facet、独立Expectedを固定する。既存Source Roleの外に独自Parserを作らない。最低一つの「Docは存在するがContractと版が合わない」負例を先行。
2. **PDC-1：既存CLIからの発見**（#4376、#1158、#3262）— 実 `commands` とPlugin resolver/inventoryを足場に、公開索引・上位関係・Doc refをread-onlyで取得する最小Operationを選定。現在のCLIで取得できないものはGap/unsupportedのまま残す。
3. **PDC-2：多段参照と否定試験**（#951、#1016、#2497、#2514）— A/B/複数親、ダイヤモンド、循環、同名別ID、版不一致、非公開ref、途中切断、Source/Grant変化、Layout-only変更、意味変更に独立Expectedを与える。許可されたSource以外を取らない。
4. **PDC-3：Skillの簡素化とHarness比較**（#2605、#2509、TAI #168）— 固定済みSkill/Tool比較を破壊せず、新版SkillとManualの必要時取得を別の固定腕で測る。外部HarnessからのTool/文書実Read・完全受領・適用できた契約版まで確認する。
5. **PDC-4：実Plugin・Studio・セルフビルド**（#4612、#4515、#2831、#1251）— X-TEA三視点等の実PluginをConsumerとして採用可能か検査し、同じSubject refから概要→モデル/図→Source/Rule→独立検証へ往復する。CLI-first self-build Fixtureが実生成・再生成を通ってからStudio Stage2と連携する。

**横断的停止条件**：Doc専用Registry/Resolver/Host/Authorityを必要とする設計、任意PluginのREADMEを実命令とする設計、意味IDを描画edge IDから推測する設計、権限未確認のRaw Source外部送出、文書の存在から実Provider/Target/Owner意味受理を主張する設計、正のFixtureだけで多段公開を正式採用する設計は差し戻す。負例の独立Expectedは#2514、Scenarioの正本は#1180とし、本節は作業計画のOwnerを重複させない。

**現状**：第19節の追記はContract/CLI/Model/Skill/Self-buildを横断する詳細計画である。公開Doc探索の製品Operation、Markdown Reader、Plugin継承合成、H2/H3正式比較、実Studio往復、Owner正式Schema受理はこの文書の追加だけでは成立しない。


### 19.12 指示・対象・能力の三つの確認をPlugin利用に接続する

TAIのProduct Skill必要時起動・Pinned Target・Handoff受入設計（[TAI #629](https://github.com/hat47x/tai-pleroma/issues/629)、[TAI #633](https://github.com/hat47x/tai-pleroma/issues/633)）を、TEIの**Instruction / Subject / Capabilityの三つの確認**へ対応させる。ただし新しい中央の「三閉包Engine」や第四のAuthorityは作らず、既存の各Ownerが返す根拠の対応表として扱う。

| 確認軸 | Plugin利用時に確認する最小情報 | 不足したときの扱い |
| --- | --- | --- |
| Instruction | 実際に読み込まれたTEI Product Skillの版・本文、当該タスクに必要な手順、実CLIが公開した操作、参照したManualの実取得/完全性 | 操作手順の未確認。安全なDiscoveryへ戻す。Skillに書かれた未実装Commandを推測して呼ばない |
| Subject | 利用者が指定したWorkspace/Source/Subject kind+stable ID、Relation occurrence ID、必要なModel/Rule/依存、Root/Revision/Scope、ambiguous/partial/unloaded | 対象を再特定する。図の座標や描画edge IDだけで意味対象を固定しない |
| Capability | 実選択Plugin/Contract/Provider/Target、Capabilityの実登録・必要Trait、実Provider Binding、Host Grant、EffectとEvidenceの区別 | 説明の存在やResolver PASSを実Invocation/Effectへ昇格させない。未確認なら読取り・計画へ縮退 |

この表は「すべてが揃うまで文書も読めない」という硬いGateではない。**読取り専用の調査では、確認済みの情報だけを返し、不足・切断・未許可を明示する**。変更、生成、配備、外部送出、DB/Host作用に進むときは、そのOperationと現在のAudience/Hostに実際に必要な資格を各Ownerの契約で判定する。単一のglobal boolean `ready` を作らない。

外部Harnessへ渡すContextは上記三軸の小さな参照集合と未解決欄で構成し、Plugin全件のDoc/DSLを常時投入しない。TAI横断Skillは複数Productの作業調整が実際に必要になった場合にのみ選択し、TEI単独調査ではProduct SkillまたはTool-onlyを維持する。Handoff v1、Subject Pin、Product-local Resolver、実Harness Sessionを別責務とする（#4515/#4503/#2605）。

### 19.13 マニュアルを「調べる」「試す」「検証する」に接続する

Plugin所有のManualは、詳しい説明を任意量持てる。一方、共通の索引では、長大な全文を返すのではなく**必要なTopic/Doc/Model/Exampleへ案内する**。論理的なDocument Topic（正式Schema未採用）には、安定ID、所有Plugin/固定Contract版、Role、短いタイトルと概要、Sourceと版、対象となる公開Subject/Operation/Extension Point、上位の公開参照、Audience/Disclosure、本文取得先、利用条件を結び付ける。Markdown内の見出し文字列だけを長期のTopic IDにしない。

利用者・AIが「どう使うか」を尋ねた場合、同じOperationから次の段階を選べるようにする。

1. **overview** — 何ができるか、現在のTarget/版で何が未対応かを短く説明。Plugin実選択や実行資格が未確認ならそのまま表示する。
2. **explain / reference** — その説明に結び付く公開Contract、型付きModel、Rule、元のMarkdown、上位Plugin由来のFacetを遡及可能なRefで示す。
3. **how-to / example** — 前提条件、利用対象、入力、期待する結果、失敗時の診断、必要なEffect/Grantを区別した小さい例を示す。単なるコードフェンスやシェル記述を、実行が受理されたScenarioへ昇格させない。
4. **validate / evidence** — 明示されたScenario/独立Expectedに紐付く実行・結果を確認。説明文の「テスト済み」は、実Plugin/Host/Targetの現在資格の代わりにならない。
5. **related / inherited** — 下位Plugin固有の説明と、許可された上位Facet/Doc/Graphの適用版・出典を説明。上位のRaw Sourceへ自由に辿る権限は付与しない。

**入口の具体化**：インストール済みの `tei` バイナリがある環境は、その実行物で `commands` と `version -json` など現行のDiscoveryを確認する。TEI Source checkoutを使い、必要なGo toolchainが揃うときに限って `go run ./cmd/tei` を代替入口とする。配布済みバイナリにGo開発環境やTEI Repo checkoutを要求しない。Skill文書に探索用 `go run` が書かれていても、Harnessの実バイナリや実Commandを別途固定する。これは既存#2605 Skillの**次版で測る改善候補**であり、H2/H3固定比較の途中で書換えない。

文書はデータであり、未信頼PluginのManualが命じる追加のshell起動、秘密の外部送信、OS設定変更をTool/Hostへ無条件に伝えない。操作例の`effect=read-only`はPlugin Contractや本番Sourceを外部モデルへ送信してよい資格を意味しない。Tool応答は本文取得成功、版一致、文書単位の完全受領、正しい対象に結び付いたかを分けて照合する。

### 19.14 配布物とドキュメントの利用資格を切り離す

TEI Pluginには静的Go組込み、別Process、外部Hostからの利用など複数の配布形態がある。Documentation Surfaceは**実行方式に依存しない論理契約**とし、文書の表示にPlugin本体の起動を原則要求しない。Pluginの説明は実Runtimeが不在でも「参照資料」として閲覧できるが、そのことを「利用可能」と表示しない。

- **配布単位と文書単位**：Plugin ID、Contract/Model版、Manualの対応版、Doc参照のRevision、選択TargetのArtifact/Lockを明示。静的Go単一バイナリへ全文Manual埋込みを標準条件にはしない。オフラインBundleを採る場合も固定版に基づく文書キャッシュとして扱う。
- **Releaseに関する責務**：公開Pluginの名前/Release/Artifact/第三者Conformanceは#2589。配布された文書と実使用Artifactのbyte・検証資格は別であり、Docs専用Release正本を増やさない。旧版Manualの読取りは許可され得るが、現在有効な使い方と明示的に分ける。
- **変更とキャッシュ**：Doc本文だけの更新、Contract/Graph/Generator更新、公開許可や現在のGrant変更を別に検出する。古いCacheに公開済み文書が残っていても現在の開示を優先し、古い実例で別版のPluginを操作しない。
- **移行と互換性**：Manualの手順が旧版にしか適用されないなら、その適用範囲・後継Doc・移行Pathを宣言可能にする。ただし互換性を単純なVersion番号比較で推定せず、Contract/Extension Point/Targetの実適合へ戻す。

### 19.15 追加の受入ケースと最小Gate

| ID | 独立負例または対照 | 不変条件 |
| --- | --- | --- |
| PDC-09 | Go無し・Source checkout無し、配布済み`tei`バイナリだけ利用可能 | Skillが`go run`前提で停止せず、実バイナリDiscoveryから始められる（次版Skillの改善要件） |
| PDC-10 | Doc/Skillは完全だが対象Subjectがold/missing/ambiguous | Instruction成立とSubject未資格を別表示。古いDiagram Nodeを現行Targetとして実操作しない |
| PDC-11 | Doc/Modelが正しいがProvider未選択・実呼出し未確認 | Capability不足を隠さず、説明閲覧・調査は続けても実Effect成功を主張しない |
| PDC-12 | Manualに外部送信/OSコマンドの命令文が混入 | 参照内容を実Hostへの命令とせず、許可されない作用・Raw送出0 |
| PDC-13 | Topic索引に概要はあるが本文取得はpartial / Tool Reply途中切断 | 読んだ範囲の局所説明だけを返し、manual full read/全件理解を主張しない |
| PDC-14 | A→B→AでSource/文書内容が復帰、途中でContract/Grant変更 | 単純な最終byte一致を継続した同一資格へ昇格させず、現在の参照資格を再判定 |
| PDC-15 | 旧版Docは配布済み、現行Pluginは別版 | 履歴閲覧と現行手順を区別し、版不一致の操作例を現在の実装へ自動適用しない |
| PDC-16 | Tool-only／短いProduct Skill＋遅延Manual／長文Skill全文 | 同じTask・実CLI・固定Source・Audience・独立Expectedで誤操作/正答/必要Token/完全Replyを比較。Skillの有無で対象と権限を変更しない |

PDC-09〜16を新しいScenario Engineへ登録するのではなく、PDC-0のFixture・#2514独立Expected・#1180 Scenario・#2605/#2509のHarness評価へ適切に配分する。D0/D1の最小受入は **「現在の対象Pluginを固定し、許可された公開Doc/Graphへ出典・版を保って辿り、Doc誤版・未ロード・非公開の負例を拒否できる」** こと。三軸すべての実Runtime/外部Harness効果を初期read-onlyのHard Gateにしない。

**現状**：本節は他チャットで進んだProduct/Harnessの指示・対象・能力の分離と、Pluginの外部Manual・配布・実発見の責務接続を設計したもの。正式なPlugin Doc QueryやPlugin Packager、次版Skillの実Load、現行Host/Owner資格を確定したものではない。
