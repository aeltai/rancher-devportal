<template>
  <div class="yaml-block" :class="{ compact }">
    <header v-if="title || showCopy" class="yaml-block-head">
      <span v-if="title" class="yaml-block-title" :title="title">
        <i v-if="icon" :class="['icon', iconClass]" />
        {{ title }}
      </span>
      <button
        v-if="showCopy"
        class="btn role-tertiary xs yaml-copy-btn"
        type="button"
        @click="copy"
      >
        <i :class="['icon', copied ? 'icon-checkmark' : 'icon-copy']" />
        {{ copied ? 'Copied' : 'Copy' }}
      </button>
    </header>
    <div class="yaml-block-body" :style="bodyStyle">
      <table class="yaml-lines" cellspacing="0">
        <tbody>
          <tr v-for="(line, idx) in lines" :key="idx">
            <td class="yaml-ln">{{ idx + 1 }}</td>
            <td class="yaml-code" v-html="line || '&nbsp;'" />
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
import { highlightYaml } from './yamlHighlight';

export default {
  name: 'YamlCodeBlock',
  props: {
    value: { type: String, default: '' },
    title: { type: String, default: '' },
    icon: { type: String, default: '' },
    compact: { type: Boolean, default: false },
    maxHeight: { type: String, default: '320px' },
    showCopy: { type: Boolean, default: true },
  },
  data() {
    return { copied: false };
  },
  computed: {
    lines() {
      const src = this.value || '—';
      return highlightYaml(src).split('\n');
    },
    iconClass() {
      const map = {
        yaml: 'icon-file',
        md: 'icon-file',
        fleet: 'icon-folder',
        file: 'icon-file',
        git: 'icon-git',
      };
      return map[this.icon] || 'icon-file';
    },
    bodyStyle() {
      return { maxHeight: this.maxHeight };
    },
  },
  methods: {
    async copy() {
      try {
        await navigator.clipboard.writeText(this.value || '');
        this.copied = true;
        setTimeout(() => { this.copied = false; }, 2000);
      } catch (_) {
        /* ignore */
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.yaml-block {
  border: 1px solid var(--border);
  border-radius: 6px;
  overflow: hidden;
  background: #0d1117;
  font-size: 0.78em;

  &.compact {
    font-size: 0.72em;
  }
}

.yaml-block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  background: #161b22;
  border-bottom: 1px solid #30363d;
  min-height: 32px;
}

.yaml-block-title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  font-size: 0.92em;
  color: #c9d1d9;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  .icon {
    color: #58a6ff;
    flex-shrink: 0;
  }
}

.yaml-copy-btn {
  flex-shrink: 0;
  font-size: 0.85em;
}

.yaml-block-body {
  overflow: auto;
  overscroll-behavior: contain;
}

.yaml-lines {
  width: 100%;
  border-collapse: collapse;

  tr:hover .yaml-ln {
    color: #8b949e;
  }
}

.yaml-ln {
  width: 1%;
  padding: 0 10px 0 8px;
  text-align: right;
  vertical-align: top;
  user-select: none;
  color: #484f58;
  font-family: ui-monospace, monospace;
  line-height: 1.55;
  border-right: 1px solid #21262d;
  background: #0d1117;
  position: sticky;
  left: 0;
}

.yaml-code {
  padding: 0 12px;
  vertical-align: top;
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  line-height: 1.55;
  color: #e6edf3;
  white-space: pre;

  :deep(.yl-key) { color: #79c0ff; }
  :deep(.yl-key-head) { color: #d2a8ff; font-weight: 600; }
  :deep(.yl-str) { color: #a5d6ff; }
  :deep(.yl-num) { color: #ffa657; }
  :deep(.yl-bool) { color: #ff7b72; }
  :deep(.yl-comment) { color: #8b949e; font-style: italic; }
  :deep(.yl-doc) { color: #8b949e; }
  :deep(.yl-punct) { color: #ff7b72; }
}
</style>
