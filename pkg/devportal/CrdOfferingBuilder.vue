<template>
  <div class="crd-offering-builder">
    <Banner
      color="info"
      label="Pick a CRD from the cluster, then generate form fields from its OpenAPI schema. Edit or remove fields before saving."
    />

    <div class="row mt-10">
      <div class="col span-4">
        <LabeledSelect
          v-model:value="localTargetCluster"
          label="Target cluster"
          :options="clusterOptions"
          option-key="value"
          option-label="label"
          @update:value="onClusterChange"
        />
      </div>
      <div class="col span-8 crd-offering-actions">
        <button class="btn role-secondary" type="button" :disabled="loadingCrds" @click="loadCrds">
          {{ loadingCrds ? 'Loading CRDs…' : 'Load CRDs' }}
        </button>
      </div>
    </div>

    <div v-if="crds.length" class="row mt-10">
      <div class="col span-6">
        <LabeledSelect
          v-model:value="selectedCrdId"
          label="Custom resource"
          :options="crdOptions"
          option-key="value"
          option-label="label"
          searchable
          @update:value="onCrdSelected"
        />
      </div>
      <div class="col span-6 crd-offering-actions">
        <button
          class="btn role-primary"
          type="button"
          :disabled="!selectedCrdId || generating"
          @click="generateFormSchema"
        >
          {{ generating ? 'Generating…' : 'Generate form from schema' }}
        </button>
      </div>
    </div>

    <p v-if="message" :class="['crd-offering-msg', messageType]">{{ message }}</p>

    <div v-if="selectedCrdMeta" class="crd-offering-meta">
      <code>{{ selectedCrdMeta.apiVersion }}</code>
      · Kind <strong>{{ selectedCrdMeta.kind }}</strong>
      · Scope {{ selectedCrdMeta.scope || '—' }}
      <span v-if="lastFieldCount"> · {{ lastFieldCount }} field(s)<span v-if="lastTruncated"> (truncated)</span></span>
    </div>

    <FieldBuilder v-model="localFormSchema" />
  </div>
</template>

<script>
import LabeledSelect from '@shell/components/form/LabeledSelect';
import { Banner } from '@components/Banner';
import FieldBuilder from './FieldBuilder.vue';

export default {
  name: 'CrdOfferingBuilder',
  components: { LabeledSelect, Banner, FieldBuilder },
  props: {
    apiVersion: { type: String, default: '' },
    kindName: { type: String, default: '' },
    targetCluster: { type: String, default: '' },
    formSchema: { type: Array, default: () => [] },
    clusterOptions: { type: Array, default: () => [] },
    apiFn: { type: Function, required: true },
  },
  emits: ['update:apiVersion', 'update:kindName', 'update:targetCluster', 'update:formSchema'],
  data() {
    return {
      localTargetCluster: this.targetCluster || 'local',
      crds: [],
      selectedCrdId: '',
      loadingCrds: false,
      generating: false,
      message: '',
      messageType: 'info',
      lastFieldCount: 0,
      lastTruncated: false,
    };
  },
  computed: {
    crdOptions() {
      return this.crds.map((c) => ({
        label: `${c.kind} (${c.apiVersion})`,
        value: c.id,
        ...c,
      }));
    },
    selectedCrdMeta() {
      if (!this.selectedCrdId) return null;
      return this.crds.find((c) => c.id === this.selectedCrdId) || null;
    },
    localFormSchema: {
      get() { return this.formSchema || []; },
      set(v) { this.$emit('update:formSchema', v); },
    },
  },
  watch: {
    targetCluster(v) {
      if (v && v !== this.localTargetCluster) this.localTargetCluster = v;
    },
    apiVersion() { this.syncSelectedFromProps(); },
    kindName() { this.syncSelectedFromProps(); },
  },
  mounted() {
    if (this.apiVersion && this.kindName) {
      this.loadCrds().then(() => this.syncSelectedFromProps());
    }
  },
  methods: {
    syncSelectedFromProps() {
      const match = this.crds.find((c) => c.apiVersion === this.apiVersion && c.kind === this.kindName);
      if (match) {
        this.selectedCrdId = match.id;
      }
    },
    onClusterChange(cluster) {
      this.$emit('update:targetCluster', cluster);
      this.crds = [];
      this.selectedCrdId = '';
      this.message = 'Cluster changed — load CRDs again.';
      this.messageType = 'info';
    },
    async loadCrds() {
      this.loadingCrds = true;
      this.message = '';
      try {
        const data = await this.apiFn(
          'GET',
          `/api/portal/crds?cluster=${encodeURIComponent(this.localTargetCluster)}`
        );
        this.crds = data.crds || [];
        this.syncSelectedFromProps();
        this.message = this.crds.length
          ? `Found ${this.crds.length} CRD(s) on cluster "${this.localTargetCluster}".`
          : `No CRDs found on cluster "${this.localTargetCluster}".`;
        this.messageType = this.crds.length ? 'success' : 'warn';
      } catch (e) {
        this.message = e.message || 'Failed to load CRDs';
        this.messageType = 'error';
      } finally {
        this.loadingCrds = false;
      }
    },
    onCrdSelected(id) {
      const crd = this.crds.find((c) => c.id === id);
      if (!crd) return;
      this.$emit('update:apiVersion', crd.apiVersion);
      this.$emit('update:kindName', crd.kind);
      this.$emit('update:targetCluster', this.localTargetCluster);
    },
    async generateFormSchema() {
      const crd = this.selectedCrdMeta;
      if (!crd) return;

      const replace = this.localFormSchema.length
        ? window.confirm('Replace existing form fields with schema-generated fields?')
        : true;
      if (!replace) return;

      this.generating = true;
      this.message = 'Reading OpenAPI schema from CRD…';
      this.messageType = 'info';
      try {
        const q = new URLSearchParams({
          cluster: this.localTargetCluster,
          group: crd.group,
          version: crd.version,
          kind: crd.kind,
        });
        const data = await this.apiFn('GET', `/api/portal/crds/form-schema?${q.toString()}`);
        const fields = (data.fields || []).map((f) => ({
          key: f.key || '',
          label: f.label || f.key || '',
          type: f.type || 'text',
          specPath: f.specPath || '',
          default: f.default != null ? String(f.default) : '',
          required: !!f.required,
          options: f.options || [],
        }));
        if (!fields.length) {
          this.message = 'No scalar fields found in CRD spec schema — add fields manually or use generic offering.';
          this.messageType = 'warn';
          return;
        }
        this.$emit('update:formSchema', fields);
        this.$emit('update:apiVersion', data.apiVersion || crd.apiVersion);
        this.$emit('update:kindName', data.kind || crd.kind);
        this.lastFieldCount = data.fieldCount || fields.length;
        this.lastTruncated = !!data.truncated;
        this.message = `Generated ${fields.length} field(s) from ${data.kind} schema.`;
        if (data.truncated) {
          this.message += ' Some nested fields were skipped (limit reached).';
        }
        this.messageType = 'success';
      } catch (e) {
        this.message = e.message || 'Schema generation failed';
        this.messageType = 'error';
      } finally {
        this.generating = false;
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.crd-offering-builder {
  margin-top: 8px;
}

.crd-offering-actions {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding-bottom: 4px;
}

.crd-offering-meta {
  margin: 10px 0;
  font-size: 0.82em;
  color: var(--muted);
}

.crd-offering-msg {
  margin: 10px 0 0;
  font-size: 0.82em;
  &.success { color: var(--success, #3f8a3f); }
  &.warn { color: #b36b00; }
  &.error { color: var(--error, #c00); }
  &.info { color: var(--muted); }
}
</style>
