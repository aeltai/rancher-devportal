<template>
  <div :class="['dp-banner', color || 'info']">
    <slot>{{ label }}</slot>
  </div>
</template>

<script>
export default {
  name: 'DpBanner',
  props: {
    color: { type: String, default: 'info' },
    label: { type: String, default: '' },
  },
};
</script>

<style scoped>
.dp-banner {
  padding: 8px 10px;
  border-radius: 4px;
  margin-bottom: 10px;
  font-size: .88em;
  line-height: 1.4;
}
.dp-banner.info { background: rgba(63,138,200,.12); }
.dp-banner.warning { background: rgba(240,173,78,.15); }
.dp-banner.error { background: rgba(204,74,74,.12); color: var(--error, #c00); }
</style>
