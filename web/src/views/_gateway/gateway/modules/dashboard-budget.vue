<script setup lang="tsx">
import { onMounted, ref } from 'vue';
import { NDataTable, NProgress, NTag } from 'naive-ui';
import type { DataTableColumn } from 'naive-ui';
import { $t } from '@/locales';
import { fetchGetBudgetSummary } from '@/service/api/gateway';
import BudgetRuleDrawer from './budget-rule-drawer.vue';

defineOptions({ name: 'DashboardBudget' });

const loading = ref(false);
const keyItems = ref<Api.Gateway.BudgetItem[]>([]);
const deptRules = ref<Api.Gateway.BudgetRuleView[]>([]);
const userRules = ref<Api.Gateway.BudgetRuleView[]>([]);
const activeTab = ref<'key' | 'dept' | 'user'>('key');

async function load() {
  loading.value = true;
  const { data, error } = await fetchGetBudgetSummary();
  loading.value = false;
  if (error || !data) return;
  keyItems.value = data.keys ?? [];
  deptRules.value = data.depts ?? [];
  userRules.value = data.users ?? [];
}

onMounted(load);

// Key 级预算列(复用现有)
const keyColumns: DataTableColumn<Api.Gateway.BudgetItem>[] = [
  { key: 'index', title: $t('common.index'), align: 'center', width: 64, render: (_: Api.Gateway.BudgetItem, index: number) => index + 1 },
  { key: 'name', title: $t('page.gateway.dashboard.budgetName'), align: 'center', minWidth: 140, ellipsis: { tooltip: true } },
  { key: 'ownerName', title: $t('page.gateway.dashboard.budgetOwner'), align: 'center', minWidth: 120 },
  {
    key: 'budgetLimit', title: $t('page.gateway.dashboard.budgetLimit'), align: 'center', minWidth: 120,
    render: (row: Api.Gateway.BudgetItem) => row.budgetLimit > 0 ? `¥${row.budgetLimit}` : $t('page.gateway.common.unlimited')
  },
  { key: 'budgetUsed', title: $t('page.gateway.dashboard.budgetUsed'), align: 'center', minWidth: 120, render: (row: Api.Gateway.BudgetItem) => `¥${row.budgetUsed}` },
  {
    key: 'usageRate', title: $t('page.gateway.dashboard.usageRate'), align: 'center', minWidth: 180,
    render: (row: Api.Gateway.BudgetItem) => (
      <NProgress type="line" percentage={Math.min(row.usageRate, 100)} status={row.usageRate >= 85 ? 'error' : row.usageRate >= 60 ? 'warning' : 'success'} />
    )
  },
  {
    key: 'hardLimit', title: $t('page.gateway.dashboard.hardLimit'), align: 'center', minWidth: 100,
    render: (row: Api.Gateway.BudgetItem) => (
      <NTag type={row.hardLimit ? 'error' : 'default'}>{$t(row.hardLimit ? 'page.gateway.common.hardLimitOn' : 'page.gateway.common.hardLimitOff')}</NTag>
    )
  },
  {
    key: 'isActive', title: $t('page.gateway.dashboard.isActive'), align: 'center', minWidth: 100,
    render: (row: Api.Gateway.BudgetItem) => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.common.active' : 'page.gateway.common.inactive')}</NTag>
  }
];

// 部门/用户级预算规则列
const ruleColumns: DataTableColumn<Api.Gateway.BudgetRuleView>[] = [
  { key: 'index', title: $t('common.index'), align: 'center', width: 64, render: (_: Api.Gateway.BudgetRuleView, index: number) => index + 1 },
  { key: 'scopeName', title: $t('page.gateway.budget.scopeName'), align: 'center', minWidth: 140, ellipsis: { tooltip: true } },
  {
    key: 'budgetLimit', title: $t('page.gateway.budget.budgetLimit'), align: 'center', minWidth: 120,
    render: (row: Api.Gateway.BudgetRuleView) => row.budgetLimit > 0 ? `¥${row.budgetLimit}` : $t('page.gateway.common.unlimited')
  },
  { key: 'budgetUsed', title: $t('page.gateway.budget.budgetUsed'), align: 'center', minWidth: 120, render: (row: Api.Gateway.BudgetRuleView) => `¥${row.budgetUsed.toFixed(2)}` },
  {
    key: 'budgetUsedPercent', title: $t('page.gateway.dashboard.usageRate'), align: 'center', minWidth: 180,
    render: (row: Api.Gateway.BudgetRuleView) => (
      <NProgress type="line" percentage={Math.min(row.budgetUsedPercent, 100)} status={row.budgetUsedPercent >= 85 ? 'error' : row.budgetUsedPercent >= 60 ? 'warning' : 'success'} />
    )
  },
  { key: 'budgetDuration', title: $t('page.gateway.budget.duration'), align: 'center', minWidth: 90 },
  {
    key: 'softWarnPercent', title: $t('page.gateway.budget.softWarnPercent'), align: 'center', minWidth: 100,
    render: (row: Api.Gateway.BudgetRuleView) => `${row.softWarnPercent}%`
  },
  {
    key: 'hardLimit', title: $t('page.gateway.budget.hardLimit'), align: 'center', minWidth: 100,
    render: (row: Api.Gateway.BudgetRuleView) => (
      <NTag type={row.budgetHardLimit ? 'error' : 'default'}>{$t(row.budgetHardLimit ? 'page.gateway.common.hardLimitOn' : 'page.gateway.common.hardLimitOff')}</NTag>
    )
  },
  {
    key: 'alertStatus', title: $t('page.gateway.budget.alertStatus'), align: 'center', minWidth: 100,
    render: (row: Api.Gateway.BudgetRuleView) =>
      row.isHardLimited ? <NTag type="error">{$t('page.gateway.budget.hardLimited')}</NTag> :
      row.isSoftWarn ? <NTag type="warning">{$t('page.gateway.budget.softWarned')}</NTag> :
      <NTag type="success">{$t('page.gateway.budget.normal')}</NTag>
  },
  {
    key: 'isActive', title: $t('page.gateway.budget.isActive'), align: 'center', minWidth: 100,
    render: (row: Api.Gateway.BudgetRuleView) => <NTag type={row.isActive ? 'success' : 'default'}>{$t(row.isActive ? 'page.gateway.budget.isActive' : 'page.gateway.budget.isActive')}</NTag>
  }
];

// 预算规则配置抽屉
const drawerVisible = ref(false);
const drawerRow = ref<Api.Gateway.BudgetRuleView | null>(null);
const drawerScope = ref<'dept' | 'user'>('dept');

function handleAdd(scope: 'dept' | 'user') {
  drawerScope.value = scope;
  drawerRow.value = null;
  drawerVisible.value = true;
}

// 编辑/删除操作(行操作按钮后续按需加)
</script>

<template>
  <NCard :title="$t('page.gateway.dashboard.budgetTitle')" size="small" :bordered="false" class="card-wrapper">
    <NTabs v-model:value="activeTab" type="line" size="small" animated>
      <NTabPane name="key" :tab="$t('page.gateway.budget.tabKey')">
        <NDataTable :columns="keyColumns" :data="keyItems" size="small" :loading="loading" />
      </NTabPane>
      <NTabPane name="dept" :tab="$t('page.gateway.budget.tabDept')">
        <template #tab>
          <div class="flex items-center gap-6px">
            <span>{{ $t('page.gateway.budget.tabDept') }}</span>
            <NButton size="tiny" quaternary type="primary" @click.stop="handleAdd('dept')">+{{ $t('page.gateway.budget.add') }}</NButton>
          </div>
        </template>
        <NDataTable :columns="ruleColumns" :data="deptRules" size="small" :loading="loading" />
      </NTabPane>
      <NTabPane name="user" :tab="$t('page.gateway.budget.tabUser')">
        <template #tab>
          <div class="flex items-center gap-6px">
            <span>{{ $t('page.gateway.budget.tabUser') }}</span>
            <NButton size="tiny" quaternary type="primary" @click.stop="handleAdd('user')">+{{ $t('page.gateway.budget.add') }}</NButton>
          </div>
        </template>
        <NDataTable :columns="ruleColumns" :data="userRules" size="small" :loading="loading" />
      </NTabPane>
    </NTabs>
    <BudgetRuleDrawer v-model:visible="drawerVisible" :row-data="drawerRow" :scope-type="drawerScope" @submitted="load" />
  </NCard>
</template>

<style scoped></style>
