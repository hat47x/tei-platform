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

共通層はPayloadを業務データとして解釈しません。JSONの`null`は空白を挟んだ場合も共通境界で拒否します。HTTPの要求エンベロープでは、上記6項目以外のキー、キーの重複、大文字小文字を変更した別名を拒否します。`encoding/json`の構造体デコードが既定で許容する大文字小文字を無視した照合や重複キーの後勝ち処理に依存しません。**Payload内部**の構文・重複キーの解釈と業務的な妥当性は、選択されたApplicationのデコーダーの責務です。認証主体、テナント、権限、レビュー済みフラグを、Payloadに記載された値だけで信用しません。どのApplicationがどのActionを実行できるかはCanonicalの解決と実行時認可で検証します。資源の識別とRevisionの原子的照合・変更は、Application所有のStoreが実施します。

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

ブラウザー側Dispatcherは受信した結果もv1の応答契約に照らして検証します。成功HTTP 200で`revision`と`error`が共存する、未知のフィールドが含まれる、成功コードが200以外、失敗応答に`error`が欠落する、または不正なエラー識別子が含まれる場合は、`invalid_response`として扱います。サーバーの返答が不整合な状態でUIだけを成功扱いしません。

現行のGo参照実装では、入力が64KiBの上限を超えた場合も`400 invalid_intent`を返します。将来のHTTP仕様との整合については、`413 Content Too Large`への変更を含めて検討します。

## 結果不明時の共通取扱い

ブラウザーの確定アダプター（`confirmed-action-commit.mjs`）は、サーバーが**書込みを行わず拒否したと判定できる応答**と、書込み後に応答が失われた可能性がある状態を分けます。

- 同一オリジンの参照用HTTP Handlerが返す、操作前の形式拒否（400／405／415）、オリジン・権限拒否（403）、版競合（409）、デコーダー欠如（503）は、応答コードとHTTP状態が規約どおり一致し、`ActionCommitError`として検証された場合に限り、確定しなかった拒否としてそのまま返します。
- 通信断、タイムアウト、サーバー500系の実行障害、解読できない応答、第三者のアダプターが投げた未検証のエラー、成功時に有効なRevisionがない場合は、`ActionConfirmationError.code = "commit_outcome_unknown"`とします。これは**成功でも失敗でもなく不明**です。
- `commit_outcome_unknown`を受けた側は自動再送もUndo履歴更新も行いません。Application自身の認証済み読取口から状態を観測し、必要なら利用者が判断します。現在値が期待値と一致していても「この操作が成功した」と断定しません。
- 成功応答でRevisionが確定した後、再取得や画面反映が失敗した場合は従来どおり確定済みRevisionをエラーに保持します。この場合も既に確定した操作を再送しません。

この分類が「書込み前の拒否」を保証できるのは、**参照HTTP Handler自身の実装順序とエラー契約**に限ります。任意のプロキシ・外部Provider・将来の実装が同じエラーコードを返しただけでは、同じ意味を保証できません。実際の保存先で権限失効を副作用確定時に原子的に検査する必要性は、[副作用の確定境界](../../reference/effect-fence/README.md)を参照します。この参照実装を利用するだけで、その検査が実装されたことにはなりません。

## アプリケーション所有の保存先へのHTTPS委譲

`actionboundary.HTTPSApplicationStore`は、`Host.Commit`で認可された共通Action v1要求を、独立したアプリケーションのHTTPS受信口へ送る**任意の参照アダプター**です。SUIに固有のCard・島・Document型を使わず、Application IDとAction IDをBindingで固定します。宛先は配備時に固定するHTTPSのオリジンとリソースの経路から構築します。ブラウザーのHost・Origin・Cookie・任意の利用者ヘッダーを転送しません。

外部アプリケーションへ利用者として要求するには、次の条件が必要です。

1. `Subjects`が実行中の認証済み主体を信頼できる文脈から取得する。
2. `Tenants`が認証済み実行文脈からテナントを取得する。`Delegations`が、**同じ主体・テナント・配備設定の受信者（Audience）**に紐づいたBearer資格情報を信頼できる認証基盤から取得する。どれかが不一致なら送信しない。RFC 6750形式に合わない資格情報、8KiB超の資格情報も拒否する。
3. 受信するアプリケーション自身が署名・発行者・Audience・期限・主体リンク・テナント・現在の権限を検証し、同じ保存トランザクションで対象Revisionを照合する。
4. TEIの`Host.Authorizer`も実セッションを根拠として認可する。認証・認可に異なる主体解決機構を設定してよいという意味ではない。

**この参照実装は2の実資格情報発行・検証機構を提供しません。** `TrustedDelegation`の値は外部実装が提供するものであり、その文字列だけで真正性を立証できません。認証済み本人の代わりにHostのサービスアカウントを使うことも禁止します。正しい`Delegations`を用意できない構成はBindingを登録せず、SUIの受付フラグも既定の無効を維持します。

HTTPS接続ではリダイレクトを追わず、標準のAction結果形式だけを受け付けます。認可拒否とRevision競合は、保存先が書込み前の拒否を保証する場合だけ確定的な拒否として扱えます。通信断、HTTP 500、形式不正な応答などは`ErrRemoteOutcomeUnknown`とし、自動再送しません。SUIの永続保存・監査・元の人間主体の認可を、TEI側の合成Storeで代替しません。

この段階でのGo試験はTLSテストサーバーと模擬認証提供元を利用します。**実SUI Backendへのログイン済み要求や権限失効を含む試験ではありません。** 権限失効と書込みの原子的照合は、[副作用の確定境界](../../reference/effect-fence/INTEGRATION.md)の残課題です。

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

検証するときは正確なCommitを固定し、`go test -race`、`tools/repository-health-check.sh`、ブラウザーとSUIの現物テストを実行してください。テストコードの追加とPASS証拠は区別します。Git Blob SHAでGitHub側のファイルと一致することを確認したローカルコピーで、Node.js 22のテスト16件を実行し、全件PASSしました。要求・応答のJSON SchemaはDraft 2020-12として検証しました。実TEIリポジトリ全体の健全性検査はまだ実施できていません。
