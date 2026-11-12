# Script・Function起動の製品検証：最初の一機能と直接テスト

- **状態**：実装前設計・作業引渡し。正式なDSL／CLI ABI／API Schemaを新たに採用するものではない。製品実行、実Provider、OS副作用、実DB、外部Consumerの新規試験は未実施。
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
