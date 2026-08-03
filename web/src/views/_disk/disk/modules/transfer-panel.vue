<script setup lang="ts">
import { computed, ref } from 'vue';
import { useDiskUpload } from '@/hooks/business/disk/use-disk-upload';
import { useAppStore } from '@/store/modules/app';
import { $t } from '@/locales';
import { formatFileSize } from '@/utils/format';
import TransferSphere from './transfer-sphere.vue';
import FileIcon from './file-icon.vue';

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

const ACTIVE: Api.Disk.UploadTaskStatus[] = ['hashing', 'uploading', 'merging'];
/** 顶层展示项:文件夹聚合条目 + 无 folderId 的独立文件;子文件(有 folderId 非聚合)折叠在聚合内 */
const topTasks = computed(() => tasks.value.filter(t => t.isFolderAgg || !t.folderId));
const activeCount = computed(() => topTasks.value.filter(t => ACTIVE.includes(t.status) || t.status === 'paused').length);
const overallProgress = computed(() => {
  const list = topTasks.value;
  if (!list.length) return 0;
  return Math.round(list.reduce((s, t) => s + (t.progress || 0), 0) / list.length);
});
const allDone = computed(() => topTasks.value.length > 0 && topTasks.value.every(t => t.status === 'success'));
/** 是否有可清空的已完成/失败任务(决定"清空已完成"按钮显隐,无可清则隐藏) */
const hasFinished = computed(() => tasks.value.some(t => t.status === 'success' || t.status === 'error'));

/** 展开的文件夹聚合 id 集合 */
const expandedFolders = ref<Set<string>>(new Set());
function toggleFolder(id: string) {
  const next = new Set(expandedFolders.value);
  if (next.has(id)) next.delete(id);
  else next.add(id);
  expandedFolders.value = next;
}
function isFolderExpanded(id: string) {
  return expandedFolders.value.has(id);
}
/** 聚合条目的子文件列表(按 folderId 过滤) */
function folderChildren(folderId: string) {
  return tasks.value.filter(t => t.folderId === folderId && !t.isFolderAgg);
}

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
      return 'rgb(var(--success-color))';
    case 'error':
      return 'rgb(var(--error-color))';
    case 'paused':
      return 'rgb(var(--base-text-color) / 0.5)'; // 暂停:中性灰,亮暗自适应
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

/** 从文件名提取扩展名(不含点号,小写);无扩展名返回空串,交 FileIcon 回退默认图标 */
function extOf(name: string): string {
  const matched = name.match(/\.([^.]+)$/);
  return matched ? matched[1].toLowerCase() : '';
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
            <span class="opacity-50">{{ activeCount }}/{{ topTasks.length }}</span>
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
            <NButton v-if="hasFinished" text size="tiny" :focusable="false" @click="clearFinished">{{ $t('page.disk.transfer.clear') }}</NButton>
            <NButton text size="tiny" :focusable="false" @click="hidePanel">
              <SvgIcon icon="material-symbols:close" class="text-16px" />
            </NButton>
          </div>
        </div>

        <!-- 列表内容 -->
        <div class="disk-transfer-scroll max-h-460px overflow-auto px-12px pb-8px">
          <template v-for="t in topTasks" :key="t.id">
            <!-- 文件夹聚合条目:汇总同顶层 relativePath 的子文件,点击展开子文件列表 -->
            <div v-if="t.isFolderAgg" class="mb-6px rd-6px px-6px py-8px transition-colors hover:bg-primary/10">
              <!-- 第1行:文件夹图标+展开箭头+名+完成数/总数 | 百分比+操作 -->
              <div class="flex items-center justify-between gap-8px">
                <div class="flex min-w-0 flex-1 cursor-pointer items-center gap-4px" @click="toggleFolder(t.id)">
                  <FileIcon file-type="folder" :size="20" />
                  <SvgIcon
                    :icon="isFolderExpanded(t.id) ? 'material-symbols:arrow-drop-down' : 'material-symbols:arrow-right'"
                    class="shrink-0 text-16px opacity-50"
                  />
                  <span class="max-w-200px truncate text-13px font-500" :title="t.name">{{ t.name }}</span>
                  <span class="shrink-0 text-12px tabular-nums opacity-50">{{ t.completedCount }}/{{ t.totalCount }}</span>
                </div>
                <div class="flex shrink-0 items-center gap-2px">
                  <span class="mr-4px text-13px font-600 tabular-nums whitespace-nowrap" :style="{ color: statusColor(t.status) }">
                    {{ t.status === 'error' ? statusText(t.status) : t.status === 'success' ? '100%' : t.status === 'pending' ? statusText(t.status) : `${t.progress}%` }}
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
              <!-- 进度条(聚合加权) -->
              <div class="mt-4px h-6px overflow-hidden rd-3px" style="background: rgb(var(--base-text-color) / 0.12)">
                <div
                  class="h-full rd-3px"
                  :style="{ width: `${t.progress}%`, backgroundColor: statusColor(t.status), transition: 'width 0.2s linear, background-color 0.3s ease' }"
                />
              </div>
              <!-- 详情行:聚合已传/总大小 + 速率;错误整行显示原因 -->
              <div
                v-if="isActive(t.status) || t.status === 'paused' || t.status === 'error' || t.status === 'pending'"
                class="mt-2px flex items-center justify-between gap-8px text-12px opacity-60"
              >
                <span v-if="t.status === 'error'" class="min-w-0 flex-1 truncate" style="color: rgb(var(--error-color))" :title="t.errorMsg">
                  {{ t.errorMsg || statusText(t.status) }}
                </span>
                <span v-else-if="t.status === 'pending'" class="tabular-nums whitespace-nowrap" :style="{ color: statusColor(t.status) }">
                  {{ statusText(t.status) }}
                </span>
                <template v-else>
                  <span class="tabular-nums">{{ formatFileSize(t.folderTransferredSize || 0) }} / {{ formatFileSize(t.folderTotalSize || 0) }}</span>
                  <div class="flex shrink-0 items-center gap-8px">
                    <span v-if="t.status === 'uploading' && t.speed" class="tabular-nums" style="color: rgb(var(--primary-color))">
                      {{ formatFileSize(t.speed) }}/s
                    </span>
                    <span v-else-if="t.status !== 'uploading'" class="tabular-nums whitespace-nowrap" :style="{ color: statusColor(t.status) }">
                      {{ statusText(t.status) }}
                    </span>
                  </div>
                </template>
              </div>
              <!-- 展开子文件列表(内部滚动,防长列表撑爆面板) -->
              <div v-if="isFolderExpanded(t.id)" class="disk-folder-children mt-6px max-h-200px overflow-auto pl-24px">
                <div v-for="c in folderChildren(t.id)" :key="c.id" class="flex items-center gap-6px py-2px text-12px">
                  <FileIcon file-type="other" :extension="extOf(c.name)" :size="16" />
                  <span class="min-w-0 flex-1 truncate" :title="c.name">{{ c.name }}</span>
                  <span class="shrink-0 tabular-nums opacity-60" :style="{ color: statusColor(c.status) }">
                    {{ c.status === 'error' ? statusText(c.status) : c.status === 'success' ? '100%' : `${c.progress}%` }}
                  </span>
                </div>
              </div>
            </div>
            <!-- 独立文件行 -->
            <div v-else class="mb-6px rd-6px px-6px py-8px transition-colors hover:bg-primary/10">
              <!-- 第1行:图标+名 | 百分比+操作 -->
              <div class="flex items-center justify-between gap-8px">
                <div class="flex min-w-0 items-center gap-8px">
                  <FileIcon file-type="other" :extension="extOf(t.name)" :size="18" />
                  <span class="max-w-280px truncate text-13px" :title="t.name">{{ t.name }}</span>
                </div>
                <div class="flex shrink-0 items-center gap-2px">
                  <span class="mr-4px text-13px font-600 tabular-nums whitespace-nowrap" :style="{ color: statusColor(t.status) }">
                    {{
                      t.status === 'error'
                        ? statusText(t.status)
                        : t.status === 'success'
                          ? '100%'
                          : t.status === 'pending' || t.status === 'hashing' || t.status === 'merging'
                            ? statusText(t.status)
                            : `${t.progress}%`
                    }}
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
              <div class="mt-4px h-6px overflow-hidden rd-3px" style="background: rgb(var(--base-text-color) / 0.12)">
                <!-- 不定进度条:hashing/merging 阶段无字节进展(采样哈希在主线程/合并推OSS),用滑动动画提示进行中 -->
                <div
                  v-if="t.status === 'hashing' || t.status === 'merging'"
                  class="disk-progress-indeterminate h-full rd-3px"
                  :style="{ backgroundColor: statusColor(t.status) }"
                />
                <div
                  v-else
                  class="h-full rd-3px"
                  :style="{ width: `${t.progress}%`, backgroundColor: statusColor(t.status), transition: 'width 0.2s linear, background-color 0.3s ease' }"
                />
              </div>
              <!-- 详情行:进行中/暂停显示已传/总+速率;错误整行显示失败原因(超长省略,悬浮看全文) -->
              <div
                v-if="isActive(t.status) || t.status === 'paused' || t.status === 'error' || t.status === 'pending'"
                class="mt-2px flex items-center justify-between gap-8px text-12px opacity-60"
              >
                <span
                  v-if="t.status === 'error'"
                  class="min-w-0 flex-1 truncate"
                  style="color: rgb(var(--error-color))"
                  :title="t.errorMsg"
                >{{ t.errorMsg || statusText(t.status) }}</span>
                <span
                  v-else-if="t.status === 'pending'"
                  class="tabular-nums whitespace-nowrap"
                  :style="{ color: statusColor(t.status) }"
                >{{ statusText(t.status) }}</span>
                <template v-else>
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
                </template>
              </div>
            </div>
          </template>

          <!-- 全局操作 -->
          <div v-if="activeCount > 1" class="disk-divider mt-4px flex items-center justify-between border-t px-6px py-8px">
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
  /* 用主题变量:container-bg/base-text 亮暗自动切换,无需 :global(.dark) 补丁 */
  color: rgb(var(--base-text-color));
  background-color: rgb(var(--container-bg-color) / 0.92);
  backdrop-filter: blur(16px);
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.24);
  border: 1px solid rgb(var(--base-text-color) / 0.12);
}
/* 全局操作行分隔线:border-line 非项目预设 class(原不生效),改用主题变量自适应亮暗 */
.disk-divider {
  border-color: rgb(var(--base-text-color) / 0.1);
}
.disk-transfer-scroll::-webkit-scrollbar {
  width: 4px;
}
.disk-transfer-scroll::-webkit-scrollbar-thumb {
  background: rgb(var(--primary-color) / 0.3);
  border-radius: 2px;
}
/* 文件夹展开子文件列表滚动条(更细更淡,与主列表区分) */
.disk-folder-children::-webkit-scrollbar {
  width: 3px;
}
.disk-folder-children::-webkit-scrollbar-thumb {
  background: rgb(var(--primary-color) / 0.2);
  border-radius: 2px;
}
/* 不定进度条滑块:hashing/merging 阶段循环滑动提示进行中 */
.disk-progress-indeterminate {
  position: relative;
  width: 100%;
}
.disk-progress-indeterminate::after {
  content: '';
  position: absolute;
  inset: 0;
  width: 35%;
  border-radius: 3px;
  background-color: inherit;
  animation: disk-indeterminate 1.1s ease-in-out infinite;
}
@keyframes disk-indeterminate {
  0% {
    left: -35%;
  }
  100% {
    left: 100%;
  }
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
