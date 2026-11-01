# 表示変更から業務操作候補への逆対応（合成 Reference）

対象：[Issue #2500](https://github.com/hat47x/tei-platform/issues/2500)。

viewintent.Interpret は参照用の純粋関数であり、実行・永続化・受理・権限発行の機構ではない。正式な TEI Core kind、永続 Schema、公開 API の採用案でもない。

## 固定する意味の差

- **元データの実体とUI上の行の出現箇所**：JOIN により L1 が3回見えても、明示的に同じ Scope/Kind/ID に束縛された単一項目変更は一つの候補になり得る。業務上の重複関係や無名行の同一視を、この試作で証明しない。
- **実体の変更と Relation occurrence の変更**：明示 edge ID を持つ unlink は relation が対象であり、端点ノードの delete に変換しない。edge が識別できなければ未解決。
- **集計結果と元データの可逆な属性**：集計点の変更から元明細への配賦を推定しない。業務用の明示配賦 Action は別件。
- **全件を必要とする検証と明示 ID の一件操作**：既知の部分取得を全件検証へ使わない。一件候補を返せても実処理での Authority/Rule/Version/Commit/Receipt は別。
- **View時点と現在の意味・Action版**：意味改版（[#2505](https://github.com/hat47x/tei-platform/issues/2505)）やContext Instance/選択 epoch 変更後の旧Changeを無条件に再解釈しない。別版への rebase は設計・Ownerの明示判断を要する。

## 限定試験

リポジトリルートで以下を実行する。

    go test ./reference/view-change-intent/...

標準 Go ライブラリのみ。race 検査でも実行できる。実 DB、UI、本番 Provider、外部 Host の存在や、元の参照モデルの業務意味に依存しない独立した合成 fixture。

この試作は candidate を返せるだけで、値の型検証、Sourceの真正性、現在権限、Ownerの意味受理、実Effect、Business Completion、Commit応答喪失後の状態は保証しない。後段は #1127/#1129/#1174/#1163 が所有する。

**実測記録**：作成者の隔離ディレクトリ（Go 1.23.2／モジュールモード無効）でこの3ファイルの focused test 12件、race、vet を実行し成功。リポジトリが指定する Go 1.26.10 による full health、CI、実Target・Host試験は未実施。PRの exact head と統合判定は別途確認する。
