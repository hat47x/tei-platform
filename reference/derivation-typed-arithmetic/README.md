# D8: 数値・単位・通貨・丸めの型付き導出Reference（#1705）

**状態: 合成PoCの実行Reference。Canonical仕様、受理済Operator、SQL互換性、本番通貨処理ではない。**

この実装は既存 `internal/semantic/derivationref.Multiply` の厳密十進数乗算を再利用し、既存 `semantic.Type` と `semantic.Attribute.TypeRef` のalias参照を解決する。別の演算子Registryや汎用式評価器を導入しない。

## 固定境界

- `MultiplyAmount`: 数量側 `std.integer/std.decimal`、単価側 `std.money`、結果側 `std.money` を、既存 `semantic.Type` のprimitive/aliasで検査する。
- 対象の意味に必須な `Unit`・`PerUnit`・`Currency` はこのFixtureで明示入力する。unitの名寄せ・換算、任意のCurrency Registry、`TypeRef`からの暗黙推測はしない。たとえば「piece × JPY/piece = JPY」だけを許し、「kg × JPY/piece」「JPYをUSDへ暗黙変換」は拒否する。
- `derivationref.Multiply` で厳密な中間値を出し、`RoundExact` が明示された `DecimalSpec{Precision,Scale,Mode}` で**最終結果だけを**丸める。HALF_UPとHALF_EVEN、正負のtie、carry、精度超過を区別し、浮動小数点化と暗黙truncateはしない。
- NULLはD1の意味に従って伝播させ、未取得・参照なし・多重度違反をNULL/0へ変換しない。
- `RequireSameCurrency` は加算や比較を受け入れる前の候補検査であり、加算Operator/通貨換算Providerではない。

## 焦点試験

```sh
go test -count=1 ./reference/derivation-typed-arithmetic
go test -race -count=5 ./reference/derivation-typed-arithmetic
go vet ./reference/derivation-typed-arithmetic
```

正常例: 2×300=600.00、0×300=0.00、1.25×0.4=0.50、HALF_UP 1.005→1.01、HALF_EVEN 1.005→1.00。負例: kgと個数の不一致、JPY/USD不一致、型不一致・unknown/循環alias、NULLと未取得の混同、DECIMAL(5,2)への1999.98、負のtie、丸め後carryによるprecision超過。

## 未実装・依存

- D1 #1637の定義／計算の既存参照と#988のOperator契約を複製しない。実Operator登録、受理済みSemantic/Plugin Lock、物理SQL/CastやWasm計算と丸め適合性の証明は**未実装**。本Referenceの値だけでは本番財務保証とならない。
- `Unit`/`Currency` はこの合成Fixtureの文字列注釈であり、Source authoringやCanonical Typeへの新規標準属性としては**未採用**。単位／通貨変換が必要な案件は適切なPluginと明示された変換規則で実現する。
- D2 #1640のSQL計画、D3 #1641の確定値・鮮度、D4 #1692のBinding解決、D5 #1693の認可、D6 #1694の実装適合、D7 #1696のEvidenceは各所有者へ委譲する。
- X-TEA #940のsource/runtime相違はD8の計算仕様へ取り込まない。旧実行系の同等性は未検証。

実行の有無、exact SHA、成功／失敗／未検証をIssue #1705とPRへ記録する。
