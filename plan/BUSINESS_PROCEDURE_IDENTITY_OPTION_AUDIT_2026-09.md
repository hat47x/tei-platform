# Business Procedure Identity Option Audit — 2026-09

Status: Design pressure audit / no ADR yet

## 目的

Business Procedureのfile-native accepted sourceをRQ-002 / Source Index / Shared Acceptanceへ接続する際に残ったidentity問題について、次の三案を比較する。

- A. minimal Canonical Procedure identity shell
- B. Subject contractをWorld外accepted design targetへ拡張
- C. Authoring identity registryを別層として正式化

full Canonical Procedure modelは比較の基準外とし、必要性が出るまで採用しない。

## 1. 前段Evidence

すでにReferenceで次を確認した。

### Canonical Procedureなしで成立したもの

- Business Action -> Application Pattern cross-view
- manual / external Business Action
- multiple Procedure -> same Pattern
- Pattern-owned meaning derivation
- cross-view diagnostic
- Reference-local stable Procedure key
- Procedure間handoff
- review history target continuity
- source fingerprintによるstale検出
- file move / logical rename分離
- Preview / Reject / Accept / reloadの機械的安全性

### 未解決として残ったもの

RQ-002の

> authoring sourceから、どのCanonical subjectへ寄与したか

というtrace targetだけである。

現行`subject.Subject`もTEI World内のsemantic / logical targetを指す契約である。

したがって、単にSource Index APIへReference-local keyを入れられることと、RQ-002を満たすことは同じではない。

## 2. 評価基準

各案を少なくとも次で比較する。

1. RQ-002を文言どおり満たす
2. RQ-003の単一正本を保つ
3. RQ-017の二重管理を増やさない
4. RQ-023のone-maintainer性を保つ
5. RQ-025のBusiness Action / Procedure独立性を保つ
6. RQ-026の三視点cross-checkへ利用できる
7. existing Source Indexを再利用できる
8. Shared Acceptance targetとして使える
9. 000039 impact analysisへ伸ばせる
10. file move / logical renameを区別できる
11. concept creepを抑えられる
12. Business Procedure詳細をCoreへ過剰吸収しない

## 3. Option A — minimal Canonical Procedure identity shell

### 概念

Business Procedureのauthoritative fileから、World側へpath-independent identityだけを正規化する。

最小形の候補:

```text
BusinessProcedureIdentity
  id
```

Reference-local sourceがauthorityであり、World identityはloaderが同じsourceから導出する。

人が同じIDを別fileへ二重入力する構成にはしない。

### RQ-002

強い。

```text
Business Procedure source
  -> Canonical BusinessProcedureIdentity
  -> Source Index Subject
```

を既存Source Index原則へ直接接続できる。

### RQ-003 / RQ-017

source fileを唯一のauthoritative inputとし、identity shellをnormalized resultとして生成するなら二重正本にはならない。

World identityからsourceへ逆書きしない。

### Shared Acceptance

candidate targetをCanonical identityとして解決できる。

source locatorはSource Indexへ残し、file moveでidentityを変えない。

### 000039

将来、Business Procedure change / Pattern impactのnodeとして既存Subject方式を利用しやすい。

### 実装波及

identityだけでも少なくとも次を検討する必要がある。

- domain type / package
- `ir.World` field
- normalize / deterministic ordering
- procedures source loader
- identity uniqueness validation
- Source Index Subject / source anchor
- World serialization / Reference goldenへの影響
- Conformance

production Reference Workspaceに`procedures` rootを追加しない限り、existing World outputを変えない形でpressure testは可能である。

### 最大のリスク

identity shellが入口となり、

- steps
- actor
- branch
- timer
- retry
- Workflow state
- authorization role

等をCoreへ吸収し始めること。

採用する場合はidentity-only上限をtest / ADRで明示する必要がある。

## 4. Option B — Subject contract拡張

### 概念

現行Subjectの

```text
inside a TEI World
```

を広げ、World外のaccepted file-native design targetもSubjectにする。

### 利点

- `ir.World`へProcedure fieldを追加しなくてよい
- Reference-local Procedure keyをそのままSource Index targetへ使える
- file move / locator分離は自然

### 問題

RQ-002は`Canonical subject`へのtraceを要求する。

Subjectだけを広げると、

```text
Canonical Subject
Authoring Subject
Evidence Subject
```

の境界が曖昧になりやすい。

「SubjectにあるからCanonical」という逆転を起こす危険がある。

### 影響範囲

コード差分は小さく見えても、Subject / Source Indexの意味契約とRQ-002解釈を広範囲に変える。

conceptual blast radiusはOption Aより大きい可能性がある。

## 5. Option C — Authoring identity registry

### 概念

World Canonical identityとは別に、file-native accepted design targetを扱うformal registryを持つ。

```text
AuthoringTarget
  key
  source locator(s)
  fingerprint
  role
```

Business ProcedureはまずAuthoringTargetとなる。

### 利点

- WorldへProcedureを追加しない
- authoring identity / review / handoffを明示的に扱える
- Source locatorとlogical keyを分離できる

### 問題

RQ-002のCanonical linkは別途必要である。

```text
AuthoringTarget -> ? Canonical Subject
```

を解けなければ、identity問題を一段追加しただけになる。

Source Index / Authoring Registry / Shared Acceptanceの三層を一人で保守することになり、RQ-023上も不利になり得る。

## 6. 比較表

| 基準 | A: identity shell | B: Subject拡張 | C: Authoring registry |
|---|---|---|---|
| RQ-002をそのまま満たす | **強い** | 要求解釈変更が必要 | Canonical linkが別途必要 |
| RQ-003 | source-derivedなら良 | 境界曖昧化に注意 | registryが第二正本化し得る |
| RQ-017 | ID導出なら良 | 良 | mapping管理が増える |
| RQ-023 | 中 | 表面上小、概念影響大 | 三層化で不利 |
| RQ-025 | identity-onlyなら保てる | 保てる | 保てる |
| RQ-026 | Subject nodeとして強い | 表現可能 | mapping経由 |
| Source Index reuse | **直接** | 契約変更して直接 | adapter必要 |
| Shared Acceptance target | **直接** | 非Canonical targetとして再定義 | registry target |
| 000039 reuse | **直接** | graph意味変更が必要 | graph adapter必要 |
| file move耐性 | 強い | 強い | 強い |
| concept creep | 要guard | Subject意味膨張 | identity層膨張 |
| Core詳細吸収リスク | identity-onlyなら抑制可能 | 低 | 低 |

## 7. 現時点の優先順位

### 第一候補: Aをexperiment-onlyでpressure test

理由:

1. RQ-002を再解釈せず満たせる唯一の案
2. Source Index / Shared Acceptance / 000039の既存target modelを再利用できる
3. source fileをauthorityとし、identity shellをderived normalized resultにすればRQ-003 / RQ-017と両立できる
4. identity-only上限を守ればRQ-025が明示する「full top-level Procedure modelは必須ではない」という余白を壊さない

これは採用決定ではない。

### 第二候補: B

Aのidentity shellが実装波及に対して過大であることが実証された場合に比較する。

Subject contract / RQ-002の意味を変えるため、コード行数だけで小さいとは判断しない。

### 第三候補: C

A/Bのどちらでもauthoring identityの実務要件を扱えない場合に検討する。

現時点で新registryを増やす根拠は弱い。

## 8. Option A experimentの厳しい上限

次のpressure testを行う場合、Canonical側へ許すのはidentityだけとする。

```text
allowed
  id

not allowed
  purpose
  trigger
  actor
  step
  instruction
  handoff
  branch
  lifecycle
  state
  timer
  retry
  authorization mapping
  runtime orchestration
```

Business meaning本体はfile-native authoring sourceに残す。

World identity shellはSource Index / target resolution / impact nodeとしてのみ使う。

## 9. Option Aで必ず確認するnegative case

- source fileなしにWorld identityだけを手編集できない
-同一IDを複数Procedure sourceが宣言したらfail closed
- file moveでCanonical identity不変
- logical ID renameはCanonical change
- Business wording editでidentity shell不変
- identity shellからBusiness detailsを再構成しようとしない
- Source Index locator / fingerprintをWorldへ混ぜない
- actorからAuthorization Roleを作らない
- Procedure handoffからruntime Workflowを生成しない

## 10. 判断Gate

Option A pressure testの結果、

```text
small implementation surface
+
RQ-002 exact trace
+
no duplicate authority
+
no Business detail creep
```

が成立する場合だけ、minimal Canonical Procedure identity contractをADR候補にする。

identity-onlyでもCore / loader / Conformanceの波及が大きすぎる場合は、採用せずOption Bとの比較へ戻る。

full Canonical Procedure modelへ直接進まない。
