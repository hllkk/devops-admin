<script setup lang="tsx">
import { computed } from 'vue';
import { $t } from '@/locales';
import { TOP_DIMENSION_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'DashboardTop' });

type TopSort = 'cost' | 'requests' | 'tokens';

interface Props {
  data: Api.Gateway.TopItem[];
  dimension: 'user' | 'model' | 'aiKey';
  sort: TopSort;
}

const props = defineProps<Props>();

interface Emits {
  (e: 'update:dimension', value: 'user' | 'model' | 'aiKey'): void;
  (e: 'update:sort', value: TopSort): void;
}

const emit = defineEmits<Emits>();

const dimOptions = computed(() => TOP_DIMENSION_OPTIONS.map(o => ({ label: $t(o.label), value: o.value as 'user' | 'model' | 'aiKey' })));

const sortOptions = computed(() => [
  { label: $t('page.gateway.dashboard.metricCost'), value: 'cost' as TopSort },
  { label: $t('page.gateway.dashboard.metricRequests'), value: 'requests' as TopSort },
  { label: $t('page.gateway.dashboard.metricTokens'), value: 'tokens' as TopSort }
]);

const dimValue = computed({
  get: () => props.dimension,
  set: v => emit('update:dimension', v)
});

const sortValue = computed({
  get: () => props.sort,
  set: v => emit('update:sort', v)
});

const columns = computed(() => [
  {
    key: 'index',
    title: $t('common.index'),
    align: 'center',
    width: 64,
    render: (_: Api.Gateway.TopItem, index: number) => index + 1
  },
  {
    key: 'name',
    title: $t('page.gateway.dashboard.topName'),
    align: 'center',
    minWidth: 130,
    ellipsis: { tooltip: true }
  },
  {
    key: 'cost',
    title: $t('page.gateway.dashboard.topCost'),
    align: 'center',
    minWidth: 100,
    render: (row: Api.Gateway.TopItem) => `¥${row.cost.toFixed(4)}`
  },
  {
    key: 'requests',
    title: $t('page.gateway.dashboard.topRequests'),
    align: 'center',
    minWidth: 90,
    render: (row: Api.Gateway.TopItem) => row.requests.toLocaleString()
  },
  {
    key: 'tokens',
    title: $t('page.gateway.dashboard.topTokens'),
    align: 'center',
    minWidth: 110,
    render: (row: Api.Gateway.TopItem) => (row.tokens ?? 0).toLocaleString()
  }
] as any);
</script>

<template>
  <NCard size="small" class="card-wrapper">
    <div class="mb-8px flex flex-wrap items-center justify-between gap-8px">
      <span class="text-14px font-medium">{{ $t('page.gateway.dashboard.topTitle') }}</span>
      <div class="flex flex-wrap items-center gap-8px">
        <NRadioGroup v-model:value="sortValue" size="small">
          <NRadioButton v-for="o in sortOptions" :key="o.value" :value="o.value">{{ o.label }}</NRadioButton>
        </NRadioGroup>
        <NRadioGroup v-model:value="dimValue" size="small">
          <NRadioButton v-for="o in dimOptions" :key="o.value" :value="o.value">{{ o.label }}</NRadioButton>
        </NRadioGroup>
      </div>
    </div>
    <NDataTable :columns="columns" :data="data" size="small" />
  </NCard>
</template>

<style scoped></style>
