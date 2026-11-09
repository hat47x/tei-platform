# DSH実HarnessでのH2/H3四Session実行（2026-11-07、#2509 / TAI #168 初回比較）

- 対象Issue: [#2509](https://github.com/hat47x/tei-platform/issues/2509)。上位計画は[TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)の初回Runbook §19〜§21で、Task・順序・Holdoutの正本はTAI側にある。
- 前提記録: 同リポジトリの[DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07](../DSH_H2_TOOL_ONLY_READ_ONLY_2026-11-07/README.md)（H2-01の実行と失敗試行の記録）
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 10.0.26100、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`、Go `go1.27.0 windows/amd64`
- 状態: **H2→H3→H3→H2の四Sessionを実行し、H3でProduct Skillの実Loadを観測**。ただし正答採点・Skill効果の因果判定・Egress・OS隔離・Owner受理は**NOT RUN**。

この記録が示すのは、**同じTask・同じ実行物・同じWorkspace byteで、Skill非公開の腕とSkill実Loadの腕を各2回、独立Sessionで実行できたこと**である。Skillに効果があったことは示さない。

## 1. 実行条件

| 要素 | 値 |
| --- | --- |
| TEI Source / 実行物 | commit `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`、Go `go1.27.0 windows/amd64`、実行物SHA-256 `186d06ce8c79a259a834c79de28b675103113d1218b78876737b15bc4ce69707`（前記録と同一） |
| Workspace | 同commitの`reference/workspace`をbyteコピーした`workspace/`（60ファイル、manifest SHA-256 `7d9a6039622c015ba383d248fbb0571783b17f0ce7bc43494aa0b08d00cfff80`） |
| Participant Task | TAI固定fixture実byte、SHA-256 `38a0b238018e050f2116f10f1ca2cb3a54319c2743e2fb9bbe128d4be96a84cf`（897 bytes、全4 Sessionで同一） |
| Harness / Model | DSH `0.2.2-rc.1` headless、`openrouter`経由`~deepseek/deepseek-flash-latest`、`maxParallelToolCalls: 1`、`session-log-deepseek.enabled: false` |
| Tool権限 | `--permission-mode read-only --approval-policy never`（全4 Sessionで同一） |
| H2の参加者Root | `bin/tei.exe`・`workspace/`・`task.txt`のみ。`.agents/skills`を置かない（2 Sessionで再利用） |
| H3の参加者Root | 同じ3要素＋`skills/yi-tei-modeling`を`.agents/skills/`へ配置（2 Sessionで再利用） |
| 順序 | `h2-a` → `h3-a` → `h3-b` → `h2-b`（RunbookのH2→H3→H3→H2に一致） |

Skillの配置はDSHの実Skill探索で発見されることを確認した。同一のSkill一覧要求に対して、H3 rootは`yi-tei-modeling`を含み、H2 rootは含まない（`skill-list-h3-root.txt`・`skill-list-h2-root.txt`）。`h2-a`はSkill配置前のrootで実行しており、ログにSkill Tool呼出しも`.agents`への参照も無い（Skill配置後にそのrootで別Sessionを走らせていない）。

## 2. 四Sessionの観測値

| Session | 腕 | Session ID | Tool call | TEI起動 | Skill Tool呼出し | Skill読込byte | 最終状態 | DSH usage（input / output / total） |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `h2-a` | H2（Skill非公開） | `session-050904a2-78c8-4294-842a-91d63394f1e2` | 56 | 32 | 0 | 0 | `completed` | 184,593 / 26,069 / 984,806 |
| `h3-a` | H3（Skill実Load） | `session-318ab3dd-d63f-4397-884a-42a0b1e72c81` | 23 | 11 | 1 | 5,482 | `completed` | 77,704 / 14,306 / 194,666 |
| `h3-b` | H3（Skill実Load） | `session-21186932-ac06-4e26-90b3-2faa2895e7ce` | 22 | 12 | 1 | 5,482 | `completed` | 56,903 / 7,494 / 362,381 |
| `h2-b` | H2（Skill非公開） | `session-95f34eef-66c1-4a48-8c67-14f782a5dc99` | 45 | 26 | 0 | 0 | `completed` | 55,225 / 15,770 / 776,019 |

`totalTokens`はDSHの算出値の合計で、**Provider側の請求額ではない**。`cacheReadTokens`は各Sessionで変動するため表に含めていない（`session-comparison.json`には入っている）。

H3の2 Sessionが受け取ったSkill本文は**byte一致**（5,482 bytes、SHA-256 `cbf3d21f4dc387f26624e47c583a8cd708c7936d230008977f54d1daebd8c33b`）。生のTool resultは`skill-load-h3-a.txt`と`skill-load-h3-b.txt`に保存した。

## 3. 回答の機械照合（採点ではない）

4つの最終回答（`answer-*.md`）に対し、Taskが要求する項目が**文字列として現れるか**だけを機械的に確かめた。

| 項目 | h2-a | h3-a | h3-b | h2-b |
| --- | --- | --- | --- | --- |
| `sales.order.total`（属性識別子） | あり | あり | あり | あり |
| `std.money`（型） | あり | あり | あり | あり |
| `order.tei`（定義元ファイル） | あり | あり | あり | あり |
| `f2063418…`（Source指紋） | あり | あり | あり | あり |
| validateのwarning 3件 | あり | あり | あり | あり |
| `not-indexed`／「未確認」の明示 | あり | あり | あり | あり |
| 回答byte数 | 6,656 | 4,546 | 7,147 | 6,695 |

**この表は正答判定ではない。** 文が正しい主張をしているか、未知を正しく残したかは、TAI RunbookのReviewer-only独立Expectedで採点する必要がある。ここで確かめたのは「要求された語が回答に現れたか」だけである。

## 4. 実行中に観測した比較条件の乱れ（重要）

- **Taskが指すpathと実参加者Rootのpathが一致していない**。Participant Taskは`reference/workspace`と書くが、隔離Rootでは`workspace/`に置いた。Taskのpathと実際に読んだ`workspace/`の関係を回答中で明示したSessionは`h3-a`・`h3-b`・`h2-b`の3件で、`h2-a`は言及していない。**四Sessionの比較条件としてこの点は不揃い**であり、腕の違いではなく経路差として扱う。次の実行ではTask側のpathを隔離Rootの実pathに合わせるか、Rootのディレクトリ名を`reference/workspace`にする。
- **read-only Sandboxは作業場所外も読める**（前記録と同じ）。実際に`h2-a`は`D:\GIT\hat47x\tei-wt-dsh-h2-20261107\reference\workspace`を直接読み、`workspace/`コピーとのByte一致を比較した上で、その外部パスに対しても`model catalog`・`validate`・`inspect`・`source-index`を実行した。**参加者Rootを分けても、同じホスト上の他rootを読める制約は残る**。H3の2 Sessionは`.agents/skills`配下だけを読んでおり、他レーン資料は読んでいない。
- `h2-a` は前記録と同じSession `session-050904a2…` を使っている。**このSessionの`cwd`は分離Rootであり、前記録の`h2-01`（`cwd`が`D:\GIT\hat47x`だった試行）とは別の試行である**。ただし同一Sessionの履歴を使い続けているため、4 Sessionは互いに独立というより「腕ごとに1 Sessionを再利用した」構成である。次の実行では腕ごとに新区間を作る。

## 5. この記録が示すこと

- 同じTask・同じ実バイナリ・同じWorkspace byteで、Skill非公開の腕2回とSkill実Loadの腕2回を、**4 Sessionで完了できた**。
- H3の2 Sessionは、**同一byteのSkill本文をTool経由で実際に受け取った**。Skillを置いただけ・名前を宣言しただけではない。
- H2の2 SessionはSkill Toolを一度も呼ばず、Skill一覧にも`yi-tei-modeling`が現れなかった。

## 6. この記録が示さないこと

- **Skillの増分価値**。4 Sessionの回答は未採点であり、Tool起動回数とTokenは腕間で大きく異なるが、その差をSkillの効果へ帰属できない（Task path不一致、Session再利用、モデル応答の揺れを含む）。
- 回答の正否、未知の保持の適切さ、誤操作の拒否。
- 実Egress、OS/Network境界の拒否、`session-log`拡張の非送出。
- TAI #168のH2/H3合格判定、Ownerの意味受理、#1119のGate。
- DS以外のHarness、`sdk-minimal`、MCP経由。

## 7. 生データ

| ファイル | 内容 |
| --- | --- |
| `session-comparison.json` | 4 SessionのTool入力・結果byte数・digest・usage。thinkingと回答本文は含まない |
| `skill-load-h3-a.txt` / `skill-load-h3-b.txt` | H3が受け取ったSkill本文の実byte |
| `skill-list-h3-root.txt` / `skill-list-h2-root.txt` | 同じSkill一覧要求への回答（Root間の非公開/公開の差） |
| `answer-h2-a.md`・`answer-h3-a.md`・`answer-h3-b.md`・`answer-h2-b.md` | 各Sessionの最終回答実byte（採点前） |

再現する場合は、`bin/tei.exe`・`workspace/`・`task.txt`だけのRootと、それに`skills/yi-tei-modeling`を`.agents/skills/`へ加えたRootを用意し、`dsh --profile headless --patch <overlay.yml> --permission-mode read-only --approval-policy never --json < task.txt`を4回、別Sessionで実行する。
