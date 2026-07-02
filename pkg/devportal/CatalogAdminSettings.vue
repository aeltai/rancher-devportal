<template>
  <section class="dp-admin-settings">
    <header class="dp-admin-settings-head">
      <div class="dp-admin-settings-title">
        <i class="icon icon-cog" aria-hidden="true" />
        <div>
          <h2>Geeko-Ops catalog</h2>
          <p>Configure collections, offerings, Git connections, and approval rules.</p>
        </div>
      </div>
      <div class="dp-admin-settings-toolbar">
        <div class="dp-mode-toggle">
          <button
            v-for="opt in modeOptions"
            :key="opt.value"
            type="button"
            :class="['btn', mode === opt.value ? 'role-primary' : 'role-secondary']"
            @click="mode = opt.value"
          >
            {{ opt.label }}
          </button>
        </div>
        <button v-if="!embedded" class="btn role-tertiary" type="button" aria-label="Close settings" @click="$emit('close')">
          <i class="icon icon-close" />
        </button>
      </div>
    </header>

    <div class="dp-admin-settings-body">
      <div v-if="mode === 'yaml'" class="dp-admin-yaml">
        <DpBanner
          color="info"
          label="Edit platform.yaml directly. Switch back to Visual editor to validate structure before saving."
        />
        <DpInput
          v-model:value="yamlText"
          type="multiline"
          label="Platform configuration (YAML)"
          :rows="24"
          class="mt-10"
        />
      </div>

      <DpTabs
        v-else
        v-model="adminTab"
        :tabs="adminTabs"
        class="dp-admin-tabbed"
      >
        <template #overview>
          <div class="dp-admin-overview">
            <DpBanner
              color="info"
              label="The catalog is stored in platform.yaml (ConfigMap platform-config). Users pick Collection → Offering in the request wizard; the operator renders Git + Fleet manifests."
            />
            <div class="dp-admin-overview-grid">
              <article class="dp-admin-overview-card">
                <h3>1. Collections</h3>
                <p>Categories in the wizard (Namespaces, VMs, Helm, …). Purely organizational.</p>
              </article>
              <article class="dp-admin-overview-card">
                <h3>2. Offerings</h3>
                <p>What users can request. Each has a <strong>kind</strong> that controls how manifests are built.</p>
                <ul>
                  <li><strong>namespace</strong> — env-* namespace (+ optional clone)</li>
                  <li><strong>helm</strong> — Fleet chart IDs</li>
                  <li><strong>crd</strong> — form fields → CR spec paths</li>
                  <li><strong>generic</strong> — Go-template YAML you write</li>
                </ul>
              </article>
              <article class="dp-admin-overview-card">
                <h3>3. CRD offerings</h3>
                <p>Use the <strong>CRD builder</strong> on offerings with kind <code>crd</code>: load CRDs from a cluster, pick one, and <strong>Generate form from schema</strong> to auto-fill fields from OpenAPI.</p>
              </article>
              <article class="dp-admin-overview-card">
                <h3>4. Where defaults live</h3>
                <p>Sample catalog ships in the repo (<code>config/platform.yaml</code>). Your cluster ConfigMap may still have the legacy <code>templates/charts</code> only — use <strong>Import bundled catalog</strong> below.</p>
              </article>
            </div>
            <div class="dp-admin-overview-actions">
              <button class="btn role-secondary" type="button" @click="importBundledCatalog">
                <i class="icon icon-download" /> Import bundled catalog
              </button>
              <button class="btn role-tertiary" type="button" @click="mode = 'yaml'">
                Open full YAML editor
              </button>
            </div>
            <p v-if="importMsg" class="dp-admin-hint">{{ importMsg }}</p>
          </div>
        </template>

        <template #collections>
          <div class="dp-admin-tab-intro">
            <p>Group offerings into catalog sections shown in the request wizard.</p>
            <button class="btn role-secondary" type="button" @click="addCollection">
              <i class="icon icon-circle-plus" /> Add collection
            </button>
          </div>

          <div v-if="!config.collections.length" class="dp-admin-empty">
            <i class="icon icon-folder" />
            <p>No collections yet. Add one to organize your catalog.</p>
          </div>

          <article
            v-for="(col, idx) in config.collections"
            :key="col.id || `col-${idx}`"
            class="dp-admin-card"
          >
            <div class="dp-admin-card-head">
              <h3>{{ col.label || col.id || 'New collection' }}</h3>
              <button class="btn role-tertiary" type="button" @click="config.collections.splice(idx, 1)">
                Remove
              </button>
            </div>
            <div class="row">
              <div class="col span-3">
                <DpInput v-model:value="col.id" label="ID" placeholder="compute" />
              </div>
              <div class="col span-4">
                <DpInput v-model:value="col.label" label="Label" placeholder="Compute" />
              </div>
              <div class="col span-2">
                <DpInput v-model:value="col.weight" label="Weight" type="number" placeholder="50" />
              </div>
              <div class="col span-12">
                <DpInput
                  v-model:value="col.description"
                  label="Description"
                  placeholder="Short summary for the request wizard"
                />
              </div>
            </div>
          </article>
        </template>

        <template #offerings>
          <div class="dp-admin-tab-intro">
            <p>Define what users can request — namespaces, Helm charts, CRDs, or generic manifests.</p>
            <button class="btn role-secondary" type="button" @click="addOffering">
              <i class="icon icon-circle-plus" /> Add offering
            </button>
          </div>

          <div v-if="!config.offerings.length" class="dp-admin-empty">
            <i class="icon icon-apps" />
            <p>No offerings yet. Add one and assign it to a collection.</p>
          </div>

          <DpCollapse
            v-for="(off, idx) in config.offerings"
            :key="off.id || `off-${idx}`"
            :title="offeringCardTitle(off)"
            :is-collapsed="collapsedOfferings[idx] === true"
            class="dp-admin-offering-card"
            @toggleCollapse="toggleOffering(idx, $event)"
          >
            <div class="dp-admin-card-actions">
              <button class="btn role-tertiary" type="button" @click="config.offerings.splice(idx, 1)">
                Remove offering
              </button>
            </div>
            <div class="row">
              <div class="col span-3">
                <DpInput v-model:value="off.id" label="ID" placeholder="team-namespace" />
              </div>
              <div class="col span-3">
                <DpSelect
                  v-model:value="off.collectionId"
                  label="Collection"
                  :options="collectionOptions"
                  option-key="value"
                  option-label="label"
                />
              </div>
              <div class="col span-3">
                <DpInput v-model:value="off.label" label="Label" placeholder="Team namespace" />
              </div>
              <div class="col span-3">
                <DpSelect
                  v-model:value="off.kind"
                  label="Kind"
                  :options="kindOptions"
                  option-key="value"
                  option-label="label"
                />
              </div>
              <div class="col span-12">
                <DpInput v-model:value="off.description" label="Description" />
              </div>
            </div>

            <div v-if="off.kind === 'helm'" class="dp-admin-subsection">
              <DpInput
                :value="(off.charts || []).join(', ')"
                label="Chart IDs"
                placeholder="rancher-monitoring, cert-manager"
                tooltip="Comma-separated chart IDs from the platform catalog"
                @update:value="off.charts = splitCsv($event)"
              />
            </div>

            <div v-if="off.kind === 'crd'" class="dp-admin-subsection">
              <CrdOfferingBuilder
                :api-version="off.apiVersion"
                :kind-name="off.kindName"
                :target-cluster="off.targetCluster"
                :form-schema="off.formSchema || []"
                :cluster-options="clusterOptions"
                :api-fn="apiFn"
                @update:api-version="off.apiVersion = $event"
                @update:kind-name="off.kindName = $event"
                @update:target-cluster="off.targetCluster = $event"
                @update:form-schema="off.formSchema = $event"
              />
            </div>

            <div v-if="off.kind === 'generic'" class="dp-admin-subsection">
              <DpInput
                v-model:value="off.manifestTemplate"
                type="multiline"
                label="Manifest template"
                :rows="8"
              />
              <FieldBuilder v-model="off.formSchema" />
            </div>

            <div v-if="off.kind === 'namespace'" class="dp-admin-subsection">
              <DpCheckbox v-model:value="off.cloneFrom" label="Allow clone from existing namespace" />
            </div>

            <div class="dp-admin-checks">
              <DpCheckbox v-model:value="off.gitOps" label="GitOps delivery" />
              <DpCheckbox v-model:value="off.requiresApproval" label="Requires admin approval" />
            </div>
          </DpCollapse>
        </template>

        <template #git>
          <div class="dp-admin-tab-intro">
            <p>Fleet Git repositories used when provisioning approved environments.</p>
            <button class="btn role-secondary" type="button" @click="addGitRepo">
              <i class="icon icon-git" /> Add Git connection
            </button>
          </div>

          <DpBanner
            v-if="gitTestMsg"
            :color="gitTestOk ? 'success' : gitTestMsg === 'Testing…' ? 'info' : 'error'"
            :label="gitTestMsg"
            class="mb-10"
          />

          <div v-if="!config.git.repos.length" class="dp-admin-empty">
            <i class="icon icon-git" />
            <p>No Git connections configured.</p>
          </div>

          <article
            v-for="(repo, idx) in config.git.repos"
            :key="repo.id || `git-${idx}`"
            class="dp-admin-card"
          >
            <div class="dp-admin-card-head">
              <h3>{{ repo.name || repo.id || 'New Git connection' }}</h3>
              <div class="dp-admin-card-head-actions">
                <button class="btn role-secondary" type="button" @click="testGit(repo)">Test connection</button>
                <button class="btn role-tertiary" type="button" @click="config.git.repos.splice(idx, 1)">Remove</button>
              </div>
            </div>
            <div class="row">
              <div class="col span-3">
                <DpInput v-model:value="repo.id" label="ID" placeholder="platform-fleet" />
              </div>
              <div class="col span-4">
                <DpInput v-model:value="repo.name" label="Display name" placeholder="Platform Fleet" />
              </div>
              <div class="col span-2">
                <DpInput v-model:value="repo.branch" label="Branch" placeholder="main" />
              </div>
              <div class="col span-3">
                <DpInput
                  v-model:value="repo.secretName"
                  label="Secret name"
                  placeholder="platform-git-credentials"
                  tooltip="Kubernetes secret in devportal-system with Git credentials"
                />
              </div>
              <div class="col span-12">
                <DpInput
                  v-model:value="repo.url"
                  label="Repository URL"
                  placeholder="https://github.com/org/fleet.git"
                />
              </div>
            </div>
          </article>
        </template>

        <template #approval>
          <div class="dp-admin-tab-intro">
            <p>Control approval gates and CRD discovery for custom resource offerings.</p>
          </div>

          <div class="dp-admin-card">
            <h3 class="dp-admin-subheading">Approval policy</h3>
            <div class="dp-admin-checks dp-admin-checks--stacked">
              <DpCheckbox v-model:value="config.approval.chartsRequireApproval" label="Helm charts require approval" />
              <DpCheckbox
                v-model:value="config.approval.customResourcesRequireApproval"
                label="Custom resources require approval"
              />
            </div>
          </div>

          <div class="dp-admin-card">
            <h3 class="dp-admin-subheading">CRD discovery</h3>
            <DpCheckbox v-model:value="config.crdDiscovery.enabled" label="Enable CRD discovery for form builders" />
            <div class="row mt-10">
              <div class="col span-6">
                <DpSelect
                  v-model:value="config.crdDiscovery.clusters"
                  label="Discovery cluster"
                  :options="clusterOptions"
                  option-key="value"
                  option-label="label"
                  :disabled="!config.crdDiscovery.enabled"
                />
              </div>
            </div>
          </div>
        </template>
      </DpTabs>
    </div>

    <footer class="dp-admin-settings-footer">
      <button class="btn role-secondary" type="button" @click="$emit('reload')">
        <i class="icon icon-refresh" /> Reload
      </button>
      <button class="btn role-primary" type="button" :disabled="saving" @click="save">
        {{ saving ? 'Saving…' : 'Save config' }}
      </button>
    </footer>
  </section>
</template>

<script>
import DpTabs from './DpTabs.vue';
import DpCollapse from './DpCollapse.vue';
import DpSelect from './DpSelect.vue';
import DpInput from './DpInput.vue';
import DpCheckbox from './DpCheckbox.vue';
import DpBanner from './DpBanner.vue';
import FieldBuilder from './FieldBuilder.vue';
import CrdOfferingBuilder from './CrdOfferingBuilder.vue';

const DEFAULT_CONFIG = () => ({
  defaults: {
    namespace: 'devportal-system',
    fleetNamespace: 'fleet-default',
    gitSecretName: 'platform-git-credentials',
    gitBranch: 'main',
    gitPathPrefix: 'environments',
  },
  git: { mode: 'single', defaultRepo: '', repos: [] },
  collections: [],
  offerings: [],
  templates: [],
  charts: [],
  crdDiscovery: { enabled: true, clusters: 'local', excludeGroups: [] },
  approval: { chartsRequireApproval: true, customResourcesRequireApproval: true },
});

export default {
  name: 'CatalogAdminSettings',
  components: {
    DpTabs,
    DpCollapse,
    DpInput,
    DpSelect,
    DpCheckbox,
    DpBanner,
    FieldBuilder,
    CrdOfferingBuilder,
  },
  props: {
    initialYaml: { type: String, default: '' },
    initialConfig: { type: Object, default: null },
    apiFn: { type: Function, required: true },
    saving: { type: Boolean, default: false },
    embedded: { type: Boolean, default: false },
  },
  emits: ['close', 'reload', 'save'],
  data() {
    return {
      mode: 'visual',
      adminTab: 'overview',
      adminTabs: [
        { name: 'overview', label: 'How it works' },
        { name: 'collections', label: 'Collections' },
        { name: 'offerings', label: 'Offerings' },
        { name: 'git', label: 'Git connections' },
        { name: 'approval', label: 'Approval & CRD' },
      ],
      yamlText: this.initialYaml,
      config: this.initialConfig ? JSON.parse(JSON.stringify(this.initialConfig)) : DEFAULT_CONFIG(),
      clusters: [],
      gitTestMsg: '',
      gitTestOk: false,
      collapsedOfferings: {},
      importMsg: '',
      modeOptions: [
        { label: 'Visual editor', value: 'visual' },
        { label: 'YAML', value: 'yaml' },
      ],
      kindOptions: [
        { label: 'Namespace', value: 'namespace' },
        { label: 'Cluster', value: 'cluster' },
        { label: 'Helm charts', value: 'helm' },
        { label: 'Custom resource (CRD)', value: 'crd' },
        { label: 'Generic manifest', value: 'generic' },
      ],
    };
  },
  computed: {
    collectionOptions() {
      return (this.config.collections || []).map((c) => ({
        label: c.label || c.id || 'Unnamed',
        value: c.id,
      }));
    },
    clusterOptions() {
      return [
        { label: 'local (management cluster)', value: 'local' },
        ...this.clusters.map((c) => ({ label: c.name || c.id, value: c.id })),
      ];
    },
  },
  watch: {
    initialYaml(v) { this.yamlText = v; },
    initialConfig(v) {
      if (v) this.config = JSON.parse(JSON.stringify(v));
    },
    mode(v) {
      if (v === 'yaml') this.switchToYaml();
    },
  },
  mounted() {
    this.loadClusters();
    if (!this.config.git) this.config.git = { repos: [] };
    if (!this.config.collections) this.config.collections = [];
    if (!this.config.offerings) this.config.offerings = [];
    this.config.offerings.forEach((_, idx) => {
      this.collapsedOfferings[idx] = idx > 0;
    });
  },
  methods: {
    splitCsv(value) {
      return String(value || '').split(',').map((s) => s.trim()).filter(Boolean);
    },
    offeringCardTitle(off) {
      const kind = this.kindOptions.find((k) => k.value === off.kind)?.label || off.kind;
      const title = off.label || off.id || 'New offering';
      return kind ? `${title} · ${kind}` : title;
    },
    toggleOffering(idx, collapsed) {
      this.collapsedOfferings = { ...this.collapsedOfferings, [idx]: collapsed };
    },
    async importBundledCatalog() {
      if (!window.confirm('Replace the current catalog with the bundled sample (collections, offerings, Harvester VM example, etc.)? Unsaved changes will be lost.')) {
        return;
      }
      this.importMsg = 'Loading bundled catalog…';
      try {
        const data = await this.apiFn('GET', '/api/portal/platform-config/bundle');
        if (data.config) {
          this.config = JSON.parse(JSON.stringify(data.config));
        }
        if (data.yaml) this.yamlText = data.yaml;
        this.importMsg = `Imported ${(data.collections || []).length} collections and ${(data.offerings || []).length} offerings from the platform bundle. Save config to apply to the cluster.`;
      } catch (e) {
        this.importMsg = e.message || 'Import failed — rebuild devportal-backend to include the bundle endpoint.';
      }
    },
    async switchToYaml() {
      try {
        const data = await this.apiFn('POST', '/api/portal/platform-config/serialize', this.config);
        this.yamlText = data.yaml || '';
      } catch (_) {
        this.yamlText = this.initialYaml;
      }
    },
    addCollection() {
      this.config.collections.push({ id: '', label: '', description: '', weight: 50 });
    },
    addOffering() {
      const idx = this.config.offerings.length;
      this.config.offerings.push({
        id: '',
        collectionId: this.config.collections[0]?.id || '',
        label: '',
        kind: 'namespace',
        gitOps: false,
        requiresApproval: false,
        formSchema: [],
      });
      this.collapsedOfferings = { ...this.collapsedOfferings, [idx]: false };
    },
    addGitRepo() {
      if (!this.config.git.repos) this.config.git.repos = [];
      this.config.git.repos.push({
        id: '',
        name: '',
        url: '',
        branch: 'main',
        secretName: 'platform-git-credentials',
      });
    },
    async loadClusters() {
      try {
        const data = await this.apiFn('GET', '/api/portal/clusters');
        this.clusters = data.clusters || [];
      } catch (_) {}
    },
    async testGit(repo) {
      this.gitTestMsg = 'Testing…';
      this.gitTestOk = false;
      try {
        await this.apiFn('POST', '/api/portal/git/test-connection', {
          url: repo.url,
          branch: repo.branch || 'main',
          secretName: repo.secretName,
        });
        this.gitTestMsg = `Connection OK: ${repo.url}`;
        this.gitTestOk = true;
      } catch (e) {
        this.gitTestMsg = e.message || 'Connection failed';
        this.gitTestOk = false;
      }
    },
    save() {
      if (this.mode === 'yaml') {
        this.$emit('save', { yaml: this.yamlText });
      } else {
        this.$emit('save', { config: this.config });
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.dp-admin-settings {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  border-top: 1px solid var(--border);
  background: var(--body-bg);
  overflow: hidden;
}

.dp-admin-settings-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--sortable-table-header-bg, var(--box-bg));
}

.dp-admin-settings-title {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  min-width: 0;

  .icon {
    font-size: 1.4em;
    color: var(--primary);
    margin-top: 2px;
  }

  h2 {
    margin: 0 0 4px;
    font-size: 1em;
    font-weight: 600;
  }

  p {
    margin: 0;
    font-size: 0.82em;
    color: var(--muted);
    line-height: 1.4;
  }
}

.dp-admin-settings-toolbar {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-shrink: 0;
}

.dp-mode-toggle {
  display: flex;
  gap: 4px;
}

.dp-admin-settings-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 0 16px 16px;
}

.dp-admin-tabbed {
  margin-top: 0;

  :deep(.tabs.horizontal) {
    border-bottom: 1px solid var(--border);
    margin-bottom: 0;
  }

  :deep(.tab-container) {
    padding: 16px 0 0;
  }
}

.dp-admin-tab-intro {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin: 12px 0 16px;

  p {
    margin: 0;
    font-size: 0.85em;
    color: var(--muted);
    line-height: 1.5;
    max-width: 640px;
  }
}

.dp-admin-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px 16px;
  border: 1px dashed var(--border);
  border-radius: 4px;
  color: var(--muted);
  text-align: center;

  .icon {
    font-size: 1.6em;
    opacity: 0.6;
  }

  p {
    margin: 0;
    font-size: 0.85em;
  }
}

.dp-admin-card {
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--body-bg);
  padding: 14px 16px;
  margin-bottom: 12px;
}

.dp-admin-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;

  h3 {
    margin: 0;
    font-size: 0.92em;
    font-weight: 600;
  }
}

.dp-admin-card-head-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.dp-admin-card-actions {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 8px;
}

.dp-admin-offering-card {
  margin-bottom: 12px;
}

.dp-admin-subsection {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid var(--border);
}

.dp-admin-subheading {
  margin: 0 0 12px;
  font-size: 0.88em;
  font-weight: 600;
}

.dp-admin-checks {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-top: 12px;
}

.dp-admin-checks--stacked {
  flex-direction: column;
  gap: 10px;
}

.dp-admin-settings-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid var(--border);
  background: var(--sortable-table-header-bg, var(--box-bg));
}

.dp-admin-yaml {
  padding-top: 12px;
}

.dp-admin-overview {
  padding-top: 12px;
}

.dp-admin-overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
  margin: 16px 0;
}

.dp-admin-overview-card {
  border: 1px solid var(--border);
  border-radius: 4px;
  padding: 14px;
  background: var(--body-bg);

  h3 {
    margin: 0 0 8px;
    font-size: 0.9em;
  }

  p, ul {
    margin: 0;
    font-size: 0.82em;
    color: var(--muted);
    line-height: 1.5;
  }

  ul {
    margin-top: 8px;
    padding-left: 18px;
  }

  code {
    font-size: 0.92em;
  }
}

.dp-admin-overview-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.dp-admin-hint {
  margin-top: 10px;
  font-size: 0.82em;
  color: var(--muted);
}

.dp-admin-crd-discover {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-end;
}

:deep(.collapsible-card) {
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--body-bg);
  min-width: 0;
  max-width: 100%;
}
</style>
