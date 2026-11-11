# 三要素分析モデル実装：Owner受理が必要な事項と未対応表現

この文書は、`internal/threeelement` パッケージおよび `tei three-element` CLI の実装により到達した現在地点を記録し、Ownerによる業務意味の受理が必要な事項や、未対応の表現を整理する。

## 受理済みの前提

- 三要素分析法は渡辺幸三氏の手法に基づき、要素は **Data / Function / Business** の三つとする（ADR-0033）。
- 既存の `tei.xeadsource.cross-view/v4` 派生方針は読み取り専用Projectionとして参照するが、本パッケージは独立した標準データモデルを提供する。
- TEI Modeler上での図面編集は将来の標準操作として想定するが、本実装はCLI/機械可読操作を優先する。

## Owner受理が必要な事項

1. **CRUDの语义的解釈**
   - 現実装はCRUDを「宣言された操作」または「FunctionElement.Operationsから派生した操作」として扱う。
   - Business Procedureのライフサイクル遷移（例: 注文の確定→キャンセル）をCRUDのU（Update）へ短絡するかどうかは、Ownerの業務意味に依存する。
   - 関連: #1125, #1129, #3263

2. **同一名称の別要素の取扱い**
   - 同一名称を持つ別要素（例: "Order"という名前のData EntityとFunction）は、安定したIdentity（`kind:id`）で区別する。
   - 図面上の表示名（label）とConfusion防止のためのdisambiguation rulesは、Ownerが受理する必要がある。
   - 関連: #2495, #2497, #2620

3. **複合キーと関数従属性**
   - DataElementは複合キー（`compositeKey`）を保持するが、関数従属性（FD）や候補キーの受理は別途 #1126 の担当。
   - CRUD表生成時に複合キーが参照整合性に与える影響は、Ownerが評価する必要がある。

4. **多重関係・有向関係の意味**
   - DataRelationは`directed`と`kind`（one-to-many, many-to-many等）を持つが、業務上の多重度（cardinality）の解釈はOwnerの受理を要する。
   - 関連: #1126, #2495

5. **Business-ActivityとFunctionの非対称対応**
   - 一つのBusiness Activityが複数Functionを呼び出す場合、また一つのFunctionが複数Activityから使われる場合の対応関係は、業務フローの粒度に依存する。
   - 現実装は宣言された`functionIds`をそのままCrossReference化するが、業務意味での「同じ処理」の識別はOwnerが行う。

## 未対応の表現

1. **グラフ・図面のレイアウト**
   - 本パッケージはモデルの意味構造を保持するが、図面上のレイアウト（位置、色、形状）は持たない。
   - 図面編集はTEI Modelerの将来標準操作として想定し、本実装では範囲外とする。

2. **時点・版の管理**
   - ThreeElementModelは特定の時点や版を識別するフィールドを持たない。版管理はtei-platform全体のWorkspace/World/Source Indexの責任範囲とする。

3. **認可・ロール**
   - BusinessElementの`role`フィールドは文字列として保持するが、認可（Grant）やロール解決（Role Provider）とは独立している。
   - 実行時の権限制御は #951, #1120 の担当。

4. **外部X-TEA/EKIモデルの取込み**
   - 本パッケージはTEI標準データモデルを前提とする。X-TEA/EKIの既存モデルを変換するAdapterは別途設計が必要。

5. **CRUDの網羅性保証**
   - GenerateCRUDTableは宣言されたFunction-Data関係からCRUD表を生成するが、「全ての必要なCRUDが宣言されている」ことを保証するのは静的解析の限界である。
   - 不足検出（CRUDGap）はCrossReferenceとの比較による候補提示に留まる。

## 次のステップ

- [ ] Ownerによる上記事項の受理
- [ ] #3263（三視点Diagram/CRUD製品受入）への接続
- [ ] #1124（統合モデラー）との編集経路統合
- [ ] #1125（業務手順）とのfunction mapping粒度調整
- [ ] #1126（Data/Relation）との複合キー・FD統合
