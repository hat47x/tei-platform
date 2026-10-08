// Reference E3: native gesture lifecycle -> application-owned draft transaction.
// This module does not implement any Canvas geometry, SUI policy, or persistence.
import {BridgeError} from './bridge.mjs';

function fail(code) { throw new BridgeError(code); }
function tag(value) { return typeof value === 'string' && value.trim().length > 0; }
function clone(value) {
  try { return JSON.parse(JSON.stringify(value)); }
  catch { fail('invalid_document'); }
}
function validAdapter(adapter) {
  return adapter && typeof adapter.preview === 'function' && typeof adapter.finish === 'function';
}

// expectedRevision is a *storage* revision, not an in-memory UI-history index.
export function beginNativeEdit(input) {
  if (!input || !tag(input.applicationId) || !tag(input.documentId) ||
      !tag(input.expectedRevision) || !tag(input.targetId) ||
      !input.document || !validAdapter(input.adapter)) fail('invalid_native_edit');
  const original = clone(input.document);
  return {...input, phase:'active', original, draft:clone(original), changed:false};
}

// Event is opaque to the generic bridge; the native adapter owns pointer delta,
// transforms, snapping, access modes and Canvas-specific preview semantics.
export function previewNativeEdit(edit, event) {
  if (edit?.phase !== 'active') fail('edit_not_active');
  const result = edit.adapter.preview(clone(edit.draft), {targetId:edit.targetId, event});
  if (!result || !result.document || typeof result.changed !== 'boolean') fail('invalid_native_preview');
  return {...edit, draft:clone(result.document), changed:edit.changed || result.changed};
}

// SUI currently finalizes the pending drag history for both pointerup and
// pointercancel. This *explicitly does not* mean a pointercancel is a rollback.
// The app owns the policy and any composite side effects at finalization.
export function finishNativeEdit(edit, reason) {
  if (edit?.phase !== 'active' || !['pointerup','pointercancel'].includes(reason)) fail('edit_not_active');
  if (!edit.changed) return {...edit, phase:'finished', draft:clone(edit.original), historyStep:null};
  const result = edit.adapter.finish(clone(edit.draft), {targetId:edit.targetId, reason});
  if (!result || !result.document) fail('invalid_native_finish');
  return {...edit, phase:'finished', draft:clone(result.document),
    historyStep:{before:clone(edit.original), after:clone(result.document)}, reason};
}

// Deliberate explicit rollback is distinct from a native pointercancel.
export function rollbackNativeEdit(edit) {
  if (edit?.phase !== 'active') fail('edit_not_active');
  return {...edit, phase:'rolled_back', draft:clone(edit.original), changed:false, historyStep:null};
}

// Persist is a separate user action, NOT implicitly triggered by pointerup.
// The authoritative store must enforce CAS and actor/tenant authorization at
// write time; the injected preflight alone is not sufficient against races.
export async function persistNativeEdit(edit, ports) {
  if (edit?.phase !== 'finished') fail('edit_not_finished');
  if (!edit.changed) return {saved:false, reason:'unchanged'};
  if (!ports || ports.applicationId !== edit.applicationId ||
      !['authorize','validate','save'].every(k=>typeof ports[k] === 'function')) fail('invalid_persist_ports');
  const request={applicationId:edit.applicationId,documentId:edit.documentId,
    expectedRevision:edit.expectedRevision,operation:'document.save'};
  if (await ports.authorize(request)!==true) fail('unauthorized');
  if (await ports.validate(clone(edit.draft),request)!==true) fail('validation_failed');
  const receipt=await ports.save({...request,document:clone(edit.draft)});
  if (!receipt || receipt.ok!==true || !tag(receipt.revision)) fail('conditional_save_failed');
  return {saved:true,revision:receipt.revision};
}
