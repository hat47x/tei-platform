# Build Authoring Intent Reference

GitHub #1109の読取り専用Referenceです。人や生成AIが書く`BuildIntent`から、実装済みの`plugin.BuildPlan`を決定的に導出する境界を、JSONの固定入力と実ファイルで確かめます。

製品のparser、Core、`.tei`のbuild構文は変更しません。構文の確定は#1109のGateに従い、#971の現行main検証と#1101・#1103のReference収束の後に扱います。

## 分ける対象

| 名前 | 役割 | 正本か |
| --- | --- | --- |
| `BuildIntent` | 人・生成AIが管理する宣言。Source参照、Builder要求、Recipe、出力、依存の論理要求、Target、Profile、非秘密のParameter、秘密の名前だけを持つ | 正本 |
| `ResolutionContext` | Resolverが観測する外部状態への入口（Source、Builder Catalog、Build依存Catalog、Target） | 観測値 |
| `ResolvedBuildReceipt` | Intent、Contextの識別、導出した`BuildPlan`を束ねる記録 | 導出物（`authority: derived-cache`） |

宣言のidentityである`IntentFingerprint`には、Source bytes、Catalogの状態、解決結果を混ぜません。同じIntentでもSourceやCatalogが変われば、別のexact planになります。

## 導出の規則

1. `ParseIntent`と`ParseFixtureCatalog`は既存の厳格なJSON読込み処理を通し、重複キー（エスケープ表記や大文字小文字の別名を含む）、不正なUTF-8、対応しないサロゲート、末尾の別JSONを拒否します。`ParseIntent`は未知のfieldと、導出値（digest、fingerprint、locator、plan等）の手書きを拒否します。人が意図して書けるexact指定は、`builder.pin.artifactDigest`だけです。
2. Source参照は`DirSourceResolver`が実ファイルの内容から`BuildSource`のdigestを導出します。digestは相対pathと内容だけで決まり、rootのHost上の場所には依存しません。symlinkやroot外への経路は拒否します。
3. `subject.definitionSource`が指す`.tei`のPlugin定義を実際のparserで読み、Subjectと一致しなければ停止し、`PluginDefinitionFingerprint`を導出します。
4. Builderは契約、`build-tool`の権限範囲、Recipe、Target、Profileに適合するProviderを列挙し、明示指定（`provider`、`pin`）で絞ります。複数残る場合は列挙順で選ばず停止します。
5. Build依存は論理要求からCatalogのproducer outputを解決します。見つからない場合と複数ある場合は停止します。
6. 導出した`BuildPlan`は`plan.Validate`と`plugin.BuildPlanFingerprint`を通します。
7. 秘密は名前（`secretRequirements`）だけを宣言し、値はIntent、`BuildPlan`、Receiptのいずれにも入りません。秘密らしいParameter名は拒否します。

`VerifyDerivedReceipt`は、保存済みReceiptをIntentとContextから再導出した結果と照合します。人が導出物を編集しても正本にはならず、乖離として拒否されます。導出物を削除しても、同じIntentとContextから同じbytesを再生成できます。

## 試験が固定する内容

`resolve_test.go`は#1109の受入項目に次のように対応します。

- 同じIntentと同じContextで同じplan fingerprint
- field順と配列順を変えても結果が不変
- Source内容の変更でclosureとplanのfingerprintが変わり、Intentのfingerprintは不変
- Provider曖昧性で停止（Catalogの順序を逆にしても同じ結果）
- 明示wiringとpinによるexact Builder固定、pin不一致で停止
- Intentへdigestを書かなくてよい（固定入力に導出値の語を含まない）
- Target変更で、Target依存の部分（Builder、Target）だけがplanで変わる
- 秘密の値がIntent、plan、Receiptへ入らない
- 導出物の削除後の再生成と、編集した導出物の拒否

## 実行

```bash
go run ./tools/reference-build-authoring-intent \
  -intent tools/reference-build-authoring-intent/testdata/intent.json \
  -catalog tools/reference-build-authoring-intent/testdata/catalog.json \
  -source-root tools/reference-build-authoring-intent/testdata/src
```

標準出力へReceiptを書きます。ファイルの作成・変更とネットワーク接続は行いません。`-verify <receipt>`を付けると、保存済みReceiptの照合だけを行います。

## 証明しないこと

- #1101のBuilder解決、#1103のBuild graph、#1096の言語別Source closure解決の実装。`SourceResolver`、`BuilderResolver`、`DependencyResolver`、`TargetCatalog`の入口へ接続する形だけを示し、Catalogと`DirSourceResolver`はこのReference用の固定入力です。
- ProviderのConformance Evidenceの検証、Provider信頼、Build成功、Acceptance、Runtime Authority。
- Goなど特定言語のclosure（`_GOOS`付きファイル、`go.mod`、外部module）。
- `.tei`のbuild構文、Host parser、Core modelへの追加、Editor表示。
- 実際のBuild実行とBuild Lockの作成。
- Subject種別はPluginのみ。Runtime Engine、Builder等の自己構築対象は扱いません。
