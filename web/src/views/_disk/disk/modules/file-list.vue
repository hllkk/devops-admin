<script setup lang="tsx">
import { ref, computed, onMounted, nextTick } from 'vue';
import { NTime, type DataTableColumns } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { $t } from '@/locales';
import { formatFileSize } from '@/utils/format';
import FileIcon from './file-icon.vue';

defineOptions({ name: 'FileList' });

interface Props {
  files: Api.Disk.FileItem[];
  loading?: boolean;
}

const props = defineProps<Props>();
const diskStore = useDiskStore();

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
    render: row => (
      <div class="flex-y-center gap-8px">
        <FileIcon fileType={row.isFolder ? 'folder' : row.fileType} extension={row.fileExtension} size="small" />
        <span class="truncate" title={row.fileName}>
          {row.fileName}
        </span>
      </div>
    )
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

const rowProps = (row: Api.Disk.FileItem) => ({
  style: 'cursor: pointer;',
  onClick: () => toggleSelect(row),
  onDblclick: () => {
    if (row.isFolder) diskStore.enterFolder(row);
  }
});

const rowClassName = (row: Api.Disk.FileItem) => (diskStore.selectedFiles.includes(row.fileId) ? 'disk-row-active' : '');
</script>

<template>
  <div ref="wrapperRef" class="h-full">
    <NScrollbar class="h-full">
      <NDataTable
        :columns="columns"
        :data="props.files"
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
