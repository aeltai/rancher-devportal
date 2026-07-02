<template>
  <div class="dp-form-controls">
    <div v-if="label" class="label">{{ label }}</div>
    <select
      :value="value"
      class="input-sm input-wide"
      @change="$emit('update:value', $event.target.value)"
    >
      <option v-if="placeholder" value="">{{ placeholder }}</option>
      <option
        v-for="opt in normalizedOptions"
        :key="opt.value"
        :value="opt.value"
      >
        {{ opt.label }}
      </option>
    </select>
  </div>
</template>

<script>
export default {
  name: 'DpSelect',
  model: { prop: 'value', event: 'update:value' },
  props: {
    value: { type: [String, Number], default: '' },
    label: { type: String, default: '' },
    options: { type: Array, default: () => [] },
    optionKey: { type: String, default: 'value' },
    optionLabel: { type: String, default: 'label' },
    placeholder: { type: String, default: '' },
  },
  emits: ['update:value'],
  computed: {
    normalizedOptions() {
      return (this.options || []).map((opt) => {
        if (typeof opt === 'string' || typeof opt === 'number') {
          return { value: opt, label: String(opt) };
        }
        return {
          value: opt[this.optionKey] ?? opt.value ?? opt.id,
          label: opt[this.optionLabel] ?? opt.label ?? opt.name ?? String(opt.value),
        };
      });
    },
  },
};
</script>

<style scoped>
.dp-form-controls { margin-bottom: 8px; }
</style>
