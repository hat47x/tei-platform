# facet-structure fixture（#4352・#2495・#2544）

`cases.json`は構造族のdescriptor、`expected.json`は**手書きの独立Expected**です。参照ツールの出力をExpectedへ写していません。

- 根拠：#4350のDSF-01〜10表、#2495（Facet）、#2544（演算束縛・矛盾判定）。
- すべて合成descriptorです。実在のSchemaや業務データを含みません。
- `corpusRef`は#2501のPressure Caseを指すための欄ですが、**現在は未設定**です。値を入れる場合は#2501の実際のCase IDだけを使い、同じIDを2ケースへ割り当てないこと（ツールが拒否します）。Pressure Corpusの第二正本は作りません。
- MCT-01〜04は負例です。Core型の要求は`rejected`として返し、採用判断はADRへ戻します。

判定語彙・採用先・gapコードは [`tools/reference-facet-structure-check/README.md`](../../../tools/reference-facet-structure-check/README.md) を参照してください。
