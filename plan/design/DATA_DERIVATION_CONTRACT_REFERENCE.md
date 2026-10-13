# データ導出規則の参照実装と検証範囲（#1637）

意味契約の正本は[データ導出規則の最小意味契約](DATA_DERIVATION_REFERENCE_CONTRACT.md)に置く。本書は参照実装の案内であり、別の意味契約を定義しない。

`internal/semantic/derivationref`は、規則定義と適用先を分離し、純粋な十進数乗算を合成入力で評価する。値、正当なNULL、入力未取得、参照先なし、多重度違反を区別し、不正な入力、依存循環、副作用要求を拒否する。

```bash
go test -count=1 ./internal/semantic/derivationref
go vet ./internal/semantic/derivationref
```

この試験は、Canonicalの公開契約への採用、DSL構文、SQL Providerでの実行、外部業務Runtimeとの同等性を証明しない。問い合わせ適用は#1640、鮮度・確定値は#1641、更新・副作用は#1129へ引き継ぐ。既存の演算子と安定した意味参照への接続、精度・丸め・通貨単位の正式な契約は引き続き未実証である。
