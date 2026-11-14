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

[TEI #4484](https://github.com/hat47x/tei-platform/pull/4484) はFunction→実SQL/Provider/RowsetのP0と文書投影の資格を横断主線へ整理した**未マージDraft**。本設計は#4484のData Context資格を先取りしない。実DBから未取得の値を、三要素分析法の各モデル/SHOで表示できたからといって取得済みと扱わない。

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

**最小GUI縦切り（2026-11-12修正）**：現Workbenchが実際にInspect可能な一つのQualified **Subject** をHeadless/Studioで照合→Studioで明示Pin→別Subject/Focusへ移動→元Subjectへ戻る→Source改版/同名異ID/後着を拒否。**Relation occurrenceのPinは現Graphのparent表示線で代用せず、正式なSource-backed Relation identityとConsumerが揃った次段**とする。SHO Editor/外部Harnessの存在はこの受入の前提としない。

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

**UIでの候補式**（実装済みAPIではない）。**集合形成とQ/E/A・Predicate/Snapshotの共通設計Ownerは [TAI Draft #629](https://github.com/hat47x/tai-pleroma/pull/629)**。ここではTEI Studio上の操作導線・適格な参照と現在Source/Model/CRUDの再解決だけを所有し、集合代数を独立仕様として再採用しない：

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
| **S-A：まず一つのQualified Subjectを保持** | 既存TEI `model locate`/Catalog/InspectとModeler/Studio (#1124/#1824)、具体実装 #4515 | 一意なSubjectをPin→Focus/別Nodeへ移動→同じWorkspace/Subject/Revisionへ復帰。現Graphの描画用Edgeから意味RelationをPinしない。Relation多重辺/CRUD Cellは#4472とSource-backed Identity確認後の次段。誤解除0・誤再束縛0 |
| **S-B：問い・時点・集合を変える** | C1合成Fixture (#4490/#4501)、#3262/#2516 Context | validAt固定でknownAt変更、partial/omitted、Tree第二親非表示、Case責任と所属の非一致。Pin意図は維持し、Q結果とMeaning/Current資格は再評価。実二時点Queryが未実装なら`NOT PROVEN` |
| **S-C：文書・Harnessへ戻す** | merged TEI #4317/#4467、SHO #36/#40、TAI #168/#271 | 同じ選択の出典と必要範囲だけをSHO Draftへ投影→著者編集/再投影差分/独立保存再Read。外部HarnessへはPublic Slice Preview→Current Grant→実Send/Receiveを独立実証。AI OFF/Adapter absentではローカル執筆が継続 |

**実装コードの第一変更対象はS-Aの`1 Subject + 1 Consumer + 旧Reply`のみ**（[implementer #4515](https://github.com/hat47x/tei-platform/issues/4515)）。Graph Rendererや新AI Sessionを先に全面再実装しない。S-B/S-Cの内容がS-Aを阻むHard Gateではない。既存P0（Plugin実Invoke #1251／Function実Read #2586）にPin UIを前提として追加しない。

**記録する値**：Expected ID、exact HEAD、対象`workspace/kind/qualified id/occurrence/source revision`、Source/Policy/Task epoch、validAt/knownAt、Query grain/coverage/omitted/unknown、UI event before/after、Focus/Selection/Pin intent、Correlation/Result revision、送信したexact Public Slice、実Receiver/Receipt、SHO disk再Read、既存Owner Meaning Acceptanceの有無。公開先には現在許された証拠列だけ出す。

**可視化とアクセシビリティ**：タッチ・キーボード・支援技術でも明示Pin/UnpinとTask切替へ到達できるか、IME確定前の一時入力でPin/Source Applyが発火しないかを独立に検証する。AI入力欄/SHO Editorの通常の作業面積をContext Shelfが常時奪わない。非信頼Markdown/SQLコメント/Tool Replyの自然言語命令はデータとして扱い、HostのAuthority/Tool Egressへ昇格しない（#4473 §16.13とTAI #232に従う）。

**停止線**：Source/Meaning/Permissionの現在資格を観測していない、実Reader/Providerが呼ばれていない、対象集合がpartial、CRUD解釈に未解決矛盾がある、同名/同Hash/行位置だけで参照が一致する、外部送出の対象や権限が不明、実Writer/Host Effectの独立Observationが欠落する場合は、その範囲を`candidate/unknown/not-proven`と表示し、作業全体を「完了」にしない。これは設計上の制約であって、製品で実測済みの防護機能ではない。

### 12.7 統合判断前の比較・撤回条件

- **既存UIとの比較**：VS Code的な一時Selection+Prompt、固定Context Shelf、GUIに依存しないCUI/Headlessで、同一Taskの再説明回数・誤対象Apply・Undo回数・関係の到達率・IME/入力遅延・支援技術での到達性・無断送出を比較する。値がない段階で「生産性改善済み」と書かない。
- **設計の優先順位**：対象精度と現在資格が最優先。その次に入力の軽さ、読解性、段階的な深掘り、製品間継続性。必要な情報の追加に比例してUI常時表示を増やさず、Context Shelfの折畳みと選択理由のInspectorで両立する。
- **撤回する案**：新しい中央Context Registryや専用Harness Hostなしでは実現しない案、既存Modeler/SHO Writerと二重にSource/Meaningを保持する案、全Projectionを自動逆編集可能とする案、無権限の秘密件数/PathをContext Chipで漏らす案、Business Authorityを組織所属から自動導出する案。一般化の根拠を示すまでOwnerへ戻す。

**反映先・依存関係**：本設計#4479は「GUI/CUI操作と実証順」のOwner。原典忠実性と三要素分析法の各モデルは#4473、CRUD実装欠陥は#4472、組織C1の意味とBehaviorは#4500/#4490/#4501、Studio実画面は#1824、文書Writer/再投影はSHO #4/#19、AI Activity/Harness隔離はTAI #168/#271/#232。MASTER_PLANには各Draftの適格レビュー後に**リンクを一つ追加するだけ**とし、計画正本を並行PRで上書きしない。

## 13. 2026-11-12 「選択形成→対象再解決→限定共有」へ実装責務を収束する

> **判定**：§1〜12の一般原則は保持するが、TEI Studioは共通Selection AlgorithmやAI Skill RouterのOwnerではない。GUIの具体的な実装・試験を先へ進めるため、**一つの実操作と観測をつなぐ接合契約**だけを本節で固める。すべての例は採用前・実Product GUI E2E **NOT RUN**。

### 13.1 並行Draftの正本を一つずつ特定する

| 責務 | 現在の設計/実装Owner | Studio側の受取り方、重複を作らない範囲 |
| --- | --- | --- |
| 選択集合の形成：自然言語Q、利用者の除外E/追加A、PredicateとSnapshot、View別Occurrence | [TAI Draft #629](https://github.com/hat47x/tai-pleroma/pull/629) | **利用者がGUIから操作できるか**を検証。集合演算・汎用Target ID/Selection ABIをStudioで再定義しない |
| Productの現在参照再解決、開示Preview、Handoff v1、外部Harness応答の戻り先 | [TAI Draft #633](https://github.com/hat47x/tai-pleroma/pull/633)、[TAI #271](https://github.com/hat47x/tai-pleroma/issues/271) | Studioからのread-only Source/Target資格・相関情報を確認し、Product既存Operationへ渡す。**旧Draft #630は未マージ終了**、引用/復元しない |
| Source/Workspaceの実同一性と改版観測 | [TEI Draft #4504](https://github.com/hat47x/tei-platform/pull/4504)、#1120/#2620 | SourceとModelのFPだけが一致してもWorkspace ID・実Source/Manifest・Revision・Targetは別。A→B→Aで昔のGrantやPin利用資格を復活させない |
| 三要素分析法のモデルFingerprint・AuthorityLimitの純粋性 | [TEI Draft #4507](https://github.com/hat47x/tei-platform/pull/4507)（#4475）、CRUD [#4472](https://github.com/hat47x/tei-platform/issues/4472) | FPをCurrent GrantやSource closureとみなさない。現実装のData/CRUD誤対応と権威制限欠落を解消する前に「対象が正しい」と自称しない。**#4507は未マージ、Goテスト未確認** |
| SHOの文書選択・Raw Span・保存と明示再読取 | SHO #4/#19、[SHO Draft #38](https://github.com/hat47x/sho-composer/pull/38) | **永続Occurrence IDを前提にしない**。#38の案(b)は選択箇所をRaw Span/版で都度解決する判断候補。Editor GUIが独自の永続IDを自動発行してSource正本化しない。strict root/Storage fenceは実Consumer側の資格 |
| SUI Card/Island/TreeとSHO Draftの選択・往復 | [SUI Draft #3437](https://github.com/hat47x/sui-sensemaking/pull/3437)、SHO #31〜33 | KJ/Outline探索はAI OFFでも利用でき、TEI Function ContextをすべてのSUI Itemへ強制しない。KJの配置をTEIのRelationへ自動昇格しない |
| Skill/Tool起動と比較 | [TAI main #631](https://github.com/hat47x/tai-pleroma/pull/631)、Draft #633、[#168](https://github.com/hat47x/tai-pleroma/issues/168) | **Tool-onlyを既定**とし、必要時のみProduct Skill→条件付きTAI Skillを選択。Studioが常駐Agent/Skill Routerを持つことを必須化しない |

実装やテストの存在と、Owner受理・main統合・現在利用資格は別。並行Draftが閉じた/再編されたらリンク/状態を改訂し、古い実証表を継ぎ足して合算しない。特に#4504の実Read負例は**UI PinやHost Grantの実証ではない**。

### 13.2 GUIからの最小操作シーケンス

**一度だけ行う事前条件**：公開合成Workspaceの限定Datasetに、同名別Subject・同端点多重Relation・同じSource Rangeに別Subjectが来る改版を含める。既存#4504 fixtureと#4472 TE-CRの独立Expectedを使い、別Fixture/Second Registryを作らない。

| UIイベント | Productでの実操作 | 合格条件 / 見えてはいけない錯覚 |
| --- | --- | --- |
| E0: Focus+一時Selection | 既存Data/Function/Business・Source/CRUDのViewを使う | 表示座標/ラベルは候補の位置にすぎず、Source編集対象を決定しない |
| E1: 明示Pin | 現行Readerが一意に解決できるSubjectのQualified ReferenceをTask-localへ保持。Relation instanceはSource-backed identityが確認できるまで別段階 | Focus/Esc/IME/別タブで変化0。Pin時のSource/Task epochも保持。Context Shelfを閉じても対象が残る |
| E2: 対象Inspect | 既存TEI `model locate`・Catalog/Inspect等を**実Consumerが**起動し、Source/Workspace/版とOccurrenceを再解決 | ID/Kind/Workspace/Source・モデル版を対照。同FP別Workspace/同offset別Subjectの意図しない再束縛0。未実装Consumerならunsupported |
| E3: View横断 | Data→Function→Business→CRUD→SHO/Sourceの関連投影・Navigation | Rendererや行位置ではなく同一Qualified関係に戻る。CRUDの矛盾・未取得・Lossがある間はread-only |
| E4: Ask/Preview | Productが許可した、現在のSource/Reader/Task/Purpose相対Public Sliceを**実際に共有する候補**として表示 | Pin/解決成功だけで実送出できない。開示/Recipient変更、部分取得、秘密Count/Path・Tool argsを検査 |
| E5: Send/Receive（後段） | Tool-onlyから実Harnessへ必要な範囲だけ渡す。実Tool ReplyとOrigin View/Taskを相関させる | Preview成功と送出/受領成功を別Receipt。Tool-onlyでは不足が測定された時だけSkillを段階追加。TAI #168 H2/H3を新Pin試験でBlockしない |
| E6: Proposal/Apply（後段） | AI候補を元Modelerへ表示し、必要ならSource Patch Preview→既存Acceptance | 同じPin/Undoで旧Grantや完了Effectを復活させない。Source/Role/Scope/Host資格の更新とDocument Writerの保存は別 |

E1→E4を**実外部Harnessなし**で先行実証する。E5/E6をStudioの初回出荷ブロッカーや、TEI全体のProvider P0-R・Plugin D1の前提にしない。

### 13.3 「選択は残るが内容は出してはいけない」状態を先に設計する

UIの安全性を「選択チップが残る」「staleバッジを表示した」だけで判定しない。たとえばSource R0で許可されていた説明本文を画面で表示中、SourceがR1へ移った場合と、**SourceがR0のままでDisclosure/Grantだけ失効した場合**を別入力で観測する。

1. **Intent継続**：そのTaskのPinは明示解除/Task削除されない限り元の参照意図として残す（公開可能なメタデータの範囲で）。対象を勝手に新IDへ変更しない。
2. **既表示の現在資格**：Reader/Policyの変更を実Consumerが観測した後、許されない旧本文・Tooltip・関連数・DOM/ARIA・Tool JSON等を**表示/送信できる状態に残さない**。観測できないオフライン環境では即時失効の実証を主張せず、再確認できるまで現在利用資格はunknown。
3. **後着結果**：E2/E4を開始したSource/Task/Disclosure epochと、結果が戻ったときの利用条件を比較。古い結果を今の対象として表示しない。旧結果の履歴保全はAudienceに許された範囲に限る。
4. **原文と編集Buffer**：SHO DraftのR0実disk版とR1 dirty bufferを区別。GUIが元のDocument/Occurrence IDを永続型だと仮定せず、原文Raw Spanと保存/Save As/相対参照の新しいSource Identityを再解決。
5. **実Resource Identity**：Windows/WSL等の異なるパス表現、Symlink/実path変更、二つのWorkspaceに同じSemantic Sourceを置いた場合でも、見かけのパス/FP/Subject ID一致だけで現File/Host Rootの同一性を認定しない。実解決先は各Product/Hostが観測し、UIからTrusted Rootを組み立てない。

この5つは新State Serviceではなく、既存Source/Host/Disclosure/SHO Writerが所有する資格を**画面操作へ戻す規則**。

### 13.4 実装担当への最小作業票とEvidence

| 実施単位 | 達成物 | 検査・拒否条件 |
| --- | --- | --- |
| **G-01 Pin操作** | Studio同一画面でInspect資格のある1 SubjectをPin→Focus移動→戻る。Tooltip/ARIA/Keyboard/IMEでも意図が一定（実装票 #4515） | 同名異ID/別Workspace同FP/Source改版/後着、Esc、空白クリック、Task切替、Undoの正負。無断解除0/別対象への再束縛0。描画用Edgeを意味RelationとしてPinしない |
| **G-02 Source実再解決** | G-01のPinを既存Headless APIへ渡し、Workspace/Manifest/Source/Qualified Subject/Revisionを対照 | #4504の同offset別Subject・同FP別Workspace・Source only改版・ABAをProduct入口で拒否。旧テストの局所PASSをGUI実験の結果と混同しない |
| **G-03 CRUDと投影** | 三要素分析法のデータモデル・機能モデル・業務モデル/CRUDを照合し、正式なRelation occurrence IDが取得できて初めてEdge Pinの可否を判定 | #4472 TE-CR-01〜04は未修正ならunverified、#4507のmerged修正はFingerprint入力破壊の限定解消であって現在権限の証明ではない。Graph表示用parent Edgeから自動的に逆編集可能としない |
| **G-04 Context Preview** | 同じTask/対象をCUIとGUIで同等にread-only指定し、送出する予定の最低限のPublic Sliceを表示 | Qのpartial/unknown/hiddenを0件にしない。Audience/Source/Grant変化後の旧Previewを送らない |
| **G-05 SHO接合（後段）** | 一要素・一Relation→既存DocumentDraft→SHO編集/保存再Read→更新時の著者編集保全 | 永続Occurrence ID不要。SUI未導入/AI OFFでも保存継続。Source Writerへの自動書戻し0 |
| **G-06 実Harness接合（後段）** | 外部ToolでTask/Target/Origin Viewを用途限定Handoffし、Receipt/返信先を実観測 | #168のOS/Host隔離資格なしで秘密Sourceを送らない。Session復帰≠Activity復帰≠実Tool受領 |

G-01〜04に必要な入力は最小の公開合成Fixture。G-05/G-06は既存SHO/TAIのOwnerで並列化できるが、GUIの試験だけから実Product/Hostの適格性は推定しない。

**実証票の最小列**（既存#1180/#2514／TAI Handoff/SHO Ledgerへリンクするだけで新SSOTは作らない）：Fixture ID / exact main+PR HEAD / Product+Consumer+View / Task ID+Revision / Pin Intent before+after / Focus event / Qualified Target+Workspace / Source Revision+Manifest / Required Read+Coverage / validAt+knownAt / Disclosure Recipient+Epoch / actual Product Operation / Result correlation / Actual Writer/Tool/Host Receipt / independent Expected+Observed / Limitations。

### 13.5 評価順と不採用条件

1. **UIなしでも成立する同等指定**：GUIから選択できる対象がCUI/Headlessの正式な適格参照で再解決可能か。最初のG-01だけをリッチUIの外観で代用しない。
2. **原典に沿う利用価値**：データモデル・機能モデル・業務モデルの相互牽制とCRUD/Sourceへの戻りが改善するか。X-TEA Modelerの三要素分析法にない機能はTEI独自拡張と明記する。
3. **人間の負荷を増やさない**：AI OFF、単一Pin、Context Shelf折畳みから開始。同一Taskの一時選択のみとの比較で、再説明数・誤対象・復旧操作・入力遅延・外部送信過多を測る。実測前の目標時間/改善率を捏造しない。
4. **Hook/Pluginの重複導入を拒否**：G-01〜G-04のために中央Pin Store、第二Source Index、第二CRUD Engine、AI専用Host/Policy、常駐Agent、共通Skill Routerを新規導入する案は、必要性の独立証拠がない限り採用しない。

**進行上の優先**：全TEIの最初のFunction→実Provider→Consumer（#2586）、Plugin実Invoke（#1251）、TAIのG0/G1/G2と独立H2/H3 4試行（#168）を妨げない。G-01〜04は実装可能な部分からread-onlyで進める。#629/#633や各Product実装が未マージの期間は、そのDraftを参照実装・正式ABIとして扱わない。

**現在の状態**：本節は設計・実装担当への既存Owner割付と受入条件の整理のみ。新コード・本節G-01〜06の実GUI/CLI/Harness/OS実験・Owner正式採用・製品受入は**NOT RUN**。


## 14. 2026-11-12 実ソースで判明した「描画Edge≠意味Relation」と最初の実装票

> **現行main静的照合の更新**。本設計の§11〜13にあった「一つのRelationを初日からPinする」を、[implementer #4515](https://github.com/hat47x/tei-platform/issues/4515)へ引き渡せる能力別の順序に修正した。**欠陥の改善／Productテスト合格を確認した追補ではない。**

### 14.1 現実に存在する操作経路

| 役割 | mainの現実装・利用価値 | この経路で**まだない**保証 |
| --- | --- | --- |
| `reference/tei-studio-workbench/main.ts` | `selected: Subject|null`、`inspect()`、`workspaceEpoch/inspectionEpoch`、`responses.run("inspection")` | 明示Pin/Undo/Task所属、Source/Policy失効通知との結合 |
| `reference/tei-studio-workbench/presentation.ts` | `readCatalog/inspectEligibility/readInspection`でSubject/WorldFP/SourceIndexFPを比較し、`relatedGraph()`で親子の描画線を合成 | Workspace実Rootの同一性、実Relation occurrence ID、Read Closure/Current Grant |
| `reference/tei-studio-graph/graph.ts` | Graph Node選択は`onSelect(nodeId)`、Edge IDは表示を識別する内部値として扱える | そのIDのTEI意味Relation一意性。Edgeをクリックして原Sourceを一意に特定する操作 |
| `reference/tei-studio-workbench/responses.ts` | laneのRequest順序・reset/dispose後の旧応答を表示しない | Host操作のCancel/Effectや、既表示DOM/ARIA/Tooltip/Tool JSONの現在Disclosure失効 |
| `cmd/tei-studio/assets/client.mjs`・`cmd/tei-studio/main.go`・`internal/studiohost/readonly.go` | `inspectSubject(workspaceID,kind,subjectID)`→`StudioBridge.InspectSubject`→allowlist RootのProduct Reader | RendererからTrusted Rootを指定する権限、新たな一意Relation Resolver、AIの外部送出資格 |
| `cmd/tei-studio/assets/workbench/main.js` | 上記Source TypeScriptから生成する実ブラウザBundle | 手修正してSourceに逆反映する経路 |

特に`relatedGraph()`は、Catalogの`parent`から表示上の線ID `parent:${subjectKey(child)}`を作る。**同端点に意味的に異なるRelationが複数存在しても、それらをこのIDで一意に識別できるとは限らない**。Graph上で線が見えることと、Relation instanceのPin/編集資格は明確に分離する。RendererのEdge IDを業務意味のStable IDとして使わない。

### 14.2 いま着手するべき実装境界

- **G-01a / Subject Pin**：既存`inspectSubject`でQualified Subjectを照合し、現在の一時`selected`と独立した**Workbench-local Intent**を一件保持。Focus移動/別Node選択/Esc/IMEで意図維持。Pinは新しい共有Session/Aggregate Schemaではない。**Parent #1824、実装票 #4515**。
- **G-01b / 実Source再解決**：Workspace/Manifest/Source/Subject/Revisionの別軸を検査し、[Draft #4504](https://github.com/hat47x/tei-platform/pull/4504)の同offset別Subject・別Workspace同FP・A→B→A負例をUIから通す。旧Pinを他Workspace/Subjectへ自動付替えしない。
- **G-01c / Relation Link**：ModelerからSource-backed Relation occurrence identity、重複/多重性、Source range、Loss/coverageを返せる**既存**Operationを確認するまで、描画線に対する正式Pinは`unsupported/unqualified`。必要になった最小Product Adapter変更は#1124/#2497 Ownerへ。CRUD根拠の誤対応は[#4472](https://github.com/hat47x/tei-platform/issues/4472)を先に受入。
- **G-02 / 外部文書・Harness**：SHOの永続Occurrence IDを新規必須とせず、Document/原文byte span/Revision/Buffer再解決を既存SHO Writer/Readerで扱う。TAIの対象集合形成は[Draft #629](https://github.com/hat47x/tai-pleroma/pull/629)、Product別再解決・Handoff実証は[Draft #633](https://github.com/hat47x/tai-pleroma/pull/633)。この分担をTEI側で重複実装しない。

### 14.3 Productに戻す判定表と停止線

| 実験 | 成立と報告できる最小条件 | 成立とは呼ばないもの |
| --- | --- | --- |
| 1 Subject Pin→別Focus→元Inspectorへ | Pinの`workspace/kind/id`が変わらず、実Inspectの対応を確認 | 現Grant、最新Sourceの永久保証、Graph Relation一意性 |
| 同じ`kind/id`の別Workspace・同FP | 元Workspaceに束縛されたPinの意図と現在再解決が別である | FP一致のみで同一Resource・Authorization |
| 同offsetのA→B→A/Source only改版 | 古いReader応答を現在Targetに反映せず、履歴と現在資格を区別 | 旧Digestに戻ったことによるMeaning/Grantの自動復権 |
| Node描画選択と表示Edgeを対照 | Nodeは実Inspectで照合できる範囲でPin、描画Edgeは意味Relation Pinを拒否 | `parent:`という文字列をRelation IDとして正式公開 |
| Source不変でGrant/Policyだけ変更 | 変更がProduct Consumerで実観測できれば、既表示内容と外部送信資格を**別々に**更新 |`responses.ts`のlane抑制だけで漏洩防止を実証 |
| AI OFF・SHO未接続 | Pin/Inspectorのread-only操作が継続 | AIやSHOの常駐を必須化 |

**着手順**：まず#4515のSource TS上の局所State・実Consumerで正負対照 → Node/TypeScript `typecheck/test/build` → GitHub上の生成物と実Browser/CLI試験 → exact HEAD/独立Expectedで検収。外部Harnessや実Source Applyを初回のP0要件へ混ぜない。すでに[merged #4507](https://github.com/hat47x/tei-platform/pull/4507)のFingerprint副作用解消があっても、構造FPがCurrent Authority/Disclosureを保証するわけではない。

**受入条件の不変部分**：X-TEA Modelerの三要素分析法（データモデル・機能モデル・業務モデル）の互いの根拠確認を重視し、汎用TEIの追加Graph/SHO/SUI構造は拡張と表示する。実モデル上で「任意の辺をPinできた」と表記するのは実Relation identity/Sourceへの一意解決・独立負例が成立した後に限る。

**状態**：本追補はmain実ソースの静的照合、既存計画の順序補正、implementer Issueの起票まで。実Pin実装/Workbench E2E/Relation resolver/SHO・TAI実Handoff、実Host/Grant、正式Owner採用は**NOT RUN**。


## 15. 2026-11-13 横断収束：参照移動・固定・変更を分け、未保存Sourceを扱う

> [TEI Studio実装 #4515](https://github.com/hat47x/tei-platform/issues/4515) のScopeを維持しつつ、他Ownerで成熟したGUI/DB/SHO/TAIの契約を、最初の実装が誤って前倒ししないための追加条件。**この追補自体はコード変更・実テストではない**。

### 15.1 画面上の「対象」の三つの動詞

| 操作 | 必須の識別・状態 | 許されること | 許されないこと |
| --- | --- | --- | --- |
| **参照移動（Navigate）** | Workspace・表示対象・Source位置/Revision・可視範囲 | 検索結果/参照元/参照先へ移動し、現在の内容を読む | 移動しただけでPin、編集、AI送出、意味受理 |
| **固定（Pin）** | Workspace・Subject/OccurrenceのQualified Identity・選択意図・その時点の観測状態 | Focus/別ビュー/AI入力欄を移動しても、元の対象を再訪する意図を保持 | Source変更・A→B→A・同名別Workspace・権限変更で、黙って別対象へ再束縛 |
| **変更（Propose/Apply）** | 変更候補、Target実再解決、元Revision、Diff、適用先と権限、独立Expected | 変更案Preview→確認→Owner/Productの適用操作→結果再読取 | Pinを編集権限・Source正本・DB Commit・業務意味受理と同一視 |

**UI表現**：最初のWorkbenchでは「表示中の対象」と「固定した対象」を別表示する。固定対象の状態は`intended / currently-inspected / needs-requalification / unavailable / unsupported`など**観測できる事実の範囲**で表示する。`currently-inspected`は現在Grant/Disclosureを証明しない。確認できない値を`valid`や`authorized`と呼ばない。

### 15.2 未保存バッファと正本Source

- Source本文の**未保存バッファ**は、閲覧・比較・AIへの候補文脈の対象となり得る。ただし`buffer identity + edit generation + encoding/byte-range`と、保存済み`Source identity + revision`を区別する。
- **未保存Buffer上の位置**を、保存済みSourceの同offsetや同Labelと同一と断定しない。変更後の位置は、編集履歴による明示Mappingか再解決がなければ`unresolved`。
- AIが未保存内容から変更案を返しても、Sourceへの適用前に現在Buffer/Source両方のRevisionと差分を再確認する。保存時はSHO Writer/TEI Productの各Ownerを利用し、Studio側に第二の保存器を作らない。
- 同じByte内容に戻る**A→B→A**は、Digestだけでは変更の履歴を表せない。観測世代・Source revision・操作Receiptを使える範囲で保持し、欠ける場合は`unknown`として再資格化する。旧Pinの資格を自動復活させない。

### 15.3 DB Workbench・三要素分析法・SHO・TAIへの射影

| Consumer | Pinの第一段階で共有できるもの | 追加のOwner検証が必要なもの |
| --- | --- | --- |
| TEI Modelerのデータ/機能/業務モデル | Qualified Subjectの参照意図、参照数/被参照数の探索入口 | 正式Relation occurrence、多重辺、CRUD Cell、意味属性、Lossと編集逆写像（#1124/#4472/#2497） |
| DB Client / Workbench | 接続と取得結果を混同しないSelection意図 | RowsetのSnapshot/完全性、Key・Version・Read epoch、Commit/Rollback/UnknownとReceipt（#4496） |
| SHO Editor / SUI | Document/Source/原文span/Buffer revisionの引用意図 | 文書保存・再開後の再解決、著者編集を保護した再投影、SHO Writerの適用結果 |
| TAI / 外部Harness | 明示的に選んだ対象集合と用途、限定されたPreview | Product再解決、現在Disclosure、実Transport/Consumer受領、Harness OperationとEffect（TAI #629/#633） |

これらのConsumerが**同じPin Storeを共有することはP0の条件ではない**。共有するのはQualified参照の意味と、Productが責任を持つ再解決・開示境界である。DB行SelectionをModelerのSubject IDに偽装しない。SHO文書位置を意味Relation IDに偽装しない。

### 15.4 #4515の独立Expectedへ追加する反例

| ID | 入力・操作 | 期待結果 |
| --- | --- | --- |
| PIN-S1-01 | Subject AをPin→Subject BへNavigate→AのSourceへ戻る | Pin IntentはAのまま。NavigateだけではPinも変更も実行されない |
| PIN-S1-02 | AをPin→未保存Bufferで文字挿入→同offsetをクリック | 同offsetをAの原位置と誤認しない。Buffer generation/Source revisionの相違を表示 |
| PIN-S1-03 | Source A→B→Aで内容Digestが復帰 | 過去のPin資格を自動復活0。観測履歴がなければ現在性はunknown |
| PIN-S1-04 | DBのRowsetをPin→Cursor/Snapshot失効→Applyを要求 | Pin意図は保持可、DB ApplyはKey/Version/Read epoch等の独立検査なしに実行0 |
| PIN-S1-05 | Source未保存/Grant未確認の状態でAI送信Preview | 開示範囲・Buffer由来・欠落を明示し、適格性を確認できない部分の自動送出0 |
| PIN-S1-06 | 同じSource位置に二つのRelation候補、Graph表示線だけ存在 | Graphの`parent:`Edgeを意味Relation IDとして固定0。正式なOccurrence解決を要求 |
| PIN-S1-07 | 一つのSubjectの参照数が0、探索がpartial | 「未使用確定」ではなく、探索範囲と未取得部分を明示 |

### 15.5 実装順序と成果判定

1. **#4515：現行Source TypeScriptで一件のSubject Pin**。既存`inspectSubject`、`inspectEligibility`、`readInspection`、`responses.ts`を再利用し、Focus移動・同名別Workspace・旧Replyを独立Expectedで確認する。ここにDB Apply/SHO Writer/外部Harnessを混ぜない。
2. **#1824/#1124：参照移動と固定を接続**。三要素分析法の参照元/参照先ジャンプと、Pinの意図・Source位置を独立して表示。Relation identityはProduct側の一意解決が得られたときだけ正式サポート。
3. **#4496/#2497：編集候補Preview**。未保存Buffer、DB Rowset、Modeler逆写像を各Ownerで別々に検証。保存・Commit・業務受理は別Receipt。
4. **SHO/TAI：外部文書・Harnessの限定共有**。SHOの保存/再開とTEI元Sourceの再解決を別判定し、TAIのIntent/Resolve/Projection/Transport/Consumer/Harnessを混同しない。
5. **証拠**：TypeScript typecheck/test/build、実Browser・CLI、同一HEADの独立Expected、旧表示のDOM/ARIA/Preview失効をそれぞれ記録。合格しない項目は`NOT RUN / BLOCKED / FAIL`を分ける。

**現状の線引き**：2026-11-13時点、#4508はmainへマージ済み、#4498はDraft。#4515はOpenで、`reference/tei-studio-workbench/main.ts`にはまだ明示Pinがない。したがってこの節は**既存計画と実装境界の整合**であり、Pin UIやSource Applyが実装されたとの主張ではない。


## 16. 2026-11-13 Desktopセルフビルド優先下でのPin実証・参照探索・AI連携の収束

> 他チャットのDesktop/セルフビルド、Studio参照探索、Data Workbench、三要素分析法、SHO/TAIの進捗を照合した実装順序の追補。**現行コード・PRの状態を超えて実装済みとは主張しない**。既存§14〜15と#4515の受入条件を置き換えず、実製品に接続するGateを明確化する。

### 16.1 最優先の製品縦断とPinの位置

TEIの最優先は、**TEI Studio Desktopを構築し、TEI自身がその構築を再現できること**。固定対象のShelf、AI文脈共有、Relation精密選択は、この縦断を補助する重要機能であり、Desktop構築を阻む新たなHard Gateにしない。

| Gate | Productでの確認対象 | 受入に含めないもの |
| --- | --- | --- |
| **D0 実Desktop読取り** | 実Windows/Wails WebView2→StudioBridge→共通API→現在のWorkspace/Subject/Source結果。生成JSとDesktop取得JSの同一性を確認 | 画面表示だけでPin・Source意味受理・Host効果まで成功したとの主張 |
| **D1 TEI管理の再構築** | Build Intent→生成Source→Toolchain/Module→バイナリ→Stage1 TEIプロセスがStage2 Desktop Buildを起動し、入力/出力/Receiptを束縛 | 単に外部スクリプトでDesktopがビルドできたことをセルフビルドと呼ぶ |
| **P0 Qualified Subject Pin** | #4515：実Workbenchの一時Selectionと独立したPin Intentを保存せず保持し、別Node/Focusへ移動しても同じSubjectを実Readerで再解決 | PinだけでSource/Grant/Policyが最新・意味Relationが一意・AIへ送信可能との主張 |
| **P1 参照探索/編集候補** | Data/Function/Business/CRUD/Sourceを横断し、参照元・参照先・未使用候補の完全性、Source改版、変更案Diffを示す | 探索がpartialなのに参照数0を未使用確定とする、Quick Fixを即Applyする |
| **P2 SHO/AI引渡し** | 文書Buffer/Source revision、選択意図、用途別Preview、Product再解決、実受領を別票で確認 | SHO保存・AI Transport・実Effect・意味受理を一つの成功フラグにする |

[D0関連 Draft #4579](https://github.com/hat47x/tei-platform/pull/4579) は、生成WorkbenchとGoサービスをWindows Wails Desktopへ接続する実験である。CIで確認された範囲とmainに統合された範囲を分離する。**D0の合格はD1のセルフビルド合格ではない**。[Desktop統合 Draft #4498](https://github.com/hat47x/tei-platform/pull/4498)も別Gateとし、Pinの実装票 #4515 をそのPRの暗黙の前提にしない。

### 16.2 「参照数0」と「固定対象が存在する」を独立して検査する

Studioの参照・被参照一覧とPinの対象選択は同じSource Index/Qualified Subjectを利用するが、判定の意味は異なる。

- **存在**：Source Indexに同名/同位置の要素があるだけでは、Product ReaderがそのSubjectを現在解決できるとは限らない。CatalogとInspectの正負、Workspace/Source/Revisionを照合する。
- **参照数**：静的探索で0件なら「探索した範囲では0件」。Dynamic Reference、Plugin生成、Native SQL、外部Source、未取得領域を含むとき「未使用確定」は禁止する。
- **固定**：Pinは対象を再訪する**意図**であり、未解決でも消さない。現在のSubjectの同一性が確認できない場合は`needs-requalification`/ `unknown`と表示し、別Subjectへ推測で付け替えない。
- **三要素分析法**：Data/Function/Business間の横断参照は、Modeler/CRUDの根拠と区別して表示。Graph表示用Edge IDをSource-backed Relation identityへ昇格しない。
- **現在性**：既存ResponseScopeは旧Replyの描画を抑えるだけ。HostのSource/Grant変更が観測できない場合は「検査済み」ではなく「現在性未確認」。既表示DOM/ARIAやAI Previewの再資格化は#2516のConsumer境界で行う。

### 16.3 CUI/GUIの同等性を、画面形状ではなくProduct結果で検査する

最初のPinはUI内のLocal Intentでよい。ただし将来のCUI/APIの正式契約は#1158/#3419に従い、GUIが勝手に新ABIを定義しない。最低限の実証は、**同一Workspace/Qualified SubjectをGUIと通常`tei` CLIで独立にInspectし、Subject identity・Source/Revision・Failure classの差を明示する**こと。CLIにない操作をダミーで成功扱いしない。

SHOではDocument/Buffer/原文byte spanとTEI Sourceを別々に再解決する。TAIはActivity/Task、Harness Session、Transport Receipt、Consumer Resultを分離する。Pin Intentを横断共有しても第二のSemantic SSOT、Central Registry、専用Host/Policy Engineは導入しない。

### 16.4 追加の独立Expectedと実装担当

| ID | 正負対照 | Expected / 判定 |
| --- | --- | --- |
| PIN-S2-01 | 実Desktopと通常CLIで同じSubjectをInspect | Workspace/Subject/SourceのQualified Identityを対照。表示のみの合成PASSは別票 |
| PIN-S2-02 | Desktopの別Node/別ビューへ移動後に元Pinへ戻る | Pin Intentは不変。Host Reader再解決が失敗した場合は失効表示し、誤再束縛0 |
| PIN-S2-03 | Source Indexに同名・同offsetがあるが、実ReaderがSubjectを解決できない | 存在/Pin資格の成功表示0。未解決理由と探索範囲を保持 |
| PIN-S2-04 | 静的参照数0、Plugin/Dynamic/外部Source探索が未完 | 「未使用確定」0。「確認した範囲では参照0件」とCoverageを表示 |
| PIN-S2-05 | 生成WorkbenchがWindows Desktopで表示されるがStage2再構築が未実行 | D0とD1を別判定。セルフビルドPASS=0 |
| PIN-S2-06 | 未保存SHO BufferをAI Previewへ含め、保存Sourceの同offsetへ戻る | Buffer generationと保存Source revisionを別表示。曖昧な再束縛・自動Apply 0 |
| PIN-S2-07 | Harnessが候補を返し、旧Source改版・Disclosure変更が発生 | Pin保持、候補の適用/再送出はProduct再資格化まで停止。Transport成功≠意味受理 |
| PIN-S2-08 | StudioのGraphで同一端点の二Relationを描画 | 表示Edge IDだけでRelation instanceの一意Pin成功扱い0 |

**担当境界**：#4515がPIN-S2-02/03の最初の実装、#1824とDesktop OwnerがS2-01/05、#1124/#4472と参照探索OwnerがS2-04/08、SHO/TAIと#2516がS2-06/07を担当する。テストはSource TS/生成Bundle/実WebView2/通常CLI/外部Harnessを**別レイヤーで**記録し、NOT RUN・BLOCKED・FAIL・PASSを区別する。

### 16.5 優先順位

1. DesktopのD0/D1縦断を妨げず、#4515で**Subject 1件のPinを現Workbench Source TypeScriptに実装**する。新しいGlobal SessionやContext Storeを作らない。
2. 実Windows DesktopでPin/Inspectを検査する前に、現在のSource/Workspace/Readerが一意であることを確かめる。Nodeの合成試験をDesktop製品受入へ転記しない。
3. 参照探索・CRUD・DB Rowset・SHO Bufferの各Qualified Identityを、既存OwnerのProduct APIに従って追加する。Dynamic参照の未探索はCoverageへ残す。
4. 最後に限定Preview→明示Share→実Consumer受領→候補Reviewを縦断する。外部Harnessの種類は製品共通の契約上等価に扱い、個別能力差はProfileで表現する。

**現在の状態**：#4508はmain統合済み。#4579/#4498/#4479はDraft、#4515はOpen。§16は実コードの新実装や実Windows/CLI/Harness試験の実施を意味しない。各PRのCI証拠とmainの製品受入は別扱い。


## 17. 2026-11-13 生成Inspectorの実使用をPin実装とセルフビルドへ接続する

> 2026-11-13の横断確認。**Desktopの生成Workbench実Consumer、生成Inspectorの実Consumer、Subject Pin、Desktop全体セルフビルドは四つの異なる証拠**であり、成功を横滑りさせない。既存#4515、#1824、#2831、#4572/#4579のOwnerとテスト正本は維持する。

### 17.1 現時点の実装基線と証拠上限

| 実装・PR | 実際に記録された事実 | 本設計で追加すべき接続／非Claim |
| --- | --- | --- |
| [#4579](https://github.com/hat47x/tei-platform/pull/4579) | exact HEAD `5b9fb2a5300ac0b2b33476ade753596693c3407e`、Windows Wails WebView2の`generated-desktop-webview2-smoke` CI成功。実WebView2が取得した生成JSのSHA-256は`351bedd94693e80d08baca611625368c05ff59450f27c79c5a6ebc0c6e247d63`、681,264 bytes。Catalog 72件、指定Subject Inspect、Validate error 0/warning 3 | 指定Fixture/Toolchain/PR HEADのread-only Desktop Consumerのみ。Pinの状態保持、全製品のBuild Intent、main統合、全環境の再現を意味しない |
| [#4572](https://github.com/hat47x/tei-platform/pull/4572) | 契約`inspection-reader.contract.json`→既存transform→`inspection-reader.generated.ts`→esbuild→Go embed→実Desktopで`readInspection`使用。PR記載の局所Node/Go/Windows検証とStage1→Stage2生成器再実行の観測 | **生成した部品が実使用されたこと**を認める。esbuild/Wails/Go embed全体がTEI Build Graphに登録されている、Stage2のGo childがSeed非依存、Pin実装済みとはしない。PR本文ではCI未実行と明記 |
| [#4548](https://github.com/hat47x/tei-platform/pull/4548) | REA ExternalCandidatesのSource Revision/受領Digest照合を限定Go契約CIで実証。Parent Receiver #4547はmain統合済み | 外部候補のPinに近い意味論の参考。**GUI Subject Pinの再資格化やHost Grantを代行しない**。REAのCandidate/ObservationをTEI Meaning/Studio選択へ自動昇格しない |
| [#4515](https://github.com/hat47x/tei-platform/issues/4515) | 現行Workbench Subject Pinの実装票がOpen | 本Issueに実コード変更・実Consumer試験がない限りPinは未実装 |

PR本文にある`Pin`や`Source Pin`は、生成時の**入力・版の固定**を指す場合がある。GUI利用者がSubjectを明示的に固定する**Pinned Target**と名称だけで同一視しない。

### 17.2 最小の実装接続：生成Inspectorを再利用し、Pinは別の状態として持つ

```text
Workbench local:
  focus/selected Subject ── InspectSubject(workspace,kind,id) ──► Product Reader
            │                                                 │
            └─ explicit Pin ─► Pinned Intent                  │
                                  │                            │
                                  └─ explicit Reinspect ────────┘
                                            │
                                  current observation / unknown
```

- Pinを作る前に**現在のProduct Readerで一意に解決できるか**を確認する。失敗時もIntentを記録できるUIなら`unqualified`と表示し、現在解決済みと混同しない。
- `readInspection`の**生成版**と既存`inspectSubject`の契約・返却Subject・Source/Workspace identityを照合する。Generator Sourceを変えずにUIの状態管理を追加できるかを先に検証し、生成物への直接手修正は避ける。
- Pinは`selected`、Catalog表示用`parent:` Edge、Inspectorの一時Result、AI Task/Session、生成器入力Source Pinとは別。Graphの意味Relation occurrenceは#1124の正式Source-backed Resolverが揃うまで`unsupported`。
- Studio Desktop/CLIでの**同一Qualified Subject**対照は、生成Inspectorが存在するだけでは成立しない。実ConsumerのRequest/Result、Source Revision/Failure class、生成部品の使用記録を同一Fixtureで照合する。

### 17.3 セルフビルドを過大評価しない四段階の証拠票

| 段階 | 必要な観測 | 未実証なら何と記すか |
| --- | --- | --- |
| **B0 生成Source** | 契約・Generator版・Source byte→生成`readInspection` byte、独立Expected | 生成物一致の証拠なし |
| **B1 実Desktop使用** | 生成部品がesbuild→Go embed→WebView2→StudioBridge経路で実使用、実取得byte/結果を観測 | 表示だけ・埋込みだけでは実使用未証明 |
| **B2 TEI管理のBuild Graph** | Build IntentがGenerator・esbuild・nested Go module・Wails・embedを入力/成果物/版として束縛し、部分変更で正しく失効 | Stage1が部品だけ生成してもDesktop Build Graphは未完 |
| **B3 Stage1→Stage2 Desktop再構築** | Stage1 TEIプロセスがStage2 Desktop Buildを起動、同一Source/Toolchain条件と独立Receipt、Seed child依存を明示 | 単体Generatorの再入・外部手動Buildを製品全体セルフビルドと呼ばない |

B0/B1の実装証拠は#4572/#4579の**それぞれのHEAD/環境**に束縛される。二つのPRの成功を合成して、まだ存在しない同一統合HEADのB2/B3 PASSを作らない。

### 17.4 #4515へ渡す追加Expected（既存PIN-S0/S1/S2の実行差分）

| Case | 独立Expected | 先行して使う現実装 |
| --- | --- | --- |
| PIN-GEN-01 | 生成`readInspection`を呼ぶSubject AとPin Intent Aが一致。Focus/別Subject BのInspect後もPin A維持 | #4572の生成Reader、#4515のWorkbench state |
| PIN-GEN-02 | 生成ReaderのSource/Contract Revision変更、旧Async Result後着でPin Aの現在資格を自動復活0 | #4572生成物の失効、既存`responses.ts` |
| PIN-GEN-03 | 同じ表示名/同offset/同FPの別Workspace A/Bで、旧PinをBへ誤再束縛0 | #4504 Source同一性負例、#4515 |
| PIN-GEN-04 | 生成JSの実WebView2配信byteとBuild Input一致、Pin画面操作を別途観測。JS一致だけでPin PASSにしない | #4579実Desktop smoke |
| PIN-GEN-05 | TEI Stage1→Stage2がInspector生成だけ実行しても、esbuild/Wails/nested module未管理ならDesktopセルフビルドPASS=0 | #4572/#2831 |
| PIN-GEN-06 | 参照数0・Dynamic/Plugin未探索のSubjectをPinしても、未使用確定/現在Grant確認済みと表示0 | #1124参照探索、#2516 Disclosure |

**テストと実装の順序**：#4515の`reference/tei-studio-workbench/main.ts`でPinの局所状態と実Reader照合を実装→Source TypeScriptの型/単体/生成Bundleを検証→#4572の生成Readerを組み込んだ同一HEADで再検証→#4579のWindows WebView2実Consumerへ拡張→#2831のB2/B3 Build Graphへ受入証拠を接続する。途中で複数PRを未統合のまま合成した結果をPASSと報告しない。

**他チャットとの整合**：X-TEA三要素分析法のPlugin化ではSource Adapter隔離と生成Host公開操作数の契約を緩めない。SHO Editor/Document Writerは独立のBuffer/Revision/Save資格を維持する。REA候補の固定Evidence/SourceRevision照合は有用な負例だが、Product/Owner Meaning/Grantの別Authorityを飛び越えない。DSH/Codex/Claude Code等のHarnessは共通Tool/Operation契約を先行し、製品固有Skillの常駐を必須化しない。

**計画反映**：本節は#4479の実装接続規範。#4515がPin UI、#4572が生成Inspector、#4579が実Desktop Consumer、#2831がセルフビルドの実証Owner。mainのMASTER_PLANにはすでに#4515とDesktopセルフビルドの作業領域があるため、並行変更の多い正本へ本節を複製しない。実装結果・CI・独立Expectedは各OwnerのIssue/PRへ追記する。

**状態**：この節はPR本文・局所CI記録の照合と設計接続であり、#4515のPin実装、#4572/#4579のmain統合、B2/B3製品全体セルフビルド、実SHO/TAI Handoffの成功は主張しない。


## 18. 2026-11-13 実装着手判定：生成Inspectorと固定対象の結合に必要な最小差分

> §17を「次の実装担当が変更・検証できる」契約へ落とす。#4515の既存Scopeを維持し、#4572/#4579はDraftのため統合HEADが成立するまで実装完了としない。**この節のExpectedは未実行**。

### 18.1 実装可能な最小状態と禁止する結合

`reference/tei-studio-workbench/main.ts`に存在する一時的`selected`をそのままPinとして使わない。最初の実装では次の**Workbench-localの読み取り専用状態**を区別する。型名は例示であり公開ABIではない。

```ts
type PinIntent = {
  workspaceId: string;
  subject: { kind: string; id: string };
  originEpoch: number;
};
type PinObservation =
  | { state: "unqualified" }
  | { state: "inspected"; observedEpoch: number }
  | { state: "needs-requalification"; reason: string }
  | { state: "unavailable"; reason: string };
type LocalPin = { intent: PinIntent; observation: PinObservation };
```

`inspected`は「このepochのReader応答と意図したSubjectを照合した」に限定する。Source Revision/WorldFP/SourceIndexFP/Manifest/Policy/Grantの未取得値を架空の必須値にせず、**現在Readerで観測可能な識別値だけを明示**する。実Productの資格は別途評価する。

- `select(subject)`は一時表示対象を変えるだけでPinに影響しない。
- `pin(subject)`は明示操作だけでIntentを記録し、Inspectの実結果を別のObservationとして持つ。Pinを選んだ瞬間に`authorized`へ遷移させない。
- `inspectResult(requestId, workspaceEpoch, subject)`は、**requestが発行された対象**と現Pin Intentが一致した場合に限ってObservationへ反映する。後着旧Replyは破棄し、Intentを書き換えない。
- `workspaceChanged`は前WorkspaceのIntentを別Workspaceへ付け替えず、`needs-requalification`にする。UIがTaskごとにPinを分離する場合も、そのTask境界を暗黙の共通Storeへ昇格しない。
- `sourceChanged`を確実に観測できるときはObservationを失効させる。**Source/Grant変更を観測する仕組みが未接続なら現在性はunknown**であり、過去のInspect成功を継続的な権限証明としない。
- `unpin`はLocal Intentを解除するだけ。Undoで戻してもObservationは`unqualified`から再照合し、旧Source資格・旧外部送信許可を復活させない。

### 18.2 UIの最小変更とアクセシビリティ

初期UIはInspectorの近傍に「現在表示中の対象」と「固定した対象」を**別領域**で示し、Pin/Unpin/再確認を明示操作にする。既存のGraph/List/Source/Editorを大きく作り直さない。Shelfを折り畳んでもIntentは消さない。

- Buttonに現在の操作と対象を識別できるAccessible Nameを付与。キーボードでも操作可能にする。
- IME確定前、Esc、空白クリック、別タブ移動で自動Unpinしない。
- 未資格の対象では、Sourceの秘密本文・Path・件数をChip/Tooltip/ARIAに再表示しない。UI状態遷移だけで漏洩防止を保証せず、現在Disclosureの検証は#2516で独立に行う。
- Relation Edgeは現`relatedGraph()`の表示用`parent:` IDしかなければPin操作を出さない。Source-backedな正式Relation occurrenceを返せるConsumerを後段で接続する。

### 18.3 統合時に起こり得る競合を先に定義

| 同時進行の変更 | 統合上の注意 | 合格条件 |
| --- | --- | --- |
| #4515 Workbench `main.ts` | PinのStateとUI操作の差分を局所化し、`selected`と混ぜない | 旧選択処理・Inspector/Graph/Commandの既存Test維持 |
| #4572 `readInspection`生成化 | 手書き関数の実装にPinを直接結合しない。既存`presentation.ts`のexport経由の契約を再利用 | 生成Readerの同一Subject応答でPinの再確認を実施 |
| #4579 生成Workbench＋Wails | Source TS→生成JS→Go embed→実WebView2のbyteとConsumerを照合 | **同一統合HEAD**の実WebView2でPin操作を観測。別PRのPASSを合成しない |
| #4504 Source/Workspace同一性 | 同offset・同FP・A→B→A負例をPinの現在性に接続 | Source Identityの不一致を再束縛で隠さない |
| SHO/TAI/REA | 文書Occurrence/外部CandidateとWorkbench SubjectのIdentityを別々に扱う | Pinから無断Apply/Send/Meaning受理へ進まない |

### 18.4 最小検証順序と合否

**最初の1本は合成FixtureでのPin State正負テスト**とする。UI表示とHost ReaderはMock/Fixtureの役割を明示し、MockのPASSを実ReaderのPASSへ昇格しない。

1. `PIN-IMP-01`：Subject Aを選択→明示Pin→Subject BをInspect→Aへ戻る。Pin IntentはAのまま、現在SelectionはBに変化してよい。
2. `PIN-IMP-02`：AをPin→旧A応答を保留→B/Workspaceを切替→旧A応答を返す。Intentの誤再束縛0、旧応答による現在資格の復活0。
3. `PIN-IMP-03`：A→B→Aで同じDigest/同offsetに戻す。世代・Source Revisionが不明なら`needs-requalification`。以前のInspect結果を再利用しない。
4. `PIN-IMP-04`：Unpin→Undo。Intentだけ戻り、Observationは未資格。外部Send/Apply/Effect=0。
5. `PIN-IMP-05`：同一端点の複数Relationを描画する。表示用Edge IDから意味RelationのPin成功0。
6. `PIN-IMP-06`：生成Readerを含む**同一HEAD**でtypecheck/test/build→実Windows WebView2→通常CLIの対象再照合を実施。各段階のexact SHA/環境/未実行を別記。

**テストのOwner**：最初のState/UIは#4515。生成Inspectorのbyte/使用証拠は#4572、実Desktopは#4579、Desktop全体Build Graphは#2831。テスト名を第二Scenario SSOTとして重複実装せず、既存#1180/#2514の独立Expectedに差分を登録する。

### 18.5 ここで実装を止める条件

- 実ReaderのSubject Identity/Workspaceが確認できず、Labelや描画Edgeだけで一意と推測する。
- `responseScope`のlane制御だけでHost認可・Disclosure失効が保証されたと解釈する。
- `readInspection`生成化が成功したことを、PinやDesktop全体セルフビルドの合格とする。
- SHО未保存Buffer/DB Rowset/REA候補の別種Identityを、Workbench Subjectと同じID空間へ押し込む。
- Source・Model・生成器・Pluginの改版で、古いObservationを新しい実行資格として再利用する。

**引継ぎ成果**：#4515が最初に変更するファイル、保持する状態、受け入れる6つの正負Expected、#4572/#4579との同一HEAD統合条件が確定した。**コード変更・独立試験・Desktop統合はこの追補では未実施**。


## 19. 2026-11-13 実コード再確認：Inspectの単一laneとPin再確認を分離する

> #4515実装着手前のコード照合。対象は現行main `reference/tei-studio-workbench/main.ts` と `responses.ts`。この節は**既存Readerを複製しないための最小設計修正**であり、コード変更やテストPASSではない。

### 19.1 観測された具体的な競合

現行`inspect(subject)`は`responses.run("inspection", ...)`で**一時的な選択対象**の最新応答だけを表示する。内部では`selected`の変更、Inspector本文・Source Status・Graph・Noticeの更新が一つの操作にまとまっている。これをそのままPin再確認へ呼び出すと、**Pin Aを再確認しただけなのに、利用者が表示中のSubject BがAへ戻る**副作用が生じる。また、Pin Aの再確認中にBをInspectすると、同一`inspection` laneの後着抑制によってAの資格確認結果が捨てられ得る。

逆に、Pinの状態へBのInspect応答を無条件に流すと、Pin AをBへ誤再束縛する。したがって、**選択Inspectorの応答laneとPin資格確認の応答laneは論理的に独立**させる。ただしProduct Reader、Host Root、Catalog/Source資格検査は同一の既存契約を利用し、第二Reader/Policy Engineを作らない。

### 19.2 最小の実装方法（内部契約、公開ABIではない）

1. `pin(selected)`は`workspaceId + Subject(kind,id) + workspaceEpoch`をIntentへ保存するだけ。現在のInspect応答を利用する場合も、発行時のSubject/WorkspaceとCatalog/Readerの一致を確認する。PinはInspector本文を自動的に複製しない。
2. `reinspectPinned()`は`inspect()`を呼ばず、**同じ`client.inspectSubject`をread-onlyで呼ぶ専用のPin observation lane**を使う。例えば`responses.run("pinned-inspection", ...)`。結果は`readInspection`の既存契約で検査し、`selected`・Editor・Graph・Noticeを変更しない。
3. `readInspection`の戻り値がPin IntentのSubject、Workspace/Catalog観測世代と一致する場合のみ、PinのObservationを更新。生成Readerの実装が切り替わってもこのReader契約を再利用する。失敗・曖昧・改版ならIntentは保持し、Observationだけ`needs-requalification`または`unavailable`へ移す。
4. Workspace変更時は`responses.reset()`だけに頼らず、Pinの`originEpoch`と`workspaceId`も照合する。旧WorkspaceのPinを新Workspaceへ自動移植しない。再確認中のA→B→Aは同じIDでも新世代として扱う。
5. 現在表示中のSource/GraphとPinの現在性は独立表示する。Pinの再確認成功をAI送出・Source Apply・現在Grant/Disclosureの許可と解釈しない。

### 19.3 新しい独立Expected

| Case | 操作 | Expected |
| --- | --- | --- |
| PIN-LANE-01 | AをPin、Bを選択表示中にPin Aを再確認 | Bの`selected`、Editor本文、Graph選択は変化0。Pin Observationだけ更新 |
| PIN-LANE-02 | Pin Aの再確認を保留し、通常Inspect Bを完了 | Bの表示とAの資格結果が**互いのlaneを取消さない**。同じProduct Readerを使用 |
| PIN-LANE-03 | Pin A再確認中にWorkspace切替、旧A応答が後着 | 新WorkspaceのPinへ再束縛0。旧Observationの現在資格復活0 |
| PIN-LANE-04 | Pin A再確認のReader応答が曖昧/STALE_MODEL_PROJECTION | Pin IntentはAのまま。現在資格は不明/再資格化必要。Inspector Bの本文・Source Statusを上書き0 |
| PIN-LANE-05 | AをUnpin/Undo後、旧Pin再確認応答が到着 | Undo後のPin Observationは未資格。旧応答で許可済み表示0 |
| PIN-LANE-06 | `readInspection`を生成版へ切替、同一HEADでA/Bを別lane検証 | 生成部品が実使用されてもA/Bの誤表示・誤束縛0。生成byte一致のみでPin合格扱い0 |

**実装Owner**：#4515で`main.ts`の最小状態とlane分離、`responses.ts`の既存API再利用を実装。#4572で生成Reader契約を確認、#4579で実Desktop Consumerへ接続。テストは`responses.test.mjs`のlane基礎試験だけで代用せず、**実Workbenchの`selected`・Editor・Pin Observationが同時に正しい**ことを検査する。独立Expectedの正本は#1180/#2514に接続する。

**実装上の停止線**：Pinのために`inspect()`を呼んでFocus/Editorを動かす、Pin Aと選択Bの結果を同一laneで競合させる、旧ResponseをGrant/Effectの根拠にする、Source Revisionが不明な状態を`authorized`と表示する案は採用しない。

**状態**：本節は現行コードから特定した競合と解消条件の計画反映。Pin実装、生成Reader統合、実Desktop/CLI試験は**NOT RUN**。


## 20. 2026-11-13 固定操作の世代管理：同一Subjectへの再Pinと旧応答のABA拒否

> 現行mainの`responses.ts`実装を確認して§19の独立lane設計を補強。**lane分離は必要だが十分ではない**。本節は実装上の正負Expectedであり、実コード変更・テストPASSではない。

### 20.1 現行`createResponseScope`の正確な保証

`responses.run(lane, operation, publish, failure)`は`context`とlaneごとのsequenceを捕捉し、結果到着時に最新の呼出しだけをpublishする。`reset()`はcontextを進めて全laneを無効化し、`invalidate(lane)`は指定laneのsequenceを進める。これは**UI応答の順序制御**であり、Host処理の取消、ProductのSource改版検知、現在Disclosure/Grantの検査を行わない。

特に、Pin Aの再確認を開始→Unpin→同じAを再Pinという操作では、**Unpin時に`pinned-inspection`をinvalidateしなければ旧A応答が新しいAのObservationへ書き込まれる**。Subject ID/Workspace IDが同じでも、Pin Intentの世代は異なる。

### 20.2 Pin generationとReader requestの二重照合

- Workbench内で`pinGeneration`を単調増加させ、Pin、Unpin、Undo、Workspace切替、明示再資格化の際に旧資格を失効させる。数値の大小を永続的なRevisionと誤認せず、現在のWorkbench lifetime内だけで比較する。
- `reinspectPinned`開始時に`{ pinGeneration, workspaceEpoch, workspaceId, kind, subjectId }`を捕捉する。応答時には**laneの最新性に加え、これらの一致**を検査する。Readerから実際に得られるSource/Catalogの現在性は`readInspection`とProduct契約で別途照合する。
- Unpin時は`responses.invalidate("pinned-inspection")`を実行し、Pin Observationを破棄する。Undoは旧Pinの**対象指定意図**のみ復元し、新しい`pinGeneration`と`unqualified`から開始する。
- `responses.reset()`がWorkspace変更で呼ばれても、旧WorkspaceのPin Intentを新Workspaceの同名Subjectへ移植しない。旧Intentを保持するUIなら`needs-requalification`とし、旧Workspace名と新Workspace名を混ぜて表示しない。
- `pinGeneration`は旧UI応答の混入を拒否するためのローカル世代。SourceのA→B→A、同Fingerprint別Workspace、Grant改版の検証を代行しない。これらはProduct Reader/Source revisionと独立の再資格化が必要。

### 20.3 実装に使う最小の純粋状態遷移

```text
pin(A):          generation++, intent=A, observation=unqualified
reinspect(A):    capture(generation, workspaceEpoch, A), reader request
select(B):       selected=B, pin intent/observation unchanged
unpin():         invalidate pinned lane, generation++, intent=null
undo():          generation++, intent=A, observation=unqualified
workspace(B):    invalidate pinned lane, generation++, old intent needs-requalification
reply(A):        publish only if lane current AND captured generation/epoch/identity current
```

`selected`がBの間に`reinspect(A)`してもEditor/Graph/Noticeを変更しない。応答失敗時もPin Observationだけを更新する。Pinの資格表示を`authorized`や`fresh`と呼ばず、**このReader観測時点で照合済み**という意味に限定する。

### 20.4 追加の負例と検収順

| ID | 操作 | Expected |
| --- | --- | --- |
| PIN-ABA-01 | AをPin→再確認開始→Unpin→Aを再Pin→旧応答到着 | 新PinのObservationは未資格。旧応答による更新0 |
| PIN-ABA-02 | AをPin→再確認開始→Unpin→Undo→旧応答到着 | UndoはIntentだけ復元。旧Observation復活0 |
| PIN-ABA-03 | AをPin→再確認開始→Workspace A→B→A→旧応答到着 | 同Workspace ID復帰でも旧epochの応答採用0 |
| PIN-ABA-04 | AをPin、Bを通常Inspect→A再確認が失敗 | BのEditor/Graph/Source Statusの変更0。Pinのみ失効 |
| PIN-ABA-05 | AをPin→A再確認→別のA再確認を開始→旧A成功が後着 | 同lane内で最新Requestだけ採用 |
| PIN-ABA-06 | AをPin→Source A→B→A、ReaderのSource revision不明 | ローカルgeneration一致だけでSource資格復活0。現在性はunknown |

**検収順**：まず`responses.test.mjs`で`invalidate("pinned-inspection")`と独立laneの既存保証を確認する。その後、Workbench Pin stateの純粋な状態遷移テストでgeneration/identity/epochの二重照合を検証し、最後に実`main.ts`のEditor/Graph/Pin表示が干渉しないことを検証する。Node合成PASSを実Windows Desktop/Host GrantのPASSに転記しない。

**担当**：#4515がローカルPin stateと実Workbench統合を所有。#4572の生成Reader、#4579の実WebView2、#2831のセルフビルドは各Ownerの証拠と同一HEADで接続する。新たな共通Session Storeや公開ABIは作らない。

**状態**：この節は`responses.ts`の現在の保証と限界に基づく計画更新。実装・typecheck・Node/Browser/Windows試験は**NOT RUN**。


## 21. 2026-11-13 実装着手の記録：Pin世代管理の純粋状態遷移

[Draft #4597](https://github.com/hat47x/tei-platform/pull/4597)（base main、HEAD `c9302d55c9d9759f95aeb61f5686fea114fd04ba`）で、#4515の第一段階を**コード化**した。

- `reference/tei-studio-workbench/pin-state.ts`：`initialPinState`、`pinSubject`、`unpinSubject`、`restorePinIntent`、`invalidatePin`、`pinRequest`、`acceptPinObservation`。同じSubjectを再Pinしてもgenerationが異なるため旧Reader応答を受理しない。
- `pin-state.test.mjs`：PIN-ABA-01〜06とUnpin後の後着拒否、計7件の正負試験を追加。
- `package.json`：Workbenchの標準`npm test`へ新テストを登録。

**重要な制約**：これはUIと切り離した純粋状態遷移であり、現`main.ts`のPin操作・`responses.invalidate("pinned-inspection")`・生成Reader `readInspection`との実結合・Windows DesktopのPin E2Eは未実装。PinObservationの`inspected`はProduct Reader検査後にのみ呼出側が渡せる観測であり、Source/Grant/Disclosureの現在性を保証しない。従って**「Pin UI実装済み」とは扱わない**。

**実行証拠**：GitHubに3ファイル変更を作成し、Draft PRを起票した。Node 24の`npm run typecheck`、`npm test`、`npm run build`、生成Bundle/実WebView2/CLIは**NOT RUN**。CIと実環境の結果を同一HEADで収集するまでマージを保留する。

**次の実装ゲート**：#4597のCI/型・テスト検証→#4515で`main.ts`へ明示Pin/Unpin/再確認のUIを結合→#4572生成Readerと#4579実Desktopの同一HEADでE2E→#2831 Build Graph。これ以上の独立Store/Policy Engineを増設しない。


## 22. 2026-11-13 #4597のexact HEAD CI観測と次の修正条件

[Pin状態遷移 Draft #4597](https://github.com/hat47x/tei-platform/pull/4597) HEAD `c9302d55c9d9759f95aeb61f5686fea114fd04ba`に対する [Workbench embedded bundle Actions #43385273632](https://github.com/hat47x/tei-platform/actions/runs/43385273632) を実ログで確認した。

| 観測 | 実際の結果 | 採用できるClaim |
| --- | --- | --- |
| `npm test` | 27件PASS/0 FAIL（PIN-ABA-01〜06とUnpin負例を含む） | 現HEADのWorkbench単体テストでPin純粋状態遷移が成立 |
| `npm run typecheck` | Step成功 | 現HEADのTypeScript型検査成功 |
| `npm run build` | Step成功 | Workbench SourceからBundle生成は成功 |
| `cmp reference/tei-studio-workbench/dist/main.js cmd/tei-studio/assets/workbench/main.js` | **FAIL**。byte 649340、line 18643で差異 | Desktop埋込みBundleのSource一致は**不成立**。生成成功をembed一致と呼ばない |
| Cache Full vs Incremental | 同HEAD Workflow success | PinのDesktop実Consumerや埋込みBundle一致の代用不可 |
| Windows Desktop Pin UI | **NOT RUN**（そもそも#4597ではUI未接続） | Pin UI/実WebView2の完成Claim不可 |

**修正ゲート**：生成`dist/main.js`と埋込み`assets/workbench/main.js`の差異について、同じNode/esbuild/依存/Sourceのもとで差分を確認し、**生成物が現在Sourceから再現されること**を証明してから同期する。原因を「Pin状態モジュール追加」や「既存mainの不整合」と断定しない。生成JSを手編集せず、#4572/#4579と競合する生成物・Build RecipeのOwnerと調整する。

**進行可能な範囲**：#4597の状態遷移コードは単体テスト・型検査の成功が確認できたため、レビューと#4515の`main.ts`局所UI設計へ進められる。ただしembed一致がFAILのままマージしない。CI再実行・差分修正後のexact HEADを別途検証する。

**注意**：上記は実ログで確認した範囲。ほかのWorkflowの未完了/結果を推測してPASSへ加算しない。


## 23. 2026-11-13 Pin初期実装のCI結果と生成Bundle失効の分離

[Draft #4597](https://github.com/hat47x/tei-platform/pull/4597)の旧HEAD `c9302d55c9d9759f95aeb61f5686fea114fd04ba` に対する[Workbench embedded bundle CI #43385273632](https://github.com/hat47x/tei-platform/actions/runs/43385273632)を確認した。

- **PASS（同HEAD）**：Workbench `npm test` 27件（Pinの新規7件を含む）、`npm run typecheck`、`npm run build`。
- **FAIL（同HEAD）**：生成`reference/tei-studio-workbench/dist/main.js`と`cmd/tei-studio/assets/workbench/main.js`のbyte比較。CIログはbyte 649340、line 18643で不一致を記録。
- **原因は未確定**：このPRの`main.ts`および埋込み`main.js`はmainと同一Git blob SHA。Pin状態遷移はentrypointに未接続。したがって「PinコードがBundle差異を起こした」とは断定せず、同時に「既存の失敗だから無視してよい」ともしない。

新HEAD `247a6ba6579d8feab618419c100a1e16bb1fe65a` で`.github/workflows/studio-pin-state-contract.yml`を追加。Pinの7件とWorkbench型チェックを独立のCIゲートで実行する。**新HEADの専用CIはcompleted/success**（[Run #43394212153](https://github.com/hat47x/tei-platform/actions/runs/43394212153)、`pin-intent-contract`のPin 7件・型チェックStepすべてsuccess）。ただしembedded bundle CIは別GateでありP2は未解決。従来のembedded bundle CIは製品統合ゲートとして維持し、通過するまでDesktopへマージしない。

### 23.1 失敗を隠さず前進する検収の分離

| Gate | 証拠 | 現在の状態 |
| --- | --- | --- |
| P0 Pin純粋状態遷移 | PIN-ABA-01〜06＋Unpin後着、Node 24 | 旧HEADで7件PASS、新HEAD専用CI success |
| P1 Workbench Source型・既存試験 | 27件＋TypeScript | 旧HEAD PASS、新HEAD専用型チェックsuccess |
| P2 Source→生成JS→埋込みJS byte一致 | 同一HEADでの`npm run build`と`cmp` | **旧HEAD FAIL、原因調査中** |
| P3 実Workbench Pin UI | `main.ts`配線、別lane、Focus/Graph非干渉 | 未実装 |
| P4 Windows実WebView2＋生成Reader | #4572/#4579と同一統合HEADの実Consumer | 未実施 |
| P5 TEI Desktopセルフビルド | #2831 Build Intent/Stage2/Receipt | 未実施 |

P0/P1がPASSしてもP2〜P5をPASSにしない。P2の解消は**現在の正しいSource・生成器・Toolchainと実埋込みbyteの差分を検証してから**行う。根拠なく生成Bundleを手編集・上書きしない。新規CIの追加は既存CIを置換しない。

**次のOwner作業**：#4597は専用CIのexact HEAD結果を確認し、P0/P1の検証を確定。Workbench生成物のOwnerはP2の入力Source・生成器版・bundle差異を調査。#4515はP3の局所UI配線、#4572/#4579はP4、#2831はP5。Draft状態を維持し、結果を個別Receiptとして記録する。


## 24. 2026-11-13 同一HEADの検証票と統合停止条件

#4597の**現HEAD** `247a6ba6579d8feab618419c100a1e16bb1fe65a` に紐づくGitHub Actions結果を再確認した。旧HEADの27件PASSと混同せず、現在の各Workflowの結果を分けて記録する。

| Workflow | Run | 結果 | 解釈 |
| --- | --- | --- | --- |
| Studio Pin intent state contract | [43394212153](https://github.com/hat47x/tei-platform/actions/runs/43394212153) | **SUCCESS** | Pin局所状態の専用契約ゲート。GUI Pin・Source資格の保証ではない |
| TEI Cache Full vs Incremental | 43394212199 | SUCCESS | キャッシュ系ゲート。PinのGUI資格へ転用しない |
| TEI Studio #8 independent consumer | 43394212142 | SUCCESS | 当該Workflowの独立Consumer試験。Pin UI未配線 |
| TEI Studio PoC A Wails Host | 43394212173 | SUCCESS | 当該Host試験。Pin操作の実Desktop証拠ではない |
| TEI Studio Workbench embedded bundle | [43394213664](https://github.com/hat47x/tei-platform/actions/runs/43394213664) | **FAILURE** | Source生成物とDesktop埋込みの整合ゲートは未通過 |
| TEI PR・main 健全性確認 | 43394212159 | **FAILURE** | 別の失敗。原因をbundleと同一と推測しない |
| 依存の既知脆弱性確認 | 43394212157 | SUCCESS | 依存の当該Workflowに限定 |

**採否**：専用Pin契約は現HEADでPASS。**PR全体は受入未完了**でありDraftを維持する。embedded bundleの差異は実生成物の入力/byteを比較してOwnerが解消し、健全性確認FAILはログを別途調査する。生成JSを手で修正しない。

**実装の次の一手**：#4515でPin UI配線を行う際、`pin-state.ts`の純粋関数を利用し、`responses.run("pinned-inspection")`のReader結果を`readInspection`で検査した後に限りObservationへ反映する。通常`inspection` laneのEditor/Graphを触らない。これを同一HEADで検証してから#4572/#4579との統合へ進む。


## 25. 2026-11-14 PR #4597の健全性失敗を原因別に修正

#4597旧HEAD `247a6ba6579d8feab618419c100a1e16bb1fe65a` の[repository-health #43394212159](https://github.com/hat47x/tei-platform/actions/runs/43394212159) のJob `repository-health`実ログを調査。失敗したのはPin状態遷移テストではなく、追加Workflow `studio-pin-state-contract.yml` がリポジトリの静的Workflow契約に反していたためだった。

具体的な失敗は、`test_every_automatic_workflow_declares_concurrency`（top-level concurrency欠如）、`test_every_checkout_drops_persisted_credentials`（Checkoutの`persist-credentials: false`欠如）、`test_every_workflow_pins_actions_and_bounds_jobs`（ActionsのSHA固定とtimeout等の規約違反）。これらを**同じPR #4597**で修正した。

**修正HEAD `f4f849f63544f514fe05306b934864e3507bed1a`**：`permissions: contents: read`、PR限定cancel付きtop-level concurrency、`ubuntu-24.04`、`timeout-minutes: 10`、Checkout/Setup Nodeの既存規約準拠SHA、`persist-credentials: false`、PR head SHA checkoutを設定。追加の共通CI基盤は作らず、既存のWorkflow設計と同じ規約に揃えた。

**検収の区別**：旧HEADのPin専用CI successは旧HEADの証拠として保持。新HEADのWorkflow修正はコミット済みだが、**新HEADのrepository-health/Pin専用CI再実行は未確認**。再実行の成功を推測してPASSへ昇格させない。独立のembedded bundle不一致は引き続きFAILであり、今回のWorkflow修正では解消していない。

**次のゲート**：新HEADの`repository-health`（静的契約）と`Studio Pin intent state contract`（7件・型検査）を確認し、embedded bundleのSource/生成器/埋込みbyte差分を別Issueで解消する。#4515の実Pin UI・#4572生成Reader・#4579実Desktopは同一HEADで統合試験する。


## 26. 2026-11-14 Pin契約の現HEAD CIとBundle再現性の修復手順

#4597 HEAD `f4f849f63544f514fe05306b934864e3507bed1a`のGitHub Actionsを照合した。

| Gate | Workflow Run | 観測 |
| --- | --- | --- |
| Pin局所契約 | [43572042693](https://github.com/hat47x/tei-platform/actions/runs/43572042693) | **completed/success** |
| Workbench embedded bundle | [43572042636](https://github.com/hat47x/tei-platform/actions/runs/43572042636) | **completed/failure**。失敗Stepは`Compare build output with Wails embedded files` |
| Cache Full vs Incremental | 43572042656 | completed/success。Pin/Bundleの代用不可 |
| Studio independent consumer | 43572042642 | completed/success。Pin UIの代用不可 |
| Wails PoC A Host | 43572042696 | completed/success。Pin実Desktopの代用不可 |
| Repository health | 43572042669 | **確認時in_progress**。先行する静的規約修正のPASSは未確定 |

### 26.1 Source→Bundle→Desktopの整合を実際のRecipeで扱う

現行`reference/tei-studio-workbench/build.mjs`は`main.ts`をesbuildで`dist/main.js`へbundleし、同時に`dependencies.json`、`THIRD_PARTY_NOTICES.txt`、`index.html`、`style.css`を生成する。CI `tei-studio-workbench-embed-ci.yml`はNode `24.21.0`、`npm ci`、`npm test`、`npm run typecheck`、`npm run build`後、**4ファイル**を`cmd/tei-studio/assets/workbench/`とbyte比較し、埋込み側に`dependencies.json`が無いことも確認する。

したがって`main.js`だけを手修正して一致させるのは不可。差分の原因を、Source/lockfile/esbuild/ビルドRecipe/出力ファイルのどこにあるか切り分ける。診断時には**同一HEAD**の4ファイルそれぞれのSHA256、byte長、最初の差分位置、`dependencies.json`に記録された入力、Node/npm/esbuild版、`git status`を収集する。差分にSourceの機密情報がある場合、ログへ本文を出さずhashと位置に限定する。

修正は**Source・lockfile・Recipeの正本**を確認した後に、生成手順で4ファイルを一括同期し、同一HEADで再ビルドして`cmp`を再実行する。生成物を直接編集した場合、TEIのWorld/Artifact Revisionの受理証拠にはならない。#4572の生成Reader、#4579の実Desktop Consumerが別Draft HEADにあるため、単独PRの成功を統合HEADの成功と混同しない。

### 26.2 次の実装ゲート

1. **P0/P1**：#4597の現HEAD専用Pin契約PASSは採用可能。Repository healthは完了を待って別票で確認。
2. **P2**：Workbench Source→dist→Desktop embeddedの4ファイルbyte一致。失敗のままDesktop統合受入へ進めない。
3. **P3**：#4515の`main.ts`へPin/Unpin/再確認を接続。既存`responses.run("inspection")`を再確認へ流用せず、独立laneと`pinGeneration`で旧応答を排除する。
4. **P4/P5**：#4572生成Reader＋#4579 Windows WebView2を同一HEADで検証し、#2831のBuild Graph/Stage2セルフビルドは別途実証する。

**今回の変更は計画への観測反映であり、Bundle差異を解消したというClaimではない。**


## 27. 2026-11-14 修正後HEADのCI確定とPin UI接続の最小契約

#4597 HEAD `f4f849f63544f514fe05306b934864e3507bed1a`の[repository-health #43572042669](https://github.com/hat47x/tei-platform/actions/runs/43572042669)は、前節の確認時には実行中だったが、今回の再照合で**completed/success**を確認した。同HEADの[Pin専用CI #43572042693](https://github.com/hat47x/tei-platform/actions/runs/43572042693)もcompleted/success。一方、[embedded bundle #43572042636](https://github.com/hat47x/tei-platform/actions/runs/43572042636)は引き続きcompleted/failure。したがって**静的Workflow規約違反は解消されたが、生成Bundle整合は未解決**という2つの結論を分離する。

### 27.1 現行WorkbenchのUI責務と最小差分

現行`main.ts`には`selected`、`workspaceEpoch`、`catalogSnapshot`、`responses.run("inspection")`があり、`inspect(subject)`は選択・Editor・Graph・Source Status・Noticeをまとめて変更する。Pinの再確認をこの関数で代用すると、Aを固定しBを閲覧中にAを再確認しただけでBの画面がAへ移動する。

最小の実装境界は次のとおり。

- **Pinボタン**：現在の`selected`を対象に`inspectEligibility(catalogSnapshot, selected)`で一意性・利用可能性を確認し、`pinSubject`でIntentを記録。選択の変更は行わない。
- **固定対象欄**：`index.html`のInspector側に`pin-subject`、`unpin-subject`、`reinspect-pin`、`pin-status`を設ける。固定対象のKind/ID/WorkspaceとObservationを読み取り専用で表示し、現在の選択対象と区別する。未固定時の再確認ボタンは無効。
- **固定対象の再確認**：`responses.run("pinned-inspection", () => client.inspectSubject(...))`を使用。応答は`readInspection(unwrap(reply), capturedSubject, capturedCatalog)`で検査したうえで、`acceptPinObservation`に渡す。通常の`inspect`、Editor、Graph、Source Status、Noticeは触らない。
- **Unpin/Undo/Workspace切替**：`responses.invalidate("pinned-inspection")`を明示的に呼び、generationを進める。Undoは対象Intentのみ復元し、Observationは`unqualified`に戻す。Workspace切替は既存`responses.reset()`と整合させ、旧WorkspaceのIntentを現在のWorkspaceの同名Subjectに自動移植しない。
- **Catalog再読込**：Pin対象の資格は自動継承せず`needs-requalification`へ移す。Source/Model Revisionの一致が検査できないとき、UIは「検査済み」を表示しない。
- **例外**：固定対象のReader失敗はPin欄にのみ反映し、選択中BのEditor/Graphを消さない。HostのReaderを二重実装せず、既存の`client.inspectSubject`だけを利用する。

### 27.2 受入負例と順序

| ID | 操作 | Expected |
| --- | --- | --- |
| PIN-UI-01 | AをPin、BをInspect、Aを再確認 | Editor/Graph/Source StatusはBのまま、Pin欄のみAのObservationを更新 |
| PIN-UI-02 | AをPin、再確認開始、Unpin、旧応答到着 | Pin欄は未固定。旧応答反映0 |
| PIN-UI-03 | AをPin、再確認開始、Workspace Bへ切替 | 旧Workspaceの結果採用0、Bの同名Subjectへ自動固定0 |
| PIN-UI-04 | Aの再確認が`STALE_MODEL_PROJECTION` | Pin欄だけ`needs-requalification`、通常Editorの破棄0 |
| PIN-UI-05 | AのCatalogがambiguous/unavailable | Pin開始拒否。Host Inspect呼出0 |
| PIN-UI-06 | Pin/Unpin/Undoを連続し旧応答到着 | 現Intent generation以外の結果反映0 |

**進行条件**：#4597のP0/静的CI PASSは採用できる。ただしP2のBundle不一致を保持したまま「実Desktop完成」とは言わない。UI実装は#4515の局所Ownerが担当し、生成JSは正規Build Recipeで再生成する。#4572生成Reader・#4579実WebView2・#2831セルフビルドは統合HEADで別途検収する。

**状態**：本節は現行`main.ts`/`index.html`の実コードと現HEAD CIに基づく**具体的実装仕様**。PIN-UI-01〜06はExpectedを定義した段階で、実装・試験は未実施。


## 28. 2026-11-14 Pin独立laneの実テストを追加

[Draft #4597](https://github.com/hat47x/tei-platform/pull/4597)に`responses.pin.test.mjs`を追加し、従来のPin状態遷移7件とは別に**4件の非同期応答回帰**をコード化した。現HEAD `a9c91e56eecdb6b588064188490fc3de950c1744`。

| テスト | 操作と独立Expected |
| --- | --- |
| PIN-LANE-01 | AをPinしたままBの通常Inspectが成功し、後着AのPin結果も採用。Bの選択結果を維持 |
| PIN-LANE-02 | Pin Reader実行中にUnpinし、`invalidate("pinned-inspection")`。後着A成功はpublishされずIntentはnull |
| PIN-LANE-03 | Pin Reader実行中にWorkspaceを切替え`responses.reset()`。旧Workspaceの後着失敗を表示しない |
| PIN-LANE-04 | 同じPin laneで旧Aと新Aを並行し、新A成功後に旧Aが後着。新Aのみ採用、通常Inspect Bは維持 |

標準`npm test`に追加し、専用`studio-pin-state-contract.yml`もPin状態遷移7件とlane回帰4件の両方を実行するよう変更した。Workflowのpathsにも新ファイルを追加した。

**検証境界**：このテストは`createResponseScope`と`pin-state`を合成した非同期順序の検証であり、`main.ts`の実UI配線、実ReaderのSource/Revision、Windows WebView2の操作試験ではない。追加後HEADのCIは**結果未確認**。旧HEADの専用CI/Repository health成功を新HEADへ転用しない。embedded bundle byte比較のFAILも未解消。

**次段階**：新HEADのCI確認後、#4515で`main.ts`の実Pin UIに同じlane/Intent検査を配線し、UIからBを表示したままAの再確認を実行するE2Eを追加する。
