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
| P2 SHO document+SUI | SHO #4/#19/#31、SUI #3391 | 文書Occurrence/byte anchorを軽量Editor内でPinし、SHO単独保存、SUIに必要な選択だけをHandoff | SH0 Reader/Writer・Storageの実格付けを確認。TEI Sourceへの自動書戻し0 |
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
