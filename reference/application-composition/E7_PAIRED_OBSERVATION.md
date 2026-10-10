# E7：同じWorkspace読取りで作ったE5診断とSource IndexをE6へ渡す

- **状態**：read-onlyのEvidence接続Reference。Production用Profile Schema／Resolver／署名付きProvenance／Runtime Authorityではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **前段**：[E5](E5_PARSED_WORLD_SOURCE_JOIN.md)の型付きApplication参照抽出、[E6](E6_TYPED_INDEX_BRIDGE.md)のSource変更検査。

## E6までの不足

E6は`baselineReport`、`baselineIndex`、`currentReport`、`currentIndex`の4入力を受け取る。個別のコマンドから再収集すると、同じ論理時点・同じWorkspaceから読み出されたかを証明できない。特にSource Indexと解析済みIRの実バイト整合は[#1017 / Draft PR #1111](https://github.com/hat47x/tei-platform/pull/1111)で改善中であり、独立実行した値の文字列一致だけでは代替できない。

そこでE5にオプトインの`-paired-evidence`を追加した。従来のデフォルトJSON出力は変更せず、オプション使用時だけ`workspace.Load`→`workspace.BuildSourceIndex`の**同一呼び出し**で得た値を次のようにまとめて出す。

```json
{
  "format": "tei.e5-paired-source-observation.v1",
  "report": {"application":"order_entry","status":"observed","qualification":"declaration-and-source-index-only","dependencies":[],"issues":[]},
  "index": {"artifacts":[],"references":[]},
  "qualification": "read-only-paired-input-not-an-authority"
}
```

上記は**形式例**であり、空配列は実際の`order_entry`のSource IndexやE5検査結果ではない。`index`にはWorkspace内の定義出典・Artifact Pathが含まれるので、外部へ送信・共有する場合は適切なアクセス制御が必要。値自体が秘密情報やSource本文を出すわけではないが、Sourceパスにも秘匿すべき情報が含まれ得る。

[E7接続器](e7-paired-evidence.mjs)は、二つのオプトインEnvelopeを読んで[E6](e6-typed-index-bridge.mjs)へ委譲する。内容指紋比較器や依存解決器は増やさない。形式／限定Qualifier不一致は`unresolved`、IndexとReportの不一致、依存集合の変更、Source変更はE6に従って`unresolved`、既知の狭い参照が同じなら`limited-unchanged`を返す。成功を`fullClosure`や`authority`へ昇格させない。

## 実行方法

信頼できるクリーンなチェックアウトで、変更前後それぞれの同一Workspace snapshotを取得できる場合に限って次を使用する。作業中のブランチとmainで差分を比較したいなら、**対象のcommitを固定した別々のclean checkout**で生成する。単に時刻の異なる二つのGitHub APIレスポンスを組み合わせない。

```sh
go run ./reference/application-composition/e5-world-index.go \
  -workspace reference/workspace -application order_entry -paired-evidence \
  > baseline.json

# 固定した比較先checkoutで、同じコマンドの出力を current.json に保存
node reference/application-composition/e7-paired-evidence.mjs baseline.json current.json
```

E5は未解決時も診断JSONを出したうえで終了コード2を返す。E7は形式不正なら終了コード1、再検証を要する判定なら2、`limited-unchanged`なら0。**終了コード0であっても、全依存Closureの不変や業務互換、Plugin適合、Runtime Authorizationを意味しない。**

## 回帰試験と統合条件

- [E5のGoテスト](e5-world-index_test.go)に、Report/Indexの組がJSON往復で維持されること、未解決Reportがペア化により昇格しないことを確認する2件を追加した。
- [E7のNode.jsテスト](e7-paired-evidence.test.mjs)に、既知依存不変、Validation変更、無関係なSource変更、依存追加、形式偽装、Qualifier偽装、未解決Report、Index不一致、Index欠落、Application変更の10件を追加した。
- **この追加部分については、正確な候補headのGoテスト／Node.jsテスト／`gofmt`／`go vet`／実WorkspaceのCLI→E7実行を行えていない**。これらのテストは記述済みだが成功未確認である。
- Trusted full checkout、exact-headの`tools/repository-health-check.sh`、実行前後clean、最新mainとの差分・競合確認が揃うまでDraft継続・main統合保留。

## 残る根本的な不足

同じ呼び出しで結果をペア化しても、Source Indexの構築時に使ったバイト列とLoaderが解析したバイト列が常に同じとは限らない。この問題は[PR #1111](https://github.com/hat47x/tei-platform/pull/1111)が扱う。E7の`format`／`qualification`はJSON文字列であり、署名や改ざん検知を提供しない。

またE5はApplicationの一部の直接参照だけを照合する。特にSemantic Namespaceの代表Sourceだけから全子定義は追えず、Pattern→Data／Relation／Capability等の推移的依存、Action実装、UI Kit、Environment、SUIのNative Canvas／保存／権限は未評価である。必要な依存閉包は**既存意味・契約解析**を利用して整備する。これを理由にComposition専用の第二依存グラフを作らない。
