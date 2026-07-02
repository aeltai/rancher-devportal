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

    <div v-if="detailTab === 'git' && request.gitPreview?.files?.length" class="dp-detail-panel">
      <GitManifestPreview
        :files="request.gitPreview.files"
        :git-repo="request.gitPreview.gitRepo || request.gitRepoUrl"
        :git-branch="request.gitPreview.gitBranch || request.gitBranch"
        :git-path="request.gitPreview.gitPath || request.gitPath"
        :selected-path="selectedGitFile"
        :selected-content="selectedGitFileContent"
        @select="$emit('select-git-file', $event)"
      />
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
      <YamlCodeBlock
        :value="request.manifestYaml || '—'"
        title="PlatformRequest manifest"
        icon="yaml"
        max-height="420px"
      />
    </div>
  </div>
</template>

<script>
import GitManifestPreview from './GitManifestPreview.vue';
import YamlCodeBlock from './YamlCodeBlock.vue';

export default {
  name: 'RequestDetailPanel',
  components: { GitManifestPreview, YamlCodeBlock },
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
};
</script>

<style lang="scss" scoped>
.dp-git-meta {
  margin: 12px 0 0;
  font-size: 0.82em;
  color: var(--muted);

  code {
    background: var(--sortable-table-header-bg, rgba(0, 0, 0, 0.04));
    padding: 2px 6px;
    border-radius: 3px;
  }
}

.dp-fleet-table {
  width: 100%;
  font-size: 0.82em;
  border-collapse: collapse;

  th, td {
    padding: 8px 10px;
    border-bottom: 1px solid var(--border);
    text-align: left;
  }

  th {
    font-size: 0.78em;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    color: var(--muted);
  }
}

.fleet-phase {
  font-weight: 600;
  font-size: 0.9em;
}

.empty {
  padding: 20px;
  font-size: 0.88em;
  color: var(--muted);
  text-align: center;
}
</style>
