<script setup lang="tsx">
import { computed, onMounted, reactive, ref } from 'vue';
import { NButton, NDataTable, NTag } from 'naive-ui';
import type { DataTableColumn } from 'naive-ui';
import { useClipboard } from '@vueuse/core';
import { $t } from '@/locales';
import { useDownload } from '@/hooks/business/download';
import { fetchGetReport, fetchGetReportList } from '@/service/api/gateway';
import ReportGenerateModal from './modules/report-generate-modal.vue';

defineOptions({ name: 'GatewayReport' });

const { download } = useDownload();
const { copy } = useClipboard();

/** 类型筛选('all'=全部，提交时剔除) */
const typeFilter = ref<'all' | 'weekly' | 'monthly' | 'custom'>('all');
const pageNum = ref(1);
const pageSize = ref(10);
const total = ref(0);
const rows = ref<Api.Gateway.EfficiencyReportView[]>([]);
const loading = ref(false);

const detail = ref<Api.Gateway.EfficiencyReportView>();
const detailLoading = ref(false);
const generateVisible = ref(false);

const typeLabel = (t: 'weekly' | 'monthly' | 'custom') => $t(`page.gateway.report.type.${t}`);

async function load() {
  loading.value = true;
  const { data, error } = await fetchGetReportList({
    reportType: typeFilter.value === 'all' ? null : typeFilter.value,
    pageNum: pageNum.value,
    pageSize: pageSize.value
  });
  loading.value = false;
  if (error || !data) return;
  rows.value = data.rows ?? [];
  total.value = data.total ?? 0;
}

async function loadDetail(id: CommonType.IdType) {
  detailLoading.value = true;
  const { data, error } = await fetchGetReport(id);
  detailLoading.value = false;
  if (error || !data) return;
  detail.value = data;
}

const pagination = reactive({
  page: pageNum,
  pageSize,
  itemCount: total,
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
});

const listColumns: DataTableColumn<Api.Gateway.EfficiencyReportView>[] = [
  {
    key: 'reportType',
    title: $t('page.gateway.report.col.type'),
    width: 90,
    render: row => (
      <NTag size="small" type={row.reportType === 'weekly' ? 'info' : row.reportType === 'monthly' ? 'success' : 'default'}>
        {typeLabel(row.reportType)}
      </NTag>
    )
  },
  {
    key: 'period',
    title: $t('page.gateway.report.col.period'),
    minWidth: 200,
    render: row => (
      <span class="font-mono text-13px">
        {row.periodStart} ~ {row.periodEnd}
      </span>
    )
  },
  {
    key: 'summary',
    title: $t('page.gateway.report.col.summary'),
    minWidth: 260,
    ellipsis: { tooltip: true }
  },
  {
    key: 'creatorName',
    title: $t('page.gateway.report.col.creator'),
    width: 120,
    render: row => <span class="text-slate-400">{row.creatorName || $t('page.gateway.report.timer')}</span>
  },
  {
    key: 'operations',
    title: $t('page.gateway.report.col.operations'),
    width: 80,
    render: row => (
      <NButton
        size="tiny"
        quaternary
        type="primary"
        disabled={detail.value?.reportId === row.reportId}
        onClick={() => loadDetail(row.reportId)}
      >
        {$t('page.gateway.report.view')}
      </NButton>
    )
  }
];

const rowProps = (row: Api.Gateway.EfficiencyReportView) => ({
  style: 'cursor: pointer;',
  onClick: () => loadDetail(row.reportId)
});

/** 详情 KPI 卡 */
const kpiCards = computed(() => {
  const kpi = detail.value?.content?.kpi;
  return [
    { key: 'coverage', label: $t('page.gateway.adoption.kpi.coverage'), value: `${(kpi?.coverage ?? 0).toFixed(1)}%`, sub: `${kpi?.activeUsers ?? 0}/${kpi?.totalUsers ?? 0}` },
    { key: 'newActive', label: $t('page.gateway.adoption.kpi.newActive'), value: `${kpi?.newActiveUsers ?? 0}` },
    { key: 'requests', label: $t('page.gateway.adoption.kpi.totalRequests'), value: `${kpi?.totalRequests ?? 0}` },
    { key: 'perCapita', label: $t('page.gateway.adoption.kpi.perCapitaTokens'), value: `${kpi?.perCapitaTokens ?? 0}` },
    { key: 'internal', label: $t('page.gateway.report.kpi.internal'), value: `¥${(kpi?.internalCost ?? 0).toFixed(2)}` },
    { key: 'external', label: $t('page.gateway.report.kpi.external'), value: `¥${(kpi?.externalCost ?? 0).toFixed(2)}` },
    { key: 'diff', label: $t('page.gateway.report.kpi.diff'), value: `¥${(kpi?.costDiff ?? 0).toFixed(2)}` },
    { key: 'days', label: $t('page.gateway.report.kpi.days'), value: `${kpi?.days ?? 0}` }
  ];
});

const deptColumns: DataTableColumn<Api.Gateway.AdoptionDeptRow>[] = [
  { key: 'deptName', title: $t('page.gateway.adoption.dept.col.name'), minWidth: 140 },
  { key: 'memberCount', title: $t('page.gateway.adoption.dept.col.member'), align: 'right', width: 80, render: r => `${r.memberCount}` },
  { key: 'activeCount', title: $t('page.gateway.adoption.dept.col.active'), align: 'right', width: 90, render: r => `${r.activeCount} / ${r.memberCount}` },
  { key: 'coverage', title: $t('page.gateway.adoption.dept.col.coverage'), align: 'right', width: 90, render: r => `${r.coverage.toFixed(1)}%` },
  { key: 'requests', title: $t('page.gateway.adoption.dept.col.requests'), align: 'right', width: 80, render: r => `${r.requests}` },
  { key: 'internalCost', title: $t('page.gateway.adoption.dept.col.internalCost'), align: 'right', width: 110, render: r => `¥${r.internalCost.toFixed(2)}` }
];

const modelColumns: DataTableColumn<Api.Gateway.AdoptionModelRow>[] = [
  { key: 'model', title: $t('page.gateway.adoption.model.col.model'), minWidth: 160 },
  { key: 'requests', title: $t('page.gateway.adoption.dept.col.requests'), align: 'right', width: 90, render: r => `${r.requests}` },
  { key: 'requestShare', title: $t('page.gateway.adoption.model.col.requestShare'), align: 'right', width: 90, render: r => `${r.requestShare.toFixed(1)}%` },
  { key: 'totalTokens', title: $t('page.gateway.adoption.dept.col.totalTokens'), align: 'right', width: 110, render: r => `${r.totalTokens}` },
  { key: 'internalCost', title: $t('page.gateway.adoption.dept.col.internalCost'), align: 'right', width: 110, render: r => `¥${r.internalCost.toFixed(2)}` },
  { key: 'activeUsers', title: $t('page.gateway.adoption.model.col.activeUsers'), align: 'right', width: 90, render: r => `${r.activeUsers}` }
];

const topUserColumns: DataTableColumn<Api.Gateway.CostDetailRow>[] = [
  { key: 'label', title: $t('page.gateway.cost.search.user'), minWidth: 140 },
  { key: 'requests', title: $t('page.gateway.adoption.dept.col.requests'), align: 'right', width: 90, render: r => `${r.requests}` },
  { key: 'totalTokens', title: $t('page.gateway.adoption.dept.col.totalTokens'), align: 'right', width: 110, render: r => `${r.totalTokens}` },
  { key: 'internalCost', title: $t('page.gateway.adoption.dept.col.internalCost'), align: 'right', width: 110, render: r => `¥${r.internalCost.toFixed(2)}` },
  { key: 'costDiff', title: $t('page.gateway.cost.detail.col.costDiff'), align: 'right', width: 110, render: r => `¥${r.costDiff.toFixed(2)}` }
];

async function copyMarkdown() {
  if (!detail.value?.contentMd) return;
  await copy(detail.value.contentMd);
  window.$message?.success($t('page.gateway.report.copySuccess'));
}

function handleExport() {
  if (!detail.value) return;
  download(`/gateway/report/export/${detail.value.reportId}`, {}, '');
}

function handleGenerated(view: Api.Gateway.EfficiencyReportView) {
  pageNum.value = 1;
  load();
  detail.value = view;
}

function listRowKey(row: Api.Gateway.EfficiencyReportView) {
  return String(row.reportId);
}

function deptRowKey(row: Api.Gateway.AdoptionDeptRow) {
  return String(row.deptId);
}

function modelRowKey(row: Api.Gateway.AdoptionModelRow) {
  return row.model;
}

function topUserRowKey(row: Api.Gateway.CostDetailRow) {
  return row.value;
}

onMounted(load);
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <NCard :title="$t('page.gateway.report.listTitle')" :bordered="false" size="small" class="card-wrapper">
      <template #header-extra>
        <NSpace align="center">
          <NRadioGroup v-model:value="typeFilter" size="small" @update:value="pageNum = 1; load()">
            <NRadioButton value="all">{{ $t('page.gateway.report.type.all') }}</NRadioButton>
            <NRadioButton value="weekly">{{ $t('page.gateway.report.type.weekly') }}</NRadioButton>
            <NRadioButton value="monthly">{{ $t('page.gateway.report.type.monthly') }}</NRadioButton>
            <NRadioButton value="custom">{{ $t('page.gateway.report.type.custom') }}</NRadioButton>
          </NRadioGroup>
          <NButton size="small" type="primary" @click="generateVisible = true">
            <template #icon>
              <icon-ant-design-plus-outlined class="text-icon" />
            </template>
            {{ $t('page.gateway.report.generate.title') }}
          </NButton>
        </NSpace>
      </template>
      <NDataTable
        :columns="listColumns"
        :data="rows"
        size="small"
        :loading="loading"
        :row-key="listRowKey"
        :pagination="pagination"
        :row-props="rowProps"
        :scroll-x="750"
      />
    </NCard>

    <NCard :bordered="false" size="small" class="card-wrapper">
      <template #header>
        <NSpace align="center">
          <span>{{ $t('page.gateway.report.detailTitle') }}</span>
          <template v-if="detail">
            <NTag size="small" :type="detail.reportType === 'weekly' ? 'info' : detail.reportType === 'monthly' ? 'success' : 'default'">
              {{ typeLabel(detail.reportType) }}
            </NTag>
            <span class="font-mono text-13px text-slate-400">{{ detail.periodStart }} ~ {{ detail.periodEnd }}</span>
          </template>
        </NSpace>
      </template>
      <template #header-extra>
        <NSpace v-if="detail">
          <span class="text-12px text-slate-400">
            {{ detail.creatorName || $t('page.gateway.report.timer') }} · {{ detail.createdAt?.replace('T', ' ').slice(0, 16) }}
          </span>
          <NButton size="small" tertiary @click="copyMarkdown">{{ $t('page.gateway.report.copyMd') }}</NButton>
          <NButton size="small" type="primary" ghost @click="handleExport">
            {{ $t('page.gateway.report.export') }}
          </NButton>
        </NSpace>
      </template>
      <NSpin :show="detailLoading">
        <template v-if="detail">
          <NAlert type="info" :bordered="false" class="mb-12px">{{ detail.summary }}</NAlert>
          <div class="grid grid-cols-2 gap-12px md:grid-cols-4 xl:grid-cols-8">
            <NCard v-for="card in kpiCards" :key="card.key" size="small" :bordered="true">
              <div class="flex flex-col gap-4px">
                <span class="text-12px text-slate-400">{{ card.label }}</span>
                <span class="text-16px font-semibold">{{ card.value }}</span>
                <span v-if="card.sub" class="text-12px text-slate-400">{{ card.sub }}</span>
              </div>
            </NCard>
          </div>
          <NTabs type="line" size="small" animated class="mt-12px">
            <NTabPane name="dept" :tab="$t('page.gateway.report.tab.dept')">
              <NDataTable
                :columns="deptColumns"
                :data="detail.content?.deptRows ?? []"
                size="small"
                :row-key="deptRowKey"
                :max-height="360"
                :scroll-x="620"
              />
            </NTabPane>
            <NTabPane name="model" :tab="$t('page.gateway.report.tab.model')">
              <NDataTable
                :columns="modelColumns"
                :data="detail.content?.modelRows ?? []"
                size="small"
                :row-key="modelRowKey"
                :max-height="360"
                :scroll-x="660"
              />
            </NTabPane>
            <NTabPane name="user" :tab="$t('page.gateway.report.tab.user')">
              <NDataTable
                :columns="topUserColumns"
                :data="detail.content?.topUsers ?? []"
                size="small"
                :row-key="topUserRowKey"
                :max-height="360"
                :scroll-x="570"
              />
            </NTabPane>
          </NTabs>
        </template>
        <NEmpty v-else :description="$t('page.gateway.report.emptyDetail')" class="py-48px" />
      </NSpin>
    </NCard>

    <ReportGenerateModal v-model:visible="generateVisible" @generated="handleGenerated" />
  </div>
</template>

<style scoped></style>
