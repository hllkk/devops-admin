<script setup lang="ts">
import { computed } from 'vue';
import { useRouteStore } from '@/store/modules/route';
import BaseLayout from '../base-layout/index.vue';
import DiskLayout from '../disk-layout/index.vue';

defineOptions({
  name: 'AutoLayout'
});

/**
 * 全局公共页(个人中心 / 消息中心等,无 meta.module)的布局外壳。
 *
 * 跟随 currentModule:disk 模块 → DiskLayout(无 tab / 无主题控件 + vertical-mix),其余 → BaseLayout。
 * currentModule 由 route store sticky 维护(进入全局页时保持来源模块),故公共页外壳与来源模块一致。
 * 不退化到 blank——公共页是需要 header/sider 的功能页,blank 会让宽布局挤变形。
 */
const routeStore = useRouteStore();
const layout = computed(() => (routeStore.currentModule === 'disk' ? DiskLayout : BaseLayout));
</script>

<template>
  <component :is="layout" />
</template>
