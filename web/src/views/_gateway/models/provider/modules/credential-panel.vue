<script setup lang="tsx">
import { ref, watch } from 'vue';
import { NTag } from 'naive-ui';
import {
  fetchBatchDeleteCredential,
  fetchGetCredentialList,
  fetchResyncCredentials,
  fetchUpdateCredential
} from '@/service/api/gateway';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { CREDENTIAL_FORMAT_OPTIONS } from '@/constants/business/gateway';
import CredentialOperateDialog from './credential-operate-dialog.vue';

interface Props {
  provider: Api.Gateway.Provider;
}
const props = defineProps<Props>();

interface Emits {
  (e: 'changed'): void;
  (e: 'toggleProvider'): void;
}
const emit = defineEmits<Emits>();

const credentialList = ref<Api.Gateway.Credential[]>([]);
const loading = ref(false);

async function getCredentialData() {
  loading.value = true;
  const { data, error } = await fetchGetCredentialList({
    pageNum: 1,
    pageSize: 100,
    credentialName: null,
    providerId: props.provider.providerId!,
    isActive: null,
    litellmSynced: null,
    params: {}
  });
  if (!error && data) credentialList.value = data.rows;
  loading.value = false;
}

watch(() => props.provider.providerId, getCredentialData, { immediate: true });

const formatLabelKey = (f?: string) => CREDENTIAL_FORMAT_OPTIONS.find(o => o.value === f)?.label ?? 'page.gateway.common.formatOpenai';

const columns = [
  { key: 'credentialName', title: () => $t('page.gateway.credential.col.credentialName'), minWidth: 60, ellipsis: { tooltip: true } },
  {
    key: 'format',
    title: () => $t('page.gateway.credential.col.format'),
    width: 100,
    render: (row: Api.Gateway.Credential) => <NTag size="small">{$t(formatLabelKey(row.credentialInfo?.format))}</NTag>
  },
  {
    key: 'apiBase',
    title: () => $t('page.gateway.credential.col.apiBase'),
    minWidth: 100,
    ellipsis: { tooltip: true },
    render: (row: Api.Gateway.Credential) => <span>{row.credentialValues?.api_base || '-'}</span>
  },
  {
    key: 'isActive',
    title: () => $t('page.gateway.credential.col.isActive'),
    width: 60,
    render: (row: Api.Gateway.Credential) => (
      <NTag size="small" type={row.isActive ? 'success' : 'default'}>
        {$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}
      </NTag>
    )
  },
  {
    key: 'operate',
    title: () => $t('common.operate'),
    width: 160,
    render: (row: Api.Gateway.Credential) => (
      <div class="flex-center gap-4px">
        <ButtonIcon
          text
          type={row.isActive ? 'warning' : 'success'}
          size="small"
          icon={row.isActive ? 'material-symbols:pause-circle-outline' : 'material-symbols:play-circle-outline'}
          tooltip-content={$t(row.isActive ? 'common.disable' : 'common.enable')}
          onClick={() => handleToggle(row)}
        />
        <ButtonIcon
          text
          type="primary"
          size="small"
          icon="material-symbols:drive-file-rename-outline-outline"
          tooltip-content={$t('common.edit')}
          onClick={() => handleEdit(row)}
        />
        <ButtonIcon
          text
          type="info"
          size="small"
          icon="material-symbols:sync-rounded"
          tooltip-content={$t('page.gateway.credential.resync')}
          onClick={() => handleResync()}
        />
        <ButtonIcon
          text
          type="error"
          size="small"
          icon="material-symbols:delete-outline"
          tooltip-content={$t('common.delete')}
          popconfirm-content={$t('common.confirmDelete')}
          onPositiveClick={() => handleDelete(row)}
        />
      </div>
    )
  }
];

const dialogVisible = ref(false);
const operateType = ref<NaiveUI.TableOperateType>('add');
const editingData = ref<Api.Gateway.Credential | null>(null);

function handleAdd() {
  operateType.value = 'add';
  editingData.value = null;
  dialogVisible.value = true;
}
function handleEdit(row: Api.Gateway.Credential) {
  operateType.value = 'edit';
  editingData.value = row;
  dialogVisible.value = true;
}
async function handleToggle(row: Api.Gateway.Credential) {
  const { error } = await fetchUpdateCredential({ ...row, isActive: !row.isActive });
  if (error) return;
  emit('changed');
  getCredentialData();
}
async function handleDelete(row: Api.Gateway.Credential) {
  const { error } = await fetchBatchDeleteCredential([row.credentialId!]);
  if (error) return;
  emit('changed');
  getCredentialData();
}
async function handleResync() {
  const { error } = await fetchResyncCredentials();
  if (error) return;
  window.$message?.success($t('page.gateway.credential.resync'));
  getCredentialData();
}
function handleDialogSubmitted() {
  dialogVisible.value = false;
  emit('changed');
  getCredentialData();
}

const supportedFormats = () => (props.provider.supportedFormats ?? []) as string[];
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper h-full flex-col-stretch">
    <template #header>
      <div class="flex items-center gap-8px">
        <span class="font-500">{{ provider.name }}</span>
        <NTag size="small" type="info">{{ provider.providerType }}</NTag>
        <NTag v-for="fmt in supportedFormats()" :key="fmt" size="small" type="success">
          {{ $t(formatLabelKey(fmt)) }}
        </NTag>
      </div>
    </template>
    <template #header-extra>
      <NSpace :size="8">
        <NButton size="small" type="primary" @click="handleAdd">
          <template #icon>
            <icon-material-symbols-add-rounded class="text-icon" />
          </template>
          {{ $t('page.gateway.credential.add') }}
        </NButton>
        <NButton size="small" :type="provider.isActive ? 'warning' : 'success'" @click="emit('toggleProvider')">
          {{ $t(provider.isActive ? 'common.disable' : 'common.enable') }}
        </NButton>
      </NSpace>
    </template>
    <NDataTable
      :columns="columns"
      :data="credentialList"
      :loading="loading"
      size="small"
      :row-key="row => row.credentialId"
      max-height="calc(100vh - 340px)"
    />
    <CredentialOperateDialog
      v-model:visible="dialogVisible"
      :operate-type="operateType"
      :row-data="editingData"
      :provider="provider"
      @submitted="handleDialogSubmitted"
    />
  </NCard>
</template>

<style scoped></style>
