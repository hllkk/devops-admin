<script setup lang="tsx">
import { computed, onMounted, ref } from 'vue';
import { NTag, NTime } from 'naive-ui';
import { fetchGetMcpLogList, fetchGetMCPServerList, fetchReconcileMcpLogs, fetchSyncMcpLogs } from '@/service/api/gateway';
import { useAppStore } from '@/store/modules/app';
import { defaultTransform, useNaivePaginatedTable } from '@/hooks/common/table';
import { $t } from '@/locales';
import UserSelect from '@/components/custom/user-select.vue';

defineOptions({ name: 'McpLogPanel' });

interface Props {
  /** 外部跳转预填的时间范围(ms 时间戳,成本分析 MCP 维「日志」跳转) */
  initialDateRange?: [number, number] | null;
  /** 外部跳转预填的 MCP 服务器筛选 */
  initialServerId?: CommonType.IdType | null;
}

const props = defineProps<Props>();

const appStore = useAppStore();

const searchParams = ref<Api.Gateway.McpLogSearchParams>({
  pageNum: 1,
  pageSize: 20,
  userId: null,
  aiKeyId: null,
  mcpServerId: null,
  toolName: null,
  status: null,
  params: {}
});

/** 时间范围(ms 时间戳,空=不限)；提交转 RFC3339(后端按 UTC 解析) */
const dateRange = ref<[number, number] | null>(null);

onMounted(() => {
  if (props.initialDateRange) dateRange.value = [...props.initialDateRange];
  if (props.initialServerId) searchParams.value.mcpServerId = props.initialServerId;
  loadServerOptions();
});

/** MCP 服务器下拉(管理端全量,不分页拉前 100) */
const serverOptions = ref<CommonType.Option<CommonType.IdType>[]>([]);

async function loadServerOptions() {
  const { data, error } = await fetchGetMCPServerList({ pageNum: 1, pageSize: 100 });
  if (!error && data) {
    serverOptions.value = (data.rows ?? []).map(row => ({ label: `${row.name} (${row.serverName})`, value: row.mcpServerId }));
  }
}

const statusOptions = computed(() => [
  { label: $t('page.gateway.mcpLog.statusSuccess'), value: 'success' },
  { label: $t('page.gateway.mcpLog.statusError'), value: 'error' }
]);

const { columns, columnChecks, data, getData, getDataByPage, loading, mobilePagination, scrollX } = useNaivePaginatedTable({
  api: () => {
    const { pageNum, pageSize, userId, mcpServerId, toolName, status } = searchParams.value;
    const params: Api.Gateway.McpLogSearchParams = { pageNum, pageSize };
    if (userId) params.userId = userId;
    if (mcpServerId) params.mcpServerId = mcpServerId;
    if (toolName) params.toolName = toolName;
    if (status) params.status = status;
    if (dateRange.value) {
      params.startTime = new Date(dateRange.value[0]).toISOString();
      params.endTime = new Date(dateRange.value[1]).toISOString();
    }
    return fetchGetMcpLogList(params);
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
      key: 'serverName',
      title: $t('page.gateway.mcpLog.server'),
      align: 'center',
      minWidth: 150,
      render: row => <span class="font-mono text-13px">{row.serverName}</span>
    },
    {
      key: 'toolName',
      title: $t('page.gateway.mcpLog.tool'),
      align: 'center',
      minWidth: 180,
      render: row => <span class="font-mono text-13px">{row.toolName || row.namespacedName}</span>
    },
    {
      key: 'status',
      title: $t('page.gateway.mcpLog.status'),
      align: 'center',
      minWidth: 90,
      render: row =>
        row.status === 'error' ? (
          <NTag size="small" type="error">{$t('page.gateway.mcpLog.statusError')}</NTag>
        ) : (
          <NTag size="small" type="success">{$t('page.gateway.mcpLog.statusSuccess')}</NTag>
        )
    },
    {
      key: 'durationMs',
      title: $t('page.gateway.usage.col.duration'),
      align: 'right',
      minWidth: 90,
      render: row => (row.durationMs > 0 ? `${(row.durationMs / 1000).toFixed(2)}s` : '-')
    },
    {
      key: 'externalCost',
      title: $t('page.gateway.mcpLog.externalCost'),
      align: 'right',
      minWidth: 110,
      render: row => <span class="font-mono">¥{row.externalCost.toFixed(4)}</span>
    },
    {
      key: 'internalCost',
      title: $t('page.gateway.mcpLog.internalCost'),
      align: 'right',
      minWidth: 110,
      render: row => <span class="font-mono">¥{row.internalCost.toFixed(4)}</span>
    }
  ]
});

const syncing = ref(false);

async function handleSync() {
  syncing.value = true;
  const { error } = await fetchSyncMcpLogs();
  syncing.value = false;
  if (error) return;
  window.$message?.success($t('page.gateway.usage.syncSuccess'));
  getData();
}

const reconciling = ref(false);

async function handleReconcile() {
  reconciling.value = true;
  const { error } = await fetchReconcileMcpLogs();
  reconciling.value = false;
  if (error) return;
  window.$message?.success($t('page.gateway.usage.reconcileSuccess'));
  getData();
}

function reset() {
  dateRange.value = null;
  searchParams.value.userId = null;
  searchParams.value.mcpServerId = null;
  searchParams.value.toolName = null;
  searchParams.value.status = null;
  getDataByPage();
}

defineExpose({ getData });

onMounted(() => {
  getData();
});
</script>

<template>
  <NCard :title="$t('page.gateway.mcpLog.title')" :bordered="false" size="small" class="card-wrapper sm:flex-1-hidden">
    <template #header-extra>
      <NSpace size="small">
        <NPopconfirm @positive-click="handleSync">
          <template #trigger>
            <NButton size="small" :loading="syncing">{{ $t('page.gateway.usage.syncNow') }}</NButton>
          </template>
          {{ $t('page.gateway.mcpLog.syncConfirm') }}
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
    <div class="mb-12px flex flex-wrap items-center gap-12px">
      <NDatePicker
        v-model:value="dateRange"
        type="datetimerange"
        clearable
        :default-time="['00:00:00', '23:59:59']"
        class="w-300px"
        @update:value="() => getDataByPage()"
      />
      <UserSelect
        v-model:value="searchParams.userId"
        clearable
        filterable
        class="w-160px"
        :placeholder="$t('page.gateway.usage.userPlaceholder')"
        @update:value="() => getDataByPage()"
      />
      <NSelect
        v-model:value="searchParams.mcpServerId"
        clearable
        filterable
        :options="serverOptions"
        class="w-200px"
        :placeholder="$t('page.gateway.mcpLog.serverPlaceholder')"
        @update:value="() => getDataByPage()"
      />
      <NInput
        v-model:value="searchParams.toolName"
        clearable
        class="w-160px"
        :placeholder="$t('page.gateway.mcpLog.toolPlaceholder')"
        @keyup.enter="() => getDataByPage()"
      />
      <NSelect
        v-model:value="searchParams.status"
        clearable
        :options="statusOptions"
        class="w-120px"
        :placeholder="$t('page.gateway.mcpLog.status')"
        @update:value="() => getDataByPage()"
      />
      <NButton size="small" @click="reset">
        <template #icon>
          <icon-ic-round-refresh class="text-icon" />
        </template>
        {{ $t('common.reset') }}
      </NButton>
    </div>
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
</template>

<style scoped></style>
