<script setup lang="ts">
import { computed, h } from 'vue';
import { useRouterPush } from '@/hooks/common/router';
import type { LastLevelRouteKey } from '@elegant-router/types';
import type { DropdownOption } from 'naive-ui';
import { ALL_MODULES, MODULE_CONFIG, type RouteModule } from '@/constants/module';
import { useRouteStore } from '@/store/modules/route';
import { useTabStore } from '@/store/modules/tab';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'ModuleSelect' });

const { routerPushByKey } = useRouterPush();
const routeStore = useRouteStore();
const tabStore = useTabStore();

/** 当前模块配置（用于触发按钮展示图标+名称） */
const currentConfig = computed(() => MODULE_CONFIG[routeStore.currentModule]);

/** 下拉选项：全部模块，当前模块 disabled */
const options = computed<DropdownOption[]>(() =>
  ALL_MODULES.map(m => ({
    label: $t(`module.${m}`),
    key: m,
    icon: () => h(SvgIcon, { icon: MODULE_CONFIG[m].icon }),
    disabled: m === routeStore.currentModule
  }))
);

/** 点击选项 → 清空旧模块 Tab → 重建 homeTab → 导航到新模块首页 */
function handleSelect(key: string) {
  if (key === routeStore.currentModule) return;
  const module = key as RouteModule;
  const homeRoute = MODULE_CONFIG[module].home as LastLevelRouteKey;
  tabStore.resetTabs(homeRoute);
  routerPushByKey(homeRoute);
}
</script>

<template>
  <NDropdown :options="options" trigger="click" @select="handleSelect">
    <NButton quaternary :focusable="false" class="h-36px text-icon">
      <div class="flex-center gap-8px">
        <SvgIcon :icon="currentConfig.icon" />
        <span class="text-14px">{{ $t(`module.${routeStore.currentModule}`) }}</span>
      </div>
    </NButton>
  </NDropdown>
</template>

<style scoped></style>
