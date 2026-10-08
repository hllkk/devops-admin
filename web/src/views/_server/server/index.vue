<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';
import { fetchGetAssetOverview } from '@/service/api/server';
import { ASSET_TYPE_OPTIONS, MONITOR_STATUS_OPTIONS, AGENT_STATUS_OPTIONS } from '@/constants/business/server';

defineOptions({ name: 'ServerOverview' });

const router = useRouter();

const loading = ref(false);
const overview = ref<Api.Server.AssetOverview | null>(null);

async function loadOverview() {
  loading.value = true;
  const { error, data } = await fetchGetAssetOverview();
  loading.value = false;
  if (!error && data) {
    overview.value = data;
  }
}

onMounted(() => {
  loadOverview();
});

function typeCount(v: string): number {
  return overview.value?.byType?.[v] ?? 0;
}
function monitorCount(v: string): number {
  return overview.value?.monitorStatus?.[v] ?? 0;
}
function agentCount(v: string): number {
  return overview.value?.agentStatus?.[v] ?? 0;
}

// ── 第一行:资产总量卡片(总数/启用/五类型) ──
const assetCards = computed(() => [
  {
    key: 'total',
    icon: 'lucide:server',
    label: $t('page.server.overview.totalAssets'),
    value: overview.value?.total ?? 0,
    sub: $t('page.server.overview.activeSub', { count: overview.value?.activeTotal ?? 0 })
  },
  ...ASSET_TYPE_OPTIONS.map(o => ({
    key: o.value,
    icon:
      o.value === 'physical'
        ? 'lucide:server'
        : o.value === 'vm'
          ? 'lucide:monitor'
          : o.value === 'docker_host'
            ? 'lucide:container'
            : o.value === 'db_instance'
              ? 'lucide:database'
              : 'lucide:network',
    label: $t(o.label),
    value: typeCount(o.value),
    sub: ''
  }))
]);

// ── 第二行:监控在线状态(启用中口径) ──
const monitorCards = computed(() =>
  MONITOR_STATUS_OPTIONS.map(o => ({
    key: o.value,
    label: $t(o.label),
    value: monitorCount(o.value),
    tone:
      o.value === 'online' ? 'success' : o.value === 'offline' ? 'error' : 'default'
  }))
);

// ── 第三行:Agent 状态(启用中口径) ──
const agentCards = computed(() =>
  AGENT_STATUS_OPTIONS.map(o => ({
    key: o.value,
    label: $t(o.label),
    value: agentCount(o.value),
    tone:
      o.value === 'running' ? 'success' : o.value === 'lost' ? 'error' : o.value === 'installing' ? 'warning' : 'default'
  }))
);

function gotoAsset() {
  router.push({ name: 'asset' });
}
</script>

<template>
  <div class="min-h-500px flex-col-stretch gap-16px overflow-hidden flex-shrink-0 lt-sm:overflow-auto">
    <NCard :title="$t('page.server.overview.title')" :bordered="false" size="small" class="card-wrapper">
      <template #header-extra>
        <NSpace size="small">
          <NButton size="small" type="primary" ghost @click="gotoAsset">
            {{ $t('page.server.overview.gotoAsset') }}
          </NButton>
          <ButtonIcon text type="primary" icon="material-symbols:refresh" :tooltip-content="$t('common.refresh')" @click="loadOverview" />
        </NSpace>
      </template>
      <NSpin :show="loading">
        <!-- 第一行:资产总量 + 五类型 -->
        <div class="grid grid-cols-2 gap-12px sm:grid-cols-3 lg:grid-cols-6">
          <NCard v-for="card in assetCards" :key="card.key" size="small" class="card-wrapper cursor-pointer" @click="gotoAsset">
            <div class="flex items-center gap-12px">
              <div class="flex h-40px w-40px shrink-0 flex-center rounded-6px bg-primary-100 dark:bg-primary-900">
                <SvgIcon :icon="card.icon" class="text-22px color-primary" />
              </div>
              <div class="flex flex-col gap-2px">
                <span class="text-12px text-slate-400">{{ card.label }}</span>
                <span class="text-22px font-600">{{ card.value }}</span>
                <span v-if="card.sub" class="text-12px text-slate-400">{{ card.sub }}</span>
              </div>
            </div>
          </NCard>
        </div>

        <!-- 第二行:监控在线状态 -->
        <div class="mt-16px flex items-center gap-12px">
          <span class="text-13px font-500">{{ $t('page.server.overview.monitorSection') }}</span>
          <NDivider class="flex-1" />
        </div>
        <div class="mt-8px grid grid-cols-2 gap-12px sm:grid-cols-3">
          <NCard v-for="card in monitorCards" :key="card.key" size="small" class="card-wrapper">
            <div class="flex items-center justify-between">
              <span class="text-13px text-slate-400">{{ card.label }}</span>
              <span
                class="text-22px font-600"
                :class="{
                  'color-success': card.tone === 'success',
                  'color-error': card.tone === 'error'
                }"
              >
                {{ card.value }}
              </span>
            </div>
          </NCard>
        </div>

        <!-- 第三行:Agent 状态 -->
        <div class="mt-16px flex items-center gap-12px">
          <span class="text-13px font-500">{{ $t('page.server.overview.agentSection') }}</span>
          <NDivider class="flex-1" />
        </div>
        <div class="mt-8px grid grid-cols-2 gap-12px sm:grid-cols-4">
          <NCard v-for="card in agentCards" :key="card.key" size="small" class="card-wrapper">
            <div class="flex items-center justify-between">
              <span class="text-13px text-slate-400">{{ card.label }}</span>
              <span
                class="text-22px font-600"
                :class="{
                  'color-success': card.tone === 'success',
                  'color-error': card.tone === 'error',
                  'color-warning': card.tone === 'warning'
                }"
              >
                {{ card.value }}
              </span>
            </div>
          </NCard>
        </div>
      </NSpin>
    </NCard>
  </div>
</template>

<style scoped></style>
