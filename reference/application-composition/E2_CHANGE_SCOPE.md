# 受注明細の二種類の変更を比較する — E2小規模検証

- **状態**: 評価用の読み取り専用Reference。アプリケーション構成プロファイルや汎用解決器の実装ではない。
- **管理課題**: [#1016](https://github.com/hat47x/tei-platform/issues/1016)。前段の[既存資産調査](source-inventory.md)をもとに、UI固有の変更と環境接続の変更で必要な判断が違うことを調べる。
- **関連課題**: UI構成キット [#997](https://github.com/hat47x/tei-platform/issues/997)、データベース接続、機能契約 [#951](https://github.com/hat47x/tei-platform/issues/951)、Web固有の処理構成 [#1043](https://github.com/hat47x/tei-platform/issues/1043)。

## 背景

案件の共通構成を整える際、異なる種類の変更を一つの「上書き」で処理すると危険である。画面の見出しを変更する作業は案件側のUI資産に閉じられる可能性が高い。しかしデータベース接続先の変更は、業務アプリケーションが要求するデータベースの定義、接続先の実在、Providerの対応、秘密情報、権限、運用上の切り替えまで影響する。

この差を既存ソースで具体的に確かめるため、**ファイルを書き換えない変更提案**を二つ作った。共通のDSLや構成マニフェストは追加しない。

## 使用したソースと対象の限定

| 比較 | 既存ソース | 確認したGit blob SHA |
| --- | --- | --- |
| UI見出しの変更 | `reference/ui-customization/web-react/project-extension.mjs` | `c454f31d575be7a99b23fda6dc4955f4f8ada090` |
| 開発環境のDB接続先変更 | `reference/workspace/environments/development/environment.tei` | `4ffe7d719dbe8cbbd48b4d713a2c6f57d203c16c` |

これらは同じ業務アプリケーション `order_entry` に関連するが、同じ責務のソースではない。作業対象に含めていない業務仕様、Application Action、認証・認可、Runtime実装を書き換えない。

## ケースA：UIの見出しだけを変える

案件固有の右側パネルにある `Details` を `案件の詳細` に変更する候補を作る。対象は `project-extension.mjs` の一箇所だけであり、UI構成キットの基底実装を変更しない。

実際の既存ソースを与えると、`proposed` と `project-ui` が返る。見出しの既知部分が欠けたり複数になったりした場合は、勝手に変更位置を選ばず `unresolved` にする。生成した文字列を実行コードへ注入しないように、引用符、逆斜線、改行を含むラベルも拒否する。

**判定の限界**: この提案はUI見出しの編集箇所を特定するだけである。ReactDOMでの実描画、アクセシビリティ、案件拡張の追従、見た目や操作の受入は別に確かめる。UIプラグイン全体の再利用性を立証したわけではない。

## ケースB：開発環境のDB接続先を変える

開発環境では `order_entry.sales` が `sales_core.dev` に対応付けられ、当該実体は `sales_core` の定義と PostgreSQL Provider を参照している。ここで接続先を仮の `sales_core.dev_v2` に切り替える提案を検討する。

**現行ソースにその実体は存在しないため、結果は `unresolved: instance-not-unique-or-missing`** となる。実体を自動作成しない。

比較用の合成入力では次も確認した。

- `master_data.dev` は実在していても、定義が `master_data` であるため、受注用のDB接続先としては拒否する。
- `sales_core.dev_v2` を同じ `sales_core` 定義として一件だけ宣言すると、接続先の文字列上の変更候補は作れる。しかし結果は `proposed-unqualified` にとどめる。
- 同名実体が複数ある、もとの接続宣言が変わった、不正な識別子が指定された場合は、推測して補わず未解決とする。

接続先の候補が宣言されていても、Providerの契約適合、移行手順、Secret・Network・Permission、外部接続の実在、実行時の権限・整合性は確認できない。構成プロファイルがこれらを保証したことにはしない。

## 比較から導かれる境界

| 観点 | UIの見出し | DB接続先 |
| --- | --- | --- |
| 変更元の所有者 | 案件UI拡張 | Environment |
| 最初に確認するもの | 公開点、案件差分、見出しの一意性 | Applicationの論理DB要求、実体と定義の一致 |
| 変更候補の扱い | UIの小さな変更を提案 | 宣言されていない実体なら停止 |
| 必要な追加検証 | UI拡張の単体試験、実ブラウザ、アクセシビリティ | Provider・移行・権限・Resource・実接続・実行時の検査 |
| 触れてはならない正本 | 既存の業務意味、Action、環境接続 | 業務意味、UI Kit |
| 共通の「上書き」 | 不要 | 不適切。接続先の選択と権限は別の判断 |

**判断**: 少なくともこの二例では、既存の資産所有者が持つ変更点を維持すればよく、独立した構成解決器や全レイヤー共通の継承を追加する根拠は得られなかった。一方、案件全体で採用するUI Kitと実行先、構造パターンの選択をどこに保持するかは引き続き未解決であり、将来の実利用事例で確認する。

## 実行方法と検証範囲

```sh
node --test reference/application-composition/e2-change-probe.test.mjs
node reference/application-composition/e2-change-probe.mjs .
```

2026-10-09、Node.js 22.16.0 で合成入力による9件の試験がすべて成功した。GitHubで確認した上記二つのソースと同じGit blob SHAになるよう再現した**二ファイルの局所スナップショット**でも、UI側は `proposed`、DB側は `unresolved` となった。対象は限定した二ファイルであり、リポジトリ全体のチェックアウトや実Provider検証ではない。

本Referenceはテキスト上の既知の部分だけを照合する。TEI DSLの汎用解析器を置き換えず、PermissionやProvider Conformanceを認定せず、Sourceの変更も加えない。提案を生成したことと、利用可能な構成として受け入れたことは区別する。

**統合上の制約**: 新しいJavaScriptを含むため、ブランチの最新コミットに対する信頼できるフルチェックアウトで `tools/repository-health-check.sh` の全項目PASSと前後cleanを取得するまで、mainへ統合しない。今回の隔離環境はGitHubへの直接cloneができず、この全体検査は未実施である。
