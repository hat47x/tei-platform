# 暫定チェーンの実行記録（D8〜D11を通した一改訂E2E）

- 状態: 実行記録。**G1・G2は合格ではない**（監査ツールのproductGateは`UNQUALIFIED`のまま）。
- 根拠: 所有者の仮承認D8〜D11（[暫定代行判断](../workstreams/TEI_PROVISIONAL_DELEGATED_DECISIONS_2026-10-23.md)、2026-11-03）。関係: [#1163](https://github.com/hat47x/tei-platform/issues/1163)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)。
- 対象: `origin/main` `4c3edc1f1`（#2838）+ 試験`tools/reference-first-web-target/provisional_chain_e2e_test.go`（`TestProvisionalChainOneRevisionE2EPacket`）。既存の一改訂E2E（`TestOneRevisionOrderEntryE2EPacket`）は変えていない。

## 実行環境

| 項目 | 内容 |
| --- | --- |
| OS / Go | Windows 11 Pro 10.0.26100 / `tools/gow.cmd`のGo |
| PostgreSQL | WSL2 Ubuntu-22.04のdockerで起動した使い捨ての`postgres:16-alpine`（実行後に破棄。127.0.0.1のみ公開） |
| DSN | 環境変数`TEI_TEST_POSTGRES_DSN`のみ。パケット・ログ・本記録にDSNとパスワードは出していない（試験が出力ファイルを走査して確認する） |
| 鍵 | Acceptance用とeffect Grant用の2本をその実行ごとに生成した使い捨て鍵（人ではない）。信頼ファイルは別スキーマ |

## 通した暫定部品

| 部品 | 経路 | 実行で確認したこと |
| --- | --- | --- |
| D8 暫定効果許可 | Web Targetの実フラグ`-provisional-effect-grant`・`-provisional-effect-grant-trust`・`-provisional-effect-grant-development-only` | 起動ログに`PROVISIONAL_EFFECT_GRANT_ACTIVE`。実行中に信頼ファイルで鍵を失効させると`HOST_EFFECT_GRANT_UNAVAILABLE`で422、行はdraftのまま、receiptなし、Runner未到達 |
| D9 provisionalなAcceptance | `acceptanceKind=provisional`で署名したDecisionをHost Storeへ取り込み、`-host-acceptance-store`で読む | Storeから読み戻した種別が`provisional`。起動ログに`SOURCE_ACCEPTANCE_PROVISIONAL` |
| D10 暫定postgres Provider | `newPostgresProvisionalRunnerOrderSubmitStore`（Runner型の注文提出、Evidence付き）。信頼されたownerルーチン経路 | 下表のEvidence |
| D11 Intent整合検査 | 同じstoreが`checkOrderSubmitIntent`を通す | OperationID `order-submit/1001`とpayloadの注文が一致して実行へ進んだ |

Webプロセスが出力したEvidence（正のrunの1件。パケットの`runner-evidence.json`の要点）:

| 項目 | 値 |
| --- | --- |
| Provenance | `provisional-grant` |
| Guarantee | `atomic-with-commit` |
| GuaranteeReason | `provisional D10: trusted-owner routine path; re-verify against real deployment` |
| Status / OperationID / ReceiptVersion | `committed` / `order-submit/1001` / 1（保存されたreceiptの行と一致） |
| Scope | `order_entry` / `order_entry.submit_order` / subject `subject-order-editor` / resource `order/1001` |

そのほか、別OSプロセスの独立読取り（`submitted`）、単価0の規則拒否（Runner未到達）、R0の起動拒否、N11（commit後にObservationが失敗。Evidenceは`committed`の1件）、S02の実行時ドリフト拒否（`PHYSICAL_SCHEMA_NOT_READY`、行・receiptとも変化なし）が既存E2Eと同じ形で通った。

## 監査ツールの結果（無改変で実行）

```
status       INCOMPLETE
productGate  UNQUALIFIED
diagnostics  PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE
             PROVISIONAL_GRANT_NOT_G2_EVIDENCE
             TRUSTED_FULL_HEALTH_NOT_RECORDED
```

期待した2つの診断が出た。`TRUSTED_FULL_HEALTH_NOT_RECORDED`は、この実行が`main`上の健全性CIの受領記録を持たないための、既存E2Eと同じ診断である。他に欠落の診断は出ていない。パケット本体（約20ファイル）はコミットしない。

## 判明した点

1. **Runner型storeには起動時のS02ゲートがない。** `requireOrderSubmitSchema`は直接書込みのstore（`*postgresOrderSubmitStore`）だけを検査し、他のstoreはそのまま返す。宣言のindexを削った表に対し、Runner型storeのWebプロセスはHTTPを公開した（`physical-s02-controls.json`の`webServedHTTP: true`）。書込み前の検査（実行時）は働く。直接storeの既存E2Eが持つ「起動時に拒否」の対照は、この経路では取れない。
2. **監査ツールはGuaranteeを見ない。** パケットにGuaranteeの欄がなく、`atomic-with-commit`の宣言（仮承認D10）は人が`runner-evidence`を読んで確認するしかない。ProvenanceとAcceptance種別も、パケット作成者の自己申告である（この試験は実行結果から読み取って書いているが、監査ツールは読み取り元を検証しない）。
3. 既存の`TestCombinedWebCLIRefusesUntrustedReaderEnvironment`は、この環境（Windows）で`first-web-target`実行ファイルが見つからず失敗する。本変更と無関係で、`.exe`なしの名前でビルドしていることによる環境依存と見られる。

## 位置づけと見直し条件

- この記録は、仮承認の部品がつながって動くことと、監査ツールが正しく不合格側に倒すことの確認である。D8〜D11の成果物はG1・G2の合格に数えない。
- 見直す条件は仮承認の記録のとおり。実Hostの権限発行元が決まったとき（D8）、組織の受理手続・署名者が決まったとき（D9）、実運用のDB構成で同じ試験を取り直したとき（D10）、`Scope.Resource`の細分設計が決まったとき（D11）。実物が得られたら同じ手順で取り直す。
- 再現: `TEI_TEST_POSTGRES_DSN`（空の使い捨てDB）と`TEI_E2E_PACKET_OUT`（リポジトリ外）を設定し、`go test ./tools/reference-first-web-target -run '^TestProvisionalChainOneRevisionE2EPacket$' -count=1 -v`。
