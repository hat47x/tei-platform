# Pluginの型付き貢献をCLI・Wails・生成物へつなぐ最小検収計画

- **状態**：2026-11-16の実装引渡し案。新しい製品Core／DSL／Runtime採用ではない。
- **上位設計**：[貢献・実現方式・配置を分離する適格性設計](../design/PLUGIN_PORTABLE_CONTRIBUTION_CONFORMANCE_2026-11.md)。
- **最優先P0**：既存のR2-P正式Builder、R3通常CLI実Provider、TEI Studio Stage1→Stage2 Windows実受領（#2595／#4544／#4495／#2831／#4583）を維持する。以下は小PRに分けて並行・従属させ、P0の新Hard Gateにしない。

## 1. 完了単位を「一Plugin」ではなく、ひとつの接続義務にする

一つのPackageが複数のContributionを実装しても、判定はContract・Scope・Target・実装義務の単位で行う。現行のProduct Inspector／Composition／Build／Hostから得られる事実だけを使い、別Ownerの対応が証明できない欄はunknownのまま残す。表示名、生成メソッド名、Node配置、同じResultだけで実装を同定しない。

最初の例は既存の正値Ruleと既存Inspectのread-only入口とする。CLIで同一Workspace／Subject／Source occurrenceを固定し、型付き結果と根拠のある接続だけを表示する。実Provider・公開HTTP・外部DBの成功はまだ数えない。

## 2. 実装パケット

| 段階 | 小PRの入力と作業 | 独立して確かめる完了条件 | 既存Owner |
| --- | --- | --- | --- |
| **PC-0：現行実体照合** | 既存Inspect、Plugin Inventory、Service Selection、Rule Readerの出力を同一Subjectで棚卸し | LegacyのResolvedを正式選択へ昇格しない。結合できない辺がunknown | #1124／#951／#1251 |
| **PC-1：型付き接続負例** | 全体Record／二Field／明示Adapterの小FixtureとPCON-01〜04 | 余分なField、partialとNULL、単位・時点の保証損失を拒否。新Runner不要 | #2497／#2620／#1180／#2514 |
| **PC-2：実選択と実Build** | 正値Ruleの受理済みSource→実Generator選択→Builder入力byte→実Artifactを同一Attemptで結ぶ | 生成の候補と実Buildを区別。Build-only／RuntimeのClosureも別検査 | #2595／#4544／#3267 |
| **PC-3：CLIのactual Provider** | 同一Operationでexact Providerを選択し通常製品CLIからactual entryを呼ぶ | 選択B／実Call A、古いLock、別Target、別Attemptを拒否。独立Consumerは別票 | #951／#1251／#4495 |
| **PC-4：Wails公開との同値・差分** | Wailsから既存Operationへ狭いHTTP Adapterを追加。CLIはdirect維持 | 同じpure入力のResult／Failureを比較。HTTPのHost Grant・Audience・Disclosureは別検収。Wails更新操作全体を公開しない | #1158／#1824／#4583 |
| **PC-5：Stage2とPlugin自己構築** | Stage1生成物で実Plugin由来の一義務を使い、Stage2 Windowsの実受領を確認 | 実Go生成・実Builder入力・Stage2実Image・WebView2 Consumer・手書き委譲を別々に計測 | #2831／#4583／#943 |

**依存関係**：PC-0／PC-1は並行可能。PC-2とPC-3は別々に完了させ、同じFunction／Artifact／Attemptで結べる場合にだけ縦断成功を宣言する。PC-4はCLI directとHTTPのAdapter差分であり、WailsがPC-3を代替してはいけない。PC-5の非Plugin Studio自己再構築はPC-4や汎用Plugin Editorの完成を待たない。

## 3. 最初に実装担当へ渡す具体的なテスト

- **PCON-01／02**：same Record全体接続と選択二FieldのConsumerが受ける項目集合を比較する。内部項目が追加された場合は公開面で除外され、同名の別Contract項目は失敗する。
- **PCON-03／04**：0件完全取得と0件部分取得、円と銭（精度保証あり／なし）を一因子変更する。単純な値一致で成功にしない。
- **PCON-05／07**：ビルド専用Generatorの非同梱を許し、実Runtime Providerの欠落と選択B／実Call Aを拒否する。
- **PCON-06／08／09**：CLI directとHTTPのHost Grant差、Source A→B→A、必須Guardと任意Observerの不調を、それぞれ現行Ownerの独立Expectedに割り付ける。HTTP・Effectの試験は該当Providerの実体が届いてから実施する。

合格票には実際の対象HEAD／Target、Source revision、契約とProvider Lock、Builder入力とArtifact、実Call Attempt、Consumer／Host Grant、試験名と観測範囲を記録する。証拠が無ければNOT RUN／unknownとする。既存#1180／#2514のExpectedを利用し、第二のScenario正本を作らない。

## 4. 方式比較・縮小判断・統合

検証対象をpure Rule、read-only Data Provider、Effectful Operationの順に増やす。比較値は、機能を追加するために必要なCore分岐数、手書きGo／TS行数、生成した実装のうち実Callされた割合、Build-only／Runtime依存サイズ、CLI直接呼出しの動作、Target変更で再資格化した範囲とする。「Plugin化できる」という自己申告を成果指標にしない。

旧API Plugin PR #5096はクローズ・未マージ。設計節が現行mainに存在することと、API公開の実装・検証が完了したことは区別する。旧PR #5133もクローズ・未マージだが本計画の同じ内容はmainにあるため、古いブランチを改めてマージしない。関連PRが競合するときは現行mainから小差分を作り直す。

各小PRは対象HEADのrequired CI、現在mainのRepository Health、変更範囲と独立Expectedの結果を別々に確認してから統合する。CI未完了・失敗、古いSource、未確認Consumer、業務Owner未受理をPASSに言い換えない。正本追加・新規Host・常駐Daemonが必要になった場合は、先に一つの失敗例と既存Ownerで解決できない理由を示す。

## 5. PC-1を三つの検収に割る（2026年11月16日）

| 手順 | 実物と確認事項 | 合格できる範囲・残す断点 |
| --- | --- | --- |
| **PC-1a（今回の小Goテスト）** | `internal/plugin/transform/plugin_field_projection_conformance_test.go`。全体Record、二Fieldだけの明示出力、余分なField、必須欠損、明示NULL | 既存Transformの**純粋な出力整形**のみ。HostのDisclosure／Plugin Port Mappingではない |
| **PC-1b（次の実装）** | 既存TypeRef／ContractのField ID・Role・Cardinality・取得完全性・Audienceを、PC-1aと同じ入力でread-only照合 | 名前や値だけでの接続を拒否。partial／unfetchedをNULLと混ぜない。Sourceへ戻せないBindingは編集不可 |
| **PC-1c（Adapter候補）** | 一段の明示pure Adapterを、固定された単位・丸め・精度・Lossの独立Expectedで比較 | Adapterの型適合は業務Law受理や実Provider Callにならない。自動Adapter探索はしない |

**既存実装を再利用する順序**：まずPC-1aの実Goテストを対象HEADで実行する。型・取得範囲の意味を持たない現行`transform.Value`へ新しい未取得状態を安易に追加せず、PC-1bは#1223／#2497／#951のContract／ContextのOwnerに渡す。PC-1cは#988／#1121に渡す。実際のSource→Read→Result→Consumerは#4495の非Builtin Providerの製品実Callと合流できる場合だけPC-3へ進める。

**独立Expectedの切り方**：PCON-01のうち「出力Valueが二Fieldのみ」という条件だけをPC-1aで判定する。PCON-02〜04のContract ID、partial、丸めについて、この純粋Transform試験からPASSを主張しない。PCON-05〜09も未実施のまま残す。Plugin Resolverの拒否コード試験はmain済み#5147の既存Coverageを利用し、第二の診断コード表を作らない。

## 6. PC-1bの一因子Fixtureと受入（2026年11月16日）

PC-1aが確認するのは、既存Transformの「出力を二Fieldに整形できる」という限定的な事実である。PC-1bでは[Plugin入出力設計§21](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)と[標準Flow設計§12.6〜12.8](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)の**既存Reader／Validator／型付きContract**を採用し、同じ`orderId`／`amount`のFixtureへ次の独立Expectedを接続する。既存`validationeval/reading_test.go`にある判断の回帰を再実装しない。

| 対照 | 一因子の変更 | PC-1bで求める結果・証拠 |
| --- | --- | --- |
| PIO-17 | 同じField Refを明示NULL／未取得へ | Source項目IDとRead状態を保持し、未取得をNULLへ変換しない。判断できなければ`undecided`と理由 |
| PIO-18 | 同じRelationの既読行で完全／部分取得を切替 | 既知の重複なら不成立を維持。重複なしのpartialは全件の一意性合格へ昇格しない |
| PIO-19／PCON-01 | 全体Recordへ非開示項目を一つ追加 | 型適合とDeliveryの許可を分離し、投影済み値だけでHost Grant／Disclosureを代行しない |
| PIO-20 | 値を変えずに実Readの項目IDだけ変更 | 実Read対象IDの不一致を拒否。値や生成メソッド名から一致と推定しない |
| PIO-21／PCON-07 | 同じ結果のまま選択ProviderだけA→B | 旧Pin・実Called・Source／Artifact現在性を独立判定 |
| DPC-09〜11 | 値状態と関係の取得範囲だけ変更 | `satisfied`／`not-satisfied`／`undecided`、Reader Errorを混ぜない。残りの未取得に依存しないときだけ判断を確定 |

**実装担当への渡し方**：#1223／#2497が意味ID・契約／項目の一致、#2620／#988が既存Readerと判断可能性、#951／#1251が選択資格、#1180／#2514が独立Expectedを担当する。まず一Source・一Producer Record・二Required Field・一Ruleだけでread-only照合し、Consumerの現在Grantと実Provider Callを伴わない段階は`candidate`／`unknown`として表示する。Rule例（`example of`）の一致は実Providerから読んだ証拠にならない。

**PC-1cへの移行条件**：PC-1bでField Ref／必要Read／欠損・NULL・部分取得・未確定の扱いが固定できた場合だけ、単位と精度を明示したpure Adapterを挿入する。新しいAdapter探索器・Stateful Port Runner・第二Validation Registryは作らない。PC-2／PC-3／PC-5を先行停止させない。

## 7. PC-1bの最初のGo負例を、次の製品接続へ渡す（2026年11月17日）

`internal/plugin/transform/plugin_field_presence_boundary_test.go`で、既存の`optional select-field`による「Field欠落とNULLの出力一致」を確認し、状態付き`Decide`の理由付き結論と実際に要求したAttribute IDを別票として照合する。これを[設計§10](../design/PLUGIN_PORTABLE_CONTRIBUTION_CONFORMANCE_2026-11.md)の**局所反証**に対応付ける。既存`reading_test.go`のRule一般試験やPC-1aの二Field Projectionを複製する目的ではない。

| 順序 | Ownerへ渡す小差分 | 独立Expectedと未達条件 |
| --- | --- | --- |
| **PC-1b-0：損失の顕在化** | `optional` Mappingの出力値が同じでも、NULLと`not-fetched`ではRule理由が異なることを同じGo回帰で固定 | PIO-26／DPC-09。結果値・出力JSONを取得完全性の証拠にしない |
| **PC-1b-1：実Source Ref付き読取** | 一Sourceに選択された`std.decimal`属性・RuleをSource Indexから読み、必要Attribute ID・読取時点・状態を一つのread-only Requirementに照合 | PIO-25／20、#1223／#2620。Field名一致をRef一致に昇格しない |
| **PC-1b-2：必要なRelationの範囲** | 同じRelation行で`partial`／`complete`、既知重複あり／なしを比較。失敗・判断保留・確定違反を分ける | PIO-17／18、DPC-10〜11。完全性をProviderの申告から自動承認しない |
| **PC-1b-3：実Consumerへ引渡し** | PC-1aのProjectionとPC-1b-1の実Read Ref・Coverageを既存Function Contextへ結び、実選択Providerと独立Consumerで同一Attemptを検収 | PIO-19／21、PCON-06／07。公開Grantや現在Providerの適格性は別のOwnerが判定 |

この順番は**追加Hard Gateではない**。PC-1b-1のSource→Read照合と通常CLIの実Provider検収#4495を並行させ、同一Attemptに揃ったときだけ縦断成立を主張する。PC-1cの一段Adapterは意味保存・精度・丸めを別検証し、`optional`投影を状態保存Adapterとして使わない。EditorのNode／Port展開は既存Source Refの投影であり、実CallやDisclosure Grantは発生させない。

**実測と限界**：PC-1b-0は対象HEADのCIで成功確認した範囲に限りPASSとする。実Binding、Native SQL／DB読取、HTTP、Studio Stage2はこの小回帰には含まれない。実装採否と独立Expectedの正本は#1180／#2514を継続利用する。
