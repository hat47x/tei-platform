// Reviewer-side mock provider (loopback only), Anthropic-Messages-compatible SSE.
// Scripted tool calls; logs every request body.
const http = require('http'), fs = require('fs'), path = require('path');
const out = process.argv[2], steps = JSON.parse(fs.readFileSync(process.argv[3], 'utf8'));
let n = 0, step = 0;
http.createServer((req, res) => {
  let b = [];
  req.on('data', d => b.push(d));
  req.on('end', () => {
    const body = Buffer.concat(b); n++;
    const id = String(n).padStart(3, '0');
    fs.writeFileSync(path.join(out, `req-${id}.json`), body);
    let j = {}; try { j = JSON.parse(body.toString() || '{}'); } catch {}
    fs.appendFileSync(path.join(out, 'requests.log'), `${id} ${req.method} ${req.url} bytes=${body.length} tools=${(j.tools||[]).length} stream=${j.stream}\n`);
    if (!/\/messages/.test(req.url)) { res.writeHead(404); return res.end('{}'); }
    const ev = [];
    const push = (type, o) => ev.push(`event: ${type}\ndata: ${JSON.stringify({ type, ...o })}\n\n`);
    push('message_start', { message: { id: 'msg_mock_' + id, type: 'message', role: 'assistant', model: j.model || 'mock', content: [], stop_reason: null, stop_sequence: null, usage: { input_tokens: 1, output_tokens: 1 } } });
    let stop;
    if (j.tools && j.tools.length && step < steps.length) {
      steps[step++].forEach((c, i) => {
        push('content_block_start', { index: i, content_block: { type: 'tool_use', id: `toolu_${id}_${i}`, name: c.name, input: {} } });
        push('content_block_delta', { index: i, delta: { type: 'input_json_delta', partial_json: JSON.stringify(c.input) } });
        push('content_block_stop', { index: i });
      });
      stop = 'tool_use';
    } else {
      push('content_block_start', { index: 0, content_block: { type: 'text', text: '' } });
      push('content_block_delta', { index: 0, delta: { type: 'text_delta', text: 'G0 preflight done.' } });
      push('content_block_stop', { index: 0 });
      stop = 'end_turn';
    }
    push('message_delta', { delta: { stop_reason: stop, stop_sequence: null }, usage: { output_tokens: 1 } });
    push('message_stop', {});
    res.writeHead(200, { 'content-type': 'text/event-stream' });
    res.end(ev.join(''));
  });
}).listen(8799, process.env.MOCK_HOST || '127.0.0.1');
