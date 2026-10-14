// Project-owned advisory transport. Not a TEI business Action or Authority.
const supportedKinds = new Set(['review_suggested', 'no_suggestion']);
export function createAdviceAction(fetchImpl) {
  if (typeof fetchImpl !== 'function') throw Error('missing_fetch');
  return async ({customerCode, note}) => {
    if (typeof customerCode !== 'string' || typeof note !== 'string' ||
        customerCode.length > 64 || note.length > 240 ||
        !/^[A-Za-z0-9-]+$/.test(customerCode.trim())) throw Error('invalid_input');
    let response;
    try {
      response = await fetchImpl('/project/advice', {
        method: 'POST', credentials: 'same-origin', cache: 'no-store', redirect: 'error',
        headers: {'Content-Type': 'application/json', 'Accept': 'application/json'},
        body: JSON.stringify({customerCode, note}),
      });
    } catch { throw Error('unknown_result'); }
    if (!response || response.status !== 200 ||
        !/^application\/json(?:\s*;|$)/i.test(response.headers?.get('content-type') ?? '')) {
      throw Error('unexpected_response');
    }
    let body;
    try { body = await response.json(); } catch { throw Error('unexpected_response'); }
    if (!body || body.schema !== 'tei.reference.project-customer-advice/v0' ||
        body.advisory !== true || !supportedKinds.has(body.kind) ||
        typeof body.message !== 'string' || body.message.length > 300) {
      throw Error('unexpected_response');
    }
    return {kind: body.kind, message: body.message};
  };
}
