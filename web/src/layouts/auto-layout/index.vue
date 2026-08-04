<script setup lang="ts">
import BaseLayout from '../base-layout/index.vue';

defineOptions({
  name: 'AutoLayout'
});

/**
 * 全局公共页(个人中心 / 消息中心等,无 meta.module)的布局外壳。
 * 统一用 BaseLayout。不退化到 blank——公共页是需要 header/sider 的功能页,blank 会让宽布局挤变形。
 */
const layout = BaseLayout;
</script>

<template>
  <component :is="layout" />
</template>
