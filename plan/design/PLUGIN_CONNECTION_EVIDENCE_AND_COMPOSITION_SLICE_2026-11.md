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


<!-- tei-plugin-w0-evidence-cut-20261115 -->
## 7. W0の最初の実装を「根拠のない接続を作らない」検収から始める（2026-11-15）

### 7.1 現行Goの責務をそのまま使い、結合できない箇所を残す

`internal/operations/inspect.go`の`InspectSubject`／`SubjectInspection`は、Workspace・正規化済みSubject・Source Indexをread-onlyに投影する。`SourceIndexFingerprint`も`WorldFingerprint`も、意味受理やProvider選択を示さない。`internal/plugin/management/inventory.go`の`management.Report`は既存Plugin Resolverから得たLegacyの宣言・選択・Lockを表示する。`internal/composition/selection.go`の`Selection`は別のService Contract・Scope・Providerの宣言Snapshotであり、`internal/composition/service.go`の`Service.Invoke`だけが、そのServiceについて実Bindingを返す。

**これらは、それぞれ正しい情報であっても、同じWorkspace／Rule／Functionを指しているとは限らない。** 初手で必要なのは新しい統合Registryでも万能Evidence Graphでもなく、既存の問いに必要な情報をOwner別に並べ、結合を許す根拠がない箇所を切断したまま示すread-only Projectionである。

| 接続する候補 | 取得元と最初に表示できる事実 | 接続を確定させるために別途必要なもの |
| --- | --- | --- |
| Subject → Source | `InspectSubject`が実読取したWorkspace identity、Subject kind/ID、Source evidence、各fingerprint | 現在の同一対象・版。意味受理／編集資格は別 |
| Rule → Function／Contract | `semantic.Validation`と明示的に受理されたFunction側の利用宣言 | どのFunction／Operationが当該Ruleを必要とするかを示す型付き利用関係。Rule IDや表示名の一致は不可 |
| Contract → Provider | `composition.Selection`のContract ID/version/shape、Scope、Provider ID、Epoch | 適切なApplication／Target／Environmentの選択に帰属すること。Legacy `Inventory.Resolved`と自動結合しない |
| Provider → Invocation | `Service.Invoke`の戻りBinding、Hostの実Caller・実Attemptにおける観測 | 同じService／選択世代・Entry・実Consumerの照合。戻り値が同じでもProvider A/Bを取り違えない |
| Generated Go → Artifact → Consumer | R1生成Go／参照Builderの限定正例と、R2-P／R3の別Owner実証 | 製品Builderの実読取・Artifact member、公開Entryと実選択Provider Call。別Function／別Attemptの成功を連結しない |

最初の出力に必要な表示状態は、(a)根拠が揃った局所事実、(b)根拠不足で`unknown`、(c)正の不一致が観測された`mismatch`、(d)対象契約に明示的に含まれない`not-applicable`である。`unknown`を`false`へ、`mismatch`を`unknown`へ、`not-applicable`を「権限不要」へ変換しない。観測が部分的なとき、未観測の呼出しやSourceの不存在を断言しない。名称は暫定的な検収表示語彙であり、新Core enumの採用ではない。

### 7.2 最初の差分とテストを4つの小単位に分ける

1. **W0-A：Sourceの根拠のみを確定する。** 一Workspace・選択済み`sales.order_line.quantity_positive`に限定し、既存`InspectSubject`と製品Rule ReaderのSource位置・型を照合する。Workbench／CLIから同じ対象へnavigateできる場合でも、実Providerや公開Entryは`unknown`のままにする。新Source Readerを作らない。
2. **W0-B：Plugin InventoryとService選択を並置する。** 既存`management.Report`のEnvironment・Lock・Scan Stateと、`composition.Selection`のContract／Scope／Epochを別区画に投影する。両者の間にOwnerが保証する対応関係が無ければ、**接続線を描かない**。余分なProviderを選び直したり、表示側で`Registry.Select`を再実行したりしない。
3. **W0-C：実Callerのいる一箇所だけ実行証拠を追加する。** R3では明示公開済みの別のpure Functionを既存の通常製品CLI→型付きServiceで一回呼び、選択Binding・実Binding・実Consumerを一Attemptで照合する。実行可能なCall Siteが未接続なら`unknown`と報告し、W0のための仮Hostや独自実行入口を追加しない。Rule R2-Pの製品Builder接続は並行する独立受入にする。
4. **W0-D：正式な利用関係が受理されてから初めて線を結ぶ。** Rule→Function／Contract、選択Provider、Build Member、実Callerが同一対象・意味・版・Target・Attemptの下で証明できた辺だけ、標準Graphの型付きRelationへ投影する。図のEdge ID・方向・多重性を保ち、Viewの座標変更や手動配線から生成・実行・Grantを発生させない。

W0-A/Bは**read-onlyの同じ画面に情報が出たこと**が目標ではない。原典／選択／呼出しの間で、実際には接続がない箇所を**誤って接続済みとしないこと**が最初の合格条件となる。最初の実装場所は`operations`の既存Readと`management`／`composition`の既存結果を受けるCLI/Inspector側の薄いProjectionを比較し、依存方向を逆転させず決める。`composition`が`operations`やStudio表示に依存する形へ移さない。

### 7.3 最初に固定する反証とOwner引渡し

| 既存Case | 最初の反証 | 合格として扱える範囲 |
| --- | --- | --- |
| CPW-01 | 異なるWorkspaceの同じSubject IDを提示 | Source位置・指紋の同値だけで同一対象へ結ばない |
| CPW-04 | `management.Row.Resolved=true`だがServiceの`Selection`は得られない | Inventoryは表示できるが、Service選択は`unknown`。新しいResolverで穴埋めしない |
| CPW-03 | Bを選択した記録に対し実Call A、または実Call証拠がない | Aの観測があれば`mismatch`、未観測なら`unknown`。同値Resultで救済しない |
| CPW-05 | SourceがA→B→A、旧読取と現byteが一致 | 旧Attemptの現在性を復活させない。必要な再読取・再資格化は元Ownerが決める |

まず既存Goのread-only負例を#1124／#3262／#1251へ、正式Builder／公開EntryのCallは#2595／#4544／#4495へ、Source版と局所失効は#2600へ引き渡す。実Hook／Effectは#1129／#1174／#4198の先行Guardと単一Caller検収後に限る。独立Expectedの置き場所は従来どおり#1180／#2514とし、ここに第二のScenario正本を作らない。

**今回確認した実装の範囲**：上記4つのGo実装の存在と責務をSourceから静的に確認した。W0-A〜Dの新しい製品Go Test、通常CLI実R3、Rule R2-P正式Builder、Windows Stage2、Effectの統合試験は本追補では**NOT RUN**。この節によって、既存PRの部分的成功を横断E2Eの合格へ昇格させない。


<!-- plugin-evidence-join-current-main-20261115 -->
## 8. Operator選択と生成コードを、実証できた接続だけで結ぶ（2026-11-15・再統合）

### 8.1 異なる成果を、一本の成功物語にしない

横断Pluginの設計を前進させる材料として、main収録済みの次の実装を使う。

| 現行の実装 | そこから言えること | まだ確認が必要なこと |
| --- | --- | --- |
| [#5007](https://github.com/hat47x/tei-platform/pull/5007) | Workspaceの`implementations`でOperatorの組込み実装を選択し、`scenario`の評価関数が選択した実装を実際に呼んだことを記録できる | Plugin Resolver／Lockからの選択、生成済みScenarioコマンドに対する選択の現在性、正式な実装適合 |
| [#5009](https://github.com/hat47x/tei-platform/pull/5009)／[#5021](https://github.com/hat47x/tei-platform/pull/5021) | `tei.rule-authoring/v0`で読んだRuleをGoへ生成し、通常の`tei self-build`のBuild Intent経路でBinaryを作り、結果を検証する限定回帰を持つ | 原Markdown・Language版・Generator版を同一の正式Build入力へ束縛したR2-P全体、Runtime Providerの実Call |
| [#5020](https://github.com/hat47x/tei-platform/pull/5020) | 選択済みRuleの元Source位置・型付き意味・生成Go ASTのRead／条件を、一試行で照合する局所回帰を持つ | 元Sourceの意味受理、正式Builderが同じ生成Attemptを消費したこと、実Consumerとの接続 |
| [#5025](https://github.com/hat47x/tei-platform/pull/5025) | Studioの実WebView2を対象とした、Workspace再読込後の旧Pin失効・再確認の回帰がmainに収録された | Plugin配線をStudioが受け取ったこと、Stage1が生成したStage2での実Consumer受領 |

#5007の「Workspaceが選んだ組込み実装」、Plugin Catalogの「選択済みProvider」、#5009の「生成・BuildされたGo」、Hostの「実呼出し」は同じ資格ではない。特に同じ`World`を保ったまま`implementations`の`use`だけをA→Bへ変えられるため、World Fingerprintだけでは生成済みScenarioコマンドの現在性を保証できない。既存の`-scenario-sha256`／`-world-sha256`／`-plugin-lock-sha256`は、その選択値を直接固定していない。この欠落は既存[#5035](https://github.com/hat47x/tei-platform/pull/5035)の計画更新と[自己記述§49](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)に沿って、次の実Go差分として扱う。

### 8.2 最初の変更は「選択だけ変えた古いコマンドの拒否」

新しい横断ResolverやPlugin用実行Hostを作る前に、既存のScenario生成・実行入口で次を確認する。

1. 同じOperator宣言・Rule・入力Fixtureに対し、manifestの実装選択A／Bだけを変える。`World`の同値と、実装の選択・実Callの相違を独立に観測する。
2. Aで生成したScenarioコマンドを、Bへ切り替えたWorkspaceで実行する。古いCommandがB選択を示すものとして通らないことを、**別Processの製品CLI実行**で確かめる。単にEvaluationの単体関数でA／Bを使い分けられることでは代えない。
3. 既存の生成Command／Build Intentの責務範囲で、Operator→実装IDの確定した選択集合を再照合する。どこに版・指紋を束縛するかはScenario／BuildのOwnerが決める。Plugin Lock、Meaning Revision、World Fingerprintへ選択値を偽装して混ぜない。
4. 選択だけの変更で必要な生成・実行資格を失効させる一方、Ruleの意味や説明文、無関係なArtifactを無条件に失効させない。既存[#2600](https://github.com/hat47x/tei-platform/issues/2600)の局所再資格化を利用し、再Build省略・Evidence再利用・現在Grantを別々に判定する。

この結果は、選択済みOperatorを実Plugin提供へ拡張する際の**最小の選択・実Call契約**になる。ただし組込み`NamedImplementation`の名前一致だけを正式Plugin Contractの適合とせず、後続では契約版・形状・Scope・実装Artifact・LockとHost Authorityを既存[#951](https://github.com/hat47x/tei-platform/issues/951)／[#1251](https://github.com/hat47x/tei-platform/issues/1251)に沿って別途照合する。

### 8.3 横断的な接続票と標準Node Viewの役割

`InspectSubject`／Source Index、Operator選択、`composition.Selection`、Builder証拠、`Service.Invoke`、Consumerの結果を**読み取り専用の問いへの回答**として結び付ける。各Ownerの証拠を新Storeへ移さず、接続に必要な識別子が欠ける場合は`unknown`、一致しない実Callがある場合は`mismatch`とする。

| 関係 | 最初に保持する識別 | 投影する事実と禁止する推測 |
| --- | --- | --- |
| `declares / uses` | Workspace、Subject kind+ID、Source occurrence、Model版 | Ruleを宣言したことと、公開Function／Entryが存在することは別 |
| `selects / binds` | Operator ID・実装ID、Contract版・形状、Scope、Selection epoch／Lock | 組込み実装選択と正式Provider選択を同名で結ばない |
| `generated-from / consumed-by` | Source／Generatorの版、生成Attempt、Go byte、Builder入力、Artifact member | Go ASTとBinaryの一致を、原Markdownの正式Build閉包へ拡張しない |
| `invokes / observed-by` | Host Attempt、実Entry・Binding、結果、必要なEffect Receipt | 値の一致をProvider一致、Commit、Consumer受領の証拠にしない |
| `projected-as` | View ID、Semantic Relation／Edge ID、対象の版、開示Audience | 画面での配線・Pin・AI案からSource変更や実行権限を生じさせない |

標準[Node View §9](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)はこの表の**表示方式**であり、第二のGraph正本でもRuntime Wiringでもない。同端点の複数Edge、Node occurrence、Data／Control／Event／Effectの関係、Portごとの型・粒度・時点・取得完全性を保持する。型が合っても必要な行の取得や現在Grantが未確認なら、そのPortは「実行可能」と描かない。配線変更は一意なSourceへ戻せる場合だけ既存Preview→Review→Acceptで提案する。

Go単一バイナリは必要なContributionだけを静的に合成し、Build専用GeneratorをRuntimeへ自動同梱しない。外部Hostや長寿命Serverへの接続は同じContractとEvidenceを再利用するが、配送・認可・世代切替の挙動はTargetごとに検収する。Wailsショーケースを含む他アプリの事例は拡張対象を検討する資料であり、TEIの正式Provider／UI実装済み証拠ではない。

### 8.4 次の独立Expectedと実装順

| 受入候補 | 一因子変更 | 期待する判定 |
| --- | --- | --- |
| CPW-11 | 実装選択A→B、Rule・World・入力は同じ。Aで生成したScenarioコマンドを再利用 | 選択の旧版を拒否するか、正しい再生成を要求。実Call Aを現在選択Bとして報告しない |
| CPW-12 | Workspaceの組込みOperatorは選択済み、Legacy Plugin Inventoryには同名行がある | 正式Pluginの`Selection`／Lockが無ければ、接続は`unknown` |
| CPW-13 | Source表現は違うがRuleと生成Go ASTは同一 | 生成実装の同値と、別Sourceの元byte／位置／Reader版を別に保持 |
| CPW-14 | Studioで旧Pinを再読込・再確認する一方、Plugin配線の実Call証拠はない | Pinの局所Consumer試験とPlugin実利用／Stage2の未検収を区別 |
| CPW-15 | 同じPackageにBuild専用Generatorと未選択Runtime Providerがある | Build依存とRuntime収録を分け、不要な実装を混入させない |
| CPW-16 | Providerの戻り値は等しいが、Contract shape・Scope・実Bindingのいずれかが違う | 実結果だけの一致を接続の成功へ昇格させない |

**最短の実装順**は、(a) CPW-11の実CLI負例と選択鮮度の修正、(b)既存InspectorへCPW-12／13のread-only接続票、(c)別々に閉じた正式R2-Pと公開pure Function R3を同一契約の下で合流、(d)通常Studioと**Stage1自身が生成したStage2**の実Consumerを独立に照合、(e)Guard→単一Caller→Commit→独立観測を伴うEffectful Hookへ拡張、である。

Ownerは#4359/#988（OperatorとDSL）、#1063/#2595/#2600（生成・失効）、#951/#1251/#4495（実Provider）、#1124/#4198（Graph／Operation）、#1824/#2831/#4583（Studio）、#1180/#2514（独立Expected）を再利用する。[#5035](https://github.com/hat47x/tei-platform/pull/5035)のProjection受入、[自己記述§49](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)のSD-O1とも目的が重なるため、**試験ケースや実装責務は二重化せず、ここでは横断接続に固有の反証を追加する**。新しい万能Hook／Resolver／Evidence Store／Graph Runtimeは採用しない。

**今回の更新は設計と実装引渡しである。** CPW-11〜16の新規Go／Windows／実Provider試験、正式R2-P全体、R3実Call、Stage1→Stage2の製品受領、Owner意味受理は、この追補によって新たに成立したものではない。


<!-- tei-plugin-join-proof-v1-20261115 -->
## 9. 型付き貢献の適合と「実際につながった」証拠を別々に確定する（2026-11-15）

### 9.1 横断Pluginを、共通Runtimeではなく貢献の検収として成立させる

標準Data／Function／Process／Node Flowにまたがって同じ機能を使えることと、すべてを共通Hostで実行することは別である。各領域のOwnerが公開する**型付き接続点**に対し、Operator／Reader／Generator／Provider／Observerなどが何を提供できるかを表し、その採用・実使用を問いごとに検査する。

検収上は、次の判断を混同しない。

| 判断する層 | 問いと必要な証拠 | これだけでは足りないもの |
| --- | --- | --- |
| **Shape** | Portの型、入出力、単位・精度、NULL／欠損・未取得、粒度、時点、Scope、取得完全性を満たすか | 同じ`takes number`という宣言は判断規則の等価性を証明しない |
| **Law** | 業務規則・演算の意味が、受理された仕様と独立Expectedに適合するか | 実装名・Plugin自己申告・一回の正常終了では受理できない |
| **Binding** | 対象Workspace／環境／Target／Contract版・形状について、**何を選んだか**。構成版／Lockは何か | 組込み`implementations`選択は、正式Plugin Resolver／Lockの選択ではない |
| **Realization** | 生成した実コード、Builder実入力、Artifact member、実Provider Entry／Callが**何だったか** | 同じ結果値やソース名だけでは、選択B→実Call Aを正当化できない |
| **Admission／Effect** | 現在のHost Grant／Audience、先行Guard、Commit・Receipt、再試行条件を満たすか | 過去のSelectionやBuild成功、UIでの配線から現在権限は導けない |
| **Consumer／Projection** | 対象・版・Attemptが一致する実CLI／Windows／外部Hostの受領証拠を持つか | 表示用Graphの完成、PoC実受領と正式Studio Stage2は同じではない |

この表は新Coreの六種Schemaや万能`ready`判定の提案ではない。各Ownerの既存Contract、Source Index、Scenario、Build Intent、Invocation、Evidenceを**必要な問いにだけ**照合する表示・検収の順序である。何の受理が欠けているかを答えられること自体が最初の価値となる。

### 9.2 mainの二つの縦断を、誤って一本へ結ばない

**Operator実行側**では、[#5007](https://github.com/hat47x/tei-platform/pull/5007)の実`Workspace.Load`→manifest選択→`validationeval.ResolveImplementations`→Scenario評価→`validationImplementations`を使う。`tei.number.non-negative`と`tei.number.non-zero`はどちらもNumberを受け取れるが、入力`0`で結果が異なる。この区別はShapeが同じでもLawが同じではないという具体例になる。現行の実Call証拠はScenario評価の**組込み実装**についてのもの。正式Plugin Providerの選択、実Binary digest、Current Grantは未接続である。

**生成・Build側**では、[#5009](https://github.com/hat47x/tei-platform/pull/5009)／[#5020](https://github.com/hat47x/tei-platform/pull/5020)のRule元位置→型付き判断→生成Go AST→通常`tei self-build`を土台にする。さらにmain統合済み[#5051](https://github.com/hat47x/tei-platform/pull/5051)は、原Markdownとbyte同一のコピーを`//go:embed`で生成Go側Packageへ含め、実Binaryが計算した原文digestを照合する限定回帰を加えた。これは原文の**複製後の実消費**を示すが、選択元Workspace path／Reader版・Generator版・元原文からコピーまでを正式Buildの同一Attemptで証明したわけではない。Build Planへ元Markdownを架空の`go-package`として追加せず、[R2-Pの生成元・消費閉包](PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)を実Go byteの受渡し点で結ぶ。

二例のRule／OperatorとBuild Subjectは現在同じ正式Function Entryではない。したがって、局所実行成功・生成Binary成功・参照Workbenchの実Windows成功を並べても、**Ruleから公開Functionへ至るPlugin E2Eは成立していない**。正式な利用宣言と公開Entryを確認した後にのみ合流させる。

### 9.3 次に成立させる「一問一対象」の接続検収票

最初の問いを「Workspaceで宣言された一Operator Ruleについて、現在の実装選択と、局所で実呼び出しされた実装は一致するか」に限定する。既存APIの観測を利用し、次の列を独立に残す。

| 記録欄 | 実在する取得元 | 接合に必要な条件 |
| --- | --- | --- |
| 定義 | Workspace／RuleのSource Index、Subject・定義範囲、Model Revision | 同じWorkspace／Source occurrence、正しいReader選択 |
| 適用 | Applicationが利用するRule／Operator ID、型、入力の必要範囲 | Rule IDを公開Entryへ自動昇格しない。必要Data Contextの完全性を確認 |
| 選択 | manifest`implementations`のOperator ID→実装ID、後続で選択指紋 | 正確な選択集合の現在性。Plugin Lockとは別欄 |
| 実Call | Scenarioの`validationImplementations`と選択Step／評価結果 | 同じOperator、Rule、実行試行。未観測を未呼出しと断定しない |
| Build／Artifact | 生成Rule Go、実Builder入力、Artifact digest | **別Rule／別Attemptのため、現時点では本Operatorに未接続** |
| 正式Plugin | Inventory、`composition.Selection`／`Service.Invoke` | 共通のContract・Scope・世代・実Entry対応が未実証なら`unknown` |
| 表示／公開 | CLI／Inspectorの対象、現在Audience、別Consumer | 表示・Source閲覧・実行Grantを同一視しない |

読み取り専用の結合は、Ownerの証拠が**同一対象・同一版・同一Scope・同一Attemptを支持する辺ごと**に行う。`unknown`（根拠不足）、`mismatch`（実証された不一致）、`not-applicable`（対象契約外）は別に扱う。閉じた観測範囲を証明できない限り、記録にないCallやRelationを「存在しない」として扱わない。[Node View §9](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)は同じ接続票のread-only Projectionとし、標準GraphのEdge identity・方向・多重性を保つ。線のドラッグでResolverの選択やSourceの変更を確定させない。

### 9.4 再資格化と横断機能の拡張順

WorkspaceのOperator実装だけをA→Bに切り替え、World／Rule／入力を据え置いた場合、**生成済みScenarioコマンドの選択資格は変わる**。この不足に対するGo修正は後継[PR #5054](https://github.com/hat47x/tei-platform/pull/5054)にあり、本書では実装を複製しない。#5054の選択指紋が成立した後も、それは実装IDの一致であって実装コードのdigest・正式Plugin Lock・Law受理ではない。A→B→Aで指紋が元に戻っても、途中の実行Attempt／Grant／Source再読取の資格を復活させない。

次のコード差分は、その後に**一つのOperator Ruleについて「定義→選択→実Call」の接続をread-onlyで照合し、未接続のBuild／Plugin／公開Entryを欠落として示す**ものとする。`cmd/tei/scenario_validation.go`の局所Callを再実行する別Runnerは作らず、既存Operation／Scenario結果のProjectionとして扱う。別々に検収したR2-Pの生成元・消費閉包、R3の正式Function→Provider実Call、通常Studioと生成PoC Bの実Windows Consumerは、対応するFunction／Artifact／Attemptが**正式に同じ場合だけ**この票へ追加する。[#5042](https://github.com/hat47x/tei-platform/pull/5042)／[#5052](https://github.com/hat47x/tei-platform/pull/5052)の生成Desktop PoC B成功もStage1自身が生成したStage2の実受領とは区別する。

将来は、同じ型付き接続の規則をAPIマッシュアップ、Node Flow、Event、VUI、データ変換へ広げられる。しかし、外部Trigger・非同期配送・更新操作・Guard／Effect／Receiptはそれぞれ追加の証拠を要する。Wailsショーケースの能力分類は検証対象を選ぶ資料であり、Web ComponentやWails Bridgeを業務Pluginの新Runtimeにしない。新しいHost／Registry／Universal Hook／Graph Engineを導入せず、既存#951／#1124／#1251／#2600／#4198へ返す。

### 9.5 独立Expectedの追加候補と最小実装単位

| Case | 一因子の差と期待する扱い |
| --- | --- |
| **CPW-17** | Operatorの入力Shapeは同じ、Numberに対するLawだけ違う（0を含む）。型適合だけで意味同等としない |
| **CPW-18** | 生成Goの判断とBinary結果は同じ、元Markdownの説明文byteだけ変更。出典の再確認と実Artifactの再利用を別に判定する |
| **CPW-19** | 実Binaryが埋込みコピーのdigestを返すが、元WorkspaceのReader／Generator版とコピーの対応がない。正式R2-P出典閉包を不成立のまま保持 |
| **CPW-20** | 選択済み組込み実装と同名のPlugin Inventory行／別ContractのProvider候補を提示。明示Bindingなしに実Plugin選択のEdgeを作らない |
| **CPW-21** | 同端点・同ラベルの二Edge、あるいは片端点だけ実観測したGraphを表示。別Identityを潰さず、不足端点から意味受理や実Callを推定しない |
| **CPW-22** | 同じ実装選択指紋に戻るA→B→A、または別Workspaceの同名Rule。古いAttempt、Grant、Source現行性を復活させない |

**実装の最初の一歩**は、#5054の旧Scenario拒否が実HEADで合格した場合の、既存`Scenario Result`→read-only検収票の局所ProjectionとCPW-17／20である。#5054が未統合なら先行するP0-Aを止めず、既存#5007の局所Callを観測する範囲で設計・負例だけ先に確定する。正式Generator入力閉包は#1121／#4544／#2595、Plugin Provider／R3は#951／#4495、Node／Studioの別Consumerは#1124／#1824／#2831／#4583、独立Expectedは#1180／#2514に引き継ぐ。

**受入の上限**：本節は接続の条件と製品実装順を更新した文書である。CPW-17〜22の新Go／Windows／DB試験、正式Plugin Runtime Provider、R2-P／R3の全経路、Ownerの意味受理は新たに実行・完了していない。全体Healthに既知の失敗が残っていても、局所成功を全体合格へ昇格しない。


<!-- tei-plugin-connection-closure-disclosure-20261115 -->
## 10. 接続の成立条件を「必要な情報の充足」と「配布先の資格」まで広げる（2026-11-15）

### 10.1 Pluginが提供する機能より、利用側が必要とするものから判定する

横断Pluginの役割は、機能一覧から適当にProviderを選んで線を引くことではない。利用側のFunction／Operation／Process Portが要求する入力と意味を先に確定し、それを満たすContributionだけを候補とする。必要条件の集合と実際に確認できた条件を、**問い合わせ対象と時点を揃えて**比べる。

| 検収する条件 | 必要な照合 | 成立していないときの扱い |
| --- | --- | --- |
| **Data Context** | Required Attribute／Relation、業務Keyと物理Key、粒度、多重度、NULL／missing／未取得、Decimal精度、validAt／knownAt、取得対象全集合 | 一部しか取得していない場合は`partial`。未観測をNULLや0件に置換しない |
| **意味と適合** | Rule／OperatorのLaw、Queryの結果意味、Native方言、Read・Transformの順序、独立Expected | 型だけ同じ、SQLが実行できた、Graphの端点が合った、というだけでは`unknown` |
| **選択と実使用** | Workspace／Environment／Target、Contract版・Scope、選択Provider、実Binding／Entry・Attempt、Artifact・Toolchain資格 | 組込みOperator選択、Plugin Inventoryの`Resolved`、`Service.Invoke`のBindingを相互に代用しない |
| **実作用と現在権限** | Read／Write権限の区別、現在Grant、Guard、更新Key・Version、Commit状態／Effect Receipt | 読み取れることから更新できると推定しない。`effectUnknown`を正常終了としない |
| **配布と受領** | Build-only／Runtime／Deliveryの別、必要な実Image・Target、Consumer／Audience、Disclosureの現在資格 | Build成功をRuntime収録や公開許可とせず、PoCの別Consumer結果を転用しない |

これらは単一の`ready`値ではなく、個別に根拠と不足を示す**問相対の接続条件**である。必要な根拠がない場合は`unknown`、実測で矛盾が見つかった場合は`mismatch`、当該Operationが要求しない条件のみ`not-applicable`とする。未取得の必須項目があるのに「Providerの能力は揃っている」と判断しない。

DB横断の設計では、既存[Data Provider適合計画](DATA_PROVIDER_CROSS_DB_CONFORMANCE_2026-11.md)のD0（PostgreSQLで実Function→Rowset→Data Context→通常CLI）を、**Data Providerが必要Contextを満たすか**の最初の製品実証として共有する。D1以降のSQLite、Document、Key/Index、Graphについては、Read結果の部分性・順序・重複・時点・能力制限をそれぞれ照合し、無断Full Scan／Local fallbackを新たな共通Providerの親切機能にしない。正式D0が閉じる前にD1〜D4の検証候補を作ることは妨げないが、P0の完成条件には加えない。

### 10.2 生成・同梱・公開を同じ資格にしない

main統合済み[#5051](https://github.com/hat47x/tei-platform/pull/5051)の`//go:embed`による原Markdownのbyte同一コピーは、生成Goの実Binaryが**特定の原文を含む**ことを検証する有用な試験である。しかし、この事実から、その原文が業務上受理されたこと、実行時に解釈されたこと、配布先に開示してよいことは導けない。[自己記述§51](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)と同じ判断をPluginのBuild／Delivery境界へ適用する。

**Build時の出典の証明**は、必要に応じてSource ID・Reader／Generator資格・byte digest・生成Attemptと実Builder入力の照合で残す。一方、**配布物への原文同梱**は独立した公開判断とする。特にWorkspaceの未公開Rule、DB定義、業務データ、Plugin Manual、添付を、再現性のためという理由だけでRuntime Binary・Web asset・外部HostのPackageに埋め込まない。Digestも情報量が小さい秘密値の開示資格を自動で与えない。

| 同じBuildからの出力 | 必要な別判定 |
| --- | --- |
| Builder側だけで使う元Source／Generator | Build入力として正しく読んだか。Runtime／Deliveryに自動収録しない |
| Runtimeに必要な生成GoとProvider Entry | 選択した実装版・実Image・Targetに実収録したか。Host Grantと実Callは別 |
| Source原byte・Manual・証拠票の公開 | Audience／Scope／現在Disclosureを満たすか。必要最小の出典参照と全文同梱を区別 |
| Studio／Web／MCP／外部Hostで表示・実行するもの | 別Consumerが何を受け取ったか、現在Grantがあるか。WailsのPinや画面表示だけで実行権限を与えない |

特に生成DesktopのN2工程は、[#5042](https://github.com/hat47x/tei-platform/pull/5042)で実Windows PoC Bの生成Bundle受領を確認した一方、`tei self-build`からのN2起動を追加した[#5059](https://github.com/hat47x/tei-platform/pull/5059)もmainへ統合され、同PRのexact HEADのWindows通常／生成Desktop両Jobが成功した。**このN2のProcess／Artifact照合が成功しても、Seedから起動した子Processである限りStage1自身のStage2再構築ではない。** この差はUIの種類ではなく、誰が選択し、何を実Buildし、どの実Image／Consumerが受け取ったかという接続の証拠で判定する。

### 10.3 最初の製品差分を変えずに、各Provider族へ展開する

最初は既存[#5007](https://github.com/hat47x/tei-platform/pull/5007)の一Operator Ruleについて**宣言→選択→局所実Call**をread-onlyで示す。旧生成Scenarioコマンドの実装選択Pinは[#5054](https://github.com/hat47x/tei-platform/pull/5054)の合格・統合を確認してから現行仕様として参照する。別Resolver／Runtime／Graph Storeを作らず、既存Scenario Evidence／Inspectorの薄いProjectionで検証する。

その接続票を、別の実Functionの実DB Read（#2586/#4496）、正式R2-PのSource→Generator→実Builder、公開pure FunctionのR3実Provider（#4495）、実Windows Stage1→Stage2（#2831/#4583）へ**それぞれ独立に**適用する。同一Contract・Source・AttemptであることをOwnerが確認できた接続だけをあとから合流する。純粋変換、DB Provider、信頼境界を越えるEffectful Providerの三例で必要な証拠を比較し、静的合成／動的ロード／外部Processの方式を最初から一つに固定しない。Go単一バイナリの標準配布は維持する。

次の独立Expected候補は既存#1180／#2514へ渡す。

| Case | 一因子を変えた反証 | 必ず残す判定 |
| --- | --- | --- |
| **CPW-23** | 必須Relationの取得が`partial`に変化。ProviderのDeclared能力は同じ | 充足せず`unknown`。未観測を0件やNULLにせず、問別Required/Loaded/Consumedを表示 |
| **CPW-24** | Source digestと実Binaryの埋込み原文一致。ただしTargetへの原文Disclosureがない | 出典の限定成功と公開不可を分離。Binary同梱から開示権限を導かない |
| **CPW-25** | RDBからDocument／Key/Index Providerへ変え、結果の見かけ上の行数は同じ | missing、ページ終端、順序、整合性、Scan条件、実Callを比較。異なるProviderを同値と決めつけない |
| **CPW-26** | 生成Windows PoCのN2をCLI経由で起動できたが、Stage1のBuild Subject／実Builder閉包がない | 一工程の実起動とStage1→Stage2セルフビルドを区別 |
| **CPW-27** | 同じRelation ID／Node端点、validAt／knownAtだけ異なる | 参照整合が成立しても時点・業務意味・現在Grantは独立判定 |
| **CPW-28** | 旧Consumerへ実装A、現ConsumerへBを配信。値は同じ、Provider epochだけ異なる | 実CallのAttempt・選択世代・実Imageを混同せず、遅着応答で現在Pinを復活させない |

**実装と計画の優先順位**：W0の一Operator Rule read-only接続票→D0 PostgreSQL一Function実Read／R2-PとR3の個別実Provider資格→Studio Stage1→Stage2実受領を**並列P0として維持**し、その後にNode View、外部Host、Effectful Hookへ接続する。CPW-23〜28は横断固有の反証候補であり、正式Meaning、Provider選択、Data操作、Disclosure／Grant、実Buildの第二正本を新設しない。

**検証境界**：この節は、mainの限定実装と進行中のPRを区別した設計・引渡しである。CPW-23〜28の新しいGo／DB／Windows試験、DB横断Provider実接続、正式R2-P／R3の全経路、Stage1→Stage2再帰セルフビルド、Owner意味・開示受理は**今回新たに検収していない**。

---

## 11. 解釈と生成を別の実現候補として、接続検収票へ加える（2026-11-15）

[Pluginの実現・最適化契約§10](PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)では、受理された型付きモデルを既存汎用関数で直接評価する方法と、同じ意味からTarget向けコードを生成する方法を、**互いに競合し得る実現候補**として整理した。これは本書§9のShape／Law／Binding／Realization／Admission・Effect／Consumerの枠組みを増やすものではなく、**Realizationの実体が異なる**例である。

最初の `quantity_positive` では、既存 `validationeval.Evaluate` と `validationgen.GeneratePositive` が生成したGoの `EvaluateSelected` を同じ型付きRule・独立Expectedで比較する。生成の入力検査とRuntime評価の前提は同一でないため、まず生成器が受け付けるShapeへ対象を絞り、正値・ゼロ・負値／nil・取得失敗／Read先と回数を判定する。ここで結果が一致しても、生成版／解釈版の**どちらが製品で選択・実呼出しされたか**を別に確認する。

| 接続検収票の欄 | 直接評価の証拠 | 生成評価の証拠 |
| --- | --- | --- |
| Meaning／Context | 選択済みRule・Model revision・必要Read／Sourceの取得範囲 | 同じRule／型に加え、Generator入力と選択Target |
| Realization | 実Evaluatorの識別、Target内での利用可能性、実Calledの記録 | 生成器版・生成Go byte・実Builder入力・Artifactと実Entry |
| Consumer／資格 | 実CallerのResult／Failure・現在のHost条件 | 実Consumerの同一Contract結果・現在Grant／Effectの別判定 |
| 改版 | Runtime側へ届いたモデルbyteと選択版の再確認 | Generator／Toolchain／Builder／Artifact・Consumerの依存失効 |

**コード生成省略は再配布省略を意味しない。** Goバイナリに型付きモデルを同梱するならそのbyteの実収録と改版を確認し、Studioから動的に読むなら現在のSource／選択実装・取得時点を確認する。無条件の生成→直接評価fallbackは、選択Providerの不一致を隠すため認めない。

最初に追加する反証は§10のRZM-01／04。独立Expectedは#1180／#2514、実現方式の比較は#1121／#4544、実Provider選択は#951／#1251、Build／差分失効は#2595／#2600に委ねる。新たな汎用Interpreter、第二Resolver／Runner、全Target同値を前提にしない。Studio Stage1→Stage2 Windowsと公開pure Function R3の優先順位は変更しない。

**状態**：main `84c335b` に収録された同一Ruleの汎用評価／生成Go比較を再利用する設計引渡しであり、重複するGo回帰は追加しない。RZM-04の正式な実現方式選択と実Callの不一致負例、実Provider・独立二Consumerは未検収。

<!-- tei-api-plugin-wails-decoupling-20261115 -->
## 12. API Pluginの機能と公開手段を分け、CLI／Serverの選択を一つの証拠へつなぐ（2026-11-15）

[Studio／CLI配布統合設計§17](STUDIO_CLI_BACKEND_PROCESS_DISTRIBUTION_UPDATE_2026-11.md)は、API PluginをWails Serverに配置しても**Operation Provider自体をWailsに依存させない**。ここではその分離を、既存のShape／Law／Binding／Realization／Admission・Effect／Consumerに適用する。新しいPlugin kindやResolverは設けない。

最初に区別するのは、(1) Operationが要求する入力・出力・失敗・Effectの意味、(2) 選択されたProviderの実装、(3) HTTP method/path・Audience・Disclosureなどの公開Contribution、(4) Wails等のTransport Host、(5) CLI／HTTP／Workbenchの実Consumerである。HTTPは実装したOperationの**公開面**であって、Wails Service名やURLがOperationの安定した意味IDになるわけではない。

標準Node Viewで可視化する場合も`requires`（契約要求）、`provides`（能力提供）、`selected`（構成による選択）、`delivered`（実Artifact収録）、`published`（Hostによる公開）、`invoked`（実呼出し）、`observed`（Consumer受領）を**用途の異なる関係として表示**する。これは表示用の関係名であり、新しい永続Core enum／配線権限ではない。型・粒度・時点・partial、匿名／認証済みActor、Grant／Disclosureが一致しない接続は、図の線を結べても適格としない。

最初の比較では既存`inspect`をCLI directとWails Server HTTPの双方へ通し、意味を同一条件で検査する。この段階の成功を「API Pluginを実行した」証拠と数えない。非Builtin Provider一件を選択できる段階になってから、その選択Lock、Target Closure、実Entry、HTTPの公開契約、独立HTTP Consumerまで順に追加する。**CLIがHTTPを呼ばずに成功すること**と、**同じProviderがServerでも実際に呼ばれること**を別の実測とする。

独立Expectedへは、CLIが隠れてWailsを起動する、HTTP公開だけで未選択Providerを呼べる、CLI主体のRead ScopeをHTTPで無断継承する、A→B→Aで古いGrantが復活する、HTTP切断で未知Effectを自動再実行する、Hostが旧GenerationのRouteを生かす、の六つを渡す。コードと検証のOwnerは既存#1158／#951／#1251／#1824／#1180／#2514へ戻し、現行P0のStudioセルフビルド／通常CLI実Providerに新しいHard dependencyを足さない。

**到達上限**：本節はPlugin接続の設計と検収候補であり、HTTP API Pluginの実製品機能、非Builtin Providerの二入口E2E、実Windows／Webの新PASSを意味しない。

### 12.1 Port全体／Field／AdapterとAPI公開の接合点

[標準Flow§11](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)の型付き配線をこのAPIにも使用する。全体Record、個別Field、明示Adapterの三方式は**Provider間で何を受け渡すか**を決める。一方、公開Contributionは**誰に、どのField・Failureを出してよいか**を決める。例えば内部CLIで読める`internalNote`を、単にRecord型が一致するという理由でHTTP応答へ含めない。Adapterが切り落としたField／partial／Lossは隠さず、既存Mapping・Disclosure・AudienceのOwnerへ渡す。

WailsのHTTP Routeが登録済みでも、(a)対象Providerが選択され、(b)そのArtifactが実Imageへ収録され、(c)現在Grant／Disclosureが成立し、(d)独立HTTP Consumerが実際の値を受け取った、の各段階は別の事実である。CLI directの結果とHTTPの結果は、同じScope・Snapshot・Contextから型付きに復号したときだけ意味を比較する。HTTP statusやWails Bindingの呼出成功で実Providerの一致を代用しない。

<!-- tei-plugin-consumer-relative-admissibility-20261117 -->
## 13. 型がつながった後に、利用目的まで満たせるかを検査する（2026-11-17横断統合）

### 13.1 「接続できる」と「判断に使える」は別の問い

[標準データ様式の取得・投影検収](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)§12と、[型付きPort／Node Viewの設計](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)§25を、実際のConsumerが要求する情報へ接続する。**Whole Port／Field Mapping／明示Adapter**はいずれも許容するが、型の適合だけではConnectionを「使用可能」にしない。Pluginが出力を返せることと、下流のRule・表示・編集・公開がその出力から正しく結論を出せることは異なる。

出力の検査には、既存Ownerが持つ次の項目を照合する。新しい横断Envelope型や第二の正本は定義しない。

| 情報 | 接続先が確認する条件 | 不明な場合の扱い |
| --- | --- | --- |
| 値・Presence | NULL／欠損／未取得／値ありを、必要な用途で識別できるか | 値だけのProjectionからPresenceを復元せず、判断は保留 |
| Identity・粒度 | 同名Field、行、業務Entity、Relation occurrence、Graph Edge IDのどれか | 表示label・行数から業務Identityや多重度を推定しない |
| 取得Scope・完全性 | Predicate、時点、対象Relation、ページング／Limit、必要範囲の実取得 | 返却された行集合を全件と申告しない |
| 意味と変換 | Contract／Shape／単位／精度／Law、Adapterの入力・出力、損失の有無 | 明示的な変換根拠なしに意味が等価としない |
| Sourceと現在性 | Source occurrence、Workspace epoch、Model／Artifact revision、実Call Attempt | 古い同値byteや別Attemptの資格を引き継がない |
| 実行・公開 | Provider選択と実Call、現在Host Grant、Audience／Disclosure、Effect／Receipt | 選択・同梱・認可・実作用を一つの成功欄にしない |

### 13.2 接続の適格性は、Consumerの要求から逆算する

「全情報を完全に保持する」ことを全Pluginの条件にすると、Field抽出、集計、匿名化、表示用整形まで不当に拒否する。代わりに、**下流の用途が必要とする識別・取得・証明を、上流の実観測から満たせるか**を評価する。

1. Consumerの用途を固定する。例：値を表示する／存在する一件を見つける／全件不存在を判断する／特定行を編集する／外部に公開する。
2. その用途が依存する属性・Relationと、識別・粒度・Presence・Coverage・時点・権限を既存Function Data Context／Rule／Admissionに照らして列挙する。
3. 各Port／Adapterの実入力・出力について、**保持した情報**と**明示的に捨てた情報**をread-onlyに比較する。説明できない欠落は「保存済み」にしない。
4. 下流の要求に照らし、usable（用途に必要な根拠が揃う）、undecided（不足・未取得）、incompatible（型／意味の矛盾）、unsupported（Target・機能が未対応）を**検収票の説明上の分類**として示す。これは新しい製品Status enumではない。実行許可・Grant・Owner受理の代替にしない。
5. 実Consumerへの配線が利用可能でも、Selected／Built-included／Invoked／ObservedとHostによる現在Admissionは別々に再照合する。公開する場合はDisclosureも独立に検証する。

意図的な情報損失には用途がある。例えば表示用の select-field(optional=true) が、欠損と明示NULLを同じ出力値に畳むなら、その値を**表示候補**に利用できる場合はあっても、元のPresenceを要するRuleの確定入力には再利用しない。逆に、元Reader／ContextでPresenceが保持されているなら、同じPort出力を捨てる必要はなく、Ruleが正しい原情報を参照できる経路を選ぶ。

### 13.3 一つのFlowでも、Consumerごとに判断の成立条件が変わる

固定例は Data Provider → Field／Record変換 → quantity_positive／存在検査 → CLI・Studioの表示 とする。**ノードはOperationの出現であり、配線図は既存GraphのProjection**。API公開は同じOperationに対する別のSurface Adapterとする。Wails上でHTTPを公開しても、Operation Providerの実装をWailsに依存させず、CLIは共通Operationへ直接到達する。HTTP Pathや画面上のNode IDを意味IDや権限IDに流用しない。

| 同じデータで変える一条件 | その用途での結論 | 誤って成立させないこと |
| --- | --- | --- |
| 実際に取得した1件に条件違反がある。残りはpartial | 「違反が存在する」はその証人の範囲で確定可能 | 「他に違反がない」「全件正常」とは言えない |
| 取得した範囲で違反は0件。残りはpartial | 全体の正常／不存在はundecided | SQLの0件／NOT EXISTSを全件証明にしない |
| Field ProjectionがNULLと未取得を同じ値へ畳む | 存在・表示用途によっては候補になり得る | 元のPresenceを必要とするRuleの確定結果にしない |
| 同じ型の行集合だがJOINで一実体が複数行に出る | 投影された行の情報は扱える | 行数＝Entity数、集計棒＝単一更新対象としない |
| 実Provider Bを選択したが実Call Aを観測 | そのConnectionは選択契約に反する | 結果JSON一致をもって成功に補正しない |
| CLIとWailsの返却値が等しい | 値の比較結果のみ確定 | 同一Grant・Disclosure・Consumer受領・実Buildを推定しない |

### 13.4 新しい独立Expected候補：CPW-11〜16

CPW-01〜10、DPC-16〜20、PC-1bの既存検証を再実装せず、**情報が失われた接続を下流の用途が誤受理するか**だけを一因子で検査する。Expectedの正本は引き続き#1180／#2514。検証結果はPRのexact HEADでのみ認定する。

| ID | 変更する一因子 | 最小の期待結果 |
| --- | --- | --- |
| CPW-11 | 同じField値で、元状態をNULLから未取得へ | Ruleの確定値を作らず、保持済み原Contextと損失ありProjectionを区別 |
| CPW-12 | 違反が見つかったpartial取得で、Consumerを存在判定から全件正常判定へ | 前者のみ成立し、後者は保留 |
| CPW-13 | 同じ2行で、業務IDを2件から同一IDの2出現へ | 行数を実体数に変換しない。Identityのある編集のみ候補 |
| CPW-14 | Whole RecordからField射影へ替え、必要Attributeを落とす | 出力Shapeが適合してもそのRule用途は使用不可／保留 |
| CPW-15 | Adapterの数値型だけ合わせ、std.decimalの精度根拠を欠く | 明示Adapterが存在しても意味・精度未保証なら不適合 |
| CPW-16 | 同一OperationのCLI実CallとWailsの公開説明だけを用意 | CLI側のCallをWails実Consumerの受領に昇格させない |

### 13.5 実装パケットと優先順位

- **PC-A（先行・read-only）**：既存 transform.Value のField投影と、Source／ContextのPresence／Coverage情報を同じ固定Fixtureで比較する。初回はCPW-11／12／14に限定し、lossyかどうかを検査票に説明する。既存 validationeval.Decide と#2495/#2497/#2506の結果を参照し、横断Runnerや別のCanonical Contextを新設しない。
- **PC-B（通常CLIの実Providerと並行）**：#2586/#4495/#1251が選択したFunction→実Query／Provider→一Consumerに、必要Coverageと実Connectionの追跡を載せる。CPW-13／15はこの経路で実証する。ProviderのRead成功と、Ruleの「全件正常」Claimを区別する。
- **PC-C（Studio／公開面）**：#2831/#4583の非Plugin Stage1→Stage2を妨げず、実Windows ConsumerとCLIの結果・Subject・epochを別々に観測する。#3422/#2390のAPI／外部Host公開ではCPW-16、現在Grant、Disclosure、Effectを別検収にする。
- **PC-D（副作用は後段）**：編集／Commitへ到達する場合だけ、既存Hook／Effect Owner（#1129/#1174）のGuard、単一Caller、Receipt、readbackを適用する。情報の不完全な接続から実Effectへ暗黙に進ませない。

**取り込み判断**：初回のPC-Aで既存Context／Rule／Mappingの照合だけで不足を説明できれば、汎用PortEnvelope・新しい型体系・独立Flow Engineは導入しない。説明できないケースが出たときのみ、一件の原Source・Target・用途・独立Expectedを添えて各Ownerへ設計差分を返す。最優先は引き続き通常CLIのactual ProviderとStudio Stage1→Stage2実Windowsセルフビルドであり、PC-A〜Dを追加のHard Gateにしない。

**証拠の状態**：本節は既存設計、現行Sourceと過去の限定回帰に基づく受入・実装計画の整理。CPW-11〜16、PC-A〜Dの新規統合試験、実Windows／独立Consumerの新しい合格、Ownerの意味受理は**NOT RUN／NOT ACCEPTED**。
