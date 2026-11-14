# TEI Studioの人間・生成AI協業UX：共通操作と作業引継ぎの設計

- 状態：**実装前の横断設計・受入計画**。利用者の要求と既存契約から成熟した方針を統合したもので、新規製品API・正式Schema・意味受理を宣言するものではない。
- 更新日：2026-11-14（JST）
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


## 11. 協業状態の境界：対象・版・根拠・受領を分ける

前節のFocus/Pin/Task/Reviewに新たな正本を追加せず、少なくとも次の**独立した現在性**をUI/CUIで説明する。表示・検証項目の設計であって、新しい永続Schemaや単一の「有効」フラグの提案ではない。

| 区分 | 問い | 不成立時の体験 | Owner |
| --- | --- | --- | --- |
| Target Identity | 今のWorkspace/Kind/ID/Occurrenceは固定対象と同じか | Pinの意図を残し、別対象への自動再束縛を止める | #4515/#1120 |
| Source/Meaning | Source bytes/Revisionと意味解釈は今回の作業の元情報と整合するか | 古い資料・提案を明示し、現在Sourceへの適用を保留 | #1124/#3262 |
| Context Completeness | この問いのRequired集合と実Read集合は閉じているか | 取得済み・未探索・partial/unknownを区別。空を不存在にしない | #3262/#2514 |
| Disclosure/Authority | 現在のConsumer/Recipientへの開示・Operation権限があるか | 送出/Applyを停止。未観測の変化はunknown | #2516/#1174 |
| Delivery/Execution | このAttemptのTool Reply、終了、Provider呼出し、実観測は揃ったか | 受領不完全・実行未確認・確定不明を別表示 | #2509/#1158/#1174 |

**反例**：Sourceの注釈・AuthorityLimitsだけが変わり構造ModelFingerprintが不変でも、旧ReviewのSource資格は自動復帰しない（TEI #4514）。同Kind同ID重複や明示CRUDの矛盾を、Indexの後勝ちや画面の最後の値で確定Meaningにしない（#4472/#4475）。source-indexの空結果から対象の不存在を推定しない。既存の診断/Producerがまだ不十分な箇所は未確認として保留する。

利用者向けには「対象は固定されています。内容は再確認が必要です」「参照先の取得が完了していません」「現在の定義と提案時の定義が異なります」「実行結果を確認できません」など短い文言を用いる。全画面に大量の証拠票や承認確認を常時表示しない。詳細根拠は必要時に開く。

## 12. 作業の交代と同時編集：最小の協業手順

まず単一Source/対象に対する一変更を扱う。多主体対応のために中央Task Storeやリアルタイム共同編集を先に増設せず、既存Operationを次の順で接続する。

1. **Locate**：Qualified ID・Workspace/Originから実対象を再取得する。画面Focusを入口に利用しても、固定する作業対象はFocusから独立する。
2. **Observe**：現在のSource/Meaning/Required/開示可能な範囲を取得する。TEIのSource読取とUI/Harnessの完全受領を別扱いにする。
3. **Propose**：対象、基準版、変更意図、実際に確認した根拠へ変更候補を結び付ける。人間案/AI案のどちらにも同じ変更契約を利用する。
4. **Review**：現在Sourceとの照合、Source差分・意味差分・影響/不足/競合を表示。再束縛や採用、破棄は明示操作とする。
5. **Apply/Observe**：既存AcceptanceとProvider資格で条件付き適用する。保存、Meaning受理、実Effect、Receipt、独立Readerの観測を別々に表示する。

これは**操作順の提案**であり、正式な外部Wire APIやTask JSON schemaではない。

### 競合をどう扱うか

- 人間AとAI Bが基準版R0から開始し、AがR1へ保存した場合、後着したBのR0提案は**R1へ無音適用しない**。Bの案を保持し、明示再比較・再計算できるようにする。
- 別ファイル・別SubjectでもFunction/Rule/Relationの依存があれば競合し得る。依存閉包がunknownなら「競合なし」もunknownとする。
- AIの終了・再開と、実Source保存/実Operation Effectを混同しない。同じHandoff PayloadやResultの再送は、元TEI Sourceの当該Attemptの再読取・実行を証明しない。
- 開示資格の変更が観測されたら既存Consumerで再評価する。観測できない期間の即時権限失効通知まで保証しない。無関係な許可済み対象の閲覧は不必要に停止しない。
- 最初は候補の保持・明示Diff・条件付き保存を検証する。自動マージ、Agent間の仲裁、リアルタイムカーソル同期は後続採否とする。

判断の経緯は既存Issue/PR、Source、設計文書、任意のOKF管理から再発見できるようにする。AI要約は出典付き候補であって第二のMeaning正本ではない。SHOのWorking Copy/文書保存/独立再読取とTEIのSource/Meaning受理を混同しない（SHO Draft #50 QH-08〜10）。

## 13. 協業パネル・レビューの最小表示

| 場所 | 標準で示す | 詳細表示で示す |
| --- | --- | --- |
| Model Map/Tree | 現在位置、モデル種別、参照元/先、未解析の有無 | Relationの向き・多重度、Source occurrence、Scope |
| Pinned Target | 対象名/Kind、固定状態、再確認要否、明示解除 | Workspace/Origin、ID、版、Source状態/現在資格 |
| AI Handoff Preview | 対象、目的、送る範囲、依頼する操作種別 | 実送信Slice、許可されたRecipient、未取得・unknown |
| Candidate Review | 変更意図、Source/意味差分、影響要約、未確認 | 実Read、基準版、Expected、依存、Tool/Provider Result |
| 実行結果 | 保存・拒否・確定不明と次の操作 | Source再読取、業務拒否/技術失敗、Receipt/Effect/Observation |

低権限利用者へ「非公開情報が何件あるか」も無条件に開示しない。CLI/CUIには同じ意味の構造化状態を渡し、画面文言そのものへの依存は作らない。

「AIへ調査を依頼」「変更案を採用」「外部作用を実行」は別Action。AI応答・外部Tool内容は信頼できない入力であり、そこに埋め込まれた命令から権限変更や自動Applyを発火させない。新しいApproval Engineを作るのでなく、既存のUser Intent/Operation/Host Grantで判定する。

## 14. 協業固有の独立Expected候補（UX-13〜22）

前節UX-01〜12と重複しない失敗モードを、既存Scenarioと独立Expectedへ引き渡す。**未実行**であり、新Hard Gateにはしない。

| Case | 反例 | Expected | Owner |
| --- | --- | --- | --- |
| UX-13 | 人間がR0→R1へ編集し、AIのR0案が後着 | 自動Apply 0。旧候補保持。現在R1との再比較が必要 | #1124/#1180 |
| UX-14 | 構造Fingerprint不変でSource byte/AuthorityLimitsのみ改訂 | 旧Reviewをcurrentと表示しない。Source/Grantの再資格化は別 | #1124/#2516 |
| UX-15 | 同Kind ID重複／明示CRUDの矛盾 | 後勝ち値を確定値として表示・書込しない。候補と出典/診断を区別 | #4472/#4475/#1124 |
| UX-16 | 同じHandoff PayloadでもTEI実Source/Subjectが異なる | Payload一致だけで成功にせず、TEI実Source再解決で相違を識別 | SHO #19/TEI #3262 |
| UX-17 | 外部Harnessで56要求/55返信・途中切断 | 完全受領・検証完了を表示しない。実Reply/終了を区別 | #2509/#3262 |
| UX-18 | Grant失効を観測後、旧成功Replyと既表示DOM/ARIAが残る | 旧Replyの新規表示拒否と既表示内容の開示再評価を別assert | #2516/#1824 |
| UX-19 | 外部AI文章に「権限を上げて自動Apply」とある | 文章から権限獲得/実Operation 0。通常提案として処理 | #1158/#1174 |
| UX-20 | AI A/Bが別ファイル編集、共通Function依存未解決 | 自動的に競合なしと宣言しない。依存のknown/unknownを表示 | #1124/#3262 |
| UX-21 | 同じOperation Resultが外部Harnessから再送 | 同じテキストの再受領を新実行・再検証とみなさない | #2509/#1174 |
| UX-22 | ScenarioはPASSでも実Windows Studio配信JSは旧版 | 実画面を合格にせず、Asset/Consumerと生成Attemptを照合 | #1824/#2831/#4583 |

UX-13/14/15はSource/Modeler系、UX-16/17はHandoff/Harness系、UX-18/22は実Desktop Consumer、UX-19/20/21は後続協業一巡へ渡す。失敗の表示では拒否・未解決・未実行・結果不明を区別する。

## 15. 次の最小縦断と採否保留事項

当面の実装優先は変更しない。#1824の実Wails Window→IPC、#4515のSubject Pin一件、#2831/#4583のStage1→Stage2 Desktop実再入を維持する。#1124の三要素クロスレファレンスと#3262のRequired取得を、実際に使える範囲で測る。

**最小シナリオ**は公開合成Subject一件の「選択→明示Pin→画面移動→同Subjectへ復帰→R0からR1へのSource改訂→旧AI/旧CLI応答後着→現在性失効表示→再Inspect」である。まずAIなしでHeadlessと実Windows Consumerを**別々に**検証し、後段にAI/SHO Handoffを追加して同じ独立Expectedと比較する。

リアルタイム共同カーソル、AI会話履歴の中央Store、専用協業権限モデル、常時全依存閉包計算、第三のEditor正本、協業専用Workflow Engineは、必要性と費用の実測まで**正式採用を保留**する。Source条件付き適用、Review候補、再資格化など既存契約で足りる部分から実装する。

**追補もdocs-only**。新しい製品CLI/GUI/Wails/SHO/外部Harness試験、独立Expected実行、Owner意味受理はNOT RUN / NOT ACCEPTED。


## 16. 2026-11-13 第3次横断統合：閲覧・固定対象の再確認・未保存編集を混ぜない

本節は[#4515](https://github.com/hat47x/tei-platform/issues/4515)と[Pin実装Draft #4597](https://github.com/hat47x/tei-platform/pull/4597)、[共通Context Draft #4479](https://github.com/hat47x/tei-platform/pull/4479)、[Desktop製品契約Draft #4498](https://github.com/hat47x/tei-platform/pull/4498)の現行差分を、**利用者が実際に感じる動作**として接続する。既存の§11〜15を置換せず、未実行の受入条件だけを追加する。正式な公開State/ABIや別Source Storeを作らない。

### 16.1 表示中の対象と、裏側で確認する対象を分離する

現行Workbenchの通常\`inspect(subject)\`は応答により\`selected\`、Editor、Graph、Noticeを変更する。人が**Bを閲覧中**に、固定した**Aの現在性をAI/Inspectorが裏側で確認**する際、同じ表示更新経路を使えばBがAへ戻り、利用者の操作を妨げる。したがって固定対象再確認は、次の区別を守る。

| 経路 | 変えてよいもの | 変えてはいけないもの |
| --- | --- | --- |
| Foreground Inspect（人間がBを開く） | Bの一時選択・Editor・Graph、BのInspect表示 | 明示したPin Aの意図 |
| Pinned Observation（Aを再確認） | Pin Aの観測結果・再確認要否、許可された固定対象欄 | Bの一時選択・Editor本文・Graph位置・編集カーソル・Dirty Buffer |
| Candidate Review（Aの変更案を確認） | Aの変更候補と差分・根拠の独立表示 | Bの未保存編集、AのSource実内容・業務意味・外部Effect |
| 明示的な「Aを開く」 | 利用者の操作によるForeground遷移 | Pinの同一性の自動変更 |

最初は既存\`client.inspectSubject\`／\`readInspection\`と\`responses.ts\`のlane制御を再利用する。\`pinned-inspection\`は**実装上の応答区分の候補**であって正式APIではない。Pin時点のgeneration／Workspace epoch／Qualified Subjectを応答受領時に照合する。同じAをUnpin→再Pinした場合も旧Aの返答を現在のPinに使わない。

このUI世代管理は**Source RevisionやCurrent Grantの証拠ではない**。旧Replyを新規表示させない検査と、既表示内容の開示資格の再評価・必要な非表示化は独立に行う。現在の開示変更を観測できない場合、即時撤回を保証したとは表示しない（#2516）。

**実装差の現況**：Draft #4597にはPin Intent/Observationの純粋状態遷移と単独テストがある。ただし\`main.ts\`の実Workbench表示・生成Bundle・実Windows Consumerには未接続。旧HEADではNodeテスト/型チェック/ビルドと、埋込Bundle byte比較の失敗が同居した。局所単体試験の合格をStudio実装完了にしない。

### 16.2 再確認時の障害を人間の作業へ波及させない

Pin Aの再確認がtimeout/permission unknown/source staleで失敗しても、**無関係なBの閲覧や未保存編集を消去・中断しない**。Aについては「固定した対象は残っていますが、内容の再確認が必要です」と示し、AIへの送信や変更適用を適格な範囲で停止する。B自身の開示・権限が失効したと観測された場合は別であり、必要な現在表示の再評価を行う。

復帰操作は「Aを開く」「Aを再確認」「Aの固定を解除」を区別する。再確認をAの画面遷移へ偽装しない。マウス・キーボード・支援技術で同じ意味を利用できるようにし、非公開Source情報をDOM/ARIA・通知文へ漏らさない。

## 17. 編集中の内容から生成AI・Buildへの版の橋渡し

### 17.1 五つの異なる内容と実際の受け手

[#4498](https://github.com/hat47x/tei-platform/pull/4498)のDirty Buffer／Build Source分離、[#2621](https://github.com/hat47x/tei-platform/issues/2621)のRaw Source、[#2497](https://github.com/hat47x/tei-platform/issues/2497)の逆写像を利用する。

| 別々に扱う対象 | 説明 | 協業時の原則 |
| --- | --- | --- |
| Editor Dirty Buffer | 人間が編集中の未保存文字列とBuffer上の位置 | 保存済Sourceとして扱わない。AIへの送出は対象・範囲と現在の開示を確認する |
| Saved Source | 実際に保存したRaw byte、Source occurrence、Source Revision | 読取・条件付きPatchの基準。Editor表示や構造Fingerprintと同一視しない |
| Interpreted/Accepted Meaning | 解釈済みの対象・意味の版・受理範囲 | SaveやParser PASSだけでOwner意味受理を宣言しない |
| Actual Build Input | Builderが実消費したSource/Toolchain/依存・Build Attempt | Build Previewや事後Hash一覧で実消費が証明されたとは言わない |
| Running/Generated Consumer | 現在のWindow/実exe/配信JS、生成後Stage2の別Artifact | 生成完了を実行中のStudio更新・Stage2実使用と同一視しない |

**ユースケース**：人間が数量の制約をDirty Bufferで編集中に、AIがSaved Source R0で調査し、R0を基準とする修正案を返す。この案を現在Bufferへ自動適用しない。Studioは「AIが調べた版」「保存されている版」「現在の未保存変更」を区別して差分を提示する。A→B→A、別Window/外部Editor、単なる書式変更でも旧修正案の資格を自動復活させない。

未保存BufferをAIへ渡すことを明示的に選んだ場合も、**Bufferのスナップショットを対象とする限定Context**として扱う。Host側のSource受理や外部Recipientへの開示資格を生まない。UTF-16のEditor位置をRaw UTF-8 Source byte spanへ同一視せず、解決不能なら候補のまま停止する。

### 17.2 操作名と実際に起こす効果を一致させる

- **調べる**：現在の許可範囲でread-only Inspect。未保存Bufferや外部Sourceへの無断送出を伴わない。
- **変更案を作る**：適用前の候補を作り、基準版・対象・差分・不足を示す。保存やビルドを始めない。
- **変更内容を比較する**：現在のDirty/Saved/Meaningとの差を表示し、可逆性・影響範囲の確定度を示す。競合のないことを未確認依存から推測しない。
- **保存する**：OwnerであるEditor/Writerの既存条件付き書込みを通す。書式糖衣・コメントの意図しない書換えを避け、再読取へ進む。
- **検証／ビルドする**：対象となる保存済Sourceまたは明示された適格な入力を表示し、**実際のBuilder実行**を別に記録する。初回Studioセルフビルドは保存済Sourceを明示選択し、暗黙Save/Buildを認めない。
- **生成物を使う**：実exe/Window/IPC/WebView2とArtifact/Attemptを照合する。別出力先へ生成して検証しただけで、現在起動中のStudioを無断置換しない。

複数View/Tab間の差分確認は可能な限り同じOperation/既存Source Readerを用い、Source Writer・Provider Grant・Build Runnerの複製は行わない。

## 18. 変更レビューの認知負荷と、複数AIの作業引継ぎ

### 18.1 Reviewは四つの問いを中心にする

利用者が最初に確認したい情報は「**何を対象に**」「**何を変更するか**」「**何を確かめたか**」「**次に何ができるか**」である。長い会話、Evidenceの全byte、Provider診断を常時表示しない。必要なときだけSource/Meaning/Build/Tool/Receiptへ展開できるようにする。

| 最小表示 | 選択時の詳細 |
| --- | --- |
| 対象：Qualified Subject、基準版、今の版、差があるか | Source occurrence、Workspace/Root、Dirty vs Saved、Identityの曖昧性 |
| 変更：Before/After、意味に影響する範囲 | Raw byte差分、Function/Relation/CRUDの依存とunknown |
| 検証：PASS/FAIL/未実行/未確認と試した対象 | 独立Expected、実Tool Reply/terminal、Provider/Consumer/Build Attempt |
| 操作：再確認／候補保持／差分修正／既存Acceptance／明示実行 | Current Grant、不可逆/Effect境界、失敗時の復帰先 |

「AIの提案」「人間による適用」「保存」「実Provider Effect」は別の出来事。AIが提案者であることを特別な承認階層の根拠にしない。相手の回答文やTool Replyに埋め込まれた命令から新しい実行権限を得ない。

### 18.2 並列Agentは最初から自動マージしない

AI AとAI Bが同じR0に基づいて別の提案を返したら、各案の**元対象・基準Source・既知依存・使用Tool・独立検証**を別々に残す。A案をR1として採用後、B案はR0の候補のまま保持し、R1との比較が済むまで自動Applyしない。別ファイルでも共有Function/Rule/Relationや取得できていない負の依存があれば「競合なし」を保証しない。

これは新しいAgent仲裁エンジン、中央Session DB、共同編集CRDTを採用する提案ではない。既存のSource条件付き変更・Git差分・Review・選択的再検証で対応できる最小ケースを優先する。Product内部操作に未実装の分散Lease/Transaction保証を付け加えない。

AIから人間、またはAI AからBへの引継ぎは、**作業目的・固定対象の意図・候補・根拠位置・未解決事項・実際に観測したAttemptへの参照**を中心にする。引継ぎ先の現在権限・外部開示許可・意味受理は継承しない。SHOのWorking CopyとTEIのAuthoritative Source、SUI Card/Canvasの表現・元Occurrenceを同一の保存正本にしない。Raw Evidenceは常時外部送出せず、既存の許可された保管・開示経路を利用する。

## 19. 第3次統合の独立Expectedと最小実装パケット

UX-01〜22・PIN-S0/ABA・Desktop DT/BD/Scenarioの既存受入へ追加する**検証観点の候補**。既存試験を重複して実装するための新SuiteやHard Gateではない。

| Case | 具体的な反例 | 独立Expected（利用者から観測すること） | 既存Owner |
| --- | --- | --- | --- |
| UX-23 | BをEditor/Graphで閲覧中にPin AのInspectを要求 | AのPin観測だけ更新。Bの選択、Editor、Graph、入力中のBuffer・カーソルは不変 | #4515/#1824 |
| UX-24 | AをUnpin→再Pinし、旧AのPinned Replyが後着 | 旧generationを採用しない。意図は現Pinに保持、現在Source資格は別再確認 | #4515（PIN-ABAを再利用） |
| UX-25 | Pin Aの再確認がtimeout、Bの閲覧は有効 | Aは「再確認待ち」。Bは無関係なら閲覧継続。AのAI送出・Applyを勝手に許可しない | #4515/#2516 |
| UX-26 | AIはSaved R0を調査、人間はDirty BufferをR0から変更済み | Bufferの未保存差分を保持。旧AI候補の自動Apply/Save/Buildは0 | #1824/#1124/#2621 |
| UX-27 | UTF-16 Editor選択とUTF-8 Raw occurrenceが一致しない | 文字位置を根拠なく同一化せず、Raw書込み0。候補の再解決を提示 | #1124/#2497/SHO #4 |
| UX-28 | AIがPreviewだけを提出し、実Apply/Buildを行っていない | Preview/Reviewのみ表示。保存・受理・Build・Effectの成功表示0 | #1124/#1158/#2831 |
| UX-29 | AI A/Bが同じR0から候補を提出し、A案だけR1へ採用 | B案はR0候補のまま保持。R1への無音Rebase/マージ0 | #1124/#1180 |
| UX-30 | 別のSourceで同名の項目、同一表示Label/ModelFingerprint | 表示名/構造一致で候補を自動再束縛しない。Qualified Subject/Originを再確認 | #4515/#1120 |
| UX-31 | Build PreviewはR1、実Builderは古いR0を消費 | 実Build入力資格不一致。生成物をR1からの最新Studioと表示しない | #4583/#2831 |
| UX-32 | Stage2生成完了だが実行中Window/JSは旧版 | 生成済みと利用中を別表示。無断置換0、当該Stage2の実Consumer検証前に完成としない | #1824/#4583 |
| UX-33 | 参照展開/Pin再確認に時間が掛かる | 無関係なEditor入力を阻害しない。未完了はprogress/partial、参照0・完全な空集合と混同しない | #1824/#3262/#2018 |
| UX-34 | AI/SHOへ候補を引き継ぎ、相手は別Scope/Grant | 目的と参照は保持できるが、今のRecipientで再資格化する。受領からApply/Disclosure権限を継承しない | #2509/#2516/SHO #19 |

**最初の実装パケットはUX-23に絞る。** まず#4597の純粋Pin状態遷移、現行\`main.ts\`のForeground Inspect、\`responses.ts\`の応答制御、\`readInspection\`のHost結果検査のFit/Gapをとる。次に実Workbenchで「B閲覧中にAを裏で確認」の正負を検証する。生成埋込JSのbyte一致と実Windows WebView2を同じHEADで別assertし、状態テストのPASSだけで製品UXのPASSへ昇格させない。

UX-26〜28のDirty Buffer→Review→条件付きSaveと、UX-31〜32のBuild/Consumerは**別の受入レーン**に残す。#4583 B1-B（N0→esbuild→Go embedの実Builder辺）→B2 Stage1実TEI→B3そのStage2実Consumerの最優先順を変更しない。AI送出や複数Agentの自動仲裁を最初のDesktop/PinのHard Gateに追加しない。

### 判定と停止線

- **この追補は文書のみ**。追加UX-23〜34、実WorkbenchのPinned Observation、Dirty Buffer競合、実AI Handoff、実Builder/Stage2、Current Disclosureの新規試験は**NOT RUN**。
- #4597の純粋Pin試験と生成Bundle不一致は別の事実として保持する。正しいUI結果の説明だけでは、exact HEADのNode/Windows/Consumer資格を代替しない。
- 新しい正式Schema、協業専用Task Store、Approval Engine、Host/Writer/Renderer、Harness Runner、実行時AI必須化は提案しない。


## 20. 2026-11-14 横断統合：DesktopとWebで共通にする協業UX

本節は[Studio #1824](https://github.com/hat47x/tei-platform/issues/1824)、[横断Plugin計画 Draft #4620](https://github.com/hat47x/tei-platform/pull/4620)、[非Plugin Build Subjectの検討 Draft #4621](https://github.com/hat47x/tei-platform/pull/4621)を、§16〜19の作業体験へ接続する。

**共通化するのはUIフレームワークや画面そのものではなく、対象・Operation・検査結果・復帰先の意味である。** 当面はGo単一バイナリ・Wails/WebView Desktopを製品P0とする。一方、Editor/Graph/Inspectorや生成成果物の契約にはWails Native専用の呼出しを混入させず、必要に応じてWebからも同じ型付きApplication Operationを使えるようにする。Reactを前提とせず、Lit/Web Components等も比較候補だが、現行TypeScript Workbenchの一気通貫実証に先んじて全体移行は決めない。

| 意味を共有する操作 | Desktop / Webで独立に判断すること |
| --- | --- |
| Catalog→Inspect、三要素図の全体→個別→参照元/先 | Window/Tab、URL、履歴、Graph/Editorの配置と入力方法 |
| Focusと固定Pin、現在資格、Dirty Buffer、候補Review | ローカル状態の保持方法、他Window/Browserとの同期、復旧 |
| 型付きOperation、Source条件付き変更、検証・結果語彙 | Wails Bridge／将来のHTTP/API transport、実Provider/Host認可 |
| Current Actor/Principal、Workspace Scopeと開示対象 | 非認証ローカルDesktopと認証された複数利用者Webの境界 |
| 生成Source→実Builder入力→Artifact→実Consumer | Wails exeとWeb配布、実WebView2、公開・署名、利用中の世代 |

プロセス構成はA＝同一Wails Processを基線にする。B0＝同じexeから起動するWorker＋local IPC、B1＝別Wails Server exe＋loopback HTTP/WS、B2＝別Server/Worker exe＋Named Pipe/UDSは、性能・OS ACL・起動復旧・通信負荷・Actor/Effectを**実測比較する後続案**である。全案を同じWails RPCと呼ばず、新Worker/Serverを初回P0の必須依存にしない。

同じSubjectの説明がDesktopとWebで一致しても、**ローカルRoot許可をWeb Principalへ継承しない**。画面の選択・PinやAIの要求がHost Grantを発生させるわけではなく、外部Harnessは第三のActorとして実際のDisclosure・Tool受領を別に検収する。GUI・CLIの結果にはQualified Identity、Source/Revision、問い合わせ範囲、観測済み・未取得・不明を保持する。

## 21. 実DBの取得範囲を、人間とAIの説明へ正しく反映する

[FunctionContext Draft #4647](https://github.com/hat47x/tei-platform/pull/4647)では、PostgreSQLから実際に観測できたRelation/RLS/Limit等を用いて、取得可能範囲が制限されていることを部分的に検出するW1〜W3のコードと試験が追加された。Native SQLの隠れた依存、W4結果列由来、正式Product CLI・Studio・外部AIへの受渡し、現在のGrantは未資格である。局所実証をStudioの実装済み表示と扱わない。

**利用者に示す結論は、取得した行数と取得の完全性を分ける。** 画面に適した表示例は次のとおりで、製品の確定文言ではない。

| 観測結果 | 表示すべき内容 | 避ける表現 |
| --- | --- | --- |
| Readerが0件を返したが、必要集合の被覆は不明 | 「今回の取得結果は0件です。対象全体については未確認です」 | 「該当データは存在しません」 |
| RLSやLimitによる制限を観測 | 「確認できた範囲に制限があります」 | 「全件を確認済み」「非公開の行が何件ある」 |
| Native SQLの必要Read-set/負の依存を閉じられない | 「参照元・対象範囲は一部未確認です」 | 「依存0件」「母集団の集計結果」 |
| Source byteは同じだが意味・開示・取得時点が変化 | 「参照条件の再確認が必要です」 | 「最新版で検証済み」 |
| 独立Requiredの一部を実取得し未取得が残る | 「指定範囲で確認済み／未確認項目あり」 | 「設計全体が完成・検証済み」 |

AIにも同じ到達限界を型付き・構造化テキストで渡し、未取得・隠れたQuery依存を0や不存在へ変換しない。表示やCLI/Toolのメタデータ、ARIA、ログを通じた**非公開Sourceの存在・件数・理由の漏えい**は#2516の二世界試験で検出する。低権限の画面には許可された情報だけを投影し、内部の拒否理由をそのまま表示しない。

AIへのContextは問いに相対する必要Source、出典位置、取得済みScope、必要な未解決項目の範囲に限る。固定PinがあるだけでSource全文・Raw Evidence・他Workspaceの会話履歴を自動送信しない。#3262のDesign Context、#2516のDisclosure、#2509の実Tool受領を既存Ownerとし、新Context Engineを加えない。

## 22. 協業の状態表示を既存の実操作へ結び付ける

§18の「対象／変更／確認したこと／次にできること」を利用者が行動できる状態へ落とし込む。会話の長さやAIの完了宣言ではなく、**元対象・元Source・Dirty Buffer・実観測・候補・現在の受理資格**を分けて表示する。

| 現在の状況 | 次に利用者ができること | 自動実行しないこと |
| --- | --- | --- |
| Pin Aが残り、現在のSource資格は不明 | 許可範囲でAだけを再Inspect、Aを開く、Unpin | 別Subjectへの再束縛、AI送信、Source変更 |
| Bを閲覧中にPin Aの再確認が失敗 | Bの編集を継続し、Aの原因を別表示 | BのEditor/Graph/Dirty Bufferを消去・移動 |
| AI案はSaved R0基準、人間には未保存R1がある | 差分を比較し、案を候補のまま保持 | 未保存Bufferの破棄、無音Apply/Save/Build |
| AI A案を採用後、AI Bが旧R0の提案を返した | B案を現Revisionへ照合し、衝突と依存を確認 | R1への自動Rebase・無音マージ |
| DBの実Readはpartial | 範囲限定で確認、追加調査、結論保留 | 全件調査済み、対象不存在、合計値確定の宣言 |
| Builderが旧R0、UIで閲覧中はR1 | 実Consumed byte/Artifact/実Consumerを確認して再Build | 「現在起動中のStudio更新済み」宣言・無断置換 |

SHOのWriterはRaw文書とOccurrenceの責務を持ち、TEIのMeaning/Source改版は既存のModeler・Writer/Acceptanceが担当する。SUIのCard/Graphからの候補、REAのExternal Candidate、Data WorkbenchのDB行も、Pinで相互参照できることと同じSave/Commitが許されることを区別する。外部AI・文書に含まれる命令は実操作の追加権限にならない。

## 23. 最小実装パケット：Pin Aの裏InspectがBの作業を変えない

11月14日にGitHubの現main実Sourceと[Pin状態のDraft #4597](https://github.com/hat47x/tei-platform/pull/4597)を照合した。

| 接合点 | 既存の事実 | 次の最小変更 |
| --- | --- | --- |
| `reference/tei-studio-workbench/main.ts` | 通常の`inspect(subject)`が`selected`、Editor、Graph、Noticeを更新する | Pin対象を確認する処理をForeground更新から分離する。B閲覧中のAの結果はAのPin欄だけへ |
| `reference/tei-studio-workbench/responses.ts` | 独立lane/epochで旧UI応答の採用を防ぐ。Host認可や実Cancelは提供しない | Pin専用laneを使い、Unpin/再Pin、Workspace切替、Disposeで旧結果を不採用にする |
| Draft #4597の`pin-state.ts` | Workspace、Subject、generationに基づく純粋なPin/Unpin/Undo/Invalidate。限定テストあり | UIの操作・実Product Reader・表示との接続。PinのgenerationとSource/Grant資格を混同しない |
| `internal/studiohost/readonly.go` | allowlisted Root→既存Product Inspectへのread-only入口 | 二つのInspectが並行してもWorkspace/Subject/Hostの既存判定を共通利用する |
| `reference/tei-studio-workbench/package.json` | 現mainの既存Node tests、型チェック、生成経路 | Pinの純粋状態テストとWorkbenchのDOM/UI契約を統合HEADで確認する |
| 生成JS・埋込JS・Windows WebView2 | 実Consumerへ届けるにはビルド経路が必要 | Source→esbuild→Embed→実Windows Window/IPCで別assert。局所テストPASSだけで製品完了としない |

**着手順**：U2-a＝#4597のPin状態を実Workbenchへ最小結線 → U2-b＝Bを表示したままPin Aの正常/失効/後着/失敗を同じUI Consumerで試す → U2-c＝Source TypeScriptと配信Bundle/Embed byteの一致 → U2-d＝Windows WebView2の実画面とキーボード/IME/支援技術。まず公開合成Source一件のみを使い、AI送信・Relation Pin・複数Actor編集は次段に残す。

現時点の#4597には純粋状態遷移の実装があるが、通常Workbenchへの結線はなく、過去のHEADではNodeテスト・型チェック・ビルドPASSと埋込Bundle byte不一致FAILが併存した。Bundle差異の原因を推測で確定せず、同一HEADの実Consumerで再検査する。

最優先のDesktopセルフビルドは独立に[#4583](https://github.com/hat47x/tei-platform/issues/4583)のnon-Plugin Studio Build Subject／B1 Builder入力・実呼出し／B2 Stage1実TEI→Stage2／B3 Stage2実WebView2の順とする。Studioを仮Pluginとして扱って形式的にBuildIntentを通したり、外部LauncherのBuildをStage1再入と呼んだりしない。[Toolchain観測 Draft #4648](https://github.com/hat47x/tei-platform/pull/4648)の`matched`は実CompilerのPlanへの正式Pin・Seed Trustを意味しない。

## 24. 11月14日の追加独立Expected（UX-35〜46）

以下は既存#1180/#2514のScenarioへ渡す**反例候補**であり、新しいSuiteや製品Hard Gateではない。過去のUX-01〜34を置き換えない。

| Case | 入力・故障 | 独立Expected | 既存Owner |
| --- | --- | --- | --- |
| UX-35 | 同じSubjectをDesktopとWebの異なる入口からInspect | Qualified対象・Source/Scope/取得状態の意味が一致。Actor/Grantは別 | #1824/#1158 |
| UX-36 | Editor部品がWails Bridgeの存在を必須とする | 表示/操作の共通部品は特定Desktop Transportを前提にしない | #1824/#997 |
| UX-37 | Local Desktopで許可されたRootを別Web Principalが指定 | DesktopのRoot/GrantをWebへ無音継承0 | #2516/#1824 |
| UX-38 | Reader結果0件、Native SQLの閉包はunknown | 全対象不存在の宣言0。戻り行数と被覆を別表示 | #2586/#3262 |
| UX-39 | RLS/Limit等の一部制限だけ観測 | 見えた制限だけpartial、秘密の存在/件数の漏えい0 | #2586/#2516 |
| UX-40 | AIがVisible Sumを全母集団の合計として提示 | 取得Scopeの値と未取得/不明を区別 | #2593/#2514 |
| UX-41 | B閲覧中のPin A refresh成功/失敗 | BのEditor/Graph/Dirty/Focus不変。AのObservationのみ更新 | #4515/#1824 |
| UX-42 | Pin純粋状態試験PASS、生成JS/実Windowは旧版 | UI製品受入は未成立。配信Bundle・Embed・Consumerを別検査 | #4597/#1824 |
| UX-43 | AI案R0、人間のDirty R1、別AgentのR0案 | Source条件付き競合を示し、無音Apply/Save/Build 0 | #1124/#2621 |
| UX-44 | StudioをPlugin Subjectに偽装してBuildする | non-Plugin Studio Buildの実Product経路を要求。擬似成功は不適格 | #4583/#2831 |
| UX-45 | 同じPlan、異なるGo toolchainでArtifact byte差 | Toolchain観測、Plan Pin、実Exec、Seed Trustを分離 | #2831 |
| UX-46 | 生成Windows Job成功、通常版Job失敗、Stage2未実行 | Job別表示。Workflow全PASS・セルフビルド完成と表示しない | #4583/#1824 |

### 今回確認した資格の上限

- [Desktop Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の11月14日観測では、生成DesktopのWebView2 Job成功と、通常DesktopのOS-version PowerShell timeoutによるWorkflow failureが併存する。これは当該HEADのJob別観測であり、Stage1→Stage2の実再入は未受理。
- [FunctionContext Draft #4647](https://github.com/hat47x/tei-platform/pull/4647)はW1〜W3の限定実PG観測で、Native SQL全依存・W4列由来・Product CLI・Studio/AI Consumerは別資格。
- [Toolchain Draft #4648](https://github.com/hat47x/tei-platform/pull/4648)の同Plan/異Go版の成果物比較は、どのCompilerでBuildしたかの限定的な観測。BuildIntentへの正式Toolchain Pinや実OS隔離の実証ではない。
- 選択したProvider/Generatorと実際に使ったProvider/Builder/Artifactの対応は[#1251](https://github.com/hat47x/tei-platform/issues/1251)／[Draft #4621](https://github.com/hat47x/tei-platform/pull/4621)の責務を維持する。選択UIの表示から実利用へ自動昇格しない。

**本追補はGitHub Source/Issue/PRの静的照合と設計文書更新のみ。** 新規Node/Go/Windows/DB/外部Harness E2E、Current Grant、正式API/Renderer採用、Ownerの業務意味受理はNOT RUN / NOT ACCEPTED。#4583のDesktopセルフビルドP0を変更せず、#4515のUX-23/41を並行して実装する。


## 25. 2026-11-14 第5次横断統合：Studioを「生成する一巡」と「使う一巡」

[Desktop統合 Draft #4650](https://github.com/hat47x/tei-platform/pull/4650)と[dogfooding再基準化 Draft #4651](https://github.com/hat47x/tei-platform/pull/4651)を、§17のDirty/Saved/Build/Runningの版区分に接続する。**異なる一巡の成功を合算して製品完成としない。**

| 一巡 | 工程と最小の完了条件 | 不成立時に許す主張 |
| --- | --- | --- |
| **G：Studioを生成する** | non-Plugin Studio Build Subject→選択Builder/実Invoker→TS/Go・Node/esbuild→Embed/Wails→**実Stage1 TEI**→Stage2 Windows Desktop→そのStage2のWebView2実Consumer。宣言/実消費/Process/Artifact/実Windowが同じBuild lineageで接続 | 生成部品・通常/生成Desktopの局所成功に限定。Stage1なしの外部Launcher二回BuildをG4としない |
| **U：Studioを使う** | Workspace→Catalog/Inspect→明示Pin→三要素モデル/診断/必要Read→typed Proposal/Preview→独立Expected→条件付きSource Save/既存Acceptance→再読取。同じ実Consumerで対象・出典・版を保持 | Pin/Source/Scope/Grant/独立Readが不明なら候補に留め、編集・実Effectの完成へ昇格しない |

Gは[#4583](https://github.com/hat47x/tei-platform/issues/4583)・[#2831](https://github.com/hat47x/tei-platform/issues/2831)の**製品P0**、Uは[#4515](https://github.com/hat47x/tei-platform/issues/4515)のPinned Observationを並行する最初のUX実装とする。UのAIなしread-only一巡はG4完了を待たずに試せるが、**生成Studioで実際に使えた**と主張するには実Artifact→配信JS→WebViewを別に確認する。

### Windows検証の最新訂正

[Desktop Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の修正後exact HEAD `9b58533df8764a74c578d478d8c6ee92a382642e`について、[Windows Workflow #43572235468](https://github.com/hat47x/tei-platform/actions/runs/43572235468)は**completed/success**、通常Desktopと生成Desktopの**両WebView2 Jobが成功**した。§24に記した旧HEAD `357345d7`の通常版PowerShell OS情報取得timeoutは歴史的な別Attemptの失敗であり、現HEADの通常版が未検証という表現へ引き継がない。確認時点では全12 Workflow中のHealth/Self-buildはなお実行中だったため、12/12 Greenとは主張しない。

この局所成功は、Stage1 TEIが正式non-Plugin BuildIntentを実行してStage2 Windows Studio exe全体を生成・使用したというGの完了とは異なる。[Builder Draft #4634](https://github.com/hat47x/tei-platform/pull/4634)・[Toolchain Draft #4648](https://github.com/hat47x/tei-platform/pull/4648)の別HEADを合算しない。

## 26. Source診断から修正・再検証へ：人間とAIに共通の作業体験

[DSL四層診断計画 #4652](https://github.com/hat47x/tei-platform/pull/4652)と[Source/実行前拒否試験 #4654](https://github.com/hat47x/tei-platform/pull/4654)をStudioの診断一覧、Sourceジャンプ、AI修正候補に接続する。**診断の種類をすべて「DSLエラー」へまとめず、実際に検査した層を示す。** 新Validator、第二Parser、承認Engineは設けない。

| 層 | Studio/AIが共通に保持する説明と操作 | 証拠上の停止線 |
| --- | --- | --- |
| L1 Source/Parse | Parserが検出した構文位置、Raw span、Sourceへのジャンプ | Source位置を持たないRuntimeエラーに架空spanを作らない |
| L2 Workspace/Meaning | Type/ID/Relation/Function/Business/Ruleの整合と必須依存、検査したSource・未取得範囲 | Parse成功や診断0件だけで意味受理・完全性を宣言しない |
| L3 Pre-dispatch | Input/Output/Outcome・Effect宣言の不整合、実Adapter呼出し前の拒否 | Adapter call=0を独立に確認。L1/L3合格をL2合格へ自動昇格しない |
| L4 Product Entry/Host | 公開Entry、実選択Provider、現在Grant、実InvocationとReceipt/Unknown Effect | Preflight PASSを公開・実呼出し・業務Effectへ昇格しない |

診断一覧は「対象」「層」「Error/Warning/Unknown/未検査」「実Source位置と再確認の必要性」「次にできる操作」を表示する。ドメイン専門家には業務上の意味、開発者には契約・依存とRaw span、AIには同じ型付き診断と既存コマンドを提供する。ただし**異なる利用主体へ同じ証拠を異なる資格として説明しない**。

例えば同名Inputは、現DSLが構文上Parseできても、Workspace全体の意味検査には未接続の可能性がある。L3では既存のCapability構造検査がAdapter呼出し前に拒否する。AIのQuick Fix後は、現在のQualified Subject/Raw Source/Revisionを照合し、修正Source→再Parse→再Workspace診断→Pre-dispatch→必要ならEntry/Hostを**実行した範囲だけ**表示する。ACD-01〜04の負例を再利用し、業務意味のOwner受理や正式公開Entryを自動生成しない。

### 診断の操作性と開発速度

入力直後の診断→Preview→Sourceの再投影→Build→実画面の各レイテンシを別々に測定する。どの操作がEditorのIME/UndoやGraph閲覧を止めたか、部分再計算・古いReply・誤対象表示があったかも記録する。Source/Meaning/Adapter/Hostの各段階を一つの“高速”指標に畳まない。Build・Cache・診断の既存Ownerを再利用し、新Build EngineやUI専用Semantic Cacheを作らない。

## 27. 46のUX観点を、実行可能な3つの作業票に集約する

UX-01〜46は設計検討の観点であり、**件数を製品実装率・独立試験Coverageへ換算しない**。新しい受入ケースを次々に増やすのではなく、次の三つを既存#1180/#2514のScenarioと独立Expectedへ割り付ける。

| 作業票 | 最初の正例 | 必須の負例 | Owner |
| --- | --- | --- | --- |
| **UX-PIN** | BのEditor/Graph/Dirtyを維持しながらPin Aを裏でInspectし、A欄のみ更新。AI OFFでも利用可能 | Unpin→再Pin後の旧Reply、Workspace A→B→A、Graph描画Edge IDの誤束縛、Bのカーソル/IME破壊 | #4515/#1824、Draft #4597 |
| **UX-DIAG** | DSL Source上の矛盾を一覧→実位置→人/AIの修正候補→再検証へつなぐ | 架空のSource span、L1通過をL2/L3通過と偽装、拒否なのにAdapter call>0、Dirty Bufferの無断保存 | #4359/#4495/#1124、Draft #4652/#4654 |
| **UX-READ** | 実Query結果を取得対象・読取条件・被覆・出典付きで提示し、AIも同じ確定度で説明 | 0件→不存在、RLS/LIMIT→全件集計、Git作業版とDB接続の混同、古いTool Replyの再利用、Current Grantの誤継承 | #2586/#3262/#2516、Draft #4644/#4647 |

**検査は三つに分ける**：①同一HEADのHeadless/純粋契約、②実Product CLI/Workbenchでの同一意味の観測、③必要な実Windows WebView2と外部Harness/異Actorの受領。別HEADの試験成功・Producerの自己申告を同一Attemptへ合算せず、実Toolの受領完了とIndependent Expectedを照合する。

実装優先順は①**UX-PINの最小UI結線**、②DSL Source→Workspace診断のFit/Gapが得られた後のUX-DIAG、③Product `tei` CLI→実Provider→Function Contextに到達した後のUX-READ。UI/Diffの試作を先にすることは可能だが、正式Product実Readの結果として表示しない。外部AI、SHO、SUIは各票の最初のHard Gateに含めず、AIなしH0を同じTaskの対照とする。

## 28. 外部製品・環境を横断するときの保存責務と採否

SUIのCard/Canvas、SHOのRaw Markdown/Occurrence、TEIのAccepted Model/Source、Data Workbenchの実DB行、Plugin Contribution、生成ArtifactとRunning Windowは、相互参照できても**別々の正本とWriter**を持つ。対象をPinして移動できることは、別のEditorやDBへ無条件で書き込めることを意味しない。SHOに保存した文書からTEIの現在Subjectを再解決する際も、Handoff Payloadの受領、TEI Readerの実再束縛、Owner Meaning Acceptance、実Tool Receiptを区別する。

Git作業版、DB接続先、Workspace、Principal/Grant、Plugin/Builder選択は独立した軸である。Gitを旧Commitへ戻しても実DBへのCommitや外部Effectが取り消されたわけではない。対象とRow値が同じでも、別DB/時点/Actorから旧Preview/条件付き更新を引き継がない。Environment/Binding・既存Source Writer/DB Provider/Receiptを再利用し、新DB Resolver、単一汎用状態表、協業専用承認Engineを設けない。

**今回成熟した採用対象**：二つの一巡の別検収、DSL四層の診断投影、三つの実行票への収束、Source/DB/Actor/Plugin/Buildの版・権限の区別。正式Activity JSON、共通Pin ABI、React/LitやWorker配置、WASM担当、複数AIの自動マージ、YIC専用Harnessの状態永続化は現時点では未決とする。

**実証状況**：今回の§25〜28は、2026-11-14の他チャット・現行GitHub Issue/PRを照合したdocs-onlyの設計・受入計画。新しいGo/Node/Windows/DB/AI試験、Product実Provider呼出し、Source Save、Owner意味受理、正式Schema採用、Studio Stage1→Stage2新一体Runは**NOT RUN / NOT ACCEPTED**である。
