<script setup lang="ts">
import { computed, ref } from 'vue';
import SvgIcon from '@/components/custom/svg-icon.vue';
import IconUrl from '@/components/custom/icon-url.vue';
import { getLocalAiIcons } from '@/utils/icon';
import { $t } from '@/locales';

defineOptions({ name: 'IconPicker' });

/**
 * 通用图标选择器(MCP/Skill 等表单共用)：
 * - 值存本地图标名(ai-xxx，SvgIcon symbol 体系)，弹层九宫格点选+搜索
 * - 历史远程 URL 值兼容展示(IconUrl 渲染)，选中新图标即覆盖
 */
const value = defineModel<string | null>('value', { default: '' });

const showPicker = ref(false);
const search = ref('');

const iconNames = getLocalAiIcons();

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return iconNames;
  return iconNames.filter(name => name.replace(/^ai-/, '').includes(q));
});

/** 触发器旁注：本地图标显示去前缀名，远程值原样 */
const isLocalIcon = computed(() => {
  const v = value.value;
  return !!v && !/^(https?:)?\/\//.test(v) && !v.startsWith('/');
});
const displayLabel = computed(() => {
  const v = value.value;
  return v && isLocalIcon.value ? v.replace(/^ai-/, '') : v || '';
});

function handleSelect(name: string) {
  value.value = name;
  showPicker.value = false;
  search.value = '';
}

function handleClear() {
  value.value = '';
}
</script>

<template>
  <div class="flex w-full items-center gap-8px">
    <NPopover v-model:show="showPicker" trigger="click" :width="368">
      <template #trigger>
        <button
          type="button"
          class="flex h-32px w-32px shrink-0 items-center justify-center rounded-4px border border-gray-200 transition-colors hover:border-primary dark:border-gray-700 dark:hover:bg-gray-800"
        >
          <IconUrl v-if="value" :value="value" :size="18" />
          <span v-else class="text-18px text-gray-400">+</span>
        </button>
      </template>
      <div class="flex flex-col gap-8px">
        <NInput v-model:value="search" size="small" clearable :placeholder="$t('common.iconPicker.searchPlaceholder')" />
        <div class="grid max-h-240px grid-cols-8 gap-4px overflow-y-auto pr-4px">
          <button
            v-for="name in filtered"
            :key="name"
            type="button"
            :title="name.replace(/^ai-/, '')"
            class="flex h-32px w-32px items-center justify-center rounded-4px transition-colors hover:bg-primary-100 dark:hover:bg-gray-700"
            :class="value === name ? 'bg-primary-100 text-primary dark:bg-gray-700' : ''"
            @click="handleSelect(name)"
          >
            <SvgIcon :local-icon="name" class="text-18px" />
          </button>
        </div>
        <p v-if="!filtered.length" class="py-8px text-center text-12px text-gray-400">
          {{ $t('common.iconPicker.noMatch') }}
        </p>
      </div>
    </NPopover>
    <span v-if="value" class="truncate text-12px text-gray-400">{{ displayLabel }}</span>
    <span v-else class="text-12px text-gray-400">{{ $t('common.iconPicker.unset') }}</span>
    <NButton v-if="value" text type="primary" size="tiny" class="ml-auto" @click="handleClear">
      {{ $t('common.clear') }}
    </NButton>
  </div>
</template>

<style scoped></style>
