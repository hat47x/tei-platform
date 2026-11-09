# RQ-C1の確認方法の観測（2026-11）

## 対象

- 要求: [X-TEA知見の受容要件](../../requirements/XTEA_LOWCODE_RECEPTION_REQUIREMENTS_2026-11.md)のRQ-C1「確認方法」
- 試験: `internal/standardbusiness/rq_c1_verification_test.go`
- 対象版: `main` の `fb766fc5f` から分岐した`test/rq-c1-verification-20261110`
- 環境: Windows、Go `1.27.0`

## 要求文の「確認方法」と、この試験の対応

| 要求文 | 試験 |
| --- | --- |
| 二つの独立したパッケージ（例：検索一覧、親子保守）を同じCoreの意味に対して定義し、同じ適合試験の枠組みで比較する | `TestRQC1TwoPackagesShareTheSameCoreMeaning` |
| パッケージを取り外したときに、受理済みの意味と他の投影が壊れない | `TestRQC1RemovingAPackageKeepsTheMeaning` |
| パッケージの追加がCoreの要素種別の追加を要求しないことを、変更の差分で確認する | `TestRQC1AddingAPackageAddsNoCoreKind` |

要求文の「Pattern IDの衝突を明示的に解決する」も、`TestRQC1PatternCollisionIsReported`で確かめています。

## 観測

- `TestRQC1TwoPackagesShareTheSameCoreMeaning` 成功。検索一覧（`browse`）と親子保守（`header-detail`）の2パッケージが、同じ`sales.order`に対して別々の種別で定義され、**同じ`validation.Validate`でどちらも受理**される。
- `TestRQC1RemovingAPackageKeepsTheMeaning` 成功。親子保守パッケージを取り外しても名前空間の要素数は変わらず、残る一覧パッケージも受理される。**取り外したパッケージが消えたPatternを参照していれば`UNKNOWN_APPLICATION_PACKAGE_MEMBER`で報告**される（黙って消えない）。
- `TestRQC1AddingAPackageAddsNoCoreKind` 成功。パッケージを足しても`semantic.Namespace`の要素（Data・Capability・Type・Calendar・Period・Validation・EffectRule・RunPlan）の数は変わらず、Patternの種別も変わらない。差分はApplicationの`Packages`だけ。
- `TestRQC1PatternCollisionIsReported` 成功。2つのパッケージが同じPatternを主張すると`APPLICATION_PACKAGE_MEMBER_COLLISION`で報告される（宣言順で解決しない）。

## 試験を書く過程で、実装の要求を知った

最初の版は**存在しないAPIと不正な形**を使っていました。実装を読みながら直した点です。

| 誤り | 実際 |
| --- | --- |
| `application.PatternList` | `application.PatternBrowse`（一覧は`browse`） |
| `semantic.Namespace.Relations`・`Queries` | Relationは`Data.Relations`、Queryは`ir.World.Queries` |
| browse PatternにQueryなし | Queryは必須（`APPLICATION_PATTERN_QUERY_REQUIRED`） |
| header-detailにDataとRelationの両方 | Relationだけ（`APPLICATION_PATTERN_DATA_NOT_ALLOWED`） |
| `environment.DatabaseProviderPostgreSQL` | `environment.ProviderPostgres` |
| `application.Database` | `application.DatabaseRequirement`（`TargetDatabase`・`Access`） |
| world に database 宣言なし | `UNKNOWN_DATABASE`になるため`sales_core`を宣言 |

いずれも**要求ではなく実装の形**を知らなかったための誤りで、要求の3点は実装の形に合わせた後すべて観測できました。

## 読み取れること

- 対話の型は**Applicationの`Packages`**に置かれ、`semantic.Namespace`の要素種別を増やさない。要求の「Coreの要素種別を増やさずに」が、構造の差分として観測できる。
- 2つの型が同じ意味を共有し、同じ検証を通る。
- パッケージの取り外しは意味を壊さず、取り残された参照は報告される。
- Patternの衝突は報告され、宣言順で暗黙に解決されない。

## 読み取れないこと

- 要求文の「適合試験の枠組み」を、パッケージごとの**独立した適合試験**として実行すること（今は`validation.Validate`の共通検査を両方に通している）。
- パッケージの配布・市場・版の解決（要求の非要求）。
- X-TEAの8機能型の再現（要求の非要求）。
- 実DB・旧Runtimeでの観測（#940）。

## 位置付け

参照実装の試験記録であり、製品の受入や本番適格性を示しません。RQ-C1の「確認方法」が参照実装で観測できることを示すもので、要求の採用判断ではありません。
