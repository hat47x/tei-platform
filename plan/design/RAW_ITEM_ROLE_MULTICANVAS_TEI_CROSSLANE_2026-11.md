# Raw Item・多重Role・複数CanvasをTEIに接続する受入設計（2026-11-08）

**状態**：採用前の横断設計・既存Ownerへの試験引継ぎ。新しいTEI Core型、標準Item Schema、Task Engine、Canvas Store、Role Resolver、Plugin Host、第二意味正本の決定ではない。

**対象**：[SUI計画Draft #3398](https://github.com/hat47x/sui-sensemaking/pull/3398) §8（特に8.8 SI-RAW01〜10）・[TEI取込計画Draft #3355](https://github.com/hat47x/tei-platform/pull/3355) §14・[TEI三視点Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)・[SHO #19](https://github.com/hat47x/sho-composer/issues/19#issuecomment-6708850562)の交点。SUI/SHO各Ownerの意味と保存契約はTEIへ移管しない。

## 1. いま確定できる境界と、確定しない分類

ユーザーが未整理の長文・PDF・URL・画像を後で読むために保管できることは、TEIのEverything is Data／標準Data Structureを試す有用な用途である。ただし「取り込んだ」から「解析した」「意味Roleを受理した」「Cardとして表示できる」「Taskを登録した」「外部に送信してよい」とはならない。巨大な内容は全文Card化せず、Inboxの限定Previewと**明示的に開くReader**を第一候補とする。

| 対象とIdentity | 独立させるべき情報 | 特に禁じる変換 |
| --- | --- | --- |
| Raw Content／Asset／外部Locator | 原byte・digest・場所・取得/非取得・取込時刻・Access/Content revision、必要時Documentのspan | URLだけ登録→原文を取得済み、同bytes→同一Subject、Preview→完全読取 |
| Semantic Subject／Item | Ownerが受理したSubject identity、Scope、複数のRole/Facet、受理版 | Card・TreeNode・Canvas node IDを本体IDとして再生成、未分類→自動Task |
| 構造とRelation | Tree/Island/Group/所属・多重Edgeの個別ID、方向・多重度・有効時点、関係出典 | Canvas間の線やGraph layoutから、未受理の業務Relationを作る |
| Canvas／View Occurrence | Canvas ID、元Subjectへの参照、各Canvas内Occurrence/Placement、選択/折畳み/LOD/表示世代 | 同じSubjectの複数Canvas表示を複製Subjectと数える、表示除去→Subject Delete |
| Representation／Document | Reader/Markdown/Diagram/CRUD/朗読の表示、Document occurrence、元Source span・変換/Renderer版・Loss | reflowページ・Canvas座標・音声時間→元Source位置/逆Writeを決め打ち |
| Action・Effect・Event／Execution | 利用意図、選択対象・Operation identity・現在Grant、Execution/Receipt、実際の出来事の観測時点・出典 | Task意図→Action許可、EKI Task completed→業務Event/Commit確定 |

**分類の結論**：Task・Action・Eventを互いに排他的なItem種類のCore enumへ固定しない。まずは用途別のRole／Relation候補として識別し、必要に応じて独立したSubject/Occurrenceを持たせる。たとえば「請求書を確認する」という文章はRaw Itemであり得るし、人が受理すると予定Taskにもなり得る。確認を依頼するAction Intentと、実際に確認が行われたEvent/Receiptは別のIdentity・出典・時刻を持つ。TaskとEventの同一視や、Event説明文字列から副作用を再実行することを禁じる。Roleの適用可能性は正式Meaning/Ownerの受理でも外部Hostの現在Grantでもない（[TEI #2620](https://github.com/hat47x/tei-platform/issues/2620)、[#2507](https://github.com/hat47x/tei-platform/issues/2507)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)）。

SUIのTask、TEIのBusiness Procedure／Function、EKIの資源Taskは**それぞれ固有の意味・完了条件**を保持する。外部Taskサービスへ写像する場合もProviderごとのCapability・外部ID/Revision・再送/競合/実Receiptを別に検証する。時間上のEventをTEI Functionの起動EventやEKI Attemptの完了と推定しない。

## 2. 複数Canvasの分割は配置操作、意味モデルの複製ではない

一つのCanvasが巨大になったとき、関連したItem群を別CanvasへShare/Extractして閲覧性を維持する。ただし最低限、**Source Subject／元Relation／Canvas内Occurrence／文書引用Occurrenceを別々のID**として扱う。

- **Share**：A上のSubject SをBに配置する。AとBのOccurrence IDは独立で、参照するSubject IDは同一。Bで座標・折畳み・表示Scopeを変更してもA/Subject/Raw byteは不変。
- **Move表示**：A側のOccurrenceを除きBへ残す。全Canvasから非表示になってもSubjectや元Relationを自動削除しない。元Sourceへ復帰・Undoできる根拠を保持。
- **Extract候補**：元Rawの範囲から新Subject候補を作る場合、原Asset・Span/Revision・抽出方式・Lossを保持。内容が同じでも受理前に元Itemを置換しない。
- **Canvas横断Relation**：元の構造Edgeを表示上省略・束ねても、Edge Identity・端点・多重度・向きは残す。同じ端点に複数の業務Relationがある場合、線だけから逆Patch先を特定しない（[TEI #2497](https://github.com/hat47x/tei-platform/issues/2497)、[#3263](https://github.com/hat47x/tei-platform/pull/3263)）。
- **表示/実Dataの粒度**：二つのCanvasで同Subjectが見えてもResult Rowが二件という意味ではない。CRUDの明示Rは保持し、未列挙セルはunknown。Function Data Context／実Provider ReadのRowset・Grainは別（[#2586](https://github.com/hat47x/tei-platform/issues/2586)）。

画面の一枚/複数枚、IslandやKJ配置、受理済み意味モデルの分解は**別の操作**。Canvas分割は通常はDisplay/Representationの変更で、意味Revision/Source Byte/Owner Acceptance/Provider Effectを変化させない。ただし編集者が関係の意味自体を変更する場合は、別のTyped Change→Preview→Source Revision/Owner Acceptanceへ送る。非可逆なLayoutや合成Edgeからの逆編集を拒否し、適格な元SourceへNavigateできることを優先する（[TEI #1124](https://github.com/hat47x/tei-platform/issues/1124)）。

## 3. TEI/SHOの実際の受領段階を分ける

| 利用場面 | 最初の実入場と残す不足 | 既存Owner/試験 |
| --- | --- | --- |
| P0-D **未分類Rawの保全** | SUI Inboxと実ReaderでRaw bytes/Locatorを別報告。TEIへのDocument/sourceとしての取込はManaged/Loaded/Parsed/Boundを別に検査し、通常Markdownを意味Sourceとして自動実行しない | SUI #3398 SI-RAW01/02/03、TEI #3346/#2621、SHO #4/20 |
| P0-I **Role候補の検査** | Rawの記述・Task/Action/EventのRole候補を、受理済み意味・Source/Role版・Requiredの完全性から分離。役割不明ならunclassifiedを保つ | TEI #2619/#2620/#3262、SUI #3398 SI-RAW10 |
| P0-M **三視点とCanvas** | 一Structured SubjectをData/Function/Business＋二CanvasにProjection。Subject/Relation/Display Occurrence・CRUD既知/unknownを実Parser/Indexから照合 | TEI #1124/#2497/#3263、SUI #3398 SI-RAW06 |
| P0-R **実Function/Query/Provider** | Canvas上のRelationやTask候補から実Read/Effectを推論しない。Function→Query/Providerの選択実行、Row/Field/Scope/Read Cohortを実観測 | TEI #2586/#2506/#2622 |
| P0-C **同一用途の接合** | Source/Meaning/Role/World Scope/Revision＋Consumer実受領を同一Attemptで照合。SHOへ渡したMarkdown図やSUI草稿をTEI正本へ逆流させない | [TEI #3355](https://github.com/hat47x/tei-platform/pull/3355)、[SHO #19](https://github.com/hat47x/sho-composer/issues/19) |
| P1〜P2 **外部/双方向操作** | Reader二Consumer・Provider D1のactual selected/invoked→文書WriterのSave/Reopen→外部Task/SHO Handoffの実Fileを別Gateで判定。P0をMCP/PERT/Writer完成待ちにしない | [TEI #3371](https://github.com/hat47x/tei-platform/pull/3371)/#1251、[SUI #3397](https://github.com/hat47x/sui-sensemaking/pull/3397)、[SHO #21](https://github.com/hat47x/sho-composer/pull/21) |

**現存Evidenceの上限**：SUI #3398は**Docs-only Draft**でSI-RAW01〜10の実Item DB/Canvas/Reader受入はNOT RUN。SUI #3397の限定Node CI成功は通常Markdown草稿を生成できる範囲で、opt-inの同一File→実SHO ReaderのH1やSHO G2 Writer/Saveを達成しない。SHO #19で受入条件を整理済みだが、実TEI Modeler＋SHO Document二Consumerの同一Attemptは未実証。TEI #3354の11 CLI Reply/32合成Controlsや#3350のPlugin能力選択は本件のIdentity/Role/Canvas実E2Eではない。

## 4. 既存Scenarioへ差し込む最小負例（新SSOT・Product Schemaを作らない）

[SUI #3398 §8.8](https://github.com/hat47x/sui-sensemaking/pull/3398)のSI-RAW01〜10、[TEI #3263 M-01〜10](https://github.com/hat47x/tei-platform/pull/3263)、[SHO #19 AS-1〜6](https://github.com/hat47x/sho-composer/issues/19#issuecomment-6708850562)、[TEI #2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)の独立Expectedを**同じ試験入力へ追加する因子**として再利用する。これらを一つのReference TestでPASSにして別製品の実動作を捏造しない。

| 最初の圧力 | 独立Expected／失敗として扱う条件 | 次段への資格 |
| --- | --- | --- |
| **V0：Raw未分類のまま保存・明示Open** | 未読・未分類・未解析を維持、ファイル原文の実読取とURL-only未取得を区別。カード化・AI送出・Task生成の義務0 | Raw Store/Readerの用途限定 |
| **V1：同じbytesの別Subjectと、同Subjectの二Canvas** | 同内容別Assetは別Identity。同Subject A/B Placementは複数Occurrenceだが元Subjectは一つ。A削除・移動・SplitでB/Relation/文書引用を消さない | Layout-only／Identity健全性 |
| **V2：Item→Task候補→Action Intent→実Event/Receipt** | Task分類の採用前はRawを保持。採用しても実行許可や事実上のEventを作らない。別Principal/外部Task ID/Grantなしの操作は保留 | Role/意味候補。実Effect未資格 |
| **V3：同Subjectの三視点CRUD/二Canvas** | 元Source/RelationのDirection/Multiplicity/Edge ID、既知R維持、未列挙Cell unknown。図上の単一線から多重Edgeを逆編集禁止 | read-only Modeler部分資格 |
| **V4：SHO/AIと源泉範囲** | 同じMarkdown可視投影でもSourceが別Root/別Revision、秘密Edgeが隠れている、Reader応答途中切断なら図解のSource完全資格なし。原Raw編集は独立に継続 | Document draft受領とTEI accepted図解を分離 |
| **V5：変更時の失効** | Source改版とCanvas位置変更、Role改版とContent同一、Display epoch変更とMeaning不変、Grant失効と歴史的Receipt保持を別々に評価 | 影響するClaimだけ再評価 |

**第一優先の4例**はSUI SI-RAW01/02/03/06を既存DocumentV1とReaderでread-only比較すること。次にTEIの既存Structured Sourceに対応する二Canvas/CRUD正例と多重Edge/unknown負例を1件。P0-Rの実ReadとP0-Mの投影は**並走可能**で、同一World/Source/Scope/Revisionの接合を示した場合だけP0-Cへ進む。最初の欠落辺がRaw Reader、Role/Source admission、Graph/CRUD product projection、Provider実Readのどれかを、各Ownerがそれぞれ記録する。

## 5. 採用条件・性能と無条件に広げない項目

- 最初の実装は既存SUI DocumentV1／TEI標準構造・Role／SHO Raw Editorを利用し、**一つのItem型・全業務共通Taskテーブル・全媒体統合Canvas Store・万能Agent Operation**を先に採用しない。
- Raw投入・Reader到達・原文保存・Scope内の再表示が明示可能であること。100,000字原稿やPDF/URLはReaderのメモリ・中断・権限条件を独立に測定し、現在のSHO Readerにある個別のサイズ上限を全Assetの一律上限としない。
- 巨大Canvasの分割は認知負荷を減らすという仮説であり、実際の作業効率や誤操作/忘却の減少を**未実証のUX効果**として扱う。30〜50 Itemの初期Hybridで操作数、Inbox→Reader再到達、同一Item複数配置・元Relation維持・Undoを計測し、1,000→10,000 Itemの性能は後段。GPU/高機能AIを必須にしない。
- 正式Grammar/Role/Plugin/保存形式、Task external synchronization、編集のOwner Acceptance、Canvas間移動の原子的Save、Provider実Grant・Effect/Receipt、外部SHO H1/G2 Writer/TEI三視点Productの合格は**未採用／NOT RUN**。SUI/TEI/SHO各Repositoryの別Runを合算しない。

**実装の責務**：SUI #3398＝Item/Inbox/Canvas/Role候補・ユーザー体験、TEI #2495/#2497/#2507/#2620/#1124/#3262＝標準構造・解釈/意味受理・投影/Source・必要範囲、TEI #2586＝実Function取得、TEI #1251/#951＝実Plugin Invocation、SHO #4/#6/#19＝原文編集・受渡し・文書表現、TEI #2514/#1180＝独立Expected。正式Issue/ADRの採否や優先順位を本Draftで変更しない。
