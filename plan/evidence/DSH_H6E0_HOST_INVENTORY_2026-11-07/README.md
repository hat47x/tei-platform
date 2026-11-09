# DSH外部HarnessのHost棚卸し：Tool Processから見たOS・環境・到達性（2026-11-07、#2515 H6-E0）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E0`（Host棚卸し）。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 11 Pro `10.0.26100`（AMD64）、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`、PowerShell `5.1.26100.8875`
- 状態: **1 Hostでの観測**。別Host・Linux比較、実Credentialの露出試験、実Egress制御は**NOT RUN**。

`H6-E0`は「実起動Harness/Profile、Tool/Process/OS principal、cwd、mount、環境変数、MCP・ネットワーク出口、Build/CLI artifactを列挙し、effective configと実OS/Tool権限を分離する」ことを求める。本記録は、**Tool Processの中から見えた範囲**を列挙したものである。config file上の宣言ではなく、実際に動いたProcessの観測である。

## 1. 何をしたか

Mock Endpointから`pwsh` Toolを6回強制し、Harnessに次の読み取り専用コマンドを実行させた。同時に、親Processへ合成sentinel環境変数（`H6E0_SECRET_SENTINEL`）とダミーCredentialを設定し、**Tool Processへ継承されるか**を確かめた。

1. PowerShell版数・Windows identity・OS名/版/アーキテクチャ
2. cwd・`DSH_SESSION_ID`・`DSH_PROFILE`・`DSH_PERMISSION_MODE`
3. 環境変数の件数、sentinelの有無、`DEEPSEEK_API_KEY`の有無
4. 論理ドライブ（mount相当）
5. `127.0.0.1`の開いているPort（8799、Mock）と閉じているPort（8798）への到達性
6. 外向き名前解決（`example.invalid`）

同じ6コマンドを**`read-only`**と**`danger-full-access`**の両方で実行し、比較した。加えて別Sessionで、Credential様の環境変数名（`DEEPSEEK_API_KEY`・`OPENROUTER_API_KEY`・`OPEN_DESIGN_API_KEY`・`OPENCODE_GO_API_KEY`・`MOCK_API_KEY`）の有無と、`KEY|TOKEN|SECRET|PASSWORD|CREDENTIAL`に一致する変数の件数を数えた。

## 2. 観測した結果

### 2.1 Processの中身

| 観測 | read-only | danger-full-access |
| --- | --- | --- |
| PowerShell | `5.1.26100.8875` | 同左 |
| Windows identity | **取得できず**（`[System.Security.Principal.WindowsIdentity]`がConstrainedLanguageで失敗） | `HAT47X\hat47x` |
| OS | **取得できず**（`Get-CimInstance`が失敗） | `Microsoft Windows 11 Pro`／`10.0.26100`／64-bit |
| 論理ドライブ | **取得できず** | `C:` ほか（`DeviceID`・`DriveType`・`Size`） |
| cwd | `D:\GIT\hat47x` | 同左 |
| DSHが渡す変数 | `DSH_SESSION_ID`・`DSH_PROFILE=headless`・`DSH_HOME`・`DSH_PROFILE_DIR`・`DSH_SHELL` | 同左 |
| `DSH_PERMISSION_MODE` | 空（設定されていない） | 空 |

`read-only`では**PowerShellがConstrainedLanguageで動く**ため、.NETの型生成（`[System.Text.UTF8Encoding]::new(...)`）や`Get-CimInstance`が失敗する。これは**Sandboxの制限が実際に効いている観測**であり、単なる「modeの表示」ではない。

### 2.2 環境変数

| 観測 | read-only | danger-full-access |
| --- | --- | --- |
| 環境変数の件数 | 53 | 53 |
| 親で設定したsentinel（`H6E0_SECRET_SENTINEL`） | **無し** | **無し** |
| `DEEPSEEK_API_KEY` | **無し** | **無し** |
| `OPENROUTER_API_KEY`／`OPEN_DESIGN_API_KEY`／`OPENCODE_GO_API_KEY`／`MOCK_API_KEY` | — | **すべて無し** |
| `KEY\|TOKEN\|SECRET\|PASSWORD\|CREDENTIAL`に一致する変数 | — | **0件** |
| `PATH`／`APPDATA` | — | あり |
| `HOME` | — | 無し |
| 親Processの環境変数 | 54件（sentinelとKEYを含む） | 54件（同） |

**親Processに設定した環境変数は、Tool Processへ継承されなかった。** 親では54件・sentinelあり・`DEEPSEEK_API_KEY`ありだったものが、子では53件・sentinelなし・KEYなしである。DSHが自前で渡す`DSH_*`の5つだけが残った。

これは**Credentialを環境変数で渡してもToolには見えない**という観測で、機密Profileの設計に効く。ただし、親Processの環境変数をDSHがどう選別しているかの実装は本記録では確認していない。

### 2.3 到達性

| 観測 | read-only | danger-full-access |
| --- | --- | --- |
| `127.0.0.1:8799`（Mockが待受け） | 到達可 | 到達可 |
| `127.0.0.1:8798`（待受けなし） | 到達不可 | 到達不可 |
| `example.invalid`の名前解決 | 実行できず（ConstrainedLanguage） | `DNS name does not exist` |

Tool Processは**loopbackのPortへ到達できる**。`H6-E1`で観測した「cwd外のファイルも読める」と合わせると、Tool Processの能力はSandbox modeの表示より広い。

### 2.4 送出

6コマンドの入出力を含む8〜10要求について、合成sentinelの**値**が要求bodyへ現れた回数は**0**だった。sentinelはTool Processに存在しなかったため、送出されようがない。

## 3. この記録が示すこと

- `H6-E0`の「effective configと実OS/Tool権限を分離」に対して、**configの表示（`DSH_PERMISSION_MODE`が空）と実効制限（ConstrainedLanguage）が一致しない**例を示した。モード名や環境変数だけを根拠に権限を判断できない。
- **親の環境変数（Credentialを含む）はTool Processへ継承されない**。環境変数経由のCredential漏出は、この構成では観測されなかった。
- `read-only`でも**loopback Portへ到達でき、cwd外も読める**（前者は本記録、後者は[H6-E1の記録](../DSH_H6E1_READ_BOUNDARY_2026-11-07/README.md)）。
- `danger-full-access`ではOS identity・ドライブ・DNSが取得でき、制限が外れていることを実測で確認した。

## 4. この記録が示さないこと

- 実Credentialの値が漏れないこと（sentinelとダミー値のみ。実Credentialの値は設定していない）。
- Windows identityの実SID、ACL、AppContainer、別User Context、mount名前空間。
- MCP Server、Subagent、Job、Web Fetch等**他のTool経路**の能力。本記録は`pwsh`のみ。
- 外向きNetwork（loopback以外）の到達性と拒否。`example.invalid`は存在しない名前であり、実Hostへの到達は試していない。
- Build/CLI artifactの列挙、別Host（Linux）との比較。
- 実Providerへの送出、Egress制御、`H6-E0`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `inventory.json` | 両modeの6コマンドstdout、ParsedFacts、要求ごとのbyte数とSHA-256、環境変数Probeの結果 |

生の要求bodyはコミットしていない。**sentinelは合成であり、実secretではない**。

再現する場合の要点:

```sh
python mock_provider_cmd_seq.py 8799 <record-dir> <commands.json>
H6E0_SECRET_SENTINEL=<synthetic> DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode <read-only|danger-full-access> --approval-policy never --json "Inventory."
```

## 6. `H6-E0`への含意（判断はしない）

`H6-E0`が必要とする「未知の入口を不存在にしない」列挙に対して、本記録は**Tool Processから見える範囲**を確定させた。一方で、`H6-E1`の拒否不成立と合わせると、**Sandbox modeの宣言だけでは機密Profileを成立させられない**。どの境界を採るかは#2515とTAI #232のOwner判断であり、本記録は判定しない。
