<script setup lang="tsx">
import { computed, ref, watch } from 'vue';
import { NButton, NForm, NFormItem, NInput, NModal, NProgress, NTag } from 'naive-ui';
import type { DataTableColumns, FormInst } from 'naive-ui';
import {
  fetchGetBalanceConfig,
  fetchGetProviderBalances,
  fetchSaveBalanceConfig,
  fetchSyncProviderBalance
} from '@/service/api/gateway';
import { $t } from '@/locales';
import { formatDateTime } from '@/utils/format';

interface Props {
  provider: Api.Gateway.Provider;
}
const props = defineProps<Props>();

const detail = ref<Api.Gateway.ProviderBalanceDetail | null>(null);
const loading = ref(false);
const syncing = ref(false);

async function getBalanceData() {
  loading.value = true;
  const { data, error } = await fetchGetProviderBalances(props.provider.providerId!);
  if (!error && data) detail.value = data;
  loading.value = false;
}

watch(() => props.provider.providerId, getBalanceData, { immediate: true });

const summary = computed(() => detail.value?.summary ?? null);

/** 剩余率(%),总额度为 0 时按 0 展示 */
function rateOf(surplus: number, total: number): number {
  if (total <= 0) return 0;
  return Math.round(((total - surplus) / total) * 1000) / 10;
}

const summaryRate = computed(() =>
  summary.value ? rateOf(summary.value.surplusValue, summary.value.totalValue) : 0
);

const SPEC_LABELS = {
  standard: 'page.gateway.balance.specStandard',
  pro: 'page.gateway.balance.specPro',
  max: 'page.gateway.balance.specMax'
} as const;

const STATUS_TAGS: Record<string, 'success' | 'warning' | 'error' | 'default'> = {
  NORMAL: 'success',
  LIMIT: 'error',
  CREATING: 'default',
  RELEASE: 'warning',
  STOP: 'warning',
  REFUNDED: 'default'
};

const columns: DataTableColumns<Api.Gateway.ProviderBalance> = [
  {
    key: 'itemName',
    title: () => $t('page.gateway.balance.col.itemName'),
    minWidth: 140,
    ellipsis: { tooltip: true },
    render: row => row.itemName || row.itemKey || '-'
  },
  {
    key: 'itemType',
    title: () => $t('page.gateway.balance.col.itemType'),
    align: 'center',
    width: 96,
    render: row => (
      <NTag size="small" type={row.itemType === 'seat' ? 'info' : 'warning'}>
        {$t(row.itemType === 'seat' ? 'page.gateway.balance.typeSeat' : 'page.gateway.balance.typePackage')}
      </NTag>
    )
  },
  {
    key: 'specType',
    title: () => $t('page.gateway.balance.col.specType'),
    align: 'center',
    width: 100,
    render: row => {
      const labelKey = SPEC_LABELS[row.specType as keyof typeof SPEC_LABELS];
      return labelKey ? $t(labelKey) : row.specType || '-';
    }
  },
  {
    key: 'status',
    title: () => $t('page.gateway.balance.col.status'),
    align: 'center',
    width: 96,
    render: row => (
      <NTag size="small" type={STATUS_TAGS[row.status] ?? 'default'}>
        {row.status || '-'}
      </NTag>
    )
  },
  {
    key: 'cycleEnd',
    title: () => $t('page.gateway.balance.col.cycleEnd'),
    align: 'center',
    width: 110,
    render: row => (row.cycleEnd ? formatDateTime(row.cycleEnd).slice(0, 10) : '-')
  },
  {
    key: 'totalValue',
    title: () => $t('page.gateway.balance.col.totalValue'),
    align: 'right',
    width: 100,
    render: row => row.totalValue.toFixed(2)
  },
  {
    key: 'usedValue',
    title: () => $t('page.gateway.balance.col.usedValue'),
    align: 'right',
    width: 100,
    render: row => row.usedValue.toFixed(2)
  },
  {
    key: 'surplusValue',
    title: () => $t('page.gateway.balance.col.surplusValue'),
    align: 'right',
    width: 100,
    render: row => row.surplusValue.toFixed(2)
  },
  {
    key: 'usageRate',
    title: () => $t('page.gateway.dashboard.usageRate'),
    align: 'center',
    minWidth: 140,
    render: row => {
      const rate = rateOf(row.surplusValue, row.totalValue);
      return (
        <NProgress
          type="line"
          percentage={Math.min(rate, 100)}
          status={rate >= 85 ? 'error' : rate >= 60 ? 'warning' : 'success'}
        />
      );
    }
  }
];

async function handleSync() {
  syncing.value = true;
  const { data, error } = await fetchSyncProviderBalance(props.provider.providerId!);
  syncing.value = false;
  if (error) return;
  window.$message?.success($t('page.gateway.balance.syncSuccess'));
  detail.value = { summary: data, items: detail.value?.items ?? [] };
  getBalanceData();
}

// 采集配置(AK/SK,掩码占位保留旧明文)
const configVisible = ref(false);
const configSaving = ref(false);
const configFormRef = ref<FormInst | null>(null);
const configModel = ref<Api.Gateway.BalanceSyncConfig>({ accessKeyId: '', accessKeySecret: '', region: '' });

async function openConfig() {
  const { data, error } = await fetchGetBalanceConfig(props.provider.providerId!);
  if (!error && data) configModel.value = { ...data };
  configVisible.value = true;
}

async function handleSaveConfig() {
  await configFormRef.value?.validate();
  configSaving.value = true;
  const { error } = await fetchSaveBalanceConfig(props.provider.providerId!, configModel.value);
  configSaving.value = false;
  if (error) return;
  window.$message?.success($t('page.gateway.balance.configSaved'));
  configVisible.value = false;
}
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper">
    <template #header>
      <div class="flex items-center gap-8px">
        <span class="font-500">{{ $t('page.gateway.balance.title') }}</span>
        <NTag size="small" type="info">{{ summary?.planLabel ?? provider.providerType }}</NTag>
        <span class="text-12px text-slate-400">{{ $t('page.gateway.balance.vendorSideNote') }}</span>
      </div>
    </template>
    <template #header-extra>
      <NSpace :size="8">
        <NButton size="small" :loading="syncing" @click="handleSync">
          <template #icon>
            <icon-material-symbols-sync-rounded class="text-icon" />
          </template>
          {{ $t('page.gateway.balance.sync') }}
        </NButton>
        <NButton size="small" type="primary" @click="openConfig">
          <template #icon>
            <icon-material-symbols-settings-outline class="text-icon" />
          </template>
          {{ $t('page.gateway.balance.config') }}
        </NButton>
      </NSpace>
    </template>

    <div v-if="summary" class="mb-12px flex flex-wrap items-center gap-x-24px gap-y-8px px-4px">
      <div class="flex items-baseline gap-6px">
        <span class="text-12px text-slate-400">{{ $t('page.gateway.balance.col.totalValue') }}</span>
        <span class="text-16px font-600">{{ summary.totalValue.toFixed(2) }}</span>
      </div>
      <div class="flex items-baseline gap-6px">
        <span class="text-12px text-slate-400">{{ $t('page.gateway.balance.col.usedValue') }}</span>
        <span class="text-16px font-600">{{ summary.usedValue.toFixed(2) }}</span>
      </div>
      <div class="flex items-baseline gap-6px">
        <span class="text-12px text-slate-400">{{ $t('page.gateway.balance.col.surplusValue') }}</span>
        <span class="text-16px font-600">{{ summary.surplusValue.toFixed(2) }}</span>
      </div>
      <div class="flex w-160px flex-col gap-2px">
        <div class="flex items-center justify-between">
          <span class="text-12px text-slate-400">{{ $t('page.gateway.dashboard.usageRate') }}</span>
          <span class="text-12px font-500">{{ summaryRate }}%</span>
        </div>
        <NProgress
          type="line"
          :show-indicator="false"
          :percentage="Math.min(summaryRate, 100)"
          :status="summaryRate >= 85 ? 'error' : summaryRate >= 60 ? 'warning' : 'success'"
        />
      </div>
      <div class="flex items-baseline gap-6px">
        <span class="text-12px text-slate-400">{{ $t('page.gateway.balance.col.seatCount') }}</span>
        <span class="text-14px font-500">{{ summary.seatCount }}</span>
        <span class="text-12px text-slate-400">{{ $t('page.gateway.balance.col.packageCount') }}</span>
        <span class="text-14px font-500">{{ summary.packageCount }}</span>
      </div>
      <div class="ml-auto text-12px text-slate-400">
        {{ $t('page.gateway.balance.lastSync') }}:
        {{ summary.syncedAt ? formatDateTime(summary.syncedAt) : $t('page.gateway.balance.neverSynced') }}
      </div>
    </div>

    <NDataTable
      :columns="columns"
      :data="detail?.items ?? []"
      :loading="loading"
      size="small"
      :row-key="row => row.balanceId"
      max-height="calc(100vh - 520px)"
    />

    <NModal
      v-model:show="configVisible"
      preset="card"
      :title="$t('page.gateway.balance.configTitle')"
      class="w-480px"
    >
      <NForm ref="configFormRef" :model="configModel" label-placement="left" label-width="120">
        <NFormItem :label="$t('page.gateway.balance.configAccessKeyId')" path="accessKeyId" required>
          <NInput v-model:value="configModel.accessKeyId" :placeholder="$t('page.gateway.balance.configMaskTip')" />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.balance.configAccessKeySecret')" path="accessKeySecret" required>
          <NInput
            v-model:value="configModel.accessKeySecret"
            type="password"
            show-password-on="click"
            :placeholder="$t('page.gateway.balance.configMaskTip')"
          />
        </NFormItem>
        <NFormItem :label="$t('page.gateway.balance.configRegion')" path="region">
          <NInput v-model:value="configModel.region" placeholder="cn-beijing" />
        </NFormItem>
        <div class="text-12px text-slate-400">{{ $t('page.gateway.balance.configTip') }}</div>
      </NForm>
      <template #footer>
        <div class="flex justify-end gap-12px">
          <NButton size="small" @click="configVisible = false">{{ $t('common.cancel') }}</NButton>
          <NButton size="small" type="primary" :loading="configSaving" @click="handleSaveConfig">
            {{ $t('common.confirm') }}
          </NButton>
        </div>
      </template>
    </NModal>
  </NCard>
</template>
