import { test, expect, TEST_USERS } from './fixtures';
import { spawn } from 'child_process';
import { mkdtempSync, rmSync, existsSync, readdirSync } from 'fs';
import { tmpdir } from 'os';
import { join } from 'path';

// Acceptance test for the silent (agent) print path. It boots the real Go print
// agent (`tools/print-agent`) in `file` transport, drives a real sale through the
// POS UI in silent mode, and asserts the receipt reached the agent, rendered to a
// `.bin` file, and that NO browser print dialog was opened (no fallback).
//
// Requires `go` on PATH (the agent is started with `go run`). Uses a non-default
// port (9124) so it does not clash with a dev agent on 9123.

const AGENT_PORT = 9125;
const AGENT_URL = `http://localhost:${AGENT_PORT}`;
const AGENT_DIR = join(__dirname, '..', '..', 'tools', 'print-agent');
const FRONTEND = process.env.FRONTEND_BASE_URL || 'http://localhost:5173';

test.describe('Silent receipt printing (real print-agent)', () => {
  let agentProc: any = null;
  let outDir = '';

  test.beforeAll(async () => {
    outDir = mkdtempSync(join(tmpdir(), 'print-agent-e2e-'));
    agentProc = spawn('go', ['run', './cmd/print-agent'], {
      cwd: AGENT_DIR,
      env: {
        ...process.env,
        PORT: String(AGENT_PORT),
        PRINT_TRANSPORT: 'file',
        PRINT_OUTPUT_DIR: outDir,
      },
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    agentProc.stderr?.on('data', (d: Buffer) => process.stderr.write(`[print-agent] ${d}`));

    const start = Date.now();
    while (Date.now() - start < 30000) {
      try {
        const res = await fetch(`${AGENT_URL}/health`);
        if (res.ok) return;
      } catch {
        /* not up yet */
      }
      await new Promise((r) => setTimeout(r, 300));
    }
    throw new Error('print-agent did not become healthy in time');
  });

  test.afterAll(async () => {
    if (agentProc) agentProc.kill('SIGKILL');
    if (outDir && existsSync(outDir)) rmSync(outDir, { recursive: true, force: true });
  });

  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.clear();
      localStorage.setItem('pos.locale', 'en');
      // Force silent mode against the agent we spawned on 9124, and trap
      // window.print so we can prove the browser dialog never opens.
      localStorage.setItem(
        'pos.printConfig',
        JSON.stringify({ mode: 'silent', agentUrl: 'http://localhost:9125' }),
      );
      (window as any)._printCallCount = 0;
      const orig = window.print;
      window.print = function () {
        (window as any)._printCallCount = ((window as any)._printCallCount || 0) + 1;
        return (orig as any).apply(window, arguments);
      };
    });
    await page.context().clearCookies();
    await page.goto(FRONTEND + '/', { waitUntil: 'domcontentloaded' });
  });

  async function completeSale(page: any) {
    await expect(page).toHaveURL(/\/login$/);
    await page.fill('#username', TEST_USERS.superadmin.username);
    await page.fill('#password', TEST_USERS.superadmin.password);
    await page.click('button[type="submit"]');
    await page.waitForURL(/\/$/, { timeout: 10000 });
    await page.click('nav button:has-text("Point of Sale")');
    await page.waitForSelector('#pos-search-input', { state: 'visible', timeout: 15000 });

    const rows = page.locator('table tbody tr');
    await rows.first().waitFor({ state: 'visible', timeout: 10000 });
    for (let i = 0; i < 2; i++) {
      await rows.nth(i).dblclick();
      await page.waitForTimeout(400);
    }

    await page.keyboard.press('F4');
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    await page.keyboard.press('F7');
    await expect(page.getByRole('dialog').locator('button:has-text("Enter")')).toBeEnabled({ timeout: 5000 });
    await page.getByRole('dialog').locator('button:has-text("Enter")').click();

    const printBtn = page.getByRole('button', { name: /Print/ });
    await printBtn.waitFor({ state: 'visible', timeout: 8000 });
    const label = (await printBtn.textContent()) || '';
    const m = label.match(/INV[-\w]+/i);
    return m ? m[0] : null;
  }

  test('completing a sale in silent mode prints to the agent and opens no dialog', async ({ page }) => {
    const invoice = await completeSale(page);

    // Poll the agent for a completed job matching this sale's invoice.
    let found: any = null;
    const pollStart = Date.now();
    while (Date.now() - pollStart < 15000) {
      try {
        const res = await page.request.get(`${AGENT_URL}/print/jobs/`);
        if (res.ok()) {
          const body = await res.json();
          const jobs: any[] = body.jobs || [];
          found = jobs.find(
            (j) => j.status === 'completed' && (!invoice || j.receipt?.invoice_number === invoice),
          );
          if (found) break;
        }
      } catch {
        /* agent not ready */
      }
      await new Promise((r) => setTimeout(r, 300));
    }

    expect(found, 'expected a completed print job on the agent').toBeTruthy();
    if (invoice) expect(found.receipt.invoice_number).toBe(invoice);

    // The rendered ESC/POS stream should have been written to disk.
    const files = readdirSync(outDir).filter((f) => f.endsWith('.bin'));
    expect(files.length).toBeGreaterThan(0);

    // Silent mode must NOT open the browser print dialog (no preview fallback).
    const printCalls = await page.evaluate(() => (window as any)._printCallCount || 0);
    expect(printCalls).toBe(0);
  });

  test('reprint button in silent mode sends another job to the agent', async ({ page }) => {
    const invoice = await completeSale(page);
    expect(invoice).toBeTruthy();

    // Count completed jobs now.
    const countCompleted = async (): Promise<number> => {
      const res = await page.request.get(`${AGENT_URL}/print/jobs/`);
      if (!res.ok()) return 0;
      const body = await res.json();
      return ((body.jobs || []) as any[]).filter((j) => j.status === 'completed').length;
    };

    const before = await countCompleted();
    await page.getByRole('button', { name: /Print/ }).click();

    let after = before;
    const pollStart = Date.now();
    while (Date.now() - pollStart < 15000) {
      after = await countCompleted();
      if (after > before) break;
      await new Promise((r) => setTimeout(r, 300));
    }
    expect(after, 'reprint should create an additional completed job').toBeGreaterThan(before);
  });
});
