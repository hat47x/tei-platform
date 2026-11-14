# 生成AIによるコードジェネレータ作成の標準方式（2026-11）

- **状態**：設計案。採用済みADRではない。最小実証（`positive` ValidationからGoの検証プログラムを生成）だけをReferenceとして実装した。正式Schema、実Target、実行許可、Plugin化は未実装・未検証。
- **基準main**：`305e024c`（2026-11-13）。
- **前提とする既存判断**：[ADR-0022](../adr/0022-ai-proposal-canonical-acceptance-boundary.md)（AI提案と受理の境界）、[ADR-0045](../adr/0045-generator-artifact-lifecycle-and-evidence.md)（提案中。生成器を版付き成果物として扱い、固定fixtureで選抜する）、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)。
- **既存担当との分担**：選抜とSource束縛は#2595、builder自体のself-buildは#943 / #979 / #1066、段階的Loweringは#1121、型付きGraphを生成器入力にする研究は#4198、DSL構文は#4359、Plugin契約は#951、意味Roleは#2620、検証済み組合せは#3267が所有し続ける。本書が扱うのは「**生成AIが生成器を作るとき、何を入力として固定し、何を成果物として残し、何で合否を決めるか**」だけである。
- **実装**：[`tools/reference-generator-selection/`](../../tools/reference-generator-selection/README.md)の`validation`契約と`testdata/validation-positive/`。

## 1. 区別する4つのもの

この文書では、次の4つを別の成果物として扱う。混同すると、生成AIが書いたプログラムの成功を、生成器の適合として数えてしまう。

| 名前 | 内容 | 最小実証での実体 |
| --- | --- | --- |
| Generator Source | 生成器そのもののsource。生成AIが書く対象 | Transform Source（`*.transform.json`） |
| 実行Generator | Generator Sourceを実行する仕組み | `internal/plugin/transform`の評価器 |
| Target成果物 | 実行Generatorが出力したコード | 生成されたGoの検証プログラム |
| 独立Expected | 生成器と無関係に、先に固定した期待値 | 手書きの`fixtures.json` |

Generator Sourceを生成する過程（生成AIの呼出し）と、Generator SourceからTarget成果物を得る過程（生成器の実行）は別の過程である。前者は非決定的で、後者は決定的でなければならない。

## 2. 3方式の比較

| 観点 | A. AIがTransform Sourceを書く | B. AIが外部Generatorを書く | C. AIがGenerator Definitionを書き、既存Builder/Loweringが実装を作る |
| --- | --- | --- | --- |
| 生成AIが書くもの | `deterministic-transform-v1`のSource | 任意言語のプログラム | 宣言（どの意味から、どのTargetへ、どの規則で） |
| 表現力 | 構造の組立てと文字列の連結だけ。生成器の中では算術も分岐もできない | 制限なし | Builderが対応する範囲だけ |
| 決定性 | 評価器の構造で保証される（時計・乱数・I/Oを持たない） | 観測でしか確かめられない（二重build、digest比較） | Builderの決定性に従う |
| 資源上限 | Sourceが宣言し、評価器が強制する | 外部の隔離機構が要る（#2515） | Builderに従う |
| 副作用 | 起こせない | 起こせる。現行の選抜は検出であり隔離ではない | Builderに従う |
| 実際に読まれたSourceの束縛 | 評価器へ渡したbytesがそのまま実行される | 残余あり（#2595。Scriptの実読込byte、依存コード） | Builderに従う |
| Target成果物の意味上の正しさ | 保証しない。独立Expectedでしか分からない | 同じ | 同じ |
| 現行mainでの状態 | 実装済みの評価器を使える | Reference実装あり（Go/Java/Python/JavaScript/WASM/WASI） | **未実装**。生成器の宣言を実装へ落とす既存Builder/Loweringは無い |

### 判断

1. **方式を一つに固定しない。合否の判定を共通にする。** どの方式の生成器も、同じGenerator Requirementと同じ独立Expectedで、Target成果物を評価する。由来で合否を変えない（ADR-0045決定2）。
2. **既定はA。** 生成器の仕事が「受理済み定義の値を、固定した雛形へ差し込む」範囲に収まるなら、Aを使う。決定性・上限・無副作用を追加の観測なしで得られるためである。Target成果物の中には算術や分岐を書けるので、Aの制約は生成器側の表現力だけに掛かる。
3. **Bは、Aで書けないときに使う。** 該当するのは、入力に応じた計算や条件分岐が生成器側に要る場合、外部toolchainを呼ぶ場合、Aの上限を超える場合である。Bを選んだ生成器には、決定性と副作用の保証が無いことをreceiptの読み手が分かるよう、由来（`origin`と言語）を残す。保証の不足は#2595と#2515の範囲で埋める。
4. **Cは今は作らない。** 新しい汎用コンパイラを要するためである。Cに相当する要求は、「適合済みの生成器を選び、受理済み定義を入力として渡す」ことで満たす。生成AIが宣言だけを書けば済む範囲を広げたいときは、#1121のLowering契約が実装へ進んだ後に、その契約の上で検討する。

## 3. 共通の最小契約

新しいSchemaは作らず、`reference-generator-selection`の既存構造へ対応づける。

| 契約 | 必須の内容 | 既存の実体 |
| --- | --- | --- |
| Generator Requirement | 受理済み意味のCanonical ID、入力の属性とprimitive、結果の状態の種類、Targetの言語と入出力形式、独立Expectedの参照 | `Contract`（`validation`または`projection`、`rounding`）と`Fixture` |
| 実装候補 | 由来、言語、source path、source digest、生成AIを使った場合はprompt・context・出力のdigestと契約digest | `CandidateManifest`、`GenerationProvenance` |
| Generator Source | bytesのdigest、形式（Transform Sourceか外部sourceか）、メタ生成の深さ | Goの生成器は`Lineage`。Transform Sourceは`GenerationProvenance`の既存欄（下記の注） |
| Build Artifact | Target成果物のsource digestとbinary digest | 選抜receiptのbuild記録。build計画が要る規模では`BuildPlan`（#2831） |
| Conformance | 契約digest、fixture、候補ごとの全caseの期待値と実際の値 | 選抜receipt |

**注**：既存の`Lineage`の検証は、Goで書かれた生成器をbuildして再実行する。Transform Sourceはこの検証に載らないため、最小実証では`GenerationProvenance`の`promptDigest`へGenerator Sourceのdigestを、`contextDigest`へ入力のdigestを入れ、候補directoryへ両方のbytesを複製した。選抜は両方をdigestで照合する。Sourceを再評価して候補と一致するかどうかは`VerifyTransformCandidate`が確かめる。「prompt」という欄名はTransform Sourceの実態と合っておらず、欄の整理は10章の未解決に残す。

Conformanceが示すのは、「この契約、このfixture、このTarget成果物の組で、全caseが一致した」ことだけである。次のものは示さない。

- 意味の受理（ADR-0022）。
- 実行許可（ADR-0045決定6）。
- 別のTarget、別の契約版、fixtureに無い入力での適格性。

## 4. 停止条件

生成器を作る前、または生成器を実行する前に、次のどれかが未解決なら止める。既定値で埋めて進めない。

| 未解決のもの | 判定の仕方 | 最小実証での実装 |
| --- | --- | --- |
| Semantic Meaning | 契約が指す定義が、受理済みWorkspaceに無い | 契約のValidation IDがWorkspaceに無ければ選抜を拒否する |
| 意味と契約の食い違い | 契約の属性・種類が、受理済み定義と一致しない | 属性またはkindが違えば拒否する |
| 生成器が対応しない種類 | 生成器が扱える種類を自分で列挙し、それ以外を拒否する | Transform Sourceの`assert-enum`で`positive`以外を拒否する |
| 型と入力の表現 | 入力のprimitiveと、その受渡し形式が契約に無い | decimalはJSON文字列で受ける。JSON数値は**未解決**としてfixtureに入れていない |
| NULL相当 | 欠落・null・評価不能を、業務上の不成立と区別する規則が無い | `unevaluable`を`not-satisfied`と別の状態にし、理由を`code`で区別する |
| 精度 | 数値を返す契約に丸めが無い | 既存の`rounding`検査。判定だけを返す契約では、判定を厳密な有理数で行うことを独立Expectedで強制する |
| Effect | 生成するコードが純粋でない（I/O、時刻、外部呼出しを要する） | 対象外。純粋でない生成対象は、Effectの担当（#988 / #1121）が決まるまで作らない |
| Target制約 | 言語、入出力形式、実行形態が契約に無い | 候補manifestの`language`と既存の候補入出力形式 |

生成AIが、足りない意味を**提案**することは妨げない。ただし提案は受理の入口（ADR-0022）へ出すものであり、Generator Requirementが参照できるのは受理済みの定義だけである。提案された意味を使って生成器を先に作り、後から受理を求める順序は取らない。

## 5. 決定性と、生成AIの過程の分離

**同じGenerator Sourceから同じTarget成果物を得る条件**は、次の4つが同じであることである。

1. Generator Sourceのbytes（digestで識別する）。
2. 実行Generatorのidentityと版（方式Aでは評価器とProfile。方式Bではtoolchain・flag・環境も含む）。
3. 入力のbytes（受理済み定義から取り出した値）。
4. 宣言した上限。

方式Aでは、この4つが同じなら出力が同じであることを評価器の構造が保証する。方式Bでは保証できないので、二重実行のdigest比較という観測に留まる。

**生成AIの過程**は、これと切り離す。

- 生成AIの呼出しは再現の対象にしない。prompt・context・出力のdigestを一度記録するだけにする（既存の`tei-generator-record`と`GenerationProvenance`）。
- 再現、検証、失効の判断は、記録されたGenerator Sourceのbytesから始める。モデルを呼び直して同じSourceが出るかどうかは問わない。
- 生成AIがGenerator Sourceを書き直したら、別のdigestを持つ別の候補として選抜をやり直す。古い合格結果を引き継がない（ADR-0045決定3）。
- 生成AIは、Target成果物を直接書いて生成器の出力として差し出してはならない。Target成果物は、記録されたGenerator Sourceを実行して得たものだけを使う。最小実証では、候補のsourceを試験の中で毎回Generator Sourceから生成し、checked-inの生成物を置かない。

## 6. 生成器を生成器で作る場合

- **Bootstrap**：最下層は、Goで書かれた評価器とGo toolchainである。これを隠さず、stage0として扱う（#943 / #979と同じ扱い）。方式Aの生成器は、この評価器を信頼の前提とする。
- **深さ**：メタ生成の深さは`Lineage.depth`で記録する。深さの上限は生成対象ごとに先に決め、超える生成は新しい変更案として扱う（ADR-0034）。
- **合否は最下段で決める**：何段のメタ生成を経ても、合否を決めるのは最終のTarget成果物と独立Expectedの比較である。上位の生成器が合格していることは、下位の生成器やTarget成果物の合格を意味しない。
- **循環の禁止**：
  - 生成器、およびその生成器から作られた生成器は、自分を評価する独立Expectedを生成してはならない。
  - 生成器は、自分のConformance記録を書いてはならない。記録は選抜側が作る。
  - 独立Expectedは、参照実装（`validationeval`など）の出力を写して作ってはならない。参照実装との一致は、差分確認として別に行う。
- **信頼境界**：生成器が合格しても、信頼できるのは「この入力範囲で、この期待値と一致した」ことだけである。評価器、toolchain、fixtureの書き手は、それぞれ別の信頼の前提として残る。

## 7. 検証手順

1. 受理済みWorkspaceを読み、契約が指す定義の存在と一致を確かめる。
2. 独立Expectedを固定する（候補を作る前。以後、候補に合わせて書き換えない）。
3. Generator Sourceを読み、digestを取る。
4. 受理済み定義を入力として実行Generatorを動かし、Target成果物を得る。入力は契約の自由記述からではなく、Workspaceから取る。
5. Target成果物のsource digestと、Generator Sourceのdigestを候補manifestへ記録する。記録したSourceを再評価し、候補のsourceとbytesが一致することを確かめる。
6. 既存の選抜で、Target成果物をbuildし、全caseを独立Expectedと比べる。
7. 誤った生成器（負例）で同じ手順を行い、不合格になることを確かめる。負例が合格するなら、独立Expectedが弱い。

## 8. 最小実証

対象は`sales.order_line.quantity_positive`（kind `positive`）である。結果の実測値は、この変更のPR本文に記す。

- **独立Expected**（手書き）：

| 区分 | 入力 | 期待 |
| --- | --- | --- |
| 正常値 | `"1"`、`"0.01"`、30桁の正の整数 | `satisfied` |
| 境界値 | 小数点以下401桁目だけが1の正の数 | `satisfied` |
| 境界値 | `"0"`、`"-0"`、`"0.00"` | `not-satisfied` |
| 負の値 | `"-1"`、`"-0.01"` | `not-satisfied` |
| NULL相当 | keyの欠落 | `unevaluable` / `missing` |
| NULL相当 | `null` | `unevaluable` / `null` |
| 型不適合 | `"abc"`、`""`、`true`、`[]` | `unevaluable` / `type-mismatch` |

- **誤った生成器**（どれも不合格にならなければならない）：
  - 0を成立とする生成器。
  - 欠落とnullを成立で埋める生成器。
  - float64で判定する生成器。
  - 型不適合を不成立へ畳む生成器。

## 9. 合成とプラグイン化のための拡張点

新設はしない。既存の構造に、次の3点を拡張点として残す。

1. **契約の種類**：`Contract`は`projection`と`validation`を排他の種類として持つ。新しい生成対象は、ここへ種類を足す形で入れる。種類ごとに別のSchemaや別の選抜を作らない。
2. **Generator Sourceの形式**：候補manifestがGenerator Sourceのdigestを持つ。形式（Transform Source、外部source）は由来として記録し、選抜の合否には使わない。
3. **合成の継ぎ目**：生成器を合成するときは、前段の出力を、digestで識別した成果物として後段の入力に渡す。合成した全体は、最終のTarget成果物を独立Expectedで評価する。部品ごとの合格を足し合わせて全体の合格としない。

生成器をPluginとして配る場合は、#951のbuild-tool Capabilityと#3267の検証済み組合せに載せる。本書の範囲では、生成器専用のCapability種別、Registry、Hostを作らない。

## 10. 未解決

- decimalをJSON数値で受け取る場合の扱い。指数表記、分数表記、先頭の`+`も独立Expectedで固定していない。
- Generator Sourceのdigestを載せる欄の整理（`Lineage`と`GenerationProvenance`のどちらへ寄せるか）。
- `reference-generator-context`が出す文脈に、`validation`契約のID・kind・入力keyが入らない。生成AIへ渡す文脈としては不足している。
- `relation-unique`の生成対象（入力が集合になるため、空集合と部分取得の扱いを先に決める必要がある。#4359の調査が示す`EmptyPolicy`の語彙を使う）。
- 方式Bの生成器に対する、実読込byteの束縛と隔離（#2595、#2515）。
- メタ生成の深さの上限を、誰がどの単位で決めるか（ADR-0045の未決を引き継ぐ）。
- ADR-0045の採否。本書はADR-0045が提案中であることを前提にしており、不採用になった場合は本書も見直す。
