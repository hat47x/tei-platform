# Script・Function起動の製品検証：最初の一機能と直接テスト

- **状態**：横断設計・実装引渡し。CapabilityのInput／Result／Effect宣言の構造検査はGo実装としてmain統合済み（#4629／#4636）。正式なDSL／CLI ABI／API Schemaは新たに採用していない。公開Entryから実Providerを呼ぶ製品E2E、実DB・外部Consumerの資格は別検収。
- **目的**：Scriptを保存・公開できること、実Providerを呼んだこと、業務効果が確認されたことを別々に検収する。そのための最初の製品実装を、既存Go単一バイナリと共通アプリケーションAPIに絞る。
- **実装と受入の正本**：[Script Launch #4502](https://github.com/hat47x/tei-platform/issues/4502)、[P0実装 #4495](https://github.com/hat47x/tei-platform/issues/4495)、[Trigger #4497](https://github.com/hat47x/tei-platform/issues/4497)、[Script/Macro製品 #2502](https://github.com/hat47x/tei-platform/issues/2502)。本書は横断した実装順と証拠の読み方を示す。独立Expected・Scenario・試験実行器の新しい正本は作らない。
- **既存判断**：[共通アプリケーションAPI ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、[Target固有成果物 ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)、[認証非内蔵 ADR-0044](../adr/0044-authentication-not-a-core-concern.md)。

## 1. 利用者ができるようになること

開発者は、既存のFunctionをCLI/CUIで調べ、公開されたEntryから型付き引数で呼び出し、選択した実Providerが起動したことと結果を確認できる。StudioやAIハーネスからも、後に同じ操作を呼べるようにする。内部のData Ruleや未公開Scriptが、自動的に公開Entryになることはない。

標準の業務設計では、**三要素分析法のデータモデル・機能モデル・業務モデル**の対応を尊重する。ただし三要素分析法をTEI Coreに固定せず、非業務用途や通常のコーディングも可能にする。Role、DSL、表示図、UIの個々の名称を実行権限の根拠にしない。

### 代表例（既存の有限Fixtureを再利用）

`quantity=2`と`unitPrice=300`から`600`を求める副作用のないFunctionを一つ使う。これは**例示する入力とExpected**であり、現行mainにこのFunctionが公開Entryとして実装されているという主張ではない。着手時に #2502 と #1130/#4410 から実在するSource・意味ID・テストFixtureを照合し、対応がなければ最初の未実装点を記録する。

同じ受理済みFunction/入力/Targetに対し、①型付きアプリケーションAPI直接呼出し、②製品GoバイナリのCLI実プロセスを実行する。両者で戻り値が`600`でも、片方がReferenceExecutor、もう片方が別のProviderなら**製品起動の入口一致は不合格**。CLIのJSON外殻や時刻ではなく、復号後の意味値、版、実呼出先、結果の検証対象を一致させる。

## 2. 最初のPRと判断材料

初回PRは[#4495のPR-A](https://github.com/hat47x/tei-platform/issues/4495)をそのまま実装対象とする。新しい`run`構文、スケジューラー、Script Runner、全GUI、EKI、実DB書込みを含めない。

| 順序 | 実装・試験で確認すること | 不合格となる最小反例 |
| --- | --- | --- |
| P0-0 | 現行mainのSource/Function/Entry、typed Operation、Provider選択/Load/Invoke、実CLI dispatchの**最初の欠落箇所**を確定 | Referenceの二入口を「製品Providerの二入口」と数える |
| P0-1 | FunctionとEntry公開を区別し、非公開EntryはProviderを一度も呼ばず拒否する | 同名Functionの存在だけで実行できる（EL-01） |
| P0-2 | actual selected・loaded・called ProviderおよびArtifactを実Attemptと結ぶ | Provider B選択に対しAを呼び、同じ600を返して合格とする（EL-04） |
| P0-3 | 直接APIと実CLIプロセスが同じ型付きOperationへ至る | APIは実Provider、CLIはMemoryUnitOfWorkの参照Executor（EL-08） |
| P0-4 | 実結果本文・診断・Failure／unknownを照合する | exit0/transport ACKのみ、実Result bodyが未取得（EL-12） |
| P0-5 | 入力のmissing、明示NULL、未取得を区別する | 未表現の状態を0へ無言で変換する（EL-05）。既存型で不能なら実装予定のGapとして残す |

PR-Aの必須の最小集合は**P0-1〜4**。PRの合格記録には実Main/PR HEAD、Fixture Sourceと独立Expected、実製品バイナリ、CLI argv/exit/結果byte、実Provider Call Site、異なる入口からの実呼出しと拒否点を示す。CIが成功しても、要求テストが0件一致・skip・MockのみならProduct Q2の合格にはしない。複数TargetやUI完成で初回PRを待たせない。

## 3. 直接テストと横断シナリオを混同しない

`*.tei.md`内の図表、DSL、SQLなど、対象要素の近傍にOKF等で管理する安定した識別子を置き、直接テストを対応付けることを検討する。この考え方は**開発時の診断・変更影響・AIによるテスト生成**に有用だが、OKF IDの存在だけではSourceのロード、Meaningの受理、実行を意味しない。

| 検査階層 | 主な問い | 既存の担当・保存先 | 成立する合格範囲 |
| --- | --- | --- | --- |
| **要素の直接テスト** | 型、Rule、式、Operator、Query、Entry宣言の単独入力・出力・拒否は適切か | #4359/#988/#2620、既存Validation/Scenario・#2514 | その要素と固定Required Setの局所検証。実DB/Provider起動は未証明 |
| **同一Function内の結合テスト** | 入力→計算/Query→Provider→Resultで同じ意味と実ロード物を使ったか | #4495（pure起動）、#2586（実Read）、#1163（同一Attempt） | 選択した実Providerへ到達した範囲まで。対象集合・権限・Commitは別資格 |
| **複数Function/Hookのライフサイクルテスト** | Business Task→Function→Data Rule→Effect→通知と戻りを、順序・取消・重複込みで確認したか | [Driver Scenario #1180](https://github.com/hat47x/tei-platform/issues/1180)、#1129/#1174/#1130/#2514 | 実Scenarioに含めた分岐・Effect/Receiptの観測範囲。独立Expectedと実Provider Evidenceが必要 |
| **Target/Consumer検収** | CLI/Web/Studio/SHO/外部Harnessで同じContractを実現し、受け手が再解決できたか | ADR-0048、#2509/#2515、#3422、#1163 | 実行した入口・受領先だけ。外部ACKや同じ出力から全Target同値を推定しない |

直接テストを編集画面の近傍に表示できても、**Driver Scenarioの第二正本を作らない**。複数要素を横断するCaseは、既存ScenarioにID/Source/Required Rule/分岐/Expectedへの参照を持たせて再利用する。合成Expectedをテスト対象のDSL evaluatorや生成AIの同一出力から作り、正解として自己採点しない。単純なケースは独立に手で固定した値、業務上の判断を要するケースは既存Ownerの通常のレビューを使う。新しい承認組織や形式的な儀式を設けない。

変更前後のカバレッジは、**設計要素／分岐／状態遷移／外部作用**を別欄にする。既知の依存グラフに対象が出ないだけで「影響なし」と判定しない。未探索・未読込・非開示の依存がある場合は、用途に応じて`unknown/partial`を残す。

### 最小の横断負例

- **T-L01** 同じSource byte spanに、改版後は別Ruleが存在する。OKF/Anchor位置だけで旧テストや旧Pinned Selectionを別Meaningへ付け替えない。
- **T-L02** `quantity`をTable定義から削除したが画面とScriptの対応欄が残る。Mapping欠落を示し、同名の別列へ自動接続せず、実DDLが変更済みとも言わない（#4502 MC-L2）。
- **T-L03** Scriptと決定表は同じ600を返すが、表側が不要な秘密列を先に読んでしまう。Value合格でもRead Footprint/Disclosureは不合格（#988 DTQ）。
- **T-L04** Functionはpure、CLIの`-out`または`-operation-log`がファイルを生成する。Function計算とHostのファイル作用を分離して判定（#2605/#2504）。
- **T-L05** Triggerは予定時刻に発火したが必要Partitionが未読込。業務期の集計完了と判定しない。同じRunの再開、差分実行、遅着・訂正後の再確定を分ける（#4497/#1130）。
- **T-L06** 既にDB Commitしたが取消要求・返信断でACK不明。取消しを未実行やrollbackに読み替えず、同一Operation IDのReceipt照会を要求（#1174）。

上記は新たな独立Fixture正本や実装済みTest名ではない。既存EL・TR・Driver Scenario・モデル直接テストの判別観点を示した**受入候補**である。

## 4. 起動が何を変えたかは別々に見る

| 資格 | 実行する主体・必要な根拠 |
| --- | --- |
| **Q0** 定義・公開 | Function/Entry/Target/Source/意味版、公開面 |
| **Q1** Reference | 合成Runner/MemoryUnitOfWorkの計算・拒否。永続性を推定しない |
| **Q2** 実製品pure起動 | Go実バイナリ、actual Provider call、CLI/API二入口、実出力 |
| **Q3** 実Owner読取り | SQL byte/bind、実Rowset、取得範囲と開示 |
| **Q4** 保護されたEffect | 現在Grant、同一Operationの実Commit/Receiptと独立Observation |
| **Q5** 外部受領・業務結果 | 相手先の受信確認と、必要な業務対象の完全性 |
| **QX** Target適合 | 必要な入力、時点、Artifact、EffectをそのTargetへ表現できる証拠 |

Q0〜Q5は**一律の逐次実装段階ではない**。pure FunctionをQ2まで実証するために実DB書込みや外部送達を要求しない。現在の`ReferenceExecutor`での`Committed=true`、`Persisted=false`、`Durability=none`は矛盾ではなく、永続化・実Host Grantを証明しない。

Script保存→Entry公開→Trigger登録→Preview→実行→履歴という利用者体験も、**公開停止／Trigger停止／実AttemptのCancel／Grant失効／Commit済みEffect／現在Disclosure**を分ける。停止ボタンが押されたという事実を、起きた副作用の取戻しと表現しない。

## 5. 起動経路をたどる参照索引と、実行前の対象確認

他チャットのStudio/Modeler設計（[#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#4515](https://github.com/hat47x/tei-platform/issues/4515)）で重視する、**項目レベルの参照・被参照一覧、未使用候補、キーボード中心の高速移動、AIへの対象指定**をScript起動にも適用する。ただし、参照インデックスは**既存のSource/Meaning/Entry/Trigger/Runから導く読取り用の索引**であり、新しい公開Registry、権限エンジン、Runtime Dispatcherではない。索引に載ることは、現在の公開・認可・実呼出しを保証しない。

### どの参照を数え、どこへ移動するか

| 参照の種類 | ナビゲーションで示すもの | 混同しない実行上の事実 |
| --- | --- | --- |
| **定義上の直接参照** | Field/Rule/Function/Script/Entry/Triggerの参照元・参照先とSource位置。削除・改版で未解決となった辺 | Source文字列上の一致、受理済み意味、実起動は別 |
| **導出された間接参照** | Function→公開Entry→Trigger Binding、Function→Query/Relation/Hook、必要なら複数段の到達経路 | 宣言上の到達性と、実際に選択・読取・呼出した経路は別 |
| **実行時の観測** | Attemptのselected/loaded/called Provider、実Read、Effect/Receiptの照合可能な範囲 | Reference結果、Planの「resolved」、過去のRunだけで現在の有効性を示さない |

参照先は**Workspace/SubjectのKind＋ID、Source/Revision、出現位置・Role、必要ならRelationの辺ID、対象Scope**で解決する。同じ表示名や同Kind同IDが別Workspace/Sourceに存在しても先勝ち・後勝ちで結ばない。三要素Modelerの**同一モデル内・同Kind同ID重複の診断と曖昧IDの索引除外は[PR #4524](https://github.com/hat47x/tei-platform/pull/4524)でmainへ統合済み**。ただしそれだけではWorkspace越境・外部Source横断の一意性、公開Entryの解決、GUIのPinや現在Grantを証明しない。正式なRelation occurrence IDが未提供の場合は該当辺への正確なPin/ジャンプを`unsupported/ambiguous`として保留し、画面描画用の一時IDを正式IDに昇格させない。

**参照数0は「調査した範囲内で既知参照が0件」**である。インデックスの対象Source/選択Plugin/Role/動的参照/外部Consumerの探索が未完了なら、`0 (partial)`または`unknown`と示し、「絶対に未使用」や「削除して安全」と断定しない。閲覧権限外の参照を件数・名称・エラーメッセージから漏らさない。未読込Pluginを全件起動して索引を構築する設計も採らない（[#1251](https://github.com/hat47x/tei-platform/issues/1251)）。

### Focus・固定対象・AI文脈・実行権限を分ける

CLI/GUIで選択した対象を**固定した作業対象（Pinned Work Set）**として保持し、通常のフォーカス移動・検索・IME・表示切替で意図せず解除しない。対象の参照可能性と現在版は、表示継続の可否とは別に再検証する。固定対象がstale/ambiguous/非開示になった場合は**表示上の固定を維持しつつ、AI共有・変更適用・起動をそれぞれ保留**する。AIハーネスに渡すのは閲覧許可された範囲のQualified Target Refと文脈のProjectionであり、Host Grantや「実行してよい」という認可を内包させない。起動はあくまで#4495の公開Entry/現在のHost権限で行う。

モデル編集でFieldが消え、UI項目・ScriptのInput Mapping・Triggerの旧参照が残った場合、**参照切れを可視化し、同名Field/Functionへの自動再接続はしない**。A→B→AのSource復帰、Source改版、権限変更、Plugin選択変更、部分取得、非同期で遅れた索引結果は、それぞれ現在資格を再判定する。定義/出現位置のジャンプ、参照元へ戻る操作、未解決への移動、現在Pinned対象への復帰を、マウス中心の操作を必須にせず提供する。ただし個々のショートカットキー割当や新CLI構文は実測前に固定しない。

### 受入と実装順（既存の独立Expectedを使用）

最初の**Q2純粋Function・実CLI二入口PRのHard Gateにはしない**。次段で#1124/#4515のread-only参照・固定選択と#4495/#4502のEntry/Trigger対象を接続し、次の対照を既存#1180/#2514のScenario/Expectedに追加する。

- **ゼロ件と未知**：実際に参照0の公開範囲、未読込Sourceがある範囲、現在Disclosureで隠された参照を分け、「未使用」と断言できる範囲を示す。
- **一意性と改版**：同名異ID、同Kind同IDの重複、同端点間の複数Edge、Source位置変化、削除→再作成、A→B→Aで旧Pinned対象/Entry/Triggerを勝手に再束縛しない。
- **操作と実行の分離**：Focusを移してもPinは残る一方、Grant/Entry/Artifactが失効したら起動拒否。AI共有は現在Disclosureで再投影し、受信したHarnessが権限を獲得したと見なさない。
- **索引と実Callの分離**：静的参照が1件あってもactual Provider未呼出ならQ2合格0件。Provider B選択・A呼出なら参照索引の健全性に関係なくEL-04不合格。
- **性能と拡張**：定義数・参照辺数・Plugin数を変えた固定Fixtureで、索引初回構築、増分更新、参照ジャンプ、Pin復元の待ち時間とメモリを計測。選択されないPluginの初期化・追加により実起動の経路や体感応答が悪化していないか、[#1251](https://github.com/hat47x/tei-platform/issues/1251)の実Provider非干渉と別に検査する。数値目標は計測結果と製品対象規模を見てOwnerが定める。

索引の**直接参照と実行経路の観測情報**は、Studio/CLI/AIに同じ意味で提供する。ただしこれを理由に新しい共通Run StoreやGraph Engineを設けず、既存OwnerのSource/Dependency/Runtime Receiptへ照会する。個々のIndex結果はSource/Plugin/Target版と探索範囲を明示し、`unknown`を0へ変換しない。

## 6. 次に進める順番

1. **いま着手**：#4495のPR-A。現mainの第一不足辺を実コードで特定し、既存Fixtureを用いてQ2 Productのpure一件を成立させる。#2586の実Read、#2514/#1180の検証基準とは並行。
2. **並行準備**：#4359/#988/#2620で直接テストの対象とSource位置/Meaning/Operator Pinを保ち、T-L01/03を局所試験へつなぐ。全Language/新Syntaxが動くまでP0を待たない。
3. **次段**：#4502/#1124でモデル変更時のdangling依存をHeadlessに観測し、#4497/#1130でTrigger/Input Closureと取消・再計画を検査。Schema変更・現DB効果は各Ownerで独立検収。
4. **実Providerが観測できてから**：Studio・SHO・外部Harnessへ同じ型付きOperation/Contextを送る。受信者が現Source/Subject/Scope/Disclosureを再確認した証拠と、Effect/Consumer受領を分ける。

**完了報告の最小形**：対象Issue/固定commit、対象Function・Entry・Source/Meaning/Artifact、実Consumer/Provider、元から独立したExpected、試験Caseと観測値、実行コマンド、PASS/FAIL/NOT RUN/UNSUPPORTED、Claimの上限、残る最初の不足点。CIの成功本数や追加したコメント・テスト数だけで製品完成率を表さない。


## 7. 現行mainの受入可能な足場と、P0でなお実証すべき最初の境界

### 7.1 今回確認した実コードの資格

| 現行mainに存在する証拠 | 実際に確認できる範囲 | #4495のP0-Lへ流用できないもの |
| --- | --- | --- |
| [#4524](https://github.com/hat47x/tei-platform/pull/4524)、`internal/threeelement/model.go`（blob `70a663e26efb3879d98a68ca76d44b4e669e2d7d`） | 単一の三要素モデルにあるData/Function/Business各Kindの重複IDを診断し、曖昧なIDを`Index()`から除外する。無関係な一意IDは参照できる | Workspace/Artifact/Entry/Targetまでの全域Identityと、Hostの実行認可 |
| [#4517](https://github.com/hat47x/tei-platform/pull/4517)、`tools/reference-b2c-change-inspector/source_bridge_test.go`（blob `3b14e58a1cfdbc566a3a3ccac7371064ce3bba45`） | テスト内で既存`workspace.Load`→`BuildSourceIndexFromLoad`→`operations.InspectionFromLoad`を使い、同じLoad時バイト列・Source occurrence・Workspace情報の限定的な一致と、変更/Root差の負例を確認する | 全Import/Sourceの読込閉包、原子的全ファイルsnapshot、Entry/Meaning受理、実Provider Invocation |
| `cmd/tei/cli_view.go`（blob `45538bbd185468652df5045adf11b317655b1948`）と`internal/studiohost/command.go`（blob `562312cedd0b89c2dbc7588268531e05bc84aa31`） | 現行の一部Command入口は`ReferenceExecutor`/`ReferenceDataExecutor`と`MemoryUnitOfWork`を使い、`Committed`と`Persisted`を分けている | 実公開Entry→実ロード/呼出Provider、DB永続化、実Host Effect、CLI/API両入口の製品Q2合格 |

上の「統合済み」は **mainの個別成果** を指す。**#4525がmain最新HEADへ追随したこと、同一checkoutで三者を結合実行したことは別**であり、この文書だけから主張しない。#4521のRelation Map/Locate、#4504のWorkspace誤再束縛、#4527の問い相対閉包は別Draftとして扱い、mainの成立済み機能へ無言で数えない。

### 7.2 「何が分かれば起動できるか」を用途相対に判定する

一つのFunctionを起動するとき、**見つかった定義・実際にLoaderが読んだSource・解釈した意味・現在使うEntry・Hostの入場資格**を別に照合する。必要条件を巨大な共通Envelopeや全Workspace走査の固定手順にしてはならない。必要な集合は「このFunctionを、このTargetで、この入力・作用級で呼ぶ」という具体的な問いで決める。

- **Discovery**：`Function`のCatalogに表示されること、同名の実Sourceがあること、参照数が1以上であることは、現在公開中のEntryがある証拠にならない。逆に静的参照数0でも、動的/外部Callがあれば「削除して安全」にはならない。
- **Source/Identity**：Workspace/Kind/ID/Source occurrenceを確かめ、重複・別Root・Source変更で曖昧なら対象を推測しない。重複IDの局所拒否で、無関係な一意Functionまで全件不能と断定しない。
- **Required Closure**：Function/Entryの実評価に必要なRule、Input、Operator、選択Provider、Artifactなどを**宣言集合・読込集合・必要集合**として比較する。未取得・解釈不能・非開示を「Requiredが0件」や「検証済み」としない。選択されていない無関係PluginをすべてLoadすることも要求しない。OKFは根拠を追跡しやすくするが、実Reader/Meaning/Host資格を発行しない。
- **Q0 Entry/Preflight**：明示公開のEntry/Contract版、Typed Input（missing/NULL/unfetchedを区別）、Targetの表現可能性、Source/Meaning/Artifactの期待版を照合。成功はあくまで実起動**候補が解決したこと**である。
- **Q2 Product**：Hostが現在のWorkspace/Principal/Scope/Provider/Grant/Resourceを検査し、API/実CLI二入口から同じ実Providerを呼び、Result bodyと失敗を観測。事前資格と実Callの間の版変更・取消にも再検査する。Q2のpure呼出しにQ3実DB取得やQ4書込みを一律に要求しない。
- **Review / Disclosure**：起動前にSource全体をAIへ転送したり、失敗時に非公開の対象ID・Path・Rule内容を出したりしない。Source・実Provider・実呼出しの内向きEvidenceと、外部に見せる診断・Resultを分ける。

### 7.3 第一PRへ追加する、判別可能な五つの拒否・限定成功条件

| 受入候補 | 単一因子を変える操作 | 独立Expectedと合格上限 |
| --- | --- | --- |
| **PL-R01** | Data/Function/Businessの同Kind同IDを二重定義、無関係な一意Functionは保持 | 曖昧な対象は#4524の範囲で索引から除外。一意な別対象は局所Inspect可能。ただしEntry非公開なら起動不可。Model Index成功だけでQ0/Q2合格としない |
| **PL-R02** | Function/Subject IDとSource本文が同一、Workspace Manifest/Rootだけ相違 | 別Workspaceとして再解決。AのPinned Target/Entry/GrantをBへ黙って移送しない（#4517/#4504） |
| **PL-R03** | `workspace.Load`の後にSourceをA→B→Aと改版し、旧Index/PreparedInvocationを持つ | 旧Readは履歴に残るが、旧実行資格の自動復活は認めない。現在の必要Source/Entry/Artifactを再資格化する |
| **PL-R04** | Required Ruleの一部Sourceは宣言済みだがReader未取得、または`*.tei.md`のLanguage/Reader未選択 | 未取得を「存在しない」「全Rule充足」と解釈しない。Previewの限定正例は可能でもEntryを必要条件未確認で実起動しない（#2823/#4359/#3262） |
| **PL-R05** | 同一Function/Entry/TargetのP0 pure呼出しについて、APIがProduct実Provider、CLIが参照Executorへ到達 | 同じ結果600でも入口の意味同値は不合格。双方の実Call Site・Artifact/Provider identity・Result/Failureを固定したときだけQ2合格（EL-04/08） |

**最初の実装PRへの昇格条件**：PL-R01〜04は現行mainのReader/Indexを**再利用できる部分だけ**初回に組み込む。すべてをP0-Q2の直列Hard Gateへ追加しない。PL-R05と既存EL-01/04/08/12が最優先であり、共通Typed Operationと実Providerの最初の不足辺を閉じる作業に集中する。独立Expected/Driver Scenarioの保存先は#2514/#1180、Host/Effectの実装は#1174/#1129、Pluginの実選択・起動は#1251のOwnerとする。

**検証状態**：main個別Sourceの静的照合・既存限定testの確認、Draft計画の変更のみ。PL-R01〜05を同じ製品Binary・同じWorkspace/Source/Target・CLI/API二入口で実行した証拠は **NOT RUN**。実行/認可/DB/外部受領やOwner受理は未証明。

## 8. Preview後に変わるSource・Provider・権限と、同一Attemptの判定

他チャットの「問いに相対的な構成・実行・証拠の閉包」（[Draft #4527](https://github.com/hat47x/tei-platform/pull/4527)）と「参照ナビゲーション→近傍直接テスト→製品Entry実起動→実Read/Effect」（[Draft #4478](https://github.com/hat47x/tei-platform/pull/4478)）を、#4495の**実行直前の判定点**へ接続する。両PRは設計上の別Ownerであり、Draft文書の存在をこの製品機能の実装・採用に数えない。

### 8.1 Previewの成功は、実起動時点の資格ではない

**Preview時点（P）**で一意に解決したFunction/Source/Entry/Provider/入力であっても、**Admission/Dispatch時点（D）**までにSource、選択Plugin、Artifact、Target、Grant、期限、対象Scopeが変わり得る。PとDの間を「同じ名前」「同じ最終fingerprint」「同じResult」で連続していると見なさない。

- **Prepare（既存のSource/Operation）**：Workspace/Source occurrence、Meaning/Contract/Entry版、Required集合（未読込・hidden・否定依存を含む）、typed Input、選択Provider/Artifact、Target/予定時点、予定Operation IDを確定できる範囲だけ固定する。足りない項目は`unknown/not-observed`とし、未確認の値を生成AIが補わない。
- **Admit（現在のHost/Provider Owner）**：現在のWorkspace/Target/Grant/Source/Artifact/Provider適合と期限・対象を再検査する。受理済みの言語/モデルと、Hostが現在許す資源・OS到達性は別。特に外部Principal/Scopeの文字列やStudioでのPinはGrantでない。
- **Invoke（実際のProvider）**：選択・収録・ロード・Callの一致を実Call Siteで観測する。入場可から始動済みへの推定は不可。途中差替えやTargetの表現不足は`stale/unsupported`等へ止める。
- **Observe/Recover（既存Effect/Consumer Owner）**：Result body、出力ファイル/ログ、Read Footprint、実Commit/Effect、Consumer実受領を主張ごとに分類する。取消し要求・transport ACK・Run終了だけで、Commit未発生や再送安全を推定しない。

この再検査は**全ファイル・全Pluginを原子的に一度に固定できると主張するものではない**。既存の保証された読取り/Lock/Host機構がない部分は、全称的な「改版なし」を捏造せず、実施可能なClaimを限定する。Resource Lease・Effect fence/実DB Receiptの設計は#1174/#1129が所有する。

### 8.2 P0-Lの判定を強くする三つの一因子対照

| 受入候補 | Prepare時点は同一で変える条件 | Productで判別すべきこと |
| --- | --- | --- |
| **PL-R06** | Preview成功後にFunction Source/Entry版/Target Bindingだけ変更して実CLIを起動 | 旧Prepared入力を同名・同byte rangeへ自動再束縛しない。実Start時の対象と版が未解決なら拒否。過去PreviewのPASSを現Grantへ昇格しない |
| **PL-R07** | Build/SelectionはProvider B。Runtime Load/実Callで別Artifact/Provider Aを与え、結果値は同じ600 | `selected/included/loaded/called`を別観測にし、実Call不一致をFAIL。Pluginの同名VersionやArtifact digest一致だけで実Callを証明しない |
| **PL-R08** | API/CLIの計算値は600で一致。CLI側だけ結果本文欠落、出力Flag書込み、またはReceiverが部分受領 | 値計算の局所成功、Host I/O/Disclosure、実Receiver資格を別判定。Q2への必要Result bodyがなければQ2不合格／未確認。Q4/Q5は自動成立しない |

**P0の実装優先は変更しない**。EL-01/04/08/12・PL-R05を最初に実証し、R06/R07はそこで利用できる版Pin/Provider Call Siteから最小追加し、R08の実OS副作用や外部Consumerは#2605/#2504/#2516と後続の別試験で扱う。R06〜08は**提案された判別条件であり実装済みテスト名ではない**。

### 8.3 一回の起動に対する最小Evidenceの読み方

同一Attemptで `(Workspace/Source/Meaning)→(Published Entry/Contract)→(selected/actual Provider+Artifact)→(Host Admission/Dispatch)→(actual Result/Failure)` を対応させる。必要な入力・Source・実Providerが未取得なら、**最初に足りなくなった辺と、そのため成立しないClaimだけ**を返す。APIと実CLIの同値性も、その両入口について同じContract/Fixture/Target/Authority条件が満たされた範囲で判断し、別入口のReference結果を混ぜない。

最初のP0-Lの試験票には、元のIssue/Fixture IDと独立Expected（#1180/#2514）への**参照**を置く。ログからExpectedを再生成しない。AIがIssueやContextを読んで補助テストを提案する場合、未提示の必要情報を自律発見したか、公開Sourceの引用を反復しただけか、Provider/Consumerへ実到達したかを区別する。TEST-T1B（#2514）は独立Reviewerの判別基準を保ち、P0-Lの合否へ無条件に加算しない。

**完了上限**：同一Attemptで成功したpure Q2は、Q3の実SQL/rowsetやQ4の保護された書込み、Q5の外部受領を含まない。GitHub Actionsのsuccessも、PRの現在HEAD・該当Testの実PASS・実CLI呼出し・観測したClaimに照らして報告する。

### 8.4 Providerの実到達証拠は配布形態によって違う

TEIの第一配布形態はGo単一バイナリである。一方、外部Host向けのJava/.NET/WASMや、Script/Pluginの動的実行も許容する。**「実ロード」を一律に共有ライブラリのLoadイベントや外部プロセス起動へ限定しない**。代わりに、どのArtifactにどのProvider実装を含め、実行時に何を選択し、実際にどのCall Siteを通ったかを、配布形態ごとに検収する。

| 実現形態 | 実Provider到達の最小観測 | 誤った合格 |
| --- | --- | --- |
| **Go静的合成・単一バイナリ（P0）** | 固定されたGo Build/ArtifactとComposition・Provider選択、実Instanceへの型付きCall。合成の比較試験では**A/B二実装を同じResultにし、独立のCall Counter/Spy・Call SiteでBのみが呼ばれた**ことを観測する | 単に`ProviderID="B"`という自己申告を返す、BがBuildに含まれている、Resultが600 |
| **解釈Script／Language Plugin** | 消費した元Script Byte・Source occurrence・Interpreter/Operator版と実Executorが一致し、公開Entryの対象として実評価される | Parse/Anchor/Role labelだけで起動資格や無副作用を付与 |
| **動的Plugin／外部Process／WASM等** | Contract/Pin・選択Runtime実装・実ロード成果物/実Process・呼出し・能力表現の一致 | Package存在やプロセス起動、Wire ACKだけでFunction/Effect成功と宣言 |

「Bのみが呼ばれた」は**既存の型付きApplication API経由でテストし、参照ExecutorやStubへの別経路を用意して満たさない**。静的Linkの監査に不適切な動的Module IDを要求してGo単一バイナリを不利にしない。逆にGo Binaryが一つであっても、その中のどのProvider実装に実Dispatchしたかを結果値から推測しない。Hook/Child Functionが追加のProviderを呼ぶ場合、その到達は親の単一pure Q2に暗黙で加算しない。

**担当**：P0の型付きProvider Callは#4495、Build/Plugin選択は#1251/#951、外部Target/Process/WASMの能力差は#2390/#4384、OS egress/実資源は#2504。ここから新ABI・共有Runner・第二Plugin Registryを採用しない。

## 9. 標準Generatorの再利用・AIによる拡張と、実行時に使ったものの照合

標準的なGeneratorをTEIの再利用可能な部品として優先し、生成AIは**設計情報との接続・設定・案件固有の差分実装**を担う。既存の実装で必要な意味を表現できない場合だけ、新しいGenerator/Pluginの候補を作る。AIがEntry、Trigger、Adapter、Provider呼出し用コードを提案・生成できても、そのコードや設定が**公開されたこと・選択されたこと・実際に呼ばれたこと**を意味しない。生成AIの判断と実行許可も同一ではない。

**位置付け**：#4495のP0は、手で固定した一つのpure Functionについて**実Goバイナリ→同じTyped Operation→実Provider**を成立させる作業であり、Generator開発を先行必須としない。本節はその経路ができた後、同じ独立Expected・Fixture・Call Siteを生成されたEntry/Adapter/Provider実装にも再利用するための設計・受入計画である。Generator DefinitionとImplementation Candidateの役割は#1121/#2595/#2514/#2600、可変Lifecycleの義務は#4198が所有する。新しいGenerator RegistryやScript Launcher、万能のGraph Compilerを設けない。

### 9.1 生成・公開・実行を分離する

| 段階 | この段階で判定すること | まだ保証しないこと |
| --- | --- | --- |
| **生成対象の定義** | 受理したFunction/Entry/型/Effect、Targetと明示した生成範囲。人間可読な元DSL/Markdown糖衣は保持 | AIが提案した意味が自動的に正本になること |
| **Generator候補と独立検証** | Generator自身の版、対応するContract/Target、固定Sourceと独立Expected、既知Mutantを落とせるか | 別Contract/新Target・別Toolchainの正しさ、生成コードの自動公開 |
| **生成実体とBuild** | 入力として**実消費した**Source/依存byte、Generator・Toolchain/Target、出力Member/ArtifactとDigest、未列挙依存 | 生成できた・Digestが一致しただけでRun資格があること |
| **Entryの明示公開・現行Composition** | 対象Function/Entry版、選択Provider、公開先Target/Host、必要CapabilityとScope | Generatorが作成したEntryが自動で公開・認可されたこと |
| **起動と結果** | 現在Grant/対象/版を検証し、実GoコードまたはInterpreter/外部Hostの**実Call**、Result/Failureを観測 | HTTP/CLI成功、Callの自己申告だけで外部Effect/業務完了となること |

**意味保存は単一の数値一致だけで判定しない。** Missing/NULL/Unfetched、金額精度と丸め、条件分岐と短絡評価、Ruleの順序、Read Footprint、業務拒否と技術失敗、Effect宣言・現在のScopeを、Generator候補の適合Claimに応じて検証する。すべてを初回のsimple multiplyに強制するのではなく、**その単純例で判別できない義務を未検証として残す**。

Go静的合成は、Generatorの出力Member/Composition・ビルド実体と、実呼出したProviderを別に確かめる。動的Script/PluginではSourceとInterpreter/loaded artifactを検証する。方式が異なっても意味と独立Expectedは共通にできるが、物理的な実ロードや実OS資源の証拠の形を無理に同一化しない。

### 9.2 既存のGenerator実装から再利用する範囲

[#2595](https://github.com/hat47x/tei-platform/issues/2595)では、実評価Sourceと宣言Sourceの不一致が検討され、[PR #3673](https://github.com/hat47x/tei-platform/pull/3673)はScriptのD1→D2→D1入替えによる旧判定の偽陽性を特性化した。その後[PR #4213](https://github.com/hat47x/tei-platform/pull/4213)がmainへ統合され、`tools/reference-generator-selection/selection.go`の限定Generator候補評価では、照合したSource byteを使った**評価用Snapshot**へ処理が進んだ。これは本節の実装に有用な参照例である。

ただし、**Generator SelectorのReference上で単一候補Sourceを固定したことは、TEI製品Launcherが実Providerを同じSource/Artifactから呼んだ証拠ではない**。相対Import、環境変数、未列挙Goファイル、Builder自体、実RuntimeやOSの書替えを自動的に閉包しない。固定したPathの前後のDigest一致も、競合中に何を実読したかの完全な独立証明ではない。#2595/#2600の正しいスコープを利用し、#4495へReferenceをそのままコピーしない。

### 9.3 既存の独立Expectedに追加する一因子負例（計画・未実行）

| ID | 比較入力・故障 | 期待する拒否・限定的な合格 |
| --- | --- | --- |
| **GL-01** | 生成AIがまだ受理されていないFunctionから公開Entryコードを生成 | 候補の生成自体は可。**明示公開されたEntryや現在Host許可がなければ呼出しは拒否**。生成成功でQ0/Q2へ昇格しない |
| **GL-02** | Generator A/Bは双方600を返すが、BだけMissingを0に変換・Rule順序を変更 | 数値一致は局所合格に留まり、独立の負例で意味保存不合格。Generatorから出したExpected自身で自己判定しない |
| **GL-03** | ManifestでSource D1を申告し、実Buildは別のmain.go D2を消費する | 実消費byteと宣言byteの帰属が不一致。#2595のGo反例・既存Selectorに接続し、同じ名称や生成成功でPASSしない |
| **GL-04** | Source/Contractは同一、Generator/Toolchain/選択Plugin版だけが変化 | Artifact再利用、Build省略、Evidence再利用、**現在起動資格**を別に判断。等Digestでも旧受理・現在Grantは自動復活しない |
| **GL-05** | 生成したLifecycle Graph/Adapterが検証Guardを飛ばし、直接Effect呼出しへ接続 | Graph上の見た目や合成実行だけではOKにせず、#4198の不変Obligation・#1174のEffect境界で拒否。P0 pureを実Effect合格に拡大しない |
| **GL-06** | DSL/Markdown説明・レイアウトだけ変更し、実入力の意味・生成用依存は不変 | 根拠を明示できれば無関係な実Binary再Buildは不要。逆にSource/Role/Operator/公開Entryが変われば表示上同じでも現在資格を再評価 |
| **GL-07** | 生成されたCLI Adapterは実Go Provider A、直接APIはBを呼ぶ。両者600 | #4495 EL-04/08を**同じ独立Expectedで再使用**し、どちらかが参照Executorへ逃げても不合格 |

この表は新テストの正式ID採番や第二Scenario定義ではない。既存#1180 Driver Scenario・#2514独立Expected・#2595 Generator Selector反例・#4198 Graphの検証義務へ**参照を戻すための作業票**である。生成AI・Sonnet等のモデルの優劣ではなく、どの候補でも自分で改変できない固定検証基準によって適合を判定する。

### 9.4 既存計画へ渡す実装順と停止線

- **P0のまま**：#4495でEL-01/04/08/12を実Go Binary＋直接API/CLI subprocessでGreenにし、実Selected ProviderとCall Site・Outputを同一Attemptで確認する。
- **P1並列**：#2595/#2600で実消費SourceとGenerator/Build/Evidenceの再利用資格、#4359/#4536/#2620で元DSLの意味とRole/Operator解釈、#2514でIndependent Expected/Mutantを分担。
- **P1実起動への接続**：一つの既存Functionから生成したEntry/Adapterを、実ProviderのA/B同値返却負例とMissing/NULL負例へ通す。生成器の全機能・すべてのTargetを先に作らない。
- **P2**：#4198のLifecycle GraphからHook/Triggerを加え、Guardの迂回・Source改版・再送/Effect unknownの独立負例を検証。実DB、EKI、Web、Java/.NET/WASM、外部Harnessは各Ownerの実証へ任せる。

**停止線**：生成されたコードがEntry公開・Grantを自動発行する、Target/Effectに表現できない部分を黙って省く、Generator自身がExpectedを決めて自己採点する、Sourceの宣言と実Build/Call入力の違いを隠す、新しい中央Host/Registry/認可Engineを必須化する、といった案は採用しない。検証できない条件は`unknown/unsupported/NOT RUN`とし、現在のQ2成功を主張しない。

### 9.5 標準Generatorの合成・レビューと、AIの担当範囲

この節は#1016のApplication Composition Profile、#951のContract/Provider選択、#1121のLowering、#4198のLifecycle Graphを**再利用するための比較計画**である。新しいGenerator固有のResolver、汎用上書き優先順位、Coreの固定Plugin型、実行Engineを設けない。

**採用する作業順序**は、①既存の標準Generator/Pluginが必要な意味型・Target・Effectを満たすか照合、②受理されたモデルと標準部品を明示接続、③対応できない案件固有の差分だけをコード化、④独立Expectedで結果を検証、⑤生成結果のレビューとEntry公開を分離、である。通常の手書きコードや既存コードの段階的リファクタリングも同じTyped Function/Contractの利用形態として許容し、必ずGenerator経由へ作り直すことを求めない。

| 構成上の問題 | 先に確かめること | 許容・停止の原則 |
| --- | --- | --- |
| **出力所有の重複** | 二つのGeneratorが同じファイル・Symbol・Entry・Artifact memberを書こうとしていないか | 合成の明示契約がなければ競合とし、登録順/実行順での暗黙上書きを禁止 |
| **順序と依存** | Aの出力をBが読むときの依存・入力版・出力所有者、独立処理の可否 | 明示された有向依存だけに従う。循環・不定順で同じ最終byteが偶然出ても適合とはしない |
| **部分失敗と再試行** | どの生成出力が有効で、どのBuild member/Entryが未完了か | 一部成功を全体公開済みへ昇格させない。部分成果物の残存・旧版混入は#2600/#943の既存Artifact/Receipt境界で診断 |
| **意味と出力の一致** | Source意味、必要Field/Role、Target制約、Input/NULL、Effect、Failureが維持されたか | Text byteや構造一致だけでは振舞い同値としない。独立Expected/既知Mutant（#2514）を段階的に適用 |
| **公開・受領** | 生成物のファイルレビュー、意味受理、Entry公開、現在Host Grantと実Provider Call | それぞれ別の判断。AI生成・テスト合格・レビュー済みだけでは実行権限を得ない |

合成演算を最初から万能の`Add/Replace/Wrap/Parallel`言語に固定しない。最初の比較は、既存標準Generator **1件＋明示した案件差分1件**に絞り、必要な依存・所有・失敗の判別例から既存Composition/Build表現の不足を見つける。構成に対応できない場合は`unsupported`を明示する。異なる実装順やPlugin配置でも、独立したMeaning/Expectedが同じなら、同じ実Provider・実出力を確認できる範囲でだけ同値性を認める。

**人間のレビュー**は、まず生成ファイル単位で差分と責務を確認できることを標準とする。一つのファイルに複数の意味役割が混在して担当判断が分かれる場合に限り、OKF等で**要素の識別子・レビュー対象範囲・判断根拠**を対応付ける。レビュー事実と受理済み意味、対象Version、Entry公開、実Grantを一つの承認フラグに畳まない。AIに見せるレビュー範囲はその時点のDisclosureに従い、別Workspace/別Roleの内容を自動送出しない。

#### 最小の判別試験（新Test SSOTではない）

| 候補 | 一つだけ変える条件 | 独立Expected / 委譲先 |
| --- | --- | --- |
| **GC-01** | 標準Generatorで対応できるFunctionに、AI作成の個別Generatorを無条件で差し込む | 既存適合契約を無視した不要な差替えを選択しない。#1016/#951（選択ルールは既存Owner） |
| **GC-02** | 二つのGeneratorが同一Entry/同一成果物Pathを生成 | 出力所有競合を検出し、後勝ちにしない。#1121/#2600 |
| **GC-03** | 依存する生成工程A/Bの順序を入れ替え、結果byteだけ同じ | 選択した入力/依存契約が未解決ならBuild資格を与えない。#2600/#2595 |
| **GC-04** | 生成器Bが失敗した後にAの旧成果物だけが残り、古いEntryが参照 | 部分成功を公開可能な一式と扱わず、必要Artifact memberの欠落/旧版混入を診断。#943/#2502 |
| **GC-05** | 生成したコードは600を返すが、NULL、条件分岐またはRead Footprintが異なる | Value PASSと意味Conformanceを区別し、独立Expectedと既知Mutantで不適合を判別。#2514/#1180 |
| **GC-06** | AI生成ファイルのレビューのみ済ませ、意味採用・Entry公開・Host Grantは未実施 | レビュー済みを実行許可へ自動昇格せず、実Provider Callは0。#2502/#4495 |
| **GC-07** | 案件固有コードを既存の標準Generatorと組み合わせるが、同じFunctionの意味定義を二重に保守 | Canonical意味を一つにし、手書きコードとモデルの不一致を診断。#1121/#2620 |

**後続作業への引渡し**：G1（Generator DefinitionとImplementation Candidateの区別）→G3（独立した意味Conformance）→G2（所有/順序/部分失敗を伴う合成）の順に検証を進める。G5の方式・ABI比較とG7の自己記述Bootstrapは並行研究とし、P0製品起動に必須化しない。G9のLifecycle Guard・G10のAI生成Pluginの信頼境界は、Entryの実行直前の認可・Effect/Read Footprint検査へ接続する。これらは**検討課題の略称**であり、新しいCore機能が採用・実装済みという意味ではない。

**実施状態**：既存Issue/標準契約と別チャットの設計を統合した作業順・受入候補のみ。GC-01〜07の製品Go/CLI/Provider/Host E2E、複数Generatorの実合成・人間レビュー運用実績は **NOT RUN**。初回P0のpure Function実起動は引き続き#4495が担当する。

## 10. Script Launch・標準Generator・TEIセルフビルドの接続点

**番号の区別**：以下のGenerator「G1→G3→G2」は[#4544](https://github.com/hat47x/tei-platform/issues/4544)内の検証順であり、[#1119](https://github.com/hat47x/tei-platform/issues/1119)の製品横断G0〜G4とは別の作業ラベルである。どちらの番号もScript Launchの新しい実行状態や必須Gateではない。

TEI Studioデスクトップ版の構築と、TEI自身によるセルフビルドを製品全体の重要な到達目標とする。一方、**実Go Functionの起動、標準Generatorの適合、セルフビルドの成功は、検収単位も未実証点も異なる**。最初から三つを一つの合格条件に束ねず、各レーンを先行並列で進め、実際の成果物の受渡しを一つの縦断例で確かめる。

| レーン（既存Owner） | 最初に成立させる独立の証拠 | 次のレーンへの引継ぎ物 |
| --- | --- | --- |
| **L：Script Launch #4495** | 現在公開される一Entryから、実Go製品CLIと直接Typed APIが同じ選択済みProviderを実呼出し。非公開・選択誤り・結果未取得を拒否 | 対象Function/EntryのMeaning、Target、実Provider identity、実CLI Artifactと観測Result、独立Expected |
| **G：標準Generator #4544/#1063** | 既存Generatorを優先選択し、Accepted MeaningからGoの一実装候補を生成・独立Conformanceで比較。使用したSource/Generator/出力物・既知Mutantを固定 | 検証済み生成候補と生成Source、実Builderへの入力、必要なRole/Operator Pin、適用可能なTarget範囲 |
| **B：TEIセルフビルド #2831** | 既存標準Build graphとGo adapterから**TEI CLI等の実Artifact**を構築し、宣言Graphと実Command/実消費Source、ArtifactSet・toolchain・stageを照合 | 実ビルドした対象CLIのdigest/Target/実行可能なartifactと、独立のBuild Evidence。Stage1/2収束≠Seed trust |

三つの合流では、**Bで得た実CLIバイナリをLのCLI入口として呼べるか**を一つの公開合成Functionで試す。ただしこれは**各独立検収後の結合試験**であり、初回LのCLIを「TEIセルフビルド済みの成果物だけ」に制限しない。Gで得たFunction実装候補も、意味適合だけでLの公開Entryを自動生成・公開・許可しない。直接APIの試験用GoバイナリとCLI製品バイナリは物理的に別でもよく、同じSource/意味・選択Provider契約・実Callをそれぞれのbuildに対して確認する。

**合流でだけ判別する失敗（新たな独立Expected SSOTは作らない）**：
- **LB-01**：Bが正常ビルドしたCLIと、Lが実行したCLIのartifact digestまたはTargetが異なる。L単独のQ2結果は保留せず対象を区別し、**B→L引継ぎの一致だけ**を不合格にする。
- **LG-01**：Gで独立検証した候補とLが実呼出ししたProviderが違う。数値が600で一致しても**G→L引継ぎは未成立**。実Provider選択と実Callの証拠を再確認する。
- **GB-01**：GのSource/Generator版がBの実消費入力に含まれず、古い生成物でCLIをビルドした。Build/Evidenceの再利用を止め、依存/Source Closureの不足を#2600/#2595へ返す。
- **SL-01**：Studio表示やPinで「実行可能」に見えるが、Hostの公開Entry/現在Grantが不一致。Studio/AIへの結果投影と実起動は別Owner（#4515/#1824）で検収する。

**主線との関係**：Lの最小Q2 → Gの最小Generator→ Bの実CLI全体Buildをすべて完了してから初めて製品を動かす、という直列化は行わない。#4495と#4544/#2831を並列に進め、足場がそろった時点でB→L、G→L、G→Bの順に接続可能なところだけ確認する。B→L統合後にStudio/Desktop（#1824）から同じTyped Operationを起動し、さらにStudio自身のBuild/更新をTEIモデルへ移す。初回のStudio GUI/Wailsや本番DB/外部HostはHard Gateにしない。

**報告**：各レーンの実ソース・Artifact/Producer・Builder/Toolchain・実Consumer・Independent Expected・試験結果/未取得を別欄にする。CI成功や同値Outputを三レーンすべての合格に加算せず、結合テスト未実施なら`NOT RUN`。本節は計画と受入候補であり、L/G/Bの新規実結合試験は**NOT RUN**。

## 11. Script Launchの実装接合：Application APIから実Providerまで（2026-11-14 横断追補）

本節は、既存[#4495](https://github.com/hat47x/tei-platform/issues/4495) PR-Aの実装担当へ渡す**最小差分と検査順の設計**である。新しいEntry正本、Run Engine、Universal Dispatcher、Grant Registry、Expected/Scenario正本を設けない。正式CLI構文・DSL schema・Runtime ABIをこの記述で確定しない。現行mainの個別Go Sourceを静的に確認した結果であり、以下の製品Go/CLI・実Host・Effectの新規試験は**NOT RUN**。

### 11.1 既存の三つの境界は役割が異なる

| 現行Source | 実装されていること | Script Launchへ直接昇格させないこと |
| --- | --- | --- |
| `internal/application/model.go`、`internal/applicationview/command_plan.go` | `InteractionCommand`・`BuildCommandPlan`で対話のCapabilityと入力を解決する。入力は`CommandPlanInput{Value string, Provided bool}` | UI Commandの宣言・`Ready`は公開Entry資格でない。`Provided`だけではexplicit NULLとunfetchedを区別できない |
| `internal/command/invoke.go` | `command.Request{Application,Pattern,Capability,Inputs map[string]string}`を受け、宣言・入力・明示Authorizer・Effect/Outcomeを検査して`Executor`へ渡す。ReferenceはMemoryUnitOfWorkでありCommit≠Persisted | 既存Requestを全Scriptの公開Entry/Target/Provider/Operation identityを備えた万能Invocationと見なさない。CLI/StudioのReference成功だけではProduct Q2でない |
| `internal/servicehost/service_carrier.go`、`internal/plugin/runtime/service_carrier.go` | Hostが保持するgeneration/route、current Service acceptance、実Provider/Artifact/Runtime provenanceの照合を行う。Hostの世代切替中もPinned routeを維持する | Service Carrierは公開Entryを解決せず、業務Request/Result shapeも検証しない。外部Service用の経路をGo静的Functionにも無条件に強制しない |

**接続判断**：公開対象は既存Accepted MeaningとComposition/Contractの**選択済みProjection**から解決する。解決済みEntryと業務Inputを型付き共通Operationで受け渡し、実行形態ごとに既存Host/Providerへ接続する。実Go経路でEntry解決または型付きInputが表現不能なら、その**第一不足辺**を示して最小変更だけを加える。新しい`run`コマンドや共通Envelopeを先に設計しない。既存API方針は[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)、Product gateは#4495が正本。

### 11.2 PrepareとDispatchは同じ資格ではない

1. **Prepare（副作用のない候補構築）**：Workspace/Qualified Function/公開Entry/Contract版・Target、必要Source/Operatorと読込状態、入力状態、期待対象Revision、選択済みProvider/Artifactを特定。表示名・Role・AI提案・Source Indexは候補であり公開権限ではない。
2. **Admit（現在のHost判断）**：Host自身が確立したPrincipal/Scope/Target/Grant、Entry公開状態、実Artifact/Provider適合、必要な期限・Resource、対象Revisionを検査。外部から受け取ったactor/role/Grant文字列を信頼済みContextへ変換しない。保護されたEffectではCommit直前の条件をそのEffect Ownerが再評価する。
3. **Invoke（実呼出し）**：Go静的合成ではリンクされた実装の**型付きCall Site**、外部Process/動的Pluginでは選択・実Load/実Process・実Invocationを観測。selected B、called Aでも600が返ればFAIL。未知のRequired入力/Target能力があれば勝手に省略せず拒否。
4. **Observe（結果の分類）**：Result本文、Failure、実Providerの識別、取消/timeout、Effect/Commit/Receipt、Consumerへの開示・実受領を必要なClaimごとに記録。pure Q2は戻り値と実Callまででよく、DB書込・外部送達を一律に要求しない。

入力については`value`、`explicit-null`、`missing`、`unfetched`を契約上区別する。とくに既存`map[string]string`の`""`へ意味を畳まない。型付き入力を既存#2056 Query入力や#988のSemantic/Operatorで再利用できるかを先に調べ、**P0では実Fixtureに必要な有限の入力形だけ**を実装する。広範な四状態Schemaの完成はPR-AのHard Gateではない。

同じEntry alias・画面上のPinが残っていても、Source A→B→A、Provider切替、Selection epoch、対象Relation/where変更、Grant失効後には旧Prepared Invocationの資格が自動復活しない。既存#4515のPinは作業対象であり、#2499/#1174のGrantではない。検索・編集中のフォーカス変更はPinを不意に解除しない一方、起動直前に現在性を再検査する。

### 11.3 P0実装のRed→Greenと独立Witness

一つの既存pure Function（例：数量2×単価300＝600、意味・NULL/丸めは#1180/#2514の独立Expectedから確認）を利用する。**API直接呼出し**と**実製品`tei` CLI subprocess**が同じApplication Operation・選択済み実Go Providerへ到達することを確かめる。事前に別の試験用Reference経路を作って同じ値を返しても合格にしない。

| 検査 | Redの変異・失敗注入 | Greenの必要証拠 |
| --- | --- | --- |
| EL-01 | Functionは定義されるがEntry未公開 | Product API/CLIで拒否、Provider Call Counter 0 |
| EL-04 | Provider B選択、Aを実Call、双方600 | Call Site外側で独立Spy/CounterがBのみの実Callを確認。自己申告ID・Output一致では不可 |
| EL-08 | APIは実Provider、CLIはReferenceExecutorで600 | 実製品CLIプロセスのbinary/argv/exit/stdoutと直接APIが同じFunction/Contract/Provider・Failure意味へ接続 |
| EL-12 | CLI exit 0／transport ACKのみ、Result body欠損・不正形 | 実Response bodyを取得・意味値を独立Expectedに照合。欠損時はProduct Q2を認定しない |

Resultの外側JSON全byte一致は要求せず、意味値・Failure・版・実Call先を比較する。生成器（#4544）の選抜、セルフビルド（#2831/#4583）、Studio画面、全Target展開を本PRの前提にしない。Go静的合成では「動的ロードイベント」を必須証拠としない。Hook／子Functionがある場合は、親の認可や単一pure合格を子の実Provider/Effectへ継承しない。

次PRでEL-05（NULL/欠落/未取得）、EL-06/07（Host context/失効）、EL-13/14（Selection/alias drift）、EL-15（子権限）を独立追加。共通Hookの順序・拒否後作用・二重実行は#4198/#1129のGuard/Effect検証に戻す。定時/Event/Batchは#4497/#1130、Commit後ACK喪失・blind retry拒否は#1174、OS/Secret/外部Egressは#2504へ渡す。新しい横断Scenario正本は作らない。

### 11.4 起動のEvidenceと高速性は、実行形態に応じて検査する

同じAttemptに対応付けるのは、`exact HEAD / Source occurrence / Meaning+Entry Contract / selected Target+Provider / actual binary or loaded artifact / actual Call / Result body / independent Expected`である。ただし各項目の**宣言・実選択・実消費・外部観測**を別Claimとして残す。Artifact digestやResult値が等しくても、Generator版・Provider actual call・現在Authority・Disclosureを同じと推定しない。

- **静的Go binary**：実BinaryとCompositionで選んだ実装の区別、型付きCall Siteの独立観測。選択されないPluginが初期化・作用を起こす反例は#1251へ引渡す。
- **解釈Script**：実消費Script bytes、選択Operator/Interpreter、Evaluator到達とSource/意味版。
- **外部Process／Java/.NET/WASM／EKI**：選択した実Artifact/実Process、必須Input/EffectのTarget表現能力、実送達・実Result body。必要フィールドを表せなければ`unsupported/not-representable`。
- **性能**：cold CLI、warm in-process呼出しを分けて、Entry解決・Provider準備・呼出し・結果整形を測る。EL-19へ接続。Plugin総当たりLoadや参照Index全件再構築を起動の既定手順にしない。数値SLOは製品Targetと実測から決める。

### 11.5 Studio・AI・Triggerとセルフビルドの合流条件

**Studio/AI**：固定対象（Pin）と実行資格（Entry＋Host Grant）をUIで別表示する。AIハーネスはTarget/Typed Invocation候補・独立テスト候補を作れるが、公開、受理、Grant発行をしない。送出・履歴表示時には現在Disclosureで再投影し、秘密値をOperation ReceiptやIssueへ保存しない。第三ConsumerとしてのStudio/Wails IPCは同じTyped Operationで、別のショートカットExecutorを持たない（#4515/#1824/#3422）。

**Trigger**：Manual/Schedule/Event/Batchは**同じEntry/Function**に異なる契機を結び、Occurrence→Prepared Invocationを作る。Schedule tick、Event delivery、Queue入場は権限でもBusiness Commitでもない。時刻/重複/再入/取消/リトライの本体は既存#4497/#1130/#1129/#1174へ委譲し、P0のpure直呼出しを遅らせない。

**セルフビルド**：#4583のB0については[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)に同一HEADの生成Inspector＋実Windows WebView2 ConsumerのCI証拠があるが、Draft未統合・B1 BuildIntentによる多工程駆動、B2 Stage1自身によるStage2 Desktop生成、B3そのStage2 exeの実WebView2は別資格。L/G/Bの合流は既存LB-01（B実CLI→L）、LG-01（G実Provider→L）、GB-01（G生成物→B実入力）に加え、**Stage2内のEntryが同じTyped Operationから実選択Providerへ到達したか**をDesktop Consumerとして後続で照合する。別HEADのGreenや生成物の同値だけを合算しない。Stage1 BuilderとStage2 Runtime Providerは異なるRoleであり、ビルド用GeneratorをStage2へ無条件同梱しない（#4601）。

### 11.6 Issueへの引渡しと停止線

| 既存Owner | 次の変更単位 |
| --- | --- |
| **#4495（P0）** | 現main exact HEADでEntry→型付きOperation→製品Go Provider→CLI subprocessの第一不足辺を示し、EL-01/04/08/12を実Go test/実CLIで閉じる |
| **#4502／#2502** | Function/Entry/Trigger/Invocation/Attemptの製品ライフサイクル、公開・停止・版を追跡する。新Registryは作らない |
| **#4497／#1130** | Trigger候補からPrepared Invocationを作り、start時のGrant/Artifact/時点再検査と二重配送のEffect限界を検証 |
| **#4198／#1129／#1174** | Guard順序・再入・子Functionの拒否、実Effect fence／Commit unknownの判断。pure Q2の合格に転用しない |
| **#4544／#2831／#4583／#1824** | G/L/B/Studio各レーンの実ArtifactとProduct Consumerを、合流時にだけ同じAttemptの証拠で接続 |
| **#1180／#2514／#1163** | 独立ExpectedとDriver Scenario、exact actual Evidence Packetを既存Ownerで管理する |

**状態と停止線**：この追補は実装用の設計整理であり、Go製品Function実起動・CLI/API二入口のProduct Q2・実DB Effect・EKI/外部Host・Stage1→Stage2 Desktopの新実証は**NOT RUN**。型付きOperationの具体的Go API/CLI引数、SourceからEntry公開を投影する構造、正式Trigger DSLは現行CodeのFit/Gapを経て各Ownerが決める。新しいRunner/TEI専用Host/Permission Engine/Scenario SSOTを導入しない。

## 12. 実装接合の再精査：Capability Resultを再利用し、Entry公開・実Call・実行資格を埋める（2026-11-14）

前節§11は現行`command.Request`と`CommandPlan`の不足辺を示した。追加で現行mainの`internal/semantic/model.go`と`internal/semantic/capabilityinvoke/invocation.go`を確認すると、**Capabilityの単一Output/名前付きOutcomeを区別する既存実装**がある。したがって、純粋Function用の値を`command.Result`に無理に追加する案は採らない。以下は既存契約の適合調査を先に行う実装指示であり、新API/Schema/CLI構文の採用を意味しない。

### 12.1 最初に再利用を検査する既存Capability境界

- `semantic.Capability`は`Inputs []Parameter`、`Output core.CanonicalID`、`Outcomes []Outcome`、`Effect *DataEffect`を持ち、単一結果型と名前付き業務結末を意味上で区別する。**Capabilityが存在するだけではどのHost/Consumerへ公開されるかは決まらない**。
- `semantic/capabilityinvoke.BusinessResult`は`Capability`、`Mode=output|outcome`、`Outcome`、`Payload any`を持つ。`Invoke`は`Adapter`呼出し、Capability identityとResult mode／Outcome名を検査する。ただし`ValidateBusinessResult`は**Output Payloadの具体的な型・NULL/丸め・単位を包括検証するものではない**。Provider本人の識別自己申告、`Payload any`、値600だけでProduct Q2としない。
- `command.Invoker`は対話CommandとData Effect/Transactionの既存Ownerであり、`Inputs map[string]string`を現状の形式として尊重する。`BusinessResult`の`Payload`を`command.Result`の`Outcome`へ偽装しない。Effectful Actionは既存Command/Effect Ownerで今後接続し、P0純粋Functionの新規経路を「全てのActionの実行規則」に先行拡大しない。
- `SelectedServiceCarrierInvoker`は、Host generation/current Service acceptanceを固定したService実装の特例的な実呼出し足場。Go静的合成pure FunctionにはCompositionが実選択した実装への直接の型付きCallを認め、**同一Result契約・実Call同一性を形態別に検査**する。

結論：**①Meaning/Result契約、②公開Entry Binding、③Host実行資格、④実Provider選択、⑤Resultの実型値、の五つは独立した根拠を必要とする**。既存の`capabilityinvoke`をProductの公開・選択・認可・実Callのすべてを持つ新Dispatcherへ変えない。

### 12.2 起動可能性は一個のReadyフラグではない

発見・プレビュー・実行可能・実行済みを単一`ready/ok`に畳まず、既存Source/Composition/Host/Runtimeから**必要なときに読取りで導出**する。専用永続Registryは作らない。

| 必要な判定 | 根拠となるOwnerと観測 | 未充足時の表示と処理 |
| --- | --- | --- |
| 定義が解決できる | Workspace/Kind/ID/Source occurrence、曖昧性・Required Source/Operator | missing/ambiguous/partial/unknown、同名fallback禁止 |
| 意味・型が適合する | Accepted Function/Capability Contract、Output type、Input state、Target fit | unsupported/invalid/not-representable、数値一致のみでPASS不可 |
| Entryとして公開されている | 選択済みApplication Composition/Profileと現在の公開版・Consumer surface | discoverableとpublishedを別表示。非公開時にProvider call0 |
| Artifactと実Providerを選べる | Build/Plugin Lock、対象Target、実Binary/Artifact、実Call route | selectedとincluded/loaded/calledの相違を明示。未選択候補は呼ばない |
| 現在のHostが入場を許す | Hostが確立したPrincipal/Scope/Grant、期間・Resource、対象Revision、期限 | authorizedがunknown/失効なら実Call拒否。PreviewやPinはGrantでない |
| 実結果を検収できる | 実Call Witness、Result bytes/typed payload、独立Expected、Failure/Effect別観測 | exit0/ACKのみならresult unverified。純粋計算とHost I/Oを分離 |

**QualifierはClaimごとに付く**。たとえばEntryはpublished、Artifactはincluded、Grantはunknown、実Callはnot-executedでよい。表記上の部分成功を全起動資格へ昇格しない。公開／現在許可／実呼出しと、履歴の保存／現在の開示許可も別。

### 12.3 最小のProduct APIへ渡す情報と、その更新境界

新しい共通Envelopeを固定する前に、既存`internal/operations`と`capabilityinvoke`の可用な値／Ownerを棚卸しする。追加が必要な場合、**一つの公開Function起動専用の型付きOperation**に限定し、Conceptualな次の項目を欠かさず扱う。これはGo構造体・DSL・Wire schemaの正式提案ではない。

- **Callerからの起動意図**：Qualified Entry Ref、Contract expected revision、typed inputs（`value`／`explicit-null`／`missing`／`unfetched`を区別）、Target/Subjectとexpected revision、必要ならoperation identity/deadline。External actor/role/grantはAuthorityとして受け取らない。
- **Hostだけが解決する値**：current Workspace/Environment/Principal/Scope/Grant、現在公開Entry、選択ProviderとArtifact、実行Profile/Resource policy、実行対象の一意性、消費可能なDisclosure。この値を文字列フィールドの申告に任せない。
- **Providerから帰る値とHostの検査**：既存Result Modeと実Payload、実Call Site/Artifactの観測、Contract Output typeに対する入力・出力検査、Failure種別。Output自体は`nil`／明示NULL／未返却の区別と契約上の期待を保持し、`Payload any`のまま無検査合格にしない。
- **履歴上のID**：Trigger occurrence／Operation／Attempt／実外部ExecutionRef／Effect receiptを混同しない。既存#1130/#1174/#4198のOwnerにある再送・実行履歴へ結び、新たな全Function共通Run Storeを作らない。pure P0では観測できないIDを架空に生成して埋めない。

Prepare時に解決できた契約・対象は**許可トークンではなく候補**である。Source A→B→Aやalias更新で同じ表示名・最終byteになっても旧Prepareを自動承認しない。開始時に実Target/Version/Grantと選択Providerを照合する。長時間実行、Commit直前、子Function呼出しの現在性は各Ownerのfence/leaseに委譲し、短命のPrepared Invocationだけで保証しない。

### 12.4 Process境界の独立Call Witness：直接APIと実CLIを同一試験で偽装しない

**同一ResultのA/B対照**が最初の核心。Call SpyをProvider内部の`ProviderID="B"`自己申告にせず、**Composition/Hostが選択した実装を呼ぶ境界の外側**で観測する。プロセス内テストのin-memory counterだけで、別プロセスの実`tei` CLIが同じProviderを呼んだことにはならない。

- 直接API試験：実Provider A/Bを同じMeaning/Result=600で構成し、Bを選択する。Host-owned outer spyでA=0、B=1と契約・typed outputの一致を確認。B選択/A呼出しの故障注入でFAIL。
- CLI試験：実`cmd/tei`製品バイナリを**subprocessとして起動**。binary digest/argv/exit/stdout/stderrを採り、CLIプロセス内の**実Provider境界**で採った観測と照合。単なる親Go testのcounterやCLIが出した`provider=B`文字列、ReferenceExecutorの一致では不合格。観測用ファイル・Traceを使うなら、そのファイル作用をテスト用Host I/Oとして明示し、Provider実計算のpure保証と混同しない。
- **再現性**：実Go binary build/実行と該当Testを同一exact HEAD・Target・Fixtureで報告し、`go test`の対象が0件・skip・mock-onlyなら不合格。独立Expected=600は#1180/#2514から参照し、Resultや生成AIの出力から正解を生成しない。
- **失敗分類**：Entry未公開、Provider選択不一致、Input invalid、Output shape不正、Result本文欠損、cancel-before-call、timeout-after-callを分ける。純粋Functionの出力欠損を「業務否認Outcome」へ変換しない。Effect unknownは#1174の既存契約へ渡す。

この検証で必要なのは**何が選ばれ、どこが実Callされ、何を返したか**である。Go静的合成に動的Module loadのイベントを要求しない。逆に外部TargetのProcess/Worker ACKをGo静的Call Witnessの代用にもしない。

### 12.5 DSL・Trait・Hook・AI・X-TEA・Desktopの統合条件

- **DSL/Role**（#988/#2620/#4359/#4599）：表層構文やGo生成コードで業務意味を二重定義しない。Output型、decimal精度・丸め、NULL、評価順、Operator版を、受理済みCapability意味から実Providerへ写す。部分的な`*.tei.md` Readerや固定OKF IDは意味受理・実Entry公開の代用にならない。
- **Trait/Plugin**（#3278/#951/#1251/#4601）：Atomic Traitのrequired methodが未実装、実Provider未選択、Targetに表現不可なら`unsupported`。Builder/Generator RoleとStage2 Runtime Provider Roleは別。Stage2の起動に不要なGeneratorや未選択Pluginを勝手に同梱/初期化しない。
- **Hook/Effect**（#1129/#1174/#4198）：`before/after validate/transform/commit/notify`の候補順序は既存Lifecycle Ownerで検証し、GuardをGenerator Hookから迂回しない。pure P0はOS出力・ログ書込み等のHost I/Oを別作用として観測。Trusted Developer Scriptのraw SQL/file/network能力とDelegated Macroの公開能力を混同しない。
- **AI/Studio**（#4515/#3422/#2502）：Inspect→Preview→Run→Observeの利用者導線を同じTyped Operationの投影として設計。Pin維持と資格失効を両立し、AI提案・Preview PASS・Generator Reviewから実行Grantを発行しない。現在Disclosureで秘密・別Workspace・未公開Entryを開示しない。
- **X-TEA移植**（#1124/#1129）：Data ScriptはTable/FieldのLoad/Save/Tx phaseのOwnerへBindし、内部Hookが全て公開Entryへ自動昇格する方式にしない。従来のTableEvaluator/Session等の意味をFacade/Pluginで提供しても、その実Host/DB Effectは別検収。
- **Self-Build**（#2831/#4583/#4601）：全体最優先は**TEIがStudio Desktop Stage2を再構築し、その実Windows Consumerを検査すること**。FunctionレーンのL0は独立先行。合流時には、Stage2実binaryに公開Entry/Runtime Providerが含まれ、実Studio IPC→共通Operation→実ProviderのCall Witnessがあることを確かめる。Draft #4587の既存B0（生成Inspector/実WebView2）のGreenをこのFunction起動成功へ流用しない。

### 12.6 既存計画の最小PR分割と、証拠に応じた停止線

| 作業単位・Owner | 実装時の作業範囲 | 初回に要求すること／後続に残すこと |
| --- | --- | --- |
| **L-A #4495 / #3422** | 既存`Capability`/結果modeと公開Entryの解決を接続し、必要最小の型付きOperationを作る。必要なら既存`capabilityinvoke`を値検査の足場とする | P0に必要な数量・単価・Result形を先に通す。すべての入力状態・全CLI ABIの同時刷新はしない |
| **L-B #4495 / #1251** | Go単一Binaryの選択B・実Call Bと、Product CLI subprocessの同一Operationを観測 | EL-01/04/08/12、独立600、実Call Witness。Reference/外部Service/MockはQ2へ昇格しない |
| **L-C #4495 / #2502 / #2514** | NULL/unfetched、型不一致、Source/Entry/Target/Artifact/Grant失効、子権限を局所負例に追加 | 初回Q2のBlockingにせず、既存Scenario SSOT/Driverで対象ごとに資格化 |
| **T-A #4497 / #1130** | TriggerDefinition→Occurrence→Prepared Invocationのread-only投影、入場再検査、重複配送・期限を既存Ownerへ渡す | Scheduler/Brokerを作らない。実Delivery/Commit/ReceiptはG2/Effectで別 |
| **LB/G/B合流 #4544 / #2831 / #4583 / #1824** | G実生成→B実消費→L実Call→Stage2実Desktop Consumerを、同じAccepted Contractと因果的に結ぶ | レーン別PASSを加算せず、実Artifact引継ぎ・実実装使用・独立Consumer検収を個別に記録 |

**変更による再検証**：World Fingerprint／Model Revision／Artifact Revisionを混同しない。Source、Meaning/Role/Operator、Provider Lock/Toolchain、Entry公開、Host Grant、実Call route、Disclosureのどれが変わったかで、そのClaimに必要な資格だけを再検査する。layout-only変更や無関係Plugin追加で全Buildを無条件にやり直さず、必要な否定依存・未読込・外部参照が不明なら`unknown/partial`を維持。A→B→Aでも旧Authorization/Entry資格は自動復活しない（#2600/#3262/#2514）。

**実施状態**：本節は現行mainの個別Sourceと既存Issue/他レーンのDraftを照合した**docs-only実装計画**。新Go Product Function起動、実`tei` subprocess、型付きOutput/Entry受入、Host/DB/EKI Effect、Stage1→Stage2 Function合流の本節による検証は**NOT RUN**。形式の最終採否・Owner Meaning Acceptanceは既存Issue/ADRで判断する。

## 13. P0-Lを実装可能にする最小の具体的変更範囲（2026-11-14）

現行main `e888bb8f21367b721f0984953530da9627e2634d` の個別Sourceをさらに確認した結果、**結果契約、製品CLI入口、最初の業務意味Fixture**の三箇所に、§11〜12では十分に特定できていなかった不足がある。本節は#4495の実装PRへ直接渡すチェックであり、独立のLaunch仕様／新Issueを増やさない。main更新後は、着手PRのexact HEADで再確認する。

### 13.1 参照Workspace内のCapability実数とP0候補の受理境界

main `e888bb8f...` で`reference/workspace/semantic`配下の6つの`*.tei` Sourceを直接読み、明示`capability`は次の3件と確認した。これは**このReference WorkspaceのSource定義の棚卸し**であり、全TEI Workspaceや全PluginのCapability一覧、個々のMeaningのOwner Acceptance、公開Entryの存在証明ではない。

| 参照WorkspaceのCapability | 宣言された結果 | P0 pureの流用判断 |
| --- | --- | --- |
| `sales.order.submit` | `input order sales.order`、`output sales.order` | 受注確定という業務用途。`Effect`未宣言という構文上の事実だけでpure/no-read/no-side-effectと推定しない。既存`capabilityinvoke`のOutput mode局所試験は維持するが、計算Functionへ無条件転用しない |
| `security.authenticate` | `authenticated/denied`の名前付きOutcome | 認証文脈を持つ契約。Result modeの参考例として利用可、公開業務計算の独立Expectedとは別 |
| `security.authorize` | `allowed/denied`の名前付きOutcome | 認可を扱うSemantic契約であり、**Hostの現在Grant決定の代わりに採用しない**。意味の一致・Effect absence・Host admissionは別根拠 |

同じReference Workspace内には`sales.order_line.quantity`・`unit_price`属性があるが、Generatorの`sales.line-amount`は`testdata/line-amount/contract.json`にのみある**`reference-local`規則**であり、上のCanonical Capability一覧には登場しない。従って、**現時点で確認できるこれらのSourceをそのまま使って「Owner受理済み・明示公開済みのpure line-amount Product Entryがある」とは主張できない**。

**A0の分岐を確定**：①別の実Workspaceに既存のOwner受理済みpure Functionが見つかれば、そのSource occurrenceとContract/Effect/Target/公開面を実検査して採用する、または②`line-amount`等の有限規則をOwnerが通常の意味レビューで明示受理し、Canonical Function・Operator/rounding/NULL/単位/Output/Effectの必要条件と独立Expectedを結び、**さらに別途Entry公開を確認**する。どちらも満たさなければA0は未成立、A1/A2のReference実行でP0 Product Q2を代替しない。採用と公開を自動化するための新承認レイヤーや中央Registryは作らない。

**合否に用いる最小対照**：未公開Functionのcall 0（EL-01）、`line-amount`が`reference-local`のままなら`published`に昇格しない、`Effect`フィールドのnilだけでpureとは宣言しない、`security.authorize`の名前付き業務OutcomeとHost Admissionを混同しない。独立Expectedの正本は#1180/#2514、Function/Entry公開のOwnerは#2502/#4495に維持する。

#### D8の実装資産とProduct起動の分界（#1705・#1637）

さらに`reference/derivation-typed-arithmetic`（#1705はclosed）には、型・単位・通貨を照合して既存`internal/semantic/derivationref.Multiply`による厳密10進演算へ接続し、`RoundExact`で最終結果のみhalf-up/half-even・precisionを検査する**合成Reference**がある。既存testには`2×300=600.00`、`1×1.005`のhalf-up/half-even分岐、kg対piece、JPY/USD、NULLと未取得、精度超過が含まれる。新たな算術Engine・通貨Registry・小数点ライブラリをP0で作らないための重要な再利用資産となる。

ただし`reference/derivation-typed-arithmetic/README.md`は「Canonical仕様、受理済Operator、本番通貨処理ではない」と明記している。`RoundExact`等は**referenceパッケージの検証対象**であり、製品のruntimeがreference fixtureをimportして本番Provider資格を得る構成にはしない。Product Providerの実装では、既存D1の意味契約と`derivationref.Multiply`をFit/Gap検討し、decimal scale/通貨・単位・NULL/OutputのルールはOwnerのAccepted Meaning/既存Operator/Target Contractへ束縛する。参照testのPASSは、業務意味の受理・Operator登録・Stage2実Provider/実CLI結果の検収には昇格しない。

**実装に有効な二経路**：A0のOwner受理を進めながら、A1の技術的な配線・Adapterの局所検査は明示的なsynthetic/reference-only Fixtureで並行してよい。ただし前者の受理済みEntryと後者の合成結果を別Subject/Contract/Attemptのまま足し算してProduct Q2としない。Product合格には、受理済み同一Contract・選択Provider・実Go/CLI起動・出力観測を一つに結ぶ必要がある。

### 13.2 実コード・既存テストから確定できる境界

| 実装／Fixture | 現在確認できること | Product Q2に必要な最小delta |
| --- | --- | --- |
| `internal/semantic/capabilityinvoke/invocation.go` | `ValidateBusinessResult`は単一Outputで`Mode=output`と空Outcomeを確認してreturnする。**`result.Payload`の実型、値や欠落はここで確認しない**。名前付きOutcomeの非nilとは分岐が異なる | 実Entry/HostのOutput契約に従う型・presence・NULL/precision・単位の検査。Generic `capabilityinvoke`へドメイン固有money規則をハードコードせず、既存Meaning/Targetの適合検証へ委譲。nilが適法なTargetもあるので「全単一Outputのnil禁止」を無条件導入しない |
| `cmd/tei/cli_commands.go` | `acceptedCommands`には通常の公開Functionを起動する製品Verbがない。Referenceの`view`等はこの不足の代用ではない | **製品Typed Operationが先**。その後、CLIを薄いAdapterとして一入口追加する。CLIの正式名・引数・Wire versionは既存CAPI-4（#3422）とCLI互換Ownerが採用してから確定。未公開Entryを引数で推測して実行しない |
| `cmd/tei/cli_commands_test.go` | `TestAcceptedCommandsMatchDispatch`が実Dispatchとコマンド一覧をASTで比較し、`TestUnknownCommandRejectionIgnoresFlags`が実`tei` subprocessの未知Verbを検査する | 新しいCLI入口を既存一覧・実Dispatch・Help・回帰テストへ**同時に**接続。単なる`acceptedCommands`追加、またはヘルプ表示だけでProduct起動としない |
| `internal/semantic/capabilityinvoke/invocation_test.go` | 既存`sales.order.submit`の単一Output mode検証と、cancel-before-dispatchのAdapter call0、実行失敗をOutcomeへ変換しない局所テストがある | 既存テストの契約は保持。`sales.order.submit`を**pure計算の受理済みFixtureに読み替えない**。Product公開Entry/Host Grant/実Provider/CLI subprocessの試験を別に足す |
| `tools/reference-generator-selection/testdata/line-amount/contract.json`と`fixtures.json` | `sales.line-amount`は**`status=reference-local`**。属性`sales.order_line.quantity`、`unit_price`、出力`std.money`はWorkspace意味に参照するが、数量×単価の**複合規則そのものはCanonical受理済みではない**。正例`2x300`は`600.00 JPY`（DECIMAL(8,2), half-up） | 生成器Referenceの結果をProductのAccepted Meaningへ**自動昇格しない**。最初のP0純粋Functionに適した既存Accepted契約を探すか、通常のOwnerレビューで小さな計算規則を受理してからEntryを公開する。後者のときも元Fixtureは独立比較入力の候補であって意味正本ではない |

### 13.3 最初の実装PRの赤→緑を、三つの責務に絞る

**PR-A0：意味と公開対象の確定（#4495／#2502／Meaning Owner）**。上記13.1の実Capability棚卸しから、**Ownerが受理したpure意味・必要Read/Effectの分類・Output型・NULL/丸め/単位・Target適合**を一件確定する。候補が`reference-local`だけならそれをIssueへ報告し、Meaningの通常レビュー/受理とEntryの明示公開を別々に確認するまで`published`へ進めない。出典Source occurrence／Contract revisionと、選択済みCompositionの明示Entry公開（非公開時call=0）を局所Testで確かめる。UI Function Catalog/静的`command`と混同しない。

**PR-A1：共通Product Operationと実Go Provider（#4495／#951／#1251／#3422）**。既存`Capability`と`capabilityinvoke`のResult modeを利用し、Hostが構成したTrusted Contextと現在Selectionから選択したGo実装へDispatchする。計算を実装する場合は既存D1の`derivationref.Multiply`と#1705の型付きD8 Referenceを再利用可能な足場として評価し、**reference-only実装のimport/テスト成功だけで採用済みOperator・製品Providerだと主張しない**。typed Inputの欠落/明示NULL/未取得を失わず検証し、実Outputは値・型・presenceをMeaningの必要条件に従って照合する。A/Bが同じ`600.00 JPY`を返しても、B選択/A実Callを外側WitnessでFAILにする。全Target共通Runner、第二Provider Registry、汎用`Result any`の無検査透過は採らない。

**PR-A2：実製品CLI二入口（#4495／#3422）**。採用済みの正式CLI起動面を`acceptedCommands`／main Dispatch／Help／既存コマンド回帰と一緒に接続。CLIがProduct Operationへ直接変換し、ReferenceExecutorや別独自計算実装を経由しないことを検証。`go build`した**別プロセスの実`tei`**のbinary/argv/exit/Result bodyと、**そのプロセス内のProvider境界**に由来する実Call Evidenceを確認する。Go API内のcounterやCLIの`provider=B`自己申告だけで入口同値を主張しない。Transport formatと契約上のResultを復号後に比較し、既存CLI JSON v0の無断変更を避ける。

この3分割は**同一#4495の小PR分割案**であって、新しい定常Gateや新規Ownerの発行ではない。PR-A0は公開対象が実際に受理されるまで未完了、PR-A1/A2は別FixtureのReference成功を合算してProduct Q2へ昇格させない。Goテスト／本番CLI未実行の段階で全工程をPASSにしない。

### 13.4 既存Fixtureを使っても誤判定できる反例

| 一因子対照 | 許される意味判定 | 現状の検査を流用する際の注意 |
| --- | --- | --- |
| `2x300` と`half-up-tie` | `600.00 JPY`、`1.01 JPY`を**受理済みの同じ規則**に基づき独立比較 | Generatorの`reference-local` Fixture自体は業務規則受理でない。単なる`600`出力一致も精度・通貨の比較を欠く |
| `missing-quantity`と`null-unit-price` | 元入力は欠落と明示NULLとして区別し、業務規則が両方を`missing`に写すか否かを受理済み契約から判定 | 既存Generator fixtureは**どちらも出力state=`missing`**なので、結果値だけでは入力での区別保持を証明できない。Typed Operationの入口で別途検証する |
| 単一Output modeだが`Payload=nil`／本文欠損 | 明示NULL可能性と未返却を混同せず、対象Output Contractで適合/拒否を決める。Result body未取得でEL-12は不合格 | 既存`ValidateBusinessResult`はOutput branchでPayload presenceを検査しない。いきなり全Capabilityへnil禁止を導入しない |
| `B selected/A invoked`、値・型・Result bodyが同一 | selected/actualの不一致でFAIL | Providerの自己申告、結果の等値、Link済みArtifact名は独立Call Witnessでない |
| 実CLIは未知Verbを正しくexit2で拒否 | 既存CLI文法の局所合格にとどめる | 新しいEntry呼出し/実ProviderのP0証拠に流用しない |

**最小受入票**：`exact HEAD / Accepted Meaning（reference-localなら未受理）/ Workspace・Source occurrence / published Entry version / typed input states / selected Provider＋実Call Site witness / actual Product CLI binary＋argv / actual Result body＋schema＋Failure / 既存Independent Expected参照 / PASS|FAIL|NOT RUN|unsupported`。実Call・Result本文と現在Host権限の実証がない項目を同じ「成功」に束ねない。秘密／Host内パスやPrincipalの詳細をPublic Receiptへ複写しない。

### 13.5 他レーンに追加するのは合流条件だけ

- **Generator #4544**：既存の`reference-local` line-amountを標準Generator比較の入力として保つ。将来同じAccepted Ruleへ接合したときだけMeaning/Output/Target/Actual Providerの一因子照合を行う。今はLの第一Hard Gateにしない。
- **DSL/Trait #988/#3278**：入力型・Output精度・欠落・実Providerの必須Methodを、表層構文や未採用Traitで自動補完しない。Target不対応はunsupported。
- **Trigger #4497**：Manual/Schedule/Event/Batchは公開Entryと必要InputをPrepared Invocationへ写すが、queued時のpreviewやActor文字列がstart時のHost Grantではない。結果本文取得と実Effectは別。
- **Studio Self-Build #2831/#4583**：全体最優先のB1→B2→B3を継続。Stage2実binaryで実Provider Dispatchを観測した時点だけ、LのFunction起動と結合成功を主張する。Build-time Generator RoleとRuntime Provider Roleを混同しない（#4601）。
- **X-TEA/SHO/SUI/AI**：Data Script/Hookの内部実行を公開Entryへ無条件昇格せず、Modeler/Composer/Canvasからの呼出しは現在Disclosure/Host Contextを経た共通Product Operationへ接続する。公開・受理・実行をAIやUI Pinだけで付与しない。

**現時点の資格**：この§13はmain`e888bb8f...`の個別Sourceと既存Reference契約を静的に照合した計画・反例の追補。PR #4618はDraft、§13による新Go Test、Product CLI、Provider/Host/DB/EKI実行、Stage2 Desktop Function起動はすべて**NOT RUN**。正式CLI構文・公開Entry schema・Canonical line-amount意味採用は**未決**。新Run Engine、Permission Issuer、第二正本を追加しない。

## 14. Studio Web/Desktop・Go WASMとの起動境界：共有するのは契約、権限と実CallはHostへ（2026-11-14）

2026-11-14のStudio Web版・Desktop版、Go WASM共通状態管理、YIC Harness/TAI Activity、Plugin/BuildIntentを巡る他レーンの設計を、現行main Sourceの実情に照合した追補。**Web/Desktopを別の業務実行実装に分けない**一方、UI race suppressionと認可・実Runを混同しない。本節は既存#4495/#3422/#4515/#4583/#2502への引渡しであり、Script Launch P0-LにWeb完成・WASM導入・全Harness統合の新Hard Gateを置かない。

### 14.1 実在するStudio境界と公開Webの安全性を区別

- `reference/tei-studio-workbench/main.ts`は共通TypeScript Workbenchを組み、`cmd/tei-studio/assets/client.mjs`の`createReadOnlyClient`というHost-neutralな呼出しPortを利用する。PortがWailsの`main.StudioBridge.*`を呼ぶという事実は、同一のProduct Operation/Current Grantが検証済みであることを意味しない。
- `cmd/tei-studio/main.go`はWails v3 `StudioBridge`を公開し、`ServerOptions.Host="127.0.0.1"`を使う。さらに`WAILS_SERVER_HOST`が未設定または**厳密に`127.0.0.1`**でなければ起動を拒否する。READMEのServer Build手順は**単一利用者・ローカル参照PoC**。一般公開Webに必要な実Session/Workspace分離・AuthN/Z・Origin/CSRF/Disclosureの証拠ではない。
- 現行`StudioBridge`と`assets/client.mjs`には`ExecuteCommand`／`ExecuteCommandWithPolicy`／`ExecuteCommandWithReferenceStore`という参照Commandの実行入口もある。Host実装`internal/studiohost/command.go`の`CommandRunValue`はReference結果で、`ReferenceAuthorizer`および`MemoryUnitOfWork`等による`persisted=false/durability=none`を明示する。**`ReadOnly`というHost型名や`createReadOnlyClient`という名前から「全methodに無作用／本番権限あり」を推論しない**。Caller指定`allowed`をHostの現在Grantに昇格させない。
- Wails v3の公式Server Buildは`server`タグのGo HTTPサーバーとしてService Bindingを使える一方、Desktop固有Window/Tray/Native Dialogは使えない。Web/Nativeで**共有可能なのはService/API契約とWorkbenchのUI Source**、固有なのはProcess/Transport/OS permission/Sessionと判断する（公式: https://v3.wails.io/guides/server-build/）。Server Buildを採る場合も既存ループバックPoCを無検証に`0.0.0.0`で公開しない。

### 14.2 共通にする契約と、共有しない実行状態

| 層／対象 | 共通にできる・既存Owner | 共通化してはいけないもの |
| --- | --- | --- |
| **Workbench UI** | Entry候補、Source/Subject Pin、入力Form、Preview・実行意思・結果/Failureの投影。TypeScriptのPort/Adapter | UI選択や`ready`、過去のPreview、AI提案によるPermission/Meaningの採用 |
| **Pure Model/Projection** | Goの小さな純粋関数の`input/contract/revision/output/failure`、同じFixture/Expected。Go/WASMへの限定コンパイル候補 | Browserに付与するHost Grant、DB/Secret、永続Store、Provider Runtime全体。Go`internal/operations`のTransport-free依存試験に`syscall/js`/Wailsを注入しない |
| **Product Application Operation** | 一件の公開Entryに対する型付きInput、Contract期待版、Source/Target、意味Outcome、Failure。#4495/#3422 | CLI/Studio/Web/WASM別々の意味判定、ReferenceExecutorを無告知Productへfallbackする経路 |
| **Host/Runtime** | 選択済みProvider・Artifact・現在Scope/Grant/Resource・同一Attemptの実Call Evidence。既存Contract/Plugin/Permission owner | JS Callerが申告した`actor/role/allowed`を権限正本にする経路、Browser local cacheのSnapshotをDispatch資格にする経路 |
| **Activity/Session/Run** | Goal/TAI ActivityとProduct Operation/Attemptの相互参照、失敗/不明/取消/観測の判定票 | Harness Session ID、Browser Tab ID、Task ACK、Transport成功をCommit/Result body/Effect観測に昇格させること |

**Go WASMは現段階では候補**。すでにHost側にある型付きOperationや`semantic/capabilityinvoke`をBrowserへ丸ごと移す義務はない。最初はSource未変更のread-only検証／projection／差分計算から、(a) Go native/Server、(b) browser Go/WASM、(c)現TypeScriptという同じ合成Fixtureを比較し、**初期ロードbyte・初回表示・warm再評価・メモリ・結果同値・未知状態・依存サイズ／ビルド再現性**を実測して採否を決める。Go公式WebAssembly案内は`GOOS=js GOARCH=wasm`とRuntime支援JSのGoバージョン整合を必要とする（https://go.dev/wiki/WebAssembly）。パフォーマンス優位を未測定の前提にしない。WebAssembly EngineやWASI外部HostをScript Launchの新しい標準Hostとして追加しない。

### 14.3 応答の世代管理は取消／再送の保証ではない

`reference/tei-studio-workbench/responses.ts`の`createResponseScope`は`context` epoch・lane sequence・disposeで**旧成功／旧失敗を描画しない**。既存`responses.workspace-switch.test.mjs`はWorkspace A→B→A・同Subjectでも旧A応答を描画しない対照を含む。**ここにはHost側Context取消、実Providerの停止、すでに生じたEffectのRollback、OperationIDの冪等制御がない**。UIが`run("inspection")`を無効化してもProductの`Run`を取消したと見なさない。

公開Entryの遷移では、共通APIが以下を区別して返す／再照会可能にすることを既存ownerに要求する。新共通`Run`テーブルを要求する意味ではない。

1. **Preview**：Version/Subject/Capability/Provider候補の検査結果。`executionPermission=false`を基本とし、現在Grant/公開状態はAdmitで再解決。
2. **Admit/Start**：Host実Contextから対象・現在Grant・選択Provider/Artifactを取得／再検査。Workspace/Entry/Source/Grant/Target driftなら実Providerへ0回呼出しの拒否。クライアントのPin/Clock/Role文字列では判断しない。
3. **Dispatched**：実Providerへ渡されたことと処理結果の確定は別。Reader/Callerがcancel・timeout・通信断になった場合、開始済みEffectがないとは断定しない。
4. **Observed/Disclosed**：Result payload、Failure、実Artifact/Provider witness、Commit/Effect（あるときだけ）を既存Observerで確認し、現在のDisclosureを再確認して表示・配信する。Task ACK、CLI exit 0、WebSocket更新だけでBusinessVerifiedにしない。
5. **Resume**：YIC/外部Harness Session/TAI Activityが再開しても、保存Goal・Task Delivery・Product Operationは別Identity。過去のPrepared Invocationや古いGrant、前セッションが作ったProposalを**無審査Dispatchしない**。protected effectが不明なら既存OwnerのReceipt照会／人間判断へ渡す（#1130/#1174）。黙ったlocal fallbackは禁止。

### 14.4 必要な圧力試験は既存Ownerに配分する（新Test SSOTを作らない）

| ID | 独立Expected（実行できたものだけPASS） | 最初のOwner |
| --- | --- | --- |
| **WCL-01** | A→B→A切替の旧Success/FailureはUI描画0。**旧Host処理が取消済みというClaimはしない** | #4515／既存responses tests |
| **WCL-02** | Preview時Grant有効→Dispatch直前失効。Go共通OperationでProvider call0。Desktop/Server/WASMのUIは拒否を同じ意味で投影 | #4495 EL-07／#3422 T08 |
| **WCL-03** | 別Session/別Workspaceの同名Entry／同じWorld FP。Remote Callerの`actor/allowed/workspaceID`で横断不可。未公開・未許可Originで実Host call0 | #3422 T08/T12／#4515 |
| **WCL-04** | StudioのReference `ExecuteCommand`が`committed=true`を返しても`persisted=false/durability=none`のまま。Product/DB writeのPASSへ昇格不可 | #1251／#4495／#1174 |
| **WCL-05** | Go/WASM側Previewが`ready`、Host側Source/Operator/Entry/Artifact版はstale。Hostは再検査し拒否、Browser cache/Previewで昇格しない | #3422／#4495／#2600 |
| **WCL-06** | 実Product CLI→Desktop IPC→Web Serverは同じContract/Owner/Providerを呼ぶ。各入口の実Transport・実Context／Call Siteを個別に採取。HTTP/BrowserやWebView未実行ならNOT RUN | #1158/#3421/#4495 |
| **WCL-07** | Dispatch後にWebSocket/Tab/外部Harness Session切断。UI suppressだけで二重再送せず、Outcome/Effect unknownは既存Receiptを照会 | #1130/#1174/#3422 |
| **WCL-08** | Stage1 BuildIntent選択Builderと実子Digest一致でも、**Stage2 Runtime Provider/Entry収録と実起動は未実証**。Generated Desktopの実Consumerから別Witnessを採る | #4583／#2831／#4495 |

WCL-01は既存UIテストの再利用を優先。WCL-02〜07は、Go直接API／実CLI→Desktop IPC→Web Serverと段階を踏み、WASM導入を前提にしない。既存#1180/#2514で独立Expectedを維持し、製品E2Eとsynthetic Referenceを混ぜない。

### 14.5 BuildIntentの進展をScript Launchへ「原則」としてのみ反映

2026-11-14時点のDraft [#4623](https://github.com/hat47x/tei-platform/pull/4623) HEAD `f1e219a25976946bcc8a19ef57f5490333ec8a84` では、`tei self-build -bind-resolved-builder`が既存`BuildIntent.Resolve`で選んだBuilder digestを、実Child executableの**起動前SHA照合**に結び、Selected B／Actual Aの負例を拒否する限定経路を実装。PR説明では当該HEADのCI 8系統成功と報告されている。ただし**Draft未merge、Plugin主体のBuildPlan、Caller供給CatalogのMeaning/Authorityは未証明**で、OS LoaderのTOCTOUとN0〜N3の実BuildIntent駆動、Stage1→Stage2 Desktopは依然未成立。

**Script Launchに移すのは、Resolve選択→実Caller/Provider→実Resultの同一Attemptでの照合パターンだけ**。BuildIntent Builder DigestをそのままFunction Entryの実行資格やHost Grantへ使い回さない。静的Go Providerには動的Loadのイベントを求めず、Artifact linked memberとHost-owned actual call witnessで独立検査する。Plugin/BuilderとFunction/Entry/Resultの実装を単一万能Runnerに統合しない。

### 14.6 段階採用・検収と現在の証拠上限

- **L0/P0 #4495**：先行§13のA0（意味受理・公開）→A1（typed Operation／実Go Provider）→A2（実CLI subprocess）。Wails/Web/WASMなしで成立可能。Host Current Grant・Effectは別資格を維持。
- **Workbench先行 #4515/#1824**：現行read-only UI/Pin/response epochに、候補Entry/Failure/Resultの**read-only Projection**を必要分だけ追加。保存・承認・実Dispatchボタンを本節だけで増やさない。
- **Desktop/Web接続 #1158/#3421/#3422**：既存Go Product Operationを共通Wails Bridge Portに接続する。Web Serverは当面loopback PoCとし、一般公開・マルチテナントは別AuthN/Z・Origin/Session分離と実Browser実行を検収してから判断。WebとDesktopのUI間で重複したビジネスロジックを置かない。
- **WASM選抜（候補）**：小さなpure Client Projectionの同一Fixture／Target／Operation shape／browser実測を比較して採否を決める。Go WASM clientがHost Runtime/Permissionを代替する案は採らない。
- **B/G/L合流 #4583/#4544**：Stage2に必要なRuntime Providerの包含・実Callを別に検収。Generatorがbuild時に必要なことを理由にStage2へ同梱しない（#4601）。B0／BuilderPin GreenでDesktop Product FunctionをPASSにしない。

**状態**：この§14はmain Source、Issue、Draft #4623と公式公開ドキュメントの静的照合による**設計・検収更新**。新しいGo Product Function/Wails Server public deployment/Go WASM同値・性能/実CLI・Stage2 Functionの実走は**NOT RUN**。本節は正式CLI/Entry JSON/Go WASMモジュール仕様、Wails Server公開認可方式、Owner Meaning Acceptanceの採用ではない。既存PR #4618はDraftのまま。

## 15. 実PostgreSQL Readを再利用した最短Product接合とCLI衝突回避（2026-11-14）

前節§13は最初の**pure** Functionについて、現Reference WorkspaceのCanonical CapabilityとGeneratorのreference-local計算契約を区別した。並行する#2586（単一Function Data Context）では、別の重要な実装が既に**main統合**済みである。ここを「まだ実Providerが存在しない」と扱うと再実装と実装順位の誤りを生む。main `e888bb8f21367b721f0984953530da9627e2634d`で現コードを再確認した。

### 15.1 成立済みのReadと、未成立の公開Entryを分ける

[merged PR #4539](https://github.com/hat47x/tei-platform/pull/4539)は`order_entry.recent_orders`で既存のBrowse Pattern→Query→Environment上の一意なDB instance→実PostgreSQL Executor→Rowset→`functioncontext.Context`→**専用参照CLI**（`tools/reference-browse-read`）を通し、使い捨てPG16で実行した。現mainの主なSourceは次の通り。

| 既存Source／契約 | 実際にあるもの | Productに追加で必要なもの |
| --- | --- | --- |
| `reference/workspace/applications/order-entry/patterns/recent-orders.tei` | `pattern ... browse { data sales.order; query order_entry.recent_orders }` | **Browse Pattern ID≠公開Function Entry ID**。Function/Entryとして公開するならOwner Meaning/Composition/Target/Accessの別判定 |
| `reference/workspace/applications/order-entry/queries/recent_orders.tei` | logical datasource `sales`、PostgreSQL native SQL、5列mapping | `:from`は既存Queryの`input`宣言にない。参照ツールの明示typed flagだけで正式Function input schemaを補完しない |
| `internal/driver/browse_query_environment.go` | Environment/Definition/Provider/Instance/Registryを照合し、欠落・不一致ならExecutor call0。Executorへ渡したQuery.Sourceのdigestを記録 | Hostが選んだ**実Connectionと申告Instanceの同一性はhost-asserted**。Executor自己申告・Query bytes digestだけでTrusted Connection/Grant・全Source Revisionを認定しない |
| `internal/functioncontext/functioncontext.go` | 1実ReadのRowset/Result Contextを一時投影。NULL・mapped column・raw duplicate・time rendering・受領に必要なUnknownを区別 | `sourceRevision=unknown`、`coverage=unestablished`、`emptyJustified=false`、結合Identity推測なし。0行/RLS不可視を「業務集合が空」と断定せず、Outputの開示と鮮度は別 |
| `tools/reference-browse-read/main.go` | `-confirm-isolated`で使い捨てDBを確認、DSNは命名された環境変数から参照、明示`name=type:value`、結果をJSONで取得する参照Consumer | **通常の`cmd/tei`製品CLIでない**。これを`internal/operations`からimportしない（`operations/dependency_test.go`は`tei/tools/`等の逆依存を禁止）。接続やCredentialをCallerの任意申告に依存させない |

**判定を二レーンに固定**：
- **L-P（pure Function Product Entry、#4495）**：§13のA0 Owner受理/明示公開→A1 Product Operation＋実Go Provider→A2通常`tei` subprocess。Generatorの`sales.line-amount`（reference-local）やD8 syntheticだけでProduct合格にしない。L-RのPG read結果でL-Pのpure証明を代替しない。
- **L-R（実Read Function Data Context、#2586）**：すでに成立したPG read／Env binding／Contextへ、必要なInput Meaning/Target/Current Grant、**通常製品Consumer**、独立Native/Studio Consumerを順次足す。DB readという外部作用はpure Functionと別。L-PのA0/A1/A2をL-Rに重複実装しない。Function Context名があっても`Query/Pattern`を未受理の公開Entryへ昇格しない。

### 15.2 L-Rに残る実装を四つの小さな接合に分ける

| 作業単位 | 既存Owner、最小の作業、完了条件 |
| --- | --- |
| **LR-A Meaning/Input** | #2586/#2506/#2387。既存Queryの`:from`の型・必須性・NULL・時点・Targetでの意味を、Owner受理済みContractへ束縛する。参照ツールの`declaredByQuery=false`をOutputから消して適合偽装しない。Source occurrence・実Source byte・Query revisionの現在性が不明ならunknown |
| **LR-B Typed Product Read** | #2586/#3422/#951。現在Host ContextでEnvironment/DB providerと実実行先を選択、実PG rowset→既存`functioncontext.From`へ流す薄いOperation Adapter。Reference CLIで使われる`tools/internal/postgresdb`に逆依存せず既存Runtime ProviderへFit。DB接続同一性がhost-assertedならその資格上限を残す |
| **LR-C 実Consumer** | #2586/#3420/#3421。通常`tei` CLI subprocessと独立Desktop Native/Studio Consumerで、同じQuery/Environment/Subjectと一実ReadのOutput/Failureを受領。**異なる読み取りAttemptはDB Snapshot/visible rowset同値を無条件に期待しない**。独立Consumerへ同じContext/同じfixed fixtureを配信した場合と、両ConsumerがそれぞれDB実Readした場合を別試験として表示 |
| **LR-D Current Authority/Disclosure** | #3422/#1174/#2516/#3421。HostでPrincipal/Workspace/Scope/Grant/Query access、Result column/row disclosureを入場時と送出時に別検証。Origin/Session/ブラウザーevent購読範囲を強制し、Session違いの応答や全接続broadcastによる漏洩を拒否。Read-only＝全行取得/全件可視/公開Web認可済みではない |

LR-A〜Dは全直列Hard Gateではない。LR-Aの受理待ち中は、LR-Bの合成Fixtureで`Environment mismatch call0`等の配線/拒否を検査可能。ただし未受理Contextと実DB read／Owner受理済み公開Entryの結果を違うHEAD/Attemptから足してQ2にしない。Stage2 Studioセルフビルドは最優先で並走させる。

### 15.3 製品CLIの改修は並行Draft #4625と同一変更面を共有する

[Draft #4625](https://github.com/hat47x/tei-platform/pull/4625)はREA Evidenceの**保存済みローカルファイル読取**を、mainの既存`operations.ExternalCandidates`へ薄くつなぐ`tei external-candidates`入口を実装中。現在の差分は`cmd/tei/main.go`、`cli_commands.go`、`cli_contract.go`と専用Go testを変更する。**Script LaunchのA2/LR-Cも同じCLI横断ファイルを変更する**ため、二つのDraftに同じHelp/Catalog/dispatch変更を別々に重複実装しない。

- 適用順は`main最新HEAD→既存PR #4625の適格性・統合状態確認→A2/LR-Cの最新基底で再設計→既存CLIのAST dispatch/help/catalog/subprocess回帰＋各入口の正負テスト`。#4625のmergeをL-P/L-Rの**技術的なHard Gateにしない**。必要なら別Branchで専用Adapter部分のみ先行し、重なるCLIファイルの統合は単一Ownerで実施する。
- #4625の`-evidence-path`は**ローカル・保存済みEvidence専用**。任意PathをRemote Studio/MCP/Webへ運べる先例にしない。L-RのDB接続・DSNもBrowserやCLIの未検証Principalが自由に指定できるProvider/Permissionへ昇格しない。REAファイル読取、PG Query読取、pure FunctionのDomain適合・対象Scopeはそれぞれ異なる。
- CLIが報告する`ok`、exit0、JSON Envelopeだけで選択済みProviderのactual entry、型付きResult body、Host Scope、DB visibility/coverageを証明しない。既存`TestAcceptedCommandsMatchDispatch`・`TestUnknownCommandRejectionIgnoresFlags`を維持し、CLI拡張に必要なHelp/JSON/Failure Code・禁じたFlagを別途検査。

### 15.4 複数Consumerの独立Expected／最初に止める負例

| 受入ID | 一因子の変更と必要判定 | Owner |
| --- | --- | --- |
| **SLR-01** | 同じQuery/Pattern/SourceでもEnvironmentのDB instance/Providerが不一致→**Executor実Call0**。Registry/申告Providerだけで実接続先が証明済みとしない | #2586 |
| **SLR-02** | `:from` typed runtime値はあるがQuery入力宣言が無い→**Function Entry input受理は未成立**。不足宣言を自動推論しない | #2586/#2506 |
| **SLR-03** | 0行／RLSで2/3件だけ可視／mask view NULL／Duplicate Row→結果を取得できても`coverage=unestablished`、`emptyJustified=false`とNullOrigin不明を保持 | #2586/#1127/#2514 |
| **SLR-04** | Product CLIの実ProcessとStudio実IPC。前者がReferenceで直接計算/Read、後者が別Providerを呼ぶ→同一Contract/Selection／Actual Callに不整合ならFAIL。Result同値だけで不可 | #4495/#3421/#1251 |
| **SLR-05** | Local-only `-evidence-path`を別Workspace/別SessionからRemote引数として送信→Hostが任意Pathを読まず拒否。内部Path/DSN/Failure messageの開示0 | #3421/#3394/#2516 |
| **SLR-06** | Web ServerでPrincipal A/Workspace Aの`Rowset/Result`通知が、B/別Tabへbroadcast配信される→秘密Payload受領0。Tab IDは現在Principal/Workspace Grantではない | #3421/#4515 |
| **SLR-07** | Source A→B→Aで同bytesの再読込や、同一Queryの別DB結果がたまたま一致→旧Host Grant/Source Revisionや別AttemptのIdentityを自動再利用しない | #2600/#2586/#3422 |

**再利用する本来の正本**：#1180/#2514の独立Expected、#1158/ADR-0048の共通Application API、#2586のFunction Data Context、#951のProvider Selection、#1174のEffect/Grant、#3421のWeb Host、#4583のStage2 Build。新Reader/SQL Engine/Runner/Provider Registry/Doc Schema/Test SSOT/認可Coreを設けない。

**証拠上限（2026-11-14）**：PR #4539はmain統合済みで既存PG16 readとReference CLIの実験を報告。#4625はDraftで通常`tei`のREAローカルReadのみ候補。新規のL-P公開Function Product CLI、L-Rの通常`tei`製品PG Read、二Consumer/Host Grant/多利用者Web/Stage2実Product Functionは**この追補ではNOT RUN**。A0のOwner Acceptance、LR-Aの入力Contract、LR-DのWeb認可・開示は未受理／未実証のまま残す。

## 16. 起動前Result契約のFail-Closed：実コード差分をP0-A1へ引き渡す（2026-11-14）

前節§13〜15はP0のA0（Meaning/Entry）とA1（型付きOperation/Provider）とA2（通常製品CLI）を並列の検証レーンへ分けた。今回、A1の中で**Ownerの受理待ちなしに変更でき、しかも潜在的なEffect前に必要な境界**を実コードへ実装した。

### 16.1 現行mainで見つかった欠落と最小修正

修正前main `e888bb8f...` の `internal/semantic/capabilityinvoke/invocation.go` は、実Adapterを先に呼び、その後で `ValidateBusinessResult` を実行する。元コードは出力契約のないCapabilityを拒否するが、**拒否前にAdapterへ到達する**。またOutputとnamed Outcomesの矛盾は、Output分岐の優先により結果検査から見逃せる。これは*不正なContractがEffectful Providerへ伝わらない*というP0の前提に反する。公開Entry/現在Grant/Business Meaningの検査をここで実装する必要はない。

[main統合済みPR #4629](https://github.com/hat47x/tei-platform/pull/4629) は既存 `capabilityinvoke.Invoke` に `validateResultContract` を**Adapter実呼出し前**へ配置し、直接 `ValidateBusinessResult` の利用でも同じ検査を行う。拒否するのは空Capability identity、Resultなし、single Output＋named Outcomesの両立、Outcome name/payload型参照欠落、duplicate outcome名。これらを一件ずつ壊したSynthetic testでAdapter call0を検証する。すでに正しいnamed Outcome／single Output／pre-dispatch cancellationは既存testで保護する。

**あえて固定しない条件**：`BusinessResult.Payload==nil`を汎用的に禁止しない（Nullable/未取得／Missingの業務意味はAccepted Type/Target/Adapter側）。Outcome中の`Payload`の値shapeやInputの型／単位・Owner意味、Entry公開、Provider選択、Host Grantを`validateResultContract`が判断したとみなさない。これらはA0/A1後段とEL-01/04/08/12に残す。

### 16.2 出来たことの合否票は「構造検査」と「製品実行」を分ける

| チェック | 新実装で主張できること | 未達のまま残ること |
| --- | --- | --- |
| **RC-01 malformed declaration** | 上記6種類の不正result contractで`Invoke`が`Adapter.Invoke`へ入らないことをGo testで判定可能 | 未公開Entryでもcall0となるEL-01の全体判定 |
| **RC-02 standalone validation** | `ValidateBusinessResult`単独でも矛盾するResult契約を拒否 | Accepted Meaning/Nullable Output/Output値構造の判定 |
| **RC-03 unaffected paths** | Authenticated/Denied、single Output、Context cancelなど既存局所契約の非退行 | 実Go selected Providerのactual linked member、Host Current Grant/Effect |
| **RC-04 build/consumer** | コードは`internal/semantic/capabilityinvoke` 2ファイルだけ。新CLI・新Provider・新Registryはなし | L-P正式Function/実`tei` CLI subprocess／Studio IPC／Web/WASM／Stage2 Product Function |

本実装は**A1内の部分実装**であってA0 Owner受理やEL-01公開Entry資格を代替せず、L-RのPostgreSQL Readや#4623/#4624のBuildIntent Validatorとも結合しない。#4625のCLIファイルとは非重複。実装・試験・統合結果は[merged #4629](https://github.com/hat47x/tei-platform/pull/4629)へ記録し、§16には到達範囲のみ反映する。

### 16.3 BuildIntentの実装と共有するのは三段の検証順序

別レーンの[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624)は、BuildIntent Resolverが宣言と違うSource/Builder/依存物を返す誤受理を、**Resolver受理段階**で拒否するGo実装修正を提出している。一方[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623)は選択Builder digestと実Child imageを**起動前**に照合する。Script Launchで同じ仕組みを新Resolver/Runnerとして取り込むのではなく、次の順序を**独立Owner間の観測様式**として共有する。

1. **構造のpreflight**（#4629）：Result契約の矛盾・欠落があればAdapter call0。これはSourceの受理やEntryの公開を意味しない。
2. **選択・現在資格**（#951/#1016/#3422）：公開Entry、Owner受理、入力版、選択Provider/Artifact、Host Current Grantを実Dispatch直前に固定・照合。B selected→A actualを拒否。#4624のBuilder契約チェックをFunction資格と混同しない。
3. **実Invoke・実Result・開示**（#4495/#1174）：実CallのProvider witness、結果Payload/NULL/型、実Effect（ある場合だけ）、独立Consumerを観測。Providerが不正Resultを返す時点で副作用が起きている可能性があり、結果検証失敗をRollbackと主張しない。

この三段は共通のAPI/Receipt/Independent Expectedの**比較視点**であって、Product Build、Function、DB Providerへ万能実行Engineを要求しない。CLIやStudioの表示だけでどれか一段を省略しない。

**統合実績（2026-11-14）**：[PR #4629](https://github.com/hat47x/tei-platform/pull/4629) のexact HEAD `00ea85234720db01a06279fd87e2c72cf1001605`で、GitHub Actions **9/9 completed/success**（[Repository Health](https://github.com/hat47x/tei-platform/actions/runs/43545545189)を含む）。変更2ファイル、追加Go回帰・既存テスト、最新mainとの競合なしを照合して **squash commit [`76b4d29a`](https://github.com/hat47x/tei-platform/commit/76b4d29a3e482f4f8729344fe423b9eaa9c7b548)でmain統合済み**。§16のRC-01〜03はその限定検査範囲でコード実装／CIの証拠があるが、RC-04の公開Entry・選択済み実Provider・Host Grant・製品CLI・二Consumerは依然NOT RUN。#4618自身はdocs-only Draftのまま変更開始時点の状態を保持する。

## 17. #4629 main統合後のProduct起動最短実装・判定境界（2026-11-14）

**現在得た不変条件**：不正なResult契約→実Adapter呼出し0はmainの`semantic/capabilityinvoke.Invoke`で成立。**必要な追加は同じ契約検査の別写しでなく、実際のProduct EntryをHostの選択Providerへ渡す欠落した辺**である。以下は既存#4495/#2586/#951/#3422/#3421/#1180への作業分配で、新しく独立したリリースGate/実行Engineを作らない。

| 優先・Owner | 実装と独立Expectedの単位 | 限界 |
| --- | --- | --- |
| **L-P0 Owner受理／公開：#4495** | Reference-localのD8/line-amountではなく、Meaning Ownerの通常受理済みpure Capabilityを選び、実際に公開したEntry/Target/Output Contract/Source Revisionの同じ版を固定。選べない間はSynthetic Fixtureで構造/Selected Providerの負例を開発する | Synthetic成功で未受理Entryをpublic化しない。Effect未宣言を自動pure判定しない |
| **L-P1 Go Product binding：#4495/#951/#3422** | 現`capabilityinvoke.Invoke`を再利用し、Hostが解決した**選択Providerと実Go Adapter呼出し**を一つのApplication Operation/Attemptに束縛。Entry不存在／無資格・選択B→実A・選択Providerなし・Input版不一致はAdapter call0。Output値/NULL/欠落はAccepted Typeで別検証 | Product Adapterに新Result Engine/Registry/Authorityを作らない。構造検査だけではProvider Identity/Grantedを証明しない |
| **L-P2 通常製品CLI：#4495/#1158** | 同じProduct Operationを通常`cmd/tei`実subprocess経由で起動し、Process・選択Provider・actual Go call witness・実Output body/Failureを対照する。CLI Help/Catalog/Dispatchは既存`acceptedCommands`等へ統合し、[#4625](https://github.com/hat47x/tei-platform/pull/4625)のREA CLIと重複しない | `exit 0`や同じ値だけで成功を認定しない。CLI/Plugin Test Fixtureからbusiness acceptanceを作らない |
| **L-R（並行）実PG Read：#2586** | merged #4539のEnv→実PG→Function Contextを再利用。`:from`のQuery input不足、Connection host-asserted、Coverage unestablishedを現況として保持し、通常製品Consumer/Grant/Disclosureへ接続 | DB read成功／0行はpure Function完了やClosed-world coverageを意味しない |
| **L-C（後続）Studio/Server/Stage2：#3421/#4583** | 同じApplication OperationをDesktop IPC/Serverで呼ぶ。実Source/Entry/Provider/Current Grant/Resultの同一Attempt比較。WebはPrincipal×Workspace×Tabで開示分離を検査。Stage2 Desktopは実Linked Provider収録・実Callの別資格 | 現Wails localhost Reference Host、旧Browser response suppression、BuildIntent Builder pinのPASSはProduct Entry結果に流用しない |

### 最初にレビュー可能にするべき反例（既存Expectedに追加）

- **EP-01**：実Providerへ到達できる合成Adapterがあっても、公開Entryが未解決→call0（#4629のRC-01とは別）。
- **EP-02**：契約・入力・Resultが全て同一でも、Selected Provider B／actual A→FAILかつB以外のCallは0。静的Go登録経路の場合は実Call witnessを独立に採る。
- **EP-03**：公開Entryは存在するがHost Current Grantが失効、Workspace/Target/Source版が違う→invoke0。Previewの成功を流用しない。
- **EP-04**：実Adapterは呼ばれたがOutput body欠落、nullable Output/NULLとMissingの区別ができない、またはResultは合格でもEffect観測が不明→`BusinessVerified`にはしない。実行後のResult検証失敗はRollbackの証拠でない。
- **EP-05**：CLIとStudioで同じSnapshotや表示値が得られても、片方はReference executor/別Provider→Product二入口資格FAIL。
- **EP-06**：Source/Build/Entryは同bytesへA→B→Aで戻っても、Owner meaning acceptance・現在Grant・Runtime loaded member・Consumer disclosureを旧資格から自動復活させない。

### 17.1 実行前宣言検査をInput/Effectへ拡張（2026-11-14）

[main統合済み #4629](https://github.com/hat47x/tei-platform/pull/4629)の`validateResultContract`はResultの構造不正をAdapter呼出し前に拒否する。一方、`semantic.Capability.Inputs`の同名重複・必須Name/Type参照欠落、`Effect`の不明Kind/対象Data欠落も、**実Adapterへ一切到達させずに検出できる宣言の不備**である。これらは意味・権限・実データへ触れずに検査可能であり、既存APIの負例を増やすのが最短。

[main統合済みPR #4636](https://github.com/hat47x/tei-platform/pull/4636)では、`capabilityinvoke.Invoke`へ`validateInvocationDeclaration`を追加し、既存Result検査＋上記Input/Effectの構造検査をProvider呼出し前にまとめる。nil Contextはpanicでなくfail-closed。新たなEntry、Provider選択、認可、Runner、Result Engineは増やさない。

独立Expectedは **IC-01** Input Name欠落、**IC-02** Type欠落、**IC-03** 同名Input、**IC-04** Unknown Effect Kind、**IC-05** Effect Data欠落、**IC-06** nil Contextの全件でGo Adapter呼出し0。既存`ValidateBusinessResult`の責務はResult Mode契約に限定したまま。型参照の非空を確認しても、現在Sourceで解決したType・単位・NULL・入力値の適合や、Output bodyを保証しない。`Effect == nil`は**安全なpureであることの証明ではなく、Effect意図未宣言**。公開Entry/Meaning Owner/Host Current Grant/Selected B→Actual A/Effect ReceiptはEP-01〜06の既存Ownerに残す。

この作業は#4495の**構造前検査の局所強化**であり、L-P0/L-P1/L-P2の実製品資格と新しいHard Gateを同一視しない。PR #4636はGoコード・回帰試験・本節の3ファイルを変更し、exact HEAD da807abc65d560134e9a6aa4c4f7b8d92d3b2fef のGitHub Actions **9/9成功**を照合して、[squash commit 18470e69](https://github.com/hat47x/tei-platform/commit/18470e69d2755b29ce55285ff847305e2e3e420d)でmainへ統合済み。これは宣言構造と呼出し0回の局所検証であり、意味・公開・実Provider／Host Grantを受理した証拠ではない。

**着手依存**：EP-01/02のSynthetic typed Adapter検証はA0のOwner意味受理と並走可能。正式公開/実GrantのPASSと合算しない。EP-05/Stage2のWebViewは実Go L-P1/P2の確定後に合流するが、TEI Studio自己再構築#4583の最優先実装をブロックしない。**現状**：この節は#4629 merged実績と他レーンの現実装を根拠とした次工程の設計。ProductのEntry/Provider/CLI/Web/Stage2 E2Eは**NOT RUN**。

### 17.2 実Query Read・BuildIntent・Wailsの検証状況をScript Launchへ接続（2026-11-14）

§17.1までに成立したのは、**構造上不正なCapabilityをGo Adapterへ渡さない**という局所的な保証である。他チャットの実装を同じHEADごとに再確認すると、Query Read、Builder選択、Wails Serverでも「宣言→実選択→実呼出し」の照合が進んでいる。ただし、検証が成立する範囲はそれぞれ異なる。

| 並行レーン | 実装・対象HEADの確認結果 | Product資格と次の不足 |
| --- | --- | --- |
| **L-P：Go Function** | [merged #4629](https://github.com/hat47x/tei-platform/pull/4629) と [merged #4636](https://github.com/hat47x/tei-platform/pull/4636)。Result、Input名/型参照、Effect種類/対象Dataの構造不備とnil Contextを呼出し前に拒否。#4636は9/9 CI成功、main 18470e69に統合 | 実入力値の型・NULL／公開Entry／Meaning Ownerのpure受理／現在Host Grant／選択Go Providerのactual call／通常製品CLIは別検収。**Effectが未宣言でもpureとは限らない** |
| **L-R0：Query Input Source** | [Draft #4626](https://github.com/hat47x/tei-platform/pull/4626) は recent_ordersの from を std.datetime と宣言。HEAD 4f238139はActions **14/17成功・3失敗**。歴史R0 FixtureはScratch側だけ元byteへ戻して過去受理の再現性を維持 | Sourceの変更は未統合。Browser→JSON APIでsqlgen symbol混入の検査失敗、Repository Health、セルフビルド失敗を残し、既存E1 Ownerに分離。旧Sourceの資格を新Sourceへ継承しない |
| **L-R1：実Executor前の型検査** | [積上げDraft #4631](https://github.com/hat47x/tei-platform/pull/4631) は#4626をbaseとし、宣言済みstd.datetimeにGo time.Timeを要求。missing/extra/NULL/不正型を実Executor呼出し前に拒否。HEAD 02af8339の限定Actionsは**2/2成功** | #4626の先行検収が必要。2件の成功はPG16実Read、通常tei CLI、二Consumer／現在Grantの成功ではない。未宣言Queryを動かす旧Reference経路はtyped Productとして受理しない |
| **Builder実選択** | [Draft #4634](https://github.com/hat47x/tei-platform/pull/4634) は宣言とResolverの適合、選択Builderと実Child Digestの検査を同一main基点HEAD 653d0d9aに集約。Actions **8/8成功** | まだDraft。非Plugin Desktop BuildExecution／ArtifactSet、N0〜N3の実消費、生成Stage1→Stage2のWindows WebView2は未実証 |
| **Wails公開面** | [Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) は実Service登録と15個の公開Go MethodをAST/reflectで棚卸し。HEAD b9cf5447のActions **5/5成功** | Commandや外部Childの入口も含む。127.0.0.1限定Reference Serverを多利用者Webに公開してよい証拠ではなく、Principal／Workspace／Tabごとの認可・結果開示は未検証 |

この対応から、次の三原則を既存Owner間の引渡し条件にする。

1. **構造検査の再利用は、意味受理の再利用ではない**。L-PのCapability宣言拒否とL-RのQuery実引数拒否は、同じ失敗を防ぐ考え方を持つが、Source・Input・Provider/Executorが異なる。片方の実行結果を他方の合格票へ加算しない。
2. **選択と実使用を同じAttemptで観測する**。Go Functionでは公開Entry／現在Grant／selected Providerとactual Adapterを、QueryではQuery byte／typed Params／実DBとRowsetを、BuildではBuilder／Child image／生成Artifactを別々に照合する。新しい共通Runner／Resolver／Registryは設けない。
3. **Consumerの受領と開示を独立させる**。通常のtei CLI、Studio、Web/IPCの受領を段階的に検収する。UIで古い応答を隠す操作はHost側の取消・Rollbackではなく、Wailsのallowed配列やPinも現在Grantではない。

**実装順への反映**：#4495はOwner受理済みpure Entryを見つけたうえで、L-P1の実選択Go Adapter→実Call、L-P2通常CLIへ進む。未受理中も合成fixtureでSelected B／Actual A・未公開Entryの呼出し0を検証可能。#2586は#4626のSource変更／歴史Fixture／E1失敗を解消してから#4631を再検収し、実PG16と独立Consumerへ進む。#4583のStudio Desktopセルフビルド最優先は維持し、#4634のBuilder限定成功をStage2の成功と見なさない。

**この節の証拠上限**：#4629/#4636はmain実装済み。#4626/#4631/#4634/#4627は指定HEADでのCI結果があってもDraft・未統合であり、公開Function Product Q2／実PG二入口／多利用者Web／Windows Stage2の完成を示さない。
