# E0実測 — 観測者用プロトコル

## 準備

[#1168](https://github.com/hat47x/tei-platform/issues/1168) の独立実測用です。参加者へは `tasks/E0-participant.md` の内容のみ提示し、`examples/first-workspace/README.md` の完成手順や既存E2Eテストの差分を先に知らせないでください。Git SHA、環境、利用できるドキュメント、開始前の関連知識、LLMの利用モデルとセッション継続条件を記録します。

同じ初期状態のWorkspaceを使い、必要ならSnapshotを保存してください。開始と終了は実際のUTC timestampを記録します。中断した場合は `finishedAt` を省略し、**未計測の所要時間を0秒にしない**でください。

## イベント収集

操作ごとに記録し、後で `tools/first-value-ux-audit` へ `observed` Session JSONとして渡します。

- 実CLI呼出し1回につき `command` を1件。exitCodeと秘匿済みstdout/stderrへの `evidenceRef` を残す。
- Authoritative Sourceの編集操作1回につき `source-edit` を1件。ファイルへの具体的な編集内容は、必要に応じ別の秘匿済み差分で保持する。
- 手入力欄を実際に埋めた操作について `manual-fields` と個数を記録。自動生成による値・単なる出力文字列は数えない。
- 修復に必要だった作業を `recovery` として記録。ただし通常の編集と両方に数えた場合は生の操作ログで判別できるようにする。
- E0では`independent-read`を記録しない。実データへの保存を観測したと主張しない。

`qualification` は常に `unverified` です。この計測器は参加者のタスク成否、実際のRuntimeやProvider、出力Evidenceの真正性を検査しません。別紙に記述した受入確認（属性追加、Source位置、修復、未実証事項の適切な説明）と区別してください。

## 2回の独立実測の後

初見開発者・初見LLMの少なくとも2件（条件を区別して）について、失敗/未完了も含め、合成Fixtureと混ぜずに集計します。時間・操作数・修復回数・Source位置特定の成功率を比較し、観測された失敗を具体的なIssueや回帰テストへ戻します。推測や架空の平均値で穴埋めしないでください。

E1（Data-backed実行）は [#1127](https://github.com/hat47x/tei-platform/issues/1127) の実装で`Source → 実行 → 別read`のEvidenceが得られるまで、`not-executed`として記録します。E0の`validate`成功をE1達成に読み替えません。
