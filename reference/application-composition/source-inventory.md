# 案件全体の構成要素 — 既存資産の読取り調査 E1

- **状態**: 評価待ち。実行機構の実装ではなく、既存ソースの読取りに基づく構成要素の棚卸し。
- **管理課題**: [#1016](https://github.com/hat47x/tei-platform/issues/1016)（案件全体の構成）。[設計上の正本](../../plan/design/PROJECT_PLUGIN_COMPOSITION_DESIGN.md)に沿って、二つの利用事例で必要な情報の所在を検証する。
- **確認時点**: 2026-10-08。TEI `main` = `5d5731decbc876435d6ee6c35fccbaf1386a9c6a`、SUI `main` = `1b9a320a457903d2bea0e2ba51611da408eee0ba`。下記のファイルをGitHubから個別に読んだ。全依存ソースを検証したという意味ではない。
- **要求**: RQ-003（正本を増やさない）、RQ-004（Native資産の保持）、RQ-005（ApplicationとDatabaseの独立）、RQ-008（実装交換）、RQ-013（Target追加）、RQ-015（既存資産の再利用）を、UIに閉じず案件全体で扱う。

## 何を構成するのか

画面部品を継承する仕組みだけでは、Applicationは成立しない。たとえば受注画面をReactで生成しても、注文の識別条件、実行するAction、参照先Database、認証方法、本番の権限、配備手順は別途決めなければならない。

一方、それら全部を「基本Pluginの親子関係」という一本の継承木へ入れると、UIを差し替えただけでDatabaseや業務規則まで上書き可能に見えてしまう。**上位の概念は「アプリケーション構成」であり、UI構成キットはその構成資産の一つ**とする。構成プロファイルは採用候補・選択・接続の記録であって、新たな業務仕様・実行権限・独立したPlugin Lockではない。

構成に含める関心事は、少なくとも以下を区別する。

| 関心事 | 主な再利用資産 | 接続の性質・決定権限 |
| --- | --- | --- |
| 業務・領域の意味 | Data、Function、Action、Procedure、業務仕様テンプレート | 案件のAccepted意味が正本。Template採用だけでは未確定 |
| 処理と共通Application機構 | Action実装、状態管理、通知、履歴、同期、検索等 | Capability/Contractで接続。Domain固有判断は固有実装に残せる |
| 永続化と外部接続 | Database、Object Storage、Cache、Queue、外部API等のProvider | データの所有者・Contract・接続先・Transaction境界を明示 |
| 信頼と権限 | 認証、認可、Secret、Resource、監査、隔離 | Runtimeの現在の権限が正本。構成で設定を選んでも権限は獲得しない |
| UIと操作 | UI構成キット、Renderer、Native UI、画面接続 | 見た目・入力とApplication Actionを分離。#997が個別担当 |
| アプリケーション構造 | 通常の画面処理、ドキュメント編集、オフライン同期、協調作業、イベント駆動 | 要求されるPortと故障・競合特性の候補。Web/MobileというTargetから自動決定しない |
| 実行先・実装技術 | Browser、Mobile、Desktop、Framework Bridge、言語別Host | CapabilityとTarget適合性を検査し、暗黙の代替をしない |
| 配布・運用・観測 | Build、Release、Migration、Deployment、Logging、Metrics、Health、Backup | Environment、Releaseと観測結果を分離。構成選択は実際の稼働証拠ではない |
| 開発・検査 | DSL/Editor、AIプロンプト、Test、Conformance、再生成差分 | 開発時の支援資産。検査と人間の採用判断を混同しない |

この表は責務の**分類**であり、全案件に必要なPluginの固定一覧や導入順序ではない。

## 事例 A — `order_entry`（TEI既存Reference）

| 構成対象 | 既存の参照先（TEI相対パス） | 読み取れたこと | 今回は主張しないこと |
| --- | --- | --- | --- |
| 業務意味 | `reference/workspace/semantic/sales/order.tei`、`sales/validation.tei` | Order・OrderLine・顧客参照・数量・行番号検証が別ソースに存在 | 案件ごとの識別条件や業務判断が確定したとはしない |
| Application契約 | `reference/workspace/applications/order-entry/application.tei` | `sales`は`sales_core`をread-write、`master`は`master_data`をreadで要求。適用するValidationを宣言 | UI KitがDatabase選択や検証規則を所有するとはしない |
| Action | `reference/workspace/applications/order-entry/patterns/submit-order.tei`、`realizations/submit-order.tei` | `order_entry.submit_order`は`sales.order.submit`を参照し、Custom Goによる実現が宣言される | 任意のPluginへの交換可能性をこの宣言だけで証明しない |
| 認証・認可 | `reference/workspace/applications/order-entry/realizations/authenticate.tei`、`authorize.tei` | `security.authenticate`と`security.authorize`のCustom Go実現が別にある | UI設定だけでSecurityが完結するとはしない |
| Providerと配布 | `reference/workspace/plugins/tei-database-postgres/1.0.0/plugin.tei`、`bundles/tei-reference-providers/1.0.0/bundle.tei` | PostgreSQLのDatabase Provider/Query Executorと、複数Pluginを要求するBundleを確認 | Bundleを全Runtime実装のAuthorityやKit継承親とみなさない |
| 環境差 | `reference/workspace/environments/development/environment.tei`、`production/environment.tei` | dev/prodで`sales_core`・`master_data`を別Instanceに接続。配備実行器、観測Provider、リソースGrant、Leaseを定義 | 同じProvider設定を全環境にコピーすることや、Releaseの完了を意味しない |
| 配備 | `reference/workspace/releases/solutions/order-rollout/2026.8.27.tei` | Database拡張→Application展開→旧版撤去→Database収縮を含むRelease手順候補を確認 | 定義があるだけで稼働・成功とはしない |
| UI | `reference/ui-customization/web-react/`、`tools/reference-first-web-target/` | React Starterは既存Go Web JSON契約に対応する独立Reference、Go側はHTML/JSON共通Action経路を持つ | React Starterが実Goサーバーへ組込済みとはしない |
| 業務Template | `reference/business-spec-templates/order-entry.template.json` | 三視点の再利用候補をSource参照で束ねる | Templateを採用済みApplication意味へ自動昇格させない |

**既存の接続にすでにある情報**: Data/Action・Database利用契約、dev/prod Instance、Provider/Bundleの要求、Release手順。これらを別の「案件構成ファイル」へ再記述することは避ける。

**現時点で不足する情報**: UI Kitと特定Targetへの採用判断、UI Kitの親版・案件差分のBinding、Application全体のArchitecture選択（オンライン前提かオフライン対応か等）、複数構成資産を同時更新した際の影響範囲。特にReact Starterは独立Referenceであり、Workspace内へ登録済みの正式UI Pluginではない。構成プロファイル候補には**この差だけ**を持たせれば足りる可能性がある。

## 事例 B — SUI型のNative Canvas

SUI側の既存資産は`hat47x/sui-sensemaking`に存在する。今回確認した`03_Implement/frontend/package.json`にはReact 18、Vite、Vitest等がある。Canvasの個別操作・Document型は`03_Implement/frontend/src/canvas/CardView.tsx`、`src/domain/types.ts`などSUI側の実装が所有する。API接続の責務も`src/api/client.ts`にあり、認証用HeaderやテナントSessionの制御を参照する。SUIの正本を「TEIの標準Web Form仕様」へ置き換える理由はない。

TEI側の`reference/sui-adoption/workspace/semantic/sui/island.tei`は階層意味の**一部分のReference表現**であり、親の不存在・循環整合等を完全に再現できるとは限らない（`reference/sui-adoption/README.md`）。`tools/reference-sui-interaction/`にはNative GestureとHTTPのshadow Referenceがあるが、実React Canvas/実TEI Driver/実SUI Serverを結んだ本番経路とは区別する。

SUIで構成対象になるのは、Canvas Rendererだけではない。Documentの保存先、同時更新時の競合、アクセス権限、外部認知処理のProvider、出力時の秘匿、必要ならオフライン実行・観測・配布等もある。**現SUIの製品実装とTEIのProvider/Environment境界がどこまで対応するかは未検証**であり、無理にWeb業務Application用Pluginを束ねて「SUI構成済み」と主張しない。

## 構成時の失敗を先に検査する

案件横断の仕組みはUI Kit用`extends / provide / replace / decorate`を任意の層へコピーするのではなく、**契約に適合するものを選び、明示的につなぎ、権限のある主体に処理を委ねる**ことを基本にする。

| 変更の例 | 構成で行うこと | 禁止する短絡 |
| --- | --- | --- |
| 受注画面へ独自パネル追加 | UI Kit公開点へ案件Layerを接続 | UI差分から`sales.order`の意味を上書きする |
| 開発環境のDatabase Provider変更 | Database Contract/Target・Environment Binding・Provider Conformance・Migration互換を検査 | 同じ`database`名やPlugin versionだけで代替可能と断定する |
| 独自認証から外部認証へ移行 | Auth Contract、Identity mapping、既存Policy、必要Grant・Session境界を確認 | UI Login画面の差替えだけで認可が変わった扱いにする |
| オフライン利用を追加 | ローカル保存、同期、競合、権限失効時の扱いをApplication構造として確認 | Cache Pluginの追加だけで整合性が保証されたとみなす |
| SUI Canvasを組み込む | Native Gesture、複合Action、保存、出力のそれぞれのPortとAuthorityを確認 | `pointermove`のたびに遠隔永続化を強制する |
| 企業共通構成を更新 | exactな元資産・案件差分・Lock/Contract変更を比較し、関係する範囲だけ再検証 | Packageの読込順で選び直す、既存Accepted意味やSecret値を上書きする |

## 次の狭い実証

**E1はSource所在の観察まで。自動の構成解決や実Targetでの再現は未了。** 次に行うのは、上の受注例に対し「UI部品を一つ変更」「Database接続を一つ変更」という**異種の二変更**を適用し、既存WorkspaceとExtensionでの変更先・必要な検証・未解決を比較すること。

判定時は、(1)既存Source/Environmentにすでにある情報、(2)構成候補で初めて必要になる情報、(3)実行時点で初めて判定可能な権限・状態を三つに分離する。三つ目を構成プロファイルの固定値に格納しない。新しい共通Manifestや汎用的な上書きResolverは、この実証で**具体的な不足が生じるまで作らない**。


## E1 Sourceの変化を読み取り専用で検出する

この棚卸しで参照したSourceの一部について、[E1観測器](e1-observe.mjs)と[合成fixtureテスト](e1-observe.test.mjs)を用意した。受注明細とSUI Stage 0の計14か所を、既存ファイル内の文字列が存在するかどうかだけで調べる。文字列が変わった場合、対象ファイルがない場合、参照経路がWorkspace外を指す場合は、観測が**未解決**であると報告し、自動補正しない。

```sh
node --test reference/application-composition/e1-observe.test.mjs
node reference/application-composition/e1-observe.mjs .
```

合成fixtureでは、通常の観測、Sourceの欠落・変更、パスの逸脱・Workspace外へのシンボリックリンクという**4件のテストが成功**した（Node.js 22.16.0）。この結果は、TEIの全Sourceを使った実行、TEI DSLの意味解析、Providerの適合、認可、Application Compositionの完成を保証しない。出力されるSHA-256は観測対象ファイルの内容識別子であり、Git blob SHA、Plugin LockのFingerprintや実行権限の証明ではない。

コード変更を含むため、実Repositoryのtrusted checkoutでの全体健全性検査が完了するまでは、PR #1020をDraftに保ち、mainへ統合しない。
