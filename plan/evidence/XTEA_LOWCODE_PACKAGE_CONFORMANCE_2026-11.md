# パッケージ契約（適合試験・投影）の検査の確認（2026-11）

## 対象

- 実装: `internal/validation/package_conformance.go`、`internal/validation/validation.go`の呼出し
- 試験: `internal/validation/package_conformance_test.go`
- 対象版: `main` の `c25664465` から分岐した`feat/package-conformance-check-20261110`
- 環境: Windows、Go `1.27.0`

## 見つけた不足

RQ-C1は「契約・版・**適合試験**を持つパッケージ」を求め、「二つの独立したパッケージを、**同じ適合試験の枠組みで比較**します」と定めています。

`application.Package`には`Conformance`と`Projection`の欄があり、例の`list-to-record.tei`も`projection screen`・`conformance list_to_record`を宣言しています。しかし**`internal/validation`も`internal/application`も、この2つを一度も読みませんでした**（`.Conformance`・`.Projection`の参照が0件）。

つまり「宣言はあるが、何も検査されない」状態で、D-9（定義にあるが効かない）と同じ形です。パッケージは任意の適合試験名を名乗れ、任意の投影を名乗れました。

## 変更

`validatePackageConformance`を追加し、`Validate`から呼びます。

| 診断 | 条件 |
| --- | --- |
| `APPLICATION_PACKAGE_CONFORMANCE_REQUIRED` | 適合試験名が空 |
| `APPLICATION_PACKAGE_CONFORMANCE_INVALID` | 名前に空白（契約の識別子として書かれていない） |
| `APPLICATION_PACKAGE_PROJECTION_REQUIRED` | 投影名が空 |
| `APPLICATION_PACKAGE_CONFORMANCE_MISMATCH` | 同じApplication内の2パッケージが**同じPattern種別**を主張するのに、適合試験名が違う |

最後の1つが要求の「同じ枠組みで比較」に対応します。種別が違えば比べる対象が無いので報告しません。

## 観測

- `TestPackageConformanceRequiresBothFields` 成功。両方空なら2件、片方だけ空ならその1件、両方あれば0件。
- `TestPackageConformanceRefusesWhitespace` 成功。`search v1`は`INVALID`。
- `TestPackageConformanceComparesSharedKinds` 成功。同じ`browse`を主張して`screen-v1`と`print-v1`に分かれると`MISMATCH`。同じ名前にすれば通る。
- `TestPackageConformanceLeavesAnUnrelatedKindAlone` 成功。`browse`と`record`で別々の名前を持つ2パッケージは比較対象が無く、報告されない。
- `internal/validation`・`internal/syntax`・`internal/application`・`internal/applicationview` は`ok`。
- 例の`validate`は`ok`（既存の`projection screen`・`conformance list_to_record`があるため）。

## 読み取れること

- パッケージの契約欄が**検査される**。名前の無いパッケージ、識別子でない名前、投影の無いパッケージは受理されない。
- 同じ種別を扱う2パッケージが別々の適合試験を名乗る場合、**同じ枠組みで比較できないことが報告される**。

## 読み取れないこと

- **適合試験そのものを実行すること**。今は「名前が宣言され、矛盾しないこと」を見るだけで、試験の中身は別の提供元の仕事です。
- 適合試験の内容と版の対応（名前が版を表しているか）。
- 投影名の語彙（`screen`・`print`等に閉じているか）。
- 実Provider・実DB・旧Runtimeでの観測。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。パッケージ契約欄の検査は参照実装で確認済み、適合試験の実行は未実装です。
