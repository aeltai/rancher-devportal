function devportalBackendUrl() {
  if (typeof window !== 'undefined') {
    const { hostname, port } = window.location;
    const isLocalHost = hostname === 'localhost' || hostname === '127.0.0.1';
    if (isLocalHost && (port === '8005' || port === '8006')) {
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
          spec: { description: 'Geeko-Ops', ttl: 3600000 },
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
      collections: [],
      offerings: [],
      requests: [],
      expandedCrName: null,
      selectedGitFiles: {},
      detailTab: 'git',
      platformGitRepo: '',
      platformGitBranch: '',
      showWizard: false,
      configYaml: '',
      platformConfig: null,
      configSaving: false,
      gitRepos: [],
      wizardCollectionId: null,
    };
  },

  computed: {
    pendingRequests() {
      return this.requests.filter((r) => this.needsAdminApproval(r));
    },
    readyCount() {
      return this.requests.filter((r) => r.phase === 'Ready').length;
    },
    failedCount() {
      return this.requests.filter((r) => r.phase === 'Failed').length;
    },
  },

  async mounted() {
    await Promise.all([this.fetchAuth(), this.loadCatalog(), this.loadRequests()]);
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
      const raw = await resp.text();
      try {
        data = raw ? JSON.parse(raw) : {};
      } catch (_) {
        const hint = raw && raw.length < 120 ? raw.trim() : '';
        throw new Error(
          hint
            ? `Backend error (${resp.status}): ${hint}`
            : `Backend unreachable or wrong URL (${resp.status}). Restart devportal-backend on :9010 and use Shell dev UI (:8005 or :8006).`
        );
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
        this.collections = data.collections || [];
        this.offerings = data.offerings || [];
        this.gitRepos = (data.git && data.git.repos) || [];
        if (data.git && data.git.defaultRepo) {
          this.platformGitRepo = data.git.defaultRepo;
        } else if (data.defaults && data.defaults.gitRepo) {
          this.platformGitRepo = data.defaults.gitRepo;
        }
      } catch (e) {
        this.error = e.message;
      }
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
        if (data.warning) this.error = data.warning;
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },

    startWizard(collectionId) {
      this.showWizard = true;
      this.wizardCollectionId = collectionId || null;
    },

    cancelWizard() {
      this.showWizard = false;
      this.wizardCollectionId = null;
    },

    requestNeedsGitOps(template, charts, customResources) {
      if ((customResources && customResources.length) || (charts && charts.length)) return true;
      const t = this.templates.find((x) => x.id === template);
      if (t && t.gitOps !== undefined) return t.gitOps;
      return template !== 'sandbox';
    },

    needsAdminApproval(r) {
      if (!r) return false;
      if (r.phase === 'PendingApproval') return true;
      if (r.phase !== 'Pending') return false;
      return this.requestNeedsGitOps(r.template, r.charts, r.customResources);
    },

    async loadPlatformConfig() {
      try {
        const data = await this.api('GET', '/api/portal/platform-config');
        this.configYaml = data.yaml || '';
        this.platformConfig = {
          defaults: data.defaults,
          git: data.git,
          collections: data.collections || [],
          offerings: data.offerings || [],
          templates: data.templates || [],
          charts: data.charts || [],
          crdDiscovery: data.crdDiscovery,
          approval: data.approval,
        };
        if (data.git && data.git.repos) this.gitRepos = data.git.repos;
      } catch (e) {
        this.error = e.message;
      }
    },

    async savePlatformConfig(payload) {
      this.configSaving = true;
      this.error = '';
      try {
        await this.api('PUT', '/api/portal/platform-config', payload);
        this.message = 'Geeko-Ops catalog saved.';
        await this.loadCatalog();
        await this.loadPlatformConfig();
      } catch (e) {
        this.error = e.message;
      } finally {
        this.configSaving = false;
      }
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
      if (this.expandedCrName === key) {
        this.expandedCrName = null;
        return;
      }
      this.reviewRequest(r);
    },

    reviewRequest(r) {
      const key = r.crName || r.name;
      this.expandedCrName = key;
      this.detailTab = this.isAdmin && this.needsAdminApproval(r) ? 'git' : 'resources';
      if (r.gitPreview?.files?.length) {
        this.selectedGitFiles = { ...this.selectedGitFiles, [key]: r.gitPreview.files[0].path };
      }
    },

    phaseLabel(phase) {
      const labels = {
        PendingApproval: 'Awaiting approval',
        Approved: 'Approved',
        Rejected: 'Rejected',
        Reconciling: 'Provisioning',
        Pushing: 'Deploying',
        Ready: 'Ready',
        Failed: 'Failed',
        Pending: 'Pending',
      };
      return labels[phase] || phase || '—';
    },

    phaseLabelFor(r) {
      if (this.needsAdminApproval(r)) return 'Awaiting approval';
      return this.phaseLabel(r.phase);
    },

    selectGitFile(r, path) {
      this.selectedGitFiles = { ...this.selectedGitFiles, [r.crName || r.name]: path };
    },

    selectedGitFile(r) {
      const key = r.crName || r.name;
      if (this.selectedGitFiles[key]) return this.selectedGitFiles[key];
      return r.gitPreview?.files?.[0]?.path || '';
    },

    selectedGitFileContent(r) {
      const path = this.selectedGitFile(r);
      const file = (r.gitPreview?.files || []).find((f) => f.path === path);
      return file?.content || '—';
    },

    fileBaseName(path) {
      return path.split('/').pop() || path;
    },

    collectionIcon(icon) {
      if (!icon) return 'icon-folder';
      return icon.startsWith('icon-') ? icon : `icon-${icon}`;
    },

    async approveRequest(r) {
      const crName = r.crName || r.name;
      this.loading = true;
      this.error = '';
      try {
        await this.api('POST', `/api/portal/requests/${encodeURIComponent(crName)}/approve`);
        this.message = `Approved "${r.displayName || r.name}".`;
        await this.loadRequests();
        this.expandedCrName = crName;
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },

    async rejectRequest(r) {
      const crName = r.crName || r.name;
      const reason = window.prompt('Rejection reason (optional):') || '';
      this.loading = true;
      this.error = '';
      try {
        const q = reason ? `?reason=${encodeURIComponent(reason)}` : '';
        await this.api('POST', `/api/portal/requests/${encodeURIComponent(crName)}/reject${q}`);
        this.message = `Rejected "${r.displayName || r.name}".`;
        await this.loadRequests();
      } catch (e) {
        this.error = e.message;
      } finally {
        this.loading = false;
      }
    },

    async submitRequest(payload) {
      this.submitting = true;
      this.error = '';
      try {
        await this.api('POST', '/api/portal/requests', payload);
        this.message = `Environment "${payload.displayName || payload.name}" submitted.`;
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
