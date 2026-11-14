# TEI横断統合：保証の連結・失効とセルフビルドの受入境界（2026-11-15）

> 状態：2026-11-14のmainへ取り込まれた成果を踏まえた受入計画。限定的な実PostgreSQL Read（#4539）とMarkdown Graphの固定Fixture（#4709）は既存実績として扱うが、通常製品CLI二Consumer・正式Markdown Reader・意味からの実コード生成・Studio完全セルフビルドの新規E2EとOwner受理を本書で実施したわけではない。新Core Schema／Gate／Runner／Host／Resolver／Registry／Authority／テスト正本は設けない。

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

## 4. 2026-11-14の実績を踏まえた実装順（#1119と製品完成計画を優先）

**実績と未達を混同しない**。#4539のmain統合はEnvironment束縛→実PG Read→Rowset→限定Function Data Context→Reference CLIを確認した成果。#4709のmain統合はMarkdown表＋Mermaidの固定Sourceと独立Expected**候補**の整備であり、正式ReaderやGo生成の成功ではない。#4583のN0実子ProcessとWindows WebView2の局所CIは重要な実績だが、Stage1の実生成StudioがStage2 Desktop全体を再構築した保証とは区別する。

1. **全体P0・Studio実セルフビルド #2831/#4583**：非Plugin Studio Build Subjectを一つのBuild Intentへ接続し、Stage1の実Process image→Builder/選択Toolchain→Stage2実Desktop artifact→Windows WebView2受領を同じ実行系列で検証。別PRの局所PASSの足し算、固定Goのdelegate、古いAssetを拒否。既存N0/B1-A実績を失わず未閉路だけ補う。
2. **並行P0・通常製品入口の実Read #2586/#1251**：#4539のRA-0/1/2R相当の限定Reference結果を基線とし、RA-2Pとして通常tei→selectedかつactual Provider→実DB→SQL/型付き引数→Function Data Context、RA-3として異質な二Consumerの同一Context受領へ進む。reference/workspaceとexamples/xtea-lowcodeの同名order_entry.recent_ordersを混ぜない。NULL/0件/partial/異なるgrainは各票で保持する。
3. **並行の小実証・CLI先行の意味ある自己生成 #2621/#951/#2831**：testdata/tei-md-graphのMarkdown表を正式Source Roleとして読込み、Graph/Ruleの向き変更を実Go判定ロジックと通常CLI結果へ反映させる。生成Pluginを選択・実呼出しし、次の生成にも再使用する。正式Readerが未接続なら明示的にunsupportedとし、固定Go委譲は自己生成に数えない。
4. **P1・Model/SHO/AIと公開文書 #1124/#2621/#4679**：三要素分析法のデータ・機能・業務モデル、CRUD、Source逆参照を原典とTEI拡張に分ける。Pin/occurrence/Grant/受領の境界、Pluginの公開Contract/Manualの階層読取をread-onlyで試す。文書閲覧権は実行・再公開許可ではない。
5. **P1/P2・Launch/Effect/Selective Reuse #4495/#1174/#2600/#3267**：同一pure FunctionのCLI/API二入口を照合してから、書込のHost Grant/Commit/独立Readを別受入とする。Plugin版・Source A→B→A・Policyの変更に対して失効と非依存再利用を実測。Graph Lifecycle全面実装や新RuntimeはP0の前提にしない。

**優先順位の正本**はplan/workstreams/PRODUCT_COMPLETION_PLAN.mdと#1119。この文書は横断の保証判定を引き受けるだけで、Ownerや製品ロードマップを上書きしない。
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

1. **既存実Readの先を正しく閉じる**：#4539の限定実PG→Reference CLIの達成を保持し、#2586では通常製品tei→実Provider→同一Context→異質な二Consumerへ進む。別Workspaceや別Attemptを継ぎ足さない。Org C1/REA/SHO/KONへの同時拡張はしない。
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

- **#2831/#4583**：Stage1実Process image・実Builder・Toolchain・required artifacts・Stage2 Desktop・Windows実受領を一系列で照合。局所N0／生成Inspector／Workbenchの成功をStudio全体成功と呼ばない。
- **#2586/#1251**：#4539限定Reference Readを再使用する範囲を明示し、通常製品teiでactual Providerを呼び、異質な二Consumerへ同一Contextを渡す。selected B→called Aの負例をFAILにする。
- **#2621/#951**：#4709のSourceとExpected候補を正式Role→意味→生成Go→CLI実動作へ接続。正式Expectedの採否は#1180/#2514が決める。
- **#2509/#2515・#1124・SHO担当**：Tool送信と外部受信、Source occurrenceとPin、人手編集の競合をそれぞれ独立に確認する。
- **#2600/#3267**：非依存変更時の再利用と実体変更時の失効を対にして測る。

完了条件は担当Issueに**独立Expected、exact HEAD、実経路、正例・負例、判定、未検証範囲**を記録したこと。計画追記やDraft PR、静的Inspectorだけでは完了としない。#1119のCritical Pathと各Ownerの責務は変更しない。

## 10. 設計資産を実装へ変える最初の縦断試験

同じNode labelを持つ別IDの二Nodeと、別IDの二有向Edgeを持つtestdata/tei-md-graphの五Sourceを使う。Markdown表はAuthoring上の定義、Mermaidは導出された表示とし、どちらが正本か曖昧な場合は意味を推測しない。独立Expected候補（proposal-only）は受入Owner #1180/#2514が採用するまで正式なExpectedではない。

| 対照 | 期待する意味・生成差 | 現在確認済みの上限／次に閉じる境界 |
| --- | --- | --- |
| baseline | n-1の出辺e-1/e-2、n-2の出辺なし | Sourceと候補Expectedはmain。正式Role→Parser→生成Go→通常CLIは未接続 |
| edge-reversed | n-1の出辺e-1、n-2の出辺e-2 | 向き差を生成Goの分岐・実CLI結果に反映することが次の受入 |
| layout-only | 意味と生成された判定結果はbaselineと同じ | Raw Sourceの版・位置は追うが、非依存の意味Buildを省略できるか別途実測 |
| duplicate-edge-id | 同じEdge IDの複数定義を拒否 | Reader/Validatorの実診断と元Markdown位置へ結び付け、黙って統合しない |
| diagram-stale | Markdown表の意味を保持し、古いMermaid表示を報告 | 見た目からSourceへ逆流する無断自動修正や意味昇格を拒否 |

初回のReaderは、**宣言領域の抽出→既存Parserの接続→元Markdownのbyte span/行位置への診断→Loader/Source Indexへの結合**を狭く実装する。Reader未接続時のTEI_MD_PARSER_UNBOUNDは成功扱いにせず、拒否から実装済みRoleのみに進む。何でも自動解釈する汎用Markdownコンパイラを作らない。

受入系列は、①実Source/Role/locator、②意味Graph/Ruleと独立Expected、③実生成されたGoの判定処理、④選択・実呼出ししたPluginとCLI、⑤変更前後の結果、⑥Stage1→Stage2への実再使用、と分ける。**③のGoが固定結果の転記や既存Go実装へのdelegateだけなら、TEI自身による業務ロジック生成の証拠にしない**。

## 11. セルフビルドの貢献度と定義簡素化を測る

「TEIで再構築できた」の意味を一つの比率に畳まない。最初に対象機能と義務の分母を固定し、以下を別々に測定する。

| 指標 | 測定対象 | 水増しを避ける条件 |
| --- | --- | --- |
| 意味ロジック生成率 | 固定したGraph/Ruleの独立義務のうち、TEI Sourceを変更すれば生成コードの動作が変わる義務 | 固定Go delegate・期待値埋込・生成しない文書は算入しない |
| Plugin自己構築率 | 対象Pluginの機能単位のうちTEI定義から生成・Build・実Call・再生成を確認できた単位 | manifestだけの生成や外部Goの呼出しは別扱い |
| Build自己制御の範囲 | Stage1実Processが実際に選択・起動・検証したStage2のrequired artifacts | 親が未起動で外部CIだけがビルドした成果物を含めない |
| 定義簡素化 | 同じ独立義務に対するDomain DSL定義量、手書き補助コード、重複、変更箇所、レビュー/修正負荷 | 単なる短縮や巨大Plugin内部への複雑性移動では改善としない |

最初はGraph/Rule→生成Go、次に**そのGenerator/Validator Plugin自身**の定義をTEI化し、次世代で前世代の生成Pluginを実利用する。Reader・生成・Plugin・CLI・Studioの成熟を一つの大きな完了フラグにしない。DSLの専門家可読性を保ち、Goのstruct/func/型引数を業務DSLへ押し出さない。

## 12. CLI・Studio・MCP・AIハーネスの対象伝達

共通化するのは対象の**意味と操作契約**であり、画面の焦点、Harnessセッション、実行権限ではない。利用者がCLI/StudioでNode・Edge・Function・Source範囲をPinした場合、Instruction（依頼内容）、Subject（対象ID/locator/Model revision/Source occurrence）、Capability（利用可能な実Providerと用途別Grant）の三つを別に照合する。AIへ渡せたこと、外部Harnessが実受領したこと、AI候補がOwnerに受理されたこと、実Hostが認可したことは別の観測点。

GUIのfocus移動、同名Node、A→B→A、Plugin版更新、Grant失効、SHO本文の人手変更を負例とする。Versionが合ってもSource occurrenceや現在の許可が違えば旧Patchは適用しない。探索用Plugin文書は公開範囲に限る。Product Skillは入口案内、詳細はPluginが所有するMarkdown/Contract/Manualを必要時に参照し、新Docs Registryや第二のAuthorizationを作らない。

## 13. 受入と差戻し

本書の設計更新によって追加された製品試験は**NOT RUN**。mainで確認できる既存の限定実Read・Source Fixture・局所Windows CIの証拠は、それぞれの元PR/Issueの対象HEADと範囲に限って引用する。Stage1→Stage2の実Studio Desktop全体、正式*.tei.md Reader、Graph/Ruleからの実Go生成、Plugin自身の再帰生成、通常tei二Consumer、外部Harnessのactual受領、Owner意味受理を一括してPASSにしてはならない。

次の実装PRでは、一つの正例と最も危険な負例を**同じ入力系・同じHEAD**で示す。検証が欠けた辺のOwnerへ局所的に戻し、追加の汎用Framework・独立Expected正本・Second Hostを要求しない。

