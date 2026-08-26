<script setup lang="tsx">
import { ref, watch } from 'vue';
import { NTag } from 'naive-ui';
import type { DataTableColumns } from 'naive-ui';
import { fetchBatchDeleteDeployment, fetchGetDeploymentList } from '@/service/api/gateway';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { BILLING_TYPE_OPTIONS, MODEL_CATEGORY_OPTIONS } from '@/constants/business/gateway';
import DeploymentOperateDrawer from './deployment-operate-drawer.vue';

interface Props {
  model: Api.Gateway.Model;
}
const props = defineProps<Props>();

interface Emits {
  (e: 'changed'): void;
}
const emit = defineEmits<Emits>();

const deploymentList = ref<Api.Gateway.Deployment[]>([]);
const loading = ref(false);

async function getDeploymentData() {
  loading.value = true;
  const { data, error } = await fetchGetDeploymentList({
    pageNum: 1,
    pageSize: 100,
    modelId: props.model.modelId!,
    credentialId: null,
    keyword: null,
    isActive: null,
    params: {}
  });
  if (!error && data) deploymentList.value = data.rows;
  loading.value = false;
}

watch(() => props.model.modelId, getDeploymentData, { immediate: true });

const categoryLabelKey = (v: string) => MODEL_CATEGORY_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.categoryOther';
const billingTypeLabelKey = (v: string) => BILLING_TYPE_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.billingTypeToken';

const columns: DataTableColumns<Api.Gateway.Deployment> = [
  {
    key: 'deployName',
    title: () => $t('page.gateway.deployment.col.deployName'),
    minWidth: 140,
    ellipsis: { tooltip: true }
  },
  {
    key: 'credentialName',
    title: () => $t('page.gateway.deployment.col.credential'),
    minWidth: 140,
    ellipsis: { tooltip: true },
    render: row => row.credentialName || <span class="text-slate-400">{$t('page.gateway.deployment.inlineParams')}</span>
  },
  {
    key: 'billingType',
    title: () => $t('page.gateway.deployment.col.billingType'),
    align: 'center',
    width: 110,
    render: row => <NTag size="small" type="info">{$t(billingTypeLabelKey(row.billingType))}</NTag>
  },
  {
    key: 'costPerCall',
    title: () => $t('page.gateway.deployment.col.costPerCall'),
    align: 'center',
    width: 100,
    render: row => (row.costPerCall == null ? <span class="text-slate-400">{$t('page.gateway.common.unlimited')}</span> : `¥${row.costPerCall}`)
  },
  {
    key: 'monthlyCallQuota',
    title: () => $t('page.gateway.deployment.col.monthlyCallQuota'),
    align: 'center',
    width: 120,
    render: row =>
      row.monthlyCallQuota == null ? (
        <span class="text-slate-400">{$t('page.gateway.common.unlimited')}</span>
      ) : (
        `${row.monthlyCallUsed ?? 0}/${row.monthlyCallQuota}`
      )
  },
  {
    key: 'isActive',
    title: () => $t('page.gateway.deployment.col.isActive'),
    align: 'center',
    width: 80,
    render: row => <NTag size="small" type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}</NTag>
  },
  {
    key: 'operate',
    title: () => $t('common.operate'),
    align: 'center',
    width: 110,
    render: row => (
      <div class="flex-center gap-4px">
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

const drawerVisible = ref(false);
const operateType = ref<NaiveUI.TableOperateType>('add');
const editingData = ref<Api.Gateway.Deployment | null>(null);

function handleAdd() {
  operateType.value = 'add';
  editingData.value = null;
  drawerVisible.value = true;
}
function handleEdit(row: Api.Gateway.Deployment) {
  operateType.value = 'edit';
  editingData.value = row;
  drawerVisible.value = true;
}
async function handleDelete(row: Api.Gateway.Deployment) {
  const { error } = await fetchBatchDeleteDeployment([row.deploymentId!]);
  if (error) return;
  emit('changed');
  getDeploymentData();
}
function handleDrawerSubmitted() {
  drawerVisible.value = false;
  emit('changed');
  getDeploymentData();
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper h-full flex-col-stretch">
    <template #header>
      <div class="flex items-center gap-8px">
        <span class="font-500">{{ model.name }}</span>
        <NTag size="small" type="info">{{ $t(categoryLabelKey(model.category)) }}</NTag>
        <NTag size="small" :type="model.isPublished ? 'success' : 'default'">
          {{ $t(model.isPublished ? 'page.gateway.common.published' : 'page.gateway.common.unpublished') }}
        </NTag>
        <NTag size="small" :type="model.isActive ? 'success' : 'default'">
          {{ $t(model.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive') }}
        </NTag>
        <NTag v-for="cap in model.capabilities ?? []" :key="cap" size="small">{{ cap }}</NTag>
      </div>
    </template>
    <template #header-extra>
      <NSpace :size="8">
        <NButton size="small" type="primary" @click="handleAdd">
          <template #icon>
            <icon-material-symbols-add-rounded class="text-icon" />
          </template>
          {{ $t('page.gateway.deployment.add') }}
        </NButton>
      </NSpace>
    </template>
    <div class="mb-8px flex items-center gap-12px text-12px text-slate-400">
      <span>{{ model.modelKey }}</span>
      <span v-if="model.description">· {{ model.description }}</span>
    </div>
    <NDataTable
      :columns="columns"
      :data="deploymentList"
      :loading="loading"
      size="small"
      :row-key="row => row.deploymentId"
      max-height="calc(100vh - 360px)"
    />
    <DeploymentOperateDrawer
      v-model:visible="drawerVisible"
      :operate-type="operateType"
      :row-data="editingData"
      :model-id="model.modelId"
      @submitted="handleDrawerSubmitted"
    />
  </NCard>
</template>

<style scoped></style>
