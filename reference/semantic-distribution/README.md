# 外部意味モデルの配布経路比較（E4の独立参照実装）

- **状態**：実装候補・検証用。プラグイン配布の製品機能ではありません。
- **対象課題**：[外部Semantic参照 #1570](https://github.com/hat47x/tei-platform/issues/1570)
- **背景**：[構成Kernelの設計判断 ADR-0032](../../plan/adr/0032-composition-kernel-replacement.md)、[案件の構成 #1016](https://github.com/hat47x/tei-platform/issues/1016)
- **役割**：既存の `semantic.Namespace` をディレクトリから取得した場合と、配布物から読み取った場合に、**意味が変わらないこと**を実際のGoモデルと検証器で確かめる。

## 比較する二つの経路

1. **作業領域からの参照（現行のE1～E3）**：利用側 `tei.workspace.yaml` の `imports` から、別の作業領域が所有する名前空間を明示的に読み込む。必要なら `semanticDigest` で内容を固定する。
2. **データだけを持つ配布物（今回のE4参照実装）**：供給側の `semantic.Namespace` を決定的なJSONに格納し、配布者の識別子・版、**配布物そのものの照合値**、**意味モデルの照合値**を別々に照合する。選択・読込み後の名前空間だけを、検証用に利用側の一時的な意味ビューへ追加する。

後者は、将来のプラグインによるモデル配布に必要な**値と失敗時の振る舞いを検証するための模擬配布物**です。現在の `internal/plugin.PluginPackageManifest` の形式でも、`BuildArtifactLock` の改訂でも、実行時に選択できる正式なプラグイン形式でもありません。

利用側での識別子 `os.process_id` や `os.process` は、取得元の配布者名・版・ソースパスによって改名されません。

## 検証する性質

- 同じモデルを二つの取得経路から読み込んでも、正規化された意味モデル・型参照・検証結果が同じ。
- 配布物の照合値と意味モデルの照合値は異なる。同じ配布者と版でも内容が変わった場合は失敗。
- 配布者名・版の明示選択が一致しなければ失敗。多数の候補からのProvider選択や「最新版」の自動選択は**行わない**。
- 予期しない実行指定等を配布物に追加しても、読み込み時に不明な項目として拒否。読み取りだけでコードを起動しない。
- 配布物を読み取っても、利用側の入力元索引に供給側の実ファイルがあると**捏造しない**。入力元のファイル・範囲は配布物だけからは復元できない。
- 意味定義の**適用・受理**と、プラグインの実行権限は別の境界。意味モデルの比較試験が成功しても、いずれも認めたことにはならない。

## 実行

```sh
go test -count=1 ./reference/semantic-distribution
```

回帰試験では、独立した提供側と利用側の作業領域を一時ディレクトリに作り、実際の `workspace.Load`、`workspace.Normalize`、`validation.Validate`、`sourceindex.Fingerprint` を利用します。モックの構文解析器や独立した意味モデルは定義しません。

## 目標設計への接続条件

次期の構成Kernelと配布機構に接続するときは、今回の模擬した `Identity`、`Expectation`、`Envelope` を新しい正式仕様の並行する正本にしません。**すでに選択・検証された**配布成果物から、データだけの意味名前空間を読み取るアダプターへ縮退させる方針です。

必要な後続は、実際の配布物を扱う選択・信頼と照合値の固定、複数の名前空間間の依存閉包、公開範囲、二つのプロジェクトでの再利用、モデル更新時の影響分析です。現行の固定的な `CapabilityKind` にモデル専用種別を増やすことは避けます。

`knowledge://`を標準の参照URIとして採用する判断も、現段階では行いません。外部との相互運用で必要になるURIは、対象の種類と供給場所を混同せず別途検討します。

## E4b：独立した二Consumerと依存の明示（2026-10-13）

追加した `two_consumer_test.go` は、同じ `os` 名前空間を「監視」「監査」の異なる二Workspaceで実際に使用する。いずれも実際のTEI構文と `workspace.Load` を使用し、共有のデータのみ配布候補とPath Importの双方で、同じCanonical意味ビューと `validation.Validate` の成功を確認する。

併せて、提供側の `os.process_id` が別名前空間 `common.machine_id` を参照する場合を検証する。配布物あるいはPath Importで `os` だけを指定しても、`common` を暗黙には付加しない。不足を既存の `UNKNOWN_SEMANTIC_TYPE_UNDERLYING` 診断で確認し、利用側が両方を明示選択した場合に検証が通ることを試す。**推移的依存閉包を自動解決する実装ではない。**

二つの異なる配布者が同じ論理名前空間を異なる意味で提供するケースでは、すでに選択した配布者と内容の期待値が一致しなければ拒否する。両者を同時に合成した場合は `DUPLICATE_SEMANTIC_NAMESPACE` を診断する。自動最新版選択、後勝ち優先、RuntimeのProvider選択は実装しない。

この試験は、共通資産を二Consumerに届けられる**意味上の最小条件**を調べるものである。実際のPlugin配布バイナリ、取得・信頼、正式なComposition Profile、Shared Acceptance、業務効果の定量評価は未検証であり、#1067の共有プラグイン昇格Triggerを満たしたとの宣言ではない。

## E4c：既存BuildArtifactLockとArtifact Storeの縦断接続（2026-10-13）

`locked_asset.go` は、TEIの既存 `plugin.BuildArtifactLock` と、その中の**明示的に選択された一つの`plugin.LockedBuild`**だけを受け取る読取り専用Adapterの参照実装です。成果物バイト列は呼出元が提供する `fetch(digest)` 関数から取得します。既存のArtifact Lockを検索して最新のProviderを選ぶ処理や、新たなRegistry・第二Lockは実装しません。

現在の `plugin.BuildOutputKind` は拡張可能な文字列型であるため、**Reference限定**の `semantic-data` 出力種別を`BuildPlan`で指定し、Runtime成果物や従来のPlugin Descriptorと区別します。正式な製品用Output ABIへの採用は未決定です。旧固定`CapabilityKind`も更新しません。

`tools/reference-build-artifact-store/semantic_distribution_test.go` は、既存のローカルContent Addressed Store（#1105）に実際にデータのみの意味モデルをPutし、TEI既存の `plugin.LockBuildOutput`で作成したビルド固定情報によりGetしたバイト列を、`InspectSelectedReference`で二重照合します。StoreのDigest検証、ビルド選択のDigest・版の照合、Semantic Digest検査は異なる境界です。

受入試験では、選択したBuildPlanの偽装、Runtime種別の誤用、異なる配布者、Storeに無い成果物、Store内の内容改変を拒否します。**これは実際のStore実装と既存Build Lock構造への接続**ですが、Plugin Packageの公開・依存候補からの自動選択、Builderの出所検証、署名、Host権限、Source Indexの元ソース再構成は行いません。テストのPlanとBuilder fingerprintは構造検証用の値で、認証済み実Builderの証拠ではありません。

検証対象：
```sh
go test -count=1 ./reference/semantic-distribution ./tools/reference-build-artifact-store
```

この段階で必要なのは、Production配布機構の新しい正本を増やすことではなく、次期Composition Kernel側にある選択・信頼・取得契約を接続した後に、このReferenceの仮のExport Envelopeを置換できることの証明です。

## 外部モデル変更の影響候補（E5aの独立参照実装）

`impact.go`は、**現在のWorkspace Loaderで実際に解析した**二つの`workspace.Result`から、既存の`modelref.Graph`へ最小限の依存辺を投影します。新しい影響解析器や依存解決の正本は作りません。

`ReviewImportedTypeImpact(before, after, "os")`は、`Result.Imports[].SemanticDigest`と実際の`World.Semantics`が一致することを確認したうえで、`modelref.ReviewImpactDiff`を使用します。供給元ファイルが移動しただけなら意味照合値は変わりません。同じCanonical IDのもとで意味定義が変わった場合は、既知の利用側データ定義を**変更影響の確認候補**として示します。

このReferenceで自動抽出する依存は、**利用側が所有するSemantic Dataの属性型（`Attribute.TypeRef`）と関連先（`Relation.Target`）のみ**です。Application Action、Validation、UI、権限、外部知識サービス等の利用箇所は走査していません。したがって、候補が空でも「影響がない」とは結論できません。

入力元の物理パスや配布者名は意味上のIDではありません。Provider側の出典位置はE2のSource Indexで独立管理します。また、このGraph上の`Authority: proposal`は、人間が事実を受理したことを意味しません。新しい正式なCanonical Kindや永続的な依存Graphは追加しません。

検証コマンド：

```sh
go test -count=1 ./reference/semantic-distribution
```

否定試験として、読み込み元の消失、曖昧なImport宣言、照合値の偽装を拒否します。対象の意味が変わった際に参照中の`monitoring.watch`を候補として表示する一方、型・関連参照を持たない`monitoring.local`を推測で影響ありとみなしません。**未走査の種類には依然として不明な影響があり得ます**。
