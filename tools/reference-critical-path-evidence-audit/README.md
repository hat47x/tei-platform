# Critical Path Evidence Packet preflight (#1119 / Reference-only)

**目的:** #1119 の G1→G2 / 最初の受注明細 Validation 変更を、一つの `R0→Preview→Decision→R1→Web→Observation` 証跡として提出する際に、受け渡しの欠落や合成証拠の昇格を静的に検出する。**既存のGate判定器、Shared Acceptance、Driver、Schema検査、Host認可を代替しない。**

これは `tei.critical-path-evidence/v0` の実験用、案件／シナリオ特化の前提検査である。正本の汎用schemaでも自動デプロイ条件でもない。ソースを変更せず、`-packet` で渡されたJSON以外の証跡を読み込まない。

## 使用

```sh
# trusted checkoutの正確なheadと、レビュー対象のmain SHAを指定する。
# ローカルmainがcurrent remote mainと一致しているかは別途確認する。
head="$(git rev-parse HEAD)"
main="$(git rev-parse main)"
go run ./tools/reference-critical-path-evidence-audit \
  -packet /path/to/receipt.json -head "$head" -main "$main"

# 単体試験（full Repository Healthの代替ではない）
go test -count=1 ./tools/reference-critical-path-evidence-audit
```

出力の `status` は `INCOMPLETE` または `REVIEW_REQUIRED`。`productGate` は**常に `UNQUALIFIED`**。`REVIEW_REQUIRED`、exit code 0、`FullHealth: PASS`という入力文字列のいずれも、Gate PASS / mainへの統合許可ではない。`REVIEW_REQUIRED` は**形式上の必須欄が揃っただけ**で、証跡の実在、真正性、検査時点、Authorityを独立して調べる必要がある。

## P0の現在地を再現する例

`testdata/p0-order-unit-price-incomplete.json` は、受注明細単価の追加Validationについて**既存のGoテストソースまでは確認したが、actual running Web・Source R0/R1・trusted checkoutのEvidenceが未取得**である現状を、そのまま機械可読化した負例である。見本データの`head`/`main`や実行receiptを、存在しないのに埋めてはならない。

```sh
go run ./tools/reference-critical-path-evidence-audit \
  -packet tools/reference-critical-path-evidence-audit/testdata/p0-order-unit-price-incomplete.json \
  -head "$(git rev-parse HEAD)" -main "$(git rev-parse main)"
```

正しい動作は`INCOMPLETE`・`productGate=UNQUALIFIED`・非ゼロ終了である。欠けている証拠には`R0→R1`、実Webの新規単価負例、Repository Health、独立Observationが含まれる。テスト`TestCurrentP0FixtureCannotBeSubmittedAsEvidence`が、「既存テストがある」という事実だけでE1を認定してしまう回帰を防止する。

## v0の対象シナリオと判断境界

- `claim` は `E0`、`E1`、`E2`（計画文書§16参照）。これは提出者の**申告区分**であって実証済み能力ではない。
- 全区分で exact candidate `head` / reviewed `main`、full Repository Health PASS、実行前後clean、検証receiptの**申告**を要求する。この条件はあくまで提出に必要な情報で、受領側がcheckoutとreceiptを検証する必要がある。
- `E0` でもProducer/Consumerの実装箇所、正負のテスト名、証拠の性質（`source-only`、`synthetic-tested`、`actual-observed`）が必要。単なるCase名では不足とする。**E1/E2で `evidenceClass=source-only` または `synthetic-tested` を指定すると、他の欄がそろっていても `E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED` で拒否する。** WebのProviderがsyntheticでも、E1では *Webサーバの実行* が必要という意味である。
- `E1` は `R0 != R1`、Candidate/Review fingerprint、Decision、R1再読込、`add-validation` と `add-application-validation-use` の同時受理、R1の `order_entry` が `sales.order_line.unit_price_positive` を選択した実Web正負試験、N11 controlを要求する。WebのProviderは合成でもよいが、`web.runMode=running-server` と**実Web操作receipt**が必要。さらに `web.negativeValidation` は**新しい単価規則そのもの**、`web.negativeEvidenceMode` は `running-server` であることを申告させる。従来の数量0による正常な拒否や、Goの `httptest` による単価0の負例だけでは、実行中Webサーバが新しい単価規則を拒否した証拠にならない。`handler-only` / `httptest` はE0の有用な証跡だが、このE1の実サーバ操作要件を満たさない。
- `E2` はE1に加え `actual-db` Provider、R1に対するDatasource/schema互換性証跡、current Host Authority、Commandと独立readの**別receipt**、同一record、S02/S05の拒否試験を要求する。Web synthetic Providerを `E2` として提出すると拒否する。
- Source、Target、DB schema、Git revisionを値の一致だけで同一視しない。DB schemaとApplicationは別のライフサイクルであり、`schemaCompatibility=compatible` は**提出された判定結果の表明**に過ぎない。Providerの実状態・readの独立性・記録時点はreviewerが確認する。
- `E0` のフォーム検査が通っても、C0/Studio/Business/Loweringの本当のproducer-consumer接続や製品上の動作を証明しない。

## 記録の例（不足しているため必ずINCOMPLETE）

```json
{
  "schema": "tei.critical-path-evidence/v0",
  "claim": "E1",
  "contract": {
    "producerPath": "", "consumerPath": "", "positiveTest": "",
    "negativeTest": "", "evidenceClass": "source-only"
  },
  "qualification": {
    "head": "", "main": "", "fullHealth": "NOT_RUN",
    "cleanBefore": false, "cleanAfter": false, "receipt": ""
  },
  "acceptance": {
    "r0": "", "r1": "", "candidateFingerprint": "",
    "reviewFingerprint": "", "decisionReceipt": "", "reloadReceipt": "",
    "delta": ["add-validation", "add-application-validation-use"]
  },
  "web": {
    "acceptedRevision": "", "application": "order_entry", "target": "web",
    "selectedValidation": "sales.order_line.unit_price_positive",
    "negativeValidation": "sales.order_line.unit_price_positive",
    "negativeEvidenceMode": "handler-only",
    "providerKind": "synthetic", "runMode": "handler-only", "positiveReceipt": "",
    "negativeReceipt": "", "unobservedControlReceipt": ""
  },
  "physical": {}
}
```

本Referenceは実際の証拠ファイルを自動取得・検証しないため、`receipt`にパスを記述しただけでは検証済みにはならない。提出物にはsecret、credentials、個人情報を含めない。

**当面のE1残Gap**: 現在の通常Web用synthetic fixtureは単価が常に正であり、受注1002の拒否理由は数量0。既存 `practical_business_rule_test.go` は `staticReadScenarioRunner` で単価0をテストできるが、これはhandler-onlyである。実行中Web Targetで新規規則の負例を観測するには、#1127がテスト専用の隔離したfixture/Target起動構成を用意する等、**受理済み業務意味・標準API・実Providerを汚さない方法**を追加検討する。このpreflight自身が `running-server` の申告や入力receiptの真偽を確認するわけではない。

## 入力の取り扱いとpreflightの追加負例

- JSONは最大1 MiB、UTF-8の厳密な入力のみを受ける。ファイル先頭1 MiBが妥当でも、その後に隠れたJSON・テキストがあれば受け付けない。
- 未知field、トップレベルと入れ子の重複key、`claim`/`Claim`のような大文字・小文字だけ異なるalias、64階層を超える深い入れ子は拒否する。Goの通常のJSON decoderが最後の同名fieldを採用する挙動で権限・証跡を上書きさせない。
- `run` のCLI全体に対しても `INCOMPLETE`時の非ゼロ終了と機械可読JSON、`REVIEW_REQUIRED`時でも常に `productGate=UNQUALIFIED` を確認する。
- focused testは `go test -race -count=1 ./tools/reference-critical-path-evidence-audit` で検査できるが、必須のfull `tools/repository-health-check.sh` を置き換えない。

## このtoolが検証しないこと

GitHub remoteの本当のHEAD、信頼できるtest runner、実Source Acceptance、現行Providerの契約/権限、実Database、readの独立性/鮮度、Business ProcedureがAcceptedであるか、case自身の真偽、署名、証拠ファイルの内容は検証しない。特に **`productGate=UNQUALIFIED` を他の自動化処理でPASSとして使用してはならない**。

Mainへ統合する場合は、まずこのPR自身のexact headで `tools/repository-health-check.sh` をtrusted full checkoutにて全件実行し、clean pre/post と current main一致を別途取得する。