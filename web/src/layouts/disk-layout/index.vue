<script setup lang="ts">
import BaseLayout from '../base-layout/index.vue';

defineOptions({
  name: 'DiskLayout'
});
</script>

<template>
  <!--
    网盘布局:复用 base 外壳,无 tab、顶部无主题设置齿轮(明暗切换保留)。
    菜单模式定死 vertical-mix 由 themeStore.effectiveLayoutMode 统一处理(disk→vertical-mix),
    base-layout/global-menu/global-sider 都读它,保证与 admin 的 vertical-mix 渲染一致。
    admin/server/gateway 仍用 layout.base(不传 prop,行为不变)。
  -->
  <BaseLayout hide-tab hide-theme-controls hide-theme-drawer hide-footer />
</template>
