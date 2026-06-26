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
        <div class="dp-steps">
          <span :class="{ active: wizardStep >= 1 }">1. Name</span>
          <span :class="{ active: wizardStep >= 2 }">2. Template</span>
          <span :class="{ active: wizardStep >= 3 }">3. Charts</span>
          <span :class="{ active: wizardStep >= 4 }">4. Review</span>
        </div>

        <div v-if="wizardStep === 1" class="dp-step">
          <label class="label">Environment name</label>
          <input v-model="form.name" class="input-sm" type="text" placeholder="my-team-dev" @input="slugifyName" />
          <p class="hint">Lowercase, numbers, hyphens. Namespace: <code>env-{{ form.slug || '…' }}</code></p>
          <label class="label">Description</label>
          <input v-model="form.description" class="input-sm" type="text" placeholder="Optional" />
        </div>

        <div v-if="wizardStep === 2" class="dp-step">
          <p class="hint">Cluster template — guardrails for the provisioned environment.</p>
          <div class="dp-card-grid">
            <button
              v-for="t in templates"
              :key="t.id"
              type="button"
              :class="['dp-card', { selected: form.template === t.id }]"
              @click="form.template = t.id"
            >
              <strong>{{ t.label }}</strong>
              <span>{{ t.description }}</span>
            </button>
          </div>
        </div>

        <div v-if="wizardStep === 3" class="dp-step">
          <p class="hint">Helm charts and operators to install via Fleet.</p>
          <div class="dp-card-grid">
            <label
              v-for="c in catalog"
              :key="c.id"
              :class="['dp-card', 'dp-card-check', { selected: form.charts.includes(c.id) }]"
            >
              <input v-model="form.charts" type="checkbox" :value="c.id" />
              <strong>{{ c.name }}</strong>
              <span class="cat">{{ c.category }}</span>
              <span class="desc">{{ c.description }}</span>
            </label>
          </div>
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
        <table v-if="requests.length" class="dp-table">
          <thead>
            <tr>
              <th>Name</th>
              <th v-if="isAdmin">Requester</th>
              <th>Phase</th>
              <th v-if="isAdmin">Message</th>
              <th>Namespace</th>
              <th v-if="isAdmin">Template</th>
              <th v-if="isAdmin">CR name</th>
              <th>Charts</th>
              <th>Created</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in requests" :key="r.crName || r.name">
              <td class="name">
                <strong>{{ r.displayName || r.name }}</strong>
                <div v-if="r.description" class="desc">{{ r.description }}</div>
              </td>
              <td v-if="isAdmin"><code>{{ r.requester || '—' }}</code></td>
              <td><span :class="['phase', r.phase]">{{ r.phase || '—' }}</span></td>
              <td v-if="isAdmin" class="status-msg">{{ r.message || '—' }}</td>
              <td><code>{{ r.namespace || '—' }}</code></td>
              <td v-if="isAdmin"><code>{{ r.template || '—' }}</code></td>
              <td v-if="isAdmin"><code>{{ r.crName || r.name || '—' }}</code></td>
              <td>{{ (r.charts || []).join(', ') || '—' }}</td>
              <td>{{ formatDate(r.createdAt || r.created) }}</td>
            </tr>
          </tbody>
        </table>
        <p v-else-if="!loading" class="empty">
          {{ isAdmin ? 'No platform requests yet.' : 'No environments yet. Use ' }}
          <strong v-if="!isAdmin">Request environment</strong>
          {{ isAdmin ? '' : ' to provision one.' }}
        </p>
      </section>

      <section v-if="stackInfo.components && stackInfo.components.length" class="dp-section dp-stack">
        <h2><i class="icon icon-fleet" /> Recommended stack</h2>
        <p v-if="stackInfo.summary" class="hint">{{ stackInfo.summary }}</p>
        <div class="dp-stack-grid">
          <div v-for="item in stackInfo.components" :key="item.name" class="dp-stack-card">
            <strong>{{ item.name }}</strong>
            <p>{{ item.role }}</p>
          </div>
        </div>
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

    formatDate(iso) {
      if (!iso) return '—';
      try {
        return new Date(iso).toLocaleString();
      } catch (_) {
        return iso;
      }
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
    gap: 12px;
    margin: 12px 0 16px;
    font-size: 0.75em;
    color: var(--muted);
    span.active { color: var(--primary); font-weight: 600; }
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

.dp-stack-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: 8px;
  margin-top: 10px;

  .dp-stack-card {
    padding: 10px 12px;
    border: 1px solid var(--border);
    border-radius: 4px;
    background: var(--body-bg);
    strong { font-size: 0.85em; display: block; }
    p { margin: 4px 0 0; font-size: 0.78em; color: var(--muted); line-height: 1.35; }
  }
}
</style>
