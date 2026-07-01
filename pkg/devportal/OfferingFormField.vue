<template>
  <div class="offering-form-field">
    <label class="label">
      {{ field.label }}
      <span v-if="field.required" class="required">*</span>
    </label>
    <select
      v-if="field.type === 'select'"
      :value="modelValue"
      class="input-sm input-wide"
      @change="$emit('update:modelValue', $event.target.value)"
    >
      <option v-for="opt in field.options || []" :key="opt" :value="opt">{{ opt }}</option>
    </select>
    <label v-else-if="field.type === 'boolean'" class="dp-checkbox-row">
      <input
        type="checkbox"
        :checked="modelValue === 'true' || modelValue === true"
        @change="$emit('update:modelValue', $event.target.checked ? 'true' : 'false')"
      >
      <span>{{ field.placeholder || field.label }}</span>
    </label>
    <input
      v-else
      :value="modelValue"
      :type="field.type === 'number' ? 'number' : 'text'"
      class="input-sm input-wide"
      :placeholder="field.placeholder || field.default || ''"
      @input="$emit('update:modelValue', $event.target.value)"
    >
  </div>
</template>

<script>
export default {
  name: 'OfferingFormField',
  props: {
    field: { type: Object, required: true },
    modelValue: { type: [String, Number, Boolean], default: '' },
  },
  emits: ['update:modelValue'],
};
</script>

<style scoped>
.offering-form-field { margin-bottom: 12px; }
.dp-checkbox-row { display: flex; align-items: center; gap: 8px; cursor: pointer; }
.required { color: var(--error, #c00); margin-left: 2px; }
</style>
