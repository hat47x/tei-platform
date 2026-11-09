# DSH外部HarnessのLane分離：worktreeだけでは他レーンのSourceを読める（2026-11-07、#2515 H6-E4）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E4`（多並列Lane）。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **合成canaryのみ**。別OS identity・ACL・mount・Network分離は**未構成**（`NOT RUN`）。同一OS principalで2 Laneを分けただけの比較。

`H6-E4`は「別worktreeのAgent A/Bを置き、**worktreeだけで隔離済みとしない**。別Identity/ACL/mount/egress制限の実拒否とLaneごとの成果・非公開入力を観測する」ことを求める。本記録は、**分離をディレクトリだけで行った場合**に何が起きるかを実測したものである。

## 1. 何をしたか

1. `laneA/`と`laneB/`を作り、それぞれに固有の合成canaryを置いた。

| Lane | root | 自分のcanary |
| --- | --- | --- |
| laneA | `…\h6e4\laneA` | `LANE_A_CANARY_401` |
| laneB | `…\h6e4\laneB` | `LANE_B_CANARY_402` |

2. Laneごとに別Session・別cwdでDSHを起動し、Mock Endpointから`read`を順に強制して、**自分のcanaryと相手Laneのcanaryの両方**を読ませた。
3. ローカルのTool結果と、Mockが受け取った要求bodyの両方でcanaryを検索した。
4. **OS identity・ACL・mount・Networkの分離は行っていない**。両Laneは同じWindows User・同じProcess treeで実行した。

## 2. 観測した結果

| Lane | 自分のcanaryを読む | **相手Laneのcanaryを読む** | 相手canaryのモデル入力到達 | 相手canaryのSession Log到達 |
| --- | --- | --- | --- | --- |
| laneA | completed | **completed** | あり（1回） | あり（3回） |
| laneB | completed | **completed** | あり（1回） | あり（3回） |

- **worktree相当のディレクトリ分離だけでは、他レーンのSourceを読むのを止められない。** laneAのSessionは`…\laneB\src\own-canary.txt`を`completed`で読み、その内容がモデル入力とSession Logへ載った。laneBも同様にlaneAのcanaryを読んだ。
- 相手Laneのcanaryは、モデル入力に1回、Session Logに3回現れた。Session Log側の回数が多いのは、`tool/result` eventが結果本文と`meta`の両方を持つためである（既存の[送出計測](../DSH_SESSION_LOG_EGRESS_MOCK_2026-11-07/README.md)と同じ構造）。
- 両Laneとも同じ要求サイズ（40,501／66,956／37,107／37,154 bytes）で、**区別できるのはcwdとcanaryだけ**だった。Process・User・Networkの差は無い。

## 3. この記録が示すこと

- `H6-E4`の「worktreeだけで隔離済みとしない」は、**この構成で実証された**。ディレクトリ分離は読取りの境界にならない。
- 並列Laneを機密分離として扱うには、**別OS identity・ACL・mount・Network egressのいずれか**が必要になる。`read-only`のSandbox modeは、[H6-E1](../DSH_H6E1_READ_BOUNDARY_2026-11-07/README.md)で示したとおり読取りを制限しない。
- Laneごとの成果と非公開入力は、同じHost・同じUserのままでは**互いに可視**である。

## 4. この記録が示さないこと

- 別OS identity・別User・AppContainer・ACL・mount名前空間・Network egress制限を構成した場合の拒否。本記録は**分離していない**状態の観測である。
- 共有Cache・Credential・MCP Portの状態（`H6-E4`が求める負例）。本記録はファイル読取りだけ。
- 同時並列実行（本記録はLaneを順に実行）。
- 実Providerへの送出、Egress制御、`H6-E4`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `crosslane.json` | fixtureの定義（Lane root・canary・digest）、両Laneのread結果、要求ごとのbyte数とSHA-256、canaryの出現 |

生の要求bodyはコミットしていない。**canaryは合成であり、実secretではない**。

再現する場合の要点:

```sh
python mock_provider_read_seq.py 8799 <record-dir> <lane-paths.json>
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "Investigate."
```

`lane-paths.json`に自Laneと他Laneのpathを両方入れ、cwdだけをLane rootに変えて2回実行する。

## 6. `H6-E4`への含意（判断はしない）

`H6-E4`が必要とする「Laneごとの実効分離」は、**ディレクトリ構成では成立しない**。別Identity/ACL/mount/egressのいずれかを構成し、その**実拒否**を観測する必要がある。どの境界を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
