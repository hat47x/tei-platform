# DSH外部Harnessが暗黙に取り込むContextの実測（2026-11-07、#2515 H6-E3）

- 対象Issue: [#2515](https://github.com/hat47x/tei-platform/issues/2515)の`H6-E3`（暗黙Context／Tool指示）、[#2509](https://github.com/hat47x/tei-platform/issues/2509)。機密境界の正本は[TAI #232](https://github.com/hat47x/tai-pleroma/issues/232)。
- 実行日: 2026-11-07（Asia/Tokyo）
- 実行環境: Windows 10.0.26100、DSH `0.2.2-rc.1`（`dsh --profile headless`）、Node `24.21.0`
- 状態: **合成canaryのみ**。実secret・実指示文は使っていない。Tool poison・実OS拒否・実Provider送出は**NOT RUN**。

この記録は、外部Harnessが**利用者の指示とは別に、どのファイルの内容を自分から取り込むか**を合成canaryで確かめたものである。`H6-E3`は「明示Context Aとは別にAGENTS/CLAUDE/Skill候補Uが入ること」「Accessible／Read／Delivered-to-CLIを区別すること」を求めている。本計測はその「Delivered」側を、wire上の要求bodyで観測した。

## 1. 何をしたか

cwdに、すべて合成のcanaryを置いた。

| file | canary | 役割 |
| --- | --- | --- |
| `AGENTS.md` | `AGENTS_CANARY_IMPLICIT_101` | リポジトリ指示 |
| `CLAUDE.md` | `CLAUDE_CANARY_IMPLICIT_102` | リポジトリ指示 |
| `.agents/skills/canary-skill/SKILL.md` | `SKILL_DESC_CANARY_IMPLICIT_104`（frontmatterの`description`）／`SKILL_BODY_CANARY_IMPLICIT_103`（本文） | Skill候補 |
| `allowed/projection.txt` | `PROJECTION_CANARY_ALLOWED_105` | 明示的に読ませる投影 |

Mock Endpointから`read`を1回だけ強制し、Harnessに投影だけを読ませた。**AGENTS.md・CLAUDE.md・SKILL.mdを読むTool呼出しは行っていない**。

## 2. 観測した結果

| 要求 | モデル入力に現れたcanary | Session Logに現れたcanary |
| --- | --- | --- |
| Agent要求（tools 24件） | `AGENTS.md` 1回、`CLAUDE.md` 1回、**Skill description 1回** | `AGENTS.md` 1回、`CLAUDE.md` 1回、Skill description 2回 |
| 題名生成（tools 0件） | なし | `AGENTS.md` 1回、`CLAUDE.md` 1回、Skill description 2回 |
| Tool結果を返す要求 | 上記＋`projection` 1回 | `projection` 2回 |

読み取った内容:

- **`AGENTS.md`と`CLAUDE.md`は、Harnessが自分から読んでモデル入力の`user`メッセージに載せていた**。Tool呼出しの記録は無い。`agents_md`と`claude_md`のcanaryは、いずれも1つのuserメッセージ（`source.kind = agent-instructions`）の中に現れた。
- **Skillの`description`（frontmatter）も、Skill本体をロードしなくてもモデル入力に載っていた**。Skill本文（`SKILL_BODY_CANARY_IMPLICIT_103`）は載っていない。つまり**Skillの一覧提示は「説明文の注入」であり、本文の読込みとは別**である。
- `AGENTS.md`・`CLAUDE.md`・Skill descriptionは、**Session Logにも同じだけ載り、外部へ出る**。題名生成のような小さい要求にも再送される。
- 明示的に読ませた投影だけが`tool_result`として現れ、その内容はモデル入力とSession Logの両方に載った。

## 3. この記録が示すこと

- `H6-E3`の「明示Context Aとは別にAGENTS/CLAUDE/Skill候補Uが入る」は、**この構成で成立する**。利用者が指定していないファイルの内容が、Harnessの判断でモデル入力へ入る。
- 「Skillを置いただけではLoadではない」という既存の区別は、**`description`の注入**と**本文の実Load**を分ける必要があることを示す。本計測では本文は載らず、`description`だけが載った。
- 暗黙Contextはモデル入力だけでなく**Session Logにも載る**ため、指示ファイルに秘密や非公開の運用情報を書くと、外部への送出量が増える。

## 4. この記録が示さないこと

- `AGENTS.md`・`CLAUDE.md`が**どう解決されたか**（親ディレクトリの探索範囲、優先順位、同一内容の重複排除）。本計測はcwd直下の1階層のみ。
- 入れ子ディレクトリの`AGENTS.md`、ユーザー側のグローバル設定、`--add-dir`で追加したrootの扱い。
- **Tool poison**（未信頼のTool結果が指示として働くか）、`H6-E3`の`Accessible`／`Read`／`Delivered-to-Service`の完全な区別。
- 実OS/ACLでの拒否、Network境界、実Providerへの送出、Egress制御。
- `H6-E3`の合否、機密Profile適合、Owner受理、#1119のGate。

## 5. 生データ

| ファイル | 内容 |
| --- | --- |
| `observations.json` | canaryの定義、3要求それぞれのモデル入力／Session Logでの出現、canaryを含むメッセージの位置（`canariesByMessage`） |

生の要求bodyはコミットしていない。**canaryは合成であり、実secretでも実指示でもない**。

再現する場合の要点:

```sh
python mock_provider_read_seq.py 8799 <record-dir> <paths.json>   # 投影だけを read させる
DEEPSEEK_API_KEY=dummy DEEPSEEK_BASE_URL=http://127.0.0.1:8799 \
  dsh --profile headless --permission-mode read-only --approval-policy never --json "Investigate."
```

cwdに`AGENTS.md`・`CLAUDE.md`・`.agents/skills/<name>/SKILL.md`を置き、そのどれもToolで読まない点が要点である。

## 6. `H6-E3`への含意（判断はしない）

`H6-E3`の「Accessible／Read／Delivered-to-CLIを区別」に対して、本計測は**ReadしていないのにDeliveredされている**経路を示した。指示ファイルとSkill一覧は、Harnessが**利用者の指定なしに**読み取ってモデル入力とSession Logへ載せる。機密Profileの設計では、この2経路を「利用者が読ませたもの」と同じ扱いにしない必要がある。採否は#2515とTAI #232のOwner判断であり、本記録は判定しない。
