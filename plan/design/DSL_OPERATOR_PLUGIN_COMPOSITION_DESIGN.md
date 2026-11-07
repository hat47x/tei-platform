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

L03の「同じ」は、Languageごとの実装・表示・Source位置が一致するという意味ではなく、比較対象のCanonical meaningが同じという意味である。V01〜V03では既存Ruleの意味契約をテスト判定基準の出典候補とし、実Evaluatorの結果をそのまま独立期待値にはしない。E02は各Providerで実行できた事実だけではなく、共通fixtureに対する結果・失敗条件を照合する。

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

**検証順とOwner**：(1)既存`check_bootstrap.js`のRB-01〜24は改変せず、まず本FixtureのInput/Expectedを仕様レビュー（#988/#2620）。(2)実`workspace.Load`/Manifest/Snapshotと`ParseSelected`/pinned loweringが対応可能なBS-02/03/06を実Goでpressure。対応不能なK0 Provenance/Registration切替は`unsupported`と記録。(3)BS-08/09/13はExisting Extension/Build/Acceptanceの**実観測点**が分かるまでは試験Passを宣言しない。(4)独立検証基準 #2514、Raw Authoring #2621、Required Source #2823、Build #2831、Consumer/Host #2516/#1174に各観測を渡す。**新Builder/Runner/Seed Parser/第二Lock・Registry・Owner Authorityは先行追加しない**。

**証拠上限**：本§13/BSは設計と静的実コード照合のみ。BS-01〜15のTEI Go/CLI/CI、独立実Owner受理、実Pluginのpartial registration、Recovery/Crash、製品自己Bootstrap、Stage2 TEI self-build、実Host/DBは**NOT RUN / NOT PROVEN**。#1119のG0〜G4と業務P0を拡張しない。


### 13.5 実Go APIに結合する最小Expected検査（BS-02/03/06）

`reference/dsl-language-plugin/bootstrap_expected_bridge_test.go`を追加し、**設計Fixtureの3ケース**だけを現在の実Go入口に渡す。JSON Expectedは製品APIの結果から再生成せず、実テストはExpectedの`earliestStop`を先に照合する。実Contextや登録機構が存在しない場合に仮のProvider RegistryやBootstrap Hostを作って検査成功扱いしない。

| Case | 実Go API | 確認対象／証拠の上限 |
| --- | --- | --- |
| **BS-02** | `workspace.Load` | `formatVersion: 99`で現在のWorkspace Loaderが明示拒否。Seed Manifest形式の限定防御。実選択済みLanguage Parser自体の初回起動証明ではない |
| **BS-03** | `ParseSelected`→`Resolve` | `impl-A`のFingerprint `dA`を要求し、同ID/同Contractでも`dB`候補では`ErrProviderMissing`、Parser callback未呼出。**dA/dBはFixtureラベル**で実バイナリのhash/証明ではない |
| **BS-06** | `ParsePipelineSyntax`→`LowerPinnedPipeline` | valid SyntaxでもOperator Pinがない場合、元Operator Token spanへの`PipelineDiagnostic`、Resolved Stepなし。Product Operator登録の有無・Owner意味受理は証明しない |

テストは**既存Packageの小さなGo試験**であり、第二Seed Reader/Resolver/Runnerではない。BS-01/04/05/07〜15（実初回信頼核、自己記述循環のProduct経路、独立Required/Owner、部分登録・Crash Recovery・Cutover、Build Release）をこの3件で合格にしない。**新Goテストをコミットした事実と exact HEAD CIのPASSは別**。CI結果の有無・Go packageの実行証拠を確認してから限定受入する。


### 13.6 実Plugin Resolverとの接続：論理解決の全件性と実有効化は別

**現mainの実コードを追加照合**：

- `internal/plugin/resolve.go: Catalog.ResolveEnvironment` は依存を一時的な`resolved` mapへ組み立て、欠落・cycle・bundle競合などのエラーで`ResolvedSet{}`とerrorを返す。結果のPlugin/Capability一覧は全解決後に生成され、失敗時に中途半端な`ResolvedSet`を返さない。**これは純粋な論理解決の局所的な全件性**であって、実RuntimeへPlugin Aだけ登録後にBで失敗したときの原子取消・永続回復ではない。
- `internal/plugin/lock.go: BuildLock / ResolveEnvironmentWithLock` は、選択したDefinition fingerprint／Artifact Digest／指定Lock版（build-trace付きの版はPlan fingerprintも）を検証し、失敗で空値＋errorを返す。Lockの**宣言値とCatalog内Definitionの一致**は扱うが、実ロードされたバイナリのbytes・その出所の信頼・ロード時init副作用、Current Host Grant、部署後のProcess世代を保証する証拠ではない。実Artifact実測は別のBuild/Runtime Ownerと束縛する。
- `internal/plugin/bootstrap.go: EvaluateBootstrapConvergence` は**ビルダー収束（SB-3）**の証拠構造・fingerprint比較。DSLを最初に解釈するK0 Source Reader／Language登録器ではない。名前が`Bootstrap`であることを機能同一性の根拠にしない。

**Go単一バイナリの標準配布への整合**：Runtimeが静的合成なら、候補の`staged`／`active`を**同一Process内に動的Registryを二世代置く要件**と解釈しない。物理的な境界は、候補Build Artifactと既存Process Image／Host配備選択／起動時Lock／現在の実行主体となり得る。検査すべき共通不変条件は、**未検証候補が現行資格を汚さず、何を有効化したかを再起動後に独立観測できること**。静的合成・動的ロード・外部Processの方式により観測対象は異なり、いずれも現在のProductとして原子的切替／再起動後復旧を実証済みとはしない（#951/#1251/#2831/#1174）。

**BS-08/09/13の解釈補正**：これらは全形式に共通する**安全要求**で、`Catalog.ResolveEnvironmentWithLock`の失敗時空Resultのみを以て「部分登録・Crash Recoveryの製品実証PASS」と判定しない。静的配布なら`g1/g2`は仮の配備世代、動的登録なら仮の実Registry世代。どちらも永続状態とCurrent Host資格を実観測できない場合`not-proven`。新しいmutable Registry／切替APIをこの§だけから導入しない。


### 13.7 `*.tei.md` 標準Authoringへの移行は、K0の構文選択と別の受理事件

2026-11-06に選択された**TEI Authoring Sourceの標準ファイル名`*.tei.md`、旧`*.tei`原則廃止**の方向性は、[main統合済みの移行計画 #3030](https://github.com/hat47x/tei-platform/pull/3030)がM0〜M6／TM-01〜14で所有する。このDSL Bootstrapでは**名前・構文・受理済みMeaningを別に検証する停止条件**だけを追加する（重複する新Loader/Markdown Parser/移行Runnerは作らない）。

- 現mainの`workspace.Load`は`*.tei`／`plugin.tei`等の固定発見経路を含み、`filepath.Ext("sample.tei.md")`は`.md`である。`*.tei.md`へ改名しただけでは旧Sourceが新しいManifest/Loaderに発見・読込・Parser選択されることも、原Markdownの糖衣構文を保持してCanonicalへ変換できることも保証されない。
- **K0の有限Seedはファイル名だけをSyntax Authorityとしない**。Seedで読めるWorkspace/明示Source Kind・Language Contract Pin・版と、原文Markdown Readerの最小処理範囲を先に確定。Markdown全文／埋込DSL／OKF等の追加文法を**そのPlugin自身の未解釈記述だけで初めて選択する**のはBS-15の初回自己解釈循環に当たる。選択不可能ならRaw Draft／読取可能なSourceとして保持し、Semantic Promote・Buildを停止する。
- 移行の反例は旧`semantic/data.tei`正常→同bytesで`data.tei.md`へrenameして`workspace-files`未列挙、またはMarkdown表／Frontmatterが現ParserのDSL grammarと異なること。**Expectedは「同内容なら意味一致」ではなく「Manifest実観測・選択Parser・原Source/糖衣・現行受理版を照合できるまではnot-qualified」**。古いCIの`*.tei` Goテスト成功は新形式の受理証拠にならない。
- K0→K2の独立Expectedは既存BS-02（Seed形式拒否）／BS-15（新Grammar自己起動不可）へ接続し、Source inventory/Load被覆は#2823、Raw Draft/Markdown Source Fidelityは#2621/#2497、形式移行とRuntime Regressionは#3030/#1920、SHO/OKFとの橋渡しは各担当に委譲する。

**資格**：これは#3030でmainへ格納された**Authoring方針と移行設計**を#988 Bootstrapへ反映したもの。新`*.tei.md` Loader/Parser／既存Sourceの一括移行／同一Meaning/GUI保存／Owner Meaning Acceptanceは未実証。既存の`BS-01〜15` Fixture件数とGo focused testの実入力は変更しない。


<!-- tei-bs-test-discriminating-positive-controls-20261106 -->
### 13.8 BS-02/03/06の「常時拒否」Mutant対照（局所Go試験を補強）

BS-02/03/06に単独の拒否条件を置いただけでは、実装が**すべての入力を拒否する**誤実装でもテストに合格する。#2514の独立期待値方針を踏まえ、同じ実Source/同じReference関数に、**一因子だけ修復した正常対照**を追加した（Go `bootstrap_expected_bridge_test.go`）。独立Expected JSONのBS番号・停止条件は変更しない。

| Case | 不正側（独立Expectedに対応） | 正常側の一因子差分 | 排除できる限定Mutant | まだ証明しないこと |
| --- | --- | --- | --- | --- |
| BS-02 | `formatVersion: 99`→Workspace Load拒否 | 同じ`semantic/data.tei`でManifest版だけ`1`へ変更→Load成功 | Loaderが常に拒否する／既知版も拒否する | 本番Language Plugin bootstrap、`*.tei.md`の読込、Source全root |
| BS-03 | Locked Parser `dA`に対して実候補`dB`→拒否・Parser未呼出 | 同じSource・同じParser callbackで`dA`と一致→候補を返しcallback実行 | Language Selectionが常に拒否する | fingerprint文字列が実Plugin bytesと一致・Owner承認 |
| BS-06 | `rows/order |> sum`にOperator Pinなし→Source位置付き拒否 | 同一SyntaxとContract catalogでPinだけを補う→1 Stepと同じSource Span | Operator Loweringが常に拒否する／Spanを失う | Operator登録の原子性・実行結果・正式意味受理 |

「上記Mutantを検出し得る」と「Mutation Runnerで故意にMutantを生成し検出した」は**異なる**。現時点で作成したのは正負対照の実Go試験Sourceであり、Mutation Runnerの実行や業務RuleのKill率測定はしていない。この3件だけでBS-01/04/05/07〜15や#1119 Gate、独立Owner意思の検証へ拡張しない。

**検証票**：各Caseに`negative_input`／`negative_expected`／`positive_delta`／`positive_expected`／`exact_head_ci`／`claim_ceiling`を**報告欄**として持てばよい。新しい正式Fixture Schemaは導入しない。候補ExpectedはGoの出力から逆算せず、先に固定したJSONと既存`workspace.Load`／`Resolve`／`LowerPinnedPipeline`契約を対照する。


<!-- tei-dsl-bootstrap-selected-source-and-cross-stage-20261106 -->
### 13.9 同一SourceのK2→K4閉包、候補解決と受理済みLock、`*.tei.md`入場拒否の統合順

2026-11-06に進んだ別レーンの[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)は、宣言済みSource rootの管理対象内で`*.tei.md`が発見され、**現Native Loaderに対応Markdown Parserが束縛されていない**場合、原文を旧`*.tei` Parserに流さず、`TEI_MD_PARSER_UNBOUND`と該当Source位置で明示停止する。対象はSemantic/Query/Persistence/Plugin/Bundle/Releaseの限定Root、通常`.md`や宣言Root外の`.tei.md`は勝手に入場させない。HEAD `63953ccb`の専用CI/Repository Healthはsuccess（**Draft、main未統合**）。これはK1「候補が存在するが解釈できない」を表示できる最初の実Product停止であり、K2 Language Pinの実選択、Markdown AST、Canonical、Owner受理が成立した証拠ではない。

**同一Sourceと二つのPinを一つのGo Reference実行で照合**：本PRの`TestBootstrapSameSourcePinnedLanguageThenOperator`は`rows/order |> std.sum`を固定し、`ParseSelected`で選択したParserの`ParsePipelineSyntax`候補を、同じSource token spanのまま`LowerPinnedPipeline`へ渡す。既存JSONの**BS-03（Language実装Pin不一致）**と**BS-06（Operator契約Pin不一致）**の独立Expectedを先に読み、同Source・同Parser実装で、1) 両方一致→1 Step候補、2) Language実装不一致→Parserを呼ばず拒否、3) Operator Pin不一致→Parserは成功するがtyped loweringで拒否、4) Catalog上の同Pin契約の出力型変更→拒否、を分離する。新Go Core/Parserは導入しない。**Product Workspace ReaderとこのSynthetic Pipelineは同じSource/同一Attemptではない**ため、K1→K2の正式Product結合証拠には昇格しない。

**Language/Plugin候補解決と「固定済み実装」は別**。実`internal/plugin/resolve.go: Catalog.ResolveEnvironment`は要求されたCompatibleMajorにおいて候補Catalogから最高互換版を選ぶ。これは**初回候補生成**として有用だが、K0 Seed/Accepted Meaningの実装資格にはそのまま使えない。一方`internal/plugin/lock.go: Catalog.ResolveEnvironmentWithLock`は既存LockのPlugin Definition Fingerprint／Artifact Digest／対応版に固定して別途照合する。だがこれも**catalogに書いたDigestと実ロードされたProcess Image/Bytesが一致する証明ではない**。#1251/#2831/#1163の既存Build/Actual Process観測を同一Attemptに結び、未検証・旧Receipt/旧LockをCurrent資格にしない。

| 反例／同一入力の一軸変更 | 独立Expected（既存Fixture/Owner） | 現在の到達範囲 |
| --- | --- | --- |
| 同じ`*.tei.md`候補がmanaged root内→外へ移る、通常`.md`と入替 | K1で「管理候補/非候補」を区別。**Root内でParser未束縛なら明示拒否**、外は新規取込しない（#2823/#3031） | #3031の実Goは当該HEADのみ。#3028とは別CI |
| 同一SourceでLanguage Pin正常→不一致 | 不一致ではParse callback未呼出（BS-03） | #3028 Go Referenceの同一Source対照（Product Reader未結合） |
| 同一SourceでOperator Pin正常→不一致、Contract OutputTypeだけ変更 | Parseは成立してもK4のError/Source Spanは別（BS-06） | #3028の限定同一Source Go Reference |
| 未受理候補Catalogへ互換Plugin新版追加／受理済みLockは不変 | 候補解決の結果は変わり得るが**現実行資格の暗黙更新は禁止**（#988/#951） | 既存`lock_test.go`の版固定試験を再利用。新Registry/Version Resolver不要 |
| Provider/Build ArtifactがCatalogの文字列Digestと異なる、または実Builder/Processが別 | Lock文字列一致だけでK0 Trusted・K6 Build/Host合格にしない（#1251/#2831/#1163） | 実ロードbytes/Process+Artifact同一Attemptは**未証明** |

**Fixture自身と実行証拠の区別**：当初独立Expected JSONの上位`status: DESIGN_ONLY_NOT_EXECUTED`は、BS-02/03/06を実Goテストへ接続後には全体を「未実行」と読めて不適切だったため、`INDEPENDENT_EXPECTED_NOT_OWNER_ACCEPTED`へ改訂した。JSONは**期待値の出所とOwner未受理**を示すもので、実行済みCase/CIの状態は更新HEADのActions/Go Testから別途記録する。15のCaseと`earliestStop`は維持し、残る12ケース、Owner意思、実Provider/登録回復はProduct未実証のまま。

**次の限定実証順**：#3028の同一Source Go選択試験をexact HEADで検査→#3031のProduct fail-closed Readerと**マージ前の同一HEADで**K1入場拒否を再観測→#3030移行Ownerが正式Markdown Parserを選択・Pinできた場合だけK1→K4一貫実Source/Source spanを比較→既存#1251/#2831/Extension Lock/HostへArtifact/実Processを引渡す。別Draftの個別CI PASSを足して単一K0→K6 E2Eと呼ばない。新Seed/Registry/Runtime Backend/Owner判断プロセスを追加しない。


<!-- tei-dsl-bootstrap-required-source-scope-naming-20261107 -->
### 13.10 必須TEI Sourceと非管理Draftを混同しないK1受入、独立期待値の用語

[移行仕様のmain統合 #3030](https://github.com/hat47x/tei-platform/pull/3030)の2026-11-07追補では、現[Guard Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)が**管理Root内で`*.tei.md`を一つでも見つけるとLoad全体を停止し得る**点を、正式Parser導入時の受入へ引き渡している。Guardは現状の誤った成功を防ぐ暫定処置で、任意Markdown草稿を意味Sourceへ自動分類する製品仕様ではない。

K1に要求する区別は**(a)宣言された必須TEI Source、(b)管理対象として発見されたがParser未束縛の候補、(c)非管理Raw Draft、(d)普通のMarkdown**。必須なら未読込・未解析を`0件の成功`にせず、候補として存在するだけならOwnerの用途・必要集合が確定するまでRead-only/unknown、非管理Draftや通常Markdownを自動実行・意味受理せず不要な妨害も避ける。

K1の新規Schema/Registryを増設する前に、#3030のTM/AU・#2823の必要Root・#2621のRaw Draft受理境界に以下の**一因子反例**を付す：
- 同じ`model.tei.md`を明示した必須Source Root内へ置く場合、未束縛なら`TEI_MD_PARSER_UNBOUND`。Root外のSHO下書きへ置く場合、Semantic Worldへ入れず、必須Closureの対象へ勝手に昇格させない。
- `README.md`の説明文に`tei`が含まれてもLanguage選択の根拠にしない。正式な`*.tei.md`は拡張子だけで採用せず、Manifest/Source KindとPin済Parserが決まるまでMeaningへ進めない。
- 新`model.tei.md`がSource Indexに存在しても、実Parserで消費されCanonical/Build入力に接続されたと推定しない。元`model.tei`と併存し同じStable IDを申告したときは黙った新旧優先順位を使わず、#3030の独立期待値で衝突を判別する。

**既存Evidenceの上限**：#3031 exact HEAD `63953ccb`の専用CI/Repo Healthは暫定拒否の実Go資格。#3028のBS-02/03/06・同SourceGo試験は旧Native Source/合成Pipelineの限定資格。両者の同HEAD・同Source実結合、正式Markdown Grammar/Source位置/Meaning/Owner受理、実BinaryのBuild/Host/Receiptは未実証。受入基準は各既存Ownerへ移譲し、#1119横断Gateを自動拡張しない。

**用語**：[main `docs/naming.md`](../../docs/naming.md)（merged #3039/#3040）の区別に従い、判定規則は**テスト判定基準**、対象実装から独立した期待結果は**独立期待値**、根拠と適用範囲を含むものは**独立検証基準**と記載する。新Fixtureの説明フィールドは`expectedBasis`へ改名した（JSON構造は設計専用、Product ABIではない）。DB製品のOracle Database、既存API/ID/履歴を無断改名しない。


<!-- tei-dsl-bootstrap-real-implementation-reference-join-20261107 -->
### 13.11 Plugin実装Source→BuildPlan→Runtime呼出しの既存参照経路を再利用する（同一Attempt接合は未完了）

新しい`Language/Operator→Build`専用Runtimeや別Registryは作らない。現mainの以下が**既に実装された限定足場**であり、#3028のGo Referenceと混同せず接合条件を作る。

| 既存実装・試験 | 直接確認できる範囲 | そこから推論しないこと |
| --- | --- | --- |
| `internal/workspace/plugin_implementation_loader.go` | `plugin-implementations/.../implementation.tei`を宣言Source Rootから実読込・Parseし、Plugin Definitionとの照合、実装Source位置・Source Index `RoleImplementation`を保持 | 正式`*.tei.md`のParser、Language選択Pin、実Binaryの起動 |
| `internal/workspace/plugin_native_implementation_reference_test.go: TestTEINativePluginImplementationReferenceRunsThroughExistingBuiltinABI` | 実WorkspaceのPlugin Definition/実装Source raw bytesをHashし、`BuildPlan`のSource Digest、`implementation.Materialize`が生成する`firstBytes/firstDigest`、Build LockとRuntime Engine Lock、Builtin登録、`DeploymentExecutor.Execute`と`PluginInvocation`を限定的に照合 | 事前のLanguage/Operator Pinからの自動Build、実OS Image/Go静的Link、正式実Provider、Owner受理・Host current Grant |
| `reference/dsl-language-plugin/bootstrap_expected_bridge_test.go`（本Draft） | 同一Pipeline文字列`rows/order |> std.sum`のK2 Exact Language→K4 Exact Operator Contract、拒否側/成功側と元Source Span | 上記Native実装Sourceとの同一改訂、Product WorkspaceでのAuthoring入場、生成Artifactの実Link/Invoke |

**次の縦断では独立に固定した1つのExpectedを保ったまま、次の証拠の結合を要求する**。

1. `Selected Source`：Authoringが意図したSource Kind、管理Root、実相対Path、raw bytes Digest、Source/Occurrence Revision、Required集合、Parser Language Contract/実装Pinを、実Readerの`read`/SourceIndexと対照する。Source Indexに列挙されただけの`loaded=unknown`は実読込扱いしない（#2823/#3030）。
2. `Actual Consumed`：**実Readerが読んだのと同じbyte列**がParser/Operator/Canonical候補の入力となったWitnessを対応づける。BS-03/06はこの部分の局所Reference基線であり、今のProduct Loaderと同一Attemptではない。
3. `Build Inputs`：Plugin Definition/Implementationのbyte Digestに紐付く`BuildPlanFingerprint`、Builder Identity、選択されたOutput Artifact Digestを記録する。上記Builtin Referenceはこの内部照合のFitだが、Source意図と実Buildを同一にしたことの証明ではない。
4. `Actual Runtime/Consumer`：現在のBuild/Runtime Engine Lockと、実登録・実呼出先・返却された`PluginInvocation`（選択Plugin/Artifactと同一）を照合する。実Go単一バイナリの最終的なリンク/起動対象や配備Hostは既存#1251/#951/#1016のOwnerで別に検証する。
5. `Meaning / Authority`：既存のOwner Accepted Meaningの出典改訂とScope、Hostの現在Grant、Provider/DBの効果Receiptは別資格。上記の参照試験が全部成功しても自動昇格させない。

**現行Referenceの実証上限に関する重要な注意**：`plugin_native_implementation_reference_test.go`の`nativeReferenceBuilderDigest`/`nativeReferenceEngineDigest`は固定テスト値であり、Runtime Engineの実行ファイルのHashを計算した証拠ではない。またこのFixtureは従来の`plugin.tei`/`implementation.tei`を使用し、移行中の`*.tei.md`からの正式Language選択を行わない。

**検出すべき最小の誤実装（一軸差分）**：同じLanguage/Operator意味とSource Revisionを持つPlugin Bを選択し、Build/Invocationだけ旧Plugin Aへ取り違える（Bの意図/選択、Aの実消費／同じ返値）。結果値が一致してもBuild/InvocationのPlugin Identityと実入力byte digestが一致しなければ**不適合**。まず[既存Runtime Reference](../../internal/workspace/plugin_native_implementation_reference_test.go)と[#2885のD1選択→Link→Invoke検査](https://github.com/hat47x/tei-platform/pull/2885)のOwnerへ、**同一Attempt・Source Revision・Target/Runtime**でこの差を判定できる観測点を引き渡す。選択B/実呼出Aの実Mutant/正負比較はまだ**NOT RUN**、新しいProduct Registryは不要。

Source移行では、`plugin.tei.md`を選んだ意図Bと、固定名Loaderが`plugin.tei`を読む実態Aがずれる例を[#2903 §24.4](https://github.com/hat47x/tei-platform/pull/2903)と共有する。現[#3031](https://github.com/hat47x/tei-platform/pull/3031)の`TEI_MD_PARSER_UNBOUND`暫定拒否を正常Parseと誤認せず、#3030のTM-03/12/14と#2823のRequired/Loaded/Consumed区分を再利用する。**いま別DraftのCIが個別にPASSでも、このK1→K6の同一Source結合は未実証**。

**品質ゲートの独立性**：#3028旧HEAD`0f6e7045`/ `5f335cc9` のRepository Healthは`gofmt`差でFAIL、同runの`go test ./...`はPASS。追加テストのStruct literal整形をGo formatterで比較したうえでcommit `62f2dd22`へ修正した。最新HEADのCI結論は完了後にのみ採用し、Go PASS≠Go fmt PASS≠正式Product受理と明記する。


<!-- tei-dsl-bootstrap-attested-computation-source-coverage-20261107 -->
### 13.12 OKF Attested ComputationとSource読込観測をK0の自己証明へ昇格させない

[横断テスト設計Draft #2811](https://github.com/hat47x/tei-platform/pull/2811)と[意味/Grant対照Draft #3026](https://github.com/hat47x/tei-platform/pull/3026)において、OKF `Attested Computation`／deterministic attesterの検証成功は、**計算に使った宣言と結果の一致を示す限定証拠**として位置づけられている。DSL Bootstrapでも、Seed Reader/Parserそのものの出自・許可をそのParserで読み込んだ記述や自己発行Receiptで承認しない（BS-01/15）。既存#2514の独立期待値と証拠発行者・対象版・改訂・Owner受理を**別の観測**に維持し、第二Attester Engineを追加しない。

**実Source消費の負例がより具体化**：[Draft #2811](https://github.com/hat47x/tei-platform/pull/2811)に記録された[TAIの実CLI負例Run #42297500012](https://github.com/hat47x/tai-pleroma/actions/runs/42297500012)では、旧固定Loaderを使い、`*.tei.md`の原文変更でSource Index Fingerprintが変わってもWorld/Catalogが不変、`*.tei`からのrenameでSubjectが消えても`validate.ok=true`となる例を観測している。**これは別プロジェクト/旧固定版の反例**であって、現在の[#3031](https://github.com/hat47x/tei-platform/pull/3031)の暫定拒否が機能しないという主張ではない。K1の`discovered`/Index変化がK3 `parsed`、K4 `actual consumed`、K6 Buildへ自動継承されないことを裏づける。

**独立Expectedの差分判定（新Caseを増やさない）**：既存BS-07（必要Source集合の偽の空）、BS-10（Parser/Conformance PASSとOwner Acceptanceの分離）、BS-15（未知GrammarでSeed自己起動不可）を、#3030 TM-12/14、#2823 Read/Consumed、#2514の既存判定基準に照合する。判定に必要な対照は、**IntentしたSourceの必要集合、Readerで実観測されたbyte列、Parserが実消費したbyte列/Source span、Build入力、現行Owner Meaning/Host**。Index Fingerprint一致だけ、`validate.ok=true`だけ、Attesterの`verified`だけ、またはCandidateのRequired自己申告だけでは、どの層の未観測も埋めない。承認・作業・実DB効果の境界は既存Ownerを保持し、#1119へ未検証Gateを追加しない。


<!-- tei-dsl-k1-readbyloader-actual-witness-20261107 -->
### 13.13 main統合済み `readByLoader` を使ったK1実Source読込の限定実証

[merged PR #3078](https://github.com/hat47x/tei-platform/pull/3078)で、`internal/workspace/loader.go`の`ManagedFile.ReadByLoader`と`SourceRootFindings`がmainに統合された。`readByLoader`は、管理Root内のファイルを**domain Loaderが実際に読んだかどうか**を、`sourceSnapshot`に存在する実読込Source pathから投影する。単なるInventoryやSource Index Artifactの列挙と区別するが、形式に合ったParse、Canonical意味受理、実Buildへの入力、現行Host Grantは保証しない。

この差をテストの独立した観測にするため、本PRに `reference/dsl-language-plugin/bootstrap_actual_read_test.go` を追加。実`workspace.Load`と`workspace.BuildSourceIndexFromLoad`を使用し、**一つのManifest（formatVersion 1、`sources: [semantic]`）**のもとで次を検査する。

| 実File | ManagedFiles | `ReadByLoader` | Source Index Artifact | 限定的に証明できること |
| --- | --- | --- | --- | --- |
| `semantic/model.tei`（`organization`の型・Data宣言） | あり | `true` | あり | 既存Native Loaderがこのbyte列を読んだ |
| `semantic/README.md` | あり | `false` | あり | Inventory/Indexに含まれても実DSL Parserが読んだことにはならない |
| `drafts/unmanaged.tei.md` | なし | 該当なし | なし | Manifestで宣言されないDraftを管理Root／意味Sourceへ勝手に取り込まない |

**反例**：`Source Index Artifacts`が2件あることを根拠に`parsed sources=2`／`accepted meaning=2`と数える実装は誤り。追加Goテストでは、Native Sourceからの`Index.References`が少なくとも1件ある一方、未読込の`README.md`から意味`Reference`を生成しないことも確認する（ArtifactがあってもSubjectを推定しない）。独立のInput期待と、`readByLoader`の実観測、Source Indexの管理Artifact列挙を**別の列**で報告する。既存BS-07（Required集合を候補の`required=[]`から自己確定しない）の**関連観測点**であり、BS-07の全要件・Required独立Closed World判定をこの局所試験のPASSで済ませない。

この試験に管理Root**内**の`*.tei.md`の「成功」を追加しない。現mainでは未対応新形式が黙って未読となり得るが、[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)の暫定`TEI_MD_PARSER_UNBOUND`はそのケースを明示拒否する。両方で同じSourceを正常に解析できたと見なすと、移行の欠落をテストが固定してしまう。正式な`*.tei.md`の入場／Loader実読込／Parser／Source span／Meaning／Buildは[main統合済み計画 #3030](https://github.com/hat47x/tei-platform/pull/3030)のTM検収がOwnerであり、次の同一HEAD統合試験まで未実証。



**未読込Markdownの内容変更対照**：同じWorkspace内で`semantic/README.md`の内容だけを書き換え、再`workspace.Load`と`BuildSourceIndexFromLoad`を行う。独立Expectedは、`sourceindex.ModelRevision`は**変わる**（管理Artifactのbyte変化）一方、`World`と`Index.References`は**変わらず**、Markdownの`ReadByLoader`もfalseを保持する、というもの。これにより「Source変更を見落とす誤実装」と「管理文書の変更を勝手にTEI意味変更へ昇格させる誤実装」を同じ一因子対照で判別する。ただし、`ModelRevision`の変化からOwnerが受理済みの意味を自動失効させるか、どのBuild Artifactを再生成すべきかは、この局所試験では判定しない（#2823/#2514/#1251 Owner）。

**資格上限**：#3078のmain統合と専用CI成功はSource Read証拠の機能を示す。本PRに追加したGoテストの成功は**この新しいPR HEADのCIが完了した場合に限る**。ProductとしてのK0〜K6通過、Owner Meaning受理、実Plugin loaded bytes、実Host/Provider Effect、BS-08/09/13のCrash/ACK復旧を主張しない。新Reader／Registry／Attester／独立Expected正本を作らない。

 
<!-- tei-dsl-bootstrap-aba-source-physical-engine-qualification-20261107 -->
### 13.14 K1 SourceのA→B→Aと、実Plugin Target検証の「実行された範囲」

[前回実証HEAD `5eb0d1d3`](https://github.com/hat47x/tei-platform/commit/5eb0d1d311463d5b1be668454327f2a8cddd90ca)はRepository Health [#42318077501](https://github.com/hat47x/tei-platform/actions/runs/42318077501)と依存脆弱性 [#42318077571](https://github.com/hat47x/tei-platform/actions/runs/42318077571)が双方PASS。ここで証明されたのは限定Go Referenceと従来Goテストの成立であり、正式`*.tei.md` Readerや実Hostへの一気通貫資格ではない。

**今回加えたK1一因子反例**：`bootstrap_actual_read_test.go`の単一Workspace（旧`semantic/model.tei`を実読込、普通の`README.md`はInventory/Index Artifactのみ）で、対象`README.md`に対して次の対照を続ける。変化させるのは未読込文書の内容／所在だけとし、モデル用Native TEIのbyte列は固定する。

| Step | ファイル | 独立期待値 | 判定できないこと |
| --- | --- | --- | --- |
| **A：初期** | `semantic/README.md` 元本文 | Managed/Index Artifactあり、`ReadByLoader=false`。Native TEIのWorld/Subject Refは存在 | Owner Meaning承認、Builder消費、Host |
| **B：内容変更** | 同Path・別本文 | **Source `ModelRevision`はAと異なる**がWorld/Subject RefsはAと一致 | 改訂の都度どの受理・成果物を失効すべきか |
| **C：Path移動** | Bと同byte列の`semantic/notes.md` | `ModelRevision`はBとも異なる。World/Subject Refsは変化しない | Path aliasや別Workspace Scopeの認可 |
| **A'：元位置・元bytesへ復帰** | `semantic/README.md` 初期本文 | content-addressed `ModelRevision(A') = ModelRevision(A)`、World/Refsも一致 | **過去のOwner受理・Trust・Host Grantの自動復活** |

この対照は、byte hashやPath fingerprintの**値の一致**と、介在したB/C変更の**履歴・再受理判断**を分離する。BS-14のA→B→A失効反例の「Source Revision投影」の局所実証として利用するが、初回SeedのOwner受理そのものは試験に存在しない。A'で指紋が戻る性質を利用して古いOwner意思を自動的に再使用する実装は未適格とする（#988/#2600/#2514）。Source Indexに通常MarkdownがあることとTEI Meaningが受理されることも別である。

**実Build/実Runtimeの既存証拠の到達点を精密化**：現mainには次の先行試験があるため、「物理Target bytesの照合は一切ない」と一般化しない。

| 既存試験／実行入口 | 直接の確認内容 | 限界と条件 |
| --- | --- | --- |
| `internal/plugin/runtime/builtin_registration_test.go`／`engine_replay_boundary_test.go` | 選択Runtime Planに対するBuiltin RegistrationのBuild/Realization/Engine Lock不一致を拒否。変更した登録はInvoke前に停止し、Accepted Instanceを用いる局所ケースあり | Reference BuiltinはOS上の実Processの実体証拠ではない |
| `internal/plugin/runtime/engine_actual_binary_lock_test.go` | NativeとWasmの**実Artifact bytes**を再Hashし、別Target Lock、入替・改ざんを拒否する限定試験 | `TEI_TARGET_BIND_NATIVE`／`TEI_TARGET_BIND_WASM`等の4環境変数を指定する専用CIでのみ実行。一般`go test ./...`ではSkipする |
| `internal/plugin/runtime/engine_lock_receipt_replay_test.go` | Evidence ReceiptのNative/Wasm実ファイル再読込、Source realization、BuildPlan/Runtime Engine Lock、改ざん・Target入替の再検査 | 専用[tei-runtime-engine-evidence-replay](https://github.com/hat47x/tei-platform/blob/main/.github/workflows/tei-runtime-engine-evidence-replay.yml)の実Artifact生成・環境変数が必要。Receipt再生≠現在Host Grant/業務Effect |
| `internal/workspace/plugin_native_implementation_reference_test.go` | 旧`implementation.tei`とDefinition SourceをHash→Materialize→Build Lock→参照Builtin Invoke→`PluginInvocation`を接続 | 新`*.tei.md`とDSL K2/K4のSourceとは別。固定Builder/Engine Digestラベルを実OS Image測定値と混同しない |

[tei-transform-wasm-conformance](https://github.com/hat47x/tei-platform/blob/main/.github/workflows/tei-transform-wasm-conformance.yml)／Evidence Replayのworkflowは**変更対象ファイルPath限定**または手動起動であり、Draft #3028のCI成功がそれらの実Target試験の**今回のHEADでの実行**を証明するわけではない。別Headの過去Evidenceは当該Head/Targetに限定し、勝手に再資格化しない。

**次に本当に残るJOIN**：`Selected Source/Revision`（Root・Intentと`ReadByLoader`）→`Selected Language+Operator Contract`（K2/K4）→`Actual BuildPlan Sources`→`Physical Target Engine`→`PluginInvocation`を、同じSourceRevision/Target/Attempt・独立期待値で接続すること。既存Runtime実証を再作成するのではなく、**Bを選んだのにAをビルド・呼び出す（返値は等しい）**誤接合のSource→Selection列に実証の不足がある。必要なBuild/Target差分は#1251/#2885/#973、Authoring Sourceは#3030/#2823、消費者は#1016、Owner認可/Receiptは#1174。欠落がある場合は`not-proven`で停止。新Registry/Host/Proof Store/新独立Expected正本は作らない。

**SHO/Studioとの分離**：[#1824](https://github.com/hat47x/tei-platform/issues/1824)で検討するRaw Markdown Writerは、`Workspace.Load`の意味受理の失敗と独立して設計する。Source Path/Raw bytesを扱うだけのDraft編集は可能でも、保存だけでK0 Seed、Semantic Admission、Build資格を付与しない。`*.tei.md`の意味Parser未Bindingを解除するのは#3030のParser/Loader側の検収による。


<!-- tei-dsl-bootstrap-bs14-current-host-preflight-closure-evidence-20261107 -->
### 13.15 BS-14の現在Host受理を実Sourceへ接続し、Build成功／閉包不適格を分離する

**状態更新**：移行仕様[PR #3030](https://github.com/hat47x/tei-platform/pull/3030)は**2026-11-07にmainへマージ済み（docs-only）**。これは`*.tei.md`標準採用とM0〜M6/TM-01〜14受入設計の格納が確定したことを示すが、実Markdown Reader/Grammar/Legacy Source移行/正式Owner受理の実装ではない。未対応Sourceの暫定`TEI_MD_PARSER_UNBOUND`は[#3031](https://github.com/hat47x/tei-platform/pull/3031)の**未統合Draft**にとどまる。

#### BS-14: ModelRevisionが戻っても現在のHost Readerを必ず再照合

現mainには、専用の`internal/sourceacceptance/workspace.go: LoadVerifiedWorkspace`がある。これは**同じ`workspace.Load`に由来するSource Index**を`BuildSourceIndexFromLoad`で組み立て、Host側の`Reader.ReadCurrentAcceptedSource`が返す現在のRecordと照合する。Readerが無い・拒否・改訂不一致なら`VerifiedWorkspace`を返さず、Source列挙だけからRuntime資格を出さない。現在の`Record.AcceptanceKind`は`provisional/unspecified`のみであり、正式な人間の意味受理を実装したとみなさない。Trusted Readerの真正性・撤回履歴はHost側の責務であり、受理済みRecordをテスト自身が作れば本物になるという仕組みではない。

そこで本Draftに`bootstrap_current_acceptance_test.go: TestBootstrapBS14ABARequiresCurrentHostRead`を追加。**既存のBS-14独立Expected** `review-current-qualification`を先に読み、同じ実Workspace/元SourceのA→B→A'を試す。

| 段階 | 実入力／Hostの現在状態 | Go APIで期待する結果 | 誤昇格を防ぐもの |
| --- | --- | --- | --- |
| A | Native `semantic/model.tei`＋通常`README.md`、SourceIndex A、**合成**Host ReaderがAの暫定Recordを返す | `LoadVerifiedWorkspace`成功（Source受理の**合成正例**に限定） | 「何でも拒否」テストを許さない |
| B | 未読込`README.md`のbytesだけ変更、ReaderはAを返す | `SOURCE_FINGERPRINT_MISMATCH`、VerifiedWorkspaceはゼロ値。Canonical WorldとSubject RefsはAのまま | Meaningが変わらなければSource閉包は照合不要、という誤り |
| A' | 元のbytesに復元し`ModelRevision(A')=A`、しかしHost Readerが現在撤回を返す | `SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE`、World非公開、Hostの私的エラー文字列を外へ出さない | ハッシュ再一致で旧Owner受理が自動復活する誤り |

このテストは実Source Readerと既存`sourceacceptance`の現在Reader IFを通すが、**Readerは試験用合成実装**。人間の正式な受理、署名の検証、revocation ledger、Current Host Grant、DB Effect/Receiptは存在しない。BS-14の「現在性確認経路」の限定証拠だけであり、Bootstrap K0→K6全体PASSではない。既存の`internal/sourceacceptance/workspace_test.go`が検証するR1→R2/非解釈文書改版/Reader失効を再実装せず、**一つのBS-14 A→B→Aと独立Expectedを結合するRegression**として位置づける。

#### Build: Targetの実観測が空ならUNKNOWN、成果物があってもSource閉包が拒否なら未適格

- 現mainの[#3091](https://github.com/hat47x/tei-platform/pull/3091)はGo選択Moduleのオフライン事前分類を追加。[Draft #3130](https://github.com/hat47x/tei-platform/pull/3130)は、`go list -deps -json -e`に**対象Module rootが0件**（空stream/標準ライブラリのみ）のとき、`supported`を返さず`unknown`とする修正を限定CIで確認。**Draft・未マージ**であることに注意し、このテスト結果を現mainの実装完了として主張しない。
- [main統合 #3144](https://github.com/hat47x/tei-platform/pull/3144)は供給の破損を`corrupt-supply`として`missing-module`から分離した。ただし`go list`だけでは展開済みModuleの改変を見落とし得て、`go mod verify`等の追加検査が必要。分類語があるだけで供給の真正性を保証しない。
- [main統合 #3156](https://github.com/hat47x/tei-platform/pull/3156)は隔離環境で`cmd/tei`の**実Build成功・artifact digestあり**と、同adapterの`ResolveSourceClosure`は`rejected`であることを独立に記録。理由には実Build側の供給とClosure Resolver側の外部Module受容方針の差がある。**Build artifactの生成成功と、認定されたSource閉包は別Claim**であり、片方を自動的にもう片方のPASS/FAILへ置き換えない。二回Build再現性、Providerの選択・実リンク、受理済みArtifact、正式Host Grantは別。
- [main統合 #3131](https://github.com/hat47x/tei-platform/pull/3131)のPostgreSQL Receipt Lookupは**receipt行の永続観測**に限定。Receiptが`observed`でも資源の最新値の独立再読取、選択Source/Providerの資格、Ownerが期待する業務効果を保証しない。

**K1→K6の新たな検収単位を作らず、既存の独立列を接合する**：Required Source→実Read→Grammar/Role/Operator→選択Target Moduleのroot数・供給改ざん検査→実BuilderのSource/Module閉包→Artifact Digest/実Binary→Invocation Identity→Hostの現在Record/Grant→Receipt/独立Read。入口で観測0件は`UNKNOWN`、実Build成果物は`artifactProduced`、既存Closureが拒否していれば`closure=unqualified`と両方残し、単純な緑色合格へ圧縮しない。BS-07/14および#2831/#1251/#951/#1174/#2514のOwnerを維持し、新Status Enum/Registry/Builder/Attester/Receipt Store/Acceptance Engineは増設しない。


<!-- tei-dsl-bootstrap-dogfood-context-test-ability-handoff-20261107 -->
### 13.16 実CLIドッグフーディングのT0をBootstrapのK1～K4へ結び、設計Contextの「未知」を保存する

2026-11-07の[main統合 #3264](https://github.com/hat47x/tei-platform/pull/3264)と[main統合 #3266](https://github.com/hat47x/tei-platform/pull/3266)は、Bootstrapとは別の**実プロセス**の制約を明らかにした。[限定T0 main push実CI #42359610226](https://github.com/hat47x/tei-platform/actions/runs/42359610226)では、既存CLI read-only 9種、Product Skill静的14件、固定Scenario 6 Step、実CLIへの故障注入4件＋独立検証基準の負例2件を扱う。#3268はScenario/Source等の変更で専用Workflowが発火するpath対象を拡張した。これは**既存Scenarioを実行・既知の誤りを区別するT0**であって、AIが新たな設計Context全体を集め、新しいScenarioを提案・生成したT1ではない。局所statement coverage C0相当10.1%、C1未計測はBS-01～15のカバレッジでもない。

本Bootstrapの検収へ引き渡す、**新規IDや別Query Schemaを増やさない**5つのContract条件：

| 既存責務 | 実Consumerに渡すべき情報／独立期待値 | 主張を止める条件 |
| --- | --- | --- |
| **BS-02/07、K0/K1** | `Intent/Required Source`（対象Scopeと必要集合の出典）、`Managed`／`ReadByLoader`／`Parsed`／`Subject Reference`を別欄。同じ固定Workspaceでも必要なLanguage/Operator/Roleの未観測を列挙。 | `validate.ok=true`、9 CLIの取得完了、`SourceIndex`へのArtifact掲載、`readByLoader=true`のみでは`Required complete`としない。 |
| **BS-03/06、K2/K4** | Source byte/occurrence/span、exact Language Contract+Implementation Pin、exact Operator Contract、Candidate／Owner Accepted Meaningを別Claimで返す。 | 同じ`semantic-data:sales.order`が読めても、Function/Business/Rule/Operatorの非取得を`absent`へ昇格しない。 |
| **BS-10/15、K0→K4** | CLI/Skillが返すJSONの実schemaVersion/mode/successShapeとTool受領（page/terminal marker）、Source/Grant改訂、未受理の理由を照合。 | 途中切断、旧版ページの混合、未取得のRole、初回Grammar自己解釈、AI生成Expectedの自己追認があれば`not-proven`。 |
| **BS-07/10、K4→Test** | Design Contextが**独立に根拠付けた**Required閉包へ到達した部分だけ、既存Scenario SSOT（#1180）と独立Expected（#2514）を参照。T0は既存Case、T1は未カバーRuleに対する**新規Case**。 | T0の6 Step PASSや既知故障注入のkillを、未知Dependencyの完備性、新たなCase生成能力、C1/全NFRへ昇格しない。 |
| **K5/K6、Self-Build** | Target選択、Providerの実Build-consumed Source、Module supply/closure資格、Artifact byte、実Target Image/Invocation、Hostのcurrent Record/Grantを**各々同一改訂・Attempt**でJOIN。 | #3156のようにArtifactが生成されてもSource Closureが`rejected`なら完成扱いしない。 |

**最小の次の縦切り**：[#3262](https://github.com/hat47x/tei-platform/issues/3262)のP-1bで「1 Subject/同一SourceRevisionのDesign Context」だけを読み、既存BS-07の`candidateDeclaredRequired=[]`と、独立`independentRequired=[operator/sum-v1]`に対応する**必要集合の出典差**をPacketに残す。正例は必要なSource/Contractの実Readと独立Expected照合が揃ったときのみ限定`complete-in-scope`、反例はそれが無ければ`unknown/incomplete`。Bootstrap Fixtureの15件・Existing Scenario・Test Ability Probeを複製せず、#3262のContext Consumer/#3223のCoverage推薦を経由してT1（[Draft #3273](https://github.com/hat47x/tei-platform/pull/3273) TG-01～06）へ接続する。製品共通Design Context Query/CLI/MCP、Harness H2/H3、AIによる新Case生成、正式Owner意味受理、実Plugin Host、T1は**まだNOT RUN**。

**保守上の実反例（同日CI、追跡先を訂正）**：[Repository Health #42359595347](https://github.com/hat47x/tei-platform/actions/runs/42359595347)では設計索引漏れが発覚。追加の[Run #42370709959](https://github.com/hat47x/tei-platform/actions/runs/42370709959)では、WorkBench/HTTP Decision/Plugin-firstの索引3件とRQ-035の受入PAS・Gap Audit未登録が残ることを確認した。暫定修正[#3290](https://github.com/hat47x/tei-platform/pull/3290)は、さらに新しい統合候補[#3297](https://github.com/hat47x/tei-platform/pull/3297)が4件の索引・RQ-035=Workbench・RQ-036=BOMの二表を包含するため**重複・PAS割当競合を避けて未マージでclosed**。RQ-035のPrimary PASは#3270/#3297のPAS-02を担当判断の基線とし、#3290途中案のPAS-03主は採用しない。全体Healthの回復は#3297の**exact HEADの完了CI**とmain側反映を確認してから主張する。T0専用Workflow合格を全Repository Healthに代用しない。


<!-- tei-dsl-bootstrap-actual-context-manifest-current-function-20261107 -->
### 13.17 BS-07のRequired閉包とManifest未束縛を、実Contextの限定資格で判定する

[main統合 #3284](https://github.com/hat47x/tei-platform/pull/3284)の`context_gate.py`は、既存read-only CLI 9応答・実Source `semantic/sales/order.tei`・`semantic-data:sales.order`・World/SourceIndex fingerprintを照合し、7個の欠落/改訂負例を拒否する。専用[CI #42370519406](https://github.com/hat47x/tei-platform/actions/runs/42370519406)の限定試行では`boundedReference=partial`、`testExecutionAdmission=exploration-only`、`qualifiedTestGeneration=false`が維持された。これは**Sourceの一部が見える**証拠であって、BS-07 `independentRequired=[operator/sum-v1]`の独立必要集合・意味全体・現在の許可を閉じる証拠ではない。

同時に、現main `internal/sourceacceptance/workspace.go`は`LoadVerifiedWorkspace`のSource Indexを**実Load時点のbyte列**で構築して現在のHost Readerと照合するが、Manifest `tei.workspace.yaml`のDigestは`VerifiedWorkspace.Manifest`として**別欄に返すだけ**で、Accepted Source IndexのArtifact集合には含めない。この仕様は既存`internal/sourceacceptance/workspace_test.go: TestLoadVerifiedWorkspaceExposesLoadTimeManifestAndDoesNotBindIt`で、ManifestがLoad後に変わっても旧LoadのSource Index照合が成立することとして実テスト済み。したがって`LoadVerifiedWorkspace`成功だけでは、**どのRootとImportが必須であるべきだったかの独立判断までHostが受理した**とはいえない。BS-07/15を閉じるには、Source Indexと並行してManifestの実raw digest、Required Root/Source集合の外部根拠、Manifest Reader資格、実Readerで消費したbytes、Source/Host改訂の取り違えを同一試行で観測する必要がある。

**試験の優先順位**：初回は宣言済みRootの内容と`readByLoader`、独立Required集合の差だけを1 Subjectで比較し、(1) Manifestだけを書き換えて同じSource Index／Worldを得てもRequired Rootが変われば閉包`unknown`、(2) 新形式`*.tei.md`を必須としたが実Parser未束縛なら`unparsed`を成功にしない、(3) Function/RuleがSource Indexに出てもBuild/Entry/Runtimeで実消費されないときは`not-proven`にする。既存の試験済みManifest境界を別のHost実装へコピーせず、[#3262](https://github.com/hat47x/tei-platform/issues/3262)のDesign Context P-1b・[#3283](https://github.com/hat47x/tei-platform/pull/3283) Function実在→Build/Entry・[#3277](https://github.com/hat47x/tei-platform/pull/3277) Plugin-first Inspector D1へ観測を引き渡す。**上記(1)〜(3)を単一Product Attemptで接続する新試験はNOT RUN**。

**BOM／数の別問題**：[Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)はRQ-035（Workbench）を維持しRQ-036（Verified BOM）を別要求にした。複数版Matrixの列挙・参照Fixture内のWitness一致と、対象のVersion候補の**全母集団**・実Provider Load／実Consumer利用・現在Host Grant／Trusted Checkを分ける。Bootstrapの独立Required集合の不足をBOM候補数／Pairwiseの行数で埋めたと扱わない。実Plugin選択・同一Build/Host資格は#951/#1251、正式Source/Meaning受理は既存Ownerに残す。

**資格のまとめ**：#3028の前回HEAD `fdc2bf1f`はDependency CI成功、Repository Health [#42370820147](https://github.com/hat47x/tei-platform/actions/runs/42370820147)は**failure**。その実Goログの失敗は`repository-plan-check`のRQ-035二表・設計索引4件であり、今回のBootstrap Goテストの直接失敗を示すものではない。一方、`go test ./...`総合はFAILであることに変わりはない。最新HEADや別DraftのCI成功は都度区別する。
