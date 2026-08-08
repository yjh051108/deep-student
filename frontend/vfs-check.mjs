import { chromium } from 'playwright-core';
import os from 'node:os'; import path from 'node:path'; import fs from 'node:fs';
const dir = path.join(os.homedir(), 'AppData', 'Local', 'ms-playwright');
let execPath = null;
for (const d of fs.readdirSync(dir).sort().reverse()) {
  if (d.startsWith('chromium-') && !d.includes('headless')) {
    const c = path.join(dir, d, 'chrome-win', 'chrome.exe');
    if (fs.existsSync(c)) { execPath = c; break; }
  }
}
const b = await chromium.launch({ executablePath: execPath, headless: true, args: ['--no-sandbox'] });
const p = await b.newPage();
await p.goto('http://localhost:34115', { waitUntil: 'domcontentloaded' });
await p.waitForTimeout(6000);
const r = await p.evaluate(async () => {
  const app = window.go.deepstudent.App;
  const out = {};
  try { out.hasVfsList = typeof app.VfsListFiles === 'function'; } catch {}
  try { out.files = JSON.stringify(await app.VfsListFiles('all', 200, 0)).slice(0, 120); } catch (e) { out.files = 'ERR:' + String(e).slice(0, 100); }
  return out;
});
console.log(JSON.stringify(r));
await b.close();
