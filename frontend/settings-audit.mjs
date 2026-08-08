// 审计设置页在暗色主题下的排版结构（macOS System Settings 对比）
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
const p = await b.newPage({ viewport: { width: 1440, height: 900 } });
await p.goto('http://localhost:34115', { waitUntil: 'domcontentloaded' });
await p.waitForTimeout(8000);
// 打开设置窗口
const item = p.locator('[data-wb-app-type="settings"], [title*="设置"]').first();
if (await item.count()) { await item.click(); await p.waitForTimeout(3500); }
// 强制暗色查看（用户机器可能是暗色系统）
await p.evaluate(() => {
  document.documentElement.setAttribute('data-theme', 'dark');
  document.body.classList.remove('light-theme');
  document.body.classList.add('dark-theme');
});
await p.waitForTimeout(1200);
const r = await p.evaluate(() => {
  const host = document.querySelector('[data-wb-settings-host]');
  const sidebar = document.querySelector('[data-wb-settings-host] .study-shell-sidebar-frame, [data-wb-settings-host] aside, [data-wb-settings-host] nav');
  const content = document.querySelector('[data-wb-settings-content-ready]');
  const g = el => el ? (() => { const s = getComputedStyle(el); const r = el.getBoundingClientRect(); return { bg: s.backgroundColor, w: Math.round(r.width), radius: s.borderRadius, font: s.fontSize }; })() : null;
  const navItems = Array.from(document.querySelectorAll('[data-wb-settings-host] [role="navigation"] [role="button"], [data-wb-settings-host] nav button'))
    .map(b => ({ text: (b.textContent||'').trim().slice(0, 12), h: Math.round(b.getBoundingClientRect().height) })).slice(0, 14);
  const cards = Array.from(document.querySelectorAll('[data-wb-settings-content-ready] [class*="rounded"], [data-wb-settings-content-ready] [class*="card"]'))
    .map(c => ({ cls: (c.className||'').toString().slice(0, 60), bg: getComputedStyle(c).backgroundColor, h: Math.round(c.getBoundingClientRect().height) })).slice(0, 8);
  return {
    host: g(host),
    sidebar: g(sidebar),
    content: g(content),
    navItems,
    cards,
    htmlTheme: document.documentElement.getAttribute('data-theme'),
  };
});
console.log(JSON.stringify(r, null, 1));
await b.close();
