# DSH外部HarnessからのTEI read-only CLI実起動（2026-11-07、#2509 H2）

- 対象Issue: [#2509 DSH外部Harness実証](https://github.com/hat47x/tei-platform/issues/2509)。本記録は同Issueの「最小実験」のうち、**外部AIハーネスの1つの実SessionでTEI実行物の実起動を観測した範囲**である。
- 上位計画: [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の初回Runbook §19〜§21。Task文面・試行順序・Holdoutの正本はTAI側にあり、この記録は複製しない。
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 10.0.26100、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`、Go `go1.27.0 windows/amd64`
- 状態: **H2 Tool-onlyの実Tool呼び出しは1 Sessionで限定達成**。H3（Product Skill実Load）、四Session比較、Skillの増分価値、Egress、OS隔離、Owner受理は**NOT RUN**。

この記録が示すのは、外部AIハーネスが**TEI実行物を実際に起動し、引数と出力が同一Sessionのログに残った**ことだけである。モデル回答が正しいこと、Skillが有用であること、外部への情報送出が許容範囲であることは示さない。

## 1. 固定した実験入力

| 要素 | 値 |
| --- | --- |
| TEI Source | `hat47x/tei-platform` commit `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`（TAI Runbook §19.1の固定値と同じ） |
| 実行物 | 同commitを`go build -o tei.exe ./cmd/tei`で作った単一実行物。SHA-256 `186d06ce8c79a259a834c79de28b675103113d1218b78876737b15bc4ce69707`、12,006,912 bytes |
| Workspace | 同commitの`reference/workspace`を実行用の隔離ディレクトリへbyteコピーしたもの（60ファイル、manifest SHA-256 `7d9a6039622c015ba383d248fbb0571783b17f0ce7bc43494aa0b08d00cfff80`） |
| Subject | `semantic-data:sales.order`。Sourceは`semantic/sales/order.tei`、SHA-256 `f2063418b893a60a9f948edbb2b23ec821f7a15ed8e5863c36e1b72abade1e33`、863 bytes（TAI Runbook §19.1と一致） |
| Participant Task | TAI固定fixture `sk02-tei-public-read-participant-task-2026-11-06.txt`のUTF-8・末尾LF込み実byte。SHA-256 `38a0b238018e050f2116f10f1ca2cb3a54319c2743e2fb9bbe128d4be96a84cf`、897 bytes（Runbook §20と一致） |
| Harness Model | `openrouter`経由`~deepseek/deepseek-flash-latest`。DSHの既定経路`deepseek-official`は本環境にCredentialが無く、起動時に`MISSING_CREDENTIAL`で停止した |
| Tool Profile | `--permission-mode read-only --approval-policy never`、`maxParallelToolCalls: 1`、`session-log-deepseek.enabled: false` |
| 起動方法 | `dsh --profile headless --patch <overlay.yml> --json < task.txt` |

実行用の隔離ディレクトリには`bin/tei.exe`・`workspace/`（byteコピー）・`task.txt`だけを置いた。Product Skill（`skills/yi-tei-modeling`）、TAI repo、Oracle、採点表は置いていない。

## 2. 観測した実行（`h2-08-openrouter-participant-task`）

- Session ID: `session-050904a2-78c8-4294-842a-91d63394f1e2`、`cwd` = 実行用隔離ディレクトリ
- Tool call 56件、Tool result 55件。結果が無い1件は、Harnessが結果を受け取る前にそのStepを終えたもの
- 最終状態: `reason.kind = completed`（エラー終了ではない）
- 使用Tool: `pwsh`・`read`・`grep`の3種。Skill Toolの呼び出しは無い
- DSHの`status.usage`合計: input 184,593 / output 26,069 / cache read 774,144（`totalTokens`はDSHの算出値の合計 984,806）。**これはDSHの集計であり、Provider側の請求額ではない**

うちTEI実行物を起動した呼び出しは32件。主要なもの（生のTool resultは同ディレクトリの`cli-*`）:

| CLI呼び出し（実引数） | 観測した結果 | 生データ |
| --- | --- | --- |
| `tei commands` | `tei.cli.discovery/v0`、`scope: selected-read-only-commands` | `cli-commands.json` |
| `tei version -json` | `tei.cli.version/v0`、`vcsRevision: 50ebf801…`、`go1.27.0 windows/amd64` | `cli-version-json.txt` |
| `tei model catalog -workspace=<copy>` | `ok: true`、`worldFingerprint: sha256:c92de09f…`、`sourceIndexFingerprint: sha256:1827f4b4…` | `cli-model-catalog.txt` |
| `tei inspect -workspace=<copy> -subject-kind semantic-data -subject-id sales.order` | 属性6件・関係2件、`sourceStatus: indexed`、`sources.artifacts[0].fingerprint: sha256:f2063418…` | `cli-inspect-sales-order.txt` |
| `tei inspect … -subject-kind semantic-attribute -subject-id sales.order.total` | `type: std.money`、定義範囲`order.tei`12行目のoffset 326–331 | `cli-inspect-sales-order-total.txt` |
| `tei source-index … -subject-kind semantic-data -subject-id sales.order` | artifact `semantic/sales/order.tei`、`role: definition` | `cli-source-index-sales-order.txt` |
| `tei validate -workspace=<copy> -json` | `ok: true`、warning 3件 | `cli-validate-json.txt` |
| `tei validate -workspace=<copy>`（text形式、`EXIT=$LASTEXITCODE`） | `ok`、`EXIT=0` | `cli-validate-text.txt` |
| `tei model catalog -workspace=<固定worktree>` | 上のコピー実行と同一の`worldFingerprint`・`sourceIndexFingerprint` | `cli-model-catalog-pinned-worktree.txt` |

固定worktree（`reference/workspace`の元）でも`validate -json`・`inspect`・`source-index`を実行し、指紋が一致した。**同じ固定commitの同じSource byteを2経路で読み、同じ指紋が返った**ことを示す。ただし両者は同一Processではなく、同時Snapshotでもない。

## 3. 失敗した試行から分かったこと

すべての試行は`attempts-summary.json`に残した。本文（thinking/answer）は含まず、Tool入力・結果のbyte数とdigestだけを持つ。

| 試行 | 結果 | 観測した原因 |
| --- | --- | --- |
| `h2-01` | `MALFORMED_RESPONSE` | モデルが1ターンに2つのTool callを連結した不正JSONを返し、Harnessが拒否した。この試行は`cwd`が`D:\GIT\hat47x`で、参加者用の隔離ディレクトリではなかった |
| `h2-01b`・`h2-02`・`h2-04`・`h2-05-serial` | `MALFORMED_RESPONSE` | 同じく複数Tool callの連結。`maxParallelToolCalls: 1`を設定しても、**モデル出力そのものの不正は解消しなかった** |
| `h2-06-invalid-config` | `INVALID_CONFIG` | DSHの`compat`へ`parallelToolCalls`という存在しないスイッチを指定した。DSHは受理できるスイッチ名を列挙して拒否した |
| `h2-03-single-tool`・`h2-07-openrouter-single-tool` | `completed` | 単一Tool呼び出しを指示した短い検証Task。**参加者Taskの比較試行ではない** |

ここから言えるのは、**Harnessの1回の実Tool呼び出しは成立するが、長いTaskではモデル出力の連結で途中停止しうる**こと。停止はTool側ではなくHarness側の応答解析による。したがって「H2が成立した」は、**完了まで到達した試行が1件ある**という意味に限る。

## 4. 情報境界で観測したこと（未解決を含む）

- `read-only`のSandboxでは、**作業場所の外にある読み取り可能なパスも読めた**。`h2-08`は固定worktreeの`reference/workspace`を直接指定し、コピーと同じ指紋を得た。参加者へ渡す範囲をディレクトリで限定する意図はread-onlyでは強制されない。Oracleや他レーン資料を同じホストに置く構成では、TAI Runbook §20のParticipant隔離条件を満たさない。
- `session-log-deepseek`は`enabled: false`で実行したが、**送出されなかったことの観測は本記録に無い**。通信経路のMock/Proxy計測は#2509の別項目で未実施。
- `workspace-write`のSandbox準備は本環境で`SetNamedSecurityInfoW failed (Win32 5)`となり、書込みを伴う比較は実行できなかった。read-onlyでは同じ操作が成功した。**書込み・Effectを伴うH2/H3は本記録の対象外**。
- Product Skillの実Load、Skill root・参照文書の実Read、H3との比較は**未実施**。実行用ディレクトリへSkillを置かなかったため、H2のSkill非露出は「置かなかった」ことによる。Skill Loaderが読む場所を観測してはいない。

## 5. モデル回答（採点前の生データ）

`final-answer.md`はHarnessが返した最終回答の実byte（6,656 bytes、SHA-256 `ecf7e56a30dfc8e2a1b14a4c9a1b7d6e9d5b8b196b3da0ca1f68d2db5835cc0b`）。内容は`inspect`・`source-index`・`validate`の出力と整合する範囲で、`sales.order.total`の型が`std.money`であること、定義元Sourceと`f206…`指紋、validateのwarning 3件を述べている。

**この回答はまだ採点していない。** TAI RunbookのReviewer-onlyの独立Expectedと照合していないため、正答・誤答・未確認の保持のいずれとも判定しない。回答には、CLI出力に直接は現れない記述（`knowledge-context`の`unresolved`項目、`std`名前空間の`not-indexed`など）が含まれる。その出所は本記録では特定していない。

## 6. この記録が示すこと

- DSHの実Session（`session-050904a2…`）で、HarnessのTool呼び出しから**TEI実行物の実起動・実引数・実出力**までを同じログ上で突合できた。
- 固定commit `50ebf801`の実行物と、同commitの`reference/workspace`のbyteコピーで、`commands`・`version`・`model catalog`・`inspect`・`source-index`・`validate`がread-onlyで成功した。`validate`は終了コード0。
- コピーと固定worktreeの2経路で、`worldFingerprint`と`sourceIndexFingerprint`が一致した。

## 7. この記録が示さないこと

- H2/H3の四Session比較、Skillの増分価値、独立Expectedによる正答採点。
- DSH以外のHarness、DSH `sdk-minimal`、MCP経由の呼び出し。
- 実Egress（`session-log`拡張、Telemetry、Provider側の保存）と、OS/Network境界での拒否。
- 書込み・Effect・RunPlan・実DB/Providerへの作用。
- Participant隔離（read-onlyで作業場所外を読めるため）と、Oracleの非漏出。
- 製品の受入、Ownerの意味受理、#1119のGate、TAI #168のH2合格判定。TAI側の採点票と突合するまで、本記録は**TEI側の実行証拠**に留まる。

## 8. 生データと再現

同ディレクトリのファイルが観測値の原本である。`cli-*`はDSHのTool result文字列をそのまま保存し、改行はそのまま残した（PowerShellの`Out-File`を通すと符号化が変わるため、Pythonの`newline=""`で書いている）。`attempts-summary.json`は全9試行のTool入力・結果digest・usageの構造化記録である。

| ファイル | bytes | SHA-256 |
| --- | --- | --- |
| `attempts-summary.json` | 93,076 | `6ec9da8d4976ad89845c45aebbd2fac409fe8e61382d35d012786bca5dbde742` |
| `cli-commands.json` | 6,350 | `d87029abc05faf42e56c246b260a4c9b775fcafd6fae1740cc138d65c7c365bd` |
| `cli-version-json.txt` | 1,380 | `8178a2eba146f9d43855b589449a4aae98572fa040d6f8f4286e18517c51a702` |
| `cli-model-catalog.txt` | 8,192 | `18f71bd6026c83d3c697c07c7982bbcc015bea6344f6c2c0c1c5d5f3b1945670` |
| `cli-model-catalog-pinned-worktree.txt` | 8,192 | `18f71bd6026c83d3c697c07c7982bbcc015bea6344f6c2c0c1c5d5f3b1945670` |
| `cli-inspect-sales-order.txt` | 2,649 | `3304790b38fabf6ad38cdea3cb5f60539d1efa4d2c9a9f3f333fb19acdf76daf` |
| `cli-inspect-sales-order-total.txt` | 1,941 | `dd2953694783e82ef2a3052a1258b06f0a932b10ee3224cd3b13974f0f81bb06` |
| `cli-source-index-sales-order.txt` | 1,882 | `72e402483584b0122ff09fb1e2d5e1a9e754cd9090185eb691ddd7eac2d79eba` |
| `cli-validate-json.txt` | 1,900 | `b32342b1e8b9a9bb7bd48d21e2f0318e02498379b72e584d8710a43a1b6b009e` |
| `cli-validate-text.txt` | 1,503 | `768de54cb47da0723e31f3f382651fc8ba8db1a7aa14f6903e026d4f6cc1e2d8` |
| `final-answer.md` | 6,656 | `ecf7e56a30dfc8e2a1b14a4c9a1b7d6e9d5b8b196b3da0ca1f68d2db5835cc0b` |

`cli-model-catalog.txt`と`cli-model-catalog-pinned-worktree.txt`はdigestが同じでも、別々の実行から得た別ファイルである。

再現する場合の要点:

```sh
git -C <tei checkout> worktree add --detach <dir> 50ebf8014340fd1906f6b35cfa30173e0ee2fc6c
cd <dir> && go build -o <isolated>/bin/tei.exe ./cmd/tei
# <isolated> には reference/workspace の byte コピーと task.txt だけを置く
dsh --profile headless --patch <overlay.yml> --permission-mode read-only --approval-policy never --json < <isolated>/task.txt
```

`<overlay.yml>`が指定するのはProvider経路・Model・`maxParallelToolCalls: 1`・`session-log-deepseek.enabled: false`だけである。Credentialは環境変数で渡し、リポジトリへ保存しない。
