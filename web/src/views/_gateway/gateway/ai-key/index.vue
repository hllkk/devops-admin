<script setup lang="tsx">
import { ref } from 'vue';
import { NProgress, NTag, NTime } from 'naive-ui';
import { fetchBatchDeleteAiKey, fetchGetAiKeyList } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { $t } from '@/locales';
import { handleCopy } from '@/utils/copy';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { KEY_TYPE_OPTIONS, OWNER_TYPE_OPTIONS } from '@/constants/business/gateway';
import AiKeySearch from './modules/ai-key-search.vue';
import AiKeyOperateDrawer from './modules/ai-key-operate-drawer.vue';

defineOptions({
  name: 'AiKeyList'
});

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.AiKeySearchParams>({
  pageNum: 1,
  pageSize: 10,
  keyType: null,
  ownerType: null,
  ownerId: null,
  name: null,
  isActive: null,
  params: {}
});

const keyTypeLabelKey = (v: string) => KEY_TYPE_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.keyPersonalScene';
const ownerTypeLabelKey = (v: string) => OWNER_TYPE_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.ownerUser';

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => fetchGetAiKeyList(searchParams.value),
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
      title: $t('page.gateway.aiKey.col.name'),
      align: 'center',
      minWidth: 120,
      ellipsis: { tooltip: true }
    },
    {
      key: 'keyType',
      title: $t('page.gateway.aiKey.col.keyType'),
      align: 'center',
      minWidth: 120,
      render: row => <NTag type={row.keyType.endsWith('_main') ? 'success' : 'info'}>{$t(keyTypeLabelKey(row.keyType))}</NTag>
    },
    {
      key: 'owner',
      title: $t('page.gateway.aiKey.col.owner'),
      align: 'center',
      minWidth: 120,
      render: row => `${$t(ownerTypeLabelKey(row.ownerType))}:${row.ownerId}`
    },
    {
      key: 'keyPrefix',
      title: $t('page.gateway.aiKey.col.keyPrefix'),
      align: 'center',
      minWidth: 180,
      render: row => (
        <span class="cursor-copy font-mono" onClick={() => handleCopy(row.keyPrefix)}>
          {row.keyPrefix}
        </span>
      )
    },
    {
      key: 'models',
      title: $t('page.gateway.aiKey.col.models'),
      align: 'center',
      minWidth: 80,
      render: row => `${(row.models ?? []).length}`
    },
    {
      key: 'budget',
      title: $t('page.gateway.aiKey.col.budget'),
      align: 'center',
      minWidth: 160,
      render: row => {
        if (row.budgetLimit == null) {
          return <span class="text-slate-400">{$t('page.gateway.common.unlimited')}</span>;
        }
        const rate = row.budgetLimit > 0 ? Math.min((row.budgetUsed / row.budgetLimit) * 100, 100) : 0;
        return (
          <div class="flex flex-col items-center gap-2px">
            <NProgress type="line" percentage={rate} status={rate >= 85 ? 'error' : rate >= 60 ? 'warning' : 'success'} />
            <span class="text-12px text-slate-400">{row.budgetUsed}/{row.budgetLimit}</span>
          </div>
        );
      }
    },
    {
      key: 'isActive',
      title: $t('page.gateway.aiKey.col.isActive'),
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
      width: 160,
      render: row => {
        const editBtn = () => (
          <ButtonIcon
            text
            type="primary"
            icon="material-symbols:drive-file-rename-outline-outline"
            tooltipContent={$t('common.edit')}
            onClick={() => handleEdit(row.aiKeyId!)}
          />
        );

        const deleteBtn = () => (
          <ButtonIcon
            text
            type="error"
            icon="material-symbols:delete-outline"
            tooltipContent={$t('common.delete')}
            popconfirmContent={$t('common.confirmDelete')}
            onPositiveClick={() => handleDelete(row.aiKeyId!)}
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

const { drawerVisible, operateType, editingData, handleAdd, handleEdit, checkedRowKeys, onBatchDeleted, onDeleted } =
  useTableOperate(data, 'aiKeyId', getData);

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteAiKey(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

async function handleDelete(aiKeyId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteAiKey([aiKeyId]);
  if (error) return;
  onDeleted();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <AiKeySearch v-model:model="searchParams" @reset="getDataByPage" @search="getDataByPage" />
    <NCard :title="$t('page.gateway.aiKey.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
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
        :row-key="row => row.aiKeyId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
      <AiKeyOperateDrawer
        v-model:visible="drawerVisible"
        :operate-type="operateType"
        :row-data="editingData"
        @submitted="getData"
      />
    </NCard>
  </div>
</template>

<style scoped></style>
