# TEI Studio標準UIの開放的拡張性：標準データ構造／意味Plugin／汎用Projectionの適合計画（2026-11-02）

## 1. 横断判断と今回の差分

Owner方針は **Script/DSL-first、標準データ構造からTEI Studio共通UIへ投影、ノードや専用画面は任意**。[#2619 Everything is Data](https://github.com/hat47x/tei-platform/issues/2619)／[#2620 Role](https://github.com/hat47x/tei-platform/issues/2620)／[#2621 Markdown authoring](https://github.com/hat47x/tei-platform/issues/2621)／[#2622 Data Operations](https://github.com/hat47x/tei-platform/issues/2622)、[#2617 Script-first](https://github.com/hat47x/tei-platform/pull/2617)、[#2625 Syntax Sugar](https://github.com/hat47x/tei-platform/pull/2625)を設計の上流とする。

今回の不足は**一般のRecord/Sequence/Value/Refの表現と、それを解釈するPluginが追加・未対応・未受理となった場合のStudio共通画面の退避動作**を、実装差分数と編集資格で検査すること。#2617は一つのScript/Control記述とその別Domainへのdescriptor交換、#2625は構文からplain-data候補への正規化を局所検証済み。ここでは**両者を製品内で統合したと誤認しないため**に、意味未解決を扱う構造的な共通表示と、Role拡張による変化部分を切り分ける。

「無限の拡張性」は任意の対象を無条件に解釈できるという保証ではなく、**Domain/Operator/Scriptの新語彙が増えても、Studio Coreの画面種別・固有分岐を増やすことが拡張の必須条件にならない**という開放的な製品アーキテクチャ目標として検証する。

## 2. 合成すべき責務（第二正本を作らない）

| 境界 | 担当／責務 | 今回の検査対象 |
| --- | --- | --- |
| Source／表層 | #2621／#988：Markdown表・簡潔DSL・式・Graphviz・native sourceを版・発生箇所付きで保持 | Source byteと意味候補は別。例示／引用は実行宣言にならない |
| 標準Data／Role | #2620：plain Record/Sequence/Reference/Expressionの意味役割を既存Resolver経由で固定 | `role: trusted-script`という値を受理済みRole／Grantに昇格させない |
| Domain Plugin | #988／#951：Operator意味、型・参照・依存・診断、Source span | 新Roleの追加／未対応にStudio専用画面を要求しない |
| Studioの共通UI | #1124／#1824：text、汎用表・ツリー・Property・参照、Diff/Preview、原文Navigate | 表示に必要な最小構造と、意味注釈の有無を分離 |
| Context／Claim | #2506／#2516／#2582：Scope、存在・NULL・未取得、Freshness、Consumer開示 | 未取得／非開示を自動falseや不存在にしない |
| Action／Target | #2598／#1174／#1129：Intent、実Request、Authority、Effect、Receipt | UIの`visible/enabled`、read-only inspect、Source候補Patchから実Command権限を導かない |

**設計上の最小合成**：

```text
Source A/B/C (Markdown / DSL / JSON / Graphviz ...)
        │  surface-specific parse + retained occurrence
        ▼
Plain standard Data candidate  ── qualifying Role/Operator/Meaning binding
        │                                        │
        ├── Generic Studio inspect/read-only ───┤ optional semantic annotation
        │                                        │
        └── Proven invertible Source mapping ───┘ candidate patch → existing Acceptance
                                                 (not automatic execution grant)
```

Source、plain candidate、Accepted Meaning、実Load Artifact、実Effectは別々の同一性／資格を持つ。Sourceの文字列そのものがRoleであると推定する設計にはしない。

## 3. 共通UIの能力は段階的に増える

**最低限の通常表示**は、任意の公開済みData Sliceに対する `path / structural type / disclosed value`、参照・診断、SourceへのNavigateとする。既存のData/Function/Business固有画面やノード図が利用可能なら追加Projectionとして組み合わせる。

- **未対応Role**：取得が許されたplain-dataだけ表示。意味Annotationはunresolvedとして保持。未知の形を新しい実行Capability・Action・Secretとして自動解釈しない。
- **Role Plugin追加後**：同じData Slice／同じ汎用Rendererを維持したまま、適格な意味Annotationと診断・依存参照が増える。新PluginによるStudioコード変更ゼロを初期比較指標とする。
- **Source editable**：Source occurrence、単一逆写像、Revision、差分、対象Typeが確定した箇所だけ**候補Patch**を提示し、既存Preview/Acceptanceに渡す。
- **Source derived/ambiguous**：描画とNavigateは継続し、原文への安全なPatchが得られるまではread-onlyとする。繰返し出現する同一式・Macro展開・集計Viewの逆写像は別扱い。
- **実行/Disclosure**：表示・活性のScriptが指定するUIの状態と、現在Hostの権限を分離する。Dataの一部が開示不可ならConsumerに非公開情報の値・存在・件数・診断を漏らさない。汎用RendererはProviderから物理Readを行ったかを証明しない。

Studioに固有UIを追加する場合は、**専門的な操作価値・入力能力・実測の利用改善**がGeneric UIでは達しないことを先に比較する。専用UIが追加された後もSource／Meaningの正本とAcceptanceは共有する。

## 4. 新しい限定Fixtureと実証可能な範囲

[合成JS Fixture](../fixtures/studio-open-world/test_open_world_presentation.js)（Nodeから実行可能なIIFE）を追加。Domain固有の画面分岐なしに、以下を比較する。

| 試験 | 保証を検査する観点 | 実施状態 |
| --- | --- | --- |
| OW-01〜04 | 単一の共通表示にpath/type/value、boolean false／number 0／明示NULLが残る | 限定合成 PASS |
| OW-05〜07 | UI条件、シミュレーション運動、書籍章の**三つの異質構造**を同じRendererで表示。未知のRoleはplain-dataへ退避、Plugin追加で意味Annotationのみ増える | 限定合成 PASS |
| OW-08〜09 | `role: trusted-script`記述、未受理Roleは実Actionや受理済み意味にならない | 限定合成 PASS |
| OW-10 | 許可済み公開Sliceが同じ二世界の表示は、非開示値だけの違いから変化しない | 限定合成 PASS |
| OW-11 | Graphのnode/edgeもRecord/Sequenceで表現して通常表示できる。専用Graph UIは任意 | 限定合成 PASS |
| OW-12〜15 | Source-aware Patch候補は正しい単一Occurrence/Revisionでのみ作る。古い版、派生、多重出現はreadonly | 限定合成 PASS |
| OW-16 | `enabled=false`等のUI表示状態はHost Capability/Effect権限でない | 限定合成 PASS |

**実行結果**：このFixtureの全ソース文字列を、GitHub書込み前に同じV8 JSコードのまま評価し、**16 assertions PASS**。GitHubファイルへUTF-8で保存し、再読取によるblob照合を行う。Node実プロセス・GitHub Actions・本物のStudio／Parser／標準Data Canonical／Plugin Inspector／Policy／実Targetはここでは**NOT RUN**。特にOW-10は与えた仮のDisclosure済みProjectionに対する有限比較であり、実Providerの物理ReadやTiming/サイズ/ログを含む完全非干渉の保証ではない。

## 5. 既存Programでの後続受入順序

1. **E0（#2619/#2620/#2621）**：同じplain-dataのRecord/Sequenceを、2種類以上の構文／Source occurrenceから生成し、受理MeaningとSource identityを区別する。#2625の限定結果を比較元とする。
2. **E1（#988/#951）**：既存のPlugin resolution/PinからRole/Operator意味Annotation・型/依存・Source spanを供給する。未知Roleのgeneric表示・diagnosticを比較し、第二Resolverは作らない。
3. **E2（#1124/#1824）**：実`tei inspect`／標準Dataから、同じStudio Core UIで**業務UI条件と別Domain**の二つを表示。Plugin差し替え前後のStudio固有コード変更量、意味情報の範囲、Source編集往復を観測する。
4. **E3（#1180/#1163）**：Headlessで対象Scope、Meaning/Plugin Revision、Plugin未解決、ValueのNULL/missing/0、断片化したContext、Source編集の拒否を既存Scenarioへ接続する。互換Targetでの実評価とUI描画は別資格。
5. **E4（#1824/#2391/#1000）**：Studio Web/Desktop／実生成UIで、標準UIの同じ表示と元Sourceの編集・再評価を観測。Computer Use/Test向け外部制御は既存#2391を使用し、Production実行物の物理除去は別に資格確認する。

**受入指標（提案）**：新Domain Pluginを2件追加した場合、(a) Studio Coreのドメイン別UI変更ファイル数=0、(b) generic text/property/diagnosticの表示可能率、(c) Sourceへの正しい逆写像率と正しくreadonlyに留めた率、(d) 未開示情報を推測する差分0、(e) source/meaning/artifact/実行証拠の誤同一視0を測る。数値0は期待する**製品試験の合格基準候補**であり、現時点で測定済みの製品指標ではない。

**既存Critical Path**：#1119のP0-a（実Effect/Receipt）・P0-b（二Function/Owner-write）およびG0〜G4／#267 Owner意味受理の優先順は変えない。新Host、第二Acceptance Source、汎用Execution Engine、独立表示用Core固定enumをこの計画から導入しない。


## 6. 空集合の構造と否定Claimの根拠を標準UIで分ける（2026-11-02追補）

[#2619](https://github.com/hat47x/tei-platform/issues/2619)／[#2582](https://github.com/hat47x/tei-platform/issues/2582)／[#2622](https://github.com/hat47x/tei-platform/issues/2622)の最新検討では、**0件のRecord/Sequenceという構造は「現在そのScopeに存在しない」証明ではなく、負の依存関係や履歴を含まない**ことが具体化した。前節OW-01〜16はopen-world構造表示とRole注釈の退避試験であり、否定依存の資格を証明していない。ここでは構造表示はそのまま保持し、条件付きUIの意味評価**だけ**を別資格として表示する。

### 6.1 UI Scriptの一方向非対称性

例（正式なTEI DSL構文ではない）：

```text
enabled(order.submit) = not exists approval.pending
```

- `approval.pending`の**適格な1件の陽性Witness**があれば「未承認案件なし」という否定条件を**反証**し得る。全件取得を必要としないことがある。
- 一方、結果が空配列`[]`だっただけでは「未承認案件なし」を**確証**できない。Query predicate／Relation Path／対象母集団・Scope／Operator・意味版／Provider Route／Snapshot／時点／必要な行集合・非出力Field／結果の完全性が揃う必要がある。
- 「否定条件が対象Scopeで成立した」というClaimからも、ボタンの**現在のHost実行権限／受理済みBusiness Action／実Requestの束縛**を導いてはならない。UI表現の `enabled` はAction OfferとExecution Grantを兼ねない（#2598/#1174）。
- t0=0件→t1=1件→t2=0件の**ABA**では、t2の現在0件と期間中に変更が無かったことは異なる。現在の評価・監査履歴・再実行/副作用の可否を別々に保持する。
- 出力Rowが0件でも依存Footprintは残る。Row IDの存在から導出する依存だけでは、新規Rowが追加されたときの `NOT EXISTS`・LEFT JOIN未一致・Top-N・filter/sortの失効を検出できない。

**Studioの通常の構造表示**：`rows=[]`をそのまま表示してよい。しかしそれに重ねる「この条件は真である」「現在実行可能」という表示は、Consumer用途相対のQualified Claim/Action経路で判断する。必要証拠が未確認なら `undetermined`／根拠未確定の説明に留める。これはStudio独自の新しい実行Engine、状態enum、Proof Registryを提案するものではない。

### 6.2 今回の限定検証とOwnerの分担

[ `test_negative_claim_affordance.js`](../fixtures/studio-open-world/test_negative_claim_affordance.js)は**自作のread-only Consumer限定fixture**で、同じ`rows=[]`の構造投影を保ちながら、complete/rowset/predicate/order/route/operator/scope/epoch/meaning版の違いによってClaimの根拠が変わることを20 assertionで比較する。真偽評価は**fixturesに与えた仮定**に従うだけで、実Query・Provider、正式Rule/Policy、現在Host実行資格を判定しない。

| Case | UI Consumerで保持する意味差分 | 技術Owner |
| --- | --- | --- |
| NC-01〜04 | 同じ`[]`と「現在の0件を確証できる」Claimは別。たとえ対象Scopeに限る確証候補があってもUI Actionは自動許可しない | #1124/#1000（表示）、#2516（Claim） |
| NC-05〜09 | Rowset/非出力predicate/演算順/Provider経路/Operator Pinの資格が欠けると否定Claimは未確定 | #2582/#2622/#2387/#988 |
| NC-10〜12／20 | Scope/Selection epoch/Role意味版が変わると、旧0件Witnessを現在の判断へ暗黙再利用しない | #2598/#2582/#2505 |
| NC-13 | 適格な陽性Witness1件による否定の**反証**と、空集合からの肯定的不存在**確証**は非対称 | #2514/#2506/#2516 |
| NC-14 | 非開示Scopeでは陽性Witnessの有無をUI文言・件数・ボタンから漏らさない（有限比較） | #2516/#1178/#1124 |
| NC-15〜19 | ABA、集合membership／非出力依存、無関係項目、通知が得られない場合を区別 | #2582/#1124 |

**現観測**：GitHub保存Fixtureを再取得したexact JS文字列のV8限定20 assertionsを確認する。Query契約等のflagは合成入力であり、実Providerの完全性・実時点・差分通知やHistory Clockを観測していない。実Node/TEI/Go/Browser/Wails/Target/OwnerはNOT RUN。旧OW-01〜16を加えて36件の合成判別**例**があるが、同一世界／同一Scopeの縦断証拠は未結合であり、36件で製品受入したとは言わない。

**次段**：[#2619実CLI縦切り](https://github.com/hat47x/tei-platform/issues/2619)を実験入口とし、#988既存Go Language Plugin／#2506 Function Data Context／#2582のRule依存を**同じSource/Query/Scope/Meaning/Plugin Pin**で読み、#1124のStudio read-only通常表示へ渡す。既存#1180 Scenario/独立#2514 Oracleで式と実Traceを比較し、必要な実時点Read・動的変更で相違を検証する。SourceとUIの表示一致を意味受理へ昇格しない。

## 7. 既存Decision Indexも特殊画面なしに構造と資格を別々に投影する

現main `b1c019fec4f05ca65e3d7491608cb849ce523d29`の`tools/reference-shared-acceptance/decision.go`（blob `9c203ea6f43781e68e63e03704900a2363aae99f`）は`DecisionIndex{schema,modelRevision,records[]}`を返し、各Recordに**別々の**`status`（current/superseded）、`sourceEdits`、`evidenceStatus`、`reviewBindingStatus`、`decisionActorStatus`、`decisionAuthorityStatus`、`decisionWitnessStatus`、`witnessDiagnostics`等が存在する。既存[PR #2311](https://github.com/hat47x/tei-platform/pull/2311)の`decisionWitnessStatus=verified`は、Decision署名のexactな検証条件であって、Ownerの業務意味受理・Current Source適用・実Target実行資格ではない。

一方、現`reference/tei-studio-workbench/presentation.ts`（blob `518ba999383fa11fe7c5162a18b425c162e22a8e`）の`readCatalog`/`readInspection`はread-only CatalogとモデルのCanonical JSON投影であり、DecisionIndexを読み込む既成のStudio経路ではない（`main.ts` blob `11815ab583dbcf6bb28baab83d7854389c1043be`）。したがって**実Studioに署名Decisionの資格がすでに表示されているとは扱わない**。

[#1124 W0〜W5既存Node合成試験](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6557448871)で、witness statusを既存Decision Consumerが落とす、またはAuthorityが未対応値と表示される事例は**既に試験済み**である。ここではその試験を再作成せず、#2631標準Studioの受入Consumerへ次の対応を引き渡す。

- DecisionIndexの許可済みフィールドを通常の`path/type/value`および標準の診断／来歴表示へ投影し、`verified`、`authority-qualified`、`current`、`evidenceStatus`を**単一の「受理済み」booleanへ畳まない**。
- 実際の確認時点、Verify policy/sourceとrevocation/Grantが不明なとき「現在も資格あり」とは表示しない。内部診断コード・署名者・SourcePath・詳細Reasonの**開示は#2516/#958の適用条件**に従い、`WITNESS_*`件数・内容を出してよいと推定しない。
- Content-addressed Record Indexは時間順・試行回数のLedgerを示さない（[#1124 AN-08](https://github.com/hat47x/tei-platform/issues/1124#issuecomment-6578931168)）。一覧の先頭/末尾を「最新承認」、Record数を「操作回数」と表示しない。
- このケースでも、Role/InterpreterやStudioの専用Decision画面・新Policy Engineを設計必須としない。必要な資格情報を共通Property/Status/SourceRef/Diagnosticで表示できるか、既存#1124 M0〜M3／#1824 PoC CのHeadless→実UI比較へ渡す。

**次段の製品受入**：現実の`reference-shared-acceptance decisions`を固定Workspaceで実行し、`modelRevision`とDecisionIndexの実受信byte、CLIのVerify構成、Studioの実読取Source/Consumer開示を同一Runの証拠へ接合する。これは**実行計画であって実証ではない**。#267/#974は署名witness解釈、#1124はModelerのSourceとUI表示、#1824は実Web/Desktop、#2516は用途・Disclosureを引き続き所有する。

本追補は既存#1119のG0〜G4、P0-a/P0-b、#267 Owner意味受理を変更しない。新Host／第二Accepted Source／万能Role enum／専用UIの新増築を不要なものとする。
