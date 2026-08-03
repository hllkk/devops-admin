<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue';
import { NCheckbox } from 'naive-ui';
import { $t } from '@/locales';
import { useDiskStore } from '@/store/modules/disk';
import { useDiskCreate } from '@/hooks/business/disk/use-disk-create';
import FileIcon from './file-icon.vue';
import FileCard from './file-card.vue';
import FileEmpty from './file-empty.vue';
import DiskContextMenu from './context-menu.vue';

defineOptions({ name: 'FileGrid' });

interface Props {
  files: Api.Disk.FileItem[];
  loading?: boolean;
}

interface Emits {
  (e: 'action', type: Api.Disk.DiskActionType, file: Api.Disk.FileItem): void;
  (e: 'refresh'): void;
}

const props = defineProps<Props>();
const emit = defineEmits<Emits>();

const diskStore = useDiskStore();
const { submit, cancel, keydown, blur } = useDiskCreate();
const createInputRef = ref<{ focus: () => void; select: () => void } | null>(null);

// 进入行内新建态时聚焦占位卡输入框
watch(
  () => diskStore.creatingType,
  type => {
    if (!type) return;
    nextTick(() => {
      createInputRef.value?.focus();
      createInputRef.value?.select();
    });
  }
);

/** 透传 FileCard 的操作事件到上层 index.vue */
function onAction(type: Api.Disk.DiskActionType, file: Api.Disk.FileItem) {
  emit('action', type, file);
}

/** 网格列宽随大小档位（大图更宽列 / 缩略紧凑列） */
const gridTemplate = computed(() =>
  diskStore.gridSize === 'large' ? 'repeat(auto-fill, minmax(170px, 1fr))' : 'repeat(auto-fill, minmax(110px, 1fr))'
);

/** 全选态：当前可见文件全部选中 */
const allChecked = computed(
  () => props.files.length > 0 && props.files.every(f => diskStore.selectedFiles.includes(f.fileId))
);

/** 半选态：有选中但未全选 */
const indeterminate = computed(() => diskStore.selectedFiles.length > 0 && !allChecked.value);

/** 全选行文案：有选中显示"已选中 N 个"，否则显示"全选" */
const selectAllLabel = computed(() =>
  diskStore.selectedFiles.length > 0
    ? $t('page.disk.column.selectedCount', { count: diskStore.selectedFiles.length })
    : $t('common.selectAll')
);

/** 全选/取消全选（仅作用于当前可见文件） */
function toggleAll(checked: boolean) {
  if (checked) {
    diskStore.setSelectedFiles(props.files.map(f => f.fileId));
  } else {
    diskStore.clearSelection();
  }
}

/**
 * NScrollbar 外层包装容器 ref（原生 div，必定是 Element）。
 * 不直接读 NScrollbar 实例的 $el：NScrollbar 是多层组件包装（外层 Scrollbar → 内部 Scrollbar → VResizeObserver → div），
 * $el 需沿组件链穿透到真实 DOM，在 v-if 切换重挂的时序下可能解析为注释占位节点，
 * 导致 querySelector 不存在而抛 “root?.querySelector is not a function”。
 * 改用自有 div 作 querySelector 起点，起点必为 Element，彻底规避该报错。
 */
const wrapperRef = ref<HTMLElement | null>(null);

/** NScrollbar 内部真实滚动容器，供父组件无限滚动 hook 监听（scroll 事件 + scrollHeight 等） */
const scrollContainer = ref<HTMLElement | null>(null);

/**
 * NScrollbar 仅在"有文件或行内新建"分支渲染（loading/空态走 NSpin/FileEmpty）。
 * 滚动容器需随该分支挂载/卸载动态重取，而非 onMounted 一次性获取——
 * 否则首次进入(loading 态)或空文件夹时拿不到容器，无限滚动失效。
 */
const scrollbarVisible = computed(
  () => !(props.loading && props.files.length === 0) && !(props.files.length === 0 && !diskStore.creatingType)
);

watch(
  scrollbarVisible,
  async visible => {
    if (!visible) {
      scrollContainer.value = null;
      return;
    }
    await nextTick();
    scrollContainer.value = wrapperRef.value?.querySelector<HTMLElement>('.n-scrollbar-container') ?? null;
  },
  { immediate: true }
);

defineExpose({ scrollContainer });

/** 右键菜单状态(文件/空白双类型,本视图单例) */
interface ContextMenuState {
  visible: boolean;
  x: number;
  y: number;
  type: 'file' | 'area';
  targetFile: Api.Disk.FileItem | null;
}
const ctxState = ref<ContextMenuState>({
  visible: false,
  x: 0,
  y: 0,
  type: 'area',
  targetFile: null
});

/**
 * 右键统一入口:靠命中元素的 data-file-id 区分文件/空白。
 * FileCard 根 div 经 attrs fallthrough 带 data-file-id;命中即文件,否则空白。
 */
function handleContextMenu(e: MouseEvent) {
  if (props.loading) return;
  const target = (e.target as HTMLElement).closest<HTMLElement>('[data-file-id]');
  if (target) {
    const fileId = target.dataset.fileId;
    const file = props.files.find(f => String(f.fileId) === fileId);
    if (file) {
      // 右键未选中的文件:替换选中为该文件;已在选中集则保留(支持对多选批量操作)
      if (!diskStore.selectedFiles.includes(file.fileId)) {
        diskStore.setSelectedFiles([file.fileId]);
      }
      ctxState.value = { visible: true, x: e.clientX, y: e.clientY, type: 'file', targetFile: file };
      return;
    }
  }
  ctxState.value = { visible: true, x: e.clientX, y: e.clientY, type: 'area', targetFile: null };
}

/** 右键菜单选中项分发:文件操作透传上层,视图/排序/刷新内部消化 */
function handleContextSelect(key: string) {
  const file = ctxState.value.targetFile;
  switch (key) {
    case 'download':
    case 'copy':
    case 'move':
    case 'rename':
    case 'delete':
      if (file) emit('action', key as Api.Disk.DiskActionType, file);
      break;
    case 'view-grid':
      diskStore.setViewMode('grid');
      break;
    case 'view-list':
      diskStore.setViewMode('list');
      break;
    case 'sort-name':
    case 'sort-size':
    case 'sort-modifyTime':
      applySort(key.slice('sort-'.length) as 'name' | 'size' | 'modifyTime');
      break;
    case 'refresh':
      emit('refresh');
      break;
    case 'reload':
      window.location.reload();
      break;
    default:
      break;
  }
}

/** 排序:同字段翻转方向,否则默认升序(对齐 remote 右键排序交互) */
function applySort(field: 'name' | 'size' | 'modifyTime') {
  const cur = diskStore.sortSettings;
  if (cur.field === field) {
    diskStore.setSort(field, cur.order === 'asc' ? 'desc' : 'asc');
  } else {
    diskStore.setSort(field, 'asc');
  }
}
</script>

<template>
  <div ref="wrapperRef" class="h-full flex flex-col" @contextmenu.prevent="handleContextMenu">
    <!-- 全选行:固定顶部,不随滚动(有文件时显示) -->
    <div v-if="files.length > 0" class="flex-y-center gap-8px px-12px pt-8px pb-8px">
      <NCheckbox :checked="allChecked" :indeterminate="indeterminate" @update:checked="toggleAll" />
      <span class="text-13px">{{ selectAllLabel }}</span>
    </div>
    <!-- 首次加载:撑满双居中 -->
    <div v-if="loading && files.length === 0" class="flex-1 min-h-0 flex-center">
      <NSpin />
    </div>
    <!-- 空状态(无文件且非新建态):撑满双居中,不走 NScrollbar -->
    <FileEmpty v-else-if="files.length === 0 && !diskStore.creatingType" class="flex-1 min-h-0" />
    <!-- 有文件或行内新建:滚动区 -->
    <NScrollbar v-else class="flex-1 min-h-0">
      <div class="px-12px py-8px">
        <div class="grid gap-16px" :style="{ gridTemplateColumns: gridTemplate }">
          <!-- 行内新建占位卡 -->
          <div
            v-if="diskStore.creatingType"
            class="relative flex-col-center gap-6px p-10px rd-8px bg-primary/5"
            @click.stop
          >
            <div class="absolute right-4px top-4px z-1 flex items-center gap-2px" @click.stop @mousedown.prevent>
              <NButton text size="tiny" :focusable="false" @click="submit">
                <SvgIcon icon="material-symbols:check" class="text-16px text-primary" />
              </NButton>
              <NButton text size="tiny" :focusable="false" @click="cancel">
                <SvgIcon icon="material-symbols:close" class="text-16px opacity-50" />
              </NButton>
            </div>
            <FileIcon :file-type="diskStore.creatingType === 'folder' ? 'folder' : 'other'" :size="diskStore.gridSize === 'large' ? 120 : 80" />
            <div class="w-full px-4px">
              <NInput
                ref="createInputRef"
                :value="diskStore.creatingName"
                size="small"
                :placeholder="$t('page.disk.modal.namePlaceholder')"
                @update:value="(v: string) => diskStore.setCreatingName(v)"
                @keydown="keydown"
                @blur="blur"
              />
            </div>
          </div>
          <FileCard v-for="f in files" :key="f.fileId" :file="f" :data-file-id="String(f.fileId)" @action="onAction" />
        </div>
      </div>
    </NScrollbar>
    <!-- 右键菜单(文件/空白双类型,统一挂载点;NDropdown teleport 到 body 不占位) -->
    <DiskContextMenu
      v-model:visible="ctxState.visible"
      :x="ctxState.x"
      :y="ctxState.y"
      :type="ctxState.type"
      :file-is-favorite="ctxState.targetFile?.isFavorite"
      @select="handleContextSelect"
    />
  </div>
</template>
