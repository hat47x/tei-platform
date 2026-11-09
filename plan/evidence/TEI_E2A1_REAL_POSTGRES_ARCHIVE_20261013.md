# #1119：E2a-1実PostgreSQL先行試験の原本保全（2026-10-12）

## 原本と実行版

[#1127 Draft PR #1399](https://github.com/hat47x/tei-platform/pull/1399)のexact head `682e1e9a9cea15054b77db7695f760a25eb0fcb7`から分離した**検証専用PR #1546（mainへ未mergeでclose）**で、GitHub仮統合版を実行した。

- [実PostgreSQL Actions #38311557882](https://github.com/hat47x/tei-platform/actions/runs/38311557882)：**success**。同版の[Health #38311557729](https://github.com/hat47x/tei-platform/actions/runs/38311557729)：**success**。
- GitHubが実際にcheckoutした仮統合commit：`a2679f0d8d52b0562543464cbd32f12c2ebd72c3`。親main `3044bf9ccaf677d42c6e7e03db9fca42a9b04900`＋検証branch `8029725e52508475a5fa442f4b3bf0e2721eae21`。
- [Artifact #11774837092](https://github.com/hat47x/tei-platform/actions/runs/38311557882/artifacts/11774837092)：ZIPのSHA-256は`34b6cbbca65edfa0f920f02bd5c07988af407514eac7df399c6898d92bfb94d5`。GitHubのDigestと**ダウンロードした実ZIPの独立SHA計算が一致**。Artifact期限：**2026-10-19 21:36 UTC**。
- ZIP内唯一の`receipt.json`は6,341バイト、raw SHA-256 `2d766bb442ee009a56a8d457d2f9568d74936f5df78af1cc44a363eb2d378361`、Git blob SHA-1 `f74d6d22ac1b6e3c5d9647ac35a11388c23e07b0`。原本を**バイト単位で無編集**で [`TEI_E2A1_REAL_POSTGRES_RECEIPT_38311557882.json`](TEI_E2A1_REAL_POSTGRES_RECEIPT_38311557882.json)へ保存。GitHubが保存したblob SHAと元ZIPからの値を一致確認済み。

これらのハッシュとGitHub runnerの当時の実行結果は**史料の来歴**。保存したJSONを後日読み込んでも、当時のDB動作をその場で再実行したことにはならない。

## 実際に観測したこと

PostgreSQL16の専用・空のfixture DBで、手書きoverlay・INSERT・Queryを使用。Go DriverがCommand Bindingを解決し、接続pool AからINSERT。別pool BのPostgreSQL backend（PID 133と134で異なる）で、同じCommand IDの1行を独立に読み、属性が送信値に一致した。観測結果は`observed`。

N08（Binding欠損）、S02類似（表なし・列なし）は事前拒否でdispatch 0。N11二件（read pool閉鎖、書込み後の表削除）ではCommand commit後の読取り失敗を`observed`と誤報しないことを確認。実行前後でDB上のユーザー表およびEvent Triggerは0件。

## 製品上はまだ証明されていないこと

**`scope.productE2=false`、`authority=none`、`generatedRoundTrip=false`を維持する。** `handWrittenOverlay=true`、SQLは手書き、Command→Persistenceの意味的対応は`unverified`、独立性は**同一プロセスの2つのpool**に限る。

これは`intake.request` / `request_log`の先行E2a-1であって、#1119の正式Accepted Source `order_entry` R1→通常Web Command→実DB書込み→**別invocationの独立read**ではない。認可・current Host Authority・S05、Source差分付きS02、別主体・再起動後の永続性、人間の承認、第二Targetを実証していない。#1406のPersistence宣言照合も同じ仮統合へ含めていない。**G2、製品E2、G1〜G4はUNQUALIFIED**。

## 保全検査

`tools/verify-e2a1-postgres-archive.py`は元raw SHAを固定し、現物の全バイト、元仮統合SHA、正負の整合、scopeの否定フラグを検査する。これは既存の受理・DB・Host Gateとは別の**証跡バイト検査**であって、製品資格判定器ではない。

```sh
python3 -B tools/verify-e2a1-postgres-archive.py
python3 -B tools/test-e2a1-postgres-archive.py
```

専用`.github/workflows/tei-e2a1-archive-ci.yml`は対象ファイル変更時にこの5件の局所試験とclean前後を実行する。テスト環境への実PostgreSQL再接続や、最新main・Host Authorityの再承認は行わない。
