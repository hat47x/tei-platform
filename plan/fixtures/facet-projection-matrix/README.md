# facet-projection-matrix fixture（#4350・#2497・#4133）

`cases.json`は4構造×8表現型のうち11セル、`expected.json`は**手書きの独立Expected**です。参照ツールの出力をExpectedへ写していません。

- 根拠：#4350のXMAP-01〜10／RSP-01〜06表、#2497（写像・損失・逆写像）、#4133（表現型トポロジー・時間軸）。
- すべて合成descriptorです。実在のSchema・画面・業務データを含みません。
- `cells`は`(structure, representation)`の添字で組を指定します。同じ組を2回書くと拒否されます。
- 触れていない組は`matrix.uncovered`に出ます。未カバーは「不可能」でも「可能」でもありません。
- `options`で#4351の検査器の既知Mutantを有効にできます。Expectedを見てからMutantへ合わせないこと。
