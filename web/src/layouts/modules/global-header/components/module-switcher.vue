<script setup lang="ts">
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { ALL_MODULES, MODULE_CONFIG } from '@/constants/module';
import { useRouteStore } from '@/store/modules/route';
import { $t } from '@/locales';

defineOptions({ name: 'ModuleSwitcher' });

const router = useRouter();
const routeStore = useRouteStore();

/** Module switcher options — labels from i18n route keys */
const options = computed(() =>
  ALL_MODULES.map(m => ({
    label: $t(`route.${m}` as App.I18n.I18nKey),
    key: m
  }))
);

/** Navigate to the selected module's home (path = /<module>) */
function handleSelect(key: string) {
  router.push(`/${key}`);
}
</script>

<template>
  <NDropdown :options="options" trigger="click" @select="handleSelect">
    <ButtonIcon :icon="MODULE_CONFIG[routeStore.currentModule].icon" />
  </NDropdown>
</template>
