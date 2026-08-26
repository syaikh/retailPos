// Local print agent for the POS.
//
// Receives receipt payloads from the browser and routes them to a printer or
// file based on the PRINT_TARGET env var (see printer.js). This is the "print
// bridge" that lets a web POS print silently to a 58mm thermal printer without
// the browser print dialog — the way real high-volume retail works.
//
// No external dependencies: runs on a bare Node install.
//
// Usage:
//   PRINT_TARGET=file   node index.js      # writes openable HTML to /tmp
//   PRINT_TARGET=thermal node index.js     # writes ESC/POS .bin (or serial)
//   PRINT_TARGET=pdf    node index.js      # HTML + optional CUPS spool
//
// Env:
//   PORT               agent listen port      (default 9123)
//   PRINT_TARGET       file | pdf | thermal   (default file)
//   PRINT_OUTPUT_DIR   output directory       (default /tmp)
//   PRINT_SERIAL_PORT  e.g. /dev/ttyUSB0      (thermal only)
//   PRINT_PDF_PRINTER  CUPS printer name      (pdf only, optional)

import { createServer } from 'node:http';
import { writeReceipt } from './printer.js';

const PORT = Number(process.env.PORT) || 9123;
const TARGET = process.env.PRINT_TARGET || 'file';
const DIR = process.env.PRINT_OUTPUT_DIR || '/tmp';
const SERIAL = process.env.PRINT_SERIAL_PORT || '';

const CORS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET,POST,OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type',
};

function sendJson(res, status, body) {
  res.writeHead(status, { ...CORS, 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
}

const server = createServer((req, res) => {
  if (req.method === 'OPTIONS') {
    res.writeHead(204, CORS);
    return res.end();
  }

  if (req.method === 'GET' && req.url === '/health') {
    return sendJson(res, 200, { ok: true, target: TARGET });
  }

  if (req.method === 'POST' && req.url === '/print') {
    let body = '';
    req.on('data', (c) => (body += c));
    req.on('end', () => {
      try {
        const payload = JSON.parse(body || '{}');
        const result = writeReceipt({
          target: TARGET,
          dir: DIR,
          serialPort: SERIAL,
          invoice: payload.invoice,
          data: payload.data,
          branding: payload.branding,
        });
        return sendJson(res, 200, { ok: true, ...result });
      } catch (err) {
        return sendJson(res, 400, { ok: false, error: String(err?.message || err) });
      }
    });
    return;
  }

  sendJson(res, 404, { ok: false, error: 'not found' });
});

server.listen(PORT, () => {
  console.log(`[print-agent] listening on :${PORT} target=${TARGET} dir=${DIR}`);
});
