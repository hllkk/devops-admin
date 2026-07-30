<script setup lang="ts">
import { computed } from 'vue';
import { useDiskStore } from '@/store/modules/disk';
import { $t } from '@/locales';
import FileIcon from './file-icon.vue';
import { formatFileSize } from '@/utils/format';

defineOptions({ name: 'FileCard' });

interface Props {
  file: Api.Disk.FileItem;
}

const props = defineProps<Props>();
const diskStore = useDiskStore();

const selected = computed(() => diskStore.selectedFiles.includes(props.file.fileId));

/** 图标尺寸随网格大小档位（大图 80 / 小图 56） */
const iconSize = computed(() => (diskStore.gridSize === 'large' ? 80 : 56));

function handleClick() {
  const id = props.file.fileId;
  if (selected.value) {
    diskStore.setSelectedFiles(diskStore.selectedFiles.filter(f => f !== id));
  } else {
    diskStore.setSelectedFiles([...diskStore.selectedFiles, id]);
  }
}

function handleDblClick() {
  // 第1期只读：双击文件夹进入；文件预览后续期补充
  if (props.file.isFolder) diskStore.enterFolder(props.file);
}
</script>

<template>
  <div
    class="flex-col-center cursor-pointer gap-6px p-10px rd-8px transition-colors"
    :class="selected ? 'disk-card-active' : 'hover:bg-layout'"
    @click="handleClick"
    @dblclick="handleDblClick"
  >
    <FileIcon :file-type="file.fileType" :extension="file.fileExtension" :size="iconSize" />
    <span class="w-full truncate text-center text-13px" :title="file.fileName">{{ file.fileName }}</span>
    <span v-if="!file.isFolder" class="text-11px opacity-50">{{ formatFileSize(file.fileSize) }}</span>
    <span v-else class="text-11px opacity-50">{{ $t('page.disk.file.folder') }}</span>
  </div>
</template>

<style scoped>
.disk-card-active {
  background-color: rgba(var(--primary-color-rgb), 0.1);
}
</style>
