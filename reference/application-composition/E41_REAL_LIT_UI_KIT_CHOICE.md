# E41：実案件Lit UIを選ぶ際のKit・Framework境界と更新競合

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)（Project/Applicationの構成選択）。
- **依存する既存の所有者**：[UI構成キット #997](https://github.com/hat47x/tei-platform/issues/997) の[設計時Reference Resolver](../ui-customization/composition/resolve-ui-kit.mjs)を直接importし、変更・複製しない。業務意味・Action・Runtime Authorityは#998/#951/#976の正本に委譲する。
- **実Source**：mainに統合済み[案件Go＋Lit PR #1567](https://github.com/hat47x/tei-platform/pull/1567)の`tools/reference-project-advice-lit/web/advice-element.mjs`、`advice-action.mjs`、`lit-core.min.js`を読み、**実Git blob SHA**（それぞれ `0f415f06f9e325b3d2f544cbbe7836f0c3f3e65f`、`609e78a87134287380edbb801efe641c6b541d66`、`30071d620d392394d56721f05fc4808e02e038e3`）と照合する。
- **再現**：[E41のNode.js回帰試験](e41_real_lit_ui_kit_choice.test.mjs)。既存Resolverに加え、実Git Blob bytesをNodeで読む。新しいManifest形式、第二のKit Resolver、Lock、Provider選択・契約受理は追加しない。

## 今回の問い

E37では、同じ**実Go補助**を2案件で再利用し、Scope別Provider選択や同一Scopeの曖昧さを既存Composition Registryで区別した。一方、統合済み案件Lit UIを標準WebのReact Starterや他のUI資産へ適用する場合には、**Target=Webが一致してもFramework／Toolkitが一致するとは限らない**。この判断をComposition Profileの新規フィールドとして重複保存すべきか、#997の既存Kit選択で十分かを最小で確認する。

### 既存UI Kit Resolverへ入力した試験用構成

| 試験用の資産 | 記述内容と目的 | 現実の証拠との関係 |
| --- | --- | --- |
| Web/Lit Root | `target=web`、`framework=lit, major=3`、公開点 `advice`／`status` | **E41の仮Kit定義**。TEIが正式なLit Kitを配布・受理済みという意味ではない |
| 案件Lit Layer | 上記Rootの**exact version**に`extends`、`advice`へ`provide` | 実在するCustom Element `tei-project-customer-advice`の埋込み候補を**合成Renderer関数**で表現するだけ |
| Web/React Root | `target=web`、`framework=react, major=18` | #997のWeb/React初期Referenceを比較先とする仮Root。実ReactDOMの統合ではない |
| 実案件Lit Asset | 実Blobで固定したLit componentと`/project/advice`の通信処理 | Sourceの実在・内容を確認。**実際のUI Kit Runtime/Hostへの組込みはしていない** |

本試験では、#997 Resolverの **`catalog` / `selected` / `target` / `framework` / `implementations`** を使う。Litを「Toolkit候補」として扱う説明用 `framework.id=lit` は、この特定の設計実験だけの入力値であって、LitをTEIの公式Framework契約に規定する採用判断ではない。

## 確かめる判断と負例

1. **明示選択**：Web/Lit Rootと案件Layerを選択すると、指定された`advice`公開点にのみ案件Renderer候補が配置され、Rootは改変されない。実SourceにはCustom Element名、ローカルLit依存、診断用`/project/advice`があるが、受注`/scenario/submit`のActionへ自動接続していない。
2. **Target一致だけでは足りない**：Web/React親をLit案件Layerの`extends`先にすると`framework_mismatch`で拒否する。同じWeb TargetであってもUIライブラリの実装互換や継承を推測しない。
3. **正確な親更新**：親Kitが`0.1.0`から`0.2.0`へ更新されても、子が旧親にpinされている限り暗黙のfallbackはしない（`missing_exact_parent`）。Project Layerが旧実装を`expect`している場合、同Version名の上流Rendererが変化しても`override_conflict`を返す。**ただしVersionだけで同一Artifactの真正性は保証されない**。
4. **案件ごとの明示選択**：異なる二案件が同じ公開点を提供しても、選択されたLayerだけが採用される。同じKit ID/versionが二つある場合は`ambiguous_kit`。
5. **Authority分離**：`authorization`や`action`の公開点を新たなVisual Slotとして捏造しない。正式なServer側権限・業務意味の保証には使わない。指定済みの実装が欠落した場合は`implementation_missing`で止め、Reactへの自動移動を行わない。

## 既存宣言で十分だった部分と、残る案件判断

#997のReferenceは、Rootの公開点、Framework/Target、明示的な単一親、`provide/replace/decorate`、`expect`による更新時の競合を扱える。**ここに第二のComposition専用`extends`や独立のKit優先度を実装する必要は観測されなかった**。

一方、案件側には「既存React Starterを使い続けるのか、Lit componentをWebの別Kit/Layerとして採用するのか」「どの親Kitのどの公開点に置くのか」「上流が変わった場合に案件の差分を適用し直すのか」という**未決定の選択**が残る。今回の試験が成功しても、それらの問いを人へ何回質問したか、生成AIが何回再生成したかという実イベントは計測していない。

また、Lit Custom ElementをReact Componentとして実マウントするブラウザーE2E、標準受注UIと案件Layerの同時動作、正式UI Kit Artifactの認証、Host Authority、実SUI Native Consumer、Production利用先の受理は未実施。既存#1567の独立診断Hostの実Chrome成功を、このKit組合せがChromeで成功したことへ流用しない。

## 実行

```sh
node --test reference/application-composition/e41_real_lit_ui_kit_choice.test.mjs
```

実SourceのBlobが改訂された場合は、元の固定証拠から外れたことを明示的に拒否し、再資格確認する。新しいComposition Profileを今すぐ正式採用するのでなく、**既存の#997 Kit選択と案件Layerをまず実Consumerで利用する**方針を維持し、追加Profileの採否はWeb/SUI両Consumerの同条件作業計測まで保留する。
