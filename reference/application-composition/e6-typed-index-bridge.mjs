// #1016 E6: join the already inspected typed-World (E5) report to the existing
// Source Index change probe (E4). Diagnostic only; no new dependency resolver.
import {assessIndexedSubjects} from './e4-index-scope.mjs';

const object = x => x !== null && typeof x === 'object' && !Array.isArray(x);
const subjectKey = s => JSON.stringify([s.kind,s.id]);
const targetKinds = Object.freeze({
  application:'application',
  'uses-semantic':'semantic',
  'application-pattern':'application-pattern',
  'pattern-data':'semantic-data',
  'pattern-relation':'semantic-relation',
  'pattern-capability':'semantic-capability',
  'pattern-query':'query',
  'pattern-contract-query':'database-contract-query',
  'uses-validation':'semantic-validation',
  'database-target':'database',
  'requires-contract':'database-contract',
});
const limited = 'application-declaration-and-direct-source-definitions-only';

function inspectReport(report, index) {
  if (!object(report) || typeof report.application !== 'string' || !report.application ||
      report.status !== 'observed' ||
      report.qualification !== 'declaration-and-source-index-only' ||
      !Array.isArray(report.issues) || report.issues.length !== 0 ||
      !Array.isArray(report.dependencies) || !report.dependencies.length ||
      !object(index) || !Array.isArray(index.artifacts) ||
      !Array.isArray(index.references)) return {reason:'incomplete_or_untrusted_report'};

  const subjects = new Map();
  let applicationCount = 0;
  for (const dep of report.dependencies) {
    if (!object(dep) || !object(dep.subject) ||
        targetKinds[dep.relation] !== dep.subject.kind ||
        typeof dep.subject.id !== 'string' || !dep.subject.id.trim() ||
        !object(dep.source) || typeof dep.source.path !== 'string' ||
        !/^sha256:[a-f0-9]{64}$/.test(dep.source.fingerprint)) {
      return {reason:'invalid_dependency_claim'};
    }
    const key = subjectKey(dep.subject);
    if (subjects.has(key)) return {reason:'duplicate_dependency_claim'};
    subjects.set(key, dep.subject);
    if (dep.relation === 'application') {
      if (dep.subject.id !== report.application) return {reason:'wrong_application_identity'};
      applicationCount++;
    }

    // E5's source locator must agree with E4's exact Index for both snapshots.
    const defs = index.references.filter(ref => object(ref) &&
      object(ref.subject) && subjectKey(ref.subject) === key &&
      ref.role === 'definition');
    const artifacts = index.artifacts.filter(a => object(a) && a.path === dep.source.path);
    if (defs.length !== 1 || defs[0].artifactPath !== dep.source.path ||
        artifacts.length !== 1 || artifacts[0].fingerprint !== dep.source.fingerprint) {
      return {reason:'report_index_source_mismatch'};
    }
  }
  if (applicationCount !== 1) return {reason:'application_definition_not_unique'};
  return {subjects};
}

export function assessParsedApplicationSourceDrift({
  baselineReport, currentReport, baselineIndex, currentIndex,
} = {}) {
  const base = inspectReport(baselineReport,baselineIndex);
  const current = inspectReport(currentReport,currentIndex);
  const context = {qualification:limited, fullClosure:false, authority:'none'};
  if (base.reason || current.reason) {
    return {...context,status:'unresolved',reason:base.reason || current.reason};
  }
  if (baselineReport.application !== currentReport.application) {
    return {...context,status:'unresolved',reason:'application_selection_changed'};
  }
  const oldSubjects = [...base.subjects.keys()].sort();
  const newSubjects = [...current.subjects.keys()].sort();
  if (JSON.stringify(oldSubjects) !== JSON.stringify(newSubjects)) {
    return {...context,status:'unresolved',reason:'declared_dependency_set_changed'};
  }
  // E4 remains the owner of Source Index role/path/fingerprint comparisons.
  // No further source resolver or lock is introduced.
  let scope;
  try {
    scope = assessIndexedSubjects({
      baselineIndex,currentIndex,
      consumers:[{id:baselineReport.application,subjects:[...base.subjects.values()]}],
    })[0];
  } catch {
    return {...context,status:'unresolved',reason:'invalid_source_index'};
  }
  if (scope.status !== 'unchanged') {
    return {...context,status:'unresolved',reason:'indexed_source_changed',
      affected:scope.affected};
  }
  return {...context,status:'limited-unchanged',
    dependencyCount:base.subjects.size};
}
