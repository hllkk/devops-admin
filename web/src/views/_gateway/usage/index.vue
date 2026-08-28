<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import { fetchGetUserSelect } from '@/service/api/system';
import { fetchGetUsageLogList, fetchReconcileLLMLogs, fetchSyncLLMLogs } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { $t } from '@/locales';

defineOptions({ name: 'GatewayUsage' });

const appStore = useAppStore();

/** 时间范围(ms 时间戳,空=不限)；提交转 RFC3339(后端按 UTC 解析) */
const dateRange = ref<[number, number] | null>(null);

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

/** 用户下拉(懒加载一次,筛选用) */
const userOptions = ref<Array<{ label: string; value: CommonType.IdType }>>([]);

async function loadUserOptions() {
  const { data, error } = await fetchGetUserSelect();
  if (!error) {
    userOptions.value = (data ?? []).map(u => ({ label: u.nickName || u.userName, value: u.userId! }));
  }
}

function fmtRFC3339(ts: number) {
  return new Date(ts).toISOString();
}

/** 应用筛选:时间范围转 RFC3339,其余直传(空值剔除,GET query 不带空串) */
function applySearch() {
  const p = searchParams.value;
  searchParams.value = {
    ...p,
    pageNum: 1,
    startTime: dateRange.value ? fmtRFC3339(dateRange.value[0]) : null,
    endTime: dateRange.value ? fmtRFC3339(dateRange.value[1]) : null
  };
}

function resetSearch() {
  dateRange.value = null;
  searchParams.value = {
    pageNum: 1,
    pageSize: searchParams.value.pageSize,
    userId: null,
    aiKeyId: null,
    deploymentId: null,
    model: null,
    provider: null,
    params: {}
  };
}

const { columns, columnChecks, data, getData, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
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
  loadUserOptions();
  getData();
});
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <NCard :bordered="false" size="small" class="card-wrapper">
      <div class="flex flex-wrap items-center gap-12px">
        <NDatePicker
          v-model:value="dateRange"
          type="datetimerange"
          size="small"
          clearable
          class="w-340px"
        />
        <NSelect
          v-model:value="searchParams.userId"
          :options="userOptions"
          size="small"
          clearable
          filterable
          :placeholder="$t('page.gateway.usage.userPlaceholder')"
          class="w-160px"
        />
        <NInput
          v-model:value="searchParams.model"
          size="small"
          clearable
          :placeholder="$t('page.gateway.usage.modelPlaceholder')"
          class="w-180px"
          @keyup.enter="applySearch"
        />
        <NInput
          v-model:value="searchParams.provider"
          size="small"
          clearable
          :placeholder="$t('page.gateway.usage.providerPlaceholder')"
          class="w-140px"
          @keyup.enter="applySearch"
        />
        <div class="flex-1" />
        <NSpace size="small">
          <NButton size="small" type="primary" @click="applySearch">{{ $t('common.search') }}</NButton>
          <NButton size="small" quaternary @click="resetSearch">{{ $t('common.reset') }}</NButton>
        </NSpace>
      </div>
    </NCard>

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
