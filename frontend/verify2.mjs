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
  for (const [k, fn] of [['lists', () => app.TodoListLists(false)], ['reminders', () => app.TodoListReminders(30)], ['settings', () => app.GetSetting('user_agreement_accepted')]]) {
    try { out[k] = JSON.stringify(await fn()).slice(0, 60); } catch (e) { out[k] = 'ERR:' + String(e).slice(0, 80); }
  }
  return out;
});
console.log(JSON.stringify(r));
await b.close();
