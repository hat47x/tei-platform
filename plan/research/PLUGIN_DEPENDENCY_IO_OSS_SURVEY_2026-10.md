# プラグイン間の依存と入出力に関する他のOSSの調査（2026年10月）

- **役割**：特定時点の技術調査の記録。[プラグイン間の依存と入出力の対応付け（方式設計候補）](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)の根拠として、他のOSSなどがプラグインや処理の段の間で、依存、受け渡すデータ、パラメータ、既定値、変換をどう表しているかを整理する。
- **調査日**：2026-10-12。
- **再照合**：2026-10-13に公式文書の参照先と仕様の要点を読み直した。任意のサービス依存、部品の版表記、作業結果の合流範囲、子チャート値の取り込みについて、条件を明記し、補足のAPI文書を追加した。各基盤を実行した結果ではない。
- **管理課題**：[#1223](https://github.com/hat47x/tei-platform/issues/1223)。
- **正本との関係**：この文書は調査の記録であり、TEIの方式を採用するものではない。取り入れる方式の判断と、その理由は設計候補の第3節に置く。各方式の仕様の正本は、それぞれのプロジェクトの公式文書である。
- **調査の範囲と限界**：各行のリンク先の公式文書で確認できた範囲だけを記す。公式文書で確認できなかった点は含めていない。各方式の版によって挙動が変わる場合があり、調査日以降の変更は反映していない。方式の優劣を比べるものではなく、TEIの前提に照らした示唆を示す。

表の「TEIへの示唆」の列に書いた節番号は、設計候補の節を指す。

## 1. 拡張機構の例

| 方式 | 依存と入出力の表し方 | TEIへの示唆 |
| --- | --- | --- |
| [Backstage（新しいフロントエンド方式）](https://backstage.io/docs/frontend-system/architecture/extensions) | 子の拡張が`attachTo`で親の入力へ接続する。受け渡すデータは`createExtensionDataRef`で作る型付きの識別子で表す。親は必要なデータの識別子を指定して入力を宣言し、指定したものだけを受け取る。入力は既定で複数を受け付け、`singleton`と`optional`で単一・任意を表す。設定はスキーマで検証し、既定値はスキーマから与える | 受け渡す項目を型付きの識別子で扱う。多重度を受け口の側に宣言する。利用側は宣言した項目だけを受け取る。寄与（4節）の典型例である |
| [JupyterLab](https://jupyterlab.readthedocs.io/en/latest/extension/extension_dev.html) | `requires`、`optional`、`provides`をトークンで宣言し、提供元を利用側より先に起動する。一つのトークンには提供元を一つだけ登録できる。任意のサービスが無ければ`null`を渡す。変換を提供するプラグインは、依存するプラグインより先に登録する必要があると文書で注意している | 契約依存の基本形として参考にする。不在時に`null`を渡すだけでは、振る舞いが利用側の実装に隠れる。TEIでは不在時の振る舞いを宣言させる（9.2）。トークンに現れない登録順の制約は、順序制約を宣言へ出す理由になる |
| [OSGi](https://docs.osgi.org/specification/osgi.core/8.0.0/framework.module.html) | 名前空間ごとの能力を、型付きの属性と条件式で要求する。`resolution:=optional`で任意、`cardinality:=multiple`で複数の接続を表す。`uses:=`で推移的な依存の一貫性を強制する。複数の候補からは、解決済みのもの、版の高いもの、識別番号の小さいものの順に選ぶ | 属性による選択と推移的な一貫性の検査を参考にする。任意の依存は、接続されないまま解決すると後の実行時に失敗しうる。TEIでは候補の暗黙の選択はせず、RQ-032に従って未解決とする |
| [WebAssemblyの部品モデル](https://component-model.bytecodealliance.org/design/worlds.html) | 部品の「world」が、取り込むインタフェースと公開するインタフェースを宣言する。[`wac`](https://component-model.bytecodealliance.org/composing-and-distributing/composing.html)で、ある部品の公開を別の部品の取り込みへ差し込む。型にはレコード、列挙型、`option`、`result`、資源がある。worldは部品が到達できる能力の境界にもなる。構成ツールの紹介文書は、版なしの`test:mypackage`と版付きの`test:mypackage@0.1.0`が一致しない例を示す。版の推定や付与はツールによって異なる | 静的な部品構成の候補（11節）。`option`と`result`で欠損と失敗を型として区別できる。紹介例の不一致をすべての版比較へ一般化せず、使うツールの照合規則を事前に定める |
| [VS Code](https://code.visualstudio.com/api/references/extension-manifest) | `extensionDependencies`で拡張機能の識別子を指定する。[`Extension.activate()`](https://code.visualstudio.com/api/references/vscode-api#Extension)は起動した拡張の公開APIを返す。`contributes`で静的な寄与を宣言する。設定はJSONスキーマで、適用範囲を`scope`で指定する | 配布単位への依存と実行時のAPIが一体になった例である。TEIでは配布依存と契約依存を分ける（ADR-0032）。寄与と設定の型を静的に宣言する点は取り入れる |
| [Eclipse](https://help.eclipse.org/latest/topic/org.eclipse.platform.doc.isv/guide/runtime_registry.htm) | 拡張点の所有者がスキーマを宣言し、寄与する側が`plugin.xml`で登録する。[スキーマは実行時の検査には使われない](https://wiki.eclipse.org/FAQ_What_is_an_extension_point_schema%3F)と説明されている | 拡張点のスキーマを所有者が持つ形を取り入れる。TEIでは、その宣言を構成時の検査に必ず使う |
| [Cordis](https://github.com/cordiverse/cordis)（[Koishiの文書](https://koishi.chat/en-US/guide/plugin/service.html)） | `inject`で必須・任意のサービスを宣言する。必須のサービスがそろうまで待ち、そのサービスが変わると利用側を巻き戻して読み直す。`optional`の依存ではサービスの待機や変更時の巻戻しを行わず、利用側が有無を確認する | [Cordis由来の評価](../design/CORDIS_CONTINUOUS_PLUGIN_UPDATE_EVALUATION_2026-10.md)ですでに扱った。任意依存の不在時の振る舞いと、更新時の世代切替を別々に宣言する必要がある |

## 2. データの流れを持つ例

| 方式 | 依存と入出力の表し方 | TEIへの示唆 |
| --- | --- | --- |
| [Gradle](https://docs.gradle.org/current/userguide/lazy_configuration.html) | `Property`と`Provider`で出力を入力へつなぐと、作業の依存が自動で加わる。`map`で値を加工しても依存は保たれる。一方、出力をファイルの値として取り出して渡すと依存の情報が失われ、[検査で警告される](https://docs.gradle.org/current/userguide/validation_problems.html)。`convention()`は既定値で、値を設定すると無視される | 出力の参照から順序を導く（8.2）。対応付けは値を写し取るのではなく、出力項目への参照として持つ。変換を通しても参照の関係を保つ |
| [Bazel](https://bazel.build/extending/rules) | ルールの結果を型付きの`provider`として返す。属性の側で、依存先が返すべき`provider`の組を指定する。`cfg = "exec"`で構築用の道具と実行用の成果物を分ける。分析の段階では入出力を行わない | 型付きの出力と、出力の型による要求を取り入れる。構築依存と実行時の依存の分離（4節）を裏付ける |
| [Tekton](https://tekton.dev/docs/pipelines/pipelines/) | 作業ごとに`params`と`results`を宣言する。`$(tasks.X.results.Y)`で結果を参照すると順序が生じ、データを伴わない順序は`runAfter`で書く。宣言した結果が実際に作られたかは事前に検査されず、存在しない結果を参照すると実行時に失敗する。共有の作業領域を通した受け渡しには、明示の`runAfter`が要る | 出力の参照から順序を導く。受け渡しの経路によって順序の導出が抜けないよう、すべての受け渡しを項目として宣言する。宣言した出力を提供元が実際に返すことは、適合試験で確かめる |
| [Terraform](https://developer.hashicorp.com/terraform/language/block/variable) | 入力変数に型、既定値、検証条件、`sensitive`、`ephemeral`、`nullable`を宣言する。式の中の参照で依存を表すのが基本で、[`depends_on`は最後の手段](https://developer.hashicorp.com/terraform/language/meta-arguments/depends_on)とされる。`depends_on`は計画を保守的にし、確定しない値を増やすと説明されている。変数の値の出どころには固定の優先順位がある | 明示の順序制約を例外にする根拠になる（8.2）。値の出どころを固定の順で扱う（7.2）。`ephemeral`のように、秘密の性質が式を通じて伝わる扱いを参考にする |
| [Crossplane](https://docs.crossplane.io/latest/guides/function-patch-and-transform/) | パッチ（`FromCompositeFieldPath`など）で値を写し、変換を`convert`、`map`、`match`、`math`、`string`の閉じた集合から選ぶ。元の値が無い場合の扱いを`Optional`と`Required`から選び、既定の`Optional`ではパッチを飛ばす | 変換の集合を閉じて静的に検査できるようにする（6.4）。欠損時の扱いを明示させる点を取り入れる。ただしTEIの既定は「飛ばす」ではなく「停止する」とする（6.2） |
| [Dagster](https://dagster.io/docs/guides/build/assets/passing-data-between-assets) | 関数の引数名から上流の資産を特定し、依存とデータの読込みを推論する。[`AssetIn`](https://dagster.io/docs/api/dagster/assets#dagster.AssetIn)の`key`で別の資産へ対応付けられる。保存と読込みを入出力の管理役へ分け、業務の処理から切り離す。[資源の設定](https://dagster.io/docs/guides/build/external-resources/configuring-resources)は型を持つ。`EnvVar`で指定した環境変数は実行開始時に評価し、値を画面に表示しない | 引数名による推論は書きやすい一方、改名の影響を受ける。TEIでは項目の識別子で接続する（5.3）。保存の方式を業務の処理から分ける考え方は、提供元の内部での変換（6.1）と同じである |
| [CWL](https://www.commonwl.org/v1.2/Workflow.html) | 型付きの入出力を`source`で結び、すべてのデータの接続が満たされたときに段を実行する。複数の出どころは`linkMerge`と`pickValue`で扱い方を指定し、`the_only_non_null`では値が二つ以上あると誤りになる。`default`は出どころが無いか`null`のときに使う。`valueFrom`の評価順は入力間で定まらず、互いの結果は見えない | 順序をデータの接続だけから導く方式の参考になる。複数の出どころの合わせ方を名前付きの方針として宣言する（9.1）。変換を互いに独立させる。`null`と未接続を同じ既定値で埋める扱いは、TEIでは欠損の区別（5.2）に従って分ける |
| [AWS Step Functions](https://docs.aws.amazon.com/step-functions/latest/dg/concepts-input-output-filtering.html)（OSSではない） | 入力の選択、引数の組立て、結果の選択、結果の置き場所、出力の選択を別々の段階で指定する。[JSONataを使う方式](https://docs.aws.amazon.com/step-functions/latest/dg/transforming-data.html)では、式の実行時間とメモリに上限がある | 対応付けを、出どころの選択、変換、行き先への配置に分けて考える参考になる。式言語を広げると実行時の上限が必要になる点は、変換を閉じた集合にする理由になる |
| [Camunda（Zeebe）](https://docs.camunda.io/docs/components/concepts/variables/) | 入力の対応付けで局所の変数を作る。ジョブやメッセージの結果を生む作業では、出力の対応付けが無いと結果全体が親の範囲へ合流し、対応付けがあれば指定した変数だけが合流する。結果を生まない副プロセスなどにはこの既定動作を適用しない。秘密情報の参照は、ジョブを作る要素の入力の対応付けで解決する | 結果全体を利用側へ暗黙に合流させず、宣言した出力項目だけを渡す（6.2）。秘密情報が流れてよい経路を限る（7.3） |
| [NixOS](https://nixos.org/manual/nixos/stable/) | 型ごとに複数の定義の合わせ方を決め、値を一つしか持てない型に二つの定義があれば誤りにする。優先度は数値で、[`mkForce`が50、通常の定義が100、`mkDefault`が1000、選択肢の既定値が1500](https://raw.githubusercontent.com/NixOS/nixpkgs/master/nixos/doc/manual/development/option-def.section.md)であり、最も小さい値を持つ定義だけが残る | 型に応じた合わせ方と、同じ強さの値の衝突を誤りにする規則を取り入れる（7.2）。一方、数値の優先度を自由に指定できると、衝突をより強い優先度で上書きして片付けやすい。TEIでは出どころの層を少数に固定し、任意の数値の優先度を設けない |
| [Helm](https://helm.sh/docs/chart_template_guide/subcharts_and_globals/) | 親のチャートが子のチャートの値を上書きし、子は親の値を読めない。`global`は下向きにだけ渡る。[`import-values`](https://helm.sh/docs/topics/charts/#importing-child-values-via-dependencies)で子の値を親へ取り込める。公式のchild-parent例では、子から取り込んだ`myint`と`mybool`が親の同じ項目を置換する。複数の取り込みが競合する場合の一般的な優先順位は今回検証していない | 値の流れる向きを明示する。合わせ方の優先順位は、文書と検査の両方で定める |
| [Pulumi](https://www.pulumi.com/docs/iac/concepts/inputs-outputs/) | `Output<T>`を`Input<T>`へ渡すと依存が記録され、`apply`を通しても保たれる。`apply`の中で作った資源は、値が確定するまで事前の確認に現れない | 値が確定する時点を照合する（8.3）。確定していない値に依存する構成を、事前に分かる形で示す |

