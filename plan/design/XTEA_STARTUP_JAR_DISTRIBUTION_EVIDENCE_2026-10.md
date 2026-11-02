# X-TEA startup JAR 配布・版・更新の証拠境界（2026-10-26）

- 状態: **外部Referenceの静的証拠整理**。X-TEAの実行検証・TEIの更新機構への受入証明ではない。
- Source of truth: [#943](https://github.com/hat47x/tei-platform/issues/943)（Source/Build/Artifact provenance）、[#1223](https://github.com/hat47x/tei-platform/issues/1223) / [ADR-0032](../adr/0032-composition-kernel-replacement.md)（Generation/更新/rollback）、[#2019](https://github.com/hat47x/tei-platform/issues/2019)（Target配備・更新のReference）、[#2020](https://github.com/hat47x/tei-platform/issues/2020) / [ADR-0036](../adr/0036-delivery-forms-as-plugin-composition.md)（配布形態）。
- この資料は既存Issueの受入条件を代替せず、X-TEA特有のJAR更新方法をTEI標準へ移植しない。

## 1. 固定した証拠

| 層 | 識別子・位置 | 状態 |
| --- | --- | --- |
| 公開M49 Source | [xead/XeadModeler commit `68f1535b1c5dd80a327dfced37be401ecd3093c9`](https://github.com/xead/XeadModeler/commit/68f1535b1c5dd80a327dfced37be401ecd3093c9) | GitHub固定版を直接静的確認 |
| M49 source `jar/MANIFEST.MF` | Git blob `3c4183c3b0cead82069cf4b00ac631685f99d1bf`, 1–8行 | `Main-Class: xeadModeler.Application` と相対 `Class-Path`。製品version/build/source digestなし |
| M49 `Application.java` | Git blob `093171faf36b2c5842aa22f94a234b713c635b67`, 49–69 / 88–96行 | `main` → `new Application(args)` → `new Modeler(args,this)` |
| M49 `DialogAbout.java` | Git blob `23efd3f221eb8f160132a45f2c44785b71d363b7`, 50行 | `FULL_VERSION = V1.R4.M49` |
| 設定探索の変更履歴 | [commit `73a09364d99c6ef9f1ace8db18e4861caed6445e`](https://github.com/xead/XeadModeler/commit/73a09364d99c6ef9f1ace8db18e4861caed6445e), `Modeler.java` `jbInitComponentsAndVariants()` 差分 | `xteamdl.properties` を作業ディレクトリ優先、次に `java.class.path`文字列から算出したパス |
| M49最終 `Modeler.java` | [固定commitの公開Source](https://github.com/xead/XeadModeler/blob/68f1535b1c5dd80a327dfced37be401ecd3093c9/src/xeadModeler/Modeler.java)、Git blob `c104e379862cf280a5944ed97c35cfe568c5f348`、SHA-256 `7cbbefe8115b20e2ba13fcfdb21ef9c5ae596debf673b6afaf21ab6b20a3edef` | 保存archiveのsnapshotと公開commitのblob一致を直接確認。181行で`properties` fieldを初期化し、1614行で初期化メソッドを呼び、1653–1705行で設定読込を試行・適用する。作業ディレクトリ直下を先に探し、無ければ`java.class.path`文字列から最後の`File.separator`までを切り出して組み立てたパスを探索する。これが実際のJAR配置先と一致するかは起動方式に依存する。既存Modeler instance用の明示reloadは同固定Sourceに見つからない。M50 JAR動作とGUI反映は未確認（[Issue #940記録](https://github.com/hat47x/tei-platform/issues/940#issuecomment-6467903431)） |
| 保存M50 Installer ZIP | `packages/XTeaModelerInstaller1_4_50.zip`, SHA-256 `d5875dac35c0fdfc19c761a78957707d218ef1baa75b854e002b70ad419196e8` | [保存物調査の記録](https://github.com/hat47x/tei-platform/issues/1223#issuecomment-6262536696)。2026-10-29に保存ZIPのraw SHA-256を再計算して既存値と一致確認。外部からの再取得は行っていない |
| ZIP内Modeler JAR | `Files/xteamdl.jar`, SHA-256 `0305791a8f5a61291d436f7f04dc8ff1180fa79841bda88b01524d636fb9f04f` | [以前のbytecode静的解析記録](https://github.com/hat47x/tei-platform/issues/940#issuecomment-6266988686)。M50の版表示は[既存#943静的照合](https://github.com/hat47x/tei-platform/issues/943#issuecomment-6234874050)由来 |
| M50 installer設定 | `INSTALL.DAT`, SHA-256 `d21eb61234d23ebf3a46049240853df3a13624087cb4b88c257848f9e7e95e92` | [既存#1223静的照合](https://github.com/hat47x/tei-platform/issues/1223#issuecomment-6262536696)。`%ProgramFilesDir%\\XTEA\\Modeler`、`.xead`関連付け、`xteamdl.bat`、shortcut宣言 |
| M50 installer内launcher | `Files/xteamdl.bat`, ZIP entry SHA-256 `8882d1e7d60aa3a0fc33f21e8b0e4573a3d7bdbe4989cb2b43386371e62f2080` | ZIP SHA-256 `d5875dac35c0fdfc19c761a78957707d218ef1baa75b854e002b70ad419196e8` のentryを静的確認。scriptは固定basename `xteamdl.jar` を指定する。原script本文は転載しない。OS上の関連付け結果・BAT実行・installed fileの同一性は未確認 |
| M50 README | SHA-256 `7c4c0fa90b6996145c0f652f0ca590f597c1981951f37f0022c5d2832f8c6829` | [既存#1223静的照合](https://github.com/hat47x/tei-platform/issues/1223#issuecomment-6262536696)。Java 1.7–1.8別途導入/確認、uninstall時のユーザー作成file保持の記述 |

## 1a. Manifest版フィールドとarchive checksumの確認（2026-10-26追補）

保存アーカイブは非公開Referenceであり、ここでは既存GitHub Issueに公表済みの識別情報と、追加で確認したchecksum台帳の行番号のみを扱う。archive Git HEADは `e5bbfeb4610dd40420afc42c0d6a7188768fef98`。

| archive `SHA256SUMS` の位置 | 対象 | 掲載されているSHA-256 | 証拠の限界 |
| --- | --- | --- | --- |
| 3行 | `packages/XTeaModelerInstaller1_4_50.zip` | `d5875dac35c0fdfc19c761a78957707d218ef1baa75b854e002b70ad419196e8` | 2026-10-29に保存ZIP bytesのSHA-256を独立再計算し、台帳と一致確認。ZIP内全entryや配布マニュアル全体の再検査を意味しない |
| 6行 | `source-history/XeadModeler.bundle` | `ddee2f44ae6d195c7e15d91ab7ead95caeaac1cd0cf322a6cbf80618dbb57c31` | 履歴の保存物であり、M50 JARのビルド来歴ではない |
| 298行 | `source/XeadModeler/jar/MANIFEST.MF` | `c6b4ee820d5b425687e374616b468bb6b8380a8aabf6250d1d6e3e2e9970bba0` | M49 source側。M50 JAR内Manifestとは同一視しない |
| 299行 | `source/XeadModeler/src/xeadModeler/Application.java` | `d89dbc9efa76a285e516e1ce1bad83ae27d532f5b159a8138da7fa6aaafe1e20` | M49起動入口 |
| 300行 | `source/XeadModeler/src/xeadModeler/DialogAbout.java` | `04fdcde99c8531e62eb1a04551647d85ce793a19827d84dcccc6cedc70d7609a` | M49 `FULL_VERSION` 定数 |

保存sourceのManifestは上記M49コミットとGit blob `3c4183c3b0cead82069cf4b00ac631685f99d1bf` が一致する。内容1–8行には `Main-Class: xeadModeler.Application` と相対 `Class-Path` を記述する一方、**`Implementation-Version` 行は存在しない**（`Manifest-Version: 1.0` はJAR Manifest形式の版であり製品版ではない）。一方、**M50 ZIP内 `Files/xteamdl.jar` の `META-INF/MANIFEST.MF` は今回直接検査できない**ため、M50の `Main-Class` や `Implementation-Version` 欠落は未確認。M50 `V1.R4.M50` は[既存のclass定数静的検査記録](https://github.com/hat47x/tei-platform/issues/943#issuecomment-6234874050)であり、Manifestがその版を宣言したという意味ではない。

[`README.md`の保存方法についての既存公表記録](https://github.com/hat47x/tei-platform/issues/943#issuecomment-6234874050)によると、配布マニュアルは元installer ZIP内に保持され、Source working tree・Git bundle・配布ZIPは別個の保存対象である。2026-10-29にはZIP全体のraw SHA-256と`Files/xteamdl.bat` entryを確認したが、配布マニュアル全体や全entryを独立再検査したわけではない。過去のREADME/INSTALL.DAT静的調査結果の再利用と、今回の確認範囲を区別する。

**製品版とデータ形式版の追加照合（既存証拠の再索引）:** M49 source `src/xeadModeler/DialogAbout.java:47,50` は `FORMAT_VERSION="1.2"` / `FULL_VERSION="V1.R4.M49"`。M50 installer内 `Files/xteamdl.jar` の `xeadModeler.DialogAbout` class定数は、[既存#940記録](https://github.com/hat47x/tei-platform/issues/940#issuecomment-6205292851)および[独立照合記録](https://github.com/hat47x/tei-platform/issues/940#issuecomment-6205334330)で `FORMAT_VERSION="1.2"` / `FULL_VERSION="V1.R4.M50"` と報告されている。**表示製品版は異なるが、形式版の文字列は一致**。これはデータ形式の読み書き互換、M50のSource provenance、M50 Jar内Manifestの `Implementation-Version` 欄の有無を証明しない。今回M50 class bytesを再解析していないため、M50値は「以前の固定artifact静的検査記録」として区分する。

**保存物とソースの非同一性:** 保存Source snapshotのM49とInstaller JARのM50は版表示が異なる。[#943既存記録](https://github.com/hat47x/tei-platform/issues/943#issuecomment-6234874050)にあるとおり、Source snapshot（working-treeを含む）とInstaller ZIPが同一Build由来である証拠はない。版違いを誤配布や互換障害と断定しない。

## 2. 起動時の層を混同しない

```text
M50 installer declaration:
  INSTALL.DAT .xead association / shortcut -> xteamdl.bat
    -> xteamdl.jar (fixed basename declared by package BAT)
    (OS registration, BAT execution, installed file identity NOT VERIFIED)

M49 source-only launch path:
  MANIFEST Main-Class: xeadModeler.Application (M49 source confirmed)
  -> Application.main -> Modeler(args, this) (M49 source confirmed)
  -> xteamdl.properties: working directory, then a path derived from the java.class.path string
     (M49 final source blob c104e379862cf280a5944ed97c35cfe568c5f348 confirmed;
      M50 binary path and runtime reload behavior NOT VERIFIED)
```

Manifest `Class-Path` は依存JAR名の列挙であり、アプリケーション製品版やどの起動JARを選ぶかを意味しない。Eclipse `.classpath` の開発機パスも配布Runtime classpathの証拠ではない。M49 Sourceが`java.class.path`から組み立てる探索パスも、実際のJAR配置先を保証しない。複数pathや起動方式によってパスの意味が変わり得る。M50 installer ZIP内のBATは`xteamdl.jar`という固定basenameを指定するが、OSがこのBATを起動すること、インストール後のBAT/JARがZIP entryと同じであること、実行時にそのJARが選ばれることは未確認とする。

### 設定値の逐次適用と不正数値（M49 Source-only）

固定M49の`Modeler.java:1671–1706`では、設定キーを順に適用し、`MainFontSize`と`IOImageFontSize`を整数変換します。読込全体は一つの`try`に含まれ、例外を空の`catch`で無視します。不正な数値で変換が失敗すると、それ以前に代入された値を戻す処理はなく、後続キーの代入へ進みません。たとえば`MainFontSize`の変換失敗は後続の`IOImageFont`等に、`IOImageFontSize`の変換失敗は後続の`IOImageFolder`等に影響し得ます。これは固定Sourceの制御フローから分かる範囲であり、不正値の許容仕様、実GUIの表示、M50配布JARでの同等動作は未確認です（[Issue #940のsource-only追補](https://github.com/hat47x/tei-platform/issues/940#issuecomment-6499417616)）。

## 3. backup / staging / rollback の非主張

`INSTALL.DAT`で分かるのはインストール先・関連付け・shortcut等の**宣言**であり、OS上の実副作用ではない。Readmeのuninstall時ユーザーfile保持の記述は、**更新前backupや失敗時rollback**の保証ではない。既存#940にあるModelerの通常モデル保存処理のbackupと、startup JAR更新のbackupは別の操作である。

M50同梱文書のbackup/staging/rollbackの規範的な章・行は今回直接確定できておらず、**存在しない**とも**実装済み**とも言えない。保存archiveのroot READMEはmanualsをZIP内に保管し単独展開していないと説明するため、当該Readme・helpの全文探索を完了したと主張しない。インストーラ実行、途中停止、旧JAR復帰、Registry/shortcut/関連付けの復旧、実Windowsファイル状態はいずれも NOT VERIFIED。

## 4. TEIに戻す最小受入案（採用・実装・実行はしていない）

- **#943 provenance**: Source closure・build definition/toolchain・JARと依存memberのdigest・installer ZIPのdigest・文書版を独立して記録。表示版/filename/Manifest/classpathをartifact identityへ昇格させず、Source↔JARのbuild link不明ならUnknown。
- **#2019 Target配備**: 旧artifact/配置・設定・Host上の関連付け等の変更前状態、staging bytes、切替/復旧後のexact digestを別々に観測。preflight拒否、staging失敗、切替途中失敗、旧世代不在、破損historyを負例にする。既存のfilesystem/SQLite local Reference PASSを実Windows/JAR復旧へ一般化しない。
- **#1223 / ADR-0032**: インストール成功・起動成功・Readiness・Availability・drain・永続副作用の可逆性を別Evidenceとする。外部副作用やユーザー作成物の保全は自動的なEffect rollbackと同値ではない。
- **#2020 / ADR-0036**: Personal/SIer/SaaSで配布、Host変更、権限、user-owned dataの所有主体をProfile/Pluginに基づき検証する。X-TEA特有のWindows登録方式は要求しない。

## 5. 調査・変更履歴

- 2026-10-26: 既存#943、#1223、#2019、#2020及びADR-0032/0036と重複照合。公開M49 GitHub source/manifestを再読。旧archive記録はGitHub上の既存hash付きコメントで確認したが、保存ZIP/同梱文書のbyte列をこの環境で再検証したわけではない。旧チャットの`systemError`結果は採用しない。
- 2026-10-28: M49最終`Modeler.java`の保存archive snapshotと公開commitのGit blob一致を直接確認し、上表の設定読込位置と固定instance内reload未確認を補った。これはM49の静的Source確認であり、M50 JARの動作・GUIでの設定反映・インストーラー更新時の復旧を確認したものではない。詳細は[#940の固定Source記録](https://github.com/hat47x/tei-platform/issues/940#issuecomment-6467903431)。
- 2026-10-29: 保存M50 ZIP raw SHA-256を再計算して台帳値と照合し、ZIP内`Files/xteamdl.bat` entryのSHAと固定basename `xteamdl.jar` 指定を静的確認。ZIP全entry・配布マニュアル全体・実行環境での挙動を検証したものではない。
- X-TEA/TEIの実行・試験、main編集、mergeなし。実実行環境、M50 Manifest/versionの独立再読、更新手順書の規範的段落・行、source→binary build record、Windowsのbackup/staging/rollback観測は保留。
