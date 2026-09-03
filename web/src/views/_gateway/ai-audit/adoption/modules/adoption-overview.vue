<script setup lang="ts">
import { computed, watch } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';
import { $t } from '@/locales';

defineOptions({ name: 'AdoptionOverview' });

interface Props {
  data?: Api.Gateway.AdoptionOverview;
}

const props = defineProps<Props>();

/** KPI 卡片统一结构(可选字段缺省不渲染) */
interface KpiCard {
  key: string;
  label: string;
  value: string;
  sub?: string;
  subTip?: string;
  tip: string;
  change?: number;
  changeSuffix?: string;
}

const cards = computed<KpiCard[]>(() => {
  const kpi = props.data?.kpi;
  return [
    {
      key: 'coverage',
      label: $t('page.gateway.adoption.kpi.coverage'),
      value: `${(kpi?.coverage ?? 0).toFixed(1)}%`,
      sub: `${(kpi?.activeUsers ?? 0).toLocaleString()} / ${(kpi?.totalUsers ?? 0).toLocaleString()}`,
      subTip: $t('page.gateway.adoption.kpi.coverageSubTip'),
      tip: $t('page.gateway.adoption.kpi.coverageTip'),
      change: kpi?.coverageChange ?? 0,
      changeSuffix: 'pp'
    },
    {
      key: 'newActive',
      label: $t('page.gateway.adoption.kpi.newActive'),
      value: (kpi?.newActiveUsers ?? 0).toLocaleString(),
      sub: `${$t('page.gateway.adoption.kpi.prevActive')}: ${(kpi?.prevActiveUsers ?? 0).toLocaleString()}`,
      tip: $t('page.gateway.adoption.kpi.newActiveTip')
    },
    {
      key: 'dailyRequests',
      label: $t('page.gateway.adoption.kpi.dailyRequests'),
      value: (kpi?.dailyRequests ?? 0).toLocaleString(undefined, { maximumFractionDigits: 1 }),
      sub: `${$t('page.gateway.adoption.kpi.totalRequests')}: ${(kpi?.totalRequests ?? 0).toLocaleString()}`,
      subTip: $t('page.gateway.adoption.kpi.totalRequestsTip'),
      tip: $t('page.gateway.adoption.kpi.dailyRequestsTip')
    },
    {
      key: 'perCapitaTokens',
      label: $t('page.gateway.adoption.kpi.perCapitaTokens'),
      value: (kpi?.perCapitaTokens ?? 0).toLocaleString(),
      tip: $t('page.gateway.adoption.kpi.perCapitaTokensTip')
    }
  ];
});

/** 覆盖率环比:百分点差(coverageChange 已是 pp) */
function fmtChange(v: number, suffix: string) {
  if (!v) return '-';
  return `${v > 0 ? '+' : ''}${v.toFixed(1)}${suffix}`;
}

const { domRef, updateOptions } = useEcharts(() => ({
  tooltip: { trigger: 'axis' },
  legend: { top: 0, right: 0, textStyle: { fontSize: 11, color: '#94a3b8' } },
  grid: { top: 32, right: 20, bottom: 30, left: 50 },
  xAxis: {
    type: 'category',
    data: [] as string[],
    axisLabel: { fontSize: 10, color: '#94a3b8' },
    axisLine: { lineStyle: { color: '#e2e8f0' } }
  },
  yAxis: [
    {
      type: 'value',
      minInterval: 1,
      axisLabel: { fontSize: 10, color: '#94a3b8' },
      splitLine: { lineStyle: { color: '#f1f5f9' } }
    },
    {
      type: 'value',
      minInterval: 1,
      axisLabel: { fontSize: 10, color: '#94a3b8' },
      splitLine: { show: false }
    }
  ],
  series: [
    { name: '', type: 'bar', barMaxWidth: 18, data: [] as number[], itemStyle: { borderRadius: [3, 3, 0, 0] } },
    { name: '', type: 'line', smooth: true, showSymbol: false, yAxisIndex: 1, data: [] as number[] }
  ]
}));

function render() {
  updateOptions(opts => {
    const trend = props.data?.trend ?? [];
    opts.xAxis.data = trend.map(i => i.date);
    opts.series[0].name = $t('page.gateway.adoption.trend.active');
    opts.series[1].name = $t('page.gateway.adoption.trend.requests');
    opts.series[0].data = trend.map(i => i.activeUsers);
    opts.series[1].data = trend.map(i => i.requests);
    return opts;
  });
}

watch(
  () => props.data,
  () => render(),
  { immediate: true }
);
</script>

<template>
  <div class="flex-col-stretch gap-12px">
    <div class="grid grid-cols-2 gap-12px xl:grid-cols-4">
      <NCard v-for="card in cards" :key="card.key" size="small" class="card-wrapper">
        <div class="flex flex-col gap-4px">
          <div class="flex items-center gap-4px">
            <span class="text-12px text-slate-400">{{ card.label }}</span>
            <NTooltip trigger="hover">
              <template #trigger>
                <icon-ant-design-question-circle-outlined class="text-12px text-slate-400" />
              </template>
              {{ card.tip }}
            </NTooltip>
          </div>
          <span class="text-20px font-semibold text-slate-900 dark:text-slate-100">{{ card.value }}</span>
          <div class="flex items-center justify-between gap-4px">
            <NTooltip v-if="card.sub && card.subTip" trigger="hover">
              <template #trigger>
                <span class="text-12px text-slate-400">{{ card.sub }}</span>
              </template>
              {{ card.subTip }}
            </NTooltip>
            <span v-else-if="card.sub" class="text-12px text-slate-400">{{ card.sub }}</span>
            <span
              v-if="card.change !== undefined"
              class="text-12px"
              :class="card.change > 0 ? 'text-emerald-500' : card.change < 0 ? 'text-red-500' : 'text-slate-400'"
            >
              {{ $t('page.gateway.adoption.kpi.change') }}: {{ fmtChange(card.change, card.changeSuffix ?? '%') }}
            </span>
          </div>
        </div>
      </NCard>
    </div>
    <NCard size="small" class="card-wrapper">
      <div class="mb-8px text-14px font-medium">{{ $t('page.gateway.adoption.trend.title') }}</div>
      <div ref="domRef" class="h-300px w-full" />
    </NCard>
  </div>
</template>

<style scoped></style>
