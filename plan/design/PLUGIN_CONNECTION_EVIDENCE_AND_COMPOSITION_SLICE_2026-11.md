# 横断Pluginの接続を「選択・実使用・再資格化」まで確かめる最小縦断

- **位置付け**：2026-11-15の横断設計。型付き接続点と三種類の配線を定めた[横断Plugin設計§25](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)を、現行製品の読取りAPIと実装境界へ下ろす。ここで新しいCore型、永続Graph、Plugin Registry、Host、Hook Runner、公開CLIコマンドを採用しない。
- **第一目標**：一つのWorkspace・一つのRule／Operationについて、「どこから来た、何を選んだ、何が実際に使われた、何がまだ分からない」を、同じ対象に関する読み取り専用の**接続検収票**で答えられること。画面の配線図を先に作らない。
- **現状の根拠**：現行mainの `internal/operations/inspect.go`、`internal/plugin/management/inventory.go`、`internal/composition/{registry,selection,service}.go`、`internal/semantic/validationgen/positive.go`、`internal/hook/{reach,sequence,run}.go` と、既存Rule R1／参照Builder試験を静的に照合。コードの存在と製品経路の一貫した実証を区別する。
- **優先順位**：通常`tei` CLIの公開pure Entry→選択実Provider（#4495／#1251）、Rule R2-Pの正式Builder実入力（#2595／#4544）、Studio Stage1→Stage2実Windows（#2831／#4583）は独立した最上位P0。本書はそれらを止める新しい前提条件ではない。

## 1. 接続の意味を、Packageや図面の配線から決めない

Pluginが提供するのは、一定の**契約に対する貢献**である。同一PackageにはLanguage Reader、Generator、Runtime Provider、UI Presentation、Manualが同居してもよい。しかし、Packageの依存、Contractの要求、Providerの選択、Build収録、Runtime呼出し、Source/Result開示はそれぞれ別の関係となる。

| 関係 | 実際に問うこと | 主張してはいけないこと |
| --- | --- | --- |
| `requires` | この機能はどのContract／Contextを必要とするか | 依存先が選択済み・起動済み |
| `selected-by` | どのTarget／Scope／版でどのProviderを選んだか | 選択Providerを実際に呼んだ |
| `built-from`／`included-in` | Builderは何のbyteを読み、何をArtifactに入れたか | Artifact内の機能が実行された |
| `invoked` | HostのどのAttemptが実Entryへ到達したか | 業務Commit／外部通知が確定した |
| `observed-by` | 誰が何を、どの範囲・時点で再読取りしたか | 観測されなかった事象の不存在 |
| `projected-as` | 同じ参照・意味をどの表示・CLI／Studio Surfaceへ投影したか | Surfaceを開けたので編集・実行できる |

これらの関係は、標準Graphの**方向・型・個別Edge ID・多重性**を保つ型付きRelationとして投影する。単なる矢印の色・表示順、同名Plugin、同じ結果JSONで関係を自動同定しない。二つの同端点・同labelのEdgeも、Source occurrence／Relation IDが異なれば別の関係のままとする。

### 1.1 接続を問いとして扱う

最初の問いは「この選択済みRuleが、実際に選んだProvider経由でCLIから動くか」。この問いの途中には、すでに実装された局所的な正例と、まだ観測していない辺が共存する。

```text
Source occurrence --[読取]--> 型付きRule --[生成]--> Go byte
      |                                |
      +--[Inspector表示]              +--[Builder実読取?]--> Artifact?
                                  Contract --[選択?]--> Provider?
                                        Provider --[実Call?]--> CLI Consumer?
```

`?`は単に未表示なのではなく、**対応する実物・版・Attemptの根拠がまだ揃っていない**ことを示す。Rule IDが存在するだけでFunction/公開Entry IDを生成しない。Instanceに表示したConnectorがRuntimeの接続を作ったとも扱わない。

## 2. 異なるOwnerの証拠を安全につなぐ照合規則

新しい横断証拠Storeを作らず、既存Ownerが発行する観測結果を、その都度**読み取り専用**で結合する。検収票は正本ではなく問いに対する一時的な回答である。

| 照合軸 | 必要な識別と出典 | 不一致・未取得時 |
| --- | --- | --- |
| 対象 | Workspace identity/epoch、Subject kind+ID、Source occurrence／anchor、元byte／Model revision | 同名・同IDでも別Workspaceなら別対象。A→B→Aの復帰を古いAttemptの再利用にしない |
| 契約 | Contract ID/version/shape、required context、Source→Meaningの受理資格 | Parse・Source Index・Read Footprint成功を意味受理に昇格しない |
| 選択 | Target/Profile、Scope、exact Provider、依存選択とepoch、Lock | 部分一致・任意の最新Provider・旧Legacy Resolverの結果を別解決へ流用しない |
| Build | 生成器と実Builderの選択、実Source byte、依存Closure、Artifact/member digest、Build Attempt | 生成Goだけ・事後Hashだけでは正式Builderの実消費にならない |
| Invocation | Entry、Hostが作った現在Principal/Grant、実Binding、実Call Attempt、input Snapshot、Result/Failure | `Selection`はGrantでなく、結果値一致でも選択B／実Call Aは不一致 |
| Effect／Consumer | Operation identity、Commit/Receipt、独立Effect Readback、Audience/Disclosure、実Consumer受領 | ACK／exit 0／View表示をEffect確定や再送許可へ変えない |

照合の最小規則：

1. **同名を結合キーにしない**。少なくともSubjectとWorkspace、ContractとScope、AttemptとTargetの対応が必要になる。Provider・Artifactが多対一／一対多でも単一Plugin IDへ潰さない。
2. **出典が示す範囲だけ結ぶ**。ReaderのSource IndexとRuntime Callは別Ownerであり、途中のBuilder／Bindingが欠けるなら「同じ実装の一連の成功」と判定しない。
3. **現在性と完全性を別にする**。最新Snapshotでも必要項目の一部しか読めなければpartial。昔の全件Readを現在の完全性・Grantへ転用しない。
4. **反証は局所的に、無事故証明は閉包を要求**。選択B・実Call Aの正の不一致証拠は明示できるが、TraceにCallが無いだけで「Callしなかった」と判定しない。実観測の対象・時間窓・取得完了が必要である。
5. **開示と編集を分ける**。Source location・Lock・Secretに由来する情報は現在AudienceのHost Policyに従い投影する。説明文書・AI Skill・Graphから権限を継承しない。

この規則は[#951](https://github.com/hat47x/tei-platform/issues/951)の単一Resolver、[#3262](https://github.com/hat47x/tei-platform/issues/3262)のRequired/Loaded/Bound、[#2600](https://github.com/hat47x/tei-platform/issues/2600)の差分失効、[#1174](https://github.com/hat47x/tei-platform/issues/1174)のEffect、[実行因果図](PROCESS_EXECUTION_CAUSAL_PROJECTIONS_2026-11.md)の接続手順を再利用する。

## 3. 現行ソースに合わせた、実装可能な最小の入口

| 既存の実コード | 得られる事実と役割 | まだ欠ける接続 |
| --- | --- | --- |
| `operations.InspectSubject`／`SubjectInspection` | Subject、Workspace、World/Source Index fingerprint、Source位置、read-onlyの権限限界 | 現在のProvider Selection、実Builder、実Consumer |
| `plugin/management.Row`・Inventory | Legacy Catalog／ResolverからのDeclared、EnvSelected、Resolved、LockPin（未供給ならunknown）、NotModelledを投影 | 新しいService Contractの製品Selector、実Invocation／現在Grant |
| `composition.Registry.Select`／`Selection` | Contract ID/version/shape、Scope、Provider ID、Epoch・依存宣言の局所スナップショット | 製品Environment／Artifact／実Host Grantへの移行は別。単独Selectionで実Call資格を持たない |
| `composition.Service.Invoke`／`InvocationBinding` | Hostが登録した実関数のCallとgeneration Bindingを局所追跡できる | 任意の製品CLI・Studioがこの同一Entryを使用したとは限らず、実Host認可・Effectも別 |
| `validationgen.GeneratePositive` | 選択済み`semantic.Validation`から実Go比較・分岐を生成する限定能力 | 正式Product BuilderへのSource/Toolchain束縛とRuntime Provider実利用 |
| `hook.RunReachable`／`RunSequence` | 既存Hook位相、Declared Call、Reach/Failureの観測 | Guard前Call・二重Callerの可能性、Rowset完全性、実Effectからの回復 |

**採用しない短絡**：Legacy Inventoryの`Resolved=true`を`composition.Selection`のProvider選択済みと結合する、`Inspection.WorldFingerprint`を受理済みMeaning revisionと呼ぶ、`InvocationBinding`を現在Grantと呼ぶ、Ruleの生成Goテストを通常製品CLI Providerの実呼出しと呼ぶ、のいずれも認めない。対応する実変換／現在Host／実Callerに根拠がない限り、欠けている辺を`unknown`と報告する。

### 3.1 最初の開発差分（製品の変更量を抑える）

**W0：Headlessの一問一対象・read-only検収票。** `InspectSubject`の実Source情報と選択済みRuleの型付きSource／Generator候補を取得し、定義・生成・候補配線だけを表示する。Plugin Inventoryの選択とService Selectionは**別欄**とし、共通キーが未実証ならEdgeを張らない。既存`operations`／`sourceindex`の照合結果を読む薄いProjectionを候補とし、CLIの新コマンドをこの段階で必須としない。出力先は既存のCLI JSON／Studio Inspectorが妥当か、#1158／#1124のOwnerが決める。

**W1：R2-P／R3の独立した実証を受け取る。** R2-Pが正式Buildの生成Go入力byte／Artifact memberを証明し、R3が既に受理されたpure Functionを実Entry→選択Provider→通常CLIから呼んだときだけ、同じContract・Source・Artifact・Attemptで成立した**個々の接続**を追加する。R2-PとR3が別Functionの間は一本のE2E線に合成しない。Rule由来実装がFunctionに採用された後、初めて結合を検証する。

**W2：同じ問い合わせをStudioへ投影。** 実Windows Stage2で選択したSubject／Workspace epochが一致する場合のみ、画面上に実接続／未接続を表示する。CLIの出力をブラウザで開いたこと、テスト用Bridgeの応答だけでは別Consumer受領としない。StudioのPin／Dirty／A→B→A・遅着Responseを独立に拒否する。

**W3：Effectfulな接続を一例だけ検収。** Hookの事前Guard・単一Caller・Commit・別Observer・通知を四つの関係として表し、既存#1129／#1174の実作用契約で検査。Guard拒否時の実Call 0、重複Call禁止、effectUnknownでlookup-firstを実行する。必要な原子性が複数資源に跨るなら、単純なafter Hookを取引保証とせず、Effect OwnerがOutbox／補償などの実方式を選ぶ。

## 4. 合成順序と再資格化を決定論的に扱う

横断Pluginの接続は**一律のHook順**ではない。データ依存・必須Guard・型・Effect境界から部分順序を導く。順序非依存性が確認できない二変換を勝手に並列化せず、必須Guardを通らない経路は拒否する。Target/Scope不一致、Provider競合、依存の循環と失効を既存Resolverで検出する。Callの主体（宣言駆動／実装駆動／値提案）を一つに限定できないeffectful Bindingは不適格にする。

| 一因子変更 | 再確認するもの | 原則として再確認を要求しないもの |
| --- | --- | --- |
| Nodeの座標・色・折畳みのみ | View projection／Editor表示 | Meaning、Generator実装、Build Artifact、Providerの実Grant |
| Doc本文・Manualの公開範囲 | Doc Source・現在Disclosure・該当Consumer | 業務Meaning。DocがBuild入力なら該当Buildだけ再確認 |
| Ruleの参照Attribute／入力意味 | Rule Source/Meaning、必要Data Context、生成Go、依存Builder／実Provider／Consumer | 無関係なPluginの成果物 |
| Provider実装・選択／Toolchain | 関係するSelection/Lock、Artifact、実Invocation/Consumer、必要Conformance | 業務Meaningの自動変更（ただし再適格性は別） |
| 実際のInput Snapshot・取得範囲 | Required/Loaded/Bound、Result/Completeness、Effect判断 | Source記法の説明文・無関係な静的Build |
| Host Grant／Audience／Target | 現在Admission、Disclosure、実TargetでのCall／観測 | 無関係なSource/生成byte。過去の実Grantを再利用しない |

この表は[#2600](https://github.com/hat47x/tei-platform/issues/2600)の既存Invalidationを置き換えず、**失効理由を表示するための比較表**とする。Byte一致は再Build省略の根拠になる場合があっても、Meaning／Evidence／実Host Grantの再利用と同値ではない。World／Model／Artifactを一つのfingerprintに潰さない。

## 5. 独立Expectedへ渡す新しい差分（CPW-01〜10）

XCJ-01〜09と既存HG／RE／DCの負例を繰り返さず、「異なるOwnerの証拠を結ぶときだけ起きる誤判定」を対象にする。#1180／#2514で一因子比較を固定し、ここに第二ExpectedやRunnerは作らない。

| Case | 比較する一因子 | 判定条件 |
| --- | --- | --- |
| CPW-01 | 同じPlugin ID・別Workspace／Source occurrence | 同名一致でConnectionを作らない。別対象・別Sourceを維持 |
| CPW-02 | 同じProvider ID、Scope／Contract shapeだけ異なる | exact選択を満たさずConnection不適合。Contractの形だけを推測して補修しない |
| CPW-03 | SelectionはBだが実Call証拠なし／実Call Aあり | 未観測と不一致を区別。結果JSONが同じでもA≠BをFAIL |
| CPW-04 | `Inventory.Resolved`だけあり、Service Contractの正式選択なし | Legacy事実のみ出し、Product Service選択はunknown |
| CPW-05 | Source A→B→Aの旧Inspectionと現在同じbyte | 旧Attemptを現在のGrant／Source readの証拠として再利用しない |
| CPW-06 | 同一packageにBuild専用GeneratorとRuntime Provider | それぞれのTarget依存閉包を別に検査し、未選択のRuntime Provider混入を拒否 |
| CPW-07 | 同じNode/Edgeがread-only Viewと製品Runtimeで別順序 | Viewの座標や列順を実Control順へ転換しない。実際の順序は契約・Traceによる |
| CPW-08 | Source原文だけ変更しMeaning/生成Go byteは同じ | Source provenanceを再確認。Build省略、Meaning再受理、Evidence再利用は別票 |
| CPW-09 | Result/CommitのreadbackがあるがAudience変更 | 元のReceiptを秘匿なしに再投影しない。現在Disclosureを再確認 |
| CPW-10 | 実Provider未観測かつTrace部分欠落／実Call不一致を観測 | 前者はunknownで不存在を断言しない。後者は限定範囲の反証を返す |

初期試験は**CPW-01／03／04／05**のread-only差分とする。実Go Testが成立してからCPW-02／06をComposition/Build Ownerへ、CPW-07／09をStudio／Disclosure Ownerへ、CPW-08／10をSource／Evidence Ownerへ順に引き渡す。

## 6. 進捗の測り方・採用を見直す条件

次の指標を各PRの比較票に残す。

- **接続の実証率**：対象となる実Call／Build入力／Consumer受領のうち、**対象・版・Attemptを一致させて観測した辺の数**。単純なGraph Edge数を進捗としない。
- **新規機能の追加負担**：一つのread-only Operatorを追加する際のCore／CLI dispatch／Studio Bridgeの意味分岐変更数、追加Source/Generated code行数、実Artifact依存、独立Expectedの再利用率。
- **再資格化の局所性**：Layout-only、Doc-only、Rule変更、Provider変更、Grant変更を一件ずつ比較し、影響しないSource/Buildを再検証せず、影響する実Consumerを誤って再利用しない。
- **ドッグフーディングの効果**：既存Goの意味ある判断のうち、TEIの受理済み構造・Operator・生成器へ移した義務の割合、必要な手書き委譲、保守対象箇所、業務専門家の読みやすさを計測する（[実コードからモデルへの折返し](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)）。

**縮小条件**：W0の一対象で複数のResolver／真偽正本が必要になる、未証明の対応をGraphが成功として描く、既存Sourceへ戻れない編集を強制する、Effect未知を通常の失敗へ丸める、CLI-firstの実生成／実Provider／Studio Stage2を停止させる、といった場合は接続Graphをread-only表示にとどめる。

**引渡し**：W0は#1124/#3262/#1251、W1は#951/#2595/#4544/#4495、W2は#1824/#2831/#4583/#4515、W3は#1129/#1174/#4198。独立Expectedは#1180/#2514、再資格化は#2600。本書は設計と実装パケットの統合であり、CPWの新Go／実Windows／実DB試験、正式Product Service選択・Owner意味受理は**NOT RUN／NOT ACCEPTED**。

