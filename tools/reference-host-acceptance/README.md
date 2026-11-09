# reference-host-acceptance

暫定のHost Acceptance Storeへ、issuerが署名したDecisionを鍵生成・署名・取込み・失効・表示する参照ツールです。効果許可は与えません。

## 暫定の受理印（D9）

所有者の仮承認D9（[`TEI_PROVISIONAL_DELEGATED_DECISIONS_2026-10-23.md`](../../plan/workstreams/TEI_PROVISIONAL_DELEGATED_DECISIONS_2026-10-23.md)）に基づき、実物の受理手続が決まるまで、このツールが署名するDecisionはすべて`provisional`の印を持つAcceptanceです。

- `sign`は`-provisional`を必須とします。省略すると何も書き出さずに失敗します。`formal`を発行する手段はありません。
- 印（`acceptanceKind`）は署名対象に含まれます。印のあるDecisionは署名スキーマ`tei.host-acceptance-decision/v1`、印のない従来のDecisionは`v0`のバイト列のままです。印を消す・足す・`formal`へ書き換えると署名検証に失敗します。
- 従来のDecision（印なし）は読めますが、読み取り結果は`unspecified`で、正式扱いにしません。`formal`を名乗るDecisionは、取込み・読み取りのどちらでも拒否します。
- `ingest`と`show`は`ACCEPTANCE_KIND=provisional`（従来のDecisionは`unspecified`）を出力します。
- `reference-first-web-target`は、provisionalまたはunspecifiedの受理で起動すると標準エラーへ警告を出し、全応答に`X-TEI-Source-Acceptance-Kind`ヘッダを付けます。
- `reference-critical-path-evidence-audit`は、パケットの`acceptance.acceptanceKind`が`provisional`または`unspecified`のとき`PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE`を出し、`productGate`は`UNQUALIFIED`のままです。
- `provisional`は、G合格・本番配備の根拠にしません。G1/G2は`UNQUALIFIED`のままです。

### 見直す条件

組織の受理手続・署名者が決まったとき。そのとき`formal`の発行経路、署名者の認証、鍵の扱いを別途設計します。

## コマンド

```
keygen  -private-key FILE -issuer ID -key-id ID -application APP [-not-before T] [-not-after T] [-revoked]
sign    -provisional -workspace DIR -application APP -decision-ref REF -authority-revision REV -issuer ID -key-id ID -private-key FILE -out FILE
ingest  -store DIR -trust FILE -decision FILE
revoke  -store DIR -trust FILE -application APP -decision-ref REF
show    -store DIR -trust FILE -application APP
```

秘密鍵はリポジトリ・Storeのディレクトリ・信頼ファイルに置きません。
