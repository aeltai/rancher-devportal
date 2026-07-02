<template>
  <label class="dp-checkbox">
    <input
      type="checkbox"
      :checked="!!value"
      @change="$emit('update:value', $event.target.checked)"
    >
    <span>{{ label }}</span>
  </label>
</template>

<script>
export default {
  name: 'DpCheckbox',
  model: { prop: 'value', event: 'update:value' },
  props: {
    value: { type: Boolean, default: false },
    label: { type: String, default: '' },
  },
  emits: ['update:value'],
};
</script>

<style scoped>
.dp-checkbox { display: flex; align-items: center; gap: 8px; margin: 6px 0; cursor: pointer; }
</style>
