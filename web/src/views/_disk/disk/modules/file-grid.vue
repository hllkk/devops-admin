<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue';
import { $t } from '@/locales';
import { useDiskStore } from '@/store/modules/disk';
import { useDiskCreate } from '@/hooks/business/disk/use-disk-create';
import FileIcon from './file-icon.vue';
import FileCard from './file-card.vue';

defineOptions({ name: 'FileGrid' });

interface Props {
  files: Api.Disk.FileItem[];
  loading?: boolean;
}

interface Emits {
  (e: 'action', type: Api.Disk.DiskActionType, file: Api.Disk.FileItem): void;
}

defineProps<Props>();
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

/** 网格列宽随大小档位（大图更宽列 / 小图紧凑列） */
const gridTemplate = computed(() =>
  diskStore.gridSize === 'large' ? 'repeat(auto-fill, minmax(150px, 1fr))' : 'repeat(auto-fill, minmax(110px, 1fr))'
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
</script>

<template>
  <div ref="wrapperRef" class="h-full">
    <NScrollbar class="h-full">
      <div class="px-12px py-8px">
        <!-- 首次加载 -->
        <div v-if="loading && files.length === 0" class="flex-center py-40px">
          <NSpin />
        </div>
        <!-- 空状态(无文件且非新建态) -->
        <div v-else-if="files.length === 0 && !diskStore.creatingType" class="flex-col-center py-40px opacity-50">
          <SvgIcon icon="material-symbols:folder-off-outline" class="text-56px" />
          <span class="mt-8px text-14px">{{ $t('page.disk.empty') }}</span>
        </div>
        <!-- 网格(有文件或行内新建时显示) -->
        <div v-else class="grid gap-8px" :style="{ gridTemplateColumns: gridTemplate }">
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
            <FileIcon :file-type="diskStore.creatingType === 'folder' ? 'folder' : 'other'" :size="diskStore.gridSize === 'large' ? 80 : 56" />
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
          <FileCard v-for="f in files" :key="f.fileId" :file="f" @action="onAction" />
        </div>
      </div>
    </NScrollbar>
  </div>
</template>
