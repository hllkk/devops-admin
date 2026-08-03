<script setup lang="ts">
import { computed, ref } from 'vue';
import { $t } from '@/locales';
import { useDiskStore } from '@/store/modules/disk';
import { useAppStore } from '@/store/modules/app';
import { useSvgIcon } from '@/hooks/common/icon';
import { useDiskUpload } from '@/hooks/business/disk/use-disk-upload';
import { collectFolderDirs } from '@/utils/disk';
import type { DropdownOption } from 'naive-ui';

defineOptions({ name: 'DiskToolbar' });

type SortField = 'name' | 'size' | 'modifyTime';
type SortOrder = 'asc' | 'desc';
type CreateType = 'file' | 'folder';
/** 视图模式：列表 / 缩略(网格小图) / 大图(网格大图) */
type ViewMode = 'list' | 'thumbnail' | 'large';

interface Emits {
  (e: 'search', keyword: string): void;
  (e: 'refresh'): void;
  (e: 'sort', field: SortField, order: SortOrder): void;
  (e: 'set-view', mode: ViewMode): void;
  (e: 'create', type: CreateType): void;
  (e: 'upload', type: CreateType, files: File[], dirs?: string[]): void;
}

const emit = defineEmits<Emits>();

const diskStore = useDiskStore();
const appStore = useAppStore();
const { SvgIconVNode } = useSvgIcon();
const { uploadingCount, togglePanel } = useDiskUpload();

const keyword = ref('');

/** 上传下拉:上传文件 / 上传文件夹 */
const uploadOptions = computed<DropdownOption[]>(() => [
  { key: 'uploadFile', label: $t('page.disk.action.uploadFile'), icon: SvgIconVNode({ localIcon: 'disk-upload-file', fontSize: 18 }) },
  { key: 'uploadFolder', label: $t('page.disk.action.uploadFolder'), icon: SvgIconVNode({ localIcon: 'disk-upload-folder', fontSize: 18 }) }
]);

/** 新建下拉:新建文件夹 / 新建文件 */
const createOptions = computed<DropdownOption[]>(() => [
  { key: 'createFolder', label: $t('page.disk.action.newFolder'), icon: SvgIconVNode({ localIcon: 'disk-create-folder', fontSize: 18 }) },
  { key: 'createFile', label: $t('page.disk.action.createFile'), icon: SvgIconVNode({ localIcon: 'disk-create-file', fontSize: 18 }) }
]);

/** 移动端 + 号合并下拉:上传 + 分隔 + 新建 */
const mobilePlusOptions = computed<DropdownOption[]>(() => [
  ...uploadOptions.value,
  { type: 'divider', key: 'mobile-divider' } as DropdownOption,
  ...createOptions.value
]);

function handleUploadSelect(key: string) {
  if (key === 'uploadFile') pickFiles('file');
  else if (key === 'uploadFolder') pickFiles('folder');
}

function handleCreateSelect(key: string) {
  if (key === 'createFile') emit('create', 'file');
  else if (key === 'createFolder') emit('create', 'folder');
}

function handleMobilePlusSelect(key: string) {
  if (key === 'uploadFile' || key === 'uploadFolder') handleUploadSelect(key);
  else handleCreateSelect(key);
}

/** 动态创建 file input 选文件/文件夹(文件夹用 webkitdirectory 保留层级)。
 *  文件夹模式额外用 webkitEntries 递归收集所有目录(含空目录),随 emit 上传供 ensure-folders 预建。 */
function pickFiles(type: CreateType) {
  const input = document.createElement('input');
  input.type = 'file';
  input.multiple = true;
  if (type === 'folder') {
    input.setAttribute('webkitdirectory', '');
    input.setAttribute('directory', '');
  }
  input.addEventListener('change', async () => {
    if (input.files && input.files.length) {
      // 文件夹模式:从 webkitEntries 递归收集所有目录(含空目录),供 ensure-folders 预建目录树
      const dirs = type === 'folder' ? await collectFolderDirs(input) : [];
      emit('upload', type, Array.from(input.files), dirs);
    }
    input.remove();
  });
  input.click();
}

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
    field === 'name'
      ? $t('page.disk.sort.name')
      : field === 'size'
        ? $t('page.disk.sort.size')
        : $t('page.disk.sort.modifyTime');
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

/** 视图模式三档图标：列表 / 缩略(网格小图) / 大图(网格大图) */
const VIEW_ICONS: Record<ViewMode, string> = {
  list: 'material-symbols:view-list',
  thumbnail: 'material-symbols:apps',
  large: 'material-symbols:grid-view'
};

/** 当前视图模式（由 store 双维度 viewMode + gridSize 组合反推） */
const currentViewMode = computed<ViewMode>(() => {
  if (diskStore.viewMode === 'list') return 'list';
  return diskStore.gridSize === 'large' ? 'large' : 'thumbnail';
});

/** 视图下拉：当前模式项前缀 ✓ 标识选中态 */
const viewOptions = computed<DropdownOption[]>(() => {
  const cur = currentViewMode.value;
  const items: { key: ViewMode; label: string; icon: string }[] = [
    { key: 'list', label: $t('page.disk.toolbar.viewList'), icon: VIEW_ICONS.list },
    { key: 'thumbnail', label: $t('page.disk.toolbar.viewThumbnail'), icon: VIEW_ICONS.thumbnail },
    { key: 'large', label: $t('page.disk.toolbar.viewLarge'), icon: VIEW_ICONS.large }
  ];
  return items.map(it => ({
    key: it.key,
    label: it.key === cur ? `✓ ${it.label}` : it.label,
    icon: SvgIconVNode({ icon: it.icon, fontSize: 18 })
  }));
});

function handleViewSelect(key: string) {
  emit('set-view', key as ViewMode);
}
</script>

<template>
  <div class="flex-y-center justify-between gap-8px px-12px py-8px flex-wrap">
    <!-- 左侧分组:主操作 + 搜索,内部 gap 紧凑排列,避免被外层 justify-between 拉散 -->
    <div class="flex-y-center gap-8px flex-wrap">
      <!-- 主操作 -->
      <template v-if="!appStore.isMobile">
        <!-- 上传 -->
        <NDropdown :options="uploadOptions" trigger="click" @select="handleUploadSelect">
          <NButton type="primary" :focusable="false">
            <SvgIcon icon="material-symbols:cloud-upload" class="text-18px" />
            <span class="text-13px">{{ $t('page.disk.action.upload') }}</span>
          </NButton>
        </NDropdown>
        <!-- 新建 -->
        <NDropdown :options="createOptions" trigger="click" @select="handleCreateSelect">
          <NButton type="primary" :focusable="false">
            <SvgIcon icon="material-symbols:add" class="text-18px" />
            <span class="text-13px">{{ $t('page.disk.action.create') }}</span>
          </NButton>
        </NDropdown>
      </template>
      <!-- 移动端:合并 + 号下拉 -->
      <NDropdown v-else :options="mobilePlusOptions" trigger="click" @select="handleMobilePlusSelect">
        <NButton type="primary" :focusable="false">
          <SvgIcon icon="material-symbols:add" class="text-18px" />
        </NButton>
      </NDropdown>

      <!-- 搜索:flex-1 + max-w 让桌面保持合理宽度,窄屏自适应收缩,min-w 防塌缩 -->
      <NInputGroup class="flex-1 max-w-240px min-w-120px">
        <NInput
          v-model:value="keyword"
          :placeholder="$t('page.disk.toolbar.searchPlaceholder')"
          clearable
          @keyup.enter="handleSearch"
          @clear="handleSearch"
        />
        <NButton type="primary" :focusable="false" @click="handleSearch">
          <SvgIcon icon="material-symbols:search" class="text-16px" />
        </NButton>
      </NInputGroup>
    </div>

    <!-- 功能按钮组 -->
    <NButtonGroup>
      <!-- 传输列表(角标显示上传中数量) -->
      <NTooltip v-if="!appStore.isMobile" trigger="hover">
        <template #trigger>
          <NButton ghost :focusable="false" @click="togglePanel">
            <NBadge :value="uploadingCount" :max="99" :show="uploadingCount > 0" :offset="[4, -2]">
              <SvgIcon icon="material-symbols:swap-vert" class="text-18px" />
            </NBadge>
          </NButton>
        </template>
        {{ $t('page.disk.action.transferList') }}
      </NTooltip>

      <!-- 排序：字段×方向合一 -->
      <NDropdown :options="sortOptions" trigger="click" @select="handleSortSelect">
        <NTooltip trigger="hover">
          <template #trigger>
            <NButton ghost :focusable="false">
              <SvgIcon icon="material-symbols:sort" class="text-18px" />
            </NButton>
          </template>
          {{ sortTooltip }}
        </NTooltip>
      </NDropdown>

      <!-- 视图模式：hover 下拉(列表 / 缩略 / 大图)，移动端隐藏固定列表 -->
      <NDropdown v-if="!appStore.isMobile" :options="viewOptions" trigger="hover" @select="handleViewSelect">
        <NButton ghost :focusable="false">
          <SvgIcon :icon="VIEW_ICONS[currentViewMode]" class="text-18px" />
        </NButton>
      </NDropdown>

      <!-- 刷新 -->
      <NTooltip trigger="hover">
        <template #trigger>
          <NButton ghost :focusable="false" @click="handleRefresh">
            <SvgIcon icon="material-symbols:refresh" class="text-18px" />
          </NButton>
        </template>
        {{ $t('page.disk.toolbar.refresh') }}
      </NTooltip>
    </NButtonGroup>
  </div>
</template>

<style scoped lang="scss">
/* 暗黑模式下 Naive 对 type="primary" 按钮的文字色按"浅色 primary"推算(其内置 darkTheme 的 primaryColor 为浅蓝),
   会配深色文字;而本项目 primary 为深紫 #646cff,深紫底 + 深色文字对比度极低,看起来就是黑字。
   这里显式锁定 primary 按钮内容为白色(图标用 currentColor 会一并跟随),保证上传/新建/搜索等按钮文字在明暗主题下都清晰。
   Naive themeOverrides 不暴露 primary 按钮文字色,故只能在组件层约束。 */
:deep(.n-button--primary-type .n-button__content) {
  color: #fff;
}
</style>
