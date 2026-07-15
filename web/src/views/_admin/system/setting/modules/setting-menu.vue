<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'SettingMenu' });

interface Props {
  activeKey: string;
}

interface Emits {
  (e: 'update:activeKey', value: string): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();
const { t } = useI18n();

const menuItems = computed(() => [
  {
    key: 'general',
    label: t('page.system.setting.general'),
    desc: t('page.system.setting.generalDesc'),
    icon: 'mdi:cog-outline'
  },
  {
    key: 'security',
    label: t('page.system.setting.security'),
    desc: t('page.system.setting.securityDesc'),
    icon: 'mdi:shield-check-outline'
  }
]);

function handleSelect(key: string) {
  emit('update:activeKey', key);
}
</script>

<template>
  <div class="h-full flex flex-col gap-8px">
    <div
      v-for="item in menuItems"
      :key="item.key"
      class="flex cursor-pointer items-center gap-12px rounded-6px p-12px transition-all"
      :class="props.activeKey === item.key ? 'bg-primary-100 dark:bg-primary-500/20' : 'hover:bg-layout'"
      @click="handleSelect(item.key)"
    >
      <SvgIcon
        :icon="item.icon"
        class="text-28px"
        :class="props.activeKey === item.key ? 'text-primary' : 'text-icon'"
      />
      <div class="flex flex-1 flex-col gap-2px">
        <span
          class="text-14px font-500"
          :class="props.activeKey === item.key ? 'text-primary' : 'text-primary-text'"
        >
          {{ item.label }}
        </span>
        <span class="text-12px text-secondary-text">{{ item.desc }}</span>
      </div>
    </div>
  </div>
</template>
