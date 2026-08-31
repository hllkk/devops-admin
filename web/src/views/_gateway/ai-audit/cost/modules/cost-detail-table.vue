<script setup lang="tsx">
import { computed, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';
import { NButton, NDataTable } from 'naive-ui';
import type { DataTableColumn, DataTableRowKey } from 'naive-ui';
import { $t } from '@/locales';
import { fetchGetCostDetail, fetchGetCostScopeUsers } from '@/service/api/gateway';

defineOptions({ name: 'CostDetailTable' });

interface Props {
  /** 筛选条件(时间/部门/用户/模型/供应商,不含维度与分页) */
  filters: Api.Gateway.CostSearchParams;
}

const props = defineProps<Props>();

const router = useRouter();

type Dimension = 'department' | 'user' | 'model' | 'aiKey' | 'provider' | 'date';
type SortKey = 'internal' | 'external' | 'requests' | 'tokens';

const dimension = ref<Dimension>('department');
const sort = ref<SortKey>('internal');

const dimensionOptions = computed(() =>
  (['department', 'user', 'model', 'aiKey', 'provider', 'date'] as Dimension[]).map(d => ({
    key: d,
    label: $t(`page.gateway.cost.detail.dimension.${d}`)
  }))
);

const sortOptions = computed(() =>
  (['internal', 'external', 'requests', 'tokens'] as SortKey[]).map(s => ({
    label: $t(`page.gateway.cost.detail.sort.${s}`),
    value: s
  }))
);

/** label 列标题随维度 */
const labelTitle = computed(() => $t(`page.gateway.cost.detail.dimension.${dimension.value}`));

const pageNum = ref(1);
const pageSize = ref(20);
const total = ref(0);
const rows = ref<Api.Gateway.CostDetailRow[]>([]);
const loading = ref(false);

async function load() {
  loading.value = true;
  const params: Api.Gateway.CostSearchParams = {
    ...props.filters,
    dimension: dimension.value,
    sort: sort.value,
    pageNum: pageNum.value,
    pageSize: pageSize.value
  };
  const { data, error } = await fetchGetCostDetail(params);
  loading.value = false;
  if (error || !data) return;
  rows.value = data.rows ?? [];
  total.value = data.total ?? 0;
}

/** 部门下钻(直挂口径):按行懒加载缓存 */
const scopeCache = reactive(new Map<string, Api.Gateway.CostScopeUserRow[]>());
const scopeLoading = reactive(new Set<string>());

watch(dimension, () => {
  pageNum.value = 1;
  scopeCache.clear();
  load();
});

watch(sort, () => {
  pageNum.value = 1;
  load();
});

const pagination = computed(() => ({
  page: pageNum.value,
  pageSize: pageSize.value,
  itemCount: total.value,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
  onChange: (page: number) => {
    pageNum.value = page;
    load();
  },
  onUpdatePageSize: (size: number) => {
    pageSize.value = size;
    pageNum.value = 1;
    load();
  }
}));

/** 行键(维度原始值) */
function rowKey(row: Api.Gateway.CostDetailRow) {
  return row.value;
}

async function loadScope(deptValue: string) {
  if (scopeCache.has(deptValue) || scopeLoading.has(deptValue)) return;
  scopeLoading.add(deptValue);
  const { data, error } = await fetchGetCostScopeUsers(deptValue, props.filters);
  scopeLoading.delete(deptValue);
  if (error || !data) return;
  scopeCache.set(deptValue, data);
}

function handleExpand(keys: DataTableRowKey[]) {
  if (dimension.value !== 'department') return;
  for (const k of keys) {
    const v = String(k);
    if (!scopeCache.has(v)) loadScope(v);
  }
}

const scopeColumns: DataTableColumn<Api.Gateway.CostScopeUserRow>[] = [
  { key: 'userName', title: $t('page.gateway.cost.search.user'), minWidth: 140 },
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

/** 行「日志」→ 调用日志页(带维度参数与当前筛选,query 预填) */
function goLogs(row: Api.Gateway.CostDetailRow) {
  const f = props.filters;
  const query: Record<string, string> = {};
  if (f.startDate) query.startDate = f.startDate;
  if (f.endDate) query.endDate = f.endDate;
  if (f.userId) query.userId = String(f.userId);
  if (f.aiKeyId) query.aiKeyId = String(f.aiKeyId);
  if (f.model) query.model = f.model;
  if (f.provider) query.provider = f.provider;
  // 维度行收敛到单值筛选(date 维把时间收敛为单日)
  switch (dimension.value) {
    case 'user':
      if (row.value !== '0') query.userId = row.value;
      break;
    case 'aiKey':
      if (row.value !== '0') query.aiKeyId = row.value;
      break;
    case 'model':
      if (row.value) query.model = row.value;
      break;
    case 'provider':
      if (row.value) query.provider = row.value;
      break;
    case 'date':
      query.startDate = row.value;
      query.endDate = row.value;
      break;
    default:
      break;
  }
  router.push({ path: '/gateway/ai-audit/usage', query });
}

const columns = computed<DataTableColumn<Api.Gateway.CostDetailRow>[]>(() => {
  const cols: DataTableColumn<Api.Gateway.CostDetailRow>[] = [];
  if (dimension.value === 'department') {
    cols.push({
      type: 'expand',
      renderExpand: row => (
        <NDataTable
          size="small"
          columns={scopeColumns}
          data={scopeCache.get(row.value) ?? []}
          loading={scopeLoading.has(row.value)}
          class="pl-24px"
        />
      )
    });
  }
  cols.push(
    {
      key: 'label',
      title: labelTitle.value,
      minWidth: 150,
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
      key: 'promptTokens',
      title: $t('page.gateway.cost.detail.col.promptTokens'),
      align: 'right',
      minWidth: 100,
      render: row => row.promptTokens.toLocaleString()
    },
    {
      key: 'completionTokens',
      title: $t('page.gateway.cost.detail.col.completionTokens'),
      align: 'right',
      minWidth: 100,
      render: row => row.completionTokens.toLocaleString()
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
      sorter: false,
      render: row => <span class="font-mono">¥{row.internalCost.toFixed(4)}</span>
    },
    {
      key: 'externalCost',
      title: $t('page.gateway.cost.detail.col.externalCost'),
      align: 'right',
      minWidth: 110,
      render: row => <span class="font-mono">¥{row.externalCost.toFixed(4)}</span>
    },
    {
      key: 'costDiff',
      title: $t('page.gateway.cost.detail.col.costDiff'),
      align: 'right',
      minWidth: 110,
      render: row => <span class="font-mono text-slate-400">¥{row.costDiff.toFixed(4)}</span>
    },
    {
      key: 'activeUsers',
      title: $t('page.gateway.cost.detail.col.activeUsers'),
      align: 'right',
      minWidth: 90,
      render: row => row.activeUsers.toLocaleString()
    },
    {
      key: 'perCapita',
      title: $t('page.gateway.cost.detail.col.perCapita'),
      align: 'right',
      minWidth: 110,
      render: row => <span class="font-mono">¥{row.perCapita.toFixed(2)}</span>
    },
    {
      key: 'operations',
      title: $t('page.gateway.cost.detail.col.operations'),
      align: 'center',
      width: 80,
      render: row => (
        <NButton size="tiny" quaternary type="primary" onClick={() => goLogs(row)}>
          {$t('page.gateway.cost.toLogs')}
        </NButton>
      )
    }
  );
  return cols;
});

defineExpose({ refresh: load });

async function init() {
  pageNum.value = 1;
  scopeCache.clear();
  await load();
}

init();
</script>

<template>
  <NCard :title="$t('page.gateway.cost.detail.title')" :bordered="false" size="small" class="card-wrapper">
    <template #header-extra>
      <NRadioGroup v-model:value="sort" size="small">
        <NRadioButton v-for="opt in sortOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</NRadioButton>
      </NRadioGroup>
    </template>
    <NTabs v-model:value="dimension" type="line" size="small" animated>
      <NTabPane v-for="opt in dimensionOptions" :key="opt.key" :name="opt.key" :tab="opt.label" />
    </NTabs>
    <NDataTable
      :columns="columns"
      :data="rows"
      size="small"
      :loading="loading"
      remote
      :row-key="rowKey"
      :pagination="pagination"
      :scroll-x="1250"
      :on-update:expanded-row-keys="handleExpand"
      class="mt-8px"
    />
  </NCard>
</template>

<style scoped></style>
