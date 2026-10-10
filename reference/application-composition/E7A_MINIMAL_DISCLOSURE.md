# E7a：Applicationに関係するSource Index証跡だけを取り出す

- **状態**：読み取り専用の限定Reference。Product用の証拠公開機構、Access Control、署名付きProvenance、Source Indexの第二正本ではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。
- **接続元**：[E7：同一呼び出しのReport/Index](E7_PAIRED_OBSERVATION.md)。E7の`-paired-evidence`出力はWorkspace全体のSource Indexを含み、対象外のSource pathまで含み得る。

## 変更したこと

[小さな証跡抽出関数](e7-minimal-envelope.mjs)は、E7の一つのEnvelopeから、E5 Reportが直接参照したSubjectに結び付く`definition`、`authoring`、`implementation`、`derived`のReferenceだけを選ぶ。そこから参照されるArtifactだけを残し、対象外のWorkspace Source pathを出力しない。未定義のJSONフィールドは明示的な許可項目へ投影し、余分なキーを引き継がない。

定義Sourceが無い・複数ある、ReportとIndexのpath・指紋が食い違う、選択したArtifactが欠落・重複した場合は`unresolved`とする。外部から与えた`runtime-authorized`等の偽Qualifierは拒否する。`projected`でも`fullClosure:false`と`authority:none`は維持する。

**所有境界**：この機能はSource Indexを再構築したり、依存関係を意味解析したりしない。E5の部分的なSubject選択に従うだけである。無関係なSourceが除かれたことは、そのSourceが業務的に無関係であるという証拠ではない。

## 局所試験

```sh
node --test reference/application-composition/e7-minimal-envelope.test.mjs
```

2026-10-10、Node.js v22.16.0で**11件PASS／0件失敗**。実際にNodeの標準テストランナーから対象ファイルを読み込んで実行した。正常例、選択したAuthoring/Implementation Referenceの保持、対象外Sourceの非開示、出典の欠落・重複・不一致、Fingerprint不一致、偽Qualifier、過剰なフィールドが結果へ漏れないことを確認。試験した二ファイルとGitHubに登録したファイルのGit blob SHA一致を確認した。

- 検査関数：`197e4042540c20d76331d4da62e3b24316efcc3d`
- 回帰試験：`79468bf0dd20015c309cfb8208a8435ac9cb1bb7`

**重要な非保証**：選択したSubjectのSource path・契約IDは依然として含まれる。匿名化や権限判定ではないため、外部AI・共有サービスへそのまま送信してよいという意味ではない。入力JSONが真正なE5出力であることや、Loader解析bytesとIndexが一致したことも証明しない。無関係なSourceの不正な索引があっても、この縮約関数は全Workspace Indexの検査器の代わりにはならない。

この出力をE6/E7へ再入力する場合は、**信頼できる入力から生成された同じ選択集合についてだけ**比較し、完全な索引検証・権限検証は元の所有者で実行すること。GitHub APIで回収したファイルをNodeで試す局所検証は、exact-headのフル`tools/repository-health-check.sh`、実Workspace、実SUI Targetの確認とは別である。
