<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import { $t } from '@/locales';

defineOptions({
  name: 'GlobalLogo'
});

interface Props {
  /** Whether to show the title */
  showTitle?: boolean;
}

withDefaults(defineProps<Props>(), {
  showTitle: true
});

const route = useRoute();

// 品牌名按业务模块切换:AI 网关模块显 AIOps Gateway,其余(后台管理/服务器管理/未带模块信息)显 AIOps Admin
const title = computed(() => (route.meta.module === 'gateway' ? $t('system.gatewayTitle') : $t('system.adminTitle')));
</script>

<template>
  <RouterLink to="/" class="w-full flex-center nowrap-hidden">
    <SystemLogo class="size-32px text-primary" />
    <h2 v-show="showTitle" class="pl-8px text-16px text-primary font-bold transition duration-300 ease-in-out">
      {{ title }}
    </h2>
  </RouterLink>
</template>

<style scoped></style>
