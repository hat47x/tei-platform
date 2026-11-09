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
       [-allow-capability CAP]... [-allow-resource PATTERN]...
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

- 信頼ファイルは既定で**issuer・keyId・Application単位**で鍵を信頼します。範囲指定（下の「信頼範囲」）が無い鍵は、信頼されたApplication内の任意のCapability・Resourceについて24時間以内のGrantを作れます。**鍵ごとに範囲指定を付けることを推奨します。** 範囲指定が無い既存の信頼ファイルは従来どおり動きます（後方互換）。Subjectは範囲指定の対象外で、制限できません。

## 信頼範囲（任意）

信頼ファイルの各鍵エントリに、次の2つの任意メンバーを置けます。指定した鍵は、Grantの Capability / Resource が範囲外なら`PROVISIONAL_GRANT_OUT_OF_TRUST_SCOPE`で拒否されます。署名と鍵の状態（失効・期間）を先に判定するため、偽造は従来どおり署名エラーになります。

```json
{
  "schema": "tei.host-effect-grant-provisional-trust/v0",
  "issuers": [{
    "issuer": "dev", "keyId": "k1", "publicKey": "<base64 Ed25519 public key>",
    "applications": ["order_entry"],
    "capabilities": ["order_entry.submit_order"],
    "resourcePatterns": ["order/*", "invoice/7"]
  }]
}
```

- `capabilities`: 許可するCapability名の配列。完全一致で、大文字小文字を区別します。`*`は使えません。
- `resourcePatterns`: 許可するResourceの配列。**完全一致**か、末尾が`/*`の**前方一致**だけです。正規表現は使えません。
  - `order/*`は`order/`に1文字以上が続くResourceに一致します。`order/1`、`order/1/lines`は許可し、`order/`、`order`、`orders/1`、`x/order/1`は許可しません。
  - 単独の`*`（全許可）は拒否します。全許可は、メンバーを書かないことで表します。`order*`（`/`の前に付かない`*`）、`a*b`、`**`、`*/x`のように`*`が末尾の`/*`以外にあるものも拒否します。
  - パターンに`.`や`..`のパス区間を含めることは拒否します。Grant側のResourceが`.`・`..`区間を含む場合、ワイルドカードには一致しません（完全一致のパターンだけが一致します）。
- 照合はResource文字列そのままで行い、正規化・デコード・大文字小文字の畳み込みはしません。Resourceを外部入力（URL等）から直接組み立てるHostは、Grantを求める前に正規化する責務をHost側で負います。
- 一方だけを指定した場合、もう一方は制限されません。両方を指定した場合は両方を満たす必要があります。
- 厳格な読み込み: 未知のメンバー・綴り（大文字小文字）違い・重複メンバー、空配列、`null`、空文字、制御文字・書式文字、256バイト超、配列内の重複、64件超は、信頼ファイルごと拒否します（`PROVISIONAL_GRANT_TRUST_SCOPE_INVALID`など）。これらのメンバーはSource Acceptance用の信頼ファイルでは未知のメンバーとして従来どおり拒否されます。
- 範囲指定を持つ信頼ファイルは、このメンバーを知らない古いバイナリでは読み込みに失敗します（安全側に倒れます）。

`keygen`は`-allow-capability CAP`と`-allow-resource PATTERN`（どちらも繰り返し可）で範囲指定つきのエントリを出力します。`show`は信頼範囲（無指定なら`UNRESTRICTED`）も表示します。複数鍵の信頼ファイルは、`keygen`の出力を手でまとめます。
- 効果許可と、Source Acceptance（`reference-host-acceptance`）で**同じ鍵を使い回さない**でください。信頼ファイルのスキーマと署名のドメインは分けてあり、取り違えは検証で拒否されますが、鍵の共用は運用で避けます。
- `reference-critical-path-evidence-audit`の`provisional-grant`検査は、パケットの作成者が`physical.hostAuthorityProvenance`に書く**自己申告のラベル**に依ります。ラベルを書かないパケットは検出できません。ツールの結果は従来どおり承認ではなく、レビューで実際のEvidenceを確認してください。
- 期間の下限は1分、上限は24時間です。`notBefore`の未来側の上限は設けていません。上限24時間の期間が全体を制限し、未来開始のGrantはその時刻まで使えず、署名者が開始時刻を選べることで新たに得る能力がないためです（署名者は元々24時間以内の任意のGrantを作れます）。必要になったら、検証側の時計を基準に上限を足します。
- テキスト項目は、256バイト以下、有効なUTF-8、制御文字・書式文字（bidi上書き、ゼロ幅など）・ASCIIの半角空白以外の空白を含まないものに限ります。JSONのキーは、重複と、綴り（大文字小文字）の違いを拒否します。

## 見直す条件

次のいずれかが成り立ったら、この暫定issuerを撤回するか、本番の権限発行へ置き換えます。

- 実際のAuthority（利用者・所属・認可規則の正本）が、版（AuthorityEpochs）つきの許可を供給できるようになった。
- 所有者がD8を撤回した、または効果許可の本番方式を決めた。
- 暫定許可の出力を、G2の判定へ使いたくなった（それは本番方式の導入が前提です）。
