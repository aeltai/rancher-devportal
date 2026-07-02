<template>
  <div class="dp-tabs">
    <nav class="dp-tabs-nav">
      <button
        v-for="tab in tabs"
        :key="tab.name"
        type="button"
        :class="['dp-tab-btn', { active: modelValue === tab.name }]"
        @click="$emit('update:modelValue', tab.name)"
      >
        {{ tab.label }}
      </button>
    </nav>
    <div class="dp-tabs-panels">
      <div v-for="tab in tabs" v-show="modelValue === tab.name" :key="tab.name">
        <slot :name="tab.name" />
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'DpTabs',
  props: {
    tabs: { type: Array, required: true },
    modelValue: { type: String, required: true },
  },
  emits: ['update:modelValue'],
};
</script>

<style scoped>
.dp-tabs-nav { display: flex; flex-wrap: wrap; gap: 4px; margin-bottom: 12px; border-bottom: 1px solid var(--border); }
.dp-tab-btn {
  padding: 8px 12px; border: none; background: none; cursor: pointer;
  border-bottom: 2px solid transparent; margin-bottom: -1px;
}
.dp-tab-btn.active { border-bottom-color: var(--primary); font-weight: 600; }
</style>
