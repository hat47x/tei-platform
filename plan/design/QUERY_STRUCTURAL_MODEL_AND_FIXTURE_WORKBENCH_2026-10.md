# SQL構造モデル・テストデータ付きQuery Workbench：最小実証計画（提案）

- 作成：2026-10-30
- 状態：**設計・検証計画候補**。公開契約の採用、実装完了、SQL/DB間の意味同等性、Ownerの業務受理を表さない。
- 基準：2026-10-30確認時のmain `2503a25fee9ffa47592d47818c2bb35c0a2c50ab`。実装着手時は再照合。
- Query構造Explainの一次Owner：**#2387**（M0静的解析確認→M1 read-only投影→M2 fixture照合→M3 Modeler/SHOへの出力）。本書/PR #2388はその追加実証計画であり、重複する製品機能Ownerではない。
- 主な担当：#1640（集合演算/評価と計画）、#1126（データモデル）、#1131（型付きDataset/Snapshot/Expectation）、#1124（Modelerの投影）、#1803（設計書導出）。
- 関連：#988（SQL/DSLのOperator）、#1706（集合束縛・重複）、#1121（Lowering）、#1063（設計時/実行時のDSL）、#1180（保存済み業務Scenario）、#1807（外部データ取込）。新しい第二Semantic正本や第二Driverを設けない。

## 1. 検証する製品価値

**SQLをDB製品から独立したDSLとして構造的に解釈し、業務エンティティからCTE・サブクエリ等の仮想関係を経て結果集合へ至る過程を、人と生成AIが直接読める設計成果として示す。** その各段階に局所/共有のテストデータを束縛し、入力・中間・最終結果と差分を検査可能にする。優先順位は **設計書の可読性 → 意味の検証可能性 → 実DBへの実現** とする。

これを全SQLの移植性保証、SQLの最適化器、正式なテストデータ生成言語、業務結果のOwner受理として扱わない。三視点（Data / Function / Business）に関係する同じ対象の投影であり、SQLだけの独立Modelerを設けない。元のデータモデル表記に基づく横長属性・関連線・高密度表現の可読性を重視するが、許諾未確認の図面や素材を転載せず、既存形式の完全複製を必須としない。

## 2. 既存の足場と現在の不足（静的読解）

- `internal/query/derived.go` の `DerivedIR` はQuery scope（root/cte/subquery）、Source occurrence、logical/physical field、出力とfield lineage、解析coverageを保持する。CTEのSource bindingを物理表と区別するテストが `internal/query/derived_test.go` に存在。
- `internal/query/relational.go` は source/join/filter/aggregate/project/distinct/sort/limit/opaque の関係ノードを提供。これをWorkBenchの**派生投影入力**として使う。
- `internal/query/derived_select.go` の現在のtarget-neutral `DerivedSelect` は意図的に狭い。Sourceが一表の単純射影以外（CTE、JOIN、計算式等）の場合は拒否する。**現状でCTE中間結果を評価可能とは言えない**。#1640の取得前後の集合範囲・安全性を維持する。
- `internal/driver/queryresult/mapping.go` はraw列からSemantic attributeへの明示mappingを保持する。CTEの一時出力や行level lineageを、この対応が自動確定した意味属性にしない。
- `tools/reference-design-output/` は受理済みWorld由来の一部文書とmanifestを導出するが、今回のQuery WorkbenchやSHO編集・保存を実装済みとはしない。
- `docs/status.md`（2026-10-29更新）は統合Modelerと汎用実行基盤を未完成としている。本計画のfixture・表示・evaluatorの試験は未実行。

## 3. 一つの問い合わせの三層構造

1. **業務データ層**：受注/受注明細/顧客等のAccepted Entity、Business identity、候補キー、FD、Relation、意味型・値域制約（NULL・精度・列挙等）、Constraint。#1126が所有。Domainという語を単一値の値域の意味で使わない。
2. **問い合わせ構造層**：著者が保持するnative SQL → analyzerのScope/RelationalIR/Field lineage → `QueryVirtualRelation`相当の**表示用概念**。各CTE/derived tableをscope-localに投影するが、新Core kindとして先行採用しない。CTEは物理的な一時表を意味せず、CTE名も業務エンティティIDではない。
3. **観測・説明層**：fixtureの入力行、中間結果、期待結果、実行結果、差分、未解析/未評価/失敗診断、SHO設計書用の図表。結果観測は意味正本ではない。

### 仮想関係のカード（表示例の意味契約）

- identity：Query ID + SQL digest + lexical scope path + projection/view revision。Schema/Business CanonicalIDへ自動昇格させない。
- interface：列順、alias、derived/physical field refs、型・NULL性（解決済み/未解決）、集合の種類、観測行数、candidate keyとその根拠。
- join・aggregate・filter等の演算参照と、入力/出力の列対応（column lineage）を保持。
- **三つの異なる線**：業務Relation（親子・参照等）／問い合わせの行集合変換／field lineage。描画を変えて識別する。
- 行lineageは実行時の寄与証拠（行ID、入力fixture版、演算ID、評価条件）を持つときだけ表示する。集計出力のfield lineageが存在することを、その集計へ実際に寄与した行の証明としない。
- key/cardinalityのラベルを **declared / statically derived / fixture-observed / unknown** に分ける。テストデータ内で一意でも全入力に対するFDの証明としない。
- SQLは基本的に重複行を保持する（bag）ので、関係図上の行集合を暗黙に集合化しない。DISTINCT、UNION ALL、重複を生むJOIN、ROW_NUMBER、GROUP BY、NULLの三値論理、並べ替えをモデル上に残す。

### 3.1 識別性・一意性の伝播（静的推論の候補）

問い合わせの途中段階では、行数・キー・一意性は**データモデルからそのまま継承されるとは限らない**。次の判定を最初の静的診断の候補とする。

| 演算 | 推論できること | 推論してはいけないこと |
| --- | --- | --- |
| WHERE | 入力集合に成立していた一意性は行の絞込みで失われない | 残った行を観測しただけで新しい業務キーを採用する |
| SELECT | 既に保証されたキー列を保持し、行増殖がなければその一意性を引き継ぎ得る | 名前がIDに見えるだけでキーを決める |
| JOIN/LEFT JOIN | 結合の相手側が条件上高々1件と証明できれば左行の重複増加がない | 外部キーがあるだけで結合先の一意性を推測する |
| GROUP BY | GROUPの項目組はSQLのグループ同値性では一意 | NULL許容のGROUP項目組をそのまま業務識別子にする |
| DISTINCT | 出力**列の組全体**は重複除去される | そこから特定の部分列が一意、または持続的な業務IDとする |
| UNION ALL | 各枝の由来と重複を保つ | 枝ごとの一意性だけで統合結果を一意とする |
| ORDER BY/LIMIT | 結果に提示順/件数上限を課す | 無指定のtie-breakerや表示行番号を業務キーとする |

重複して**値が完全に同じ二行**を、画面の値だけで一意に識別できると仮定しない。UIの行ハイライトに試験実行専用の観測IDを付けることはできるが、再実行をまたいで安定するBusiness identityではない。結合や集計のrow lineageを示す場合は入力fixtureの行IDとscope/operator IDに束縛し、追跡不能なら「未取得」と表示する。

## 4. 構造とモデル表示の実証対象

最小SQL（業務名/入力は架空の合成例）：

```sql
WITH line_totals AS (
  SELECT l.order_id, SUM(l.quantity * l.unit_price) AS amount
  FROM order_lines l
  GROUP BY l.order_id
),
open_orders AS (
  SELECT o.order_id, o.customer_id,
         CASE WHEN t.order_id IS NULL THEN 0 ELSE t.amount END AS amount
  FROM orders o
  LEFT JOIN line_totals t ON o.order_id = t.order_id
  WHERE o.status = 'OPEN'
)
SELECT q.order_id, c.customer_name, q.amount
FROM open_orders q JOIN customers c ON q.customer_id = c.customer_id
WHERE q.amount >= :threshold
ORDER BY q.order_id
```

各段階の表示：

```text
[orders] ──[親子Relation]──> [order_lines]     [customers]
                                     │ SUM/GROUP
                                     ▼
                              <line_totals>
                                     │
[orders] ───────── LEFT JOIN ──────────┘
   │ WHERE status=OPEN
   ▼
<open_orders> ── JOIN customers ── WHERE amount>=:threshold
   └───────────────> <final result>
```

上記は**説明用の配置**であり、SQLの実行順序/物理マテリアライズを示すものではない。Modelerは「エンティティ＋属性一覧」「CTEの演算DAG」「field lineage」「fixture行と中間結果」「SQL本文のsource span」を切り替え/連動させる。SELECT列を選択すればどのscope/演算/元属性に由来するか移動できる。

### 固定fixture F0（#2387のQXケースに対する補助ケース。値・期待値は独立に指定）

- customers: (C01, 青空商事), (C02, 山田商店)
- orders: (O01,C01,OPEN), (O02,C01,OPEN), (O03,C02,OPEN), (O04,C02,CLOSED)
- order_lines: (O01,1,2,100), (O01,2,1,50), (O02,1,1,50), (O04,1,3,100)
- データ型/制約前提：orders.order_id一意、order_lines(受注ID,行番号)一意、参照先存在、**F0版ではquantity/price非NULL**。整数例は桁あふれしない。通貨/丸めなど業務採用は対象外。
- `line_totals`: (O01,250),(O02,50),(O04,300)。
- `open_orders`: (O01,C01,250),(O02,C01,50),(O03,C02,0)。
- `threshold=200`、final: (O01,青空商事,250)。
- assert：O03は明細なし→LEFT JOINで残る→CASEで0→最終WHEREで落ちる。O04は集計に存在してもOPEN絞込で落ちる。

期待値はSQL/evaluatorから逆生成せず、手書きの独立oracleとして固定。fixtureは**事前にデータモデルへの適合性を検査**する。

### 反証ケース（予定。現在未実行）

| ID | 最小変更 | 期待する区別/診断 |
| --- | --- | --- |
| N1a / N1b | F0（quantity非NULL）へNULLを入れる / **nullableを明示した別モデル版**でO02.quantity=NULLを評価する | N1aはモデル制約違反として入力拒否。N1bはO02のSUM=NULL、O03のabsent group→0を区別。意味の違う二つのDataset/Model版を同じfixtureと見なさない |
| N2 | JOIN先customersの重複キーを混入 | モデル制約違反としてfixture拒否。JOINの二重計上と誤った一意性を隠さない |
| N3 | 件数制限を集計・Filterより前へ移す候補 | 不正/非同等な計画を拒否。既存#1640の先行Limit禁止を再利用 |
| N4 | GROUP BY前に同じ受注の明細が複数存在 | 2行が1集約行へ寄与することを明示。row lineageが無ければ寄与行は未確認 |
| N5 | ORDER BYの一意なtie-breakerなしにページ分割 | 安定結果を保証しない表示。実行の偶然一致で保証済みにしない |
| N6 | 未解決のCTE alias/列、サポート外の再帰/ウィンドウ関数 | opaque/unresolvedとsource span。成功したかのように空結果を返さない |
| N7 | 局所fixtureが共有fixtureと同じキーの異なる値を供給 | 合成衝突を表示して明示拒否/人の選択へ戻す。last-wins不可 |
| N8 | SQL digestは同じでmodel/dataset/rule revisionが変化 | staleを検出。旧結果を現在の意味評価として表示しない |
| N9 | Providerが対象集合の一部しか返さない | observed subsetとcomplete inputを区別。全件集合述語やCOUNTの証明に使わない |
| N10 | 参照元の読取り権限がなく、集約結果だけ許可される | trusted providerの権限分離に従い、入力行の詳細lineageを漏らさず開示能力を分離 |
| N11 | 相関EXISTS/NOT EXISTS | scalar/boolean条件・相関bindingとして表示し、普通の内部JOINとして行数を増やさない |
| N12 | 生成データの制約が矛盾/未対応 | unsatisfiable と unsupported と evaluation error を別分類 |

N1は原fixtureのO02の行を**差替え**、O03は明細ゼロのままにする（同一テスト内の衝突を作らない）。N2はキー制約検査前に問い合わせ実行しない。

### 4.1 最小SQL観測とその限界（2026-10-30追加）

[再現用SQLite試料](../../plan/fixtures/query-workbench-f0/verify_sqlite.py)は**独立のSQL実行観測**に限定する。使い捨てのin-memory SQLite、テスト用架空データ、手書き期待値のみ使用。外部DB、TEI評価器、Host/Provider、Productionの認可には接続しない。単に「期待値が一度一致した」ことをSQLの意味証明へ昇格させない。

```sh
python3 plan/fixtures/query-workbench-f0/verify_sqlite.py
```

作業コンテナで同等の初期試料を手動実行した結果（**GitHub上の保存ファイルのexact-byte検証、TEIの実行試験、CIではない**）：F0のline_totals=[(O01,250),(O02,50),(O04,300)]、open_orders=[(O01,C01,250),(O02,C01,50),(O03,C02,0)]、final=[(O01,青空商事,250)]。N1a非NULL制約違反で拒否、N1b nullable別モデルではO02の集計がNULL、N2の顧客キー重複を拒否、N3の早すぎるLimitは空結果となりF0と非同等。CI/Goテスト/実PostgreSQL/Modeler操作は未実行。SQLite内部の検証が成功しても、中間CTEを元SQL実行時の物理的な作業表として取得した事実にはならない。

**重要な追加境界**：通常のCTEに対し`WITH ... SELECT * FROM cte`を別SQLとして発行して得る値は、元のQueryと**別の実行**の観測である。純粋演算・同じ入力snapshot等の条件を固定した説明用の結果として扱う。時刻・乱数・volatile関数、再帰CTE、DMLを含むWITH、複数参照CTE、評価停止条件、環境/権限差があれば、別実行の中間結果を元SQLのactual中間結果と断定できない。PostgreSQLのCTEはinline/実体化等の選択が可能（https://www.postgresql.org/docs/current/queries-with.html）。Explain/Analyzeの物理的な推定・計測行数は、論理的な全中間行の値とは別の観測である。

## 5. テストデータはモデルに束縛された資産

- #1131の型付き`Dataset / Snapshot / Expectation`を所有元とし、問い合わせ局所のfixtureは独立した業務Scenario正本ではなく**Queryテスト用の局所入力と結果期待**。
- 共有Datasetを選択し、必要なEntityの行をscopeへ投影。上書きは暗黙で行わず、差分と衝突を先に検証する。組合せにはDataset revision、局所patch digest、選択基準、パラメータ、意味型・値域制約/Constraint revision、評価時計とlocaleを固定する。
- Modeler編集UIは、キー・親子・参照・サブタイプ・意味型・値域制約・NULL/欠損の区別を表示。入力値の修正はDataset側に戻し、CTE中間結果の表を直接編集しても元のデータモデルへ反映しない。
- CTE局所テストで仮想出力に直接行を与える場合、「下流scopeのみを検証」「上流未検証」を明示。全問い合わせ試験と同じ資格にしない。
- 実DBからデータ取込する際は#1807のschema/key/権限/個人情報・来歴検査を通す。外部の実データをそのまま公開fixtureへ混ぜない。
- Model/fixture/SQLの三つのrevisionは独立。テスト結果は`input snapshot + operator/SQL digest + evaluator/target + execution context + expected + observed + unresolved`へ追跡できる形で既存Evidence境界に紐付け、新しい一律Registryを作らない。

### 制約と生成（段階候補）

M1では既存の意味型・値域制約（型・範囲・列挙値）とRelation（キー・参照）の決定的な合成入力、境界値、負例を優先。M2以降にConstraint-based generation（組合せ制約、Coverage-guided generation、失敗時の縮小）を追加。Lean/SMTは一つの形式的記法候補であり、最初に新証明基盤は採用しない。任意のLean述語からの完全な生成/判定は要求しない。

生成の四つの状態を分ける：`satisfying candidate`（検査できた範囲で制約充足）、`unsatisfiable`（確認できた矛盾）、`unsupported/timeout`、`unverified`。生成できたこと、Queryの独立oracleが一致したこと、Ownerが業務意味を受理したことは非同等。意味のあるNULL/欠損/空集合や多重度はOwner未決なら仮定を明記する。

## 6. 解析・評価・実行の責務と停止条件

- **解析**：native SQLは正本、Scope/IRは導出。クエリ名やSQL aliasから業務意味を推測して昇格しない。構文/解決不完全なら各要素にcoverageを表示。
- **意味評価器**：主要業務のRead-only関係演算を外部Hostなしで検証する候補。現在の`DerivedSelect`はCTEを拒否するため、そのまま実装済みとして接続しない。対応演算と意味同値を独立fixtureで限定実証する。
- **SQL Provider**：必要な権限と同版snapshotがある場合にsandbox/使い捨てDBで評価。テストのために元SQLを切り出して中間結果を得る場合、書換え後の問合せと元の意味同値を確認できない限り**説明用の派生評価**と表示し、正式結果と同一視しない。
- **実行計画**：#1640のFilter/Derive/Aggregate/Sort/Limitの依存順を守る。全行に対する述語は、Providerが返却した集合Rへの`checkAll(R)`だけで、業務対象集合Lの性質`P(L)`を証明しない。`R=L`を検証する入力同定・Provider完全性・同版鮮度は別の証拠。
- **副作用**：Queryの局所試験はread-only。業務Actionの変更前後、Commit、Receipt、Replayは#1129/#1174/#1180の別経路で検証し、表示上の成功を実際の業務効果へ読み替えない。
- **開示**：権限のないフィールドや中間行を表示しない。許可された集約結果から保護対象の元行を逆表示することを禁止する。
- **停止**：NULL/文字照合/丸め/時点/重複/相関/Recursive等の意味が実現先で一致する根拠がない場合、unsupported/unresolvedを示しsilent fallbackしない。

## 6.1 直交するScopeと説明の可否

他レーン（#2386組織/時点、#1769行スコープ、#1706集合完全性、#2056入力NULL）との接続では**一つのScope enumに統合しない**。少なくとも四つの独立軸を使う。

| 軸 | 主な問い | 誤って同一視しない対象 |
| --- | --- | --- |
| Lexical SQL Scope | このCTE/alias/相関列はどこで見えるか | Tenant/業務対象の許可範囲 |
| Business/Temporal Scope | どの受注/組織関係・業務有効時点が対象か | SQLの単なるWHERE文字列 |
| Read-set Scope | どの集合Rが取得され、R=Lと証明できるか、どのrevisionのsnapshotか | Providerの返したページや一部subset |
| Authority/Disclosure Scope | 誰がどの元行・属性・集約結果・Lineageを見られるか | モデルに宣言された組織上のRole |

「取得できない」「0件だった」「業務集合が空」「認可により非公開」「評価不能」「NULL」は別の状態。NOT EXISTSや全件合計での不在の確定は、対象集合と読み取りの完全性・鮮度・権限の根拠が必要。要約だけ開示可能な場合に、詳細行lineageを画面へ漏らさない。

組織例（#2386、Draft PR #2389）への将来適用では「組織の複数経路JOIN→重複計上」や「as-of組織版・Rule版だけ変更・委任失効」を別の反例とする。Query説明結果が同じでも、Rule版・組織版/Grant版が異なれば現在の判断/表示が有効とは限らない。組織用の新規SQL実行器・認可正本は導入しない。

### 6.2 説明用の評価表の最小レコード（表示向け概念、公開schema非採用）

- Queryのauthoring sourceとsource digest（native SQL、2waySQLならtemplate source digest、選択条件/適用分岐、実際に評価するSQL digestを分ける）、解析器/言語/Operatorの固定版。
- Query scope/演算IDと入力/出力のField IDs、parse/resolution/lowering/説明coverage。Opaqueや解決不能の残りを正常値へ埋めない。
- model revision、Dataset/局所patch revision、parameter presence（missing/null/present）、snapshot/read-set completeness、評価時点/時区間、Rule/意味型・値域制約 revision。
- **expected（独立oracle）**と**observed（評価器/Providerの実測）**、行/多重度の比較規則、unorderedならbagの多重度比較、ORDER BYの保証がある場合だけ順序比較。デフォルトの行番号をIdentityや安定ソート保証にしない。
- Evaluation path：説明専用/TEI意味評価/別SQLへの差替え実行/元SQLの実行/対象DB物理計画の観測を区別。元SQLとは異なるSQLへの書換えを同一実行結果として表示しない。
- Completeness/Authority/Provider/Execution failure、stale reason、証拠未取得の状態を別欄に記載。署名・byte一致・可読性評価は別資格。

### 6.3 実装順に伴う停止条件

- M0ではCTEの解析coverageを固定SQLで確認。Query側のscope-lowering coverageがFULLでも、Evaluatorの対応範囲は個別判定。
- M1ではread-only図/表だけを出す。『CTE結果』として示す手書き期待値はexpectedと明記し、Provider actualと混同しない。
- M2では既存#1131のDatasetで独立期待値・型付き検査・反例を実施。`DerivedSelect`の拒否を緩めず、必要なExecutor追加は#1640/#1121の別レビュー。
- M3でModeler/SHOの版付き表示を展開。文書内の画像・参照先・説明文の改訂とSource/fixture/Ruleの改訂を分ける。説明を読めるかは「なぜO03=0か」「O02=NULLの理由」「GROUP後の一意性が業務IDでない理由」「全件集計の完全性」を人が正しく再説明できるかで圧力試験。

## 7. 表示とSHO設計書

1. **構造ビュー**：実Entityと仮想Relationの図。Entityの親子/参照は関係線、CTE導出は演算線。ひとつのQueryを高密度表・図・ツリーに投影。
2. **値ビュー**：fixture行、演算前後の行数、行の残留/除外、金額などの値変化。行を選択すると確認できた根拠範囲だけ強調。
3. **論証ビュー**：key/FD/cardinalityの根拠、型・NULL/重複、解析coverage、期待/実際の差、未検証の前提とその依存範囲。
4. **Sourceビュー**：SQL本文のsource spanと選択したscope/field、編集できるauthoritative sourceへの戻り先。派生中間表を編集可能な正本として誤認させない。
5. **設計書ビュー**：SHO向けに入力Entity、各CTEの役割、仮想表、試料、期待・観測結果、未確認事項を配置。説明文、モデル由来の図、読み取った画像などの表示依存を分け、固定納品版と更新可能な作業版を区別。

#1803と既存`reference-design-output`のsourceRevision/structureRevision/artifact digestを再利用する。fixture revision、evaluated context、evaluator version等は**説明に必要な追加の来歴参照**であり、artifact byte一致だけで評価・意味・人のレビューの現在性を保証しない。SHO Composerの別文書正本を作らない。

### 7.1 Plugin-firstと外部Consumerへの投影（#2390と整合）

- **解析/DSL**：SQL構文・テンプレート分岐・Operator意味は#988のLanguage/Operator Pluginによる解釈候補。Pluginを組み込むことだけで受理済みMeaningを拡張しない。
- **純粋な説明出力**：既存のDerivedIR/RelationalIRと許されたモデル参照から、仮想関係、列lineage、演算依存、Coverage、source locator、許可済みfixture例を返すread-only投影。出力構造を新しい公開標準ABIへ即時固定しない。内部のGo実装やTEI Studio UIを**必須の実行依存にしない**構成を最初に比較する。
- **Consumer**：Modeler、SHO文書、CLI/静的HTML、外部Host（例：外部Editorのプラグイン）から利用できるように、構造の提示・ナビゲーション指示・画面描画を分離。外部Hostでの呼出し可能性は#2390のC0〜C3資格を参照し、Query ExplainのReferenceだけで外部利用可能と宣言しない。
- **操作と権限**：read-only Explainの契約はDB read grant、Sourceを書き換える権限、テストデータの注入権限、AI/自動テスト向けUI外部制御権限を内包しない。必要操作は既存のHost/Invocation Context、Provider binding、明示Capabilityを介す。生成アプリで外部制御プラグインを含めないBuild構成も独立に維持する。
- **AIハーネス**：最小のtyped inspect/explain/fixture-compare結果が読めれば、AIはユーザー向け可読性と根拠を確認できる。AIが承認/受理済み意味へ変換することや、GUI制御を必須の正確性証拠にすることは避ける。

最初のP0受入では、少なくとも**CLI/JSON等のUI非依存出力**が同じscope/field identityを参照できることを確認し、Modelerへの投影は次段階とする。このConsumer境界は、TEI専用Hostを再導入することを意味しない。

### 7.2 M0の先行Gate：SQLワーカーの出所、スコープ付き束縛と説明の真偽（2026-10-31追補、未実行）

[#2387のQX-12 相関サブクエリ・scalar・LATERAL・Windowの検討](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6534035206)を踏まえ、複雑なSQLの演算対応に進む前に、**現在存在する単純な列参照の解決でも誤った「完全解決」が生じないこと**を先に保証する。旧X-TEAの設計例、組織の関係経路、Smart DataのFunction Port、Value Contract/型付き項目にも共通する根拠境界である。

**コードに基づく所見（main `4325ca79a52bf0d576db6fb4027bdc516b1023fd`、実Goテスト未実行）**

- `internal/query/analyzer/client.go` は `RunPythonWorker` を備え、`internal/query/analyzer/model.go` の `ColumnOccurrence` は `ScopeID` と `Correlated/OuterScopeID` を備える。一方、`reference/workspace/plugins/tei-editor-intelligence-sql/1.0.0/plugin.tei` は外部 `python3 -m tei_sql_worker` と `sqlglot` を宣言するが、確認したmainの全追跡ファイル内に `tei_sql_worker` のPythonソースはない。配布済み外部パッケージや開発環境への導入の有無は**未確認**。同宣言を「クリーンcheckoutでSQL→wire ResultのE2Eが可能」とみなさない。
- `DeriveFromAnalysis` 内の `resolveColumn` は、物理列解決に**ScopeID/OuterScopeIDを渡さず**、平坦な `TableAliases` と全Scopeの `SourceRef` を見て先行一致する別名を採る。内側と外側で同じ別名が異なる物理表を指すと、異なる列が同じ表へ誤束縛され得る（**静的コード読取りに基づく反例候補**。実行結果の観測ではない）。
- 同じ物理表の二つのSource occurrenceを使った自己JOINでは、列の曖昧性を表名で `uniqueStrings` に潰すと、未修飾列を一意に解決したと誤認し得る。名称が同じでもSource occurrenceが二つなら同一参照ではない。
- `outputByScopeName[scope][lowercase(name)] = outputID` は、同Scopeに重複する出力別名があると最後の列で上書きし得る。CTE列参照を「一意」と断定するには実際に一意か検証が必要。またCTEの `SourceKind` が`cte`でも、`Binding` が存在するScopeIDを参照しているかを検証せず `resolved++` する箇所がある。
- 既存の `Coverage.Parse=full` は `result.Status=ok` から、`Coverage.Lowering=full` はworker側の申告文字列から導出される。これらと、**スコープ付き物理列解決・CTE参照・実SQLの利用形態まで一貫して確証されたこと**を同値扱いしない。

**M0の最小反証テスト（いずれも今回NOT RUN）**

| ID | 入力/対象 | 判別すべき事実 | 不合格／停止条件 |
| --- | --- | --- | --- |
| QX-M0-1 | クリーンなmainとPlugin宣言 | SQLワーカーの実物・配布単位・依存固定版・実行環境・protocol version・出所を特定 | descriptorだけでworker実装・SQLGlot実行をPASSにする |
| QX-M0-2 | rootで`public.orders AS x`、subqueryで`public.order_lines AS x`、両方に`order_id`列 | inner `x.order_id`→`order_lines`、outer `x.order_id`→`orders`、同じalias文字列でも別Source occurrence | 先に列挙されたorders、または平坦Alias Mapで双方を同じ表へ束縛し、`Resolution=full`と表示 |
| QX-M0-3 | `orders a JOIN orders b` 内の未修飾 `order_id` | 二つのSource occurrenceがあるため曖昧性を維持し、実SQLが拒否されるなら解析/実行エラーを保持 | 物理表名が同じことだけで曖昧性を消去 |
| QX-M0-4 | 同ScopeのCTE出力別名二重宣言、CTE Bindingの未存在ScopeID | 出力別名の一意性・Binding参照先・未知列を検査し、少なくとも未解決/拒否 | mapの後勝ちや存在しない`Binding`から完全Lineageを描く |
| QX-M0-5 | outer相関列、inner同名列、`EXISTS`/scalar/LATERAL各利用形態 | `OuterScopeID`と式の利用形態は独立。具体的利用形態が現wireで未識別なら部分/unsupported | 相関という理由だけでscalar/EXISTSを普通のJOINへ推測変換 |
| QX-M0-6 | source SQLのみ変更、Schema Snapshotのみ変更、Output ordinalのみ変更 | 同じ名前・位置でも別identity/意味の場合は再解決・stale表示。構造・Lineage・fixture・評価観測の無効化境界を区別 | 古い解決済みLineage/受理済み意味を自動継承 |

最初の作業単位は `internal/query/derived_test.go` の**synthetic `analyzer.Result` を使うQX-M0-2/3/4の負例**と、既存 `resolveColumn` の対象範囲の狭い修正を別の小Draftで検討すること。今回の計画文書はコードを変更しない。Testを実行しないうちは不具合の実測PASS/FAILとは呼ばず、原因候補として保持する。誤束縛は見た目の問題だけでなくSource/Field lineageが誤った物理表を指すため、単なる`unresolved`より危険な**false-positive resolution**として先行対処する。

次に、実SQL→ワーカー→`analyzer.Result`→`DeriveFromAnalysis`へ同じSQLを流して、静的Go fixtureと差分比較を行う。workerが利用できないなら、実物と出所を特定する段階を明示的に停止し、Goで手組みしたResultをE2E証拠へ読み替えない。Plugin配布・Host/実行許可は#988/#2390/#951を参照し、通常のQuery説明のために独自Hostを追加しない。

### 7.2.1 M0の実装修正候補とその資格（2026-10-31）

QX-M0-2/3/4への最初のGo側対策は、[Draft PR #2398](https://github.com/hat47x/tei-platform/pull/2398)に**別branch**で分離した。#2388は設計・fixtureのDraft、#2398は`internal/query/derived.go`と`derived_test.go`のみを扱うコードDraftである。双方を同じ完成判定にしない。

- コード候補：`ColumnOccurrence.ScopeID`と明示`OuterScopeID`に基づく物理・論理Source occurrence解決、自己JOINにおける出現箇所保持、CTE出力別名重複の後勝ち回避、存在しないCTE Bindingの未解決化、物理＋CTEに同名列がある場合の曖昧性維持。
- 合成Go回帰テスト（Draftへの追加定義であり**まだ実測ではない**）：同名Alias異Scope、内外の無修飾列、自己JOIN、相関した外側物理列、重複CTE出力、存在しないBinding、物理＋CTE出力衝突、相関した外側CTE列。
- Goコードの静的な読み直しとGitHub保存は完了。`go test ./internal/query -count=1`と`gofmt`、クリーンcheckoutのwire/worker E2Eは未実行。GitHubを直接cloneできない環境、Go 1.23.2とrepo指定Go 1.26.9の差による制約があるため、結果をPASSへ昇格させない。
- 特に残る問題：CTE Bindingの**可視性/名前解決**はtarget scopeの存在・kindのみでは確定しない。SELECT出力のデータ型・NULL可能性、引用識別子の方言別大文字小文字規則、外部workerの出所・runtime・SQLからの実写像も後続Gate。Go側の誤束縛防止が単独でAnalyzerの完全性やSQL実行許可を与えない。
- 合格条件：exact headのfocused Go testsが再現可能にPASSし、残る未解決/曖昧性を`Resolution/Lineage/Coverage`として説明でき、既存のCTE合成testとD2拒否が後退しないこと。次にQX-M0-1の実worker実体の特定と、実SQL→wire→IRを別の資格として実施する。

### 7.3 設計者が受入できる最小の閲覧操作

最初のread-only Consumerは、**SQL本文の一つの式を選択→対応するScope/Source occurrence/演算/出力列へ移動→逆方向に根拠へ戻る**という一往復を最小操作とする。静的解析のCoverageと評価証拠は別欄、未解決・反例は非表示にしない。CTE中間表・相関した外側行ごとの例は固定Dataset/Scenario参照から示し、テスト値を実DBの物理中間値と呼ばない。

合格条件は、同じ別名を持つ二つのScopeと同じ物理表を二度参照する自己JOINにおいて、選択先が正しいSource occurrenceか未解決であり、他Scopeへ誤Navigateしないこと。失敗した場合は投影を広げる前に束縛・source locatorの不確実性を表示する。編集・Source Acceptance・Host実行・Smart Data Hookはこの操作に含めない。

## 8. 実装順・担当引継ぎ（提案）

| 段階 | 所有 | 実施 | 完了の根拠 |
| --- | --- | --- | --- |
| P0：静的説明スライス | #1640 + #1126、投影は#1124 | 既存CTE scope/RelationalIR→仮想関係の表・三種の線。source span、未解決ラベル | F0の構造と入力→出力属性の対応。未評価を明示し、SQLを変更しない |
| P1：fixture/期待値の一巡 | #1131 + #1640 | F0、N1〜N4、独立oracle、CTE各段階の期待/実際と差分 | 正負と0件/NULL/重複を区別。既存D2の範囲を再実装しない |
| P2：Modeler/資料連結 | #1124 + #1803 | SQL ↔ モデル ↔ 行 ↔ 設計書の一往復。編集可能元と導出部分を区別 | 同じrevisionを辿れ、変更後はstale表示。既存manifest/文書出力を活用 |
| P3：生成制約と別Target比較 | #1131 + #988/#1121、#1180はScenario再利用 | 意味型・値域制約/Constraintに基づく生成、最小の第二eval/実DB比較 | 生成の適用範囲と差異を説明できる。Lean/SMT、全SQL、各DBは先行決定しない |

#2387が静的read-only説明の一次Owner。#2387のQX初期fixture（数量0・顧客C1/C2/C3等）を最初の構造試験に使い、本書F0（数量が非NULL、二CTE・thresholdでFilter）はP1で演算・NULL比較を明確にする**補助fixture**とする。二つの独立Dataset正本を作らず#1131の同じ型付き契約へ接続する。#2386のORG-13（対象集合完全性）・ORG-21（複数経路計上）はその次の拡張例とする。P0/P1の固定fixture・紙上oracleは並行可能。P2はP0のID/出力を受け取る。P3はP1で見つかった実不足をもとに範囲決定。既存Issue/PRに所有機能があればそこへ差分を寄せ、新しいWorkBenchモノリスを作らない。

## 9. 将来の形式検証・行の由来を扱う研究参照（採用候補）

SQLの各結果行について「どの入力行が、どの演算によってこの出力を支えるか」を一般化する際、**row provenance**（由来情報）の理論が有効。Provenance semiringでは、単純な入力行のリスト以上に、複数の導出経路・重複・JOIN・集計について何を示せるかを選べる。設計ビューでは次の三種を区別する。

- **which**：寄与した入力行の和集合（軽量。ただし導出方法は失う）。
- **why**：結果を成立させる入力行の組合せ（証拠の候補を示せる）。
- **how**：導出経路と多重度を表す代数的表現（コストが高い）。「寄与の証拠」と「業務上の因果・正当性」は非同等。

公開の **provenance-lean v1.2.0**（2026-09-22、Pierre Senellart、DOI: https://doi.org/10.5281/zenodo.22900284、https://github.com/PierreSenellart/provenance-lean）は、Lean 4でsemiring由来情報、差集合、集計、HAVINGを含むモデルを形式化する。**TEI向け実装・テストデータ生成器・全SQL意味の正本ではない**ため、P3以降の比較対象とする。特に集計後の由来と、負例としてのNOT EXISTS/差集合を単純な行追跡で同一視しない。

Apache Calciteの関係代数・変換規則（https://calcite.apache.org/docs/algebra.html）は意味保存の検査観点、SQLancerのTLP/NoREC（https://github.com/sqlancer/sqlancer）は独立oracleに加えて同値変換を利用するメタモルフィック試験の参考となる。ただしTEIでのSQL書換えや各DB製品での同値を未検証のまま採用しない。

最初のReferenceでは、固定fixtureの明細2行が集計結果1行へ寄与したことを**観測/手書きoracle**で示す範囲にとどめる。一般的なwhy/how provenanceや証明書の生成、Leanライブラリの依存導入、実行時追跡の性能予算は、実際の表示価値と必要性を確認してから決める。

## 10. Owner判断と未実証

- Owner判断：このQuery Workbenchの最初の製品受入対象を「**設計書上の可読性**」に限定し、意味評価/制約生成を段階追加としてよいか。query-local fixtureを#1131 Datasetへ接続する最小契約と、手書き期待値のreview主体は誰か。NULL/空集合/0件の業務上の扱いが業務固有なら受理前に固定しない。
- 未実証：CTE evaluator、SQL変換の同値性、行lineage、編集UI、SHOへのQuery資料生成、Lean/SMT生成、別Providerの結果同値、Owner業務受理。
- **本書の採否**：現時点では候補。既存#1640/#1131/#1124/#1803の受入条件を無断変更せず、担当レビュー後に最初の実証範囲へ反映する。実装や試験を実行したとの主張はしない。
