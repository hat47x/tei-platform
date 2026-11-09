# E7a：Applicationに関係するSource Index証跡だけを取り出す

- **状態**：読み取り専用の限定Reference。Product用の証拠公開機構、Access Control、署名付きProvenance、Source Indexの第二正本ではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **接続元**：[E7：同一呼び出しのReport/Index](E7_PAIRED_OBSERVATION.md)。E7の`-paired-evidence`出力はWorkspace全体のSource Indexを含み、対象外のSource pathまで含み得る。

## 変更したこと

[小さな証跡抽出関数](e7-minimal-envelope.mjs)は、E7の一つのEnvelopeから、E5 Reportが直接参照したSubjectに結び付く`definition`、`authoring`、`implementation`、`derived`のReferenceだけを選ぶ。そこから参照されるArtifactだけを残し、対象外のWorkspace Source pathを出力しない。未定義のJSONフィールドは明示的な許可項目へ投影し、余分なキーを引き継がない。

定義Sourceが無い・複数ある、ReportとIndexのpath・指紋が食い違う、選択したArtifactが欠落・重複した場合は`unresolved`とする。外部から与えた`runtime-authorized`等の偽Qualifierは拒否する。`projected`でも`fullClosure:false`と`authority:none`は維持する。

**所有境界**：この機能はSource Indexを再構築したり、依存関係を意味解析したりしない。E5の部分的なSubject選択に従うだけである。無関係なSourceが除かれたことは、そのSourceが業務的に無関係であるという証拠ではない。

## 追加監査：Application identityと依存Relationの整合

E7aの初期実装では、`report`の`status:observed`とIndex上の定義出典が一致するだけで投影できた。そのため、**Application本体の定義参照が存在しないReport**、**`report.application`に一致しないApplication参照**、または**未対応のRelation名を名乗るSource参照**でも`projected`になり得た。GitHub上の実装を使う隔離V8検査で、`fabricated-secret`による`private/internal.tei`の投影が成立することを修正前に再現した。

修正では既存[E6のtyped Relation-to-Subject照合](e6-typed-index-bridge.mjs)を**同じSnapshotのbaseline/currentに対する自己照合**として再利用する。既知Relationの型、Application本体の定義、`report.application`との一致、E4のSource Index整合が揃わなければ`unverified_dependency_report`として投影を停止する。E7a独自の許可Relation一覧／第二Resolverは導入しない。

**限定事項**：自己照合に成功しても、外部から偽造された「形式上は有効な」E5 Reportを検知する署名や権限判定はない。必要な依存辺がすべて列挙されていること、選択されたSourceが外部への開示を許可されていることも証明しない。信頼できるE5生成と上位のACLが必要である。

## 局所試験

```sh
node --test reference/application-composition/e7-minimal-envelope.test.mjs
```

2026-10-10、Node.js v22.16.0で**11件PASS／0件失敗**。実際にNodeの標準テストランナーから対象ファイルを読み込んで実行した。正常例、選択したAuthoring/Implementation Referenceの保持、対象外Sourceの非開示、出典の欠落・重複・不一致、Fingerprint不一致、偽Qualifier、過剰なフィールドが結果へ漏れないことを確認。試験した二ファイルとGitHubに登録したファイルのGit blob SHA一致を確認した。

- 検査関数：`197e4042540c20d76331d4da62e3b24316efcc3d`
- 回帰試験：`79468bf0dd20015c309cfb8208a8435ac9cb1bb7`

**重要な非保証**：選択したSubjectのSource path・契約IDは依然として含まれる。匿名化や権限判定ではないため、外部AI・共有サービスへそのまま送信してよいという意味ではない。入力JSONが真正なE5出力であることや、Loader解析bytesとIndexが一致したことも証明しない。無関係なSourceの不正な索引があっても、この縮約関数は全Workspace Indexの検査器の代わりにはならない。

この出力をE6/E7へ再入力する場合は、**信頼できる入力から生成された同じ選択集合についてだけ**比較し、完全な索引検証・権限検証は元の所有者で実行すること。GitHub APIで回収したファイルをNodeで試す局所検証は、exact-headのフル`tools/repository-health-check.sh`、実Workspace、実SUI Targetの確認とは別である。

### 2026-10-10 追加の回帰検査

不正なRelation、Application定義なし、Application ID不一致、存在する非公開Sourceへの偽Relation参照について3件のadversarialテスト（うち複数負例）を追加し、**合計14件をV8のインメモリ互換ランナーで14 PASS／0 FAIL**確認した。既存11件のNode.js直接実行に関する記載は修正前のblobに対する過去の検証であり、**新headの正式なNode.js直接テストではない**。

このV8検証はGitHub原文E4/E6/E7aおよびE7aテストを取得してimport/exportを置換し、テスト用の基本assert関数を代替実装した限定試験である。検証原文のGit blob：E7a実装`474dda4cfe0a7a63bde27c7252a9395340392c2e`、テスト`98afba72805af76d30d7cf675bd58dc7d0a07ae7`。GoのE5コード、実Workspace、trusted exact-head Repository Healthは検証していない。
