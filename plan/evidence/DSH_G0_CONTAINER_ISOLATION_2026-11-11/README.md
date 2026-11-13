# コンテナとuidで読取りを隔離する：DSH実Toolからの許可Readと拒否Readの対（2026-11-11、#2515 / TAI #168 G0）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)（実効隔離）、[#2509](https://github.com/hat47x/tei-platform/issues/2509)（DSH実証）。比較計画は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)、機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-11（Asia/Tokyo）
- 実行環境: Windows 11 Pro上のWSL2 Ubuntu 24.04.1（kernel `6.18.33.1`）、Docker `29.5.3`、`node:24-bookworm-slim@sha256:01d566e8d94d8c619591d78652babfccd3c6992a3dffa9ed5ab39e11cf6c70e9`、nvmで導入したNode `v24.22.2`、DSH `0.2.2-rc.1`（`dsh --profile headless`）
- 状態: **合成canaryとReviewer側のMock Providerだけ**を使った事前検査。実Providerへの送信は0件。**H2/H3の比較Sessionは起動していない**。合否とOwner受理は各Issueの担当。

これまでの記録は、DSHの`read-only`や`workspace-write`が読取りを止めないこと（[H6-E1 read境界](../DSH_H6E1_READ_BOUNDARY_2026-11-07/README.md)）と、WindowsのACLなら同一ユーザーの読取りを止められること（[OSのACL](../DSH_H6E1_OS_ACL_2026-11-09/README.md)）を示した。本記録は、[TAIの実行票](https://github.com/hat47x/tai-pleroma/blob/main/docs/yi-constellation/h2-h3-comparison-first-eligible-session-execution-card-2026-11-08.md)のG0が求める形、つまり**4 Slotを別々の初期状態に置き、他SlotとReviewer資料が「在るが拒否される」こと**をLinuxで確かめる。

## 1. なぜDSHのモードでは止まらないか

`@deepseek-ai/dsh-sandbox-local` 0.2.2-rc.1の`README.md`は、Sandboxが統制する対象を書込み効果として説明している。Linuxのbwrapプロファイルはホストrootを読み取り専用で見せる。WindowsのACL runnerについては「reads stay unconfined」と書いている。同じ文書は、プロセスを隔離環境で動かす必要があるならコンテナ等を選ぶよう案内している。読取りを拒否できるのは、プロセスの資格を検査するOSである。

WSL上でDSHを直接動かす方式は使えない。WSLのプロセスは`/mnt/c`からWindows側のファイルを読め、Windowsの実行物も起動できる。

## 2. 構成

| 要素 | 内容 |
| --- | --- |
| Slot | `h2-a` `h3-a` `h3-b` `h2-b`の4つ。Slotごとに別のuid（2001〜2004）、Slotディレクトリはmode 700 |
| 参加者コンテナ | `--user <uid> --read-only --cap-drop ALL --security-opt no-new-privileges`。`/trial`（自Slotのroot、ro）、`/opt/tei`（ro）、`/dsh-home`（自SlotのDSH Home）だけをmount |
| H2とH3の差 | H3の2 Slotだけ、固定Skillを`/trial/.agents/skills/yi-tei-modeling`に置く。それ以外は同一 |
| Network | Slotごとに`--internal`のDocker network。同じnetworkにいるのはReviewer側のMockコンテナだけ |
| Mock | `/v1/messages`（ストリーミング）に応答し、あらかじめ決めた10個のTool呼出しを順に返す。要求bodyは参加者のmountの外に記録する |
| 試験時だけのmount | 他Slotの親ディレクトリを`/others`、Reviewer用ディレクトリ（uid 1000、mode 700）を`/reviewer`に、どちらもroで置く。「在るが拒否」を観測するためで、比較Sessionでは置かない |

モデルは介在しない。Tool呼出しはMockが強制するので、拒否は参加者モデルの自発的な回避ではない。

## 3. 固定した値

| 項目 | 値 |
| --- | --- |
| TEI Source | commit `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c` |
| 実行物 | 同commitを`go1.27.0 linux/amd64`で`go build -o tei ./cmd/tei`。SHA-256 `8b3988753020c90519a8831d9f9b3cc7839e1b9a726fca4a99d14cf52367eabe`、11,955,085 bytes |
| Workspace | `reference/workspace` 60ファイル。ファイル別SHA-256の一覧のSHA-256は `9e648fe35ee4c19a2a80160d296b205fa82b3e3e2a3f47908ebd73593f24c1c1` |
| Skill | `skills/yi-tei-modeling/SKILL.md` blob `9cd8475bd6489c6661ae644bf771cfcce04188b1`（SHA-256 `cf137ae534e655881dfc686d70afbf14138cb21a61885b987c224dd893ea2282`） |
| Catalogの直接実行 | 13,190 bytes、SHA-256 `3b7d69763be1161e07a7b1461e8aacd90531465d4971f9e31af2f12bed2331fc` |

実行物は[四Sessionの記録](../DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md)のWindows版（SHA-256 `186d06ce…`）とは別物である。

## 4. 観測した結果

| 項目 | h2-a | h3-a | h3-b | h2-b |
| --- | --- | --- | --- | --- |
| Tool Processのuid | 2001 | 2002 | 2003 | 2004 |
| `CapEff`／`NoNewPrivs` | 0／1 | 同じ | 同じ | 同じ |
| `cwd`と実行物のSHA-256 | `/trial`、`8b398875…` | 同じ | 同じ | 同じ |
| Workspaceのdigest | `9e648fe3…` | 同じ | 同じ | 同じ |
| `version` `commands` `inspect` `trace` `validate` | 全てexit 0 | 同じ | 同じ | 同じ |
| 自Slotの`SKILL.md` | 無い | 読めた（`cf137ae5…`） | 読めた（同じ） | 無い |
| `skill` Tool | 未知と応答 | 5,431 bytesを受信 | 同じbyte | 未知と応答 |
| 他3 Slotのcanary・root・Skill、Reviewer資料（`bash`） | 10件ともPermission denied | 同じ | 同じ | 同じ |
| 同じ対象への`read` Tool | `EACCES` | 同じ | 同じ | 同じ |
| `/others`への`grep` Tool | os error 13 | 同じ | 同じ | 同じ |
| Tool呼出しと結果 | 10対10、IDが全件対応 | 同じ | 同じ | 同じ |
| `model catalog`のTool結果 | 13,190 bytes、直接実行とbyte一致 | 同じ | 同じ | 同じ |
| 外部IPへの接続、外部名の解決 | 失敗 | 同じ | 同じ | 同じ |
| `/trial`と`/others`への書込み | Read-only file system | 同じ | 同じ | 同じ |
| canaryの出現（要求body、stdout、DSH Home） | 0件 | 0件 | 0件 | 0件 |

- 拒否は`ls -ld`で対象の存在を確かめた上での結果である。
- [四Sessionの記録](../DSH_H2_H3_FOUR_SESSION_2026-11-07/README.md)で見た8,192 bytesでの切断は、Linuxの`bash` Toolでは13,190 bytesの出力で起きなかった。
- `skill` Toolの結果は、Windowsでの5,482 bytesと大きさが違う。ラッパーに入るBase directoryのパスが違うためと見ているが、照合はしていない。
- 要求bodyの最上位には`dsh_session_log`と`dsh_plugin_packages`が載っていた。
- 先行して1 Slotで行った対照では、DSHのモードを`danger-full-access`にしても拒否は変わらず、所有者のuidで実行すると同じ手順でcanaryが読めた（要求body・stdout・Session Logに計106件）。走査が検出できることの確認である。この対照の生データは本記録に含めていない。

## 5. この記録が示さないこと

- **実Providerへの送出**。Mockだけを相手にした。実際の送出範囲、Provider側の保存は未観測。
- **H2/H3の比較**。Task、Skill bytes、独立Expected、割付順は変えておらず、比較Sessionは1件も起動していない。既存の四Sessionの資格も変えない。
- 内部networkからDockerホストのgatewayアドレスへ届くかどうか。
- DSH Home配下の各ファイルの内容の棚卸し。Session Logは`zstdDecompressSync`で展開して走査したが、frame数は数えていない。
- コンテナ脱出やカーネル共有への耐性。Dockerの既定の強度に依存する。
- Linux版の実行物と`bash` Toolを新しいTrial Groupの固定値として受理するかどうか。TAI #168のOwner判断である。

## 6. 生データと再現

| ファイル | 内容 |
| --- | --- |
| `four-slot-summary.json` | Slotごとの要求数、Tool呼出しと結果の対応、Catalogのbyte照合、`skill` Toolの結果、拒否の終了コード、canaryの走査結果 |
| `trial.sh` | 4 Slotの生成、uidの付与、Slotごとのnetwork・Mock・参加者コンテナの起動、集計 |
| `gen_steps.js` | Slotごとの10個のTool呼出しを生成する |
| `mock.js` | Reviewer側のMock Provider |
| `results2.js` | 要求bodyとDSH Homeから結果とcanaryを集計する |

要求の生bodyとSession Logはコミットしていない。**canaryは合成であり、実secretではない**。

前提は次の4つである。`trial.sh`は`~/dsh-poc`配下にこれらがある想定で書いている。

```sh
# 1. 固定Sourceの展開とビルド
git archive 50ebf8014340fd1906f6b35cfa30173e0ee2fc6c | tar -x -C ~/dsh-poc/tei-src
(cd ~/dsh-poc/tei-src && GOFLAGS=-buildvcs=false go build -o ~/dsh-poc/g0/bin/tei ./cmd/tei)
# 2. 直接実行用の公開Workspace
mkdir -p ~/dsh-poc/g0/slot/reference && cp -r ~/dsh-poc/tei-src/reference/workspace ~/dsh-poc/g0/slot/reference/
# 3. Node 24とDSH
nvm install 24.22.2 && npm install --prefix ~/dsh-native @deepseek-ai/dsh@0.2.2-rc.1
# 4. 実行
bash trial.sh
```

Slotディレクトリは別uidの所有になるので、片付けにはrootの補助（`docker run`でのmount）が要る。`trial.sh`の冒頭が行う。
