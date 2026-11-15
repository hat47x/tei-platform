# DSL構文候補の比較票（S0・#4361）

- **状態**：設計比較。正式構文の選定ではなく、専門家の可読性やビルド速度の実測も行っていない。
- **親**：[DSL構文拡充 #4359](https://github.com/hat47x/tei-platform/issues/4359)
- **独立Expected**：同じディレクトリの `rule_authoring_expected_20261111.json`
- **対象**：`reference/workspace/semantic/sales/validation.tei` の正数・関係内一意性の2規則
- **関連**：[Frontend／Backendを分離した横断計画 Draft #4599](https://github.com/hat47x/tei-platform/pull/4599)、[Role・Operator境界 #988](https://github.com/hat47x/tei-platform/issues/988)

## 競合するのは3つのFrontend方式

| 方式 | 表層の候補 | 実装・評価の現状 |
| --- | --- | --- |
| A1 Go式を利用 | 明示的な式位置でGo互換の表現を読む | 構文ノイズ、型と位置の変換、意味の保存を比較する候補 |
| A2 独自構文 | ドメインの対象→条件→判定に沿う低ノイズな記法 | 既存`validation`および候補`rule`の差を比較する |
| A3 ハイブリッド | 宣言はA2、必要な式位置に限り外部式Parserを利用 | **候補の骨格のみ**。汎用式Parserとの統合を実装済みとはしない |

**A4「DSL→Goコード生成」はFrontendの第四候補ではない。** A1〜A3のいずれかを解釈し、既存の型付き意味へLoweringした後で選べるBackend／Targetの一つである。A2/A3とGoコード生成は併用できる。TEIのGo単一バイナリ配布方針と、日常DSLにGoの`package`／`func`／`struct`等を露出させない方針は両立する。

## 比較軸と、まだ証明できないこと

| 観点 | Frontendごとに確認すること | 現時点の上限 |
| --- | --- | --- |
| 可読性と認知順 | 2規則のSourceを並べ、対象、Scope、述語、結果を読み取る順序と記述上の重複を比較 | 根拠付きの設計判断のみ。AIだけで専門家評価済みとはしない |
| 型・意味 | 同じ明示Rule ID、Attribute、Relation、型・NULL/未取得Policyを既存`semantic.Validation`へ変換できるか | SF-A/Bの限定変換。SF-C/Dは未実装 |
| Source位置・診断 | 元SourceのUTF-8 byte span、CRLF、重複ID、曖昧参照、stale編集の拒否 | 候補`rule`の単一parse/anchorはPR #4405で局所実装。旧`semantic_anchor.go`の再Lexは別途#4364 |
| 演算・情報要求 | 0と負数、空集合と取得不能、NULLと部分取得、実Read集合・短絡が変わらないか | Stub上の値判定のみ。実Data Context・Providerの完全性は#2586/#2506 |
| 金額精度・単位 | 受理済みMeaningの数量・単位・丸め位相を各Backendが維持するか | Goに組込decimal型が無いことは、生成Targetが精度を保てないことの証拠にはならない。Target別検証が必要 |
| AIによる編集 | 一因子修正の差分、対象外byte不変、旧Revision・曖昧Occurrenceの拒否 | 比較計画のみ。生成AIの変更品質は未測定 |
| ビルド速度 | 同一WorkloadでParse／Lower／Buildの実時間と再計算数を比較 | **未測定**。A1/A2/A3やGo生成の優劣を採点しない |
| Go内部親和性 | Go AST／既存Lexer／Source逆写像の実装コストを比較 | 内部評価項目。専門家の可読性に優先しない |

## Fixtureと独立Expectedの扱い

`SF-01〜04`は旧`validation`と新`rule`候補の同じ明示ID・意味候補を比較する。`SF-05` Markdown表はProfile/Reader未実装。`SF-06` Operator呼出し案は**Rule IDを宣言しておらず、同一Validation IDは未成立**。Operator名からIDやPinを暗黙に作らない。

`EX-01〜03`の正数判定と`EX-06〜08`の一意性は数学的な意味を基準に置く。`EX-04/05/09`のNULL・未取得は選択されたPolicy、`EX-10〜16`の診断名は既存実装との互換性をそれぞれ検証する。**新Parserの出力から独立**であることと、**既存Evaluatorの正しさから独立**であることを混同しない。

`EX-08`の空集合は完全に観測したStubの空Scopeに限定する。実Providerから0行返った事実だけで、全件一意性が成立したと判定しない。また、各グループの登録数とGo Testが実際に消費するCase数を別に報告する。

## 次の受入と境界

最初は#4361のExpected→#4362/#4363のParseと型付き意味→#4364のSource位置→#988/#2586の実Readerという順で評価し、Desktopセルフビルド#4583/#2831とは並行させる。生成GoはSourceの意味正本ではなく、実Builder・Artifact・Consumerを別途検証する。

**採否**：A3は引き続き有力な検討候補だが、正式採用・全Reader・実Provider・編集往復・実Targetの同値性をこの票から認定しない。人間被験者・仮想専門家による採点、未実装表層の合格判定は行わない。
