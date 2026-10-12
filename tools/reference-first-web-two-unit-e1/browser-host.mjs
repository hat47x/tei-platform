// Diagnostic same-origin static Browser host for Issue #1182. Not a
// deployable reverse proxy, Authorization Provider, or new Runtime Target.
// The Browser asset and Go API remain two separately built physical units.
import { createServer, request as httpRequest } from "node:http";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";

const [assetDir, apiOrigin] = process.argv.slice(2);
if (!assetDir || !apiOrigin || !/^http:\/\/127\.0\.0\.1:[1-9][0-9]*$/.test(apiOrigin)) {
  throw new Error("usage: node browser-host.mjs /browser-asset-dir http://127.0.0.1:API_PORT");
}
const api = new URL(apiOrigin);
const apiPort = Number(api.port);
if (!Number.isSafeInteger(apiPort) || apiPort > 65535) {
  throw new Error("invalid loopback API port");
}
const html = readFileSync(resolve(assetDir, "index.html"));
if (!html.includes('<form method="post" action="/custom-submit"')) {
  throw new Error("the selected static asset does not declare the existing form contract");
}
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
  if ((path.pathname === "/" || path.pathname === "/custom-submit") &&
      req.method === "GET" && !path.search) {
    res.writeHead(200, {
      ...header,
      "Content-Type": "text/html; charset=utf-8",
      "Content-Security-Policy": "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; base-uri 'none'; frame-ancestors 'none'",
    });
    res.end(html);
    return;
  }
  const allowed = (
    (req.method === "GET" && path.pathname === "/manifest.json" && !path.search) ||
    (req.method === "GET" && path.pathname === "/scenario/read") ||
    (req.method === "POST" && path.pathname === "/scenario/submit" && !path.search) ||
    (req.method === "POST" && path.pathname === "/custom-submit" && !path.search)
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
  // X-Forwarded-* or Host headers. Existing First Web Target owns the
  // scenario and the legacy fixture verifier; this proxy grants nothing.
  const outbound = httpRequest({
    hostname: "127.0.0.1",
    port: apiPort,
    method: req.method,
    path: path.pathname + path.search,
    timeout: 5000,
    headers: req.method === "POST"
      ? { "content-type": type, "content-length": String(body.length) }
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
