# SQL構造モデル・テストデータ付きQuery Workbench：最小実証計画（提案）

- 作成：2026-10-30
- 状態：**設計・検証計画候補**。公開契約の採用、実装完了、SQL/DB間の意味同等性、Ownerの業務受理を表さない。
- 基準：2026-10-30確認時のmain `2503a25fee9ffa47592d47818c2bb35c0a2c50ab`。実装着手時は再照合。
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

1. **業務データ層**：受注/受注明細/顧客等のAccepted Entity、Business identity、候補キー、FD、Relation、Domain、Constraint。#1126が所有。
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

### 固定fixture F0（値・期待値は独立に指定）

- customers: (C01, 青空商事), (C02, 山田商店)
- orders: (O01,C01,OPEN), (O02,C01,OPEN), (O03,C02,OPEN), (O04,C02,CLOSED)
- order_lines: (O01,1,2,100), (O01,2,1,50), (O02,1,1,50), (O04,1,3,100)
- データ型/制約前提：orders.order_id一意、order_lines(受注ID,行番号)一意、参照先存在、quantity/price非NULL。整数例は桁あふれしない。通貨/丸めなど業務採用は対象外。
- `line_totals`: (O01,250),(O02,50),(O04,300)。
- `open_orders`: (O01,C01,250),(O02,C01,50),(O03,C02,0)。
- `threshold=200`、final: (O01,青空商事,250)。
- assert：O03は明細なし→LEFT JOINで残る→CASEで0→最終WHEREで落ちる。O04は集計に存在してもOPEN絞込で落ちる。

期待値はSQL/evaluatorから逆生成せず、手書きの独立oracleとして固定。fixtureは**事前にデータモデルへの適合性を検査**する。

### 反証ケース（予定。現在未実行）

| ID | 最小変更 | 期待する区別/診断 |
| --- | --- | --- |
| N1 | 明細のないO03と、明細1件だがquantity=NULLのO02を比較する | absent groupを0にするCASEと、存在するNULL集計を別扱い。WHEREのUNKNOWNも識別 |
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

## 5. テストデータはモデルに束縛された資産

- #1131の型付き`Dataset / Snapshot / Expectation`を所有元とし、問い合わせ局所のfixtureは独立した業務Scenario正本ではなく**Queryテスト用の局所入力と結果期待**。
- 共有Datasetを選択し、必要なEntityの行をscopeへ投影。上書きは暗黙で行わず、差分と衝突を先に検証する。組合せにはDataset revision、局所patch digest、選択基準、パラメータ、Domain/Constraint revision、評価時計とlocaleを固定する。
- Modeler編集UIは、キー・親子・参照・サブタイプ・Domainの値域・NULL/欠損の区別を表示。入力値の修正はDataset側に戻し、CTE中間結果の表を直接編集しても元のデータモデルへ反映しない。
- CTE局所テストで仮想出力に直接行を与える場合、「下流scopeのみを検証」「上流未検証」を明示。全問い合わせ試験と同じ資格にしない。
- 実DBからデータ取込する際は#1807のschema/key/権限/個人情報・来歴検査を通す。外部の実データをそのまま公開fixtureへ混ぜない。
- Model/fixture/SQLの三つのrevisionは独立。テスト結果は`input snapshot + operator/SQL digest + evaluator/target + execution context + expected + observed + unresolved`へ追跡できる形で既存Evidence境界に紐付け、新しい一律Registryを作らない。

### 制約と生成（段階候補）

M1では既存Domain（型・範囲・列挙値）とRelation（キー・参照）の決定的な合成入力、境界値、負例を優先。M2以降にConstraint-based generation（組合せ制約、Coverage-guided generation、失敗時の縮小）を追加。Lean/SMTは一つの形式的記法候補であり、最初に新証明基盤は採用しない。任意のLean述語からの完全な生成/判定は要求しない。

生成の四つの状態を分ける：`satisfying candidate`（検査できた範囲で制約充足）、`unsatisfiable`（確認できた矛盾）、`unsupported/timeout`、`unverified`。生成できたこと、Queryの独立oracleが一致したこと、Ownerが業務意味を受理したことは非同等。意味のあるNULL/欠損/空集合や多重度はOwner未決なら仮定を明記する。

## 6. 解析・評価・実行の責務と停止条件

- **解析**：native SQLは正本、Scope/IRは導出。クエリ名やSQL aliasから業務意味を推測して昇格しない。構文/解決不完全なら各要素にcoverageを表示。
- **意味評価器**：主要業務のRead-only関係演算を外部Hostなしで検証する候補。現在の`DerivedSelect`はCTEを拒否するため、そのまま実装済みとして接続しない。対応演算と意味同値を独立fixtureで限定実証する。
- **SQL Provider**：必要な権限と同版snapshotがある場合にsandbox/使い捨てDBで評価。テストのために元SQLを切り出して中間結果を得る場合、書換え後の問合せと元の意味同値を確認できない限り**説明用の派生評価**と表示し、正式結果と同一視しない。
- **実行計画**：#1640のFilter/Derive/Aggregate/Sort/Limitの依存順を守る。全行に対する述語は、Providerが返却した集合Rへの`checkAll(R)`だけで、業務対象集合Lの性質`P(L)`を証明しない。`R=L`を検証する入力同定・Provider完全性・同版鮮度は別の証拠。
- **副作用**：Queryの局所試験はread-only。業務Actionの変更前後、Commit、Receipt、Replayは#1129/#1174/#1180の別経路で検証し、表示上の成功を実際の業務効果へ読み替えない。
- **開示**：権限のないフィールドや中間行を表示しない。許可された集約結果から保護対象の元行を逆表示することを禁止する。
- **停止**：NULL/文字照合/丸め/時点/重複/相関/Recursive等の意味が実現先で一致する根拠がない場合、unsupported/unresolvedを示しsilent fallbackしない。

## 7. 表示とSHO設計書

1. **構造ビュー**：実Entityと仮想Relationの図。Entityの親子/参照は関係線、CTE導出は演算線。ひとつのQueryを高密度表・図・ツリーに投影。
2. **値ビュー**：fixture行、演算前後の行数、行の残留/除外、金額などの値変化。行を選択すると確認できた根拠範囲だけ強調。
3. **論証ビュー**：key/FD/cardinalityの根拠、型・NULL/重複、解析coverage、期待/実際の差、未検証の前提とその依存範囲。
4. **Sourceビュー**：SQL本文のsource spanと選択したscope/field、編集できるauthoritative sourceへの戻り先。派生中間表を編集可能な正本として誤認させない。
5. **設計書ビュー**：SHO向けに入力Entity、各CTEの役割、仮想表、試料、期待・観測結果、未確認事項を配置。説明文、モデル由来の図、読み取った画像などの表示依存を分け、固定納品版と更新可能な作業版を区別。

#1803と既存`reference-design-output`のsourceRevision/structureRevision/artifact digestを再利用する。fixture revision、evaluated context、evaluator version等は**説明に必要な追加の来歴参照**であり、artifact byte一致だけで評価・意味・人のレビューの現在性を保証しない。SHO Composerの別文書正本を作らない。

## 8. 実装順・担当引継ぎ（提案）

| 段階 | 所有 | 実施 | 完了の根拠 |
| --- | --- | --- | --- |
| P0：静的説明スライス | #1640 + #1126、投影は#1124 | 既存CTE scope/RelationalIR→仮想関係の表・三種の線。source span、未解決ラベル | F0の構造と入力→出力属性の対応。未評価を明示し、SQLを変更しない |
| P1：fixture/期待値の一巡 | #1131 + #1640 | F0、N1〜N4、独立oracle、CTE各段階の期待/実際と差分 | 正負と0件/NULL/重複を区別。既存D2の範囲を再実装しない |
| P2：Modeler/資料連結 | #1124 + #1803 | SQL ↔ モデル ↔ 行 ↔ 設計書の一往復。編集可能元と導出部分を区別 | 同じrevisionを辿れ、変更後はstale表示。既存manifest/文書出力を活用 |
| P3：生成制約と別Target比較 | #1131 + #988/#1121、#1180はScenario再利用 | Domain/Constraintに基づく生成、最小の第二eval/実DB比較 | 生成の適用範囲と差異を説明できる。Lean/SMT、全SQL、各DBは先行決定しない |

P0/P1の固定fixture・紙上oracleは並行可能。P2はP0のID/出力を受け取る。P3はP1で見つかった実不足をもとに範囲決定。既存Issue/PRに所有機能があればそこへ差分を寄せ、新しいWorkBenchモノリスを作らない。

## 9. Owner判断と未実証

- Owner判断：このQuery Workbenchの最初の製品受入対象を「**設計書上の可読性**」に限定し、意味評価/制約生成を段階追加としてよいか。query-local fixtureを#1131 Datasetへ接続する最小契約と、手書き期待値のreview主体は誰か。NULL/空集合/0件の業務上の扱いが業務固有なら受理前に固定しない。
- 未実証：CTE evaluator、SQL変換の同値性、行lineage、編集UI、SHOへのQuery資料生成、Lean/SMT生成、別Providerの結果同値、Owner業務受理。
- **本書の採否**：現時点では候補。既存#1640/#1131/#1124/#1803の受入条件を無断変更せず、担当レビュー後に最初の実証範囲へ反映する。実装や試験を実行したとの主張はしない。
