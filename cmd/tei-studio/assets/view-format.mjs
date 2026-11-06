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
      header: ["位相", "読み取り", "使用", "生成", "失敗時"],
      rows: view.hooks.map((hook) => [
        valueOf(hook.phase),
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
  if (Array.isArray(view.periods) && view.periods.length) {
    sections.push({
      title: "期間",
      header: ["ID", "Data", "業務日付", "状態"],
      rows: view.periods.map((period) => [valueOf(period.id), valueOf(period.data), valueOf(period.businessDate), joined(period.states)]),
    });
  }
  return sections;
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

export function summarizeView(value) {
  if (!value || typeof value !== "object") {
    return [{title: "ビュー", rows: [["値", valueOf(value)]]}];
  }
  if (value.plan) {
    return summarizeCommand(value.plan).concat([{
      title: "参照実行",
      rows: [
        ["mode", valueOf(value.mode)],
        ["committed", value.committed ? "true" : "false"],
        ["outcome", value.outcome ? `${value.outcome.name} -> ${value.outcome.payload}` : ""],
        ["effect", value.effect ? `${value.effect.kind} ${value.effect.data}` : ""],
        ["records", Array.isArray(value.records) ? String(value.records.length) : "0"],
      ],
    }]);
  }
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


