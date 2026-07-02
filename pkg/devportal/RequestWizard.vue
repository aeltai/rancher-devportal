<template>
  <div class="request-wizard">
    <SimpleWizard
      :steps="steps"
      :errors="errors"
      banner-title="Geeko-Ops — new request"
      @cancel="$emit('cancel')"
      @finish="submit"
    >
      <template #basics>
        <label class="label">Environment name</label>
        <input v-model="form.name" class="input-sm" type="text" placeholder="my-team-dev" @input="slugifyName" />
        <p class="hint">Lowercase, numbers, hyphens. Namespace: <code>env-{{ form.slug || '…' }}</code></p>
        <label class="label">Description</label>
        <input v-model="form.description" class="input-sm input-wide" type="text" placeholder="Optional" />
      </template>

      <template #collection>
        <p class="step-lead">Choose a catalog collection — what kind of resource do you need?</p>
        <div class="dp-card-grid">
          <label
            v-for="col in collections"
            :key="col.id"
            :class="['dp-card', { selected: form.collectionId === col.id }]"
          >
            <input v-model="form.collectionId" type="radio" class="sr-only" :value="col.id" @change="onCollectionChange">
            <i :class="['icon', collectionIcon(col.icon)]" />
            <span class="dp-card-title">{{ col.label }}</span>
            <span class="dp-card-desc">{{ col.description }}</span>
          </label>
        </div>
      </template>

      <template #offering>
        <p class="step-lead">Pick an offering from <strong>{{ collectionLabel }}</strong>.</p>
        <div class="dp-card-grid">
          <label
            v-for="off in filteredOfferings"
            :key="off.id"
            :class="['dp-card', { selected: form.offeringId === off.id }]"
          >
            <input v-model="form.offeringId" type="radio" class="sr-only" :value="off.id" @change="onOfferingChange">
            <OfferingIcon :offering="off" />
            <span class="dp-card-title">{{ off.label }}</span>
            <span class="dp-card-desc">{{ off.description }}</span>
            <span v-if="off.detail" class="dp-card-detail">{{ off.detail }}</span>
          </label>
        </div>
        <div v-if="selectedOffering && selectedOffering.cloneFrom" class="dp-clone-panel">
          <h4>Based on existing namespace</h4>
          <label class="label">Cluster</label>
          <select v-model="cloneFrom.clusterId" class="input-sm input-wide" @change="loadExistingResources">
            <option value="">Select cluster…</option>
            <option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }} ({{ c.state }})</option>
          </select>
          <label class="label">Source namespace</label>
          <select v-model="cloneFrom.namespace" class="input-sm input-wide" :disabled="!existingResources.length">
            <option value="">Select namespace…</option>
            <option v-for="r in existingResources" :key="r.name" :value="r.name">{{ r.name }}</option>
          </select>
        </div>
      </template>

      <template #configure>
        <div v-if="selectedOffering && selectedOffering.kind === 'helm'">
          <p class="step-lead">Select Helm charts to include (optional extras beyond the offering defaults).</p>
          <fieldset class="dp-chart-list">
            <label v-for="c in catalog" :key="c.id" :class="['dp-chart-option', { selected: form.charts.includes(c.id) }]">
              <input v-model="form.charts" type="checkbox" :value="c.id">
              <span class="dp-chart-body">
                <span class="dp-chart-title">{{ c.name }}</span>
                <span class="dp-chart-tag">{{ c.category }}</span>
              </span>
            </label>
          </fieldset>
        </div>
        <div v-else-if="selectedOffering && hasFormSchema">
          <p class="step-lead">Configure your {{ selectedOffering.label }}.</p>
          <OfferingFormField
            v-for="field in selectedOffering.formSchema"
            :key="field.key"
            :field="field"
            :model-value="form.formValues[field.key] ?? field.default ?? ''"
            @update:model-value="setFormValue(field.key, $event)"
          />
        </div>
        <div v-else>
          <p class="step-lead">No additional configuration required for this offering.</p>
        </div>
      </template>

      <template #delivery>
        <p class="step-lead">Fleet manifests are pushed to Git after approval.</p>
        <div class="dp-form-grid">
          <div v-if="gitRepos.length > 1">
            <label class="label">Git repository</label>
            <select v-model="form.gitRepoId" class="input-sm input-wide" @change="applyGitRepoSelection">
              <option v-for="r in gitRepos" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
          </div>
          <div>
            <label class="label">Git repository URL <span class="required">*</span></label>
            <input v-model="form.gitRepo" class="input-sm input-wide" type="url" />
          </div>
          <div>
            <label class="label">Branch</label>
            <input v-model="form.gitBranch" class="input-sm" type="text" />
          </div>
          <div>
            <label class="label">Path in repo</label>
            <input v-model="form.gitPath" class="input-sm input-wide" type="text" />
          </div>
          <div>
            <label class="label">Target clusters</label>
            <input v-model="form.targetClustersText" class="input-sm input-wide" type="text" placeholder="Empty = all clusters" />
          </div>
        </div>
        <details class="dp-advanced">
          <summary>Advanced Git settings</summary>
          <label class="label">Credentials Secret</label>
          <input v-model="form.gitSecretName" class="input-sm input-wide" type="text" />
        </details>
      </template>

      <template #review>
        <dl class="dp-review-dl">
          <dt>Name</dt><dd>{{ form.name }}</dd>
          <dt>Collection</dt><dd>{{ collectionLabel }}</dd>
          <dt>Offering</dt><dd>{{ selectedOffering ? selectedOffering.label : '—' }}</dd>
          <dt>Charts</dt>
          <dd>
            <span v-if="!effectiveCharts.length" class="muted">None</span>
            <span v-for="id in effectiveCharts" :key="id" class="dp-chip">{{ id }}</span>
          </dd>
          <template v-if="hasFormSchema">
            <dt>Configuration</dt>
            <dd>
              <ul class="dp-form-review">
                <li v-for="field in selectedOffering.formSchema" :key="field.key">
                  <strong>{{ field.label }}:</strong> {{ form.formValues[field.key] || field.default || '—' }}
                </li>
              </ul>
            </dd>
          </template>
          <template v-if="needsGitOps">
            <dt>Git repo</dt><dd><code>{{ form.gitRepo || '—' }}</code></dd>
            <dt>Branch</dt><dd><code>{{ form.gitBranch || 'main' }}</code></dd>
            <dt>Path</dt><dd><code>{{ effectiveGitPath }}</code></dd>
          </template>
        </dl>
      </template>
    </SimpleWizard>
  </div>
</template>

<script>
import SimpleWizard from './SimpleWizard.vue';
import OfferingFormField from './OfferingFormField.vue';
import OfferingIcon from './OfferingIcon.vue';

const SLUG_RE = /^[a-z0-9]([a-z0-9-]{1,28}[a-z0-9])?$/;

export default {
  name: 'RequestWizard',
  components: { SimpleWizard, OfferingFormField, OfferingIcon },
  props: {
    collections: { type: Array, default: () => [] },
    offerings: { type: Array, default: () => [] },
    catalog: { type: Array, default: () => [] },
    gitRepos: { type: Array, default: () => [] },
    platformGitRepo: { type: String, default: '' },
    platformGitBranch: { type: String, default: 'main' },
    initialCollectionId: { type: String, default: null },
    apiFn: { type: Function, required: true },
  },
  emits: ['cancel', 'submit'],
  data() {
    return {
      errors: [],
      clusters: [],
      existingResources: [],
      cloneFrom: { clusterId: '', namespace: '' },
      form: {
        name: '',
        slug: '',
        description: '',
        collectionId: '',
        offeringId: '',
        charts: [],
        formValues: {},
        gitRepo: '',
        gitRepoId: '',
        gitBranch: 'main',
        gitPath: '',
        gitSecretName: 'platform-git-credentials',
        targetClustersText: '',
      },
    };
  },
  computed: {
    selectedOffering() {
      return this.offerings.find((o) => o.id === this.form.offeringId) || null;
    },
    filteredOfferings() {
      if (!this.form.collectionId) return this.offerings;
      return this.offerings.filter((o) => o.collectionId === this.form.collectionId);
    },
    collectionLabel() {
      const c = this.collections.find((x) => x.id === this.form.collectionId);
      return c ? c.label : '—';
    },
    hasFormSchema() {
      return !!(this.selectedOffering && this.selectedOffering.formSchema && this.selectedOffering.formSchema.length);
    },
    needsGitOps() {
      const o = this.selectedOffering;
      if (!o) return false;
      if (o.gitOps) return true;
      if (o.kind === 'helm' && (this.form.charts.length || (o.charts && o.charts.length))) return true;
      if (this.hasFormSchema) return true;
      return false;
    },
    effectiveCharts() {
      const o = this.selectedOffering;
      if (this.form.charts.length) return this.form.charts;
      return o && o.charts ? o.charts : [];
    },
    effectiveGitPath() {
      if (this.form.gitPath) return this.form.gitPath;
      return this.form.slug ? `environments/${this.form.slug}` : 'environments/…';
    },
    steps() {
      const s = [
        { name: 'basics', label: 'Basics', subtext: 'Name your environment', ready: SLUG_RE.test(this.form.slug) },
        { name: 'collection', label: 'Collection', subtext: 'Choose a catalog category', ready: !!this.form.collectionId },
        { name: 'offering', label: 'Offering', subtext: 'Pick what to request', ready: this.offeringReady },
        { name: 'configure', label: 'Configure', subtext: 'Set options', ready: this.configureReady },
      ];
      if (this.needsGitOps) {
        s.push({
          name: 'delivery',
          label: 'Delivery',
          subtext: 'Git & Fleet targets',
          ready: /^https?:\/\/.+/.test((this.form.gitRepo || '').trim()),
        });
      }
      s.push({ name: 'review', label: 'Review', subtext: 'Confirm and submit', ready: true });
      return s;
    },
    offeringReady() {
      if (!this.form.offeringId) return false;
      const o = this.selectedOffering;
      if (o && o.cloneFrom) {
        return !!(this.cloneFrom.clusterId && this.cloneFrom.namespace);
      }
      return true;
    },
    configureReady() {
      if (!this.hasFormSchema) return true;
      return this.selectedOffering.formSchema.every((f) => {
        if (!f.required) return true;
        const v = this.form.formValues[f.key];
        return v !== undefined && String(v).trim() !== '';
      });
    },
  },
  mounted() {
    this.resetForm();
    this.loadClusters();
  },
  methods: {
    resetForm() {
      this.form = {
        name: '',
        slug: '',
        description: '',
        collectionId: this.initialCollectionId || this.collections[0]?.id || '',
        offeringId: '',
        charts: [],
        formValues: {},
        gitRepo: this.platformGitRepo || '',
        gitRepoId: this.gitRepos[0]?.id || '',
        gitBranch: this.platformGitBranch || 'main',
        gitPath: '',
        gitSecretName: 'platform-git-credentials',
        targetClustersText: '',
      };
      this.cloneFrom = { clusterId: '', namespace: '' };
      this.applyGitRepoSelection();
    },
    slugifyName() {
      this.form.slug = (this.form.name || '')
        .toLowerCase()
        .replace(/[^a-z0-9-]+/g, '-')
        .replace(/^-+|-+$/g, '')
        .slice(0, 30);
      if (!this.form.gitPath && this.form.slug) {
        this.form.gitPath = `environments/${this.form.slug}`;
      }
    },
    collectionIcon(icon) {
      const map = { namespace: 'icon-namespace', cluster: 'icon-cluster', apps: 'icon-apps', vm: 'icon-vm', file: 'icon-file' };
      return map[icon] || 'icon-folder';
    },
    onCollectionChange() {
      this.form.offeringId = '';
      const first = this.filteredOfferings[0];
      if (first) this.form.offeringId = first.id;
      this.onOfferingChange();
    },
    onOfferingChange() {
      const o = this.selectedOffering;
      if (o && o.charts) this.form.charts = [...o.charts];
      if (o && o.formSchema) {
        const vals = {};
        o.formSchema.forEach((f) => { vals[f.key] = f.default || ''; });
        this.form.formValues = vals;
      }
    },
    setFormValue(key, val) {
      this.form.formValues = { ...this.form.formValues, [key]: val };
    },
    applyGitRepoSelection() {
      const repo = this.gitRepos.find((r) => r.id === this.form.gitRepoId);
      if (repo) {
        this.form.gitRepo = repo.url;
        if (repo.branch) this.form.gitBranch = repo.branch;
        if (repo.secretName) this.form.gitSecretName = repo.secretName;
      }
    },
    async loadClusters() {
      try {
        const data = await this.apiFn('GET', '/api/portal/clusters');
        this.clusters = data.clusters || [];
      } catch (_) {
        this.clusters = [];
      }
    },
    async loadExistingResources() {
      this.cloneFrom.namespace = '';
      if (!this.cloneFrom.clusterId) {
        this.existingResources = [];
        return;
      }
      try {
        const data = await this.apiFn('GET', `/api/portal/existing-resources?cluster=${encodeURIComponent(this.cloneFrom.clusterId)}`);
        this.existingResources = data.resources || [];
      } catch (_) {
        this.existingResources = [];
      }
    },
    submit() {
      this.errors = [];
      if (!SLUG_RE.test(this.form.slug)) {
        this.errors = ['Invalid environment name'];
        return;
      }
      const payload = {
        name: this.form.slug,
        displayName: this.form.name,
        description: this.form.description,
        offeringId: this.form.offeringId,
        collectionId: this.form.collectionId,
        formValues: this.form.formValues,
        charts: this.form.charts,
        gitRepo: this.needsGitOps ? this.form.gitRepo.trim() : '',
        gitRepoId: this.form.gitRepoId,
        gitBranch: this.form.gitBranch,
        gitPath: this.form.gitPath || this.effectiveGitPath,
        gitSecretName: this.form.gitSecretName,
        targetClusters: this.form.targetClustersText
          ? this.form.targetClustersText.split(',').map((s) => s.trim()).filter(Boolean)
          : [],
      };
      if (this.selectedOffering && this.selectedOffering.cloneFrom && this.cloneFrom.namespace) {
        payload.cloneFromRef = {
          clusterId: this.cloneFrom.clusterId,
          namespace: this.cloneFrom.namespace,
        };
      }
      this.$emit('submit', payload);
    },
  },
};
</script>

<style lang="scss" scoped>
.request-wizard {
  margin-top: 8px;
  width: 100%;
  max-width: min(1080px, 100%);
  margin-left: auto;
  margin-right: auto;
  padding: 0 clamp(8px, 1.5vw, 20px);
  box-sizing: border-box;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

:deep(.outer-container) {
  position: relative;
  flex: 1;
  min-height: 0;
  padding-bottom: 4.5rem;
}

:deep(.step-container) {
  padding-top: 12px;
}

:deep(#wizard-footer-controls.controls-row) {
  position: sticky;
  bottom: 0;
  left: 0;
  right: 0;
  width: 100%;
  margin-left: 0 !important;
  margin-right: 0 !important;
  padding: 12px 0;
  justify-content: flex-start;
  align-items: center;
  gap: 10px;
  box-sizing: border-box;
  z-index: 2;
}

:deep(.controls-steps) {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  margin-left: auto;

  .btn {
    margin-left: 0 !important;
  }
}

:deep(.footer-error) {
  margin-top: 0;
  margin-bottom: 12px;
}

@media (max-width: 1440px) {
  .request-wizard {
    max-width: min(960px, 100%);
  }

  :deep(.steps) {
    margin: 0 8px !important;
  }

  :deep(.steps li.step .controls > span:last-of-type) {
    display: none;
  }
}

@media (max-width: 1100px) {
  .request-wizard {
    max-width: 100%;
    padding: 0 12px;
  }
}

@media (max-width: 1200px) {
  :deep(.header .step-sequence) {
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
  }

  :deep(.steps) {
    min-width: max-content;
    justify-content: flex-start !important;
    margin: 0 !important;
    padding-bottom: 4px;
  }
}

@media (max-width: 768px) {
  :deep(#wizard-footer-controls.controls-row) {
    flex-wrap: wrap;
  }

  :deep(.controls-steps) {
    width: 100%;
    margin-left: 0;
    justify-content: stretch;
  }

  :deep(.controls-steps .btn) {
    flex: 1 1 auto;
    min-width: 0;
  }

  :deep(#wizard-footer-controls.controls-row > .btn.role-secondary) {
    flex: 1 1 100%;
  }
}

.step-lead { color: var(--muted); margin: 0 0 16px; font-size: 13px; }
.hint { font-size: 12px; color: var(--muted); margin: 4px 0 12px; }
.label { display: block; font-weight: 600; margin-bottom: 4px; font-size: 13px; }
.input-sm { padding: 6px 10px; border: 1px solid var(--border); border-radius: 4px; background: var(--input-bg); color: var(--body-text); }
.input-wide { width: 100%; max-width: 480px; }
.dp-card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 12px; }
.dp-card { display: flex; flex-direction: column; gap: 6px; padding: 14px; border: 1px solid var(--border); border-radius: 6px; cursor: pointer; background: var(--body-bg); align-items: flex-start; }
.dp-card.selected { border-color: var(--primary); box-shadow: 0 0 0 1px var(--primary); }
.dp-card-title { font-weight: 600; }
.dp-card-desc, .dp-card-detail { font-size: 12px; color: var(--muted); }
.dp-chart-list { border: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 8px; }
.dp-chart-option { display: flex; gap: 10px; padding: 10px; border: 1px solid var(--border); border-radius: 4px; cursor: pointer; }
.dp-chart-option.selected { border-color: var(--primary); }
.dp-form-grid { display: grid; gap: 12px; max-width: 520px; }
.dp-clone-panel { margin-top: 16px; padding: 12px; border: 1px dashed var(--border); border-radius: 6px; }
.dp-review-dl { display: grid; grid-template-columns: 120px 1fr; gap: 8px 16px; font-size: 13px; }
.dp-review-dl dt { font-weight: 600; color: var(--muted); }
.dp-chip { display: inline-block; padding: 2px 8px; margin: 2px; border-radius: 4px; background: var(--box-bg); font-size: 12px; }
.dp-form-review { margin: 0; padding-left: 16px; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0,0,0,0); }
.required { color: var(--error, #c00); }
.muted { color: var(--muted); }
</style>
