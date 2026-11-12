# GUI/CUI共通「固定対象・文脈棚」設計・実証計画（2026-11-12）

> **状態**：設計・検証計画／Ownerの正式採用前。新しい公開ABI、State Store、認可基盤、Harness Runtimeを採用した文書ではない。ここに記すPinned Selection GUI・CUI・実Harness接続・再起動復元・Product E2Eは**NOT RUN**。
> **主担当**：[TEI #1824](https://github.com/hat47x/tei-platform/issues/1824) StudioのWorkBench UI/Adapter。[TEI #1124](https://github.com/hat47x/tei-platform/issues/1124) Modeler対象・Source往復。
> **参照先**：[#1158](https://github.com/hat47x/tei-platform/issues/1158)/[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)共通Operation、[#3419](https://github.com/hat47x/tei-platform/issues/3419)契約、[#1120](https://github.com/hat47x/tei-platform/issues/1120) Identity、[#2497](https://github.com/hat47x/tei-platform/issues/2497) Projection/逆写像、[#2516](https://github.com/hat47x/tei-platform/issues/2516)用途相対Context/開示、[#2621](https://github.com/hat47x/tei-platform/issues/2621) Source/Raw Draft、[#2391](https://github.com/hat47x/tei-platform/issues/2391) GUI外部制御、SHO [#4](https://github.com/hat47x/sho-composer/issues/4)/[#19](https://github.com/hat47x/sho-composer/issues/19)/[#31](https://github.com/hat47x/sho-composer/issues/31)、TAI [#168](https://github.com/hat47x/tai-pleroma/issues/168)/[#271](https://github.com/hat47x/tai-pleroma/issues/271)。
> **基線**：TEI main `0a4d3e3132cd34d9d67689608a54b30fbd97ca2c` を2026-11-12確認。実製品がこの文書の形式を読めることは未確認。

## 1. 利用価値と採否の問い

1. 人は**今見ている場所**と**AIへ指示した対象**を混同せず、複数のTEI Data/Function/Business、関係の特定Edge、Graph/表の構成要素、SHO本文Occurrence、出典、SUI Card/Islandなどを仕事の対象として固定できる。
2. 画面切替、カーソル移動、IME変換、Esc、空白クリック、プロンプト編集、別Viewの選択、非同期再描画によって固定対象を失ったり、別対象へ無断変更したりしない。
3. 一つのHarness Sessionへ複数製品から出入りするとき、固定対象の**意図・参照・未解決事項**は必要に応じて引き継げるが、Sourceの現在性、Grant、Disclosure、意味受理、外部実行権は都度確認する。
4. 対象と目的の再説明回数、誤った対象へのAI作用、範囲指定に必要な操作数、復旧手数を、通常の一時選択+プロンプトと同一Taskで比較する。固定選択の複雑さが人の仕事を遅くするなら、既定の軽量性/段階開示を見直す。

**「優れたGUI」は見た目の機能数ではなく、正しい対象・適切な文脈・撤回可能な操作を、少ない認知負担で扱えること**とする。3ペイン固定、常時AI Chat、大きなCanvasや常駐Agentを必須UIとしない。

## 2. 状態・責務・操作の分離

| 層（概念） | 何を意味するか | 典型的な変化 | 何を意味しないか |
| --- | --- | --- | --- |
| Focus | 現在の入力先、hover、カーソル位置 | 入力・クリック・Tab | 固定対象 |
| Transient Selection | 行・文字列・Node等の一時範囲 | 別の範囲選択、Esc、Editor移動 | AIに渡すことの承認 |
| **Pinned Target Set** | 意図して追加・除外・役割付与した**参照の集合** | 明示追加/削除/Undo、別版への明示再束縛 | 現在の内容・権限・外部送信 |
| Task/Activity Context | 目的、対象間の役割、比較、除外、前提、出典、未決、作業段階 | 利用者が編集するTask票のrevision | Source/Meaningの第二正本 |
| Harness Session | あるHarness/Provider/Tool Profileによる会話・操作系列 | 接続/切断/Session分岐 | Productの選択や権限の継承 |
| Operation/Attempt | 製品/Host所有の実Request、Receipt、観測 | Preview、Call、送出、Receipt | 画面で操作を望んだだけの状態 |

Pinned Target Setは **Intentとして安定し、現在性としては失効し得る**。表示したチップが残っていることと、AI/Providerへ今も利用可能であることを別々に描画する。Pinした参照を秘密な生Sourceのキャッシュとして使わない。

Pinは選択候補の受理であり、OwnerによるBusiness Meaning受理ではない。「選択範囲を固定」はボタン・キーボード・CUIから同じ意味になるよう設計する。AI側が対象候補を増やす場合も、自動Pinせず「候補を追加」の提案を出す。

## 3. 固定する対象の必要情報（仕様候補であり新Canonical Schemaではない）

- **誰の何か**：Product/WorkspaceのQualified Identity、SubjectのStable Identity、対象kind/Role、対象の関係Identity（Graph Edgeでは方向と多重辺のID）、Document/Occurrence、Canvas配置と元対象の区別。
- **どの部分か**：Source identityと正規化前Raw locator、意図したSource revision、Dirty Buffer revision、解析可能なraw UTF-8 byte span（SHO GUIのUTF-16 offsetと別）、Expression/Table cell/Field/Relation path等の適切なgranularity。
- **どの時点/ビューか**：Business valid-time、Snapshot/cohort、Projection/Renderer版、View scope、表示の多対一写像/非可逆性・Loss、取得のcomplete/partial/unknown/not-fetched。
- **どう使うか**：主対象・比較A/B・関連・根拠・除外のTask-local役割、目的（調査/説明/変更案/外部作用）、要求する閉包、期限、Audience。機密情報そのものではなく適格な参照を保持する。
- **何が未確定か**：同名異ID・多重Edge・旧Span・Source移動・Path再束縛・A→B→A、未解決Relation、Plugin/Reader不在、Scope不一致、未取得/非開示、未検証ReviewとCurrent Grant。

同じ表示文字列でも原Sourceが異なることがあり、同じID/World FingerprintでもSource locatorやGrantは変わる。表示座標、ファイル名、行番号、Label、Digest単独を安定対象の唯一の識別子としない。**安定した人間記入IDの必須化**も要求しない（既存Identity解決を利用）。

## 4. 静的集合と動的集合は別の操作

**A. 明示した個別IDの固定**：例えば「Order.total」「OrderLine→Orderの特定Edge」「設計書の当該Occurrence」を3個の明示参照として保持する。候補の増減は明示操作だけ。各参照は後のSource再資格化でstale/ambiguousとなり得る。

**B. 条件の固定**：例えば「Function Xが到達する二段以内のRelation（方向=outbound）」「2026-11-01時点で有効な兼務」「R1とR2の変更差分」「選択したDataへのC/U/D影響」。これは**Predicate/Query意図を固定**するだけであり、現時点の対象ID全集合が固定されたと主張しない。実行/送出前に必要なら現在のConsumer/Scope/Providerで条件再評価し、件数・多重度・時間条件・集計/欠落・非開示を確認する。

**C. Snapshot選択**：Bのある時点の評価結果を利用者が「このN個を別の固定集合にする」と**明示確定した場合だけ**A型へ変換。部分集合/unknown/非開示を含むなら完全なN個という主張を拒否する。旧Snapshotと現在Query結果の差分を表示し、勝手にAを更新しない。

集合の追加・差集合・交差・比較役割はTask文脈の操作であり、Source/Business Relationを編集しない。Graphの単純な隣接展開が、Function Data Contextの必要母集団やQuery完全性の証拠になるわけではない。

## 5. UIとCUIの具体動作

### GUIでの既定動線

- 左：Project/Outline/Model、中央：Graph/Table/Source/SHO Editor、右または下：**折畳み可能なContext Shelf**。通常の執筆/モデル確認時にはShelfを小さなバーへ縮退し、SHOのIME・Undo・入力遅延、TEI Graphの大画面を優先。
- 一時Selectionは通常のEditor/Graph規則に従う。ピン留めは明示した「作業対象に追加」で行う。選択後にPromptへFocusを移してもPinは残り、Escは現在のPopup/一時操作を閉じるだけでPinを消さない。
- Shelfの項目は**名前/種別/役割/参照版/現在性の要約**を示す。必要な詳細はInspectorへ展開。秘匿対象の名称/数/存在は現在の表示資格がないAudienceへ出さない。
- 対象単位の削除はUndoでき、全消去は明示した「固定対象をすべて解除」。重要な対象が複数ある場合は確認と復旧可能性を示す。Undoは**ローカル選択編集だけ**であり、過去の外部送信・Host作用や正式Source保存の取消ではない。
- プロンプト入力中も固定対象の一覧・役割・除外を参照できる。プロンプト中の自然言語「これ」「さっきの」はPinned参照への候補Bindingとして説明し、曖昧なら確認する。文字列からの暗黙Pin/Scope拡張を許さない。
- タッチ/キーボード/スクリーンリーダーから同じ操作へ到達できる。ドラッグ・Ctrl+クリック専用にせず、追加・除外・並べ替え/役割変更・Context確認の明示操作を提供。aria-describedby等には許可された対象情報だけを出す。

### CUIでの同等操作（**表記例であり正式CLIではない**）

```text
context target add --subject sales.order.total --role primary
context target add --relation-id REL-042 --role related
context target add --document docs/design.md --occurrence OCC-7 --role evidence
context target query --source sales.order.total --direction outbound --depth 2 --role candidate
context inspect --scope current
context preview --consumer external-harness --purpose impact-analysis
context share --consumer chosen-session   # preview と現在の許可を再確認
```

CUI操作は既存Application Operationの意味/Client Adapterから導く候補とし、表示のためだけに新しい巨大な統一dispatcherや別認可器を作らない。形式やサブコマンドの正式化は[#1158](https://github.com/hat47x/tei-platform/issues/1158)/[#3419](https://github.com/hat47x/tei-platform/issues/3419)の既存契約検証後とする。

### GUI/CUIとAIの双方向操作

通常の「選択→AIへ説明」だけでなく、**AIによる対象候補の提案→GUI上のプレビュー→人による補正→同じTaskへ反映**を独立した操作として扱う。例えばAIは「このFunctionと参照する2 Relationを同時に調べるべき」と提案できるが、ツール結果には候補のqualified identity、根拠、Source版、集合被覆、Loss、現在Scopeを付け、GUIは**既存Pinned集合との差分**を示す。AIが選択を増減するのは人間のPin/Unpinと等価な明示操作を受けた後であり、既定で変更されない。

提案の可視化はハイライト/一時オーバーレイでよく、正式なRelation追加やCanvas上の物理配置を必要としない。SUIではCard/Island/Tree/Canvasの複数表現を保持し、CSWの新しいまとまり提案は**探索候補**として提示する。TEIのData/Business Relation、SHOの章構成、SEIの根拠評価を暗黙に書き換えない。特に同じ島に配置されたことと、同じ組織・Function・Source identityであることは別。GUIは「提案対象をプレビュー」「候補からPinへ追加」「今回は除外」「変更なしで質問を続ける」を明示し、CUIも同じ操作結果を受け取る。

複数のActivityを同時に扱う場合、Pinned集合の意図した所属TaskとTask revisionを示す。タブの移動で別Activityの対象を黙って共有しない。**現在Focus中のTask**と**外部Harnessが実際に受領したContext版**も別々に示し、古いAI応答が別Taskの現行候補を上書きしない。

## 6. 対象固定の持続性・再束縛

「選択が消えない」は**見た目でチップが存在すること**よりも、「ユーザーが指定した対象意図が、意図せず別の対象にならないこと」。保持対象は階層ごとに分ける。

| 契機 | 固定Intent | 現在のContent / 操作資格 |
| --- | --- | --- |
| 空白クリック・Esc・IME・タブ切替・Paneの閉鎖 | そのまま保持 | 変更なし。ただし別のSource状態通知は別扱い |
| Source改版 R0→R1、Source移動 | 旧Bindingを保持 | stale / 再解決。自動Relinkしない |
| 同名異ID、複数Occurrence、多重Edge | 候補と選択意図を保持 | ambiguous。編集/送信は保留 |
| A→B→A、外部Editor Save、Dirty Buffer変化 | 元選択履歴を保持 | byte復帰だけで過去の受理/送出許可を復活させない |
| Scope/Grant/Policy変更、Plugin撤去 | 参照の継続可能な最小情報のみ保持 | 利用資格を失効させ、許可されない内容を現画面・DOM/ARIA/Tool JSONから撤去/非表示 |
| Window再起動・別端末・共有Harnessへ移動 | 保存/再開が提供されるなら版と復旧状態を表示 | 作業票復元は実参照/権限の再評価まで未資格。同期ACKと相手が実読取したことは別 |

初期段階で無理に中央Context Storeを作らない。**同一プロセス内**のタブ間保持から実証し、再起動時の復旧は既存Workspace/Document Draft機構の適格性を確認した後の独立ケースとする。「永続化したはず」と見せて保存根拠がない場合はnot persistedを明示。機密参照の保存期間/暗号化/削除は既存Product/Hostの保存ポリシーに従う。

## 7. AI送出・結果受領・実作用のフェンス

```text
UI Focus/Transient Selection
  → User-committed Pinned Intent
  → Task Context（目的、比較、除外、範囲、未決）
  → Product-owned Source/Projection Resolution
  → Current Disclosure & Required Coverage Check
  → Actually-to-be-shared content/bytes/args Preview
  → Explicit Handoff to one external Harness Session
  → Actual Receive/Tool Result (observed separately)
  → Candidate/Explanation/Proposal
  → TEIなら既存 Preview / Shared Acceptance / reload
     SHOなら Raw Draft / Writer / reopened disk read
     外部Effectなら Host Grant / Attempt / Receipt / independent observation
```

「Pinしてある」「Tool read-only」「Harness Session継続」「Source digest一致」「モデル表示済み」は、今の外部送信/Actual Effect/意味受理を保証しない。Rendererからtrusted RootやAuthorityを直接受け取らず、現在のProduct/Hostで再解決する。

**スコープ/開示**：必要な対象のQualified Public Sliceだけを送出候補にする。隠れたSourceの個数/名前/存在/Path/IDも、Context Shelf・Autocomplete・候補数・Tooltip・診断・AI Tool discovery・送信ログ・HTML/ARIAへ漏れないことを二世界比較する。旧Previewでの許可はSend直前のCurrent Disclosureに代わらない。送信実体のbyteと「送信したつもり」の表示も分離。

**後着**：Source/Buffer/Selection/Task/Consumer/Policyのepochを結果がどの版で計算されたかに結び付ける。旧応答をCurrentのタブに挿入しない。受領済みの旧結果は許可される範囲で履歴として残し、再検証まで現在の作業結果へ昇格させない。

## 8. 第1受入の実証パケットと敵対的負例

新しいScenario SSOTは作らず、[#1824のCTX-01〜08](https://github.com/hat47x/tei-platform/issues/1824#issuecomment-6792128219)と[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)、SHO #31 S1、TAI #271 ASVを同じ独立Expectedへ接続する。下表は**追加観点**であり、まだ製品試験を実行した結果ではない。

| ケース | 敵対的操作 | Expected / 何を否定するか |
| --- | --- | --- |
| CTX-01/UX-A | Node + 同端点多重Edgeの1本をPin、別タブ/Inspector/SHOへ移動、空白クリック/Esc/IME | PinのEdge ID/Role/Source版不変。Focusを固定対象と混同しない |
| CTX-02/UX-B | Data属性、Function、Business Action、SHO Occurrenceを主/根拠/比較A-B/除外で固定 | SourceとRole/Occurrenceの独立性を保ち、表記名で上書きしない |
| CTX-03/UX-C | Graph集計/非単射、同名異ID、複数Occurrence、Sourceに対応しない図 | candidate/ambiguousで止まり、直接編集・自動Relinkしない |
| CTX-04/UX-D | R0→R1、移動/ABA、Scope/Grant失効、R0 Async遅延応答 | Intent継続・現在資格失効・既表示内容非開示・後着不採用を別assert |
| CTX-05/UX-E | 静的ID集合と条件指定集合を比較し、Readerがpartial/未取得/非開示 | 条件を「N件確定」とせず、Queryを黙ってSnapshotへ昇格しない |
| CTX-06/UX-F | Harness未接続、外部Plugin未導入、SHO単独/AI OFF | Selection管理とRaw Markdown保存が独立して成立し、未送信を送信済みとしない |
| CTX-07/UX-G | J公開/S非公開の二世界＋別Audience | Tool args/Prompt/DOM/ARIA/件数/Log/候補の隠れた差を拒否。現在Grantの実判定は別 |
| CTX-08/UX-H | CLI→Studio→SHO→TAIへ用途限定Handoff | Identity/役割/SourceとTask時点を照合。Live Session継続とActivity再開は別観測 |
| UX-I | Pinした後に編集で同じ文字列を複製・移動、外部Editorが競合Save | Source anchor曖昧なら再束縛保留。SHO UTF-16→UTF-8 byte span/BOM/CRLFとR0/R1を比較 |
| UX-J | Pinを一つ誤削除→Undo、全削除→復元、送信済み後にUndo | ローカルPin履歴のみUndo、過去の送信/実EffectはUndoで取り消された扱いにしない |
| UX-K | AIが「この関連も含める」と提案、Preview後にScope/Policyが改版 | AI候補を自動Pinせず、送信直前のDisclosure再確認。古いPreviewの実送信0 |
| UX-L | 同じ選択で一般Editor範囲選択のみ vs Pinned Shelf/CUI vs 外部Harness Adapter | 再説明手数、対象間違い、元Source到達、修復/Undo、外部送信過多、Latencyを同一Taskで比較。値は実測まで空欄 |

**必要な証拠列**：exact Workspace/Source/Buffer/Projection/Task revision、Product/Consumer、Pinned intent before/after、Actual authorized target set（許可された範囲で）、Reason/Unknown、操作Eventと実UI DOM/ARIA、外部送信/受領byteまたはdigest/Scope、OS/Tool実観測、保存/Writer/Host effectとOwner判断を別列に保持する。Snapshot一致だけで現在Permissionを保証しない。

## 9. 実装分割・依存順序・停止条件

| Slice | 既存Owner | 最小成果 | 停止/成功条件 |
| --- | --- | --- | --- |
| P0 read-only（先行） | #1120/#1158/#3419/#2516・#1824 | 既存Inspect/Source経路でqualified Subject/Relation/Occurrenceの対象票を組み立てる。CLIとStudioに入力できる候補 | Reader/Scope未資格ならunsupported/unknownを保持。新Parser/Resolver/Hostを実装しない |
| P1 UI pinned interaction | #1824 + #1124 | 同一ConsumerでFocus→Pin→別View→Inspect→明示解除/Undo。Graph/List/Command初期選択の実イベント合流 | 旧DOMクリック/Async/ABAで違うIDへ乗換え0、失効した表示と許可情報の混同0 |
| P2 SHO document+SUI | SHO #4/#19/#31、SUI #3391 | 文書Occurrence/byte anchorを軽量Editor内でPinし、SHO単独保存、SUIに必要な選択だけをHandoff | SHO Reader/Writer・Storageの実格付けを確認。TEI Sourceへの自動書戻し0 |
| P3 Task Context/AI adapter | TAI #168/#271、TEI #2516/#2391 | 既存Harnessの一SessionまたはActivity再開へ公開SliceのみPreview→実受領を限定観測 | 機密のOS/Host隔離G0・G1/G2を満たさない環境で外部送信しない |
| P4 typed proposal/Agent co-work | #1124/#2497、SHO #4、TAI #271 | AI提案→可逆性/Scope/Impact→既存AcceptanceまたはSHO Writerへ戻す | 元Source以外の書戻し禁止。選択した対象への反映とCurrent Grant/Owner受理/実Effectを別受入 |

**進行順の上限**：P0/P1は既存TEI #1119のCritical Pathおよび実Function→Provider Read #2586／Plugin D1 #1251をBlockしない。P2/P3はSHO G1b/G2/G3とTAI H2/H3の先行受入を無断で変更しない。Eclipse GLSP/CodeMirror/Wails等はUI候補であり、採用・依存・新Native Hostをこの文書で固定しない。Wails v3は既存#1824の第一候補のまま。

## 10. 別の見方からの反証（採用前に必ず確認）

- Shelfが重く常時目立つことで、SHOの文章入力や通常のModelerで探索を阻害しないか。SHO単独のAI OFFで価値を持つ構成は維持されるか。
- 固定対象が安全に保存できても、原Sourceと現Grantの再解決待ちが多すぎ、ユーザーが意図しない古い対象を使い続けないか。安心感と実安全の区別をUXとして説明できるか。
- 条件選択・複数役割・依存展開・Task Scopeを全て開示するとUIが複雑化する。1対象1目的から始め、必要時に展開できるか。
- 対象Chipに出典Path/秘密名/Countを出してよいか。非公開の存在や参照をRoleに応じて出さない必要性を実二世界で確かめられるか。
- 複数ハーネス/製品で一枚のTask票を引き回すより、必要最小の参照だけを渡す方が誤継承・Token使用を下げるのではないか。B0/B1/B2の同一Task比較で見直す。
- CUIの自動操作がGUIの偶然のFocus選択へ依存しないか。GUIもCUIも同じTask/Source Scopeを明示して再現できるか。

**未決事項**：Pin/Task票の永続化方法・保持期間・復元のUX、静的/動的選択を正式に表現する型、外部Harness接続形式、可視化の優先順位、Keyboard shortcut、外部送信Previewの粒度、Owner/OS別のDisclosure。この文書は設計案であり、比較と実Product試験なしに正式採用しない。

## 11. 2026-11-12 横断照合：実装済みの文書往復とGUI統合の不足辺

### 11.1 新しい実装基線を区別して受け取る

| 実際に確認したGitHub統合 | 本計画に再利用できる範囲 | まだ証明しない範囲 |
| --- | --- | --- |
| [TEI #4317](https://github.com/hat47x/tei-platform/pull/4317) merged | 受理モデルの一要素/一関係からSHO向け文書Draftを生成し、保存・再開・再確認する最小経路 | 全Data/Function/Business意味保持、GUI操作の往復、外部Host/実DB |
| [TEI #4467](https://github.com/hat47x/tei-platform/pull/4467) merged | 文書再投影時のマージ方針を既存実装へ接続 | 著者文・図・出典・多重Relation・部分欠落を含む一般ケースの安全性 |
| [SHO #36](https://github.com/hat47x/sho-composer/pull/36) merged | TEIモデル由来の文書Draftを消費側で開き、編集し、保存・再開する経路 | Studio埋込みの操作全体、同時編集の強い競合解決、Owner受理 |
| [SHO #40](https://github.com/hat47x/sho-composer/pull/40) merged | Studio組込み形態を測るためのランナー | Studio/SHO統合GUIの実E2E資格、デスクトップ上の実利用者操作 |

この四つは**局所実装の統合**であり、各PRのCIを相互に足して「同じHEAD/AttemptのGUI往復PASS」としない。具体的な最新統合コミットは各PRのMerge Commitを参照する。GUI初期実装では既存DocumentDraft/Writer/Readerを**外側から操作するAdapter**を優先し、Editor内に独自のDraft Merge/TEI Meaning Serviceを再実装しない。

[TEI #4484](https://github.com/hat47x/tei-platform/pull/4484) はFunction→実SQL/Provider/RowsetのP0と文書投影の資格を横断主線へ整理した**未マージDraft**。本設計は#4484のData Context資格を先取りしない。実DBから未取得の値を、三視点/SHOで表示できたからといって取得済みと扱わない。

### 11.2 実画面の具体的な状態遷移（同一作業例）

**題材**：受注Function `order.submit` と `OrderLine` への特定Relationを調べ、CRUD表からSHO設計書の一段落へ説明を追加し、AIに影響範囲を確認する。

| 画面/操作 | 見せるもの | 実操作・不変条件 |
| --- | --- | --- |
| Modeler / Data | Node・typed Relation、同端点多重Edge、現在Source版、partial/unknown | Edgeを一つ選び「作業対象へ固定」。位置やハイライト変更はPresentationのみ |
| Modeler / Function | Functionと宣言されたRead/Create/Update/Delete、Source到達 | 固定したRelationとFunctionを別のQualified Identityとして保持 |
| Modeler / Business | Procedure/TaskActionとFunction参照、manual/external action | Derived potential pathを実Effectと同一表示しない |
| CRUD / Cross-reference | Cellの根拠Edge/Function、Declared/Derived/Observed区分、coverage | Cell→構成要素→Sourceへ移動。集計Cellから一意なSourceへ逆編集できなければread-only |
| SHO Composer | 受理モデル由来のDocumentDraft、編集本文、引用/Occurrence、Raw/Dirty/保存済み版 | 著者が追記した文章はSource意味に自動書戻ししない。TEI更新時は再投影差分/衝突を提示 |
| Context Shelf | 主対象・根拠・比較/除外、Source版・現在資格・用途 | Focus/Tab/IME/Prompt編集でPinを解除しない。別Taskへの暗黙コピー禁止 |
| AI/Review | 実際に共有するPublic Slice、根拠と未解決、提案差分 | Current Disclosureを送信前再確認。AI提案はPin/Meaning/文書への自動適用ではない |
| Save/Accept | SHO Draft Save、TEI Meaning Acceptance、Host Effectを別の操作/Receipt | Source byte一致、Writer Save、Owner受理、実Provider Effectは別の資格として表示 |

**UX原則**：全画面に常駐する巨大なInspectorではなく、操作した要素の**必要な状態だけ**を表示する。SHO単独では執筆幅/IME/Undoを優先し、外部モデルやHarnessのReaderを編集Hot Pathで自動起動しない。ModelerはData/Function/Businessを独立した第二正本にせず、同じSourceの役割別Projectionとして扱う。

### 11.3 GUI状態の最小契約（正式API型ではない）

各操作で、画面の**状態名**を単一の成功/失敗に圧縮しない。例えば以下は独立した表示軸であり、UI側の新しいAuthorityではない。

- **Draft**：clean / dirty / saving / saved / conflict / unknown。保存した事実はTEI解釈の成功を示さない。
- **Resolution**：current / stale / ambiguous / partial / unavailable / permission-denied / unknown。PinしたIntentが残っていても現在の操作可否は別。
- **Projection**：source-linked / derived / lossy / not-supported / unknown。Diagram表示できることと安全に逆編集できることを分離。
- **Review**：candidate / previewed / accepted / rejected / not-applicable / unknown。選択をPinしたことはOwner Meaning Acceptanceではない。
- **Handoff**：not-requested / previewed / dispatched / acknowledged / received / failed / unknown。Harnessへ送信したことと相手が正しく受領・利用したことを分離。
- **Execution**：not-requested / proposed / authorized / attempted / observed / unknown。文書保存やAIの「実行しました」という回答をHost EffectのReceiptに昇格しない。

複数軸の合成による「全部OK」バッジを既定で設けない。UIでの短縮表示は、利用者が行う**次の操作に必要な不一致と根拠**を優先する。

### 11.4 再投影のユーザー体験と保存競合

1. 既存TEI Source版R0から文書Draft D0を生成し、SHOがD0を編集・保存する。保存済みDocument RevisionとDirty Buffer Revisionを別に保持。
2. TEI SourceがR1へ変更されたとき、R1から再投影したD1を、**D0原型・SHO著者編集・D1生成候補**の三者として比較する。二者差分で著者編集を消さない。
3. 既存#4467の再投影方針を利用し、再生成可能な領域と著者編集領域、参照版/出典、曖昧な対応付け、削除された対象を分類。自動適用可能性は個別に検証する。
4. 古いSource/Document版、外部Editorの競合Save、同名別Occurrence、BOM/CRLF、未取得のRelationがあれば、ユーザーに根拠付き候補と保留理由を表示。**新しい版の全自動Merge成功を既定にしない**。
5. 適用後は実disk再Read、元Source/文書のidentityとbyte/Revision、Projection、Citation/Referenceを再確認。TEI Meaningの更新は別のShared Acceptance Boundaryへ送る。

SHO→SUI選択Handoffの[SHO #31](https://github.com/hat47x/sho-composer/issues/31)では、元文書のUTF-16選択と原文UTF-8 byte span、R0 diskとDirty Buffer、選択外の秘匿情報を別扱いにする。GUIのPinは**未保存Bufferの選択**も表現できるが、実diskのSourceを示すように偽装しない。外部送出可能な内容の決定は既存SHO/Hostの現在資格に委ねる。

### 11.5 統合GUIの追加実証（既存Expectedへの差分）

| Case | 実操作の連鎖 | 独立Expected（現時点はNOT RUN） |
| --- | --- | --- |
| GUI-X1 | Relation Edge固定→Function/Business/CRUDへ移動→SHOへ引用→戻る | Edge ID・Source位置・版が一致し、同端点の別Edgeを誤採用しない |
| GUI-X2 | TEI R0→SHO D0編集→TEI R1→文書再投影→保存再開 | 著者文・根拠・未解決を保持し、無断上書き/Source逆書戻し0 |
| GUI-X3 | 日本語IME・BOM/CRLF・UTF-16範囲選択→Pin→外部Editor Save | byte span/Document Revisionを再確認し、旧Spanから別Occurrenceを誤選択しない |
| GUI-X4 | Data/Function/Businessで同一Subjectを異なるラベルで表示→AIが関係候補提案 | 既存Pinと候補を分離し、ユーザーの明示採用前のPin変更0 |
| GUI-X5 | CRUD derived cell、部分取得、Function実Read未接続 | 「Declared/Derived/Observed」「partial/unknown」を維持し、実DB ReadやBusiness Effectの証拠にしない |
| GUI-X6 | CLI/headless、Studio Web、SHO standalone、Harness未接続を切替 | 同じ対象参照を適格な経路で再解決。未接続でもSHOのRaw SaveとTEIのread-only inspectは各々独立 |
| GUI-X7 | 保存中にSource/Grant変更、AI旧応答後着、Handoff途中切断 | Source/Task/Disclosure epoch不一致を拒否し、送出/受領/保存の各Receiptを別記録 |
| GUI-X8 | 画面遷移/Inspector表示/外部Editor往復の利用者試験 | Pinの誤解除数、対象再説明回数、目的の再確認手数、誤編集数、SHO入力遅延をB0（固定なし）と比較 |

GUI-X1〜X8は既存#1824 CTX、#1124 M、#1180 Scenario、SHO #4/#31のExpectedへ対応付ける**Cross-view受入票**であり、別Runner/第二SSOTを新設しない。まずX1・X3・X5をread-only/公開Fixtureで実施し、Writerや外部AIが必要なX2・X7を後続とする。

### 11.6 横断責務の最小境界

- **TEI**：Source/Meaning/Typed Operation、Data/Function/Business/CRUD Projection、可逆性・Impact、Shared Acceptance。Function実Read/Plugin Provider資格は#2586/#1251のP0が別途所有。
- **SHO**：Raw Markdown/文書Occurrence/著者編集/保存再開/再投影の消費側。モデルを読んだだけで第二Meaning Authorityを作らない。
- **SUI/CSW**：Card/Island/知識構造と探索提案を必要最小のSelection/Handoffとして渡す。配置や近接性をTEIの業務Relationと混同しない。
- **SEI**：根拠評価と推論の候補。採否/Owner Meaningを代行しない。
- **KON**：適格な保存・版・権限のサービス候補。採用済みの万能Host/中央Registryであるかのように要求しない。
- **TAI/外部Harness**：Activity/Session継続と必要なQualified Contextの受渡し。各製品のAuthority、OS/Host隔離、Grant、Source資格を継承したとみなさない。

### 11.7 次に着手する実装の順序を狭める

**最小GUI縦切り**：実Source上の一つのRelation EdgeをCLI/headlessでqualified inspect → StudioのData viewでPin → Function/Business/CRUDへ移動 → Inspector/Context Shelfで同じEdgeへ戻る → Source改版/同名異ID/後着を拒否。SHO Editor/外部Harnessの存在はこの受入の前提としない。

**第2縦切り**：#4317/#4467とSHO #36/#40の既存実装を再利用し、受理モデル一要素+一関係→SHO Draft→著者編集→保存再開→再投影差分の実UI Consumerを測る。まず既存Headless/Reader/Writerを呼び、別のGraph Parser/文書Merge Engineを追加しない。

**第3縦切り**：SHOの選択OccurrenceとTEIのRelationをTaskの比較/根拠として固定→送出予定Public Slice Preview→実外部Harnessで選択的Handoff→戻った候補をGUI上で明示採用/却下。OS/Host隔離とDisclosure Gate未資格なら外部送信は行わずローカル候補で止める。

**計画反映先**：この節は#1824 Studio GUI、#1124三要素分析法の各モデル、SHO #19文書UI、SHO #31 Handoff、TAI #271 Activityの横断リンクを担当。MASTER_PLAN作業領域Gの内容は既存PR #4325/#4484のレビュー後に最小リンクのみ追加し、同じ規範を複数文書へ複写しない。



## 12. 2026-11-12 横断成熟版：問い・時点・業務意味を固定する操作仕様

> 本節は#4473の三要素分析法のモデル・CRUD/問い相対の検証、#4500/#4501の組織C1と時点、#4472の現実装CRUD欠陥、TEI→SHO実Draft、SUI/CSW/SEI/TAIの活動継続を**GUI/CUI操作に接続**する補足。これらのOwnerのExpectedやMeaning正本を複製しない。独立Expectedの実実行は**NOT RUN**。

### 12.1 方法論の名前と、汎用TEIへの写像の境界

渡辺幸三氏の**三要素分析法**とX-TEA Modelerに示される**データモデル・機能モデル・業務モデル**の相互牽制を基準とする。三つのTEI Role（Data/Function/Business）へ単純分類したことだけで、この方法論を実現したとは主張しない。データモデル図、機能モデルの画面/階層/CRUD、業務モデルの作業/手順、クロスレファレンス、サブシステム境界などは各々原典対応を検討する。[設計統合Draft #4473 §16.14](https://github.com/hat47x/tei-platform/pull/4473)が原典とTEI拡張の差異判定を所有する。

本GUIのSelectionは「三要素分析法のモデルの一要素/関係」を出発点にできるが、SUI Card、SHO Occurrence、時系列Query/Graph、拡張的REA等への連続参照は**TEI独自のUI/Projection拡張**である。原典準拠機能・TEIによる異表現・TEI独自拡張・未照合という差異は設計/検収に残す。旧図形の見かけが同じことを方法論互換と呼ばない。

### 12.2 重要な修正：Pinned Targetは対象IDだけでなく「問い」を束縛する

単なる`{workspace,kind,id}`の固定は不足する。人が選ぶのは**問いに対する対象の意図**。次の区別をTask Contextの参照として保持する（**新しい公開Core Schema/永続ABIではない**）。

| 分類 | 最小の参照値 | 固定しても保証されないこと |
| --- | --- | --- |
| 意味対象 | qualified Workspace / kind / subject / relation occurrence / Source location + revision | 原典方法論への完全対応、現行Meaning受理 |
| 表示対象 | View kind / Projection rule version / Canvas instance / UI occurrence / lossy mapping | 一意な意味対象・編集逆写像 |
| 問い | purpose / asked relation or CRUD / required closure / expected output grain | Query完全性、対象集合の確定 |
| 時点 | validAt（業務有効時点）/ knownAt（その情報を把握した時点）/ Source・Rule・Case revision | 各Providerの同一Read Cohort、過去Receiptの変更 |
| 責務 | Membership・Case Responsibility・Approval Candidate・Handoff ACK・Host Grantを**別Relation/評価** | 所属からの権限付与・暗黙の責任移管 |
| 利用と開示 | Consumer/Audience/Purpose、現在のDisclosure・Source資格を参照する方法 | 旧Grant再利用、外部送信/実Effect |
| 作業継続 | Task/Activity・固定意図のrevision・Result correlationと送信済みAttempt | 同一Harness Session、相手側の受信・業務成功 |

一例は、組織C1の「**2026-11-01を有効時点とし、11-12時点で判明している、C1案件の責任移管に影響するFunction/Relationを調べる**」。同じC1 IDを固定しても、有効日・把握日・責任Rule版・Queryのgrainが異なれば、同じ問いの結果とみなさない。日時は独立Synthetic Fixtureの設定例であり、実組織データや現在の状態を示さない。

### 12.3 固定集合の代数：意図・Query結果・適格な公開集合を分離

**UIでの候補式**（実装済みAPIではない）：

```text
Q = evaluate(query intent, scope, validAt, knownAt, reader revision)
E = explicitly excluded target identities
A = explicitly added target identities
RequestedSet = (Q - E) union A
ReleasableSet = authorize_and_project(RequestedSet, consumer, purpose, current grant)
```

ここで**Qを未観測なら空集合にしてはいけない**。Qは`complete | partial | unknown | unavailable`と各要素の根拠・coverageを伴う評価結果であり、`RequestedSet`もQ未完了なら「確定全集合」ではない。Aの要素の存在・最新性・版は別途再資格化する。EとAで同じQualified Identityを要求した場合は、その選択が同じ対象/役割/Scopeを指すことを確認し、**競合をユーザーへ示して明示解決まで外部Apply/Sendを保留**する。数式の演算順を暗黙の業務優先順位としない。

Qを現在の**観測済み集合**へスナップショット化する操作と、条件そのものを固定する操作は異なる。Qの件数上限、`total/omitted`、continuation、unresolved、non-output SQL dependencies、秘密フィールドの除外を保持する。隠れたQの件数やSourceの存在を無権限Audienceに見せない。表やグラフで省略した辺を「関係なし」と扱わない。

**操作の同等性**：GUIは「この要素を固定」「この条件を固定」「今回の評価結果を別集合として固定」を別Actionにする。CUIは同じ既存Qualified Reference/Query intent/Task revisionを明示して行う。CUIの新構文を公開APIとして先に採用せず、#1158/#3419の操作発見・既存CLIとの互換検証を先行する。

### 12.4 Pinの継続と安全な利用を分離する最小状態遷移

二軸状態機械の**候補**。Pin Intentの保持は`unpinned/pinned`、参照と利用資格は`current/stale/ambiguous/partial/unknown/withheld`などで説明する。「一状態のreadyフラグ」で統合しない。

| 発生イベント | Intent | 参照・開示の次状態 | 許容操作 |
| --- | --- | --- | --- |
| Pinを明示追加 | pinned | 最初はunqualified、実Reader/Current Grantで確認後current等 | ローカル保持、必要なread-only再解決 |
| Focus/Tab/IME/Esc/空白クリック | pinned維持 | 現資格を**操作だけで変えない** | 表示/別Viewへのnavigate |
| Source/Meaning/Case/Rule valid-known改版・移動・ABA | pinned維持 | stale / ambiguous / unknown | 元Bindingと履歴は保持、Resolve/Compare、適用停止 |
| Query範囲/粒度/Reader cohort変更・partial観測 | pinned維持 | partial/unknown | 不足理由・継続要求、確定全集合としてSend/Applyしない |
| Grant/Policy/Recipient/Disclosure変更を観測 | pinned維持（開示可能な参照のみ） | withheld/unknown/currentを現在Grantで再判定 | 許されない既表示本文/DOM/ARIA/Tool候補を現在表示から撤去。漏洩不能は実検証が必要 |
| AI候補/旧Async Reply/Tool途中切断 | pinned維持 | 現行対象は変更なし、旧結果は候補/unknown | Correlation/epoch付きで隔離、Preview、明示候補追加 |
| Unpinを明示→Undo | 指定履歴から復元 | **unqualifiedへ戻して再判定** | ローカル参照のUndoのみ。旧Grant/Effect/送信履歴をUndoしない |
| 新しいActivity/別Harnessへ移動 | Taskへ属するPinを勝手に移動しない | 受け取りProductの現在資格で再解決 | 最小Handoffと受領観測。暗黙のTask混合を拒否 |

重要な**連続性**は三つに分割する：Context Continuity（誰が何を問い、どの参照を保持したか）、Resource Continuity（Source/Document/ProviderのIdentityと版・再読取）、Operation Continuity（Request/Attempt/Result/Receipt/Effect）。一つの成功を他の成功へ昇格しない。対象の**権限が未知になったのに旧内容を表示し続ける**ケースは、単に赤いstaleバッジを付けるだけでは安全ではない。

### 12.5 C1組織・CRUDモデルの意味の取り違えを先に潰す

同一Synthetic Case `C1`で、P1の所属D1→D2が起きてもC1のhandlingUnit/Responsibilityが更新されない場合を使う。表示上は同じP1/D1/D2/C1へ到達するGraphでも、「所属」「Position」「Case取扱部門」「責任」「承認候補」「通知」「ACK」「実Grant」はそれぞれ異なる対象をPin可能とする。valid/knownを変更した時点別結果、Treeの隠れた第二親、別のRelation occurrence、二経路で重複する集計grainをread-onlyで確認する。Orgの独立Expectedは[#4490](https://github.com/hat47x/tei-platform/issues/4490)/[#4501](https://github.com/hat47x/tei-platform/issues/4501)に従い、この文書に第二Org Scenarioを作らない。

**先行して潰す具体的な実装リスク**：[TEI #4472](https://github.com/hat47x/tei-platform/issues/4472) は現`internal/threeelement`における`Index()`のData/Business索引誤用、明示のみCRUDDefinitionの脱落、同一Pairの後勝ち矛盾隠蔽、`ok`判定と矛盾の共存を列挙している。修正前のCRUD/横断参照結果を、GUIが表示できたという理由で正しい対象集合としない。GUIのPin/Cell→Sourceを検証する前に、#4472 TE-CR-01〜04の独立Expectedへ接続し、Data ID/Relation occurrence/Function×Data/Declarative vs Derivedの根拠を照合する。**#4472の修正・試験Ownerは変えない**。実証前にまだ欠陥が残る場合はGUI側のCRUDに`unverified/unsupported`を表示し、CellからSourceへの可逆編集を止める。

### 12.6 GUI/CUI/Harnessの最小縦切りと合否を固定する

既存CTX/UX/PIN/GUI-X/CX/TE-CRを新しい試験群として増殖させず、**実際に走る最小の3段**として照合する。

| スライス | Product入口（現物のOwner） | 最小の独立Expectedと停止線 |
| --- | --- | --- |
| **S-A：一つの正しいRelationを保持** | 既存TEI `model locate`/Catalog/InspectとModeler/Studio (#1124/#1824)、#4472 CRUD | 同名異ID・同端点多重Edgeのうち特定の1本をPin→AI入力へFocus→Data/Function/Business/CRUD/Sourceを移動→同一Qualified ID/Source revisionへ戻る。誤解除0、誤再束縛0。CRUDが未修正なら正しいと自称しない |
| **S-B：問い・時点・集合を変える** | C1合成Fixture (#4490/#4501)、#3262/#2516 Context | validAt固定でknownAt変更、partial/omitted、Tree第二親非表示、Case責任と所属の非一致。Pin意図は維持し、Q結果とMeaning/Current資格は再評価。実二時点Queryが未実装なら`NOT PROVEN` |
| **S-C：文書・Harnessへ戻す** | merged TEI #4317/#4467、SHO #36/#40、TAI #168/#271 | 同じ選択の出典と必要範囲だけをSHO Draftへ投影→著者編集/再投影差分/独立保存再Read。外部HarnessへはPublic Slice Preview→Current Grant→実Send/Receiveを独立実証。AI OFF/Adapter absentではローカル執筆が継続 |

**実装コードの第一変更対象はS-Aの`1 Relation + 1 Consumer + 旧Reply`のみ**。Graph Rendererや新AI Sessionを先に全面再実装しない。S-B/S-Cの内容がS-Aを阻むHard Gateではない。既存P0（Plugin実Invoke #1251／Function実Read #2586）にPin UIを前提として追加しない。

**記録する値**：Expected ID、exact HEAD、対象`workspace/kind/qualified id/occurrence/source revision`、Source/Policy/Task epoch、validAt/knownAt、Query grain/coverage/omitted/unknown、UI event before/after、Focus/Selection/Pin intent、Correlation/Result revision、送信したexact Public Slice、実Receiver/Receipt、SHO disk再Read、既存Owner Meaning Acceptanceの有無。公開先には現在許された証拠列だけ出す。

**可視化とアクセシビリティ**：タッチ・キーボード・支援技術でも明示Pin/UnpinとTask切替へ到達できるか、IME確定前の一時入力でPin/Source Applyが発火しないかを独立に検証する。AI入力欄/SHO Editorの通常の作業面積をContext Shelfが常時奪わない。非信頼Markdown/SQLコメント/Tool Replyの自然言語命令はデータとして扱い、HostのAuthority/Tool Egressへ昇格しない（#4473 §16.13とTAI #232に従う）。

**停止線**：Source/Meaning/Permissionの現在資格を観測していない、実Reader/Providerが呼ばれていない、対象集合がpartial、CRUD解釈に未解決矛盾がある、同名/同Hash/行位置だけで参照が一致する、外部送出の対象や権限が不明、実Writer/Host Effectの独立Observationが欠落する場合は、その範囲を`candidate/unknown/not-proven`と表示し、作業全体を「完了」にしない。これは設計上の制約であって、製品で実測済みの防護機能ではない。

### 12.7 統合判断前の比較・撤回条件

- **既存UIとの比較**：VS Code的な一時Selection+Prompt、固定Context Shelf、GUIに依存しないCUI/Headlessで、同一Taskの再説明回数・誤対象Apply・Undo回数・関係の到達率・IME/入力遅延・支援技術での到達性・無断送出を比較する。値がない段階で「生産性改善済み」と書かない。
- **設計の優先順位**：対象精度と現在資格が最優先。その次に入力の軽さ、読解性、段階的な深掘り、製品間継続性。必要な情報の追加に比例してUI常時表示を増やさず、Context Shelfの折畳みと選択理由のInspectorで両立する。
- **撤回する案**：新しい中央Context Registryや専用Harness Hostなしでは実現しない案、既存Modeler/SHO Writerと二重にSource/Meaningを保持する案、全Projectionを自動逆編集可能とする案、無権限の秘密件数/PathをContext Chipで漏らす案、Business Authorityを組織所属から自動導出する案。一般化の根拠を示すまでOwnerへ戻す。

**反映先・依存関係**：本設計#4479は「GUI/CUI操作と実証順」のOwner。原典忠実性と三要素分析法の各モデルは#4473、CRUD実装欠陥は#4472、組織C1の意味とBehaviorは#4500/#4490/#4501、Studio実画面は#1824、文書Writer/再投影はSHO #4/#19、AI Activity/Harness隔離はTAI #168/#271/#232。MASTER_PLANには各Draftの適格レビュー後に**リンクを一つ追加するだけ**とし、計画正本を並行PRで上書きしない。
