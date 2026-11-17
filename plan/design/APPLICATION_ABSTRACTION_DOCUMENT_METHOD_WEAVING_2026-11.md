# アプリケーション抽象と文書手法を結ぶ、意味を保った構成の設計

> **位置づけ（2026-11-15）**：横断的な設計・実証計画。新しいCore Schema、DSL文法、文書Engine、共通Host、意味の第二正本を採用した文書ではない。ここに示す検証例は**未実行**。既存の実装・証拠の状態は[実装状況](../../docs/status.md)、着手順は[全体計画](../MASTER_PLAN.md)を正本とする。
>
> **目的**：アプリケーション、設計文書、開発手順に繰り返し現れる構造を見極め、既存のData／Role・Facet／Relation／Function／Process／Plugin／Projectionに編み込む。単にモデルの種類を増やすのではなく、**異なる抽象化の合成が安全に成立する条件**と、そこから生まれる機能を具体的に検証する。

## 1. 問題：同じ形の記述でも、同じ意味ではない

「表」「図」「文章」「型付きノード」を単一の標準構造として読めても、その形から業務上の意味は決まらない。たとえば、まったく同じ条件表が、現行の定義、提案中の変更、操作説明の引用、独立した試験入力として存在し得る。どれも構造化できるが、業務Ruleへ読み込んだり実行してよい資格は異なる。

一方、業務Ruleが一つの受理済み意味に対応する場合でも、Markdownの表、DSL、図、Inspector、生成コード、利用者向け説明は異なる構造を持つ。同じ意味を表示することと、表示面での編集を元の定義へ安全に戻せることは別である。

この二つの非対称性を同時に扱う。**形を共有しても意味を推測せず、意味を共有しても表現と操作の同一性を推測しない。** 既存の[意味RoleとSource束縛](EVERYTHING_DATA_INTERPRETATION_AND_SOURCE_BINDING_2026-11.md)、[原文保持](EVERYTHING_DATA_MARKDOWN_SOURCE_FIDELITY_2026-11.md)、[表現写像の適格性](COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)を、この判断の実装責務とする。

## 2. 「層」と「横断的な問い」を分ける

TEIのAuthoring → Canonical Semantic → Binding / Realization → Implementation / Executionという既存の四層は維持する。ただし、アプリケーション一般の構造は階層だけでは捉えられない。設計時・生成時・実行時・観測時に繰り返し現れる、次の**直交する問い**として整理する。

| 問い | 代表的な構造 | 既存の担当 | 混同してはいけないもの |
| --- | --- | --- | --- |
| 何を指すか | Identity、Subject、Occurrence | Semantic ID、Source Index | 表示ID、行番号、パス |
| どんな形・性質か | Type、集合、Relation、単位、時間 | Data、Facet、Trait | 物理配置と意味上の性質 |
| 何を主張するか | Definition、Claim、Rule、Constraint | Role、Rule、Acceptance | 引用例・推論候補と受理済み意味 |
| 何を組み合わせるか | Composition、Dependency、Operator | Function、Process、Plugin | 型の一致と意味上の適合 |
| どのように見せるか | Table、Graph、Document、View | Projection、SHO／Studio Consumer | 表示同値と逆編集可能性 |
| 何が外部へ起きるか | Invocation、Effect、Receipt | Operation、Provider | 実行要求と実副作用 |
| 何を確かめたか | Source Revision、Expected、Evidence | 既存の検証・受入機構 | 自己申告と独立観測 |

これは新しい七層アーキテクチャでも、七種類のCore Kind追加案でもない。新しい機能を考える際に「どの問いを既存機構で解けるか」「どの資格が不足しているか」を調べるための**設計上の比較軸**である。時間・状態・権限・来歴は複数の問いをまたぐため、単一層へ閉じ込めない。

## 3. 文書に固有の抽象能力をアプリケーションへつなぐ

文書は単なる入出力形式ではない。章節によるスコープ、表による関係の整理、図による視点の切替、共通部品の引用、変更の根拠、手順と期待結果の接続を備えた構成手法でもある。これらを専用の文書Engineに複製せず、TEIの既存能力に割り当てる。

| 文書手法 | 汎化する能力 | 初めに生まれる機能 | 制限 |
| --- | --- | --- | --- |
| 章節・見出し・注釈 | 文書内スコープと記述の役割 | 説明と定義を分けたSource診断 | 章節を自動で意味の名前空間としない |
| 定義表・決定表・遷移表 | 同じ表構造への用途別Role適用 | Rule／Data／Processの直接検証 | 表の見た目だけで実行Roleを付けない |
| 図・表・文章の併記 | 一対象の複数Projection | Inspectorと説明・図の連動 | 非単射の図から元情報を復元しない |
| 引用・再利用・差し込み | 参照、構成、版付き解決 | 一つの規則を画面と文書で再利用 | 文書の引用はOperationの起動ではない |
| 仕様・判断理由・具体例 | Claimと根拠と独立Expected | 変更理由から試験への往復参照 | 説明文を試験Oracleの代替にしない |
| 改訂・差分・手順書 | 型付き変更、依存、適用順序 | 変更影響と更新候補の提示 | 差分検出を自動受理や実行許可にしない |

文書上の一片について、**物理構造**（表・段落・図）、**記述上の役割**（定義・引用・例・判断根拠）、**意味への参照**（どのSubjectか）、**現在の扱い**（候補・受理・履歴）を区別して診断できればよい。これを新たな永続的なDocument Semantic Worldとはせず、Sourceの役割と既存Subject／Source Referenceの関係として検討する。

Markdownの原文順、列揃え、注釈、フェンス、空白はAuthoring Sourceの責務であり、Roleを解決したからといって全体を再直列化しない。文書閲覧時のOccurrenceはsnapshot-localであり、永続的Subject IDに昇格させない。引用を記録しただけで外部Sourceを取得・検証したと扱わない（[文書Draft引用の境界](DOCUMENT_DRAFT_CITATION_AND_OCCURRENCE_ID_2026-11.md)）。

## 4. 編み込みの中核は「合成可能性の検査」

異なる抽象要素をつなぐ場面では、最低限、次の問いを別々に判定する。すべてを一つの`composable`フラグに圧縮しない。

1. **構造**：入力・出力の型、粒度、同一性、多重度、欠損・NULL、完全性、時間、単位が適合するか。
2. **意味**：Role、Rule、Function、Relation、参照の適用条件が満たされるか。局所的な型一致から全体の意味同等性を主張しない。
3. **表現**：写像が保持・喪失する情報は何か。複数Viewの表示が同じでも、元のSubjectや編集対象が一意とは限らない。
4. **実現**：選択したPlugin／Provider／Targetが、必要な契約とEffectを実装できるか。Build-only部品と配布Runtime部品は区別する。
5. **現在資格**：参照Source、受理済みMeaning、Build入力と生成器、表示中のScope、実行時Grant、Receiptをそれぞれ照合できるか。

**合成に関する暫定法則**：

- 型・構造上の適合は、意味上の同等性・安全性・業務完了の十分条件ではない。
- 純粋な制約の論理的合成と、順序を持つ操作・副作用の合成は異なる。前者の交換可能性を後者へ持ち込まない。
- 複数のProjectionを連結したとき、失われたIdentity、順序、重複、粒度、時間、開示制限は途中で自然回復しない。
- 読取り専用Projectionからの更新は、単射性、必要な補足情報、対象一意性を確かめられる範囲に限定する。そうでなければ明示的な候補提示に留める。
- Source→Meaning→Build→Consumerのどこかが失効した場合、前段のbyte一致や過去のPASSから後段の現在資格を推測しない。
- 自己記述・標準Plugin追加にも同じ規律を適用する。解釈の有限な根と版を持ち、Plugin自身の記述を無制限な実行資格に昇格させない。

合成の必要条件は用途ごとに異なる。すべての処理に最も強い証明を強制するのではなく、**何を主張したいかに応じた必要条件と未解消条件を出す**。前提・保証の組合せは[Function Patternの合成設計](FUNCTION_PATTERN_ASSUME_GUARANTEE_COMPOSITION_2026-11.md)に、Traitの履行は[原子的Traitの実装義務](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)に委ねる。

## 5. 新しい能力へつなぐ四つの適用経路

**A. 文書を生きた設計面にする。** Markdownの定義表を既存Readerで読み、Subject／Source位置を付けてInspector・表・図にread-only投影する。編集の往復は保存済みSource、未保存Buffer、選択対象、元byte、意味受理を照合した差分提案から始める。[Authoring #2621](https://github.com/hat47x/tei-platform/issues/2621)、[Projection #2497](https://github.com/hat47x/tei-platform/issues/2497)、[View→Intent #2500](https://github.com/hat47x/tei-platform/issues/2500)が別々に所有する。

**B. 文書の再利用を機能の再利用へつなぐ。** 共通のRule／Functionを文書・UI・処理で参照し、共通部分と差異をCompositionやBindingとして表す。文書の差し込みとOperation実行は異なるため、Effect／Capability／Grantを持つ接続は追加検査する。ノード表現も正式なProcess／Relationの一Viewとする（[標準Flow](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)、[宣言合成](DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)）。

**C. 変更自体を再利用可能な操作として扱う。** 「項目追加」「Rule更新」「Provider変更」「説明文のみ修正」を区別し、影響対象と失効範囲を既存Dependency／Semantic Diffで導出する。変更のRecipe相当は既存Operator／Generatorに収まるかを先に確かめ、独立した変更Engineは作らない。[変更影響 #2582](https://github.com/hat47x/tei-platform/issues/2582)と[Source対応](SOURCE_REVISION_CORRESPONDENCE_AND_CONSUMER_IMPACT_2026-11.md)に接続する。

**D. 実コードから抽象化を育て直す。** TEI自身に現れた実際のRule、Generator、Studio部品から、責務・型・関係・判断を読み取って説明・図・DSLへ戻す。構文解析で分かった事実、テスト／実行で観測できた事実、Ownerが受理すべき意味を分ける。[実装からモデルへのdogfooding](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)をOwnerとする。抽象化後の定義の簡素化だけでなく、生成された実コードと実Consumerまで比較する。

## 6. 反証可能な最小ケース（DW-01〜DW-10）

以下は**既存の#1180/#2514に引き渡す独立Expectedの候補**であり、合格実績ではない。受入に使う際は前提Source・Selected Language／Plugin・Expectedを実装と独立して固定する。

| ケース | 一因子だけ変える入力 | 不合格にすべき振る舞い |
| --- | --- | --- |
| DW-01 引用と定義 | 同じMarkdown表を「説明中の引用」から明示選択した定義へ | 表を見ただけでRuleが有効になる |
| DW-02 文書Role | 定義表の列名は同じでRoleだけを決定表／対応表に変更 | 表の形だけで同じOperatorへ解決する |
| DW-03 原文保持 | Rule値は維持し、Markdownの空白・列揃えのみ変更 | 原文が不要に整形される／Meaning変更と断定する |
| DW-04 意味変更 | 説明文だけの変更と、閾値だけの変更 | 同じ失効範囲・同じ生成結果として扱う |
| DW-05 非単射Projection | 二つのRelationを一つのTree表示へ集約 | 見た目から更新対象を一意と断定する |
| DW-06 合成条件 | 同じ数値型だが粒度・単位・時点の一つが異なる | 型一致だけでFunction合成を許す |
| DW-07 Effect順序 | 二つの副作用付きOperationの順序だけを交換 | 純粋Constraintの法則で順序変更を正当化する |
| DW-08 Provider改版 | Meaning同一、GeneratorまたはProviderの実選択版だけ変更 | 旧Artifact／実行資格を無条件に再利用する |
| DW-09 A→B→A | Workspace／Pinを切り替え、古い非同期応答が到着 | 同名Aを理由に旧Sourceの表示／編集権限を復活させる |
| DW-10 自己適用 | 同一Ruleについて元記述→生成Go→Builder実消費を照合 | 単なるdelegateや合成Mockを製品セルフビルド完了とする |

まずDW-01／03／04を、現行の内蔵Markdown Rule Readerで扱う一つの実Sourceに限定して検証する。DW-05／06はread-onlyの型付き投影で検証し、更新や実Providerを先取りしない。DW-08／10は製品BuilderとStudio Stage1→Stage2の既存受入に従う。別Domainの反例として、数量・単位・観測時刻を持つ測定データを後続へ残し、受注の実装事情だけで抽象法則を確定しない。

## 7. 採否の比較と実装順

| 方式 | 利点 | 採否判断 |
| --- | --- | --- |
| 文書・表・図ごとに独立した意味体系を持つ | 個別画面の実装は速い | 意味の二重保守と横断検証の難しさが大きいため第一候補にしない |
| すべてを単一の巨大なメタモデルへ統合する | 全体を一つの形式で扱える | Domain差、編集原文、Effectの相違を潰すため先行採用しない |
| 既存標準構造へRole・写像・合成資格を適用する | Meaningを共有しつつ表現を残せる | **第一候補**。反例で不足が実証された部分だけ契約を拡張する |
| すべての表示を完全に双方向編集可能にする | 編集体験が統一される | 集計・情報損失があるため、逆編集可能な部分のみ段階導入 |
| 横断Hookを任意コード挿入として扱う | 拡張は容易 | Effect・順序・現在資格が不透明になり得る。契約化された地点だけから始める |

**実装順**：

1. **D0 現行Fit/Gap**：内蔵Rule Readerの選択、Source Index、Inspector、GeneratePositive、Builderの実際の接続点を記録する。比較対象は同一Source／版とし、欠けた辺だけを対象とする。
2. **D1 Headless実証**：選択済みMarkdown Rule一件について、引用と定義の区別、Source位置・原文保持、説明改版と条件改版の失効差を独立Expectedで検査する。#2620/#2621/#2497の範囲。
3. **D2 実生成・実Consumer**：同一Ruleから意味を持つGo判断を生成し、実Builderが消費したbyteとArtifactを一致させる。次に通常CLIの実Provider・独立Consumerを別資格として照合する。#4544/#2595/#4495へ引き渡す。
4. **D3 二表現・Studio自己適用**：read-only Inspectorと文書・図の同一Subject投影を示し、現在Pin/Sourceと実Windows Assetを確認する。編集は安全な候補差分から開始。Studio Stage1→Stage2 #2831/#4583のP0を維持する。
5. **D4 異質Domainとメタ拡張**：測定データ、Graph、Node Flow、Plugin自身の定義で同じ法則が保たれるか検証する。ここで初めて新Role、文法、変換APIの固定要否を判断する。

合格条件は「新しい用語を定義できたこと」ではなく、**手書き判断の削減、意味を失わない表現再利用、変更差分の局所化、独立Expectedが発見した誤接続、実Consumerでの成立**を確認できることとする。定義の短さだけを測らず、必要なPlugin、設定、Generated Source、差分調整まで含めた総維持費を比較する。

### Owner・既存文書との境界

- **標準構造とRole**：#2495／#2620／#2544、[Role自己記述](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)。Core変更はここで確定しない。
- **入力と表現**：#2621／#2497／#2500、[原文保持](EVERYTHING_DATA_MARKDOWN_SOURCE_FIDELITY_2026-11.md)、[複合表現](COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)。
- **処理と構成**：#2507／#988／#951、[前提と保証](FUNCTION_PATTERN_ASSUME_GUARANTEE_COMPOSITION_2026-11.md)、[Flow](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)。
- **実生成と自己適用**：#4544／#2595／#4495／#2831／#4583、[dogfooding](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)。
- **判定と証拠**：#1180／#2514。新しいScenario SSOTや受理Gateを作らない。

**見直し条件**：D1の時点でRole判定やSource位置が既存Readerで区別できない、D2で生成された判断が本体の実Builderへ結び付かない、または二つの異質Domainで共通抽象が却って複雑になる場合は、全体的な一般化を停止し、失敗した境界に限って設計を修正する。

## 8. 実装済みの足場から設計の問いを絞る（2026-11-15追補）

前節のDW-01〜10は、独立Expectedを作るための**横断的な問い**である。現行mainと他レーンの成果を照合すると、すでに局所実装で扱える部分がある。未実装という理由で同じReader／Inspector／Builderを作り直さず、残った接続条件へ試験対象を移す。

| 既存の実物 | 確認済みの限定的な範囲 | なお別に必要なこと |
| --- | --- | --- |
| `cmd/tei/cli_inspect_markdown_data_table_process_test.go`（#5058） | `tei.authoring/v1`で印を付けたData／Attribute／Ruleを、実製品CLI `inspect`と直接`InspectSubject`で比較。印のない例表や同名の非定義を拒否し、説明追記によるMeaningとSource証拠の差を観測する | Roleの一般的な競合・付け替え、表からの安全な編集、全DomainのDSL解釈 |
| `cmd/tei/cli_self_build_generated_rule_go_test.go`（#5051） | 選択したRuleから正値判断を含むGoを生成し、選択元Markdownのbyte同一コピーを通常`tei self-build`の実入力へ含め、完成Binaryの読み取りで確認する | Reader／Generatorの実装byteと元Sourceの正式Build閉包、公開Entry→実Provider呼出し |
| `internal/semantic/validationgen/positive_shape_obligation_baseline_test.go`（#5064） | Required Shapeについて、手書き分岐の現行挙動を独立Expectedに固定する | 自己記述から検査を構成するM1、手書き分岐の実置換M2 |
| `internal/semantic/validationeval/evaluator.go`と`validationgen/positive.go` | 一つの`semantic.Validation`を、既存の汎用評価関数で解釈する経路と、Goへ生成する経路が存在する | 同一Rule・同一入力に対する評価の対応、両方式の選抜・配布・Effectの契約 |

したがって、従来のD0〜D4をもう一度最初から実行する必要はない。**次の一歩は、同じ意味定義を複数の方式で実現したときに、どの同等性を主張できるかを狭く確かめること**である。局所試験の合格を、正式R2-P、通常CLI R3、Studio Stage1→Stage2に加算しない。

### 8.1 文書の役割と演算の役割は異なる

一つのMarkdownには、説明文、例示用の表、明示的なData定義、Ruleの宣言が同居できる。そのうちどれが**定義として選ばれたか**は文書内の印、選択されたLanguage Reader、実Sourceの位置で決まる。定義として選ばれた後に、そのRuleが**どの演算を意味し、どの実装を使うか**を別に解決する。

この二段階は、最近の自己記述設計で整理した`Shape`（形・参照先）、`Law`（意味的な判断）、`Binding`（具体的な実装選択）と対応する。ただし、文書上の「説明」「引用」「例」という記述の働きを、`semantic.Validation`のKindやOperator IDに直接写してはいけない。文書の論述上の役割と、アプリケーションで意味を持つRoleは、対応付けの前後で別物である。

図に矢印が描かれた、表に「実行」と書かれた、説明文に「必須」と書かれたという理由だけでは、Relation、Effect、Grant、実Providerを成立させない。これはNode View／Plugin接続／外部Evidence／組織関係にも共通する。

## 9. 「解釈して動かす」と「コードを生成する」を同じ選択の下に置く

標準データ構造の内容を実際の振る舞いへ移す手段は、コード生成だけではない。小さなRule、図表のパラメータ、関数の組合せは、**Target側に備わる汎用関数が型付き定義を直接評価する**ほうが単純な場合もある。一方、静的に展開した方が性能、配布、診断、自己構築に適する場合もある。

実現方式を固定された製品階層とはせず、受理対象の意味・Targetの制約・性能・変更頻度に応じた**選択結果**として扱う。

| 実現方式 | 向く条件 | 守る条件 | 初回に用いる既存資産 |
| --- | --- | --- | --- |
| **汎用関数で解釈** | 演算の種類が限定され、パラメータで十分に表現できる。実行先へその評価関数を含めてよい | 評価時の型・NULL／欠損、Operator版、実装選択、実Callを追える。読めない入力を正常結果へ変えない | `validationeval.Evaluate`／`EvaluateWithOperators` |
| **コード生成** | Targetへ静的に展開したい、コンパイル時の最適化や自己構築で手書きを減らしたい | 生成された判断が意味を持ち、生成器・原Source・Builder入力・Artifact・Consumerを別資格で照合する | `validationgen.GeneratePositive`＋`tei self-build` |
| **既存実装へ束縛** | 専門機能や外部Providerの既存実装に業務上の意味を委ねられる | Shape適合とLaw適合を別に評価し、選択済み実装と実Call、Grant、Effectを区別する | 既存Operator実装選択、Plugin／Service契約 |

いずれも**同じ受理済みMeaningを参照する**ことを原則とし、方式別に別の業務Rule正本を作らない。必要な場合はTarget固有のLowering・評価関数・Providerを持たせるが、TEI専用の常駐Hostや万能な実行Engineを前提にしない。生成方式と解釈方式を一つの生成物へ混在させる場合も、どの部分を何が実現したかは追跡できるようにする。

原Markdownを完成Artifactへ同梱する方式では、Build入力として正しく収録できたことと、**その原文を対象Consumerへ開示してよいこと**も別に判断する。説明文や引用に機微情報が入る可能性を、意味が同じだからという理由で無視しない。

どちらがよいかは、手書きの分岐・設定量、生成物の量、性能、起動時の更新可能性、必要な依存物、検証の容易さ、外部効果の境界を比較して決める。**「生成されたコードの行数が多い」「汎用関数が短い」というだけでは採否を決めない。**

### 9.1 第一の実測は正値Rule一件に限定する

同一`Workspace.Load`から選んだ`sales.order_line.quantity_positive`について、既存`validationeval.Evaluate`と、`validationgen.GeneratePositive`から通常`tei self-build`で生成した実Binaryの判断を比較する。入力は正値・ゼロ・負値・明示NULL・未取得の五つとし、**結果とNumberの読取り回数**を独立Expectedで固定する。

このPRの`TestSelectedRuleGoThroughRealCLISelfBuild`には、既存の実Binaryの結果照合に加え、同じ選択済みRuleを汎用評価関数へ渡す局所比較を追加する。双方が同じ結果でも、**別Processの局所評価と実Builderが使った生成Goの同値**だけを主張する。実Providerの選択・呼出し、正式Plugin Lock、Owner受理、Studio実OSでの受領は未証明のままとする。Goの実行経路を持たないTargetに、この五件の結果を一般化しない。

## 10. 文書の抽象化を進めるための追加反証

汎用評価と生成の局所同値を確かめても、文書の説明や図から、任意の定義を安全に実行できるようになったわけではない。そこでDW-01〜10の問いを、次の**差分一件ずつの検収候補**へ絞る。番号はこの計画内の索引であり、新しいScenario正本ではない。

| ケース | 意図した一因子差分 | 確かめる不合格条件と引渡し先 |
| --- | --- | --- |
| DW-11 文書Roleの範囲 | 印のない説明用表へ定義らしい列名を追加 | Language Readerが印なし表から意味定義を作成したら不合格。#2621／#2620、既存#5058の否定例を拡張 |
| DW-12 同じShape・違うLaw | Ruleの入力型と参照項目は同一、`> 0`から`>= 0`へ変更 | 型一致や正値での同一結果を根拠に意味同等としたら不合格。#2544／#988／#2514 |
| DW-13 解釈と生成の同値上限 | 意味・入力を固定し、評価方式だけ変更 | 結果・Read回数が異なる、またはRuntime Provider実証へ飛躍したら不合格。#4544／#1180／#2514 |
| DW-14 出典の独立失効 | 意味は同じまま、説明文だけ変更 | Source証拠の改版を無視する、または無条件で業務意味改版としたら不合格。#2621／#2595 |
| DW-15 Graphの情報損失 | 同じ二端点に同名の二Edgeを持つGraphをTreeへ投影 | Edge ID・向き・多重性を失ったTreeから一意の更新対象を推定したら不合格。#2497／#4198 |
| DW-16 選択済み実装の鮮度 | Rule形状は同じで、Operatorの実装選択だけA→B→A | 過去のAで得た実Call／Grant／Scenarioの資格を復活させたら不合格。#951／#1180／#2514 |

**実装順は二経路に分ける。** 製品CLIで確認できるDW-11／12／14は、既存MarkdownのReader・Inspector・Source Indexへ一因子の負例を追加する。DW-13は前節の正値Ruleの汎用評価・生成Go比較から始め、Target変更は後続へ残す。DW-15／16は既存のGraph・Plugin選択のOwnerが持つ試験に渡す。すべてを満たす新しい横断Runnerや共通Gateは作らない。

文書・業務・Graphの異質な三題材を比較して、`Shape`、`Law`、`Binding`が有効な問いとして保てるか確かめる。結果が揃わなければ、標準データ構造を増やす前に**どの意味を失ったために合成できなかったか**を説明できる状態へ戻す。現時点の受入済み機能を広げる根拠は、設計書の増加ではなく、反例を検出できる実装と独立Expectedである。

## 11. 抽象経路をテストの契約として扱う（2026-11-17追補）

アプリケーション抽象は、設計や実装の再利用だけでなく、**特定の条件で通るべき意味上の経路を通り、その間に保つべき状態が保持されたか**を検査する際にも役立つ。この用途では、生成されたGoの関数名、テストメソッド名、Nodeの表示名を検査対象の識別子にしない。

先行する[因果的な実行経路の検収§48〜61](PROCESS_EXECUTION_CAUSAL_PROJECTIONS_2026-11.md)では、静的な到達可能性、実際の通過、不変条件、Consumerの受領を分け、さらにRead ID・値の状態・Coverage・Scopeの不足を明らかにした。本書はその設計を複製するのではなく、**文書から選んだ意味定義を、複数の実現方式へ移した後も同じテスト義務で検査できるか**に対象を絞る。

### 11.1 一つのテストに必要な四つの識別

| 対象 | 保持するもの | 同一と推測しないもの |
| --- | --- | --- |
| **テストの義務** | 保存・受理されたDriver ScenarioのTest ID、独立Expected、対象Qualified Subjectと適用条件 | 言語別のテストメソッド名、実行順で振られた通番 |
| **意味上の通過点** | 必須Read／Guard／Rule／Operation、各点で保つ値の状態・粒度・Identity・Scope | Goの関数境界、UIの線、同じ名前の別Relation |
| **一回の観測** | Source・選択実装・Attempt、実ReadのID・状態・Coverage、Result、必要なEffect／Receipt | 別AttemptのRead、Fixtureの値、選択済みだが未呼出しのProvider |
| **実装上の位置** | 生成関数名、Goのテスト名、Artifact内の位置など、対象へ辿るためのLocator | Test ID、受理済みMeaning、現在のBuild／実行資格 |

実装がインライン化・関数分割・改名されても、対象Meaningと観測すべき条件が変わらない限り、テストの義務は変更しない。一方、同じCase IDを表示できても、実際に読んだ属性や状態が違えば不合格とする。**識別子の一致は、経路の正しさを証明しない。**

抽象経路を検査する際は、すべての内部処理を一列に固定せず、必要な通過点と因果的な前後関係、途中で保持すべき条件、禁止された作用を指定する。独立したpureな処理の順序が変わることは許し、認可前のCommitや、確定前の通知のような意味を持つ順序の破綻は拒否する。ただし、禁止作用が**起きなかった**と主張するには、その作用を観測できる範囲が閉じていることを別途確かめる。

### 11.2 正値Ruleの実Binaryで「結果だけでは不足」を検査する

現行mainの[実Go生成と汎用評価の対照](../../internal/operations/inspect_markdown_data_table_rule_binary_parity_test.go)（#5118／#5132）には、同じRuleから生成した`EvaluateSelected`、名前だけ変えた`EvaluateRenamed`、参照先を取り違えた`EvaluateMisbound`の三つを、別Processの実Binaryで検査する足場がある。

今回の最小差分では、この試験に**試験内のCase ID**と**観測したRead State**を加える。六入力（正値、0、負値、負の0、NULL、未取得）について、三実装それぞれで次の内容を突き合わせる。

- Rule IDと入力条件からなるCase ID。Go関数名を変えても同じIDを返すこと。
- 独立に固定したResult／Errorと、実ReadのQualified Attribute ID・回数。
- 読取時に観測した`present`／`null`／`not-fetched`。NULLと未取得が従来のP/G入口ではともにエラーになっても、**読取状態そのものを同一視しない**こと。

適格な関数名変更は、出力値とRead ID・状態を保つ。一方、誤った属性を読んで同じ値・同じ状態・同じCase IDを返しても、その実装は不適格である。これを独立Expectedと実Binaryの記録で検査する。

今回のCase IDは既存Go回帰内の**比較用識別子**であり、正式なDriver Scenario保存形式や一般Test ID生成器を実装したものではない。Read Stateも試験用Sourceが実際に受けた読取を記録したもので、実DBの全件性・Snapshot・現在Grantの証拠ではない。既存の`Evaluate`／生成GoがNULL・未取得についてエラーを返す範囲と、状態付き`Decide`が理由付き`undecided`を返す範囲も、**同じ実装資格にまとめない**。

### 11.3 一つの結果では判断できないケースを、異なるDomainへ渡す

| 検収候補 | 入力条件・一因子の変更 | 必要な結論とOwner |
| --- | --- | --- |
| **DW-17：関数名の変更** | 同じRule／入力／Readのまま生成Goの関数名だけ変更 | Case IDは不変、実Artifact byteの再資格化は別。#1121／#4544／#2514 |
| **DW-18：値の状態** | NULLと未取得を入替。両方の最終Errorは一致 | Read Stateの違いを保持。現在のLegacy P/G結果を状態付き`Decide`の同値証明としない。#988／#1180 |
| **DW-19：誤った参照先** | 読取先だけ別のAttributeへ変更し、Fixtureから同じ数値を返す | 同じCase ID・結果・回数でもRead ID違いで不合格。#2621／#2514 |
| **DW-20：範囲を変えた関係判断** | 同じGroupの期間重複Ruleで、確定した重複Witnessは残し、Coverageだけcomplete→partialへ | 観測済み重複の不成立は保持。ただし重複を観測しないpartialから全件正常を推測しない。#4198／#2514 |
| **DW-21：実行許可と作用** | 意味上の経路とResultを固定し、Providerの選択／Grant・Commit確認を変更 | Readerや実Goの局所成功から現在の実Call・EffectやConsumer受領へ飛躍しない。#951／#1174／#4495 |

DW-17〜19は今回のGo試験で扱う限定的な範囲、DW-20は現行`validationeval.Decide`の期間・関係の既存Fixtureを再利用する後続の横断検収、DW-21は実Provider／Effect／Consumerの別実験へ引き渡す候補とする。**すべてを追加した製品テストが合格した、という意味ではない。**

## 12. 文書とテストを接続する順序

文書には、定義表、説明、実例、期待値を近接して置ける。現行mainの#5143は印のある記録例と期待結果を`CheckRuleExamples`で照合できるようにし、#5146／#5148では日付の前後や関係行の期間重複へRuleの対象を広げた。これを、説明表から任意のTest IDや業務Meaningを自動受理できることと取り違えない。

次の順序は、新しいTest RunnerやScenario正本を増やさずに実施する。

1. **選ぶ**：明示されたLanguage ReaderとSource Indexから定義・例・期待位置を識別し、曖昧な表を業務Ruleへ昇格させない。
2. **検査義務を確定する**：既存Driver ScenarioにStable Test IDと独立Expectedを置き、対象Qualified Subject、適用条件、必須通過点、不変条件を固定する。文書の例を候補として使う場合も、実装と独立に受理する。
3. **複数方式で実行する**：まず同じRuleをP（直接評価）とG（Go生成）で試し、実Read ID・状態・ResultとTarget別の実装位置を別欄にする。D（外部Provider委譲）は実Call／Scope／Grantの観測があるときに限る。
4. **不足を残して提示する**：未観測の中間状態・Provider／DB全件性・Effect不在は`not-proven`として残す。確定違反のWitnessがあれば、その部分だけ不成立と判定する。
5. **変更に耐えるか確かめる**：関数名・図の配置・説明文の変更と、Read ID・RuleのLaw・実装選択・Source byteの変更で、必要な再資格化が異なることを確認する。

実装優先度は既存のP0（通常CLIの実Provider／独立Consumer、正式Source→Builder、Studio Stage1→Stage2実Windows）を変更しない。設計が検証すべき対象を増やすほど、**追加するのは新しい共通実行機構ではなく、既存Scenarioの一因子反例と正しい観測範囲**である。
