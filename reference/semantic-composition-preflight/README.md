# E4d検証：Composition Kernelの選択結果とデータ専用Semantic資産の接続

- 対象：[外部Semantic参照 #1570](https://github.com/hat47x/tei-platform/issues/1570)
- 状態：**参照実装**。製品のProvider選択・配布仕様・受理手順ではない。
- 先行：[`reference/semantic-distribution`](../semantic-distribution/README.md)の既存ビルド固定情報・意味内容照合・名前空間集合監査
- 合成機構：[ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)および`internal/composition.Registry.InspectWiringSet`

## 接続の目的と境界

新たなSemantic専用Resolver/Lockを実装するのではなく、
既存のComposition Registryが提示する**明示指定の取得サービス**を読み取り、
既存の`BuildArtifactLock`に固定されたデータ専用Semantic配布物を読み込む。

`Inspect`の処理順序は次のとおり。

1. 呼出元が明示した`ProviderID`、契約、Scope、論理名前空間IDの組を確認する。
2. 既存`InspectWiringSet`で同一Scope・一貫したRegistry版の**構造的候補**を得る。
3. `ProposalCurrent`で選択宣言の鮮度を確認する。これはHost認可ではない。
4. `InspectLockedSemanticAsset`へ各項目の**個別に選ばれたBuildとLock、期待する二種類のDigest、バイト取得関数**を渡す。配布物の任意探索や最新候補への切替はない。
5. ローカル意味定義と配布された名前空間を一時ビューで組み合わせ、既存`AuditSelectedNamespaceSet`で直接参照・未一致・重複・循環候補を調べる。未一致参照は閉包を自動拡張せずに失敗させる。
6. 既存標準ライブラリを含めて`validation.Validate`で意味検証する。
7. 再度`ProposalCurrent`で検査中の宣言変更を検出する。

ただしこの最終確認も実行時までの鮮度・排他・権限を保証しない。実行時Hostは独立して現在の権限、Provider実体、配布成果物・署名を確認する。

## 意味上の循環はサービス依存の循環と異なる

`common`と`os`が互いの業務Dataを参照することと、
Composition Serviceが互いの起動・利用を必須とすることは別の関係である。
`AuditSelectedNamespaceSet`の`CycleCandidates`はレビュー候補として保持する。
Semantic参照をKernelの`Provider.Requires`へ機械的に投影しないため、
合法な相互参照でサービス依存の循環エラーを捏造しない。

## 何を保証しないか

- `Composition.WiringFinding`の候補は、Artifact・署名・権限の真正性を証明しない。
- `BuildArtifactLock`と配布データのDigest一致も、Builder・発行者の信頼を証明しない。
- Namespaceの取得先Service Provider IDと配布物のPackage IDは別軸。実運用の発行者証明との結び付けはここでは**未実施**。
- `validation.Validate`の成功は、実際のApplication、Database、Host Runtime、Plugin実装の完全な動作確認ではない。
- 返す`Report`は読取り専用の候補・診断であり、Project Acceptance、Invocation Authority、Workspace Source Indexを変更しない。
- 二番目のRegistry、契約体系、知識URI Scheme、推移的依存の自動取得、Providerへの命令発行は作らない。

## 回帰試験

```sh
go test -count=1 ./reference/semantic-composition-preflight
```

実際の二つのWorkspace（提供元・利用元）の`.tei`を読み込み、
`common`と`os`を独立したBuildArtifactLock付き模擬配布物として構成。
複数の明示選択と契約一致、間違ったScope・Provider・契約、
同名Namespaceの二重選択、配布物のDigest不一致、依存先未選択、
バイト取得中のRegistry改訂を確認する。実Store・実署名は使わず、
選択・信頼・受理の境界に未完了部分を明示する。

この参照実装が有効な場合に限り、次段階でProductのComposition Profile、
信頼済みArtifact取得、提供元の公開範囲・同意・受理記録との結合を検討する。
