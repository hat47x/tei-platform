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

- `keygen`は秘密鍵（Ed25519のseed、base64）をファイルへ書き、信頼ファイルのエントリを標準出力へ出します。`sign`は`-out`へ署名済みGrantを書きます。どちらも既存ファイルを上書きしません（`O_EXCL`）。したがって`-out`に秘密鍵・信頼ファイル・以前のGrantを指定しても壊しません。
- 秘密鍵のパスがGit作業ツリーの内側なら`keygen`も`sign`も拒否します（`PRIVATE_KEY_IN_REPOSITORY`）。文字列上の判定に加え、ファイルを開いたハンドルからOSが報告する実体パスで判定するため、Windowsのジャンクションやシンボリックリンク経由でも作業ツリー内へは書きません（`keygen`は秘密を書く前に確認し、内側なら作成したファイルを削除します）。鍵は、リポジトリ、Store、信頼ファイルの外へ置きます。ファイルモードは0600を要求しますが、強制されるのはOSが対応する範囲です。
- `sign`の`-not-before`と`-not-after`は必須で、期間は24時間以内です。署名対象は、`provisional: true`、issuer、keyId、Application、Capability、Subject、Resource、Epochs、期間を含む、ドメイン分離した正準JSONです。
- `show`は現在時刻で検証し、署名は表示せず、要約だけを出します。

## 検証の規則

`internal/security/provisionaleffectgrant`が実装します。

- 署名・信頼・鍵の失効と期間は、`internal/sourceacceptance/hoststore`の`TrustPolicy.VerifyDetached`を再利用します。信頼ファイルのスキーマは別（`tei.host-effect-grant-provisional-trust/v0`）なので、Source Acceptance用の鍵は効果許可に使えず、逆も使えません。同じ公開鍵の重複、未知のメンバーも拒否します。
- 拒否するもの: 署名の改ざん、期限切れ、未来の開始、失効、別issuer、別keyId、Application・Capability・Subject・Resourceの不一致、既知フィールド外、期間が24時間を超えるもの、`provisional`が真でないもの。
- `FileSource`は、読み取りのたびにGrantと信頼ファイルを読み直して検証します。再起動なしで、失効と期限が反映されます。
- エラーは固定の符号だけで、署名対象のバイト列、署名、ファイルの内容を含みません。

## 限界と運用上の注意

- 信頼ファイルは**issuer・keyId・Application単位**で鍵を信頼します。Capability・Resource・Subjectは制限しません。鍵の保持者は、その鍵が信頼されたApplication内の任意のCapability・Subject・Resourceについて、24時間以内のGrantを作れます。範囲を絞れるのは、鍵を分けて発行する運用だけです。
- 効果許可と、Source Acceptance（`reference-host-acceptance`）で**同じ鍵を使い回さない**でください。信頼ファイルのスキーマと署名のドメインは分けてあり、取り違えは検証で拒否されますが、鍵の共用は運用で避けます。
- `reference-critical-path-evidence-audit`の`provisional-grant`検査は、パケットの作成者が`physical.hostAuthorityProvenance`に書く**自己申告のラベル**に依ります。ラベルを書かないパケットは検出できません。ツールの結果は従来どおり承認ではなく、レビューで実際のEvidenceを確認してください。
- 期間の下限は1分、上限は24時間です。`notBefore`の未来側の上限は設けていません。上限24時間の期間が全体を制限し、未来開始のGrantはその時刻まで使えず、署名者が開始時刻を選べることで新たに得る能力がないためです（署名者は元々24時間以内の任意のGrantを作れます）。必要になったら、検証側の時計を基準に上限を足します。
- テキスト項目は、256バイト以下、有効なUTF-8、制御文字・書式文字（bidi上書き、ゼロ幅など）・ASCIIの半角空白以外の空白を含まないものに限ります。JSONのキーは、重複と、綴り（大文字小文字）の違いを拒否します。

## 見直す条件

次のいずれかが成り立ったら、この暫定issuerを撤回するか、本番の権限発行へ置き換えます。

- 実際のAuthority（利用者・所属・認可規則の正本）が、版（AuthorityEpochs）つきの許可を供給できるようになった。
- 所有者がD8を撤回した、または効果許可の本番方式を決めた。
- 暫定許可の出力を、G2の判定へ使いたくなった（それは本番方式の導入が前提です）。
