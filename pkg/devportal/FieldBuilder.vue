<template>
  <div class="field-builder">
    <div class="field-builder-head">
      <div>
        <strong>Form fields</strong>
        <p>Dynamic inputs shown when users configure this offering.</p>
      </div>
      <button class="btn role-secondary" type="button" @click="addField">
        <i class="icon icon-circle-plus" /> Add field
      </button>
    </div>

    <div v-if="!localFields.length" class="field-builder-empty">
      No form fields defined.
    </div>

    <article v-for="(field, idx) in localFields" :key="idx" class="field-builder-row">
      <div class="field-builder-row-head">
        <span>{{ field.label || field.key || `Field ${idx + 1}` }}</span>
        <button class="btn role-tertiary" type="button" aria-label="Remove field" @click="removeField(idx)">
          <i class="icon icon-trash" />
        </button>
      </div>
      <div class="row">
        <div class="col span-3">
          <DpInput v-model:value="field.key" label="Key" placeholder="cpu" />
        </div>
        <div class="col span-3">
          <DpInput v-model:value="field.label" label="Label" placeholder="CPU cores" />
        </div>
        <div class="col span-3">
          <DpSelect
            v-model:value="field.type"
            label="Type"
            :options="typeOptions"
            option-key="value"
            option-label="label"
          />
        </div>
        <div class="col span-3">
          <DpInput v-model:value="field.default" label="Default" />
        </div>
        <div class="col span-6">
          <DpInput
            v-model:value="field.specPath"
            label="Spec path"
            placeholder="spec.template.spec.domain.cpu.cores"
            tooltip="Dot path on the CR spec where this value is written (CRD offerings only)"
          />
        </div>
        <div v-if="field.type === 'select'" class="col span-6">
          <DpInput
            :value="(field.options || []).join(', ')"
            label="Options"
            placeholder="small, medium, large"
            @update:value="field.options = splitCsv($event)"
          />
        </div>
        <div class="col span-12">
          <DpCheckbox v-model:value="field.required" label="Required field" />
        </div>
      </div>
    </article>
  </div>
</template>

<script>
import DpInput from './DpInput.vue';
import DpSelect from './DpSelect.vue';
import DpCheckbox from './DpCheckbox.vue';

export default {
  name: 'FieldBuilder',
  components: { DpInput, DpSelect, DpCheckbox },
  props: {
    modelValue: { type: Array, default: () => [] },
  },
  emits: ['update:modelValue'],
  data() {
    return {
      typeOptions: [
        { label: 'Text', value: 'text' },
        { label: 'Number', value: 'number' },
        { label: 'Boolean', value: 'boolean' },
        { label: 'Select', value: 'select' },
      ],
    };
  },
  computed: {
    localFields: {
      get() { return this.modelValue || []; },
      set(v) { this.$emit('update:modelValue', v); },
    },
  },
  methods: {
    splitCsv(value) {
      return String(value || '').split(',').map((s) => s.trim()).filter(Boolean);
    },
    addField() {
      this.$emit('update:modelValue', [
        ...(this.modelValue || []),
        { key: '', label: '', type: 'text', default: '', required: false },
      ]);
    },
    removeField(idx) {
      const next = [...(this.modelValue || [])];
      next.splice(idx, 1);
      this.$emit('update:modelValue', next);
    },
  },
};
</script>

<style scoped>
.field-builder-head { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 12px; }
.field-builder-head p { margin: 4px 0 0; font-size: .85em; opacity: .8; }
.field-builder-empty { padding: 16px; text-align: center; opacity: .7; border: 1px dashed var(--border); border-radius: 4px; }
.field-builder-row { border: 1px solid var(--border); border-radius: 4px; padding: 10px; margin-bottom: 10px; }
.field-builder-row-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
</style>
