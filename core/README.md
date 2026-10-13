# コア

このディレクトリは、TEIのコアを独立したサブプロジェクトとして管理するための入口です。**Subject値型・一時Effect Scope・M0識別子の正本を段階的に移管した状態であり、コア全体がここだけで構築できる状態ではありません。**

コアは、サービス契約に基づく合成、依存関係の解決、実行世代の切り替え、信頼境界など、個別の実装に左右されない最小限の仕組みを担います。具体的な認証方式、データベース接続、画面・実行先固有の機構を抱え込まない方針です。

## 最初の実ソース移管

[Subject値型](subject/README.md)は、論理対象をソース位置から切り離して識別する小さな値型です。実装の正本を`core/subject/`へ移し、旧`internal/subject/`は型エイリアスの互換入口としました。旧型と新型はGo言語上同一で、値判定とJSON形式をテストしています。

`bash tools/check-core-subject-isolation.sh`は正本のソースだけを一時Goモジュールへ複製し、Go標準ライブラリ以外への依存を持たず、別モジュールの利用者から使えることをオフラインで検証します。この一例から、新Service Contractや合成基盤が製品コアへ移管済みとは判断しません。

[一時Effect Scope](effect/README.md)は、`internal/composition/scope.go`にあった、逆順の一時資源解放と失敗集約を担当する実装を移したものです。旧`composition.Scope`は`effect.Scope`への型エイリアスとなり、旧APIの利用者は同じ状態管理とエラーを引き続き使用します。合成基盤の契約選択・Host権限・世代管理は旧パッケージ側に残したままです。

`bash tools/check-core-isolation.sh`では、CompositionとEffectの実ソースを別々の一時Goモジュールへ取り込み、隔離した外部Consumerからもこの型の同一性を確認します。

[M0識別子と参照の正本](identity/README.md)では、既存の`CanonicalID`・`LocalName`・`Kind`・`Ref`・数値バージョン・標準名称導出を`internal/core`から移しました。旧入口は**Go型エイリアスと関数委譲のみ**です。識別子のハイフン許容など将来の規則改訂は、この機械的移管と混同しません。

`bash tools/check-core-identity-isolation.sh`で`core/identity`・`core/subject`・`core/effect`の実ソースをまとめて、独立の一時Goモジュールからテスト・外部利用できることを確認します。

## 現在の実装

現在の主な関連コードは、リポジトリ直下の次の場所にあります。

- [基本要素](../internal/core/)
- [合成基盤の試作](../internal/composition/)
- [従来のプラグイン管理機構](../internal/plugin/)

これらを機械的にまとめたものが、将来のコアになるわけではありません。既存実装の成立範囲は[実装状況](../docs/status.md)を参照してください。

## Service Contractから既存Runtimeへの移行中の接続

`internal/plugin/runtime/service_carrier.go`には、選択済みService Contract/Providerを、**Hostが明示した既存Runtime Carrier**へ結び付ける移行期の入口があります。`EnvironmentPlan.ServiceBindings`は旧CapabilityKindの意味をService Contractへ転用する宣言ではなく、選択されたProviderの論理識別子・具体的なArtifact digestと、旧Runtimeがすでに持つCapability ID・Operationへの一時的な経路指定です。追加した束縛は既存Runtime Policy fingerprintの対象となり、同名Contractの重複、Scope違い、選択済みProvider/ArtifactとHost計画の不一致は実プロセス起動前に拒否します。実行時のPermission、Resource Lease、Artifact verification、Sandbox、Runtime Provenanceは既存`Registry.InvokeWithProvenance`が担います。

この入口は**Contract resolverではありません**。Hostが採用した契約と適合性証跡を正当に取得したか、Shape検証、Publisher Authority、Conformance、Contract互換性を証明できません。結果の`AuthorityLimits`へ未検証を明示します。旧Carrierの経路が新しい製品用Universal ABIになることもありません。将来新Service Contractの正式なCore定義・解決・Host権限へ移行したときは、重複する定義を残さずこの一時対応付けを撤去します。

### 呼出し時の現在受理をHostに照会する

製品`Registry.InvokeSelectedServiceCarrier`は、Serviceの照合に成功しても、それだけでは起動しません。`Registry.ServiceAcceptance`にHostが注入する`ServiceAcceptanceReader`を**呼出しごと**に照会し、選択したContract/Provider/Artifactと一致する現在の受理記録を求めます。読取元が無い、受理記録が欠ける、選択対象が違う、現在の受理が撤回された場合は、既存RuntimeのVerifier・Lease・子プロセスへ進みません。受理後も、従来の`InvokeWithProvenance`が成果物検証・権限・資源制御を独立に実行します。

このInterfaceは**既存の信頼済み受理情報源を読むための接続点**であり、新たなContract発行者・適合性判定器・証跡発行機構ではありません。Plugin自身やリクエストから受理元を渡すことは認めず、Hostが管理する情報源の実装が現在性・Issuer・Scope・失効の確認を負います。ContractとProviderの受理参照文字列が揃ったことだけで暗号学的な署名や正式なConformanceを証明したとは解釈しません。提供するPluginのManifestや性能証明を全案件へ追加で要求するものでもありません。

## 独立構築の確認

合成基盤が他のTEIパッケージに依存せず試験できるかを調べるため、[隔離試験の手順](../plan/design/CORE_SUBPROJECT_ISOLATION_PROBE_2026-10.md)を用意しています。リポジトリのルートから`tools/check-core-isolation.sh`を実行します。これは現在の`internal/composition/`を一時的なモジュールへ複製して試すもので、`core/`への移管完了を意味しません。

## 構成の判断

- [サブプロジェクト化の設計判断](../plan/adr/0033-core-and-plugin-subprojects.md)
- [新しい合成基盤の設計判断](../plan/adr/0032-composition-kernel-replacement.md)
- [全体構成](../docs/architecture.md)

今後、公開契約、内部実装、ビルド入口、検証方法を確定してから段階的に移管します。移管が成立するまでは、従来の`go.mod`とテスト方法が有効です。
