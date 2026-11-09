# Everything is Data：表層構文からプレーンな標準データへ収束する条件の限定Reference（2026-11-02）

- **状態**：横断設計・合成Fixtureによる限定実測。TEI本体の正式Normal Form / Syntax / ABI / Meaning採用ではない。
- **親**：[Everything is Data 統合 #2619](https://github.com/hat47x/tei-platform/issues/2619)
- **主Owner**：[意味Role・自己記述 #2620](https://github.com/hat47x/tei-platform/issues/2620)、[Syntax Plugin / Operator #988](https://github.com/hat47x/tei-platform/issues/988)、[Markdown Authoring #2621](https://github.com/hat47x/tei-platform/issues/2621)
- **Consumer/契約Owner**：#1124/#1824 Studio、#2497情報保存、#2507 Function Composition/A-G、#2622 Data Operations・Data Provider、#1121 Target Lowering、#1131/#1180 Scenario、#2514独立Oracle、#2516開示、#2502 Script/Macro、#943 Artifact/Lock。
- **Reference**：[`plan/fixtures/everything-data-syntax/verify_surface_normal_form.py`](../fixtures/everything-data-syntax/verify_surface_normal_form.py)
- **Goal**：Lisp的なEverything is Dataを、TEIの**型付き参照・意味Role・Plugin・Target**と両立させる。Lisp構文、GoaのGo評価、MukuroLのUI語彙、専用Host、第二のCanonical/Resolverの採用を前提としない。

## A. 今回固有の設計判断

前回のScript-first [Draft #2617](https://github.com/hat47x/tei-platform/pull/2617)は「**Pluginが解決したと仮定する**Source span/typed dependencyから、汎用inspect・Patch候補へ安全に投影する」局所Reference。今回の差分はその**さらに上流**：複数の表層表現から、どの情報を保持したときだけ同一プレーンデータ／意味候補と言えるかを、具体的な有限正負ケースで判別すること。

概念上の受渡しは次の通り。

```text
Authoritative Source (Markdown/indented DSL/chain/typed DOT/JSON/...)
    │  source bytes, span, file identity, syntax-plugin version, comments/layout
    ▼
Syntax parse / desugar (language-plugin-specific, preserving provenance)
    ▼
Standard data candidate (Record/Sequence/Ref/Literal/Role/Operator encoded as DATA)
    │  syntax-independent, resolved IDs and operator bindings where available
    ▼
Semantic Role application + typed validation + explicit Acceptance
    │  Accepted Meaning / source mapping / exact dependency versions
    ├── Function/Data Context/Scenario/Derived Dataset/SHO views
    ├── Studio generic inspect + source-aware reversible edit candidate
    └── Target-specific realization + independent Observation
```

**区別する三つの同一性**：

1. **Source identity**：同一Source bytes、ファイル、span、Revision/Source移動やコメント。MarkdownとMethod Chainは違うSourceなので同じSource byteになることは要求しない。
2. **Plain-data candidate identity**：同じ解決済みRole/Op/参照/順序/引数の構造データ。表層構文やDOTレイアウトの違いから独立させられる範囲だけ同一とする。データFingerprintをここで試せるが、**それはMeaning Acceptanceの証明ではない**。
3. **Accepted/observable meaning**：Operator law、scope/grain、NULL/presence、失敗、effect、clock、外部Target/Owner判断によって意味が決まる。候補データが同形でも、実行適合・業務受理・DB Effect・Clockの意味は別検証。

Markdown表・Method Chain・Graphviz図を完全に可逆と一律に仮定しない。Source faithfulかつ実際の意味操作可能性はProjection/編集用途ごとに判定（#2497）。人が読める記法を編集正本に選ぶ場合も、導出されたプレーンデータを**第二の編集正本**にはしない。

## B. プレーンデータだけで役割を持たせる

**データは文字列だけではない。** 必要なNode、属性、Operator ID、引数順、Subject ID、Reference、Literal、依存関係、Effect特性も標準Record/Sequence/Referenceの値として保持する。

架空の関数 `sales.confirmed_total` は、`orders`を`status=confirmed`で絞り、`amount`を集計する。以下はいずれも表現例（正式TEI構文ではない）。

**Markdown Table**：

```markdown
# function: sales.confirmed_total
| step | op | source | parameters |
| --- | --- | --- | --- |
| eligible | where | orders | status=confirmed |
| total | sum | eligible | amount |
```

**インデント表現（MukuroL風の発想、独自の仮記法）**：

```text
function sales.confirmed_total
  eligible: where orders status confirmed
  total: sum eligible amount
```

**Method Chain**：

```text
function sales.confirmed_total = orders.where(status==confirmed).sum(amount)
```

**プレーンデータの一表記（JSON例）**：

```json
{
  "role": "function",
  "id": "sales.confirmed_total",
  "result": {"ref": "total"},
  "steps": [
    {"id": "eligible", "op": "tei.where@v1", "args": [
      {"ref": "sales.orders@R1"}, {"path": "status"}, {"literal": "confirmed"}]},
    {"id": "total", "op": "tei.sum@v1", "args": [
      {"ref": "eligible"}, {"path": "amount"}]}
  ]
}
```

JSONと上記Role/Operator ID、`sales.orders@R1`は**Fixtureに固有の架空識別子**。公式標準名やSchemaを提案するものではない。

**Graphviz/DOT**：単に`orders -> filter -> total`を描いた図には、Operator・入力Scope・predicate・Result identityが欠ける。型付き意味属性と対象の安定IDを持つ限定DOTだけを同じ標準データ候補へ解決し、layout/rank/色は描画側の情報として分離する。

## C. 重要な合成則と誤同値

1. **同じ見た目 ≠ 同じ参照**：文字列 `orders` が一致しても、Bindingが `sales.orders@R1` なのか `external.orders@R1` なのかで異なる。Stable Subject/Operator/Role identityを比較する。
2. **同じデータ候補 ≠ 同じSource**：コメント、改行、原文の違いはSource fidelityとして保持。Presentation-only DiffとDomain Meaning Diffを区別する。
3. **同じOperator名 ≠ 同じOperator**：`tei.sum@v1`と`@v2`、意味law、NULL、grain、unit、precision、failure、effect/clockを別検証する（#2544/#2593/#988）。
4. **無順序Graph ≠ 実行順序を無視可能**：Method ChainやFunction Stepの順序・短絡・副作用を、型/ノード集合だけから安全に再配置できるとは限らない。
5. **図の線 ≠ Domain Connector**：有向辺は参照、dataflow、event、constraint、effectを自動決定しない。#988のConnector意味、#2507のA/G条件を使う。
6. **元データを推測した復元 ≠ 可逆変換**：Derived Dataset/平均・3階層→2階層の集約など、非単射Projectionの逆操作は明示Actionが必要（#2622/#2500）。
7. **Roleをデータ化 ≠ 自動実行**：`role: test`等はScenario/Oracleへ照合する入力。Role文字列に権限・効果・試験PASSを付与せず、明示Source Acceptanceと実行環境資格は従来のOwner。
8. **一意なNormalized候補 ≠ 安全なAI編集**：AIの修正対象へ書き戻すためにはSource span、安定ID、Revision、編集の逆写像を#1124/#2497で検証。抽出時に失われたコメント/表レイアウト/元位置を黙って再構築した扱いにしない。

## D. 自己記述とBootstrapをどう成立させるか

「すべてをデータで記述する」ことを**無限の自己解釈手順**にしない。最小限の概念層を比較する。

- **Boot-0（物理読み込み）**：最小のUnicode/byte、Record/Sequence/Value/Ref、安定識別と構文Plugin解決に必要な機構。これは実装可能なコードであり、TEIのあらゆる機能を自己記述し終えてから実行するわけではない。
- **Boot-1（Role/Operator定義データ）**：`Function`、`Model`、`Test`、`Rule`、`Projection`等のRole適用条件・型/演算契約・バージョンを標準データ値として記述する。多重Role・競合・循環を診断。
- **Boot-2（解釈・受理・実現）**：既存のLanguage/Operator PluginとSemantic Extension Lock/Acceptanceで解決する。Target/Host/Providerの実行能力は別の資格。Plugin自己実装を解釈データだけから自動保証しない。

Boot-0にどの程度の値型を置くか、`Ref`の解決をいつ固定するか、Role定義の自己改版をどう安全に切り替えるかは、**既存#988/#951/#1121/#2620の比較Ownerで判断**する。新たな独立元データレジストリ/共通Runtimeは前提としない。

## E. 実測：局所標準ライブラリ Fixture 23ケース

[Fixture](../fixtures/everything-data-syntax/verify_surface_normal_form.py) は**汎用Markdown/Goa/Graphviz/TEI Parserではない**。一つの架空関数・2ステップに限定した、表層ごとの厳密な記述と検証用の硬いMappingを使う。

**実行**：`cd plan/fixtures/everything-data-syntax && python3 -m unittest -v verify_surface_normal_form.py`。さらに`python3 -m py_compile verify_surface_normal_form.py`。

**観測済み：Python stdlib unittest 23/23 PASS、py_compile PASS**（隔離コンテナ）。実行したファイルの SHA-256 は `a6cede81cd5b9ab88d18ef36fb2943618f1a2cef439959045d375803baab2d87`、Git Blob SHA-1 は `ed121363e00eaf1d7808396cad1e429cc2337242`。GitHubに保存したBlob SHAと一致。実行前の試行ではDOTに過剰なテキスト一致しか検査していなかったため、ノードごとの属性・有向辺・Semantic IDを検査する負例を追加してから最終23/23を取得した。旧失敗は合格扱いしない。

| 試験 | 限定的に実測したこと |
| --- | --- |
| S-01〜04 | Markdown/Indent/Chain/JSONの4表層→同じデータ候補、Source digestは異なる |
| S-05〜09 | 説明表の誤実行、predicate欠落、未知Op、順序変更、違うBindingを拒否 |
| S-10〜12 | ラベルのみDOTは意味未成立、明示属性付き限定DOTとlayout-only変更を区別 |
| S-13〜18 | Indent誤り、JSON引数/Op版、重複ID、同名別Ref、追加Effect chainを拒否 |
| S-19〜23 | Source文字列とNormalized digest区別、DOTのNode属性誤配置・辺逆転・ID不一致・未知semantic属性を拒否 |

このテストは固定された期待Normal Formとの照合であり、**複数の本物のParserが同じ受理済み意味を生成できるという製品証明ではない**。Operatorの実計算/データセット/NULL/Target/開示/権限/DB Effect、実Studio/SHO、Owner意味受理、AIの構文学習率はNOT RUN。

## F. 次の実証（「AIフレンドリー」を測る）

- **N0 Syntax→Data**：#988の実Language Plugin/解釈から同じSource/Version/Span付き標準データを実取得し、Fixtureの**ハードコードしたMappingを撤去**する。参照元・Rule版・Operator版が一致することを確認。
- **N1 Role→Function/Model/Test**：#2620へ5種以上のRoleを広げ、同じ構造APIからtyped inspectできること、Role衝突・未知Plugin・自己解釈循環を診断。#2621の文書・表テストは独立Expectedと元位置を保持。
- **N2 Source→Studio→Source**：#1124/#2497が一意のSource mappingのみ編集候補へ戻し、Markdown説明表・図示専用DOT・macro展開後はread-only/navigationとして説明。
- **N3 Domain → Data Operations**：#2622のRelation Path/Function Data Contextに対して、標準演算の意味（NULL/重複/範囲/Measure/書戻し）とProvider差を検証する。Syntax/Roleの一致をProviderの同じ結果保証としない。
- **N4 AI-first比較**：Goa風/MukuroL風/Markdown/JSONで、AIへ固定Grammar+Plugin Schema+正負例を渡す場合と渡さない場合を比較。初回成功率は構文、型付き参照、意味、局所Patch、token・repair回数に分けて測る。学習済み構文との親和性を推定ではなく実測。
- **N5 Target/SHO**：同一受理モデルから設計書とTarget成果物を展開し、Source実消費物・Actual Result/Effect・独立Oracle・Consumer開示の各資格を明示。TEI専用Hostなしの.NET/Java/WASM等へ段階的に。

**並行化**：AI探索は4表層/異分野（業務、組織、ゲーム、科学、文書）を広く並行。統合時にはSource/Meaning/Role/Operator/Provider/Target exact版と個別OwnerのEvidenceを結び、局所合成のPASS件数をProduct合格数へ加算しない。

## G. 計測と保留項目

**構文候補の評価**：Grammarの複雑さと記述密度、AIのfirst-pass構文妥当・意味適合、1箇所変更のPatch縮小率、型・Source位置・Unknown診断、Plugin作者の追加コード量、Studio固有UIの増分、Designer/OwnerがレビューできるDiff、実Targetの生成・受入までの変更回数。

**未決**：正式の共通Normal Form/元Source間の所有規則、Role解釈の順序と循環・多重Role、式/参照/型の最小Bootstrap、Data OperationへのOperator/Scopeの接続、Lossy projectionのSource-aware編集の範囲、AI向け簡潔なDSLの文法選定。

これらは今後の実証で絞り込む。**Everything is Dataを標語で終わらせないために、表層差と意味差が分離できることを、まず小さな負例で示す**。
