<script setup lang="ts">
import { computed, ref, watch } from 'vue';
import { useEcharts } from '@/hooks/common/echarts';
import { $t } from '@/locales';

defineOptions({ name: 'DashboardTrend' });

interface Props {
  data: Api.Gateway.TrendItem[];
}

const props = defineProps<Props>();

const metric = ref<'cost' | 'requests'>('cost');

const { domRef, updateOptions } = useEcharts(() => ({
  tooltip: { trigger: 'axis' },
  grid: { top: 20, right: 20, bottom: 30, left: 50 },
  xAxis: {
    type: 'category',
    data: [] as string[],
    axisLabel: { fontSize: 10, color: '#94a3b8' },
    axisLine: { lineStyle: { color: '#e2e8f0' } }
  },
  yAxis: {
    type: 'value',
    axisLabel: { fontSize: 10, color: '#94a3b8', formatter: '{value}' },
    splitLine: { lineStyle: { color: '#f1f5f9' } }
  },
  series: [
    {
      type: 'bar',
      data: [] as number[],
      itemStyle: { color: '#6366f1', borderRadius: [4, 4, 0, 0] }
    }
  ]
}));

function render() {
  updateOptions(opts => {
    opts.xAxis.data = props.data.map(i => i.date);
    opts.series[0].data = props.data.map(i => (metric.value === 'cost' ? i.cost : i.requests));
    opts.yAxis.axisLabel.formatter = metric.value === 'cost' ? '¥{value}' : '{value}';
    return opts;
  });
}

watch(
  () => props.data,
  () => render(),
  { immediate: true }
);

watch(metric, () => render());

const title = computed(() => $t('page.gateway.dashboard.trendTitle'));
</script>

<template>
  <NCard size="small" class="card-wrapper">
    <div class="mb-8px flex items-center justify-between">
      <span class="text-14px font-medium">{{ title }}</span>
      <NRadioGroup v-model:value="metric" size="small">
        <NRadioButton value="cost">{{ $t('page.gateway.dashboard.metricCost') }}</NRadioButton>
        <NRadioButton value="requests">{{ $t('page.gateway.dashboard.metricRequests') }}</NRadioButton>
      </NRadioGroup>
    </div>
    <div ref="domRef" class="h-300px w-full" />
  </NCard>
</template>

<style scoped></style>
