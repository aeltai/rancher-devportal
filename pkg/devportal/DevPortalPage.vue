<template>
  <div class="devportal" :class="themeClass">
    <header class="dp-hero">
      <div class="dp-hero-text">
        <h1>Developer Portal</h1>
        <p>Self-service environments — virtual cluster, Fleet GitOps, and operators you pick.</p>
        <div v-if="authUser" class="dp-hero-meta">
          Signed in as <strong>{{ authUser.displayName || authUser.username }}</strong>
          <span v-if="stackInfo.recommended" class="dp-stack-badge">{{ stackInfo.recommended }}</span>
        </div>
      </div>
      <button class="btn role-primary" @click="startWizard">+ Request environment</button>
    </header>

    <div v-if="error" class="banner error">{{ error }} <button class="dismiss" @click="error = ''">&times;</button></div>
    <div v-if="message" class="banner success">{{ message }} <button class="dismiss" @click="message = ''">&times;</button></div>

    <div v-if="showWizard" class="dp-wizard">
      <div class="dp-wizard-header">
        <h2>New environment</h2>
        <button class="btn role-tertiary xs" @click="cancelWizard">&times;</button>
      </div>
      <div class="dp-steps">
        <span :class="{ active: wizardStep >= 1 }">1. Name</span>
        <span :class="{ active: wizardStep >= 2 }">2. Template</span>
        <span :class="{ active: wizardStep >= 3 }">3. Charts</span>
        <span :class="{ active: wizardStep >= 4 }">4. Review</span>
      </div>

      <div v-if="wizardStep === 1" class="dp-step">
        <label>Environment name</label>
        <input v-model="form.name" type="text" placeholder="my-team-dev" @input="slugifyName" />
        <p class="hint">Lowercase, numbers, hyphens. Creates namespace <code>env-{{ form.slug || '…' }}</code></p>
        <label>Description</label>
        <input v-model="form.description" type="text" placeholder="Optional" />
      </div>

      <div v-if="wizardStep === 2" class="dp-step">
        <p class="hint">Cluster template — guardrails for the provisioned environment.</p>
        <div class="dp-template-grid">
          <button
            v-for="t in templates"
            :key="t.id"
            :class="['dp-template-card', { selected: form.template === t.id }]"
            @click="form.template = t.id"
          >
            <strong>{{ t.label }}</strong>
            <span>{{ t.description }}</span>
          </button>
        </div>
      </div>

      <div v-if="wizardStep === 3" class="dp-step">
        <p class="hint">Select Helm charts / operators to install via Fleet on your environment.</p>
        <div class="dp-catalog-grid">
          <label
            v-for="c in catalog"
            :key="c.id"
            :class="['dp-catalog-card', { selected: form.charts.includes(c.id) }]"
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
            <span v-for="id in form.charts" :key="id" class="chip">{{ chartName(id) }}</span>
          </dd>
        </dl>
      </div>

      <div class="dp-wizard-actions">
        <button v-if="wizardStep > 1" class="btn role-secondary" @click="wizardStep--">Back</button>
        <button v-if="wizardStep < 4" class="btn role-primary" :disabled="!canNext" @click="wizardStep++">Next</button>
        <button v-if="wizardStep === 4" class="btn role-primary" :disabled="submitting" @click="submitRequest">
          {{ submitting ? 'Submitting…' : 'Submit request' }}
        </button>
      </div>
    </div>

    <section class="dp-section">
      <h2>My environments</h2>
      <button class="btn role-tertiary xs refresh-btn" :disabled="loading" @click="loadRequests">Refresh</button>
      <table v-if="requests.length" class="dp-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Phase</th>
            <th>Namespace</th>
            <th>Fleet repo</th>
            <th>Charts</th>
            <th>Created</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in requests" :key="r.name">
            <td class="name">{{ r.displayName || r.name }}</td>
            <td><span :class="['phase', r.phase]">{{ r.phase }}</span></td>
            <td><code>{{ r.namespace || '—' }}</code></td>
            <td><code>{{ r.fleetGitRepo || '—' }}</code></td>
            <td>{{ (r.charts || []).join(', ') || '—' }}</td>
            <td>{{ r.created || '—' }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else-if="!loading" class="empty">No environments yet — click <strong>Request environment</strong> to start.</p>
    </section>

    <section class="dp-section dp-stack">
      <h2>Recommended stack</h2>
      <div class="dp-stack-grid">
        <div v-for="item in stackInfo.components" :key="item.name" class="dp-stack-card">
          <strong>{{ item.name }}</strong>
          <p>{{ item.role }}</p>
        </div>
      </div>
      <p class="hint">{{ stackInfo.summary }}</p>
    </section>
  </div>
</template>

<script>
const BACKEND_URL = 'http://localhost:9010';

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
    themeClass() {
      return 'theme-dark';
    },
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
      const resp = await fetch(`${BACKEND_URL}${path}`, {
        method,
        headers,
        body: body ? JSON.stringify(body) : undefined,
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error || `HTTP ${resp.status}`);
      return data;
    },

    async fetchAuth() {
      try {
        const data = await this.api('GET', '/api/auth/me');
        this.authUser = data.user;
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
.devportal {
  padding: 16px 20px;
  max-width: 1200px;
  margin: 0 auto;
  color: #e0e0e0;
  background: #0d0d0d;
  min-height: calc(100vh - 60px);
}

.dp-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 24px;
  padding: 20px 24px;
  border-radius: 10px;
  background: linear-gradient(135deg, #1a237e 0%, #283593 40%, #1b5e20 100%);
  h1 { margin: 0 0 8px; font-size: 1.6em; color: #fff; }
  p { margin: 0; opacity: 0.9; max-width: 520px; }
  .dp-hero-meta { margin-top: 12px; font-size: 0.85em; opacity: 0.85; }
  .dp-stack-badge {
    margin-left: 8px;
    padding: 2px 8px;
    border-radius: 4px;
    background: rgba(255,255,255,0.15);
    font-size: 0.85em;
  }
}

.banner {
  padding: 8px 12px;
  border-radius: 6px;
  margin-bottom: 12px;
  font-size: 0.9em;
  &.error { background: #fdecea; color: #b71c1c; }
  &.success { background: #e8f5e9; color: #1b5e20; }
  .dismiss { background: none; border: none; cursor: pointer; float: right; }
}

.dp-wizard {
  background: #1a1a1a;
  border: 1px solid #333;
  border-radius: 10px;
  padding: 20px;
  margin-bottom: 24px;
  .dp-wizard-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    h2 { margin: 0; font-size: 1.1em; color: #90caf9; }
  }
  .dp-steps {
    display: flex;
    gap: 16px;
    margin: 16px 0;
    font-size: 0.8em;
    color: #666;
    span.active { color: #4caf50; font-weight: 600; }
  }
  label { display: block; margin: 12px 0 4px; font-size: 0.85em; color: #aaa; }
  input[type="text"] {
    width: 100%;
    max-width: 400px;
    padding: 8px 10px;
    border: 1px solid #444;
    border-radius: 6px;
    background: #252525;
    color: #eee;
  }
  .hint { font-size: 0.8em; color: #888; margin: 8px 0; code { color: #81c784; } }
  .dp-wizard-actions {
    display: flex;
    gap: 8px;
    margin-top: 20px;
    padding-top: 16px;
    border-top: 1px solid #333;
  }
}

.dp-template-grid, .dp-catalog-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
  margin-top: 12px;
}

.dp-template-card, .dp-catalog-card {
  text-align: left;
  padding: 14px;
  border: 2px solid #333;
  border-radius: 8px;
  background: #252525;
  cursor: pointer;
  color: #ccc;
  strong { display: block; color: #90caf9; margin-bottom: 4px; }
  span { font-size: 0.8em; display: block; }
  &.selected { border-color: #4caf50; background: rgba(76,175,80,0.08); }
  input { margin-right: 8px; }
  .cat { color: #ffb74d; font-size: 0.75em; text-transform: uppercase; }
  .desc { color: #888; margin-top: 4px; }
}

.dp-review dl {
  display: grid;
  grid-template-columns: 120px 1fr;
  gap: 8px 16px;
  dt { color: #888; }
  .chip {
    display: inline-block;
    margin: 2px 4px 2px 0;
    padding: 2px 8px;
    border-radius: 4px;
    background: #333;
    font-size: 0.85em;
  }
}

.dp-section {
  margin-bottom: 28px;
  position: relative;
  h2 { font-size: 1em; color: #4caf50; margin-bottom: 12px; }
  .refresh-btn { position: absolute; top: 0; right: 0; }
  .empty { color: #888; font-size: 0.9em; }
}

.dp-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85em;
  th, td { padding: 8px 10px; text-align: left; border-bottom: 1px solid #333; }
  th { color: #4caf50; background: #252525; }
  .name { font-weight: 600; color: #64b5f6; }
  code { font-size: 0.9em; color: #81c784; }
  .phase {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.85em;
    font-weight: 600;
    &.Ready { background: #2e7d32; color: #a5d6a7; }
    &.Provisioning { background: #1565c0; color: #90caf9; }
    &.Failed { background: #c62828; color: #ffcdd2; }
    &.Pending { background: #555; color: #ccc; }
  }
}

.dp-stack-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
  .dp-stack-card {
    padding: 12px;
    border-radius: 8px;
    border: 1px solid #333;
    background: #1a1a1a;
    strong { color: #90caf9; }
    p { margin: 6px 0 0; font-size: 0.8em; color: #888; }
  }
}
</style>
