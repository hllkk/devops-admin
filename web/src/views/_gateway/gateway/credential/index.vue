<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import { fetchBatchDeleteCredential, fetchGetCredentialList, fetchGetProviderList, fetchResyncCredentials } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable, useTableOperate } from '@/hooks/common/table';
import { $t } from '@/locales';
import ButtonIcon from '@/components/custom/button-icon.vue';
import CredentialSearch from './modules/credential-search.vue';
import CredentialOperateDrawer from './modules/credential-operate-drawer.vue';

defineOptions({
  name: 'CredentialList'
});

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.CredentialSearchParams>({
  pageNum: 1,
  pageSize: 10,
  credentialName: null,
  providerId: null,
  isActive: null,
  litellmSynced: null,
  params: {}
});

const providerMap = ref<Record<string, string>>({});

async function loadProviderMap() {
  const { data } = await fetchGetProviderList({
    pageNum: 1,
    pageSize: 100,
    name: null,
    providerType: null,
    billingType: null,
    isActive: null,
    params: {}
  });
  if (data) {
    providerMap.value = Object.fromEntries(data.rows.map(p => [String(p.providerId), p.name]));
  }
}

onMounted(loadProviderMap);

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } =
  useNaivePaginatedTable({
    api: () => fetchGetCredentialList(searchParams.value),
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
        key: 'credentialName',
        title: $t('page.gateway.credential.col.credentialName'),
        align: 'center',
        minWidth: 140,
        ellipsis: { tooltip: true }
      },
      {
        key: 'provider',
        title: $t('page.gateway.credential.col.provider'),
        align: 'center',
        minWidth: 120,
        render: row => providerMap.value[String(row.providerId)] ?? row.providerId
      },
      {
        key: 'format',
        title: $t('page.gateway.credential.col.format'),
        align: 'center',
        minWidth: 100,
        render: row => <NTag type="info">{row.credentialInfo?.format ?? 'openai'}</NTag>
      },
      {
        key: 'litellmSynced',
        title: $t('page.gateway.credential.col.litellmSynced'),
        align: 'center',
        minWidth: 110,
        render: row => <NTag type={row.litellmSynced ? 'success' : 'warning'}>{$t(row.litellmSynced ? 'page.gateway.common.synced' : 'page.gateway.common.unsynced')}</NTag>
      },
      {
        key: 'isActive',
        title: $t('page.gateway.credential.col.isActive'),
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
              onClick={() => handleEdit(row.credentialId!)}
            />
          );

          const deleteBtn = () => (
            <ButtonIcon
              text
              type="error"
              icon="material-symbols:delete-outline"
              tooltipContent={$t('common.delete')}
              popconfirmContent={$t('common.confirmDelete')}
              onPositiveClick={() => handleDelete(row.credentialId!)}
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
  useTableOperate(data, 'credentialId', getData);

async function handleBatchDelete() {
  const { error } = await fetchBatchDeleteCredential(checkedRowKeys.value);
  if (error) return;
  onBatchDeleted();
}

async function handleDelete(credentialId: CommonType.IdType) {
  const { error } = await fetchBatchDeleteCredential([credentialId]);
  if (error) return;
  onDeleted();
}

async function handleResync() {
  const { data, error } = await fetchResyncCredentials();
  if (error) return;
  window.$message?.success($t('page.gateway.credential.resyncSuccess', { pushed: data?.pushed ?? 0, total: data?.total ?? 0 }));
  getData();
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <CredentialSearch v-model:model="searchParams" @reset="getDataByPage" @search="getDataByPage" />
    <NCard :title="$t('page.gateway.credential.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
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
        >
          <template #prefix>
            <NButton ghost size="small" @click="handleResync">
              <template #icon>
                <icon-material-symbols-sync-rounded class="text-icon" />
              </template>
              {{ $t('page.gateway.credential.resync') }}
            </NButton>
          </template>
        </TableHeaderOperation>
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
        :row-key="row => row.credentialId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
      <CredentialOperateDrawer
        v-model:visible="drawerVisible"
        :operate-type="operateType"
        :row-data="editingData"
        @submitted="getData"
      />
    </NCard>
  </div>
</template>

<style scoped></style>
