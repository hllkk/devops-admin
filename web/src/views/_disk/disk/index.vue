<script setup lang="ts">
import { ref, computed, reactive, onMounted, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useLoading } from '@sa/hooks';
import { useDialog, useMessage } from 'naive-ui';
import { useDiskStore } from '@/store/modules/disk';
import { fetchGetFileList, mapBackendFileList, fetchGetQuota, fetchDelete, fetchMove, fetchCopy } from '@/service/api/disk';
import { useInfiniteScroll } from '@/hooks/business/use-infinite-scroll';
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
function handleUpload(type: 'file' | 'folder', files: File[], dirs?: string[]) {
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

function confirmDelete(file: Api.Disk.FileItem) {
  dialog.warning({
    title: $t('page.disk.action.delete'),
    content: $t('page.disk.msg.deleteConfirmContent', { count: 1 }),
    positiveText: $t('page.disk.modal.confirm'),
    negativeText: $t('page.disk.modal.cancel'),
    onPositiveClick: async () => {
      const { error } = await fetchDelete({ fileIds: [file.fileId] });
      if (error) {
        message.error($t('page.disk.msg.operateFail'));
        return;
      }
      message.success($t('page.disk.msg.deleteSuccess'));
      getFileList();
    }
  });
}

/** 移动/复制弹窗 */
const moveCopyModal = reactive({
  visible: false,
  mode: 'move' as 'move' | 'copy',
  source: null as Api.Disk.FileItem | null
});

function openMoveCopy(file: Api.Disk.FileItem, mode: 'move' | 'copy') {
  moveCopyModal.mode = mode;
  moveCopyModal.source = file;
  moveCopyModal.visible = true;
}

async function confirmMoveCopy(targetPath: string) {
  const source = moveCopyModal.source;
  if (!source) return;
  const params = { fileIds: [source.fileId], targetPath };
  const { error } = moveCopyModal.mode === 'move' ? await fetchMove(params) : await fetchCopy(params);
  if (error) {
    message.error($t('page.disk.msg.operateFail'));
    return;
  }
  message.success(moveCopyModal.mode === 'move' ? $t('page.disk.msg.moveSuccess') : $t('page.disk.msg.copySuccess'));
  moveCopyModal.visible = false;
  getFileList();
}

/** 文件项操作菜单分发 */
function handleAction(type: Api.Disk.DiskActionType, file: Api.Disk.FileItem) {
  switch (type) {
    case 'rename':
      beginRename(file);
      break;
    case 'delete':
      confirmDelete(file);
      break;
    case 'move':
      openMoveCopy(file, 'move');
      break;
    case 'copy':
      openMoveCopy(file, 'copy');
      break;
    default:
      break;
  }
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
    const decodedNewPath = newPath ? decodeURIComponent(newPath as string) : '/';
    if (decodedNewPath !== diskStore.getCurrentPathString()) {
      const success = await diskStore.restoreFromPath(decodedNewPath);
      if (!success && decodedNewPath !== '/') {
        router.replace({ name: 'disk' });
      }
      getFileList();
    }
  }
);

onMounted(async () => {
  const pathParam = route.query.path as string;
  if (pathParam) {
    const success = await diskStore.restoreFromPath(decodeURIComponent(pathParam));
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
        />
        <FileList
          v-else
          ref="fileListRef"
          :files="fileList"
          :loading="loading"
          class="flex-1 min-h-0"
          @action="handleAction"
        />
        <!-- 上传传输面板(有任务时显示) -->
        <TransferPanel />
        <!-- 加载更多状态 -->
        <div v-if="loadingMore" class="flex-center gap-8px py-12px">
          <NSpin size="small" />
          <span class="text-13px opacity-60">{{ $t('page.disk.loadingMore') }}</span>
        </div>
      </NCard>

      <!-- 移动/复制 目录树选择弹窗 -->
      <MoveCopyModal
        v-model:visible="moveCopyModal.visible"
        :mode="moveCopyModal.mode"
        :source="moveCopyModal.source"
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
