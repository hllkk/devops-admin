<script setup lang="tsx">
import { ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import { useBoolean } from '@sa/hooks';
import { fetchBatchDeleteModel, fetchGetModelList } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { MODEL_CATEGORY_OPTIONS } from '@/constants/business/gateway';
import ModelSearch from './modules/model-search.vue';
import ModelOperateDrawer from './modules/model-operate-drawer.vue';
import DeploymentManageDrawer from './modules/deployment-manage-drawer.vue';

defineOptions({
  name: 'ModelList'
});

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.ModelSearchParams>({
  pageNum: 1,
  pageSize: 10,
  name: null,
  modelKey: null,
  category: null,
  isActive: null,
  isPublished: null,
  params: {}
});

/** 部署管理抽屉(行内点「部署」打开) */
const { bool: deployDrawerVisible, setTrue: openDeployDrawer } = useBoolean();
const deployModel = ref<{ modelId: CommonType.IdType; modelKey: string } | null>(null);

function handleManageDeploy(row: Api.Gateway.Model) {
  deployModel.value = { modelId: row.modelId!, modelKey: row.modelKey };
  openDeployDrawer();
}

const categoryLabelKey = (v: string) => MODEL_CATEGORY_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.categoryChat';

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => fetchGetModelList(searchParams.value),
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.pageNum = params.page;
    searchParams.value.pageSize = params.pageSize;
  },
  columns: () => [
    {
      type: 'selection',
      align: 'center',
      width: 48
    },
    {
      key: 'index',
      title: $t('common.index'),
      align: 'center',
      width: 64,
      render: (_, index) => index + 1
    },
    {
      key: 'name',
      title: $t('page.gateway.model.col.name'),
      align: 'center',
      minWidth: 140,
      ellipsis: { tooltip: true }
    },
    {
      key: 'modelKey',
      title: $t('page.gateway.model.col.modelKey'),
      align: 'center',
      minWidth: 140,
      ellipsis: { tooltip: true }
    },
    {
      key: 'category',
      title: $t('page.gateway.model.col.category'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag type="info">{$t(categoryLabelKey(row.category))}</NTag>
    },
    {
      key: 'capabilities',
      title: $t('page.gateway.model.col.capabilities'),
      align: 'center',
      minWidth: 180,
      render: row => (row.capabilities ?? []).map((c, i) => <NTag key={i} size="small" class="mr-4px">{c}</NTag>)
    },
    {
      key: 'deploymentCount',
      title: $t('page.gateway.model.col.deploymentCount'),
      align: 'center',
      minWidth: 110,
      render: row => `${row.activeDeploymentCount ?? 0}/${row.deploymentCount ?? 0}`
    },
    {
      key: 'isPublished',
      title: $t('page.gateway.model.col.isPublished'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag type={row.isPublished ? 'success' : 'default'}>{$t(row.isPublished ? 'page.gateway.common.published' : 'page.gateway.common.unpublished')}</NTag>
    },
    {
      key: 'isActive',
      title: $t('page.gateway.model.col.isActive'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}</NTag>
    },
    {
      key: 'createTime',
      title: $t('page.gateway.common.createTime'),
      align: 'center',
      minWidth: 170,
      render: row => <NTime time={Date.parse(row.createTime)} format="yyyy-MM-dd HH:mm:ss" />
    },
    {
      key: 'operate',
      title: $t('common.operate'),
      align: 'center',
      width: 220,
      render: row => {
        const editBtn = () => (
          <ButtonIcon
            text
            type="primary"
            icon="material-symbols:drive-file-rename-outline-outline"
            tooltipContent={$t('common.edit')}
            onClick={() => edit(row.modelId!)}
          />
        );

        const deployBtn = () => (
          <ButtonIcon
            text
            type="primary"
            icon="material-symbols:device-hub"
            tooltipContent={$t('page.gateway.deployment.manageTitle')}
            onClick={() => handleManageDeploy(row)}
          />
        );

        const deleteBtn = () => (
          <ButtonIcon
            text
            type="error"
            icon="material-symbols:delete-outline"
            tooltipContent={$t('common.delete')}
            popconfirmContent={$t('common.confirmDelete')}
            onPositiveClick={() => handleDelete(row.modelId!)}
          />
        );

        return (
          <div class="flex-center gap-8px">
            {editBtn()}
            {deployBtn()}
            {deleteBtn()}
          </div>
        );
      }
    }
  ]
});

const { drawerVisible, operateType, editingData, handleAdd, handleEdit, checkedRowKeys, onBatchDeleted, onDeleted } =
  useTableOperate(data, 'modelId', getData);

function edit(modelId: CommonType.IdType) {
  handleEdit(modelId);
}

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteModel(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

async function handleDelete(modelId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteModel([modelId]);
  if (error) return;
  onDeleted();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <ModelSearch v-model:model="searchParams" @reset="getDataByPage" @search="getDataByPage" />
    <NCard :title="$t('page.gateway.model.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <TableHeaderOperation
          v-model:columns="columnChecks"
          :disabled-delete="checkedRowKeys.length === 0"
          :loading="loading"
          :show-add="true"
          :show-delete="true"
          @add="handleAdd"
          @delete="handleBatchDelete"
          @refresh="getData"
        />
      </template>
      <NDataTable
        v-model:checked-row-keys="checkedRowKeys"
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="scrollX"
        :loading="loading"
        remote
        :row-key="row => row.modelId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
      <ModelOperateDrawer
        v-model:visible="drawerVisible"
        :operate-type="operateType"
        :row-data="editingData"
        @submitted="getData"
      />
      <DeploymentManageDrawer
        v-model:visible="deployDrawerVisible"
        :model-id="deployModel?.modelId"
        :model-key="deployModel?.modelKey"
      />
    </NCard>
  </div>
</template>

<style scoped></style>
