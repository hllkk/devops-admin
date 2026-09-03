<script setup lang="tsx">
import { computed, onMounted, ref } from 'vue';
import { NButton, NDataTable, NTag, NTooltip } from 'naive-ui';
import type { DataTableColumn } from 'naive-ui';
import { $t } from '@/locales';
import { fetchGetHealthSummary, fetchHealthCheckDeployments } from '@/service/api/gateway';

defineOptions({ name: 'GatewayHealth' });

const summary = ref<Api.Gateway.HealthSummary>();
const loading = ref(false);
const checking = ref(false);

async function load() {
  loading.value = true;
  const { data, error } = await fetchGetHealthSummary();
  loading.value = false;
  if (error || !data) return;
  summary.value = data;
}

/** 手动巡检模型部署(路由组级 ping,完成后刷新汇总) */
async function checkNow() {
  checking.value = true;
  const { data, error } = await fetchHealthCheckDeployments();
  checking.value = false;
  if (error) return;
  window.$message?.success($t('page.gateway.health.checked', { n: data ?? 0 }));
  load();
}

/** 状态 → 标签类型 */
const statusType = (status: string) => {
  switch (status) {
    case 'healthy':
      return 'success';
    case 'unhealthy':
      return 'error';
    case 'warning':
      return 'warning';
    case 'danger':
      return 'error';
    default:
      return 'default';
  }
};

const statusLabel = (status: string) => {
  switch (status) {
    case 'healthy':
      return $t('page.gateway.health.status.healthy');
    case 'unhealthy':
      return $t('page.gateway.health.status.unhealthy');
    case 'warning':
      return $t('page.gateway.health.freshness.warning');
    case 'danger':
      return $t('page.gateway.health.freshness.danger');
    default:
      return $t('page.gateway.health.status.unknown');
  }
};

/** 四状态卡数据 */
interface SummaryCard {
  key: string;
  label: string;
  healthy: number;
  unhealthy: number;
  unknown: number;
  total: number;
  sub: string;
}

const cards = computed<SummaryCard[]>(() => {
  const s = summary.value;
  const comp = s?.components ?? [];
  const compHealthy = comp.filter(i => i.status === 'healthy').length;
  const f = s?.freshness;
  return [
    {
      key: 'mcp',
      label: $t('page.gateway.health.card.mcp'),
      healthy: s?.mcp.healthy ?? 0,
      unhealthy: s?.mcp.unhealthy ?? 0,
      unknown: s?.mcp.unknown ?? 0,
      total: s?.mcp.total ?? 0,
      sub: `${$t('page.gateway.health.card.total', { n: s?.mcp.total ?? 0 })}`
    },
    {
      key: 'deployment',
      label: $t('page.gateway.health.card.deployment'),
      healthy: s?.deployment.healthy ?? 0,
      unhealthy: s?.deployment.unhealthy ?? 0,
      unknown: s?.deployment.unknown ?? 0,
      total: s?.deployment.total ?? 0,
      sub: `${$t('page.gateway.health.card.total', { n: s?.deployment.total ?? 0 })}`
    },
    {
      key: 'components',
      label: $t('page.gateway.health.card.components'),
      healthy: compHealthy,
      unhealthy: comp.length - compHealthy,
      unknown: 0,
      total: comp.length,
      sub: comp.filter(i => i.status !== 'healthy').map(i => i.name).join(', ') || $t('page.gateway.health.status.healthy')
    },
    {
      key: 'freshness',
      label: $t('page.gateway.health.card.freshness'),
      healthy: f?.status === 'healthy' ? 1 : 0,
      unhealthy: f?.status === 'danger' ? 1 : 0,
      unknown: f?.status === 'unknown' ? 1 : 0,
      total: 1,
      sub: f?.lastSyncAt || $t('page.gateway.health.freshness.unknown')
    }
  ];
});

const mcpColumns: DataTableColumn<Api.Gateway.HealthMcpItem>[] = [
  { key: 'name', title: $t('page.gateway.health.col.name'), minWidth: 160, render: row => <span class="font-medium">{row.name}</span> },
  { key: 'serverName', title: $t('page.gateway.health.col.serverName'), minWidth: 140 },
  {
    key: 'healthStatus',
    title: $t('page.gateway.health.col.status'),
    width: 100,
    render: row => (
      <NTag size="small" type={statusType(row.healthStatus) as never}>
        {statusLabel(row.healthStatus)}
      </NTag>
    )
  },
  {
    key: 'lastHealthCheck',
    title: $t('page.gateway.health.col.lastCheck'),
    width: 170,
    render: row => <span class="text-slate-400">{row.lastHealthCheck || '-'}</span>
  },
  {
    key: 'healthCheckError',
    title: $t('page.gateway.health.col.error'),
    minWidth: 260,
    ellipsis: { tooltip: true },
    render: row => (row.healthCheckError ? <span class="text-red-500">{row.healthCheckError}</span> : '-')
  }
];

const depColumns: DataTableColumn<Api.Gateway.HealthDeploymentItem>[] = [
  { key: 'modelName', title: $t('page.gateway.health.col.model'), minWidth: 160, render: row => <span class="font-medium">{row.modelName}</span> },
  { key: 'deployName', title: $t('page.gateway.health.col.deployName'), minWidth: 140 },
  { key: 'modelKey', title: $t('page.gateway.health.col.modelKey'), minWidth: 160, render: row => <span class="font-mono text-12px">{row.modelKey}</span> },
  {
    key: 'healthStatus',
    title: $t('page.gateway.health.col.status'),
    width: 100,
    render: row => (
      <NTag size="small" type={statusType(row.healthStatus) as never}>
        {statusLabel(row.healthStatus)}
      </NTag>
    )
  },
  {
    key: 'lastHealthCheck',
    title: $t('page.gateway.health.col.lastCheck'),
    width: 170,
    render: row => <span class="text-slate-400">{row.lastHealthCheck || '-'}</span>
  },
  {
    key: 'healthCheckError',
    title: $t('page.gateway.health.col.error'),
    minWidth: 260,
    ellipsis: { tooltip: true },
    render: row => (row.healthCheckError ? <span class="text-red-500">{row.healthCheckError}</span> : '-')
  }
];

const componentName = (name: string) => {
  switch (name) {
    case 'litellm':
      return 'LiteLLM';
    case 'postgresql':
      return 'PostgreSQL';
    case 'redis':
      return 'Redis';
    default:
      return name;
  }
};

const componentColumns: DataTableColumn<Api.Gateway.HealthComponentItem>[] = [
  { key: 'name', title: $t('page.gateway.health.col.component'), minWidth: 140, render: row => <span class="font-medium">{componentName(row.name)}</span> },
  {
    key: 'status',
    title: $t('page.gateway.health.col.status'),
    width: 100,
    render: row => (
      <NTag size="small" type={statusType(row.status) as never}>
        {statusLabel(row.status)}
      </NTag>
    )
  },
  {
    key: 'latencyMs',
    title: $t('page.gateway.health.col.latency'),
    width: 120,
    render: row => `${row.latencyMs} ms`
  },
  {
    key: 'message',
    title: $t('page.gateway.health.col.message'),
    minWidth: 260,
    ellipsis: { tooltip: true },
    render: row => row.message || '-'
  }
];

const freshnessRows = computed(() => {
  const f = summary.value?.freshness;
  return [
    {
      key: 'llm',
      label: $t('page.gateway.health.freshness.llm'),
      value: f?.llmSyncAt || '-'
    },
    {
      key: 'mcp',
      label: $t('page.gateway.health.freshness.mcp'),
      value: f?.mcpSyncAt || '-'
    },
    {
      key: 'last',
      label: $t('page.gateway.health.freshness.lastSync'),
      value: f?.lastSyncAt || $t('page.gateway.health.freshness.unknown')
    }
  ];
});

function mcpRowKey(row: Api.Gateway.HealthMcpItem) {
  return String(row.mcpServerId);
}

function depRowKey(row: Api.Gateway.HealthDeploymentItem) {
  return String(row.deploymentId);
}

function compRowKey(row: Api.Gateway.HealthComponentItem) {
  return row.name;
}

onMounted(load);
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <NCard :bordered="false" size="small" class="card-wrapper">
      <template #header-extra>
        <NSpace align="center">
          <span class="text-12px text-slate-400">
            {{ $t('page.gateway.health.checkedAt') }}: {{ summary?.checkedAt || '-' }}
          </span>
          <NButton size="small" type="primary" ghost :loading="checking" @click="checkNow">
            <template #icon>
              <icon-ant-design-heart-outlined class="text-icon" />
            </template>
            {{ $t('page.gateway.health.checkNow') }}
          </NButton>
          <NButton size="small" quaternary :loading="loading" @click="load">
            <template #icon>
              <icon-ic-round-refresh class="text-icon" />
            </template>
            {{ $t('common.refresh') }}
          </NButton>
        </NSpace>
      </template>
      <div class="grid grid-cols-2 gap-12px xl:grid-cols-4">
        <NCard v-for="card in cards" :key="card.key" size="small" :bordered="true">
          <div class="flex flex-col gap-4px">
            <div class="flex items-center gap-4px">
              <span class="text-12px text-slate-400">{{ card.label }}</span>
              <NTooltip v-if="card.key === 'freshness'" trigger="hover">
                <template #trigger>
                  <icon-ant-design-question-circle-outlined class="text-12px text-slate-400" />
                </template>
                {{ $t('page.gateway.health.freshness.tip') }}
              </NTooltip>
            </div>
            <span class="text-20px font-semibold" :class="card.unhealthy > 0 ? 'text-red-500' : 'text-slate-900 dark:text-slate-100'">
              {{ card.healthy }} / {{ card.total }}
            </span>
            <div class="flex items-center gap-8px text-12px">
              <span v-if="card.healthy" class="text-emerald-500">{{ $t('page.gateway.health.status.healthy') }} {{ card.healthy }}</span>
              <span v-if="card.unhealthy" class="text-red-500">{{ $t('page.gateway.health.status.unhealthy') }} {{ card.unhealthy }}</span>
              <span v-if="card.unknown" class="text-slate-400">{{ $t('page.gateway.health.status.unknown') }} {{ card.unknown }}</span>
            </div>
            <span class="text-12px text-slate-400">{{ card.sub }}</span>
          </div>
        </NCard>
      </div>
    </NCard>

    <NCard :bordered="false" size="small" class="card-wrapper">
      <NTabs type="line" size="small" animated>
        <NTabPane name="mcp" :tab="$t('page.gateway.health.tab.mcp')">
          <NDataTable
            :columns="mcpColumns"
            :data="summary?.mcpItems ?? []"
            size="small"
            :loading="loading"
            :row-key="mcpRowKey"
            :max-height="420"
            :scroll-x="850"
          />
        </NTabPane>
        <NTabPane name="deployment" :tab="$t('page.gateway.health.tab.deployment')">
          <NDataTable
            :columns="depColumns"
            :data="summary?.deploymentItems ?? []"
            size="small"
            :loading="loading"
            :row-key="depRowKey"
            :max-height="420"
            :scroll-x="950"
          />
        </NTabPane>
        <NTabPane name="component" :tab="$t('page.gateway.health.tab.component')">
          <div class="flex flex-col gap-12px">
            <NDataTable
              :columns="componentColumns"
              :data="summary?.components ?? []"
              size="small"
              :loading="loading"
              :row-key="compRowKey"
            />
            <div class="grid grid-cols-1 gap-12px sm:grid-cols-3">
              <NCard v-for="row in freshnessRows" :key="row.key" size="small" :bordered="true">
                <div class="flex flex-col gap-4px">
                  <span class="text-12px text-slate-400">{{ row.label }}</span>
                  <span class="font-mono text-14px">{{ row.value }}</span>
                </div>
              </NCard>
            </div>
          </div>
        </NTabPane>
      </NTabs>
    </NCard>
  </div>
</template>

<style scoped></style>
