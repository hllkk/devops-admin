<script setup lang="ts">
import { computed, ref } from 'vue';
import { useDiskUpload } from '@/hooks/business/disk/use-disk-upload';
import { useAppStore } from '@/store/modules/app';
import { $t } from '@/locales';
import { formatFileSize } from '@/utils/format';
import TransferSphere from './transfer-sphere.vue';

defineOptions({ name: 'TransferPanel' });

const {
  tasks,
  panelVisible,
  pauseTask,
  resumeTask,
  retryTask,
  cancelTask,
  pauseAll,
  resumeAll,
  clearFinished,
  hidePanel
} = useDiskUpload();
const appStore = useAppStore();

// 视图:移动端默认 sphere,桌面默认 list
const viewMode = ref<'list' | 'sphere'>(appStore.isMobile ? 'sphere' : 'list');

/** 球体 ↔ 列表切换:点工具栏图标收成球,点球体展开为列表 */
function toggleViewMode() {
  viewMode.value = viewMode.value === 'sphere' ? 'list' : 'sphere';
}

const ACTIVE: Api.Disk.UploadTaskStatus[] = ['pending', 'hashing', 'uploading', 'merging'];
const activeCount = computed(() => tasks.value.filter(t => ACTIVE.includes(t.status) || t.status === 'paused').length);
const overallProgress = computed(() => {
  const list = tasks.value;
  if (!list.length) return 0;
  return Math.round(list.reduce((s, t) => s + (t.progress || 0), 0) / list.length);
});
const allDone = computed(() => tasks.value.length > 0 && tasks.value.every(t => t.status === 'success'));

const statusText = (s: Api.Disk.UploadTaskStatus): string =>
  (
    {
      pending: $t('page.disk.transfer.pending'),
      hashing: $t('page.disk.transfer.hashing'),
      uploading: $t('page.disk.transfer.uploading'),
      paused: $t('page.disk.transfer.paused'),
      merging: $t('page.disk.transfer.merging'),
      success: $t('page.disk.transfer.success'),
      error: $t('page.disk.transfer.error')
    } as Record<Api.Disk.UploadTaskStatus, string>
  )[s];

function statusColor(s: Api.Disk.UploadTaskStatus): string {
  switch (s) {
    case 'success':
      return '#18a058';
    case 'error':
      return '#d03050';
    case 'paused':
      return '#909399';
    default:
      return 'rgb(var(--primary-color))';
  }
}
const isActive = (s: Api.Disk.UploadTaskStatus) => ACTIVE.includes(s);

/** 剩余秒数 → 可读时长(<60s 显示 Ns,否则 m:ss 或 NhNm) */
function formatEta(sec?: number): string {
  if (!sec || !isFinite(sec)) return '--';
  if (sec < 60) return `${Math.ceil(sec)}s`;
  if (sec < 3600) {
    const m = Math.floor(sec / 60);
    const s = Math.round(sec % 60);
    return `${m}:${String(s).padStart(2, '0')}`;
  }
  const h = Math.floor(sec / 3600);
  const m = Math.round((sec % 3600) / 60);
  return `${h}h${m}m`;
}
</script>

<template>
  <Teleport to="body">
    <Transition name="disk-panel" mode="out-in">
      <!-- 水波纹球视图:纯球体,无面板壳,点击展开为列表 -->
      <div
        v-if="panelVisible && viewMode === 'sphere'"
        key="sphere"
        class="fixed bottom-20px right-20px z-1000 cursor-pointer transition-transform hover:scale-105 lt-sm:right-10px lt-sm:bottom-10px"
        :title="$t('page.disk.toolbar.listView')"
        @click="toggleViewMode"
      >
        <TransferSphere :progress="overallProgress" :done="allDone" />
      </div>
      <!-- 列表视图:完整传输面板 -->
      <div
        v-else-if="panelVisible"
        key="list"
        class="disk-transfer-panel fixed bottom-20px right-20px z-1000 flex flex-col rd-12px lt-sm:right-10px lt-sm:bottom-10px lt-sm:w-340px"
        :style="{ width: appStore.isMobile ? '340px' : 'min(560px, 92vw)' }"
      >
        <!-- 头部 -->
        <div class="flex-y-center justify-between px-12px py-8px">
          <span class="text-13px font-500">
            {{ $t('page.disk.transfer.title') }}
            <span class="opacity-50">{{ activeCount }}/{{ tasks.length }}</span>
          </span>
          <div class="flex items-center gap-4px">
            <!-- 收起为水波纹球 -->
            <NTooltip v-if="!appStore.isMobile" trigger="hover">
              <template #trigger>
                <NButton quaternary size="tiny" :focusable="false" @click="toggleViewMode">
                  <SvgIcon icon="ep:position" class="text-14px" />
                </NButton>
              </template>
              {{ $t('page.disk.transfer.sphereView') }}
            </NTooltip>
            <NButton text size="tiny" :focusable="false" @click="clearFinished">{{ $t('page.disk.transfer.clear') }}</NButton>
            <NButton text size="tiny" :focusable="false" @click="hidePanel">
              <SvgIcon icon="material-symbols:close" class="text-16px" />
            </NButton>
          </div>
        </div>

        <!-- 列表内容 -->
        <div class="disk-transfer-scroll max-h-460px overflow-auto px-12px pb-8px">
          <div v-for="t in tasks" :key="t.id" class="mb-6px rd-6px px-6px py-8px transition-colors hover:bg-primary/5">
            <!-- 第1行:图标+名 | 百分比+操作 -->
            <div class="flex items-center justify-between gap-8px">
              <div class="flex min-w-0 items-center gap-8px">
                <SvgIcon
                  :icon="t.status === 'success' ? 'material-symbols:check-circle' : 'material-symbols:insert-drive-file'"
                  class="shrink-0 text-16px"
                  :style="{ color: statusColor(t.status) }"
                />
                <span class="max-w-280px truncate text-13px" :title="t.name">{{ t.name }}</span>
              </div>
              <div class="flex shrink-0 items-center gap-2px">
                <span class="mr-4px text-13px font-600 tabular-nums" :style="{ color: statusColor(t.status) }">
                  {{ t.status === 'error' ? statusText(t.status) : t.status === 'success' ? '100%' : `${t.progress}%` }}
                </span>
                <NButton v-if="isActive(t.status)" text size="tiny" :focusable="false" :title="$t('page.disk.transfer.pause')" @click="pauseTask(t.id)">
                  <SvgIcon icon="material-symbols:pause" class="text-16px" />
                </NButton>
                <NButton v-if="t.status === 'paused'" text size="tiny" :focusable="false" :title="$t('page.disk.transfer.resume')" @click="resumeTask(t.id)">
                  <SvgIcon icon="material-symbols:play-arrow" class="text-16px" />
                </NButton>
                <NButton v-if="t.status === 'error'" text size="tiny" :focusable="false" :title="$t('page.disk.transfer.retry')" @click="retryTask(t.id)">
                  <SvgIcon icon="material-symbols:refresh" class="text-16px" />
                </NButton>
                <NButton text size="tiny" :focusable="false" :title="$t('page.disk.transfer.cancel')" @click="cancelTask(t.id)">
                  <SvgIcon icon="material-symbols:close" class="text-16px" />
                </NButton>
              </div>
            </div>
            <!-- 进度条 -->
            <div class="mt-4px h-6px overflow-hidden rd-3px" style="background: rgb(var(--primary-color) / 0.1)">
              <div
                class="h-full rd-3px"
                :style="{ width: `${t.progress}%`, backgroundColor: statusColor(t.status), transition: 'width 0.2s linear, background-color 0.3s ease' }"
              />
            </div>
            <!-- 详情行:左侧已传输/总大小;右侧上传中展示速率+剩余时间,其它进行中状态展示状态文字 -->
            <div v-if="isActive(t.status) || t.status === 'paused'" class="mt-2px flex items-center justify-between text-12px opacity-60">
              <span class="tabular-nums">{{ formatFileSize(t.transferredSize || 0) }} / {{ formatFileSize(t.size) }}</span>
              <div class="flex shrink-0 items-center gap-8px">
                <!-- 上传中(继续下载):展示速率与剩余时间 -->
                <template v-if="t.status === 'uploading'">
                  <span v-if="t.speed" class="tabular-nums" style="color: rgb(var(--primary-color))">
                    {{ formatFileSize(t.speed) }}/s
                  </span>
                  <span v-if="t.remainingTime" class="tabular-nums whitespace-nowrap">
                    {{ $t('page.disk.transfer.remaining', { v: formatEta(t.remainingTime) }) }}
                  </span>
                </template>
                <!-- 其它状态:展示状态文字(已暂停/合并中/计算中/等待) -->
                <span v-else class="tabular-nums whitespace-nowrap" :style="{ color: statusColor(t.status) }">
                  {{ statusText(t.status) }}
                </span>
              </div>
            </div>
          </div>

          <!-- 全局操作 -->
          <div v-if="activeCount > 1" class="mt-4px flex items-center justify-between border-t border-line px-6px py-8px">
            <span class="text-12px opacity-60">{{ $t('page.disk.transfer.taskCount', { n: activeCount }) }}</span>
            <div class="flex gap-8px">
              <NButton text size="tiny" :focusable="false" @click="pauseAll">{{ $t('page.disk.transfer.pauseAll') }}</NButton>
              <NButton text size="tiny" :focusable="false" @click="resumeAll">{{ $t('page.disk.transfer.resumeAll') }}</NButton>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.disk-transfer-panel {
  background-color: var(--n-color, rgba(255, 255, 255, 0.92));
  backdrop-filter: blur(16px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.18);
  border: 1px solid rgb(var(--primary-color) / 0.15);
}
:global(.dark) .disk-transfer-panel {
  background-color: rgba(30, 30, 30, 0.92);
}
.disk-transfer-scroll::-webkit-scrollbar {
  width: 4px;
}
.disk-transfer-scroll::-webkit-scrollbar-thumb {
  background: rgb(var(--primary-color) / 0.3);
  border-radius: 2px;
}
.disk-panel-enter-active,
.disk-panel-leave-active {
  transition:
    opacity 0.2s ease,
    transform 0.2s ease;
}
.disk-panel-enter-from,
.disk-panel-leave-to {
  opacity: 0;
  transform: translateY(20px);
}
</style>
