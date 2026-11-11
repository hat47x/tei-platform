# 外部Repository転記の照合下限と証跡（TRACE-01）

本ファイルは、業務外領域probe（`research/tei-cross-domain-probe-2026-11/`）が**外部Repositoryから手で写した値**を、どこまで機械的に照合するかを定め、実際に照合した証跡を記録します。

親Issue：[#4134](https://github.com/hat47x/tei-platform/issues/4134)。研究文書：[TEIの汎用モデリング能力を業務外領域で確かめる限定実証](../../plan/research/TEI_CROSS_DOMAIN_PROBE_CSW_VS_BEAR_2026-11.md) §9 TRACE-01。

## 1. 何を問題にするか

probeは、TEIの意味DSLが写せる範囲を調べるために、**CSWとVS Bear Simulatorの値を手で転記**しています。この転記には、probe自身の試験では検出できない失敗モードがあります。

| 失敗モード | probe内の自己一致試験で検出できるか |
| --- | --- |
| probeの定数とGo側の受理値集合がずれる | **できる**（`TestWorkspaceEnumsMatchRuntimeContract` 等） |
| TEIの宣言とprobeの定数がずれる | **できる**（Workspaceを読み込んで比較する試験） |
| **上流の値が変わったのにprobeが追随していない** | **できない**（probeは上流を読まない） |
| **最初から転記が誤っている**（上流に無い値を書いた／別フィールドの値を書いた） | **できない**（自己一致しているので全試験が通る） |

実例：初回の実装では `bear.decision_source` に `local_typed_hybrid`・`async_typed_hybrid` を書いていましたが、これらは `provider_id()` の値であり `BearTypedDecision.source` ではありませんでした。**TEI宣言とprobe定数が互いに一致していたため、全試験がPASSしていました。** CSWでも帰属ラベルが3値で、`SKILL.md` の4区分のうち1つが欠けていました。

したがって、自己一致試験だけでは転記の正しさを確認できません。**上流の原文と照合する経路**が必要です。

## 2. 照合の下限（本probeが守る規則）

### 2.1 すべての転記値に anchor を持つ

転記した**列挙値・境界値・スキーマ名・fallbackラベル・遷移理由**のそれぞれに、`(file, literal, why)` の3つ組を与えます。

- `file`：上流Repository内の相対パス。probeのファイルから見た位置ではなく、上流のルートからの位置。
- `literal`：上流ファイルに実際に現れる文字列。**加工しない**。整形やコメント除去をした文字列を anchor にしない（照合が上流の整形に依存してしまう）。
- `why`：その literal がprobeの何を支えているか。1〜数語。

### 2.2 anchorは「上流に在る」ことだけを確認する

anchorの試験は `strings.Contains(source, literal)` で存在を確認します。**意味の正しさは確認しません。** たとえば `provider_speed <= 0` を `provider_speed > 0` に書き換えても、その文字列が上流のどこかに在れば通ります。意味の照合は、人が上流を読む review で行います（SM-01）。

この限界は意図的です。機械照合の目的は「上流が変わった／転記が上流に無い」を検出することで、意味の受理ではありません。

### 2.3 anchorが解決できない環境は skip し、skip を一致の証拠にしない

兄弟チェックアウトが届かない環境（CI、他の作業者のマシン）では、anchor試験は `t.Skip` します。

- skip は「転記が現行版と一致すること」の証拠では**ありません**。
- したがって、skip した実行結果を照合証跡として記録しません（§4の表で `skipped` と明示する）。
- 環境変数 `VS_BEAR_REPO` / `CSW_REPO` で上流の場所を指定でき、未指定時は `../../../..` 配下の兄弟ディレクトリを探します。

### 2.4 覆う範囲：本体値だけでなく fallback ラベルも

列挙の anchor が「本体の値」だけを覆うと、fallback ラベル（`unaware`・`unknown`・`else "unknown"`）が抜けます。fallback は機械の状態ではありませんが**記録に現れる値**であり、上流が変わればprobeの列挙も追随する必要があります。したがって anchor に含めます。

### 2.5 覆わないもの

- **上流の内部実装**（計算式・分岐の順序そのもの）。probeは遷移表と理由だけを写し、step の実装は写さない。
- **上流のドキュメントの散文**（`docs/*.md` の説明文）。probeが写すのは値であって説明ではない。ただし CSW の帰属4値は例外で、`SKILL.md` の区分の一文を逐語で anchor にした（区分そのものが意味の単位だから）。
- **上流の意味の妥当性**。§2.2 のとおり。

## 3. 実装

| 対象 | 実装 | 試験 |
| --- | --- | --- |
| VSクマの転記値 | `bear/contract_test.go` の `upstreamAnchors`（31件） | `TestUpstreamAnchorsResolveInTheCitedFiles` |
| VSクマの遷移理由 | `bear/statemachine.go` の `TransitionReasons`（11件） | `TestRecoveryMachineTranscriptionMatchesUpstreamAnchors` |
| CSWの帰属4値 | `csw/anchor_test.go` の `cswAttributionAnchors`（4件） | `TestCSWAttributionAnchorsResolveInSKILLMD` |
| CSWの節見出し・任意性の根拠文 | `csw/anchor_test.go` の `cswSectionAnchors`（2件） | 同上 |

追加の規則：`TestCSWAttributionAnchorsMatchDeclaredValues` が、anchor が名指す値とprobeの受理値集合を突き合わせます。anchor が受理されない値を名指すと、その anchor は記録できないラベルを検査していることになります。

## 4. 照合証跡

| 項目 | 値 |
| --- | --- |
| 実施日 | 2026-11-11 |
| 実施host | Windows 11、Intel Core i7-8750H、Go 1.27.0 windows/amd64 |
| 上流（VSクマ）のcommit | `2a7e2990bb22da57deafb2aedfecb9a654094048` |
| 上流（CSW）のcommit | `73efa57ef66ae4d53db3ff9bb63c927f624f44a8` |
| 照合したanchor数 | VSクマ 31 + 遷移理由 11 + CSW 6 = **48件** |
| 照合結果 | 48件すべて**解決**（skip 0件） |
| 実行コマンド | `go test ./research/tei-cross-domain-probe-2026-11/... -count=1 -run 'Anchor|Transcription' -v` |
| skipした実行 | なし（両方のチェックアウトが届いた） |
| probeのcommit | `7705318ff` |

照合した時点のprobe試験は **36件 PASS**（bear 26・csw 10）です。

**注意**：この表は1回の照合記録です。上流が進んだ場合、または別の host で実行した場合の結果を代弁しません。§2.3 のとおり、skip した実行は照合の証拠になりません。

### 4.1 証跡自体にも誤りがあった（2026-11-11 追記）

上記の表は、**VSクマのcommitを `2f7e2990bb22da57deafb2aedfecb9a654094048` と記録していましたが、このcommitは実在しません。** 正しい値は `2a7e2990bb22da57deafb2aedfecb9a654094048` です（`git rev-parse HEAD` で確認）。

誤りは**anchor試験の結果には影響しません**（anchorは`literal`の存在だけを確認し、commit SHAは見ません）。しかし証跡の価値は「どの版に対して照合したか」に依るため、誤ったSHAは証跡を再現不能にします。発見の経緯は、本ファイルの著者が後続の判断作業で上流のcommitを調べ直したことによるもので、機械的な検査ではありません。

**教訓**：commit SHA も転記対象です。`git rev-parse HEAD` で確認した値を書き、書き終わった後に存在確認（`git cat-file -t <sha>`）を行うべきです。本ファイルでは4箇所（本表・`README.md`・`PLACEMENT_POLICY.md`・`bear/STATE_MACHINE_REVIEW.md`）を修正しました。

## 5. 未実施（NOT RUN）

- anchor試験の CI 必須検査への接続（A3 の別判断。研究物を CI の必須経路に入れるかは未決）。
- 上流変更への自動追随（Dependabot 相当の仕組みは作らない）。
- ネットワーク経由の上流取得（ローカルの兄弟チェックアウトのみ）。
- 意味の照合の自動化（§2.2 のとおり、人が上流を読む review に残す）。
- 上流の別バージョン・別OS・別Goでの実行。

## 6. 判断が必要な事項

1. **anchor を CI で常に走らせるか。** 兄弟チェックアウトが無い CI では skip するため、CI に置いても何も検査しません。置くなら上流を checkout する手順が必要で、外部Repositoryへの依存がCI設定に入ります。
2. **anchor の数を増やすか。** 現在は「列挙・境界値・スキーマ名・fallback・遷移理由」を覆っています。上流の全定数を覆うかは、費用対効果の判断が必要です（覆うほど上流の些細な変更でprobeが落ちます）。
3. **意味の照合を誰がやるか。** §2.2 の限界を埋めるのは review です（SM-01 が review 記録の票）。review の要求範囲をどう定めるかは未決です。
