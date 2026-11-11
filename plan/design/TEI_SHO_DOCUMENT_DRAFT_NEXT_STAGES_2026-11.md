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
| SHO側の消費・編集・保存・再開 | SHO PR #36、17試験PASS・merge後スイート105 PASS/0 FAIL |
| 相互実証（合成Workspace・同一端末） | PR本文の記録 |

| 未実装 | この文書の段階 |
| --- | --- |
| CRUD表・三要素分析図 | 段階2・3 |
| 仕様書（複数要素の束ね） | 段階4 |
| 固定納品版（Revision pin） | 段階5 |
| TEI Studio実組込み | 横断C-2 |
| 相互CI証跡（同一Attempt） | 横断C-1 |
| 文書→モデルの逆方向経路 | 横断C-3（経路は作らない） |
| `*.tei.md` のTEI側取込み | 横断C-5（Parser未束綝でブロック） |
| 再投影のマージ方針 | 横断C-4（Owner判断ゲート） |

## 1. 段階2：`crud-table`

- **目的**：`internal/threeelement` のモデルJSONを第2入力として、Function→DataのCRUD表をDraft化する。[#3263](https://github.com/hat47x/tei-platform/issues/3263)の受入条件（Source identity・多重辺・loss・read-only逆編集拒否）を再利用する。
- **依存**：段階1。`internal/threeelement`（main統合済み）。**CRUDの意味的解釈は[THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md](../issues/THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md)のOwner未決事項のまま**（本段階は投影のみ）。
- **触るファイル**：`internal/documentdraft/`（第2入力と`crud-table`投影）、`cmd/tei/cli_document_draft.go`（`-three-element-model`）、`internal/threeelement/`（原則変更しない）。
- **契約の差分（最小）**：`Request`に入力種別（workspace／three-element model file）を追加。`ModelRef`は**入力種別で中身が変わる**ことを`model.inputKind`で明示する（Workspace readとmodel file readを同じ形に潰さない）。`documentKind: "crud-table"`。
- **最小実装**：`threeelement.GenerateCRUDTable` の各行をOccurrence化（1セル＝1Occurrence、`subjectRefs`に関数とデータのIdentity、`generationConditions.rule="crud-cell-v0"`）。表全体のOccurrenceは作らない（セル単位で追跡するため）。
- **独立Expected**：
  - S2-a 明示CRUD（`Source: declared`）と派生CRUD（`derived`）を`generationConditions`で区別する。
  - S2-b 未宣言のCRUDは作らない（網羅性を主張しない）。
  - S2-c 同一function-data対で矛盾するCRUDは`contradictory`として報告し、採用しない。
  - S2-d モデルJSONを改変（1行変更）→ reconcileで該当セルが`content-changed`、他は`unchanged`。
  - S2-e 未実装の`documentKind`（`three-element-diagram`等）は`UNSUPPORTED_DOCUMENT_KIND`（段階1の既存挙動を維持）。
- **Done means**：S2-a〜eがGo試験でPASSし、CLIで実CRUD表Draftが得られる。
- **非目標**：CRUD網羅性の保証・図面レイアウト・Owner意味受理。

## 2. 段階3：`three-element-diagram`

- **目的**：Data/Function/Businessの**構造記述**（node/edge identity・方向・多重度・出典）をDraft化する。レイアウト・色・形状は持たない（[THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md](THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)と同じ扱い）。
- **依存**：段階2（第2入力）。[#1124](https://github.com/hat47x/tei-platform/issues/1124)・[#3263](https://github.com/hat47x/tei-platform/issues/3263)の受入条件。
- **最小実装**：`diagram`種別のOccurrenceに`nodes[]`（identity・kind・出典）と`edges[]`（from/to identity・relation・多重度・resolution）を入れる。SHO側は`diagram`を**構造記述として保存するだけ**（描画・編集は別資格）。
- **独立Expected**：
  - S3-a 平行edge（同じfrom/to・別relation identity）を縮約しない。
  - S3-b 未解決targetは`unresolved`のまま（推測で接続しない）。
  - S3-c 同名の別要素（Data "Order" と Function "Order"）を`kind:id`で区別する。
  - S3-d モデル改変で影響するedgeのみ`content-changed`（`AnalyzeImpact`の結果と一致）。
- **Done means**：S3-a〜dがGo試験でPASS。**図面編集は作らない**。
- **非目標**：Canvas・レイアウト・Renderer・可逆図上編集。

## 3. 段階4：`specification`

- **目的**：複数要素・複数関係を一つの文書に束ね、章構成を持つ仕様書Draftを作る。
- **依存**：段階1。`subjects[]`は既に複数件を持てるので、追加は**束ね規則と章立て**のみ。
- **最小実装**：`Request`に要素・関係のリスト（順序付き）を受け取り、章＝1要素、節＝1関係、付録＝引用、とする決定的な並びにする。`meaningInputRange`（対象とした版の範囲）をDraftに明記する。
- **独立Expected**：
  - S4-a 同一入力から決定的な同一Draft（順序・digest安定）。
  - S4-b 要素が1件でも未宣言なら**その章を作らず**、`skippedSubjects`に理由を残す（全体を失敗にしない）。
  - S4-c 関係の対象が別要素なら、その要素も`subjects`に加わる（引用の切れを作らない）。
  - S4-d 部分的な章の欠落は`needsReconfirmation`に出る（黙って省略しない）。
- **Done means**：S4-a〜dがGo試験でPASS。
- **非目標**：目次の自動生成・表現の潤飾・Owner意味の統合判断。

## 4. 段階5：`frozen-delivery`

- **目的**：固定納品版。Draftが**特定Revisionにpin**され、以後のモデル改版は「失効」として報告され、自動更新しない。
- **依存**：段階1。**pinの意味（再確認の要否・再納品の条件）はOwner判断**。
- **契約の差分（最小）**：`Draft.frozen`（bool）と`Draft.frozenAtModelRevision`。Reconcileは`frozen && 参照が動いた`場合に`modelReference.status="stale"`かつ`detail`に`frozen-delivery-reference-moved`を付与し、`authorityLimits`に`frozen-delivery-is-not-refreshed-automatically`を追加する。
- **独立Expected**：
  - S5-a pin後・モデル不改変 → `current`。
  - S5-b pin後・モデル改変 → `stale`＋Occurrence単位の再確認リスト。**Draftも文書も書換わない**。
  - S5-c `frozen`を手で外したDraftは`INVALID_DRAFT`（pinの解除は明示操作のみ。手編集で解除できないことを試験で固定するなら、解除は新Draft生成時にのみ可能とする）。
  - S5-d 別Workspaceの同じ内容（同`worldFingerprint`）でも`modelRevision`が違えば`stale`（Workspace identityの違いを報告）。
- **Done means**：S5-a〜dがGo試験でPASS。
- **非目標**：納品物の署名・配布・受領証。

## 5. 横断の課題（段階と独立）

### C-1 相互CI証跡（同一Attempt）

- **目的**：「TEI実CLI → SHO実CLI → TEI実reconcile」を**CI上で同一Attempt**として実行し、ローカル実証だけにしない。
- **最小実装**：TEI側のGitHub Actionsに1ジョブ追加。SHO repositoryを**pinしたSHA**でcheckoutし、合成Workspace（testdata）でround tripを実行、JSON receiptをartifactとして保存する。SHO側の変更は追従しない（pinを明示的に更新するPRを別に出す）。
- **独立Expected**：C1-a SHO pin SHAを変えるまで同一receipt。C1-b SHO側の破壊的変更（CLI名・フラグ変更）でFAIL。C1-c 合成データのみ（顧客データ・非公開资料なし）。
- **Done means**：両OS（Ubuntu/Windows）でsuccessし、artifactのreceiptが読める。
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

1. **段階2（`crud-table`）** — 第2入力の型（`model.inputKind`）を先に決める。ここが後続3段階の入力契約の土台になる。目安：`internal/documentdraft`への入力種別追加＋セルOccurrence化＋5テスト。
2. **C-1（相互CI）** — 段階2の実証と同時に組める。SHO pin SHAを固定した1ジョブでround tripを回し、ローカル実証をCI証跡に昇格する。
3. **C-4（再投影方針）のOwner判断起案** — 実装より先に判断。段階4・5の受入条件がこれに依存する。

C-2（Studio組込み）はC-1のCI証跡が出てからでよい（部品境界が安定してから測定する）。C-3は全段階で常に維持。C-5はParser（[#3346](https://github.com/hat47x/tei-platform/issues/3346)）の完了待ちで、**着手しない**。

## 8. Owner判断が必要な事項（一覧）

| # | 事項 | 影響する段階 | 現状の既定案 |
| --- | --- | --- | --- |
| 1 | Draft引用（`recorded-not-resolved`）とSHO引用Registerの関係 | 全段階 | 別モードとして併存。統合するかはOwner判断 |
| 2 | Occurrence IDの永続性（現状はsnapshot-local） | 段階4・5 | 永続化しない。必要ならADR |
| 3 | 固定納品版のpinの意味と再納品条件 | 段階5 | pin後は失効報告のみ、自動更新しない |
| 4 | SHO文書編集部品の配置（SHO standalone／TEI Studio組込み） | C-2 | CLI部品境界。組込み形態は測定後に決定 |
| 5 | 文書→モデルの変更提案レーンの要否 | C-3 | 経路を作らない（既存の変更計画・Owner受理のみ） |
| 6 | CRUD・三要素の意味的解釈（業務意味） | 段階2・3 | [THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md](../issues/THREE_ELEMENT_OWNER_ACCEPTANCE_ITEMS_2026-11.md)の未決事項のまま |
