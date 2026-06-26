<template>
  <div class="devportal-page">
    <div v-if="error" class="banner error">
      {{ error }}
      <button class="dismiss" type="button" @click="error = ''">&times;</button>
    </div>
    <div v-if="message" class="banner success">
      {{ message }}
      <button class="dismiss" type="button" @click="message = ''">&times;</button>
    </div>
    <div v-if="loading" class="loading-bar" />

    <div class="dp-panel">
      <header class="dp-header">
        <div class="dp-header-left">
          <i class="icon icon-compass dp-header-icon" aria-hidden="true" />
          <div class="dp-header-text">
            <h1>Developer Portal</h1>
            <p>Self-service environments — namespaces, Fleet GitOps, and operators.</p>
            <div v-if="authUser" class="dp-meta">
              <span class="dp-badge user">{{ authUser.displayName || authUser.username }}</span>
              <span v-if="isAdmin" class="dp-badge admin">Administrator</span>
              <span v-if="stackInfo.recommended" class="dp-badge muted">{{ stackInfo.recommended }}</span>
            </div>
          </div>
        </div>
        <button class="btn role-primary" type="button" @click="startWizard">
          <i class="icon icon-circle-plus" /> Request environment
        </button>
      </header>

      <div v-if="showWizard" class="dp-wizard">
        <div class="dp-wizard-header">
          <h2>New environment</h2>
          <button class="btn role-tertiary xs" type="button" @click="cancelWizard">
            <i class="icon icon-close" />
          </button>
        </div>
        <div class="dp-steps" role="list">
          <span role="listitem" :class="['dp-step-pill', { active: wizardStep === 1, done: wizardStep > 1 }]">1. Name</span>
          <span role="listitem" :class="['dp-step-pill', { active: wizardStep === 2, done: wizardStep > 2 }]">2. Template</span>
          <span role="listitem" :class="['dp-step-pill', { active: wizardStep === 3, done: wizardStep > 3 }]">3. Charts</span>
          <span role="listitem" :class="['dp-step-pill', { active: wizardStep === 4 }]">4. Review</span>
        </div>

        <div v-if="wizardStep === 1" class="dp-step">
          <label class="label">Environment name</label>
          <input v-model="form.name" class="input-sm" type="text" placeholder="my-team-dev" @input="slugifyName" />
          <p class="hint">Lowercase, numbers, hyphens. Namespace: <code>env-{{ form.slug || '…' }}</code></p>
          <label class="label">Description</label>
          <input v-model="form.description" class="input-sm" type="text" placeholder="Optional" />
        </div>

        <div v-if="wizardStep === 2" class="dp-step dp-step-select">
          <p class="step-lead">Pick an environment profile — this sets namespace guardrails and whether Fleet GitOps is provisioned.</p>
          <fieldset class="dp-template-list">
            <legend class="sr-only">Environment template</legend>
            <label
              v-for="t in templates"
              :key="t.id"
              :class="['dp-template-option', { selected: form.template === t.id }]"
            >
              <input v-model="form.template" type="radio" class="dp-template-radio" :value="t.id">
              <span class="dp-template-icon" aria-hidden="true">
                <i :class="['icon', templateIcon(t.id)]" />
              </span>
              <span class="dp-template-body">
                <span class="dp-template-title">{{ t.label }}</span>
                <span class="dp-template-desc">{{ t.description }}</span>
                <span v-if="t.detail" class="dp-template-detail">{{ t.detail }}</span>
              </span>
            </label>
          </fieldset>
        </div>

        <div v-if="wizardStep === 3" class="dp-step dp-step-select">
          <p class="step-lead">Optional Helm charts — installed as Fleet bundles under your environment Git path.</p>
          <fieldset class="dp-chart-list">
            <legend class="sr-only">Helm charts</legend>
            <label
              v-for="c in catalog"
              :key="c.id"
              :class="['dp-chart-option', { selected: form.charts.includes(c.id) }]"
            >
              <input v-model="form.charts" type="checkbox" class="dp-chart-checkbox" :value="c.id">
              <span class="dp-chart-body">
                <span class="dp-chart-title">{{ c.name }}</span>
                <span class="dp-chart-tag">{{ c.category }}</span>
                <span class="dp-chart-desc">{{ c.description }}</span>
              </span>
            </label>
          </fieldset>
        </div>

        <div v-if="wizardStep === 4" class="dp-step dp-review">
          <dl>
            <dt>Name</dt><dd>{{ form.name }}</dd>
            <dt>Template</dt><dd>{{ templateLabel }}</dd>
            <dt>Charts</dt>
            <dd>
              <span v-if="!form.charts.length" class="muted">None</span>
              <span v-for="id in form.charts" :key="id" class="dp-chip">{{ chartName(id) }}</span>
            </dd>
          </dl>
        </div>

        <div class="dp-wizard-actions">
          <button v-if="wizardStep > 1" class="btn role-secondary" type="button" @click="wizardStep--">Back</button>
          <button v-if="wizardStep < 4" class="btn role-primary" type="button" :disabled="!canNext" @click="wizardStep++">Next</button>
          <button v-if="wizardStep === 4" class="btn role-primary" type="button" :disabled="submitting" @click="submitRequest">
            {{ submitting ? 'Submitting…' : 'Submit request' }}
          </button>
        </div>
      </div>

      <section class="dp-section">
        <div class="dp-section-head">
          <h2>
            <i class="icon icon-namespace" />
            {{ isAdmin ? 'All platform requests' : 'My environments' }}
          </h2>
          <button class="btn role-tertiary xs" type="button" :disabled="loading" @click="loadRequests">
            <i class="icon icon-refresh" /> Refresh
          </button>
        </div>
        <p v-if="platformGitRepo" class="dp-git-hint">
          <i class="icon icon-fleet" />
          GitOps repo: <code>{{ platformGitRepo }}</code>
          <span v-if="platformGitBranch"> · branch <code>{{ platformGitBranch }}</code></span>
          <span class="muted"> — future PRs land under <code>environments/&lt;name&gt;/</code></span>
        </p>
        <p class="dp-table-hint">
          <i class="icon icon-info" />
          Click a row or <strong>View manifest</strong> to see the PlatformRequest YAML, Fleet GitRepo bundles, and GitOps paths.
        </p>
        <table v-if="requests.length" class="dp-table dp-table-requests">
          <thead>
            <tr>
              <th class="col-expand" />
              <th>Name</th>
              <th v-if="isAdmin">Requester</th>
              <th>Phase</th>
              <th>Namespace</th>
              <th>Template</th>
              <th>Charts</th>
              <th>Created</th>
              <th class="col-actions">Manifest</th>
            </tr>
          </thead>
          <tbody v-for="r in requests" :key="r.crName || r.name">
            <tr
              class="dp-request-row"
              :class="{ expanded: expandedCrName === (r.crName || r.name) }"
              @click="toggleRequestDetail(r)"
            >
              <td class="col-expand">
                <i :class="['icon', expandedCrName === (r.crName || r.name) ? 'icon-chevron-down' : 'icon-chevron-right']" />
              </td>
              <td class="name">
                <strong>{{ r.displayName || r.name }}</strong>
                <code v-if="isAdmin" class="cr-name">{{ r.crName }}</code>
              </td>
              <td v-if="isAdmin"><code>{{ r.requester || '—' }}</code></td>
              <td><span :class="['phase', r.phase]">{{ r.phase || '—' }}</span></td>
              <td><code>{{ r.namespace || '—' }}</code></td>
              <td><code>{{ r.template || '—' }}</code></td>
              <td>{{ (r.charts || []).join(', ') || '—' }}</td>
              <td>{{ formatDate(r.createdAt || r.created) }}</td>
              <td class="col-actions">
                <button
                  class="btn role-tertiary xs"
                  type="button"
                  @click.stop="openRequestDetail(r)"
                >
                  <i class="icon icon-file" /> View manifest
                </button>
              </td>
            </tr>
            <tr v-if="expandedCrName === (r.crName || r.name)" class="dp-request-detail-row">
              <td :colspan="isAdmin ? 9 : 8">
                <div class="dp-request-detail">
                  <div v-if="r.description" class="dp-detail-banner">
                    <strong>Description:</strong> {{ r.description }}
                  </div>
                  <div v-if="r.message" class="dp-detail-banner">
                    <strong>Status:</strong> {{ r.message }}
                  </div>
                  <div v-if="r.pullRequestHint" class="dp-detail-banner muted">
                    <i class="icon icon-github" /> {{ r.pullRequestHint }}
                  </div>

                  <div class="dp-detail-grid">
                    <div class="dp-detail-panel">
                      <h3><i class="icon icon-file" /> PlatformRequest manifest</h3>
                      <pre class="dp-yaml"><code>{{ r.manifestYaml || '—' }}</code></pre>
                    </div>
                    <div class="dp-detail-panel">
                      <h3><i class="icon icon-fleet" /> Fleet &amp; cluster resources</h3>
                      <table v-if="r.fleetResources && r.fleetResources.length" class="dp-fleet-table">
                        <thead>
                          <tr>
                            <th>Kind</th>
                            <th>Name</th>
                            <th>Namespace</th>
                            <th>Git path</th>
                            <th>Phase</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr v-for="(f, idx) in r.fleetResources" :key="idx">
                            <td><code>{{ f.kind }}</code></td>
                            <td>{{ f.name }}</td>
                            <td><code>{{ f.namespace || '—' }}</code></td>
                            <td>
                              <code v-if="f.path">{{ f.path }}</code>
                              <span v-else class="muted">—</span>
                            </td>
                            <td><span :class="['fleet-phase', f.phase]">{{ f.phase }}</span></td>
                          </tr>
                        </tbody>
                      </table>
                      <p v-else class="empty">No Fleet resources planned for this template.</p>
                      <p v-if="r.gitRepoUrl" class="dp-git-meta">
                        Repo: <code>{{ r.gitRepoUrl }}</code>
                        <span v-if="r.gitBranch"> · {{ r.gitBranch }}</span>
                        <span v-if="r.gitPath"> · path <code>{{ r.gitPath }}</code></span>
                      </p>
                    </div>
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else-if="!loading" class="empty">
          {{ isAdmin ? 'No platform requests yet.' : 'No environments yet. Use ' }}
          <strong v-if="!isAdmin">Request environment</strong>
          {{ isAdmin ? '' : ' to provision one.' }}
        </p>
      </section>

      <section class="dp-section dp-overview">
        <div class="dp-section-head">
          <h2><i class="icon icon-file" /> What gets generated</h2>
        </div>
        <p class="dp-overview-lead">
          When you submit a request, the platform creates these Kubernetes resources and pushes manifests to Git.
          The preview below reflects the <strong>{{ templateLabel }}</strong> template
          <span v-if="form.charts.length"> with {{ form.charts.map(chartName).join(', ') }}</span>.
        </p>

        <div class="dp-overview-grid">

          <div class="dp-overview-panel">
            <div class="dp-overview-panel-head"><i class="icon icon-namespace" /> PlatformRequest CR</div>
            <pre class="dp-yaml"><code>{{ previewPlatformRequestYaml }}</code></pre>
          </div>

          <div class="dp-overview-panel">
            <div class="dp-overview-panel-head"><i class="icon icon-namespace" /> Namespace</div>
            <pre class="dp-yaml"><code>{{ previewNamespaceYaml }}</code></pre>
          </div>

          <div v-if="form.template !== 'sandbox'" class="dp-overview-panel">
            <div class="dp-overview-panel-head"><i class="icon icon-fleet" /> Fleet GitRepo</div>
            <pre class="dp-yaml"><code>{{ previewGitRepoYaml }}</code></pre>
          </div>

          <div class="dp-overview-panel">
            <div class="dp-overview-panel-head"><i class="icon icon-folder" /> Git repository layout</div>
            <pre class="dp-yaml dp-yaml-tree"><code>{{ previewGitTree }}</code></pre>
          </div>

        </div>

        <p class="dp-overview-repo-hint">
          <i class="icon icon-github" />
          Platform Git repo:
          <a :href="platformGitRepo" target="_blank" rel="noopener"><code>{{ platformGitRepo }}</code></a>
          <span v-if="platformGitBranch"> · branch <code>{{ platformGitBranch }}</code></span>
          <span class="muted"> — future: automation opens a PR here when you submit a request</span>
        </p>
      </section>
    </div>
  </div>
</template>

<script>
function devportalBackendUrl() {
  if (typeof window !== 'undefined') {
    const { hostname, port } = window.location;
    if (hostname === 'localhost' && port === '8005') {
      return '/devportal-api';
    }
  }
  return 'http://localhost:9010';
}

const BACKEND_URL = devportalBackendUrl();

let _tokenCache = { token: null, expires: 0 };
async function getRancherToken() {
  if (_tokenCache.token && Date.now() < _tokenCache.expires) return _tokenCache.token;
  const base = window.location.origin;
  const paths = [
    '/k8s/clusters/local/apis/ext.cattle.io/v1/tokens',
    '/v1/tokens.ext.cattle.io',
  ];
  let lastErr;
  for (const apiPath of paths) {
    try {
      const resp = await fetch(`${base}${apiPath}`, {
        method: 'POST',
        credentials: 'include',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          apiVersion: 'ext.cattle.io/v1',
          kind: 'Token',
          metadata: { generateName: 'devportal-' },
          spec: { description: 'Developer Portal', ttl: 3600000 },
        }),
      });
      if (!resp.ok) throw new Error(`Token API ${resp.status}`);
      const data = await resp.json();
      const token = data.status?.bearerToken || data.status?.value || data.token;
      if (token) _tokenCache = { token, expires: Date.now() + 50 * 60 * 1000 };
      return token;
    } catch (e) {
      lastErr = e;
    }
  }
  throw lastErr || new Error('Could not get Rancher token');
}

export default {
  name: 'DevPortalPage',
  layout: 'plain',

  data() {
    return {
      loading: false,
      submitting: false,
      error: '',
      message: '',
      authUser: null,
      isAdmin: false,
      catalog: [],
      templates: [],
      requests: [],
      expandedCrName: null,
      platformGitRepo: '',
      platformGitBranch: '',
      stackInfo: { recommended: '', summary: '', components: [] },
      showWizard: false,
      wizardStep: 1,
      form: {
        name: '',
        slug: '',
        description: '',
        template: 'sandbox',
        charts: [],
      },
    };
  },

  computed: {
    canNext() {
      if (this.wizardStep === 1) return /^[a-z0-9]([a-z0-9-]{1,28}[a-z0-9])?$/.test(this.form.slug);
      if (this.wizardStep === 2) return !!this.form.template;
      return true;
    },
    templateLabel() {
      return this.templates.find((t) => t.id === this.form.template)?.label || this.form.template;
    },

    _previewEnv() {
      return this.form.slug || '<env-name>';
    },

    previewPlatformRequestYaml() {
      const env = this._previewEnv;
      const lines = [
        'apiVersion: platform.devportal.io/v1alpha1',
        'kind: PlatformRequest',
        'metadata:',
        `  name: pr-${env}`,
        `  namespace: devportal-system`,
        'spec:',
        `  name: ${env}`,
        `  displayName: "${this.form.name || env}"`,
        `  template: ${this.form.template || 'sandbox'}`,
      ];
      if (this.form.charts.length) {
        lines.push('  charts:');
        this.form.charts.forEach((c) => lines.push(`    - ${c}`));
      }
      if (this.form.description) {
        lines.push(`  description: "${this.form.description}"`);
      }
      return lines.join('\n');
    },

    previewNamespaceYaml() {
      const env = this._previewEnv;
      return [
        'apiVersion: v1',
        'kind: Namespace',
        'metadata:',
        `  name: env-${env}`,
        '  labels:',
        `    devportal.io/env: "${env}"`,
        `    devportal.io/template: "${this.form.template || 'sandbox'}"`,
      ].join('\n');
    },

    previewGitRepoYaml() {
      const env = this._previewEnv;
      const repo = this.platformGitRepo || 'https://github.com/your-org/platform';
      const branch = this.platformGitBranch || 'main';
      return [
        'apiVersion: fleet.cattle.io/v1alpha1',
        'kind: GitRepo',
        'metadata:',
        `  name: fleet-${env}`,
        '  namespace: fleet-default',
        'spec:',
        `  repo: ${repo}`,
        `  branch: ${branch}`,
        `  paths:`,
        `    - environments/${env}`,
        '  targets:',
        '    - clusterSelector: {}',
      ].join('\n');
    },

    previewGitTree() {
      const env = this._previewEnv;
      const charts = this.form.charts.length ? this.form.charts : ['<chart-name>'];
      const lines = [
        `${this.platformGitRepo || 'platform-repo'}/`,
        `└── environments/`,
        `    └── ${env}/`,
        `        ├── namespace.yaml`,
        `        ├── fleet.yaml`,
      ];
      charts.forEach((c, i) => {
        const isLast = i === charts.length - 1;
        lines.push(`        ${isLast ? '└' : '├'}── charts/`);
        lines.push(`        ${isLast ? ' ' : '│'}   └── ${c}/`);
        lines.push(`        ${isLast ? ' ' : '│'}       ├── Chart.yaml`);
        lines.push(`        ${isLast ? ' ' : '│'}       └── values.yaml`);
      });
      return lines.join('\n');
    },
  },

  async mounted() {
    await Promise.all([this.fetchAuth(), this.loadCatalog(), this.loadStack(), this.loadRequests()]);
  },

  methods: {
    async api(method, path, body) {
      const headers = { 'Content-Type': 'application/json' };
      try {
        const token = await getRancherToken();
        if (token) headers.Authorization = `Bearer ${token}`;
      } catch (_) {}
      let resp;
      try {
        resp = await fetch(`${BACKEND_URL}${path}`, {
          method,
          headers,
          body: body ? JSON.stringify(body) : undefined,
        });
      } catch (e) {
        throw new Error(`Backend unreachable — ${e.message}. Is devportal-backend running on :9010?`);
      }
      let data;
      try {
        data = await resp.json();
      } catch (_) {
        throw new Error(`Invalid response from backend (${resp.status})`);
      }
      if (!resp.ok) throw new Error(data.error || `HTTP ${resp.status}`);
      return data;
    },

    async fetchAuth() {
      try {
        const data = await this.api('GET', '/api/auth/me');
        this.authUser = data.user;
        this.isAdmin = !!(data.capabilities && (data.capabilities.admin || data.capabilities.listAllRequests));
      } catch (_) {}
    },

    async loadCatalog() {
      try {
        const data = await this.api('GET', '/api/portal/catalog');
        this.catalog = data.charts || [];
        this.templates = data.templates || [];
      } catch (e) {
        this.error = e.message;
      }
    },

    async loadStack() {
      try {
        const data = await this.api('GET', '/api/portal/stack');
        this.stackInfo = data;
      } catch (_) {}
    },

    async loadRequests() {
      this.loading = true;
      this.error = '';
      try {
        const data = await this.api('GET', '/api/portal/requests');
        this.requests = data.requests || [];
        if (data.listAll) this.isAdmin = true;
        this.platformGitRepo = data.gitRepo || '';
        this.platformGitBranch = data.gitBranch || '';
        if (data.warning) {
          this.error = data.warning;
        }
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },

    startWizard() {
      this.showWizard = true;
      this.wizardStep = 1;
      this.form = { name: '', slug: '', description: '', template: 'sandbox', charts: [] };
    },

    cancelWizard() {
      this.showWizard = false;
    },

    slugifyName() {
      this.form.slug = this.form.name
        .toLowerCase()
        .replace(/[^a-z0-9-]+/g, '-')
        .replace(/^-+|-+$/g, '')
        .slice(0, 30);
    },

    chartName(id) {
      return this.catalog.find((c) => c.id === id)?.name || id;
    },

    templateIcon(id) {
      const icons = {
        sandbox: 'icon-namespace',
        team: 'icon-fleet',
        vcluster: 'icon-cluster',
      };
      return icons[id] || 'icon-question';
    },

    formatDate(iso) {
      if (!iso) return '—';
      try {
        return new Date(iso).toLocaleString();
      } catch (_) {
        return iso;
      }
    },

    toggleRequestDetail(r) {
      const key = r.crName || r.name;
      this.expandedCrName = this.expandedCrName === key ? null : key;
    },

    openRequestDetail(r) {
      this.expandedCrName = r.crName || r.name;
    },

    async submitRequest() {
      this.submitting = true;
      this.error = '';
      try {
        await this.api('POST', '/api/portal/requests', {
          name: this.form.slug,
          displayName: this.form.name,
          description: this.form.description,
          template: this.form.template,
          charts: this.form.charts,
        });
        this.message = `Environment "${this.form.name}" submitted — provisioning started.`;
        this.showWizard = false;
        await this.loadRequests();
      } catch (e) {
        this.error = e.message;
      } finally {
        this.submitting = false;
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.devportal-page {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  /* Full viewport width — escape IndentedPanel 90% constraint in plain layout */
  width: 111.12%;
  margin-left: -5.56%;
  min-height: calc(100vh - 60px);
  height: calc(100vh - 60px);
  max-width: none;
  padding: 8px 12px;
  color: var(--body-text);
  background: var(--body-bg);
}

.banner {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 6px 10px;
  border-radius: 4px;
  margin-bottom: 8px;
  font-size: 0.8em;
  &.error { background: var(--error-banner-bg, rgba(204, 74, 74, 0.15)); color: var(--error, #c00); }
  &.success { background: var(--success-banner-bg, rgba(63, 138, 63, 0.15)); color: var(--success, #3f8a3f); }
  .dismiss { background: none; border: none; cursor: pointer; font-size: 1.1em; padding: 0 4px; opacity: 0.7; }
}

.loading-bar {
  height: 2px;
  background: var(--primary);
  margin-bottom: 6px;
}

.dp-panel {
  flex: 1;
  min-height: 0;
  width: 100%;
  max-width: none;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--body-bg);
  overflow: auto;
}

.dp-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  background: var(--sortable-table-header-bg, var(--box-bg));

  .dp-header-left {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    min-width: 0;
  }

  .dp-header-icon {
    font-size: 1.75em;
    color: var(--primary);
    margin-top: 2px;
    flex-shrink: 0;
  }

  h1 {
    margin: 0 0 4px;
    font-size: 1.05em;
    font-weight: 600;
    color: var(--body-text);
  }

  p {
    margin: 0;
    font-size: 0.82em;
    color: var(--muted);
    max-width: none;
    line-height: 1.4;
  }

  .dp-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 8px;
  }

  .dp-badge {
    padding: 2px 8px;
    border-radius: 3px;
    font-size: 0.72em;
    font-weight: 600;
    &.user {
      background: var(--primary-banner-bg, rgba(0, 100, 200, 0.12));
      color: var(--primary);
    }
    &.admin {
      background: rgba(156, 39, 176, 0.12);
      color: #9c27b0;
    }
    &.muted {
      background: var(--default-light-bg, rgba(0, 0, 0, 0.05));
      color: var(--muted);
      font-weight: 500;
    }
  }
}

.dp-wizard {
  margin: 12px 16px;
  padding: 16px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--sortable-table-row-bg, var(--body-bg));

  .dp-wizard-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    h2 { margin: 0; font-size: 0.95em; font-weight: 600; }
  }

  .dp-steps {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin: 12px 0 18px;
    font-size: 0.78em;
  }

  .dp-step-pill {
    padding: 4px 10px;
    border-radius: 999px;
    border: 1px solid var(--border);
    color: var(--muted);
    background: var(--body-bg);

    &.active {
      color: var(--primary);
      border-color: var(--primary);
      font-weight: 600;
      background: var(--sortable-table-selected-bg, rgba(0, 100, 200, 0.06));
    }

    &.done {
      color: var(--success, #3f8a3f);
      border-color: rgba(63, 138, 63, 0.35);
    }
  }

  .label {
    display: block;
    margin: 10px 0 4px;
    font-size: 0.78em;
    color: var(--input-label, var(--muted));
  }

  .input-sm {
    width: 100%;
    max-width: 360px;
    padding: 6px 8px;
    font-size: 0.85em;
    border: 1px solid var(--input-border, var(--border));
    border-radius: 4px;
    background: var(--input-bg);
    color: var(--input-text);
  }

  .hint {
    font-size: 0.78em;
    color: var(--muted);
    margin: 6px 0;
    code { font-size: 0.95em; }
  }

  .dp-wizard-actions {
    display: flex;
    gap: 8px;
    margin-top: 16px;
    padding-top: 12px;
    border-top: 1px solid var(--border);
  }

  .dp-step-select {
    .step-lead {
      margin: 0 0 14px;
      font-size: 0.85em;
      color: var(--muted);
      line-height: 1.45;
    }
  }
}

.dp-select-grid {
  display: grid;
  gap: 12px;
  width: 100%;

  &--charts {
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  }
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

.dp-template-list,
.dp-chart-list {
  border: 0;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 640px;
}

.dp-template-option,
.dp-chart-option {
  display: flex;
  flex-direction: row;
  align-items: flex-start;
  gap: 12px;
  margin: 0;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--body-bg);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;

  &:hover {
    border-color: var(--primary);
    background: var(--sortable-table-hover-bg, var(--body-bg));
  }

  &.selected {
    border-color: var(--primary);
    background: var(--sortable-table-selected-bg, rgba(0, 100, 200, 0.06));
    box-shadow: inset 3px 0 0 var(--primary);
  }
}

.dp-template-radio,
.dp-chart-checkbox {
  flex-shrink: 0;
  margin: 4px 0 0;
  width: 16px;
  height: 16px;
  accent-color: var(--primary);
}

.dp-template-icon {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  margin-top: 1px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--default-light-bg, rgba(0, 0, 0, 0.05));
  color: var(--primary);

  .icon { font-size: 1.1em; }
}

.dp-template-body,
.dp-chart-body {
  display: block;
  flex: 1;
  min-width: 0;
}

.dp-template-title,
.dp-chart-title {
  display: block;
  font-size: 0.92em;
  font-weight: 600;
  line-height: 1.35;
  color: var(--body-text);
  margin-bottom: 4px;
}

.dp-template-desc,
.dp-chart-desc {
  display: block;
  font-size: 0.82em;
  line-height: 1.45;
  color: var(--muted);
}

.dp-template-detail {
  display: block;
  margin-top: 6px;
  font-size: 0.75em;
  line-height: 1.4;
  color: var(--muted);
  font-style: italic;
}

.dp-chart-tag {
  display: inline-block;
  margin: 0 0 4px;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 0.65em;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--primary);
  background: var(--primary-banner-bg, rgba(0, 100, 200, 0.1));
}

.dp-card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 10px;
  margin-top: 8px;
}

.dp-card {
  text-align: left;
  padding: 12px;
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--body-bg);
  cursor: pointer;
  color: var(--body-text);
  transition: border-color 0.15s, background 0.15s;

  strong { display: block; font-size: 0.88em; margin-bottom: 4px; }
  span { font-size: 0.78em; color: var(--muted); display: block; line-height: 1.35; }

  &:hover { border-color: var(--primary); background: var(--sortable-table-hover-bg, var(--body-bg)); }
  &.selected {
    border-color: var(--primary);
    background: var(--sortable-table-selected-bg, rgba(0, 100, 200, 0.06));
  }

  &.dp-card-check input { margin-right: 6px; vertical-align: middle; }
  .cat { text-transform: uppercase; font-size: 0.68em; letter-spacing: 0.03em; margin-top: 4px; color: var(--primary); }
}

.dp-review dl {
  display: grid;
  grid-template-columns: 100px 1fr;
  gap: 6px 12px;
  font-size: 0.85em;
  dt { color: var(--muted); }
  .dp-chip {
    display: inline-block;
    margin: 2px 4px 2px 0;
    padding: 2px 8px;
    border-radius: 3px;
    background: var(--tag-bg, var(--default-light-bg));
    font-size: 0.9em;
  }
  .muted { color: var(--muted); }
}

.dp-section {
  padding: 12px 16px 16px;
  border-top: 1px solid var(--border);

  .dp-section-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;
  }

  h2 {
    margin: 0;
    font-size: 0.9em;
    font-weight: 600;
    color: var(--body-text);
    .icon { margin-right: 6px; color: var(--muted); font-size: 0.95em; }
  }

  .empty { font-size: 0.82em; color: var(--muted); margin: 0; }
  .admin-hint { margin: 0 0 10px; }
}

.dp-git-hint {
  font-size: 0.78em;
  color: var(--muted);
  margin: 0 0 12px;
  .icon { margin-right: 4px; color: var(--primary); }
  code { font-size: 0.95em; }
}

.dp-table-hint {
  font-size: 0.78em;
  color: var(--muted);
  margin: 0 0 10px;
  .icon { margin-right: 4px; color: var(--primary); }
}

.dp-table-requests {
  .col-expand {
    width: 28px;
    text-align: center;
    color: var(--muted);
  }

  .col-actions {
    width: 130px;
    white-space: nowrap;
  }

  .dp-request-row {
    cursor: pointer;
    &.expanded { background: var(--sortable-table-hover-bg); }
    .cr-name {
      display: block;
      margin-top: 4px;
      font-size: 0.75em;
      color: var(--muted);
    }
  }
}

.dp-request-detail-row td {
  padding: 0 !important;
  border-bottom: 1px solid var(--border);
  background: var(--sortable-table-row-bg, var(--body-bg));
}

.dp-request-detail {
  padding: 12px 14px 16px;
}

.dp-detail-banner {
  font-size: 0.82em;
  margin-bottom: 10px;
  padding: 8px 10px;
  border-radius: 4px;
  background: var(--default-light-bg, rgba(0, 0, 0, 0.04));
  border: 1px solid var(--border);
  &.muted { color: var(--muted); }
  .icon { margin-right: 4px; }
}

.dp-detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;

  @media (max-width: 900px) {
    grid-template-columns: 1fr;
  }
}

.dp-detail-panel {
  border: 1px solid var(--border);
  border-radius: 4px;
  background: var(--body-bg);
  overflow: hidden;

  h3 {
    margin: 0;
    padding: 8px 10px;
    font-size: 0.82em;
    font-weight: 600;
    border-bottom: 1px solid var(--border);
    background: var(--sortable-table-header-bg, var(--box-bg));
    .icon { margin-right: 6px; color: var(--primary); }
  }
}

.dp-fleet-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.78em;

  th, td {
    padding: 6px 8px;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }

  th {
    font-size: 0.72em;
    text-transform: uppercase;
    color: var(--muted);
    background: var(--sortable-table-header-bg, var(--box-bg));
  }

  .fleet-phase {
    text-transform: capitalize;
    &.ready, &.active { color: var(--success, #3f8a3f); }
    &.planned, &.pending { color: var(--muted); }
  }
}

.dp-git-meta {
  margin: 8px 10px 10px;
  font-size: 0.75em;
  color: var(--muted);
  code { font-size: 0.95em; }
}

.dp-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.82em;

  th, td {
    padding: 8px 10px;
    text-align: left;
    border-bottom: 1px solid var(--border);
  }

  th {
    color: var(--sortable-table-group-label, var(--muted));
    background: var(--sortable-table-header-bg, var(--box-bg));
    font-weight: 600;
    font-size: 0.78em;
    text-transform: uppercase;
    letter-spacing: 0.02em;
  }

  tbody tr:hover { background: var(--sortable-table-hover-bg); }
  .name {
    font-weight: 600;
    strong { display: block; }
    .desc-line { display: block; font-weight: 400; font-size: 0.85em; color: var(--muted); margin-top: 2px; }
  }
  .status-msg { max-width: 360px; color: var(--muted); font-size: 0.9em; }
  code { font-size: 0.92em; color: var(--muted); }
  .cr-name { font-size: 0.8em; }

  .phase {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 3px;
    font-size: 0.85em;
    font-weight: 600;
    &.Ready { background: rgba(63, 138, 63, 0.15); color: var(--success, #3f8a3f); }
    &.Provisioning { background: rgba(0, 100, 200, 0.12); color: var(--primary); }
    &.Failed { background: rgba(204, 74, 74, 0.15); color: var(--error, #c00); }
    &.Pending { background: var(--default-light-bg); color: var(--muted); }
  }
}

.dp-overview {
  border-top: 1px solid var(--border);

  .dp-overview-lead {
    font-size: 0.82em;
    color: var(--muted);
    margin: 0 0 14px;
    line-height: 1.5;

    strong { color: var(--primary); font-weight: 600; }
  }
}

.dp-overview-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}

.dp-overview-panel {
  border: 1px solid var(--border);
  border-radius: 4px;
  overflow: hidden;
  background: var(--body-bg);
}

.dp-overview-panel-head {
  padding: 7px 10px;
  font-size: 0.78em;
  font-weight: 600;
  border-bottom: 1px solid var(--border);
  background: var(--sortable-table-header-bg, var(--box-bg));
  color: var(--body-text);

  .icon { margin-right: 6px; color: var(--primary); }
}

.dp-yaml {
  margin: 0;
  padding: 10px 12px;
  max-height: 240px;
  overflow: auto;
  font-size: 0.72em;
  line-height: 1.5;
  background: #0d1117;
  color: #e6edf3;

  code {
    background: transparent;
    padding: 0;
    color: inherit;
    white-space: pre;
    word-break: normal;
    font-family: 'JetBrains Mono', 'Fira Code', ui-monospace, monospace;
  }

  &.dp-yaml-tree code {
    color: #a5d6ff;
  }
}

.dp-overview-repo-hint {
  font-size: 0.78em;
  color: var(--muted);
  margin: 8px 0 0;

  .icon { margin-right: 4px; color: var(--muted); }
  code { font-size: 0.95em; color: var(--body-text); }
  a { color: var(--primary); text-decoration: none; &:hover { text-decoration: underline; } }
  .muted { color: var(--muted); }
}
</style>

<style lang="scss">
/* Unscoped — keep template option text on separate lines */
.devportal-page {
  .dp-template-body,
  .dp-chart-body {
    display: block !important;
  }

  .dp-template-title,
  .dp-template-desc,
  .dp-template-detail,
  .dp-chart-title,
  .dp-chart-desc,
  .dp-chart-tag {
    display: block !important;
    white-space: normal !important;
    overflow: visible !important;
    text-overflow: unset !important;
  }

  .dp-chart-tag {
    display: inline-block !important;
  }
}
</style>
