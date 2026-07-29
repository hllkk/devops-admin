<script setup lang="ts">
import { ref } from 'vue';
import { useDiskStore } from '@/store/modules/disk';
import { $t } from '@/locales';

defineOptions({ name: 'DiskToolbar' });

interface Emits {
  (e: 'search', keyword: string): void;
  (e: 'refresh'): void;
}

const emit = defineEmits<Emits>();

const diskStore = useDiskStore();
const keyword = ref('');

function handleSearch() {
  emit('search', keyword.value);
}

function handleRefresh() {
  emit('refresh');
}

/** 排序字段下拉 */
const sortOptions = [
  { label: $t('page.disk.sort.name'), key: 'name' },
  { label: $t('page.disk.sort.size'), key: 'size' },
  { label: $t('page.disk.sort.modifyTime'), key: 'modifyTime' }
];

function handleSort(key: string) {
  diskStore.setSort(key as 'name' | 'size' | 'modifyTime', diskStore.sortSettings.order);
}

function toggleOrder() {
  diskStore.setSort(diskStore.sortSettings.field, diskStore.sortSettings.order === 'asc' ? 'desc' : 'asc');
}
</script>

<template>
  <div class="flex-center-y gap-8px px-12px py-8px">
    <!-- 搜索 -->
    <NInput
      v-model:value="keyword"
      :placeholder="$t('page.disk.toolbar.searchPlaceholder')"
      clearable
      size="small"
      class="w-220px"
      @keyup.enter="handleSearch"
      @clear="handleSearch"
    >
      <template #prefix>
        <SvgIcon icon="material-symbols:search" class="text-16px" />
      </template>
    </NInput>
    <NButton size="small" quaternary @click="handleSearch">{{ $t('page.disk.toolbar.search') }}</NButton>

    <NDivider vertical />

    <!-- 刷新 -->
    <NButton size="small" quaternary :focusable="false" @click="handleRefresh">
      <SvgIcon icon="material-symbols:refresh" class="text-18px" />
    </NButton>

    <!-- 视图切换 -->
    <NButtonGroup size="small">
      <NButton :type="diskStore.viewMode === 'grid' ? 'primary' : 'default'" :focusable="false" @click="diskStore.setViewMode('grid')">
        <SvgIcon icon="material-symbols:grid-view" class="text-18px" />
      </NButton>
      <NButton :type="diskStore.viewMode === 'list' ? 'primary' : 'default'" :focusable="false" @click="diskStore.setViewMode('list')">
        <SvgIcon icon="material-symbols:view-list" class="text-18px" />
      </NButton>
    </NButtonGroup>

    <!-- 排序 -->
    <NDropdown :options="sortOptions" trigger="click" @select="handleSort">
      <NButton size="small" quaternary :focusable="false">
        <span class="flex-center-y gap-2px">
          {{ $t('page.disk.toolbar.sort') }}
          <SvgIcon icon="material-symbols:arrow-drop-down" class="text-18px" />
        </span>
      </NButton>
    </NDropdown>
    <NButton size="small" quaternary :focusable="false" @click="toggleOrder">
      <SvgIcon
        :icon="diskStore.sortSettings.order === 'asc' ? 'material-symbols:arrow-upward' : 'material-symbols:arrow-downward'"
        class="text-18px"
      />
    </NButton>
  </div>
</template>
