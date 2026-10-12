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

## 多数のProviderを扱うときの照合

証拠参照は、Provider IDと適用範囲の組を鍵として一度だけ分類する。構成案の検証対象に含まれないProviderや適用範囲の参照は、その索引に保持しない。個々の検証対象について、他Tenantや他Providerの全参照を走査せず、大量の無関係な資料を一時的に蓄積しない。検証対象の並び順、同一対象の複数参照を曖昧として返す規則、契約不一致・世代ずれ・参照欠落の優先順位は変えない。これは参照の探索負荷を抑えるだけであり、権限や証拠の真正性を保証するものではない。

`handoff_test.go`には64段階の依存関係と他範囲・契約不一致・世代ずれ・正確な参照の重複が同時に存在する試験、および1万2千件の無関係な参照が検証対象や照合結果を変えないことを確かめる試験を追加した。また96段階の依存関係を対象とする性能測定用の試験を用意した。性能値は測定環境に依存するため、数値を取得するまでは高速化を保証しない。

## テストの範囲

`handoff_test.go`で、実`composition.Registry`から生成した構成案を利用し、欠落、世代ずれ、契約対象不一致、不完全な証拠参照、重複、適用範囲の不一致、矛盾した構成案、推移的依存の網羅と共有依存の重複除去を検査した。

Go 1.23.2の独立した局所モジュールで、本PRの`proposal.go`・`selection.go`、実`registry.go`と、RouterのBinding型だけを代用する試験用定義を使用し、`go test -race -count=100 -timeout=120s ./reference/composition-evidence-handoff ./internal/composition`、`go vet ./reference/composition-evidence-handoff ./internal/composition`、`gofmt -l`が成功。新規の`handoff.go`・`handoff_test.go`はGitHubのBlob SHAとローカルの`git hash-object`が一致した。

これは**完全なTEIパッケージの試験、PR exact-headの全Repository Health、実際の#951解決処理との結合試験ではない**。本番採用を判断するには、これらの検証に加えて依存関係の規模ごとの性能、証拠の失効と発行者のAuthorityを扱う境界を評価する。


## E10：実際の契約解決結果から参照を作る縦断試験

#951の参照実装には、正確な契約・実装・成果物・適合証拠から解決結果を求める`resolveCapability`がある。これとComposition側の構成案を接続するため、[Capability Contract Reference](../../tools/reference-plugin-capability-contract/README.md)と同じPackageに`composition_handoff.go`と`composition_handoff_test.go`を追加した。

接続関数は、次の順番で動く。

1. `composition.InspectWiringSet`で得た検証対象に、**どの正確な#951 Provider実装を対応させるか**を明示的に指定する。名前が似ているProviderを暗黙に対応させない。
2. 既存の`resolveCapability`へ契約・Provider・適合証拠・権限区分を渡す。#951の解決器を複製しない。
3. 解決できた結果の契約ID・**正確な版**・契約全体の指紋・実装ID・Plugin版・**成果物のダイジェスト**が、明示した対応関係と一致する場合だけ、E9の`EvidenceReference`を作る。
4. E9の`Reconcile`で、構成案の検証対象と解決結果への参照を照合する。

**二種類のScopeは区別する**：Compositionの`Scope`はそのRegistry内での隔離・選択範囲であり、#951の`AuthorityScope`（設計ツール、ビルド、アプリケーション実行など）とは同義ではない。両者の対応は、検証対象と実装を結び付ける明示的な設定によって示す。Compositionの`Contract.Shape`を#951の`ContractFingerprint`へ読み替えることもしない。

### 結合試験の対象

- 実際の#951の`resolveCapability`から得たProvider、契約指紋、適合証拠指紋、成果物、解決結果の識別情報がE9に欠落なく引き渡されること。
- 適合証拠なし、権限区分の相違、契約の意味の変更、互換しない要求版、正確な実装・成果物・契約指紋からの逸脱では参照を作らないこと。
- 証拠の更新では解決結果の識別情報が不変でも、証拠指紋が変化することを区別すること。両方の証拠参照が同時にある場合に一方を暗黙に選ばないこと。
- Providerの撤去・同名再登録後は、以前の構成案に結び付く参照が宣言世代の古いものとして診断されること。
- 同じ実装の複数の物理成果物が適合していても、成果物を明示しない限り解決器が曖昧として拒否すること。

### 検証の状態

E9の既存局所パッケージについては、ローカルで`go test -race -count=10 -timeout=90s ./reference/composition-evidence-handoff ./internal/composition`、`go vet`、Go整形検査が再度成功した。

一方、今回追加した**実際の#951パッケージを使うE10の結合試験は、GitHubへソースを追加しBlob SHAを確認した段階であり、実行していない**。作業コンテナからの`git clone`はDNSで失敗する。E9局所試験の成功をE10の結合試験やPR exact-headの全体テストの成功とみなさない。

なお、#951の参照解決器が渡された情報から結果を求められたことは、契約定義の発行権限、適合証拠の真正性・現在性、署名、Hostの実行許可を意味しない。E10の成功時にも最終診断を`reference-linked-unverified`に留める。Productionへ昇格するには#951の信頼できる正本・発行者検証、#943の実成果物、Hostの現在権限への接続が必要である。


## E11：現在のRegistryを再照合してから証拠参照を表示する

E9の`Reconcile`は、構成案を作った時点の宣言と証拠参照を照合する。これは過去の判定履歴を調べるには有用だが、**作成後にProviderや依存先が変更された構成案を現在も有効と誤認する危険**があった。

そこで、`internal/composition/proposal_freshness.go`に読み取り専用の`Registry.ProposalCurrent`を追加し、E9側には`ReconcileCurrent`を設けた。

- `ProposalCurrent`は、構成案のRegistry版、対象Providerの選択、推移的な依存先の宣言世代、検証対象の一覧を、現在のRegistryと同じMutex保護下で再照合する。
- 既存の`Registry.validateSelectionLocked`を利用し、第二の依存解決器を作らない。無関係な更新は許容し、依存先の宣言変更・撤去・同名再登録・選択結果の改変を拒否する。
- `ReconcileCurrent`は、鮮度検査に不合格なら`Blocked`を返し、証拠参照を表示しない。E10の実Resolverとの試験にもこの経路を採用した。
- 旧`Reconcile`は**取得時点の履歴照合専用**として残す。結果を現在の実装の利用可能性と読み替えない。

これは**実行許可のための原子的な予約ではない**。鮮度検査後にRegistryが更新される可能性があるため、Hostは配備・呼び出しの際に自身の選択・現在権限・Lease・Fencingを改めて検査する。証拠の署名・発行者の正当性・失効も別の正式な検証責務であり、鮮度検査の成功から推定しない。

### 追加した試験と現時点の検証範囲

`proposal_freshness_test.go`では、推移的な依存先の変更、他Tenantの無関係な登録、Provider撤去・同名再登録、偽装した宣言世代・対象Provider・適用範囲を扱う。

`current_test.go`では、現在の証拠参照、無関係な更新、依存先の変更、循環する無効な編集の不採用、古い構成案の遮断と過去の参照照合を扱う。`composition_handoff_test.go`も、実Resolverから参照を作る正常系とProvider再登録時に`ReconcileCurrent`を使用するように変更した。

**これらE11の追加試験は、GitHubにソースを登録した段階であり、PR exact-headのGo test実行証拠はまだ無い。** コンテナからGitHubを取得する接続はDNS・ネットワーク制約で失敗し、GitHubの当該HEADにも自動チェック結果はない。前段のE9局所試験の成功をE11へ流用しない。

今後は、完全なcheckoutで`go test -race ./internal/composition ./reference/composition-evidence-handoff ./tools/reference-plugin-capability-contract`を実行し、`tools/repository-health-check.sh`の全項目PASSを取得した後に統合を判断する。
