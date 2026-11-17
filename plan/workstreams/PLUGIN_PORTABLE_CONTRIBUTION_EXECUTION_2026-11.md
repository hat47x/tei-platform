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

PC-1aが確認するのは、既存Transformの「出力を二Fieldに整形できる」という限定的な事実である。PC-1bでは[Plugin入出力設計§21](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)と[標準Flow設計§12.6〜12.8](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)の**既存Reader／Validator／型付きContract**を採用する。PC-1aの`orderId`／`amount`は純粋投影用Fixtureであり、PC-1bの実Query行Readerが使う組織Relationとはまだ同じ入力ではない。Field Refの対応を受理する前に両試験を一つのE2Eへ合成しない。既存`validationeval/reading_test.go`にある判断の回帰を再実装しない。

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

## 7. PC-1bの実装入口を「値の状態」と「取得範囲の根拠」に分ける（2026年11月17日）

[Plugin入出力設計§23](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)に対応し、PC-1aの限定実証はmain統合済み[#5156](https://github.com/hat47x/tei-platform/pull/5156)を使う。実装の再発明を避けるため、PC-1bの初回は既存`internal/driver/validationread.FromHeaderDetail`と`validationeval.Decide`を対象にする。[#5148](https://github.com/hat47x/tei-platform/pull/5148)の期間Rule、[#5149](https://github.com/hat47x/tei-platform/pull/5149)の結果行Readerもmainに存在する。ただし、それぞれのGo試験を別々に通したことは通常製品CLIの実Provider→Rule→Consumerの成功ではない。

**最初の小PR（PC-1b-0）**は、既存のRelationと二つの型付きAttributeを使い、Query解決で得たRelation／Target／AttributeのIDと、同じAttemptの実行結果行の観測を突き合わせる。`validationread.Detail`が返す値あり／NULL／未取得と、`Decide`の結果・判定不能理由を、手書きExpectedで検査する。意図的に別Attributeの同じ値を用意し、結果が等しくてもRead IDが異なれば不一致となる負例を設ける。値の読み方に新しいGo実装やPort Schemaを導入せず、SourceとFieldの結び付きが既存契約だけで確認できないなら、そこで未受理として止める。

**次の小PR（PC-1b-1）**は`coverage=complete`を単なる呼出し側申告として記録し、QueryのFilter／Paging／継続位置、Scope／As-of、Provider受領の証拠によって全対象を取得したと説明できるときだけ`verified-complete`とする。既存ReaderのCoverage語彙を勝手に増やすのではなく、検収票の「主張」と「裏付け」を分ける。反例が既読行にあれば、partialのままでも規則違反を確定できる。逆に反例が無いpartialを成功へ昇格させない。

**PC-2で並行して確認する診断の穴**は、追加の万能Resolverではなく既存のPlugin Ownerへ渡す。

| 実コードの観測 | 受入時に区別すること | 検査・Owner |
| --- | --- | --- |
| #5160：解析不能なPlugin版が適合候補なしへ畳まれる | 版宣言の誤り／解析可能だが不適合／候補なし | 既存Resolver・management、PCON-02 |
| #5197：Capabilityが0件と複数件で同じfalseになる | 不在／曖昧／一意に選択済み | 既存Plugin Catalog、PIO-34 |
| #5201：Lock経由の失敗で元Resolver理由が落ちる | Lock不一致／選択要件不適合／原因未確定 | 既存Lock・Resolver、PIO-35 |
| #5204：Supports述語は宣言集合の所属判定 | 宣言対応／実Provider資格・実Called／Grant | 既存Capability／Host、PIO-36 |

**検収の提出物**は一対象・一Attemptの対象ID、Source版、Field Ref、実QueryのScope、Provider・Readerが観測した値の状態、Coverage申告と裏付け、Rule結果とReason、独立Expectedの対照で十分である。データに無い根拠を得るための新たなEvidences Storeは作らない。StudioやNode Viewへ結果を投影するのは、この最小試験が成立してから別Consumerで行う。

これらは既存#1180／#2514のExpectedと#2586／#1251の読取・Provider Ownerに引き渡す**実装計画**であり、本節の更新では新しいGo／DB／Windows試験を実施していない。通常CLI実ProviderとStudio Stage1→Stage2のP0は止めない。

### 7.1 状態を失う投影の負例を、実Readの適格性へつなぐ

[PR #5214](https://github.com/hat47x/tei-platform/pull/5214)では`plugin_field_presence_boundary_test.go`を追加し、任意Fieldの**欠損と明示NULLが同じ値へ投影される**こと、および`Decide`側が別途渡された`null`と`not-fetched`を理由付きで区別することを固定する。Ruleの実要求Attribute IDは記録するが、このテストが実Source／QueryのBindingや実DB Readerの呼出しを接続したことにはならない。既存の`validationread.FromHeaderDetail`と`functioncontext`の観測情報は、それらのOwnerの正本のまま使用する。

次の最小実装は§7の一Source・一Attemptを維持し、**Requiredの意味ID → Queryの選択・Binding → 実ReadしたField ID／値状態／Coverage → Rule判断 → Consumer受領**を段階別に検収する。値が同じでも別Attributeを読んだ負例、NULL／未取得、partialでも確定する違反、虚偽complete、0件取得、Provider Errorを独立Expectedへ照合する。`UNBOUND_DATASOURCE`や実Provider不在を`undecided`という業務判断へ置き換えない。新たな状態型・万能Binder・独立Evidence Storeを追加せず、既存#1180／#2514および#2586の受入を用いる。

**位置付け**：PC-1bの局所負例の合格だけでPC-1b全体・PC-1c・正式R2-P／R3・Stage1→Stage2の達成とはしない。UIのPort展開は受理済みMappingの投影であり、CLI directの実ProviderとWails公開の権限は別資格とする。
