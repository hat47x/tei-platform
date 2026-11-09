# opensshwitness

`ISSUE-000974`（OpenSSH signed authoring witness provider）のprovider固有部分を、
`tools/reference-authoring-witness-openssh/pressure-test.sh`で実証済みの手順から
型付きGo libraryへ抽出したReference実装です。

## 提供するもの

- `Payload` / `Message()` / `Namespace()`: 署名対象メッセージとprovider namespaceを
  payloadから決定論的に導出します。署名者・呼び出し側がnamespace文字列を選んで
  authority scopeを選ぶことはできません。
- `Sign`: payloadのcanonical messageを実際の`ssh-keygen -Y sign`で署名します
  （Reference / test用。運用ではoperator自身のSSH agentに署名させます）。
- `Verify`: `ssh-keygen -Y verify`を`-O verify-time`固定・allowed-signers policy・
  任意KRLで実行し、`Evidence`（`gateStatus`・診断分類・provider出力原文・
  message/signature digest・evidence fingerprint）を返します。

## fail closed条件

- provider binaryが無い → `ErrProviderUnavailable`。自己申告actorへのfallbackはしません。
- `-O verify-time`非対応のOpenSSH（例: 8.9p1）→ `ErrVerifyTimeUnsupported`。
  新鮮性uncheckedの検証は実行しません。
- principal不一致・namespace不一致・payload改変・policy満了/未開始・KRL失効は
  `gateStatus=authority-unresolved`のEvidenceを返し、決してqualifiedにしません。

## 主張しないこと

- このpackageはShared Acceptance契約ではなく、production identity providerでも
  ありません。`authority-qualified`なEvidenceは、受理・source書込みを自動的に
  許可しません。candidate・review projection・base revisionの再検証は
  `tools/reference-shared-acceptance`側の責務です。
- `witnessNonce`は省略可能ですが、省略と存在でメッセージが変わるため、
  replay中断の判定はpayload側（発行時刻・nonce運用）と組み合わせて行います。
  署名replay・policy source差替え等のnegative pressureは#974の将来作業です。

## 検証

```bash
go test ./tools/internal/opensshwitness -count=1
```

実OpenSSHが無い環境、および`-O verify-time`非対応のprovider環境では、
provider向けtestはpassせずskipします（`TEI_OPENSSH_WITNESS_SKIP=1`で
明示的に全skipできます）。
