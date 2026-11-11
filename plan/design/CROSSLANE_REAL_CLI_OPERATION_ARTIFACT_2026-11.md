# Everything is Data：実TEI CLIでのOperation→Artifact受渡しを同一版で検証する

**状態（2026-11-03更新）**：実製品CLIの限定read-only縦断に対し、前headのexact-head CI成功を観測した。追加した再資格化反例の現headは別CIで再確認する。製品のFunction Data Context/実Provider/RDB/API/外部Consumer/Owner業務意味受理ではない。

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

- 過去headのCI成功を現headへ移さない。新しく追加した負例のexact-head CIが成功するまで、その新ケースは**NOT VERIFIED**。
- CLI output／Rule契約／Artifact Bodyが変わった場合は、各Consumerにとって何を同値とすべきかをOwnerへ返す。むやみに全属性のbyte同値を強制しない。
- 実Providerや業務意味、現在認可、Manifest/Consumer不足範囲を「モデルとArtifactが一致したから成立」に昇格しない。
- 他Issueの確定意味・ADR/Gate・main固定版を更新しない。この作業は独立ブランチで行い、変更範囲の検証結果を記録して統合する。Draftは作業中の表示であり、解除して統合する。


## 7. 実Go CLIのexact-head CI観測と、完成度が一致しない反例（2026-11-03）

### 7.1 局所実証が成立した版（過去headの事実）

[専用GitHub Actions #41578082203](https://github.com/hat47x/tei-platform/actions/runs/41578082203)／job `observed-real-cli`（ID 125726160626）は、**候補head `57b87801ff462127574a20885d5b091b516cdd81` に対して completed/success**。CIログを直接確認した。

ログに記録された実Go CLIバイナリのVCS revisionは上記head、検査スクリプトのGit blobは `16b62ff36e502808277245a72e1f6ece43205b49` であった。次の二つのWorkspace/Applicationを**別々のCLI process**で観測した。

| Application／公開Fixture | Operation件数 | Operation gaps | Artifact件数 | Manifest Complete | 出力Bodyとの照合 |
| --- | ---: | ---: | ---: | --- | --- |
| `ledger` ／ `reference/standard-crud/workspace` | 5 | 4 | 5 | `false` | 共有Field一致／ファイルbyte一致・改変拒否 |
| `order_entry` ／ `reference/workspace` | 8 | **0** | 4 | **`false`** | 共有Field一致／ファイルbyte一致・改変拒否 |

**新しく独立に確認できた反例：Operation Surfaceの`gaps == 0`でも、同一Applicationから生成したArtifactSetの`Manifest.Complete`は`false`になり得る。** 元の設計では「操作面の導出の不足がない」と「設計書等の別種Artifactに未解決がない」は別Claimである。前者が成立しても後者を自動昇格させない。さらにManifest Completeは、既存#2633で確認されたとおり、外部Consumerの期待Member、受領byte、現在の開示資格までは証明しない。

`-out`の保存Bodyをテスト内で意図的に改変したときに**検査スクリプトが不一致を拒否した**ことは観測できた。これはTEI製品の一般Readerがすべての改変を拒否するという証拠ではない。保存先が一時ディレクトリであること、`artifacts -out`がファイル書込みを行うことも、Operation Surfaceのread-only性と分離する。

| 検証 | 結果 | 証明 |
| --- | --- | --- |
| `internal/artifact` の単体試験 | PASS | `go test -count=1 ./internal/artifact` |
| EvidenceBundle の生成・検証 | PASS | `TestNewEvidenceBundleRecordsExactSourceAndArtifactState` 他 |
| Stage-Publish の原子性 | PASS | `TestStageArtifactsIsolatesAndVerifiesOutput` |
| 公開後の stale file 削除 | PASS | `TestPublishStageMovesFilesAndRemovesStaleFiles` |
| 改竄検出（artifact content） | PASS | `TestEvidenceBundleRejectsTamperedArtifactContent` |
| 改竄検出（staged evidence） | PASS | `TestStagedSetVerifiesEvidenceFileWhenPresent` |
| Source 変更の検出 | PASS | `TestSourceMutationChangesModelRevisionAndEvidenceBundle` |
| 同一入力の再現性 | PASS | `TestSameSourceAndToolchainProduceSameEvidenceBundle` |
| Toolchain 変更の検出 | PASS | 同一テストで generator 変更を確認 |
| Manifest の読戻し | PASS | `TestReadPublishedManifestReturnsEnvelopeContents` |
| Output 省略（outDir 未指定） | PASS | `TestWriteEvidenceBundleOmitsOutputWhenDirectoryEmpty` |
| ファイル名の安定性 | PASS | `TestArtifactFileNameMappingIsStable` |
| CLI artifact 統合検証 | PASS | `go test -count=1 ./cmd/tei -run 'TestArtifact|TestArtifactPublication|TestThreeElement|TestOperations'` |
| CLI build 成功 | PASS | `go build ./cmd/tei` |

### 7.2 未解決の課題（pre-existing）

1. **cmd/tei のサブプロセスタイムアウト**: `TestMissingRequiredFlagIsAlwaysInvalidArgument/trace_without_-subject-id` が
    subprocess 実行で 2 分を超える。本事象は artifact evidence bundle 変更の前に存在する事前失敗であり、
    本機能では未着手。回避策: artifact 関連テストに絞って検証済み。
2. **Windows Go ビルドキャッシュの破損**: 検証実行中に `go-build` キャッシュのファイル欠落が発生。
   - 回避策: 一時キャッシュ (`%TEMP%`) を使うことで `internal/artifact` の検証は実行可能。
   - 恒久対策: キャッシュディレクトリの ACL または Go ツールチェインの更新が必要。

### 7.3 今後の作業

- `cmd/tei` のサブプロセスタイムアウトを分離して解決し、`go test ./cmd/tei` の完全通過を目指す。
- 参照ツールによる deterministic build の独立確認を実施する。
- Plugin の実体記録（binary digest）を設計する。
- EvidenceBundle コンシューマー読み取りのスキーマ実装を設計する。
