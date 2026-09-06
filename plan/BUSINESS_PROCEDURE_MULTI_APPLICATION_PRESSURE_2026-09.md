# Business Procedure Multi-Application Pressure — 2026-09

Status: Test-only design pressure / production Reference unchanged

## 目的

Business Procedureを現在の`order_entry`一つのApplication containerへ固定せず、複数Application / external / manual workを跨げるBusiness viewpointとして保てるか確認する。

current production Reference WorkspaceにはApplicationが`order_entry`一つしかないため、test内のtemp Workspaceへ第二Applicationを追加してpressure testする。

## 1. なぜ必要か

Business Procedureは社会的な仕事の進め方であり、Application境界と1:1とは限らない。

```text
Business Procedure
  ├─ manual work
  ├─ Application A Pattern
  ├─ external work
  └─ Application B Pattern
```

を許容できる必要がある。

一つのApplication containerをCanonical Procedure identityへ焼き込むと、Business viewpointをsystem boundaryへ還元する危険がある。

## 2. current fixtureの`application`

現行Reference fixtureは

```json
"application": "order_entry"
```

を持つ。

current checkerではこのApplicationの存在を確認する一方、各`applicationPatterns`はqualified identityでWorld全体からexact解決する。

したがって現実装上、Pattern reference自体は一つのApplicationへ閉じていない。

この性質を維持する。

## 3. test-only第二Application

`multi_application_pressure_test.go`ではtemp Workspaceへ最小の第二Applicationを追加する。

```text
application fulfillment_console {
}

pattern fulfillment_console.order_record record {
  data sales.order
}
```

Database / Contract / datasource設定は意図的に持たせない。

このPressureで確認したいのはqualified Application Pattern identityとBusiness Procedureからのcross-Application参照だけであり、Gate BのDatabase realization条件をtestへ混ぜないためである。

既存`application_pattern_loader_test.go`でも空のApplication bodyと独立Pattern fileからApplication Patternをloadできることを確認しているため、その最小形へ合わせる。

production `reference/workspace`は変更しない。

Business Procedureへsynthetic stepを追加する。

```text
BP-XA
  Business Action:
    inspect the same order from a separate fulfillment application context

  Application Pattern:
    fulfillment_console.order_record
```

## 4. 期待する性質

### cross-view resolution

`fulfillment_console.order_record`をexact解決し、Pattern-owned `sales.order`をderivedできる。

### no single-Application Canonical container

thin Canonical Business meaning probeにはtop-level `Application` fieldを置かない。

ApplicationとのrelationはStepのqualified Pattern refs等から明示する。

### no implicit app guessing

`order_record`のようなunqualified名称や文字列類似からApplicationを推測しない。

### unrelated realization independence

second ApplicationのDB / Contract設定がなくてもPattern identity pressureは成立する。

multi-Application Business semanticsの検証をDatabase/provider readinessへ不要に結合しない。

### manual / external coexistence

複数Application Patternを参照できることから、manual / external stepを削除しない。

## 5. Canonical contractへの意味

production contract候補を考える際、次を避ける。

```text
Procedure belongsTo exactly one Application
```

を普遍ルールとして固定すること。

少なくともcurrent pressureでは、

```text
Procedure
  -> zero or more qualified Application Pattern refs
```

を許容する方が自然である。

これはApplicationとDatabaseのN:M性とは別の話であり、Business Procedure viewpointをApplication境界へ還元しないための圧力である。

## 6. fixture `application` fieldの扱い

現時点では削除しない。

Reference checkerのbaseline context / current experiment routingとして利用されているためである。

ただしthin Canonical Business meaningへそのまま昇格させない。

今後、複数Application Referenceが実際に増えたときに、

- artifact context metadata
- primary review context
- semantic relation

のどれなのかを再判定する。

## 7. Verification

`tools/reference-business-procedure-check/multi_application_pressure_test.go`

- minimal second Applicationをtemp Workspaceへ追加
- one Procedureからsecond Application Patternを参照
- cross-view Pattern meaningをexact導出
- thin Canonical meaningにsingle `Application` fieldがないことをguard
- qualified Pattern identityをそのまま保持
- unrelated DB / Contract realizationへ依存しない

actual Repository runnerがGo testへ到達するまではtest-only specificationである。

## 8. 次のPressure

multi-Applicationが成立した後に確認すべきなのは、Business Procedure自体をApplication release lifecycleへ従属させるかである。

現時点では従属させない。

Business Procedure changeとApplication semantic changeは別々に起こり得るため、release / impact relationは実Referenceが要求した時点で検証する。
