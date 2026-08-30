# ADR-0004: Source-to-Canonical TraceabilityをCanonical Worldから分離する

- 状態: 採用
- 日付: 2026-08-30
- 関連Issue: #6, #14

## 背景

M0.22.1までのTEIでは、Canonical IDをfile pathから独立させ、Semantic DefinitionからApplication / Database / Release / Deployment / Plugin / Runtimeまでを一つの意味の流れとして追跡できるところまで進みました。

一方、Authoring sourceからCanonicalな意味へ入る境界は、実行側ほど詳しく扱っていません。

現行Lexerはoffset / line / columnを保持していますが、Workspace loaderがCanonical entityへ残す`SourceLocation`は、多くの場合「どのfileから来たか」を示すだけで、entityやmemberの正確な範囲を保持していません。また、`.sql`や`.go`等のnative fileはmanaged fileとして列挙できますが、「どのCanonicalな意味やRealizationへ、どの役割で寄与したsourceなのか」は十分に表現していません。

Validationの`Diagnostic`も現在はCode / Severity / Messageが中心で、semantic findingからsourceへ機械的に戻る共通契約がありません。

この状態のままEditorやAI authoringを先に作ると、file pathをsemantic identityへ混ぜたり、各UIやanalyzerが独自のsource mapを持ったりする可能性があります。M0.23では、GUIを作る前にsourceとCanonical meaningの境界を固定します。

## 判断

Source-to-Canonicalの追跡情報は、**Canonical Worldの一部ではなく、Authoring / Workspace側が所有するderivedなSource Indexとして扱います。**

Canonical domain modelへfile path、line、column等を埋め込みません。Canonical IDも従来どおりpathから独立させます。

概念上、Source Indexは次の情報を結びます。

```text
Source Artifact
    ↓ contains
Source Range
    ↓ contributes as a role
Semantic Subject
    ↓ normalized into
Canonical World
```

ここでSemantic Subjectは、Application、Database、Query、Semantic Type等のpath-independentな対象を指します。Source Artifactは、その意味を記述・実装しているWorkspace内の物理sourceです。

同じCanonical entityに複数sourceが関係することは許します。たとえば、定義を持つ`.tei`とcustom implementationの`.go`は別の役割で同じ意味へ関係し得ます。ただし、**同じsubjectを同じauthoritative roleで複数sourceが競合して定義する場合、明示的なcomposition ruleがない限り暗黙にmergeしません。**

## Source Artifact

M0.23では、Workspace内のmanaged fileをSource Artifactの対象とします。

Source ArtifactのlocatorはWorkspace-relative pathを基本とします。ただし、このpathはsemantic identityではありません。file move / renameではSource Indexのlocatorだけが変わり、Canonical IDやnormalized Worldは変わらないことを不変条件とします。

Source Artifactには、source contentのfingerprintを持たせます。

Source Rangeは特定のfile内容に対する位置なので、pathだけでは「そのrangeが現在の内容にも有効か」を判断できません。fingerprintを合わせて保持することで、古いSource Indexを新しいfile内容へ誤適用しないようにします。

fingerprintはsemantic identityではなく、Source Indexの鮮度とprovenanceを確認するための情報です。

## Source Range

Source Rangeの基準位置にはbyte offsetを使います。

範囲は原則として半開区間`[start, end)`で表します。line / columnは人間向けの表示やdiagnosticのために併記・導出できますが、rangeの基準にはしません。

byte offsetを基準にする理由は、source bytesと一意に対応し、UTF-8、Editor protocol、表示上のcolumn定義の差をsemantic modelへ持ち込まずに済むためです。将来LSP等がUTF-16 positionを要求する場合は、Source Artifactの内容から境界adapterで変換します。

既存Lexerが持つPositionは、このSource Rangeを構成する入力として利用できます。ただしLexer / Parserの内部表現をそのままpublic contractとして固定することはしません。

## Parserとの境界

Parserが返すdomain valueへSource Rangeを埋め込みません。

Parser / loaderは概念上、次の二つを分けて返します。

1. path-independentなdomain value
2. そのvalueやmemberがどのsource rangeから構成されたかを示すsource anchor

実装上、一つのparse result structで返すことは構いませんが、domain packageの型そのものがphysical source locationを所有しないことを条件とします。

これにより、同じ意味を別のauthoring front-endから生成してもCanonical modelを共有できます。またfile moveだけで`reference/world.json`等のCanonical goldenが変化することを避けられます。

## Validation / Diagnostic

semantic validationはfile pathを直接解釈しません。

Validation findingは、まずCode / Severity / Messageに加えて、可能な場合はpath-independentなSemantic Subjectを指します。その後、Workspace / Authoring側がSource Indexを使い、subjectから一つ以上のSource Referenceへ投影します。

```text
semantic validation
    ↓
Diagnostic + Semantic Subject
    ↓ Source Index
Source Reference / Range
    ↓
CLI / Editor / AI navigation
```

一方、parse errorのようにCanonical subjectがまだ成立していない問題は、sourceそのものに対するdiagnosticです。この場合はSemantic Subjectを無理に作らず、Source Referenceを直接持てます。

この二種類を区別することで、Validation layerをphysical pathへ依存させず、それでも利用者を正確なsource位置へ戻せるようにします。

## native source

M0.23ではnative source全般を一気に一般化せず、SQLを一つのvertical sliceとして扱います。

SQLがauthoring source of truthであるQueryでは、SQL本文を`.tei`へ複製することを必須にしません。Workspace内のnative `.sql` fileをauthoritative sourceとして参照し、loader / analyzerがその内容をQuery modelとDerived Query IRへ接続できる形を検証します。

現行`query.EmbeddedSQL`はSQL本文を保持しますがphysical pathを持たないため、この性質を維持します。外部`.sql`を読み込む場合も、Canonical / normalized側には必要なSQL内容と意味を渡し、physical pathやrangeはSource Indexへ残します。

したがって、同じSQL内容を別pathへmoveしてもCanonical Query IDとnormalized meaningは変わらず、Source Indexだけが更新される形を目標とします。

Derived Query IRやanalysis resultはauthoritative sourceではありません。sourceへ追跡できることは求めますが、derived artifactからnative SQLへ暗黙に逆書きする経路は設けません。

## 採用しなかった案

### domain modelへSourceLocationを埋め込む

実装は単純ですが、Application、Query、Semantic Type等がAuthoring形式やfile layoutを知ることになります。file moveだけでCanonical artifactが変わりやすくなり、別authoring front-endとの共有も難しくなるため採用しません。

### file pathをCanonical identityの一部にする

sourceと意味の対応は分かりやすくなりますが、rename / moveが意味変更になってしまいます。既存のCanonical ID原則にも反するため採用しません。

### Editorが独自にsource mapを持つ

Core実装を変えずに進められますが、CLI、validation、AI、将来のEditorが別々の追跡規則を持つことになります。Source-to-Canonical traceabilityはEditor機能ではなく共通基盤なので採用しません。

### すべてのnative sourceをM0.23で一般化する

Go AST、各SQL dialect、template、UI source等まで含めると、Source Indexの設計検証より個別language integrationが主作業になります。M0.23ではnative SQL一例に絞り、モデルが成立することを先に確認します。

## 不変条件

M0.23の実装では、少なくとも次をConformanceとして確認します。

- source fileのrename / moveだけではCanonical IDとnormalized Worldが変わらない
- source contentが変わればfingerprintが変わり、古いrangeをそのまま有効扱いしない
- entity / memberのSource Rangeをfile先頭ではなく実際の位置へ戻せる
- semantic diagnosticはSemantic Subjectを経由してsourceへ解決できる
- parse diagnosticはCanonical subjectがなくてもsource rangeを示せる
- native SQLは`.sql`のままauthoritative sourceにできる
- Derived Query IR等からauthoritative sourceへ暗黙に逆書きしない
- 同じsemantic subject / authoritative roleの競合を暗黙にmergeしない

## 影響

Source Indexという新しいderived boundaryが増えるため、Parser / Workspace loader / Validation / CLIの間に明示的な受け渡しが必要になります。単に`Sources map[string]SourceLocation`へ項目を増やすより実装量は増えます。

一方で、この境界を先に置くことで、将来のWails Editor、AI authoring、code navigation、diagnostic表示が同じsource provenanceを利用できます。EditorをCanonicalな意味の唯一の所有者にせず、native sourceもsource of truthのまま扱えます。

また、Canonical WorldとSource Indexを分けることで、意味の変更とfile layoutの変更を別々にreviewできます。これはTEIの「意味と実現を必要以上に一体化しない」という原則をAuthoring側にも適用する判断です。

## M0.23の範囲

このADRはSource Indexの具体的なGo型名や`.tei` syntaxを永久に固定するものではありません。M0.23では、まず次の縦切りが成立する最小構成を実装します。

1. Parserが正確なsource anchorを保持する
2. Workspaceがpath-independentなsubjectとSource Artifact / Rangeをsidecar indexで結ぶ
3. semantic diagnosticをsubjectからsourceへ投影できる
4. native `.sql`を一例としてauthoritative sourceのままQueryへ接続する
5. rename / move invarianceとderived artifactの片方向性をテストする

GUI Editor、Go source semantic analysis、汎用Language Server、全native format対応はM0.23には含めません。
