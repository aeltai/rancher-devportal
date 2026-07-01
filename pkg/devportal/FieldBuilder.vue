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
          <LabeledInput v-model:value="field.key" label="Key" placeholder="cpu" />
        </div>
        <div class="col span-3">
          <LabeledInput v-model:value="field.label" label="Label" placeholder="CPU cores" />
        </div>
        <div class="col span-3">
          <LabeledSelect
            v-model:value="field.type"
            label="Type"
            :options="typeOptions"
            option-key="value"
            option-label="label"
          />
        </div>
        <div class="col span-3">
          <LabeledInput v-model:value="field.default" label="Default" />
        </div>
        <div class="col span-6">
          <LabeledInput
            v-model:value="field.specPath"
            label="Spec path"
            placeholder="spec.template.spec.domain.cpu.cores"
            tooltip="Dot path on the CR spec where this value is written (CRD offerings only)"
          />
        </div>
        <div v-if="field.type === 'select'" class="col span-6">
          <LabeledInput
            :value="(field.options || []).join(', ')"
            label="Options"
            placeholder="small, medium, large"
            @update:value="field.options = splitCsv($event)"
          />
        </div>
        <div class="col span-12">
          <Checkbox v-model:value="field.required" label="Required field" />
        </div>
      </div>
    </article>
  </div>
</template>

<script>
import LabeledSelect from '@shell/components/form/LabeledSelect';
import { LabeledInput } from '@components/Form/LabeledInput';
import { Checkbox } from '@components/Form/Checkbox';

export default {
  name: 'FieldBuilder',
  components: { LabeledInput, LabeledSelect, Checkbox },
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
        ...this.localFields,
        { key: '', label: '', type: 'text', specPath: '', default: '', required: false, options: [] },
      ]);
    },
    removeField(idx) {
      const next = [...this.localFields];
      next.splice(idx, 1);
      this.$emit('update:modelValue', next);
    },
  },
};
</script>

<style lang="scss" scoped>
.field-builder {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.field-builder-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 12px;

  strong {
    display: block;
    font-size: 0.88em;
    margin-bottom: 2px;
  }

  p {
    margin: 0;
    font-size: 0.78em;
    color: var(--muted);
  }
}

.field-builder-empty {
  padding: 12px;
  font-size: 0.82em;
  color: var(--muted);
  border: 1px dashed var(--border);
  border-radius: 4px;
  text-align: center;
}

.field-builder-row {
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 12px;
  margin-bottom: 10px;
  background: var(--sortable-table-row-bg, var(--body-bg));
}

.field-builder-row-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
  font-size: 0.85em;
  font-weight: 600;
}
</style>
