// Read-only summary of a TEI view for the Studio screen.
// The formatter is pure so it can be tested without a DOM.

function valueOf(value) {
  return value === null || value === undefined ? "" : String(value);
}

function joined(values) {
  return Array.isArray(values) ? values.map(valueOf).join(", ") : "";
}

function commandRows(command) {
  return [
    command.label || command.capability || "",
    command.capability || "",
    joined(command.inputs),
    joined(command.outcomes),
    command.effect ? `${command.effect.kind} ${command.effect.data}` : "",
  ];
}

function summarizeRecord(view) {
  const sections = [{
    title: "基本",
    rows: [
      ["pattern", valueOf(view.pattern)],
      ["kind", valueOf(view.kind)],
      ["data", valueOf(view.data)],
      ["semantics", valueOf(view.semanticFingerprint)],
      ["print-template", valueOf(view.printTemplate)],
    ],
  }];
  sections.push({
    title: "項目",
    header: ["表示名", "型", "入力", "必須", "更新禁止", "役割", "値域"],
    rows: (view.items || []).map((item) => [
      item.label || item.attribute,
      valueOf(item.type),
      valueOf(item.input),
      item.required ? "必須" : "",
      item.immutable ? "更新禁止" : "",
      valueOf(item.role),
      joined(item.values),
    ]),
  });
  if (Array.isArray(view.commands) && view.commands.length) {
    sections.push({
      title: "コマンド",
      header: ["ラベル", "能力", "入力", "結末", "効果"],
      rows: view.commands.map(commandRows),
    });
  }
  if (Array.isArray(view.hooks) && view.hooks.length) {
    sections.push({
      title: "フック",
      header: ["位相", "範囲", "読み取り", "使用", "生成", "失敗時"],
      rows: view.hooks.map((hook) => [
        valueOf(hook.phase),
        valueOf(hook.scope),
        joined(hook.reads),
        joined(hook.uses),
        joined(hook.produces),
        valueOf(hook.failure),
      ]),
    });
  }
  if (Array.isArray(view.unused) && view.unused.length) {
    sections.push({
      title: "未使用の属性",
      header: ["属性", "表示名", "型"],
      rows: view.unused.map((item) => [valueOf(item.attribute), valueOf(item.label), valueOf(item.type)]),
    });
  }
  return sections;
}

function summarizeBrowse(view) {
  const sections = [{
    title: "一覧",
    rows: [
      ["pattern", valueOf(view.pattern)],
      ["data", valueOf(view.data)],
      ["query", valueOf(view.query)],
      ["page-size", valueOf(view.pageSize)],
      ["semantics", valueOf(view.semanticFingerprint)],
    ],
  }];
  sections.push({
    title: "列",
    header: ["表示名", "属性", "型"],
    rows: (view.columns || []).map((column) => [column.label || column.attribute, valueOf(column.attribute), valueOf(column.type)]),
  });
  if (Array.isArray(view.filters) && view.filters.length) {
    sections.push({
      title: "絞り込み",
      header: ["表示名", "演算子", "パラメータ"],
      rows: view.filters.map((filter) => [filter.label || filter.attribute, joined(filter.operators), valueOf(filter.parameter)]),
    });
  }
  if (Array.isArray(view.sorts) && view.sorts.length) {
    sections.push({
      title: "並び順",
      header: ["表示名", "方向"],
      rows: view.sorts.map((sort) => [sort.label || sort.attribute, valueOf(sort.direction)]),
    });
  }
  return sections;
}

function summarizeHeaderDetail(view) {
  const sections = [{
    title: "見出し明細",
    rows: [
      ["pattern", valueOf(view.pattern)],
      ["relation", valueOf(view.relation)],
      ["header", valueOf(view.header)],
      ["detail", valueOf(view.detail)],
      ["semantics", valueOf(view.semanticFingerprint)],
    ],
  }];
  sections.push({
    title: "明細列",
    header: ["表示名", "属性", "型"],
    rows: (view.detailColumns || []).map((column) => [column.label || column.attribute, valueOf(column.attribute), valueOf(column.type)]),
  });
  if (Array.isArray(view.aggregates) && view.aggregates.length) {
    sections.push({
      title: "集計",
      header: ["名前", "関数", "属性"],
      rows: view.aggregates.map((aggregate) => [valueOf(aggregate.name), valueOf(aggregate.function), aggregate.label || valueOf(aggregate.attribute)]),
    });
  }
  return sections;
}

function summarizeApplication(view) {
  const sections = [{
    title: "Application",
    rows: [
      ["application", valueOf(view.application)],
      ["semantics", valueOf(view.semanticFingerprint)],
    ],
  }];
  sections.push({
    title: "Pattern",
    header: ["ID", "種別"],
    rows: (view.patterns || []).map((pattern) => [valueOf(pattern.id), valueOf(pattern.kind)]),
  });
  if (Array.isArray(view.packages) && view.packages.length) {
    sections.push({
      title: "Package",
      header: ["ID", "版", "束縛"],
      rows: view.packages.map((pkg) => [valueOf(pkg.id), valueOf(pkg.version), pkg.bound ? "束縛済み" : ""]),
    });
  }
  if (Array.isArray(view.standardBusiness) && view.standardBusiness.length) {
    sections.push({
      title: "標準業務",
      header: ["種別", "Data", "属性"],
      rows: view.standardBusiness.map((ref) => [valueOf(ref.kind), valueOf(ref.data), joined(ref.attributes)]),
    });
  }
  if (Array.isArray(view.periods) && view.periods.length) {
    sections.push({
      title: "期間",
      header: ["ID", "Data", "業務日付", "状態", "暦", "締め日"],
      rows: view.periods.map((period) => [valueOf(period.id), valueOf(period.data), valueOf(period.businessDate), joined(period.states), valueOf(period.calendar), closingOf(period)]),
    });
  }
  return sections;
}

// closingOf renders the declared closing day, so a period narrower than a month
// is visible without the reader inferring a default day.
function closingOf(period) {
  const day = period.closingDay;
  if (typeof day !== "number" || day <= 0) {
    return "";
  }
  return `${day}日`;
}

function summarizePackage(view) {
  const sections = [{
    title: "Package",
    rows: [
      ["package", valueOf(view.package)],
      ["version", valueOf(view.version)],
      ["projection", valueOf(view.projection)],
      ["conformance", valueOf(view.conformance)],
      ["bound", view.bound ? "true" : "false"],
    ],
  }];
  sections.push({
    title: "メンバー",
    header: ["種別", "Pattern"],
    rows: (view.members || []).map((member) => [valueOf(member.kind), valueOf(member.pattern)]),
  });
  if (Array.isArray(view.requires) && view.requires.length) {
    sections.push({
      title: "前提",
      header: ["種別", "ID"],
      rows: view.requires.map((requirement) => [valueOf(requirement.kind), valueOf(requirement.id)]),
    });
  }
  return sections;
}

function summarizeCommand(view) {
  const sections = [{
    title: "Command",
    rows: [
      ["capability", valueOf(view.capability)],
      ["label", valueOf(view.label)],
      ["effect", view.effect ? `${view.effect.kind} ${view.effect.data}` : ""],
      ["period", view.period ? valueOf(view.period.id) : ""],
      ["ready", view.ready === undefined ? "" : view.ready ? "true" : "false"],
    ],
  }];
  sections.push({
    title: "入力",
    header: ["名前", "型", "属性", "値", "状態"],
    rows: (view.inputs || []).map((input) => [
      valueOf(input.name),
      valueOf(input.type),
      valueOf(input.attribute),
      valueOf(input.value),
      input.provided === false ? "未提供" : input.provided ? "提供済み" : "",
    ]),
  });
  if (Array.isArray(view.outcomes) && view.outcomes.length) {
    sections.push({
      title: "結末",
      header: ["名前", "型"],
      rows: view.outcomes.map((outcome) => [valueOf(outcome.name), valueOf(outcome.payload)]),
    });
  }
  return sections;
}

// differenceSections lists the operations the declaration did not describe, so
// a reader can see which data the execution reached beyond the declared effect.
// hookSummary renders the hook report a reference execution returned. It
// counts executed, warned, and not-evaluated phases, and names the phases that
// did not run so the summary does not imply a pass.
function hookSummary(hooks) {
  if (!hooks || typeof hooks !== "object") return "not declared";
  const executed = Array.isArray(hooks.executed) ? hooks.executed : [];
  const warned = Array.isArray(hooks.warned) ? hooks.warned : [];
  const notEvaluated = Array.isArray(hooks.notEvaluated) ? hooks.notEvaluated : [];
  const skipped = Array.isArray(hooks.skipped) ? hooks.skipped : [];
  const parts = [`executed ${executed.length}`, `warned ${warned.length}`];
  if (notEvaluated.length) parts.push(`not evaluated ${notEvaluated.join(",")}`);
  if (skipped.length) parts.push(`skipped ${skipped.join(",")}`);
  if (hooks.refused) parts.push(`refused ${hooks.refused}`);
  // A proposal is a candidate value, not an applied one. Naming it here keeps
  // "executed" from reading as "the hook changed nothing".
  const uncalled = hooks.uncalled && typeof hooks.uncalled === "object" ? hooks.uncalled : {};
  for (const phase of Object.keys(uncalled).sort()) {
    const names = Array.isArray(uncalled[phase]) ? uncalled[phase] : [];
    if (names.length) parts.push(`uncalled at ${phase} ${names.join(",")}`);
  }
  const proposed = hooks.produced && typeof hooks.produced === "object" ? hooks.produced : {};
  for (const phase of Object.keys(proposed).sort()) {
    const values = proposed[phase] && typeof proposed[phase] === "object" ? proposed[phase] : {};
    const names = Object.keys(values).sort();
    if (names.length) {
      parts.push(`proposes at ${phase} ${names.map((name) => `${name}=${values[name]}`).join(",")}`);
    }
  }
  return parts.join("; ");
}

function differenceSections(differences) {
  if (!Array.isArray(differences) || !differences.length) return [];
  return [{
    title: "宣言との差",
    header: ["種別", "宣言", "適用"],
    rows: differences.map((difference) => [
      valueOf(difference.code),
      difference.declared ? `${difference.declared.kind} ${difference.declared.data}` : "",
      difference.applied ? `${difference.applied.kind} ${difference.applied.data}` : "",
    ]),
  }];
}

// validationSections shows the diagnostics and their unanswered proposals, so a
// reader sees that a correction exists without mistaking it for an accepted fix.
function validationSections(value) {
  const sections = [];
  if (Array.isArray(value.diagnostics)) {
    sections.push({
      title: "診断",
      header: ["重大度", "符号", "内容"],
      rows: value.diagnostics.map((item) => [
        valueOf(item.diagnostic?.severity),
        valueOf(item.diagnostic?.code),
        valueOf(item.diagnostic?.message),
      ]),
    });
  }
  if (Array.isArray(value.proposals)) {
    sections.push({
      title: "修正案（未受理）",
      header: ["符号", "受理", "版"],
      rows: value.proposals.map((proposal) => [
        valueOf(proposal.code),
        proposal.accepted ? "受理済み" : "未受理",
        valueOf(proposal.revision),
      ]),
    });
  }
  return sections;
}

export function summarizeView(value) {
  if (!value || typeof value !== "object") {
    return [{title: "ビュー", rows: [["値", valueOf(value)]]}];
  }
  if (value.plan) {
    return summarizeCommand(value.plan).concat([{
      title: "参照実行",
      rows: [
        ["mode", valueOf(value.mode)],
        ["authorization", valueOf(value.authorizationMode)],
        ["committed", value.committed ? "true (unit of work)" : "false"],
        ["persisted", value.persisted ? "true" : "false"],
        ["persistence", valueOf(value.persistence)],
        ["durability", valueOf(value.durability)],
        ["outcome", value.outcome ? `${value.outcome.name} -> ${value.outcome.payload}` : ""],
        ["effect", value.effect ? `${value.effect.kind} ${value.effect.data}` : ""],
        ["records", Array.isArray(value.records) ? String(value.records.length) : "0"],
        ["differences", Array.isArray(value.differences) ? String(value.differences.length) : "0"],
        ["difference-check", value.appliedUnobserved ? "not observed" : "checked"],
        ["reported-effect", value.reportedEffectMissing ? "missing" : "reported"],
        // The hook row names what the reference execution did with the pattern's
        // declared hooks. A phase that did not run is listed as not evaluated,
        // so an empty row never reads as "the hooks passed".
        ["hooks", hookSummary(value.hooks)],
      ],
    }]).concat(differenceSections(value.differences));
  }
  if (value.diagnostics !== undefined || value.proposals !== undefined) return validationSections(value);
  if (value.package && value.version !== undefined) return summarizePackage(value);
  if (Array.isArray(value.items)) return summarizeRecord(value);
  if (Array.isArray(value.columns)) return summarizeBrowse(value);
  if (Array.isArray(value.detailColumns)) return summarizeHeaderDetail(value);
  if (Array.isArray(value.patterns)) return summarizeApplication(value);
  if (value.capability) return summarizeCommand(value);
  return [{
    title: "ビュー",
    rows: Object.entries(value).map(([key, entry]) => [key, JSON.stringify(entry)]),
  }];
}












