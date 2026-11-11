# Dependency-based cache invalidation reference（2026-11-11）

## 対象

- 参照実装: `tools/reference-cache-invalidation/`
- 関連Issue: #2595（Build artifact provenance）、#2600（増分再利用と性能）、#3267（Verified Composition BOM）
- 関連ADR: ADR-0045（生成器と生成物の版管理）、ADR-0041（証拠の分離）

## 背景

TEIでは、World Fingerprint、Model Revision、Artifact Revision、Evidence Revisionをそれぞれ別に管理する。既存の参照実装（`reference-derivation-cache`、`reference-evidence-router-connect`、`reference-digest-change-reuse`）は、個別の境界は確認しているが、**依存関係に基づくキャッシュ失効判定**を一つの参照にまとめていない。

本参照は、次を満たす判定エンジンを提供する：

1. World/Model/Artifact/Evidence/Source/Operator/Plugin/Generatorの改版を個別に追跡
2. 依存関係の推移的closureを計算し、再使用可否を判定
3. Artifact再利用、Build省略、Evidence再利用を独立に判定
4. Source Byte一致だけでは安全に再利用できない場合を正しく扱う
5. 不明な依存関係は安全側へ倒す

## 検証条件

| 条件 | 期待 |
| --- | --- |
| Source変更 | Artifact/Build/Evidence再利用を拒否 |
| Model Revision変更 | Artifact/Build/Evidence再利用を拒否 |
| World Fingerprint変更 | Artifact/Build/Evidence再利用を拒否 |
| Artifact Revision変更 | Artifact再利用を拒否 |
| Evidence Revision変更 | Evidence再利用を拒否 |
| Operator/Plugin/Generator変更 | Build省略を拒否 |
| 依存関係追加・削除 | Build省略を拒否 |
| A→B→A（World同一、Artifact Revision変化） | Evidence再利用を拒否（continuity proofなし） |
| 部分取得・未知依存 | 全ての再利用を拒否 |
| 空集合（明示的な依存なし） | 再利用を許可 |
| 途中失敗（partial/failed） | Build省略を拒否 |

## 実施結果

```powershell
go test ./tools/reference-cache-invalidation/... -count=1 -v
```

結果：28テストすべてPASS。

## 性能測定

```powershell
go test ./tools/reference-cache-invalidation/... -count=1 -bench=. -benchmem
```

| ベンチマーク | 平均時間 | 平均メモリ |
| --- | --- | --- |
| `BenchmarkReuseArtifact` | 48.25 ns/op | 0 B/op |
| `BenchmarkReuseBuild` | 93.59 ns/op | 0 B/op |
| `BenchmarkReuseEvidence` | 52.99 ns/op | 0 B/op |
| `BenchmarkComputeClosure` | 1.23 us/op | 400 B/op |

## 追加検証（2026-11-11）

- 関連する参照実装の回帰テスト（`reference-derivation-cache`、`reference-digest-change-reuse`、`reference-evidence-router-connect`）がすべてPASS。
- 本参照はread-onlyで、Source書換えやreceipt発行をしない。

## 未対応の失効パターン

- 実キャッシュストアとの統合
- 分散環境での原子的Snapshot保証
-  cryptographic continuity proofの検証
- 実Build/実Evidenceとの接続

## 位置付け

本参照は、キャッシュ失効エンジンの**最小契約と正負対照**を提供する。製品の分散キャッシュ、履歴保存、認可エンジンを代替しない。
