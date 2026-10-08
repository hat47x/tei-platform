# UI構成キット（UI Composition Kit）— 階層化Reference

対応: [#997](https://github.com/hat47x/tei-platform/issues/997)。Web/React縦切り: [#1000](https://github.com/hat47x/tei-platform/issues/1000)。**業務仕様テンプレートは別レイヤー**: [#998](https://github.com/hat47x/tei-platform/issues/998)。

## 呼称の再検討

採用候補の第一位は **「UI構成キット」（UI Composition Kit）** とする。プロジェクトに取り込んで変更できる**構成資産の集合**を指し、個々のRuntime実装を指す「Plugin」とは明確に区別する。

| 候補 | 利点 | 問題 | 判定 |
| --- | --- | --- | --- |
| 基本UIプラグイン | 既存の呼称に近い | 単一Pluginとして読みやすく、生成元・配布・Runtime実装・案件差分を混同 | **置換** |
| UIテンプレート | 短い | TEI内部の**業務仕様テンプレート**と紛らわしい。派生元だけに見える | 不採用 |
| UIフレームワーク | 汎用的 | React/Flutter等の技術基盤そのものと混同 | 不採用 |
| UIスターターキット | SIerが改造して使う起点が明瞭 | 初期コピー後も継続するExtension/Delegate/Plugin運用を表しにくい | 内部の一資産の名称 |
| UIコンポーネントキット | 可視UI資産を説明 | Host/Action接続・Prompt・案件差分を含む範囲が狭い | 部分資産 |
| **UI構成キット** | Starter、Framework Bridge、UI Renderer、拡張点、案件Layerをまとめられる | 未定義の新Core kindと誤解されないよう説明が必要 | **採用候補** |

「UI構成キット」は**論理的な配布・構成単位**であって、TEIに新しい固定Core `UIKind` / `CapabilityKind`を追加するものではない。実際のコード・Theme・Renderer・Target Bridge・Project Overlay・プロンプトは、既存Plugin / Source / Build/Target artifactへ適切にMappingする。BundleとPlugin packageと個々のRuntime Capabilityも区別する。

標準配布物の呼び分け例:

- `UI Host Bridge`: TEI Action・Result・AuthorityとWeb/React等を接続する実装。
- `UI Starter`: 案件生成・編集の出発点となるソース。
- `UI Component / Renderer Set`: Form、List、Panel等のUI部品。
- `Project UI Layer`: 案件が所有する差分・構成・Native UI。
- `UI要件整理プロンプト`: 既存資産を検査し案件要件を最小指示から抽出する開発用素材。

## 「継承・移譲・上書き」は別の仕組み

| 操作 | 意味 | 推奨する場所 |
| --- | --- | --- |
| **参照継承（extends）** | ベースKitのUI公開点を引き継ぐ。元ソースを複製せず、宣言された差分だけを加える | **デザイン時・Build時**の明示的な単一親チェーン |
| **組合せ（composition / contribution）** | 独立したUI部品・機能を既存の公開点へ追加する | Hostの**型付きExtension Point**。必要なら子要素複数・順序明示 |
| **移譲（delegation）** | UIはUI描画を、Action/Validation/PermissionはTEI側の**正本と実行契約**を利用する。Native Canvas操作の詳細はNative側に移譲する | Runtime Port／Application Action Binding |
| **明示的な上書き（replace）** | 現在採用されている特定UI公開点のRendererを交換する。置換対象と期待する前実装Identityを宣言する | Project UI Layer / Extension |
| **装飾（decorate）** | 元のUI実装を呼び出し、外側から変更する。元処理を使わない差替えとは別 | UI renderer / wrapper |
| **ソース分岐（fork）** | 契約から外れる大幅改変を案件側のSourceとして所有する | 最後の選択肢。Upstream自動追従は約束しない |

**クラス継承は第一選択にしない。** 多重継承と暗黙の優先順位は組合せ順序依存とUpgrade時の破綻を生む。UI構成キットの初期実装では、単一親の`extends` + **名前付き公開点** + **明示replace/decorate**が第一候補。複数Kitを一つの配布物で扱う場合も、Plugin依存グラフとUI Renderer優先度を自動的に一致させない。将来の複数基底は単一親の受入Evidenceで不足が判明してから設計する。

### 層モデル

```text
TEI Accepted Business Meaning (#998 は候補素材のみ)
               |
      Application Action / Data / Authority  <-- UIからはdelegate、上書き禁止
               |
    Framework/Target Host Bridge [runtime]
               |
  TEI UI構成キット Web/React [source + logical package]
               |
   企業・業界共通UI Layer [optional UI-only]
               |
      案件固有UI Layer [explicit UI-only changes]
               +-- Native component [its own gesture, undo and domain state]
```

Web/React LayerをWeb/VueやReact Nativeへそのまま継承しない。Target/Frameworkと互換majorを照合し、互換性が違えば別のUI実装を選ぶ。一方、Framework-neutralな**Action Contract**は共有できる。

## Referenceでの仮説実装

`resolve-ui-kit.mjs`は**デザイン時**のReference（実Runtime Plugin resolverではない）。Root Kitがvisual extension pointを宣言し、子Kitは次のoperationだけを使う。

- `provide`: 空の公開点にUI実装を渡す。
- `replace`: `expect`に合う現実装を明示してUI実装を交換する。
- `decorate`: `expect`が一致するときに既存Rendererを包む。

各Layerは`id`とexact `version`を指定し、`extends`もexact pinにする。解決時に継承cycle、duplicate identity、missing parent、target/framework不一致、暗黙overwrite、期待する上書き対象の不一致、実装の欠落、未宣言のSlotを拒否する。得られた各Slotには**どのLayerが何を変更したかというtrace**を付与する。

```js
// 概念例（このReferenceは未配布のKit descriptor候補）
root.slots = ['header', 'aside', 'footer'];
company.extends = {id: root.id, version: root.version};
company.changes = [{op:'replace',slot:'header',expect:'standard.header',implementation:'company.header'}];
project.extends = {id: company.id, version: company.version};
project.changes = [
  {op:'decorate',slot:'header',expect:'company.header',implementation:'project.decorateHeader'},
  {op:'provide',slot:'aside',implementation:'project.aside'}
];
// 結果: header=company-headerをproject装飾 / aside=project / footer=未指定
```

Visual Slotの名前は特定の固定enumをCoreに設けず、信頼されたUI Host Kitが宣言する。Rootが宣言していない公開点の利用は拒否する。`action.invoke`や`authorization`をUI Slotに転用する設計は受け入れない。**名称を禁止するだけではSecurityにならない**ため、本番ではFramework Hostの信頼境界とServer側のAuthority/Validationを別途担保する。

**検証:**

```sh
node --test reference/ui-customization/composition/resolve-ui-kit.test.mjs
```

2026-10-08、Node.js v22.16.0で **13/13 PASS**。テストを実行したファイルとRepositoryへ登録したGit blob SHAが一致している:
- `resolve-ui-kit.mjs`: `a852de9751e596ac0460215aa389a1d3f404c267`
- `resolve-ui-kit.test.mjs`: `805fd52a99be9dd77bee329e17566da9bee7c366`

検証したのは明示Layer構成、非破壊、上書き・装飾、exact pin、循環、名前衝突、未宣言点、権限境界の名目分離、新しいvisual slotの拡張可能性。**本物のReactDOM UIに接続した実証、Plugin Lock/Builder、生成AIの変更、実SUI Native Component、性能・可搬性実証は未了**。Referenceの`implementations`は信頼されたProject Registryからの関数であり、未検証のmanifestにある名前から任意コードを実行するものではない。

### 仮説の制限と次のGate

1. Visual Extension Pointの表示契約（ReactElement / Fragment / mobile widgetなど）を**型付き・Framework互換**で検証する必要がある。現在のReferenceはSlot名とJS関数の存在だけを検査する。
2. exact version pinは**artifact digest / Plugin Lock / source fingerprintの真正性を保証しない**。同じversion名を付けた異なるソースや悪意あるComponentはProductionでは別途拒否する。
3. 同じParentを二つのAddonが拡張し、同じSlotをreplaceするケースは、現時点では重ねられない。暗黙の順位づけを設けず、どちらの変更を採用するか**案件が明示**するまでは未解決にする。
4. Project Overrideの上書き対象がVersionアップで変わった場合は黙って追従せず`override_conflict`で止め、AIへ**元の意図・旧実装・新実装・差分**を提示して再生成・テスト・人の採否へ進む。
5. Native UIは複雑なstateやパフォーマンスをUI Kitへ吸収しない。TEI Action Contractへの委譲が成立すればよく、SUI実連携は[#993](https://github.com/hat47x/tei-platform/issues/993)が担当する。

### 参照した先行事例

- [Backstage Frontend Extensions / Extension Overrides](https://backstage.io/docs/frontend-system/architecture/extensions/) — attach point、extension tree、同一IDの明示的な差し替え、pluginのoverride。実装と構成の分離に参考になるが、そのままTEIの安全性モデルとして採用しない。
- [Backstage Frontend Plugins](https://backstage.io/docs/frontend-system/architecture/plugins/) — application-owned override packageの扱い。
- [shadcn Registry](https://ui.shadcn.com/docs/registry/getting-started) — UI部品だけでなくプロジェクトが所有するsource/config/rules等を配る考え方。Source-ownedなKitとRuntime Pluginの区別に参考になる。

これらを踏まえても、**配布単位としての「Kit」／実行機能の「Plugin」／案件の「UI Layer」／TEI内部の「業務仕様テンプレート」**を同一物とはしない。
