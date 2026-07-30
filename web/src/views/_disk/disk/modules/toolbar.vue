<script setup lang="ts">
import { computed, ref } from 'vue';
import { $t } from '@/locales';
import { useDiskStore } from '@/store/modules/disk';
import { useAppStore } from '@/store/modules/app';
import { useSvgIcon } from '@/hooks/common/icon';
import type { DropdownOption } from 'naive-ui';

defineOptions({ name: 'DiskToolbar' });

type SortField = 'name' | 'size' | 'modifyTime';
type SortOrder = 'asc' | 'desc';

interface Emits {
  (e: 'search', keyword: string): void;
  (e: 'refresh'): void;
  (e: 'sort', field: SortField, order: SortOrder): void;
  (e: 'toggleView'): void;
  (e: 'toggleGridSize'): void;
}

const emit = defineEmits<Emits>();

const diskStore = useDiskStore();
const appStore = useAppStore();
const { SvgIconVNode } = useSvgIcon();

const keyword = ref('');

/** 排序下拉：字段 × 方向合一，一次点选；方向以 ↑/↓ 表达，复用现有 i18n key */
const sortOptions = computed<DropdownOption[]>(() => {
  const fields: { key: SortField; label: string; icon: string }[] = [
    { key: 'name', label: $t('page.disk.sort.name'), icon: 'material-symbols:abc' },
    { key: 'size', label: $t('page.disk.sort.size'), icon: 'material-symbols:database' },
    { key: 'modifyTime', label: $t('page.disk.sort.modifyTime'), icon: 'material-symbols:schedule' }
  ];
  return fields.flatMap(f => [
    { key: `${f.key}-asc`, label: `${f.label} ↑`, icon: SvgIconVNode({ icon: f.icon, fontSize: 18 }) },
    { key: `${f.key}-desc`, label: `${f.label} ↓`, icon: SvgIconVNode({ icon: f.icon, fontSize: 18 }) }
  ]);
});

/** 排序按钮 tooltip：反映当前排序状态 */
const sortTooltip = computed(() => {
  const { field, order } = diskStore.sortSettings;
  if (!field) return $t('page.disk.toolbar.sort');
  const fieldLabel =
    field === 'name' ? $t('page.disk.sort.name') : field === 'size' ? $t('page.disk.sort.size') : $t('page.disk.sort.modifyTime');
  return `${$t('page.disk.toolbar.sort')} · ${fieldLabel} ${order === 'asc' ? '↑' : '↓'}`;
});

function handleSearch() {
  emit('search', keyword.value);
}

function handleRefresh() {
  emit('refresh');
}

function handleSortSelect(key: string) {
  const [field, order] = key.split('-') as [SortField, SortOrder];
  emit('sort', field, order);
}

function handleToggleView() {
  emit('toggleView');
}

function handleToggleGridSize() {
  emit('toggleGridSize');
}
</script>

<template>
  <div class="flex-y-center gap-8px px-12px py-8px flex-wrap">
    <!-- 搜索：flex-1 占位，将右侧功能组推到行尾；max-w 限制桌面宽度，窄屏自适应收缩 -->
    <NInputGroup class="flex-1 max-w-240px" size="small">
      <NInput
        v-model:value="keyword"
        :placeholder="$t('page.disk.toolbar.searchPlaceholder')"
        clearable
        @keyup.enter="handleSearch"
        @clear="handleSearch"
      />
      <NButton size="small" type="primary" :focusable="false" @click="handleSearch">
        <SvgIcon icon="material-symbols:search" class="text-16px" />
      </NButton>
    </NInputGroup>

    <!-- 功能按钮组 -->
    <NButtonGroup size="small">
      <!-- 排序：字段×方向合一 -->
      <NDropdown :options="sortOptions" trigger="click" @select="handleSortSelect">
        <NTooltip trigger="hover">
          <template #trigger>
            <NButton quaternary :focusable="false">
              <SvgIcon icon="material-symbols:sort" class="text-18px" />
            </NButton>
          </template>
          {{ sortTooltip }}
        </NTooltip>
      </NDropdown>

      <!-- 视图切换：单按钮 toggle（移动端隐藏，固定列表） -->
      <NTooltip v-if="!appStore.isMobile" trigger="hover">
        <template #trigger>
          <NButton quaternary :focusable="false" @click="handleToggleView">
            <SvgIcon
              :icon="diskStore.viewMode === 'grid' ? 'material-symbols:grid-view' : 'material-symbols:view-list'"
              class="text-18px"
            />
          </NButton>
        </template>
        {{ diskStore.viewMode === 'grid' ? $t('page.disk.toolbar.listView') : $t('page.disk.toolbar.gridView') }}
      </NTooltip>

      <!-- 大图/小图切换：仅 grid 模式 + 非移动端 -->
      <NTooltip v-if="!appStore.isMobile && diskStore.viewMode === 'grid'" trigger="hover">
        <template #trigger>
          <NButton quaternary :focusable="false" @click="handleToggleGridSize">
            <SvgIcon
              :icon="diskStore.gridSize === 'large' ? 'material-symbols:apps' : 'material-symbols:grid-view'"
              class="text-18px"
            />
          </NButton>
        </template>
        {{ diskStore.gridSize === 'large' ? $t('page.disk.toolbar.smallGrid') : $t('page.disk.toolbar.largeGrid') }}
      </NTooltip>

      <!-- 刷新 -->
      <NTooltip trigger="hover">
        <template #trigger>
          <NButton quaternary :focusable="false" @click="handleRefresh">
            <SvgIcon icon="material-symbols:refresh" class="text-18px" />
          </NButton>
        </template>
        {{ $t('page.disk.toolbar.refresh') }}
      </NTooltip>
    </NButtonGroup>
  </div>
</template>
