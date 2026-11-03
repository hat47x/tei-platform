# reference-host-effect-grant

開発・検証専用の**暫定効果許可（provisional effect Grant）**を、issuerが署名して発行する参照ツールです。

## これは何でないか

- **本番の権限発行ではありません。** 所有者の仮承認D8（[暫定の委任判断](../../plan/workstreams/TEI_PROVISIONAL_DELEGATED_DECISIONS_2026-10-23.md)、追記は#2791）に基づく、開発・検証のための参照実装です。
- 署名が示すのは「秘密鍵の保持者が、この（Application、Capability、Subject、Resource、Epochs、期間）を署名した」ことだけです。許可を与えた人間が誰か、実際の認可Authorityがその時点で許可していたかは示しません。
- この許可から生じたEvidenceは`provisional-grant`のProvenanceを持ちます。**G2へ数えません**（G1/G2はUNQUALIFIEDのままです）。`tools/reference-critical-path-evidence-audit`は、パケットの`physical.hostAuthorityProvenance`が`provisional-grant`のとき、主張の種類によらず`PROVISIONAL_GRANT_NOT_G2_EVIDENCE`を出して`INCOMPLETE`にします。
- 既定のCLI経路・本番のHostは、この許可を読みません。このパッケージ自体は、Reader、Runner、Web Targetへ配線していません。

## 使い方

```text
keygen -private-key FILE -issuer ID -key-id ID -application APP [-not-before T] [-not-after T] [-revoked]
sign   -private-key FILE -issuer ID -key-id ID -application APP -capability CAP -subject ID -resource RES
       -epoch-source S -subject-link-epoch N -membership-epoch N -policy-epoch N
       -not-before T -not-after T -out FILE
show   -grant FILE -trust FILE
```

`T`はRFC3339のUTCで、末尾が`Z`です。

- `keygen`は秘密鍵（Ed25519のseed、base64）をファイルへ書き、信頼ファイルのエントリを標準出力へ出します。既存ファイルは上書きしません。
- 秘密鍵のパスがGit作業ツリーの内側なら`keygen`も`sign`も拒否します（`PRIVATE_KEY_IN_REPOSITORY`）。鍵は、リポジトリ、Store、信頼ファイルの外へ置きます。ファイルモードは0600を要求しますが、強制されるのはOSが対応する範囲です。
- `sign`の`-not-before`と`-not-after`は必須で、期間は24時間以内です。署名対象は、`provisional: true`、issuer、keyId、Application、Capability、Subject、Resource、Epochs、期間を含む、ドメイン分離した正準JSONです。
- `show`は現在時刻で検証し、署名は表示せず、要約だけを出します。

## 検証の規則

`internal/security/provisionaleffectgrant`が実装します。

- 署名・信頼・鍵の失効と期間は、`internal/sourceacceptance/hoststore`の`TrustPolicy.VerifyDetached`を再利用します。信頼ファイルのスキーマは別（`tei.host-effect-grant-provisional-trust/v0`）なので、Source Acceptance用の鍵は効果許可に使えず、逆も使えません。同じ公開鍵の重複、未知のメンバーも拒否します。
- 拒否するもの: 署名の改ざん、期限切れ、未来の開始、失効、別issuer、別keyId、Application・Capability・Subject・Resourceの不一致、既知フィールド外、期間が24時間を超えるもの、`provisional`が真でないもの。
- `FileSource`は、読み取りのたびにGrantと信頼ファイルを読み直して検証します。再起動なしで、失効と期限が反映されます。
- エラーは固定の符号だけで、署名対象のバイト列、署名、ファイルの内容を含みません。

## 見直す条件

次のいずれかが成り立ったら、この暫定issuerを撤回するか、本番の権限発行へ置き換えます。

- 実際のAuthority（利用者・所属・認可規則の正本）が、版（AuthorityEpochs）つきの許可を供給できるようになった。
- 所有者がD8を撤回した、または効果許可の本番方式を決めた。
- 暫定許可の出力を、G2の判定へ使いたくなった（それは本番方式の導入が前提です）。
