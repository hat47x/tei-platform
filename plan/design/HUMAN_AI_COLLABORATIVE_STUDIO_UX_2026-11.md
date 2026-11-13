# TEI Studioの人間・生成AI協業UX：共通操作と作業引継ぎの設計

- 状態：**実装前の横断設計・受入計画**。利用者の要求と既存契約から成熟した方針を統合したもので、新規製品API・正式Schema・意味受理を宣言するものではない。
- 更新日：2026-11-13（JST）
- 優先順位：**TEI Studio Desktopの実動作と、TEI自身によるStudio Desktopセルフビルドを最優先**とする。協業UXの検討はこれを妨げず、read-onlyから段階的に進める。
- 主な既存担当：[#1824 Studio](https://github.com/hat47x/tei-platform/issues/1824)、[#1124 Modeler](https://github.com/hat47x/tei-platform/issues/1124)、[#4515 Subject Pin](https://github.com/hat47x/tei-platform/issues/4515)、[#1158 共通Operation](https://github.com/hat47x/tei-platform/issues/1158)、[#1251 Plugin実呼出し](https://github.com/hat47x/tei-platform/issues/1251)、[#2831 セルフビルド](https://github.com/hat47x/tei-platform/issues/2831)、[#4583 Desktop Build Intent](https://github.com/hat47x/tei-platform/issues/4583)。
- 接続先：[固定対象・高度ContextのDraft #4479](https://github.com/hat47x/tei-platform/pull/4479)、[#2509 外部AI Harness](https://github.com/hat47x/tei-platform/issues/2509)、[#2516 開示](https://github.com/hat47x/tei-platform/issues/2516)、[#2497 投影・逆写像](https://github.com/hat47x/tei-platform/issues/2497)、[#1180 Scenario](https://github.com/hat47x/tei-platform/issues/1180)、[#2514 独立Expected](https://github.com/hat47x/tei-platform/issues/2514)、[#4496 Data Workbench](https://github.com/hat47x/tei-platform/issues/4496)、[#3503 Interaction Flow](https://github.com/hat47x/tei-platform/issues/3503)、[SHO Composer #19](https://github.com/hat47x/sho-composer/issues/19)。

## 1. 本文の役割・設計判断

本書は、人間向け画面、生成AI向け操作、両者の引継ぎを**利用体験として接続する**。Semantic Selectionの詳細契約は#4515とDraft #4479、Sourceの正本編集は#1124、Application Operation/Providerは#1158/#1251、実権限/副作用は#1174、Studio物理実装は#1824、セルフビルドは#2831の責務を維持する。ここで新しい統合Host、Task Registry、Meaning Store、Approval Engine、Harness Runner、第二のSelection Resolverを作らない。

**核となる判断**：
1. 人間とAIのために別の意味モデルを作らない。Authoritative Source/Accepted Meaning/同じ型付きOperationを、GUI・CLI・外部Harnessへ投影する。
2. 人間単独・AI単独・協業のいずれも正規利用形態。AIパネルは任意で、編集と閲覧に不可欠な領域を占有しない。
3. 協業の中心はチャット履歴でなく、**対象・目的・差分・検査・結果**。会話は提案と説明の入口であり、確定した設計の正本ではない。
4. AIによる提案は、通常の人間による提案と同じPreview/Impact/既存Acceptanceへ接続する。AIのTool成功、候補JSON、説明文をMeaning受理やEffect確定に昇格しない。
5. Targetの表現は多様でも、同一対象を辿れることを優先する。三要素分析法の原典由来のデータ・機能・業務の各モデル・図法は忠実に比較し、TEI独自のEverything is Data/Role表現と黙って同一視しない。

## 2. 共通操作と三つの利用形態

| 利用行為 | 人間向けStudio | AI/CUI向け | 共通の判定点 |
| --- | --- | --- | --- |
| 見つける | Workspace/モデル一覧、全体図、検索、参照元/先 | Catalog、Design Context Query、Trace | Qualified Subject/Scope、取得済み・未取得・不明 |
| 読み解く | Property、Table、Graph、DSL/Source、診断、統計 | Inspect、依存/Source位置、機械可読診断 | Semantic ID、Source occurrence、Revision、開示可能範囲 |
| 対象を指定 | Focus、一時選択、明示Pin、比較対象 | 同じSelection/Qualified Targetの構造化指定 | Pinの意図と現在の利用資格は別 |
| 直す | Source/表/可逆な図編集、保留差分 | 型付きChange/パッチ候補 | 逆写像可能性、Source条件、実対象不変 |
| 確かめる | 意味差分、影響、Scenario、未解決表示 | Validation、独立Expected、Evidence | 本当に検査された対象/版と範囲 |
| 採用・実行 | 既存レビュー/受理と明示Operation | 既存Application Operationへの明示要求 | Host/Provider Grant、Effect、Receipt、独立観測 |

同じ情報量や同じ画面配置は必要ない。人間には理解しやすいラベルと関係図、AIには型付きID・位置・版・未解決情報を与える。ただし表示名・図の描画ID・DOM座標を意味識別子や権限根拠にしない。

## 3. 利用者と協業の分担

- **ドメイン専門家**：業務概念、業務手順、例外、期待する結果を確認する。高度なデータモデリング、SQL方言、ビルド実装のすべてを単独で担う前提は置かない。
- **開発者**：型、NULL/未取得、Relation多重度、データ保存、Provider、テスト、性能、可逆編集の技術的整合性を扱い、必要に応じて専門家の判断を支援する。
- **生成AI**：既存の根拠を読み、候補を生成し、差分・テスト・不足を提示する。人間と同じOperation契約を利用するが、業務意味の受理・Host認可・外部Effectを自ら獲得しない。

全変更に新たな承認手順を強制しない。通常開発のレビューを基本とし、AIが未取得の情報を推測で埋めたり、出典のない業務意味を採用済みと宣言したりすることを防ぐ。

## 4. 画面の情報設計：全体図を道案内として残す

Studioの初期構成は、左に対象一覧/検索、中に表・図・DSL/Source、右に可変幅のInspector/Pin/協業、下に診断・差分・テストを置く**再配置可能な作業領域**を基準とする。これは配置の提案であって固定レイアウトの正式採用ではない。

- **全体と詳細の往復**：Data/Function/Business、Function/Rule/Operation/Providerを横断する全体図・概要を維持し、詳細を開いても現在地と復帰先が分かる。巨大な図は意味上の区画へ分解する。AI向けには同じ全体・現在地・近傍を構造化テキストで渡す（RQ-036、#1124）。
- **X-TEAの利点を保持する**：項目単位の参照元・参照先、該当箇所へのジャンプ、参照数/未使用候補、表・CRUD・関係図を標準の探索機能として扱う。参照0件は「現在解決できた対象集合において参照なし」であって、未解析や開示制限があるときの全域未使用証明ではない。
- **高密度な作業**：キーボード検索/移動、戻る/進む、表上の入力、複数タブ、Diff、診断の位置ジャンプをAIなしで利用できる。チャットを必須の操作経路にしない。
- **標準と変更可能範囲の区別**：標準/設定/Plugin/生成部品/案件固有、read-only/可逆編集/候補作成の違いを可視化する。Graph描画上の見た目や表示用Relation IDを、確定した編集対象へ昇格しない。
- **Sourceを保つ**：Markdown/DSL/表/図は同じAuthoritative Sourceへの投影とし、非可逆変換の編集は拒否・Sourceへ移動する。既存の書式・コメント・糖衣表現を不用意に壊さない（#2497/#2621）。

## 5. Focus・Pin・Task・Reviewの責務

| 概念 | 保持するもの | 保持してはいけない保証 |
| --- | --- | --- |
| Focus/Temporary Selection | 現在のカーソル、ハイライト、図の一時選択 | 他画面へ移動してもAI対象が不変という保証 |
| Pinned Target | 利用者が固定したQualified Subject/意図、再確認に必要なOrigin | 現時点のSource内容・Grant/開示・Meaning資格 |
| Activity/Task（UI表示概念） | 目的、参照対象、提案、検証、未解決事項へのリンク | 新しい中央Task正本・継続的な実行資格 |
| Review | 変更差分、対象、Source/意味/テストの実際に検査した版 | 未実行のテスト、Owner受理、Commit成功の偽装 |

最初は#4515の**明示的なSubject Pin一件**を完了させる。Focus変更、IME、Esc、空白クリックで解除せず、明示Unpinと局所Undoを備える。Workspace切替・同名別ID・Source移動・A→B→A・後着の古い応答で勝手に別Subjectへ再束縛しない。見た目のグラフEdge IDを意味Relation IDとしてPinすることは禁止。Pin Intentの保持と、現在の参照/開示資格を別表示する。

Activity/TaskとReviewは初期段階では**既存Operation結果を寄せて見せるUI概念**である。専用の永続Task schemaやマルチAgent共有Storeを先に標準化しない。実際の引継ぎと競合が必要とする最小情報を試験で決める。

## 6. AIへの受渡しと復帰

### 6.1 出す前に確認する

選択した作業対象に対して、既存のSource/Identity/Permission投影で取得できる範囲に限り、次を人間が確認できるようにする。これは**必要情報の一覧**であり、現時点で正式な新Envelope Schemaを採用するものではない。

- Workspace/許可Root、Qualified Subject、Source occurrence・Revision/観測世代、固定Selectionの意図
- 作業目的、選択/除外範囲、静的対象集合と動的Filterの違い、AIに渡す近傍・引用根拠
- 取得済み/部分的/未取得/開示不可/不明、必要なPlugin/Interpreter/Toolと実際のProvider
- 外部Harnessへ**実際に送信する開示可能Slice**（Tool Reply/Log/履歴再送の経路を含む）、許可範囲、過大な受渡しの防止

「Pinした」「Inspectに成功した」だけでは現在の開示・外部送信が許可されたことにならない。Host側の現行資格と、#2516のDisclosure/二世界試験を使い、非公開情報の存在や件数を低権限UI/ARIA/Tool JSONへ漏らさない。外部HarnessのOS隔離・送出は#2509/#2515の別実証とし、見た目の確認ボタンで強制されたと主張しない。

### 6.2 戻った提案を処理する

AIの返答は、説明文だけで終えず、**対象付き候補、根拠、未確認事項、要求するOperation、変更案、実際のTool/検証受領**へ分解して画面に投影する。旧Sourceや別Workspace、欠けたTool Reply、古いSelectionの候補は無音で新Sourceに適用しない。候補が危険・曖昧なときは、別Draftとして保持し再探索/再検証を促す。

「提案を採用」はモデルSourceの条件付き変更であり、「実行」はHost/Providerの権限を伴う別操作。「保存」「Meaning受理」「DB Commit」「Receipt確定」「独立Readerの観測」を一つの成功バッジにしない。実行結果がunknownの場合は同じOperation IDで状態照会を行い、二重送信しない（#1174/#4496）。

### 6.3 SHOと外部ツール

TEI→SHOはSource/Occurrence/Revisionを維持した文書側へのHandoffであり、SHOは単独起動・Markdown編集を継続できる。TEI Accepted Meaning/生成Artifactへの直接書戻しは不可。GUI/CLI/AIのどこからも、同じ「どの対象のどの版を何の目的で扱うか」に復帰できるようにする。AI HarnessはDSHなどの外部Consumerを利用し、TEI専用Agent Loopは作らない。MCPは必要性が実証された場合のAdapterであり必須ではない。

## 7. 初期縦断シナリオと独立Expected

第一候補は公開合成受注モデルの**受注明細.数量**を対象とする。同じ固定Fixture・同じTask・独立Expectedを、人間単独（H0）、AI単独（H1）、協業（H2）で比較する。最初はread-onlyを完了させ、続いて候補変更・既存Acceptance、さらに実Provider Effectへ段階を分ける。合成Source/CLI/Browser/Wails/外部Harnessの結果を別HEAD・別Attemptから合算しない。

| Case | 作業・反例 | 最小の受入条件 | Owner |
| --- | --- | --- | --- |
| UX-01 | Studio/CLIで数量をCatalog→Inspectし同じ対象へ到達 | Qualified ID、Source/版、部分/不明が一致。実Wails受領は別assert | #1824/#1158 |
| UX-02 | Data表→Function→Business/CRUD→Sourceへ往復 | 同じ意味対象へジャンプし、参照元/先・現在Scopeの統計を表示。未取得の参照を0扱いしない | #1124/#3262 |
| UX-03 | Subject Pin→他画面へ移動→戻る | Pinの意図不変。古い資格は別表示し現在性を再確認 | #4515 |
| UX-04 | 同名別Workspace、同offset別Subject、改版/ABA、後着応答 | 無音の再束縛0、古いReply現在表示0、再資格化待ちを示す | #4515/#1824 |
| UX-05 | 二重Graph Edge/描画用ID、非可逆Projection | 誤ったRelation特定・Source書込0。根拠を辿れる | #1124/#2497 |
| UX-06 | 人間が選んだ対象の限定ContextをAIへ送る | 送出前Previewと実送信Slice/実受領を別観測。遮蔽した情報・存在は漏らさない | #2516/#2509 |
| UX-07 | AI候補と人手案を同じSourceへ適用予定 | 同じtyped Change/Preview/Impactを用い、競合・未解決時は保留。受理を偽装しない | #1124/#1180 |
| UX-08 | AI/Studioが数量0の拒否検証を示す | 独立Expectedが正負例を区別。Tool successだけでPASSにしない | #1180/#2514 |
| UX-09 | Data Workbenchの固定1行1項目変更 | Pending diff→適格CAS/Grant→Receipt→独立Read。JOIN/集計や未取得を自動更新しない | #4496/#1174 |
| UX-10 | TEI→SHO文書、文書→TEI対象へ復帰 | Source/版・保存の責務を保持。固定成果物を無断上書きしない | SHO #19/#4 |
| UX-11 | Windows生成StudioとCLIで同じread-only操作 | 同一実Provider選択/実Invoke、実WebView/IPC受領を別々に記録 | #1824/#1251 |
| UX-12 | 協業部品をTEIが生成する | Build Intentに実Consumed Source/Node/esbuild/Wails/nested moduleが含まれ、Stage1実TEI→Stage2実Windows Consumerへ到達 | #2831/#4583 |

UX-01〜12は**既存ScenarioとIssueの独立Expectedへの引渡し候補**であり、追加の製品Hard Gateではない。実行していない試験はNOT RUNとする。

## 8. 実装順とFit/Gap

| 段階 | 追加する範囲 | 先行条件・不成立なら |
| --- | --- | --- |
| U0 実Desktop基線（P0並行） | 既存Wails Window→IPC→Source-backed Inspect、生成Workbench/Inspectorが実Consumerで使われる | #1824/#2831/#4583の現行P0。UX開発を理由にStudioセルフビルドを後回しにしない |
| U1 読み取り・全体ナビゲーション | 現行Catalog/Inspect、Subject同定、モデル横断ジャンプ、参照統計（取得Scope明示） | 既存Source/意味資格が不足ならpartialを示し、探索用の第二Resolverを作らない |
| U2 明示PinとContext Preview | #4515の一件Pin、現在資格の再確認、AI OFFでも使えるInspector表示 | Pin/Context Shelfの正式永続Schemaや新Task Storeは採用しない |
| U3 外部Harnessとの任意往復 | 公開合成Sliceのみを限定Handoff、Tool/返信の完全性、候補と根拠のViewを比較 | OS/通信/Disclosure資格不明なら送出を停止し、CLI-only H0は継続 |
| U4 候補変更・差分レビュー | 可逆なSource/typed Change一件→Preview/Impact→既存Acceptance→再読取 | Source保存/意味受理/Effectは別経路。競合時の黙示Rebaseは禁止 |
| U5 高度な編集と複数主体 | 複数Pin、Relation occurrence、SHO span、DB Rowset/差分、複数Agent協業 | 先行縦断の不足に基づいてのみ追加し、初期P0のHard dependencyにしない |

U0とU1のHeadless部分は並走可能。U3/U4/U5を#2831のDesktopセルフビルド受入へ追加の必須条件にしない。

## 9. 効果をどう測るか

同一Fixture/独立Expected/同じ許可条件でH0（人間のみ）、H1（外部AIのみ）、H2（協業）を比較する。最初は測定を先行し、根拠のない固定閾値を置かない。

- **到達の効率**：対象発見までの操作回数・時間、参照元→実Sourceへの移動回数、視点切替時の迷い/誤対象到達。
- **編集の品質**：誤った対象への変更0、非可逆なSource破損0、誤った「検証済み」「受理済み」「Commit成功」の表示0。
- **根拠と開示**：必要Sourceの取得率、partial/unknownを正しく保留した割合、選択外・非公開情報の送出と存在漏れ0。
- **実行と再現性**：同一ScenarioのCLI/Studio実Operation・Provider/Receiptの結び付け、実Consumer受領の証拠、異版・後着の誤採用0。
- **導入と運用費用**：AI呼出し数、Tool Reply/ログ量、レビュー時間、変更後の再資格化費用、非AI利用時のUI操作性。

## 10. 現在の事実と未採用事項

2026-11-13のIssue/PR記録において、#4515のSubject Pinはread-only実装・試験前、Draft #4479は固定対象/Contextの設計であり実GUI/外部Harnessの完了ではない。#4572/#4579/#4583は生成部品・Windows DesktopとBuild Intent/Stage1→Stage2に異なる資格を持ち、別Attemptの成功を一体セルフビルドの成功へ昇格できない。#2509には外部Harnessの限定read-only/Skill Load実証があるが、実効OS隔離・完全なTool受領・外部開示適格性・AI編集往復は未証明。

本書では設計と受入条件を整理したのみで、実TEI CLI、Node、Go/Wails、OS隔離、外部AIの新しい試験、Owner Meaning Acceptance、正式仕様の採用は**NOT RUN / NOT ACCEPTED**。以後の状態は各Owner Issueとdocs/status.mdを正本として確認する。
