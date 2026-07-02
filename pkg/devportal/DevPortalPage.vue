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

    <div class="dp-panel" :class="{ 'dp-panel-admin': isAdmin }">
      <DevPortalAdminView
        v-if="isAdmin"
        :auth-user="authUser"
        :loading="loading"
        :collections="collections"
        :offerings="offerings"
        :catalog="catalog"
        :git-repos="gitRepos"
        :platform-git-repo="platformGitRepo"
        :platform-git-branch="platformGitBranch"
        :requests="requests"
        :pending-requests="pendingRequests"
        :show-wizard="showWizard"
        :wizard-collection-id="wizardCollectionId"
        :config-yaml="configYaml"
        :platform-config="platformConfig"
        :config-saving="configSaving"
        :expanded-cr-name="expandedCrName"
        :detail-tab="detailTab"
        :api-fn="api"
        :needs-admin-approval="needsAdminApproval"
        :phase-label-for="phaseLabelFor"
        :format-date="formatDate"
        :selected-git-file="selectedGitFile"
        :selected-git-file-content="selectedGitFileContent"
        @start-wizard="startWizard"
        @cancel-wizard="cancelWizard"
        @submit-request="submitRequest"
        @open-settings="loadPlatformConfig"
        @reload-config="loadPlatformConfig"
        @save-config="savePlatformConfig"
        @refresh="loadRequests"
        @toggle-detail="toggleRequestDetail"
        @review-request="reviewRequest"
        @approve-request="approveRequest"
        @reject-request="rejectRequest"
        @update:detail-tab="detailTab = $event"
        @select-git-file="(r, path) => selectGitFile(r, path)"
      />

      <DevPortalUserView
        v-else
        :auth-user="authUser"
        :loading="loading"
        :collections="collections"
        :offerings="offerings"
        :catalog="catalog"
        :git-repos="gitRepos"
        :platform-git-repo="platformGitRepo"
        :platform-git-branch="platformGitBranch"
        :requests="requests"
        :show-wizard="showWizard"
        :initial-collection-id="wizardCollectionId"
        :expanded-cr-name="expandedCrName"
        :detail-tab="detailTab"
        :api-fn="api"
        :needs-admin-approval="needsAdminApproval"
        :phase-label-for="phaseLabelFor"
        :format-date="formatDate"
        :collection-icon="collectionIcon"
        :selected-git-file="selectedGitFile"
        :selected-git-file-content="selectedGitFileContent"
        @start-wizard="startWizard"
        @cancel-wizard="cancelWizard"
        @submit-request="submitRequest"
        @refresh="loadRequests"
        @toggle-detail="toggleRequestDetail"
        @update:detail-tab="detailTab = $event"
        @select-git-file="(r, path) => selectGitFile(r, path)"
      />
    </div>
  </div>
</template>

<script>
import devportalMixin from './devportalMixin';
import DevPortalAdminView from './DevPortalAdminView.vue';
import DevPortalUserView from './DevPortalUserView.vue';

export default {
  name: 'DevPortalPage',
  layout: 'plain',
  components: { DevPortalAdminView, DevPortalUserView },
  mixins: [devportalMixin],
};
</script>

<style lang="scss" scoped>
.devportal-page {
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  width: 100%;
  max-width: 100%;
  margin: 0;
  min-height: calc(100vh - 60px);
  height: calc(100vh - 60px);
  padding: 8px clamp(8px, 1.5vw, 16px);
  color: var(--body-text);
  background: var(--body-bg);
  overflow: hidden;
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
  overflow: hidden;

  &.dp-panel-admin {
    /* admin layout handled inside DevPortalAdminView */
  }
}
</style>

<style lang="scss">
/* Shared table + detail styles for admin view and request detail panel */
.devportal-page {
  .dp-table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.85em;

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
      .name-sub { display: block; font-weight: 400; font-size: 0.85em; color: var(--muted); margin-top: 2px; }
    }

    .status-snippet { display: block; font-size: 0.82em; color: var(--muted); margin-top: 2px; }
    .col-expand { width: 28px; color: var(--muted); }
    .col-actions { width: 140px; text-align: right; white-space: nowrap; }
  }

  .dp-request-row { cursor: pointer; }
  .dp-request-detail-row td {
    padding: 0 !important;
    border-bottom: 1px solid var(--border);
    background: var(--sortable-table-row-bg, var(--body-bg));
  }

  .dp-request-detail { padding: 12px 14px 16px; }

  .dp-detail-tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 10px;
    border-bottom: 1px solid var(--border);
  }

  .dp-detail-tab {
    border: none;
    background: none;
    padding: 8px 12px;
    font-size: 0.8em;
    font-weight: 600;
    color: var(--muted);
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;

    &.active {
      color: var(--primary);
      border-bottom-color: var(--primary);
    }
  }

  .dp-detail-banner {
    font-size: 0.82em;
    margin-bottom: 10px;
    padding: 8px 10px;
    border-radius: 4px;
    background: var(--default-light-bg, rgba(0, 0, 0, 0.04));
    border: 1px solid var(--border);
  }

  .dp-approval-banner {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 10px;
    justify-content: space-between;
  }

  .dp-approval-actions { display: flex; gap: 8px; }

  .phase {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 3px;
    font-size: 0.85em;
    font-weight: 600;
    &.Ready { background: rgba(63, 138, 63, 0.15); color: var(--success, #3f8a3f); }
    &.Provisioning, &.Reconciling { background: rgba(0, 100, 200, 0.12); color: var(--primary); }
    &.Failed { background: rgba(204, 74, 74, 0.15); color: var(--error, #c00); }
    &.Pending, &.PendingApproval { background: rgba(230, 140, 20, 0.15); color: #b36b00; }
    &.Approved { background: rgba(0, 100, 200, 0.12); color: var(--primary); }
    &.Rejected { background: rgba(204, 74, 74, 0.15); color: var(--error, #c00); }
    &.Pushing { background: rgba(0, 100, 200, 0.12); color: var(--primary); }
  }

  .dp-fleet-table {
    width: 100%;
    font-size: 0.82em;
    border-collapse: collapse;
    th, td { padding: 6px 8px; border-bottom: 1px solid var(--border); text-align: left; }
  }

  .empty {
    padding: 16px;
    font-size: 0.85em;
    color: var(--muted);
    text-align: center;
  }
}
</style>
