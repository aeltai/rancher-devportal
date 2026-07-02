<template>
  <li class="git-tree-item" :class="[node.type, { selected: node.path === selectedPath }]">
    <button
      v-if="node.type === 'file'"
      type="button"
      class="git-tree-btn"
      :title="node.path"
      @click="$emit('select', node.path)"
    >
      <i :class="['icon', fileIconClass(node.name)]" />
      <span>{{ node.name }}</span>
    </button>
    <template v-else>
      <div class="git-tree-dir">
        <i class="icon icon-folder" />
        <span>{{ node.name }}</span>
      </div>
      <ul v-if="node.children?.length" class="git-tree git-tree-nested">
        <GitTreeNode
          v-for="child in node.children"
          :key="child.key"
          :node="child"
          :selected-path="selectedPath"
          @select="$emit('select', $event)"
        />
      </ul>
    </template>
  </li>
</template>

<script>
export default {
  name: 'GitTreeNode',
  props: {
    node: { type: Object, required: true },
    selectedPath: { type: String, default: '' },
  },
  emits: ['select'],
  methods: {
    fileIconClass(name) {
      if (name.endsWith('.yaml') || name.endsWith('.yml')) return 'icon-file';
      if (name.endsWith('.md')) return 'icon-file';
      return 'icon-file';
    },
  },
};
</script>

<style lang="scss" scoped>
.git-tree-item {
  margin: 0;
  padding: 0;

  &.file.selected .git-tree-btn {
    background: rgba(0, 100, 200, 0.12);
    color: var(--primary);
    font-weight: 600;
  }
}

.git-tree-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 5px 12px;
  border: none;
  background: none;
  text-align: left;
  font-size: 0.82em;
  color: var(--body-text);
  cursor: pointer;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;

  .icon { color: var(--muted); font-size: 0.95em; }

  &:hover {
    background: var(--sortable-table-hover-bg, rgba(0, 0, 0, 0.04));
  }
}

.git-tree-dir {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  font-size: 0.78em;
  font-weight: 600;
  color: var(--muted);

  .icon { font-size: 0.95em; }
}

.git-tree-nested {
  padding-left: 8px;
  margin-left: 8px;
  border-left: 1px solid var(--border);
}
</style>
