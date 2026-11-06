# DSL Language Plugin / Operator Composition 設計候補

- **状態:** 評価待ち（2026-10-08）。採用済みArchitecture / ADRでも、実装済みReferenceでもない。
- **役割:** DSL自体のPlugin化と、業務意図を表現するOperatorを組み合わせる方式の機能設計候補。
- **追跡:** [GitHub #988](https://github.com/hat47x/tei-platform/issues/988)
- **上位正本:** [RQ-032 / RQ-033](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、[Architecture](../../docs/architecture.md)。本書はそれらを置き換えない。
- **関連:** ISSUE-000077 Semantic Extension、ISSUE-000079 Plugin self-build、ISSUE-000080 extensible Capability。
- **対象外:** 既存 `.tei` Parserの即時置換、Production-ready汎用式エンジン、native SQLからの強制移行、任意のPluginコードに意味変更権限を与えること。

## 1. 目的と避けたい失敗

TEIは、業務上の意味を実装手段から独立させ、短い記述から検証可能なApplicationを組み立てることを目指す。DSLを読みやすくするだけでは十分ではない。新しい業務概念を追加するたびにGoのParser、Validation enum、Driver、Web Targetを同時に修正するなら、利用者の記述だけが短くなっても拡張可能な基盤にはならない。

ここでは **DSL自身の構文・解析・検証・意味への変換をLanguage Pluginが担い、Operatorの意味をSemantic Extensionとして独立させる**構造を検討する。高抽象度Operatorは小さなOperatorの合成を第一候補とし、どうしても合成できない場合は既存Capability契約による実装を許す。

避けたい失敗は次のとおり。

1. `where`等の単語だけPlugin化し、実際の構文と意味の変更をHost Parserへ固定する。
2. Pluginの読込み順や優先順位で同じSourceの解釈が変化する。
3. NULL、未取得値、重複、関係多重度、精度、時点などがProviderごとに変わる。
4. SQL、Go、既存Validation等のauthoritative sourceを黙って第二正本へ複写する。
5. 自動的に構築したExecution Planが、宣言していない書込みや外部副作用を獲得する。
6. DSLの文法拡張、Semantic Extension、Runtime Plugin、Build Toolの依存・権限を混ぜる。
7. 可読性のためだけに高抽象度Operatorを乱造し、意味を説明できなくなる。

## 2. 参考にする考え方

ユニケージ開発手法のTukubaiでは、`self`（項目選択）、`msort`（整列）、`sm2`（キー単位の集計）、`join1`（一致したトランザクションへのマスター情報の結合）など、小さく役割の明確なコマンドを組み合わせる。特に`sm2`の記法は、現場で交わされる「この列をキーにして、この列の合計を求める」という作業指示を直接表現しようとする思想に基づく。

参考: [USP研究所によるTukubai製品紹介](https://www.usp-lab.com/DOWNLOAD/PDF/PRODUCT_uspTukubai.pdf)、[sm2の説明](https://www.uni-tama.com/cgi/COMMAND_MAN2.CGI?COMMAND=sm2)、[作業指示とコマンド記法](https://www.uni-tama.com/cgi/DOCUMENT.CGI?CATEGORY=COMMAND&POMPA=COMMAND_JISAKU_02)。

ただしTukubaiのコマンド名や列番号をTEIにそのまま移すのではない。Tukubaiではソートやキー一意性を利用者が満たすコマンドがある。TEIでは、名前付きAttribute、Relation、型、Cardinality、Constraintを参照し、前提を検査できることを重視する。Tukubaiの物理的なパイプとTEIの意味上の合成も同一視しない。

## 3. 既存設計との接続

現行TEIは `.tei` Authoring Source、Canonical Semantic Model、Binding / Realization、Runtimeを分離し、Source Indexからauthoritative sourceへ戻れる。SQL-authored Queryではnative `.sql`が正本であり、Query IRはderivedである。今回もこの境界を保持する。

既存 `validation sales.order_line.quantity_positive positive` と `validation sales.order.lines.line_number_unique relation-unique` は、Gate Aで実際に評価した意味である。新しいOperator記法はこれらの置換完了を主張せず、まず対応可能な表現と対応不能な表現を比較する。

Semantic Extensionのidentity / dependency / Accepted Semantic Set LockにはRQ-032とISSUE-000077の既存方式を利用する。Runtime Plugin Lock、BuildArtifactLock、Semantic Set Lockを一つの汎用Lockへ統合しない。Language PluginのParser実装をGo / Wasm等の異なるTargetで提供する場合も、**何を意味するか**と**どの実装が解釈するか**を別々に追跡する。

## 4. 提案する三つの境界

### 4.1 Language Plugin（DSLそのもの）

Language Pluginは次の責務を提供する。

| 契約 | 内容 |
| --- | --- |
| Identity | Language ID、contract version、Semantic Extension identity、Source / fingerprint |
| Parse | 指定されたLanguage Sourceをsyntax treeとsource spanへ変換する |
| Validate syntax | 文法上の問題、名前空間、未対応syntaxの診断 |
| Lower | syntax treeを既存Canonical meaning / typed semantic expression候補へ変換する |
| Discover | キーワード、宣言可能な構造、参照するOperator contractを機械可読に提示する |
| Format（任意） | 意味を保持する正規整形。未実装なら未対応を返す |
| Project（任意） | Editor / AI向けの補完・位置対応をderivedとして提供する |

`Parse`や`Lower`の実装はPluginから供給できるが、受理する意味体系はAccepted Semantic Setとして固定する。Plugin実装が登録された事実だけで新しい文法や意味をAcceptedとして扱わない。意味の有効性に影響するParser/Evaluatorの選択は単なるEnvironment-local Runtime negotiationで決めない。

**Bootstrapの順序（候補）:**

```text
workspace manifest + minimal trusted loader
    ↓ source-kind / language requirements（ファイル拡張子の推測だけに依存しない）
language contracts と syntax dependencies の deterministic resolution / lock照合
    ↓
明示的に選択されたLanguage Pluginのparser / syntax extension setを構築
    ↓ parse → syntax diagnostics（source span保持）
typed operator resolution / semantic checks（既存Semantic Setと照合）
    ↓
Canonical meaning candidate + Source Index
    ↓ Shared Acceptance Boundary
Accepted authoritative source → reload
    ↓
Binding / Realization / Target
```

初回にLanguage Pluginを読み取るため、Workspace Manifestの最小読取りはbootstrap seedとして残る。Language Pluginが自分自身を無限に読ませる循環を前提にしない。固定manifest形式を導入する場合も、TEI固有言語をCoreへ追加し続ける抜け道にしない。

同じ`.tei`ファイル内で複数の文法を自動推測しない。Language選択はsource root / file / explicit directive等から一意に解決する。競合、未知、未固定の場合はUnresolvedとする。

### 4.2 Operator Semantic Contract（読み方と意味）

Operatorは関数名の登録だけではない。少なくとも次を宣言できる必要がある。既存Semantic Extension descriptorと重複する情報は参照で共有する。

| 項目 | 最小の意味 |
| --- | --- |
| `operatorId` / `contractVersion` | 名前空間付きidentityと互換性判断 |
| `signature` | 入出力の型、引数名、nullable性、Cardinality、Collectionの種類 |
| `semanticDependencies` | Relation、Constraint、別Operator、Semantic Interfaceへの要求 |
| `preconditions` | キー一意性、入力型、順序、関係多重度など |
| `effect` | pure / read / invoke等。抽象的な作用を権限と混同しない |
| `meaningPolicies` | 欠損、NULL、比較、精度、順序、空集合、時点、localeの扱い |
| `diagnostics` | 不正なSource、未解決参照、意味の競合、評価不能の区別 |
| `realizations` | SQL / Go / Wasm等へのloweringまたはCapability契約参照 |
| `conformance` | Sourceから到達できる正負のテスト、例、仕様fingerprint |

Operatorの実装場所・速度・物理パッケージが変わっても意味が同じなら、semantic identityを変更しない。ただし同じ名前でもNULL等の契約が違えばsemantic versionまたは別identityで区別する。

Language Pluginは `summarize(...)` と `|> summarize(...)` を異なる書き方として扱える。一方、Operator契約自体は文法の`|>`というトークンを所有しない。

### 4.3 Realization Plugin（意味を実行へ写す）

SQL Pushdown、Go evaluator、Wasm module等は、受理済みMeaning / Operator ContractからExecution Planを作る実現方式である。Compilerが実装を選択するときは必要Capability、Target制約、Cost、Semantic Conformanceを検証する。

pureなOperatorの合成から、Permissionを伴う`invoke`を暗黙に追加してはならない。外部read / Commandは既存のCapability、Binding、Permission、Business Result、Execution Error、Observation境界を通す。

## 5. Operator候補と抽象度

最小セットは**言語機能**ではなく、最初に圧力試験する意味上の操作である。

| 候補 | 意図 | 設計上の注意 |
| --- | --- | --- |
| `from` | 名前付き集合を入力とする | データのread authorityと組み合わせる |
| `where` | 条件による抽出 | 真偽未確定 / NULLを黙ってfalseにしない |
| `pick` | 属性集合を選ぶ | 名称変更だけで別の業務identityを生成しない |
| `derive` | 既存値から新しい値を計算する | Decimal / Moneyのprecisionとoverflow |
| `expand` | 定義済みRelationを辿る | 1:N増殖、未存在、Cardinality |
| `join` | 明示的なキー/Relationで結合 | 片側欠損・多重一致の意味 |
| `summarize` | group key単位で集計する | 空集合、欠損値、countとsumの違い |
| `sort` | 順序を要求する | 既定の順序を推測せずtie-breakerを扱う |
| `require` | 満たすべき条件を宣言する | 不成立と評価不能を分ける |
| `unique` | スコープ内での一意性を検査する | Relation/期間/tenant等のスコープ |
| `diff` | 型付き値や集合を比較する | 同一性の定義・観測の鮮度 |
| `reconcile` | 期待値と実測値を照合する | 合成Operator候補。副作用なし |
| `as_of` | 時点を指定して参照する | 過去データの実在・取得可能性を保証しない |
| `transition` | 業務状態遷移を宣言する | Lifecycleの意味と実行Capabilityを分ける |
| `invoke` | Capabilityを明示的に呼ぶ | 権限検査、結果、観測、再試行制御 |

`group` / `map` / `window` / `rank`等は、この最小セットで表せない具体的なReferenceが現れた場合に追加する。先に全語彙を標準採用しない。外部製品のコマンド一覧をAPI互換として移植することもしない。

## 6. 具体的なDSL表現候補（いずれも未実装）

### 6.1 同じ意味に対する二つの書式

可読性の高いパイプ表現：

```tei
flow sales.daily_amount {
    from sales.order
    |> where(status == "submitted")
    |> expand(relation: sales.order.lines)
    |> summarize(
        by: [ordered_on, lines.product_id],
        total: sum(lines.amount)
    )
}
```

構文を持たないOperator呼出しの表現：

```text
summarize(
  expand(
    where(from(sales.order), eq(status, "submitted")),
    relation=sales.order.lines
  ),
  by=(ordered_on, lines.product_id),
  total=sum(lines.amount)
)
```

どちらも同じOperator IDと型付き意味表現へloweringできるかをReferenceで確かめる。実データの`lines.amount`が本当に存在するか、Precisionが合致するかはReference fixtureで別途確認する。上の例を現在のTEI Syntaxとして読み込めるとは主張しない。

### 6.2 業務Validationの既存意味を継承する

現行の実例：

```tei
validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}
```

比較する候補表現：

```tei
validation sales.order_line.quantity_positive {
    require sales.order_line.quantity > 0
}
```

既存`positive`と新構文が同等になるのは、数値型・欠損入力・評価エラー・精度・Source解決・Validationの所有スコープが一致する範囲だけである。`require`自体をBoolean結果へ短絡させず、`Satisfied` / `NotSatisfied`とEvaluation Errorを区別する既存契約を保持する。Relation一意性についても、空集合、重複、要素の欠損を個別に検証する。

### 6.3 高抽象度Operatorの合成

```tei
derive inventory.stock_gap {
    reconcile(
        expected: inventory.planned,
        observed: inventory.counted,
        by: [warehouse_id, sku]
    )
}
```

`reconcile`は、明示キーによる照合、片側のみの行の分類、型付き差分の計算、結果集合の構成を合成する候補である。`observed`という引数名だけで現在の外部世界を観測したとは扱わない。実際にいつ・どこから取得したかはSource / Observation provenanceに残す。数量ゼロと「片側に行が存在しない」も区別する。

## 7. 意味が曖昧になりやすい点と暫定policy

以下は採用済み仕様ではなく、Referenceで検証すべき設計仮説である。

- **重複:** 集合かmultisetかを型で示す。一般の`where`は入力行の重複を暗黙除去しない。
- **Cardinality:** `expand` / `join`は関係定義を検査する。多対多への増殖を黙って1:1と扱わない。
- **NULL / missing:** nullableな値、入力不在、取得不能、未解決意味を同じNULLへ変換しない。`where`のunknown policyを省略して黙ってfalseにする案は採らず、型・契約による決定または明示を要求する。
- **集計:** `count(*)`、`count(value)`、`sum(value)`の空集合・NULLを別々に定義する。Decimal / Moneyでは丸め時点、桁あふれ、単位を契約で扱う。
- **順序:** `summarize`や`join`に自然順序を仮定しない。`sort`時にはtie-breaker、照合規則、NULLの位置を定義する。
- **時点:** `as_of`は履歴取得能力の存在を示さない。Providerが必要な時点を再現できなければunsupported / unresolvedを返す。
- **評価不能:** 条件不成立、未知の参照、実行環境の不具合を混同しない。
- **安全性:** pure式から任意SQL、shell、外部通信を呼び出せない。副作用は別の明示的Capability invocationへ分離する。

これらの意味が合致しないProviderへの最適化やpushdownは拒否する。実行できることと等価であることは異なる。

## 8. Syntax Extensionの競合と解決

**既定候補は名前付き関数呼出し**とする。パイプ`|>`は単一Language Pluginの表現上の糖衣とし、共通Operator ABIの必須要素にはしない。

独自の中置演算子・予約語・優先順位を複数Extensionが提供すると、字句分割と構文解析の時点で競合する。受理する場合は、Language側が宣言したExtension pointに対して、トークン・優先順位・結合方向・適用範囲・名前解決を含むcontractを固定する。二つの拡張が同じ字面に互換でない意味を与えるなら、読込み順で勝者を決めず診断する。

文法同士が合成不能なら、同じSyntax Treeへ強制統合する必要はない。LanguageごとのASTから同じCanonical meaningへloweringできればよい。Editorは対象sourceに対応するLanguage Pluginを呼ぶ。Formattingによる表面上の整列をCanonical identity変更と見なさない。

## 9. Referenceの設計と失敗例

Referenceを増やす前に、次の最小fixtureを仕様化する。合格と不合格を両方準備し、実装側が都合のよい正常系だけを選ばないようにする。

| ID | 入力・条件 | 必須の観測 | 失敗とする振る舞い |
| --- | --- | --- | --- |
| L01 | 同じSource / 同じpin、Language plugin登録順のみ変更 | 同じCanonical fingerprint | 読込み順で結果が変化 |
| L02 | 未知または競合したLanguage ID | 対象sourceと理由を含むUnresolved | 適当なParserで読み込む |
| L03 | `|>`表現と関数表現 | 同じOperator identity・型付き意味 | 字句の違いを業務identityへ混入 |
| L04 | 未登録Operator | 型付き診断 / source span | Go側の暗黙のfallback関数を呼ぶ |
| O01 | `where` / duplicated records | 重複保持 | 暗黙distinct |
| O02 | `join` / 片側二重キー | Cardinalityの明示/診断 | 1件だけ恣意的に選ぶ |
| O03 | `expand` / optional relation missing | declared policyどおり | 欠損を空集合と偽装 |
| O04 | `summarize` / 空集合・NULL・Money | 明示したidentityとmissing/overflow policy | Provider既定で黙って値が変わる |
| O05 | `sort` / 同順値、NULL | 宣言した安定性・照合規則 | 不定順序を契約に含む |
| V01 | quantity 2.5 / 0 / -1 | 既存positiveと同じ成立/不成立 | negativeを評価エラーへ変える |
| V02 | required input missing | Evaluation Error | 不成立へ丸める |
| V03 | relation-unique / empty, duplicate | 既存Evaluatorとの同値 | emptyを失敗にする |
| E01 | pure`reconcile`からexternal write | capability/permissionなしでは拒否 | writeを実行する |
| E02 | SQL / Go / Wasmの候補が非同値 | そのrealizationを拒否 | 実行できるだけで適合と認める |
| S01 | ソースの物理file pathだけ変更 | Semantic identity不変 | pathをmeaning fingerprintへ混入 |
| S02 | native SQLをauthoritativeとして使用 | SQL source of truthを保持 | DSLへSQLを複写して第二正本化 |
| B01 | Language / Operator version変更 | semantic change candidate / stale検出 | pin無しで自動更新 |
| B02 | Parser artifact再build | Source/Build/Artifact trace確認 | Build成功をAcceptanceと誤認 |

L03の「同じ」は、Languageごとの実装・表示・Source位置が一致するという意味ではなく、比較対象のCanonical meaningが同じという意味である。V01〜V03は既存Ruleの契約をオラクルとする。E02は各Providerで実行できた事実だけではなく、共通fixtureに対する結果・失敗条件を照合する。

## 10. 段階的な実施計画

### E0 — 契約設計と境界の圧力試験（最初）

- Language Plugin、Operator Semantic Contract、Realizationの責務を分離する。
- RQ-032 Semantic Set、既存Compiler、Plugin Lock、BuildAuthorityと接続し、新しい汎用resolverを作らない。
- existing `.tei` sourceとnative SQLのOwnershipを保持する。
- 受理不能・未解決・評価エラーの分類をfixtureとして固定する。

**停止条件:** Language選択に必要なmanifest/lockが循環参照となる、または意味の競合をload順なしに解消できなければ、syntax APIを先に一般化しない。

### E1 — 最小Language Plugin Reference

既存ParserをAdapterで包む方式と、極小の独立Language Plugin方式を比較する。初期段階から全Parserを書き直さない。固定manifestからLanguageを選び、同じSourceから型付きCanonical candidateを再現する。source spanとSource Indexを保持する。

### E2 — 最小Operator Reference

まず既存`positive` / `relation-unique`を新Operator経路から参照し、旧Evaluatorと比較する。続いて`where` / `expand` / `summarize`の少数Operatorでパイプ表現を試す。Evaluation Error、NULL、重複、多重度が曖昧なら標準化を保留する。

### E3 — 高抽象度の業務Operator Reference

`reconcile`を第一候補とし、複数Operatorの合成を展開・診断できるか検証する。異なるProviderで同値な結果を再現できた範囲のみConformanceとして報告する。副作用Operatorの導入は既存Capability実行とAuthorizationが成立した後に限定する。

### E4 — 性能とPortability

Pluginの論理単位と実行artifactを1:1にしない。Linked / Wasm / processを比較し、未使用Operatorのtree shaking、キャッシュ、転送量、起動時間、常駐メモリー、SQL pushdownの意味保持を測る。意味を維持した性能最適化であることが前提。

## 11. 敵対的レビューで残る論点

1. **同じOperatorを複数のLanguageで呼べるだけでは足りない。** 文字列・日時・数値のliteralが別解釈なら同一とはいえない。Lower後の型付き意味とLiteral policyの照合が必要。
2. **高抽象度Operatorの分解が正しいとは限らない。** `reconcile`で期待と実測の片側が欠落した場合、通常の`join`だけでは「存在しない」と「ゼロ」を区別できないことがある。必要なPresence情報を保持する。
3. **最適化が意味を破る。** SQL側の集計NULL、Decimal、照合規則が異なる場合はpushdownを拒否する。
4. **単純な関数呼出しを標準にすると人間にとって読みにくくなる。** 共通契約は関数形でも、利用者に提供するLanguage固有syntaxは業務記述として可読な形を許す。
5. **文法Extensionが任意に増えると互換性爆発を起こす。** ASTを何でも合成するのではなく、Extension pointとLanguage境界を限定する。
6. **self-build可能だから信頼できるわけではない。** Parser / EvaluatorのBuild結果とConformance、Acceptance、Runtime Authorityを別に扱う。
7. **DSLを全面採用することが価値ではない。** 既存SQLで自然に書ける処理はSQL正本のままにし、Semantic Traceabilityだけを接続する方が有益な場合がある。

## 12. 本段階での判断と非主張

採用を提案するのは「小さな意味上のOperatorを合成できる」「DSL自体をLanguage Pluginとして供給できる」「Operatorを構文ではなく意味契約で解決する」という方向性までである。

最終構文、Operator一覧、言語固有AST/IRのschema、Language bootstrap lockの物理形式、SQL推論器の選定、標準namespaceへの昇格時期は、Referenceが揃うまで固定しない。ここに示したDSLは構文例であり、現行のTEI Parserで有効とは限らない。設計候補を実装済み機能やProduction Evidenceとして扱わない。


<!-- dsl-bootstrap-seed-order-recovery-20261106 -->
## 13. 2026-11-06 横断補足：初回Seedから部分登録回復までのBootstrap受入順（#988）

**位置付け：設計・独立Expected Fixture。正式Runtime/API採用でも実TEI Bootstrap E2E試験でもない。** 本書§4.1の「manifest→Language選択→parse→Operator解決→Canonical候補→Shared Acceptance→Target」を、[#2620の既存Role Bootstrap RB-01〜24](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)、[Go Language Reference #2639](https://github.com/hat47x/tei-platform/pull/2639)、[Source/Role接続 #2860](https://github.com/hat47x/tei-platform/pull/2860)と照合する。RB-05/06 eager/deferred、RB-20〜24推移Pin自体はすでにあるため、ここでは**初回解釈器の出所、Parser/Operatorの段階差、部分的な登録・起動回復、Build資格**だけを追加する。

### 13.1 「登録」と「有効化」を分けるDAG（全工程を同じParserで読む単純な直列手順ではない）

| 段階 | 入力の信頼条件／許された処理 | この段階から**まだ**言えないこと |
| --- | --- | --- |
| **K0 既存信頼核** | 固定された実Loader/Manifest Reader・受理済み形式版・Seed側のLock検証規則・小さな基本データ表現。**誰がSeedの実Binary/契約を信頼したかは外部受入前提として明記**。旧Seedに自身の信頼性を自己宣言させない | K0自身の正しさやOwner意思が証明されたこと、任意の新Languageを読めること |
| **K1 Source/Manifest観測** | K0がSource Root/Kind/Import/FormatVersionと取得した原bytesを固定。現`workspace.Result.ManifestIdentity`とLoad時snapshotは部分的に存在し、必要Source全件取得・Import束縛を別に検査（#2823）。未知Formatや未読rootは停止 | parsed/semantic complete、現Workspace以外も網羅したこと |
| **K2 言語選択と隔離したParser候補** | seedで読めるdescriptor/Lockから**正確なLanguage Contract＋実Implementation＋推移依存**を選択し、その版に合うParserだけを候補として使う。#2639の`ParseSelected`は選択済みPinに従う限定Go Reference。新Parserが自分の選択定義を初めて読む循環は不可 | Providerの存在＝信頼、読み込み順による自動採用、Parser成功＝Accepted Meaning |
| **K3 Parse→Canonical候補** | 選択言語でSyntax/Source spanを構成し、Canonical/typed candidateへlower。現Semantic Parserと標準IRの実到達を個別確認。Syntax v2をK0 v1だけで処理できなければ`unsupported/unresolved` | CandidateがAccepted Set/業務意味/実行資格であること |
| **K4 Operator契約を固定して解決** | Parseされたuse-siteに対し、別に**受理されたOperator Contract ID/版/実Implementation Pin**と引数型・効果・依存閉包を照合。`LowerPinnedPipeline`は外部から与えられたPinを使う限定Reference。候補Operatorを発見・登録する作業は隔離し、既存active選択を上書きしない | `operator: pure`自己宣言＝副作用なし、登録された＝合法実行、Plugin自動fallback |
| **K5 意味受理境界** | K0〜K4のSource/Parser/Contract資格、独立Required Shape、独立Expected、必要ならOwnerの**外部の明示決定**を同一対象・版・Scopeで照合。提案段階のConformanceと正式Meaning Acceptanceは別 | 記述自身の`required=[]`やUnit PASSがOwner受理となること |
| **K6 Build/Target/Active Consumer** | Accepted Source/Meaning・実Build Input・Builder/Toolchain・ArtifactSet・Consumerの現在資格を各既存Ownerへ束縛する。受理前の隔離**候補Build**は検証として可能だが、release／active generationとは別 | Build成功＝Seed信頼、意味受理、Provider Effect、現Host Grant。TEI自己ビルドStage1→Stage2実再入は#2831の別Claim |

**順序上の注意**：Parser実装を選ぶにはその前に**Seedで読める記述形式**が要る。通常のDSLのParserが決まった後でなければそのDSLのOperator use-siteは解析できない。一方、Operator契約の候補は先に**Seed由来の既知Lock**として収集・Pin可能である必要がある。「候補収集」「一時Staging」「新Activeへの切替」を一つの`register`で処理すると、途中失敗から半登録・曖昧な意味選択が発生する。二相に見える構想は**要求する安全性**であり、現製品にatomic registry swap/transactionが存在するという主張ではない。

### 13.2 初回信頼核・自己記述循環の棄却条件

- **Seed Trustの明示**：K0の実装由来・形式仕様・Trust anchor・起動時に選んだ実Artifact/版と、後段Candidateが説明する「信頼されるべきParser」を分離。実検証・所有者の責任境界が不明なら`unqualified`。Checksum一致だけで信頼された作者・意味Owner・権限を証明しない。K0の変更/新Grammar導入には**以前の信頼経路で検証できる移行方法か別途認証済みSeed**が要り、新DSLが自己更新を自己承認しない。
- **循環は位相で判定**：K0 Reader→選択Descriptor→Language Parser→同Parserを初めて解釈するにはそのParserが必要、という**初回解釈の前提閉路**は拒否。既存RB-05のeager循環をBootstrapの段階に置く。反対に`Node.children: Ref<Node>`等の遅延関係はRB-06のとおり許容候補であり、意味評価・Traversalに入るまで自動展開しない。
- **版不一致は別の差異**：Manifest FormatVersion（#1920）、Language syntax Contract/実Parser、Operator Contract/実Implementationと推移依存、Source revision/occurrence、Accepted Meaning Review、Build Toolchain/Artifact/現在Hostの資格を分離。同一Language ID/semverでも実byte違いは不合格。同一Source A→B→Aでも旧Review/Grantは自動復活させない。未知・新しい版を黙って旧Parserへfallbackしない。
- **物理Plugin方式とは独立**：Static/linked、動的、外部Processのいずれでも「どの実Implementationを選ぶか」「未承認コードがロード時に実副作用を起こさないか」は別。Seed自身に任意Pluginコードを無検証実行させない（方式選定Owner #951/#1251/[Draft #2885](https://github.com/hat47x/tei-platform/pull/2885)）。正式配布方式・Go単一Binaryとの適合は未決。

### 13.3 部分登録・障害後の回復：状態を一つに潰さない

| 故障時点 | 独立に必要なExpected／復旧動作（設計） |
| --- | --- |
| **A：Manifest/Lock/Parserの事前検査で失敗** | 未受理Candidate／新OperatorをActiveへ露出しない。既存`g1`を保全できた事実を現Snapshotで確認する。上流に未知Sourceがあれば以前の成功と現在資格を混同しない |
| **B：新候補`g2`のOperator Aはstaged、Bの登録に失敗** | **部分集合Aだけを有効にしない**。隔離済みなら`g1`維持・`g2`破棄または全Pin再検証。隔離していない既存実装なら「`g1`は無傷」と仮定せず`partial/unknown`で停止し実状態を再読取 |
| **C：Owner受理済みだがGeneration切替前に停止** | Ownerの意味判断は履歴として保持。切替が未完了なら`g2`有効化やBuild配布を主張しない。再起動後にCurrent Source/Lock/受理ScopeとTargetを再照合 |
| **D：切替試行後のACK/記録応答を喪失** | **切替済みか不明**を保持し、実永続状態・公開Generation・Review/Receiptを独立再読取。盲目的に同じ登録を再送／自動Rollbackしない（Source受理ツールの`Source changed, Decision ledger failed`とも区別） |
| **E：旧Source/旧Operatorへ戻す（ABA）** | 内容同一でも旧受理のCurrent資格は復活させない。新たなRevision/Review/Pin・必要Consumerを再判定 |

全ケースで、Kernel/実Process・Staged登録・Active Selection/Lock・Canonical Candidate・Accepted Meaning・Build Artifact・Consumer/Host Grantを別々に記録する。実装が「候補をどこへ置くか」「切替原子性」「再起動時に何を読むか」をまだ持たないなら**合格ではなくGap**とし、第二のRegistry/Host/Recovery Engineを作らない。

### 13.4 独立Expected付きFixtureと最初の結合試験

**新しい設計専用の入力・期待対照**：
[`plan/fixtures/role-bootstrap/bootstrap_order_recovery_expected_20261106.json`](../fixtures/role-bootstrap/bootstrap_order_recovery_expected_20261106.json)（**BS-01〜15**、`DESIGN_ONLY_NOT_EXECUTED`）。
`expected`はCandidateが返した値でなく、**起動Trust、版固定、eager依存、独立Required、意味受理、障害復旧**の先行契約から決める。値`g1/g2`、各状態名、模擬digestはFixtureの仮名でありProduct API/実Owner決定/暗号学的な真正性ではない。

**特に判別すべき対照**：
- BS-02：Manifest未知版→Seedで停止、Parserを呼ばない。BS-03：同versionでも異Implementation bytes→Pin mismatch。
- BS-04：初回Parserのeager自己循環→停止、BS-05：通常の`Ref<Node>`→Bootstrap上は停止しない（ただしMeaning未受理）。
- BS-06/07：Syntax可でもOperator未Pin・Required Root欠落→Candidateのみ。
- BS-08/09：半登録・再起動→新Generationを公開しない（隔離が証明されないと旧Generationの完全性もunknown）。
- BS-10/11：Parse/Conformance/候補Build PASSでもOwner意思が無い→Accepted/Releaseにしない。
- BS-12/13：外部受理済みでも、Confirmed CutoverとACK-loss unknownは別。前者もBuild/Host資格を発行しない。
- BS-14/15：A→B→AのReview旧権限非復権、Seedで読めない新文法による自己移行拒否。

**検証順とOwner**：(1)既存`check_bootstrap.js`のRB-01〜24は改変せず、まず本FixtureのInput/Expectedを仕様レビュー（#988/#2620）。(2)実`workspace.Load`/Manifest/Snapshotと`ParseSelected`/pinned loweringが対応可能なBS-02/03/06を実Goでpressure。対応不能なK0 Provenance/Registration切替は`unsupported`と記録。(3)BS-08/09/13はExisting Extension/Build/Acceptanceの**実観測点**が分かるまでは試験Passを宣言しない。(4)独立Oracle #2514、Raw Authoring #2621、Required Source #2823、Build #2831、Consumer/Host #2516/#1174に各観測を渡す。**新Builder/Runner/Seed Parser/第二Lock・Registry・Owner Authorityは先行追加しない**。

**証拠上限**：本§13/BSは設計と静的実コード照合のみ。BS-01〜15のTEI Go/CLI/CI、独立実Owner受理、実Pluginのpartial registration、Recovery/Crash、製品自己Bootstrap、Stage2 TEI self-build、実Host/DBは**NOT RUN / NOT PROVEN**。#1119のG0〜G4と業務P0を拡張しない。
