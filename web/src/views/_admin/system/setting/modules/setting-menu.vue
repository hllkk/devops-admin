<script setup lang="ts">
import SvgIcon from '@/components/custom/svg-icon.vue';

defineOptions({ name: 'SettingMenu' });

interface MenuItem {
  key: string;
  label: string;
  desc: string;
  icon: string;
}

interface Props {
  activeKey: string;
  menuItems: MenuItem[];
}

interface Emits {
  (e: 'update:activeKey', value: string): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

function handleSelect(key: string) {
  emit('update:activeKey', key);
}
</script>

<template>
  <DarkModeContainer class="h-full flex flex-col rounded-md gap-2 p-2">
    <div
      v-for="item in menuItems"
      :key="item.key"
      class="ml-0 flex flex-none flex-row cursor-pointer items-center justify-start rounded-md bg-container p-4 transition-all duration-200 ease-in-out space-x-2 active:bg-primary-100 hover:bg-primary-50 dark:active:bg-primary-800/30 dark:hover:bg-primary-900/20"
      :class="[
        props.activeKey === item.key
          ? [
            'border-solid border-0 rounded-r-none border-r-3 border-primary-600',
            'bg-gradient-to-r from-primary-200/80 to-primary-100/60',
            'dark:border-primary-400 dark:from-primary-800/60 dark:to-primary-900/40',
            'shadow-sm'
          ]
          : 'border-transparent'
      ]"
      @click="handleSelect(item.key)"
    >
      <SvgIcon
        :icon="item.icon"
        class="text-28px"
        :class="props.activeKey === item.key ? 'text-primary' : 'text-icon'"
      />
      <div class="flex flex-col flex-1 space-y-1">
        <span
          class="select-none font-semibold transition-colors"
          :class="props.activeKey === item.key ? 'text-primary-800 dark:text-primary-200' : 'text-base-text'"
        >
          {{ item.label }}
        </span>
        <span
          class="select-none text-xs transition-colors"
          :class="[
            props.activeKey === item.key
              ? 'text-primary-700/80 dark:text-primary-300/80'
              : 'text-base-text/70 opacity-75'
          ]"
        >
          {{ item.desc }}
        </span>
      </div>
    </div>
  </DarkModeContainer>
</template>
