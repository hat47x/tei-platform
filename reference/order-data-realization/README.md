# データ定義から実DBまで(order-data-realization)

データ担当者が、検証用に複製した受注業務の作業領域で受注に属性を一つ追加し、その定義を同じ版のまま実PostgreSQLの移行、一覧と親子明細の取得、登録と別接続での再読込まで通す技術検証用サンプルです。業務キーの受理と一意制約の宣言という二つの判断が、期待状態、移行の差分、登録時の重複拒否まで同じ版で効くかを確かめます。拡張として、旧版の書き手と共存する移行と、同じ最終版をSQLiteへ実現したときの値の往復を加えます。

- 状態: 設計案は提案段階です。工程1〜10を一つの通しの記録としてつなぐ実装を[`tools/reference-order-data-realization`](../../tools/reference-order-data-realization/README.md)に置き、使い捨ての実PostgreSQLとPython標準の`sqlite3`で通しの検証を行いました。実環境での確認ではありません。属性の追加と一意制約の宣言は複製への直接編集で、共通受理を通ったのは業務キー(候補キー)だけです。
- 設計の正本: [技術検証用サンプルの統合設計](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md)の4.3節。技術検証用サンプルの位置づけは同じ文書の1.2節、要件の目録は2章、資産の対応表は6章にあります。DBの共通層と方言層の分離は[データベースプラグインの方言契約設計](../../plan/design/DATABASE_DIALECT_PLUGIN_DESIGN_2026-10.md)が扱います。
- 各資産の使い方と実装状況は、それぞれのREADMEが正本です。このREADMEは工程との対応を案内するだけで、内容を写しません。

## 工程

「現状」は、通しの検証([`tools/reference-order-data-realization`](../../tools/reference-order-data-realization/README.md))で確かめた区分です。工程の間は、版の照合値(最終版R3)で受け渡します。既存資産を呼べた工程と、呼べずに同じ内部部品を直接使った工程の別と理由は、実装のREADMEの「工程と既存資産」にあります。

| 区分 | 意味 |
| --- | --- |
| 実装あり | コードはあるが、資産の中に試験が見当たらない |
| 合成データで試験 | 合成データ、検証用の作業領域、使い捨てのDBを使う試験や受領記録が資産にある |
| 通しで確認 | 検証用に複製した作業領域と、検証器が起動した使い捨ての実PostgreSQL(工程10はPython標準の`sqlite3`)で、通しの検証の中で確かめた。実環境での確認ではない |
| 実環境で確認 | 実際の提供元や運用環境で確かめた記録がある |
| 足場なし | 工程を担う既存資産がない |
| 未確認 | 資産のREADMEから状況を確かめられなかった |

| 工程 | 内容 | 前工程から受け取るもの | 使う既存資産 | 現状 |
| --- | --- | --- | --- | --- |
| 1 | 属性の追加と版の記録 | なし(作業領域の複製、変更前の版R0) | なし。共有入力は[受注業務の作業領域](../workspace/)(複製して使う)。共通受理に属性追加の操作がないため、サンプル側で複製を直接編集する | 通しで確認(受理は未接続) |
| 2 | 設計質問への回答と業務キーの受理 | 属性を追加した複製と版 | [reference-example-slide](../../tools/reference-example-slide/README.md)、[reference-shared-acceptance](../../tools/reference-shared-acceptance/README.md)(`add-candidate-key`) | 通しで確認 |
| 3 | 論理ERと物理表の対応、一意制約の判断 | 候補キーを受理した版 | [reference-er-table-crossview](../../tools/reference-er-table-crossview/README.md)。一意制約の宣言は保存写像への直接編集 | 通しで確認(一意制約の受理は未接続) |
| 4 | 標準機能候補の生成 | 最終版の複製 | [reference-standard-function-candidates](../../tools/reference-standard-function-candidates/README.md) | 通しで確認 |
| 5 | 旧構造のDBの構築、差分の検出、移行の計画 | 最終版とR0から導く期待状態 | [reference-ddl-entry](../../tools/reference-ddl-entry/README.md)。[reference-postgres-integration](../../tools/reference-postgres-integration/)と[reference-postgres-migration-rendering](../../tools/reference-postgres-migration-rendering/)は呼べない(同じ内部部品を直接使う) | 通しで確認 |
| 6 | 移行前の登録の拒否 | 旧構造のDBと期待状態、工程4の不足項目 | [reference-data-first-persistence](../../tools/reference-data-first-persistence/README.md)は呼べない(受付依頼に固定)。同じ段階の実行前確認を受注へ当てる | 通しで確認 |
| 7 | 移行の適用と再観測での収束 | 工程5の計画と観測、工程6で拒否された旧構造のDB | [reference-scenario-state-diff](../../tools/reference-scenario-state-diff/README.md)。適用は内部の入口を直接使う | 通しで確認 |
| 8 | 一覧と親子明細の取得 | 収束したDB | 一覧の入口を新設。[reference-postgres-header-detail-runtime](../../tools/reference-postgres-header-detail-runtime/README.md)は呼べない(判定段階Bが固定)。同じ内部の入口を一覧の結果から呼ぶ | 通しで確認 |
| 9 | 登録、別接続での再読込、重複拒否 | 工程8のDBと登録前の一覧、工程4の不足項目 | 工程6と同じ | 通しで確認 |
| 10 | 第二の製品(SQLite)での実現と値の往復(拡張) | 最終版と、工程9で登録した受注と境界値 | なし。SQLiteの提供元がないため、Python標準の`sqlite3`を別プロセスで使い、方言の対応をサンプルの宣言に置く | 通しで確認(TEIのSQLiteの提供元ではない) |

補足します。

- 工程5と7の二つのツール(`reference-postgres-integration`、`reference-postgres-migration-rendering`)は、`package main`で表を手組みし、無条件の削除を持つため、通しでは呼んでいません。通しは同じ内部部品(期待状態の導出、観測、差分、移行の計画と適用)を直接使います。
- 工程6と9の資産は受付依頼(`examples/first-workspace`)に固定されています。通しは同じ段階の実行前確認と、駆動部の実行・独立の読取りの入口を受注へ当てます。登録のアダプターは手書きで、受注の契約の書込み範囲は検査していません。
- 工程8の一覧取得を担っていた2件は#1197で削除済みです。通しは駆動部の一覧の入口で受注の一覧を取得します。
- 工程10は、TEIのSQLiteの提供元がないまま、サンプルの宣言とPython標準の`sqlite3`で確かめました。確かめたことと確かめていないことは、[実装のREADME](../../tools/reference-order-data-realization/README.md#sqliteで確かめたことと確かめていないこと)にあります。

## 取り込むピーキーな要件と失敗注入

注入する失敗と合否の観測の詳細は、設計文書の4.3節にあります。要件の内容は2章の要件目録を参照してください。「固有」は目録にない、このサンプル固有の困難です。

| 要件 | 工程 | 注入する失敗(要点) | 観測する合否(要点) |
| --- | --- | --- | --- |
| P-MIG-03 | 6 | 列のない旧構造のDBへ、変更後の定義で登録する | 登録用アダプターを呼ばずに拒否し、既存の行が変わらない |
| 固有 | 5・7 | 計画から適用までの間に、別の担当者が表の構造を変える | 計画時と適用直前の観測が食い違えば停止する |
| 固有 | 2・3・9 | 業務キーと物理の主キー・一意制約が一致しない | 主キーから業務キーを推測せず、宣言した場合だけ一意制約が現れる |
| 固有 | 9 | 二つの接続が同じ外部番号をほぼ同時に登録する | 一件だけ成立し、確定した失敗と結果不明を区別する |
| P-TX-02 | 9 | 登録のコミット後に、再読込の接続や表が失われる | 登録の結果と別接続の観測を別の根拠として扱う |
| P-MIG-03 | 7 | 移行の適用中に、旧版の登録処理が書き込み続ける(拡張) | 旧版の行でも新しい属性が宣言どおりになり、縮約が拒否される |
| P-INT-03 | 9 | 取消しで欠番を作り、外部番号の採番元を止める | 欠番がすべて説明でき、採番元の停止時に宣言どおり振る舞う |
| P-DB-03 | 10 | 同じ最終版をSQLiteへ実現し、境界値を往復させる(拡張) | 差分0件か、宣言した既知の差分だけ |

## 通しの完了条件

次の条件を、通しの検証(`python3 tools/verify-order-data-realization.py`)で確かめます。合否と観測値は、検証器が出す通しの記録(`chain-record.json`)の`completionConditions`と`faultInjections`に入ります。条件ごとの確かめ方は[実装のREADME](../../tools/reference-order-data-realization/README.md#通しの完了条件)にあります。拡張の条件(最後の項目)も含めて成立しましたが、SQLiteの側はTEIの提供元ではなく、サンプルの宣言とPython標準の`sqlite3`による確認です。

- 属性追加と二つの判断(業務キーの受理、一意制約の宣言)が、最終版に結び付いたまま、期待状態、移行の差分、一覧、登録のすべてに使われる。別の版や別の複製を読めば、検証器が停止する。
- 二つの判断を行った版と行わなかった版で、工程5の差分と工程9の結果が異なることを、一つの実行系列で示す。
- 変更前の構造のDBへの登録は実行前に拒否され、移行と再観測での収束の後に同じ登録が通る。既存の行は失われない。
- 一覧で選んだ受注の識別子が、そのまま親子明細の取得へ渡る。
- 登録の結果と別接続での再読込が別々の根拠として記録され、再読込に失敗した登録は結果不明として残る。
- 検証器は工程をまたぐ使い捨てのDBを一つだけ使い、開始前に空であることを確かめ、終了時に所有印のあるものだけを削除する。
- (拡張)旧版と共存する移行が宣言どおりに進み、SQLiteで読み戻した値がPostgreSQLの値と一致するか、宣言した既知の差分だけである。SQLiteの側はTEIの提供元ではなく、サンプルの宣言とPython標準の`sqlite3`で確かめる。

## 部分工程の診断(互換入口)

次の資産は、工程の一つの境界を単独で確かめる診断として、位置と入口を変えずに残します。実行方法は各資産のREADMEを参照してください。

| 工程 | 資産 | 確かめる境界 |
| --- | --- | --- |
| 1(作法の手本) | [first-workspace](../../examples/first-workspace/README.md)、[first-capability-effect](../../examples/first-capability-effect/)(READMEなし)、[first-value-ux-audit](../../tools/first-value-ux-audit/README.md) | 別のデータ(受付依頼)の最小例。工程の成否には使わない |
| 1 | [reference-model-search](../../tools/reference-model-search/README.md)、[reference-list-roundtrip](../../tools/reference-list-roundtrip/README.md)、[reference-partial-import-plan](../../tools/reference-partial-import-plan/README.md)、[reference-type-reorg-impact](../../tools/reference-type-reorg-impact/README.md)、[reference-format-version-migration](../../tools/reference-format-version-migration/README.md) | 意味モデルの編集を変更案として出す。手作業の編集は置き換えない |
| 3 | [reference-design-output](../../tools/reference-design-output/README.md)、[reference-table-definition-document](../../tools/reference-table-definition-document/README.md)、[reference-function-definition-document](../../tools/reference-function-definition-document/README.md)、[reference-access-matrix](../../tools/reference-access-matrix/README.md) | 受理済みの意味から設計文書を導く |
| 5 | [reference-ddl-entry](../../tools/reference-ddl-entry/README.md) | 提供元が出すDDLを表ごとに取り出す(P-DB-03) |
| 7 | [reference-scenario-state-diff](../../tools/reference-scenario-state-diff/README.md) | 移行の前後で既存の行が変わらないことの確認 |
| 8 | [reference-postgres-header-detail-runtime](../../tools/reference-postgres-header-detail-runtime/README.md) | 親子明細の取得(判定段階Bが固定。既定の動作と出力は変えない) |

## 基盤への改善圧力

詳細(現状、求める条件、確認方法)は設計文書の4.3節「基盤への改善圧力と現状」にあります。通しの検証で、基盤にないためサンプル側で補ったものは、[実装のREADMEの「基盤の不足」](../../tools/reference-order-data-realization/README.md#基盤の不足)に記録しています。

| 要件 | 不足 |
| --- | --- |
| 固有 | 意味の変更を受理版として同定し、物理への影響を分類する操作がない |
| 固有 | 移行計画の観測の照合値を、適用の前に確かめていない |
| P-MIG-03 | 保存の実行前確認が受注に使えず、操作と保存写像と契約の対応が未検査である |
| 固有 | 期待状態を保存写像から導く経路が、移行と差分の確認に接続されていない |
| 固有 | 定義にない物理側の列を、意味と結んで扱う条件がない |
| 固有 | 使い捨てDBの隔離と破棄が部品ごとに別実装で、工程をまたげない |
| P-DB-03 | 共通層と方言層の分離が設計案にとどまり、二つの製品で同じ一覧を確かめられない |
| 固有 | 通しの検証根拠がなく、実DBで確認した範囲が文書に反映されていない |
| P-MIG-03 | 旧版の書込みと共存する移行手順が提供元にない |
| P-INT-03 | 業務キーの採番と欠番の扱いを宣言できない |
