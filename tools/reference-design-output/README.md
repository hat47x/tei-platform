# reference-design-output（#1803）

受理済みのWorkspaceから、設計者・レビュー者が読む成果物を一つのdirectoryへ導出する参照です。表定義書、機能定義書、Application × Dataの対応表とその逆表、データベース別のDDL、および全成果物を一つのrevisionへ束縛するmanifestを出力します。

```text
go run ./tools/reference-design-output/cmd/tei-design-output \
  -workspace reference/workspace -out /tmp/tei-design-output
```

`-out`は存在しないdirectoryを指定します。既存directoryを置き換えません。出力先の親directoryにstaging directoryを作り、全fileを書いてから一度のRenameで公開します。途中で失敗した場合、部分的なbundleは残りません。

## 成果物

| 名前 | 内容 | 導出元 |
| --- | --- | --- |
| `table-definitions.md` | Dataごとの属性、候補キー、関数従属性、関連、物理表 | `reference-table-definition-document` |
| `function-definitions.md` | ApplicationごとのPattern、Capability、Validation、利用する意味 | `reference-function-definition-document` |
| `application-data-matrix.md` | Application × Dataの宣言された利用 | `reference-access-matrix` |
| `data-application-matrix.md` | 同じ宣言をData側から読み直した表 | `reference-access-matrix` |
| `ddl/<database>.sql` | 提供元が出力できる新規作成用DDL | `reference-ddl-entry` |
| `manifest.json` | 導出元revision、成果物ごとのdigest、未解決、対象外 | このツール |
| `index.html` | 成果物へのリンク、revision、未解決を示す静的なナビゲーション | このツール |

`index.html`はナビゲーションであり、`manifest.json`のartifactには含めません。外部script・stylesheet・font・imageを使わず、成果物を編集する機能も持ちません。

一覧では、受理済みWorldの宣言から導出した文書・DDLと、任意の未選抜コード候補・参照用契約を分けて表示します。bundle全体を承認済みとは表示しません。各成果物の未解決事項をリンクのそばに示し、digest・byte数・構造の版は「版とファイル情報」で確認できます。Businessなど出力できない項目も別に残します。未知の種別は「その他」に表示し、文書や候補と推定しません。

コード候補を含む場合は、`generated/candidate.json`で生成器ID・宣言版・strategy・各digest、`generated/contract.json`で入力契約を確認する案内を表示します。宣言版から現在の生成器資格を判断しません。表示は既存manifestの種別情報を使い、新しい受理状態や来歴台帳を追加しません。

manifestは各成果物の`digest`（描画後byte列のSHA-256）と`structureRevision`（その文書が導出した構造の版）を持ちます。`sourceRevision`はWorkspaceの管理対象Source全体のrevisionで、一つのbundleが一つの受理済み状態から作られたことを示します。

## 文書とコード候補を同じWorkspaceから導出する

任意の `-contract` と `-strategy` を指定すると、一度読み込んだWorldを文書導出と既存の固定生成器で共有します。契約の入力属性・型と出力型をそのWorldへ照合し、不在・型不一致・不正な契約・未知strategyでは出力を公開しません。

```text
go run ./tools/reference-design-output/cmd/tei-design-output \
  -workspace reference/workspace -out /tmp/tei-design-code-output \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy template -generator-version 1
```

既存の文書・DDLに加え、`generated/`へsource（Goなら`main.go`）、`candidate.json`、入力契約の原byte列`contract.json`を収録します。全fileのdigestを既存manifestのartifact一覧へ載せ、共通の`sourceRevision`から追跡できます。`candidate.json`は生成器ID・宣言version・strategy・template digest・contract digest・source digestを持ちます。コードの変換・描画は既存の生成器を再利用し、第二の生成器や来歴台帳は作りません。flagを省略すれば従来の文書出力だけを行います。

この契約は、Workspaceで解決した属性・型を使う **reference-localな複合規則** です。複合規則そのものの受理、モデル全体のコード化、文書とコードの意味的な網羅・同値性を証明しません。sourceは`unqualified-source-candidate`として載せ、build・実行・選抜は行いません。生成器versionは呼出元の宣言で、registryの現在性や署名者Authorityの確認ではありません。固定fixtureによる選抜、実Targetでの実行、現在の実行権限は別途確認します。

これはSHO固有の書式やプレビュー・編集・保存との統合ではありません。Business cross-viewの`unsupported`も維持します。Workspaceの信頼できる受理元、読み取り中の変更を排除する原子的snapshot、入力版の真正性も、この来歴manifestだけでは保証しません。

## 保存済みbundleを現在の入力と照合する

`-verify <directory>` は、呼出元が指定したWorkspace・provider・生成器設定から同じ処理で再導出し、保存済みの全fileをbyte単位で比較します。文書、DDL、コード候補、契約、candidate manifest、`manifest.json`に加え、ナビゲーションの`index.html`も対象です。保存されたmanifestからpathや生成器設定を取り出して信用する方式ではありません。

```text
go run ./tools/reference-design-output/cmd/tei-design-output \
  -workspace reference/workspace -verify /tmp/tei-design-code-output \
  -contract tools/reference-generator-selection/testdata/line-amount/contract.json \
  -strategy template -generator-version 1
```

文書のみのbundleは、`-contract`等を付けずに照合します。コード候補付きのbundleは、生成時と同じ入力契約のbyte列・strategy・宣言versionを呼出元が指定します。`-verify`と`-out`は併用できません。成功時は確認したmanifestを出力してexit 0、不一致や読取失敗ではexit 1を返します。bundle、Workspace、stagingへの書込み、コードのbuild・実行・選抜は行いません。

変更・欠落・余分なfile、bundle内のsymlinkや通常file以外の項目を拒否します。保存側のdigestを書き直して自己整合させても、現在の入力からの再導出と一致しなければ拒否します。通常のfileは期待サイズと照合し、期待byte数＋1を上限として読みます。入力Sourceの空白だけが変わった場合も、`sourceRevision`が変われば現在版との一致にはなりません。単なる不一致を、改ざんや業務意味の変更と断定しません。

この確認は、現在の導出処理と呼出元が選んだ入力に対するbyte一致です。過去版を検証するには対応する入力・導出処理を別途保持する必要があります。ナビゲーションの表示だけが変わった場合も、`index.html`のbyte列が違えば不一致になります。同じ生成器の誤り、文書と実行の意味的一致、署名・発行主体・受理元・registryの現在性・実行権限は確認しません。同時書込みを排除するsnapshot、filesystem全体のsandbox、実DB・SHO・実Targetの受入も保証しません。自分で編集した文書を保存・レビューする経路とは別の照合であり、編集内容をSourceへ戻しません。

## 守っている境界

- 文書・DDLの導出元はWorkspaceの受理済みWorldとSource Indexです。`.tei/decisions`、`.tei/confirmations`、未受理のcandidate、Business DFDの参照用fixtureを、受理済みの意味へ昇格しません。
- 成果物は派生であり、編集してもWorkspaceへ戻りません。成果物からSourceを書き換える経路はありません。
- 一覧はPatternやCapabilityが宣言した内容です。実際の副作用、権限、実行結果の観測ではありません。
- DDLは提供元（既定はPostgreSQL）が出力できた表だけを含みます。提供元が拒否した表は`manifest.json`の`unresolved`へ残し、このツールでSQLを補いません。出力は新規作成用であり、既存データベースへの移行文ではありません。
- Business（業務手順）とFunction・Dataの対応一覧は、受理済みのBusiness Procedure sourceがまだ提供されていないため（#1125）、`unsupported`として記録し、非CanonicalなDFDから生成しません。
- 出力は時刻を含めません。同じrevisionと同じproviderからは、同じbyte列とmanifestが得られます。
- 導出元の主体（人・AI・CI）は認証しません。`sourceRevision`と生成手段だけを記録します。

## 検証

- 同じWorkspaceから2回出力し、全fileがbyte一致すること。
- `manifest.json`の`sourceRevision`が、同じWorkspaceから構築したSource Indexの`ModelRevision`と一致すること。
- 各成果物の`digest`と`bytes`が実fileと一致すること。
- 既存directoryを拒否し、staging directoryを残さないこと。
- 未受理のcandidateをWorkspaceへ置いても、成果物へ混入しないこと。
- Business cross-viewを`unsupported`として記録し、Owner（#1125）を明示すること。

## 未実装

- 本文の書式差し替え（Markdown以外の出力plugin）。`index.html`はナビゲーションのみで、本文のHTML化は行いません。
- 受理済みBusiness Procedureを用いたBusiness × Function × Dataの対応一覧（#1125待ち）。
- DDLのコメント・命名規則の選択と、PostgreSQL以外の提供元。
- 成果物上の編集を変更案へ戻す経路。編集可能な正本の位置表示は未実装です。
- 実データベースのスキーマとの照合（#1132）。
