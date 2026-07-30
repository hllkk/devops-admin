<script setup lang="ts">
import { computed } from 'vue';
import { useRouterPush } from '@/hooks/common/router';
import type { LastLevelRouteKey } from '@elegant-router/types';
import type { DropdownOption } from 'naive-ui';
import { ALL_MODULES, MODULE_CONFIG, type RouteModule } from '@/constants/module';
import { useAuthStore } from '@/store/modules/auth';
import { useRouteStore } from '@/store/modules/route';
import { useTabStore } from '@/store/modules/tab';
import { useSvgIcon } from '@/hooks/common/icon';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'ModuleSelect' });

const { routerPushByKey } = useRouterPush();
const { SvgIconVNode } = useSvgIcon();
const authStore = useAuthStore();
const routeStore = useRouteStore();
const tabStore = useTabStore();

/** 当前模块配置（用于触发按钮展示图标+名称） */
const currentConfig = computed(() => MODULE_CONFIG[routeStore.currentModule]);

/**
 * 可见模块：数据源为后端 getUserInfo.apps（按模块菜单权限聚合下发），与首页"我的应用"同源。
 * 不再遍历前端 ALL_MODULES,确保无权限模块(如纯 user 角色的后台管理)不出现;并兜底过滤非法/历史脏值。
 */
const visibleApps = computed<RouteModule[]>(() =>
  (authStore.userInfo.apps ?? []).filter((m): m is RouteModule => ALL_MODULES.includes(m as RouteModule))
);

/** 下拉选项：仅可见模块，当前模块 disabled */
const options = computed<DropdownOption[]>(() =>
  visibleApps.value.map(m => ({
    label: $t(`module.${m}`),
    key: m,
    icon: SvgIconVNode({ icon: MODULE_CONFIG[m].icon }),
    disabled: m === routeStore.currentModule
  }))
);

/** 点击选项 → 清空旧模块 Tab → 重建 homeTab → 导航到新模块首页 */
function handleSelect(key: string) {
  if (key === routeStore.currentModule) return;
  const module = key as RouteModule;
  // 兜底:仅允许跳转至已下发权限的模块,防脏数据/手动构造请求绕过
  if (!(authStore.userInfo.apps ?? []).includes(module)) return;
  const homeRoute = MODULE_CONFIG[module].home as LastLevelRouteKey;
  tabStore.resetTabs(homeRoute);
  routerPushByKey(homeRoute);
}
</script>

<template>
  <NDropdown v-if="visibleApps.length" :options="options" trigger="click" @select="handleSelect">
    <NButton quaternary :focusable="false" class="h-36px text-icon">
      <div class="flex-center gap-8px">
        <SvgIcon :icon="currentConfig.icon" />
        <span class="text-14px">{{ $t(`module.${routeStore.currentModule}`) }}</span>
      </div>
    </NButton>
  </NDropdown>
</template>

<style scoped></style>
