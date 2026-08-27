<script setup lang="ts">
import { computed } from 'vue';
import { NEmpty, NProgress, NTag } from 'naive-ui';
import { $t } from '@/locales';
import { formatDateTime } from '@/utils/format';

defineOptions({ name: 'DashboardBalance' });

interface Props {
  data: Api.Gateway.ProviderBalanceSummary[];
}
const props = defineProps<Props>();

/** 已用率(%),总额度 0 时按 0 展示 */
function rateOf(surplus: number, total: number): number {
  if (total <= 0) return 0;
  return Math.round(((total - surplus) / total) * 1000) / 10;
}

const empty = computed(() => !props.data.length);
</script>

<template>
  <NCard :bordered="false" size="small" class="card-wrapper">
    <template #header>
      <div class="flex items-center gap-8px">
        <span class="font-500">{{ $t('page.gateway.balance.dashboardTitle') }}</span>
        <span class="text-12px text-slate-400">{{ $t('page.gateway.balance.vendorSideNote') }}</span>
      </div>
    </template>
    <div v-if="!empty" class="flex flex-col gap-10px">
      <div
        v-for="row in data"
        :key="row.providerId"
        class="flex flex-wrap items-center gap-x-16px gap-y-6px rounded-md bg-slate-50 px-12px py-10px dark:bg-slate-800"
      >
        <div class="flex min-w-140px items-center gap-8px">
          <span class="text-13px font-500">{{ row.providerName }}</span>
          <NTag size="small" type="info">{{ row.planLabel }}</NTag>
        </div>
        <span class="text-12px text-slate-400">
          {{ $t('page.gateway.balance.col.totalValue') }}
          <span class="text-13px font-600 text-slate-700 dark:text-slate-200">{{ row.totalValue.toFixed(2) }}</span>
        </span>
        <span class="text-12px text-slate-400">
          {{ $t('page.gateway.balance.col.usedValue') }}
          <span class="text-13px font-600 text-slate-700 dark:text-slate-200">{{ row.usedValue.toFixed(2) }}</span>
        </span>
        <span class="text-12px text-slate-400">
          {{ $t('page.gateway.balance.col.surplusValue') }}
          <span class="text-13px font-600 text-slate-700 dark:text-slate-200">{{ row.surplusValue.toFixed(2) }}</span>
        </span>
        <span class="text-12px text-slate-400">
          {{ $t('page.gateway.balance.col.seatCount') }} {{ row.seatCount }}
          <template v-if="row.packageCount">
            · {{ $t('page.gateway.balance.col.packageCount') }} {{ row.packageCount }}
          </template>
        </span>
        <div class="ml-auto flex w-180px items-center gap-8px">
          <NProgress
            type="line"
            :show-indicator="false"
            class="flex-1"
            :percentage="Math.min(rateOf(row.surplusValue, row.totalValue), 100)"
            :status="rateOf(row.surplusValue, row.totalValue) >= 85 ? 'error' : rateOf(row.surplusValue, row.totalValue) >= 60 ? 'warning' : 'success'"
          />
          <span class="w-42px text-right text-12px font-500">{{ rateOf(row.surplusValue, row.totalValue) }}%</span>
        </div>
        <span class="w-full text-right text-11px text-slate-400 lt-lg:w-auto">
          {{ $t('page.gateway.balance.lastSync') }}:
          {{ row.syncedAt ? formatDateTime(row.syncedAt) : $t('page.gateway.balance.neverSynced') }}
        </span>
      </div>
    </div>
    <div v-else class="flex-center py-16px">
      <NEmpty :description="$t('page.gateway.balance.dashboardEmpty')" size="small" />
    </div>
  </NCard>
</template>
