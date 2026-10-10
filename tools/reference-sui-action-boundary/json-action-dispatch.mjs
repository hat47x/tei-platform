// Browser-side same-origin transport for the reference native action boundary.
// The endpoint must be hosted by the application runtime, not a direct third-party URL.
export class ActionCommitError extends Error {
  constructor(code, status) {
    super(code);
    this.name = "ActionCommitError";
    this.code = code;
    this.status = status;
  }
}

export function createJSONActionDispatcher(endpoint, fetchImpl = globalThis.fetch) {
  if (typeof endpoint !== "string" || !endpoint.startsWith("/") || endpoint.startsWith("//") || endpoint.includes("\\") ||
      /[\u0000-\u001f\u007f]/.test(endpoint) || typeof fetchImpl !== "function") {
    throw new TypeError("same-origin endpoint path and fetch function are required");
  }
  return async (intent) => {
    if (!intent || intent.protocolVersion !== "1" ||
        [intent.applicationID, intent.resourceID, intent.actionID, intent.expectedRevision]
          .some((value) => typeof value !== "string" || value.length === 0) ||
        intent.payload === null || intent.payload === undefined) {
      throw new TypeError("invalid versioned resource Action intent");
    }
    const result = await fetchImpl(endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-TEI-Action": "commit" },
      credentials: "same-origin",
      mode: "same-origin",
      redirect: "error",
      cache: "no-store",
      body: JSON.stringify(intent),
    });
    let body;
    try {
      body = await result.json();
    } catch {
      throw new ActionCommitError("invalid_response", result.status);
    }
    if (body?.protocolVersion !== "1") {
      throw new ActionCommitError("invalid_response", result.status);
    }
    if (!result.ok) {
      const code = typeof body?.error === "string" ? body.error : "execution_failed";
      throw new ActionCommitError(code, result.status);
    }
    if (typeof body?.revision !== "string" || body.revision.length === 0) {
      throw new ActionCommitError("invalid_response", result.status);
    }
    return body.revision;
  };
}
