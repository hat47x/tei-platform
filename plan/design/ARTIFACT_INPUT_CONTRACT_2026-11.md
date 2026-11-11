# 成果物の入力契約（設計記録）

- **状態**: 実装済み（検証済みの範囲と未解決の課題を記録する）
- **対象課題**: [#2482](https://github.com/hat47x/tei-platform/issues/2482)、[#2604](https://github.com/hat47x/tei-platform/issues/2604)、[#2595](https://github.com/hat47x/tei-platform/issues/2595)
- **関連**: [導出成果物の設計候補](DERIVED_DESIGN_ARTIFACTS_DESIGN_2026-11.md)、[ArtifactSetの対象閉包設計候補](ARTIFACT_SET_SCOPE_AND_CONSUMER_READ_EVIDENCE_2026-11.md)
- **実装ブランチ**: `feat/artifact-evidence-bundle-20261111`
- **検証日**: 2026-11-11

## 1. 目的

同じ入力・同じ規則・同じ Toolchain で再ビルドしたとき、成果物の bytes と検証結果が一致することを
機械が確認できるようにする。入力が変更された場合や、別の Toolchain で処理した場合に、
既存の成果物を誤って再利用しない仕組みを整備する。

## 2. 入力の定義

成果物の生成に影響する入力は、次の六つに分類する。

| 分類 | 内容 | 現在の扱い | 課題 |
| --- | --- | --- | --- |
| Source | 作業領域内の `.tei`、`.sql`、画像等のbyte実体 | `sourceindex.Index` が path と fingerprint を記録 | 検査後に変更された場合の再利用防止は実証が必要 |
| Model | 受理済みの Canonical World（Data/Capability/Validation/Persistence 等） | `ir.World` が正本 | World fingerprint と Source Index fingerprint は分けて記録する |
| Dependency | 外部の Plugin、Provider、方言、標準ライブラリ | `go.mod` / `plugins/` で宣言 | 実行時に取り込まれた Plugin の実体記録は未実施 |
| Operator | 成果物を要求する人/システム、Application ID、用途 | CLI の `-application` / `-workspace` | Consumer purpose を EvidenceBundle に追加する |
| Generator | 導出器の実装（テーブル定義書/機能定義書/DDL 等） | `artifact.Artifact.Generator` に記録 | 同じ rule でも別の実装が異なる bytes を生成する場合は generator を分ける |
| Toolchain | Go の version、build settings、OS/ARCH、実行バイナリの identity | `cliBuildIdentity` / `BuildEvidence` が記録 | `-ldflags` や `-tags` の差異が artifact digest に与える影響は測定が必要 |

### 2.1 Source の検査時点とコピー時点

- `workspace.Load` は pinned root で Source を読み、`sourceSnapshot` と `sourceManagedExtra` に Load-time bytes を保持する。
- `BuildSourceIndexFromLoad` は、Load-time bytes を再利用して Source Index を作る。検査後の再読込みは行わない。
- この仕組みにより、「検査後に Source が変更された場合」に古い bytes で成果物を生成するリスクを
  検出できる（Model Revision が変化する）。

## 3. 成果物の再現性と非決定的要素

### 3.1 既に決定論的な部分

- `artifact.Artifact` の digest は、`inputRevision + rule + generator + content + unresolved` から
  `sha256` で一意に決まる。
- `Manifest` の digest は、入力 revision と Artifact の digest 一覧から決まる。
- `SourceIndex.ModelRevision` は、Managed Source の (path, fingerprint) の辞書順連結から決まる。
- `WorkspaceProjectionFingerprints` は、World と Source Index の JSON 正規化表現から決まる。

### 3.2 非決定的な要素とその扱い

| 要素 | 理由 | 現在の扱い | 今後の対応 |
| --- | --- | --- | --- |
| 生成時刻 | `time.Now()` を EvidenceBundle に記録 | `GeneratedAt` は OpenTelemetry の Trace ID と同様に観測目的で含める | 成果物の digest には含めない |
| 並列処理の順序 | マップの反復順序 | `sort.Slice` で固定 | 既に Artifact/Manifest で実施済み |
| Toolchain の再現性 | Go の compiler 最適化、環境変数 | `cliBuildIdentity` で Go version / build settings を記録 | `reference-go-build-reproducibility` で二回の独立ビルドを確認する |
| Plugin の動的選択 | 複数の Provider が同じ Contract を実装 | `generatorselection` で fixture に基づく選択を記録 | 動的選択を artifact digest に混ぜない |

## 4. 検証結果と Evidence の対応

### 4.1 検証の三段階

```
Source inspection (Load + BuildSourceIndex)
    ↓
Generation (RenderWorkspace + NewArtifact)
    ↓
Verification (CheckSet + Manifest.Verify + Artifact.Verify)
```

### 4.2 EvidenceBundle の内容

- **Input**: `workspace` / `applicationId` / `worldFingerprint` / `sourceIndexFingerprint` / `modelRevision`
- **Toolchain**: `generator` / `cli` identity
- **Render**: `artifactCount` / `complete` / `manifestDigest` / `artifacts[]`
- **Verify**: `findingsCount` / `findingCodes[]` / `verified`
- **Output**: `directory` / `files[]` / `evidenceFile` / `evidenceDigest`

### 4.3 検証結果と成果物の対応表

| 検証項目 | 成功条件 | 失敗時の挙動 | Evidence への記録 |
| --- | --- | --- | --- |
| Source が一致 | Model Revision が再実行でも同一 | 変更検出（Model Revision の差分） | `modelRevision` フィールド |
| Artifact digest が一致 | 同一入力・rule・generator で同一 digest | 不一致時は `FindingArtifactTampered` | `render.artifacts[].digest` |
| Manifest が一致 | Manifest digest が再実行でも同一 | 不一致時は `FindingManifestTampered` | `render.manifestDigest` |
| CheckSet がパス | Findings が空 | 非空の場合は `findings` に出力 | `verify.findingsCount` |
| 公開が原子的 | Stage で検証後に Publish | 途中失敗時は outDir を変更しない | `output.directory` + `files[]` |

## 5. 隔離された出力領域と比較検証

### 5.1 Stage-Publish パターン

```
StageArtifacts(outDir, set)
    ↓  tmp/.tei-artifacts-* 内で検証
PublishStage(stage, outDir)
    ↓  検証済みファイルだけを outDir に移動
WriteEvidenceBundle(outDir, bundle)
    ↓  evidence.json の on-disk digest を記録
```

- 検証前に `outDir` を変更しない。
- 重複ファイル名は Stage 時点で拒否する。
- 旧ファイルは Publish 時に削除する。

### 5.2 誤再利用の防止

- Source 変更 → Model Revision の変化 → 新規の EvidenceBundle が必要
- Toolchain 変更 → `cliBuildIdentity` の差分 → 同じ Source でも別の Toolchain 記録
- 異なる Application ID → 別の EvidenceBundle（同一 Workspace 内でも分離）

## 6. 未解決の課題

1. **Source のタイムスタンプ**: ファイルの mtime を入力に含めるか。現在は bytes のみ。
2. **Plugin の実体記録**: `go.mod` の version だけでは不足。実行時に取り込まれた Plugin の
   binary digest を記録する手段は未設計。
3. **大規模 Workspace のパフォーマンス**: Load-time bytes の保持はメモリに影響する。
   1000 ファイルを超える Workspace の挙動は未測定。
4. **Windows のファイルパス**: パス正規化は実施済みだが、長いパスや特殊文字の挙動は未検証。
5. **Deterministic build の証明**: 二回の独立ビルドで同一 bytes を確認する再現性テストは
   参照ツール側に存在するが、`tei artifacts` CLI への統合は今後の作業とする。

## 7. 検証結果（2026-11-11 現在）

### 7.1 通過した検証

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

### 7.2 未解決の課題（ブロッカー）

1. **cmd/tei パッケージの事前ビルド失敗**: `cli_three_element_test.go` の `fixtureBytes` 未定義、`cli_db_introspect.go` の型誤り。
   - 本事象は artifact evidence bundle 変更の前に存在する事前失敗であり、本機能では未着手。
   - 回避策: `internal/artifact` 単体で検証済み。`cmd/tei` 統合テストは本事象の解決後に実行する。
2. **Windows Go ビルドキャッシュの破損**: 検証実行中に `go-build` キャッシュのファイル欠落が発生。
   - 回避策: 一時キャッシュ (`%TEMP%`) を使うことで `internal/artifact` の検証は実行可能。
   - 恒久対策: キャッシュディレクトリの ACL または Go ツールチェインの更新が必要。

### 7.3 今後の作業

- `cmd/tei` の事前ビルド失敗を分離して解決し、`go test ./cmd/tei` を実行する。
- 参照ツールによる deterministic build の独立確認を実施する。
- Plugin の実体記録（binary digest）を設計する。
