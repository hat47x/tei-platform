# reference-generator-attestation（#2016）

生成器候補の選抜receiptからcanonicalなstatementを作り、`ssh-keygen`のdetached signatureで署名・検証する参照です。署名はstatementのbyte列に束縛され、候補のsource/binary digest・契約・Workspace revision・fixture・receipt digestを含みます。

ただし、署名者のAuthority、trust root、候補の正しさ、実Provider・実Targetでの動作は証明しません。`grantsAuthority`と`verifiesTrustRoot`は常にfalseです。

```text
go run ./tools/reference-generator-attestation/cmd/tei-generator-attest \
  statement -receipt /tmp/selection.json -candidate template -out /tmp/statement.json

go run ./tools/reference-generator-attestation/cmd/tei-generator-attest \
  sign -statement /tmp/statement.json -key /tmp/key -out /tmp/statement.sig

go run ./tools/reference-generator-attestation/cmd/tei-generator-attest \
  verify -statement /tmp/statement.json -signature /tmp/statement.sig \
  -allowed-signers /tmp/allowed_signers -identity signer
```

## statement

```json
{
  "schema": "tei.reference.generator-candidate-statement/v0",
  "candidateId": "template",
  "language": "go",
  "origin": "template",
  "sourceDigest": "sha256:...",
  "binaryDigest": "sha256:...",
  "contractId": "sales.line-amount",
  "contractVersion": "1",
  "contractDigest": "sha256:...",
  "workspaceRevision": "rev-...",
  "fixtureDigest": "sha256:...",
  "suiteVersion": "sha256:...",
  "buildInputDigest": "sha256:...",
  "receiptDigest": "sha256:..."
}
```

- `receiptDigest`は選抜receipt全体のdigestで、statementを一つのreceiptへ束縛します。
- `contractDigest`は生成器provenanceまたはgeneration provenanceから取ります。
- statementはstructのfield順で整形し、末尾に改行を付けます。同じreceipt・候補からは同じbyte列になります。
- 署名対象はstatement fileのbyte列そのものです。JSONを再整形すると署名は一致しません。

## 署名と検証

- `sign`は`ssh-keygen -Y sign -f <key> -n tei-generator <statement>`を使います。
- 署名fileの既定は`<statement>.sig`です。
- `verify`は`ssh-keygen -Y verify -f <allowed_signers> -I <identity> -n tei-generator -s <signature>`へstatementをstdinで渡します。
- `allowed_signers`と`identity`は呼出し側が指定するtrust入力です。この参照は、そのtrust入力が妥当かどうかを判断しません。
- passphrase付きの鍵は対話入力が必要なため、この参照では扱いません。

## 検証すること・しないこと

| 検証する | 検証しない |
| --- | --- |
| statementが署名後に改変されていないこと | 署名者が実在するか |
| 指定したallowed signersのidentityと鍵で署名が成立すること | 署名者・roleのAuthority |
| statementとreceiptのdigest、候補のsource/binary digest | 候補の正しさ・実Provider/実Targetの動作 |
| 署名のnamespace | trust root・鍵の失効・承認workflow |

署名が成立しても、意味の受理・権限付与・本番配備の許可にはならない。Authorityは#951 / #2016 / ADR-0035の境界へ戻す。

## 鍵の扱い

- 参照試験では一時directoryにpassphraseなしのed25519鍵を生成し、終了時に捨てる。
- 実運用の鍵はこの参照へ含めない。秘密鍵をrepository・Issue・PR・ログへ置かない。
- 署名の失効・ローテーション・複数署名・承認者選択は未実装。

## 検証

`go test ./tools/reference-generator-attestation/... -count=1`で、次を確認します。

- 同じ選抜receipt・候補から同じstatementが得られる。
- 未知の候補を拒否する。
- `ssh-keygen`で署名し、allowed signersのidentityで検証できる。
- statement改変後の署名を拒否する。
- allowed signersにないidentityを拒否する。
- 署名receiptの`grantsAuthority`と`verifiesTrustRoot`がfalse。

`ssh-keygen`が無い環境では署名試験をskipします。

## 未実装・限界

- 署名者のAuthority・trust root・鍵の失効・ローテーション。
- 複数署名・閾値署名・承認workflow。
- buildの完全な再現性、artifact store、実行ファイルの束縛。
- 実Provider・実Target・本番配備の受理。
- Windows/Unix間のssh-keygen実装差の完全な吸収。
