# 案件構成で追加すべき情報と、変更後の再検証範囲 — E3

- **状態**: 評価待ちの読み取り専用Reference（2026-10-09）。正式な構成プロファイル、依存解決器、Plugin Lock、Runtime Authorityではない。
- **管理**: [#1016](https://github.com/hat47x/tei-platform/issues/1016)。前段は[E1：既存資産の所在](source-inventory.md)と[E2：UI／DB変更の比較](E2_CHANGE_SCOPE.md)。
- **関連**: UIキット #997、Capability契約 #951、Web要求・認証構成 #1043。共通機構へWeb固有の処理チェーンを複製しない。
- **対象要求**: RQ-003（正本を増やさない）、RQ-004（既存Native資産）、RQ-005（Application／DB分離）、RQ-008（実装交換）、RQ-013（Target追加）、RQ-015（再利用）。

## 今回確かめること

案件構成を宣言したいからといって、既存Application、Environment、Plugin Lock、UI構成キットの情報を新しいファイルへ丸ごと複製する必要はない。まず、現在のソースから確実に読み取れる内容を整理し、複数の資産を同時に採用するときに**誰が決めたか不明になる選択**だけを抽出する。

また、参照するソースの文字列が残っていても、中身の変更を見落とすことがある。更新の影響範囲を、すべての案件へ一律に広げず、関連する利用先だけへ知らせられるかを検査する。

## 既存の正本にすでにあるもの

| 案件構成の要素 | 現在の所有者・既存の所在 | 新しいプロファイルへ複製しないもの |
| --- | --- | --- |
| 業務の意味、Data、Validation | `reference/workspace/semantic/`、Applicationの`uses`宣言 | 識別子、計算規則、業務上の判定 |
| Actionと論理DB要求 | `applications/order-entry/application.tei`と各`pattern` | Actionの意味、DB契約、アクセス種別 |
| 開発・本番DBの物理接続 | `environments/development`／`production` | 接続先実体、Provider名、Credentialの実値 |
| Provider選択・依存の固定 | Bundle、Plugin定義、`.tei/plugin-lock.json` | Providerの二重Lockや推定した適合性 |
| 配備・観測・Grant | EnvironmentとReleaseのSource | Secret、許可、実行済みという事実 |
| UI実装の案件差分 | `reference/ui-customization/web-react/project-extension.mjs` | 右パネルの実装やActionを再定義する業務規則 |
| UIキット内部の親子関係 | #997のUI構成キットReference | `extends / replace / decorate`の独立した第二解決 |

これらは**GitHub上で定義の所在を観測した**という意味であり、すべてが本番用の正式なPluginとして結線済みという主張ではない。

## 現時点で残る案件固有の選択

以下は新しい製品Schemaとして採用する項目ではなく、既存資産を結び付けるための**最小限の意思決定候補**である。採用済みの定義がある場合はそれを参照し、別の正本を作らない。

| 選択の候補 | なぜ現在のSourceだけでは確定しないか | 記録するなら必要な最少情報 |
| --- | --- | --- |
| UI構成キットとApplicationの対応 | React Starterは独立Referenceで、`order_entry`の正式なPlugin選択ではない | Applicationへの参照、選ぶKitの識別子・確定版・内容指紋、接続するHostの契約 |
| 実行Target／Framework | WebであるだけではReactかVueか、WebViewかを決められない | TargetとFrameworkを別々に選び、互換性の検査先へ参照で戻す |
| Applicationの構造パターン | 同じWebでもオンライン中心、オフライン同期、Document編集は異なる | 採用する構造候補と必要Port、競合・保存等の未解決条件 |
| 案件独自の差分と更新基点 | UI資産の変更はあるが、上流Kitの更新と案件差分を関連付ける正規の記録は未実証 | 案件が所有するSourceへの参照、期待する上流資産・指紋 |
| 選択・接続を承認する主体 | 自動生成と本番での受理・実行権限は異なる | どの採用境界へ提出する変更候補か。現在のPermissionは固定値にしない |

構成プロファイルに必要だと分かったのは、このような**選択と参照の境界**であって、すべてのデータモデルやProviderを再定義する容量ではない。しかし正式なUIキットと二つ目のApplicationの実結合がないため、これらを新DSLやCore機構として固定する判断はまだ早い。

## 実ソースで確認した変化

E1で読んだ `order_entry` ApplicationのGit blob SHAは `b559df7338d590ed173ae83b6a4653fa38770377` だった。2026-10-09時点の同じファイルは `00c29ac0e557cb50fdcd13f2477d0b538a528c80` に変わり、Validation参照 `sales.order_line.line_number_positive` が追加されている。一方、案件UI拡張は `c454f31d575be7a99b23fda6dc4955f4f8ada090`、開発Environmentは `4ffe7d719dbe8cbbd48b4d713a2c6f57d203c16c` のままだった。

[E1の文字列観測器](e1-observe.mjs)は「既知の文字列がある」ことを示すため、このApplication変更を**問題なしと判定する機構ではない**。内容指紋の照合と、影響を受ける利用先の確認を分ける必要がある。

## E3で追加した局所的な再検証試験

[観測したSourceを利用先ごとに比較する試作](e3-requalification.mjs)と[合成試験](e3-requalification.test.mjs)では、**既存Sourceの内容識別子と、利用先が参照するSourceの一覧**を外部から与える。試作自体はWorkspaceを解析せず、SourceやLockを書き換えない。

- 利用先が参照するSourceの指紋が変わらなければ `unchanged`。無関係なSource変更で、全案件を失効させない。
- 対象Sourceが変更された場合は `unresolved: dependency_changed`。文字列上の変更可能性を、契約上の互換性と取り違えない。
- 必要な指紋がない、または同じSourceが重複して提示された場合は判定を止める。
- E2のUI変更候補 `proposed` は、対象Source群が変わっていないときだけ**局所編集の適用候補** `applicable` とする。正式採用や権限付与はしない。
- E2のDB接続先変更は、対象が同一のDB定義でも `proposed-unqualified` にとどまり、Provider適合やGrant等の確認が済むまで `unresolved` とする。

試験には、上記のGit blob SHAの変化を固定した一例も含む。Source単位の保守的な検査なので、Applicationファイル内の一つのValidation追加から、その影響を受けない細部まで自動的に切り分けることはできない。将来、意味・Contract単位の依存解析が成立したら、**既存の依存関係／Impact解析を再利用**する。別の本番向け依存グラフは作らない。

```sh
node --test reference/application-composition/e2-change-probe.test.mjs reference/application-composition/e3-requalification.test.mjs
```

Node.js 22.16.0の局所環境で、E2の9件とE3の11件、計20件を成功と確認した。試作のGit blob SHAと、GitHubに登録したファイルのSHAが一致することは別途照合する。**実リポジトリ全体のhealth、実React／Provider／SUI連携は未確認**である。

## 判断と次の実証条件

E3は「どのSourceが変更されたか」と「どの選択・利用先を再確認すべきか」を区別できることを示した。ただし、Source依存一覧は合成fixtureで与えており、TEIが実際の全依存を抽出できる証拠ではない。実際に使える構成支援にするには、既存の索引や契約解析で依存を導出し、資格確認・採用境界へ渡す必要がある。

次に必要なのは、**本物のUI Kit実装のApplicationへの採用**と、**SUI型Native UIが第二Applicationとして同じ接続・更新検査を通過すること**である。現行SUI Stage 0はSemanticの一部分にとどまり、正式な第二Applicationが存在するとみなしてはいけない。

新しい構成プロファイルSchema／Plugin Resolver／Capability種別は、以上の実Consumerで不足が確定するまで採用しない。今回のJavaScriptを含むブランチも、信頼できるフルチェックアウトによるexact-head健全性検査が済むまではmainに統合しない。
