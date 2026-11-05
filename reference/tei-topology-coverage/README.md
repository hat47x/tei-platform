# 複合図表の限定不在判定：read-only合成Reference

- **状態**：評価用Reference。新Core型・正式なProjectionWitness/Source Reader・製品API・権限契約ではありません。
- **設計根拠**：[トポロジー・複合図表・開発パイプライン設計 §9〜11](../../plan/design/TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md)。
- **既存責務**：#1120 ModelRef、#1124 Studio Consumer、#2620 標準データ＋Role、#943 Build、#951 Provider、#2516 Disclosure、#267 Source Acceptance。第二Resolver、Host、Source正本は設けません。

## 実験の問い

「図に表示されないので存在しない」と言えるのはいつか。図に含まれる要素集合、Relationの**すべてのroleと端点**、表と図の被覆、閉じたscope、参加した同一版、Source/Plugin/Provider解決、表示省略、opaque処理を別々に扱います。

`project`は同一合成SnapshotからGraphとTableを導出します。Graph上のrelationノード＋role付きEdgeと、表のrole配列は同じ多項関係を表します。`selectView`はview IDをscope付きsubjectへ戻す**候補**だけ返し、InspectやEditは呼びません。`boundedAbsence`は完全性witnessが供給された**閉じた合成集合の範囲だけ**`absent-in-declared-snapshot`を返し、不足・非開示・版混在なら`unknown`です。

この関数はwitnessの**真正性も網羅性も証明できません**。`subjectsComplete: true`などはfixtureの申告値であり、実際のTEIで主張するには独立したSource Reader、Composition/Plugin closure、Disclosure Policy、Target-bound Evidenceによる検証が必要です。外部Builder、OS、DB、ネットワークの未知副作用がないという証明にも使えません。

## 実行

```sh
node --test reference/tei-topology-coverage/coverage.test.mjs
```

Node.js 22以上の組込みtest runnerだけを用います。実験ケースは、同kind/ID異scope、複数View、三項関係role保持、閉じた集合内の条件付き不在、欠落role、Graph/Table欠落、opaque/省略/未固定Reader/開示不明、Source版混在、レイアウトだけの差分、欠落端点、非開示下の現在情報です。

**未実証**：実Go ModelRef/Source Index・TEI CLI出力との結線、Providerによる実閉包証明、実Graph Renderer/Workbench、複数Window/旧DOM、実SHO/AI、外部Effect、認知負荷削減。#1124 M0/M1を代替するテストではなく、#2352 P1の後続評価用合成試験です。実験を製品化する前に、既存型からの変換のlosslessnessと現在のReader/Policy資格を独立検証します。
