<template>
  <div class="dp-request-detail">
    <div v-if="isAdmin && needsApproval" class="dp-detail-banner dp-approval-banner">
      <span><strong>Admin approval</strong> — review Git files, then approve to provision.</span>
      <div class="dp-approval-actions">
        <button class="btn role-primary xs" type="button" @click="$emit('approve')">Approve</button>
        <button class="btn role-secondary xs" type="button" @click="$emit('reject')">Reject</button>
      </div>
    </div>

    <div class="dp-detail-tabs">
      <button
        v-for="tab in visibleTabs"
        :key="tab.id"
        :class="['dp-detail-tab', { active: detailTab === tab.id }]"
        type="button"
        @click="$emit('update:detailTab', tab.id)"
      >
        {{ tab.label }}
      </button>
    </div>

    <div v-if="detailTab === 'git' && request.gitPreview?.files?.length" class="dp-detail-panel dp-git-preview">
      <p v-if="request.gitPreview.gitRepo" class="dp-git-meta">
        <code>{{ request.gitPreview.gitRepo }}</code>
        <span v-if="request.gitPreview.gitBranch"> · {{ request.gitPreview.gitBranch }}</span>
      </p>
      <div class="dp-git-preview-layout">
        <pre class="dp-yaml dp-yaml-tree dp-yaml-compact"><code>{{ request.gitPreview.tree }}</code></pre>
        <div class="dp-git-file-pane">
          <div class="dp-git-file-tabs">
            <button
              v-for="f in request.gitPreview.files"
              :key="f.path"
              :class="['dp-git-file-tab', { active: selectedGitFile === f.path }]"
              type="button"
              @click="$emit('select-git-file', f.path)"
            >
              {{ fileBaseName(f.path) }}
            </button>
          </div>
          <pre class="dp-yaml dp-yaml-compact"><code>{{ selectedGitFileContent }}</code></pre>
        </div>
      </div>
    </div>
    <p v-else-if="detailTab === 'git'" class="empty">No Git manifests for this request.</p>

    <div v-if="detailTab === 'resources'" class="dp-detail-panel">
      <table v-if="request.fleetResources?.length" class="dp-fleet-table">
        <thead>
          <tr>
            <th>Kind</th>
            <th>Name</th>
            <th>Path</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(f, idx) in request.fleetResources" :key="idx">
            <td><code>{{ f.kind }}</code></td>
            <td>{{ f.name }}</td>
            <td><code>{{ f.path || '—' }}</code></td>
            <td><span :class="['fleet-phase', f.phase]">{{ f.phase }}</span></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="empty">No Fleet resources yet.</p>
      <p v-if="request.gitCommit" class="dp-git-meta">
        Commit <code>{{ request.gitCommit }}</code>
        <span v-if="request.fleetGitRepoName"> · {{ request.fleetGitRepoName }}</span>
      </p>
    </div>

    <div v-if="detailTab === 'yaml'" class="dp-detail-panel">
      <pre class="dp-yaml dp-yaml-compact"><code>{{ request.manifestYaml || '—' }}</code></pre>
    </div>
  </div>
</template>

<script>
export default {
  name: 'RequestDetailPanel',
  props: {
    request: { type: Object, required: true },
    isAdmin: { type: Boolean, default: false },
    needsApproval: { type: Boolean, default: false },
    detailTab: { type: String, default: 'resources' },
    selectedGitFile: { type: String, default: '' },
    selectedGitFileContent: { type: String, default: '—' },
  },
  emits: ['approve', 'reject', 'update:detailTab', 'select-git-file'],
  computed: {
    visibleTabs() {
      const tabs = [
        { id: 'resources', label: 'Resources' },
        { id: 'yaml', label: 'YAML' },
      ];
      if (this.isAdmin || this.request.gitPreview?.files?.length) {
        tabs.unshift({ id: 'git', label: 'Git preview' });
      }
      return tabs;
    },
  },
  methods: {
    fileBaseName(path) {
      return path.split('/').pop() || path;
    },
  },
};
</script>
