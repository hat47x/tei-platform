# E1 — Language Pluginを通じたSemantic候補生成の最小Reference

- 状態: 実装候補（採用済みLanguage Plugin APIではない）
- 設計: [DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md](../../plan/design/DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md)
- 作業: [#988](https://github.com/hat47x/tei-platform/issues/988)
- 対象: Accepted済みLanguage選択から実装を一意に決め、既存ParserでSemantic候補を生成するための**小さな検証**。

## なぜこのReferenceが必要か

Operatorを拡張できても、DSLの構文解析だけが固定のHost機能に閉じていると、「DSL自体がPluginである」という要件は成立しません。一方、最初からParser・Resolver・Build機構をすべて作り直すと、既存のSemantic Extensionと権限境界が二重化します。

ここでは、既存`syntax.NewParser(...).ParseSemanticFile()`を通常のProvider callbackとして利用し、別言語からも同じ`semantic.Namespace`候補を生成できることを最小限の実験対象にしています。選択済みのLanguage契約と実装fingerprintの一致だけをこのReferenceで検査し、依存解決、Lock発行、Acceptanceは上位境界へ残します。

## 実装

`selection.go`は特定のParser、構文、Canonical型に依存しないジェネリックな小型Selectorです。

```text
Already accepted language/implementation selection (exact pin)
                            |
                            v
      Resolve<T> (zero / one / multiple matching providers)
                            |
                            v
        ParseSelected<T> (returns candidate, not acceptance)
                            |
                            v
 Existing Semantic Validation / Shared Acceptance Boundary
```

`Selection`のlanguage contractとimplementation identity/fingerprintを区別します。いずれかが不足している場合や実装が見つからない場合は、入力の推測、最新versionへの自動切替、Catalog先頭Providerの採用を行いません。同一のpinに複数のProviderが該当する場合も拒否します。Parserからのerrorは`%w`で包み、既存Syntax Errorのsource位置情報を維持します。

## 比較対象

- `selection_test.go`: Provider登録順を変えても同じ候補になること、未知・未固定・fingerprint不一致・重複・Parser不在で失敗すること。
- `adapters_test.go`: 現行TEIのSemantic Parserと、Reference専用のJSON Parser callbackが同じ`semantic.Namespace`候補を生成できること。
- `adapters_test.go`: 現行Parserのsource位置つきDiagnosticがPlugin呼出し境界を越えて残ること。

**JSON Adapterはテスト内だけに置きます。** 実運用のDSLを増やしたことでも、JSONで書いた定義がAccepted Semanticになることでもありません。両Parserが同じ候補を返しても、その候補が妥当であるか、authoritative sourceへ採用できるかは別の確認が必要です。

## 実行方法

Repository rootで:

```bash
go test ./reference/dsl-language-plugin
```

コードをmainへ統合する場合はRepositoryの正規手順`tools/repository-health-check.sh`のexact-head PASSが必要です。isolatedテスト成功だけでこの条件を満たしたことにはしません。

## 今回扱わないこと

- Workspace Manifestへの新しいLanguage選択syntax
- 実際のPlugin descriptorやAccepted Semantic Setからの依存解決
- PluginのBuild / Artifact / Runtime Lock、権限を持つParserの起動
- Operatorの型推論・意味検証・SQL pushdown
- Source Indexへの実際の登録や、Source spanを別ASTに写す共通ABI
- Editorに公開する補完・整形・エラー一覧
- 正式なLanguage Plugin ABIの固定

## 次の停止線

E1から本実装へ進む前に、少なくとも**Language解決の責任者**と**Parser ABIの意味上の責務**を特定します。現行のSemantic Extension resolverと異なる依存解決を増設してはいけません。Language Sourceからの候補がSemantic Setに影響する場合、実行時に選ばれたPluginだけで意味を変えてはいけません。

E2以降は既存`positive`と`relation-unique`を基準に、Operatorの可読な表現、評価エラー、NULL/重複などの同値性へ進みます。最終構文と広範なOperator Catalogは、それらの実証まで標準化しません。


## E2試作：固定したOperator契約とSemantic参照

E2以降の参照実装は、受理済みの意味契約を**外部から正確に受け取る**前提で構成する。契約の採用、依存解決、Lockの発行は既存のSemantic ExtensionとShared Acceptance Boundaryの責務であり、このディレクトリには別の正本を作らない。

`operator.go`の`OperatorContract` / `ResolveOperator`がOperator契約照合の唯一の経路である。 旧`ComposePure`も、受理済みCatalogを照合せず任意の契約を合成できるため削除した。Operatorの型付き合成は必ず`ComposePinned`で各段階の契約を再照合する。`pinned_pipeline.go`は、構文中のOperator識別子と**位置順に与えられた**契約を照合する。`pinned_composition.go`は型の連続性を調べ、契約のバージョン・fingerprint・効果を再検証する。識別子のみで登録Catalogから選択する旧`ResolveCall` / `ParsePipeline` / `Compose`は、暗黙の意味変更を許すため廃止した。

`pinned_semantic_binding.go`は既存のSemantic NamespaceのData識別子を確認する。`semantic_path_reference.go`はData/Attribute/RelationのCanonical IDと多重度を検証する。`relation_keys_reference.go`は**複数Namespaceを含む`ir.World`**のRelation参照元・参照先を一意に特定し、結合キーの意味検査は既存の`validation.ValidateSemanticRelationKeys`へ委譲する。`expand_relation_reference.go`はこの検証結果と固定済み`std.op.expand`契約を照合する。`relation_path_reference.go`はNamespaceをまたぐ複数Relationについて、各段を検証しながら多重度を伝播する。

### 検証できること

- 同じOperator名でも、契約バージョンやfingerprintが一致しなければ採用しない。
- Operatorの入出力型が連続しない場合や、副作用を含む契約を拒否する。
- Data・Attribute・Relationの欠落、重複、未定義の多重度、結合キーの欠陥を拒否する。
- Relationの一段または複数段の連鎖で、`one`と`many`の合成を検証する。得られる多重度は**起点1件あたりに想定される結果の多重性**であり、存在保証や実レコード件数ではない。

### 既存Relation検証器との一元化（2026-10-10）

以前のReferenceは、結合キーの型一致や重複を独自に実装しており、Relationの参照元と参照先が**同じNamespaceにあること**を前提としていた。この方式では既存World上の正当なNamespace間Relationを拒否しかねなかったため、独自のキー規則を廃止した。

現在は`ValidateRelationKeys(ir.World, sourceID, relationID)`で対象のData・Relation・Targetの一意性とCardinalityを確認し、RelationのCanonical IDが**World内の別Dataと衝突しないこと**も検査する。結合キーの参照先・所属・Semantic型同一性・重複については既存の`ValidateSemanticRelationKeys`に委ねる。なお、`internal/validation/semantic_relation_keys.go`が返す対象RelationのDiagnosticを使用する。`ExpandRelationReference`と`TraceRelationPath`も同じWorldを受け取る。これにより、DSL Referenceが独自に別のキー意味論を採用する危険を抑える。

`relation_world_conformance_test.go`はsales・customer・addressの3つのNamespaceをまたぐRelationと、既存検証器が拒否する不正キーのDiagnostic同一性を試験対象とする。**この追加テストは関連Referenceソースと既存Semanticキー検証器を取り出した局所Go検証環境でPASSした**。ただし正規のTEI全ソースではなく、周辺型定義は検証用stubである。また、関数へWorldを渡した事実はShared AcceptanceやWorld全体のValidationの完了を示さない。呼び出し側が正式に受理したSemantic Worldを供給する責務は変わらない。

### Canonical fixtureと効果判定の追加確認（2026-10-10）

多段RelationのReference fixtureを`shop.order` / `shop.customer` / `shop.address`の正規Canonical ID、明示した`shop.key`型と対応するRelationキーへ修正した。Namespace間のfixtureも`customer.customer_id`型を定義した。両正常系に`validation.Validate(world)`と`validation.ValidateSemanticRelationKeys(world)`の結果を確認するテストを追加し、**キーだけ合っていて全体として無効な業務モデル**を正例にしない。

`ResolveOperator`では不完全な契約を`ErrOperatorInvalid`、明示された`pure`以外の効果を`ErrEffectEscalation`として区別した。変更後の`operator.go` / `operator_test.go`はGitHub Blob SHAと一致する局所Go 1.23.2環境で`go test`・`go test -race`・`go vet`がPASS、`gofmt -l`は空。**World全体の正例テストは実リポジトリ上で未実行**であり、この局所結果をPR統合の根拠にしない。

### まだ検証できないこと

- 本物のAccepted Semantic Setからのpin取得、Source Index、実Pluginの実現方式、実行権限
- Alias型の展開、型の幅・精度、NULLと欠損、外部結合、順序・重複、集計の同値性
- 構文引数とSource spanの共通表現、DSLとNative SQLの同値性
- 実データの結合。結合キーの整合性は、参照先レコードの存在やキーの一意性を証明しない
- 属性とRelationの完全な意味型を伝える正式な型体系。`rows/<dataID>`は移行用の文字列表現にすぎない

### 構文解析と固定契約への変換（今回の更新）

`ParsePipelineSyntax`は`|>`で区切った識別子を解析し、入力と各Operatorについて元ソースの**半開区間のUTF-8バイト位置**を保持する。構文解析段階では、契約のversion・fingerprint・入出力型を推定しない。`LowerPinnedPipeline`は外部で受理済みとされたpinと照合し、`ResolvedPipelineStep`に型・契約・位置を保持する。`ParsePinnedPipeline`は既存Reference呼び出し用の簡易Adapterであり、位置を利用する場合は前二者を使用する。Source Indexの正式なspan ABIに接続したものではない。

`ResolveOperator`はIDとversionが同じ契約がCatalogに複数ある場合、fingerprintの異同にかかわらず拒否する。つまり同じID/versionに異なる契約内容が混入しても、指定したpinが存在することだけを根拠に選ばない。

実装の一部について独立した小型Go検証環境で`go test`、`go test -race`、`go vet`と`gofmt`の実行を確認した。ただし、その検証環境にはTEIリポジトリ全体のファイルがなく、PR headに対する正規のテストPASSの証拠ではない。統合条件を満たすには、リモートのPR headそのものからクリーンなチェックアウトを用意する必要がある。

### 診断位置と局所検証（2026-10-10）

`PipelineDiagnostic`は構文と意味照合の失敗を元Sourceの半開区間（UTF-8バイト位置）へ対応付け、原因を`Unwrap`で保持する。`errors.Is` / `errors.As`によって、契約不在・不正契約の種類と位置を独立に利用できる。これはReference専用表現であり、TEIの共通Source Indexや診断ABIの置換ではない。

次の8ファイルは、GitHub側のBlob SHAと試験環境に配置した`git hash-object`の結果が完全一致した状態で、Go 1.23.2による局所検証を行った。

- `operator.go` / `operator_test.go`
- `pipeline_syntax.go` / `pipeline_syntax_test.go`
- `pipeline_diagnostic.go` / `pipeline_diagnostic_test.go`
- `pinned_lowering.go` / `pinned_lowering_test.go`

この8ファイルのみからなる試験用Go moduleで`go test -count=1 ./...`、`go test -race -count=1 ./...`、`go vet ./...`がPASSし、`gofmt -l .`は空だった。試験には`selection`、`semantic`、既存`syntax`、他のReferenceコード、repository health checkは含まれない。**PR exact-head全体の統合資格は未取得**である。

### 既存Semantic Extensionへの接続条件（調査結果）

`tools/reference-semantic-extension-catalog/`は現在`package main`の独立した検証器であり、Language Referenceから直接importできる共通APIではない。ここに別の契約解決器を設けるのではなく、**既存Semantic Catalogの解決・Lock照合が成功した後の読み取り専用Projection**を、呼び出し側から受け取る形を第一候補とする。

既存の`SemanticLock.Interfaces`はInterface ID、Version、ContractFingerprint、所属Extensionを保持する。`SemanticLock.InterfaceWirings`は提供先、Conformance Profile等を保持する。Catalog側の`CatalogSemanticInterface`は入出力の意味、unknown / failureの意味を持つ。ただし、これらをそのまま`OperatorContract`へコピーしても**完全なOperatorの型と副作用は確定しない**。現行の`OperatorContract.InputType` / `OutputType`は暫定文字列であり、`pure`というEffectをLockに無い情報から推定してはいけない。

接続する際の必要条件は次のとおり。

1. 既存手順でSemantic Lockを再検証し、同じSemantic Setのcatalog projectionを得る。source-localにLockを再発行しない。
2. Operator ID、Version、ContractFingerprintが、そのprojection内の**同一の受理済みInterface**に一致することを確認する。同じIDの別versionを自動採用しない。
3. 入出力型、NULL・評価不能の意味、副作用、Cardinality、使用するRelation等、Interface表現に足りない事項は**受理対象の追加意味契約**として明示する。Reference側で推測した値は受理権限とみなさない。
4. providerの選択と適合根拠には既存Interface Wiring・Conformanceを利用する。Operatorのcatalog登録順でproviderを再選択しない。
5. `std.*`のOperatorは既存Namespace Authorityを通す。名前空間を名乗るだけのExtensionを標準実装として採用しない。
6. 最低限の結合試験として、不一致Lock、同名別version、fingerprint drift、authority不在、unknown/failure policy不一致、副作用未宣言を拒否する。

**この接続はまだ実装・試験されていない。** したがって現時点の`pins`と`catalog`は、外部で受理済みだという前提を置いたReference入力にすぎない。

### 実行・統合の停止条件

参照パッケージ単体は`go test ./reference/dsl-language-plugin`で確認できる。コードをmainへ統合するには、`AGENTS.md`に従い、**exact-headのクリーンなチェックアウトで完全な`tools/repository-health-check.sh`のPASS**を記録する。Draft PR、部分テスト、未実行のテストコードを統合根拠としない。

次の実装判断では、既存のAccepted Semantic Setから契約pinを得る接続と、入力DSL中の演算引数・Source spanの意味付けを優先する。NULLや多重度に関する推論結果を実行挙動と混同しない。
