<template>
  <div class="dp-user-view">
    <header class="dp-user-hero">
      <div class="dp-user-hero-text">
        <span class="dp-user-eyebrow">Geeko-Ops</span>
        <h1>Welcome{{ authUser ? `, ${authUser.displayName || authUser.username}` : '' }}</h1>
        <p>Browse the ops marketplace — request namespaces, Helm stacks, VMs, and Fleet-managed environments in minutes.</p>
      </div>
      <button v-if="!showWizard" class="btn role-primary dp-user-cta" type="button" @click="$emit('start-wizard')">
        <i class="icon icon-circle-plus" /> Request environment
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
      :initial-collection-id="initialCollectionId"
      :api-fn="apiFn"
      @cancel="$emit('cancel-wizard')"
      @submit="$emit('submit-request', $event)"
    />

    <section v-if="!showWizard && collections.length" class="dp-user-catalog">
      <div class="dp-user-section-head">
        <div>
          <h2><i class="icon icon-apps" /> Ops marketplace</h2>
          <p>Choose a collection and start a request.</p>
        </div>
      </div>
      <div class="dp-user-catalog-grid">
        <button
          v-for="col in collections"
          :key="col.id"
          class="dp-user-catalog-card"
          type="button"
          @click="$emit('start-wizard', col.id)"
        >
          <i :class="['icon', collectionIcon(col.icon)]" />
          <strong>{{ col.label || col.id }}</strong>
          <span>{{ col.description }}</span>
          <span class="dp-user-catalog-count">{{ offeringsInCollection(col.id).length }} offering(s)</span>
        </button>
      </div>
    </section>

    <section v-if="!showWizard" class="dp-user-environments">
      <div class="dp-user-section-head">
        <div>
          <h2><i class="icon icon-namespace" /> My environments</h2>
          <p>Track status of your platform requests.</p>
        </div>
        <button class="btn role-tertiary xs" type="button" :disabled="loading" @click="$emit('refresh')">
          <i class="icon icon-refresh" />
        </button>
      </div>

        <RequestStatusTabs
          v-model="statusFilter"
          variant="user"
          :requests="requests"
          :needs-admin-approval="needsAdminApproval"
        />

      <div v-if="filteredRequests.length" class="dp-user-env-grid">
        <article
          v-for="r in filteredRequests"
          :key="r.crName || r.name"
          class="dp-user-env-card"
          :class="{ expanded: expandedCrName === (r.crName || r.name) }"
        >
          <div class="dp-user-env-card-main" @click="$emit('toggle-detail', r)">
            <div class="dp-user-env-card-top">
              <strong>{{ r.displayName || r.name }}</strong>
              <span :class="['phase', needsAdminApproval(r) ? 'PendingApproval' : r.phase]">
                {{ phaseLabelFor(r) }}
              </span>
            </div>
            <p class="dp-user-env-meta">{{ r.template || r.offeringId || 'environment' }}</p>
            <p v-if="r.message" class="dp-user-env-message">{{ r.message }}</p>
            <div class="dp-user-env-footer">
              <span>{{ formatDate(r.createdAt || r.created) }}</span>
              <span class="dp-user-env-action">
                {{ expandedCrName === (r.crName || r.name) ? 'Hide details' : 'View details' }}
                <i :class="['icon', expandedCrName === (r.crName || r.name) ? 'icon-chevron-up' : 'icon-chevron-down']" />
              </span>
            </div>
          </div>
          <div v-if="expandedCrName === (r.crName || r.name)" class="dp-user-env-detail">
            <RequestDetailPanel
              :request="r"
              :is-admin="false"
              :needs-approval="needsAdminApproval(r)"
              :detail-tab="detailTab"
              :selected-git-file="selectedGitFile(r)"
              :selected-git-file-content="selectedGitFileContent(r)"
              @update:detail-tab="$emit('update:detailTab', $event)"
              @select-git-file="$emit('select-git-file', r, $event)"
            />
          </div>
        </article>
      </div>

      <div v-else-if="!loading" class="dp-user-empty">
        <i class="icon icon-namespace" />
        <h3>{{ emptyTitle }}</h3>
        <p>{{ emptyMessage }}</p>
        <button v-if="statusFilter === 'all' || statusFilter === 'new'" class="btn role-primary" type="button" @click="$emit('start-wizard')">
          Request environment
        </button>
      </div>
    </section>
  </div>
</template>

<script>
import RequestWizard from './RequestWizard.vue';
import RequestDetailPanel from './RequestDetailPanel.vue';
import RequestStatusTabs from './RequestStatusTabs.vue';
import { filterRequestsByStatus, statusTabEmptyMessage } from './requestStatus';

export default {
  name: 'DevPortalUserView',
  components: { RequestWizard, RequestDetailPanel, RequestStatusTabs },
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
    showWizard: { type: Boolean, default: false },
    initialCollectionId: { type: String, default: null },
    expandedCrName: { type: String, default: null },
    detailTab: { type: String, default: 'resources' },
    apiFn: { type: Function, required: true },
    needsAdminApproval: { type: Function, required: true },
    phaseLabelFor: { type: Function, required: true },
    formatDate: { type: Function, required: true },
    collectionIcon: { type: Function, required: true },
    selectedGitFile: { type: Function, required: true },
    selectedGitFileContent: { type: Function, required: true },
  },
  emits: [
    'start-wizard',
    'cancel-wizard',
    'submit-request',
    'refresh',
    'toggle-detail',
    'update:detailTab',
    'select-git-file',
  ],
  data() {
    return {
      statusFilter: 'all',
    };
  },
  computed: {
    filteredRequests() {
      return filterRequestsByStatus(this.requests, this.statusFilter, this.needsAdminApproval);
    },
    emptyMessage() {
      const base = statusTabEmptyMessage(this.statusFilter, 'user');
      if (this.statusFilter === 'all' && !this.requests.length) {
        return `${base} Start by requesting an environment from the catalog.`;
      }
      return base;
    },
    emptyTitle() {
      if (this.statusFilter === 'all' && !this.requests.length) return 'No environments yet';
      if (this.statusFilter === 'new') return 'Nothing awaiting approval';
      if (this.statusFilter === 'approved') return 'No active environments';
      if (this.statusFilter === 'declined') return 'No declined requests';
      return 'No matching requests';
    },
  },
  methods: {
    offeringsInCollection(collectionId) {
      return this.offerings.filter((o) => o.collectionId === collectionId);
    },
  },
};
</script>

<style lang="scss" scoped>
.dp-user-view {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 16px;
}

.dp-user-hero {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  padding: 20px 22px;
  margin-bottom: 20px;
  border: 1px solid var(--border);
  border-radius: 6px;
  background: linear-gradient(135deg, var(--sortable-table-header-bg, var(--box-bg)) 0%, var(--body-bg) 100%);
}

.dp-user-eyebrow {
  display: inline-block;
  font-size: 0.72em;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--primary);
  margin-bottom: 6px;
}

.dp-user-hero-text h1 {
  margin: 0 0 8px;
  font-size: 1.35em;
  font-weight: 600;
}

.dp-user-hero-text p {
  margin: 0;
  max-width: 520px;
  font-size: 0.88em;
  color: var(--muted);
  line-height: 1.5;
}

.dp-user-cta {
  flex-shrink: 0;
  margin-top: 4px;
}

.dp-user-section-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;

  h2 {
    margin: 0;
    font-size: 0.95em;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 8px;

    .icon { color: var(--primary); }
  }

  p {
    margin: 4px 0 0;
    font-size: 0.82em;
    color: var(--muted);
  }
}

.dp-user-catalog {
  margin-bottom: 24px;
}

.dp-user-catalog-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 12px;
}

.dp-user-catalog-card {
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

  .dp-user-catalog-count {
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

.dp-user-env-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.dp-user-env-card {
  border: 1px solid var(--border);
  border-radius: 6px;
  background: var(--body-bg);
  overflow: hidden;

  &.expanded {
    border-color: var(--primary);
  }
}

.dp-user-env-card-main {
  padding: 14px 16px;
  cursor: pointer;

  &:hover {
    background: var(--sortable-table-hover-bg, rgba(0, 0, 0, 0.02));
  }
}

.dp-user-env-card-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 4px;

  strong {
    font-size: 0.95em;
  }
}

.dp-user-env-meta {
  margin: 0 0 6px;
  font-size: 0.82em;
  color: var(--muted);
}

.dp-user-env-message {
  margin: 0 0 8px;
  font-size: 0.8em;
  color: var(--body-text);
  line-height: 1.4;
}

.dp-user-env-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 0.78em;
  color: var(--muted);
}

.dp-user-env-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--primary);
  font-weight: 600;
}

.dp-user-env-detail {
  padding: 0 16px 16px;
  border-top: 1px solid var(--border);
  background: var(--sortable-table-row-bg, var(--body-bg));
}

.dp-user-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 48px 24px;
  border: 1px dashed var(--border);
  border-radius: 6px;
  text-align: center;
  color: var(--muted);

  .icon {
    font-size: 2em;
    opacity: 0.5;
  }

  h3 {
    margin: 0;
    font-size: 1em;
    color: var(--body-text);
  }

  p {
    margin: 0 0 8px;
    font-size: 0.85em;
  }
}

.phase {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 3px;
  font-size: 0.78em;
  font-weight: 600;
  white-space: nowrap;

  &.Ready { background: rgba(63, 138, 63, 0.15); color: var(--success, #3f8a3f); }
  &.Provisioning, &.Reconciling { background: rgba(0, 100, 200, 0.12); color: var(--primary); }
  &.Failed { background: rgba(204, 74, 74, 0.15); color: var(--error, #c00); }
  &.Pending, &.PendingApproval { background: rgba(230, 140, 20, 0.15); color: #b36b00; }
  &.Approved { background: rgba(0, 100, 200, 0.12); color: var(--primary); }
  &.Rejected { background: rgba(204, 74, 74, 0.15); color: var(--error, #c00); }
  &.Pushing { background: rgba(0, 100, 200, 0.12); color: var(--primary); }
}
</style>
