<script setup lang="tsx">
import { ref } from 'vue';
import { NDataTable, NProgress } from 'naive-ui';
import type { DataTableColumn } from 'naive-ui';
import { $t } from '@/locales';
import { fetchGetAdoptionModels } from '@/service/api/gateway';

defineOptions({ name: 'AdoptionModels' });

interface Props {
  /** 筛选条件(时间/部门) */
  filters: Api.Gateway.AdoptionSearchParams;
}

const props = defineProps<Props>();

const rows = ref<Api.Gateway.AdoptionModelRow[]>([]);
const loading = ref(false);

async function load() {
  loading.value = true;
  const { data, error } = await fetchGetAdoptionModels(props.filters);
  loading.value = false;
  if (error || !data) return;
  rows.value = data;
}

function rowKey(row: Api.Gateway.AdoptionModelRow) {
  return row.model;
}

const columns: DataTableColumn<Api.Gateway.AdoptionModelRow>[] = [
  {
    key: 'model',
    title: $t('page.gateway.adoption.model.col.model'),
    minWidth: 200,
    render: row => <span class="font-medium">{row.model}</span>
  },
  {
    key: 'requests',
    title: $t('page.gateway.adoption.dept.col.requests'),
    align: 'right',
    width: 110,
    render: row => row.requests.toLocaleString()
  },
  {
    key: 'requestShare',
    title: $t('page.gateway.adoption.model.col.requestShare'),
    width: 170,
    render: row => (
      <div class="flex items-center gap-8px pr-8px">
        <NProgress
          type="line"
          percentage={Number(row.requestShare.toFixed(1))}
          show-indicator={false}
          height={6}
          class="flex-1"
        />
        <span class="w-48px shrink-0 text-right font-mono text-12px">{row.requestShare.toFixed(1)}%</span>
      </div>
    )
  },
  {
    key: 'totalTokens',
    title: $t('page.gateway.adoption.dept.col.totalTokens'),
    align: 'right',
    width: 120,
    render: row => row.totalTokens.toLocaleString()
  },
  {
    key: 'internalCost',
    title: $t('page.gateway.adoption.dept.col.internalCost'),
    align: 'right',
    width: 120,
    render: row => <span class="font-mono">¥{row.internalCost.toFixed(4)}</span>
  },
  {
    key: 'costShare',
    title: $t('page.gateway.adoption.model.col.costShare'),
    align: 'right',
    width: 110,
    render: row => <span class="font-mono text-slate-400">{row.costShare.toFixed(1)}%</span>
  },
  {
    key: 'activeUsers',
    title: $t('page.gateway.adoption.model.col.activeUsers'),
    align: 'right',
    width: 100,
    render: row => row.activeUsers.toLocaleString()
  }
];

defineExpose({ refresh: load });

load();
</script>

<template>
  <NCard :title="$t('page.gateway.adoption.model.title')" :bordered="false" size="small" class="card-wrapper">
    <template #header-extra>
      <span class="text-12px text-slate-400">{{ $t('page.gateway.adoption.model.tip') }}</span>
    </template>
    <NDataTable
      :columns="columns"
      :data="rows"
      size="small"
      :loading="loading"
      :row-key="rowKey"
      :max-height="420"
      :scroll-x="950"
    />
  </NCard>
</template>

<style scoped></style>
