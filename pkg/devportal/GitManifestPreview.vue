<template>
  <div class="git-preview">
    <header v-if="gitRepo" class="git-preview-head">
      <div class="git-preview-repo">
        <i class="icon icon-globe" />
        <code>{{ gitRepo }}</code>
        <span v-if="gitBranch" class="git-branch">{{ gitBranch }}</span>
      </div>
      <span v-if="gitPath" class="git-path">{{ gitPath }}/</span>
    </header>

    <div class="git-preview-layout">
      <aside class="git-tree-pane">
        <div class="git-tree-head">Files</div>
        <ul class="git-tree">
          <GitTreeNode
            v-for="child in treeChildren"
            :key="child.key"
            :node="child"
            :selected-path="selectedPath"
            @select="$emit('select', $event)"
          />
        </ul>
      </aside>

      <div class="git-file-pane">
        <YamlCodeBlock
          v-if="selectedContent != null"
          :value="selectedContent"
          :title="selectedPath"
          icon="yaml"
          :max-height="maxHeight"
        />
        <p v-else class="git-empty">Select a file to preview manifests.</p>
      </div>
    </div>
  </div>
</template>

<script>
import YamlCodeBlock from './YamlCodeBlock.vue';
import GitTreeNode from './GitTreeNode.vue';
import { buildFileTree } from './yamlHighlight';

function flattenTree(node, prefix = '') {
  const items = [];
  const dirNames = Object.keys(node.dirs).sort();
  for (const d of dirNames) {
    const child = node.dirs[d];
    const key = prefix ? `${prefix}/${d}` : d;
    items.push({
      key,
      type: 'dir',
      name: d,
      children: flattenTree(child, key),
    });
  }
  const files = [...node.files].sort((a, b) => a.name.localeCompare(b.name));
  for (const f of files) {
    items.push({
      key: f.path,
      type: 'file',
      name: f.name,
      path: f.path,
    });
  }
  return items;
}

export default {
  name: 'GitManifestPreview',
  components: { YamlCodeBlock, GitTreeNode },
  props: {
    files: { type: Array, default: () => [] },
    gitRepo: { type: String, default: '' },
    gitBranch: { type: String, default: '' },
    gitPath: { type: String, default: '' },
    selectedPath: { type: String, default: '' },
    selectedContent: { type: String, default: '' },
    maxHeight: { type: String, default: '360px' },
  },
  emits: ['select'],
  computed: {
    treeChildren() {
      const paths = this.files.map((f) => f.path);
      return flattenTree(buildFileTree(paths));
    },
  },
};
</script>

<style lang="scss" scoped>
.git-preview {
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
  background: var(--body-bg);
}

.git-preview-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 8px 16px;
  padding: 10px 12px;
  background: var(--sortable-table-header-bg, var(--box-bg));
  border-bottom: 1px solid var(--border);
  font-size: 0.82em;
}

.git-preview-repo {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;

  .icon-globe { color: var(--primary); }

  code {
    background: transparent;
    padding: 0;
    font-size: 0.95em;
    overflow: hidden;
    text-overflow: ellipsis;
  }
}

.git-branch {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 0.85em;
  font-weight: 600;
  background: rgba(0, 100, 200, 0.12);
  color: var(--primary);
}

.git-path {
  color: var(--muted);
  font-family: ui-monospace, monospace;
  font-size: 0.9em;
}

.git-preview-layout {
  display: grid;
  grid-template-columns: minmax(200px, 280px) 1fr;
  min-height: 280px;
  align-items: stretch;
}

@media (max-width: 720px) {
  .git-preview-layout {
    grid-template-columns: 1fr;
  }
}

.git-tree-pane {
  border-right: 1px solid var(--border);
  background: var(--sortable-table-header-bg, var(--box-bg));
  overflow: auto;
  max-height: 400px;
}

.git-tree-head {
  padding: 8px 12px;
  font-size: 0.72em;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--muted);
  border-bottom: 1px solid var(--border);
}

.git-tree {
  list-style: none;
  margin: 0;
  padding: 6px 0 8px;
}

.git-file-pane {
  padding: 10px;
  min-width: 0;
  background: var(--body-bg);
}

.git-empty {
  margin: 0;
  padding: 24px;
  text-align: center;
  color: var(--muted);
  font-size: 0.88em;
}
</style>
