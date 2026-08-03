<script setup lang="tsx">
import { ref, computed, nextTick, watch } from 'vue';
import { NTime, NDropdown, NButton, NInput, type DataTableColumns } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { useDiskCreate } from '@/hooks/business/disk/use-disk-create';
import { $t } from '@/locales';
import { formatFileSize } from '@/utils/format';
import SvgIcon from '@/components/custom/svg-icon.vue';
import FileIcon from './file-icon.vue';
import FileEmpty from './file-empty.vue';

defineOptions({ name: 'FileList' });

interface Props {
  files: Api.Disk.FileItem[];
  loading?: boolean;
}

const props = defineProps<Props>();
interface Emits {
  (e: 'action', type: Api.Disk.DiskActionType, file: Api.Disk.FileItem): void;
}
const emit = defineEmits<Emits>();
const diskStore = useDiskStore();
const { submit, cancel, keydown, blur } = useDiskCreate();

/** 行内新建虚拟行 fileId 哨兵(不与真实 id 冲突) */
const CREATE_ID = '__disk_create__';
const createInputRef = ref<{ focus: () => void; select: () => void } | null>(null);

/** 列表数据:行内新建时在顶部插一条虚拟创建行 */
const tableData = computed<Api.Disk.FileItem[]>(() => {
  if (!diskStore.creatingType) return props.files;
  const isFolder = diskStore.creatingType === 'folder';
  // 字段补全:虚拟行只填 4 字段就 as FileItem,会导致 modifyTime/fileSize 等列读到 undefined——
  // NTime(Date.parse(''))=NaN、formatFileSize(undefined) 触发渲染异常,创建行整行不显示。
  // 补全为安全空值,配合下方各列对 CREATE_ID 的占位返回,杜绝渲染崩溃。
  const creationRow = {
    fileId: CREATE_ID,
    fileName: '',
    isFolder,
    fileType: isFolder ? 'folder' : 'other',
    fileSize: 0,
    fileExtension: '',
    modifyTime: '',
    createTime: '',
    updateTime: '',
    createBy: '',
    updateBy: '',
    parentId: null
  } as Api.Disk.FileItem;
  return [creationRow, ...props.files];
});

// 进入新建/重命名态时聚焦并选中文本
watch(
  () => [diskStore.creatingType, diskStore.renamingId],
  () => {
    if (!diskStore.creatingType && !diskStore.renamingId) return;
    nextTick(() => {
      createInputRef.value?.focus();
      createInputRef.value?.select();
    });
  }
);

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
 * NScrollbar 仅在"有数据/新建态"分支渲染（loading/空态走 NSpin/FileEmpty）。
 * 滚动容器需随该分支挂载/卸载动态重取，而非 onMounted 一次性获取——
 * 否则首次进入(loading 态)或空文件夹时拿不到容器，无限滚动失效。
 */
const scrollbarVisible = computed(() => tableData.value.length > 0);

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

const columns = computed<DataTableColumns<Api.Disk.FileItem>>(() => [
  {
    // selection 列：名称左侧独立复选框列，表头自带全选；受控于 store.selectedFiles
    type: 'selection',
    disabled: (row: Api.Disk.FileItem) => row.fileId === CREATE_ID
  },
  {
    key: 'fileName',
    title:
      diskStore.selectedFiles.length > 0
        ? $t('page.disk.column.selectedCount', { count: diskStore.selectedFiles.length })
        : $t('page.disk.column.name'),
    render: row => {
      const isCreating = row.fileId === CREATE_ID;
      const isRenaming = !!diskStore.renamingId && diskStore.renamingId === row.fileId;
      if (isCreating || isRenaming) {
        const iconType = isCreating
          ? diskStore.creatingType === 'folder'
            ? 'folder'
            : 'other'
          : row.isFolder
            ? 'folder'
            : row.fileType;
        return (
          <div class="flex-y-center gap-8px" onClick={(e: Event) => e.stopPropagation()}>
            <FileIcon fileType={iconType} extension={row.fileExtension} size="small" />
            <NInput
              ref={(el: any) => {
                createInputRef.value = el;
              }}
              value={diskStore.creatingName}
              onUpdate:value={(v: string) => diskStore.setCreatingName(v)}
              size="small"
              placeholder={$t('page.disk.modal.namePlaceholder')}
              onKeydown={keydown}
              onBlur={blur}
            />
            <button
              type="button"
              class="flex-center h-22px w-22px cursor-pointer border-none rd-4px bg-primary/10 text-primary hover:bg-primary/20"
              onMousedown={(e: Event) => e.preventDefault()}
              onClick={(e: Event) => {
                e.stopPropagation();
                submit();
              }}
            >
              <SvgIcon icon="material-symbols:check" class="text-14px" />
            </button>
            <button
              type="button"
              class="flex-center h-22px w-22px cursor-pointer border-none rd-4px opacity-50 hover:bg-layout"
              onMousedown={(e: Event) => e.preventDefault()}
              onClick={(e: Event) => {
                e.stopPropagation();
                cancel();
              }}
            >
              <SvgIcon icon="material-symbols:close" class="text-14px" />
            </button>
          </div>
        );
      }
      return (
        <div class="flex-y-center gap-8px">
          <FileIcon fileType={row.isFolder ? 'folder' : row.fileType} extension={row.fileExtension} size="small" />
          <span class="truncate" title={row.fileName}>
            {row.fileName}
          </span>
        </div>
      );
    }
  },
  {
    key: 'fileSize',
    title: $t('page.disk.column.size'),
    width: 120,
    render: row => (row.fileId === CREATE_ID || row.isFolder ? '-' : formatFileSize(row.fileSize))
  },
  {
    key: 'modifyTime',
    title: $t('page.disk.column.modifyTime'),
    width: 180,
    render: row =>
      row.fileId === CREATE_ID ? (
        '-'
      ) : (
        <NTime time={Date.parse(row.modifyTime ?? '')} format="yyyy-MM-dd HH:mm:ss" />
      )
  },
  {
    key: 'actions',
    title: $t('page.disk.action.more'),
    width: 70,
    render: row => {
      if (row.fileId === CREATE_ID) return null;
      return (
        <NDropdown
          trigger="click"
          options={[
            ...(!row.isFolder ? [{ label: $t('page.disk.action.download'), key: 'download' }] : []),
            { label: $t('page.disk.action.rename'), key: 'rename' },
            { label: $t('page.disk.action.move'), key: 'move' },
            { label: $t('page.disk.action.copy'), key: 'copy' },
            { label: $t('page.disk.action.delete'), key: 'delete' }
          ]}
          onSelect={(key: string) => emit('action', key as Api.Disk.DiskActionType, row)}
        >
          <NButton quaternary size="tiny" focusable={false}>
            ···
          </NButton>
        </NDropdown>
      );
    }
  }
]);

/** NDataTable selection 列行 key */
const rowKey = (row: Api.Disk.FileItem) => row.fileId;

/** selection 列选中变更：同步到 store（受控，表头全选/单行勾选均走这里） */
function onCheckedRowKeysChange(keys: Array<string | number>) {
  diskStore.setSelectedFiles(keys as CommonType.IdType[]);
}

const rowProps = (row: Api.Disk.FileItem) => {
  // 创建行不参与进入
  if (row.fileId === CREATE_ID) return { style: '' };
  return {
    style: 'cursor: pointer;',
    // 选中交由 selection 列复选框负责（避免行点击与 selection 双重 toggle 抵消）
    onDblclick: () => {
      if (row.isFolder) diskStore.enterFolder(row);
    }
  };
};

const rowClassName = (row: Api.Disk.FileItem) => {
  // 创建行:整行高亮(对齐 grid 占位卡 bg-primary/5 的视觉提示),优先于选中态
  if (row.fileId === CREATE_ID) return 'disk-row-creating';
  return diskStore.selectedFiles.includes(row.fileId) ? 'disk-row-active' : '';
};
</script>

<template>
  <div ref="wrapperRef" class="h-full">
    <!-- 首次加载:撑满双居中 -->
    <div v-if="props.loading && tableData.length === 0" class="h-full flex-center">
      <NSpin />
    </div>
    <!-- 空状态(无数据):撑满双居中,不渲染表格→表头自然消失 -->
    <FileEmpty v-else-if="tableData.length === 0" class="h-full" />
    <!-- 有数据/新建态:表格 -->
    <NScrollbar v-else class="h-full">
      <NDataTable
        :columns="columns"
        :data="tableData"
        :loading="props.loading"
        :bordered="false"
        :row-key="rowKey"
        :checked-row-keys="diskStore.selectedFiles"
        :row-props="rowProps"
        :row-class-name="rowClassName"
        @update:checked-row-keys="onCheckedRowKeysChange"
      />
    </NScrollbar>
  </div>
</template>

<style scoped>
:deep(.disk-row-active td) {
  background-color: rgba(var(--primary-color-rgb), 0.1) !important;
}
/* 行内新建虚拟行:整行浅色高亮,对齐 grid 占位卡的视觉提示 */
:deep(.disk-row-creating td) {
  background-color: rgba(var(--primary-color-rgb), 0.06) !important;
}
</style>
