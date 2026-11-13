# X-TEA三視点プラグイン（読取り専用）

X-TEAの設計Source（`.xead`）一件を読み、三要素分析法のData・Function・Businessの対象、関係、Function単位のCRUD、項目レベルの参照を、TEIの共通操作「三視点ナビゲーション」の結果として返すプラグインです。Studio Desktopへ載せる最初の読取り専用プラグインであり、[#1124](https://github.com/hat47x/tei-platform/issues/1124)のX1（三視点Modelerの表示）へ向けた最小の切片です。

**参照実装です。** 編集、保存、意味の受理、実DBの更新、旧Driverとの互換、帳票、バッチは含みません。

## 責務の境界

| 置き場所 | 持つもの | 持たないもの |
| --- | --- | --- |
| このプラグイン | X-TEA固有の読み方：どの構成要素をどの視点に置くか（[`reading-rules.json`](reading-rules.json)）、関係の呼び名、Source上の位置の引き方、投影の組立て | 意味の受理、Providerの選択、Workspaceの許可 |
| [`internal/xeadsource`](../../../../internal/xeadsource/) | `.xead`の読取りと、原典に明記された関係・複数の事実から導いた関係の区別（[#950](https://github.com/hat47x/tei-platform/issues/950)、[#1031](https://github.com/hat47x/tei-platform/issues/1031)） | 視点への配置、画面向けの形 |
| [`internal/operations`の三視点契約](../../../../internal/operations/three_view.go) | Source形式に依らない結果の型と、Hostが行う整合検査 | X-TEAの語彙、実装 |
| [`internal/threeviewadapter`](../../../../internal/threeviewadapter/) | 既存のComposition RegistryによるProviderの選択と、実際に入った関数の観測 | Providerの発見、適合性の認定、権限 |
| Studio（`internal/studiohost`、`cmd/tei-studio`） | 許可済みWorkspace IDから信頼できるrootを引き、選択済みProviderへ渡す | 読取りの再実装、Providerの指名 |

TEI Coreの意味モデル（`ir.World`）と、既存の固定`CapabilityKind`には何も追加していません。CRUD行は既存の標準投影[`threeelement.GenerateCRUDTable`](../../../../internal/threeelement/crud.go)で作ります。Studioの入口は既存の`StudioBridge`に一つのメソッドを足しただけで、別のGUI用Backendはありません。

### 置き場所の理由

このプラグインは、ルートmoduleの`plugins/`ではなく、Studioのmodule（`tei/cmd/tei-studio`）の中にあります。製品パッケージ（ルートmoduleの`cmd`・`internal`・`plugins`）は、X-TEAの読取り部`internal/xeadsource`が無くてもビルドできなければなりません（[`source_adapter_isolation_test.go`](../../../../tools/repository-architecture-check/source_adapter_isolation_test.go)、[#950](https://github.com/hat47x/tei-platform/issues/950)）。読取り部に依存するこのプラグインをルートmoduleへ置くと、その性質が崩れます。既存の`ObserveXead`の読取りも同じ理由でStudioのmoduleにあります。

`plugins/`配下の独立したGo moduleへ移すには、入れ子moduleを漏れなく検査する仕組みが先に要ります（[#1251](https://github.com/hat47x/tei-platform/issues/1251)）。それまでの配置です。Hostは識別子`tei.plugin.xtea-three-view`で選択するので、移しても選択の指定は変わりません。

## 区別して返すもの

結果の各項目は、次の三つの層を混ぜません。

- **原典のSource Fact**：Sourceに明記された要素と関係。要素の種別、Source上の位置、Source ID（無い・重複している、も事実として返す）。
- **構造候補**：プラグインの読み。要素をどの視点に置いたか、複数の事実をつないだ経路（業務Action→Function→Data）。後者は導出規則の指紋を持ちます。
- **受理済みMeaning**：この版では**結び付けません**。契約v0は`not-bound`以外を受け付けず、プラグインが「受理済み」と名乗る結果はHostが拒否します。

同じ対象・同じ関係は、三視点の一覧、CRUD行、項目参照のどこからも同じ`occurrence`で指します。`occurrence`はSourceの正確なバイト列の中の位置を指す住所であり、識別子（Identity）ではありません。Sourceが一文字でも変われば別の値になります。

誤った結び付けを避けるための扱いは次のとおりです。

- Source IDの無い業務Actionへ、並び順や名前からIDを作りません。`occurrence`でだけ指せます。
- 名前が同じでIDが違う要素は、別の対象のままです。互いを`sameNameOccurrences`で示すだけで、CRUDも関係も合流させません。
- 参照先のIDが重複している関係は`unresolved`とし、どちらにも線を引きません。
- CRUDはFunctionの宣言です。業務Action固有のCRUDとしては返しません。
- 読めた範囲は`coverage`で示します。視点に置けなかった要素、未知の要素、未解決の関係、表示しなかった関係を数え、一つでもあれば`partial`です。上限を超えるSourceは一部を表示せず、全体を拒否します。
- 固定した対象（pin）は呼び出し側の値です。別の対象を選び直しても変わらず、そのまま返ります。Sourceが変わった後は`stale-source-changed`と報告し、同じ名前やIDを持つ要素へ付け替えません。

## 検証

Goの実行環境で次を実行します。

```bash
go test -count=1 ./internal/threeviewadapter/ ./internal/xeadsource/
go test -count=1 -run ThreeView ./internal/operations/
cd cmd/tei-studio && CGO_ENABLED=0 go test -mod=readonly -count=1 -tags server ./...
node --test cmd/tei-studio/test/three-view-client.test.mjs
```

- [`navigate_test.go`](navigate_test.go)：合成Workspaceで、Business→Function→CRUD行→Source位置を同じ`occurrence`でたどる。層の分離、同名別ID、ID重複、ID欠落、部分取得、pinの保持と失効。
- [`host_test.go`](../../../../internal/threeviewadapter/host_test.go)：最小の答えを返す試験用Providerで、Bを選択したらBの関数へ入る（Aも同じ答えを返す）ことを確かめる。BのIDにAのコードを結んだ構成、在庫に無いID、別Workspaceについての答え、受理済みを名乗る答えを拒否する。
- [`host_three_view_test.go`](../../host_three_view_test.go)：通常のStudio Host（`newStudioHost`）の`StudioBridge.NavigateThreeView`が、このプラグインの`Navigate`と同じ結果を返す。通常の選択で入った関数のシンボルが`tei/cmd/tei-studio/plugins/xteathreeview.Navigate`であることを、Hostの観測で確かめる。

合成Workspaceは[`testdata/workspace`](testdata/workspace/)にあります。この試験のために書いた架空のデータで、Data・Function・Businessが各1件です。参照モデルや利用者のデータは含みません。

### 確認できた範囲と、できていない範囲

- 確認した：上記のGo・Node試験。Studioのserverビルド（`-tags server`）を合成Workspaceで起動し、ブラウザで`/three-view/`を開いて、Business→Function→CRUD→DataのSource位置への移動と、固定した対象が選び直しの後も保持されることを画面で確かめた。
- **未実施**：実Desktop（WebView2のウィンドウ）でのGUI受入。serverビルドは同じ`StudioBridge`と同じ画面資産を使いますが、Desktopの確認の代わりにはなりません。
- **未実施**：実際のX-TEA設計Sourceでの確認。合成データでの成功は、実Sourceを正しく読める証拠ではありません。
- 関数のシンボルと実行ファイルのダイジェストは、このプロセス自身の観測です。成果物の署名、適合性の認定、権限の付与ではありません。

## 既知の難所と、分けて進められる作業

1. **Desktopでの受入**：実WebView2で`/three-view/`を開く試験。既存のDesktop向けスモーク試験へ一画面を足す形で、この変更と独立に進められます。Workbench（`/workbench/`）への入口追加は、生成済みバンドルの再構築を伴うため別の変更にします。
2. **受理済みMeaningの結び付け**：既存の受理経路で受理されたCanonical対象へ、`occurrence`をどう結ぶか。契約のv1で扱います。受理の記録を根拠にし、名前やIDの一致からは結びません。
3. **項目レベルの参照の拡張**：現在は、キーの構成項目→項目だけです。画面・帳票の項目からテーブル項目への参照は、原典での表現を確かめてから加えます。
4. **Hostの選択機構の一本化**：`threeviewadapter`は`inspectoradapter`と同じ選択・束縛の処理を持っています。二つを一つの汎用Hostへまとめる作業は、Inspectorの経路を変えるため分けました（[#1251](https://github.com/hat47x/tei-platform/issues/1251)、[#951](https://github.com/hat47x/tei-platform/issues/951)）。
5. **CLIからの同じ操作**：`tei`コマンドに同じ型付き操作の入口を足し、StudioとCLIで同じProviderへ入ることを比べる。
6. **pinの保存**：pinは呼び出し側が持つ値で、Hostは状態を持ちません。再起動をまたぐ保存と、複数画面での共有は未着手です。
7. **`.xeaf`と複数Source**：この切片は`.xead`一件だけを読みます。設計と実行定義の対応、複数Sourceにまたがる参照は扱いません。
