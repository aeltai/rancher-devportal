<template>
  <div class="dp-admin-view">
    <header class="dp-admin-view-head">
      <div class="dp-admin-view-title">
        <span class="dp-admin-badge">Geeko-Ops Admin</span>
        <h1>Geeko-Ops</h1>
        <p class="dp-admin-tagline">The ops marketplace your clusters deserve — catalog, approvals, Fleet delivery.</p>
      </div>
    </header>

    <nav class="dp-admin-main-tabs" role="tablist">
      <button
        v-for="tab in mainTabs"
        :key="tab.id"
        :class="['dp-admin-main-tab', { active: activeTab === tab.id }]"
        type="button"
        role="tab"
        :aria-selected="activeTab === tab.id"
        @click="selectTab(tab.id)"
      >
        <i v-if="tab.icon" :class="['icon', tab.icon]" />
        {{ tab.label }}
      </button>
    </nav>

    <div class="dp-admin-view-body">
      <!-- Request env — mirrors the user self-service flow -->
      <div v-if="activeTab === 'request-env'" class="dp-admin-tab-panel">
        <header class="dp-admin-request-hero">
          <div>
            <span class="dp-admin-request-eyebrow">Marketplace</span>
            <h2>Request an environment</h2>
            <p>Pick from the Geeko-Ops catalog — same self-service flow your users get.</p>
          </div>
          <button
            v-if="!showWizard"
            class="btn role-primary"
            type="button"
            @click="$emit('start-wizard')"
          >
            <i class="icon icon-circle-plus" /> New request
          </button>
        </header>

        <RequestWizard
          v-if="showWizard"
          :collections="collections"
          :offerings="offerings"
          :catalog="catalog"
          :git-repos="gitRepos"
          :platform-git-repo="platformGitRepo"
          :platform-git-branch="platformGitBranch"
          :initial-collection-id="wizardCollectionId"
          :api-fn="apiFn"
          @cancel="$emit('cancel-wizard')"
          @submit="$emit('submit-request', $event)"
        />

        <section v-if="!showWizard && collections.length" class="dp-admin-request-catalog">
          <h3><i class="icon icon-apps" /> Browse catalog</h3>
          <div class="dp-admin-catalog-grid">
            <button
              v-for="col in collections"
              :key="col.id"
              class="dp-admin-catalog-card"
              type="button"
              @click="$emit('start-wizard', col.id)"
            >
              <i :class="['icon', collectionIcon(col.icon)]" />
              <strong>{{ col.label || col.id }}</strong>
              <span>{{ col.description }}</span>
              <span class="dp-admin-catalog-count">{{ offeringsInCollection(col.id).length }} offering(s)</span>
            </button>
          </div>
        </section>
      </div>

      <!-- Platform requests -->
      <div v-else-if="activeTab === 'requests'" class="dp-admin-tab-panel">
        <section v-if="statusFilter === 'new' && filteredPendingRequests.length" class="dp-admin-queue">
          <div class="dp-section-head">
            <h2><i class="icon icon-checkmark" /> Needs your approval <span class="dp-count">{{ filteredPendingRequests.length }}</span></h2>
          </div>
          <div class="dp-approval-queue">
            <article v-for="r in filteredPendingRequests" :key="r.crName || r.name" class="dp-approval-card">
              <div class="dp-approval-card-body">
                <strong>{{ r.displayName || r.name }}</strong>
                <span class="dp-approval-meta">
                  {{ r.requester || 'unknown' }} · {{ r.template || r.offeringId }} ·
                  <code>{{ r.gitPath || `environments/${r.name}` }}</code>
                </span>
              </div>
              <div class="dp-approval-card-actions">
                <button class="btn role-secondary xs" type="button" @click="$emit('review-request', r)">Review</button>
                <button class="btn role-primary xs" type="button" @click="$emit('approve-request', r)">Approve</button>
                <button class="btn role-tertiary xs" type="button" @click="$emit('reject-request', r)">Reject</button>
              </div>
            </article>
          </div>
        </section>

        <section class="dp-admin-requests">
          <div class="dp-section-head">
            <h2><i class="icon icon-list" /> Ops queue</h2>
            <button class="btn role-tertiary xs" type="button" :disabled="loading" @click="$emit('refresh')">
              <i class="icon icon-refresh" />
            </button>
          </div>

          <RequestStatusTabs
            v-model="statusFilter"
            :requests="requests"
            :needs-admin-approval="needsAdminApproval"
          />

          <table v-if="filteredRequests.length" class="dp-table dp-table-requests">
            <thead>
              <tr>
                <th class="col-expand" />
                <th>Name</th>
                <th>Requester</th>
                <th>Status</th>
                <th>Created</th>
                <th class="col-actions">Actions</th>
              </tr>
            </thead>
            <tbody v-for="r in filteredRequests" :key="r.crName || r.name">
              <tr
                class="dp-request-row"
                :class="{ expanded: expandedCrName === (r.crName || r.name) }"
                @click="$emit('toggle-detail', r)"
              >
                <td class="col-expand">
                  <i :class="['icon', expandedCrName === (r.crName || r.name) ? 'icon-chevron-down' : 'icon-chevron-right']" />
                </td>
                <td class="name">
                  <strong>{{ r.displayName || r.name }}</strong>
                  <span class="name-sub">{{ r.template || r.offeringId }}</span>
                </td>
                <td><code>{{ r.requester || '—' }}</code></td>
                <td>
                  <span :class="['phase', needsAdminApproval(r) ? 'PendingApproval' : r.phase]">{{ phaseLabelFor(r) }}</span>
                  <span v-if="r.message && expandedCrName !== (r.crName || r.name)" class="status-snippet">{{ r.message }}</span>
                </td>
                <td>{{ formatDate(r.createdAt || r.created) }}</td>
                <td class="col-actions">
                  <button
                    v-if="needsAdminApproval(r)"
                    class="btn role-primary xs"
                    type="button"
                    @click.stop="$emit('approve-request', r)"
                  >
                    Approve
                  </button>
                  <button class="btn role-tertiary xs" type="button" @click.stop="$emit('review-request', r)">
                    {{ needsAdminApproval(r) ? 'Review' : 'Details' }}
                  </button>
                </td>
              </tr>
              <tr v-if="expandedCrName === (r.crName || r.name)" class="dp-request-detail-row">
                <td colspan="6">
                  <RequestDetailPanel
                    :request="r"
                    :is-admin="true"
                    :needs-approval="needsAdminApproval(r)"
                    :detail-tab="detailTab"
                    :selected-git-file="selectedGitFile(r)"
                    :selected-git-file-content="selectedGitFileContent(r)"
                    @approve="$emit('approve-request', r)"
                    @reject="$emit('reject-request', r)"
                    @update:detail-tab="$emit('update:detailTab', $event)"
                    @select-git-file="$emit('select-git-file', r, $event)"
                  />
                </td>
              </tr>
            </tbody>
          </table>
          <p v-else-if="!loading" class="empty">{{ emptyMessage }}</p>
        </section>
      </div>

      <!-- Platform settings -->
      <div v-else-if="activeTab === 'settings'" class="dp-admin-tab-panel dp-admin-tab-panel-settings">
        <CatalogAdminSettings
          class="dp-admin-settings-panel"
          :embedded="true"
          :initial-yaml="configYaml"
          :initial-config="platformConfig"
          :api-fn="apiFn"
          :saving="configSaving"
          @reload="$emit('reload-config')"
          @save="$emit('save-config', $event)"
        />
      </div>
    </div>
  </div>
</template>

<script>
import RequestWizard from './RequestWizard.vue';
import CatalogAdminSettings from './CatalogAdminSettings.vue';
import RequestDetailPanel from './RequestDetailPanel.vue';
import RequestStatusTabs from './RequestStatusTabs.vue';
import { filterRequestsByStatus, statusTabEmptyMessage } from './requestStatus';

export default {
  name: 'DevPortalAdminView',
  components: { RequestWizard, CatalogAdminSettings, RequestDetailPanel, RequestStatusTabs },
  props: {
    authUser: { type: Object, default: null },
    loading: { type: Boolean, default: false },
    collections: { type: Array, default: () => [] },
    offerings: { type: Array, default: () => [] },
    catalog: { type: Array, default: () => [] },
    gitRepos: { type: Array, default: () => [] },
    platformGitRepo: { type: String, default: '' },
    platformGitBranch: { type: String, default: '' },
    requests: { type: Array, default: () => [] },
    pendingRequests: { type: Array, default: () => [] },
    showWizard: { type: Boolean, default: false },
    wizardCollectionId: { type: String, default: null },
    configYaml: { type: String, default: '' },
    platformConfig: { type: Object, default: null },
    configSaving: { type: Boolean, default: false },
    expandedCrName: { type: String, default: null },
    detailTab: { type: String, default: 'git' },
    apiFn: { type: Function, required: true },
    needsAdminApproval: { type: Function, required: true },
    phaseLabelFor: { type: Function, required: true },
    formatDate: { type: Function, required: true },
    selectedGitFile: { type: Function, required: true },
    selectedGitFileContent: { type: Function, required: true },
  },
  emits: [
    'start-wizard',
    'cancel-wizard',
    'submit-request',
    'reload-config',
    'save-config',
    'refresh',
    'toggle-detail',
    'review-request',
    'approve-request',
    'reject-request',
    'update:detailTab',
    'select-git-file',
    'open-settings',
  ],
  data() {
    return {
      activeTab: 'requests',
      statusFilter: 'all',
    };
  },
  computed: {
    mainTabs() {
      return [
        { id: 'request-env', label: 'Request env', icon: 'icon-circle-plus' },
        { id: 'requests', label: 'Ops queue', icon: 'icon-list' },
        { id: 'settings', label: 'Catalog & config', icon: 'icon-cog' },
      ];
    },
    filteredRequests() {
      return filterRequestsByStatus(this.requests, this.statusFilter, this.needsAdminApproval);
    },
    filteredPendingRequests() {
      if (this.statusFilter === 'approved' || this.statusFilter === 'declined') return [];
      return this.pendingRequests;
    },
    emptyMessage() {
      return statusTabEmptyMessage(this.statusFilter, 'admin');
    },
  },
  watch: {
    showWizard(open) {
      if (open) this.activeTab = 'request-env';
    },
  },
  methods: {
    selectTab(tabId) {
      this.activeTab = tabId;
      if (tabId === 'settings') {
        this.$emit('open-settings');
      }
    },
    collectionIcon(icon) {
      const map = { namespace: 'icon-namespace', cluster: 'icon-cluster', apps: 'icon-apps', vm: 'icon-vm', file: 'icon-file' };
      return map[icon] || 'icon-folder';
    },
    offeringsInCollection(collectionId) {
      return this.offerings.filter((o) => o.collectionId === collectionId);
    },
  },
};
</script>

<style lang="scss" scoped>
.dp-admin-view {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.dp-admin-view-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  padding: 14px 16px 10px;
  border-bottom: 1px solid var(--border);
  background: var(--sortable-table-header-bg, var(--box-bg));
  flex-shrink: 0;
}

.dp-admin-view-title {
  min-width: 0;

  h1 {
    margin: 6px 0 0;
    font-size: 1.05em;
    font-weight: 600;
  }

  .dp-admin-tagline {
    margin: 6px 0 0;
    font-size: 0.82em;
    color: var(--muted);
    line-height: 1.4;
    max-width: 520px;
  }
}

.dp-admin-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 3px;
  font-size: 0.68em;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  background: rgba(156, 39, 176, 0.15);
  color: #9c27b0;
}

.dp-admin-main-tabs {
  display: flex;
  gap: 0;
  padding: 0 16px;
  border-bottom: 1px solid var(--border);
  background: var(--sortable-table-header-bg, var(--box-bg));
  flex-shrink: 0;
}

.dp-admin-main-tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  border: none;
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
  border-radius: 0;
  background: none;
  color: var(--muted);
  font-size: 0.84em;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;

  &.active {
    color: var(--primary);
    border-bottom-color: var(--primary);
  }

  &:hover:not(.active) {
    color: var(--body-text);
  }

  .icon {
    font-size: 1em;
  }
}

.dp-admin-view-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.dp-admin-tab-panel {
  padding: 16px;
  min-height: 100%;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  text-align: left;

  &.dp-admin-tab-panel-settings {
    padding: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;

    .dp-admin-settings-panel {
      flex: 1;
      min-height: 0;
    }
  }
}

.dp-admin-request-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  padding: 18px 20px;
  margin-bottom: 20px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: linear-gradient(135deg, var(--sortable-table-header-bg, var(--box-bg)) 0%, var(--body-bg) 100%);

  h2 {
    margin: 0 0 6px;
    font-size: 1.1em;
    font-weight: 600;
  }

  p {
    margin: 0;
    max-width: 520px;
    font-size: 0.85em;
    color: var(--muted);
    line-height: 1.5;
  }
}

.dp-admin-request-eyebrow {
  display: inline-block;
  font-size: 0.72em;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--primary);
  margin-bottom: 6px;
}

.dp-admin-request-catalog {
  h3 {
    margin: 0 0 12px;
    font-size: 0.92em;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 8px;

    .icon { color: var(--primary); }
  }
}

.dp-admin-catalog-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.dp-admin-catalog-card {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  padding: 16px;
  text-align: left;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--body-bg);
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;

  .icon {
    font-size: 1.4em;
    color: var(--primary);
  }

  strong {
    font-size: 0.92em;
  }

  span {
    font-size: 0.8em;
    color: var(--muted);
    line-height: 1.4;
  }

  .dp-admin-catalog-count {
    margin-top: 4px;
    font-size: 0.72em;
    font-weight: 600;
    color: var(--primary);
  }

  &:hover {
    border-color: var(--primary);
    box-shadow: 0 2px 8px var(--shadow, rgba(0, 0, 0, 0.08));
  }
}

.dp-admin-queue,
.dp-admin-requests {
  margin-bottom: 16px;
}

.dp-section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;

  h2 {
    margin: 0;
    font-size: 0.92em;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 8px;

    .icon { color: var(--primary); }
  }
}

.dp-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 3px;
  font-size: 0.85em;
  background: rgba(230, 140, 20, 0.2);
  color: #b36b00;
}

.dp-approval-queue {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 20px;
}

.dp-approval-card {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  border: 1px solid rgba(230, 140, 20, 0.35);
  border-radius: 4px;
  background: rgba(230, 140, 20, 0.06);
}

.dp-approval-card-body strong {
  display: block;
  margin-bottom: 4px;
}

.dp-approval-meta {
  font-size: 0.8em;
  color: var(--muted);
}

.dp-approval-card-actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}
</style>
