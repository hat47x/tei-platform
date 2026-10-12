//go:build !tei_repair_only && !tei_api_only

package main

// This is presentation text, not an API capability; the API-only Go package
// must not compile this file or carry the custom UI link in its source set.
const orderScenarioIndexFooterHTML = `<footer>JSON projection: <a href="/manifest.json">/manifest.json</a> · read-only Driver slice: <a href="/scenario/read">/scenario/read</a> · submit Command: <code>POST /scenario/submit</code> · Custom UI: <a href="/custom-submit">/custom-submit</a> · health: <a href="/healthz">/healthz</a></footer>`
