<script setup lang="ts">
import { ref, computed, reactive, onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useLoading } from '@sa/hooks';
import { useDialog, useMessage } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { fetchGetFileList, mapBackendFileList, fetchGetQuota, fetchDelete, fetchMove, fetchCopy } from '@/service/api/disk';
import { useInfiniteScroll } from '@/hooks/business/use-infinite-scroll';
import { useDownload } from '@/hooks/business/download';
import { useDiskUpload } from '@/hooks/business/disk/use-disk-upload';
import { useDiskCreate } from '@/hooks/business/disk/use-disk-create';
import { $t } from '@/locales';
import FileTypeMenu from './modules/file-type-menu.vue';
import Toolbar from './modules/toolbar.vue';
import Breadcrumb from './modules/breadcrumb.vue';
import FileGrid from './modules/file-grid.vue';
import FileList from './modules/file-list.vue';
import MoveCopyModal from './modules/move-copy-modal.vue';
import TransferPanel from './modules/transfer-panel.vue';
import DropZone from './modules/drop-zone.vue';

defineOptions({
  name: 'DiskHome'
});

const diskStore = useDiskStore();
const route = useRoute();
const router = useRouter();
const { loading, startLoading, endLoading } = useLoading();
const dialog = useDialog();
const message = useMessage();
const { uploadFiles } = useDiskUpload();
const { direct: downloadLink } = useDownload();
const { beginCreate, beginRename, registerRefresh } = useDiskCreate();

// 显示容量开关
const showCapacity = ref(true);
const quotaLoading = ref(false);

async function loadQuotaInfo() {
  quotaLoading.value = true;
  const { data, error } = await fetchGetQuota();
  if (!error && data) {
    diskStore.updateQuotaInfo(data);
  }
  quotaLoading.value = false;
}

const fileGridRef = ref<InstanceType<typeof FileGrid>>();
const fileListRef = ref<InstanceType<typeof FileList>>();
const totalCount = ref(0);

// === 无限滚动：缓存 + 分页 ===
const PAGE_SIZE = 50;

/** 真实文件累计缓存（无限滚动追加） */
const realFilesCache = ref<Api.Disk.FileItem[]>([]);
const realTotal = ref(0);
const currentRealPage = ref(1);
const hasMore = ref(true);

/** 合并排序后的显示列表（文件夹优先 → sortBy → name ASC，与后端 sortFileListEntries 逻辑一致） */
const fileList = computed(() => mergeAndSort(realFilesCache.value, diskStore.sortSettings));

/** 根据当前视图模式，获取实际滚动容器 */
const scrollContainer = computed<HTMLElement | null>(() => {
  if (diskStore.viewMode === 'grid') {
    return fileGridRef.value?.scrollContainer ?? null;
  }
  return fileListRef.value?.scrollContainer ?? null;
});

/** 搜索关键词缓存 */
const searchKeyword = ref<string | null>(null);

/** 文件列表请求序号：连续搜索/切换目录时丢弃过期响应，防止旧请求覆盖新结果 */
let listRequestId = 0;

/** 合并并排序（文件夹优先 + sortSettings） */
function mergeAndSort(
  files: Api.Disk.FileItem[],
  sortSettings: { field: string | null; order: string | null }
): Api.Disk.FileItem[] {
  const { field, order } = sortSettings;
  const sortOrder = order === 'desc' ? -1 : 1;

  return [...files].sort((a, b) => {
    if (a.isFolder !== b.isFolder) return a.isFolder ? -1 : 1;
    switch (field) {
      case 'name':
        return sortOrder * a.fileName.localeCompare(b.fileName);
      case 'size':
        return sortOrder * (a.fileSize - b.fileSize);
      case 'modifyTime':
        return (
          sortOrder *
          (new Date(a.modifyTime || a.updateTime || '').getTime() -
            new Date(b.modifyTime || b.updateTime || '').getTime())
        );
      default:
        return a.fileName.localeCompare(b.fileName);
    }
  });
}

/** 首次加载：清空缓存，请求第一页 */
async function getFileList() {
  const reqId = ++listRequestId;
  startLoading();

  realFilesCache.value = [];
  currentRealPage.value = 1;
  hasMore.value = true;

  const fileType = diskStore.currentFileType === 'all' ? null : diskStore.currentFileType;

  const { data, error } = await fetchGetFileList({
    pageNum: 1,
    pageSize: PAGE_SIZE,
    fileType,
    keyword: searchKeyword.value,
    parentId: null,
    sortField: diskStore.sortSettings.field,
    sortOrder: diskStore.sortSettings.order
  });

  // 丢弃过期响应（搜索词或目录已变化）
  if (reqId !== listRequestId) {
    endLoading();
    return;
  }

  if (!error && data) {
    const mapped = mapBackendFileList(data);
    realFilesCache.value = mapped.rows;
    totalCount.value = mapped.total;
    realTotal.value = mapped.total;
    hasMore.value = realFilesCache.value.length < realTotal.value;
  } else {
    realFilesCache.value = [];
    totalCount.value = 0;
    hasMore.value = false;
  }

  endLoading();
}

/** 加载更多：追加下一页 */
async function loadMoreFiles() {
  if (!hasMore.value) return;

  const reqId = listRequestId;
  currentRealPage.value++;

  const fileType = diskStore.currentFileType === 'all' ? null : diskStore.currentFileType;

  const { data, error } = await fetchGetFileList({
    pageNum: currentRealPage.value,
    pageSize: PAGE_SIZE,
    fileType,
    keyword: searchKeyword.value,
    parentId: null,
    sortField: diskStore.sortSettings.field,
    sortOrder: diskStore.sortSettings.order
  });

  // 列表已被 getFileList 重置（搜索/切目录），丢弃本次分页响应
  if (reqId !== listRequestId) return;

  if (!error && data) {
    const mapped = mapBackendFileList(data);
    realFilesCache.value.push(...mapped.rows);
    hasMore.value = realFilesCache.value.length < realTotal.value;
  }
}

// 无限滚动 hook
const { loadingMore } = useInfiniteScroll({
  onLoadMore: loadMoreFiles,
  hasMore,
  scrollContainerRef: scrollContainer,
  isLoading: loading
});

function handleSearch(keyword: string) {
  searchKeyword.value = keyword || null;
  getFileList();
}

function handleRefresh() {
  getFileList();
}

/** 工具栏触发上传:传当前目录,每个文件成功后刷新列表;type 区分文件/文件夹。
 *  文件夹模式额外传 dirs(含空目录的目录路径列表),由引擎先调 ensure-folders 预建目录树。 */
function handleUpload(_type: 'file' | 'folder', files: File[], dirs?: string[]) {
  uploadFiles(files, diskStore.getCurrentPathString(), dirs, getFileList);
}

/** 拖拽落下:复用 uploadFiles,上传到当前目录(基础版;与 toolbar 文件夹上传同等机制,含空目录预建) */
function handleDrop(files: { file: File; relativePath?: string }[], dirs: string[]) {
  uploadFiles(files, diskStore.getCurrentPathString(), dirs, getFileList);
}

/** 工具栏触发行内新建 */
function handleCreate(type: 'file' | 'folder') {
  beginCreate(type);
}

function handleSort(field: 'name' | 'size' | 'modifyTime', order: 'asc' | 'desc') {
  diskStore.setSort(field, order);
}

/** 视图模式三选一：list 列表 / thumbnail 缩略(grid small) / large 大图(grid large) */
function handleSetView(mode: 'list' | 'thumbnail' | 'large') {
  if (mode === 'list') {
    diskStore.setViewMode('list');
    return;
  }
  diskStore.setViewMode('grid');
  diskStore.setGridSize(mode === 'large' ? 'large' : 'small');
}

// === 第2期 文件 CRUD(删除;移动/复制 A2) ===
// 注:新建文件夹 / 新建文件 / 重命名 已改为行内输入,逻辑见 use-disk-create。

function confirmDelete(fileIds: Api.Disk.FileItem['fileId'][]) {
  if (fileIds.length === 0) return;
  dialog.warning({
    title: $t('page.disk.action.delete'),
    content: $t('page.disk.msg.deleteConfirmContent', { count: fileIds.length }),
    positiveText: $t('page.disk.modal.confirm'),
    negativeText: $t('page.disk.modal.cancel'),
    onPositiveClick: async () => {
      const { error } = await fetchDelete({ fileIds });
      if (error) {
        message.error($t('page.disk.msg.operateFail'));
        return;
      }
      message.success($t('page.disk.msg.deleteSuccess'));
      diskStore.clearSelection();
      getFileList();
    }
  });
}

/** 移动/复制弹窗 */
const moveCopyModal = reactive({
  visible: false,
  mode: 'move' as 'move' | 'copy',
  sources: [] as Api.Disk.FileItem[]
});

function openMoveCopy(files: Api.Disk.FileItem[], mode: 'move' | 'copy') {
  if (files.length === 0) return;
  moveCopyModal.mode = mode;
  moveCopyModal.sources = files;
  moveCopyModal.visible = true;
}

async function doMoveCopy(targetPath: string, conflict?: 'overwrite' | 'rename') {
  const sources = moveCopyModal.sources;
  if (sources.length === 0) return;
  const base = { fileIds: sources.map(f => f.fileId), targetPath };
  const req = conflict ? { ...base, conflict } : base;
  const { data, error } = moveCopyModal.mode === 'move' ? await fetchMove(req) : await fetchCopy(req);
  if (error) {
    return; // 拦截器已提示错误,不再重复弹"操作失败"(消除双提示)
  }
  if (data?.conflict) {
    // 命中同名:弹选择框。覆盖=整体替换(目标项进回收站),保留两者=加序号;关闭/取消=不做任何操作。
    // 第二次调用带 conflict,后端执行策略不再返回 conflict。
    dialog.warning({
      title: $t('page.disk.msg.conflictTitle'),
      content: $t('page.disk.msg.conflictDesc', { names: data.conflictFiles.join(', ') }),
      positiveText: $t('page.disk.msg.overwrite'),
      negativeText: $t('page.disk.msg.keepBoth'),
      onPositiveClick: () => doMoveCopy(targetPath, 'overwrite'),
      onNegativeClick: () => doMoveCopy(targetPath, 'rename')
    });
    return;
  }
  message.success(moveCopyModal.mode === 'move' ? $t('page.disk.msg.moveSuccess') : $t('page.disk.msg.copySuccess'));
  moveCopyModal.visible = false;
  diskStore.clearSelection();
  getFileList();
}

function confirmMoveCopy(targetPath: string) {
  doMoveCopy(targetPath);
}

/** 文件项操作菜单分发 */
function handleAction(type: Api.Disk.DiskActionType, file: Api.Disk.FileItem) {
  switch (type) {
    case 'rename':
      beginRename(file);
      break;
    case 'delete':
      confirmDelete([file.fileId]);
      break;
    case 'move':
      openMoveCopy([file], 'move');
      break;
    case 'copy':
      openMoveCopy([file], 'copy');
      break;
    case 'download':
      if (file.isFolder) {
        // 目录 → 打包下载(递归后代流式 Zip)
        downloadLink(`/file-meta/package-download?fileIds=${file.id ?? ''}`, `${file.name ?? 'download'}.zip`);
      } else {
        // 文件 → 原生单文件下载(GET + httpOnly cookie 鉴权,浏览器接管,大文件零内存可续传)
        downloadLink(`/file-meta/download?fileId=${file.id ?? ''}`, file.name ?? '');
      }
      break;
    default:
      break;
  }
}

/** 批量下载选中项(打包,流式 Zip) */
function handleBatchDownload() {
  const ids = diskStore.selectedFiles;
  if (ids.length === 0) return;
  downloadLink(`/file-meta/package-download?${ids.map(id => `fileIds=${id}`).join('&')}`, `disk-download-${ids.length}.zip`);
}

/** 批量删除选中项 */
function handleBatchDelete() {
  confirmDelete(diskStore.selectedFiles);
}

/** 批量重命名:仅单选可用,从当前列表反查 FileItem 进入行内重命名 */
function handleBatchRename() {
  if (diskStore.selectedFiles.length !== 1) return;
  const target = diskStore.currentFileList.find(f => f.fileId === diskStore.selectedFiles[0]);
  if (target) beginRename(target);
}

/** 批量移动/复制:对全部选中项 */
function handleBatchMoveCopy(mode: 'move' | 'copy') {
  const files = diskStore.currentFileList.filter(f => diskStore.selectedFiles.includes(f.fileId));
  openMoveCopy(files, mode);
}

// 同步 fileList 到 diskStore（供其他组件读取 currentFileList）
watch(
  fileList,
  list => {
    diskStore.currentFileList = list;
  },
  { deep: false }
);

// 文件类型变化 → 重新加载
watch(
  () => diskStore.currentFileType,
  () => getFileList()
);

// 目录变化 → 重新加载
watch(
  () => diskStore.currentParentId,
  () => getFileList()
);

// 排序变化 → 重新加载
watch(
  () => diskStore.sortSettings,
  () => getFileList(),
  { deep: true }
);

// 浏览器前进/后退：URL path 变化恢复目录
watch(
  () => route.query.path,
  async newPath => {
    // route.query.path 已由 vue-router 的 parseQuery 解码,无需再 decodeURIComponent
    const pathStr = (newPath as string) || '/';
    if (pathStr !== diskStore.getCurrentPathString()) {
      const success = await diskStore.restoreFromPath(pathStr);
      if (!success && pathStr !== '/') {
        router.replace({ name: 'disk' });
      }
      getFileList();
    }
  }
);

onMounted(async () => {
  const pathParam = route.query.path as string;
  if (pathParam) {
    const success = await diskStore.restoreFromPath(pathParam);
    if (!success) {
      router.replace({ name: 'disk' });
    }
  }
  registerRefresh(getFileList);
  getFileList();
  loadQuotaInfo();
});
</script>

<template>
  <TableSiderLayout sider-title="文件管理">
    <template #header-extra>
      <NTooltip trigger="hover">
        <template #trigger>
          <NSwitch v-model:value="showCapacity" :round="false" />
        </template>
        {{ $t('page.disk.toolbar.showCapacity') }}
      </NTooltip>
    </template>
    <template #sider>
      <FileTypeMenu :show-capacity="showCapacity" :quota-info="diskStore.quotaInfo" :quota-loading="quotaLoading" />
    </template>
    <div class="h-full flex-col-stretch gap-12px overflow-hidden">
      <NCard
        :bordered="false"
        size="small"
        class="card-wrapper flex-1-hidden"
        :content-style="{ padding: 0, height: '100%', display: 'flex', flexDirection: 'column' }"
      >
        <!-- 工具栏 -->
        <Toolbar
          @search="handleSearch"
          @refresh="handleRefresh"
          @sort="handleSort"
          @set-view="handleSetView"
          @create="handleCreate"
          @upload="handleUpload"
          @batch-download="handleBatchDownload"
          @batch-delete="handleBatchDelete"
          @batch-rename="handleBatchRename"
          @batch-move="handleBatchMoveCopy('move')"
          @batch-copy="handleBatchMoveCopy('copy')"
        />
        <!-- 面包屑 -->
        <Breadcrumb v-if="fileList.length > 0 || diskStore.currentPath.length > 0" :total-count="totalCount" />
        <!-- 文件内容 -->
        <FileGrid
          v-if="diskStore.viewMode === 'grid'"
          ref="fileGridRef"
          :files="fileList"
          :loading="loading"
          class="flex-1 min-h-0"
          @action="handleAction"
          @refresh="handleRefresh"
        />
        <FileList
          v-else
          ref="fileListRef"
          :files="fileList"
          :loading="loading"
          class="flex-1 min-h-0"
          @action="handleAction"
          @refresh="handleRefresh"
        />
        <!-- 上传传输面板(有任务时显示) -->
        <TransferPanel />
        <!-- 加载更多状态 -->
        <div v-if="loadingMore" class="flex-center gap-8px py-12px">
          <NSpin size="small" />
          <span class="text-13px opacity-60">{{ $t('page.disk.loadingMore') }}</span>
        </div>
      </NCard>

      <!-- 拖拽上传落区(document 监听 + 全屏蒙层,仅本页 mount 时生效) -->
      <DropZone @drop="handleDrop" />

      <!-- 移动/复制 目录树选择弹窗 -->
      <MoveCopyModal
        v-model:visible="moveCopyModal.visible"
        :mode="moveCopyModal.mode"
        :sources="moveCopyModal.sources"
        @confirm="confirmMoveCopy"
      />
    </div>
  </TableSiderLayout>
</template>

<style scoped lang="scss">
:deep(.n-card__content) {
  padding: 0 !important;
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>
