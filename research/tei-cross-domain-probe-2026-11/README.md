# TEI 業務外領域probe（CSW・VSクマ）

CSW（Cultural Substrate Weaving）とVSクマ（VS Bear Simulator）を独立した比較対象として、TEIの構造表現・関係・状態・時系列・検証機構がどこまで使えるかを確かめるread-onlyの試作です。研究上の比較記録は[TEIの汎用モデリング能力を業務外領域で確かめる限定実証](../../plan/research/TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md)にあります。

## このprobeがすること

- 2つのTEI Workspaceを読み込み、`validate`と宣言の意味を確認する。
- 既存形式のJSON fixtureを、TEIが宣言した型・列挙・関係に照らして読み取り専用で検査する。
- 対象ごとに独立した期待値と試験を持ち、片方の結果をもう片方へ適用しない。

## このprobeがしないこと

- CSWの連想・異質性・仮説・問い・比較結果を、確定した意味へ変換しない。本文は型付き境界の外に置き、参照とハッシュだけを持つ。
- VSクマのGodot物理、描画、ゲームループ、既存の認知実装を置き換えない。GDScriptを生成せず、実行時依存も作らない。
- 状態機械を製品の意味DSLへ追加しない。遷移表はprobe内のread-only投影であり、`state`・`transition`の製品実装ではない。
- Workspaceへの書き込み、CSW Skillの変更、ゲームへの組込み、製品アーキテクチャの変更を行わない。
- 実データ、実セッション、実機性能、マージ判断を扱わない。

## 構成

| パス | 内容 |
| --- | --- |
| `csw/workspace/` | CSWの型付き受け渡し境界だけを宣言したWorkspace（`StudyInput`・`StudyRun`・`CandidateFragment`・`AuthorDecision`） |
| `csw/fixtures/` | 合成のhand-off記録。成立例と、拒否される例 |
| `csw/handoff.go` | 記録の厳格な読み取りと参照整合性の検査。本文フィールドを拒否する |
| `csw/handoff_test.go` | WorkspaceとGo側の列挙一致、SKILL.mdの由来区分との対応、本文を型へ入れないこと、参照・順序・作者決定の検査 |
| `bear/workspace/` | VSクマの観測・判断・シナリオ期待の最小宣言 |
| `bear/fixtures/` | `bot-scenario-1`の合成写しと、判断snapshotの成立例・範囲外例 |
| `bear/contract.go` | 判断の境界検査、wire形式の独立実装、シナリオ期待の投影 |
| `bear/contract_test.go` | 実行時定数との列挙一致、上流ファイルへの転記照合、境界値、期限、wire往復、シナリオ検査 |
| `bear/statemachine.go` | 回復状態機械のread-only遷移表。到達不能・デッドエンド・未知端点・優先度衝突・global遷移のスコープ違反を診断する |
| `bear/statemachine_test.go` | 上流ファイルへの転記照合、正例の構造証拠、4件のmutation試験 |

## 実行

```bash
go test ./research/tei-cross-domain-probe-2026-11/... -count=1
go test ./research/tei-cross-domain-probe-2026-11/... -run '^$' -bench . -benchtime=200x
go run ./cmd/tei validate -workspace research/tei-cross-domain-probe-2026-11/csw/workspace -json
go run ./cmd/tei validate -workspace research/tei-cross-domain-probe-2026-11/bear/workspace -json
```

上流ファイルへの転記照合（`TestUpstreamAnchorsResolveInTheCitedFiles`）は、兄弟のVSクマチェックアウトが届く環境でのみ実行します。既定では`<repository>/../vs-bear-simulator`を探し、`VS_BEAR_REPO`で上書きできます。届かない環境ではskipします。skipは「転記が現行版と一致すること」の証拠ではありません。

## 対象版

| 対象 | 確認した版 | 備考 |
| --- | --- | --- |
| TEI | `b8cbd432a` | このprobeの分岐元。`origin/main` `b987b38e7`での再実行でも試験と宣言数は不変 |
| CSW | `73efa57ef66a` | 3 Skillと`plan/`の研究文書を参照。実行時コードは持たない |
| VSクマ | `5dae4c873e39` | `cognition/bear/*.gd`と`tests/fixtures/bot_scenario_smoke.json`を参照。`origin/main`の`a7d0cbae9`で対象pathに差分がないことを確認 |

## 期待値の由来

- CSWの列挙と関係は、CSWの`SKILL.md`が求める由来の区別（原稿・体系由来・接触から生まれた読み・著者の判断）だけを写した。分類は本文から計算しない。
- VSクマのwire例`maintain|+045|+08|+12|10|88`、意図4種、境界値、期限、シナリオのassertion名は、上記版のGDScriptとfixtureに書かれた事実を写した。Godotを起動して取得した値ではない。
- `bear.decision_source`は`BearTypedDecision.source`が実際に書く値（`none`・`typed_wire`・`jev-shaped-rules`・`failure-aware-rules@0.1.0`・transport既定の`async-wire`）です。`local-typed-hybrid@…`・`async-typed-hybrid@…`はcontroller provider IDで、別のフィールドです。transportは新しい`backend_id`を宣言できるため、未記録のproducerはerrorではなくwarningとして扱います。

このため、上流が変わったときにprobeが自動で追随することはありません。列挙の一致は試験で検出しますが、境界値・wire形式・fixtureの意味は、上流の変更を確認して手で更新する必要があります。転記照合のanchor試験は、この手順を機械化できる範囲を覆います。
