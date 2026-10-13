# E4c: 既存Plugin BuildArtifactLockを利用した意味データ配布の接続実証

- 対応課題: [#1570](https://github.com/hat47x/tei-platform/issues/1570)
- 前段: [E4a/E4b](../semantic-distribution/README.md)
- 設計判断: [ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)

## 今回つながるもの

`reference/semantic-distribution`で定義した**データ専用の模擬配布物**を、既存`internal/plugin`の以下の機構で固定してから読み取る。

- `plugin.BuildPlan`：入力モデル、Builder、Recipe、出力種別、Targetを明示する。
- `plugin.LockBuildOutput`：ビルド計画のFingerprintと出力バイトのDigestを`plugin.LockedBuild`へ固定。
- `plugin.BuildArtifactLock.Validate`：既存のBuild Artifact Lockの版と構造を確認。
- `InspectLockedExport`：呼び出し元が**明示指定した**BuildPlanがLockに対応する正確なPlan/Artifact fingerprintを生成したことを確かめ、**既存の`semanticdistribution.InspectLockedSemanticAsset`**を呼び出す。この下位AdapterがBuildArtifactLockと配布元・正規化意味モデルを照合する。

**新たなLock/Registry/Resolverは設けない。** データのみの配布資産の判定は、すでにmainにある`semanticdistribution.InspectLockedSemanticAsset`（実CAS縦断のPR #1683）へ委譲する。試験用の出力種別は同Adapterの`SemanticReferenceOutputKind = semantic-data`を共用する。TEI標準の新しいOutputKind、CapabilityKind、RuntimeKind、Service Contract等は制定しない。

この接続により、同じ`os`名前空間を二つの独立したWorkspaceから、Path参照と**選択済み・ビルド固定された模擬配布物**の両方で利用できることを試す。

## 注意すべき境界

この実証は実行時プラグインの配布・ロードではない。ビルド実行や本物の配布装置の取得、署名、信頼されたBuilderの真正性、Runtime Authority、Shared Acceptanceの保証も行わない。テスト中の`BuildPlan`とArtifactは**参考値を使って組み立てたもの**であり、独立署名された発行物ではない。

`plugin.BuildArtifactLock`は一つの計画と生成物の紐づけを表すのみで、特定のファイルや提供者の信頼性の最終根拠にはならない。呼び出し側が同じ疑わしい情報からPlan/Lock/Artifactを捏造できる環境では、これらの一致検証によってセキュリティを確保したことにはならない。

`InspectLockedExport`に渡す`selected`は明示指定が必須。別の提供者や最新版へ勝手にfallbackしない。Runtime用途の出力はデータ配布物として読み込まず、非明示な元ソースをWorkspace Source Indexに追加しない。

## 検証コマンド

```sh
go test -count=1 ./reference/semantic-build-bridge ./reference/semantic-distribution
```

固定HEADでの実行結果は[PR](https://github.com/hat47x/tei-platform/pulls)のCI履歴に残す。READMEの存在を実テスト成功と混同しない。

## 次に確認するもの

次期Composition Kernel/#1223とAsset Store/#1105で、**すでに外部で署名・照合済みのBuildPlan/Artifact/選択**を受け取れることを縦断する。正式な配布資産の型やパッケージ設計が決まったら、今回の試験用`semantic-data`と模擬Envelopeを正式なAsset Adapterへ置き換える。旧Pluginの固定`CapabilityKind`にSemantic機能を追加せず、製品で使用する正式なBuildOutputKindの採用はその時点で判断する。

E4cは**本番機能の完成判定ではなく、既存ビルド・Lock表現の再利用可否を試したReference**。

## #1683との役割整理

[PR #1683](https://github.com/hat47x/tei-platform/pull/1683)は、既存`BuildArtifactLock`で明示された成果物を実ローカルCASから取得するReferenceを先にmainへ統合しました。このディレクトリの**追加価値は、読み取る前に呼出側のBuildPlanとLockedBuildの入力計画Fingerprintを照合することと、二つの独立Consumerの意味的同等性を検証すること**です。元のCAS Adapterと別の配布物取得・選択・Lockを実装しない構造へ縮退させています。
