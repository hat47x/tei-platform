# OpenSSH authoring witness Reference pressure

Status: Research fixture for ISSUE-000974. Productionのauthoring authority providerではありません。

## 目的

Shared Acceptanceのhuman Decision witnessを、自己申告のactor文字列で代用せず、実在する署名方式で検証します。

このfixtureはOpenSSHの`ssh-keygen -Y sign / verify`を使います。OpenSSHをTEIの必須依存にするためのものではありません。閉域環境でも使える既存技術を一つ通し、Shared Acceptance側に必要なprovider境界を実物から抽出するためのpressure testです。

## このfixtureが確認すること

`pressure-test.sh`は一時的なEd25519 keyを作り、`fixture/witness-v0.txt`へ署名します。正しいprincipal、namespace、policyで検証できることを確認した後、次の条件で検証が失敗することを確かめます。

- signer principalが異なる
- accept用署名をreject namespaceへ流用する
- 署名後にcandidate fingerprintを変更する
- allowed signerの有効期限が切れている
- allowed signerがまだ有効ではない
- signing keyがrevocation listに含まれる

OpenSSHが見つからない環境では終了コード77で停止します。その場合はPASSを主張しません。

## Reference namespace

fixtureでは次のnamespaceを使います。

```text
tei-authoring-v1.ws-ref-001.accept.scope-aaaaaaaa@tei-platform
```

workspace、decision semantics、scope identityをnamespaceへ含める理由は、署名鍵がTEI authoring全体の権限を持つように扱わないためです。

この文字列形式はproduction schemaではありません。OpenSSHの実物でpattern、長さ、運用負荷を確認した後に、ISSUE-000974で採否を判断します。

## Witness message

`witness-v0.txt`は、次を固定順で並べたReference用messageです。

- schema version
- workspace identity
- candidate fingerprint
- review fingerprint
- base model revision
- accept / reject
- decision reason digest
- scope digest
- witness時刻
- nonce

この形式もproduction schemaではありません。現段階では、署名がcandidate、Review Projection、base revision、decision semantics、scopeへexactに結び付くかをpressureするためだけに使います。

## Authority policyとの境界

allowed signersはfixtureが一時ディレクトリへ生成します。candidateやwitness messageからallowed signersを書き換える経路はありません。

ただし、実運用で同じCLI利用者が任意のallowed signers fileを選べるだけではauthoring authorityの根拠になりません。production採用へ進む前に、policy sourceをdeploymentまたはorganizationの管理境界から与える必要があります。

## 実行

```bash
tools/reference-authoring-witness-openssh/pressure-test.sh
```

current OpenSSHの`verify-time` optionを使い、fixtureの検証時刻を固定します。

このChatGPT実行環境には`ssh-keygen`が無かったため、ここで実行PASSは出していません。OpenSSHがあるtrusted local cloneでpressure testを実行し、exact commitと結果をISSUE-000974へ記録するまで、このReferenceを採用済みとは扱いません。

## 非主張

このfixtureがPASSしても、次はまだ保証しません。

- signerが組織上の承認者である
- allowed signersの配布元が安全である
- StudioやWeb UIでuser presenceを確認できる
- Source mutationまでのShared Acceptance全体が完了した
- Runtime PermissionやResource authorityを得た

OpenSSH providerの検証成功はhuman Decision witnessのEvidence候補であり、Canonical Sourceを書き換えるDecisionそのものではありません。
