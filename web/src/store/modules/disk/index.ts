import { computed, ref, watch } from 'vue';
import { useBreakpoints } from '@vueuse/core';
import { defineStore } from 'pinia';
import { SetupStoreId } from '@/enum';
import { router } from '@/router';
import { fetchResolvePath } from '@/service/api/disk/file';

/**
 * 网盘 store（第1期精简版）
 * 承载只读文件列表所需的全局状态：当前文件类型筛选 / 面包屑路径 / 视图模式 / 排序 / 选中 / 配额。
 * 上传/分享/移动复制/重命名/收藏等状态后续期补充。
 */
export const useDiskStore = defineStore(SetupStoreId.Disk, () => {
  // 当前选中的文件类型
  const currentFileType = ref<Api.Disk.FileType>('all');

  // 当前路径（面包屑）
  const currentPath = ref<Api.Disk.FileItem[]>([]);

  // 当前父文件夹ID
  const currentParentId = ref<CommonType.IdType | null>(null);

  // 视图模式
  const viewMode = ref<'grid' | 'list'>('grid');

  // 网格图标大小档位：large 大图（参照 remote 80px）/ small 小图（56px）
  const gridSize = ref<'small' | 'large'>('small');

  // 移动端强制列表模式
  const breakpoints = useBreakpoints({ sm: 640 });
  const isMobile = breakpoints.smaller('sm');
  watch(isMobile, mobile => {
    if (mobile) viewMode.value = 'list';
  }, { immediate: true });

  // 排序设置
  const sortSettings = ref<{
    field: 'name' | 'size' | 'modifyTime' | 'type' | null;
    order: 'asc' | 'desc';
  }>({
    field: null,
    order: 'asc'
  });

  // 当前目录文件列表（由 DiskPage 同步）
  const currentFileList = ref<Api.Disk.FileItem[]>([]);

  // 选中的文件
  const selectedFiles = ref<CommonType.IdType[]>([]);

  // 配额信息（跨页面共享，由 DiskPage loadQuotaInfo 写入）
  const quotaInfo = ref<Api.Disk.QuotaInfo>({
    usedSpace: 0,
    quota: 0,
    unlimited: false,
    quotaSource: 'none'
  });

  // 计算属性：面包屑路径显示
  const breadcrumbPath = computed(() => currentPath.value.map(item => item.fileName));

  // 切换文件类型
  function setFileType(type: Api.Disk.FileType) {
    currentFileType.value = type;
    currentParentId.value = null;
    currentPath.value = [];
    selectedFiles.value = [];
    syncStoreToUrl();
  }

  // 进入文件夹
  function enterFolder(folder: Api.Disk.FileItem) {
    currentParentId.value = folder.fileId;
    currentPath.value.push(folder);
    selectedFiles.value = [];
    syncStoreToUrl();
  }

  // 返回上一级（index 指定则截断到该层）
  function goBack(index?: number) {
    if (index !== undefined) {
      currentPath.value = currentPath.value.slice(0, index + 1);
    } else {
      currentPath.value.pop();
    }
    currentParentId.value = currentPath.value.length > 0
      ? currentPath.value[currentPath.value.length - 1].fileId
      : null;
    selectedFiles.value = [];
    syncStoreToUrl();
  }

  // 重置路径到根目录
  function resetPath() {
    currentPath.value = [];
    currentParentId.value = null;
    selectedFiles.value = [];
    syncStoreToUrl();
  }

  // 计算当前路径字符串
  function getCurrentPathString(): string {
    if (currentPath.value.length === 0) return '/';
    return '/' + currentPath.value.map(f => f.fileName).join('/');
  }

  // 同步Store状态到URL（使用query参数，避免路由组件重建）
  function syncStoreToUrl() {
    const pathStr = getCurrentPathString();
    if (pathStr !== '/') {
      router.push({ name: 'disk', query: { path: pathStr } });
    } else {
      router.push({ name: 'disk' });
    }
  }

  // 从URL恢复路径状态（接收已解码的路径）
  async function restoreFromPath(decodedPath: string): Promise<boolean> {
    if (!decodedPath || decodedPath === '/') {
      currentParentId.value = null;
      currentPath.value = [];
      return true;
    }

    try {
      const { data } = await fetchResolvePath(decodedPath);
      if (data) {
        currentParentId.value = data.fileId;
        // 从面包屑构建currentPath（排除根目录项：fileId为null或0）
        currentPath.value = data.breadcrumb
          .filter(b => b.fileId !== null && b.fileId !== 0)
          .map(b => ({
            fileId: b.fileId!,
            fileName: b.fileName,
            filePath: b.filePath,
            fileType: 'folder',
            fileSize: 0,
            isFolder: true,
            createTime: '',
            updateTime: '',
            modifyTime: '',
            createBy: '',
            updateBy: '',
            parentId: null
          }));
        return true;
      }
    } catch {
      // 路径解析失败，返回false
    }
    return false;
  }

  // 切换视图模式
  function setViewMode(mode: 'grid' | 'list') {
    viewMode.value = mode;
  }

  // 切换网格图标大小档位
  function setGridSize(size: 'small' | 'large') {
    gridSize.value = size;
  }

  // 设置排序
  function setSort(field: 'name' | 'size' | 'modifyTime' | 'type' | null, order: 'asc' | 'desc') {
    sortSettings.value = { field, order };
  }

  // 设置选中文件
  function setSelectedFiles(fileIds: CommonType.IdType[]) {
    selectedFiles.value = fileIds;
  }

  // 清空选中
  function clearSelection() {
    selectedFiles.value = [];
  }

  // 更新配额信息
  function updateQuotaInfo(info: Api.Disk.QuotaInfo) {
    quotaInfo.value = info;
  }

  // 重置所有状态
  function $reset() {
    currentFileType.value = 'all';
    currentPath.value = [];
    currentParentId.value = null;
    viewMode.value = 'grid';
    gridSize.value = 'small';
    sortSettings.value = { field: null, order: 'asc' };
    selectedFiles.value = [];
  }

  return {
    // state
    currentFileType,
    currentPath,
    currentParentId,
    viewMode,
    gridSize,
    sortSettings,
    currentFileList,
    selectedFiles,
    quotaInfo,
    // computed
    breadcrumbPath,
    // actions
    setFileType,
    enterFolder,
    goBack,
    resetPath,
    setViewMode,
    setGridSize,
    setSort,
    setSelectedFiles,
    clearSelection,
    updateQuotaInfo,
    getCurrentPathString,
    syncStoreToUrl,
    restoreFromPath,
    $reset
  };
});

export default useDiskStore;
