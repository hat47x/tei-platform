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

## v0の対象シナリオと判断境界

- `claim` は `E0`、`E1`、`E2`（計画文書§16参照）。これは提出者の**申告区分**であって実証済み能力ではない。
- 全区分で exact candidate `head` / reviewed `main`、full Repository Health PASS、実行前後clean、検証receiptの**申告**を要求する。この条件はあくまで提出に必要な情報で、受領側がcheckoutとreceiptを検証する必要がある。
- `E1` は `R0 != R1`、Candidate/Review fingerprint、Decision、R1再読込、`add-validation` と `add-application-validation-use` の同時受理、R1の `order_entry` が `sales.order_line.unit_price_positive` を選択した実Web正負試験、N11 controlを要求する。WebのProviderは合成でもよいが、**実Web操作receipt**が必要。
- `E2` はE1に加え `actual-db` Provider、R1に対するDatasource/schema互換性証跡、current Host Authority、Commandと独立readの**別receipt**、同一record、S02/S05の拒否試験を要求する。Web synthetic Providerを `E2` として提出すると拒否する。
- Source、Target、DB schema、Git revisionを値の一致だけで同一視しない。DB schemaとApplicationは別のライフサイクルであり、`schemaCompatibility=compatible` は**提出された判定結果の表明**に過ぎない。Providerの実状態・readの独立性・記録時点はreviewerが確認する。
- `E0` のフォーム検査が通っても、C0/Studio/Business/Loweringの本当のproducer-consumer接続や製品上の動作を証明しない。

## 記録の例（不足しているため必ずINCOMPLETE）

```json
{
  "schema": "tei.critical-path-evidence/v0",
  "claim": "E1",
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
    "providerKind": "synthetic", "positiveReceipt": "",
    "negativeReceipt": "", "unobservedControlReceipt": ""
  },
  "physical": {}
}
```

本Referenceは実際の証拠ファイルを自動取得・検証しないため、`receipt`にパスを記述しただけでは検証済みにはならない。提出物にはsecret、credentials、個人情報を含めない。

## このtoolが検証しないこと

GitHub remoteの本当のHEAD、信頼できるtest runner、実Source Acceptance、現行Providerの契約/権限、実Database、readの独立性/鮮度、Business ProcedureがAcceptedであるか、case自身の真偽、署名、証拠ファイルの内容は検証しない。特に **`productGate=UNQUALIFIED` を他の自動化処理でPASSとして使用してはならない**。

Mainへ統合する場合は、まずこのPR自身のexact headで `tools/repository-health-check.sh` をtrusted full checkoutにて全件実行し、clean pre/post と current main一致を別途取得する。