<script setup lang="ts">
import { ref, computed, h } from 'vue';
import { NTime, type DataTableColumns } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { $t } from '@/locales';
import SvgIcon from '@/components/custom/svg-icon.vue';
import { getFileIcon } from '@/service/api/disk/file';
import { formatFileSize } from '../utils/format';

defineOptions({ name: 'FileList' });

interface Props {
  files: Api.Disk.FileItem[];
  loading?: boolean;
}

const props = defineProps<Props>();
const diskStore = useDiskStore();

/** 暴露滚动容器供父组件无限滚动 hook 监听 */
const scrollContainer = ref<HTMLElement>();
defineExpose({ scrollContainer });

const columns = computed<DataTableColumns<Api.Disk.FileItem>>(() => [
  {
    key: 'fileName',
    title: $t('page.disk.column.name'),
    render: row =>
      h('div', { class: 'flex-center-y gap-8px' }, [
        h(SvgIcon, {
          icon: row.isFolder ? 'material-symbols:folder' : getFileIcon(row.fileExtension),
          style: { fontSize: '20px' }
        }),
        h('span', { class: 'truncate', title: row.fileName }, row.fileName)
      ])
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
    render: row => h(NTime, { time: Date.parse(row.modifyTime ?? ''), format: 'yyyy-MM-dd HH:mm:ss' })
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
  <div ref="scrollContainer" class="h-full overflow-y-auto">
    <NDataTable
      :columns="columns"
      :data="props.files"
      :loading="props.loading"
      :bordered="false"
      :row-props="rowProps"
      :row-class-name="rowClassName"
    />
  </div>
</template>

<style scoped>
:deep(.disk-row-active td) {
  background-color: rgba(var(--primary-color-rgb), 0.1) !important;
}
</style>
