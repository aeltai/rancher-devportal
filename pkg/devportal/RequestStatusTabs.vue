<template>
  <div class="dp-status-tabs" role="tablist">
    <button
      v-for="tab in tabsWithCounts"
      :key="tab.id"
      :class="['dp-status-tab', { active: modelValue === tab.id }]"
      type="button"
      role="tab"
      :aria-selected="modelValue === tab.id"
      @click="$emit('update:modelValue', tab.id)"
    >
      {{ tab.label }}
      <span class="dp-status-tab-count">({{ tab.count }})</span>
    </button>
  </div>
</template>

<script>
import { statusTabsForVariant, countRequestsByStatus } from './requestStatus';

export default {
  name: 'RequestStatusTabs',
  props: {
    modelValue: { type: String, default: 'all' },
    requests: { type: Array, default: () => [] },
    needsAdminApproval: { type: Function, required: true },
    variant: { type: String, default: 'admin' },
  },
  emits: ['update:modelValue'],
  computed: {
    tabsWithCounts() {
      const counts = countRequestsByStatus(this.requests, this.needsAdminApproval);
      return statusTabsForVariant(this.variant).map((tab) => ({
        ...tab,
        count: counts[tab.id] ?? 0,
      }));
    },
  },
};
</script>

<style lang="scss" scoped>
.dp-status-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0;
  margin-bottom: 14px;
  border-bottom: 1px solid var(--border);
}

.dp-status-tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 8px 14px;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  border-radius: 0;
  background: none;
  color: var(--muted);
  font-size: 0.82em;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s, border-color 0.15s;

  &.active {
    color: var(--primary);
    border-bottom-color: var(--primary);
  }

  &:hover:not(.active) {
    color: var(--body-text);
  }
}

.dp-status-tab-count {
  font-size: 0.9em;
  font-weight: 500;
  opacity: 0.75;
}
</style>
