<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import { fetchGetUsageLogList, fetchReconcileLLMLogs, fetchSyncLLMLogs } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { $t } from '@/locales';
import UsageSearch from './modules/usage-search.vue';

defineOptions({ name: 'GatewayUsage' });

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.UsageLogSearchParams>({
  pageNum: 1,
  pageSize: 20,
  userId: null,
  aiKeyId: null,
  deploymentId: null,
  model: null,
  provider: null,
  params: {}
});

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => {
    const { pageNum, pageSize, userId, model, provider } = searchParams.value;
    const params: Api.Gateway.UsageLogSearchParams = { pageNum, pageSize };
    if (userId) params.userId = userId;
    if (model) params.model = model;
    if (provider) params.provider = provider;
    if (searchParams.value.startTime) params.startTime = searchParams.value.startTime;
    if (searchParams.value.endTime) params.endTime = searchParams.value.endTime;
    return fetchGetUsageLogList(params);
  },
  transform: response => defaultTransform(response),
  onPaginationParamsChange: params => {
    searchParams.value.pageNum = params.page;
    searchParams.value.pageSize = params.pageSize;
  },
  columns: () => [
    {
      key: 'startedAt',
      title: $t('page.gateway.usage.col.time'),
      align: 'center',
      minWidth: 165,
      render: row => <NTime time={new Date(row.startedAt)} type="datetime" />
    },
    {
      key: 'userName',
      title: $t('page.gateway.usage.col.user'),
      align: 'center',
      minWidth: 110,
      render: row => row.userName || <span class="text-slate-400">{$t('page.gateway.usage.unattributed')}</span>
    },
    {
      key: 'aiKeyName',
      title: $t('page.gateway.usage.col.aiKey'),
      align: 'center',
      minWidth: 120,
      render: row => row.aiKeyName || <span class="text-slate-400">{$t('page.gateway.usage.unattributed')}</span>
    },
    {
      key: 'model',
      title: $t('page.gateway.usage.col.model'),
      align: 'center',
      minWidth: 170,
      render: row => <span class="font-mono text-13px">{row.model}</span>
    },
    {
      key: 'deploymentName',
      title: $t('page.gateway.usage.col.deployment'),
      align: 'center',
      minWidth: 130,
      render: row => row.deploymentName || <span class="text-slate-400">-</span>
    },
    {
      key: 'callType',
      title: $t('page.gateway.usage.col.callType'),
      align: 'center',
      minWidth: 95,
      render: row => (row.callType ? <NTag size="small">{row.callType}</NTag> : <span class="text-slate-400">-</span>)
    },
    {
      key: 'promptTokens',
      title: $t('page.gateway.usage.col.promptTokens'),
      align: 'right',
      minWidth: 100,
      render: row => row.promptTokens?.toLocaleString() ?? '0'
    },
    {
      key: 'completionTokens',
      title: $t('page.gateway.usage.col.completionTokens'),
      align: 'right',
      minWidth: 100,
      render: row => row.completionTokens?.toLocaleString() ?? '0'
    },
    {
      key: 'externalCost',
      title: $t('page.gateway.usage.col.cost'),
      align: 'right',
      minWidth: 100,
      render: row => <span class="font-mono">¥{row.externalCost.toFixed(6)}</span>
    },
    {
      key: 'durationMs',
      title: $t('page.gateway.usage.col.duration'),
      align: 'right',
      minWidth: 90,
      render: row => (row.durationMs > 0 ? `${(row.durationMs / 1000).toFixed(2)}s` : '-')
    }
  ]
});

/** 手动回流/对账(兜底工具；定时任务 5 分钟/1 小时自动跑) */
const syncing = ref(false);

async function handleSync() {
  syncing.value = true;
  const { error } = await fetchSyncLLMLogs();
  syncing.value = false;
  if (error) return;
  window.$message?.success($t('page.gateway.usage.syncSuccess'));
  getData();
}

const reconciling = ref(false);

async function handleReconcile() {
  reconciling.value = true;
  const { error } = await fetchReconcileLLMLogs();
  reconciling.value = false;
  if (error) return;
  window.$message?.success($t('page.gateway.usage.reconcileSuccess'));
  getData();
}

onMounted(() => {
  getData();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <UsageSearch v-model:model="searchParams" @search="getDataByPage" />

    <NCard :title="$t('page.gateway.usage.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
      <template #header-extra>
        <NSpace size="small">
          <NPopconfirm @positive-click="handleSync">
            <template #trigger>
              <NButton size="small" :loading="syncing">{{ $t('page.gateway.usage.syncNow') }}</NButton>
            </template>
            {{ $t('page.gateway.usage.syncConfirm') }}
          </NPopconfirm>
          <NPopconfirm @positive-click="handleReconcile">
            <template #trigger>
              <NButton size="small" :loading="reconciling">{{ $t('page.gateway.usage.reconcileNow') }}</NButton>
            </template>
            {{ $t('page.gateway.usage.reconcileConfirm') }}
          </NPopconfirm>
          <TableHeaderOperation
            v-model:columns="columnChecks"
            :loading="loading"
            :show-add="false"
            :show-delete="false"
            :show-refresh="true"
            @refresh="getData"
          />
        </NSpace>
      </template>
      <NDataTable
        :columns="columns"
        :data="data"
        size="small"
        :flex-height="!appStore.isMobile"
        :scroll-x="scrollX"
        :loading="loading"
        remote
        :row-key="row => row.logId"
        :pagination="mobilePagination"
        class="sm:h-full"
      />
    </NCard>
  </div>
</template>

<style scoped></style>
