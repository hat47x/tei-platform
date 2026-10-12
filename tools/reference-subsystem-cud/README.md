# Subsystem CUD / Procedure cross-view pressure (Reference-only)

This is an **experimental read-only diagnostic**, scoped to Issue #1132 and the parent #1119. It consumes the semantic Data and Application Pattern identities from a caller-supplied `tei world` JSON snapshot, plus **explicit candidate** ownership, Function-level declared I/O, and Procedure-step facts. The latter facts are never accepted Canonical meaning merely because the probe reads them.

Run it with:

```sh
go run ./tools/reference-subsystem-cud -world /path/to/world.json -facts /path/to/facts.json
```

`facts.json` must use `schemaVersion: "subsystem-cud-candidate-v1"` and contain:

- `owners`: `{data, subsystem, source}` tuples.
- `functionIO`: `{pattern, subsystem, data, operation, source}`; operation is exactly one of C/R/U/D.
- `procedureSteps`: `{id, subsystem, realization, pattern?, source}` with realization `system`, `manual`, or `external`.

`source` is a local, non-secret **evidence locator supplied by the caller**, not proof that the declaration is Accepted. The tool never reads database rows, invokes a Capability, rewrites Source, or makes authorization and transactional claims. **A clean report does not prove business ownership correctness or actual I/O effects.** Unknown references, duplicated ownership, unknown operation codes, missing manual/system bindings, empty candidate fact sets, and mismatched schema stay unresolved. The report includes explicit counts for the World identities and supplied candidate facts; these counts do not establish semantic coverage of the application. A declared cross-subsystem C/U/D prompts review; R remains possible without an ownership violation. Delegation is modeled as a distinct owner-side function rather than a self-asserted bypass flag.

This reference intentionally avoids a second Canonical model or an automatic adapter that would promote legacy observations to project truth. Next integration step: use the existing Source/Acceptance mechanisms to derive an exact, provenance-bound and accepted set of ownership and I/O facts from TEI, and combine the report with #1120 typed references and #1124 Studio diagnostics. Until then the result is **candidate inspection** only.

For local checks, `go test ./tools/reference-subsystem-cud` and `go vet ./tools/reference-subsystem-cud` should run without external Go libraries. Repository-wide tests and trusted `tools/repository-health-check.sh` must still be run against the exact PR head before merging.

## Exact-input accounting and non-inference (E0 hardening)

- The CLI attaches `inputFingerprints.worldJSON` and `inputFingerprints.factsJSON` (SHA-256 of **exact supplied bytes**) to its JSON report. This identifies an input snapshot for downstream review, but does **not** verify that the World was regenerated from current accepted Source or that candidate facts have been accepted.
- If a declared Function I/O points to a missing or ambiguous World Pattern, the tool reports `IO_PATTERN_UNRESOLVED` and **does not** infer `CROSS_SUBSYSTEM_WRITE_INTENT` from that entry. Data identity and owner must also be exact before write-boundary review; unknown values are not replaced with an arbitrary match.
- The source JSON has real `semantics[].data[].id` and `applications[].patterns[].id` arrays. This tool intentionally reads only those known fields; it does not derive owner, actual effect, or Function I/O from a Pattern name, `kind`, or `capability`.
- The typed Data/Relation/Pattern edge provider is owned by #1120 (Draft PR #1139, `modelref/derivedbridge`). When it is qualified, #1132 should consume those read-only edges by exact scoped identity rather than make another parallel graph representation. Accepted Subsystem ownership and explicitly scoped declared I/O still require a separate verified source/acceptance contract.
- A complete Source→World→typed edge→CUD diagnostic→Studio navigation path has **not** been run; the local isolated unit tests only qualify this tool's candidate logic.

## Stale-world rejection (E0 snapshot binding)

`facts.json` additionally **must** include `worldJSONFingerprint`, e.g. `"sha256:b21ed42b17ca10937dcf7020d27fce1546a389327375f2ceb958b0a09937d51c"` for the exact bytes of `testdata/world.json`. To prepare a candidate for your own snapshot, compute `sha256sum /path/to/world.json` and prefix the 64 hexadecimal characters with `sha256:`. This binding is deliberately byte-exact: reformatting JSON requires updating the candidate evidence.

- **Matching supplied World bytes**: the CLI runs the existing read-only candidate checks, recording both input digests in the report.
- **Missing fingerprint**: `WORLD_SNAPSHOT_UNBOUND` (`unresolved`, nonzero exit).
- **Changed World bytes**: `WORLD_SNAPSHOT_STALE` (`unresolved`, nonzero exit), even if the edited JSON still describes the same identities.
- **Malformed World/facts JSON**: input decoding fails before inspection. The CLI does not accept an unrelated pre-decoded `World` beside the digest bytes; `InspectSnapshot` decodes and inspects exactly the supplied bytes in one function. A pure `Inspect(w,f)` remains available for candidate-only unit tests but is **not** a snapshot-integrity check.

**This is consistency, not authenticity.** The same caller controls both `facts.json` and `world.json`, so a newly computed matching digest cannot establish that either comes from an accepted TEI Workspace. It is not a signature, Source Index conformance, Shared Acceptance authorization, or proof of Function-level/actual effects. #1120 should provide the shared typed Source/Graph evidence; #1132 still needs an accepted ownership/declaration producer, exact Workspace revision binding, and a real integration test before qualification.

## 同じ機能に複数の担当Subsystemが宣言された場合

候補の`functionIO`で、同じApplication Patternに異なるSubsystemが割り当てられている場合、`IO_PATTERN_SUBSYSTEM_CONFLICT`を未解決として報告します。対象Patternに関する越境更新の診断は、この矛盾が解消されるまで確定しません。読み取り（R）と更新（C/U/D）が混在する場合や、操作対象のDataが異なる場合も、Patternの実行主体については同じ確認が必要です。

一方、一つのSubsystemが同じPatternから複数のDataを扱うことや、他SubsystemのDataを読み取ることだけでは、担当の矛盾とはみなしません。ここで調べるのはあくまで呼出元が記述した候補同士の整合性であり、実際の業務担当や更新権限を認定するものではありません。

## 業務手順資料の版を候補へ結び付ける（任意）

候補の `procedureSteps` が、Worldとは別に保存された業務手順資料に由来するときは、Worldの指紋だけでは鮮度を確認できません。業務手順の本文だけが変わる場合にも古い候補を検出できるよう、`-procedure` を追加しました。

```sh
go run ./tools/reference-subsystem-cud \
  -world reference/world.json \
  -procedure reference/business-system-design/order-entry-business-procedure.json \
  -facts /path/to/facts.json
```

使用するときは、候補の `facts.json` に既存の `worldJSONFingerprint` とともに `procedureJSONFingerprint` を設定します。値は対象ファイルの**正確なバイト列**に対する `sha256:<64桁の16進数>` です。`sha256sum` などで求められます。

| 状態 | 診断 |
| --- | --- |
| 指紋と入力資料が一致 | 既存の候補検査を続行。出力の `inputFingerprints.procedureJSON` に指紋を記録 |
| `-procedure` に資料を渡したが候補に指紋がない | `PROCEDURE_SNAPSHOT_UNBOUND`（未解決） |
| 資料のバイト列が候補作成時から変更された | `PROCEDURE_SNAPSHOT_STALE`（未解決） |
| 候補には手順資料の指紋があるが、資料を指定しない | `PROCEDURE_SNAPSHOT_UNAVAILABLE`（未解決） |
| 指定した資料がJSONオブジェクトではない | 入力エラー |

従来のWorldだけを扱う呼出しは維持し、手順資料を結び付けた候補に限って追加の版照合を要求します。この機能は**入力バイト列の一致を確認するだけ**です。業務手順の意味や手順とFunctionの対応関係は既存の業務手順検査器の担当であり、この検査器では再解釈しません。ファイルの出所、Shared Acceptanceによる正式採用、Subsystemの業務上の責任、実際のCUD操作の存在も証明しません。

回帰試験では現行の受注業務手順資料から `BP-04`（手作業）と `BP-07`（明示されたApplication Pattern）を**試験内の候補**として参照し、手作業から機能や更新操作を推測しないこと、およびWorldが同一でも手順資料変更を検知することを確認します。


## 業務手順の記述と候補の参照を照合する

`-procedure`を指定したとき、業務手順資料の指紋が一致するだけでは不十分です。候補が別の手順番号や機能の参照を記述していれば、その候補の検査結果を成功とは扱えません。

現時点では、未承認の参照資料`business-procedure-reference-v0`（`reference-authoring-fixture`）について、**候補に明示した手順だけ**を次のように照合します。

- 候補の手順番号が資料に一意に存在すること。欠落は`PROCEDURE_STEP_NOT_FOUND`、重複は`PROCEDURE_SOURCE_STEP_AMBIGUOUS`とします。
- `manual-unresolved`の手順を`system`へ変更したり、資料にない機能を付与したりしないこと。
- `application-use`では、資料が明示する機能参照が一つであるときだけ、候補の`system`と同一参照の組を照合します。機能参照の違いは`PROCEDURE_PATTERN_MISMATCH`、複数参照など単一候補へ安全に対応付けられない場合は`PROCEDURE_PATTERN_CARDINALITY_UNSUPPORTED`とします。
- 検証、認可、観測、手作業と外部作業の境界など、現在の候補形式に安全な対応がない手順は`PROCEDURE_REALIZATION_UNSUPPORTED`とします。無理に機能へ変換しません。
- 対応外の資料形式、正式な正本であることを自己宣言した資料、手順配列がない資料は`PROCEDURE_SOURCE_UNSUPPORTED`とします。資料が存在するのに候補手順が空の場合も未解決です。

すべて未解決の診断であり、従来の入力JSONの指紋不一致やCUD候補の診断も引き続き有効です。これらの照合が成功しても、**受理済みの業務手順・Subsystem所有者・実行時更新・業務規則の網羅性**を証明したことにはなりません。正本採用やSource Indexの照合は既存の担当機構で行い、この参照用検査器に第二の承認機構を設けません。


## 既存の出所索引への参照を確認する（任意）

`-procedure`で資料を指定した場合、さらに既存の出所索引（Source Index）を照合できます。この検査器は新しい索引や受理台帳を作成せず、`internal/sourceindex`の既存の指紋・位置検証を使います。

```sh
go run ./tools/reference-subsystem-cud \
  -world /path/to/world.json \
  -procedure /path/to/workspace/procedures/order-handling.json \
  -facts /path/to/facts.json \
  -source-index /path/to/source-index.json \
  -source-root /path/to/workspace \
  -procedure-source-path procedures/order-handling.json
```

三つの索引用引数は、手順資料の指定とともに一式で渡してください。出所索引の作成・正式な受理は既存のWorkspace／Shared Acceptanceの担当です。呼び出し側は**同一Workspaceの版から取得した**索引と読取り用資料を用意する必要があります。

検査では、索引の全資料の内容指紋と参照位置を既存の検証器で確認し、手順資料の正確なバイト列、Workspace相対パス、`business-procedure-authoring`の手順ID、`authoring`役割の一意な参照を照合します。資料の取り違え、古い索引、不明な参照、重複や競合する定義はすべて未解決として扱います。成功時には`indexedProcedure`に**出所位置の候補**を返し、その情報に権限がない旨を明記します。索引を読まない従来の呼出しは引き続き利用できます。

**出所を探せることと、業務定義が受理されたことは異なります。** 出所索引や読取り関数自体も呼び出し元が提供するため、索引と資料が相互に一致しても、Shared Acceptanceの正本、業務上の責任、機能のCUD権限、実行結果や独立観測は確定しません。正式な受理の確認は#1163と既存の受理機構に委ねます。
