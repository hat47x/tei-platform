# TEI MVP dogfooding：判断の移管から次世代の自己構築まで

- **状態**：2026-11-16の実装・受入パケット。既存のM0〜M3、AW-0〜3、R2-P／R3、Desktop P0を具体的につなぐ計画であり、新しい製品実行の合格報告ではない。
- **対象**：既存 `internal/semantic/validationgen/positive.go` に残るRequired Shape判断を一条件ずつTEIの定義・実装へ移し、最終的にTEI自身の次の版がそれを使用して再構築されること。
- **既存の設計**：[実装→モデルdogfooding](../design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md) §13〜15、[自己記述](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md) §51〜53、[図表→実装の選択](../design/DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md)、[製品完成計画](PRODUCT_COMPLETION_PLAN.md)。

## 1. MVPで証明する価値を一つに絞る

TEIで記述されたアプリが動くことと、TEI自身をTEIで作れることは異なる。最初のdogfoodingの主張を、**既存Goが担当していた判断をTEIが実際に引き受け、その判断の変更が次世代の製品実行へ届く**ことに限定する。

ここでの「一義務」は、単に関数や生成Goの一行ではない。対象Subject、受理する入力、拒否する入力、失敗の区別、要求される実経路、独立Expectedを固定した判断の単位である。メソッド名、Graphの描画ノード名、Target側Symbolは、その義務の正本IDにならない。

同じ意味に対して汎用Functionによる直接評価（P）、Target Go生成（G）、外部Providerへの委譲（D）、混合（H）を比較できる。しかし**各Targetで実際に選択・使用した方式**は別の事実である。Pで十分な一般業務機能に生成を強制しない一方、セルフビルドM3には、意味を持つ生成物が実Builderに消費される経路を少なくとも一つ求める。

## 2. 現行mainの実証と、まだ存在しない接続を分ける

| 現在の足場 | 実コード・受入上限 | 次に閉じる辺 |
| --- | --- | --- |
| 型付きRuleと生成Go | `internal/semantic/validationgen/positive.go`、[PR #5077](https://github.com/hat47x/tei-platform/pull/5077)のData表＋Rule回帰。選択した型・Read先からGoを生成 | この検査器自身のShape条件をTEIから実行するM1 |
| 独立基線 | `positive_shape_obligation_baseline_test.go`。正常形と形状・型の拒否、Go package名拒否を別対照 | 後続の生成器・判定器がExpectedを自己生成しない |
| 実CLI Build | `cmd/tei/cli_self_build_generated_rule_go_test.go`、[PR #5009](https://github.com/hat47x/tei-platform/pull/5009)／[PR #5087](https://github.com/hat47x/tei-platform/pull/5087)。生成Goを通常 `tei self-build` でBinary化し、正値・ゼロ・負値・NULL・未取得とRead一回を直接Evaluatorと比較 | M2の手書きShape分岐置換、元Source・Generator・Builderの正式な同一Attempt閉包 |
| Build差替え拒否 | [PR #5056](https://github.com/hat47x/tei-platform/pull/5056)はPlan解決前の生成Go byte取り違えを拒否。Plan解決後の生成Go変異拒否も[PR #5109](https://github.com/hat47x/tei-platform/pull/5109)としてmainへ統合済み | 別Attempt・時間差差替えの反証を実Runnerまで通す |
| 自己記述Data | `tei.authoring/v1`でDataとして表現するFixtureとReader試験は現行mainに存在する（[PR #5097](https://github.com/hat47x/tei-platform/pull/5097)自体は未マージでクローズ） | Dataを読めることを実Shape契約の判定と混同しない |
| 抽象経路の指定 | [PR #5110](https://github.com/hat47x/tei-platform/pull/5110)のAW-0〜3は計画としてmainへ統合 | 経路の存在・実通過・状態保存を別の実行証拠で確認 |

**現在の停止線**：`GeneratePositive`のGoには、`ID`・`Attribute`の非空と`Relation`の空をまとめて確かめる手書き分岐が残る。M1〜M3の製品実置換、自己記述Fixtureの最新mainでの実行合格、正式R2-P／R3、Studio Stage1→Stage2の新たな完了を主張しない。PRの状態と現行mainでの実機能の有無は分離して照合する。

## 3. 最初の判断移管を失敗させ得る条件

`GeneratePositive`の複合条件は、実装移管の計測では次の**三つの原子的義務**へ分ける。組としてのRequired Shapeを一件と呼んでも、完了数の分母を曖昧にしない。

| 義務 | 独立した反例 | 誤った代替 |
| --- | --- | --- |
| SH-1：Rule IDが必要 | `ID=""`を拒否 | 名前付きGo Functionの存在をRule IDの保証とする |
| SH-2：参照Attributeが必要 | `Attribute=""`を拒否 | `std.decimal`だけでRead対象が一意とみなす |
| SH-3：positive RuleではRelationを指定しない | `Relation`に有効なIDを指定しても拒否 | Data表の`required=no`を`forbidden`と読み替える |

`Kind=positive`の適用条件、解決済み`std.decimal`、生成Goのpackage名、NULL／未取得／Read失敗時の評価は**別の義務**として維持する。SH-1〜3を移したことをもって、positive Law全体やTarget適合の移管完了とはしない。

採用すべき既存標準Data／Facet／Role／Operatorの表現が不足する場合は「未表現の条件」と不足した接続を明記し、Readerへ説明表の意味を密かに埋め込まない。最初のPRではSH-1〜3のうち**実際に型付き意味として受理し、実行できる一条件だけ**を選ぶ。新Grammar、第二Meta Interpreter、試験専用Runner、Hostは作らない。

## 4. 実装PRを分ける境界

| Packet | 変更と照合する製品経路 | 合格条件・停止条件 |
| --- | --- | --- |
| **D-0：固定基線**（M0、既存） | `Workspace.Load`→Source Index→選択Rule・解決型→旧`GeneratePositive`／Independent Expected | 各拒否例、元byte／選択Reader、比較用旧Goを固定。旧GoをOwner意味受理としない |
| **D-1：一条件のTEI実判定**（M1、最初の新コードPR） | 既存のReaderと意味を選択→既存Operatorの直接評価または既存Generatorで検査を実行 | SH-1〜3の選択条件について正例と拒否例が独立Expectedに一致し、**どのTEI入力・実装が判断したか**を示す。説明DataやDescriptorだけなら失格 |
| **D-2：製品の旧分岐を置換**（M2、別PR） | 本来の`GeneratePositive`製品経路がD-1の実判定を使い、対応するGo手書き条件を削除 | 同義のGo条件を二重保持しない。D-1を呼ばない通常製品経路、黙った旧Goフォールバック、テストだけの配線は失格 |
| **D-3：生成元と実消費の閉包**（R2-P、既存Owner） | 原Markdown／Reader／Generator版→生成Go byte→Go Entrypoint BuildIntent→実Builder入力→Artifact→実Binary | `definitionSource`を原DSLと誤置換しない。選択G／実P、生成B／配置A、Plan解決後の差替え、旧Source／Plugin版を拒否。現行の既存Recipe・Builder・Catalogを使用 |
| **D-4：次世代での実使用**（M3、Desktop P0） | Stage0 Seed→Stage1実TEIのProcess／実Builder呼出し→Stage2実TEIが検査を使う | Stage1自身が選択Sourceと実Generatorを用いStage2を生んだ因果を確認。Seedによる二重ビルド、Stage2の旧検査器利用は失格。正式Studio Windows受領は#2831/#4583の別合格 |

**PRの順序**：D-1で実行可能な条件を確定するまでD-2の製品分岐は削除しない。D-2の差分は検査器の追加だけでなく、**既存の手書き条件を実際に減らすこと**を必須にする。D-3は既存RuleのR2-P検証と並行してよいが、D-2で生成器自身が移管した判断の実使用を証明するには、当該移管を含む同一Build Attemptが必要である。D-4のWindows資格はD-1の先行条件にしない。

## 5. 抽象経路・型付き接続・変更前後の検収

同じSubjectのテストで識別するのは **`Subject + 義務 + 前提 + 要求経路 + 不変条件 + 独立Expected + Attempt`** である。Go関数名・Designerの表示ノード・GraphのEdgeラベルが変わってもSubjectを乗り換えない。

| 反証 | 観測する証拠 | 合格しないときの扱い |
| --- | --- | --- |
| **Q-1 選択G、実CallerはP** | Selected implementation／Target／実Call先と入口を別取得。結果値が一致しても拒否 | #951／#4495／#2514へ。旧実装へ黙って退避しない |
| **Q-2 定義内容を実変更** | SH-1〜3の選択一条件を変えた前後のTEI Source、判定結果／生成Go、Stage2が実使用する版 | 旧Expectedを改定せずに変更後を合格扱いしない。Meaning変更はOwner受理と新Expectedが必要 |
| **Q-3 説明・レイアウトのみ変更** | 原byte／Source Indexと判断Go、UI Resource／Bundle Artifactを別照合 | 旧出典を再利用しない。業務判断のGo再生成・再ビルドは実依存が同じなら機械的に強制しない |
| **Q-4 生成元とBuilderが不一致** | 実Generator byte、staged `generated.go`、Plan解決後の実Builder読取、Resultの出所 | 出力が正しくても旧Plan／旧Inputを拒否。PR #5056／#5109と重複実装しない |
| **Q-5 Source A→B→A／Operator差替え** | Source Revisionと選択Pin、Generator版、旧Attempt、現在の実Call | byte一致だけで旧Evidence・現在Grantを復活させない |
| **Q-6 型付きPlugin配線／Hook** | Whole／Field／明示AdapterのInput/Output型、Multiplicity、必須性、単位、Loss、Effect境界 | Nodeの配線表示を適格性とみなさない。型不一致・必須欠落・無断Field公開を拒否 |
| **Q-7 実Callは成功、Consumer未受領** | Actual Provider、Effect／Receipt／独立Consumer、Studio WebView2実Imageを別記録 | 実行図やCLIの成功から外部公開・業務確定・Stage2 Windowsへ昇格しない |

Q-1〜7は新しいTest RegistryのIDではなく、[自己記述§53](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)のSD-53反証、[既存の独立Expected](https://github.com/hat47x/tei-platform/issues/2514)へつなぐ**作業票の見出し**である。Source／Reader、Meaning／Law、実現選択、Build、実受領の五層を一つのPASSへ畳み込まない。

## 6. 維持負担の計測と次の採否

対象をSH-1〜3の**三条件**に固定し、D-1／D-2の改訂前後で、次を同一Scenario・Target・Expectedで比較する。

1. TEI自身が実際に判定した条件数／3、旧Goに残る条件数／3、外部実装へ委譲した条件数／3（判断の二重保持は別記）。
2. 定義・Generator・Hook・Target設定・手書きGoの変更ファイル数と箇所、レビューに必要な手作業、独立Expectedの追加／変更点。
3. Source変更→診断→プレビュー→生成→実Builder→Consumerの各所要時間、再Buildの必要性、再利用したArtifactと再資格化が必要なEvidence。
4. 専門家が読む順序（入力→対象→条件→結果→評価不能）と、表示Projectionから元Sourceへ安全に戻れる範囲。

**採用判断**：生成Go行数・図の数・テストGreen件数は成果の分子にしない。M2まで達しても手書き判断が減らない、SourceやGeneratorの保守が増える、既存汎用Functionの直接解釈の方が単純である場合は、その方式を採用する理由を再評価する。保守費が勝る場合は一部をOpaque Goとして維持し、TEIが引き受けなかった義務を明示する。

## 7. Owner・並行P0・マージ条件

- **Language／意味**：#2620／#2621／#988／#1018。`*.tei.md` の専門家可読性を守り、Go構文をDSLへ露出しない。
- **Generator・Build／Plugin**：#1121／#4544／#2595／#943／#951／#1251。Go単一バイナリ、外部Host、CLIとWails非依存の共通Operationを維持する。
- **独立Expected／変更失効**：#1180／#2514／#2600。選択・実使用・出典・Owner受理を分ける。
- **外側の最優先実証**：#2586の通常CLI→実Provider Read、#2831／#4583の非Plugin Studio Stage1→Stage2／Windows WebView2は**独立P0として並行**する。D-1が完了するまで両P0を停止しない。

各PRは同一Issueの未解決義務を一つ減らす差分に限定し、mainと対象PRの**exact HEADのrequired CI**、最新mainのRepository Health、変更ファイルの競合、別のOwnerのIndependent Expectedを確認してから統合する。未実行のGo試験、ドラフト文書、別HEADの成功を合格根拠に流用しない。Generator／Reader／Stage2の実証が不足する部分は、不足する辺を明示したまま次のPacketへ返す。

## 8. M1の成立を先に確かめる：Data定義と実際の検査は別物

最初の実装PRを始める前に、現行Reader／Operator／型付きMappingで次の三つを区別できるかを実コードで確認する。

| 元Goが検査する条件 | Data項目表だけで言えること | M1に必要な残りの意味 |
| --- | --- | --- |
| `rule.ID != ""` | `required=yes`なら項目の存在が必要だと表現できる | **空文字列の拒否**。Fieldの欠落・NULL・空文字列を同じものとして扱わない。現在のGoは厳密に`""`との比較であり、空白だけのIDに別の規則を加えない |
| `rule.Attribute != ""` | 参照項目を必須として記述できる | **空の参照IDの拒否**と、実際の`semantic.Validation.Attribute`への型付きBinding。型`std.decimal`の解決結果だけでは参照先の存在を保証しない |
| `rule.Relation == ""` | `required=no`は「なくてもよい」の意味 | **positive Ruleに限ってRelationを禁止する**という条件。optionalを禁止に読み替えず、Rule Kindの条件付き適用を保つ |

既存の`ValidationDeclaration` Dataを正しく読み込むこと（D1／D2自己記述）と、**実際のValidationインスタンスにこの条件を適用して受理・拒否すること**（M1）は別である。Data表だけでは未実現の条件が残る場合、`M1=unsupported`として不足する「型付きインスタンスへのBinding」または「既存Operatorの適用能力」を明示する。暫定的なGo手書き`if`をTEI Checkerと命名して合格にしない。

**独立反証の入力境界**も分ける。元`*.tei.md`の製品Readerが空IDを文法段階で拒否するなら、そこから不正な`semantic.Validation`が選択されたと偽装してはいけない。Source選択の正例は製品`Workspace.Load`で確認し、`GeneratePositive`の型付き入力に対する不正Shape負例は既存`positive_shape_obligation_baseline_test.go`のように明示的に与える。双方の証拠を連結しない限り、Raw Sourceからの全経路でM1が成立したとは認定しない。

最初の**実装可能性判定票**は、`対象条件 / 既存の定義表現 / Instance Binding / 実Checker・Operator / 元Goとの差 / 既存Independent Expected / 選択・実使用 / 未実装の辺`を一行にする。SH-1〜3のいずれかで元Goと**同じ真偽と拒否理由**を実現できた場合だけ、D-1でその一条件を実行する。存在・空・禁止のどれも表現できない場合は、専用メタRuntimeを追加せず、#2620／#988／#1121へ必要な最小能力を戻す。

## 9. CLIとStudioでTEI自身を検査する一往復

M1〜M3は生成器側の自己置換であり、**利用者がTEI自身の設計を扱えること**は別のdogfoodingで検証する。両者は同じSource、正規Subject、Operationを共有するが、CLI／Wails／Designerそれぞれに業務意味の正本を複製しない。

| 段階 | 最初に使用する実経路 | 認める到達点と次の反証 |
| --- | --- | --- |
| **U-0 read-only inspect** | 実`tei inspect`別OSプロセスと直接`operations.InspectSubject`で同じ`quantity_positive`を読む。現行mainのWails非依存`studiohost.NewReadOnly`を使う三入口Go試験を再検収する（[PR #5119](https://github.com/hat47x/tei-platform/pull/5119)自体は未マージでクローズ） | 型付きRule／元byte位置／Source状態／AuthorityLimitsと失敗分類を一致させる。同名の非公開Capability、存在しないRule、非許可Workspaceを拒否。読み取れたことは実行Grantではない |
| **U-1 controlled change** | テスト用WorkspaceのRule説明文だけ、次に参照Attributeだけを一因子ずつ変更し、通常CLI inspect／生成器を再実行 | 説明変更ではSource Indexを取り直し判断は保持。属性変更では型・Read先・Go・独立Expectedの関係を再判定。古いStudio選択Event／別Workspace epochから編集対象を復活させない |
| **U-2 実Buildと実受領** | 現行の`tei self-build`／BuildIntent／実Builder／別プロセスBinaryを使い、U-1で選択した意味の出所と生成byteを結ぶ | Source位置だけの一致、Go単体実行、手書きDriverの成功ではR2-P／R3に昇格しない。Plan解決後の差替えはmain済み#5109の反証を再利用 |
| **U-3 Desktop自己再入** | 生成Workbench／Inspectorを含むStage1実TEI→Stage2 DesktopのBuilderとWindows WebView2実Consumer | N2実CLI起動・生成部品のWindows受領と、Stage1自身がStage2を生んだ因果を分ける。#2831／#4583の既存Desktop P0で独立検収 |

**次のUI実装を増やさない判断**：U-0は現行mainの三入口Goテストと重複実装しない。Studioの選択EventはIntentでありAuthorityではない。Graph／Node Viewは正規Data／Relation／Processの投影であって、同名Nodeや多重Edgeを単純化しても元SourceやMappingを編集しない。PluginのWhole／Field／Adapter接続も、型・粒度・多重度・時点・開示・部分取得の全経路適格性が確かめられるまで、図で配線されたことを実Call成功に数えない。

**優先度**：D-1の最小の実判定と、U-0のread-only三入口一致は**独立した並行Packet**。U-1からU-2への橋渡しは、既存のSource Index・生成Go・Builderを再利用して検証する。CLIにWails依存を持ち込まず、Wails ServerをProviderとして扱わない。API Pluginを公開する場合も共通Operationに対する認証済みAdapterの追加として扱い、実Host・現在Grant／Disclosureは別Ownerとする。

## 10. マージと次の着手判断

この計画自体のCI・文書リンク・現行mainとの競合が解消した時点でdocs-only PRを統合する。D-1実装はそのPRに混ぜず、**現行Reader／Operator／MappingにM1条件を厳密に表現・実Callできるか**の判定を先に提出する。実現できる条件が一つあれば、その一条件に絞ったGo実装PRを作る。できなければ不足した意味能力だけを既存Ownerに返し、`required`を`nonempty`や`forbidden`へ自動的に読み替える修正はしない。

計画の統合は製品実装の検収ではない。現行の通常CLI→実Provider Read #2586とStudio Stage1→Stage2 Windows #2831／#4583をP0に維持し、ドッグフーディングの便利な画面表示や生成Goのbyte一致だけでP0を完了にしない。

## 11. 11月16日の製品実装を次の判断移管へ反映する

前回の計画作成後、現行mainで次の実装を確認した。PRの統合履歴と、実際にmainにあるファイルは区別して読む。この追記は**設計・実装引渡し**であり、ここに列挙したGoテストを今回新たに実行してはいない。

| 実装済みの事実 | 今回のdogfoodingで再利用するもの | 証明していないこと |
| --- | --- | --- |
| [#5142](https://github.com/hat47x/tei-platform/pull/5142)：`validationeval.Decide`の状態付き評価 | `present / null / not-fetched`と関係の`complete / partial`、理由付き`Undecided`。読んでいない部分で結果が変わりうる場合に成立・不成立を返さない | 業務レコードの**値を評価する**機構であり、`semantic.Validation.ID != ""`など宣言の**Shapeを検査する**機構ではない。空文字とNULLは同義でない |
| [#5146](https://github.com/hat47x/tei-platform/pull/5146)：`not-after`／`if present` | 二つの暦日の比較、相手がNULLのときの条件付き成立。Sourceの型・状態・範囲を記録する習慣をRuleへ広げる | `if present`はpositiveのRelation禁止を意味しない。複数行の重複検査や生成Go、実DB Date Providerへの対応は別 |
| [#5132](https://github.com/hat47x/tei-platform/pull/5132)：Rule意味IDとTarget Go関数名の分離 | 関数名変更でも意味を保つ正例、結果が同じでも別Attributeを読んでいたら拒否する実Binaryの負例 | 手書きDriverでの比較は製品実現選択／Actual Call Receipt／M2の手書き分岐置換ではない |
| [#5129](https://github.com/hat47x/tei-platform/pull/5129)：元Markdownの時間差改変拒否 | BuildPlan確定後、生成Goが不変でも同梱原文を改変した旧Planで通常`tei self-build`を拒否。#5109の生成Go改変拒否と独立に使う | Generator版・Source Locator・実Builder・Stage2実使用の正式R2-P全閉包ではない |
| 現行mainの`tei_md_self_description_test.go`と`cli_inspect_rule_markdown_process_test.go` | Data→Reader→Source位置の限定試験と、実CLI／直接Operation／read-only Studio Hostの三入口比較 | 当初の[#5097](https://github.com/hat47x/tei-platform/pull/5097)／[#5119](https://github.com/hat47x/tei-platform/pull/5119)がclosed/unmergedでも同種のコードがmainに存在する。ただし**コード存在≠最新HEAD試験PASS**、Inspect成功≠実行Grant |

[#5148](https://github.com/hat47x/tei-platform/pull/5148)の関係期間重複Ruleは確認時点で未マージ。採用済みの言語能力・製品受入に加えない。追加Ruleの数や、Value Stateを扱えることを「TEIが自分自身の検査判断を移管した条件数」に数えない。

## 12. D-1の実装Gateと、P/G／Studioの分離した検証

**最初のPacketはSH-1〜3の一つを正確に実行することだけに限定する。** 現在の`validationgen.GeneratePositive`では、`ID == "" || Attribute == "" || Relation != ""`が手書きGoに残る。Dataの`required`は項目の存在であって`!= ""`と同一ではない。新しい`Decide`が区別するNULLや未取得も、Rule宣言の空文字・禁止条件を代用しない。

| 検査義務 | M1で実際に必要な接続 | 独立反証 |
| --- | --- | --- |
| SH-1：ID非空 | TEI定義→`semantic.Validation.ID`の型付きField Binding→厳密な空文字判定→製品検査の実使用 | `ID=""`を拒否。文法段階で生成できない原Markdownの負例と型付きインスタンスの負例を混同しない |
| SH-2：Attribute非空 | TEI定義→`semantic.Validation.Attribute`のBinding→空文字判定→生成器が実使用 | `Attribute=""`の拒否。`std.decimal`の型解決だけでRead対象の存在を保証しない |
| SH-3：Relation禁止 | `Kind=positive`の適用条件→`Relation`のBinding→存在時に拒否 | 有効なRelation IDでもpositiveでは拒否。optionalをforbiddenとみなさない |

この三つを現行標準Data／Rule／Operator／Mappingで表現し実行できない場合、`M1=unsupported`として不足する接続（Field Binding・Predicate・Kind適用・拒否理由）を#2620／#988／#1121へ返す。単にGoの`if`を別関数へ移すこと、新しい専用Meta Runtimeを作ること、Expectedを生成器から逆算することはしない。独立Expected #1180／#2514を保持する。

**P/Gの同値範囲**も更新する。生成Goの`EvaluateSelected`は従来の`validationeval.Source`から数値を読み、NULL・未取得ではResultではなくエラーにする。一方`Decide`は状態付きSourceで理由付き`Undecided`を返す。正値・ゼロ・負値・Read ID・回数の共通入力を保持し、NULL／未取得は**呼出入口ごとの期待Result・Error・Undecided**を別票で比較する。両者がすべての状態で無条件に等しいという主張はしない。状態付きGo生成の導入は別Ownerの判断とする。

**Studio側のU-0**は現行mainの三入口試験を起点とし、次に一つのテスト用Ruleで説明文変更→参照先変更→再Inspect→生成Go→実Builderを同一Subjectでつなぐ。CLI／Wails／HTTPに意味の正本を複製せず、選択Intentや表示済みSourceから編集・実行・Disclosureの権限を作らない。Source改版後の古い選択を拒否する条件も明記する。

**並行P0とマージGate**：#2586の通常CLI実Provider Read、#2831／#4583のStage1→Stage2 Windowsは独立して継続する。確認時点のmain `df76ca9fadc6`ではCache Verification、Windows WebView2の一部が成功した一方、`reference-self-build`・Wails Hostの一部Jobが失敗し、Repository Healthは実行中だった。異なるJobの合格を全体Healthに読み替えない。D-2／M2はM1の独立反証後、D-3は元Source・Generator・Builderの同一Attempt照合後、D-4はStage2が実際に新しい検査を用いた後に認定する。

## 13. 異質なRuleで実現方式の選択をdogfoodingする

今後のM1は、すべてをGo生成へ揃えることではない。既存の`validationeval`による解釈（P）と`validationgen.GeneratePositive`による限定的なGo生成（G）が併存していることを、**利用者にとって予測可能な実現選択**として示す必要がある。

現行mainの`internal/operations/inspect_markdown_data_table_date_rule_realization_test.go`には、項目定義表の日時二項目を暦日に変更して`not-after`を選択する実試験がある。これは`Workspace.Load`→Validator→Source Index→Inspector→`Decide`の局所経路で意味を受理し、**対応していないGの要求は拒否する**というP/G境界の実装例である。別のTarget生成器、一般的なRuntime方式Resolver、実Provider経路の完成ではない。

| 実現方式の要求 | 現時点の対象 | このPacketで求める結果 |
| --- | --- | --- |
| **Pを明示** | 型付き`positive`または暦日`not-after` | 対象Rule、選択されたReader／Operator、必要Read、`Decision`と不能理由が元Sourceへ結び付く |
| **Gを明示** | `positive`＋解決済み`std.decimal` | 元Source・型・生成GoのReader、Target契約、実Builderが消費するbyteを区別して検査する |
| **未対応Gを要求** | `not-after`または未対応Relation Rule | `GeneratePositive`が未対応Kindを拒否する。返されたエラーをPへの無言のフォールバックと扱わない |
| **方式を変えて再試行** | 同じ意味SubjectのP→G、G→P | 結果が一致しても実装選択の変更を新しいAttemptとして扱い、以前のArtifact／実Call資格を転用しない |

**M1へ適用する順序**：まずSH-1〜3のうちTEIの定義で実判定できる一条件を確立し、その後に「その条件をどの方式で実現するか」を選ぶ。定義のShapeを厳密に検査できることは方式選択より前の条件であり、未対応Gの拒否試験だけでM1にはならない。

**受入の停止線**：`Pの評価ができた`、`Gのコードが生成された`、`実BuilderがGを消費した`、`製品Callerが選択した方式で実行した`、`Stage2が当該実装を使った`を、それぞれ別の証拠として残す。既存のSource・World／Model・Artifact／Build Attempt・現在Grantを一つの真偽値にまとめない。Graph／Node表示やPlugin Whole／Field／Adapter接続はこの証拠の投影であり、描画や接続可能表示だけで実行権限を与えない。

**次の小さな反証**：`not-after`を同一SourceからPで実評価できる正例と、同じSubjectのG要求を拒否する負例を、既存の製品LoaderとInspectorで比較する。これが成功しても、Studio Stage1→Stage2や通常CLI→Provider Readを完了と数えない。

## 14. latest-main CIを実失敗の単位で修復する（2026-11-16）

main `dc3f310ed52e652efbda2c8736a2f45d6f4b8a6a` の実GitHub Actionsログを確認した。Cache Verification／実Windows WebView2の一部が成功しても、**Repository Health・参照セルフビルド・Wails Hostは失敗**している。これを「CI不調」の一語で済ませず、修正PRと反証を一つずつ切り出す。以下はこのHEADの観測であり、後続HEADへ無条件に引き継がない。

| 失敗した検収とログの事実 | 直近の修正対象 | 独立に確認する終了条件 |
| --- | --- | --- |
| [Repository Health](https://github.com/hat47x/tei-platform/actions/runs/43866732336)：`TestTEIValidationDeclarationSelfDescriptionIsOnlyData`で期待した項目0=`rule_id`に対し`attribute_id`が返る | **自己記述の試験Owner**。項目の正規IDと型・必須性・原Source位置を照合し、Canonicalの並び順をSource表の行順だと誤認していないか確認 | 全四項目がIDごとに一致し、重複・欠落・余剰項目を拒否。表の説明データを実Ruleへ昇格させない。別の失敗までGreen扱いしない |
| 同Healthで`TestDiagnosticCodeCoverageOnlyImproves`が`NO_OVERLAP`関連の未被覆診断コード**4件**を検出 | 新しいRule種別の診断Owner／[未統合PR #5148](https://github.com/hat47x/tei-platform/pull/5148) | コードごとに**実際に発行する不正入力**の負例を追加。期待文字列を空で登録するだけで被覆と扱わない |
| 同Healthと[生成Host](https://github.com/hat47x/tei-platform/actions/runs/43866732297)でREA Local Readerが`OPERATION_FAILED: external Evidence is unavailable` | REA Reader／Source Locator／Studio Hostの既存Owner | 正常な独立Workspace、実別Process、許可されないWorkspace、Source差替えの正負を同一Reader版・Evidenceで再試験。Stubで成功を偽装しない |
| [生成Host](https://github.com/hat47x/tei-platform/actions/runs/43866732297)で`TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader`が失敗（埋込`main.js`に生成Inspector Readerがない） | Stage1生成物→TS/esbuild→Go embed→実Desktopの資産Owner | Source contract digest、生成Reader、埋込byte、Windows実Consumerの一致を別票にし、手書きReaderや古いBundleで代替しない |
| [参照セルフビルド](https://github.com/hat47x/tei-platform/actions/runs/43866732292)でRule由来の通常`tei` Build Intentステップがexit 1 | 自己構築／Build Owner。単一の失敗ステップと元入力を保存してから原因を切り分ける | 正常Build／元Markdown変更／生成Go変更の三経路を**そのHEAD**の実Builder入力で照合。ログのexitだけからSource不一致など原因を推測しない |

併せてHealthには`gofmt -l . is empty`の失敗や、Web API scoped Driver、Reference CRUDの特徴付け試験の失敗も観測された。これらは別Ownerの修正案件とし、M1実験のGoコード変更へ混ぜない。

**統合Gate**：`repository-health`と`cache-verification`が対象PRのexact HEADで成功し、最新mainのHealthと実CIの失敗が解消した後に、この計画ブランチをマージする。docs-onlyにしても既存Healthの失敗を免除しない。実装修正は各Owned failureで独立PR・独立Expectedを保持し、失敗試験を無条件に無効化しない。

## 15. MVPをStudioの一回の自己再入へ収束させる（2026-11-17）

優先順位の正本は[いま進めること](../NOW.md)に置く。全体の第一目標は受注一件の**実PostgreSQL→Rule変更→生成Web画面**であり、Studioの自己再入はそこに認められた**唯一の並行作業**である。Studio側ではショーケースや研究文書を追加せず、**Studioで定義の必須項目を一つ増やし、生成器の実装に手を入れず、Studio自身が作った次のStudioのWindows画面へその変更を出す**一巡に集中する。§1〜14のM1〜M3は否定しないが、今回のStudio P0の直列前提にはしない。

前回の[PR #5215](https://github.com/hat47x/tei-platform/pull/5215)は**closed／unmerged**であり、そこに記載した到達点をmainの実装と扱わない。これに対しmainには、[PR #5188](https://github.com/hat47x/tei-platform/pull/5188)による実CLI／Studio Hostの共通Inspect、[PR #5303](https://github.com/hat47x/tei-platform/pull/5303)による選択Subjectと検証済みInspector表示の一体化、[PR #5306](https://github.com/hat47x/tei-platform/pull/5306)による**候補**Workbenchの一時生成と型検査が取り込まれている。候補がビルドできることと、製品の`main.ts`／Wails埋込Assetに実採用されたことは別である。

また、main `8d02d2070fe7a4d6f696881bb8a186f00151b4af`では`reference-self-build`、Cache Verification、Windows WebView2の独立Jobが成功した。ただし、**参照セルフビルドの成功だけではStudio Stage1→Stage2の実再入は成立しない**。当該HEADのRepository Healthは確認時点で実行中であり、失敗した歴史的HEADと同じ状態だと決めつけない。

### 15.1 一巡で変えるもの、変えないもの

一巡の対象は、既存Inspectorの**応答定義に必須項目を一つ加える変更**とする。変更前と変更後の定義Source・生成Reader・画面の差を固定し、元のInspectorが選んだ正規Subject、Source fingerprint・byte範囲、Authority LimitsをStage1／Stage2で追う。増やした必須項目は、生成器を改修せず新定義から生成され、Stage2画面で読めなければ不合格。三要素分析の図面や型付きインスタンスの表示は別の表現であり、今回の改修対象にしない。

| 境界 | この一巡で必要な実物 | 今回は増やさないもの |
| --- | --- | --- |
| **D0：定義の変更と独立Expected** | StudioでInspector応答の必須項目を一つ増やす前後の定義byteと期待する表示差分。正規Subject／Source Revision、同名異ID・別Workspace・後着応答の負例 | 生成器の改修、別の業務DSL、生成器自身が出したExpected正本 |
| **D1：生成対象と入力** | Stage1が解決した非Plugin StudioのBuildIntent、選択したGeneratorとToolchain、正規Workbench／生成ReaderのSource Closure | Wails固有の業務意味、説明用の仮想Builder |
| **D2：実Builder** | Stage1の実Processが指定したBuilder／Invokerの選択と実Call。Stage1自身が起動した因果関係 | CI Shellが二回`go build`したことだけを「自己再入」と呼ぶ扱い |
| **D3：実Artifact** | Stage2の実ImageとBundle、元Source・生成TS／Go・正規esbuild・Go embedに対応するArtifact revision／byte | 候補Bundle、古い固定Closure、同じ見た目の別Artifactによる代替 |
| **D4：次世代の実起動** | **生成されたStage2 Binary**を実行したProcessと、それが採用した生成Inspector Reader | 参照Goセルフビルドの収束値からStage2 Desktopの実起動を推論すること |
| **D5：Windows実Consumer** | Stage2のWindows WebView2で同じSubjectのInspect／Pinを受け、Source・Scope・後着応答を確認する | 既存PoCのWindowやStage1の画面をStage2として再利用すること |

**D1〜D3を同一Attemptで結べない限り、D4〜D5を合格にしない。** World／Model／Artifact Revision、Source・生成器・Toolchainの版、どのProcessがどのBuilderを呼びどのbyteを読んだかを、ひと続きの受入票にする。A→B→Aの再現byteやGeneratorの自己申告だけで旧Evidenceを復活させない。[PR #5291](https://github.com/hat47x/tei-platform/pull/5291)の宣言・申告・外部観測の分離は**未統合の補助検証候補**であり、当面のStudio実再入をそのPRの統合待ちで停止させない。

### 15.2 最初のコード変更は生成Inspector Readerの正式収録

最初に閉じる断線は、[Web設計§21〜22](../design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)のWEB-P1a-2bである。**第一段は生成Readerを唯一のReaderとして製品へ復旧**する。現行mainの`readInspectorPresentation(raw, {workspaceId, selected, catalog})`と候補`main.inspector-candidate.ts`を確認し、歴史的`main.ts`の選択・Pin・応答laneを保ったまま、**契約Source→生成Reader→正規`main.ts`での実import・呼出し→esbuild→Wails埋込JS**へ接続する。**第二段は、必須項目を一つ足した定義だけを更新して同じ生成器を再実行**し、Stage2画面に差が現れることを確かめる。

実装PRの合格条件は、生成ReaderがBundle内に一つだけ存在し、現在の契約Digestに一致すること、手書きの代替Readerが並存しないこと、埋込JS／HTML／CSS／第三者通知が**同一HEADの正規Build**と一致すること。既存の`TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader`を緩めず、生成Desktopの実WebView2でもReader使用を観測する。過去のPinを新しいSourceと偽って更新せず、**新しいSource Closureを新しい版として検収**する。

Reader／Presenterの表示成功はHostの閲覧Grantではない。選択SubjectとCatalog不一致、別Workspace、古いSource Index、部分取得、明示NULL、Authority Limits欠落、後着の古い応答は、既存の独立Expectedと`responses.run("inspection")`の選択laneでそれぞれ拒否・保持を確かめる。現在のHostが認めない原文Openや編集保存を、Inspectorの`textContent`表示だけで成功扱いしない。

### 15.3 M1とRuleの標準化を「Studioの一巡」のHard Gateにしない

[PR #5290](https://github.com/hat47x/tei-platform/pull/5290)の採用済み方針では、業務Ruleの正規形を**Operatorの署名と名前付き入力**で説明し、既存のValidation種別を際限なく増やさない。実装済み[PR #5187](https://github.com/hat47x/tei-platform/pull/5187)の`required-if`は**業務レコードのPresence**の検査であり、`GeneratePositive`が手書きで確認する`semantic.Validation.ID/Attribute != ""`や`Kind=positive`でのRelation指定禁止の自動置換ではない。

したがって自己記述M1の三条件は、現行`required`表のままでは**実判定へのBindingが不足**した状態として管理し、既存Operator・型付きField条件を優先して最小能力を別Packetで検討する。新しいValidationKindや第二Meta InterpreterをStudioのために追加しない。M1で一条件を**TEI定義が実判定**し、M2で対応するGo分岐が削除され実Callerが採用したときだけ、自己置換の進捗に計上する。

**直近の実装順**は、①生成Inspector Readerを唯一のReaderとして復旧、②**定義の必須項目を一つ追加し、生成器のコード差分がゼロのまま**生成・埋込byteを照合、③Stage1が実Buildを選択・起動したD1〜D3の同一Attemptを確認、④Stage2 Windows WebView2のInspectorで変更した項目を実表示（D4〜D5）。受注一件を実ProviderからWebへ届ける主線は[いま進めること](../NOW.md)の優先順位を維持し、Studioの成功で代替しない。

## 16. 11月17日：実CLIと正規Studio Readerを基線に、次の実証を選ぶ

**優先順位は[いま進めること](../NOW.md)が正本**であり、製品の第一目標は「受注の一変更」の五手順である。すでに実PostgreSQLで動く手順1〜4を前提に、最初は**異なるData・Relationを持つ別Workspaceでも同じCLIが動くこと（道の1）**、次に生成Webで同じ答えを見ること（道の2）、その後に全件性の宣言と検証（道の3）へ進む。§15の**Studioが次のStudioを作る一巡**は、NOWが認める唯一の並行作業であって、製品主線に優先しない。最新mainで統合された[#5300](https://github.com/hat47x/tei-platform/pull/5300)（明示Environmentで`tei view`から実PostgreSQL受注明細を読み、選択Ruleで判定）、[#5325](https://github.com/hat47x/tei-platform/pull/5325)（生成Inspector Readerを正規Workbench／Wails埋込へ接続）、[#5331](https://github.com/hat47x/tei-platform/pull/5331)（直後のHealth不具合修復）を、それぞれ**実CLIの局所受入、生成Readerの製品同梱、CI修復**として記録する。どれもStage2の実起動やWindows画面での自己変更受領を単独では証明しない。

| 最初の独立Packet | 現在使える基線 | 次の一因子と反証 | 完了を主張しない範囲 |
| --- | --- | --- | --- |
| **P1：異なる業務モデルでの実CLI** | [#5327](https://github.com/hat47x/tei-platform/pull/5327)の入口一般化はmain済み。固定名除去と**同じRelationの別名Pattern**の実PostgreSQL試験が動く | **異なるData・Relation**を持つ別Workspaceに同じ`tei view -environment`をコード変更なしで適用し、主キー・入力Binding・Environment選択・Rule結果を独立Expectedと照合。新JSONの`perRow`／`perRelation`／`rows[].key`は次のConsumerでも使う | 別名PatternのGreenや固定名除去だけで別の業務モデルへの適用を認定しない |
| **P2：生成Webの実受領** | [#5317](https://github.com/hat47x/tei-platform/pull/5317)の静的HTML候補はDraft、実PostgreSQLとHTMLの専用CIは成功 | P1の別モデル検収後、同じRule結果と行ID・未取得・NULL・`coverage:partial`を実Browserの別Consumerで照合。#5327の`perRow`／`perRelation`／`key`への名称変更に、#5317の旧`destinationRead`／`perLine`表示を合わせてから実行 | 静的HTML出力を生成Web Components・公開Host・完全な実Browser受入と呼ばない |
| **S1：定義から正規Bundleまで** | #5325の生成Inspector Reader／正規`main.ts`／Wails embed、§15 D0〜D3 | Inspector応答の**必須項目を一つだけ**TEI定義へ追加。Generatorソースを変えず、Stage1が選んだSource byte・Generator版・実Builder入力と、新しいReader／Bundle byteを一試行で照合。古いReader、手書きFallback、別Sourceの同名IDを負例にする | CIが生成できたことだけでStage1の実Build選択・Stage2の実使用を認めない |
| **S2：次世代の実受領** | §15 D4〜D5、既存のWebView2 Smoke／Inspector／Pin | S1のArtifactを**Stage2 Binaryとして起動**し、Windows WebView2で新必須項目を含む同じSubject／Revision／Scopeの表示を確認。古い応答、別Workspace、現在のGrant外Sourceを拒否 | Stage1の画面、既存PoC、参照セルフビルドの収束値をStage2と呼ばない |
| **R1：実受注明細の全件性** | #5300の`tei view -environment`、`order_entry.order_lines`、`Detail.Decide` | 受注1001／1002／1003の独立Expected（明細3／2／0件）を業務IDで照合。Query Scope／Binding／絞込み／ページ終端／Snapshotから**全件性の根拠**を示す。部分Readで見つかった違反は確定、見つからないことによる成立は保留 | 現行CLIの`coverage: partial`を無根拠に`complete`へ昇格させず、生成Webの実受領も未達とする |
| **R2：別Consumer** | #5300の実CLIと既存Rule／Source Index | 同じ選択Source／Rule／実Read IDと結果を生成Webの独立Consumerで照合。SQL列NULL・未取得・Decimal精度とAuthority／表示Scopeを分離 | Webの表示成功をProvider更新・公開Grant・業務Effectとしない |
| **M1：自己記述の実判断** | 自己記述Dataと`GeneratePositive`の手書きShape拒否試験、[#5187](https://github.com/hat47x/tei-platform/pull/5187)の`required-if` | `semantic.Validation`の型付きインスタンスを選択済み宣言へ結ぶ。ID／Attribute非空またはpositiveでRelation指定禁止の**一条件**をTEI定義から判定し、旧Goと独立Expectedで比較。対応Go分岐の削除は次のM2 | `required=yes`を非空文字、`required=no`を禁止、業務レコードの`required-if`をMeta Shapeと読み替えない |

**設計の共通境界**：三要素モデルのData／Function／Business、標準Graphの同名異IDと並行辺、型付きPluginのWhole／Field／Adapter、RuleのOperator署名は、同じ意味IDとSourceを別のProjectionで説明するために用いる。図面のEdgeやPlugin配線を、実Read、認可、業務Functionの実Effectへ自動昇格させない。Markdown原本への編集は現在のSource Span／Revisionを確かめた局所パッチとし、参照を壊す削除には診断と明示確認を要求する。

**着手順と停止条件**：製品の主線はNOWのP1→P2→全件性R1とし、唯一の並行作業はS1→S2とする。R2はP2のConsumer受入に含め、M1→M2は自己記述に残る設計課題として記録するのみで、製品五手順が通るまで新たな設計PRや別実装を起こさない。各Packetで実Source／実Builderまたは実Provider／実Consumer／独立Expectedを照合する。[#5324](https://github.com/hat47x/tei-platform/pull/5324)は並行統合で前提が古くなり競合したため未マージで閉じ、本節に最新mainで有効な判断だけを引き継いだ。
