<script setup lang="tsx">
import { ref, computed, onMounted, nextTick, watch } from 'vue';
import { NTime, NDropdown, NButton, NInput, type DataTableColumns } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { useDiskCreate } from '@/hooks/business/disk/use-disk-create';
import { $t } from '@/locales';
import { formatFileSize } from '@/utils/format';
import SvgIcon from '@/components/custom/svg-icon.vue';
import FileIcon from './file-icon.vue';

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
  const creationRow = {
    fileId: CREATE_ID,
    fileName: '',
    isFolder,
    fileType: isFolder ? 'folder' : 'other'
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

onMounted(async () => {
  await nextTick();
  scrollContainer.value = wrapperRef.value?.querySelector<HTMLElement>('.n-scrollbar-container') ?? null;
});

defineExpose({ scrollContainer });

const columns = computed<DataTableColumns<Api.Disk.FileItem>>(() => [
  {
    key: 'fileName',
    title: $t('page.disk.column.name'),
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
    render: row => (row.isFolder ? '-' : formatFileSize(row.fileSize))
  },
  {
    key: 'modifyTime',
    title: $t('page.disk.column.modifyTime'),
    width: 180,
    render: row => <NTime time={Date.parse(row.modifyTime ?? '')} format="yyyy-MM-dd HH:mm:ss" />
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

function toggleSelect(row: Api.Disk.FileItem) {
  const id = row.fileId;
  if (diskStore.selectedFiles.includes(id)) {
    diskStore.setSelectedFiles(diskStore.selectedFiles.filter(f => f !== id));
  } else {
    diskStore.setSelectedFiles([...diskStore.selectedFiles, id]);
  }
}

const rowProps = (row: Api.Disk.FileItem) => {
  // 创建行不参与选中/进入
  if (row.fileId === CREATE_ID) return { style: '' };
  return {
    style: 'cursor: pointer;',
    onClick: () => toggleSelect(row),
    onDblclick: () => {
      if (row.isFolder) diskStore.enterFolder(row);
    }
  };
};

const rowClassName = (row: Api.Disk.FileItem) => (diskStore.selectedFiles.includes(row.fileId) ? 'disk-row-active' : '');
</script>

<template>
  <div ref="wrapperRef" class="h-full">
    <NScrollbar class="h-full">
      <NDataTable
        :columns="columns"
        :data="tableData"
        :loading="props.loading"
        :bordered="false"
        :row-props="rowProps"
        :row-class-name="rowClassName"
      />
    </NScrollbar>
  </div>
</template>

<style scoped>
:deep(.disk-row-active td) {
  background-color: rgba(var(--primary-color-rgb), 0.1) !important;
}
</style>
