# Everything is Data：実TEI CLIでのOperation→Artifact受渡しを同一版で検証する

**状態（2026-11-03未明）**：実製品CLIの**限定的なread-only縦断検証**を追加するDraft。試験実行・exact-head GitHub Actionsの成否は実際のCI観測まで**NOT RUN**。製品のFunction Data Context/実Provider/RDB/API/外部Consumer/Owner業務意味受理ではない。

## 1. 他レーンからの成熟した設計判断と現実の実装状況

親の横断計画は [#2619](https://github.com/hat47x/tei-platform/issues/2619) と、**mainへ統合済み**の [PR #2629](https://github.com/hat47x/tei-platform/pull/2629) の `plan/workstreams/EVERYTHING_DATA_CROSSLANE_HANDOFF_2026-11.md` に置く。この文書は**その第二正本ではない**。H0〜H5のうち、既存製品の読取りで直ちに試せる**「受理済みモデル→標準Operation→設計Artifact→ファイル出力」の一歩**だけを専有する。

- [PR #2587](https://github.com/hat47x/tei-platform/pull/2587) は**統合済み**で、`tei operations`、`tei artifacts`、`internal/operations.SurfaceWorkspace` と `internal/artifact.RenderWorkspace` が現実のGo CLI入口になった。
- [PR #2624](https://github.com/hat47x/tei-platform/pull/2624) も**統合済み**だが、自作PythonのFunction別明示MappingとRule依存の**有限Reference**であり、正式TEI Data Context経路へ接続した証拠ではない。
- [PR #2626](https://github.com/hat47x/tei-platform/pull/2626) は**統合済み**で、`model catalog→inspect→trace→validate` の実Go CLI同一版読取りを専用CIで観測した。今回これを再実装せず、**別の既存入口であるOperationとArtifactの接合**だけを調べる。
- [PR #2627](https://github.com/hat47x/tei-platform/pull/2627) は**統合済み**だが、多Provider/Read Cohortの**合成比較**。実Provider Target/Scopeの保証や二Provider同Snapshotを証明しない。
- [PR #2633](https://github.com/hat47x/tei-platform/pull/2633) はManifest/Consumer検収の**特性検査が統合済み**。ArtifactのDigest/CompleteをConsumerが要するMember/Scope/実配布byteの証明に昇格させない。本試行はこの不足を**実CLIファイル出力の一用途**から狭く確かめる。
- [PR #2639](https://github.com/hat47x/tei-platform/pull/2639) はDSL→実Go Language Pluginの**Draft**であり、今回のCLI試行はその新規Role/DSLを解釈していない。先行の自作Python・Go部分試験を同じE2Eへ合算しない。

## 2. 検証したい受渡しの構造

```text
実TEI checkout（exact HEAD）
  ├─ 実 Go binary: tei operations -workspace W -application A
  │     ├─ app A / modelRevision / worldFingerprint / Rule ID・版
  │     ├─ Operations / Gaps / Conflicts / Summary
  │     └─ authorityLimits（実行権限ではない）
  └─ 実 Go binary: tei artifacts -workspace W -application A
        ├─ inputRevision / actual Artifacts / Manifest / Findings
        ├─ operation-surface artifact: 限定JSON Projection + Unresolved
        └─ tei artifacts ... -out DIR
                 ├─ operation-surface.json（実ファイルbytes）
                 └─ manifest.json（実ファイルbytes）
```

同一Worktreeの実Sourceを複数CLIプロセスで再読取りし、`operations.ModelRevision == artifacts.InputRevision == manifest.InputRevision`、`schemaVersion/application/rule/operations/gaps/summary` の一致、Gap→Unresolvedの内容一致、Manifestと実Artifactのメンバー/ダイジェスト/件数照合、および**ディスクに実書込されたファイルbyte**を確認する。

**重要な制約：ArtifactのOperation Surfaceは意図的に限定したProjectionであり、現在の公開構造では`worldFingerprint/modelRevision/conflicts/authorityLimits`を本文に含めない。** 一致した共通欄から、CLI Object全部とArtifact Contentの意味・開示・現在実行資格の同等性を推定しない。部分Projectionと全Objectの等価性を安易に主張しない。将来のOperation/Artifact出力に変更があれば、対象Consumerの要件を再審査する。

## 3. 実証の判別対と必要な観測

| ID | 実際に呼ぶ入口・入力 | 合格を主張できる最小範囲 |
| --- | --- | --- |
| OA-01 | `tei version -json` とCI checkout `HEAD` | 実行したバイナリがCandidateのexact HEADに帰属し、改変Buildでない |
| OA-02 | `tei operations`、`tei artifacts`を**別プロセス**で同じW/Aに実行 | Model revisionと共通欄の限定Projection一致 |
| OA-03 | Artifact Contentの`gaps`とArtifact `unresolved` | Reason/Subject/Detailの全件対応（順序差を許容） |
| OA-04 | 実Artifact集合とManifest entries | 実際に含むArtifact ID/Digest/Unresolved数が一致。**Consumerが必要とする全Memberを保証しない** |
| OA-05 | `tei artifacts -out DIR`→保存したJSONファイルを独立に読む | 実ファイルのManifest内容／Operation Surface bytesがCLI本体出力と一致 |
| OA-06 | 書出ファイルbyteを局所改変してチェックを再実行 | 改変済みBodyを正しい実ファイルとして受理しない |
| OA-07 | `reference/standard-crud/workspace`、app `ledger` | 合成負例の未解決Gapがあり、Manifest Completeを誤称しない |
| OA-08 | `reference/workspace`、app `order_entry` | 同版Operation `order_entry.update.sales.order`を実CLIが列挙。**実保存／Owner受理なし** |
| OA-09 | 同一Sourceに再度CLI呼出し | 2回のread-only JSON結果の決定性を観測 |
| OA-10 | 正の比較表にないフィールド群 | `authorityLimits`・`conflicts`・`worldFingerprint`は**限定Artifact等価の根拠に流用しない** |

判定は各CLIが返した**実際の観測値**に基づき、上記表以外の網羅性・暗黙的な信頼属性を付与しない。特に`Manifest.Complete`は提示されたArtifactのUnresolvedが0であることを指すだけで、公開契約、Consumer側の期待Member、Data Contextの実取得完全性、Receiptが正しいことではない。

## 4. 追加した実装範囲と検証手段

- `plan/fixtures/crosslane-cli/verify_operations_artifacts.py`：Python3標準ライブラリで**実TEIバイナリを外部プロセスとして実行する**read-only検査器。モデル・Operation・Manifest・Digestの独自実装は行わない。
- `.github/workflows/everything-data-cli-handoff.yml`：PRの**exact head SHA**をCheckoutし、`go.mod`のGo toolchainでCLIをcheckout外の一時ディレクトリにBuildする。`version.vcsRevision`とHEADを突き合わせ、公開Reference Workspace二つで読み、`-out`の保存先も一時Directoryに限定。修正前後のworktreeはクリーンであることを確認する。
- 実CLI呼出し前のPython AST静的検査は標準Libraryだけ。`py_compile`が作るキャッシュや実Datasetの改変は行わない。

この仕事の本質は**成熟した既存の製品入口の接続の資格付け**であって、新DSL/Parser/Provider/Runtime/Workflow Engine/認可Engineの追加ではない。ScriptやCLIに新Action/権限を発行する処理を一切入れない。

## 5. H0〜H5への受渡しと次の未解決

**確実に説明できる対象**：Source→実Go Workspace/既存Operation導出→Artifact生成→保存されたファイルBody、というread-onlyな個別アプリケーションの接合。Source Model Revision、Operation Rule版、Artifact Rule Fingerprint、出力file bytesを分けて記録できる。

**本テストでは証明しないこと**：

- #2620/#2621/#2639のMarkdown/DSL糖衣からの**意味Role受理**、標準構造・Function Data Context Mappingの正式Go正本。
- #2506/#2622のFunction指定Read footprintとRule dependencyの閉包、RDB/APIの実物理Read/Scope/Rowset/Field観測完全性。
- #2611の多Mapping・複数ProviderによるOperationの候補消失と現在選択Targetの適合。新しく単一Providerを正本として選び直さない。
- #2387/#2582のSQL Analyzer Requestと解析Source byte、非出力WHERE/Relation path/Predicateの実依存、正常0件の依存・Reactive Invalidation。
- #2500/#2598/#1174の画面変更→Business Action→実Request/Authority/Commit/Receipt/独立観測。
- #1124/#2516のStudio/SHO/AI向けの実表示・出典復元とConsumer別開示Policy、外部Harnessへの実配信。

**次の段階**：今回のOA-01〜10の実測が得られた後、#2586の**同一試行H1→H2**へOperation/Artifactの実出力を組み込み、既存Data Contextの明示Mapping・実Query/Read Providerと**別資格**で結びつける。対象が未対応なら「まだ接合していない」と報告し、合成FixtureのPASSを埋め草に使わない。

## 6. 統合/停止条件

- CIを実行してexact-headの成功が確認できるまでは**NOT RUN**。保存済みソースの存在だけでPASSと書かない。
- CLI output／Rule契約／Artifact Bodyが変わった場合は、各Consumerにとって何を同値とすべきかをOwnerへ返す。むやみに全属性のbyte同値を強制しない。
- 実Providerや業務意味、現在認可、Manifest/Consumer不足範囲を「モデルとArtifactが一致したから成立」に昇格しない。
- 他Issueの確定意味・ADR/Gate・main固定版を更新しない。この作業は独立ブランチDraftで保持し、資格確認まで統合しない。
