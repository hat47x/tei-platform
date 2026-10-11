# E4：既存Source Indexと案件構成の接続境界

- **状態**：読み取り専用Reference。製品用依存Resolver、汎用DSL解析器、新しい構成ProfileのSchemaではない。
- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)／[Draft PR #1051](https://github.com/hat47x/tei-platform/pull/1051)。前段は[E3](E3_REQUALIFICATION_AND_RESIDUAL_CHOICES.md)。
- **観察した実装**：`internal/workspace/source_index.go`、`internal/sourceindex/model.go`、`internal/workspace/loader.go`、`reference/workspace/applications/order-entry/application.tei`、`reference/workspace/semantic/sales/validation.tei`。
- **並行作業**：[PR #1111](https://github.com/hat47x/tei-platform/pull/1111) はLoader解析時のbytesとSource Indexの参照元を揃える。こちらではGoのLoader／Source Indexを変更しない。

## 確認できたことと、まだ足りないこと

Source Indexの`Index`は、`Artifact{Path, Fingerprint}`と`Reference{Subject, ArtifactPath, Range, Role}`を保持する。Fingerprintは`sha256:<64桁>`であり、E3で用いたGit blob SHA-1と**異なる識別方式**である。二つを変換したり、同じLockの証拠と見なしたりしない。

`BuildSourceIndex`は既存Workspace Loaderの`Sources`および`SourceContributions`を基にソースの所在を索引化する。ただし`Reference.Subject`が示すのは、その定義に結び付くソース位置であり、Applicationから利用先Validationへの依存辺とは限らない。

実際に`order_entry`のApplication Sourceには`uses validation sales.order_line.line_number_positive`があり、当該Validationの定義は`semantic/sales/validation.tei`にある。しかし**この利用関係をSource Indexの単なる定義一覧から推測してはいけない**。Workspace Loader／IR・意味検査から利用関係を明示的に取り出せるかは別途検証する必要がある。

## E4の読み取り専用試作

- [`e4-index-scope.mjs`](e4-index-scope.mjs)：Source IndexのJSON相当のArtifact、Reference、Role、Fingerprintを読み、**呼び出し側が明示したSubject集合**に対応するソースが変化したかを検査する。
- [`e4-index-scope.test.mjs`](e4-index-scope.test.mjs)：ApplicationとValidationに対応する**合成Index**を用いて、対象外の変更を一律失効させないことと、選択から漏れた依存先の変更を検出できない限界を検証する。
- Referenceは`definition`だけでなく`authoring`／`implementation`／`derived`の所在も対象にする。選択済みSubjectの参照先Path・Role・Fingerprintのどれかが変われば`unresolved`とし、Sourceが欠落した場合も同様に扱う。
- 重複Artifact、参照先のないReference、形式不正なFingerprint、Consumer内の重複Subjectは拒否する。形の検査は、入力Source Indexの真正性や完全性を保証しない。

```sh
node --test reference/application-composition/e4-index-scope.test.mjs
```

Node.js v22.16.0の隔離環境で**12件成功・0件失敗**。実リポジトリの`workspace.Load`→`BuildSourceIndex`が生成したJSONを渡す統合試験、trusted full checkoutのRepository Healthは**未実施**。この試験の`unchanged`は「呼び出し側が列挙したSubjectの、索引上で紐付くSource内容とRole／Pathが不変」という意味に限る。

### 意図的に示した負の例

ApplicationだけをConsumerの対象Subjectに選択した場合、そのApplication Sourceが不変なら、別ファイルのValidation定義が変更されても`unchanged`となる。Applicationの`uses validation`を**呼び出し側が依存選択へ反映していない**ためであり、業務互換性を示したことにはならない。この例は、新しい構成Profileに手作業の依存一覧を定着させる危険を明確にする。

## E4での判断

1. **Source Indexを変更影響の唯一の依存関係正本にしない。** 定義位置・内容の根拠は既存Source Indexへ委譲し、利用関係は既存Loader／IR／意味・契約検査で確定したものだけを受け取る。
2. **Application Compositionは依存辺を生成しない。** 対象Application・UI Kit等の選択を記録し、依存辺の導出元と検証結果へ参照で戻る。
3. **依存辺の欠落を`unchanged`の根拠としない。** 呼び出し側が列挙した対象についてのみ`unchanged`と返す。全体の適合判定は依存一覧の完全性を別途確認するまで`unresolved`とする。
4. **Source IndexとGit blobの指紋を混ぜない。** E3とE4は異なる観察境界を持ち、製品用の新しいLockを追加する根拠にはならない。
5. **第二ConsumerのSUIを通過したとは扱わない。** Native Canvas操作・Document保存・外部認知の依存関係はSUI側の実装が正本であり、Stage 0のSemantic参照だけでは閉じない。

次の狭い検証は、`order_entry`の`uses validation`／DB Contractを**既存の解析済み表現から**Source IndexのSubjectへ結び、選択漏れ・曖昧参照を検出すること。その後にSUI型Consumerへ同じ境界を適用して、構成記述に残る本当の不足を計測する。ここまでは新しい共通Schema、Resolver、Runtime Authorityを導入しない。
