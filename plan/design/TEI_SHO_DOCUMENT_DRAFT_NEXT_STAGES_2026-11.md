# TEI→SHO文書Draft：後続段階の実行計画（2026-11-11）

## この文書の役割と状態

- **状態**：`実装前設計（段階的拡張の計画）`。2026-11-11時点で実装済みなのは段階1（`element-explanation`）のみ。
- **正本との関係**：契約自体は[TEI_SHO_DOCUMENT_DRAFT_INTEGRATION_2026-11.md](TEI_SHO_DOCUMENT_DRAFT_INTEGRATION_2026-11.md)が正本で、この文書は**その契約をどう伸ばすか**の順序・受入条件・依存だけを置く。契約の重複は書かない。
- **対象**：当初目標のうち未達の部分——三要素分析図・CRUD表・仕様書・固定納品版、TEI Studio実組込み、相互のCI証跡、逆方向の経路、`*.tei.md` の往復。
- **この文書が主張しないこと**：各段階の受入条件を満たした事実。ここに書くのは「何を・どの順で・何をもって完了とするか」のみ。

## 0. 現在地（2026-11-11）

| 実装済み | 根拠 |
| --- | --- |
| `internal/documentdraft`（Build・Occurrence追跡・Reconcile） | TEI PR #4317、Go試験PASS |
| `tei model document-draft`・`model document-reconcile` | 同上、契約試験PASS |
| `documentKind: element-explanation`（一要素＋一関係） | 同上 |
| `documentKind: crud-table`（一ApplicationのCRUD行列） | 段階2、Go試験＋CLI試験PASS |
| `documentKind: specification`（複数要素の章立て） | 段階4、Go試験＋CLI試験PASS |
| `-frozen`（固定納品版のpin） | 段階5、Go試験＋CLI試験PASS |
| SHO側の消費・編集・保存・再開 | SHO PR #36、17試験PASS・merge後スイート127 PASS/0 FAIL |
| SHO側のローカル相互ランナー・入出力契約試験・部品境界文書 | SHO PR #36、相互ランナー実PASS |
| **相互CI（同一Attempt）** | SHO repository の [sho-tei-draft-roundtrip.yml](https://github.com/hat47x/sho-composer/blob/feat/tei-document-draft-core-20261111/.github/workflows/sho-tei-draft-roundtrip.yml) が **success**（run 42995269509・artifact に receipt） |
| 相互実証（合成Workspace・同一端末） | PR本文の記録 |

| 未実装 | この文書の段階 |
| --- | --- |
| 三要素分析図（Data/Function/Business） | 段階3（`internal/threeelement` は [PR #4283](https://github.com/hat47x/tei-platform/pull/4283) OPEN） |
| TEI Studio実組込み | 横断C-2 |
| 文書→モデルの逆方向経路 | 横断C-3（経路は作らない） |
| `*.tei.md` のTEI側取込み | 横断C-5（Parser未束綝でブロック） |
| 再投影のマージ方針 | 横断C-4（Owner判断ゲート） |

## 1. 段階2：`crud-table`（2026-11-11 実装済み）

- **目的**：一ApplicationがマップしたDataに対して導出できる標準CRUD操作をDraft化する。
- **依存**：段階1。**`internal/threeelement` は main に存在しない**（[PR #4283](https://github.com/hat47x/tei-platform/pull/4283) が OPEN のまま）。三要素分析モデルへの依存は持たず、main に既にある `internal/operations` の operation surface（`tei.operation-surface/standard-crud/v1`）を利用する。**CRUDの意味的解釈は[THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md](../issues/THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md)のOwner未決事項のまま**（本段階は投影のみ）。
- **触るファイル**：`internal/documentdraft/crud.go`（新規）・`draft.go`（入力排他と`scope`）・`reconcile.go`（CRUD経路）・`cmd/tei/cli_document_draft.go`（`-document-kind`・`-application`）・CLI JSON契約文書。
- **実装**：表の各行は導出された標準操作。1行＝1 CRUD操作で、`rowSubjects`に対象データとApplicationのIdentityを持つ。未導出・矛盾は行列に混ぜず別Occurrenceにする。
- **独立Expected（実装済み）**：
  - S2-a 導出規則（`scope.crudRule`）と各行の`basis`を記録する。
  - S2-b 未宣言のCRUDは作らない（`updateIntent`が無ければretire gapは出さない）。
  - S2-c 未知のApplicationは`APPLICATION_NOT_FOUND`。**空の表を「このApplicationは何もしない」と読ませない**。
  - S2-d モデル改変（表名変更・書き込み権限撤回・update intent追加）で該当Occurrenceが`content-changed`・`occurrence-removed`・`occurrence-added`。
  - S2-e 未実装の`documentKind`は`UNSUPPORTED_DOCUMENT_KIND`。`element-explanation`は`-application`を拒否し、`crud-table`は要素指定と`-relation-id`を拒否する。
- **Done means**：S2-a〜eがGo試験とCLI試験でPASS。
- **非目標**：CRUD網羅性の保証・図面レイアウト・Owner意味受理・実行権限。

## 2. 段階3：`three-element-diagram`

- **目的**：Data/Function/Businessの**構造記述**（node/edge identity・方向・多重度・出典）をDraft化する。レイアウト・色・形状は持たない（[THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)と同じ扱い）。
- **依存**：段階2。[#1124](https://github.com/hat47x/tei-platform/issues/1124)・[#3263](https://github.com/hat47x/tei-platform/issues/3263)の受入条件。**`internal/threeelement` は main に無い**（[PR #4283](https://github.com/hat47x/tei-platform/pull/4283) OPEN）ため、三要素分析モデルを入力にする場合は同PRの統合後にする。統合前は main の `operations.ModelMap`（型付きナビゲーション投影）で代替できるかを別途検討する。
- **最小実装**：`diagram`種別のOccurrenceに`nodes[]`（identity・kind・出典）と`edges[]`（from/to identity・relation・多重度・resolution）を入れる。SHO側は`diagram`を**構造記述として保存するだけ**（描画・編集は別資格）。
- **独立Expected**：
  - S3-a 平行edge（同じfrom/to・別relation identity）を縮約しない。
  - S3-b 未解決targetは`unresolved`のまま（推測で接続しない）。
  - S3-c 同名の別要素（Data "Order" と Function "Order"）を`kind:id`で区別する。
  - S3-d モデル改変で影響するedgeのみ`content-changed`（`AnalyzeImpact`の結果と一致）。
- **Done means**：S3-a〜dがGo試験でPASS。**図面編集は作らない**。
- **非目標**：Canvas・レイアウト・Renderer・可逆図上編集。

## 3. 段階4：`specification`（2026-11-11 実装済み）

- **目的**：複数要素・複数関係を一つの文書に束ね、章構成を持つ仕様書Draftを作る。
- **依存**：段階1。`subjects[]`・`relations[]`・Occurrenceは既に複数件を持てるので、追加は**束ね規則と章立て**のみ。
- **形**：`-section <kind>:<id>[@<relation-id>]`（repeatable・章順）。章＝1要素、節＝1関係、各章に引用。
- **独立Expected（実装済み）**：
  - S4-a 同一入力から決定的な同一Draft（順序・digest安定）。
  - S4-b 未宣言の要素は**章を作らず**、`scope.skippedSubjects` に理由を残す（全体を失敗にしない）。未宣言の関係は章を残し関係節のみスキップ。
  - S4-c 入力形状の混在（application・要素指定・トップレベルrelation）は拒否。
  - S4-d 一章の改版はその章のみ`content-changed`、他章は`generation-conditions-changed`（内容は同一）。要素削除は`occurrence-removed`。
- **Done means**：S4-a〜dがGo試験とCLI試験でPASS。
- **非目標**：目次の自動生成・表現の潤飾・Owner意味の統合判断。

## 4. 段階5：`frozen-delivery`（2026-11-11 実装済み）

- **目的**：固定納品版。Draftが**特定Revisionにpin**され、以後のモデル改版は「失効」として報告され、自動更新しない。
- **依存**：段階1。**pinの意味（再確認の要否・再納品の条件）はOwner判断**。
- **形**：`-frozen` は**種類ではなくpin**。Draftは `frozen: true` と `frozenRevision`（読出した時点のmodelRevision）を持つ。`frozen-delivery` を種類として指定した場合は拒否する。`frozen` でpin無しは `INVALID_DRAFT`。
- **独立Expected（実装済み）**：
  - S5-a pin後・モデル不改変 → `current`・再確認0件。
  - S5-b pin後・モデル改変 → `stale`＋`frozenRevision` diff＋`frozen-delivery-is-not-refreshed-automatically`。**Draftも文書も書換わない**。
  - S5-c `frozen-delivery` を種類として指定 → `UNSUPPORTED_DOCUMENT_KIND`。
  - S5-d `frozen: true` で `frozenRevision` 無し → `INVALID_DRAFT`（手編集で期限を不能にできない）。
  - S5-e CRUD表にも同じpinが乗る（種類とpinは直交）。
- **Done means**：S5-a〜eがGo試験とCLI試験でPASS。
- **非目標**：納品物の署名・配布・受領証。

## 5. 横断の課題（段階と独立）

### C-1 相互CI証跡（同一Attempt）

- **目的**：「TEI実CLI → SHO実CLI → TEI実reconcile」を**CI上で同一Attempt**として実行し、ローカル実証だけにしない。
- **配置は SHO repository 側**（[sho-tei-draft-roundtrip.yml](https://github.com/hat47x/sho-composer/blob/feat/tei-document-draft-core-20261111/.github/workflows/sho-tei-draft-roundtrip.yml)）。**逆方向は認可境界で成立しない**: SHO repository は private で、tei-platform の `github.token` は読めず API tarball が 404 になる。tei-platform は public なので、SHO 側から clone する方針に credential が要らない。
- **pin は TEI 側のSHA**（`TEI_PIN`）。latest へ暗黙に追随しない。動かす場合は理由をPRへ書き、新しいpairのreceiptを読んでから更新する。
- **独立Expected**：C1-a TEI pin SHAを変えるまで同一receipt。C1-b TEI側の破壊的変更（CLI名・flag・contract変更）でFAIL。C1-c 合成データのみ（顧客データ・非公開资料なし）。C1-d receiptの全claimがfalse（緑は受理ではない）。
- **Done means**：job が success で、artifact のreceiptが読める。**2026-11-11 時点で達成**（run 42995269509）。
- **非目標**：製品E2E・Owner受理・外部配布。

### C-2 TEI Studio実組込み（部品再利用の証明）

- **目的**：TEI StudioがSHOの文書編集コンポーネントを**同じ契約の部品**として利用できることを実測する。製品別の二重実装を作らない。
- **依存**：A/B測定（子プロセス起動／Wails資産への静的組込み／外部プロセス）。**測定なしに形態を決めない**。
- **最小実装**：`cmd/tei-studio` に**限定read-onlyの試作面**を1つ追加し、`bin/sho-doc.cjs` を子プロセスとして起動してopen/resumeだけを行う（保存は明示操作）。起動・標準入出力・失敗分類・Port認可を観測する。
- **独立Expected**：C2-a Studio起動でdraft open→resumeが通り、生成領域が`intact`。C2-b SHO不在・版不一致・起動失敗を区別し、Studioが沈まない。C2-c Studioは意味を解釈しない（表示はDraftの文字列をそのまま）。C2-d 保存はStudio側から行わない（SHOの明示操作のみ）。
- **Done means**：C2-a〜dが記録され、組込み形態の選択根拠（起動時間・失敗分類・配布）が文書化される。
- **非目標**：Studio製品化・共通Host・SHO全Pluginの自動ロード。

### C-3 逆方向（文書→モデル）の経路は作らない

- **方針**：SHO文書の文章修正はTEIモデル変更の根拠にならない。モデル変更は既存の変更計画・検証・Owner受理（[#267](https://github.com/hat47x/tei-platform/issues/267)）のみ。
- **最小実装（担保）**：`internal/documentdraft` に**書込みAPIが存在しないこと**を、試験と静的検査で担保する。（a）`workspace` への書込み関数を呼ばない、（b）`Reconcile`がWorkspaceのbyte不変であることを試験で固定（実装済み：`TestReconcileNeverWrites`）、（c）将来この方針を覆す場合はADRを先に通す。
- **独立Expected**：C3-a `TestReconcileNeverWrites`（既存）が維持される。C3-b 新しいPRが`internal/documentdraft`へ書込みを追加すると、方針違反としてレビューで落ちる（ADR未通の場合）。
- **非目標**：文書からの変更提案レーン（欲しい場合は別IssueでOwner判断）。

### C-4 再投影のマージ方針（Owner判断ゲート）

- **現状**：モデル改版時は**新しいDraftを別文書として提示**する。著者編集と新Draftの突合せは行わない。
- **判断が必要なこと**：(a) 現状維持（別文書提示）、(b) 著者編集を保ったまま生成領域だけ差し替える、(c) 3-way merge。**(b)は「著者文を自動上書きしない」要件と両立するかが論点**。
- **独立Expected（判断後に実装）**：C4-a 生成領域差し替え時に著者領域のbyte不変。C4-b 生成領域内の著者編集は保持され、再確認として提示。C4-c 衝突時は自動判定せず人的判断へ。
- **Done means**：Ownerが1案を選びADRとして残る。**採用前にmergeを実装しない**。

### C-5 `*.tei.md` のTEI側取込み（Parser未束綝でブロック）

- **現状**：TEI mainのLoaderは未束綝のMarkdown Sourceを`TEI_MD_PARSER_UNBOUND`で拒否する（[#3346](https://github.com/hat47x/tei-platform/issues/3346)・[#2621](https://github.com/hat47x/tei-platform/issues/2621)）。SHOで`*.tei.md`を保存できてもTEI Accepted Sourceではない。
- **最小実装（Parser完成後）**：`tei model document-import -draft <file> -workspace <root>` を**read-only候補化**として追加する。取込みは（1）SHO保存byteのdigest照合、（2）Parser/Role/Source Indexでの解釈、（3）Owner受理、の3段階を別表示にする。
- **独立Expected**：C5-a Parser未束綝のまま実行すると`parser-unbound`（取込み済みと表示しない）。C5-b SHO保存byteとTEIが見たbyteのdigest不一致は`source-byte-mismatch`。C5-c 受理前のCandidateは`external-candidate`のまま（Canonicalにしない）。
- **Done means**：Parser束綝後にC5-a〜cがPASS。**それまで取込み経路は実装しない**。
- **非目標**：新Markdown Grammarの決定（既存Ownerの比較に委ねる）。

## 6. 全体の「完了」定義と停止線

### 段階的拡張の完了＝

1. 段階2〜5が、それぞれ**独立したGo試験とCLI実証**を持ち、前の段階のPASSを流用しない。
2. 各段階の`documentKind`は、未実装のものを`UNSUPPORTED_DOCUMENT_KIND`で拒否し続ける（実装済みとして扱わない）。
3. C-1の相互CIが両OSでsuccessし、artifactのreceiptが読める。

### 常に守る停止線（全段階共通）

- **Draftは派生Projection。** 第二のcanonical model・意味正本・Host権限を作らない。
- **再投影は著者文を上書きしない。** 生成領域内の著者編集は報告のみ。
- **推測で补完しない。** 未宣言・未解決・不明は`unknown`のまま。
- **同名・同bytes別所在で同一意味と推定しない。** Identityは`kind:id`＋byte照合。
- **保存成功をモデル反映成功にしない。** `claims.teiModelIngested`は常にfalse。
- **局部PASSを製品受入・Owner受理に昇格しない。** CI成功は「そのcommitの試験が再現された」ことのみ。
- **非公開path/count/byteをエラー・ログ・artifactへ漏らさない。**

## 7. 最初の3手（次実装者が最初にやること）

1. **C-2（TEI Studio実組込み）** — 相互CIの証跡が出て、部品境界（[起動契約](https://github.com/hat47x/sho-composer/blob/feat/tei-document-draft-core-20261111/docs/sho-doc-component-boundary-2026-11.md)）が安定した。子プロセス／Wails資産のA/B測定をする。
2. **C-4（再投影方針）のOwner判断起案** — 実装より先に判断。
3. **段階3（`three-element-diagram`）** — [PR #4283](https://github.com/hat47x/tei-platform/pull/4283) の統合後に入力契約（`model.inputKind`）を決める。統合前に着手しない。

C-3は全段階で常に維持。C-5はParser（[#3346](https://github.com/hat47x/tei-platform/issues/3346)）の完了待ちで、**着手しない**。

## 8. Owner判断が必要な事項（一覧）

| # | 事項 | 影響する段階 | 現状の既定案 |
| --- | --- | --- | --- |
| 1 | Draft引用（`recorded-not-resolved`）とSHO引用Registerの関係 | 全段階 | 別モードとして併存。統合するかはOwner判断 |
| 2 | Occurrence IDの永続性（現状はsnapshot-local） | 段階4・5 | 永続化しない。必要ならADR |
| 3 | 固定納品版のpinの意味と再納品条件 | 段階5 | pin後は失効報告のみ、自動更新しない |
| 4 | SHO文書編集部品の配置（SHO standalone／TEI Studio組込み） | C-2 | CLI部品境界。組込み形態は測定後に決定 |
| 5 | 文書→モデルの変更提案レーンの要否 | C-3 | 経路を作らない（既存の変更計画・Owner受理のみ） |
| 6 | CRUD・三要素の意味的解釈（業務意味） | 段階2・3 | [THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md](../issues/THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md)の未決事項のまま |
