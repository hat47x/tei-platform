# Everything is Data：引用データ・意味Role適用・参照束縛・Source編集の境界（2026-11-02）

> **状態：Draft・設計追加／ED-B01〜B14はすべてNOT RUN／正式Grammar・Core schema・Owner意味受理は未決。**
> 親[#2619](https://github.com/hat47x/tei-platform/issues/2619)、標準構造・Role[#2620](https://github.com/hat47x/tei-platform/issues/2620)、Authoring[#2621](https://github.com/hat47x/tei-platform/issues/2621)、Language Plugin/Operator[#988](https://github.com/hat47x/tei-platform/issues/988)の既存構成に、「**データとして読み取れたことと、定義として意味解釈してよいことは違う**」という検証条件を接続する。第二Canonical/Resolver/Authority/Hostは設けない。

## 1. 主命題：同じプレーンな構造でも用途・出所によって解釈資格が異なる

TEIのMarkdown表、Graphviz/DOT、Goa風宣言、MukuroL風インデントDSL、関数式、メソッドチェーン、JSONは、既存標準Record/Sequence/Relation/Expression/Ref等へdesugarする**表層構文**として扱う。Role/Operator/Rule/Test/Scriptもデータで表せる。ただし同じ構造値が「業務データ」「文書中の引用例」「テスト入力」「開発者が受理した実行定義」のいずれであるかは、値の見た目から決めない。

### 観測同値の最小対（全て架空）

同じ構造化値：

```json
{"op":"delete","target":"order-42"}
```

- **世界A：資料本文に掲載されたサンプル**。引用元のCSV/Markdown/DOTにこの文字列があり、ドキュメントを検査・要約・比較する対象である。構造化できるが、TEIの操作定義・実行要求ではない。
- **世界B：信頼できるAuthoring Source上で明示的にRole付与されたAction定義の一部分**。意味Role/対象・Operator/版/入出力/前提を解決する候補になり得る。それでも**定義の解釈・意味受理・現在の実行Grant・実Invocation/Effect**は異なる資格。
- **世界C：非信頼資料内に `role: trusted-script` と書かれている**。そのRole文字列も入力データであり、作者が勝手に実行資格を得たことにはならない。正当なRole適用先とOwner受理の参照へ照合できなければ提示はcandidate/unknown。
- **世界D：正式な管理者Script**。Host/組織が正式に許可したraw DB/OS操作を行える設計余地は維持する。「Dataとして見えるから危険」ではなく、**実際の作用は対象Host/ProviderのGrantと受理済みContractで成立**する。

この設計での「引用／定義」は**SourceとConsumerに相対的な解釈資格**であって、万能の `quote` Runtime状態や、新しい必須Role enumを増やす提案ではない。Markdownの表、コードフェンス、DOTの有向辺を見つけただけではAction/Rule/Effectを実行しない。Studio/AIは「この入力をFunctionとして解釈する提案」を出せるが、正式Meaningへ採用するには既存Project Source/Role binding/Owner判断へ戻る。

## 2. 解釈の段階と誰が所有するか

概念上（**新公開APIではない**）：

```text
Authoritative source S (bytes/path/revision, declared scope)
  ↓ syntax plugin L@v: parse + source span / trivia / diagnostics
plain standard data D (literal / record / sequence / typed ref candidates)
  ↓ explicit or project-authorized Role bindings R@v + environment E
resolved semantic interpretations M (Operator/ID/type/grain/order/effect requirements)
  ↓ existing meaning acceptance / semantic lock / target lowering
derived View, Scenario, SHO document, generated code, read-only Intent candidate
  ↓ existing Host/Provider authority + actual Request/Effect observation (separate)
```

- **#988**：Language pluginの実体／版、構文の競合、Operator/評価順/Effectの意味。
- **#2620**：plain dataとRoleの対応、複数Roleの適合／競合、最低限のbootstrap、自己記述がどこまで成立するか。
- **#2621**：Markdownファイル、表・引用・Fixtureの役割指定、Source span、近接テスト、編集後の原典追跡。
- **#1120/#2505/#2497/#1124**：Stable semantic ID／版間対応候補／部分的逆写像／Studio編集結果から原Sourceへの合法な反映。
- **#2502/#2499/#1174**：正式Script/Macroの採用・配布・現在Grant・実Effect。外部Prompt/未信頼資料の「実行して」指示をRole受理やHost認可へ昇格しない（#2515／SEI #958の既存Consumer区別）。
- **#1131/#1180/#2514**：ScenarioのExpectedを生成側から独立に定義・検査する。MarkdownのExpectedセルを同じ実行候補に評価させてOracleを汚染しない。

## 3. 「すべてデータ」を実用にするための正規化・束縛の約束候補

### 3.1 意味正規形と編集原文の両立

*Meaning normalization* の同値判断では、Role/Operatorとその実際の版、解決済みRef identity、lexical scope、依存・評価順、Null/Missing/Precision/Unit/Bag/Clock/Effect等の**当該Claimに必要な区別**を維持する。表・チェーン・簡易DSLが同じShapeになっても、依存条件が未確定なら「意味同値」と主張しない。

表層の**Source fidelity**は別の合格条件：コメント、空行、Markdown周辺文章、見出し順、DOT layout/色、source span、引用元を保持できるか。Presentation-only変更で生成コードやBusiness意味が必ず変わるとは限らない。逆に意味データが同値だから、ユーザーのコメント・編集位置・文書レビューを捨ててよいわけではない。

「Same Normal Form」→「同じSourceを安全に編集できる」は成り立たない。特に、一つのSemantic nodeを2つのSource occurrenceから導出する場合、Graphなどの派生Viewに戻り先が一意とは限らない。**read-only Navigate**、またはSource/Occurrence/Role/Revisionが一意で合法な限定PATCHで扱い、無条件の全Source再生成をしない。

### 3.2 Lexical scope・Alpha-renaming・Reference hygiene

文字列の名前と参照対象を分離する。たとえば外側の変数 `x` と、関数 `f(x)` の引数 `x` が同名でも、それぞれ別の束縛である。単なるRecord flattenや名前の整列で同一Refにしてはならない。ローカル変数名の健全な改名（alpha-renaming）が意味を変えない条件と、逆にScopeを誤解した改名が別参照を壊す反例を比較する。

同じ `delete`／`where` の文字列でも、Operator Plugin ID・version・import scope・Role・SourceのBindingが異なる場合は、名前だけで同一の実行意味としない。マクロ展開後のノードは、元宣言と展開Instance／出所を分け、受理済みMacro Sourceの編集位置を推測して自動書戻ししない。

### 3.3 決定性と合法な差

固定した**実Source bytes＋Syntax/Role/Operator/Plugin依存Lock＋意味環境**で、同じ解釈候補・診断・必要Ref集合を再現できるか検証する。一方、文書装飾変更やDOT座標移動後に、同じ意味を導ける場合は、Source fingerprintの変化とMeaningの変化を別に示す。

構文糖を使ったチェーン `where→sum` の並び替えを許すかは、Operator law、lazy/eager、短絡、失敗、Side effect、NULL、Bag粒度に依存する。Shape同一だけで可換・結合・分配法則を推定しない。通常のRDB pushdownとは別のNative SQL assetの正本性も維持（#2622）。

## 4. 最小の独立正負ケース（ED-B01〜14／全件NOT RUN）

| Case | 圧力 | 期待する境界／不適合 |
| --- | --- | --- |
| **B01 Literal Action** | 外部CSVに `op=delete`。これをStudioで表として読む | Dataとしてinspect可能。Action Scriptの意味採用や実行は発生しない |
| **B02 Self-asserted Role** | 未信頼Markdownが `role: trusted-script` と宣言 | Role文字列は観測可能だが、未承認SourceからのRole適用はaccepted/authorizedにならない |
| **B03 Explicit Definition** | Projectが指定した正式Source/RoleのFunction宣言 | 同じplain dataからRole/Operator参照を解決可能。ただし実Host Grantを与えない |
| **B04 Multi-role nonconflict** | 一つのDatasetをFunctionの例とTest Inputに参照 | Role別の用途・Source Ref/時点/独立Expectedを保持。無用にデータ複製しない |
| **B05 Role conflict** | 同じ定義に互いに矛盾するOperator/Role宣言 | Loader順・書式順で片方を黙って優先せず、競合理由を診断 |
| **B06 Literal vs Reference** | 文字列 `"x"` と式Ref `x` を同じ表で記載 | plain Record/Seqに解決後もliteralとrefを区別。存在しないRefは未解決 |
| **B07 Lexical Shadowing** | Outer `x` と `f(x)`の引数、nested lambda | lexical identity維持。単純な名前一致で別Scopeへ束縛しない |
| **B08 Alpha-renaming** | 引数 `x`→`a` の合法改名、他Scopeの同名参照 | 意味同値を保証できる範囲と、参照捕獲で非同値になる範囲を分ける |
| **B09 Same Data Multiple Origins** | 複数Markdown表の出現が同一Function Roleを参照 | Meaning ID一致だけでは更新すべきSource spanを一意にしない。編集不可理由または明示occurrenceを提示 |
| **B10 Source-only Change** | Markdown装飾/コメント、DOT layoutだけ変更 | Source byte/offset更新とMeaningへの影響を別判定。コメント・配置情報を暗黙削除しない |
| **B11 Chain Evaluation** | `where→sum` と `sum→where`、短絡式・effectful call | 同じノード集合でも順序/対象粒度・失敗が違えば非同値／unsupported |
| **B12 Grammar/Role Upgrade** | Syntax Plugin版、Operator版、Role Contract版のいずれか変更 | 旧Sourceが同じでも解釈可能性を現行環境で再検証。旧Accepted意味を黙って新定義へ継承しない |
| **B13 Paper/Documentation** | 学術論文のコード例・数式・引用・階層リンク | 参考Sourceを構造として表現可能。引用しただけで実装可能なFunctionやBusiness Acceptedへ昇格させない |
| **B14 AI Repair** | AIがB05/B07/B09のエラーを1箇所直そうとする | 不要な原Source・独立Expected・秘密の領域を変更せず、ProposalとDiffを返す。正式Meaningの採否・実Effectは別 |

### 判定の方法と新しい正本を作らないこと

- まずB01/B02/B03（**Data/Role/Authorityを識別する3世界**）、B06/B07（名前解決）、B09/B10（Source Fidelity）を自作Fixtureとして独立の期待値を固定する。Markdownの説明表と正式Function表を、字面だけで識別する実装は不適合。
- 次にB04/B05/B08/B11/B12（Role合成／演算法則／版）を、既存#988のReferenceと#2620の語彙に照合する。新たな解釈器を正式ABIとして先行導入するより、**既存Standard Data／Semantic Extension Lock／既存Source Resolverで表現できる範囲**を測定する。
- 最後にB13/B14で、同じ文書をSHO／CUI／TEI Studioへ見せたとき、引用・未確定Role・Source位置・読み取り可否を失わず説明できるか検証。非公開原本を外部AIへ移すことを要件にしない。
- 成果指標：Roleの誤昇格数、同名Ref誤束縛数、誤書戻し数、編集対象の無関係byte差分、Core専用Editor追加数、AIのfirst-pass-valid/修正往復、未受理をPASS扱いする例の数。局所Fixtureの完全一致とOwnerの意味受理・現行権限・実Provider Effectは別の証拠。

## 5. 計画・横断受渡し

**#2619**：Everything is Dataの横断統合・受入表。B01/B02/B03を基本参照例とし、Roleの正式採用やArtifact/Host保証を親Issueに重複実装しない。

**#2620**：標準データ・Role解釈・bootstrap所有。B04〜B08/B12の適合／競合、同じplain dataからの複数合法解釈と誤解釈防止を担当。

**#2621**：Markdown/Source occurrence/quoted example、ReferenceとActual編集。B01/B09/B10/B13のAuthoring/文書再解決を担当し、SHO #4/#6、MMI C-01/T2への現行適用性を分ける。

**#988**：Syntax PluginのGrammar/version、lexical Scope、Operator評価順、macro expansionとSource span。B06/B07/B08/B11/B12を既存Operator/Language Conformanceへ渡す。

**#2502**：Trusted Developer/Admin ScriptとDelegated Macroの製品ライフサイクルで、B01/02の「資料中の命令はScript起動ではない」をConsumer側で比較。実際のMacro権限は#2499/#1174と現行HostのOwnerで実証。

**#2516/#2515**：引用やデータを外部AIへ渡すとき、未信頼資料の指示をControl authorityへ昇格させない。また、ソースへのアクセス許可と外部送信可否を分離。実OS/Network境界はそれらの既存Issueで実測。

**#2622/#2507**：Data Operation/Actionやサービス適用は、同じ `op` 文字列だけで成立しない。Source Role・実Field/Scope/Context/Action purpose・Provider能力を照合。実Effectは#1174。

### 状態

この文書・上記ケースは採用前の**設計／評価計画のみ**。正式Role runtime、Grammar、Canonical schema、CI、Go/Java/DB/GUI、外部Harness/OS、独立Owner意味受理は今回NOT RUN。既存P0実DB・Host縦切り、#1119 S0/G0〜G4、MMI G0-0、SHO S0を延期させない。専門UI/Parser/Registry/Trust Engine/Hostの第二実装を先行提案しない。


## 6. 2026-11-02 横断追加：意味正規化と再利用・実行資格の影響次元を分ける

[#2619横断接合計画／Draft #2629](https://github.com/hat47x/tei-platform/pull/2629)、[#2600増分ビルド](https://github.com/hat47x/tei-platform/issues/2600)、[#2624 Function Mapping/Rule Footprint](https://github.com/hat47x/tei-platform/pull/2624)、[#2627 Provider Route](https://github.com/hat47x/tei-platform/pull/2627)、[#2604 Artifact公開](https://github.com/hat47x/tei-platform/issues/2604)に接続する新しい判別条件は、**Sourceと意味が同じでも、Read、現在の実行権限、文書、公開世代の資格は同一とは限らない**ことである。

### 6.1 段階別の資格と再利用の限界

| 比較軸 | 変更例と少なくとも見直す資格 |
| --- | --- |
| Source/Occurrence | 原文byte、引用位置、ファイル所在、参照解決、SHO編集元 |
| Syntax/Role/Meaning | Language/Operator Plugin版、ProjectによるRole Binding、Lexical Scope、Accepted Meaning |
| Function/Rule inputs | Function Data Context Mapping、取得範囲、**非表示でも受理済みRuleが要求する**別入力、selection epoch |
| Provider/physical route | Primary/Archive/OData、実Read、現在のBinding、Provider世代、Result/Effectの対象 |
| Authority/Effect | Grant/Principal/Scopeの現在性、対象への作用、独立Receipt、業務完了 |
| Document/Delivery | 編集上の説明、SHO Review、ManifestからPathと実読取byte、公開Generation、開示先 |

一つのFingerprintで全段階の一致を主張する代わりに、既存Source/Meaning/Dependency/Artifact/Resultの証拠を**用途相対に照合**する。Roleだけを変更した場合は原文byteが不変でも解釈は変わり得る。一方、Host Grant失効では意味やCodeを再生成することより、実行前の現在資格の再確認が重要になる。表示額600が同じでも、別の与信Ruleが参照する上限500/700は判断を変える（#2624 RF-01〜18）。古い成功Receiptがあっても新しい実行許可は自明でない。

### 6.2 一軸変更の反証票 EI-I01〜I13（実TEIの新試験はNOT RUN）

| ID | 変更する軸 | 影響を分ける狙い |
| --- | --- | --- |
| I01 | Source byteのみ | 再parse/Source出典とMeaning同値の証明を区別 |
| I02 | Syntax Plugin版のみ | Source同一でも現行Parser結果は再確認 |
| I03 | Project Role Bindingのみ | 同じ文面が引用例からFunctionへ変わり得る |
| I04 | Operator ID/versionのみ | 同名式の型/順序/NULL/Effect意味を再確認 |
| I05 | Function Mappingの全件→部分のみ | Read/Completeness/Claim/Action資格を再確認 |
| I06 | 非表示Rule Dependencyのみ | 同じ表示値から別のValidation結果が出る |
| I07 | Primary→Archive Providerのみ | Meaning同一でも物理Read/Write対象が異なる |
| I08 | Context selection epochのみ | 旧Intentの対象・Anchor・時点を再照合 |
| I09 | Host Grant許可→失効のみ | Code/Meaning同一でも実行可否は変わる |
| I10 | SHO編集上の説明のみ | Code同一候補でも文書/Review資格は変わる |
| I11 | Publication Generationのみ | Manifest→Path→実読取byteの一致を再検査 |
| I12 | Consumer開示Profileのみ | 実Truth同じでも新たな外部Delivery資格は別 |
| I13 | Source所在のみ（本文同一） | Source locator再解決。相対Import等が変わる場合Meaningも再検査 |

これらは**何を再確認するか**の条件であり、必ず全段階を再コンパイルするという規約ではない。旧新Dependency Graphと変更前後のSelection述語、実消費Sourceの閉包が完全な場合に限り、Claim相対のSelective Reuseを検証する（#1103/#2600）。依存不明なら必要範囲で保守的な全再検査かunknownを示す。元原文の内容変更でMeaning同値を主張するには、独立の正規化・解釈照合が必要。

### 6.3 隔離Reference（実TEI Cache/Provider試験とは別）

[Python stdlibの独立Fixture](../fixtures/everything-data-impact/check_phase_impact.py) を追加した。Source、Parsed、Meaning、Read、Runtime、Document、Publicationの**試験用Signature**に対し、13次元の単独変更、およびSourceやMeaningが同じでも下流資格が変わる反例を記述。隔離環境で **unittest 3 methods PASS、py_compile PASS**（内部subTestは13+7+5ケース、実製品を実行していない）。

Fixtureは意図的に**保守的な影響判定**を行う。Source byteが変われば意味検査を再実施する前提とし、Semantic Equalityが独立に証明できた後の高速skipまでは実装しない。Source所在変更でParsed/Meaningが不変という合成ケースは、Import/相対Refが意味へ影響しないと限定された環境だけの例である。Signatureそのものを製品Cache Key・ABI/Manifestへ採用する提案ではない。

### 6.4 既存計画への受渡し

- **#2620/#988／H0〜H1**：I02/I03/I04のRole/Operator/Plugin版の独立変更を取り込み、SourceだけをCache Keyとする誤成功を検証。
- **#2621／SHO**：I01/I10/I13で原典・Meaning・文書Reviewの現在性を独立に照合。
- **#2624/#2622/#2627／H2**：I05/I06/I07を組み合わせず単独比較し、暗黙なRule Readと誤ったProvider Targetを区別。
- **#2598/#1174/#2604／H4**：I08/I09/I11/I12を、実Request・Grant・現行公開・実Consumer読取の個別資格として扱う。
- **#2600 増分ビルド**：I01〜I13を既存IB-01〜14の追加Pressureへ引き渡し、**誤再利用、過剰無効化、旧版誤公開、不要なCurrent Authority再利用**を別集計。高速化の実測、SHO実描画、実DB/Host/Owner受理はNOT RUN。

既存#1119 P0、正式Owner意味受理、現行Host/Provider Authorityの順序を維持する。新Cache Engine、共通Host/Registry、別の意味正本、追加の公式Gateは設けない。


## 7. 2026-11-02追加：操作経路の一意性とConsumerの候補可視性を分離する

既存[#2611 複数Persistence Mapping](https://github.com/hat47x/tei-platform/issues/2611)の現行[Draft #2587](https://github.com/hat47x/tei-platform/pull/2587)静的所見、および[実Operation向けcharacterization Draft #2632](https://github.com/hat47x/tei-platform/pull/2632)、[#2627](https://github.com/hat47x/tei-platform/pull/2627) Provider候補、[#2516](https://github.com/hat47x/tei-platform/issues/2516) Consumer相対開示、[#2507](https://github.com/hat47x/tei-platform/issues/2507) Action discoveryを横断する。新しい検証条件は、**利用者が見えるOperation経路が一つであることは、実際に利用可能な意味上の経路が一意である証拠ではない**こと。

### 7.1 競合する「一意性」の意味

同じ `sales.order`／`order_entry.update.sales.order` に対し、`archive-db` と `primary-db` の二つの異なるMappingがある自作世界を考える。Archive RouteをConsumerが知る資格はないが、Primary Routeだけが画面／Agentに提示されたとする。

- **Semantic candidate universe**：正式Data Model/Function Context/目的・Action・Route Mappingが対象とする候補集合。Consumer向け表示から逆算して作らない。
- **Route equivalence**：Alias名ではなく、実Endpoint/Mapping版/Scope/Action purpose/認可契約と必要なRead/Write意味が一致することを確認した範囲だけを同一クラス候補にできる。二つのAliasが同じDB名を指すというだけで完全な実行資格を主張しない。Aliasに応じたGrant等の差も保持する。
- **Selection**：Accepted Action/明示Binding/Ownerの正当なRoute Policyが経路を指定するか、同等性を確認した候補群が一意の場合だけ単一候補になり得る。**「閲覧可能なものが一件」「今Grantがあるものが一件」「辞書順先頭」から業務上の経路を推定しない**。
- **Current execution**：選んだ経路について現Host/Providerが現Grant・expected-state・実Targetを再検査する。選択済みのPrimaryがdenied/staleでも、別Archiveへ暗黙fallbackしない。合法なPurposeごとのReplica Read/Primary Writeを明示的に組み合わせる構成は可能だが、共有Snapshot・commit資格は別。
- **Consumer disclosure**：利用者／外部AIには、許される対象名、候補数、Rule/診断の範囲だけを提示。非公開経路の存在、名称、拒否理由、内部表の数なども開示に該当する。内部の `ambiguous`、`denied`、`unresolved` がConsumerへ同じ `unavailable` として投影される場合がある。開示の粒度は#2516/実Host PolicyのOwnerで判断。

形式的には、**内部で意味的に解決する対象**と、**その解決結果をConsumerに公開できる情報**は異なる関数である。候補の開示Projectionを変更しても、同一受理意味・実Provider設定の内部操作対象が変わるべきではない。逆にGrant・Routeの変更をUI側で隠したからといって実対象資格が維持されるわけではない。

### 7.2 有限合成Reference RD-01〜12

[Python stdlib 12-test Reference](../fixtures/everything-data-impact/test_route_disclosure.py)で、以下の正負例を固定した。試験用 `Route` と `resolve/publish` は**製品APIや正式な権限／開示実装ではない**。

| ID | 独立に比較する条件 | 期待 |
| --- | --- | --- |
| RD-01 | 同じ実Route・Mapping/Scope/Purpose/Policyの別Alias | 同一候補クラスになり得る |
| RD-02 | Visible一件だがHidden別Routeあり | 内部の一意なSelectionへ自動昇格しない |
| RD-03 | Grant一件だがSemantic Route二件 | Grantを業務目的のRoute選択として使わない |
| RD-04 | Alias辞書順だけ変更 | 選択結果・Unresolvedが順序で変わらない |
| RD-05 | 明示Targetが存在し、公開可能名は隠す | 内部で候補を解決し、Consumerへ必要最小結果を投影可能 |
| RD-06 | 明示Primaryがdenied、Archiveは許可 | Archiveへ暗黙fallbackしない |
| RD-07 | 同じSemantic SubjectでもPurposeがread/updateで異なる | 当該Operation目的に限定して候補を区別 |
| RD-08 | Replica readはfreshだがPrimary updateはstale | Readの成立をWriteの現在状態一致へ転用しない |
| RD-09 | Hidden曖昧とDeniedの外部診断 | 開示不可なら双方Generic `unavailable` 候補（Timingの機密性は未実証） |
| RD-10 | 同じEndpointでもMapping versionが異なる | 単なる同名で経路を統合しない |
| RD-11 | 同じEndpointでもTenant Scopeが異なる | 同じ操作対象群と推定しない |
| RD-12 | Disclosure権限だけ反転 | 内部の意味的Resolutionを変えない |

**実測**：Python stdlib `unittest discover` **12/12 PASS**、`py_compile` PASS。GitHub保存blob `1373754109ff7c1bd6b305bb29eb0b6eeed8a8aa`は、ローカル実測ファイルのgit blobと完全一致。この実測が意味するのは、固定された合成判別ルールが期待ケースを満たしたことのみ。**本物のTEI `DeriveOperationSurface`／World Validator／SQL/Provider・Grant／Policy／外部Consumerに適用した証拠ではない**。

また、Generic Errorを一つにまとめるだけでは、Response timing、エラー種別の差、繰返し照会、件数依存の性能差による情報漏えいは証明できない。実際の情報非干渉/開示の判定は#2516/#2515とHost/OSが担当する。

### 7.3 P1の接合順序と増分ビルドに対する影響

1. **#2611 / Draft #2632**：最初に本物の `DeriveOperationSurface` の4件characterizationを実Goテストし、多Mappingが正式World Validatorの有効な入力かを判定。現候補の重複消失/辞書順依存という特性と、Ownerが採用すべき仕様を混同しない。
2. **#2507/#2622**：既存Pattern/Action Purpose/Mapping資格と、read-only discoveryをConsumer別の閲覧Projectionと独立させる。新Registry/専用UI/第二Resolverを増やさず、現在のContractへ接続。
3. **#2598/#1174**：選択されたProvider/Mapping/Request Targetを実Hostで照合。Primary/Replica等の時点、失効、Result/Receipt、実Write後の独立Readを一試行で結ぶ。候補だけの成功を実EffectやBusiness Completionとして扱わない。
4. **#2516/#1178／TAI**：外部Harness/SHO向けの説明は同じ内部対象を基にした用途限定Projectionとし、Hidden Routeの件数／存在を漏らさない条件を別Ownerが検証する。Productが開示を認めた要約と、Raw/内部Policyへのアクセスは別。
5. **#2600／Everything is Data**：Alias改名、Mapping変更、Purpose/Route変更、Disclosure Profile更新、Current Grant失効を、**別々の変更影響と再資格**として計測。明示的Route BindingとSource/Operation IDが同じでも実Provider対象が違い得る。Code再生成/意味再解決/実行前検査/Consumer再公開を用途別に扱う。

現行の#1119 P0実DB/Host縦切りを維持し、本§7は**P1 Reference・設計接合のみ**。正式Operation導出の修正、Runtime Resolver、Host認可、本番開示、Go製品試験、Owner意味受理は本件ではNOT RUN／未採用。
