# reference-productivity-benchmark（#2018）

[生産性ベンチマーク手順](../../plan/design/PRODUCTIVITY_BENCHMARK_PROTOCOL_2026-10.md)のrunを記録・集計するReferenceです。比較の実行、被験者の割当て、結果の主張は行いません。

```text
go run ./tools/reference-productivity-benchmark/cmd/tei-productivity-benchmark \
  record -input run.json -out runs/

go run ./tools/reference-productivity-benchmark/cmd/tei-productivity-benchmark \
  aggregate -dir runs/
```

## run

```json
{
  "schema": "tei.reference.benchmark-run/v0",
  "taskId": "add-master",
  "method": "tei",
  "runIndex": 1,
  "startedAt": "2026-10-19T00:00:00Z",
  "endedAt": "2026-10-19T00:10:00Z",
  "environment": {
    "os": "linux",
    "cpu": "x86_64",
    "memory": "16GB",
    "network": "offline",
    "model": "none",
    "tools": "go1.26"
  },
  "metrics": {
    "ttfwcSeconds": 60,
    "humanInterventions": 1,
    "revalidationRequired": 4,
    "revalidationRun": 2,
    "acceptedProposals": 1,
    "rejectedProposals": 0,
    "changedFiles": 3,
    "generatedArtifacts": 1
  },
  "completion": {
    "met": true,
    "evidence": ["tests pass"]
  },
  "unresolved": []
}
```

- `method`は`conventional`・`lowcode`・`tei`。
- `runIndex`は1〜3。1タスクにつき各方法3回を集計対象とする。
- `ttfwcSeconds`は開始から最初の動作確認までの秒数。記録したdurationを超えない。
- 人間の介入、再検証の対象数と実行数、受理・却下された提案、変更file、生成物を記録する。
- 完了条件を満たした場合は、evidenceを1件以上持つ。
- 不明点は`unresolved`へ残し、空の成功へ潰さない。

## 集計

`aggregate`は1つのtaskについて、3方法それぞれ3 runを要求します。TTFWC、人間の介入、再検証の実行数・対象数の中央値・最小・最大を出します。

3 runに満たない場合は集計せず、エラーにします。中央値と範囲は記録の要約であり、統計的有意差や一般化を示しません。

## 守っている境界

- runの記録と集計だけを行い、比較を自動実行しない。
- 結果を生成・捏造しない。fixtureと実測を区別する。
- 単一のrunや3 run未満を、生産性の証拠として扱わない。
- 環境・モデル・ツール・完了条件の差を記録し、結果の解釈に必要な文脈を落とさない。
- 手法の優劣を、このtoolだけで判定しない。

## 未実装

- 被験者割当て、順序のランダム化、学習効果の統制。
- 実行時間の自動計測、画面操作の記録。
- 統計検定、信頼区間、欠測の補完。
- 4タスク・3方法の実runと、#2018への結果記録。
