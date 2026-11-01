# Java Libraryの外部Consumer・Artifact Store受入計画（2026-10-30）

- **状態**：検討用Draft。正式なJava公開ABI、Store schema、Artifact key、製品Host、ADRを採用する文書ではない。
- **照合main**：`2503a25fee9ffa47592d47818c2bb35c0a2c50ab`（2026-10-30）。`tools/reference-generator-artifact-store/store.go` blob `be9f5bfffa2931fea61e92abe660074cefdc00a4`、`store_test.go` blob `8db7efb94bd8816cc3b69399a6bdad651bc45ef9`、`tools/reference-generator-selection/selection.go` blob `61c8f3b0d25c1a462e346c372c38e72f3184efa5`。
- **責務**：[多言語生成 #2077](https://github.com/hat47x/tei-platform/issues/2077)、[外部製品からのPlugin利用 #2390](https://github.com/hat47x/tei-platform/issues/2390)、[Artifact Set #1102](https://github.com/hat47x/tei-platform/issues/1102)、[Build/試行の分離 #1107](https://github.com/hat47x/tei-platform/issues/1107)、[Target Profile #2019](https://github.com/hat47x/tei-platform/issues/2019)、[Harness #2016](https://github.com/hat47x/tei-platform/issues/2016)に分ける。横断集約は[既存Draft #2321](https://github.com/hat47x/tei-platform/pull/2321)の担当に任せ、このファイルはJava固有の受入境界だけを扱う。

## 目標と非目標

**目標**：TEIで生成・検証された計算機能を、外部の通常Java HostがTEI独自Host、`tei` CLI、Go Runtime、TEI Studio／Registry常駐サービスなしで公開型付きAPIとして呼び出せる。その実行に用いたsource・Build入力・**全class集合**・classpath／JDK・Consumerの実ロード元・結果を、同じ試行のEvidenceで結ぶ。

**非目標**：最初からJAR配布、汎用Java Plugin SDK、共通ABI、WIT/MCP採用、TEI独自Host、完全hermetic build、署名者Authority、実業務DB commit、全Target対応を実装すること。

**現状の確定的区別**：現行Java `java-template`はstdin/stdoutの`Main.java`を生成し、固定fixtureをCLI経由で評価する**参照**であり、外部製品向けの`LineAmounts.calculate`型付き公開Libraryではない。これを「公開Library実装済み」と表記しない。`sales.line-amount/v1`の契約statusは`reference-local`で、複合業務規則のOwner受理は未確認。

## 証拠境界（各行を一つの成功フラグに潰さない）

| 段階 | 根拠・試験での観測 | それだけでは主張しないこと |
| --- | --- | --- |
| S0 契約／Oracle | exact contract、型付き9例、入力変換1例、統合10例、独立期待値 | 業務Ownerによる意味受理 |
| S1 Source closure | 全Java sourceの相対path、実byte/digest、snapshot | 候補manifestの主Sourceだけで全Build入力が閉じている |
| S2 Build Attempt | 各Buildの入力集合、実javac、JDK vendor/version、flags／環境、sourcepath／classpath、独立Attempt | 二回同じdigestが出たことを二回の独立Build実行と自動同一視すること |
| S3 Artifact Set | `Main.class`と`Main$Input.class`等、再帰的全class相対path・size・digest、missing/extra/changedの差 | 主class digest一致を集合一致へ拡大 |
| S4 Store | Put後の実全byte、Getの内部整合、旧版保持、失敗後の状態、再試行 | Put成功／Get成功から実行byteの固定や旧版保全を推測 |
| S5 Runtime / Consumer | 隔離した外部Java Consumer、固定classディレクトリ、実classpath順、ロード元、呼出し先API、実戻り値 | JAR配布、他JDK・他Host・全入力での適合 |
| S6 現在利用資格 | Host側のcurrent authorization、必要Resource/副作用条件とTarget Profile | 過去のbyte一致やfixture PASSによる権限付与 |

同一Sourceと同一出力でもBuild環境が違えばclass byteが変わり得る。Artifact Setのcontent identity、classpathの**相対配置**、絶対Locator、実ロード先は別物。Javaの異なるTarget用class集合を必ず同一byteへ統合する要件は置かない。

## 着手順（それぞれ独立に完了できる）

### P0-A：型付きJava Libraryの独立Consumer一呼出し — #2077 / #2390 C1

1. 最小の公開計算API候補（`BigDecimal`、数量・金額・通貨・単位、欠損／NULLを表す型、返値と計算拒否の区別）を設計する。JSON Adapterは公開計算APIの外側に置く。業務拒否とparse等の技術失敗を混ぜない。
2. まず手書きのReferenceでAPIと受入条件を確かめ、次に同一モデル由来の**生成Java Library**から同じAPIを外部Consumerで呼び出す。両者を同一の実証として記録しない。Consumerは別source／別classディレクトリでコンパイルする。
3. 既存`sales.line-amount/v1`（half-up）で数量1×単価1.005 JPY/piece→`Present(1.01 JPY)`を一呼出し。型付き9例、字句拒否1例、CLI統合10例を別列で評価。`1e2`や重複JSONキーによりtyped inputが違うときはAdapter差であり計算非同値としない。
4. Consumerのsource／class byte、公開API参照、JVMがロードしたclassの取得元、実結果を対応付ける。classpath shadowを一件含め、**同じLibrary bytesでも異なる同名classを解決し得る**ことを検証する。

**完了**：TEI依存なしの別Consumerが**生成Java Library**の公開APIを実際に呼んだEvidenceがあり、要求したJDK/Classpath、Source・全class集合・Consumer結果の対象が一貫している。手書きサンプルだけなら「手書き参考PoC」に限定する。JARは不要。

### P0-B：既存CLIのStore復元閉包 — JR-01 / #2077

現行`Store.buildArtifact`はJavaをコンパイルして`Main.class`だけを保存し、通常の`Main$Input.class`を保存しない。次を同一Source/JDKで対照する。

- JR-01A：コンパイル直後の全class directoryから`Main`を隔離起動。
- JR-01B：既存Put/Getで取得した`artifact`だけを空classpathへ配置して同じ入力を実行。
- JR-01C：同一Buildの全class集合を相対pathどおり配置して同じ入力を実行。

**完了**：Getの保存整合性と、取り出した集合の実行可能性を別々に観測。外部依存や元ビルドdirectoryからのclass混入を拒否し、失敗した場合は実例外とExitを記録。JARではなくclass directoryでもよい。

### 2026-11-01 JR-01用の実Store Put/Get試験票（提案・**未実行**）

**新しい固定版**：TEI main `43517af5450aacb2fdd6c48585dc143f52d57da4`。現在の`store.go` blob `b57570dae7d20521b60d945893c288aed11ec1c9`は、以前の`be9f5bff...`から**Go候補の一時build配置**が変化した。一方、Javaの`javac -d <temp> Main.java`→**`Main.class`のみを`artifact`へ保存**する実装（348〜369行）は不変。既存`TestPutAndGetJavaArtifact`は実Put/Get digestが中心で、Store復元物の隔離Java実行を確認していない。

**必要入力を固定すること（TEI正本から取得）**：
- Git revision、`contract.json`の実byte/契約ID・版・digest（`sales.line-amount/1`・`reference-local`・half-up／scale=2／precision=8）、`fixtures.json`実byte/10件のIDと期待値、`GenerateVersion("java-template","1")`後の`candidate.json`・`Main.java`実byte。
- `Select`実行時に得た当該Candidateの`SourceDigest/BinaryDigest/BuildEvidence`と選抜Receiptの**生byte**。**Store.Putの真正な入力前提**として固定する。前回の手作業テンプレート展開からこのReceiptを逆算して作った扱いにしない。
- `java`/`javac`の実行パスと`-version`、JDK vendor、`JDK_JAVAC_OPTIONS`等の影響環境、`javac -d <classes> Main.java`のcwd、完全な`-cp <isolated-class-dir> Main`引数。元候補dir・CLASSPATH・以前のtest実行dirからのclass混入を避ける。JDK/Classpathは**Consumer実使用条件**と照合する。
- `Put`後と`Get`後の`Passed/Reused/Reason/error`、`ObjectPath`、保存objectの`source,artifact,receipt.json,object.json`各実byte/digest・相対path、`Get`時点の`BinaryDigest`。4ファイル一致の判定を実Javaロードの成立と混ぜない。

| 段階 | 実行／観測 | 受入／未証明 |
| --- | --- | --- |
| JR-01A（対照） | **同じ候補Main.java**を独立`javac -d`でcompileし、再帰的全class相対path/size/digest（少なくとも`Main.class`と`Main$Input.class`）を固定。**このclass集合**から隔離`java -cp`で公開10入力を再実行。 | 10/10と各classの存在・Load条件を観測。ただしこの集合はStoreから復元したものではない。 |
| JR-01B（本題） | 実`Put→Get`で得た**Store内`artifact`の実byte**を新しい空directoryの`Main.class`へ配置。追加memberがないことを記録して同一入力で起動。 | 現行コードでは`NoClassDefFoundError: Main$Input`を予測。実Exit/stderrが必要。Put/Get成功のみを「復元可能」と認定しない。 |
| JR-01C（対照） | JR-01Aと**同一compileの全class集合**を別の空directoryへコピーし、同じ公開10入力を実行。 | 10/10を期待するが、補完した`Main$Input.class`は**Store外**のためStore復元成功とは記録しない。 |
| Source/Receipt境界 | `Generated.SourceDigest`、`Selected.CandidateReceipt.SourceDigest/BinaryDigest`、`Put/Get`および実読取byteを同一試行で列挙。 | これは試験fixtureの選抜Receiptを**Put入力へ結合**した観測。新しい独立Selector資格や当時のモデル生成因果の認定、Owner業務意味受理は別。 |

**再利用できる既存実測**：[#2077の2026-10-31報告](https://github.com/hat47x/tei-platform/issues/2077#issuecomment-6538058027)で、手作業展開した現行テンプレートの6,851-byte Source、全classの`Main.class`（5,346-byte）・`Main$Input.class`（604-byte）、公開10/10、主class単独の`NoClassDefFoundError`は**OpenJDK21.0.12.1の隔離試験でRUN**。新試行はこれを「妥当な期待と比較元」として利用するが、別のJDK・Build/Locator・Store世代で自動的に同byte/同結果とは数えない。特に生fixture byte／Source byte／class byte／classpath実ロードを混同しない。

**試験コードの準備状態**：既存`selectionPaths`・`GenerateVersion`・`Select`・`Put`・`Get`を再利用する`TestJavaStoreRestorationWitness`（試験専用Goファイル）は、作業用環境で`gofmt`処理した**未実行の提案物**として準備済み。repoへのテストfile追加、CI、`go test`実行は未了。実行環境Go 1.23.2はmainの`go 1.26.10`に不足し、外部ネットワークも不可。**SKIPをPASSに数えない**。既存試験に追加する場合、過去の欠落を恒久仕様へ固定する「失敗期待」は、全class保存の改修時に置換する。

**完了条件**：実`Store.Put/Get`の証拠、Store全ファイルbyte、独立compileの全class集合、`-cp`と実Exit/結果、公開10fixtureの**同一Attemptの接合**が確認されること。技術検証が成立してもJR-02〜06、外部型付きC1 Library、配備/Authority、Owner意味受理は別。新しいStore schema・artifact key・共通Hostの採用は先行しない。

### P0-C：同Source別Build・復旧 — JR-02（既存票との差分限定）

[二回目Putの4ファイル比較](https://github.com/hat47x/tei-platform/issues/2077#issuecomment-6502165611)および[再試行と応答喪失](https://github.com/hat47x/tei-platform/issues/2077#issuecomment-6516087790)を試験対象にする。合法な別Build A/Bを準備できた場合だけJR-02を実行する。

- R1：同artifact／整合した別provenance。Put拒否後も**全4ファイルbyte不変**・旧A取得を独立判定。
- R2：別artifact／整合Receipt。同じSource-keyへ書込みが進むか、旧AがStore内に残るかを観測。
- R3：`source→artifact→receipt.json→object.json`の途中停止。Put失敗、部分byte、Get、旧A保全を分離。
- R4：途中更新後に**同じBを再Put**したときに拒否し得る経路、全書込み後に**成功応答だけを失う**経路を別々に反証する。

**完了**：各隔離Storeの二回目Put前後の全相対path・実byte、Putの`error/Passed/Reused/Reason`または応答なし、独立Get、旧Aが元Storeから取得可能か、同B再試行結果を記録。Store外へ退避した試験用旧byteを「Storeが旧版保持」と数えない。故障注入はsleepに頼らない試験専用同期点を使用し、製品Hook/APIではない。

### P0-D：再利用／検査後実読取の不一致 — JR-05 / JR-06

[個別の受入票](https://github.com/hat47x/tei-platform/issues/2077#issuecomment-6516221760)に従う。

- **JR-05**：保存objectの`source`だけを意図的に壊し、同Candidate/同ReceiptのPutが`Reused=true`を返し得るのに独立Getは拒否する条件。これは既存の「receipt欠落補完」のテストとは別。
- **JR-06**：Getが実artifact Xを検証した後に、同Sourceの正当な別Build BのPutでpath上のbyteがYへ切り替わり、後続Consumerが実際に読み込むのがYとなる条件。**最初はbyte読取まで**を検査し、Java全classのRuntime実ロード証明を先取りしない。

**完了**：Put／Reused、Get、実Consumer読取byte、旧A保持を混同せず観測。新しいStore schemaやキー変更を最初の試験の前提にしない。

## P1：保存方式の比較（P0の結果が先行）

次の三候補を同一の故障マトリクスで比較する。**未採用**。

| 候補 | 長所の仮説 | 別途検証が必要 |
| --- | --- | --- |
| 別Buildは既存Source-keyへ上書きせず拒否 | 旧byte保全の最小手当て | 複数成果物共存とStore内部不整合の回復 |
| 全class集合を不変objectとして保存、外部参照で選択 | 複数Toolchain／Target版の共存と読取pin | 完全なartifact closure、GC/prune、読み手の世代pin、原子的公開 |
| 隔離staging→内容検査→公開切替 | 部分更新を公開しない設計 | 同一FS rename、Windows/UNIX差、fsync／電源断、同時writer/reader、旧版保持 |

ここで使う`immutable`は、単にdigest記録済みという意味ではない。実読込中にbyteが変わらない保存・開放の運用条件を別に確認する。署名・Attestation・Host Authorityは別。

**P1の停止条件**：現行JR-02の不具合が再現したという理由だけで、共通Build contractや全TargetのKey/Store schemaを一斉に変更しない。別の有効なBuild成果物と、破損／途中書込み／失効を区別してから方式を選ぶ。

## P2：#2390の相互運用性・SHO説明へ接続

- #2390のC0（契約のみ）とC1（通常Libraryの外部組込み）を分ける。Java C1の証拠は#2077の同一Consumer試行から**参照**するが、#2390のread-only quote-lines 23例（正常8＋業務拒否15）を`sales.line-amount/v1`の10例へ自動読み替えない。別Contract・別Oracle・別実装の場合は別のConformanceが必要。
- 既存Hostが提供するloader/permission/revocationに委譲し、TEI専用Host・共通Registry・第二のOracle・全言語共通ABIを前提としない。
- Java ServiceLoader、明示的な型付きAPI、外部process/Wasm等の方式選択は、実Consumerと追加依存の測定後に判断する。
- SHOの説明では「生成Source取得」「全class集合再現」「Store復元」「実Consumer呼出し」「業務Owner受理」「現在のHost利用資格」を個別の**根拠付き主張**として表示。保存した本文／図／Evidenceの差と、当時のReviewの有効性を混ぜない。
- UI操作・テスト制御・AI Computer Use用の外部制御プラグインは、生成Libraryに無条件同梱しない。必要なTarget／デバッグ構成でのみ含め、除去を選ぶ場合は配布artifactと依存・exportされた機能を検査する。スイッチOFFだけを物理的除去済みとは扱わない（実装Ownerは関連UI/Plugin課題）。

## 実測／予測／Owner判断の台帳

| 区分 | 現状 |
| --- | --- |
| **既存コードの静的観測** | SelectorのJava二回Buildは`Main.class`のみ比較。StoreはJavaの`Main.class`一個を保存。再利用判定は保存`source`を再読取せず、Getは読み、Getの`ObjectPath`は不変Snapshotではない。 |
| **既存試験** | CLIのJava fixture評価・Java単体artifact Put/Getのテストが存在。ただし全class保存復元／外部公開Library Consumer／JR-02〜06の実測とは別。 |
| **この議論の手書きJava参考小実験（RUN、TEI外）** | Linux OpenJDK 21.0.12.1で`1 × 1.005`→`1.01`の別Consumer一呼出し。classpath先行の同名classによって`9.99`となり、class-load logで別class dirを確認。`-g`対`-g:none`で主class・内部classのbyteも変化。生成TEI Libraryの適合性は非主張。 |
| **未実行** | TEI版JR-01〜06、生成Java型付きLibrary、Store上書き／部分失敗／復旧、TEI非依存の正式Consumer／外部Host実行、JAR配布、全TargetはNOT RUN。 |
| **Owner未決** | 外部Libraryの正式公開契約、対応JDK／Target範囲、旧Buildの保持／回復要件、Store schema・Key方式、JSON字句許容、複合計算規則のBusiness意味受理、現在Host Authority。 |

## 2026-10-31の実測追補：現行Javaテンプレートのclass閉包（JR-01の前段、**StoreはNOT RUN**）

対象main `1e8d3031aee7881bf6907f7774c9dba7a438e11e` を再照合した。テンプレートGit blob `32a7c6877ab34ca26009f4436c2fe7ce487cb7ac`、契約Git blob `e72ae7fbcbff036a96bd45f78585d34e2c11f3d5`、公開fixture Git blob `4c8948b9e6f5b0a82609e8abe58cb073609a7ec1`。本試行は**TEI Generator CLI/Selector/Storeを呼ばず**、現行`Main.java.tmpl`へ`generator.go`に記載された固定値（`line-amount-java-template`、`sales.line-amount/1`、`java-template`、`JPY`、`scale=2`、`integerDigits=6`）を代入した隔離実行である。

- **Source同一性の実測**：GitHubから取得したテンプレートへ独立に同じ値を代入したSourceのSHA-256=`0c724dd5c40f46853697aa71b0152296dd1b6a1197aabe5077370d96bce7ddf5`は、ローカル`Main.java`のSHA-256と一致（6,851 byte）。従って手書きJava参考実験ではなく、**公開された現行テンプレートの展開byte**そのものを検査した。ただし生成CLIのReceiptとSourceを結ぶ試行は未実施。
- **Fixture同一性の実測**：公開10件の`cases/id/input/want`を再構成し、JSON object key順を正規化したSHA-256=`0f8d1302bb4cd020fb56f3c5145737dc8e9f14c74beae060b8b7011b9db2c5da`がGitHubの公開fixtureと一致。**JSON値内容の同一性**であり、元fixtureの生JSONファイルbyte一致ではない。
- **実Java実行**：OpenJDK 21.0.12.1。コンパイルで`Main.class`（5,346 byte、SHA-256=`461c17f20bd389c17dddf020fabbee07bef0de11bdd0b4904cf4857ef3fd861a`）と`Main$Input.class`（604 byte、SHA-256=`bb92ff231a6cc8574cb636c25f9918868f9126fa13debcb608d67836c261737b`）の**二member**を観測。全classの隔離directoryで公開fixture **10/10** 合格。入力`1×1.005 JPY/piece`も`1.01 JPY`。
- **最小反証の実測**：同じBuildから`Main.class`**だけ**を空directoryへ置き、同じ入力を`java -cp <isolated> Main`で起動すると**exit 1**、`java.lang.NoClassDefFoundError: Main$Input`。これにより現行StoreのJava分岐が採用する「主classのみを保存する形」の**実行閉包不足を隔離再現**できた。**現行Store.Put/Getを経由したJR-01Bの実測ではない**。
- **Adapter追加観測（契約外・未採用）**：`quantity:null`の後に重複キー`quantity:"1"`を与えると`missing`、引用符なしJSON数値`quantity:1`は`present 1.00`、正常object後の余分な文字列` trailing`があっても`present 1.00`となった。Java独自parserの挙動でありJSON字句の業務受理やGoとの計算非同値としない。特に末尾ゴミを受理する問題は**Adapterの構文境界**として別に検証する。Go参照テンプレートも`json.NewDecoder(os.Stdin).Decode(&in)`を**一度だけ**呼び、後続の余分なbyteまで消費済みか再確認していない。ローカルGo標準Decoderの同条件試験でも、末尾` trailing`は拒否されず、`quantity=1`と解釈された。従って**末尾ゴミはJavaだけの差ではなく、両参照Adapterでの入力終端未検証**として扱う。重複キー`null→"1"`ではJavaは`missing`、Go Decoderは最終値`1`、数値トークン`quantity:1`ではJavaは計算へ進み、Go Decoderは型不一致エラーを返すことをこの限定ローカル比較で確認した。Goの実生成CLIや正式なJSON字句契約の受入を確認したものではない。

**Gate更新**：P0-Bは「現行公開テンプレートの全class vs 主class単独の隔離実行」を実測したが、**Store経由のJR-01A/B/Cは未完了**。次はStoreの現実装で`Put→Get→新規clean directoryへ実byte復元→同一JDK/入力で起動`を一試行に束縛する。P0-A（**TEI生成の型付き公開Java Libraryを独立Consumerから呼ぶ**）、JR-02〜06、Runtime資格、Owner意味受理は依然未実行。正式Store schema、鍵形式、ABI、JAR、共通Hostは未採用。

**#2390横断の補足**：可搬性はPluginの属性単体でなく、`Consumer要求 × Provider提供 × Host保証 × Context/時点`の関係で評価する（[#2390の10/31追補](https://github.com/hat47x/tei-platform/issues/2390#issuecomment-6536745970)）。本書のJava C1は**callableな純粋計算**に限定され、`std.message`の非実行定義・Smart Data HookのHost phase・業務データ取得の集合完全性の適合を代用しない。#1861のNode/Chromium/Wasmtimeでの実WASM検証は既存証拠として参照し、同じ外部Consumer実験を重複実装しない。

試験材料は対話内の隔離ZIPに保全し、repoへ実行バイナリを追加していない。これは製品JDK互換性の保証でも、Owner受理済み契約への格上げでもない。

## 引渡しと責任

- **#2077／Generator・Java担当**：P0-A/B、Java Source→Build→class集合とConsumerのWitness。新規公開APIの最小形は比較後に決める。
- **#2016／Artifact Store担当**：P0-C/D、Put/Getの状態と修復・保存の技術境界。試験が失敗したとき「拒否したから不変」を推論しない。
- **#1102／Build担当**：多出力のmember identityとBuild Attemptを既存Referenceへ対応づける。Java専用layoutをdomain-neutral Setへ黙って混ぜない。
- **#2390／外部Plugin利用担当**：P0-Aから実Consumer・依存閉包の証拠を取り込み、C0/C1の区別と移植費を評価。E0/E1/E2の第二正本や重複実験は作らない。
- **#2019／Host・Target担当**：現在利用資格、解決対象版、配備とrollbackを別証拠で評価。SQLiteで実測した性質をfilesystem Java Storeへ自動転用しない。
- **SHO／構成議論 #2321**：上記Evidence範囲を人間向け説明・計画へリンク。文書が統合されたことを実装完了へ昇格しない。

**完成判定（この設計書）**：各P0に検証入力・実観測対象・受理条件・Ownerが指定され、P1/P2で採用待ちを明示できていること。**P0がすべて完了したことではない**。本書は検討成果の保存に限る。
