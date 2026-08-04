<script setup lang="ts">
import { computed } from 'vue';
import type { Component } from 'vue';
import { useAppStore } from '@/store/modules/app';
import { useThemeStore } from '@/store/modules/theme';
import VerticalMenu from './modules/vertical-menu.vue';
import VerticalMixMenu from './modules/vertical-mix-menu.vue';
import VerticalHybridHeaderFirst from './modules/vertical-hybrid-header-first.vue';
import HorizontalMenu from './modules/horizontal-menu.vue';
import TopHybridSidebarFirst from './modules/top-hybrid-sidebar-first.vue';
import TopHybridHeaderFirst from './modules/top-hybrid-header-first.vue';

defineOptions({
  name: 'GlobalMenu'
});

const appStore = useAppStore();
const themeStore = useThemeStore();

const activeMenu = computed(() => {
  const menuMap: Record<UnionKey.ThemeLayoutMode, Component> = {
    vertical: VerticalMenu,
    'vertical-mix': VerticalMixMenu,
    'vertical-hybrid-header-first': VerticalHybridHeaderFirst,
    horizontal: HorizontalMenu,
    'top-hybrid-sidebar-first': TopHybridSidebarFirst,
    'top-hybrid-header-first': TopHybridHeaderFirst
  };

  return menuMap[themeStore.effectiveLayoutMode];
});

/**
 * 菜单强制重建 key：所有菜单均经 <Teleport> 挂到 #GLOBAL_SIDER_MENU_ID / #GLOBAL_HEADER_MENU_ID。
 * is-mobile 切换会让 AdminLayout 重建 sider/header 的 DOM，原 target 节点销毁后以新节点重建，
 * 而 Teleport 不会自动把内容回挂到新 target —— 菜单及其顶部 logo 图标会因此永久消失
 * （典型复现：缩小到手机再放大窗口）。
 * 借 :key 随 isMobile 翻转强制重建菜单组件，使其 Teleport 在 target 恢复后重新解析挂载。
 * 原仅 vertical 模式有此补丁，vertical-mix 等其余 Teleport 菜单遗漏，故放大后菜单消失。
 */
const reRenderMenu = computed(() => appStore.isMobile);
</script>

<template>
  <component :is="activeMenu" :key="reRenderMenu" />
</template>

<style scoped></style>
