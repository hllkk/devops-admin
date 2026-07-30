<script setup lang="tsx">
import { ref, computed, onMounted, nextTick } from 'vue';
import type { ComponentPublicInstance } from 'vue';
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

const scrollbarRef = ref<ComponentPublicInstance | null>(null);

/** NScrollbar 内部真实滚动容器，供父组件无限滚动 hook 监听（scroll 事件 + scrollHeight 等） */
const scrollContainer = ref<HTMLElement | null>(null);

onMounted(async () => {
  await nextTick();
  const root = scrollbarRef.value?.$el as HTMLElement | undefined;
  scrollContainer.value = root?.querySelector<HTMLElement>('.n-scrollbar-container') ?? null;
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
  <NScrollbar ref="scrollbarRef" class="h-full">
    <NDataTable
      :columns="columns"
      :data="props.files"
      :loading="props.loading"
      :bordered="false"
      :row-props="rowProps"
      :row-class-name="rowClassName"
    />
  </NScrollbar>
</template>

<style scoped>
:deep(.disk-row-active td) {
  background-color: rgba(var(--primary-color-rgb), 0.1) !important;
}
</style>
