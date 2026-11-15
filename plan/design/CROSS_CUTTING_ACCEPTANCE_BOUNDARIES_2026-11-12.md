# TEI横断統合：保証範囲・失効・受渡しの検収記録（2026-11-12）

> 状態：既存設計の整理と受入計画。新たなCore Schema／Gate／Runner／Host／Resolver／Registry／Authority／テスト正本は採用しない。実Provider・実Host・実Consumerの新規E2EとOwner意味受理は **NOT RUN**。

## 1. 統合で維持する区別

TEIのCore最小化、Everything is Data、Everything is Plugin、専門家可読のDSL、Go単一バイナリを基本とし、外部Host向けJava/.NET/WASM等は既存のTarget契約から実現する。SQLはSourceとして許容し、Data Operations／明示Function Data Contextで通常Queryを表す。Native SQLを無条件排除したり、構文のGo親和性から人間のAuthoringを決めたりしない。

**三要素分析法**は渡辺幸三氏の設計手法として、データ構成・機能構成・業務構成、および**データモデル・機能モデル・業務モデル**を区別して扱う。TEIのGraph／CRUD表／Model Mapは、X-TEA Modelerのモデルと相互参照するための拡張Projection候補であり、図面やCRUD表だけを業務意味の第二正本にしない。旧計画の便宜的な「三視点」表記を手法の正式名称としては使用しない。X-TEA由来の図面・関係・粒度・参照位置とTEI拡張部分のFit/Gapは #1124 / [Draft #3263](https://github.com/hat47x/tei-platform/pull/3263) / #2514 に従って個別検証する。参考：[渡辺幸三氏の三要素分析法の説明](https://atmarkit.itmedia.co.jp/fbiz/cbuild/complete/doa/01/02.html)。

## 2. 同じものとみなさない境界

| 境界 | 受渡しに必要な実証 | 典型的な誤昇格 | 既存のOwner |
| --- | --- | --- | --- |
| 外部原文→管理Source→受理意味 | 原文byte/位置・Root・Parser actual・Candidate/Owner acceptanceの区別 | 原文の存在やMarkdown保存を実Load・意味受理とみなす | #2621/#3262/#2620/#1126 |
| Function/Query宣言→Plugin実選択/Load | Source revision、選択Lock、Artifact member、actual Provider/SQL bytes/args | Bを選択しAを呼んでも結果が同値ならPASS | #951/#1251/#2586/#4495 |
| Provider実Read→Context/Model投影 | actual Rowset、Presence/NULL/missing、grain/identity、Scope/coverage、現時点の開示条件 | 0件、partial、JOIN 6行を不存在・完全性・Entity 6件とみなす | #2586/#2506/#2497/#2516 |
| Model/SHO/GUI固定対象→AI/他Consumer | 選択対象ID/occurrence・revision・用途・受信者・実受領・競合処理 | focusやUI表示成功を、固定対象の再解決・権限・受領済みとみなす | #1124/#1158/#2516、Draft #4479、SHO Owner |
| 公開Entry/Trigger→Invocation→Effect | Entry/Artifact/Provider現在性、Host trusted context、実Attempt、Receipt/独立Read | Trigger発火・ACK・exit 0をCommit/効果/再送安全へ昇格 | #4495/#4497/#4502/#1174 |
| 一試行の合格→変更後の再利用/BOM | 依存閉包・実入力・Toolchain/Target/Consumer/Grant/目的と適格性 | Source hash一致、別試行の部分PASSを全資格へ合算 | #2600/#2514/#1180/#3267/#2831 |

「同一ID」だけでは同一対象・同一出典・同一時点・同一資格ではない。各境界の検査では、既存Scenario/Independent Expectedに **Source location/occurrence、World/Meaning revision、actual inputs、SelectionとInvokedの実体、Host/Grant、scope/grain、受信したConsumer、Attempt/Effect Receipt、観測者、exact HEAD、未探索依存**を必要な範囲だけ対応させる。これは検収票の観点であり新しい公開データ型ではない。

## 3. 選択的再検証の原則（担当#3の主張を保持）

変更イベントから「既知の影響」と「未知の依存」を別々に求め、**失効する保証だけを再検査**する。ただし未知依存・Source解決失敗・未資格の外部Providerについて、安全な試験省略を認めない。不要な全体再Buildを避けることと、未確認なのにPASSを返さないことを同時に満たす。

| 対照 | 期待される判別／停止 |
| --- | --- |
| 同Source byte・同Meaning IDでPlugin実装／推移依存の実byteだけ変更 | 既存受理意味の履歴は保持可能でも、Build/実Invoke資格を別途失効・再検証 |
| A→B→A、Source移動、同名別Root、外部Manifest改版 | 同hashへの復帰や同名で現在の読込実体・依存を証明しない。actual Load/Source位置を再照合 |
| 新しいSQL/Provider候補だけ追加、元SQL byte不変 | 「存在しない候補」への依存が変わる。Query選択と実SQL/Providerの再検証 |
| Row Scope/RLS、Grant、用途/Recipient、Policy epochだけ変更 | 古い実Rowset/Consumer開示の資格を再利用しない。意味定義・Artifactの歴史記録を不必要に消さない |
| 無関係なPlugin追加／layout座標だけ変更 | 選択・依存・生成入力に非影響と示せる部分を再利用候補とし、全体を自動失効させない |
| Commit後Receipt喪失、ACK loss、Consumer body不達 | 成功/失敗の二値化、無条件の再送、ログ送出成功から業務結果の確定を禁止 |

再利用は **Artifact reuse / Build skip / Evidence reuse / 現在の実行・開示資格**を分離する。再検証範囲の正しさは#2600/#2514、実Buildセルフビルドは#2831、exact tupleのBOMは#3267が所有する。独立Expectedや実Product E2Eが無ければ、unknown/NOT RUNのままとする。

## 4. 次の実装順（既存Critical Path #1119を上書きしない）

1. **P0-R Product Read**：一つのWorkspaceと一つの`order_entry.recent_orders`を固定。現行mainの`reference/workspace`と別`examples/xtea-lowcode`は同名QueryでもSQL/Parameter/Fixtureが異なるため混合しない。宣言→実SQL/args→実PostgreSQL Provider→Rowset→Function Data Context→一Consumerを同一Attemptで検証（#2586）。B-OtherSQL・B-Presence/Partialを負例とする。
2. **並列P0-T/P0-H**：実製品CLI/StudioのSelected=B→actual Invoked=B（#1251）と、外部HarnessのTool出力・受信/許可Sinkまでの実受領（#2509/#2515）を別個の資格として確認。Referenceや合成CI成功を製品合格へ移さない。
3. **P1 Model/Authoring**：データモデル・機能モデル・業務モデルのSource逆参照とCRUD、SHO Draft/正式保存・再読取、固定対象Selectionの失効・ABA・人的編集競合を同じ改訂で対照（#1124/#2621/#4501、Draft #4479）。X-TEA Modelerとの一致・TEI独自拡張を列ごとに判別する。
4. **P1 Launch Read-only**：同じ受理済みFunctionのDirect API／CLI二入口、Entry公開とactual Invoke、一つのpure OperationでEL-01〜12を検証。Schedule/Event/BatchのTR試験は独立後続（#4495/#4497/#4502）。Trigger/Graphの正式GeneratorをP0-RのHard Dependencyにしない。
5. **後続P2 Effect/Composition**：protected書込を実Host Grant、Operation ID、Commit／Recorder故障、独立Read/Receiptで検証（#1174）。Provider 2版×Consumer 2版の4 exact tupleを各実Build/Invokeで試験（#3267）。REA、SUI、SEI、CSW、KON、SHO、B2C、組織C1の個別Consumer適格は別票。実Twin/大規模Ontology、全Lifecycle Graph Generator、全体G4をこの段の局所成功から推論しない。

## 5. 反証と撤回条件

- **正本の増殖**：diagram/matrix/SHO/外部Evidence/AI候補がTEIのAccepted Meaningや既存Owner Acceptanceを直接置換する場合は撤回。
- **権限の混線**：Selected・Loaded・Exposed・Authenticated・Authorized・Effect-observedを同一boolとする設計は撤回。
- **根拠の合成**：別Run／別Workspace／別Function／別ProviderのPASSをつないでE2E合格とする設計は撤回。
- **過剰な共通化**：新Host/Registry/Resolver/Graph Compiler/Policy/Workflow/Schedulerを、既存Ownerで説明可能な課題の必須条件にする場合は差戻し。
- **可読性の後退**：Goの構文都合やAIツール都合で専門家の認知順、三要素分析法の原典の関係・粒度・表記を失わせる場合は差戻し。
- **未知の隠蔽**：unknown/unresolved/partial/not-observedや未探索依存をPASS・ゼロ件・差分なしとして表現した場合は失格。

この文書は既存#1119（実施順）、#2586（初回実Read）、#1180/#2514（独立検証基準）および各Owner Issueへの**判断・反証用参照**である。正式な採否と製品受理は各Ownerの証拠に戻す。


## 6. 統合受入の最小Witness Pack（今回の具体化）

独立Expectedは既存#1180/#2514で管理し、この節は実装のための**ケース割当表**のみとする。各ケースは`Expected ID / Source identity+revision / selected / actual / observer / output+completeness / verdict / evidence link`を記録し、fixtureを実装出力から逆生成しない。CLIの文字列表現の同一性ではなく、型付きResultと実際のEffect/Scope/Failureを比較する。

| Witness | 対象の最小差分 | 独立Expected・停止条件 | Owner/実装順 |
| --- | --- | --- | --- |
| AW-01 Read identity | `reference/workspace`の`order_entry.recent_orders`だけを固定し、別`examples/xtea-lowcode`の同名Functionと対照 | Query宣言と`:from` Binder・SQL/args・実PG Seed・Rowsが同一Sourceに属すこと。別Workspaceの3/2/0件を流用したらFAIL | #2586 P0 |
| AW-02 Plugin selection | Bを選択してAを呼び、結果bytesは同じ | 実Load/Invoke Provider identity不一致でFAIL。静的Inspectorの合格は実Product選択の代用不可 | #1251/#951 P0 |
| AW-03 Presence/coverage | SQL投影の列欠落、格納NULL、未取得、RLS部分集合、取得0件 | Field Presence・値NULL・未観測・0件を区別。Count/Graph/CRUD/SHOへの誤伝播を拒否 | #2586/#2506/#2497 P0/P1 |
| AW-04 Three-element fidelity | 同label別ID・多重有向Relation・複合Key・異時点をデータ/機能/業務モデルとCRUDへ投影 | X-TEA Modeler由来の構成とTEI拡張を分け、Source ID/Relation多重性/時点/Source locatorを保持。非可逆表示からの無審査書戻し禁止 | #1124/#1126/#2497 P1 |
| AW-05 Pin & AI edit | Graph EdgeとSHO occurrenceをPin→Focus移動→改版/ABA→旧AI Patch | Pin保持と現在Source/Grant/Owner受理を分離。旧Patch拒否、別対象誤更新なし。作者によるSHO編集を無言で上書きしない | #1824/#1158、Draft #4479、SHO P1 |
| AW-06 Entry/Trigger | 同じpure FunctionをDirect API/製品CLIから呼び、Entry非公開・未選択・Grant失効を注入 | 同じ意味型付きOutcome、実Provider照合。Trigger保存/発火は実Invoke/権限ではない。未対応Targetはunsupported | #4495/#4497/#4502 P1 |
| AW-07 Effect uncertainty | protected書込のACK/Receipt loss、timeout、同Operation ID再送 | committed/unknown/not-recordedを区別。実Host/DB独立ReadなしにCommit主張不可、unknownのblind retry不可 | #1174/#3422 P2 |
| AW-08 Reuse qualification | Plugin Bだけ改版、Source A→B→A、Grant/Recipientだけ変更、無関係C追加 | Artifact/Build/Evidence/現在の開示資格を別々に失効。無関係な変更で不必要に全再Buildしない。未知依存をPASSにしない | #2600/#3267 P1/P2 |
| AW-09 Organization C1 | Membership異動、Case責任移管、受信ACK、二重Path、Private Fieldを別操作 | 所属≠責任≠Grant≠Handoff。公開Sliceが同じ二世界でCount/Tooltip/ログからPrivate差が漏れない | #2386/#4501/#2516 P1/P2 |
| AW-10 External Harness | 実Tool stdout/stderr/argv→外部受領とScope/Recipientを照合 | Tool起動/送出/受領/意味採用を別結果にする。外部Harnessの候補ContextをHost trusted contextに昇格させない | #2509/#2515/#2516 P0-H |

### 6.1 実行票の判定語彙

- `PASS`：指定した実機構・Source・Scope・独立Expectedを満たした。
- `FAIL`：独立Expectedに対する不一致または禁止された副作用を観測した。
- `NOT RUN`：その経路・Provider・Target・実Consumerをまだ実行していない。
- `UNSUPPORTED`：契約上の対象外。静かなfallbackをしない。
- `UNKNOWN`：試行済みだが必要な実Observationが欠落し、真偽を確定できない。
- `STALE`：対象版・実体・Policy・Grant・用途等の資格が失効し、現在の保証に利用できない。

`PASS`には`Claim scope`（Source-static／synthetic／real Provider／Product consumer／Effect／Owner accepted等）を必須で併記し、**部分的なPASSの和集合を全体E2E PASSにしない**。この語彙は既存Conformance票への記入指針であり新Status Core型ではない。

## 7. 変化を伝播するが、正本を増やさない実装境界

```text
Authoring Source / external Observation
  → Actual Parser/Loader + Source Locator
  → Structural Candidate → Owner Accepted Meaning
  → Function/Query/Rule/Model Graph
  → selected Plugin + actual Artifact/Provider
  → typed Application Operation
  → Host-authorized Invocation
  → actual Read / Effect Observation
  → Consumer Projection / Disclosure
```

これは**論理的な照合順**であり、全ての処理を必ず通る単一Pipelineや新しい中央Graph Compilerを要求しない。各矢印には既存のOwnerと失効条件があり、GUI/SHO/AI/Harnessは必要な節だけをConsumerとして使用できる。read-onlyはEffect Hostの完成を待たない。書込・保護データでは実Host Grant/Policyがない限り拒否する。

**統合するのは「何を同じと判定したか」のEvidence関係であり、Sourceや意味の正本、操作権限ではない。** 同じ型付きData構造で表現できることと、同じ業務意味としてOwnerが受理することは別である。Graph座標変更は意味変更と区別し、Relation/Key/Rule/Targetの変更は見た目が同じでも依存・保証の再評価を必要とする。

## 8. 追加の敵対的レビューと順序判断

1. **最初の一往復の正しさを優先**：#2586の実Readは、一つのWorkspace・一つのQuery・一つのProvider・一つのConsumerに絞る。Org C1/REA/SHO/KONへの同時拡張はしない。
2. **実呼出しの不一致を検出する**：#1251のProduct Inspectorでselected Bとactual Aの偽PASSを潰す。Graph Lifecycle全経路の完成を初回S2の前提にしない。
3. **Sourceと外部出力の真正性を別検査**：同じSource位置でも外部Provider raw byteやArtifactが違う場合、旧Evidenceを再利用しない。実Toolが送った内容と外部Harnessが受け取った内容を独立に確認する。
4. **三要素分析法の忠実性をモデル単位で確認**：データモデル・機能モデル・業務モデルの構造、CRUDの対応、Source逆参照を先に検証。TEI独自の時点・多重Relation・Graph/SHO投影を原典と混同しない。X-TEA Modelerの原典資料・実出力が未取得の項目は`unverified fidelity`とする。
5. **変更時にどの保証を失うか測る**：Source byte、Plugin Artifact、Rule/Policy、Consumer用途、受信者のそれぞれを単独で変更し、誤PASS・過剰失効の両方を計測する。速さは正しさの確認後に評価する。
6. **Productへの昇格には実経路を要求**：静的Goテスト、Reference、Synthetic、別OperationのPG成功、docs-only PRを、実Studio/CLI/Host/Owner意味受理の証拠に転用しない。

### 8.1 実装担当への引渡し単位

各Ownerは新しい全体フレームワークを待たず、担当AWの**一つの正例と一つの最も危険な負例**を既存テストDriverへ追加する。最初に独立Expectedとexact HEAD/Source/実呼出し点を固定し、最小deltaを実装、実出力をReadbackする。失敗時は「最初に欠けた辺」とOwnerを一件に絞って戻す。全体計画の優先順は#1119、独立Expectedは#1180/#2514、実Provider Readは#2586に従う。

この追補のAW-01〜10は**設計上の検収候補**であり、各テストの実装・実行・CI合格を意味しない。

## 9. 保証の合成と失効の最小契約（11/13追補）

AWの合否を並べるだけでは製品機能の保証にならない。既存Conformance票・Build Receipt・実行Receiptに共通する**照合規則**として扱い、新しい中央Registryや実行基盤は設けない。

### 9.1 Evidenceを合成できる条件

1. **対象一致**：Workspace root、Source locator/occurrence、Function/Operation ID、Target、Model revisionを対応付ける。同名だけでは不可。
2. **実行系列一致**：selected/loaded/invoked Provider、Artifact digest、引数、Host、Attempt/Operation identityを照合する。別Attemptをつなぐには独立した同値性の証拠が必要。
3. **観測範囲一致**：Rowsetのgrain・coverage・Presence、Consumer用途・Recipient・Grant/Policy時点を照合する。部分集合を全体へ昇格しない。
4. **Claim種別一致**：構文解析、Meaning受理、実Read、実Consumer受領、Effect確認は相互代替しない。欠けた境界はNOT RUNまたはUNKNOWN。
5. **現在資格**：Source移動、Plugin再選択、Grant変更、外部依存の不明化を再評価する。履歴のPASSを保持しつつ現在資格をSTALEにできる。

合成は**必要な境界だけ**に適用する。pure ReadにEffect Receiptを要求しない。protected WriteではACKやHTTP 200をDB Commitの代用にしない。意味の受理は通常の開発手順に従い、AI候補から自動付与しない。

### 9.2 失効・再利用の判断表

| 単独変更 | 保持できる可能性があるもの | 再確認が必要なもの | 禁止する短絡 |
| --- | --- | --- | --- |
| Layout/Graph座標だけ | Source/Meaning・型付きRead | 実UI対象解決と表示を主張するConsumer | 全Buildを無条件に失効 |
| Source移動・同名別Root・A→B→A | 過去の受理履歴 | 現在locator、参照先、依存閉包 | 同じbytesなら同じSource |
| Plugin実体/選択Lock変更 | 意味受理・無関係Artifact | actual Loaded/Invoked、生成物、Provider結果 | selected文字列だけでInvokeを保証 |
| SQL投影/Scope/RLS変更 | 無関係Function・意味定義 | Query、実Rowset、coverage、開示 | 0件を不存在と断定 |
| Grant/Recipient/用途変更 | 過去のReceiptと業務結果の履歴 | 現在の実行・開示・受領資格 | 古いPASSを現在の許可に流用 |
| Generator/Toolchain変更 | Sourceと意味の受理 | Artifact・Build/Target適格 | Source hashで生成物同一を保証 |
| ACK/Receipt喪失 | 送信試行の事実 | Host/DB独立観測、Operation ID | unknownを失敗としてblind retry |

依存閉包が未探索なら「影響なし」と判定しない。他方、既存Index/Build Graphで非依存を証明できる領域は不要に失効させない。検証省略の理由もEvidenceに残す（#2600/#3267）。

### 9.3 Ownerへの最小引渡し

- **#2586**：AW-01/03の独立Expectedを固定し、reference/workspaceに限定した実PG Read→型付きContext→一Consumerを同一Attemptで照合。
- **#1251**：AW-02を実CLI/Studioで検査。選択B／実呼出しAをFAILとし、実Loadの観測点がなければPASSにしない。
- **#2509/#2515**：AW-10のTool送信と外部受信を別観測で結ぶ。受領と意味受理を区別。
- **#1124・SHO担当**：AW-04/05のSource occurrence・固定対象・人手編集競合を検証し、見た目から書戻し対象を推測しない。
- **#2600/#3267**：AW-08で無関係変更の再利用と、実体変更時の失効を対で測る。

完了条件は担当Issueに**独立Expected、exact HEAD、実経路、正例・負例、判定、未検証範囲**を記録したこと。計画追記やDraft PR、静的Inspectorだけでは完了としない。#1119のCritical Pathと各Ownerの責務は変更しない。
