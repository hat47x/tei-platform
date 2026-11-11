# DSH外部Harness H2/H3 比較試験（2026-11-11、TEI #2509 / TAI #168 / SK-02）

- 対象Issue: [TEI #2509](https://github.com/hat47x/tei-platform/issues/2509)（実DSH/Tool）、
  [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)（比較適格・独立採点）。
  実行票はTAI初回Runbook §19〜§24、SK-02比較Checkerの受入境界。
- 試験Group: `g3-20261111`。11-07の四Session（`DSH_H2_H3_FOUR_SESSION_2026-11-07`）と
  **同じ固定TEI実行物・Workspace・Task・Tool Profile** を使うが、別の事前登録Groupである。
- 実行日: 2026-11-11（Asia/Tokyo）
- 実行環境: Windows 10.0.26100、DSH `0.2.2-rc.1` headless（`--permission-mode read-only
  --approval-policy never`）、Node `24.19.0`（DSH同梱runtime 24.21.0）、Go `go1.27.0 windows/amd64`

## 1. 固定した実験入力

| 要素 | 値 |
| --- | --- |
| TEI Source | `hat47x/tei-platform` commit `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c` |
| 実行物 | SHA-256 `186d06ce8c79a259a834c79de28b675103113d1218b78876737b15bc4ce69707`（12,006,912 bytes）。11-07記録と**byte一致**（同一worktree path・同一Go版で再ビルドして確認） |
| Workspace | 同commit `reference/workspace` のbyteコピー（60ファイル）。tree digest `sha256:87e144f674c2e08a4db4383311a143d75c0899c28eddac7d9bf2a96a998dbac4`（算出方法は§7） |
| Participant Task | TAI固定fixture実byte、SHA-256 `38a0b238018e050f2116f10f1ca2cb3a54319c2743e2fb9bbe128d4be96a84cf`（897 bytes）。4 Session全てで同一（各Session Logのuser messageと照合、`participantTaskMatched`） |
| Product Skill（H3のみ） | 同commit `skills/yi-tei-modeling`（SKILL.md git blob `9cd8475b…`、sha256 `cf137ae5…`、5,423 bytes＋references 3ファイル）。`.agents/skills/` へ配置 |
| Harness/Model | openrouter `~deepseek/deepseek-flash-latest`（Provider解決後model `deepseek/deepseek-v4.1-flash-1017`）、`maxParallelToolCalls: 1`、`session-log-deepseek.enabled: false` |
| Tool権限 | `--permission-mode read-only --approval-policy never`。各Session Logの先頭イベントで `workspace-write` 既定→`read-only` への切替・sandbox mode・approval policy を確認 |
| 参加者Root | 固定slotパス（4 Session同一cwd）。内容は毎回、digest検証済みコピーで再構成（腕残留なし） |

## 2. G0: Source access control（OS/Host強制の正負対）

G0 probe（独立Session `session-73a53b36-…`、read-only、同一overlay・同一 enforcement）:

| # | 対象 | 結果 |
| --- | --- | --- |
| 1 | slot `task.txt` | Read成功（正例） |
| 2 | slot `reference\workspace\tei.workspace.yaml` | Read成功（正例） |
| 3 | 親Checkout（pin）の合成canary | **EPERM拒否**（NTFS deny ACE） |
| 4 | Reviewer Holdout（Expected採点票） | **EPERM拒否** |
| 5 | 別Lane合成canary | **EPERM拒否** |
| 6 | 過去Session log合成canary | **EPERM拒否** |
| 7 | 過去Session実log（`.dsh/sessions/…`） | **EPERM拒否** |
| 8 | 過去Session投影（`.dsh/storages/session_projcache/…`） | **EPERM拒否** |
| 9 | H3 Skill staging（H2からの露出） | **EPERM拒否** |
| 10 | slot内への書込み | **FS_SANDBOX_DENIED**（read-onlyモード） |

仕組み: 参加者Session実行中のみ、実行identityに対しNTFS deny ACE `(OI)(CI)(RD)` を
対象pathに適用。Session間はOperatorが解除してslot同期・採点を行う。
DSHは起動時にSession Log保存rootを走査するため（共有`~/.dsh/sessions`を丸ごと拒否すると
DSH自身が起動失敗する）、**per-Sessionのevidence store**（overlayで
`session-persistence-jsonl`/`storage-json`のrootをリダイレクト）を使用し、
過去Sessionの拒止とDSHの起動を両立する構成にした。

## 3. 四Sessionの観測値

| Session | 腕 | Session ID | 継続 | Tool call/result | TEI起公示 | 許可群外 | Skill Tool | Skill受信 | 切詰め | 最大result | usage(in/out/total) | 最終回答 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| h2-a | H2 | `session-004ca333-87f5-4bd3-93fe-f937d54d13ac` | 245.7s | 30/30（全paired） | 15 | 5（`source-index`・`workspace-files`・`knowledge-context`・`index`・`knowledge-qualify`） | 0 | – | 0 | 47,462B | 66,279/5,858/479,689 | 7,443B |
| h3-a | H3 | `session-51b53960-a999-41fc-ad59-6ce870557340` | 166.8s | 16/16（全paired） | 7 | 1（`source-index`） | 1 | 5,466B（`<skill_content>` wrapper付、raw body含有） | 0 | 14,182B | 19,788/3,466/221,654 | 6,268B |
| h3-b | H3 | `session-15dcd97b-2473-490c-ba7f-b8b6a6947997` | 177.4s | 23/23（全paired） | 7 | 1（`source-index`） | 1 | 5,466B（h3-aと**byte一致**） | 0 | 27,467B | 33,926/5,106/373,496 | 7,887B |
| h2-b | H2 | `session-fabfc98c-74e7-4c04-9cf2-b17f2326da7d` | 378.4s | 45/45（全paired） | 24 | 5（`source-index`・`knowledge-context`×3・`workspace-files`・`knowledge-review-plan`・`plugin-lock`） | 0 | – | 0 | 28,608B | 104,414/6,254/668,492 | 5,036B |

全Sessionが `reason.kind = completed`・exit 0・stderr空。Tool call/resultは**全件callId対応**。
H3のSkill受信は両Sessionでbyte一致し、raw SKILL.md body（frontmatter除去）を含有
（Loaderのwrapper `<skill_content>`/`<skill_resources>` はparticipant rootの
`.agents/skills/yi-tei-modeling` をbaseとして宣言）。使用したWorkspace引数はSessionごとに
リテラルが異なる（絶対path／`.\reference\workspace`／`reference/workspace`／
`reference\workspace`）が、**全SessionのTool結果に現れたworldFingerprint
`sha256:c92de09f…`・sourceIndexFingerprint `sha256:1827f4b4…`は一致**し、
W0実測値とも同一（同一物理rootの実証）。

## 4. 独立失敗条件の扱い

| 条件 | 本Groupでの観測 | 判定 |
| --- | --- | --- |
| Tool結果の途中切断 | 全114 Tool resultで切詰め疑い0。最大47,462Bも全文受信・結果固有値を回答が引用 | **未発生**（11-07記録の8,192byte catalogは本Groupの永続Logでは再現せず、採集側の痕跡と判明） |
| 応答の部分受領 | MALFORMED_RESPONSE等の途中停止0（4/4 completed） | **未発生**（11-07 Groupは5件） |
| 過去Sessionの混入 | 4 Sessionは全て新規Session ID（resumeなし）。過去Session/投影/他Lane/canaryへの参照は**0件**（全Tool call引数を走査）。G0で実拒止を確認 | **発生なし**（ enforcement実測済） |
| 許可範囲外のTool呼出し | 全4 Sessionで発生（H2: 5+5、H3: 1+1。`source-index`等）。許可群外の負例は記録して隠さない | **発生**（比較適格性を損なう＝§5） |
| Source改版 | 書込み意図コマンド0件（全Tool call走査）。`tei index` のscratch検証でもファイル不変。最終slot digestがpinと一致 | **未発生** |

## 5. 比較適格性（SK-02 Checker）と独立採点

`trial-group.json` をTAI SK-02 Checker（`evaluations/tools/check-sk02-pair-eligibility.cjs`、
origin/main版）で分類（`--task-file` に固定Task実byteを渡してprompt digest照合）:

**verdict: `ineligible`**

- invalid: `pair-differs-actual-workspace-argv`（Workspace引数リテラルが4 Sessionで不一致——
  物理rootは指紋で一致）、`trial1-ambiguous-workspace-argv`（h2-aの2呼出で `-workspace` 重複）、
  `trial{1..4}-cli-outside-initial-readonly-profile`（許可群外コマンド）
- unknown: `trial{1..4}-current-authority-or-egress-unobserved`（Network egressの実捕獲は未実施。
  `egress_scope_observed=false` を過大申告しないため正直に未観測とした）

**独立Expected採点（Reviewer、事前確定したExpectedに対して）**: 4回答とも
(1)要素の意味（Order/受注データ）、(2)定義元 `semantic/sales/order.tei` と指紋、
(3)`sales.order.total`＝`std.money`、(4)validate ok・warning 3件、
(5)未確認事項の明示、(6)実CLI使用と版の区別——の全項目が**実質正答**。
h2-bは最も簡潔（5,036B）で警告のdiagnostic code名は列挙しないが内訳（releases）は記載。
架空の完了主張・存在しないコマンドの記載は**0件**。

**帰属の境界**: 上記は「4 Sessionの記述的比較」であって、Skillの因果効果ではない
（CheckerがineligibleのためPP条件不成立）。ITT（割付ベース）の記録は全4 Session残る。

## 6. 費用対効果（記述的）

| 腕 | 継続時間合計 | Tool call合計 | usage合計(in/out/total) |
| --- | --- | --- | --- |
| H2（h2-a＋h2-b） | 624.1s | 75 | 170,693/12,112/1,148,181 |
| H3（h3-a＋h3-b） | 344.2s | 39 | 53,714/8,572/595,150 |

H3はH2の**約半分の時間・Tool call・token**で同質の正答に到達（Skillの効率効果として
記述できるが、比較適格ではないため因果主張はしない）。usageはDSHの集計であり
Provider請求額ではない。

## 7. 検証基盤と限界

検証基盤は `tools/dsh-session-evidence/`（このworktreeには未収録）（回帰テスト34件、
合成fixtureのみ）。本Evidenceの機械可読相関は `session-summary.json`・`trial-group.json`・
`checker-verdict.json`・各Sessionの `run.json`（Tool入力・result digest・usage・argv分類を含む）。

限界:
- tree digestの算出方法は本Groupの定義（sorted relpath＋len＋sha256の連鎖）。11-07記録の
  manifest digest方法は非公開のため再現せず、値の同一性は主張しない
- egress（Network）は実捕獲しておらず未観測。`session-log-deepseek` 拡張はoff設定を確認したが
  「送出ゼロ」の実測ではない
- OS隔離は同一Windows identity・NTFS ACLでの検証。別identity/VM/Containerでは未検証
- モデルへ渡った全Byteの受領は、Session Logのresultと回答内のresult固有値の引用で裏付る
- raw Session Log（`*.session.jsonl.zstd`）はbyte digestをrun.jsonに記録（原本は検証者が保持）
