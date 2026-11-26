// Walk through the Cockpit web UI with a visible Chrome + agent-hud.
// Usage: node scripts/walk-web-ui.js
const { spawn } = require('child_process');
const { chromium } = require('/usr/local/lib/node_modules/playwright-core');
const hud = require('/Users/faris/dev/labs/agent-hud');

const PORT = 7777;
const BASE = `http://127.0.0.1:${PORT}`;
const ROOT = '/Users/faris/dev/labs/cockpit';

async function waitForPort(ms = 30000) {
  const start = Date.now();
  while (Date.now() - start < ms) {
    try {
      const r = await fetch(BASE + '/');
      if (r.ok) return true;
    } catch (e) {}
    await new Promise(r => setTimeout(r, 500));
  }
  throw new Error('cockpit did not start');
}

(async () => {
  hud; // keep require
  const daemon = spawn('go', ['run', './cmd/cockpit', 'serve', '--foreground'], {
    cwd: ROOT,
    stdio: 'inherit',
    env: { ...process.env, COCKPIT_OPEN_BROWSER: 'false' },
  });

  try {
    await waitForPort();
    const browser = await chromium.launch({ headless: false, channel: 'chrome' });
    const page = await browser.newPage();
    await hud.attach(page);

    await hud.think(page, 'Opening Cockpit web UI');
    await page.goto(BASE);
    await hud.see(page, 'Dashboard loaded');

    await hud.think(page, 'Scroll to Machines panel');
    const machines = page.locator('#machines').first();
    await machines.scrollIntoViewIfNeeded();
    await hud.act(page, 'Inspecting machines table');
    await page.waitForTimeout(1000);

    const recheck = page.getByRole('button', { name: 'Re-check' }).first();
    if (await recheck.count()) {
      await hud.act(page, 'Click Re-check on first machine');
      await recheck.click();
      await page.waitForTimeout(500);
      await hud.see(page, 'Loading state should show Checking…');
    }

    const pair = page.getByRole('button', { name: 'Pair' }).first();
    if (await pair.count()) {
      await hud.act(page, 'Click Pair on first machine');
      await pair.click();
      await page.waitForTimeout(500);
      await hud.see(page, 'Loading state should show Pairing…');
    }

    await hud.note(page, 'Walkthrough complete');
    await page.waitForTimeout(3000);
    await browser.close();
  } finally {
    daemon.kill('SIGTERM');
  }
})();
