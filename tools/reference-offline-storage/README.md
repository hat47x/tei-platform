# reference-offline-storage（#2020）

個人向けの提供形態で使う、ローカル保存の最小Referenceです。ADR-0036の方針に従い、個人向けのアプリケーション実装ではなく、アプリケーションが利用できるpluginとして提供します。

```text
go run ./tools/reference-offline-storage/cmd/tei-offline-storage \
  put -root /tmp/local-store -key notes/a.txt -value hello

go run ./tools/reference-offline-storage/cmd/tei-offline-storage \
  get -root /tmp/local-store -key notes/a.txt
```

## 保証

- storage rootの外を読んだり書いたりしない。
- 絶対key、`..`、root外へのシンボリックリンクを拒否する。
- 同じkeyへの上書きと、再オープン後のreadをサポートする。
- `single-user`を前提に、ネットワークサービスを使わない。

## 保証しないこと

- 複数端末間の同期、競合解決。
- 保存時の暗号化、バックアップ。
- 複数writerの同時更新。
- 電源断時の完全な原子性。書き込みは`O_TRUNC`で行い、クラッシュ時に部分的な内容が残る可能性がある。

## plugin記述

`plugin.json`は、能力`offline-storage`・`single-user`、保証、保証しないことを宣言します。Delivery Profileの`verify`は、`offline-storage`を選んだ構成に対して次を確認します。

- 自rootへのwriteとread
- 再オープン後のread
- `..`によるpath traversalの拒否

`single-user`には独立したconformance testがないため、`unverified`として残します。

## 未実装

- 暗号化、バックアップ、同期。
- 複数writer、ロック、トランザクション。
- 容量制限、削除の安全な伝播。
