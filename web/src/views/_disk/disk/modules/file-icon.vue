<script setup lang="ts">
import { computed } from 'vue';
import { getFileIcon } from '@/service/api/disk/file';

defineOptions({ name: 'FileIcon' });

interface Props {
  /** 文件类型 (folder/image/document/video/audio/other) */
  fileType: string;
  /** 文件扩展名（不含点号） */
  extension?: string;
  /** 图标尺寸 px */
  size?: number;
}

const props = withDefaults(defineProps<Props>(), {
  extension: undefined,
  size: 40
});

const iconName = computed(() => {
  if (props.fileType === 'folder') return 'material-symbols:folder';
  return getFileIcon(props.extension);
});
</script>

<template>
  <SvgIcon :icon="iconName" :style="{ fontSize: `${size}px` }" class="shrink-0" />
</template>
