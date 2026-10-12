# 構成案と契約適合証拠の照合（E9・参照実装）

- **状態**：試験用の独立した参照実装。Production の契約適合判定器や Runtime Authority ではない。
- **正本**：[Composition の E5～E8](../composition-kernel/README.md)、[AI向けの最小プラグイン規約](../../plan/design/AI_NATIVE_PLUGIN_COMPOSITION_DESIGN_2026-10.md)、[Capability Contract Reference（#951）](../../tools/reference-plugin-capability-contract/README.md)。
- **機能**：`composition.InspectWiringSet`が列挙した検証対象と、#951 の既存解決結果を指す外部参照を、**同じ宣言上の対象かどうか**に限って照合する。

## 原則

生成AIが「既存のPluginを組み合わせれば実装不要」と提案したとしても、Providerの構造上の適格性だけでは実際の利用を認められない。

今回の`Reconcile`は、構成案が参照するProviderと推移的な依存先を重複なく列挙し、外部から渡された証拠参照の対応を確認する。実行Kernelへ新しいResolverや証拠保管庫を追加しない。独立した読み取り専用の参照機構なので、通常の静的Targetへ組み込む必要はない。

判定とその意味は以下の通り。

| 診断 | 意味 |
| --- | --- |
| `reference-linked-unverified` | 宣言情報が正確に対応する**参照先がある**。適合性・真正性はまだ確認していない |
| `reference-missing` | 対象Providerに対応する参照先が見つからない |
| `reference-stale` | 対象の宣言世代と参照先の宣言世代が異なる。**証拠発行者の失効を確認したわけではない** |
| `reference-target-mismatch` | 対応するProviderの契約ID・版・形状など、宣言上の対象が一致しない |
| `reference-incomplete` | 実装ID・Plugin版・成果物のSHA-256・契約指紋・適合試験の種類・証拠指紋など、参照に必要な情報が欠ける |
| `reference-ambiguous` | 同一の検証対象に複数の正確な参照先があり、一意に選べない |

構成案そのものが競合・未判定・混合Scopeである場合、結果は`Blocked`であり、検証対象を表示しない。依存先を複数のConsumerが共有していても、その検証対象を重複させない。

### 信頼境界

`EvidenceReference`は、#951の`Resolution`と`ConformanceEvidence`に存在する識別情報に合わせた**照合用の参照形式**であり、署名済み証拠や受理済みの契約ではない。外部から渡された情報は偽装できるため、`reference-linked-unverified`でも実行許可にはならない。

次の確認は意図的に本機構の責務から除いた。

1. #951の正本Contract定義・Contract Fingerprint・Conformance Profile・Provider実装・Conformance Evidence・発行Authorityの現在性を、独立した正規の検証器で評価する。
2. #943のビルド成果物と実際に配備されたArtifact Digestの対応を確認する。
3. Hostが現在のPermission、Lease、Resource、Fencing、実行世代、Ready状態を再検査する。

**Compositionの`Contract.Shape`は、#951の`ContractFingerprint`とは別の識別情報**である。両者が同じ文字列、同じ版であることから意味的互換性を推定しない。いずれも正本の結び付けを確認する別の担当機構が必要である。

`Reconcile`は外部参照からAuthorityを生成しない。参照がある場合も`reference-linked-unverified`という名前を維持し、将来これを`verified`へ自動昇格させるべきではない。

## テストの範囲

`handoff_test.go`で、実`composition.Registry`から生成した構成案を利用し、欠落、世代ずれ、契約対象不一致、不完全な証拠参照、重複、適用範囲の不一致、矛盾した構成案、推移的依存の網羅と共有依存の重複除去を検査した。

Go 1.23.2の独立した局所モジュールで、本PRの`proposal.go`・`selection.go`、実`registry.go`と、RouterのBinding型だけを代用する試験用定義を使用し、`go test -race -count=100 -timeout=120s ./reference/composition-evidence-handoff ./internal/composition`、`go vet ./reference/composition-evidence-handoff ./internal/composition`、`gofmt -l`が成功。新規の`handoff.go`・`handoff_test.go`はGitHubのBlob SHAとローカルの`git hash-object`が一致した。

これは**完全なTEIパッケージの試験、PR exact-headの全Repository Health、実際の#951解決処理との結合試験ではない**。本番採用を判断するには、これらの検証に加えて依存関係の規模ごとの性能、証拠の失効と発行者のAuthorityを扱う境界を評価する。
