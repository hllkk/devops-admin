<script setup lang="ts">
import { ref, watch } from 'vue';
import { fetchGetAssetMetricsTrend } from '@/service/api/server';
import { useEcharts } from '@/hooks/common/echarts';
import { $t } from '@/locales';

defineOptions({ name: 'AssetMetricsChart' });

interface Props {
  assetId: CommonType.IdType | null;
}

const props = defineProps<Props>();

type RangeKey = '1h' | '1d' | '7d' | '30d';
type MetricKey = 'cpu' | 'mem' | 'disk' | 'net';

const range = ref<RangeKey>('1h');
const metric = ref<MetricKey>('cpu');
const loading = ref(false);
const points = ref<Api.Server.MetricPoint[]>([]);

async function load() {
  if (!props.assetId) return;
  loading.value = true;
  const { error, data } = await fetchGetAssetMetricsTrend(props.assetId, range.value);
  loading.value = false;
  points.value = !error && data ? data.points : [];
  render();
}

const { domRef, updateOptions } = useEcharts(() => ({
  tooltip: {
    trigger: 'axis',
    confine: true
  },
  legend: { show: false } as Record<string, any>,
  grid: { top: 24, right: 16, bottom: 30, left: 52 },
  xAxis: {
    type: 'category',
    data: [] as string[],
    axisLabel: { fontSize: 10, color: '#94a3b8' },
    axisLine: { lineStyle: { color: '#e2e8f0' } }
  },
  yAxis: {
    type: 'value',
    axisLabel: { fontSize: 10, color: '#94a3b8' } as Record<string, any>,
    splitLine: { lineStyle: { color: '#f1f5f9' } }
  },
  series: [] as Record<string, any>[]
}));

function fmtTime(ts: string, isHourly: boolean): string {
  const d = new Date(ts);
  const pad = (n: number) => String(n).padStart(2, '0');
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`;
  if (isHourly) {
    return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}h`;
  }
  if (range.value === '1h' || range.value === '1d') {
    return hm;
  }
  return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${hm}`;
}

function render() {
  updateOptions(opts => {
    const isHourly = range.value === '7d' || range.value === '30d';
    opts.xAxis.data = points.value.map(p => fmtTime(p.ts, isHourly));
    opts.series = [];
    opts.yAxis.axisLabel = { fontSize: 10, color: '#94a3b8' };
    if (metric.value === 'net') {
      opts.yAxis.axisLabel.formatter = (v: number) => (v >= 1024 * 1024 ? `${(v / 1024 / 1024).toFixed(0)}MB/s` : `${(v / 1024).toFixed(0)}KB/s`);
      opts.series = [
        { type: 'line', name: $t('page.server.metrics.netIn'), data: points.value.map(p => p.netIn), smooth: true, showSymbol: false, itemStyle: { color: '#6366f1' } },
        { type: 'line', name: $t('page.server.metrics.netOut'), data: points.value.map(p => p.netOut), smooth: true, showSymbol: false, itemStyle: { color: '#10b981' } }
      ];
      opts.legend = { show: true, top: 0, textStyle: { fontSize: 10, color: '#94a3b8' } };
    } else {
      opts.yAxis.axisLabel.formatter = '{value}%';
      const field = metric.value === 'cpu' ? 'cpu' : metric.value === 'mem' ? 'memPct' : 'diskPct';
      const maxField = metric.value === 'cpu' ? 'cpuMax' : metric.value === 'mem' ? 'memPctMax' : 'diskPctMax';
      opts.series = [
        {
          type: 'line',
          name: $t('page.server.metrics.avgLine'),
          data: points.value.map(p => p[field]),
          smooth: true,
          showSymbol: false,
          areaStyle: { opacity: 0.08 },
          itemStyle: { color: '#6366f1' }
        },
        {
          type: 'line',
          name: $t('page.server.metrics.maxLine'),
          data: points.value.map(p => p[maxField]),
          smooth: true,
          showSymbol: false,
          lineStyle: { type: 'dashed', opacity: 0.6 },
          itemStyle: { color: '#f59e0b' }
        }
      ];
      opts.legend = { show: true, top: 0, textStyle: { fontSize: 10, color: '#94a3b8' } };
    }
    return opts;
  });
}

watch(
  () => props.assetId,
  () => load(),
  { immediate: true }
);
watch(range, () => load());
</script>

<template>
  <div class="flex flex-col gap-8px">
    <div class="flex items-center justify-between">
      <span class="text-13px font-500">{{ $t('page.server.metrics.trendTitle') }}</span>
      <NSpace size="small">
        <NRadioGroup v-model:value="metric" size="small" @update:value="render">
          <NRadioButton value="cpu">CPU</NRadioButton>
          <NRadioButton value="mem">{{ $t('page.server.agentOps.memory') }}</NRadioButton>
          <NRadioButton value="disk">{{ $t('page.server.agentOps.disk') }}</NRadioButton>
          <NRadioButton value="net">{{ $t('page.server.agentOps.network') }}</NRadioButton>
        </NRadioGroup>
        <NRadioGroup v-model:value="range" size="small">
          <NRadioButton value="1h">1h</NRadioButton>
          <NRadioButton value="1d">1d</NRadioButton>
          <NRadioButton value="7d">7d</NRadioButton>
          <NRadioButton value="30d">30d</NRadioButton>
        </NRadioGroup>
      </NSpace>
    </div>
    <NSpin :show="loading" size="small">
      <!-- 容器恒渲染(ECharts 需稳定尺寸),无数据时空态覆盖 -->
      <div class="relative h-260px w-full">
        <div ref="domRef" class="h-full w-full" />
        <div v-if="!loading && points.length === 0" class="absolute inset-0 flex-center">
          <NEmpty :description="$t('page.server.metrics.noData')" />
        </div>
      </div>
    </NSpin>
  </div>
</template>

<style scoped></style>
