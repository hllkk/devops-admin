<script setup lang="tsx">
import { ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import { fetchBatchDeleteProvider, fetchGetProviderList } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import { BILLING_TYPE_OPTIONS } from '@/constants/business/gateway';
import ProviderSearch from './modules/provider-search.vue';
import ProviderOperateDrawer from './modules/provider-operate-drawer.vue';

defineOptions({
  name: 'ProviderList'
});

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.ProviderSearchParams>({
  pageNum: 1,
  pageSize: 10,
  name: null,
  providerType: null,
  billingType: null,
  isActive: null,
  params: {}
});

const billingTypeLabelKey = (v: string) => BILLING_TYPE_OPTIONS.find(o => o.value === v)?.label ?? 'page.gateway.common.billingTypeToken';

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } =
  useNaivePaginatedTable({
    api: () => fetchGetProviderList(searchParams.value),
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
        title: $t('page.gateway.provider.col.name'),
        align: 'center',
        minWidth: 140,
        ellipsis: { tooltip: true }
      },
      {
        key: 'providerType',
        title: $t('page.gateway.provider.col.providerType'),
        align: 'center',
        minWidth: 120
      },
      {
        key: 'billingType',
        title: $t('page.gateway.provider.col.billingType'),
        align: 'center',
        minWidth: 120,
        render: row => <NTag type="info">{$t(billingTypeLabelKey(row.billingType))}</NTag>
      },
      {
        key: 'monthlyBudget',
        title: $t('page.gateway.provider.col.monthlyBudget'),
        align: 'center',
        minWidth: 120,
        render: row => (row.monthlyBudget == null ? <span class="text-slate-400">{$t('page.gateway.common.unlimited')}</span> : `${row.monthlyBudget}`)
      },
      {
        key: 'monthlyUsed',
        title: $t('page.gateway.provider.col.monthlyUsed'),
        align: 'center',
        minWidth: 120,
        render: row => `${row.monthlyUsed ?? 0}`
      },
      {
        key: 'isActive',
        title: $t('page.gateway.provider.col.isActive'),
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
              onClick={() => handleEdit(row.providerId!)}
            />
          );

          const deleteBtn = () => (
            <ButtonIcon
              text
              type="error"
              icon="material-symbols:delete-outline"
              tooltipContent={$t('common.delete')}
              popconfirmContent={$t('common.confirmDelete')}
              onPositiveClick={() => handleDelete(row.providerId!)}
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
  useTableOperate(data, 'providerId', getData);

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteProvider(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

async function handleDelete(providerId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteProvider([providerId]);
  if (error) return;
  onDeleted();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <ProviderSearch v-model:model="searchParams" @reset="getDataByPage" @search="getDataByPage" />
    <NCard :title="$t('page.gateway.provider.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
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
        :row-key="row => row.providerId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
      <ProviderOperateDrawer
        v-model:visible="drawerVisible"
        :operate-type="operateType"
        :row-data="editingData"
        @submitted="getData"
      />
    </NCard>
  </div>
</template>

<style scoped></style>
