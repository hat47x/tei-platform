# AI Tuning Observation Join — 限定Reference（試験専用）

生成AIによる性能チューニングの前段として、設計ヒント、観測統計、呼出しProvider、現行の開示条件、受領完全性を同じ条件で照合するための**合成反証試験**です。正式なTEI CLI、Product API、Optimizer、Telemetry Collector、Cache、認可基盤ではありません。

- 設計正本：[AI性能チューニング・統計観測](../../plan/design/AI_PERFORMANCE_TUNING_OBSERVABILITY_2026-11.md)。新しい意味正本や契約を作りません。
- TEI CLI、DB、Cache、実Provider、外部AI、Toolの実際の配送は動かしません。input-consistentは**入力内容の一致**に限定され、実呼出し・正しい統計・業務受理・性能改善を意味しません。
- InputはSynthetic JSONだけで、他ファイル・ネットワークの読取りや書込みはしません。

## 実行方法

Python 3.9以降の標準ライブラリのみ使用します。

    python3 -m unittest discover -s tools/reference-ai-tuning-observation -p 'test_*.py' -v

任意の合成JSONを標準入力で読み取る場合：

    python3 tools/reference-ai-tuning-observation/probe.py < synthetic.json

標準出力には事前定義した判定コードのみを出し、入力識別子、SQL、Tenant名、Trace ID、秘密参照を複写しません。外部のConsumerに開示できない場合は、入力が正しいかどうかを問わず固定の withheld 応答だけを返します。

## 照合できる範囲

1. Designの対象/Query/Source Revision/World Generation/取得Scope/選択Providerを、Callの申告された実呼出しProvider/Grant Epoch/統計Windowと照合。
2. 分割受領の連番/件数/digest書式/end markerを照合。ただし**digestの計算や実Tool配送は検証していない**ため、実際の完全受領を証明しない。
3. 統計の標本数・欠測を保持。partialな標本から全Workloadの不存在、p99、改善を推論しない。
4. 同じAttemptでQuery Readが反復された場合も、単に investigate-repeated-reads という**調査候補**に留める。N+1と断定しない。
5. 旧Worldの結果がA→B→Aで無条件に復活しないよう、Source RevisionとWorld Generationを同時に比較。ただし実消費Input Closure・Artifact・Grant再資格化は別の既存契約に従う。
6. 選択B→実呼出しAは、値が同一でも不適合。Provider名・実行結果の自己申告を独立したRuntime証拠として扱わない。

## 製品への接続条件

最初は#2586のFunction→Query→Data Source Binding→**実Provider Read→Rowset**の同一Attempt観測に接続し、実Provider/Trace/Rows/Scopeが取得できた範囲だけを実観測と認定します。次に#3262のDesign Context受領完全性、#2516の開示、#3271のCache/Snapshot、#3278の原子的Trait、#1180の独立Expectedへ接続します。

このReferenceのExpectedは合成入力に独立に固定した**19件**です。実Targetの再現・計測費・採用可能な最適化候補の性能改善については、まだ何も検証していません。実装時は既存のTyped Application API/Provider構成を使い、第二Reader、第二Query Planner、常駐Host、第二認可基盤を設けません。
