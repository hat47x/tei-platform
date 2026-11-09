# 受注承認と出荷準備の一階層DFD用Reference

#1125向けの**TEIオリジナルの合成ケース**です。既存資料の転載ではありません。案件で採用された業務仕様でも、動作するWorkflowでもありません。

## ケースの範囲

受注内容の閲覧→顧客への電話確認→外部在庫情報の確認→人による承認・差戻し→受注コマンドの依頼→倉庫への条件付き業務引継ぎ→別の外部状態観測、という一巡です。

- `order-approval-procedure.json` は手作業を含むBusiness ProcedureのReference-local authoring fixture。2件のApplication Pattern参照以外はFunctionを要求しません。
- `warehouse-preparation.json` は倉庫側の独立したProcedure。倉庫側の現物準備もアプリ機能へ強制変換しません。
- `procedure-set.json` は両Procedureを明示的に宣言し、条件付きのHandoffを一件記録します。
- `review-proposal.json` は**未受理**の人・イベント・時間による起動契機、分岐・例外の経路、業務Role、外部相手方、概念Store、情報の授受を記述します。

画面の配置は保存せず、Sourceと切り離した派生ビューを生成します。

## 再現コマンド

リポジトリのルートから次を実行します。

```sh
go run ./tools/reference-business-dfd \
  -procedure reference/business-system-design/dfd-order-approval/order-approval-procedure.json \
  -review reference/business-system-design/dfd-order-approval/review-proposal.json \
  -set reference/business-system-design/dfd-order-approval/procedure-set.json \
  -format json

go run ./tools/reference-business-dfd \
  -procedure reference/business-system-design/dfd-order-approval/order-approval-procedure.json \
  -review reference/business-system-design/dfd-order-approval/review-proposal.json \
  -set reference/business-system-design/dfd-order-approval/procedure-set.json \
  -format dot > /tmp/order-approval-dfd.dot
```

`dot -Tsvg /tmp/order-approval-dfd.dot` で可視化できます（Graphvizは任意）。

## レビューに必要な境界

差戻し経路を図示しても承認規程が確定したことにはなりません。Business RoleとAuthorization Roleは別であり、`APP-40`のCommand呼出し成功は`WH-00`の倉庫準備や`APP-60`の現実状態を証明しません。従ってHandoffを自動実行しません。

情報Flowの `payload` は説明用の提案であり、永続化の意味を持ちません。Dataのexact ID解決に成功しても、物理DB・書込み権限・実効性までは確定しません。

既存のSource Acceptance、Studio双方向編集、trigger/schedule runtime bindingの実証は別Issue担当です。このデモのGo試験は `TestOrderApprovalAndWarehousePreparationDemonstration` にありますが、GitHub上の追加時点でテスト実行済みとは扱いません。
