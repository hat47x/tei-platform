# TEI Action接続プロファイル v1（参照契約）

## 適用範囲

この文書は、**アプリケーションが所有するリソースへの条件付き変更操作**を、共通のAction境界へ渡すための参照契約です。汎用Action全体の仕様ではありません。読取り操作、リソースを持たないコマンド、複数リソースをまたぐトランザクションは別のプロファイルで検討します。

対象はSUIのCard移動に限りません。Inventoryの在庫調整など、異なるアプリケーションも同じ外側の構造を使います。UIのプレビューやドラッグ操作は対象外で、**利用者が変更を確定した時点**にだけ要求を送ります。

- 要求契約の正本：`action-protocol-v1.schema.json`（JSON Schema Draft 2020-12）
- 応答契約の正本：`action-result-v1.schema.json`（同規格。成功の`revision`と失敗の`error`は排他的）
- ブラウザー参照実装：`native-action-bridge.mjs`と`json-action-dispatch.mjs`
- HTTP参照実装：`actionboundary.JSONCommitHandler`
- 実行境界：`actionboundary.Host.Commit`

これはTEI内で検証中の標準化候補です。外部標準化団体が策定したAction規格であるとは主張しません。

## 共通の要求形式

```json
{
  "protocolVersion": "1",
  "applicationID": "inventory",
  "resourceID": "warehouse-a/item-42",
  "actionID": "inventory.adjust",
  "expectedRevision": "v7",
  "payload": {
    "sku": "item-42",
    "delta": 4
  }
}
```

| フィールド | 規則 |
| --- | --- |
| `protocolVersion` | 必須。参照実装が受け付ける値は文字列の`"1"`だけです |
| `applicationID` | 必須。ActionとPayloadを所有するApplicationの識別子です |
| `resourceID` | 必須。アプリケーション内の更新対象を表します。Documentに限定しません |
| `actionID` | 必須。宣言されたApplication所有のActionを示します |
| `expectedRevision` | 必須。対象リソースに対する楽観的競合制御の条件です |
| `payload` | 必須、null不可。形式と意味は`(applicationID, actionID)`が選ぶアプリケーション固有の定義に従います |

共通層はPayloadを業務データとして解釈しません。認証主体、テナント、権限、レビュー済みフラグを、Payloadに記載された値だけで信用しません。どのApplicationがどのActionを実行できるかはCanonicalの解決と実行時認可で検証します。資源の識別とRevisionの原子的照合・変更は、Application所有のStoreが実施します。

リソースID中の`/`は許容します。Authorization Resourceを構築する場合は、Application ID、Resource ID、Action IDをそれぞれ独立に符号化し、連結後の曖昧な区切りを生じさせません。

## 実行結果とエラー

HTTPを利用する場合は、同一オリジンへの`POST`、JSON本文、`Content-Type: application/json`、`X-TEI-Action: commit`を使用します。成功時はHTTP 200と以下を返します。

```json
{"protocolVersion":"1","revision":"v8"}
```

失敗時も`protocolVersion`を返し、機械判定可能な`error`コードを示します。

| HTTP状態 | `error` | 意味 |
| --- | --- | --- |
| 400 | `invalid_intent` / `invalid_payload` / `invalid_action` | 形式・アプリケーション固有入力・操作自体が不正 |
| 403 | `request_origin_denied` / `action_denied` | オリジン検査・実行時認可等で拒否 |
| 409 | `revision_conflict` | 確定前のRevisionから対象が変化した |
| 405 | `method_not_allowed` | POST以外を要求した |
| 415 | `unsupported_content_type` | JSON以外のメディア型 |
| 500 / 503 | `execution_failed` / `payload_decoder_unavailable` | 実行失敗・配備不足 |

`revision_conflict`を受け取ったブラウザーは、旧Revisionで自動再送しません。最新状態を取得し、利用者の意図を必要に応じて確認します。`request_origin_denied`などで拒否された場合も、画面上の見かけの変更を成功とみなしません。

現行のGo参照実装では、入力が64KiBの上限を超えた場合も`400 invalid_intent`を返します。将来のHTTP仕様との整合については、`413 Content Too Large`への変更を含めて検討します。

## アプリケーション非依存性の判定

非依存と判断する最低条件は次のとおりです。

1. SUIと第二の独立したApplicationが、**同一のHTTP Handler、Host、要求スキーマ、応答スキーマ**を使います。
2. 両者のPayloadの型は異なり、共通層はどちらのCard、SKU等も知りません。`PayloadDecoder`はApplication IDとAction IDの組合せで選択します。
3. Application、対象リソース、Revisionの境界を越える操作は拒否します。
4. UI・サーバー・保存側それぞれの責務が分かれます。ブラウザーやCLI以外のターゲットでも、共通Intentを再利用できます。
5. 新たなApplicationを加えるために、TEI Coreへそのアプリケーション専用の列挙値やデータモデルを増やす必要がありません。

`TestStandardActionProtocolSupportsIndependentApplications`はSUIとInventoryを用いる合成テストです。**第二の実製品への導入を示すものではありません。** 実環境での複数アプリ検証が完了するまで、標準機能への昇格判断を保留します。

## 非互換変更の扱い

従来の未確定ドラフトでは共通要求に`documentID`を用いていました。v1では`resourceID`へ統一し、`protocolVersion`を必須化しました。**旧形式を暗黙に受け入れません。** 利用側は同時に移行し、バージョン不一致を400として検出します。

Versionは文字列のメジャー番号です。今後の変更では、既存の`"1"`のフィールド名や意味を変更せず、破壊的変更に対して別バージョンの契約を定義します。既存のサーバーが未知のバージョンを黙って解釈しないことを優先します。

## 本番運用への未解決事項

この参照実装の`Host.CommandGate`は設定任意です。実配備では宣言済みAction解決と実行時認可を必須とする構成検証が必要です。実セッション認証・信頼できるプロキシ経由の同一オリジン判定・CSRF対策・テナント分離・永続的なCAS・重複操作の冪等性・操作監査・スキーマ検証・タイムアウトは未完成です。`X-TEI-Action`ヘッダーと同一オリジン照合だけで本番の安全性を保証しません。

Go参照Storeの`documentID`は、SUI固有の文書保存アダプター内部の識別子として残します。汎用Hostと通信境界には持ち込みません。

検証するときは正確なCommitを固定し、`go test -race`、`tools/repository-health-check.sh`、ブラウザーとSUIの現物テストを実行してください。テストコードの追加とPASS証拠は区別します。ブラウザー側の参照ファイルと要求JSON SchemaのGit Blob SHAを照合したローカルNodeテスト15件はPASSしましたが、応答JSON Schemaの追加後のテストはまだ未実行です。実TEIリポジトリ全体の健全性検査はまだ実施できていません。
