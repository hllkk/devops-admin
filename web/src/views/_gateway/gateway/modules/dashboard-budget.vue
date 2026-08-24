<script setup lang="tsx">
import { computed } from 'vue';
import { NProgress, NTag } from 'naive-ui';
import { $t } from '@/locales';

defineOptions({ name: 'DashboardBudget' });

interface Props {
  data: Api.Gateway.BudgetItem[];
}

defineProps<Props>();

const columns = computed(() => [
  {
    key: 'index',
    title: $t('common.index'),
    align: 'center',
    width: 64,
    render: (_: Api.Gateway.BudgetItem, index: number) => index + 1
  },
  {
    key: 'name',
    title: $t('page.gateway.dashboard.budgetName'),
    align: 'center',
    minWidth: 140,
    ellipsis: { tooltip: true }
  },
  {
    key: 'ownerName',
    title: $t('page.gateway.dashboard.budgetOwner'),
    align: 'center',
    minWidth: 120
  },
  {
    key: 'budgetLimit',
    title: $t('page.gateway.dashboard.budgetLimit'),
    align: 'center',
    minWidth: 120,
    render: (row: Api.Gateway.BudgetItem) => (row.budgetLimit > 0 ? `¥${row.budgetLimit}` : $t('page.gateway.common.unlimited'))
  },
  {
    key: 'budgetUsed',
    title: $t('page.gateway.dashboard.budgetUsed'),
    align: 'center',
    minWidth: 120,
    render: (row: Api.Gateway.BudgetItem) => `¥${row.budgetUsed}`
  },
  {
    key: 'usageRate',
    title: $t('page.gateway.dashboard.usageRate'),
    align: 'center',
    minWidth: 180,
    render: (row: Api.Gateway.BudgetItem) => (
      <NProgress
        type="line"
        percentage={Math.min(row.usageRate, 100)}
        status={row.usageRate >= 85 ? 'error' : row.usageRate >= 60 ? 'warning' : 'success'}
      />
    )
  },
  {
    key: 'hardLimit',
    title: $t('page.gateway.dashboard.hardLimit'),
    align: 'center',
    minWidth: 100,
    render: (row: Api.Gateway.BudgetItem) => (
      <NTag type={row.hardLimit ? 'error' : 'default'}>{$t(row.hardLimit ? 'page.gateway.common.hardLimitOn' : 'page.gateway.common.hardLimitOff')}</NTag>
    )
  },
  {
    key: 'isActive',
    title: $t('page.gateway.dashboard.isActive'),
    align: 'center',
    minWidth: 100,
    render: (row: Api.Gateway.BudgetItem) => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}</NTag>
  }
] as any);
</script>

<template>
  <NCard :title="$t('page.gateway.dashboard.budgetTitle')" size="small" :bordered="false" class="card-wrapper">
    <NDataTable :columns="columns" :data="data" size="small" />
  </NCard>
</template>

<style scoped></style>
