import { promises as fs } from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = process.cwd();
const triagePath = path.join(root, 'docs', 'generated', 'native-command-triage.json');
const srcTauriDir = path.join(root, 'src-tauri', 'src');
const srcDir = path.join(root, 'src');
const outDir = path.join(root, 'docs', 'generated');
const outJsonPath = path.join(outDir, 'rust-retirement-map.json');
const outMdPath = path.join(outDir, 'rust-retirement-map.md');

const frontendExts = new Set(['.ts', '.tsx', '.js', '.jsx']);
const rustExts = new Set(['.rs']);

async function listFiles(dir, exts, out = []) {
  let entries = [];
  try {
    entries = await fs.readdir(dir, { withFileTypes: true });
  } catch {
    return out;
  }
  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'target' || entry.name === '.git') {
        continue;
      }
      await listFiles(fullPath, exts, out);
    } else if (exts.has(path.extname(entry.name))) {
      out.push(fullPath);
    }
  }
  return out;
}

function rel(filePath) {
  return path.relative(root, filePath).replaceAll(path.sep, '/');
}

function escapeRegex(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

function lineNumber(text, index) {
  return text.slice(0, index).split(/\r?\n/).length;
}

function uniqueSorted(values) {
  return Array.from(new Set(values)).sort((a, b) => a.localeCompare(b));
}

function firstLines(values, max = 10) {
  const sorted = uniqueSorted(values);
  if (sorted.length <= max) {
    return sorted.join(', ');
  }
  return `${sorted.slice(0, max).join(', ')}; +${sorted.length - max} more`;
}

function formatMetricValue(value) {
  if (value && typeof value === 'object') {
    return Object.entries(value)
      .map(([key, entryValue]) => `${key}: ${entryValue}`)
      .join(', ');
  }
  return String(value);
}

function tauriInvokeNames(text) {
  const names = new Set();
  const staticImportPattern = /import\s*\{([^}]+)\}\s*from\s*['"]@tauri-apps\/api\/core['"]/g;
  let match;
  while ((match = staticImportPattern.exec(text)) !== null) {
    for (const rawPart of match[1].split(',')) {
      const part = rawPart.trim();
      const alias = /^invoke\s+as\s+([A-Za-z_$][\w$]*)$/.exec(part);
      if (part === 'invoke') {
        names.add('invoke');
      } else if (alias) {
        names.add(alias[1]);
      }
    }
  }

  const dynamicAliasPattern = /\{\s*invoke\s*:\s*([A-Za-z_$][\w$]*)\s*\}\s*=\s*await\s+import\(\s*['"]@tauri-apps\/api\/core['"]\s*\)/g;
  while ((match = dynamicAliasPattern.exec(text)) !== null) {
    names.add(match[1]);
  }
  if (/\{\s*invoke\s*\}\s*=\s*await\s+import\(\s*['"]@tauri-apps\/api\/core['"]\s*\)/.test(text)) {
    names.add('invoke');
  }
  if (/import\(\s*['"]@tauri-apps\/api\/core['"]\s*\)\s*\)?\s*\.invoke/.test(text)) {
    names.add('invoke');
  }
  if (/import\(\s*['"]@tauri-apps\/api\/core['"]\s*\)\.then\(m\s*=>\s*m\.invoke\)/.test(text)) {
    names.add('invoke');
  }

  return names;
}

function generateHandlerBodies(text) {
  const bodies = [];
  const marker = 'tauri::generate_handler![';
  let searchIndex = 0;

  while (searchIndex < text.length) {
    const markerIndex = text.indexOf(marker, searchIndex);
    if (markerIndex === -1) {
      break;
    }

    const bodyStart = markerIndex + marker.length;
    let depth = 1;
    let cursor = bodyStart;
    while (cursor < text.length && depth > 0) {
      const char = text[cursor];
      if (char === '[') {
        depth += 1;
      } else if (char === ']') {
        depth -= 1;
      }
      cursor += 1;
    }

    if (depth === 0) {
      bodies.push({ start: bodyStart, text: text.slice(bodyStart, cursor - 1) });
      searchIndex = cursor;
    } else {
      searchIndex = bodyStart;
    }
  }

  return bodies;
}

function commandRegistrationsInGenerateHandler(libText, name) {
  const commandPattern = new RegExp(`(?<![A-Za-z0-9_])${escapeRegex(name)}(?![A-Za-z0-9_])`, 'g');
  const registrations = [];

  for (const body of generateHandlerBodies(libText)) {
    commandPattern.lastIndex = 0;
    let match;
    while ((match = commandPattern.exec(body.text)) !== null) {
      registrations.push({
        file: rel(path.join(srcTauriDir, 'lib.rs')),
        line: lineNumber(libText, body.start + match.index),
      });
    }
  }

  return registrations;
}

async function main() {
  const triage = JSON.parse(await fs.readFile(triagePath, 'utf8'));
  const rustFiles = await listFiles(srcTauriDir, rustExts);
  const frontendFiles = await listFiles(srcDir, frontendExts);
  const rustTexts = new Map(await Promise.all(rustFiles.map(async file => [file, await fs.readFile(file, 'utf8')])));
  const frontendTexts = new Map(await Promise.all(frontendFiles.map(async file => [file, await fs.readFile(file, 'utf8')])));

  const libPath = path.join(srcTauriDir, 'lib.rs');
  const libText = rustTexts.get(libPath) ?? '';
  const wailsBridgePath = path.join(srcDir, 'runtime', 'wailsBridge.ts');
  const wailsBridgeText = frontendTexts.get(wailsBridgePath) ?? '';

  const items = triage.items.map(item => {
    const name = item.name;
    const functionPattern = new RegExp(`\\bpub\\s+(?:async\\s+)?fn\\s+${escapeRegex(name)}\\s*\\(`, 'g');
    const definitions = [];
    for (const [file, text] of rustTexts.entries()) {
      functionPattern.lastIndex = 0;
      let match;
      while ((match = functionPattern.exec(text)) !== null) {
        definitions.push({ file: rel(file), line: lineNumber(text, match.index) });
      }
    }

    const registrations = commandRegistrationsInGenerateHandler(libText, name);

    const bridgePattern = new RegExp(`command\\s*===\\s*['"]${escapeRegex(name)}['"]`);
    const goBridgeImplemented = bridgePattern.test(wailsBridgeText);

    const directTauriFiles = [];
    for (const [file, text] of frontendTexts.entries()) {
      const relative = rel(file);
      if (relative.startsWith('src/runtime/')) {
        continue;
      }
      const invokeNames = tauriInvokeNames(text);
      if (invokeNames.size === 0) {
        continue;
      }
      for (const invokeName of invokeNames) {
        const invokePattern = new RegExp(`(?<![A-Za-z0-9_$])${escapeRegex(invokeName)}(?:<[^>]+>)?\\s*\\(\\s*['"]${escapeRegex(name)}['"]`, 'g');
        if (invokePattern.test(text)) {
          directTauriFiles.push(relative);
          break;
        }
      }
    }

    return {
      name,
      domain: item.domain,
      status: item.status,
      count: item.count,
      frontendFiles: item.files,
      goBridgeImplemented,
      rustRegistered: registrations.length > 0,
      rustDefinitions: definitions,
      rustRegistrations: registrations,
      directTauriFiles: uniqueSorted(directTauriFiles),
      retirementCandidate: item.status === 'merge' && goBridgeImplemented && (registrations.length > 0 || definitions.length > 0),
      directTauriBlocked: item.status === 'merge' && directTauriFiles.length > 0,
    };
  });

  const summary = {
    generatedAt: new Date().toISOString(),
    uniqueCommands: triage.uniqueCommands,
    byStatus: triage.byStatus,
    mergeCommands: items.filter(item => item.status === 'merge').length,
    mergeWithGoBridge: items.filter(item => item.status === 'merge' && item.goBridgeImplemented).length,
    mergeRustRegistered: items.filter(item => item.status === 'merge' && item.rustRegistered).length,
    mergeRustDefined: items.filter(item => item.status === 'merge' && item.rustDefinitions.length > 0).length,
    retirementCandidates: items.filter(item => item.retirementCandidate).length,
    directTauriBlockedMerged: items.filter(item => item.directTauriBlocked).length,
    directTauriBlockedEdges: items
      .filter(item => item.status === 'merge')
      .reduce((total, item) => total + item.directTauriFiles.length, 0),
    directTauriBlockedFiles: uniqueSorted(items
      .filter(item => item.status === 'merge')
      .flatMap(item => item.directTauriFiles)).length,
    replaceRustRegistered: items.filter(item => item.status === 'replace' && item.rustRegistered).length,
  };

  const byRustFile = {};
  for (const item of items.filter(item => item.retirementCandidate)) {
    const files = item.rustDefinitions.length > 0
      ? item.rustDefinitions.map(definition => definition.file)
      : item.rustRegistrations.map(registration => registration.file);
    for (const file of uniqueSorted(files)) {
      byRustFile[file] ??= [];
      byRustFile[file].push(item.name);
    }
  }

  const directBlockers = items
    .filter(item => item.directTauriBlocked)
    .sort((a, b) => b.directTauriFiles.length - a.directTauriFiles.length || a.name.localeCompare(b.name));

  const output = {
    summary,
    retirementBatches: Object.fromEntries(
      Object.entries(byRustFile)
        .sort((a, b) => b[1].length - a[1].length || a[0].localeCompare(b[0]))
        .map(([file, commands]) => [file, uniqueSorted(commands)])
    ),
    directTauriBlockers: directBlockers.map(item => ({
      name: item.name,
      domain: item.domain,
      directTauriFiles: item.directTauriFiles,
    })),
    items,
  };

  await fs.mkdir(outDir, { recursive: true });
  await fs.writeFile(outJsonPath, JSON.stringify(output, null, 2) + '\n', 'utf8');

  const lines = [];
  lines.push('# Rust Retirement Map');
  lines.push('');
  lines.push(`Generated: ${summary.generatedAt}`);
  lines.push('');
  lines.push('## Summary');
  lines.push('');
  lines.push('| Metric | Count |');
  lines.push('| --- | ---: |');
  for (const [key, value] of Object.entries(summary)) {
    if (key === 'generatedAt') {
      continue;
    }
    lines.push(`| ${key} | ${formatMetricValue(value)} |`);
  }
  lines.push('');
  lines.push('## Retirement Batches');
  lines.push('');
  lines.push('| Rust file | Merged Go commands still present in Rust | Commands |');
  lines.push('| --- | ---: | --- |');
  for (const [file, commands] of Object.entries(output.retirementBatches)) {
    lines.push(`| \`${file}\` | ${commands.length} | ${commands.map(command => `\`${command}\``).join(', ')} |`);
  }
  lines.push('');
  lines.push('## Direct Tauri Call Blockers');
  lines.push('');
  lines.push('Merged commands in this table already have Go/Wails routing, but at least one frontend file still calls the command through direct `@tauri-apps/api/core` imports. Move these callers to `src/runtime/native.ts` before deleting the matching Rust command batch.');
  lines.push('');
  lines.push('| Command | Domain | Direct frontend files |');
  lines.push('| --- | --- | --- |');
  for (const item of directBlockers) {
    lines.push(`| \`${item.name}\` | ${item.domain} | ${firstLines(item.directTauriFiles)} |`);
  }
  lines.push('');
  lines.push('## Replace Commands Still Registered In Rust');
  lines.push('');
  lines.push('| Command | Domain | Rust definitions |');
  lines.push('| --- | --- | --- |');
  for (const item of items.filter(item => item.status === 'replace' && item.rustRegistered).sort((a, b) => a.domain.localeCompare(b.domain) || a.name.localeCompare(b.name))) {
    const definitions = item.rustDefinitions.map(definition => `${definition.file}:${definition.line}`);
    lines.push(`| \`${item.name}\` | ${item.domain} | ${firstLines(definitions.length > 0 ? definitions : item.rustRegistrations.map(registration => `${registration.file}:${registration.line}`))} |`);
  }
  lines.push('');
  lines.push('## Notes');
  lines.push('');
  lines.push('- `retirementCandidate` means the command is marked `merge`, has a Wails bridge route, and still has a Rust registration or definition.');
  lines.push('- `directTauriBlocked` means deleting the Rust command could still break a frontend caller that bypasses the native facade.');
  lines.push('- This is a generated snapshot. Re-run `node scripts/rust-retirement-map.mjs` after triage or bridge changes.');
  lines.push('');

  await fs.writeFile(outMdPath, lines.join('\n'), 'utf8');
  console.log(`Wrote ${rel(outJsonPath)} and ${rel(outMdPath)}`);
  console.log(JSON.stringify(summary, null, 2));
}

main().catch(error => {
  console.error(error);
  process.exitCode = 1;
});
