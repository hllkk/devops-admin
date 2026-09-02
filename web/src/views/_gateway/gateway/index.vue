<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import {
  fetchAggregateUsage,
  fetchGetBalanceSummary,
  fetchGetDashboardBudget,
  fetchGetDashboardOverview,
  fetchGetDashboardTop,
  fetchGetDashboardTrend
} from '@/service/api/gateway';
import { $t } from '@/locales';
import { DASHBOARD_RANGE_OPTIONS } from '@/constants/business/gateway';
import DashboardOverview from './modules/dashboard-overview.vue';
import DashboardTrend from './modules/dashboard-trend.vue';
import DashboardTop from './modules/dashboard-top.vue';
import DashboardBudget from './modules/dashboard-budget.vue';
import DashboardBalance from './modules/dashboard-balance.vue';

defineOptions({ name: 'GatewayDashboard' });

type RangeKey = 'today' | '7d' | 'thisMonth' | '30d' | 'lastMonth' | 'custom';

const range = ref<RangeKey>('7d');
const scope = ref<'all' | 'self'>('all');
const dateRange = ref<[number, number] | null>(null);

const rangeOptions = computed(() => [
  ...DASHBOARD_RANGE_OPTIONS.map(o => ({ label: $t(o.label), value: o.value as RangeKey })),
  { label: $t('page.gateway.dashboard.customRange'), value: 'custom' as RangeKey }
]);

function fmt(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

const queryParams = computed(() => {
  if (range.value === 'custom' && dateRange.value) {
    return {
      startDate: fmt(new Date(dateRange.value[0])),
      endDate: fmt(new Date(dateRange.value[1])),
      scope: scope.value
    };
  }
  const now = new Date();
  const end = new Date(now);
  let start = new Date(now);
  switch (range.value) {
    case 'today':
      break;
    case '7d':
      start.setDate(start.getDate() - 6);
      break;
    case 'thisMonth':
      start = new Date(now.getFullYear(), now.getMonth(), 1);
      break;
    case '30d':
      start.setDate(start.getDate() - 29);
      break;
    case 'lastMonth': {
      start = new Date(now.getFullYear(), now.getMonth() - 1, 1);
      end.setTime(new Date(now.getFullYear(), now.getMonth(), 0).getTime());
      break;
    }
    default:
      break;
  }
  return { startDate: fmt(start), endDate: fmt(end), scope: scope.value };
});

const overview = ref<Api.Gateway.DashboardOverview>();
const trend = ref<Api.Gateway.TrendItem[]>([]);
const top = ref<Api.Gateway.TopItem[]>([]);
const budget = ref<Api.Gateway.BudgetItem[]>([]);
const balanceSummary = ref<Api.Gateway.ProviderBalanceSummary[]>([]);
const topDimension = ref<'user' | 'model' | 'aiKey'>('user');
const topSort = ref<'cost' | 'requests' | 'tokens'>('cost');
const loading = ref(false);

async function loadTop() {
  const { data, error } = await fetchGetDashboardTop({ ...queryParams.value, dimension: topDimension.value, sort: topSort.value });
  if (!error) top.value = data ?? [];
}

async function loadAll() {
  loading.value = true;
  try {
    const [ov, tr, bg, bal] = await Promise.all([
      fetchGetDashboardOverview(queryParams.value),
      fetchGetDashboardTrend(queryParams.value),
      fetchGetDashboardBudget({ scope: scope.value }),
      fetchGetBalanceSummary()
    ]);
    if (!ov.error) overview.value = ov.data;
    if (!tr.error) trend.value = tr.data ?? [];
    if (!bg.error) budget.value = bg.data ?? [];
    if (!bal.error) balanceSummary.value = bal.data ?? [];
    await loadTop();
  } finally {
    loading.value = false;
  }
}

async function handleAggregate() {
  const { data, error } = await fetchAggregateUsage();
  if (error) return;
  window.$message?.success(
    $t('page.gateway.dashboard.aggregateSuccess', {
      synced: data?.synced ?? 0,
      rebuilt: data?.rebuilt ?? 0,
      disabled: data?.keysDisabled ?? 0
    })
  );
  await loadAll();
}

watch(queryParams, () => loadAll());
watch(topDimension, () => loadTop());
watch(topSort, () => loadTop());

onMounted(loadAll);
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <NCard :bordered="false" size="small" class="card-wrapper">
      <div class="flex flex-wrap items-center gap-12px">
        <NSelect v-model:value="range" :options="rangeOptions" size="small" class="w-140px" />
        <NDatePicker
          v-if="range === 'custom'"
          v-model:value="dateRange"
          type="daterange"
          size="small"
          clearable
          class="w-280px"
        />
        <NDivider vertical />
        <NRadioGroup v-model:value="scope" size="small">
          <NRadioButton value="all">{{ $t('page.gateway.dashboard.scopeAll') }}</NRadioButton>
          <NRadioButton value="self">{{ $t('page.gateway.dashboard.scopeSelf') }}</NRadioButton>
        </NRadioGroup>
        <div class="flex-1" />
        <NButton type="primary" ghost size="small" :loading="loading" @click="handleAggregate">
          <template #icon>
            <icon-material-symbols-bolt-rounded class="text-icon" />
          </template>
          {{ $t('page.gateway.dashboard.aggregate') }}
        </NButton>
      </div>
    </NCard>

    <NSpin :show="loading">
      <div class="flex-col-stretch gap-16px">
        <DashboardOverview :data="overview" />
        <div class="grid grid-cols-1 gap-16px lg:grid-cols-2">
          <DashboardTrend :data="trend" />
          <DashboardTop v-model:dimension="topDimension" v-model:sort="topSort" :data="top" />
        </div>
        <DashboardBalance :data="balanceSummary" />
        <DashboardBudget :data="budget" />
      </div>
    </NSpin>
  </div>
</template>

<style scoped></style>
