<script setup lang="tsx">
import { ref, watch } from 'vue';
import { NTag } from 'naive-ui';
import { fetchBatchDeleteDeployment, fetchGetDeploymentList } from '@/service/api/gateway';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { BILLING_TYPE_OPTIONS } from '@/constants/business/gateway';
import DeploymentOperateDrawer from './deployment-operate-drawer.vue';

defineOptions({ name: 'DeploymentManageDrawer' });

interface Props {
  /** 关联模型ID */
  modelId?: CommonType.IdType | null;
  /** 关联模型路由名(标题展示) */
  modelKey?: string;
}

const props = defineProps<Props>();

const visible = defineModel<boolean>('visible', {
  default: false
});

const searchParams = ref<Api.Gateway.DeploymentSearchParams>({
  pageNum: 1,
  pageSize: 10,
  modelId: props.modelId ?? null,
  credentialId: null,
  keyword: null,
  isActive: null,
  params: {}
});

watch(
  () => props.modelId,
  v => {
    searchParams.value.modelId = v ?? null;
  }
);

const billingTypeLabelKey = (v: string) => BILLING_TYPE_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.billingTypeToken';

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => fetchGetDeploymentList(searchParams.value),
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.pageNum = params.page;
    searchParams.value.pageSize = params.pageSize;
  },
  columns: () => [
    {
      key: 'index',
      title: $t('common.index'),
      align: 'center',
      width: 64,
      render: (_, index) => index + 1
    },
    {
      key: 'deployName',
      title: $t('page.gateway.deployment.col.deployName'),
      align: 'center',
      minWidth: 140,
      ellipsis: { tooltip: true }
    },
    {
      key: 'credentialName',
      title: $t('page.gateway.deployment.col.credential'),
      align: 'center',
      minWidth: 140,
      ellipsis: { tooltip: true },
      render: row => row.credentialName || <span class="text-slate-400">{$t('page.gateway.deployment.inlineParams')}</span>
    },
    {
      key: 'billingType',
      title: $t('page.gateway.deployment.col.billingType'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag type="info">{$t(billingTypeLabelKey(row.billingType))}</NTag>
    },
    {
      key: 'costPerCall',
      title: $t('page.gateway.deployment.col.costPerCall'),
      align: 'center',
      minWidth: 100,
      render: row => (row.costPerCall == null ? <span class="text-slate-400">{$t('page.gateway.common.unlimited')}</span> : `¥${row.costPerCall}`)
    },
    {
      key: 'monthlyCallQuota',
      title: $t('page.gateway.deployment.col.monthlyCallQuota'),
      align: 'center',
      minWidth: 120,
      render: row =>
        row.monthlyCallQuota == null ? (
          <span class="text-slate-400">{$t('page.gateway.common.unlimited')}</span>
        ) : (
          `${row.monthlyCallUsed ?? 0}/${row.monthlyCallQuota}`
        )
    },
    {
      key: 'isActive',
      title: $t('page.gateway.deployment.col.isActive'),
      align: 'center',
      minWidth: 100,
      render: row => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}</NTag>
    },
    {
      key: 'operate',
      title: $t('common.operate'),
      align: 'center',
      width: 160,
      render: row => {
        const editBtn = () => (
          <ButtonIcon
            text
            type="primary"
            icon="material-symbols:drive-file-rename-outline-outline"
            tooltipContent={$t('common.edit')}
            onClick={() => handleEdit(row.deploymentId!)}
          />
        );

        const deleteBtn = () => (
          <ButtonIcon
            text
            type="error"
            icon="material-symbols:delete-outline"
            tooltipContent={$t('common.delete')}
            popconfirmContent={$t('common.confirmDelete')}
            onPositiveClick={() => handleDelete(row.deploymentId!)}
          />
        );

        return (
          <div class="flex-center gap-8px">
            {editBtn()}
            {deleteBtn()}
          </div>
        );
      }
    }
  ]
});

const { drawerVisible, operateType, editingData, handleAdd, handleEdit, onDeleted } = useTableOperate(data, 'deploymentId', getData);

async function handleDelete(deploymentId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteDeployment([deploymentId]);
  if (error) return;
  onDeleted();
}

watch(visible, () => {
  if (visible.value) {
    getDataByPage();
  }
});
</script>

<template>
  <NDrawer v-model:show="visible" :title="$t('page.gateway.deployment.manageTitle')" display-directive="show" :width="1000" class="max-w-90%">
    <NDrawerContent :title="`${$t('page.gateway.deployment.manageTitle')}${modelKey ? ` · ${modelKey}` : ''}`" :native-scrollbar="false" closable>
      <div class="flex-col-stretch gap-12px">
        <div class="flex justify-end">
          <TableHeaderOperation
            v-model:columns="columnChecks"
            :loading="loading"
            :show-add="true"
            :show-delete="false"
            @add="handleAdd"
            @refresh="getData"
          />
        </div>
        <NDataTable
          :columns="columns"
          :data="data"
          size="small"
          :scroll-x="scrollX"
          :loading="loading"
          remote
          :row-key="row => row.deploymentId"
          :pagination="mobilePagination"
          class="min-h-300px"
        />
        <DeploymentOperateDrawer
          v-model:visible="drawerVisible"
          :operate-type="operateType"
          :row-data="editingData"
          :model-id="modelId"
          @submitted="getData"
        />
      </div>
    </NDrawerContent>
  </NDrawer>
</template>

<style scoped></style>
