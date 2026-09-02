# Legacy Authentication Adapter Conformance

## 目的

このConformanceは、既存システム固有の認証状態や認証規則を失わずに、Semantic Capability `security.authenticate`へ接続できることを確認するものです。

ここで重要なのは、既存の認証schemaをTEI標準のUser schemaへ作り替えることではありません。既存recordと既存ruleをadapterの向こう側に保ったまま、Application / Driverからは共通のCapability契約として利用できる境界を作ります。

対象はIssue #68およびADR-0019です。業務上の結果とExecution Errorの分離には、Issue #77 / ADR-0020で確定した`capabilityinvoke`の共通境界をそのまま利用します。

## 既存user recordの扱い

v1のReference adapterは、既存システムにあり得る次のmetadataを保持できます。

- login ID
- authenticated subjectへ対応するidentity
- credentialの参照
- 無効化状態
- 有効期間
- login失敗回数
- lock期限
- 組織コード
- roleコード

この型はTEI標準User schemaではありません。既存schemaや既存APIから得た情報を、認証adapterへ渡すためのReference境界です。

credentialについても、legacy user recordには検証対象そのものを保存せず、credentialの参照だけを保持します。

## credential検証の境界

credential検証は`CredentialVerifier`へ分離します。

```text
runtime input
  login ID
  presented credential
        │
        ▼
Legacy Authentication Adapter
        │
        ├─ UserStore ──> legacy user metadata
        │
        └─ CredentialVerifier
             credential reference
             + presented credential
```

提示されたcredentialは実行時入力としてVerifierへ渡します。Canonical World、Source Index、Business Resultへコピーしません。

Reference testにも「正解となるsecret値」を保存せず、Verifierの成否を注入して境界だけを検証します。

## 正常な業務結果

次は、認証処理を正常に完了して得られる`denied` Outcomeです。

| 条件 | Business Result |
| --- | --- |
| 利用者が見つからない | `denied / credential_mismatch` |
| credential不一致 | `denied / credential_mismatch` |
| 無効化済み | `denied / account_disabled` |
| 有効期間前 | `denied / not_yet_valid` |
| 有効期間切れ | `denied / expired` |
| lock中 | `denied / account_locked` |
| 既存システム固有ruleが不成立 | `denied / <legacy reason>` |

利用者が見つからない場合とcredential不一致は、Referenceでは同じreasonへ畳みます。利用者の存在確認を結果から不用意に露出させないためです。

認証が成立した場合は`authenticated` Outcomeを返し、payloadにはsubject identityだけを置きます。

## Execution Error

次は業務上の認証拒否ではありません。

- legacy user storeへ到達できない
- credential verifierを実行できない
- 既存ruleの評価そのものを完了できない
- legacy recordにsubject identityがない
- legacy recordにcredential参照がない
- `security.authenticate`以外のCapabilityを誤ってこのadapterへ渡す
- adapterの入力型が契約と一致しない

これらを`denied`へ変換すると、利用者の認証が成立しなかったのか、認証判断そのものを実行できなかったのかが分からなくなります。

そのため、`capabilityinvoke.Invoke`へGo `error`として返し、Execution Errorの境界を維持します。

## 既存システム固有rule

field対応だけでは表現しにくい既存条件を、TEI共通ruleへ無理に変換しません。

`LegacyRule`は、既存user metadataと評価時刻を受け取り、次のどちらかを返します。

- 正常に評価でき、許可または拒否を判断できた
- 評価処理そのものを完了できなかった

前者の拒否は`denied`、後者はExecution Errorです。

Reference Conformanceでは、組織コード、roleコード、login失敗回数が既存metadataとしてruleへ届くことも確認します。

## この段階で固定しないこと

このConformanceだけでは、次を確定しません。

- TEI標準User schema
- `std.auth.*`の標準Capability
- production用password hash方式
- LDAP / IdP / OIDC等の製品選択
- credential rotation方式
- Authorization rule
- login画面
- 汎用的な認証error階層

これらをAuthentication adapter一例から先回りして一般化しません。

## 次の接続

次の縦切りでは、このadapter境界をReference Workspaceの明示的なCapability Bindingへ接続し、legacy mappingの来歴とsecret非混入をRepository上のEvidenceとして確認します。

その後、`security.authorize`を`order_entry`のbrowse / edit / submitへ接続し、Gate Aを再評価します。

関連: #68 / #69 / #77 / #124 / ADR-0019 / ADR-0020
