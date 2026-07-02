<template>
  <article class="dp-collapse">
    <button type="button" class="dp-collapse-head" @click="open = !open">
      <span>{{ title }}</span>
      <i :class="['icon', open ? 'icon-chevron-up' : 'icon-chevron-down']" />
    </button>
    <div v-show="open" class="dp-collapse-body">
      <slot />
    </div>
  </article>
</template>

<script>
export default {
  name: 'DpCollapse',
  props: {
    title: { type: String, default: '' },
    isCollapsed: { type: Boolean, default: true },
  },
  emits: ['toggleCollapse'],
  data() {
    return { open: !this.isCollapsed };
  },
  watch: {
    isCollapsed(v) {
      this.open = !v;
    },
    open(v) {
      this.$emit('toggleCollapse', !v);
    },
  },
};
</script>

<style scoped>
.dp-collapse { border: 1px solid var(--border); border-radius: 4px; margin-bottom: 10px; }
.dp-collapse-head {
  width: 100%; display: flex; justify-content: space-between; align-items: center;
  padding: 10px 12px; border: none; background: var(--body-bg); cursor: pointer; text-align: left;
}
.dp-collapse-body { padding: 0 12px 12px; }
</style>
