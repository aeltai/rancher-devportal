<template>
  <div class="crd-offering-builder">
    <DpBanner
      color="info"
      label="Pick a CRD from the cluster, then generate form fields from its OpenAPI schema. Edit or remove fields before saving."
    />

    <div class="row mt-10">
      <div class="col span-4">
        <DpSelect
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
        <DpSelect
          v-model:value="selectedCrdId"
          label="Custom resource"
          :options="crdOptions"
          option-key="value"
          option-label="label"
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
import DpSelect from './DpSelect.vue';
import DpBanner from './DpBanner.vue';
import FieldBuilder from './FieldBuilder.vue';

export default {
  name: 'CrdOfferingBuilder',
  components: { DpSelect, DpBanner, FieldBuilder },
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
    onClusterChange(v) {
      this.$emit('update:targetCluster', v);
      this.crds = [];
      this.selectedCrdId = '';
    },
    syncSelectedFromProps() {
      if (!this.apiVersion || !this.kindName || !this.crds.length) return;
      const match = this.crds.find((c) => c.apiVersion === this.apiVersion && c.kind === this.kindName);
      if (match) this.selectedCrdId = match.id;
    },
    async loadCrds() {
      this.loadingCrds = true;
      this.message = '';
      try {
        const data = await this.apiFn('GET', `/api/portal/crds?cluster=${encodeURIComponent(this.localTargetCluster)}`);
        this.crds = data.crds || [];
        if (!this.crds.length) {
          this.message = 'No CRDs found on this cluster.';
          this.messageType = 'info';
        }
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
    },
    async generateFormSchema() {
      const crd = this.selectedCrdMeta;
      if (!crd) return;
      this.generating = true;
      this.message = '';
      try {
        const data = await this.apiFn('POST', '/api/portal/crds/generate-form', {
          cluster: this.localTargetCluster,
          apiVersion: crd.apiVersion,
          kind: crd.kind,
        });
        this.$emit('update:formSchema', data.formSchema || []);
        this.lastFieldCount = data.fieldCount || (data.formSchema || []).length;
        this.lastTruncated = !!data.truncated;
        this.message = `Generated ${this.lastFieldCount} field(s).`;
        this.messageType = 'info';
      } catch (e) {
        this.message = e.message || 'Generate failed';
        this.messageType = 'error';
      } finally {
        this.generating = false;
      }
    },
  },
};
</script>

<style scoped>
.crd-offering-actions { display: flex; align-items: flex-end; padding-bottom: 8px; }
.crd-offering-msg { margin: 10px 0; font-size: .88em; }
.crd-offering-msg.error { color: var(--error, #c00); }
.crd-offering-meta { margin: 8px 0 12px; font-size: .85em; opacity: .85; }
</style>
