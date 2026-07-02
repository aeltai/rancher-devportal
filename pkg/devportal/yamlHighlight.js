/** Escape HTML for safe v-html rendering. */
export function escapeHtml(s) {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

function highlightYamlValue(raw) {
  const t = raw.trim();
  if (t === '' || t === '|' || t === '>') return escapeHtml(raw);
  if (/^["']/.test(t)) return `<span class="yl-str">${escapeHtml(raw)}</span>`;
  if (/^(true|false|null|~)$/i.test(t)) return `<span class="yl-bool">${escapeHtml(raw)}</span>`;
  if (/^-?\d+(\.\d+)?$/.test(t)) return `<span class="yl-num">${escapeHtml(raw)}</span>`;
  return escapeHtml(raw);
}

/** Lightweight YAML syntax highlighting (no external deps). */
export function highlightYaml(source) {
  if (!source || source === '—') return escapeHtml(source || '—');
  return String(source)
    .split('\n')
    .map((line) => {
      if (/^\s*#/.test(line)) {
        return `<span class="yl-comment">${escapeHtml(line)}</span>`;
      }
      const doc = line.match(/^(\s*---\s*)$/);
      if (doc) return `<span class="yl-doc">${escapeHtml(line)}</span>`;

      const kv = line.match(/^(\s*)([A-Za-z0-9_.-]+)(\s*:\s*)(.*)$/);
      if (kv) {
        const [, indent, key, sep, rest] = kv;
        const keyCls = ['apiVersion', 'kind', 'metadata', 'spec', 'name', 'namespace'].includes(key)
          ? 'yl-key yl-key-head'
          : 'yl-key';
        return `${escapeHtml(indent)}<span class="${keyCls}">${escapeHtml(key)}</span>${escapeHtml(sep)}${highlightYamlValue(rest)}`;
      }

      const list = line.match(/^(\s*-\s+)(.*)$/);
      if (list) {
        return `<span class="yl-punct">${escapeHtml(list[1])}</span>${highlightYamlValue(list[2])}`;
      }

      return escapeHtml(line);
    })
    .join('\n');
}

/** Build nested tree nodes from flat file paths. */
export function buildFileTree(paths) {
  const root = { name: '', dirs: {}, files: [] };
  for (const path of paths) {
    const parts = path.split('/').filter(Boolean);
    let node = root;
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      const isFile = i === parts.length - 1;
      if (isFile) {
        node.files.push({ name: part, path });
      } else {
        if (!node.dirs[part]) node.dirs[part] = { name: part, dirs: {}, files: [] };
        node = node.dirs[part];
      }
    }
  }
  return root;
}

export function fileIcon(name) {
  if (name.endsWith('.yaml') || name.endsWith('.yml')) return 'yaml';
  if (name.endsWith('.md')) return 'md';
  if (name === 'fleet.yaml') return 'fleet';
  return 'file';
}
