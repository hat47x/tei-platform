// Presentation-only join of an already disclosed Catalog validation summary.
// Real ModelCatalog.validation comes from operations.CatalogWorkspace (#3728).
// No Source read, validator, Policy, inspection right or Owner acceptance here.
const isText = v => typeof v === "string" && v.length > 0 && v.trim() === v;
const fail = () => { throw new Error("unqualified_catalog_validation"); };
const record = (value, names) => {
  if (!value || typeof value !== "object" || Array.isArray(value)) return false;
  const p = Object.getPrototypeOf(value);
  if (p !== Object.prototype && p !== null) return false;
  return Reflect.ownKeys(value).every(k => {
    if (typeof k !== "string" || !names.includes(k)) return false;
    const d = Object.getOwnPropertyDescriptor(value, k);
    return !!d && Object.hasOwn(d, "value");
  });
};
const count = x => Number.isSafeInteger(x) && x >= 0;

// Caller must supply a CURRENT, qualified public summary separately.
// A successful Catalog read can coexist with validation.ok=false.
export function bindQualifiedCatalogValidation(publicProjection, input) {
  if (!record(publicProjection, ["snapshot", "workspaceObservation", "views",
       "relationTable", "publicDiagnostics", "coverage"]) ||
      !record(publicProjection.coverage, ["scope", "allSourceCompleteness"]) ||
      !isText(publicProjection.coverage.scope) ||
      !isText(publicProjection.snapshot) ||
      !isText(publicProjection.workspaceObservation) ||
      !record(input, ["scope", "workspaceObservation", "snapshot", "catalogReadSucceeded",
        "validation"]) ||
      !isText(input.scope) || !isText(input.snapshot) ||
      !isText(input.workspaceObservation) ||
      input.scope !== publicProjection.coverage.scope ||
      input.snapshot !== publicProjection.snapshot ||
      input.workspaceObservation !== publicProjection.workspaceObservation ||
      input.catalogReadSucceeded !== true ||
      !record(input.validation, ["ok", "errorCount", "warningCount"]) ||
      typeof input.validation.ok !== "boolean" ||
      !count(input.validation.errorCount) || !count(input.validation.warningCount)) fail();
  // Do not infer overall validity from this read or certify individual subjects.
  return {
    scope: input.scope, snapshot: input.snapshot,
    workspaceObservation: input.workspaceObservation,
    catalogReadSucceeded: true,
    validation: {
      ok: input.validation.ok,
      errorCount: input.validation.errorCount,
      warningCount: input.validation.warningCount,
    },
    allSourceCompleteness: "not-proven",
  };
}
