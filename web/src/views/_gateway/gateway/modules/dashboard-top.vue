<script setup lang="tsx">
import { computed } from 'vue';
import { $t } from '@/locales';
import { TOP_DIMENSION_OPTIONS } from '@/constants/business/gateway';

defineOptions({ name: 'DashboardTop' });

interface Props {
  data: Api.Gateway.TopItem[];
  dimension: 'user' | 'model' | 'aiKey';
}

const props = defineProps<Props>();

interface Emits {
  (e: 'update:dimension', value: 'user' | 'model' | 'aiKey'): void;
}

const emit = defineEmits<Emits>();

const dimOptions = computed(() => TOP_DIMENSION_OPTIONS.map(o => ({ label: $t(o.label), value: o.value as 'user' | 'model' | 'aiKey' })));

const dimValue = computed({
  get: () => props.dimension,
  set: v => emit('update:dimension', v)
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
    minWidth: 140,
    ellipsis: { tooltip: true }
  },
  {
    key: 'cost',
    title: $t('page.gateway.dashboard.topCost'),
    align: 'center',
    minWidth: 120,
    render: (row: Api.Gateway.TopItem) => `¥${row.cost}`
  },
  {
    key: 'requests',
    title: $t('page.gateway.dashboard.topRequests'),
    align: 'center',
    minWidth: 120
  }
] as any);
</script>

<template>
  <NCard size="small" class="card-wrapper">
    <div class="mb-8px flex items-center justify-between">
      <span class="text-14px font-medium">{{ $t('page.gateway.dashboard.topTitle') }}</span>
      <NRadioGroup v-model:value="dimValue" size="small">
        <NRadioButton v-for="o in dimOptions" :key="o.value" :value="o.value">{{ o.label }}</NRadioButton>
      </NRadioGroup>
    </div>
    <NDataTable :columns="columns" :data="data" size="small" />
  </NCard>
</template>

<style scoped></style>
