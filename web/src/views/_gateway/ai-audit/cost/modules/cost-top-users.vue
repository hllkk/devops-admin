<script setup lang="tsx">
import { ref } from 'vue';
import type { DataTableColumn } from 'naive-ui';
import { $t } from '@/locales';
import { fetchGetCostDetail } from '@/service/api/gateway';

defineOptions({ name: 'CostTopUsers' });

interface Props {
  filters: Api.Gateway.CostSearchParams;
}

const props = defineProps<Props>();

const rows = ref<Api.Gateway.CostDetailRow[]>([]);
const loading = ref(false);

async function load() {
  loading.value = true;
  const { data, error } = await fetchGetCostDetail({ ...props.filters, dimension: 'user', sort: 'internal', pageNum: 1, pageSize: 10 });
  loading.value = false;
  if (error || !data) return;
  rows.value = data.rows ?? [];
}

const columns: DataTableColumn<Api.Gateway.CostDetailRow>[] = [
  {
    key: 'rank',
    title: $t('page.gateway.cost.top.rank'),
    align: 'center',
    width: 60,
    render: row => {
      const idx = rows.value.indexOf(row);
      return idx < 3 ? <span class="font-semibold text-amber-500">{idx + 1}</span> : <span class="text-slate-400">{idx + 1}</span>;
    }
  },
  {
    key: 'label',
    title: $t('page.gateway.cost.detail.dimension.user'),
    minWidth: 120,
    render: row => <span class="font-medium">{row.label}</span>
  },
  {
    key: 'requests',
    title: $t('page.gateway.cost.detail.col.requests'),
    align: 'right',
    minWidth: 90,
    render: row => row.requests.toLocaleString()
  },
  {
    key: 'totalTokens',
    title: $t('page.gateway.cost.detail.col.totalTokens'),
    align: 'right',
    minWidth: 110,
    render: row => row.totalTokens.toLocaleString()
  },
  {
    key: 'internalCost',
    title: $t('page.gateway.cost.detail.col.internalCost'),
    align: 'right',
    minWidth: 110,
    render: row => <span class="font-mono">¥{row.internalCost.toFixed(4)}</span>
  },
  {
    key: 'externalCost',
    title: $t('page.gateway.cost.detail.col.externalCost'),
    align: 'right',
    minWidth: 110,
    render: row => <span class="font-mono">¥{row.externalCost.toFixed(4)}</span>
  }
];

defineExpose({ refresh: load });

load();
</script>

<template>
  <NCard :title="$t('page.gateway.cost.top.title')" :bordered="false" size="small" class="card-wrapper">
    <NDataTable :columns="columns" :data="rows" size="small" :loading="loading" :scroll-x="600" />
  </NCard>
</template>

<style scoped></style>
