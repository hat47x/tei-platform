# E25：別リポジトリの構成資産を変更箇所だけ再確認する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。実SUIの認証・受信・保存は[#993](https://github.com/hat47x/tei-platform/issues/993)の責務。
- **検証**：[E25のNode.js試験](e25_crossrepo_requalification.test.mjs)。新しい依存解決器や第二の固定台帳は作らず、既存の[E3変更判定器](e3-requalification.mjs)を実際のGit改訂と内容指紋に適用する。
- **固定した読取り基点**：
  - TEI：`006af2dab2269348f6f2cb2a7d63174b4666f22d`。受注明細と共通Actionの五つのファイルを確認。
  - SUI旧：`72368e7c1822fa894e2323a38f207117a521d2ed`。
  - SUI新：`4bd493f9942890d52117461886207d628f3554e2`（PR #3227）。
  - SUI旧→新は**1コミット**。GitHubのファイル比較と、指定改訂の個別ファイルのGit blob SHAを照合した。
- **位置付け**：出典を明記した読み取り専用の比較証拠であり、同時に配備済みの組合せ、正規の採用版、実行権限、構成の全依存関係を示すものではない。

## 更新されたのは何か

SUIの上記2改訂では、CIの定義1件が新規追加され、既存のテスト4件が変更された。一方、現在の共通Action接続に関わる次の**実装4ファイルの内容指紋は同じ**だった。

| SUIのファイル（`03_Implement/`以下） | 旧・新のGit blob SHA | 判定 |
| --- | --- | --- |
| `frontend/src/api/tei_card_move_action.ts` | `667e1ed411851db57b887cc77443234ffed4ce75` | 同一 |
| `frontend/src/api/client.ts` | `ca1851ecb4dafd4bab98716451575453bce8ec94` | 同一 |
| `backend/src/sui_sensemaking_api/routes/docs.py` | `d30b75e4e7ef78a07e00a10f00256536c56b3352` | 同一 |
| `backend/src/sui_sensemaking_api/card_move_command.py` | `cc46de3a60c97d48f4eab49e7bab5e5f461d40f7` | 同一 |

| SUIの証拠ファイル（`03_Implement/`以下） | 旧Git blob SHA | 新Git blob SHA |
| --- | --- | --- |
| `frontend/src/api/client.test.ts` | `1fa722f2efc4f8345c9d29b95192c90f7d43e1bc` | `c3884fd3df9e10add994134836b0863efdd69854` |
| `backend/tests/test_document_action_cas.py` | `178804f3ab7896b241f7588383264ee0500f411d` | `83305e742554313a45eea4b9481fd58973985c2a` |
| `backend/tests/test_docs_roundtrip.py` | `d84126cdcfc5ec19b116efe1078200d60b880fc8` | `d0de530a0db795165c837298b904509af7b68569` |
| `frontend/src/canvas/CardView.native_commit.test.ts` | `2f422e2350e28934476443389cf7e0a3c1d32b04` | `7d6c64bafba635cde910ebd59c16b5d7714c96e0` |
| `../.github/workflows/native-action-contract-ci.yml` | なし | `84fe3181ea2bdc7c256d54f38981e6cf385ab294` |

最終行のCI定義はリポジトリ直下の`.github/`にあり、`03_Implement/`配下ではない。旧改訂には存在しないため、新しい出典の**追加**として扱う。

TEI側では、共通Action形式`280b04486dfc65a9a213c9084ef1b56d32b157de`、受注明細Application`00c29ac0e557cb50fdcd13f2477d0b538a528c80`、送信Command`edbb31af1a2018efe1f7e830cac4d06d2f59b188`、案件UI差分`c454f31d575be7a99b23fda6dc4955f4f8ada090`、E24試験`b235400f28d4b7f94a499d76b3187ef82b738ef9`を、上記TEI改訂で読んだ。**TEI側について二つのGit改訂を比較したわけではない**。

## 既存機構の再利用方法

[E3](E3_REQUALIFICATION_AND_RESIDUAL_CHOICES.md)の`assessScopedChanges`へ、観測済みの`path`と`gitBlobSha`を渡す。正本のファイルをTEIへコピーしない。

ファイルのキーは`hat47x/sui-sensemaking/<相対パス>`、`hat47x/tei-platform/<相対パス>`とし、異なるリポジトリの同名ファイルを混同しない。これは**試験内の識別**であり、実装する第二のSource Indexや正式なLock形式ではない。

各利用先が参照するファイル一覧は、この試験が**明示的に選んだ限定的な依存**であり、自動的に導出した全依存閉包ではない。結果の`unchanged`は「列挙した内容指紋が変わっていない」ことだけ、`unresolved`は「再検証が必要」なことだけを示す。変更がなかったという理由で、適合性・本人の操作権限・実配備を承認しない。

| 限定的な利用先 | 旧→新の判定 | 理由 |
| --- | --- | --- |
| SUI Native操作送信と受信 | `unchanged` | 参照するSUIの実装4件とTEIの共通Action形式が同じ |
| 受注明細Web操作／案件UI差分 | `unchanged` | この比較ではTEI側の基点を固定した |
| SUIのクライアント・保存・画面・再読込み試験 | `unresolved` | 実テスト4件の内容指紋が変化 |
| SUI Native専用CIの証拠 | `unresolved` | 旧改訂にはCIファイルがなく、新改訂で追加 |
| TEI共通Action形式の仮変更 | `unresolved` | 受注明細とSUIの共通Action利用先の双方へ再検証が必要 |

これは実際の**改訂1件について変更5ファイルを分類した結果**である。二つの完成したアプリケーションを運用環境で構成した実績、手動の接続工程数、生成AIへの追加質問数、構成採用に必要な総変更箇所数の計測ではない。それらの値は引き続き未計測である。

## 負例と再現方法

回帰試験では次を確認する。

1. SUIの実装4件とTEIの固定されたSourceは同一と判定し、SUIのテスト4件は変更、CIは新しい出典の追加として再検証を要求する。
2. 同名のファイルでもリポジトリが違えば同一識別子としない。
3. TEI共通Action形式だけが変化した仮定では、SUI送信・受信と受注明細操作を再確認候補にする。
4. SUI受信口の出典が失われた場合は、その利用先を`unresolved`にし、無関係な受注明細を一律失効させない。
5. 同一パスの二重提示、不正な出典パス、同じ利用先内での二重依存を拒否する。

```sh
node --test reference/application-composition/e25_crossrepo_requalification.test.mjs
```

**制約**：この試験は、GitHubの特定改訂から事前に取得・照合したファイル識別子を固定している。CIのNode.js試験は毎回GitHubへアクセスして実ファイルの改訂を検証するものではない。出典の真正性・更新時の再取得・依存列挙の完全性は、それぞれ別の検証が必要である。

## 機構としての判断

既存のE3判定器とリポジトリを区別したSourceの識別によって、少なくとも今回の実際の更新については「SUIリポジトリのコミットが更新された」という事実を理由に、受注明細・SUIのすべての構成資産を一律に失効させずに済む。

ただし、型付きSUI候補とSUI実契約の意味一致、実Provider・Target・UI Kit選択、別リポジトリにまたがる成果物の署名・適合証明、現在の実行権限はこれによって満たされない。正式な構成Profile、第二の版固定台帳、Core側の新しい機構は今回も導入しない。
