# E15：実Goの型付きImpactをE12の読取り専用レビューへ渡す結合試験

**状態：条件付きGo結合テストを実装済み。実行・PASSは未取得。** 本Referenceは業務仕様の受理、実行権限、全依存閉包、リポジトリrevisionや解析bytesの真正性を保証しない。

管理：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。参照する並行契約は[#1120 / Draft PR #1139](https://github.com/hat47x/tei-platform/pull/1139)。Source IndexがParserと同じbytesを扱う保証は[#1017 / Draft PR #1111](https://github.com/hat47x/tei-platform/pull/1111)の担当である。

## 既存の役割を増やさずに接続する

[E14](E14_CROSS_VIEW_SOURCE_WITNESS.md)では、実`reference/workspace`の`order_entry → customer_reference → sales.order.customer → customer.customer`の**3本の直接宣言**を検査するGoテストを追加した。一方、#1120の`modelref`には、すでに**実Worldから型付きGraphを作成し、Relationの擬似的な変更を`Impact`に渡してPatternとApplicationを列挙する**Integration Testが存在する。

E15はそれらの重複試験ではなく、**E5のSource定義観測・E13の同一World由来Derived・実Goの`modelref.Impact`出力・既存E11/E12のJavaScript診断**の受け渡しが成立するかを試す。

[E15結合テスト](e15_modelref_candidate_test.go)は、**ビルドタグ`tei_modelref_candidate`のある場合だけ**コンパイル対象となる。通常の#1016にはまだ#1120のパッケージが存在しないため、デフォルトの`go test ./reference/application-composition`に未統合パッケージの依存を追加しない。

1. `workspace.Load(reference/workspace)`を一度だけ呼び、`BuildSourceIndex(root,loaded)`とE5の`inspectE5`を実行する。未解決のApplicationはここで失敗とする。
2. 同じ`loaded.World`から既存`index.Build`、#1120の`derivedbridge.FromIndex`、`modelref.Validate`、`modelref.Impact`を呼ぶ。**直接参照の逆方向候補として`order_entry.customer_reference`と`order_entry`の両方が存在する**ことを検査する。
3. E5/E13のPaired EnvelopeとGoの`{projection,diagnostics,affected}`を、権限0600のテンポラリJSONとして渡し、実際のNode.js子プロセスで[E12 CLI](e12-bound-review.mjs)を起動する。E12は内部で既存E11→E7→E6→E4を利用する。新しいGraph・DSL・Lock・変換用の意味解析器を作らない。
4. E12の終了コードは必ず**2**（再検証が必要）、出力は`status:unresolved`、`authority:none`、`fullClosure:false`、`adoption:not-decided`でなければならない。Goの型付き参照・Source診断が全て限定的に解決済みでも、**`unverified_shared_snapshot`**で停止し、PatternとApplicationのレビュー候補を保持する。
5. GoのProjectionに未解決の参照やValidateのDiagnosticがある場合、E12が`unresolved_typed_references`を返すことを確認したうえで、**結合Gate自体は失敗**させる。未解決のまま「PASS」とは数えない。

### 実Go JSONとJS側契約の違い

#1120の`derivedbridge.Projection.Unresolved`は`json:"unresolved,omitempty"`である。Goの空Sliceは、省略や`null`としてJSONへ現れる場合がある。一方E11は、`projection.unresolved`・`projection.rules`・`diagnostics`・`affected`を明示的なJSON配列として要求する。

そのためE15では、**信頼されたGoのシリアライズ地点に限って**これらを空配列`[]`として正規化する。E11／E12の外部JSON入力について「省略されていれば未解決0と仮定する」変更は行わない。これによって、Goの正常な空Sliceを受け渡しつつ、低信頼Callerによる依存・Diagnosticの省略を成功へ昇格させない。

## 実行条件

#1120はまだ別のDraftのため、次のコマンドは**#1016と#1120の変更を安全に組み合わせた破棄可能な検証用チェックアウト**でのみ実行する。

```sh
go test -tags=tei_modelref_candidate -count=1 \
  ./reference/application-composition \
  -run '^TestE15ActualGoImpactToE12ReadOnlyReview$'
```

Node.jsの`node`実行環境が必要。GoテストのPackage作業ディレクトリにあるE12 JavaScriptと、その既存Importが存在していなければ失敗する。**2026-10-11時点ではこの結合環境を取得・実行していない**。したがってPASSは記録しない。#1016の共有ブランチを強制更新したり、#1120を無条件にmainへmergeしたりしない。

## 依然として成立しない推論

- 実Goの`modelref.Impact`に渡す`version-changed`は**人工的なChange候補**であり、2つのGit revision間の実際のSource変更を観測したものではない。
- E5とmodelrefが同じインメモリ`loaded.World`を使っても、`BuildSourceIndex`の解析bytesとSource Indexの完全一致、ファイルアクセスのTOCTOU、Git revisionの固定、外部Callerの正当性までは証明しない。
- E12の出力にSource path、Canonical IDなどが含まれ得るため、試験結果の**外部LLMへの送信・表示はDisclosure認可の確認なしに行わない**。本テストはRaw Envelopeを標準出力へ表示しない。
- 実Provider互換、Business Acceptance、UI/CLI/Desktop間の機能同等性、SUI Native保存／CASはこの試験の外側である。

## 次の受入条件

最初に#1016、#1120、#1017の境界とAPI差を安全な統合Checkoutで照合し、E15の**実Go＋Node局所実行結果**を採取する。さらに、異なる固定Source snapshotのRelation変更を使ったE7→Go Impact→E12の2時点検査を実施する。

PRを正式統合するには、これと別に`AGENTS.md`所定の**exact PR head**に対するclean full checkout、`tools/repository-health-check.sh`の省略なし全項目PASS、前後clean、現在のmain照合、Draft解除が必要。E15の局所PASSだけでmainへmergeしない。
