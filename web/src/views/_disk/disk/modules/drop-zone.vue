<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue';
import { useMessage } from 'naive-ui';
import { collectDropEntries } from '@/utils/disk';
import { $t } from '@/locales';

defineOptions({ name: 'DiskDropZone' });

interface DroppedFile {
  file: File;
  relativePath?: string;
}
interface Emits {
  (e: 'drop', files: DroppedFile[], dirs: string[]): void;
}
const emit = defineEmits<Emits>();

const message = useMessage();

/** 拖拽悬停(显示蒙层) */
const isDragging = ref(false);
/** 延迟隐藏定时器:dragenter/dragover 时 clear,dragleave 时 100ms 后隐藏,避免子元素间 dragleave 闪烁 */
let hideTimer: ReturnType<typeof setTimeout> | null = null;

/** 拖拽上限:防海量小文件全量读入 OOM。数量/体积与 remote 对齐(5000 / 50GB) */
const MAX_DROP_COUNT = 5000;
const MAX_DROP_TOTAL_SIZE = 50 * 1024 ** 3; // 50GB

/** 仅响应"文件"类拖拽(过滤选中文本/链接等非文件拖拽) */
function isFilesDrag(e: DragEvent): boolean {
  return !!e.dataTransfer && Array.from(e.dataTransfer.types || []).includes('Files');
}

function showOverlay() {
  if (hideTimer) {
    clearTimeout(hideTimer);
    hideTimer = null;
  }
  isDragging.value = true;
}
function hideOverlay() {
  if (hideTimer) clearTimeout(hideTimer);
  hideTimer = setTimeout(() => {
    isDragging.value = false;
    hideTimer = null;
  }, 100);
}

function onDragEnter(e: DragEvent) {
  if (!isFilesDrag(e)) return;
  e.preventDefault();
  showOverlay();
}
function onDragOver(e: DragEvent) {
  if (!isFilesDrag(e)) return;
  e.preventDefault(); // 必须 preventDefault,否则 drop 事件不触发(浏览器会直接打开文件)
  if (e.dataTransfer) e.dataTransfer.dropEffect = 'copy';
  showOverlay();
}
function onDragLeave(e: DragEvent) {
  if (!isFilesDrag(e)) return;
  e.preventDefault();
  hideOverlay();
}

async function onDrop(e: DragEvent) {
  if (!isFilesDrag(e)) return;
  e.preventDefault();
  e.stopPropagation();
  isDragging.value = false;
  if (hideTimer) {
    clearTimeout(hideTimer);
    hideTimer = null;
  }
  const dt = e.dataTransfer;
  if (!dt) return;

  // 命门:必须在 drop 同步期内、任何 await 之前把所有 entry 快照出来。
  // 一旦 await 让出执行权,浏览器清空拖拽数据存储,后续 webkitGetAsEntry() 返回 null
  // → 多文件/文件夹拖拽只产生一个任务的经典 bug。FileSystemEntry 是稳定引用,快照后可安全异步读取。
  const items = dt.items ? Array.from(dt.items) : [];
  const entries: FileSystemEntry[] = [];
  if (items.length > 0 && typeof items[0].webkitGetAsEntry === 'function') {
    for (const item of items) {
      const entry = item.webkitGetAsEntry?.();
      if (entry) entries.push(entry);
    }
  }

  if (entries.length > 0) {
    const { files, dirs, truncated } = await collectDropEntries(entries, {
      maxCount: MAX_DROP_COUNT,
      maxSize: MAX_DROP_TOTAL_SIZE
    });
    if (truncated === 'count') {
      message.warning($t('page.disk.dropZone.truncatedCount', { max: MAX_DROP_COUNT }));
    } else if (truncated === 'size') {
      message.warning($t('page.disk.dropZone.truncatedSize'));
    }
    if (files.length > 0) emit('drop', files, dirs);
    return;
  }

  // 降级:webkitGetAsEntry 不支持(老浏览器) → 直接用 dataTransfer.files(无目录层级,均为散文件)
  const files = Array.from(dt.files || []).map(file => ({ file }));
  if (files.length > 0) emit('drop', files, []);
}

onMounted(() => {
  document.addEventListener('dragenter', onDragEnter);
  document.addEventListener('dragover', onDragOver);
  document.addEventListener('dragleave', onDragLeave);
  document.addEventListener('drop', onDrop);
});
onBeforeUnmount(() => {
  document.removeEventListener('dragenter', onDragEnter);
  document.removeEventListener('dragover', onDragOver);
  document.removeEventListener('dragleave', onDragLeave);
  document.removeEventListener('drop', onDrop);
  if (hideTimer) clearTimeout(hideTimer);
});
</script>

<template>
  <Teleport to="body">
    <Transition name="drop-zone-overlay">
      <div
        v-if="isDragging"
        class="fixed inset-0 z-9999 flex-center pointer-events-none bg-[var(--primary-color)]/8 backdrop-blur-4px"
      >
        <div class="flex flex-col items-center gap-12px">
          <SvgIcon icon="material-symbols:cloud-upload" class="text-64px text-[var(--primary-color)]" />
          <span class="text-18px font-500 text-[var(--color)]">{{ $t('page.disk.dropZone.hint') }}</span>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped lang="scss">
/* 蒙层淡入淡出 */
.drop-zone-overlay-enter-active,
.drop-zone-overlay-leave-active {
  transition: opacity 0.15s ease;
}
.drop-zone-overlay-enter-from,
.drop-zone-overlay-leave-to {
  opacity: 0;
}
</style>
