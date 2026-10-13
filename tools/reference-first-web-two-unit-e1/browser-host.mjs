// Diagnostic same-origin static Browser host for Issue #1182. Not a
// deployable reverse proxy, Authorization Provider, or new Runtime Target.
// The Browser asset and Go API remain two separately built physical units.
import { createHash } from "node:crypto";
import { createServer, request as httpRequest } from "node:http";
import { lstatSync, readFileSync, realpathSync } from "node:fs";
import { basename, dirname, resolve } from "node:path";

const [assetDir, apiOrigin, measurementPath] = process.argv.slice(2);
if (!assetDir || !apiOrigin || !/^http:\/\/127\.0\.0\.1:[1-9][0-9]*$/.test(apiOrigin)) {
  throw new Error("usage: node browser-host.mjs /browser-asset-dir http://127.0.0.1:API_PORT");
}
const api = new URL(apiOrigin);
const apiPort = Number(api.port);
if (!Number.isSafeInteger(apiPort) || apiPort > 65535) {
  throw new Error("invalid loopback API port");
}
const html = readFileSync(resolve(assetDir, "index.html"));
const legacyForm = html.includes('<form method="post" action="/custom-submit"');
const jsonForm = html.includes('<form id="json-command" method="post" action="/scenario/submit"');
if (legacyForm === jsonForm) {
  throw new Error("static asset must declare exactly one supported form transport");
}
const browserScript = jsonForm ? readFileSync(resolve(assetDir, "browser-command.js")) : null;
if (jsonForm && !html.includes('src="/browser-command.js"')) {
  throw new Error("JSON form transport script is missing");
}
// A diagnostic JSON Browser must bind to an exact Application/Handover/Target
// identity, not just a reachable TCP endpoint. This metadata is produced by
// the existing builder; the Host checks its Browser bytes, while the existing
// full two-unit preflight separately checks API and frozen World bytes.
// it is NOT an independent Host authority or a signed acceptance record.
if (jsonForm && !measurementPath) {
  throw new Error("JSON Browser requires its existing build measurement");
}
// Verify the bytes this Host actually serves, not just the build-time
// staging receipt. Use the existing Browser unit's measurement; do not create
// another lock or accept a measurement from a different artifact directory.
// All verified bytes are held in memory, so later path changes cannot alter
// what this diagnostic Host publishes during its lifetime.
const sha256 = bytes => createHash("sha256").update(bytes).digest("hex");
const realRegular = (path, label) => {
  if (realpathSync(path) !== path || !lstatSync(path).isFile()) {
    throw new Error(label + " is not a real regular artifact file");
  }
  return readFileSync(path);
};
const assetRoot = resolve(assetDir);
const measurementFile = measurementPath ? resolve(measurementPath) : null;
if (jsonForm && (
  basename(assetRoot) !== "browser" ||
  measurementFile !== resolve(dirname(assetRoot), "measurement.json") ||
  realpathSync(assetRoot) !== assetRoot
)) {
  throw new Error("JSON Browser measurement must belong to the same artifact root");
}
const measurement = measurementFile ?
  JSON.parse(jsonForm ?
    realRegular(measurementFile, "measurement").toString("utf8") :
    readFileSync(measurementFile, "utf8")) : null;
if (jsonForm) {
  const checks = [
    { name: "Browser HTML", bytes: html, file: resolve(assetRoot, "index.html"),
      hash: "browserSHA256", size: "browserBytes" },
    { name: "Browser script", bytes: browserScript, file: resolve(assetRoot, "browser-command.js"),
      hash: "browserScriptSHA256", size: "browserScriptBytes" },
  ];
  for (const check of checks) {
    if (realpathSync(check.file) !== check.file || !lstatSync(check.file).isFile()) {
      throw new Error(check.name + " must be a real file in the Browser unit");
    }
    const claimed = measurement[check.hash];
    if (typeof claimed !== "string" || !/^[0-9a-f]{64}$/.test(claimed) ||
        !Number.isSafeInteger(measurement[check.size]) || measurement[check.size] <= 0 ||
        claimed !== sha256(check.bytes) || measurement[check.size] !== check.bytes.length) {
      throw new Error(check.name + " byte/digest binding differs from measurement");
    }
  }
}
const expectedBinding = jsonForm ? {
  applicationId: measurement?.applicationId,
  handoverBaselineId: measurement?.handoverBaselineId,
  baselineRepositoryRevision: measurement?.baselineRepositoryRevision,
  targetBuildRevision: measurement?.browserAndApiSourceRevision,
} : null;
if (jsonForm && (
  expectedBinding.applicationId !== "order_entry" ||
  typeof expectedBinding.handoverBaselineId !== "string" ||
  !/^[a-f0-9]{40}$/.test(expectedBinding.baselineRepositoryRevision || "") ||
  !/^[a-f0-9]{40}$/.test(expectedBinding.targetBuildRevision || "") ||
  measurement.browserTransport !== "same-origin-json"
)) {
  throw new Error("invalid expected JSON Browser/API binding");
}
const checkAPIBinding = async () => {
  if (!expectedBinding) return;
  const response = await fetch(apiOrigin + "/manifest.json", {
    redirect: "error",
    signal: AbortSignal.timeout(3000),
    headers: { "Cache-Control": "no-cache" },
  });
  if (response.status !== 200) throw new Error("API manifest unavailable");
  if (Number(response.headers.get("content-length") || "0") > 65536) {
    throw new Error("API manifest exceeds diagnostic limit");
  }
  const body = await response.text();
  if (Buffer.byteLength(body) > 65536) throw new Error("API manifest exceeds diagnostic limit");
  const manifest = JSON.parse(body);
  for (const [field, required] of Object.entries(expectedBinding)) {
    if (manifest[field] !== required) {
      throw new Error("API binding differs: " + field);
    }
  }
};
// Refuse to expose even the static diagnostic Browser when its chosen API
// is missing or bound to a different B/R/T. No role/capability is inferred.
await checkAPIBinding();
const header = {
  "Cache-Control": "no-store",
  "X-Content-Type-Options": "nosniff",
};
const report = (res, status, message, extra = {}) => {
  res.writeHead(status, {
    ...header, "Content-Type": "text/plain; charset=utf-8", ...extra,
  });
  res.end(message + "\n");
};
const server = createServer(async (req, res) => {
  const listening = server.address();
  const localHost = "127.0.0.1:" + listening.port;
  const localOrigin = "http://" + localHost;
  // Reject DNS rebinding and mismatched Origin, including on GET.
  if (req.headers.host !== localHost ||
      (req.headers.origin !== undefined && req.headers.origin !== localOrigin)) {
    report(res, 403, "origin is not trusted");
    return;
  }
  if (!req.url?.startsWith("/") || req.url.startsWith("//")) {
    report(res, 400, "invalid request target");
    return;
  }
  const path = new URL(req.url, localOrigin);
  if (path.hash || path.username || path.password) {
    report(res, 400, "invalid request target");
    return;
  }
  if ((path.pathname === "/" || (legacyForm && path.pathname === "/custom-submit")) &&
      req.method === "GET" && !path.search) {
    res.writeHead(200, {
      ...header,
      "Content-Type": "text/html; charset=utf-8",
      "Referrer-Policy": "no-referrer",
      "Content-Security-Policy": jsonForm
        ? "default-src 'none'; style-src 'unsafe-inline'; script-src 'self'; connect-src 'self'; form-action 'none'; base-uri 'none'; frame-ancestors 'none'"
        : "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'",
    });
    res.end(html);
    return;
  }
  if (jsonForm && path.pathname === "/browser-command.js" &&
      req.method === "GET" && !path.search) {
    res.writeHead(200, {
      ...header,
      "Content-Type": "text/javascript; charset=utf-8",
      "Referrer-Policy": "no-referrer",
    });
    res.end(browserScript);
    return;
  }
  const allowed = (
    (req.method === "GET" && path.pathname === "/manifest.json" && !path.search) ||
    (req.method === "GET" && path.pathname === "/scenario/read") ||
    (req.method === "POST" && path.pathname === "/scenario/submit" && !path.search) ||
    (legacyForm && req.method === "POST" && path.pathname === "/custom-submit" && !path.search)
  );
  if (!allowed) {
    report(res, 404, "route is not part of this reference");
    return;
  }
  // A diagnostic browser form MUST present the exact same Origin; a
  // cross-site form cannot silently acquire Go API credentials/authority.
  if (req.method === "POST" && req.headers.origin !== localOrigin) {
    report(res, 403, "POST requires the exact browser Origin");
    return;
  }
  const type = (req.headers["content-type"] || "").split(";")[0].trim().toLowerCase();
  if (req.method === "POST" &&
      (path.pathname === "/custom-submit" ? type !== "application/x-www-form-urlencoded" : type !== "application/json")) {
    report(res, 415, "unsupported media type");
    return;
  }

  // An endpoint that matched during start can be replaced by another API.
  // Verify the running manifest again before forwarding a mutating command.
  // This narrows accidental cross-unit binding; it is not an atomic trust
  // handshake, deployment signature, or proof about the downstream Provider.
  if (jsonForm && req.method === "POST") {
    try {
      await checkAPIBinding();
    } catch (_) {
      report(res, 502, "API binding unknown");
      return;
    }
  }

  let body = Buffer.alloc(0);
  if (req.method === "POST") {
    for await (const part of req) {
      if (body.length + part.length > 64 * 1024) {
        report(res, 413, "payload too large");
        return;
      }
      body = Buffer.concat([body, part]);
    }
  }
  // Never forward browser-supplied identity/role, cookie, Authorization,
  // X-Forwarded-* or Host headers. This is only an equality guard, not an
  // Authorization token: untrusted clients could reconstruct these values.
  // The API-only binary validates them against its OWN in-process projection
  // on the same HTTP request as the actual command dispatch. A stale API
  // without the explicit endpoint answers 404 instead of running a command.
  const boundCommand = jsonForm && req.method === "POST" && path.pathname === "/scenario/submit";
  const outbound = httpRequest({
    hostname: "127.0.0.1",
    port: apiPort,
    method: req.method,
    path: (boundCommand ? "/scenario/submit-bound" : path.pathname) + path.search,
    timeout: 5000,
    headers: req.method === "POST"
      ? {
          "content-type": type,
          "content-length": String(body.length),
          ...(boundCommand ? {
            "x-tei-diagnostic-application": expectedBinding.applicationId,
            "x-tei-diagnostic-baseline-id": expectedBinding.handoverBaselineId,
            "x-tei-diagnostic-baseline-revision": expectedBinding.baselineRepositoryRevision,
            "x-tei-diagnostic-target-revision": expectedBinding.targetBuildRevision,
          } : {}),
        }
      : {},
  }, upstream => {
    const chunks = [];
    let size = 0;
    upstream.on("data", part => {
      size += part.length;
      if (size > 8 * 1024 * 1024) {
        upstream.destroy(new Error("API response exceeds diagnostic limit"));
        return;
      }
      chunks.push(part);
    });
    upstream.on("end", () => {
      if (res.headersSent) return;
      if (boundCommand && upstream.statusCode === 409) {
        // A target-instance mismatch is never a Business Result.
        report(res, 502, "API binding unknown");
        return;
      }
      res.writeHead(upstream.statusCode || 502, {
        ...header,
        "Content-Type": upstream.headers["content-type"] || "application/octet-stream",
      });
      res.end(Buffer.concat(chunks));
    });
    upstream.on("error", () => {
      if (!res.headersSent) report(res, 502, "API outcome unknown");
    });
  });
  outbound.on("timeout", () => outbound.destroy(new Error("API timeout")));
  outbound.on("error", () => {
    if (!res.headersSent) report(res, 502, "API outcome unknown");
  });
  outbound.end(body);
});
server.requestTimeout = 10000;
server.headersTimeout = 10000;
server.listen(0, "127.0.0.1", () => {
  process.stdout.write("BROWSER_HOST=http://127.0.0.1:" + server.address().port + "\n");
});
