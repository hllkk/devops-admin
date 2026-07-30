<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue';
import type { ComponentPublicInstance } from 'vue';
import { $t } from '@/locales';
import { useDiskStore } from '@/store/modules/disk';
import FileCard from './file-card.vue';

defineOptions({ name: 'FileGrid' });

interface Props {
  files: Api.Disk.FileItem[];
  loading?: boolean;
}

defineProps<Props>();

const diskStore = useDiskStore();

/** 网格列宽随大小档位（大图更宽列 / 小图紧凑列） */
const gridTemplate = computed(() =>
  diskStore.gridSize === 'large' ? 'repeat(auto-fill, minmax(150px, 1fr))' : 'repeat(auto-fill, minmax(110px, 1fr))'
);

const scrollbarRef = ref<ComponentPublicInstance | null>(null);

/** NScrollbar 内部真实滚动容器，供父组件无限滚动 hook 监听（scroll 事件 + scrollHeight 等） */
const scrollContainer = ref<HTMLElement | null>(null);

onMounted(async () => {
  await nextTick();
  const root = scrollbarRef.value?.$el as HTMLElement | undefined;
  scrollContainer.value = root?.querySelector<HTMLElement>('.n-scrollbar-container') ?? null;
});

defineExpose({ scrollContainer });
</script>

<template>
  <NScrollbar ref="scrollbarRef" class="h-full">
    <div class="px-12px py-8px">
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
      <div v-else class="grid gap-8px" :style="{ gridTemplateColumns: gridTemplate }">
        <FileCard v-for="f in files" :key="f.fileId" :file="f" />
      </div>
    </div>
  </NScrollbar>
</template>
