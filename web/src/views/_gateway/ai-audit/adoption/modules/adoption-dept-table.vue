<script setup lang="tsx">
import { reactive, ref } from 'vue';
import { NDataTable, NProgress, NTag } from 'naive-ui';
import type { DataTableColumn, DataTableRowKey } from 'naive-ui';
import { $t } from '@/locales';
import { fetchGetAdoptionDepartments, fetchGetAdoptionDeptUsers } from '@/service/api/gateway';

defineOptions({ name: 'AdoptionDeptTable' });

interface Props {
  /** 筛选条件(时间/部门,不含分页) */
  filters: Api.Gateway.AdoptionSearchParams;
}

const props = defineProps<Props>();

const rows = ref<Api.Gateway.AdoptionDeptRow[]>([]);
const loading = ref(false);

async function load() {
  loading.value = true;
  const { data, error } = await fetchGetAdoptionDepartments(props.filters);
  loading.value = false;
  if (error || !data) return;
  rows.value = data;
}

/** 成员下钻(含未激活,兼未使用人员清单):按部门懒加载缓存 */
const userCache = reactive(new Map<string, Api.Gateway.AdoptionUserRow[]>());
const userLoading = reactive(new Set<string>());

async function loadUsers(deptId: string) {
  if (userCache.has(deptId) || userLoading.has(deptId)) return;
  userLoading.add(deptId);
  const { data, error } = await fetchGetAdoptionDeptUsers(deptId, props.filters);
  userLoading.delete(deptId);
  if (error || !data) return;
  userCache.set(deptId, data);
}

function handleExpand(keys: DataTableRowKey[]) {
  for (const k of keys) {
    const v = String(k);
    if (!userCache.has(v)) loadUsers(v);
  }
}

function rowKey(row: Api.Gateway.AdoptionDeptRow) {
  return String(row.deptId);
}

const userColumns: DataTableColumn<Api.Gateway.AdoptionUserRow>[] = [
  { key: 'userName', title: $t('page.gateway.adoption.dept.user.name'), minWidth: 140 },
  {
    key: 'active',
    title: $t('page.gateway.adoption.dept.user.status'),
    width: 90,
    render: row =>
      row.active ? (
        <NTag size="small" type="success">
          {$t('page.gateway.adoption.dept.user.activeYes')}
        </NTag>
      ) : (
        <NTag size="small" type="default">
          {$t('page.gateway.adoption.dept.user.activeNo')}
        </NTag>
      )
  },
  {
    key: 'requests',
    title: $t('page.gateway.adoption.dept.col.requests'),
    align: 'right',
    width: 100,
    render: row => row.requests.toLocaleString()
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
    width: 110,
    render: row => <span class="font-mono">¥{row.internalCost.toFixed(4)}</span>
  },
  {
    key: 'lastActiveAt',
    title: $t('page.gateway.adoption.dept.user.lastActive'),
    width: 150,
    render: row => <span class="text-slate-400">{row.lastActiveAt || '-'}</span>
  }
];

const columns: DataTableColumn<Api.Gateway.AdoptionDeptRow>[] = [
  {
    type: 'expand',
    renderExpand: row => (
      <NDataTable
        size="small"
        columns={userColumns}
        data={userCache.get(String(row.deptId)) ?? []}
        loading={userLoading.has(String(row.deptId))}
        class="pl-24px"
      />
    )
  },
  {
    key: 'deptName',
    title: $t('page.gateway.adoption.dept.col.name'),
    minWidth: 160,
    render: row => <span class="font-medium">{row.deptName}</span>
  },
  {
    key: 'memberCount',
    title: $t('page.gateway.adoption.dept.col.member'),
    align: 'right',
    width: 90,
    render: row => row.memberCount.toLocaleString()
  },
  {
    key: 'activeCount',
    title: $t('page.gateway.adoption.dept.col.active'),
    align: 'right',
    width: 110,
    render: row => (
      <span>
        <span class="font-medium">{row.activeCount.toLocaleString()}</span>
        <span class="text-slate-400"> / {row.memberCount.toLocaleString()}</span>
      </span>
    )
  },
  {
    key: 'coverage',
    title: $t('page.gateway.adoption.dept.col.coverage'),
    width: 170,
    render: row => (
      <div class="flex items-center gap-8px pr-8px">
        <NProgress
          type="line"
          percentage={Number(row.coverage.toFixed(1))}
          show-indicator={false}
          height={6}
          class="flex-1"
          color={row.coverage >= 50 ? '#18a058' : row.coverage > 0 ? '#f0a020' : '#d1d5db'}
        />
        <span class="w-48px shrink-0 text-right font-mono text-12px">{row.coverage.toFixed(1)}%</span>
      </div>
    )
  },
  {
    key: 'requests',
    title: $t('page.gateway.adoption.dept.col.requests'),
    align: 'right',
    width: 100,
    render: row => row.requests.toLocaleString()
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
  }
];

defineExpose({ refresh: load });

async function init() {
  userCache.clear();
  await load();
}

init();
</script>

<template>
  <NCard :title="$t('page.gateway.adoption.dept.title')" :bordered="false" size="small" class="card-wrapper">
    <template #header-extra>
      <span class="text-12px text-slate-400">{{ $t('page.gateway.adoption.dept.expandTip') }}</span>
    </template>
    <NDataTable
      :columns="columns"
      :data="rows"
      size="small"
      :loading="loading"
      :row-key="rowKey"
      :max-height="480"
      :scroll-x="1000"
      :on-update:expanded-row-keys="handleExpand"
    />
  </NCard>
</template>

<style scoped></style>
