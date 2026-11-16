# TEIプラグインの貢献・実現方式・配置を分離する適格性設計

- **位置付け**：2026-11-16の横断設計・実装引渡し案。既存のContract／Provider、標準Data／Process、Build、Host Authority、独立Expectedを接続する。新しいCore列挙、Registry、Host、Graph Runtime、万能Adapterや実行許可の正本は提案しない。
- **対象**：Language Reader／Operator／Data Provider／Target Generator／API公開／Studio表示など、一つのPackageに同居し得る複数の貢献。
- **優先事項**：通常製品CLIからの実Provider Call（#4495）、正式Sourceから実Builder入力までのR2-P（#2595／#4544）、Studio Stage1からStage2実Windows WebView2の再構築（#2831／#4583）を止めない。
- **接続する既存設計**：[プラグイン入出力Mapping](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)、[役割別合成§25〜26](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)、[接続Evidence](PLUGIN_CONNECTION_EVIDENCE_AND_COMPOSITION_SLICE_2026-11.md)、[標準Flowの型付きPort](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)、[Studio／CLIの配置](STUDIO_CLI_BACKEND_PROCESS_DISTRIBUTION_UPDATE_2026-11.md)、[図表の実現方式PR #5070](https://github.com/hat47x/tei-platform/pull/5070)。

## 1. 移植可能なのは、実行ファイルではなく受理済みの機能契約である

TEIのプラグインは、あるHostへ読み込むバイナリというより、**既存の契約へ機能を提供する貢献**である。同じ意味上のOperationをGo単一バイナリへ静的合成しても、別Processへ隔離しても、認証済みServerへ委譲してもよい。ただし、実行位置が違えば、現在Grant、通信、失敗、再試行、公開先は改めて検収する。

次の四軸は独立に選択する。どれか一つを決めても残りは決まらない。

| 軸 | 正本・選択の責任 | 比較するもの |
| --- | --- | --- |
| **意味と入出力** | 受理済みMeaning、標準Data／Function／Process、Contract Owner | 必要な値・Read Context・結果・Failure・Effectの意味 |
| **実現方式** | Generator／Operator／Providerの既存選択・実装契約 | 汎用関数による解釈、Targetコード生成、外部委譲、混合。分類をCore enumにしない |
| **実行配置** | Target／Build Intent／既存Composition・Deployment Owner | 静的同梱、独立Process、別Hostへの接続。配置は業務意味を変更しない |
| **入口・公開** | Application Operation、CLI／Wails／Server Adapter、Host Policy | direct呼出し、HTTP公開、Studio表示、MCP等のAudience・Grant・Consumer |

したがって、API PluginをWailsのServiceやURLそのものとして定義しない。CLIは同じOperationの直接Callerであり、Wails Serverは必要なOperationだけを公開するHost Adapterとなる。CLIがHTTP ListenerやWailsを起動する必要はない。HTTP側がCLIをshell実行して意味の共通化を代替することも認めない。公開時には現在のHostがPrincipal／Grant／Audienceを評価し、入力のactor文字列やPluginの自己申告を権限へ昇格させない。具体的なHTTP Routeの実装・検収は既存Studio／CLI計画のOwnerで扱う。旧PR #5096自体はクローズ・未マージだが、現行mainのStudio／CLI配置設計§17には責務分離の記述が存在する。文書の収録をHTTP Routeの製品実装とみなさない。

## 2. 型付き接続はデータの構造だけでなく、保証の受渡しである

接続の記述量を減らす手段は三つある。**同一契約の全体Port接続**、**個別Fieldの明示Mapping**、**型変換用の明示Adapter**である。見た目のPort数を減らすために複雑な値を基本型へ分解する必要はない。逆に、一部しか必要ないConsumerへRecord全体を無条件に開示してもいけない。

各方式は既存のContract／Mappingで表現する。次の照合は新しい型体系の宣言ではなく、提供側の保証と利用側の要求を比較する観点である。

| 照合対象 | 典型的な不適合 | 不適合時の扱い |
| --- | --- | --- |
| Identity・項目ID・Cardinality | 同名の別業務キー、単一値へ複数件 | 別IDを統合せず、明示Mappingか拒否 |
| Presence・Completeness | NULL、欠損、未取得、部分取得を同一視 | 必要なReadを満たせなければ実行しない |
| Unit・Precision・Time・Grain | 円と銭、明細と月次集計、validAtとknownAt | 変換の意味と誤差／時点を明示し、保証不能なら拒否 |
| Source・Version・Law | 古いSource occurrence、異なる演算則 | 対象と版を固定し、必要な受理・再生成をやり直す |
| Audience・Disclosure | 内部項目がHTTP／Node Viewへ漏れる | Hostで公開範囲を再評価し、最小結果だけ開示 |
| Effect・Ordering・Caller | Guard迂回、二重Call、再送の競合 | 既存Lifecycle／Effect Ownerへ戻し、暗黙の順序やRetryを禁止 |

**純粋な宣言的Mapping**と、**外部参照・状態・Effectを伴う変換Provider**を区別する。変換可能であっても、その変換が必要な業務意味を保存する保証は別である。Adapterの暗黙挿入、出力項目名からの推測、型が合ったことによる権限付与は採用しない。経路全体の保証は各辺の適合の単純な積ではなく、Read範囲・時点・Loss・公開範囲を最後のConsumerまで合成して検査する。

## 3. 同じ貢献を複数Targetに載せるときの最小の受渡し票

新しいマニフェスト形式を先に作らず、以下の欄を既存Ownerの証拠から**読み取り専用で組み立てる**。不足する欄を推測で埋めない。

| 段階 | 確認したい識別・根拠 | 成立したと誤認しないもの |
| --- | --- | --- |
| Source・Meaning | Workspace、Subject、Source occurrence／revision、受理済み意味・Contract版 | Reader成功だけでの意味受理 |
| Offer・Requirement | Package／Contribution／Contract、必要Trait、入力・結果の形、Mapping、Source位置 | 候補の発見だけでの適合 |
| Selection | Scope、Target、Profile、exact Provider、Lock、依存Closure | Legacy InventoryとService Selectionの暗黙接合 |
| Realization・Build | 選んだInterpreter／Generator／Delegate、Toolchain、実Builder入力byte、Artifact/member digest | 生成候補だけでの製品収録 |
| Exposure・Invocation | 公開Entry、Adapter、実Process image、Host Grant、Attempt、実Provider Binding／Call | URL存在・結果値一致による実Call推定 |
| Observation | Result／Failure／Receipt、Effect OwnerのReadback、独立Consumer・Audience | ACKによるCommit確定、Trace欠落による未Call断定 |

必要な証拠が揃わなければ **unknown／partial／unqualified** を使い、実際に反証された場合は **mismatch／rejected** とする。新しい共通Readyフラグへ畳まない。World Fingerprint・Model Revision・Artifact Revision、Source A→B→A、Plugin実装版とGrant世代は別に保持する。

Stage1がビルドで使用したGeneratorは、Stage2に必ず含めるものではない。Runtimeで必要なContributionは逆に、Targetの実Imageと依存Closureから欠落してはいけない。同じPackageにGeneratorとRuntime Providerが同居していても、Build用とRuntime用の依存・版・同梱・呼出しを別に照合する。

## 4. Plugin自身をTEIで記述する際の境界

自己記述は、**宣言の検査 → 型付き貢献の生成 → 製品Builderで実消費 → 製品Entryから実Call → 次のTEI構築への利用**の順に一段ずつ成立させる。SourceがTEI記法であるという事実やDescriptorだけの生成を、Pluginロジックの自己構築と数えない。

最小対象は正値RuleとそのEvaluator／Generatorとする。既存Readerの同一Source・型・Read Footprintから、汎用解釈と生成Goの結果を独立Expectedで比較する。その後、正式Build Intentの中で選択したGeneratorとBuilder実入力byteを一致させ、生成した実装を選択Providerとして通常CLIから呼ぶ。初めてその実装が次のビルドを担った場合に、自己構築の寄与として計測する。生成Goが手書きDriverへ委譲した部分は別計測とする。

既存プラグインをTEIで全面再記述するのではなく、意味のある一つの義務（例：入力の正値判定）を移す。生成Goの手修正は避け、Target固有・案件固有の差分は明示Extension Point／Adapterへ置く。生成AIが作る実装候補も、固定入力・Target・許可された編集先・独立Expectedが揃う閉じた課題に限る。採用・実行権限は別判定とする。

## 5. 方式の比較は三例で行い、先に単一方式へ固定しない

| 例 | 比較する実現・配置 | 受入の焦点 |
| --- | --- | --- |
| **pure Rule** | 既存汎用Evaluator／生成Go、Go静的同梱 | 正値・0・負値・NULL・未取得、Read ID一致、選択A→B失効、通常CLI Call |
| **read-only Data Provider** | 静的Provider／独立Process／認証済みRemote | 同じ型付き結果とFailure、partial、Scope／Lock／Target依存、CLI direct・HTTP開示差 |
| **effectful Operation** | Host Authority下のProviderと必要な通信Adapter | 必須Guard前の実Call 0、単一Caller、Commit・通知別Receipt、effectUnknown後の照会優先 |

第一例をP0の追加Gateにしない。第二例で隔離や更新性の必要が明確になってから、動的ロード・子Process・外部Hostの追加費用を評価する。第三例で再送・失敗を正しく実証できるまでは、便利な共通Hookを理由にEffectfulな自動配線を公開しない。

## 6. 新たに渡す独立Expected（PCON-01〜09）

| ID | 一因子の変更 | 拒否または維持すべき不変条件 |
| --- | --- | --- |
| PCON-01 | 同じRecordを全体接続／二FieldだけMapping | 必要な二Field以外がConsumerへ開示されない |
| PCON-02 | Fieldの名前だけ同じでContract IDが違う | 暗黙接続を拒否し、Mappingの受理を求める |
| PCON-03 | completeな0件／partialな0件 | 両者を同じNULL・空集合として実行しない |
| PCON-04 | 円→銭変換で丸めと精度保証を欠く | Adapterの型適合だけでは承認しない |
| PCON-05 | Build専用GeneratorをRuntimeへ入れない／必要Providerを入れない | 前者は許容、後者はTarget Closure不適格 |
| PCON-06 | CLI directとWails HTTPで同じpure結果 | 意味同値は確認し、HTTPのGrant・Disclosureを独立評価 |
| PCON-07 | 選択Provider Bのまま実CallだけAになる | 同じJSONでも不一致を拒否 |
| PCON-08 | Source A→B→A、旧Artifact／旧Grantを再利用 | byte一致で現在資格を復元しない |
| PCON-09 | 必須Guardの欠落／任意Observerの不調 | 前者は作用前停止、後者は契約に従い独立報告 |

これらは[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ渡す**未実行の反例候補**である。既存XCJ／CPW／Flow／DPCと同じ内容を二重実装せず、主に「全体接続時の開示」「方式変更時の保証保存」「同一Operationの配置差」の不足分だけを実試験に採用する。

## 7. 採用条件と縮小条件

一つの貢献について、定義・型・Source・Provider選択・Build入力・実Call・Audienceの責任者を既存Ownerに対応付けられること、実行配置を変えても業務Meaningが暗黙に変わらないこと、CLI directがWailsなしで働くこと、Studioの表示用Edgeが実行指示や権限にならないことを採用条件とする。

新しいCore抽象や汎用Adapterを作らなければ一例も動かない、各入口が別Resolverを必要とする、第二のEvidence正本を生む、Stage2や実Providerの既存P0を遅らせる場合は、設計を一例のread-only検収へ縮小する。

**現状**：既存コード／設計の照合と次の受入条件の定義まで。PCONの実Go試験、実HTTP・外部Process試験、Studio Stage2 Windows試験、業務Ownerによる意味受理は、この文書では実施していない。実装順は[貢献・配置の小さな製品検収計画](../workstreams/PLUGIN_PORTABLE_CONTRIBUTION_EXECUTION_2026-11.md)へ渡す。

## 8. 最初の実装差分：値の選択と契約・公開の保証を取り違えない（2026-11-16）

現行の`internal/plugin/transform`には、閉じた純粋変換として`input`、`select-field`、`construct-object`が存在する。新しいPlugin Port Runtimeを作らず、`internal/plugin/transform/plugin_field_projection_conformance_test.go`で次の四点を比較する。

1. 同じRecordをそのまま出す場合、追加の`internalNote`も残る。**全体接続の候補適合は、公開可能な項目だけを運ぶ保証ではない**。
2. `orderId`と`amount`だけを明示的に構成すると、追加項目の有無で出力byteが変わらず、Consumerへ渡すValueに余計な項目は含まれない。
3. 必須`amount`が欠けた場合は失敗する。値を`0`やNULLで暗黙補完しない。
4. `amount`が**明示的NULL**なら、欠損とは区別して結果に残る。

**この試験の上限**は純粋Transformにおける出力整形である。Field名はContractの項目IDではなく、`Value`はunfetched／partialという取得資格を内包しない。現在Grant・Disclosure、業務意味、Adapterの単位変換、実Plugin選択、実Builder入力、通常CLI、Wails配信、独立Consumerの成立を主張しない。実Go testと対象HEADのCIは、そのHEADで成功を確認した場合にだけPASSと記録する。

次に、PC-1bで既存のContract／Mapping Ownerによる**明示Field ID・要求型・取得範囲・Audience**の照合を同じFixtureに接続する。PC-1cで一段のpure Adapterの単位・精度・NULL・Lossを検査する。名称が一致してもContract IDが異なる場合や、Readerが部分取得だった場合は、出力が同じでも不適格とする。これらをPC-2のBuildやPC-3のactual Providerへ先取りして加算しない。

また、mainに統合された[PR #5147](https://github.com/hat47x/tei-platform/pull/5147)はPlugin Resolverの診断コードについて、定数宣言も被覆検査の対象にし、拒否の負例を追加した限定的な改善である。未被覆例を理由付き許可リストへ残す点と、全Pluginの実行契約に適合したこととは区別する。失敗分類を返す接続の検収では、エラー文字列の一致だけでなく、既存の**診断コード**を固定する。

## 9. 部分取得と判定可能性を、Portの型適合から切り離す（2026-11-16）

main統合済み[Plugin I/O設計PR #5162](https://github.com/hat47x/tei-platform/pull/5162)と[標準Data設計PR #5163](https://github.com/hat47x/tei-platform/pull/5163)によって、接続の適格性と業務Ruleの判断可能性を、より正確に分けられる。既存`internal/semantic/validationeval.Decide`は`present`／`null`／`not-fetched`、Relationの`complete`／`partial`、および`satisfied`／`not-satisfied`／`undecided`と理由を表す。これらを`transform.Value`のNULLに押し込まない。

Consumerに渡せるかを考えるとき、以下の三層は別の問いである。

| 層 | 判定内容 | 適格性を誤認する例 |
| --- | --- | --- |
| **接続可能性** | Contractの項目ID、意味Role、型、Cardinality、Mapping、必要Contextの一致 | `amount`という名前だけを合わせ、別のContractを暗黙接続する |
| **判断可能性** | 実Readの値状態・Relation取得範囲・Ruleの種類に照らし、未取得部分があっても結論が不変か | partialを一律エラーとする、または未観測の重複が無いと断定する |
| **配送可能性** | 目的Consumerと現在Grant・Disclosure・Target・実Called Providerに対する許可 | 二Fieldに投影できることを根拠に、HTTP公開も許可されたと解釈する |

**判定不能はPluginの失敗と同義ではない。** 数値の正値Ruleに未取得があると、`undecided(value-not-fetched)`が妥当となる。部分取得のRelationでは、既知の重複から不成立を確定できる場合もある。他方、現時点で重複が見つからないことから一意性の成立を確定するには、Ruleの必要範囲が満たされなければならない。読取状態に矛盾があれば、判定不能ではなくReaderのエラーとして拒否する。

この区別をPC-1bへ渡す際、新しいStateful Port型やPlugin専用Evaluatorは作らない。Sourceの意味ID／Field Ref・Read Footprint・Requirement／Result・現在資格は既存Ownerが保持し、Node Viewは根拠を**読み取り専用で投影**する。公開可能なFieldが決まって初めて明示Projectionを検討するが、ProjectionはHostの認可判定を代行しない。明示Adapterの単位・丸め・Lossはさらに別のPC-1cで検収する。

追加の独立Expectedは既存[PIO-17〜23](PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)と[DPC-09〜13](SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)に結び付け、新しい検収ID群を増やさない。PR #5162／#5163の文書統合をPlugin Portや実Providerの製品PASSへ昇格させない。
