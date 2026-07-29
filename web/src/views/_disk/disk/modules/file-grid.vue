<script setup lang="ts">
import { ref } from 'vue';
import { $t } from '@/locales';
import FileCard from './file-card.vue';

defineOptions({ name: 'FileGrid' });

interface Props {
  files: Api.Disk.FileItem[];
  loading?: boolean;
}

defineProps<Props>();

/** 暴露滚动容器供父组件无限滚动 hook 监听 */
const scrollContainer = ref<HTMLElement>();
defineExpose({ scrollContainer });
</script>

<template>
  <div ref="scrollContainer" class="h-full overflow-y-auto px-12px py-8px">
    <!-- 首次加载 -->
    <div v-if="loading && files.length === 0" class="flex-center py-40px">
      <NSpin />
    </div>
    <!-- 空状态 -->
    <div v-else-if="files.length === 0" class="flex-col-center py-40px opacity-50">
      <SvgIcon icon="material-symbols:folder-off-outline" class="text-56px" />
      <span class="mt-8px text-14px">{{ $t('page.disk.empty') }}</span>
    </div>
    <!-- 网格 -->
    <div v-else class="grid gap-8px" style="grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));">
      <FileCard v-for="f in files" :key="f.fileId" :file="f" />
    </div>
  </div>
</template>
