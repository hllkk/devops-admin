<script setup lang="ts">
import { computed, watch } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';
import { $t } from '@/locales';

defineOptions({ name: 'CostOverview' });

interface Props {
  data?: Api.Gateway.CostOverview;
}

const props = defineProps<Props>();

/** 环比文案：上期无消耗(0)显示'-'，否则 ±x.x% */
function fmtChange(v: number) {
  if (!v) return '-';
  return `${v > 0 ? '+' : ''}${v.toFixed(1)}%`;
}

const cards = computed(() => {
  const kpi = props.data?.kpi;
  return [
    {
      key: 'internal',
      label: $t('page.gateway.cost.kpi.internal'),
      value: `¥${(kpi?.internalCost ?? 0).toFixed(2)}`,
      change: kpi?.internalChange ?? 0,
      tip: $t('page.gateway.cost.kpi.changeTip')
    },
    {
      key: 'external',
      label: $t('page.gateway.cost.kpi.external'),
      value: `¥${(kpi?.externalCost ?? 0).toFixed(2)}`,
      change: kpi?.externalChange ?? 0,
      tip: $t('page.gateway.cost.kpi.changeTip')
    },
    {
      key: 'diff',
      label: $t('page.gateway.cost.kpi.diff'),
      value: `¥${(kpi?.costDiff ?? 0).toFixed(2)}`,
      tip: $t('page.gateway.cost.kpi.diffTip')
    },
    {
      key: 'dailyAvg',
      label: $t('page.gateway.cost.kpi.dailyAvg'),
      value: `¥${(kpi?.dailyAvgInternal ?? 0).toFixed(2)}`,
      tip: $t('page.gateway.cost.kpi.dailyAvgTip')
    }
  ] as const;
});

const { domRef, updateOptions } = useEcharts(() => ({
  tooltip: { trigger: 'axis' },
  legend: { top: 0, right: 0, textStyle: { fontSize: 11, color: '#94a3b8' } },
  grid: { top: 32, right: 20, bottom: 30, left: 60 },
  xAxis: {
    type: 'category',
    data: [] as string[],
    axisLabel: { fontSize: 10, color: '#94a3b8' },
    axisLine: { lineStyle: { color: '#e2e8f0' } }
  },
  yAxis: {
    type: 'value',
    axisLabel: { fontSize: 10, color: '#94a3b8', formatter: '¥{value}' },
    splitLine: { lineStyle: { color: '#f1f5f9' } }
  },
  series: [
    { name: '', type: 'line', smooth: true, showSymbol: false, data: [] as number[], areaStyle: { opacity: 0.08 } },
    { name: '', type: 'line', smooth: true, showSymbol: false, data: [] as number[], lineStyle: { type: 'dashed' } }
  ]
}));

function render() {
  updateOptions(opts => {
    const trend = props.data?.trend ?? [];
    opts.xAxis.data = trend.map(i => i.date);
    opts.series[0].name = $t('page.gateway.cost.trend.internal');
    opts.series[1].name = $t('page.gateway.cost.trend.external');
    opts.series[0].data = trend.map(i => Number(i.internalCost.toFixed(4)));
    opts.series[1].data = trend.map(i => Number(i.externalCost.toFixed(4)));
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
          <span
            v-if="card.key === 'internal' || card.key === 'external'"
            class="text-12px"
            :class="(card.change ?? 0) > 0 ? 'text-red-500' : (card.change ?? 0) < 0 ? 'text-emerald-500' : 'text-slate-400'"
          >
            {{ $t('page.gateway.cost.kpi.change') }}: {{ fmtChange(card.change) }}
          </span>
        </div>
      </NCard>
    </div>
    <NCard size="small" class="card-wrapper">
      <div class="mb-8px text-14px font-medium">{{ $t('page.gateway.cost.trend.title') }}</div>
      <div ref="domRef" class="h-300px w-full" />
    </NCard>
  </div>
</template>

<style scoped></style>
