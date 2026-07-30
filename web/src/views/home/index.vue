<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { useClipboard } from '@vueuse/core';
import type { LastLevelRouteKey } from '@elegant-router/types';
import { useAuthStore } from '@/store/modules/auth';
import { useTabStore } from '@/store/modules/tab';
import { useThemeStore } from '@/store/modules/theme';
import { useRouterPush } from '@/hooks/common/router';
import { useEcharts } from '@/hooks/common/echarts';
import { useSvgIcon } from '@/hooks/common/icon';
import { getRgb } from '@sa/color';
import { ALL_MODULES, MODULE_CONFIG, type RouteModule } from '@/constants/module';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';
import SystemLogo from '@/components/common/system-logo.vue';
import defaultAvatar from '@/assets/imgs/soybean.jpg';

defineOptions({ name: 'Home' });

const authStore = useAuthStore();
const user = computed(() => authStore.userInfo.user);
const { SvgIconVNode } = useSvgIcon();
const { copy: copyText, copied } = useClipboard({ legacy: true, copiedDuring: 2000 });

const showFullKey = ref(false);
const isLoading = ref(true);

const tabStore = useTabStore();
const { routerPushByKey } = useRouterPush();

// ===== 我的应用：数据由后端 getUserInfo.apps 下发（按模块菜单权限聚合），前端只渲染 =====
// 再与 ALL_MODULES 对齐:保证类型安全 + 兜底后端可能下发的非法/历史脏值。
const myApps = computed<RouteModule[]>(() =>
  (authStore.userInfo.apps ?? []).filter((m): m is RouteModule => ALL_MODULES.includes(m as RouteModule))
);

/** 点击模块卡片 → 清空旧模块 Tab → 重建 homeTab → 导航到模块首页（与 module-select 一致） */
function handleOpenApp(module: RouteModule) {
  const homeRoute = MODULE_CONFIG[module].home as LastLevelRouteKey;
  tabStore.resetTabs(homeRoute);
  routerPushByKey(homeRoute);
}

// ===== 顶部 Tab：我的应用 / 我的AI身份 =====
type HomeTab = 'apps' | 'identity';
type HomeTabI18nKey = 'page.home.myApps.title' | 'page.home.identity.navIdentity';
const homeTabs: { key: HomeTab; i18nKey: HomeTabI18nKey }[] = [
  { key: 'apps', i18nKey: 'page.home.myApps.title' },
  { key: 'identity', i18nKey: 'page.home.identity.navIdentity' }
];
const activeTab = ref<HomeTab>('apps');

// ===== 用户下拉（个人中心 / 退出登录）=====
const dropdownOptions = computed(() => [
  { label: $t('common.logout'), key: 'logout', icon: SvgIconVNode({ icon: 'ph:sign-out', fontSize: 18 }) }
]);

function handleLogout() {
  window.$dialog?.info({
    title: $t('common.tip'),
    content: $t('common.logoutConfirm'),
    positiveText: $t('common.confirm'),
    negativeText: $t('common.cancel'),
    onPositiveClick: () => {
      authStore.logout();
    }
  });
}

function handleUserMenu(key: string) {
  if (key === 'logout') {
    handleLogout();
  }
}

// ============ MOCK 数据：AI 网关后端就绪后替换为真实接口 ============
// 当前 devops-admin 的 AI 网关后端尚未实现，个人中心的 AI 身份 / 用量 / 申请
// 数据用前端假数据跑通展示；后端 /gateway/identity/* 就绪后改为真实请求。
interface IdentityKey {
  /** API Key 明文（仅前端脱敏展示，真实场景由后端按需下发） */
  keyValue: string;
  /** 是否激活 */
  isActive: boolean;
  /** 预算口径 */
  budgetScope: 'unified' | 'per_type' | 'per_resource';
  /** 统一预算上限（null 表示不限） */
  budgetLimit: number | null;
  /** 分类型预算：模型总额 */
  budgetModelsTotal: number | null;
  /** 分类型预算：MCP 总额 */
  budgetMcpsTotal: number | null;
  /** 已授权模型 */
  models: string[];
  /** 已授权 MCP（ID 列表） */
  mcps: number[];
  /** 已授权 Skill（ID 列表） */
  skills: number[];
}
interface UsageKpi {
  totalCost: number;
  totalRequests: number;
}
interface TrendItem {
  period: string;
  cost: number;
}
interface ResourceApplication {
  id: number;
  resourceType: 'model' | 'mcp' | 'skill' | 'agent';
  resourceId: number;
  resourceName: string;
  status: 'pending' | 'approved' | 'rejected';
}

const mainKey = ref<IdentityKey | null>(null);
const kpi = ref<UsageKpi | null>(null);
const trend = ref<TrendItem[]>([]);
const applications = ref<ResourceApplication[]>([]);
const mcpNames = ref<Record<number, string>>({});
const skillNames = ref<Record<number, string>>({});

const MOCK_KEY: IdentityKey = {
  keyValue: 'sk-devops-9f3c2a1b7e8d4c5f0a6b3c2d1e8f',
  isActive: true,
  budgetScope: 'unified',
  budgetLimit: 500,
  budgetModelsTotal: null,
  budgetMcpsTotal: null,
  models: ['gpt-4o', 'claude-3.5-sonnet', 'deepseek-chat', 'qwen-max'],
  mcps: [101, 102],
  skills: [201, 202, 203]
};
const MOCK_KPI: UsageKpi = { totalCost: 326.84, totalRequests: 18420 };
const MOCK_TREND: TrendItem[] = [
  { period: '07-23', cost: 28.5 },
  { period: '07-24', cost: 42.1 },
  { period: '07-25', cost: 35.7 },
  { period: '07-26', cost: 51.3 },
  { period: '07-27', cost: 18.9 },
  { period: '07-28', cost: 62.4 },
  { period: '07-29', cost: 87.94 }
];
const MOCK_APPLICATIONS: ResourceApplication[] = [
  { id: 1, resourceType: 'model', resourceId: 0, resourceName: 'gpt-4o', status: 'approved' },
  { id: 2, resourceType: 'mcp', resourceId: 103, resourceName: 'MCP-103', status: 'pending' },
  { id: 3, resourceType: 'skill', resourceId: 204, resourceName: 'Skill-204', status: 'rejected' }
];
const MOCK_MCP_NAMES: Record<number, string> = { 101: '文件检索', 102: '代码执行', 103: '数据库查询' };
const MOCK_SKILL_NAMES: Record<number, string> = { 201: 'SQL 生成', 202: '文档摘要', 203: '代码评审', 204: '数据分析' };

const fullKey = computed(() => mainKey.value?.keyValue ?? '');
const maskedKey = computed(() => {
  const value = fullKey.value;
  if (!value) return 'sk-xxxxxxxxxxxx';
  return value.length > 12 ? `${value.slice(0, 7)}****${value.slice(-4)}` : value;
});
const displayKey = computed(() => (showFullKey.value ? fullKey.value : maskedKey.value));

const budgetDisplay = computed(() => {
  const key = mainKey.value;
  if (!key) return $t('page.home.identity.budgetUnlimited');
  if (key.budgetScope === 'unified') {
    return key.budgetLimit ? `¥${key.budgetLimit}` : $t('page.home.identity.budgetUnlimited');
  }
  if (key.budgetScope === 'per_type') {
    const parts: string[] = [];
    if (key.budgetModelsTotal) parts.push($t('page.home.identity.budgetModels', { amount: key.budgetModelsTotal }));
    if (key.budgetMcpsTotal) parts.push($t('page.home.identity.budgetMcps', { amount: key.budgetMcpsTotal }));
    return parts.length ? parts.join(' / ') : $t('page.home.identity.budgetUnlimited');
  }
  return $t('page.home.identity.budgetPerResource');
});

const totalBudget = computed(() => {
  const key = mainKey.value;
  if (!key) return null;
  if (key.budgetScope === 'unified') return key.budgetLimit ?? null;
  if (key.budgetScope === 'per_type') {
    const models = key.budgetModelsTotal ?? 0;
    const mcps = key.budgetMcpsTotal ?? 0;
    return models + mcps > 0 ? models + mcps : null;
  }
  return null;
});

const budgetUsedPercent = computed(() => {
  const budget = totalBudget.value;
  if (!budget) return null;
  const spent = kpi.value?.totalCost ?? 0;
  return Math.min((spent / budget) * 100, 100);
});

const dailyAvgCost = computed(() => {
  const cost = kpi.value?.totalCost ?? 0;
  const day = new Date().getDate();
  return day > 0 ? cost / day : 0;
});

const themeStore = useThemeStore();
/** 图表主色:跟随主题(Echarts 用 canvas 渲染,不解析 CSS 变量,故取实际色值) */
const chartColor = computed(() => themeStore.themeColors.primary);
const chartColorRgb = computed(() => getRgb(chartColor.value));

const { domRef: trendChartRef, updateOptions: updateTrendChart } = useEcharts(() => {
  const { r, g, b } = chartColorRgb.value;
  return {
    tooltip: { trigger: 'axis' },
    grid: { top: 20, right: 20, bottom: 30, left: 50 },
    xAxis: {
      type: 'category',
      data: [] as string[],
      axisLabel: { fontSize: 10, color: '#94a3b8' },
      axisLine: { lineStyle: { color: '#e2e8f0' } }
    },
    yAxis: {
      type: 'value',
      axisLabel: { fontSize: 10, color: '#94a3b8', formatter: '¥{value}' },
      splitLine: { lineStyle: { color: '#f1f5f9' } }
    },
    series: [
      {
        type: 'line',
        data: [] as number[],
        smooth: true,
        symbol: 'circle',
        symbolSize: 4,
        lineStyle: { color: chartColor.value, width: 2 },
        itemStyle: { color: chartColor.value },
        areaStyle: {
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: `rgba(${r},${g},${b},0.15)` },
              { offset: 1, color: `rgba(${r},${g},${b},0)` }
            ]
          }
        }
      }
    ]
  };
});

/** 图表主色随主题切换重绘 */
function applyChartColor() {
  const { r, g, b } = chartColorRgb.value;
  updateTrendChart(opts => {
    opts.series[0].lineStyle.color = chartColor.value;
    opts.series[0].itemStyle.color = chartColor.value;
    const gradient = opts.series[0].areaStyle?.color as { colorStops?: { color: string }[] } | undefined;
    if (gradient?.colorStops) {
      gradient.colorStops[0].color = `rgba(${r},${g},${b},0.15)`;
      gradient.colorStops[1].color = `rgba(${r},${g},${b},0)`;
    }
    return opts;
  });
}

watch(chartColorRgb, () => {
  if (trend.value.length) applyChartColor();
});

function getTypeLabel(type: ResourceApplication['resourceType']): string {
  const map: Record<ResourceApplication['resourceType'], string> = {
    model: $t('page.home.identity.typeModel'),
    mcp: $t('page.home.identity.typeMcp'),
    skill: $t('page.home.identity.typeSkill'),
    agent: $t('page.home.identity.typeAgent')
  };
  return map[type];
}

function getStatusLabel(status: ResourceApplication['status']): string {
  if (status === 'pending') return $t('page.home.identity.statusPending');
  if (status === 'approved') return $t('page.home.identity.statusApproved');
  return $t('page.home.identity.statusRejected');
}

function getStatusTagType(status: ResourceApplication['status']): 'warning' | 'success' | 'error' {
  if (status === 'approved') return 'success';
  if (status === 'rejected') return 'error';
  return 'warning';
}

function getStatusIcon(status: ResourceApplication['status']): string {
  if (status === 'pending') return 'lucide:clock';
  if (status === 'approved') return 'lucide:circle-check';
  return 'lucide:circle-x';
}

function handleCopy() {
  if (!fullKey.value) return;
  copyText(fullKey.value);
}

onMounted(async () => {
  // MOCK：模拟异步加载；后端就绪后替换为 Promise.all([...真实接口])
  await new Promise(resolve => {
    setTimeout(resolve, 300);
  });
  mainKey.value = MOCK_KEY;
  kpi.value = MOCK_KPI;
  trend.value = MOCK_TREND;
  applications.value = MOCK_APPLICATIONS;
  mcpNames.value = MOCK_MCP_NAMES;
  skillNames.value = MOCK_SKILL_NAMES;
  isLoading.value = false;

  if (trend.value.length) {
    updateTrendChart(opts => {
      opts.xAxis.data = trend.value.map(item => item.period);
      opts.series[0].data = trend.value.map(item => item.cost);
      return opts;
    });
    applyChartColor();
  }
});
</script>

<template>
  <div
    class="home-page min-h-screen"
  >
    <!-- 自定义 header（不使用 global-header）-->
    <header
      class="sticky top-0 z-50 border-b border-white/50 bg-white/70 backdrop-blur-xl dark:border-slate-700/50 dark:bg-slate-900/70"
    >
      <div class="relative flex h-60px items-center px-24px">
        <!-- 左：项目图标 + 我的主页 -->
        <div class="flex items-center gap-10px">
          <SystemLogo class="size-36px" />
          <span class="text-18px font-bold text-slate-900 dark:text-slate-100">{{ $t('route.home') }}</span>
        </div>

        <!-- 中：Tab 切换（我的应用 / 我的AI身份）-->
        <nav class="absolute left-1/2 flex -translate-x-1/2 items-center gap-4px">
          <button
            v-for="tab in homeTabs"
            :key="tab.key"
            class="rounded-8px px-16px py-8px text-14px font-medium transition-colors"
            :class="
              activeTab === tab.key
                ? 'home-accent-box'
                : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900 dark:text-slate-300 dark:hover:bg-slate-700/60'
            "
            @click="activeTab = tab.key"
          >
            {{ $t(tab.i18nKey) }}
          </button>
        </nav>

        <!-- 右：头像 + 昵称 + 下拉 -->
        <div class="ml-auto">
          <NDropdown placement="bottom-end" trigger="click" :options="dropdownOptions" @select="handleUserMenu">
            <div
              class="flex cursor-pointer items-center gap-8px rounded-8px px-8px py-4px transition-colors hover:bg-slate-100 dark:hover:bg-slate-700/60"
            >
              <NAvatar :size="32" round :src="user?.avatar || defaultAvatar" />
              <span class="max-w-120px truncate text-14px font-medium text-slate-700 dark:text-slate-200">
                {{ user?.nickName || user?.userName }}
              </span>
              <SvgIcon icon="lucide:chevron-down" class="text-16px text-slate-400" />
            </div>
          </NDropdown>
        </div>
      </div>
    </header>

    <!-- 内容区 -->
    <main class="mx-auto max-w-5xl px-16px pb-24px pt-16px">
      <div class="flex flex-col gap-16px">
        <!-- 加载骨架 -->
        <template v-if="isLoading">
          <div class="h-280px animate-pulse rounded-24px bg-slate-200/60 dark:bg-slate-700/40" />
          <div class="h-160px animate-pulse rounded-16px bg-slate-200/60 dark:bg-slate-700/40" />
          <div class="h-280px animate-pulse rounded-16px bg-slate-200/60 dark:bg-slate-700/40" />
        </template>

        <template v-else>
          <!-- 我的应用：按权限展示可访问的业务模块 -->
          <section v-show="activeTab === 'apps'" class="flex flex-col gap-12px">
            <div class="flex items-center gap-8px px-4px">
              <SvgIcon icon="mdi:apps" class="home-accent text-20px" />
              <span class="text-16px font-bold text-slate-900 dark:text-slate-100">
                {{ $t('page.home.myApps.title') }}
              </span>
            </div>
            <div v-if="myApps.length" class="grid grid-cols-2 gap-12px sm:grid-cols-3 lg:grid-cols-4">
              <div
                v-for="m in myApps"
                :key="m"
                class="home-app-card group flex cursor-pointer items-center gap-12px rounded-16px border border-slate-200/60 bg-white/80 p-16px shadow-sm backdrop-blur-xl transition-all hover:-translate-y-1px hover:shadow-md dark:border-slate-700/60 dark:bg-slate-800/60"
                @click="handleOpenApp(m)"
              >
                <div
                  class="home-app-icon flex size-40px shrink-0 items-center justify-center rounded-12px text-24px transition-colors"
                >
                  <SvgIcon :icon="MODULE_CONFIG[m].icon" />
                </div>
                <span class="min-w-0 flex-1 truncate text-14px font-semibold text-slate-900 dark:text-slate-100">
                  {{ $t(`module.${m}`) }}
                </span>
                <SvgIcon
                  icon="lucide:arrow-right"
                  class="home-app-arrow shrink-0 text-16px text-slate-300 transition-colors"
                />
              </div>
            </div>
            <p
              v-else
              class="rounded-16px bg-white/60 px-16px py-24px text-center text-14px text-slate-400 dark:bg-slate-800/40"
            >
              {{ $t('page.home.myApps.empty') }}
            </p>
          </section>

          <!-- AI 身份证卡片 -->
          <section
            v-show="activeTab === 'identity'"
            class="rounded-24px border border-slate-200/60 bg-white/80 p-10px shadow-md backdrop-blur-xl dark:border-slate-700/60 dark:bg-slate-800/60"
          >
            <div
              class="home-card-inner relative min-h-280px overflow-hidden rounded-18px"
            >
              <!-- 右侧品牌渐变区 -->
              <div
                class="home-brand-gradient absolute right-0 top-0 h-full w-30%"
                style="clip-path: polygon(30% 0, 100% 0, 100% 100%, 0 100%)"
              >
                <div
                  class="absolute right-24px top-48px select-none text-120px leading-none font-extrabold text-white/10"
                >
                  AI
                </div>
              </div>

              <!-- 内容区 -->
              <div class="relative z-10 flex w-72% flex-col p-24px">
                <!-- 顶部 logo + 状态 -->
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-10px">
                    <div
                      class="home-brand-gradient flex h-36px w-36px items-center justify-center rounded-12px text-14px font-bold text-white"
                    >
                      AI
                    </div>
                    <div>
                      <h3 class="text-14px font-extrabold text-slate-900 dark:text-slate-100">
                        {{ $t('page.home.identity.cardTitle') }}
                      </h3>
                      <p class="text-10px tracking-1.5px text-slate-400">ACCESS PASS</p>
                    </div>
                  </div>
                  <NTag
                    v-if="mainKey"
                    :type="mainKey.isActive ? 'primary' : 'default'"
                    size="small"
                    round
                    :bordered="false"
                  >
                    {{ mainKey.isActive ? $t('page.home.identity.active') : $t('page.home.identity.inactive') }}
                  </NTag>
                </div>

                <!-- 用户信息 -->
                <div class="mt-20px">
                  <h1 class="text-20px leading-none font-extrabold text-slate-900 dark:text-slate-100">
                    {{ user?.nickName || user?.userName || '-' }}
                  </h1>
                  <p class="mt-4px text-14px text-slate-400">{{ user?.email || '-' }}</p>
                  <p v-if="user?.deptName" class="mt-4px text-12px text-slate-500">{{ user.deptName }}</p>
                </div>

                <!-- API Key 区 -->
                <div
                  v-if="mainKey"
                  class="home-accent-border mt-20px rounded-16px border bg-white/80 px-20px py-16px backdrop-blur-10px dark:bg-slate-900/60"
                >
                  <div class="flex items-center justify-between gap-12px">
                    <div class="min-w-0 flex-1">
                      <div class="home-accent text-11px tracking-1px font-bold">
                        {{ $t('page.home.identity.apiKeyLabel') }}
                      </div>
                      <code class="mt-6px block break-all text-14px font-bold text-slate-900 dark:text-slate-100">
                        {{ displayKey }}
                      </code>
                    </div>
                    <div class="flex shrink-0 gap-6px">
                      <NButton size="small" quaternary circle @click="showFullKey = !showFullKey">
                        <template #icon>
                          <SvgIcon :icon="showFullKey ? 'lucide:eye-off' : 'lucide:eye'" />
                        </template>
                      </NButton>
                      <NButton size="small" :type="copied ? 'success' : 'primary'" ghost @click="handleCopy">
                        <template #icon>
                          <SvgIcon :icon="copied ? 'lucide:check' : 'lucide:copy'" />
                        </template>
                        {{ copied ? $t('page.home.identity.copied') : $t('page.home.identity.copy') }}
                      </NButton>
                    </div>
                  </div>

                  <!-- 底部 meta -->
                  <div class="mt-12px flex gap-24px border-t border-slate-100 pt-12px dark:border-slate-700">
                    <div>
                      <div class="text-10px tracking-1px text-slate-400">
                        {{ $t('page.home.identity.budgetLabel') }}
                      </div>
                      <div class="mt-2px text-12px font-bold text-slate-900 dark:text-slate-100">
                        {{ budgetDisplay }}
                      </div>
                    </div>
                    <div>
                      <div class="text-10px tracking-1px text-slate-400">
                        {{ $t('page.home.identity.modelsLabel') }}
                      </div>
                      <div class="mt-2px text-12px font-bold text-slate-900 dark:text-slate-100">
                        {{ mainKey.models.length }}
                      </div>
                    </div>
                    <div>
                      <div class="text-10px tracking-1px text-slate-400">{{ $t('page.home.identity.mcpLabel') }}</div>
                      <div class="mt-2px text-12px font-bold text-slate-900 dark:text-slate-100">
                        {{ mainKey.mcps.length }}
                      </div>
                    </div>
                    <div>
                      <div class="text-10px tracking-1px text-slate-400">{{ $t('page.home.identity.skillLabel') }}</div>
                      <div class="mt-2px text-12px font-bold text-slate-900 dark:text-slate-100">
                        {{ mainKey.skills.length }}
                      </div>
                    </div>
                  </div>
                </div>
                <div v-else class="mt-20px text-center text-14px text-slate-400">
                  {{ $t('page.home.identity.noIdentity') }}
                </div>
              </div>
            </div>
          </section>

          <!-- 我的资源 -->
          <section v-if="mainKey" v-show="activeTab === 'identity'" class="flex flex-col gap-12px">
            <NCard :bordered="false" size="small" class="card-wrapper shadow-md">
              <div class="mb-12px flex items-center gap-8px">
                <SvgIcon icon="lucide:cpu" class="home-accent text-16px" />
                <span class="text-14px font-medium">{{ $t('page.home.identity.resModel') }}</span>
                <span class="ml-auto text-12px text-slate-400">
                  {{ $t('page.home.identity.resCount', { count: mainKey.models.length }) }}
                </span>
              </div>
              <div v-if="mainKey.models.length" class="flex flex-wrap gap-8px">
                <NTag v-for="model in mainKey.models" :key="model" type="primary" size="small" round :bordered="false">
                  {{ model }}
                </NTag>
              </div>
              <p v-else class="text-12px text-slate-400">{{ $t('page.home.identity.resEmptyModels') }}</p>
            </NCard>

            <NCard :bordered="false" size="small" class="card-wrapper shadow-md">
              <div class="mb-12px flex items-center gap-8px">
                <SvgIcon icon="lucide:server" class="text-16px text-emerald-600" />
                <span class="text-14px font-medium">{{ $t('page.home.identity.resMcp') }}</span>
                <span class="ml-auto text-12px text-slate-400">
                  {{ $t('page.home.identity.resCount', { count: mainKey.mcps.length }) }}
                </span>
              </div>
              <div v-if="mainKey.mcps.length" class="flex flex-wrap gap-8px">
                <NTag v-for="id in mainKey.mcps" :key="id" type="success" size="small" round :bordered="false">
                  {{ mcpNames[id] || `#${id}` }}
                </NTag>
              </div>
              <p v-else class="text-12px text-slate-400">{{ $t('page.home.identity.resEmptyMarket') }}</p>
            </NCard>

            <NCard :bordered="false" size="small" class="card-wrapper shadow-md">
              <div class="mb-12px flex items-center gap-8px">
                <SvgIcon icon="lucide:sparkles" class="text-16px text-amber-500" />
                <span class="text-14px font-medium">{{ $t('page.home.identity.resSkill') }}</span>
                <span class="ml-auto text-12px text-slate-400">
                  {{ $t('page.home.identity.resCount', { count: mainKey.skills.length }) }}
                </span>
              </div>
              <div v-if="mainKey.skills.length" class="flex flex-wrap gap-8px">
                <NTag v-for="id in mainKey.skills" :key="id" type="warning" size="small" round :bordered="false">
                  {{ skillNames[id] || `#${id}` }}
                </NTag>
              </div>
              <p v-else class="text-12px text-slate-400">{{ $t('page.home.identity.resEmptyMarket') }}</p>
            </NCard>
          </section>

          <!-- 用量概览 -->
          <NCard
            v-if="kpi"
            v-show="activeTab === 'identity'"
            :bordered="false"
            size="small"
            class="card-wrapper shadow-md"
            :title="$t('page.home.identity.overviewTitle')"
          >
            <div class="grid grid-cols-2 gap-12px sm:grid-cols-4">
              <div class="rounded-12px bg-slate-50 px-16px py-12px dark:bg-slate-700/40">
                <div class="text-12px text-slate-400">{{ $t('page.home.identity.overviewMonthBudget') }}</div>
                <div class="mt-4px text-18px font-semibold text-slate-900 dark:text-slate-100">{{ budgetDisplay }}</div>
              </div>
              <div class="rounded-12px bg-slate-50 px-16px py-12px dark:bg-slate-700/40">
                <div class="text-12px text-slate-400">{{ $t('page.home.identity.overviewSpent') }}</div>
                <div class="mt-4px text-18px font-semibold text-slate-900 dark:text-slate-100">
                  ¥{{ (kpi.totalCost ?? 0).toFixed(2) }}
                </div>
                <div v-if="budgetUsedPercent !== null" class="mt-2px text-12px text-slate-400">
                  {{ budgetUsedPercent.toFixed(1) }}%
                </div>
              </div>
              <div class="rounded-12px bg-slate-50 px-16px py-12px dark:bg-slate-700/40">
                <div class="text-12px text-slate-400">{{ $t('page.home.identity.overviewRequests') }}</div>
                <div class="mt-4px text-18px font-semibold text-slate-900 dark:text-slate-100">
                  {{ (kpi.totalRequests ?? 0).toLocaleString() }}
                </div>
              </div>
              <div class="rounded-12px bg-slate-50 px-16px py-12px dark:bg-slate-700/40">
                <div class="text-12px text-slate-400">{{ $t('page.home.identity.overviewDailyAvg') }}</div>
                <div class="mt-4px text-18px font-semibold text-slate-900 dark:text-slate-100">
                  ¥{{ dailyAvgCost.toFixed(2) }}
                </div>
              </div>
            </div>

            <!-- 预算进度条 -->
            <div v-if="budgetUsedPercent !== null" class="mt-16px">
              <div class="flex items-center justify-between text-12px text-slate-400">
                <span>{{ $t('page.home.identity.overviewBudgetUsage') }}</span>
                <span>{{ budgetUsedPercent.toFixed(1) }}%</span>
              </div>
              <div class="mt-6px h-8px overflow-hidden rounded-full bg-slate-100 dark:bg-slate-700">
                <div
                  class="h-full rounded-full transition-all"
                  :class="
                    budgetUsedPercent > 100 ? 'bg-red-500' : budgetUsedPercent > 80 ? 'bg-amber-500' : 'home-progress-accent'
                  "
                  :style="{ width: `${Math.min(budgetUsedPercent, 100)}%` }"
                />
              </div>
            </div>

            <!-- 趋势图表 -->
            <div v-if="trend.length" class="mt-20px">
              <div ref="trendChartRef" class="h-200px w-full" />
            </div>
          </NCard>

          <!-- 我的申请 -->
          <NCard
            v-if="applications.length"
            v-show="activeTab === 'identity'"
            :bordered="false"
            size="small"
            class="card-wrapper shadow-md"
            :title="$t('page.home.identity.appsTitle')"
          >
            <div class="flex flex-col gap-8px">
              <div
                v-for="app in applications"
                :key="app.id"
                class="flex items-center gap-12px rounded-8px bg-slate-50 px-16px py-10px dark:bg-slate-700/40"
              >
                <SvgIcon
                  :icon="getStatusIcon(app.status)"
                  class="shrink-0 text-16px"
                  :class="
                    app.status === 'pending'
                      ? 'text-amber-500'
                      : app.status === 'approved'
                        ? 'text-green-500'
                        : 'text-red-400'
                  "
                />
                <NTag size="small" :bordered="false">{{ getTypeLabel(app.resourceType) }}</NTag>
                <span class="min-w-0 flex-1 truncate text-14px">{{ app.resourceName || `#${app.resourceId}` }}</span>
                <NTag size="small" round :bordered="false" :type="getStatusTagType(app.status)">
                  {{ getStatusLabel(app.status) }}
                </NTag>
              </div>
            </div>
          </NCard>
        </template>
      </div>
    </main>
  </div>
</template>

<style scoped>
/* home 色调:全部经主题 CSS 变量派生,自动跟随 themeColor 与 dark/light 切换(dark 下色阶由主题注入) */
.home-page {
  background-color: rgb(var(--layout-bg-color));
}
.home-card-inner {
  background-image: linear-gradient(to bottom right, rgb(var(--container-bg-color)), rgb(var(--primary-600-color) / 0.08));
}
.home-accent {
  color: rgb(var(--primary-600-color));
}
.home-accent-box {
  background-color: rgb(var(--primary-600-color) / 0.10);
  color: rgb(var(--primary-600-color));
}
.home-accent-border {
  border-color: rgb(var(--primary-600-color) / 0.12);
}
.home-brand-gradient {
  background-image: linear-gradient(to bottom right, rgb(var(--primary-600-color)), rgb(var(--primary-700-color)));
}
.home-app-icon {
  background-color: rgb(var(--primary-600-color) / 0.10);
  color: rgb(var(--primary-600-color));
}
.home-app-card:hover {
  border-color: rgb(var(--primary-600-color) / 0.40);
}
.home-app-card:hover .home-app-icon {
  background-color: rgb(var(--primary-600-color) / 0.18);
}
.home-app-card:hover .home-app-arrow {
  color: rgb(var(--primary-600-color));
}
.home-progress-accent {
  background-color: rgb(var(--primary-500-color));
}
</style>
