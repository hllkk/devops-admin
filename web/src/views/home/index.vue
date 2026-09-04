<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
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
import { copyTextToClipboard, selectText } from '@/utils/copy';
import {
  fetchGetDashboardOverview,
  fetchGetDashboardTrend,
  fetchGetMyApplications,
  fetchGetMyIdentity
} from '@/service/api/gateway';
import SvgIcon from '@/components/custom/svg-icon.vue';
import SystemLogo from '@/components/common/system-logo.vue';
import ModelSquarePanel from './modules/model-square-panel.vue';
import defaultAvatar from '@/assets/imgs/soybean.jpg';

defineOptions({ name: 'Home' });

const authStore = useAuthStore();
const user = computed(() => authStore.userInfo.user);
const { SvgIconVNode } = useSvgIcon();
const showFullKey = ref(false);
/** 主 Key 复制状态：成功才点亮"已复制"，失败弹错误提示(不假亮) */
const keyCopied = ref(false);
let keyCopiedTimer: ReturnType<typeof setTimeout> | null = null;
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

// ===== 顶部 Tab：我的应用 / 我的AI身份 / 模型广场 =====
// 「我的应用」仅对有模块权限的用户可见(apps 由后端 getUserInfo 按菜单权限聚合下发,超管=全部);
// 无任何模块权限的普通用户 Tab 收敛为 身份+广场,且打开 home 默认落在「我的AI身份」页
type HomeTab = 'apps' | 'identity' | 'square';
type HomeTabI18nKey = 'page.home.myApps.title' | 'page.home.identity.navIdentity' | 'page.home.square.tab';
const homeTabs = computed<{ key: HomeTab; i18nKey: HomeTabI18nKey }[]>(() => [
  ...(myApps.value.length ? [{ key: 'apps' as const, i18nKey: 'page.home.myApps.title' as const }] : []),
  { key: 'identity', i18nKey: 'page.home.identity.navIdentity' },
  { key: 'square', i18nKey: 'page.home.square.tab' }
]);
const activeTab = ref<HomeTab>('identity');

// 模型广场面板懒挂载：首次切到该 Tab 才加载数据（identity 与身份卡共享，不重复请求）
const squareLoaded = ref(false);
watch(activeTab, val => {
  if (val === 'square') squareLoaded.value = true;
});

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

// ============ AI 身份/用量：真实接口（P1 后端已就绪） ============
// identity/my 管理员创建制：主 Key 由管理员后台创建，未开通 opened=false(身份卡显示空态)；
// 已开通返回主 Key 明文 + 场景 Key 列表 + 可用模型；
// 用量 KPI/趋势调 dashboard scope=self；mcps/skills/申请列表为 P2 资源申请能力，P1 占位。
// identity/my 全量保存(未开通 opened=false 也带可见模型与接入点)；mainKey 仅开通后有值
const identity = ref<Api.Gateway.MyIdentity | null>(null);
const mainKey = computed(() => (identity.value?.opened ? identity.value : null));
const kpi = ref<Api.Gateway.DashboardOverview | null>(null);
const trend = ref<Api.Gateway.TrendItem[]>([]);

// ===== 可见模型(按发布可见性过滤)与主 Key 已授权集合：「我的资源」卡数据源 =====
const visibleModels = computed(() => identity.value?.availableModels ?? []);
const authorizedKeys = computed(() => new Set(mainKey.value?.models ?? []));
const authorizedCount = computed(() => visibleModels.value.filter(m => authorizedKeys.value.has(m.modelKey)).length);

/** 可见 MCP(未开通也展示)与已授权 serverName 集合 */
const visibleMcps = computed(() => identity.value?.availableMcps ?? []);
const authorizedMcpNames = computed(() => new Set(mainKey.value?.mcps ?? []));
const authorizedMcpCount = computed(() => visibleMcps.value.filter(m => authorizedMcpNames.value.has(m.serverName)).length);

/** 可见 Skill(未开通也展示)与已授权 skillId 集合(P2) */
const visibleSkills = computed(() => identity.value?.availableSkills ?? []);
const authorizedSkillIds = computed(() => new Set(mainKey.value?.skills ?? []));
const authorizedSkillCount = computed(() => visibleSkills.value.filter(s => authorizedSkillIds.value.has(String(s.skillId))).length);

const fullKey = computed(() => mainKey.value?.keyValue ?? '');
const maskedKey = computed(() => {
  const value = fullKey.value;
  if (!value) return 'sk-xxxxxxxxxxxx';
  return value.length > 12 ? `${value.slice(0, 7)}****${value.slice(-4)}` : value;
});
const displayKey = computed(() => (showFullKey.value ? fullKey.value : maskedKey.value));

const budgetDisplay = computed(() => {
  const key = mainKey.value;
  // P1 统一预算口径(budgetLimit)；多维预算(budgetScope per_type/per_resource)留 P3 成本效能
  if (!key || key.budgetLimit === null) return $t('page.home.identity.budgetUnlimited');
  return `¥${key.budgetLimit}`;
});

/** 主 Key 有效期(过期由 LiteLLM expires_at 原生拦截；null=永不过期) */
const expiresDisplay = computed(() => {
  const raw = mainKey.value?.expiresAt;
  if (!raw) return $t('page.home.identity.expiresNever');
  const d = new Date(raw);
  return Number.isNaN(d.getTime()) ? raw : d.toLocaleDateString();
});

const totalBudget = computed(() => mainKey.value?.budgetLimit ?? null);

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

// ===== 我的申请(P2 资源申请审批)：最近 10 条,审批结果由站内通知推送 =====
const applications = ref<Api.Gateway.ApplicationItem[]>([]);

const APP_STATUS_META: Record<
  Api.Gateway.ApplicationItem['status'],
  { label: 'statusPending' | 'statusApproved' | 'statusRejected'; type: 'warning' | 'success' | 'error' }
> = {
  pending: { label: 'statusPending', type: 'warning' },
  approved: { label: 'statusApproved', type: 'success' },
  rejected: { label: 'statusRejected', type: 'error' }
};

function fmtTime(raw: string) {
  const d = new Date(raw);
  return Number.isNaN(d.getTime()) ? raw : d.toLocaleString();
}

/** 「我的申请」空态入口：切到模型广场 Tab（广场已内嵌 home，不再跳独立路由） */
function goSquare() {
  activeTab.value = 'square';
}

async function handleCopy(evt?: MouseEvent) {
  if (!fullKey.value) return;
  // 同行可见 code 作选区载体(copy 事件载体)；显示掩码也不影响——写入值由 copy.ts 显式指定原文
  const code = (evt?.currentTarget as HTMLElement | null)?.parentElement?.querySelector('code');
  try {
    await copyTextToClipboard(fullKey.value, code);
    keyCopied.value = true;
    if (keyCopiedTimer) clearTimeout(keyCopiedTimer);
    keyCopiedTimer = setTimeout(() => {
      keyCopied.value = false;
    }, 2000);
  } catch {
    // 兜底：自动选中文本，引导 Ctrl+C 手动复制(不依赖剪贴板 API，任何环境可用)
    selectText(code);
    window.$message?.warning($t('common.copyFailed'));
  }
}

async function loadIdentity() {
  // identity/my 管理员创建制：未开通(opened=false)时 mainKey computed 为 null,身份卡走
  // "暂无 AI 身份"空态;可见模型照常返回,「我的资源」卡对未开通用户也展示可开通的模型
  const { data, error } = await fetchGetMyIdentity();
  if (!error && data) identity.value = data;
}

async function loadApplications() {
  const { data, error } = await fetchGetMyApplications({ pageNum: 1, pageSize: 10, params: {} });
  if (!error && data) applications.value = data.rows;
}

async function loadUsage() {
  const [overview, trendRes] = await Promise.all([
    fetchGetDashboardOverview({ scope: 'self' }),
    fetchGetDashboardTrend({ scope: 'self' })
  ]);
  if (!overview.error && overview.data) kpi.value = overview.data;
  if (!trendRes.error && trendRes.data) trend.value = trendRes.data;
}

onMounted(async () => {
  // 真实接口：身份 + 用量(KPI/趋势) 并行加载；申请列表 P2 资源申请能力，P1 占位
  await Promise.all([loadIdentity(), loadUsage(), loadApplications()]);
  isLoading.value = false;

  if (trend.value.length) {
    updateTrendChart(opts => {
      opts.xAxis.data = trend.value.map(item => item.date);
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

        <!--
 中：Tab 切换（我的应用 / 我的AI身份 / 模型广场）segmented control。
             按钮背景全部显式声明(含未选中 bg-transparent)，防浏览器 auto-dark(系统深色)给无背景元素强制填深底 
-->
        <nav
          class="absolute left-1/2 flex -translate-x-1/2 items-center rounded-10px bg-slate-100/90 p-2px max-md:static max-md:ml-16px max-md:translate-x-0 dark:bg-slate-800/80"
        >
          <button
            v-for="tab in homeTabs"
            :key="tab.key"
            class="max-w-160px truncate rounded-8px px-14px py-6px text-14px font-medium whitespace-nowrap transition-colors max-md:px-10px"
            :class="
              activeTab === tab.key
                ? 'bg-white text-slate-900 shadow-sm dark:bg-slate-700 dark:text-slate-100'
                : 'bg-transparent text-slate-500 hover:text-slate-900 dark:text-slate-400 dark:hover:text-slate-100'
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
              class="flex cursor-pointer items-center gap-8px rounded-8px bg-transparent px-8px py-4px transition-colors hover:bg-slate-100 dark:hover:bg-slate-700/60"
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
          <!-- 我的应用：按权限展示可访问的业务模块(无模块权限时 Tab 与内容一并收敛) -->
          <section v-if="myApps.length" v-show="activeTab === 'apps'" class="flex flex-col gap-12px">
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
                      <NButton size="small" :type="keyCopied ? 'success' : 'primary'" ghost @click="handleCopy($event)">
                        <template #icon>
                          <SvgIcon :icon="keyCopied ? 'lucide:check' : 'lucide:copy'" />
                        </template>
                        {{ keyCopied ? $t('page.home.identity.copied') : $t('page.home.identity.copy') }}
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
                      <div class="text-10px tracking-1px text-slate-400">
                        {{ $t('page.home.identity.expiresLabel') }}
                      </div>
                      <div class="mt-2px text-12px font-bold text-slate-900 dark:text-slate-100">
                        {{ expiresDisplay }}
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

          <!-- 我的资源(可见模型不需要已开通：未开通用户也可见可开通的模型) -->
          <section v-show="activeTab === 'identity'" class="flex flex-col gap-12px">
            <NCard :bordered="false" size="small" class="card-wrapper shadow-md">
              <div class="mb-12px flex items-center gap-8px">
                <SvgIcon icon="lucide:cpu" class="home-accent text-16px" />
                <span class="text-14px font-medium">{{ $t('page.home.identity.resModel') }}</span>
                <span class="ml-auto text-12px text-slate-400">
                  {{ $t('page.home.identity.resCount', { authorized: authorizedCount, visible: visibleModels.length }) }}
                </span>
              </div>
              <div v-if="visibleModels.length" class="flex flex-wrap gap-8px">
                <NTooltip v-for="model in visibleModels" :key="model.modelId" trigger="hover">
                  <template #trigger>
                    <NTag
                      :type="authorizedKeys.has(model.modelKey) ? 'primary' : 'default'"
                      size="small"
                      round
                      :bordered="authorizedKeys.has(model.modelKey)"
                    >
                      {{ model.name }}
                    </NTag>
                  </template>
                  {{
                    authorizedKeys.has(model.modelKey)
                      ? model.modelKey
                      : model.requiresApproval
                        ? $t('page.home.identity.resApproval')
                        : $t('page.home.identity.resNotAuthorized')
                  }}
                </NTooltip>
              </div>
              <p v-else class="text-12px text-slate-400">{{ $t('page.home.identity.resEmptyModels') }}</p>
            </NCard>

            <NCard :bordered="false" size="small" class="card-wrapper shadow-md">
              <div class="mb-12px flex items-center gap-8px">
                <SvgIcon icon="lucide:server" class="text-16px text-emerald-600" />
                <span class="text-14px font-medium">{{ $t('page.home.identity.resMcp') }}</span>
                <span class="ml-auto text-12px text-slate-400">
                  {{ $t('page.home.identity.resCount', { authorized: authorizedMcpCount, visible: visibleMcps.length }) }}
                </span>
              </div>
              <div v-if="visibleMcps.length" class="flex flex-wrap gap-8px">
                <NTooltip v-for="mcp in visibleMcps" :key="mcp.mcpServerId" trigger="hover">
                  <template #trigger>
                    <NTag
                      :type="authorizedMcpNames.has(mcp.serverName) ? 'primary' : 'default'"
                      size="small"
                      round
                      :bordered="authorizedMcpNames.has(mcp.serverName)"
                    >
                      {{ mcp.name }}
                    </NTag>
                  </template>
                  {{ authorizedMcpNames.has(mcp.serverName) ? mcp.serverName : $t('page.home.identity.resApproval') }}
                </NTooltip>
              </div>
              <p v-else class="text-12px text-slate-400">{{ $t('page.home.identity.resEmptyMarket') }}</p>
            </NCard>

            <NCard :bordered="false" size="small" class="card-wrapper shadow-md">
              <div class="mb-12px flex items-center gap-8px">
                <SvgIcon icon="lucide:sparkles" class="text-16px text-amber-500" />
                <span class="text-14px font-medium">{{ $t('page.home.identity.resSkill') }}</span>
                <span class="ml-auto text-12px text-slate-400">
                  {{ $t('page.home.identity.resCount', { authorized: authorizedSkillCount, visible: visibleSkills.length }) }}
                </span>
              </div>
              <div v-if="visibleSkills.length" class="flex flex-wrap gap-8px">
                <NTooltip v-for="skill in visibleSkills" :key="skill.skillId" trigger="hover">
                  <template #trigger>
                    <NTag
                      :type="authorizedSkillIds.has(String(skill.skillId)) ? 'primary' : 'default'"
                      size="small"
                      round
                      :bordered="authorizedSkillIds.has(String(skill.skillId))"
                    >
                      {{ skill.name }}
                    </NTag>
                  </template>
                  {{
                    authorizedSkillIds.has(String(skill.skillId))
                      ? `v${skill.version}`
                      : skill.requiresApproval
                        ? $t('page.home.identity.resApproval')
                        : $t('page.home.identity.resNotAuthorized')
                  }}
                </NTooltip>
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

          <!-- 我的申请(P2 资源申请审批)：最近 10 条,展示资源/类型/状态/申请时间/审批意见 -->
          <NCard
            v-show="activeTab === 'identity'"
            :bordered="false"
            size="small"
            class="card-wrapper shadow-md"
            :title="$t('page.home.identity.appsTitle')"
          >
            <div v-if="applications.length" class="flex flex-col gap-8px">
              <div
                v-for="app in applications"
                :key="app.applicationId"
                class="flex flex-wrap items-center gap-8px rounded-10px bg-slate-50 px-12px py-8px dark:bg-slate-700/40"
              >
                <span class="text-13px font-medium">{{ app.resourceName || app.resourceKey }}</span>
                <NTag size="tiny" :bordered="false">{{ $t('page.home.identity.typeModel') }}</NTag>
                <NTag size="tiny" :bordered="false" :type="APP_STATUS_META[app.status].type">
                  {{ $t(`page.home.identity.${APP_STATUS_META[app.status].label}`) }}
                </NTag>
                <span class="ml-auto text-12px text-slate-400">
                  {{ $t('page.home.identity.appsTime') }} {{ fmtTime(app.createTime) }}
                </span>
                <p v-if="app.reviewNotes" class="w-full text-12px text-slate-400">
                  {{ $t('page.home.identity.appsNotes') }}：{{ app.reviewNotes }}
                </p>
              </div>
            </div>
            <div v-else class="flex flex-col items-center gap-8px px-16px py-20px">
              <p class="text-center text-14px text-slate-400">{{ $t('page.home.identity.appsEmpty') }}</p>
              <NButton size="small" type="primary" ghost @click="goSquare">
                {{ $t('page.home.identity.goSquare') }}
              </NButton>
            </div>
          </NCard>

          <!-- 模型广场：可见模型浏览 + 接入信息 + 申请订阅（懒挂载） -->
          <section v-show="activeTab === 'square'" class="flex flex-col gap-12px">
            <ModelSquarePanel v-if="squareLoaded" :identity="identity" @applied="loadApplications" />
          </section>
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
