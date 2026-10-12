// O2 read-only bridge: present a validated TEI display manifest through the
// existing project-customizable React starter without editing that starter.
import {createActionFormStarter} from '../../ui-customization/web-react/starter.mjs';
import {referenceUIFields} from '../../ui-customization/web-react/order-entry-example.mjs';
import {createReferenceSubmitAction} from '../../ui-customization/web-react/action-port.mjs';

const hexDigest = /^([a-f0-9]{64})$/;
const locales = new Set(['ja-JP', 'en-US']);

export function validateOrderTitleManifest(manifest) {
  if (!manifest || typeof manifest !== 'object' || Array.isArray(manifest) ||
      manifest.kind !== 'tei.multilingual.display-manifest.o2' ||
      manifest.environment !== 'env.order-reference' ||
      typeof manifest.lockFingerprint !== 'string' ||
      !hexDigest.test(manifest.lockFingerprint) ||
      manifest.target?.id !== 'std.order-entry' ||
      manifest.target?.version !== '1.2.0' ||
      manifest.resource?.owner !== 'std.order-entry' ||
      manifest.resource?.target?.scope !== 'sales' ||
      manifest.resource?.target?.subject?.kind !== 'function' ||
      manifest.resource?.target?.subject?.id !== 'sales.order.submit' ||
      manifest.resource?.field !== 'display-name' ||
      !locales.has(manifest.locale) ||
      typeof manifest.text !== 'string' || !manifest.text.trim() ||
      typeof manifest.sourceRevision !== 'string' ||
      !manifest.sourceRevision.startsWith('sha256:') ||
      !hexDigest.test(manifest.sourceRevision.slice(7)) ||
      typeof manifest.fromSource !== 'boolean' ||
      typeof manifest.fallback !== 'boolean' || manifest.fallback ||
      manifest.fromSource !== (manifest.locale === 'ja-JP')) {
    throw new Error('invalid_locked_display_manifest');
  }
  return manifest;
}

// Only the displayed form title is localized in O2. The action, field IDs,
// business rules and access control remain wholly owned by the normal UI port.
export function createLocalizedReferenceOrderEntryUI(React, {
  manifest, invoke, fetchImpl, slots,
} = {}) {
  validateOrderTitleManifest(manifest);
  const action = invoke ?? createReferenceSubmitAction({fetchImpl});
  if (typeof action !== 'function') throw new Error('invalid_action_port');
  return createActionFormStarter(React, {
    title: manifest.text,
    fields: referenceUIFields,
    invoke: action,
    slots,
  });
}
