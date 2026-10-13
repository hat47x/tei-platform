# #1124 次のマイルストーン：四視点の受理境界

## 現在地

PR #1273 は Studio の表・ER・Function・参照用Businessの表示、Source Indexアンカー照合、変更候補、Go Shared Acceptance審査、Scratch受理・再投影、実Chrome編集の自動試験をmainへ統合した。

ただし、現在のBusiness DFDは `reference-only` であり、Procedure / Step はCanonical Sourceとして受理されていない。単なるID一致は同一revisionの受理済みBusinessの証拠にならない。Studioが独自にProcedureの正本を新設してはならない。

## 統合する契約の提案

1. **Source snapshot**：同一Workspaceから取得した `modelRevision` と `sourceIndex` を単一の読取境界に束縛する。途中でrevisionが変わればfail closed。
2. **Business authority**：#1125が提供する受理済みProcedure/Stepの定義・Source locator・revision・受理状態を使う。現在のDFD fixtureからの導出値を `accepted` に昇格しない。
3. **Typed identity**：#1120のscope/kind/ID契約を使い、`step:BP-05` のようなReference-local keyを正式なグローバルIDとして流用しない。
4. **Four projections**：表・ER・Function・Businessを同一snapshotから投影し、参照先ID、Derived/Accepted、source pointer、revision、未解決参照を表示する。Business authorityが無い場合はBusiness viewを明示的に `unqualified` とする。
5. **Proposal**：Studio UI/CLIとAIの候補は同じShared Acceptanceへ渡し、Preview/Review/Acceptの権限境界を複製しない。編集可能なSource locatorとDerived表示を分離する。
6. **Reprojection**：Accept後は新revisionを再読込して四視点を作り直す。古いrevisionの候補・審査・Undo操作は拒否する。Undoは以前のSourceを暗黙復元せず、新たな審査・受理を要する候補として扱う。

## 受入シナリオと否定例

| 対象 | 合格条件 | 拒否すべき状態 |
| --- | --- | --- |
| 同一IDナビゲーション | 四視点のtyped IDとsource pointerが同じrevisionに結合 | 同名別scope、古いDFD、別revision |
| ER編集 | Relation変更候補→Validation/Impact→Shared Acceptance→R1再投影 | 存在しないtarget、Derived直書き |
| 表編集 | Validation変更候補→同一Acceptance→R2再投影 | 古いReview指紋、別revision |
| Business影響 | 受理済みProcedure/Stepから明示参照を追跡 | 非Canonical DFDをAcceptedと表示 |
| 並行編集 | 同一R0からの2候補で先行受理後、後続はstale拒否 | 失敗時のSource変更、暗黙再base |
| 手動/AI | originの差に関係なく同じ審査・受理・記録 | AI案の自動受理、画面独自の受理経路 |
| 実画面 | 実Chrome/WebView操作と受理後のR1表示を確認 | Node VMだけを人間の承認と表現 |
| 性能 | 実業務分布で表示遅延と操作応答を記録 | 合成10k件を製品SLOとみなす |

## 実装着手条件

- #1125で受理済みBusiness Procedureの正式Source契約が提供されるまでは、Studio側は既存の参照専用表示を維持し、独自のAccepted Business schemaを作らない。
- #1120のtyped referenceに適合する識別子を優先する。互換アダプタが必要ならReference境界に限定し、coreの意味を変えない。
- #1121の実行・Provider側の動作をStudioの表示試験で証明したと主張しない。
- Human Acceptance、実Provider、実業務分布SLOは自動CIのPASSと別のEvidenceとして残す。

この文書は**設計・受入計画**であり、実装完了や正式受理の宣言ではない。
