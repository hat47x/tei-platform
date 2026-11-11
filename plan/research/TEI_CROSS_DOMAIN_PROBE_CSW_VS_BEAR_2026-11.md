# TEIの汎用モデリング能力を業務外領域で確かめる限定実証（CSW・VSクマ）

Status: **研究・比較・小規模試作のみ（2026-11-11）**。製品本体のアーキテクチャ、CSWの3 Skill、VSクマのGodot実装を変更していません。採用判断は行っておらず、下記の判断事項も未決です。

対象版はTEI `b8cbd432a`、CSW `73efa57ef66a`、VSクマ `5dae4c873e39`です。VSクマは後続の`origin/main` `a7d0cbae9`でも参照したpathに差分がないことを確認しました。版が進んだ場合は、ここに書いた内容を最新版の説明として扱いません。

**2026-11-11 追記（再検証）**：TEI `main`が`b987b38e7`へ進んだ後、このprobeを`origin/main`のcleanな隔離worktreeへコピーして再実行しました。21→24件のGo試験が通り、`tei validate`の宣言数はCSW 37・VSクマ 68で変化なしでした。`b8cbd432a`以降の`main`の変更（期間が名指す能力の効果Dataの照合、tools索引）は、probeが使う意味DSL・Workspace読込み・型列挙の範囲に触れません。ただし**この再検証はprobeの適用確認であり、上流（CSW・VSクマ）の最新版での再確認ではありません**。VSクマの上流照合は、兄弟チェックアウトが届く環境でのみ`TestUpstreamAnchorsResolveInTheCitedFiles`が実行します（§8）。

再現用の一式は[業務外領域probe](../../research/tei-cross-domain-probe-2026-11/README.md)にあります。関連IssueはTEI [#4193](https://github.com/hat47x/tei-platform/issues/4193)・[#2510](https://github.com/hat47x/tei-platform/issues/2510)・[#2533](https://github.com/hat47x/tei-platform/issues/2533)・[#3475](https://github.com/hat47x/tei-platform/issues/3475)・[#4134](https://github.com/hat47x/tei-platform/issues/4134)、CSW [#593](https://github.com/hat47x/cultural-substrate-weaving/issues/593)・[#543](https://github.com/hat47x/cultural-substrate-weaving/issues/543)、VSクマ [#95](https://github.com/hat47x/vs-bear-simulator/issues/95)・[#17](https://github.com/hat47x/vs-bear-simulator/issues/17)です。本稿はこれらのIssueの結論を変更しません。

**Issue番号の修正**：VSクマの関連Issueを`#402`・`#316`と記載していましたが、型付き認知の境界と採否の担当は[#95](https://github.com/hat47x/vs-bear-simulator/issues/95)（temporal state・evidence accumulation・action commitment・body feedback）と[#17](https://github.com/hat47x/vs-bear-simulator/issues/17)（M5 Bio-inspired cognition decision gate）です。`#402`はstory/cognitionの永続知識と一時状態の分離、`#316`はモデル駆動の反実仮想試験で、契約の写しの受入判断を返す先ではありません。判断の返し先を実態に合わせて修正しました。

## 1. 何を確かめ、何を確かめないか

TEIは「業務システムに限らず、分野固有の知識と技術を分けて扱うことが役立つ領域」を対象とすると説明しています（[README](../../README.md)）。この説明が、業務アプリケーション以外でも実用的な価値を持つかを、2つの独立した対象で限定的に確かめます。

確かめることは、TEIの宣言だけで対象のどの部分を写せて、どの部分が写せず、写せた部分が読みやすさ・意味の保持・実装負荷・性能・変更容易性の面で何をもたらすかです。確かめないことは、CSWやVSクマの品質改善、TEIによるコード生成の優越性、実運用での効果です。

境界として、次の制約を守りました。CSWの連想、異質性、仮説、問い、比較結果を、業務的な確定意味へ自動変換しません。VSクマのGodot物理処理、描画、ゲームループ、既存の認知実装を置き換えません。2つの対象の結果は別々に評価し、片方で成立した方式をもう片方へ自動的に適用しません。初回は研究・比較・小規模試作までとし、製品本体のアーキテクチャ変更を行いません。

## 2. 対象と現行方式

| 対象 | 確認した版 | 読んだ主な対象 | 現行方式 |
| --- | --- | --- | --- |
| CSW | `73efa57ef66a` | 3つの`SKILL.md`、`plan/`の研究文書、`plan/rd-first-run-inputs-v1/`の固定入力 | MarkdownのSkill本文と著者の散文。追跡対象27ファイル。実行時コード、Schema、CIを持たない |
| VSクマ | `5dae4c873e39` | `cognition/bear/*.gd`、`docs/BEAR_TYPED_COGNITION_BOUNDARY.md`、`tests/fixtures/bot_scenario_*.json` | Godot 4.7.2のGDScript。型付き観測・判断の境界と、決定論的な専用runner群 |

CSWのBaselineは、文化的・思想的体系を一時的な認知場として開き、著者が散文として受け取る方式です。由来の区別（原稿にあること、体系から生まれた問い、接触から生まれた読み、著者の判断）は文章で保持されますが、機械で照合できる成果物にはなっていません。固定入力のSHA-256と最初のR-Dの入力は記録済みですが、生成モデルへの独立投入はNOT RUNと明記されています。

VSクマのBaselineは、`BearTypedDecision`、`BearLocalTypedStateEncoder`、`BearTypedWireCodec`、`BearMotionFeedback`が入力の無効値・有限性・期限・範囲を実行時に検査し、専用runnerが境界を多数のチェックで確認する方式です。`docs/BEAR_TYPED_COGNITION_BOUNDARY.md`には112・64・188・319・316・85+121件のチェックが記録されています。これらは既に成立している検査であり、TEIはこれを置き換えません。

## 3. 両対象で共通に使えたTEIの構造と、使えなかった構造

### 3.1 共通に使えた構造

| TEIの構造 | CSWでの用途 | VSクマでの用途 | 確認した範囲 |
| --- | --- | --- | --- |
| 名前付き`data`と`attribute` | 入力・実行・断片・作者決定の4種 | 観測・身体サンプル・判断・シナリオ期待の4種 | `validate`が`ok`。宣言数はCSW 37、VSクマ 68 |
| 型の別名と列挙 `type ... from std.code values { ... }` | 帰属4値、決定4値 | 意図4値、認知モード4値、障害物モード4値、回復状態5値、観測版、判断源5値 | 実行時定数との一致をGo試験で照合 |
| `relation`（`one`/`many`、`key`） | 断片→入力、断片→実行、決定→断片、実行→断片群 | 判断→観測、観測→身体サンプル | 参照先の解決と、`relation-unique`の前提を確認 |
| `capability`と`effect` | 記録4操作（入力・実行・断片・決定） | 記録2操作（観測・判断）とシナリオ期待1操作 | 型の実在と効果の宣言のみ。権限・実行主体は与えない |
| `validation`（`positive`・`relation-unique`） | 順序の正値、実行内での順序一意 | 合計tickと最小stepの正値 | 2種のみ。範囲・列挙・時系列は表現できない |
| Workspaceの読込み・正規化・診断 | 型付き境界の妥当性 | 契約の写しの妥当性 | `tei validate -json`と`internal/workspace`の双方で確認 |

### 3.2 使えなかった構造

| 必要だったもの | TEIでの可否 | 代替した方法 |
| --- | --- | --- |
| 状態機械と遷移の網羅（VSクマのscheduler・mailbox・episode・回復状態） | 現行の意味DSLに`state`・`transition`がない | 状態名を列挙型として写し、遷移規則はGDScriptのまま。状態機械の表現は[#4193](https://github.com/hat47x/tei-platform/issues/4193)の担当 |
| 数値範囲・有限性・時間窓（±0.18 rad、±0.25、0..1、100/250 ms） | `validation`は`positive`と`relation-unique`のみ | read-onlyのGo probeで検査。TEIは範囲を持たない |
| 任意項目の欠如（身体サンプルがない観測） | `relation`に「無い」を表す cardinality がない | probeで空サンプルを別扱い。モデル上は存在を前提にしない |
| 配列属性（`final_position`、`stop_reasons`） | 属性に配列型がない | スカラーだけをTEIへ写し、配列はprobeで形を検査 |
| 対象の同一性（シナリオにIDがない） | `data`はCanonical IDを要求する | 呼出し側が`scenario_id`を付与。上流形式にない識別子をTEI側が要求する差分として残した |
| 行為者・権限（作者だけが決定できる） | `capability`にPrincipalや認可条件がない | `author_ref`を必須にし、probeで空を拒否。認可の証明ではない |
| 本文・連想・問いの意味分類 | 表現できるが、しない判断をした | 本文は型の外に置き、ハッシュと参照だけを持つ |
| 開いた集合（transportが新しいbackend_idを宣言できる） | 列挙は閉じた集合としてしか書けない | 記録済みの値を列挙し、未記録の値はwarningとして扱う。閉じた集合として書かない |
| 履歴・時系列の制約（実行→断片→決定の順序） | 時刻属性は持てるが、順序制約・履歴・版を持てない | probeで参照整合性だけを検査。時系列の保証はしない |

### 3.3 共通構造から言えること

両対象で共通に使えたのは、**同一性・名前・単位・列挙・関係・操作宣言・2種類の検証**です。これは「その境界に何があり、何を参照し、どの操作で作られるか」を機械可読に写せます。一方、両対象の実務上の検査が集まる**範囲・有限性・時間・状態遷移・欠如**はTEIの外に残りました。したがって現行TEIは、この2領域では境界の**鏡**にはなれますが、境界の**権威**にはなれません。

## 4. CSW：BaselineとTEI候補の差分

### 4.1 Baseline（現行mainのまま）

CSWは3つのSkill本文と参考テキストで成立し、成果は著者の散文です。連想・異質性・仮説・問い・比較は探索の材料として保持され、確定した意味へ変換されません。`cultural-substrate-weaving/SKILL.md`は「生成過程の由来を保持する」段で、原稿にあること・体系から生まれた問い・接触から生まれた読み・著者の判断を区別します。同時に「全候補へのラベルや一覧表は要らない」とし、記録を必須の手続きにしていません。

### 4.2 候補（型付き受け渡しだけ）

CSWで候補にしたのは、入力材料と成果を型付きで受け渡す部分だけです。Workspaceは4つの`data`で構成します。

- `StudyInput`：固定入力の識別子、出典版、内容のSHA-256、順序。
- `StudyRun`：実行の識別子、モデルとHarnessの自己申告、独立コンテキストの宣言、開始時刻。
- `CandidateFragment`：断片の識別子、入力と実行への参照、順序、帰属ラベル、本文のSHA-256と参照。
- `AuthorDecision`：決定の識別子、断片への参照、決定値、作者参照、決定時刻、根拠のSHA-256（任意）。

帰属ラベルは`source_material`・`generated_proposal`・`contact_reading`・`author_stated`の4値で、CSWが文章で区別している由来だけを写しました。4値は`skills/cultural-substrate-weaving/SKILL.md`の「生成過程の由来を保持する」が列挙する4つの区分と一対一に対応し、どれが欠けても二つの由来が混ざります。`TestAttributionCoversEveryCSWProvenanceDistinction`が引用と値の対応を固定します。「仮説」「問い」「比較」といった語を列挙型にしていません。それらを固定の分類にすると、探索の途中にある未整理の材料を確定した種類へ押し込むことになるためです。

**初期案からの修正**：最初の案は3値で、`contact_reading`（原稿と体系の接触から新しく生まれた読み）が抜けていました。SKILL.mdは4つの区分を明示しており、3値では「接触から生まれた読み」を`source_material`か`generated_proposal`へ押し込むことになります。TEIに写せる範囲と、写すときに落とせる区分は別の検討であり、後者はprobeの転記の問題として修正しました。

### 4.3 候補から外したもの

次のものは型付き境界へ入れていません。連想の内容、異質性の記述、仮説、問い、体系間の比較結果、作品本文、体系解釈、作者の意図の意味です。型が持てるのは、それらが「どの入力から、どの実行で、どの帰属で出てきたか」という参照とハッシュだけです。本文をフィールドとして渡すと`DecodeHandoff`が未知フィールドとして拒否することを試験で固定しています。

### 4.4 独立した期待結果と試験

CSW側の期待結果は、TEI Workspaceが宣言した内容そのものです。`validate -json`は`ok:true`で、宣言数は名前空間1・型4・`data`4・属性22・`capability`4・`validation`2・合計37でした。Go側の8試験が、次を独立に確認します。

- TEIの`csw.attribution`の値集合と、Goの受理値集合が一致すること。
- 帰属4値がSKILL.mdの4つの由来区分と一対一で対応し、探索語彙が値にならないこと。
- 成立fixtureが帰属4値をすべて実演すること。
- `CandidateFragment`の属性が参照とハッシュだけで、本文属性を持たないこと。
- 本文フィールドを持つ記録が拒否されること。
- 成立fixtureが診断0件で通ること。
- 未知の帰属と作者参照のない決定が拒否されること。
- 同一実行内の順序重複と、存在しない断片への決定が拒否されること。

probeの実測は、Windows 11・Intel Core i7-8750H・Go 1.27.0で、Workspaceの読込みと検証が4.8〜6.2 ms、成立fixtureの読取りと検査が130〜585 µsでした（3回の実測の範囲。同hostでも実行ごとに数倍振れます）。これはprobeの費用であり、CSWの執筆やモデル実行の費用ではありません。

### 4.5 CSWの評価

意味の保持について、候補は「何を・いつ・どの入力から・誰の判断で」を保持し、「何を意味するか」を保持しません。これは不足ではなく、CSWがTEIへ渡すと決めた範囲です。実装負荷は`handoff.tei` 81行とprobe 429行（試験込み）で、CSWの製品側は0行です。可読性は高く、型付き境界の全体が1ファイルで読めます。

リスクは、帰属ラベルが運用で分類表に育つことです。CSWは全候補へのラベルを求めていないため、この境界は任意で、著者が使うときだけ記録する形でなければなりません。また、ハッシュは記録者が渡す値で、probeは本文を開いて照合しません。来歴の真正性は、この候補だけでは保証できません。

## 5. VSクマ：BaselineとTEI候補の差分

### 5.1 Baseline（現行mainのまま）

VSクマは、観測（`typed-observation@1.1.0`）と身体フィードバック（`body-motion-feedback@1.0.0`）を入力に、6項目のwire形式で判断を受け取り、有限性・範囲・期限・減衰・RESET・条件変更をGDScriptで検査します。シナリオは`bot-scenario-1`のJSONで、`total_ticks`・`min_steps`・`stop_reasons`・`final_position`・`final_position_tolerance`・`trajectory_sha256`をassertionとして持ちます。

### 5.2 候補（TEIに適した最小要素）

状態遷移、運動パラメータ、認知モデルの入出力、テストシナリオから、表現・検証・生成に適した最小要素だけを選びました。

- 観測：版、episode/request、観測時刻、可視フラグ、方位・距離・方位変化・速度・slip・支持、障害物の前方距離、接触、grip、前回速度と速度差、認知モード、記憶年齢、不確かさ、予測確信度、再発見回数、障害物交渉モード。
- 身体サンプル：有効・判定可能、サンプル年齢、身体目標速度、試行速度、実進捗、適用yaw率、回復状態、回復episode、側。
- 判断：episode/request、発行時刻、期限、意図、heading・persistence・braceのバイアス、不確かさ、確信度、判断源。
- シナリオ期待：seed、合計tick、最小step、最終位置の許容差、軌道ハッシュ。

物理積分、描画、ゲームループ、schedulerの内部状態、知覚記憶の全項目、判断の補正後制限はモデル化していません。

### 5.3 置換しないもの

Godotの物理処理、描画、ゲームループ、既存の認知実装を置き換えません。GDScriptを生成せず、ゲームからTEIへの実行時依存も作りません。TEIは境界の写しと検証の補助に留まり、実行時の権威は既存のGDScriptと専用runnerに残ります。

### 5.4 独立した期待結果と試験

VSクマ側の期待結果は、上流に書かれた事実から独立に用意しました。wireの例`maintain|+045|+08|+12|10|88`、意図4種、境界値、期限、`bot-scenario-1`のassertion名とsmoke fixtureの値です。Go側の16試験が、次を確認します。

- TEIの列挙（意図・認知モード・障害物モード・回復状態・判断源・観測版）が実行時定数と一致すること。
- 転記した列挙・境界値・スキーマ名が、引用した上流ファイルに実際に存在すること（兄弟チェックアウトが届く環境のみ。届かない場合はskip）。
- 判断の境界値が端点を含み、端点を超えた値が拒否されること。
- 未来の判断、期限切れ、期限の逆転が拒否されること。
- 記録されていないproducer名がwarningにとどまり、判断自体は使えること。運行時はtransportが新しいbackend_idを宣言でき、`effective_weight`はproducer名を見ないためです。
- wire符号化が上流の例と一致し、範囲外の値を丸めずに拒否すること。
- wire解析が往復し、壊れた行と範囲外の行を拒否し、TTLを0.01〜0.50秒へ制限すること。
- シナリオのsmoke fixtureが期待どおり投影され、未知のassertion、短いハッシュ、0以下の許容差、空の停止理由が拒否されること。

`validate -json`は`ok:true`で、宣言数は型9・`data`4・属性49・`capability`3・`validation`2・合計68でした。probeの実測は、Workspaceの読込みと検証が7.6〜13.9 ms、wire符号化が1.5〜7.7 µs、シナリオ投影が51.5〜74.8 µsでした（3回の実測の範囲。同hostでも実行ごとに数倍振れます）。

### 5.5 VSクマの評価

意味の保持について、名前・単位・列挙・関係は写せましたが、範囲・有限性・時間窓・状態遷移は写せませんでした。したがってTEI単独では境界の不変条件を保持せず、probeのGoコードがその一部を代替しています。可読性は高く、境界の形はGDScriptを読まずに把握できます。一方で同じ境界をGDScriptとTEIの2箇所に持つため、列挙以外は自動で同期しません。上流が変わればprobeは追随せず、手で更新する必要があります。

**初期案からの修正**：`bear.decision_source`を`typed_wire`・`local_typed_hybrid`・`async_typed_hybrid`・`none`と写していましたが、これは誤りです。`local-typed-hybrid@…`と`async-typed-hybrid@…`は`provider_id()`の値で、`BearTypedDecision.source`ではありません。上流が実際に書く`source`は`none`（既定）・`typed_wire`・`jev-shaped-rules`・`failure-aware-rules@0.1.0`・transportの`backend_id()`（既定`async-wire`）です。controller provider IDとdecision producerは別のフィールドで、混同すると「判断の出所」を誤って読むことになります。加えて、transportは新しい`backend_id`を宣言できるため`source`は閉じた集合ではありません。未記録のproducerをerrorにすると、正当な新しいbackendを不合格としてしまうため、warningとして扱います。この転記誤りは、probe内の自己一致試験だけでは検出できませんでした。上流ファイルへの照合（`TestUpstreamAnchorsResolveInTheCitedFiles`）は、この種の誤りを検出するために追加しました。

実装負荷はTEI 104行とprobe 750行（試験込み）です。既存のGDScript 435行（`typed_decision`・`typed_wire_codec`・`local_typed_state_encoder`・`motion_feedback`の合計）を置き換えないため、総量は増えます。変更容易性は、意図を1つ追加する場合にGDScript・TEI・probeの3箇所を直す必要があり、現状では悪化します。

### 5.6 状態機械のread-only投影（#4193への判断材料）

回復状態機械（`controllers/bear/traversal_recovery.gd`）を、probe内にread-onlyの遷移表として写しました。**TEIの意味DSLには`state`・`transition`がなく、製品へは追加していません。** この投影は[#4193](https://github.com/hat47x/tei-platform/issues/4193)の受入条件1（「少なくとも二つの異質なDomainの独立Expectedと具体的不足で示す」）に対する、2例目のDomainとしての材料です。

写した結果は、状態4（IDLE・WALL_FOLLOW・ENTER_TRAVERSABLE・CLEARING）、遷移11（内1はglobal）です。`research/tei-cross-domain-probe-2026-11/bear/statemachine.go`に実装し、10件の試験（内4件はmutation）で確認しました。

| 観測 | 内容 | #4193の該当 |
| --- | --- | --- |
| 到達不能状態なし | 初期状態IDLEから4状態すべてへ到達できる | §5.4 到達可能性 |
| デッドエンドなし | 全状態に退出あり。ただし`observation_invalid`というglobal遷移があるためです。global行を落とすとCLEARINGがデッドエンドになります | §5.4 |
| Guard優先は実在し、かつ合計 | 各source状態の分岐は優先度1..nで競合しません。しかし**優先度はif/elifのソース順**であって、宣言されたpolicyではありません | §5.2 遷移決定性 |
| global遷移の可視性 | `observation_invalid`は個別状態のchainの前に一度だけ書かれ、任意の非初期状態からIDLEへ戻ります。source状態ごとの行だけを並べた平坦な表では、この退出が見えません | §5.1 型・構造 |
| 自己遷移 | `traversable_retargeted`はENTER_TRAVERSABLE→ENTER_TRAVERSABLE。平坦な表では「遷移が欠けている」ように読めます | §5.1 |
| 列挙と状態の不一致 | 記録される`recovery_state`は5値（4状態＋`unknown`）ですが、`unknown`は機械の状態ではなく、未認識の状態名に対して`motion_feedback.gd`が書くフォールバックラベルです。**列挙は状態集合ではありません** | §5.1 |
| 現行DSLでの表現 | `state`・`transition`がないため、遷移規則はGDScriptのまま。probeが表と診断を運びます | §6 進め方 |

**具体的な不足**として、#4193へ返せるのは次の**4点**です。(1) 優先度を宣言されたpolicyとして書く手段がない（ソース順に依存）。(2) global遷移を表の行として表現する手段がない（個別状態の行だけでは退出が隠れる）。(3) 列挙型で状態を書くと、フォールバックラベルが状態として読まれる。(4) 状態だけでは、遷移をまたいで保持される付随変数（ここでは`_latched_side`）を表せない。`_latched_side`は`reset()`と`_finish_episode()`でしか0に戻らず、`blocking_returned`と`traversable_returned`では以前に選んだ側が保持されます。遷移表は状態名だけを運ぶため、この保持は表に出ません。

**review記録**：[SM-01 review記録](../../research/tei-cross-domain-probe-2026-11/bear/STATE_MACHINE_REVIEW.md)が、上流523行を直接読んで投影を再確認しました。3点の独立確認は成立しています。同時に**blocking 1件**（IDLE→WALL_FOLLOWのguardが`_update_stall_age()`の連言条件のうち3条件を落としていた）を検出し、修正しました。この誤りは**anchor試験では検出できません**（落とした条件の文字列も上流に実在するため、存在確認は通ります）。§8の「転記の誤りは自己一致試験では検出できない」が、意味の欠落という形でもう一度現れた例です。

**判断材料としての限界**：この投影は1つのDomain・1つの状態機械です。#4193の受入条件1は「少なくとも二つの異質なDomain」を求めており、本稿の§5.2（型付き認知の入出力境界）と本節を併せても、Domainは「ゲーム」で1つです。UI非同期と通信Session（#4193 §6のS1）が未実施のままです。

## 6. 対象別の比較

### 6.1 CSWの5観点

| 観点 | 評価 | 根拠 |
| --- | --- | --- |
| 可読性 | 高い | 型付き境界が81行1ファイル。語彙は4つの`data`と4つの帰属値に限られる |
| 意味の保持 | 意図した範囲で成立 | 入力・実行・帰属4値・作者決定・本文ハッシュを保持。本文と連想は保持しない |
| 実装負荷 | 小さい | 製品側0行。probeは429行（試験込み）で、CSWの実行経路に入らない |
| 性能 | 問題にならない | 読込みと検証4.8〜6.2 ms、1ファイルの検査130〜585 µs。執筆の速度に影響する経路がない |
| 変更容易性 | 中程度 | 帰属値の追加はTEIとGoの2箇所。データ形状の版管理はWorkspaceの`formatVersion`のみ |

### 6.2 VSクマの5観点

| 観点 | 評価 | 根拠 |
| --- | --- | --- |
| 可読性 | 高い | 境界が104行で読める。列挙と単位が明示される |
| 意味の保持 | 形状のみ | 範囲・有限性・時間窓・状態遷移はTEIの外。probeが一部を代替 |
| 実装負荷 | 増える | 既存435行を置き換えず、TEI 104行とprobe 750行を追加 |
| 性能 | 実行時影響なし | probeは1.5〜7.7 µs／51.5〜74.8 µs／7.6〜13.9 ms。ゲームループに入らない |
| 変更容易性 | 悪化 | 列挙は一致試験があるが、境界値・wire・fixtureは手で同期。1変更が3箇所に及ぶ |

### 6.3 比較から言えること、言えないこと

言えることは、TEIの現行の意味DSLが、業務外の2領域でも**境界の同一性・関係・列挙・操作宣言**を一貫して写せたことです。両方で`validate`が通り、既存形式のfixtureをread-onlyで照合できました。また、両対象とも「TEIに写せない部分」が同じ種類（範囲・有限性・時間・状態遷移・欠如・配列）に集中しました。これはTEIの汎用性の限界が、領域の違いではなくDSLの表現力に由来することを示します。

さらに、**転記の誤りは自己一致試験では検出できない**ことが両対象で確認できました。CSWではSKILL.mdの4区分の1つが、VSクマではcontroller provider IDとdecision producerの混同が、それぞれ「TEIの宣言」と「probeの定数」が一致している状態で成立していました。probeの試験が一致を確認できるのは、転記元と同じものを二度書いていないことを確認できるだけです。上流の原文へ照合するanchor試験（`TestUpstreamAnchorsResolveInTheCitedFiles`、`TestAttributionCoversEveryCSWProvenanceDistinction`）が、この種の誤りを検出します。

言えないことは、TEIを使うことでCSWの執筆品質やVSクマの認知品質が上がること、コード生成が既存実装より有利であること、実運用の性能・保守性が改善することです。CSWは製品実行経路がないため、VSクマで生じた二重管理のコストをそのまま当てはめられません。逆に、CSWで成立した任意の記録方式を、VSクマの実行時契約へ自動適用してはいけません。両者の評価は独立です。

## 7. 採用判断が必要な事項

1. **TEIの検証語彙を広げるか**（TEI中核）。範囲・有限性・列挙値の検査を`validation`で表せるようにするか。両対象の実務検査はここに集中しており、これがない限りTEIは境界の権威になれません。本稿は採用を提案せず、[#4193](https://github.com/hat47x/tei-platform/issues/4193)と[#3475](https://github.com/hat47x/tei-platform/issues/3475)の判断材料として残します。
2. **状態機械の表現をどう扱うか**（TEI中核）。VSクマのscheduler・episode・回復状態は、現行DSLでは列挙に留まります。状態・遷移・時系列を意味モデルへ入れるかは[#4193](https://github.com/hat47x/tei-platform/issues/4193)の範囲です。本稿はどちらとも決めません。
3. **開いた集合を列挙型として書いてよいか**（TEI中核）。VSクマのdecision sourceは、transportが新しい`backend_id`を宣言できるため閉じた集合ではありません。列挙型に写すと、正当な新しいproducerを未知として拒否するか、未知を常に許して列挙の意味を薄めるかの二択になります。本稿は「記録済みの値を列挙し、未記録をwarning」という暫定対処を選びました。`unknown`をどう扱うかの規約は[#2533](https://github.com/hat47x/tei-platform/issues/2533)の判断材料として残します。
4. **転記の照合をどこまで要求するか**（TEI中核）。probeの誤り2件は、TEI宣言とprobe定数の自己一致では検出できませんでした。上流原文への照合は、兄弟チェックアウトがある環境でしか実行できず、CI必須検査にも接続していません。外部Repositoryの転記を伴う研究物に、どの照合を求めるかは未決です。
5. **CSWの型付き受け渡しを任意の道具として維持するか**（CSW）。推奨は、Skill本文を変えず、著者が使うときだけ記録する任意の形式として残すことです。必須化すると「全候補へのラベルは要らない」という現行方針と衝突します。判断はCSW [#593](https://github.com/hat47x/cultural-substrate-weaving/issues/593)に返します。
6. **VSクマで契約の写しを受け入れるか**（VSクマ）。受け入れる条件は、列挙の一致試験に加えて、境界値・wire・fixtureの同期を確認する手順を決めることです。現状は二重管理のコストが利益を上回る可能性があり、推奨は「保留」。判断はVSクマ [#95](https://github.com/hat47x/vs-bear-simulator/issues/95)と[#17](https://github.com/hat47x/vs-bear-simulator/issues/17)に返します。
7. **2対象の共通部分を標準ライブラリへ抽出するか**（TEI）。2例だけでは根拠が不足します。抽出しない判断を推奨し、3例目が出た時点で再検討します。
8. **probeを`research/`に残すか`tools/`へ移すか**（TEI）。現状は`research/`が適切です。製品の受入経路やCI必須検査には接続せず、上流の版を固定した研究物として扱います。

## 8. 実施済み・未実施（NOT RUN）

実施したことは、対象版と関連Issueの調査、2つのTEI Workspaceの作成、read-only probeの実装、**38件のGo試験**と5件のbenchmark、`tei validate`による確認、上流ファイルへの**48件の転記照合**（[ANCHOR_POLICY.md](../../research/tei-cross-domain-probe-2026-11/ANCHOR_POLICY.md)）、状態機械のreview（[STATE_MACHINE_REVIEW.md](../../research/tei-cross-domain-probe-2026-11/bear/STATE_MACHINE_REVIEW.md)）、本稿の整理です。

未実施は次のとおりです。CSWの実際の執筆セッション、生成モデルへの固定入力の投入、作者による採用判断。VSクマのGodot起動、既存runnerの再実行、実画面・実機での確認。TEIからGDScriptやSQLへの生成、probeのCI必須検査への接続、上流変更への自動追随。anchor試験のCIでの常時実行（兄弟チェックアウトがある環境のみ）。別ハードウェアでの性能測定、独立レビューアによる再検証、本節の判断事項の決定。

**性能実測の性質**：本稿のprobe費用は、Windows 11・Intel Core i7-8750H・Go 1.27.0という1台のhostでの3回の実測範囲です。同hostでも実行ごとに数倍振れるため、絶対値を性能目標や比較優位の根拠として使わないでください。TEIの製品コードやVSクマのゲームループの性能は測定していません。

## 9. 後続作業の作業票

§7の判断事項を、後続の実装者が迷わずに最後まで進められる形に分解しました。各票は[AGENTS.md](../../AGENTS.md)の「Issueでの委任」が求める項目を持ちます。**票は着手可能な状態まで具体化したもので、採用の決定ではありません。** 開始base SHAは票の作成時点の`origin/main`です。作業を始めるときは、最新の`main`と対象pathの差分を再確認してください。

| 票 | 親Issue | 要求ロール | 概要 | 状態 |
| --- | --- | --- | --- | --- |
| VOC-01 | [#2510](https://github.com/hat47x/tei-platform/issues/2510) | `implementer` | 範囲・列値の検証を`validation`へ追加する設計案とADR草案 | 未着手 |
| SM-01 | [#4193](https://github.com/hat47x/tei-platform/issues/4193) | `reviewer` | 状態機械のread-only投影の成果を#4193の受入条件へ接続するreview | **実施済み**（blocking 1件を修正） |
| SM-02 | [#4193](https://github.com/hat47x/tei-platform/issues/4193) | `implementer` | UI非同期と通信Sessionのread-only解釈（#4193 §6のS1） | 未着手 |
| OPEN-01 | [#2533](https://github.com/hat47x/tei-platform/issues/2533) | `architect` | 開いた集合を列挙型として書くときの規約 | **設計案あり**（[OPEN_SET_POLICY.md](../../research/tei-cross-domain-probe-2026-11/OPEN_SET_POLICY.md)） |
| TRACE-01 | [#4134](https://github.com/hat47x/tei-platform/issues/4134) | `implementer` | 外部Repository転記の照合下限と証跡の記録 | **実施済み**（CI接続は未実施） |
| CSW-01 | CSW [#593](https://github.com/hat47x/cultural-substrate-weaving/issues/593) | Owner判断 | 型付き受け渡しを任意の道具として残す場合の仕様 | 判断待ち |
| BEAR-01 | VSクマ [#95](https://github.com/hat47x/vs-bear-simulator/issues/95) | Owner判断 | 契約の写しを受け入れる場合の条件と同期手順 | 判断待ち（推奨は保留） |
| PLACE-01 | [#4193](https://github.com/hat47x/tei-platform/issues/4193) | `integrator` | `research/`から`tools/`への昇格基準 | **設計案あり**（[PLACEMENT_POLICY.md](../../research/tei-cross-domain-probe-2026-11/PLACEMENT_POLICY.md)） |

共通の開始base SHA：`b987b38e741f8cfda74159e405979cf369521b88`（2026-11-11時点の`origin/main`）。
共通の非目標：製品の受入Gateへの接続、CI必須検査の追加、CSWの3 Skillの変更、VSクマのGDScript生成、第二の意味正本の作成。

---

### VOC-01

**親Issue**：[#2510](https://github.com/hat47x/tei-platform/issues/2510)（多分野Pressure Corpusと独立Conformance）。**要求ロール**：`implementer`。**状態**：未着手。

**目的**：範囲・有限性・列値の検査を`validation`で表せるようにするかどうかの**採用判断に必要な設計案**を作る。判断そのものはADRで行う。

**許容変更範囲**：
- `plan/design/`に設計文書を1件追加する。
- `internal/validation`の**読み取り専用の照合**（PoC）を、`plan/fixtures/`または`research/`に追加する。製品コードは変更しない。
- 既存の`positive`・`relation-unique`の意味は変えない。

**非目標**：
- `validation`の構文を製品へ実装する（ADR採択後の別Issue）。
- 既存の`reference/workspace`を変更する。
- 状態機械・時系列・配列を同時に解決する。範囲と列値に限定する。

**受入条件**：
1. 2領域（CSWの帰属4値・VSクマの判断境界6項目）の**実在する検査**を、設計案の構文で写せることを示す。
2. 写せないもの（時間窓・状態遷移・欠如・配列）を、設計案の限界として明示する。
3. 独立Expectedを手書きし、実装から期待値を自動写しない。
4. 導入しない場合の代替（probeが代替する現状）と、導入した場合に削除できるものを対比する。
5. 「効果がなければどう撤回するか」を1段落で書く。

**検証**：変更したpackageの`go test`、`go run ./cmd/tei validate -workspace reference/workspace`、追加したfixtureの実行。実行できなかった検証は、その制約を設計文書へ書く。

**成果物**：設計文書1件、read-only照合1件、独立Expected付きfixture。

**レビュー返却ロール**：`reviewer`。重要な設計論点は`independent-reviewer`へ争点を限定して渡す。

---

### SM-01

**親Issue**：[#4193](https://github.com/hat47x/tei-platform/issues/4193)。**要求ロール**：`reviewer`。**状態**：**実施済み**（2026-11-11）。review記録は[STATE_MACHINE_REVIEW.md](../../research/tei-cross-domain-probe-2026-11/bear/STATE_MACHINE_REVIEW.md)。

**目的**：§5.6の状態機械のread-only投影を、#4193の受入条件1に対する**材料として妥当か** reviewする。投影の正しさと、投影が#4193へ返せる「具体的不足」の4点が実態に合っているかを確認する。

**許容変更範囲**：#4193へのコメント追記、研究文書§5.6の修正。製品コードは変更しない。

**非目標**：状態機械の製品実装、ADR起案、#4193の結論変更。

**受入条件**：
1. 遷移表11件が上流`traversal_recovery.gd`と一致すること（anchor試験で確認済み。reviewerは上流を直接読んで再確認する）。
2. 「優先度はソース順」「global遷移が平坦な表で隠れる」「列挙は状態集合ではない」の3点が、上流のソースから独立に確認できること。
3. §5.6の限界（Domainが1つしかないこと）が明記されていること。

**検証**：`go test ./research/tei-cross-domain-probe-2026-11/... -count=1`、上流ファイルの直接閲覧。

**成果物**：#4193へのreviewコメント（blocking／non-blocking／question の分類付き）。

**レビュー返却ロール**：`independent-reviewer`（設計上の争点がある場合）。

**実施結果の要約**：受入条件1は**当初未達**でした。IDLE→WALL_FOLLOWのguardが`_update_stall_age()`の連言条件のうち3条件を落としており、**blocking F-1**として検出しました。修正と回帰試験2件（`TestRecoveryMachineGuardNamesTheRealCondition`、`TestRecoveryMachineGuardsAreNotJustThresholds`）を追加済みです。受入条件2・3は成立。不足は3点から**4点**へ増えました（`_latched_side`の保持を表せない）。

---

### SM-02

**親Issue**：[#4193](https://github.com/hat47x/tei-platform/issues/4193) §6のS1。**要求ロール**：`implementer`。**状態**：未着手。

**目的**：#4193の受入条件1が求める「二つの異質なDomain」のうち、残る2領域（UI非同期・通信Session）をread-onlyで解釈する。SM-01で得られた「具体的不足」の3点が、別Domainでも同じ形で現れるかを確認する。

**許容変更範囲**：
- `research/tei-cross-domain-probe-2026-11/`にUI非同期と通信Sessionのread-only投影を追加する。
- 対象は**公開済み・合成の**資料に限定する。実案件のデータは使わない。

**非目標**：製品DSLへの`state`・`transition`追加、共通Runtime、新しいWorkflow Engine、UI実装。

**受入条件**：
1. 各Domainで、未知State/Event・重複Transition ID・未束縛のSource/Target・曖昧Guardの反例を**独立Expected**で1件以上示す。
2. 表の網羅チェックPASSを、イベント時刻の正しさの証明としない（#4193 §5の制限を守る）。
3. SM-01の3点の不足が再利用できるか、Domain固有の新しい不足が出るかを区別して書く。

**検証**：追加したpackageの`go test`、`gofmt -l`、`go vet`。

**成果物**：投影2件、独立Expected付きfixture、研究文書への追記。

**レビュー返却ロール**：`reviewer`。

---

### OPEN-01

**親Issue**：[#2533](https://github.com/hat47x/tei-platform/issues/2533)。**要求ロール**：`architect`。**状態**：未着手（probeには暫定対処あり）。

**目的**：開いた集合（上流が新しい値を宣言できる）を列挙型として書くときの規約を決める。VSクマのdecision sourceが実例。

**許容変更範囲**：規約の設計文書1件。製品コードは変更しない。

**非目標**：列挙型の廃止、未知値の常時許可、`unknown`の標準値化。

**受入条件**：
1. 「記録済みの値を列挙する」「未記録の値をwarningとして扱う」「閉じた集合として扱わない」の3択のトレードオフを、2領域の実例で示す。
2. 未記録の値をerrorにした場合に何が拒否されるか（VSクマでは正当な新backend）を具体的に書く。
3. warningを実行時に無視できる経路があるか、ないかを確認する。

**検証**：設計文書の整合確認。既存probeのwarning対処を実例として引用する。

**成果物**：規約の設計文書1件。

**レビュー返却ロール**：`independent-reviewer`。

---

### TRACE-01

**親Issue**：[#4134](https://github.com/hat47x/tei-platform/issues/4134)。**要求ロール**：`implementer`。**状態**：一部実施済み（anchor試験は実装済み、CI接続と証跡記録は未実施）。

**目的**：外部Repositoryの転記を伴う研究物に、どこまで照合を求めるかの下限を決め、証跡として記録する。

**許容変更範囲**：
- anchor試験の一覧（`upstreamAnchors`）への追加。
- 研究文書への**照合証跡**（実施日・host・上流commit・照合件数・skip件数）の記録。

**非目標**：CI必須検査への接続（A3の別判断）、上流の自動追随、ネットワーク経由の取得。

**受入条件**：
1. 転記した列挙・境界値・スキーマ名のすべてに、`(file, literal, why)`のanchorがあること。
2. anchor試験が上流に届かない環境で**skip**し、skipを一致の証拠として扱わないこと。
3. 照合証跡に、実施日・host・上流commit・照合件数・skip件数を書くこと。
4. 自己一致試験（TEI宣言とprobe定数の比較）だけでは転記の正しさが確認できないことを、証跡に明記すること。

**検証**：`go test ./research/tei-cross-domain-probe-2026-11/... -count=1 -run TestUpstreamAnchors -v`。兄弟チェックアウトがある環境とない環境の両方で実行し、結果を証跡へ書く。

**成果物**：anchor一覧の更新、照合証跡の記録。

**レビュー返却ロール**：`reviewer`。

---

### CSW-01

**親Issue**：CSW [#593](https://github.com/hat47x/cultural-substrate-weaving/issues/593)。**要求ロール**：Owner判断（`planner`が整理）。**状態**：判断待ち。

**目的**：§4.2の型付き受け渡しを、任意の道具として残すかどうかを著者が判断できる形にする。**判断は著者に戻します。本票は採用を提案しません。**

**採用した場合に追加されるもの（これだけ）**：
- `skills/writing-flow/references/`に、任意の受け渡しカードの雛形として、型の名前と4つの帰与値の対応を1節追加する。既存の[受け渡しカード](https://github.com/hat47x/cultural-substrate-weaving/blob/73efa57ef66ae4d53db3ff9bb63c927f624f44a8/skills/writing-flow/references/handoff-card.md)の雛形を**置き換えず**、補足として置く。
- 帰属4値（`source_material`・`generated_proposal`・`contact_reading`・`author_stated`）と、`SKILL.md`の4区分の対応を1行ずつ書く。

**採用しない場合に起きないこと**：型の追加、Schema、CI、テスト、配布基盤。CSWの製品側は0行のままです。

**必須条件（採用の前提）**：
- 「全候補へのラベルや一覧表は要らない」という現行方針を変えない。記録は**任意**で、著者が使うときだけ行う。
- 「仮説」「問い」「比較」を値にしない。
- Skill本文の手順は変えない。

**判断の材料**：§4.5のリスク2件（帰属ラベルが分類表に育つ、ハッシュは本文と照合されない）。

**受入条件**：著者が「残す／残さない」を明示し、その理由を1段落で#593へ書く。

---

### BEAR-01

**親Issue**：VSクマ [#95](https://github.com/hat47x/vs-bear-simulator/issues/95)と[#17](https://github.com/hat47x/vs-bear-simulator/issues/17)。**要求ロール**：Owner判断。**状態**：判断待ち。**推奨は「保留」**。

**目的**：契約の写し（TEI 104行）を受け入れるかどうかを、条件付きで判断できる形にする。

**受け入れる場合に必要な手順（これが対価）**：
1. 列挙の一致試験に加えて、**境界値・wire形式・fixtureの同期**を確認する手順を決める。
2. 同期手順を、`docs/BEAR_TYPED_COGNITION_BOUNDARY.md`の検証節に1行追加する（専用runnerは作らない）。
3. 上流が変わったときにprobeが追随しないことを、受け入れ側の記録に明記する。

**受入条件（adoption の判定基準）**：
1. 上記3点の手順が決まり、実施された証拠があること。
2. 二重管理のコスト（GDScript・TEI・probeの3箇所）が、可読性・review理解の利益を上回らないこと。
3. 判断が`deterministic-baseline`の既定とM5の採否基準を変更しないこと。

**非目標**：GDScript生成、ゲームループへの依存、既定providerの変更。

**判断の材料**：§5.5（意味の保持は形状のみ、実装負荷は増える、変更容易性は悪化）、§5.6（状態機械も現行DSLでは列挙に留まる）。

---

### PLACE-01

**親Issue**：[#4193](https://github.com/hat47x/tei-platform/issues/4193)（横断の記録先）。**要求ロール**：`integrator`。**状態**：未着手。

**目的**：probeを`research/`に残すか`tools/`へ移すかの基準を決める。[AGENTS.md](../../AGENTS.md)の情報の置き場所の区分（`research/`は研究目的と有効な制約があり既存の参照実装で代替できない実装・データ、`tools/`は再現可能な検証・参照ツール）を、このprobeにどう当てはめるかを確認する。

**許容変更範囲**：基準の設計文書1件。移動は行わない。

**非目標**：probeの移動、`tools/README.md`の変更、CI必須化。

**受入条件**：
1. `research/`にとどまる条件と、`tools/`へ移す条件を、それぞれ2項以上で書く。
2. 現在のprobeがどちらに当たるかを、基準に照らして判定する（現状の判定は「`research/`が適切」）。
3. 昇格した場合に、`tools/README.md`と既存の索引へ何を追加するかを書く。

**検証**：設計文書の整合確認。`go test ./research/tei-cross-domain-probe-2026-11/... -count=1`が現状通り通ることを確認する。

**成果物**：基準の設計文書1件。

**レビュー返却ロール**：`reviewer`。

## 10. 参照

- [TEI README](../../README.md)、[TEIの考え方](../../docs/overview.md)、[TEIを動かす](../../docs/getting-started.md)
- [業務外領域probe](../../research/tei-cross-domain-probe-2026-11/README.md)
- CSW：[3つのSkillの説明](https://github.com/hat47x/cultural-substrate-weaving/blob/73efa57ef66ae4d53db3ff9bb63c927f624f44a8/README.md)、[文化的体系を執筆に使う](https://github.com/hat47x/cultural-substrate-weaving/blob/73efa57ef66ae4d53db3ff9bb63c927f624f44a8/skills/cultural-substrate-weaving/SKILL.md)、[最初のR-D入力](https://github.com/hat47x/cultural-substrate-weaving/blob/73efa57ef66ae4d53db3ff9bb63c927f624f44a8/plan/CSW_MAGI_RD_FIRST_RUN_INPUTS_20261102.md)
- VSクマ：[型付き認知の入出力境界](https://github.com/hat47x/vs-bear-simulator/blob/5dae4c873e393a9d9507ad037a4cf5e11988575a/docs/BEAR_TYPED_COGNITION_BOUNDARY.md)、[`BearTypedDecision`](https://github.com/hat47x/vs-bear-simulator/blob/5dae4c873e393a9d9507ad037a4cf5e11988575a/cognition/bear/typed_decision.gd)、[`BearTypedWireCodec`](https://github.com/hat47x/vs-bear-simulator/blob/5dae4c873e393a9d9507ad037a4cf5e11988575a/cognition/bear/typed_wire_codec.gd)、[smokeシナリオ](https://github.com/hat47x/vs-bear-simulator/blob/5dae4c873e393a9d9507ad037a4cf5e11988575a/tests/fixtures/bot_scenario_smoke.json)
- [CSW-MAGIの前提・根拠マップ](https://github.com/hat47x/cultural-substrate-weaving/blob/73efa57ef66ae4d53db3ff9bb63c927f624f44a8/plan/CSW_MAGI_ASSUMPTION_CONTEXT_AND_EVIDENCE_MAP_20261029.md)、[CSWの認知再フレーミング研究ノート](https://github.com/hat47x/cultural-substrate-weaving/blob/73efa57ef66ae4d53db3ff9bb63c927f624f44a8/plan/CSW_COGNITIVE_REFRAMING_RESEARCH_NOTE_20261101.md)
