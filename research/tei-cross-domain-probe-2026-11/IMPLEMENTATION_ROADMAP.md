# 業務外領域probe：後続実装者のための具体化ロードマップ

親Issue：[#4193](https://github.com/hat47x/tei-platform/issues/4193)。研究文書：[TEIの汎用モデリング能力を業務外領域で確かめる限定実証](../plan/research/TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md)。

本書類は、研究・設計案の段階で止まったwork packetを、「実装者が最後まで作業できる具体的な手順」へ変換します。**製品アーキテクチャの変更・CSW Skill本文の手順変更・VSクマのGodot実装変更は行いません。** 判断材料の整備と、その先の実行可能な形までを対象とします。

## 1. 決定済みの事項（触らない）

| 項目 | 決定 | 記録 |
| --- | --- | --- |
| CSW-01 | **残す**（任意の道具として）。対価は`skills/writing-flow/references/handoff-card.md`への補足1節のみ。 | [OWNER_DECISIONS.md](./OWNER_DECISIONS.md) |
| BEAR-01 | **保留**。境界値・wire形式・fixtureの同期手順が決まり、証拠ができてから再検討。 | [OWNER_DECISIONS.md](./OWNER_DECISIONS.md) |
| 検証語彙 | 製品DSL変更を提案しない。研究物としてVOC-01a-dの増分実装を想定。 | [VALIDATION_VOCABULARY.md](./VALIDATION_VOCABULARY.md) |
| 開いた集合 | 暫定案「列挙＋warning」を採るが、製品化は別判断。 | [OPEN_SET_POLICY.md](./OPEN_SET_POLICY.md) |
| 昇格 | 現状は`research/`が適切。`tools/`への移す条件は未充足。 | [PLACEMENT_POLICY.md](./PLACEMENT_POLICY.md) |

## 2. 未着手のwork packetと次の一手

### 2.1 CSW-01（残す判断の具体化）

**状態**：決定済み。ただし、CSWリポジトリの`skills/writing-flow/references/handoff-card.md`は**git未追跡**であり、`73efa57ef66ae4d53db3ff9bb63c927f624f44a8`および現在の`main`にも存在しません。

**次の一手**（順番に実行）：

1. **CSWリポジトリで`skills/writing-flow/references/`を公開する**（著者の判断）
   - ディレクトリをgitに追加し、追跡可能にする
   - これが最初の一手であり、著者の判断が必要
2. **補足1節をcommitする**
   - ファイル：`skills/writing-flow/references/handoff-card.md`
   - 操作：既存ファイルの末尾に§付録Aの文案を追記する（置き換えない）
   - 文案：[OWNER_DECISIONS.md](./OWNER_DECISIONS.md) 付録Aを参照
3. **READMEの参照を更新する**
   - `skills/writing-flow/README.md` または `skills/writing-flow/SKILL.md` に、補足1節への参照を1行追加する
   - 任意の道具であることを明記する（必須にしない）
4. **probeのanchor試験を更新する**（TEIリポジトリ）
   - `csw/anchor_test.go` に、補足1節のanchorを追加する
   - 追加するanchor：`skills/writing-flow/references/handoff-card.md` 内の「由来の区分を使う場合」節の見出しと、4つの帰属値の説明文
   - 試験は`snippets`の形で書くか、ファイル全体の`strings.Contains`で確認する

**受け入れ条件**：
- `skills/writing-flow/references/handoff-card.md` がgitで追跡されている
- 補足1節が既存の本文を置き換えていない
- 4つの帰属値の説明が`SKILL.md`の4区分と一致している（anchor試験で確認）

**非目標**：Skill本文の手順変更、Schema導入、CI追加、ラベル作成。

**実施例（コマンド）**：
```bash
# CSWリポジトリで
git add skills/writing-flow/references/handoff-card.md
git commit -m "docs(writing-flow): 受け渡しカードに由来の区分の補足を追加（任意）"
git push origin main

# TEIリポジトリでanchorを追加
cd tei-platform
go test ./research/tei-cross-domain-probe-2026-11/csw/ -count=1 -run Anchor -v
```

### 2.2 BEAR-01（保留の具体化）

**状態**：保留。製品コードは変更しない。

**次の一手**（いつでも再検討できる状態を維持）：

1. **同期手順の草案を作成する**（VSクマリポジトリ）
   - 対象ファイル：`docs/BEAR_TYPED_COGNITION_BOUNDARY.md` の検証節
   - 追加内容：列挙の一致試験に加えて、境界値・wire形式・fixtureの同期を確認する手順
   - 専用runnerは作らない
2. **上流変更の追跡方法を決める**
   - 方法案A：probeのanchor試験をCIで実行する（兄弟チェックアウトが必要）
   - 方法案B：上流のcommitを定期的に確認し、差分が出たら手で更新する
   - 方法案C：VSクマのreleaseノートにTEI投影の更新を記載する
3. **二重管理コストの測定**
   - GDScript・TEI・probeの3箇所で同一の値を管理するコストを、実際の変更イベントで記録する
   - 1回の変更が何ファイル・何行に波及するかを数える

**受け入れ条件**：
- 上記1〜3のうち最低1点が実施され、証拠（文書または計測）が残っている
- その時点で再検討し、受け入れるか・保留を続けるかを決定する

**非目標**：GDScript生成、ゲームループへの依存、既定providerの変更。

**明確に禁止**：現時点でTEIを`internal/semantic`へマージしたり、VSクマの実行時契約をTEIに置き換えたりしない。

### 2.3 VOC-01（検証語彙の増分実装）

**状態**：設計案（製品DSL変更を提案しない）。

**次の一手**：

1. **VOC-01a：属性宣言内の`range`・`finite`キーワードをパーサに追加する**
   - 対象：`internal/syntax/parser.go`
   - 範囲：属性宣言のパース時に`range`・`finite`を読み込む
   - 非目標：constraint分離（C案）は実装しない
2. **VOC-01b：バリデータに範囲チェックを追加する**
   - 対象：`internal/validation/`の新規または既存ファイル
   - 動作：`range`で指定された区間内にあることを確認する
3. **VOC-01c：workspaceの`tei validate -json`に検証結果を出力する**
   - 対象：`cmd/tei/`のvalidate出力
   - 動作：`positive`・`relation-unique`に加え、`range`・`finite`の結果をJSONに含める
4. **VOC-01d：CSW・VSクマのprobeで検証語彙を試す**
   - 対象：`research/tei-cross-domain-probe-2026-11/`のworkspace
   - 動作：`heading_bias_rad`等に`range`・`finite`を追加し、`tei validate`が通ることを確認

**受け入れ条件**：
- `go test ./internal/... -count=1` がPASS
- `go run ./cmd/tei validate -workspace research/tei-cross-domain-probe-2026-11/bear/workspace -json` が`range`・`finite`を出力
- 既存の`positive`・`relation-unique`の動作が変わらない

**非目標**：製品DSLの`constraint`キーワード導入、時間窓の実装。

### 2.4 OPEN-01（開いた集合の規約の製品化判断）

**状態**：設計案（製品化判断は別）。

**次の一手**：

1. **TEIコアのIssue #2533 でwarning severityの設計を議論する**
   - 現在の`validation`がerror中心である問題を、`warning` severityで緩和するか
   - 暫定案：「開いた集合は列挙＋warning」を採る
2. **パーサに`warning` severityのパースを追加する**
   - 対象：`internal/syntax/parser.go` のvalidationパース部
   - 範囲：`validation`キーワードにseverityを追加する（例：`validation foo warning`）
3. **workspaceの`tei validate -json`にwarningを出力する**
   - 対象：`cmd/tei/`のvalidate出力
   - 動作：errorとwarningを分けて出力する
4. **probeの`ValidDecisionSources`をwarningで実行する**
   - 対象：`bear/contract.go` の`Weight`関数
   - 動作：未記録のproducerをerrorではなくwarningとして報告する

**受け入れ条件**：
- `go test ./internal/... -count=1` がPASS
- `go run ./cmd/tei validate -workspace research/tei-cross-domain-probe-2026-11/bear/workspace -json` がwarningを出力
- 既存のerror severityの動作が変わらない

**非目標**：`unknown`の標準値化、列挙型の廃止。

### 2.5 TRACE-01（anchor試験のCI接続）

**状態**：anchor試験は実装済み。CI接続は未実施。

**次の一手**：

1. **CIで外部チェックアウトを取得する手順を書く**
   - 対象：`.github/workflows/` の既存ワークフローまたは新規
   - 内容：`actions/checkout`でVSクマとCSWを取得する
   - 注意：skipを避けるため、必須経路に置くか、明示的に「skipは許容」と書く
2. **anchor試験をCIで実行する**
   - コマンド：`go test ./research/tei-cross-domain-probe-2026-11/... -count=1 -run 'Anchor|Transcription'`
3. **結果の記録**
   - CIのartifactにanchorの照合結果を保存する
   - 失敗した場合、上流の変更を読んでprobeを更新する

**受け入れ条件**：
- CIがanchor試験を実行し、結果が記録される
- 上流のcommitが変わっても、CIが検出する（anchorが不一致の場合に失敗する）

**非目標**：ネットワーク経由の上流取得、自動追随。

### 2.6 SM-01 / SM-02（状態機械投影の製品化判断）

**状態**：read-only投影は完了。製品化判断はTEI中核。

**次の一手**：

1. **TEIコアのIssue #4193 で状態機械の表現を議論する**
   - 現在のDSLに`state`・`transition`を追加するか
   - 追加する場合、優先度・global・付随変数の保持をどう表現するか
2. **状態機械のパーサを追加する**（製品化が決まった場合）
   - 対象：`internal/syntax/parser.go`
   - 範囲：`state`・`transition`・`guard`のパース
3. **状態機械のバリデータを追加する**
   - 対象：`internal/validation/`
   - 範囲：到達不能・デッドエンド・重複ID・未束縛source/targetの検査

**受け入れ条件**：
- `go test ./internal/... -count=1` がPASS
- `go run ./cmd/tei validate -workspace research/tei-cross-domain-probe-2026-11/bear/workspace -json` が状態機械を検証

**非目標**：Godotランタイムへの接続、UI実装。

## 3. 依存関係図

```
CSW-01 ──(著者がreferences/を公開)──> CSW-01 実装 ──> anchor更新
BEAR-01 ──(同期手順の決定)──> BEAR-01 再検討
VOC-01a ──> VOC-01b ──> VOC-01c ──> VOC-01d
OPEN-01a ──> OPEN-01b ──> OPEN-01c
TRACE-01 ──(CI整備)──> anchor試験の常時実行
SM-01/SM-02 ──(#4193判断)──> 状態機械パーサ
```

## 4. 実装者が最初に読むべきファイル

| 目的 | ファイル |
| --- | --- |
| 全体像 | `plan/research/TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md` |
| 今回の決定 | `research/tei-cross-domain-probe-2026-11/OWNER_DECISIONS.md` |
| CSW補足文案 | `OWNER_DECISIONS.md` 付録A |
| anchorのルール | `research/tei-cross-domain-probe-2026-11/ANCHOR_POLICY.md` |
| 状態機械の不足 | `research/tei-cross-domain-probe-2026-11/bear/STATE_MACHINE_REVIEW.md` |
| 開いた集合の規約 | `research/tei-cross-domain-probe-2026-11/OPEN_SET_POLICY.md` |
| 検証語彙の設計 | `research/tei-cross-domain-probe-2026-11/VALIDATION_VOCABULARY.md` |
| 昇格基準 | `research/tei-cross-domain-probe-2026-11/PLACEMENT_POLICY.md` |

## 5. 禁止事項（やってはいけないこと）

1. **CSW-01の補足を、著者が`skills/writing-flow/references/`を公開する前にcommitしない**。未追跡ファイルの公開は著者の判断に委ねる。
2. **BEAR-01を保留中にVSクマの製品コードを変更しない**。契約の写しは受け入れられていない。
3. **現行DSLの`validation`キーワードを削除・改名しない**。VOC-01は追加案であり、既存の`positive`・`relation-unique`は不変。
4. **anchor試験をCIでskipする運用にしない**。skipは「検査していない」の記録であり、一致の証拠ではない。
5. **CSWで成立した方式をVSクマに自動適用しない**。両者の評価は独立であり、CSW-01の「任意の記録」をVSクマの実行時契約に持ってきてはいけない。

## 6. 未実施（NOT RUN）

- CSW-01の補足commit（著者の公開判断待ち）
- BEAR-01の同期手順の草案（保留中）
- VOC-01a〜01dの製品DSL実装（設計案のみ）
- OPEN-01のwarning severity製品実装（設計案のみ）
- TRACE-01のCI接続（A3の別判断）
- SM-01/SM-02の状態機械パーサ追加（#4193の判断待ち）
- Godot起動・実行時観測・別hostでの性能測定
