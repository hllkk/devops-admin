<script setup lang="ts">
import { computed, defineAsyncComponent, onMounted } from 'vue';
import { AdminLayout, LAYOUT_SCROLL_EL_ID } from '@sa/materials';
import type { LayoutMode } from '@sa/materials';
import { useAppStore } from '@/store/modules/app';
import { useThemeStore } from '@/store/modules/theme';
import GlobalHeader from '../modules/global-header/index.vue';
import GlobalSider from '../modules/global-sider/index.vue';
import GlobalTab from '../modules/global-tab/index.vue';
import GlobalContent from '../modules/global-content/index.vue';
import GlobalFooter from '../modules/global-footer/index.vue';
import ThemeDrawer from '../modules/theme-drawer/index.vue';
import { provideMixMenuContext } from '../modules/global-menu/context';
import { getServiceBaseURL } from '@/utils/service';
import { initWebSocket } from '@/utils/websocket';
import { initSSE } from '@/utils/sse';

defineOptions({
  name: 'BaseLayout'
});

const appStore = useAppStore();
const themeStore = useThemeStore();
const { secondLevelMenus, childLevelMenus, isActiveFirstLevelMenuHasChildren } = provideMixMenuContext();

interface Props {
  /** 隐藏标签页 */
  hideTab?: boolean;
  /** 隐藏 header 主题设置齿轮(明暗切换仍保留) */
  hideThemeControls?: boolean;
  /** 不挂载主题抽屉(彻底无主题设置入口) */
  hideThemeDrawer?: boolean;
}

const props = withDefaults(defineProps<Props>(), {
  hideTab: false,
  hideThemeControls: false,
  hideThemeDrawer: false
});

/** 实际菜单模式:disk 模块定死 vertical-mix(themeStore.effectiveLayoutMode),其余用用户设置 */
const effectiveMode = computed(() => themeStore.effectiveLayoutMode);

const GlobalMenu = defineAsyncComponent(() => import('../modules/global-menu/index.vue'));

const layoutMode = computed(() => {
  const vertical: LayoutMode = 'vertical';
  const horizontal: LayoutMode = 'horizontal';
  return effectiveMode.value.includes(vertical) ? vertical : horizontal;
});

const headerProps = computed(() => {
  const mode = effectiveMode.value;

  const headerPropsConfig: Record<UnionKey.ThemeLayoutMode, App.Global.HeaderProps> = {
    vertical: {
      showLogo: false,
      showMenu: false,
      showMenuToggler: true
    },
    'vertical-mix': {
      showLogo: false,
      showMenu: false,
      showMenuToggler: false
    },
    'vertical-hybrid-header-first': {
      showLogo: !isActiveFirstLevelMenuHasChildren.value,
      showMenu: true,
      showMenuToggler: false
    },
    horizontal: {
      showLogo: true,
      showMenu: true,
      showMenuToggler: false
    },
    'top-hybrid-sidebar-first': {
      showLogo: true,
      showMenu: true,
      showMenuToggler: false
    },
    'top-hybrid-header-first': {
      showLogo: true,
      showMenu: true,
      showMenuToggler: isActiveFirstLevelMenuHasChildren.value
    }
  };

  return headerPropsConfig[mode];
});

const siderVisible = computed(() => effectiveMode.value !== 'horizontal');

const isVerticalMix = computed(() => effectiveMode.value === 'vertical-mix');

const isVerticalHybridHeaderFirst = computed(() => effectiveMode.value === 'vertical-hybrid-header-first');

const isTopHybridSidebarFirst = computed(() => effectiveMode.value === 'top-hybrid-sidebar-first');

const isTopHybridHeaderFirst = computed(() => effectiveMode.value === 'top-hybrid-header-first');

const siderWidth = computed(() => getSiderAndCollapsedWidth(false));

const siderCollapsedWidth = computed(() => getSiderAndCollapsedWidth(true));

function getSiderAndCollapsedWidth(isCollapsed: boolean) {
  const {
    mixChildMenuWidth,
    collapsedWidth,
    width: themeWidth,
    mixCollapsedWidth,
    mixWidth: themeMixWidth
  } = themeStore.sider;

  const width = isCollapsed ? collapsedWidth : themeWidth;
  const mixWidth = isCollapsed ? mixCollapsedWidth : themeMixWidth;

  if (isTopHybridHeaderFirst.value) {
    return isActiveFirstLevelMenuHasChildren.value ? width : 0;
  }

  if (isVerticalHybridHeaderFirst.value && !isActiveFirstLevelMenuHasChildren.value) {
    return 0;
  }

  const isMixMode = isVerticalMix.value || isTopHybridSidebarFirst.value || isVerticalHybridHeaderFirst.value;
  let finalWidth = isMixMode ? mixWidth : width;

  if (isVerticalMix.value && appStore.mixSiderFixed && secondLevelMenus.value.length) {
    finalWidth += mixChildMenuWidth;
  }

  if (isVerticalHybridHeaderFirst.value && appStore.mixSiderFixed && childLevelMenus.value.length) {
    finalWidth += mixChildMenuWidth;
  }

  return finalWidth;
}

onMounted(() => {
  // baseURL 与普通 axios 请求同源：dev proxy 模式为 /proxy-default，构建模式为 VITE_SERVICE_BASE_URL。
  // 不要直接引用 VITE_APP_BASE_API（项目未定义该变量，会拼出 "undefined/resource/sse"）。
  const isHttpProxy = import.meta.env.DEV && import.meta.env.VITE_HTTP_PROXY === 'Y';
  const { baseURL } = getServiceBaseURL(import.meta.env, isHttpProxy);

  const protocol = window.location.protocol === 'https:' ? 'wss://' : 'ws://';
  initWebSocket(`${protocol}${window.location.host}${baseURL}/resource/websocket`);
  initSSE(`${baseURL}/resource/sse`);
});
</script>

<template>
  <AdminLayout
    v-model:sider-collapse="appStore.siderCollapse"
    :mode="layoutMode"
    :scroll-el-id="LAYOUT_SCROLL_EL_ID"
    :scroll-mode="themeStore.layout.scrollMode"
    :is-mobile="appStore.isMobile"
    :full-content="appStore.fullContent"
    :fixed-top="themeStore.fixedHeaderAndTab"
    :header-height="themeStore.header.height"
    :tab-visible="!hideTab && themeStore.tab.visible"
    :tab-height="themeStore.tab.height"
    :content-class="appStore.contentXScrollable ? 'overflow-x-hidden' : ''"
    :sider-visible="siderVisible"
    :sider-width="siderWidth"
    :sider-collapsed-width="siderCollapsedWidth"
    :footer-visible="themeStore.footer.visible"
    :footer-height="themeStore.footer.height"
    :fixed-footer="themeStore.footer.fixed"
    :right-footer="themeStore.footer.right"
  >
    <template #header>
      <GlobalHeader v-bind="headerProps" :show-theme-controls="!hideThemeControls" />
    </template>
    <template #tab>
      <GlobalTab v-if="!hideTab" />
    </template>
    <template #sider>
      <GlobalSider />
    </template>
    <GlobalMenu />
    <GlobalContent />
    <ThemeDrawer v-if="!hideThemeDrawer" />
    <template #footer>
      <GlobalFooter />
    </template>
  </AdminLayout>
</template>

<style lang="scss">
#__SCROLL_EL_ID__ {
  @include scrollbar();
}
</style>
