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
await p.waitForTimeout(9000);
const r = await p.evaluate(async () => {
  const app = window.go.deepstudent.App;
  const out = {};
  try { out.todoReminders = JSON.stringify(await app.TodoListReminders()).slice(0, 80); } catch (e) { out.todoReminders = 'ERR: ' + String(e).slice(0, 120); }
  try { out.llmConfigs = JSON.stringify(await app.LLMCfgListApiConfigurations()).slice(0, 80); } catch (e) { out.llmConfigs = 'ERR: ' + String(e).slice(0, 120); }
  return out;
});
console.log(JSON.stringify(r));
await b.close();
