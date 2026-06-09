import { promises as fs } from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();
const srcDir = path.join(root, 'src');
const exts = new Set(['.ts', '.tsx', '.js', '.jsx']);

const args = new Set(process.argv.slice(2));
const asJson = args.has('--json');
const summaryOnly = args.has('--summary');

async function walk(dir) {
  const entries = await fs.readdir(dir, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    if (entry.name === 'node_modules' || entry.name === 'dist') continue;

    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...await walk(fullPath));
    } else if (entry.isFile() && exts.has(path.extname(entry.name))) {
      files.push(fullPath);
    }
  }

  return files;
}

function rel(file) {
  return path.relative(root, file).replaceAll(path.sep, '/');
}

function collectMatches(text, regex, type, file) {
  const out = [];
  for (const match of text.matchAll(regex)) {
    out.push({
      type,
      name: match.groups?.name ?? match[1],
      file: rel(file),
      line: text.slice(0, match.index).split('\n').length,
    });
  }
  return out;
}

function byNameThenFile(a, b) {
  return a.name.localeCompare(b.name) || a.file.localeCompare(b.file) || a.line - b.line;
}

function groupByName(items) {
  const grouped = new Map();
  for (const item of items) {
    const current = grouped.get(item.name) ?? { name: item.name, count: 0, files: new Set() };
    current.count += 1;
    current.files.add(item.file);
    grouped.set(item.name, current);
  }

  return [...grouped.values()]
    .map(item => ({ ...item, files: [...item.files].sort() }))
    .sort((a, b) => b.count - a.count || a.name.localeCompare(b.name));
}

function printTable(title, rows) {
  console.log(`\n## ${title}`);
  console.log('| Count | Name | Files |');
  console.log('| ---: | --- | --- |');
  for (const row of rows) {
    console.log(`| ${row.count} | \`${row.name}\` | ${row.files.map(file => `\`${file}\``).join('<br>')} |`);
  }
}

const files = await walk(srcDir);
const inventory = [];

for (const file of files) {
  const text = await fs.readFile(file, 'utf8');
  inventory.push(...collectMatches(text, /\b(?:invoke|nativeInvoke)(?:<[^>]+>)?\(\s*['"`](?<name>[^'"`]+)['"`]/g, 'invoke', file));
  inventory.push(...collectMatches(text, /\blisten(?:<[^>]+>)?\(\s*['"`](?<name>[^'"`]+)['"`]/g, 'listen', file));
  inventory.push(...collectMatches(text, /\bemit(?:<[^>]+>)?\(\s*['"`](?<name>[^'"`]+)['"`]/g, 'emit', file));
}

inventory.sort(byNameThenFile);

const result = {
  generatedAt: new Date().toISOString(),
  root,
  totals: {
    filesScanned: files.length,
    nativeReferences: inventory.length,
    invokes: inventory.filter(item => item.type === 'invoke').length,
    listens: inventory.filter(item => item.type === 'listen').length,
    emits: inventory.filter(item => item.type === 'emit').length,
    uniqueInvokes: new Set(inventory.filter(item => item.type === 'invoke').map(item => item.name)).size,
    uniqueListens: new Set(inventory.filter(item => item.type === 'listen').map(item => item.name)).size,
    uniqueEmits: new Set(inventory.filter(item => item.type === 'emit').map(item => item.name)).size,
  },
  invokes: groupByName(inventory.filter(item => item.type === 'invoke')),
  listens: groupByName(inventory.filter(item => item.type === 'listen')),
  emits: groupByName(inventory.filter(item => item.type === 'emit')),
  references: inventory,
};

if (asJson) {
  console.log(JSON.stringify(result, null, 2));
} else {
  console.log(`# Native API Inventory`);
  console.log(`\nGenerated: ${result.generatedAt}`);
  console.log(`Files scanned: ${result.totals.filesScanned}`);
  console.log(`Native references: ${result.totals.nativeReferences}`);
  console.log(`Invokes: ${result.totals.invokes} (${result.totals.uniqueInvokes} unique)`);
  console.log(`Listens: ${result.totals.listens} (${result.totals.uniqueListens} unique)`);
  console.log(`Emits: ${result.totals.emits} (${result.totals.uniqueEmits} unique)`);

  if (!summaryOnly) {
    printTable('Invokes', result.invokes);
    printTable('Listens', result.listens);
    printTable('Emits', result.emits);
  }
}
