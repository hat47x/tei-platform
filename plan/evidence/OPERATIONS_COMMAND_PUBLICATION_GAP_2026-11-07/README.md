# 操作面は CLI の公開面に出ているか（2026-11-07）

- 対象: [操作面の設計候補](../../design/OPERATION_SURFACE_DESIGN_2026-11.md)の W5（CLI と出力契約）と、15章の未決1（公開面を増やす判断）
- 位置づけ: **実行して観測した記録**。公開するかどうかの判断・Owner受理・Gate合否ではない。
- 計測した版: `raw/head.txt`（`origin/main` `074ec2d9` の作業ツリー。追加は`plan/evidence/`配下のみ）。

設計の W5 は「`tei operation surface` が動き、`tei commands` に read-only として現れ、**失敗時も同じエンベロープになる**」を完了条件にしています。この記録は、その3点を**現行のCLIについて**確かめたものです。

## 1. 観測したこと

| 観測 | 結果 |
| --- | --- |
| `tei commands` の `scope` | `selected-read-only-commands` |
| 公開されているコマンド数 | 19 |
| CLIが受け付けるコマンド数 | 53 |
| **受け付けるが公開していない** | **34**（`operations`・`artifacts`・`periods`・`grants` など） |
| **公開しているが受け付けない** | 0 |
| `tei operations` の成功出力 | 単段のJSON（`schemaVersion` は `tei.operations.operation-surface/v0`）。**`ok`も`result`も無い** |
| `tei operations` の失敗時 | **stdout 0バイト**、stderrにテキスト。JSONエンベロープではない |

`tei commands` は「選んだ読取りコマンド」だけを載せると明示しており、全コマンドの一覧ではありません。したがって「34件が漏れている」は**欠陥ではなく設計上の選択**です。一方、`operations` はその選択に含まれていません。

## 2. 設計との差

- 設計9章の出力契約は「外側が `ok`／`result`／`error` を持ち、`result` が操作面」という**二段の型**を求めています。現行の `tei operations` は**一段**で、`ok`も`result`もありません。
- 設計のW5完了条件のうち「`tei commands` に read-only として現れる」と「失敗時も同じエンベロープになる」は、**現行では満たしていません**。
- 設計の実装状況（2026-11-01）は「W5（CLI）の読み取り部分を追加しました」と書いており、上の2点を**追加済みとは書いていません**。記述と観測は食い違っていません。

つまり、**操作面は動くが、公開面には出ていない**。15章の未決1（公開してよいか）は、まだ判断されていないので、公開されていないこと自体は設計どおりです。

## 3. この観測が示すこと

- W5 の3条件のうち、**動く**は満たし、**公開面へ出る**と**失敗時のエンベロープ**は満たしていない。
- 15章の未決1を決める前に、**公開した場合に何が変わるか**が具体的に見えます。`tei commands` に1行足すこと、失敗時に1件のJSONを返すことが必要になります。
- `tei commands` の `scope` が「選んだ読取りコマンド」である以上、`operations` を載せるかどうかは**公開面の選択**であり、機械的な漏れの修正ではありません。

## 4. この記録が示さないこと

- **公開すべきという判断**。15章の未決1はOwner判断です。この記録は現状の差を示すだけです。
- **他の34コマンドの扱い**。`tei commands` が選んだ範囲だけを載せる設計であるため、それらが「載るべき」とは主張しません。
- **失敗時エンベロープの設計**。どの `error.code` を返すか、既存の `validate -json` の形に合わせるかは、CLI契約（#2824）側の判断です。
- **実運用**。合成のWorkspaceと読み取りのみです。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `compare_commands.py` | 受け付けるコマンドと公開するコマンドの**名前だけ**を比較する読み取り専用の計器（Goを解析しない） |
| `run.sh` | 再現手順 |
| `raw/head.txt` | 計測した版とToolchain |
| `raw/pre-status.txt` | 計測前の作業領域（このディレクトリを除き0行） |
| `raw/command-coverage.txt` | 34件の名前と、公開されているが受け付けない0件 |
| `raw/commands-catalog.json` | `tei commands` の出力 |
| `raw/operations-success.json` | `tei operations` の成功出力 |
| `raw/operations-failure.stdout.txt`・`raw/operations-failure.stderr.txt` | 失敗時の出力（stdoutは0バイト） |
| `raw/summary.txt` | 上の表を作る元の一覧 |

## 6. 生データのSHA-256

| ファイル | SHA-256 | バイト数 |
| --- | --- | --- |
| `compare_commands.py` | `787108ebffb32f73f1966074ec02f9e167bbe005d59f98207e7274d2986c08e5` | 2571 |
| `run.sh` | `dd97a0c09592eba2c0fffacff33e611e0e7a949dd309de5544efe449f647863c` | 3267 |
| `raw/head.txt` | `2d48ba32c80bcd6184b6dcfbd0a240099b857753da2d293494aa069e066f19b9` | 73 |
| `raw/pre-status.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |
| `raw/command-coverage.txt` | `f321c9ef47eee333c562fd7018ae758afa73f232812bb300369c7e8a3835f294` | 761 |
| `raw/commands-catalog.json` | `8e2482171409c1dcf6f94f036bfe03d048e7321f5f1c184494731385672220b9` | 7474 |
| `raw/operations-success.json` | `e6a8443c13d8715f2015de58d0abf25691065d8d1c8a42a0474723692622d171` | 3268 |
| `raw/operations-failure.stdout.txt` | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`（空） | 0 |
| `raw/operations-failure.stderr.txt` | `272b2ef23b2e45821f21375b7f5cf6b812afae73f6e6a0bfa58d73c91940bbda` | 195 |
| `raw/summary.txt` | `1c2b465544b15a5d600daa7b09c8658583d086feb70696956ae7ad4c35b71fba` | 810 |

`raw/operations-success.json` のSHA-256は、[構成Kernel移行Gateの実行記録](../COMPOSITION_FIRST_CONSUMER_RUN_2026-11-07/README.md)の`raw/operations.json`と同じ値です。同じHEADではない版で同じ入力から同じ出力が出たことを、ハッシュの一致で確かめられます（`standard-crud`のWorkspaceの内容が同じであることの傍証で、同一性の証明ではありません）。
