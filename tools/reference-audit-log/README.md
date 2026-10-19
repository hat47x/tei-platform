# reference-audit-log（#2020）

SaaS・SIer向けの提供形態で使う、追記専用の監査ログ基盤の最小Referenceです。ADR-0036の方針に従い、提供形態ごとのアプリケーション実装ではなく、アプリケーションが利用できるpluginとして提供します。

```text
go run ./tools/reference-audit-log/cmd/tei-audit-log append \
  -log /tmp/audit.jsonl -tenant tenant-a -actor user-1 -action read -subject order/1

go run ./tools/reference-audit-log/cmd/tei-audit-log verify -log /tmp/audit.jsonl
```

## エントリ

各エントリは、テナント、actor、action、subject、時刻、直前のhash、自身のhashを持ちます。

```json
{
  "seq": 1,
  "tenantId": "tenant-a",
  "actor": "user-1",
  "action": "read",
  "subject": "order/1",
  "at": "2026-10-19T12:00:00Z",
  "prevHash": "sha256:0000...",
  "hash": "sha256:...."
}
```

`hash`は、直前のhash、seq、テナント、actor、action、subject、時刻を順に連結したsha256です。

## 保証

- 追記専用のJSON Linesとして保存する。
- 既存エントリの変更は、hash chainの検証で検出する。
- 壊れたchainへは追記しない。
- 再オープン後も直前のhashからchainを継続する。

## 保証しないこと

- 末尾のエントリを削除された場合の検出。最新hashを外部へ固定しない限り、短いchainは検証を通る。
- 外部ストレージの不変性、WORM、法的な監査要件への適合。
- 内容の機密性。必要なら保存時の暗号化を別pluginで行う。
- 複数writerの競合制御。このReferenceは単一writerを前提とする。

## plugin記述

`plugin.json`は、能力`audit`、保証、保証しないことを宣言します。Delivery Profileの`verify`は、`audit`を選んだ構成に対して次のconformanceを実行します。

- 2件の追記
- chainの検証
- 既存エントリの改ざん検出
- 改ざん後の追記拒否

## 未実装

- 最新hashの外部固定、透明性ログ、署名。
- 保存期間、削除、アーカイブ。
- 複数プロセス・複数writerでの順序保証。
- 暗号化、テナントごとの鍵。
