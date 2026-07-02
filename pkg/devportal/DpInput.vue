<template>
  <div class="dp-form-controls">
    <div v-if="label" :class="['label', { required }]">{{ label }}</div>
    <textarea
      v-if="type === 'multiline'"
      :value="value"
      :rows="rows || 4"
      :placeholder="placeholder"
      class="input-sm input-wide"
      @input="$emit('update:value', $event.target.value)"
    />
    <input
      v-else
      :value="value"
      :type="type || 'text'"
      :placeholder="placeholder"
      class="input-sm"
      @input="$emit('update:value', $event.target.value)"
    />
    <p v-if="tooltip" class="hint">{{ tooltip }}</p>
  </div>
</template>

<script>
export default {
  name: 'DpInput',
  model: { prop: 'value', event: 'update:value' },
  props: {
    value: { type: [String, Number], default: '' },
    label: { type: String, default: '' },
    type: { type: String, default: 'text' },
    placeholder: { type: String, default: '' },
    rows: { type: Number, default: 4 },
    tooltip: { type: String, default: '' },
    required: { type: Boolean, default: false },
  },
  emits: ['update:value'],
};
</script>

<style scoped>
.dp-form-controls { margin-bottom: 8px; }
.label.required::after { content: ' *'; color: var(--error, #c00); }
.hint { margin: 4px 0 0; font-size: .8em; opacity: .75; }
</style>
