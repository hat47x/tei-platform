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

## 依存閉包を扱う後続段階の受入条件（E8候補・未実装）

E7の限定的な観測を、依存閉包が成立したという意味に読み替えてはならない。次段階で閉包を導入する際には、以下を**すべて**満たすまでは `limited-unchanged` より強い判定を提供しない。

1. **起点と範囲を固定する**：Application ID、Workspace snapshot、対象の定義種別、解析器のバージョン、除外した辺の種類を機械可読に示す。未知の定義種別や未対応の辺を黙って無視しない。
2. **既存解析器から辺を得る**：Pattern→Capability／Data／Relation等の推移的参照は、既存の意味解析・契約解析が確定した型付き参照を再利用する。Composition独自の文字列走査や第二の意味グラフを正本にしない。
3. **Source証跡と結合する**：各到達SubjectをSource Indexの一意なDefinitionとFingerprintに結び、欠落・複数候補・未指紋化は `unresolved` とする。名前が一致しただけで依存が解決したと扱わない。
4. **失敗側に倒す**：閉包の途中で解析不能な定義、未対応プラグイン、循環に関する未解決状態があれば、完全性を宣言しない。循環そのものは訪問済み集合で停止し、循環が許されるかどうかは既存意味規則に委ねる。
5. **同一snapshotを検証する**：Loaderが解析した入力とIndexの指紋対象の同一性を確認する。E7のEnvelopeを同じ呼び出しで生成しただけではバイト同一性や改ざん耐性は証明できない（#1017）。
6. **負例で境界を固定する**：直接参照が不変でも推移依存だけが変更された例、曖昧なSource定義、未対応の辺、循環、別aliasが同一Contractを共有する例、異なるsnapshotを混ぜた例を回帰テストにする。
7. **権限を分離する**：閉包が完全でも、互換性・生成許可・デプロイ可否・実行時認可を自動的に承認しない。これらは各専用検証器の判定に委ねる。

この節は設計上の受入条件であり、E8の実装・テスト完了を主張しない。既存のE5/E6/E7のJSON契約や終了コードは変更しない。

## E8接続点のコード調査（2026-10-10）

ブランチ上の既存コードを確認した。以下は**実装済みAPIと、未実装の接続処理の区別**である。

| 既存の型・API | 所在 | E8での役割と制約 |
| --- | --- | --- |
| `ir.World` | `internal/ir/world.go` | `Applications`、`Semantics`、`Databases`、`Plugins`、`Environments`などを保持する正規化済みM0入力。依存辺そのものを列挙するAPIではない。 |
| `sourceindex.Subject` | `internal/sourceindex/model.go` | 中立の`internal/subject.Subject`への互換alias。依存先の識別子に使用できるが、依存関係を自動生成しない。 |
| `sourceindex.Index.ReferencesFor(subject)` | 同上 | 指定SubjectのSource参照を列挙。定義参照の一意性は呼出側で検査する必要がある。 |
| `sourceindex.RoleDefinition` | 同上 | 定義証跡の選択に使用。Authoring/Derived/Implementationを定義証跡に昇格させない。 |
| `sourceindex.Index.Validate(read)` | 同上 | Artifact fingerprintとRangeの整合を読み直して検査する。Loaderが解析した入力との同一性までは証明しない。 |
| `e5UniqueSource` | `reference/application-composition/e5-world-index.go` | Definitionが一意でfingerprintが存在することを確認する既存の限定的join。E8では重複実装より共通化を検討。 |

**不足している接続契約**：既存意味解析器から、起点Subject・到達先Subject・辺の意味・解析不能理由を列挙する型付き依存辺の公開契約が必要。これは調査対象であり、上記の`World`や`Index`が既に提供しているとは扱わない。まず意味解析器側の既存処理と責務を調査し、公開するなら循環・未知種別・不正参照を区別できる失敗型を定める。Source Indexは証跡joinを担当し、意味辺の推定は担当しない。

**実装順序案**：①既存意味解析器の参照解決箇所を特定 → ②最小の型付き辺を読み取り専用で公開 → ③E5の直接依存と統合し、Source証跡へjoin → ④推移閉包と失敗側判定を追加 → ⑤間接依存だけが変化する負例を通す。現時点では①の全面調査や②以降の実装・テストは未完了。
