<script setup lang="ts">
import { computed } from 'vue';
import { $t } from '@/locales';

defineOptions({ name: 'DashboardOverview' });

interface Props {
  data?: Api.Gateway.DashboardOverview;
}

const props = defineProps<Props>();

const cards = computed(() => {
  const d = props.data;
  return [
    {
      key: 'requests',
      label: 'page.gateway.dashboard.totalRequests',
      value: `${d?.totalRequests ?? 0}`,
      sub: ''
    },
    {
      key: 'cost',
      label: 'page.gateway.dashboard.totalCost',
      value: `¥${d?.totalCost ?? 0}`,
      sub: `${$t('page.gateway.dashboard.internalCost')} ¥${d?.internalCost ?? 0}`
    },
    {
      key: 'tokens',
      label: 'page.gateway.dashboard.totalTokens',
      value: `${d?.totalTokens ?? 0}`,
      sub: `${$t('page.gateway.dashboard.input')} ${d?.inputTokens ?? 0} / ${$t('page.gateway.dashboard.output')} ${d?.outputTokens ?? 0}`
    },
    {
      key: 'cache',
      label: 'page.gateway.dashboard.cacheRead',
      value: `${d?.cacheReadTokens ?? 0}`,
      sub: ''
    },
    {
      key: 'budget',
      label: 'page.gateway.dashboard.budgetTotal',
      value: `¥${d?.budgetUsedTotal ?? 0} / ¥${d?.budgetLimitTotal ?? 0}`,
      sub: ''
    }
  ] as const;
});
</script>

<template>
  <div class="grid grid-cols-2 gap-12px sm:grid-cols-3 lg:grid-cols-5">
    <NCard v-for="card in cards" :key="card.key" size="small" class="card-wrapper">
      <div class="flex flex-col gap-4px">
        <span class="text-12px text-slate-400">{{ $t(card.label) }}</span>
        <span class="text-20px font-semibold text-slate-900 dark:text-slate-100">{{ card.value }}</span>
        <span v-if="card.sub" class="text-12px text-slate-400">{{ card.sub }}</span>
      </div>
    </NCard>
  </div>
</template>

<style scoped></style>
