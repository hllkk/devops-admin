<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { fetchGetProviderList } from '@/service/api/gateway';
import { $t } from '@/locales';

defineOptions({ name: 'GatewayManage' });

// ===== 顶部 Tab：供应商/模型/AI密钥/用量，对齐 AIHelms 各 *Manage 能力，逐切片补 =====
type GatewayTab = 'provider' | 'model' | 'aiKey' | 'usage';
type GatewayTabI18nKey =
  | 'page.gateway.tab.provider'
  | 'page.gateway.tab.model'
  | 'page.gateway.tab.aiKey'
  | 'page.gateway.tab.usage';
const tabs: { key: GatewayTab; i18nKey: GatewayTabI18nKey }[] = [
  { key: 'provider', i18nKey: 'page.gateway.tab.provider' },
  { key: 'model', i18nKey: 'page.gateway.tab.model' },
  { key: 'aiKey', i18nKey: 'page.gateway.tab.aiKey' },
  { key: 'usage', i18nKey: 'page.gateway.tab.usage' }
];
const activeTab = ref<GatewayTab>('provider');
const activeTabMeta = computed(() => tabs.find(t => t.key === activeTab.value)!);

// ===== 供应商列表（左面板，read-only：新建/编辑/删除/启停在下一步落地）=====
const providers = ref<Api.Gateway.Provider[]>([]);
const selectedProvider = ref<Api.Gateway.Provider | null>(null);
const searchName = ref('');
const loading = ref(false);

async function fetchProviders() {
  loading.value = true;
  try {
    const { data, error } = await fetchGetProviderList({ pageNum: 1, pageSize: 100 });
    if (!error && data) {
      providers.value = data.rows;
    }
  } finally {
    loading.value = false;
  }
}

// 客户端按名称过滤（供应商量小，免后端往返）
const filteredProviders = computed(() => {
  const kw = searchName.value.trim().toLowerCase();
  if (!kw) return providers.value;
  return providers.value.filter(p => p.name.toLowerCase().includes(kw));
});

function handleSelectProvider(p: Api.Gateway.Provider) {
  selectedProvider.value = p;
}

onMounted(() => {
  fetchProviders();
});
</script>

<template>
  <div class="h-full flex flex-col gap-12px p-16px">
    <!-- 顶部 Tab 切换 -->
    <div
      class="flex items-center gap-4px rounded-12px border border-slate-200/60 bg-white/80 p-4px dark:border-slate-700/60 dark:bg-slate-800/60"
    >
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="rounded-8px px-16px py-6px text-14px font-medium transition-colors"
        :class="
          activeTab === tab.key
            ? 'text-white'
            : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-700/60'
        "
        :style="activeTab === tab.key ? { backgroundColor: 'rgb(var(--primary-600-color))' } : undefined"
        @click="activeTab = tab.key"
      >
        {{ $t(tab.i18nKey) }}
      </button>
    </div>

    <!-- 供应商 Tab：master-detail（左供应商列表 + 右凭证占位），对齐 AIHelms ProviderManage 布局 -->
    <div v-show="activeTab === 'provider'" class="flex flex-1 gap-12px overflow-hidden">
      <!-- 左：供应商列表 -->
      <div
        class="flex w-300px shrink-0 flex-col overflow-hidden rounded-16px border border-slate-200/60 bg-white/80 shadow-sm dark:border-slate-700/60 dark:bg-slate-800/60"
      >
        <div
          class="flex h-48px shrink-0 items-center justify-between border-b border-slate-200/60 px-16px dark:border-slate-700/60"
        >
          <span class="text-14px font-semibold text-slate-900 dark:text-slate-100">
            {{ $t('page.gateway.provider.title') }}
          </span>
          <span class="text-12px text-slate-400">{{ filteredProviders.length }}</span>
        </div>
        <div class="shrink-0 p-8px">
          <NInput
            v-model:value="searchName"
            size="small"
            clearable
            :placeholder="$t('page.gateway.provider.searchPlaceholder')"
          />
        </div>
        <div class="flex-1 overflow-y-auto p-8px">
          <div v-if="loading" class="py-24px text-center text-12px text-slate-400">…</div>
          <template v-else>
            <div
              v-for="p in filteredProviders"
              :key="p.providerId"
              class="mb-4px flex cursor-pointer items-center gap-8px rounded-8px px-12px py-10px transition-colors hover:bg-slate-100 dark:hover:bg-slate-700/60"
              :style="
                selectedProvider?.providerId === p.providerId
                  ? { backgroundColor: 'rgb(var(--primary-600-color) / 0.10)' }
                  : undefined
              "
              @click="handleSelectProvider(p)"
            >
              <span class="size-8px shrink-0 rounded-full" :class="p.isActive ? 'bg-green-500' : 'bg-slate-300'" />
              <div class="min-w-0 flex-1">
                <div class="truncate text-14px font-medium text-slate-900 dark:text-slate-100">{{ p.name }}</div>
                <div class="mt-2px truncate text-12px text-slate-400">{{ p.providerType }}</div>
              </div>
            </div>
            <div v-if="!filteredProviders.length" class="py-24px text-center text-12px text-slate-400">
              {{ $t('page.gateway.provider.empty') }}
            </div>
          </template>
        </div>
      </div>

      <!-- 右：凭证管理占位（待后端 slice2 Credential 落地补全 master-detail 右半）-->
      <div
        class="flex flex-1 items-center justify-center rounded-16px border border-dashed border-slate-300 bg-white/60 dark:border-slate-700 dark:bg-slate-800/40"
      >
        <div v-if="!selectedProvider" class="text-14px text-slate-400">
          {{ $t('page.gateway.provider.selectPrompt') }}
        </div>
        <div v-else class="flex flex-col items-center gap-8px">
          <span class="text-16px font-semibold text-slate-700 dark:text-slate-200">{{ selectedProvider.name }}</span>
          <p class="text-12px text-slate-400">{{ $t('page.gateway.provider.credPlaceholder') }}</p>
        </div>
      </div>
    </div>

    <!-- 其他 Tab 占位（随各自后端落地逐个实现）-->
    <div
      v-show="activeTab !== 'provider'"
      class="flex flex-1 items-center justify-center rounded-16px border border-dashed border-slate-300 bg-white/60 dark:border-slate-700 dark:bg-slate-800/40"
    >
      <span class="text-14px text-slate-400">
        {{ $t(activeTabMeta.i18nKey) }} · {{ $t('page.gateway.comingSoon') }}
      </span>
    </div>
  </div>
</template>
