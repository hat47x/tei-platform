import {BridgeError, moveSuiCard} from './bridge.mjs';

function fail(code) { throw new BridgeError(code); }
function validOpaqueTag(value) {
  return typeof value === 'string' && value.length > 0 && /^[\x21\x23-\x7e]+$/.test(value) && value !== '*';
}
function parseStrongEtag(header) {
  if (typeof header !== 'string' || !/^"[^"\x00-\x20\x7f]+"$/.test(header)) fail('missing_strong_etag');
  const tag = header.slice(1, -1);
  if (!validOpaqueTag(tag)) fail('missing_strong_etag');
  return tag;
}
function secureHeaders(headers) {
  const result = {};
  for (const [key, value] of Object.entries(headers ?? {})) {
    if (!/^[a-z0-9-]+$/i.test(key) ||
        ['if-match', 'if-none-match', 'content-type', 'host', 'cookie'].includes(key.toLowerCase()) ||
        typeof value !== 'string' || /[\r\n]/.test(value)) fail('unsafe_request_header');
    result[key] = value;
  }
  return result;
}
function validateConfig(config) {
  if (!config || typeof config.applicationId !== 'string' || config.applicationId.trim() === '' ||
      typeof config.authorize !== 'function' || typeof config.validate !== 'function' ||
      typeof config.fetchImpl !== 'function' || typeof config.now !== 'function') fail('invalid_http_config');
  let base;
  try { base = new URL(config.baseUrl); } catch { fail('invalid_api_url'); }
  if (!['http:', 'https:'].includes(base.protocol) || base.username || base.password ||
      base.search || base.hash) fail('invalid_api_url');
  return base.href.replace(/\/$/, '');
}

// Reference-only transport adapter for SUI's GET/PUT /docs/{docId}, not a
// replacement for SUI's authenticated session client or backend authorization.
// It requires strict If-Match; the native API otherwise permits LWW.
export function createSuiHttpPorts(config) {
  const base = validateConfig(config);
  const suppliedHeaders = secureHeaders(config.headers);
  const getUrl = (id) => {
    if (typeof id !== 'string' || !id || id === '.' || id === '..') fail('invalid_document_id');
    return `${base}/docs/${encodeURIComponent(id)}`;
  };
  const send = async (url, init) => {
    try { return await config.fetchImpl(url, init); }
    catch { fail('transport_failed'); }
  };
  return {
    applicationId: config.applicationId,
    authorize: config.authorize,
    validate: config.validate,
    apply: moveSuiCard,
    async load(documentId) {
      const response = await send(getUrl(documentId), {
        method: 'GET', headers: {...suppliedHeaders, Accept: 'application/json'},
        cache: 'no-store', credentials: 'same-origin',
      });
      if (response.status === 401 || response.status === 403) fail('provider_unauthorized');
      if (response.status === 404) fail('document_not_found');
      if (!response.ok) fail('document_load_failed');
      // A content hash calculated by the caller is not sufficient as an ETag.
      const revision = parseStrongEtag(response.headers.get('ETag'));
      let document;
      try { document = await response.json(); } catch { fail('invalid_document_response'); }
      if (!document || document.version !== 1 || document.id !== documentId ||
          !Array.isArray(document.cards)) fail('invalid_document_response');
      return {applicationId: config.applicationId, documentId, revision, document};
    },
    async save({applicationId, documentId, expectedRevision, document}) {
      if (applicationId !== config.applicationId) fail('application_mismatch');
      if (!validOpaqueTag(expectedRevision)) fail('invalid_revision');
      if (!document || document.version !== 1 || document.id !== documentId ||
          !Array.isArray(document.cards)) fail('invalid_document_response');
      // SUI frontend refreshes updatedAt before PUT; never rewrite unknown fields.
      const updatedAt = config.now();
      if (typeof updatedAt !== 'string' || !Number.isFinite(Date.parse(updatedAt))) fail('invalid_timestamp');
      const response = await send(getUrl(documentId), {
        method: 'PUT', headers: {...suppliedHeaders, Accept: 'application/json',
          'Content-Type': 'application/json', 'If-Match': `"${expectedRevision}"`},
        body: JSON.stringify({...document, updatedAt}),
        credentials: 'same-origin',
      });
      if (response.status === 409 || response.status === 412) return {ok:false};
      if (response.status === 401 || response.status === 403) fail('provider_unauthorized');
      if (!response.ok) fail('document_save_failed');
      // If the write succeeded without a revision receipt, force a fresh read;
      // do not retry the same write using LWW or pretend to know the new version.
      const revision = parseStrongEtag(response.headers.get('ETag'));
      return {ok:true, revision};
    },
  };
}
